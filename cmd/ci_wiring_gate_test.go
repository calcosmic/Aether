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

// blanketGateRunSubstring is the exact `go test` invocation the blanket
// release-gate step's run line must contain.
const blanketGateRunSubstring = "go test ./... -count=1 -timeout 900s"

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
// filter has fallen behind the guard tests it exists to run, if the step is
// missing entirely, or if the blanket `go test ./...` release-gate step it
// sits alongside has been narrowed, disabled, or removed. "Narrowed or
// disabled" is checked structurally by blanketReleaseGateProblem — the step
// must exist under its exact name, its run line must still contain the real
// command, and the step must have no `if:` condition, no
// `continue-on-error`, and no exit-status-swallowing fallback such as
// `|| true` — not merely have the command's text appear somewhere in the
// file, which a non-blocking decoy step can also satisfy.
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

// stepBlock locates the step whose `- name:` value is exactly stepName,
// anchored to the end of the line, and returns the block of workflow text
// running from that name line up to (but excluding) the next line at the
// same step indentation that begins a `- name:` entry, or to end of file.
//
// Anchoring to end-of-line is load-bearing: "Run Go tests" is a strict
// prefix of "Run Go tests with race detection" (ci.yml:44), so an
// unanchored substring search would resolve to the race step the moment the
// real gate step is deleted — replacing one decoy with another and
// re-creating the exact bug this function exists to fix.
//
// This is the single step locator in the package; extractWiringGateRunArg
// and blanketReleaseGateProblem both call it rather than each scanning the
// workflow text independently.
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
		if afterEnd == len(workflow) || workflow[afterEnd] == '\n' {
			lineStart := strings.LastIndexByte(workflow[:absIdx], '\n') + 1
			indent := workflow[lineStart:absIdx]
			rest := workflow[absIdx:]

			firstNL := strings.IndexByte(rest, '\n')
			if firstNL == -1 {
				return rest, nil
			}
			tail := rest[firstNL:]
			nextMarker := "\n" + indent + nameMarkerPrefix
			if nextIdx := strings.Index(tail, nextMarker); nextIdx != -1 {
				return rest[:firstNL+nextIdx], nil
			}
			return rest, nil
		}
		searchFrom = afterEnd
	}
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

// blanketReleaseGateProblem returns nil if the blanket release-gate step
// named exactly blanketGateStepName is present in workflow, its run line
// contains blanketGateRunSubstring (and is not the race-detection step's
// run line), and the step is structurally capable of failing the build —
// no `if:` condition, no `continue-on-error`, and no exit-status-swallowing
// fallback on its run line. Any other outcome returns a distinct, named
// error describing exactly which of those properties is missing.
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

	if !strings.Contains(runLine, blanketGateRunSubstring) {
		return fmt.Errorf("CI step %q's run line does not contain %q — found: %s", blanketGateStepName, blanketGateRunSubstring, trimmedRunLine)
	}
	if strings.Contains(runLine, "-race") {
		return fmt.Errorf("CI step %q's run line contains -race — the step locator resolved to the race-detection step instead of the blanket gate: %s", blanketGateStepName, trimmedRunLine)
	}

	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "if:") {
			return fmt.Errorf("CI step %q has a conditional %s — a conditional step can be arranged not to run, so it cannot be relied on to gate", blanketGateStepName, trimmed)
		}
	}

	if strings.Contains(block, "continue-on-error") {
		return fmt.Errorf("CI step %q has continue-on-error set — a step whose failure does not fail the job is not a gate", blanketGateStepName)
	}

	for _, fallback := range []string{"|| true", "|| echo", "|| :"} {
		if strings.Contains(runLine, fallback) {
			return fmt.Errorf("CI step %q's run line routes its exit status through %q, a fallback that cannot fail: %s", blanketGateStepName, fallback, trimmedRunLine)
		}
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

	tests := []struct {
		name     string
		workflow string
		wantErr  bool
		wantSub  string
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
			} else if err != nil {
				t.Fatalf("expected nil, got: %v", err)
			}
		})
	}
}

// extractWiringGateRunArg locates the step whose `- name:` value is exactly
// wiringGateStepName and returns its `run:` line's -run argument.
func extractWiringGateRunArg(workflow string) (string, error) {
	block, err := stepBlock(workflow, wiringGateStepName)
	if err != nil {
		return "", fmt.Errorf("%v — add it per D-15 so a wiring failure reads as a wiring problem, not one anonymous failure among hundreds", err)
	}

	runLine, ok := runLineOf(block)
	if !ok {
		return "", fmt.Errorf("CI step %q has no run: line", wiringGateStepName)
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
