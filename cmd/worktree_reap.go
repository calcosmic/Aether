package cmd

import (
	"fmt"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// worktreeReapCmd is the named, operator-invoked destruction command D-01
// requires: destruction of a WORKTREE THAT MAY HOLD DIRTY OR UNMERGED WORK
// is deferred to an explicit command a human types — never to an automatic
// cleanup running inside resume, continue, init, build, build-finalize,
// run, or any autopilot path. That boundary is enforced by
// TestWorktreeReapHasNoLifecycleCaller (cmd/worktree_crash_safety_test.go),
// which scans every lifecycle source file for a call to runWorktreeReap or
// the literal command name and fails the build if either appears outside a
// human-invoked context.
//
// This is NOT a claim that worktreeReapCmd is the only caller of
// removeGitWorktree in the codebase — a prior version of this comment made
// that claim and it was wrong (CR-05). Two other, narrower callers remain
// and are intentionally out of scope for the D-01 boundary because neither
// destroys a worktree without first knowing its contents were already
// handled: allocateBuildWorktree's own rollback (cmd/codex_build_worktree.go)
// removes a worktree it JUST created in the same call and never registered
// as holding worker output, and finalizeBuildWorktree
// (cmd/codex_build_worktree.go) runs only after reconcileWorktreeWave has
// already synced a successful worker's changes into the root checkout —
// the crash-recovery paths this phase protects (resume, continue, init,
// and cleanupBuildWorktrees on the build path) all route through
// worktreeDestructionSafety and preserveWorktreeWork before ever reaching
// removeGitWorktree, and none of the three calls it directly.
var worktreeReapCmd = &cobra.Command{
	Use:   "worktree-reap",
	Short: "Remove finished worker workspaces",
	Long: "Removes worker workspaces (git worktrees) that are no longer needed. " +
		"By default it only shows what it would remove and deletes nothing — " +
		"pass --force to actually remove anything. Work that was never merged " +
		"back, or that has unsaved changes, is kept even with --force; pass " +
		"--include-unmerged as well if you are certain you want that work gone " +
		"too, and even then it is saved first so it can be recovered.",
	Args: cobra.NoArgs,
	RunE: runWorktreeReap,
}

func init() {
	rootCmd.AddCommand(worktreeReapCmd)
	worktreeReapCmd.Flags().Bool("force", false, "actually remove worker workspaces; without this flag nothing is deleted")
	worktreeReapCmd.Flags().Bool("include-unmerged", false, "also remove workspaces holding unsaved or unmerged work (requires --force; the work is saved to a stash first)")
	worktreeReapCmd.Flags().String("branch", "", "act on exactly one worker workspace, named by its branch")
	worktreeReapCmd.Flags().Bool("json", false, "output structured JSON")
}

// worktreeReapCandidate is one worktree entry paired with the safety
// verdict computed for it, used for both the read-only report and the
// --force execution pass.
type worktreeReapCandidate struct {
	entry  colony.WorktreeEntry
	safety worktreeSafety
}

// worktreeReapActionCategory is this command's classification, checked
// against isDestructiveWorktreeAction so the command is discoverable as
// destructive the same way recover_repair.go classifies "dirty_worktree" —
// a sibling category on the same isDestructiveCategory-style ledger.
const worktreeReapActionCategory = worktreeDestructionCategory

func runWorktreeReap(cmd *cobra.Command, args []string) error {
	if !isDestructiveWorktreeAction(worktreeReapActionCategory) {
		// Defensive: this command's whole reason for existing is to be the
		// one destructive, operator-invoked path. If its own category ever
		// stops classifying as destructive, something is structurally
		// wrong — fail loudly rather than silently behave as if it were
		// safe.
		return fmt.Errorf("internal error: worktree-reap is not classified as destructive")
	}
	if store == nil {
		outputErrorMessage("no store initialized")
		return nil
	}

	force, _ := cmd.Flags().GetBool("force")
	includeUnmerged, _ := cmd.Flags().GetBool("include-unmerged")
	branchFilter, _ := cmd.Flags().GetString("branch")
	jsonOut, _ := cmd.Flags().GetBool("json")

	root := resolveAetherRoot()

	// Read-only path: no --force means report only, mutate nothing. This
	// satisfies CLAUDE.md's corollary that an inspection command must not
	// mutate state — the corollary exists because consolidation-phase-end
	// --dry-run and consolidation-seal --dry-run wrote to instincts.json for
	// months despite their flag help saying otherwise.
	if !force {
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{"candidates": []interface{}{}, "force": false})
			return nil
		}

		candidates := worktreeReapCandidates(root, state.Worktrees, branchFilter)
		reportWorktreeReapPlan(candidates, jsonOut)
		return nil
	}

	// --force: iterate tracked worktrees, guard every entry with
	// worktreeDestructionSafety before any removeGitWorktree call. Mutate
	// through UpdateJSONAtomically, matching gcOrphanedWorktrees's shape
	// rather than the older LoadJSON/SaveJSON pair, which has a race window
	// between load and save.
	var removedBranches []string
	var preservedBranches []string
	var state colony.ColonyState
	if err := store.UpdateJSONAtomically("COLONY_STATE.json", &state, func() error {
		var remaining []colony.WorktreeEntry
		for _, entry := range state.Worktrees {
			if branchFilter != "" && entry.Branch != branchFilter {
				remaining = append(remaining, entry)
				continue
			}

			safety := worktreeDestructionSafety(root, entry)

			if safety.Safe {
				// Clean and fully merged (or path already gone) — safe to
				// remove outright.
				if removeErr := removeGitWorktree(root, safety.Path, entry.Branch); removeErr != nil {
					// Removal failed — do not lose the entry, keep it and
					// report why.
					reportWorktreePreservation(safety, fmt.Sprintf("removal failed (%v); branch %s was left alone", removeErr, entry.Branch))
					remaining = append(remaining, entry)
					preservedBranches = append(preservedBranches, entry.Branch)
					continue
				}
				removedBranches = append(removedBranches, entry.Branch)
				continue
			}

			if !includeUnmerged {
				// Even the explicit destruction command does not throw away
				// unmerged or dirty work on --force alone (D-01: "never
				// destroyed"). --include-unmerged is the separate opt-in.
				// The worktree itself is not being touched in this branch —
				// nothing is removed, so there is nothing to stash. Stashing
				// here would needlessly disturb a worktree that is being
				// left alone.
				reportWorktreePreservation(safety, fmt.Sprintf("branch %s was left alone", entry.Branch))
				entry.Status = colony.WorktreeOrphaned
				remaining = append(remaining, entry)
				preservedBranches = append(preservedBranches, entry.Branch)
				continue
			}

			// --force --include-unmerged: the operator explicitly asked for
			// dirty or unmerged work to go. Preserve FIRST (stash) so the
			// changes land safely before the worktree goes, then destroy.
			preservedOK, detail, preserveErr := preserveWorktreeWork(root, entry, safety)
			if preserveErr != nil || !preservedOK {
				// Could not even stash, or the state was undeterminable so
				// nothing was actually saved (CR-04) — either way, refuse to
				// destroy on top of a save that did not happen. Discarding
				// the preservedOK boolean here (the old `_,` pattern) is
				// what let this path print "its changes were saved first"
				// and then destroy a worktree whose contents were never
				// examined.
				reportWorktreePreservation(safety, fmt.Sprintf("could not save the work before removal; branch %s was left alone", entry.Branch))
				remaining = append(remaining, entry)
				preservedBranches = append(preservedBranches, entry.Branch)
				continue
			}
			visualFprintln(stderr, worktreeReapSavedWorkMessage199(entry.Branch, detail))
			if removeErr := removeGitWorktree(root, safety.Path, entry.Branch); removeErr != nil {
				remaining = append(remaining, entry)
				preservedBranches = append(preservedBranches, entry.Branch)
				continue
			}
			removedBranches = append(removedBranches, entry.Branch)
		}
		state.Worktrees = remaining
		return nil
	}); err != nil {
		outputError(2, fmt.Sprintf("failed to update colony state: %v", err), nil)
		return nil
	}

	outputOK(map[string]interface{}{
		"force":     true,
		"removed":   removedBranches,
		"preserved": preservedBranches,
	})
	return nil
}

func worktreeReapSavedWorkMessage199(branch, detail string) string {
	return fmt.Sprintf("Removing worker workspace on branch %s — its changes were saved first (%s). Inspect the saved work with `aether maintenance recovery-inspect` (State effect: none). To restore runnable lifecycle state, run `aether resume`.", branch, detail)
}

// worktreeReapCandidates computes the safety verdict for every worktree
// entry a read-only worktree-reap call would consider, without mutating
// anything.
func worktreeReapCandidates(root string, worktrees []colony.WorktreeEntry, branchFilter string) []worktreeReapCandidate {
	var candidates []worktreeReapCandidate
	for _, entry := range worktrees {
		if branchFilter != "" && entry.Branch != branchFilter {
			continue
		}
		candidates = append(candidates, worktreeReapCandidate{
			entry:  entry,
			safety: worktreeDestructionSafety(root, entry),
		})
	}
	return candidates
}

// reportWorktreeReapPlan prints what a --force run would do, for a
// non-technical reader, without deleting or mutating anything.
func reportWorktreeReapPlan(candidates []worktreeReapCandidate, jsonOut bool) {
	if jsonOut {
		results := make([]map[string]interface{}, 0, len(candidates))
		for _, c := range candidates {
			results = append(results, map[string]interface{}{
				"branch":       c.entry.Branch,
				"phase":        c.entry.Phase,
				"would_remove": c.safety.Safe,
				"reason":       c.safety.Reason,
			})
		}
		outputOK(map[string]interface{}{"candidates": results, "force": false})
		return
	}

	if len(candidates) == 0 {
		visualFprintln(stdout, "No worker workspaces to check.")
		return
	}

	visualFprintln(stdout, "This only shows what would happen — nothing is deleted without --force.")
	for _, c := range candidates {
		if c.safety.Safe {
			visualFprintf(stdout, "Would remove: branch %s (phase %d) — %s\n", c.entry.Branch, c.entry.Phase, c.safety.Reason)
		} else {
			visualFprintf(stdout, "Would keep: branch %s (phase %d) — %s. Add --include-unmerged with --force to remove it anyway (its work is saved first).\n", c.entry.Branch, c.entry.Phase, c.safety.Reason)
		}
	}
}
