package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type skipPhaseReport struct {
	Phase          int      `json:"phase"`
	PhaseName      string   `json:"phase_name"`
	Reason         string   `json:"reason"`
	SkippedAt      string   `json:"skipped_at"`
	PreviousState  string   `json:"previous_state"`
	NextState      string   `json:"next_state"`
	CurrentPhase   int      `json:"current_phase"`
	TasksClosed    []string `json:"tasks_closed,omitempty"`
	CheckpointPath string   `json:"checkpoint_path"`
	Next           string   `json:"next"`
}

var skipPhaseCmd = &cobra.Command{
	Use:   "skip-phase <phase>",
	Short: "Emergency-skip an active or ready phase after an unrecoverable build",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := parsePositivePhaseArg(args[0])
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		force, _ := cmd.Flags().GetBool("force")
		reason, _ := cmd.Flags().GetString("reason")
		result, updated, skipped, nextPhase, err := runSkipPhase(phaseNum, force, reason)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputWorkflow(result, renderSkipPhaseVisual(updated, skipped, nextPhase, result))
		return nil
	},
}

func runSkipPhase(phaseNum int, force bool, reason string) (map[string]interface{}, colony.ColonyState, colony.Phase, *colony.Phase, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
	}
	if !force {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("skip-phase is destructive; rerun with --force after confirming the phase should be abandoned")
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "manual force skip"
	}

	previous, err := loadActiveColonyState()
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(previous.Plan.Phases) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(previous.Plan.Phases) {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(previous.Plan.Phases))
	}
	if err := validateSkipPhaseTarget(previous, phaseNum); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-skip-phase-%d.json", phaseNum)))
	if err := store.SaveJSON(checkpointRel, previous); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}

	now := time.Now().UTC()
	var updated colony.ColonyState
	var skipped colony.Phase
	var nextPhase *colony.Phase
	var nextCommand string
	var tasksClosed []string
	previousState := string(previous.State)

	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updated, func() error {
		if err := validateSkipPhaseTarget(updated, phaseNum); err != nil {
			return err
		}

		idx := phaseNum - 1
		skipped = updated.Plan.Phases[idx]
		for taskIdx := range updated.Plan.Phases[idx].Tasks {
			if updated.Plan.Phases[idx].Tasks[taskIdx].Status == colony.TaskCompleted {
				continue
			}
			tasksClosed = append(tasksClosed, buildTaskID(updated.Plan.Phases[idx].Tasks[taskIdx], taskIdx))
			updated.Plan.Phases[idx].Tasks[taskIdx].Status = colony.TaskCompleted
		}
		updated.Plan.Phases[idx].Status = colony.PhaseCompleted
		updated.BuildStartedAt = nil

		updated.Events = append(trimmedEvents(updated.Events),
			fmt.Sprintf("%s|phase_skipped|skip-phase|Force skipped phase %d: %s", now.Format(time.RFC3339), phaseNum, reason),
		)

		if idx == len(updated.Plan.Phases)-1 {
			updated.State = colony.StateCOMPLETED
			updated.CurrentPhase = phaseNum
			nextCommand = "aether seal"
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_completed|skip-phase|Skipped final phase %d", now.Format(time.RFC3339), phaseNum),
			)
		} else {
			nextIdx := idx + 1
			if updated.Plan.Phases[nextIdx].Status == "" || updated.Plan.Phases[nextIdx].Status == colony.PhasePending {
				updated.Plan.Phases[nextIdx].Status = colony.PhaseReady
			}
			updated.CurrentPhase = updated.Plan.Phases[nextIdx].ID
			updated.State = colony.StateREADY
			nextCopy := updated.Plan.Phases[nextIdx]
			nextPhase = &nextCopy
			nextCommand = fmt.Sprintf("aether build %d", updated.CurrentPhase)
			updated.Events = append(updated.Events,
				fmt.Sprintf("%s|phase_advanced|skip-phase|Skipped phase %d, ready for phase %d", now.Format(time.RFC3339), phaseNum, updated.CurrentPhase),
			)
		}
		skipped = updated.Plan.Phases[idx]
		return nil
	}); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	reportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "skip.json"))
	report := skipPhaseReport{
		Phase:          phaseNum,
		PhaseName:      skipped.Name,
		Reason:         reason,
		SkippedAt:      now.Format(time.RFC3339),
		PreviousState:  previousState,
		NextState:      string(updated.State),
		CurrentPhase:   updated.CurrentPhase,
		TasksClosed:    uniqueSortedStrings(tasksClosed),
		CheckpointPath: displayDataPath(checkpointRel),
		Next:           nextCommand,
	}
	if err := store.SaveJSON(reportRel, report); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write skip report: %w", err)
	}

	summary := fmt.Sprintf("Force skipped phase %d: %s", phaseNum, reason)
	if _, err := syncColonyArtifacts(updated, colonyArtifactOptions{
		CommandName:   "skip-phase",
		SuggestedNext: nextCommand,
		Summary:       summary,
		SafeToClear:   "YES - Phase was force-skipped and state was advanced",
		HandoffTitle:  "Phase Force-Skipped",
		WriteHandoff:  true,
	}); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to save recovery artifacts: %w", err)
	}

	result := map[string]interface{}{
		"skipped":        true,
		"phase":          phaseNum,
		"phase_name":     skipped.Name,
		"reason":         reason,
		"state":          updated.State,
		"current_phase":  updated.CurrentPhase,
		"next":           nextCommand,
		"checkpoint":     displayDataPath(checkpointRel),
		"skip_report":    displayDataPath(reportRel),
		"tasks_closed":   uniqueSortedStrings(tasksClosed),
		"previous_state": previousState,
	}
	if nextPhase != nil {
		result["next_phase"] = nextPhase.ID
		result["next_phase_name"] = nextPhase.Name
	}
	return result, updated, skipped, nextPhase, nil
}

func validateSkipPhaseTarget(state colony.ColonyState, phaseNum int) error {
	if len(state.Plan.Phases) == 0 {
		return fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	phase := state.Plan.Phases[phaseNum-1]
	if phase.Status == colony.PhaseCompleted {
		return fmt.Errorf("phase %d is already completed", phaseNum)
	}
	for i := 0; i < phaseNum-1; i++ {
		if state.Plan.Phases[i].Status != colony.PhaseCompleted {
			return fmt.Errorf("phase %d is not complete yet; skip phases in order", state.Plan.Phases[i].ID)
		}
	}

	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.CurrentPhase != phaseNum {
			return fmt.Errorf("phase %d is active; skip that phase before phase %d", state.CurrentPhase, phaseNum)
		}
	case colony.StateREADY:
		next := recoveryPhase(&state)
		if next == nil || next.ID != phaseNum {
			return fmt.Errorf("phase %d is not the next ready phase", phaseNum)
		}
	default:
		return fmt.Errorf("state %s is not skippable", state.State)
	}
	return nil
}

func renderSkipPhaseVisual(state colony.ColonyState, skipped colony.Phase, nextPhase *colony.Phase, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("skip-phase"), fmt.Sprintf("Skip Phase %d", skipped.ID)))
	b.WriteString(visualDivider)
	b.WriteString("Phase was force-skipped.\n")
	b.WriteString(renderProgressSummary(skipped.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Skipped: ")
	b.WriteString(skipped.Name)
	b.WriteString("\n")
	if reason := strings.TrimSpace(stringValue(result["reason"])); reason != "" {
		b.WriteString("Reason: ")
		b.WriteString(reason)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderArtifactsSection(
		stringValue(result["checkpoint"]),
		stringValue(result["skip_report"]),
	))
	if nextPhase != nil {
		b.WriteString(renderStageMarker("Next Phase"))
		b.WriteString(fmt.Sprintf("Next phase ready: %d - %s\n", nextPhase.ID, nextPhase.Name))
	}
	next := strings.TrimSpace(stringValue(result["next"]))
	if next == "" {
		next = nextCommandFromState(state)
	}
	b.WriteString(renderNextUp(`Run ` + "`" + next + "`" + ` when ready.`))
	return b.String()
}
