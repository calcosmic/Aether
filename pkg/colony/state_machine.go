package colony

import "fmt"

// Transition validates and returns the target state if the transition from
// current to target is legal. Returns ErrInvalidTransition if not allowed.
func Transition(current, target State) error {
	allowed, ok := legalTransitions[current]
	if !ok {
		return fmt.Errorf("%w: %s has no outgoing transitions", ErrInvalidTransition, current)
	}
	for _, a := range allowed {
		if a == target {
			return nil
		}
	}
	return fmt.Errorf("%w: %s -> %s is not allowed", ErrInvalidTransition, current, target)
}

// AdvancePhase finds the next pending phase after currentPhase, marks it as
// ready, and returns its ID. Returns an error if no pending phase exists or
// if currentPhase is beyond the phases array.
func AdvancePhase(currentPhase int, phases []Phase) (int, error) {
	if len(phases) == 0 {
		return 0, fmt.Errorf("no phases defined")
	}

	// currentPhase is 1-indexed in the colony system (0 = no phase started)
	// Find the first pending phase after the current one
	for i, phase := range phases {
		if phase.ID > currentPhase && phase.Status == PhasePending {
			phases[i].Status = PhaseReady
			return phase.ID, nil
		}
	}

	return 0, fmt.Errorf("no pending phases after phase %d", currentPhase)
}
