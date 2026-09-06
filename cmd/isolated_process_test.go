package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

const isolatedProcessChildEnv = "AETHER_ISOLATED_PROCESS_TEST_CHILD"

const (
	isolatedProcessSuccessProbeName = "TestIsolatedProcessHelperSuccessProbe"
	isolatedProcessFailureProbeName = "TestIsolatedProcessHelperFailureProbe"
)

var isolatedProcessSemaphore = make(chan struct{}, 4)

var isolatedProcessTestNames = map[string]struct{}{
	"TestReviewerArtifactExternalFinalizeFailsClosed":                  {},
	"TestReviewerArtifactDirectFlowFailsClosed":                        {},
	"TestLegacyReviewerSeverityExternalFinalizeFailsClosed":            {},
	"TestSuggestionOnlyCriticalExternalFinalizeFailsBeforeAdvancement": {},
	"TestSuggestionOnlyCriticalDirectFlowFailsBeforeAdvancement":       {},
	"TestFullLifecycleInDownstreamRepo":                                {},
	isolatedProcessSuccessProbeName:                                    {},
	isolatedProcessFailureProbeName:                                    {},
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
	if os.Getenv(isolatedProcessChildEnv) == testName {
		body(t)
		return
	}

	t.Parallel()
	isolatedProcessSemaphore <- struct{}{}
	defer func() { <-isolatedProcessSemaphore }()

	output, err := runIsolatedTestBinary(testName)
	if err != nil {
		t.Fatalf("isolated child %s failed: %v\n%s", testName, err, output)
	}
}

func runIsolatedTestBinary(testName string) ([]byte, error) {
	command, err := isolatedTestCommand(testName)
	if err != nil {
		return nil, err
	}
	return command.CombinedOutput()
}

func isolatedTestCommand(testName string) (*exec.Cmd, error) {
	if err := validateIsolatedProcessTestName(testName); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve current test binary: %w", err)
	}

	exactRun := "^" + regexp.QuoteMeta(testName) + "$"
	command := exec.Command(executable, "-test.run="+exactRun, "-test.count=1")
	command.Env = isolatedProcessChildEnvironment(testName)
	return command, nil
}

func validateIsolatedProcessTestName(testName string) error {
	if _, allowed := isolatedProcessTestNames[testName]; !allowed {
		return fmt.Errorf("isolated process test name %q is not allowed", testName)
	}
	return nil
}

func isolatedProcessChildEnvironment(testName string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if key == isolatedProcessChildEnv || key == "AETHER_HUB_DIR" {
			continue
		}
		environment = append(environment, entry)
	}
	return append(environment, isolatedProcessChildEnv+"="+testName)
}

func TestIsolatedProcessHelperReportsChildStatus(t *testing.T) {
	output, err := runIsolatedTestBinary(isolatedProcessSuccessProbeName)
	if err != nil {
		t.Fatalf("success probe returned an error: %v\n%s", err, output)
	}

	output, err = runIsolatedTestBinary(isolatedProcessFailureProbeName)
	if err == nil {
		t.Fatalf("failure probe returned success:\n%s", output)
	}
	if !strings.Contains(string(output), "intentional isolated child failure") {
		t.Fatalf("failure probe output omitted child diagnostic: %v\n%s", err, output)
	}

	if _, err := isolatedTestCommand("TestIsolatedProcessHelperSuccessProbe|TestAnythingElse"); err == nil {
		t.Fatal("unapproved regex-like test name was accepted")
	}
}

func TestIsolatedProcessHelperSuccessProbe(t *testing.T) {
	if os.Getenv(isolatedProcessChildEnv) != t.Name() {
		return
	}
}

func TestIsolatedProcessHelperFailureProbe(t *testing.T) {
	if os.Getenv(isolatedProcessChildEnv) != t.Name() {
		return
	}
	t.Fatal("intentional isolated child failure")
}
