package cmd

import (
	"fmt"
	"math"
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

// --- trophallaxis-diagnose ---

var trophallaxisDiagnoseCmd = &cobra.Command{
	Use:   "trophallaxis-diagnose",
	Short: "Analyze error and suggest retry strategy",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		errMsg := mustGetString(cmd, "error")
		if errMsg == "" {
			return nil
		}

		diagnosis := diagnoseError(errMsg)

		outputOK(diagnosis)
		return nil
	},
}

func diagnoseError(errMsg string) map[string]interface{} {
	errLower := strings.ToLower(errMsg)
	var strategy string
	var retryable bool

	switch {
	case strings.Contains(errLower, "permission denied") || strings.Contains(errLower, "eacces"):
		strategy = "check_file_permissions"
		retryable = false
	case strings.Contains(errLower, "file not found") || strings.Contains(errLower, "enoent"):
		strategy = "check_file_exists"
		retryable = false
	case strings.Contains(errLower, "timeout") || strings.Contains(errLower, "deadline exceeded"):
		strategy = "retry_with_backoff"
		retryable = true
	case strings.Contains(errLower, "connection refused") || strings.Contains(errLower, "econnrefused"):
		strategy = "retry_with_backoff"
		retryable = true
	case strings.Contains(errLower, "lock") || strings.Contains(errLower, "busy"):
		strategy = "retry_after_delay"
		retryable = true
	case strings.Contains(errLower, "invalid json") || strings.Contains(errLower, "unmarshal"):
		strategy = "check_data_integrity"
		retryable = false
	default:
		strategy = "manual_review"
		retryable = false
	}

	backoff := 0
	if retryable {
		backoff = 5 // default 5 seconds
	}

	return map[string]interface{}{
		"error":     errMsg,
		"strategy":  strategy,
		"retryable": retryable,
		"backoff_s": backoff,
		"diagnosed": time.Now().UTC().Format(time.RFC3339),
	}
}

// --- trophallaxis-retry ---

var trophallaxisRetryCmd = &cobra.Command{
	Use:   "trophallaxis-retry",
	Short: "Record retry attempt with backoff info",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		commandName := mustGetString(cmd, "command")
		if commandName == "" {
			return nil
		}
		attempt := mustGetInt(cmd, "attempt")

		maxAttempts := 3
		if attempt >= maxAttempts {
			outputOK(map[string]interface{}{
				"retry":   false,
				"reason":  "max_attempts_reached",
				"attempt": attempt,
				"max":     maxAttempts,
			})
			return nil
		}

		// Exponential backoff: 2^attempt * base
		backoff := int(math.Pow(2, float64(attempt)) * 2)

		outputOK(map[string]interface{}{
			"retry":     true,
			"attempt":   attempt + 1,
			"backoff_s": backoff,
			"max":       maxAttempts,
		})
		return nil
	},
}

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
	trophallaxisDiagnoseCmd.Flags().String("error", "", "Error message to diagnose (required)")
	trophallaxisRetryCmd.Flags().String("command", "", "Command name (required)")
	trophallaxisRetryCmd.Flags().Int("attempt", 0, "Current attempt number (required)")
	scarAddCmd.Flags().String("error", "", "Error message (required)")
	scarAddCmd.Flags().String("pattern", "", "Pattern to match (required)")
	scarCheckCmd.Flags().String("command", "", "Command to check (required)")

	for _, c := range []*cobra.Command{
		trophallaxisDiagnoseCmd, trophallaxisRetryCmd,
		scarAddCmd, scarListCmd, scarCheckCmd, immuneAutoScarCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
