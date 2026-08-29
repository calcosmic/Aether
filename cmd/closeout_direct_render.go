package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// closeoutDirectVisual is the D-12 structural fix: it renders the chat-path
// closing screen using the SAME renderer the direct command-line path uses,
// fed by the finalizer's own saved result rather than a second, narrower
// hand-derivation. It returns ("", false) when it cannot (or does not yet)
// handle the given workflow -- the caller (renderCeremonyCloseout) must fall
// through to the existing generic renderCeremonyCloseoutVisual unchanged.
//
// Plan 198-01 wired "continue"; this plan (198-04) adds "plan" and "seal",
// closing roadmap criterion 1 (all three named workflows covered by one
// parity test).
func closeoutDirectVisual(workflow string, result map[string]interface{}, state colony.ColonyState) (string, bool) {
	switch workflow {
	case "continue":
		return closeoutContinueDirectVisual(result, state)
	case "plan":
		return closeoutPlanDirectVisual(result, state)
	case "seal":
		return closeoutSealDirectVisual(result, state)
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

// closeoutPlanDirectVisual renders the plan closing screen from
// closeoutDirectVisual's ceremony result map, mirroring
// closeoutContinueDirectVisual's completion_raw / double-envelope-unwrap
// shape exactly -- a plan-finalize (or direct `aether plan`) result has
// neither a manifest key nor a dispatches/results/workers key either, so
// closeoutCompletionDetails' shared unwrap heuristic does not recognize it
// as an envelope, same as continue's already-documented gap (198-01).
func closeoutPlanDirectVisual(result map[string]interface{}, state colony.ColonyState) (string, bool) {
	raw := mapValue(result["completion_raw"])
	if len(raw) == 0 {
		raw = result
	}
	if _, hasGoal := raw["goal"]; !hasGoal {
		if nested := mapValue(raw["result"]); len(nested) > 0 {
			raw = nested
		}
	}

	normalized, ok := closeoutPlanRenderInputs(raw)
	if !ok {
		return "", false
	}
	return renderPlanVisual(normalized), true
}

// closeoutPlanRenderInputs reshapes a JSON-round-tripped plan/plan-finalize
// completion map into the exact dynamic shape renderPlanVisual's own
// type-switches expect for the fields that differ between the two paths --
// a completion file, once json.Unmarshal'd, only ever produces primitives,
// map[string]interface{}, and []interface{}, but both runCodexPlanWithOptions
// and runCodexPlanFinalize always store "confidence" as a codexPlanConfidence
// struct (cmd/codex_plan.go:786; cmd/codex_plan_finalize.go:455,872) and
// "plan_revision" as a colony.PlanRevision struct (cmd/codex_plan.go:783)
// when one is present. renderPlanVisual's own struct/map type-switches for
// planning_loop and plan_revision already handle both shapes identically
// (dual-type precedent, 198-PATTERNS.md), but its confidence branch
// (codex_visuals.go:1489) recognizes only map[string]interface{} -- normalize
// confidence back to the struct so a completion-file value renders exactly
// like the in-process value renders (including "renders nothing" if a later
// change to that branch has not landed yet), rather than the completion-file
// path showing content the direct in-process path does not.
//
// ok is false when raw carries no "goal" -- every real plan/plan-finalize
// result (completed, existing, mid-loop, or manifest-only) always sets it,
// so its absence means this is not a resolvable plan completion and the
// caller must fall back to the generic renderer rather than render a
// half-screen.
func closeoutPlanRenderInputs(raw map[string]interface{}) (map[string]interface{}, bool) {
	if strings.TrimSpace(stringValue(raw["goal"])) == "" {
		return nil, false
	}

	normalized := make(map[string]interface{}, len(raw))
	for key, value := range raw {
		normalized[key] = value
	}
	if confidence, ok := planConfidenceFromValue(raw["confidence"]); ok {
		normalized["confidence"] = confidence
	}
	if revision, ok := planRevisionFromValue(raw["plan_revision"]); ok {
		normalized["plan_revision"] = revision
	}
	return normalized, true
}

// planConfidenceFromValue converts a round-tripped "confidence" value (a
// map[string]interface{} once a completion file has been through
// json.Unmarshal) back into the codexPlanConfidence struct both plan
// construction paths always populate it with, so it carries the identical
// dynamic type -- and therefore renders identically, whatever renderPlanVisual
// currently does with it -- as the in-process value.
func planConfidenceFromValue(raw interface{}) (codexPlanConfidence, bool) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return codexPlanConfidence{}, false
	}
	data, err := json.Marshal(m)
	if err != nil {
		return codexPlanConfidence{}, false
	}
	var confidence codexPlanConfidence
	if err := json.Unmarshal(data, &confidence); err != nil {
		return codexPlanConfidence{}, false
	}
	return confidence, true
}

// planRevisionFromValue is planConfidenceFromValue's sibling for
// "plan_revision": renderPlanVisual's struct and map branches for this field
// render genuinely different text (the map branch omits the revision's
// Reason entirely), so a completion-file map form must be converted back to
// the colony.PlanRevision struct construction always stores here, matching
// the in-process value's own rendering exactly rather than a thinner
// map-shaped rendering of the same data.
func planRevisionFromValue(raw interface{}) (colony.PlanRevision, bool) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return colony.PlanRevision{}, false
	}
	data, err := json.Marshal(m)
	if err != nil {
		return colony.PlanRevision{}, false
	}
	var revision colony.PlanRevision
	if err := json.Unmarshal(data, &revision); err != nil {
		return colony.PlanRevision{}, false
	}
	return revision, true
}

// closeoutSealDirectVisual renders the seal closing screen from
// closeoutDirectVisual's ceremony result map. Mirrors
// closeoutContinueDirectVisual/closeoutPlanDirectVisual's
// completion_raw / double-envelope-unwrap shape: a seal(-finalize) result has
// no manifest key and no dispatches/results/workers key either, so the
// shared unwrap heuristic in closeoutCompletionDetails does not recognize it
// (same documented gap as continue, 198-01).
func closeoutSealDirectVisual(result map[string]interface{}, state colony.ColonyState) (string, bool) {
	raw := mapValue(result["completion_raw"])
	if len(raw) == 0 {
		raw = result
	}
	if _, hasSealed := raw["sealed"]; !hasSealed {
		if nested := mapValue(raw["result"]); len(nested) > 0 {
			raw = nested
		}
	}

	summaryPath, ok := closeoutSealRenderInputs(raw, state)
	if !ok {
		return "", false
	}
	return renderSealVisual(raw, state, summaryPath), true
}

// closeoutSealRenderInputs resolves renderSealVisual's summaryPath argument
// from the completion map's own "summary" key, falling back to the standard
// CROWNED-ANTHILL.md location completeSealRuntime always writes
// (cmd/codex_workflow_cmds.go:781) when the key is absent.
//
// ok is false when raw's "sealed" flag is not true -- covering both a
// genuinely different payload shape AND the D-04..D-07 confirmation gate's
// own "not proceeding" result (runSealConfirmationGate,
// cmd/seal_confirmation.go:323-329: "sealed": false,
// "awaiting_owner_confirmation": true, ...), which carries no phase count, no
// override fields, and nothing renderSealVisual's "Colony sealed at Crowned
// Anthill" screen could honestly show -- rendering it anyway would be a
// half-screen actively claiming a seal that has not happened. The direct
// path never renders this pending result as a visual either: both
// sealCmd's RunE and runSealFinalize hand it to outputOK, which always
// writes the raw JSON envelope regardless of output mode (cmd/helpers.go),
// so "not handled" here is genuine parity with the direct path, not a new
// omission.
func closeoutSealRenderInputs(raw map[string]interface{}, state colony.ColonyState) (string, bool) {
	if !boolValue(raw["sealed"]) {
		return "", false
	}
	summaryPath := strings.TrimSpace(stringValue(raw["summary"]))
	if summaryPath == "" {
		summaryPath = defaultCrownedAnthillSummaryPath()
	}
	return summaryPath, true
}

// defaultCrownedAnthillSummaryPath mirrors completeSealRuntime's own
// summaryPath construction (cmd/codex_workflow_cmds.go:781:
// filepath.Join(aetherDir, "CROWNED-ANTHILL.md")) for the case where a
// completion file's own "summary" key is absent.
func defaultCrownedAnthillSummaryPath() string {
	if store == nil {
		return "CROWNED-ANTHILL.md"
	}
	return filepath.Join(filepath.Dir(store.BasePath()), "CROWNED-ANTHILL.md")
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
