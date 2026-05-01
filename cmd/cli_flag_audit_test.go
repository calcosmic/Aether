package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

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

	// Subcommands known to be called from markdown but intentionally not
	// registered as direct subcommands (they are shell-only, aliases, or
	// handled by other mechanisms).
	skipSubcommands := map[string]bool{
		"verify-castes": true, // markdown-only command, no Go subcommand
	}

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

					if skipSubcommands[subcmd] {
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
		"suggest-approve":  "suggest-approve (called from build playbooks)",
		"versions":         "versions (called from build playbooks)",
		"chamber-compare":  "chamber-compare (called from tunnels.md)",
		"council":          "council (parent command, called from council.md)",
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
