package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Task 1 (205-03): a finish-and-archive run must complete even when the
// hand-off note is absent, and a resume must leave a usable note behind so
// the sequence never sees an empty required slot in the first place.
// See .planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md,
// finding 2.
// ---------------------------------------------------------------------------

// TestResumeThenSealThenEntombCompletes reproduces the exact field-report
// sequence: pause, resume ("return to work"), seal, then the archive
// preflight -- with no other command in between. seedVerifiedEntombLifecycleAt
// only writes .aether/HANDOFF.md when it is absent, so the note resume writes
// below survives the simulated seal step untouched, proving seal never
// rewrites or removes it.
func TestResumeThenSealThenEntombCompletes(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	if _, err := resumeColonyAt(fixture.now.Add(time.Minute)); err != nil {
		t.Fatalf("resume: %v", err)
	}

	handoffPath := filepath.Join(fixture.root, ".aether", "HANDOFF.md")
	if _, err := os.Stat(handoffPath); err != nil {
		t.Fatalf("return-to-work did not leave a hand-off note behind: %v", err)
	}

	seedVerifiedEntombLifecycleAt(t, fixture.root, fixture.dataDir)
	state := readEntombState199(t, fixture.dataDir)

	preflight, err := prepareEntombPreflight(entombTransactionInput{
		Root: fixture.root, DataRoot: fixture.dataDir, Now: fixture.now.Add(2 * time.Hour),
	}, state)
	if err != nil {
		t.Fatalf("archive preflight failed after resume -> seal: %v", err)
	}
	if len(preflight.Sources) == 0 {
		t.Fatal("expected a non-empty archive source list")
	}

	tombstone := entombPreparedSourceByKind(preflight, "tombstone_input")
	if tombstone == nil {
		t.Fatal("expected a tombstone_input source in the preflight")
	}
	if tombstone.Manifest.Synthesized {
		t.Fatal("the resume-written hand-off note should have been read from disk, not synthesized")
	}
}

// TestEntombStillFailsOnAGenuinelyMissingSource proves the synthesised
// fallback is scoped to exactly the tombstone_input (hand-off note) path: a
// genuinely missing COLONY_STATE.json -- a required source with no
// fallback -- still fails the archive by name.
func TestEntombStillFailsOnAGenuinelyMissingSource(t *testing.T) {
	saveGlobals(t)
	binding := bindCommandTestRepository(t)
	goal := "Exercise required-source failure"
	createTestColonyState(t, binding.DataDir, colony.ColonyState{
		Goal: &goal, CurrentPhase: 1, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted}}},
	})
	seedVerifiedEntombLifecycleAt(t, binding.Root, binding.DataDir)
	state := readEntombState199(t, binding.DataDir)

	statePath := filepath.Join(binding.DataDir, "COLONY_STATE.json")
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove colony state fixture: %v", err)
	}

	_, err := prepareEntombPreflight(entombTransactionInput{
		Root: binding.Root, DataRoot: binding.DataDir, Now: time.Now().UTC(),
	}, state)
	if err == nil {
		t.Fatal("expected the archive preflight to fail when COLONY_STATE.json is genuinely missing")
	}
	if !strings.Contains(err.Error(), "COLONY_STATE.json") {
		t.Fatalf("expected the error to name the missing COLONY_STATE.json source, got: %v", err)
	}
}

// TestEntombSynthesizedInputIsMarkedAsSynthesized proves the archive never
// presents invented content as retrieved content: a missing hand-off note
// produces a stand-in carrying the colony's own goal, phase count/completion
// state and the seal outcome's verdict, and that stand-in is recorded with a
// field distinguishing it from a source actually read from disk.
func TestEntombSynthesizedInputIsMarkedAsSynthesized(t *testing.T) {
	saveGlobals(t)
	binding := bindCommandTestRepository(t)
	goal := "Exercise synthesized tombstone input"
	createTestColonyState(t, binding.DataDir, colony.ColonyState{
		Goal: &goal, CurrentPhase: 1, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted}}},
	})
	seedVerifiedEntombLifecycleAt(t, binding.Root, binding.DataDir)

	handoffPath := filepath.Join(binding.Root, ".aether", "HANDOFF.md")
	if err := os.Remove(handoffPath); err != nil {
		t.Fatalf("remove handoff fixture: %v", err)
	}
	state := readEntombState199(t, binding.DataDir)

	preflight, err := prepareEntombPreflight(entombTransactionInput{
		Root: binding.Root, DataRoot: binding.DataDir, Now: time.Now().UTC(),
	}, state)
	if err != nil {
		t.Fatalf("archive preflight with a missing hand-off note: %v", err)
	}

	tombstone := entombPreparedSourceByKind(preflight, "tombstone_input")
	if tombstone == nil {
		t.Fatal("expected a tombstone_input source in the preflight")
	}
	if !tombstone.Manifest.Synthesized {
		t.Fatal("expected the missing hand-off note's source to be marked synthesized")
	}
	if tombstone.Actual != "" {
		t.Fatalf("expected a synthesized source to carry no live Actual path, got %q", tombstone.Actual)
	}
	if !strings.Contains(string(tombstone.Content), goal) {
		t.Fatalf("synthesized stand-in does not carry the colony's own goal: %s", tombstone.Content)
	}
	if !strings.Contains(string(tombstone.Content), string(state.State)) {
		t.Fatalf("synthesized stand-in does not carry the colony's completion state: %s", tombstone.Content)
	}
	if !strings.Contains(string(tombstone.Content), string(state.SealOutcome.Disposition)) {
		t.Fatalf("synthesized stand-in does not carry the seal outcome's verdict: %s", tombstone.Content)
	}
}

func entombPreparedSourceByKind(preflight entombPreflight, kind string) *entombPreparedSource {
	for i := range preflight.Sources {
		if preflight.Sources[i].Manifest.Kind == kind {
			return &preflight.Sources[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Task 2 (205-03): the seal preflight confirmation gate must never write
// human text to a machine-readable stdout stream. See the same field report,
// finding 4.
// ---------------------------------------------------------------------------

func sealFinalizeGateTestPreflight() SealPreflight {
	return SealPreflight{
		Disposition: colony.SealDispositionVerified,
		OutcomeKind: colony.OutcomeKindVerifiedCompletion,
	}
}

// TestSealFinalizeJSONStdoutIsPureJSON captures standard output for both the
// proceed and the do-not-proceed branch under machine-readable mode and
// asserts each parses as a single JSON document with no trailing bytes --
// the human preflight card and the confirmation question must never precede
// it.
func TestSealFinalizeJSONStdoutIsPureJSON(t *testing.T) {
	saveGlobals(t)
	bindCommandTestRepository(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	// Simulate the real seal-finalize invocation context: root.go's
	// PersistentPreRunE sets this from the cobra command name on every real
	// run. The direct, interactive `aether seal` command intentionally keeps
	// mixing prose with its JSON envelope (see
	// sealPreflightGateShouldWriteHumanOutput's doc comment) -- only a
	// -finalize command is held to the pure-JSON contract under test here.
	currentStreamingCommand = "seal-finalize"

	preflight := sealFinalizeGateTestPreflight()
	question := SealConfirmationCopy(preflight)
	if _, err := recordSealConfirmationAnswer(question, "yes", "seal-confirmation"); err != nil {
		t.Fatalf("record seal confirmation answer: %v", err)
	}

	// Do-not-proceed branch: SealConfirmationCopy's question text depends
	// only on Disposition (not on UnresolvedItems for a verified seal), so a
	// forced-incomplete disposition is what actually produces a different
	// question the recorded verified-seal "yes" above does not cover.
	forcedPreflight := SealPreflight{
		Disposition:     colony.SealDispositionForcedIncomplete,
		OutcomeKind:     colony.OutcomeKindForcedIncompleteClosure,
		UnresolvedItems: []SealUnresolvedItem{{Summary: "a check the recorded yes never named"}},
	}
	var notProceedBuf bytes.Buffer
	stdout = &notProceedBuf
	proceed, pending := runSealPreflightConfirmationGate(forcedPreflight)
	if proceed {
		t.Fatal("expected the gate not to proceed for a forced-incomplete question the recorded yes never covered")
	}
	if notProceedBuf.Len() != 0 {
		t.Fatalf("gate wrote %q to stdout under machine-readable mode (do-not-proceed branch)", notProceedBuf.String())
	}
	// The real callers (cmd/seal_final_review.go, cmd/codex_workflow_cmds.go)
	// both finish this branch with exactly this call.
	outputOK(pending)
	assertSingleJSONDocument(t, notProceedBuf.Bytes())

	// Proceed branch: the recorded "yes" covers today's exact, problem-free
	// question.
	var proceedBuf bytes.Buffer
	stdout = &proceedBuf
	proceed, pending = runSealPreflightConfirmationGate(preflight)
	if !proceed {
		t.Fatalf("expected the gate to proceed on a covering recorded yes, pending=%#v", pending)
	}
	if proceedBuf.Len() != 0 {
		t.Fatalf("gate wrote %q to stdout under machine-readable mode (proceed branch)", proceedBuf.String())
	}
	// Stand-in for whatever completeSealRuntime eventually emits -- the
	// point under test is that the gate itself never precedes it.
	outputOK(map[string]interface{}{"sealed": true})
	assertSingleJSONDocument(t, proceedBuf.Bytes())
}

// TestSealFinalizeHumanOutputIsUnchanged asserts the human-mode card text
// still appears exactly as before -- this is a routing fix only.
func TestSealFinalizeHumanOutputIsUnchanged(t *testing.T) {
	saveGlobals(t)
	store = nil
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	currentStreamingCommand = "seal-finalize"

	preflight := sealFinalizeGateTestPreflight()
	var buf bytes.Buffer
	stdout = &buf
	proceed, _ := runSealPreflightConfirmationGate(preflight)
	if proceed {
		t.Fatal("expected the gate to stop and ask without a recorded answer")
	}

	rendered := buf.String()
	if !strings.Contains(rendered, "Seal Preflight") {
		t.Fatalf("human-mode output is missing the preflight card: %s", rendered)
	}
	if !strings.Contains(rendered, SealConfirmationCopy(preflight)) {
		t.Fatalf("human-mode output is missing the confirmation question: %s", rendered)
	}
}

func assertSingleJSONDocument(t *testing.T, data []byte) {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("stdout did not parse as a single JSON document: %v\nstdout: %s", err, data)
	}
}
