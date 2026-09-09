package cmd

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// testBuildStartOptions keeps every nondeterministic input and every
// conditional transaction effect visible at the fixture boundary. Tests may
// omit values that are immaterial to their assertion; defaults are derived
// from the selected closed build-start variant, never from package globals.
type testBuildProcessState string

const (
	testBuildProcessUnspecified testBuildProcessState = ""
	testBuildProcessLive        testBuildProcessState = "live"
	testBuildProcessDead        testBuildProcessState = "dead"
)

type testBuildStartOptions struct {
	Variant         buildStartVariant
	Authority       *planAuthorityDecision
	Phase           int
	GeneratedAt     time.Time
	AttemptID       string
	RunID           string
	ProcessID       int
	ProcessState    testBuildProcessState
	HostPlatform    string
	SelectedTasks   []string
	Dispatches      []codexBuildDispatch
	ExecutionOwner  string
	DispatchMode    string
	MakeLatest      *bool
	CheckpointPath  string
	ManifestPath    string
	Manifest        *codexBuildManifest
	Completion      *codexExternalBuildCompletion
	ClaimsPath      string
	Claims          *codexBuildClaims
	PromoteState    *bool
	ParentAttemptID string
	ParentJobName   string
	CheckFix        *checkFixAttemptRecord
	ReviewerWindow  buildStartReviewerWindowEffect
	StalePaths      []string
	PrepareRoot     func(root string)
}

// testBuildStartFixture exposes only data read back after commitBuildStart.
// It deliberately has no alternate attempt/pointer/state writer.
type testBuildStartFixture struct {
	Root        string
	DataRoot    string
	Request     buildStartRequest
	Receipt     buildStartReceipt
	AttemptPath string
	Attempt     buildAttemptRecord
	Latest      *latestBuildAttemptPointer
	State       colony.ColonyState
	Phase       colony.Phase
	Manifest    *codexBuildManifest
}

func testBuildStartBool(value bool) *bool {
	return &value
}

// commitTestBuildStart creates a real accepted planning authority, binds the
// package store to that physically contained repository, and invokes the same
// receipt-last transaction used by production. Prerequisite specification,
// candidate, timeline, and acceptance artifacts come from the canonical Plan
// 29/30 fixture builders; no build-start target is written directly here.
func commitTestBuildStart(t *testing.T, options testBuildStartOptions) testBuildStartFixture {
	t.Helper()
	saveGlobals(t)
	root, seed, _ := buildStartTransaction200CurrentAuthorityFixture(t)
	dataRoot := filepath.Join(root, ".aether", "data")
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", dataRoot)
	repositoryStore, err := storage.NewStore(dataRoot)
	if err != nil {
		t.Fatalf("open canonical build-start fixture store: %v", err)
	}
	store = repositoryStore

	contained, err := filepath.Rel(filepath.Clean(root), filepath.Clean(store.BasePath()))
	if err != nil || contained != filepath.Join(".aether", "data") {
		t.Fatalf("build-start fixture store %q is not physically contained by repository %q", store.BasePath(), root)
	}
	if options.PrepareRoot != nil {
		options.PrepareRoot(root)
	}
	testBuildStartEnsureGoal(t, root)

	state := mustReadSpecificationTestState(t, root)
	phaseID := seed.Phase
	if err := validateTestBuildStartOptions(options, phaseID); err != nil {
		t.Fatalf("canonical build-start fixture options: %v", err)
	}
	phase, ok := buildStartPhase(state, phaseID)
	if !ok {
		t.Fatalf("accepted build-start fixture has no phase %d", phaseID)
	}
	generatedAt := options.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = seed.GeneratedAt
	}
	generatedAt = generatedAt.UTC()
	processID := testBuildStartProcessID(t, options)
	attemptID := options.AttemptID
	if attemptID == "" {
		attemptID = deriveBuildAttemptID(generatedAt, processID)
	}
	runID := options.RunID
	if runID == "" {
		runID = fmt.Sprintf("run-%032d", processID)
	}
	hostPlatform := options.HostPlatform
	if hostPlatform == "" {
		hostPlatform = "test"
	}
	variant := options.Variant
	if variant == "" {
		variant = buildStartDirect
	}
	rule, ok := buildStartRuleFor(variant)
	if !ok {
		t.Fatalf("unsupported fixture build-start variant %q", variant)
	}
	executionOwner, dispatchMode := testBuildStartOwnerAndMode(variant)
	if options.ExecutionOwner != "" {
		executionOwner = options.ExecutionOwner
	}
	if options.DispatchMode != "" {
		dispatchMode = options.DispatchMode
	}

	facts := lifecycleFactsFromStateSnapshot(state, false, generatedAt)
	facts.Root = root
	authority, err := preflightCodexBuildPlanAuthority(facts, loadPlanAuthorityVerifiedBindings(root, facts))
	if err != nil {
		t.Fatalf("derive canonical test build authority: %v", err)
	}
	if options.Authority != nil {
		authority = *options.Authority
	}
	stateSHA, err := jsonSHA256(state)
	if err != nil {
		t.Fatalf("hash canonical fixture state: %v", err)
	}
	workspaceSHA, err := codex.WorkspaceFingerprint(root)
	if err != nil {
		t.Fatalf("fingerprint canonical fixture workspace: %v", err)
	}

	selectedTasks := append([]string{}, options.SelectedTasks...)
	request := buildStartRequest{
		SchemaVersion:   buildStartSchemaVersion,
		Variant:         variant,
		StateSHA256:     stateSHA,
		PlanAuthority:   authority,
		Phase:           phaseID,
		SelectedTasks:   selectedTasks,
		ExecutionOwner:  executionOwner,
		DispatchMode:    dispatchMode,
		GeneratedAt:     generatedAt,
		AttemptID:       attemptID,
		RunID:           runID,
		ProcessID:       processID,
		HostPlatform:    hostPlatform,
		WorkspaceSHA256: workspaceSHA,
		Dispatches:      append([]codexBuildDispatch(nil), options.Dispatches...),
	}
	request.Effects = testBuildStartEffects(t, root, state, phase, request, rule, options)

	receipt, err := commitBuildStart(root, request, buildStartOptions{})
	if err != nil {
		t.Fatalf("commit canonical test build start: %v", err)
	}
	attemptPath := buildStartAttemptPath(request.Phase, request.AttemptID)
	var attempt buildAttemptRecord
	testBuildStartReadJSON(t, filepath.Join(dataRoot, filepath.FromSlash(attemptPath)), &attempt)
	committedState := mustReadSpecificationTestState(t, root)
	committedPhase, ok := buildStartPhase(committedState, request.Phase)
	if !ok {
		t.Fatalf("committed build-start fixture lost phase %d", request.Phase)
	}

	fixture := testBuildStartFixture{
		Root: root, DataRoot: dataRoot, Request: request, Receipt: receipt,
		AttemptPath: attemptPath, Attempt: attempt, State: committedState, Phase: committedPhase,
	}
	if request.Effects.MakeLatest {
		var latest latestBuildAttemptPointer
		testBuildStartReadJSON(t, filepath.Join(dataRoot, filepath.FromSlash(latestBuildAttemptPointerPath(request.Phase))), &latest)
		fixture.Latest = &latest
	}
	if request.Effects.Manifest != nil {
		var manifest codexBuildManifest
		testBuildStartReadJSON(t, filepath.Join(dataRoot, filepath.FromSlash(request.Effects.ManifestPath)), &manifest)
		fixture.Manifest = &manifest
	}
	return fixture
}

func validateTestBuildStartOptions(options testBuildStartOptions, acceptedPhase int) error {
	if options.Phase != 0 && options.Phase != acceptedPhase {
		return fmt.Errorf("requested phase %d does not match accepted phase %d", options.Phase, acceptedPhase)
	}
	if options.Phase != 0 && ((strings.TrimSpace(options.ExecutionOwner) == "") != (strings.TrimSpace(options.DispatchMode) == "")) {
		return fmt.Errorf("explicit phase %d requires execution owner and dispatch mode together", options.Phase)
	}
	if options.ProcessID != 0 && options.ProcessState != testBuildProcessUnspecified {
		return fmt.Errorf("process_id and process_state are mutually exclusive; choose an explicit identity or liveness fixture")
	}
	switch options.ProcessState {
	case testBuildProcessUnspecified, testBuildProcessLive, testBuildProcessDead:
		return nil
	default:
		return fmt.Errorf("unsupported fixture process state %q", options.ProcessState)
	}
}

func testBuildStartProcessID(t *testing.T, options testBuildStartOptions) int {
	t.Helper()
	if options.ProcessID != 0 {
		return options.ProcessID
	}
	switch options.ProcessState {
	case testBuildProcessLive:
		return os.Getpid()
	case testBuildProcessDead:
		const deadProcessID = 3500
		if processAlive(deadProcessID) {
			t.Fatalf("fixed dead process fixture PID %d is unexpectedly live; supply an explicit dead ProcessID", deadProcessID)
		}
		return deadProcessID
	default:
		// Compatibility callers created before Plan 49 remain deterministic.
		// New accepted-execution builders below require ProcessState explicitly.
		return 3500
	}
}

// testBuildStartEnsureGoal makes the accepted planning fixture a complete
// active colony through the same repository-session writer used by planning.
// The canonical candidate fixture intentionally starts from a zero-value
// colony state; public build callers require a goal before they can reach the
// provider-preflight behavior these fixtures exercise.
func testBuildStartEnsureGoal(t *testing.T, root string) {
	t.Helper()
	if err := withPlanningMutationSession(root, "test-build-start-goal", func(session *planningMutationSession) error {
		var state colony.ColonyState
		exists, err := session.LoadJSON(lifecycleTransactionRootData, "COLONY_STATE.json", &state)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("canonical build-start fixture state is missing")
		}
		if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
			return nil
		}
		goal := "Exercise the canonical build-start transaction"
		state.Goal = &goal
		return persistPlanningColonyStateInSession(session, "test-build-start-goal", state)
	}); err != nil {
		t.Fatalf("seed canonical build-start goal: %v", err)
	}
}

func testBuildStartOwnerAndMode(variant buildStartVariant) (string, string) {
	switch variant {
	case buildStartPlanOnly:
		return "host-queen", "plan-only"
	case buildStartQueenLed:
		return "host-queen", "queen-led"
	case buildStartExternalUnbound:
		return "external-task", "external-task"
	case buildStartAutomaticCheckFix:
		return "check-fix-attempt", "check-fix-attempt"
	case buildStartCoherentChildRetry:
		return "runtime-worker-dispatch", "coherent-child-retry"
	default:
		return "runtime-worker-dispatch", "direct"
	}
}

func testBuildStartEffects(t *testing.T, root string, state colony.ColonyState, phase colony.Phase, request buildStartRequest, rule buildStartRule, options testBuildStartOptions) buildStartEffects {
	t.Helper()
	effects := buildStartEffects{
		MakeLatest: rule.latest, PromoteState: rule.promote, ReviewerWindow: rule.reviewer,
		ParentAttemptID: options.ParentAttemptID, ParentJobName: options.ParentJobName,
		CheckFix: options.CheckFix, StalePaths: append([]string(nil), options.StalePaths...),
	}
	if options.MakeLatest != nil {
		effects.MakeLatest = *options.MakeLatest
	}
	if options.PromoteState != nil {
		effects.PromoteState = *options.PromoteState
	}
	if options.ReviewerWindow != buildStartReviewerNone {
		effects.ReviewerWindow = options.ReviewerWindow
	}
	if rule.checkpoint {
		effects.CheckpointPath = options.CheckpointPath
		if effects.CheckpointPath == "" {
			effects.CheckpointPath = filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", request.Phase)))
		}
	}

	planSHA, err := planStateHash(state.Plan)
	if err != nil {
		t.Fatalf("hash canonical fixture plan: %v", err)
	}
	manifest := codexBuildManifest{
		Phase: request.Phase, PhaseName: phase.Name, Root: root,
		PlanOnly:     request.Variant == buildStartPlanOnly || request.Variant == buildStartQueenLed,
		DispatchMode: request.DispatchMode, ExecutionOwner: request.ExecutionOwner,
		ClaimsPath:  displayDataPath(options.ClaimsPath),
		GeneratedAt: request.GeneratedAt.Format(time.RFC3339), PlanAuthority: request.PlanAuthority,
		PlanRevisionID: request.PlanAuthority.ActiveRevision.ID, PlanStateHash: planSHA,
		State: string(state.State), Dispatches: append([]codexBuildDispatch(nil), request.Dispatches...),
		SelectedTasks: append([]string{}, request.SelectedTasks...),
	}
	if options.Manifest != nil {
		manifest = *options.Manifest
	}
	if rule.manifest {
		effects.ManifestPath = options.ManifestPath
		if effects.ManifestPath == "" {
			effects.ManifestPath = filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", request.Phase), "manifest.json"))
		}
		effects.Manifest = &manifest
	}
	if rule.completion {
		effects.Completion = options.Completion
		if effects.Completion == nil {
			effects.Completion = &codexExternalBuildCompletion{DispatchManifest: &manifest}
		}
	}
	if rule.claims {
		effects.ClaimsPath = options.ClaimsPath
		if effects.ClaimsPath == "" {
			effects.ClaimsPath = "last-build-claims.json"
		}
		effects.Claims = options.Claims
		if effects.Claims == nil {
			effects.Claims = &codexBuildClaims{BuildPhase: request.Phase, Timestamp: request.GeneratedAt.Format(time.RFC3339)}
		}
	}
	if rule.parent {
		if effects.ParentAttemptID == "" {
			effects.ParentAttemptID = "attempt-parent-test"
		}
		if effects.ParentJobName == "" {
			effects.ParentJobName = "test-parent-job"
		}
	}
	if rule.checkFix && effects.CheckFix == nil {
		effects.CheckFix = &checkFixAttemptRecord{
			RecordedAt: request.GeneratedAt.Format(time.RFC3339), Phase: request.Phase,
			Check: "tests", Reason: "canonical fixture check fix", ParentAttemptID: "attempt-parent-test", Outcome: "pending",
		}
	}
	return effects
}

func testBuildStartReadJSON(t *testing.T, path string, value any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read canonical build-start target %s: %v", path, err)
	}
	if err := json.Unmarshal(content, value); err != nil {
		t.Fatalf("decode canonical build-start target %s: %v", path, err)
	}
}

func TestBuildAttemptFixtureMigration200(t *testing.T) {
	owned := []string{
		"build_start_test_helpers_200_test.go",
		"build_attempt_test.go",
		"preflight_phase_198_3_test.go",
		"autopilot_checkpoints_test.go",
		"floor_fix_attempt_test.go",
		"codex_build_test.go",
	}
	forbiddenCalls := map[string]bool{
		"beginBuildAttempt":       true,
		"beginBuildAttemptRecord": true,
	}
	var violations []string
	for _, name := range owned {
		set := token.NewFileSet()
		parsed, err := parser.ParseFile(set, name, nil, 0)
		if err != nil {
			t.Fatalf("parse owned build-start fixture file %s: %v", name, err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			identifier, ok := call.Fun.(*ast.Ident)
			if ok && forbiddenCalls[identifier.Name] {
				violations = append(violations, fmt.Sprintf("%s:%d calls legacy %s", name, set.Position(call.Pos()).Line, identifier.Name))
			}
			return true
		})
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			hasAttemptRecord := false
			hasLatestPointer := false
			ast.Inspect(function.Body, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				identifier, ok := literal.Type.(*ast.Ident)
				if !ok {
					return true
				}
				hasAttemptRecord = hasAttemptRecord || identifier.Name == "buildAttemptRecord"
				hasLatestPointer = hasLatestPointer || identifier.Name == "latestBuildAttemptPointer"
				return true
			})
			if hasAttemptRecord && hasLatestPointer {
				violations = append(violations, fmt.Sprintf("%s:%d function %s reconstructs an attempt plus latest pointer", name, set.Position(function.Pos()).Line, function.Name.Name))
			}
		}
	}
	if len(violations) > 0 {
		t.Fatalf("owned build-start fixtures bypass the canonical transaction:\n%s", strings.Join(violations, "\n"))
	}
}

func TestBuildAttemptFixtureUsesCanonicalTransaction(t *testing.T) {
	generatedAt := time.Date(2026, time.September, 9, 12, 35, 0, 0, time.UTC)
	processID := 3501
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt:    generatedAt,
		AttemptID:      deriveBuildAttemptID(generatedAt, processID),
		RunID:          "run-35353535353535353535353535353501",
		ProcessID:      processID,
		ExecutionOwner: "fixture-owner",
		DispatchMode:   "fixture-direct",
		MakeLatest:     testBuildStartBool(true),
	})

	if fixture.Receipt.Path != buildStartReceiptPath(fixture.Request.Phase, fixture.Request.AttemptID) {
		t.Fatalf("fixture receipt path = %q, want canonical build-start receipt", fixture.Receipt.Path)
	}
	if fixture.Attempt.ID != fixture.Request.AttemptID || fixture.Attempt.RunID != fixture.Request.RunID {
		t.Fatalf("fixture attempt lost explicit identity: request=%+v attempt=%+v", fixture.Request, fixture.Attempt)
	}
	wantDataRoot := filepath.Join(fixture.Root, ".aether", "data")
	if got := filepath.Clean(store.BasePath()); got != filepath.Clean(wantDataRoot) {
		t.Fatalf("fixture store root = %q, want physically contained %q", got, wantDataRoot)
	}
	if fixture.Latest == nil || fixture.Latest.AttemptID != fixture.Attempt.ID {
		t.Fatalf("fixture latest pointer = %+v, want attempt %q", fixture.Latest, fixture.Attempt.ID)
	}
}

func TestCanonicalBuildStartFixtureAuthority200(t *testing.T) {
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Phase:          1,
		Variant:        buildStartPlanOnly,
		ExecutionOwner: "host-queen",
		DispatchMode:   "plan-only",
		ProcessState:   testBuildProcessDead,
	})

	if fixture.Request.Phase != 1 || fixture.Attempt.Phase != 1 {
		t.Fatalf("canonical fixture phase = request %d / attempt %d, want exact accepted phase 1", fixture.Request.Phase, fixture.Attempt.Phase)
	}
	if !fixture.Request.PlanAuthority.Eligible || fixture.Request.PlanAuthority.Classification != planAuthorityCurrentAccepted {
		t.Fatalf("canonical fixture did not earn current accepted authority: %+v", fixture.Request.PlanAuthority)
	}
	if fixture.Request.ExecutionOwner != "host-queen" || fixture.Request.DispatchMode != "plan-only" {
		t.Fatalf("canonical fixture execution identity = %q/%q, want host-queen/plan-only", fixture.Request.ExecutionOwner, fixture.Request.DispatchMode)
	}
	if fixture.Receipt.PlanAuthority.ActiveRevision.ID != fixture.Request.PlanAuthority.ActiveRevision.ID ||
		fixture.Receipt.PlanAuthority.Specification.ID != fixture.Request.PlanAuthority.Specification.ID {
		t.Fatalf("durable receipt lost exact plan/specification authority: receipt=%+v request=%+v", fixture.Receipt.PlanAuthority, fixture.Request.PlanAuthority)
	}
}

func TestCanonicalBuildStartFixtureLiveProcess200(t *testing.T) {
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		Phase:          1,
		ExecutionOwner: "runtime-worker-dispatch",
		DispatchMode:   "direct",
		ProcessState:   testBuildProcessLive,
	})

	if fixture.Request.ProcessID != os.Getpid() || fixture.Attempt.ProcessID != os.Getpid() {
		t.Fatalf("live fixture process = request %d / attempt %d, want current process %d", fixture.Request.ProcessID, fixture.Attempt.ProcessID, os.Getpid())
	}
	if !buildAttemptProcessAlive(fixture.Attempt) {
		t.Fatalf("explicit live fixture was not live: %+v", fixture.Attempt)
	}
}

func TestCanonicalBuildStartFixtureRefusesMismatch200(t *testing.T) {
	err := validateTestBuildStartOptions(testBuildStartOptions{
		Phase:          2,
		ExecutionOwner: "host-queen",
		DispatchMode:   "plan-only",
		ProcessState:   testBuildProcessDead,
	}, 1)
	if err == nil || !strings.Contains(err.Error(), "requested phase 2") || !strings.Contains(err.Error(), "accepted phase 1") {
		t.Fatalf("mismatched accepted phase error = %v, want exact requested/accepted phase refusal", err)
	}

	err = validateTestBuildStartOptions(testBuildStartOptions{Phase: 1, ExecutionOwner: "host-queen"}, 1)
	if err == nil || !strings.Contains(err.Error(), "execution owner and dispatch mode") {
		t.Fatalf("half-specified execution identity error = %v, want explicit owner/mode refusal", err)
	}
}
