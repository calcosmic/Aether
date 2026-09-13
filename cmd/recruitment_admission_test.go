package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	intentsPath := filepath.Join(store.BasePath(), filepath.FromSlash(recruitmentIntentsPath))
	if err := os.MkdirAll(filepath.Dir(intentsPath), 0755); err != nil {
		t.Fatalf("create recruitment dir: %v", err)
	}
	if err := os.MkdirAll(intentsPath, 0755); err != nil {
		t.Fatalf("create directory at intents path: %v", err)
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

	// Budget.
	recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax)
	budgetResult := spawnCanSpawnDecision(spawnDecisionInput{RequesterName: "C1", RequesterDepth: 0, Caste: "builder", Task: "y"})
	wantBudget := fmt.Sprintf("whole-run helper budget exhausted: %d of %d helpers already spawned in this run; C1 may not spawn another", spawnTreeBudgetMax, spawnTreeBudgetMax)
	if budgetResult.Reason != "budget" || budgetResult.Detail != wantBudget {
		t.Fatalf("budget detail changed: reason=%q detail=%q, want reason=budget detail=%q", budgetResult.Reason, budgetResult.Detail, wantBudget)
	}
}
