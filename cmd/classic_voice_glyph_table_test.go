package cmd

// Phase "Classic Visual Voice" plan 01, Task 3 -- there is one glyph
// vocabulary in the runtime, and a guard that refuses a second one. Exactly
// two literals of the shape a hand-rolled caste-or-signal glyph table takes
// are allowed to exist: casteEmojiMap (the pre-existing caste table) and
// voiceGlyphMap (the one new semantic-line-type table this phase adds). Any
// third literal of that shape -- anywhere in the cmd package, at any scope --
// is refused by name, following the pattern
// TestNextActionHardcodeDetectsAPlantedViolation already uses for a synthetic
// on-disk fixture.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// voiceGlyphTableSite is one composite literal in source that has the shape
// of a hand-rolled caste-or-signal glyph table: two or more recognised caste
// names, or two or more of FOCUS/REDIRECT/FEEDBACK, among its keys, with at
// least one non-ASCII (glyph) value.
type voiceGlyphTableSite struct {
	File         string
	Function     string
	AssignedName string
	OffendingKey string
}

// voiceGlyphTableAllowedNames is the closed set of variable names a matching
// literal may legitimately be assigned to. Anything else is a second table.
var voiceGlyphTableAllowedNames = map[string]bool{
	"casteEmojiMap": true,
	"voiceGlyphMap": true,
}

// knownCasteNamesForGlyphScan reads the real caste name set from the
// production table (casteEmojiMap) rather than re-typing a parallel list --
// a caste added to the production table is automatically recognised here.
func knownCasteNamesForGlyphScan() map[string]bool {
	names := make(map[string]bool, len(casteEmojiMap))
	for caste := range casteEmojiMap {
		names[caste] = true
	}
	return names
}

var voiceGlyphTableSignalNames = map[string]bool{
	"FOCUS":    true,
	"REDIRECT": true,
	"FEEDBACK": true,
}

// isGlyphRune reports whether r falls in a Unicode range this codebase
// actually draws its emoji glyphs from -- not merely "non-ASCII", which
// would also match ordinary prose punctuation such as an em dash or a
// curly quote and false-positive on plain string tables like
// casteModelReasons (caste-keyed justification prose, not glyphs).
func isGlyphRune(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FFFF: // emoji, symbols, pictographs, supplements
		return true
	case r >= 0x2600 && r <= 0x27BF: // misc symbols, dingbats
		return true
	case r >= 0x2B00 && r <= 0x2BFF: // misc symbols and arrows (e.g. ⭐)
		return true
	case r >= 0xFE00 && r <= 0xFE0F: // variation selectors
		return true
	case r == 0x200D: // zero-width joiner (combined emoji)
		return true
	case r >= 0x2190 && r <= 0x21FF: // arrows (➡️ decomposes to U+2794-ish set; ➡ itself is U+27A1, covered above)
		return true
	}
	return false
}

func hasGlyphRune(s string) bool {
	for _, r := range s {
		if isGlyphRune(r) {
			return true
		}
	}
	return false
}

// stringLitValue unquotes a *ast.BasicLit string literal, returning ("", false)
// for anything else (a non-literal key/value, which this scan does not chase).
func stringLitValue(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return v, true
}

// glyphTableVisitor walks an AST tracking which function (if any) encloses
// the node currently being visited, so a matched literal can be reported
// with its enclosing function name.
type glyphTableVisitor struct {
	path        string
	funcName    string
	casteNames  map[string]bool
	signalNames map[string]bool
	sites       *[]voiceGlyphTableSite
}

func (v *glyphTableVisitor) checkLit(lit *ast.CompositeLit, assignedName string) {
	mapType, ok := lit.Type.(*ast.MapType)
	if !ok {
		return
	}
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return
	}
	valIdent, ok := mapType.Value.(*ast.Ident)
	if !ok || valIdent.Name != "string" {
		return
	}

	casteMatches := 0
	signalMatches := 0
	totalKeys := 0
	hasGlyphValue := false
	var firstOffendingKey string
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := stringLitValue(kv.Key)
		if !ok {
			continue
		}
		totalKeys++
		if v.casteNames[key] {
			casteMatches++
			if firstOffendingKey == "" {
				firstOffendingKey = key
			}
		}
		if v.signalNames[key] {
			signalMatches++
			if firstOffendingKey == "" {
				firstOffendingKey = key
			}
		}
		if val, ok := stringLitValue(kv.Value); ok && hasGlyphRune(val) {
			hasGlyphValue = true
		}
	}

	if !hasGlyphValue || totalKeys == 0 {
		return
	}
	// A hand-rolled caste/signal duplicate is a SMALL table that is
	// (almost) entirely made of caste or signal-type keys -- casteEmojiMap
	// itself, or a 2-3-entry local `emojiFor := map[string]string{"FOCUS":
	// ..., "REDIRECT": ..., "FEEDBACK": ...}`. A big, legitimately different
	// table that merely shares a FEW key names by vocabulary coincidence
	// (commandEmojiMap has "oracle", "chaos", "porter", "medic" as command
	// names that also happen to be caste names) must not trip this guard --
	// so the match also requires the caste/signal keys to be at least half
	// of the literal's own keys, not just an absolute count of two.
	casteRatio := float64(casteMatches) / float64(totalKeys)
	signalRatio := float64(signalMatches) / float64(totalKeys)
	matchesByCount := casteMatches >= 2 || signalMatches >= 2
	matchesByRatio := casteRatio >= 0.5 || signalRatio >= 0.5
	if !matchesByCount || !matchesByRatio {
		return
	}
	if voiceGlyphTableAllowedNames[assignedName] {
		return
	}
	fn := v.funcName
	if fn == "" {
		fn = "(package level)"
	}
	*v.sites = append(*v.sites, voiceGlyphTableSite{
		File:         v.path,
		Function:     fn,
		AssignedName: assignedName,
		OffendingKey: firstOffendingKey,
	})
}

func (v *glyphTableVisitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}
	switch node := n.(type) {
	case *ast.FuncDecl:
		child := *v
		if node.Name != nil {
			child.funcName = node.Name.Name
		}
		return &child
	case *ast.GenDecl:
		if node.Tok == token.VAR {
			for _, spec := range node.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.CompositeLit); ok {
						v.checkLit(lit, name.Name)
					}
				}
			}
		}
	case *ast.AssignStmt:
		for i, lhs := range node.Lhs {
			if i >= len(node.Rhs) {
				continue
			}
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			if lit, ok := node.Rhs[i].(*ast.CompositeLit); ok {
				v.checkLit(lit, ident.Name)
			}
		}
	}
	return v
}

// scanVoiceGlyphTableSites walks every non-test .go file directly inside dir
// (not recursively -- matching how the cmd package itself is laid out) and
// returns every composite literal matching the hand-rolled glyph table
// shape, whatever it is assigned to.
func scanVoiceGlyphTableSites(dir string) ([]voiceGlyphTableSite, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	casteNames := knownCasteNamesForGlyphScan()
	fset := token.NewFileSet()
	var sites []voiceGlyphTableSite
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil, err
		}
		visitor := &glyphTableVisitor{
			path:        path,
			casteNames:  casteNames,
			signalNames: voiceGlyphTableSignalNames,
			sites:       &sites,
		}
		ast.Walk(visitor, file)
	}
	return sites, nil
}

// TestVoiceGlyphsHaveOneTable asserts the only two composite literals in the
// cmd package shaped like a hand-rolled caste-or-signal glyph table are
// casteEmojiMap and voiceGlyphMap. Any third literal fails, naming the file,
// the enclosing function, and the offending key.
func TestVoiceGlyphsHaveOneTable(t *testing.T) {
	sites, err := scanVoiceGlyphTableSites(".")
	if err != nil {
		t.Fatalf("scan cmd/ for glyph table literals: %v", err)
	}
	if len(sites) > 0 {
		var lines []string
		for _, s := range sites {
			lines = append(lines, s.File+":"+s.Function+" assigns "+s.AssignedName+" (offending key "+s.OffendingKey+")")
		}
		t.Errorf("%d second glyph table(s) found beside casteEmojiMap/voiceGlyphMap:\n  %s",
			len(sites), strings.Join(lines, "\n  "))
	}
}

// TestVoiceGlyphTableGuardCanFail plants a synthetic file with a fresh
// caste-glyph literal of the forbidden shape and asserts the scanner locates
// it -- proving the guard can fail, not only pass. Follows the same
// synthetic-fixture pattern TestNextActionHardcodeDetectsAPlantedViolation
// uses.
func TestVoiceGlyphTableGuardCanFail(t *testing.T) {
	tmp := t.TempDir()
	planted := "package cmd\n\n" +
		"func renderSyntheticSteering() string {\n" +
		"\temojiFor := map[string]string{\"FOCUS\": \"\\U0001F3AF\", \"REDIRECT\": \"\\U0001F6AB\"}\n" +
		"\treturn emojiFor[\"FOCUS\"]\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(tmp, "synthetic.go"), []byte(planted), 0644); err != nil {
		t.Fatalf("write synthetic fixture: %v", err)
	}

	sites, err := scanVoiceGlyphTableSites(tmp)
	if err != nil {
		t.Fatalf("scan synthetic fixture: %v", err)
	}

	found := false
	for _, s := range sites {
		if s.Function == "renderSyntheticSteering" && s.AssignedName == "emojiFor" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("a planted second glyph table in a clean synthetic file was not detected; found sites: %+v", sites)
	}
}
