package cmd

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 19 -- the D-03/D-05/D-07/D-08/CAP-066 wiring root cause fix.
// These tests prove buildWorkCloseoutDetails (cmd/work_closeout.go) resolves
// a real sealed build attempt to a declared verdict, that a real build's
// finished screen actually carries it on both lanes, and that the wiring
// cannot be quietly disconnected again.

// ---------------------------------------------------------------------------
// TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus
// ---------------------------------------------------------------------------

// workCloseoutTerminalStatuses names the four declared terminal attempt
// status constants (buildAttemptStatusTerminal's own switch,
// cmd/build_attempt.go) -- the named Go constants themselves, never a
// retyped string literal.
var workCloseoutTerminalStatuses = []string{
	buildAttemptBuilt,
	buildAttemptFailed,
	buildAttemptInterrupted,
	buildAttemptPartial,
}

// workCloseoutTestPhaseCounter hands out a fresh phase number per fixture
// attempt within one shared test store, so sub-tests never collide on the
// same phase's "latest attempt" pointer.
type workCloseoutTestPhaseCounter struct{ next int }

func (c *workCloseoutTestPhaseCounter) id() int {
	c.next++
	return c.next
}

// buildEndReviewerFixtureDispatches builds one dispatch per post-wave
// reviewer caste, derived from queenBuildPostWavePlans -- the SAME source
// buildEndReviewerCastes (cmd/work_closeout.go) derives from -- rather than
// a second, independently typed caste list. status is the raw per-dispatch
// status every reviewer reports.
func buildEndReviewerFixtureDispatches(status string) []codexBuildDispatch {
	dispatches := make([]codexBuildDispatch, 0, len(queenBuildPostWavePlans))
	for i, plan := range queenBuildPostWavePlans {
		dispatches = append(dispatches, codexBuildDispatch{
			Caste:  plan.caste,
			Name:   fmt.Sprintf("Reviewer-%d", i+1),
			Status: status,
		})
	}
	return dispatches
}

// TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus is Task 1's own
// acceptance proof: every terminal attempt status resolves to SOME declared
// verdict (iterated from the real status constants), every declared verdict
// (colony.AllWorkOutcomes()) is producible from some genuinely constructible
// attempt (built through the real attempt writers -- deriveBuildAttempt,
// attachBuildFreeCheckReport, attachVerificationBoundary -- never a
// hand-written attempt JSON shape the runtime cannot produce), the
// no-check-report case never claims checks passed, the check-step case
// never carries the success verdict, and the attempt's own knowledge deltas
// reach the resolved details' evidence.
func TestBuildWorkCloseoutDetailsCoversEveryTerminalStatus(t *testing.T) {
	assertWorkingTreeUnchanged(t)
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	counter := &workCloseoutTestPhaseCounter{}

	t.Run("no sealed attempt at all reports not-ok", func(t *testing.T) {
		phaseID := counter.id()
		if _, ok := buildWorkCloseoutDetails(phaseID); ok {
			t.Fatalf("phase %d has no attempt at all, want ok=false", phaseID)
		}
	})

	t.Run("every terminal status resolves to a declared verdict", func(t *testing.T) {
		for _, status := range workCloseoutTerminalStatuses {
			status := status
			t.Run(status, func(t *testing.T) {
				phaseID := counter.id()
				rel, _ := seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-status-%d", phaseID), func(r *buildAttemptRecord) {
					r.Status = status
					if status == buildAttemptFailed {
						r.Error = "a worker reported a genuine blocker"
					}
				})
				if status == buildAttemptBuilt {
					// The built status alone is not yet a declared branch --
					// buildBuiltStatusCloseoutDetails needs SOME resolution
					// path (here: no check report at all, the honest default).
					_ = rel
				}
				details, ok := buildWorkCloseoutDetails(phaseID)
				if !ok {
					t.Fatalf("status %q did not resolve to a closeout at all", status)
				}
				if !details.WorkOutcome.Valid() {
					t.Fatalf("status %q resolved to an undeclared verdict %q", status, details.WorkOutcome)
				}
			})
		}
	})

	t.Run("every declared verdict is producible from some genuinely constructible attempt", func(t *testing.T) {
		for _, verdict := range colony.AllWorkOutcomes() {
			verdict := verdict
			t.Run(string(verdict), func(t *testing.T) {
				phaseID := counter.id()
				attemptID := fmt.Sprintf("attempt-verdict-%d", phaseID)
				var rel string
				switch verdict {
				case colony.WorkOutcomeSuccess:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptBuilt
						r.Dispatches = buildEndReviewerFixtureDispatches("completed")
					})
					if err := attachBuildFreeCheckReport(rel, buildFreeCheckReport{Passed: true, ChecksRun: []string{"go build ./..."}}); err != nil {
						t.Fatalf("attach free checks: %v", err)
					}
					if err := attachVerificationBoundary(rel, verificationBoundaryDecision{
						Choice: verificationBoundaryChoiceBuildEnd,
						Source: verificationBoundarySourceQueen,
						Reason: "release-gated phase",
					}); err != nil {
						t.Fatalf("attach verification boundary: %v", err)
					}
				case colony.WorkOutcomeNoChange:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptBuilt
						r.Dispatches = []codexBuildDispatch{{Caste: "builder", Name: "Mason-1", Status: "no_change"}}
					})
				case colony.WorkOutcomePartial:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptPartial
					})
				case colony.WorkOutcomeBlocker:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptFailed
						r.Error = "a worker reported a schema conflict"
					})
				case colony.WorkOutcomeTimeout:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptFailed
						r.Dispatches = []codexBuildDispatch{{Caste: "builder", Name: "Mason-1", Status: "timed_out"}}
					})
				case colony.WorkOutcomeInterrupted:
					rel, _ = seedBuildAttemptForTest(t, phaseID, attemptID, func(r *buildAttemptRecord) {
						r.Status = buildAttemptInterrupted
					})
				default:
					t.Fatalf("no fixture recipe for declared verdict %q -- add one", verdict)
				}
				_ = rel

				details, ok := buildWorkCloseoutDetails(phaseID)
				if !ok {
					t.Fatalf("verdict %q: expected a resolved closeout, got none", verdict)
				}
				if details.WorkOutcome != verdict {
					t.Fatalf("verdict %q: resolved %q instead", verdict, details.WorkOutcome)
				}
			})
		}
	})

	t.Run("no check report at all never claims checks passed", func(t *testing.T) {
		phaseID := counter.id()
		seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-nocheck-%d", phaseID), func(r *buildAttemptRecord) {
			r.Status = buildAttemptBuilt
		})
		details, ok := buildWorkCloseoutDetails(phaseID)
		if !ok {
			t.Fatalf("expected a resolved closeout")
		}
		if details.WorkOutcome == colony.WorkOutcomeSuccess {
			t.Fatalf("a built attempt with no check report carries the success verdict")
		}
		lowered := strings.ToLower(details.Summary)
		if strings.Contains(lowered, "checks passed") {
			t.Fatalf("no-check-report summary falsely claims checks passed: %q", details.Summary)
		}
		if !strings.Contains(lowered, "have not run") {
			t.Fatalf("no-check-report summary does not say the checks have not run yet: %q", details.Summary)
		}
	})

	t.Run("built with the check-step default and passing checks is never the success verdict", func(t *testing.T) {
		phaseID := counter.id()
		rel, _ := seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-checkstep-%d", phaseID), func(r *buildAttemptRecord) {
			r.Status = buildAttemptBuilt
		})
		if err := attachBuildFreeCheckReport(rel, buildFreeCheckReport{Passed: true, ChecksRun: []string{"go build ./...", "go vet ./..."}}); err != nil {
			t.Fatalf("attach free checks: %v", err)
		}
		details, ok := buildWorkCloseoutDetails(phaseID)
		if !ok {
			t.Fatalf("expected a resolved closeout")
		}
		if details.WorkOutcome == colony.WorkOutcomeSuccess {
			t.Fatalf("check-step boundary with no reviewer carries the success verdict: %+v", details)
		}
	})

	t.Run("a failing check report is a blocker, naming the failed checks", func(t *testing.T) {
		phaseID := counter.id()
		rel, _ := seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-failcheck-%d", phaseID), func(r *buildAttemptRecord) {
			r.Status = buildAttemptBuilt
		})
		if err := attachBuildFreeCheckReport(rel, buildFreeCheckReport{Passed: false, ChecksRun: []string{"go build ./...", "go test ./..."}, Failed: []string{"go test ./..."}}); err != nil {
			t.Fatalf("attach free checks: %v", err)
		}
		details, ok := buildWorkCloseoutDetails(phaseID)
		if !ok {
			t.Fatalf("expected a resolved closeout")
		}
		if details.WorkOutcome != colony.WorkOutcomeBlocker {
			t.Fatalf("failing checks resolved to %q, want blocker", details.WorkOutcome)
		}
		if !strings.Contains(details.Summary, "go test ./...") {
			t.Fatalf("blocker summary does not name the failed check: %q", details.Summary)
		}
	})

	t.Run("knowledge deltas reach the resolved details as evidence", func(t *testing.T) {
		phaseID := counter.id()
		rel, _ := seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-deltas-%d", phaseID), func(r *buildAttemptRecord) {
			r.Status = buildAttemptPartial
		})
		wantSummary := "Chose Postgres for the write path."
		if err := attachBuildKnowledgeDeltas(rel, []buildAttemptKnowledgeDelta{{Kind: "decision", Summary: wantSummary}}); err != nil {
			t.Fatalf("attach knowledge deltas: %v", err)
		}
		details, ok := buildWorkCloseoutDetails(phaseID)
		if !ok {
			t.Fatalf("expected a resolved closeout")
		}
		found := false
		for _, e := range details.Evidence {
			if strings.Contains(e.Summary, wantSummary) {
				found = true
			}
		}
		if !found {
			t.Fatalf("resolved details evidence %+v does not carry the attempt's own knowledge delta %q", details.Evidence, wantSummary)
		}
	})

	t.Run("renderBuildResultFileSection is empty for an attempt with no recorded files", func(t *testing.T) {
		phaseID := counter.id()
		seedBuildAttemptForTest(t, phaseID, fmt.Sprintf("attempt-nofiles-%d", phaseID), func(r *buildAttemptRecord) {
			r.Status = buildAttemptBuilt
		})
		if got := renderBuildResultFileSection(phaseID); got != "" {
			t.Fatalf("expected empty file section for an attempt with no recorded files, got %q", got)
		}
	})
}

// ---------------------------------------------------------------------------
// TestBuildCloseoutCarriesTheRealVerdict
// ---------------------------------------------------------------------------

// workCloseoutNextActionCardHeading is the exact rendered "What Next" banner
// text appendLifecycleCloseoutVisual itself searches for (its own "legacy"
// variable, cmd/lifecycle_closeout.go) -- renderBanner letter-spaces its
// title ("W H A T   N E X T"), so a literal "What Next" substring search
// never matches the rendered screen. Deriving this from the same production
// call, rather than a hand-typed spaced string, is what keeps this count
// correct if the banner's rendering ever changes.
func workCloseoutNextActionCardHeading() string {
	return strings.TrimRight(renderBanner(commandEmoji("status"), "What Next"), "\n")
}

// workCloseoutRealBuildFixture drives one real native build to completion
// (the same fixture shape TestBuildWritesDispatchArtifactsAndUpdatesState
// uses) and returns the sealed phase number plus the captured direct-lane
// screen.
func workCloseoutRealBuildFixture(t *testing.T) (phaseID int, directScreen string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	goal := "Prove the closeout carries the real verdict"
	taskID := "1.1"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "full",
		CurrentPhase: 0,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:          1,
					Name:        "Prove the wiring",
					Description: "One task, one worker, a real sealed attempt",
					Status:      colony.PhaseReady,
					Tasks: []colony.Task{
						{ID: &taskID, Goal: "Do the one thing this phase asks for", Status: colony.TaskPending},
					},
					SuccessCriteria: []string{"The task is done"},
				},
			},
		},
	})

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(accepted.Root); err != nil {
		t.Fatalf("chdir to test root: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })

	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	buf, ok := stdout.(*bytes.Buffer)
	if !ok {
		t.Fatalf("stdout is not a *bytes.Buffer in test mode: %T", stdout)
	}
	return 1, buf.String()
}

// TestBuildCloseoutCarriesTheRealVerdict drives a real build to completion,
// then proves both build lanes' finished screens carry the verdict
// buildWorkCloseoutDetails derives for that exact sealed attempt: the
// direct lane's own live ending screen (the native lane never attaches a
// free-check report today, so it honestly renders the no-check-report
// partial verdict), and the wrapper's aether ceremony closeout for the same
// attempt after the evidence an external build-finalize would additionally
// have recorded (free checks, credited/uncredited files, knowledge deltas)
// is attached through the exact same production writers those lanes use.
func TestBuildCloseoutCarriesTheRealVerdict(t *testing.T) {
	phaseID, directScreen := workCloseoutRealBuildFixture(t)

	// --- Direct lane: the live screen from the real CLI run. ---
	preDetails, ok := buildWorkCloseoutDetails(phaseID)
	if !ok {
		t.Fatalf("no closeout resolved for the just-sealed attempt")
	}
	labels := colony.WorkOutcomeLabels()
	wantPreLabel := labels[preDetails.WorkOutcome]
	if wantPreLabel == "" {
		t.Fatalf("resolved verdict %q has no label", preDetails.WorkOutcome)
	}
	if !strings.Contains(directScreen, wantPreLabel) {
		t.Fatalf("direct lane screen does not carry the resolved verdict label %q:\n%s", wantPreLabel, directScreen)
	}
	if got := strings.Count(stripANSI(directScreen), spendCostLineHeading); got != 1 {
		t.Fatalf("direct lane screen carries %d cost-and-time block(s), want exactly 1:\n%s", got, directScreen)
	}
	if got := strings.Count(stripANSI(directScreen), workCloseoutNextActionCardHeading()); got != 1 {
		t.Fatalf("direct lane screen carries %d next-action card(s), want exactly 1:\n%s", got, directScreen)
	}

	// --- Wrapper lane: attach the evidence an external build-finalize would
	// additionally record, through the exact same production writers, then
	// render aether ceremony closeout for the same sealed attempt. ---
	attemptRel, attempt, ok := loadLatestBuildAttempt(phaseID)
	if !ok {
		t.Fatalf("no sealed attempt to attach evidence to")
	}
	checkCommands := []string{"go build ./...", "go vet ./...", "go test ./..."}
	if err := attachBuildFreeCheckReport(attemptRel, buildFreeCheckReport{Passed: true, ChecksRun: checkCommands}); err != nil {
		t.Fatalf("attach free checks: %v", err)
	}
	creditedFile := "cmd/example.go"
	uncreditedFile := buildAttemptUncreditedFile{Path: "cmd/orphan.go", Location: "the repository"}
	if err := attachResultFilePrecision(attemptRel, []string{creditedFile}, []buildAttemptUncreditedFile{uncreditedFile}); err != nil {
		t.Fatalf("attach result file precision: %v", err)
	}
	deltaSummary := "Chose Postgres for the write path."
	if err := attachBuildKnowledgeDeltas(attemptRel, []buildAttemptKnowledgeDelta{{Kind: "decision", Summary: deltaSummary}}); err != nil {
		t.Fatalf("attach knowledge deltas: %v", err)
	}
	_ = attempt

	postDetails, ok := buildWorkCloseoutDetails(phaseID)
	if !ok {
		t.Fatalf("no closeout resolved after attaching evidence")
	}
	wantPostLabel := labels[postDetails.WorkOutcome]
	if wantPostLabel == "" {
		t.Fatalf("resolved verdict %q has no label", postDetails.WorkOutcome)
	}

	_, wrapperScreen := renderCeremonyCloseout("build", "")

	if !strings.Contains(wrapperScreen, wantPostLabel) {
		t.Fatalf("wrapper closeout does not carry the resolved verdict label %q:\n%s", wantPostLabel, wrapperScreen)
	}
	successLabel := labels[colony.WorkOutcomeSuccess]
	if postDetails.WorkOutcome != colony.WorkOutcomeSuccess && strings.Contains(wrapperScreen, successLabel) {
		t.Fatalf("wrapper closeout carries the success verdict's own label %q for a non-success verdict %q:\n%s", successLabel, postDetails.WorkOutcome, wrapperScreen)
	}
	lowered := strings.ToLower(wrapperScreen)
	if !strings.Contains(lowered, "not been verified") {
		t.Fatalf("wrapper closeout does not plainly say verification has not run yet:\n%s", wrapperScreen)
	}

	// Recommended action: a runnable command, a one-sentence reason, and the
	// alternatives beneath it (D-07).
	action, actionErr := recommendedActionForWorkOutcome(postDetails.WorkOutcome, attempt)
	if actionErr != nil {
		t.Fatalf("recommendedActionForWorkOutcome: %v", actionErr)
	}
	if !strings.Contains(wrapperScreen, action.Command) {
		t.Fatalf("wrapper closeout does not name the recommended command %q:\n%s", action.Command, wrapperScreen)
	}
	if !strings.Contains(wrapperScreen, action.Reason) {
		t.Fatalf("wrapper closeout does not carry the recommended action's reason %q:\n%s", action.Reason, wrapperScreen)
	}
	for _, alt := range action.Alternatives {
		if !strings.Contains(wrapperScreen, alt) {
			t.Fatalf("wrapper closeout does not list the alternative %q:\n%s", alt, wrapperScreen)
		}
	}

	// Credited/uncredited file card, each uncredited file's location, and
	// the knowledge delta (D-08/CAP-066).
	for _, want := range []string{creditedFile, uncreditedFile.Path, uncreditedFile.Location, deltaSummary} {
		if !strings.Contains(wrapperScreen, want) {
			t.Fatalf("wrapper closeout does not carry %q:\n%s", want, wrapperScreen)
		}
	}

	if got := strings.Count(stripANSI(wrapperScreen), spendCostLineHeading); got != 1 {
		t.Fatalf("wrapper closeout carries %d cost-and-time block(s), want exactly 1:\n%s", got, wrapperScreen)
	}
	if got := strings.Count(stripANSI(wrapperScreen), workCloseoutNextActionCardHeading()); got != 1 {
		t.Fatalf("wrapper closeout carries %d next-action card(s), want exactly 1:\n%s", got, wrapperScreen)
	}
}

// ---------------------------------------------------------------------------
// TestBuildCloseoutVerdictHasProductionCallers
// ---------------------------------------------------------------------------

// workCloseoutVerdictTargets are the two resolver functions this guard
// requires at least one production (non-test) caller for, and requires both
// named build render paths to reach.
var workCloseoutVerdictTargets = []string{
	"buildWorkCloseoutDetails",
	"renderBuildResultFileSection",
}

// workCloseoutFindCobraRunE parses filename and returns the RunE *ast.FuncLit
// of the first &cobra.Command{} composite literal whose Use field starts
// with usePrefix -- the same shape next_action_hardcode_ratchet_test.go's
// own cobra-literal scan already uses (that scan is package-wide; this one
// is scoped to one file and one command, since buildCmd's RunE is an
// anonymous closure continueDecisionPackageFuncs cannot see -- it only
// indexes named top-level function declarations).
func workCloseoutFindCobraRunE(t *testing.T, fset *token.FileSet, filename, usePrefix string) *ast.FuncLit {
	t.Helper()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	var found *ast.FuncLit
	ast.Inspect(file, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		comp, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := comp.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Command" {
			return true
		}
		var useVal string
		var runE *ast.FuncLit
		for _, elt := range comp.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Use":
				if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if v, unquoteErr := strconv.Unquote(lit.Value); unquoteErr == nil {
						useVal = v
					}
				}
			case "RunE":
				if lit, ok := kv.Value.(*ast.FuncLit); ok {
					runE = lit
				}
			}
		}
		if runE != nil && strings.HasPrefix(useVal, usePrefix) {
			found = runE
		}
		return true
	})
	return found
}

// workCloseoutBodyCallsTarget reports whether target is invoked anywhere
// within body, including inside nested closures -- ast.Inspect naturally
// descends into a nested *ast.FuncLit, so a call made inside an inner
// "if details, ok := buildWorkCloseoutDetails(...); ok { ... }" block is
// still found.
func workCloseoutBodyCallsTarget(body ast.Node, target string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == target {
			found = true
		}
		return true
	})
	return found
}

// TestBuildCloseoutVerdictHasProductionCallers is the structural guard:
// buildWorkCloseoutDetails and renderBuildResultFileSection each have at
// least one direct caller among the package's non-test functions, and both
// named build render paths -- the wrapper's renderCeremonyCloseout and the
// direct lane's buildCmd RunE closure -- reach both of them. A synthetic
// disconnection is refused by name and file position, proving the guard can
// actually fail.
func TestBuildCloseoutVerdictHasProductionCallers(t *testing.T) {
	assertWorkingTreeUnchanged(t)

	t.Run("both resolvers have at least one production caller", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)
		for _, target := range workCloseoutVerdictTargets {
			if _, ok := funcs[target]; !ok {
				t.Fatalf("%s is not declared in the cmd package", target)
			}
			if callers := continueDecisionDirectCallers(funcs, target); len(callers) == 0 {
				t.Errorf("%s has zero callers outside _test.go files -- the resolver is orphaned", target)
			}
		}
	})

	t.Run("both build render paths reach both targets", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := continueDecisionPackageFuncs(t, fset)

		wrapperFn, ok := funcs["renderCeremonyCloseout"]
		if !ok {
			t.Fatal("renderCeremonyCloseout is not declared in the cmd package")
		}
		for _, target := range workCloseoutVerdictTargets {
			if !workCloseoutBodyCallsTarget(wrapperFn.Body, target) {
				t.Errorf("renderCeremonyCloseout (the wrapper's build closeout) does not reach %s", target)
			}
		}

		directRunE := workCloseoutFindCobraRunE(t, fset, "codex_workflow_cmds.go", "build ")
		if directRunE == nil {
			t.Fatal("buildCmd's RunE closure was not found in codex_workflow_cmds.go")
		}
		for _, target := range workCloseoutVerdictTargets {
			if !workCloseoutBodyCallsTarget(directRunE.Body, target) {
				t.Errorf("buildCmd's direct-dispatch RunE does not reach %s", target)
			}
		}
	})

	t.Run("guard fails a synthetic disconnection by name and position", func(t *testing.T) {
		src := `package cmd

func renderCeremonyCloseout(workflow, completionFile string) (map[string]interface{}, string) {
	return nil, ""
}
`
		fset := token.NewFileSet()
		extra, err := parser.ParseFile(fset, "fixture_disconnected_work_closeout_lane.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic disconnection fixture: %v", err)
		}
		funcs := continueDecisionPackageFuncs(t, fset, extra)
		wrapperFn, ok := funcs["renderCeremonyCloseout"]
		if !ok {
			t.Fatal("synthetic fixture did not register renderCeremonyCloseout")
		}

		disconnected := false
		var offenderTarget string
		for _, target := range workCloseoutVerdictTargets {
			if !workCloseoutBodyCallsTarget(wrapperFn.Body, target) {
				disconnected = true
				offenderTarget = target
				break
			}
		}
		if !disconnected {
			t.Fatalf("expected the guard to flag the synthetically disconnected renderCeremonyCloseout, but it reached every target")
		}

		pos := fset.Position(wrapperFn.Pos())
		if pos.Filename == "" || pos.Line == 0 {
			t.Fatalf("violation carries no file/line position: %+v", pos)
		}
		if !strings.Contains(pos.Filename, "fixture_disconnected_work_closeout_lane.go") {
			t.Fatalf("violation position %+v does not name the synthetic offender's file", pos)
		}
		t.Logf("guard correctly refused renderCeremonyCloseout (missing %s) at %s:%d", offenderTarget, pos.Filename, pos.Line)
	})
}

// ---------------------------------------------------------------------------
// TestCheckWorkOutcome (Plan 201-20, Task 1)
// ---------------------------------------------------------------------------

// TestCheckWorkOutcome proves checkWorkOutcome (cmd/work_closeout.go) is
// total over continueAcceptVerifyAdvanceDecision's two declared verdict
// values, that every one of the six declared work verdicts
// (colony.AllWorkOutcomes()) is reachable from some genuinely constructible
// decision-and-report pair, and that it never falls through to success by
// default.
func TestCheckWorkOutcome(t *testing.T) {
	reached := map[colony.WorkOutcome]bool{}

	cases := []struct {
		name     string
		decision continueAcceptVerifyAdvanceDecision
		report   codexContinueVerificationReport
		want     colony.WorkOutcome
	}{
		{
			name:     "advancing with executed steps and no partial success is success",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictAdvance},
			report: codexContinueVerificationReport{
				Steps: []codexVerificationStep{{Name: "tests", Passed: true}},
			},
			want: colony.WorkOutcomeSuccess,
		},
		{
			name:     "advancing with the decision's own partial-success flag is partial",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictAdvance, PartialSuccess: true},
			report: codexContinueVerificationReport{
				Steps: []codexVerificationStep{{Name: "tests", Passed: true}},
			},
			want: colony.WorkOutcomePartial,
		},
		{
			name:     "advancing with no executed steps is no-change",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictAdvance},
			report:   codexContinueVerificationReport{},
			want:     colony.WorkOutcomeNoChange,
		},
		{
			name:     "blocked with a timed-out step is timeout, even when partial success is also set",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictBlock, PartialSuccess: true},
			report: codexContinueVerificationReport{
				Steps: []codexVerificationStep{{Name: "tests", TimedOut: true}},
			},
			want: colony.WorkOutcomeTimeout,
		},
		{
			name:     "blocked with no executed steps is interrupted",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictBlock},
			report:   codexContinueVerificationReport{},
			want:     colony.WorkOutcomeInterrupted,
		},
		{
			name:     "blocked with executed steps and no timeout is blocker",
			decision: continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdictBlock, BlockingReasons: []string{"tests failed"}},
			report: codexContinueVerificationReport{
				Steps: []codexVerificationStep{{Name: "tests", Passed: false}},
			},
			want: colony.WorkOutcomeBlocker,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := checkWorkOutcome(tc.decision, tc.report)
			if got != tc.want {
				t.Fatalf("checkWorkOutcome() = %q, want %q", got, tc.want)
			}
			if !got.Valid() {
				t.Fatalf("checkWorkOutcome() returned an invalid verdict %q", got)
			}
			reached[got] = true
		})
	}

	for _, verdict := range colony.AllWorkOutcomes() {
		if !reached[verdict] {
			t.Errorf("verdict %q is never reachable from checkWorkOutcome", verdict)
		}
	}

	t.Run("an undeclared decision verdict never defaults to success", func(t *testing.T) {
		got := checkWorkOutcome(continueAcceptVerifyAdvanceDecision{Verdict: continueAdvanceVerdict("bogus")}, codexContinueVerificationReport{})
		if got == colony.WorkOutcomeSuccess {
			t.Fatalf("an undeclared decision verdict resolved to success -- checkWorkOutcome must refuse by name, never default to success")
		}
		if got.Valid() {
			t.Fatalf("an undeclared decision verdict resolved to a declared verdict %q -- want the invalid zero value", got)
		}
	})
}
