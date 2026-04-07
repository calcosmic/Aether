package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxAutoCheckpoints = 10

// AutoCheckpoint creates a timestamped checkpoint of the current state before
// a destructive operation. The checkpoint is stored in checkpoints/auto-<timestamp>.json.
// After creating the checkpoint, it prunes old auto-checkpoints to keep at most
// maxAutoCheckpoints (10). Manual checkpoints (files without "auto-" prefix) are
// never deleted.
func AutoCheckpoint(s *Store, beforeState []byte) error {
	timestamp := time.Now().UTC().Format("20060102-150405")
	checkpointPath := fmt.Sprintf("checkpoints/auto-%s.json", timestamp)

	if err := s.AtomicWrite(checkpointPath, beforeState); err != nil {
		return fmt.Errorf("auto-checkpoint: %w", err)
	}

	return pruneAutoCheckpoints(s, maxAutoCheckpoints)
}

// pruneAutoCheckpoints removes old auto-checkpoints, keeping only the last maxKeep.
// Manual checkpoints (files not prefixed with "auto-") are never deleted.
func pruneAutoCheckpoints(s *Store, maxKeep int) error {
	checkpointsDir := filepath.Join(s.BasePath(), "checkpoints")

	entries, err := os.ReadDir(checkpointsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("prune checkpoints: %w", err)
	}

	var autoCheckpoints []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "auto-") && strings.HasSuffix(e.Name(), ".json") {
			autoCheckpoints = append(autoCheckpoints, e.Name())
		}
	}

	if len(autoCheckpoints) <= maxKeep {
		return nil
	}

	// Sort by name (which sorts by timestamp since format is deterministic)
	sort.Strings(autoCheckpoints)

	// Delete oldest auto-checkpoints, keeping the last maxKeep
	toDelete := autoCheckpoints[:len(autoCheckpoints)-maxKeep]
	for _, name := range toDelete {
		path := filepath.Join(checkpointsDir, name)
		if err := os.Remove(path); err != nil {
			// Log but don't fail -- pruning is best-effort
			fmt.Fprintf(os.Stderr, "checkpoint: failed to remove old checkpoint %q: %v\n", name, err)
		}
	}

	return nil
}
