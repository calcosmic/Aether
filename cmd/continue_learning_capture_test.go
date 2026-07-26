package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestLearningCaptureOnDefaultContinue locks in the fix for "the colony never
// learns." Durable learning capture previously existed only inside
// continue-finalize — a command the continue wrapper explicitly forbids on the
// default fast path — so in normal daily use pkg/learn never fired, no
// hypothesis was ever recorded, and every CROWNED-ANTHILL report showed
// "Learnings captured: 0" while the rest of seal worked.
//
// This test calls the shared capture point exactly as the default continue
// path does (empty runID, learning enabled, gates passed) and asserts a
// hypothesis entry lands in the colony learn store.
func TestLearningCaptureOnDefaultContinue(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{
		ID:          3,
		Name:        "Wire the exporter",
		Description: "Connect the vault exporter in cmd/exporter.go to the dashboard",
	}
	workerFlow := []codexContinueWorkerFlowStep{
		{Name: "Mason-10", Caste: "builder", Status: "completed", Summary: "implemented exporter wiring in cmd/exporter.go"},
		{Name: "Watch-4", Caste: "watcher", Status: "completed", Summary: "verified with go test ./cmd/..."},
	}
	gates := codexContinueGateReport{
		Passed: true,
		Checks: []gateCheck{
			{Name: "manifest_present", Passed: true},
			{Name: "verification_steps_passed", Passed: true},
		},
	}

	captureContinueLearning(phase, workerFlow, gates, "", false, time.Now().UTC())

	// The colony learn store persists to entries.json in the data dir
	// (pkg/learn/colony_store.go: entriesFile = "entries.json").
	entriesPath := filepath.Join(s.BasePath(), "entries.json")
	data, err := os.ReadFile(entriesPath)
	if err != nil {
		t.Fatalf("default-path continue captured no learning: %v", err)
	}
	if !strings.Contains(string(data), "Wire the exporter") {
		t.Fatalf("learn store entry does not reference the completed phase: %s", string(data))
	}
	if !strings.Contains(string(data), "hypothesis") {
		t.Fatalf("captured learning should start as a hypothesis: %s", string(data))
	}
}

// TestLearningCaptureRespectsFailedWorkers asserts the eligibility gate
// survived the extraction: a failed worker means no durable learning.
func TestLearningCaptureRespectsFailedWorkers(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 4, Name: "Failing phase", Description: "One worker failed"}
	workerFlow := []codexContinueWorkerFlowStep{
		{Name: "Mason-11", Caste: "builder", Status: "failed", Summary: "could not complete"},
	}
	gates := codexContinueGateReport{Passed: true}

	captureContinueLearning(phase, workerFlow, gates, "", false, time.Now().UTC())

	entriesPath := filepath.Join(s.BasePath(), "entries.json")
	if _, err := os.Stat(entriesPath); err == nil {
		data, _ := os.ReadFile(entriesPath)
		t.Fatalf("learning captured despite a failed worker: %s", string(data))
	}
}
