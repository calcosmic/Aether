package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// Task 1: the eight-segment record with an honest unmeasured sentinel.
// ---------------------------------------------------------------------

func TestUnmeasuredSegmentIsNeverDerived(t *testing.T) {
	t.Run("an unmeasured segment reads as unmeasured, never as zero", func(t *testing.T) {
		capture := newJobTelemetryCapture()
		capture.measure(jobTelemetrySegmentContext, 2*time.Second, "brief assembly")
		capture.measure(jobTelemetrySegmentWork, 10*time.Second, "worker execution")
		capture.markUnmeasured(jobTelemetrySegmentQueue, "no queue boundary recorded in this fixture")

		record := newJobTelemetryRecord("attempt-x", "job-x", capture, time.Now())

		if record.Queue.Measured {
			t.Fatal("expected Queue to be recorded as unmeasured")
		}
		if record.Queue.Duration != 0 {
			t.Fatalf("expected Queue's untouched Duration field to be zero, got %v", record.Queue.Duration)
		}
		if got := record.Queue.RenderedDuration(); got != jobTelemetryUnmeasuredFigure {
			t.Fatalf("RenderedDuration() on an unmeasured segment = %q, want %q", got, jobTelemetryUnmeasuredFigure)
		}
		if record.Queue.Source == "" {
			t.Fatal("expected the unmeasured segment to still carry its stated reason")
		}

		// A segment the capture was never told about at all (Model,
		// ToolCall, Verification, Wait, Preflight here) must read
		// identically to one explicitly marked unmeasured -- an untouched
		// segment and a reasoned-unmeasured one both mean "no measurement",
		// and both must be indistinguishable to RenderedDuration.
		for _, name := range []string{
			jobTelemetrySegmentPreflight,
			jobTelemetrySegmentModel,
			jobTelemetrySegmentToolCall,
			jobTelemetrySegmentVerification,
			jobTelemetrySegmentWait,
		} {
			for _, named := range record.namedSegments() {
				if named.Name != name {
					continue
				}
				if named.Segment.Measured {
					t.Fatalf("segment %q was never captured but reads as measured", name)
				}
				if got := named.Segment.RenderedDuration(); got != jobTelemetryUnmeasuredFigure {
					t.Fatalf("segment %q RenderedDuration() = %q, want %q", name, got, jobTelemetryUnmeasuredFigure)
				}
			}
		}

		// The largest measured segment must be Work (10s), never Queue --
		// proving an unmeasured segment's zero-value Duration never enters
		// a comparison against real measurements.
		name, seg, ok := record.largestMeasuredSegment()
		if !ok || name != jobTelemetrySegmentWork || seg.Duration != 10*time.Second {
			t.Fatalf("largestMeasuredSegment() = (%q, %+v, %v), want (work, 10s, true)", name, seg, ok)
		}
	})

	t.Run("no function in this file computes a segment from the total or from other segments", func(t *testing.T) {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "job_telemetry.go", nil, 0)
		if err != nil {
			t.Fatalf("parse cmd/job_telemetry.go: %v", err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			be, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			if be.Op == token.SUB {
				t.Fatalf("found a subtraction at %s -- a timing segment must never be derived as total minus the other segments", fset.Position(be.Pos()))
			}
			return true
		})
	})
}

func TestSegmentRequiresAnInstrumentationSource(t *testing.T) {
	if _, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, ""); err == nil {
		t.Fatal("expected a measured segment with no source to be refused")
	} else if !strings.Contains(err.Error(), jobTelemetrySegmentWork) {
		t.Fatalf("refusal error %q does not name the offending segment", err.Error())
	}
	if _, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, "   "); err == nil {
		t.Fatal("expected a measured segment with a whitespace-only source to be refused")
	}
	if seg, err := newMeasuredJobTelemetrySegment(jobTelemetrySegmentWork, 5*time.Second, "worker execution"); err != nil {
		t.Fatalf("unexpected refusal of a valid measured segment: %v", err)
	} else if !seg.Measured || seg.Duration != 5*time.Second || seg.Source != "worker execution" {
		t.Fatalf("unexpected segment shape: %+v", seg)
	}

	if _, err := unmeasuredJobTelemetrySegment(jobTelemetrySegmentPreflight, ""); err == nil {
		t.Fatal("expected an unmeasured segment with no reason to be refused")
	} else if !strings.Contains(err.Error(), jobTelemetrySegmentPreflight) {
		t.Fatalf("refusal error %q does not name the offending segment", err.Error())
	}
	if seg, err := unmeasuredJobTelemetrySegment(jobTelemetrySegmentPreflight, "no first-response boundary is observable today"); err != nil {
		t.Fatalf("unexpected refusal of a valid unmeasured segment: %v", err)
	} else if seg.Measured || seg.Source == "" {
		t.Fatalf("unexpected segment shape: %+v", seg)
	}

	// jobTelemetryCapture.measure/markUnmeasured must swallow a refused
	// segment rather than ever panicking or corrupting the record --
	// instrumentation is never allowed to fail a build (this plan's own
	// must_have).
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, 5*time.Second, "")
	capture.markUnmeasured(jobTelemetrySegmentWait, "")
	record := newJobTelemetryRecord("attempt-refused", "job-refused", capture, time.Now())
	if record.Work.Measured {
		t.Fatal("a refused measured segment must not silently become measured")
	}
	if record.Wait.Measured {
		t.Fatal("a refused unmeasured-mark must not silently become measured")
	}
}

func TestAllUnmeasuredRecordWritesNoFile(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	capture := newJobTelemetryCapture()
	capture.markUnmeasured(jobTelemetrySegmentQueue, "no boundary recorded")
	capture.markUnmeasured(jobTelemetrySegmentContext, "no boundary recorded")
	record := newJobTelemetryRecord("attempt-all-unmeasured", "job-all-unmeasured", capture, time.Now())

	if record.hasAnyMeasuredSegment() {
		t.Fatal("test fixture is broken: expected an all-unmeasured record")
	}
	if err := writeJobTelemetryRecord(record); err != nil {
		t.Fatalf("writing an all-unmeasured record must not itself error, got: %v", err)
	}

	if _, ok := readJobTelemetryRecord("attempt-all-unmeasured"); ok {
		t.Fatal("an all-unmeasured record was written to disk, want no file at all")
	}
}

func TestTelemetryStorageFailureIsNonFatal(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, 3*time.Second, "worker execution")

	// A path-traversal attempt id is refused by attemptBoundArtifactPath's
	// own containment validation -- writeJobTelemetryRecord must surface
	// that as a returned error, never a panic, and never corrupt anything
	// on disk.
	record := newJobTelemetryRecord("../escape", "job-escape", capture, time.Now())
	if err := writeJobTelemetryRecord(record); err == nil {
		t.Fatal("expected a storage failure for a path-traversal attempt id")
	}

	// The caller's own result is unaffected: reading back the (never
	// written) record still reports nothing, and a second, legitimate
	// write for a different attempt still succeeds -- proving the failed
	// write left no dangling state behind that could affect anyone else.
	if _, ok := readJobTelemetryRecord("../escape"); ok {
		t.Fatal("a failed write must not be readable afterward")
	}
	goodRecord := newJobTelemetryRecord("attempt-after-failure", "job-after-failure", capture, time.Now())
	if err := writeJobTelemetryRecord(goodRecord); err != nil {
		t.Fatalf("a legitimate write after an induced storage failure must still succeed: %v", err)
	}
	if _, ok := readJobTelemetryRecord("attempt-after-failure"); !ok {
		t.Fatal("expected the legitimate write to be readable")
	}
}

// TestJobTelemetryJSONRoundTrip is the plan's declared behavior "A record
// round-trips through JSON preserving measured and unmeasured segments
// distinctly", verified directly rather than folded into another test.
func TestJobTelemetryJSONRoundTrip(t *testing.T) {
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentContext, 1500*time.Millisecond, "brief assembly")
	capture.measure(jobTelemetrySegmentVerification, 42*time.Second, "deterministic floor run")
	capture.markUnmeasured(jobTelemetrySegmentModel, "the platform reports total tokens per worker but no call duration")
	record := newJobTelemetryRecord("attempt-roundtrip", "job-roundtrip", capture, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC))

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded jobTelemetryRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(record, decoded) {
		t.Fatalf("round trip mismatch:\n  before: %+v\n  after:  %+v", record, decoded)
	}
	if decoded.Context.Measured != true || decoded.Context.Duration != 1500*time.Millisecond {
		t.Fatalf("measured segment lost precision across the round trip: %+v", decoded.Context)
	}
	if decoded.Model.Measured != false || decoded.Model.Source == "" {
		t.Fatalf("unmeasured segment lost its reason across the round trip: %+v", decoded.Model)
	}
}

// TestJobTelemetryKeyedByAttemptAndJob is the plan's declared behavior "The
// record is keyed by the attempt identifier and the job name."
func TestJobTelemetryKeyedByAttemptAndJob(t *testing.T) {
	capture := newJobTelemetryCapture()
	capture.measure(jobTelemetrySegmentWork, time.Second, "worker execution")
	record := newJobTelemetryRecord("  attempt-keyed  ", "  job-keyed  ", capture, time.Now())
	if record.AttemptID != "attempt-keyed" || record.JobName != "job-keyed" {
		t.Fatalf("expected trimmed attempt/job identity, got AttemptID=%q JobName=%q", record.AttemptID, record.JobName)
	}
}
