package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLegacyAutopilotCommandsAbsent(t *testing.T) {
	legacyCommands := []string{
		"autopilot-init",
		"autopilot-update",
		"autopilot-status",
		"autopilot-stop",
		"autopilot-set-headless",
		"autopilot-headless-check",
		"autopilot-resume",
	}
	wantedSupported := map[string]bool{"run": false, "status": false}
	for _, entry := range buildAuditCatalog(rootCmd) {
		for _, name := range legacyCommands {
			if entry.Name == name {
				t.Errorf("legacy autopilot command %q is still registered", name)
			}
		}
		if _, ok := wantedSupported[entry.Name]; ok {
			wantedSupported[entry.Name] = true
		}
	}
	for name, found := range wantedSupported {
		if !found {
			t.Errorf("supported autopilot lifecycle command %q is not registered", name)
		}
	}

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate autopilot removal test")
	}
	productionFiles, err := filepath.Glob(filepath.Join(filepath.Dir(testFile), "*.go"))
	if err != nil {
		t.Fatalf("list command sources: %v", err)
	}
	legacySymbols := map[string]struct{}{
		"autopilotInitCmd":              {},
		"autopilotUpdateCmd":            {},
		"autopilotStatusCmd":            {},
		"autopilotStopCmd":              {},
		"autopilotSetHeadlessCmd":       {},
		"autopilotHeadlessCheckCmd":     {},
		"autopilotResumeCmd":            {},
		"checkAutopilotPauseConditions": {},
	}
	for _, path := range productionFiles {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", filepath.Base(path), err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			identifier, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			if _, retired := legacySymbols[identifier.Name]; retired {
				t.Errorf("legacy autopilot symbol %q remains in %s", identifier.Name, filepath.Base(path))
			}
			return true
		})
	}
}
