package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Immune system tracks retries and self-healing patterns.

type scarEntry struct {
	ID        string `json:"id"`
	Error     string `json:"error"`
	Pattern   string `json:"pattern"`
	CreatedAt string `json:"created_at"`
}

type scarsData struct {
	Scars []scarEntry `json:"scars"`
}

const maxScars = 100

// --- scar-add ---

var scarAddCmd = &cobra.Command{
	Use:   "scar-add",
	Short: "Record a failure pattern for future avoidance",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		errMsg := mustGetString(cmd, "error")
		if errMsg == "" {
			return nil
		}
		pattern := mustGetString(cmd, "pattern")
		if pattern == "" {
			return nil
		}

		var sd scarsData
		if err := store.LoadJSON("scars.json", &sd); err != nil {
			sd = scarsData{}
		}

		scar := scarEntry{
			ID:        fmt.Sprintf("scar_%d", time.Now().UnixNano()),
			Error:     errMsg,
			Pattern:   pattern,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}

		sd.Scars = append(sd.Scars, scar)
		// Cap at maxScars, evicting oldest
		if len(sd.Scars) > maxScars {
			sd.Scars = sd.Scars[len(sd.Scars)-maxScars:]
		}

		if err := store.SaveJSON("scars.json", sd); err != nil {
			outputError(2, fmt.Sprintf("failed to save scars: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"added": true, "scar_id": scar.ID, "total": len(sd.Scars)})
		return nil
	},
}

// --- scar-list ---

var scarListCmd = &cobra.Command{
	Use:   "scar-list",
	Short: "Return all recorded scars",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var sd scarsData
		if err := store.LoadJSON("scars.json", &sd); err != nil {
			outputOK(map[string]interface{}{"scars": []scarEntry{}, "total": 0})
			return nil
		}

		outputOK(map[string]interface{}{"scars": sd.Scars, "total": len(sd.Scars)})
		return nil
	},
}

// --- scar-check ---

var scarCheckCmd = &cobra.Command{
	Use:   "scar-check",
	Short: "Check if command matches any scar pattern",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		command := mustGetString(cmd, "command")
		if command == "" {
			return nil
		}

		var sd scarsData
		if err := store.LoadJSON("scars.json", &sd); err != nil {
			outputOK(map[string]interface{}{"scarred": false, "matches": 0})
			return nil
		}

		cmdLower := strings.ToLower(command)
		var matches []scarEntry
		for _, s := range sd.Scars {
			if strings.Contains(cmdLower, strings.ToLower(s.Pattern)) {
				matches = append(matches, s)
			}
		}

		if len(matches) > 0 {
			outputOK(map[string]interface{}{
				"scarred": true,
				"matches": len(matches),
				"scars":   matches,
			})
		} else {
			outputOK(map[string]interface{}{"scarred": false, "matches": 0})
		}
		return nil
	},
}

// --- immune-auto-scar ---

var immuneAutoScarCmd = &cobra.Command{
	Use:   "immune-auto-scar",
	Short: "Auto-detect failure patterns from midden",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Load midden for recent failures
		midden, err := loadMiddenFile(store)
		if err != nil {
			outputOK(map[string]interface{}{"detected": 0, "reason": "no midden data"})
			return nil
		}

		var sd scarsData
		if err := store.LoadJSON("scars.json", &sd); err != nil {
			sd = scarsData{}
		}

		// Build existing pattern set for dedup
		existingPatterns := make(map[string]bool)
		for _, s := range sd.Scars {
			existingPatterns[strings.ToLower(s.Pattern)] = true
		}

		var newScars []scarEntry
		for _, entry := range midden.Entries {
			category := entry.Category
			description := entry.Message
			if category == "" || description == "" {
				continue
			}

			pattern := strings.ToLower(category)
			if existingPatterns[pattern] {
				continue
			}

			scar := scarEntry{
				ID:        fmt.Sprintf("scar_%d", time.Now().UnixNano()+int64(len(newScars))),
				Error:     description,
				Pattern:   category,
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			}
			newScars = append(newScars, scar)
			existingPatterns[pattern] = true
		}

		if len(newScars) == 0 {
			outputOK(map[string]interface{}{"detected": 0})
			return nil
		}

		sd.Scars = append(sd.Scars, newScars...)
		if len(sd.Scars) > maxScars {
			sd.Scars = sd.Scars[len(sd.Scars)-maxScars:]
		}

		if err := store.SaveJSON("scars.json", sd); err != nil {
			outputError(2, fmt.Sprintf("failed to save scars: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"detected": len(newScars), "total": len(sd.Scars)})
		return nil
	},
}

func init() {
	scarAddCmd.Flags().String("error", "", "Error message (required)")
	scarAddCmd.Flags().String("pattern", "", "Pattern to match (required)")
	scarCheckCmd.Flags().String("command", "", "Command to check (required)")

	for _, c := range []*cobra.Command{
		scarAddCmd, scarListCmd, scarCheckCmd, immuneAutoScarCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
