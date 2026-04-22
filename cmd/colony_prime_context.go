package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
)

type colonyPrimeOutput struct {
	Context       string            `json:"context"`
	PromptSection string            `json:"prompt_section"`
	SignalCount   int               `json:"signal_count"`
	InstinctCount int               `json:"instinct_count"`
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

	queenPath := filepath.Join(hubDir, "QUEEN.md")
	userPrefs := readUserPreferences(queenPath)
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
	result.LogLine = fmt.Sprintf("colony-prime loaded %d signal(s), %d instinct(s), used %d/%d chars", result.SignalCount, result.InstinctCount, ranking.Used, budget)
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
