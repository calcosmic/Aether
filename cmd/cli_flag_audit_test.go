package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// flagAuditSkipEntry is one tolerated skip-list entry. Same shape as
// orphanAllowlistEntry (cmd/subcommand_reachability_ratchet_test.go) —
// name, reason, owner_phase — so the two guarded lists (D-12) are
// queryable the same way, even though this file cannot import that
// unexported type's test-only helpers directly.
type flagAuditSkipEntry struct {
	Name       string `json:"name"`
	Reason     string `json:"reason"`
	OwnerPhase string `json:"owner_phase"`
}

// skipSubcommands lists subcommands known to be called from markdown but
// intentionally not registered as direct subcommands (they are shell-only,
// aliases, or handled by other mechanisms). Promoted from a
// TestCLIFlagAudit-local map[string]bool to a package-level declaration
// (D-12) so TestFlagAuditSkipListOnlyShrinks can read it without
// duplicating the list.
var skipSubcommands = []flagAuditSkipEntry{
	{Name: "verify-castes", Reason: "markdown-only-shorthand", OwnerPhase: "RECLAIM"},     // markdown-only command, no Go subcommand
	{Name: "pending-decisions", Reason: "markdown-only-shorthand", OwnerPhase: "RECLAIM"}, // playbook shorthand; actual Go subcommands are pending-decision-{add,list,resolve}
}

// skipSubcommandNames returns skipSubcommands as a name-only set, the shape
// TestCLIFlagAudit's scan loop needs for its lookup.
func skipSubcommandNames() map[string]bool {
	names := make(map[string]bool, len(skipSubcommands))
	for _, e := range skipSubcommands {
		names[e.Name] = true
	}
	return names
}

// TestCLIFlagAudit systematically compares markdown CLI calls against Go
// registrations. This resolves RESEARCH.md open questions 1 and 2 with
// concrete evidence.
//
// It scans .claude/commands/ant/*.md, .opencode/commands/ant/*.md, and
// .aether/docs/command-playbooks/*.md for "aether <subcommand> --flag value"
// patterns, then verifies each subcommand and flag exists in the Go runtime.
func TestCLIFlagAudit(t *testing.T) {
	markdownDirs := []string{
		"../.claude/commands/ant/",
		"../.opencode/commands/ant/",
		"../.aether/docs/command-playbooks/",
	}

	// Regex to extract "aether <subcommand>" calls with optional flags.
	// Matches patterns like:
	//   aether pheromone-write --type "FOCUS" --content "..."
	//   aether build $ARGUMENTS --plan-only
	re := regexp.MustCompile(`aether\s+([\w][\w-]*)\s+((?:--[\w][\w-]*(?:=\S*|\s+\S*)?\s*)*)`)

	skipSet := skipSubcommandNames()

	// Build lookup: subcommand name -> set of registered flags
	registered := make(map[string]map[string]bool)
	registeredAliases := make(map[string]string) // alias -> canonical
	for _, c := range rootCmd.Commands() {
		flags := make(map[string]bool)
		c.Flags().VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = true
		})
		c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			flags[f.Name] = true
		})
		registered[c.Name()] = flags

		// Track aliases so we can resolve "flag-create" -> "flag-add"
		for _, alias := range c.Aliases {
			registeredAliases[alias] = c.Name()
		}
	}

	type mismatch struct {
		file    string
		lineNum int
		message string
	}
	var mismatches []mismatch

	// Track which subcommands were found in markdown
	foundSubcommands := make(map[string]bool)

	for _, dir := range markdownDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // directory may not exist in test environment
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(dir + entry.Name())
			if err != nil {
				continue
			}
			lines := strings.Split(string(data), "\n")
			for lineNum, line := range lines {
				matches := re.FindAllStringSubmatch(line, -1)
				for _, m := range matches {
					subcmd := m[1]
					flagsStr := m[2]

					foundSubcommands[subcmd] = true

					if skipSet[subcmd] {
						continue
					}

					// Check subcommand exists (resolve aliases)
					canonical := subcmd
					flagSet, exists := registered[subcmd]
					if !exists {
						if canonicalName, isAlias := registeredAliases[subcmd]; isAlias {
							canonical = canonicalName
							flagSet = registered[canonicalName]
							exists = true
						}
					}
					if !exists {
						mismatches = append(mismatches, mismatch{
							file:    entry.Name(),
							lineNum: lineNum + 1,
							message: fmt.Sprintf("subcommand %q not registered in Go runtime", subcmd),
						})
						continue
					}

					// Extract --flag names from the flags portion
					flagRe := regexp.MustCompile(`--([\w][\w-]*)`)
					flagMatches := flagRe.FindAllStringSubmatch(flagsStr, -1)
					for _, fm := range flagMatches {
						flagName := fm[1]
						if !flagSet[flagName] {
							mismatches = append(mismatches, mismatch{
								file:    entry.Name(),
								lineNum: lineNum + 1,
								message: fmt.Sprintf("subcommand %q missing flag --%s", canonical, flagName),
							})
						}
					}
				}
			}
		}
	}

	if len(mismatches) > 0 {
		// Deduplicate mismatches (same message can appear from many lines)
		seen := make(map[string]bool)
		var unique []string
		for _, mm := range mismatches {
			key := mm.message
			if !seen[key] {
				seen[key] = true
				unique = append(unique, fmt.Sprintf("  %s:%d: %s", mm.file, mm.lineNum, mm.message))
			}
		}
		sort.Strings(unique)
		t.Errorf("CLI flag audit found %d unique mismatches:\n%s",
			len(unique), strings.Join(unique, "\n"))
	}

	// Log coverage summary
	t.Logf("Audit coverage: %d unique subcommands found in markdown", len(foundSubcommands))
	t.Logf("Registered subcommands in Go runtime: %d", len(registered))
}

// TestCLIFlagAuditSubcommandsRegistered verifies the 5 specific subcommands
// that this plan (71-02) is responsible for registering.
func TestCLIFlagAuditSubcommandsRegistered(t *testing.T) {
	required := map[string]string{
		"suggest-approve": "suggest-approve (called from build playbooks)",
		"versions":        "versions (called from build playbooks)",
		"chamber-compare": "chamber-compare (called from tunnels.md)",
		"council":         "council (parent command, called from council.md)",
	}

	registered := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		registered[c.Name()] = true
	}

	// Check flag-create alias separately
	flagCreateFound := false
	for _, c := range rootCmd.Commands() {
		for _, alias := range c.Aliases {
			if alias == "flag-create" {
				flagCreateFound = true
			}
		}
	}

	for name, desc := range required {
		if !registered[name] {
			t.Errorf("missing subcommand: %s (%s)", name, desc)
		}
	}
	if !flagCreateFound {
		t.Error("missing alias: flag-create should be an alias for flag-add")
	}
}

// loadFlagAuditSkipBaseline reads the committed baseline JSON for the flag
// audit's skip list. Same shape as loadOrphanAllowlist
// (cmd/subcommand_reachability_ratchet_test.go, same package) — a
// [{"name","reason","owner_phase"}, ...] array read by relative testdata
// path — implemented as its own function rather than a shared call because
// that file's shrink-only diff (TestOrphanAllowlistOnlyShrinks) is inlined
// in the test body, not extracted into a callable comparator this file could
// import. Documented in this plan's SUMMARY as a follow-up: the two
// set-membership diffs below and in TestOrphanAllowlistOnlyShrinks must be
// kept in step by hand until a later phase unifies them.
func loadFlagAuditSkipBaseline(t *testing.T, path string) []flagAuditSkipEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []flagAuditSkipEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

// TestFlagAuditSkipListOnlyShrinks is D-12: the flag audit's skipSubcommands
// list gets the identical shrink-only treatment
// TestOrphanAllowlistOnlyShrinks (cmd/subcommand_reachability_ratchet_test.go)
// applies to the orphan allowlist — a pure set-membership diff against a
// committed baseline copy. Same rules, same reasons: no count comparison (a
// one-out-one-in swap must fail), no git-history comparison (CI
// shallow-clones), no override field, no sign-off path, no environment
// variable (D-10/D-11, applied here per D-12).
func TestFlagAuditSkipListOnlyShrinks(t *testing.T) {
	baseline := loadFlagAuditSkipBaseline(t, "testdata/flag_audit_skiplist_baseline.json")

	baselineNames := map[string]bool{}
	for _, e := range baseline {
		baselineNames[e.Name] = true
	}

	var added []string
	for _, e := range skipSubcommands {
		if !baselineNames[e.Name] {
			added = append(added, e.Name)
		}
	}

	if len(added) > 0 {
		sort.Strings(added)
		t.Errorf("%d command(s) were added to the tolerated skip list without being added to the committed baseline: %s\n"+
			"The skip list may only shrink. Make the documented call correct, or delete its entry — do not edit the baseline to make this pass.",
			len(added), strings.Join(added, ", "))
	}
}
