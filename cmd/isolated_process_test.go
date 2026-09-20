package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	isolatedProcessChildEnv = "AETHER_ISOLATED_PROCESS_TEST_CHILD"
	isolatedProcessHubEnv   = "AETHER_HUB_DIR"
)

const (
	isolatedProcessSuccessProbeName = "TestIsolatedProcessHelperSuccessProbe"
	isolatedProcessFailureProbeName = "TestIsolatedProcessHelperFailureProbe"
	isolatedProcessTimeoutProbeName = "TestIsolatedProcessHelperTimeoutProbe"
)

const (
	isolatedProcessFallbackCommandTimeout = 4 * time.Minute
	isolatedProcessParentDeadlineCushion  = 10 * time.Second
	isolatedProcessChildExitCushion       = 2 * time.Second
	isolatedProcessWaitDelay              = 2 * time.Second
)

var isolatedProcessSemaphore = make(chan struct{}, 4)

var isolatedProcessTestNames = map[string]struct{}{
	"TestReviewerArtifactExternalFinalizeFailsClosed":                  {},
	"TestReviewerArtifactDirectFlowFailsClosed":                        {},
	"TestLegacyReviewerSeverityExternalFinalizeFailsClosed":            {},
	"TestSuggestionOnlyCriticalExternalFinalizeFailsBeforeAdvancement": {},
	"TestSuggestionOnlyCriticalDirectFlowFailsBeforeAdvancement":       {},
	"TestFullLifecycleInDownstreamRepo":                                {},
	"TestTerritoryLifecycleTransactionalPublish":                       {},
	"TestPauseResume199SafeBoundary":                                   {},
	"TestSealTransaction199HivePolicy":                                 {},
	"TestEveryLifecycleCommandEndsWithNextAction":                      {},
	isolatedProcessSuccessProbeName:                                    {},
	isolatedProcessFailureProbeName:                                    {},
	isolatedProcessTimeoutProbeName:                                    {},
}

type isolatedProcessBudget struct {
	testTimeout    time.Duration
	commandTimeout time.Duration
}

// runIsolatedProcessTest runs the named test body directly only in its
// explicitly identified child process. The parent participates in Go's
// parallel scheduler, then limits this package to four concurrent children.
func runIsolatedProcessTest(t *testing.T, testName string, body func(*testing.T)) {
	t.Helper()
	if err := validateIsolatedProcessTestName(testName); err != nil {
		t.Fatal(err)
	}
	if t.Name() != testName {
		t.Fatalf("isolated process wrapper name = %q, want exact top-level name %q", t.Name(), testName)
	}
	if isIsolatedProcessChildInvocation(testName) {
		body(t)
		return
	}

	t.Parallel()
	isolatedProcessSemaphore <- struct{}{}
	defer func() { <-isolatedProcessSemaphore }()

	output, err := runIsolatedTestBinary(t, testName)
	if err != nil {
		t.Fatalf("isolated child %s failed: %v\n%s", testName, err, output)
	}
}

func runIsolatedTestBinary(t *testing.T, testName string) ([]byte, error) {
	t.Helper()
	deadline, hasDeadline := t.Deadline()
	budget, err := isolatedProcessBudgetForDeadline(time.Now(), deadline, hasDeadline)
	if err != nil {
		return nil, err
	}
	return runIsolatedTestBinaryWithBudget(t, testName, budget)
}

func runIsolatedTestBinaryWithBudget(t *testing.T, testName string, budget isolatedProcessBudget) ([]byte, error) {
	t.Helper()
	if err := validateIsolatedProcessTestName(testName); err != nil {
		return nil, err
	}
	deadline, hasDeadline := t.Deadline()
	budget, err := clampIsolatedProcessBudgetForDeadline(budget, time.Now(), deadline, hasDeadline)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), budget.commandTimeout)
	defer cancel()
	return withIsolatedProcessTestHub(ctx, func(hubDir string) ([]byte, error) {
		contextDeadline, ok := ctx.Deadline()
		if !ok {
			return nil, errors.New("isolated process command context has no deadline")
		}
		childBudget, err := isolatedProcessBudgetForRemaining(budget, time.Until(contextDeadline))
		if err != nil {
			return nil, err
		}

		command, err := isolatedTestCommand(ctx, testName, hubDir, os.Getpid(), childBudget)
		if err != nil {
			return nil, err
		}
		output, runErr := command.CombinedOutput()
		if ctxErr := ctx.Err(); ctxErr != nil {
			return output, errors.Join(runErr, fmt.Errorf("isolated child %s exceeded setup-and-command timeout %s: %w", testName, budget.commandTimeout, ctxErr))
		}
		return output, runErr
	})
}

func isolatedTestCommand(ctx context.Context, testName, hubDir string, parentPID int, budget isolatedProcessBudget) (*exec.Cmd, error) {
	if err := validateIsolatedProcessTestName(testName); err != nil {
		return nil, err
	}
	if err := validateIsolatedProcessBudget(budget); err != nil {
		return nil, err
	}
	if strings.TrimSpace(hubDir) == "" {
		return nil, errors.New("isolated process child hub is empty")
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve current test binary: %w", err)
	}

	exactRun := "^" + regexp.QuoteMeta(testName) + "$"
	command := exec.CommandContext(ctx, executable,
		"-test.run="+exactRun,
		"-test.count=1",
		"-test.timeout="+budget.testTimeout.String(),
	)
	command.Env = isolatedProcessChildEnvironment(testName, hubDir, parentPID)
	// Process-group controls are not portable across Aether's supported
	// platforms. The Go test timeout terminates the child test binary first;
	// CommandContext forcibly kills that binary if it does not cooperate, and
	// WaitDelay bounds any inherited output pipe held open by a descendant.
	// Together they guarantee CombinedOutput returns so the caller's semaphore
	// release defer can run without OS-specific process handling.
	command.WaitDelay = isolatedProcessWaitDelay
	return command, nil
}

func isolatedProcessBudgetForDeadline(now, deadline time.Time, hasDeadline bool) (isolatedProcessBudget, error) {
	commandTimeout := isolatedProcessFallbackCommandTimeout
	if hasDeadline {
		commandTimeout = deadline.Sub(now) - isolatedProcessParentDeadlineCushion
	}
	if commandTimeout <= isolatedProcessChildExitCushion {
		return isolatedProcessBudget{}, fmt.Errorf("insufficient parent test time for isolated child: %s available after deadline cushion", commandTimeout)
	}
	return isolatedProcessBudget{
		testTimeout:    commandTimeout - isolatedProcessChildExitCushion,
		commandTimeout: commandTimeout,
	}, nil
}

func clampIsolatedProcessBudgetForDeadline(requested isolatedProcessBudget, now, deadline time.Time, hasDeadline bool) (isolatedProcessBudget, error) {
	if err := validateIsolatedProcessBudget(requested); err != nil {
		return isolatedProcessBudget{}, err
	}
	limit, err := isolatedProcessBudgetForDeadline(now, deadline, hasDeadline)
	if err != nil {
		return isolatedProcessBudget{}, err
	}
	if requested.commandTimeout > limit.commandTimeout {
		requested.commandTimeout = limit.commandTimeout
	}
	maxTestTimeout := requested.commandTimeout - isolatedProcessChildExitCushion
	if requested.testTimeout > maxTestTimeout {
		requested.testTimeout = maxTestTimeout
	}
	if err := validateIsolatedProcessBudget(requested); err != nil {
		return isolatedProcessBudget{}, err
	}
	return requested, nil
}

func isolatedProcessBudgetForRemaining(requested isolatedProcessBudget, remaining time.Duration) (isolatedProcessBudget, error) {
	requested.commandTimeout = remaining
	maxTestTimeout := remaining - isolatedProcessChildExitCushion
	if requested.testTimeout > maxTestTimeout {
		requested.testTimeout = maxTestTimeout
	}
	if err := validateIsolatedProcessBudget(requested); err != nil {
		return isolatedProcessBudget{}, fmt.Errorf("insufficient time after isolated child hub setup: %w", err)
	}
	return requested, nil
}

func validateIsolatedProcessBudget(budget isolatedProcessBudget) error {
	if budget.testTimeout <= 0 {
		return fmt.Errorf("isolated child test timeout must be positive, got %s", budget.testTimeout)
	}
	if budget.commandTimeout-budget.testTimeout < isolatedProcessChildExitCushion {
		return fmt.Errorf("isolated child command timeout %s must leave test timeout %s the %s exit cushion", budget.commandTimeout, budget.testTimeout, isolatedProcessChildExitCushion)
	}
	return nil
}

func validateIsolatedProcessTestName(testName string) error {
	if _, allowed := isolatedProcessTestNames[testName]; !allowed {
		return fmt.Errorf("isolated process test name %q is not allowed", testName)
	}
	return nil
}

func isIsolatedProcessChildInvocation(testName string) bool {
	runFlag := flag.Lookup("test.run")
	countFlag := flag.Lookup("test.count")
	if runFlag == nil || countFlag == nil {
		return false
	}
	return validateIsolatedProcessChildInvocation(
		os.Getenv(isolatedProcessChildEnv),
		os.Getppid(),
		testName,
		runFlag.Value.String(),
		countFlag.Value.String(),
	) == nil
}

func validateIsolatedProcessChildInvocation(marker string, actualParentPID int, testName, runSelector, countValue string) error {
	markerPIDText, markerTestName, ok := strings.Cut(marker, ":")
	if !ok || markerPIDText == "" || markerTestName == "" {
		return errors.New("isolated process child marker is malformed")
	}
	markerPID, err := strconv.Atoi(markerPIDText)
	if err != nil || markerPID <= 0 {
		return fmt.Errorf("isolated process child marker parent PID %q is invalid", markerPIDText)
	}
	if markerPID != actualParentPID {
		return fmt.Errorf("isolated process child parent PID = %d, want %d", actualParentPID, markerPID)
	}
	if markerTestName != testName {
		return fmt.Errorf("isolated process child marker test = %q, want %q", markerTestName, testName)
	}
	exactRun := "^" + regexp.QuoteMeta(testName) + "$"
	if runSelector != exactRun {
		return fmt.Errorf("isolated process child run selector = %q, want %q", runSelector, exactRun)
	}
	count, err := strconv.Atoi(countValue)
	if err != nil || count != 1 {
		return fmt.Errorf("isolated process child count = %q, want 1", countValue)
	}
	return nil
}

func isolatedProcessChildMarker(parentPID int, testName string) string {
	return strconv.Itoa(parentPID) + ":" + testName
}

func isolatedProcessChildEnvironment(testName, hubDir string, parentPID int) []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key == isolatedProcessChildEnv || key == isolatedProcessHubEnv {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment,
		isolatedProcessChildEnv+"="+isolatedProcessChildMarker(parentPID, testName),
		isolatedProcessHubEnv+"="+hubDir,
	)
}

func withIsolatedProcessTestHub(ctx context.Context, run func(string) ([]byte, error)) (output []byte, err error) {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("isolated process child hub setup deadline: %w", ctxErr)
	}
	hubDir, err := createIsolatedProcessTestHub()
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(hubDir); cleanupErr != nil {
			err = errors.Join(err, fmt.Errorf("remove isolated process child hub %s: %w", hubDir, cleanupErr))
		}
	}()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, fmt.Errorf("isolated process child hub setup exceeded deadline: %w", ctxErr)
	}
	return run(hubDir)
}

func createIsolatedProcessTestHub() (string, error) {
	versionData, err := isolatedProcessSourceVersion()
	if err != nil {
		return "", err
	}
	hubDir, err := os.MkdirTemp("", "aether-isolated-test-hub-")
	if err != nil {
		return "", fmt.Errorf("create isolated process child hub: %w", err)
	}
	cleanupAfterSetupError := func(setupErr error) (string, error) {
		cleanupErr := os.RemoveAll(hubDir)
		if cleanupErr != nil {
			cleanupErr = fmt.Errorf("remove incomplete isolated process child hub %s: %w", hubDir, cleanupErr)
		}
		return "", errors.Join(setupErr, cleanupErr)
	}
	if err := os.MkdirAll(filepath.Join(hubDir, "system"), 0o755); err != nil {
		return cleanupAfterSetupError(fmt.Errorf("seed isolated process child hub system directory: %w", err))
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), versionData, 0o644); err != nil {
		return cleanupAfterSetupError(fmt.Errorf("seed isolated process child hub version: %w", err))
	}
	return hubDir, nil
}

func isolatedProcessSourceVersion() ([]byte, error) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, errors.New("resolve isolated process helper source path")
	}
	versionPath := filepath.Join(filepath.Dir(sourceFile), "..", ".aether", "version.json")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return nil, fmt.Errorf("read source version for isolated process child hub: %w", err)
	}
	var version struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(versionData, &version); err != nil {
		return nil, fmt.Errorf("decode source version for isolated process child hub: %w", err)
	}
	if strings.TrimSpace(version.Version) == "" {
		return nil, errors.New("source version for isolated process child hub is empty")
	}
	return versionData, nil
}

func TestIsolatedProcessHelperReportsChildStatus(t *testing.T) {
	if deadline, ok := t.Deadline(); ok && time.Until(deadline) < 30*time.Second {
		t.Skipf("less than 30s remains in the parent package budget; launching nested test binaries would violate the required exit cushion")
	}
	output, err := runIsolatedTestBinary(t, isolatedProcessSuccessProbeName)
	if err != nil {
		t.Fatalf("success probe returned an error: %v\n%s", err, output)
	}

	output, err = runIsolatedTestBinary(t, isolatedProcessFailureProbeName)
	if err == nil {
		t.Fatalf("failure probe returned success:\n%s", output)
	}
	if !strings.Contains(string(output), "intentional isolated child failure") {
		t.Fatalf("failure probe output omitted child diagnostic: %v\n%s", err, output)
	}

	timeoutBudget := isolatedProcessBudget{testTimeout: 250 * time.Millisecond, commandTimeout: 5 * time.Second}
	started := time.Now()
	output, err = runIsolatedTestBinaryWithBudget(t, isolatedProcessTimeoutProbeName, timeoutBudget)
	if err == nil {
		t.Fatalf("timeout probe returned success:\n%s", output)
	}
	if !strings.Contains(string(output), "test timed out after") {
		t.Fatalf("timeout probe omitted Go test timeout diagnostic after %s: %v\n%s", time.Since(started), err, output)
	}

	if _, err := isolatedTestCommand(context.Background(), "TestIsolatedProcessHelperSuccessProbe|TestAnythingElse", t.TempDir(), os.Getpid(), timeoutBudget); err == nil {
		t.Fatal("unapproved regex-like test name was accepted")
	}
}

func TestIsolatedProcessHelperValidatesLaunchMarker(t *testing.T) {
	const parentPID = 4242
	testName := isolatedProcessSuccessProbeName
	exactRun := "^" + regexp.QuoteMeta(testName) + "$"
	validMarker := isolatedProcessChildMarker(parentPID, testName)
	tests := []struct {
		name         string
		marker       string
		actualPID    int
		testName     string
		runSelector  string
		countValue   string
		wantAccepted bool
	}{
		{name: "exact child launch", marker: validMarker, actualPID: parentPID, testName: testName, runSelector: exactRun, countValue: "1", wantAccepted: true},
		{name: "ambient legacy marker", marker: testName, actualPID: parentPID, testName: testName, runSelector: exactRun, countValue: "1"},
		{name: "stale parent PID", marker: validMarker, actualPID: parentPID + 1, testName: testName, runSelector: exactRun, countValue: "1"},
		{name: "wrong exact selector", marker: validMarker, actualPID: parentPID, testName: testName, runSelector: "^TestAnythingElse$", countValue: "1"},
		{name: "broad selector", marker: validMarker, actualPID: parentPID, testName: testName, runSelector: testName, countValue: "1"},
		{name: "wrong count", marker: validMarker, actualPID: parentPID, testName: testName, runSelector: exactRun, countValue: "2"},
		{name: "wrong test name", marker: validMarker, actualPID: parentPID, testName: isolatedProcessFailureProbeName, runSelector: exactRun, countValue: "1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIsolatedProcessChildInvocation(tc.marker, tc.actualPID, tc.testName, tc.runSelector, tc.countValue)
			if tc.wantAccepted && err != nil {
				t.Fatalf("valid child launch rejected: %v", err)
			}
			if !tc.wantAccepted && err == nil {
				t.Fatal("invalid child launch was accepted")
			}
		})
	}

	ambientHub := t.TempDir()
	childHub := filepath.Join(t.TempDir(), "child-hub")
	t.Setenv(isolatedProcessChildEnv, isolatedProcessChildMarker(parentPID+1, testName))
	t.Setenv(isolatedProcessHubEnv, ambientHub)
	environment := isolatedProcessChildEnvironment(testName, childHub, parentPID)
	if got := isolatedProcessEnvironmentValues(environment, isolatedProcessChildEnv); len(got) != 1 || got[0] != validMarker {
		t.Fatalf("child marker environment = %v, want one freshly bound marker %q", got, validMarker)
	}
	if got := isolatedProcessEnvironmentValues(environment, isolatedProcessHubEnv); len(got) != 1 || got[0] != childHub {
		t.Fatalf("child hub environment = %v, want one explicit hub %q", got, childHub)
	}
}

func TestIsolatedProcessHelperDerivesBoundedBudget(t *testing.T) {
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	budget, err := isolatedProcessBudgetForDeadline(now, now.Add(90*time.Second), true)
	if err != nil {
		t.Fatal(err)
	}
	if budget.commandTimeout != 80*time.Second || budget.testTimeout != 78*time.Second {
		t.Fatalf("deadline budget = %+v, want command=1m20s test=1m18s", budget)
	}

	budget, err = isolatedProcessBudgetForDeadline(now, time.Time{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if budget.commandTimeout != isolatedProcessFallbackCommandTimeout || budget.testTimeout != isolatedProcessFallbackCommandTimeout-isolatedProcessChildExitCushion {
		t.Fatalf("fallback budget = %+v", budget)
	}

	oversized := isolatedProcessBudget{testTimeout: 23 * time.Hour, commandTimeout: 24 * time.Hour}
	budget, err = clampIsolatedProcessBudgetForDeadline(oversized, now, now.Add(90*time.Second), true)
	if err != nil {
		t.Fatal(err)
	}
	if budget.commandTimeout != 80*time.Second || budget.testTimeout != 78*time.Second {
		t.Fatalf("oversized explicit budget was not clamped to the live deadline: %+v", budget)
	}

	budget, err = isolatedProcessBudgetForRemaining(budget, 25*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if budget.commandTimeout != 25*time.Second || budget.testTimeout != 23*time.Second {
		t.Fatalf("hub setup time was not charged against child execution: %+v", budget)
	}

	if _, err := isolatedProcessBudgetForDeadline(now, now.Add(isolatedProcessParentDeadlineCushion+isolatedProcessChildExitCushion), true); err == nil {
		t.Fatal("deadline with no child execution room was accepted")
	}
	if _, err := clampIsolatedProcessBudgetForDeadline(oversized, now, now.Add(isolatedProcessParentDeadlineCushion+isolatedProcessChildExitCushion), true); err == nil {
		t.Fatal("oversized explicit budget bypassed an insufficient live deadline")
	}
}

func TestIsolatedProcessHelperSeedsAndRemovesChildHub(t *testing.T) {
	ambientHub := t.TempDir()
	t.Setenv(isolatedProcessHubEnv, ambientHub)
	wantVersion, err := isolatedProcessSourceVersion()
	if err != nil {
		t.Fatal(err)
	}

	var childHubs []string
	for iteration := 0; iteration < 2; iteration++ {
		output, err := withIsolatedProcessTestHub(context.Background(), func(childHub string) ([]byte, error) {
			childHubs = append(childHubs, childHub)
			if childHub == ambientHub {
				return nil, errors.New("child reused ambient hub")
			}
			gotVersion, err := os.ReadFile(filepath.Join(childHub, "version.json"))
			if err != nil {
				return nil, err
			}
			if string(gotVersion) != string(wantVersion) {
				return nil, errors.New("child hub version does not match source version")
			}
			info, err := os.Stat(filepath.Join(childHub, "system"))
			if err != nil {
				return nil, err
			}
			if !info.IsDir() {
				return nil, errors.New("child hub system path is not a directory")
			}
			return []byte("seeded"), nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(output) != "seeded" {
			t.Fatalf("hub callback output = %q", output)
		}
		if _, err := os.Stat(childHubs[iteration]); !os.IsNotExist(err) {
			t.Fatalf("parent did not remove child hub %s: %v", childHubs[iteration], err)
		}
	}
	if childHubs[0] == childHubs[1] {
		t.Fatalf("child hub was reused across runs: %s", childHubs[0])
	}
	if _, err := os.Stat(ambientHub); err != nil {
		t.Fatalf("child hub cleanup touched ambient hub: %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	callbackRan := false
	if _, err := withIsolatedProcessTestHub(canceled, func(string) ([]byte, error) {
		callbackRan = true
		return nil, nil
	}); err == nil {
		t.Fatal("canceled setup context was accepted")
	}
	if callbackRan {
		t.Fatal("child callback ran after the setup deadline expired")
	}
}

func isolatedProcessEnvironmentValues(environment []string, key string) []string {
	var values []string
	for _, entry := range environment {
		entryKey, value, ok := strings.Cut(entry, "=")
		if ok && entryKey == key {
			values = append(values, value)
		}
	}
	return values
}

func TestIsolatedProcessHelperSuccessProbe(t *testing.T) {
	if !isIsolatedProcessChildInvocation(t.Name()) {
		return
	}
	assertIsolatedProcessChildHub(t)
}

func TestIsolatedProcessHelperFailureProbe(t *testing.T) {
	if !isIsolatedProcessChildInvocation(t.Name()) {
		return
	}
	assertIsolatedProcessChildHub(t)
	t.Fatal("intentional isolated child failure")
}

func TestIsolatedProcessHelperTimeoutProbe(t *testing.T) {
	if !isIsolatedProcessChildInvocation(t.Name()) {
		return
	}
	assertIsolatedProcessChildHub(t)
	time.Sleep(time.Minute)
}

func assertIsolatedProcessChildHub(t *testing.T) {
	t.Helper()
	hubDir := os.Getenv(isolatedProcessHubEnv)
	if strings.TrimSpace(hubDir) == "" {
		t.Fatal("isolated child has no explicit hub")
	}
	if _, err := os.Stat(filepath.Join(hubDir, "version.json")); err != nil {
		t.Fatalf("isolated child hub version is unavailable: %v", err)
	}
	info, err := os.Stat(filepath.Join(hubDir, "system"))
	if err != nil {
		t.Fatalf("isolated child hub system directory is unavailable: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("isolated child hub system path is not a directory")
	}
}
