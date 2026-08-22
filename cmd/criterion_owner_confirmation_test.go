package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
