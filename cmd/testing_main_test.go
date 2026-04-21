package cmd

import (
	"os"
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

	code := m.Run()

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
