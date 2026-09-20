package cmd

import (
	"fmt"
	"strings"
	"time"
)

// Phase 201 plan 12 (WORK-08, CEC-06, D-13/D-14/D-16) -- the eight-segment,
// attempt-bound timing record. Before this file, the only timing this
// codebase ever recorded was one whole-attempt elapsed figure
// (spendElapsedFigure, cmd/spend_cost_line.go, plan 201-06): honest, but
// silent about where inside that span the time actually went. This file
// answers "where did the thirty minutes go" one named segment at a time,
// without ever inventing a number for a segment nobody measured.
//
// The discipline is modeled directly on verificationScope
// (cmd/verification_scope.go): every fact this record carries is EITHER a
// real measurement naming the instrumentation point that captured it, OR an
// explicit, reasoned admission that nothing was captured. There is no third
// state, and there is no code path anywhere in this file that computes one
// segment from the total or from the other seven -- that is exactly the
// failure this plan's own objective names as the most likely way to get
// this wrong (the same failure that once shipped a token undercount behind
// a green test).

// jobTelemetrySchemaVersion is bumped only if this record's on-disk shape
// changes in a way an old reader could misinterpret.
const jobTelemetrySchemaVersion = 1

// The eight named segments (WORK-08, D-13). This exact set -- no more, no
// fewer -- is what jobTelemetryRecord carries.
const (
	jobTelemetrySegmentQueue        = "queue"
	jobTelemetrySegmentPreflight    = "preflight"
	jobTelemetrySegmentModel        = "model"
	jobTelemetrySegmentToolCall     = "tool_call"
	jobTelemetrySegmentContext      = "context"
	jobTelemetrySegmentWork         = "work"
	jobTelemetrySegmentVerification = "verification"
	jobTelemetrySegmentWait         = "wait"
)

// jobTelemetrySegmentOrder is the one fixed, deterministic order every
// iteration and every rendering of the eight segments walks them in --
// never Go's randomized map iteration order.
var jobTelemetrySegmentOrder = []string{
	jobTelemetrySegmentQueue,
	jobTelemetrySegmentPreflight,
	jobTelemetrySegmentModel,
	jobTelemetrySegmentToolCall,
	jobTelemetrySegmentContext,
	jobTelemetrySegmentWork,
	jobTelemetrySegmentVerification,
	jobTelemetrySegmentWait,
}

// jobTelemetrySegmentLabels are the plain-English names a reader who has
// never opened this repository can understand (CLAUDE.md's communication
// rule) -- "tool_call" is a JSON key, not a word to show anyone.
var jobTelemetrySegmentLabels = map[string]string{
	jobTelemetrySegmentQueue:        "queue",
	jobTelemetrySegmentPreflight:    "preflight",
	jobTelemetrySegmentModel:        "model calls",
	jobTelemetrySegmentToolCall:     "tool calls",
	jobTelemetrySegmentContext:      "context assembly",
	jobTelemetrySegmentWork:         "worker execution",
	jobTelemetrySegmentVerification: "verification",
	jobTelemetrySegmentWait:         "waiting",
}

// jobTelemetrySegmentLabel returns name's plain-English label, or name
// itself when it is not one of the eight known segments (defensive only --
// every production caller uses the named constants).
func jobTelemetrySegmentLabel(name string) string {
	if label, ok := jobTelemetrySegmentLabels[name]; ok {
		return label
	}
	return name
}

// jobTelemetryUnmeasuredFigure is what a reader of ANY unmeasured segment
// sees -- never zero, which would read as "this took no time" rather than
// "this was never captured". Mirrors spendNotReportedFigure's discipline
// (cmd/spend_cost_line.go) one layer down, at the segment rather than the
// worker.
const jobTelemetryUnmeasuredFigure = "unmeasured"

// jobTelemetrySegment is one segment's recorded fact: either a measured
// duration captured at a named instrumentation point, or an explicit
// admission that nothing was captured and why. There is no third
// possibility and no default.
type jobTelemetrySegment struct {
	// Measured is true only when Duration was captured at a real
	// instrumentation point. Callers must check this before reading
	// Duration -- see RenderedDuration, the one safe accessor.
	Measured bool `json:"measured"`
	// Duration is populated only when Measured is true. Reading it directly
	// on an unmeasured segment would silently return Go's zero value, which
	// is exactly the ambiguity RenderedDuration exists to close off.
	Duration time.Duration `json:"duration_ns,omitempty"`
	// Source is REQUIRED in both states, and plays a different role in
	// each: when Measured is true it names the instrumentation point that
	// captured Duration; when Measured is false it is the plain-English
	// reason nothing could be captured. One field carries both roles
	// because both answer the same question a reader asks of this segment:
	// "why does it read what it reads?"
	Source string `json:"source,omitempty"`
}

// RenderedDuration is the one safe way to read a segment's duration for
// display: the unmeasured sentinel when Measured is false, and the rounded
// duration string otherwise. No caller should ever read .Duration directly
// without first checking .Measured.
func (s jobTelemetrySegment) RenderedDuration() string {
	if !s.Measured {
		return jobTelemetryUnmeasuredFigure
	}
	return s.Duration.Round(time.Millisecond).String()
}

// newMeasuredJobTelemetrySegment builds a measured segment for the named
// segment, refusing one offered with no instrumentation point named. A
// negative duration (a clock anomaly, never a legitimate measurement) is
// clamped to zero rather than trusted -- zero is still a real measurement
// ("this was instantaneous"), which is why it stays Measured: true, unlike
// an unmeasured segment's untouched zero value.
func newMeasuredJobTelemetrySegment(name string, duration time.Duration, source string) (jobTelemetrySegment, error) {
	if strings.TrimSpace(source) == "" {
		return jobTelemetrySegment{}, fmt.Errorf("job telemetry segment %q: refused -- no instrumentation point named", name)
	}
	if duration < 0 {
		duration = 0
	}
	return jobTelemetrySegment{Measured: true, Duration: duration, Source: strings.TrimSpace(source)}, nil
}

// unmeasuredJobTelemetrySegment builds a segment explicitly recorded as not
// measured, carrying the stated reason. An empty reason is refused for the
// identical honesty discipline a measured segment's missing source is
// refused for -- a silent, reasonless unmeasured segment is exactly the
// ambiguity this record exists to close.
func unmeasuredJobTelemetrySegment(name, reason string) (jobTelemetrySegment, error) {
	if strings.TrimSpace(reason) == "" {
		return jobTelemetrySegment{}, fmt.Errorf("job telemetry segment %q: refused -- no reason named for leaving it unmeasured", name)
	}
	return jobTelemetrySegment{Measured: false, Source: strings.TrimSpace(reason)}, nil
}

// jobTelemetryRecord is one job's eight-segment timing record for one
// attempt (WORK-08). It is keyed by the same durable attempt identifier the
// evidence and the cost figures already use (CEC-06), plus the job name the
// segments belong to.
type jobTelemetryRecord struct {
	SchemaVersion int    `json:"schema_version"`
	AttemptID     string `json:"attempt_id"`
	JobName       string `json:"job_name,omitempty"`
	RecordedAt    string `json:"recorded_at"`

	Queue        jobTelemetrySegment `json:"queue"`
	Preflight    jobTelemetrySegment `json:"preflight"`
	Model        jobTelemetrySegment `json:"model"`
	ToolCall     jobTelemetrySegment `json:"tool_call"`
	Context      jobTelemetrySegment `json:"context"`
	Work         jobTelemetrySegment `json:"work"`
	Verification jobTelemetrySegment `json:"verification"`
	Wait         jobTelemetrySegment `json:"wait"`
}

// jobTelemetryNamedSegment pairs one segment with its canonical name -- the
// one shape every iteration over the record's eight segments returns,
// walked in jobTelemetrySegmentOrder rather than struct field order (which
// Go does not guarantee to match declaration order under reflection either,
// and which this function does not use at all).
type jobTelemetryNamedSegment struct {
	Name    string
	Segment jobTelemetrySegment
}

// namedSegments returns the record's eight segments as an ordered, named
// slice -- the one path rendering and analysis code should walk instead of
// touching the eight fields individually.
func (r jobTelemetryRecord) namedSegments() []jobTelemetryNamedSegment {
	byName := map[string]jobTelemetrySegment{
		jobTelemetrySegmentQueue:        r.Queue,
		jobTelemetrySegmentPreflight:    r.Preflight,
		jobTelemetrySegmentModel:        r.Model,
		jobTelemetrySegmentToolCall:     r.ToolCall,
		jobTelemetrySegmentContext:      r.Context,
		jobTelemetrySegmentWork:         r.Work,
		jobTelemetrySegmentVerification: r.Verification,
		jobTelemetrySegmentWait:         r.Wait,
	}
	named := make([]jobTelemetryNamedSegment, 0, len(jobTelemetrySegmentOrder))
	for _, name := range jobTelemetrySegmentOrder {
		named = append(named, jobTelemetryNamedSegment{Name: name, Segment: byName[name]})
	}
	return named
}

// hasAnyMeasuredSegment reports whether at least one of the eight segments
// was genuinely measured. A record where this is false is never written --
// see writeJobTelemetryRecord.
func (r jobTelemetryRecord) hasAnyMeasuredSegment() bool {
	for _, named := range r.namedSegments() {
		if named.Segment.Measured {
			return true
		}
	}
	return false
}

// largestMeasuredSegment returns the name and value of the longest measured
// duration among the eight, and ok=false when nothing was measured at all.
// Ties resolve to whichever comes first in jobTelemetrySegmentOrder. This
// never touches an unmeasured segment's zero-value Duration as though it
// were a real (and always-losing) zero-length measurement -- unmeasured
// segments are skipped outright, not compared.
func (r jobTelemetryRecord) largestMeasuredSegment() (name string, segment jobTelemetrySegment, ok bool) {
	for _, named := range r.namedSegments() {
		if !named.Segment.Measured {
			continue
		}
		if !ok || named.Segment.Duration > segment.Duration {
			name = named.Name
			segment = named.Segment
			ok = true
		}
	}
	return name, segment, ok
}

// jobTelemetryCapture accumulates a job's segments as each real
// instrumentation point fires during a build or check pass, in whatever
// order they happen to complete. Its zero value is ready to use. A segment
// offered with no source (measured) or no reason (unmeasured) is refused
// and simply not recorded -- callers that need to surface the refusal
// should call newMeasuredJobTelemetrySegment / unmeasuredJobTelemetrySegment
// directly instead.
type jobTelemetryCapture struct {
	segments map[string]jobTelemetrySegment
}

// newJobTelemetryCapture returns a ready-to-use capture.
func newJobTelemetryCapture() *jobTelemetryCapture {
	return &jobTelemetryCapture{segments: map[string]jobTelemetrySegment{}}
}

// measure records name as measured for duration, captured at source. A
// nil receiver or a refused segment (empty source) is a silent no-op --
// instrumentation must never be able to fail a build (this plan's own
// must_have).
func (c *jobTelemetryCapture) measure(name string, duration time.Duration, source string) {
	if c == nil {
		return
	}
	segment, err := newMeasuredJobTelemetrySegment(name, duration, source)
	if err != nil {
		return
	}
	if c.segments == nil {
		c.segments = map[string]jobTelemetrySegment{}
	}
	c.segments[name] = segment
}

// markUnmeasured records name as explicitly, honestly unmeasured, carrying
// reason. A nil receiver or a refused segment (empty reason) is a silent
// no-op for the same non-fatal reason measure's is.
func (c *jobTelemetryCapture) markUnmeasured(name, reason string) {
	if c == nil {
		return
	}
	segment, err := unmeasuredJobTelemetrySegment(name, reason)
	if err != nil {
		return
	}
	if c.segments == nil {
		c.segments = map[string]jobTelemetrySegment{}
	}
	c.segments[name] = segment
}

// newJobTelemetryRecord assembles the final record from whatever a capture
// accumulated. A segment the capture never touched at all (neither
// measured nor explicitly marked unmeasured) renders as an untouched zero
// value -- Measured: false, Source: "" -- which is indistinguishable, by
// design, from any other unmeasured segment to every reader in this file;
// it is production instrumentation's job (Task 2) to make sure every
// segment it can reason about at all is touched one way or the other.
func newJobTelemetryRecord(attemptID, jobName string, capture *jobTelemetryCapture, now time.Time) jobTelemetryRecord {
	get := func(name string) jobTelemetrySegment {
		if capture == nil {
			return jobTelemetrySegment{}
		}
		if segment, ok := capture.segments[name]; ok {
			return segment
		}
		return jobTelemetrySegment{}
	}
	return jobTelemetryRecord{
		SchemaVersion: jobTelemetrySchemaVersion,
		AttemptID:     strings.TrimSpace(attemptID),
		JobName:       strings.TrimSpace(jobName),
		RecordedAt:    now.UTC().Format(time.RFC3339Nano),
		Queue:         get(jobTelemetrySegmentQueue),
		Preflight:     get(jobTelemetrySegmentPreflight),
		Model:         get(jobTelemetrySegmentModel),
		ToolCall:      get(jobTelemetrySegmentToolCall),
		Context:       get(jobTelemetrySegmentContext),
		Work:          get(jobTelemetrySegmentWork),
		Verification:  get(jobTelemetrySegmentVerification),
		Wait:          get(jobTelemetrySegmentWait),
	}
}

// attemptArtifactKindTelemetry registers this record as a third attempt-
// bound artifact kind alongside claims and verification
// (cmd/attempt_artifacts.go, plan 201-08/CAP-071). It has no pre-existing
// colony-wide filename -- this record never existed before this plan -- so
// its "legacy" name (registered in legacyAttemptArtifactNames) is simply a
// root-level name that will never be found on any real colony, letting
// readJobTelemetryRecord reuse readAttemptBoundArtifact's identical
// validation path rather than a second one.
const attemptArtifactKindTelemetry = "telemetry"

// writeJobTelemetryRecord writes record at its attempt-bound path, and
// skips writing entirely when nothing in it was measured -- a run that
// measured no segment at all writes no timing record rather than a record
// of zeros (this plan's own must_have), matching the exact "a run that
// filed nothing leaves no file" rule attemptBoundArtifactPath's siblings
// already hold. The returned error is for tests: every production call
// site treats it as non-fatal (recording timing must never fail a build or
// a check) and discards it after, at most, a warning.
func writeJobTelemetryRecord(record jobTelemetryRecord) error {
	if !record.hasAnyMeasuredSegment() {
		return nil
	}
	return writeAttemptBoundArtifact(record.AttemptID, attemptArtifactKindTelemetry, record)
}

// readJobTelemetryRecord reads the one telemetry record for attemptID, if
// any was ever written. ok is false both when none exists (nothing was
// measured, so nothing was written) and when the stored record cannot be
// read -- reading telemetry is exactly as best-effort as writing it, never
// a reason to fail whatever screen is trying to show it.
func readJobTelemetryRecord(attemptID string) (jobTelemetryRecord, bool) {
	attemptID = strings.TrimSpace(attemptID)
	if attemptID == "" {
		return jobTelemetryRecord{}, false
	}
	var record jobTelemetryRecord
	if err := readAttemptBoundArtifact(attemptID, attemptArtifactKindTelemetry, &record); err != nil {
		return jobTelemetryRecord{}, false
	}
	return record, true
}
