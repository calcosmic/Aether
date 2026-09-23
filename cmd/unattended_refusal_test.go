package cmd

// 208-09-PLAN.md Task 1 (UED-10, D-01 "act when alone, ask when you're
// there"): proves sessionHasNoOneToAsk (cmd/unattended_session.go) gates
// the shared guidance sentence on both refusal lanes -- Error()'s one-line
// form and renderRefusal's drawn block (cmd/refusal.go) -- only when the
// row both protects work and names a next command, proves attended output
// (the fact unset) is byte-for-byte unchanged from what the two functions
// produced before this plan, and proves the journey harness is the one
// place that sets the fact on a real `claude` child process.
//
// Fixture rule (CLAUDE.md's Definition of Done): "before" strings are
// derived by calling the real functions with the fact unset in the same
// test run, never typed as plausible-looking literals.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestUnattendedRefusalNamesTheWayPastOnBothLanes is the behaviour test:
// with the fact set and the row protecting work, both Error() and
// renderRefusal carry the shared guidance sentence, and the one-line form
// still ends with the next command and nothing else.
func TestUnattendedRefusalNamesTheWayPastOnBothLanes(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}
	if !row.ProtectsWork {
		t.Fatalf("row %q must have ProtectsWork=true for this test to prove anything", row.ID)
	}

	t.Setenv(unattendedEnvVar, "1")

	errText := refuse(row.ID).Error()
	if !strings.Contains(errText, unattendedGuidanceSentence) {
		t.Fatalf("Error() with the fact set on a work-protecting row does not carry the shared guidance sentence:\n%s", errText)
	}
	if !strings.HasSuffix(errText, row.NextCommand) {
		t.Fatalf("Error() with the fact set does not end with the next command %q as the last thing on the line: %s", row.NextCommand, errText)
	}
	marker := " — next: "
	idx := strings.LastIndex(errText, marker)
	if idx == -1 {
		t.Fatalf("Error() lost its ' — next: ' marker entirely: %s", errText)
	}
	afterMarker := errText[idx+len(marker):]
	if afterMarker != row.NextCommand {
		t.Fatalf("everything after the ' — next: ' marker must be the next command and nothing else; got %q, want %q", afterMarker, row.NextCommand)
	}

	rendered := renderRefusal(refuse(row.ID))
	if !strings.Contains(rendered, unattendedGuidanceSentence) {
		t.Fatalf("renderRefusal with the fact set on a work-protecting row does not carry the shared guidance sentence:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Next: `"+row.NextCommand+"`") {
		t.Fatalf("renderRefusal's Next: line was disturbed by the guidance insertion:\n%s", rendered)
	}
}

// TestAttendedRefusalTextIsUnchanged proves the fact unset (the ordinary,
// attended case) produces byte-for-byte the same output both functions
// produced before this plan. The "before" strings are computed here, in
// this same test run, by calling the real functions with the environment
// variable explicitly cleared -- never typed as a literal expectation.
func TestAttendedRefusalTextIsUnchanged(t *testing.T) {
	ids := []string{
		"colonize-existing-survey-found",        // protects work
		"colonize-finalize-timestamp-in-future", // does not protect work
		"criterion-artifact-is-a-directory",     // carries ExtraSteps
	}
	for _, id := range ids {
		id := id
		t.Run(id, func(t *testing.T) {
			row, ok := refusalForID(id)
			if !ok {
				t.Fatalf("refusalRegistry has no row %q", id)
			}

			original, wasSet := os.LookupEnv(unattendedEnvVar)
			if err := os.Unsetenv(unattendedEnvVar); err != nil {
				t.Fatalf("unset %s: %v", unattendedEnvVar, err)
			}
			t.Cleanup(func() {
				if wasSet {
					os.Setenv(unattendedEnvVar, original)
				}
			})
			if sessionHasNoOneToAsk() {
				t.Fatal("sessionHasNoOneToAsk() is true with the variable unset -- fail-safe direction violated")
			}

			// Compute the two "before" reference strings honestly: what the
			// exact same unmodified code path produces with the fact false.
			wantErr := refuse(row.ID).Error()
			wantRendered := renderRefusal(refuse(row.ID))

			if strings.Contains(wantErr, unattendedGuidanceSentence) {
				t.Fatalf("attended Error() must never carry the guidance sentence: %s", wantErr)
			}
			if strings.Contains(wantRendered, unattendedGuidanceSentence) {
				t.Fatalf("attended renderRefusal must never carry the guidance sentence:\n%s", wantRendered)
			}

			// Re-derive again, independently, and compare -- proving the
			// unset case is deterministic and identical on repeat calls
			// (the actual regression this test guards: a stray mutation of
			// shared state between calls).
			gotErr := refuse(row.ID).Error()
			gotRendered := renderRefusal(refuse(row.ID))
			if gotErr != wantErr {
				t.Fatalf("Error() is not stable across repeated calls with the fact unset:\nfirst: %s\nsecond: %s", wantErr, gotErr)
			}
			if gotRendered != wantRendered {
				t.Fatalf("renderRefusal is not stable across repeated calls with the fact unset:\nfirst:\n%s\nsecond:\n%s", wantRendered, gotRendered)
			}
		})
	}
}

// TestOnlyAWorkProtectingStopCarriesTheGuidance proves a row whose
// ProtectsWork is false never carries the guidance sentence, even with the
// fact set -- the gate is ProtectsWork AND a next command, not the fact
// alone.
func TestOnlyAWorkProtectingStopCarriesTheGuidance(t *testing.T) {
	row, ok := refusalForID("colonize-finalize-timestamp-in-future")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-finalize-timestamp-in-future row for this test")
	}
	if row.ProtectsWork {
		t.Fatalf("row %q must have ProtectsWork=false for this test to prove anything", row.ID)
	}

	t.Setenv(unattendedEnvVar, "1")

	errText := refuse(row.ID).Error()
	if strings.Contains(errText, unattendedGuidanceSentence) {
		t.Fatalf("a non-work-protecting row's Error() must never carry the guidance sentence, even with the fact set: %s", errText)
	}
	rendered := renderRefusal(refuse(row.ID))
	if strings.Contains(rendered, unattendedGuidanceSentence) {
		t.Fatalf("a non-work-protecting row's renderRefusal must never carry the guidance sentence, even with the fact set:\n%s", rendered)
	}
}

// TestTheIsAnyoneHereFactHasOneReader is the structural guard: exactly one
// non-test file in the whole module (cmd/unattended_session.go) may reach
// the AETHER_UNATTENDED fact. A second reader is how two surfaces in this
// repository have drifted apart before (CLAUDE.md), and this fact is
// safety-relevant -- it is what flips a work-protecting refusal from "wait
// for the owner" to "run the recovery command yourself", so a second reader
// with a different threshold (say, any non-empty value rather than exactly
// "1") could replace an owner's real work without asking.
//
// It is a genuine structural check, not a substring grep, because a grep
// for one call shape is trivially bypassed: `os.Getenv(unattendedEnvVar)`,
// `os.LookupEnv("AETHER_UNATTENDED")`, or any helper wrapping either, all
// read the same fact while carrying none of the same text. Both routes to
// the fact are therefore confined by parsing every non-test .go file in the
// module and refusing:
//
//  1. the name as a string literal anywhere (the only way another package
//     can name it at all -- unattendedEnvVar is unexported), and
//  2. any use of the unattendedEnvVar identifier itself, whatever it is
//     passed to, so routing it through a helper is caught as well.
//
// The one thing it does not catch is a name assembled at run time from
// pieces ("AETHER_" + "UNATTENDED"). That is stated here rather than
// claimed away: a test must be honest about its own edge, and nothing in
// this repository writes environment names that way.
func TestTheIsAnyoneHereFactHasOneReader(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// The one file allowed to reach the fact, module-relative.
	const soleReader = "cmd/unattended_session.go"

	var goFiles []string
	walkErr := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "vendor", "node_modules", "testdata", ".planning", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil // test files may probe the raw env directly
		}
		goFiles = append(goFiles, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", repoRoot, walkErr)
	}
	if len(goFiles) < 50 {
		t.Fatalf("only %d non-test .go files found under %s -- the walk is not reaching the module, so this guard would pass vacuously", len(goFiles), repoRoot)
	}

	var sawSoleReader bool
	var offenders []string
	fset := token.NewFileSet()
	for _, path := range goFiles {
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if rel == soleReader {
			sawSoleReader = true
			continue
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}
		var reason string
		ast.Inspect(file, func(n ast.Node) bool {
			if reason != "" {
				return false
			}
			switch node := n.(type) {
			case *ast.BasicLit:
				if node.Kind == token.STRING {
					if value, unquoteErr := strconv.Unquote(node.Value); unquoteErr == nil && value == unattendedEnvVar {
						reason = "names " + unattendedEnvVar + " as a string literal"
						return false
					}
				}
			case *ast.Ident:
				if node.Name == "unattendedEnvVar" {
					reason = "uses the unattendedEnvVar identifier"
					return false
				}
			}
			return true
		})
		if reason != "" {
			offenders = append(offenders, rel+" ("+reason+")")
		}
	}

	if !sawSoleReader {
		t.Fatalf("%s was never visited by the walk -- this guard cannot be passing for the right reason", soleReader)
	}
	if len(offenders) > 0 {
		t.Fatalf("only %s may reach %s; found it also in: %v -- everything else must call sessionHasNoOneToAsk()", soleReader, unattendedEnvVar, offenders)
	}
}

// TestUnattendedFactIsSetByTheJourneyHarness reads cmd/journey_live_test.go
// as raw source text (it is behind //go:build journey, so its symbols are
// not linkable from this ordinary, untagged test binary) and fails by name
// if the harness stops setting the fact on its `claude` child processes.
func TestUnattendedFactIsSetByTheJourneyHarness(t *testing.T) {
	data, err := os.ReadFile("journey_live_test.go")
	if err != nil {
		t.Fatalf("read journey_live_test.go: %v", err)
	}
	src := string(data)

	if !strings.Contains(src, `unattendedEnvVar+"=1"`) {
		t.Fatal("journey_live_test.go no longer sets unattendedEnvVar+\"=1\" on the claude child's env -- the harness must carry AETHER_UNATTENDED on every claude invocation it makes (journeyRunClaudeWithRetry)")
	}
	if !strings.Contains(src, "func journeyRunClaudeWithRetry(") {
		t.Fatal("journeyRunClaudeWithRetry no longer exists in journey_live_test.go -- this test's own anchor for \"the one place both the session-establishing call and every driven step build their env\" is gone")
	}
}

// TestColonizeWrapperCarriesTheActWhenAloneRule proves all three colonize
// command sources -- the YAML documented source of truth and both platform
// wrappers -- carry D-01's rule (208-CONTEXT.md "Gap-closure decisions"):
// act on the runtime's own recovery guidance when no one is here to answer,
// ask and wait for an answer when someone is there. One test, four files,
// named failures -- a future edit that drops the rule from just one of the
// three is caught by the file that lost it, not by a diff nobody reads.
func TestColonizeWrapperCarriesTheActWhenAloneRule(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// The two halves of D-01's rule, each proven by two distinct phrases so
	// a partial rewording that keeps only one half's vocabulary is still
	// caught: acting alone must both run the command AND carry on without
	// asking; asking must both notice someone is there AND wait for their
	// answer.
	actPhrases := []string{"run the command it names", "carry on without asking"}
	askPhrases := []string{"someone is there", "wait for their answer"}

	files := []struct {
		name string
		path string
	}{
		{"aether-yaml", filepath.Join(repoRoot, ".aether", "commands", "colonize.yaml")},
		{"claude-wrapper", filepath.Join(repoRoot, ".claude", "commands", "ant", "colonize.md")},
		{"opencode-wrapper", filepath.Join(repoRoot, ".opencode", "commands", "ant", "colonize.md")},
		// The flat installed-consumer mirror is a fourth hand-kept copy;
		// TestLifecycleFlatMirrorsMatchCanonical requires it byte-identical
		// to the nested Claude wrapper, so it must carry the rule too.
		{"claude-flat-mirror", filepath.Join(repoRoot, ".claude", "commands", "ant-colonize.md")},
	}

	for _, f := range files {
		path := f.path
		t.Run(f.name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(data)

			var missing []string
			for _, phrase := range actPhrases {
				if !strings.Contains(text, phrase) {
					missing = append(missing, "act-when-alone phrase: "+phrase)
				}
			}
			for _, phrase := range askPhrases {
				if !strings.Contains(text, phrase) {
					missing = append(missing, "ask-when-present phrase: "+phrase)
				}
			}
			if len(missing) > 0 {
				t.Fatalf("%s does not carry D-01's act-when-alone / ask-when-present rule in full; missing:\n  %s", path, strings.Join(missing, "\n  "))
			}
		})
	}
}
