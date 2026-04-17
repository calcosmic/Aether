package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// initCmd implements the `aether init` command.
// It creates the colony directory structure, COLONY_STATE.json, session.json,
// CONTEXT.md, and activity.log. It is idempotent -- if a colony is already
// initialized, it reports the existing state without overwriting.
//
// Sealed colony detection: if a sealed colony is detected, the command checks
// for uncommitted changes (in-progress seal) before allowing overwrite.
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

		// Check idempotency: if COLONY_STATE.json exists, inspect it
		statePath := filepath.Join(dataDir, "COLONY_STATE.json")
		if _, err := os.Stat(statePath); err == nil {
			// Colony already initialized -- load and inspect
			var existing colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &existing); loadErr == nil {
				// An entombed/reset colony leaves the state scaffold in place but
				// clears the goal. Treat that as no active colony.
				if existing.Goal == nil || strings.TrimSpace(ptrStr(existing.Goal)) == "" || existing.State == colony.StateIDLE {
					goto createFreshColony
				}
				// If colony is sealed, check for in-progress seal (uncommitted changes)
				if existing.Milestone == "Crowned Anthill" {
					if sealInProgress(dataDir) {
						outputError(1, "a seal operation appears to be in progress (COLONY_STATE.json has uncommitted changes with Crowned Anthill milestone). Wait for the seal to complete, commit the seal state, or run `aether entomb` first.", nil)
						return nil
					}
					// Sealed colony with committed state — allow overwrite (fresh init)
					// Fall through to create new colony state
				} else {
					// Active (non-sealed) colony — block
					outputError(1, fmt.Sprintf("colony already initialized (state=%s, phase=%d, goal=%q)",
						existing.State, existing.CurrentPhase, ptrStr(existing.Goal)), nil)
					return nil
				}
			}
		}

	createFreshColony:
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

		// Backup old colony state before overwriting (sealed colony fresh-init)
		if _, err := os.Stat(statePath); err == nil {
			backupDir := filepath.Join(dataDir, "backups")
			if err := os.MkdirAll(backupDir, 0755); err == nil {
				backupFile := filepath.Join(backupDir, fmt.Sprintf("COLONY_STATE.pre-init.%s.bak", time.Now().Format("20060102-150405")))
				if err := copyFile(statePath, backupFile); err == nil {
					fmt.Fprintf(os.Stderr, "warning: backed up previous colony state to %s\n", backupFile)
				}
			}
		}

		// Create COLONY_STATE.json v3.0
		state := colony.ColonyState{
			Version:       "3.0",
			Goal:          &goal,
			ColonyVersion: 0,
			State:         colony.StateREADY,
			CurrentPhase:  0,
			SessionID:     &sessionID,
			InitializedAt: &now,
			Plan:          colony.Plan{Phases: []colony.Phase{}},
			Memory: colony.Memory{
				PhaseLearnings: []colony.PhaseLearning{},
				Decisions:      []colony.Decision{},
				Instincts:      []colony.Instinct{},
			},
			Errors: colony.Errors{
				Records:         []colony.ErrorRecord{},
				FlaggedPatterns: []colony.FlaggedPattern{},
			},
			Signals:      []colony.Signal{},
			Graveyards:   []colony.Graveyard{},
			Events:       []string{},
			ParallelMode: colony.ModeInRepo,
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
			SuggestedNext:    "aether plan",
			ActiveTodos:      []string{},
			Summary:          "Colony initialized",
		}

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(1, fmt.Sprintf("failed to create session.json: %v", err), nil)
			return nil
		}

		if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
			CommandName:   "init",
			SuggestedNext: "aether plan",
			Summary:       "Colony initialized",
			HandoffTitle:  "Initialized Colony",
			WriteHandoff:  true,
		}); err != nil {
			outputError(1, fmt.Sprintf("failed to create recovery artifacts: %v", err), nil)
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

		result := map[string]interface{}{
			"state":    string(colony.StateREADY),
			"goal":     goal,
			"version":  "3.0",
			"phase":    0,
			"session":  sessionID,
			"data_dir": dataDir,
		}
		outputWorkflow(result, renderInitVisual(goal, sessionID, dataDir))
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

// sealInProgress checks whether COLONY_STATE.json has uncommitted changes
// that indicate a seal is in progress (working tree has Crowned Anthill milestone
// but the HEAD commit does not). This prevents a new colony init from overwriting
// a seal that hasn't been committed yet.
func sealInProgress(dataDir string) bool {
	// Resolve the repo root from dataDir (strip .aether/data to get project root)
	projectRoot := filepath.Dir(filepath.Dir(dataDir))

	// Check if we're in a git repo
	gitCmd := exec.Command("git", "rev-parse", "--git-dir")
	gitCmd.Dir = projectRoot
	if _, err := gitCmd.CombinedOutput(); err != nil {
		return false
	}

	// Get the repo root so we can compute a path relative to it
	topCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	topCmd.Dir = projectRoot
	rootOut, err := topCmd.CombinedOutput()
	if err != nil {
		return false
	}
	gitRoot := strings.TrimSpace(string(rootOut))

	// Build path relative to git root.
	// Use gitRoot (resolved by git) instead of dataDir to avoid
	// macOS /var -> /private/var symlink mismatch.
	stateRelPath := filepath.Join(".aether", "data", "COLONY_STATE.json")

	// Check if COLONY_STATE.json has uncommitted changes
	diffCmd := exec.Command("git", "diff", "--name-only", "HEAD", "--", stateRelPath)
	diffCmd.Dir = gitRoot
	diffOut, err := diffCmd.CombinedOutput()
	if err != nil || len(strings.TrimSpace(string(diffOut))) == 0 {
		return false
	}

	// The working tree differs from HEAD — check if HEAD has the seal milestone
	showCmd := exec.Command("git", "show", "HEAD:"+stateRelPath)
	showCmd.Dir = gitRoot
	showOut, err := showCmd.CombinedOutput()
	if err != nil {
		// File doesn't exist in HEAD — it's a new file, not an in-progress seal
		return false
	}

	// Check if the committed version has Crowned Anthill
	committed := string(showOut)
	if !strings.Contains(committed, "Crowned Anthill") {
		// HEAD does NOT have the seal — the seal is uncommitted
		return true
	}

	return false
}
