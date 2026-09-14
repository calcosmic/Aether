package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// sourceProposalTestRepo builds a real, throwaway git repository with one
// commit on branch "main" -- never the working repository this test process
// itself lives in. Every test in this file operates inside one of these.
func sourceProposalTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test")
	runGit(t, root, "checkout", "-b", "main")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("initial\n"), 0644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "initial")
	return root
}

// chdirTemp changes the process working directory to dir for the duration
// of the test, restoring the original directory on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
}

// sourceProposalCurrentBranch returns root's currently checked-out branch.
func sourceProposalCurrentBranch(t *testing.T, root string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v\n%s", err, out)
	}
	return strings.TrimSpace(string(out))
}

// sourceProposalBranchesWithPrefix lists every local branch in root carrying
// sourceProposalBranchPrefix.
func sourceProposalBranchesWithPrefix(t *testing.T, root string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", root, "branch", "--list", sourceProposalBranchPrefix+"*").CombinedOutput()
	if err != nil {
		t.Fatalf("list branches: %v\n%s", err, out)
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "* "))
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

// TestProposalCreatesABranchAndNothingElse asserts the branch exists, the
// original branch is checked out, and the working tree is as it was.
func TestProposalCreatesABranchAndNothingElse(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "a proposed change\n"}},
		Message: "propose an improvement",
	}
	result, created, err := proposeSourceImprovement("candidate-alpha", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}
	if !created {
		t.Fatal("expected created=true on a fresh proposal")
	}
	if result.VerificationState != sourceProposalStateProposed {
		t.Fatalf("VerificationState = %q, want %q", result.VerificationState, sourceProposalStateProposed)
	}
	if !sourceProposalBranchExists(root, result.Branch) {
		t.Fatalf("branch %q was not created", result.Branch)
	}
	if got := sourceProposalCurrentBranch(t, root); got != "main" {
		t.Fatalf("current branch = %q, want main (original branch restored)", got)
	}
	dirty, err := sourceProposalWorkingTreeIsDirty(root)
	if err != nil {
		t.Fatalf("check dirty: %v", err)
	}
	if dirty {
		t.Fatal("working tree is dirty after a successful proposal, want clean")
	}
	if _, err := os.Stat(filepath.Join(root, "improvement.txt")); err == nil {
		t.Fatal("improvement.txt exists on the original branch's working tree -- the change leaked off its own branch")
	}
}

// TestDirtyTreeProposalIsRefusedByName asserts a proposal on a dirty
// working tree is refused by name and creates nothing.
func TestDirtyTreeProposalIsRefusedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	if err := os.WriteFile(filepath.Join(root, "uncommitted.txt"), []byte("wip\n"), 0644); err != nil {
		t.Fatalf("write uncommitted file: %v", err)
	}

	before := sourceProposalBranchesWithPrefix(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	_, created, err := proposeSourceImprovement("candidate-dirty", nil, changes)
	if err == nil {
		t.Fatal("expected an error on a dirty working tree, got nil")
	}
	if !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("error %q does not name the dirty working tree", err.Error())
	}
	if created {
		t.Fatal("expected created=false on a refused dirty-tree proposal")
	}

	after := sourceProposalBranchesWithPrefix(t, root)
	if len(after) != len(before) {
		t.Fatalf("branch count changed on a refused proposal: before=%v after=%v", before, after)
	}
}

// TestBranchNameCollisionIsRefused asserts a proposal whose branch name
// collides with an existing branch is refused by name rather than
// overwriting it.
func TestBranchNameCollisionIsRefused(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	collidingBranch := sourceProposalBranchName("candidate-collide")
	runGit(t, root, "branch", collidingBranch)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	_, created, err := proposeSourceImprovement("candidate-collide", nil, changes)
	if err == nil {
		t.Fatal("expected an error on a branch-name collision, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error %q does not name the collision", err.Error())
	}
	if created {
		t.Fatal("expected created=false on a refused collision")
	}
	if got := sourceProposalCurrentBranch(t, root); got != "main" {
		t.Fatalf("current branch = %q, want main (untouched by the refused proposal)", got)
	}
}

// TestProposalReplayCreatesNoSecondBranch asserts proposing the same
// improvement twice returns the first proposal and creates no second
// branch.
func TestProposalReplayCreatesNoSecondBranch(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "same content\n"}},
		Message: "propose an improvement",
	}
	first, created1, err := proposeSourceImprovement("candidate-replay", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("first proposeSourceImprovement: %v", err)
	}
	if !created1 {
		t.Fatal("expected created=true on the first call")
	}

	second, created2, err := proposeSourceImprovement("candidate-replay", []string{"evidence-1"}, changes)
	if err != nil {
		t.Fatalf("second proposeSourceImprovement: %v", err)
	}
	if created2 {
		t.Fatal("expected created=false on a replayed proposal")
	}
	if second.ProposalID != first.ProposalID {
		t.Fatalf("replay returned a different proposal: %q vs %q", second.ProposalID, first.ProposalID)
	}

	branches := sourceProposalBranchesWithPrefix(t, root)
	if len(branches) != 1 {
		t.Fatalf("expected exactly 1 proposal branch after a replay, got %d: %v", len(branches), branches)
	}
}

// TestUnverifiedProposalIsReportedNotReady asserts the readiness report
// names the outstanding verification for a proposal that has not been
// independently verified.
func TestUnverifiedProposalIsReportedNotReady(t *testing.T) {
	if len(sourceProposalVerificationStateVocabulary) != len(sourceProposalVerificationStateNames()) {
		t.Fatalf("verification-state const block (%d) and names slice (%d) have different lengths",
			len(sourceProposalVerificationStateVocabulary), len(sourceProposalVerificationStateNames()))
	}

	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-not-ready", nil, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}

	report := renderSourceProposalReadiness(proposal)
	if !strings.Contains(report, "not ready") {
		t.Fatalf("report %q does not say the proposal is not ready", report)
	}
	if !strings.Contains(report, string(sourceProposalStateProposed)) {
		t.Fatalf("report %q does not name the outstanding verification state %q", report, sourceProposalStateProposed)
	}
}

// TestProposalCannotVerifyItself asserts a proposal is never marked
// verified by the same identity that created it, and that recordIndependent
// Verification is the only function ever setting the verified state.
func TestProposalCannotVerifyItself(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-self-verify", nil, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}

	if err := recordIndependentVerification(proposal.ProposalID, "candidate-self-verify", "result-1"); err == nil {
		t.Fatal("expected an error when the proposal's own creator verifies it")
	} else if !strings.Contains(err.Error(), "cannot verify itself") {
		t.Fatalf("error %q does not name the self-verification refusal", err.Error())
	}

	unchanged, ok, err := readSourceProposal(proposal.ProposalID)
	if err != nil || !ok {
		t.Fatalf("read back proposal: ok=%v err=%v", ok, err)
	}
	if unchanged.VerificationState != sourceProposalStateProposed {
		t.Fatalf("VerificationState = %q after a refused self-verification, want unchanged %q",
			unchanged.VerificationState, sourceProposalStateProposed)
	}

	if err := recordIndependentVerification(proposal.ProposalID, "an-independent-verifier", "result-1"); err != nil {
		t.Fatalf("recordIndependentVerification from a different verifier: %v", err)
	}
	verified, ok, err := readSourceProposal(proposal.ProposalID)
	if err != nil || !ok {
		t.Fatalf("read back verified proposal: ok=%v err=%v", ok, err)
	}
	if verified.VerificationState != sourceProposalStateVerified {
		t.Fatalf("VerificationState = %q, want %q", verified.VerificationState, sourceProposalStateVerified)
	}
	if verified.IndependentVerificationID != "result-1" {
		t.Fatalf("IndependentVerificationID = %q, want %q", verified.IndependentVerificationID, "result-1")
	}
}

// TestProposalRecordNamesItsCandidateAndEvidence asserts the proposal
// record carries the candidate it came from and the evidence for it.
func TestProposalRecordNamesItsCandidateAndEvidence(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	root := sourceProposalTestRepo(t)
	chdirTemp(t, root)

	changes := sourceChangeSet{
		Files:   []sourceChangeSetFile{{Path: "improvement.txt", Content: "x\n"}},
		Message: "propose an improvement",
	}
	proposal, _, err := proposeSourceImprovement("candidate-naming", []string{"evidence-a", "evidence-b"}, changes)
	if err != nil {
		t.Fatalf("proposeSourceImprovement: %v", err)
	}
	if proposal.CandidateID != "candidate-naming" {
		t.Fatalf("CandidateID = %q, want %q", proposal.CandidateID, "candidate-naming")
	}
	if len(proposal.EvidenceIDs) != 2 || proposal.EvidenceIDs[0] != "evidence-a" || proposal.EvidenceIDs[1] != "evidence-b" {
		t.Fatalf("EvidenceIDs = %v, want [evidence-a evidence-b]", proposal.EvidenceIDs)
	}
	if proposal.Branch == "" {
		t.Fatal("Branch is empty")
	}
	if proposal.BaseCommit == "" {
		t.Fatal("BaseCommit is empty")
	}
}

// ---------------------------------------------------------------------
// Task 3: structural proof there is no path from proposing a change to
// merging, publishing or deploying it.
// ---------------------------------------------------------------------

// sourceProposalReachabilityEntryPoints names every function this checker
// walks from: the whole public surface cmd/source_proposal.go exposes. No
// command in this codebase invokes either one yet -- Task 2 declares the
// library functions but wires no cobra command -- so these two entry
// points ARE "the command that invokes it" in the only sense currently
// reachable: everything outside this file that could call into a
// proposal's lifecycle today calls one of these two.
var sourceProposalReachabilityEntryPoints = []string{
	"proposeSourceImprovement",
	"recordIndependentVerification",
}

// sourceProposalModulePath is this repository's own module path
// (go.mod's `module` line), used to resolve an import spec back to one of
// this checker's own indexed package directories.
const sourceProposalModulePath = "github.com/calcosmic/Aether"

// sourceProposalCallGraphFunc is one function declaration this checker
// indexed, keyed by (PackageDir, Name) rather than by bare name alone --
// this is a syntactic walk, not a type-checked one (matching this
// repository's existing structural-check convention, e.g.
// cmd/episode_ledger_test.go's episodeLedgerWriteViolationsInFile), and a
// bare-name-only index would let an unrelated function elsewhere in this
// large, multi-package tree that merely SHARES a common method name (e.g.
// "Run", satisfied by both os/exec's *exec.Cmd.Run and an unrelated
// package's own Run function) be mistaken for the thing actually called.
// Scoping by directory (this repository's own package boundary) before
// resolving a call keeps the walk honest: only a call this checker can
// trace to a REAL local declaration is ever expanded, and every other call
// site -- resolvable or not -- is still checked for a direct name match.
type sourceProposalCallGraphFunc struct {
	Name       string
	File       string
	PackageDir string
	Decl       *ast.FuncDecl
}

// sourceProposalCallGraphIndex is the full repository-wide index this
// checker walks: every function/method declaration, keyed by package
// directory and name, plus, per file, the import-alias -> package-directory
// table needed to resolve a package-qualified call (codex.Foo()) back to a
// real declaration.
type sourceProposalCallGraphIndex struct {
	byPkgAndName map[string]*sourceProposalCallGraphFunc
	fileImports  map[string]map[string]string
}

func sourceProposalCallGraphKey(pkgDir, name string) string {
	return pkgDir + "\x00" + name
}

// sourceProposalBuildCallGraphIndex walks every directory in dirs, parsing
// every non-test .go file and indexing every function/method declaration
// plus every file's own module-local import aliases.
func sourceProposalBuildCallGraphIndex(t *testing.T, dirs ...string) (*token.FileSet, *sourceProposalCallGraphIndex) {
	t.Helper()
	fset := token.NewFileSet()
	index := &sourceProposalCallGraphIndex{
		byPkgAndName: map[string]*sourceProposalCallGraphFunc{},
		fileImports:  map[string]map[string]string{},
	}
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	for _, dir := range dirs {
		walkErr := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				return fmt.Errorf("parse %s: %w", path, perr)
			}
			absPath, aerr := filepath.Abs(path)
			if aerr != nil {
				return aerr
			}
			pkgDir := filepath.Dir(absPath)

			imports := map[string]string{}
			for _, imp := range file.Imports {
				importPath, uerr := strconv.Unquote(imp.Path.Value)
				if uerr != nil || !strings.HasPrefix(importPath, sourceProposalModulePath) {
					continue
				}
				rel := strings.TrimPrefix(strings.TrimPrefix(importPath, sourceProposalModulePath), "/")
				resolvedDir := filepath.Join(repoRoot, filepath.FromSlash(rel))
				alias := filepath.Base(rel)
				if imp.Name != nil {
					alias = imp.Name.Name
				}
				if alias == "" || alias == "_" || alias == "." {
					continue
				}
				imports[alias] = resolvedDir
			}
			index.fileImports[absPath] = imports

			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				index.byPkgAndName[sourceProposalCallGraphKey(pkgDir, fn.Name.Name)] = &sourceProposalCallGraphFunc{
					Name: fn.Name.Name, File: absPath, PackageDir: pkgDir, Decl: fn,
				}
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", dir, walkErr)
		}
	}
	if len(index.byPkgAndName) == 0 {
		t.Fatal("fixture is broken: no function declarations were indexed")
	}
	return fset, index
}

// sourceProposalIsSubprocessConstruction reports whether call constructs a
// subprocess via exec.Command or exec.CommandContext.
func sourceProposalIsSubprocessConstruction(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	return pkgIdent.Name == "exec" && (sel.Sel.Name == "Command" || sel.Sel.Name == "CommandContext")
}

// sourceProposalMatchForbiddenFamily reports which forbidden operation
// family (if any) name names, matching case-insensitively as a substring --
// "gitMerge", "MergeBranch" and a literal "merge" subprocess argument all
// name the same "merge" family.
func sourceProposalMatchForbiddenFamily(name string, forbidden []string) string {
	lower := strings.ToLower(name)
	for _, family := range forbidden {
		if strings.Contains(lower, family) {
			return family
		}
	}
	return ""
}

// sourceProposalCallGraphViolations walks the call graph reachable from
// entryPoints (declared in entryPkgDir) through index and returns one
// violation string per reachable direct call or subprocess-literal
// argument naming a family in forbidden.
//
// Every call site is checked for a direct name match regardless of whether
// it can be resolved to a real declaration (an unresolvable call -- a
// value-method call like store.LoadJSON, or a call into the standard
// library -- is still checked, just never expanded further). A call IS
// expanded, and its own body walked, only when it resolves to a genuine
// declaration this index holds: an unqualified call resolves within the
// calling function's own package directory (the only place an unqualified
// Go call can legally resolve to), and a package-qualified call resolves
// through the calling file's own import-alias table into another indexed
// package directory. This is what keeps the walk from wandering into an
// unrelated function elsewhere in the tree that merely shares a name.
func sourceProposalCallGraphViolations(fset *token.FileSet, index *sourceProposalCallGraphIndex, forbidden []string, entryPoints []string, entryPkgDir string) []string {
	visited := map[string]bool{}
	var violations []string

	var walk func(fn *sourceProposalCallGraphFunc)
	walk = func(fn *sourceProposalCallGraphFunc) {
		key := sourceProposalCallGraphKey(fn.PackageDir, fn.Name)
		if visited[key] {
			return
		}
		visited[key] = true

		fileImports := index.fileImports[fn.File]

		ast.Inspect(fn.Decl.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			var calledName string
			var target *sourceProposalCallGraphFunc
			switch f := call.Fun.(type) {
			case *ast.Ident:
				calledName = f.Name
				if t, ok := index.byPkgAndName[sourceProposalCallGraphKey(fn.PackageDir, calledName)]; ok {
					target = t
				}
			case *ast.SelectorExpr:
				calledName = f.Sel.Name
				if xIdent, ok := f.X.(*ast.Ident); ok {
					if pkgDir, ok := fileImports[xIdent.Name]; ok {
						if t, ok2 := index.byPkgAndName[sourceProposalCallGraphKey(pkgDir, calledName)]; ok2 {
							target = t
						}
					}
				}
			}

			if calledName != "" {
				if family := sourceProposalMatchForbiddenFamily(calledName, forbidden); family != "" {
					violations = append(violations, fmt.Sprintf(
						"%s (%s:%d) directly invokes %q, naming the forbidden %s operation",
						fn.Name, fn.File, fset.Position(call.Pos()).Line, calledName, family,
					))
				}
			}
			if target != nil {
				walk(target)
			}

			if sourceProposalIsSubprocessConstruction(call) {
				for _, arg := range call.Args {
					lit, ok := arg.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						continue
					}
					value, err := strconv.Unquote(lit.Value)
					if err != nil {
						continue
					}
					if family := sourceProposalMatchForbiddenFamily(value, forbidden); family != "" {
						violations = append(violations, fmt.Sprintf(
							"%s (%s:%d) constructs a subprocess call naming %q, the forbidden %s operation",
							fn.Name, fn.File, fset.Position(call.Pos()).Line, value, family,
						))
					}
				}
			}
			return true
		})
	}
	for _, entryName := range entryPoints {
		if fn, ok := index.byPkgAndName[sourceProposalCallGraphKey(entryPkgDir, entryName)]; ok {
			walk(fn)
		}
	}
	return violations
}

// sourceProposalParseForbiddenOperationsFromSource parses
// sourceProposalForbiddenOperations' declaration directly out of
// cmd/source_proposal.go's source, resolving each element identifier
// through the sourceProposalForbiddenOperation const block's own string
// values -- never importing the Go value itself, so this checker's list
// cannot silently drift from what the source actually declares.
func sourceProposalParseForbiddenOperationsFromSource(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source_proposal.go", nil, 0)
	if err != nil {
		t.Fatalf("parse source_proposal.go: %v", err)
	}

	constValues := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeIdent, ok := vs.Type.(*ast.Ident)
			if !ok || typeIdent.Name != "sourceProposalForbiddenOperation" {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				constValues[name.Name] = value
			}
		}
	}
	if len(constValues) == 0 {
		t.Fatal("fixture is broken: no sourceProposalForbiddenOperation consts were parsed")
	}

	var result []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name.Name != "sourceProposalForbiddenOperations" {
					continue
				}
				if i >= len(vs.Values) {
					continue
				}
				comp, ok := vs.Values[i].(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, elt := range comp.Elts {
					ident, ok := elt.(*ast.Ident)
					if !ok {
						continue
					}
					if value, ok := constValues[ident.Name]; ok {
						result = append(result, value)
					}
				}
			}
		}
	}
	if len(result) == 0 {
		t.Fatal("fixture is broken: sourceProposalForbiddenOperations was not found or resolved to nothing")
	}
	return result
}

// TestSourceProposalCannotMergePublishOrDeploy walks the real call graph
// reachable from proposeSourceImprovement and recordIndependentVerification
// across every non-test file in cmd/ and pkg/, and fails naming any
// reachable function or subprocess-literal argument invoking a merge, push,
// publish or deploy operation.
func TestSourceProposalCannotMergePublishOrDeploy(t *testing.T) {
	forbidden := sourceProposalParseForbiddenOperationsFromSource(t)
	fset, index := sourceProposalBuildCallGraphIndex(t, ".", "../pkg")
	cmdDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("resolve cmd package directory: %v", err)
	}

	violations := sourceProposalCallGraphViolations(fset, index, forbidden, sourceProposalReachabilityEntryPoints, cmdDir)
	if len(violations) != 0 {
		t.Fatalf("found a path from a source proposal to a forbidden operation:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("synthetic fixtures prove the detector sees all four families", func(t *testing.T) {
		cases := []struct {
			name   string
			src    string
			family string
			fn     string
		}{
			{
				name: "merge",
				src: `package cmd

func proposeSourceImprovementFixtureMerge() {
	doGitMerge()
}

func doGitMerge() {}
`,
				family: "merge",
				fn:     "proposeSourceImprovementFixtureMerge",
			},
			{
				name: "push",
				src: `package cmd

import "os/exec"

func proposeSourceImprovementFixturePush() {
	exec.Command("git", "push", "origin", "main")
}
`,
				family: "push",
				fn:     "proposeSourceImprovementFixturePush",
			},
			{
				name: "publish",
				src: `package cmd

type releaseTool struct{}

func (r releaseTool) Publish() {}

func proposeSourceImprovementFixturePublish() {
	var r releaseTool
	r.Publish()
}
`,
				family: "publish",
				fn:     "proposeSourceImprovementFixturePublish",
			},
			{
				name: "deploy",
				src: `package cmd

import (
	"context"
	"os/exec"
)

func proposeSourceImprovementFixtureDeploy() {
	exec.CommandContext(context.Background(), "porter", "deploy")
}
`,
				family: "deploy",
				fn:     "proposeSourceImprovementFixtureDeploy",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				fset := token.NewFileSet()
				filename := "fixture_source_proposal_" + c.name + ".go"
				file, err := parser.ParseFile(fset, filename, c.src, 0)
				if err != nil {
					t.Fatalf("parse fixture source: %v", err)
				}
				const pkgDir = "fixture-package"
				index := &sourceProposalCallGraphIndex{
					byPkgAndName: map[string]*sourceProposalCallGraphFunc{},
					fileImports:  map[string]map[string]string{filename: {}},
				}
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil {
						continue
					}
					index.byPkgAndName[sourceProposalCallGraphKey(pkgDir, fn.Name.Name)] = &sourceProposalCallGraphFunc{
						Name: fn.Name.Name, File: filename, PackageDir: pkgDir, Decl: fn,
					}
				}
				violations := sourceProposalCallGraphViolations(fset, index, []string{"merge", "push", "publish", "deploy"}, []string{c.fn}, pkgDir)
				if len(violations) == 0 {
					t.Fatalf("detector failed to catch the synthetic %s violation", c.name)
				}
				found := false
				for _, v := range violations {
					if strings.Contains(v, c.fn) && strings.Contains(v, c.family) && strings.Contains(v, filename) {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("violation(s) %v do not name function %q, family %q and file %q together", violations, c.fn, c.family, filename)
				}
			})
		}
	})
}

// TestForbiddenOperationListIsReadFromTheSource asserts the list
// TestSourceProposalCannotMergePublishOrDeploy uses (parsed from source)
// matches the real declared sourceProposalForbiddenOperations slice
// member-for-member, so the two can never silently drift apart.
func TestForbiddenOperationListIsReadFromTheSource(t *testing.T) {
	parsed := sourceProposalParseForbiddenOperationsFromSource(t)

	declared := make([]string, 0, len(sourceProposalForbiddenOperations))
	for _, op := range sourceProposalForbiddenOperations {
		declared = append(declared, string(op))
	}

	if len(parsed) != len(declared) {
		t.Fatalf("parsed forbidden list has %d members, declared slice has %d: parsed=%v declared=%v",
			len(parsed), len(declared), parsed, declared)
	}
	for i := range declared {
		if parsed[i] != declared[i] {
			t.Fatalf("member %d differs: parsed=%q declared=%q", i, parsed[i], declared[i])
		}
	}
}
