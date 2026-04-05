package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/spf13/cobra"
)

// initCmd implements the `aether init` command.
// It creates the colony directory structure, COLONY_STATE.json, session.json,
// CONTEXT.md, and activity.log. It is idempotent -- if a colony is already
// initialized, it reports the existing state without overwriting.
var initCmd = &cobra.Command{
	Use:   "init <goal>",
	Short: "Initialize a new colony in the current directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		goal := strings.TrimSpace(args[0])
		if goal == "" {
			outputError(1, "goal must not be empty", nil)
			return nil
		}

		dataDir := store.BasePath()
		aetherDir := filepath.Dir(dataDir)

		// Check idempotency: if COLONY_STATE.json exists, report and exit
		statePath := filepath.Join(dataDir, "COLONY_STATE.json")
		if _, err := os.Stat(statePath); err == nil {
			// Colony already initialized -- load and report
			var existing colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &existing); loadErr == nil {
				outputError(1, fmt.Sprintf("colony already initialized (state=%s, phase=%d, goal=%q)",
					existing.State, existing.CurrentPhase, ptrStr(existing.Goal)), nil)
				return nil
			}
		}

		now := time.Now()
		nowStr := now.Format(time.RFC3339)

		// Generate session ID: first word of goal + timestamp
		sanitizedGoal := strings.ToLower(strings.Fields(goal)[0])
		sessionID := fmt.Sprintf("%s_%d", sanitizedGoal, now.Unix())

		// Create directory structure
		if err := os.MkdirAll(filepath.Join(aetherDir, "dreams"), 0755); err != nil {
			outputError(1, fmt.Sprintf("failed to create directory structure: %v", err), nil)
			return nil
		}

		// Create COLONY_STATE.json v3.0
		state := colony.ColonyState{
			Version:        "3.0",
			Goal:           &goal,
			ColonyVersion:  0,
			State:          colony.StateREADY,
			CurrentPhase:   0,
			SessionID:      &sessionID,
			InitializedAt:  &now,
			Plan:           colony.Plan{Phases: []colony.Phase{}},
			Memory: colony.Memory{
				PhaseLearnings: []colony.PhaseLearning{},
				Decisions:      []colony.Decision{},
				Instincts:      []colony.Instinct{},
			},
			Errors: colony.Errors{
				Records:         []colony.ErrorRecord{},
				FlaggedPatterns: []colony.FlaggedPattern{},
			},
			Signals:    []colony.Signal{},
			Graveyards: []colony.Graveyard{},
			Events:     []string{},
		}

		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(1, fmt.Sprintf("failed to create COLONY_STATE.json: %v", err), nil)
			return nil
		}

		// Create session.json
		session := colony.SessionFile{
			SessionID:        sessionID,
			StartedAt:        nowStr,
			ColonyGoal:       goal,
			CurrentPhase:     0,
			CurrentMilestone: "",
			SuggestedNext:    "plan",
			ActiveTodos:      []string{},
		}

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(1, fmt.Sprintf("failed to create session.json: %v", err), nil)
			return nil
		}

		// Create CONTEXT.md
		contextContent := fmt.Sprintf(`# Colony Context

> Initialized: %s

## Goal

%s

## State

- **Status:** READY
- **Current Phase:** 0
- **Version:** 3.0
`, nowStr, goal)

		contextPath := filepath.Join(aetherDir, "CONTEXT.md")
		if err := os.WriteFile(contextPath, []byte(contextContent), 0644); err != nil {
			outputError(1, fmt.Sprintf("failed to create CONTEXT.md: %v", err), nil)
			return nil
		}

		// Initialize activity.log with first entry
		activityEntry := map[string]interface{}{
			"timestamp": nowStr,
			"action":    "COLONY_INITIALIZED",
			"detail":    fmt.Sprintf("goal=%q session=%s", goal, sessionID),
		}

		if err := store.AppendJSONL("activity.log", activityEntry); err != nil {
			outputError(1, fmt.Sprintf("failed to create activity.log: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"state":    string(colony.StateREADY),
			"goal":     goal,
			"version":  "3.0",
			"phase":    0,
			"session":  sessionID,
			"data_dir": dataDir,
		})
		return nil
	},
}

// ptrStr safely dereferences a *string, returning "" if nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func init() {
	rootCmd.AddCommand(initCmd)
}
