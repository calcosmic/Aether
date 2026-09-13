package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// Phase commits — the save-point after every verified phase. When a phase
// durably advances (both continue paths, after the atomic state write), the
// runtime makes ONE git commit containing exactly the files the phase's
// workers reported changing, so repo history mirrors colony history and the
// Archaeologist can walk it. Boundaries, each load-bearing:
//
//   - EXPLICIT PATHS ONLY: `git add -- <paths>` (needed so newly created
//     worker files are known to git) followed by `git commit -m <msg> --
//     <paths>`. Both steps name exactly the worker-reported files; the
//     pathspec commit never touches the rest of the index, so the owner's
//     stray edits — even ones they had already staged — cannot be swept
//     into a colony commit. Never `git add -A`, never `git add .`.
//   - The path list comes from the phase's persisted worker handoffs
//     (changed_files), which finalize already validates as repo-relative.
//   - NON-FATAL: a commit failure never blocks the advance (same contract
//     as consolidation). It writes the uncommitted-changes.marker that the
//     autopilot pause engine checks (autopilot.go), so an unattended run
//     pauses instead of stacking phases on a broken tree; a later success
//     removes the marker.
//   - NEVER pushes. TestPhaseCommitNeverPushes makes this an invariant.
//   - Off switch: `aether phase-commits set off` (ColonyState.PhaseCommits,
//     default on).

// uncommittedChangesMarkerFile is the autopilot pause hook file. The pause
// engine has checked this name since the classic port; nothing wrote it
// until phase commits revived it.
const uncommittedChangesMarkerFile = "uncommitted-changes.marker"

type phaseCommitResult struct {
	Committed  bool   `json:"committed"`
	SHA        string `json:"sha,omitempty"`
	Files      int    `json:"files,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
	Err        string `json:"error,omitempty"`
	// Unaccounted names every worker-claimed path this commit could NOT
	// record: the file is absent from the working tree AND untracked, so
	// git can express neither a change nor a deletion for it. A warning,
	// never a gate -- the phase has already advanced by the time this runs.
	//
	// Before this, such a path was dropped from the commit set in silence,
	// and silence meant two different things: "everything claimed is saved"
	// and "something a worker claimed cannot be found at all". The second is
	// a worker claiming credit for a file it never wrote, which is exactly
	// the case worth hearing about.
	//
	// This is deliberately NOT "claimed paths still dirty after the commit".
	// That check could only ask about paths already in the claimed set, so
	// it could never have detected the v1.0.75 downstream failure (files
	// missing FROM that set) -- it would have been a warning that cannot
	// fire. The cause there is closed by the union in
	// buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go); this covers
	// the different, reachable gap.
	Unaccounted []string `json:"unaccounted,omitempty"`
}

// phaseCommitGitRunner is the exec seam: tests swap it to record argv and to
// prove no git subcommand outside the allowed set ever runs. The production
// runner shells out with the shared GitTimeout.
var phaseCommitGitRunner = func(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// phaseCommitEnabled reads the colony's toggle: on unless explicitly off.
func phaseCommitEnabled(state colony.ColonyState) bool {
	return state.PhaseCommits.Enabled()
}

// phaseChangedFilesFromHandoffs unions the changed_files of every persisted
// worker handoff for the phase — the build's own record of what it touched.
func phaseChangedFilesFromHandoffs(phaseID int) []string {
	records, err := loadWorkerHandoffRecords()
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	for _, record := range records {
		if record.Phase != phaseID {
			continue
		}
		for _, path := range record.ChangedFiles {
			path = strings.TrimSpace(path)
			if path != "" {
				set[path] = true
			}
		}
	}
	files := make([]string, 0, len(set))
	for path := range set {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// phaseCommitMessage is the stable, greppable commit format the
// Archaeologist walks: subject `aether(phase-N): <name> verified complete`,
// body context, and an `Aether-Phase: N` git trailer as the machine field
// (`git log --format='%(trailers:key=Aether-Phase)'`).
func phaseCommitMessage(phaseID int, phaseName, goal string, phaseCount int) string {
	name := strings.TrimSpace(phaseName)
	if name == "" {
		name = fmt.Sprintf("Phase %d", phaseID)
	}
	goal = strings.TrimSpace(goal)
	if len(goal) > 72 {
		goal = goal[:69] + "..."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "aether(phase-%d): %s verified complete\n\n", phaseID, name)
	if goal != "" {
		fmt.Fprintf(&b, "Colony: %s\n", goal)
	}
	if phaseCount > 0 {
		fmt.Fprintf(&b, "Phase: %d of %d\n", phaseID, phaseCount)
	}
	fmt.Fprintf(&b, "\nAether-Phase: %d\n", phaseID)
	return b.String()
}

// commitPhaseAdvance makes the phase's save-point commit. Called at both
// durable advance points strictly AFTER the atomic state write returns —
// side effect, never a gate. Every skip is a clean non-error; only a real
// git failure sets Err (and the autopilot marker).
func commitPhaseAdvance(root string, state colony.ColonyState, phase colony.Phase) phaseCommitResult {
	if !phaseCommitEnabled(state) {
		return phaseCommitResult{SkipReason: "phase commits are off (aether phase-commits set on)"}
	}
	if _, err := phaseCommitGitRunner(root, "rev-parse", "--is-inside-work-tree"); err != nil {
		return phaseCommitResult{SkipReason: "not a git repository"}
	}

	files := phaseChangedFilesFromHandoffs(phase.ID)
	if len(files) == 0 {
		return phaseCommitResult{SkipReason: "no recorded file changes for this phase"}
	}

	// Only commit paths that still exist OR are tracked deletions git can
	// record; pathspec commit errors on paths git knows nothing about, so
	// filter to what the working tree or index can actually express.
	commitable := make([]string, 0, len(files))
	var unaccounted []string
	for _, rel := range files {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			commitable = append(commitable, rel)
			continue
		}
		// The file is gone from the working tree. If git tracks it, the
		// pathspec commit records the deletion — honest tree state.
		if _, err := phaseCommitGitRunner(root, "ls-files", "--error-unmatch", "--", rel); err == nil {
			commitable = append(commitable, rel)
			continue
		}
		// Neither on disk nor tracked: git can say nothing about it. It was
		// dropped in silence before; it is named now.
		unaccounted = append(unaccounted, rel)
	}
	sort.Strings(unaccounted)
	if len(commitable) == 0 {
		return phaseCommitResult{SkipReason: "no recorded file changes exist on disk"}
	}

	// Nothing to commit is a clean skip, not an error: in worktree mode the
	// merge-back already landed the changes at build-finalize, and a re-run
	// continue finds a clean tree.
	statusArgs := append([]string{"status", "--porcelain", "--"}, commitable...)
	statusOut, err := phaseCommitGitRunner(root, statusArgs...)
	if err == nil && strings.TrimSpace(statusOut) == "" {
		return phaseCommitResult{SkipReason: "no uncommitted changes for this phase"}
	}

	// Stage ONLY the worker-reported paths so newly created files are known
	// to git (a pathspec commit cannot include untracked files). This add
	// names explicit paths — the no-sweep guarantee lives in that
	// explicitness, proven by TestPhaseCommitDoesNotSweepUnrelatedDirtyFile
	// and the argv assertions in TestPhaseCommitNeverPushes.
	addArgs := append([]string{"add", "--"}, commitable...)
	if out, err := phaseCommitGitRunner(root, addArgs...); err != nil {
		if store != nil {
			_ = store.AtomicWrite(uncommittedChangesMarkerFile, []byte(fmt.Sprintf("phase %d commit failed at %s: %v\n", phase.ID, time.Now().UTC().Format(time.RFC3339), err)))
		}
		return phaseCommitResult{Err: fmt.Sprintf("git add failed: %v (%s)", err, strings.TrimSpace(out))}
	}

	goal := ""
	if state.Goal != nil {
		goal = *state.Goal
	}
	message := phaseCommitMessage(phase.ID, phase.Name, goal, len(state.Plan.Phases))
	commitArgs := append([]string{"commit", "-m", message, "--"}, commitable...)
	if out, err := phaseCommitGitRunner(root, commitArgs...); err != nil {
		// Failure path: record loudly, write the autopilot pause marker
		// (autopilot checks os.Stat on this exact path), never block the
		// advance.
		if store != nil {
			_ = store.AtomicWrite(uncommittedChangesMarkerFile, []byte(fmt.Sprintf("phase %d commit failed at %s: %v\n", phase.ID, time.Now().UTC().Format(time.RFC3339), err)))
		}
		summary := strings.TrimSpace(out)
		if len(summary) > 200 {
			summary = summary[:200]
		}
		return phaseCommitResult{Err: fmt.Sprintf("git commit failed: %v (%s)", err, summary)}
	}

	sha, _ := phaseCommitGitRunner(root, "rev-parse", "--short", "HEAD")
	if store != nil {
		_ = os.Remove(filepath.Join(store.BasePath(), uncommittedChangesMarkerFile))
	}
	return phaseCommitResult{
		Committed:   true,
		SHA:         sha,
		Files:       len(commitable),
		Unaccounted: unaccounted,
	}
}

// attachPhaseCommitResult surfaces the outcome on the command result map —
// one honest line whether it committed, skipped, or failed.
func attachPhaseCommitResult(result map[string]interface{}, commit phaseCommitResult) {
	if result == nil {
		return
	}
	entry := map[string]interface{}{"committed": commit.Committed}
	if commit.SHA != "" {
		entry["sha"] = commit.SHA
	}
	if commit.Files > 0 {
		entry["files"] = commit.Files
	}
	if commit.SkipReason != "" {
		entry["skip_reason"] = commit.SkipReason
	}
	if commit.Err != "" {
		entry["error"] = commit.Err
	}
	if len(commit.Unaccounted) > 0 {
		entry["unaccounted"] = commit.Unaccounted
		entry["unaccounted_warning"] = fmt.Sprintf(
			"%d file(s) a worker claimed could not be saved: not present in the project and not known to git. Check whether that work was actually done: %s",
			len(commit.Unaccounted), strings.Join(commit.Unaccounted, ", "))
	}
	result["phase_commit"] = entry
}

// --- phase-commits get|set ---

var phaseCommitsCmd = &cobra.Command{
	Use:   "phase-commits",
	Short: "Get or set the per-phase git save-point behaviour",
	Args:  cobra.NoArgs,
}

var phaseCommitsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show whether the colony commits each verified phase",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{"mode": "on", "source": "default"})
			return nil
		}
		mode := string(state.PhaseCommits)
		source := "state"
		if mode == "" {
			mode = "on"
			source = "default"
		}
		outputOK(map[string]interface{}{"mode": mode, "source": source})
		return nil
	},
}

var phaseCommitsSetCmd = &cobra.Command{
	Use:   "set <on|off>",
	Short: "Turn the per-phase git save-point on or off",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		raw := mustGetString(cmd, "mode")
		if len(args) == 1 {
			raw = args[0]
		}
		mode := colony.PhaseCommitMode(raw)
		if mode == "" {
			outputError(1, "usage: aether phase-commits set <on|off>", nil)
			return nil
		}
		if !mode.Valid() {
			outputError(1, fmt.Sprintf("invalid phase-commits mode %q: must be on or off", string(mode)), nil)
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}
		state.PhaseCommits = mode
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"mode": string(mode), "source": "cli"})
		return nil
	},
}

func init() {
	phaseCommitsSetCmd.Flags().String("mode", "", "on or off")
	phaseCommitsCmd.AddCommand(phaseCommitsGetCmd)
	phaseCommitsCmd.AddCommand(phaseCommitsSetCmd)
	rootCmd.AddCommand(phaseCommitsCmd)
}
