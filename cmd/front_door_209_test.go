package cmd

// 209-03 (D-02, owner ruling 2026-09-24): the default front-door help
// screen is now a short, ten-command everyday menu laid out as two named
// groups ("Everyday commands" and "When you need them") instead of the full
// twenty-two-command catalogue -- gated behind the machine-wide
// advanced-commands setting (cmd/advanced_commands.go). The owner reviewed
// the real rendered screen at this plan's checkpoint task and amended
// 209-CONTEXT.md D-02's flat six-command proposal to this ten-command,
// two-group set; see 209-03-SUMMARY.md for the exact screen he approved.
// Nothing is deleted, disabled, or deprecated to get there: every demoted
// command stays registered and runnable when typed directly. These tests
// prove that guarantee against the real rendered screen
// (frontDoorHelpOutput199, reused from cmd/front_door_199_test.go) rather
// than against a description of it.

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

// approvedDefaultMenu209Entry names one command the owner ruled onto the
// default menu, plus the group it renders under.
type approvedDefaultMenu209Entry struct {
	group       string
	command     string
	description string
}

// approvedDefaultMenu209 is the owner's literal ruling on the default
// front-door menu, recorded 2026-09-24 after he reviewed the actual
// rendered screen at this plan's checkpoint task. It amends and supersedes
// 209-CONTEXT.md D-02's flat six-command proposal: he widened six to ten,
// laid them out as two named groups instead of one flat list, and approved
// this exact set of command names, order, grouping, and wording.
//
// This slice is hand-written -- deliberately NOT derived from
// frontDoorDefaultMenuGroups (cmd/root.go) -- so that a later change to
// that variable (adding, dropping, reordering, rewording, or silently
// regrouping an entry) fails THIS test by name instead of passing quietly.
var approvedDefaultMenu209 = []approvedDefaultMenu209Entry{
	{"Everyday commands", `/ant-go "<what you want>"`, "Do one piece of ordinary work, from a typo fix to a whole feature."},
	{"Everyday commands", `/ant-init "goal"`, "Start a new project with one goal."},
	{"Everyday commands", "/ant-status", "Show where the project actually stands right now."},
	{"Everyday commands", "/ant-continue", "Check what was built, then move on."},
	{"Everyday commands", "/ant-flags", "Show what is waiting on your decision."},
	{"Everyday commands", "/ant-resume", "Pick back up after a break."},
	{"Everyday commands", "/ant-seal", "Mark the work finished."},
	{"When you need them", "/ant-oracle", "Research a question properly, going over it until the answer is solid."},
	{"When you need them", `/ant-swarm "<bug>"`, "Chase down a confusing bug from four angles at once."},
	{"When you need them", "/ant-dream", "Let it look around the project and brainstorm what it notices."},
}

// TestDefaultMenuShowsOnlyTheApprovedSet proves the default (advanced
// setting off) screen shows exactly the owner's ruled ten commands, in the
// owner's ruled two groups and order, and no other registered wrapper
// command -- checked against the full set of registered wrapper names
// (wrapperCommandNames), not a hand-written exclusion list, so a wrapper
// added later is covered automatically.
func TestDefaultMenuShowsOnlyTheApprovedSet(t *testing.T) {
	root := t.TempDir()
	// Deliberately do NOT call frontDoorTurnOnAdvancedCommandsForTest: the
	// advanced setting must be genuinely off (the default) for this test.
	got := frontDoorHelpOutput199(t, root, 100)
	compact := strings.Join(strings.Fields(got), " ")

	// Group order: both approved group headings must appear, in order.
	groupOrder := []string{"Everyday commands", "When you need them"}
	last := -1
	for _, group := range groupOrder {
		index := strings.Index(got, group)
		if index < 0 {
			t.Fatalf("default help is missing approved group heading %q:\n%s", group, got)
		}
		if index <= last {
			t.Fatalf("approved group heading %q is out of order:\n%s", group, got)
		}
		last = index
	}

	approved := map[string]bool{}
	for _, entry := range approvedDefaultMenu209 {
		name, _, _ := strings.Cut(entry.command, " ")
		approved[name] = true
		want := strings.Join(strings.Fields(entry.command+"  "+entry.description), " ")
		if !strings.Contains(compact, want) {
			t.Errorf("default help is missing approved entry copy %q under group %q:\n%s", want, entry.group, got)
		}
	}

	// Scope the exclusion scan to the menu itself (from the first approved
	// group heading onward), not the whole screen: the opening lines above
	// it legitimately name /ant-init and /ant-go ("No project is set up
	// here yet... Start one with /ant-init \"what you want built\", or
	// just describe a job with /ant-go...") regardless of which entries
	// render below -- that onboarding prose is unconditional, not a
	// catalog exposure, and is unrelated to the owner's ten-entry ruling.
	menuStart := strings.Index(got, groupOrder[0])
	if menuStart < 0 {
		t.Fatalf("default help is missing its own first group heading %q:\n%s", groupOrder[0], got)
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
			// not an eleventh entry on it.
			continue
		}
		if tokens[name] {
			t.Errorf("default help exposes non-approved command %q, but the default menu must show only the owner's ten approved commands:\n%s", name, got)
		}
	}

	wantClosing := "Everything else still works exactly as before when you type it directly -- turn the full list back on for this machine with: aether advanced-commands set on"
	if !strings.Contains(compact, wantClosing) {
		t.Errorf("default help does not carry the owner-approved closing line naming the way to see the full list:\n%s", got)
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
// default (off) ten-command, two-group everyday menu and the full (on)
// catalogue -- are purely read-only: neither creates a data directory, a
// lock directory, or a preferences file as a side effect of merely being
// asked for.
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
