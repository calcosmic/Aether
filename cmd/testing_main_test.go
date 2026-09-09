package cmd

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/calcosmic/Aether/pkg/trace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// TestMain saves all package-level globals before running tests and restores
// them after. This prevents test pollution where one test's store/stdout/stderr
// assignment leaks into subsequent tests. Belt-and-suspenders with per-test
// cleanup via saveGlobals.
func TestMain(m *testing.M) {
	extendDefaultCommandPackageTestTimeout()
	origOutputMode, hadOutputMode := os.LookupEnv("AETHER_OUTPUT_MODE")
	origHivePolicy, hadHivePolicy := os.LookupEnv(hivePolicyEnv)
	_ = os.Setenv("AETHER_OUTPUT_MODE", "json")
	_ = os.Setenv(hivePolicyEnv, "promote")

	// Spawn-line model tags resolve ANTHROPIC_DEFAULT_*_MODEL at render
	// time; a developer whose shell redirects those slots would otherwise
	// see every golden fixture with a spawn line fail. Neutralize for the
	// suite — tests that exercise the override use t.Setenv.
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_SONNET_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_OPUS_MODEL")
	_ = os.Unsetenv("ANTHROPIC_DEFAULT_HAIKU_MODEL")

	// Isolate the whole suite from the developer's real hub. This TestMain
	// enables hive promotion above, and the v1.25 multi-agent review found
	// TestSealPromoteInstincts had promoted its fixture into the user's actual
	// ~/.aether/hive/wisdom.json on every full-suite run. AETHER_HUB_DIR is
	// the supported override; individual tests may still t.Setenv their own.
	var testHubDir string
	if os.Getenv("AETHER_HUB_DIR") == "" {
		if dir, err := os.MkdirTemp("", "aether-test-hub-"); err == nil {
			testHubDir = dir
			_ = os.Setenv("AETHER_HUB_DIR", dir)
			// Make the isolated hub minimally valid so tests that check
			// "hub installed" (version.json present) behave as they would on
			// a machine with Aether installed — without touching the real one.
			// The version mirrors the source checkout's so stale-publish
			// detection sees binary and hub in agreement.
			hubVersion := `{"version":"0.0.0-test"}`
			if data, err := os.ReadFile(filepath.Join("..", ".aether", "version.json")); err == nil {
				hubVersion = string(data)
			}
			_ = os.WriteFile(filepath.Join(dir, "version.json"), []byte(hubVersion), 0644)
			_ = os.MkdirAll(filepath.Join(dir, "system"), 0755)
		}
	}

	origStore := store
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origTracer := tracer
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath

	code := m.Run()

	// Clean up git worktrees and branches created by tests.
	// Tests like TestWorktreeAllocateAuditLog create real git worktrees
	// that persist after the test suite finishes.
	cleanupTestWorktrees()

	store = origStore
	stdout = origStdout
	stderr = origStderr
	flagTypeFilter = origFlagType
	flagStatusFilter = origFlagStatus
	flagListJSON = origFlagListJSON
	historyJSON = origHistoryJSON
	phaseJSON = origPhaseJSON
	historyLimit = origHistoryLimit
	historyFilter = origHistoryFilter
	phaseNumber = origPhaseNumber
	tracer = origTracer
	continueContextUpdater = origContinueContextUpdater
	continueSignalHousekeeper = origContinueSignalHousekeeper
	newCodexWorkerInvoker = origNewCodexWorkerInvoker
	activeBuildCeremony = origActiveBuildCeremony
	narratorLookPath = origNarratorLookPath
	narratorCommandContext = origNarratorCommandContext
	narratorRuntimePath = origNarratorRuntimePath
	if hadOutputMode {
		_ = os.Setenv("AETHER_OUTPUT_MODE", origOutputMode)
	} else {
		_ = os.Unsetenv("AETHER_OUTPUT_MODE")
	}
	if hadHivePolicy {
		_ = os.Setenv(hivePolicyEnv, origHivePolicy)
	} else {
		_ = os.Unsetenv(hivePolicyEnv)
	}
	if testHubDir != "" {
		_ = os.Unsetenv("AETHER_HUB_DIR")
		_ = os.RemoveAll(testHubDir)
	}

	os.Exit(code)
}

// extendDefaultCommandPackageTestTimeout keeps the growing integration suite
// from reaching Go's stock ten-minute deadline before t.Parallel isolation
// wrappers are scheduled. Explicit non-default budgets (including every
// isolated child budget) remain untouched and therefore stay fail-closed.
func extendDefaultCommandPackageTestTimeout() {
	if !flag.Parsed() {
		flag.Parse()
	}
	timeoutFlag := flag.Lookup("test.timeout")
	if timeoutFlag == nil || timeoutFlag.Value.String() != (10*time.Minute).String() {
		return
	}
	_ = timeoutFlag.Value.Set((30 * time.Minute).String())
}

// saveGlobals captures the current values of all mutable package-level globals
// and restores them when the test completes. Every test that assigns to store,
// stdout, stderr, flagTypeFilter, or flagStatusFilter must call this as its
// first action.
func saveGlobals(t *testing.T) {
	t.Helper()
	origStdout := stdout
	origStderr := stderr
	origFlagType := flagTypeFilter
	origFlagStatus := flagStatusFilter
	origFlagListJSON := flagListJSON
	origHistoryJSON := historyJSON
	origPhaseJSON := phaseJSON
	origHistoryLimit := historyLimit
	origHistoryFilter := historyFilter
	origPhaseNumber := phaseNumber
	origContinueContextUpdater := continueContextUpdater
	origContinueSignalHousekeeper := continueSignalHousekeeper
	origNewCodexWorkerInvoker := newCodexWorkerInvoker
	origActiveBuildCeremony := activeBuildCeremony
	origNarratorLookPath := narratorLookPath
	origNarratorCommandContext := narratorCommandContext
	origNarratorRuntimePath := narratorRuntimePath
	origResumeNoHandoff := resumeNoHandoff
	origColonyPrimeTemplatesPathOverride := colonyPrimeTemplatesPathOverride
	t.Cleanup(func() {
		// A Store and its tracer are repository authorities, not ordinary test
		// values. Never resurrect one after its temporary repository may have
		// been deleted by another cleanup.
		store = nil
		stdout = origStdout
		stderr = origStderr
		flagTypeFilter = origFlagType
		flagStatusFilter = origFlagStatus
		flagListJSON = origFlagListJSON
		historyJSON = origHistoryJSON
		phaseJSON = origPhaseJSON
		historyLimit = origHistoryLimit
		historyFilter = origHistoryFilter
		phaseNumber = origPhaseNumber
		tracer = nil
		continueContextUpdater = origContinueContextUpdater
		continueSignalHousekeeper = origContinueSignalHousekeeper
		newCodexWorkerInvoker = origNewCodexWorkerInvoker
		activeBuildCeremony = origActiveBuildCeremony
		narratorLookPath = origNarratorLookPath
		narratorCommandContext = origNarratorCommandContext
		narratorRuntimePath = origNarratorRuntimePath
		resumeNoHandoff = origResumeNoHandoff
		colonyPrimeTemplatesPathOverride = origColonyPrimeTemplatesPathOverride
		resetColonyPrimeTemplatesCache()
	})
}

func forceJSONOutputModeForTest(t *testing.T) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", "json")
}

type commandTestRepository struct {
	Root    string
	DataDir string
	Store   *storage.Store
}

// bindCommandTestRepository gives command tests the same single physical
// authority production requires: AETHER_ROOT, COLONY_DATA_DIR, store, and
// tracer all point at one repository-local .aether/data tree. Cleanup is
// registered after TempDir so process globals are cleared before the directory
// is removed; testing.T.Setenv preserves absent-versus-present environment
// state exactly.
func bindCommandTestRepository(t *testing.T) commandTestRepository {
	t.Helper()
	return bindCommandTestRepositoryAt(t, t.TempDir())
}

// bindCommandTestRepositoryAt applies the command-test authority contract to
// an existing temporary repository. Worktree fixtures need this form because
// they create real Git state beneath the same root that owns .aether/data.
func bindCommandTestRepositoryAt(t *testing.T, repositoryRoot string) commandTestRepository {
	t.Helper()

	dataDir := filepath.Join(repositoryRoot, ".aether", "data")
	authority, err := storage.OpenRepositoryRoot(repositoryRoot, dataDir)
	if err != nil {
		t.Fatalf("open command-test repository authority: %v", err)
	}
	boundStore, err := storage.NewRepositoryStore(authority)
	if err != nil {
		_ = authority.Close()
		t.Fatalf("create command-test repository store: %v", err)
	}

	t.Setenv("AETHER_ROOT", repositoryRoot)
	t.Setenv("COLONY_DATA_DIR", dataDir)

	store = boundStore
	tracer = trace.NewTracer(boundStore)
	rootCmd.SetArgs([]string{})
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	resetFlags(rootCmd)

	t.Cleanup(func() {
		// This cleanup runs before Setenv restoration and TempDir deletion.
		// Do not restore a possibly stale repository authority here.
		store = nil
		tracer = nil
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		rootCmd.SetErr(os.Stderr)
		resetFlags(rootCmd)
		_ = authority.Close()
	})

	return commandTestRepository{
		Root:    repositoryRoot,
		DataDir: dataDir,
		Store:   boundStore,
	}
}

func selfRestoringAetherRootCleanupSites200(fileSet *token.FileSet, file *ast.File) []string {
	var sites []string
	ast.Inspect(file, func(node ast.Node) bool {
		deferred, ok := node.(*ast.DeferStmt)
		if !ok || deferred.Call == nil || len(deferred.Call.Args) != 2 {
			return true
		}
		setenv, ok := deferred.Call.Fun.(*ast.SelectorExpr)
		if !ok || setenv.Sel.Name != "Setenv" {
			return true
		}
		setenvPackage, ok := setenv.X.(*ast.Ident)
		if !ok || setenvPackage.Name != "os" {
			return true
		}
		name, ok := deferred.Call.Args[0].(*ast.BasicLit)
		if !ok || name.Kind != token.STRING || name.Value != `"AETHER_ROOT"` {
			return true
		}
		getenvCall, ok := deferred.Call.Args[1].(*ast.CallExpr)
		if !ok || len(getenvCall.Args) != 1 {
			return true
		}
		getenv, ok := getenvCall.Fun.(*ast.SelectorExpr)
		if !ok || getenv.Sel.Name != "Getenv" {
			return true
		}
		getenvPackage, ok := getenv.X.(*ast.Ident)
		if !ok || getenvPackage.Name != "os" {
			return true
		}
		getenvName, ok := getenvCall.Args[0].(*ast.BasicLit)
		if !ok || getenvName.Kind != token.STRING || getenvName.Value != `"AETHER_ROOT"` {
			return true
		}
		sites = append(sites, fileSet.Position(deferred.Pos()).String())
		return true
	})
	return sites
}

func TestNoSelfRestoringAetherRootCleanup200(t *testing.T) {
	const sample = `package cmd

import "os"

// defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT")) is explanatory prose.
const explanation = ` + "`" + `defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))` + "`" + `

func brokenCleanup() {
	defer os.Setenv("AETHER_ROOT", os.Getenv("AETHER_ROOT"))
}
`

	sampleSet := token.NewFileSet()
	sampleFile, err := parser.ParseFile(sampleSet, "sample_test.go", sample, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse in-memory cleanup sample: %v", err)
	}
	if sites := selfRestoringAetherRootCleanupSites200(sampleSet, sampleFile); len(sites) != 1 {
		t.Fatalf("active cleanup sample sites = %v, want exactly one executable defer", sites)
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read cmd test sources: %v", err)
	}
	var activeSites []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		fileSet := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fileSet, entry.Name(), nil, parser.ParseComments)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", entry.Name(), parseErr)
		}
		activeSites = append(activeSites, selfRestoringAetherRootCleanupSites200(fileSet, parsed)...)
	}
	sort.Strings(activeSites)
	if len(activeSites) != 0 {
		t.Fatalf("self-restoring AETHER_ROOT cleanup returned after the 120-site migration:\n%s", strings.Join(activeSites, "\n"))
	}
}

func TestRepositoryTestBinding200(t *testing.T) {
	originalRoot, hadRoot := os.LookupEnv("AETHER_ROOT")
	originalDataDir, hadDataDir := os.LookupEnv("COLONY_DATA_DIR")
	originalStore := store
	originalTracer := tracer
	defer func() {
		if hadRoot {
			_ = os.Setenv("AETHER_ROOT", originalRoot)
		} else {
			_ = os.Unsetenv("AETHER_ROOT")
		}
		if hadDataDir {
			_ = os.Setenv("COLONY_DATA_DIR", originalDataDir)
		} else {
			_ = os.Unsetenv("COLONY_DATA_DIR")
		}
		store = originalStore
		tracer = originalTracer
		resetFlags(rootCmd)
	}()

	_ = os.Unsetenv("AETHER_ROOT")
	_ = os.Unsetenv("COLONY_DATA_DIR")
	store = nil
	tracer = nil
	resetFlags(rootCmd)

	var repositoryRoot string
	t.Run("binds one contained authority", func(t *testing.T) {
		binding := bindCommandTestRepository(t)
		repositoryRoot = binding.Root

		if got := os.Getenv("AETHER_ROOT"); got != binding.Root {
			t.Fatalf("AETHER_ROOT = %q, want %q", got, binding.Root)
		}
		if got := os.Getenv("COLONY_DATA_DIR"); got != binding.DataDir {
			t.Fatalf("COLONY_DATA_DIR = %q, want %q", got, binding.DataDir)
		}
		if store != binding.Store {
			t.Fatalf("package store = %p, want bound store %p", store, binding.Store)
		}
		if tracer == nil {
			t.Fatal("package tracer was not bound with the repository store")
		}
		if got := filepath.Clean(binding.Store.BasePath()); got != filepath.Clean(binding.DataDir) {
			t.Fatalf("store base path = %q, want %q", got, binding.DataDir)
		}
		rel, err := filepath.Rel(binding.Root, binding.DataDir)
		if err != nil || rel != filepath.Join(".aether", "data") {
			t.Fatalf("data root is not the repository-local .aether/data path: rel=%q err=%v", rel, err)
		}

		rootCmd.SetArgs([]string{"stale-command"})
		rootCmd.SetOut(&strings.Builder{})
		rootCmd.SetErr(&strings.Builder{})
		if err := historyCmd.Flags().Set("limit", "999"); err != nil {
			t.Fatalf("mutate Cobra flag: %v", err)
		}
	})

	if _, configured := os.LookupEnv("AETHER_ROOT"); configured {
		t.Fatal("AETHER_ROOT was initially absent but cleanup left it present")
	}
	if _, configured := os.LookupEnv("COLONY_DATA_DIR"); configured {
		t.Fatal("COLONY_DATA_DIR was initially absent but cleanup left it present")
	}
	if store != nil || tracer != nil {
		t.Fatalf("bootstrap globals survived cleanup: store=%p tracer=%p", store, tracer)
	}
	if got := rootCmd.OutOrStdout(); got != os.Stdout {
		t.Fatalf("Cobra stdout was not reset: got %T", got)
	}
	if got := rootCmd.ErrOrStderr(); got != os.Stderr {
		t.Fatalf("Cobra stderr was not reset: got %T", got)
	}
	if historyLimit != 20 {
		t.Fatalf("Cobra flag state survived cleanup: history limit = %d, want 20", historyLimit)
	}
	if _, err := os.Stat(repositoryRoot); !os.IsNotExist(err) {
		t.Fatalf("temporary repository still exists after cleanup: %v", err)
	}
}

func isPermissionDeniedForTest(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "operation not permitted") || strings.Contains(text, "permission denied")
}

// resetRootCmd restores rootCmd state (SetArgs, SetOut) and resets all
// subcommand local flags to their defaults. Cobra local flags persist across
// Execute() calls, so without this, --type blocker on flag-add would leak
// into the next test that runs flag-add without --type.
// Every test that calls rootCmd.Execute must call this early in the function.
func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.SetOut(os.Stdout)
		// Reset local flags on all subcommands to their defaults.
		// This prevents flag value leakage between tests.
		resetFlags(rootCmd)
	})
}

// resetFlags recursively resets all local flags on cmd and its subcommands
// to their default values. This prevents Cobra flag leakage between tests.
func resetFlags(cmd *cobra.Command) {
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if sliceValue, ok := f.Value.(pflag.SliceValue); ok {
			if f.DefValue == "" || f.DefValue == "[]" {
				_ = sliceValue.Replace(nil)
			}
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetFlags(sub)
	}
}

type testOwnedWorktree struct {
	repoRoot string
	path     string
	branch   string
}

var testOwnedWorktrees struct {
	sync.Mutex
	entries []testOwnedWorktree
}

// registerTestOwnedWorktree records positive ownership before a test creates a
// real worktree. The registry is process-local, so only this test run can grant
// cleanup permission for an exact repository, path, and branch.
func registerTestOwnedWorktree(t *testing.T, repoRoot, path, branch string) {
	t.Helper()
	entry, err := validateTestOwnedWorktree(repoRoot, path, branch)
	if err != nil {
		t.Fatalf("register test-owned worktree: %v", err)
	}
	testOwnedWorktrees.Lock()
	testOwnedWorktrees.entries = append(testOwnedWorktrees.entries, entry)
	testOwnedWorktrees.Unlock()
}

// cleanupTestWorktrees removes only worktrees and branches that this test
// process registered explicitly. Missing artifacts are harmless: cleanup is
// deliberately safe to replay.
func cleanupTestWorktrees() {
	testOwnedWorktrees.Lock()
	entries := append([]testOwnedWorktree(nil), testOwnedWorktrees.entries...)
	testOwnedWorktrees.entries = nil
	testOwnedWorktrees.Unlock()

	for _, registered := range entries {
		entry, err := validateTestOwnedWorktree(registered.repoRoot, registered.path, registered.branch)
		if err != nil {
			continue
		}
		_ = exec.Command("git", "-C", entry.repoRoot, "worktree", "remove", "--force", entry.path).Run()
		_ = exec.Command("git", "-C", entry.repoRoot, "branch", "-D", "--", entry.branch).Run()
	}
}

func validateTestOwnedWorktree(repoRoot, path, branch string) (testOwnedWorktree, error) {
	root, err := canonicalTestRepositoryRoot(repoRoot)
	if err != nil {
		return testOwnedWorktree{}, err
	}
	if strings.TrimSpace(path) == "" {
		return testOwnedWorktree{}, fmt.Errorf("worktree path is required")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	resolvedPath, err := resolveTestOwnedPath(path)
	if err != nil {
		return testOwnedWorktree{}, fmt.Errorf("resolve worktree path: %w", err)
	}
	relativePath, err := filepath.Rel(root, resolvedPath)
	if err != nil || relativePath == "." || relativePath == "" || filepath.IsAbs(relativePath) || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return testOwnedWorktree{}, fmt.Errorf("worktree path %q is not beneath repository %q", resolvedPath, root)
	}
	if branch == "" || branch != strings.TrimSpace(branch) {
		return testOwnedWorktree{}, fmt.Errorf("branch name is required")
	}
	if output, err := exec.Command("git", "check-ref-format", "--branch", branch).CombinedOutput(); err != nil {
		return testOwnedWorktree{}, fmt.Errorf("invalid branch %q: %s", branch, strings.TrimSpace(string(output)))
	}
	return testOwnedWorktree{repoRoot: root, path: resolvedPath, branch: branch}, nil
}

func canonicalTestRepositoryRoot(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("repository root is required")
	}
	absoluteRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", fmt.Errorf("make repository root absolute: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	info, err := os.Stat(resolvedRoot)
	if err != nil {
		return "", fmt.Errorf("stat repository root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root %q is not a directory", resolvedRoot)
	}
	topLevelOutput, err := exec.Command("git", "-C", resolvedRoot, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("validate repository root: %w", err)
	}
	topLevel, err := filepath.EvalSymlinks(strings.TrimSpace(string(topLevelOutput)))
	if err != nil {
		return "", fmt.Errorf("resolve git top-level: %w", err)
	}
	if filepath.Clean(topLevel) != filepath.Clean(resolvedRoot) {
		return "", fmt.Errorf("repository root %q is not git top-level %q", resolvedRoot, topLevel)
	}
	return filepath.Clean(resolvedRoot), nil
}

// resolveTestOwnedPath resolves every existing path component, then appends
// any not-yet-created suffix. This catches symlink escapes while still allowing
// registration before git creates the worktree directory.
func resolveTestOwnedPath(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	candidate := filepath.Clean(absolutePath)
	var missing []string
	for {
		if _, err := os.Lstat(candidate); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(candidate)
			if resolveErr != nil {
				return "", resolveErr
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", fmt.Errorf("no existing ancestor for %q", absolutePath)
		}
		missing = append(missing, filepath.Base(candidate))
		candidate = parent
	}
}
