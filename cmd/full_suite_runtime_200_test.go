package cmd

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFullSuiteRuntime200PartitionExactOnce(t *testing.T) {
	discovered := []string{"TestDelta", "TestAlpha", "TestCharlie", "TestBravo"}
	serial := map[string]struct{}{"TestCharlie": {}}
	costs := map[string]time.Duration{
		"TestAlpha":   4 * time.Second,
		"TestBravo":   time.Second,
		"TestCharlie": 9 * time.Second,
		"TestDelta":   3 * time.Second,
	}

	first, err := planFullSuiteLanes(discovered, serial, costs, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := planFullSuiteLanes(discovered, serial, costs, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("full-suite partition is not deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if err := validateFullSuitePlan(discovered, first); err != nil {
		t.Fatalf("valid exact-once partition rejected: %v", err)
	}
	if len(first) != 3 || first[0].Name != fullSuiteSerialLaneName || !first[0].Serial || !reflect.DeepEqual(first[0].Tests, []string{"TestCharlie"}) {
		t.Fatalf("serial lane = %#v, want the one explicitly classified test", first)
	}

	duplicated := append([]fullSuiteLane(nil), first...)
	duplicated[1].Tests = append(duplicated[1].Tests, "TestCharlie")
	if err := validateFullSuitePlan(discovered, duplicated); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate plan validation error = %v, want explicit duplicate diagnostic", err)
	}
	omitted := append([]fullSuiteLane(nil), first...)
	omitted[0].Tests = nil
	if err := validateFullSuitePlan(discovered, omitted); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("omitted plan validation error = %v, want explicit missing diagnostic", err)
	}
	if _, err := planFullSuiteLanes(append(discovered, "TestAlpha"), serial, costs, 2); err == nil {
		t.Fatal("duplicate discovery entry was silently accepted")
	}
}

func TestFullSuiteRuntime200PreservesFocusedRun(t *testing.T) {
	tests := []struct {
		name string
		in   fullSuiteInvocation
		want bool
	}{
		{name: "exact unfiltered count one", in: fullSuiteInvocation{Count: 1}, want: true},
		{name: "focused selector", in: fullSuiteInvocation{Count: 1, Run: "^TestOne$", RunExplicit: true}},
		{name: "explicit empty selector", in: fullSuiteInvocation{Count: 1, RunExplicit: true}},
		{name: "discovery listing", in: fullSuiteInvocation{Count: 1, List: "^Test", ListExplicit: true}},
		{name: "benchmark", in: fullSuiteInvocation{Count: 1, Bench: ".", BenchExplicit: true}},
		{name: "fuzzing", in: fullSuiteInvocation{Count: 1, Fuzz: "FuzzOne", FuzzExplicit: true}},
		{name: "skip filter", in: fullSuiteInvocation{Count: 1, Skip: "Slow", SkipExplicit: true}},
		{name: "profile output", in: fullSuiteInvocation{Count: 1, Profiled: true}},
		{name: "explicit parallelism", in: fullSuiteInvocation{Count: 1, Parallel: "4", ParallelSet: true}},
		{name: "caller requested repetitions", in: fullSuiteInvocation{Count: 2}},
		{name: "child recursion marker", in: fullSuiteInvocation{Count: 1, ShardMarker: "123:parallel-01"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRunFullSuiteController(tc.in); got != tc.want {
				t.Fatalf("shouldRunFullSuiteController(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFullSuiteRuntime200PropagatesFailure(t *testing.T) {
	lanes := []fullSuiteLane{
		{Name: "parallel-01", Tests: []string{"TestAlpha"}},
		{Name: "parallel-02", Tests: []string{"TestBravo"}},
	}
	runner := func(_ context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
		if request.Lane.Name == "parallel-02" {
			return fullSuiteChildResult{
				Output:   "bravo user output\npanic: deliberate child diagnostic\nWARNING: DATA RACE",
				Executed: []string{"TestBravo"},
				Err:      errors.New("exit status 2"),
			}
		}
		return fullSuiteChildResult{Output: "alpha user output\n", Executed: append([]string(nil), request.Lane.Tests...)}
	}

	report, err := runFullSuiteLanes(context.Background(), "/current/cmd.test", lanes, 2, runner)
	if err == nil {
		t.Fatal("child failure was swallowed")
	}
	for _, want := range []string{"parallel-02", "exit status 2", "deliberate child diagnostic", "DATA RACE"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("failure %q omitted actionable child detail %q", err, want)
		}
	}
	if report.Passed {
		t.Fatalf("failed child produced passing report: %+v", report)
	}
	var rendered strings.Builder
	writeFullSuiteReport(&rendered, report)
	output := rendered.String()
	for _, want := range []string{"alpha user output", "bravo user output", "deliberate child diagnostic", "DATA RACE"} {
		if !strings.Contains(output, want) {
			t.Fatalf("rendered report omitted child output %q:\n%s", want, output)
		}
	}
	if strings.Index(output, "alpha user output") > strings.Index(output, "bravo user output") {
		t.Fatalf("concurrent child output was not emitted in deterministic lane order:\n%s", output)
	}
}

func TestFullSuiteRuntime200UsesCurrentBinary(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	lanes := []fullSuiteLane{{Name: "parallel-01", Tests: []string{"TestAlpha", "TestBravo"}}}
	var observed fullSuiteChildRequest
	runner := func(_ context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
		observed = request
		return fullSuiteChildResult{Executed: append([]string(nil), request.Lane.Tests...)}
	}
	report, err := runFullSuiteLanes(context.Background(), executable, lanes, 1, runner)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Executable != executable {
		t.Fatalf("child executable = %q, want current race/build-instrumented binary %q", observed.Executable, executable)
	}
	if strings.Contains(strings.Join(observed.Args, " "), "go test") {
		t.Fatalf("child recursively invoked go test: %v", observed.Args)
	}
	if !strings.Contains(strings.Join(observed.Args, " "), "-test.run=^(TestAlpha|TestBravo)$") {
		t.Fatalf("child args do not carry the exact anchored lane selector: %v", observed.Args)
	}
	if report.Discovered != 2 || report.Executed != 2 || !report.Passed {
		t.Fatalf("current-binary report = %+v, want 2/2 passing", report)
	}

	realLane := []fullSuiteLane{{Name: "probe", Tests: []string{"TestFullSuiteRuntime200CurrentBinaryProbe"}}}
	realReport, err := runFullSuiteLanes(context.Background(), executable, realLane, 1, runFullSuiteChildProcess)
	if err != nil {
		t.Fatalf("current test binary could not execute an exact focused child: %v", err)
	}
	if realReport.Discovered != 1 || realReport.Executed != 1 || !realReport.Passed {
		t.Fatalf("real current-binary report = %+v, want 1/1 passing", realReport)
	}

	discoveryContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	discovered, err := discoverFullSuiteTests(discoveryContext, executable)
	if err != nil {
		t.Fatalf("current test binary discovery failed: %v", err)
	}
	foundProbe := false
	seen := make(map[string]struct{}, len(discovered))
	for _, testName := range discovered {
		if _, duplicate := seen[testName]; duplicate {
			t.Fatalf("current test binary discovery duplicated %s", testName)
		}
		seen[testName] = struct{}{}
		if testName == "TestFullSuiteRuntime200CurrentBinaryProbe" {
			foundProbe = true
		}
	}
	if !foundProbe {
		t.Fatalf("current test binary discovery omitted the compiled probe from %d top-level entries", len(discovered))
	}
}

func TestFullSuiteRuntime200CurrentBinaryProbe(t *testing.T) {
	if strings.TrimSpace(os.Getenv(fullSuiteShardEnv)) == "" {
		return
	}
}

func TestFullSuiteRuntime200IsolatesChildEnvironment(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(isolatedProcessHubEnv, t.TempDir())
	t.Setenv("AETHER_ROOT", "/stale/parent/repository")
	t.Setenv("COLONY_DATA_DIR", "/stale/parent/repository/.aether/data")

	lanes := []fullSuiteLane{
		{Name: "isolation-alpha", Tests: []string{"TestFullSuiteRuntime200ChildEnvironmentProbeAlpha"}},
		{Name: "isolation-beta", Tests: []string{"TestFullSuiteRuntime200ChildEnvironmentProbeBeta"}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	report, err := runFullSuiteLanes(ctx, executable, lanes, 2, runFullSuiteChildProcess)
	if err != nil {
		t.Fatalf("isolated current-binary children failed: %v", err)
	}

	hubs := map[string]bool{}
	for _, lane := range report.Lanes {
		for _, line := range strings.Split(lane.Output, "\n") {
			const marker = "FULL-SUITE-PROBE-HUB="
			index := strings.Index(line, marker)
			if index >= 0 {
				hubs[strings.TrimSpace(line[index+len(marker):])] = true
			}
		}
	}
	if len(hubs) != len(lanes) {
		t.Fatalf("child hubs = %v, want %d distinct private hubs", hubs, len(lanes))
	}
}

func TestFullSuiteRuntime200SerializesSharedResources(t *testing.T) {
	want := map[string]string{
		"TestColonyStateWriteAllowlistOnlyShrinks": "fixed checked-in allowlist",
		"TestCurrentVocabulary199":                 "live tracked checkout inventory",
		"TestNextActionNeverHardcoded":             "fixed checked-in allowlist",
		"TestOrphanAllowlistOnlyShrinks":           "fixed checked-in allowlist",
		"TestPhase199GateReceipt":                  "live repository receipt",
		"TestWorktreeAllocateAgentPhase":           "source checkout worktree registration",
		"TestWorktreeAllocateAuditLog":             "source checkout worktree registration",
		"TestWorktreeAllocateHumanBranch":          "source checkout worktree registration",
		"TestWorktreeAllocateMergedBranchAllowed":  "source checkout worktree registration",
	}
	inventory := fullSuiteSerialInventory()
	if len(inventory) != len(want) {
		t.Fatalf("serial inventory = %v, want exactly %v", inventory, want)
	}
	for testName, evidence := range want {
		reason, ok := inventory[testName]
		if !ok {
			t.Errorf("evidence-backed shared-resource test %s is not serialized", testName)
			continue
		}
		if !strings.Contains(reason, evidence) {
			t.Errorf("serial reason for %s = %q, want concrete evidence containing %q", testName, reason, evidence)
		}
	}

	discovered := []string{"TestPatrolCheckAllHealthy"}
	for testName := range want {
		discovered = append(discovered, testName)
	}
	lanes, err := planFullSuiteLanes(discovered, fullSuiteSerialTests(inventory), nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(lanes) < 2 || lanes[0].Name != fullSuiteSerialLaneName || !lanes[0].Serial {
		t.Fatalf("serial lane missing or not first: %#v", lanes)
	}
	wantSerial := make([]string, 0, len(want))
	for testName := range want {
		wantSerial = append(wantSerial, testName)
	}
	for i := 0; i < len(wantSerial); i++ {
		for j := i + 1; j < len(wantSerial); j++ {
			if wantSerial[j] < wantSerial[i] {
				wantSerial[i], wantSerial[j] = wantSerial[j], wantSerial[i]
			}
		}
	}
	if !reflect.DeepEqual(lanes[0].Tests, wantSerial) {
		t.Fatalf("serialized tests = %v, want exact narrow inventory %v", lanes[0].Tests, wantSerial)
	}
	for _, testName := range lanes[0].Tests {
		if testName == "TestPatrolCheckAllHealthy" {
			t.Fatal("ordinary isolated test was broadened into the serial lane")
		}
	}
}

func TestFullSuiteRuntime200AdversarialOrder(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	probes := []string{
		"TestRepositoryTestBinding200",
		"TestWorktreeFixtureRestoresRepositoryAuthority200",
		"TestHookPreToolUseBlocksProtectedPath",
		"TestPatrolCheckAllHealthy",
		"TestPlanningStateCurrentRoundTripIsStable",
		"TestBoundaryBuildFixturesUseCanonicalAuthority200",
	}
	for _, seed := range []string{"20055", "-20055"} {
		seed := seed
		t.Run(seed, func(t *testing.T) {
			lane := fullSuiteLane{Name: "adversarial-" + seed, Tests: append([]string(nil), probes...)}
			request := fullSuiteChildRequestForLane(executable, lane)
			request.Args = append(request.Args, "-test.shuffle="+seed)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			result := runFullSuiteChildProcess(ctx, request)
			if result.Err != nil {
				t.Fatalf("adversarial current-binary order failed: %v\n%s", result.Err, result.Output)
			}
			if err := validateFullSuiteExecution(probes, result.Executed); err != nil {
				t.Fatalf("adversarial execution accounting: %v\n%s", err, result.Output)
			}
		})
	}
}

func TestFullSuiteRuntime200ChildEnvironmentProbeAlpha(t *testing.T) {
	fullSuiteRuntime200AssertChildEnvironment(t)
}

func TestFullSuiteRuntime200ChildEnvironmentProbeBeta(t *testing.T) {
	fullSuiteRuntime200AssertChildEnvironment(t)
}

func fullSuiteRuntime200AssertChildEnvironment(t *testing.T) {
	t.Helper()
	if strings.TrimSpace(os.Getenv(fullSuiteShardEnv)) == "" {
		return
	}
	for _, name := range []string{"AETHER_ROOT", "COLONY_DATA_DIR"} {
		if value, ok := os.LookupEnv(name); ok {
			t.Fatalf("isolated full-suite child inherited %s=%q", name, value)
		}
	}
	assertIsolatedProcessChildHub(t)
	t.Logf("FULL-SUITE-PROBE-HUB=%s", os.Getenv(isolatedProcessHubEnv))
}

func TestFullSuiteRuntime200ReportsCompleteAccounting(t *testing.T) {
	report := fullSuiteRunReport{
		Discovered: 3,
		Executed:   3,
		Passed:     true,
		Lanes: []fullSuiteLaneReport{
			{Name: "parallel-01", Planned: 2, Executed: 2, Duration: time.Second, Successful: true, Output: "alpha output\n"},
			{Name: "parallel-02", Planned: 1, Executed: 1, Duration: 3 * time.Second, Successful: true, Output: "beta output\n"},
		},
	}
	var rendered strings.Builder
	writeFullSuiteReport(&rendered, report)
	output := rendered.String()
	for _, want := range []string{
		"FULL-SUITE PASS discovered=3 executed=3 lanes=2",
		"FULL-SUITE lane=parallel-01 status=PASS planned=2 executed=2 duration=1s",
		"FULL-SUITE lane=parallel-02 status=PASS planned=1 executed=1 duration=3s",
		"FULL-SUITE slowest rank=1 lane=parallel-02 duration=3s",
		"FULL-SUITE slowest rank=2 lane=parallel-01 duration=1s",
		"alpha output",
		"beta output",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("complete accounting omitted %q:\n%s", want, output)
		}
	}
	firstOutput := strings.Index(output, "FULL-SUITE output-begin")
	lastAccounting := strings.Index(output, "FULL-SUITE slowest rank=2")
	if firstOutput < 0 || lastAccounting < 0 || firstOutput < lastAccounting {
		t.Fatalf("child output obscured the complete accounting preamble:\n%s", output)
	}

	classified, err := planFullSuiteLanes(
		[]string{"TestClassicContractPhase200CausalExecution", "TestPlanningGapEdgeAccounting200", "TestLight"},
		nil,
		fullSuiteMeasuredCosts(),
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	heavyNames := map[string]bool{}
	for _, lane := range classified {
		if !lane.Heavy {
			continue
		}
		for _, testName := range lane.Tests {
			heavyNames[testName] = true
		}
	}
	for _, testName := range []string{"TestClassicContractPhase200CausalExecution", "TestPlanningGapEdgeAccounting200"} {
		if !heavyNames[testName] {
			t.Errorf("measured heavyweight %s was not assigned to a heavy affinity lane: %#v", testName, classified)
		}
	}
	if heavyNames["TestLight"] {
		t.Fatalf("unmeasured light test was put in a heavy affinity lane: %#v", classified)
	}

	lanes := []fullSuiteLane{
		{Name: "heavy-01", Heavy: true, Tests: []string{"TestHeavyOne"}},
		{Name: "heavy-02", Heavy: true, Tests: []string{"TestHeavyTwo"}},
		{Name: "heavy-03", Heavy: true, Tests: []string{"TestHeavyThree"}},
		{Name: "heavy-04", Heavy: true, Tests: []string{"TestHeavyFour"}},
		{Name: "light-01", Tests: []string{"TestLightOne"}},
		{Name: "light-02", Tests: []string{"TestLightTwo"}},
		{Name: "light-03", Tests: []string{"TestLightThree"}},
	}
	var activeHeavy atomic.Int32
	var maximumHeavy atomic.Int32
	var lightRuns atomic.Int32
	var lightOverlappedHeavy atomic.Bool
	heavyReady := make(chan struct{})
	releaseHeavy := make(chan struct{})
	var readyOnce sync.Once
	var releaseOnce sync.Once
	runner := func(_ context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
		if request.Lane.Heavy {
			active := activeHeavy.Add(1)
			for {
				maximum := maximumHeavy.Load()
				if active <= maximum || maximumHeavy.CompareAndSwap(maximum, active) {
					break
				}
			}
			if active == fullSuiteHeavyWorkers {
				readyOnce.Do(func() { close(heavyReady) })
			}
			select {
			case <-releaseHeavy:
			case <-time.After(time.Second):
				return fullSuiteChildResult{Err: errors.New("light lane never overlapped the heavy queue")}
			}
			activeHeavy.Add(-1)
			if !strings.Contains(strings.Join(request.Args, " "), "-test.parallel=1") {
				return fullSuiteChildResult{Err: errors.New("heavy lane did not receive serial top-level scheduling")}
			}
		} else {
			select {
			case <-heavyReady:
				if activeHeavy.Load() > 0 {
					lightOverlappedHeavy.Store(true)
				}
				releaseOnce.Do(func() { close(releaseHeavy) })
			case <-time.After(time.Second):
				return fullSuiteChildResult{Err: errors.New("heavy workers did not start while light queue drained")}
			}
			lightRuns.Add(1)
		}
		return fullSuiteChildResult{Executed: append([]string(nil), request.Lane.Tests...)}
	}
	scheduled, err := runFullSuiteLanes(context.Background(), "/current/cmd.test", lanes, 5, runner)
	if err != nil {
		t.Fatal(err)
	}
	if !scheduled.Passed || scheduled.Executed != len(lanes) {
		t.Fatalf("scheduled accounting = %+v, want every heavy and light lane", scheduled)
	}
	if maximumHeavy.Load() != fullSuiteHeavyWorkers {
		t.Fatalf("maximum concurrent heavy lanes = %d, want %d", maximumHeavy.Load(), fullSuiteHeavyWorkers)
	}
	if lightRuns.Load() != 3 {
		t.Fatalf("light lanes executed = %d, want 3 while heavy queue drains", lightRuns.Load())
	}
	if !lightOverlappedHeavy.Load() {
		t.Fatal("light work did not overlap the bounded heavy queue")
	}
}
