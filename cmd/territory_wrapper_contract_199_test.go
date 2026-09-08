package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestTerritoryWrapperAuthority199 freezes the wrapper/runtime ownership
// boundary around automatic territory refresh. Platform wrappers may conduct
// manifest-declared workers, but only the Go host and finalizers may classify
// freshness, validate completion evidence, publish survey state, or mutate the
// accepted plan.
func TestTerritoryWrapperAuthority199(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	type authoritySurface struct {
		name     string
		path     string
		required []string
	}
	path := func(rel string) string { return filepath.Join(repoRoot, filepath.FromSlash(rel)) }
	surfaces := []authoritySurface{
		{
			name: "colonize YAML",
			path: path(".aether/commands/colonize.yaml"),
			required: []string{
				"source_of_truth", "aether host colonize", "result.colonize_manifest",
				"output_paths", "aether colonize-finalize --completion-file",
				"let the runtime write survey artifacts", "do not parse visual output",
				"do not write colony state files",
			},
		},
		{
			name: "plan YAML",
			path: path(".aether/commands/plan.yaml"),
			required: []string{
				"source_of_truth", "aether host plan", "plan_manifest",
				"aether plan-finalize --completion-file", "one runtime-authorized stage at a time",
				"never parse visual output as state", "do not edit planning artifacts",
			},
		},
	}

	for _, platform := range []struct {
		name string
		dir  string
	}{
		{name: "Claude canonical", dir: ".claude/commands/ant"},
		{name: "OpenCode", dir: ".opencode/commands/ant"},
	} {
		surfaces = append(surfaces,
			authoritySurface{
				name: platform.name + " colonize",
				path: path(filepath.Join(platform.dir, "colonize.md")),
				required: []string{
					"runtime owns final survey artifacts", "aether host colonize",
					"result.colonize_manifest", "manifest is the only source",
					"output paths", "files_created", "aether colonize-finalize --completion-file",
					"do not parse it as state", "do not hand-edit `.aether/data/`",
				},
			},
			authoritySurface{
				name: platform.name + " plan",
				path: path(filepath.Join(platform.dir, "plan.md")),
				required: []string{
					"aether spec --inspect", "aether host plan", "result.plan_manifest",
					"exactly one authorized scout dispatch", "aether plan-finalize --completion-file",
					"all authoritative mutation occurs through go commands", "never parse visual output as authority",
					"this wrapper never edits specification projections",
				},
			},
		)
	}
	for _, verb := range []string{"colonize", "plan"} {
		surfaces = append(surfaces, authoritySurface{
			name: "Claude flat " + verb,
			path: path(filepath.Join(".claude", "commands", "ant-"+verb+".md")),
			required: func() []string {
				if verb == "colonize" {
					return []string{"runtime owns final survey artifacts", "aether host colonize", "result.colonize_manifest", "aether colonize-finalize --completion-file"}
				}
				return []string{"aether spec --inspect", "aether host plan", "result.plan_manifest", "aether plan-finalize --completion-file"}
			}(),
		})
	}

	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`(?im)^\s*(?:echo|printf|cat|tee|jq|sed|awk|perl|python(?:3)?)\b[^\n]*(?:\.aether/data/survey|COLONY_STATE\.json|session\.json)`),
		regexp.MustCompile(`(?i)\b(?:territory|survey)\s+(?:is\s+)?(?:fresh|stale|missing|unavailable)\b`),
		regexp.MustCompile(`(?im)^\s*(?:[-*]\s*)?(?:assume|synthesize|fabricate|default)\b[^\n]{0,80}\b(?:success|completed|passed)\b`),
		regexp.MustCompile(`(?i)--(?:skip|no)[-_]?(?:snapshot|territory|survey|verify|validation|finalize)`),
		regexp.MustCompile(`(?im)^\s*(?:sed|awk|perl|python(?:3)?)\b[^\n]*(?:ansi|\\x1b|escape sequence)`),
	}

	texts := make(map[string]string, len(surfaces))
	for _, surface := range surfaces {
		content, err := os.ReadFile(surface.path)
		if err != nil {
			t.Fatalf("read %s (%s): %v", surface.name, surface.path, err)
		}
		text := string(content)
		lower := strings.ToLower(text)
		texts[surface.path] = text
		for _, required := range surface.required {
			if !strings.Contains(lower, strings.ToLower(required)) {
				t.Errorf("%s does not delegate authority through %q", surface.name, required)
			}
		}
		for _, pattern := range forbidden {
			if match := pattern.FindString(text); match != "" {
				t.Errorf("%s contains host-owned territory behavior %q", surface.name, match)
			}
		}
	}

	for _, verb := range []string{"colonize", "plan"} {
		canonical := path(filepath.Join(".claude", "commands", "ant", verb+".md"))
		for _, mirror := range []string{
			path(filepath.Join(".claude", "commands", "ant-"+verb+".md")),
			path(filepath.Join(".opencode", "commands", "ant", verb+".md")),
		} {
			if texts[canonical] != texts[mirror] {
				t.Errorf("%s wrapper mirror drifted from canonical Claude authority contract", mirror)
			}
		}
	}
}
