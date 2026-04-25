package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var (
	continueAdvanceEventPattern  = regexp.MustCompile(`\|phase_advanced\|(?:continue|skip-phase)\|(?:Completed|Skipped) phase (\d+), ready for phase (\d+)`)
	continueCompleteEventPattern = regexp.MustCompile(`\|phase_completed\|(?:continue|skip-phase)\|(?:Completed|Skipped) final phase (\d+)`)
	legacyPhaseCompletePattern   = regexp.MustCompile(`(?i)\bphase[- ]?(\d+)[-_ ]?(?:complete|completed)\b`)
)

func repairMissingPlanFromArtifacts(state colony.ColonyState) (colony.ColonyState, bool, error) {
	if store == nil || state.Goal == nil || strings.TrimSpace(*state.Goal) == "" || len(state.Plan.Phases) > 0 {
		return state, false, nil
	}

	artifact, generatedAt, err := loadPersistedPlanArtifact(state)
	if err != nil {
		return state, false, nil
	}

	phases := buildWorkerPlanPhases(artifact)
	if len(phases) == 0 {
		return state, false, nil
	}

	repaired := state
	repaired.Plan.Phases = phases
	repaired.Plan.GeneratedAt = &generatedAt
	if artifact.Confidence.Overall > 0 {
		confidence := float64(artifact.Confidence.Overall) / 100.0
		repaired.Plan.Confidence = &confidence
	}

	completed, current := inferRecoveredPhaseProgress(repaired)
	applyRecoveredPlanProgress(&repaired, completed, current)
	repaired.Events = append(trimmedEvents(repaired.Events),
		fmt.Sprintf("%s|plan_recovered|state|Recovered %d phases from planning artifact after COLONY_STATE.json lost its saved plan", time.Now().UTC().Format(time.RFC3339), len(repaired.Plan.Phases)),
	)

	if err := store.SaveJSON("COLONY_STATE.json", repaired); err != nil {
		return state, false, fmt.Errorf("failed to persist repaired colony state: %w", err)
	}
	return repaired, true, nil
}

func loadPersistedPlanArtifact(state colony.ColonyState) (codexWorkerPlanArtifact, time.Time, error) {
	path := filepath.Join(store.BasePath(), "planning", "phase-plan.json")
	info, err := os.Stat(path)
	if err != nil {
		return codexWorkerPlanArtifact{}, time.Time{}, err
	}

	generatedAt := info.ModTime().UTC()
	if state.InitializedAt != nil && generatedAt.Before(state.InitializedAt.UTC()) {
		return codexWorkerPlanArtifact{}, time.Time{}, fmt.Errorf("planning artifact predates colony initialization")
	}

	var artifact codexWorkerPlanArtifact
	if err := store.LoadJSON(filepath.ToSlash(filepath.Join("planning", "phase-plan.json")), &artifact); err != nil {
		return codexWorkerPlanArtifact{}, time.Time{}, err
	}
	if len(artifact.Phases) == 0 {
		return codexWorkerPlanArtifact{}, time.Time{}, fmt.Errorf("planning artifact contained no phases")
	}
	return artifact, generatedAt, nil
}

func inferRecoveredPhaseProgress(state colony.ColonyState) (int, int) {
	total := len(state.Plan.Phases)
	if total == 0 {
		return 0, 0
	}

	completed := highestCompletedPhaseFromEvents(state.Events)
	reportCompleted := highestCompletedPhaseFromContinueReports(state)
	if reportCompleted > completed {
		completed = reportCompleted
	}
	if completed > total {
		completed = total
	}

	current := state.CurrentPhase
	switch state.State {
	case colony.StateCOMPLETED:
		completed = total
		current = total
	case colony.StateREADY:
		if current < 1 {
			current = completed + 1
		}
		if current <= completed && completed < total {
			current = completed + 1
		}
		if current > completed+1 && completed == 0 {
			completed = current - 1
		}
	case colony.StateEXECUTING, colony.StateBUILT:
		if current < 1 {
			current = completed + 1
		}
		if current <= completed && completed < total {
			current = completed + 1
		}
	default:
		if current < 1 {
			current = completed + 1
		}
	}

	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	if completed >= current && state.State != colony.StateCOMPLETED {
		completed = current - 1
	}
	if completed < 0 {
		completed = 0
	}
	return completed, current
}

func highestCompletedPhaseFromEvents(events []string) int {
	completed := 0
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}

		if matches := continueAdvanceEventPattern.FindStringSubmatch(event); len(matches) == 3 {
			if phase := atoiOrZero(matches[1]); phase > completed {
				completed = phase
			}
			continue
		}

		if matches := continueCompleteEventPattern.FindStringSubmatch(event); len(matches) == 2 {
			if phase := atoiOrZero(matches[1]); phase > completed {
				completed = phase
			}
			continue
		}

		if matches := legacyPhaseCompletePattern.FindStringSubmatch(event); len(matches) == 2 {
			if phase := atoiOrZero(matches[1]); phase > completed {
				completed = phase
			}
		}
	}
	return completed
}

func highestCompletedPhaseFromContinueReports(state colony.ColonyState) int {
	if store == nil {
		return 0
	}

	matches, err := filepath.Glob(filepath.Join(store.BasePath(), "build", "phase-*", "continue.json"))
	if err != nil {
		return 0
	}

	completed := 0
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if state.InitializedAt != nil && info.ModTime().UTC().Before(state.InitializedAt.UTC()) {
			continue
		}

		rel, err := filepath.Rel(store.BasePath(), path)
		if err != nil {
			continue
		}

		var report codexContinueReport
		if err := store.LoadJSON(filepath.ToSlash(rel), &report); err != nil {
			continue
		}
		if !report.Advanced && !report.Completed {
			continue
		}
		if report.Phase > completed {
			completed = report.Phase
		}
	}
	return completed
}

func applyRecoveredPlanProgress(state *colony.ColonyState, completed, current int) {
	total := len(state.Plan.Phases)
	if state == nil || total == 0 {
		return
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}

	for i := range state.Plan.Phases {
		phase := &state.Plan.Phases[i]
		switch {
		case i < completed || state.State == colony.StateCOMPLETED:
			phase.Status = colony.PhaseCompleted
			setRecoveredTaskStatuses(phase, colony.TaskCompleted)
		case i == current-1:
			if state.State == colony.StateEXECUTING || state.State == colony.StateBUILT {
				phase.Status = colony.PhaseInProgress
				setRecoveredActiveTaskStatuses(phase)
			} else {
				phase.Status = colony.PhaseReady
				setRecoveredTaskStatuses(phase, colony.TaskPending)
			}
		default:
			phase.Status = colony.PhasePending
			setRecoveredTaskStatuses(phase, colony.TaskPending)
		}
	}

	if state.State == colony.StateCOMPLETED {
		state.CurrentPhase = total
		return
	}
	state.CurrentPhase = current
}

func setRecoveredTaskStatuses(phase *colony.Phase, status string) {
	for i := range phase.Tasks {
		phase.Tasks[i].Status = status
	}
}

func setRecoveredActiveTaskStatuses(phase *colony.Phase) {
	activeMarked := false
	for i := range phase.Tasks {
		if !activeMarked {
			phase.Tasks[i].Status = colony.TaskInProgress
			activeMarked = true
			continue
		}
		phase.Tasks[i].Status = colony.TaskPending
	}
}

func atoiOrZero(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return value
}
