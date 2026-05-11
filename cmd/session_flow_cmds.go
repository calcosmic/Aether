package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
)

const sessionStaleThreshold = 24 * time.Hour
const handoffStateFence = "aether-colony-state"

var handoffPhaseLinePattern = regexp.MustCompile(`(?i)(?:Current:|Phase:)\s*([0-9]+)\s*/\s*([0-9]+)(?:\s*(?:—|-)\s*(.*))?`)
var resumeNoHandoff bool

// sessionFreshnessResult describes how fresh a session is for resume.
type sessionFreshnessResult struct {
	Fresh       bool
	Age         time.Duration
	GitMatch    bool
	GitCheck    bool // whether git HEAD comparison was performed
	SessionID   string
	BaselineSHA string
	CurrentSHA  string
}

// sessionVerifyFresh checks session age and git HEAD to detect stale sessions.
func sessionVerifyFresh(s *storage.Store) sessionFreshnessResult {
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		return sessionFreshnessResult{Fresh: false}
	}

	result := sessionFreshnessResult{
		SessionID:   session.SessionID,
		BaselineSHA: session.BaselineCommit,
	}

	// Check age from started_at
	if startedAt := strings.TrimSpace(session.StartedAt); startedAt != "" {
		if t, err := time.Parse(time.RFC3339, startedAt); err == nil {
			result.Age = time.Since(t)
			result.Fresh = result.Age < sessionStaleThreshold
		} else {
			result.Fresh = false
		}
	} else {
		result.Fresh = false
	}

	// Check git HEAD match
	currentHEAD := getGitHEAD()
	result.CurrentSHA = currentHEAD
	if session.BaselineCommit != "" && currentHEAD != "" {
		result.GitCheck = true
		result.GitMatch = session.BaselineCommit == currentHEAD
		// Git mismatch means repo changed since session — treat as stale
		if !result.GitMatch {
			result.Fresh = false
		}
	}

	return result
}

var pauseColonyCmd = &cobra.Command{
	Use:   "pause-colony",
	Short: "Save colony state and write a handoff for later resumption",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return nil
		}
		pausedAt := time.Now().UTC().Format(time.RFC3339)
		state.Paused = true
		state.PausedAt = &pausedAt
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to mark colony paused: %v", err), nil)
			return nil
		}

		nextAction := "aether resume"
		contextCleared := true
		session, err := syncColonyArtifacts(state, colonyArtifactOptions{
			CommandName:    "pause-colony",
			SuggestedNext:  nextAction,
			Summary:        fmt.Sprintf("Paused at phase %d", state.CurrentPhase),
			SafeToClear:    "YES — Colony paused, safe to clear context",
			HandoffTitle:   "Paused Colony",
			WriteHandoff:   true,
			ContextCleared: &contextCleared,
		})
		if err != nil {
			outputError(2, fmt.Sprintf("failed to save recovery artifacts: %v", err), nil)
			return nil
		}
		goal := session.ColonyGoal
		if goal == "" && state.Goal != nil {
			goal = *state.Goal
		}

		result := map[string]interface{}{
			"paused":        true,
			"goal":          goal,
			"state":         state.State,
			"current_phase": state.CurrentPhase,
			"phase_name":    lookupPhaseName(state, state.CurrentPhase),
			"handoff_path":  handoffDocumentPath(),
			"next":          "aether resume",
		}
		outputWorkflow(result, renderPauseVisual(result))
		return nil
	},
}

// staleSignalInfo represents a stale FOCUS signal for wrapper consumption.
type staleSignalInfo struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Content      string `json:"content"`
	SourcePhase  int    `json:"source_phase"`
	CurrentPhase int    `json:"current_phase"`
}

// detectStaleFocusSignals finds active FOCUS signals whose source_phase
// is less than the current colony phase. Signals without source_phase are
// NOT flagged (backward compatible with signals created before this feature).
// Only FOCUS signals are checked per D-07.
func detectStaleFocusSignals(s *storage.Store, currentPhase int) []staleSignalInfo {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}
	var stale []staleSignalInfo
	for _, sig := range pf.Signals {
		if !sig.Active || sig.Type != "FOCUS" {
			continue
		}
		if sig.SourcePhase == nil {
			continue // Unknown phase -- backward compat, don't flag
		}
		if *sig.SourcePhase < currentPhase {
			stale = append(stale, staleSignalInfo{
				ID:           sig.ID,
				Type:         sig.Type,
				Content:      extractContentText(sig.Content),
				SourcePhase:  *sig.SourcePhase,
				CurrentPhase: currentPhase,
			})
		}
	}
	return stale
}

var resumeColonyCmd = &cobra.Command{
	Use:     "resume-colony",
	Short:   "Restore colony context from handoff and mark the session resumed",
	Aliases: []string{"resume"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		now := time.Now().UTC()
		handoffPath := handoffDocumentPath()
		handoffData, _ := readHandoffDocument()
		handoffText := strings.TrimSpace(string(handoffData))

		// Rotate trace file if it has grown too large
		if rotated, rotateErr := trace.RotateTraceFile(store, 50); rotateErr == nil && rotated {
			fmt.Fprintf(os.Stderr, "warning: rotated trace.jsonl before resume\n")
		}

		// Verify session freshness
		freshness := sessionVerifyFresh(store)

		state, recoveredFromHandoff, stateErr := loadResumeState(handoffText, resumeNoHandoff)
		if stateErr != nil {
			renderRecoveryMenu("resume", stateErr.Error(), nil)
			return nil
		}
		if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
			state.Paused = false
			state.PausedAt = nil
			if state.State == colony.StateEXECUTING && state.BuildStartedAt == nil {
				state.State = colony.StateREADY
			}
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				renderRecoveryMenu("resume", fmt.Sprintf("failed to restore runnable colony state: %v", err), nil)
				return nil
			}
			if state.State != colony.StateEXECUTING || state.BuildStartedAt == nil {
				rotateSpawnTree(store)
			}
			// Clear stale spawn state if session is not fresh
			if !freshness.Fresh {
				state.BuildStartedAt = nil
				// Generate new run_id for resumed stale session
				newRunID := fmt.Sprintf("resume_%d_%s", now.Unix(), randomHex(4))
				state.RunID = &newRunID
				if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
					renderRecoveryMenu("resume", fmt.Sprintf("failed to clear stale spawn state: %v", err), nil)
					return nil
				}
				if tracer != nil && state.RunID != nil {
					_ = tracer.LogIntervention(*state.RunID, "resume.spawn-clear", "resume-colony", map[string]interface{}{
						"reason": "stale_session",
					})
				}
			}
			contextCleared := false
			if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
				CommandName:    "resume-colony",
				SuggestedNext:  nextCommandFromState(state),
				Summary:        "Colony resumed",
				HandoffTitle:   "Resumed Colony",
				WriteHandoff:   false,
				ContextCleared: &contextCleared,
			}); err != nil {
				outputError(2, fmt.Sprintf("failed to save session: %v", err), nil)
				return nil
			}
			if recoveredFromHandoff {
				freshness.Fresh = false
			}

			var session colony.SessionFile
			if err := store.LoadJSON("session.json", &session); err == nil {
				resumedAt := now.Format(time.RFC3339)
				session.ResumedAt = &resumedAt
				if err := store.SaveJSON("session.json", session); err != nil {
					outputError(2, fmt.Sprintf("failed to mark session resumed: %v", err), nil)
					return nil
				}
			}
		}

		// Clean up any orphaned worktrees before resuming
		gcCleaned, gcOrphaned, gcErr := gcOrphanedWorktrees()

		// Detect stale FOCUS pheromones (D-07, D-08)
		var staleSignalsList []staleSignalInfo
		var freshState colony.ColonyState
		if stateLoadErr := store.LoadJSON("COLONY_STATE.json", &freshState); stateLoadErr == nil {
			ns := normalizeLegacyColonyState(freshState)
			staleSignalsList = detectStaleFocusSignals(store, ns.CurrentPhase)
		}

		result := buildResumeDashboardResult()
		result["resumed"] = true
		if recoveredFromHandoff {
			result["state_recovered_from_handoff"] = true
		}
		result["freshness"] = map[string]interface{}{
			"fresh":      freshness.Fresh,
			"age_hours":  fmt.Sprintf("%.1f", freshness.Age.Hours()),
			"git_match":  freshness.GitMatch,
			"git_check":  freshness.GitCheck,
			"session_id": freshness.SessionID,
		}
		result["handoff_found"] = handoffText != ""
		result["handoff_path"] = handoffPath
		if gcErr != nil {
			result["worktree_gc_error"] = gcErr.Error()
		}
		if gcCleaned > 0 || gcOrphaned > 0 {
			result["worktree_gc"] = map[string]interface{}{
				"cleaned":  gcCleaned,
				"orphaned": gcOrphaned,
			}
		}
		if len(staleSignalsList) > 0 {
			staleData := make([]map[string]interface{}, 0, len(staleSignalsList))
			for _, ss := range staleSignalsList {
				staleData = append(staleData, map[string]interface{}{
					"id":            ss.ID,
					"type":          ss.Type,
					"content":       ss.Content,
					"source_phase":  ss.SourcePhase,
					"current_phase": ss.CurrentPhase,
				})
			}
			result["stale_signals"] = staleData
		}

		if handoffText != "" {
			if err := removeHandoffDocument(); err == nil {
				result["handoff_removed"] = true
			} else {
				result["handoff_removed"] = false
				result["handoff_remove_error"] = err.Error()
			}
		} else {
			result["handoff_removed"] = false
		}

		outputWorkflow(result, renderResumeVisual(result, handoffText, true))
		return nil
	},
}

func loadResumeState(handoffText string, noHandoff bool) (colony.ColonyState, bool, error) {
	var rawState colony.ColonyState
	stateLoaded := false
	if err := store.LoadJSON("COLONY_STATE.json", &rawState); err == nil {
		stateLoaded = true
		state := normalizeLegacyColonyState(rawState)
		if resumeStateIsRunnable(state) {
			return state, false, nil
		}
	}

	if noHandoff {
		return colony.ColonyState{}, false, fmt.Errorf("COLONY_STATE.json is not runnable and HANDOFF.md fallback is disabled")
	}

	state, err := restoreStateFromHandoff(handoffText)
	if err != nil {
		return colony.ColonyState{}, false, fmt.Errorf("runtime state is broken and could not be restored from handoff: %w", err)
	}
	if stateLoaded {
		warnResumeHandoffFallback(rawState, state)
		if resumeHandoffGoalMismatch(rawState, state) {
			return colony.ColonyState{}, false, fmt.Errorf("HANDOFF.md appears to belong to a different colony; current COLONY_STATE.json goal %q does not match handoff goal %q", colonyStateGoalText(rawState), colonyStateGoalText(state))
		}
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return colony.ColonyState{}, false, fmt.Errorf("failed to restore COLONY_STATE.json from handoff: %w", err)
	}
	return state, true, nil
}

func warnResumeHandoffFallback(currentState, handoffState colony.ColonyState) {
	fmt.Fprintln(stderr, "warning: COLONY_STATE.json is not runnable; attempting to restore from HANDOFF.md")

	currentGoal := colonyStateGoalText(currentState)
	handoffGoal := colonyStateGoalText(handoffState)
	if currentGoal != "" && handoffGoal != "" && !goalsMatch(currentGoal, handoffGoal) {
		fmt.Fprintf(stderr, "warning: HANDOFF.md goal %q does not match current COLONY_STATE.json goal %q\n", handoffGoal, currentGoal)
	}

	warnIfHandoffOlderThanCurrentState()
}

func resumeHandoffGoalMismatch(currentState, handoffState colony.ColonyState) bool {
	currentGoal := colonyStateGoalText(currentState)
	handoffGoal := colonyStateGoalText(handoffState)
	return currentGoal != "" && handoffGoal != "" && !goalsMatch(currentGoal, handoffGoal)
}

func warnIfHandoffOlderThanCurrentState() {
	if store == nil {
		return
	}
	stateInfo, stateErr := os.Stat(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	handoffInfo, handoffErr := os.Stat(handoffDocumentPath())
	if stateErr != nil || handoffErr != nil {
		return
	}
	if handoffInfo.ModTime().Before(stateInfo.ModTime()) {
		fmt.Fprintf(stderr, "warning: HANDOFF.md is older than COLONY_STATE.json; verify the recovered colony before continuing\n")
	}
}

func colonyStateGoalText(state colony.ColonyState) string {
	if state.Goal == nil {
		return ""
	}
	return strings.TrimSpace(*state.Goal)
}

func goalsMatch(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func resumeStateIsRunnable(state colony.ColonyState) bool {
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return false
	}
	state = normalizeLegacyColonyState(state)
	switch state.State {
	case colony.StateIDLE, colony.StateREADY, colony.StateEXECUTING, colony.StateBUILT, colony.StateCOMPLETED:
	default:
		return false
	}
	if state.CurrentPhase < 0 {
		return false
	}
	if len(state.Plan.Phases) > 0 && state.CurrentPhase > len(state.Plan.Phases) {
		return false
	}
	return true
}

func restoreStateFromHandoff(handoffText string) (colony.ColonyState, error) {
	if strings.TrimSpace(handoffText) == "" {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md is missing or empty")
	}
	if state, ok := restoreStateFromHandoffSnapshot(handoffText); ok {
		return state, nil
	}
	return restoreStateFromLegacyHandoff(handoffText)
}

func restoreStateFromHandoffSnapshot(handoffText string) (colony.ColonyState, bool) {
	block := extractFencedBlock(handoffText, handoffStateFence)
	if block == "" {
		return colony.ColonyState{}, false
	}
	var state colony.ColonyState
	if err := json.Unmarshal([]byte(block), &state); err != nil {
		return colony.ColonyState{}, false
	}
	state = normalizeLegacyColonyState(state)
	if !resumeStateIsRunnable(state) {
		return colony.ColonyState{}, false
	}
	return state, true
}

func restoreStateFromLegacyHandoff(handoffText string) (colony.ColonyState, error) {
	goal := parseHandoffGoal(handoffText)
	phaseID, totalPhases, phaseName := parseHandoffPhase(handoffText)
	stateValue := parseHandoffState(handoffText)
	if strings.TrimSpace(goal) == "" {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md does not contain a recoverable goal")
	}
	if phaseID < 0 {
		return colony.ColonyState{}, fmt.Errorf("HANDOFF.md does not contain a recoverable phase")
	}
	if totalPhases < phaseID {
		totalPhases = phaseID
	}
	if totalPhases < 0 {
		totalPhases = 0
	}
	if stateValue == "" {
		stateValue = colony.StateREADY
	}

	phases := make([]colony.Phase, 0, totalPhases)
	for i := 1; i <= totalPhases; i++ {
		status := colony.PhaseReady
		if i < phaseID {
			status = colony.PhaseCompleted
		}
		if i == phaseID {
			status = colony.PhaseInProgress
		}
		name := fmt.Sprintf("Phase %d", i)
		if i == phaseID && strings.TrimSpace(phaseName) != "" {
			name = strings.TrimSpace(phaseName)
		}
		phases = append(phases, colony.Phase{ID: i, Name: name, Status: status})
	}
	if phaseID > 0 && phaseID <= len(phases) {
		tasks := parseHandoffTasks(handoffText, phaseID)
		phases[phaseID-1].Tasks = tasks
	}

	return colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		Scope:        colony.ScopeProject,
		State:        stateValue,
		CurrentPhase: phaseID,
		Plan:         colony.Plan{Phases: phases},
		Events: []string{
			fmt.Sprintf("%s|state_recovered|resume|Restored COLONY_STATE.json from HANDOFF.md", time.Now().UTC().Format(time.RFC3339)),
		},
	}, nil
}

func extractFencedBlock(text, fenceName string) string {
	lines := strings.Split(text, "\n")
	inBlock := false
	var block []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") && strings.TrimSpace(strings.TrimPrefix(trimmed, "```")) == fenceName {
				inBlock = true
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			return strings.TrimSpace(strings.Join(block, "\n"))
		}
		block = append(block, line)
	}
	return ""
}

func parseHandoffGoal(text string) string {
	if section := extractHandoffMarkdownSection(text, "Goal"); section != "" {
		if value := firstMeaningfulHandoffLine(section); value != "" {
			return value
		}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "Goal:"); ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseHandoffPhase(text string) (int, int, string) {
	for _, line := range strings.Split(text, "\n") {
		matches := handoffPhaseLinePattern.FindStringSubmatch(strings.TrimSpace(line))
		if len(matches) == 0 {
			continue
		}
		phaseID, _ := strconv.Atoi(matches[1])
		total, _ := strconv.Atoi(matches[2])
		phaseName := ""
		if len(matches) > 3 {
			phaseName = strings.TrimSpace(matches[3])
		}
		return phaseID, total, phaseName
	}
	return -1, -1, ""
}

func parseHandoffState(text string) colony.State {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		var value string
		if raw, ok := strings.CutPrefix(line, "State:"); ok {
			value = strings.TrimSpace(raw)
		}
		switch colony.State(strings.ToUpper(value)) {
		case colony.StateIDLE, colony.StateREADY, colony.StateEXECUTING, colony.StateBUILT, colony.StateCOMPLETED:
			return colony.State(strings.ToUpper(value))
		}
	}
	return ""
}

func parseHandoffTasks(text string, phaseID int) []colony.Task {
	section := extractHandoffMarkdownSection(text, "Tasks")
	if section == "" {
		section = extractHandoffMarkdownSection(text, "Open Tasks")
	}
	var tasks []colony.Task
	taskNum := 1
	for _, line := range strings.Split(section, "\n") {
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[ ]"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[>]"))
		value = strings.TrimSpace(strings.TrimPrefix(value, "[x]"))
		if value == "" || strings.EqualFold(value, "none") {
			continue
		}
		id := fmt.Sprintf("%d.%d", phaseID, taskNum)
		tasks = append(tasks, colony.Task{ID: &id, Goal: value, Status: colony.TaskPending})
		taskNum++
	}
	return tasks
}

func extractHandoffMarkdownSection(text, heading string) string {
	lines := strings.Split(text, "\n")
	inSection := false
	var section []string
	target := "## " + strings.TrimSpace(heading)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, target) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if inSection {
			section = append(section, line)
		}
	}
	return strings.TrimSpace(strings.Join(section, "\n"))
}

func firstMeaningfulHandoffLine(section string) string {
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))
		if line == "" || strings.EqualFold(line, "none") {
			continue
		}
		return line
	}
	return ""
}

func buildHandoffDocument(now time.Time, state colony.ColonyState, session colony.SessionFile, nextAction string) string {
	var b strings.Builder
	goal := session.ColonyGoal
	if goal == "" && state.Goal != nil {
		goal = *state.Goal
	}
	totalPhases := len(state.Plan.Phases)
	phaseName := lookupPhaseName(state, state.CurrentPhase)

	b.WriteString("# Colony Handoff\n\n")
	b.WriteString("Paused: ")
	b.WriteString(now.Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("Goal: ")
	b.WriteString(emptyFallback(goal, "No goal set"))
	b.WriteString("\n")
	b.WriteString("State: ")
	b.WriteString(string(state.State))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Phase: %d/%d", state.CurrentPhase, totalPhases))
	if strings.TrimSpace(phaseName) != "" && phaseName != "(unnamed)" {
		b.WriteString(" — ")
		b.WriteString(phaseName)
	}
	b.WriteString("\n")
	b.WriteString("Next: ")
	b.WriteString(nextAction)
	b.WriteString("\n")
	b.WriteString("Suggested resume: aether resume\n\n")

	openTasks := currentOpenTasks(state)
	if len(openTasks) > 0 {
		b.WriteString("## Open Tasks\n")
		for _, task := range openTasks {
			b.WriteString("- ")
			b.WriteString(task)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	if strings.TrimSpace(session.Summary) != "" {
		b.WriteString("## Session Summary\n")
		b.WriteString(session.Summary)
		b.WriteString("\n")
	}
	b.WriteString(renderHandoffStateSnapshot(state))

	return b.String()
}

func currentOpenTasks(state colony.ColonyState) []string {
	if state.CurrentPhase < 1 || state.CurrentPhase > len(state.Plan.Phases) {
		return nil
	}
	phase := state.Plan.Phases[state.CurrentPhase-1]
	var tasks []string
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			continue
		}
		if strings.TrimSpace(task.Goal) == "" {
			continue
		}
		tasks = append(tasks, strings.TrimSpace(task.Goal))
	}
	return tasks
}

func loadOrCreateSessionSummary(now time.Time, state colony.ColonyState) (colony.SessionFile, error) {
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err == nil {
		return session, nil
	}

	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}
	return colony.SessionFile{
		SessionID:        fmt.Sprintf("%d_%s", now.Unix(), randomHex(4)),
		StartedAt:        now.Format(time.RFC3339),
		ColonyGoal:       goal,
		ColonyMode:       state.EffectiveColonyMode(),
		CurrentPhase:     state.CurrentPhase,
		CurrentMilestone: state.Milestone,
		SuggestedNext:    "aether resume",
		ContextCleared:   true,
		BaselineCommit:   getGitHEAD(),
		ResumedAt:        nil,
		ActiveTodos:      currentOpenTasks(state),
		Summary:          "Session paused",
	}, nil
}

func init() {
	resumeColonyCmd.Flags().BoolVar(&resumeNoHandoff, "no-handoff", false, "disable HANDOFF.md fallback when COLONY_STATE.json is not runnable")
	rootCmd.AddCommand(pauseColonyCmd)
	rootCmd.AddCommand(resumeColonyCmd)
}
