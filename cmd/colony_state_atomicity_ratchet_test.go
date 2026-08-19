package cmd

// T-188-10/T-188-11/T-188-12 (188-04): the COLONY_STATE.json atomicity
// ratchet.
//
// ROADMAP criterion 4 asked for "a grep-ratchet against non-atomic
// COLONY_STATE writes." This file is deliberately NOT a grep: it parses
// cmd/*.go with go/ast, the same structural discipline
// worktree_destruction_reachability_test.go and
// subcommand_reachability_ratchet_test.go already established in this
// codebase, following the model the planning brief named explicitly.
//
// D-07 (188-CONTEXT.md) scopes this deliberately narrow: converting the ~30
// pre-existing store.SaveJSON("COLONY_STATE.json", ...) /
// store.AtomicWrite("COLONY_STATE.json", ...) call sites across ~25 files to
// atomic writes is the ROADMAP's own deferred LOCK requirement category, not
// this phase's goal. Instead this file builds two independent checks:
//
//  1. TestColonyStateWriteAllowlistOnlyShrinks -- a shrink-only baseline
//     allowlist (testdata/colony_state_write_allowlist.json), regenerable
//     via -update-colony-state-write-allowlist, mirroring
//     subcommand_reachability_ratchet_test.go's -update-orphan-allowlist
//     idiom exactly. A genuinely new non-atomic write site fails it. A
//     stale allowlist entry (naming a site that no longer exists in source)
//     also fails it -- the allowlist cannot rot in the safe direction
//     either.
//  2. TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically -- a
//     zero-tolerance check with NO allowlist exemption possible for
//     anything reachable, by same-package call-graph BFS, from
//     advancePhase (cmd/advance_phase.go, 188-02's shared phase-advance
//     core). That one function is the actual subject of "one phase-advance
//     discipline," so it alone is held to the full standard.
//
// Both checks fail loudly, never vacuously, if their AST analysis finds
// zero sites, zero call-graph nodes, or a suspiciously small reachable set
// -- the exact discipline 187-VERIFICATION.md documents as necessary and
// which worktree_destruction_reachability_test.go and
// subcommand_reachability_ratchet_test.go both already apply.
//
// Lessons carried forward from 187-VERIFICATION.md's three proven, reproduced
// evasions of that sibling guard (read in full before writing this file):
//   - "A detector that only matches literal-argument calls misses writes
//     built from variables." This ratchet's literal-argument requirement
//     (the first argument must be the *ast.BasicLit "COLONY_STATE.json") is
//     accepted as a KNOWN, documented scope bound, not a blind spot: every
//     real call site in this codebase passes the path as a literal (grep-
//     confirmed in 188-CONTEXT.md's canonical_refs), and a caller that
//     builds the path from a variable would already be an unusual, reviewable
//     deviation from this codebase's own convention.
//   - "A detector with no os.RemoveAll-equivalent in its vocabulary misses
//     whole categories of the operation it claims to police." This ratchet's
//     equivalent risk was confirmed by direct inspection, not assumed: a
//     COLONY_STATE.json write can live inside an ordinary *ast.FuncDecl
//     (cmd/abandon_cmd.go's RunE: runAbandon, cmd/state_cmds.go's
//     executeFieldMode) OR inside an inline RunE: func(...) {...} closure
//     attached to a &cobra.Command{...} composite literal
//     (cmd/init_cmd.go:300, cmd/worktree.go:220) -- a shape with NO
//     enclosing *ast.FuncDecl at all. A scanner that only walked FuncDecl
//     bodies would silently miss every site of the second shape. Both
//     shapes are scanned (see scanColonyStateSource's doc comment).
//   - "A gating check satisfied by any textual mention of the right
//     identifier can be fooled by a same-named local." This ratchet has no
//     gating identifier to spoof: TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically
//     never inspects a caller-supplied "this is safe" signal, it structurally
//     recomputes reachability from advancePhase every run and consults no
//     allowlist at all for that set.
//   - "Entry-point discovery that only recognises one syntactic form misses
//     the others." scanColonyStateSource's cobra.Command detection matches
//     the composite literal ANYWHERE it appears in a file -- a top-level
//     `var X = &cobra.Command{...}` declaration or an inline literal such as
//     `AddCommand(&cobra.Command{...})` (cmd/host_cmd.go's pattern,
//     recorded by 187-VERIFICATION.md's Addendum as a real, still-open blind
//     spot in the sibling worktree guard's root-set derivation) -- rather
//     than only the var-declaration form.

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// updateColonyStateWriteAllowlist regenerates
// testdata/colony_state_write_allowlist.json from the scanner's real,
// honest output. Mirrors subcommand_reachability_ratchet_test.go's
// -update-orphan-allowlist flag exactly.
var updateColonyStateWriteAllowlist = flag.Bool("update-colony-state-write-allowlist", false, "regenerate testdata/colony_state_write_allowlist.json from the scanner's real output")

// colonyStateWriteSelectors is the set of *storage.Store method names this
// ratchet treats as a non-atomic write against "COLONY_STATE.json". Do not
// modify these primitives (pkg/storage/storage.go) -- this ratchet only
// detects calls against them.
//
//   - SaveJSON: plain load-then-save, non-atomic.
//   - AtomicWrite: a single write with no read-modify-write currency check
//     the way UpdateJSONAtomically's mutate closure has -- a caller using
//     AtomicWrite against COLONY_STATE.json has typically already read and
//     mutated a local copy with no re-validation against a concurrent
//     writer, so it counts as "non-atomic" for THIS ratchet's purposes
//     despite its name (see cmd/advance_phase.go's own doc comment and
//     188-04-PLAN.md's <interfaces> section).
//
// store.UpdateJSONAtomically is deliberately absent from this set: it is
// the safe primitive this ratchet does not flag.
var colonyStateWriteSelectors = map[string]bool{
	"SaveJSON":    true,
	"AtomicWrite": true,
}

// colonyStateWriteSite is one non-atomic write call against
// "COLONY_STATE.json", found structurally by parsing cmd/*.go with go/ast
// -- never by grepping source text for the primitive name.
type colonyStateWriteSite struct {
	File      string // repo-relative, e.g. "cmd/init_cmd.go"
	Function  string // enclosing function name; see scanColonyStateSource's doc comment for the two syntactic shapes this covers
	Primitive string // "SaveJSON" or "AtomicWrite"
}

// colonyStateWriteAllowlistEntry is one checked-in, accepted pre-existing
// non-atomic COLONY_STATE.json write site. D-08 (188-CONTEXT.md): keyed by
// (File, Function, Primitive), never by line number -- line numbers drift
// on any unrelated edit above the call site, but function names are stable
// identifiers within a file.
type colonyStateWriteAllowlistEntry struct {
	File      string `json:"file"`
	Function  string `json:"function"`
	Primitive string `json:"primitive"`
	Reason    string `json:"reason"`
}

// colonyStateScanResult is the combined output of one AST parse pass over
// cmd/*.go: every non-atomic COLONY_STATE.json write site
// (TestColonyStateWriteAllowlistOnlyShrinks's subject) and the same-package
// call graph those sites' enclosing functions participate in
// (TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically's subject).
// Both checks share this one scan so cmd/*.go is parsed once, not twice.
type colonyStateScanResult struct {
	sites []colonyStateWriteSite
	// calls maps an enclosing-function node name to the set of names its
	// body calls (by bare identifier, or by selector name for a qualified
	// or method call -- see recordColonyStateCallsAndSites). Only free
	// top-level functions (no receiver) and synthetic "cobra:<Use>" RunE
	// closure nodes become graph nodes with an entry here; a node NOT
	// present as a key is a dead end for BFS purposes, exactly like
	// worktree_destruction_reachability_test.go's reachableFrom.
	calls        map[string]map[string]bool
	filesScanned int
}

// scanColonyStateSource parses every non-test .go file directly under
// cmdDir and finds every call of the shape
// store.SaveJSON("COLONY_STATE.json", ...) or
// store.AtomicWrite("COLONY_STATE.json", ...), plus a conservative
// same-package call graph. The _test.go suffix exclusion below is also this
// ratchet's own self-exclusion: this file is itself named
// colony_state_atomicity_ratchet_test.go, so it is never scanned as
// production code by its own scanner.
//
// Two syntactic shapes are scanned, confirmed necessary by direct
// inspection of this repo's own known write sites (not assumed):
//
//  1. Every top-level *ast.FuncDecl -- both free functions and methods.
//     Only free functions (Recv == nil) become call-graph nodes (matching
//     worktree_destruction_reachability_test.go's own choice to exclude
//     methods from its graph, since none of the destructive call sites it
//     polices are methods and receiver-type resolution is unneeded
//     complexity here too); a method's body is still scanned for write
//     sites, only its outgoing edges are not recorded. This shape alone
//     covers a command whose RunE field names a top-level function
//     directly (cmd/abandon_cmd.go's `RunE: runAbandon`) as well as any
//     ordinary function or method.
//  2. Every `&cobra.Command{...}` composite literal found ANYWHERE in the
//     file -- a top-level `var X = &cobra.Command{...}` declaration, or an
//     inline literal such as `AddCommand(&cobra.Command{...})`
//     (cmd/host_cmd.go's pattern; 187-VERIFICATION.md's Addendum records
//     the inline form as a real, confirmed-open blind spot in the sibling
//     worktree-destruction guard's root-set derivation, so this scanner
//     does not repeat that specific miss) -- whose RunE field is an inline
//     `func(...) {...}` closure (cmd/init_cmd.go:300,
//     cmd/worktree.go:220's shape). These closures are not *ast.FuncDecl
//     nodes and are invisible to shape 1 alone. Attributed to a synthetic
//     function name "cobra:<Use string>", mirroring
//     worktree_destruction_reachability_test.go's own cobraEntryNodes
//     naming convention rather than inventing a new one.
//
// Known, accepted scope bound (documented rather than silently missed, the
// same discipline 187-VERIFICATION.md's WARNING-not-BLOCKER items use): a
// COLONY_STATE.json write inside a package-scope var initializer that is
// NEITHER a plain function/method call NOR a cobra.Command literal (e.g. a
// hypothetical package-scope IIFE) is not scanned by either shape above.
// This repo's own convention for one-time package-scope setup is a named
// init() function (itself an ordinary *ast.FuncDecl, fully covered by shape
// 1), and no real site in this codebase uses the IIFE shape -- confirmed by
// cross-referencing every site this scanner finds against 188-CONTEXT.md's
// canonical_refs grep list.
func scanColonyStateSource(cmdDir string) (*colonyStateScanResult, error) {
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read cmd dir: %w", err)
	}

	res := &colonyStateScanResult{calls: map[string]map[string]bool{}}
	fset := token.NewFileSet()

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(cmdDir, name)
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", name, readErr)
		}
		file, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", name, parseErr)
		}
		res.filesScanned++
		relName := filepath.Join("cmd", name)
		scanFileForColonyStateWrites(res, file, relName)
	}

	if res.filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero .go files in %s", cmdDir)
	}
	return res, nil
}

// scanFileForColonyStateWrites applies both shapes described in
// scanColonyStateSource's doc comment to one already-parsed file.
func scanFileForColonyStateWrites(res *colonyStateScanResult, file *ast.File, relName string) {
	// Shape 1: every top-level FuncDecl (free functions and methods).
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		fnName := fd.Name.Name
		isNode := fd.Recv == nil
		if isNode && res.calls[fnName] == nil {
			res.calls[fnName] = map[string]bool{}
		}
		recordColonyStateCallsAndSites(res, fd.Body, relName, fnName, isNode)
	}

	// Shape 2: every &cobra.Command{...} composite literal anywhere in the
	// file, var-declared or inline, whose RunE field is an inline closure.
	// A RunE field that instead names a top-level function directly (e.g.
	// `RunE: runAbandon`) needs no special handling here -- that function is
	// already indexed by shape 1 regardless of how it is referenced.
	ast.Inspect(file, func(n ast.Node) bool {
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
		if runE == nil {
			return true
		}

		fnName := "cobra:<unnamed>"
		if useVal != "" {
			fnName = "cobra:" + useVal
		}
		if res.calls[fnName] == nil {
			res.calls[fnName] = map[string]bool{}
		}
		recordColonyStateCallsAndSites(res, runE.Body, relName, fnName, true)
		return true
	})
}

// recordColonyStateCallsAndSites walks node (a function or closure body),
// recording every COLONY_STATE.json write site found and, when trackCalls
// is true, every outgoing call-graph edge from fnName. A nested
// *ast.FuncLit (e.g. the mutate callback passed to
// store.UpdateJSONAtomically, or any other inline closure) is walked as
// part of the SAME enclosing fnName by ast.Inspect's own recursion -- a
// write buried inside a closure is still this ratchet's business to
// attribute to its enclosing named function, not a separate anonymous
// node.
func recordColonyStateCallsAndSites(res *colonyStateScanResult, node ast.Node, relFile, fnName string, trackCalls bool) {
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if trackCalls {
				res.calls[fnName][fn.Name] = true
			}
		case *ast.SelectorExpr:
			if trackCalls {
				// Recorded by selector name alone, regardless of receiver --
				// a deliberately conservative reachability superset, not a
				// precise call graph (188-04-PLAN.md's <interfaces>
				// section). Same idiom
				// worktree_destruction_reachability_test.go's
				// recordCallsInExpr uses for g.calls[nodeName][fn.Sel.Name].
				res.calls[fnName][fn.Sel.Name] = true
			}
			if colonyStateWriteSelectors[fn.Sel.Name] && len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					if val, unquoteErr := strconv.Unquote(lit.Value); unquoteErr == nil && val == "COLONY_STATE.json" {
						res.sites = append(res.sites, colonyStateWriteSite{
							File:      relFile,
							Function:  fnName,
							Primitive: fn.Sel.Name,
						})
					}
				}
			}
		}
		return true
	})
}

// findColonyStateWriteSites is TestColonyStateWriteAllowlistOnlyShrinks's
// scanner entry point.
func findColonyStateWriteSites(cmdDir string) ([]colonyStateWriteSite, error) {
	res, err := scanColonyStateSource(cmdDir)
	if err != nil {
		return nil, err
	}
	return res.sites, nil
}

// siteAllowlistKey is the D-08 compound key: (File, Function, Primitive).
func siteAllowlistKey(file, function, primitive string) string {
	return file + "\x00" + function + "\x00" + primitive
}

// loadColonyStateWriteAllowlist reads a committed allowlist JSON file.
// Mirrors subcommand_reachability_ratchet_test.go's loadOrphanAllowlist.
func loadColonyStateWriteAllowlist(t *testing.T, path string) []colonyStateWriteAllowlistEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update-colony-state-write-allowlist to create cmd/testdata/colony_state_write_allowlist.json)", path, err)
	}
	var entries []colonyStateWriteAllowlistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

// writeColonyStateWriteAllowlist writes the scanner's real, honest output to
// testdata/colony_state_write_allowlist.json. Mirrors
// subcommand_reachability_ratchet_test.go's writeOrphanAllowlist.
//
// Deduplicates by the same (File, Function, Primitive) compound key
// TestColonyStateWriteAllowlistOnlyShrinks compares against (D-08: the
// allowlist is keyed by file+function, not by line number). Several real
// functions legitimately contain more than one COLONY_STATE.json write call
// at different line numbers -- e.g. runCodexPlanWithOptions has three -- and
// since two calls in the same function collapse to one key either way, a
// non-deduplicated file would carry repeated, identical-looking JSON blocks
// that read like a scanner bug rather than the multiple genuine call sites
// they represent.
func writeColonyStateWriteAllowlist(t *testing.T, sites []colonyStateWriteSite) int {
	t.Helper()
	seen := map[string]bool{}
	entries := make([]colonyStateWriteAllowlistEntry, 0, len(sites))
	for _, s := range sites {
		key := siteAllowlistKey(s.File, s.Function, s.Primitive)
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, colonyStateWriteAllowlistEntry{
			File:      s.File,
			Function:  s.Function,
			Primitive: s.Primitive,
			Reason:    "pre-existing, not migrated by phase 188 (see 188-CONTEXT.md D-07)",
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		if entries[i].Function != entries[j].Function {
			return entries[i].Function < entries[j].Function
		}
		return entries[i].Primitive < entries[j].Primitive
	})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal colony state write allowlist: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile("testdata/colony_state_write_allowlist.json", data, 0644); err != nil {
		t.Fatalf("write testdata/colony_state_write_allowlist.json: %v", err)
	}
	return len(entries)
}

// TestColonyStateWriteAllowlistOnlyShrinks is the shrink-only baseline
// ratchet (T-188-11): every non-atomic COLONY_STATE.json write site found
// by the AST scanner must already be recorded in the checked-in baseline
// testdata/colony_state_write_allowlist.json, and every baseline entry must
// still correspond to a real site in source -- so the file cannot rot stale
// in either direction.
func TestColonyStateWriteAllowlistOnlyShrinks(t *testing.T) {
	sites, err := findColonyStateWriteSites(".")
	if err != nil {
		t.Fatalf("scan cmd/ for COLONY_STATE.json write sites: %v", err)
	}

	// T-188-12: a scanner that finds zero sites has broken, not proven the
	// codebase atomic overnight. A guard that finds nothing must fail, not
	// pass.
	if len(sites) == 0 {
		t.Fatal("findColonyStateWriteSites found zero non-atomic COLONY_STATE.json write sites across cmd/*.go -- the AST walker likely broke (wrong selector names, wrong string literal match, wrong directory), not that every write site became atomic.")
	}

	if *updateColonyStateWriteAllowlist {
		written := writeColonyStateWriteAllowlist(t, sites)
		t.Logf("wrote testdata/colony_state_write_allowlist.json with %d unique (file, function, primitive) entries from %d raw scanned site(s)", written, len(sites))
		return
	}

	allowlist := loadColonyStateWriteAllowlist(t, "testdata/colony_state_write_allowlist.json")

	allowed := map[string]bool{}
	for _, e := range allowlist {
		allowed[siteAllowlistKey(e.File, e.Function, e.Primitive)] = true
	}
	found := map[string]bool{}
	for _, s := range sites {
		found[siteAllowlistKey(s.File, s.Function, s.Primitive)] = true
	}

	var newSites []string
	for _, s := range sites {
		if !allowed[siteAllowlistKey(s.File, s.Function, s.Primitive)] {
			newSites = append(newSites, fmt.Sprintf("%s:%s (%s)", s.File, s.Function, s.Primitive))
		}
	}
	var staleEntries []string
	for _, e := range allowlist {
		if !found[siteAllowlistKey(e.File, e.Function, e.Primitive)] {
			staleEntries = append(staleEntries, fmt.Sprintf("%s:%s (%s)", e.File, e.Function, e.Primitive))
		}
	}
	sort.Strings(newSites)
	sort.Strings(staleEntries)

	if len(newSites) > 0 {
		t.Errorf("%d new non-atomic COLONY_STATE.json write site(s) found that are not in testdata/colony_state_write_allowlist.json:\n  %s\nEither make the write atomic (store.UpdateJSONAtomically) or, if this is a genuinely reviewed exception, run with -update-colony-state-write-allowlist to regenerate the allowlist from the scanner's real output (never hand-edit the JSON file).",
			len(newSites), strings.Join(newSites, "\n  "))
	}
	if len(staleEntries) > 0 {
		t.Errorf("%d testdata/colony_state_write_allowlist.json entry(ies) no longer match any real write site in source:\n  %s\nThe file/function was removed or renamed, or the write became atomic -- run with -update-colony-state-write-allowlist to regenerate the allowlist so it always reflects the scanner's real output (never hand-edit the JSON file to drop a stale entry alone).",
			len(staleEntries), strings.Join(staleEntries, "\n  "))
	}
}

// colonyStateReachableFrom performs a BFS over calls starting at every node
// in startNodes and returns the set of every node name reached, including
// the start nodes themselves. Mirrors
// worktree_destruction_reachability_test.go's reachableFrom: an edge is
// only followed to a callee name that is itself an indexed node (a
// same-package function or synthetic cobra: node this scan recorded a body
// for) -- calls into another package, the standard library, or an
// unindexed method are dead ends, which is correct: advancePhase and every
// COLONY_STATE.json write primitive it or its callees could reach live in
// package cmd, so same-package reachability is exactly what this check
// needs.
func colonyStateReachableFrom(calls map[string]map[string]bool, startNodes []string) map[string]bool {
	visited := map[string]bool{}
	queue := append([]string{}, startNodes...)
	for _, s := range startNodes {
		visited[s] = true
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for callee := range calls[cur] {
			if visited[callee] {
				continue
			}
			if _, ok := calls[callee]; !ok {
				continue
			}
			visited[callee] = true
			queue = append(queue, callee)
		}
	}
	return visited
}

// TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically is the
// zero-tolerance guard (T-188-10): no allowlist exemption is consulted for
// anything reachable, by same-package call-graph BFS, from advancePhase
// (cmd/advance_phase.go, 188-02's shared phase-advance core) -- that one
// function is the actual subject of "one phase-advance discipline," so it
// alone is held to the full standard, structurally verified rather than
// asserted in prose.
func TestNoFunctionReachableFromAdvancePhaseWritesNonAtomically(t *testing.T) {
	res, err := scanColonyStateSource(".")
	if err != nil {
		t.Fatalf("scan cmd/ for COLONY_STATE.json write sites and call graph: %v", err)
	}

	// T-188-12, shared with TestColonyStateWriteAllowlistOnlyShrinks: a
	// scanner that finds zero sites has broken.
	if len(res.sites) == 0 {
		t.Fatal("scanColonyStateSource found zero non-atomic COLONY_STATE.json write sites across cmd/*.go -- the AST walker likely broke, not that the codebase became fully atomic overnight.")
	}

	if _, ok := res.calls["advancePhase"]; !ok {
		t.Fatal("advancePhase is not present in the call graph -- either cmd/advance_phase.go was not found under cmd/ or the function was renamed; this check has exactly one root (advancePhase, cmd/advance_phase.go, produced by 188-02) and cannot proceed without it")
	}

	reachable := colonyStateReachableFrom(res.calls, []string{"advancePhase"})

	// T-188-12: advancePhase is known to call validateRuntimeStateStillCurrent,
	// trimmedEvents, and store.UpdateJSONAtomically at minimum -- a reachable
	// set of size 1 (just advancePhase itself) means the call-graph walker
	// broke, not that advancePhase calls nothing. A guard that finds
	// nothing meaningful must fail, not pass.
	if len(reachable) <= 1 {
		names := make([]string, 0, len(reachable))
		for n := range reachable {
			names = append(names, n)
		}
		sort.Strings(names)
		t.Fatalf("reachability BFS rooted at advancePhase found only %d node(s) (%s) -- advancePhase is known to call same-package helpers (validateRuntimeStateStillCurrent, trimmedEvents) at minimum, so a reachable set this small means the call-graph walker likely broke, not that advancePhase calls nothing.",
			len(reachable), strings.Join(names, ", "))
	}

	// Zero-tolerance: the allowlist TestColonyStateWriteAllowlistOnlyShrinks
	// loads is never loaded or consulted anywhere in this test, by design --
	// every site whose enclosing function is in the advancePhase-reachable
	// set fails this test regardless of whether it is also recorded as an
	// accepted, pre-existing exception in
	// testdata/colony_state_write_allowlist.json.
	var violations []string
	for _, s := range res.sites {
		if reachable[s.Function] {
			violations = append(violations, fmt.Sprintf("%s:%s (%s)", s.File, s.Function, s.Primitive))
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("%d non-atomic COLONY_STATE.json write site(s) found inside a function reachable from advancePhase -- NO allowlist exemption is possible for this set, by design:\n  %s\nRoute this write through store.UpdateJSONAtomically instead.",
			len(violations), strings.Join(violations, "\n  "))
	}
}
