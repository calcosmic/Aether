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
		"AETHER_OUTPUT_MODE=visual aether status",
		"AETHER_OUTPUT_MODE=json aether continue --plan-only $ARGUMENTS",
		"result.continue_manifest",
		"AETHER_OUTPUT_MODE=json aether spawn-log",
		`subagent_type="{agent_name}"`,
		"AETHER_OUTPUT_MODE=json aether spawn-complete",
		"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
		"## Verification Gates",
		"Gatekeeper",
		"Auditor",
		"Probe",
		"## Learning Extraction",
		"## After Continue",
		"signal housekeeping",
		"what expired, what remained active, and what that means for the next phase",
		"### If the phase advanced",
		"/ant-build N+1",
		"### If continue is blocked",
		"specific recovery command",
		"/ant-continue",
		"### If the colony completed",
		"/ant-seal",
	}

	inOrder := []string{
		"## What Continue Means",
		"## Colony Context",
		"## Continue Manifest",
		"AETHER_OUTPUT_MODE=json aether continue --plan-only $ARGUMENTS",
		"## Wave Execution",
		"## Completion Packet",
		"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
		"## Learning Extraction",
		"## After Continue",
		"### If the phase advanced",
		"### If continue is blocked",
		"### If the colony completed",
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
		if guardrail := "Do NOT run `aether continue` without `--plan-only` from this wrapper."; !strings.Contains(text, guardrail) {
			t.Errorf("%s missing guardrail %q", wrapperPath, guardrail)
		}
		for _, forbidden := range []string{"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s should not contain direct visual continue pass-through %q", wrapperPath, forbidden)
			}
		}

		assertSubstringsInOrder(t, wrapperPath, text, inOrder)

		advancedSection := sliceBetweenMarkers(t, wrapperPath, text, "### If the phase advanced", "### If continue is blocked")
		for _, want := range []string{"/ant-build N+1", "signal housekeeping"} {
			if !strings.Contains(advancedSection, want) {
				t.Errorf("%s advanced section missing %q", wrapperPath, want)
			}
		}
		for _, forbidden := range []string{"It's safe to clear your context now.", "/ant-resume"} {
			if strings.Contains(advancedSection, forbidden) {
				t.Errorf("%s advanced section should not contain %q (runtime owns context-clear)", wrapperPath, forbidden)
			}
		}

		blockedSection := sliceBetweenMarkers(t, wrapperPath, text, "### If continue is blocked", "### If the colony completed")
		for _, want := range []string{"specific recovery command", "/ant-continue"} {
			if !strings.Contains(blockedSection, want) {
				t.Errorf("%s blocked section missing %q", wrapperPath, want)
			}
		}
		for _, forbidden := range []string{"It's safe to clear your context now.", "/ant-resume"} {
			if strings.Contains(blockedSection, forbidden) {
				t.Errorf("%s blocked section should not contain %q", wrapperPath, forbidden)
			}
		}

		finalSection := sliceBetweenMarkers(t, wrapperPath, text, "### If the colony completed", "## Guardrails")
		for _, want := range []string{"/ant-seal", "signal housekeeping"} {
			if !strings.Contains(finalSection, want) {
				t.Errorf("%s final section missing %q", wrapperPath, want)
			}
		}
		for _, forbidden := range []string{"It's safe to clear your context now.", "/ant-resume"} {
			if strings.Contains(finalSection, forbidden) {
				t.Errorf("%s final section should not contain %q (runtime owns context-clear)", wrapperPath, forbidden)
			}
		}
	}

	// Runtime-level assertion: verify renderContinueVisual() emits context-clear guidance
	goal := "Runtime contract check"
	now := time.Now()
	state := colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1, BuildStartedAt: &now}
	phase := colony.Phase{ID: 1, Name: "Contract check"}

	// Non-final case
	nonFinalOutput := renderContinueVisual(state, phase, nil, false, &colony.Phase{ID: 2, Name: "Next"}, nil)
	if !strings.Contains(nonFinalOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueVisual() non-final missing context-clear guidance\n%s", nonFinalOutput)
	}

	// Final case
	finalOutput := renderContinueVisual(state, phase, nil, true, nil, nil)
	if !strings.Contains(finalOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueVisual() final missing context-clear guidance\n%s", finalOutput)
	}

	// Blocked case must NOT contain guidance
	blockedOutput := renderContinueBlockedVisual(state, phase, nil)
	if strings.Contains(blockedOutput, "It's safe to clear your context now.") {
		t.Errorf("renderContinueBlockedVisual() should not contain context-clear guidance\n%s", blockedOutput)
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
