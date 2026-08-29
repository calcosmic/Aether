package cmd

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Phase 198 plan 09 (SHOW-02, roadmap criterion 2) -- the invariant closing
// this phase: every top-level key a finalizer's result map carries is either
// read by a render function reachable from that workflow's screen, or listed
// in a shrink-only exception file with a written reason. This is an
// invariant over what the finalizers carry, not a check that a page happens
// to have the right headings (CONTEXT.md, Claude's Discretion) -- so both
// sides of the comparison are derived programmatically, never typed as a
// literal key list, per CLAUDE.md's "derive fixture values the way the
// runtime derives them" rule.

// renderedFieldAllowlistEntry is one committed exception: a finalizer's
// top-level result-map key that is genuinely provenance-only, machine-only,
// or a discovered-and-deferred gap this plan's own file scope cannot fix
// (mirroring the 198-04 precedent: record honestly, do not silently patch a
// file another concurrent plan owns). Every entry carries a reason and the
// name of the finalizer it belongs to.
type renderedFieldAllowlistEntry struct {
	Key       string `json:"key"`
	Finalizer string `json:"finalizer"`
	Reason    string `json:"reason"`
}

// loadRenderedFieldAllowlist reads a committed allowlist JSON file, following
// the exact loader shape TestOrphanAllowlistOnlyShrinks established
// (cmd/subcommand_reachability_ratchet_test.go) for this repo's other
// shrink-only exception list.
func loadRenderedFieldAllowlist(t *testing.T, path string) []renderedFieldAllowlistEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []renderedFieldAllowlistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

// renderedFieldCarriedKeys derives the top-level key set of a
// map[string]interface{} by reading the keys off the produced map --
// never a literal key list typed into the test. The maps themselves come
// from wrapperParity*Result(), the fixture-construction pattern
// 198-PATTERNS.md establishes for this exact phase: real Go structs for
// every nested value (codexPlanConfidence, codexContinueVerificationReport,
// etc.), the same shapes runCodexPlanFinalize / runCodexContinueFinalize /
// completeSealRuntime actually populate, not hand-typed nested maps.
func renderedFieldCarriedKeys(m map[string]interface{}) map[string]bool {
	keys := make(map[string]bool, len(m))
	for k := range m {
		keys[k] = true
	}
	return keys
}

// renderedFieldEntryPoint is one root the call-graph walk starts from: a
// function name plus the name (parameter OR local variable -- go/ast makes
// no distinction the walk needs) of the identifier carrying the top-level
// map/raw value at that point.
type renderedFieldEntryPoint struct {
	Func     string
	MapParam string
}

// renderedFieldFinalizer names one of the five finalizer result-map shapes
// this invariant covers (must_haves.artifacts, 198-09-PLAN.md), its carried
// top-level keys, and the render entry points whose call graphs together
// must read each of those keys. Two roots per finalizer, mirroring both
// paths a completion actually renders on (read_first, 198-09-PLAN.md Task
// 1): the direct-path entry renderer in cmd/codex_visuals.go, and the
// chat-path bridge in cmd/closeout_direct_render.go, which independently
// reads several of the same top-level fields (phase lookups, the seal
// "sealed"/"summary" gate) before ever calling the renderer.
type renderedFieldFinalizer struct {
	Name    string
	Carried map[string]bool
	Entries []renderedFieldEntryPoint
}

// renderedFieldFinalizers builds the five finalizer fixtures this invariant
// scans: plan completed, plan mid-loop, continue advanced, continue blocked,
// and seal (198-09-PLAN.md acceptance criteria).
func renderedFieldFinalizers() []renderedFieldFinalizer {
	return []renderedFieldFinalizer{
		{
			Name:    "plan completed",
			Carried: renderedFieldCarriedKeys(wrapperParityPlanFinalizeResult()),
			Entries: []renderedFieldEntryPoint{
				{Func: "renderPlanVisual", MapParam: "result"},
				{Func: "closeoutPlanDirectVisual", MapParam: "raw"},
			},
		},
		{
			Name:    "plan mid-loop",
			Carried: renderedFieldCarriedKeys(wrapperParityPlanIterationResult()),
			Entries: []renderedFieldEntryPoint{
				{Func: "renderPlanVisual", MapParam: "result"},
				{Func: "closeoutPlanDirectVisual", MapParam: "raw"},
			},
		},
		{
			Name:    "continue advanced",
			Carried: renderedFieldCarriedKeys(wrapperParityAdvanceResult()),
			Entries: []renderedFieldEntryPoint{
				{Func: "renderContinueVisual", MapParam: "result"},
				{Func: "closeoutContinueDirectVisual", MapParam: "raw"},
				// closeLifecycleRun (advanceExternalContinue's own last call,
				// before this result is ever handed to a renderer) folds the
				// map's own "next" into the unified next-action envelope
				// renderLifecycleClosingForState's closing card reads back --
				// the exact mechanism that makes a blocked continue's "next"
				// (its named recovery command) reach the screen. Seeding it
				// here traces that real pipeline instead of allow-listing
				// around it.
				{Func: "closeLifecycleRun", MapParam: "result"},
			},
		},
		{
			Name:    "continue blocked",
			Carried: renderedFieldCarriedKeys(wrapperParityBlockedResult()),
			Entries: []renderedFieldEntryPoint{
				{Func: "renderContinueBlockedVisual", MapParam: "result"},
				{Func: "closeoutContinueDirectVisual", MapParam: "raw"},
				{Func: "closeLifecycleRun", MapParam: "result"},
			},
		},
		{
			Name:    "seal",
			Carried: renderedFieldCarriedKeys(wrapperParitySealResult()),
			Entries: []renderedFieldEntryPoint{
				{Func: "renderSealVisual", MapParam: "result"},
				{Func: "closeoutSealDirectVisual", MapParam: "raw"},
				{Func: "closeLifecycleRun", MapParam: "result"},
			},
		},
	}
}

// renderedFieldASTFuncs parses every non-test .go file in the cmd package
// (the working directory `go test` runs in) and indexes every top-level
// function declaration by name, so the render call graph can be traced
// without needing go/types or a compiled build.
func renderedFieldASTFuncs(t *testing.T) map[string]*ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parse cmd package for render call graph: %v", err)
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
	return funcs
}

// renderedFieldCalleeName returns the plain identifier name of a call
// expression's function, or ok=false for a method call, package-qualified
// call, or anything else this invariant does not need to follow (this
// codebase's render helpers are all plain package-level functions).
func renderedFieldCalleeName(fun ast.Expr) (string, bool) {
	ident, ok := fun.(*ast.Ident)
	if !ok {
		return "", false
	}
	return ident.Name, true
}

// renderedFieldParamNameAtArgIndex maps a call's positional argument index to
// the callee's own parameter name, accounting for Go's grouped-parameter
// declaration shape (func f(a, b string, c int)). Returns "" when the
// position has no name (an unnamed parameter, or the index is out of range).
func renderedFieldParamNameAtArgIndex(fn *ast.FuncDecl, argIndex int) string {
	if fn.Type.Params == nil {
		return ""
	}
	pos := 0
	for _, field := range fn.Type.Params.List {
		names := field.Names
		if len(names) == 0 {
			if pos == argIndex {
				return ""
			}
			pos++
			continue
		}
		for _, name := range names {
			if pos == argIndex {
				return name.Name
			}
			pos++
		}
	}
	return ""
}

// renderedFieldKeysReadByFunc walks fn's body, collecting every string
// literal indexed off the identifier named mapParam (fn's own parameter
// carrying the map/raw value at this point in the call graph), and recurses
// into any callee that receives that SAME value unnarrowed -- a bare
// identifier argument matching mapParam, not a further index expression --
// the exact "renderLifecycleClosing(result, ...)" passthrough shape a key
// read by a helper must follow (198-09-PLAN.md Task 1, item 2). visited
// prevents infinite recursion across mutually-calling functions.
func renderedFieldKeysReadByFunc(fn *ast.FuncDecl, mapParam string, funcs map[string]*ast.FuncDecl, visited map[string]bool) map[string]bool {
	keys := map[string]bool{}
	if fn == nil || fn.Body == nil {
		return keys
	}
	signature := fn.Name.Name + "|" + mapParam
	if visited[signature] {
		return keys
	}
	visited[signature] = true

	// Direct literal index reads: result["x"] / raw["x"].
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		idx, ok := n.(*ast.IndexExpr)
		if !ok {
			return true
		}
		ident, ok := idx.X.(*ast.Ident)
		if !ok || ident.Name != mapParam {
			return true
		}
		lit, ok := idx.Index.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		key, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		keys[key] = true
		return true
	})

	// Passthrough calls: a callee that receives the whole mapParam value
	// unnarrowed also gets to read top-level keys -- follow it.
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		calleeName, ok := renderedFieldCalleeName(call.Fun)
		if !ok {
			return true
		}
		callee, ok := funcs[calleeName]
		if !ok {
			return true
		}
		for argIndex, arg := range call.Args {
			argIdent, ok := arg.(*ast.Ident)
			if !ok || argIdent.Name != mapParam {
				continue
			}
			calleeParam := renderedFieldParamNameAtArgIndex(callee, argIndex)
			if calleeParam == "" {
				continue
			}
			for k := range renderedFieldKeysReadByFunc(callee, calleeParam, funcs, visited) {
				keys[k] = true
			}
		}
		return true
	})

	return keys
}

// renderedFieldKeysReadForEntries walks every one of a finalizer's entry
// points and unions the keys their call graphs read.
func renderedFieldKeysReadForEntries(t *testing.T, entries []renderedFieldEntryPoint, funcs map[string]*ast.FuncDecl) map[string]bool {
	t.Helper()
	all := map[string]bool{}
	for _, entry := range entries {
		fn, ok := funcs[entry.Func]
		if !ok {
			t.Fatalf("entry function %q not found while parsing the cmd package -- has it been renamed?", entry.Func)
		}
		for k := range renderedFieldKeysReadByFunc(fn, entry.MapParam, funcs, map[string]bool{}) {
			all[k] = true
		}
	}
	return all
}

// TestRenderedVisualsShowEveryCarriedField is the invariant itself
// (SHOW-02, roadmap criterion 2): for each of the five finalizer result
// maps, every top-level key is either read by at least one render function
// reachable from that workflow's screen, or listed in
// testdata/rendered_field_allowlist.json with a reason. A key added to a
// finalizer result map and read by nothing fails this test by name.
func TestRenderedVisualsShowEveryCarriedField(t *testing.T) {
	funcs := renderedFieldASTFuncs(t)
	allowlist := loadRenderedFieldAllowlist(t, "testdata/rendered_field_allowlist.json")

	allowedByFinalizer := map[string]map[string]string{}
	for i, e := range allowlist {
		if strings.TrimSpace(e.Key) == "" {
			t.Fatalf("rendered_field_allowlist.json entry %d has an empty key", i)
		}
		if strings.TrimSpace(e.Finalizer) == "" {
			t.Fatalf("rendered_field_allowlist.json entry %d (key %q) has an empty finalizer", i, e.Key)
		}
		if strings.TrimSpace(e.Reason) == "" {
			t.Fatalf("rendered_field_allowlist.json entry for key %q (finalizer %q) has an empty reason", e.Key, e.Finalizer)
		}
		if allowedByFinalizer[e.Finalizer] == nil {
			allowedByFinalizer[e.Finalizer] = map[string]string{}
		}
		allowedByFinalizer[e.Finalizer][e.Key] = e.Reason
	}

	for _, f := range renderedFieldFinalizers() {
		f := f
		t.Run(strings.ReplaceAll(f.Name, " ", "_"), func(t *testing.T) {
			readKeys := renderedFieldKeysReadForEntries(t, f.Entries, funcs)
			allowed := allowedByFinalizer[f.Name]

			var missing []string
			for key := range f.Carried {
				if readKeys[key] {
					continue
				}
				if _, ok := allowed[key]; ok {
					continue
				}
				missing = append(missing, key)
			}
			if len(missing) > 0 {
				sort.Strings(missing)
				t.Errorf(
					"finalizer %q carries key(s) never rendered and not allow-listed: %s\n"+
						"Either render it on that screen, or add {\"key\": ..., \"finalizer\": %q, \"reason\": ...} to testdata/rendered_field_allowlist.json.",
					f.Name, strings.Join(missing, ", "), f.Name,
				)
			}
		})
	}
}

// renderedFieldAllowlistEntryID is the shrink-only ratchet's comparability
// key: (finalizer, key) together, not "key" alone -- the same key name
// legitimately recurs across different finalizers with different reasons
// (e.g. "next" is allow-listed separately for "plan completed" and
// "continue advanced"), and a key moving to a NEW finalizer it was not
// previously exempted for is exactly the kind of addition this ratchet
// exists to catch.
func renderedFieldAllowlistEntryID(e renderedFieldAllowlistEntry) string {
	return e.Finalizer + "|" + e.Key
}

// TestRenderedFieldAllowlistOnlyShrinks is 198-09's shrink-only ratchet
// (Task 2), mirroring TestOrphanAllowlistOnlyShrinks's structure and failure
// messages (cmd/subcommand_reachability_ratchet_test.go) so the pattern is
// recognisable: a pure set-membership diff against the committed baseline.
// Entries may be removed from the live list freely as they get rendered;
// adding one requires editing the baseline too, the explicit reviewed act
// this ratchet exists to force.
func TestRenderedFieldAllowlistOnlyShrinks(t *testing.T) {
	live := loadRenderedFieldAllowlist(t, "testdata/rendered_field_allowlist.json")
	baseline := loadRenderedFieldAllowlist(t, "testdata/rendered_field_allowlist_baseline.json")

	baselineIDs := map[string]bool{}
	for _, e := range baseline {
		baselineIDs[renderedFieldAllowlistEntryID(e)] = true
	}

	var added []string
	for _, e := range live {
		if !baselineIDs[renderedFieldAllowlistEntryID(e)] {
			added = append(added, fmt.Sprintf("%s (key %q)", e.Finalizer, e.Key))
		}
	}

	if len(added) > 0 {
		sort.Strings(added)
		t.Errorf("%d entr(y/ies) were added to the tolerated unrendered-field list without being added to the committed baseline: %s\n"+
			"The allowlist may only shrink. Render the field, or add the same entry to testdata/rendered_field_allowlist_baseline.json -- do not edit the baseline to make this pass without a reviewed reason.",
			len(added), strings.Join(added, ", "))
	}
}
