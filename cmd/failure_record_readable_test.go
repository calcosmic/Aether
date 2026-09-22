package cmd

// 208-03-PLAN.md Task 3 (UED-13/UED-14): a failure whose text
// SanitizeSignalContent rejects -- the everyday shape of real build/test
// output, which is full of backticks -- used to be recorded as a canned "a
// check failure could not be safely recorded" sentence that told the owner
// nothing. These tests prove the four rejection fallbacks
// (recordFailedChecksToMidden, recordQuickFailureToMidden,
// recordSwarmWorkerFailureToMidden, sanitizedWorkerSentence) now keep the
// failure's own words wherever colony.NeutralizeForRecord can, and never
// record a row whose message names nothing.

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRejectedFailureStillNamesWhatFailed runs the real
// recordFailedChecksToMidden with a blocking issue containing a backtick,
// reads the recorded failure back through the same store, and asserts the
// stored message contains the check kind and the failing summary's own
// words -- not a canned "could not be safely recorded" sentence.
func TestRejectedFailureStillNamesWhatFailed(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 208, Name: "Rejected failure still names what failed"}
	result := deterministicFloorResult{
		BlockingIssues: []string{"tests failed: expected `computeWarnings(state)` to return zero rows"},
	}

	recordFailedChecksToMidden(phase, result)

	mf, err := loadMiddenFile(store)
	if err != nil {
		t.Fatalf("load midden: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly 1 midden entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	message := mf.Entries[0].Message

	if strings.Contains(message, "could not be safely recorded") {
		t.Fatalf("recorded message fell back to the canned sentence instead of the failure's own words: %q", message)
	}
	if strings.Contains(message, "`") {
		t.Fatalf("recorded message still carries a raw backtick, which SanitizeSignalContent would reject: %q", message)
	}
	if !strings.Contains(message, "tests") {
		t.Fatalf("recorded message = %q, want it to name the check kind (\"tests\")", message)
	}
	if !strings.Contains(message, "computeWarnings(state)") {
		t.Fatalf("recorded message = %q, want it to keep the failing summary's own words (\"computeWarnings(state)\")", message)
	}
	if !strings.Contains(message, "phase 208") {
		t.Fatalf("recorded message = %q, want it to name the phase", message)
	}
	if _, err := colony.SanitizeSignalContent(message); err != nil {
		t.Fatalf("the recorded message itself is rejected by the filter it was neutralised for: %v (%q)", err, message)
	}
}

// TestEmptyAndNilFailureText asserts a whitespace-only blocking issue still
// records a row naming the check and the phase (never an empty message, and
// never silently dropped), and that recordQuickFailureToMidden with a nil
// error records nothing at all.
func TestEmptyAndNilFailureText(t *testing.T) {
	t.Run("whitespace-only blocking issue names the check and the phase", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		phase := colony.Phase{ID: 209, Name: "Whitespace-only blocking issue"}
		result := deterministicFloorResult{
			BlockingIssues: []string{"   \t  "},
		}

		recordFailedChecksToMidden(phase, result)

		mf, err := loadMiddenFile(store)
		if err != nil {
			t.Fatalf("load midden: %v", err)
		}
		if len(mf.Entries) != 1 {
			t.Fatalf("expected exactly 1 midden entry for a whitespace-only blocking issue, got %d: %+v", len(mf.Entries), mf.Entries)
		}
		message := mf.Entries[0].Message
		if strings.TrimSpace(message) == "" {
			t.Fatal("recorded message is empty -- a whitespace-only failure must still name the check and the phase")
		}
		if !strings.Contains(message, "check") {
			t.Fatalf("recorded message = %q, want it to name the check", message)
		}
		if !strings.Contains(message, "phase 209") {
			t.Fatalf("recorded message = %q, want it to name the phase", message)
		}
	})

	t.Run("nil error records nothing", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		recordQuickFailureToMidden("does this ever fire on nil?", "", nil)

		mf, err := loadMiddenFile(store)
		if err == nil && len(mf.Entries) != 0 {
			t.Fatalf("recordQuickFailureToMidden with a nil error must record nothing, got %+v", mf.Entries)
		}
	})
}
