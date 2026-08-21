package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVisualsConfigDoesNotReappear is the Phase 191 criterion 2 reappearance
// ratchet for colony/ceremony/visuals.md, deleted in 191-03. Every field the
// file ever defined already matched cmd/codex_visuals.go's compiled
// defaults exactly (see cmd/codex_visuals_test.go's
// TestVisualsConfigFoldedDefaultsMatchOriginalFile and
// TestVisualsConfigOriginalFileValuesMatchCompiledDefaults), and the file's
// own YAML frontmatter carried a pre-existing parse defect that meant it
// never successfully loaded in any dev checkout, ever (see
// TestVisualsConfigOriginalFileHadPreexistingParseDefect) --
// loadVisualsConfig() (cmd/visuals_config.go) is kept as a
// permanently-fallback path per 191-CONTEXT.md's D-04. This is a Tier-1,
// pure file-existence ratchet (191-PATTERNS.md's "The Ratchet House Style")
// -- proportionate for a single deleted file, no AST scanning needed.
//
// If this file reappears, either it has a genuine new reader (update the
// fold in cmd/codex_visuals.go with that evidence and this ratchet) or it
// should be deleted again.
func TestVisualsConfigDoesNotReappear(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	p := filepath.Join(root, "colony", "ceremony", "visuals.md")
	if _, err := os.Stat(p); err == nil {
		t.Errorf("colony/ceremony/visuals.md has reappeared -- this file was ruled zero-behavioral-effect and deleted in Phase 191 plan 03 (ROADMAP criterion 2, the third of the four CWD-relative silent-fallback loaders); if it is back, either it has a new reader (update the fold in cmd/codex_visuals.go with the evidence and this ratchet) or it should be deleted again")
	}
}
