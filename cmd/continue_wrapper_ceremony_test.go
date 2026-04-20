package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS",
		"## Verification Gates",
		"Gatekeeper",
		"Auditor",
		"Probe",
		"## Learning Extraction",
		"## After Continue",
		"### If the phase advanced",
		"/ant:build N+1",
		"### If continue is blocked",
		"/ant:continue",
		"### If the colony completed",
		"/ant:seal",
		"It's safe to clear your context now.",
		"/ant:resume",
	}

	inOrder := []string{
		"## What Continue Means",
		"## Verification Gates",
		"## Learning Extraction",
		"## Execute",
		"AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS",
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

		assertSubstringsInOrder(t, wrapperPath, text, inOrder)

		advancedSection := sliceBetweenMarkers(t, wrapperPath, text, "### If the phase advanced", "### If continue is blocked")
		for _, want := range []string{"/ant:build N+1", "It's safe to clear your context now.", "/ant:resume"} {
			if !strings.Contains(advancedSection, want) {
				t.Errorf("%s advanced section missing %q", wrapperPath, want)
			}
		}

		blockedSection := sliceBetweenMarkers(t, wrapperPath, text, "### If continue is blocked", "### If the colony completed")
		for _, want := range []string{"/ant:continue"} {
			if !strings.Contains(blockedSection, want) {
				t.Errorf("%s blocked section missing %q", wrapperPath, want)
			}
		}
		for _, forbidden := range []string{"It's safe to clear your context now.", "/ant:resume"} {
			if strings.Contains(blockedSection, forbidden) {
				t.Errorf("%s blocked section should not contain %q", wrapperPath, forbidden)
			}
		}

		finalSection := sliceBetweenMarkers(t, wrapperPath, text, "### If the colony completed", "## Guardrails")
		for _, want := range []string{"/ant:seal", "It's safe to clear your context now.", "/ant:resume"} {
			if !strings.Contains(finalSection, want) {
				t.Errorf("%s final section missing %q", wrapperPath, want)
			}
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
