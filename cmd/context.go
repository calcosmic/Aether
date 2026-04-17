package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// ContextCapsuleOutput is the typed output for context-capsule (DIFF-02).
// The shell version uses jq string interpolation; this uses typed structs with JSON marshaling.
type ContextCapsuleOutput struct {
	Exists        bool   `json:"exists"`
	State         string `json:"state"`
	NextAction    string `json:"next_action"`
	WordCount     int    `json:"word_count"`
	PromptSection string `json:"prompt_section"`
	Goal          string `json:"goal"`
	Phase         int    `json:"phase"`
	TotalPhases   int    `json:"total_phases"`
	PhaseName     string `json:"phase_name"`
}

// resumeDashboardCmd returns a read-only session recovery dashboard.
var resumeDashboardCmd = &cobra.Command{
	Use:   "resume-dashboard",
	Short: "Show session recovery information without restoring handoff state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		result := buildResumeDashboardResult()
		outputWorkflow(result, renderResumeVisual(result, "", false))
		return nil
	},
}

func buildResumeDashboardResult() map[string]interface{} {
	var session colony.SessionFile
	sessionFound := store.LoadJSON("session.json", &session) == nil
	if !sessionFound {
		if restored, err := ensureLegacySessionMirror(store); err == nil && restored {
			sessionFound = store.LoadJSON("session.json", &session) == nil
		}
	}
	handoffExists := false
	if _, err := os.Stat(handoffDocumentPath()); err == nil {
		handoffExists = true
	}

	// Load COLONY_STATE.json. If missing, return defaults.
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		result := map[string]interface{}{
			"current": map[string]interface{}{
				"phase":         0,
				"phase_name":    "",
				"total_phases":  0,
				"state":         "UNKNOWN",
				"goal":          "",
				"parallel_mode": "in-repo",
			},
			"memory_health": map[string]interface{}{
				"wisdom_count":       0,
				"pending_promotions": 0,
				"recent_failures":    0,
			},
			"data_safety": map[string]interface{}{},
			"recent": map[string]interface{}{
				"decisions": []interface{}{},
				"events":    []interface{}{},
			},
			"drill_down": map[string]interface{}{
				"command":   "aether memory-details",
				"available": true,
			},
		}
		if sessionFound {
			result["session"] = map[string]interface{}{
				"summary":         session.Summary,
				"suggested_next":  session.SuggestedNext,
				"active_todos":    session.ActiveTodos,
				"last_command":    session.LastCommand,
				"context_cleared": session.ContextCleared,
			}
		}
		result["recovery"] = map[string]interface{}{
			"context_path":   contextDocumentPath(),
			"handoff_path":   handoffDocumentPath(),
			"handoff_exists": handoffExists,
		}
		return result
	}

	phaseName := lookupPhaseName(state, state.CurrentPhase)
	if state.CurrentPhase <= 0 {
		phaseName = ""
	}
	recovery := recoveryPhase(&state)
	nextPhaseID := 0
	nextPhaseName := ""
	if recovery != nil && recovery.ID != state.CurrentPhase {
		nextPhaseID = recovery.ID
		nextPhaseName = recovery.Name
	}

	// Extract core state fields
	currentPhase := state.CurrentPhase
	stateStr := string(state.State)
	goal := "No goal set"
	if state.Goal != nil && *state.Goal != "" {
		goal = *state.Goal
	}
	totalPhases := len(state.Plan.Phases)

	// Extract parallel_mode with default fallback
	parallelMode := string(state.ParallelMode)
	if parallelMode == "" {
		parallelMode = "in-repo"
	}

	// Compute memory health inline
	wisdomCount := 0
	pendingCount := 0
	var learnings colony.LearningFile
	if err := store.LoadJSON("learning-observations.json", &learnings); err == nil {
		for _, obs := range learnings.Observations {
			if obs.TrustScore != nil {
				wisdomCount++
			} else {
				pendingCount++
			}
		}
	}

	failureCount := 0
	var midden colony.MiddenFile
	if err := store.LoadJSON("midden/midden.json", &midden); err == nil {
		failureCount = len(midden.Entries)
	}

	// Extract recent decisions (last 5, reversed)
	recentDecisions := extractRecentDecisions(state.Memory.Decisions, 5)

	// Extract recent events (last 10, wrapped as objects)
	recentEvents := extractRecentEvents(state.Events, 10)

	// Load data safety stats
	dataSafety := map[string]interface{}{}
	var safetyStats map[string]interface{}
	if raw, err := store.ReadFile("safety-stats.json"); err == nil {
		json.Unmarshal(raw, &safetyStats)
		if safetyStats != nil {
			dataSafety = safetyStats
		}
	}

	result := map[string]interface{}{
		"current": map[string]interface{}{
			"phase":         currentPhase,
			"phase_name":    phaseName,
			"total_phases":  totalPhases,
			"state":         stateStr,
			"goal":          goal,
			"parallel_mode": parallelMode,
		},
		"memory_health": map[string]interface{}{
			"wisdom_count":       wisdomCount,
			"pending_promotions": pendingCount,
			"recent_failures":    failureCount,
		},
		"data_safety": dataSafety,
		"recent": map[string]interface{}{
			"decisions": recentDecisions,
			"events":    recentEvents,
		},
		"drill_down": map[string]interface{}{
			"command":   "aether memory-details",
			"available": true,
		},
		"recovery": map[string]interface{}{
			"context_path":   contextDocumentPath(),
			"handoff_path":   handoffDocumentPath(),
			"handoff_exists": handoffExists,
		},
	}

	if nextPhaseID > 0 {
		result["next_phase"] = map[string]interface{}{
			"id":   nextPhaseID,
			"name": nextPhaseName,
		}
	}
	if sessionFound {
		todos := session.ActiveTodos
		if len(todos) == 0 {
			todos = sessionActiveTodosFromState(state)
		}
		suggestedNext := strings.TrimSpace(session.SuggestedNext)
		if suggestedNext == "" {
			suggestedNext = nextCommandFromState(state)
		}
		summary := strings.TrimSpace(session.Summary)
		if summary == "" {
			summary = defaultProgressSummary(state, suggestedNext)
		}
		result["session"] = map[string]interface{}{
			"summary":         summary,
			"suggested_next":  suggestedNext,
			"active_todos":    todos,
			"last_command":    session.LastCommand,
			"context_cleared": session.ContextCleared,
		}
	}
	return result
}

// contextCapsuleCmd assembles worker context for prompt injection (CMD-16).
var contextCapsuleCmd = &cobra.Command{
	Use:   "context-capsule",
	Short: "Assemble worker context for prompt injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		compact, _ := cmd.Flags().GetBool("compact")
		maxSignals, _ := cmd.Flags().GetInt("max-signals")
		maxDecisions, _ := cmd.Flags().GetInt("max-decisions")
		maxRisks, _ := cmd.Flags().GetInt("max-risks")
		maxWords, _ := cmd.Flags().GetInt("max-words")
		outputOK(buildContextCapsuleOutput(compact, maxSignals, maxDecisions, maxRisks, maxWords))
		return nil
	},
}

func buildContextCapsuleOutput(compact bool, maxSignals, maxDecisions, maxRisks, maxWords int) ContextCapsuleOutput {
	if store == nil {
		return ContextCapsuleOutput{
			Exists:        false,
			WordCount:     0,
			PromptSection: "",
		}
	}
	if maxSignals < 1 {
		maxSignals = 1
	}
	if maxDecisions < 1 {
		maxDecisions = 1
	}
	if maxRisks < 1 {
		maxRisks = 1
	}
	if maxWords < 80 {
		maxWords = 80
	}

	sc := cache.NewSessionCache(store.BasePath())
	sc.ClearStale(24 * time.Hour)

	var state colony.ColonyState
	statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
	if err := sc.Load(statePath, &state); err != nil {
		return ContextCapsuleOutput{
			Exists:        false,
			WordCount:     0,
			PromptSection: "",
		}
	}

	goal := "No goal set"
	if state.Goal != nil {
		goal = *state.Goal
	}
	goal = truncateString(goal, 160)

	stateStr := string(state.State)
	if stateStr == "" {
		stateStr = "IDLE"
	}

	phase := state.CurrentPhase
	totalPhases := len(state.Plan.Phases)
	phaseName := lookupPhaseName(state, phase)
	nextAction := computeNextAction(stateStr, phase, totalPhases)

	decisionTexts := extractDecisionTexts(state.Memory.Decisions, maxDecisions)
	riskTexts := extractRiskTexts(maxRisks)
	pf, _ := loadPheromonesOnce(store, sc)
	signalTexts := extractSignalTextsFrom(&pf, maxSignals)
	summaryTexts := extractRollingSummary(3)

	var b strings.Builder
	b.WriteString("--- CONTEXT CAPSULE ---\n")
	fmt.Fprintf(&b, "Goal: %s\n", goal)
	fmt.Fprintf(&b, "State: %s\n", stateStr)
	fmt.Fprintf(&b, "Phase: %d/%d - %s\n", phase, totalPhases, phaseName)
	fmt.Fprintf(&b, "Next: %s\n", nextAction)

	if len(signalTexts) > 0 {
		b.WriteString("\nActive signals:\n")
		for _, s := range signalTexts {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}

	if len(decisionTexts) > 0 {
		b.WriteString("\nRecent decisions:\n")
		for _, d := range decisionTexts {
			fmt.Fprintf(&b, "- %s\n", d)
		}
	}

	if len(riskTexts) > 0 {
		b.WriteString("\nOpen risks:\n")
		for _, r := range riskTexts {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}

	if len(summaryTexts) > 0 {
		b.WriteString("\nRecent narrative:\n")
		for _, s := range summaryTexts {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}

	b.WriteString("--- END CONTEXT CAPSULE ---\n")

	promptSection := b.String()
	wc := wordCount(promptSection)
	if compact && wc > maxWords {
		promptSection = trimSection(promptSection, "Recent narrative:")
		wc = wordCount(promptSection)
	}
	if compact && wc > maxWords {
		promptSection = trimSection(promptSection, "Open risks:")
		wc = wordCount(promptSection)
	}

	return ContextCapsuleOutput{
		Exists:        true,
		State:         stateStr,
		NextAction:    nextAction,
		WordCount:     wc,
		PromptSection: promptSection,
		Goal:          goal,
		Phase:         phase,
		TotalPhases:   totalPhases,
		PhaseName:     phaseName,
	}
}

// PRContextOutput is the typed output for pr-context (CMD-17).
// It assembles comprehensive context from 10+ sources for CI agents and worker spawning.
type PRContextOutput struct {
	Schema          string                 `json:"schema"`
	GeneratedAt     string                 `json:"generated_at"`
	Branch          string                 `json:"branch"`
	CacheStatus     map[string]string      `json:"cache_status"`
	Queen           map[string]interface{} `json:"queen"`
	Signals         map[string]interface{} `json:"signals"`
	Hive            map[string]interface{} `json:"hive"`
	ColonyState     map[string]interface{} `json:"colony_state"`
	Blockers        map[string]interface{} `json:"blockers"`
	Decisions       map[string]interface{} `json:"decisions"`
	Midden          map[string]interface{} `json:"midden"`
	PromptSection   string                 `json:"prompt_section"`
	CharCount       int                    `json:"char_count"`
	Budget          int                    `json:"budget"`
	TrimmedSections []string               `json:"trimmed_sections"`
	Warnings        []string               `json:"warnings"`
	FallbacksUsed   []string               `json:"fallbacks_used"`
}

// prContextCmd assembles comprehensive CI agent context from 10+ data sources (CMD-17).
var prContextCmd = &cobra.Command{
	Use:   "pr-context",
	Short: "Assemble CI agent context from multiple data sources",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		compact, _ := cmd.Flags().GetBool("compact")
		branch, _ := cmd.Flags().GetString("branch")
		ciRunID, _ := cmd.Flags().GetString("ci-run-id")

		budget := 6000
		if compact {
			budget = 3000
		}

		// Auto-detect branch if not provided
		if branch == "" {
			branch = detectGitBranch()
		}

		// Cache git process results for this invocation.
		aetherRoot := resolveAetherRootPath()

		var fallbacks []string
		cacheStatus := map[string]string{}

		// Initialize session cache for this invocation. All JSON file reads below
		// go through the cache to avoid redundant disk I/O within the same command.
		sc := cache.NewSessionCache(store.BasePath())

		// Auto-cleanup stale cache files older than 24 hours (non-blocking)
		sc.ClearStale(24 * time.Hour)

		// Pre-load COLONY_STATE.json once for all sections that need it.
		// Each section preserves its own error-handling semantics via colStateErr.
		var colState colony.ColonyState
		colStateErr := sc.Load(filepath.Join(store.BasePath(), "COLONY_STATE.json"), &colState)

		// 1. queen_global: Read ~/.aether/QUEEN.md
		hubDir := resolveHubPath()
		queenGlobal := readQUEENMd(filepath.Join(hubDir, "QUEEN.md"))
		if queenGlobal == nil {
			queenGlobal = map[string]string{}
			fallbacks = append(fallbacks, "queen_global: no file found")
		}
		cacheStatus["queen_global"] = "read"

		// 2. queen_local: Read AETHER_ROOT/.aether/QUEEN.md
		queenLocal := readQUEENMd(filepath.Join(aetherRoot, ".aether", "QUEEN.md"))
		if queenLocal == nil {
			queenLocal = map[string]string{}
			fallbacks = append(fallbacks, "queen_local: no file found")
		}
		cacheStatus["queen_local"] = "read"

		// 3. user_preferences: Extract from queen files
		var userPrefs []string
		userPrefs = append(userPrefs, readUserPreferences(filepath.Join(hubDir, "QUEEN.md"))...)
		userPrefs = append(userPrefs, readUserPreferences(filepath.Join(aetherRoot, ".aether", "QUEEN.md"))...)

		// 4. signals: Load pheromones and classify (shared loader)
		var redirects, focusSignals, feedbackSignals []string
		var instincts []colony.Instinct
		signalCount := 0
		pf, _ := loadPheromonesOnce(store, sc)
		if pf.Signals != nil {
			for _, sig := range pf.Signals {
				if !sig.Active {
					continue
				}
				text := extractSignalText(sig.Content)
				if text == "" {
					continue
				}
				signalCount++
				switch sig.Type {
				case "REDIRECT":
					redirects = append(redirects, text)
				case "FOCUS":
					focusSignals = append(focusSignals, text)
				case "FEEDBACK":
					feedbackSignals = append(feedbackSignals, text)
				}
			}
		} else {
			fallbacks = append(fallbacks, "pheromones: no active signals")
		}

		// Also load instincts from the standalone instincts store, falling back to state.
		if colStateErr == nil {
			instincts = loadRuntimeInstincts(store, &colState)
		}

		signalsMap := map[string]interface{}{
			"count":     signalCount,
			"redirects": redirects,
			"focus":     focusSignals,
			"feedback":  feedbackSignals,
			"instincts": len(instincts),
		}
		cacheStatus["signals"] = "read"

		// 5. hive_wisdom: Read from hive, fallback to eternal
		hiveEntries := readHiveWisdom(hubDir, 5, &fallbacks)
		hiveMap := map[string]interface{}{
			"entries": hiveEntries,
			"count":   len(hiveEntries),
		}
		cacheStatus["hive_wisdom"] = "read"

		// 6. colony_state: Use pre-loaded COLONY_STATE.json
		colonyStateMap := map[string]interface{}{"exists": false}
		if colStateErr != nil {
			fallbacks = append(fallbacks, "colony_state: COLONY_STATE.json missing")
			cacheStatus["colony_state"] = "missing"
		} else {
			goal := ""
			if colState.Goal != nil {
				goal = *colState.Goal
			}
			phaseName := lookupPhaseName(colState, colState.CurrentPhase)
			colonyStateMap = map[string]interface{}{
				"exists":        true,
				"goal":          goal,
				"state":         string(colState.State),
				"current_phase": colState.CurrentPhase,
				"total_phases":  len(colState.Plan.Phases),
				"phase_name":    phaseName,
			}
			cacheStatus["colony_state"] = "read"
		}

		// 7. blockers: Load flags.json, filter unresolved
		blockerList := extractBlockerTexts()
		if blockerList == nil {
			fallbacks = append(fallbacks, "blockers: flags.json missing")
			blockerList = []string{}
			cacheStatus["blockers"] = "missing"
		} else {
			cacheStatus["blockers"] = "read"
		}
		blockersMap := map[string]interface{}{
			"count": len(blockerList),
			"items": blockerList,
		}

		// 8. decisions: From pre-loaded COLONY_STATE.json (silently skip if missing)
		var decisionClaims []string
		if colStateErr == nil {
			decisionTexts := extractDecisionTexts(colState.Memory.Decisions, 10)
			decisionClaims = decisionTexts
		}
		decisionsMap := map[string]interface{}{
			"count": len(decisionClaims),
			"items": decisionClaims,
		}
		cacheStatus["decisions"] = "read"

		// 9. midden: Load midden.json
		middenMap := map[string]interface{}{"count": 0, "items": []string{}}
		var midden colony.MiddenFile
		if err := sc.Load(filepath.Join(store.BasePath(), "midden", "midden.json"), &midden); err != nil {
			fallbacks = append(fallbacks, "midden: midden.json missing")
			cacheStatus["midden"] = "missing"
		} else {
			// Sort by timestamp descending, take last 10
			entries := midden.Entries
			sort.SliceStable(entries, func(i, j int) bool {
				return entries[i].Timestamp > entries[j].Timestamp
			})
			var middenItems []string
			limit := 10
			if len(entries) < limit {
				limit = len(entries)
			}
			for i := 0; i < limit; i++ {
				middenItems = append(middenItems, truncateString(entries[i].Message, 160))
			}
			middenMap = map[string]interface{}{
				"count": len(entries),
				"items": middenItems,
			}
			cacheStatus["midden"] = "read"
		}

		// 10. context_capsule: Build inline (silently skip if colony state missing)
		var capsulePromptSection string
		if colStateErr == nil {
			goal := "No goal set"
			if colState.Goal != nil {
				goal = *colState.Goal
			}
			stateStr := string(colState.State)
			if stateStr == "" {
				stateStr = "IDLE"
			}
			phase := colState.CurrentPhase
			totalPhases := len(colState.Plan.Phases)
			phaseName := lookupPhaseName(colState, phase)
			nextAction := computeNextAction(stateStr, phase, totalPhases)

			var cb strings.Builder
			cb.WriteString("--- CONTEXT CAPSULE ---\n")
			fmt.Fprintf(&cb, "Goal: %s\n", truncateString(goal, 160))
			fmt.Fprintf(&cb, "State: %s\n", stateStr)
			fmt.Fprintf(&cb, "Phase: %d/%d - %s\n", phase, totalPhases, phaseName)
			fmt.Fprintf(&cb, "Next: %s\n", nextAction)
			cb.WriteString("--- END CONTEXT CAPSULE ---\n")
			capsulePromptSection = cb.String()
		}
		cacheStatus["context_capsule"] = "read"

		// 11. phase_learnings: From pre-loaded COLONY_STATE.json (silently skip if missing)
		var learningSummaries []string
		if colStateErr == nil {
			for _, pl := range colState.Memory.PhaseLearnings {
				for _, l := range pl.Learnings {
					learningSummaries = append(learningSummaries, truncateString(l.Claim, 160))
				}
			}
		}

		// 12. rolling_summary: last 20 lines
		rollingLines := extractRollingSummary(20)

		// Build prompt_section text
		var b strings.Builder

		// QUEEN WISDOM (Global)
		if len(queenGlobal) > 0 {
			b.WriteString(buildPRSectionHeader("QUEEN WISDOM (Global)"))
			for k, v := range queenGlobal {
				fmt.Fprintf(&b, "%s: %s\n", k, v)
			}
		}

		// QUEEN WISDOM (Local)
		if len(queenLocal) > 0 {
			b.WriteString(buildPRSectionHeader("QUEEN WISDOM (Local)"))
			for k, v := range queenLocal {
				fmt.Fprintf(&b, "%s: %s\n", k, v)
			}
		}

		// USER PREFERENCES
		if len(userPrefs) > 0 {
			b.WriteString(buildPRSectionHeader("USER PREFERENCES"))
			for _, pref := range userPrefs {
				fmt.Fprintf(&b, "- %s\n", pref)
			}
		}

		// ACTIVE SIGNALS
		if signalCount > 0 {
			b.WriteString(buildPRSectionHeader("ACTIVE SIGNALS (Colony Guidance)"))
			if len(redirects) > 0 {
				b.WriteString("REDIRECT (HARD CONSTRAINTS):\n")
				for _, r := range redirects {
					fmt.Fprintf(&b, "- %s\n", r)
				}
			}
			if len(focusSignals) > 0 {
				b.WriteString("FOCUS (Active Guidance):\n")
				for _, f := range focusSignals {
					fmt.Fprintf(&b, "- %s\n", f)
				}
			}
			if len(feedbackSignals) > 0 {
				b.WriteString("FEEDBACK (Adjustments):\n")
				for _, fb := range feedbackSignals {
					fmt.Fprintf(&b, "- %s\n", fb)
				}
			}
			b.WriteString("--- END SIGNALS ---\n")
		}

		// HIVE WISDOM
		if len(hiveEntries) > 0 {
			b.WriteString(buildPRSectionHeader("HIVE WISDOM (Cross-Colony Patterns)"))
			for _, e := range hiveEntries {
				fmt.Fprintf(&b, "- %s\n", e)
			}
		}

		// CONTEXT CAPSULE
		if capsulePromptSection != "" {
			b.WriteString(capsulePromptSection)
		}

		// PHASE LEARNINGS
		if len(learningSummaries) > 0 {
			b.WriteString(buildPRSectionHeader("PHASE LEARNINGS"))
			for _, l := range learningSummaries {
				fmt.Fprintf(&b, "- %s\n", l)
			}
		}

		// KEY DECISIONS
		if len(decisionClaims) > 0 {
			b.WriteString(buildPRSectionHeader("KEY DECISIONS"))
			for _, d := range decisionClaims {
				fmt.Fprintf(&b, "- %s\n", d)
			}
		}

		// BLOCKERS (NEVER trimmed)
		if len(blockerList) > 0 {
			b.WriteString(buildPRSectionHeader("BLOCKERS (CRITICAL)"))
			for _, bl := range blockerList {
				fmt.Fprintf(&b, "- %s\n", bl)
			}
		}

		// ROLLING SUMMARY
		if len(rollingLines) > 0 {
			b.WriteString(buildPRSectionHeader("ROLLING SUMMARY"))
			for _, r := range rollingLines {
				fmt.Fprintf(&b, "- %s\n", r)
			}
		}

		promptSection := b.String()
		charCount := len(promptSection)

		// Budget enforcement with trim order
		trimmedSections := []string{}
		if charCount > budget {
			trimOrder := []string{
				"--- ROLLING SUMMARY ---",
				"--- PHASE LEARNINGS ---",
				"--- KEY DECISIONS ---",
				"--- HIVE WISDOM",
				"--- CONTEXT CAPSULE ---",
				"--- USER PREFERENCES ---",
				"--- QUEEN WISDOM (Global) ---",
				"--- QUEEN WISDOM (Local) ---",
				"--- ACTIVE SIGNALS",
				// BLOCKERS is NEVER trimmed
			}
			for _, header := range trimOrder {
				if len(promptSection) <= budget {
					break
				}
				if strings.Contains(promptSection, header) {
					trimmed := removePRSection(promptSection, header)
					if trimmed != promptSection {
						promptSection = trimmed
						// Use a clean section name for tracking
						sectionName := strings.Trim(header, "- ")
						sectionName = strings.TrimSpace(sectionName)
						trimmedSections = append(trimmedSections, sectionName)
					}
				}
			}
			charCount = len(promptSection)
		}

		// Build warnings
		var warnings []string
		if len(fallbacks) > 0 {
			warnings = append(warnings, fmt.Sprintf("used %d fallbacks", len(fallbacks)))
		}

		// Preserve ci-run-id in cache status if provided
		if ciRunID != "" {
			cacheStatus["ci_run_id"] = ciRunID
		}

		outputOK(PRContextOutput{
			Schema:          "pr-context-v1",
			GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
			Branch:          branch,
			CacheStatus:     cacheStatus,
			Queen:           map[string]interface{}{"global": queenGlobal, "local": queenLocal},
			Signals:         signalsMap,
			Hive:            hiveMap,
			ColonyState:     colonyStateMap,
			Blockers:        blockersMap,
			Decisions:       decisionsMap,
			Midden:          middenMap,
			PromptSection:   promptSection,
			CharCount:       charCount,
			Budget:          budget,
			TrimmedSections: trimmedSections,
			Warnings:        warnings,
			FallbacksUsed:   fallbacks,
		})
		return nil
	},
}

func init() {
	contextCapsuleCmd.Flags().Bool("compact", false, "Compact mode with word budget")
	contextCapsuleCmd.Flags().Bool("json", false, "Output as JSON only")
	contextCapsuleCmd.Flags().Int("max-signals", 8, "Maximum pheromone signals")
	contextCapsuleCmd.Flags().Int("max-decisions", 3, "Maximum recent decisions")
	contextCapsuleCmd.Flags().Int("max-risks", 2, "Maximum open risks")
	contextCapsuleCmd.Flags().Int("max-words", 220, "Word budget for compact mode")

	prContextCmd.Flags().Bool("compact", false, "Compact mode (3000 char budget)")
	prContextCmd.Flags().String("branch", "", "Git branch (auto-detected if empty)")
	prContextCmd.Flags().String("ci-run-id", "", "CI run identifier")

	contextUpdateCmd.Flags().String("summary", "", "Summary text to append")
	contextUpdateCmd.Flags().Bool("append", false, "Append to existing summary (default: replace)")
	contextUpdateCmd.Flags().String("section", "", "Section to update (e.g., constraint)")
	contextUpdateCmd.Flags().String("key", "", "Key within section (e.g., redirect, focus, feedback)")
	contextUpdateCmd.Flags().String("content", "", "Content for section update")

	rootCmd.AddCommand(resumeDashboardCmd)
	rootCmd.AddCommand(contextCapsuleCmd)
	rootCmd.AddCommand(prContextCmd)
	rootCmd.AddCommand(contextUpdateCmd)
}

// --- Helper functions (file-private) ---

// extractSignalText extracts the text field from a pheromone signal's content.
// It tries parsing as {"text": "..."}, else uses the raw string.
func extractSignalText(content json.RawMessage) string {
	if len(content) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(content, &m); err == nil {
		if text, ok := m["text"].(string); ok {
			return truncateString(strings.TrimSpace(text), 160)
		}
	}
	// Fallback to raw string, stripping quotes
	raw := strings.TrimSpace(string(content))
	raw = strings.Trim(raw, "\"")
	return truncateString(raw, 160)
}

// computeEffectiveStrength computes the effective strength of a signal
// based on time-based decay. Decay periods: FOCUS=30 days, REDIRECT=60, FEEDBACK=90.
func computeEffectiveStrength(signal colony.PheromoneSignal, now time.Time) float64 {
	strength := 1.0
	if signal.Strength != nil {
		strength = *signal.Strength
	}

	createdAt, err := time.Parse(time.RFC3339, signal.CreatedAt)
	if err != nil {
		return strength
	}

	elapsedDays := now.Sub(createdAt).Hours() / 24.0
	if elapsedDays < 0 {
		elapsedDays = 0
	}

	decayDays := 30.0
	switch signal.Type {
	case "FOCUS":
		decayDays = 30.0
	case "REDIRECT":
		decayDays = 60.0
	case "FEEDBACK":
		decayDays = 90.0
	}

	effective := strength * (1.0 - elapsedDays/decayDays)
	if effective < 0 {
		effective = 0
	}
	return effective
}

// signalPriority returns a numeric priority for sorting signals.
// Lower number = higher priority: REDIRECT=1, FOCUS=2, FEEDBACK=3, other=5.
func signalPriority(typeStr string) int {
	switch typeStr {
	case "REDIRECT":
		return 1
	case "FOCUS":
		return 2
	case "FEEDBACK":
		return 3
	default:
		return 5
	}
}

// truncate shortens a string to maxLen runes, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return string(runes[:maxLen-3]) + "..."
	}
	return string(runes[:maxLen])
}

// wordCount returns the number of words in a string.
func wordCount(s string) int {
	return len(strings.Fields(s))
}

// trimSection removes a section delimited by sectionHeader from the prompt string.
// It keeps the "--- END CONTEXT CAPSULE ---" line intact.
func trimSection(prompt, sectionHeader string) string {
	idx := strings.Index(prompt, sectionHeader)
	if idx == -1 {
		return prompt
	}

	// Find the end of this section (next section or END marker)
	endMarker := "--- END CONTEXT CAPSULE ---"
	afterHeader := prompt[idx+len(sectionHeader):]

	// Look for the next section marker (a line starting with a section header or the end marker)
	lines := strings.Split(afterHeader, "\n")
	endIdx := len(lines)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == endMarker || (trimmed != "" && trimmed != sectionHeader && !strings.HasPrefix(trimmed, "-")) {
			// This is either the end marker or the start of the next section
			if trimmed == endMarker {
				// Include the end marker, stop before it
				endIdx = i
			}
			break
		}
	}

	// Remove from sectionHeader start to endIdx lines after it
	before := prompt[:idx]
	after := strings.Join(lines[endIdx:], "\n")
	return before + after
}

// extractRecentDecisions returns the last N decisions (reversed) from the decisions slice.
func extractRecentDecisions(decisions []colony.Decision, n int) []interface{} {
	total := len(decisions)
	if total == 0 {
		return []interface{}{}
	}
	if n > total {
		n = total
	}
	result := make([]interface{}, n)
	for i := 0; i < n; i++ {
		dec := decisions[total-1-i]
		result[i] = map[string]interface{}{
			"id":        dec.ID,
			"phase":     dec.Phase,
			"claim":     dec.Claim,
			"rationale": dec.Rationale,
			"timestamp": dec.Timestamp,
		}
	}
	return result
}

// extractRecentEvents returns the last N events as wrapped objects.
func extractRecentEvents(events []string, n int) []interface{} {
	total := len(events)
	if total == 0 {
		return []interface{}{}
	}
	if n > total {
		n = total
	}
	result := make([]interface{}, n)
	for i := 0; i < n; i++ {
		result[i] = map[string]string{"event": events[total-1-i]}
	}
	return result
}

// computeNextAction determines the recommended next action based on colony state.
func computeNextAction(stateStr string, currentPhase, totalPhases int) string {
	switch {
	case stateStr == "IDLE":
		return `aether init "goal"`
	case totalPhases == 0:
		return "aether plan"
	case stateStr == "EXECUTING":
		return "aether continue"
	case stateStr == "COMPLETED":
		return "aether entomb"
	case stateStr == "READY" && currentPhase == 0:
		return "aether build 1"
	case stateStr == "READY" && currentPhase < totalPhases:
		return fmt.Sprintf("aether build %d", currentPhase+1)
	case stateStr == "READY" && currentPhase >= totalPhases:
		return "aether seal"
	case stateStr == "BUILT":
		return "aether continue"
	default:
		return "aether status"
	}
}

// lookupPhaseName finds the phase name by phase ID.
func lookupPhaseName(state colony.ColonyState, phaseID int) string {
	for _, p := range state.Plan.Phases {
		if p.ID == phaseID {
			return p.Name
		}
	}
	return "(unnamed)"
}

// extractDecisionTexts returns the last N decision claims (reversed), truncated.
func extractDecisionTexts(decisions []colony.Decision, n int) []string {
	total := len(decisions)
	if total == 0 {
		return nil
	}
	if n > total {
		n = total
	}
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = truncateString(decisions[total-1-i].Claim, 160)
	}
	return result
}

// extractRiskTexts loads flags.json and returns descriptions of unresolved blockers/issues.
func extractRiskTexts(maxRisks int) []string {
	var flags colony.FlagsFile
	if err := store.LoadJSON("flags.json", &flags); err != nil {
		// Try alternate name
		if err2 := store.LoadJSON("pending-decisions.json", &flags); err2 != nil {
			return nil
		}
	}

	var risks []string
	for _, f := range flags.Decisions {
		if f.Resolved {
			continue
		}
		if f.Type != "blocker" && f.Type != "issue" {
			continue
		}
		risks = append(risks, truncateString(f.Description, 160))
		if len(risks) >= maxRisks {
			break
		}
	}
	return risks
}

// extractSignalTexts loads pheromones.json, computes effective strengths, sorts, and returns formatted signals.
func extractSignalTexts(maxSignals int) []string {
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}

	now := time.Now()

	// Filter and compute effective strengths
	type scoredSignal struct {
		priority          int
		effectiveStrength float64
		text              string
	}

	var scored []scoredSignal
	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		eff := computeEffectiveStrength(sig, now)
		if eff < 0.1 {
			continue
		}
		text := extractSignalText(sig.Content)
		if text == "" {
			continue
		}
		scored = append(scored, scoredSignal{
			priority:          signalPriority(sig.Type),
			effectiveStrength: eff,
			text:              fmt.Sprintf("%s: %s", sig.Type, text),
		})
	}

	// Sort by priority (ascending), then by effective strength (descending)
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].priority != scored[j].priority {
			return scored[i].priority < scored[j].priority
		}
		return scored[i].effectiveStrength > scored[j].effectiveStrength
	})

	// Take top N
	if len(scored) > maxSignals {
		scored = scored[:maxSignals]
	}

	result := make([]string, len(scored))
	for i, s := range scored {
		result[i] = s.text
	}
	return result
}

// extractRollingSummary reads rolling-summary.log and extracts last N entries.
func extractRollingSummary(n int) []string {
	data, err := store.ReadFile("rolling-summary.log")
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	total := len(lines)
	if total == 0 {
		return nil
	}

	start := 0
	if total > n {
		start = total - n
	}

	var result []string
	for i := start; i < total; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "|", 5)
		if len(fields) >= 4 {
			entry := fmt.Sprintf("%s: %s", strings.TrimSpace(fields[1]), strings.TrimSpace(fields[3]))
			result = append(result, truncateString(entry, 160))
		}
	}
	return result
}

// ensure compile-time check that os.ReadFile is available
var _ = os.ReadFile

// --- pr-context helper functions ---

// resolveAetherRootPath returns the Aether root directory.
func resolveAetherRootPath() string {
	if store != nil {
		if root := aetherRootFromDataPath(store.BasePath()); root != "" {
			return root
		}
	}
	if dataDir := strings.TrimSpace(os.Getenv("COLONY_DATA_DIR")); dataDir != "" {
		if root := aetherRootFromDataPath(dataDir); root != "" {
			return root
		}
	}
	if root := strings.TrimSpace(os.Getenv("AETHER_ROOT")); root != "" {
		return root
	}
	ctx, cancel := context.WithTimeout(context.Background(), GeneralTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	dir, _ := os.Getwd()
	return dir
}

func aetherRootFromDataPath(dataPath string) string {
	dataPath = strings.TrimSpace(dataPath)
	if dataPath == "" {
		return ""
	}
	clean := filepath.Clean(dataPath)
	if filepath.Base(clean) != "data" {
		return ""
	}
	parent := filepath.Base(filepath.Dir(clean))
	if parent != ".aether" {
		return ""
	}
	return filepath.Dir(filepath.Dir(clean))
}

// readQUEENMd reads a QUEEN.md file and parses key-value pairs from Wisdom and Patterns sections.
func readQUEENMd(filePath string) map[string]string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	result := map[string]string{}
	lines := strings.Split(string(data), "\n")
	inWisdomSection := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track which section we're in
		if strings.HasPrefix(trimmed, "## ") {
			sectionName := strings.TrimPrefix(trimmed, "## ")
			inWisdomSection = sectionName == "Wisdom" || sectionName == "Patterns" ||
				strings.HasPrefix(sectionName, "Wisdom") || strings.HasPrefix(sectionName, "Patterns")
			continue
		}

		if !inWisdomSection {
			continue
		}

		// Parse "key: value" lines
		if idx := strings.Index(trimmed, ": "); idx > 0 {
			key := strings.TrimSpace(trimmed[:idx])
			val := strings.TrimSpace(trimmed[idx+2:])
			if key != "" && val != "" {
				result[key] = val
			}
		}
	}

	return result
}

// readUserPreferences extracts lines starting with "- " from the "## User Preferences" section.
func readUserPreferences(filePath string) []string {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var prefs []string
	lines := strings.Split(string(data), "\n")
	inPrefs := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inPrefs = strings.Contains(trimmed, "User Preferences")
			continue
		}
		if inPrefs && strings.HasPrefix(trimmed, "- ") {
			pref := strings.TrimPrefix(trimmed, "- ")
			pref = strings.TrimSpace(pref)
			if pref != "" {
				prefs = append(prefs, truncateString(pref, 500))
			}
		}
	}
	return prefs
}

// readHiveWisdom reads hive wisdom entries, falling back to eternal memory.
// Returns up to limit entries as simple text strings for prompt assembly.
func readHiveWisdom(hubDir string, limit int, fallbacks *[]string) []string {
	// Try hive first
	wisdomPath := filepath.Join(hubDir, "hive", "wisdom.json")
	if data, err := os.ReadFile(wisdomPath); err == nil {
		var wf struct {
			Entries []struct {
				Text       string  `json:"text"`
				Confidence float64 `json:"confidence"`
			} `json:"entries"`
		}
		if err := json.Unmarshal(data, &wf); err == nil && len(wf.Entries) > 0 {
			var results []string
			count := limit
			if len(wf.Entries) < count {
				count = len(wf.Entries)
			}
			for i := 0; i < count; i++ {
				results = append(results, truncateString(wf.Entries[i].Text, 200))
			}
			return results
		}
	}

	// Fallback to eternal memory
	eternalPath := filepath.Join(hubDir, "eternal", "memory.json")
	if data, err := os.ReadFile(eternalPath); err == nil {
		var entries []struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(data, &entries) == nil && len(entries) > 0 {
			var results []string
			count := limit
			if len(entries) < count {
				count = len(entries)
			}
			for i := 0; i < count; i++ {
				results = append(results, truncateString(entries[i].Text, 200))
			}
			return results
		}
	}

	*fallbacks = append(*fallbacks, "hive_wisdom: no hive or eternal data")
	return nil
}

// extractBlockerTexts loads flags.json and returns unresolved blocker/issue descriptions.
func extractBlockerTexts() []string {
	var flags colony.FlagsFile
	if err := store.LoadJSON("flags.json", &flags); err != nil {
		if err2 := store.LoadJSON("pending-decisions.json", &flags); err2 != nil {
			return nil
		}
	}

	var blockers []string
	for _, f := range flags.Decisions {
		if f.Resolved {
			continue
		}
		if f.Type == "blocker" || f.Type == "issue" || strings.Contains(f.Description, "CRITICAL") {
			blockers = append(blockers, truncateString(f.Description, 160))
		}
	}
	return blockers
}

// buildPRSectionHeader returns a formatted section header for pr-context prompt sections.
func buildPRSectionHeader(title string) string {
	return fmt.Sprintf("\n--- %s ---\n", title)
}

// removePRSection removes a section between its header and the next "---" line.
func removePRSection(prompt, header string) string {
	idx := strings.Index(prompt, header)
	if idx == -1 {
		return prompt
	}

	// Find the end: next "---" line after the header start
	afterHeader := prompt[idx:]
	lines := strings.Split(afterHeader, "\n")

	// The first line is the header itself; find where the next "---" section begins
	endIdx := len(lines)
	for i := 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "---") && i > 0 {
			endIdx = i
			break
		}
	}

	// Remove from idx to the end of this section
	before := prompt[:idx]
	remaining := strings.Join(lines[endIdx:], "\n")
	return before + remaining
}

// detectGitBranch returns the current git branch name.
func detectGitBranch() string {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	for _, args := range [][]string{
		{"rev-parse", "--abbrev-ref", "HEAD"},
		{"symbolic-ref", "--short", "HEAD"},
	} {
		cmd := exec.CommandContext(ctx, "git", args...)
		if out, err := cmd.Output(); err == nil {
			branch := strings.TrimSpace(string(out))
			if branch != "" && branch != "HEAD" {
				return branch
			}
		}
	}
	return "unknown"
}

// --- context-update ---

// contextUpdateCmd updates the rolling summary stored in rolling-summary.log.
// When a positional argument is provided, it dispatches to a sub-action handler
// (init, build-start, build-progress, build-complete, worker-spawn, worker-complete).
// When no positional argument is provided, it falls back to --summary/--append behavior.
var contextUpdateCmd = &cobra.Command{
	Use:          "context-update [action] [args...]",
	Short:        "Update colony context summary",
	Args:         cobra.ArbitraryArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Check for --section flag first (takes priority over positional args)
		section, _ := cmd.Flags().GetString("section")
		if section != "" {
			key, _ := cmd.Flags().GetString("key")
			secContent, _ := cmd.Flags().GetString("content")
			return runContextSectionUpdate(section, key, secContent, args)
		}

		// If positional arg provided, dispatch to sub-action
		if len(args) > 0 {
			return runContextSubAction(args)
		}

		// Fallback: existing --summary/--append behavior
		summary, _ := cmd.Flags().GetString("summary")
		appendMode, _ := cmd.Flags().GetBool("append")

		if summary == "" {
			outputErrorMessage("flag --summary is required (or provide a sub-action or --section)")
			return nil
		}

		ts := time.Now().UTC().Format(time.RFC3339)
		entry := fmt.Sprintf("%s|context|user|%s", ts, summary)

		if appendMode {
			// Read existing and append
			existing, _ := store.ReadFile("rolling-summary.log")
			var data []byte
			if len(existing) > 0 {
				data = append(existing, '\n')
			}
			data = append(data, []byte(entry)...)
			if err := store.AtomicWrite("rolling-summary.log", data); err != nil {
				outputErrorMessage(fmt.Sprintf("failed to write summary: %v", err))
				return nil
			}
		} else {
			// Replace the summary
			if err := store.AtomicWrite("rolling-summary.log", []byte(entry)); err != nil {
				outputErrorMessage(fmt.Sprintf("failed to write summary: %v", err))
				return nil
			}
		}

		outputOK(map[string]interface{}{
			"updated":        true,
			"summary_length": len(summary),
		})
		return nil
	},
}
