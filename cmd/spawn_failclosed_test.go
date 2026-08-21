package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
// Bounded residue, discovered while writing these tests, and corrected
// 2026-08-13 by plan 173-11: at the time these tests were first written,
// agent.SpawnTree's Parse() (pkg/agent/spawn_tree.go) treated a
// store.ReadFile error on spawn-tree.txt the same as a missing file -- it
// swallowed the error inside parseFile() and always returned a nil error,
// and a malformed pipe-delimited line was silently skipped rather than
// surfaced as an error. No byte content could make Parse() return an error
// at that time. cmd/internal_cmds.go's Task 1 fix worked around this by
// checking existence and type (os.Stat) ahead of calling Parse(), which is
// what TestSpawnCanSpawnSwarmFailsClosedOnUnreadableSpawnTree exercises
// below: it makes the tree path unreadable by putting a directory there
// (store.ReadFile / os.ReadFile fail on a directory), a real input Parse()'s
// caller rejected even though Parse() would have accepted it.
//
// What changed in plan 173-11: parseFile() now draws a three-way
// distinction instead of collapsing everything into "empty, no error". An
// absent file, a zero-byte file, or a whitespace-only file still parse to
// empty with a nil error -- the one narrow, justified exception (T-173-24),
// so a fresh colony is never locked out of spawning. Every other outcome --
// a read failure that is not "file does not exist" (a directory at the
// path, a permission denial), or content that does not parse as valid
// pipe-delimited spawn-tree format -- now returns a non-nil error wrapping
// agent.ErrSpawnTreeCorrupt. The identical three-way distinction now applies
// to spawn-runs.json (loadRunStateLocked). The "ledger-corrupt" axes added
// by plan 173-13 below, on every guard that reads spawn-tree.txt, and the
// "run-state-obstructed" axis on spawn-log, are the proof that each guard
// actually turns this new error into a deny.

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
// as zero live spawns with the full budget free. This test makes the tree
// unreadable by putting a directory at its path.
//
// Corrected 2026-08-13 (plan 173-11 gave agent.SpawnTree.Parse() a real
// error return): before 173-11, the directory route was the ONLY fault this
// guard could be exercised against, because no byte content written over
// spawn-tree.txt made Parse() return an error. The directory route and a
// corrupted-content route are now two distinct, separately exercised
// faults -- this test proves the directory route (an os.Stat rejection
// ahead of Parse()); the "ledger-corrupt" axis on this same command in
// delegationGuardFaultTable below proves the content route (Parse() itself
// rejecting a present-but-unparseable file).
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

// Plan 173-10 task 2 (SPAWN-04/D-19, cross-guard proof).
//
// The tests above prove spawn-can-spawn-swarm alone. The two tests below
// prove the property across EVERY delegation guard at once, so a future
// guard added without its own fault-injection test cannot silently ship
// fail-open. This plan's <interfaces> section ("What each guard actually
// reads") is the source of truth for which fault each guard is genuinely
// subject to: three of the four guards never read COLONY_STATE.json, and
// hook-pre-tool-use reads no file at all. Asserting a colony-state fault
// against a guard that does not read it would force an unplanned colony-state
// read into the decision chain just to make the test pass -- exactly what
// this table refuses to do.

// delegationGuardFaultAxis is one fault a delegation guard is genuinely
// subject to, named per this plan's <interfaces> "What each guard actually
// reads" section -- never a file the guard does not read.
type delegationGuardFaultAxis struct {
	// Name identifies the axis in test names and failure output.
	Name string
	// NotExercisable, when non-empty, states why this axis cannot currently
	// be injected -- for example, a fault whose only known trigger requires
	// production code this repository does not have yet. The axis is
	// recorded with its reason rather than deleted, so
	// TestDelegationGuardTableCoversEveryGuardCommand can see it was
	// considered rather than silently dropped. Verify must be nil when this
	// is set.
	//
	// Corrected 2026-08-13 (plan 173-13): this comment's worked example used
	// to assert a specific claim about agent.SpawnTree.Parse()'s runtime
	// behaviour. Plan 173-11 made that claim false, and an example that
	// asserts current runtime behaviour is exactly how the drift happened --
	// described further in the file-top note above. The example here is now
	// abstract on purpose.
	NotExercisable string
	// Verify sets up its own isolated store, injects the fault, executes the
	// guard, and asserts both a deny outcome and a non-empty deny reason.
	Verify func(t *testing.T)
}

// delegationGuardTableEntry covers one registered delegation-guard command.
type delegationGuardTableEntry struct {
	// Command is the exact registered cobra command name (cobra's Name(),
	// the first word of Use) this entry covers.
	Command string
	Axes    []delegationGuardFaultAxis
}

// delegationGuardFaultTable is the enumerated set TestDelegationGuardTableCoversEveryGuardCommand
// holds to the live command registry, and TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs
// executes axis by axis.
var delegationGuardFaultTable = []delegationGuardTableEntry{
	{
		Command: "spawn-can-spawn",
		Axes: []delegationGuardFaultAxis{
			{
				// Corrupt bytes in spawn-runs.json that read successfully but
				// fail json.Unmarshal -- the genuine run-state error path
				// (loadRunStateLocked swallows a ReadFile error as empty
				// state, per pkg/agent/spawn_tree.go). Reaches
				// spawnTreeBudgetReason's error branch via
				// spawnCanSpawnDecision's second check.
				Name: "run-state-unreadable",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
					if err := os.WriteFile(runStatePath, []byte("{not valid json"), 0644); err != nil {
						t.Fatalf("corrupt spawn-runs.json: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs([]string{"spawn-can-spawn", "--enforce"})
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-can-spawn --enforce with an unreadable run state did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "budget") {
						t.Fatalf("deny message does not name the budget: %s", errBuf.String())
					}
				},
			},
			{
				// A single line of non-pipe-format text written directly over
				// spawn-tree.txt as a regular file -- 173-VERIFICATION.md's
				// Gap 1 reproduction, byte-for-byte. The ledger is present but
				// its content is not a valid spawn ledger, so st.Parse() inside
				// spawnTreeBudgetState (cmd/spawn_budget.go) returns a non-nil
				// error wrapping agent.ErrSpawnTreeCorrupt, and
				// spawnTreeBudgetReason denies rather than treating the
				// corruption as an empty, fresh tree. This is deliberately NOT
				// the run-state-unreadable axis above (which corrupts
				// spawn-runs.json, a different file) and not a directory at
				// the path (which is what an unreadable-tree axis would use,
				// were one needed here) -- this axis proves content rejection
				// specifically, which is the fault the exploit actually used.
				Name: "ledger-corrupt",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
					if err := os.WriteFile(treePath, []byte("garbage not pipe format"), 0644); err != nil {
						t.Fatalf("corrupt spawn-tree.txt: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs([]string{"spawn-can-spawn", "--enforce"})
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-can-spawn --enforce against a corrupted ledger did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "budget") || !strings.Contains(msg, "unverifiable") {
						t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
					}
				},
			},
			{
				// --name populates RequesterName even when the lookup fails
				// (cmd/spawn.go's spawnCanSpawnCmd), so a --name naming an
				// agent absent from a READABLE spawn-tree.txt reaches
				// spawnAncestorCycleReason's unresolvable-startName branch.
				// Plan 173-12 task 1 moved the ledger-integrity check
				// (st.Parse()) ahead of the budget's no-run-yet exception, so
				// a genuinely CORRUPT tree is now caught -- and denied -- by
				// the budget check first, before this ancestor path is ever
				// reached. The fault this axis injects is therefore a
				// WELL-FORMED tree that simply does not mention A1, so the
				// ancestor guard's own proof stays pinned to the ancestor
				// guard rather than being satisfied by the budget instead.
				Name: "requester-not-recorded",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf

					// Record a real A1 first, so the overwrite below is a
					// genuine "was recorded, now isn't" case.
					runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "A1", "0"))

					// Overwrite the tree with a well-formed 7-field spawn
					// line for a DIFFERENT agent: the ledger parses cleanly
					// (one helper of twenty, well under the whole-run
					// budget), and A1 is genuinely absent from a READABLE
					// tree rather than dropped by a parse failure.
					treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
					if err := os.WriteFile(treePath, []byte("2026-04-01T12:00:00Z|Queen|builder|B9|t|1|spawned\n"), 0644); err != nil {
						t.Fatalf("overwrite spawn-tree.txt: %v", err)
					}

					buf.Reset()
					errBuf.Reset()
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs([]string{"spawn-can-spawn", "--enforce", "--name", "A1"})
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-can-spawn --enforce --name A1 with A1 absent from a readable tree did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "ancestor chain unreadable") {
						t.Fatalf("deny message does not name the ancestor chain as unreadable: %s", errBuf.String())
					}
				},
			},
		},
	},
	{
		Command: "spawn-log",
		Axes: []delegationGuardFaultAxis{
			{
				// Same injection as spawn-can-spawn's run-state-unreadable
				// axis; same budget deny path, reached via spawn-log's own
				// call to spawnCanSpawnDecision.
				Name: "run-state-unreadable",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
					if err := os.WriteFile(runStatePath, []byte("{not valid json"), 0644); err != nil {
						t.Fatalf("corrupt spawn-runs.json: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-log with an unreadable run state did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "budget") || !strings.Contains(msg, "unverifiable") {
						t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
					}
				},
			},
			{
				// The same single line of non-pipe-format text as the
				// ledger-corrupt axis on spawn-can-spawn, exercised on the
				// guard that actually consumes budget rather than the
				// advisory one. Named specifically as Queen-parented: a
				// coordinator sentinel makes deriveSpawnDepth's sentinel
				// branch (spawnParentIsRoot) skip the tree lookup entirely and
				// return depth 1 without ever reading spawn-tree.txt, so the
				// budget check inside spawnCanSpawnDecision is the only guard
				// left standing against this fault -- which is exactly why
				// 173-VERIFICATION.md's Gap 1 reproduction worked against a
				// Queen-parented spawn-log call and not against a
				// non-sentinel-parented one (see parent-not-recorded below,
				// which denies via a different guard for a different reason).
				Name: "ledger-corrupt",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
					corruptBytes := []byte("garbage not pipe format")
					if err := os.WriteFile(treePath, corruptBytes, 0644); err != nil {
						t.Fatalf("corrupt spawn-tree.txt: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-log against a corrupted ledger did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "budget") || !strings.Contains(msg, "unverifiable") {
						t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
					}

					after, err := store.ReadFile("spawn-tree.txt")
					if err != nil {
						t.Fatalf("read spawn-tree.txt after refused spawn: %v", err)
					}
					if !bytes.Equal(corruptBytes, after) {
						t.Fatalf("spawn-tree.txt bytes changed after a refused spawn (D-09 requires the refusal to leave no trace):\nbefore=%q\nafter=%q", corruptBytes, after)
					}
				},
			},
			{
				// A directory obstructing spawn-runs.json's path -- the
				// second route to a free budget 173-VERIFICATION.md's
				// planning phase found, needing no tampering with the ledger
				// at all. spawn-tree.txt is deliberately left absent (a
				// fresh, valid, empty ledger): the point of this axis is that
				// the LEDGER is fine and the RUN RECORD is not. Distinct from
				// the run-state-unreadable axis above: that axis injects
				// unparseable CONTENT (invalid JSON) into spawn-runs.json;
				// this one makes the FILE itself unreadable by putting a
				// directory at its path. Before plan 173-11,
				// loadRunStateLocked's collapsed condition treated a
				// directory-obstructed spawn-runs.json identically to "no
				// runs have ever been recorded" and returned a nil error --
				// this axis is that route, now closed. The sharper variant,
				// where a run genuinely WAS active and its record is then
				// deleted mid-run against a FULL ledger, is proven instead by
				// cmd/spawn_budget_test.go's
				// TestErasingTheRunRecordDoesNotResetTheWholeRunBudget, which
				// this axis's comment names rather than duplicates.
				Name: "run-state-obstructed",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					runStatePath := filepath.Join(store.BasePath(), "spawn-runs.json")
					if err := os.MkdirAll(runStatePath, 0755); err != nil {
						t.Fatalf("create directory at spawn-runs.json path: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs(spawnLogArgs("Queen", "W1", "0"))
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-log against an obstructed run record did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "budget") || !strings.Contains(msg, "unverifiable") {
						t.Fatalf("deny message does not name the budget as unverifiable: %s", errBuf.String())
					}
				},
			},
			{
				// A --parent naming an agent whose spawn-tree.txt line has
				// been corrupted, and which is not a coordinator sentinel,
				// reaches deriveSpawnDepth's unknown-parent branch. D-09
				// requires the refusal to leave no trace, so this axis also
				// asserts the tree's bytes are unchanged.
				Name: "parent-not-recorded",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf

					runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "A1", "0"))

					treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
					if err := os.WriteFile(treePath, []byte("not-a-timestamp|Queen|builder|A1|t|not-a-number|spawned\n"), 0644); err != nil {
						t.Fatalf("corrupt spawn-tree.txt: %v", err)
					}

					before, err := store.ReadFile("spawn-tree.txt")
					if err != nil {
						t.Fatalf("read spawn-tree.txt before attempt: %v", err)
					}

					buf.Reset()
					errBuf.Reset()
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs(spawnLogArgs("A1", "C1", "0"))
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-log naming a dropped parent did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
					}
					env := parseEnvelope(t, errBuf.String())
					if env["ok"] != false {
						t.Fatalf("expected ok:false, got: %s", errBuf.String())
					}
					msg, _ := env["error"].(string)
					if !strings.Contains(msg, "unknown parent") {
						t.Fatalf("deny message does not name the unknown parent: %s", errBuf.String())
					}

					after, err := store.ReadFile("spawn-tree.txt")
					if err != nil {
						t.Fatalf("read spawn-tree.txt after refused attempt: %v", err)
					}
					if !bytes.Equal(before, after) {
						t.Fatalf("spawn-tree.txt bytes changed after a refused spawn (D-09 requires the refusal to leave no trace):\nbefore=%q\nafter=%q", before, after)
					}
				},
			},
		},
	},
	{
		Command: "spawn-can-spawn-swarm",
		Axes: []delegationGuardFaultAxis{
			{
				// The D-19 named defect this plan's tests above already
				// prove directly; re-asserted here so the cross-guard table
				// is complete. spawn-can-spawn-swarm is the ONLY guard in
				// this table exercised against COLONY_STATE.json.
				Name: "colony-state-unreadable",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

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
						t.Fatalf("spawn-can-spawn-swarm with unreadable colony state did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
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
						t.Fatalf("deny message does not name the unverifiable budget: %s", errBuf.String())
					}
				},
			},
			{
				// This guard needs no production change to close this axis:
				// its parseErr branch (cmd/internal_cmds.go, ahead of the
				// os.Stat check below) has denied on a non-nil Parse() error
				// since this guard's own D-19 fix -- it was simply waiting for
				// an error the parser could not yet produce. Plan 173-11 gave
				// it one. This is deliberately NOT the directory-at-path fault
				// the spawn-tree-unreadable axis below uses -- that axis
				// proves the os.Stat route, this one proves Parse() itself
				// rejecting a present-but-unparseable regular file, and the
				// whole point of this gap closure is that the two are
				// different faults, not the same one reached two ways.
				Name: "ledger-corrupt",
				Verify: func(t *testing.T) {
					saveGlobals(t)
					resetRootCmd(t)

					s, tmpDir := newTestStore(t)
					defer os.RemoveAll(tmpDir)
					store = s

					writeValidColonyState(t, store)

					treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
					if err := os.WriteFile(treePath, []byte("garbage not pipe format"), 0644); err != nil {
						t.Fatalf("corrupt spawn-tree.txt: %v", err)
					}

					var buf, errBuf bytes.Buffer
					stdout = &buf
					stderr = &errBuf
					renderedCommandExitCode.Store(0)
					rootCmd.SetArgs([]string{"spawn-can-spawn-swarm"})
					_ = rootCmd.Execute()

					if code := int(renderedCommandExitCode.Load()); code == 0 {
						t.Fatalf("spawn-can-spawn-swarm against a corrupted ledger did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
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
						t.Fatalf("deny message does not name the unverifiable budget: %s", errBuf.String())
					}
				},
			},
			{
				// A directory at spawn-tree.txt's path is a real input this
				// guard's cmd/internal_cmds.go os.Stat check rejects ahead of
				// calling Parse().
				//
				// Corrected 2026-08-13 (plan 173-11 gave Parse() a real error
				// return): the directory route and a corrupted-content route
				// are now two distinct faults, both exercised in this table --
				// this axis proves the directory route; the "ledger-corrupt"
				// axis above proves the content route, which Parse() would
				// have accepted as an empty tree before 173-11.
				Name: "spawn-tree-unreadable",
				Verify: func(t *testing.T) {
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
						t.Fatalf("spawn-can-spawn-swarm with unreadable spawn tree did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
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
						t.Fatalf("deny message does not name the unverifiable budget: %s", errBuf.String())
					}
				},
			},
		},
	},
	{
		Command: "hook-pre-tool-use",
		Axes: []delegationGuardFaultAxis{
			{
				// This guard reads no file at all -- its unresolvable input
				// is its own stdin payload. agent_type "general-purpose" on
				// a subagent-originated dispatch (agent_id present) is
				// indistinguishable, on that field alone, from a first-tier
				// worker following .aether/workers.md's documented
				// general-purpose fallback (173-HOOK-FINDINGS.md), so
				// hookSpawnDenyReason denies rather than guessing. Plan 07
				// already proves this in
				// TestHookPreToolUseDeniesUnresolvedRequesterDepth; this axis
				// re-asserts it so the cross-guard table is complete.
				Name: "requester-identity-unresolvable",
				Verify: func(t *testing.T) {
					saveGlobalsCmd(t)
					resetRootCmd(t)

					var buf bytes.Buffer
					stdout = &buf
					var errBuf bytes.Buffer
					stderr = &errBuf

					_, tmpDir := newTestStoreCmd(t)
					defer os.RemoveAll(tmpDir)

					setHookStdin(t, `{"session_id":"sess_1","agent_id":"ae93ff782863d564f","agent_type":"general-purpose","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Dispatch leaf agent","prompt":"Reply with the single word: leaf","subagent_type":"general-purpose"}}`)

					rootCmd.SetArgs([]string{"hook-pre-tool-use"})
					if err := rootCmd.Execute(); err != nil {
						t.Fatalf("hook-pre-tool-use returned error: %v", err)
					}

					var result map[string]interface{}
					if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
						t.Fatalf("unmarshal hook output: %v (%q)", err, buf.String())
					}
					if result["decision"] != "block" {
						t.Fatalf("decision = %v, want block", result["decision"])
					}
					reason, _ := result["reason"].(string)
					if reason == "" {
						t.Fatalf("hook block carries an empty reason -- a silent denial is almost as bad as a silent allow")
					}
				},
			},
		},
	},
}

// delegationGuardTableAllValidControl is the paired control D-19's table
// requires: without it, every deny assertion above could also be satisfied
// by a guard that denies unconditionally, which is a different and much
// worse defect than the one this table exists to catch.
func delegationGuardTableAllValidControl(t *testing.T) {
	t.Helper()

	t.Run("control/spawn-can-spawn allows when everything is valid", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf
		renderedCommandExitCode.Store(0)
		rootCmd.SetArgs([]string{"spawn-can-spawn", "--enforce"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("spawn-can-spawn --enforce with valid inputs returned error: %v\nstderr: %s", err, errBuf.String())
		}
		if code := int(renderedCommandExitCode.Load()); code != 0 {
			t.Fatalf("spawn-can-spawn --enforce with valid inputs exited %d: %s", code, errBuf.String())
		}
	})

	t.Run("control/spawn-log allows when everything is valid", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf
		result := runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "W1", "0"))
		if result["recorded"] != true {
			t.Fatalf("spawn-log with valid inputs was not recorded: %v", result)
		}
	})

	t.Run("control/spawn-can-spawn-swarm allows when everything is valid", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		writeValidColonyState(t, store)

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf
		renderedCommandExitCode.Store(0)
		rootCmd.SetArgs([]string{"spawn-can-spawn-swarm"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("spawn-can-spawn-swarm with valid inputs returned error: %v\nstderr: %s", err, errBuf.String())
		}
		if code := int(renderedCommandExitCode.Load()); code != 0 {
			t.Fatalf("spawn-can-spawn-swarm with valid inputs exited %d: %s", code, errBuf.String())
		}
		env := parseEnvelope(t, buf.String())
		result, _ := env["result"].(map[string]interface{})
		if result == nil || result["can_spawn"] != true {
			t.Fatalf("spawn-can-spawn-swarm with valid inputs did not allow: %s", buf.String())
		}
	})

	t.Run("control/hook-pre-tool-use allows when everything is valid", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf
		var errBuf bytes.Buffer
		stderr = &errBuf

		_, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)

		setHookStdin(t, `{"session_id":"sess_1","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Nested dispatch experiment level 1","subagent_type":"general-purpose"}}`)

		rootCmd.SetArgs([]string{"hook-pre-tool-use"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("hook-pre-tool-use with valid inputs returned error: %v", err)
		}
		if strings.TrimSpace(buf.String()) != "" {
			t.Fatalf("hook-pre-tool-use with valid inputs was blocked: %s", buf.String())
		}
	})
}

// TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs is the D-19
// cross-guard proof: every exercisable (guard, axis) pair in
// delegationGuardFaultTable must deny with a non-empty reason, and the paired
// all-valid control must allow on every guard -- in one command, rather than
// one test per guard that a newly added guard could silently skip.
func TestEveryDelegationGuardFailsClosedOnItsOwnUnreadableInputs(t *testing.T) {
	for _, entry := range delegationGuardFaultTable {
		for _, axis := range entry.Axes {
			if axis.NotExercisable != "" {
				t.Logf("%s/%s: not exercisable -- %s", entry.Command, axis.Name, axis.NotExercisable)
				continue
			}
			if axis.Verify == nil {
				t.Fatalf("%s/%s: exercisable axis has a nil Verify func", entry.Command, axis.Name)
			}
			t.Run(entry.Command+"/"+axis.Name, axis.Verify)
		}
	}

	delegationGuardTableAllValidControl(t)
}

// delegationGuardColonyStateSourceFiles is the exact set of files
// TestDelegationGuardTableCoversEveryGuardCommand's narrowing assertion
// parses -- the three files this plan's <interfaces> section established
// carry the delegation decision (spawn.go's spawnCanSpawnDecision,
// spawn_budget.go's budget check, spawn_ancestor.go's ancestor-cycle check).
var delegationGuardColonyStateSourceFiles = []string{"spawn.go", "spawn_budget.go", "spawn_ancestor.go"}

// TestDelegationGuardTableCoversEveryGuardCommand is the guard on the
// narrowing itself (T-173-54/T-173-58/T-173-59): it enumerates the REGISTERED
// commands rather than trusting a hardcoded list, so a future fifth
// delegation guard must join delegationGuardFaultTable or fail the build; it
// fails a guard with zero fault axes, only non-exercisable ones, or a
// non-exercisable axis with an empty reason, so the fail-closed guarantee
// cannot decay to nothing while this test stays green; and it parses the
// three files the decision chain actually lives in with go/parser, failing
// if any STRING LITERAL (comments excluded, deliberately -- this plan's own
// residue document names COLONY_STATE.json in prose) is or contains
// "COLONY_STATE.json", so a guard that starts reading colony state cannot
// silently make 173-RESIDUE.md's narrowed claim false.
func TestDelegationGuardTableCoversEveryGuardCommand(t *testing.T) {
	// 1. Enumerate the registered commands this table must cover.
	var registered []string
	for _, c := range rootCmd.Commands() {
		name := c.Name()
		if name == "spawn-log" || name == "hook-pre-tool-use" || strings.HasPrefix(name, "spawn-can-spawn") {
			registered = append(registered, name)
		}
	}
	sort.Strings(registered)

	if len(registered) < 4 {
		t.Fatalf("enumeration over rootCmd.Commands() found only %d guard command(s) (%v) -- expected at least 4 (spawn-can-spawn, spawn-can-spawn-swarm, spawn-log, hook-pre-tool-use); a walk that silently finds nothing would pass forever, so this is treated as a fatal enumeration failure", len(registered), registered)
	}

	tableCommands := make(map[string]bool, len(delegationGuardFaultTable))
	for _, e := range delegationGuardFaultTable {
		tableCommands[e.Command] = true
	}

	for _, name := range registered {
		if !tableCommands[name] {
			t.Errorf("registered command %q (matched spawn-can-spawn*/spawn-log/hook-pre-tool-use) has no entry in delegationGuardFaultTable -- a new delegation guard must join the table or fail the build", name)
		}
	}

	// 2. No guard may decay to zero exercisable axes, or to a
	// non-exercisable axis with no stated reason.
	for _, e := range delegationGuardFaultTable {
		if len(e.Axes) == 0 {
			t.Errorf("guard %q declares zero fault axes -- every registered delegation guard must be exercised against at least one fault it genuinely depends on", e.Command)
			continue
		}
		hasExercisable := false
		for _, axis := range e.Axes {
			if axis.NotExercisable == "" {
				hasExercisable = true
				continue
			}
			if strings.TrimSpace(axis.NotExercisable) == "" {
				t.Errorf("guard %q axis %q is marked non-exercisable with an empty reason -- a skipped axis must never be silent", e.Command, axis.Name)
			}
		}
		if !hasExercisable {
			t.Errorf("guard %q declares only non-exercisable fault axes -- the fail-closed guarantee has decayed to nothing while this test stays green", e.Command)
		}
	}

	// 3. The narrowing itself must still be accurate: no delegation-decision
	// source file may contain a COLONY_STATE.json string literal. AST-based,
	// not grep, so a comment (including this plan's own residue prose) can
	// never trip it.
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	filesInspected := 0
	for _, f := range delegationGuardColonyStateSourceFiles {
		path := filepath.Join(repoRoot, "cmd", f)
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		filesInspected++

		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			val, unquoteErr := strconv.Unquote(lit.Value)
			if unquoteErr != nil {
				return true
			}
			if strings.Contains(val, "COLONY_STATE.json") {
				t.Errorf("cmd/%s:%d contains a COLONY_STATE.json string literal (%q) -- a delegation guard has started reading colony state, so this table's axes and 173-RESIDUE.md's narrowed claim must both be widened to match", f, fset.Position(lit.Pos()).Line, val)
			}
			return true
		})
	}
	if filesInspected != len(delegationGuardColonyStateSourceFiles) {
		t.Fatalf("inspected %d file(s), expected exactly %d -- a shrunk walk would silently narrow this check", filesInspected, len(delegationGuardColonyStateSourceFiles))
	}
}
