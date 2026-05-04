package codex

import (
	"path/filepath"
	"strings"
	"testing"
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
