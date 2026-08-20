package cmd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This file closes D-190-05-A (Phase 190 Plan 06): continue's native review
// and watcher dispatches independently double-delivered prior-worker
// handoffs -- once via ContextCapsule (resolveCodexWorkerContext(), which
// unconditionally renders "## Previous Worker Handoffs" for "build"-workflow
// records, cmd/colony_prime_context.go:695) and once via a separately-set
// HandoffSection field (renderWorkerHandoffSection("continue", ...)).
//
// Unlike D-190-03-A's pheromone case (190-05), the two channels here do NOT
// carry identical content -- the capsule is hardcoded to "build"-workflow
// records; the dedicated field was filtered to "continue"-workflow records
// (a real relay between sibling continue dispatches, per
// plannedContinueWatcherDispatch's own "nothing in the design justified the
// asymmetry" comment). Deleting the field, as 190-05 did for pheromones,
// would have dropped that relay content to zero. The fix instead gives the
// dedicated channel (renderRelatedWorkflowHandoffSection) its own heading
// ("## Related Worker Handoffs") distinct from the capsule's
// ("## Previous Worker Handoffs"), so both channels keep exactly one home.
//
// The audit that produced this fix (throwaway probe, deleted after use)
// found the IDENTICAL shape on four more native dispatch paths never named
// by D-190-05-A: colonize, plan, seal, and swarm all pair
// resolveCodexWorkerContext() with a same-field, own-workflow-tagged
// HandoffSection. Per this repo's Definition of Done (a "final gap of the
// phase" claim must not leave a proven, reproduced instance of the exact bug
// being closed undiscovered-but-discoverable), those four are fixed here too
// -- see codexColonizeHandoffCase190_06 through swarmHandoffCase190_06 below
// and cmd/build_handoff_190_06_breadth_findings.md-equivalent notes in the
// plan summary. codexWorkerDispatchesForRecovery (build's retry-instruction
// builder) used the SAME "build" workflow tag as the capsule -- a TRUE
// duplicate, not a materially-different one -- so its HandoffSection was
// removed outright, mirroring 190-05's own PheromoneSection fix at that
// exact call site.

// seedRelatedWorkflowHandoff190_06 persists one additional stored handoff
// tagged with the given workflow, carrying a sentinel unique to the calling
// (sub)test. Used alongside seedHandoffColonyForBriefTests (which seeds a
// "build"-workflow handoff, the content the shared capsule always renders)
// so a case can prove BOTH the capsule's build-carryover content and the
// dedicated field's own-workflow relay content survive the fix, each under
// its own heading -- neither silently dropped to zero.
func seedRelatedWorkflowHandoff190_06(t *testing.T, workflow string, phaseID int, workerName string, sentinel string) {
	t.Helper()
	dispatch := codex.WorkerDispatch{
		WorkerName: workerName,
		Caste:      "builder",
		TaskID:     "related-190-06",
		Workflow:   workflow,
		Phase:      phaseID,
		Wave:       1,
	}
	result := codex.DispatchResult{
		WorkerName: workerName,
		Status:     "completed",
		WorkerResult: &codex.WorkerResult{
			WorkerName: workerName,
			Caste:      "builder",
			TaskID:     "related-190-06",
			Status:     "completed",
			Summary:    "seeded related-workflow handoff for 190-06",
			Handoff: codex.WorkerHandoff{
				NextWorkerInstructions: []string{sentinel},
				Freshness:              time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	if err := persistDispatchWorkerHandoff(dispatch, result); err != nil {
		t.Fatalf("seed %s-workflow handoff: %v", workflow, err)
	}
}

const (
	previousWorkerHandoffsHeading190_06 = "## Previous Worker Handoffs"
	relatedWorkerHandoffsHeading190_06  = "## Related Worker Handoffs"
)

// assertHandoffDeliveredOnceEachChannel190_06 is the shared assertion for
// both the dedicated continue tests and the breadth suite below: the
// capsule's own "## Previous Worker Handoffs" heading (build-workflow
// carryover) must appear exactly once and carry buildSentinel; the dedicated
// field's "## Related Worker Handoffs" heading (same-workflow sibling relay)
// must ALSO appear exactly once and carry ownSentinel. Never twice (the bug
// this closes) and never zero (the trap the fix must not fall into).
func assertHandoffDeliveredOnceEachChannel190_06(t *testing.T, label string, r steeringResult190_05, buildSentinel, ownSentinel string) {
	t.Helper()
	assembled := codex.AssembleHostedPrompt(r.contextCapsule, r.handoffSection, r.skillSection, r.pheromoneSection, r.taskBrief)

	if n := strings.Count(assembled, previousWorkerHandoffsHeading190_06); n != 1 {
		t.Fatalf("%s: assembled prompt has %d %q headings, want exactly 1\ncapsule=%q\nhandoffSection=%q", label, n, previousWorkerHandoffsHeading190_06, r.contextCapsule, r.handoffSection)
	}
	if n := strings.Count(assembled, relatedWorkerHandoffsHeading190_06); n != 1 {
		t.Fatalf("%s: assembled prompt has %d %q headings, want exactly 1\nhandoffSection=%q", label, n, relatedWorkerHandoffsHeading190_06, r.handoffSection)
	}
	if n := strings.Count(assembled, buildSentinel); n != 1 {
		t.Fatalf("%s: build-workflow handoff content (delivered via the capsule) appears %d times, want exactly 1 -- the capsule channel must not silently drop this content to zero while fixing the duplicate heading:\n%s", label, n, assembled)
	}
	if n := strings.Count(assembled, ownSentinel); n != 1 {
		t.Fatalf("%s: own-workflow relay content (delivered via HandoffSection) appears %d times, want exactly 1 -- the dedicated channel must not silently drop this content to zero:\n%s", label, n, assembled)
	}
}

// TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading is the primary
// fail-then-pass regression lock for D-190-05-A on
// plannedContinueReviewDispatches.
//
// BEFORE this plan's fix: seeding a "build"-workflow handoff (which
// ContextCapsule renders under "## Previous Worker Handoffs",
// cmd/colony_prime_context.go:695) alongside a "continue"-workflow handoff
// (which HandoffSection rendered under the SAME heading, via
// renderWorkerHandoffSection("continue", ...)) produced an assembled prompt
// with that heading appearing twice:
//
//	continue_review: assembled prompt has 2 "## Previous Worker Handoffs" headings, want exactly 1
//
// (reproduced by the throwaway probe this fix was developed against; see the
// plan summary for the full empirical trace).
//
// AFTER: HandoffSection now renders under "## Related Worker Handoffs"
// instead (renderRelatedWorkflowHandoffSection), so both the capsule's
// build-carryover and the dedicated field's continue-sibling relay survive,
// each under its own heading, exactly once.
func TestContinueReviewHandoffStaysExactlyOnceViaOwnHeading(t *testing.T) {
	saveGlobalsCmd(t)

	const buildSentinel = "SENTINEL-190-06-CONTINUE-REVIEW-BUILD"
	const ownSentinel = "SENTINEL-190-06-CONTINUE-REVIEW-CONTINUE"
	root, phase, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
	seedRelatedWorkflowHandoff190_06(t, "continue", phase.ID, "PriorContinueReviewer-Seed-190-06", ownSentinel)

	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthHeavy)
	if len(dispatches) == 0 {
		t.Fatal("fixture broken: expected non-empty continue review dispatches at heavy depth")
	}

	assertHandoffDeliveredOnceEachChannel190_06(t, "continue_review", assembledFromDispatch190_05(dispatches[0]), buildSentinel, ownSentinel)
}

// TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading is the primary
// fail-then-pass regression lock for D-190-05-A on
// plannedContinueWatcherDispatch. Same shape and same BEFORE failure text
// (with "continue_watcher" in place of "continue_review") as the review test
// above -- see that test's doc comment for the full BEFORE/AFTER trace.
func TestContinueWatcherHandoffStaysExactlyOnceViaOwnHeading(t *testing.T) {
	saveGlobalsCmd(t)

	const buildSentinel = "SENTINEL-190-06-CONTINUE-WATCHER-BUILD"
	const ownSentinel = "SENTINEL-190-06-CONTINUE-WATCHER-CONTINUE"
	root, phase, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
	seedRelatedWorkflowHandoff190_06(t, "continue", phase.ID, "PriorContinueWatcher-Seed-190-06", ownSentinel)

	invoker := &codex.FakeInvoker{}
	d := plannedContinueWatcherDispatch(root, phase, codexContinueManifest{}, nil, codexClaimVerification{}, codexWatcherVerification{}, invoker, 0)

	assertHandoffDeliveredOnceEachChannel190_06(t, "continue_watcher", assembledFromDispatch190_05(d), buildSentinel, ownSentinel)
}

// TestNineCommandsDeliverHandoffExactlyOnce is the breadth lock for
// D-190-05-A / 190-06, mirroring 190-05's TestEightCommandsDeliverPheromoneExactlyOnce
// exactly: every command that assembles a worker prompt through
// AssemblePrompt/AssembleHostedPrompt must deliver stored handoff content
// without a repeated section heading. Nine cases, matching 190-05's set:
//
//   - build_native: HandoffSection is never set on this path
//     (attachBuildDispatchContext is not called there); the capsule alone
//     delivers "## Previous Worker Handoffs" exactly once. Already locked by
//     TestNativeDispatchHandoffStaysExactlyOnceViaCapsule; re-asserted here
//     for structural parity with 190-05's breadth suite.
//   - continue_review, continue_watcher, plan, colonize, seal, swarm: capsule
//     (build-workflow, "## Previous Worker Handoffs") + dedicated field
//     (own-workflow, "## Related Worker Handoffs") -- both present, distinct
//     headings, neither zero.
//   - quick, oracle: their capsules (renderQuickContextCapsule,
//     renderOracleContextCapsule) never render handoffs at all, so
//     HandoffSection (still renderWorkerHandoffSection, unchanged) remains
//     their SOLE channel under "## Previous Worker Handoffs" -- exactly one
//     occurrence, no capsule-side content to collide with.
func TestNineCommandsDeliverHandoffExactlyOnce(t *testing.T) {
	// Same store-leak contract TestEightCommandsDeliverPheromoneExactlyOnce
	// documents: seedHandoffColonyForBriefTests sets the package-global
	// `store` and relies on its caller to restore it. saveGlobalsCmd here
	// restores it once, after every subtest below has finished.
	saveGlobalsCmd(t)

	t.Run("build_native", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BUILD-NATIVE-BUILD"
		dataDir := setupBuildFlowTest(t)
		root := filepath.Dir(filepath.Dir(dataDir))
		chdirForTest190_05(t, root)

		goal := "190-06 breadth: build native handoff once-ness"
		createTestColonyState(t, dataDir, colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ColonyDepth:  "full",
			CurrentPhase: 0,
			Plan: colony.Plan{
				Phases: []colony.Phase{
					{ID: 1, Name: "190-06 native handoff", Status: colony.PhaseReady, SuccessCriteria: []string{"Handoff content reaches the worker exactly once"}},
				},
			},
		})
		recent := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		handoffs := workerHandoffFile{Entries: []workerHandoffRecord{
			{ID: "190-06-build-native", Workflow: "build", Phase: 1, WorkerName: "PriorFixtureWorker-190-06", Status: "completed", VerificationStatus: "pass", Summary: buildSentinel, Freshness: recent},
		}}
		if err := store.SaveJSON(workerHandoffsPath, handoffs); err != nil {
			t.Fatalf("save handoffs: %v", err)
		}

		phase := colony.Phase{ID: 1, Name: "190-06 native handoff"}
		dispatches := []codexBuildDispatch{{Name: "NativeWorker-1", Caste: "builder", Task: "do the thing"}}
		spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := spawnTree.RecordSpawn("Queen", "builder", "NativeWorker-1", "do the thing", 1); err != nil {
			t.Fatalf("seed spawn tree: %v", err)
		}
		invoker := &codex.FakeInvoker{}
		results, _, _, err := executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), invoker, colony.ModeInRepo, 0, 3, false, nil)
		if err != nil {
			t.Fatalf("executeCodexBuildDispatches: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("expected at least one dispatch result")
		}
		capsule := resolveCodexWorkerContext()
		assembled := codex.AssembleHostedPrompt(capsule, results[0].HandoffSection, "", "", "task brief")
		if n := strings.Count(assembled, previousWorkerHandoffsHeading190_06); n != 1 {
			t.Fatalf("build_native: assembled prompt has %d %q headings, want exactly 1:\n%s", n, previousWorkerHandoffsHeading190_06, assembled)
		}
		if n := strings.Count(assembled, buildSentinel); n != 1 {
			t.Fatalf("build_native: build-workflow handoff content appears %d times, want exactly 1:\n%s", n, assembled)
		}
	})

	t.Run("continue_review", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-CONTINUE-REVIEW-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-CONTINUE-REVIEW-OWN"
		root, phase, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
		seedRelatedWorkflowHandoff190_06(t, "continue", phase.ID, "PriorContinueReviewer-Breadth-190-06", ownSentinel)

		invoker := &codex.FakeInvoker{}
		dispatches := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthHeavy)
		if len(dispatches) == 0 {
			t.Fatal("fixture broken: expected non-empty continue review dispatches")
		}
		assertHandoffDeliveredOnceEachChannel190_06(t, "continue_review", assembledFromDispatch190_05(dispatches[0]), buildSentinel, ownSentinel)
	})

	t.Run("continue_watcher", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-CONTINUE-WATCHER-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-CONTINUE-WATCHER-OWN"
		root, phase, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
		seedRelatedWorkflowHandoff190_06(t, "continue", phase.ID, "PriorContinueWatcher-Breadth-190-06", ownSentinel)

		invoker := &codex.FakeInvoker{}
		d := plannedContinueWatcherDispatch(root, phase, codexContinueManifest{}, nil, codexClaimVerification{}, codexWatcherVerification{}, invoker, 0)
		assertHandoffDeliveredOnceEachChannel190_06(t, "continue_watcher", assembledFromDispatch190_05(d), buildSentinel, ownSentinel)
	})

	t.Run("plan", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-PLAN-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-PLAN-OWN"
		root, _, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
		chdirForTest190_05(t, root)
		seedRelatedWorkflowHandoff190_06(t, "plan", 0, "PriorPlanner-Breadth-190-06", ownSentinel)

		spy := &pheromoneCaptureInvoker190_05{}
		if _, err := dispatchRealPlanningWorkersWithIterationContext(context.Background(), root, codexSurveyContext{}, spy, 0, "", "Plan something"); err != nil {
			t.Fatalf("dispatchRealPlanningWorkersWithIterationContext: %v", err)
		}
		configs := spy.captured()
		if len(configs) == 0 {
			t.Fatal("expected at least one captured planning worker config")
		}
		assertHandoffDeliveredOnceEachChannel190_06(t, "plan", assembledFromConfig190_05(configs[0]), buildSentinel, ownSentinel)
	})

	t.Run("colonize", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-COLONIZE-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-COLONIZE-OWN"
		root, _, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
		chdirForTest190_05(t, root)
		seedRelatedWorkflowHandoff190_06(t, "colonize", 0, "PriorColonizer-Breadth-190-06", ownSentinel)

		spy := &pheromoneCaptureInvoker190_05{}
		if _, err := dispatchRealSurveyorsWithTimeout(context.Background(), root, spy, 0); err != nil {
			t.Fatalf("dispatchRealSurveyorsWithTimeout: %v", err)
		}
		configs := spy.captured()
		if len(configs) == 0 {
			t.Fatal("expected at least one captured surveyor worker config")
		}
		assertHandoffDeliveredOnceEachChannel190_06(t, "colonize", assembledFromConfig190_05(configs[0]), buildSentinel, ownSentinel)
	})

	t.Run("seal", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-SEAL-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-SEAL-OWN"
		root, phase, state := seedHandoffColonyForBriefTests(t, buildSentinel)
		seedRelatedWorkflowHandoff190_06(t, "seal", phase.ID, "PriorSealer-Breadth-190-06", ownSentinel)

		invoker := &codex.FakeInvoker{}
		dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, 0, colony.VerificationDepthHeavy)
		if len(dispatches) == 0 {
			t.Fatal("fixture broken: expected non-empty seal review dispatches")
		}
		assertHandoffDeliveredOnceEachChannel190_06(t, "seal", assembledFromDispatch190_05(dispatches[0]), buildSentinel, ownSentinel)
	})

	t.Run("swarm", func(t *testing.T) {
		const buildSentinel = "SENTINEL-190-06-BREADTH-SWARM-BUILD"
		const ownSentinel = "SENTINEL-190-06-BREADTH-SWARM-OWN"
		root, _, _ := seedHandoffColonyForBriefTests(t, buildSentinel)
		seedRelatedWorkflowHandoff190_06(t, "swarm", 0, "PriorSwarmer-Breadth-190-06", ownSentinel)

		spy := &pheromoneCaptureInvoker190_05{}
		plan := swarmWorkerPlan{Name: "Tracker-1", Caste: "tracker", Role: "tracker", Task: "investigate the reported bug", AgentName: "aether-tracker"}
		responsePath := filepath.Join(t.TempDir(), "swarm-response.json")
		if _, _, err := invokeSwarmWorker(context.Background(), root, "some reported bug", "swarm-190-06", plan, "", responsePath, spy); err != nil {
			t.Fatalf("invokeSwarmWorker: %v", err)
		}
		configs := spy.captured()
		if len(configs) == 0 {
			t.Fatal("expected at least one captured worker config")
		}
		assertHandoffDeliveredOnceEachChannel190_06(t, "swarm", assembledFromConfig190_05(configs[0]), buildSentinel, ownSentinel)
	})

	t.Run("quick", func(t *testing.T) {
		const ownSentinel = "SENTINEL-190-06-BREADTH-QUICK-OWN"
		root, _, _ := seedHandoffColonyForBriefTests(t, "irrelevant-190-06-quick")
		chdirForTest190_05(t, root)
		seedRelatedWorkflowHandoff190_06(t, "quick", 0, "PriorQuickScout-Breadth-190-06", ownSentinel)

		spy := &pheromoneCaptureInvoker190_05{}
		origFactory := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
		t.Cleanup(func() { newQuickWorkerInvoker = origFactory })

		if _, err := runQuickScout("what does this repository do?", 0); err != nil {
			t.Fatalf("runQuickScout: %v", err)
		}
		configs := spy.captured()
		if len(configs) != 1 {
			t.Fatalf("expected exactly 1 captured worker config, got %d", len(configs))
		}
		r := assembledFromConfig190_05(configs[0])
		assembled := codex.AssembleHostedPrompt(r.contextCapsule, r.handoffSection, r.skillSection, r.pheromoneSection, r.taskBrief)
		if n := strings.Count(assembled, previousWorkerHandoffsHeading190_06); n != 1 {
			t.Fatalf("quick: assembled prompt has %d %q headings, want exactly 1 (HandoffSection is quick's sole channel):\n%s", n, previousWorkerHandoffsHeading190_06, assembled)
		}
		if n := strings.Count(assembled, ownSentinel); n != 1 {
			t.Fatalf("quick: own-workflow handoff content appears %d times, want exactly 1:\n%s", n, assembled)
		}
	})

	t.Run("oracle", func(t *testing.T) {
		const ownSentinel = "SENTINEL-190-06-BREADTH-ORACLE-OWN"
		_, _, _ = seedHandoffColonyForBriefTests(t, "irrelevant-190-06-oracle")
		seedRelatedWorkflowHandoff190_06(t, "oracle", 0, "PriorOracle-Breadth-190-06", ownSentinel)

		invoker := &codex.FakeInvoker{}
		paths := oraclePaths{Root: t.TempDir(), AgentName: "aether-oracle"}
		state := oracleStateFile{Topic: "Investigate something", Iteration: 1, MaxIterations: 5, Phase: "investigate", TargetConfidence: 80}
		plan := oraclePlanFile{}
		target := oracleQuestion{ID: "q1", Text: "What is the answer?"}
		policy := oracleAttemptPolicy{Timeout: time.Minute}
		responsePath := filepath.Join(t.TempDir(), "oracle-response.json")

		cfg := buildOracleWorkerConfig(invoker, paths, state, plan, "go", nil, nil, target, 1, policy, responsePath)
		r := assembledFromConfig190_05(cfg)
		assembled := codex.AssembleHostedPrompt(r.contextCapsule, r.handoffSection, r.skillSection, r.pheromoneSection, r.taskBrief)
		if n := strings.Count(assembled, previousWorkerHandoffsHeading190_06); n != 1 {
			t.Fatalf("oracle: assembled prompt has %d %q headings, want exactly 1 (HandoffSection is oracle's sole channel):\n%s", n, previousWorkerHandoffsHeading190_06, assembled)
		}
		if n := strings.Count(assembled, ownSentinel); n != 1 {
			t.Fatalf("oracle: own-workflow handoff content appears %d times, want exactly 1:\n%s", n, assembled)
		}
	})
}
