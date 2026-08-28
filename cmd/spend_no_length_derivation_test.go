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
	"unicode"
)

// Phase 196 plan 02 — D-01 as amended (owner, 2026-08-27).
//
// The owner was shown a contradiction: the cost line promised no figure guessed
// from text length, and prompt-character-count was the only estimate mechanism
// in the tree, so a marked estimate would have been exactly the forbidden thing
// wearing a label. Asked to choose, he ruled: show no number at all. A worker
// whose tool reported nothing renders as "not reported", and the run total
// counts only measured workers and says so.
//
// The binding sentence: "Nothing may derive a token count from a character or
// prompt length, anywhere — not in the ledger, not in the renderer, not in
// `aether spend`."
//
// This is that rule made executable, per CLAUDE.md's Definition of Done: a
// requirement is satisfied only when a command exists that FAILS when it is
// unmet. Run against the tree before this plan's deletion it named
// pkg/codex/usage.go and pkg/codex/platform_dispatch.go — that failing run is
// the evidence the rule had a real violation rather than being a rule about
// nothing, and the evidence the scan is wide enough to see the file the
// violation actually lived in.
//
// SCOPE IS BY PACKAGE, NOT BY FILENAME. D-01 says anywhere. A pattern confined
// to the spend and wrapper-usage filenames would not have covered the one file
// the violation actually lived in, nor the chat-model client, the agent pool,
// the model-reason table, the closeout renderer or either finalize path —
// every one of them touched by this phase. Scanning whole packages means a
// file a later plan adds is covered the day it appears.
//
// THE ONLY EXEMPTION IS A SINGLE STATED RULE: a file whose name ends in
// _test.go is a test, and a test may legitimately construct a forbidden shape
// in order to prove this ratchet catches it. There is no list of individually
// pardoned files, and there must never become one — an exemption list is how a
// ratchet quietly loosens.
//
// THIS GUARD HAS NO ENVIRONMENT SWITCH AND NO SKIP PATH. That is not left as a
// promise: this file is registered in wiringGateGuardFiles, so
// TestWiringGuardsHaveNoRuntimeEscapeHatch scans it for exactly those, and
// TestWiringGateStepRunsEveryWiringTest requires the named CI step to run it.

// forbiddenLengthDerivationIdentifiers are named symbols that exist only to
// turn a length into a token count. Held here, in Go source, so nothing has to
// be grepped out of prose.
//
// Both were deleted by this plan. They stay named so a reintroduction under
// the SAME name fails immediately; a reintroduction under a NEW name is caught
// by the shape rules below instead.
var forbiddenLengthDerivationIdentifiers = map[string]string{
	"EstimateUsage":         "the character-derived usage fallback deleted by Phase 196 plan 02",
	"estimateCharsPerToken": "the characters-per-token ratio constant deleted by Phase 196 plan 02",
}

// lengthWords are the identifier words that mean "an amount of text", as
// opposed to an amount of tokens. Matched as whole camelCase words, never as
// substrings, so `silent`, `golden` and `resize` cannot trip on "len" or "size".
var lengthWords = map[string]bool{
	"char":       true,
	"chars":      true,
	"character":  true,
	"characters": true,
	"len":        true,
	"length":     true,
	"size":       true,
	"bytes":      true,
	"runes":      true,
}

// tokenWords are the identifier words that mean "an amount of tokens".
var tokenWords = map[string]bool{
	"token":  true,
	"tokens": true,
}

// TestNoTokenCountIsDerivedFromLength is the ratchet. It fails if any
// production file in the packages this phase touches derives a token count
// from a character count, a prompt length, or a characters-per-token ratio.
func TestNoTokenCountIsDerivedFromLength(t *testing.T) {
	packages := lengthScanPackages(t)

	// Anti-vacuity floor: a scan that silently enumerates nothing would pass
	// forever, which is the failure mode this whole file exists to prevent.
	var scanned int
	var offenders []string
	for _, dir := range packages {
		files := lengthScanProductionFiles(t, dir)
		if len(files) == 0 {
			t.Fatalf("scanned package %s and found no production Go files — an enumeration that finds nothing passes forever", dir)
		}
		scanned += len(files)
		for _, path := range files {
			offenders = append(offenders, lengthDerivationOffenders(t, path, nil)...)
		}
	}
	if scanned < 100 {
		t.Fatalf("the scan covered only %d production file(s) across %d package(s) — expected far more; "+
			"a silently narrowed scan is not a guard", scanned, len(packages))
	}

	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("%d site(s) derive a token count from a character or prompt length — D-01 as amended forbids this "+
			"ANYWHERE: a worker whose tool reported nothing shows no number at all, never a guess wearing a label:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}

	t.Run("a planted violation is caught in every scanned package", func(t *testing.T) {
		for _, dir := range packages {
			for _, planted := range []struct {
				name string
				src  string
			}{
				{
					name: "a length divided by a bare ratio into a token-named value",
					src: `package planted

func plantedRatioDivision(prompt string) int64 {
	tokens := int64(len(prompt) / 4)
	return tokens
}
`,
				},
				{
					name: "a named characters-per-token ratio",
					src: `package planted

const charsPerToken = 4

func plantedNamedRatio(promptChars int) int {
	return promptChars / charsPerToken
}
`,
				},
				{
					name: "the deleted helper reintroduced by name",
					src: `package planted

func plantedRevival(n int) interface{} {
	return EstimateUsage(n)
}
`,
				},
			} {
				path := filepath.Join(dir, "planted_length_derivation.go")
				found := lengthDerivationOffenders(t, path, planted.src)
				if len(found) == 0 {
					t.Errorf("planted violation %q in %s went unreported — the ratchet cannot see this package",
						planted.name, lengthScanRelPath(t, dir))
					continue
				}
				for _, offender := range found {
					if !strings.Contains(offender, lengthScanRelPath(t, path)) {
						t.Errorf("offender %q does not name the file it was found in", offender)
					}
				}
			}
		}
	})

	t.Run("the only exemption is that a _test.go file is a test", func(t *testing.T) {
		// The enumerator's rule, asserted directly: exactly one suffix is
		// exempt, and nothing else is. If this ever needs a second clause,
		// that is a reviewed change to a rule — never a new name added to a
		// list of pardons.
		for name, wantScanned := range map[string]bool{
			"usage.go":                           true,
			"platform_dispatch.go":               true,
			"spend_ledger.go":                    true,
			"usage_test.go":                      false,
			"spend_no_length_derivation_test.go": false,
		} {
			if got := lengthScanIsProductionFile(name); got != wantScanned {
				t.Errorf("lengthScanIsProductionFile(%q) = %v, want %v", name, got, wantScanned)
			}
		}

		// And the exemption must actually be excluding something, or it is
		// describing a condition that never arises.
		for _, dir := range lengthScanPackages(t) {
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("read %s: %v", dir, err)
			}
			var tests int
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), "_test.go") {
					tests++
				}
			}
			if tests == 0 {
				t.Errorf("package %s contains no test files at all — the stated exemption excludes nothing there", lengthScanRelPath(t, dir))
			}
		}
	})
}

// lengthDerivationOffenders parses one file and returns a description of every
// site in it that derives a token count from a length. Passing a non-nil src
// parses that source at the given path without touching disk, which is how the
// planted-violation subtest proves the scan catches a violation inside each
// real package directory.
func lengthDerivationOffenders(t *testing.T, path string, src interface{}) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	rel := lengthScanRelPath(t, path)
	var offenders []string
	report := func(pos token.Pos, symbol, why string) {
		offenders = append(offenders, fmt.Sprintf("%s:%d %s — %s", rel, fset.Position(pos).Line, symbol, why))
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		// Rule 1: a named symbol that exists only to turn a length into tokens.
		case *ast.Ident:
			if why, forbidden := forbiddenLengthDerivationIdentifiers[node.Name]; forbidden {
				report(node.Pos(), node.Name, why)
			}
		case *ast.SelectorExpr:
			if why, forbidden := forbiddenLengthDerivationIdentifiers[node.Sel.Name]; forbidden {
				report(node.Sel.Pos(), node.Sel.Name, why)
			}

		// Rule 2: an operand of * or / that names a characters-to-tokens ratio.
		case *ast.BinaryExpr:
			if node.Op != token.MUL && node.Op != token.QUO {
				return true
			}
			for _, operand := range []ast.Expr{node.X, node.Y} {
				if name, ok := identifierName(operand); ok && namesARatio(name) {
					report(operand.Pos(), name, "names a characters-to-tokens ratio used in an arithmetic expression")
				}
			}

		// Rule 3: a token-named target computed from a length.
		case *ast.AssignStmt:
			for i, lhs := range node.Lhs {
				if i >= len(node.Rhs) {
					break
				}
				checkTokenTargetFromLength(lhs, node.Rhs[i], report)
			}
		case *ast.ValueSpec:
			for i, name := range node.Names {
				if i >= len(node.Values) {
					break
				}
				checkTokenTargetFromLength(name, node.Values[i], report)
			}
		case *ast.KeyValueExpr:
			checkTokenTargetFromLength(node.Key, node.Value, report)
		}
		return true
	})

	return offenders
}

// checkTokenTargetFromLength reports a value assigned into a token-named
// target when that value multiplies or divides something length-shaped.
func checkTokenTargetFromLength(target ast.Expr, value ast.Expr, report func(token.Pos, string, string)) {
	name, ok := identifierName(target)
	if !ok || !namesTokens(name) {
		return
	}
	ast.Inspect(value, func(n ast.Node) bool {
		bin, isBinary := n.(*ast.BinaryExpr)
		if !isBinary || (bin.Op != token.MUL && bin.Op != token.QUO) {
			return true
		}
		for _, operand := range []ast.Expr{bin.X, bin.Y} {
			if isLengthExpression(operand) {
				report(target.Pos(), name,
					"is a token count computed by scaling a character or prompt length")
				return false
			}
		}
		return true
	})
}

// isLengthExpression reports whether an expression measures an amount of text:
// a len(...) call, or a name whose words say so.
func isLengthExpression(expr ast.Expr) bool {
	if call, ok := expr.(*ast.CallExpr); ok {
		if ident, isIdent := call.Fun.(*ast.Ident); isIdent && ident.Name == "len" {
			return true
		}
		if name, named := identifierName(call.Fun); named && namesLength(name) {
			return true
		}
		for _, arg := range call.Args {
			if isLengthExpression(arg) {
				return true
			}
		}
		return false
	}
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return isLengthExpression(paren.X)
	}
	name, ok := identifierName(expr)
	return ok && namesLength(name)
}

// identifierName renders an identifier or selector as a single name.
func identifierName(expr ast.Expr) (string, bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name, true
	case *ast.SelectorExpr:
		return e.Sel.Name, true
	}
	return "", false
}

// namesLength, namesTokens and namesARatio split an identifier into camelCase
// words and match whole words, never substrings.
func namesLength(name string) bool { return anyWordIn(name, lengthWords) }

func namesTokens(name string) bool { return anyWordIn(name, tokenWords) }

func namesARatio(name string) bool {
	return anyWordIn(name, lengthWords) && anyWordIn(name, tokenWords)
}

func anyWordIn(name string, vocabulary map[string]bool) bool {
	for _, word := range splitIdentifierWords(name) {
		if vocabulary[word] {
			return true
		}
	}
	return false
}

// splitIdentifierWords breaks camelCase, PascalCase and snake_case identifiers
// into lowercase words.
func splitIdentifierWords(name string) []string {
	var words []string
	var current []rune
	flush := func() {
		if len(current) > 0 {
			words = append(words, strings.ToLower(string(current)))
			current = nil
		}
	}
	for _, r := range name {
		switch {
		case r == '_':
			flush()
		case unicode.IsUpper(r):
			flush()
			current = append(current, r)
		default:
			current = append(current, r)
		}
	}
	flush()
	return words
}

// lengthScanPackages are the package directories D-01 is enforced over: every
// package this phase touches. Scanning by package rather than by filename is
// what makes a file added by a later plan covered the day it appears.
func lengthScanPackages(t *testing.T) []string {
	t.Helper()
	root := lengthScanRepoRoot(t)
	return []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "pkg", "codex"),
		filepath.Join(root, "pkg", "llm"),
		filepath.Join(root, "pkg", "agent"),
	}
}

// lengthScanIsProductionFile is the single stated exemption rule: a file whose
// name ends in _test.go is a test, everything else ending in .go is scanned.
func lengthScanIsProductionFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

func lengthScanProductionFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read package dir %s: %v", dir, err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !lengthScanIsProductionFile(entry.Name()) {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files
}

func lengthScanRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find the repository root (no go.mod found walking up)")
		}
		dir = parent
	}
}

func lengthScanRelPath(t *testing.T, path string) string {
	t.Helper()
	rel, err := filepath.Rel(lengthScanRepoRoot(t), path)
	if err != nil {
		return path
	}
	return rel
}
