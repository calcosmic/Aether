package cmd

import (
	"encoding/json"

	"github.com/calcosmic/Aether/pkg/colony"
)

// closeoutDirectVisual is the D-12 structural fix: it renders the chat-path
// closing screen using the SAME renderer the direct command-line path uses,
// fed by the finalizer's own saved result rather than a second, narrower
// hand-derivation. It returns ("", false) when it cannot (or does not yet)
// handle the given workflow -- the caller (renderCeremonyCloseout) must fall
// through to the existing generic renderCeremonyCloseoutVisual unchanged.
//
// This plan wires only the "continue" branch; "plan" and "seal" follow in
// plan 198-04 once this architecture is proven end to end.
func closeoutDirectVisual(workflow string, result map[string]interface{}, state colony.ColonyState) (string, bool) {
	switch workflow {
	case "continue":
		return closeoutContinueDirectVisual(result, state)
	default:
		return "", false
	}
}

// closeoutContinueDirectVisual renders the continue closing screen from
// closeoutDirectVisual's ceremony result map, using completion_raw (the
// finalizer's whole saved result) when present, falling back to the ceremony
// result map itself otherwise -- the shape the parity test drives directly.
func closeoutContinueDirectVisual(result map[string]interface{}, state colony.ColonyState) (string, bool) {
	raw := mapValue(result["completion_raw"])
	if len(raw) == 0 {
		raw = result
	}
	// closeoutCompletionDetails only unwraps a {"result": {...}} envelope
	// when the nested map already looks like a manifest/worker payload
	// (closeoutManifest/closeoutWorkerMaps) -- a continue-finalize result has
	// neither (it carries worker_flow, not dispatches/results/workers, and no
	// manifest key), so completion_raw can still be the un-unwrapped outer
	// envelope. Unwrap one more level here, locally, rather than widening
	// that shared heuristic for every other workflow.
	if _, hasContinuedPhase := raw["continued_phase"]; !hasContinuedPhase {
		if nested := mapValue(raw["result"]); len(nested) > 0 {
			raw = nested
		}
	}

	phase, nextPhase, housekeeping, final, reviewDepth, ok := closeoutContinueRenderInputs(raw, state)
	if !ok {
		return "", false
	}

	if boolValue(raw["blocked"]) {
		return renderContinueBlockedVisual(state, phase, raw, reviewDepth), true
	}
	return renderContinueVisual(state, phase, housekeeping, final, nextPhase, raw, reviewDepth), true
}

// closeoutContinueRenderInputs reconstructs the non-map arguments
// renderContinueVisual/renderContinueBlockedVisual need from the saved
// completion result plus live colony state: the continued phase by
// continued_phase ID, the next phase by next_phase ID, housekeeping decoded
// from signal_housekeeping, final from completed, and depth via
// reviewDepthFromResult. ok is false when the continued phase cannot be
// resolved -- the caller must then fall back to the generic renderer rather
// than render a half-screen.
func closeoutContinueRenderInputs(raw map[string]interface{}, state colony.ColonyState) (colony.Phase, *colony.Phase, *signalHousekeepingResult, bool, colony.VerificationDepth, bool) {
	reviewDepth := reviewDepthFromResult(raw)
	final := boolValue(raw["completed"])

	phase, found := colonyPhaseByID(state, intValue(raw["continued_phase"]))
	if !found {
		return colony.Phase{}, nil, nil, false, reviewDepth, false
	}

	var nextPhase *colony.Phase
	if next, ok := colonyPhaseByID(state, intValue(raw["next_phase"])); ok {
		nextPhase = &next
	}

	var housekeeping *signalHousekeepingResult
	if decoded, ok := decodeSignalHousekeepingResult(raw["signal_housekeeping"]); ok {
		housekeeping = decoded
	}

	return phase, nextPhase, housekeeping, final, reviewDepth, true
}

// colonyPhaseByID looks up a phase by ID in live state -- the source of
// truth for the phase's name and other fields the completion result does not
// itself carry.
func colonyPhaseByID(state colony.ColonyState, id int) (colony.Phase, bool) {
	if id <= 0 {
		return colony.Phase{}, false
	}
	for _, p := range state.Plan.Phases {
		if p.ID == id {
			return p, true
		}
	}
	return colony.Phase{}, false
}

// decodeSignalHousekeepingResult accepts either the in-process typed struct
// or the JSON-round-tripped map[string]interface{} a completion file
// produces -- the same dual-type precedent renderContinueWorkerFlowValue
// established (198-PATTERNS.md).
func decodeSignalHousekeepingResult(raw interface{}) (*signalHousekeepingResult, bool) {
	switch v := raw.(type) {
	case signalHousekeepingResult:
		return &v, true
	case *signalHousekeepingResult:
		return v, v != nil
	case map[string]interface{}:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var decoded signalHousekeepingResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			return nil, false
		}
		return &decoded, true
	default:
		return nil, false
	}
}
