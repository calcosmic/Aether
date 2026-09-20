package shadow

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// shadowIsolationTargetTypes names the two types this file's structural
// scans protect: the grader (FrozenEvaluator) and the acceptance criteria
// it grades against (AcceptanceCriteria). Both must stay unreachable and
// unalterable from anything a candidate could touch.
var shadowIsolationTargetTypes = map[string]bool{
	"FrozenEvaluator":    true,
	"AcceptanceCriteria": true,
}

// shadowNonTestGoFiles returns every non-test .go file directly under dir.
func shadowNonTestGoFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	return files, nil
}

// TestCandidateCannotAlterItsEvaluatorDigest constructs a grader, captures
// its digest, then attempts every code path a candidate could reach: run
// it, copy the value, pass it through an interface, store it in a map and
// read it back, take its digest twice -- and asserts the digest is
// byte-identical at every point.
func TestCandidateCannotAlterItsEvaluatorDigest(t *testing.T) {
	evaluator := NewFrozenEvaluator([]byte("shadow-isolation-definition"), alwaysPassRun)
	original := evaluator.Digest()

	id, scope, expectedBenefit, harms, expiry, rollbackPlan := validCandidateFields()
	candidate, err := NewCandidate(id, scope, expectedBenefit, harms, expiry, rollbackPlan)
	if err != nil {
		t.Fatalf("unexpected refusal building a fixture candidate: %v", err)
	}
	task := NewTask("t-1")

	// Path 1: run it.
	evaluator.Run(candidate, task)
	if evaluator.Digest() != original {
		t.Fatalf("digest changed after Run: %x vs %x", evaluator.Digest(), original)
	}

	// Path 2: copy the value and run the copy.
	copied := evaluator
	copied.Run(candidate, task)
	if copied.Digest() != original || evaluator.Digest() != original {
		t.Fatalf("digest changed after copying and running the copy: copy=%x original=%x want=%x", copied.Digest(), evaluator.Digest(), original)
	}

	// Path 3: pass it through an interface.
	type runner interface {
		Run(Candidate, Task) Result
		Digest() [32]byte
	}
	var r runner = evaluator
	r.Run(candidate, task)
	if r.Digest() != original {
		t.Fatalf("digest changed after passing through an interface: %x vs %x", r.Digest(), original)
	}

	// Path 4: store it in a map and read it back.
	m := map[string]FrozenEvaluator{"x": evaluator}
	got := m["x"]
	got.Run(candidate, task)
	if got.Digest() != original {
		t.Fatalf("digest changed after storing in and reading from a map: %x vs %x", got.Digest(), original)
	}

	// Path 5: take the digest twice.
	if evaluator.Digest() != evaluator.Digest() {
		t.Fatal("two consecutive Digest() calls disagreed")
	}
}

// --- TestEvaluatorHasNoMutator ---

// scanEvaluatorMutatorViolations reports every exported field on
// FrozenEvaluator/AcceptanceCriteria and every pointer-receiver method on
// either type, found in file.
func scanEvaluatorMutatorViolations(file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !shadowIsolationTargetTypes[ts.Name.Name] {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						if ast.IsExported(name.Name) {
							violations = append(violations, fmt.Sprintf("%s has exported field %s", ts.Name.Name, name.Name))
						}
					}
				}
			}
		case *ast.FuncDecl:
			if d.Recv == nil || len(d.Recv.List) == 0 {
				continue
			}
			star, isPtr := d.Recv.List[0].Type.(*ast.StarExpr)
			if !isPtr {
				continue
			}
			ident, ok := star.X.(*ast.Ident)
			if !ok || !shadowIsolationTargetTypes[ident.Name] {
				continue
			}
			violations = append(violations, fmt.Sprintf("func %s has a pointer receiver on %s", d.Name.Name, ident.Name))
		}
	}
	return violations
}

// TestEvaluatorHasNoMutator is an AST scan of this package's real,
// non-test source proving FrozenEvaluator and AcceptanceCriteria declare no
// exported field and no pointer-receiver method -- a pointer receiver
// anywhere on either type would defeat the whole isolation guarantee this
// package exists to make.
func TestEvaluatorHasNoMutator(t *testing.T) {
	files, err := shadowNonTestGoFiles(".")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("fixture is broken: no non-test .go files found in pkg/shadow")
	}
	var violations []string
	for _, path := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		violations = append(violations, scanEvaluatorMutatorViolations(file)...)
	}
	if len(violations) != 0 {
		t.Fatalf("found a mutator on the grader:\n  %s", strings.Join(violations, "\n  "))
	}

	t.Run("a synthetic pointer-receiver mutator is caught", func(t *testing.T) {
		src := `package shadow

type FrozenEvaluator struct {
	Digest [32]byte
}

func (e *FrozenEvaluator) SetDigest(d [32]byte) {
	e.Digest = d
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fixture_mutator.go", src, 0)
		if err != nil {
			t.Fatalf("parse fixture source: %v", err)
		}
		got := scanEvaluatorMutatorViolations(file)
		if len(got) == 0 {
			t.Fatal("scanner failed to detect a synthetic pointer-receiver mutator and a synthetic exported field")
		}
	})
}

// --- TestNoExportedFunctionReturnsAMutableEvaluator ---

// scanMutableEvaluatorReturnViolations reports every exported function in
// file whose return type includes *FrozenEvaluator or *AcceptanceCriteria.
func scanMutableEvaluatorReturnViolations(file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !fn.Name.IsExported() || fn.Type.Results == nil {
			continue
		}
		for _, field := range fn.Type.Results.List {
			star, ok := field.Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			ident, ok := star.X.(*ast.Ident)
			if !ok || !shadowIsolationTargetTypes[ident.Name] {
				continue
			}
			violations = append(violations, fmt.Sprintf("func %s returns *%s", fn.Name.Name, ident.Name))
		}
	}
	return violations
}

// TestNoExportedFunctionReturnsAMutableEvaluator is an AST scan proving no
// exported function anywhere in this package's real, non-test source
// returns a pointer to the grader or to the acceptance criteria -- the
// third structural condition: no function anywhere hands out a changeable
// reference to either type.
func TestNoExportedFunctionReturnsAMutableEvaluator(t *testing.T) {
	files, err := shadowNonTestGoFiles(".")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}
	var violations []string
	for _, path := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		violations = append(violations, scanMutableEvaluatorReturnViolations(file)...)
	}
	if len(violations) != 0 {
		t.Fatalf("found an exported function returning a mutable reference to the grader:\n  %s", strings.Join(violations, "\n  "))
	}

	t.Run("a synthetic mutable-return function is caught", func(t *testing.T) {
		src := `package shadow

func BuildEvaluator() *FrozenEvaluator {
	return nil
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fixture_return.go", src, 0)
		if err != nil {
			t.Fatalf("parse fixture source: %v", err)
		}
		got := scanMutableEvaluatorReturnViolations(file)
		if len(got) == 0 {
			t.Fatal("scanner failed to detect a synthetic function returning *FrozenEvaluator")
		}
	})
}

// --- TestCandidateHoldsNoEvaluator ---

// candidateFieldTypeName resolves the plain type-identifier name of a
// struct field, whether declared by value or by pointer.
func candidateFieldTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// scanCandidateHoldsEvaluatorViolations reports every field on a struct
// named Candidate whose type is FrozenEvaluator, *FrozenEvaluator,
// AcceptanceCriteria or *AcceptanceCriteria, found in file.
func scanCandidateHoldsEvaluatorViolations(file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.TYPE {
			continue
		}
		for _, spec := range d.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Candidate" {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok || st.Fields == nil {
				continue
			}
			for _, field := range st.Fields.List {
				typeName := candidateFieldTypeName(field.Type)
				if !shadowIsolationTargetTypes[typeName] {
					continue
				}
				fieldNames := "<embedded>"
				if len(field.Names) > 0 {
					var names []string
					for _, n := range field.Names {
						names = append(names, n.Name)
					}
					fieldNames = strings.Join(names, ",")
				}
				violations = append(violations, fmt.Sprintf("Candidate field %s has type %s", fieldNames, typeName))
			}
		}
	}
	return violations
}

// TestCandidateHoldsNoEvaluator is an AST scan proving Candidate declares
// no field of type FrozenEvaluator or AcceptanceCriteria -- the fourth
// structural condition: the candidate type has no place to hold a
// changeable reference to what grades it.
func TestCandidateHoldsNoEvaluator(t *testing.T) {
	files, err := shadowNonTestGoFiles(".")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}
	var violations []string
	found := false
	for _, path := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.TYPE {
				for _, spec := range d.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == "Candidate" {
						found = true
					}
				}
			}
		}
		violations = append(violations, scanCandidateHoldsEvaluatorViolations(file)...)
	}
	if !found {
		t.Fatal("fixture is broken: no declaration of type Candidate found anywhere in pkg/shadow")
	}
	if len(violations) != 0 {
		t.Fatalf("Candidate holds a field it must not:\n  %s", strings.Join(violations, "\n  "))
	}

	t.Run("a synthetic evaluator-holding candidate is caught", func(t *testing.T) {
		src := `package shadow

type Candidate struct {
	id      string
	grader  FrozenEvaluator
}
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fixture_candidate.go", src, 0)
		if err != nil {
			t.Fatalf("parse fixture source: %v", err)
		}
		got := scanCandidateHoldsEvaluatorViolations(file)
		if len(got) == 0 {
			t.Fatal("scanner failed to detect a synthetic Candidate carrying a grader field")
		}
	})
}

// TestPackageImportsNothingFromCmd is the import check
// TestEvaluatorHasNoMutator's own acceptance criteria requires: pkg/shadow
// imports nothing from cmd, the compartment the isolation this package
// provides would be pointless inside.
func TestPackageImportsNothingFromCmd(t *testing.T) {
	files, err := shadowNonTestGoFiles(".")
	if err != nil {
		t.Fatalf("list package files: %v", err)
	}
	fset := token.NewFileSet()
	for _, path := range files {
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse imports of %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.Contains(importPath, "/cmd") {
				t.Fatalf("%s imports %q -- pkg/shadow must import nothing from the cmd package", path, importPath)
			}
		}
	}
}
