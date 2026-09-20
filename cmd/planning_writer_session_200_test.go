package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const planningWriterHelperEnvironment200 = "AETHER_PLANNING_WRITER_200_HELPER"

type planningWriterASTFunction200 struct {
	decl  *ast.FuncDecl
	calls map[string]int
}

// TestPlanningWriterCoverage200 is a source-level ratchet over real Go call
// expressions. Comments and string literals cannot satisfy it. Each row names
// one authoritative branch whose first read through final mutation must be
// dominated by Plan 28's repository session.
func TestPlanningWriterCoverage200(t *testing.T) {
	functions := planningWriterParseFunctions200(t,
		"codex_plan.go",
		"codex_plan_finalize.go",
		"specification.go",
		"plan_revision.go",
		"state_extra.go",
	)
	required := []struct {
		branch   string
		function string
		call     string
	}{
		{"plan/research-docs", "runCodexPlanWithOptions", "withPlanningMutationSession"},
		{"plan/verification-depth-existing-return", "runCodexPlanWithOptions", "withPlanningMutationSession"},
		{"plan/verification-depth-new-or-refresh", "runCodexPlanWithOptions", "withPlanningMutationSession"},
		{"plan/existing-plan-return", "runCodexPlanWithOptions", "withPlanningMutationSession"},
		{"plan/scout-run-header-and-stage", "persistPlanningScoutStage", "withPlanningMutationSession"},
		{"plan/refresh-cleanup", "clearFallbackPlanningArtifacts", "DeclareRemoval"},
		{"plan/immediate-finalizer", "runCodexPlanFinalize", "withPlanningMutationSession"},
		{"plan/staged-scout-finalizer", "runCodexScoutStageFinalize", "withPlanningMutationSession"},
		{"plan/staged-route-finalizer", "runCodexRouteStageFinalize", "withPlanningMutationSession"},
		{"plan/intermediate-iteration", "persistIntermediatePlanningIteration", "withPlanningMutationSession"},
		{"plan/finalizer-failure-record", "recordPlanFinalizeFailure", "withPlanningMutationSession"},
		{"specification/draft", "createSpecificationDraft", "withPlanningMutationSession"},
		{"specification/revise", "reviseSpecification", "withPlanningMutationSession"},
		{"specification/approve", "approveSpecification", "withPlanningMutationSession"},
		{"candidate/accept", "acceptPlanCandidate", "withPlanningMutationSession"},
		{"candidate/insert-phase", "createPhaseInsertCandidate", "withPlanningMutationSession"},
		{"candidate/route-publication", "persistPlanningRouteCandidate", "withPlanningMutationSession"},
	}
	for _, item := range required {
		item := item
		t.Run(item.branch, func(t *testing.T) {
			_, ok := functions[item.function]
			if !ok {
				t.Fatalf("inventory function %s is missing", item.function)
			}
			if !planningWriterCallsTransitively200(functions, item.function, item.call, nil) {
				t.Errorf("%s does not contain an AST call to %s", item.function, item.call)
			}
		})
	}

	for _, functionName := range []string{
		"runCodexPlanWithOptions", "runCodexPlanWithOptionsInSession",
		"persistPlanningScoutStage", "persistPlanningScoutStageInSession",
		"persistIntermediatePlanningIteration", "persistIntermediatePlanningIterationInSession",
		"runCodexPlanFinalize", "runCodexPlanFinalizeInSession", "recordPlanFinalizeFailure",
		"runCodexScoutStageFinalize", "runCodexScoutStageFinalizeInSession",
		"runCodexRouteStageFinalize", "runCodexRouteStageFinalizeInSession",
		"createSpecificationDraft", "createSpecificationDraftInSession",
		"reviseSpecification", "reviseSpecificationInSession",
		"approveSpecification", "approveSpecificationInSession",
		"createPhaseInsertCandidate", "createPhaseInsertCandidateInSession",
		"runPhaseInsertCommand", "runPhaseInsertCommandInSession",
	} {
		function := functions[functionName]
		for _, forbidden := range []string{"SaveJSON", "UpdateJSONAtomically", "os.Remove", "os.RemoveAll"} {
			if count := function.calls[forbidden]; count != 0 {
				t.Errorf("%s contains %d direct %s AST call(s); declare exact session transaction targets instead", functionName, count, forbidden)
			}
		}
	}

	if !planningWriterPhaseInsertHandlerUsesSession200(t, functions) {
		t.Error("phaseInsertCmd RunE does not delegate to a named session-bound handler")
	}
}

func planningWriterCallsTransitively200(functions map[string]planningWriterASTFunction200, functionName, wanted string, visiting map[string]bool) bool {
	if visiting == nil {
		visiting = make(map[string]bool)
	}
	if visiting[functionName] {
		return false
	}
	visiting[functionName] = true
	defer delete(visiting, functionName)
	function, ok := functions[functionName]
	if !ok {
		return false
	}
	if function.calls[wanted] > 0 {
		return true
	}
	for called := range function.calls {
		if _, owned := functions[called]; owned && planningWriterCallsTransitively200(functions, called, wanted, visiting) {
			return true
		}
	}
	return false
}

func planningWriterParseFunctions200(t *testing.T, names ...string) map[string]planningWriterASTFunction200 {
	t.Helper()
	result := make(map[string]planningWriterASTFunction200)
	for _, name := range names {
		path := filepath.Join(".", name)
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				continue
			}
			result[function.Name.Name] = planningWriterASTFunction200{decl: function, calls: planningWriterCallInventory200(function.Body)}
		}
	}
	return result
}

func planningWriterCallInventory200(node ast.Node) map[string]int {
	calls := make(map[string]int)
	ast.Inspect(node, func(candidate ast.Node) bool {
		call, ok := candidate.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			calls[function.Name]++
		case *ast.SelectorExpr:
			calls[function.Sel.Name]++
			if receiver, ok := function.X.(*ast.Ident); ok {
				calls[receiver.Name+"."+function.Sel.Name]++
			}
		}
		return true
	})
	return calls
}

func planningWriterPhaseInsertHandlerUsesSession200(t *testing.T, functions map[string]planningWriterASTFunction200) bool {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(".", "state_extra.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range parsed.Decls {
		value, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, rawSpec := range value.Specs {
			spec, ok := rawSpec.(*ast.ValueSpec)
			if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "phaseInsertCmd" || len(spec.Values) != 1 {
				continue
			}
			composite, ok := spec.Values[0].(*ast.UnaryExpr)
			if !ok {
				return false
			}
			literal, ok := composite.X.(*ast.CompositeLit)
			if !ok {
				return false
			}
			for _, element := range literal.Elts {
				pair, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := pair.Key.(*ast.Ident)
				if !ok || key.Name != "RunE" {
					continue
				}
				calls := map[string]int{}
				switch value := pair.Value.(type) {
				case *ast.FuncLit:
					calls = planningWriterCallInventory200(value.Body)
				case *ast.Ident:
					calls[value.Name]++
				}
				for called := range calls {
					if function, exists := functions[called]; exists && function.calls["withPlanningMutationSession"] > 0 {
						return true
					}
				}
			}
		}
	}
	return false
}

// TestPlanningWriterConcurrentProcesses200 uses real child processes and
// explicit barriers. It covers the plan branch, run-header/stage publication,
// intermediate iteration state, specification stale derivation, and public
// insert candidate families.
func TestPlanningWriterConcurrentProcesses200(t *testing.T) {
	if os.Getenv(planningWriterHelperEnvironment200) == "1" {
		planningWriterProcessHelper200()
		return
	}

	t.Run("existing-plan verification depth waits for repository session", func(t *testing.T) {
		goal := "session-bound existing plan"
		state := specificationTestPlanState()
		state.Goal = &goal
		state.State = colony.StateREADY
		state.Plan.AcceptancePolicy = colony.PlanAcceptanceLegacyUnbound
		state.Plan.Phases[0].Status = colony.PhaseReady
		root := newSpecificationTestRepository(t, state)
		planningWriterAssertBlockedBySession200(t, root, "plan-existing", nil)
	})

	t.Run("intermediate iteration waits for repository session", func(t *testing.T) {
		root := newSpecificationTestRepository(t, colony.ColonyState{})
		planningWriterAssertBlockedBySession200(t, root, "intermediate", nil)
	})

	t.Run("run header cannot appear before scout stage transaction", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".aether", "data"), 0o755); err != nil {
			t.Fatal(err)
		}
		running, manifest := planningStageReceiptTestScoutDispatch(t, root)
		header := planningScoutStageTestHeader(t, manifest)
		payload, err := json.Marshal(struct {
			Header   planningRunHeader     `json:"header"`
			Running  planningStageState    `json:"running"`
			Manifest planningStageManifest `json:"manifest"`
		}{header, running, manifest})
		if err != nil {
			t.Fatal(err)
		}
		child, ready := planningWriterStartProcess200(t, root, "scout-stage", payload)
		headerPath := filepath.Join(root, ".aether", "data", "planning", manifest.RunID, "run-header.json")
		err = withPlanningMutationSession(root, "test-block-scout-stage", func(*planningMutationSession) error {
			planningWriterAwaitFile200(t, ready, true)
			deadline := time.Now().Add(750 * time.Millisecond)
			for time.Now().Before(deadline) {
				if _, statErr := os.Stat(headerPath); statErr == nil {
					return fmt.Errorf("run header became visible before the held stage session released")
				}
				time.Sleep(10 * time.Millisecond)
			}
			return nil
		})
		waitErr := <-child.wait
		if err != nil {
			t.Error(err)
		}
		planningWriterRequireProcessSuccess200(t, child, waitErr)
	})

	t.Run("insert candidate waits for repository session", func(t *testing.T) {
		root, state, specificationItemID := insertPhaseAcceptedPlanFixture(t)
		active, ok := activePlanRevision(state.Plan)
		if !ok {
			t.Fatal("insert fixture has no active revision")
		}
		payload, err := json.Marshal(phaseInsertCandidateRequest{
			After: 0, Name: "Stabilize session ordering", Description: "Keep phase insertion behind one repository session.",
			SpecificationItemID: specificationItemID, ExpectedBasePlanRevisionID: active.ID,
			CreatedAt: time.Date(2026, time.September, 9, 10, 0, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatal(err)
		}
		planningWriterAssertBlockedBySession200(t, root, "insert", payload)
	})

	t.Run("stale specification derivation refuses the winner", func(t *testing.T) {
		root := newSpecificationTestRepository(t, specificationTestPlanState())
		draft, err := createSpecificationDraft(root, specificationTestDraftRequest(t, colony.SpecScopeWholeGoal), specificationMutationOptions{})
		if err != nil {
			t.Fatal(err)
		}
		stale := planningWriterRevisionRequest200(draft, "stale subprocess", time.Date(2026, time.September, 9, 11, 0, 0, 0, time.UTC))
		winnerRequest := planningWriterRevisionRequest200(draft, "committed winner", time.Date(2026, time.September, 9, 11, 1, 0, 0, time.UTC))
		payload, err := json.Marshal(stale)
		if err != nil {
			t.Fatal(err)
		}
		child, ready := planningWriterStartProcess200(t, root, "spec-stale", payload)
		planningWriterAwaitFile200(t, ready, true)
		winner, err := reviseSpecification(root, winnerRequest, specificationMutationOptions{})
		if err != nil {
			t.Fatalf("commit specification winner: %v", err)
		}
		if err := os.WriteFile(child.release, []byte("release\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		waitErr := <-child.wait
		var result planningWriterProcessResult200
		planningWriterDecodeProcess200(t, child, waitErr, &result)
		if result.Error == "" {
			t.Error("stale specification subprocess committed instead of refusing an absent mutation session")
		}
		current := mustReadSpecificationTestState(t, root)
		revision, ok := currentSpecificationRevision(*current.Specification)
		if !ok || revision.ID != winner.Revision.ID || revision.ContentHash != winner.Revision.ContentHash {
			t.Fatalf("stale writer replaced winner: current=%+v winner=%+v", revision, winner.Revision)
		}
	})
}

// TestPlanningNumericBoundaries200 carries the four requirement tags in its
// executable subtest names so the final audit maps every edge truth directly.
func TestPlanningNumericBoundaries200(t *testing.T) {
	t.Run("PLAN-01/boundary-values", func(t *testing.T) {
		for _, field := range []string{"knowledge", "requirements", "risks", "dependencies", "effort"} {
			for _, value := range []int{0, 100} {
				score, err := decodePlanningWholeScore(field, json.RawMessage(fmt.Sprintf("%d", value)))
				if err != nil || score != value {
					t.Errorf("%s endpoint %d: score=%d err=%v", field, value, score, err)
				}
			}
			for _, value := range []string{"-1", "101"} {
				score, err := decodePlanningWholeScore(field, json.RawMessage(value))
				if err == nil {
					t.Errorf("%s accepted out-of-range score %s as %d", field, value, score)
				}
			}
		}
	})

	t.Run("PLAN-01/precision-overflow", func(t *testing.T) {
		for _, value := range []string{"0.5", "99.9", "1e2", "1e309", "9223372036854775808", "-9223372036854775809"} {
			score, err := decodePlanningWholeScore("knowledge", json.RawMessage(value))
			if err == nil {
				t.Errorf("accepted non-whole or overflowing confidence %s as %d", value, score)
			}
		}
	})

	t.Run("PLAN-04/boundary-values", func(t *testing.T) {
		for _, endpoint := range []string{"zero", "len"} {
			root, state, specificationItemID := insertPhaseAcceptedPlanFixture(t)
			active, _ := activePlanRevision(state.Plan)
			after := 0
			if endpoint == "len" {
				after = len(state.Plan.Phases)
			}
			if _, err := createPhaseInsertCandidate(root, phaseInsertCandidateRequest{
				After: after, Name: "Boundary " + endpoint, Description: "Exercise exact insertion endpoint.",
				SpecificationItemID: specificationItemID, ExpectedBasePlanRevisionID: active.ID,
				CreatedAt: time.Date(2026, time.September, 9, 12, after, 0, 0, time.UTC),
			}); err != nil {
				t.Errorf("position %s (%d) rejected: %v", endpoint, after, err)
			}
		}
		for _, after := range []int{-1, 3} {
			root, state, specificationItemID := insertPhaseAcceptedPlanFixture(t)
			active, _ := activePlanRevision(state.Plan)
			before := planningWriterArtifactInventory200(t, root)
			_, err := createPhaseInsertCandidate(root, phaseInsertCandidateRequest{
				After: after, Name: "Invalid boundary", Description: "Must not mutate.",
				SpecificationItemID: specificationItemID, ExpectedBasePlanRevisionID: active.ID,
				CreatedAt: time.Date(2026, time.September, 9, 12, 30, 0, 0, time.UTC),
			})
			if err == nil {
				t.Errorf("position %d accepted for plan length %d", after, len(state.Plan.Phases))
			}
			planningWriterAssertInventory200(t, root, before)
		}
	})

	t.Run("PLAN-04/precision-overflow", func(t *testing.T) {
		for _, raw := range []string{"0.5", "1e2", "9223372036854775808"} {
			root, _, _ := insertPhaseAcceptedPlanFixture(t)
			before := planningWriterArtifactInventory200(t, root)
			resetRootCmd(t)
			rootCmd.SetArgs([]string{"insert-phase", "invalid numeric position", "--after", raw})
			if err := rootCmd.Execute(); err == nil {
				t.Errorf("public insert accepted position %q", raw)
			}
			planningWriterAssertInventory200(t, root, before)
		}
		for _, identity := range []string{"1.5", strings.Repeat("f", 11), strings.Repeat("f", 128)} {
			root, state, specificationItemID := insertPhaseAcceptedPlanFixture(t)
			before := planningWriterArtifactInventory200(t, root)
			_, err := createPhaseInsertCandidate(root, phaseInsertCandidateRequest{
				After: 0, Name: "Invalid identity", Description: "Must refuse non-canonical identity.",
				SpecificationItemID: specificationItemID, ExpectedBasePlanRevisionID: identity,
				CreatedAt: time.Date(2026, time.September, 9, 12, 45, 0, 0, time.UTC),
			})
			if err == nil {
				t.Errorf("insert accepted malformed or prefix identity %q (current %q)", identity, state.Plan.ActiveRevisionID)
			}
			planningWriterAssertInventory200(t, root, before)
		}
	})
}

type planningWriterProcess200 struct {
	command *exec.Cmd
	output  *bytes.Buffer
	ready   string
	release string
	wait    <-chan error
}

type planningWriterProcessResult200 struct {
	Error string `json:"error,omitempty"`
	OK    bool   `json:"ok"`
}

func planningWriterStartProcess200(t *testing.T, root, mode string, payload []byte) (*planningWriterProcess200, string) {
	t.Helper()
	barriers := t.TempDir()
	ready := filepath.Join(barriers, "ready")
	release := filepath.Join(barriers, "release")
	command := exec.Command(os.Args[0], "-test.run=^TestPlanningWriterConcurrentProcesses200$", "-test.count=1")
	command.Env = append(os.Environ(),
		planningWriterHelperEnvironment200+"=1",
		"AETHER_PLANNING_WRITER_200_ROOT="+root,
		"AETHER_PLANNING_WRITER_200_MODE="+mode,
		"AETHER_PLANNING_WRITER_200_PAYLOAD="+string(payload),
		"AETHER_PLANNING_WRITER_200_READY="+ready,
		"AETHER_PLANNING_WRITER_200_RELEASE="+release,
	)
	output := &bytes.Buffer{}
	command.Stdout, command.Stderr = output, output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	wait := make(chan error, 1)
	go func() { wait <- command.Wait() }()
	return &planningWriterProcess200{command: command, output: output, ready: ready, release: release, wait: wait}, ready
}

func planningWriterAssertBlockedBySession200(t *testing.T, root, mode string, payload []byte) {
	t.Helper()
	var child *planningWriterProcess200
	var waitErr error
	bypassed := false
	err := withPlanningMutationSession(root, "test-block-"+mode, func(*planningMutationSession) error {
		child, _ = planningWriterStartProcess200(t, root, mode, payload)
		planningWriterAwaitFile200(t, child.ready, true)
		select {
		case waitErr = <-child.wait:
			bypassed = true
		case <-time.After(500 * time.Millisecond):
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bypassed {
		waitErr = <-child.wait
	}
	planningWriterRequireProcessSuccess200(t, child, waitErr)
	if bypassed {
		t.Errorf("%s writer completed while another process held the repository mutation session", mode)
	}
}

func planningWriterProcessHelper200() {
	root := os.Getenv("AETHER_PLANNING_WRITER_200_ROOT")
	mode := os.Getenv("AETHER_PLANNING_WRITER_200_MODE")
	payload := []byte(os.Getenv("AETHER_PLANNING_WRITER_200_PAYLOAD"))
	result := planningWriterProcessResult200{}
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err == nil {
		store = s
	}
	if err == nil {
		err = os.WriteFile(os.Getenv("AETHER_PLANNING_WRITER_200_READY"), []byte("ready\n"), 0o600)
	}
	if err == nil {
		switch mode {
		case "plan-existing":
			_, err = runCodexPlanWithOptions(root, codexPlanOptions{Preset: "fast", PresetSet: true, VerificationDepth: "heavy"})
		case "intermediate":
			manifest := codexPlanManifest{Goal: "iteration", Root: root, PlanningRunID: "writer-session-run", Iteration: 1, Depth: "fast", PlanningDepth: "standard", TargetConfidence: 90, MaxIterations: 6}
			phasePlan := codexWorkerPlanArtifact{Confidence: codexPlanConfidence{Knowledge: 80, Requirements: 80, Risks: 80, Dependencies: 80, Effort: 80, Overall: 80}}
			_, err = persistIntermediatePlanningIteration(root, manifest, nil, codexScoutReport{}, phasePlan, phasePlan.Confidence, []string{"one gap"}, strings.Repeat("a", 64), codexPlanningLoop{}, codexPlanProvenance{})
		case "scout-stage":
			var decoded struct {
				Header   planningRunHeader     `json:"header"`
				Running  planningStageState    `json:"running"`
				Manifest planningStageManifest `json:"manifest"`
			}
			if err = json.Unmarshal(payload, &decoded); err == nil {
				err = persistPlanningScoutStage(root, decoded.Header, decoded.Running, decoded.Manifest)
			}
		case "insert":
			var request phaseInsertCandidateRequest
			if err = json.Unmarshal(payload, &request); err == nil {
				_, err = createPhaseInsertCandidate(root, request)
			}
		case "spec-stale":
			var request specificationRevisionRequest
			if err = json.Unmarshal(payload, &request); err == nil {
				err = planningWriterCommitStaleSpecification200(root, request)
			}
		default:
			err = fmt.Errorf("unknown helper mode %q", mode)
		}
	}
	result.Error = planningMutationErrorString(err)
	result.OK = err == nil
	_ = json.NewEncoder(os.Stdout).Encode(result)
}

func planningWriterCommitStaleSpecification200(root string, request specificationRevisionRequest) error {
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		return err
	}
	predecessorIndex := specificationRevisionIndex(*state.Specification, request.PredecessorRevisionID)
	if predecessorIndex < 0 {
		return fmt.Errorf("predecessor missing")
	}
	base, err := specificationAtPredecessor(*state.Specification, predecessorIndex)
	if err != nil {
		return err
	}
	built, successor, _, err := buildSpecificationSuccessor(base, state.Plan, request)
	if err != nil {
		return err
	}
	updated, err := cloneColonyState(state)
	if err != nil {
		return err
	}
	updated.Specification = &built
	targets, err := specificationStateProjectionTargets(updated)
	if err != nil {
		return err
	}
	if err := os.WriteFile(os.Getenv("AETHER_PLANNING_WRITER_200_READY"), []byte("derived\n"), 0o600); err != nil {
		return err
	}
	if err := planningWriterWaitForFile200(os.Getenv("AETHER_PLANNING_WRITER_200_RELEASE"), 10*time.Second); err != nil {
		return err
	}
	_, err = commitSpecificationTargets(root, specificationTransactionID("revise-stale-process", successor.ContentHash), "specification-revise", targets, specificationMutationOptions{})
	return err
}

func planningWriterRevisionRequest200(draft specificationMutationResult, label string, createdAt time.Time) specificationRevisionRequest {
	return specificationRevisionRequest{
		PredecessorRevisionID: draft.Revision.ID, PredecessorContentHash: draft.Revision.ContentHash,
		Scope: draft.Revision.Scope, CreatedAt: createdAt,
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify, Section: specificationSectionOutcomes,
			TargetID: draft.Revision.Outcomes[0].ID,
			Item:     specificationItemInput{Description: "Outcome from " + label + ".", EvidenceIDs: []string{"test:" + strings.ReplaceAll(label, " ", "-")}},
		}},
	}
}

func planningWriterRequireProcessSuccess200(t *testing.T, child *planningWriterProcess200, waitErr error) {
	t.Helper()
	var result planningWriterProcessResult200
	planningWriterDecodeProcess200(t, child, waitErr, &result)
	if result.Error != "" || !result.OK {
		t.Fatalf("writer subprocess failed: %s", result.Error)
	}
}

func planningWriterDecodeProcess200(t *testing.T, child *planningWriterProcess200, waitErr error, destination any) {
	t.Helper()
	if waitErr != nil {
		t.Fatalf("writer subprocess: %v\n%s", waitErr, child.output.String())
	}
	lines := strings.Split(strings.TrimSpace(child.output.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("writer subprocess returned no output")
	}
	encoded := ""
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			encoded = strings.TrimSpace(line)
			break
		}
	}
	if encoded == "" {
		t.Fatalf("writer subprocess returned no JSON result: %q", child.output.String())
	}
	if err := json.Unmarshal([]byte(encoded), destination); err != nil {
		t.Fatalf("decode writer subprocess output %q: %v", child.output.String(), err)
	}
}

func planningWriterAwaitFile200(t *testing.T, path string, wantExists bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_, err := os.Stat(path)
		if (err == nil) == wantExists {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for file existence=%t: %s", wantExists, path)
}

func planningWriterWaitForFile200(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for barrier %s", path)
}

func planningWriterArtifactInventory200(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, start := range []string{filepath.Join(root, ".aether", "data"), filepath.Join(root, ".planning")} {
		_ = filepath.WalkDir(start, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || strings.Contains(path, string(filepath.Separator)+"transactions"+string(filepath.Separator)) || strings.Contains(path, lifecycleTransactionDirectory) || strings.Contains(path, ".locks") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read inventory %s: %v", path, readErr)
			}
			digest := sha256.Sum256(content)
			relative, _ := filepath.Rel(root, path)
			result[filepath.ToSlash(relative)] = hex.EncodeToString(digest[:])
			return nil
		})
	}
	return result
}

func planningWriterAssertInventory200(t *testing.T, root string, before map[string]string) {
	t.Helper()
	after := planningWriterArtifactInventory200(t, root)
	if !equalPlanningWriterInventory200(before, after) {
		t.Fatalf("rejected input mutated artifacts\nbefore=%v\nafter=%v", sortedPlanningWriterInventory200(before), sortedPlanningWriterInventory200(after))
	}
}

func equalPlanningWriterInventory200(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, digest := range left {
		if right[path] != digest {
			return false
		}
	}
	return true
}

func sortedPlanningWriterInventory200(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for path, digest := range values {
		result = append(result, path+"="+digest)
	}
	sort.Strings(result)
	return result
}
