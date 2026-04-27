package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Output types
// ---------------------------------------------------------------------------

// PatrolCheckFile represents the status of a single file in the JSON validity check.
type PatrolCheckFile struct {
	Path     string `json:"path"`
	Status   string `json:"status"`
	Severity string `json:"severity,omitempty"`
	Details  string `json:"details,omitempty"`
	Size     *int   `json:"size,omitempty"`
}

// PatrolStaleEntry represents a single stale pheromone signal.
type PatrolStaleEntry struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// PatrolCheck is a single health check result.
type PatrolCheck struct {
	Name         string              `json:"name"`
	Status       string              `json:"status"`
	Files        []PatrolCheckFile   `json:"files,omitempty"`
	StaleCount   int                 `json:"stale_count,omitempty"`
	StaleSignals []PatrolStaleEntry  `json:"stale_signals,omitempty"`
	Artifacts    []string            `json:"artifacts,omitempty"`
}

// PatrolResult is the top-level output of patrol-check.
type PatrolResult struct {
	Checks        []PatrolCheck `json:"checks"`
	OverallStatus string        `json:"overall_status"`
}

// ---------------------------------------------------------------------------
// Command
// ---------------------------------------------------------------------------

var patrolCheckCmd = &cobra.Command{
	Use:   "patrol-check",
	Short: "Run colony health checks (JSON validity, stale signals, interrupted builds)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		basePath := store.BasePath()
		checks := []PatrolCheck{
			runJSONValidityCheck(basePath),
			runStalePheromonesCheck(basePath),
			runInterruptedBuildsCheck(basePath),
		}

		overall := "healthy"
		for _, c := range checks {
			switch c.Status {
			case "error":
				overall = "error"
			case "warning":
				if overall != "error" {
					overall = "warning"
				}
			}
		}

		outputOK(PatrolResult{
			Checks:        checks,
			OverallStatus: overall,
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(patrolCheckCmd)
}

// ---------------------------------------------------------------------------
// Check 1: JSON Validity
// ---------------------------------------------------------------------------

func runJSONValidityCheck(basePath string) PatrolCheck {
	files := []string{"COLONY_STATE.json", "pheromones.json", "session.json"}
	var results []PatrolCheckFile
	hasError := false

	for _, filename := range files {
		fc := checkPatrolFile(basePath, filename)
		results = append(results, fc)
		if fc.Status == "error" {
			hasError = true
		}
	}

	status := "healthy"
	if hasError {
		status = "error"
	}

	return PatrolCheck{
		Name:   "json_validity",
		Status: status,
		Files:  results,
	}
}

func checkPatrolFile(basePath, filename string) PatrolCheckFile {
	fullPath := filepath.Join(basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return PatrolCheckFile{Path: filename, Status: "missing", Severity: "info"}
		}
		return PatrolCheckFile{Path: filename, Status: "error", Severity: "warning",
			Details: fmt.Sprintf("read error: %v", err)}
	}

	if len(data) == 0 {
		return PatrolCheckFile{Path: filename, Status: "empty", Severity: "info"}
	}

	if !json.Valid(data) {
		return PatrolCheckFile{Path: filename, Status: "error", Severity: "warning",
			Details: "invalid JSON"}
	}

	size := len(data)
	return PatrolCheckFile{Path: filename, Status: "healthy", Size: &size}
}

// ---------------------------------------------------------------------------
// Check 2: Stale Pheromone Detection
// ---------------------------------------------------------------------------

func runStalePheromonesCheck(basePath string) PatrolCheck {
	// Load current phase
	var state colony.ColonyState
	stateData, err := os.ReadFile(filepath.Join(basePath, "COLONY_STATE.json"))
	if err != nil || !json.Valid(stateData) {
		return PatrolCheck{Name: "stale_pheromones", Status: "healthy",
			StaleCount: 0, StaleSignals: []PatrolStaleEntry{}}
	}
	if err := json.Unmarshal(stateData, &state); err != nil {
		return PatrolCheck{Name: "stale_pheromones", Status: "healthy",
			StaleCount: 0, StaleSignals: []PatrolStaleEntry{}}
	}

	// Load pheromones
	var pf colony.PheromoneFile
	pherData, err := os.ReadFile(filepath.Join(basePath, "pheromones.json"))
	if err != nil || !json.Valid(pherData) {
		return PatrolCheck{Name: "stale_pheromones", Status: "healthy",
			StaleCount: 0, StaleSignals: []PatrolStaleEntry{}}
	}
	if err := json.Unmarshal(pherData, &pf); err != nil {
		return PatrolCheck{Name: "stale_pheromones", Status: "healthy",
			StaleCount: 0, StaleSignals: []PatrolStaleEntry{}}
	}

	var stale []PatrolStaleEntry
	for _, sig := range pf.Signals {
		if !sig.Active {
			continue
		}
		if sig.SourcePhase != nil && *sig.SourcePhase < state.CurrentPhase {
			stale = append(stale, PatrolStaleEntry{
				ID:     sig.ID,
				Type:   sig.Type,
				Reason: fmt.Sprintf("references completed phase %d, current is %d",
					*sig.SourcePhase, state.CurrentPhase),
			})
			continue
		}
		if sig.Strength != nil && *sig.Strength <= 0 {
			stale = append(stale, PatrolStaleEntry{
				ID: sig.ID, Type: sig.Type, Reason: "zero strength",
			})
		}
	}

	status := "healthy"
	if len(stale) > 0 {
		status = "warning"
	}

	return PatrolCheck{
		Name:         "stale_pheromones",
		Status:       status,
		StaleCount:   len(stale),
		StaleSignals: stale,
	}
}

// ---------------------------------------------------------------------------
// Check 3: Interrupted Build Detection
// ---------------------------------------------------------------------------

func runInterruptedBuildsCheck(basePath string) PatrolCheck {
	patterns := []string{
		"build_manifest_*.json",
		"dispatch_manifest_*.json",
		"spawn-tree*.json",
	}

	var artifacts []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(basePath, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			artifacts = append(artifacts, filepath.Base(m))
		}
	}

	status := "healthy"
	if len(artifacts) > 0 {
		status = "warning"
	}

	return PatrolCheck{
		Name:      "interrupted_builds",
		Status:    status,
		Artifacts: artifacts,
	}
}
