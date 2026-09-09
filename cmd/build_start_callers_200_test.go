package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestBuildStartCallers200(t *testing.T) {
	t.Run("direct and host prepared starts use the canonical transaction", func(t *testing.T) {
		file := buildStartCallerFile200(t, "codex_build.go")
		for _, function := range []string{"runCodexBuildPlanOnlyWithOptions", "runCodexBuildWithOptions"} {
			calls := buildStartCallerCalls200(t, file, function)
			if calls["commitBuildStart"] != 1 {
				t.Fatalf("%s commitBuildStart calls = %d, want exactly one", function, calls["commitBuildStart"])
			}
			for _, forbidden := range []string{"beginBuildAttempt", "beginBuildAttemptRecord", "clearPhaseDispatchWindow", "cleanupStaleBuildAttemptArtifacts"} {
				if calls[forbidden] != 0 {
					t.Fatalf("%s still calls split build-start writer %s", function, forbidden)
				}
			}
		}

		queenCalls := buildStartCallerCalls200(t, file, "runCodexBuildQueenLed")
		if queenCalls["runCodexBuildPlanOnlyWithOptions"] != 1 {
			t.Fatalf("Queen-led start does not reuse the prepared plan-only path: calls=%v", queenCalls)
		}
		for _, forbidden := range []string{"beginBuildAttempt", "prepareBuildAttemptManifestBinding", "bindBuildAttemptManifest", "SaveJSON"} {
			if queenCalls[forbidden] != 0 {
				t.Fatalf("Queen-led start performs a second partial start through %s", forbidden)
			}
		}
	})

	t.Run("auxiliary starts use the canonical transaction", func(t *testing.T) {
		tests := []struct {
			file      string
			function  string
			forbidden []string
		}{
			{file: "codex_build_finalize.go", function: "runCodexBuildFinalize", forbidden: []string{"beginBuildAttempt", "beginBuildAttemptRecord"}},
			{file: "check_fix_attempt.go", function: "applyAutomaticCheckFixAttempt", forbidden: []string{"beginBuildAttempt", "beginBuildAttemptRecord"}},
			{file: "coherent_job_retry.go", function: "commitPartialBuildRetryPlan", forbidden: []string{"beginChildBuildAttempt", "beginBuildAttemptRecord", "attachBuildAttemptParentLink"}},
		}
		for _, test := range tests {
			calls := buildStartCallerCalls200(t, buildStartCallerFile200(t, test.file), test.function)
			if calls["commitBuildStart"] != 1 {
				t.Fatalf("%s commitBuildStart calls = %d, want exactly one", test.function, calls["commitBuildStart"])
			}
			for _, forbidden := range test.forbidden {
				if calls[forbidden] != 0 {
					t.Fatalf("%s still calls split build-start writer %s", test.function, forbidden)
				}
			}
		}
	})

	t.Run("plan-only and Queen-led starts publish one durable receipt", func(t *testing.T) {
		tests := []struct {
			name string
			run  func(string) (map[string]interface{}, error)
			mode string
		}{
			{
				name: "plan-only",
				run: func(root string) (map[string]interface{}, error) {
					result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{})
					return result, err
				},
				mode: "plan-only",
			},
			{
				name: "queen-led",
				run: func(root string) (map[string]interface{}, error) {
					result, _, _, _, err := runCodexBuildQueenLed(root, 1, nil, codexBuildOptions{})
					return result, err
				},
				mode: "queen-led",
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				root := setupExternalBuildAttemptTest(t)
				result, err := test.run(root)
				if err != nil {
					t.Fatalf("start: %v", err)
				}
				attemptDisplay, _ := result["attempt"].(string)
				attemptRel := strings.TrimPrefix(filepath.ToSlash(attemptDisplay), ".aether/data/")
				if attemptRel == "" {
					t.Fatal("start result omitted attempt path")
				}
				receiptRel := strings.TrimSuffix(attemptRel, ".json") + ".start-receipt.json"
				if _, err := os.Stat(filepath.Join(root, ".aether", "data", filepath.FromSlash(receiptRel))); err != nil {
					t.Fatalf("durable start receipt missing: %v", err)
				}
				records := make([]buildAttemptRecord, 0)
				for _, record := range listBuildAttemptsForPhase(1) {
					if validBuildAttemptID(record.ID) {
						records = append(records, record)
					}
				}
				if len(records) != 1 {
					t.Fatalf("attempt records = %d, want one", len(records))
				}
				manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
				if !ok || manifest.DispatchMode != test.mode || manifest.AttemptID != records[0].ID {
					t.Fatalf("manifest was not atomically bound for %s: %+v", test.mode, manifest)
				}
			})
		}
	})

	t.Run("callbacks observe receipts and transaction faults authorize no work", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		callbacks := 0
		_, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
			BuildStartOptions: buildStartOptions{Dispatch: func(receipt buildStartReceipt) error {
				callbacks++
				if _, err := os.Stat(filepath.Join(root, ".aether", "data", filepath.FromSlash(receipt.Path))); err != nil {
					return fmt.Errorf("callback ran before durable receipt: %w", err)
				}
				return nil
			}},
		})
		if err != nil || callbacks != 1 {
			t.Fatalf("successful plan-only callback = %d, err=%v; want one callback after receipt", callbacks, err)
		}

		root = setupExternalBuildAttemptTest(t)
		callbacks = 0
		injected := errors.New("plan-34 caller fault")
		_, _, _, _, err = runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
			BuildStartOptions: buildStartOptions{
				Fault: func(point string) error {
					if point == buildStartBeforeCommitFaultPoint {
						return injected
					}
					return nil
				},
				Dispatch: func(buildStartReceipt) error { callbacks++; return nil },
			},
		})
		if !errors.Is(err, injected) {
			t.Fatalf("faulted plan-only start error = %v, want injected fault", err)
		}
		if callbacks != 0 {
			t.Fatalf("faulted plan-only start authorized %d callbacks", callbacks)
		}
	})

	t.Run("direct worker invocation follows the receipt", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		invoker := &timeoutRecordingInvoker{}
		originalFactory := newCodexWorkerInvoker
		newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
		t.Cleanup(func() { newCodexWorkerInvoker = originalFactory })
		injected := errors.New("plan-34 direct start fault")
		_, err := runCodexBuildWithOptions(root, 1, nil, false, codexBuildOptions{
			BuildStartOptions: buildStartOptions{Fault: func(point string) error {
				if point == buildStartBeforeCommitFaultPoint {
					return injected
				}
				return nil
			}},
		})
		if !errors.Is(err, injected) {
			t.Fatalf("faulted direct start error = %v, want injected fault", err)
		}
		invoker.mu.Lock()
		invocations := len(invoker.calls)
		invoker.mu.Unlock()
		if invocations != 0 {
			t.Fatalf("faulted direct start invoked %d workers", invocations)
		}
	})

	t.Run("bound external replay and Queen reuse do not duplicate attempts", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		manifest, completion := prepareExternalBuildCompletion(t, root)
		if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
			t.Fatalf("first bound external finalize: %v", err)
		}
		if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
			t.Fatalf("bound external replay: %v", err)
		}
		validAttempts := 0
		for _, record := range listBuildAttemptsForPhase(1) {
			if validBuildAttemptID(record.ID) {
				validAttempts++
			}
		}
		if validAttempts != 1 {
			t.Fatalf("bound replay for %s left %d attempts, want one", manifest.AttemptID, validAttempts)
		}
	})

	t.Run("production callers have no partial writer before canonical commit", func(t *testing.T) {
		tests := []struct {
			file       string
			function   string
			boundary   string
			dispatches []string
		}{
			{file: "codex_build.go", function: "runCodexBuildPlanOnlyWithOptions", boundary: "commitBuildStart"},
			{file: "codex_build.go", function: "runCodexBuildQueenLed", boundary: "runCodexBuildPlanOnlyWithOptions"},
			{file: "codex_build.go", function: "runCodexBuildWithOptions", boundary: "commitBuildStart", dispatches: []string{"executeCodexBuildDispatches"}},
			{file: "codex_build_finalize.go", function: "runCodexBuildFinalize", boundary: "commitBuildStart"},
			{file: "check_fix_attempt.go", function: "applyAutomaticCheckFixAttempt", boundary: "commitBuildStart", dispatches: []string{"dispatchBatchByWaveWithVisuals"}},
			{file: "coherent_job_retry.go", function: "commitPartialBuildRetryPlan", boundary: "commitBuildStart"},
		}
		legacy := map[string]bool{
			"beginBuildAttempt": true, "beginBuildAttemptRecord": true,
			"beginChildBuildAttempt": true, "attachBuildAttemptParentLink": true,
		}
		preCommitWriters := map[string]bool{
			"SaveJSON": true, "UpdateJSONAtomically": true, "AtomicWrite": true,
			"Remove": true, "RemoveAll": true, "clearPhaseDispatchWindow": true,
			"cleanupStaleBuildAttemptArtifacts": true,
		}
		for _, test := range tests {
			calls := buildStartCallerOrderedCalls200(t, buildStartCallerFile200(t, test.file), test.function)
			boundary := buildStartCallerFirstCall200(t, calls, test.boundary)
			for _, call := range calls {
				if legacy[call.Name] {
					t.Fatalf("%s retains legacy partial writer %s", test.function, call.Name)
				}
				if call.Pos < boundary.Pos && preCommitWriters[call.Name] {
					t.Fatalf("%s calls direct start writer %s before %s", test.function, call.Name, test.boundary)
				}
			}
			for _, dispatch := range test.dispatches {
				if got := buildStartCallerFirstCall200(t, calls, dispatch); got.Pos < boundary.Pos {
					t.Fatalf("%s dispatches through %s before durable receipt", test.function, dispatch)
				}
			}
		}
	})
}

// TestBuildStartConcurrentProcesses200 deterministically pauses the public
// plan-only caller after it has prepared revision A but before it enters the
// repository session. A second OS process (the parent test process) accepts
// revision B, after which the stale build is released and must refuse without
// invoking its post-receipt dispatch callback.
func TestBuildStartConcurrentProcesses200(t *testing.T) {
	if os.Getenv("AETHER_BUILD_START_200_HELPER") == "1" {
		buildStartConcurrentProcessHelper200(t)
		return
	}

	root, _, _ := buildStartTransaction200CurrentAuthorityFixture(t)
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatal(err)
	}
	store = s
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	// The accepted-authority fixture is intentionally usable below the CLI
	// layer and therefore does not need a colony goal. This race exercises the
	// public build entry point, whose honest initialization gate does.
	goal := "Serialize a public build against accepted living-plan revisions"
	state.Goal = &goal
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("complete public build fixture: %v", err)
	}
	base, ok := activePlanRevision(state.Plan)
	if !ok || state.Specification == nil {
		t.Fatal("accepted race fixture lacks active plan/specification")
	}
	specification, ok := currentSpecificationRevision(*state.Specification)
	if !ok || len(specification.Requirements) == 0 {
		t.Fatal("accepted race fixture lacks a current requirement")
	}
	inserted, err := createPhaseInsertCandidate(root, phaseInsertCandidateRequest{
		After: len(state.Plan.Phases), Name: "Concurrent accepted revision",
		Description:                "Replace the public build authority while its request is prepared.",
		Constraints:                "The stale build must never authorize worker dispatch.",
		SpecificationItemID:        specification.Requirements[0].ID,
		ExpectedBasePlanRevisionID: base.ID,
		CreatedAt:                  time.Date(2026, time.September, 9, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create concurrent candidate: %v", err)
	}

	temp := t.TempDir()
	ready := filepath.Join(temp, "prepared")
	release := filepath.Join(temp, "release")
	dispatched := filepath.Join(temp, "dispatched")
	command := exec.Command(os.Args[0], "-test.run=^TestBuildStartConcurrentProcesses200$")
	command.Env = append(os.Environ(),
		"AETHER_BUILD_START_200_HELPER=1",
		"AETHER_BUILD_START_200_ROOT="+root,
		"AETHER_BUILD_START_200_READY="+ready,
		"AETHER_BUILD_START_200_RELEASE="+release,
		"AETHER_BUILD_START_200_DISPATCHED="+dispatched,
		"AETHER_ROOT="+root,
	)
	var output bytes.Buffer
	command.Stdout, command.Stderr = &output, &output
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	if err := buildStartWaitForPath200(ready, 10*time.Second); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("public build did not reach prepared boundary: %v\n%s", err, output.String())
	}

	accepted, acceptErr := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(inserted.Candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:plan-34-race", AcceptedAt: time.Date(2026, time.September, 9, 13, 1, 0, 0, time.UTC),
	})
	if err := os.WriteFile(release, []byte("continue\n"), 0o600); err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatal(err)
	}
	if acceptErr != nil {
		_ = command.Wait()
		t.Fatalf("accept winning revision: %v", acceptErr)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("stale public build helper: %v\n%s", err, output.String())
	}

	var response buildStartConcurrentProcessResult200
	encoded := ""
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		if strings.HasPrefix(line, "{") {
			encoded = line
		}
	}
	if err := json.Unmarshal([]byte(encoded), &response); err != nil {
		t.Fatalf("decode build helper output %q: %v", output.String(), err)
	}
	if response.Error == "" || (!strings.Contains(response.Error, "stale") && !strings.Contains(response.Error, "authority")) {
		t.Fatalf("losing public build error = %q, want stale-authority refusal", response.Error)
	}
	if _, err := os.Stat(dispatched); !os.IsNotExist(err) {
		t.Fatalf("losing public build authorized dispatch marker: %v", err)
	}
	state, err = loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if state.Plan.ActiveRevisionID != accepted.Revision.ID || state.State != colony.StateREADY {
		t.Fatalf("race mixed authority/state: active=%q state=%s, want %q/READY", state.Plan.ActiveRevisionID, state.State, accepted.Revision.ID)
	}
	validAttempts := 0
	for _, record := range listBuildAttemptsForPhase(1) {
		if validBuildAttemptID(record.ID) {
			validAttempts++
		}
	}
	if validAttempts != 0 {
		t.Fatalf("losing public build created %d attempts", validAttempts)
	}
}

type buildStartConcurrentProcessResult200 struct {
	Attempt string `json:"attempt,omitempty"`
	Error   string `json:"error,omitempty"`
}

func buildStartConcurrentProcessHelper200(t *testing.T) {
	root := os.Getenv("AETHER_BUILD_START_200_ROOT")
	var err error
	store, err = storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatal(err)
	}
	result, _, _, _, runErr := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{
		BuildStartBeforeCommit: func() error {
			if err := os.WriteFile(os.Getenv("AETHER_BUILD_START_200_READY"), []byte("prepared\n"), 0o600); err != nil {
				return err
			}
			return buildStartWaitForPath200(os.Getenv("AETHER_BUILD_START_200_RELEASE"), 10*time.Second)
		},
		BuildStartOptions: buildStartOptions{Dispatch: func(receipt buildStartReceipt) error {
			if _, err := os.Stat(filepath.Join(root, ".aether", "data", filepath.FromSlash(receipt.Path))); err != nil {
				return fmt.Errorf("dispatch callback preceded durable receipt: %w", err)
			}
			return os.WriteFile(os.Getenv("AETHER_BUILD_START_200_DISPATCHED"), []byte(receipt.ContentHash+"\n"), 0o600)
		}},
	})
	response := buildStartConcurrentProcessResult200{}
	if runErr != nil {
		response.Error = runErr.Error()
	} else if attempt, ok := result["attempt"].(string); ok {
		response.Attempt = attempt
	}
	content, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(content))
}

func buildStartWaitForPath200(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s", path)
		}
		time.Sleep(time.Millisecond)
	}
}

type buildStartCallerCall200 struct {
	Name string
	Pos  token.Pos
}

func buildStartCallerOrderedCalls200(t *testing.T, file *ast.File, function string) []buildStartCallerCall200 {
	t.Helper()
	var calls []buildStartCallerCall200
	found := false
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function {
			continue
		}
		found = true
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch target := call.Fun.(type) {
			case *ast.Ident:
				name = target.Name
			case *ast.SelectorExpr:
				name = target.Sel.Name
			}
			if name != "" {
				calls = append(calls, buildStartCallerCall200{Name: name, Pos: call.Pos()})
			}
			return true
		})
		break
	}
	if !found {
		t.Fatalf("function %s not found", function)
	}
	return calls
}

func buildStartCallerFirstCall200(t *testing.T, calls []buildStartCallerCall200, name string) buildStartCallerCall200 {
	t.Helper()
	for _, call := range calls {
		if call.Name == name {
			return call
		}
	}
	t.Fatalf("call %s not found", name)
	return buildStartCallerCall200{}
}

func buildStartCallerFile200(t *testing.T, name string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return file
}

func buildStartCallerCalls200(t *testing.T, file *ast.File, function string) map[string]int {
	t.Helper()
	calls := map[string]int{}
	found := false
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Name.Name != function {
			continue
		}
		found = true
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch target := call.Fun.(type) {
			case *ast.Ident:
				calls[target.Name]++
			case *ast.SelectorExpr:
				calls[target.Sel.Name]++
			}
			return true
		})
		break
	}
	if !found {
		t.Fatalf("function %s not found", function)
	}
	return calls
}
