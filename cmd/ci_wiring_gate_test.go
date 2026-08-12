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
