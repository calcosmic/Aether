package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// The status screen (what `aether status` draws) used to be one long block
// of text: versions, loop-safety event dumps with hashes, "0 active" counts,
// proof counters, memory tables. The owner chose the same layout for it as
// for the check screen on 2026-09-27: a verdict box first saying where the
// project stands, then short sections under heavy headers, with zero-count
// noise gone and the full dump one flag away (`aether status --detail`).
// These tests lock that layout on the real renderer, built from the same
// test store and buildStatusResult the command itself uses.

// statusScreenFixture builds a status render the way the command does: the
// shared test store, the flags written where the runtime reads them, the
// state saved back to disk (the closing card re-reads it), then
// buildStatusResult. A private hub carrying the runtime's own version keeps
// the machine's real hub out of the picture.
func statusScreenFixture(t *testing.T, flags []colony.FlagEntry, mutate func(*colony.ColonyState)) (colony.ColonyState, *storage.Store, map[string]interface{}) {
	t.Helper()
	saveGlobals(t)
	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	t.Setenv("AETHER_ROOT", root)
	hubDir := filepath.Join(t.TempDir(), ".aether")
	if err := os.MkdirAll(hubDir, 0o755); err != nil {
		t.Fatalf("hub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(fmt.Sprintf(`{"version": %q}`, resolveVersion())), 0o644); err != nil {
		t.Fatalf("hub version: %v", err)
	}
	t.Setenv("AETHER_HUB_DIR", hubDir)
	writeTestFlags(t, flags...)
	// The fixture's older flags.json fallback carries a blocker; the shared
	// decision store written above is what the runtime reads first, so drop
	// the legacy copy to keep each case's flags exactly what it wrote.
	_ = os.Remove(filepath.Join(s.BasePath(), "flags.json"))

	state := loadStatusTestState(t, s)
	now := time.Now().UTC()
	state.InitializedAt = &now
	state.Plan.GeneratedAt = &now
	if err := writeInstalledVersionMarker(root, resolveVersion()); err != nil {
		t.Fatal(err)
	}

	goal := "Safe Basics installer for the flashcard app"
	state.Goal = &goal
	if mutate != nil {
		mutate(&state)
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	return state, s, buildStatusResult(state, s)
}

func renderStatusScreenFixture(t *testing.T, flags []colony.FlagEntry, mutate func(*colony.ColonyState)) string {
	t.Helper()
	state, s, result := statusScreenFixture(t, flags, mutate)
	return stripANSI(renderDashboard(state, s, result))
}

func statusReady(state *colony.ColonyState) {
	state.State = colony.StateREADY
	state.BuildStartedAt = nil
	state.Paused = false
}

func TestStatusScreenLeadsWithTheVerdict(t *testing.T) {
	cases := []struct {
		name     string
		flags    []colony.FlagEntry
		mutate   func(*colony.ColonyState)
		kind     string
		verdict  string
		sentence string
	}{
		{
			name: "ready", mutate: statusReady, kind: "phase",
			verdict: "PHASE 2 OF 3 — READY TO BUILD", sentence: "Core Features. 1 phase finished.",
		},
		{
			name: "paused",
			mutate: func(state *colony.ColonyState) {
				state.Paused = true
			},
			kind: "paused", verdict: "PAUSED", sentence: "Run `aether resume` to pick up where you left off.",
		},
		{
			name:   "blocked",
			flags:  []colony.FlagEntry{{ID: "b1", Type: "blocker", Description: "the release gate is red"}},
			mutate: statusReady, kind: "blocked",
			verdict: "BLOCKED — 1 problem needs you", sentence: "Phase 2 of 3: Core Features. Fix it before building on.",
		},
		{
			name: "finished",
			mutate: func(state *colony.ColonyState) {
				state.State = colony.StateCOMPLETED
				state.BuildStartedAt = nil
				for i := range state.Plan.Phases {
					state.Plan.Phases[i].Status = colony.PhaseCompleted
				}
			},
			kind: "finished", verdict: "PROJECT FINISHED", sentence: "All 3 phases are done.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderStatusScreenFixture(t, tc.flags, tc.mutate)
			box := firstNonBlankLines(rendered, 4)
			if len(box) != 4 {
				t.Fatalf("expected a four-line verdict box at the top, got %q\n%s", box, rendered)
			}
			if !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
				t.Errorf("verdict box is not framed by full heavy lines: %q\n%s", box, rendered)
			}
			if want := voiceLine(tc.kind, tc.verdict); strings.TrimSpace(box[1]) != want {
				t.Errorf("verdict line = %q, want %q\n%s", strings.TrimSpace(box[1]), want, rendered)
			}
			if strings.TrimSpace(box[2]) != tc.sentence {
				t.Errorf("verdict sentence = %q, want %q\n%s", strings.TrimSpace(box[2]), tc.sentence, rendered)
			}
			top := strings.Index(rendered, box[0])
			if strings.TrimSpace(rendered[:top]) != "" {
				t.Errorf("something is drawn above the verdict box:\n%s", rendered[:top])
			}
			body := checkScreenBody(rendered)
			assertSectionsSeparated(t, body, []string{"Progress"})
			if !strings.Contains(body, "Full detail: `aether status --detail`") {
				t.Errorf("status screen drops detail without saying where to find it:\n%s", body)
			}
		})
	}
}

// statusBusyFixture is a realistic busy project: open blockers, issues and
// notes past the cap, three steering notes, lessons learned, twenty old
// failures waiting for review, and loop-safety events carrying hashes.
func statusBusyFlags() []colony.FlagEntry {
	flags := []colony.FlagEntry{
		{ID: "b1", Type: "blocker", Description: "the release gate is red"},
		{ID: "b2", Type: "blocker", Description: "backup step is skipped", Source: "escalation"},
	}
	for i := 0; i < 6; i++ {
		flags = append(flags, colony.FlagEntry{ID: fmt.Sprintf("i%d", i), Type: "issue", Description: fmt.Sprintf("issue number %d", i)})
	}
	flags = append(flags, colony.FlagEntry{ID: "n1", Type: "note", Description: "revisit the caching layer later", Acknowledged: true})
	return flags
}

func seedStatusFailures(t *testing.T, s *storage.Store, count int) {
	t.Helper()
	var mf colony.MiddenFile
	for i := 0; i < count; i++ {
		mf.Entries = append(mf.Entries, colony.MiddenEntry{
			ID: fmt.Sprintf("m%d", i), Category: "build", Source: "builder",
			Message: fmt.Sprintf("old failure %d", i), Timestamp: "2026-09-01T10:00:00Z",
		})
	}
	if err := s.SaveJSON("midden.json", mf); err != nil {
		t.Fatalf("seed failures: %v", err)
	}
}

func TestStatusScreenHidesZeroNoise(t *testing.T) {
	t.Run("zero counts, hashes and matching versions are gone", func(t *testing.T) {
		state, s, _ := statusScreenFixture(t, nil, statusReady)
		emitLoopBreakEvent("watcher_skip", "3 consecutive failures error_sha256=9f86d081884c7d65", "auto-skipped watcher", "aether-continue")
		rendered := stripANSI(renderDashboard(state, s, buildStatusResult(state, s)))
		for _, noise := range []string{
			"Existing blocker work: 0", "Focus: 0 areas", "Avoid: 0 patterns", "Flags: 0 blockers",
			"error_sha256", "9f86d081", "Loop Safety", "L O O P", "Runtime:", "Hub (the installed copy",
			"0 unacknowledged", "Recent failures: 0", "C O L O N Y   S T A T U S",
		} {
			if strings.Contains(rendered, noise) {
				t.Errorf("status screen still shows %q:\n%s", noise, rendered)
			}
		}
		if strings.Contains(rendered, spacedTitle("Needs You")) {
			t.Errorf("NEEDS YOU appears with nothing needing the owner:\n%s", rendered)
		}
	})

	t.Run("a non-zero failure count is still shown in NEEDS YOU", func(t *testing.T) {
		state, s, _ := statusScreenFixture(t, nil, statusReady)
		seedStatusFailures(t, s, 20)
		rendered := stripANSI(renderDashboard(state, s, buildStatusResult(state, s)))
		start := strings.Index(rendered, spacedTitle("Needs You"))
		if start < 0 {
			t.Fatalf("20 waiting failures but no NEEDS YOU section:\n%s", rendered)
		}
		section := checkScreenBody(rendered[start:])
		if !strings.Contains(section, "20 unacknowledged failure(s)") || !strings.Contains(section, "aether midden-review") {
			t.Errorf("NEEDS YOU hides the 20 waiting failures:\n%s", section)
		}
	})

	t.Run("a version disagreement is still shown in NEEDS YOU", func(t *testing.T) {
		state, s, _ := statusScreenFixture(t, nil, statusReady)
		hubDir := os.Getenv("AETHER_HUB_DIR")
		if err := os.WriteFile(filepath.Join(hubDir, "version.json"), []byte(`{"version": "99.99.99"}`), 0o644); err != nil {
			t.Fatalf("hub version: %v", err)
		}
		rendered := stripANSI(renderDashboard(state, s, buildStatusResult(state, s)))
		start := strings.Index(rendered, spacedTitle("Needs You"))
		if start < 0 || !strings.Contains(rendered[start:], "MISMATCH") {
			t.Errorf("a runtime/hub disagreement is hidden:\n%s", rendered)
		}
	})
}

func TestStatusScreenStaysShort(t *testing.T) {
	state, s, _ := statusScreenFixture(t, statusBusyFlags(), statusReady)
	seedStatusFailures(t, s, 20)
	emitLoopBreakEvent("watcher_skip", "3 consecutive failures error_sha256=9f86d081884c7d65", "auto-skipped watcher", "aether-continue")
	rendered := stripANSI(renderDashboard(state, s, buildStatusResult(state, s)))
	body := strings.TrimRight(checkScreenBody(rendered), "\n")
	lines := strings.Split(body, "\n")
	if len(lines) > 35 {
		t.Errorf("status screen body is %d lines, want at most 35:\n%s", len(lines), body)
	}
	assertSectionsSeparated(t, body, []string{"Progress", "Needs You", "Steering", "Memory"})
	for _, want := range []string{
		"BLOCKED — 2 problems need you",
		"[Phase 1/3]", "[Tasks 2/4]",
		"the release gate is red", "backup step is skipped",
		"… 4 more",
		"20 unacknowledged failure(s)",
		"Existing blocker work: 2 active (1 escalated)",
		"2 learned", "1 strong",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("busy status screen is missing %q:\n%s", want, body)
		}
	}
	// NEEDS YOU names at most five flags by title before folding the rest.
	for _, hidden := range []string{"issue number 3", "issue number 5", "revisit the caching layer later"} {
		if strings.Contains(body, hidden) {
			t.Errorf("NEEDS YOU lists more than five flags (%q):\n%s", hidden, body)
		}
	}
	for _, banned := range []string{"Proof", "Memory Health", "Colony Mode", "Granularity:", "Parallel:", "Scope:", "Depth:", "error_sha256", "Colony health"} {
		if strings.Contains(body, banned) {
			t.Errorf("busy status screen still shows %q:\n%s", banned, body)
		}
	}
}

// TestStatusDetailKeepsTheFullDump: everything that left the default screen
// is still one flag away, not deleted.
func TestStatusDetailKeepsTheFullDump(t *testing.T) {
	state, s, result := statusScreenFixture(t, statusBusyFlags(), statusReady)
	detail := stripANSI(renderStatusDetail(state, s, result))
	for _, want := range []string{"Memory Health", "Proof", "Scope:", "Parallel:", "Existing blocker work: 2 active (1 escalated)", "Instincts:"} {
		if !strings.Contains(detail, want) {
			t.Errorf("status --detail lost %q:\n%s", want, detail)
		}
	}
}

// TestStatusAndPlanScreensLeaveTheResultUntouched: the layout change is
// presentation only. Drawing either screen must leave the result map -- the
// same map the JSON output is written from -- byte-identical.
func TestStatusAndPlanScreensLeaveTheResultUntouched(t *testing.T) {
	t.Run("status", func(t *testing.T) {
		state, s, result := statusScreenFixture(t, statusBusyFlags(), statusReady)
		before, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal before: %v", err)
		}
		renderDashboard(state, s, result)
		renderStatusDetail(state, s, result)
		after, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal after: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("drawing the status screen changed the status result:\nbefore: %s\nafter:  %s", before, after)
		}
	})
	t.Run("plan", func(t *testing.T) {
		for name, result := range map[string]map[string]interface{}{
			"candidate": planCandidateReviewResult(screenPlanCandidateFixture()),
			"accepted":  planScreenAcceptedResult(),
			"legacy":    planScreenLegacyResult(),
		} {
			before, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("%s marshal before: %v", name, err)
			}
			renderPlanVisual(result)
			after, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("%s marshal after: %v", name, err)
			}
			if string(before) != string(after) {
				t.Errorf("drawing the %s plan screen changed the plan result:\nbefore: %s\nafter:  %s", name, before, after)
			}
		}
	})
}

func TestStatusScreenForcedClosureCannotClaimFinished(t *testing.T) {
	state, _, result := statusScreenFixture(t, nil, func(state *colony.ColonyState) {
		state.State = colony.StateCOMPLETED
		for i := range state.Plan.Phases {
			state.Plan.Phases[i].Status = colony.PhaseCompleted
		}
	})
	state.SealOutcome = &colony.SealOutcome{Disposition: colony.SealDispositionForcedIncomplete, OwnerReason: "Keeping the unfinished prototype"}
	_, verdict, sentence := statusScreenVerdict(state, result)
	if strings.Contains(verdict, "PROJECT FINISHED") || !strings.Contains(verdict, "CLOSED EARLY") || !strings.Contains(sentence, state.SealOutcome.OwnerReason) {
		t.Fatalf("forced closure lost its meaning: %s %s", verdict, sentence)
	}
}
