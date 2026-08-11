package cmd

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Phase 172, plan 05 (D-15).
//
// The named CI step "Verify subcommand wiring and CLI flag contracts" exists
// so a wiring failure reads as a wiring problem, not as one anonymous
// failure among hundreds inside the blanket `go test ./...` step. That only
// holds if the step's `-run` filter actually covers every guard test that
// exists. A step that exists in YAML with a filter matching nothing would
// report green forever while checking nothing — the same shape as a command
// whose only call site sat behind `2>/dev/null` (CLAUDE.md's own history).
// This test is what keeps the filter honest as the guard files grow.

// wiringGateStepName is the exact `- name:` value the CI step must carry.
const wiringGateStepName = "Verify subcommand wiring and CLI flag contracts"

// wiringGateGuardFiles are the guard files whose every top-level `func
// Test…` must be matched by the named CI step's `-run` regex. Declared here
// (in the file this plan owns) rather than reused from
// subcommand_reachability_ratchet_test.go's local guardFiles slice inside
// TestWiringGuardsHaveNoRuntimeEscapeHatch, which is unexported and scoped
// to a different, narrower purpose (escape-hatch scanning, not -run
// coverage) and does not include spawn_enforce_test.go or this file.
var wiringGateGuardFiles = []string{
	"subcommand_reachability_ratchet_test.go",
	"command_call_audit_test.go",
	"cli_flag_audit_test.go",
	"spawn_enforce_test.go",
	"ci_wiring_gate_test.go",
}

// runFlagArgRe pulls the single-quoted argument that follows `-run` out of a
// workflow `run:` line, e.g. `go test ./cmd -run 'Foo|Bar' -count=1 -v`.
var runFlagArgRe = regexp.MustCompile(`-run\s+'([^']*)'`)

// TestWiringGateStepRunsEveryWiringTest fails if the named CI step's -run
// filter has fallen behind the guard tests it exists to run, if the step is
// missing entirely, or if the blanket `go test ./...` release-gate step it
// sits alongside has been narrowed or removed.
func TestWiringGateStepRunsEveryWiringTest(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	workflow := string(data)

	// The named step adds legibility; it must never be mistaken for a
	// replacement that narrows CI coverage. Per D-15 both must run.
	if !strings.Contains(workflow, "go test ./... -count=1 -timeout 900s") {
		t.Fatalf("%s no longer contains the blanket `go test ./... -count=1 -timeout 900s` release-gate step — "+
			"the named wiring step adds legibility to an existing failure, it does not replace the coverage that finds it",
			workflowPath)
	}

	runArg, err := extractWiringGateRunArg(workflow)
	if err != nil {
		t.Fatalf("%v", err)
	}

	re, err := regexp.Compile(runArg)
	if err != nil {
		t.Fatalf("CI step %q has an uncompilable -run regex %q: %v", wiringGateStepName, runArg, err)
	}

	var testNames []string
	for _, f := range wiringGateGuardFiles {
		path := filepath.Join(repoRoot, "cmd", f)
		if _, statErr := os.Stat(path); statErr != nil {
			t.Fatalf("guard file %s does not exist on disk: %v", path, statErr)
		}
		testNames = append(testNames, testFuncNamesIn(t, path)...)
	}

	if len(testNames) < 8 {
		t.Fatalf("AST enumeration over %d guard file(s) found only %d top-level Test function(s) — expected at least 8; "+
			"a walk that silently finds nothing would pass forever, so this is treated as a fatal enumeration failure",
			len(wiringGateGuardFiles), len(testNames))
	}

	var uncovered []string
	for _, name := range testNames {
		// go test's own -run matching is unanchored substring semantics
		// (regexp.MatchString against the test name), so mirror that here
		// rather than requiring a full-string match.
		if !re.MatchString(name) {
			uncovered = append(uncovered, name)
		}
	}
	if len(uncovered) > 0 {
		sort.Strings(uncovered)
		t.Errorf("CI step %q's -run filter does not match %d guard test(s) — they would silently stop running under the named step "+
			"even though `go test ./...` still finds them, which is exactly the illegible-failure mode D-15 exists to prevent:\n  %s",
			wiringGateStepName, len(uncovered), strings.Join(uncovered, "\n  "))
	}
}

// extractWiringGateRunArg locates the step whose `- name:` value is exactly
// wiringGateStepName and returns its `run:` line's -run argument.
func extractWiringGateRunArg(workflow string) (string, error) {
	nameMarker := "- name: " + wiringGateStepName
	idx := strings.Index(workflow, nameMarker)
	if idx == -1 {
		return "", fmt.Errorf(".github/workflows/ci.yml has no step named %q — add it per D-15 so a wiring failure reads as a wiring problem, not one anonymous failure among hundreds", wiringGateStepName)
	}

	rest := workflow[idx+len(nameMarker):]
	runIdx := strings.Index(rest, "run:")
	if runIdx == -1 {
		return "", fmt.Errorf("CI step %q has no run: line", wiringGateStepName)
	}
	runLine := rest[runIdx:]
	if nl := strings.IndexByte(runLine, '\n'); nl != -1 {
		runLine = runLine[:nl]
	}

	m := runFlagArgRe.FindStringSubmatch(runLine)
	if m == nil {
		return "", fmt.Errorf("CI step %q's run line does not contain a `-run '<regex>'` argument: %s", wiringGateStepName, strings.TrimSpace(runLine))
	}
	return m[1], nil
}

// testFuncNamesIn enumerates every top-level `func Test…` name declared in
// path, using declNameAndBody (cmd/visual_writer_discipline_test.go) so a
// cobra-style `var xCmd = &cobra.Command{...}` block is walked the same way
// a function declaration is — though in practice every guard file's Test
// functions are plain FuncDecls.
func testFuncNamesIn(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var names []string
	for _, decl := range file.Decls {
		name, body := declNameAndBody(decl)
		if body == nil || name == "" {
			continue
		}
		if strings.HasPrefix(name, "Test") {
			names = append(names, name)
		}
	}
	return names
}
