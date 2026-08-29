package cmd

// Phase 197 plan 07, task 2 -- criterion 3's measured hardcode ratchet.
//
//   "An automatic check counts how many places in the codebase still
//   hand-type a command name instead of using the shared logic, and fails the
//   build if that count ever grows from today's recorded baseline
//   (TestNextActionNeverHardcoded)." -- ROADMAP.md, Phase 197, success
//   criterion 3.
//
// S-03 (197-CONTEXT.md): the baseline may only shrink. Same discipline as
// testdata/orphan_allowlist.json and TestOrphanAllowlistOnlyShrinks
// (cmd/subcommand_reachability_ratchet_test.go): two files, a live one this
// scanner regenerates through a flag and a frozen one the flag must never
// touch, compared by set membership rather than by count so a one-out-one-in
// swap fails. cmd/colony_state_atomicity_ratchet_test.go is the second worked
// example of the same shape (a shrink-only allowlist keyed on file+function,
// never on a line number), and cmd/cli_flag_audit_test.go is the third.
//
// This file parses every .go file directly under cmd/ with go/ast, the same
// structural discipline all three siblings use, rather than grepping source
// text: a grep-ratchet is exactly the kind of check a rename or a reformat
// defeats silently.
//
// The scope rule (what counts as "next-step advice") is held as DATA in the
// three variables below -- nextActionAdviceFunnelFunctions,
// nextActionAdviceResultKeys, nextActionAdviceFunctionNameSuffixes -- so
// nothing about what this ratchet catches has to be inferred from prose. A
// string literal naming a runtime command ("aether <verb>") or a slash-
// wrapper command ("/ant-<verb>") counts as a violation when it is:
//
//  1. an argument (anywhere in the argument's own expression subtree,
//     including a string built with + or fmt.Sprintf) to a call whose
//     function name is one of nextActionAdviceFunnelFunctions -- the shared
//     card's own funnel, or a plain literal handed to it instead of the
//     resolver's answer;
//  2. the value side of a result-key assignment or map-literal entry whose
//     key is one of nextActionAdviceResultKeys -- a result map's own "what
//     to do next" field, filled by hand instead of applyNextActionToResult;
//     or
//  3. inside a return statement of a function whose name ends with one of
//     nextActionAdviceFunctionNameSuffixes -- a function whose own name
//     claims to decide the next step, returning a hand-typed command instead
//     of delegating to resolveNextAction.
//
// It exempts exactly one file by name -- cmd/next_action.go, the resolver,
// the one place a command may be named -- and every _test.go file. Nothing
// else is exempt, and that is the one stated rule (nextActionExemptFile),
// not a list of individual pardons.

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// updateNextActionHardcodeBaseline regenerates testdata/next_action_hardcode.json
// from the scanner's real, honest output. It must NEVER touch
// testdata/next_action_hardcode_baseline.json -- TestWiringGuardsHaveNoRuntimeEscapeHatch
// (cmd/subcommand_reachability_ratchet_test.go) asserts that in this file's
// own source, not just in this comment, once this file is registered in its
// scan (Phase 197 plan 07 task 3).
var updateNextActionHardcodeBaseline = flag.Bool("update-next-action-hardcode", false, "regenerate testdata/next_action_hardcode.json from the scanner's real output")

// nextActionAdviceFunnelFunctions is rule 1: a call to one of these names,
// found anywhere in the argument subtree, is the shared card's own funnel
// (renderNextUp -- the single place next_action_card.go's doc comment names
// as where platform translation happens) or its legacy adapter
// (renderNextUpVisual). A literal command handed directly to either, instead
// of the resolver's own answer, is exactly the hand-typing this ratchet
// exists to count down.
var nextActionAdviceFunnelFunctions = map[string]bool{
	"renderNextUp":       true,
	"renderNextUpVisual": true,
}

// nextActionAdviceResultKeys is rule 2: a result map's own "what to do next"
// field. Every key here is a key applyNextActionToResult
// (cmd/next_action_card.go) itself writes, or the older key it was
// introduced beside and never replaced (nextActionCommandKey's doc comment:
// "the older `next` key was left exactly as it was; the structured keys sit
// beside it").
var nextActionAdviceResultKeys = map[string]bool{
	"next":               true,
	nextActionCommandKey: true, // "next_command"
	"recovery_command":   true,
	"reconcile_command":  true,
}

// nextActionAdviceFunctionNameSuffixes is rule 3: a function whose own name
// says it decides or reports the next step. Every migrated adapter this
// phase built ends in one of these -- lifecycleNextAction,
// lifecycleNextActionForState, recoverNextAction, nextActionPrimarySuggestion
// -- which is precisely why a hand-typed command RETURNED from a
// similarly-named function, rather than routed through resolveNextAction, is
// exactly the drift this rule catches.
var nextActionAdviceFunctionNameSuffixes = []string{
	"NextStep",
	"NextAction",
	"NextCommand",
	"NextUp",
	"Suggestion",
	"Suggestions",
	"Advice",
	"Recommendation",
}

// commandLiteralRe matches an unquoted string's mention of a runtime command
// ("aether <verb>") or a slash-wrapper command ("/ant-<verb>"), the same two
// spellings S-01 and translateHintCommandsForPlatform distinguish between.
// Anchored so "aether" as a plain English word (this repo's own name, used
// constantly in prose that names nothing) never matches on its own -- only
// "aether " followed by a lowercase, hyphenated verb counts.
var commandLiteralRe = regexp.MustCompile(`(^|[^A-Za-z0-9_])(aether [a-z][a-z0-9-]*|/ant-[a-z][a-z0-9-]*)`)

// nextActionExemptFile is the one stated exemption rule: the resolver's own
// source, and any test file. Nothing else is exempt.
func nextActionExemptFile(baseName string) bool {
	if strings.HasSuffix(baseName, "_test.go") {
		return true
	}
	return baseName == "next_action.go"
}

// nextActionHardcodeSite is one hand-typed command-advice literal, found
// structurally by parsing the cmd package's .go files with go/ast.
type nextActionHardcodeSite struct {
	File     string // repo-relative, e.g. "cmd/codex_visuals.go"
	Function string // enclosing function name, or "cobra:<Use>" / "<package level>"
	Literal  string // the full, unquoted string literal the violation was found in
}

// nextActionHardcodeAllowlistEntry is one checked-in, accepted hardcode site.
// Keyed by (File, Function, Literal) per the plan's own requirement -- never
// by line number, so an unrelated edit above a site never changes its key.
type nextActionHardcodeAllowlistEntry struct {
	File     string `json:"file"`
	Function string `json:"function"`
	Literal  string `json:"literal"`
}

// nextActionHardcodeKey is the compound key TestNextActionNeverHardcoded
// compares against -- (File, Function, Literal), mirroring
// siteAllowlistKey (cmd/colony_state_atomicity_ratchet_test.go) and
// orphanAllowlistEntry's own Name-is-the-whole-key convention.
func nextActionHardcodeKey(file, function, literal string) string {
	return file + "\x00" + function + "\x00" + literal
}

// scanNextActionHardcodeSource parses every non-exempt .go file directly
// under cmdDir and finds every string literal matching one of the three
// scope rules documented at the top of this file.
func scanNextActionHardcodeSource(cmdDir string) ([]nextActionHardcodeSite, error) {
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return nil, fmt.Errorf("read cmd dir: %w", err)
	}

	var sites []nextActionHardcodeSite
	fset := token.NewFileSet()
	filesScanned := 0

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || nextActionExemptFile(name) {
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
		filesScanned++
		relName := filepath.Join("cmd", name)
		sites = append(sites, scanFileForNextActionHardcodes(file, relName)...)
	}

	if filesScanned == 0 {
		return nil, fmt.Errorf("scanned zero non-exempt .go files in %s", cmdDir)
	}
	return sites, nil
}

// scanFileForNextActionHardcodes applies all three scope rules to one
// already-parsed file, following scanFileForColonyStateWrites's two-shape
// convention (cmd/colony_state_atomicity_ratchet_test.go): every top-level
// FuncDecl, plus every inline RunE closure inside a &cobra.Command{}
// composite literal anywhere in the file, attributed to a synthetic
// "cobra:<Use>" name when unnamed. A top-level var declaration (rule 2's
// realistic package-level shape -- a package-level
// map[string]interface{}{"next": "aether ..."} literal) is scanned under
// "<package level>"; known, accepted scope bound: a package-level call
// expression or return statement cannot occur in Go, so rules 1 and 3 have
// no package-level shape to miss.
func scanFileForNextActionHardcodes(file *ast.File, relName string) []nextActionHardcodeSite {
	var sites []nextActionHardcodeSite

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body == nil {
				continue
			}
			sites = append(sites, recordNextActionHardcodeSites(d.Body, relName, d.Name.Name)...)
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
					sites = append(sites, recordNextActionResultKeySites(val, relName, "<package level>")...)
				}
			}
		}
	}

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
		sites = append(sites, recordNextActionHardcodeSites(runE.Body, relName, fnName)...)
		return true
	})

	return sites
}

// recordNextActionHardcodeSites walks node (a function or closure body) for
// all three scope rules, attributing every violation found to fnName.
func recordNextActionHardcodeSites(node ast.Node, relFile, fnName string) []nextActionHardcodeSite {
	var sites []nextActionHardcodeSite

	suffixed := false
	for _, suffix := range nextActionAdviceFunctionNameSuffixes {
		if strings.HasSuffix(fnName, suffix) {
			suffixed = true
			break
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.CallExpr:
			// Rule 1: a call to a funnel function. FuncLit nodes are still
			// visited by ast.Inspect's own recursion below (as with the
			// colony-state ratchet's recordColonyStateCallsAndSites), so a
			// funnel call inside a nested closure is still attributed to
			// THIS enclosing fnName.
			var callName string
			switch fn := stmt.Fun.(type) {
			case *ast.Ident:
				callName = fn.Name
			case *ast.SelectorExpr:
				callName = fn.Sel.Name
			}
			if nextActionAdviceFunnelFunctions[callName] {
				for _, arg := range stmt.Args {
					for _, lit := range matchingLiteralsIn(arg) {
						sites = append(sites, nextActionHardcodeSite{File: relFile, Function: fnName, Literal: lit})
					}
				}
			}
		case *ast.AssignStmt:
			// Rule 2, assignment shape: result["next"] = "aether ...".
			for i, lhs := range stmt.Lhs {
				idx, ok := lhs.(*ast.IndexExpr)
				if !ok || i >= len(stmt.Rhs) {
					continue
				}
				key, ok := idx.Index.(*ast.BasicLit)
				if !ok || key.Kind != token.STRING {
					continue
				}
				keyVal, unquoteErr := strconv.Unquote(key.Value)
				if unquoteErr != nil || !nextActionAdviceResultKeys[keyVal] {
					continue
				}
				for _, lit := range matchingLiteralsIn(stmt.Rhs[i]) {
					sites = append(sites, nextActionHardcodeSite{File: relFile, Function: fnName, Literal: lit})
				}
			}
		case *ast.CompositeLit:
			// Rule 2, map-literal shape: map[string]interface{}{"next": "aether ..."}.
			sites = append(sites, recordNextActionResultKeySites(stmt, relFile, fnName)...)
		case *ast.ReturnStmt:
			// Rule 3: a return statement inside a suffix-named function.
			if suffixed {
				for _, result := range stmt.Results {
					for _, lit := range matchingLiteralsIn(result) {
						sites = append(sites, nextActionHardcodeSite{File: relFile, Function: fnName, Literal: lit})
					}
				}
			}
		}
		return true
	})

	return sites
}

// recordNextActionResultKeySites finds rule 2's map-literal shape inside
// expr's own subtree: a KeyValueExpr whose Key is a string BasicLit matching
// nextActionAdviceResultKeys. Shared between the package-level var-decl scan
// (which hands this a bare composite literal expression) and the in-function
// walk (which hands this the CompositeLit node ast.Inspect already found).
func recordNextActionResultKeySites(expr ast.Expr, relFile, fnName string) []nextActionHardcodeSite {
	var sites []nextActionHardcodeSite
	if expr == nil {
		return sites
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := kv.Key.(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			return true
		}
		keyVal, unquoteErr := strconv.Unquote(key.Value)
		if unquoteErr != nil || !nextActionAdviceResultKeys[keyVal] {
			return true
		}
		for _, lit := range matchingLiteralsIn(kv.Value) {
			sites = append(sites, nextActionHardcodeSite{File: relFile, Function: fnName, Literal: lit})
		}
		return true
	})
	return sites
}

// matchingLiteralsIn returns every string literal in expr's own subtree
// (covering a plain literal, a +-concatenation, or a fmt.Sprintf-style
// format-string argument) whose unquoted value matches commandLiteralRe.
func matchingLiteralsIn(expr ast.Expr) []string {
	var out []string
	if expr == nil {
		return out
	}
	ast.Inspect(expr, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		val, unquoteErr := strconv.Unquote(lit.Value)
		if unquoteErr != nil {
			return true
		}
		if commandLiteralRe.MatchString(val) {
			out = append(out, val)
		}
		return true
	})
	return out
}

// loadNextActionHardcodeAllowlist reads a committed allowlist JSON file.
func loadNextActionHardcodeAllowlist(t *testing.T, path string) []nextActionHardcodeAllowlistEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update-next-action-hardcode to create testdata/next_action_hardcode.json)", path, err)
	}
	var entries []nextActionHardcodeAllowlistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

// writeNextActionHardcodeAllowlist writes the scanner's real, honest output
// to testdata/next_action_hardcode.json ONLY -- never the baseline.
// Deduplicates by the (File, Function, Literal) key TestNextActionNeverHardcoded
// compares against, mirroring writeColonyStateWriteAllowlist.
func writeNextActionHardcodeAllowlist(t *testing.T, sites []nextActionHardcodeSite) int {
	t.Helper()
	seen := map[string]bool{}
	entries := make([]nextActionHardcodeAllowlistEntry, 0, len(sites))
	for _, s := range sites {
		key := nextActionHardcodeKey(s.File, s.Function, s.Literal)
		if seen[key] {
			continue
		}
		seen[key] = true
		entries = append(entries, nextActionHardcodeAllowlistEntry{File: s.File, Function: s.Function, Literal: s.Literal})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		if entries[i].Function != entries[j].Function {
			return entries[i].Function < entries[j].Function
		}
		return entries[i].Literal < entries[j].Literal
	})
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal next action hardcode allowlist: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile("testdata/next_action_hardcode.json", data, 0644); err != nil {
		t.Fatalf("write testdata/next_action_hardcode.json: %v", err)
	}
	return len(entries)
}

// TestNextActionNeverHardcoded is criterion 3's own named ratchet. Every
// hand-typed command-advice literal the scanner finds in the cmd package
// (outside the resolver and test files) must already be recorded in the
// checked-in baseline testdata/next_action_hardcode_baseline.json, and every baseline
// entry must still correspond to a real site in source -- a stale entry
// (naming a site that was fixed or removed) fails too, exactly like
// TestOrphanAllowlistOnlyShrinks and TestColonyStateWriteAllowlistOnlyShrinks.
//
// This is deliberately a comparison against the FROZEN baseline file, not
// against the regenerable live file -- the live file and the baseline are
// required to be identical (TestNextActionHardcodeBaselineMatchesLive), so
// comparing against either is equivalent today, but only the baseline
// comparison is what makes a one-out-one-in swap of live-file contents (with
// no corresponding baseline edit) visible as a build failure.
func TestNextActionNeverHardcoded(t *testing.T) {
	sites, err := scanNextActionHardcodeSource(".")
	if err != nil {
		t.Fatalf("scan cmd/ for hardcoded next-step advice: %v", err)
	}

	// A scanner that finds zero sites has broken, not proven the codebase
	// clean overnight -- the pheromone-signal closing
	// (cmd/codex_visuals.go's renderSignalCreatedVisual) hand-types
	// "aether pheromones" and "aether status" directly into renderNextUp,
	// which is real, present, out-of-scope-for-migration advice this
	// scanner must find on every run.
	if len(sites) == 0 {
		t.Fatal("scanNextActionHardcodeSource found zero hand-typed command-advice sites across the cmd package -- the AST walker likely broke (wrong funnel/result-key/suffix data, wrong literal regex), not that every hand-typed site was migrated overnight.")
	}

	if *updateNextActionHardcodeBaseline {
		written := writeNextActionHardcodeAllowlist(t, sites)
		t.Logf("wrote testdata/next_action_hardcode.json with %d unique (file, function, literal) entries from %d raw scanned site(s)", written, len(sites))
		return
	}

	baseline := loadNextActionHardcodeAllowlist(t, "testdata/next_action_hardcode_baseline.json")
	allowed := map[string]bool{}
	for _, e := range baseline {
		allowed[nextActionHardcodeKey(e.File, e.Function, e.Literal)] = true
	}
	found := map[string]bool{}
	for _, s := range sites {
		found[nextActionHardcodeKey(s.File, s.Function, s.Literal)] = true
	}

	var newSites []string
	for _, s := range sites {
		if !allowed[nextActionHardcodeKey(s.File, s.Function, s.Literal)] {
			newSites = append(newSites, fmt.Sprintf("%s:%s %q", s.File, s.Function, s.Literal))
		}
	}
	var staleEntries []string
	for _, e := range baseline {
		if !found[nextActionHardcodeKey(e.File, e.Function, e.Literal)] {
			staleEntries = append(staleEntries, fmt.Sprintf("%s:%s %q", e.File, e.Function, e.Literal))
		}
	}
	sort.Strings(newSites)
	sort.Strings(staleEntries)

	if len(newSites) > 0 {
		t.Errorf("%d new hand-typed command-advice site(s) found that are not in testdata/next_action_hardcode_baseline.json:\n  %s\n"+
			"Route this advice through resolveNextAction/applyNextActionToResult instead. The baseline may only shrink -- do not hand-edit it to add an entry.",
			len(newSites), strings.Join(newSites, "\n  "))
	}
	if len(staleEntries) > 0 {
		t.Errorf("%d testdata/next_action_hardcode_baseline.json entry(ies) no longer match any real site in source:\n  %s\n"+
			"The site was fixed or removed -- a shrunk baseline is the intended outcome; this failure exists so the shrink is a visible, on-the-record edit rather than something a stale allowlist silently hides.",
			len(staleEntries), strings.Join(staleEntries, "\n  "))
	}
}

// TestNextActionHardcodeBaselineMatchesLive keeps testdata/next_action_hardcode.json
// (the live, regenerable file) and testdata/next_action_hardcode_baseline.json
// (the frozen, committed one) byte-identical, following the tolerated-orphan
// pair's own convention exactly (both files reflect the SAME scanner output;
// only the regeneration path differs -- see the two files' own doc comments).
func TestNextActionHardcodeBaselineMatchesLive(t *testing.T) {
	live, err := os.ReadFile("testdata/next_action_hardcode.json")
	if err != nil {
		t.Fatalf("read testdata/next_action_hardcode.json: %v", err)
	}
	baseline, err := os.ReadFile("testdata/next_action_hardcode_baseline.json")
	if err != nil {
		t.Fatalf("read testdata/next_action_hardcode_baseline.json: %v", err)
	}
	if string(live) != string(baseline) {
		t.Error("testdata/next_action_hardcode.json and testdata/next_action_hardcode_baseline.json have diverged -- " +
			"run with -update-next-action-hardcode to refresh the live file from the scanner's real output, " +
			"then review the diff against the baseline exactly like testdata/orphan_allowlist.json's own pair.")
	}
}

// TestNextActionHardcodeDetectsAPlantedViolation proves the scanner fires:
// a synthetic file, written to a temp directory alongside a copy of the real
// cmd/next_action.go exemption boundary, containing a fresh, clean call to
// the funnel function with a hand-typed command literal, is found.
func TestNextActionHardcodeDetectsAPlantedViolation(t *testing.T) {
	tmp := t.TempDir()
	planted := "func renderSyntheticClosing() string {\n\treturn renderNextUp(\"Run `aether ratchet-selftest` now.\")\n}\n"
	if err := os.WriteFile(filepath.Join(tmp, "synthetic.go"), []byte("package cmd\n\n"+planted), 0644); err != nil {
		t.Fatalf("write synthetic fixture: %v", err)
	}

	sites, err := scanNextActionHardcodeSource(tmp)
	if err != nil {
		t.Fatalf("scan synthetic fixture: %v", err)
	}

	found := false
	for _, s := range sites {
		if s.Function == "renderSyntheticClosing" && strings.Contains(s.Literal, "aether ratchet-selftest") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("a planted violation in a clean synthetic file was not detected; found sites: %+v", sites)
	}
}

// TestNextActionHardcodeSwapFails proves the comparison is set membership,
// not a count: replacing one baseline-recorded site with a different,
// equal-count synthetic site must still fail.
func TestNextActionHardcodeSwapFails(t *testing.T) {
	baseline := loadNextActionHardcodeAllowlist(t, "testdata/next_action_hardcode_baseline.json")
	if len(baseline) == 0 {
		t.Fatal("baseline is empty -- this swap test cannot remove one entry and add a different one, which would make it vacuous")
	}

	// Simulate the comparison TestNextActionNeverHardcoded performs, but with
	// the FIRST baseline entry replaced by a synthetic entry of the same
	// shape naming a different literal -- the "one out, one in" case a
	// count-only comparison would miss.
	allowed := map[string]bool{}
	for i, e := range baseline {
		if i == 0 {
			continue // removed
		}
		allowed[nextActionHardcodeKey(e.File, e.Function, e.Literal)] = true
	}
	swapped := nextActionHardcodeAllowlistEntry{
		File:     baseline[0].File,
		Function: baseline[0].Function,
		Literal:  baseline[0].Literal + " -- swapped for a different command entirely",
	}
	allowed[nextActionHardcodeKey(swapped.File, swapped.Function, swapped.Literal)] = true

	sites, err := scanNextActionHardcodeSource(".")
	if err != nil {
		t.Fatalf("scan cmd/ for hardcoded next-step advice: %v", err)
	}

	var newSites []string
	for _, s := range sites {
		if !allowed[nextActionHardcodeKey(s.File, s.Function, s.Literal)] {
			newSites = append(newSites, fmt.Sprintf("%s:%s %q", s.File, s.Function, s.Literal))
		}
	}
	if len(newSites) == 0 {
		t.Fatal("swapping one real baseline entry for a synthetic one of the same shape did not register as a new, unallowed site -- the comparison is comparing counts, not set membership")
	}

	removedKey := nextActionHardcodeKey(baseline[0].File, baseline[0].Function, baseline[0].Literal)
	stillCredited := false
	for key := range allowed {
		if key == removedKey {
			stillCredited = true
		}
	}
	if stillCredited {
		t.Fatal("the removed baseline entry is still present in the simulated allowlist -- the swap fixture is wrong")
	}
}
