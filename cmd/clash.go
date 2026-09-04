package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// Clash detection prevents file conflicts between worktrees.

// --- clash-check ---

var clashCheckCmd = &cobra.Command{
	Use:   "clash-check",
	Short: "Check if file is modified in another worktree",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		file := mustGetString(cmd, "file")
		if file == "" {
			return nil
		}

		// List all worktrees and check if the file is modified in any of them
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain").Output()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree list timed out after %v", GitTimeout), nil)
				return nil
			}
			outputOK(map[string]interface{}{"clash": false, "reason": "not a git worktree repo"})
			return nil
		}

		clashingWorktrees := []string{}

		// Check for modifications in each worktree
		worktreePaths := parseWorktreePaths(string(out))
		for _, wtPath := range worktreePaths {
			// Check if file has changes in that worktree
			diffCtx, diffCancel := context.WithTimeout(context.Background(), GitTimeout)
			defer diffCancel()
			diffCmd := exec.CommandContext(diffCtx, "git", "-C", wtPath, "diff", "--name-only", "HEAD", "--", file)
			diffOut, diffErr := diffCmd.Output()
			if diffErr == nil && strings.TrimSpace(string(diffOut)) != "" {
				clashingWorktrees = append(clashingWorktrees, wtPath)
			}
		}

		if len(clashingWorktrees) > 0 {
			outputOK(map[string]interface{}{
				"clash":     true,
				"file":      file,
				"worktrees": clashingWorktrees,
				"count":     len(clashingWorktrees),
			})
		} else {
			outputOK(map[string]interface{}{"clash": false, "file": file})
		}
		return nil
	},
}

func parseWorktreePaths(porcelain string) []string {
	var paths []string
	lines := strings.Split(porcelain, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	return paths
}

// --- clash-setup ---

var clashSetupCmd = &cobra.Command{
	Use:   "clash-setup",
	Short: "Install clash detection hooks and merge driver",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up git merge driver for clash detection
		driverCmd := "aether clash-check --file %A"
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		if err := exec.CommandContext(ctx, "git", "config", "--local", "merge.aether-clash.name", "Aether Clash Detection").Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git config timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to set merge driver name: %v", err), nil)
			return nil
		}
		if err := exec.CommandContext(ctx, "git", "config", "--local", "merge.aether-clash.driver", driverCmd).Run(); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git config timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to set merge driver: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"setup": true, "driver": "aether-clash"})
		return nil
	},
}

// --- worktree-create ---

var worktreeCreateCmd = &cobra.Command{
	Use:   "worktree-create",
	Short: "Create worktree with pheromone injection",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}

		// Create worktree
		ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
		defer cancel()
		out, err := exec.CommandContext(ctx, "git", "worktree", "add", branch, "-b", branch).CombinedOutput()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				outputError(2, fmt.Sprintf("git worktree add timed out after %v", GitTimeout), nil)
				return nil
			}
			outputError(2, fmt.Sprintf("failed to create worktree: %v: %s", err, string(out)), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"branch":  branch,
			"path":    branch,
		})
		return nil
	},
}

// --- worktree-cleanup ---

// resolveWorktreePathForBranch finds the actual worktree directory for a
// given branch by asking git directly (`git -C root worktree list
// --porcelain`), rather than assuming the branch name IS the path. Passing
// a branch name like "phase-1/builder-1" straight to `git worktree remove`
// as its path argument only works by coincidence when the branch name
// happens to equal the worktree's directory name -- in every other case git
// fails to resolve it, or worse, silently matches nothing.
func resolveWorktreePathForBranch(root, branch string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GitTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", root, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list: %w", err)
	}

	var currentPath string
	wantRef := "branch refs/heads/" + branch
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			currentPath = strings.TrimPrefix(line, "worktree ")
		case line == wantRef:
			if currentPath == "" {
				return "", fmt.Errorf("found branch %q in worktree list but no preceding worktree path", branch)
			}
			return currentPath, nil
		}
	}
	return "", fmt.Errorf("no worktree found for branch %q", branch)
}

var worktreeCleanupCmd = &cobra.Command{
	Use:   "worktree-cleanup",
	Short: "Clean up worktree after merge",
	Long: "Removes a worker workspace (git worktree) for the given branch after " +
		"its work has been merged. Before removing anything, checks whether the " +
		"worktree still holds uncommitted or unmerged work; if it does, the work " +
		"is kept and reported instead of being destroyed (run with --force to " +
		"remove it anyway -- the work is saved to a stash first, exactly as " +
		"`aether worktree-reap` does).",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		branch := mustGetString(cmd, "branch")
		if branch == "" {
			return nil
		}
		force, _ := cmd.Flags().GetBool("force")

		root := resolveAetherRoot()

		wtPath, resolveErr := resolveWorktreePathForBranch(root, branch)
		if resolveErr != nil {
			outputError(2, fmt.Sprintf("failed to locate worker workspace for branch %q: %v", branch, resolveErr), nil)
			return nil
		}

		entry := colony.WorktreeEntry{
			Path:   wtPath,
			Branch: branch,
		}
		// If this worktree is tracked in COLONY_STATE.json, prefer that
		// entry -- it may carry additional bookkeeping (phase, agent) that
		// downstream reporting relies on, though only Path and Branch are
		// used by the safety check itself.
		if store != nil {
			var state colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil {
				for _, wt := range state.Worktrees {
					if wt.Branch == branch {
						entry = wt
						break
					}
				}
			}
		}

		safety := worktreeDestructionSafety(root, entry)

		if !safety.Safe && !force {
			reportWorktreePreservation(safety, fmt.Sprintf("branch %s was left alone", branch))
			outputOK(map[string]interface{}{
				"cleaned":   false,
				"preserved": true,
				"branch":    branch,
				"reason":    safety.Reason,
			})
			return nil
		}

		if !safety.Safe && force {
			// --force alone does not throw away unmerged or dirty work
			// (D-01) -- it is preserved (stashed) first, exactly like
			// worktree-reap's --force --include-unmerged path, then removed.
			preservedOK, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
			if preserveErr != nil || !preservedOK {
				reportWorktreePreservation(safety, fmt.Sprintf("could not save the work before removal; branch %s was left alone", branch))
				outputOK(map[string]interface{}{
					"cleaned":   false,
					"preserved": true,
					"branch":    branch,
					"reason":    safety.Reason,
				})
				return nil
			}
			fmt.Fprintln(os.Stderr, clashPreservedWorkMessage199(safety, fmt.Sprintf("removing anyway (--force) — its changes were saved first (%s)", detail)))
		}

		if err := removeGitWorktree(root, wtPath, branch); err != nil {
			outputError(2, fmt.Sprintf("failed to remove worktree: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"cleaned": true,
			"branch":  branch,
		})
		return nil
	},
}

// clashPreservedWorkMessage199 keeps the forced-cleanup preservation notice on
// the same read-only inspection and resume routes as every other saved-worker
// path. The branch and save evidence remain part of the emitted message.
func clashPreservedWorkMessage199(safety worktreeSafety, detail string) string {
	return describeWorktreePreservation(safety, detail)
}

func init() {
	clashCheckCmd.Flags().String("file", "", "File path to check (required)")
	worktreeCreateCmd.Flags().String("branch", "", "Branch name (required)")
	worktreeCleanupCmd.Flags().String("branch", "", "Branch name (required)")
	worktreeCleanupCmd.Flags().Bool("force", false, "remove even if the worktree holds uncommitted or unmerged work (saved to a stash first)")

	for _, c := range []*cobra.Command{
		clashCheckCmd, clashSetupCmd, worktreeCreateCmd, worktreeCleanupCmd,
	} {
		rootCmd.AddCommand(c)
	}
}
