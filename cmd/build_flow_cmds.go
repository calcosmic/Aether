package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// versionCacheFile is the relative path to the version check cache.
const versionCacheFile = ".aether/data/.version-check-cache"

// versionCacheEntry represents a cached version check result.
type versionCacheEntry struct {
	Version  string `json:"version"`
	CachedAt string `json:"cached_at"`
}

var versionCheckCachedCmd = &cobra.Command{
	Use:   "version-check-cached",
	Short: "Display version with caching",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := resolveDataDir()
		cachePath := filepath.Join(dataDir, ".version-check-cache")

		// Try to read existing cache
		cacheData, err := os.ReadFile(cachePath)
		if err == nil {
			var entry versionCacheEntry
			if json.Unmarshal(cacheData, &entry) == nil {
				cachedAt, parseErr := time.Parse(time.RFC3339, entry.CachedAt)
				if parseErr == nil && time.Since(cachedAt) < 24*time.Hour {
					outputOK(map[string]interface{}{
						"version":   entry.Version,
						"cached":    true,
						"cached_at": entry.CachedAt,
					})
					return nil
				}
			}
		}

		// No valid cache; use the binary's built-in version
		currentVersion := Version
		now := time.Now().UTC().Format(time.RFC3339)

		// Write cache
		entry := versionCacheEntry{
			Version:  currentVersion,
			CachedAt: now,
		}
		entryData, _ := json.Marshal(entry)
		os.MkdirAll(filepath.Dir(cachePath), 0755)
		os.WriteFile(cachePath, entryData, 0644)

		outputOK(map[string]interface{}{
			"version":   currentVersion,
			"cached":    false,
			"cached_at": now,
		})
		return nil
	},
}

// resolveDataDir returns the .aether/data directory path.
func resolveDataDir() string {
	// Use the same resolution as pkg/storage
	if root := os.Getenv("AETHER_ROOT"); root != "" {
		return filepath.Join(root, ".aether", "data")
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, ".aether", "data")
}

var milestoneDetectCmd = &cobra.Command{
	Use:   "milestone-detect",
	Short: "Detect current milestone from colony state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("COLONY_STATE.json not found: %v", err), nil)
			return nil
		}

		milestone := state.Milestone
		phaseCount := len(state.Plan.Phases)
		completedCount := 0
		for _, p := range state.Plan.Phases {
			if p.Status == colony.PhaseCompleted {
				completedCount++
			}
		}

		// If no milestone set, try to detect from progress
		if milestone == "" && phaseCount > 0 {
			ratio := float64(completedCount) / float64(phaseCount)
			switch {
			case ratio >= 1.0:
				milestone = "Sealed Chambers"
			case ratio >= 0.75:
				milestone = "Ventilated Nest"
			case ratio >= 0.5:
				milestone = "Brood Stable"
			case ratio >= 0.25:
				milestone = "Open Chambers"
			default:
				milestone = "First Mound"
			}
		}

		if milestone == "" {
			milestone = "First Mound"
		}

		outputOK(map[string]interface{}{
			"milestone":       milestone,
			"phase_count":     phaseCount,
			"completed_count": completedCount,
		})
		return nil
	},
}

var updateProgressCmd = &cobra.Command{
	Use:   "update-progress",
	Short: "Update phase progress in colony state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		phaseNum := mustGetInt(cmd, "phase")
		status := mustGetString(cmd, "status")
		if status == "" {
			return nil
		}

		// Validate status
		validStatuses := []string{"pending", "ready", "in_progress", "completed"}
		valid := false
		for _, s := range validStatuses {
			if s == status {
				valid = true
				break
			}
		}
		if !valid {
			outputError(1, fmt.Sprintf("invalid status %q, must be one of: %s", status, strings.Join(validStatuses, ", ")), nil)
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("COLONY_STATE.json not found: %v", err), nil)
			return nil
		}

		// Phase is 1-indexed from the user; convert to 0-indexed
		idx := phaseNum - 1
		if idx < 0 || idx >= len(state.Plan.Phases) {
			outputError(1, fmt.Sprintf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases)), nil)
			return nil
		}

		// Check for --task flag
		taskID, _ := cmd.Flags().GetString("task")
		if taskID != "" {
			// Find and update specific task
			found := false
			for i, t := range state.Plan.Phases[idx].Tasks {
				if t.ID != nil && *t.ID == taskID {
					state.Plan.Phases[idx].Tasks[i].Status = status
					found = true
					break
				}
			}
			if !found {
				outputError(1, fmt.Sprintf("task %q not found in phase %d", taskID, phaseNum), nil)
				return nil
			}
		} else {
			// Update phase status
			state.Plan.Phases[idx].Status = status
		}

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"updated": true,
			"phase":   phaseNum,
			"status":  status,
		})
		return nil
	},
}

var printNextUpCmd = &cobra.Command{
	Use:   "print-next-up",
	Short: "Display next steps based on colony state",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, fmt.Sprintf("COLONY_STATE.json not found: %v", err), nil)
			return nil
		}

		var suggestions []string
		colonyState := string(state.State)

		switch colonyState {
		case "READY":
			nextPhase := state.CurrentPhase + 1
			suggestions = append(suggestions, fmt.Sprintf("Run /ant:build %d to start the next phase", nextPhase))
		case "EXECUTING":
			suggestions = append(suggestions, "Run /ant:continue to verify work and advance")
		case "BUILT":
			suggestions = append(suggestions, "Run /ant:continue to verify and advance")
		case "COMPLETED":
			suggestions = append(suggestions, "Colony complete! Run /ant:seal to finalize")
		default:
			suggestions = append(suggestions, "Run /ant:status to check colony state")
		}

		outputOK(map[string]interface{}{
			"suggestions":   suggestions,
			"current_phase": state.CurrentPhase,
			"state":         colonyState,
		})
		return nil
	},
}

var dataSafetyStatsCmd = &cobra.Command{
	Use:   "data-safety-stats",
	Short: "Display data integrity statistics",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := resolveDataDir()

		// Check data directory exists
		info, err := os.Stat(dataDir)
		if err != nil {
			outputOK(map[string]interface{}{
				"data_dir":         dataDir,
				"exists":           false,
				"file_count":       0,
				"state_exists":     false,
				"state_size_bytes": 0,
				"pheromones_count": 0,
				"session_active":   false,
				"lock_files_count": 0,
			})
			return nil
		}

		_ = info

		// Count files in data directory
		fileCount := 0
		lockCount := 0
		entries, _ := os.ReadDir(dataDir)
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
				if strings.HasSuffix(e.Name(), ".lock") {
					lockCount++
				}
			}
		}

		// Check COLONY_STATE.json
		stateExists := false
		stateSize := int64(0)
		statePath := filepath.Join(dataDir, "COLONY_STATE.json")
		if si, err := os.Stat(statePath); err == nil {
			stateExists = true
			stateSize = si.Size()
		}

		// Check pheromones.json
		pheromonesCount := 0
		pheromonesPath := filepath.Join(dataDir, "pheromones.json")
		if pData, err := os.ReadFile(pheromonesPath); err == nil {
			var pheromones struct {
				Signals []interface{} `json:"signals"`
			}
			if json.Unmarshal(pData, &pheromones) == nil {
				pheromonesCount = len(pheromones.Signals)
			}
		}

		// Check session.json
		sessionActive := false
		sessionPath := filepath.Join(dataDir, "session.json")
		if sData, err := os.ReadFile(sessionPath); err == nil {
			var session struct {
				ColonyGoal *string `json:"colony_goal"`
			}
			if json.Unmarshal(sData, &session) == nil && session.ColonyGoal != nil && *session.ColonyGoal != "" {
				sessionActive = true
			}
		}

		outputOK(map[string]interface{}{
			"data_dir":         dataDir,
			"exists":           true,
			"file_count":       fileCount,
			"state_exists":     stateExists,
			"state_size_bytes": stateSize,
			"pheromones_count": pheromonesCount,
			"session_active":   sessionActive,
			"lock_files_count": lockCount,
		})
		return nil
	},
}

func init() {
	updateProgressCmd.Flags().Int("phase", 0, "Phase number to update (required)")
	updateProgressCmd.Flags().String("status", "", "New status (required: pending, ready, in_progress, completed)")
	updateProgressCmd.Flags().String("task", "", "Task ID to update (optional, for task-level updates)")

	rootCmd.AddCommand(versionCheckCachedCmd)
	rootCmd.AddCommand(milestoneDetectCmd)
	rootCmd.AddCommand(updateProgressCmd)
	rootCmd.AddCommand(printNextUpCmd)
	rootCmd.AddCommand(dataSafetyStatsCmd)
}
