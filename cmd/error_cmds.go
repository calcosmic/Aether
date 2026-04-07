package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// --- error-add ---

var errorAddCmd = &cobra.Command{
	Use:   "error-add",
	Short: "Add an error record to COLONY_STATE.json",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		category := mustGetString(cmd, "category")
		if category == "" {
			return nil
		}
		severity := mustGetString(cmd, "severity")
		if severity == "" {
			return nil
		}
		description := mustGetString(cmd, "description")
		if description == "" {
			return nil
		}
		phaseStr, _ := cmd.Flags().GetString("phase")

		// Generate ID: err_{unix_timestamp}_{4_random_hex}
		ts := time.Now().UTC()
		rnd := make([]byte, 2)
		rand.Read(rnd)
		id := fmt.Sprintf("err_%d_%s", ts.Unix(), hex.EncodeToString(rnd))

		// Load or initialize state
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			// Initialize with minimal state
			state = colony.ColonyState{
				Version: "3.0",
				Errors:  colony.Errors{Records: []colony.ErrorRecord{}},
			}
		}
		if state.Errors.Records == nil {
			state.Errors.Records = []colony.ErrorRecord{}
		}

		// Build the error record
		record := colony.ErrorRecord{
			ID:          id,
			Category:    category,
			Severity:    severity,
			Description: description,
			RootCause:   nil,
			TaskID:      nil,
			Timestamp:   ts.Format(time.RFC3339),
		}

		// Parse optional phase
		if phaseStr != "" {
			phaseNum, err := strconv.Atoi(phaseStr)
			if err == nil {
				record.Phase = &phaseNum
			}
		}

		// Append record
		state.Errors.Records = append(state.Errors.Records, record)

		// Cap at 50 records, trimming oldest
		if len(state.Errors.Records) > 50 {
			state.Errors.Records = state.Errors.Records[len(state.Errors.Records)-50:]
		}

		// Save
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":       id,
			"category": category,
		})
		return nil
	},
}

// --- error-flag-pattern ---

// ErrorPattern represents a single error pattern in error-patterns.json.
type ErrorPattern struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Severity    string   `json:"severity"`
	FirstSeen   string   `json:"first_seen"`
	LastSeen    string   `json:"last_seen"`
	Occurrences int      `json:"occurrences"`
	Projects    []string `json:"projects"`
	Resolved    bool     `json:"resolved"`
}

// ErrorPatternsFile represents the error-patterns.json file structure.
type ErrorPatternsFile struct {
	Version  float64        `json:"version"`
	Patterns []ErrorPattern `json:"patterns"`
}

var errorFlagPatternCmd = &cobra.Command{
	Use:   "error-flag-pattern",
	Short: "Create or update a recurring error pattern",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}
		description := mustGetString(cmd, "description")
		if description == "" {
			return nil
		}
		severity, _ := cmd.Flags().GetString("severity")
		if severity == "" {
			severity = "warning"
		}

		ts := time.Now().UTC().Format(time.RFC3339)
		projectName := "unknown"
		if cwd, err := os.Getwd(); err == nil {
			// Get the last path component as project name
			for i := len(cwd) - 1; i >= 0; i-- {
				if cwd[i] == '/' {
					projectName = cwd[i+1:]
					break
				}
			}
		}

		// Load or initialize patterns file
		var pf ErrorPatternsFile
		if err := store.LoadJSON("error-patterns.json", &pf); err != nil {
			pf = ErrorPatternsFile{Version: 1, Patterns: []ErrorPattern{}}
		}
		if pf.Patterns == nil {
			pf.Patterns = []ErrorPattern{}
		}

		// Check if pattern already exists
		for i := range pf.Patterns {
			if pf.Patterns[i].Name == name {
				// Update existing pattern
				pf.Patterns[i].Occurrences++
				pf.Patterns[i].LastSeen = ts
				// Add project if not already present
				found := false
				for _, p := range pf.Patterns[i].Projects {
					if p == projectName {
						found = true
						break
					}
				}
				if !found {
					pf.Patterns[i].Projects = append(pf.Patterns[i].Projects, projectName)
				}

				if err := store.SaveJSON("error-patterns.json", pf); err != nil {
					outputError(2, fmt.Sprintf("failed to save patterns: %v", err), nil)
					return nil
				}

				outputOK(map[string]interface{}{
					"updated":     true,
					"pattern":     name,
					"occurrences": pf.Patterns[i].Occurrences,
				})
				return nil
			}
		}

		// Add new pattern
		pf.Patterns = append(pf.Patterns, ErrorPattern{
			Name:        name,
			Description: description,
			Severity:    severity,
			FirstSeen:   ts,
			LastSeen:    ts,
			Occurrences: 1,
			Projects:    []string{projectName},
			Resolved:    false,
		})

		if err := store.SaveJSON("error-patterns.json", pf); err != nil {
			outputError(2, fmt.Sprintf("failed to save patterns: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"pattern": name,
		})
		return nil
	},
}

// --- error-summary ---

var errorSummaryCmd = &cobra.Command{
	Use:   "error-summary",
	Short: "Summarize error records by category and severity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		records := state.Errors.Records
		if records == nil {
			records = []colony.ErrorRecord{}
		}

		byCategory := map[string]int{}
		bySeverity := map[string]int{}
		for _, r := range records {
			byCategory[r.Category]++
			bySeverity[r.Severity]++
		}

		outputOK(map[string]interface{}{
			"total":       len(records),
			"by_category": byCategory,
			"by_severity": bySeverity,
		})
		return nil
	},
}

// --- error-pattern-check (deprecated) ---

var errorPatternCheckCmd = &cobra.Command{
	Use:        "error-pattern-check",
	Short:      "Check for recurring error patterns (deprecated)",
	Args:       cobra.NoArgs,
	Deprecated: "use error-flag-pattern instead",
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		records := state.Errors.Records
		if records == nil {
			outputOK([]interface{}{})
			return nil
		}

		// Group by category
		groups := map[string][]colony.ErrorRecord{}
		for _, r := range records {
			groups[r.Category] = append(groups[r.Category], r)
		}

		type patternGroup struct {
			Category  string `json:"category"`
			Count     int    `json:"count"`
			FirstSeen string `json:"first_seen"`
			LastSeen  string `json:"last_seen"`
		}

		var result []patternGroup
		for cat, recs := range groups {
			if len(recs) >= 3 {
				// Sort by timestamp to find first and last
				sorted := make([]colony.ErrorRecord, len(recs))
				copy(sorted, recs)
				sort.Slice(sorted, func(i, j int) bool {
					return sorted[i].Timestamp < sorted[j].Timestamp
				})
				result = append(result, patternGroup{
					Category:  cat,
					Count:     len(recs),
					FirstSeen: sorted[0].Timestamp,
					LastSeen:  sorted[len(sorted)-1].Timestamp,
				})
			}
		}

		if result == nil {
			outputOK([]interface{}{})
			return nil
		}
		outputOK(result)
		return nil
	},
}

func init() {
	errorAddCmd.Flags().String("category", "", "Error category (required)")
	errorAddCmd.Flags().String("severity", "", "Error severity (required)")
	errorAddCmd.Flags().String("description", "", "Error description (required)")
	errorAddCmd.Flags().String("phase", "", "Optional phase number")

	errorFlagPatternCmd.Flags().String("name", "", "Pattern name (required)")
	errorFlagPatternCmd.Flags().String("description", "", "Pattern description (required)")
	errorFlagPatternCmd.Flags().String("severity", "warning", "Pattern severity")

	rootCmd.AddCommand(errorAddCmd)
	rootCmd.AddCommand(errorFlagPatternCmd)
	rootCmd.AddCommand(errorSummaryCmd)
	rootCmd.AddCommand(errorPatternCheckCmd)
}
