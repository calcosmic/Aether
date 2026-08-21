package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// pheromoneCaptureInvoker190_05 is a WorkerInvoker spy used by this file's
// tests to record the exact WorkerConfig each dispatch function hands to the
// invoker -- proving the real wiring (D-190-03-A, closed by 190-05), not a
// parallel computation that merely happens to agree with the fix.
type pheromoneCaptureInvoker190_05 struct {
	mu      sync.Mutex
	configs []codex.WorkerConfig
}

func (p *pheromoneCaptureInvoker190_05) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	p.mu.Lock()
	p.configs = append(p.configs, config)
	p.mu.Unlock()
	return (&codex.FakeInvoker{}).Invoke(ctx, config)
}

func (p *pheromoneCaptureInvoker190_05) IsAvailable(ctx context.Context) bool { return true }

func (p *pheromoneCaptureInvoker190_05) ValidateAgent(path string) error { return nil }

func (p *pheromoneCaptureInvoker190_05) captured() []codex.WorkerConfig {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]codex.WorkerConfig{}, p.configs...)
}

// seedActiveSignal190_05 seeds one active FOCUS pheromone signal carrying a
// sentinel string unique to the calling (sub)test, so counting the
// sentinel's occurrences in an assembled prompt counts delivery channels,
// not coincidental text.
func seedActiveSignal190_05(t *testing.T, sentinel string) {
	t.Helper()
	recent := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{
		{Type: "FOCUS", Content: json.RawMessage(`{"text":"` + sentinel + `"}`), Active: true, Strength: floatPtr(0.9), CreatedAt: recent},
	}}
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed pheromone signal: %v", err)
	}
}

// chdirForTest190_05 changes to root for the duration of the test and
// restores the original working directory on cleanup. Several dispatch
// paths (build, plan, colonize, quick) resolve their workspace root from the
// current working directory.
func chdirForTest190_05(t *testing.T, root string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir to test root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
}

// steeringResult190_05 captures the five fields AssemblePrompt/
// AssembleHostedPrompt join together for one worker's assembled context.
type steeringResult190_05 struct {
	contextCapsule   string
	handoffSection   string
	skillSection     string
	pheromoneSection string
	taskBrief        string
}

func assembledFromConfig190_05(cfg codex.WorkerConfig) steeringResult190_05 {
	return steeringResult190_05{
		contextCapsule:   cfg.ContextCapsule,
		handoffSection:   cfg.HandoffSection,
		skillSection:     cfg.SkillSection,
		pheromoneSection: cfg.PheromoneSection,
		taskBrief:        cfg.TaskBrief,
	}
}

func assembledFromDispatch190_05(d codex.WorkerDispatch) steeringResult190_05 {
	return steeringResult190_05{
		contextCapsule:   d.ContextCapsule,
		handoffSection:   d.HandoffSection,
		skillSection:     d.SkillSection,
		pheromoneSection: d.PheromoneSection,
		taskBrief:        d.TaskBrief,
	}
}

// TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule is the primary
// regression lock for D-190-03-A, closed by Phase 190 Plan 05.
//
// BEFORE this plan's fix, executeCodexBuildDispatches (the native/direct
// build dispatch path -- the one `aether build <phase>` with no
// --plan-only uses, including every autopilot /ant-run build) set BOTH
// ContextCapsule (resolveCodexWorkerContext(), which already renders
// "## Pheromone Signals" unconditionally whenever a signal is active) AND a
// separately-resolved PheromoneSection on every dispatch.
// AssemblePrompt/AssembleHostedPrompt join both as two independent,
// both-included parts, so an active signal reached a native-dispatched
// worker's prompt twice, under two different headings.
//
// This test captures the REAL WorkerConfig executeCodexBuildDispatches hands
// to the invoker (via a spy, not a parallel computation that merely happens
// to agree with the fix) and assembles the prompt exactly as
// AssembleHostedPrompt does in production, then counts.
func TestNativeDispatchPheromoneStaysExactlyOnceViaCapsule(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	chdirForTest190_05(t, root)

	goal := "Prove the native dispatch path keeps pheromone delivery to exactly once"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:              1,
					Name:            "Native dispatch pheromone",
					Description:     "Native/direct dispatch must keep pheromone delivery to exactly once",
					Status:          colony.PhaseReady,
					SuccessCriteria: []string{"Pheromone content reaches the worker exactly once"},
				},
			},
		},
	})

	const sentinel = "SENTINEL-PHEROMONE-ONE-HOME-190-05"
	seedActiveSignal190_05(t, sentinel)

	phase := colony.Phase{ID: 1, Name: "Native dispatch pheromone"}
	dispatches := []codexBuildDispatch{{Name: "NativeWorker-1", Caste: "builder", Task: "do the thing"}}

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "NativeWorker-1", "do the thing", 1); err != nil {
		t.Fatalf("failed to seed spawn tree: %v", err)
	}

	spy := &pheromoneCaptureInvoker190_05{}
	results, _, _, err := executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), spy, colony.ModeInRepo, 0, 3, false, nil)
	if err != nil {
		t.Fatalf("executeCodexBuildDispatches failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one dispatch result")
	}

	configs := spy.captured()
	if len(configs) != 1 {
		t.Fatalf("expected exactly 1 captured worker config, got %d", len(configs))
	}
	cfg := configs[0]

	if !strings.Contains(cfg.ContextCapsule, sentinel) {
		t.Fatalf("fixture broken: the native path's own capsule does not carry the seeded signal, so a clean PheromoneSection would prove no-home, not one-home:\n%s", cfg.ContextCapsule)
	}

	assertPheromoneOnce190_05(t, "build native", assembledFromConfig190_05(cfg), sentinel)
}

// assertPheromoneOnce190_05 assembles a prompt from the four steering fields
// (via the same production AssembleHostedPrompt every hosted platform
// dispatcher calls) and asserts the active signal reaches it exactly once --
// never twice (D-190-03-A), never zero (the "quiet drop" trap).
// pheromoneHeadingSubstring190_05 is the text shared by both heading forms
// this codebase uses for pheromone content: the capsule's own
// "## Pheromone Signals" (cmd/colony_prime_context.go:571) and
// resolvePheromoneSection's raw "### Active Pheromone Signals"
// (cmd/codex_build.go). composeBuildManifestBrief's own rewrap logic
// (cmd/codex_build.go, includeSteeringSections branch) treats these as the
// SAME semantic section under two different possible headings, not two
// different sections -- this constant matches that established convention
// so the breadth test does not fail quick/oracle (whose raw
// PheromoneSection is never rewrapped) for a heading-text difference that
// is not a duplication bug.
const pheromoneHeadingSubstring190_05 = "Pheromone Signals"

func assertPheromoneOnce190_05(t *testing.T, label string, r steeringResult190_05, sentinel string) {
	t.Helper()
	assembled := codex.AssembleHostedPrompt(r.contextCapsule, r.handoffSection, r.skillSection, r.pheromoneSection, r.taskBrief)
	if n := strings.Count(assembled, pheromoneHeadingSubstring190_05); n != 1 {
		t.Fatalf("%s: assembled prompt has %d %q headings, want exactly 1\ncapsule=%q\npheromoneSection=%q", label, n, pheromoneHeadingSubstring190_05, r.contextCapsule, r.pheromoneSection)
	}
	if n := strings.Count(assembled, sentinel); n != 1 {
		t.Fatalf("%s: assembled prompt carries the active signal's own text %d times, want exactly 1:\n%s", label, n, assembled)
	}
}

// TestEightCommandsDeliverPheromoneExactlyOnce is the breadth test for
// D-190-03-A / 190-05: every command that assembles a worker prompt through
// AssemblePrompt/AssembleHostedPrompt must deliver an active pheromone
// signal exactly once, whether through the shared capsule
// (resolveCodexWorkerContext(), for build/continue/plan/colonize/seal/swarm)
// or through the dedicated PheromoneSection channel (for quick/oracle, whose
// capsules are custom and do not render pheromones at all). Never twice.
// Never zero -- a caller whose capsule does not embed pheromones and whose
// PheromoneSection was wrongly stripped would silently drop to zero, which
// is worse than the duplication bug this closes.
func TestEightCommandsDeliverPheromoneExactlyOnce(t *testing.T) {
	// Every case below calls seedHandoffColonyForBriefTests, which sets the
	// package-global `store` and relies on its CALLER to restore it
	// (the same contract TestPlanOnlyDispatchesCarryNoHandoffSection follows
	// via saveGlobalsCmd) -- its own t.Cleanup only removes the temp
	// directory, leaving `store` a dangling pointer into a deleted path.
	// saveGlobalsCmd here restores `store` once, after every subtest below
	// has finished, so this test does not leak a broken global `store` into
	// unrelated tests that run later in the same package (caught the hard
	// way: without this, TestBuildDispatchStartsHeartbeatMonitor started
	// failing with "storage: open lock file ... no such file or directory"
	// whenever it ran after this test in the full suite).
	saveGlobalsCmd(t)

	cases := []struct {
		name string
		run  func(t *testing.T, sentinel string) steeringResult190_05
	}{
		{"build_native", runBuildNativeCase190_05},
		{"continue_review", runContinueReviewCase190_05},
		{"continue_watcher", runContinueWatcherCase190_05},
		{"plan", runPlanCase190_05},
		{"colonize", runColonizeCase190_05},
		{"seal", runSealCase190_05},
		{"swarm", runSwarmCase190_05},
		{"quick", runQuickCase190_05},
		{"oracle", runOracleCase190_05},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			sentinel := "SENTINEL-PHEROMONE-190-05-" + strings.ToUpper(tc.name)
			result := tc.run(t, sentinel)
			if !strings.Contains(result.contextCapsule, sentinel) && !strings.Contains(result.pheromoneSection, sentinel) {
				t.Fatalf("fixture broken: neither the capsule nor a dedicated PheromoneSection carries the seeded signal for %s -- the case below cannot prove one-home", tc.name)
			}
			assertPheromoneOnce190_05(t, tc.name, result, sentinel)
		})
	}
}

func runBuildNativeCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, phase, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-build")
	chdirForTest190_05(t, root)
	seedActiveSignal190_05(t, sentinel)

	dispatches := []codexBuildDispatch{{Name: "NativeWorker-1", Caste: "builder", Task: "do the thing"}}
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "NativeWorker-1", "do the thing", 1); err != nil {
		t.Fatalf("seed spawn tree: %v", err)
	}

	spy := &pheromoneCaptureInvoker190_05{}
	if _, _, _, err := executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), spy, colony.ModeInRepo, 0, 3, false, nil); err != nil {
		t.Fatalf("executeCodexBuildDispatches: %v", err)
	}
	configs := spy.captured()
	if len(configs) == 0 {
		t.Fatal("expected at least one captured worker config")
	}
	return assembledFromConfig190_05(configs[0])
}

func runContinueReviewCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, phase, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-continue-review")
	seedActiveSignal190_05(t, sentinel)

	invoker := &codex.FakeInvoker{}
	dispatches := plannedContinueReviewDispatches(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, invoker, 0, colony.VerificationDepthHeavy)
	if len(dispatches) == 0 {
		t.Fatal("fixture broken: expected non-empty continue review dispatches at heavy depth")
	}
	return assembledFromDispatch190_05(dispatches[0])
}

func runContinueWatcherCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, phase, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-continue-watcher")
	seedActiveSignal190_05(t, sentinel)

	invoker := &codex.FakeInvoker{}
	d := plannedContinueWatcherDispatch(root, phase, codexContinueManifest{}, nil, codexClaimVerification{}, codexWatcherVerification{}, invoker, 0)
	return assembledFromDispatch190_05(d)
}

func runPlanCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, _, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-plan")
	chdirForTest190_05(t, root)
	seedActiveSignal190_05(t, sentinel)

	spy := &pheromoneCaptureInvoker190_05{}
	if _, err := dispatchRealPlanningWorkersWithIterationContext(context.Background(), root, codexSurveyContext{}, spy, 0, "", "Plan something"); err != nil {
		t.Fatalf("dispatchRealPlanningWorkersWithIterationContext: %v", err)
	}
	configs := spy.captured()
	if len(configs) == 0 {
		t.Fatal("expected at least one captured planning worker config")
	}
	return assembledFromConfig190_05(configs[0])
}

func runColonizeCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, _, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-colonize")
	chdirForTest190_05(t, root)
	seedActiveSignal190_05(t, sentinel)

	spy := &pheromoneCaptureInvoker190_05{}
	if _, err := dispatchRealSurveyorsWithTimeout(context.Background(), root, spy, 0); err != nil {
		t.Fatalf("dispatchRealSurveyorsWithTimeout: %v", err)
	}
	configs := spy.captured()
	if len(configs) == 0 {
		t.Fatal("expected at least one captured surveyor worker config")
	}
	return assembledFromConfig190_05(configs[0])
}

func runSealCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, phase, state := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-seal")
	seedActiveSignal190_05(t, sentinel)

	invoker := &codex.FakeInvoker{}
	dispatches := plannedSealFinalReviewDispatches(root, state, phase, invoker, 0, colony.VerificationDepthHeavy)
	if len(dispatches) == 0 {
		t.Fatal("fixture broken: expected non-empty seal review dispatches at heavy depth")
	}
	return assembledFromDispatch190_05(dispatches[0])
}

func runSwarmCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, _, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-swarm")
	seedActiveSignal190_05(t, sentinel)

	spy := &pheromoneCaptureInvoker190_05{}
	plan := swarmWorkerPlan{Name: "Tracker-1", Caste: "tracker", Role: "tracker", Task: "investigate the reported bug", AgentName: "aether-tracker"}
	responsePath := filepath.Join(t.TempDir(), "swarm-response.json")
	if _, _, err := invokeSwarmWorker(context.Background(), root, "some reported bug", "swarm-190-05", plan, "", responsePath, spy); err != nil {
		t.Fatalf("invokeSwarmWorker: %v", err)
	}
	configs := spy.captured()
	if len(configs) != 1 {
		t.Fatalf("expected exactly 1 captured worker config, got %d", len(configs))
	}
	return assembledFromConfig190_05(configs[0])
}

func runQuickCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	root, _, _ := seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-quick")
	chdirForTest190_05(t, root)
	seedActiveSignal190_05(t, sentinel)

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
	cfg := configs[0]
	if strings.Contains(cfg.ContextCapsule, "## Pheromone Signals") {
		t.Fatalf("quick's capsule (renderQuickContextCapsule) now unexpectedly renders its own Pheromone Signals section -- if this changed intentionally, PheromoneSection must be removed at this call site too (see resolvePheromoneSection's doc comment) or this becomes a duplicate")
	}
	return assembledFromConfig190_05(cfg)
}

func runOracleCase190_05(t *testing.T, sentinel string) steeringResult190_05 {
	_, _, _ = seedHandoffColonyForBriefTests(t, "irrelevant-handoff-sentinel-oracle")
	seedActiveSignal190_05(t, sentinel)

	invoker := &codex.FakeInvoker{}
	paths := oraclePaths{Root: t.TempDir(), AgentName: "aether-oracle"}
	state := oracleStateFile{Topic: "Investigate something", Iteration: 1, MaxIterations: 5, Phase: "investigate", TargetConfidence: 80}
	plan := oraclePlanFile{}
	target := oracleQuestion{ID: "q1", Text: "What is the answer?"}
	policy := oracleAttemptPolicy{Timeout: time.Minute}
	responsePath := filepath.Join(t.TempDir(), "oracle-response.json")

	cfg := buildOracleWorkerConfig(invoker, paths, state, plan, "go", nil, nil, target, 1, policy, responsePath)
	if strings.Contains(cfg.ContextCapsule, "## Pheromone Signals") {
		t.Fatalf("oracle's capsule (renderOracleContextCapsule) now unexpectedly renders its own Pheromone Signals section -- if this changed intentionally, PheromoneSection must be removed at this call site too (see resolvePheromoneSection's doc comment) or this becomes a duplicate")
	}
	return assembledFromConfig190_05(cfg)
}
