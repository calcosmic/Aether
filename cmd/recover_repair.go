package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Repair orchestrator
// ---------------------------------------------------------------------------

// performRecoverRepairs is the entry point for aether recover --apply. It creates
// a backup, then dispatches each fixable issue to the appropriate category-specific
// repair function. Destructive categories (dirty_worktree, bad_manifest) require
// confirmation unless --force is set.
func performRecoverRepairs(issues []HealthIssue, dataDir string, force bool, jsonMode bool) (*RepairResult, error) {
	// Backup before any mutations.
	backupPath, err := createBackup(dataDir)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	// Keep only 3 most recent backups.
	backupsDir := filepath.Dir(backupPath)
	_ = cleanupOldBackups(backupsDir, 3)

	// Filter to fixable issues only.
	var fixable []HealthIssue
	for _, issue := range issues {
		if issue.Fixable {
			fixable = append(fixable, issue)
		}
	}

	// Sort by severity: critical first, then warning, then info.
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

	// Deduplicate by category+message key.
	seen := make(map[string]bool)
	for _, issue := range fixable {
		key := issue.Category + ":" + issue.Message
		if seen[key] {
			continue
		}
		seen[key] = true

		// Destructive categories require confirmation (unless --force or --json).
		if isDestructiveCategory(issue.Category) && !force {
			if jsonMode {
				record := RepairRecord{
					Category: issue.Category,
					File:     issue.File,
					Action:   "skip",
					Error:    "non-interactive mode",
				}
				result.Skipped++
				result.Repairs = append(result.Repairs, record)
				logRepairToTrace(record, dataDir)
				continue
			}
			if !confirmRepair(issue) {
				record := RepairRecord{
					Category: issue.Category,
					File:     issue.File,
					Action:   "skip",
					Error:    "user declined",
				}
				result.Skipped++
				result.Repairs = append(result.Repairs, record)
				logRepairToTrace(record, dataDir)
				continue
			}
		}

		record := dispatchRecoverRepair(issue, dataDir, force)

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
		logRepairToTrace(record, dataDir)
	}

	// If any repair failed, attempt rollback from backup.
	if result.Failed > 0 {
		if rollbackErr := restoreFromBackup(backupPath, dataDir); rollbackErr != nil {
			// Log the rollback failure but don't mask the original result.
			fmt.Fprintf(os.Stderr, "  [warn] rollback failed: %v\n", rollbackErr)
		} else {
			// Rollback succeeded -- mark previously-succeeded repairs as rolled back
			// so the result accurately reflects the final state on disk.
			for i := range result.Repairs {
				if result.Repairs[i].Success {
					result.Repairs[i].Action += " (rolled back)"
					result.Succeeded--
					result.Failed++
				}
			}
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Confirmation and classification
// ---------------------------------------------------------------------------

// isDestructiveCategory returns true for repair categories that can mutate
// user data (worktree files, manifest content) in potentially irreversible ways.
func isDestructiveCategory(category string) bool {
	switch category {
	case "dirty_worktree", "bad_manifest":
		return true
	}
	return false
}

// confirmRepair prompts the user on stderr and reads a yes/no response from stdin.
func confirmRepair(issue HealthIssue) bool {
	fmt.Fprintf(os.Stderr, "\n  [confirm] %s (%s)\n", issue.Message, issue.File)
	fmt.Fprintf(os.Stderr, "  Apply fix? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	trimmed := strings.TrimSpace(strings.ToLower(response))
	return trimmed == "y" || trimmed == "yes"
}

// ---------------------------------------------------------------------------
// Rollback
// ---------------------------------------------------------------------------

// restoreFromBackup copies all files from a backup directory back into the
// data directory, overwriting whatever is there. This enables atomic rollback
// when a repair in the batch fails.
func restoreFromBackup(backupPath string, dataDir string) error {
	entries, err := os.ReadDir(backupPath)
	if err != nil {
		return fmt.Errorf("read backup dir: %w", err)
	}

	for _, entry := range entries {
		if entry.Name() == "_backup_manifest.json" {
			continue
		}

		src := filepath.Join(backupPath, entry.Name())
		dst := filepath.Join(dataDir, entry.Name())

		if entry.IsDir() {
			if err := backupCopyDir(src, dst); err != nil {
				return fmt.Errorf("restore dir %s: %w", entry.Name(), err)
			}
			continue
		}

		if err := backupCopyFile(src, dst); err != nil {
			return fmt.Errorf("restore file %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

// dispatchRecoverRepair routes an issue to the appropriate category-specific
// repair function.
func dispatchRecoverRepair(issue HealthIssue, dataDir string, force bool) RepairRecord {
	switch issue.Category {
	case "missing_build_packet":
		return repairMissingBuildPacket(issue, dataDir)
	case "stale_spawned":
		return repairStaleSpawned(issue, dataDir)
	case "partial_phase":
		return repairPartialPhase(issue, dataDir)
	case "broken_survey":
		return repairBrokenSurvey(issue, dataDir)
	case "missing_agents":
		return repairMissingAgentFiles(issue, dataDir)
	case "dirty_worktree":
		return repairDirtyWorktree(issue, dataDir, force)
	case "bad_manifest":
		return repairBadManifest(issue, dataDir)
	default:
		return RepairRecord{
			Category: issue.Category,
			File:     issue.File,
			Action:   "skip",
			Error:    "unsupported category",
		}
	}
}

// ---------------------------------------------------------------------------
// REPAIR-01: Missing Build Packet
// ---------------------------------------------------------------------------

// repairMissingBuildPacket resets colony state from EXECUTING/BUILT to READY
// and clears build_started_at. This lets the user re-run aether build.
// When the current state is EXECUTING, the repair transitions through BUILT
// first since EXECUTING -> READY is not a legal direct transition.
func repairMissingBuildPacket(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
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

	record.Before = string(state.State)

	// Transition to READY. EXECUTING requires going through BUILT first.
	switch state.State {
	case colony.StateEXECUTING:
		if err := colony.Transition(state.State, colony.StateBUILT); err != nil {
			record.Error = fmt.Sprintf("invalid transition %s -> BUILT: %v", state.State, err)
			return record
		}
		state.State = colony.StateBUILT
		fallthrough
	case colony.StateBUILT:
		if err := colony.Transition(state.State, colony.StateREADY); err != nil {
			record.Error = fmt.Sprintf("invalid transition %s -> READY: %v", state.State, err)
			return record
		}
		state.State = colony.StateREADY
	default:
		record.Error = fmt.Sprintf("unexpected state %s for missing_build_packet repair", state.State)
		return record
	}

	state.BuildStartedAt = nil

	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		record.Error = fmt.Sprintf("marshal state: %v", err)
		return record
	}
	encoded = append(encoded, '\n')
	if err := atomicWriteFile(statePath, encoded); err != nil {
		record.Error = fmt.Sprintf("write state: %v", err)
		return record
	}

	record.Action = "reset_to_ready"
	record.After = string(colony.StateREADY)
	record.Success = true
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-02: Stale Spawned Workers
// ---------------------------------------------------------------------------

// repairStaleSpawned resets spawn runs that have been active/running for more
// than 1 hour to "failed" status and clears current_run_id.
func repairStaleSpawned(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	spawnPath := filepath.Join(dataDir, "spawn-runs.json")
	data, err := os.ReadFile(spawnPath)
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
	if err := json.Unmarshal(data, &spawnState); err != nil {
		record.Error = fmt.Sprintf("parse spawn-runs: %v", err)
		return record
	}

	reset := 0
	for i := range spawnState.Runs {
		run := &spawnState.Runs[i]
		if run.Status == "running" || run.Status == "active" {
			started := parseTimestamp(run.StartedAt)
			if !started.IsZero() && time.Since(started) > time.Hour {
				run.Status = "failed"
				reset++
			}
		}
	}

	if reset > 0 {
		spawnState.CurrentRunID = ""
	}

	if reset == 0 {
		record.Error = "no stale spawn runs found"
		return record
	}

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

	record.Action = "reset_stale_spawns"
	record.After = fmt.Sprintf("reset %d stale runs to failed", reset)
	record.Success = true
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-03: Partial Phase
// ---------------------------------------------------------------------------

// repairPartialPhase handles two sub-cases:
// 1. All manifest dispatches are terminal -> transition EXECUTING -> BUILT
// 2. No manifest or incomplete dispatches -> reset phase to pending, state to READY
func repairPartialPhase(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
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

	record.Before = string(state.State)

	manifest := loadCodexContinueManifest(state.CurrentPhase)

	if manifest.Present && len(manifest.Data.Dispatches) > 0 {
		allTerminal := true
		for _, d := range manifest.Data.Dispatches {
			if d.Status != "completed" && d.Status != "failed" {
				allTerminal = false
				break
			}
		}
		if allTerminal {
			if err := colony.Transition(state.State, colony.StateBUILT); err != nil {
				record.Error = fmt.Sprintf("invalid transition %s -> BUILT: %v", state.State, err)
				return record
			}
			state.State = colony.StateBUILT
			record.Action = "transition_to_built"
			record.After = string(colony.StateBUILT)
		} else {
			// Incomplete dispatches -- reset phase to pending.
			for i := range state.Plan.Phases {
				if state.Plan.Phases[i].ID == state.CurrentPhase {
					state.Plan.Phases[i].Status = "pending"
					break
				}
			}
			if err := colony.Transition(state.State, colony.StateREADY); err != nil {
				record.Error = fmt.Sprintf("invalid transition %s -> READY: %v", state.State, err)
				return record
			}
			state.State = colony.StateREADY
			state.BuildStartedAt = nil
			record.Action = "reset_phase_to_pending"
			record.After = string(colony.StateREADY)
		}
	} else {
		// No manifest -- find the current phase and reset to pending.
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].ID == state.CurrentPhase {
				state.Plan.Phases[i].Status = "pending"
				break
			}
		}
		// Transition EXECUTING -> BUILT -> READY (EXECUTING -> READY is not legal).
		if state.State == colony.StateEXECUTING {
			if err := colony.Transition(state.State, colony.StateBUILT); err != nil {
				record.Error = fmt.Sprintf("invalid transition %s -> BUILT: %v", state.State, err)
				return record
			}
			state.State = colony.StateBUILT
		}
		if err := colony.Transition(state.State, colony.StateREADY); err != nil {
			record.Error = fmt.Sprintf("invalid transition %s -> READY: %v", state.State, err)
			return record
		}
		state.State = colony.StateREADY
		state.BuildStartedAt = nil
		record.Action = "reset_phase_to_pending"
		record.After = string(colony.StateREADY)
	}

	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		record.Error = fmt.Sprintf("marshal state: %v", err)
		return record
	}
	encoded = append(encoded, '\n')
	if err := atomicWriteFile(statePath, encoded); err != nil {
		record.Error = fmt.Sprintf("write state: %v", err)
		return record
	}

	record.Success = true
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-04: Broken Survey
// ---------------------------------------------------------------------------

// repairBrokenSurvey clears the territory_surveyed flag and removes broken
// survey files. State update happens first (least destructive), then files
// are deleted.
func repairBrokenSurvey(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
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

	state.TerritorySurveyed = nil

	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		record.Error = fmt.Sprintf("marshal state: %v", err)
		return record
	}
	encoded = append(encoded, '\n')
	if err := atomicWriteFile(statePath, encoded); err != nil {
		record.Error = fmt.Sprintf("write state: %v", err)
		return record
	}

	// Delete broken survey files.
	removed := 0
	surveyDir := filepath.Join(dataDir, "survey")
	for _, name := range surveyFiles {
		filePath := filepath.Join(surveyDir, name+".json")
		if _, err := os.Stat(filePath); err != nil {
			continue
		}
		if err := os.Remove(filePath); err == nil {
			removed++
		}
	}

	record.Action = "clear_broken_survey"
	record.After = fmt.Sprintf("cleared territory_surveyed, removed %d broken files", removed)
	record.Success = true
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-05: Missing Agent Files
// ---------------------------------------------------------------------------

// repairMissingAgentFiles copies agent definition files from the hub
// (~/.aether/system/) to the repo if they are missing locally.
func repairMissingAgentFiles(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	repoRoot := resolveAetherRoot()
	hubBase := filepath.Join(homeDir(), ".aether", "system")

	surfaces := []struct {
		hubSub  string
		destDir string
	}{
		{"claude/agents", filepath.Join(repoRoot, ".claude", "agents", "ant")},
		{"opencode/agents", filepath.Join(repoRoot, ".opencode", "agents")},
		{"codex/agents", filepath.Join(repoRoot, ".codex", "agents")},
	}

	copied := 0
	hubExists := false

	for _, surface := range surfaces {
		hubDir := filepath.Join(hubBase, surface.hubSub)
		entries, err := os.ReadDir(hubDir)
		if err != nil {
			continue
		}
		hubExists = true

		if err := os.MkdirAll(surface.destDir, 0755); err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(hubDir, entry.Name())
			dst := filepath.Join(surface.destDir, entry.Name())

			if _, err := os.Stat(dst); err == nil {
				continue // already exists
			}

			content, err := os.ReadFile(src)
			if err != nil {
				continue
			}
			if err := os.WriteFile(dst, content, 0644); err != nil {
				continue
			}
			copied++
		}
	}

	if copied == 0 && !hubExists {
		record.Error = "hub has no agent files -- run `aether update` first"
		return record
	}

	record.Action = "restore_agent_files"
	record.After = fmt.Sprintf("restored %d agent files from hub", copied)
	record.Success = copied > 0
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-06: Dirty Worktree (destructive -- requires confirmation)
// ---------------------------------------------------------------------------

// repairDirtyWorktree handles three sub-types based on issue.Message content:
// 1. state-disk mismatch / not in git worktree list -> remove orphan entry
// 2. uncommitted change -> git stash
// 3. Orphan branch -> git branch -D
func repairDirtyWorktree(issue HealthIssue, dataDir string, force bool) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
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

	msg := issue.Message

	switch {
	case strings.Contains(msg, "state-disk mismatch") || strings.Contains(msg, "not in git worktree list"):
		var remaining []colony.WorktreeEntry
		for _, wt := range state.Worktrees {
			if wt.Path != issue.File {
				remaining = append(remaining, wt)
			}
		}
		state.Worktrees = remaining
		record.Action = "remove_orphan_worktree_entry"

	case strings.Contains(msg, "uncommitted change"):
		cmd := exec.Command("git", "-C", issue.File, "stash", "--include-untracked")
		if output, err := cmd.CombinedOutput(); err != nil {
			record.Error = fmt.Sprintf("git stash failed: %v: %s", err, string(output))
			return record
		}
		record.Action = "stash_worktree_changes"

	case strings.Contains(msg, "Orphan branch"):
		branchName := issue.File
		cmd := exec.Command("git", "branch", "-D", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			record.Error = fmt.Sprintf("git branch -D failed: %v: %s", err, string(output))
			return record
		}
		record.Action = "delete_orphan_branch"

	default:
		record.Error = "unrecognized dirty_worktree sub-type"
		return record
	}

	encoded, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		record.Error = fmt.Sprintf("marshal state: %v", err)
		return record
	}
	encoded = append(encoded, '\n')
	if err := atomicWriteFile(statePath, encoded); err != nil {
		record.Error = fmt.Sprintf("write state: %v", err)
		return record
	}

	record.Success = true
	return record
}

// ---------------------------------------------------------------------------
// REPAIR-07: Bad Manifest (destructive -- requires confirmation)
// ---------------------------------------------------------------------------

// repairBadManifest handles two sub-types based on issue.Message content:
// 1. "parse failed" -> attempt JSON recovery or remove corrupt file
// 2. "mismatch" or "empty" -> fix specific manifest fields
func repairBadManifest(issue HealthIssue, dataDir string) RepairRecord {
	record := RepairRecord{
		Category: issue.Category,
		File:     issue.File,
	}

	filePath := filepath.Join(dataDir, issue.File)
	raw, err := os.ReadFile(filePath)
	if err != nil {
		record.Error = fmt.Sprintf("read manifest: %v", err)
		return record
	}

	msg := issue.Message

	switch {
	case strings.Contains(msg, "parse failed"):
		recovered := findLastValidJSON(raw)
		if recovered == nil {
			// Cannot recover -- remove the corrupt file.
			if err := os.Remove(filePath); err != nil {
				record.Error = fmt.Sprintf("remove corrupt manifest: %v", err)
				return record
			}
			record.Action = "remove_corrupt_manifest"
			record.Success = true
			return record
		}
		// Validate recovered bytes.
		if !json.Valid(recovered) {
			record.Error = "recovered JSON is not valid"
			return record
		}
		if err := atomicWriteFile(filePath, recovered); err != nil {
			record.Error = fmt.Sprintf("write recovered manifest: %v", err)
			return record
		}
		record.Action = "recover_manifest_json"
		record.Success = true
		return record

	case strings.Contains(msg, "mismatch") || strings.Contains(msg, "empty"):
		// Parse the manifest to fix fields.
		var manifest codexBuildManifest
		if err := json.Unmarshal(raw, &manifest); err != nil {
			// If we can't parse, try recovery first.
			recovered := findLastValidJSON(raw)
			if recovered != nil && json.Valid(recovered) {
				if err := atomicWriteFile(filePath, recovered); err != nil {
					record.Error = fmt.Sprintf("write recovered: %v", err)
					return record
				}
				raw = recovered
				if err := json.Unmarshal(raw, &manifest); err != nil {
					record.Error = fmt.Sprintf("parse recovered manifest: %v", err)
					return record
				}
			} else {
				record.Error = fmt.Sprintf("parse manifest: %v", err)
				return record
			}
		}

		// Load colony state for current phase.
		statePath := filepath.Join(dataDir, "COLONY_STATE.json")
		stateData, err := os.ReadFile(statePath)
		if err != nil {
			record.Error = fmt.Sprintf("read state: %v", err)
			return record
		}
		var state colony.ColonyState
		if err := json.Unmarshal(stateData, &state); err != nil {
			record.Error = fmt.Sprintf("parse state: %v", err)
			return record
		}

		// Fix fields.
		manifest.Phase = state.CurrentPhase
		manifest.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
		manifest.State = "executing"

		encoded, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			record.Error = fmt.Sprintf("marshal manifest: %v", err)
			return record
		}
		encoded = append(encoded, '\n')
		if err := atomicWriteFile(filePath, encoded); err != nil {
			record.Error = fmt.Sprintf("write manifest: %v", err)
			return record
		}

		record.Action = "repair_manifest_fields"
		record.Success = true
		return record

	default:
		record.Error = "unrecognized bad_manifest sub-type"
		return record
	}
}
