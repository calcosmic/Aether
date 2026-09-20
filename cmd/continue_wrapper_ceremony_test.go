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
		"## Required Cross-Stage State",
		"**Purpose:**",
		"**Reads:**",
		"**Stop conditions:**",
		"<success_criteria>",
		"<failure_modes>",
		"<read_only>",
		"Why this matters",
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
			// D-10 item 9 (CONTEXT.md regression fence): no wrapper-driven git
			// stash or commit; checkpointing, if it ever returns, is runtime
			// work. continue-finalize.md:273 carries a literal
			// `git add -A && git commit` immediately inside the finalize
			// ceremony text, and continue-verify.md:81 carries a `git stash
			// pop` rollback procedure -- both files are named in CONTEXT.md's
			// canonical mining list, so a mining pass could copy either
			// verbatim alongside the prose that is actually wanted.
			"git stash",
			"git add -A",
			"git commit",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s should not contain stale direct continue command %q", wrapperPath, forbidden)
			}
		}

		assertSubstringsInOrder(t, wrapperPath, text, inOrder)

		for _, forbidden := range []string{"safe to clear your context now.", "/ant-resume"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s should not contain %q (runtime owns context-clear)", wrapperPath, forbidden)
			}
		}
	}

	// Runtime-level assertion: verify renderContinueVisual() tells the owner
	// whether it is safe to close the chat. The safe-to-close CLAIM is
	// verified, not assumed: it appears only when .aether/HANDOFF.md is
	// actually on disk, so the fixture writes one; the no-file case must hold
	// the user back.
	//
	// Phase 197 plan 04: the sentence now comes from the shared closing card
	// rather than from renderContextClearGuidance, so the wording asserted
	// below is the card's. The contract is unchanged -- the runtime owns the
	// verdict, and the claim still requires the file to be seen.
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Runtime contract check"
	now := time.Now()
	state := colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateBUILT, CurrentPhase: 1, BuildStartedAt: &now}
	phase := colony.Phase{ID: 1, Name: "Contract check"}

	const safeToClose = "it is safe to close this chat"
	const holdTheChat = "Don't close this chat yet"

	// Without a handoff on disk: an explicit hold, never the claim.
	heldOutput := renderContinueVisual(state, phase, nil, false, &colony.Phase{ID: 2, Name: "Next"}, nil, colony.VerificationDepthLight)
	if strings.Contains(heldOutput, safeToClose) {
		t.Errorf("renderContinueVisual() claims it is safe to close the chat with no handoff on disk\n%s", heldOutput)
	}
	if !strings.Contains(heldOutput, holdTheChat) {
		t.Errorf("renderContinueVisual() missing the explicit hold when the handoff is absent\n%s", heldOutput)
	}

	handoffPath := filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md")
	if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(handoffPath, []byte("# handoff\n"), 0644); err != nil {
		t.Fatalf("write handoff: %v", err)
	}

	// Non-final case
	nonFinalOutput := renderContinueVisual(state, phase, nil, false, &colony.Phase{ID: 2, Name: "Next"}, nil, colony.VerificationDepthLight)
	if !strings.Contains(nonFinalOutput, safeToClose) {
		t.Errorf("renderContinueVisual() non-final never says whether it is safe to close the chat\n%s", nonFinalOutput)
	}

	// Final case
	finalOutput := renderContinueVisual(state, phase, nil, true, nil, nil, colony.VerificationDepthLight)
	if !strings.Contains(finalOutput, safeToClose) {
		t.Errorf("renderContinueVisual() final never says whether it is safe to close the chat\n%s", finalOutput)
	}

	// A blocked check reads the same rule as every other screen: with the
	// handover note gone from disk, it holds the owner back rather than
	// claiming it is safe to walk away.
	if err := os.Remove(handoffPath); err != nil {
		t.Fatalf("remove handoff: %v", err)
	}
	blockedOutput := renderContinueBlockedVisual(state, phase, nil, colony.VerificationDepthLight)
	if strings.Contains(blockedOutput, safeToClose) {
		t.Errorf("renderContinueBlockedVisual() claims it is safe to close the chat with no handover note on disk\n%s", blockedOutput)
	}
	if !strings.Contains(blockedOutput, holdTheChat) {
		t.Errorf("renderContinueBlockedVisual() never tells the owner whether to close the chat\n%s", blockedOutput)
	}
}

// TestContinueWrapperInstructsCapsuleAndPheromoneDelivery proves 189-02 D-13:
// both canonical continue wrapper paths (Claude Code and OpenCode) instruct
// reading continue_manifest.context_capsule and .pheromone_section once from
// the manifest and delivering dispatch.skill_section per reviewer -- the same
// delivery formula build.md already states for dispatch_manifest.context_capsule.
// Before this plan's fix, continue.md's entire delivery instruction was "Pass
// each dispatch's runtime-provided `brief` verbatim", naming none of these
// three keys.
func TestContinueWrapperInstructsCapsuleAndPheromoneDelivery(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// 190-VERIFICATION (third pass): the capsule is the SOLE carrier of
	// pheromone signals on the continue wrapper flow — the runtime no longer
	// populates continue_manifest.pheromone_section, and the wrapper prose
	// must not instruct prepending it (that concatenation shipped every
	// active signal to heavy-depth reviewers twice). This test previously
	// required the pheromone_section delivery instruction; flipped
	// deliberately alongside the manifest change in codex_continue_plan.go.
	required := []string{
		"continue_manifest.context_capsule",
		"SOLE source of pheromone signals",
		"dispatch.skill_section",
	}
	forbidden := []string{
		// The old concatenation instructions, in both shapes they appeared.
		"+ `continue_manifest.pheromone_section`",
		"and `continue_manifest.pheromone_section`",
	}

	for _, wrapperPath := range canonicalWrapperPaths(repoRoot, "continue") {
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}
		text := string(content)
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing delivery instruction %q -- a wrapper-spawned continue reviewer would never receive it", wrapperPath, want)
			}
		}
		for _, ban := range forbidden {
			if strings.Contains(text, ban) {
				t.Errorf("%s still instructs prepending pheromone_section (%q) -- the capsule is the sole carrier; concatenating both delivers every active signal twice", wrapperPath, ban)
			}
		}
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

// TestContinueWrapperStageSkeletonAndParity extends the continue wrapper
// contract with the D-05 stage-skeleton density check, the D-09
// ordered-heading-parity check, the D-06 structured method blocks, the
// CMD-02 method-outweighs-envelope-mechanics proportion check, and a
// widened context-clear / D-10-item-9 git-mutation fence that now also
// covers the flat installed mirror (.claude/commands/ant-continue.md).
func TestContinueWrapperStageSkeletonAndParity(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	canonicalPaths := canonicalWrapperPaths(repoRoot, "continue")
	exemptHeadings := []string{
		"## Ownership Split",
		"## Required Cross-Stage State",
		"## Cross-Platform Drift Guard",
		"## Guardrails",
	}

	t.Run("stage_skeleton_density", func(t *testing.T) {
		for _, path := range canonicalPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			assertStageSkeletonDensity(t, path, string(content), exemptHeadings)
		}
	})

	t.Run("ordered_heading_parity", func(t *testing.T) {
		claudeContent, err := os.ReadFile(canonicalPaths[0])
		if err != nil {
			t.Fatalf("read %s: %v", canonicalPaths[0], err)
		}
		opencodeContent, err := os.ReadFile(canonicalPaths[1])
		if err != nil {
			t.Fatalf("read %s: %v", canonicalPaths[1], err)
		}
		assertOrderedHeadingParity(t, "continue", stripCommentLines(string(claudeContent)), stripCommentLines(string(opencodeContent)))
	})

	t.Run("structured_blocks_present", func(t *testing.T) {
		required := []string{
			"<success_criteria>",
			"<failure_modes>",
			"<read_only>",
			"## Required Cross-Stage State",
		}
		for _, path := range canonicalPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, want := range required {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing structured block %q", path, want)
				}
			}
		}
	})

	t.Run("method_outweighs_envelope_mechanics", func(t *testing.T) {
		for _, path := range canonicalPaths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			methodCount := countMarkerLines(text, stageSkeletonMarkers())
			envelopeCount := countMarkerLines(text, envelopeMechanicsMarkers())
			if methodCount < 3*envelopeCount {
				t.Errorf("%s: stage-skeleton marker lines (%d) must be at least 3x envelope-mechanics marker lines (%d)", path, methodCount, envelopeCount)
			}
		}
	})

	t.Run("context_clear_stays_runtime_owned", func(t *testing.T) {
		paths := append(append([]string{}, canonicalPaths...), flatMirrorPath(repoRoot, "continue"))
		forbidden := []string{
			"safe to clear your context now.",
			"/ant-resume",
			// D-10 item 9 (CONTEXT.md regression fence), widened to the flat
			// mirror alongside the context-clear fence -- see the comment on
			// TestContinueWrapperCeremonyContract's forbidden slice above for
			// the source-line citations.
			"git stash",
			"git add -A",
			"git commit",
		}
		for _, path := range paths {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, want := range forbidden {
				if strings.Contains(text, want) {
					t.Errorf("%s should not contain %q (runtime owns context-clear / D-10 item 9 git fence)", path, want)
				}
			}
		}
	})
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
