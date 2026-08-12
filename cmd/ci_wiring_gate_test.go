package cmd

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// Phase 172, plan 05 (D-15), hardened by plan 07.
//
// The named CI step "Verify subcommand wiring and CLI flag contracts" exists
// so a wiring failure reads as a wiring problem, not as one anonymous
// failure among hundreds inside the blanket `go test ./...` step. That only
// holds if the step's `-run` filter actually covers every guard test that
// exists, AND if the blanket `go test ./...` release-gate step it sits
// alongside is still present as a step that can actually fail the build —
// not merely as a text substring somewhere in the workflow file. Plan 05's
// original check was an unscoped whole-file substring test for the literal
// blanket command, which the non-blocking `Test summary` step's
// `|| echo 'unknown'` decoy satisfies even after the real
// gate step is deleted (independently reproduced during 172-VERIFICATION.md
// gap 2). This file's blanketReleaseGateProblem replaces that with a
// step-scoped, failing-capability-aware check.

// wiringGateStepName is the exact `- name:` value the CI step must carry.
const wiringGateStepName = "Verify subcommand wiring and CLI flag contracts"

// blanketGateStepName is the exact `- name:` value of the blanket
// release-gate step that must remain present alongside the named wiring
// step, and must remain structurally capable of failing the build.
const blanketGateStepName = "Run Go tests"

// blanketGateRunCommand is the exact command the blanket release-gate
// step's run line must EQUAL, once the leading `run:` token is stripped —
// not merely contain. This is a whitelist of the one accepted command, not
// a blocklist of known-bad spellings, and it exists only as a cheap sanity
// check on the run line's shape.
//
// It is NOT the proof that the release gate cannot be neutered, and must
// never be mistaken for one — that is exactly the mistake that produced two
// failed attempts at this criterion (172-VERIFICATION.md GAP A). The proof
// is TestReleaseGateCommandFailsATreeWithAFailingTest, which executes this
// command, read live from ci.yml, against a passing and a deliberately
// failing probe tree and requires the exit codes to discriminate. A
// weakened pin (this constant itself edited to something permissive) would
// still be caught by that execution-based test, because it reads the
// command from ci.yml at test time rather than trusting this constant.
const blanketGateRunCommand = "go test ./... -count=1 -timeout 900s"

// nameMarkerPrefix is the YAML step-name marker every step block begins
// with. Declared once here so stepBlock is the single step locator in this
// package — a second, independently-drifting scan is how the original
// unanchored `strings.Index(workflow, nameMarker)` in extractWiringGateRunArg
// ended up being duplicated logic instead of shared logic.
const nameMarkerPrefix = "- name: "

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
// filter has fallen behind the guard tests it exists to run (in either
// direction — a guard test missing from the filter, or a stale alternative
// in the filter matching no guard test), if the step is missing entirely or
// carries an `if:`/`continue-on-error` of its own, or if the blanket
// `go test ./...` release-gate step it sits alongside has been narrowed,
// disabled, or removed. "Narrowed or disabled" is checked structurally by
// blanketReleaseGateProblem — the step must exist under its exact name, its
// run line must equal the real command exactly, and the step must have no
// `if:` condition, no `continue-on-error`, and no exit-status-swallowing
// fallback such as `|| true` — not merely have the command's text appear
// somewhere in the file, which a non-blocking decoy step can also satisfy.
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
	// replacement that narrows CI coverage. Per D-15 both must run, and the
	// blanket step must be present as a gate that can fail, not present as a
	// string.
	if err := blanketReleaseGateProblem(workflow); err != nil {
		t.Fatalf("%v", err)
	}

	// The named wiring step itself must also be held to the failing-
	// capability standard, per D-15's own stated purpose (WR-02 / T-172-57)
	// — a `continue-on-error` here would pass every other check while the
	// step's own failure never fails the job.
	wiringBlock, err := stepBlock(workflow, wiringGateStepName)
	if err != nil {
		t.Fatalf("%v — add it per D-15 so a wiring failure reads as a wiring problem, not one anonymous failure among hundreds", err)
	}
	if err := stepCanFailTheBuild(wiringBlock, wiringGateStepName); err != nil {
		t.Fatalf("%v", err)
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

	// Anti-vacuity floor (mirrors the convention cli_flag_audit_test.go:225
	// and command_call_audit_test.go:1329 already use): measured 30 top-level
	// Test functions across the five guard files as of plan 172-11; 20 is set
	// just under that so ordinary churn does not trip it while a silent AST
	// walk, or two-thirds of the guard tests being deleted, still does.
	if len(testNames) < 20 {
		t.Fatalf("AST enumeration over %d guard file(s) found only %d top-level Test function(s) — expected at least 20 (measured 30); "+
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

	// The reverse direction (WR-04 / T-172-56): every alternative actually
	// present in the filter must match at least one real guard test name.
	// `go test -run` exits 0 when a pattern matches nothing, so a renamed or
	// deleted guard test left as a stale alternative silently stops running
	// under the named step with nothing going red.
	var stale []string
	for _, alt := range strings.Split(runArg, "|") {
		altRe, compileErr := regexp.Compile(alt)
		if compileErr != nil {
			t.Fatalf("CI step %q's -run filter alternative %q does not compile as a regex: %v", wiringGateStepName, alt, compileErr)
		}
		matched := false
		for _, name := range testNames {
			if altRe.MatchString(name) {
				matched = true
				break
			}
		}
		if !matched {
			stale = append(stale, alt)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("CI step %q's -run filter contains %d alternative(s) that match no guard test — go test -run exits 0 when a pattern matches "+
			"nothing, so a renamed or deleted guard test left in the filter would silently stop running with nothing going red:\n  %s",
			wiringGateStepName, len(stale), strings.Join(stale, "\n  "))
	}
}

// stepBlock locates the step whose `- name:` value is exactly stepName,
// anchored so a comment can never satisfy it, and returns the block of
// workflow text running from that name line up to (but excluding) whichever
// comes first: the next line at the same step indentation that begins a
// `- name:` entry, or the first subsequent non-blank line whose indentation
// is strictly less than the step marker's own indentation.
//
// Anchoring to end-of-line is load-bearing: "Run Go tests" is a strict
// prefix of "Run Go tests with race detection" (ci.yml:44), so an
// unanchored substring search would resolve to the race step the moment the
// real gate step is deleted — replacing one decoy with another and
// re-creating the exact bug this function exists to fix.
//
// The marker must ALSO be preceded on its own line by pure whitespace
// (CR-02). Without that, `# - name: Run Go tests` — a step commented out of
// existence — satisfies a plain substring search and is counted as present.
// A match whose line prefix is not pure whitespace is rejected and the
// search continues past it, so a real step further down (or nothing at all)
// is what actually gets found.
//
// Bounding the block by indentation, rather than running to end of file
// when no next same-indent marker exists, is what stops the `if:` /
// `continue-on-error` scan in stepCanFailTheBuild from reading an unrelated
// LATER step and misdiagnosing it as belonging to the step being checked —
// the false "CI step \"Run Go tests\" has a conditional if: always()"
// produced against a deleted step, recorded as IN-05.
//
// This is the single step locator in the package; extractWiringGateRunArg,
// blanketReleaseGateProblem, and releaseGateCommandFromWorkflow all call it
// rather than each scanning the workflow text independently.
func stepBlock(workflow, stepName string) (string, error) {
	marker := nameMarkerPrefix + stepName

	searchFrom := 0
	for {
		rel := strings.Index(workflow[searchFrom:], marker)
		if rel == -1 {
			return "", fmt.Errorf(".github/workflows/ci.yml has no step named %q", stepName)
		}
		absIdx := searchFrom + rel
		afterEnd := absIdx + len(marker)

		// The marker must be followed immediately by end-of-line (a newline
		// or end of file) — otherwise this is a prefix collision like
		// "Run Go tests" matching inside "Run Go tests with race
		// detection", and the search continues past it.
		if afterEnd != len(workflow) && workflow[afterEnd] != '\n' {
			searchFrom = afterEnd
			continue
		}

		lineStart := strings.LastIndexByte(workflow[:absIdx], '\n') + 1
		indent := workflow[lineStart:absIdx]

		// The marker must be the first non-whitespace content on its line.
		// A "# - name: ..." comment has non-whitespace (the "#") before the
		// marker and is not a step — keep searching past it rather than
		// treating the comment as a present step (CR-02).
		if strings.TrimSpace(indent) != "" {
			searchFrom = afterEnd
			continue
		}

		rest := workflow[absIdx:]
		return rest[:blockEnd(rest, indent)], nil
	}
}

// blockEnd returns the offset within rest — which begins at the step
// marker's own `- name:` line — where that step's block ends. The block
// ends at whichever comes first: the next line at the same indent (indent)
// beginning `- name: `, or the first subsequent non-blank line whose
// indentation is strictly less than indent. If neither occurs, the block
// runs to the end of rest.
func blockEnd(rest, indent string) int {
	firstNL := strings.IndexByte(rest, '\n')
	if firstNL == -1 {
		return len(rest)
	}

	nextMarker := indent + nameMarkerPrefix
	pos := firstNL + 1
	for pos < len(rest) {
		lineEnd := strings.IndexByte(rest[pos:], '\n')
		var line string
		if lineEnd == -1 {
			line = rest[pos:]
		} else {
			line = rest[pos : pos+lineEnd]
		}

		if strings.TrimSpace(line) != "" {
			if strings.HasPrefix(line, nextMarker) {
				return pos
			}
			lineIndent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			if len(lineIndent) < len(indent) {
				return pos
			}
		}

		if lineEnd == -1 {
			return len(rest)
		}
		pos += lineEnd + 1
	}
	return len(rest)
}

// runLineOf extracts the single-line `run:` value out of a step block,
// trimmed to just that line (block may continue past it with further step
// keys or the next step).
func runLineOf(block string) (string, bool) {
	runIdx := strings.Index(block, "run:")
	if runIdx == -1 {
		return "", false
	}
	runLine := block[runIdx:]
	if nl := strings.IndexByte(runLine, '\n'); nl != -1 {
		runLine = runLine[:nl]
	}
	return runLine, true
}

// stepCanFailTheBuild returns nil if block — the text of one step, as
// returned by stepBlock — is structurally capable of failing the build: no
// `if:` condition and no `continue-on-error`. A step wearing either
// property can be arranged never to redden the job regardless of what its
// run line does, so neither is a gate. Applied to both the blanket
// release-gate step and the named wiring step (WR-02 / T-172-57) — D-15's
// stated purpose for the named step is legibility, which a
// `continue-on-error` on that step alone would already silently defeat.
func stepCanFailTheBuild(block, stepName string) error {
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if:") {
			return fmt.Errorf("CI step %q has a conditional %s — a conditional step can be arranged not to run, so it cannot be relied on to gate", stepName, trimmed)
		}
	}

	if strings.Contains(block, "continue-on-error") {
		return fmt.Errorf("CI step %q has continue-on-error set — a step whose failure does not fail the job is not a gate", stepName)
	}

	return nil
}

// blanketReleaseGateProblem returns nil if the blanket release-gate step
// named exactly blanketGateStepName is present in workflow, its run line
// (once the leading `run:` token is stripped) is EXACTLY
// blanketGateRunCommand — no appended arguments, redirections, or
// fallbacks — and the step is structurally capable of failing the build.
// Any other outcome returns a distinct, named error describing exactly
// which of those properties is missing.
func blanketReleaseGateProblem(workflow string) error {
	block, err := stepBlock(workflow, blanketGateStepName)
	if err != nil {
		return fmt.Errorf("%v — the blanket release gate has been removed or renamed; the named wiring step adds legibility to an existing failure, it does not replace the coverage that finds it", err)
	}

	runLine, ok := runLineOf(block)
	if !ok {
		return fmt.Errorf("CI step %q has no run: line", blanketGateStepName)
	}
	trimmedRunLine := strings.TrimSpace(runLine)
	cmd := strings.TrimSpace(strings.TrimPrefix(trimmedRunLine, "run:"))

	if strings.HasPrefix(cmd, "#") {
		return fmt.Errorf("CI step %q's run line is commented out: %s", blanketGateStepName, trimmedRunLine)
	}
	if cmd != blanketGateRunCommand {
		return fmt.Errorf("CI step %q's run line must be exactly %q (no appended arguments, redirections, or fallbacks) — found: %s", blanketGateStepName, blanketGateRunCommand, cmd)
	}

	if err := stepCanFailTheBuild(block, blanketGateStepName); err != nil {
		return err
	}

	return nil
}

// TestBlanketGateCheckRejectsADecoyStep is a hermetic, table-driven test
// over blanketReleaseGateProblem using synthetic workflow strings built in
// the test body, so the guarantee that a decoy step cannot satisfy the
// check keeps holding on every CI run rather than resting on a one-time
// transcript.
func TestBlanketGateCheckRejectsADecoyStep(t *testing.T) {
	const stepsPreamble = "jobs:\n  go:\n    steps:\n"

	// decoyStep is the real, verbatim shape of ci.yml's non-blocking `Test
	// summary` step: `if: always()` and a `|| echo 'unknown'` fallback that
	// wraps the blanket command inside an echo, so its exit status can never
	// fail the job. Plan 05's original whole-file `strings.Contains` check
	// was satisfied by this decoy alone, with no `Run Go tests` step
	// present anywhere — that is the bug this test exists to keep fixed.
	const decoyStep = `      - name: Test summary
        if: always()
        run: |
          echo "Go tests: $(go test ./... -count=1 -timeout 900s -v 2>&1 | grep -c '--- PASS' || echo 'unknown') passed"
`

	const raceOnlyStep = `      - name: Run Go tests with race detection
        run: go test ./... -race -count=1 -timeout 2400s
`

	const happyStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s
`

	const ifAlwaysStep = `      - name: Run Go tests
        if: always()
        run: go test ./... -count=1 -timeout 900s
`

	const continueOnErrorStep = `      - name: Run Go tests
        continue-on-error: true
        run: go test ./... -count=1 -timeout 900s
`

	const fallbackStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s || true
`

	// The five remaining CR-01 bypasses of the old "contains" check — each
	// appends something after the required substring, which a contains
	// check cannot see but the new exact-equality check must reject.
	const semicolonTrueStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s; true
`

	const orExitZeroStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s || exit 0
`

	const pipeCatStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s | cat
`

	const secondRunFlagStep = `      - name: Run Go tests
        run: go test ./... -count=1 -timeout 900s -run TestNothing
`

	// The required text is present but sits inside a comment; the actual
	// command the shell would run is "echo skip" — CR-01's shell-comment
	// case.
	const commentedRunLineStep = `      - name: Run Go tests
        run: echo skip # go test ./... -count=1 -timeout 900s
`

	// A fully commented-out step in a minimal workflow with no other step
	// at all (CR-02's core reproduction).
	const commentedStepAlone = `      # - name: Run Go tests
      #   run: go test ./... -count=1 -timeout 900s
`

	// The commented-out step is followed by a REAL later step carrying its
	// own if: always() — reproducing IN-05's misdiagnosis directly. Before
	// the indentation-bounded block fix, stepBlock had no real match here
	// either, but if it had spuriously matched the comment line, an
	// unbounded block would have run to EOF and picked up this later step's
	// if: always(), reporting "has a conditional if: always()" for a step
	// that had been deleted. wantNotSub asserts that misdiagnosis cannot
	// happen.
	const commentedStepThenUnrelatedIfStep = `      # - name: Run Go tests
      #   run: go test ./... -count=1 -timeout 900s
      - name: Test summary
        if: always()
        run: echo done
`

	tests := []struct {
		name       string
		workflow   string
		wantErr    bool
		wantSub    string
		wantNotSub string
	}{
		{
			name:     "decoy step alone does not satisfy the blanket gate",
			workflow: stepsPreamble + decoyStep,
			wantErr:  true,
			wantSub:  "Run Go tests",
		},
		{
			name:     "race-detection step name is a superstring, not a match",
			workflow: stepsPreamble + raceOnlyStep,
			wantErr:  true,
			wantSub:  "Run Go tests",
		},
		{
			name:     "happy path: real gate present and capable of failing",
			workflow: stepsPreamble + happyStep,
			wantErr:  false,
		},
		{
			name:     "if: always() on the real gate defeats it",
			workflow: stepsPreamble + ifAlwaysStep,
			wantErr:  true,
			wantSub:  "if: always()",
		},
		{
			name:     "continue-on-error on the real gate defeats it",
			workflow: stepsPreamble + continueOnErrorStep,
			wantErr:  true,
			wantSub:  "continue-on-error",
		},
		{
			name:     "trailing fallback swallows the exit status",
			workflow: stepsPreamble + fallbackStep,
			wantErr:  true,
			wantSub:  "|| true",
		},
		{
			name:     "appended semicolon-true is invisible to a contains check but not to equality",
			workflow: stepsPreamble + semicolonTrueStep,
			wantErr:  true,
			wantSub:  "; true",
		},
		{
			name:     "appended or-exit-zero is invisible to a contains check but not to equality",
			workflow: stepsPreamble + orExitZeroStep,
			wantErr:  true,
			wantSub:  "|| exit 0",
		},
		{
			name:     "piping through cat is invisible to a contains check but not to equality",
			workflow: stepsPreamble + pipeCatStep,
			wantErr:  true,
			wantSub:  "| cat",
		},
		{
			name:     "a second appended -run flag is invisible to a contains check but not to equality",
			workflow: stepsPreamble + secondRunFlagStep,
			wantErr:  true,
			wantSub:  "-run TestNothing",
		},
		{
			name:     "the required text is present but sits inside a comment",
			workflow: stepsPreamble + commentedRunLineStep,
			wantErr:  true,
			wantSub:  "echo skip",
		},
		{
			name:     "a fully commented-out step in a minimal workflow with no other step",
			workflow: stepsPreamble + commentedStepAlone,
			wantErr:  true,
			wantSub:  "Run Go tests",
		},
		{
			name:       "a commented-out step followed by an unrelated later step's if: is not misdiagnosed as that step's condition",
			workflow:   stepsPreamble + commentedStepThenUnrelatedIfStep,
			wantErr:    true,
			wantSub:    "Run Go tests",
			wantNotSub: "if: always()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := blanketReleaseGateProblem(tt.workflow)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected a non-nil error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantSub) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantSub)
				}
				if tt.wantNotSub != "" && strings.Contains(err.Error(), tt.wantNotSub) {
					t.Fatalf("error %q incorrectly contains %q — this is exactly the misdiagnosis bug (IN-05) being guarded against", err.Error(), tt.wantNotSub)
				}
			} else if err != nil {
				t.Fatalf("expected nil, got: %v", err)
			}
		})
	}
}

// extractWiringGateRunArg locates the step whose `- name:` value is exactly
// wiringGateStepName and returns its `run:` line's -run argument.
//
// Three distinct failure modes are named rather than folded into one
// generic error (CR-03):
//   - zero `-run '<regex>'` occurrences on the line;
//   - MORE than one — `go test` honours only the LAST `-run` flag it is
//     given, so a second, appended `-run` would silently make this guard
//     validate a filter that never actually executes;
//   - a run line that does not invoke `go test ./cmd` at all, so an
//     unrelated command that merely contains the right-looking text (e.g.
//     `echo "-run '...'"`) cannot satisfy this check.
func extractWiringGateRunArg(workflow string) (string, error) {
	block, err := stepBlock(workflow, wiringGateStepName)
	if err != nil {
		return "", fmt.Errorf("%v — add it per D-15 so a wiring failure reads as a wiring problem, not one anonymous failure among hundreds", err)
	}

	runLine, ok := runLineOf(block)
	if !ok {
		return "", fmt.Errorf("CI step %q has no run: line", wiringGateStepName)
	}
	trimmedRunLine := strings.TrimSpace(runLine)

	if !strings.Contains(runLine, "go test ./cmd ") {
		return "", fmt.Errorf("CI step %q's run line does not invoke `go test ./cmd`: %s", wiringGateStepName, trimmedRunLine)
	}

	ms := runFlagArgRe.FindAllStringSubmatch(runLine, -1)
	if len(ms) == 0 {
		return "", fmt.Errorf("CI step %q's run line does not contain a `-run '<regex>'` argument: %s", wiringGateStepName, trimmedRunLine)
	}
	if len(ms) > 1 {
		return "", fmt.Errorf("CI step %q's run line carries %d `-run` flags; go test honours only the last, so this guard would validate a filter that never executes: %s", wiringGateStepName, len(ms), trimmedRunLine)
	}
	return ms[0][1], nil
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

// Plan 172-10 (gap closure, third attempt at ROADMAP success criterion 4's
// durability half).
//
// Two prior attempts asserted on the TEXT of .github/workflows/ci.yml and
// were each defeated in turn: attempt 1 was a whole-file substring search,
// defeated by a decoy line elsewhere in the file; attempt 2 (plan 172-07,
// blanketReleaseGateProblem above) was a step-scoped substring/first-match
// check, defeated by five independently reproduced mutations (`; true`,
// `|| exit 0`, an appended `-run` filter, `| cat`, and a commented-out run
// line — 172-VERIFICATION.md GAP A). A third, cleverer text check would only
// be defeated by a sixth mutation nobody has enumerated yet, because a
// blocklist of known-bad spellings cannot establish the property the
// criterion actually asks for.
//
// This block instead EXECUTES the release gate's own command — read from
// ci.yml at test time, never a hardcoded copy — against two throwaway probe
// modules: one with a passing test, one with a deliberately failing one.
// gateCommandDiscriminates requires the command to exit 0 on the passing
// module and non-zero on the failing one. Every mutation above collapses
// that discrimination and is therefore caught by running the command, not
// by recognising its spelling — including whatever the sixth mutation turns
// out to be. TestGateProbeCatchesEveryKnownGateNeutering (below, plan
// 172-10 task 2) is evidence that this mechanism works; it is not the
// mechanism itself.

// releaseGateCommandFromWorkflow locates the blanket release-gate step
// (blanketGateStepName) via the existing stepBlock/runLineOf locators and
// returns its run line's command verbatim — with only the leading `run:`
// token and surrounding whitespace stripped. Nothing else is filtered,
// rewritten, or sanitised: the point is to run exactly what the workflow
// actually runs, so the workflow and this guard cannot silently drift apart.
func releaseGateCommandFromWorkflow(workflow string) (string, error) {
	block, err := stepBlock(workflow, blanketGateStepName)
	if err != nil {
		return "", fmt.Errorf("%v — cannot extract the release gate's own command to execute it", err)
	}

	runLine, ok := runLineOf(block)
	if !ok {
		return "", fmt.Errorf("CI step %q has no run: line", blanketGateStepName)
	}

	cmd := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(runLine), "run:"))
	if cmd == "" {
		return "", fmt.Errorf("CI step %q's run line is empty after stripping the leading run: token", blanketGateStepName)
	}
	return cmd, nil
}

// writeGateProbeModule writes a throwaway, dependency-free, single-package
// Go module into dir: a go.mod declaring module aethergateprobe with the
// same `go` directive as the repo's own go.mod (read and parsed at test
// time, never hardcoded, so a future toolchain bump cannot silently break
// the probe), and gateprobe_test.go declaring package gateprobe with a
// single func TestGateProbe(t *testing.T). When failing is false, the test
// body is empty and the module's own test suite passes; when true, the body
// calls t.Fatal so `go test` against this module exits non-zero.
func writeGateProbeModule(t *testing.T, dir string, failing bool) {
	t.Helper()

	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	goModData, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		t.Fatalf("read repo go.mod: %v", err)
	}

	var goDirective string
	for _, line := range strings.Split(string(goModData), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "go ") {
			goDirective = strings.TrimSpace(strings.TrimPrefix(trimmed, "go"))
			break
		}
	}
	if goDirective == "" {
		t.Fatalf("repo go.mod has no `go <version>` directive — cannot declare a matching toolchain requirement for the probe module")
	}

	goModContent := fmt.Sprintf("module aethergateprobe\n\ngo %s\n", goDirective)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goModContent), 0o644); err != nil {
		t.Fatalf("write probe go.mod in %s: %v", dir, err)
	}

	var body string
	if failing {
		body = "\tt.Fatal(\"deliberate failure: the release gate command must report this\")\n"
	}
	testContent := fmt.Sprintf("package gateprobe\n\nimport \"testing\"\n\nfunc TestGateProbe(t *testing.T) {\n%s}\n", body)
	if err := os.WriteFile(filepath.Join(dir, "gateprobe_test.go"), []byte(testContent), 0o644); err != nil {
		t.Fatalf("write probe test file in %s: %v", dir, err)
	}
}

// gateCommandTimeout bounds every probe-module execution below. A gate
// command that hangs past this deadline is treated as a defect, never a
// silent pass — see runGateCommand.
const gateCommandTimeout = 120 * time.Second

// runGateCommand executes command via `sh -c` with cmd.Dir set to dir,
// bounded by the gateCommandTimeout context deadline, and returns its exit
// code. cmd.Env is left nil so the subprocess inherits the parent
// environment implicitly — this file may not name os.Environ
// (TestWiringGuardsHaveNoRuntimeEscapeHatch rejects it). A command that
// fails to start, errors for a reason other than a non-zero exit, or hits
// the deadline is a defect in the harness or the gate command itself, not a
// pass, so those cases t.Fatalf rather than returning a code.
func runGateCommand(t *testing.T, command, dir string) int {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), gateCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	if err == nil {
		t.Logf("command %q in %s exited 0\noutput:\n%s", command, dir, output)
		return 0
	}

	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("command %q in %s did not complete within 120s — a gate command that hangs or cannot start is a defect, not a pass\noutput so far:\n%s", command, dir, output)
	}

	var ee *exec.ExitError
	if errors.As(err, &ee) {
		t.Logf("command %q in %s exited %d\noutput:\n%s", command, dir, ee.ExitCode(), output)
		return ee.ExitCode()
	}

	t.Fatalf("command %q in %s failed to run (not an ordinary non-zero exit): %v\noutput so far:\n%s", command, dir, err, output)
	return -1
}

// gateCommandDiscriminates returns nil ONLY when command exits 0 in passDir
// (the clean tree) AND non-zero in failDir (the tree carrying a
// deliberately failing test). Either half breaking alone means the gate
// cannot be relied on: exiting non-zero on a clean tree is a false alarm
// that would redden every good build, and exiting zero on a tree with a
// failing test is exactly the silent-narrowing failure mode this plan
// exists to catch — the release gate reporting success while a test failed.
func gateCommandDiscriminates(t *testing.T, command, passDir, failDir string) error {
	t.Helper()

	passExit := runGateCommand(t, command, passDir)
	failExit := runGateCommand(t, command, failDir)

	if passExit != 0 {
		return fmt.Errorf("command %q reported failure (exit %d) on a tree containing no failing test — a false alarm that would make every clean build report red", command, passExit)
	}
	if failExit == 0 {
		return fmt.Errorf("command %q reported success (exit %d) on a tree containing a deliberately failing test — it cannot fail the build when a test fails", command, failExit)
	}
	return nil
}

// TestReleaseGateCommandFailsATreeWithAFailingTest is the behavioural half
// of ROADMAP Phase 172 success criterion 4: it takes the release gate's own
// command, read from ci.yml at test time via releaseGateCommandFromWorkflow,
// and proves BY EXECUTION that it fails a tree with a failing test and
// passes a tree without one — the discrimination the criterion names,
// established by running the command and watching it go red, not by
// reading the workflow file.
func TestReleaseGateCommandFailsATreeWithAFailingTest(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}

	command, err := releaseGateCommandFromWorkflow(string(data))
	if err != nil {
		t.Fatalf("%v", err)
	}

	if _, lookErr := exec.LookPath("go"); lookErr != nil {
		t.Fatalf("go toolchain not found on PATH — cannot execute the release gate command: %v", lookErr)
	}
	if _, lookErr := exec.LookPath("sh"); lookErr != nil {
		t.Fatalf("sh not found on PATH — cannot execute the release gate command: %v", lookErr)
	}

	// Sanity floor only — NOT the proof, and must never be allowed to become
	// the proof. A command that contains "go test" could still be neutered
	// by any of the mutations this plan exists to catch (or by one nobody
	// has thought of yet); the actual proof is the discrimination assertion
	// below, which executes the command against both trees.
	if command == "" || !strings.Contains(command, "go test") {
		t.Fatalf("extracted release gate command %q does not look like a go test invocation — refusing to proceed", command)
	}
	t.Logf("release gate command extracted from ci.yml: %s", command)

	root := t.TempDir()
	passDir := filepath.Join(root, "pass")
	failDir := filepath.Join(root, "fail")
	if mkErr := os.Mkdir(passDir, 0o755); mkErr != nil {
		t.Fatalf("mkdir %s: %v", passDir, mkErr)
	}
	if mkErr := os.Mkdir(failDir, 0o755); mkErr != nil {
		t.Fatalf("mkdir %s: %v", failDir, mkErr)
	}
	writeGateProbeModule(t, passDir, false)
	writeGateProbeModule(t, failDir, true)

	if discErr := gateCommandDiscriminates(t, command, passDir, failDir); discErr != nil {
		t.Fatalf("%v", discErr)
	}
}

// TestGateProbeCatchesEveryKnownGateNeutering is EVIDENCE that the execution
// mechanism above (gateCommandDiscriminates) works — it is NOT the
// mechanism itself, and must never be read as one. Do not extend this table
// to "fix" a future bypass; a bypass this table does not yet name is still
// caught, because every row below is caught by RUNNING the mutated command
// and observing it fail to discriminate, not by matching its text against a
// list of known-bad spellings. The control row (the unmutated command)
// exists so a future reader can confirm the harness is not simply rejecting
// everything it is given.
//
// Each row's mutated command is built from the command extracted live from
// ci.yml, never a literal copy, so the table cannot drift from what CI
// actually runs.
func TestGateProbeCatchesEveryKnownGateNeutering(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}

	command, err := releaseGateCommandFromWorkflow(string(data))
	if err != nil {
		t.Fatalf("%v", err)
	}

	// One passing and one failing probe module directory, built once and
	// reused across every row below, so the table stays bounded (Go's build
	// cache is warm after the first row) rather than rebuilding a module
	// seven times.
	root := t.TempDir()
	passDir := filepath.Join(root, "pass")
	failDir := filepath.Join(root, "fail")
	if mkErr := os.Mkdir(passDir, 0o755); mkErr != nil {
		t.Fatalf("mkdir %s: %v", passDir, mkErr)
	}
	if mkErr := os.Mkdir(failDir, 0o755); mkErr != nil {
		t.Fatalf("mkdir %s: %v", failDir, mkErr)
	}
	writeGateProbeModule(t, passDir, false)
	writeGateProbeModule(t, failDir, true)

	rows := []struct {
		name    string
		mutate  func(string) string
		wantErr bool
	}{
		{
			name:    "control: the unmutated command still discriminates",
			mutate:  func(c string) string { return c },
			wantErr: false,
		},
		{
			name:    "appended semicolon-true swallows the exit status",
			mutate:  func(c string) string { return c + "; true" },
			wantErr: true,
		},
		{
			name:    "appended or-exit-zero swallows the exit status",
			mutate:  func(c string) string { return c + " || exit 0" },
			wantErr: true,
		},
		{
			name:    "an appended -run filter matches nothing, so zero tests execute",
			mutate:  func(c string) string { return c + " -run TestNothingAtAll" },
			wantErr: true,
		},
		{
			name:    "piping through cat discards the real exit status",
			mutate:  func(c string) string { return c + " | cat" },
			wantErr: true,
		},
		{
			name:    "the whole command commented out never runs",
			mutate:  func(c string) string { return "# " + c },
			wantErr: true,
		},
		{
			name:    "the required text is present but sits inside a comment",
			mutate:  func(c string) string { return "echo skip # " + c },
			wantErr: true,
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			mutated := row.mutate(command)
			discErr := gateCommandDiscriminates(t, mutated, passDir, failDir)
			if row.wantErr {
				if discErr == nil {
					t.Fatalf("mutated command %q was expected to fail discrimination but gateCommandDiscriminates returned nil — "+
						"this mutation slipped through, which is exactly the defect this phase has failed on twice before", mutated)
				}
				t.Logf("mutated command %q correctly caught: %v", mutated, discErr)
			} else if discErr != nil {
				t.Fatalf("control row (unmutated command %q) failed to discriminate: %v — the harness itself is broken, not just failing to catch a mutation", mutated, discErr)
			} else {
				t.Logf("control row (unmutated command %q) correctly discriminates", mutated)
			}
		})
	}
}

// Plan 172-11 task 2 (T-172-54).
//
// Every guard above — blanketReleaseGateProblem, stepCanFailTheBuild,
// TestReleaseGateCommandFailsATreeWithAFailingTest — inspects a STEP. None
// of them can see the one route that disables all of them at once: the
// workflow simply never being invoked (its `on:` triggers narrowed to
// something that never fires on a pull request or push), or the job that
// contains every step being switched off with a job-level `if:`. A step
// that is perfectly structured to fail the build is not a gate if GitHub
// Actions never runs it.

// TestReleaseGateWorkflowActuallyRuns fails by name when the workflow's
// on: block no longer names both pull_request and push, when the go: job
// carries a job-level if: condition, or when the go: job is missing its
// steps: key or a non-empty runs-on: value — each of which would leave
// every step-scoped guard in this file green while nothing in the workflow
// ever executes.
func TestReleaseGateWorkflowActuallyRuns(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}
	lines := strings.Split(string(data), "\n")

	// Locate the top-level `on:` and `jobs:` keys (column 0, no indent) so a
	// nested key that happens to be spelled "on:" elsewhere is never
	// mistaken for the workflow-level trigger block.
	onLineIdx, jobsLineIdx := -1, -1
	for i, line := range lines {
		switch strings.TrimRight(line, " ") {
		case "on:":
			if onLineIdx == -1 {
				onLineIdx = i
			}
		case "jobs:":
			jobsLineIdx = i
		}
		if onLineIdx != -1 && jobsLineIdx != -1 {
			break
		}
	}
	if onLineIdx == -1 || jobsLineIdx == -1 || jobsLineIdx <= onLineIdx {
		t.Fatalf(".github/workflows/ci.yml does not have the expected top-level on:/jobs: shape")
	}
	onBlock := strings.Join(lines[onLineIdx:jobsLineIdx], "\n")

	if !strings.Contains(onBlock, "pull_request:") {
		t.Fatalf("the check that is supposed to run on every change would no longer run: .github/workflows/ci.yml's on: block no longer names the pull_request trigger")
	}
	if !strings.Contains(onBlock, "push:") {
		t.Fatalf("the check that is supposed to run on every change would no longer run: .github/workflows/ci.yml's on: block no longer names the push trigger")
	}

	// Locate the `go:` job key at 2-space indent under `jobs:`.
	goJobLineIdx := -1
	for i := jobsLineIdx; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " ") == "  go:" {
			goJobLineIdx = i
			break
		}
	}
	if goJobLineIdx == -1 {
		t.Fatalf(".github/workflows/ci.yml has no top-level %q job", "go")
	}

	// The job header runs from the line after `go:` up to (but excluding)
	// whichever comes first: a line back at 2-space indent or shallower (a
	// sibling job, or the end of the jobs: block), or the job's own
	// `steps:` key. Scanning only this header — never the steps below it —
	// is what stops a step-level `if:` (already handled by
	// stepCanFailTheBuild) from being double-reported here as a job-level
	// condition.
	stepsLineIdx := -1
	headerEndIdx := len(lines)
	for i := goJobLineIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		indent := len(lines[i]) - len(strings.TrimLeft(lines[i], " "))
		if indent <= 2 {
			headerEndIdx = i
			break
		}
		if trimmed == "steps:" {
			stepsLineIdx = i
			headerEndIdx = i
			break
		}
	}

	var jobHeaderLines []string
	if goJobLineIdx+1 < headerEndIdx {
		jobHeaderLines = lines[goJobLineIdx+1 : headerEndIdx]
	}

	for _, line := range jobHeaderLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if:") {
			t.Fatalf("the check that is supposed to run on every change would no longer run: the %q job carries a job-level condition (%s) that can switch it off entirely", "go", trimmed)
		}
	}

	if stepsLineIdx == -1 {
		t.Fatalf("the %q job has no steps: key — a hollowed-out job would otherwise satisfy every step-scoped guard while running nothing", "go")
	}

	runsOnValue := ""
	for _, line := range jobHeaderLines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "runs-on:") {
			runsOnValue = strings.TrimSpace(strings.TrimPrefix(trimmed, "runs-on:"))
		}
	}
	if runsOnValue == "" {
		t.Fatalf("the %q job has no non-empty runs-on: value — a hollowed-out job would otherwise satisfy every step-scoped guard while running nothing", "go")
	}
}

// Plan 172-12 (round 4, CR-01/CR-02/CR-03).
//
// Every guard above inspects the workflow by searching for TEXT it already
// knows to look for — a step's name, a run line's exact command, a
// trigger's substring presence. Each of those is a blocklist: it catches
// the mutations someone thought to name. CR-01 (`jobs.go.continue-on-error:
// true`), CR-02 (`jobs.go.env.GOFLAGS`, and its step-level spelling), and
// CR-03 (`paths-ignore: ['**']` under pull_request) each landed a key that
// none of the checks above were looking for, at a scope none of them
// inspect (job-level and trigger-level, not step-level).
//
// auditWorkflowShape is the generalising move: instead of searching for
// known-bad keys, it asserts the workflow's root, its triggers, the go
// job, and the two gate steps carry EXACTLY a reviewed set of keys —
// declared in the five whitelists below — so any key absent from that set
// fails by default, including one nobody has enumerated yet. Adding a
// legitimate key later is then a one-line, reviewed, visible diff to a
// named slice in this file, not a silent behavioural change.

// workflowRootAllowedKeys is a WHITELIST of every key permitted at the
// root of .github/workflows/ci.yml. Anything absent from this list fails
// by default (auditWorkflowShape rule b) — the check that is supposed to
// run on every change could otherwise be switched off or redirected by an
// unreviewed root key. Adding a member here is a deliberate, reviewed code
// change whose diff is visible in this file.
var workflowRootAllowedKeys = []string{"jobs", "name", "on"}

// gateJobAllowedKeys is a WHITELIST of every key permitted on the `go` job
// (gateJobName) under `jobs:`. `continue-on-error` (CR-01) and `env`
// (CR-02) are absent and therefore rejected by rule g, along with `if`,
// `strategy`, `timeout-minutes`, `container`, `services`, `defaults`,
// `outputs`, `permissions`, `concurrency` and `needs` — none of them named
// anywhere in this file. Adding a member here is a deliberate, reviewed
// code change.
var gateJobAllowedKeys = []string{"runs-on", "steps"}

// releaseTriggerAllowedNames is a WHITELIST of the trigger names permitted
// under the workflow's `on:` block — exactly these, both required, none
// extra (rule d). Adding `workflow_dispatch` later is a one-line reviewed
// change to this list.
var releaseTriggerAllowedNames = []string{"pull_request", "push"}

// releaseTriggerAllowedKeys is a WHITELIST of every key permitted on each
// trigger's own value. `paths-ignore`, `paths`, `types` and
// `branches-ignore` are absent and therefore rejected by rule e — CR-03 is
// closed because the key is missing from this list, not because
// `paths-ignore` is named anywhere. Adding a member here is a deliberate,
// reviewed code change.
var releaseTriggerAllowedKeys = []string{"branches"}

// gateStepAllowedKeys is a WHITELIST of every key permitted on the two
// audited gate steps (blanketGateStepName and wiringGateStepName). `env`
// (CR-02's step-level spelling) is absent and therefore rejected by rule
// h, along with step-level `if`, `continue-on-error`, `working-directory`,
// `shell` and `timeout-minutes`. Only these two steps are audited — every
// other step (`uses:`, `with:`, `id:`, the goreleaser step's own `env:`)
// is deliberately untouched; a whitelist scoped wider than the gate steps
// would break ordinary CI work that has nothing to do with the release
// gate.
var gateStepAllowedKeys = []string{"name", "run"}

// gateJobName is the job key under `jobs:` this audit inspects.
const gateJobName = "go"

// workflowShapeAudit records what auditWorkflowShape actually inspected —
// not just the errors it found — so a caller can prove the audit is not
// vacuous. A parser that silently walked nothing would otherwise report
// "no problems" forever, which is the same failure shape as the substring
// checks this plan replaces.
type workflowShapeAudit struct {
	RootKeys      []string
	TriggerNames  []string
	TriggerKeys   map[string][]string
	GateJobKeys   []string
	GateStepNames []string
	StepCount     int
}

// yamlMappingPair is one key/value pair from a yaml.Node mapping, kept as
// nodes (rather than resolved strings) so callers can report `.Line` in
// failure messages.
type yamlMappingPair struct {
	key   *yaml.Node
	value *yaml.Node
}

// yamlMappingPairs returns the key/value pairs of mapping node m. m must
// already be known to have Kind == yaml.MappingNode; callers check that
// before calling this.
func yamlMappingPairs(m *yaml.Node) []yamlMappingPair {
	var out []yamlMappingPair
	for i := 0; i+1 < len(m.Content); i += 2 {
		out = append(out, yamlMappingPair{key: m.Content[i], value: m.Content[i+1]})
	}
	return out
}

// describeYAMLSequence renders a yaml.Node for a failure message: its
// scalar elements if it is a sequence, or a description naming its actual
// kind and value otherwise. Used only for error text.
func describeYAMLSequence(n *yaml.Node) string {
	if n == nil {
		return "<missing>"
	}
	if n.Kind != yaml.SequenceNode {
		return fmt.Sprintf("a non-sequence value %q", n.Value)
	}
	vals := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		vals = append(vals, c.Value)
	}
	return fmt.Sprintf("%v", vals)
}

// auditWorkflowShape parses workflow — the raw text of a GitHub Actions
// workflow file — into a yaml.Node tree (gopkg.in/yaml.v3 decodes the `on:`
// key as the plain string "on", not the YAML-1.1 boolean true; verified
// against the live file during planning) and checks it against the five
// reviewed whitelists declared above. It returns every violation found,
// never stopping at the first, plus a workflowShapeAudit recording what it
// actually inspected. Failure prose is written for a non-technical
// operator: each message names the offending key, the scope it was found
// at, the source line, and states that the check meant to run on every
// change could be switched off or redirected by that key.
func auditWorkflowShape(workflow string) (workflowShapeAudit, []error) {
	var audit workflowShapeAudit
	audit.TriggerKeys = map[string][]string{}
	var errs []error

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(workflow), &doc); err != nil {
		return audit, []error{fmt.Errorf("workflow shape audit: could not parse YAML: %w", err)}
	}
	if len(doc.Content) == 0 || doc.Content[0] == nil {
		return audit, []error{fmt.Errorf("workflow shape audit: document has no content — a parse that silently produces nothing must not be treated as a pass")}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return audit, []error{fmt.Errorf("workflow shape audit: root node (line %d) is not a mapping", root.Line)}
	}

	var onNode, jobsNode *yaml.Node
	foundOn, foundJobs := false, false
	for _, p := range yamlMappingPairs(root) {
		audit.RootKeys = append(audit.RootKeys, p.key.Value)
		if !stringSliceContains(workflowRootAllowedKeys, p.key.Value) {
			errs = append(errs, fmt.Errorf(
				"workflow root key %q (line %d, scope: workflow) is not on the reviewed whitelist workflowRootAllowedKeys %v — the check that is supposed to run on every change could be switched off or redirected by this key; add it to workflowRootAllowedKeys as a deliberate, reviewed change if it belongs",
				p.key.Value, p.key.Line, workflowRootAllowedKeys))
		}
		if p.key.Value == "on" {
			foundOn = true
			onNode = p.value
		}
		if p.key.Value == "jobs" {
			foundJobs = true
			jobsNode = p.value
		}
	}

	// Rule c: the root mapping must contain a key node whose .Value is
	// exactly "on" — a hazard warning distinct from rule b's whitelist
	// check, guarding against a future library or schema change silently
	// re-resolving a bare `on` key to the YAML-1.1 boolean true, which
	// would make every trigger check below silently inspect nothing.
	if !foundOn {
		errs = append(errs, fmt.Errorf(
			"workflow root (line %d, scope: workflow) has no key node whose value is exactly \"on\" — some YAML readers resolve a bare `on` key to the boolean true, which would make every trigger check below silently inspect nothing",
			root.Line))
	}

	if onNode != nil {
		if onNode.Kind != yaml.MappingNode {
			errs = append(errs, fmt.Errorf("workflow trigger block \"on\" (line %d, scope: workflow) is not a mapping", onNode.Line))
		} else {
			triggerPairs := yamlMappingPairs(onNode)
			for _, p := range triggerPairs {
				audit.TriggerNames = append(audit.TriggerNames, p.key.Value)
			}

			// Rule d: EXACTLY releaseTriggerAllowedNames — both required,
			// none extra. An extra trigger is rejected the same way an
			// extra key is; adding workflow_dispatch later is a one-line
			// reviewed change to the list.
			for _, p := range triggerPairs {
				if !stringSliceContains(releaseTriggerAllowedNames, p.key.Value) {
					errs = append(errs, fmt.Errorf(
						"workflow trigger %q (line %d, scope: workflow) is not on the reviewed whitelist releaseTriggerAllowedNames %v",
						p.key.Value, p.key.Line, releaseTriggerAllowedNames))
				}
			}
			for _, want := range releaseTriggerAllowedNames {
				if !stringSliceContains(audit.TriggerNames, want) {
					errs = append(errs, fmt.Errorf(
						"workflow trigger block (line %d, scope: workflow) is missing the required trigger %q named in releaseTriggerAllowedNames %v",
						onNode.Line, want, releaseTriggerAllowedNames))
				}
			}

			// Rule e: each trigger's own keys, and the branches: value.
			for _, p := range triggerPairs {
				triggerName, triggerVal := p.key.Value, p.value
				scope := fmt.Sprintf("trigger %q", triggerName)
				if triggerVal.Kind != yaml.MappingNode {
					errs = append(errs, fmt.Errorf("%s value (line %d, scope: %s) is not a mapping", scope, triggerVal.Line, scope))
					continue
				}
				var branchesNode *yaml.Node
				for _, tp := range yamlMappingPairs(triggerVal) {
					audit.TriggerKeys[triggerName] = append(audit.TriggerKeys[triggerName], tp.key.Value)
					if !stringSliceContains(releaseTriggerAllowedKeys, tp.key.Value) {
						errs = append(errs, fmt.Errorf(
							"%s key %q (line %d, scope: %s) is not on the reviewed whitelist releaseTriggerAllowedKeys %v — paths-ignore, paths, types, branches-ignore and every other unreviewed trigger key are each rejected by absence from this list, not by name",
							scope, tp.key.Value, tp.key.Line, scope, releaseTriggerAllowedKeys))
					}
					if tp.key.Value == "branches" {
						branchesNode = tp.value
					}
				}
				if branchesNode == nil {
					errs = append(errs, fmt.Errorf("%s (line %d, scope: %s) has no branches: key", scope, triggerVal.Line, scope))
				} else if branchesNode.Kind != yaml.SequenceNode || len(branchesNode.Content) != 1 || branchesNode.Content[0].Value != "main" {
					errs = append(errs, fmt.Errorf(
						"%s's branches: value (line %d, scope: %s) must be a sequence of exactly one scalar \"main\" — found %s",
						scope, branchesNode.Line, scope, describeYAMLSequence(branchesNode)))
				}
			}
		}
	}

	if !foundJobs {
		errs = append(errs, fmt.Errorf("workflow root (line %d, scope: workflow) has no %q key", root.Line, "jobs"))
	} else if jobsNode.Kind != yaml.MappingNode {
		errs = append(errs, fmt.Errorf("workflow %q key (line %d, scope: workflow) is not a mapping", "jobs", jobsNode.Line))
	} else {
		// Rule f: the jobs value must contain gateJobName. Sibling jobs are
		// not whitelisted — a sibling job cannot disable the go job, only
		// the go job's own keys can, and a sibling's own `needs:` on the go
		// job is itself caught by rule g below.
		var goJobNode *yaml.Node
		for _, p := range yamlMappingPairs(jobsNode) {
			if p.key.Value == gateJobName {
				goJobNode = p.value
			}
		}
		if goJobNode == nil {
			errs = append(errs, fmt.Errorf("workflow %q block (line %d, scope: workflow) has no %q job", "jobs", jobsNode.Line, gateJobName))
		} else if goJobNode.Kind != yaml.MappingNode {
			errs = append(errs, fmt.Errorf("job %q (line %d, scope: job %q) is not a mapping", gateJobName, goJobNode.Line, gateJobName))
		} else {
			scope := fmt.Sprintf("job %q", gateJobName)
			var stepsNode, runsOnNode *yaml.Node
			for _, p := range yamlMappingPairs(goJobNode) {
				audit.GateJobKeys = append(audit.GateJobKeys, p.key.Value)
				if !stringSliceContains(gateJobAllowedKeys, p.key.Value) {
					errs = append(errs, fmt.Errorf(
						"%s key %q (line %d, scope: %s) is not on the reviewed whitelist gateJobAllowedKeys %v — the check that is supposed to run on every change could be switched off or redirected by this key",
						scope, p.key.Value, p.key.Line, scope, gateJobAllowedKeys))
				}
				if p.key.Value == "runs-on" {
					runsOnNode = p.value
				}
				if p.key.Value == "steps" {
					stepsNode = p.value
				}
			}

			if runsOnNode == nil || runsOnNode.Kind != yaml.ScalarNode || strings.TrimSpace(runsOnNode.Value) == "" {
				errs = append(errs, fmt.Errorf("%s (line %d, scope: %s) has no non-empty runs-on: scalar value", scope, goJobNode.Line, scope))
			}

			if stepsNode == nil || stepsNode.Kind != yaml.SequenceNode || len(stepsNode.Content) == 0 {
				errs = append(errs, fmt.Errorf("%s (line %d, scope: %s) has no non-empty steps: sequence", scope, goJobNode.Line, scope))
			} else {
				audit.StepCount = len(stepsNode.Content)

				// Rule h: each of the two gate steps must appear exactly
				// once, and its own key set must be a subset of
				// gateStepAllowedKeys.
				for _, wantStepName := range []string{blanketGateStepName, wiringGateStepName} {
					var matches []*yaml.Node
					for _, stepNode := range stepsNode.Content {
						if stepNode.Kind != yaml.MappingNode {
							continue
						}
						for _, sp := range yamlMappingPairs(stepNode) {
							if sp.key.Value == "name" && sp.value.Value == wantStepName {
								matches = append(matches, stepNode)
							}
						}
					}
					stepScope := fmt.Sprintf("step %q", wantStepName)
					if len(matches) != 1 {
						errs = append(errs, fmt.Errorf(
							"%s (line %d, scope: %s) must contain exactly one step named %q — found %d",
							scope, stepsNode.Line, scope, wantStepName, len(matches)))
						continue
					}
					audit.GateStepNames = append(audit.GateStepNames, wantStepName)
					for _, sp := range yamlMappingPairs(matches[0]) {
						if !stringSliceContains(gateStepAllowedKeys, sp.key.Value) {
							errs = append(errs, fmt.Errorf(
								"%s key %q (line %d, scope: %s) is not on the reviewed whitelist gateStepAllowedKeys %v — the check that is supposed to run on every change could be switched off or redirected by this key",
								stepScope, sp.key.Value, sp.key.Line, stepScope, gateStepAllowedKeys))
						}
					}
				}
			}
		}
	}

	return audit, errs
}

// TestReleaseGateWorkflowShapeIsWhitelisted is the structural complement
// to TestReleaseGateCommandFailsATreeWithAFailingTest (the behavioural
// proof) and TestReleaseGateWorkflowActuallyRuns (the trigger-presence
// proof): it asserts the live .github/workflows/ci.yml carries EXACTLY the
// reviewed key shape at workflow, job, trigger and gate-step scope, so
// CR-01 (`jobs.go.continue-on-error`), CR-02 (`jobs.go.env` and its
// step-level spelling) and CR-03 (`paths-ignore`) each fail because the
// key is absent from a reviewed whitelist, not because any of them is
// individually named in a check.
func TestReleaseGateWorkflowShapeIsWhitelisted(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read %s: %v", workflowPath, err)
	}

	audit, errs := auditWorkflowShape(string(data))
	if len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		t.Fatalf("auditWorkflowShape found %d problem(s) with the live workflow's shape:\n  %s", len(errs), strings.Join(msgs, "\n  "))
	}

	// Anti-vacuity pins (T-172-63): a parser that silently walked nothing
	// would report zero errors forever, which is the same failure shape as
	// the substring checks this plan replaces. Pinning the measured counts
	// against the live file means a shrunken audit — one that stopped
	// looking — is fatal here, not silently passing.
	t.Logf("audited root keys: %v", audit.RootKeys)
	t.Logf("audited trigger names: %v", audit.TriggerNames)
	t.Logf("audited trigger keys: %v", audit.TriggerKeys)
	t.Logf("audited go job keys: %v", audit.GateJobKeys)
	t.Logf("audited gate step names: %v", audit.GateStepNames)
	t.Logf("audited step count: %d", audit.StepCount)

	if len(audit.RootKeys) != 3 {
		t.Fatalf("audit inspected %d root key(s) (%v), expected exactly 3 — a shrunken audit means the whitelist stopped looking, which passes forever, and is therefore fatal", len(audit.RootKeys), audit.RootKeys)
	}
	if len(audit.TriggerNames) != 2 {
		t.Fatalf("audit inspected %d trigger name(s) (%v), expected exactly 2 — a shrunken audit means the whitelist stopped looking, which passes forever, and is therefore fatal", len(audit.TriggerNames), audit.TriggerNames)
	}
	if len(audit.GateJobKeys) != 2 {
		t.Fatalf("audit inspected %d go-job key(s) (%v), expected exactly 2 — a shrunken audit means the whitelist stopped looking, which passes forever, and is therefore fatal", len(audit.GateJobKeys), audit.GateJobKeys)
	}
	if len(audit.GateStepNames) != 2 {
		t.Fatalf("audit found %d gate step name(s) (%v), expected exactly 2 (%q and %q) — a shrunken audit means the whitelist stopped looking, which passes forever, and is therefore fatal", len(audit.GateStepNames), audit.GateStepNames, blanketGateStepName, wiringGateStepName)
	}
	if audit.StepCount < 10 {
		t.Fatalf("audit found only %d step(s) in the %q job (measured 20 today) — expected at least 10; a shrunken audit means the whitelist stopped looking, which passes forever, and is therefore fatal", audit.StepCount, gateJobName)
	}
}

// Plan 172-12 task 2 (T-172-62, T-172-63).
//
// TestReleaseGateWorkflowShapeIsWhitelisted above proves the audit accepts
// the live workflow and rejects the five known defects it was written
// against. That alone would not distinguish a genuine whitelist from a
// disguised blocklist that happens to name exactly those five keys. The
// test below is the generality proof: it is hermetic (synthetic workflow
// strings built in the test body, never the live file, so it cannot be
// quietly satisfied by a change to ci.yml) and it rejects mutations this
// plan's own source never names — including a loop over invented key
// names generated at runtime — which is the property that decides whether
// this phase terminates per 172-STOP-RULE.md.

// testShapeBaseWorkflow is the one minimal, valid synthetic workflow every
// row below mutates. It is built from the real blanketGateStepName,
// blanketGateRunCommand and wiringGateStepName constants (never a literal
// duplicate of them) so it cannot silently drift from what
// auditWorkflowShape actually looks for.
var testShapeBaseWorkflow = fmt.Sprintf(`name: Test Workflow

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - name: %s
        run: %s
      - name: %s
        run: go test ./cmd -run 'TestSomething' -count=1 -v
`, blanketGateStepName, blanketGateRunCommand, wiringGateStepName)

// testShapeBlanketStepNameLine and testShapeWiringStepNameLine are the
// exact `- name: ...` lines the base workflow carries for the two gate
// steps, used as insertion markers by the step-scope mutations below.
var (
	testShapeBlanketStepNameLine = fmt.Sprintf("      - name: %s\n", blanketGateStepName)
	testShapeWiringStepNameLine  = fmt.Sprintf("      - name: %s\n", wiringGateStepName)
)

// testShapeMustInsertAfter inserts insertion immediately after the single
// occurrence of marker in workflow. It fails the (sub)test immediately if
// marker is missing or not unique, rather than silently mutating the wrong
// spot — the hermetic table must stay in sync with its own base template.
func testShapeMustInsertAfter(t *testing.T, workflow, marker, insertion string) string {
	t.Helper()
	if strings.Count(workflow, marker) != 1 {
		t.Fatalf("test workflow template marker %q does not appear exactly once — the hermetic table is out of sync with its own base template", marker)
	}
	idx := strings.Index(workflow, marker)
	pos := idx + len(marker)
	return workflow[:pos] + insertion + workflow[pos:]
}

// testShapeMustReplaceOnce replaces the single occurrence of old in
// workflow with replacement, failing immediately if old is missing or not
// unique.
func testShapeMustReplaceOnce(t *testing.T, workflow, old, replacement string) string {
	t.Helper()
	if strings.Count(workflow, old) != 1 {
		t.Fatalf("test workflow template substring %q does not appear exactly once — the hermetic table is out of sync with its own base template", old)
	}
	return strings.Replace(workflow, old, replacement, 1)
}

// joinAuditErrs renders a slice of auditWorkflowShape errors as one string
// for substring assertions in the table below.
func joinAuditErrs(errs []error) string {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "\n")
}

// TestWorkflowShapeWhitelistRejectsUnenumeratedKeys is hermetic and
// table-driven over synthetic workflow strings built in the test body
// (never the live file), so the guarantee that an unenumerated key is
// rejected keeps holding on every CI run rather than resting on a
// one-time transcript against ci.yml.
func TestWorkflowShapeWhitelistRejectsUnenumeratedKeys(t *testing.T) {
	type shapeRow struct {
		name    string
		mutate  func(t *testing.T, workflow string) string
		wantErr bool
		wantSub []string
		zeroMsg string
	}

	rows := []shapeRow{
		{
			// Row 1: the control. Without this row, a whitelist that
			// rejects everything would look like a working whitelist —
			// this proves the base template itself is accepted.
			name:    "control: the unmutated base workflow returns zero errors",
			mutate:  func(t *testing.T, wf string) string { return wf },
			wantErr: false,
			zeroMsg: "the harness itself is broken (the base template does not even satisfy its own audit), not that a mutation slipped through",
		},

		// The five known defects (CR-01, CR-02 at two scopes, CR-03), each
		// mutating the base string.
		{
			name: "known defect: job-level continue-on-error",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    continue-on-error: true\n")
			},
			wantErr: true,
			wantSub: []string{"continue-on-error", `job "go"`},
		},
		{
			name: "known defect: job-level env with GOFLAGS",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    env:\n      GOFLAGS: -run=TestNothingZZZ\n")
			},
			wantErr: true,
			wantSub: []string{"env", `job "go"`},
		},
		{
			name: "known defect: workflow-root env",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "name: Test Workflow\n", "env:\n  GOFLAGS: -run=TestNothingZZZ\n")
			},
			wantErr: true,
			wantSub: []string{"env", "scope: workflow"},
		},
		{
			name: "known defect: gate-step env",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, testShapeBlanketStepNameLine, "        env:\n          GOFLAGS: -run=TestNothingZZZ\n")
			},
			wantErr: true,
			wantSub: []string{"env", fmt.Sprintf("step %q", blanketGateStepName)},
		},
		{
			name: "known defect: paths-ignore under pull_request",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "  pull_request:\n    branches: [main]\n", "    paths-ignore: ['**']\n")
			},
			wantErr: true,
			wantSub: []string{"paths-ignore", `trigger "pull_request"`},
		},

		// Generality rows: mutations that are not one of the five known
		// defects, are not named anywhere in auditWorkflowShape, and would
		// each be missed by a blocklist.
		{
			name: "generality: root concurrency",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "name: Test Workflow\n", "concurrency: ci-${{ github.ref }}\n")
			},
			wantErr: true,
			wantSub: []string{"concurrency", "scope: workflow"},
		},
		{
			name: "generality: root defaults",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "name: Test Workflow\n", "defaults:\n  run:\n    shell: bash\n")
			},
			wantErr: true,
			wantSub: []string{"defaults", "scope: workflow"},
		},
		{
			name: "generality: root permissions",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "name: Test Workflow\n", "permissions: read-all\n")
			},
			wantErr: true,
			wantSub: []string{"permissions", "scope: workflow"},
		},
		{
			name: "generality: trigger types",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "  pull_request:\n    branches: [main]\n", "    types: [opened]\n")
			},
			wantErr: true,
			wantSub: []string{"types", `trigger "pull_request"`},
		},
		{
			name: "generality: trigger branches-ignore",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "  pull_request:\n    branches: [main]\n", "    branches-ignore: [dev]\n")
			},
			wantErr: true,
			wantSub: []string{"branches-ignore", `trigger "pull_request"`},
		},
		{
			name: "generality: trigger paths",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "  pull_request:\n    branches: [main]\n", "    paths: ['**.go']\n")
			},
			wantErr: true,
			wantSub: []string{"paths", `trigger "pull_request"`},
		},
		{
			name: "generality: branches value is not exactly [main]",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustReplaceOnce(t, wf, "  pull_request:\n    branches: [main]\n", "  pull_request:\n    branches: [main-disabled]\n")
			},
			wantErr: true,
			wantSub: []string{"branches:", `trigger "pull_request"`},
		},
		{
			name: "generality: job timeout-minutes",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    timeout-minutes: 1\n")
			},
			wantErr: true,
			wantSub: []string{"timeout-minutes", `job "go"`},
		},
		{
			name: "generality: job strategy",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    strategy:\n      matrix:\n        os: [ubuntu-latest]\n")
			},
			wantErr: true,
			wantSub: []string{"strategy", `job "go"`},
		},
		{
			name: "generality: job needs",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    needs: []\n")
			},
			wantErr: true,
			wantSub: []string{"needs", `job "go"`},
		},
		{
			name: "generality: job if",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", "    if: success()\n")
			},
			wantErr: true,
			wantSub: []string{`"if"`, `job "go"`},
		},
		{
			name: "generality: gate-step working-directory",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, testShapeBlanketStepNameLine, "        working-directory: .\n")
			},
			wantErr: true,
			wantSub: []string{"working-directory", fmt.Sprintf("step %q", blanketGateStepName)},
		},
		{
			name: "generality: gate-step shell",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, testShapeWiringStepNameLine, "        shell: bash\n")
			},
			wantErr: true,
			wantSub: []string{"shell", fmt.Sprintf("step %q", wiringGateStepName)},
		},
		{
			name: "generality: gate-step continue-on-error (step-level, not job-level)",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, testShapeBlanketStepNameLine, "        continue-on-error: true\n")
			},
			wantErr: true,
			wantSub: []string{"continue-on-error", fmt.Sprintf("step %q", blanketGateStepName)},
		},
		{
			name: "generality: on key missing entirely",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustReplaceOnce(t, wf, "\non:\n  pull_request:\n    branches: [main]\n  push:\n    branches: [main]\n", "\n")
			},
			wantErr: true,
			wantSub: []string{`"on"`},
		},
		{
			name: "generality: extra workflow_dispatch trigger",
			mutate: func(t *testing.T, wf string) string {
				return testShapeMustInsertAfter(t, wf, "on:\n", "  workflow_dispatch: {}\n")
			},
			wantErr: true,
			wantSub: []string{"workflow_dispatch"},
		},
		{
			name: "generality: a second step also named the blanket gate step's name",
			mutate: func(t *testing.T, wf string) string {
				return wf + fmt.Sprintf("      - name: %s\n        run: echo decoy\n", blanketGateStepName)
			},
			wantErr: true,
			wantSub: []string{fmt.Sprintf("named %q", blanketGateStepName), "found 2"},
		},

		{
			// Negative control: only the two gate steps are audited. A
			// non-gate step carrying uses/with/env must not trip the
			// audit, or a whitelist scoped wider than the gate steps
			// would break ordinary CI work that has nothing to do with
			// the release gate.
			name: "negative control: a non-gate step with uses/with/env returns zero errors",
			mutate: func(t *testing.T, wf string) string {
				return wf + "      - name: Some Other Step\n        uses: actions/checkout@v4\n        with:\n          foo: bar\n        env:\n          BAZ: qux\n"
			},
			wantErr: false,
			zeroMsg: "a whitelist scoped wider than the gate steps would break ordinary CI work that has nothing to do with the release gate",
		},
	}

	if len(rows) < 24 {
		t.Fatalf("shapeRow table has only %d row(s) — expected at least 24 (control + 5 known defects + 17 generality rows + negative control)", len(rows))
	}

	notKnownDefectCount := 0
	for _, row := range rows {
		if strings.HasPrefix(row.name, "generality:") || strings.HasPrefix(row.name, "negative control:") {
			notKnownDefectCount++
		}
	}
	if notKnownDefectCount < 17 {
		t.Fatalf("only %d row(s) are NOT among the five known defects — expected at least 17", notKnownDefectCount)
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			workflow := row.mutate(t, testShapeBaseWorkflow)
			_, errs := auditWorkflowShape(workflow)
			if row.wantErr {
				if len(errs) == 0 {
					t.Fatalf("mutation %q was expected to be rejected but auditWorkflowShape returned zero errors", row.name)
				}
				joined := joinAuditErrs(errs)
				for _, sub := range row.wantSub {
					if !strings.Contains(joined, sub) {
						t.Fatalf("mutation %q errors do not contain %q:\n%s", row.name, sub, joined)
					}
				}
			} else if len(errs) != 0 {
				msg := row.zeroMsg
				if msg == "" {
					msg = "expected zero errors"
				}
				t.Fatalf("mutation %q was expected to return zero errors but got %d — %s:\n%s", row.name, len(errs), msg, joinAuditErrs(errs))
			}
		})
	}

	// Row 25: the strongest generality proof, as a loop rather than a
	// table entry. Twelve invented key names, fixed and written directly
	// in this file, plus one built by string concatenation at runtime so
	// its exact value never appears as a literal anywhere in source, are
	// each injected at all four scopes this audit inspects (job, workflow
	// root, trigger, gate step) and every single injection must be
	// rejected, with the key named in the error. This establishes that
	// the whitelist rejects a key BECAUSE it is absent from a reviewed
	// list, not because anyone predicted it — the property that decides
	// whether this phase terminates (172-STOP-RULE.md).
	t.Run("generality loop: twelve-plus invented key names rejected at all four scopes", func(t *testing.T) {
		fixedNames := []string{
			"zzz-unreviewed-key-1", "zzz-unreviewed-key-2", "zzz-unreviewed-key-3",
			"zzz-unreviewed-key-4", "zzz-unreviewed-key-5", "zzz-unreviewed-key-6",
			"zzz-unreviewed-key-7", "zzz-unreviewed-key-8", "zzz-unreviewed-key-9",
			"zzz-unreviewed-key-10", "zzz-unreviewed-key-11", "zzz-unreviewed-key-12",
		}
		// extraName is built by concatenating a prefix and an index at
		// runtime, rather than written as a single literal string
		// anywhere in this file — so this specific injected value could
		// not have been anticipated by reading the file's literals.
		extraPrefix := "zzz-unreviewed-key-extra-"
		extraName := extraPrefix + fmt.Sprintf("%d", 13)
		names := append(append([]string{}, fixedNames...), extraName)

		if len(names) < 12 {
			t.Fatalf("only %d invented key name(s) — expected at least 12", len(names))
		}

		type scopeInjector struct {
			scope  string
			inject func(t *testing.T, wf, key string) string
		}
		scopes := []scopeInjector{
			{
				scope: "job",
				inject: func(t *testing.T, wf, key string) string {
					return testShapeMustInsertAfter(t, wf, "    runs-on: ubuntu-latest\n", fmt.Sprintf("    %s: true\n", key))
				},
			},
			{
				scope: "workflow-root",
				inject: func(t *testing.T, wf, key string) string {
					return testShapeMustInsertAfter(t, wf, "name: Test Workflow\n", fmt.Sprintf("%s: true\n", key))
				},
			},
			{
				scope: "trigger",
				inject: func(t *testing.T, wf, key string) string {
					return testShapeMustInsertAfter(t, wf, "  pull_request:\n    branches: [main]\n", fmt.Sprintf("    %s: true\n", key))
				},
			},
			{
				scope: "gate-step",
				inject: func(t *testing.T, wf, key string) string {
					return testShapeMustInsertAfter(t, wf, testShapeBlanketStepNameLine, fmt.Sprintf("        %s: true\n", key))
				},
			},
		}

		attempted := 0
		rejected := 0
		for _, key := range names {
			for _, sc := range scopes {
				attempted++
				mutated := sc.inject(t, testShapeBaseWorkflow, key)
				_, errs := auditWorkflowShape(mutated)
				joined := joinAuditErrs(errs)
				if len(errs) > 0 && strings.Contains(joined, key) {
					rejected++
				} else {
					t.Errorf("invented key %q at scope %q was NOT rejected naming that key — this injection slipped through:\n%s", key, sc.scope, joined)
				}
			}
		}

		t.Logf("invented-key generality loop: %d name(s) x %d scope(s) = %d injection(s) attempted, %d rejected", len(names), len(scopes), attempted, rejected)

		if attempted < 48 {
			t.Fatalf("attempted only %d injection(s) — expected at least 48 (twelve names x four scopes); a loop that silently walks fewer iterations must fail", attempted)
		}
		if rejected != attempted {
			t.Fatalf("%d of %d injection(s) were rejected — expected all %d to be rejected", rejected, attempted, attempted)
		}
	})
}
