package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// classicContractPhase202RequiredNegativeCaseIDs are the six rejected
// shortcuts Task 2 of 202-15 requires -- one per Phase 202 group, each
// proving a specific corner-cutting alternative never returned: a second
// event transport, an invented liveness row, four cosmetically identical
// Swarm lenses, an uncheckpointed repair, a prefix-based episode deletion,
// and an Oracle synthesis that leads with its sources instead of its
// recommendation. Each ID must exist as a case in the corpus.
var classicContractPhase202RequiredNegativeCaseIDs = []string{
	"live-colony.events.no-second-event-transport",
	"live-colony.cockpit.no-invented-liveness-row",
	"swarm.lens.no-four-cosmetically-identical-lenses",
	"swarm.repair.no-uncheckpointed-repair",
	"swarm.episode.no-prefix-based-deletion",
	"oracle.synthesis.no-sources-first-ordering",
}

// TestClassicContractPhase202Cases is Task 2's own structural proof: every
// registered SYN-202 decision has at least one case, every Phase 202 group
// has at least two cases including one negative, every added case carries a
// causal (state or forbidden-artifact) assertion in addition to its
// semantic assertions, every case's go_test_symbol resolves to a real Go
// test function in the parsed cmd package, the six required negative cases
// are present by name, a case naming a nonexistent symbol fails by name,
// a case referencing an unregistered decision fails coverage validation by
// name, and a case asserting only rendered text fails schema validation. It
// never mutates the corpus or mechanism files it reads (proved by a
// before-and-after hash snapshot).
func TestClassicContractPhase202Cases(t *testing.T) {
	contractDir := classicContractFixtureDir(t)
	before := snapshotStoreFileHashesForTest(t, contractDir)

	document := loadClassicContractCorpus(t)
	symbols := classicContractCmdTestSymbols(t)

	decisionCounts := make(map[string]int, len(classicContractPhase202Decisions))
	groupCounts := make(map[string]int, len(classicContractPhase202Groups))
	present := make(map[string]bool)
	var phase202Cases []classicContractCase
	for _, testCase := range document.Cases {
		if !strings.HasPrefix(testCase.SynthesisDecision, "SYN-202-") {
			continue
		}
		phase202Cases = append(phase202Cases, testCase)
		present[testCase.ID] = true
		decisionCounts[testCase.SynthesisDecision]++
		groupCounts[testCase.Group]++
		if !slices.Contains(classicContractPhase202Groups, testCase.Group) {
			t.Errorf("case %q has group %q, want one of the six Phase 202 groups", testCase.ID, testCase.Group)
		}
		if len(testCase.Expected.StateAssertions) == 0 && len(testCase.Expected.ForbiddenArtifacts) == 0 {
			t.Errorf("case %q lacks a causal state or forbidden-artifact assertion", testCase.ID)
		}

		symbol, ok := classicContractCaseGoTestSymbol(testCase)
		if !ok {
			t.Errorf("case %q has no go_test_symbol semantic field", testCase.ID)
			continue
		}
		if !symbols[symbol] {
			t.Errorf("case %q names go_test_symbol %q, which does not exist in the parsed cmd package", testCase.ID, symbol)
		}
	}

	if len(phase202Cases) == 0 {
		t.Fatal("no Phase 202 cases were added to the corpus")
	}
	for _, id := range classicContractPhase202Decisions {
		if decisionCounts[id] == 0 {
			t.Errorf("SYN-202 decision %q has no case", id)
		}
	}
	for _, group := range classicContractPhase202Groups {
		if groupCounts[group] < 2 {
			t.Errorf("Phase 202 group %q has %d cases, want at least 2", group, groupCounts[group])
		}
	}
	for _, id := range classicContractPhase202RequiredNegativeCaseIDs {
		if !present[id] {
			t.Errorf("required negative case %q is missing", id)
		}
	}

	t.Run("a case asserting only rendered text fails schema validation", func(t *testing.T) {
		textOnly := phase202Cases[0]
		textOnly.Expected.StateAssertions = nil
		textOnly.Expected.ForbiddenArtifacts = nil
		invalidDoc := classicContractDocument{SchemaVersion: classicContractSchemaVersion, Cases: []classicContractCase{textOnly}}
		if err := validateClassicContractDocument(invalidDoc); err == nil || !strings.Contains(err.Error(), "missing causal state_assertions or forbidden_artifacts") {
			t.Fatalf("validation error = %v, want a text-only refusal", err)
		}
	})

	t.Run("a case naming a nonexistent Go test symbol fails by name", func(t *testing.T) {
		fixture := phase202Cases[0]
		fixture.ID = "phase202.probe.nonexistent-symbol"
		bogus := "TestPhase202NonexistentProofThatDoesNotExist"
		cloned := make(map[string]json.RawMessage, len(fixture.Expected.SemanticFields))
		for key, value := range fixture.Expected.SemanticFields {
			cloned[key] = value
		}
		cloned["go_test_symbol"] = json.RawMessage(`"` + bogus + `"`)
		fixture.Expected.SemanticFields = cloned

		symbol, ok := classicContractCaseGoTestSymbol(fixture)
		if !ok {
			t.Fatalf("fixture case lost its go_test_symbol field")
		}
		if symbol != bogus {
			t.Fatalf("fixture go_test_symbol = %q, want %q", symbol, bogus)
		}
		if symbols[symbol] {
			t.Fatalf("fixture symbol %q unexpectedly resolved -- the negative fixture must name a symbol that does not exist", symbol)
		}
	})

	t.Run("a case referencing an unregistered decision fails coverage validation by name", func(t *testing.T) {
		registry, err := loadClassicMechanismRegistry(filepath.Join(contractDir, "mechanisms.json"))
		if err != nil {
			t.Fatalf("load Classic mechanism registry: %v", err)
		}
		invalid := cloneClassicMechanismRegistry(t, registry)
		invalid.Mechanisms = classicMechanismsWithoutDecision(invalid.Mechanisms, "SYN-202-06")
		testCase := classicContractValidDocument().Cases[0]
		testCase.ID = "live-colony.probe.unregistered-decision"
		testCase.SynthesisDecision = "SYN-202-06"
		err = validateClassicCaseSynthesisCoverage([]classicContractCase{testCase}, invalid)
		if err == nil || !strings.Contains(err.Error(), `"SYN-202-06"`) {
			t.Fatalf("coverage validation error = %v, want it to name SYN-202-06", err)
		}
	})

	after := snapshotStoreFileHashesForTest(t, contractDir)
	assertHashSnapshotsEqualForTest(t, "Classic Phase 202 case validation", before, after)
}

// classicContractCaseGoTestSymbol extracts the go_test_symbol semantic
// field a Phase 202 case carries, returning ok=false when the field is
// absent or not a JSON string.
func classicContractCaseGoTestSymbol(testCase classicContractCase) (string, bool) {
	raw, ok := testCase.Expected.SemanticFields["go_test_symbol"]
	if !ok {
		return "", false
	}
	trimmed := strings.Trim(string(raw), `"`)
	if trimmed == "" {
		return "", false
	}
	return trimmed, true
}

// classicContractCmdTestSymbols parses every *_test.go file directly under
// cmd/ (this package) and returns the set of top-level Test function names
// declared anywhere in the package -- the same parsed-syntax-tree approach
// classicPhase200GoTestSymbolResolves uses for a single cited file,
// generalized here to the whole package so a case's go_test_symbol need not
// also appear in its own source_citations to be found. Never mutates
// anything it reads.
func classicContractCmdTestSymbols(t *testing.T) map[string]bool {
	t.Helper()
	repoRoot := findTestModuleRoot(t)
	cmdDir := filepath.Join(repoRoot, "cmd")
	matches, err := filepath.Glob(filepath.Join(cmdDir, "*_test.go"))
	if err != nil {
		t.Fatalf("glob cmd test files: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no cmd/*_test.go files found under %s", cmdDir)
	}

	symbols := make(map[string]bool)
	fileSet := token.NewFileSet()
	for _, path := range matches {
		parsed, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			if strings.HasPrefix(function.Name.Name, "Test") {
				symbols[function.Name.Name] = true
			}
		}
	}
	return symbols
}
