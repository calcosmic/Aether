package cmd

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestResumeAfterRetiredStalePauseReconstructs is Phase 210 blocker 12
// (French Basics, 2026-09-27): the project was paused, the runtime then kept
// writing state (a check ran on the paused project), so resume correctly
// refused the now-conflicting pause evidence and named the documented exit --
// retire the stale handoff with `aether state-mutate --field pause_handoff
// --value null`, then resume reconstructs. After following that exit, resume
// still tried to complete the stale pause transaction and failed with
// "receipt conflicts with coordinator stage", a permanent loop. Once the
// owner has retired the handoff, resume must reconstruct instead.
func TestResumeAfterRetiredStalePauseReconstructs(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	// The runtime keeps writing after the pause, as a check run on a paused
	// project did.
	state := load199State(t)
	state.Events = append(state.Events, "2026-09-27T10:49:03Z|continue|runtime|check ran on a paused project")
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write after pause: %v", err)
	}
	first, err := resumeColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("first resume should stop, not error: %v", err)
	}
	if first.Provenance != colony.RecoveryProvenanceConflicting {
		t.Fatalf("first resume provenance = %q, want conflicting", first.Provenance)
	}
	if !strings.Contains(first.Message, "pause_handoff") {
		t.Fatalf("conflict did not name the retire-the-handoff exit: %q", first.Message)
	}

	// Follow the named exit exactly.
	retired := load199State(t)
	retired.PauseHandoff = nil
	if err := store.SaveJSON("COLONY_STATE.json", retired); err != nil {
		t.Fatalf("retire handoff: %v", err)
	}
	second, err := resumeColonyAt(fixture.now.Add(2 * time.Minute))
	if err != nil {
		t.Fatalf("resume after retiring the stale handoff dead-ended: %v", err)
	}
	if second.Provenance != colony.RecoveryProvenanceReconstructed {
		t.Fatalf("resume after retiring provenance = %q, want reconstructed (message %q)", second.Provenance, second.Message)
	}
	if load199State(t).Paused {
		t.Fatal("reconstructed resume left the project paused")
	}
}

// TestContinueRefusesAPausedProjectBeforeWriting is the other half of Phase
// 210 blocker 12: `aether continue` on a paused project ran six minutes of
// checks, recorded their results into state, and only then refused with
// "colony is paused" -- and those writes are what made the pause evidence
// conflict. Both continue lanes now refuse first, naming resume, and write
// nothing.
func TestContinueRefusesAPausedProjectBeforeWriting(t *testing.T) {
	for _, lane := range []string{"direct", "wrapper"} {
		t.Run(lane, func(t *testing.T) {
			fixture := newPauseResume199Fixture(t)
			state := load199State(t)
			state.State = colony.StateBUILT
			if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
				t.Fatalf("seed built state: %v", err)
			}
			if _, err := pauseColonyAt(fixture.now); err != nil {
				t.Fatalf("pause: %v", err)
			}
			before := runnableFingerprint199(t, fixture.root, fixture.dataDir)
			var err error
			if lane == "direct" {
				_, _, _, _, _, _, err = runCodexContinue(fixture.root, codexContinueOptions{})
			} else {
				_, _, _, _, err = runCodexContinuePlanOnly(fixture.root, codexContinueOptions{})
			}
			var typed refusal
			if !errors.As(err, &typed) || typed.ID != "continue-on-paused-project" {
				t.Fatalf("continue on a paused project did not refuse up front: %v", err)
			}
			if !strings.Contains(typed.Error(), "aether resume") {
				t.Fatalf("refusal does not name resume: %q", typed.Error())
			}
			if after := runnableFingerprint199(t, fixture.root, fixture.dataDir); after != before {
				t.Fatal("continue wrote state on a paused project")
			}
		})
	}
}
