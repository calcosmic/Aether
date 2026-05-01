package learn

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/memory"
)

// WorkerResult represents a single worker's outcome for evidence collection.
type WorkerResult struct {
	Name         string
	Caste        string
	Status       string
	FilesTouched []string
}

// GateResult represents gate pass/fail counts for evidence collection.
type GateResult struct {
	Passed int
	Total  int
}

// CollectEvidence assembles a full Evidence struct from run data.
// Confidence is computed via memory.Calculate trust scoring (D-09).
// Scope defaults to "repo-local" if empty.
func CollectEvidence(
	runID string,
	phase int,
	workers []WorkerResult,
	gates GateResult,
	scope string,
) Evidence {
	workerEvidence := make([]WorkerEvidence, len(workers))
	var allFiles []string
	for i, w := range workers {
		workerEvidence[i] = WorkerEvidence{
			Name:   w.Name,
			Caste:  w.Caste,
			Status: w.Status,
		}
		allFiles = append(allFiles, w.FilesTouched...)
	}

	// Compute confidence via trust scoring engine (RESEARCH.md: "Don't Hand-Roll")
	// SourceType "success_pattern" maps to 0.8 weight in trust scoring.
	trustResult := memory.Calculate(memory.TrustInput{
		SourceType: "success_pattern",
		Evidence:   "test_verified",
		DaysSince:  0, // fresh run
	})

	if scope == "" {
		scope = "repo-local"
	}

	return Evidence{
		RunID:        runID,
		Phase:        phase,
		Workers:      workerEvidence,
		FilesTouched: allFiles,
		GatesPassed:  gates.Passed,
		GatesTotal:   gates.Total,
		Confidence:   trustResult.Score,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Scope:        scope,
	}
}

// FormatConfidence returns a human-readable confidence tier string.
func FormatConfidence(score float64) string {
	tierName, _ := memory.Tier(score)
	return fmt.Sprintf("%.2f (%s)", score, tierName)
}
