package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
)

// Plan 203-06 Task 1 (BIO-02): five new admission dimensions -- parent
// authority, permission, path containment, cost, and duplicate intent --
// layered onto the existing spawnCanSpawnDecision chokepoint, each denying
// by name and each failing closed, applying ONLY to a real recruitment
// (spawnOriginRecruit) per recruitmentAdmissionChecks' declared table.

// recruitmentAdmissionSeedParent writes one real, non-terminal spawn-tree
// entry for "A1" under the coordinator sentinel "Queen" and returns the
// input skeleton every subtest below starts from -- built the way the
// runtime itself builds one (a real RecordSpawn call), never a typed
// literal pretending to be a parsed ledger line.
func recruitmentAdmissionSeedParent(t *testing.T, st *agent.SpawnTree) spawnDecisionInput {
	t.Helper()
	if err := st.RecordSpawn("Queen", "builder", "A1", "help with the login form", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}
	return spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 "fix the pagination bug",
		Origin:               spawnOriginRecruit,
		Workspace:            "",
		CostSlots:            1,
		IntentID:             "intent-under-test",
		AttemptID:            "attempt-under-test",
	}
}

// --- Parent authority ---

func TestRecruitmentAdmissionParentDeniesUnknownParent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	in := spawnDecisionInput{
		RequesterName: "Ghost",
		Origin:        spawnOriginRecruit,
		Caste:         "builder",
		Task:          "do a thing",
		Workspace:     tmpDir,
		CostSlots:     1,
	}
	reason := recruitmentParentAuthorityReason(in)
	if reason == "" {
		t.Fatal("expected a deny reason for a parent with no recorded spawn entry")
	}
	if !strings.Contains(reason, "Ghost") {
		t.Fatalf("deny reason does not name the parent %q: %s", "Ghost", reason)
	}
}

func TestRecruitmentAdmissionParentDeniesTerminalParent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	in := recruitmentAdmissionSeedParent(t, st)
	if err := st.UpdateStatus("A1", "completed", "done"); err != nil {
		t.Fatalf("mark parent completed: %v", err)
	}

	reason := recruitmentParentAuthorityReason(in)
	if reason == "" {
		t.Fatal("expected a deny reason for a terminal-status parent")
	}
	if !strings.Contains(reason, "completed") {
		t.Fatalf("deny reason does not name the terminal status: %s", reason)
	}
}

func TestRecruitmentAdmissionParentAllowsCoordinatorSentinel(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)

	reason := recruitmentParentAuthorityReason(spawnDecisionInput{RequesterName: "Queen", Origin: spawnOriginRecruit})
	if reason != "" {
		t.Fatalf("expected the coordinator sentinel to be allowed, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionParentAllowsActiveParent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	in := recruitmentAdmissionSeedParent(t, st)

	if reason := recruitmentParentAuthorityReason(in); reason != "" {
		t.Fatalf("expected an active, recorded parent to be allowed, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionParentFailsClosedOnUnreadableLedger(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath, []byte("garbage not pipe format"), 0644); err != nil {
		t.Fatalf("corrupt spawn-tree.txt: %v", err)
	}

	reason := recruitmentParentAuthorityReason(spawnDecisionInput{RequesterName: "A1", Origin: spawnOriginRecruit})
	if reason == "" {
		t.Fatal("expected an unreadable ledger to deny (D-19 fail-closed), got allow")
	}
	if !strings.Contains(reason, "unreadable") {
		t.Fatalf("deny reason does not name the ledger as unreadable: %q", reason)
	}
}

// --- Permission ---

func TestRecruitmentAdmissionPermissionDeniesReadOnlyCaste(t *testing.T) {
	reason := recruitmentPermissionReason(spawnDecisionInput{Caste: "includer"})
	if reason == "" {
		t.Fatal("expected a read-only caste to be denied")
	}
	if !strings.Contains(reason, "includer") || !strings.Contains(reason, "repository_read_only") {
		t.Fatalf("deny reason does not name both the caste and the profile: %q", reason)
	}
}

func TestRecruitmentAdmissionPermissionAllowsWriteCaste(t *testing.T) {
	reason := recruitmentPermissionReason(spawnDecisionInput{Caste: "builder"})
	if reason != "" {
		t.Fatalf("expected a workspace-write caste to be allowed, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionPermissionFailsClosedOnMissingCaste(t *testing.T) {
	reason := recruitmentPermissionReason(spawnDecisionInput{Caste: "   "})
	if reason == "" {
		t.Fatal("expected a request with no caste to be denied rather than silently resolving a permission profile")
	}
}

// --- Path containment ---

func TestRecruitmentAdmissionPathDeniesOutsideColonyRoot(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	outside := t.TempDir()
	reason := recruitmentPathReason(spawnDecisionInput{Workspace: outside})
	if reason == "" {
		t.Fatalf("expected a workspace outside the colony root (%q vs colony root %q) to be denied", outside, tmpDir)
	}
	if !strings.Contains(reason, outside) {
		t.Fatalf("deny reason does not name the offending path %q: %s", outside, reason)
	}
}

func TestRecruitmentAdmissionPathAllowsInsideColonyRoot(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	inside := filepath.Join(tmpDir, "workspace")
	if err := os.MkdirAll(inside, 0755); err != nil {
		t.Fatalf("create workspace dir: %v", err)
	}
	if reason := recruitmentPathReason(spawnDecisionInput{Workspace: inside}); reason != "" {
		t.Fatalf("expected a workspace inside the colony root to be allowed, got deny: %q", reason)
	}
}

// TestRecruitmentAdmissionPathDeniesSymlinkedSibling proves the containment
// boundary is symlink-aware, not a lexical prefix match: a symlink whose
// target directory NAME shares the colony root's own name as a prefix (a
// classic "<root>-evil" bypass) must still be refused once resolved.
func TestRecruitmentAdmissionPathDeniesSymlinkedSibling(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	evilSibling := tmpDir + "-evil"
	if err := os.MkdirAll(evilSibling, 0755); err != nil {
		t.Fatalf("create sibling dir: %v", err)
	}
	defer os.RemoveAll(evilSibling)

	link := filepath.Join(tmpDir, "escape-link")
	if err := os.Symlink(evilSibling, link); err != nil {
		t.Skipf("symlinks unsupported in this environment: %v", err)
	}

	reason := recruitmentPathReason(spawnDecisionInput{Workspace: link})
	if reason == "" {
		t.Fatal("expected a symlink escaping to a name-prefix sibling to be denied")
	}
}

// --- Cost ---

// recruitmentAdmissionFillBudget records n live, whole-run-counted spawn
// entries under the coordinator sentinel so spawnTreeBudgetState() reports a
// real, non-zero Consumed count -- built the way the runtime derives it
// (real RecordSpawn calls), never a typed literal.
func recruitmentAdmissionFillBudget(t *testing.T, st *agent.SpawnTree, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("Filler-%d", i)
		if err := st.RecordSpawn("Queen", "builder", name, "filler work", 1); err != nil {
			t.Fatalf("seed filler spawn %d: %v", i, err)
		}
	}
}

func TestRecruitmentAdmissionCostDeniesWhenRequestExceedsRemaining(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	// spawnTreeBudgetMax is 20; leave exactly 2 remaining.
	recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax-2)

	reason := recruitmentCostReason(spawnDecisionInput{CostSlots: 5})
	if reason == "" {
		t.Fatal("expected a cost request exceeding the remaining budget to be denied")
	}
	for _, want := range []string{"5", "2", fmt.Sprintf("%d", spawnTreeBudgetMax)} {
		if !strings.Contains(reason, want) {
			t.Fatalf("deny reason %q does not name %q (requested/remaining/max)", reason, want)
		}
	}
}

func TestRecruitmentAdmissionCostAllowsWhenWithinRemaining(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	if reason := recruitmentCostReason(spawnDecisionInput{CostSlots: 1}); reason != "" {
		t.Fatalf("expected a cost request well within budget to be allowed, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionCostFailsClosedOnUnverifiableBudget(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	treePath := filepath.Join(store.BasePath(), "spawn-tree.txt")
	if err := os.WriteFile(treePath, []byte("garbage not pipe format"), 0644); err != nil {
		t.Fatalf("corrupt spawn-tree.txt: %v", err)
	}

	reason := recruitmentCostReason(spawnDecisionInput{CostSlots: 1})
	if reason == "" {
		t.Fatal("expected an unverifiable budget to deny (D-19 fail-closed), got allow")
	}
	if !strings.Contains(reason, "unverifiable") {
		t.Fatalf("deny reason does not name the budget as unverifiable: %q", reason)
	}
}

// --- Duplicate intent ---

func TestRecruitmentAdmissionDuplicateDeniesPendingSameSubtree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "top-level task", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}

	pending := recruitmentIntentRecord{
		Intent: recruitmentIntent{
			SchemaVersion: recruitmentSchemaVersion,
			IntentID:      "pending-1",
			ParentName:    "A1",
			Caste:         "builder",
			Objective:     "Fix the login form.",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if _, err := recordRecruitmentIntent(pending); err != nil {
		t.Fatalf("record pending intent: %v", err)
	}

	reason := recruitmentDuplicateReason(spawnDecisionInput{
		RequesterName: "A1",
		Origin:        spawnOriginRecruit,
		Caste:         "builder",
		Task:          "fix the login form", // normalizes to the same text as "Fix the login form."
		IntentID:      "new-intent",
	})
	if reason == "" {
		t.Fatal("expected a pending duplicate in the same subtree to be denied")
	}
	if !strings.Contains(reason, "pending-1") {
		t.Fatalf("deny reason does not name the pending intent identifier: %q", reason)
	}
}

func TestRecruitmentAdmissionDuplicateAllowsDecidedIntent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "top-level task", 1); err != nil {
		t.Fatalf("seed parent: %v", err)
	}

	decided := recruitmentIntentRecord{
		Intent: recruitmentIntent{
			SchemaVersion: recruitmentSchemaVersion,
			IntentID:      "decided-1",
			ParentName:    "A1",
			Caste:         "builder",
			Objective:     "fix the login form",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if _, err := recordRecruitmentIntent(decided); err != nil {
		t.Fatalf("record intent: %v", err)
	}
	if _, err := recordRecruitmentDecision("decided-1", recruitmentDecisionResult{Allowed: true}, "2026-01-01T00:00:01Z"); err != nil {
		t.Fatalf("record decision: %v", err)
	}

	reason := recruitmentDuplicateReason(spawnDecisionInput{
		RequesterName: "A1",
		Origin:        spawnOriginRecruit,
		Caste:         "builder",
		Task:          "fix the login form",
		IntentID:      "new-intent",
	})
	if reason != "" {
		t.Fatalf("expected an already-decided intent to never block a new request, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionDuplicateAllowsDifferentSubtree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "task a", 1); err != nil {
		t.Fatalf("seed A1: %v", err)
	}
	if err := st.RecordSpawn("Queen", "builder", "B1", "task b", 1); err != nil {
		t.Fatalf("seed B1: %v", err)
	}

	pending := recruitmentIntentRecord{
		Intent: recruitmentIntent{
			SchemaVersion: recruitmentSchemaVersion,
			IntentID:      "pending-under-b1",
			ParentName:    "B1",
			Caste:         "builder",
			Objective:     "fix the login form",
		},
		CreatedAt: "2026-01-01T00:00:00Z",
	}
	if _, err := recordRecruitmentIntent(pending); err != nil {
		t.Fatalf("record pending intent: %v", err)
	}

	reason := recruitmentDuplicateReason(spawnDecisionInput{
		RequesterName: "A1",
		Origin:        spawnOriginRecruit,
		Caste:         "builder",
		Task:          "fix the login form",
		IntentID:      "new-intent",
	})
	if reason != "" {
		t.Fatalf("expected a pending intent under an unrelated sibling subtree to never block, got deny: %q", reason)
	}
}

func TestRecruitmentAdmissionDuplicateFailsClosedOnUnreadableIntents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// A regular file where the "recruitment" directory belongs makes
	// os.Stat(".../recruitment/intents.json") fail with ENOTDIR -- a real
	// stat error distinct from "does not exist", which is the fault
	// store.FileExists actually surfaces as an error rather than a plain
	// false. (A directory placed directly at intents.json's own path is
	// NOT sufficient: FileExists reports "not a regular file" as exists=false,
	// err=nil, which reads as an ordinary absent-file allow, not a fault.)
	recruitmentDirPath := filepath.Join(store.BasePath(), "recruitment")
	if err := os.WriteFile(recruitmentDirPath, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("create file blocking the recruitment directory: %v", err)
	}

	reason := recruitmentDuplicateReason(spawnDecisionInput{RequesterName: "Queen", Origin: spawnOriginRecruit, Caste: "builder", Task: "x"})
	if reason == "" {
		t.Fatal("expected an unreadable intents ledger to deny (D-19 fail-closed), got allow")
	}
	if !strings.Contains(reason, "unreadable") {
		t.Fatalf("deny reason does not name the intents ledger as unreadable: %q", reason)
	}
}

// --- Applicability table ---

// TestRecruitmentAdmissionChecksCoverEveryOrigin proves
// recruitmentAdmissionChecks names every origin spawnDecisionOrigins()
// returns, by name, so a future origin added without a row fails here
// rather than silently applying no new checks.
func TestRecruitmentAdmissionChecksCoverEveryOrigin(t *testing.T) {
	for _, origin := range spawnDecisionOrigins() {
		if _, ok := recruitmentAdmissionChecks[origin]; !ok {
			t.Errorf("spawnDecisionOrigins() declares %q, but recruitmentAdmissionChecks has no row for it", origin)
		}
	}
}

// TestRecruitmentAdmissionChecksOnlyApplyToRecruit proves the five new
// dimensions apply to spawn-log and spawn-can-spawn's origins NOT AT ALL --
// applying them there would deny every ordinary spawn on missing data
// rather than skip a check that does not apply.
func TestRecruitmentAdmissionChecksOnlyApplyToRecruit(t *testing.T) {
	newChecks := []string{recruitmentReasonParent, recruitmentReasonPermission, recruitmentReasonPath, recruitmentReasonCost, recruitmentReasonDuplicate}
	for _, origin := range []spawnDecisionOrigin{spawnOriginSpawnLog, spawnOriginSpawnCanSpawn} {
		for _, check := range newChecks {
			if recruitmentCheckApplies(origin, check) {
				t.Errorf("check %q incorrectly applies to origin %q -- only spawnOriginRecruit should carry the five new dimensions", check, origin)
			}
		}
	}
	for _, check := range newChecks {
		if !recruitmentCheckApplies(spawnOriginRecruit, check) {
			t.Errorf("check %q does not apply to spawnOriginRecruit -- every new dimension must apply there", check)
		}
	}
}

// --- Depth override (D-11) ---

func TestRecruitmentAdmissionDepthOverrideDefaultStaysAtTwo(t *testing.T) {
	cap, raised := recruitmentDepthOverride(0)
	if raised {
		t.Fatal("expected no raise for an absent --max-depth flag")
	}
	if cap != spawnMaxDelegationDepth {
		t.Fatalf("expected the default cap %d, got %d", spawnMaxDelegationDepth, cap)
	}

	cap, raised = recruitmentDepthOverride(spawnMaxDelegationDepth)
	if raised {
		t.Fatal("expected no raise when --max-depth equals the existing default")
	}
	if cap != spawnMaxDelegationDepth {
		t.Fatalf("expected the default cap %d, got %d", spawnMaxDelegationDepth, cap)
	}
}

func TestRecruitmentAdmissionDepthOverrideRaisesAboveDefault(t *testing.T) {
	cap, raised := recruitmentDepthOverride(3)
	if !raised {
		t.Fatal("expected a --max-depth above the default to raise the cap")
	}
	if cap != 3 {
		t.Fatalf("expected cap 3, got %d", cap)
	}
}

// TestRecruitmentAdmissionDepthOverrideRecordsOnTheLedger drives the real
// `recruit` command end to end at a depth the default cap of 2 would refuse,
// and asserts --max-depth 3 both admits it AND leaves a readable record of
// the raise on the resulting spawn-tree entry -- ruling (c)'s "always
// visible after the fact" requirement.
func TestRecruitmentAdmissionDepthOverrideRecordsOnTheLedger(t *testing.T) {
	if os.Getenv("AETHER_RECRUIT_CHILD") == "1" {
		fmt.Println("recruited-child-ok")
		return
	}

	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	// Queen -> A1 (depth 1) -> A2 (depth 2). A2 recruiting would be depth 3,
	// past the default cap of 2, so this is genuinely exercising the raise.
	if err := st.RecordSpawn("Queen", "builder", "A1", "top task", 1); err != nil {
		t.Fatalf("seed A1: %v", err)
	}
	if err := st.RecordSpawn("A1", "builder", "A2", "sub task", 2); err != nil {
		t.Fatalf("seed A2: %v", err)
	}

	t.Setenv("AETHER_RECRUIT_BINARY", os.Args[0])
	t.Setenv("AETHER_RECRUIT_ARGS", "-test.run=^TestRecruitmentAdmissionDepthOverrideRecordsOnTheLedger$")
	t.Setenv("AETHER_RECRUIT_TIMEOUT", "30s")

	rootCmd.SetArgs([]string{
		"recruit",
		"--parent", "A2",
		"--caste", "builder",
		"--objective", "help finish this",
		"--reason", "over the depth cap",
		"--max-depth", "3",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("recruit command returned an error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result, _ := env["result"].(map[string]interface{})
	if result == nil || result["admitted"] != true {
		t.Fatalf("expected --max-depth 3 to admit a depth-3 recruitment: %s", buf.String())
	}

	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	var found bool
	for _, e := range entries {
		if e.ParentName == "A2" && strings.Contains(e.Task, "depth-raise") && strings.Contains(e.Task, "max-depth=3") {
			found = true
		}
	}
	if !found {
		t.Fatalf("no spawn-tree entry under A2 records the depth-raise: entries=%+v", entries)
	}
}

// --- Existing checks unchanged ---

// TestRecruitmentAdmissionExistingChecksUnchanged asserts the depth, budget,
// and ancestor-cycle detail sentences are byte-identical to their current
// values after this plan's extension -- must_haves' explicit requirement.
func TestRecruitmentAdmissionExistingChecksUnchanged(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Depth.
	depthResult := spawnCanSpawnDecision(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       2,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 "x",
	})
	wantDepth := "A1 is at depth 2; a helper spawned from here would be depth 3, past the cap of 2"
	if depthResult.Reason != "depth" || depthResult.Detail != wantDepth {
		t.Fatalf("depth detail changed: reason=%q detail=%q, want reason=depth detail=%q", depthResult.Reason, depthResult.Detail, wantDepth)
	}

	// Ancestor-cycle (reuses the established fixture pattern).
	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "B1", "fix the login form", 1); err != nil {
		t.Fatalf("seed B1: %v", err)
	}
	cycleResult := spawnCanSpawnDecision(spawnDecisionInput{
		RequesterName:        "B1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 "fix the login form",
	})
	wantCycle := "builder was already asked to do this task by B1 at depth 1; spawning it again would repeat work already in progress above"
	if cycleResult.Reason != "ancestor-cycle" || cycleResult.Detail != wantCycle {
		t.Fatalf("ancestor-cycle detail changed: reason=%q detail=%q, want reason=ancestor-cycle detail=%q", cycleResult.Reason, cycleResult.Detail, wantCycle)
	}

	// Budget -- an isolated fresh store and an explicit run window, so this
	// assertion exercises the plain "in this run" sentence rather than the
	// whole-ledger fallback's differently-worded one (both are real,
	// separately-tested paths; this is specifically the ordinary one).
	budgetStore, budgetTmpDir := newTestStore(t)
	defer os.RemoveAll(budgetTmpDir)
	store = budgetStore
	budgetSt := agent.NewSpawnTree(store, "spawn-tree.txt")
	if _, err := budgetSt.BeginRun("test-run", time.Time{}); err != nil {
		t.Fatalf("begin run: %v", err)
	}
	recruitmentAdmissionFillBudget(t, budgetSt, spawnTreeBudgetMax)
	budgetResult := spawnCanSpawnDecision(spawnDecisionInput{RequesterName: "C1", RequesterDepth: 0, Caste: "builder", Task: "y"})
	wantBudget := fmt.Sprintf("whole-run helper budget exhausted: %d of %d helpers already spawned in this run; C1 may not spawn another", spawnTreeBudgetMax, spawnTreeBudgetMax)
	if budgetResult.Reason != "budget" || budgetResult.Detail != wantBudget {
		t.Fatalf("budget detail changed: reason=%q detail=%q, want reason=budget detail=%q", budgetResult.Reason, budgetResult.Detail, wantBudget)
	}
}

// Plan 203-06 Task 2 (BIO-02/BIO-04): the admitted child is recorded
// atomically before any process starts, and a failed manifest write denies
// launch rather than proceeding unrecorded.

// TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure forces the
// manifest write to fail (a regular file blocking the "recruitment"
// directory, the same real stat-error fault used elsewhere in this file)
// and asserts the whole recruitment is refused, naming the write error, and
// that no spawn-tree entry and no recruitment result were ever created --
// the launch never happened.
func TestRecruitmentManifestAmendmentDeniesLaunchOnWriteFailure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "top task", 1); err != nil {
		t.Fatalf("seed A1: %v", err)
	}
	before, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt before attempt: %v", err)
	}

	// Block ONLY manifest.json's own path with a directory -- "recruitment/"
	// itself stays a real directory so the sibling recruitment/intents.json
	// write (recordRecruitmentIntent, which runs earlier in recruitCmd)
	// still succeeds; this fault is scoped to amendRecruitmentManifest's own
	// write specifically.
	manifestDirPath := filepath.Join(store.BasePath(), filepath.FromSlash(recruitmentManifestPath))
	if err := os.MkdirAll(manifestDirPath, 0755); err != nil {
		t.Fatalf("create directory at manifest path: %v", err)
	}

	rootCmd.SetArgs([]string{
		"recruit",
		"--parent", "A1",
		"--caste", "builder",
		"--objective", "help with x",
		"--reason", "stuck on y",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("recruit command returned an error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result, _ := env["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("expected a result object: %s", buf.String())
	}
	if result["admitted"] != false {
		t.Fatalf("expected a manifest write failure to refuse the recruitment, got: %s", buf.String())
	}
	if reason, _ := result["reason"].(string); reason != recruitmentReasonUnresolved {
		t.Fatalf("expected reason %q, got %q: %s", recruitmentReasonUnresolved, reason, buf.String())
	}
	detail, _ := result["detail"].(string)
	if detail == "" {
		t.Fatalf("expected a non-empty detail naming the write failure: %s", buf.String())
	}

	after, err := store.ReadFile("spawn-tree.txt")
	if err != nil {
		t.Fatalf("read spawn-tree.txt after refused attempt: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("spawn-tree.txt bytes changed after a launch denied by a failed manifest write:\nbefore=%q\nafter=%q", before, after)
	}

	if exists, _ := store.FileExists(recruitmentResultsPath); exists {
		t.Fatal("expected no recruitment result to exist -- the child never launched")
	}
}

// TestRecruitmentManifestAmendmentAtomicWithSpawnTreeEntry drives an
// ADMITTED recruitment end to end and asserts exactly one manifest entry and
// exactly one new spawn-tree entry exist afterward.
func TestRecruitmentManifestAmendmentAtomicWithSpawnTreeEntry(t *testing.T) {
	if os.Getenv("AETHER_RECRUIT_CHILD") == "1" {
		fmt.Println("recruited-child-ok")
		return
	}

	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "A1", "top task", 1); err != nil {
		t.Fatalf("seed A1: %v", err)
	}

	t.Setenv("AETHER_RECRUIT_BINARY", os.Args[0])
	t.Setenv("AETHER_RECRUIT_ARGS", "-test.run=^TestRecruitmentManifestAmendmentAtomicWithSpawnTreeEntry$")
	t.Setenv("AETHER_RECRUIT_TIMEOUT", "30s")

	rootCmd.SetArgs([]string{
		"recruit",
		"--parent", "A1",
		"--caste", "builder",
		"--objective", "help with x",
		"--reason", "stuck on y",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("recruit command returned an error: %v", err)
	}
	env := parseEnvelope(t, buf.String())
	result, _ := env["result"].(map[string]interface{})
	if result == nil || result["admitted"] != true {
		t.Fatalf("expected an admitted recruitment: %s", buf.String())
	}
	childName, _ := result["child"].(string)

	var manifestFile recruitmentManifestFile
	if err := store.LoadJSON(recruitmentManifestPath, &manifestFile); err != nil {
		t.Fatalf("load recruitment manifest: %v", err)
	}
	if len(manifestFile.Entries) != 1 {
		t.Fatalf("expected exactly one manifest entry, got %d", len(manifestFile.Entries))
	}
	if manifestFile.Entries[0].ChildName != childName {
		t.Fatalf("manifest entry child name = %q, want %q", manifestFile.Entries[0].ChildName, childName)
	}

	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	found := 0
	for _, e := range entries {
		if e.AgentName == childName {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("expected exactly one spawn-tree entry for %q, found %d", childName, found)
	}
}

// TestRecruitmentManifestAmendmentStateTransitionsAdmittedToDispatched
// proves the amendment's State reads "admitted" before dispatch and
// "dispatched" after, by reading the file at both points -- unit-level,
// directly against amendRecruitmentManifest/recruitmentManifestEntryByIntentID,
// which is the exact sequence recruitCmd's own RunE performs.
func TestRecruitmentManifestAmendmentStateTransitionsAdmittedToDispatched(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const intentID = "intent-state-transition"
	if _, err := amendRecruitmentManifest(recruitmentManifestRecord{
		IntentID:  intentID,
		ChildName: "C1",
		State:     recruitmentManifestStateAdmitted,
	}); err != nil {
		t.Fatalf("amend (admitted): %v", err)
	}

	entry, err := recruitmentManifestEntryByIntentID(intentID)
	if err != nil {
		t.Fatalf("read manifest entry: %v", err)
	}
	if entry.State != recruitmentManifestStateAdmitted {
		t.Fatalf("expected state %q before dispatch, got %q", recruitmentManifestStateAdmitted, entry.State)
	}

	if _, err := amendRecruitmentManifest(recruitmentManifestRecord{
		IntentID: intentID,
		State:    recruitmentManifestStateDispatched,
	}); err != nil {
		t.Fatalf("amend (dispatched): %v", err)
	}

	entry, err = recruitmentManifestEntryByIntentID(intentID)
	if err != nil {
		t.Fatalf("read manifest entry: %v", err)
	}
	if entry.State != recruitmentManifestStateDispatched {
		t.Fatalf("expected state %q after dispatch, got %q", recruitmentManifestStateDispatched, entry.State)
	}
	// The child identity recorded at admission time must survive the later
	// state-only update -- an upsert that only touches State/AdapterKind.
	if entry.ChildName != "C1" {
		t.Fatalf("expected ChildName to survive the state transition, got %q", entry.ChildName)
	}
}

// TestRecruitmentManifestAmendmentUnknownSchemaVersion proves an entry
// written under a schema version this runtime does not recognise is
// reported as unknown rather than a fabricated zero-value read.
func TestRecruitmentManifestAmendmentUnknownSchemaVersion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	file := recruitmentManifestFile{Entries: []recruitmentManifestRecord{
		{SchemaVersion: "recruitment-manifest/v999", IntentID: "old-intent", State: "admitted"},
	}}
	if err := store.SaveJSON(recruitmentManifestPath, file); err != nil {
		t.Fatalf("seed manifest file: %v", err)
	}

	_, err := recruitmentManifestEntryByIntentID("old-intent")
	if err == nil {
		t.Fatal("expected an unrecognised schema version to be reported as an error, got a successful read")
	}
	if !errors.Is(err, errRecruitmentManifestUnknownSchema) {
		t.Fatalf("expected errRecruitmentManifestUnknownSchema, got: %v", err)
	}
}

// Plan 203-06 Task 3: lock the single admission authority so a second gate
// cannot be added without a test naming it.

// recruitmentAdmissionAuthorityAllowlist is the exact, exhaustive set of
// functions in cmd/ permitted to return spawnDecisionResult or
// recruitmentDecisionResult -- the ONE admission chokepoint
// (spawnCanSpawnDecision, a package-level var), its five declared check
// helpers, and two functions that return the SAME struct shape for a
// genuinely different concern (intent-shape validation and adapter choice,
// neither of which decides whether a spawn or recruitment may proceed).
// Adding a name here is the explicit, reviewed act TestOneAdmissionAuthority
// exists to force.
var recruitmentAdmissionAuthorityAllowlist = map[string]bool{
	"spawnCanSpawnDecision":            true, // the one chokepoint (package-level var)
	"spawnTreeBudgetReason":            true,
	"spawnAncestorCycleReason":         true,
	"recruitmentParentAuthorityReason": true,
	"recruitmentPermissionReason":      true,
	"recruitmentPathReason":            true,
	"recruitmentCostReason":            true,
	"recruitmentDuplicateReason":       true,
	"validateRecruitmentIntent":        true, // intent-shape validation, not spawn admission
	"chooseRecruitmentAdapter":         true, // adapter choice, not admission
}

// recruitmentAdmissionAuthorityResultTypes names the two result shapes a
// second admission authority could plausibly reuse.
var recruitmentAdmissionAuthorityResultTypes = map[string]bool{
	"spawnDecisionResult":       true,
	"recruitmentDecisionResult": true,
}

// TestOneAdmissionAuthority walks every non-test .go file in cmd/ and fails
// by name for any function -- a regular declaration, or a package-level var
// assigned a func literal (the shape spawnCanSpawnDecision itself uses) --
// that returns spawnDecisionResult or recruitmentDecisionResult and is not
// on the declared allowlist above.
func TestOneAdmissionAuthority(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	cmdDir := filepath.Join(repoRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	resultTypeMatches := func(fields *ast.FieldList) bool {
		if fields == nil || len(fields.List) != 1 {
			return false
		}
		ident, ok := fields.List[0].Type.(*ast.Ident)
		return ok && recruitmentAdmissionAuthorityResultTypes[ident.Name]
	}

	fset := token.NewFileSet()
	var violations []string
	filesInspected := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(cmdDir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		filesInspected++

		ast.Inspect(file, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				if decl.Recv != nil {
					return true
				}
				if resultTypeMatches(decl.Type.Results) && !recruitmentAdmissionAuthorityAllowlist[decl.Name.Name] {
					violations = append(violations, fmt.Sprintf(
						"%s: func %q returns an admission-shaped result and is not on recruitmentAdmissionAuthorityAllowlist",
						fset.Position(decl.Pos()).String(), decl.Name.Name,
					))
				}
			case *ast.ValueSpec:
				for i, value := range decl.Values {
					lit, ok := value.(*ast.FuncLit)
					if !ok || i >= len(decl.Names) {
						continue
					}
					if resultTypeMatches(lit.Type.Results) && !recruitmentAdmissionAuthorityAllowlist[decl.Names[i].Name] {
						violations = append(violations, fmt.Sprintf(
							"%s: var %q is a func literal returning an admission-shaped result and is not on recruitmentAdmissionAuthorityAllowlist",
							fset.Position(decl.Pos()).String(), decl.Names[i].Name,
						))
					}
				}
			}
			return true
		})
	}
	if filesInspected == 0 {
		t.Fatal("fixture is broken: no non-test .go files found in cmd/")
	}
	if len(violations) != 0 {
		t.Fatalf(
			"a second admission authority was found -- only spawnCanSpawnDecision and its declared check helpers may return an admission decision:\n%s",
			strings.Join(violations, "\n"),
		)
	}
}

// recruitmentDispatchSourceFiles is the declared universe
// TestEveryChildDispatchPassesAdmission's call-graph scan covers: every
// site in this plan's own recruitment path that can launch a real child
// process for a recruit. Scoped to these two files (not the whole cmd
// package) because a package-wide scan would also catch unrelated child
// processes (git, npm, gh, ...) this plan's admission gate was never meant
// to govern -- BIO-02's chokepoint is specifically the recruitment path.
var recruitmentDispatchSourceFiles = []string{"recruitment.go", "recruitment_dispatch.go"}

// recruitmentProcessLaunchSentinel/recruitmentAdmissionSentinel are the two
// external targets the call graph below detects without expanding further:
// an actual process launch (exec.Command/exec.CommandContext) and a real
// admission decision (spawnCanSpawnDecision). The launch sentinel starts
// with a NUL byte so it can never collide with a real Go identifier.
const recruitmentProcessLaunchSentinel = "\x00process-launch"
const recruitmentAdmissionSentinel = "spawnCanSpawnDecision"

// recruitmentBuildCallGraph parses the declared source files and returns a
// name -> set-of-called-names graph. A top-level func declaration and a
// top-level var whose initializer contains a call are both nodes -- the
// latter covers a cobra command declaration like recruitCmd, whose real
// work lives inside its RunE field's closure; ast.Inspect walks into that
// closure regardless, so a bare call anywhere in the whole var initializer
// (including inside a nested func literal) becomes an edge from that var's
// name. Every bare call (`foo(...)`) becomes an edge to "foo"; every
// exec.Command/CommandContext selector call becomes an edge to
// recruitmentProcessLaunchSentinel.
func recruitmentBuildCallGraph(t *testing.T, repoRoot string) map[string]map[string]bool {
	t.Helper()
	graph := map[string]map[string]bool{}
	fset := token.NewFileSet()

	addEdge := func(from, to string) {
		if graph[from] == nil {
			graph[from] = map[string]bool{}
		}
		graph[from][to] = true
	}

	collectCalls := func(node ast.Node, from string) {
		if node == nil {
			return
		}
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				addEdge(from, fn.Name)
			case *ast.SelectorExpr:
				if pkgIdent, ok := fn.X.(*ast.Ident); ok && pkgIdent.Name == "exec" &&
					(fn.Sel.Name == "Command" || fn.Sel.Name == "CommandContext") {
					addEdge(from, recruitmentProcessLaunchSentinel)
				}
			}
			return true
		})
	}

	filesInspected := 0
	for _, name := range recruitmentDispatchSourceFiles {
		path := filepath.Join(repoRoot, "cmd", name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		filesInspected++

		ast.Inspect(file, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				if decl.Recv != nil {
					return true
				}
				collectCalls(decl.Body, decl.Name.Name)
			case *ast.ValueSpec:
				for i, value := range decl.Values {
					if i >= len(decl.Names) {
						continue
					}
					collectCalls(value, decl.Names[i].Name)
				}
			}
			return true
		})
	}
	if filesInspected != len(recruitmentDispatchSourceFiles) {
		t.Fatalf("inspected %d file(s), expected exactly %d -- a shrunk walk would silently narrow this check", filesInspected, len(recruitmentDispatchSourceFiles))
	}
	return graph
}

// recruitmentGraphReaches reports whether target is reachable from from via
// any chain of edges in graph.
func recruitmentGraphReaches(graph map[string]map[string]bool, from, target string) bool {
	visited := map[string]bool{}
	var walk func(string) bool
	walk = func(node string) bool {
		if visited[node] {
			return false
		}
		visited[node] = true
		for callee := range graph[node] {
			if callee == target {
				return true
			}
			if walk(callee) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

// TestEveryChildDispatchPassesAdmission derives, from the real call graph of
// this plan's own recruitment source files, every ROOT entrypoint (a
// declared node nothing else in this scan calls -- a cobra command
// declaration, in practice) and asserts that any root whose call tree
// reaches a real process launch also reaches spawnCanSpawnDecision
// somewhere in that same tree. Checking roots rather than every node avoids
// flagging dispatchRecruitment in isolation: admission happens in its
// CALLER (recruitCmd), by design, not inside dispatchRecruitment itself.
func TestEveryChildDispatchPassesAdmission(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	graph := recruitmentBuildCallGraph(t, repoRoot)

	called := map[string]bool{}
	for _, callees := range graph {
		for callee := range callees {
			called[callee] = true
		}
	}

	var roots []string
	for node := range graph {
		if !called[node] {
			roots = append(roots, node)
		}
	}
	sort.Strings(roots)

	launchingRoots := 0
	for _, root := range roots {
		if !recruitmentGraphReaches(graph, root, recruitmentProcessLaunchSentinel) {
			continue
		}
		launchingRoots++
		if !recruitmentGraphReaches(graph, root, recruitmentAdmissionSentinel) {
			t.Errorf("%q launches a child process somewhere in its call tree but never reaches spawnCanSpawnDecision first", root)
		}
	}
	if launchingRoots == 0 {
		t.Fatal("fixture is broken: no root entrypoint in the recruitment dispatch source files reaches a process launch")
	}
}

// recruitmentAdmissionReasonEmitters maps each declared reason string
// (recruitmentAdmissionReasons()) to a fixture that LIVE-reproduces it by
// calling the real function that emits it, restoring the package-level
// store afterward. TestEveryAdmissionReasonIsReachable fails by name when a
// declared reason has no entry here (unreachable) or when an entry's live
// result does not actually confirm the reason it claims to prove.
var recruitmentAdmissionReasonEmitters = map[string]func(t *testing.T) bool{
	"depth": func(t *testing.T) bool {
		result := spawnCanSpawnDecision(spawnDecisionInput{RequesterDepth: spawnMaxDelegationDepth})
		return result.Reason == "depth"
	},
	"budget": func(t *testing.T) bool {
		orig := store
		defer func() { store = orig }()
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if _, err := st.BeginRun("test-run", time.Time{}); err != nil {
			t.Fatalf("begin run: %v", err)
		}
		recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax)
		result := spawnCanSpawnDecision(spawnDecisionInput{RequesterName: "Y"})
		return result.Reason == "budget"
	},
	"ancestor-cycle": func(t *testing.T) bool {
		orig := store
		defer func() { store = orig }()
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "Z1", "task z", 1); err != nil {
			t.Fatalf("seed: %v", err)
		}
		result := spawnCanSpawnDecision(spawnDecisionInput{RequesterName: "Z1", RequesterDepth: 1, DepthIsAuthoritative: true, Caste: "builder", Task: "task z"})
		return result.Reason == "ancestor-cycle"
	},
	recruitmentReasonParent: func(t *testing.T) bool {
		orig := store
		defer func() { store = orig }()
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		reason := recruitmentParentAuthorityReason(spawnDecisionInput{RequesterName: "Ghost"})
		return reason != ""
	},
	recruitmentReasonPermission: func(t *testing.T) bool {
		reason := recruitmentPermissionReason(spawnDecisionInput{Caste: "includer"})
		return reason != ""
	},
	recruitmentReasonPath: func(t *testing.T) bool {
		reason := recruitmentPathReason(spawnDecisionInput{Workspace: ""})
		return reason != ""
	},
	recruitmentReasonCost: func(t *testing.T) bool {
		reason := recruitmentCostReason(spawnDecisionInput{CostSlots: 0})
		return reason != ""
	},
	recruitmentReasonDuplicate: func(t *testing.T) bool {
		orig := store
		defer func() { store = orig }()
		store = nil
		reason := recruitmentDuplicateReason(spawnDecisionInput{RequesterName: "Queen"})
		return reason != ""
	},
	recruitmentReasonUnresolved: func(t *testing.T) bool {
		orig := store
		defer func() { store = orig }()
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		origStdout, origStderr := stdout, stderr
		defer func() { stdout, stderr = origStdout, origStderr }()
		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf

		saveGlobals(t)
		resetRootCmd(t)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "A1", "top task", 1); err != nil {
			t.Fatalf("seed A1: %v", err)
		}
		manifestDirPath := filepath.Join(store.BasePath(), filepath.FromSlash(recruitmentManifestPath))
		if err := os.MkdirAll(manifestDirPath, 0755); err != nil {
			t.Fatalf("create directory at manifest path: %v", err)
		}

		rootCmd.SetArgs([]string{
			"recruit",
			"--parent", "A1",
			"--caste", "builder",
			"--objective", "help with x",
			"--reason", "stuck on y",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("recruit command returned an error: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		result, _ := env["result"].(map[string]interface{})
		reason, _ := result["reason"].(string)
		return reason == recruitmentReasonUnresolved
	},
}

// TestEveryAdmissionReasonIsReachable cross-checks recruitmentAdmissionReasons()
// against recruitmentAdmissionReasonEmitters in both directions: a declared
// reason with no emitter (or an emitter that fails to reproduce it live)
// fails by name, and an emitter registered for a reason
// recruitmentAdmissionReasons() does not declare also fails by name.
func TestEveryAdmissionReasonIsReachable(t *testing.T) {
	declared := recruitmentAdmissionReasons()
	declaredSet := map[string]bool{}
	for _, r := range declared {
		declaredSet[r] = true
	}

	for _, reason := range declared {
		emit, ok := recruitmentAdmissionReasonEmitters[reason]
		if !ok {
			t.Errorf("declared reason %q has no registered emitter in recruitmentAdmissionReasonEmitters", reason)
			continue
		}
		if !emit(t) {
			t.Errorf("declared reason %q has a registered emitter, but it did not reproduce that reason live", reason)
		}
	}
	for reason := range recruitmentAdmissionReasonEmitters {
		if !declaredSet[reason] {
			t.Errorf("recruitmentAdmissionReasonEmitters has an entry for %q, but recruitmentAdmissionReasons() does not declare it", reason)
		}
	}
}
