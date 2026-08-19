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

// This file replaces two Phase 187 ratchet tests
// (TestGCNeverCallsRemoveGitWorktree and TestWorktreeReapHasNoLifecycleCaller,
// both formerly in cmd/worktree_crash_safety_test.go) with a single guard
// that asserts the real safety property instead of checking for the
// presence of two hardcoded names.
//
// The property, stated the way CLAUDE.md's Definition of Done requires a
// property to be stated -- as something a command can fail on, not as a
// section that merely needs to exist:
//
//   NO function reachable from an automatic lifecycle entry point (build,
//   continue, init, resume-colony, run/autopilot, build-finalize,
//   continue-finalize) may destroy a worktree -- call removeGitWorktree, run
//   `git worktree remove`, or run `git branch -D` -- without its own
//   enclosing function first calling worktreeDestructionSafety.
//
// The two tests it replaces were defeated in exactly the way this repo's
// Definition of Done warns against: TestGCNeverCallsRemoveGitWorktree only
// ever looked inside the one function named gcOrphanedWorktrees, and
// TestWorktreeReapHasNoLifecycleCaller only ever grepped for two literal
// strings ("runWorktreeReap", "worktree-reap") in a hardcoded file list. CR-05
// (.planning/phases/187-crash-safe-worktrees-ecosystem-neutrality/187-REVIEW.md)
// found a second destructive function, cleanupBuildWorktrees, that was
// unguarded, lived in cmd/codex_build_worktree.go (already on the hardcoded
// list) yet contained neither literal string, and was called on every build
// -- and both tests passed anyway.
//
// This guard instead:
//   1. Parses every non-test .go file under cmd/ with go/ast and finds every
//      call to removeGitWorktree, plus every direct `git worktree remove` /
//      `git branch -D` invocation constructed via os/exec, by walking the AST
//      rather than grepping for a name.
//   2. Builds a call graph from every top-level function in cmd/, seeded by
//      the RunE closures of Cobra commands whose Use string is one of the
//      known automatic lifecycle entry points, and computes which functions
//      are transitively reachable from those closures.
//   3. For every destruction site whose enclosing function is
//      lifecycle-reachable, requires that the SAME enclosing function also
//      calls worktreeDestructionSafety. A destructive helper under any new
//      name, in any file, added tomorrow and wired into build/continue/
//      init/resume/run, is caught by this the same way cleanupBuildWorktrees
//      would have been.
//   4. Separately asserts worktree-reap's own RunE (the one legitimate
//      operator-invoked destruction path) is NOT lifecycle-reachable at all.
//   5. Fails loudly -- never vacuously -- if it finds zero destruction sites,
//      zero entry points, zero reachable functions, or cannot parse a file.

// ---------------------------------------------------------------------------
// Step 1: parse cmd/ and index every top-level function + every destruction
// call site found inside each, plus every Cobra command's Use string and its
// RunE closure's call set.
// ---------------------------------------------------------------------------

// destructionSite is one call, inside one enclosing function, that can
// destroy a worktree or its branch.
type destructionSite struct {
	file        string // repo-relative, e.g. "cmd/codex_build_worktree.go"
	enclosingFn string // top-level function name containing the call
	line        int
	description string // human-readable, e.g. "removeGitWorktree(...)" or "git worktree remove"

	// safetyGated is true when this call site is lexically nested (at any
	// depth, following if/else-if chains, for-loops, ranges, switches, and
	// same-node closures) inside an `if` condition that references a
	// `.Safe` field -- i.e. the call is only reached after branching on a
	// worktreeDestructionSafety verdict, not merely somewhere in a function
	// that also happens to call worktreeDestructionSafety elsewhere.
	safetyGated bool
}

// cmdFuncGraph is the parsed call graph for every top-level function
// declared directly in cmd/*.go (excluding _test.go files), plus every
// destruction site found anywhere in those files (including inside var-block
// Cobra command RunE closures, which are not top-level funcs themselves but
// are attributed to a synthetic node keyed by the command's Use string).
type cmdFuncGraph struct {
	// calls maps a function/closure node name to the set of names it calls.
	// Top-level functions are keyed by their Go identifier (e.g.
	// "gcOrphanedWorktrees"). Cobra command RunE closures are keyed by
	// "cobra:<Use string>" (e.g. "cobra:build <phase>").
	calls map[string]map[string]bool

	// destructions maps a node name (same keying as calls) to every
	// destruction site found lexically inside it.
	destructions map[string][]destructionSite

	// cobraEntryNodes maps a command's Use string to its synthetic node
	// name, for every var X = &cobra.Command{...} declaration found.
	cobraEntryNodes map[string]string

	// identToCobraNode maps the Go variable identifier a Cobra command
	// literal was assigned to (e.g. "abandonCmd") to its synthetic node name
	// (e.g. "cobra:abandon"). cobraEntryNodes is keyed by the command's Use
	// string, which is what a human reads and what lifecycleEntryCommands
	// lists by name; identToCobraNode is keyed by the Go identifier, which is
	// what an AddCommand(...) call site actually references. Registration
	// derivation (Step 1.5 below) needs the latter: it walks every
	// `.AddCommand(...)` call and every `[]*cobra.Command{...}` slice literal
	// ranged into one, both of which pass variable identifiers, never Use
	// strings.
	identToCobraNode map[string]string

	// registeredIdents is the set of every Go identifier this scan found
	// passed to some `.AddCommand(...)` call, directly or via a
	// `[]*cobra.Command{...}` slice literal that is later ranged over into an
	// AddCommand call (the pattern cmd/clash.go, cmd/worktree.go,
	// cmd/autopilot.go, cmd/immune.go, cmd/queen.go, and cmd/swarm.go all
	// use). This is populated by registration-call scanning (Step 1.5), not
	// by the per-function call-graph walk (Step 1), since AddCommand
	// arguments are not calls to same-package top-level functions and would
	// never appear in g.calls.
	registeredIdents map[string]bool

	filesScanned int
	funcsIndexed int
	sitesFound   int
	entriesFound int
}

// buildCmdFuncGraph parses every non-test .go file directly under cmdDir and
// returns the call graph described above.
func buildCmdFuncGraph(cmdDir string) (*cmdFuncGraph, error) {
	g := &cmdFuncGraph{
		calls:            map[string]map[string]bool{},
		destructions:     map[string][]destructionSite{},
		cobraEntryNodes:  map[string]string{},
		identToCobraNode: map[string]string{},
		registeredIdents: map[string]bool{},
	}

	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read cmd dir: %w", err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
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
		g.filesScanned++
		relName := filepath.Join("cmd", name)
		indexFile(g, fset, file, relName)
		files = append(files, file)
	}

	if g.filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero .go files in %s", cmdDir)
	}

	// Step 1.5: registration-call scanning. Runs AFTER every file has been
	// indexed (so identToCobraNode is fully populated first) and walks every
	// file a second time looking for `.AddCommand(...)` calls and
	// `[]*cobra.Command{...}` slice literals, recording every identifier
	// argument found. This is what lets the guard derive its root set from
	// the actual command registrations instead of a hand-maintained list --
	// see indexRegistrationCalls' own doc comment.
	for _, file := range files {
		indexRegistrationCalls(g, file)
	}

	return g, nil
}

// indexFile walks one parsed file: every top-level FuncDecl becomes a call
// graph node, and every top-level `var X = &cobra.Command{...}` declaration
// with a RunE field becomes a second node keyed by "cobra:<Use>", so RunE
// closures (which are not FuncDecls) still participate in the graph.
func indexFile(g *cmdFuncGraph, fset *token.FileSet, file *ast.File, relName string) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body == nil || d.Recv != nil {
				// Skip methods (receiver funcs) -- none of the known
				// destructive call sites are methods, and including them
				// would require receiver-type resolution this guard does
				// not need.
				continue
			}
			name := d.Name.Name
			g.funcsIndexed++
			indexFuncBody(g, fset, relName, name, d.Body)

		case *ast.GenDecl:
			if d.Tok != token.VAR {
				continue
			}
			for _, spec := range d.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				// vs.Names and vs.Values are parallel slices for a
				// single-value `var X = expr` spec (the only shape every
				// Cobra command declaration in cmd/ uses -- confirmed by
				// this guard's own zero-entries fail-loud check, which
				// would catch a shape change silently producing no
				// entries). A multi-name spec (`var a, b = x, y`) still
				// pairs correctly index-for-index; anything shorter is
				// skipped rather than guessed at.
				for i, val := range vs.Values {
					var varName string
					if i < len(vs.Names) {
						varName = vs.Names[i].Name
					}
					indexCobraCommandLiteral(g, fset, relName, varName, val)
				}
			}
		}
	}
}

// indexCobraCommandLiteral recognizes `&cobra.Command{Use: "...", RunE: func(...) {...}, ...}`
// and, when found, indexes the RunE closure body under a synthetic node
// "cobra:<Use>" so it participates in the call graph as an entry point
// candidate. When varName is non-empty (the command was assigned to a
// top-level `var X = &cobra.Command{...}`), also records identToCobraNode[varName]
// so registration-call scanning (indexRegistrationCalls) can resolve an
// `AddCommand(X)` argument back to this same node.
func indexCobraCommandLiteral(g *cmdFuncGraph, fset *token.FileSet, relName, varName string, expr ast.Expr) {
	unary, ok := expr.(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return
	}
	comp, ok := unary.X.(*ast.CompositeLit)
	if !ok {
		return
	}
	// Confirm this is a cobra.Command literal (SelectorExpr Command on some
	// package alias) rather than an unrelated struct literal.
	sel, ok := comp.Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" {
		return
	}

	var use string
	var runEBody *ast.BlockStmt
	var runEIdent string // set when RunE references an existing named function, e.g. "RunE: runWorktreeReap"
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
			if lit, ok := kv.Value.(*ast.BasicLit); ok {
				use = strings.Trim(lit.Value, "\"")
			}
		case "RunE":
			switch v := kv.Value.(type) {
			case *ast.FuncLit:
				runEBody = v.Body
			case *ast.Ident:
				// RunE set to a plain identifier (e.g. `RunE: runWorktreeReap`)
				// referencing a top-level function declared elsewhere in
				// cmd/. Do not resolve its body here -- indexFile already
				// (or will already) index that function as its own node;
				// this entry point simply needs a call-graph edge to it, so
				// reachability flows through naturally regardless of scan
				// order.
				runEIdent = v.Name
			}
		}
	}
	if use == "" || (runEBody == nil && runEIdent == "") {
		return
	}

	// The Use string may carry positional args, e.g. "build <phase>" -- the
	// bare command name (first whitespace-delimited token) is what callers
	// and the lifecycle entry-point list key on.
	bareUse := strings.Fields(use)
	if len(bareUse) == 0 {
		return
	}
	node := "cobra:" + bareUse[0]
	g.cobraEntryNodes[bareUse[0]] = node
	if varName != "" {
		g.identToCobraNode[varName] = node
	}
	g.entriesFound++
	if runEBody != nil {
		indexFuncBody(g, fset, relName, node, runEBody)
		return
	}
	// runEIdent case: create the node with a single edge to the named
	// function, so BFS reachability (Step 2) walks into it exactly as if it
	// were a direct call.
	if g.calls[node] == nil {
		g.calls[node] = map[string]bool{}
	}
	g.calls[node][runEIdent] = true
}

// indexRegistrationCalls walks an entire parsed file looking for the two
// shapes this codebase uses to register a Cobra command on some parent
// command (rootCmd, or a sub-command like ceremonyCmd or clashCmd):
//
//  1. A direct call `<something>.AddCommand(arg1, arg2, ...)` -- every
//     identifier argument is recorded as registered.
//  2. A `[]*cobra.Command{ident1, ident2, ...}` slice literal -- recorded
//     unconditionally, since every such literal in this codebase exists to
//     be ranged over into an AddCommand call in the same init() function
//     (cmd/clash.go, cmd/worktree.go, cmd/autopilot.go, cmd/immune.go,
//     cmd/queen.go, cmd/swarm.go). This guard does not attempt to verify the
//     range->AddCommand data-flow itself -- doing so would require local
//     variable tracking this AST walk does not otherwise need -- so it
//     records the slice's contents directly, which is safe in the
//     conservative direction: a command identifier only ends up inside one
//     of these slice literals because a human put it there to be
//     registered, and worst case this treats an unregistered-but-listed
//     command as reachable-checkable when it was never actually wired in,
//     which can only ADD false coverage, never remove real coverage.
//
// This is what lets the guard's root set be derived from the actual command
// registrations (rootCmd.AddCommand(...) / Use: fields) instead of a
// hand-maintained list, per 187-08's design requirement: a command added
// tomorrow and registered either way is picked up here without anyone
// updating this file.
func indexRegistrationCalls(g *cmdFuncGraph, file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			sel, ok := node.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "AddCommand" {
				return true
			}
			for _, arg := range node.Args {
				if ident, ok := arg.(*ast.Ident); ok {
					g.registeredIdents[ident.Name] = true
				}
			}

		case *ast.CompositeLit:
			arrType, ok := node.Type.(*ast.ArrayType)
			if !ok {
				return true
			}
			starExpr, ok := arrType.Elt.(*ast.StarExpr)
			if !ok {
				return true
			}
			sel, ok := starExpr.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Command" {
				return true
			}
			for _, elt := range node.Elts {
				if ident, ok := elt.(*ast.Ident); ok {
					g.registeredIdents[ident.Name] = true
				}
			}
		}
		return true
	})
}

// registeredCommandNodes resolves g.registeredIdents (the Go identifiers
// found passed to some AddCommand call or listed in a
// []*cobra.Command{...} registration slice) into the synthetic cobra: node
// names BFS reachability needs, via identToCobraNode. An identifier that
// never resolves (e.g. a command variable that exists in source but is
// truly never registered anywhere) is silently dropped here rather than
// treated as an entry point -- it cannot be, by definition, since nothing
// calls AddCommand on it.
func registeredCommandNodes(g *cmdFuncGraph) []string {
	var nodes []string
	for ident := range g.registeredIdents {
		if node, ok := g.identToCobraNode[ident]; ok {
			nodes = append(nodes, node)
		}
	}
	sort.Strings(nodes)
	return nodes
}

// indexFuncBody walks a function or closure body, recording every call it
// makes (by callee identifier name, for both plain calls and destruction
// detection) and every destruction site found lexically inside it. Nested
// FuncLits (closures passed as arguments, e.g. to
// store.UpdateJSONAtomically) are walked as part of the SAME enclosing node
// -- a destructive call made from inside a closure passed to
// UpdateJSONAtomically is still lexically "inside" cleanupBuildWorktrees for
// the purpose of this guard, matching how a human reader would attribute it.
//
// It also tracks, per destruction site, whether the call is lexically
// enclosed (at any nesting depth, following `if`/`else if` chains) by a
// condition that tests a variable's `.Safe` field -- i.e. whether the
// destructive call is actually gated on the safety verdict, not merely
// whether worktreeDestructionSafety was called SOMEWHERE in the function.
// This distinction is the whole point of the guard: a function can call
// worktreeDestructionSafety and then destroy unconditionally anyway (exactly
// CR-05's injected-defect shape used in this file's own proof), and a
// "calls the gate function somewhere" check cannot tell the two apart.
func indexFuncBody(g *cmdFuncGraph, fset *token.FileSet, relFile, nodeName string, body *ast.BlockStmt) {
	if g.calls[nodeName] == nil {
		g.calls[nodeName] = map[string]bool{}
	}
	sc := &safetyGateContext{
		verdictIdents:   findApprovedSafetyVerdictIdents(body),
		aggregateIdents: findApprovedUnsafeAggregateIdents(body),
	}
	walkStmtsForDestruction(g, fset, relFile, nodeName, body.List, false, sc)
}

// safetyGateContext bundles the two kinds of identifier this guard trusts as
// a genuine `.Safe`-gating signal within one enclosing function:
//
//   - verdictIdents: identifiers assigned directly from an
//     approvedSafetyVerdictProducers call (worktreeDestructionSafety,
//     branchMergeSafety) -- gated by testing `<ident>.Safe` directly.
//   - aggregateIdents: identifiers built by the "collect every unsafe
//     verdict from a range loop" idiom (findApprovedUnsafeAggregateIdents)
//     -- gated by testing `len(<ident>) > 0`, `<ident> > 0`, or
//     `<ident> == 0`.
//
// Bundling these into one struct (rather than two more map[string]bool
// parameters threaded through every walker function) keeps the walker
// signatures stable as this guard's vocabulary of recognized gating shapes
// grows -- a third kind added later extends this struct, not every call
// site's parameter list.
type safetyGateContext struct {
	verdictIdents   map[string]bool
	aggregateIdents map[string]bool
}

// mergeIdentSets returns the union of two identifier sets, either of which
// may be nil.
func mergeIdentSets(a, b map[string]bool) map[string]bool {
	merged := map[string]bool{}
	for k := range a {
		merged[k] = true
	}
	for k := range b {
		merged[k] = true
	}
	return merged
}

// approvedSafetyVerdictProducers is the set of function names whose return
// value is trusted to represent a genuine worktreeDestructionSafety-family
// verdict. A `.Safe` selector only counts as gating this guard's destruction
// check when the variable it selects from was itself assigned, somewhere in
// the enclosing function, from a direct call to one of these -- not from a
// same-named local struct literal with a hardcoded field (BS-2,
// 187-VERIFICATION.md's second reproduced evasion: `fakeSafety := struct{
// Safe bool }{Safe: true}` satisfies a purely-syntactic `.Safe` match while
// gating nothing real).
var approvedSafetyVerdictProducers = map[string]bool{
	"worktreeDestructionSafety": true,
	"branchMergeSafety":         true,
}

// findApprovedSafetyVerdictIdents scans an entire function body (every
// nesting depth -- ifs, closures, loops, switches; this is a single
// whole-body pass done ONCE per function, not per statement, so it does not
// need to track control flow the way walkStmtsForDestruction does) and
// returns the set of identifier names that were assigned, anywhere in the
// body, directly from a call to an approvedSafetyVerdictProducers entry --
// e.g. `safety := worktreeDestructionSafety(root, entry)` or
// `safety = branchMergeSafety(root, branch)`.
//
// This is deliberately NOT full dataflow analysis: it does not track
// reassignment, does not follow the value through further indirection (e.g.
// `other := safety; other.Safe`), and does not distinguish shadowed
// identifiers with the same name in different scopes. It only needs to
// answer one narrower question well: "was SOME identifier with this name,
// anywhere in this function, ever assigned from a real producer call?" A
// same-named local declared as a struct literal alongside a real call (the
// exact BS-2 injected shape: `safety := branchMergeSafety(...)` immediately
// followed by `fakeSafety := struct{Safe bool}{Safe: true}`) still fails this
// check correctly, because "fakeSafety" itself was never assigned from an
// approved producer -- only "safety" was, and the injected code deliberately
// tests fakeSafety.Safe, not safety.Safe. Full dataflow tracking (SSA-level
// shadowing, reassignment-to-a-non-producer-value invalidating a prior
// binding) is more than this codebase's actual call sites need; every real
// gating condition in production either tests the same identifier the
// producer call assigned, or does not test .Safe at all.
func findApprovedSafetyVerdictIdents(body *ast.BlockStmt) map[string]bool {
	idents := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		var lhs []ast.Expr
		var rhs []ast.Expr
		switch s := n.(type) {
		case *ast.AssignStmt:
			lhs = s.Lhs
			rhs = s.Rhs
		default:
			return true
		}
		if len(lhs) != len(rhs) {
			// Multi-value assignment from a single call (e.g. `a, err :=
			// f()`) -- a safety verdict producer here returns a single
			// value, not (value, error), so this shape cannot apply; skip
			// rather than guess.
			return true
		}
		for i, r := range rhs {
			call, ok := r.(*ast.CallExpr)
			if !ok {
				continue
			}
			fnIdent, ok := call.Fun.(*ast.Ident)
			if !ok || !approvedSafetyVerdictProducers[fnIdent.Name] {
				continue
			}
			if target, ok := lhs[i].(*ast.Ident); ok {
				idents[target.Name] = true
			}
		}
		return true
	})
	return idents
}

// approvedSafetyVerdictSliceProducers is the set of function names whose
// return value is a slice of verdicts, each individually derived from an
// approvedSafetyVerdictProducers call -- i.e. functions that themselves wrap
// the real safety check per-element rather than returning a single verdict.
// scanUnrecordedWorktrees (cmd/worktree_safety.go) is the one production
// example: it walks a directory and calls worktreeDestructionSafety once per
// entry found, returning []worktreeSafety. A range loop over its result that
// tests each element's `.Safe` field is exactly as trustworthy as a range
// loop over a literal []worktreeSafety built one worktreeDestructionSafety
// call at a time -- the indirection through one more function does not
// change what the loop variable actually is.
var approvedSafetyVerdictSliceProducers = map[string]bool{
	"scanUnrecordedWorktrees": true,
}

// findApprovedUnsafeAggregateIdents scans an entire function body for the
// "count/collect the unsafe verdicts" idiom every real GAP-3/GAP-5 fix in
// this codebase actually uses:
//
//	for _, v := range <approved verdict slice> {
//	    if !v.Safe {
//	        unsafe = append(unsafe, v)   // or: counter++ / counter += n
//	    }
//	}
//	if len(unsafe) > 0 { ...preserve, do not destroy... }
//
// It returns the set of identifier names that were built this way -- an
// accumulator (slice or int) that only grows when a loop variable ranging
// over an approved-producer's (or approved-slice-producer's) result tested
// `!v.Safe` first. condReferencesSafe treats `len(accum) > 0`, `accum > 0`,
// and `accum == 0` as equivalent to a direct `.Safe` selector test for any
// identifier in this set -- see condReferencesSafe's own doc comment for why
// this is the right level of generality (a second, narrow, named rule, not
// general dataflow analysis).
//
// This does NOT track arbitrary aggregation shapes -- only append-inside-a-
// range-with-a-negated-.Safe-guard, and only when the range's source is
// itself provably a slice of approved verdicts (a direct call to an
// approvedSafetyVerdictSliceProducers entry, or an identifier this same scan
// already resolved as such). A counter incremented from some other,
// unrelated condition is not recognized, and correctly so -- recognizing it
// would be exactly the kind of false-safe verdict this guard exists to
// refuse.
func findApprovedUnsafeAggregateIdents(body *ast.BlockStmt) map[string]bool {
	// First pass: identify which identifiers hold a []worktreeSafety-shaped
	// value from an approved slice producer, so a range over that identifier
	// (rather than the call expression directly) is still recognized.
	approvedSliceIdents := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i, r := range assign.Rhs {
			call, ok := r.(*ast.CallExpr)
			if !ok {
				continue
			}
			fnIdent, ok := call.Fun.(*ast.Ident)
			if !ok || !approvedSafetyVerdictSliceProducers[fnIdent.Name] {
				continue
			}
			if target, ok := assign.Lhs[i].(*ast.Ident); ok {
				approvedSliceIdents[target.Name] = true
			}
		}
		return true
	})

	isApprovedSliceExpr := func(e ast.Expr) bool {
		switch v := e.(type) {
		case *ast.Ident:
			return approvedSliceIdents[v.Name]
		case *ast.CallExpr:
			if fnIdent, ok := v.Fun.(*ast.Ident); ok {
				return approvedSafetyVerdictSliceProducers[fnIdent.Name]
			}
		}
		return false
	}

	aggregates := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok || !isApprovedSliceExpr(rng.X) {
			return true
		}
		loopVarName := ""
		if valueIdent, ok := rng.Value.(*ast.Ident); ok {
			loopVarName = valueIdent.Name
		}
		if loopVarName == "" {
			return true
		}
		// Within this range body, find `if !<loopVar>.Safe { ... }` guard
		// bodies and record every identifier assigned or incremented inside
		// them as an approved unsafe-aggregate.
		ast.Inspect(rng.Body, func(inner ast.Node) bool {
			ifs, ok := inner.(*ast.IfStmt)
			if !ok {
				return true
			}
			unary, ok := ifs.Cond.(*ast.UnaryExpr)
			if !ok || unary.Op != token.NOT {
				return true
			}
			sel, ok := unary.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Safe" {
				return true
			}
			base, ok := sel.X.(*ast.Ident)
			if !ok || base.Name != loopVarName {
				return true
			}
			// This is `if !<loopVar>.Safe { ... }` ranging over an approved
			// verdict slice -- every identifier assigned to (append) or
			// incremented (++, +=) directly inside this if-body is an
			// approved unsafe-aggregate.
			ast.Inspect(ifs.Body, func(bodyNode ast.Node) bool {
				switch bn := bodyNode.(type) {
				case *ast.AssignStmt:
					for i, r := range bn.Rhs {
						if call, ok := r.(*ast.CallExpr); ok {
							if fnIdent, ok := call.Fun.(*ast.Ident); ok && fnIdent.Name == "append" {
								if target, ok := bn.Lhs[i].(*ast.Ident); ok {
									aggregates[target.Name] = true
								}
							}
						}
					}
					if bn.Tok == token.ADD_ASSIGN {
						if target, ok := bn.Lhs[0].(*ast.Ident); ok {
							aggregates[target.Name] = true
						}
					}
				case *ast.IncDecStmt:
					if target, ok := bn.X.(*ast.Ident); ok {
						aggregates[target.Name] = true
					}
				}
				return true
			})
			return true
		})
		return true
	})

	// Propagation pass: `wtPreserved += len(unrecordedUnsafe)` (init_cmd.go's
	// actual shape) is a SECOND aggregation step outside the range loop
	// itself -- unrecordedUnsafe is already an approved aggregate after the
	// pass above, but the guard clause the code actually branches on
	// (`if wtPreserved == 0`) tests a DIFFERENT identifier that a later
	// statement folded it into. One additional pass recognizes `Y +=
	// len(X)` / `Y = Y + len(X)` / `Y += X` / `Y = X` where X is already a
	// known aggregate, and adds Y to the set too. This is still a single,
	// named, one-hop propagation -- not a fixed-point dataflow solver -- so
	// it is run exactly once; a chain of three or more foldings would not
	// be traced, which is fine, since no production call site in this
	// codebase needs more than one hop.
	ast.Inspect(body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		derivesFromAggregate := func(e ast.Expr) bool {
			switch v := e.(type) {
			case *ast.Ident:
				return aggregates[v.Name]
			case *ast.CallExpr:
				if fnIdent, ok := v.Fun.(*ast.Ident); ok && fnIdent.Name == "len" && len(v.Args) == 1 {
					if argIdent, ok := v.Args[0].(*ast.Ident); ok {
						return aggregates[argIdent.Name]
					}
				}
			}
			return false
		}
		if assign.Tok == token.ADD_ASSIGN && len(assign.Lhs) == 1 && len(assign.Rhs) == 1 {
			if target, ok := assign.Lhs[0].(*ast.Ident); ok && derivesFromAggregate(assign.Rhs[0]) {
				aggregates[target.Name] = true
			}
		}
		return true
	})
	return aggregates
}

// walkStmtsForDestruction walks a statement list in order, tracking
// safetyGated -- whether the statements being walked are nested inside an
// `if` (or `if`/`else if` chain) whose condition references a `.Safe`
// field, OR come lexically AFTER a guard clause of the shape `if
// <condition referencing .Safe> { ...; return/continue/break }` earlier in
// the SAME statement list. That second case is the idiom every production
// fix in this phase actually uses -- worktree-reap
// (`if safety.Safe { removeGitWorktree(...) ...continue }`,
// then further down `if !includeUnmerged { ...continue }` before a second
// removeGitWorktree), worktree-cleanup (`if !safety.Safe && !force {
// ...return nil }` followed by `if !safety.Safe && force { ... }` then an
// unconditional removeGitWorktree only reachable once both guard clauses
// have fallen through), worktree-merge-back (`if !safety.Safe { ...
// return nil }` followed by the destructive git commands), and
// recover_repair.go's orphan-branch case (`if !safety.Safe { ... return
// record }` followed by `git branch -D`). Treating only literal
// if/else-if-chain NESTING as "gated" and everything else as ungated
// would flag every one of those real, working preservation checks as a
// violation -- CR-05's own lesson (a check that only recognizes ONE
// syntactic shape misses the shapes it was never shown), applied to this
// guard's own detection logic rather than to production code.
//
// For each statement it visits every expression the statement directly
// contains (via recordCallsInExpr, which records call-graph edges and
// destruction sites but does NOT descend into nested IfStmts or FuncLits --
// those are re-entered here explicitly so safetyGated is tracked correctly
// for their own subtree).
func walkStmtsForDestruction(g *cmdFuncGraph, fset *token.FileSet, relFile, nodeName string, stmts []ast.Stmt, safetyGated bool, sc *safetyGateContext) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *ast.IfStmt:
			// A condition anywhere in this if/else-if chain that tests
			// `.Safe` gates every branch of the chain -- both branches are
			// reached only after the safety verdict was computed and
			// branched on, which is the shape gcOrphanedWorktrees and
			// cleanupBuildWorktrees (post-fix) both use: `if !safety.Safe {
			// preserve } ... ` and `if _, statErr := ...; statErr == nil {
			// defer to operator }`.
			gated := safetyGated || ifChainReferencesSafe(s, sc)
			if s.Init != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Init, safetyGated, sc)
			}
			if s.Cond != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Cond, safetyGated, sc)
			}
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, gated, sc)
			if s.Else != nil {
				switch e := s.Else.(type) {
				case *ast.BlockStmt:
					walkStmtsForDestruction(g, fset, relFile, nodeName, e.List, gated, sc)
				case *ast.IfStmt:
					walkStmtsForDestruction(g, fset, relFile, nodeName, []ast.Stmt{e}, gated, sc)
				}
			}

			// Guard-clause idiom: a `.Safe`-referencing if/else-if chain
			// with NO else branch, whose body unconditionally terminates
			// the enclosing statement list (return/continue/break on every
			// path through the chain's bodies), implicitly negates its
			// condition for every statement that follows it in THIS
			// statement list. Once true, safetyGated latches for the rest
			// of this walkStmtsForDestruction call (it is never un-set --
			// there is no shape in this codebase where a LATER statement
			// in the same list would need to un-gate).
			if s.Else == nil && ifChainReferencesSafe(s, sc) && ifChainAlwaysTerminates(s) {
				safetyGated = true
			}

		case *ast.BlockStmt:
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.List, safetyGated, sc)

		case *ast.ForStmt:
			if s.Init != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Init, safetyGated, sc)
			}
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, safetyGated, sc)

		case *ast.RangeStmt:
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, safetyGated, sc)

		case *ast.SwitchStmt:
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CaseClause); ok {
					walkStmtsForDestruction(g, fset, relFile, nodeName, cc.Body, safetyGated, sc)
				}
			}

		case *ast.SelectStmt:
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CommClause); ok {
					walkStmtsForDestruction(g, fset, relFile, nodeName, cc.Body, safetyGated, sc)
				}
			}

		default:
			recordCallsInExpr(g, fset, relFile, nodeName, stmt, safetyGated, sc)
		}
	}
}

// ifChainReferencesSafe reports whether the given IfStmt's condition, or any
// `else if` condition in the same chain, contains a selector expression
// ending in `.Safe` whose base identifier is a member of sc
// (see findApprovedSafetyVerdictIdents) -- i.e. a variable this function
// actually assigned from a real worktreeDestructionSafety/branchMergeSafety
// call, not merely any locally-declared value with a field named `Safe`.
func ifChainReferencesSafe(s *ast.IfStmt, sc *safetyGateContext) bool {
	if s.Cond != nil && condReferencesSafe(s.Cond, sc) {
		return true
	}
	if elseIf, ok := s.Else.(*ast.IfStmt); ok {
		return ifChainReferencesSafe(elseIf, sc)
	}
	return false
}

// condReferencesSafe reports whether cond contains EITHER:
//
//  1. A selector expression `X.Safe` where X is a bare identifier present in
//     sc.verdictIdents -- a variable this function actually assigned from a
//     real worktreeDestructionSafety/branchMergeSafety call, not merely any
//     locally-declared value with a field named `Safe`; or
//  2. A `len(X) > 0`, `X > 0`, or `X == 0` comparison where X is a bare
//     identifier present in sc.aggregateIdents -- the "collect every unsafe
//     verdict from a range loop, then gate on whether anything was
//     collected" idiom (findApprovedUnsafeAggregateIdents), which
//     cmd/init_cmd.go's `if wtPreserved == 0` and
//     cmd/entomb_cmd.go's `if len(unsafe) > 0 { ...; return nil }` both use.
//
// BS-2 (187-VERIFICATION.md's second reproduced evasion): before this fix,
// case 1 matched ANY `.Safe` selector syntactically, regardless of what it
// was selecting from. `fakeSafety := struct{ Safe bool }{Safe: true}; if
// !fakeSafety.Safe { ...destroy... }` satisfied that check even though
// fakeSafety was never a real safety verdict -- the destructive call ran
// unconditionally in practice while the guard read the condition as gated.
// Requiring the selector's base identifier to be one this function actually
// assigned from an approvedSafetyVerdictProducers call closes that evasion:
// fakeSafety is never in sc.verdictIdents (only a real producer-assigned
// identifier like `safety` would be), so a condition built entirely from it
// is correctly NOT recognized as gating.
//
// Case 2 exists because adding case 1 alone, with no equivalent for the
// aggregation idiom, made this stricter detector flag cmd/init_cmd.go:262
// and cmd/entomb_cmd.go:746 as false positives -- both are genuinely gated,
// just via a count/slice built across a loop rather than a single `.Safe`
// selector. Recognizing this ONE additional named shape (not general
// dataflow analysis -- see findApprovedUnsafeAggregateIdents's own doc
// comment for exactly what it does and does not track) keeps both real
// production sites correctly recognized as gated without adding either to
// a blanket function-level sanctioned-exception entry, which would have
// silenced ANY future ungated destruction added to those same functions,
// not just the one already-safe call site.
//
// This intentionally only inspects the selector's/comparison's immediate
// base identifier (`X` in `X.Safe` or `len(X)`), not arbitrary
// sub-expressions (`a.b.Safe`, `arr[i].Safe`) -- every real gating
// condition in this codebase's production fixes tests a single
// bare-identifier verdict or aggregate variable directly, so this covers
// every real shape without needing a general expression-provenance tracer.
func condReferencesSafe(cond ast.Expr, sc *safetyGateContext) bool {
	found := false
	ast.Inspect(cond, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.SelectorExpr:
			if expr.Sel.Name != "Safe" {
				return true
			}
			base, ok := expr.X.(*ast.Ident)
			if !ok {
				return true
			}
			if sc.verdictIdents[base.Name] {
				found = true
				return false
			}

		case *ast.BinaryExpr:
			if aggregateComparisonReferencesIdent(expr, sc.aggregateIdents) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// aggregateComparisonReferencesIdent reports whether the given BinaryExpr is
// one of the three comparison shapes findApprovedUnsafeAggregateIdents'
// identifiers are gated with in production: `len(X) > 0`, `X > 0`, or
// `X == 0`, where X is present in aggregateIdents. Operand order (`0 ==
// len(X)` vs `len(X) == 0`) is checked both ways since Go does not enforce
// either ordering.
func aggregateComparisonReferencesIdent(expr *ast.BinaryExpr, aggregateIdents map[string]bool) bool {
	if expr.Op != token.GTR && expr.Op != token.EQL {
		return false
	}
	identNamed := func(e ast.Expr) (string, bool) {
		switch v := e.(type) {
		case *ast.Ident:
			return v.Name, true
		case *ast.CallExpr:
			if fnIdent, ok := v.Fun.(*ast.Ident); ok && fnIdent.Name == "len" && len(v.Args) == 1 {
				if argIdent, ok := v.Args[0].(*ast.Ident); ok {
					return argIdent.Name, true
				}
			}
		}
		return "", false
	}
	isZeroLit := func(e ast.Expr) bool {
		lit, ok := e.(*ast.BasicLit)
		return ok && lit.Kind == token.INT && lit.Value == "0"
	}
	// `<ident-or-len> > 0` / `0 < <ident-or-len>` (BinaryExpr only stores the
	// operator as written, so only the GTR-with-zero-on-the-right and
	// EQL-either-side shapes need checking; Go's own idiom for these guard
	// clauses is always `X > 0` / `X == 0`, never `0 < X`).
	if name, ok := identNamed(expr.X); ok && aggregateIdents[name] {
		if (expr.Op == token.GTR || expr.Op == token.EQL) && isZeroLit(expr.Y) {
			return true
		}
	}
	if name, ok := identNamed(expr.Y); ok && aggregateIdents[name] {
		if expr.Op == token.EQL && isZeroLit(expr.X) {
			return true
		}
	}
	return false
}

// ifChainAlwaysTerminates reports whether an if/else-if chain with NO final
// else branch unconditionally terminates the enclosing statement list on
// every path THROUGH ITS OWN BODIES -- i.e. every `if`/`else if` body in the
// chain ends in a return/continue/break/goto, so control can only reach the
// statements AFTER this chain by having the chain's combined condition be
// false. This is what makes the guard-clause idiom
// (`if !safety.Safe { ...; return nil }`, or worktree-reap's
// `if safety.Safe { ...; continue }`) an implicit gate on everything that
// follows: reaching the next statement is only possible when none of the
// chain's conditions held.
//
// A chain WITH a final else is deliberately excluded (return false) even if
// every branch terminates -- that shape is already fully handled by
// walkStmtsForDestruction's direct recursion into both the if-body and the
// else-body with `gated` set on each, so it needs no fallthrough inference
// here; conflating the two would double-count the same gating decision two
// different ways for no benefit.
func ifChainAlwaysTerminates(s *ast.IfStmt) bool {
	if s.Else != nil {
		return false
	}
	if !blockAlwaysTerminates(s.Body) {
		return false
	}
	return true
}

// blockAlwaysTerminates reports whether the last statement in a block is a
// return, a branch statement (continue/break/goto/fallthrough), or an
// if/else-if/else chain where EVERY branch (including a final else, if
// present) itself always terminates. This intentionally does not attempt
// full control-flow analysis (no panic-call detection, no exhaustive switch
// analysis) -- it only needs to recognize the guard-clause shapes this
// codebase's own worktree-destruction fixes actually use, all of which end
// their guard body in a single return/continue statement.
func blockAlwaysTerminates(b *ast.BlockStmt) bool {
	if b == nil || len(b.List) == 0 {
		return false
	}
	last := b.List[len(b.List)-1]
	switch s := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.BranchStmt:
		// continue, break, goto, fallthrough -- any of these leaves the
		// current statement list without falling through to what follows.
		return true
	case *ast.IfStmt:
		if !blockAlwaysTerminates(s.Body) {
			return false
		}
		switch e := s.Else.(type) {
		case nil:
			return false // no else: there is a fallthrough path
		case *ast.BlockStmt:
			return blockAlwaysTerminates(e)
		case *ast.IfStmt:
			return blockAlwaysTerminates(&ast.BlockStmt{List: []ast.Stmt{e}})
		}
	}
	return false
}

// recordCallsInExpr inspects n (an expression or a non-IfStmt/FuncLit
// statement) for CallExprs. Every CallExpr found is recorded as a
// call-graph edge; a call to removeGitWorktree, a direct destructive git
// exec, or a directory-destroying os.RemoveAll/os.Rename targeting a
// worktree-bearing path is additionally recorded as a destructionSite
// tagged with the given safetyGated value. Nested IfStmts are deliberately
// NOT re-entered here -- walkStmtsForDestruction is the only place that
// descends into an IfStmt, so safetyGated is always tracked at that single
// point of truth. Nested FuncLits ARE re-entered here (under the same node
// name, per indexFuncBody's doc comment), since a FuncLit cannot itself be
// reached by walkStmtsForDestruction's statement-list walk -- it only ever
// appears inside an expression.
func recordCallsInExpr(g *cmdFuncGraph, fset *token.FileSet, relFile, nodeName string, n ast.Node, safetyGated bool, sc *safetyGateContext) {
	if n == nil {
		return
	}
	ast.Inspect(n, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if ifs, ok := node.(*ast.IfStmt); ok {
			walkStmtsForDestruction(g, fset, relFile, nodeName, []ast.Stmt{ifs}, safetyGated, sc)
			return false
		}
		if lit, ok := node.(*ast.FuncLit); ok {
			// A closure gets its OWN reaching-definition scan -- a safety
			// verdict assigned in the enclosing function is visible to a
			// closure that captures it (Go closures close over variables,
			// not just values), so idents found in either scope are valid.
			// findApprovedSafetyVerdictIdents/findApprovedUnsafeAggregateIdents
			// on the FuncLit's own body additionally catch a verdict or
			// aggregate built INSIDE the closure itself, which the
			// enclosing-function scan (run once, before any closure is
			// descended into) cannot see.
			litVerdictIdents := findApprovedSafetyVerdictIdents(lit.Body)
			litAggregateIdents := findApprovedUnsafeAggregateIdents(lit.Body)
			merged := sc
			if len(litVerdictIdents) > 0 || len(litAggregateIdents) > 0 {
				merged = &safetyGateContext{
					verdictIdents:   mergeIdentSets(sc.verdictIdents, litVerdictIdents),
					aggregateIdents: mergeIdentSets(sc.aggregateIdents, litAggregateIdents),
				}
			}
			walkStmtsForDestruction(g, fset, relFile, nodeName, lit.Body.List, safetyGated, merged)
			return false
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		switch fn := call.Fun.(type) {
		case *ast.Ident:
			g.calls[nodeName][fn.Name] = true
			if fn.Name == "removeGitWorktree" {
				pos := fset.Position(call.Pos())
				g.destructions[nodeName] = append(g.destructions[nodeName], destructionSite{
					file:        relFile,
					enclosingFn: nodeName,
					line:        pos.Line,
					description: "removeGitWorktree(...)",
					safetyGated: safetyGated,
				})
				g.sitesFound++
			}

		case *ast.SelectorExpr:
			// Record qualified calls too (e.g. pkg.Func(...)) by their
			// selector name, so a call graph edge like
			// "someHelper" -> "pkg.SomeFunc" doesn't silently vanish --
			// though only unqualified identifiers can ever match a
			// same-package top-level function name during graph closure.
			g.calls[nodeName][fn.Sel.Name] = true

			// Detect direct exec.Command / exec.CommandContext invocations
			// that pass "worktree" "remove" or "branch" "-D"/"-d" as
			// literal arguments -- the same destructive git operations
			// removeGitWorktree wraps, but invoked directly instead of
			// through it. This is what CR-05's own review flagged as a risk
			// worth generalizing against: "a new destructive helper added
			// tomorrow under any name must be found by this discovery, not
			// by having been listed."
			if isExecCommandCall(fn) {
				if desc, isDestructive := destructiveGitArgs(call.Args); isDestructive {
					pos := fset.Position(call.Pos())
					g.destructions[nodeName] = append(g.destructions[nodeName], destructionSite{
						file:        relFile,
						enclosingFn: nodeName,
						line:        pos.Line,
						description: desc,
						safetyGated: safetyGated,
					})
					g.sitesFound++
				}
			}

			// BS-1 (187-VERIFICATION.md's first reproduced evasion):
			// os.RemoveAll and os.Rename are just as destructive as
			// removeGitWorktree or a direct `git worktree remove` when their
			// target is a worktree-bearing path -- GAP-5's exact original
			// shape was an unconditional os.RemoveAll(worktreesDir), and
			// neither this detector nor removeGitWorktree's own git-command
			// wrapping ever saw it, because it goes around git entirely and
			// deletes the directory straight off disk. See
			// isWorktreePathDestruction's own doc comment for what pattern
			// is recognized and why.
			if isOSPackageCall(fn, "RemoveAll", "Rename") {
				if isWorktreePathDestruction(call.Args) {
					pos := fset.Position(call.Pos())
					g.destructions[nodeName] = append(g.destructions[nodeName], destructionSite{
						file:        relFile,
						enclosingFn: nodeName,
						line:        pos.Line,
						description: fmt.Sprintf("os.%s(...) targeting a worktree-bearing path", fn.Sel.Name),
						safetyGated: safetyGated,
					})
					g.sitesFound++
				}
			}
		}

		return true
	})
}

// isExecCommandCall reports whether a SelectorExpr is exec.Command or
// exec.CommandContext.
func isExecCommandCall(sel *ast.SelectorExpr) bool {
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "exec" {
		return false
	}
	return sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext"
}

// destructiveGitArgs inspects the string-literal arguments passed to an
// exec.Command(Context) call and reports whether they constitute a direct
// `git worktree remove` or `git branch -D`/`-d` invocation -- the same two
// destructive operations removeGitWorktree wraps. Only literal string
// arguments are inspected; a call built entirely from variables cannot be
// classified this way and is not flagged (the discovery in requirement 1 is
// about a NEW destructive helper being introduced under a new NAME, not
// about defeating string literal detection with obfuscation -- callers who
// go out of their way to construct args dynamically are not the class of
// regression this guard defends against, and removeGitWorktree itself is
// exempted below since it IS the sanctioned wrapper).
func destructiveGitArgs(args []ast.Expr) (string, bool) {
	var lits []string
	for _, arg := range args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			// exec.CommandContext's first two args are ctx and the binary
			// name; non-literal args (variables) break literal-only
			// detection, but "git" is always a literal in this codebase's
			// call sites, so a non-literal binary name simply yields no
			// match rather than a false positive.
			lits = append(lits, "")
			continue
		}
		lits = append(lits, strings.Trim(lit.Value, "\""))
	}
	joined := strings.Join(lits, " ")
	if !strings.Contains(joined, "git") {
		return "", false
	}
	if strings.Contains(joined, "worktree") && strings.Contains(joined, "remove") {
		return "git worktree remove (direct exec)", true
	}
	if strings.Contains(joined, "branch") && (strings.Contains(joined, "-D") || strings.Contains(joined, "-d")) {
		return "git branch -D/-d (direct exec)", true
	}
	return "", false
}

// isOSPackageCall reports whether a SelectorExpr is os.<name> for one of the
// given names (e.g. isOSPackageCall(fn, "RemoveAll", "Rename")).
func isOSPackageCall(sel *ast.SelectorExpr, names ...string) bool {
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "os" {
		return false
	}
	for _, name := range names {
		if sel.Sel.Name == name {
			return true
		}
	}
	return false
}

// worktreePathMarkers is the set of literal path segments and bare
// identifiers this codebase actually uses to name the worktrees directory --
// see cmd/worktree.go's own worktreeBaseDir constant (".aether/worktrees")
// and every worktreesDir local variable in cmd/entomb_cmd.go, cmd/abandon_cmd.go,
// and cmd/init_cmd.go, all of which are built as
// filepath.Join(<root>, ".aether", "worktrees") or reference worktreeBaseDir
// directly.
var worktreePathIdentMarkers = map[string]bool{
	"worktreesDir":    true,
	"worktreeBaseDir": true,
}

// isWorktreePathDestruction reports whether the first argument to an
// os.RemoveAll/os.Rename call is an expression that names a worktree-bearing
// path -- either:
//
//  1. A bare identifier matching a known worktree-directory variable/const
//     name (worktreePathIdentMarkers) -- e.g. `os.RemoveAll(worktreesDir)`.
//  2. A filepath.Join(...) call whose arguments include the string literal
//     "worktrees" -- e.g. `os.RemoveAll(filepath.Join(aetherRoot, ".aether",
//     "worktrees"))`, the shape every production call site in this codebase
//     actually uses.
//
// This is deliberately a conservative, legible heuristic rather than a
// dataflow tracer that resolves an identifier back to the string that built
// it (per this plan's own instruction: a false positive costs a human
// reviewer one look and, if genuine, a one-line sanctioned-exception entry
// with written justification; a false negative silently ships GAP-5 again).
// What it catches: any RemoveAll/Rename call whose argument expression, read
// as source text, names a worktree directory the way every real call site in
// this repo already does. What it does NOT catch: a path built through an
// intermediate variable with a name this list doesn't know
// (`p := someOtherName; os.RemoveAll(p)`), a path built by string
// concatenation instead of filepath.Join, or a path assembled dynamically at
// runtime in a way no static read of the source could resolve. Widen
// worktreePathIdentMarkers, or teach this function a new shape, the day a
// real call site needs it -- do not weaken this to silence a false positive
// on a site that is not actually worktree-related; move that site to a
// differently-named local instead, which documents the non-relationship
// better than a detector exception would.
func isWorktreePathDestruction(args []ast.Expr) bool {
	if len(args) == 0 {
		return false
	}
	return exprNamesWorktreePath(args[0])
}

func exprNamesWorktreePath(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		return worktreePathIdentMarkers[e.Name]

	case *ast.SelectorExpr:
		// A qualified reference to the shared constant, e.g. some future
		// `cmdpkg.WorktreeBaseDir` -- matched by its selector name alone,
		// the same conservative-by-name approach as the bare-identifier
		// case above.
		return worktreePathIdentMarkers[e.Sel.Name]

	case *ast.CallExpr:
		sel, ok := e.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok || pkgIdent.Name != "filepath" || sel.Sel.Name != "Join" {
			return false
		}
		for _, arg := range e.Args {
			if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				if strings.Trim(lit.Value, "\"") == "worktrees" {
					return true
				}
			}
			// A filepath.Join argument that is itself a worktree-path
			// expression (nested Join, or a worktreesDir/worktreeBaseDir
			// identifier passed as a segment) also counts -- this handles
			// filepath.Join(worktreesDir, extra) as well as the direct
			// literal-segment case.
			if exprNamesWorktreePath(arg) {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// ---------------------------------------------------------------------------
// Step 2: reachability from lifecycle entry points.
// ---------------------------------------------------------------------------

// lifecycleEntryCommands is the set of Cobra command names (the bare Use
// token) that represent an AUTOMATIC path -- something that runs without a
// human specifically choosing to destroy a worktree. This must stay in sync
// with the commands this repo documents as the lifecycle (see CLAUDE.md's
// "Autopilot" and colony workflow sections, and 187-CONTEXT.md's own list:
// resume, continue, init). worktree-reap and the standalone worktree-*
// commands (worktree-allocate, worktree-merge-back, worktree-create,
// worktree-cleanup, worktree-orphan-scan) are deliberately excluded: those
// require a human to type the exact destructive command, which is the D-01
// boundary this whole phase exists to protect.
//
// Being excluded from this list is NOT a claim that these commands are
// unguarded. 187-VERIFICATION.md (GAP-1, GAP-2) found that worktree-merge-back
// and worktree-cleanup previously destroyed dirty/unmerged work unconditionally
// -- being "operator-invoked" is not, on its own, a safety property. Both were
// fixed to call worktreeDestructionSafety internally before destroying,
// exactly like worktree-reap already did; see cmd/worktree.go's Step 5 and
// cmd/clash.go's worktreeCleanupCmd. This guard cannot see that internal
// gating (its BFS starts only from lifecycleEntryCommands), so it is not what
// proves these two commands are safe -- the real-git fail-then-pass tests in
// cmd/worktree_operator_destruction_test.go are.
var lifecycleEntryCommands = []string{
	"build",
	"continue",
	"init",
	"resume-colony",
	"run",
	"build-finalize",
	"continue-finalize",
}

// operatorOnlyDestructionCommands are the named, explicitly operator-invoked
// commands whose whole purpose is to destroy a worktree. worktree-reap must
// NEVER become lifecycle-reachable; this guard asserts that separately from
// the main property.
var operatorOnlyDestructionCommands = []string{"worktree-reap"}

// reachableFrom performs a BFS over g.calls starting at every node in
// startNodes and returns the set of every node name reached, INCLUDING the
// start nodes themselves.
func reachableFrom(g *cmdFuncGraph, startNodes []string) map[string]bool {
	visited := map[string]bool{}
	queue := append([]string{}, startNodes...)
	for _, s := range startNodes {
		visited[s] = true
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for callee := range g.calls[cur] {
			if visited[callee] {
				continue
			}
			// Only follow edges to names that are themselves indexed nodes
			// (i.e. same-package top-level functions or cobra: entry
			// nodes) -- calls to external packages, stdlib, etc. have no
			// further edges to walk and are simply dead ends, which is
			// correct: this guard only needs same-package reachability
			// since every destruction site and worktreeDestructionSafety
			// itself lives in package cmd.
			if _, ok := g.calls[callee]; !ok {
				continue
			}
			visited[callee] = true
			queue = append(queue, callee)
		}
	}
	return visited
}

// ---------------------------------------------------------------------------
// The guard itself.
// ---------------------------------------------------------------------------

// TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate is the
// property guard replacing TestGCNeverCallsRemoveGitWorktree and
// TestWorktreeReapHasNoLifecycleCaller. See the file-level doc comment for
// the full property statement and the CR-05 history motivating it.
func TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}

	// Requirement 4: fail loudly, never vacuously.
	if g.entriesFound == 0 {
		t.Fatalf("found zero Cobra command entry points while scanning %d files -- a guard that finds no entry points to check would pass vacuously forever", g.filesScanned)
	}
	if g.sitesFound == 0 {
		t.Fatalf("found zero worktree destruction sites (removeGitWorktree calls or direct `git worktree remove`/`git branch -D` invocations) across %d files and %d functions -- either the scan is broken or removeGitWorktree itself was renamed/removed; either way this must not silently pass", g.filesScanned, g.funcsIndexed)
	}

	// Resolve the lifecycle entry-point node names, and fail loudly if any
	// named lifecycle command could not be found -- this guard's whole
	// value depends on this list actually corresponding to real, live
	// Cobra commands in the source, not to names that used to exist.
	var startNodes []string
	var missingEntries []string
	for _, cmdName := range lifecycleEntryCommands {
		node, ok := g.cobraEntryNodes[cmdName]
		if !ok {
			missingEntries = append(missingEntries, cmdName)
			continue
		}
		startNodes = append(startNodes, node)
	}
	if len(missingEntries) > 0 {
		t.Fatalf("expected to find Cobra commands named %v among %d indexed entry points, but these were missing: %v -- update lifecycleEntryCommands or investigate why the command was not found (renamed Use string? moved file? this guard cannot verify a property against entry points it cannot locate)", lifecycleEntryCommands, g.entriesFound, missingEntries)
	}

	reachable := reachableFrom(g, startNodes)

	// Sanctioned exceptions: functions that ARE lifecycle-reachable and DO
	// call removeGitWorktree without a `.Safe`-gated condition wrapping the
	// call, but are not crash-recovery paths and are documented as safe by
	// worktree_reap.go's own doc comment -- finalizeBuildWorktree (runs only
	// after a successful worker's changes are already synced to root) and
	// allocateBuildWorktree's own rollback (removes a worktree it just
	// created in the same call, never registered as holding output).
	// removeGitWorktree ITSELF is also sanctioned here: it is the single
	// wrapper around the two literal git commands (`git worktree remove`,
	// `git branch -D`) that every other destruction site in this guard
	// already counts as a distinct site when calling removeGitWorktree by
	// name -- flagging removeGitWorktree's own internal git invocations
	// again would be double-counting the same destructive act once as
	// "calls removeGitWorktree" (at the real call site) and once as "runs
	// git worktree remove" (inside removeGitWorktree's own body). The
	// safety DECISION belongs to removeGitWorktree's callers, exactly as
	// its own doc comment states ("Every caller of this function reads a
	// non-nil error as..."); removeGitWorktree has no way to know why it
	// was called and cannot itself branch on a safety verdict it was never
	// given. This exception set is named explicitly so a change to any of
	// these three functions' reasoning still requires a human decision, not
	// a silent pass.
	//
	// Note (187-09): cmd/init_cmd.go's `os.RemoveAll(worktreesDir)` (gated
	// on `if wtPreserved == 0`) and cmd/entomb_cmd.go's
	// clearActiveColonyRuntimeFiles (gated on `if len(unsafe) > 0 { ...;
	// return nil }`) are NOT in this exception map -- they are real,
	// correctly-gated sites recognized directly by condReferencesSafe's
	// aggregate-comparison branch (see findApprovedUnsafeAggregateIdents),
	// not exempted by name. A blanket function-name exception here would
	// have silenced ANY future ungated destruction added to those
	// functions, not just this one already-safe call site -- see BS-1's
	// own fail-then-pass proof in 187-09-SUMMARY.md for why that
	// distinction matters in practice.
	sanctioned := map[string]bool{
		"finalizeBuildWorktree": true,
		"allocateBuildWorktree": true,
		"removeGitWorktree":     true,
	}

	// The main property: every destruction site whose enclosing function is
	// lifecycle-reachable must be gated on a `.Safe` condition -- i.e. the
	// destructive call must be nested inside an `if` (or chain) that
	// branches on a worktreeDestructionSafety verdict, not merely live in a
	// function that also happens to call worktreeDestructionSafety
	// somewhere else in its body (that weaker check is exactly what let
	// CR-05's shape through before this guard existed; see this test file's
	// own fail-then-pass proof in 187-06-SUMMARY.md).
	var violations []string
	for nodeName, sites := range g.destructions {
		if !reachable[nodeName] || sanctioned[nodeName] {
			continue
		}
		for _, site := range sites {
			if site.safetyGated {
				continue
			}
			violations = append(violations, fmt.Sprintf(
				"%s:%d — function %s is reachable from an automatic lifecycle path (build/continue/init/resume-colony/run/build-finalize/continue-finalize) and calls %s, but that call is not nested inside a condition that checks a worktreeDestructionSafety verdict's .Safe field. Either route this call through worktreeDestructionSafety + preserveWorktreeWork and branch on the result before destroying (see cmd/codex_build_worktree.go's gcOrphanedWorktrees for the pattern), or if this call is genuinely safe for a documented reason (e.g. it runs only after work is already synced elsewhere, like finalizeBuildWorktree), add it to the sanctioned-exception map in this test with the same reasoning worktree_reap.go's doc comment gives for its existing exceptions.",
				site.file, site.line, nodeName, site.description))
		}
	}
	sort.Strings(violations)

	if len(violations) > 0 {
		t.Errorf("found %d unguarded worktree destruction site(s) reachable from an automatic lifecycle path:\n%s",
			len(violations), strings.Join(violations, "\n"))
	}

	// The separate, explicit assertion: worktree-reap must NOT be
	// lifecycle-reachable at all.
	for _, opName := range operatorOnlyDestructionCommands {
		opNode, ok := g.cobraEntryNodes[opName]
		if !ok {
			t.Fatalf("expected to find the operator-invoked command %q among indexed entry points, but it was missing -- this guard cannot assert an isolation property against a command it cannot locate", opName)
		}
		if reachable[opNode] {
			t.Errorf("the operator-invoked destruction command %q is reachable from an automatic lifecycle path -- destruction must stay behind a human explicitly typing the command, never wired into build/continue/init/resume/run", opName)
		}
	}
}

// ---------------------------------------------------------------------------
// Step 3 (187-08): the extended property, covering every registered command.
// ---------------------------------------------------------------------------

// sanctionedUngatedDestructionFuncs is the set of functions allowed to
// perform worktree destruction without a `.Safe`-gated condition, checked by
// TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate below. This is a
// SEPARATE list from the narrower guard's `sanctioned` map above (same
// reasoning duplicated rather than shared so a change to one guard's
// exception set does not silently loosen the other), plus one more entry
// specific to the all-commands property:
//
//   - finalizeBuildWorktree, allocateBuildWorktree, removeGitWorktree: see
//     the narrower guard's `sanctioned` map above for the full reasoning:
//     each destroys only work already synced elsewhere, or is itself the
//     sanctioned low-level wrapper other sites are counted through.
//   - cobra:worktree-allocate: worktreeAllocateCmd's own RunE closure
//     (cmd/worktree.go) removes a worktree it JUST created moments earlier
//     in the SAME call, only when the immediately following
//     store.SaveJSON fails -- the identical shape allocateBuildWorktree's
//     rollback already covers above (never registered as holding worker
//     output, so there is nothing to preserve; the worktree did not exist
//     a few lines earlier in the same function invocation). This is a
//     distinct Go identifier from allocateBuildWorktree (a different
//     function, reached only via a different, operator-typed command) so
//     it needs its own entry, not a rename of the existing one.
//
// Note (187-09): cmd/init_cmd.go's `os.RemoveAll(worktreesDir)` (gated on
// `if wtPreserved == 0`) and cmd/entomb_cmd.go's clearActiveColonyRuntimeFiles
// (gated on `if len(unsafe) > 0 { ...; return nil }`) are deliberately NOT
// in this exception map, for the same reason given at the narrower guard's
// `sanctioned` map above: both are real, correctly-gated sites recognized
// directly by condReferencesSafe's aggregate-comparison branch (see
// findApprovedUnsafeAggregateIdents), not exempted by name. A blanket
// function-name exception would have silenced ANY future ungated
// destruction added to those functions, not just this one already-safe
// call site.
//
// Every other name in this list corresponds to a real fix landed in 187-08
// closing GAP-4/GAP-5, documented at its own call site with a `.Safe`-gated
// branch -- if a name here is destructive WITHOUT being safety-gated, that is
// a real regression this guard exists to catch, not something to add here to
// silence.
var sanctionedUngatedDestructionFuncs = map[string]bool{
	"finalizeBuildWorktree":   true,
	"allocateBuildWorktree":   true,
	"removeGitWorktree":       true,
	"cobra:worktree-allocate": true,
}

// TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate is the
// extension 187-VERIFICATION.md's GAP-4/GAP-5 remediation requires: the
// narrower guard above only ever starts its BFS from
// lifecycleEntryCommands, a hand-maintained list of seven automatic entry
// points. Every one of the five defects found after that guard was written
// (worktree-merge-back, worktree-cleanup, recover, abandon, entomb) lived in
// an operator-TYPED command -- a category the narrower guard does not police
// AT ALL, by design (D-01 says operator-typed destruction is legitimate, so
// the guard deliberately excludes it from the "automatic path" property).
// But "deliberately excluded from ONE property" is not the same claim as
// "verified safe by some OTHER property" -- and until this test, no
// structural guard existed to check operator-typed commands at all. Three
// successive review sweeps each found more, by hand, in files nobody had
// happened to open.
//
// This test asks a different, broader question than the narrow guard: of
// EVERY command actually registered on rootCmd (derived from real
// `.AddCommand(...)` calls and `[]*cobra.Command{...}` registration slices,
// not a hand-maintained list -- see registeredCommandNodes), does every
// worktree-destruction call site reachable from it sit behind a
// `.Safe`-gated condition? Operator-typed commands are explicitly WITHIN
// scope here, unlike the narrower guard -- worktree-reap, worktree-cleanup,
// worktree-merge-back, recover, and abandon must all pass this property,
// because each one destroys a worktree via a call site that MUST be gated
// (worktree-reap's own `if safety.Safe` branch, worktree-cleanup's
// `!safety.Safe && !force` refusal, worktree-merge-back's Step 5,
// recover_repair.go's `if !safety.Safe` orphan-branch preservation). A
// command whose entire job is "destroy without checking" would have no
// legitimate reason to exist un-gated; the one command that IS allowed to
// destroy freely (worktree-reap when safety.Safe is literally true) still
// reaches that decision via a `.Safe`-gated branch, so it satisfies this
// property too -- see this test's own assertion below.
func TestNoRegisteredCommandDestroysWorktreeWithoutSafetyGate(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}

	// Requirement: fail loudly, never vacuously -- same discipline as the
	// narrower guard, applied to the new root-set derivation too.
	if g.entriesFound == 0 {
		t.Fatalf("found zero Cobra command entry points while scanning %d files -- a guard that finds no entry points to check would pass vacuously forever", g.filesScanned)
	}
	if g.sitesFound == 0 {
		t.Fatalf("found zero worktree destruction sites across %d files and %d functions -- either the scan is broken or removeGitWorktree itself was renamed/removed; either way this must not silently pass", g.filesScanned, g.funcsIndexed)
	}
	if len(g.registeredIdents) == 0 {
		t.Fatalf("found zero identifiers passed to any .AddCommand(...) call or listed in a []*cobra.Command{...} registration slice across %d files -- either every command registration in cmd/ was removed (implausible) or indexRegistrationCalls stopped matching the AST shapes this codebase actually uses; a guard that derives zero registered commands would silently check nothing and still pass", g.filesScanned)
	}

	startNodes := registeredCommandNodes(g)
	if len(startNodes) == 0 {
		t.Fatalf("resolved zero registered command nodes from %d registered identifiers -- identToCobraNode could not match any of them back to an indexed Cobra command literal; this guard cannot derive a root set it cannot resolve", len(g.registeredIdents))
	}

	// Every entry this repo's lifecycle guard already trusts must also
	// appear here -- the all-commands root set must be a superset of the
	// automatic-lifecycle root set, never a narrower one. If it is not, the
	// derivation itself is broken (a registered command silently failed to
	// resolve), which would make this test's whole reason for existing --
	// catching MORE than the narrow guard, never less -- false.
	startNodeSet := map[string]bool{}
	for _, n := range startNodes {
		startNodeSet[n] = true
	}
	for _, cmdName := range lifecycleEntryCommands {
		node, ok := g.cobraEntryNodes[cmdName]
		if !ok {
			continue // already asserted as fatal by the narrower guard above
		}
		if !startNodeSet[node] {
			t.Fatalf("lifecycle command %q (node %q) is not present in the all-commands root set derived from actual registrations -- the registration-derived root set must be a superset of lifecycleEntryCommands, or this guard's coverage claim (it checks strictly MORE than the narrow guard) does not hold. Check indexRegistrationCalls: is %q's variable actually passed to some .AddCommand(...) call in source?", cmdName, node, cmdName)
		}
	}

	reachable := reachableFrom(g, startNodes)

	var violations []string
	for nodeName, sites := range g.destructions {
		if !reachable[nodeName] || sanctionedUngatedDestructionFuncs[nodeName] {
			continue
		}
		for _, site := range sites {
			if site.safetyGated {
				continue
			}
			violations = append(violations, fmt.Sprintf(
				"%s:%d — function %s is reachable from a registered Aether command and calls %s, but that call is not nested inside a condition that checks a worktreeDestructionSafety verdict's .Safe field. This property covers EVERY registered command, not just the automatic lifecycle subset -- operator-typed commands (recover, abandon, entomb, worktree-cleanup, worktree-merge-back, worktree-reap, and any command added in the future) are all in scope. Either route this call through worktreeDestructionSafety (or the branch-only branchMergeSafety variant, for a bare branch with no worktree directory) and branch on the result before destroying, or if this call is genuinely safe for a documented reason, add it to sanctionedUngatedDestructionFuncs in this test with the same reasoning worktree_reap.go's doc comment gives for its existing exceptions.",
				site.file, site.line, nodeName, site.description))
		}
	}
	sort.Strings(violations)

	if len(violations) > 0 {
		t.Errorf("found %d unguarded worktree destruction site(s) reachable from a registered command:\n%s",
			len(violations), strings.Join(violations, "\n"))
	}

	// worktree-reap remains legitimately destructive by design (D-01's
	// named, explicit, operator-invoked destruction command), but it must
	// satisfy the SAME property everything else does: every destructive call
	// it makes must sit behind a `.Safe`-gated branch. It is deliberately
	// NOT given a blanket exemption here the way the narrower guard exempts
	// it from reachability entirely -- runWorktreeReap's own destructive
	// call (removeGitWorktree inside `if safety.Safe { ... }`) is walked and
	// checked exactly like every other site, and passes because it already
	// is gated. This is the one place this test intentionally diverges from
	// the narrow guard's shape: rather than asserting worktree-reap is
	// UNREACHABLE (a property that would be false here on purpose --
	// worktree-reap. IS a registered command, so it IS reachable from this
	// root set), it asserts worktree-reap's destruction sites satisfy the
	// same safety-gating property as everything else, which is the actual
	// invariant D-01 requires.
	for _, opName := range operatorOnlyDestructionCommands {
		opNode, ok := g.cobraEntryNodes[opName]
		if !ok {
			t.Fatalf("expected to find the operator-invoked command %q among indexed entry points, but it was missing -- this guard cannot assert a property against a command it cannot locate", opName)
		}
		if !reachable[opNode] {
			t.Fatalf("expected %q to be reachable from the all-commands root set (it IS a registered command) -- if it is not, registeredCommandNodes failed to resolve it, which would make every assertion in this test about it meaningless", opName)
		}
		sites := g.destructions[opNode]
		for _, site := range sites {
			if !site.safetyGated {
				t.Errorf("%s:%d — %q's own destruction site is not `.Safe`-gated; this is the one command whose entire purpose is destruction, so if even IT does not gate on the safety verdict, nothing in this codebase reliably does", site.file, site.line, opName)
			}
		}
	}
}
