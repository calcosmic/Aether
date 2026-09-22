package cmd

// 208-06-PLAN.md Task 1 (UED-11): the honest size of the problem. This file
// walks the ten declared lifecycle files (refusalDeclaredFiles,
// cmd/refusal_register.go) with Go's own AST packages and counts every
// rule-applying error return that is not yet a typed refuse(...) call. That
// count is compared, two-sided, against the recorded floor in
// cmd/testdata/refusals/untyped-floor.json -- the same shrink-only ratchet
// shape cmd/eval_gates.go's seedBankUnguardedFloor and
// cmd/journey_expected_red_test.go's journeyExpectedRedCeiling already use
// in this package, applied here to a JSON-recorded number instead of a Go
// literal because the count moves every time a later 208-0x plan converts a
// site, and a committed JSON file -- not a typed-in constant -- is what a
// human reviewer actually re-measures against.

import (
	"encoding/json"
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

// refusalSite is one candidate refusal site the enumerator found: a return
// statement whose error operand is a direct call to fmt.Errorf, errors.New
// or refuse. IsRefuse marks a site that is already a typed refuse(...) call
// -- already registered, never counted toward the untyped floor. ID carries
// the refuse(...) call's own first argument (the literal id string) when
// IsRefuse is true, empty otherwise.
type refusalSite struct {
	File     string
	Line     int
	Func     string
	IsRefuse bool
	ID       string
}

// isWrappingErrorfCall reports whether call -- already known to be a call to
// fmt.Errorf -- wraps another error rather than applying a rule of its own:
// its format string names a %w verb, or one of its trailing arguments is
// itself error-shaped (an identifier or selector literally named "err" or
// ending "Err", or a call to a method literally named "Error" or "Unwrap").
// This is a plain-AST heuristic -- this package's other structural scanners
// (cmd/rendered_fields_invariant_test.go's renderedFieldASTFuncs) are
// deliberately written "without needing go/types or a compiled build", and
// this scanner follows the same discipline. TestRefusalEnumerationCanFail is
// the guard that this heuristic actually distinguishes the two shapes.
func isWrappingErrorfCall(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
		if unquoted, err := strconv.Unquote(lit.Value); err == nil {
			if strings.Contains(unquoted, "%w") {
				return true
			}
		} else if strings.Contains(lit.Value, "%w") {
			return true
		}
	}
	for _, arg := range call.Args[1:] {
		if refusalArgLooksLikeError(arg) {
			return true
		}
	}
	return false
}

// refusalArgLooksLikeError is the identifier/selector/call-name heuristic
// isWrappingErrorfCall applies to every non-format argument of a candidate
// fmt.Errorf call.
func refusalArgLooksLikeError(arg ast.Expr) bool {
	switch e := arg.(type) {
	case *ast.Ident:
		return refusalNameLooksLikeError(e.Name)
	case *ast.SelectorExpr:
		return refusalNameLooksLikeError(e.Sel.Name)
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			return sel.Sel.Name == "Error" || sel.Sel.Name == "Unwrap"
		}
	}
	return false
}

func refusalNameLooksLikeError(name string) bool {
	if name == "err" || name == "Err" {
		return true
	}
	return strings.HasSuffix(name, "Err") || strings.HasSuffix(name, "Error")
}

// refusalCallCallee returns the plain callee name for a direct package-level
// call (fmt.Errorf -> "fmt.Errorf", errors.New -> "errors.New", refuse(...)
// -> "refuse"), or "" for anything else (a method call, a variable holding a
// function, etc.) -- exactly the "direct call" the plan's action text scopes
// enumeration to.
func refusalCallCallee(call *ast.CallExpr) string {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return fn.Name
	case *ast.SelectorExpr:
		pkg, ok := fn.X.(*ast.Ident)
		if !ok {
			return ""
		}
		return pkg.Name + "." + fn.Sel.Name
	}
	return ""
}

// refusalCallLiteralID extracts a refuse(...) call's first argument as a
// plain string, when it is a string literal -- the id every refuse(...) call
// in this codebase is written with.
func refusalCallLiteralID(call *ast.CallExpr) (string, bool) {
	if len(call.Args) == 0 {
		return "", false
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return unquoted, true
}

// enumerateRefusalSitesInSource walks one already-parsed source file's AST
// and returns every refusalSite it contains. filename is used only for
// reporting -- src may be a real file's bytes or a small fixture literal
// (TestRefusalEnumerationCanFail).
func enumerateRefusalSitesInSource(filename string, src []byte) ([]refusalSite, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	var sites []refusalSite
	var funcStack []string
	// popMarkers is a parallel stack, one entry per node ast.Inspect opens
	// (true call), recording whether that node's opening pushed onto
	// funcStack -- ast.Inspect calls f(nil) exactly once for every node
	// whose f(node) call returned true, immediately after that node's own
	// children are fully visited (go/ast's documented depth-first-with-a-
	// closing-nil-call contract), so this is what lets a FuncLit's pushed
	// name come back off funcStack the moment its own body is done, rather
	// than leaking into the next sibling statement.
	var popMarkers []bool
	currentFunc := func() string {
		if len(funcStack) == 0 {
			return ""
		}
		return funcStack[len(funcStack)-1]
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			if len(popMarkers) > 0 {
				last := popMarkers[len(popMarkers)-1]
				popMarkers = popMarkers[:len(popMarkers)-1]
				if last {
					funcStack = funcStack[:len(funcStack)-1]
				}
			}
			return true
		}
		pushed := false
		switch node := n.(type) {
		case *ast.FuncDecl:
			funcStack = append(funcStack, node.Name.Name)
			pushed = true
		case *ast.FuncLit:
			funcStack = append(funcStack, currentFunc()+" (closure)")
			pushed = true
		case *ast.ReturnStmt:
			for _, result := range node.Results {
				call, ok := result.(*ast.CallExpr)
				if !ok {
					continue
				}
				callee := refusalCallCallee(call)
				pos := fset.Position(call.Pos())
				switch callee {
				case "fmt.Errorf":
					if isWrappingErrorfCall(call) {
						continue
					}
					sites = append(sites, refusalSite{
						File: filename, Line: pos.Line, Func: currentFunc(),
					})
				case "errors.New":
					sites = append(sites, refusalSite{
						File: filename, Line: pos.Line, Func: currentFunc(),
					})
				case "refuse":
					id, _ := refusalCallLiteralID(call)
					sites = append(sites, refusalSite{
						File: filename, Line: pos.Line, Func: currentFunc(),
						IsRefuse: true, ID: id,
					})
				}
			}
		}
		popMarkers = append(popMarkers, pushed)
		return true
	})

	return sites, nil
}

// enumerateRefusalSites reads and enumerates every declared file (base
// filenames, resolved relative to the cmd package directory -- the working
// directory `go test` runs tests from). Returns an error, never a silently
// empty file, on a missing or unparsable file -- refusalDeclaredFiles is a
// closed, checked-in list, and a file going missing is itself a fact this
// test must fail loudly on rather than quietly enumerate fewer sites.
func enumerateRefusalSites(files []string) ([]refusalSite, error) {
	var all []refusalSite
	for _, f := range files {
		data, err := os.ReadFile(filepath.Clean(f))
		if err != nil {
			return nil, fmt.Errorf("read declared refusal file %s: %w", f, err)
		}
		sites, err := enumerateRefusalSitesInSource(f, data)
		if err != nil {
			return nil, err
		}
		all = append(all, sites...)
	}
	return all, nil
}

// untypedRefusalSites filters sites down to the ones not yet a refuse(...)
// call -- the count TestEveryRefusalSiteIsRegisteredOrCounted and
// TestUntypedRefusalFloorOnlyShrinks both compare against the recorded
// floor.
func untypedRefusalSites(sites []refusalSite) []refusalSite {
	var out []refusalSite
	for _, s := range sites {
		if !s.IsRefuse {
			out = append(out, s)
		}
	}
	return out
}

// untypedRefusalFloor is the committed shape of
// cmd/testdata/refusals/untyped-floor.json.
type untypedRefusalFloor struct {
	Untyped int    `json:"untyped"`
	Reason  string `json:"reason"`
}

const untypedRefusalFloorPath = "testdata/refusals/untyped-floor.json"

func loadUntypedRefusalFloor(t *testing.T) untypedRefusalFloor {
	t.Helper()
	data, err := os.ReadFile(untypedRefusalFloorPath)
	if err != nil {
		t.Fatalf("read %s: %v", untypedRefusalFloorPath, err)
	}
	var floor untypedRefusalFloor
	if err := json.Unmarshal(data, &floor); err != nil {
		t.Fatalf("parse %s: %v", untypedRefusalFloorPath, err)
	}
	if strings.TrimSpace(floor.Reason) == "" {
		t.Fatalf("%s carries no reason -- the floor's own schema requires one", untypedRefusalFloorPath)
	}
	return floor
}

// formatRefusalSiteList renders a sorted, human-readable list of sites for
// a failure message -- never a bare count with nothing to act on.
func formatRefusalSiteList(sites []refusalSite) string {
	lines := make([]string, 0, len(sites))
	for _, s := range sites {
		lines = append(lines, fmt.Sprintf("%s:%d (%s)", s.File, s.Line, s.Func))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n  ")
}

// TestEveryRefusalSiteIsRegisteredOrCounted is the "real count may not
// exceed the recorded floor" half of the two-sided ratchet: every
// rule-applying error return in the declared files is either a registered
// refuse(...) call or accounted for on the committed, shrink-only floor.
// Anti-vacuity: the enumeration must find at least one site in every
// declared file, and at least one already-typed refuse(...) call overall,
// or this test fails saying the enumeration itself is broken rather than
// passing on an empty result.
func TestEveryRefusalSiteIsRegisteredOrCounted(t *testing.T) {
	sites, err := enumerateRefusalSites(refusalDeclaredFiles)
	if err != nil {
		t.Fatalf("enumerateRefusalSites: %v", err)
	}
	if len(sites) == 0 {
		t.Fatalf("enumeration found zero refusal sites across %d declared files -- the enumeration is broken, not vacuously complete", len(refusalDeclaredFiles))
	}

	perFile := map[string]int{}
	sawRefuse := false
	for _, s := range sites {
		perFile[s.File]++
		if s.IsRefuse {
			sawRefuse = true
		}
	}
	for _, f := range refusalDeclaredFiles {
		if perFile[f] == 0 {
			t.Errorf("declared file %s contributed zero refusal sites -- the enumeration is broken for this file, not honestly empty", f)
		}
	}
	if !sawRefuse {
		t.Errorf("enumeration found zero already-typed refuse(...) calls across the declared files -- the enumeration is broken, since refusalRegistry has registered rows reachable from these files")
	}
	if t.Failed() {
		return
	}

	untyped := untypedRefusalSites(sites)
	floor := loadUntypedRefusalFloor(t)
	if len(untyped) > floor.Untyped {
		t.Fatalf(
			"untyped refusal site count %d exceeds the recorded floor %d in %s -- register these as refuse(...) calls, or raise the floor in the same reviewed change with a written reason:\n  %s",
			len(untyped), floor.Untyped, untypedRefusalFloorPath, formatRefusalSiteList(untyped),
		)
	}
}

// TestUntypedRefusalFloorOnlyShrinks is the other half of the two-sided
// ratchet: the recorded floor may never sit ABOVE the real untyped count --
// the case TestEveryRefusalSiteIsRegisteredOrCounted's own inequality cannot
// see, since a floor left stale and high after a conversion would still
// pass that test forever. Together the two tests pin the floor to equal the
// real count exactly, mirroring cmd/eval_gates.go's
// assertSeedBankUnguardedWithinFloor / assertSeedBankUnguardedFloorIsExact
// pair.
func TestUntypedRefusalFloorOnlyShrinks(t *testing.T) {
	sites, err := enumerateRefusalSites(refusalDeclaredFiles)
	if err != nil {
		t.Fatalf("enumerateRefusalSites: %v", err)
	}
	untyped := untypedRefusalSites(sites)
	floor := loadUntypedRefusalFloor(t)
	if floor.Untyped > len(untyped) {
		t.Fatalf(
			"recorded untyped-floor.json count (%d) is inflated above the real untyped count (%d) -- lower it to %d in the same change that converted the sites",
			floor.Untyped, len(untyped), len(untyped),
		)
	}
}

// TestRefusalEnumerationCanFail is the guard on the guard (mirrors
// cmd/eval_gates.go's TestSeedBankUnguardedFloorIsTheRealCount /
// cmd/subcommand_reachability_ratchet_test.go's own anti-vacuity proofs): it
// runs the enumerator over a small fixture source containing one wrapping
// error and one non-wrapping error and asserts exactly the non-wrapping one
// is found -- proof the wrapping/non-wrapping distinction is load-bearing,
// not decorative.
func TestRefusalEnumerationCanFail(t *testing.T) {
	const fixture = `package fixture

import "fmt"

func wrapping(err error) error {
	if err != nil {
		return fmt.Errorf("something failed: %w", err)
	}
	return nil
}

func nonWrapping(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
`
	sites, err := enumerateRefusalSitesInSource("fixture.go", []byte(fixture))
	if err != nil {
		t.Fatalf("enumerateRefusalSitesInSource: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("expected exactly 1 refusal site (the non-wrapping return), got %d: %+v", len(sites), sites)
	}
	if sites[0].Func != "nonWrapping" {
		t.Fatalf("expected the one site to be in nonWrapping, got %q", sites[0].Func)
	}
}

// TestEveryRowCarriesAClassificationReason: every row in refusalRegistry
// carries a non-empty Reason and a Disposition of exactly "stop" or "warn";
// a row whose ProtectsWork is true must never carry Disposition "warn".
func TestEveryRowCarriesAClassificationReason(t *testing.T) {
	for _, row := range refusalRegistry {
		if strings.TrimSpace(row.Reason) == "" {
			t.Errorf("refusal row %q has an empty Reason", row.ID)
		}
		if row.Disposition != "stop" && row.Disposition != "warn" {
			t.Errorf("refusal row %q has Disposition %q, want exactly \"stop\" or \"warn\"", row.ID, row.Disposition)
		}
		if row.ProtectsWork && row.Disposition == "warn" {
			t.Errorf("refusal row %q protects work (ProtectsWork=true) but is marked Disposition=\"warn\" (carries on) -- a refusal that protects against losing work must never be reclassified as a warning that carries on", row.ID)
		}
	}
}
