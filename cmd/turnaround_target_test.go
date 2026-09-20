package cmd

import (
	"strings"
	"testing"
	"time"
)

// Phase 201 plan 13 (WORK-08, D-13) -- the ratified turnaround target and
// its three-state comparison against a real jobTelemetryRecord. The target
// figure here (4m30s, targeting the work segment) is taken verbatim from
// the owner-ratified number in
// .planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md.

// buildTurnaroundFixtureRecord builds a jobTelemetryRecord the same way
// production instrumentation does -- through jobTelemetryCapture and
// newJobTelemetryRecord -- rather than constructing the struct literal
// directly, so these tests exercise the same honesty discipline the real
// record does (CLAUDE.md's "derive fixture values the way the runtime
// derives them").
func buildTurnaroundFixtureRecord(t *testing.T, measureWork *time.Duration) jobTelemetryRecord {
	t.Helper()
	capture := newJobTelemetryCapture()
	if measureWork != nil {
		capture.measure(jobTelemetrySegmentWork, *measureWork, "worker execution")
	} else {
		capture.markUnmeasured(jobTelemetrySegmentWork, "the build never reached telemetry's write point")
	}
	return newJobTelemetryRecord("attempt-turnaround-fixture", "turnaround-fixture-job", capture, time.Now())
}

// ---------------------------------------------------------------------
// The stored target matches the ratified baseline figure exactly.
// ---------------------------------------------------------------------

func TestRatifiedTurnaroundTargetMatchesBaseline(t *testing.T) {
	want := 4*time.Minute + 30*time.Second
	if ratifiedTurnaroundTarget.TotalDuration != want {
		t.Fatalf("ratifiedTurnaroundTarget.TotalDuration = %v, want %v (4m30s, the owner-ratified figure)", ratifiedTurnaroundTarget.TotalDuration, want)
	}
	if ratifiedTurnaroundTarget.Segment != jobTelemetrySegmentWork {
		t.Fatalf("ratifiedTurnaroundTarget.Segment = %q, want %q (the segment the baseline names as primarily targeted)", ratifiedTurnaroundTarget.Segment, jobTelemetrySegmentWork)
	}
}

// ---------------------------------------------------------------------
// Met / missed, with an exact shortfall.
// ---------------------------------------------------------------------

func TestTurnaroundTargetComparison(t *testing.T) {
	target, err := newTurnaroundTarget(4*time.Minute+30*time.Second, jobTelemetrySegmentWork)
	if err != nil {
		t.Fatalf("newTurnaroundTarget: %v", err)
	}

	t.Run("a record beating the target reports met", func(t *testing.T) {
		fast := 4 * time.Minute
		record := buildTurnaroundFixtureRecord(t, &fast)
		got := compareTurnaroundTarget(target, record)
		if got.Status != turnaroundComparisonMet {
			t.Fatalf("Status = %q, want %q", got.Status, turnaroundComparisonMet)
		}
		if got.Shortfall != 0 {
			t.Fatalf("Shortfall on a met comparison = %v, want 0", got.Shortfall)
		}
	})

	t.Run("a record exactly at the target reports met, not missed", func(t *testing.T) {
		exact := 4*time.Minute + 30*time.Second
		record := buildTurnaroundFixtureRecord(t, &exact)
		got := compareTurnaroundTarget(target, record)
		if got.Status != turnaroundComparisonMet {
			t.Fatalf("Status for an exact match = %q, want %q", got.Status, turnaroundComparisonMet)
		}
	})

	t.Run("a record missing the target reports missed with the exact shortfall", func(t *testing.T) {
		// 408s (the measured baseline run) against a 270s target: shortfall
		// is exactly 138s.
		slow := 408 * time.Second
		record := buildTurnaroundFixtureRecord(t, &slow)
		got := compareTurnaroundTarget(target, record)
		if got.Status != turnaroundComparisonMissed {
			t.Fatalf("Status = %q, want %q", got.Status, turnaroundComparisonMissed)
		}
		wantShortfall := 138 * time.Second
		if got.Shortfall != wantShortfall {
			t.Fatalf("Shortfall = %v, want exactly %v", got.Shortfall, wantShortfall)
		}
	})
}

// ---------------------------------------------------------------------
// Unmeasurable never resolves to met or missed.
// ---------------------------------------------------------------------

func TestUnmeasurableNeverReportsMetOrMissed(t *testing.T) {
	target, err := newTurnaroundTarget(4*time.Minute+30*time.Second, jobTelemetrySegmentWork)
	if err != nil {
		t.Fatalf("newTurnaroundTarget: %v", err)
	}

	record := buildTurnaroundFixtureRecord(t, nil) // work segment left unmeasured
	got := compareTurnaroundTarget(target, record)

	if got.Status != turnaroundComparisonUnmeasurable {
		t.Fatalf("Status for an unmeasured relevant segment = %q, want %q", got.Status, turnaroundComparisonUnmeasurable)
	}
	if got.Status == turnaroundComparisonMet || got.Status == turnaroundComparisonMissed {
		t.Fatalf("an unmeasurable comparison must never resolve to met or missed, got %q", got.Status)
	}
	if strings.TrimSpace(got.Reason) == "" {
		t.Fatal("an unmeasurable comparison must carry a stated reason")
	}
	if got.Shortfall != 0 {
		t.Fatalf("Shortfall on an unmeasurable comparison = %v, want 0 (no shortfall claim without a measurement)", got.Shortfall)
	}
}

// ---------------------------------------------------------------------
// An out-of-vocabulary segment name is refused by name.
// ---------------------------------------------------------------------

func TestTurnaroundTargetRefusesOutOfVocabularySegment(t *testing.T) {
	_, err := newTurnaroundTarget(5*time.Minute, "networking")
	if err == nil {
		t.Fatal("expected a segment name outside the eight-segment vocabulary to be refused")
	}
	if !strings.Contains(err.Error(), "networking") {
		t.Fatalf("error %q does not name the offending value %q", err.Error(), "networking")
	}
}

// ---------------------------------------------------------------------
// Reading and comparing writes nothing.
// ---------------------------------------------------------------------

func TestTurnaroundTargetReadWritesNothing(t *testing.T) {
	setupSpendTestStore(t)
	fast := 4 * time.Minute
	record := buildTurnaroundFixtureRecord(t, &fast)

	digestBefore := hashDirForTest(t, store.BasePath())

	target, err := newTurnaroundTarget(4*time.Minute+30*time.Second, jobTelemetrySegmentWork)
	if err != nil {
		t.Fatalf("newTurnaroundTarget: %v", err)
	}
	_ = compareTurnaroundTarget(target, record)
	_ = compareTurnaroundTarget(ratifiedTurnaroundTarget, record)

	digestAfter := hashDirForTest(t, store.BasePath())
	if digestBefore != digestAfter {
		t.Fatal("constructing a target and comparing it against a record wrote to the fixture directory")
	}
}
