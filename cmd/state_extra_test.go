package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// --- state-checkpoint tests ---

func TestStateCheckpoint(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "before-rebuild"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["checkpoint"] != "before-rebuild" {
		t.Errorf("checkpoint = %v, want before-rebuild", result["checkpoint"])
	}
	if result["path"] != "checkpoints/before-rebuild.json" {
		t.Errorf("path = %v, want checkpoints/before-rebuild.json", result["path"])
	}

	// Verify the checkpoint file was created and has matching content
	var checkpoint colony.ColonyState
	if err := s.LoadJSON("checkpoints/before-rebuild.json", &checkpoint); err != nil {
		t.Fatalf("checkpoint file not created: %v", err)
	}
	if *checkpoint.Goal != "test goal" {
		t.Errorf("checkpoint goal = %q, want %q", *checkpoint.Goal, "test goal")
	}
}

func TestStateCheckpointMissingName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-checkpoint"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --name, got: %v", env["ok"])
	}
}

func TestStateCheckpointNoStateFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "test"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false when COLONY_STATE.json missing, got: %v", env["ok"])
	}
}

// --- state-write tests ---

func TestStateWrite(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "original goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-write", "--field", "version", "--value", "4.0"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}
	if result["field"] != "version" {
		t.Errorf("field = %v, want version", result["field"])
	}
	if result["value"] != "4.0" {
		t.Errorf("value = %v, want 4.0", result["value"])
	}

	// Verify the file was actually updated
	data, _ := s.ReadFile("COLONY_STATE.json")
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["version"] != "4.0" {
		t.Errorf("version in file = %v, want 4.0", m["version"])
	}
}

func TestStateWritePositionalArg(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// state-write should accept a positional JSON blob and write it directly
	rootCmd.SetArgs([]string{"state-write", `{"goal":"updated goal","version":"3.0"}`})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected state-write to accept positional JSON, got error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify the file was updated with the new JSON
	data, _ := s.ReadFile("COLONY_STATE.json")
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	if m["goal"] != "updated goal" {
		t.Errorf("goal in file = %v, want 'updated goal'", m["goal"])
	}
}

func TestStateWriteMissingField(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"state-write"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing --field, got: %v", env["ok"])
	}
}

// --- phase-insert tests ---

func TestPhaseInsert(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "1", "--name", "Fix Bug", "--description", "Fix critical bug"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["inserted"] != true {
		t.Errorf("inserted = %v, want true", result["inserted"])
	}
	// Sequential-ID invariant (H-02): the inserted phase takes the ID of its
	// slice position (index 1 → ID 2) and later phases renumber, instead of
	// the old max+1 assignment that produced orders like [1,3,2].
	if result["phase_id"] != float64(2) {
		t.Errorf("phase_id = %v, want 2", result["phase_id"])
	}
	if result["after"] != float64(1) {
		t.Errorf("after = %v, want 1", result["after"])
	}

	// Verify the phase was inserted at the right position
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) != 3 {
		t.Fatalf("phase count = %d, want 3", len(updated.Plan.Phases))
	}
	if updated.Plan.Phases[1].Name != "Fix Bug" {
		t.Errorf("inserted phase name = %q, want 'Fix Bug'", updated.Plan.Phases[1].Name)
	}
	if updated.Plan.Phases[1].Status != colony.PhasePending {
		t.Errorf("inserted phase status = %q, want pending", updated.Plan.Phases[1].Status)
	}
}

func TestPhaseInsertAtEnd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "1", "--name", "Phase 2", "--description", "Second phase"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["inserted"] != true {
		t.Errorf("inserted = %v, want true", result["inserted"])
	}

	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if len(updated.Plan.Phases) != 2 {
		t.Fatalf("phase count = %d, want 2", len(updated.Plan.Phases))
	}
	if updated.Plan.Phases[1].Name != "Phase 2" {
		t.Errorf("phase at index 1 = %q, want 'Phase 2'", updated.Plan.Phases[1].Name)
	}
}

func TestPhaseInsertReopensCompletedUnsealedColony(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 2,
		Milestone:    "Brood Stable",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "2", "--name", "Corrective Phase", "--description", "Repair blocker"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	var updated colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &updated); err != nil {
		t.Fatalf("failed to load updated state: %v", err)
	}
	if updated.State != colony.StateREADY {
		t.Fatalf("state = %s, want READY", updated.State)
	}
	if updated.CurrentPhase != 3 {
		t.Fatalf("current_phase = %d, want 3", updated.CurrentPhase)
	}
	if got := updated.Plan.Phases[2].Status; got != colony.PhaseReady {
		t.Fatalf("inserted phase status = %q, want ready", got)
	}
	if err := validateCodexBuildState(updated, 3, nil, false); err != nil {
		t.Fatalf("inserted phase should be buildable: %v", err)
	}
}

func TestPhaseInsertDoesNotReopenCrownedAnthill(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateCOMPLETED,
		CurrentPhase: 2,
		Milestone:    "Crowned Anthill",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "2", "--name", "Post Seal", "--description", "Should not reopen"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	var updated colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &updated); err != nil {
		t.Fatalf("failed to load updated state: %v", err)
	}
	if updated.State != colony.StateCOMPLETED {
		t.Fatalf("state = %s, want COMPLETED", updated.State)
	}
	if updated.CurrentPhase != 2 {
		t.Fatalf("current_phase = %d, want 2", updated.CurrentPhase)
	}
	if got := updated.Plan.Phases[2].Status; got != colony.PhasePending {
		t.Fatalf("inserted phase status = %q, want pending", got)
	}
	if err := validateCodexBuildState(updated, 3, nil, false); err == nil {
		t.Fatal("sealed Crowned Anthill insertion should not make the new phase buildable")
	}
}

func TestPhaseInsertInvalidAfter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Tasks: []colony.Task{}},
			},
		},
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"phase-insert", "--after", "5", "--name", "Bad", "--description", "Invalid"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid after index, got: %v", env["ok"])
	}
}

// TestPhaseInsertMissingRequiredFlagsFailLoudly locks behaviour that two
// independent audits misread as a silent no-op. The `if name == "" { return
// nil }` guards in phase-insert look like they swallow the failure, but
// mustGetString (cmd/helpers.go:76-79) has already emitted an ok:false
// envelope and set a non-zero exit code by then; the guards only stop the
// command continuing.
//
// It is locked rather than left implicit because the audit reading was
// plausible: a future refactor that switches these flags to a non-erroring
// accessor would turn the guards into exactly the silent success that was
// alleged, and no existing test would notice.
//
// Asserts all three halves of "fail loudly": exactly one ok:false envelope, a
// non-zero process exit via the real Execute() entry point, and an unchanged
// plan on disk.
func TestPhaseInsertMissingRequiredFlagsFailLoudly(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"missing description", []string{"phase-insert", "--after", "1", "--name", "Handle quoted fields"}},
		{"missing name", []string{"phase-insert", "--after", "1", "--description", "Parse quoted CSV fields"}},
		{"missing both", []string{"phase-insert", "--after", "1"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			var buf bytes.Buffer
			stdout = &buf
			stderr = &buf

			s, tmpDir := newTestStore(t)
			defer os.RemoveAll(tmpDir)
			store = s

			goal := "test"
			state := colony.ColonyState{
				Version: "3.0",
				Goal:    &goal,
				Plan: colony.Plan{
					Phases: []colony.Phase{
						{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
						{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{}},
					},
				},
			}
			s.SaveJSON("COLONY_STATE.json", state)

			rootCmd.SetArgs(tc.args)

			// Execute() (not rootCmd.Execute()) is the real entry point and the
			// only place the rendered-error exit code is converted into a
			// non-zero process result.
			if err := Execute(); err == nil {
				t.Errorf("expected a non-zero exit, got nil error (silent success)")
			}

			out := strings.TrimSpace(buf.String())
			if out == "" {
				t.Fatalf("expected an error message, got no output at all")
			}
			// Exactly one envelope: a second guard that re-reports the same
			// failure produces two JSON objects and breaks every consumer that
			// parses this output.
			if lines := len(strings.Split(out, "\n")); lines != 1 {
				t.Errorf("expected exactly 1 error envelope, got %d:\n%s", lines, out)
			}
			env := parseEnvelope(t, out)
			if env["ok"] != false {
				t.Errorf("expected ok:false, got: %v", env["ok"])
			}

			// The plan must be untouched — proving the command reported the
			// same thing it actually did.
			var after colony.ColonyState
			if err := s.LoadJSON("COLONY_STATE.json", &after); err != nil {
				t.Fatalf("state unreadable: %v", err)
			}
			if len(after.Plan.Phases) != 2 {
				t.Errorf("plan has %d phases, want 2 unchanged", len(after.Plan.Phases))
			}
		})
	}
}

type scriptedPhaseInsertPrompt struct {
	interactive bool
	answers     []string
	questions   []string
}

func (p *scriptedPhaseInsertPrompt) Interactive() bool {
	return p.interactive
}

func (p *scriptedPhaseInsertPrompt) Ask(question string) (string, error) {
	p.questions = append(p.questions, question)
	if len(p.answers) == 0 {
		return "", nil
	}
	answer := p.answers[0]
	p.answers = p.answers[1:]
	return answer, nil
}

func seedGuidedPhaseInsertState(t *testing.T, s interface {
	SaveJSON(string, interface{}) error
}, currentPhase int) {
	t.Helper()
	goal := "test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: currentPhase,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseCompleted, Tasks: []colony.Task{}},
				{ID: 2, Name: "Phase 2", Status: colony.PhaseReady, Tasks: []colony.Task{}},
				{ID: 3, Name: "Phase 3", Status: colony.PhasePending, Tasks: []colony.Task{}},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}
}

func executeGuidedPhaseInsert(t *testing.T, args []string, prompt phaseInsertPromptSession) (map[string]interface{}, colony.ColonyState) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedGuidedPhaseInsertState(t, s, 2)

	originalFactory := phaseInsertPromptSessionFactory
	phaseInsertPromptSessionFactory = func(*cobra.Command) phaseInsertPromptSession { return prompt }
	t.Cleanup(func() { phaseInsertPromptSessionFactory = originalFactory })

	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\n%s", args, err, buf.String())
	}

	env := parseEnvelope(t, strings.TrimSpace(buf.String()))
	if env["ok"] != true {
		t.Fatalf("execute %v returned ok=%v: %s", args, env["ok"], buf.String())
	}
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("execute %v result = %T, want object: %s", args, env["result"], buf.String())
	}

	var updated colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &updated); err != nil {
		t.Fatalf("load updated state: %v", err)
	}
	return result, updated
}

func TestGuidedInsertPhaseResolvesExplicitAndShorthandInput(t *testing.T) {
	nonInteractive := &scriptedPhaseInsertPrompt{interactive: false}
	tests := []struct {
		name            string
		args            []string
		wantAfter       int
		wantName        string
		wantDescription string
	}{
		{
			name:            "existing explicit automation remains compatible",
			args:            []string{"phase-insert", "--after", "1", "--name", "Fix auth", "--description", "Repair token persistence"},
			wantAfter:       1,
			wantName:        "Fix auth",
			wantDescription: "Repair token persistence",
		},
		{
			name:            "one issue sentence defaults after to current phase",
			args:            []string{"insert-phase", "login retries lose state"},
			wantAfter:       2,
			wantName:        "Stabilize login retries lose state",
			wantDescription: "login retries lose state",
		},
		{
			name:            "explicit flags override issue-derived fields",
			args:            []string{"insert-phase", "login retries lose state", "--after", "1", "--name", "Repair sessions", "--description", "Keep retries idempotent"},
			wantAfter:       1,
			wantName:        "Repair sessions",
			wantDescription: "Keep retries idempotent",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, updated := executeGuidedPhaseInsert(t, tc.args, nonInteractive)
			if got := int(result["after"].(float64)); got != tc.wantAfter {
				t.Fatalf("after = %d, want %d", got, tc.wantAfter)
			}
			insertAt := tc.wantAfter
			if got := updated.Plan.Phases[insertAt].Name; got != tc.wantName {
				t.Errorf("inserted name = %q, want %q", got, tc.wantName)
			}
			if got := updated.Plan.Phases[insertAt].Description; got != tc.wantDescription {
				t.Errorf("inserted description = %q, want %q", got, tc.wantDescription)
			}
		})
	}
}

func TestGuidedInsertPhaseBoundsDerivedNameAndRetainsConstraints(t *testing.T) {
	issue := "the login retry flow repeatedly loses session state during provider outages and reconnects"
	_, updated := executeGuidedPhaseInsert(t, []string{
		"insert-phase", issue,
		"--constraints", "Do not change the authentication provider",
	}, &scriptedPhaseInsertPrompt{interactive: false})

	inserted := updated.Plan.Phases[2]
	if !strings.HasPrefix(inserted.Name, "Stabilize ") {
		t.Fatalf("derived name = %q, want Stabilize prefix", inserted.Name)
	}
	if words := len(strings.Fields(inserted.Name)); words > 7 {
		t.Fatalf("derived name has %d words, want at most 7: %q", words, inserted.Name)
	}
	if len(inserted.Name) > 80 {
		t.Fatalf("derived name has %d bytes, want at most 80: %q", len(inserted.Name), inserted.Name)
	}
	for _, want := range []string{issue, "Do not change the authentication provider"} {
		if !strings.Contains(inserted.Description, want) {
			t.Errorf("description does not retain %q: %q", want, inserted.Description)
		}
	}
}

func TestGuidedInsertPhaseTTYAsksThreeQuestionsAndUsesSharedResolver(t *testing.T) {
	prompt := &scriptedPhaseInsertPrompt{
		interactive: true,
		answers: []string{
			"login retries lose state",
			"Retries preserve the authenticated session",
			"Do not change the authentication provider",
		},
	}

	result, updated := executeGuidedPhaseInsert(t, []string{"insert-phase"}, prompt)
	if len(prompt.questions) != 3 {
		t.Fatalf("prompt attempts = %d, want exactly 3: %v", len(prompt.questions), prompt.questions)
	}
	if got := int(result["after"].(float64)); got != 2 {
		t.Fatalf("after = %d, want current phase 2", got)
	}
	inserted := updated.Plan.Phases[2]
	if inserted.Name != "Stabilize login retries lose state" {
		t.Errorf("interactive name = %q, want same derived name as shorthand", inserted.Name)
	}
	for _, want := range []string{
		"login retries lose state",
		"Retries preserve the authenticated session",
		"Do not change the authentication provider",
	} {
		if !strings.Contains(inserted.Description, want) {
			t.Errorf("interactive description does not retain %q: %q", want, inserted.Description)
		}
	}
}

func TestGuidedInsertPhaseEmptyTTYAnswerCancelsWithoutMutation(t *testing.T) {
	prompt := &scriptedPhaseInsertPrompt{interactive: true, answers: []string{""}}

	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedGuidedPhaseInsertState(t, s, 2)
	before, err := s.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state before: %v", err)
	}

	originalFactory := phaseInsertPromptSessionFactory
	phaseInsertPromptSessionFactory = func(*cobra.Command) phaseInsertPromptSession { return prompt }
	t.Cleanup(func() { phaseInsertPromptSessionFactory = originalFactory })

	rootCmd.SetArgs([]string{"insert-phase"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	after, err := s.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("empty interactive answer mutated COLONY_STATE.json")
	}
	if len(prompt.questions) != 1 {
		t.Fatalf("prompt attempts = %d, want 1 before cancellation", len(prompt.questions))
	}
	env := parseEnvelope(t, strings.TrimSpace(buf.String()))
	result, _ := env["result"].(map[string]interface{})
	if result["status"] != "input_required" {
		t.Fatalf("status = %v, want input_required: %s", result["status"], buf.String())
	}
}

func TestGuidedInsertPhaseMissingNonTTYInputIsNonMutating(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	var buf bytes.Buffer
	stdout = &buf
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedGuidedPhaseInsertState(t, s, 2)
	before, err := s.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state before: %v", err)
	}

	originalFactory := phaseInsertPromptSessionFactory
	phaseInsertPromptSessionFactory = func(*cobra.Command) phaseInsertPromptSession {
		return &scriptedPhaseInsertPrompt{interactive: false}
	}
	t.Cleanup(func() { phaseInsertPromptSessionFactory = originalFactory })

	rootCmd.SetArgs([]string{"insert-phase"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	after, err := s.ReadFile("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("read state after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("missing non-TTY input mutated COLONY_STATE.json")
	}

	env := parseEnvelope(t, strings.TrimSpace(buf.String()))
	if env["ok"] != true {
		t.Fatalf("ok = %v, want structured input_required result: %s", env["ok"], buf.String())
	}
	result, ok := env["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result = %T, want object: %s", env["result"], buf.String())
	}
	if result["status"] != "input_required" {
		t.Errorf("status = %v, want input_required", result["status"])
	}
	example, _ := result["example"].(string)
	if example != `aether insert-phase "problem to stabilise"` {
		t.Errorf("example = %q, want exact quoted command", example)
	}
}

// --- validate-oracle-state tests ---

func TestValidateOracleState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("oracle/state.json", map[string]string{"status": "active"})
	s.SaveJSON("oracle/plan.json", map[string]string{"plan": "research"})

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}
	files := result["files"].(map[string]interface{})
	if files["state.json"] != true {
		t.Errorf("state.json valid = %v, want true", files["state.json"])
	}
	if files["plan.json"] != true {
		t.Errorf("plan.json valid = %v, want true", files["plan.json"])
	}
}

func TestValidateOracleStateMissing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false when files missing", result["valid"])
	}
	issues := result["issues"].([]interface{})
	if len(issues) != 2 {
		t.Errorf("issues count = %d, want 2", len(issues))
	}
}

func TestValidateOracleStateInvalidJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.AtomicWrite("oracle/state.json", []byte("not json"))
	s.SaveJSON("oracle/plan.json", map[string]string{"plan": "research"})

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false for invalid JSON", result["valid"])
	}
}

func TestValidateOracleStatePathValidation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Write oracle state via store.SaveJSON (the atomic storage path)
	s.SaveJSON("oracle/state.json", map[string]string{"status": "active"})
	s.SaveJSON("oracle/plan.json", map[string]string{"plan": "research"})

	rootCmd.SetArgs([]string{"validate-oracle-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Fatalf("expected valid=true when state written via SaveJSON, got: %v", result["valid"])
	}
	issues := result["issues"].([]interface{})
	for _, issue := range issues {
		if strings.Contains(issue.(string), "outside .aether/data/oracle") {
			t.Errorf("unexpected path validation issue: %s", issue)
		}
	}
}

// --- view-state tests ---

func TestViewStateInit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-init"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["initialized"] != true {
		t.Errorf("initialized = %v, want true", result["initialized"])
	}
}

func TestViewStateGetNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-get", "--key", "nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false", result["found"])
	}
	if result["value"] != nil {
		t.Errorf("value = %v, want nil", result["value"])
	}
}

func TestViewStateSetAndGet(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Set a value
	rootCmd.SetArgs([]string{"view-state-set", "--key", "theme", "--value", "dark"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["set"] != true {
		t.Errorf("set = %v, want true", result["set"])
	}

	// Now get it back
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-get", "--key", "theme"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("found = %v, want true", result["found"])
	}
	if result["value"] != "dark" {
		t.Errorf("value = %v, want dark", result["value"])
	}
}

func TestViewStateToggle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-toggle", "--key", "sidebar"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["value"] != true {
		t.Errorf("value = %v, want true (default false -> toggle to true)", result["value"])
	}

	// Toggle again should flip to false
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-toggle", "--key", "sidebar"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["value"] != false {
		t.Errorf("value = %v, want false (true -> toggle to false)", result["value"])
	}
}

func TestViewStateExpand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-expand", "--section", "details"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["expanded"] != true {
		t.Errorf("expanded = %v, want true", result["expanded"])
	}
	if result["section"] != "details" {
		t.Errorf("section = %v, want details", result["section"])
	}

	// Verify the key was set correctly
	buf.Reset()
	rootCmd.SetArgs([]string{"view-state-get", "--key", "expanded_details"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env = parseEnvelope(t, buf.String())
	result = env["result"].(map[string]interface{})
	if result["value"] != true {
		t.Errorf("expanded_details = %v, want true", result["value"])
	}
}

func TestViewStateCollapse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"view-state-collapse", "--section", "details"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["expanded"] != false {
		t.Errorf("expanded = %v, want false", result["expanded"])
	}
}

// --- grave tests ---

func TestGraveAdd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-add", "--agent", "builder-1", "--reason", "timeout", "--phase", "2"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["agent"] != "builder-1" {
		t.Errorf("agent = %v, want builder-1", result["agent"])
	}
	if result["buried"] != true {
		t.Errorf("buried = %v, want true", result["buried"])
	}
}

func TestGraveAddWithoutPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-add", "--agent", "builder-2", "--reason", "panic", "--phase", ""})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	// Verify the entry was saved
	var entries []GraveEntry
	s.LoadJSON("graveyard.json", &entries)
	if len(entries) != 1 {
		t.Fatalf("entries count = %d, want 1", len(entries))
	}
	if entries[0].Agent != "builder-2" {
		t.Errorf("agent = %q, want builder-2", entries[0].Agent)
	}
	if entries[0].Reason != "panic" {
		t.Errorf("reason = %q, want panic", entries[0].Reason)
	}
	if entries[0].Phase != "" {
		t.Errorf("phase = %q, want empty string", entries[0].Phase)
	}
	if entries[0].Created == "" {
		t.Error("created_at should not be empty")
	}
}

func TestGraveCheckFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	entries := []GraveEntry{
		{Agent: "builder-1", Reason: "timeout", Phase: "1", Created: "2026-01-01T00:00:00Z"},
		{Agent: "builder-2", Reason: "panic", Created: "2026-01-02T00:00:00Z"},
	}
	s.SaveJSON("graveyard.json", entries)

	rootCmd.SetArgs([]string{"grave-check", "--agent", "builder-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("found = %v, want true", result["found"])
	}
	matching := result["entries"].([]interface{})
	if len(matching) != 1 {
		t.Errorf("entries count = %d, want 1", len(matching))
	}
}

func TestGraveCheckNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-check", "--agent", "nonexistent"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false", result["found"])
	}
	entries := result["entries"].([]interface{})
	if len(entries) != 0 {
		t.Errorf("entries count = %d, want 0", len(entries))
	}
}

func TestGraveCheckNoFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"grave-check", "--agent", "builder-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false when no graveyard file", result["found"])
	}
}

// --- nil store tests ---
// Note: PersistentPreRunE always initializes the store before RunE runs,
// so nil store can only occur if PersistentPreRunE is bypassed.
// The nil checks exist as defensive guards in the code.

func TestStateCheckpointNilStore(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Verify the command works when store IS initialized (the normal path)
	goal := "test"
	state := colony.ColonyState{Version: "3.0", Goal: &goal}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-checkpoint", "--name", "test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}
}

func insertPhaseAcceptedPlanFixture(t *testing.T) (string, colony.ColonyState, string) {
	t.Helper()
	root, candidate := planCandidateTestPending(t)
	if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner"}); err != nil {
		t.Fatalf("accept insert-phase base candidate: %v", err)
	}
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatalf("open insert-phase fixture store: %v", err)
	}
	store = s
	state := mustReadSpecificationTestState(t, root)
	revision, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("accepted insert-phase fixture has no current specification")
	}
	ids := planImpactSpecificationIDs(revision)
	if len(ids) == 0 {
		t.Fatal("accepted insert-phase fixture has no specification item IDs")
	}
	return root, state, ids[0]
}

func executeInsertPhaseForCurrentPlan(t *testing.T, args ...string) (map[string]interface{}, error) {
	t.Helper()
	resetRootCmd(t)
	var out bytes.Buffer
	stdout = &out
	stderr = &out
	rootCmd.SetArgs(append([]string{"insert-phase"}, args...))
	err := rootCmd.Execute()
	if out.Len() == 0 {
		return nil, err
	}
	return parseEnvelope(t, out.String()), err
}

func TestInsertPhaseCandidatePreservesActiveRevisionAndAcceptsExactly(t *testing.T) {
	saveGlobals(t)
	root, before, specItemID := insertPhaseAcceptedPlanFixture(t)
	predecessor, ok := activePlanRevision(before.Plan)
	if !ok {
		t.Fatal("fixture has no active plan revision")
	}
	predecessorBytes, err := json.Marshal(predecessor)
	if err != nil {
		t.Fatal(err)
	}

	envelope, err := executeInsertPhaseForCurrentPlan(t,
		"login retries lose state",
		"--after", "1",
		"--spec-item", specItemID,
		"--base-plan-revision", predecessor.ID,
	)
	if err != nil {
		t.Fatalf("insert phase candidate: %v", err)
	}
	if envelope["ok"] != true {
		t.Fatalf("insert phase envelope = %#v, want success", envelope)
	}
	result := envelope["result"].(map[string]interface{})
	if result["candidate_created"] != true || result["inserted"] != false {
		t.Fatalf("insert result = %#v, want non-active candidate", result)
	}
	candidateID, _ := result["candidate_id"].(string)
	if candidateID == "" || result["review_command"] != "aether plan --candidate" || !strings.Contains(result["acceptance_command"].(string), candidateID) {
		t.Fatalf("insert result omits exact review/accept path: %#v", result)
	}

	afterCandidate := mustReadSpecificationTestState(t, root)
	if afterCandidate.Plan.ActiveRevisionID != before.Plan.ActiveRevisionID || !reflect.DeepEqual(afterCandidate.Plan.Phases, before.Plan.Phases) {
		t.Fatalf("candidate changed active plan: before=%+v after=%+v", before.Plan, afterCandidate.Plan)
	}
	retained, ok := activePlanRevision(afterCandidate.Plan)
	if !ok {
		t.Fatal("active predecessor disappeared after candidate creation")
	}
	retainedBytes, err := json.Marshal(retained)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retainedBytes, predecessorBytes) {
		t.Fatal("candidate creation mutated predecessor PlanRevision bytes")
	}

	artifact, err := loadPlanCandidateArtifact(root, candidateID)
	if err != nil {
		t.Fatalf("load insert phase candidate: %v", err)
	}
	if artifact.Candidate.Status != colony.PlanCandidatePendingReview || artifact.Candidate.BasePlanRevisionID != predecessor.ID {
		t.Fatalf("candidate binding = %+v, want pending against %s", artifact.Candidate, predecessor.ID)
	}
	accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(artifact.Candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner"})
	if err != nil {
		t.Fatalf("accept insert phase candidate through normal boundary: %v", err)
	}
	if accepted.Revision.ParentID != predecessor.ID || len(accepted.Revision.Phases) != len(before.Plan.Phases)+1 {
		t.Fatalf("accepted insert revision = %+v, want one successor phase", accepted.Revision)
	}
}

func TestInsertPhaseImmutableStableIDsAndCompletedStatus(t *testing.T) {
	saveGlobals(t)
	root, before, specItemID := insertPhaseAcceptedPlanFixture(t)
	if len(before.Plan.Phases) == 0 {
		t.Fatal("fixture has no phase")
	}
	before.Plan.Phases[0].Status = colony.PhaseCompleted
	for index := range before.Plan.Phases[0].Tasks {
		before.Plan.Phases[0].Tasks[index].Status = colony.TaskCompleted
	}
	if err := store.SaveJSON("COLONY_STATE.json", before); err != nil {
		t.Fatal(err)
	}
	base := before.Plan.ActiveRevisionID

	envelope, err := executeInsertPhaseForCurrentPlan(t,
		"add covered corrective work",
		"--after", "1",
		"--spec-item", specItemID,
		"--base-plan-revision", base,
	)
	if err != nil || envelope["ok"] != true {
		t.Fatalf("insert candidate failed: envelope=%#v err=%v", envelope, err)
	}
	candidateID := envelope["result"].(map[string]interface{})["candidate_id"].(string)
	artifact, err := loadPlanCandidateArtifact(root, candidateID)
	if err != nil {
		t.Fatal(err)
	}

	original := before.Plan.Phases[0]
	var preserved *colony.Phase
	maxID := 0
	for index := range artifact.Candidate.Proposal.Phases {
		phase := &artifact.Candidate.Proposal.Phases[index]
		if phase.ID > maxID {
			maxID = phase.ID
		}
		if phase.SemanticID == original.SemanticID {
			preserved = phase
		}
	}
	if preserved == nil || preserved.ID != original.ID || preserved.SemanticID != original.SemanticID {
		t.Fatalf("proposal did not preserve predecessor stable/display IDs: original=%+v proposal=%+v", original, artifact.Candidate.Proposal.Phases)
	}
	if maxID <= original.ID {
		t.Fatalf("inserted phase reused/renumbered predecessor ordinal: %+v", artifact.Candidate.Proposal.Phases)
	}

	accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(artifact.Candidate), planCandidateAcceptanceOptions{AcceptedBy: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	var completed *colony.Phase
	for index := range accepted.Revision.Phases {
		if accepted.Revision.Phases[index].SemanticID == original.SemanticID {
			completed = &accepted.Revision.Phases[index]
		}
	}
	if completed == nil || completed.Status != colony.PhaseCompleted {
		t.Fatalf("accepted insertion lost unaffected completion: %+v", accepted.Revision.Phases)
	}
}

func TestInsertPhaseRefusesMissingCoverageWithoutWrites(t *testing.T) {
	saveGlobals(t)
	root, before, _ := insertPhaseAcceptedPlanFixture(t)
	snapshot := planCandidateTestSnapshot(t, root)
	envelope, err := executeInsertPhaseForCurrentPlan(t,
		"unapproved new material scope", "--after", "1", "--base-plan-revision", before.Plan.ActiveRevisionID,
	)
	if err == nil || envelope == nil || envelope["ok"] != false || !strings.Contains(envelope["error"].(string), "specification coverage") {
		t.Fatalf("missing coverage result = %#v err=%v", envelope, err)
	}
	planCandidateTestAssertSnapshot(t, root, snapshot)
}

func TestInsertPhaseRefusesActiveAttemptWithoutWrites(t *testing.T) {
	saveGlobals(t)
	root, before, specItemID := insertPhaseAcceptedPlanFixture(t)
	before.State = colony.StateEXECUTING
	if err := store.SaveJSON("COLONY_STATE.json", before); err != nil {
		t.Fatal(err)
	}
	snapshot := planCandidateTestSnapshot(t, root)
	envelope, err := executeInsertPhaseForCurrentPlan(t,
		"do not race active work", "--after", "1", "--spec-item", specItemID, "--base-plan-revision", before.Plan.ActiveRevisionID,
	)
	if err == nil || envelope == nil || envelope["ok"] != false || !strings.Contains(envelope["error"].(string), "active") {
		t.Fatalf("active attempt result = %#v err=%v", envelope, err)
	}
	planCandidateTestAssertSnapshot(t, root, snapshot)
}

func TestInsertPhaseRefusesStaleBaseWithoutWrites(t *testing.T) {
	saveGlobals(t)
	root, before, specItemID := insertPhaseAcceptedPlanFixture(t)
	snapshot := planCandidateTestSnapshot(t, root)
	envelope, err := executeInsertPhaseForCurrentPlan(t,
		"stale corrective request", "--after", "1", "--spec-item", specItemID, "--base-plan-revision", before.Plan.ActiveRevisionID+"-stale",
	)
	if err == nil || envelope == nil || envelope["ok"] != false || !strings.Contains(envelope["error"].(string), "base plan revision") {
		t.Fatalf("stale base result = %#v err=%v", envelope, err)
	}
	planCandidateTestAssertSnapshot(t, root, snapshot)
}
