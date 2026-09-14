package cmd

// LEARN-05 (204-07-PLAN.md): splits this project's one enormous test run
// into seven named gates -- fast, focused, integration, provider, overnight,
// race and release -- each with its own declared selection, environment and
// observable time budget, so nobody has to choose between twenty-one
// minutes and nothing. This is the documented root cause behind three
// separate entries in this project's own defect register: an unqualified
// full-suite run reported a result after executing roughly a third of the
// tests, printing a complete-looking summary of the fraction it finished.
// Every gate this file declares carries the discovered-equals-executed
// check that would have caught that defect on day one.
//
// Nothing here builds a second test runner or task orchestrator. Every gate
// is a `go test` invocation against the corpus that already exists, tiered
// by build tag and `-run` pattern -- the same mechanism CLAUDE.md's own
// Verification Commands section already uses, never a new one.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// --- Paths ---

// evalGateManifestPath is the repo-relative path of the committed, versioned
// gate manifest. Committed source, like the fixture bank and the Classic
// contract corpus -- read/written directly against the repository checkout,
// not through storage.Store.
const evalGateManifestPath = "cmd/testdata/eval-gates/gates.json"

// evalGateSentinelsPath is the repo-relative path of the hard-gate sentinel
// list: tests that may never be skipped or removed.
const evalGateSentinelsPath = "cmd/testdata/eval-gates/sentinels.json"

// evalGateHoldoutsPath is the repo-relative path of the hidden holdout set,
// stored as content digests so it cannot be read off by name.
const evalGateHoldoutsPath = "cmd/testdata/eval-gates/holdouts.json"

// evalGateManifestSchemaVersion is the schema_version constant the gate
// manifest carries.
const evalGateManifestSchemaVersion = "eval-gates/v1"

// evalGateRepoRootOverride lets tests redirect gate-manifest reads to an
// isolated temporary directory instead of the real repository checkout,
// mirroring fixtureBankRepoRootOverride's precedent exactly.
var evalGateRepoRootOverride string

// evalGateRepoRoot resolves the directory evalGateManifestPath and its
// siblings are relative to: evalGateRepoRootOverride when a test has set
// it, otherwise the real Aether module root.
func evalGateRepoRoot() (string, error) {
	if evalGateRepoRootOverride != "" {
		return evalGateRepoRootOverride, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("locate Aether module root from %s", cwd)
	}
	return root, nil
}

// --- Declared vocabularies ---
//
// Same completeness convention as cmd/recruitment_credit.go's
// recruitmentCreditOutcomeVocabulary and cmd/fixture_conversion.go's
// fixtureAllowedMutationVocabulary: const block, names slice, names()
// helper, declared-membership predicate.

// evalGateName is the declared, closed vocabulary of the seven named gates.
type evalGateName string

const (
	evalGateFast        evalGateName = "fast"
	evalGateFocused     evalGateName = "focused"
	evalGateIntegration evalGateName = "integration"
	evalGateProvider    evalGateName = "provider"
	evalGateOvernight   evalGateName = "overnight"
	evalGateRace        evalGateName = "race"
	evalGateRelease     evalGateName = "release"
)

var evalGateNameVocabulary = []evalGateName{
	evalGateFast,
	evalGateFocused,
	evalGateIntegration,
	evalGateProvider,
	evalGateOvernight,
	evalGateRace,
	evalGateRelease,
}

func evalGateNames() []string {
	names := make([]string, 0, len(evalGateNameVocabulary))
	for _, n := range evalGateNameVocabulary {
		names = append(names, string(n))
	}
	return names
}

func evalGateNameDeclared(n evalGateName) bool {
	for _, v := range evalGateNameVocabulary {
		if v == n {
			return true
		}
	}
	return false
}

// evalGateRequirement is the declared, closed vocabulary of environment
// facts a gate may require.
type evalGateRequirement string

const (
	evalGateRequiresNone                evalGateRequirement = "none"
	evalGateRequiresNetwork             evalGateRequirement = "network"
	evalGateRequiresProviderCredentials evalGateRequirement = "provider_credentials"
	evalGateRequiresLongWallClock       evalGateRequirement = "long_wall_clock"
)

var evalGateRequirementVocabulary = []evalGateRequirement{
	evalGateRequiresNone,
	evalGateRequiresNetwork,
	evalGateRequiresProviderCredentials,
	evalGateRequiresLongWallClock,
}

func evalGateRequirementNames() []string {
	names := make([]string, 0, len(evalGateRequirementVocabulary))
	for _, r := range evalGateRequirementVocabulary {
		names = append(names, string(r))
	}
	return names
}

func evalGateRequirementDeclared(r evalGateRequirement) bool {
	for _, v := range evalGateRequirementVocabulary {
		if v == r {
			return true
		}
	}
	return false
}

// --- Data types ---

// evalGate is one declared gate: its own selection, environment and budget.
type evalGate struct {
	Name          evalGateName        `json:"name"`
	Packages      []string            `json:"packages"`
	BuildTags     []string            `json:"build_tags"`
	RunPattern    string              `json:"run_pattern"`
	Requires      evalGateRequirement `json:"requires"`
	BudgetSeconds int                 `json:"budget_seconds"`
	Purpose       string              `json:"purpose"`
	// Race declares whether this gate's own invocation passes -race. Not
	// part of the plan's named seven fields, but required to express the
	// race gate's own defining characteristic honestly in the manifest
	// rather than hard-coding it only in the Makefile.
	Race bool `json:"race,omitempty"`
}

// evalGateManifest is the on-disk container at evalGateManifestPath.
type evalGateManifest struct {
	SchemaVersion string     `json:"schema_version"`
	Gates         []evalGate `json:"gates"`
}

// loadEvalGateManifest reads and parses the committed gate manifest.
// Loading twice returns identical gates in identical declared order
// (TestEvalGateManifestLoadIsStable) -- this is a plain read with no
// reordering of any kind.
func loadEvalGateManifest() (evalGateManifest, error) {
	root, err := evalGateRepoRoot()
	if err != nil {
		return evalGateManifest{}, err
	}
	path := filepath.Join(root, evalGateManifestPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return evalGateManifest{}, fmt.Errorf("read eval gate manifest %s: %w", path, err)
	}
	var manifest evalGateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return evalGateManifest{}, fmt.Errorf("unmarshal eval gate manifest %s: %w", path, err)
	}
	return manifest, nil
}

// evalGateByName returns the gate named name from manifest, or ok=false.
func evalGateByName(manifest evalGateManifest, name evalGateName) (evalGate, bool) {
	for _, g := range manifest.Gates {
		if g.Name == name {
			return g, true
		}
	}
	return evalGate{}, false
}

// --- Selection resolution ---

// evalGateTestNamePattern matches a top-level Test/Example/Benchmark
// function name, the same convention testing_main_test.go's
// discoverFullSuiteTests already uses.
var evalGateTestNamePattern = regexp.MustCompile(`^(Test|Example|Benchmark)[A-Za-z0-9_]*$`)

// evalGateSelectionTimeout bounds a single `go test -list` invocation.
var evalGateSelectionTimeout = 3 * time.Minute

// resolveEvalGateSelection lists the tests gate's own selection would run,
// via `go test -list`, rather than by executing them. An empty RunPattern
// means "all" (go test -list=.). A gate whose selection matches nothing
// returns a nil, non-erroring slice -- validateEvalGateManifest is the
// caller that turns an empty result into a refusal.
func resolveEvalGateSelection(g evalGate) ([]string, error) {
	root, err := evalGateRepoRoot()
	if err != nil {
		return nil, err
	}
	pattern := strings.TrimSpace(g.RunPattern)
	if pattern == "" {
		pattern = "."
	}
	packages := g.Packages
	if len(packages) == 0 {
		packages = []string{"./..."}
	}
	args := []string{"test", "-list=" + pattern}
	if len(g.BuildTags) > 0 {
		args = append(args, "-tags="+strings.Join(g.BuildTags, ","))
	}
	args = append(args, packages...)

	ctx, cancel := context.WithTimeout(context.Background(), evalGateSelectionTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, output)
	}

	var names []string
	for _, line := range strings.Split(string(output), "\n") {
		name := strings.TrimSpace(line)
		if evalGateTestNamePattern.MatchString(name) {
			names = append(names, name)
		}
	}
	return names, nil
}

// evalGateSelectionResolver is resolveEvalGateSelection, overridable by
// tests so vocabulary/budget-only test cases never pay for a real `go
// test -list` subprocess call.
var evalGateSelectionResolver = resolveEvalGateSelection

// --- Validation ---

// validateEvalGateManifest checks manifest against every rule the manifest
// itself must satisfy: the declared vocabulary matches exactly (both
// directions), every budget is positive, every requires value is declared,
// and every gate's own selection resolves to at least one real test.
// Returns a joined error naming every violation found, or nil.
func validateEvalGateManifest(manifest evalGateManifest) error {
	var violations []error

	seen := map[evalGateName]bool{}
	for _, g := range manifest.Gates {
		if seen[g.Name] {
			violations = append(violations, fmt.Errorf("eval gate manifest declares gate %q more than once", g.Name))
			continue
		}
		seen[g.Name] = true
	}

	for _, want := range evalGateNameVocabulary {
		if !seen[want] {
			violations = append(violations, fmt.Errorf(
				"eval gate %q is declared in the constants vocabulary but absent from the manifest", want,
			))
		}
	}
	for name := range seen {
		if !evalGateNameDeclared(name) {
			violations = append(violations, fmt.Errorf(
				"eval gate %q is present in the manifest but absent from the constants vocabulary", name,
			))
		}
	}

	for _, g := range manifest.Gates {
		if g.BudgetSeconds <= 0 {
			violations = append(violations, fmt.Errorf(
				"eval gate %q has a non-positive budget_seconds (%d)", g.Name, g.BudgetSeconds,
			))
		}
		if !evalGateRequirementDeclared(g.Requires) {
			violations = append(violations, fmt.Errorf(
				"eval gate %q declares requires %q, not in the declared vocabulary %v", g.Name, g.Requires, evalGateRequirementNames(),
			))
		}
	}

	for _, g := range manifest.Gates {
		names, err := evalGateSelectionResolver(g)
		if err != nil {
			violations = append(violations, fmt.Errorf("eval gate %q: resolve selection: %w", g.Name, err))
			continue
		}
		if len(names) == 0 {
			violations = append(violations, fmt.Errorf(
				"eval gate %q selection (packages=%v build_tags=%v run_pattern=%q) matches no test in the repository",
				g.Name, g.Packages, g.BuildTags, g.RunPattern,
			))
		}
	}

	if len(violations) == 0 {
		return nil
	}
	return errors.Join(violations...)
}

// --- Shared AST-based test-function index ---
//
// Resolves sentinels (Task 2) and fixture-bank guards (Task 3) against the
// real repository by parsing test function declarations, never by
// executing them -- the same discipline resolveEvalGateSelection already
// applies to gate selections.

// evalGateTestFunctionLocations returns the set of "<packageDir>::<TestName>"
// keys for every top-level Test/Example/Benchmark function declared in any
// *_test.go file under the repo-relative directories named by dirs.
func evalGateTestFunctionLocations(repoRoot string, dirs []string) (map[string]bool, error) {
	locations := map[string]bool{}
	for _, rel := range dirs {
		start := filepath.Join(repoRoot, rel)
		walkErr := filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, parseErr := parser.ParseFile(fset, path, nil, 0)
			if parseErr != nil {
				return nil
			}
			pkgDir, relErr := filepath.Rel(repoRoot, filepath.Dir(path))
			if relErr != nil {
				pkgDir = filepath.Dir(path)
			}
			pkgDir = filepath.ToSlash(pkgDir)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil {
					continue
				}
				if evalGateTestNamePattern.MatchString(fn.Name.Name) {
					locations[pkgDir+"::"+fn.Name.Name] = true
				}
			}
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return locations, nil
}

// evalGateFuncDeclByName parses every *_test.go file directly under dir
// (non-recursive) and returns the *ast.FuncDecl for the top-level function
// named name, or nil if not found.
func evalGateFuncDeclByName(dir, name string) (*ast.FuncDecl, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if parseErr != nil {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if fn.Name.Name == name {
				return fn, nil
			}
		}
	}
	return nil, nil
}

// evalGateFuncBodyHasSkipCall reports whether fn's body contains any call to
// t.Skip/t.Skipf/t.SkipNow (any receiver, since the *testing.T parameter
// name varies) -- a hard-gate sentinel is never allowed to carry one, so
// this deliberately does not attempt to distinguish a conditionally-guarded
// skip from an unconditional one: any skip capability at all defeats "may
// never be skipped."
func evalGateFuncBodyHasSkipCall(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Body == nil {
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
		switch sel.Sel.Name {
		case "Skip", "Skipf", "SkipNow":
			found = true
		}
		return true
	})
	return found
}
