package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// SPAWN-04 / D-19.
//
// spawn-can-spawn-swarm used to answer "yes, spawn away, budget 5" the
// moment it could not read COLONY_STATE.json or spawn-tree.txt -- a guard
// that cannot see its own data reporting a checked answer it never checked.
// These tests are fault-injection red-proofs: they must go red the moment
// either fail-open branch returns.
//
// Bounded residue, discovered while writing these tests: agent.SpawnTree's
// Parse() (pkg/agent/spawn_tree.go) treats a store.ReadFile error on
// spawn-tree.txt identically to a missing file -- it swallows the error
// inside parseFile() and always returns a nil error, and malformed
// pipe-delimited lines are silently skipped rather than surfaced as an
// error. Parse() itself can therefore never return a non-nil error, and no
// byte content exists that makes it do so. cmd/internal_cmds.go's Task 1 fix
// works around this by checking existence and type (os.Stat) ahead of
// calling Parse(), which is what
// TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree exercises below: it
// makes the tree path genuinely unreadable by putting a directory there
// (store.ReadFile / os.ReadFile fail on a directory), which is a real input
// Parse()'s caller rejects, even though Parse() would not reject it itself.

func writeValidColonyState(t *testing.T, s interface {
	SaveJSON(string, interface{}) error
}) {
	t.Helper()
	goal := "prove the guard fails closed"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write valid COLONY_STATE.json: %v", err)
	}
}

// TestSpawnCanSpawnSwarmFailsClosedOnUnreadableColonyState is the T-173-12
// assertion: an unloadable COLONY_STATE.json must deny, not grant a full
// budget.
func TestSpawnCanSpawnSwarmFailsClosedOnUnreadableColonyState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Invalid JSON bytes are sufficient and portable: store.LoadJSON reads
	// the file fine but json.Unmarshal fails, which is the load error path
	// spawnCanSpawnSwarmCmd actually treats as "state unreadable".
	statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
	if err := os.WriteFile(statePath, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("corrupt COLONY_STATE.json: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn-swarm"})
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("unreadable colony state did not set a non-zero exit code: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}

	env := parseEnvelope(t, errBuf.String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false, got: %s", errBuf.String())
	}
	details, _ := env["details"].(map[string]interface{})
	if details == nil || details["can_spawn"] != false {
		t.Fatalf("expected can_spawn:false in details, got: %s", errBuf.String())
	}
	msg, _ := env["error"].(string)
	if !strings.Contains(msg, "cannot verify spawn budget") {
		t.Fatalf("error message %q does not contain %q", msg, "cannot verify spawn budget")
	}
}

// TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree is the T-173-13
// assertion: an unreadable spawn-tree.txt must deny rather than be counted
// as zero live spawns with the full budget free. See the bounded-residue
// note above the file's top: this test makes the tree unreadable by putting
// a directory at its path, since agent.SpawnTree.Parse() cannot itself be
// made to return an error.
func TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	writeValidColonyState(t, store)

	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.MkdirAll(treePath, 0755); err != nil {
		t.Fatalf("create directory at spawn-tree.txt path: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn-swarm"})
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("unreadable spawn tree did not set a non-zero exit code: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}

	env := parseEnvelope(t, errBuf.String())
	if env["ok"] != false {
		t.Fatalf("expected ok:false, got: %s", errBuf.String())
	}
	details, _ := env["details"].(map[string]interface{})
	if details == nil || details["can_spawn"] != false {
		t.Fatalf("expected can_spawn:false in details, got: %s", errBuf.String())
	}
	msg, _ := env["error"].(string)
	if !strings.Contains(msg, "cannot verify spawn budget") {
		t.Fatalf("error message %q does not contain %q", msg, "cannot verify spawn budget")
	}
}

// TestSpawnCanSpawnSwarmAllowsWhenStateAndTreeAreBothReadable is the T-173-14
// negative control. Without it, the two deny tests above would also pass
// against a guard that denies unconditionally -- a different defect than the
// one this plan fixes. A valid state and a tree with fewer live entries than
// the budget must still report can_spawn:true with a zero exit code.
func TestSpawnCanSpawnSwarmAllowsWhenStateAndTreeAreBothReadable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	writeValidColonyState(t, store)

	// One active spawn, well under the default budget of 5. The single space
	// before "active" (and nowhere else) is deliberate: it is what makes this
	// fixture double as the red-proof fixture below -- agent.SpawnTree.Parse()
	// trims the status field, so it reads as pipe-delimited exactly as
	// intended, while the old whitespace-token scanner this plan replaces
	// (splitLineFields) also sees "active" as a standalone token and counts
	// it. A line with a task description containing spaces (e.g. "do the
	// work") defeats the old scanner entirely, which is the undercount bug
	// T-173-15 names -- this fixture avoids that so the negative control
	// stays green under both implementations.
	treeLine := "2026-08-12T00:00:00Z|queen|builder|Mason-1|work|1| active\n"
	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath, []byte(treeLine), 0644); err != nil {
		t.Fatalf("write valid spawn-tree.txt: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn-swarm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("valid state + valid tree returned error: %v\nstderr: %s", err, errBuf.String())
	}

	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("valid state + valid tree set a non-zero exit code %d: stdout=%s stderr=%s", code, buf.String(), errBuf.String())
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", buf.String())
	}
	result, _ := env["result"].(map[string]interface{})
	if result == nil || result["can_spawn"] != true {
		t.Fatalf("expected can_spawn:true, got: %s", buf.String())
	}
	if spawns, _ := result["current_spawns"].(float64); spawns != 1 {
		t.Fatalf("expected current_spawns:1, got: %s", buf.String())
	}
}
