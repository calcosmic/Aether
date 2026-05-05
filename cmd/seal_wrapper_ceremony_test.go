package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSealWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, rel := range []string{
		".claude/commands/ant/seal.md",
		".opencode/commands/ant/seal.md",
	} {
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, rel))
			if err != nil {
				t.Fatalf("read wrapper: %v", err)
			}
			text := string(data)
			required := []string{
				"AETHER_OUTPUT_MODE=json aether seal --plan-only $ARGUMENTS",
				"result.seal_manifest",
				"Gatekeeper, Auditor, and Probe",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether spawn-complete",
				"AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file",
				"Do NOT run `aether seal` without `--plan-only`",
			}
			for _, needle := range required {
				if !strings.Contains(text, needle) {
					t.Fatalf("%s missing required text %q", rel, needle)
				}
			}
			forbidden := []string{
				"manually update milestone via COLONY_STATE.json",
				"archive_dir",
				".aether/aether-utils.sh",
			}
			for _, needle := range forbidden {
				if strings.Contains(text, needle) {
					t.Fatalf("%s contains forbidden text %q", rel, needle)
				}
			}
		})
	}
}
