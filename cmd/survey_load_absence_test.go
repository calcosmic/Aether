package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestSurveyLoadAbsentAndUncalled pins the resolution of LOUD-01: `survey-load`
// was resolved "done differently" from the other five broken call sites in
// this plan — instead of correcting its arguments, the command was deleted
// from the Go runtime entirely (RESEARCH.md). There is no flag form and no
// positional form that works; any reference to it is a defect by definition.
//
// This test is the command that fails the moment `survey-load` reappears,
// either as a registered Go subcommand or as an instruction in any live or
// reference markdown/YAML file, so that a future contributor cannot silently
// re-add the dead call the way it was silently added the first time.
func TestSurveyLoadAbsentAndUncalled(t *testing.T) {
	t.Run("NotRegistered", func(t *testing.T) {
		for _, c := range rootCmd.Commands() {
			if c.Name() == "survey-load" {
				t.Fatalf("survey-load is registered as a subcommand in the Go runtime (rootCmd.Commands()); this contradicts LOUD-01's resolution that survey-load was deleted, not fixed")
			}
			for _, alias := range c.Aliases {
				if alias == "survey-load" {
					t.Fatalf("survey-load is registered as an alias of %q in the Go runtime; this contradicts LOUD-01's resolution that survey-load was deleted, not fixed", c.Name())
				}
			}
		}
	})

	t.Run("NotReferenced", func(t *testing.T) {
		repoRoot, err := repoRootForCommandSourceTest()
		if err != nil {
			t.Fatalf("could not resolve repo root: %v", err)
		}

		dirs := []string{
			filepath.Join(repoRoot, ".claude", "commands", "ant"),
			filepath.Join(repoRoot, ".opencode", "commands", "ant"),
			filepath.Join(repoRoot, ".aether", "docs", "command-playbooks"),
			filepath.Join(repoRoot, ".aether", "commands"),
			filepath.Join(repoRoot, "colony", "playbooks"),
		}

		// Built via concatenation so this file's own literal cannot be
		// mistaken for an offending reference by a whole-repo grep.
		const needle = "aether " + "survey-load"

		type offense struct {
			file string
			line int
			text string
		}
		var offenses []offense

		for _, dir := range dirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				// Directory may not exist in every checkout/test environment.
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if !strings.HasSuffix(name, ".md") && !strings.HasSuffix(name, ".yaml") {
					continue
				}

				path := filepath.Join(dir, name)
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				lines := strings.Split(string(data), "\n")
				for lineNum, line := range lines {
					if strings.Contains(line, needle) {
						rel, relErr := filepath.Rel(repoRoot, path)
						if relErr != nil {
							rel = path
						}
						offenses = append(offenses, offense{
							file: rel,
							line: lineNum + 1,
							text: strings.TrimSpace(line),
						})
					}
				}
			}
		}

		if len(offenses) > 0 {
			var lines []string
			for _, o := range offenses {
				lines = append(lines, fmt.Sprintf("  %s:%d: %s", o.file, o.line, o.text))
			}
			sort.Strings(lines)
			t.Errorf("found %d reference(s) to %q, a command with no runtime implementation:\n%s",
				len(offenses), needle, strings.Join(lines, "\n"))
		}
	})
}
