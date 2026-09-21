package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// RUNTIME-01: stale decision/session leakage between colonies.
//
// A new colony must not inherit the previous colony's conversational residue.
// The exposure is not hypothetical bookkeeping: pending-decisions.json is
// rendered into every worker prompt as CLARIFIED INTENT
// (cmd/colony_prime_context.go:843), and handoffs/worker-handoffs.json is
// rendered as Previous Worker Handoffs (:670). Before this test, init cleared
// session.json, worktrees and reviews — and left both of those files behind,
// so workers on a brand-new goal opened with the previous project's decisions
// and relay notes presented as their own colony's context.
func TestInitClearsPriorColonyDecisionResidue(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)
	dataDir := s.BasePath()

	// A sealed prior colony, so re-init is legitimate with --confirm-reinit.
	sealedGoal := "the previous project"
	prior := colony.ColonyState{
		Version: "3.0",
		Goal:    &sealedGoal,
		State:   colony.StateCOMPLETED,
	}
	if err := s.SaveJSON("COLONY_STATE.json", prior); err != nil {
		t.Fatalf("seed prior state: %v", err)
	}

	// Residue from the prior colony's conversations and workers. The
	// pending-decisions.json fixture carries two very different rows: an
	// old-style resolved clarification (the prior colony's own
	// conversation, which must never leak into the new one), and a still-
	// open note -- the owner's own "deal with this later" reminder, which is
	// not part of the old conversation and must survive into the new
	// project (with its phase number cleared).
	residue := map[string]string{
		"session.json":           `{"colony_goal":"the previous project"}`,
		"pending-decisions.json": `{"decisions":[{"id":"d1","question":"Old colony question?","status":"resolved","resolution":"an answer for the previous goal"},{"id":"n1","type":"note","description":"come back to the caching layer","resolved":false,"phase":3}]}`,
		"assumptions.json":       `{"assumptions":[{"text":"assumption made for the previous goal"}]}`,
	}
	for name, content := range residue {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	handoffDir := filepath.Join(dataDir, "handoffs")
	if err := os.MkdirAll(handoffDir, 0755); err != nil {
		t.Fatalf("seed handoffs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(handoffDir, "worker-handoffs.json"), []byte(`{"handoffs":[{"summary":"relay note from the previous colony"}]}`), 0644); err != nil {
		t.Fatalf("seed handoffs: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--confirm-reinit", "a completely new goal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("init did not create a colony: %v", err)
	}

	for _, leak := range []string{
		"assumptions.json",
		filepath.Join("handoffs", "worker-handoffs.json"),
	} {
		if _, err := os.Stat(filepath.Join(dataDir, leak)); !os.IsNotExist(err) {
			t.Errorf("%s survived re-init; the new colony's workers will be briefed with the previous colony's context", leak)
		}
	}

	// pending-decisions.json is a special case: it must survive re-init when
	// it still carries an owner's own open note or issue, but the prior
	// colony's own conversation (the resolved clarification) must not.
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("pending-decisions.json did not survive re-init even though it held an open note: %v", err)
	}
	if len(ff.Decisions) != 1 {
		t.Fatalf("pending-decisions.json after re-init = %+v, want exactly the one carried-forward note", ff.Decisions)
	}
	carried := ff.Decisions[0]
	if carried.ID != "n1" || carried.Description != "come back to the caching layer" {
		t.Errorf("the carried-forward row is not the owner's open note: %+v", carried)
	}
	if carried.Phase != nil {
		t.Errorf("the carried-forward note kept its old phase number %v; it should be cleared", *carried.Phase)
	}
	for _, decision := range ff.Decisions {
		if decision.ID == "d1" {
			t.Error("the prior colony's own resolved clarification (d1) survived re-init; it should not leak into the new colony's prompts")
		}
	}

	// session.json is recreated fresh for the NEW colony — assert it carries
	// the new goal rather than merely existing.
	sessionData, err := os.ReadFile(filepath.Join(dataDir, "session.json"))
	if err != nil {
		t.Fatalf("init did not create a fresh session: %v", err)
	}
	if string(sessionData) == residue["session.json"] {
		t.Error("session.json still holds the previous colony's session")
	}
}
