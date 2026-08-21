package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/storage"
)

// Budget constants for buildColonyPrimeOutput (D-03). Named so every later
// measurement (e.g. Plan 06's budget-ceiling invariant test) refers to the
// same number instead of a literal that can silently drift out of sync.
const (
	colonyPrimeBudgetChars        = 8000
	colonyPrimeCompactBudgetChars = 4000
)

// charterNoGovernanceFallback is generateCharter's placeholder string
// (cmd/init_research.go) for a colony where no governance tooling was
// detected. It carries no binding rule, so it must not surface as one.
const charterNoGovernanceFallback = "No formal governance detected -- colony should establish conventions"

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
	writeSectionHeader(&sb, "prior_reviews", "## Prior Reviews\n\n")

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

// colonyPrimeOptions parameterizes the briefing assembler. Question is the
// ask-mode addition: when set, sections that overlap the question's words
// get a relevance boost (so "why is phase 3 blocked?" ranks blockers above
// boilerplate) and an activity-tail section joins the roster. colony-prime
// had taken zero inputs since it was written; this is its first
// parameterization, kept behind an options struct so the next one does not
// change every call site again.
type colonyPrimeOptions struct {
	Compact  bool
	Question string
}

func buildColonyPrimeOutput(compact bool) colonyPrimeOutput {
	return buildColonyPrimeOutputOpts(colonyPrimeOptions{Compact: compact})
}

func buildColonyPrimeOutputOpts(opts colonyPrimeOptions) colonyPrimeOutput {
	compact := opts.Compact
	budget := colonyPrimeBudgetChars
	if compact {
		budget = colonyPrimeCompactBudgetChars
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

	// Ask mode (--question) is a pure inspection and must leave
	// .aether/data byte-identical — the session cache both writes
	// acceleration entries and prunes stale ones, so ask mode bypasses it
	// entirely. Locked by TestColonyPrimeQuestionIsReadOnly.
	askMode := strings.TrimSpace(opts.Question) != ""
	var sc *cache.SessionCache
	if !askMode {
		sc = cache.NewSessionCache(store.BasePath())
		sc.ClearStale(24 * time.Hour)
	}
	cachedLoad := func(path, rel string, dest interface{}) error {
		if sc != nil {
			if err := sc.Load(path, dest); err == nil {
				return nil
			}
		}
		return store.LoadJSON(rel, dest)
	}

	sections := make([]colonyPrimeSection, 0, 9)

	var state colony.ColonyState
	statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
	_ = cachedLoad(statePath, "COLONY_STATE.json", &state)

	var stateSection strings.Builder
	writeSectionHeader(&stateSection, "state", "## Colony State\n\n")
	if state.Goal != nil {
		stateSection.WriteString(fmtOrFallback("state", func(t *sectionTemplate) string { return t.GoalFormat }, "Goal: %s\n", *state.Goal))
	}
	stateSection.WriteString(fmtOrFallback("state", func(t *sectionTemplate) string { return t.StateFormat }, "State: %s\n", state.State))
	stateSection.WriteString(fmtOrFallback("state", func(t *sectionTemplate) string { return t.PhaseFormat }, "Phase: %d\n", state.CurrentPhase))
	if len(state.Plan.Phases) > 0 && state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		phase := state.Plan.Phases[state.CurrentPhase-1]
		stateSection.WriteString(fmtOrFallback("state", func(t *sectionTemplate) string { return t.PhaseNameFormat }, "Phase Name: %s\n", phase.Name))
		if len(phase.Tasks) > 0 {
			stateSection.WriteString(sectionString("state", func(t *sectionTemplate) string { return t.TasksHeader }, "Tasks:\n"))
			for _, t := range phase.Tasks {
				stateSection.WriteString(fmtOrFallback("state", func(tmpl *sectionTemplate) string { return tmpl.TaskFormat }, "  - [%s] %s\n", t.Status, t.Goal))
			}
		}
	}
	mode := state.ParallelMode
	if mode == "" {
		mode = colony.ModeInRepo
	}
	stateSection.WriteString(fmtOrFallback("state", func(t *sectionTemplate) string { return t.ParallelModeFormat }, "Parallel Mode: %s\n", mode))
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
		storedDepthStr := strings.TrimSpace(state.VerificationDepth)
		reviewDepth := resolveVerificationDepth(reviewPhase, len(state.Plan.Phases), false, false, storedDepthStr)
		var depthText string
		switch reviewDepth {
		case colony.VerificationDepthLight:
			depthText = sectionString("review_depth", func(t *sectionTemplate) string { return t.LightText }, "Light review -- core verification only")
		case colony.VerificationDepthStandard:
			depthText = sectionString("review_depth", func(t *sectionTemplate) string { return t.StandardText }, "Standard review -- watcher and probe verification")
		case colony.VerificationDepthHeavy:
			depthText = sectionString("review_depth", func(t *sectionTemplate) string { return t.HeavyText }, "Heavy review -- full quality gauntlet")
		default:
			depthText = sectionString("review_depth", func(t *sectionTemplate) string { return t.DefaultText }, "Standard review -- watcher and probe verification")
		}
		var rdSB strings.Builder
		writeSectionHeader(&rdSB, "review_depth", "## Review Depth\n\n")
		rdSB.WriteString(depthText)
		rdSB.WriteString("\n")
		sections = append(sections, colonyPrimeSection{
			name:           "review_depth",
			title:          "Review Depth",
			source:         statePath,
			content:        rdSB.String(),
			priority:       6,
			freshnessScore: 1.0,
		})
	}

	// Charter section (CONTEXT-06, D-09): the colony charter is
	// user-approved governance and must reach every worker as a binding
	// rule, not background information -- silence is what made the charter
	// decorative in the first place. Skip entirely when there is nothing to
	// say (no charter, or generateCharter's "no formal governance" fallback):
	// an empty heading is noise that costs budget. Do NOT string-concatenate
	// this into any brief renderer -- routing it through colonyPrimeSection
	// is the security control (T-163-03); it inherits AssessPromptSource and
	// RankContextCandidates below for free, same as every other section.
	if state.Charter != nil {
		governanceText := strings.TrimSpace(state.Charter.Governance)
		if governanceText == charterNoGovernanceFallback {
			governanceText = ""
		}
		constraintsText := strings.TrimSpace(state.Charter.Constraints)

		// Intent, Vision, Goals, TechStack and KeyRisks are synthesized from the
		// operator's own words at init and shown for approval, then were never
		// read by anything: only Governance and Constraints reached a worker.
		// A colony could capture exactly what the operator wanted, store it,
		// and tell the workers none of it — which on this repo meant five
		// populated fields silently withheld while the one empty field was the
		// only thing forwarded.
		//
		// They are emitted as context, deliberately below the hard-rules block
		// and under their own framing. Governance and Constraints are approved
		// rules; intent and risks are orientation. Presenting them as equally
		// binding would make the "hard rules" sentence untrue and invite a
		// worker to treat a vision statement as a constraint.
		type charterContextField struct {
			label string
			value string
		}
		contextFields := []charterContextField{
			{"Intent", strings.TrimSpace(state.Charter.Intent)},
			{"Vision", strings.TrimSpace(state.Charter.Vision)},
			{"Goals", strings.TrimSpace(state.Charter.Goals)},
			{"Tech stack", strings.TrimSpace(state.Charter.TechStack)},
			{"Key risks", strings.TrimSpace(state.Charter.KeyRisks)},
		}
		hasContextField := false
		for _, field := range contextFields {
			if field.value != "" {
				hasContextField = true
				break
			}
		}

		if governanceText != "" || constraintsText != "" || hasContextField {
			var charterSB strings.Builder
			writeSectionHeader(&charterSB, "charter", charterFallbackHeading+"\n\n")
			if governanceText != "" || constraintsText != "" {
				charterSB.WriteString("The colony operator approved the following governance. These are hard rules every worker must follow, not background information:\n\n")
				if governanceText != "" {
					charterSB.WriteString(fmt.Sprintf("Governance: %s\n", governanceText))
				}
				if constraintsText != "" {
					charterSB.WriteString(fmt.Sprintf("Constraints: %s\n", constraintsText))
				}
			}
			if hasContextField {
				if governanceText != "" || constraintsText != "" {
					charterSB.WriteString("\n")
				}
				charterSB.WriteString("The operator described the work this way. Treat it as orientation for judgement calls, not as additional hard rules:\n\n")
				for _, field := range contextFields {
					if field.value == "" {
						continue
					}
					charterSB.WriteString(fmt.Sprintf("%s: %s\n", field.label, field.value))
				}
			}
			charterProtected, charterPreserveReason := protectedSectionPolicy("charter")
			sections = append(sections, colonyPrimeSection{
				name:              "charter",
				title:             "Charter",
				source:            statePath,
				content:           charterSB.String(),
				priority:          9,
				freshnessScore:    1.0,
				confirmationScore: 1.0,
				relevanceScore:    sectionRelevanceScore("charter"),
				protected:         charterProtected,
				preserveReason:    charterPreserveReason,
			})
		}
	}

	now := time.Now().UTC()
	pf, phErr := loadPheromonesOnce(store, sc)
	if phErr == nil && len(pf.Signals) > 0 {
		activeSignals := filterSignalsForPrompt(pf.Signals, now)
		result.SignalCount = len(activeSignals)
		if len(activeSignals) > 0 {
			var phSB strings.Builder
			writeSectionHeader(&phSB, "pheromones", "## Pheromone Signals\n\n")
			phSB.WriteString(colonyLifecycleSignalContext(state))
			phSB.WriteString("\n\n")
			for _, sig := range activeSignals {
				text := extractText(sig.Content)
				if text == "" {
					continue
				}
				phSB.WriteString(fmtOrFallback("pheromones", func(t *sectionTemplate) string { return t.SignalFormat }, "- [%s] %s\n", sig.Type, text))
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
	instinctsLoaded := cachedLoad(instinctsPath, "instincts.json", &instFile) == nil
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
		writeSectionHeader(&instSB, "instincts", "## Active Instincts\n\n")
		for _, inst := range instincts {
			instSB.WriteString(fmtOrFallback("instincts", func(t *sectionTemplate) string { return t.InstinctFormat }, "- [%s] %s (confidence: %.2f)\n", inst.trigger, inst.action, inst.confidence))
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
		writeSectionHeader(&decSB, "decisions", "## Key Decisions\n\n")
		for _, d := range state.Memory.Decisions {
			decSB.WriteString(fmtOrFallback("decisions", func(t *sectionTemplate) string { return t.DecisionFormat }, "- Phase %d: %s — %s\n", d.Phase, d.Claim, d.Rationale))
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
		writeSectionHeader(&learnSB, "learnings", "## Phase Learnings\n\n")
		for _, pl := range state.Memory.PhaseLearnings {
			learnSB.WriteString(fmtOrFallback("learnings", func(t *sectionTemplate) string { return t.PhaseHeaderFormat }, "### Phase %d: %s\n", pl.Phase, pl.PhaseName))
			for _, l := range pl.Learnings {
				learnSB.WriteString(fmtOrFallback("learnings", func(t *sectionTemplate) string { return t.LearningFormat }, "  - %s [%s]\n", l.Claim, l.Status))
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

	if handoffSection := renderWorkerHandoffSection("build", state.CurrentPhase, ""); strings.TrimSpace(handoffSection) != "" {
		sections = append(sections, colonyPrimeSection{
			name:              "worker_handoffs",
			title:             "Previous Worker Handoffs",
			source:            filepath.Join(store.BasePath(), workerHandoffsPath),
			content:           handoffSection,
			priority:          4,
			freshnessScore:    0.95,
			confirmationScore: 0.70,
			relevanceScore:    0.75,
		})
	}

	hubDir := resolveHubPath()
	var fallbacks []string
	repoRoot := ""
	if store != nil && strings.TrimSpace(store.BasePath()) != "" {
		repoRoot = filepath.Dir(filepath.Dir(store.BasePath()))
	}
	hiveEntries := readHiveWisdomEntriesForDomains(hubDir, 5, readRegistryDomainsForRepo(hubDir, repoRoot), &fallbacks)

	// Surface why hive wisdom was withheld. Retrieval is default-on per D-02 —
	// there is no per-colony opt-in state to distinguish anymore, so these
	// reasons always surface unconditionally rather than being silently
	// dropped. A colony that expects wisdom and sees none needs to know
	// whether the hub is empty, the domain didn't match, everything decayed
	// to dormant, or AETHER_HIVE_POLICY=off disabled retrieval entirely.
	result.Warnings = append(result.Warnings, fallbacks...)

	hiveLines := buildHiveWisdomLines(hiveEntries)
	if len(hiveLines) > 0 {
		var hiveSB strings.Builder
		writeSectionHeader(&hiveSB, "hive_wisdom", "## HIVE WISDOM (Cross-Colony Patterns)\n\n")
		for _, entry := range hiveLines {
			hiveSB.WriteString(fmtOrFallback("hive_wisdom", func(t *sectionTemplate) string { return t.EntryFormat }, "- %s\n", entry))
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

	// Learned Memory -- durable learning entries from successful builds (D-13, D-14, D-15, HIVE-03)
	// D-15: colony-prime re-assembles per dispatch, so the snapshot refreshes between waves automatically.
	learnStore := learn.NewColonyStore(store)
	learnEntries, _ := learnStore.List(learn.EntryFilter{
		MinConfidence: 0.3, // filter out very low confidence
		Limit:         20,  // cap entries to prevent budget exhaustion (Pitfall 5)
	})
	if len(learnEntries) > 0 {
		var learnSB strings.Builder
		writeSectionHeader(&learnSB, "learned_memory", "## LEARNED MEMORY (Verified Outcomes)\n\n")
		for _, entry := range learnEntries {
			learnSB.WriteString(fmtOrFallback("learned_memory", func(t *sectionTemplate) string { return t.EntryFormat }, "- [Phase %d] %s (confidence: %.0f%%, classification: %s)\n",
				entry.Phase, entry.Content, entry.Confidence*100, entry.Classification))
		}

		// Compute scores per D-13: phase -> priority, recency -> freshness, confidence -> confirmation
		latestEntry := learnEntries[len(learnEntries)-1]
		learnFreshness := 0.5
		if latestEntry.Evidence.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, latestEntry.Evidence.Timestamp); err == nil {
				hoursSince := time.Since(t).Hours()
				if hoursSince < 24 {
					learnFreshness = 0.95
				} else if hoursSince < 72 {
					learnFreshness = 0.8
				} else if hoursSince < 168 {
					learnFreshness = 0.6
				}
			}
		}
		learnConfidence := 0.5
		for _, e := range learnEntries {
			learnConfidence += e.Confidence / float64(len(learnEntries))
		}
		if learnConfidence > 1.0 {
			learnConfidence = 1.0
		}

		sections = append(sections, colonyPrimeSection{
			name:              "learned_memory",
			title:             "Learned Memory",
			source:            filepath.Join(store.BasePath(), "entries.json"),
			content:           learnSB.String(),
			priority:          5, // same as global queen wisdom (per D-13)
			freshnessScore:    learnFreshness,
			confirmationScore: learnConfidence,
			relevanceScore:    sectionRelevanceScore("learned_memory"),
		})
	}

	// Global QUEEN.md wisdom (cross-colony, hub-level)
	globalQueenPath := filepath.Join(hubDir, "QUEEN.md")
	globalWisdom := readQUEENMd(globalQueenPath)
	if len(globalWisdom) > 0 {
		var gwSB strings.Builder
		writeSectionHeader(&gwSB, "global_queen_md", "## GLOBAL QUEEN WISDOM (Cross-Colony)\n\n")
		for _, v := range globalWisdom {
			gwSB.WriteString(fmtOrFallback("global_queen_md", func(t *sectionTemplate) string { return t.EntryFormat }, "- %s\n", v))
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
		writeSectionHeader(&prefsSB, "user_preferences", "## USER PREFERENCES\n\n")
		for _, pref := range userPrefs {
			prefsSB.WriteString(fmtOrFallback("user_preferences", func(t *sectionTemplate) string { return t.EntryFormat }, "- %s\n", pref))
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
		writeSectionHeader(&lwSB, "local_queen_wisdom", "## LOCAL QUEEN WISDOM (Repo-Specific)\n\n")
		for _, v := range localWisdom {
			lwSB.WriteString(fmtOrFallback("local_queen_wisdom", func(t *sectionTemplate) string { return t.EntryFormat }, "- %s\n", v))
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

	clarifiedIntent := clarifiedIntentPromptRenderResultForScope(pendingDecisionScopeFromState(state))
	if len(clarifiedIntent.Warnings) > 0 {
		result.Warnings = append(result.Warnings, clarifiedIntent.Warnings...)
	}
	if len(clarifiedIntent.Blocked) > 0 {
		result.Ledger.Blocked = append(result.Ledger.Blocked, clarifiedIntent.Blocked...)
	}
	if len(clarifiedIntent.Lines) > 0 {
		var clarifySB strings.Builder
		writeSectionHeader(&clarifySB, "clarified_intent", "## CLARIFIED INTENT\n\n")
		for _, clarification := range clarifiedIntent.Lines {
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
				writeSectionHeader(&blockerSB, "blockers", "## Active Blockers\n\n")
			}
			blockerSB.WriteString(fmtOrFallback("blockers", func(t *sectionTemplate) string { return t.BlockerFormat }, "- %s\n", blocker.Description))
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

	// Midden (recent failures) -- 188-VERIFICATION.md Gap 1: this function
	// (via resolveCodexWorkerContext -> buildColonyPrimeOutput, the one
	// every live build/continue/colonize/plan/seal/swarm worker dispatch
	// actually calls) had zero midden-reading code, before or after this
	// phase's original five plans. Uses the canonical shared helper
	// (cmd/midden_shared.go) -- never a hand-built read path, which is
	// exactly the split Phase 188 existed to end. Filtered to
	// still-unacknowledged entries only (an acknowledged failure has
	// already been handled -- resurfacing it forever would grow this
	// section without bound) and hard-capped at
	// middenCapsuleSectionEntryLimit, newest first, so the "protected"
	// (never-trimmed) status this section carries (see
	// protectedSectionPolicy's "midden" case) stays safe no matter how many
	// failures accumulate in midden.json over a colony's lifetime.
	if midden, middenErr := loadMiddenFile(store); middenErr == nil && len(midden.Entries) > 0 {
		unacked := make([]colony.MiddenEntry, 0, len(midden.Entries))
		for _, entry := range midden.Entries {
			if entry.Acknowledged != nil && *entry.Acknowledged {
				continue
			}
			unacked = append(unacked, entry)
		}
		if len(unacked) > 0 {
			sort.SliceStable(unacked, func(i, j int) bool {
				return unacked[i].Timestamp > unacked[j].Timestamp
			})
			const middenCapsuleSectionEntryLimit = 5
			shown := unacked
			remaining := 0
			if len(shown) > middenCapsuleSectionEntryLimit {
				remaining = len(shown) - middenCapsuleSectionEntryLimit
				shown = shown[:middenCapsuleSectionEntryLimit]
			}
			var middenSB strings.Builder
			writeSectionHeader(&middenSB, "midden", "## Recent Failures\n\n")
			middenTimestamps := make([]string, 0, len(shown))
			for _, entry := range shown {
				middenSB.WriteString(fmtOrFallback("midden", func(t *sectionTemplate) string { return t.EntryFormat }, "- [%s] %s\n", entry.Category, truncateString(entry.Message, 160)))
				middenTimestamps = append(middenTimestamps, entry.Timestamp)
			}
			if remaining > 0 {
				fmt.Fprintf(&middenSB, "+%d more unacknowledged\n", remaining)
			}
			middenProtected, middenPreserveReason := protectedSectionPolicy("midden")
			sections = append(sections, colonyPrimeSection{
				name:              "midden",
				title:             "Recent Failures",
				source:            filepath.Join(store.BasePath(), middenCanonicalPath),
				content:           middenSB.String(),
				priority:          9,
				freshnessScore:    latestFreshnessScore(now, 0.75, middenTimestamps...),
				confirmationScore: 1.0,
				relevanceScore:    sectionRelevanceScore("midden"),
				protected:         middenProtected,
				preserveReason:    middenPreserveReason,
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
			writeSectionHeader(&healthSB, "medic_health", "## Colony Health Issues\n\n")
			healthSB.WriteString(fmtOrFallback("medic_health", func(t *sectionTemplate) string { return t.ScanTimestampFormat }, "Last scan: %s\n\n", lastScan.Timestamp))
			for _, issue := range criticalIssues {
				healthSB.WriteString(fmtOrFallback("medic_health", func(t *sectionTemplate) string { return t.IssueFormat }, "- [%s] %s", issue.Severity, issue.Message))
				if issue.File != "" {
					healthSB.WriteString(fmtOrFallback("medic_health", func(t *sectionTemplate) string { return t.IssueFileFormat }, " (%s)", issue.File))
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

	// Ask mode: recent activity is history colony-prime never carried —
	// state.Events is frequently empty while activity.log holds the real
	// feed — and a question about "what happened" needs it.
	if strings.TrimSpace(opts.Question) != "" {
		if tail := buildActivityTailSection(); tail != nil {
			sections = append(sections, *tail)
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
		// Question-aware relevance: a small additive boost for sections
		// whose content overlaps the question's words, on top of the static
		// per-section score — no new ranking system.
		sec.relevanceScore += questionRelevanceBoost(opts.Question, sec)
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

	// Cross-project wisdom adoption is never silent: the ledger records
	// exactly which entries were retrieved into this colony's context.
	if len(hiveEntries) > 0 {
		hiveIDs := make([]string, 0, len(hiveEntries))
		for _, entry := range hiveEntries {
			hiveIDs = append(hiveIDs, entry.ID)
		}
		recordHiveRetrieval := func(items []colonyPrimeLedgerItem) {
			for i := range items {
				if items[i].Name == "hive_wisdom" {
					items[i].Decision = strings.TrimSpace(items[i].Decision + "; retrieved entries: " + strings.Join(hiveIDs, ", "))
				}
			}
		}
		recordHiveRetrieval(result.Ledger.Included)
		recordHiveRetrieval(result.Ledger.Trimmed)
	}

	context := strings.TrimSpace(assembled.String())
	result.Context = context
	result.PromptSection = context
	result.Used = ranking.Used
	result.LogLine = fmt.Sprintf("colony-prime loaded %d signal(s), %d instinct(s), %d review(s), used %d/%d chars", result.SignalCount, result.InstinctCount, result.ReviewCount, ranking.Used, budget)
	return result
}

// questionRelevanceBoost scores how much a section's content overlaps the
// question's words — keyword overlap only, deliberately: it nudges ranking
// under budget pressure so the sections a question is ABOUT survive the
// trim; it never invents relevance. Zero when there is no question.
func questionRelevanceBoost(question string, sec colonyPrimeSection) float64 {
	question = strings.ToLower(strings.TrimSpace(question))
	if question == "" {
		return 0
	}
	haystack := strings.ToLower(sec.name + " " + sec.title + " " + sec.content)
	matched := 0
	total := 0
	for _, word := range strings.Fields(question) {
		word = strings.Trim(word, "?.,!\"'")
		if len(word) < 4 {
			continue
		}
		total++
		if strings.Contains(haystack, word) {
			matched++
		}
	}
	if total == 0 || matched == 0 {
		return 0
	}
	return 2.0 * float64(matched) / float64(total)
}

// buildActivityTailSection carries the last entries of activity.log — the
// history feed an ask question about "what happened" needs. state.Events is
// frequently empty (nothing durable writes it between phases) while
// activity.log holds the real per-command record; /ant-history reads only
// the former, which is exactly why "what changed?" had no good answer.
func buildActivityTailSection() *colonyPrimeSection {
	if store == nil {
		return nil
	}
	lines, err := store.ReadJSONL("activity.log")
	if err != nil || len(lines) == 0 {
		return nil
	}
	const activityTailMax = 20
	if len(lines) > activityTailMax {
		lines = lines[len(lines)-activityTailMax:]
	}
	var b strings.Builder
	b.WriteString("## Recent Activity\n\n")
	for _, raw := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		ts := strings.TrimSpace(stringValue(entry["timestamp"]))
		action := strings.TrimSpace(stringValue(entry["action"]))
		detail := strings.TrimSpace(stringValue(entry["detail"]))
		if action == "" {
			continue
		}
		if ts != "" {
			fmt.Fprintf(&b, "- %s %s", ts, action)
		} else {
			fmt.Fprintf(&b, "- %s", action)
		}
		if detail != "" {
			fmt.Fprintf(&b, " — %s", detail)
		}
		b.WriteString("\n")
	}
	protected, preserveReason := protectedSectionPolicy("activity_tail")
	return &colonyPrimeSection{
		name:              "activity_tail",
		title:             "Recent Activity",
		source:            filepath.Join(store.BasePath(), "activity.log"),
		content:           b.String(),
		priority:          6,
		freshnessScore:    1.0,
		confirmationScore: 1.0,
		relevanceScore:    sectionRelevanceScore("activity_tail"),
		protected:         protected,
		preserveReason:    preserveReason,
	}
}

func resolveCodexWorkerContext() string {
	context, _ := resolveCodexWorkerContextWithTrim()
	return context
}

// resolveCodexWorkerContextWithTrim returns the same capsule
// resolveCodexWorkerContext returns, plus the names of the sections the token
// budget dropped while assembling it.
//
// The trim list exists so a caller can tell a DELIBERATE omission from a
// SILENT one. --print-brief needs exactly that distinction (190-190/WR-02):
// a steering section that is absent because the budget evicted it is a real
// but explicable finding about this colony's context pressure, while a
// section absent with no eviction on record is a delivery defect. Reporting
// both as the same failure would either cry wolf on a busy colony or stay
// quiet on a genuine drop.
//
// Callers that only need the text keep using resolveCodexWorkerContext, which
// delegates here so there is one assembly path and the capsule's side effects
// (hive retrieval recording, ledger writes) happen once per call, not twice.
func resolveCodexWorkerContextWithTrim() (string, []string) {
	output := buildColonyPrimeOutput(true)
	context := strings.TrimSpace(output.PromptSection)
	trimmed := append([]string(nil), output.Trimmed...)
	if context == "" {
		// Fallback assembly: a different builder with its own budget, so the
		// colony-prime trim ledger above does not describe it.
		context = buildContextCapsuleOutput(true, 8, 3, 2, 220).PromptSection
		trimmed = nil
	}
	if len(context) < 128 {
		fmt.Fprintf(os.Stderr, "⚠ Context capsule below minimum threshold (%d chars, min 128) — dispatch blocked to prevent zero-context worker execution\n", len(context))
		return "", trimmed
	}
	return context, trimmed
}
