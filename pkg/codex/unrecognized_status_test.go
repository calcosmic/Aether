package codex

import (
	"strings"
	"testing"
)

// Observed in a real colony (2026-08-21): a watcher came back "failed" and the
// build halted with an empty error message, an empty summary, no blockers, and
// a worker report still reading "Status: spawned". The operator was told the
// build failed and given nothing at all to act on. The cause was here: any
// status outside the vocabulary was overwritten with a bare "failed" and the
// word the worker actually used was discarded.
func TestUnrecognizedWorkerStatusSaysWhatItWas(t *testing.T) {
	claims := normalizeWorkerClaims(workerClaims{
		Status:  "partial",
		Summary: "verified two of three criteria; ran out of budget on the third",
	}, WorkerConfig{Root: t.TempDir(), Caste: "watcher", TaskID: "verification"})

	if claims.Status != "failed" {
		t.Fatalf("an unrecognized status must not be trusted as success, got %q", claims.Status)
	}
	if len(claims.Blockers) == 0 {
		t.Fatal("a coerced failure with no blockers is undiagnosable -- the operator sees only \"failed\"")
	}
	joined := strings.Join(claims.Blockers, "; ")
	if !strings.Contains(joined, "partial") {
		t.Fatalf("the status the worker actually reported must survive the coercion, got %q", joined)
	}
	if !strings.Contains(joined, "ran out of budget") {
		t.Fatalf("the worker's own summary must survive when it is the only explanation there is, got %q", joined)
	}
}

func TestMissingWorkerStatusSaysSo(t *testing.T) {
	claims := normalizeWorkerClaims(workerClaims{}, WorkerConfig{Root: t.TempDir(), Caste: "watcher"})
	if claims.Status != "failed" {
		t.Fatalf("a missing status must fail, got %q", claims.Status)
	}
	if len(claims.Blockers) == 0 || !strings.Contains(strings.Join(claims.Blockers, "; "), "no status") {
		t.Fatalf("a result with no status at all must say that, got %+v", claims.Blockers)
	}
}

// A worker that reported real blockers already explains itself; the synthetic
// note must not be bolted on top of a genuine explanation.
func TestBlockedWorkerKeepsOnlyItsOwnBlockers(t *testing.T) {
	claims := normalizeWorkerClaims(workerClaims{
		Status:   "wedged",
		Blockers: []string{"the API key is missing"},
	}, WorkerConfig{Root: t.TempDir(), Caste: "builder"})

	if claims.Status != "blocked" {
		t.Fatalf("a worker with blockers is blocked, got %q", claims.Status)
	}
	if len(claims.Blockers) != 1 || claims.Blockers[0] != "the API key is missing" {
		t.Fatalf("a worker that explained itself must not be second-guessed, got %+v", claims.Blockers)
	}
}

// The recognized vocabulary must never fall into the coercion branch.
func TestRecognizedStatusesAreNeverCoerced(t *testing.T) {
	for raw, want := range map[string]string{
		"completed":           "completed",
		"code_written":        "completed",
		"completed_no_change": "completed_no_change",
		"verified_existing":   "completed_no_change",
		"interrupted":         "interrupted",
		"rate_limited":        "interrupted",
		"failed":              "failed",
		"blocked":             "blocked",
	} {
		claims := normalizeWorkerClaims(workerClaims{Status: raw}, WorkerConfig{Root: t.TempDir(), Caste: "builder"})
		if claims.Status != want {
			t.Errorf("normalizeWorkerClaims(%q) status = %q, want %q", raw, claims.Status, want)
		}
		for _, blocker := range claims.Blockers {
			if strings.Contains(blocker, "unrecognized status") {
				t.Errorf("%q was wrongly treated as unrecognized", raw)
			}
		}
	}
}
