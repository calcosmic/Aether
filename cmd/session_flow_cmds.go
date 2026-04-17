package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

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

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil {
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

		result := buildResumeDashboardResult()
		result["resumed"] = true
		result["handoff_found"] = handoffText != ""
		result["handoff_path"] = handoffPath

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
	rootCmd.AddCommand(pauseColonyCmd)
	rootCmd.AddCommand(resumeColonyCmd)
}
