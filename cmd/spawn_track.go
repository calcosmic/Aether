package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// Spawn tracking for agent timeout enforcement.
// Tracks when agents start and whether they've exceeded their timeout.

type spawnTrackEntry struct {
	Agent    string `json:"agent"`
	Task     string `json:"task"`
	Start    int64  `json:"start"`
	Timeout  int    `json:"timeout"` // seconds, 0 = no timeout
	TimedOut bool   `json:"timed_out"`
}

var spawnTrackCmd = &cobra.Command{
	Use:   "spawn-track",
	Short: "Track agent spawn times for timeout enforcement",
	Long: `Track when agents start and check if they've exceeded their timeout.
Used by timeout guard hooks to block agents that run too long.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		action := mustGetString(cmd, "action")
		if action == "" {
			return nil
		}

		switch action {
		case "start":
			return spawnTrackStart(cmd)
		case "check":
			return spawnTrackCheck(cmd)
		case "clear":
			return spawnTrackClear(cmd)
		default:
			outputError(1, fmt.Sprintf("unknown action %q: must be start, check, or clear", action), nil)
			return nil
		}
	},
}

func spawnTrackStart(cmd *cobra.Command) error {
	agent := mustGetString(cmd, "agent")
	if agent == "" {
		return nil
	}
	task := mustGetString(cmd, "task")
	timeout := mustGetInt(cmd, "timeout")

	entry := spawnTrackEntry{
		Agent:   agent,
		Task:    task,
		Start:   time.Now().Unix(),
		Timeout: timeout,
	}

	if err := writeSpawnTrack(entry); err != nil {
		outputError(2, fmt.Sprintf("failed to write spawn track: %v", err), nil)
		return nil
	}

	outputOK(map[string]interface{}{
		"tracking": true,
		"agent":    agent,
		"task":     task,
		"timeout":  timeout,
	})
	return nil
}

func spawnTrackCheck(cmd *cobra.Command) error {
	agent := mustGetString(cmd, "agent")
	if agent == "" {
		return nil
	}

	entry, err := readSpawnTrack()
	if err != nil {
		// No tracking file = no timeout enforced
		outputOK(map[string]interface{}{
			"tracked":   false,
			"timed_out": false,
			"agent":     agent,
		})
		return nil
	}

	if entry.Agent != agent {
		outputOK(map[string]interface{}{
			"tracked":   false,
			"timed_out": false,
			"agent":     agent,
		})
		return nil
	}

	elapsed := time.Now().Unix() - entry.Start
	timedOut := false
	if entry.Timeout > 0 && int(elapsed) > entry.Timeout {
		timedOut = true
	}

	outputOK(map[string]interface{}{
		"tracked":   true,
		"agent":     agent,
		"task":      entry.Task,
		"elapsed":   elapsed,
		"timeout":   entry.Timeout,
		"timed_out": timedOut,
	})
	return nil
}

func spawnTrackClear(cmd *cobra.Command) error {
	trackFile := filepath.Join(store.BasePath(), ".aether", "data", "spawn-track.json")
	os.Remove(trackFile)
	outputOK(map[string]interface{}{"cleared": true})
	return nil
}

func writeSpawnTrack(entry spawnTrackEntry) error {
	trackFile := filepath.Join(store.BasePath(), ".aether", "data", "spawn-track.json")
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(trackFile, data, 0644)
}

func readSpawnTrack() (spawnTrackEntry, error) {
	trackFile := filepath.Join(store.BasePath(), ".aether", "data", "spawn-track.json")
	data, err := os.ReadFile(trackFile)
	if err != nil {
		return spawnTrackEntry{}, err
	}
	var entry spawnTrackEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return spawnTrackEntry{}, err
	}
	return entry, nil
}

func init() {
	spawnTrackCmd.Flags().String("action", "", "Action: start, check, or clear (required)")
	spawnTrackCmd.Flags().String("agent", "", "Agent name (required for start/check)")
	spawnTrackCmd.Flags().String("task", "", "Task ID (required for start)")
	spawnTrackCmd.Flags().Int("timeout", 0, "Timeout in seconds (0 = no timeout)")
	rootCmd.AddCommand(spawnTrackCmd)
}
