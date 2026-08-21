package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// createAutofixCheckpoint snapshots COLONY_STATE.json before a risky
// auto-repair (RECLAIM-01). Called in-process by medic's repair cycle and by
// the `autofix-checkpoint` command. Returns the checkpoint name and its
// store-relative path.
func createAutofixCheckpoint(issue string) (string, string, error) {
	if store == nil {
		return "", "", fmt.Errorf("no store initialized")
	}
	data, err := store.ReadFile("COLONY_STATE.json")
	if err != nil {
		return "", "", fmt.Errorf("COLONY_STATE.json not found")
	}

	timestamp := time.Now().UTC().Format("20060102T150405Z")
	checkpointName := fmt.Sprintf("autofix-%s", timestamp)
	checkpointPath := filepath.Join("checkpoints", checkpointName+".json")

	checkpoint := map[string]interface{}{
		"checkpoint": checkpointName,
		"issue":      issue,
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"data":       string(data),
	}
	if err := store.SaveJSON(checkpointPath, checkpoint); err != nil {
		return "", "", fmt.Errorf("failed to create checkpoint: %v", err)
	}
	return checkpointName, checkpointPath, nil
}

var autofixCheckpointCmd = &cobra.Command{
	Use:   "autofix-checkpoint",
	Short: "Create a checkpoint before autofix attempt",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		issue := mustGetString(cmd, "issue")
		if issue == "" {
			return nil
		}
		checkpointName, checkpointPath, err := createAutofixCheckpoint(issue)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputOK(map[string]interface{}{
			"checkpoint": checkpointName,
			"path":       checkpointPath,
			"issue":      issue,
		})
		return nil
	},
}

var autofixRollbackCmd = &cobra.Command{
	Use:   "autofix-rollback",
	Short: "Rollback to a checkpoint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		checkpointID := mustGetString(cmd, "checkpoint-id")
		if checkpointID == "" {
			return nil
		}

		// Read checkpoint file
		checkpointPath := filepath.Join("checkpoints", checkpointID+".json")
		data, err := store.ReadFile(checkpointPath)
		if err != nil {
			outputError(1, fmt.Sprintf("checkpoint %q not found", checkpointID), nil)
			return nil
		}

		// The checkpoint is an envelope; the colony state is its "data"
		// field. Writing the raw file would overwrite COLONY_STATE.json with
		// the envelope itself — the exact corruption this command exists to
		// undo. That bug shipped and was never caught because the command had
		// no caller until the reclaim sweep wired it.
		var checkpoint struct {
			Data string `json:"data"`
		}
		restore := data
		if err := json.Unmarshal(data, &checkpoint); err == nil && strings.TrimSpace(checkpoint.Data) != "" {
			restore = []byte(checkpoint.Data)
		}
		var sanity map[string]interface{}
		if err := json.Unmarshal(restore, &sanity); err != nil {
			outputError(2, fmt.Sprintf("checkpoint %q does not contain restorable colony state: %v", checkpointID, err), nil)
			return nil
		}

		if err := store.AtomicWrite("COLONY_STATE.json", restore); err != nil {
			outputError(2, fmt.Sprintf("failed to rollback: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"rolled_back": true,
			"checkpoint":  checkpointID,
		})
		return nil
	},
}

func init() {
	autofixCheckpointCmd.Flags().String("issue", "", "Description of the issue being fixed (required)")

	autofixRollbackCmd.Flags().String("checkpoint-id", "", "Checkpoint name to rollback to (required)")

	rootCmd.AddCommand(autofixCheckpointCmd)
	rootCmd.AddCommand(autofixRollbackCmd)
}
