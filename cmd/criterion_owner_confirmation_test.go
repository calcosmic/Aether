package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestOwnerConfirmationAnsweredRejectsCrossScopeResolution(t *testing.T) {
	const (
		phaseID        = 6
		taskID         = "6.1"
		criterion      = "The exported ledger matches the owner-approved layout"
		otherTaskID    = "6.2"
		otherCriterion = "The archive download opens in the desktop client"
	)
	initializedAt := time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	createdAt := initializedAt.Add(time.Minute)
	currentGoal := "Finish the scoped export flow"
	currentSession := "session-current-owner-confirmation"

	tests := []struct {
		name           string
		sessionID      string
		goalHash       string
		wantAnswered   bool
		wantBlockers   int
		wantTargetLive bool
	}{
		{
			name:         "current scope answer removes only its matching blocker",
			sessionID:    currentSession,
			goalHash:     pendingDecisionGoalHash(currentGoal),
			wantAnswered: true,
			wantBlockers: 1,
		},
		{
			name:           "foreign session leaves the seal blocker live",
			sessionID:      "session-foreign-owner-confirmation",
			goalHash:       pendingDecisionGoalHash(currentGoal),
			wantBlockers:   2,
			wantTargetLive: true,
		},
		{
			name:           "foreign goal leaves the seal blocker live",
			goalHash:       pendingDecisionGoalHash("Replace the export flow"),
			wantBlockers:   2,
			wantTargetLive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			s, _ := newTestStore(t)
			store = s
			state := colony.ColonyState{
				Version:       "3.0",
				Goal:          &currentGoal,
				SessionID:     &currentSession,
				InitializedAt: &initializedAt,
				Plan: colony.Plan{Phases: []colony.Phase{{
					ID:     phaseID,
					Name:   "Owner confirmation scope",
					Status: colony.PhaseCompleted,
				}}},
			}
			if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatalf("seed current colony state: %v", err)
			}
			phase := phaseID
			if err := s.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
				ID:          "pd_owner_confirmation_answer",
				Type:        clarificationDecisionType,
				Description: formatClarificationDescription(ownerConfirmationQuestionText(phaseID, taskID, criterion), nil),
				Phase:       &phase,
				Source:      "owner-confirmation-test",
				SessionID:   tt.sessionID,
				GoalHash:    tt.goalHash,
				Resolution:  "confirmed",
				Resolved:    true,
				CreatedAt:   createdAt.Format(time.RFC3339),
				ResolvedAt:  createdAt.Add(time.Minute).Format(time.RFC3339),
			}}}); err != nil {
				t.Fatalf("seed legacy owner answer: %v", err)
			}
			report := codexContinueVerificationReport{
				Phase:       phaseID,
				GeneratedAt: createdAt.Format(time.RFC3339),
				Criteria: []codexCriterionVerification{
					{TaskID: taskID, Criterion: criterion, State: criterionStateNeedsOwnerConfirmation},
					{TaskID: otherTaskID, Criterion: otherCriterion, State: criterionStateNeedsOwnerConfirmation},
				},
			}
			if err := s.SaveJSON(continuePlanArtifactsPath(phaseID, "verification.json"), report); err != nil {
				t.Fatalf("seed owner-confirmation verification: %v", err)
			}

			if got := ownerConfirmationAnswered(phaseID, taskID, criterion); got != tt.wantAnswered {
				t.Fatalf("ownerConfirmationAnswered = %v, want %v", got, tt.wantAnswered)
			}
			blockers := ownerConfirmationSealBlockers(state)
			if len(blockers) != tt.wantBlockers {
				t.Fatalf("seal blockers = %d, want %d: %+v", len(blockers), tt.wantBlockers, blockers)
			}
			targetLive := false
			otherLive := false
			for _, blocker := range blockers {
				targetLive = targetLive || strings.Contains(blocker.Description, criterion)
				otherLive = otherLive || strings.Contains(blocker.Description, otherCriterion)
			}
			if targetLive != tt.wantTargetLive {
				t.Fatalf("matching criterion blocker live = %v, want %v: %+v", targetLive, tt.wantTargetLive, blockers)
			}
			if !otherLive {
				t.Fatalf("unanswered sibling criterion blocker was removed: %+v", blockers)
			}
		})
	}
}

// TestOwnerConfirmationCommandIsShellSafe proves CR-02 (193-REVIEW.md): a
// criterion's free-form text can contain shell metacharacters -- a command
// substitution ($(...)), a backtick pair, and a bare $VAR reference -- and
// the "exact command to paste into a shell" ownerConfirmationCommand
// builds for the (explicitly non-technical, per this repo's own CLAUDE.md)
// project owner must neutralize all of them. This runs the rendered
// command through a REAL POSIX shell (not just string inspection) to
// prove nothing expands, substitutes, or executes, and that the
// --question value a real shell parses out round-trips exactly back to
// the original question text -- so ownerConfirmationAnswered's exact-text
// match still resolves the same criterion.
func TestOwnerConfirmationCommandIsShellSafe(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available in this environment")
	}
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	marker := filepath.Join(root, "pwned.txt")
	criterion := fmt.Sprintf("Handles `echo hi`, $(touch %s), and a $HOME reference correctly", marker)
	phaseID := 7
	taskID := "3.2"

	cmdStr := ownerConfirmationCommand(phaseID, taskID, criterion)
	if !strings.Contains(cmdStr, "'") {
		t.Fatalf("expected the rendered command to use shell single-quoting, got %q", cmdStr)
	}

	// Round-trip cmdStr through a REAL shell: "set --" splits it the exact
	// way a shell would when the owner pastes it in, then each resulting
	// argument is dumped NUL-separated so this test can recover the
	// --question value without a second layer of shell interpretation.
	script := "set -- " + strings.TrimPrefix(cmdStr, "aether ") + "\nfor a in \"$@\"; do printf '%s\\0' \"$a\"; done"
	out, err := exec.Command("sh", "-c", script).Output()
	if err != nil {
		t.Fatalf("round-trip shell invocation failed: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatalf("expected the embedded $(...) NEVER to execute, but marker file was created: %s", marker)
	}

	args := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var question string
	found := false
	for i, a := range args {
		if a == "--question" && i+1 < len(args) {
			question = args[i+1]
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find --question among round-tripped args, got %q", args)
	}

	expected := ownerConfirmationQuestionText(phaseID, taskID, criterion)
	if question != expected {
		t.Fatalf("round-tripped --question value does not match the original question text:\ngot:  %q\nwant: %q", question, expected)
	}

	// Prove the round trip is not just textually equal but functionally
	// equivalent to what the owner running this command would produce:
	// recording the round-tripped question as an answer resolves the SAME
	// criterion ownerConfirmationAnswered checks against.
	if ownerConfirmationAnswered(phaseID, taskID, criterion) {
		t.Fatalf("test premise broken: criterion already answered before recording anything")
	}
	if _, err := recordDecisionAnswer(question, "confirmed", phaseID, "owner-confirmation-test"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}
	if !ownerConfirmationAnswered(phaseID, taskID, criterion) {
		t.Fatalf("expected ownerConfirmationAnswered to resolve after recording the round-tripped question text")
	}
}

// TestShellQuoteEscapesEmbeddedSingleQuotes proves shellQuote's own
// contract directly: an embedded single quote must not terminate the
// quoted string early, and the escaped form must round-trip through a
// real shell back to the original text.
func TestShellQuoteEscapesEmbeddedSingleQuotes(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available in this environment")
	}
	original := `it's a "test" with $VAR and ` + "`backticks`"
	quoted := shellQuote(original)

	out, err := exec.Command("sh", "-c", "printf '%s' "+quoted).Output()
	if err != nil {
		t.Fatalf("round-trip shell invocation failed: %v", err)
	}
	if string(out) != original {
		t.Fatalf("shellQuote did not round-trip: got %q want %q", out, original)
	}
}
