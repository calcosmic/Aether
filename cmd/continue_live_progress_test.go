package cmd

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// setVisualOutputMode is a small test helper that sets AETHER_OUTPUT_MODE for
// the duration of the test and restores whatever value was there before. It
// also resets currentStreamingCommand to "" (always-allowed, per
// streamingAllowedForCurrentCommand's own early return) and restores it on
// cleanup: currentStreamingCommand is a package-level var saveGlobals does
// not cover, and a prior test in the same binary that ran a real cobra
// command through rootCmd.Execute() (root.go sets it on every invocation)
// can leave it pointing at a quiet-classified command name -- silently
// swallowing every emitVisualProgress call in a test that never touches
// rootCmd at all, exactly like TestCeremonyLevelGatesStreaming
// (display_house_style_test.go) already has to guard against.
func setVisualOutputMode(t *testing.T, mode string) {
	t.Helper()
	orig := os.Getenv("AETHER_OUTPUT_MODE")
	os.Setenv("AETHER_OUTPUT_MODE", mode)
	t.Cleanup(func() {
		os.Setenv("AETHER_OUTPUT_MODE", orig)
	})

	origStreamingCommand := currentStreamingCommand
	currentStreamingCommand = ""
	t.Cleanup(func() {
		currentStreamingCommand = origStreamingCommand
	})
}

// assertLiveCheckLinesForAllChecks fails the test unless the captured output
// contains both a start line ("Running {check}…") and a finish line
// ({Check} ✓/✗ ...) for every check name given (SHOW-03 D-01: two lines per
// check, live).
func assertLiveCheckLinesForAllChecks(t *testing.T, output string, checks []string) {
	t.Helper()
	lower := strings.ToLower(output)
	for _, name := range checks {
		label := verificationStepDisplayName(name)
		startNeedle := "running " + strings.ToLower(label)
		if !strings.Contains(lower, startNeedle) {
			t.Errorf("missing start line for %q (wanted %q) in output:\n%s", name, startNeedle, output)
		}
		if !strings.Contains(output, label+" ✓") && !strings.Contains(output, label+" ✗") {
			t.Errorf("missing finish line for %q in output:\n%s", name, output)
		}
	}
}

// TestBothContinueLanesEmitLiveCheckLines proves SHOW-03 D-01 holds on both
// continue lanes: the in-process lane (runCodexContinueVerification) and the
// wrapper/external lane (runCodexContinueVerificationSnapshot). Both reach
// runVerificationStep only through runDeterministicFloor, so a start line and
// a finish line for all four checks must appear on both lanes' captured
// output — a guarantee that holds only on one lane is worth nothing
// (CLAUDE.md).
func TestBothContinueLanesEmitLiveCheckLines(t *testing.T) {
	checks := []string{"build", "types", "lint", "tests"}

	t.Run("in-process lane", func(t *testing.T) {
		saveGlobals(t)
		setVisualOutputMode(t, "visual")

		var buf bytes.Buffer
		stdout = &buf

		fixture := deterministicFloorFixtures()[0] // "all checks green"
		root, phase, manifest := fixture.build(t)

		runCodexContinueVerification(context.Background(), root, colony.ColonyState{}, phase, manifest, time.Second, 5*time.Second, true)

		assertLiveCheckLinesForAllChecks(t, buf.String(), checks)
	})

	t.Run("snapshot lane", func(t *testing.T) {
		saveGlobals(t)
		setVisualOutputMode(t, "visual")

		var buf bytes.Buffer
		stdout = &buf

		fixture := deterministicFloorFixtures()[0] // "all checks green"
		root, phase, manifest := fixture.build(t)

		runCodexContinueVerificationSnapshot(root, phase, manifest, time.Now().UTC(), 5*time.Second, true)

		assertLiveCheckLinesForAllChecks(t, buf.String(), checks)
	})
}

// TestFailedCheckLineCarriesItsReasonInline proves D-02: a failing check's
// finish line carries the step's own one-line Summary inline, on the same
// line as the fail mark — never a second, invented summary format.
func TestFailedCheckLineCarriesItsReasonInline(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	var buf bytes.Buffer
	stdout = &buf

	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: false")
	phase := colony.Phase{ID: 1, Name: "One check failing"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	var failedStep codexVerificationStep
	found := false
	for _, step := range floor.Steps {
		if step.Name == "tests" {
			failedStep = step
			found = true
		}
	}
	if !found {
		t.Fatalf("no tests step found in floor: %+v", floor.Steps)
	}
	if failedStep.Passed {
		t.Fatalf("expected tests step to fail, got Passed=true: %+v", failedStep)
	}
	reason := strings.TrimSpace(failedStep.Summary)
	if reason == "" {
		t.Fatalf("expected the failing step to carry a non-empty Summary")
	}

	output := buf.String()
	var failLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "✗") && strings.Contains(strings.ToLower(line), "tests") {
			failLine = line
		}
	}
	if failLine == "" {
		t.Fatalf("no failing finish line found for tests in output:\n%s", output)
	}
	if !strings.Contains(failLine, reason) {
		t.Fatalf("failing line %q does not carry the step's own reason %q inline", failLine, reason)
	}
}

// TestVerificationStepDurationIsMeasured proves Phase 196 D-01 for the new
// field: the duration on a step that really ran a command is measured (never
// estimated), landing in a sane band around a deliberately slow fixture
// command, while a step that never ran a command (Skipped) reports zero
// duration and its finish line shows no elapsed time at all.
func TestVerificationStepDurationIsMeasured(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	var buf bytes.Buffer
	stdout = &buf

	s, root := newTestStore(t)
	store = s
	// tests: deliberately slow so its measured duration is unambiguous.
	// lint: deliberately left unresolved (no command anywhere) so it stays
	// Skipped -- no go.mod/package.json/etc in this temp root to trigger a
	// language-fallback default.
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- tests: sh -c \"sleep 0.3 && true\"")
	phase := colony.Phase{ID: 1, Name: "Duration fixture"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	var testsStep, lintStep codexVerificationStep
	for _, step := range floor.Steps {
		switch step.Name {
		case "tests":
			testsStep = step
		case "lint":
			lintStep = step
		}
	}

	if testsStep.Duration <= 0.2 || testsStep.Duration > 5.0 {
		t.Fatalf("expected tests duration in a sane band around 0.3s, got %v", testsStep.Duration)
	}
	if !lintStep.Skipped {
		t.Fatalf("expected lint to be skipped (no command resolved), got %+v", lintStep)
	}
	if lintStep.Duration != 0 {
		t.Fatalf("expected a skipped step to report zero duration, got %v", lintStep.Duration)
	}

	output := buf.String()
	var testsFinishLine, lintFinishLine string
	for _, line := range strings.Split(output, "\n") {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "tests") && (strings.Contains(line, "✓") || strings.Contains(line, "✗")) {
			testsFinishLine = line
		}
		if strings.Contains(lowerLine, "lint") && strings.Contains(lowerLine, "skip") {
			lintFinishLine = line
		}
	}
	if testsFinishLine == "" {
		t.Fatalf("no finish line found for tests in output:\n%s", output)
	}
	if !strings.Contains(testsFinishLine, "s)") {
		t.Fatalf("expected the tests finish line to show a measured elapsed time, got %q", testsFinishLine)
	}
	if lintFinishLine == "" {
		t.Fatalf("no skip line found for lint in output:\n%s", output)
	}
	if strings.ContainsAny(lintFinishLine, "0123456789") {
		t.Fatalf("expected a skipped check's line to claim no elapsed time, got %q", lintFinishLine)
	}
}

// TestLiveCheckLinesAreAppendOnly locks SEE-03: live check lines are plain,
// append-only scrollback -- no in-place repaint, no cursor-control escape
// sequences, no shrinking output. Escape sequences are built from raw
// control-character codes rather than typed as a literal escape in source,
// so this test cannot be satisfied by a comment (CLAUDE.md's Definition of
// Done: a test must be able to fail).
func TestLiveCheckLinesAreAppendOnly(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	var buf bytes.Buffer
	stdout = &buf

	_, root := newTestStore(t)

	names := []string{"build", "types", "lint", "tests"}
	prevLen := 0
	prevNonBlankLines := 0
	for _, name := range names {
		runVerificationStep(context.Background(), root, name, false, "true", 5*time.Second)

		current := buf.String()
		if len(current) < prevLen {
			t.Fatalf("output shrank after running the %q check — lines were repainted, not appended", name)
		}

		nonBlank := 0
		for _, line := range strings.Split(current, "\n") {
			if strings.TrimSpace(line) != "" {
				nonBlank++
			}
		}
		if nonBlank <= prevNonBlankLines {
			t.Fatalf("expected the live line count to grow after running %q, had %d before and %d after", name, prevNonBlankLines, nonBlank)
		}
		prevLen = len(current)
		prevNonBlankLines = nonBlank
	}

	output := buf.String()
	// Built from control-character codes: ESC (0x1b) starting a CSI sequence,
	// and a bare carriage return -- both are cursor-control moves a
	// repaintable panel would use and an append-only stream must never emit.
	escapeSequence := string([]byte{0x1b, '['})
	if strings.Contains(output, escapeSequence) {
		t.Fatalf("output contains an ANSI cursor-control escape sequence — live check lines must be append-only (SEE-03)")
	}
	if strings.Contains(output, "\r") {
		t.Fatalf("output contains a bare carriage return — live check lines must be append-only (SEE-03)")
	}
}

// TestLiveCheckLinesUseTheVisualWriter is an AST check, not a text grep
// (CLAUDE.md's Definition of Done: a comment must not be able to satisfy or
// break this). It asserts emitVerificationStepStart and
// emitVerificationStepFinish never write directly to stdout/stderr and
// always reach emitVisualProgress -- the one streaming channel whose
// writeVisualOutput exit performs command-name translation. It also proves
// that with visual output suppressed (JSON mode), a full four-check run
// writes nothing at all, so a machine-readable run is never polluted by
// progress text.
func TestLiveCheckLinesUseTheVisualWriter(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "codex_continue.go", nil, 0)
	if err != nil {
		t.Fatalf("parse codex_continue.go: %v", err)
	}

	targets := map[string]bool{
		"emitVerificationStepStart":  false,
		"emitVerificationStepFinish": false,
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if _, want := targets[fn.Name.Name]; !want {
			continue
		}
		targets[fn.Name.Name] = true

		reachesEmitVisualProgress := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fnExpr := call.Fun.(type) {
			case *ast.Ident:
				if fnExpr.Name == "emitVisualProgress" {
					reachesEmitVisualProgress = true
				}
			case *ast.SelectorExpr:
				pkg, ok := fnExpr.X.(*ast.Ident)
				if !ok || pkg.Name != "fmt" {
					return true
				}
				switch fnExpr.Sel.Name {
				case "Fprint", "Fprintf", "Fprintln":
					if len(call.Args) > 0 {
						if target, ok := call.Args[0].(*ast.Ident); ok && (target.Name == "stdout" || target.Name == "stderr") {
							t.Errorf("%s writes directly to %s instead of going through the visual writer", fn.Name.Name, target.Name)
						}
					}
				}
			}
			return true
		})

		if !reachesEmitVisualProgress {
			t.Errorf("%s does not call emitVisualProgress — live check lines must go through the one streaming channel", fn.Name.Name)
		}
	}

	for name, found := range targets {
		if !found {
			t.Errorf("expected to find function %s declared in codex_continue.go", name)
		}
	}

	// With visual output suppressed (JSON mode), running the full four-check
	// floor must write nothing to stdout at all.
	saveGlobals(t)
	setVisualOutputMode(t, "json")

	var buf bytes.Buffer
	stdout = &buf

	s, root := newTestStore(t)
	store = s
	writeAgentsVerificationCommands(t, root,
		"- build: true", "- types: true", "- lint: true", "- tests: true")
	phase := colony.Phase{ID: 1, Name: "JSON mode fixture"}
	manifest := codexContinueManifest{}
	runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	if buf.Len() != 0 {
		t.Fatalf("expected no progress output when AETHER_OUTPUT_MODE=json, got %q", buf.String())
	}
}

// TestVerificationStepDisplayNameIsRuneSafe pins WR-03 (198-REVIEW.md):
// verificationStepDisplayName's fallback branch labels an arbitrary,
// CLAUDE.md-configured check name. Slicing the first BYTE
// (trimmed[:1] + trimmed[1:]) panics on a check name whose first character
// takes more than one byte to store. Taking the first RUNE degrades
// gracefully instead. The four known keys today ("build", "types", "lint",
// "tests") are all ASCII, so this was latent rather than currently
// triggered -- this test exercises the previously-panicking case directly.
func TestVerificationStepDisplayNameIsRuneSafe(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "école", want: "École"},
		{name: "日本語", want: "日本語"},
		{name: "custom_check", want: "Custom_check"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("verificationStepDisplayName(%q) panicked: %v", tc.name, r)
				}
			}()
			got := verificationStepDisplayName(tc.name)
			if got != tc.want {
				t.Fatalf("verificationStepDisplayName(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}
