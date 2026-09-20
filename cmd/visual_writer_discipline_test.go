package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// visualWriterExemptions lists the call sites allowed to write to stdout/stderr
// without going through writeVisualOutput, each with the reason it is not
// human-facing visual text. Anything not on this list must use
// writeVisualOutput so platform command naming is applied.
//
// Keyed by "<file>:<function>".
var visualWriterExemptions = map[string]string{
	// Machine surfaces: wrappers, the TS host and downstream tooling parse
	// these and execute or store the commands they name. Translating would
	// hand `/ant-continue` to exec, which is not a binary.
	"helpers.go:outputOK":                        "JSON success envelope",
	"helpers.go:outputError":                     "JSON error envelope; the visual branch uses writeVisualOutput",
	"hook_cmds.go:runHookContext":                "JSON hook payload consumed by the platform",
	"hook_cmds.go:emitHookBlock":                 "JSON hook block consumed by the platform",
	"unblock_cmd.go:init":                        "JSON branch; the visual branch uses writeVisualOutput",
	"spawn_reap.go:writeSpawnOrphansEnvelope":    "JSON branch; the visual branch uses writeVisualOutput",
	"spend_cmd.go:writeSpendEnvelope":            "JSON branch; the visual branch uses writeVisualOutput",
	"medic_cmd.go:runMedic":                      "JSON branches and mode-agnostic error strings",
	"build_print_brief.go:runPrintBrief":         "worker brief — agent-facing; workers invoke the raw CLI",
	"build_print_brief.go:printWorkerBriefs":     "worker brief — agent-facing; workers invoke the raw CLI",
	"eventbus.go:streamEventBusNDJSON":           "NDJSON event stream",
	"exchange.go:runExportPheromones":            "XML export, normally piped to a file",
	"exchange.go:runExportRegistry":              "XML export, normally piped to a file",
	"exchange.go:runExportWisdom":                "XML export, normally piped to a file",
	"audit_catalog.go:outputCatalogJSON":         "JSON audit artifact",
	"audit_catalog.go:outputReliabilityJSON":     "JSON audit artifact",
	"audit_catalog.go:outputCatalogMarkdown":     "markdown audit artifact, normally piped to a file",
	"audit_catalog.go:outputReliabilityMarkdown": "markdown audit artifact, normally piped to a file",

	"oracle_progress.go:appendOracleProgressEvent": "diagnostics for a failed progress-log write; the log itself is machine-read by --follow and the line the operator sees goes through emitVisualLine",

	"gate.go:gateResultsReadCmd":             "JSON gate results for scripts and wrappers",
	"gate.go:gateResultsWriteCmd":            "JSON write acknowledgement",
	"gate.go:shouldSkipGateCmd":              "bare boolean consumed by scripts",
	"gate.go:gateRecoveryTemplateCmd":        "recovery template text consumed by wrappers",
	"source_check.go:runSourceCheckCommand":  "JSON branch; the visual branch uses outputWorkflow",
	"trace_cmds.go:traceExportCmd":           "JSON trace export",
	"unblock_cmd.go:unblockCmd":              "JSON branch; the visual branch uses writeVisualOutput",
	"recovery_engine.go:renderRecoveryMenu":  "JSON error envelope; the visual branch returns text for the caller to render",
	"recover.go:runRecover":                  "JSON branch; the visual branch uses visualFprint",
	"integrity_cmd.go:renderIntegrityResult": "JSON branch behind --json; the visual branch uses visualFprint",
	"porter_cmd.go:renderPorterResult":       "JSON branch behind --json",
	"fixer_dispatch.go:dispatchFixer":        "JSON dispatch payload; prose notices use visualFprintf",

	// Raw worker output echoed under --verbose. This is what a worker printed;
	// rewriting it would misreport what actually happened.
	"output_filter.go:filteredPrintln": "raw worker output under --verbose",
	"output_filter.go:filteredFprintf": "raw worker output under --verbose",
}

// declNameAndBody returns a stable name for a top-level declaration and a node
// to search under it. Function declarations give their own name; a var block
// gives its first declared name, which is how cobra commands are written
// (`var pheromoneDisplayCmd = &cobra.Command{...}`).
func declNameAndBody(decl ast.Decl) (string, ast.Node) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Body == nil {
			return "", nil
		}
		return d.Name.Name, d.Body
	case *ast.GenDecl:
		if d.Tok != token.VAR {
			return "", nil
		}
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 {
				continue
			}
			return vs.Names[0].Name, vs
		}
	}
	return "", nil
}

// TestHumanFacingOutputGoesThroughWriteVisualOutput fails when a new direct
// write to stdout/stderr appears without an exemption.
//
// This is the guard that makes the writeVisualOutput chokepoint hold. Command
// naming is only correct if every human-facing byte passes through it, and the
// failure mode is silent: a new fmt.Fprint(stdout, ...) renders perfectly well,
// it just tells a Claude Code user to type a command that does not exist there.
// The welcome banner did exactly that — the first thing a new user ever saw
// listed three commands none of which they could run.
func TestHumanFacingOutputGoesThroughWriteVisualOutput(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	var offenders []string
	fset := token.NewFileSet()

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		// Walk every top-level declaration, not just FuncDecl. Most cobra
		// commands are `var xCmd = &cobra.Command{RunE: func(...){...}}` — a
		// function literal inside a var, which an ast.FuncDecl-only walk skips
		// entirely. That blind spot hid `pheromone-display` writing a raw table
		// and a JSON envelope to stdout on every invocation.
		for _, decl := range file.Decls {
			enclosing, body := declNameAndBody(decl)
			if body == nil {
				continue
			}

			ast.Inspect(body, func(inner ast.Node) bool {
				call, ok := inner.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok || pkg.Name != "fmt" {
					return true
				}
				switch sel.Sel.Name {
				case "Fprint", "Fprintf", "Fprintln":
				default:
					return true
				}
				if len(call.Args) == 0 {
					return true
				}
				target, ok := call.Args[0].(*ast.Ident)
				if !ok || (target.Name != "stdout" && target.Name != "stderr") {
					return true
				}
				key := fmt.Sprintf("%s:%s", name, enclosing)
				if _, exempt := visualWriterExemptions[key]; exempt {
					return true
				}
				offenders = append(offenders, fmt.Sprintf("%s (line %d)", key, fset.Position(call.Pos()).Line))
				return true
			})
		}
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("direct writes to stdout/stderr bypass writeVisualOutput, so command names are not translated for the reader's platform:\n  %s\n\n"+
			"Either route the text through writeVisualOutput, or add an entry to visualWriterExemptions saying why it is not human-facing visual text.",
			strings.Join(offenders, "\n  "))
	}
}
