package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// writePhaseOutcomeDocument is the ONLY producer of build/phase-N/outcome.md
// (WIRE-07, 198.2 D-11). It re-derives nothing: its entire content comes from
// closeoutContinueDirectVisual -- the same D-12 renderer (198 D-12), fed the
// same finalizer result map, that produces the closing screen the owner reads
// on every check lane. A second, hand-built summary here would recreate
// exactly the drift 198 D-12 removed. The doc comment is enforced by
// TestOutcomeFileMatchesTheClosingScreen: any prose this function builds
// itself, rather than reads from the renderer's own output, breaks that
// test's string-equality assertion.
//
// phaseID is supplied explicitly rather than read only from result["continued_phase"]
// because the two call-site shapes disagree on the key: the finalize lane's
// result map already carries "continued_phase" (set by advanceExternalContinue /
// finalizeBlockedExternalContinue), but the direct lane's result map carries
// only "current_phase" -- a different phase in the blocked case (the phase
// that was being continued, not necessarily colony.CurrentPhase after the
// call). writePhaseOutcomeDocument sets "continued_phase" to phaseID on a
// copy of result before handing it to the renderer, so both lanes resolve
// identically without mutating the caller's own result map (the caller may
// still be reading it after this call returns).
//
// Returns a non-nil error, and writes nothing, when closeoutContinueDirectVisual
// cannot resolve a closing screen from the given phaseID/result/state (e.g.
// phaseID does not name a phase in state.Plan.Phases) -- never a half-written
// or empty file. Called twice for the same phaseID, the second call's
// AtomicWrite replaces the first -- one outcome per phase, never appended.
func writePhaseOutcomeDocument(phaseID int, result map[string]interface{}, state colony.ColonyState) error {
	if store == nil {
		return fmt.Errorf("phase outcome: no active store to write outcome for phase %d", phaseID)
	}

	fed := make(map[string]interface{}, len(result)+1)
	for k, v := range result {
		fed[k] = v
	}
	fed["continued_phase"] = phaseID

	visual, ok := closeoutContinueDirectVisual(fed, state)
	if !ok {
		return fmt.Errorf("phase outcome: could not resolve a closing screen for phase %d from the finalizer result", phaseID)
	}

	plain := stripPhaseOutcomeANSI(visual)
	path := continuePlanArtifactsPath(phaseID, "outcome.md")
	return store.AtomicWrite(path, []byte(plain))
}

// stripPhaseOutcomeANSI removes terminal colour escape sequences from a
// rendered closing screen before it is persisted to outcome.md -- the
// document is read back into a later phase's worker prompt (plan 07), where
// raw escape codes would be noise, not signal. This reimplements
// pkg/codex/platform_dispatch.go's unexported stripANSIEscapeCodes (same
// byte-for-byte algorithm): that function lives in a different package and
// is unexported there, so it cannot be imported from cmd, and
// cmd/golden_workflow_test.go's stripANSI is a test-only helper unavailable
// to production code.
func stripPhaseOutcomeANSI(value string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}
