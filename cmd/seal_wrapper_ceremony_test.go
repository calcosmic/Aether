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
				"aether host seal $ARGUMENTS",
				"result.seal_manifest",
				"temporary manifest file outside `.aether/data/`",
				"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow seal --manifest-file <manifest_file>",
				"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow seal",
				// 205-04: the wrapper no longer promises a fixed reviewer trio.
				// The reviewer claim itself is held to the runtime by
				// TestSealWrapperReviewClaimMatchesTheRuntime; this contract only
				// keeps the honest phrasing in place.
				"never a fixed list",
				"AETHER_OUTPUT_MODE=json aether spawn-log",
				"AETHER_OUTPUT_MODE=json aether spawn-complete",
				"AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow seal --worker-file <worker_file>",
				"AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file",
				"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file",
				"Do NOT bypass `aether host seal`",
				// The confirmation gate remains runtime-owned: the state-of-play
				// card, review, explicit owner question, and recorded answer must
				// remain present. Phase 199 tightened the final boundary: Autopilot
				// never seals, stops at the explicit seal boundary, and leaves
				// owner-supplied force and confirmation to the Go runtime.
				"awaiting_owner_confirmation",
				"state-of-play card",
				"\"what did we learn\" review",
				"Finish this project?",
				"Finish anyway with",
				"result.question",
				"result.next",
				"Autopilot never seals a project.",
				"It stops at the explicit seal boundary",
				"Force flags pass only when directly supplied by the owner.",
				"owner-supplied force request",
				"owner-confirmation question.",
				"Go runtime.",
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
