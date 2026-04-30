package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

// --- Task 2: RED tests for Loop Safety section in /ant-status ---

func TestRenderLoopSafetySectionWithEvents(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Publish 3 loop-break events
	emitLoopBreakEvent("watcher_skip", "3 consecutive failures", "auto-skipped watcher", "aether-continue")
	emitLoopBreakEvent("circuit_break", "5 failures threshold 3", "tripped Builder-42", "aether-build")
	emitLoopBreakEvent("cycle_detected", "phase 1 -> phase 2 -> phase 1", "plan rejected", "aether-plan")

	// Load events via the function we're testing
	evts := loadRecentLoopBreakEvents(s)
	if len(evts) != 3 {
		t.Fatalf("loadRecentLoopBreakEvents returned %d events, want 3", len(evts))
	}

	output := renderLoopSafetySection(evts)

	if output == "" {
		t.Fatal("renderLoopSafetySection returned empty string, want non-empty")
	}
	if !strings.Contains(output, "Loop Safety") {
		t.Error("output missing 'Loop Safety' banner")
	}
	if !strings.Contains(output, "watcher_skip") {
		t.Error("output missing 'watcher_skip' loop type")
	}
	if !strings.Contains(output, "circuit_break") {
		t.Error("output missing 'circuit_break' loop type")
	}
	if !strings.Contains(output, "cycle_detected") {
		t.Error("output missing 'cycle_detected' loop type")
	}
}

func TestRenderLoopSafetySectionEmpty(t *testing.T) {
	output := renderLoopSafetySection(nil)
	if output != "" {
		t.Errorf("renderLoopSafetySection(nil) = %q, want empty string", output)
	}

	output = renderLoopSafetySection([]events.Event{})
	if output != "" {
		t.Errorf("renderLoopSafetySection([]) = %q, want empty string", output)
	}
}

func TestLoadRecentLoopBreakEventsQuery(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// Publish 7 events -- should return at most 5
	for i := 0; i < 7; i++ {
		emitLoopBreakEvent("watcher_skip",
			time.Now().Add(time.Duration(i)*time.Minute).String(),
			"auto-skipped",
			"test")
		// Small sleep to ensure distinct timestamps
		time.Sleep(time.Millisecond * 10)
	}

	evts := loadRecentLoopBreakEvents(s)
	if len(evts) > 5 {
		t.Errorf("loadRecentLoopBreakEvents returned %d events, want at most 5", len(evts))
	}
	if len(evts) == 0 {
		t.Fatal("loadRecentLoopBreakEvents returned 0 events, want some")
	}

	// Verify newest-first order (first event should have a later timestamp than last)
	if len(evts) >= 2 {
		if evts[0].Timestamp < evts[len(evts)-1].Timestamp {
			t.Errorf("events not in newest-first order: first=%s, last=%s", evts[0].Timestamp, evts[len(evts)-1].Timestamp)
		}
	}
}
