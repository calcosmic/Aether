package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// liveModelSingletonViolation names one candidate transport type that a
// single function both produces (takes as a parameter/receiver, or
// constructs via a composite literal) and passes to a real file-write call
// -- all within that one function's own body. That combination is the
// structural signature of a second event transport reaching durable
// storage, not merely an in-memory or wire-transmission shape (which the
// streaming server's wsEvent has, without ever touching a file).
type liveModelSingletonViolation struct {
	TypeName string
	Position string
	Function string
}

// liveModelSingletonPermittedTypes names candidate-shaped types that are
// known, reviewed exceptions -- expressed as the type's own symbol, never
// the file it happens to live in today, so renaming cmd/serve.go (or any
// other file) can never silently widen or narrow this guard. wsEvent
// (cmd/serve.go) is the streaming server's WebSocket wire-transmission
// shape: it carries a timestamp and a type field structurally, exactly like
// the retired transport did, but is never passed to a file write anywhere
// in the package -- TestOneLiveEventModelOnly's "wsEvent structurally
// matches" subtest proves the detector actually sees it rather than this
// guard being vacuous.
var liveModelSingletonPermittedTypes = map[string]bool{
	"wsEvent": true,
}

// liveModelSingletonPermittedFunctions names functions already known to
// legitimately touch event-shaped data on a durable-storage path without
// constituting a second transport -- expressed as the function's own
// symbol, never a file path. emitColonyLive (cmd/live_events.go) is the one
// live-event emission boundary; it hands events.ColonyLivePayload (declared
// in pkg/events, so it can never itself be a "candidate" from this
// package's own scan) to events.Bus.Publish, which persists via
// pkg/storage -- never a raw os-file call inside this package.
// streamEventBusNDJSON (cmd/eventbus.go) is the CLI subcommand surface; it
// writes NDJSON to stdout, never to a file. Neither is required for the
// guard to pass today (see the structural reasoning above), but both are
// named here so a real, reviewed exception has a place to be added without
// weakening the structural detector itself.
var liveModelSingletonPermittedFunctions = map[string]bool{
	"emitColonyLive":       true,
	"streamEventBusNDJSON": true,
}

// fieldSymbols returns every name a struct field is addressable by for
// shape matching: its Go field name(s), and its JSON tag's name segment (if
// any, excluding "-").
func fieldSymbols(field *ast.Field) []string {
	var out []string
	for _, name := range field.Names {
		out = append(out, name.Name)
	}
	if field.Tag != nil {
		if tagVal, err := strconv.Unquote(field.Tag.Value); err == nil {
			if jsonTag := reflect.StructTag(tagVal).Get("json"); jsonTag != "" {
				if name := strings.Split(jsonTag, ",")[0]; name != "" && name != "-" {
					out = append(out, name)
				}
			}
		}
	}
	return out
}

// structHasTimestampAndKindShape reports whether st declares a field
// identifiable (by Go field name or JSON tag) as a timestamp, and a
// separate field identifiable as an event kind/type -- the structural
// fingerprint a durable event-line transport carries (the retired
// cmd/event_types.go EventLine's own Type/Timestamp pair), independent of
// what the type or its fields happen to be named.
func structHasTimestampAndKindShape(st *ast.StructType) bool {
	if st.Fields == nil {
		return false
	}
	hasTimestamp := false
	hasKind := false
	for _, field := range st.Fields.List {
		for _, symbol := range fieldSymbols(field) {
			lower := strings.ToLower(symbol)
			if strings.Contains(lower, "timestamp") {
				hasTimestamp = true
			}
			switch lower {
			case "type", "kind", "eventtype", "event_type", "eventkind", "event_kind":
				hasKind = true
			}
		}
	}
	return hasTimestamp && hasKind
}

// findLiveModelSingletonCandidates walks every top-level type declaration
// across files and returns each struct type matching
// structHasTimestampAndKindShape, keyed by type name and pointing at the
// type name's own declaration position.
func findLiveModelSingletonCandidates(files []*ast.File) map[string]token.Pos {
	candidates := map[string]token.Pos{}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if structHasTimestampAndKindShape(st) {
					candidates[ts.Name.Name] = ts.Name.Pos()
				}
			}
		}
	}
	return candidates
}

// typeNameOf resolves the bare identifier name of an expression naming a
// type -- unwrapping pointer and slice/array wrappers -- or "" if it does
// not name a simple identifier-based type.
func typeNameOf(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return typeNameOf(t.X)
	case *ast.ArrayType:
		return typeNameOf(t.Elt)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// functionMentionsCandidate reports whether fn's receiver, parameters, or
// body (via a composite literal) reference candidateName -- i.e. fn is a
// producer of that candidate type.
func functionMentionsCandidate(fn *ast.FuncDecl, candidateName string) bool {
	if fn.Recv != nil {
		for _, field := range fn.Recv.List {
			if typeNameOf(field.Type) == candidateName {
				return true
			}
		}
	}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if typeNameOf(field.Type) == candidateName {
				return true
			}
		}
	}
	found := false
	if fn.Body != nil {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			cl, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			if typeNameOf(cl.Type) == candidateName {
				found = true
			}
			return true
		})
	}
	return found
}

// functionPerformsRawFileWrite reports whether fn's own body calls a raw
// file-write primitive directly (os.WriteFile, os.OpenFile, os.Create, or
// ioutil.WriteFile) -- the shape a second event transport's own writer
// would use, as the retired cmd/event_writer.go's WriteEvent did with
// os.OpenFile. Writes routed through pkg/storage (the surviving bus's own
// persistence) or to an http.ResponseWriter / websocket connection (the
// streaming server) never match this.
func functionPerformsRawFileWrite(fn *ast.FuncDecl) bool {
	if fn.Body == nil {
		return false
	}
	found := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		switch pkgIdent.Name {
		case "os":
			switch sel.Sel.Name {
			case "WriteFile", "OpenFile", "Create":
				found = true
			}
		case "ioutil":
			if sel.Sel.Name == "WriteFile" {
				found = true
			}
		}
		return true
	})
	return found
}

// findLiveModelSingletonViolations parses candidate transport types from
// files (structural detection over the syntax tree, never a maintained
// name list), then reports each one a single function both produces and
// passes to a real file-write call within that same function's own body --
// unless the type or the function is on the explicit, symbol-keyed
// permitted set.
func findLiveModelSingletonViolations(fset *token.FileSet, files []*ast.File, permittedTypes, permittedFunctions map[string]bool) []liveModelSingletonViolation {
	candidates := findLiveModelSingletonCandidates(files)

	var violations []liveModelSingletonViolation
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if permittedFunctions[fn.Name.Name] {
				continue
			}
			if !functionPerformsRawFileWrite(fn) {
				continue
			}
			for typeName, pos := range candidates {
				if permittedTypes[typeName] {
					continue
				}
				if functionMentionsCandidate(fn, typeName) {
					violations = append(violations, liveModelSingletonViolation{
						TypeName: typeName,
						Position: fset.Position(pos).String(),
						Function: fn.Name.Name,
					})
				}
			}
		}
	}
	return violations
}

// TestOneLiveEventModelOnly proves, from the parsed syntax tree rather than
// from review, that no top-level type in package cmd both declares a
// serialized timestamp-plus-event-kind shape (the retired
// cmd/event_types.go EventLine's own fingerprint) and is passed to a real
// file-write call by the same function -- i.e. that the surviving bus
// (pkg/events, reached through cmd/live_events.go's emitColonyLive) is the
// only event serialization path reaching durable storage in this package.
// A future reintroduction, under any file name, fails this test by naming
// the offending type and its declaration position.
func TestOneLiveEventModelOnly(t *testing.T) {
	fset := token.NewFileSet()
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}

	var realFiles []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		realFiles = append(realFiles, file)
	}

	t.Run("real package has zero second-transport violations", func(t *testing.T) {
		violations := findLiveModelSingletonViolations(fset, realFiles, liveModelSingletonPermittedTypes, liveModelSingletonPermittedFunctions)
		if len(violations) != 0 {
			t.Fatalf("found event-shaped type(s) reaching durable storage outside the surviving bus: %+v", violations)
		}
	})

	t.Run("wsEvent structurally matches the candidate shape but is not reported", func(t *testing.T) {
		candidates := findLiveModelSingletonCandidates(realFiles)
		if _, ok := candidates["wsEvent"]; !ok {
			t.Fatal("fixture is broken: wsEvent (cmd/serve.go) no longer matches the timestamp+event-kind shape -- this subtest exists to prove the structural detector is not vacuous; if wsEvent genuinely changed shape, replace it with another real candidate")
		}
	})

	t.Run("a fixture second transport is reported by name and declaration position", func(t *testing.T) {
		const fixtureSrc = `package cmd

import "os"

type sneakyEventLine struct {
	Kind      string ` + "`json:\"kind\"`" + `
	Timestamp string ` + "`json:\"timestamp\"`" + `
}

func sneakyEventWriter(path string, line sneakyEventLine) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line.Kind)
	return err
}
`
		fixtureFset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fixtureFset, "fixture_second_transport.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse synthetic fixture: %v", err)
		}

		violations := findLiveModelSingletonViolations(fixtureFset, []*ast.File{fixtureFile}, liveModelSingletonPermittedTypes, liveModelSingletonPermittedFunctions)
		if len(violations) != 1 {
			t.Fatalf("expected exactly 1 violation from the fixture, got %d: %+v", len(violations), violations)
		}
		if violations[0].TypeName != "sneakyEventLine" {
			t.Fatalf("violation named type %q, want %q", violations[0].TypeName, "sneakyEventLine")
		}
		if violations[0].Function != "sneakyEventWriter" {
			t.Fatalf("violation named function %q, want %q", violations[0].Function, "sneakyEventWriter")
		}
		if !strings.Contains(violations[0].Position, "fixture_second_transport.go") {
			t.Fatalf("violation position %q does not name the fixture file", violations[0].Position)
		}
	})

	t.Run("permitting a type name suppresses its own violation", func(t *testing.T) {
		const fixtureSrc = `package cmd

import "os"

type allowedEventLine struct {
	Kind      string ` + "`json:\"kind\"`" + `
	Timestamp string ` + "`json:\"timestamp\"`" + `
}

func allowedEventWriter(path string, line allowedEventLine) error {
	_, err := os.Create(path)
	return err
}
`
		fixtureFset := token.NewFileSet()
		fixtureFile, err := parser.ParseFile(fixtureFset, "fixture_allowed.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse synthetic fixture: %v", err)
		}

		permittedTypes := map[string]bool{"allowedEventLine": true}
		violations := findLiveModelSingletonViolations(fixtureFset, []*ast.File{fixtureFile}, permittedTypes, liveModelSingletonPermittedFunctions)
		if len(violations) != 0 {
			t.Fatalf("expected the explicitly permitted type to be suppressed, got %+v", violations)
		}
	})
}
