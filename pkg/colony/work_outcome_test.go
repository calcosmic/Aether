package colony

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestWorkOutcomeVocabularyIsClosed proves the six-verdict vocabulary is
// closed: exactly six constants, Valid() rejects everything else, decoding
// an unrecognised value fails and names the offending value, and a
// marshal/unmarshal round trip returns the same value for every declared
// verdict.
func TestWorkOutcomeVocabularyIsClosed(t *testing.T) {
	all := AllWorkOutcomes()
	if len(all) != 6 {
		t.Fatalf("AllWorkOutcomes() returned %d verdicts, want exactly 6: %v", len(all), all)
	}

	for _, verdict := range all {
		if !verdict.Valid() {
			t.Errorf("declared verdict %q reports Valid() == false", verdict)
		}
	}

	for _, bogus := range []WorkOutcome{"", "SUCCESS", "success ", "done", "ok", "in_progress"} {
		if bogus.Valid() {
			t.Errorf("undeclared value %q reports Valid() == true", bogus)
		}
	}

	// Decoding an unrecognised verdict string fails and names the offending
	// value.
	var decoded WorkOutcome
	err := json.Unmarshal([]byte(`"bogus_verdict"`), &decoded)
	if err == nil {
		t.Fatalf("expected an error decoding an unrecognised work outcome, got nil (decoded=%q)", decoded)
	}
	if !strings.Contains(err.Error(), "bogus_verdict") {
		t.Errorf("error %q does not name the offending value %q", err.Error(), "bogus_verdict")
	}

	// Marshalling a verdict and decoding it again returns the same value,
	// for every declared verdict.
	for _, verdict := range all {
		data, err := json.Marshal(verdict)
		if err != nil {
			t.Fatalf("marshal %q: %v", verdict, err)
		}
		var roundTripped WorkOutcome
		if err := json.Unmarshal(data, &roundTripped); err != nil {
			t.Fatalf("unmarshal %q (from %s): %v", verdict, data, err)
		}
		if roundTripped != verdict {
			t.Errorf("round trip of %q produced %q", verdict, roundTripped)
		}
	}
}

// TestWorkOutcomeLifecycleMappingIsTotal proves LifecycleOutcome() is a
// total function over the declared verdict set: iterating every constant
// from AllWorkOutcomes(), each one returns a valid OutcomeKind with no
// error -- never a case that falls through to a default.
func TestWorkOutcomeLifecycleMappingIsTotal(t *testing.T) {
	for _, verdict := range AllWorkOutcomes() {
		outcome, err := verdict.LifecycleOutcome()
		if err != nil {
			t.Errorf("verdict %q has no lifecycle outcome mapping: %v", verdict, err)
			continue
		}
		if !outcome.Valid() {
			t.Errorf("verdict %q mapped to invalid OutcomeKind %q", verdict, outcome)
		}
	}
}

// TestAbsentWorkOutcomeIsNotSuccess proves an absent (zero-value) verdict is
// reported as absent -- never coerced to success -- both directly and after
// decoding a record whose JSON simply has no work-outcome key.
func TestAbsentWorkOutcomeIsNotSuccess(t *testing.T) {
	var zero WorkOutcome
	if zero.Valid() {
		t.Errorf("zero-value WorkOutcome reports Valid() == true")
	}
	if zero.IsSuccess() {
		t.Errorf("zero-value WorkOutcome reports IsSuccess() == true")
	}

	type wrapper struct {
		Verdict WorkOutcome `json:"verdict,omitempty"`
	}
	var decoded wrapper
	if err := json.Unmarshal([]byte(`{}`), &decoded); err != nil {
		t.Fatalf("decode record with no verdict key: %v", err)
	}
	if decoded.Verdict != "" {
		t.Errorf("decoding a record with no verdict key produced %q, want absent (empty)", decoded.Verdict)
	}
	if decoded.Verdict.IsSuccess() {
		t.Errorf("an absent verdict reports IsSuccess() == true")
	}
	if decoded.Verdict.Valid() {
		t.Errorf("an absent verdict reports Valid() == true")
	}
}

// TestWorkOutcomeLabelsAreComplete proves WorkOutcomeLabels() gives exactly
// one entry per declared verdict, with no gaps and no extras.
func TestWorkOutcomeLabelsAreComplete(t *testing.T) {
	labels := WorkOutcomeLabels()
	if len(labels) != 6 {
		t.Fatalf("WorkOutcomeLabels() returned %d entries, want exactly 6: %v", len(labels), labels)
	}
	for _, verdict := range AllWorkOutcomes() {
		if strings.TrimSpace(labels[verdict]) == "" {
			t.Errorf("verdict %q has no label", verdict)
		}
	}
}
