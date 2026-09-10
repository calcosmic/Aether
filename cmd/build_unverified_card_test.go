package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 05 (D-05) -- the honest unverified build card. A build
// finishing under the check-step verification boundary (D-01, the default)
// must say plainly that the work is built, the program's own checks passed,
// verification has not run, and name the one command that runs it -- with
// no token borrowed from the success verdict's label and the identical full
// ceremony a verified card gets (plan 201-04's equal-ceremony guarantee).
// See cmd/codex_build.go for the mechanism this file proves
// (buildUnverifiedCloseoutDetails, buildVerifiedCloseoutDetails).

// buildCloseoutFixture builds one "build" command closeout using
// lifecycleCloseout199Projection's own established test infrastructure
// (cmd/lifecycle_closeout_199_test.go), with the colony left in the BUILT
// state so the projection's own Next Up naturally resolves to the real
// "run continue" recommendation -- this test never invents one.
func buildCloseoutFixture(t *testing.T, details LifecycleCloseoutDetails) LifecycleCloseout {
	t.Helper()
	projection := lifecycleCloseout199Projection(colony.StateBUILT, "build")
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "build", details); err != nil {
		t.Fatalf("apply build closeout: %v", err)
	}
	return lifecycleCloseout199MustResult(t, result)
}

// TestBuildBeforeVerificationSaysItIsNotVerifiedYet is the honest-unverified
// proof: a build closeout built under the check-step boundary (D-01) never
// carries the success verdict, names the exact deterministic checks that
// passed and the one command to run next, and shares no token with the
// success verdict's own label.
func TestBuildBeforeVerificationSaysItIsNotVerifiedYet(t *testing.T) {
	checkCommands := []string{"go build ./...", "go vet ./...", "go test ./..."}
	closeout := buildCloseoutFixture(t, buildUnverifiedCloseoutDetails(checkCommands))

	if closeout.WorkOutcome == nil {
		t.Fatalf("expected a work outcome on the unverified build closeout, got none: %+v", closeout)
	}
	if *closeout.WorkOutcome == colony.WorkOutcomeSuccess {
		t.Fatalf("unverified build closeout carries the success verdict: %+v", closeout)
	}

	visual := renderLifecycleCloseout(closeout, "claude")
	for _, command := range checkCommands {
		if !strings.Contains(visual, command) {
			t.Errorf("rendered card omits the deterministic check command %q:\n%s", command, visual)
		}
	}
	lowered := strings.ToLower(visual)
	if !strings.Contains(lowered, "not been verified") && !strings.Contains(lowered, "not verified") {
		t.Errorf("rendered card does not plainly say verification has not run:\n%s", visual)
	}

	nextCommand := strings.TrimSpace(closeout.NextUp.RuntimeCommand)
	if nextCommand == "" {
		t.Fatalf("expected the projection to name one next command, got none: %+v", closeout.NextUp)
	}
	// The rendered card translates the runtime command through the
	// platform's own owner-facing wrapper vocabulary (e.g. "aether continue"
	// -> "/ant-continue" for platform "claude", per writeVisualOutput's
	// single translation chokepoint) -- so this checks the "Next Up" slot
	// names an actionable check-the-phase command, not a byte match against
	// the untranslated runtime string.
	if !strings.Contains(lowered, "next up") || !strings.Contains(lowered, "continue") {
		t.Errorf("rendered card does not name a next command to check and advance the phase:\n%s", visual)
	}

	// The card shares no token with the success verdict's label -- derived
	// from colony.WorkOutcomeLabels() here, never a copy of the wording, so
	// this cannot go stale when the label wording changes (matches
	// TestNonSuccessVerdictBorrowsNoSuccessWording's own discipline,
	// cmd/lifecycle_closeout_work_outcome_test.go).
	labels := colony.WorkOutcomeLabels()
	successTokens := lifecycleWorkOutcomeTokens(labels[colony.WorkOutcomeSuccess])
	if len(successTokens) == 0 {
		t.Fatalf("success label produced no tokens to check against: %q", labels[colony.WorkOutcomeSuccess])
	}
	cardTokens := lifecycleWorkOutcomeTokens(visual)
	for token := range successTokens {
		if cardTokens[token] {
			t.Errorf("unverified build card shares token %q with the success verdict's label %q:\n%s", token, labels[colony.WorkOutcomeSuccess], visual)
		}
	}

	// No repository-invented vocabulary -- the owner reads this card without
	// having opened a file (CLAUDE.md's own "READ THIS BEFORE YOU WRITE
	// ANYTHING TO THE OWNER" rule; mirrors the section-heading wording check
	// TestLifecycleCloseout199Contract already applies to this same render
	// path, cmd/lifecycle_closeout_199_test.go).
	for _, jargon := range []string{"verification boundary", "the queen", "post-wave", "deterministic floor", "caste"} {
		if strings.Contains(lowered, jargon) {
			t.Errorf("rendered card contains repository-invented vocabulary %q:\n%s", jargon, visual)
		}
	}
}

// TestUnverifiedCardKeepsTheFullCeremony reuses the equal-ceremony
// invariant plan 201-04 built (colony.WorkOutcome, the closed six-verdict
// vocabulary; lifecycleCloseoutSlots' full-ceremony gate): an unverified
// build card renders the identical canonical slot set a verified build card
// renders -- a non-success verdict never looks less considered than a
// passing one.
func TestUnverifiedCardKeepsTheFullCeremony(t *testing.T) {
	unverified := buildCloseoutFixture(t, buildUnverifiedCloseoutDetails([]string{"go build ./..."}))
	verified := buildCloseoutFixture(t, buildVerifiedCloseoutDetails([]string{"go build ./..."}, []string{"Auditor"}))

	if !reflect.DeepEqual(unverified.Slots, verified.Slots) {
		t.Fatalf("unverified card slots = %#v, want the same full set a verified card renders: %#v", unverified.Slots, verified.Slots)
	}
	if len(unverified.Slots) != len(lifecycleCloseoutCanonicalSlots) {
		t.Fatalf("unverified card slot count = %d, want every canonical slot (%d)", len(unverified.Slots), len(lifecycleCloseoutCanonicalSlots))
	}
	if verified.WorkOutcome == nil || *verified.WorkOutcome != colony.WorkOutcomeSuccess {
		t.Fatalf("fixture is broken: the verified comparison card must carry the success verdict, got %+v", verified.WorkOutcome)
	}
}
