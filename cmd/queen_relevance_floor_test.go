package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 182. Measured against the live runtime on 2026-08-14 using the real
// CalVault phase text -- copying 110 markdown files into a folder tree in an
// Obsidian vault containing no program code:
//
//	riskLevel:             low
//	hasSecuritySignal:     false
//	gatekeeper relevance:  0
//	requiredCastes:        [builder probe watcher]
//
// Two defects, both confirmed rather than suspected:
//
//  1. A Probe -- the test-coverage specialist -- is *required* on a phase with
//     no code to cover. queenPhaseIsDocumentationOnly needs one of eight
//     documentation words to be present, and "copy markdown files into a folder
//     tree" contains none, so the phase is classified as code work.
//
//  2. The runtime scored the security specialist at 0 and did not require it.
//     It ran anyway, for 107,155 tokens, because the Queen asked for it and
//     queenApplyJudgement never refuses a caste for having nothing to do. The
//     rule already exists one file away -- keywordGatedCastes in
//     caste_relevance.go returns 0 precisely so that "a caste dispatched to a
//     phase it has no work on ... costs a worker run and reads to the user as
//     the colony being confused" -- but only the deterministic engine consults
//     it. The Queen's proposal bypasses the check written to prevent this.

func calVaultPhase() colony.Phase {
	return colony.Phase{
		ID:          3,
		Name:        "Template recovery",
		Description: "Recover ~173 lost templates from copies in sibling vaults, dedupe, and reinstate as a top-level TEMPLATES/ folder. Sources read-only.",
		SuccessCriteria: []string{
			"All 110 identified templates are copied into the new folder tree.",
			"A log records every copy with its source path.",
		},
		Tasks: []colony.Task{{
			Goal: "Copy 110 already-identified markdown files from known absolute paths into a new folder tree, keeping a log.",
			Constraints: []string{
				"Never create them through Obsidian's in-app New note command: .obsidian/plugins/templater-obsidian/data.json has trigger_on_file_creation true and a single file_templates rule whose regex .+ matches every new file vault-wide.",
				"Source vaults are read-only; do not modify them.",
			},
			Hints: []string{"The recovery matrix lists every source path and dedupe verdict."},
		}},
	}
}

func securityPhase() colony.Phase {
	return colony.Phase{
		ID:              4,
		Name:            "Password reset by email",
		Description:     "Let a user reset their password with an emailed token, and store the credential hash.",
		SuccessCriteria: []string{"A reset token expires after one hour."},
		Tasks: []colony.Task{{
			Goal: "Implement the password reset endpoint and its token store.",
		}},
	}
}

// A repository whose only files are notes. A test-coverage specialist here can
// do nothing but report that it found nothing.
func notesOnlyRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Daily note.md"), []byte("# note\n"), 0644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	withWorkingDir(t, dir)
}

func codeRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	withWorkingDir(t, dir)
}

func TestProbeNotRequiredInARepositoryWithNoCode(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	required := queenBuildSafetyRequiredCastes(calVaultPhase())
	for _, caste := range required {
		if caste == "probe" {
			t.Fatalf("a test-coverage specialist is required on a phase copying markdown files in a repository with no program code; required = %v", required)
		}
	}
}

// TestProbeStillRequiredWhenTheRepositoryHasCode is retired -- see
// .aether/docs/retired-tests-ledger.md. Its "always required at standard
// continue" claim was superseded by plan 194-05 (D-13): standard depth now
// requires nothing unconditionally, Probe included. The code-detection gate
// it protected survives as a REFUSAL rule instead
// (TestProbeIsRequiredOnlyWhereItCanFindSomething's "produces testable code"
// subtest, cmd/queen_probe_gating_test.go), and the full depth table is
// pinned by TestContinueRequiredSetByDepth (cmd/owner_dials_test.go).

func TestZeroRelevanceCasteIsRefused(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	phase := calVaultPhase()
	if score := casteRelevanceScore(phase, "gatekeeper"); score != 0 {
		t.Fatalf("precondition: expected the security specialist to score 0 on this phase, got %d", score)
	}

	judgement := queenApplyJudgement(
		[]string{"builder", "watcher", "gatekeeper"},
		"a security review seems prudent",
		phase, "build", colony.ColonyState{},
	)

	for _, caste := range judgement.Final {
		if caste == "gatekeeper" {
			t.Fatalf("a specialist the runtime scored at zero relevance was still dispatched; final = %v", judgement.Final)
		}
	}
	if len(judgement.Refused) == 0 {
		t.Fatal("the refusal was silent; the operator must be told a specialist was dropped and why")
	}
}

func TestSecurityCasteSurvivesWhenThePhaseNeedsIt(t *testing.T) {
	saveGlobals(t)
	codeRepo(t)

	phase := securityPhase()
	if score := casteRelevanceScore(phase, "gatekeeper"); score == 0 {
		t.Fatalf("precondition: a password reset phase should score the security specialist above zero")
	}

	judgement := queenApplyJudgement(
		[]string{"builder", "watcher", "gatekeeper"},
		"this changes credentials",
		phase, "build", colony.ColonyState{},
		map[string]string{
			"watcher":    "confirming the password/token change actually works",
			"gatekeeper": "this changes credentials",
		},
	)

	found := false
	for _, caste := range judgement.Final {
		if caste == "gatekeeper" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the security specialist was refused on work that changes passwords and tokens; final = %v, refused = %v", judgement.Final, judgement.Refused)
	}
}

// The relevance floor must never be able to remove a caste the phase requires.
// Cost control and the thing that checks the work are different decisions, and
// this is the one that must not be traded away.
//
// Plan 194-02 (D-07) shrank the build floor to the builder alone, so this
// phase's precondition (non-empty required-caste set) now holds on builder
// rather than watcher -- watcher is no longer unconditionally required at
// build, and asserting it specifically here would just reassert the deleted
// rule. The claim this test protects is unchanged: whatever IS required must
// survive judgement.
func TestRequiredCasteIsNeverRefusedForRelevance(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	phase := calVaultPhase()
	required := queenBuildSafetyRequiredCastes(phase)
	if len(required) == 0 {
		t.Fatal("precondition: expected this phase to require at least the builder")
	}

	judgement := queenApplyJudgement([]string{"builder"}, "", phase, "build", colony.ColonyState{})

	finalSet := map[string]bool{}
	for _, caste := range judgement.Final {
		finalSet[caste] = true
	}
	for _, caste := range required {
		if !finalSet[caste] {
			t.Fatalf("required caste %q was lost; final = %v", caste, judgement.Final)
		}
	}
}

func TestJudgementSummaryNamesWhatItRefused(t *testing.T) {
	saveGlobals(t)
	notesOnlyRepo(t)

	judgement := queenApplyJudgement(
		[]string{"builder", "watcher", "gatekeeper"},
		"a security review seems prudent",
		calVaultPhase(), "build", colony.ColonyState{},
	)

	summary := judgement.Summary()
	if !strings.Contains(summary, "gatekeeper") {
		t.Fatalf("summary does not name the refused specialist: %q", summary)
	}
	if !strings.Contains(strings.ToLower(summary), "nothing") && !strings.Contains(strings.ToLower(summary), "no ") {
		t.Fatalf("summary does not say why the specialist was refused: %q", summary)
	}
}
