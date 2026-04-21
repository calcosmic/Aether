package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/trace"
)

// RepairResult holds the outcome of a full repair cycle.
type RepairResult struct {
	Attempted int
	Succeeded int
	Failed    int
	Skipped   int
	Repairs   []RepairRecord
}

// RepairRecord documents a single repair attempt.
type RepairRecord struct {
	Category string
	File     string
	Action   string
	Before   string // truncated snippet
	After    string // truncated snippet
	Success  bool
	Error    string // empty if success
}

// severityOrder maps severity to sort priority (lower = repair first).
var severityOrder = map[string]int{
	"critical": 0,
	"warning":  1,
	"info":     2,
}

// createBackup copies all files from .aether/data/ into a timestamped backup directory.
// Returns the backup path or an error.
func createBackup(dataPath string) (string, error) {
	backupsDir := filepath.Join(filepath.Dir(dataPath), "..", "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(backupsDir, "medic-"+timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("create backup subdir: %w", err)
	}

	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return "", fmt.Errorf("read data dir: %w", err)
	}

	copied := 0
	for _, entry := range entries {
		src := filepath.Join(dataPath, entry.Name())
		dst := filepath.Join(backupPath, entry.Name())

		if entry.IsDir() {
			if err := backupCopyDir(src, dst); err != nil {
				return "", fmt.Errorf("copy dir %s: %w", entry.Name(), err)
			}
			copied++
			continue
		}

		if err := backupCopyFile(src, dst); err != nil {
			return "", fmt.Errorf("copy %s: %w", entry.Name(), err)
		}
		copied++
	}

	// Write a manifest so we know what was backed up
	manifest := map[string]interface{}{
		"timestamp":     timestamp,
		"files_copied":  copied,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"source_path":   dataPath,
	}
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	manifestData = append(manifestData, '\n')
	_ = os.WriteFile(filepath.Join(backupPath, "_backup_manifest.json"), manifestData, 0644)

	return backupPath, nil
}

// cleanupOldBackups removes all but the most recent `keep` medic backup directories.
func cleanupOldBackups(backupsDir string, keep int) error {
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("list backups dir: %w", err)
	}

	// Filter to only medic-* directories
	var medicDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "medic-") {
			medicDirs = append(medicDirs, entry.Name())
		}
	}

	if len(medicDirs) <= keep {
		return nil
	}

	// Sort by name (timestamp is embedded in name, so lexical sort = chronological)
	sort.Strings(medicDirs)

	// Remove oldest directories (keep the last `keep`)
	for i := 0; i < len(medicDirs)-keep; i++ {
		path := filepath.Join(backupsDir, medicDirs[i])
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove old backup %s: %w", medicDirs[i], err)
		}
	}

	return nil
}

// logRepairToTrace appends a trace entry for a repair operation to trace.jsonl.
func logRepairToTrace(record RepairRecord, dataPath string) {
	b := make([]byte, 4)
	rand.Read(b)
	entryID := fmt.Sprintf("medic_%d_%s", time.Now().Unix(), hex.EncodeToString(b))

	payload := map[string]interface{}{
		"action":  record.Action,
		"file":    record.File,
		"success": record.Success,
	}
	if record.Before != "" {
		payload["before"] = record.Before
	}
	if record.After != "" {
		payload["after"] = record.After
	}
	if record.Error != "" {
		payload["error"] = record.Error
	}

	entry := trace.TraceEntry{
		ID:        entryID,
		RunID:     "medic",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     trace.TraceLevelIntervention,
		Topic:     "medic.repair",
		Payload:   payload,
		Source:    "medic",
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return
	}

	tracePath := filepath.Join(dataPath, "trace.jsonl")
	f, err := os.OpenFile(tracePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.Write(append(line, '\n'))
}

// performRepairs orchestrates the full repair cycle: backup, filter, sort, dispatch, trace.
func performRepairs(scanResult *ScannerResult, opts MedicOptions, dataPath string) (*RepairResult, error) {
	if !opts.Fix {
		return nil, nil
	}

	// Create backup before any modifications
	backupPath, err := createBackup(dataPath)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	// Cleanup old backups, keeping most recent 3
	backupsDir := filepath.Dir(backupPath)
	_ = cleanupOldBackups(backupsDir, 3)

	// Filter to fixable issues only
	var fixable []HealthIssue
	for _, issue := range scanResult.Issues {
		if issue.Fixable {
			fixable = append(fixable, issue)
		}
	}

	// Sort by severity: critical first, then warning, then info
	sort.SliceStable(fixable, func(i, j int) bool {
		pi, ok := severityOrder[fixable[i].Severity]
		if !ok {
			pi = 99
		}
		pj, ok := severityOrder[fixable[j].Severity]
		if !ok {
			pj = 99
		}
		return pi < pj
	})

	result := &RepairResult{
		Attempted: len(fixable),
	}

	// Deduplicate by category+message to avoid repairing the same issue twice
	seen := make(map[string]bool)
	for _, issue := range fixable {
		key := issue.Category + ":" + issue.Message
		if seen[key] {
			continue
		}
		seen[key] = true

		var record RepairRecord
		switch issue.Category {
		case "state":
			record = repairStateIssues(issue, opts, dataPath)
		case "pheromone":
			record = repairPheromoneIssues(issue, opts, dataPath)
		case "session":
			record = repairSessionIssues(issue, opts, dataPath)
		case "data":
			record = repairDataIssues(issue, opts, dataPath)
		default:
			record = RepairRecord{
				Category: issue.Category,
				File:     issue.File,
				Action:   "skip",
				Error:    "unsupported category",
			}
			result.Skipped++
		}

		if record.Category == "" {
			record.Category = issue.Category
		}
		if record.File == "" {
			record.File = issue.File
		}

		if record.Error != "" && !record.Success {
			result.Failed++
		} else if record.Success {
			result.Succeeded++
		} else {
			result.Skipped++
		}

		result.Repairs = append(result.Repairs, record)
		logRepairToTrace(record, dataPath)
	}

	return result, nil
}

// repairStateIssues handles repairs for colony state issues.
func repairStateIssues(issue HealthIssue, opts MedicOptions, dataPath string) RepairRecord {
	record := RepairRecord{
		Category: "state",
		File:     issue.File,
	}

	statePath := filepath.Join(dataPath, "COLONY_STATE.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		record.Error = fmt.Sprintf("read state: %v", err)
		return record
	}

	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		record.Error = fmt.Sprintf("parse state: %v", err)
		return record
	}

	record.Before = repairTruncate(string(data), 200)

	switch {
	case strings.Contains(issue.Message, "orphaned"):
		record.Action = "remove_orphaned_worktrees"
		beforeCount := len(state.Worktrees)

		var remaining []colony.WorktreeEntry
		for _, wt := range state.Worktrees {
			if wt.Status != colony.WorktreeOrphaned {
				remaining = append(remaining, wt)
			}
		}
		state.Worktrees = remaining

		afterCount := len(state.Worktrees)
		record.After = fmt.Sprintf("%d worktree entries (removed %d orphaned)", afterCount, beforeCount-afterCount)
		record.Success = beforeCount > afterCount

	case strings.Contains(issue.Message, "deprecated") && strings.Contains(issue.Message, "signals"):
		record.Action = "migrate_deprecated_signals"
		migrated := len(state.Signals)
		state.Signals = nil
		record.After = fmt.Sprintf("cleared %d deprecated signals", migrated)
		record.Success = migrated > 0

	case strings.Contains(issue.Message, "EXECUTING"):
		record.Action = "reset_executing_no_phase"
		record.Before = string(state.State)
		state.State = colony.StateREADY
		record.After = string(state.State)
		record.Success = true

	default:
		if strings.Contains(issue.Message, "deprecated") {
			// Legacy state normalization
			record.Action = "normalize_legacy_state"
			normalized := normalizeLegacyColonyState(state)
			state = normalized
			record.After = string(state.State)
			record.Success = true
		} else {
			record.Action = "skip"
			record.Error = "unrecognized state issue"
			return record
		}
	}

	// Save modified state via AtomicWrite
	if record.Success {
		encoded, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("marshal state: %v", err)
			return record
		}
		encoded = append(encoded, '\n')
		if err := atomicWriteFile(statePath, encoded); err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("write state: %v", err)
		}
	}

	return record
}

// repairPheromoneIssues handles repairs for pheromone file issues.
func repairPheromoneIssues(issue HealthIssue, opts MedicOptions, dataPath string) RepairRecord {
	record := RepairRecord{
		Category: "pheromone",
		File:     issue.File,
	}

	pheromonesPath := filepath.Join(dataPath, "pheromones.json")
	data, err := os.ReadFile(pheromonesPath)
	if err != nil {
		record.Error = fmt.Sprintf("read pheromones: %v", err)
		return record
	}

	var pheromones colony.PheromoneFile
	if err := json.Unmarshal(data, &pheromones); err != nil {
		record.Error = fmt.Sprintf("parse pheromones: %v", err)
		return record
	}

	record.Before = repairTruncate(string(data), 200)
	now := time.Now()
	modified := false

	switch {
	case strings.Contains(issue.Message, "expired") && strings.Contains(issue.Message, "active"):
		record.Action = "deactivate_expired_signals"
		deactivated := 0
		archivedAt := now.UTC().Format(time.RFC3339)
		for i := range pheromones.Signals {
			sig := &pheromones.Signals[i]
			if sig.Active && sig.ExpiresAt != nil && *sig.ExpiresAt != "" {
				expiresAt := parseTimestamp(*sig.ExpiresAt)
				if !expiresAt.IsZero() && now.After(expiresAt) {
					sig.Active = false
					sig.ArchivedAt = &archivedAt
					deactivated++
				}
			}
		}
		record.After = fmt.Sprintf("deactivated %d expired signals", deactivated)
		record.Success = deactivated > 0
		modified = record.Success

	case strings.Contains(issue.Message, "missing ID"):
		record.Action = "assign_missing_ids"
		assigned := 0
		for i := range pheromones.Signals {
			if pheromones.Signals[i].ID == "" {
				b := make([]byte, 2)
				rand.Read(b)
				pheromones.Signals[i].ID = fmt.Sprintf("sig_%d_%s", now.Unix(), hex.EncodeToString(b))
				assigned++
			}
		}
		record.After = fmt.Sprintf("assigned %d missing IDs", assigned)
		record.Success = assigned > 0
		modified = record.Success

	case strings.Contains(issue.Message, "Invalid signal type"):
		record.Action = "fix_invalid_signal_types"
		fixed := 0
		for i := range pheromones.Signals {
			sig := &pheromones.Signals[i]
			validTypes := map[string]bool{"FOCUS": true, "REDIRECT": true, "FEEDBACK": true}
			if !validTypes[sig.Type] && sig.Type != "" {
				sig.Type = "FOCUS"
				fixed++
			}
		}
		record.After = fmt.Sprintf("fixed %d invalid types to FOCUS", fixed)
		record.Success = fixed > 0
		modified = record.Success

	default:
		record.Action = "skip"
		record.Error = "unrecognized pheromone issue"
		return record
	}

	// Save via AtomicWrite
	if modified && record.Success {
		encoded, err := json.MarshalIndent(pheromones, "", "  ")
		if err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("marshal pheromones: %v", err)
			return record
		}
		encoded = append(encoded, '\n')
		if err := atomicWriteFile(pheromonesPath, encoded); err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("write pheromones: %v", err)
		}
	}

	return record
}

// repairSessionIssues handles repairs for session file issues.
func repairSessionIssues(issue HealthIssue, opts MedicOptions, dataPath string) RepairRecord {
	record := RepairRecord{
		Category: "session",
		File:     issue.File,
	}

	sessionPath := filepath.Join(dataPath, "session.json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		record.Error = fmt.Sprintf("read session: %v", err)
		return record
	}

	var session colony.SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		record.Error = fmt.Sprintf("parse session: %v", err)
		return record
	}

	record.Before = repairTruncate(string(data), 200)

	// Load colony state for cross-reference
	statePath := filepath.Join(dataPath, "COLONY_STATE.json")
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		record.Error = fmt.Sprintf("read state for cross-ref: %v", err)
		return record
	}

	var state colony.ColonyState
	if err := json.Unmarshal(stateData, &state); err != nil {
		record.Error = fmt.Sprintf("parse state for cross-ref: %v", err)
		return record
	}

	modified := false

	switch {
	case strings.Contains(issue.Message, "current_phase") && strings.Contains(issue.Message, "doesn't match"):
		record.Action = "fix_phase_mismatch"
		record.Before = fmt.Sprintf("phase=%d", session.CurrentPhase)
		session.CurrentPhase = state.CurrentPhase
		record.After = fmt.Sprintf("phase=%d", session.CurrentPhase)
		record.Success = true
		modified = true

	case strings.Contains(issue.Message, "goal") && strings.Contains(issue.Message, "doesn't match"):
		record.Action = "fix_goal_mismatch"
		record.Before = session.ColonyGoal
		if state.Goal != nil {
			session.ColonyGoal = *state.Goal
		}
		record.After = session.ColonyGoal
		record.Success = true
		modified = true

	default:
		record.Action = "skip"
		record.Error = "unrecognized session issue"
		return record
	}

	// Save via AtomicWrite
	if modified && record.Success {
		encoded, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("marshal session: %v", err)
			return record
		}
		encoded = append(encoded, '\n')
		if err := atomicWriteFile(sessionPath, encoded); err != nil {
			record.Success = false
			record.Error = fmt.Sprintf("write session: %v", err)
		}
	}

	return record
}

// repairDataIssues handles repairs for data file issues.
func repairDataIssues(issue HealthIssue, opts MedicOptions, dataPath string) RepairRecord {
	record := RepairRecord{
		Category: "data",
		File:     issue.File,
	}

	switch {
	case strings.Contains(issue.Message, "corrupted"):
		record.Action = "recover_corrupted_json"
		if !opts.Force {
			record.Error = "requires --force for destructive repair"
			return record
		}

		filePath := filepath.Join(dataPath, issue.File)
		raw, err := os.ReadFile(filePath)
		if err != nil {
			record.Error = fmt.Sprintf("read file: %v", err)
			return record
		}

		record.Before = repairTruncate(string(raw), 200)

		// Attempt to find last valid JSON closing
		recovered := findLastValidJSON(raw)
		if recovered == nil {
			record.Error = "could not recover valid JSON"
			return record
		}

		// Verify the recovered bytes are valid JSON
		if !json.Valid(recovered) {
			record.Error = "recovered content is not valid JSON"
			return record
		}

		if err := atomicWriteFile(filePath, recovered); err != nil {
			record.Error = fmt.Sprintf("write recovered: %v", err)
			return record
		}

		record.After = repairTruncate(string(recovered), 200)
		record.Success = true

	case strings.Contains(issue.Message, "ghost"):
		record.Action = "reset_ghost_constraints"
		constraintsPath := filepath.Join(dataPath, "constraints.json")
		if err := atomicWriteFile(constraintsPath, []byte("{}\n")); err != nil {
			record.Error = fmt.Sprintf("write constraints: %v", err)
			return record
		}
		record.Before = "non-empty"
		record.After = "{}"
		record.Success = true

	case strings.Contains(issue.Message, "cache") || strings.Contains(issue.Message, "index"):
		record.Action = "clear_stale_cache"
		cleared := 0
		for _, cacheFile := range []string{".cache_COLONY_STATE.json", ".cache_instincts.json"} {
			path := filepath.Join(dataPath, cacheFile)
			if err := os.Remove(path); err == nil {
				cleared++
			}
		}
		record.After = fmt.Sprintf("cleared %d cache files", cleared)
		record.Success = cleared > 0

	case strings.Contains(issue.Message, "spawn") && strings.Contains(issue.Message, "stale"):
		record.Action = "reset_stale_spawn_state"
		spawnPath := filepath.Join(dataPath, "spawn-runs.json")
		raw, err := os.ReadFile(spawnPath)
		if err != nil {
			record.Error = fmt.Sprintf("read spawn-runs: %v", err)
			return record
		}

		var spawnState struct {
			CurrentRunID string `json:"current_run_id"`
			Runs         []struct {
				ID        string `json:"id"`
				StartedAt string `json:"started_at"`
				Status    string `json:"status"`
			} `json:"runs"`
		}
		if err := json.Unmarshal(raw, &spawnState); err != nil {
			record.Error = fmt.Sprintf("parse spawn-runs: %v", err)
			return record
		}

		reset := false
		for i := range spawnState.Runs {
			run := &spawnState.Runs[i]
			if run.Status == "running" || run.Status == "active" {
				started := parseTimestamp(run.StartedAt)
				if !started.IsZero() && time.Since(started) > time.Hour {
					run.Status = "failed"
					reset = true
				}
			}
		}

		if reset {
			spawnState.CurrentRunID = ""
			encoded, err := json.MarshalIndent(spawnState, "", "  ")
			if err != nil {
				record.Error = fmt.Sprintf("marshal spawn-runs: %v", err)
				return record
			}
			encoded = append(encoded, '\n')
			if err := atomicWriteFile(spawnPath, encoded); err != nil {
				record.Error = fmt.Sprintf("write spawn-runs: %v", err)
				return record
			}
			record.Success = true
			record.After = "stale runs reset to failed"
		} else {
			record.Success = false
			record.Error = "no stale spawn runs found"
		}

	default:
		record.Action = "skip"
		record.Error = "unrecognized data issue"
		return record
	}

	return record
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// atomicWriteFile writes data atomically using temp file + rename.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}

	b := make([]byte, 4)
	rand.Read(b)
	tmpPath := fmt.Sprintf("%s.tmp.%d-%s", path, os.Getpid(), hex.EncodeToString(b))

	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp %q: %w", tmpPath, err)
	}

	// Clean up on failure
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %q -> %q: %w", tmpPath, path, err)
	}

	success = true
	return nil
}

// backupCopyFile copies a single file from src to dst.
func backupCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// backupCopyDir recursively copies a directory from src to dst.
func backupCopyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := backupCopyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := backupCopyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// findLastValidJSON attempts to find the last valid JSON closing in raw bytes.
func findLastValidJSON(raw []byte) []byte {
	// Try to find the last valid closing brace or bracket
	// Scan backwards for } or ] and attempt to parse progressively
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == '}' || raw[i] == ']' {
			candidate := strings.TrimSpace(string(raw[:i+1]))
			if json.Valid([]byte(candidate)) {
				return []byte(candidate + "\n")
			}
		}
	}
	return nil
}

// repairTruncate truncates a string to maxLen characters for repair snippets.
func repairTruncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
