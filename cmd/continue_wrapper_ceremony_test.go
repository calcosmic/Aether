package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestContinueWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "continue.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "continue.md"),
	}

	required := []string{
		"Use the Go `aether` CLI as the source of truth.",
		"AETHER_OUTPUT_MODE=visual aether status",
		"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS",
		"aether host continue --dry-run --classic-ceremony $ARGUMENTS",
		"aether host continue --dry-run --verification-depth heavy $ARGUMENTS",
		"The wrapper is the sole conductor for interactive reviewer spawning",
		"temporary manifest file outside `.aether/data/`",
		"result.manifest.continue_manifest",
		"Do not set `run_in_background`",
		"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file",
		"## After Continue",
		"/ant-build N+1",
		"/ant-seal",
	}

	inOrder := []string{
		"## Default Continue",
		"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS",
		"## Heavy External Review",
		"aether host continue --dry-run --classic-ceremony $ARGUMENTS",
		"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file",
		"## After Continue",
	}

	for _, wrapperPath := range wrapperPaths {
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}

		text := string(content)
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing %q", wrapperPath, want)
			}
		}
		if guardrail := "Do NOT use `--plan-only` or `continue-finalize` for default fast continue."; !strings.Contains(text, guardrail) {
			t.Errorf("%s missing guardrail %q", wrapperPath, guardrail)
		}
		for _, forbidden := range []string{
			"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS",
			"AETHER_OUTPUT_MODE=json aether continue --plan-only --verification-depth heavy $ARGUMENTS",
			"The TS host is the sole entry point to the Go CLI for manifest generation.",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s should not contain stale direct continue command %q", wrapperPath, forbidden)
			}
		}

		assertSubstringsInOrder(t, wrapperPath, text, inOrder)

		for _, forbidden := range []string{"It's safe to clear your context now.", "/ant-resume"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s should not contain %q (runtime owns context-clear)", wrapperPath, forbidden)
			}
		}
	}

	// Runtime-level assertion: verify renderContinueVisual() emits context-clear guidance
	goal := "Runtime contract check"
	now := time.Now()
	state := colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1, BuildStartedAt: &now}
	phase := colony.Phase{ID: 1, Name: "Contract check"}

	// Non-final case
	nonFinalOutput := renderContinueVisual(state, phase, nil, false, &colony.Phase{ID: 2, Name: "Next"}, nil, colony.VerificationDepthLight)
	if !strings.Contains(nonFinalOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueVisual() non-final missing context-clear guidance\n%s", nonFinalOutput)
	}

	// Final case
	finalOutput := renderContinueVisual(state, phase, nil, true, nil, nil, colony.VerificationDepthLight)
	if !strings.Contains(finalOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueVisual() final missing context-clear guidance\n%s", finalOutput)
	}

	// Blocked case must NOT contain guidance
	blockedOutput := renderContinueBlockedVisual(state, phase, nil, colony.VerificationDepthLight)
	if strings.Contains(blockedOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueBlockedVisual() should not contain context-clear guidance\n%s", blockedOutput)
	}
}

func TestContinueWrapperSourcesUseFastDevContinue(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	command := "AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS"
	paths := []string{
		filepath.Join(repoRoot, ".aether", "commands", "continue.yaml"),
		filepath.Join(repoRoot, ".claude", "commands", "ant-continue.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "continue.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "continue.md"),
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(content), command) {
			t.Fatalf("%s missing fast-dev continue command %q", path, command)
		}
	}

	for _, path := range []string{
		filepath.Join(repoRoot, ".aether", "commands", "claude", "continue.md"),
		filepath.Join(repoRoot, ".aether", "commands", "opencode", "continue.md"),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("retired command mirror still exists: %s", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func sliceBetweenMarkers(t *testing.T, path, content, start, end string) string {
	t.Helper()

	startIdx := strings.Index(content, start)
	if startIdx < 0 {
		t.Fatalf("%s missing section marker %q", path, start)
	}
	startIdx += len(start)

	endIdx := len(content)
	if end != "" {
		relativeEnd := strings.Index(content[startIdx:], end)
		if relativeEnd < 0 {
			t.Fatalf("%s missing section marker %q", path, end)
		}
		endIdx = startIdx + relativeEnd
	}

	return content[startIdx:endIdx]
}
