package cmd

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildAttemptFixtureUsesCanonicalTransaction(t *testing.T) {
	generatedAt := time.Date(2026, time.September, 9, 12, 35, 0, 0, time.UTC)
	processID := 3501
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt:    generatedAt,
		AttemptID:      deriveBuildAttemptID(generatedAt, processID),
		RunID:          "run-35353535353535353535353535353501",
		ProcessID:      processID,
		ExecutionOwner: "fixture-owner",
		DispatchMode:   "fixture-direct",
		MakeLatest:     true,
	})

	if fixture.Receipt.Path != buildStartReceiptPath(fixture.Request.Phase, fixture.Request.AttemptID) {
		t.Fatalf("fixture receipt path = %q, want canonical build-start receipt", fixture.Receipt.Path)
	}
	if fixture.Attempt.ID != fixture.Request.AttemptID || fixture.Attempt.RunID != fixture.Request.RunID {
		t.Fatalf("fixture attempt lost explicit identity: request=%+v attempt=%+v", fixture.Request, fixture.Attempt)
	}
	wantDataRoot := filepath.Join(fixture.Root, ".aether", "data")
	if got := filepath.Clean(store.BasePath()); got != filepath.Clean(wantDataRoot) {
		t.Fatalf("fixture store root = %q, want physically contained %q", got, wantDataRoot)
	}
	if fixture.Latest == nil || fixture.Latest.AttemptID != fixture.Attempt.ID {
		t.Fatalf("fixture latest pointer = %+v, want attempt %q", fixture.Latest, fixture.Attempt.ID)
	}
}
