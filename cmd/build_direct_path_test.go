package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildWrapperUsesDirectGoPath is the WP3 drift guard: every surface that
// describes the interactive build flow must fetch the dispatch manifest
// directly from the Go runtime (`aether build <phase> --plan-only`) and must
// not route the fetch through the TS host. The host remains available for
// other flows (plan) but is off the primary build path.
func TestBuildWrapperUsesDirectGoPath(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	surfaces := []struct {
		path     string
		required string
	}{
		{".claude/commands/ant/build.md", "aether build $ARGUMENTS --plan-only"},
		{".opencode/commands/ant/build.md", "aether build $ARGUMENTS --plan-only"},
		{".aether/commands/build.yaml", "aether build $ARGUMENTS --plan-only"},
		{".aether/skills/colony/aether-colony-build-cycle/SKILL.md", "aether build <phase> --plan-only"},
	}

	for _, surface := range surfaces {
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(surface.path)))
		if err != nil {
			t.Fatalf("read %s: %v", surface.path, err)
		}
		text := string(content)
		if !strings.Contains(text, surface.required) {
			t.Errorf("%s does not fetch the manifest via the direct Go path (%q missing)", surface.path, surface.required)
		}
		if strings.Contains(text, "aether host build --dry-run") {
			t.Errorf("%s still routes the manifest fetch through the TS host", surface.path)
		}
		if strings.Contains(text, "result.manifest.dispatch_manifest") {
			t.Errorf("%s still parses the TS host envelope shape", surface.path)
		}
	}

	// The Codex command guide is generated from Go; assert the catalog text.
	def, ok := commandGuideCatalog()["build"]
	if !ok {
		t.Fatal("command guide catalog has no build entry")
	}
	guide := strings.Join(def.PreSteps, "\n")
	if !strings.Contains(guide, "aether build <phase> --plan-only") {
		t.Error("Codex build guide does not fetch the manifest via the direct Go path")
	}
	if strings.Contains(guide, "aether host build") {
		t.Error("Codex build guide still routes the manifest fetch through the TS host")
	}
}
