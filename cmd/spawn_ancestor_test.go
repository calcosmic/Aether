package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Plan 173-06 (SPAWN-05 red-proofs).
//
// The five tests below prove spawnAncestorCycleReason and normalizeSpawnTask
// by execution: a spawn tree cannot contain a literal cycle (a spawn-tree
// entry names one parent and is written once), so the hazard SPAWN-05
// guards against is repetition along the ancestor chain — a caste asked to
// do the same task it was itself spawned to do, one level down, never
// exceeding the depth cap and never terminating on its own.

// spawnLogArgsWithCasteTask builds a full spawn-log argument vector with an
// explicit caste and task, unlike spawn_enforce_test.go's spawnLogArgs
// (fixed "builder"/"t") — these tests need to control caste and task
// directly, since that is exactly what the ancestor-cycle check compares.
func spawnLogArgsWithCasteTask(parent, name, caste, task string) []string {
	return []string{"spawn-log", "--parent", parent, "--caste", caste, "--name", name, "--task", task, "--depth", "0"}
}

// TestSpawnCanSpawnDeniesAncestorCycle is the A-to-B-to-A proof, driven end
// to end through the CLI. Queen spawns A1 (depth 1, builder, task T); A1
// then attempts to spawn a child with the same caste and the same task it
// was itself asked to do (depth 2, well within the cap) — the loop a depth
// cap alone cannot see, because each hop is only one level down and the
// chain never terminates on its own.
func TestSpawnCanSpawnDeniesAncestorCycle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	const task = "fix the login form"

	// Queen -> A1 (depth 1, builder, task T).
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "A1", "builder", task))

	before, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt before attempt: %v", err)
	}

	// A1 attempts to spawn a child repeating its own caste and task, at
	// depth 2 — within the cap of 2, so a real ancestor-cycle check (not
	// the depth cap) must be what refuses this.
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgsWithCasteTask("A1", "C1", "builder", task))
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("spawn-log repeating A1's own caste and task did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	env := parseEnvelope(t, errBuf.String())
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "A1") {
		t.Fatalf("deny message does not name the matched ancestor %q: %s", "A1", errBuf.String())
	}

	after, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt after attempt: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("spawn-tree.txt bytes changed after a refused spawn:\nbefore=%q\nafter=%q", before, after)
	}

	// spawn-can-spawn registers no --caste/--task flags — only spawn-log's
	// authoritative call populates spawnDecisionInput.Caste/Task — so the
	// CLI alone cannot surface a "reason" field for this specific check.
	// Call the real, un-substituted spawnCanSpawnDecision var directly
	// (the same production seam spawn-log's RunE calls) with the exact
	// input spawn-log built for the attempt above, and confirm the reason
	// is "ancestor-cycle", not "depth" — if the reason were "depth", this
	// fixture would be proving the wrong check and the test would prove
	// nothing.
	result := spawnCanSpawnDecision(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 task,
	})
	if result.Allowed {
		t.Fatalf("direct spawnCanSpawnDecision call unexpectedly allowed the repeat")
	}
	if result.Reason != "ancestor-cycle" {
		t.Fatalf("expected reason %q, got %q (detail: %s) -- if the reason is \"depth\", the fixture is wrong and the test proves nothing", "ancestor-cycle", result.Reason, result.Detail)
	}
	if !strings.Contains(result.Detail, "A1") {
		t.Fatalf("deny detail does not name A1: %s", result.Detail)
	}
}

// TestSpawnAncestorCheckAllowsDifferentTaskSameCaste is the first
// false-positive control: without it, TestSpawnCanSpawnDeniesAncestorCycle
// could also be satisfied by a guard that refuses any repeated caste,
// which is a different and much worse behaviour than a repeated
// caste-AND-task pair. Two different tasks for the same caste in one chain
// is ordinary delegation.
func TestSpawnAncestorCheckAllowsDifferentTaskSameCaste(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "A1", "builder", "fix the login form"))

	result := runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("A1", "C1", "builder", "add pagination to the results list"))
	if result["recorded"] != true {
		t.Fatalf("same-caste different-task spawn was not recorded: %v", result)
	}
}

// TestSpawnAncestorCheckAllowsSameTaskDifferentCaste is the second
// false-positive control: the same task text reviewed or built by a
// different caste is ordinary delegation, not a repeat.
func TestSpawnAncestorCheckAllowsSameTaskDifferentCaste(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "A1", "builder", "fix the login form"))

	result := runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("A1", "C1", "watcher", "fix the login form"))
	if result["recorded"] != true {
		t.Fatalf("same-task different-caste spawn was not recorded: %v", result)
	}
}

// TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop is the
// bounded statement of the residue (T-173-31): normalizeSpawnTask matches
// text, not meaning. It asserts the four collapsing rules apply (case,
// whitespace runs, leading/trailing spaces, one trailing full stop) and
// that nothing else is collapsed — three pairs that must NOT collide are
// asserted alongside the pairs that must.
func TestNormalizeSpawnTaskMatchesOnlyWhitespaceCaseAndTrailingStop(t *testing.T) {
	collide := []struct{ a, b string }{
		{"Fix the login form.", "fix the login form"},
		{"  fix   the login   form  ", "fix the login form"},
		{"FIX THE LOGIN FORM", "fix the login form."},
	}
	for _, tc := range collide {
		na, nb := normalizeSpawnTask(tc.a), normalizeSpawnTask(tc.b)
		if na != nb {
			t.Fatalf("expected %q and %q to normalize to the same string, got %q and %q", tc.a, tc.b, na, nb)
		}
	}

	distinct := []struct{ a, b, why string }{
		{"fix the login form", "fix the signup form", "differ by a middle word"},
		{"fix bug 123", "fix bug 124", "differ by a number"},
		{"add pagination, sort order", "add pagination sort order", "differ by an interior punctuation mark"},
	}
	for _, tc := range distinct {
		na, nb := normalizeSpawnTask(tc.a), normalizeSpawnTask(tc.b)
		if na == nb {
			t.Fatalf("expected %q and %q (%s) to normalize differently, both became %q -- normalizeSpawnTask must not collapse more than case, whitespace, and a trailing full stop", tc.a, tc.b, tc.why, na)
		}
	}

	if got, want := normalizeSpawnTask("  Hello   World.  "), "hello world"; got != want {
		t.Fatalf("normalizeSpawnTask(%q) = %q, want %q", "  Hello   World.  ", got, want)
	}
	if got, want := normalizeSpawnTask("Wait, don't deploy."), "wait, don't deploy"; got != want {
		t.Fatalf("normalizeSpawnTask should lowercase and strip only a trailing full stop, preserving interior punctuation: got %q, want %q", got, want)
	}
}

// TestSpawnAncestorCheckFailsClosedOnUnreadableTree is the D-19/T-173-30
// proof: when the ancestor chain cannot be walked, the check denies rather
// than allowing. It carries its own negative control in the same function
// — a readable tree with no matching ancestor allows — showing the guard
// is not simply always-deny.
func TestSpawnAncestorCheckFailsClosedOnUnreadableTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "A1", "builder", "fix the login form"))

	// Negative control: with the tree still readable, a caste/task pair
	// that does not appear anywhere in A1's chain is allowed.
	allowReason := spawnAncestorCycleReason(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "watcher",
		Task:                 "review the pull request",
	})
	if allowReason != "" {
		t.Fatalf("expected a readable tree with no matching ancestor to allow, got deny reason: %q", allowReason)
	}

	// Corrupt the tree on disk directly (bypassing store writes): a
	// non-numeric depth field on a 7-field line now makes the whole ledger
	// fail to parse at all (it returns a non-nil error wrapping
	// agent.ErrSpawnTreeCorrupt), so the ancestor chain is unreadable —
	// the same failure mode a hand-edited or corrupted tree would produce.
	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath, []byte("not-a-timestamp|Queen|builder|A1|fix the login form|not-a-number|spawned\n"), 0644); err != nil {
		t.Fatalf("corrupt spawn-tree.txt: %v", err)
	}

	denyReason := spawnAncestorCycleReason(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 "fix the login form",
	})
	if denyReason == "" {
		t.Fatalf("expected an unreadable ancestor chain to deny (D-19 fail-closed), got allow")
	}
	if !strings.Contains(denyReason, "unreadable") {
		t.Fatalf("deny reason does not name the chain as unreadable: %q", denyReason)
	}
}
