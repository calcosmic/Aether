package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

type colonyPrimeOutput struct {
	Context       string            `json:"context"`
	PromptSection string            `json:"prompt_section"`
	SignalCount   int               `json:"signal_count"`
	InstinctCount int               `json:"instinct_count"`
	ReviewCount   int               `json:"review_count"`
	LogLine       string            `json:"log_line"`
	Budget        int               `json:"budget"`
	Used          int               `json:"used"`
	Sections      int               `json:"sections"`
	Trimmed       []string          `json:"trimmed"`
	Warnings      []string          `json:"warnings,omitempty"`
	Ledger        colonyPrimeLedger `json:"ledger"`
}

type colonyPrimeLedger struct {
	Included  []colonyPrimeLedgerItem `json:"included"`
	Trimmed   []colonyPrimeLedgerItem `json:"trimmed"`
	Preserved []colonyPrimeLedgerItem `json:"preserved,omitempty"`
	Blocked   []colonyPrimeLedgerItem `json:"blocked,omitempty"`
}

type colonyPrimeLedgerItem struct {
	Name           string                          `json:"name"`
	Title          string                          `json:"title"`
	Source         string                          `json:"source"`
	Priority       int                             `json:"priority"`
	Chars          int                             `json:"chars"`
	BaseTrustClass colony.PromptTrustClass         `json:"base_trust_class,omitempty"`
	TrustClass     colony.PromptTrustClass         `json:"trust_class,omitempty"`
	Action         colony.PromptIntegrityAction    `json:"action,omitempty"`
	Blocked        bool                            `json:"blocked,omitempty"`
	Findings       []colony.PromptIntegrityFinding `json:"findings,omitempty"`
	Score          colony.ContextScoreBreakdown    `json:"score_breakdown,omitempty"`
	Preserved      bool                            `json:"preserved,omitempty"`
	PreserveReason string                          `json:"preserve_reason,omitempty"`
	TrimReason     string                          `json:"trim_reason,omitempty"`
	Decision       string                          `json:"decision,omitempty"`
}

type colonyPrimeSection struct {
	name              string
	title             string
	source            string
	content           string
	priority          int // legacy relevance hint retained for proof output
	baseTrustClass    colony.PromptTrustClass
	trustClass        colony.PromptTrustClass
	action            colony.PromptIntegrityAction
	findings          []colony.PromptIntegrityFinding
	freshnessScore    float64
	confirmationScore float64
	relevanceScore    float64
	protected         bool
	preserveReason    string
}

func (s colonyPrimeSection) ledgerItem() colonyPrimeLedgerItem {
	return colonyPrimeLedgerItem{
		Name:           s.name,
		Title:          s.title,
		Source:         filepath.ToSlash(s.source),
		Priority:       s.priority,
		Chars:          len(s.content),
		BaseTrustClass: s.baseTrustClass,
		TrustClass:     s.trustClass,
		Action:         s.action,
		Blocked:        s.action == colony.PromptIntegrityActionBlock,
		Findings:       append([]colony.PromptIntegrityFinding(nil), s.findings...),
	}
}

func (s colonyPrimeSection) rankingCandidate() colony.ContextCandidate {
	return colony.ContextCandidate{
		Name:              s.name,
		Title:             s.title,
		Source:            s.source,
		Content:           s.content,
		BudgetMetric:      "chars",
		PriorityHint:      s.priority,
		BaseTrustClass:    s.baseTrustClass,
		TrustClass:        s.trustClass,
		Action:            s.action,
		FreshnessScore:    s.freshnessScore,
		ConfirmationScore: s.confirmationScore,
		RelevanceScore:    s.relevanceScore,
		Protected:         s.protected,
		PreserveReason:    s.preserveReason,
	}
}

func colonyPrimeLedgerItemFromRanked(item colony.RankedContextCandidate) colonyPrimeLedgerItem {
	return colonyPrimeLedgerItem{
		Name:           item.Name,
		Title:          item.Title,
		Source:         filepath.ToSlash(strings.TrimSpace(item.Source)),
		Priority:       item.PriorityHint,
		Chars:          len(item.Content),
		BaseTrustClass: item.BaseTrustClass,
		TrustClass:     item.TrustClass,
		Action:         item.Action,
		Blocked:        item.Action == colony.PromptIntegrityActionBlock,
		Score:          item.Score,
		Preserved:      item.Preserved,
		PreserveReason: item.PreserveReason,
		TrimReason:     item.TrimReason,
		Decision:       item.Decision,
	}
}

// priorReviewsCache stores the assembled prior-reviews text and per-domain counts.
type priorReviewsCache struct {
	Text         string         `json:"text"`
	DomainCounts map[string]int `json:"domain_counts"`
	TotalOpen    int            `json:"total_open"`
	CacheWriteAt string         `json:"cache_write_at"`
}

func severityRank(s colony.ReviewSeverity) int {
	switch s {
	case colony.ReviewSeverityHigh:
		return 4
	case colony.ReviewSeverityMedium:
		return 3
	case colony.ReviewSeverityLow:
		return 2
	case colony.ReviewSeverityInfo:
		return 1
	default:
		return 0
	}
}

func domainPosition(domain string) int {
	for i, d := range colony.DomainOrder {
		if d == domain {
			return i
		}
	}
	return len(colony.DomainOrder)
}

func buildPriorReviewsSection(s *storage.Store, compact bool) (colonyPrimeSection, int) {
	budget := 800
	if compact {
		budget = 400
	}
	maxFindingsPerDomain := 2
	maxDescLen := 60

	// 1. Check cache (D-04, D-05, D-06)
	cachePath := "reviews/_summary_cache.json"
	var cache priorReviewsCache
	cacheFresh := false

	cacheFullPath := filepath.Join(s.BasePath(), cachePath)
	cacheStat, cacheStatErr := os.Stat(cacheFullPath)
	if cacheStatErr == nil {
		if err := s.LoadJSON(cachePath, &cache); err == nil && cache.Text != "" {
			// Check if any ledger file is newer than cache (D-06)
			cacheFresh = true
			for _, d := range colony.DomainOrder {
				ledgerStat, err := os.Stat(filepath.Join(s.BasePath(), "reviews", d, "ledger.json"))
				if err == nil && ledgerStat.ModTime().After(cacheStat.ModTime()) {
					cacheFresh = false
					break
				}
			}
		}
	}

	if cacheFresh && cache.Text != "" {
		return colonyPrimeSection{
			name:              "prior_reviews",
			title:             "Prior Reviews",
			source:            cacheFullPath,
			content:           cache.Text,
			priority:          8,
			freshnessScore:    1.0,
			confirmationScore: 1.0, // D-12
			relevanceScore:    sectionRelevanceScore("prior_reviews"),
		}, cache.TotalOpen
	}

	// 2. Read all 7 ledgers, collect open findings per domain (D-02)
	type domainData struct {
		domain string
		open   []colony.ReviewLedgerEntry
		maxSev colony.ReviewSeverity
	}
	var domains []domainData
	var latestTimestamp string

	for _, d := range colony.DomainOrder {
		var lf colony.ReviewLedgerFile
		if err := s.LoadJSON(fmt.Sprintf("reviews/%s/ledger.json", d), &lf); err != nil {
			continue
		}
		var openEntries []colony.ReviewLedgerEntry
		var maxSev colony.ReviewSeverity
		for _, e := range lf.Entries {
			if e.Status == "open" {
				openEntries = append(openEntries, e)
				if severityRank(e.Severity) > severityRank(maxSev) {
					maxSev = e.Severity
				}
				if e.GeneratedAt > latestTimestamp {
					latestTimestamp = e.GeneratedAt
				}
			}
		}
		if len(openEntries) > 0 {
			domains = append(domains, domainData{domain: d, open: openEntries, maxSev: maxSev})
		}
	}

	// D-11: Omit entirely when no open findings
	if len(domains) == 0 {
		return colonyPrimeSection{}, 0
	}

	// 3. Sort domains by max-severity descending, tiebreak by domainOrder position (D-07, D-09)
	sort.SliceStable(domains, func(i, j int) bool {
		ri, rj := severityRank(domains[i].maxSev), severityRank(domains[j].maxSev)
		if ri != rj {
			return ri > rj
		}
		return domainPosition(domains[i].domain) < domainPosition(domains[j].domain)
	})

	// 4. Format section content with budget management (D-01, D-03, D-08)
	var sb strings.Builder
	sb.WriteString("## Prior Reviews\n\n")

	domainCounts := make(map[string]int)
	totalOpen := 0

	for _, dd := range domains {
		totalOpen += len(dd.open)
		domainCounts[dd.domain] = len(dd.open)

		lineParts := make([]string, 0, len(dd.open))
		shown := 0
		for _, e := range dd.open {
			if shown >= maxFindingsPerDomain {
				break // D-03
			}
			loc := ""
			if e.File != "" {
				loc = e.File
				if e.Line > 0 {
					loc = fmt.Sprintf("%s:%d", e.File, e.Line)
				}
			}
			desc := e.Description
			if len(desc) > maxDescLen {
				desc = desc[:maxDescLen-3] + "..."
			}
			if loc != "" {
				lineParts = append(lineParts, fmt.Sprintf("%s -- %s %s", string(e.Severity), loc, desc))
			} else {
				lineParts = append(lineParts, fmt.Sprintf("%s -- %s", string(e.Severity), desc))
			}
			shown++
		}

		domainLabel := fmt.Sprintf("- %s (%d open)", strings.Title(dd.domain), len(dd.open))

		fullLine := domainLabel + ": " + strings.Join(lineParts, ", ")
		remaining := len(dd.open) - shown
		if remaining > 0 {
			fullLine += fmt.Sprintf(" +%d more", remaining)
		}

		if sb.Len()+len(fullLine)+1 <= budget {
			sb.WriteString(fullLine)
			sb.WriteString("\n")
		} else if sb.Len()+len(domainLabel)+1 <= budget {
			// D-08: Truncate to counts-only
			sb.WriteString(domainLabel)
			sb.WriteString("\n")
		} else {
			// D-08: Drop entirely
			break
		}
	}

	content := sb.String()

	// 5. Write cache (D-04)
	cache = priorReviewsCache{
		Text:         content,
		DomainCounts: domainCounts,
		TotalOpen:    totalOpen,
		CacheWriteAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = s.SaveJSON(cachePath, cache)

	// 6. Compute scores (D-12)
	now := time.Now().UTC()
	freshnessScore := 1.0
	if latestTimestamp != "" {
		freshnessScore = freshnessScoreFromTimestamp(latestTimestamp, now, 0.85)
	}

	return colonyPrimeSection{
		name:              "prior_reviews",
		title:             "Prior Reviews",
		source:            filepath.Join(s.BasePath(), cachePath),
		content:           content,
		priority:          8,
		freshnessScore:    freshnessScore,
		confirmationScore: 1.0, // D-12: findings are factual
		relevanceScore:    sectionRelevanceScore("prior_reviews"),
	}, totalOpen
}

func buildColonyPrimeOutput(compact bool) colonyPrimeOutput {
	budget := 8000
	if compact {
		budget = 4000
	}
	result := colonyPrimeOutput{
		Budget:   budget,
		Trimmed:  []string{},
		Warnings: []string{},
		Ledger: colonyPrimeLedger{
			Included:  []colonyPrimeLedgerItem{},
			Trimmed:   []colonyPrimeLedgerItem{},
			Preserved: []colonyPrimeLedgerItem{},
			Blocked:   []colonyPrimeLedgerItem{},
		},
	}
	if store == nil {
		return result
	}

	sc := cache.NewSessionCache(store.BasePath())
	sc.ClearStale(24 * time.Hour)

	sections := make([]colonyPrimeSection, 0, 9)

	var state colony.ColonyState
	statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
	if err := sc.Load(statePath, &state); err != nil {
		_ = store.LoadJSON("COLONY_STATE.json", &state)
	}

	var stateSection strings.Builder
	stateSection.WriteString("## Colony State\n\n")
	if state.Goal != nil {
		stateSection.WriteString(fmt.Sprintf("Goal: %s\n", *state.Goal))
	}
	stateSection.WriteString(fmt.Sprintf("State: %s\n", state.State))
	stateSection.WriteString(fmt.Sprintf("Phase: %d\n", state.CurrentPhase))
	if len(state.Plan.Phases) > 0 && state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		phase := state.Plan.Phases[state.CurrentPhase-1]
		stateSection.WriteString(fmt.Sprintf("Phase Name: %s\n", phase.Name))
		if len(phase.Tasks) > 0 {
			stateSection.WriteString("Tasks:\n")
			for _, t := range phase.Tasks {
				stateSection.WriteString(fmt.Sprintf("  - [%s] %s\n", t.Status, t.Goal))
			}
		}
	}
	mode := state.ParallelMode
	if mode == "" {
		mode = colony.ModeInRepo
	}
	stateSection.WriteString(fmt.Sprintf("Parallel Mode: %s\n", mode))
	stateProtected, statePreserveReason := protectedSectionPolicy("state")
	sections = append(sections, colonyPrimeSection{
		name:              "state",
		title:             "Colony State",
		source:            statePath,
		content:           stateSection.String(),
		priority:          5,
		freshnessScore:    1.0,
		confirmationScore: 1.0,
		relevanceScore:    sectionRelevanceScore("state"),
		protected:         stateProtected,
		preserveReason:    statePreserveReason,
	})

	// Review depth section (D-13, D-14)
	if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		reviewPhase := state.Plan.Phases[state.CurrentPhase-1]
		reviewDepth := resolveReviewDepth(reviewPhase, len(state.Plan.Phases), false, false)
		var depthText string
		if reviewDepth == ReviewDepthLight {
			depthText = "Light review -- core verification only"
		} else {
			depthText = "Heavy review -- full quality gauntlet"
		}
		sections = append(sections, colonyPrimeSection{
			name:           "review_depth",
			title:          "Review Depth",
			source:         statePath,
			content:        fmt.Sprintf("## Review Depth\n\n%s\n", depthText),
			priority:       6,
			freshnessScore: 1.0,
		})
	}

	now := time.Now().UTC()
	pf, phErr := loadPheromonesOnce(store, sc)
	if phErr == nil && len(pf.Signals) > 0 {
		activeSignals := filterSignalsForPrompt(pf.Signals, now)
		result.SignalCount = len(activeSignals)
		if len(activeSignals) > 0 {
			var phSB strings.Builder
			phSB.WriteString("## Pheromone Signals\n\n")
			phSB.WriteString(colonyLifecycleSignalContext(state))
			phSB.WriteString("\n\n")
			for _, sig := range activeSignals {
				text := extractText(sig.Content)
				if text == "" {
					continue
				}
				phSB.WriteString(fmt.Sprintf("- [%s] %s\n", sig.Type, text))
			}
			if strings.TrimSpace(phSB.String()) != "" {
				signalTimestamps := make([]string, 0, len(activeSignals))
				for _, sig := range activeSignals {
					signalTimestamps = append(signalTimestamps, sig.CreatedAt)
				}
				signalsProtected, signalsPreserveReason := protectedSectionPolicy("pheromones")
				sections = append(sections, colonyPrimeSection{
					name:              "pheromones",
					title:             "Pheromone Signals",
					source:            filepath.Join(store.BasePath(), "pheromones.json"),
					content:           phSB.String(),
					priority:          9,
					freshnessScore:    latestFreshnessScore(now, 0.85, signalTimestamps...),
					confirmationScore: confidenceScoreFromSignals(activeSignals, now),
					relevanceScore:    sectionRelevanceScore("pheromones"),
					protected:         signalsProtected,
					preserveReason:    signalsPreserveReason,
				})
			}
		}
	}

	instinctEntries := make([]colony.InstinctEntry, 0)
	var instincts []struct {
		trigger    string
		action     string
		confidence float64
	}
	var instFile colony.InstinctsFile
	instinctsPath := filepath.Join(store.BasePath(), "instincts.json")
	instinctsLoaded := false
	if err := sc.Load(instinctsPath, &instFile); err == nil {
		instinctsLoaded = true
	} else if err := store.LoadJSON("instincts.json", &instFile); err == nil {
		instinctsLoaded = true
	}
	if instinctsLoaded {
		for _, inst := range instFile.Instincts {
			if inst.Archived {
				continue
			}
			instinctEntries = append(instinctEntries, inst)
			instincts = append(instincts, struct {
				trigger    string
				action     string
				confidence float64
			}{trigger: inst.Trigger, action: inst.Action, confidence: inst.Confidence})
		}
	} else if state.Memory.Instincts != nil {
		for _, inst := range state.Memory.Instincts {
			instincts = append(instincts, struct {
				trigger    string
				action     string
				confidence float64
			}{trigger: inst.Trigger, action: inst.Action, confidence: inst.Confidence})
		}
	}
	if len(instincts) > 0 {
		var instSB strings.Builder
		instSB.WriteString("## Active Instincts\n\n")
		for _, inst := range instincts {
			instSB.WriteString(fmt.Sprintf("- [%s] %s (confidence: %.2f)\n", inst.trigger, inst.action, inst.confidence))
		}
		source := instinctsPath
		if !instinctsLoaded {
			source = statePath
		}
		sections = append(sections, colonyPrimeSection{
			name:              "instincts",
			title:             "Active Instincts",
			source:            source,
			content:           instSB.String(),
			priority:          6,
			freshnessScore:    latestInstinctFreshness(now, instinctEntries, instincts),
			confirmationScore: instinctConfidenceScore(instinctEntries, instincts),
			relevanceScore:    sectionRelevanceScore("instincts"),
		})
	}
	result.InstinctCount = len(instincts)

	if state.Memory.Decisions != nil && len(state.Memory.Decisions) > 0 {
		var decSB strings.Builder
		decSB.WriteString("## Key Decisions\n\n")
		for _, d := range state.Memory.Decisions {
			decSB.WriteString(fmt.Sprintf("- Phase %d: %s — %s\n", d.Phase, d.Claim, d.Rationale))
		}
		sections = append(sections, colonyPrimeSection{
			name:              "decisions",
			title:             "Key Decisions",
			source:            statePath,
			content:           decSB.String(),
			priority:          3,
			freshnessScore:    latestDecisionFreshness(now, state.Memory.Decisions),
			confirmationScore: confidenceScoreFromDecisions(state.Memory.Decisions, state.CurrentPhase),
			relevanceScore:    phaseScopedRelevance(sectionRelevanceScore("decisions"), state.CurrentPhase, decisionPhases(state.Memory.Decisions)...),
		})
	}

	if state.Memory.PhaseLearnings != nil && len(state.Memory.PhaseLearnings) > 0 {
		var learnSB strings.Builder
		learnSB.WriteString("## Phase Learnings\n\n")
		for _, pl := range state.Memory.PhaseLearnings {
			learnSB.WriteString(fmt.Sprintf("### Phase %d: %s\n", pl.Phase, pl.PhaseName))
			for _, l := range pl.Learnings {
				learnSB.WriteString(fmt.Sprintf("  - %s [%s]\n", l.Claim, l.Status))
			}
		}
		sections = append(sections, colonyPrimeSection{
			name:              "learnings",
			title:             "Phase Learnings",
			source:            statePath,
			content:           learnSB.String(),
			priority:          2,
			freshnessScore:    latestPhaseLearningFreshness(now, state.Memory.PhaseLearnings),
			confirmationScore: phaseLearningConfidenceScore(state.Memory.PhaseLearnings),
			relevanceScore:    phaseScopedRelevance(sectionRelevanceScore("learnings"), state.CurrentPhase, phaseLearningPhases(state.Memory.PhaseLearnings)...),
		})
	}

	hubDir := resolveHubPath()
	var fallbacks []string
	hiveEntries := readHiveWisdomEntries(hubDir, 5, &fallbacks)
	hiveLines := buildHiveWisdomLines(hiveEntries)
	if len(hiveLines) > 0 {
		var hiveSB strings.Builder
		hiveSB.WriteString("## HIVE WISDOM (Cross-Colony Patterns)\n\n")
		for _, entry := range hiveLines {
			hiveSB.WriteString(fmt.Sprintf("- %s\n", entry))
		}
		sections = append(sections, colonyPrimeSection{
			name:              "hive_wisdom",
			title:             "Hive Wisdom",
			source:            filepath.Join(hubDir, "hive", "wisdom.json"),
			content:           hiveSB.String(),
			priority:          4,
			freshnessScore:    hiveFreshnessScore(now, hiveEntries),
			confirmationScore: confidenceScoreFromHive(hiveEntries),
			relevanceScore:    sectionRelevanceScore("hive_wisdom"),
		})
	}

	// Global QUEEN.md wisdom (cross-colony, hub-level)
	globalQueenPath := filepath.Join(hubDir, "QUEEN.md")
	globalWisdom := readQUEENMd(globalQueenPath)
	if len(globalWisdom) > 0 {
		var gwSB strings.Builder
		gwSB.WriteString("## GLOBAL QUEEN WISDOM (Cross-Colony)\n\n")
		for _, v := range globalWisdom {
			gwSB.WriteString(fmt.Sprintf("- %s\n", v))
		}
		gqProtected, gqPreserveReason := protectedSectionPolicy("global_queen_md")
		sections = append(sections, colonyPrimeSection{
			name:              "global_queen_md",
			title:             "Global Queen Wisdom",
			source:            globalQueenPath,
			content:           gwSB.String(),
			priority:          5,
			freshnessScore:    0.85,
			confirmationScore: 0.90,
			relevanceScore:    sectionRelevanceScore("global_queen_md"),
			protected:         gqProtected,
			preserveReason:    gqPreserveReason,
		})
	}

	queenPath := filepath.Join(hubDir, "QUEEN.md")
	userPrefs := readUserPreferences(queenPath)
	// Also read local repo QUEEN.md user preferences
	localQueenPath := filepath.Join(filepath.Dir(store.BasePath()), "QUEEN.md")
	userPrefs = append(userPrefs, readUserPreferences(localQueenPath)...)
	if len(userPrefs) > 0 {
		var prefsSB strings.Builder
		prefsSB.WriteString("## USER PREFERENCES\n\n")
		for _, pref := range userPrefs {
			prefsSB.WriteString(fmt.Sprintf("- %s\n", pref))
		}
		prefsProtected, prefsPreserveReason := protectedSectionPolicy("user_preferences")
		sections = append(sections, colonyPrimeSection{
			name:              "user_preferences",
			title:             "User Preferences",
			source:            queenPath,
			content:           prefsSB.String(),
			priority:          7,
			freshnessScore:    0.85,
			confirmationScore: 0.90,
			relevanceScore:    sectionRelevanceScore("user_preferences"),
			protected:         prefsProtected,
			preserveReason:    prefsPreserveReason,
		})
	}

	// Prior Reviews -- open review findings from domain ledgers
	priorReviewsSection, reviewCount := buildPriorReviewsSection(store, compact)
	if reviewCount > 0 {
		result.ReviewCount = reviewCount
		sections = append(sections, priorReviewsSection)
	}

	// Local QUEEN.md wisdom (repo-specific)
	localWisdom := readQUEENMd(localQueenPath)
	if len(localWisdom) > 0 {
		var lwSB strings.Builder
		lwSB.WriteString("## LOCAL QUEEN WISDOM (Repo-Specific)\n\n")
		for _, v := range localWisdom {
			lwSB.WriteString(fmt.Sprintf("- %s\n", v))
		}
		sections = append(sections, colonyPrimeSection{
			name:              "local_queen_wisdom",
			title:             "Local Queen Wisdom",
			source:            localQueenPath,
			content:           lwSB.String(),
			priority:          5,
			freshnessScore:    0.80,
			confirmationScore: 0.85,
			relevanceScore:    sectionRelevanceScore("local_queen_wisdom"),
		})
	}

	if clarifications := clarifiedIntentPromptEntries(); len(clarifications) > 0 {
		var clarifySB strings.Builder
		clarifySB.WriteString("## CLARIFIED INTENT\n\n")
		for _, clarification := range clarifications {
			clarifySB.WriteString(clarification)
			clarifySB.WriteString("\n")
		}
		intentProtected, intentPreserveReason := protectedSectionPolicy("clarified_intent")
		sections = append(sections, colonyPrimeSection{
			name:              "clarified_intent",
			title:             "Clarified Intent",
			source:            filepath.Join(store.BasePath(), pendingDecisionsFile),
			content:           clarifySB.String(),
			priority:          8,
			freshnessScore:    1.0,
			confirmationScore: 1.0,
			relevanceScore:    sectionRelevanceScore("clarified_intent"),
			protected:         intentProtected,
			preserveReason:    intentPreserveReason,
		})
	}

	var blockerFile colony.FlagsFile
	blockerSource := filepath.Join(store.BasePath(), pendingDecisionsFile)
	if err := store.LoadJSON("pending-decisions.json", &blockerFile); err != nil {
		blockerSource = filepath.Join(store.BasePath(), "flags.json")
		_ = store.LoadJSON("flags.json", &blockerFile)
	}
	if len(blockerFile.Decisions) > 0 {
		var blockerSB strings.Builder
		blockerTimestamps := make([]string, 0, len(blockerFile.Decisions))
		for _, blocker := range blockerFile.Decisions {
			if blocker.Resolved || blocker.Type != "blocker" {
				continue
			}
			if blockerSB.Len() == 0 {
				blockerSB.WriteString("## Active Blockers\n\n")
			}
			blockerSB.WriteString(fmt.Sprintf("- %s\n", blocker.Description))
			blockerTimestamps = append(blockerTimestamps, blocker.CreatedAt)
		}
		if blockerSB.Len() > 0 {
			blockerProtected, blockerPreserveReason := protectedSectionPolicy("blockers")
			sections = append(sections, colonyPrimeSection{
				name:              "blockers",
				title:             "Active Blockers",
				source:            blockerSource,
				content:           blockerSB.String(),
				priority:          10,
				freshnessScore:    latestFreshnessScore(now, 0.9, blockerTimestamps...),
				confirmationScore: 1.0,
				relevanceScore:    sectionRelevanceScore("blockers"),
				protected:         blockerProtected,
				preserveReason:    blockerPreserveReason,
			})
		}
	}

	// Medic health section — inject critical issues from last scan
	if lastScan, err := loadMedicLastScan(store.BasePath()); err == nil {
		var criticalIssues []HealthIssue
		for _, issue := range lastScan.Issues {
			if issue.Severity == "critical" {
				criticalIssues = append(criticalIssues, issue)
			}
		}
		if len(criticalIssues) > 0 {
			var healthSB strings.Builder
			healthSB.WriteString("## Colony Health Issues\n\n")
			healthSB.WriteString(fmt.Sprintf("Last scan: %s\n\n", lastScan.Timestamp))
			for _, issue := range criticalIssues {
				healthSB.WriteString(fmt.Sprintf("- [%s] %s", issue.Severity, issue.Message))
				if issue.File != "" {
					healthSB.WriteString(fmt.Sprintf(" (%s)", issue.File))
				}
				healthSB.WriteString("\n")
			}
			healthProtected, healthPreserveReason := protectedSectionPolicy("medic_health")
			sections = append(sections, colonyPrimeSection{
				name:              "medic_health",
				title:             "Colony Health Issues",
				source:            filepath.Join(store.BasePath(), medicLastScanFile),
				content:           healthSB.String(),
				priority:          9,
				freshnessScore:    latestFreshnessScore(now, 0.8, lastScan.Timestamp),
				confirmationScore: 1.0,
				relevanceScore:    sectionRelevanceScore("medic_health"),
				protected:         healthProtected,
				preserveReason:    healthPreserveReason,
			})
		}
	}

	result.Sections = len(sections)
	allowedCandidates := make([]colony.ContextCandidate, 0, len(sections))
	for _, sec := range sections {
		assessment := colony.AssessPromptSource(sec.source, sec.content)
		sec.baseTrustClass = assessment.BaseTrustClass
		sec.trustClass = assessment.TrustClass
		sec.action = assessment.Action
		sec.findings = append([]colony.PromptIntegrityFinding(nil), assessment.Findings...)
		if sec.action == colony.PromptIntegrityActionBlock {
			result.Warnings = append(result.Warnings, assessment.Warning(sec.name, sec.source))
			result.Ledger.Blocked = append(result.Ledger.Blocked, sec.ledgerItem())
			continue
		}
		allowedCandidates = append(allowedCandidates, sec.rankingCandidate())
	}

	ranking := colony.RankContextCandidates(allowedCandidates, budget)
	var assembled strings.Builder
	for _, item := range ranking.Included {
		if assembled.Len() > 0 {
			assembled.WriteString("\n")
		}
		assembled.WriteString(strings.TrimRight(item.Content, "\n"))
		ledgerItem := colonyPrimeLedgerItemFromRanked(item)
		result.Ledger.Included = append(result.Ledger.Included, ledgerItem)
		if item.Preserved {
			result.Ledger.Preserved = append(result.Ledger.Preserved, ledgerItem)
		}
	}
	for _, item := range ranking.Trimmed {
		result.Trimmed = append(result.Trimmed, item.Name)
		result.Ledger.Trimmed = append(result.Ledger.Trimmed, colonyPrimeLedgerItemFromRanked(item))
	}

	context := strings.TrimSpace(assembled.String())
	result.Context = context
	result.PromptSection = context
	result.Used = ranking.Used
	result.LogLine = fmt.Sprintf("colony-prime loaded %d signal(s), %d instinct(s), %d review(s), used %d/%d chars", result.SignalCount, result.InstinctCount, result.ReviewCount, ranking.Used, budget)
	return result
}

func resolveCodexWorkerContext() string {
	context := strings.TrimSpace(buildColonyPrimeOutput(true).PromptSection)
	if context == "" {
		context = buildContextCapsuleOutput(true, 8, 3, 2, 220).PromptSection
	}
	if len(context) < 128 {
		fmt.Fprintf(os.Stderr, "⚠ Context capsule below minimum threshold (%d chars, min 128) — dispatch blocked to prevent zero-context worker execution\n", len(context))
		return ""
	}
	return context
}
