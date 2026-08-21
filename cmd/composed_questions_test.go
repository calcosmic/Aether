package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS4 — Queen-composed questions: the wrapper's model composes clarification
// questions from THIS goal and THIS codebase, and the runtime materializes
// them into the SAME pending-decisions pipeline the canned generator uses.
// Grounding is typed and required; hard constraints ride a typed field.

func seedComposedQuestionColony(t *testing.T) {
	t.Helper()
	goal := "Ship the CSV export for the reports page"
	state := colony.ColonyState{Version: "3.0", State: colony.StateREADY, Goal: &goal}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}
}

func runDiscussArgs(t *testing.T, buf, errBuf *bytes.Buffer, args ...string) map[string]interface{} {
	t.Helper()
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(append([]string{"discuss"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("discuss %v: %v\nstderr: %s", args, err, errBuf.String())
	}
	var env map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("parse envelope: %v\nraw: %s", err, buf.String())
	}
	result, _ := env["result"].(map[string]interface{})
	return result
}

func TestDiscussAddQuestionRoundTrip(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedComposedQuestionColony(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	result := runDiscussArgs(t, &buf, &errBuf,
		"--add-question", "The reports page already streams JSON — should the CSV export stream too, or buffer whole files?",
		"--options", "Stream like JSON|Buffer whole files",
		"--category", "integration",
		"--grounding", "discuss-analyze found streaming JSON handlers in reports/handler.go",
		"--hard",
	)
	if result["created"] != true {
		t.Fatalf("composed question not created: %v", result)
	}
	id, _ := result["id"].(string)
	if id == "" {
		t.Fatalf("no decision id returned: %v", result)
	}

	// The decision landed in the SAME pipeline, with typed fields.
	var pending PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &pending); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	var decision *PendingDecision
	for i := range pending.Decisions {
		if pending.Decisions[i].ID == id {
			decision = &pending.Decisions[i]
		}
	}
	if decision == nil {
		t.Fatalf("composed decision not persisted")
	}
	if !decision.HardConstraint {
		t.Fatalf("hard constraint not recorded as a typed field: %+v", decision)
	}
	if !strings.Contains(decision.Grounding, "reports/handler.go") {
		t.Fatalf("grounding not persisted: %+v", decision)
	}
	if question, opts := parseClarificationDescription(decision.Description); !strings.Contains(question, "CSV export") || len(opts) != 2 {
		t.Fatalf("description does not round-trip through the canned parser: q=%q opts=%v", question, opts)
	}

	// Resolution is the UNCHANGED pipeline: a hard clarification's answer
	// becomes a REDIRECT signal.
	runDiscussArgs(t, &buf, &errBuf, "--resolve", id, "--answer", "Stream like JSON")
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatalf("load pheromones: %v", err)
	}
	found := false
	for _, sig := range pf.Signals {
		if sig.Type == "REDIRECT" && strings.Contains(string(sig.Content), "Stream like JSON") {
			found = true
		}
	}
	if !found {
		t.Fatalf("resolving a hard composed question did not emit a REDIRECT signal: %+v", pf.Signals)
	}
}

func TestDiscussAddQuestionRequiresGrounding(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedComposedQuestionColony(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"discuss", "--add-question", "Some question?", "--options", "a|b"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("an ungrounded composed question was accepted: %s", buf.String())
	}
	if !strings.Contains(errBuf.String(), "grounding") {
		t.Fatalf("rejection does not explain the grounding requirement: %s", errBuf.String())
	}
	var pending PendingDecisionFile
	_ = store.LoadJSON(pendingDecisionsFile, &pending)
	if len(pending.Decisions) != 0 {
		t.Fatalf("ungrounded question was persisted anyway: %+v", pending.Decisions)
	}
}

func TestDiscussAddQuestionDedupsBySource(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	seedComposedQuestionColony(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	args := []string{
		"--add-question", "Should exports need auth?",
		"--options", "Yes|No",
		"--category", "verification",
		"--grounding", "the goal names a user-facing export surface",
		"--source", "wrapper:export-auth",
	}
	first := runDiscussArgs(t, &buf, &errBuf, args...)
	if first["created"] != true {
		t.Fatalf("first materialization not created: %v", first)
	}
	second := runDiscussArgs(t, &buf, &errBuf, args...)
	if second["created"] != false {
		t.Fatalf("re-running the same composition doubled the question: %v", second)
	}
	var pending PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &pending); err != nil {
		t.Fatalf("load pending decisions: %v", err)
	}
	if len(pending.Decisions) != 1 {
		t.Fatalf("dedup failed: %d decisions", len(pending.Decisions))
	}
}

// TestHardConstraintTypedField proves control flow rides the typed field:
// no ":hard" source suffix anywhere, yet the constraint holds.
func TestHardConstraintTypedField(t *testing.T) {
	decision := PendingDecision{Source: "wrapper:plain-slug", HardConstraint: true}
	if !clarificationIsHardConstraint(decision) {
		t.Fatalf("typed HardConstraint field does not drive the hard-constraint decision")
	}
	// The legacy suffix still works as a fallback for existing data.
	if !clarificationIsHardConstraint(PendingDecision{Source: "discuss:scope:hard"}) {
		t.Fatalf("legacy :hard suffix fallback broken")
	}
	if clarificationIsHardConstraint(PendingDecision{Source: "wrapper:plain-slug"}) {
		t.Fatalf("a plain question is treated as a hard constraint")
	}
}

// TestDiscussWrapperComposesGroundedQuestions is the check that fails if
// discuss regresses to same-three-canned-questions: the wrapper contract
// pins compose-first with grounding, and the fallback stays a typed
// condition.
func TestDiscussWrapperComposesGroundedQuestions(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/discuss.md", "../.claude/commands/ant-discuss.md", "../.opencode/commands/ant/discuss.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{
			"Compose the Questions",
			"SPECIFIC to this goal and this",
			"--grounding",
			"--add-question",
			"AskUserQuestion",
			"Nothing is recorded without the\n   user's explicit pick",
		} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the composition contract anchor %q", path, anchor)
			}
		}
	}
}

func TestDiscussGeneratorIsFallbackOnly(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/discuss.md", "../.claude/commands/ant-discuss.md", "../.opencode/commands/ant/discuss.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		composeIdx := strings.Index(text, "Compose the Questions")
		fallbackIdx := strings.Index(text, "Canned fallback (typed condition")
		if composeIdx == -1 || fallbackIdx == -1 {
			t.Fatalf("%s missing compose or fallback sections", path)
		}
		if fallbackIdx < composeIdx {
			t.Fatalf("%s orders the canned generator before composition", path)
		}
		if !strings.Contains(text, "no scan context or the goal is empty") {
			t.Fatalf("%s fallback condition is not typed", path)
		}
	}
}
