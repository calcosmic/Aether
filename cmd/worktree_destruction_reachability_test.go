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

	filesScanned int
	funcsIndexed int
	sitesFound   int
	entriesFound int
}

// buildCmdFuncGraph parses every non-test .go file directly under cmdDir and
// returns the call graph described above.
func buildCmdFuncGraph(cmdDir string) (*cmdFuncGraph, error) {
	g := &cmdFuncGraph{
		calls:           map[string]map[string]bool{},
		destructions:    map[string][]destructionSite{},
		cobraEntryNodes: map[string]string{},
	}

	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read cmd dir: %w", err)
	}

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
		g.filesScanned++
		relName := filepath.Join("cmd", name)
		indexFile(g, fset, file, relName)
	}

	if g.filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero .go files in %s", cmdDir)
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
				for _, val := range vs.Values {
					indexCobraCommandLiteral(g, fset, relName, val)
				}
			}
		}
	}
}

// indexCobraCommandLiteral recognizes `&cobra.Command{Use: "...", RunE: func(...) {...}, ...}`
// and, when found, indexes the RunE closure body under a synthetic node
// "cobra:<Use>" so it participates in the call graph as an entry point
// candidate.
func indexCobraCommandLiteral(g *cmdFuncGraph, fset *token.FileSet, relName string, expr ast.Expr) {
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
	walkStmtsForDestruction(g, fset, relFile, nodeName, body.List, false)
}

// walkStmtsForDestruction walks a statement list in order, tracking
// safetyGated -- whether the statements being walked are nested inside an
// `if` (or `if`/`else if` chain) whose condition references a `.Safe` field.
// For each statement it visits every expression the statement directly
// contains (via recordCallsInExpr, which records call-graph edges and
// destruction sites but does NOT descend into nested IfStmts or FuncLits --
// those are re-entered here explicitly so safetyGated is tracked correctly
// for their own subtree).
func walkStmtsForDestruction(g *cmdFuncGraph, fset *token.FileSet, relFile, nodeName string, stmts []ast.Stmt, safetyGated bool) {
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
			gated := safetyGated || ifChainReferencesSafe(s)
			if s.Init != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Init, safetyGated)
			}
			if s.Cond != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Cond, safetyGated)
			}
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, gated)
			if s.Else != nil {
				switch e := s.Else.(type) {
				case *ast.BlockStmt:
					walkStmtsForDestruction(g, fset, relFile, nodeName, e.List, gated)
				case *ast.IfStmt:
					walkStmtsForDestruction(g, fset, relFile, nodeName, []ast.Stmt{e}, gated)
				}
			}

		case *ast.BlockStmt:
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.List, safetyGated)

		case *ast.ForStmt:
			if s.Init != nil {
				recordCallsInExpr(g, fset, relFile, nodeName, s.Init, safetyGated)
			}
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, safetyGated)

		case *ast.RangeStmt:
			walkStmtsForDestruction(g, fset, relFile, nodeName, s.Body.List, safetyGated)

		case *ast.SwitchStmt:
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CaseClause); ok {
					walkStmtsForDestruction(g, fset, relFile, nodeName, cc.Body, safetyGated)
				}
			}

		case *ast.SelectStmt:
			for _, c := range s.Body.List {
				if cc, ok := c.(*ast.CommClause); ok {
					walkStmtsForDestruction(g, fset, relFile, nodeName, cc.Body, safetyGated)
				}
			}

		default:
			recordCallsInExpr(g, fset, relFile, nodeName, stmt, safetyGated)
		}
	}
}

// ifChainReferencesSafe reports whether the given IfStmt's condition, or any
// `else if` condition in the same chain, contains a selector expression
// ending in `.Safe`.
func ifChainReferencesSafe(s *ast.IfStmt) bool {
	if s.Cond != nil && condReferencesSafe(s.Cond) {
		return true
	}
	if elseIf, ok := s.Else.(*ast.IfStmt); ok {
		return ifChainReferencesSafe(elseIf)
	}
	return false
}

func condReferencesSafe(cond ast.Expr) bool {
	found := false
	ast.Inspect(cond, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "Safe" {
			found = true
			return false
		}
		return true
	})
	return found
}

// recordCallsInExpr inspects n (an expression or a non-IfStmt/FuncLit
// statement) for CallExprs. Every CallExpr found is recorded as a
// call-graph edge; a call to removeGitWorktree or a direct destructive git
// exec is additionally recorded as a destructionSite tagged with the given
// safetyGated value. Nested IfStmts are deliberately NOT re-entered here --
// walkStmtsForDestruction is the only place that descends into an IfStmt, so
// safetyGated is always tracked at that single point of truth. Nested
// FuncLits ARE re-entered here (under the same node name, per indexFuncBody's
// doc comment), since a FuncLit cannot itself be reached by
// walkStmtsForDestruction's statement-list walk -- it only ever appears
// inside an expression.
func recordCallsInExpr(g *cmdFuncGraph, fset *token.FileSet, relFile, nodeName string, n ast.Node, safetyGated bool) {
	if n == nil {
		return
	}
	ast.Inspect(n, func(node ast.Node) bool {
		if node == nil {
			return false
		}
		if ifs, ok := node.(*ast.IfStmt); ok {
			walkStmtsForDestruction(g, fset, relFile, nodeName, []ast.Stmt{ifs}, safetyGated)
			return false
		}
		if lit, ok := node.(*ast.FuncLit); ok {
			walkStmtsForDestruction(g, fset, relFile, nodeName, lit.Body.List, safetyGated)
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
