package cmd

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRuntimeRecoveryRoutes199 is the executable vocabulary ratchet for the
// owner-visible recovery doors. It deliberately combines rendered output with
// an AST string-literal check: comments and implementation identifiers do not
// define product vocabulary, but a newly added emitted command must not evade
// the exercised fixtures.
func TestRuntimeRecoveryRoutes199(t *testing.T) {
	suggestions := collectRuntimeRecoverySuggestions199(t)
	for name, output := range suggestions {
		t.Run(name, func(t *testing.T) {
			assertNoRetiredRuntimeRecoverySuggestion199(t, output)
		})
	}

	for _, name := range []string{"codex_visuals.go", "init_cmd.go", "entomb_cmd.go"} {
		t.Run("active literals "+name, func(t *testing.T) {
			assertNoRetiredRuntimeRecoveryLiterals199(t, name)
		})
	}

	for _, name := range []string{"visual preserved workspaces", "no-color preserved workspaces"} {
		output := suggestions[name]
		if !strings.Contains(output, "aether maintenance recovery-inspect") {
			t.Fatalf("%s must offer the read-only preserved-work inspection route:\n%s", name, output)
		}
		if !strings.Contains(output, "aether resume") {
			t.Fatalf("%s must keep resume as the restoration route:\n%s", name, output)
		}
	}
}

// collectRuntimeRecoverySuggestions199 returns the executed fixture set so a
// later vocabulary ratchet can reuse these exact renderings rather than
// reimplementing a source-only scan.
func collectRuntimeRecoverySuggestions199(t *testing.T) map[string]string {
	t.Helper()
	saveGlobals(t)

	result := map[string]interface{}{
		"next": "aether resume",
		"worktrees_preserved": map[string]interface{}{
			"cleaned":   0,
			"preserved": 1,
		},
	}
	visual := renderResumeVisual(result, "", true)

	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var visualOut bytes.Buffer
	stdout = &visualOut
	outputWorkflow(result, visual)

	t.Setenv("NO_COLOR", "1")
	var noColorOut bytes.Buffer
	stdout = &noColorOut
	outputWorkflow(result, visual)

	t.Setenv("AETHER_OUTPUT_MODE", "json")
	var jsonOut bytes.Buffer
	stdout = &jsonOut
	outputWorkflow(result, visual)

	return map[string]string{
		"visual preserved workspaces":   visualOut.String(),
		"no-color preserved workspaces": noColorOut.String(),
		"json lifecycle result":         jsonOut.String(),
	}
}

func assertNoRetiredRuntimeRecoverySuggestion199(t *testing.T, output string) {
	t.Helper()
	for _, retired := range []string{"aether recover", "aether abandon", "pause-colony", "resume-colony"} {
		if strings.Contains(output, retired) {
			t.Fatalf("retired recovery suggestion %q leaked into runtime output:\n%s", retired, output)
		}
	}
}

func assertNoRetiredRuntimeRecoveryLiterals199(t *testing.T, name string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	file := filepath.Join(wd, name)
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value := strings.Trim(literal.Value, "\"")
		assertNoRetiredRuntimeRecoverySuggestion199(t, value)
		return true
	})
}
