package codex

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeWorkerHandoffNormalizesPathsAndFreshness(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "cmd", "worker.go")

	handoff := NormalizeWorkerHandoff(root, WorkerHandoff{
		ChangedFiles:       []string{absPath, " cmd/worker.go ", ""},
		CommandsRun:        []string{"go test ./...", "go test ./..."},
		VerificationStatus: "not run",
	})

	if len(handoff.ChangedFiles) != 1 || handoff.ChangedFiles[0] != "cmd/worker.go" {
		t.Fatalf("changed files = %#v, want repo-relative deduped path", handoff.ChangedFiles)
	}
	if handoff.VerificationStatus != "not_run" {
		t.Fatalf("verification status = %q, want not_run", handoff.VerificationStatus)
	}
	if strings.TrimSpace(handoff.Freshness) == "" {
		t.Fatal("freshness should be populated")
	}
	if err := ValidateWorkerHandoff(handoff); err != nil {
		t.Fatalf("normalized handoff should validate: %v", err)
	}
}

func TestValidateWorkerHandoffRejectsInvalidVerificationStatus(t *testing.T) {
	err := ValidateWorkerHandoff(WorkerHandoff{VerificationStatus: "maybe"})
	if err == nil {
		t.Fatal("expected invalid verification status to fail validation")
	}
}

// The shipped worker-handoff contract promises "timestamp or statement"; the
// contract doc's own worked example used prose. A packet following the doc
// must never be rejected — prose freshness validates and is coerced to a
// sortable RFC3339 receipt time on normalization.
func TestWorkerHandoffFreshnessStatementIsAcceptedAndCoerced(t *testing.T) {
	prose := WorkerHandoff{Freshness: "Evidence collected after latest edit."}
	if err := ValidateWorkerHandoff(prose); err != nil {
		t.Fatalf("prose freshness must validate (contract doc's own example): %v", err)
	}

	normalized := NormalizeWorkerHandoff(t.TempDir(), prose)
	if _, err := time.Parse(time.RFC3339, normalized.Freshness); err != nil {
		t.Fatalf("prose freshness must be coerced to RFC3339, got %q", normalized.Freshness)
	}

	kept := NormalizeWorkerHandoff(t.TempDir(), WorkerHandoff{Freshness: "2026-07-31T14:00:00Z"})
	if kept.Freshness != "2026-07-31T14:00:00Z" {
		t.Fatalf("valid RFC3339 freshness must be kept verbatim, got %q", kept.Freshness)
	}

	notRun := NormalizeWorkerHandoff(t.TempDir(), WorkerHandoff{Freshness: "not_run"})
	if notRun.Freshness != "not-run" {
		t.Fatalf("not_run must normalize to not-run, got %q", notRun.Freshness)
	}
}
