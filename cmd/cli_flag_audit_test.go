package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// guardedAllowlistFiles are the four files the two shrink-only guards read —
// repo-root-relative, matching how a human reader would recognise them in
// prose. TestAllowlistPolicyNamesEveryGuardedFile derives its check from
// this slice rather than re-typing the paths inside the test body or trusting
// the policy document's own prose, so renaming or adding a guarded file here
// makes the policy doc go stale loudly instead of silently.
var guardedAllowlistFiles = []string{
	"cmd/testdata/orphan_allowlist.json",
	"cmd/testdata/orphan_allowlist_baseline.json",
	"cmd/cli_flag_audit_test.go",
	"cmd/testdata/flag_audit_skiplist_baseline.json",
	// Added by 172-09: the frozen, byte-identical, name-keyed snapshot of
	// orphan_allowlist_baseline.json as it stood immediately before the
	// path-key migration. TestPathMigrationDidNotWidenTolerance diffs the
	// migrated baseline against this file forever, so the migration itself
	// stays auditable — this is also a guarded file, and the policy document
	// must name it too.
	"cmd/testdata/orphan_allowlist_baseline_pre_path_migration.json",
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
	// Cobra adds `help` lazily, inside Execute. rootCmd.Commands() therefore
	// omits it until something in this process has run a command, so whether
	// this audit sees `help` registered depended on which other tests happened
	// to share its lane -- it passed for years and went red the moment the
	// suite re-sharded. Initialise it explicitly so the audit compares the
	// markdown corpus against the complete command tree every time.
	rootCmd.InitDefaultHelpCmd()

	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	// Root-resolved, not `..`-relative: a `..`-relative path silently
	// mis-scopes the corpus depending on the test binary's working
	// directory, and cannot be told apart from "directory legitimately
	// moved" (T-172-38). The three directories stay exactly the three D-06
	// puts in scope.
	markdownDirs := []string{
		filepath.Join(root, ".claude", "commands", "ant"),
		filepath.Join(root, ".opencode", "commands", "ant"),
		filepath.Join(root, ".aether", "docs", "command-playbooks"),
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

	// scannedFiles counts every .md file actually read into the audit. This
	// is the anti-vacuity floor's numerator: a directory that moves or is
	// renamed must not silently reduce this to zero and still pass — see
	// the floor check after the loop.
	scannedFiles := 0

	for _, dir := range markdownDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			// A declared input directory is not optional. Reported loudly
			// (not a silent `continue`) so moving or renaming a corpus
			// directory cannot switch this audit off with no code change
			// and no red test (T-172-38) — 172-04's shrink-only skip-list
			// guard is built on this test staying non-vacuous.
			t.Errorf("read declared corpus directory %s: %v — a declared input directory that cannot be read is a loud failure for this audit, not a silent skip", dir, err)
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			filePath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("read %s: %v — a .md entry the directory listing just reported must be readable", filePath, err)
				continue
			}
			scannedFiles++

			// filepath.Rel-derived repo-relative path, not entry.Name(): a
			// bare basename does not identify the file when the same name
			// exists in more than one corpus — build.md exists in both
			// .claude/commands/ant/ and .opencode/commands/ant/ — and
			// ROADMAP criterion 3 requires a failure to name the file
			// (T-172-39).
			relPath, relErr := filepath.Rel(root, filePath)
			if relErr != nil {
				relPath = filePath
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
							file:    relPath,
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
								file:    relPath,
								lineNum: lineNum + 1,
								message: fmt.Sprintf("subcommand %q missing flag --%s", canonical, flagName),
							})
						}
					}
				}
			}
		}
	}

	// Anti-vacuity floor (T-172-38): a guard that can pass while reading
	// zero files is indistinguishable from success. The measured count
	// today is 141 .md files across the three corpora (63 + 63 + 15); 120
	// leaves room for ordinary churn while still failing if a directory
	// disappears or comes back empty. scannedFiles=0, foundSubcommands=0 is
	// exactly the historical bug: a guard reading nothing passes forever.
	if scannedFiles < 120 || len(foundSubcommands) == 0 {
		t.Fatalf("flag audit read too little to trust: scannedFiles=%d (want >= 120), foundSubcommands=%d — a guard reading nothing passes forever, silently, the moment its declared corpus directories move, are renamed, or come back empty",
			scannedFiles, len(foundSubcommands))
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
	t.Logf("scanned %d files across %d corpora", scannedFiles, len(markdownDirs))
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

	rootCmd.InitDefaultHelpCmd() // see TestCLIFlagAudit: cobra adds `help` lazily
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

// TestAllowlistPolicyNamesEveryGuardedFile is T-172-15: a policy document
// that stops naming a file it guards is worse than no policy at all, because
// it makes the gap invisible instead of absent. This derives its expected
// file list from guardedAllowlistFiles above — never re-typed here, never
// copied from the document's own prose — and fails naming any guarded file
// whose path text is missing from .aether/docs/orphan-allowlist-policy.md.
func TestAllowlistPolicyNamesEveryGuardedFile(t *testing.T) {
	// Mirrors wiringGateGuardFiles's floor pattern
	// (subcommand_reachability_ratchet_test.go's TestWiringGuardsHaveNoRuntimeEscapeHatch):
	// a future edit that trims guardedAllowlistFiles must fail loudly here
	// rather than silently narrowing this policy-naming check.
	if len(guardedAllowlistFiles) < 5 {
		t.Fatalf("guardedAllowlistFiles has only %d entries — expected at least 5; a shrunk inventory would silently narrow this check", len(guardedAllowlistFiles))
	}

	data, err := os.ReadFile("../.aether/docs/orphan-allowlist-policy.md")
	if err != nil {
		t.Fatalf("read .aether/docs/orphan-allowlist-policy.md: %v", err)
	}
	doc := string(data)

	var missing []string
	for _, f := range guardedAllowlistFiles {
		if !strings.Contains(doc, f) {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf(".aether/docs/orphan-allowlist-policy.md does not name %d guarded file(s): %s\n"+
			"Add each path to the document, or the policy silently drifts out of step with what the guards actually read.",
			len(missing), strings.Join(missing, ", "))
	}
}
