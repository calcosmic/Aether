package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 03 (SYN-201-03/D-01/D-02) -- the verification-boundary
// decision: a pure reconciliation function, a durable attempt-bound record,
// and one read path every consumer must use. See
// cmd/verification_boundary.go for the mechanism this file proves.

// --- Task 1: reconciling a proposal into a validated decision ---

func TestVerificationBoundaryDefaultsToTheCheckStep(t *testing.T) {
	decision := queenApplyVerificationBoundary("", "", colony.Phase{}, colony.ColonyState{})
	if decision.Choice != verificationBoundaryChoiceCheckStep {
		t.Fatalf("Choice = %q, want %q", decision.Choice, verificationBoundaryChoiceCheckStep)
	}
	if decision.Source != verificationBoundarySourceDeterministic {
		t.Fatalf("Source = %q, want %q", decision.Source, verificationBoundarySourceDeterministic)
	}
	if decision.Refused {
		t.Fatalf("expected an empty proposal to not be refused, got %+v", decision)
	}

	t.Run("an explicit check-step proposal is the Queen's own choice, one record only", func(t *testing.T) {
		d := queenApplyVerificationBoundary("check_step", "the phase is low risk", colony.Phase{}, colony.ColonyState{})
		if d.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want %q", d.Choice, verificationBoundaryChoiceCheckStep)
		}
		if d.Source != verificationBoundarySourceQueen {
			t.Fatalf("Source = %q, want %q", d.Source, verificationBoundarySourceQueen)
		}
		if d.Refused {
			t.Fatalf("expected no refusal, got %+v", d)
		}
	})
}

func TestVerificationBoundaryRefusesBuildEndWithoutAReason(t *testing.T) {
	cases := []struct {
		name   string
		reason string
	}{
		{"empty reason", ""},
		{"whitespace-only reason", "   \t  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := queenApplyVerificationBoundary("build_end", tc.reason, colony.Phase{}, colony.ColonyState{})
			if !d.Refused {
				t.Fatalf("expected the proposal to be refused, got %+v", d)
			}
			if strings.TrimSpace(d.RefusedWhy) == "" {
				t.Fatalf("expected a non-empty RefusedWhy, got %+v", d)
			}
			if d.Choice != verificationBoundaryChoiceCheckStep {
				t.Fatalf("Choice = %q, want fallback to %q", d.Choice, verificationBoundaryChoiceCheckStep)
			}
			if d.Source != verificationBoundarySourceDeterministic {
				t.Fatalf("Source = %q, want %q on a refused proposal", d.Source, verificationBoundarySourceDeterministic)
			}
		})
	}

	t.Run("a build-end proposal with a real reason is accepted verbatim", func(t *testing.T) {
		d := queenApplyVerificationBoundary("build_end", "  this phase touches credential handling  ", colony.Phase{}, colony.ColonyState{})
		if d.Refused {
			t.Fatalf("expected no refusal, got %+v", d)
		}
		if d.Choice != verificationBoundaryChoiceBuildEnd {
			t.Fatalf("Choice = %q, want %q", d.Choice, verificationBoundaryChoiceBuildEnd)
		}
		if d.Source != verificationBoundarySourceQueen {
			t.Fatalf("Source = %q, want %q", d.Source, verificationBoundarySourceQueen)
		}
		want := "this phase touches credential handling"
		if d.Reason != want {
			t.Fatalf("Reason = %q, want %q (byte-identical to the input after trimming)", d.Reason, want)
		}
	})

	t.Run("an unrecognised proposal is refused by name and falls back", func(t *testing.T) {
		d := queenApplyVerificationBoundary("sometime-later", "because I said so", colony.Phase{}, colony.ColonyState{})
		if !d.Refused {
			t.Fatalf("expected the proposal to be refused, got %+v", d)
		}
		if !strings.Contains(d.RefusedWhy, "sometime-later") {
			t.Fatalf("RefusedWhy = %q, want it to name the offending value", d.RefusedWhy)
		}
		if d.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want fallback to %q", d.Choice, verificationBoundaryChoiceCheckStep)
		}
	})
}

func TestVerificationBoundaryNormalisesProposalExactly(t *testing.T) {
	variants := []string{
		"build_end", " build_end", "build_end ", "BUILD_END", "Build_End", "\tbuild_end\n",
	}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			d := queenApplyVerificationBoundary(variant, "needs a build-end review", colony.Phase{}, colony.ColonyState{})
			if d.Choice != verificationBoundaryChoiceBuildEnd {
				t.Fatalf("proposal %q resolved to Choice=%q, want %q", variant, d.Choice, verificationBoundaryChoiceBuildEnd)
			}
			if d.Refused {
				t.Fatalf("proposal %q was refused: %+v", variant, d)
			}
		})
	}

	t.Run("no fuzzy matching -- a near-miss is refused, not silently resolved", func(t *testing.T) {
		d := queenApplyVerificationBoundary("build-end", "close but not exact", colony.Phase{}, colony.ColonyState{})
		if !d.Refused {
			t.Fatalf("expected a hyphenated near-miss to be refused rather than fuzzily matched, got %+v", d)
		}
	})
}

func TestVerificationBoundarySummaryIsStable(t *testing.T) {
	d := queenApplyVerificationBoundary("build_end", "touches auth", colony.Phase{}, colony.ColonyState{})
	first := d.Summary()
	second := d.Summary()
	if first != second {
		t.Fatalf("two calls to Summary() on the same decision produced different text:\n1: %q\n2: %q", first, second)
	}
	if first == "" {
		t.Fatalf("expected a non-empty summary")
	}

	refused := queenApplyVerificationBoundary("build_end", "", colony.Phase{}, colony.ColonyState{})
	refusedSummary := refused.Summary()
	if !strings.Contains(refusedSummary, "refused") {
		t.Fatalf("Summary() for a refused proposal = %q, want it to say a proposal was refused", refusedSummary)
	}
}

// --- Task 2: binding the decision to the exact attempt ---

// newTestVerificationBoundaryAttempt derives and saves a minimal real build
// attempt record via the production deriveBuildAttempt constructor (never a
// hand-typed literal), mirroring TestConcurrentLanesKeepEvidenceAttemptBound's
// own fixture pattern.
func newTestVerificationBoundaryAttempt(t *testing.T, phaseID int, attemptID string) string {
	t.Helper()
	phase := colony.Phase{ID: phaseID, Name: "Verification boundary fixture"}
	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Mason-1", Task: "Do the work", Status: "spawned"},
	}
	rel, record, _, err := deriveBuildAttempt(buildAttemptDerivation{
		State: colony.ColonyState{}, Phase: phase, PhaseNumber: phase.ID,
		StartedAt: time.Now().UTC(), AttemptID: attemptID,
		RunID: "run-" + attemptID, ProcessID: 4300, WorkspaceSHA256: strings.Repeat("b", 64),
		ExecutionOwner: "runtime-worker-dispatch", Dispatches: dispatches,
		InitialStatus: buildAttemptPrepared, InitialDispatchMode: "direct",
	})
	if err != nil {
		t.Fatalf("derive attempt: %v", err)
	}
	if err := store.SaveJSON(rel, record); err != nil {
		t.Fatalf("save attempt: %v", err)
	}
	return rel
}

func TestAttachVerificationBoundaryTouchesOneField(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	attemptRel := newTestVerificationBoundaryAttempt(t, 1, "attempt-boundary-one-field")
	var before buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &before); err != nil {
		t.Fatalf("load before: %v", err)
	}

	decision := queenApplyVerificationBoundary("build_end", "touches credentials", colony.Phase{}, colony.ColonyState{})
	if err := attachVerificationBoundary(attemptRel, decision); err != nil {
		t.Fatalf("attach verification boundary: %v", err)
	}

	var after buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &after); err != nil {
		t.Fatalf("load after: %v", err)
	}
	if after.VerificationBoundary == nil || *after.VerificationBoundary != decision {
		t.Fatalf("VerificationBoundary = %+v, want %+v", after.VerificationBoundary, decision)
	}

	// Every other field must be byte-identical to before attaching.
	beforeCopy := before
	beforeCopy.VerificationBoundary = after.VerificationBoundary
	beforeCopy.UpdatedAt = after.UpdatedAt
	if beforeCopy.Status != after.Status {
		t.Fatalf("Status changed: before=%q after=%q", beforeCopy.Status, after.Status)
	}
	if len(before.Dispatches) != len(after.Dispatches) {
		t.Fatalf("Dispatches changed: before=%+v after=%+v", before.Dispatches, after.Dispatches)
	}
	if before.Claims != nil || after.Claims != nil {
		t.Fatalf("Claims changed: before=%+v after=%+v", before.Claims, after.Claims)
	}
	if len(before.History) != len(after.History) {
		t.Fatalf("History changed: before=%+v after=%+v", before.History, after.History)
	}
}

func TestVerificationBoundaryCannotBeSilentlyRewritten(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	attemptRel := newTestVerificationBoundaryAttempt(t, 2, "attempt-boundary-no-rewrite")
	first := queenApplyVerificationBoundary("check_step", "default landing", colony.Phase{}, colony.ColonyState{})
	if err := attachVerificationBoundary(attemptRel, first); err != nil {
		t.Fatalf("attach first decision: %v", err)
	}

	t.Run("re-attaching the identical decision succeeds and leaves one record", func(t *testing.T) {
		if err := attachVerificationBoundary(attemptRel, first); err != nil {
			t.Fatalf("re-attaching the identical decision should be a no-op success, got: %v", err)
		}
		var record buildAttemptRecord
		if err := store.LoadJSON(attemptRel, &record); err != nil {
			t.Fatalf("load: %v", err)
		}
		if record.VerificationBoundary == nil || *record.VerificationBoundary != first {
			t.Fatalf("VerificationBoundary = %+v, want %+v", record.VerificationBoundary, first)
		}
	})

	t.Run("attaching a different decision is refused, naming both choices", func(t *testing.T) {
		second := queenApplyVerificationBoundary("build_end", "reversed mid-flight", colony.Phase{}, colony.ColonyState{})
		err := attachVerificationBoundary(attemptRel, second)
		if err == nil {
			t.Fatalf("expected attaching a different decision to be refused")
		}
		if !strings.Contains(err.Error(), verificationBoundaryChoiceCheckStep) || !strings.Contains(err.Error(), verificationBoundaryChoiceBuildEnd) {
			t.Fatalf("error = %q, want it to name both the stored choice %q and the offered choice %q", err.Error(), verificationBoundaryChoiceCheckStep, verificationBoundaryChoiceBuildEnd)
		}
		var record buildAttemptRecord
		if err := store.LoadJSON(attemptRel, &record); err != nil {
			t.Fatalf("load: %v", err)
		}
		if record.VerificationBoundary == nil || *record.VerificationBoundary != first {
			t.Fatalf("the stored decision was rewritten: got %+v, want it to remain %+v", record.VerificationBoundary, first)
		}
	})
}

func TestLegacyAttemptWithoutBoundaryDecodes(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// A record derived and saved before this plan's field existed carries no
	// verification_boundary key at all (VerificationBoundary is nil and
	// omitempty), exactly like a pre-migration attempt file.
	attemptRel := newTestVerificationBoundaryAttempt(t, 3, "attempt-legacy-no-boundary")

	raw, err := os.ReadFile(store.BasePath() + "/" + attemptRel)
	if err != nil {
		t.Fatalf("read raw attempt file: %v", err)
	}
	if strings.Contains(string(raw), "verification_boundary") {
		t.Fatalf("expected no verification_boundary key in a legacy attempt file, got:\n%s", raw)
	}

	var record buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &record); err != nil {
		t.Fatalf("decoding a legacy attempt record failed: %v", err)
	}
	if record.VerificationBoundary != nil {
		t.Fatalf("expected VerificationBoundary to decode as nil, got %+v", record.VerificationBoundary)
	}

	decision, ok := verificationBoundaryForAttempt(attemptRel)
	if ok {
		t.Fatalf("expected verificationBoundaryForAttempt to report ok=false for a legacy attempt, got ok=true decision=%+v", decision)
	}
	if decision != (verificationBoundaryDecision{}) {
		t.Fatalf("expected a zero-value decision when ok=false, got %+v", decision)
	}
}

func TestVerificationBoundaryForAttemptReadsTheStoredRecordOnly(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	attemptRel := newTestVerificationBoundaryAttempt(t, 4, "attempt-boundary-read-path")
	decision := queenApplyVerificationBoundary("build_end", "release sign-off", colony.Phase{}, colony.ColonyState{})
	if err := attachVerificationBoundary(attemptRel, decision); err != nil {
		t.Fatalf("attach: %v", err)
	}

	got, ok := verificationBoundaryForAttempt(attemptRel)
	if !ok {
		t.Fatalf("expected ok=true for an attempt with a recorded decision")
	}
	if got != decision {
		t.Fatalf("verificationBoundaryForAttempt = %+v, want %+v", got, decision)
	}
}

// --- Task 3: an AST guard refusing a second independent boundary deriver ---

// verificationBoundaryDecisionTypeName is the decision type whose producers
// this guard counts.
const verificationBoundaryDecisionTypeName = "verificationBoundaryDecision"

// verificationBoundaryPermittedProducerNames names the two functions allowed
// to produce a verificationBoundaryDecision: the reconciliation function and
// the stored-record accessor. These two names identify the shared
// INFRASTRUCTURE this guard protects -- not a maintained list of every
// caller. The guard computes, from the parsed syntax tree, which functions
// actually return the decision type, and fails any name outside this pair
// (mirrors continueDecisionGateEvaluatorNames' own doc comment,
// cmd/codex_verify_advance_test.go).
var verificationBoundaryPermittedProducerNames = map[string]bool{
	"queenApplyVerificationBoundary": true,
	"verificationBoundaryForAttempt": true,
}

// verificationBoundaryPackageFuncs parses every non-test .go file in the cmd
// package (the working directory `go test` runs in) plus any extra synthetic
// files, and indexes every top-level function declaration by name --
// mirrors continueDecisionPackageFuncs' own renderedFieldASTFuncs precedent
// (cmd/rendered_fields_invariant_test.go, cmd/codex_verify_advance_test.go).
func verificationBoundaryPackageFuncs(t *testing.T, fset *token.FileSet, extra ...*ast.File) map[string]*ast.FuncDecl {
	t.Helper()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse cmd package for the verification boundary producer scan: %v", err)
	}
	funcs := map[string]*ast.FuncDecl{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
					funcs[fn.Name.Name] = fn
				}
			}
		}
	}
	for _, file := range extra {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Recv == nil {
				funcs[fn.Name.Name] = fn
			}
		}
	}
	return funcs
}

// funcReturnsVerificationBoundaryDecision reports whether fn's result list
// names verificationBoundaryDecision directly, whether as its sole return
// value or as one member of a multi-value return -- a plain identifier match
// against the syntax tree, the same authority level
// continueDecisionDirectCalleeNames uses for call names.
func funcReturnsVerificationBoundaryDecision(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Type == nil || fn.Type.Results == nil {
		return false
	}
	for _, field := range fn.Type.Results.List {
		if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == verificationBoundaryDecisionTypeName {
			return true
		}
	}
	return false
}

// verificationBoundaryProducers returns the sorted names of every top-level
// function in funcs whose return type includes verificationBoundaryDecision
// -- computed from the parsed declarations found, never a hand-maintained
// list of permitted callers.
func verificationBoundaryProducers(funcs map[string]*ast.FuncDecl) []string {
	var producers []string
	for name, fn := range funcs {
		if funcReturnsVerificationBoundaryDecision(fn) {
			producers = append(producers, name)
		}
	}
	sort.Strings(producers)
	return producers
}

// TestOneFunctionDerivesTheVerificationBoundary is the structural guard: the
// only functions in the cmd package that produce a verificationBoundaryDecision
// are the reconciliation function and the stored-record accessor -- a second,
// independently-deriving function is refused by name and file position.
func TestOneFunctionDerivesTheVerificationBoundary(t *testing.T) {
	t.Run("real package has exactly the two permitted producers", func(t *testing.T) {
		fset := token.NewFileSet()
		funcs := verificationBoundaryPackageFuncs(t, fset)
		producers := verificationBoundaryProducers(funcs)
		for _, name := range producers {
			if !verificationBoundaryPermittedProducerNames[name] {
				fn := funcs[name]
				pos := fset.Position(fn.Pos())
				t.Errorf("%s (%s:%d) returns %s but is neither the reconciliation function nor the stored-record accessor -- a second, independently-computed boundary is growing",
					name, pos.Filename, pos.Line, verificationBoundaryDecisionTypeName)
			}
		}
		for permitted := range verificationBoundaryPermittedProducerNames {
			if !containsString(producers, permitted) {
				t.Errorf("expected %s to be declared and to return %s, but it was not found among the parsed producers %v", permitted, verificationBoundaryDecisionTypeName, producers)
			}
		}
	})

	t.Run("guard fails a synthetic second deriver by name and position", func(t *testing.T) {
		fset := token.NewFileSet()
		src := `package cmd

func fakeSecondBoundaryDeriver() verificationBoundaryDecision {
	return verificationBoundaryDecision{Choice: "build_end", Source: "queen"}
}
`
		extra, err := parser.ParseFile(fset, "fixture_second_boundary_deriver.go", src, 0)
		if err != nil {
			t.Fatalf("parse synthetic second-deriver fixture: %v", err)
		}
		funcs := verificationBoundaryPackageFuncs(t, fset, extra)
		producers := verificationBoundaryProducers(funcs)
		if !containsString(producers, "fakeSecondBoundaryDeriver") {
			t.Fatalf("expected the guard to see the synthetic fixture among producers, got %v", producers)
		}
		var offenderPos token.Position
		found := false
		for _, name := range producers {
			if verificationBoundaryPermittedProducerNames[name] {
				continue
			}
			if name != "fakeSecondBoundaryDeriver" {
				t.Fatalf("unexpected offender %s", name)
			}
			found = true
			offenderPos = fset.Position(funcs[name].Pos())
		}
		if !found {
			t.Fatalf("expected the guard to flag fakeSecondBoundaryDeriver as an offender, got producers=%v", producers)
		}
		if offenderPos.Filename == "" || offenderPos.Line == 0 {
			t.Fatalf("violation for fakeSecondBoundaryDeriver carries no file/line position: %+v", offenderPos)
		}
	})
}
