package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	heartbeatFilePrefix            = "heartbeat-"
	heartbeatScanInterval          = 15 * time.Second
	heartbeatStaleWarningThreshold = 90 * time.Second
	heartbeatCleanupPattern        = "heartbeat-*.json"
)

// HeartbeatFile represents a worker's liveness signal written to disk.
// Workers write this file every ~30 seconds while active.
type HeartbeatFile struct {
	WorkerID  string `json:"worker_id"`
	Caste     string `json:"caste"`
	Timestamp string `json:"timestamp"` // RFC3339
	Phase     int    `json:"phase"`
}

// StartHeartbeatMonitor starts a background goroutine that scans heartbeat
// files at a regular interval and emits visual warnings for stale workers.
// Returns a cancel function that stops the goroutine.
func StartHeartbeatMonitor(ctx context.Context, dataDir string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(heartbeatScanInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				scanHeartbeatFiles(dataDir)
			}
		}
	}()

	return cancel
}

// scanHeartbeatFiles reads the data directory for heartbeat files and emits
// visual warnings for any that are stale (timestamp older than the threshold).
// Malformed JSON files are skipped silently.
func scanHeartbeatFiles(dataDir string) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := filepath.Base(entry.Name())
		if !strings.HasPrefix(name, heartbeatFilePrefix) {
			continue
		}

		path := filepath.Join(dataDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var hf HeartbeatFile
		if err := json.Unmarshal(data, &hf); err != nil {
			// Skip malformed JSON silently
			continue
		}

		ts, err := time.Parse(time.RFC3339, hf.Timestamp)
		if err != nil {
			continue
		}

		elapsed := time.Since(ts)
		if elapsed > heartbeatStaleWarningThreshold {
			emitVisualProgress(fmt.Sprintf(
				"Heartbeat stale for worker %s (last seen %s ago)",
				hf.WorkerID,
				formatDuration(elapsed),
			))
		}
	}
}

// cleanupHeartbeatFiles removes the heartbeat file for a specific worker.
// Returns nil if the file does not exist.
func cleanupHeartbeatFiles(dataDir, workerID string) error {
	path := filepath.Join(dataDir, heartbeatFilePrefix+workerID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove heartbeat file for worker %s: %w", workerID, err)
	}
	return nil
}

// cleanupAllHeartbeatFiles removes all heartbeat files in the data directory.
func cleanupAllHeartbeatFiles(dataDir string) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := filepath.Base(entry.Name())
		if strings.HasPrefix(name, heartbeatFilePrefix) && strings.HasSuffix(name, ".json") {
			_ = os.Remove(filepath.Join(dataDir, name))
		}
	}
}

// formatDuration returns a human-readable duration string (e.g., "2m30s").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) - minutes*60
	return fmt.Sprintf("%dm%ds", minutes, seconds)
}
