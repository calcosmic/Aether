package cmd

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestMain saves all package-level globals before running tests and restores
// them after. This prevents test pollution where one test's store/stdout/stderr
// assignment leaks into subsequent tests. Belt-and-suspenders with per-test
// cleanup via saveGlobals.
func TestMain(m *testing.M) {
	origStore := store
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origTracer := tracer
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath

	code := m.Run()

	// Clean up git worktrees and branches created by tests.
	// Tests like TestWorktreeAllocateAuditLog create real git worktrees
	// that persist after the test suite finishes.
	cleanupTestWorktrees()

	store = origStore
	stdout = origStdout
	stderr = origStderr
	flagTypeFilter = origFlagType
	flagStatusFilter = origFlagStatus
	flagListJSON = origFlagListJSON
	historyJSON = origHistoryJSON
	phaseJSON = origPhaseJSON
	historyLimit = origHistoryLimit
	historyFilter = origHistoryFilter
	phaseNumber = origPhaseNumber
	tracer = origTracer
	continueContextUpdater = origContinueContextUpdater
	continueSignalHousekeeper = origContinueSignalHousekeeper
	newCodexWorkerInvoker = origNewCodexWorkerInvoker
	activeBuildCeremony = origActiveBuildCeremony
	narratorLookPath = origNarratorLookPath
	narratorCommandContext = origNarratorCommandContext
	narratorRuntimePath = origNarratorRuntimePath

	os.Exit(code)
}

// saveGlobals captures the current values of all mutable package-level globals
// and restores them when the test completes. Every test that assigns to store,
// stdout, stderr, flagTypeFilter, or flagStatusFilter must call this as its
// first action.
func saveGlobals(t *testing.T) {
	t.Helper()
	origStore := store
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origTracer := tracer
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath
	t.Cleanup(func() {
		store = origStore
		stdout = origStdout
		stderr = origStderr
		flagTypeFilter = origFlagType
		flagStatusFilter = origFlagStatus
		flagListJSON = origFlagListJSON
		historyJSON = origHistoryJSON
		phaseJSON = origPhaseJSON
		historyLimit = origHistoryLimit
		historyFilter = origHistoryFilter
		phaseNumber = origPhaseNumber
		tracer = origTracer
		continueContextUpdater = origContinueContextUpdater
		continueSignalHousekeeper = origContinueSignalHousekeeper
		newCodexWorkerInvoker = origNewCodexWorkerInvoker
		activeBuildCeremony = origActiveBuildCeremony
		narratorLookPath = origNarratorLookPath
		narratorCommandContext = origNarratorCommandContext
		narratorRuntimePath = origNarratorRuntimePath
	})
}

// resetRootCmd restores rootCmd state (SetArgs, SetOut) and resets all
// subcommand local flags to their defaults. Cobra local flags persist across
// Execute() calls, so without this, --type blocker on flag-add would leak
// into the next test that runs flag-add without --type.
// Every test that calls rootCmd.Execute must call this early in the function.
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		// Reset local flags on all subcommands to their defaults.
		// This prevents flag value leakage between tests.
		resetFlags(rootCmd)
	})
}

// resetFlags recursively resets all local flags on cmd and its subcommands
// to their default values. This prevents Cobra flag leakage between tests.
func resetFlags(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if sliceValue, ok := f.Value.(pflag.SliceValue); ok {
			if f.DefValue == "" || f.DefValue == "[]" {
				_ = sliceValue.Replace(nil)
			}
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

// cleanupTestWorktrees removes git worktrees and branches created by the test
// suite. Tests like TestWorktreeAllocateAuditLog create real worktrees in
// cmd/.aether/worktrees/ that persist after tests finish. This runs once after
// all tests complete.
func cleanupTestWorktrees() {
	// Remove all non-main worktrees
	out, _ := exec.Command("git", "worktree", "list", "--porcelain").Output()
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		path := strings.TrimPrefix(line, "worktree ")
		// Skip the main working tree (matches the repo root)
		if !strings.Contains(path, "/.aether/worktrees/") && !strings.Contains(path, "-worktrees/") {
			continue
		}
		exec.Command("git", "worktree", "remove", path, "--force").Run()
	}

	// Prune any remaining detached entries
	exec.Command("git", "worktree", "prune").Run()

	// Delete branches that match test-created patterns
	branchOut, _ := exec.Command("git", "branch", "--list").Output()
	branches := strings.Split(string(branchOut), "\n")
	for _, br := range branches {
		br = strings.TrimSpace(br)
		br = strings.TrimPrefix(br, "* ")
		if br == "" || br == "main" || br == "master" {
			continue
		}
		// Only delete branches that match test patterns
		if strings.HasPrefix(br, "feature/test-audit-") ||
			strings.HasPrefix(br, "phase-") ||
			(strings.HasPrefix(br, "feature/") && len(strings.Split(br, "/")) == 2) {
			exec.Command("git", "branch", "-D", br).Run()
		}
	}
}
