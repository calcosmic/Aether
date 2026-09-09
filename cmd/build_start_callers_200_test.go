package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
