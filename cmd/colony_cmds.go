package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var colonyNameCmd = &cobra.Command{
	Use:   "colony-name",
	Short: "Get the colony name from state",
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

		name := ""

		// Try session_id first (format: "colonyname_timestamp")
		if state.SessionID != nil && *state.SessionID != "" {
			parts := strings.Split(*state.SessionID, "_")
			if len(parts) >= 1 {
				name = parts[0]
			}
		}

		// Fallback to colony_name
		if name == "" && state.ColonyName != nil {
			name = *state.ColonyName
		}

		// Fallback to goal (first word, sanitized)
		if name == "" && state.Goal != nil && *state.Goal != "" {
			words := strings.Fields(*state.Goal)
			if len(words) > 0 {
				name = strings.ToLower(words[0])
				name = strings.Map(func(r rune) rune {
					if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
						return r
					}
					return -1
				}, name)
			}
		}

		outputOK(map[string]interface{}{
			"name": name,
		})
		return nil
	},
}

var colonyDepthCmd = &cobra.Command{
	Use:   "colony-depth",
	Short: "Get or set colony depth",
	Args:  cobra.NoArgs,
}

var colonyDepthGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current colony depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{
				"depth":  "standard",
				"source": "default",
			})
			return nil
		}

		depth := string(state.ColonyDepth)
		source := "state"
		if depth == "" {
			depth = "standard"
			source = "default"
		}

		outputOK(map[string]interface{}{
			"depth":  depth,
			"source": source,
		})
		return nil
	},
}

var colonyDepthSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set colony depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		depth := mustGetString(cmd, "depth")
		if depth == "" {
			return nil
		}

		d := colony.ColonyDepth(depth)
		if !d.Valid() {
			outputError(1, fmt.Sprintf("invalid depth %q: must be light, standard, deep, or full", depth), nil)
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		state.ColonyDepth = d
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"depth":  depth,
			"source": "cli",
		})
		return nil
	},
}

var planGranularityCmd = &cobra.Command{
	Use:   "plan-granularity",
	Short: "Get or set planning granularity",
	Args:  cobra.NoArgs,
}

var planGranularityGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current planning granularity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{
				"granularity": "none",
				"source":      "default",
				"min":         0,
				"max":         0,
			})
			return nil
		}

		g := string(state.PlanGranularity)
		source := "state"
		if g == "" {
			g = "none"
			source = "default"
		}
		min, max := colony.GranularityRange(state.PlanGranularity)
		if state.PlanGranularity == "" {
			min, max = 0, 0
		}
		outputOK(map[string]interface{}{
			"granularity": g,
			"source":      source,
			"min":         min,
			"max":         max,
		})
		return nil
	},
}

var planGranularitySetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set planning granularity",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		granularity := mustGetString(cmd, "granularity")
		if granularity == "" {
			return nil
		}

		g := colony.PlanGranularity(granularity)
		if !g.Valid() {
			outputError(1, fmt.Sprintf("invalid granularity %q: must be sprint, milestone, quarter, or major", granularity), nil)
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		state.PlanGranularity = g
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		min, max := colony.GranularityRange(g)
		outputOK(map[string]interface{}{
			"granularity": granularity,
			"source":      "cli",
			"min":         min,
			"max":         max,
		})
		return nil
	},
}

var domainDetectCmd = &cobra.Command{
	Use:   "domain-detect",
	Short: "Detect project domain from file patterns",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		domains := []string{}

		checks := map[string][]string{
			"go":     {"go.mod", "go.sum"},
			"web":    {"package.json", "next.config.js", "vite.config.ts"},
			"ruby":   {"Gemfile", "Rakefile"},
			"python": {"requirements.txt", "setup.py", "pyproject.toml"},
			"rust":   {"Cargo.toml"},
		}

		// Search in project root (parent of .aether/data)
		searchDir := "."
		if store != nil {
			searchDir = filepath.Dir(filepath.Dir(store.BasePath()))
		}

		for domain, files := range checks {
			for _, f := range files {
				if _, err := os.Stat(filepath.Join(searchDir, f)); err == nil {
					domains = append(domains, domain)
					break
				}
			}
		}

		if domains == nil {
			domains = []string{}
		}

		outputOK(map[string]interface{}{
			"domains": domains,
		})
		return nil
	},
}

func init() {
	colonyDepthSetCmd.Flags().String("depth", "", "Depth level: light, standard, deep, full (required)")

	colonyDepthCmd.AddCommand(colonyDepthGetCmd)
	colonyDepthCmd.AddCommand(colonyDepthSetCmd)

	planGranularitySetCmd.Flags().String("granularity", "", "Granularity level: sprint, milestone, quarter, major (required)")

	planGranularityCmd.AddCommand(planGranularityGetCmd)
	planGranularityCmd.AddCommand(planGranularitySetCmd)

	rootCmd.AddCommand(colonyNameCmd)
	rootCmd.AddCommand(colonyDepthCmd)
	rootCmd.AddCommand(planGranularityCmd)
	rootCmd.AddCommand(domainDetectCmd)
}
