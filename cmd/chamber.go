package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var chamberCreateCmd = &cobra.Command{
	Use:   "chamber-create",
	Short: "Create a chamber archive entry",
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
		goal, _ := cmd.Flags().GetString("goal")
		milestone, _ := cmd.Flags().GetString("milestone")
		phasesCompleted, _ := cmd.Flags().GetInt("phases-completed")
		totalPhases, _ := cmd.Flags().GetInt("total-phases")

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chamberDir := filepath.Join(aetherRoot, ".aether", "chambers", name)

		if err := os.MkdirAll(chamberDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create chamber directory: %v", err), nil)
			return nil
		}

		manifest := map[string]interface{}{
			"name":             name,
			"goal":             goal,
			"milestone":        milestone,
			"phases_completed": phasesCompleted,
			"total_phases":     totalPhases,
		}

		manifestData, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to marshal manifest: %v", err), nil)
			return nil
		}
		manifestData = append(manifestData, '\n')

		manifestPath := filepath.Join(chamberDir, "manifest.json")
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write manifest: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"name":    name,
			"path":    chamberDir,
		})
		return nil
	},
}

var chamberVerifyCmd = &cobra.Command{
	Use:   "chamber-verify",
	Short: "Verify chamber integrity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chamberDir := filepath.Join(aetherRoot, ".aether", "chambers", name)

		manifestPath := filepath.Join(chamberDir, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			outputError(1, fmt.Sprintf("chamber %q not found: %v", name, err), nil)
			return nil
		}

		if !json.Valid(data) {
			outputError(1, fmt.Sprintf("chamber %q has invalid manifest.json", name), nil)
			return nil
		}

		// List files in chamber directory
		entries, err := os.ReadDir(chamberDir)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read chamber directory: %v", err), nil)
			return nil
		}

		files := make([]string, 0, len(entries))
		for _, e := range entries {
			files = append(files, e.Name())
		}
		sort.Strings(files)

		outputOK(map[string]interface{}{
			"name":  name,
			"valid": true,
			"files": files,
		})
		return nil
	},
}

var chamberListCmd = &cobra.Command{
	Use:   "chamber-list",
	Short: "List all chambers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")

		entries, err := os.ReadDir(chambersDir)
		if err != nil {
			if os.IsNotExist(err) {
				outputOK(map[string]interface{}{
					"chambers": []interface{}{},
					"by_scope": emptyChamberScopeGroups(),
					"total":    0,
				})
				return nil
			}
			outputError(1, fmt.Sprintf("failed to read chambers directory: %v", err), nil)
			return nil
		}

		chambers := []interface{}{}
		byScope := emptyChamberScopeGroups()
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(chambersDir, entry.Name(), "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue // skip directories without manifest.json
			}
			var manifest map[string]interface{}
			if err := json.Unmarshal(data, &manifest); err != nil {
				continue // skip invalid manifests
			}
			manifest = manifestWithEffectiveScope(manifest)
			chambers = append(chambers, manifest)
			scope := string(chamberManifestScope(manifest))
			byScope[scope] = append(byScope[scope], manifest)
		}

		outputOK(map[string]interface{}{
			"chambers": chambers,
			"by_scope": byScope,
			"total":    len(chambers),
		})
		return nil
	},
}

var chamberCompareCmd = &cobra.Command{
	Use:   "chamber-compare [name]",
	Short: "Compare chamber archive with current colony state",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		name, _ := cmd.Flags().GetString("name")
		if len(args) > 0 && name == "" {
			name = args[0]
		}
		if name == "" {
			outputErrorMessage("chamber name is required (--name or positional arg)")
			return nil
		}

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		manifestPath := filepath.Join(aetherRoot, ".aether", "chambers", name, "manifest.json")

		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			outputOK(map[string]interface{}{
				"chamber": name,
				"error":   fmt.Sprintf("chamber '%s' not found", name),
				"matches": []interface{}{},
				"diffs":   []interface{}{},
			})
			return nil
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			outputOK(map[string]interface{}{
				"chamber": name,
				"error":   fmt.Sprintf("invalid manifest: %v", err),
				"matches": []interface{}{},
				"diffs":   []interface{}{},
			})
			return nil
		}

		var matches []interface{}
		var diffs []interface{}
		totalCompared := 0

		// Load colony state (graceful degradation if not available)
		state, stateErr := loadActiveColonyState()

		// Compare goal
		totalCompared++
		manifestGoal := stringValue(manifest["goal"])
		var currentGoal string
		if stateErr == nil && state.Goal != nil {
			currentGoal = *state.Goal
		}
		if manifestGoal == currentGoal {
			matches = append(matches, map[string]interface{}{"field": "goal", "chamber": manifestGoal, "current": currentGoal})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "goal", "chamber_value": manifestGoal, "current_value": currentGoal})
		}

		// Compare milestone
		totalCompared++
		manifestMilestone := stringValue(manifest["milestone"])
		var currentMilestone string
		if stateErr == nil {
			currentMilestone = state.Milestone
		}
		if manifestMilestone == currentMilestone {
			matches = append(matches, map[string]interface{}{"field": "milestone", "chamber": manifestMilestone, "current": currentMilestone})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "milestone", "chamber_value": manifestMilestone, "current_value": currentMilestone})
		}

		// Compare phases_completed
		totalCompared++
		manifestPhases := toInt(manifest["phases_completed"])
		currentPhases := 0
		if stateErr == nil {
			for _, p := range state.Plan.Phases {
				if p.Status == colony.PhaseCompleted {
					currentPhases++
				}
			}
		}
		if manifestPhases == currentPhases {
			matches = append(matches, map[string]interface{}{"field": "phases_completed", "chamber": manifestPhases, "current": currentPhases})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "phases_completed", "chamber_value": manifestPhases, "current_value": currentPhases})
		}

		// Compare total_phases
		totalCompared++
		manifestTotal := toInt(manifest["total_phases"])
		currentTotal := 0
		if stateErr == nil {
			currentTotal = len(state.Plan.Phases)
		}
		if manifestTotal == currentTotal {
			matches = append(matches, map[string]interface{}{"field": "total_phases", "chamber": manifestTotal, "current": currentTotal})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "total_phases", "chamber_value": manifestTotal, "current_value": currentTotal})
		}

		if matches == nil {
			matches = []interface{}{}
		}
		if diffs == nil {
			diffs = []interface{}{}
		}

		result := map[string]interface{}{
			"chamber":        name,
			"matches":        matches,
			"diffs":          diffs,
			"total_compared": totalCompared,
		}

		if stateErr != nil {
			result["error"] = "colony state not available"
		}

		outputOK(result)
		return nil
	},
}

func init() {
	chamberCreateCmd.Flags().String("name", "", "Chamber name (required)")
	chamberCreateCmd.Flags().String("goal", "", "Colony goal")
	chamberCreateCmd.Flags().String("milestone", "", "Milestone name")
	chamberCreateCmd.Flags().Int("phases-completed", 0, "Number of phases completed")
	chamberCreateCmd.Flags().Int("total-phases", 0, "Total number of phases")

	chamberVerifyCmd.Flags().String("name", "", "Chamber name (required)")

	chamberCompareCmd.Flags().String("name", "", "Chamber name to compare")

	rootCmd.AddCommand(chamberCreateCmd)
	rootCmd.AddCommand(chamberVerifyCmd)
	rootCmd.AddCommand(chamberListCmd)
	rootCmd.AddCommand(chamberCompareCmd)
}

func chamberManifestScope(manifest map[string]interface{}) colony.ColonyScope {
	scope, err := colony.ParseColonyScope(stringValue(manifest["scope"]))
	if err != nil {
		return colony.ScopeProject
	}
	return scope.Effective()
}

func manifestWithEffectiveScope(manifest map[string]interface{}) map[string]interface{} {
	if manifest == nil {
		return nil
	}
	manifest["scope"] = string(chamberManifestScope(manifest))
	return manifest
}

func emptyChamberScopeGroups() map[string][]interface{} {
	return map[string][]interface{}{
		string(colony.ScopeProject): {},
		string(colony.ScopeMeta):    {},
	}
}

// toInt converts an interface{} to int, handling float64 (JSON numbers) and int types.
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
