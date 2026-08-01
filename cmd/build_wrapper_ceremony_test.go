package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"),
	}

	required := []string{
		"Use the Go `aether` CLI as the source of truth.",
		"AETHER_OUTPUT_MODE=visual aether status",
		"## Active Signals",
		"REDIRECT",
		"FOCUS",
		"FEEDBACK",
		"strength or remaining-life context",
		"## Phase Framing",
		"Phase N of M -- Name",
		"## Dispatch Manifest",
		"aether host build --dry-run",
		"temporary manifest file outside `.aether/data/`",
		"result.manifest.dispatch_manifest",
		"## Runtime Spawn Ceremony",
		"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest_file>",
		"Do not set `run_in_background`",
		"Do NOT run `aether host build` without `--dry-run` from this wrapper",
		"Do NOT run `aether build --synthetic` after real",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
		"/ant-continue",
		"dispatch_manifest.context_capsule",
		"brief_path",
	}

	inOrder := []string{
		"## Ownership Split",
		"## Colony Context",
		"## Active Signals",
		"## Phase Framing",
		"## Dispatch Manifest",
		"aether host build --dry-run",
		"## Runtime Spawn Ceremony",
		"AETHER_OUTPUT_MODE=json aether build-finalize $ARGUMENTS --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file",
		"## After the Build",
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
		for _, forbidden := range []string{
			"Do NOT load playbooks",
			"\nAETHER_OUTPUT_MODE=visual aether build $ARGUMENTS\n",
			"\nAETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only\n",
			"Do NOT run `aether build` without `--plan-only` from this wrapper.",
			"Do NOT run direct `aether build` from this wrapper for manifest generation",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s still contains old pass-through contract %q", wrapperPath, forbidden)
			}
		}
	}
}

// briefPathContractSurfaces is the exact four-surface list named by
// .aether/commands/build.yaml:16's drift_guard ("Update this YAML,
// Claude/OpenCode wrappers, the Codex skill, and cmd/command_guide.go
// together."). A change touching fewer than all four breaks that guard's own
// promise, so TestBriefPathReferencedAcrossAllFourSurfaces asserts len == 4
// to catch a future edit that quietly drops a surface.
var briefPathContractSurfaces = []string{
	".claude/commands/ant/build.md",
	".opencode/commands/ant/build.md",
	".aether/skills/colony/aether-colony-build-cycle/SKILL.md",
	"cmd/command_guide.go",
}

// TestBriefPathReferencedAcrossAllFourSurfaces proves the D-12 brief_path
// contract is sanctioned on every one of the four drift-guarded surfaces:
// each must mention both `brief_path` and (case-insensitively) "verbatim".
// It fails if brief_path is deleted from any one of the four files.
func TestBriefPathReferencedAcrossAllFourSurfaces(t *testing.T) {
	if len(briefPathContractSurfaces) != 4 {
		t.Fatalf("briefPathContractSurfaces has %d entries, want exactly 4 (per .aether/commands/build.yaml:16 drift_guard)", len(briefPathContractSurfaces))
	}

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range briefPathContractSurfaces {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(content)
		if !strings.Contains(text, "brief_path") {
			t.Errorf("%s missing brief_path", path)
		}
		if !strings.Contains(strings.ToLower(text), "verbatim") {
			t.Errorf("%s missing a case-insensitive mention of verbatim", path)
		}
	}
}

// TestBuildWrapperVerbatimBriefBulletsStayMirrored proves the .claude and
// .opencode build.md wrappers' VERBATIM-brief bullet (worker spawning
// section) and prompt-recipe bullet (worker prompt composition) remain
// textually identical -- catching drift where one wrapper gains brief_path
// wording the other loses.
func TestBuildWrapperVerbatimBriefBulletsStayMirrored(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeContent, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"))
	if err != nil {
		t.Fatalf("read claude build.md: %v", err)
	}
	opencodeContent, err := os.ReadFile(filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"))
	if err != nil {
		t.Fatalf("read opencode build.md: %v", err)
	}

	claudeText := string(claudeContent)
	opencodeText := string(opencodeContent)

	claudeBriefBullet := mustExtractLineContaining(t, claudeText, "Each dispatch carries `brief`")
	opencodeBriefBullet := mustExtractLineContaining(t, opencodeText, "Each dispatch carries `brief`")
	if claudeBriefBullet != opencodeBriefBullet {
		t.Errorf("VERBATIM-brief bullet drifted between wrappers:\nclaude:   %s\nopencode: %s", claudeBriefBullet, opencodeBriefBullet)
	}

	claudeRecipeBullet := mustExtractLineContaining(t, claudeText, "The worker's prompt =")
	opencodeRecipeBullet := mustExtractLineContaining(t, opencodeText, "The worker's prompt =")
	if claudeRecipeBullet != opencodeRecipeBullet {
		t.Errorf("prompt-recipe bullet drifted between wrappers:\nclaude:   %s\nopencode: %s", claudeRecipeBullet, opencodeRecipeBullet)
	}
}

func mustExtractLineContaining(t *testing.T, content, marker string) string {
	t.Helper()
	line := extractLineContaining(content, marker)
	if line == "" {
		t.Fatalf("no line containing %q found", marker)
	}
	return line
}

func assertSubstringsInOrder(t *testing.T, path, content string, ordered []string) {
	t.Helper()

	cursor := 0
	for _, needle := range ordered {
		idx := strings.Index(content[cursor:], needle)
		if idx < 0 {
			t.Fatalf("%s missing ordered marker %q", path, needle)
		}
		cursor += idx + len(needle)
	}
}
