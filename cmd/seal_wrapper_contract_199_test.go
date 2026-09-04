package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSealWrapperContract199(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}

	paths := []string{
		".aether/commands/seal.yaml",
		".claude/commands/ant-seal.md",
		".claude/commands/ant/seal.md",
		".opencode/commands/ant/seal.md",
	}
	for _, relativePath := range paths {
		relativePath := relativePath
		t.Run(relativePath, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(relativePath)))
			if err != nil {
				t.Fatalf("read %s: %v", relativePath, err)
			}
			text := string(content)
			for _, required := range []string{
				"Close a verified colony, or explicitly record an owner-forced incomplete closure.",
				"Force flags pass only when directly supplied by the owner.",
				"The Go runtime owns final review, preflight, confirmation, transaction, and rendering.",
				"A forced-incomplete closure is not verified success.",
				"After sealing, run `AETHER_OUTPUT_MODE=visual aether status` first to review the retained sealed state; `aether entomb` is a separate optional owner-confirmed archive-and-clear action.",
				"Never invoke entomb automatically",
			} {
				if !strings.Contains(text, required) {
					t.Errorf("%s missing %q", relativePath, required)
				}
			}
			for _, forbidden := range []string{
				"--force --reason \"<their words>\"",
				"then rerun with `--force",
			} {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s must not create force authority or auto-close: found %q", relativePath, forbidden)
				}
			}
		})
	}
}
