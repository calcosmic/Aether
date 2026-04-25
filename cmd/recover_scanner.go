package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// performStuckStateScan runs all 7 stuck-state detectors in dependency order
// and returns the collected issues.
func performStuckStateScan(dataDir string) ([]HealthIssue, error) {
	state, stateErr := loadActiveColonyState()
	if stateErr != nil {
		return []HealthIssue{{
			Severity: "critical",
			Category: "state",
			Message:  colonyStateLoadMessage(stateErr),
			Fixable:  false,
		}}, nil
	}

	var issues []HealthIssue

	// Detection order matters: stale workers first (independent), then bad manifest
	// (validates manifest), then missing build packet (depends on manifest state),
	// then partial phase (depends on manifest being valid), then independent checks.
	issues = append(issues, scanStaleSpawnedWorkers(dataDir)...)
	issues = append(issues, scanBadManifest(state, dataDir)...)
	issues = append(issues, scanMissingBuildPacket(state, dataDir)...)
	issues = append(issues, scanPartialPhase(state, dataDir)...)
	issues = append(issues, scanDirtyWorktrees(state)...)
	issues = append(issues, scanBrokenSurvey(state, dataDir)...)
	issues = append(issues, scanMissingAgentFiles()...)

	return issues, nil
}

// ---------------------------------------------------------------------------
// DETECT-01: Missing Build Packet
// ---------------------------------------------------------------------------

// scanMissingBuildPacket detects when the colony is in EXECUTING or BUILT state
// but no valid build manifest exists for the current phase.
func scanMissingBuildPacket(state colony.ColonyState, dataDir string) []HealthIssue {
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return nil
	}
	if state.CurrentPhase < 1 {
		return nil
	}

	manifest := loadCodexContinueManifest(state.CurrentPhase)
	if !manifest.Present || manifest.Data.PlanOnly || len(manifest.Data.Dispatches) == 0 {
		return []HealthIssue{
			fixableIssue(issueCritical("missing_build_packet",
				fmt.Sprintf("build/phase-%d/manifest.json", state.CurrentPhase),
				fmt.Sprintf("No build packet for phase %d (state=%s)", state.CurrentPhase, state.State))),
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// DETECT-02: Stale Spawned Workers
// ---------------------------------------------------------------------------

// scanStaleSpawnedWorkers detects spawn runs that have been active or running
// for longer than 1 hour, indicating stuck workers.
func scanStaleSpawnedWorkers(dataDir string) []HealthIssue {
	spawnPath := filepath.Join(dataDir, "spawn-runs.json")
	raw, err := os.ReadFile(spawnPath)
	if err != nil {
		// No spawn-runs.json is fine -- no spawn tracking data.
		return nil
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
		return nil
	}

	staleCount := 0
	for _, run := range spawnState.Runs {
		if run.Status != "active" && run.Status != "running" {
			continue
		}
		started := parseTimestamp(run.StartedAt)
		if !started.IsZero() && time.Since(started) > time.Hour {
			staleCount++
		}
	}

	if staleCount > 0 {
		return []HealthIssue{
			fixableIssue(issueCritical("stale_spawned", "spawn-runs.json",
				fmt.Sprintf("%d spawned worker(s) exceeded 1-hour threshold", staleCount))),
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// DETECT-03: Partial Phase
// ---------------------------------------------------------------------------

// scanPartialPhase detects when a phase build completed but continue was never run,
// or when a phase is marked in_progress but was never actually built.
func scanPartialPhase(state colony.ColonyState, dataDir string) []HealthIssue {
	if state.State != colony.StateEXECUTING {
		return nil
	}
	if state.CurrentPhase < 1 {
		return nil
	}

	var issues []HealthIssue

	// Check if manifest has all-terminal dispatches but continue was not run.
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
			continuePath := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", state.CurrentPhase), "continue.json")
			if _, err := os.Stat(continuePath); os.IsNotExist(err) {
				issues = append(issues, fixableIssue(issueWarning("partial_phase",
					fmt.Sprintf("build/phase-%d/continue.json", state.CurrentPhase),
					"Build completed but continue not run")))
			}
		}
	}

	// Check if phase is marked in_progress but never built (no manifest, no build_started_at).
	if !manifest.Present {
		// Find the current phase in the plan to check its status.
		for _, p := range state.Plan.Phases {
			if p.ID == state.CurrentPhase {
				if p.Status == "in_progress" {
					issues = append(issues, fixableIssue(issueWarning("partial_phase",
						"COLONY_STATE.json",
						fmt.Sprintf("Phase %d marked in_progress but never built", state.CurrentPhase))))
				}
				break
			}
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// DETECT-04: Bad Manifest
// ---------------------------------------------------------------------------

// scanBadManifest detects corrupted, invalid, or inconsistent build manifests.
func scanBadManifest(state colony.ColonyState, dataDir string) []HealthIssue {
	if state.CurrentPhase < 1 {
		return nil
	}

	manifestPath := filepath.Join(dataDir, "build", fmt.Sprintf("phase-%d", state.CurrentPhase), "manifest.json")

	// Check if the manifest file exists on disk.
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		// Missing manifest is not a bad manifest -- it is the missing build packet
		// detector's job.
		return nil
	}

	// Read the raw file.
	raw, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		return []HealthIssue{
			issueCritical("bad_manifest", manifestPath,
				fmt.Sprintf("Cannot read manifest: %v", readErr)),
		}
	}

	// Try to parse as JSON.
	var parsed codexBuildManifest
	if jsonErr := json.Unmarshal(raw, &parsed); jsonErr != nil {
		return []HealthIssue{
			issueCritical("bad_manifest", manifestPath,
				fmt.Sprintf("Manifest JSON parse failed: %v", jsonErr)),
		}
	}

	var issues []HealthIssue

	// Check phase field matches.
	if parsed.Phase != state.CurrentPhase {
		issues = append(issues, fixableIssue(issueWarning("bad_manifest", manifestPath,
			fmt.Sprintf("Phase field mismatch: manifest has phase %d, state has %d", parsed.Phase, state.CurrentPhase))))
	}

	// Check generated_at is not empty.
	if parsed.GeneratedAt == "" {
		issues = append(issues, fixableIssue(issueWarning("bad_manifest", manifestPath,
			"Manifest has empty generated_at field")))
	}

	// Check state field is not empty.
	if parsed.State == "" {
		issues = append(issues, fixableIssue(issueWarning("bad_manifest", manifestPath,
			"Manifest has empty state field")))
	}

	// Check dispatches reference valid tasks.
	if len(parsed.Dispatches) > 0 {
		// Build a set of task IDs from the plan phase.
		validTaskIDs := make(map[string]bool)
		for _, p := range state.Plan.Phases {
			if p.ID == state.CurrentPhase {
				for idx, t := range p.Tasks {
					taskID := buildTaskID(t, idx)
					validTaskIDs[taskID] = true
				}
				break
			}
		}
		for _, d := range parsed.Dispatches {
			if d.TaskID != "" && !validTaskIDs[d.TaskID] {
				issues = append(issues, fixableIssue(issueWarning("bad_manifest", manifestPath,
					fmt.Sprintf("Dispatch references non-existent task: %s", d.TaskID))))
			}
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// DETECT-05: Dirty Worktrees
// ---------------------------------------------------------------------------

// scanDirtyWorktrees detects worktrees with uncommitted changes, state-disk
// mismatches, and orphan branches.
func scanDirtyWorktrees(state colony.ColonyState) []HealthIssue {
	if len(state.Worktrees) == 0 {
		return nil
	}

	var issues []HealthIssue

	// Get actual git worktree paths for cross-referencing.
	diskPaths := getGitWorktreePaths()

	for _, wt := range state.Worktrees {
		// Skip already-merged or orphaned entries.
		if wt.Status == colony.WorktreeMerged || wt.Status == colony.WorktreeOrphaned {
			continue
		}

		if wt.Path == "" {
			continue
		}

		// Check if the worktree path exists on disk.
		if _, err := os.Stat(wt.Path); os.IsNotExist(err) {
			// State says allocated/in-progress but path is gone.
			issues = append(issues, fixableIssue(issueWarning("dirty_worktree", wt.Path,
				fmt.Sprintf("Worktree state-disk mismatch: state says %s but path does not exist", wt.Status))))
			continue
		}

		// Check if git worktree list knows about this path.
		if !diskPaths[wt.Path] {
			issues = append(issues, fixableIssue(issueWarning("dirty_worktree", wt.Path,
				"Worktree in state but not in git worktree list")))
			continue
		}

		// Run git status --porcelain in the worktree directory.
		cmd := exec.Command("git", "-C", wt.Path, "status", "--porcelain")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(output)) != "" {
			lineCount := len(strings.Split(strings.TrimSpace(string(output)), "\n"))
			issues = append(issues, issueCritical("dirty_worktree", wt.Path,
				fmt.Sprintf("Worktree has %d uncommitted change(s)", lineCount)))
		}
	}

	// Check for orphan branches.
	orphans, orphanErr := reportOrphanBranches()
	if orphanErr == nil {
		for _, orphan := range orphans {
			name, _ := orphan["branch"].(string)
			if name == "" {
				name = "unknown"
			}
			issues = append(issues, fixableIssue(issueWarning("dirty_worktree", name,
				fmt.Sprintf("Orphan branch: %s (no worktree, not tracked in state)", name))))
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// DETECT-06: Broken Survey
// ---------------------------------------------------------------------------

// scanBrokenSurvey detects missing, corrupted, or empty survey files when
// a survey was previously run.
func scanBrokenSurvey(state colony.ColonyState, dataDir string) []HealthIssue {
	if state.TerritorySurveyed == nil {
		return nil
	}

	surveyDir := filepath.Join(dataDir, "survey")
	var issues []HealthIssue

	for _, name := range surveyFiles {
		filePath := filepath.Join(surveyDir, name+".json")

		// Check if file exists.
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			issues = append(issues, fixableIssue(issueWarning("broken_survey",
				fmt.Sprintf("survey/%s.json", name),
				fmt.Sprintf("Survey file missing: %s.json", name))))
			continue
		}

		// Read and validate JSON.
		raw, readErr := os.ReadFile(filePath)
		if readErr != nil {
			issues = append(issues, fixableIssue(issueCritical("broken_survey",
				fmt.Sprintf("survey/%s.json", name),
				fmt.Sprintf("Cannot read survey file %s.json: %v", name, readErr))))
			continue
		}

		if !json.Valid(raw) {
			issues = append(issues, fixableIssue(issueCritical("broken_survey",
				fmt.Sprintf("survey/%s.json", name),
				fmt.Sprintf("Survey file has invalid JSON: %s.json", name))))
			continue
		}

		// Check for empty content (null, {}, []).
		trimmed := strings.TrimSpace(string(raw))
		if trimmed == "null" || trimmed == "{}" || trimmed == "[]" || trimmed == "" {
			issues = append(issues, fixableIssue(issueWarning("broken_survey",
				fmt.Sprintf("survey/%s.json", name),
				fmt.Sprintf("Survey file is empty: %s.json", name))))
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// DETECT-07: Missing Agent Files
// ---------------------------------------------------------------------------

// scanMissingAgentFiles checks that all three agent surfaces have the expected
// number of agent definition files.
func scanMissingAgentFiles() []HealthIssue {
	repoRoot := resolveAetherRoot()

	surfaces := []struct {
		name     string
		pattern  string
		expected int
	}{
		{"Claude agents", filepath.Join(repoRoot, ".claude", "agents", "ant", "*.md"), expectedClaudeAgents},
		{"OpenCode agents", filepath.Join(repoRoot, ".opencode", "agents", "*.md"), expectedOpenCodeAgents},
		{"Codex agents", filepath.Join(repoRoot, ".codex", "agents", "*.toml"), expectedCodexAgents},
	}

	var issues []HealthIssue

	for _, surface := range surfaces {
		files, globErr := filepath.Glob(surface.pattern)
		if globErr != nil {
			continue
		}
		count := len(files)
		if count < surface.expected {
			msg := fmt.Sprintf("%s: found %d files, expected %d", surface.name, count, surface.expected)

			// Cross-check hub for available files.
			hubNote := ""
			hubDir := ""
			switch surface.name {
			case "Claude agents":
				hubDir = filepath.Join(homeDir(), ".aether", "system", "claude", "agents")
			case "OpenCode agents":
				hubDir = filepath.Join(homeDir(), ".aether", "system", "opencode", "agents")
			case "Codex agents":
				hubDir = filepath.Join(homeDir(), ".aether", "system", "codex", "agents")
			}
			if hubDir != "" {
				if hubFiles, err := filepath.Glob(filepath.Join(hubDir, "*")); err == nil && len(hubFiles) > 0 {
					hubNote = " (files available in hub)"
				}
			}
			msg += hubNote

			issues = append(issues, fixableIssue(issueWarning("missing_agents", surface.pattern, msg)))
		}
	}

	return issues
}

// homeDir returns the user's home directory.
func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}
