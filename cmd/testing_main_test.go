package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestMain saves all package-level globals before running tests and restores
// them after. This prevents test pollution where one test's store/stdout/stderr
// assignment leaks into subsequent tests. Belt-and-suspenders with per-test
// cleanup via saveGlobals.
func TestMain(m *testing.M) {
	// The suite's outcome must not depend on which terminal launches it:
	// platform detection otherwise sniffs the process tree, so the same
	// test could pass under a Codex shell and fail under Claude Code. Pin
	// the historical codex baseline; tests exercising claude/opencode
	// rendering pin their own platform with t.Setenv, which overrides this.
	if os.Getenv("AETHER_PLATFORM") == "" {
		os.Setenv("AETHER_PLATFORM", "codex")
	}
	if !flag.Parsed() {
		flag.Parse()
	}
	if shouldRunFullSuiteController(currentFullSuiteInvocation()) {
		os.Exit(runFullSuiteController())
	}
	origOutputMode, hadOutputMode := os.LookupEnv("AETHER_OUTPUT_MODE")
	origHivePolicy, hadHivePolicy := os.LookupEnv(hivePolicyEnv)
	_ = os.Setenv("AETHER_OUTPUT_MODE", "json")
	_ = os.Setenv(hivePolicyEnv, "promote")

	// Spawn-line model tags resolve ANTHROPIC_DEFAULT_*_MODEL at render
	// time; a developer whose shell redirects those slots would otherwise
	// see every golden fixture with a spawn line fail. Neutralize for the
	// suite — tests that exercise the override use t.Setenv.
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_SONNET_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")

	// Isolate the whole suite from the developer's real hub. This TestMain
	// enables hive promotion above, and the v1.25 multi-agent review found
	// TestSealPromoteInstincts had promoted its fixture into the user's actual
	// ~/.aether/hive/wisdom.json on every full-suite run. AETHER_HUB_DIR is
	// the supported override; individual tests may still t.Setenv their own.
	var testHubDir string
	if os.Getenv("AETHER_HUB_DIR") == "" {
		if dir, err := os.MkdirTemp("", "aether-test-hub-"); err == nil {
			testHubDir = dir
			_ = os.Setenv("AETHER_HUB_DIR", dir)
			// Make the isolated hub minimally valid so tests that check
			// "hub installed" (version.json present) behave as they would on
			// a machine with Aether installed — without touching the real one.
			// The version mirrors the source checkout's so stale-publish
			// detection sees binary and hub in agreement.
			hubVersion := `{"version":"0.0.0-test"}`
			if data, err := os.ReadFile(filepath.Join("..", ".aether", "version.json")); err == nil {
				hubVersion = string(data)
			}
			_ = os.WriteFile(filepath.Join(dir, "version.json"), []byte(hubVersion), 0644)
			_ = os.MkdirAll(filepath.Join(dir, "system"), 0755)
		}
	}

	origStore := store
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origTracer := tracer
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath

	code := m.Run()

	// Clean up git worktrees and branches created by tests.
	// Tests like TestWorktreeAllocateAuditLog create real git worktrees
	// that persist after the test suite finishes.
	cleanupTestWorktrees()

	store = origStore
	stdout = origStdout
	stderr = origStderr
	flagTypeFilter = origFlagType
	flagStatusFilter = origFlagStatus
	flagListJSON = origFlagListJSON
	historyJSON = origHistoryJSON
	phaseJSON = origPhaseJSON
	historyLimit = origHistoryLimit
	historyFilter = origHistoryFilter
	phaseNumber = origPhaseNumber
	tracer = origTracer
	continueContextUpdater = origContinueContextUpdater
	continueSignalHousekeeper = origContinueSignalHousekeeper
	newCodexWorkerInvoker = origNewCodexWorkerInvoker
	activeBuildCeremony = origActiveBuildCeremony
	narratorLookPath = origNarratorLookPath
	narratorCommandContext = origNarratorCommandContext
	narratorRuntimePath = origNarratorRuntimePath
	if hadOutputMode {
		_ = os.Setenv("AETHER_OUTPUT_MODE", origOutputMode)
	} else {
		_ = os.Unsetenv("AETHER_OUTPUT_MODE")
	}
	if hadHivePolicy {
		_ = os.Setenv(hivePolicyEnv, origHivePolicy)
	} else {
		_ = os.Unsetenv(hivePolicyEnv)
	}
	if testHubDir != "" {
		_ = os.Unsetenv("AETHER_HUB_DIR")
		_ = os.RemoveAll(testHubDir)
	}

	os.Exit(code)
}

const (
	fullSuiteShardEnv        = "AETHER_CMD_FULL_SUITE_SHARD"
	fullSuiteSerialLaneName  = "serial-shared-checkout"
	fullSuiteLogicalShards   = 48
	fullSuiteWorkers         = 12
	fullSuiteHeavyWorkers    = 6
	fullSuiteHeavyLaneBudget = 3 * time.Minute
	fullSuiteHeavyThreshold  = 8 * time.Second
	fullSuiteChildParallel   = 3
	fullSuiteChildProcs      = 3
	// Fallback ceilings used only when the caller passed no readable
	// -test.timeout. The live ceilings come from resolveFullSuiteCeilings,
	// which tracks the outer go tool's deadline so the controller stops
	// ORDERLY (full per-lane accounting) just before the tool would SIGQUIT.
	fullSuiteChildTimeoutFallback   = 9 * time.Minute
	fullSuiteCommandTimeoutFallback = 9*time.Minute + 15*time.Second
	fullSuiteOverallTimeoutFallback = 9*time.Minute + 30*time.Second
)

// Live per-lane ceilings, defaulted to the fallbacks and overwritten by
// runFullSuiteController from resolveFullSuiteCeilings. Package-level so the
// per-lane helpers (and direct-call tests) read a single source.
var (
	fullSuiteChildTimeout   = fullSuiteChildTimeoutFallback
	fullSuiteCommandTimeout = fullSuiteCommandTimeoutFallback
)

type fullSuiteInvocation struct {
	Run           string
	RunExplicit   bool
	List          string
	ListExplicit  bool
	Bench         string
	BenchExplicit bool
	Fuzz          string
	FuzzExplicit  bool
	Skip          string
	SkipExplicit  bool
	Count         int
	Short         bool
	FailFast      bool
	CPU           string
	Parallel      string
	ParallelSet   bool
	Shuffle       string
	Profiled      bool
	ShardMarker   string
}

type fullSuiteLane struct {
	Name          string
	Serial        bool
	Heavy         bool
	Tests         []string
	EstimatedCost time.Duration
}

type fullSuiteChildRequest struct {
	Executable string
	Lane       fullSuiteLane
	Args       []string
	Env        []string
}

type fullSuiteChildResult struct {
	Output   string
	Executed []string
	Duration time.Duration
	Err      error
}

type fullSuiteLaneReport struct {
	Name       string
	Planned    int
	Executed   int
	Duration   time.Duration
	Successful bool
	Output     string
}

type fullSuiteRunReport struct {
	Discovered int
	Executed   int
	Passed     bool
	Lanes      []fullSuiteLaneReport
}

type fullSuiteChildRunner func(context.Context, fullSuiteChildRequest) fullSuiteChildResult

func currentFullSuiteInvocation() fullSuiteInvocation {
	visited := make(map[string]bool)
	flag.Visit(func(value *flag.Flag) {
		visited[value.Name] = true
	})
	return fullSuiteInvocation{
		Run:           testFlagString("test.run"),
		RunExplicit:   visited["test.run"],
		List:          testFlagString("test.list"),
		ListExplicit:  visited["test.list"],
		Bench:         testFlagString("test.bench"),
		BenchExplicit: visited["test.bench"],
		Fuzz:          testFlagString("test.fuzz"),
		FuzzExplicit:  visited["test.fuzz"],
		Skip:          testFlagString("test.skip"),
		SkipExplicit:  visited["test.skip"],
		Count:         testFlagInt("test.count", 1),
		Short:         testFlagBool("test.short"),
		FailFast:      testFlagBool("test.failfast"),
		CPU:           testFlagString("test.cpu"),
		Parallel:      testFlagString("test.parallel"),
		ParallelSet:   visited["test.parallel"],
		Shuffle:       testFlagString("test.shuffle"),
		Profiled:      fullSuiteProfileRequested(),
		ShardMarker:   os.Getenv(fullSuiteShardEnv),
	}
}

func testFlagString(name string) string {
	if value := flag.Lookup(name); value != nil {
		return value.Value.String()
	}
	return ""
}

func testFlagInt(name string, fallback int) int {
	value := flag.Lookup(name)
	if value == nil {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscan(value.Value.String(), &parsed); err != nil {
		return fallback
	}
	return parsed
}

func testFlagBool(name string) bool {
	return testFlagString(name) == "true"
}

func fullSuiteProfileRequested() bool {
	for _, name := range []string{
		"test.blockprofile",
		"test.cpuprofile",
		"test.memprofile",
		"test.mutexprofile",
		"test.trace",
		"test.gocoverdir",
	} {
		if strings.TrimSpace(testFlagString(name)) != "" {
			return true
		}
	}
	return false
}

func shouldRunFullSuiteController(invocation fullSuiteInvocation) bool {
	if invocation.Count != 1 || invocation.ShardMarker != "" {
		return false
	}
	if invocation.RunExplicit || invocation.ListExplicit || invocation.BenchExplicit || invocation.FuzzExplicit || invocation.SkipExplicit {
		return false
	}
	if invocation.Run != "" || invocation.List != "" || invocation.Bench != "" || invocation.Fuzz != "" || invocation.Skip != "" {
		return false
	}
	if invocation.Short || invocation.FailFast || invocation.Profiled || invocation.ParallelSet || strings.TrimSpace(invocation.CPU) != "" {
		return false
	}
	return invocation.Shuffle == "" || invocation.Shuffle == "off"
}

func planFullSuiteLanes(discovered []string, serialTests map[string]struct{}, heavyCosts, balanceCosts map[string]time.Duration, parallelLaneCount int) ([]fullSuiteLane, error) {
	if parallelLaneCount < 1 {
		return nil, fmt.Errorf("full-suite parallel lane count must be positive, got %d", parallelLaneCount)
	}
	seen := make(map[string]struct{}, len(discovered))
	ordered := append([]string(nil), discovered...)
	for _, testName := range ordered {
		if strings.TrimSpace(testName) == "" {
			return nil, errors.New("full-suite discovery returned an empty test name")
		}
		if _, exists := seen[testName]; exists {
			return nil, fmt.Errorf("full-suite discovery returned duplicate test %s", testName)
		}
		seen[testName] = struct{}{}
	}
	for testName := range serialTests {
		if _, exists := seen[testName]; !exists {
			return nil, fmt.Errorf("full-suite serial inventory names undiscovered test %s", testName)
		}
	}
	sort.Strings(ordered)

	serialLane := fullSuiteLane{Name: fullSuiteSerialLaneName, Serial: true}
	// Solo-measured costs classify heavy lanes; load-observed balancing
	// weights (falling back to solo, then a small default) pack every lane
	// so no straggler lane dominates the tail.
	weightFor := func(testName string) time.Duration {
		if weight := balanceCosts[testName]; weight > 0 {
			return weight
		}
		if weight := heavyCosts[testName]; weight > 0 {
			return weight
		}
		return 300 * time.Millisecond
	}
	heavyTests := make([]string, 0, len(ordered))
	parallelTests := make([]string, 0, len(ordered))
	for _, testName := range ordered {
		cost := heavyCosts[testName]
		if cost <= 0 {
			cost = time.Second
		}
		if _, serial := serialTests[testName]; serial {
			serialLane.Tests = append(serialLane.Tests, testName)
			serialLane.EstimatedCost += weightFor(testName)
			continue
		}
		if cost >= fullSuiteHeavyThreshold {
			heavyTests = append(heavyTests, testName)
			continue
		}
		parallelTests = append(parallelTests, testName)
	}

	byDescendingCost := func(tests []string) {
		sort.SliceStable(tests, func(i, j int) bool {
			leftCost := weightFor(tests[i])
			rightCost := weightFor(tests[j])
			if leftCost != rightCost {
				return leftCost > rightCost
			}
			return tests[i] < tests[j]
		})
	}
	byDescendingCost(heavyTests)
	byDescendingCost(parallelTests)

	// Heavy tests run serially inside their child (-test.parallel=1), so a
	// lane's estimated cost is its serial wall clock. Cap each lane by a
	// budget instead of a fixed lane count: the previous fixed split packed
	// 8-minute lanes that blew the 9-minute child timeout. First-fit over
	// descending costs; a single test above budget gets its own lane.
	var heavy []fullSuiteLane
	for _, testName := range heavyTests {
		cost := heavyCosts[testName]
		placed := false
		for index := range heavy {
			if heavy[index].EstimatedCost+cost <= fullSuiteHeavyLaneBudget {
				heavy[index].Tests = append(heavy[index].Tests, testName)
				heavy[index].EstimatedCost += cost
				placed = true
				break
			}
		}
		if !placed {
			heavy = append(heavy, fullSuiteLane{Name: fmt.Sprintf("heavy-io-%02d", len(heavy)+1), Heavy: true, Tests: []string{testName}, EstimatedCost: cost})
		}
	}

	if parallelLaneCount > len(parallelTests) && len(parallelTests) > 0 {
		parallelLaneCount = len(parallelTests)
	}
	parallel := make([]fullSuiteLane, parallelLaneCount)
	for index := range parallel {
		parallel[index].Name = fmt.Sprintf("parallel-%03d", index+1)
	}
	for _, testName := range parallelTests {
		lightest := 0
		for index := 1; index < len(parallel); index++ {
			if parallel[index].EstimatedCost < parallel[lightest].EstimatedCost {
				lightest = index
			}
		}
		cost := weightFor(testName)
		if cost <= 0 {
			cost = time.Second
		}
		parallel[lightest].Tests = append(parallel[lightest].Tests, testName)
		parallel[lightest].EstimatedCost += cost
	}

	lanes := make([]fullSuiteLane, 0, len(heavy)+len(parallel)+1)
	if len(serialLane.Tests) > 0 {
		lanes = append(lanes, serialLane)
	}
	for _, lane := range heavy {
		sort.Strings(lane.Tests)
		if len(lane.Tests) > 0 {
			lanes = append(lanes, lane)
		}
	}
	for _, lane := range parallel {
		sort.Strings(lane.Tests)
		if len(lane.Tests) > 0 {
			lanes = append(lanes, lane)
		}
	}
	if err := validateFullSuitePlan(discovered, lanes); err != nil {
		return nil, err
	}
	return lanes, nil
}

func validateFullSuitePlan(discovered []string, lanes []fullSuiteLane) error {
	discoveredCounts := make(map[string]int, len(discovered))
	for _, testName := range discovered {
		discoveredCounts[testName]++
		if discoveredCounts[testName] > 1 {
			return fmt.Errorf("full-suite discovery contains duplicate test %s", testName)
		}
	}
	plannedCounts := make(map[string]int, len(discovered))
	for _, lane := range lanes {
		if strings.TrimSpace(lane.Name) == "" {
			return errors.New("full-suite plan contains an unnamed lane")
		}
		for _, testName := range lane.Tests {
			plannedCounts[testName]++
			if plannedCounts[testName] > 1 {
				return fmt.Errorf("full-suite plan contains duplicate test %s", testName)
			}
			if discoveredCounts[testName] == 0 {
				return fmt.Errorf("full-suite plan contains undiscovered test %s", testName)
			}
		}
	}
	var missing []string
	for testName := range discoveredCounts {
		if plannedCounts[testName] == 0 {
			missing = append(missing, testName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("full-suite plan is missing tests: %s", strings.Join(missing, ", "))
	}
	return nil
}

func runFullSuiteController() int {
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "full-suite controller: resolve current test binary: %v\n", err)
		return 1
	}
	overall, command, child := resolveFullSuiteCeilings()
	fullSuiteCommandTimeout = command
	fullSuiteChildTimeout = child
	ctx, cancel := context.WithTimeout(context.Background(), overall)
	defer cancel()
	discovered, err := discoverFullSuiteTests(ctx, executable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "full-suite controller: %v\n", err)
		return 1
	}
	serialInventory := fullSuiteSerialInventory()
	lanes, err := planFullSuiteLanes(discovered, fullSuiteSerialTests(serialInventory), fullSuiteMeasuredCosts(), fullSuiteBalancingCosts(), fullSuiteLogicalShards)
	if err != nil {
		fmt.Fprintf(os.Stderr, "full-suite controller: %v\n", err)
		return 1
	}
	report, runErr := runFullSuiteLanes(ctx, executable, lanes, fullSuiteWorkers, runFullSuiteChildProcess)
	writeFullSuiteReport(os.Stdout, report)
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "full-suite controller failed: %v\n", runErr)
		return 1
	}
	return 0
}

// resolveFullSuiteCeilings derives the controller's overall, per-lane, and
// child ceilings from the -test.timeout the outer go tool passed to this
// binary. The go tool SIGQUITs the whole process at that deadline, so the
// controller reserves a grace margin and stops itself first with complete
// accounting. A run given a generous explicit timeout (the receipt gates)
// therefore executes the whole corpus; a bare command on go's 10m default
// stops orderly at ~9m15s. Fallbacks apply only when the flag is unreadable.
func resolveFullSuiteCeilings() (overall, command, child time.Duration) {
	overall = fullSuiteOverallTimeoutFallback
	command = fullSuiteCommandTimeoutFallback
	child = fullSuiteChildTimeoutFallback
	raw := strings.TrimSpace(testFlagString("test.timeout"))
	if raw == "" {
		return overall, command, child
	}
	deadline, err := time.ParseDuration(raw)
	if err != nil || deadline <= 0 {
		return overall, command, child
	}
	// Reserve a grace margin so the controller returns before the go tool
	// kills the process; scale the margin with the deadline but keep it
	// bounded. Very small deadlines fall back rather than starve.
	grace := 45 * time.Second
	if deadline > 20*time.Minute {
		grace = 90 * time.Second
	}
	if deadline <= grace+time.Minute {
		return overall, command, child
	}
	overall = deadline - grace
	// A lane must be able to finish inside the overall ceiling; the slowest
	// known single test approaches 9 minutes under load, so give lanes the
	// whole overall window minus a small settle margin, capped so one stuck
	// lane cannot consume the entire budget past reporting.
	command = overall - 15*time.Second
	child = command - 15*time.Second
	return overall, command, child
}

func discoverFullSuiteTests(ctx context.Context, executable string) ([]string, error) {
	output, err := withIsolatedProcessTestHub(ctx, func(hubDir string) ([]byte, error) {
		command := exec.CommandContext(ctx, executable,
			"-test.list=^(Test|Example)",
			"-test.count=1",
			"-test.timeout=30s",
		)
		command.Env = fullSuiteChildEnvironment(os.Environ(), "discovery", hubDir)
		command.WaitDelay = isolatedProcessWaitDelay
		return command.CombinedOutput()
	})
	if err != nil {
		return nil, fmt.Errorf("discover top-level tests with current binary: %w\n%s", err, output)
	}
	validName := regexp.MustCompile(`^(Test|Example)[A-Za-z0-9_]*$`)
	var discovered []string
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if validName.MatchString(name) {
			discovered = append(discovered, name)
		}
	}
	if len(discovered) == 0 {
		return nil, fmt.Errorf("current test binary reported no top-level tests:\n%s", output)
	}
	return discovered, nil
}

func fullSuiteChildRequestForLane(executable string, lane fullSuiteLane) fullSuiteChildRequest {
	quoted := make([]string, 0, len(lane.Tests))
	for _, testName := range lane.Tests {
		quoted = append(quoted, regexp.QuoteMeta(testName))
	}
	runSelector := "^(" + strings.Join(quoted, "|") + ")$"
	args := []string{
		"-test.run=" + runSelector,
		"-test.count=1",
		"-test.timeout=" + fullSuiteChildTimeout.String(),
		"-test.v=true",
	}
	childParallel := fullSuiteChildParallel
	if lane.Heavy {
		childParallel = 1
	}
	args = append(args, fmt.Sprintf("-test.parallel=%d", childParallel))
	if testFlagBool("test.fullpath") {
		args = append(args, "-test.fullpath=true")
	}
	return fullSuiteChildRequest{
		Executable: executable,
		Lane:       lane,
		Args:       args,
		Env:        fullSuiteChildEnvironment(os.Environ(), lane.Name, ""),
	}
}

func fullSuiteChildEnvironment(parent []string, laneName, hubDir string) []string {
	environment := make([]string, 0, len(parent)+2)
	for _, entry := range parent {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case fullSuiteShardEnv, isolatedProcessHubEnv, "AETHER_ROOT", "COLONY_DATA_DIR", "GOMAXPROCS":
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment, fullSuiteShardEnv+"="+fmt.Sprintf("%d:%s", os.Getpid(), laneName))
	environment = append(environment, fmt.Sprintf("GOMAXPROCS=%d", fullSuiteChildProcs))
	if hubDir != "" {
		environment = append(environment, isolatedProcessHubEnv+"="+hubDir)
	}
	return environment
}

func runFullSuiteChildProcess(ctx context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
	started := time.Now()
	childCtx, cancel := context.WithTimeout(ctx, fullSuiteCommandTimeout)
	defer cancel()
	output, err := withIsolatedProcessTestHub(childCtx, func(hubDir string) ([]byte, error) {
		command := exec.CommandContext(childCtx, request.Executable, request.Args...)
		command.Env = fullSuiteChildEnvironment(request.Env, request.Lane.Name, hubDir)
		command.WaitDelay = isolatedProcessWaitDelay
		return command.CombinedOutput()
	})
	if childCtx.Err() != nil {
		err = errors.Join(err, fmt.Errorf("lane %s exceeded %s: %w", request.Lane.Name, fullSuiteCommandTimeout, childCtx.Err()))
	}
	return fullSuiteChildResult{
		Output:   string(output),
		Executed: fullSuiteExecutedTopLevels(output),
		Duration: time.Since(started),
		Err:      err,
	}
}

// fullSuiteSerialInventory is deliberately small and names each test instead
// of classifying broad prefixes. Every entry has concrete evidence that it can
// observe or register a resource belonging to the source checkout.
func fullSuiteSerialInventory() map[string]string {
	return map[string]string{
		"TestColonyStateWriteAllowlistOnlyShrinks":             "fixed checked-in allowlist has an explicit regeneration path",
		"TestCurrentVocabulary199":                             "live tracked checkout inventory is read through git ls-files",
		"TestColonyPrimeMdDeletionProducesByteIdenticalOutput": "byte-identical golden output is perturbed by CPU contention under parallel load",
		"TestEveryLifecycleCommandEndsWithNextAction":          "renders every command; timing-sensitive under parallel load",
		"TestGoldenBuildVisualOutput":                          "byte-identical golden visual output is perturbed by CPU contention under parallel load",
		"TestHeartbeatScanDetectsStale":                        "fixed staleness clock thresholds are timing-sensitive under parallel load",
		"TestNewSubcommandFlags":                               "asserts exact command registry; ordering-sensitive under parallel load",
		"TestOracleCompatibilityWritesHeartbeatWhileRunning":   "measures a live background heartbeat interval; timing-sensitive under parallel load",
		"TestBuildDispatchStartsHeartbeatMonitor":              "measures a live background heartbeat monitor; timing-sensitive under parallel load",
		"TestHeartbeatMonitorStopsOnCancel":                    "measures a live background heartbeat monitor lifecycle; timing-sensitive under parallel load",
		"TestGoldenContinueVisualOutput":                       "byte-identical golden visual output is perturbed by CPU contention under parallel load",
		"TestGoldenPlanVisualOutput":                           "byte-identical golden visual output is perturbed by CPU contention under parallel load",
		"TestNextActionNeverHardcoded":                         "fixed checked-in allowlist has an explicit regeneration path",
		"TestOrphanAllowlistOnlyShrinks":                       "fixed checked-in allowlist has an explicit regeneration path",
		"TestLoadNextActionInputDoesNotMutate":                 "asserts no-mutation on shared next-action inputs; ordering-sensitive under parallel load",
		"TestPackedNPMReleaseCandidateContract":                "real npm installs, a staged release server, and the shared npm cache are load-sensitive",
		"TestPhase199GateReceipt":                              "live repository receipt validates git identity and protected fingerprints",
		"TestWorktreeAllocateAgentPhase":                       "source checkout worktree registration guards a legacy allocation path",
		"TestWorktreeAllocateAuditLog":                         "source checkout worktree registration guards a legacy allocation path",
		"TestWorktreeAllocateHumanBranch":                      "source checkout worktree registration guards a legacy allocation path",
		"TestWorktreeAllocateMergedBranchAllowed":              "source checkout worktree registration guards a legacy allocation path",
	}
}

func fullSuiteSerialTests(inventory map[string]string) map[string]struct{} {
	tests := make(map[string]struct{}, len(inventory))
	for testName := range inventory {
		tests[testName] = struct{}{}
	}
	return tests
}

// fullSuiteMeasuredCosts records the slowest top-level tests observed by the
// first uncached Plan 200-55 diagnostic. Values are coarse weights rather than
// deadlines: the greedy planner uses them only to keep expensive tests apart.
func fullSuiteMeasuredCosts() map[string]time.Duration {
	return map[string]time.Duration{
		"TestClassicContractPhase200CausalExecution":                                 273 * time.Second,
		"TestPlanningGapEdgeAccounting200":                                           260 * time.Second,
		"TestPlanningNumericBoundaries200":                                           144 * time.Second,
		"TestPlanningRealRepo200":                                                    106 * time.Second,
		"TestPlanningRouteStageRejectsStaleBindingsAndForbiddenAuthority":            92 * time.Second,
		"TestPlanningExpiryPresentation200":                                          46 * time.Second,
		"TestBothBuildLanesEmitTheHeadsUp":                                           52 * time.Second,
		"TestNonInteractiveRunsStillPrintTheHeadsUp":                                 52 * time.Second,
		"TestBothBuildLanesAgreeOnBoundaryQuestionSignal":                            50 * time.Second,
		"TestBuildStartTransaction200FaultsRollbackAndNeverDispatch":                 47 * time.Second,
		"TestBuildStartTransaction200ReloadsCanonicalAcceptedAuthority":              35 * time.Second,
		"TestBuildBufferedOutputBreaksJSONUnderVisualEnv":                            24 * time.Second,
		"TestCodexVisualParity":                                                      21 * time.Second,
		"TestBuildStartConcurrentProcesses200":                                       19 * time.Second,
		"TestBuildAttemptExternalUnboundStartReplayIsByteStable":                     19 * time.Second,
		"TestStatusSurfacesFailedAttemptAfterLifecycleRollback":                      19 * time.Second,
		"TestGroupedJobPartialRetryIsIdempotent":                                     18 * time.Second,
		"TestForceRedispatchMarksActiveAttemptInterrupted":                           18 * time.Second,
		"TestResumeDashboardDoesNotRedispatchLiveBuildProcess":                       18 * time.Second,
		"TestBuildAttemptPersistsTransitionsAndTerminalEvidence":                     18 * time.Second,
		"TestBuildAttemptFixtureUsesCanonicalTransaction":                            18 * time.Second,
		"TestNoLaneRendersTwoCostLines":                                              18 * time.Second,
		"TestCanonicalBuildStartFixtureLiveProcess200":                               18 * time.Second,
		"TestPartialRetryAttemptDoesNotBecomeTheLatestAttempt":                       18 * time.Second,
		"TestBuildRepairsCompletedPriorPhaseTasksFromTrustedManifest":                17 * time.Second,
		"TestBuildAttemptExternalAttemptEnumerationExcludesStartReceipts200":         17 * time.Second,
		"TestBuildAttemptChildLinksParentWithoutMutation":                            17 * time.Second,
		"TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromSealedAttempt":       17 * time.Second,
		"TestFailedCheckSendsExactlyOneBuilderFixAttempt":                            17 * time.Second,
		"TestSecondFailureBlocksAndNamesTheCommand":                                  16 * time.Second,
		"TestPartialRetryCommandIsAcceptedOnThePlanOnlyPath":                         16 * time.Second,
		"TestRuntimeCheckpointGenerationAcceptsJournalBoundDirectFinalProjection":    16 * time.Second,
		"TestFixAttemptIsCountedSeparately":                                          16 * time.Second,
		"TestContinueFinalizeRecordsExternalReviewAndAdvances":                       15 * time.Second,
		"TestFixAttemptNeverOverwritesTheFirstResult":                                15 * time.Second,
		"TestRunCompatibilityPassesWorkerTimeoutToBuildAndContinue":                  14 * time.Second,
		"TestBuildSupportsTaskScopedRedispatch":                                      14 * time.Second,
		"TestDispatchEntryCarriesBriefPath":                                          14 * time.Second,
		"TestRunCompatibilityExecutesSinglePhase":                                    14 * time.Second,
		"TestContinueEndToEndAfterAbandonedRecovery":                                 14 * time.Second,
		"TestPendingDecisionStillRendersFullCheckinCard":                             14 * time.Second,
		"TestContinueFinalizeWritesWorkerOutcomeReports":                             14 * time.Second,
		"TestNoSecondAutomaticFixAttempt":                                            14 * time.Second,
		"TestBuildPlanOnlyHeavyReviewAllowsPolicyMeasurerAndChaos":                   13 * time.Second,
		"TestBuildPlanOnlyCLIForwardsVerificationDepth":                              13 * time.Second,
		"TestBuildVisualOutputShowsSpawnPlan":                                        13 * time.Second,
		"TestOneWorkerWithForcedReviewerWaiverStillPauses":                           13 * time.Second,
		"TestPlanEmitsLifecycleCeremonyEvents":                                       13 * time.Second,
		"TestCanonicalBuildStartFixtureAuthority200":                                 13 * time.Second,
		"TestBuildJobProposalRoundTrip":                                              12 * time.Second,
		"TestEnsureUniqueBuildDispatchNamesSuffixesCollisionFromDifferentPhase":      12 * time.Second,
		"TestGoldenStateMutations":                                                   12 * time.Second,
		"TestPartialRetryCommandNeverRedispatchesCreditedWork":                       11 * time.Second,
		"TestPlanUsesSurveyAndRecordsPlanningDispatches":                             11 * time.Second,
		"TestBuildFinalizeRecordsExternalTaskResultsForContinue":                     11 * time.Second,
		"TestGoldenContinueVisualOutput":                                             11 * time.Second,
		"TestBuildStartLegacyBinding200":                                             11 * time.Second,
		"TestBuildCLIForwardsVerificationDepth":                                      11 * time.Second,
		"TestGoldenBuildVisualOutput":                                                10 * time.Second,
		"TestInstallUsesEmbeddedAssetsWithoutPackageDir":                             10 * time.Second,
		"TestMigratedWorkLoopClosingsSayItOnce":                                      10 * time.Second,
		"TestAutopilotPolicyPlanAuthorityRefusalIsZeroEffect":                        10 * time.Second,
		"TestBuildStartTransaction200TargetMatrix":                                   10 * time.Second,
		"TestCodexPlanFinalizeRouteExposesNextScoutBoundary":                         9 * time.Second,
		"TestBuildWritesDispatchArtifactsAndUpdatesState":                            9 * time.Second,
		"TestMaintenanceArchive199RepairRollback":                                    9 * time.Second,
		"TestLifecycleTransactionFaultMatrix":                                        8 * time.Second,
		"TestPlanFinalizePendingIterationDoesNotWriteFinalPlanAndDrivesNextManifest": 8 * time.Second,
		"TestSpawnLogFailsClosedWhenWaiverWindowCannotPersist":                       8 * time.Second,
		"TestMaintenanceArchive199ForcedMarker":                                      8 * time.Second,
	}
}

func fullSuiteExecutedTopLevels(output []byte) []string {
	const prefix = "=== RUN   "
	var names []string
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		if name != "" && !strings.Contains(name, "/") {
			names = append(names, name)
		}
	}
	return names
}

func runFullSuiteLanes(ctx context.Context, executable string, lanes []fullSuiteLane, parallelism int, runner fullSuiteChildRunner) (fullSuiteRunReport, error) {
	report := fullSuiteRunReport{Lanes: make([]fullSuiteLaneReport, len(lanes))}
	if strings.TrimSpace(executable) == "" {
		return report, errors.New("full-suite current executable is empty")
	}
	if parallelism < 1 {
		return report, fmt.Errorf("full-suite parallelism must be positive, got %d", parallelism)
	}
	if runner == nil {
		return report, errors.New("full-suite child runner is nil")
	}
	var discovered []string
	for _, lane := range lanes {
		discovered = append(discovered, lane.Tests...)
	}
	if err := validateFullSuitePlan(discovered, lanes); err != nil {
		return report, err
	}
	report.Discovered = len(discovered)

	results := make([]fullSuiteChildResult, len(lanes))
	runLane := func(index int) {
		request := fullSuiteChildRequestForLane(executable, lanes[index])
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					results[index] = fullSuiteChildResult{Err: fmt.Errorf("child runner panic: %v", recovered)}
				}
			}()
			results[index] = runner(ctx, request)
		}()
	}
	var heavyIndexes []int
	var lightIndexes []int
	for index, lane := range lanes {
		if lane.Serial {
			runLane(index)
			continue
		}
		if lane.Heavy {
			heavyIndexes = append(heavyIndexes, index)
		} else {
			lightIndexes = append(lightIndexes, index)
		}
	}
	heavyJobs := make(chan int, len(heavyIndexes))
	lightJobs := make(chan int, len(lightIndexes))
	for _, index := range heavyIndexes {
		heavyJobs <- index
	}
	close(heavyJobs)
	for _, index := range lightIndexes {
		lightJobs <- index
	}
	close(lightJobs)

	var workers sync.WaitGroup
	workerCount := parallelism
	if total := len(heavyIndexes) + len(lightIndexes); workerCount > total {
		workerCount = total
	}
	heavyWorkerCount := fullSuiteHeavyWorkers
	if heavyWorkerCount > len(heavyIndexes) {
		heavyWorkerCount = len(heavyIndexes)
	}
	if heavyWorkerCount > workerCount {
		heavyWorkerCount = workerCount
	}
	for worker := 0; worker < heavyWorkerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range heavyJobs {
				runLane(index)
			}
			for index := range lightJobs {
				runLane(index)
			}
		}()
	}
	for worker := heavyWorkerCount; worker < workerCount; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range lightJobs {
				runLane(index)
			}
		}()
	}
	workers.Wait()

	var failures []string
	for index, lane := range lanes {
		result := results[index]
		laneReport := fullSuiteLaneReport{
			Name:       lane.Name,
			Planned:    len(lane.Tests),
			Executed:   len(result.Executed),
			Duration:   result.Duration,
			Successful: result.Err == nil,
			Output:     result.Output,
		}
		report.Lanes[index] = laneReport
		report.Executed += len(result.Executed)
		if accountingErr := validateFullSuiteExecution(lane.Tests, result.Executed); accountingErr != nil {
			laneReport.Successful = false
			report.Lanes[index] = laneReport
			failures = append(failures, fmt.Sprintf("lane %s accounting: %v", lane.Name, accountingErr))
		}
		if result.Err != nil {
			failures = append(failures, fmt.Sprintf("lane %s: %v\n%s", lane.Name, result.Err, strings.TrimSpace(result.Output)))
		}
	}
	report.Passed = len(failures) == 0 && report.Executed == report.Discovered
	if !report.Passed {
		if len(failures) == 0 {
			failures = append(failures, fmt.Sprintf("full-suite accounting discovered=%d executed=%d", report.Discovered, report.Executed))
		}
		return report, errors.New(strings.Join(failures, "\n"))
	}
	return report, nil
}

func validateFullSuiteExecution(planned, executed []string) error {
	plannedCounts := make(map[string]int, len(planned))
	for _, testName := range planned {
		plannedCounts[testName]++
	}
	executedCounts := make(map[string]int, len(executed))
	for _, testName := range executed {
		executedCounts[testName]++
		if executedCounts[testName] > 1 {
			return fmt.Errorf("duplicate executed test %s", testName)
		}
		if plannedCounts[testName] == 0 {
			return fmt.Errorf("unexpected executed test %s", testName)
		}
	}
	var missing []string
	for testName := range plannedCounts {
		if executedCounts[testName] == 0 {
			missing = append(missing, testName)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("missing executed tests: %s", strings.Join(missing, ", "))
	}
	return nil
}

func writeFullSuiteReport(output io.Writer, report fullSuiteRunReport) {
	status := "PASS"
	if !report.Passed {
		status = "FAIL"
	}
	fmt.Fprintf(output, "FULL-SUITE %s discovered=%d executed=%d lanes=%d\n", status, report.Discovered, report.Executed, len(report.Lanes))
	for _, lane := range report.Lanes {
		laneStatus := "PASS"
		if !lane.Successful {
			laneStatus = "FAIL"
		}
		fmt.Fprintf(output, "FULL-SUITE lane=%s status=%s planned=%d executed=%d duration=%s\n", lane.Name, laneStatus, lane.Planned, lane.Executed, lane.Duration.Round(time.Millisecond))
	}
	ranked := append([]fullSuiteLaneReport(nil), report.Lanes...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Duration != ranked[j].Duration {
			return ranked[i].Duration > ranked[j].Duration
		}
		return ranked[i].Name < ranked[j].Name
	})
	for index, lane := range ranked {
		fmt.Fprintf(output, "FULL-SUITE slowest rank=%d lane=%s duration=%s\n", index+1, lane.Name, lane.Duration.Round(time.Millisecond))
	}
	for _, lane := range report.Lanes {
		if lane.Output != "" {
			fmt.Fprintf(output, "FULL-SUITE output-begin lane=%s\n", lane.Name)
			_, _ = io.WriteString(output, lane.Output)
			if !strings.HasSuffix(lane.Output, "\n") {
				_, _ = io.WriteString(output, "\n")
			}
			fmt.Fprintf(output, "FULL-SUITE output-end lane=%s\n", lane.Name)
		}
	}
}

// saveGlobals captures the current values of all mutable package-level globals
// and restores them when the test completes. Every test that assigns to store,
// stdout, stderr, flagTypeFilter, or flagStatusFilter must call this as its
// first action.
func saveGlobals(t *testing.T) {
	t.Helper()
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath
	origResumeNoHandoff := resumeNoHandoff
	origColonyPrimeTemplatesPathOverride := colonyPrimeTemplatesPathOverride
	// root.go sets this on every command execution and nothing resets it, so a
	// test that ran a quiet command (anything *-finalize, spawn-log, version...)
	// left every later test's emitVisualLine silently emitting nothing. That is
	// how TestOracleStatusFollowStreamsExistingRoundsAndExitsOnRunEnd could pass
	// alone and fail in the full suite depending on lane order.
	origCurrentStreamingCommand := currentStreamingCommand
	t.Cleanup(func() {
		// A Store and its tracer are repository authorities, not ordinary test
		// values. Never resurrect one after its temporary repository may have
		// been deleted by another cleanup.
		store = nil
		stdout = origStdout
		stderr = origStderr
		flagTypeFilter = origFlagType
		flagStatusFilter = origFlagStatus
		flagListJSON = origFlagListJSON
		historyJSON = origHistoryJSON
		phaseJSON = origPhaseJSON
		historyLimit = origHistoryLimit
		historyFilter = origHistoryFilter
		phaseNumber = origPhaseNumber
		tracer = nil
		continueContextUpdater = origContinueContextUpdater
		continueSignalHousekeeper = origContinueSignalHousekeeper
		newCodexWorkerInvoker = origNewCodexWorkerInvoker
		activeBuildCeremony = origActiveBuildCeremony
		currentStreamingCommand = origCurrentStreamingCommand
		narratorLookPath = origNarratorLookPath
		narratorCommandContext = origNarratorCommandContext
		narratorRuntimePath = origNarratorRuntimePath
		resumeNoHandoff = origResumeNoHandoff
		colonyPrimeTemplatesPathOverride = origColonyPrimeTemplatesPathOverride
		resetColonyPrimeTemplatesCache()
	})
}

func forceJSONOutputModeForTest(t *testing.T) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
}

type commandTestRepository struct {
	Root    string
	DataDir string
	Store   *storage.Store
}

// bindCommandTestRepository gives command tests the same single physical
// authority production requires: AETHER_ROOT, COLONY_DATA_DIR, store, and
// tracer all point at one repository-local .aether/data tree. Cleanup is
// registered after TempDir so process globals are cleared before the directory
// is removed; testing.T.Setenv preserves absent-versus-present environment
// state exactly.
func bindCommandTestRepository(t *testing.T) commandTestRepository {
	t.Helper()
	return bindCommandTestRepositoryAt(t, t.TempDir())
}

// bindCommandTestRepositoryAt applies the command-test authority contract to
// an existing temporary repository. Worktree fixtures need this form because
// they create real Git state beneath the same root that owns .aether/data.
func bindCommandTestRepositoryAt(t *testing.T, repositoryRoot string) commandTestRepository {
	t.Helper()

	dataDir := filepath.Join(repositoryRoot, ".aether", "data")
	authority, err := storage.OpenRepositoryRoot(repositoryRoot, dataDir)
	if err != nil {
		t.Fatalf("open command-test repository authority: %v", err)
	}
	boundStore, err := storage.NewRepositoryStore(authority)
	if err != nil {
		_ = authority.Close()
		t.Fatalf("create command-test repository store: %v", err)
	}

	t.Setenv("AETHER_ROOT", repositoryRoot)
	t.Setenv("COLONY_DATA_DIR", dataDir)

	store = boundStore
	tracer = trace.NewTracer(boundStore)
	rootCmd.SetArgs([]string{})
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	resetFlags(rootCmd)

	t.Cleanup(func() {
		// This cleanup runs before Setenv restoration and TempDir deletion.
		// Do not restore a possibly stale repository authority here.
		store = nil
		tracer = nil
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		rootCmd.SetErr(os.Stderr)
		resetFlags(rootCmd)
		_ = authority.Close()
	})

	return commandTestRepository{
		Root:    repositoryRoot,
		DataDir: dataDir,
		Store:   boundStore,
	}
}

func selfRestoringAetherRootCleanupSites200(fileSet *token.FileSet, file *ast.File) []string {
	var sites []string
	ast.Inspect(file, func(node ast.Node) bool {
		deferred, ok := node.(*ast.DeferStmt)
		if !ok || deferred.Call == nil || len(deferred.Call.Args) != 2 {
			return true
		}
		setenv, ok := deferred.Call.Fun.(*ast.SelectorExpr)
		if !ok || setenv.Sel.Name != "Setenv" {
			return true
		}
		setenvPackage, ok := setenv.X.(*ast.Ident)
		if !ok || setenvPackage.Name != "os" {
			return true
		}
		name, ok := deferred.Call.Args[0].(*ast.BasicLit)
		if !ok || name.Kind != token.STRING || name.Value != `"AETHER_ROOT"` {
			return true
		}
		getenvCall, ok := deferred.Call.Args[1].(*ast.CallExpr)
		if !ok || len(getenvCall.Args) != 1 {
			return true
		}
		getenv, ok := getenvCall.Fun.(*ast.SelectorExpr)
		if !ok || getenv.Sel.Name != "Getenv" {
			return true
		}
		getenvPackage, ok := getenv.X.(*ast.Ident)
		if !ok || getenvPackage.Name != "os" {
			return true
		}
		getenvName, ok := getenvCall.Args[0].(*ast.BasicLit)
		if !ok || getenvName.Kind != token.STRING || getenvName.Value != `"AETHER_ROOT"` {
			return true
		}
		sites = append(sites, fileSet.Position(deferred.Pos()).String())
		return true
	})
	return sites
}

func TestNoSelfRestoringAetherRootCleanup200(t *testing.T) {
	const sample = `package cmd

import "os"

// defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT")) is explanatory prose.
const explanation = ` + "`" + `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))` + "`" + `

func brokenCleanup() {
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
}
`

	sampleSet := token.NewFileSet()
	sampleFile, err := parser.ParseFile(sampleSet, "sample_test.go", sample, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse in-memory cleanup sample: %v", err)
	}
	if sites := selfRestoringAetherRootCleanupSites200(sampleSet, sampleFile); len(sites) != 1 {
		t.Fatalf("active cleanup sample sites = %v, want exactly one executable defer", sites)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd test sources: %v", err)
	}
	var activeSites []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fileSet, entry.Name(), nil, parser.ParseComments)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", entry.Name(), parseErr)
		}
		activeSites = append(activeSites, selfRestoringAetherRootCleanupSites200(fileSet, parsed)...)
	}
	sort.Strings(activeSites)
	if len(activeSites) != 0 {
		t.Fatalf("self-restoring AETHER_ROOT cleanup returned after the 120-site migration:\n%s", strings.Join(activeSites, "\n"))
	}
}

func TestRepositoryTestBinding200(t *testing.T) {
	originalRoot, hadRoot := os.LookupEnv("AETHER_ROOT")
	originalDataDir, hadDataDir := os.LookupEnv("COLONY_DATA_DIR")
	originalStore := store
	originalTracer := tracer
	defer func() {
		if hadRoot {
			_ = os.Setenv("AETHER_ROOT", originalRoot)
		} else {
			_ = os.Unsetenv("AETHER_ROOT")
		}
		if hadDataDir {
			_ = os.Setenv("COLONY_DATA_DIR", originalDataDir)
		} else {
			_ = os.Unsetenv("COLONY_DATA_DIR")
		}
		store = originalStore
		tracer = originalTracer
		resetFlags(rootCmd)
	}()

	_ = os.Unsetenv("AETHER_ROOT")
	_ = os.Unsetenv("COLONY_DATA_DIR")
	store = nil
	tracer = nil
	resetFlags(rootCmd)

	var repositoryRoot string
	t.Run("binds one contained authority", func(t *testing.T) {
		binding := bindCommandTestRepository(t)
		repositoryRoot = binding.Root

		if got := os.Getenv("AETHER_ROOT"); got != binding.Root {
			t.Fatalf("AETHER_ROOT = %q, want %q", got, binding.Root)
		}
		if got := os.Getenv("COLONY_DATA_DIR"); got != binding.DataDir {
			t.Fatalf("COLONY_DATA_DIR = %q, want %q", got, binding.DataDir)
		}
		if store != binding.Store {
			t.Fatalf("package store = %p, want bound store %p", store, binding.Store)
		}
		if tracer == nil {
			t.Fatal("package tracer was not bound with the repository store")
		}
		if got := filepath.Clean(binding.Store.BasePath()); got != filepath.Clean(binding.DataDir) {
			t.Fatalf("store base path = %q, want %q", got, binding.DataDir)
		}
		rel, err := filepath.Rel(binding.Root, binding.DataDir)
		if err != nil || rel != filepath.Join(".aether", "data") {
			t.Fatalf("data root is not the repository-local .aether/data path: rel=%q err=%v", rel, err)
		}

		rootCmd.SetArgs([]string{"stale-command"})
		rootCmd.SetOut(&strings.Builder{})
		rootCmd.SetErr(&strings.Builder{})
		if err := historyCmd.Flags().Set("limit", "999"); err != nil {
			t.Fatalf("mutate Cobra flag: %v", err)
		}
	})

	if _, configured := os.LookupEnv("AETHER_ROOT"); configured {
		t.Fatal("AETHER_ROOT was initially absent but cleanup left it present")
	}
	if _, configured := os.LookupEnv("COLONY_DATA_DIR"); configured {
		t.Fatal("COLONY_DATA_DIR was initially absent but cleanup left it present")
	}
	if store != nil || tracer != nil {
		t.Fatalf("bootstrap globals survived cleanup: store=%p tracer=%p", store, tracer)
	}
	if got := rootCmd.OutOrStdout(); got != os.Stdout {
		t.Fatalf("Cobra stdout was not reset: got %T", got)
	}
	if got := rootCmd.ErrOrStderr(); got != os.Stderr {
		t.Fatalf("Cobra stderr was not reset: got %T", got)
	}
	if historyLimit != 20 {
		t.Fatalf("Cobra flag state survived cleanup: history limit = %d, want 20", historyLimit)
	}
	if _, err := os.Stat(repositoryRoot); !os.IsNotExist(err) {
		t.Fatalf("temporary repository still exists after cleanup: %v", err)
	}
}

func isPermissionDeniedForTest(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "operation not permitted") || strings.Contains(text, "permission denied")
}

// resetRootCmd restores rootCmd state (SetArgs, SetOut) and resets all
// subcommand local flags to their defaults. Cobra local flags persist across
// Execute() calls, so without this, --type blocker on flag-add would leak
// into the next test that runs flag-add without --type.
// Every test that calls rootCmd.Execute must call this early in the function.
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		// Reset local flags on all subcommands to their defaults.
		// This prevents flag value leakage between tests.
		resetFlags(rootCmd)
	})
}

// resetFlags recursively resets all local flags on cmd and its subcommands
// to their default values. This prevents Cobra flag leakage between tests.
func resetFlags(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if sliceValue, ok := f.Value.(pflag.SliceValue); ok {
			if f.DefValue == "" || f.DefValue == "[]" {
				_ = sliceValue.Replace(nil)
			}
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

type testOwnedWorktree struct {
	repoRoot string
	path     string
	branch   string
}

var testOwnedWorktrees struct {
	sync.Mutex
	entries []testOwnedWorktree
}

// registerTestOwnedWorktree records positive ownership before a test creates a
// real worktree. The registry is process-local, so only this test run can grant
// cleanup permission for an exact repository, path, and branch.
func registerTestOwnedWorktree(t *testing.T, repoRoot, path, branch string) {
	t.Helper()
	entry, err := validateTestOwnedWorktree(repoRoot, path, branch)
	if err != nil {
		t.Fatalf("register test-owned worktree: %v", err)
	}
	testOwnedWorktrees.Lock()
	testOwnedWorktrees.entries = append(testOwnedWorktrees.entries, entry)
	testOwnedWorktrees.Unlock()
}

// cleanupTestWorktrees removes only worktrees and branches that this test
// process registered explicitly. Missing artifacts are harmless: cleanup is
// deliberately safe to replay.
func cleanupTestWorktrees() {
	testOwnedWorktrees.Lock()
	entries := append([]testOwnedWorktree(nil), testOwnedWorktrees.entries...)
	testOwnedWorktrees.entries = nil
	testOwnedWorktrees.Unlock()

	for _, registered := range entries {
		entry, err := validateTestOwnedWorktree(registered.repoRoot, registered.path, registered.branch)
		if err != nil {
			continue
		}
		_ = exec.Command("git", "-C", entry.repoRoot, "worktree", "remove", "--force", entry.path).Run()
		_ = exec.Command("git", "-C", entry.repoRoot, "branch", "-D", "--", entry.branch).Run()
	}
}

func validateTestOwnedWorktree(repoRoot, path, branch string) (testOwnedWorktree, error) {
	root, err := canonicalTestRepositoryRoot(repoRoot)
	if err != nil {
		return testOwnedWorktree{}, err
	}
	if strings.TrimSpace(path) == "" {
		return testOwnedWorktree{}, fmt.Errorf("worktree path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	resolvedPath, err := resolveTestOwnedPath(path)
	if err != nil {
		return testOwnedWorktree{}, fmt.Errorf("resolve worktree path: %w", err)
	}
	relativePath, err := filepath.Rel(root, resolvedPath)
	if err != nil || relativePath == "." || relativePath == "" || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return testOwnedWorktree{}, fmt.Errorf("worktree path %q is not beneath repository %q", resolvedPath, root)
	}
	if branch == "" || branch != strings.TrimSpace(branch) {
		return testOwnedWorktree{}, fmt.Errorf("branch name is required")
	}
	if output, err := exec.Command("git", "check-ref-format", "--branch", branch).CombinedOutput(); err != nil {
		return testOwnedWorktree{}, fmt.Errorf("invalid branch %q: %s", branch, strings.TrimSpace(string(output)))
	}
	return testOwnedWorktree{repoRoot: root, path: resolvedPath, branch: branch}, nil
}

func canonicalTestRepositoryRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("repository root is required")
	}
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("make repository root absolute: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("stat repository root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root %q is not a directory", resolvedRoot)
	}
	topLevelOutput, err := exec.Command("git", "-C", resolvedRoot, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("validate repository root: %w", err)
	}
	topLevel, err := filepath.EvalSymlinks(strings.TrimSpace(string(topLevelOutput)))
	if err != nil {
		return "", fmt.Errorf("resolve git top-level: %w", err)
	}
	if filepath.Clean(topLevel) != filepath.Clean(resolvedRoot) {
		return "", fmt.Errorf("repository root %q is not git top-level %q", resolvedRoot, topLevel)
	}
	return filepath.Clean(resolvedRoot), nil
}

// resolveTestOwnedPath resolves every existing path component, then appends
// any not-yet-created suffix. This catches symlink escapes while still allowing
// registration before git creates the worktree directory.
func resolveTestOwnedPath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(absolutePath)
	var missing []string
	for {
		if _, err := os.Lstat(candidate); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(candidate)
			if resolveErr != nil {
				return "", resolveErr
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("no existing ancestor for %q", absolutePath)
		}
		missing = append(missing, filepath.Base(candidate))
		candidate = parent
	}
}
