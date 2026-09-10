package cmd

import (
	"fmt"
	"strings"
	"time"
)

// Phase 201 plan 13 (WORK-08, D-13) -- "fast enough" as a number the
// program can check, rather than a feeling. Plan 201-12 built the
// eight-segment timing record (cmd/job_telemetry.go); this file stores the
// ONE ratified figure a real run is checked against and the honest,
// three-state comparison that never guesses when a run's evidence is
// incomplete.
//
// The target's value below is not invented here. It is copied, verbatim,
// from the owner-ratified figure in
// .planning/phases/201-queen-led-work-cycle/201-TURNAROUND-BASELINE.md:
// 4 minutes 30 seconds for a small, single-task job, primarily attacking
// the "work" segment (worker dispatch/execution) -- the one segment that
// baseline's own measured run actually captured. The owner was explicit
// that this is a floor-setting benchmark for SMALL jobs, not a claim every
// job should finish in 4m30s; nothing in this file enforces that scope --
// it stores and compares a duration, and leaves the "was this a small job"
// judgment to whatever later plan or phase reads the comparison.
//
// This file follows the same honesty discipline jobTelemetryRecord and
// verificationScope already hold: a comparison that cannot be made
// honestly reports so explicitly (turnaroundComparisonUnmeasurable) rather
// than defaulting to a guessed pass or fail. There is no code path here
// that resolves an unmeasured segment to met or missed.

// turnaroundTarget is the ratified bar a real job's telemetry is compared
// against: an exact total duration, and the single named segment (from
// jobTelemetrySegmentOrder's eight-segment vocabulary) that duration is
// measured against.
type turnaroundTarget struct {
	// TotalDuration is the ratified target duration -- the bar Segment's
	// measured value is compared to.
	TotalDuration time.Duration
	// Segment is the one telemetry segment this target is compared
	// against, e.g. jobTelemetrySegmentWork. Always one of the eight names
	// jobTelemetrySegmentOrder lists -- newTurnaroundTarget refuses any
	// other value.
	Segment string
}

// newTurnaroundTarget builds a turnaroundTarget, refusing a segment name
// outside the eight-segment vocabulary jobTelemetryRecord uses (naming the
// offending value) and a non-positive duration (a target of zero or less
// is not a bar anything could ever miss honestly).
func newTurnaroundTarget(total time.Duration, segment string) (turnaroundTarget, error) {
	segment = strings.TrimSpace(segment)
	if _, ok := jobTelemetrySegmentLabels[segment]; !ok {
		return turnaroundTarget{}, fmt.Errorf("turnaround target: refused -- %q is not one of the eight known telemetry segments", segment)
	}
	if total <= 0 {
		return turnaroundTarget{}, fmt.Errorf("turnaround target: refused -- total duration must be positive, got %v", total)
	}
	return turnaroundTarget{TotalDuration: total, Segment: segment}, nil
}

// mustTurnaroundTarget is newTurnaroundTarget for the one package-level
// value below, whose inputs are both compile-time constants copied from
// the ratified baseline document -- a construction failure here would be a
// programming error in this file, not a runtime condition any caller needs
// to handle.
func mustTurnaroundTarget(total time.Duration, segment string) turnaroundTarget {
	target, err := newTurnaroundTarget(total, segment)
	if err != nil {
		panic(err)
	}
	return target
}

// ratifiedTurnaroundTarget is the owner-ratified turnaround target
// (ratified 2026-09-10, .planning/phases/201-queen-led-work-cycle/
// 201-TURNAROUND-BASELINE.md): 4 minutes 30 seconds for a small,
// single-task job, primarily attacking the work segment (worker
// dispatch/execution).
var ratifiedTurnaroundTarget = mustTurnaroundTarget(4*time.Minute+30*time.Second, jobTelemetrySegmentWork)

// The three states a comparison can resolve to. There is no fourth state,
// and unmeasurable is never collapsed into either of the other two.
const (
	turnaroundComparisonMet          = "met"
	turnaroundComparisonMissed       = "missed"
	turnaroundComparisonUnmeasurable = "unmeasurable"
)

// turnaroundComparison is the outcome of comparing one jobTelemetryRecord
// against a turnaroundTarget.
type turnaroundComparison struct {
	// Status is one of turnaroundComparisonMet, turnaroundComparisonMissed,
	// or turnaroundComparisonUnmeasurable.
	Status string
	// Shortfall is populated only when Status is turnaroundComparisonMissed
	// -- the exact amount by which the record's segment exceeded the
	// target. Zero in every other state, including missed-by-nothing,
	// which cannot occur (an exact match reports met, not missed).
	Shortfall time.Duration
	// Reason is populated only when Status is
	// turnaroundComparisonUnmeasurable -- the plain-English explanation for
	// why no honest met/missed verdict could be reached, carried through
	// from the record's own segment.Source when available.
	Reason string
}

// compareTurnaroundTarget compares record's measurement of target.Segment
// against target.TotalDuration. When that segment was not measured for
// record, the comparison reports unmeasurable and carries the segment's
// own stated reason -- it NEVER falls back to met or missed on incomplete
// evidence, the same honesty rule jobTelemetryRecord's own unmeasured
// segments already hold.
//
// This is a pure read: no store access, no write, no dispatch. Every
// caller may call it as often as it likes against the same inputs and get
// byte-identical results.
func compareTurnaroundTarget(target turnaroundTarget, record jobTelemetryRecord) turnaroundComparison {
	for _, named := range record.namedSegments() {
		if named.Name != target.Segment {
			continue
		}
		if !named.Segment.Measured {
			reason := strings.TrimSpace(named.Segment.Source)
			if reason == "" {
				reason = fmt.Sprintf("the %s segment was not measured for this record", jobTelemetrySegmentLabel(target.Segment))
			}
			return turnaroundComparison{Status: turnaroundComparisonUnmeasurable, Reason: reason}
		}
		if named.Segment.Duration <= target.TotalDuration {
			return turnaroundComparison{Status: turnaroundComparisonMet}
		}
		return turnaroundComparison{Status: turnaroundComparisonMissed, Shortfall: named.Segment.Duration - target.TotalDuration}
	}
	// Unreachable in production: newTurnaroundTarget refuses any segment
	// name outside jobTelemetrySegmentOrder, and namedSegments() always
	// returns all eight. Reported as unmeasurable rather than panicking or
	// fabricating a verdict, matching this file's own no-guessing rule.
	return turnaroundComparison{
		Status: turnaroundComparisonUnmeasurable,
		Reason: fmt.Sprintf("target segment %q was not found among the record's segments", target.Segment),
	}
}
