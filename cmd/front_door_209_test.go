package cmd

// 209-03 (D-02): the default front-door help screen is now a short,
// six-command everyday menu instead of the full twenty-two-command
// catalogue -- gated behind the machine-wide advanced-commands setting
// (cmd/advanced_commands.go). Nothing is deleted, disabled, or deprecated
// to get there: every demoted command stays registered and runnable when
// typed directly. These tests prove that guarantee against the real
// rendered screen (frontDoorHelpOutput199, reused from cmd/front_door_199_test.go)
// rather than against a description of it.

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestDefaultMenuShowsOnlyTheApprovedSet proves the default (advanced
// setting off) screen shows exactly the approved everyday commands and no
// other registered wrapper command -- checked against the full set of
// registered wrapper names (wrapperCommandNames), not a hand-written
// exclusion list, so a wrapper added later is covered automatically.
func TestDefaultMenuShowsOnlyTheApprovedSet(t *testing.T) {
	root := t.TempDir()
	// Deliberately do NOT call frontDoorTurnOnAdvancedCommandsForTest: the
	// advanced setting must be genuinely off (the default) for this test.
	got := frontDoorHelpOutput199(t, root, 100)

	approved := map[string]bool{}
	for _, entry := range frontDoorDefaultMenu {
		name, _, _ := strings.Cut(entry.command, " ")
		approved[name] = true
		if !strings.Contains(got, entry.command) {
			t.Errorf("default help is missing approved entry %q:\n%s", entry.command, got)
		}
	}

	// Scope the exclusion scan to the menu itself, not the whole screen: the
	// standing line above it legitimately names /ant-init ("No colony is
	// active... /ant-init \"goal\"") regardless of which menu renders below
	// it -- that onboarding nudge is unconditional prose, not a catalog
	// exposure, and is unrelated to D-02's six-command menu.
	menuStart := strings.Index(got, frontDoorDefaultGroupTitle)
	if menuStart < 0 {
		t.Fatalf("default help is missing its own group title %q:\n%s", frontDoorDefaultGroupTitle, got)
	}
	menuSection := got[menuStart:]

	// Exact-token comparison, not substring containment: "/ant-flag" is a
	// literal substring of the approved "/ant-flags", so a naive
	// strings.Contains check would misreport /ant-flag as present.
	tokens := map[string]bool{}
	for _, field := range strings.Fields(menuSection) {
		tokens[field] = true
	}
	for verb := range wrapperCommandNames {
		name := "/ant-" + verb
		if verb == "help" || approved[name] {
			// /ant-help is always reachable; it is the screen being shown,
			// not a seventh entry on it (209-CONTEXT.md D-02).
			continue
		}
		if tokens[name] {
			t.Errorf("default help exposes non-approved command %q, but the default menu must show only the approved six:\n%s", name, got)
		}
	}

	if !strings.Contains(got, "aether advanced-commands set on") {
		t.Errorf("default help does not name the way to see the full list back on:\n%s", got)
	}
}

// frontDoorNonCLIWrapperCommands names public wrapper commands that were
// ALREADY not backed by any registered cobra command before 209-03 -- they
// are prompt-only ceremonies (no `runtime.command` in their
// .aether/commands/*.yaml source) that rootCmd.Find can never resolve
// regardless of what this plan does. D-02's "hiding never disables"
// guarantee is about cobra command registration; there is nothing at that
// layer for these two to lose, and their absence from rootCmd.Find predates
// this phase (confirmed against the pre-209-03 binary). They are excluded
// here rather than silently passed, so a future genuine regression in this
// set is still visible: TestHiddenCommandsStillRun asserts every OTHER
// previously-catalogued command resolves.
var frontDoorNonCLIWrapperCommands = map[string]bool{
	"dream":     true,
	"interpret": true,
}

// TestHiddenCommandsStillRun proves D-02's central promise: hiding a
// command off the default menu never disables it. Every command that was
// part of the OLD three-group catalogue (frontDoorHelpGroups) and is absent
// from the new approved default menu must still be registered, unhidden,
// undeprecated, and runnable -- and two of them are actually run end to end
// in a temporary project to prove it, not just inspected.
func TestHiddenCommandsStillRun(t *testing.T) {
	approved := map[string]bool{}
	for _, entry := range frontDoorDefaultMenu {
		name, _, _ := strings.Cut(entry.command, " ")
		approved[strings.TrimPrefix(name, "/ant-")] = true
	}

	seen := map[string]bool{}
	var demoted []string
	for _, group := range frontDoorHelpGroups {
		for _, entry := range group.entries {
			name, _, _ := strings.Cut(entry.command, " ")
			verb := strings.TrimPrefix(name, "/ant-")
			if approved[verb] || seen[verb] || frontDoorNonCLIWrapperCommands[verb] {
				continue
			}
			seen[verb] = true
			demoted = append(demoted, verb)
		}
	}
	sort.Strings(demoted)
	if len(demoted) == 0 {
		t.Fatal("no demoted commands found -- cannot prove the hiding-never-disables guarantee against an empty set")
	}

	for _, verb := range demoted {
		command, _, err := rootCmd.Find([]string{verb})
		if err != nil || command == nil || command == rootCmd {
			t.Errorf("demoted command %q is not registered: %v", verb, err)
			continue
		}
		if command.Hidden {
			t.Errorf("demoted command %q is marked Hidden -- hiding off the default menu must never disable a command", verb)
		}
		if command.Deprecated != "" {
			t.Errorf("demoted command %q is marked Deprecated (%q) -- hiding off the default menu must never disable a command", verb, command.Deprecated)
		}
		if command.RunE == nil && command.Run == nil && !command.HasSubCommands() {
			t.Errorf("demoted command %q has no RunE/Run and no subcommands -- it cannot actually run", verb)
		}
	}

	// Actually run two demoted commands end to end in a temporary project,
	// not merely inspect their registration.
	runFrontDoor209DemotedCommand(t, "history")
	runFrontDoor209DemotedCommand(t, "focus", "prove hiding never disables a command")
}

func runFrontDoor209DemotedCommand(t *testing.T, verb string, args ...string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(root, ".aether", "data"))
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "1")
	store = nil
	var out bytes.Buffer
	stdout = &out
	stderr = &out
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append([]string{verb}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("demoted command %q failed to run end to end: %v (output: %s)", verb, err, out.String())
	}
	if out.Len() == 0 {
		t.Fatalf("demoted command %q produced no output", verb)
	}
}

// TestAskingForTheMenuWritesNothing proves both rendered screens -- the
// default (off) six-command menu and the full (on) catalogue -- are purely
// read-only: neither creates a data directory, a lock directory, or a
// preferences file as a side effect of merely being asked for.
func TestAskingForTheMenuWritesNothing(t *testing.T) {
	workDir := t.TempDir()

	hubOff := t.TempDir() // nothing stored: the advanced setting reads as its default, off
	hubOn := t.TempDir()  // pre-configured on, BEFORE the snapshot below, so this test proves
	// rendering itself writes nothing in either state -- not that the
	// initial "turn it on" write was somehow free.
	t.Setenv("AETHER_HUB_DIR", hubOn)
	if err := writeAdvancedCommandsSetting(true); err != nil {
		t.Fatalf("prepare pre-existing advanced-commands=on fixture: %v", err)
	}

	beforeWork := snapshotDirTree209(t, workDir)
	beforeHubOff := snapshotDirTree209(t, hubOff)
	beforeHubOn := snapshotDirTree209(t, hubOn)

	t.Setenv("AETHER_HUB_DIR", hubOff)
	_ = frontDoorHelpOutput199(t, workDir, 100)

	t.Setenv("AETHER_HUB_DIR", hubOn)
	_ = frontDoorHelpOutput199(t, workDir, 100)

	if got := snapshotDirTree209(t, workDir); got != beforeWork {
		t.Errorf("asking for help wrote to the working directory:\nbefore:\n%s\nafter:\n%s", beforeWork, got)
	}
	if got := snapshotDirTree209(t, hubOff); got != beforeHubOff {
		t.Errorf("asking for the default (off) help wrote to the hub:\nbefore:\n%s\nafter:\n%s", beforeHubOff, got)
	}
	if got := snapshotDirTree209(t, hubOn); got != beforeHubOn {
		t.Errorf("asking for the full (on) help wrote to the hub:\nbefore:\n%s\nafter:\n%s", beforeHubOn, got)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".aether")); !os.IsNotExist(err) {
		t.Errorf("asking for help created a data directory in the working directory: %v", err)
	}
}

// snapshotDirTree209 returns a stable, order-independent description of
// every entry under root (path plus content hash for files, a trailing
// slash marker for directories), for before/after comparison.
func snapshotDirTree209(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			entries = append(entries, rel+"/")
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(data)
		entries = append(entries, fmt.Sprintf("%s:%x", rel, sum))
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	sort.Strings(entries)
	return strings.Join(entries, "\n")
}
