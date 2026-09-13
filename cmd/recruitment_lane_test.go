package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This file proves two things 203-09 exists to guarantee:
//
//  1. (Task 2) The Go in-repo build lane either governs a worker's spawn
//     claims for real, or refuses them honestly before launch -- it never
//     silently discards them (the pre-203-09 behavior) and never dispatches
//     one ungoverned.
//  2. (Task 3) Both lanes -- the TypeScript host's bridge (203-09 Task 1)
//     and the interactive/native lane (cmd/recruitment.go, Task 2) --
//     consume the SAME whole-run admission counter and speak the SAME
//     fixed reason vocabulary, and a future lane that dispatches without
//     reaching the gate fails a named test.
//
// File-ownership note: cmd/recruitment_lane.go (the production code this
// file tests) and the one-line call site it needed in cmd/codex_build.go
// are both outside this plan's declared files_modified -- see
// 203-09-SUMMARY.md's Deviations section for the full account of why that
// expansion was necessary and how it was scoped.

// --- Task 2: the in-repo lane governs or honestly refuses ---

func TestInRepoLaneSpawnClaims(t *testing.T) {
	if os.Getenv("AETHER_RECRUIT_CHILD") == "1" {
		// Recursively invoked as the recruited child, exactly like
		// TestRecruitmentTracerEndToEnd's own gate (cmd/recruitment_test.go).
		fmt.Println("recruited-child-ok")
		return
	}

	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf bytes.Buffer
	stdout = &buf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}

	t.Setenv("AETHER_RECRUIT_BINARY", os.Args[0])
	t.Setenv("AETHER_RECRUIT_ARGS", "-test.run=^TestInRepoLaneSpawnClaims$")
	t.Setenv("AETHER_RECRUIT_TIMEOUT", "30s")

	dispatch := codex.WorkerDispatch{WorkerName: "Builder-01", Caste: "builder", Root: tmpDir}
	result := codex.WorkerResult{
		// A worker that emitted a structured claim as JSON survives
		// pkg/codex/worker.go's stringList decoding as raw JSON text (its
		// own documented mixed-array fallback) -- this is that shape.
		Spawns: []string{`{"caste":"scout","task":"research the pagination bug"}`},
	}

	routeInRepoSpawnClaims(tmpDir, colony.ModeInRepo, dispatch, result)

	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	var childEntry *agent.SpawnEntry
	for i := range entries {
		if entries[i].ParentName == "Builder-01" && entries[i].AgentName != "Builder-01" {
			childEntry = &entries[i]
		}
	}
	if childEntry == nil {
		t.Fatalf("expected exactly one spawn-tree entry for the well-formed claim, found none (entries: %+v)", entries)
	}
	if childEntry.Depth != 2 {
		t.Fatalf("child entry depth = %d, want 2 (Builder-01's own depth 1, plus 1)", childEntry.Depth)
	}

	var manifestFile recruitmentManifestFile
	if err := store.LoadJSON(recruitmentManifestPath, &manifestFile); err != nil {
		t.Fatalf("load recruitment manifest: %v", err)
	}
	if len(manifestFile.Entries) != 1 {
		t.Fatalf("expected exactly 1 manifest amendment, got %d", len(manifestFile.Entries))
	}
	if manifestFile.Entries[0].ParentName != "Builder-01" {
		t.Fatalf("manifest entry parent = %q, want %q", manifestFile.Entries[0].ParentName, "Builder-01")
	}
}

func TestInRepoLaneBareStringClaimRefusedAsSchema(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}
	before, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}

	claim := parseInRepoSpawnClaim("please spawn a helper for me")
	if claim.WellFormed {
		t.Fatalf("expected a bare string claim to be reported not well-formed, got %+v", claim)
	}

	dispatch := codex.WorkerDispatch{WorkerName: "Builder-01", Caste: "builder", Root: tmpDir}
	result := codex.WorkerResult{Spawns: []string{"please spawn a helper for me"}}
	routeInRepoSpawnClaims(tmpDir, colony.ModeInRepo, dispatch, result)

	after, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("a bare-string (schema-invalid) claim must dispatch no child: entries before=%d after=%d", len(before), len(after))
	}
}

func TestUnsupportedLaneRefusesBeforeLaunch(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	// TestMain (cmd/testing_main_test.go) pins AETHER_OUTPUT_MODE=json for
	// the whole suite; shouldRenderVisualOutput checks that BEFORE
	// AETHER_FORCE_VISUAL, so this test overrides the mode directly to
	// actually observe the refusal sentence emitVisualProgress writes.
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var buf bytes.Buffer
	stdout = &buf

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}
	before, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}

	dispatch := codex.WorkerDispatch{WorkerName: "Builder-01", Caste: "builder", Root: tmpDir}
	result := codex.WorkerResult{
		Spawns: []string{`{"caste":"scout","task":"research the pagination bug"}`},
	}

	// A worktree-isolated build worker's spawn claim is ruled unsupported
	// for real dispatch on this lane (see recruitmentLaneUnsupportedDetail).
	routeInRepoSpawnClaims(tmpDir, colony.ModeWorktree, dispatch, result)

	after, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	// The launch counter: no spawn-tree entry beyond the seeded parent.
	if launched := len(after) - len(before); launched != 0 {
		t.Fatalf("launch counter = %d, want 0 (the unsupported lane must start no child)", launched)
	}

	if !strings.Contains(buf.String(), recruitmentLaneUnsupportedDetail) {
		t.Fatalf("expected the refusal sentence %q in output, got: %s", recruitmentLaneUnsupportedDetail, buf.String())
	}

	exists, err := s.FileExists(recruitmentManifestPath)
	if err != nil {
		t.Fatalf("check manifest existence: %v", err)
	}
	if exists {
		t.Fatal("expected no recruitment manifest amendment for an unsupported-lane refusal")
	}
}

func TestNoClaimsMeansNoAdmissionCall(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := st.RecordSpawn("Queen", "builder", "Builder-01", "build the feature", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}

	dispatch := codex.WorkerDispatch{WorkerName: "Builder-01", Caste: "builder", Root: tmpDir}
	result := codex.WorkerResult{} // no Spawns at all -- the plain, ordinary case

	routeInRepoSpawnClaims(tmpDir, colony.ModeInRepo, dispatch, result)

	// A single admission call anywhere in this path creates
	// recruitment/intents.json (recordRecruitmentIntent always runs before
	// any admission decision). Its continued absence is the zero-calls
	// counter.
	exists, err := s.FileExists(recruitmentIntentsPath)
	if err != nil {
		t.Fatalf("check recruitment intents existence: %v", err)
	}
	if exists {
		t.Fatal("a worker with no spawn claims must trigger zero admission calls -- recruitment/intents.json should not exist")
	}
}

// --- Task 3: one counter, one vocabulary, both lanes ---

// TestBothLanesShareOneAdmissionCounter proves the host lane (simulated by
// driving spawnCanSpawnDecision the exact way the TS bridge's
// `aether spawn-can-spawn` call does, origin spawnOriginSpawnCanSpawn) and
// the interactive lane (origin spawnOriginRecruit) consume the SAME
// spawn-tree ledger -- never two independent counters. Both sides are
// derived from real SpawnTree.RecordSpawn calls and the runtime's own
// spawnTreeBudgetState(), never a typed literal standing in for either.
func TestBothLanesShareOneAdmissionCounter(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_ = tmpDir

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	// Leave exactly one slot: spawnTreeBudgetMax fillers minus 2, plus the
	// one parent entry seeded below, is spawnTreeBudgetMax-1 consumed.
	recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax-2)
	if err := st.RecordSpawn("Queen", "builder", "A1", "help with the login form", 1); err != nil {
		t.Fatalf("seed parent spawn: %v", err)
	}

	state, err := spawnTreeBudgetState()
	if err != nil {
		t.Fatalf("read budget state: %v", err)
	}
	if state.Max-state.Consumed != 1 {
		t.Fatalf("fixture is broken: expected exactly 1 remaining slot before the interactive lane's request, got %d (consumed=%d max=%d)", state.Max-state.Consumed, state.Consumed, state.Max)
	}

	// The interactive lane (aether recruit's own admission call shape)
	// consumes the last slot.
	interactive := spawnCanSpawnDecision(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "scout",
		Task:                 "research the pagination bug",
		Origin:               spawnOriginRecruit,
		Workspace:            tmpDir,
		CostSlots:            1,
		IntentID:             "intent-interactive-lane",
		AttemptID:            "attempt-interactive-lane",
	})
	if !interactive.Allowed {
		t.Fatalf("expected the interactive lane to consume the last real slot, got denied: reason=%q detail=%q", interactive.Reason, interactive.Detail)
	}
	if err := st.RecordSpawn("A1", "scout", "A1-child", "research the pagination bug", 2); err != nil {
		t.Fatalf("record the interactive lane's admitted child: %v", err)
	}

	// The host lane -- exactly the admission shape `aether spawn-can-spawn`
	// exposes to the TS bridge (Task 1) -- asks about a SECOND child from
	// the SAME requester, against the SAME real ledger state.
	host := spawnCanSpawnDecision(spawnDecisionInput{
		RequesterName:        "A1",
		RequesterDepth:       1,
		DepthIsAuthoritative: true,
		Caste:                "builder",
		Task:                 "build the follow-up feature",
		Origin:               spawnOriginSpawnCanSpawn,
	})
	if host.Allowed {
		t.Fatal("expected the host lane's second child to be refused: the interactive lane already consumed the last real slot on the SAME ledger")
	}
	if host.Reason != "budget" {
		t.Fatalf("host lane reason = %q, want %q", host.Reason, "budget")
	}
}

// assertSameLaneAnswer fails the test unless native and host agree on both
// allow/deny and, when both deny, the exact reason string.
func assertSameLaneAnswer(t *testing.T, dimension string, native, host recruitmentDecisionResult) {
	t.Helper()
	if native.Allowed != host.Allowed {
		t.Fatalf("%s: native.Allowed=%v host.Allowed=%v -- the two lanes disagree on whether this claim is admitted", dimension, native.Allowed, host.Allowed)
	}
	if native.Reason != host.Reason {
		t.Fatalf("%s: native reason = %q, host reason = %q -- the two lanes must speak the same reason vocabulary", dimension, native.Reason, host.Reason)
	}
}

// TestBothLanesUseOneReasonVocabulary is CR-01's parity fix (203-REVIEW.md).
//
// spawnOriginSpawnCanSpawn's OWN declared check table stays empty (the
// first assertion below): an ORDINARY spawn-can-spawn advisory call -- no
// --recruitment flag -- still applies none of BIO-02's five recruitment-only
// dimensions, exactly as before this fix, because spawn-log and an ordinary
// spawn-can-spawn call never populate Permission/Workspace/CostSlots/
// IntentID, and applying these checks to them would deny every ordinary
// spawn on missing data rather than skip a check that does not apply.
//
// But a REAL recruitment claim on the host/autopilot lane no longer asks
// under that bare origin. recruitmentClaimAdmission (cmd/recruitment_admission.go)
// -- the function spawnCanSpawnCmd's --recruitment flag calls, and the
// function the TypeScript host's spawn-orchestrator bridge now drives
// through that flag -- carries a claim through the SAME spawnOriginRecruit
// gate the in-repo build lane (cmd/recruitment_lane.go) and the interactive
// `aether recruit` command already use.
//
// For each of the four dimensions CR-01 found missing on the host lane
// (permission, path, cost, duplicate), this test seeds the SAME ledger
// state the way the runtime actually produces it -- a real RecordSpawn
// call, a real recordRecruitmentIntent call, a real budget fill, never a
// hand-typed literal pretending to be one -- and asserts that BOTH a direct
// spawnOriginRecruit decision (the reference every native-lane caller --
// recruitCmd, dispatchOneInRepoRecruitment -- already reaches) and
// recruitmentClaimAdmission (the host lane's own new gate call) return the
// identical allow/deny answer and the identical reason string.
func TestBothLanesUseOneReasonVocabulary(t *testing.T) {
	applicable := recruitmentAdmissionChecks[spawnOriginSpawnCanSpawn]
	if len(applicable) != 0 {
		t.Fatalf("spawnOriginSpawnCanSpawn must be declared with an empty check set (an ORDINARY advisory check must never gain BIO-02's five recruitment-only dimensions), got %v", applicable)
	}
	// None of BIO-02's five recruitment-only reasons can ever surface for an
	// ordinary (non-recruitment) spawn-can-spawn call -- confirmed
	// structurally above, and reinforced here by asking recruitmentCheckApplies
	// directly.
	for _, reason := range []string{
		recruitmentReasonParent, recruitmentReasonPermission, recruitmentReasonPath,
		recruitmentReasonCost, recruitmentReasonDuplicate,
	} {
		if recruitmentCheckApplies(spawnOriginSpawnCanSpawn, reason) {
			t.Fatalf("recruitmentCheckApplies(spawnOriginSpawnCanSpawn, %q) = true, want false", reason)
		}
	}

	t.Run("ordinary_advisory_check_is_unaffected", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")

		got := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterDepth:       2,
			DepthIsAuthoritative: true,
			Origin:               spawnOriginSpawnCanSpawn,
		})
		if got.Allowed || got.Reason != "depth" {
			t.Fatalf("depth: allowed=%v reason=%q, want denied with reason %q", got.Allowed, got.Reason, "depth")
		}

		recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax)
		budgetResult := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        "Queen",
			RequesterDepth:       0,
			DepthIsAuthoritative: true,
			Origin:               spawnOriginSpawnCanSpawn,
		})
		if budgetResult.Allowed || budgetResult.Reason != "budget" {
			t.Fatalf("expected reason %q with a full ledger, got allowed=%v reason=%q", "budget", budgetResult.Allowed, budgetResult.Reason)
		}
	})

	t.Run("permission", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "A1", "top-level task", 1); err != nil {
			t.Fatalf("seed parent: %v", err)
		}

		// "includer" resolves to the repository-read-only permission
		// profile (TestRecruitmentAdmissionPermissionDeniesReadOnlyCaste) --
		// every recruitment dispatches a real, write-capable child process,
		// so this caste must be refused on BOTH lanes.
		native := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        "A1",
			RequesterDepth:       1,
			DepthIsAuthoritative: true,
			Caste:                "includer",
			Task:                 "audit the accessibility of the new form",
			Origin:               spawnOriginRecruit,
			Workspace:            tmpDir,
			CostSlots:            1,
			IntentID:             "native-permission-check",
			AttemptID:            "native-permission-check",
		})
		_, host := recruitmentClaimAdmission("A1", 1, true, "includer", "audit the accessibility of the new form", tmpDir, 1)

		assertSameLaneAnswer(t, "permission", recruitmentDecisionResult{Allowed: native.Allowed, Reason: native.Reason}, host)
		if host.Allowed || host.Reason != recruitmentReasonPermission {
			t.Fatalf("expected the host lane to refuse a read-only caste with reason %q, got allowed=%v reason=%q", recruitmentReasonPermission, host.Allowed, host.Reason)
		}
	})

	t.Run("path", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "A1", "top-level task", 1); err != nil {
			t.Fatalf("seed parent: %v", err)
		}

		outside := t.TempDir()

		native := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        "A1",
			RequesterDepth:       1,
			DepthIsAuthoritative: true,
			Caste:                "builder",
			Task:                 "fix the pagination bug",
			Origin:               spawnOriginRecruit,
			Workspace:            outside,
			CostSlots:            1,
			IntentID:             "native-path-check",
			AttemptID:            "native-path-check",
		})
		_, host := recruitmentClaimAdmission("A1", 1, true, "builder", "fix the pagination bug", outside, 1)

		assertSameLaneAnswer(t, "path", recruitmentDecisionResult{Allowed: native.Allowed, Reason: native.Reason}, host)
		if host.Allowed || host.Reason != recruitmentReasonPath {
			t.Fatalf("expected the host lane to refuse a workspace outside the colony root with reason %q, got allowed=%v reason=%q", recruitmentReasonPath, host.Allowed, host.Reason)
		}
	})

	t.Run("cost", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "A1", "top-level task", 1); err != nil {
			t.Fatalf("seed parent: %v", err)
		}
		// Leave exactly 2 slots remaining (spawnTreeBudgetMax - 2 fillers,
		// minus the 1 the parent above already consumed) -- ENOUGH that the
		// generic whole-run spawnTreeBudgetReason check (shared by every
		// origin, checked before BIO-02's five dimensions) still passes, so
		// a request for 5 slots is denied specifically by
		// recruitmentCostReason's own "this request exceeds what remains"
		// dimension, not merely re-proving the already-shared budget check.
		recruitmentAdmissionFillBudget(t, st, spawnTreeBudgetMax-3)

		native := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        "A1",
			RequesterDepth:       1,
			DepthIsAuthoritative: true,
			Caste:                "builder",
			Task:                 "fix the pagination bug",
			Origin:               spawnOriginRecruit,
			Workspace:            tmpDir,
			CostSlots:            5,
			IntentID:             "native-cost-check",
			AttemptID:            "native-cost-check",
		})
		_, host := recruitmentClaimAdmission("A1", 1, true, "builder", "fix the pagination bug", tmpDir, 5)

		assertSameLaneAnswer(t, "cost", recruitmentDecisionResult{Allowed: native.Allowed, Reason: native.Reason}, host)
		if host.Allowed || host.Reason != recruitmentReasonCost {
			t.Fatalf("expected the host lane to refuse a request exceeding the remaining budget with reason %q, got allowed=%v reason=%q", recruitmentReasonCost, host.Allowed, host.Reason)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
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
				IntentID:      "pending-shared",
				ParentName:    "A1",
				Caste:         "builder",
				Objective:     "Fix the login form.",
			},
			CreatedAt: "2026-01-01T00:00:00Z",
		}
		if _, err := recordRecruitmentIntent(pending); err != nil {
			t.Fatalf("record pending intent: %v", err)
		}

		native := spawnCanSpawnDecision(spawnDecisionInput{
			RequesterName:        "A1",
			RequesterDepth:       1,
			DepthIsAuthoritative: true,
			Caste:                "builder",
			Task:                 "fix the login form", // normalizes to the same text as "Fix the login form."
			Origin:               spawnOriginRecruit,
			Workspace:            tmpDir,
			CostSlots:            1,
			IntentID:             "native-duplicate-check",
			AttemptID:            "native-duplicate-check",
		})
		_, host := recruitmentClaimAdmission("A1", 1, true, "builder", "fix the login form", tmpDir, 1)

		assertSameLaneAnswer(t, "duplicate", recruitmentDecisionResult{Allowed: native.Allowed, Reason: native.Reason}, host)
		if host.Allowed || host.Reason != recruitmentReasonDuplicate {
			t.Fatalf("expected the host lane to refuse a claim matching an already-pending intent in the same subtree with reason %q, got allowed=%v reason=%q", recruitmentReasonDuplicate, host.Allowed, host.Reason)
		}
	})
}

// TestNoBudgetArithmeticOutsideGo is a source scan of .aether/ts-host/src
// failing BY NAME for any file (outside a small, reviewed allowlist of
// files with their OWN, differently-scoped budget concept) that still
// defines the specific symbol names this plan deleted from
// spawn-orchestrator.ts.
//
// confidence-loop.ts and research-confidence.ts are allowlisted: both
// independently define a "totalBudget"/"DEFAULT_TOTAL_BUDGET" for the
// CONFIDENCE ITERATION loop (how many build iterations to try), a
// pre-existing, unrelated concept that happens to share a name -- not the
// spawn-admission arithmetic this plan retires. Reviewed and named here,
// exactly the "coincidental literal collision" 203-04-SUMMARY.md's
// WHICH_LOOKUP_TIMEOUT_MS deviation already established the precedent for.
func TestNoBudgetArithmeticOutsideGo(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	srcDir := repoRoot + "/.aether/ts-host/src"
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		t.Fatalf("read %s: %v", srcDir, err)
	}

	allowlist := map[string]bool{
		"confidence-loop.ts":     true,
		"research-confidence.ts": true,
	}
	forbidden := []string{"DEFAULT_TOTAL_BUDGET", "MAX_SPAWN_DEPTH", "consumedBudget", "remainingBudget"}

	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".ts") || allowlist[name] {
			continue
		}
		data, err := os.ReadFile(srcDir + "/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		for _, symbol := range forbidden {
			if strings.Contains(text, symbol) {
				violations = append(violations, fmt.Sprintf("%s contains %q", name, symbol))
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("budget arithmetic reintroduced outside Go:\n%s", strings.Join(violations, "\n"))
	}
}

// TestEveryChildDispatchLaneReachesTheGate derives its lane inventory from
// spawnDecisionOrigins() (cmd/spawn.go's own completeness accessor) rather
// than a hand-typed list -- the same registry discipline
// TestEveryLifecycleLaneEmitsLiveEvents uses (cmd/live_lane_coverage_test.go)
// -- so an origin added later with no registered real dispatch entry point
// fails BY NAME, never silently passing.
var recruitmentLaneEntryPoints = map[spawnDecisionOrigin]func(t *testing.T) bool{
	spawnOriginSpawnLog: func(t *testing.T) bool {
		t.Helper()
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf

		reached := false
		orig := spawnCanSpawnDecision
		spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
			reached = true
			return orig(in)
		}
		defer func() { spawnCanSpawnDecision = orig }()

		rootCmd.SetArgs([]string{"spawn-log", "--parent", "Queen", "--caste", "builder", "--name", "L1", "--task", "do a thing"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("spawn-log returned an error: %v", err)
		}
		return reached
	},
	spawnOriginSpawnCanSpawn: func(t *testing.T) bool {
		t.Helper()
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf

		reached := false
		orig := spawnCanSpawnDecision
		spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
			reached = true
			return orig(in)
		}
		defer func() { spawnCanSpawnDecision = orig }()

		rootCmd.SetArgs([]string{"spawn-can-spawn", "--depth", "1", "--caste", "scout", "--task", "research"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("spawn-can-spawn returned an error: %v", err)
		}
		return reached
	},
	spawnOriginRecruit: func(t *testing.T) bool {
		t.Helper()
		saveGlobals(t)
		resetRootCmd(t)
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		var buf, errBuf bytes.Buffer
		stdout = &buf
		stderr = &errBuf

		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn("Queen", "builder", "P1", "do a thing", 1); err != nil {
			t.Fatalf("seed grandparent spawn: %v", err)
		}
		if err := st.RecordSpawn("P1", "builder", "P2", "do a sub-thing", 2); err != nil {
			t.Fatalf("seed parent spawn: %v", err)
		}

		reached := false
		orig := spawnCanSpawnDecision
		spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
			reached = true
			return orig(in)
		}
		defer func() { spawnCanSpawnDecision = orig }()

		// P2 is already at depth 2 (the cap): this recruitment denies at
		// the depth check, inside spawnCanSpawnDecision, before any real
		// process could ever launch -- no AETHER_RECRUIT_* env plumbing
		// needed for this entry point.
		rootCmd.SetArgs([]string{
			"recruit",
			"--parent", "P2",
			"--caste", "builder",
			"--objective", "help with x",
			"--reason", "stuck on y",
		})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("recruit returned an error: %v", err)
		}
		return reached
	},
}

func TestEveryChildDispatchLaneReachesTheGate(t *testing.T) {
	origins := spawnDecisionOrigins()
	if len(origins) == 0 {
		t.Fatal("fixture is broken: spawnDecisionOrigins() returned no declared origins")
	}
	for _, origin := range origins {
		origin := origin
		t.Run(string(origin), func(t *testing.T) {
			entry, ok := recruitmentLaneEntryPoints[origin]
			if !ok {
				t.Fatalf("no real dispatch entry point is registered in recruitmentLaneEntryPoints for declared origin %q -- every origin in spawnDecisionOrigins() must have one", origin)
			}
			if !entry(t) {
				t.Fatalf("origin %q's real dispatch path never reached spawnCanSpawnDecision", origin)
			}
		})
	}
}
