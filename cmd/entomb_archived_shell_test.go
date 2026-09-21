package cmd

// Release 1.0.83 -- "a finished-and-archived project must not look damaged".
//
// A entomb'd (archived) colony writes an active-state shell that has no goal,
// no phases, and nothing in flight, with only an ArchiveReference pointing at
// the chamber that holds the full history. Between 1.0.79 and 1.0.82 that
// shell also silently retained Phase 200's plan-authority fields
// (acceptance_policy, active_revision_id, candidates, revisions) and the
// whole Specification, because the archive's clearing code was a hand-kept
// field list that was never updated when those fields were added three days
// later. The strict planning reader then refused the shell ("active plan
// phases do not match active revision"), which made a perfectly finished,
// already-archived project read as damaged on `aether resume` and
// `aether status`.
//
// This file locks two things: the shape entomb now writes has no such
// residue (TestEntombResetLeavesNoPlanAuthority, a reflection-based
// invariant so a field added to colony.Plan later and left uncleared fails
// this test by name), and a shell already left in the OLD, buggy shape by a
// 1.0.79-1.0.82 binary is tolerated in memory by every reader without ever
// rewriting the file on disk (TestArchivedShellFromOlderVersionReadsAsStartNew),
// while genuinely damaged states are not silently waved through
// (TestArchivedShellToleranceIsNarrow).

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// buildArchivedShellFromOlderVersionFixture drives the real
// init -> plan acceptance -> build -> continue -> seal -> entomb flow in a
// fresh downstream repo (via runRealLifecycleToSealForTest, the same helper
// TestFullLifecycleInDownstreamRepo uses), then re-attaches to the genuinely
// entombed on-disk state exactly the fields a 1.0.79-1.0.82 binary would have
// left behind -- copied from the real pre-entomb (sealed) state, never
// hand-typed. It returns the downstream repo root and the reconstructed
// old-shaped shell, and leaves that shell saved as COLONY_STATE.json on disk.
func buildArchivedShellFromOlderVersionFixture(t *testing.T) (string, colony.ColonyState) {
	t.Helper()
	downstream, sealedState := runRealLifecycleToSealForTest(t)

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb failed: %v", err)
	}

	var postEntomb colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &postEntomb); err != nil {
		t.Fatalf("load post-entomb state: %v", err)
	}

	// Simulate the 1.0.79-1.0.82 shape: everything the fixed entomb clears is
	// re-attached here, copied straight from the real pre-entomb state.
	oldShell := postEntomb
	oldShell.Plan.EvidencePolicy = sealedState.Plan.EvidencePolicy
	oldShell.Plan.AcceptancePolicy = sealedState.Plan.AcceptancePolicy
	oldShell.Plan.ActiveRevisionID = sealedState.Plan.ActiveRevisionID
	oldShell.Plan.PendingCandidateID = sealedState.Plan.PendingCandidateID
	oldShell.Plan.Candidates = sealedState.Plan.Candidates
	oldShell.Plan.Revisions = sealedState.Plan.Revisions
	oldShell.Plan.Phases = []colony.Phase{}
	oldShell.Specification = sealedState.Specification

	if oldShell.ArchiveReference == nil {
		t.Fatal("fixture guard: real post-entomb state had no ArchiveReference")
	}
	if oldShell.Plan.ActiveRevisionID == "" || len(oldShell.Plan.Revisions) == 0 || oldShell.Specification == nil {
		t.Fatal("fixture guard: pre-entomb sealed state did not carry plan authority + specification to re-attach")
	}

	if err := store.SaveJSON("COLONY_STATE.json", oldShell); err != nil {
		t.Fatalf("save old-shaped archived shell: %v", err)
	}
	return downstream, oldShell
}

// TestArchivedShellFromOlderVersionReadsAsStartNew is A2's positive case: a
// shell already left on disk in the pre-1.0.83 buggy shape must read as a
// confirmed, finished-and-archived project -- never as damaged -- and no
// reader may rewrite it to get there.
func TestArchivedShellFromOlderVersionReadsAsStartNew(t *testing.T) {
	downstream, _ := buildArchivedShellFromOlderVersionFixture(t)

	statePath := filepath.Join(downstream, ".aether", "data", "COLONY_STATE.json")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state before: %v", err)
	}

	root := resolveAetherRootPath()
	facts, err := loadLifecycleFacts(root, store, time.Now().UTC())
	if err != nil {
		t.Fatalf("loadLifecycleFacts: %v", err)
	}
	if facts.State.Source.Provenance != LifecycleFactConfirmed {
		t.Errorf("facts.State.Source.Provenance = %q, want %q (diagnostic: %s)",
			facts.State.Source.Provenance, LifecycleFactConfirmed, facts.State.Source.Diagnostic)
	}

	answer := resolveNextAction(loadNextActionInputForCommand(""))
	if answer.Projection == nil || answer.Projection.NextAction.ID != "initialize" {
		gotID := ""
		if answer.Projection != nil {
			gotID = answer.Projection.NextAction.ID
		}
		t.Errorf("closing action id = %q, want %q\nreason: %s", gotID, "initialize", answer.Recommendation)
	}

	var resumeOut bytes.Buffer
	oldStdout, oldStderr := stdout, stderr
	stdout, stderr = &resumeOut, &resumeOut
	rootCmd.SetArgs([]string{"resume"})
	resumeErr := rootCmd.Execute()
	stdout, stderr = oldStdout, oldStderr
	if resumeErr != nil {
		t.Fatalf("aether resume failed: %v", resumeErr)
	}
	if strings.Contains(strings.ToLower(resumeOut.String()), "malformed") {
		t.Errorf("aether resume output names the archived project malformed:\n%s", resumeOut.String())
	}

	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("read state after: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("COLONY_STATE.json was rewritten by read-only inspection\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestArchivedShellToleranceIsNarrow is A2's negative case: the tolerance
// above is for the archived shell shape ONLY. Any of three mutations that
// make the shell genuinely inconsistent -- no archive pointer, a live goal,
// or a surviving phase -- must still be treated as needing recovery, never
// silently accepted as "start the next one".
func TestArchivedShellToleranceIsNarrow(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(colony.ColonyState) colony.ColonyState
	}{
		{
			name: "no archive pointer",
			mutate: func(s colony.ColonyState) colony.ColonyState {
				s.ArchiveReference = nil
				return s
			},
		},
		{
			name: "goal present",
			mutate: func(s colony.ColonyState) colony.ColonyState {
				goal := "Still working on it"
				s.Goal = &goal
				return s
			},
		},
		{
			name: "phase present",
			mutate: func(s colony.ColonyState) colony.ColonyState {
				s.Plan.Phases = []colony.Phase{{ID: 1, Name: "Residual phase", Status: colony.PhaseReady}}
				return s
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			_, oldShell := buildArchivedShellFromOlderVersionFixture(t)
			mutated := test.mutate(oldShell)
			if err := store.SaveJSON("COLONY_STATE.json", mutated); err != nil {
				t.Fatalf("save mutated shell: %v", err)
			}

			root := resolveAetherRootPath()
			facts, err := loadLifecycleFacts(root, store, time.Now().UTC())
			if err != nil {
				t.Fatalf("loadLifecycleFacts: %v", err)
			}
			if facts.State.Source.Provenance != LifecycleFactMalformed {
				t.Errorf("facts.State.Source.Provenance = %q, want %q for mutation %q", facts.State.Source.Provenance, LifecycleFactMalformed, test.name)
			}

			answer := resolveNextAction(loadNextActionInputForCommand(""))
			gotID := ""
			if answer.Projection != nil {
				gotID = answer.Projection.NextAction.ID
			}
			if gotID == "initialize" {
				t.Errorf("closing action id = %q for mutation %q, want anything but %q", gotID, test.name, "initialize")
			}
		})
	}
}

// TestEntombResetLeavesNoPlanAuthority is A3: an invariant test that fails by
// name the moment a field is added to colony.Plan and not cleared by
// resetColonyStateForEntomb. It sets every settable field of colony.Plan to a
// non-zero value via reflection (plus a non-nil Specification), calls the
// real reset function, and asserts every Plan field came back to its zero
// value except Phases, which must be an empty, non-nil slice, and that
// Specification is nil.
func TestEntombResetLeavesNoPlanAuthority(t *testing.T) {
	var plan colony.Plan
	planValue := reflect.ValueOf(&plan).Elem()
	for i := 0; i < planValue.NumField(); i++ {
		setReflectFieldNonZeroForTest(t, planValue.Field(i))
	}

	state := colony.ColonyState{
		Plan:          plan,
		Specification: &colony.Specification{},
	}
	reset := resetColonyStateForEntomb(state)

	resetPlanValue := reflect.ValueOf(reset.Plan)
	resetPlanType := resetPlanValue.Type()
	for i := 0; i < resetPlanType.NumField(); i++ {
		field := resetPlanValue.Field(i)
		name := resetPlanType.Field(i).Name
		if name == "Phases" {
			if field.IsNil() {
				t.Errorf("Plan.Phases is nil after entomb reset, want an empty non-nil slice")
			} else if field.Len() != 0 {
				t.Errorf("Plan.Phases = %v after entomb reset, want empty", field.Interface())
			}
			continue
		}
		if !field.IsZero() {
			t.Errorf("Plan.%s = %#v after entomb reset, want the zero value -- a field was added to colony.Plan and not cleared by resetColonyStateForEntomb", name, field.Interface())
		}
	}
	if reset.Specification != nil {
		t.Errorf("Specification = %+v after entomb reset, want nil", reset.Specification)
	}
}

// setReflectFieldNonZeroForTest sets field to some non-zero value appropriate
// to its kind, recursing through pointers and filling one element into any
// slice so the slice itself is non-nil and non-empty. It only needs to
// support the kinds colony.Plan's own fields actually use.
func setReflectFieldNonZeroForTest(t *testing.T, field reflect.Value) {
	t.Helper()
	if !field.CanSet() {
		t.Fatalf("reflect: field of type %s is not settable", field.Type())
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString("non-zero-test-value")
	case reflect.Bool:
		field.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(1)
	case reflect.Float32, reflect.Float64:
		field.SetFloat(1.5)
	case reflect.Ptr:
		elem := reflect.New(field.Type().Elem())
		setReflectFieldNonZeroForTest(t, elem.Elem())
		field.Set(elem)
	case reflect.Slice:
		elemType := field.Type().Elem()
		slice := reflect.MakeSlice(field.Type(), 1, 1)
		if elemType.Kind() == reflect.Struct {
			setReflectFieldNonZeroForTest(t, slice.Index(0))
		}
		field.Set(slice)
	case reflect.Struct:
		if field.Type() == reflect.TypeOf(time.Time{}) {
			field.Set(reflect.ValueOf(time.Now()))
			return
		}
		for i := 0; i < field.NumField(); i++ {
			sub := field.Field(i)
			if sub.CanSet() {
				setReflectFieldNonZeroForTest(t, sub)
			}
		}
	default:
		t.Fatalf("reflect: unhandled kind %s for colony.Plan field of type %s -- extend setReflectFieldNonZeroForTest", field.Kind(), field.Type())
	}
}

// TestResumeOnAnArchivedProjectSaysThereIsNothingToResume covers the owner
// typing `aether resume` in a folder whose last project is finished and
// archived. Before this, resume answered "Recovery evidence is unknown" and
// "Inspect the conflicting recovery evidence" -- alarming and wrong: nothing
// conflicts, there is simply nothing to pick back up. Both the freshly
// entombed shape and the older-version shape must get the same plain answer,
// and resume must not rewrite the saved record to give it.
func TestResumeOnAnArchivedProjectSaysThereIsNothingToResume(t *testing.T) {
	for _, shape := range []string{"fresh", "older-version"} {
		t.Run(shape, func(t *testing.T) {
			var root string
			if shape == "fresh" {
				root, _ = runRealLifecycleToSealForTest(t)
				rootCmd.SetArgs([]string{"entomb", "--confirm"})
				if err := rootCmd.Execute(); err != nil {
					t.Fatalf("entomb failed: %v", err)
				}
			} else {
				root, _ = buildArchivedShellFromOlderVersionFixture(t)
			}
			statePath := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
			before, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatalf("read state: %v", err)
			}

			t.Setenv("AETHER_OUTPUT_MODE", "visual")
			t.Setenv("AETHER_PLATFORM", "claude")
			var out bytes.Buffer
			stdout, stderr = &out, &out
			defer func() { stdout, stderr = os.Stdout, os.Stderr }()
			rootCmd.SetArgs([]string{"resume"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("resume failed: %v", err)
			}
			got := out.String()
			for _, want := range []string{"finished and archived", "nothing to resume", "/ant-init"} {
				if !strings.Contains(got, want) {
					t.Errorf("resume on an archived project does not say %q:\n%s", want, got)
				}
			}
			// The shared what-next card must not make claims that are false for
			// an archived project: no goal is saved, and nothing is in flight
			// that closing the chat could lose.
			for _, forbidden := range []string{"conflicting", "malformed", "Recovery evidence is unknown", "Unresolved", "The goal is saved", "Don't close this chat"} {
				if strings.Contains(got, forbidden) {
					t.Errorf("resume on an archived project still says %q:\n%s", forbidden, got)
				}
			}
			after, err := os.ReadFile(statePath)
			if err != nil {
				t.Fatalf("re-read state: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("resume rewrote the archived project's saved record")
			}
		})
	}
}
