package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// commandFileEntry maps a command name to its directory and required files.
type commandFileEntry struct {
	dir   string
	files []string
}

// commandFileMap returns the command-to-directory-and-files mapping
// used by session-clear and session-verify-fresh.
func commandFileMap() map[string]commandFileEntry {
	return map[string]commandFileEntry{
		"survey": {
			dir:   ".aether/data/survey",
			files: []string{"PROVISIONS.md", "TRAILS.md", "BLUEPRINT.md", "CHAMBERS.md", "DISCIPLINES.md", "SENTINEL-PROTOCOLS.md", "PATHOGENS.md"},
		},
		"oracle": {
			dir:   ".aether/oracle",
			files: []string{"state.json", "plan.json", "gaps.md", "synthesis.md", "research-plan.md", ".stop", ".last-topic"},
		},
		"watch": {
			dir:   ".aether/data",
			files: []string{"watch-status.txt", "watch-progress.txt"},
		},
		"swarm": {
			dir:   ".aether/data/swarm",
			files: []string{"findings.json", "display.json", "timing.json"},
		},
		"init": {
			dir:   ".aether/data",
			files: []string{"COLONY_STATE.json", "constraints.json"},
		},
		"seal": {
			dir:   ".aether/data/archive",
			files: []string{"manifest.json"},
		},
		"entomb": {
			dir:   ".aether/data/archive",
			files: []string{"manifest.json"},
		},
	}
}

// protectedCommands lists commands that cannot be auto-cleared.
var protectedCommands = map[string]bool{
	"init":   true,
	"seal":   true,
	"entomb": true,
}

// ---------------------------------------------------------------------------
// session-init (CMD-09)
// ---------------------------------------------------------------------------

var sessionInitCmd = &cobra.Command{
	Use:   "session-init",
	Short: "Initialize a new colony session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sessionID, _ := cmd.Flags().GetString("session-id")
		goal, _ := cmd.Flags().GetString("goal")

		if sessionID == "" {
			sessionID = fmt.Sprintf("%d_%s", time.Now().Unix(), randomHex(4))
		}

		// Rotate spawn-tree
		rotateSpawnTree(store)

		// Get baseline commit
		baseline := getGitHEAD()

		session := colony.SessionFile{
			SessionID:        sessionID,
			StartedAt:        time.Now().UTC().Format(time.RFC3339),
			ColonyGoal:       goal,
			CurrentPhase:     0,
			CurrentMilestone: "First Mound",
			SuggestedNext:    "aether plan",
			ContextCleared:   false,
			BaselineCommit:   baseline,
			ResumedAt:        nil,
			ActiveTodos:      []string{},
			Summary:          "Session initialized",
		}

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(2, fmt.Sprintf("failed to save session: %v", err), nil)
			return nil
		}

		// Mirror to legacy path if COLONY_DATA_DIR is set
		mirrorToLegacy(store)

		outputOK(map[string]interface{}{
			"session_id": sessionID,
			"goal":       goal,
			"file":       "session.json",
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// session-read (CMD-10)
// ---------------------------------------------------------------------------

var sessionReadCmd = &cobra.Command{
	Use:   "session-read",
	Short: "Read session data",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var session colony.SessionFile
		if err := store.LoadJSON("session.json", &session); err != nil {
			outputOK(map[string]interface{}{
				"exists":  false,
				"session": nil,
			})
			return nil
		}

		// Staleness check: >24h = stale
		isStale := false
		ageHours := 0
		ts := session.LastCommandAt
		if ts == "" {
			ts = session.StartedAt
		}
		if ts != "" {
			if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
				age := time.Since(parsed)
				ageHours = int(age.Hours())
				if age.Hours() > 24 {
					isStale = true
				}
			}
		}

		outputOK(map[string]interface{}{
			"exists":    true,
			"is_stale":  isStale,
			"age_hours": ageHours,
			"session":   session,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// session-update (CMD-11)
// ---------------------------------------------------------------------------

var sessionUpdateCmd = &cobra.Command{
	Use:   "session-update",
	Short: "Update session data",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		commandName := mustGetString(cmd, "command")
		if commandName == "" {
			return nil
		}

		suggestedNext, _ := cmd.Flags().GetString("suggested-next")
		summary, _ := cmd.Flags().GetString("summary")

		var session colony.SessionFile
		sessionExists := true
		if err := store.LoadJSON("session.json", &session); err != nil {
			// Auto-init if missing
			sessionExists = false
			sessionID := fmt.Sprintf("%d_%s", time.Now().Unix(), randomHex(4))
			session = colony.SessionFile{
				SessionID:        sessionID,
				StartedAt:        time.Now().UTC().Format(time.RFC3339),
				ColonyGoal:       "",
				CurrentPhase:     0,
				CurrentMilestone: "First Mound",
				SuggestedNext:    "aether plan",
				ContextCleared:   false,
				BaselineCommit:   getGitHEAD(),
				ResumedAt:        nil,
				ActiveTodos:      []string{},
				Summary:          "Session auto-initialized",
			}
		}

		// Try to load COLONY_STATE.json for goal/phase/milestone overrides
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err == nil {
			if state.Goal != nil && *state.Goal != "" {
				session.ColonyGoal = *state.Goal
			}
			if state.CurrentPhase > 0 {
				session.CurrentPhase = state.CurrentPhase
			}
			if state.Milestone != "" {
				session.CurrentMilestone = state.Milestone
			}
			session.ActiveTodos = sessionActiveTodosFromState(state)
		}

		// Update fields
		now := time.Now().UTC().Format(time.RFC3339)
		session.LastCommand = commandName
		session.LastCommandAt = now
		if suggestedNext != "" {
			session.SuggestedNext = suggestedNext
		}
		if summary != "" {
			session.Summary = summary
		}
		session.BaselineCommit = getGitHEAD()

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(2, fmt.Sprintf("failed to save session: %v", err), nil)
			return nil
		}

		// Mirror to legacy path
		mirrorToLegacy(store)

		outputOK(map[string]interface{}{
			"updated":          true,
			"command":          commandName,
			"auto_initialized": !sessionExists,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// session-clear (CMD-12)
// ---------------------------------------------------------------------------

var sessionClearCmd = &cobra.Command{
	Use:   "session-clear",
	Short: "Clear command-scoped session files",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		commandName := mustGetString(cmd, "command")
		if commandName == "" {
			return nil
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		// Protected command blocking
		if protectedCommands[commandName] {
			outputError(1, fmt.Sprintf("Command %q is protected and cannot be auto-cleared.", commandName), nil)
			return nil
		}

		// Look up command files
		cfm := commandFileMap()
		entry, ok := cfm[commandName]
		if !ok {
			outputError(1, fmt.Sprintf("Unknown command: %s", commandName), nil)
			return nil
		}

		baseDir := storage.ResolveAetherRoot(context.Background())
		dir := filepath.Join(baseDir, entry.dir)

		var cleared []string
		var errors []string

		for _, file := range entry.files {
			path := filepath.Join(dir, file)
			if _, err := os.Stat(path); err != nil {
				continue // file doesn't exist, skip
			}
			if dryRun {
				cleared = append(cleared, file)
				continue
			}
			if err := os.Remove(path); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", file, err))
			} else {
				cleared = append(cleared, file)
			}
		}

		// For oracle: also clear discoveries/ subdirectory
		if commandName == "oracle" {
			discoveriesDir := filepath.Join(dir, "discoveries")
			if info, err := os.Stat(discoveriesDir); err == nil && info.IsDir() {
				if !dryRun {
					if err := os.RemoveAll(discoveriesDir); err != nil {
						errors = append(errors, fmt.Sprintf("discoveries/: %v", err))
					} else {
						cleared = append(cleared, "discoveries/")
					}
				} else {
					cleared = append(cleared, "discoveries/")
				}
			}
		}

		outputOK(map[string]interface{}{
			"command": commandName,
			"cleared": cleared,
			"errors":  errors,
			"dry_run": dryRun,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// session-verify-fresh (CMD-14)
// ---------------------------------------------------------------------------

var sessionVerifyFreshCmd = &cobra.Command{
	Use:   "session-verify-fresh [--command NAME] [--force] [session_start_time]",
	Short: "Detect stale files by mtime comparison",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		commandName := mustGetString(cmd, "command")
		if commandName == "" {
			return nil
		}

		force, _ := cmd.Flags().GetBool("force")

		// Parse optional session_start_time from positional arg
		var sessionStart int64
		if len(args) > 0 {
			_, err := fmt.Sscanf(args[0], "%d", &sessionStart)
			if err != nil {
				outputError(1, fmt.Sprintf("invalid session_start_time: %s", args[0]), nil)
				return nil
			}
		}

		// Look up command files
		cfm := commandFileMap()
		entry, ok := cfm[commandName]
		if !ok {
			outputError(1, fmt.Sprintf("Unknown command: %s", commandName), nil)
			return nil
		}

		baseDir := storage.ResolveAetherRoot(context.Background())
		dir := filepath.Join(baseDir, entry.dir)

		var fresh, stale, missing []string
		var totalLines int

		for _, file := range entry.files {
			path := filepath.Join(dir, file)
			info, err := os.Stat(path)
			if err != nil {
				missing = append(missing, file)
				continue
			}

			mtime := info.ModTime().Unix()

			if force {
				fresh = append(fresh, file)
			} else if sessionStart > 0 {
				if mtime >= sessionStart {
					fresh = append(fresh, file)
				} else {
					stale = append(stale, file)
				}
			} else {
				// No session start time provided: all existing files are fresh
				fresh = append(fresh, file)
			}

			// Count lines in the file
			lineCount, _ := countLines(path)
			totalLines += lineCount
		}

		pass := len(stale) == 0 || force

		outputOK(map[string]interface{}{
			"ok":          pass,
			"command":     commandName,
			"fresh":       fresh,
			"stale":       stale,
			"missing":     missing,
			"total_lines": totalLines,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// session-mark-resumed (CMD-13)
// ---------------------------------------------------------------------------

var sessionMarkResumedCmd = &cobra.Command{
	Use:   "session-mark-resumed",
	Short: "Mark session as resumed",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		var session colony.SessionFile
		if err := store.LoadJSON("session.json", &session); err != nil {
			outputError(1, "No active session to mark as resumed.", nil)
			return nil
		}

		resumedAt := time.Now().UTC().Format(time.RFC3339)
		session.ResumedAt = &resumedAt
		session.ContextCleared = false

		if err := store.SaveJSON("session.json", session); err != nil {
			outputError(2, fmt.Sprintf("failed to save session: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"resumed":   true,
			"timestamp": resumedAt,
		})
		return nil
	},
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// randomHex generates n random bytes and returns their hex encoding.
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// getGitHEAD returns the current git HEAD commit hash, or "" on error.
func getGitHEAD() string {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// rotateSpawnTree archives spawn-tree.txt if it exists and is non-empty.
// Keeps only the 5 most recent archive files.
func rotateSpawnTree(s *storage.Store) {
	spawnTreePath := filepath.Join(s.BasePath(), "spawn-tree.txt")

	data, err := os.ReadFile(spawnTreePath)
	if err != nil || len(bytes.TrimSpace(data)) == 0 {
		return
	}

	// Create archive directory
	archiveDir := filepath.Join(filepath.Dir(spawnTreePath), "spawn-tree-archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("rotateSpawnTree: failed to create archive dir %s: %v", archiveDir, err)
	}

	// Copy to archive with timestamp
	timestamp := time.Now().Format("20060102_150405")
	archivePath := filepath.Join(archiveDir, "spawn-tree."+timestamp+".txt")
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		log.Printf("rotateSpawnTree: failed to write archive %s: %v", archivePath, err)
	}

	// Truncate original in-place
	if err := os.WriteFile(spawnTreePath, []byte{}, 0644); err != nil {
		log.Printf("rotateSpawnTree: failed to truncate spawn-tree %s: %v", spawnTreePath, err)
	}

	// Keep only 5 most recent archives
	entries, err := os.ReadDir(archiveDir)
	if err != nil {
		return
	}
	var archives []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "spawn-tree.") {
			archives = append(archives, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(archives)))
	for i := 5; i < len(archives); i++ {
		if err := os.Remove(filepath.Join(archiveDir, archives[i])); err != nil {
			log.Printf("rotateSpawnTree: failed to remove old archive %s: %v", archives[i], err)
		}
	}
}

// mirrorToLegacy copies session.json to the legacy path if COLONY_DATA_DIR
// is set and differs from the default data directory.
func mirrorToLegacy(s *storage.Store) {
	dataDir := storage.ResolveDataDir(context.Background())
	defaultDir := filepath.Join(storage.ResolveAetherRoot(context.Background()), ".aether", "data")
	if dataDir == defaultDir {
		return // no mirror needed
	}

	// Read from store
	data, err := os.ReadFile(filepath.Join(s.BasePath(), "session.json"))
	if err != nil {
		return
	}

	// Write to legacy path
	legacyDir := defaultDir
	os.MkdirAll(legacyDir, 0755)
	os.WriteFile(filepath.Join(legacyDir, "session.json"), data, 0644)
}

// countLines counts the number of lines in a file.
func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// ---------------------------------------------------------------------------
// init: register all session commands
// ---------------------------------------------------------------------------

func init() {
	// session-init
	sessionInitCmd.Flags().String("session-id", "", "Session ID (auto-generated if empty)")
	sessionInitCmd.Flags().String("goal", "", "Colony goal")
	rootCmd.AddCommand(sessionInitCmd)

	// session-read
	rootCmd.AddCommand(sessionReadCmd)

	// session-update
	sessionUpdateCmd.Flags().String("command", "", "Command being executed (required)")
	sessionUpdateCmd.Flags().String("suggested-next", "", "Suggested next command")
	sessionUpdateCmd.Flags().String("summary", "", "Session summary")
	rootCmd.AddCommand(sessionUpdateCmd)

	// session-clear
	sessionClearCmd.Flags().String("command", "", "Command scope to clear (required)")
	sessionClearCmd.Flags().Bool("dry-run", false, "Show what would be cleared without deleting")
	rootCmd.AddCommand(sessionClearCmd)

	// session-verify-fresh
	sessionVerifyFreshCmd.Flags().String("command", "", "Command to check freshness for (required)")
	sessionVerifyFreshCmd.Flags().Bool("force", false, "Force all files to be considered fresh")
	rootCmd.AddCommand(sessionVerifyFreshCmd)

	// session-mark-resumed
	rootCmd.AddCommand(sessionMarkResumedCmd)
}
