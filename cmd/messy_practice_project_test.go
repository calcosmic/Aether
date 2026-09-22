package cmd

// 207-02-PLAN.md Task 2 (UED-07): one named, genuinely failable check per
// declared trap, plus the builder script's own edge behaviour (clean twice,
// refuses a foreign directory, requires a destination).
//
// Every subtest is driven off loadJourneyTraps() -- the declared manifest --
// never a re-typed id list, so the manifest and this file can never
// silently drift apart (mirrors TestMessyPracticeProjectTrapListAndScriptAgree's
// own check that the manifest and the builder script agree).
//
// Every assertion resolves to a filesystem or git fact, or a call into the
// exact runtime function the trap exercises (classifySurveyFreshness,
// specPublicPathLineage, entombArchiveKey, planningRunIsSuperseded,
// middenGuidedAction's own MiddenFile shape) -- never the builder script's
// own printed output (CLAUDE.md: "never assert on prose").

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- Shared, once-built fixture -------------------------------------------
//
// Building the practice project runs a real `go build ./cmd/aether` plus a
// real `aether update --force` plus every trap's own construction (including
// the specification-correction trap's several sequential specification
// mutations and the stale-survey trap's 27 churn
// commits) -- expensive enough that rebuilding it per subtest would make
// this file unusable as fast feedback. Built ONCE for the whole package test
// run via sync.Once and reused by every test function below; only the two
// tests that must observe the builder's OWN process behaviour (running it
// twice from scratch, pointing it at a foreign directory) pay for their own
// additional build, and both are skipped under -short.
//
// Deliberately NOT cleaned up via TestMain: cmd/testing_main_test.go already
// owns the package's one TestMain, and it does not expose a registration
// hook for additional cleanup. Leaving one bounded temp directory tree
// behind is the same tradeoff scripts/proof-screens-reach-the-owner.sh's own
// $WORK directory makes when a script is interrupted before its own `trap
// ... EXIT` fires; an OS temp directory is reclaimed by the machine over
// time regardless.
var (
	journeyTestSharedOnce sync.Once
	journeyTestSharedBin  string
	journeyTestSharedDest string
	journeyTestSharedErr  error
)

// journeyTestRepoRoot resolves the real Aether module root -- the same
// resolution journeyTrapsRepoRoot uses when no test override is set.
func journeyTestRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := journeyTrapsRepoRoot()
	if err != nil {
		t.Fatalf("resolve Aether module root: %v", err)
	}
	return root
}

// journeyTestLoadTraps returns the declared traps in file order. Every test
// in this file resolves trap identity through this one call.
func journeyTestLoadTraps(t *testing.T) []journeyTrap {
	t.Helper()
	file, err := loadJourneyTraps()
	if err != nil {
		t.Fatalf("load declared trap manifest: %v", err)
	}
	if len(file.Traps) == 0 {
		t.Fatal("declared trap manifest has no traps")
	}
	return file.Traps
}

// journeyTestBuilderScript resolves the committed builder script's absolute
// path.
func journeyTestBuilderScript(t *testing.T) string {
	t.Helper()
	return filepath.Join(journeyTestRepoRoot(t), "scripts", "build-messy-practice-project.sh")
}

// journeyTestSharedProject builds the aether binary and one base practice
// project once, returning the binary path and the destination directory
// (whose own "repo" subdirectory is the practice project itself).
func journeyTestSharedProject(t *testing.T) (bin, dest string) {
	t.Helper()
	journeyTestSharedOnce.Do(func() {
		root := journeyTestRepoRoot(t)
		tmp, err := os.MkdirTemp("", "messy-practice-project-shared-")
		if err != nil {
			journeyTestSharedErr = fmt.Errorf("create shared temp dir: %w", err)
			return
		}
		builtBin := filepath.Join(tmp, "aether-under-test")
		// -tags=journey: the hidden journey-seed-* trap constructors the
		// builder script below calls are excluded from the default build
		// (CR-01, 207-REVIEW.md) and only compile in under this tag.
		buildCmd := exec.Command("go", "build", "-tags=journey", "-o", builtBin, "./cmd/aether")
		buildCmd.Dir = root
		if out, err := buildCmd.CombinedOutput(); err != nil {
			journeyTestSharedErr = fmt.Errorf("go build ./cmd/aether: %w\n%s", err, out)
			return
		}
		builtDest := filepath.Join(tmp, "dest")
		script := filepath.Join(root, "scripts", "build-messy-practice-project.sh")
		runCmd := exec.Command(script, builtDest, "--aether-bin", builtBin)
		if out, err := runCmd.CombinedOutput(); err != nil {
			journeyTestSharedErr = fmt.Errorf("build practice project: %w\n%s", err, out)
			return
		}
		journeyTestSharedBin = builtBin
		journeyTestSharedDest = builtDest
	})
	if journeyTestSharedErr != nil {
		t.Fatalf("shared practice project setup failed: %v", journeyTestSharedErr)
	}
	return journeyTestSharedBin, journeyTestSharedDest
}

// journeyTestCloneRepo copies the shared project's repo directory (symlinks
// preserved, never dereferenced -- `cp -R` on both BSD and GNU coreutils
// copies a symbolic link as a link unless -L/--dereference is given) into a
// fresh, disposable location a test may safely mutate without disturbing the
// shared fixture other tests read.
func journeyTestCloneRepo(t *testing.T, srcRepo string) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "repo")
	if out, err := exec.Command("cp", "-R", srcRepo, dst).CombinedOutput(); err != nil {
		t.Fatalf("clone practice project from %s: %v\n%s", srcRepo, err, out)
	}
	return dst
}

// --- Per-trap checkers ------------------------------------------------------
//
// Pure functions (repo, trap) -> error, keyed by the assertion's Kind (never
// by trap id -- see cmd/journey_traps.go's own doc comment on why Kind is a
// separate vocabulary). Kept as plain functions, not methods on *testing.T,
// so TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder can collect
// an ordered pass/fail list without a failing subtest also failing its
// parent (Go's t.Run always propagates a subtest failure to its parent;
// this file needs to deliberately produce two failures and still report the
// ordering test itself as green).

type journeyTrapChecker func(repo string, trap journeyTrap) error

var journeyTrapCheckers = map[string]journeyTrapChecker{
	"symlink_to_directory":  checkSymlinkToDirectory,
	"symlink_loop":          checkSymlinkLoop,
	"case_only_collision":   checkCaseOnlyCollision,
	"nested_git_repo":       checkNestedGitRepo,
	"long_lineage":          checkLongLineage,
	"stale_survey":          checkStaleSurvey,
	"superseded_plan":       checkSupersededPlan,
	"midden_unacknowledged": checkMiddenUnacknowledged,
	"dirty_worktree":        checkDirtyWorktree,
}

func gitStatusPorcelain(repo string, extraArgs ...string) (string, error) {
	args := append([]string{"-C", repo, "status", "--porcelain"}, extraArgs...)
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git status: %w\n%s", err, out)
	}
	return string(out), nil
}

// checkSymlinkToDirectory asserts a real symbolic link, resolving to a real
// directory inside the project, left as an unsaved (uncommitted) change --
// the folder-shortcut trap.
func checkSymlinkToDirectory(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" {
		return fmt.Errorf("trap %q declares no assertion path", trap.ID)
	}
	full := filepath.Join(repo, trap.Assertion.Path)
	info, err := os.Lstat(full)
	if err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.Path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("%s is not a symbolic link", trap.Assertion.Path)
	}
	target, err := os.Readlink(full)
	if err != nil {
		return fmt.Errorf("%s: read symlink target: %w", trap.Assertion.Path, err)
	}
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(filepath.Dir(full), target)
	}
	targetInfo, err := os.Stat(resolved)
	if err != nil {
		return fmt.Errorf("%s: symlink target %q does not resolve: %w", trap.Assertion.Path, target, err)
	}
	if !targetInfo.IsDir() {
		return fmt.Errorf("%s: symlink target %q is not a directory", trap.Assertion.Path, target)
	}
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return err
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return err
	}
	if absResolved != absRepo && !strings.HasPrefix(absResolved, absRepo+string(filepath.Separator)) {
		return fmt.Errorf("%s: symlink target %q resolves outside the project", trap.Assertion.Path, target)
	}
	status, err := gitStatusPorcelain(repo, "--", trap.Assertion.Path)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("%s is committed, not left as an unsaved change", trap.Assertion.Path)
	}
	return nil
}

// checkSymlinkLoop asserts two symbolic links that point at each other, and
// that following the loop terminates within a bounded walk instead of
// hanging -- the shortcut-cycle trap (a robustness net, not tied to one of
// the six 2026-09-21 blockers).
func checkSymlinkLoop(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" || trap.Assertion.SecondPath == "" {
		return fmt.Errorf("trap %q declares fewer than two assertion paths", trap.ID)
	}
	a := filepath.Join(repo, trap.Assertion.Path)
	b := filepath.Join(repo, trap.Assertion.SecondPath)
	for _, p := range []string{a, b} {
		info, err := os.Lstat(p)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("%s is not a symbolic link", p)
		}
	}
	targetA, err := os.Readlink(a)
	if err != nil {
		return err
	}
	targetB, err := os.Readlink(b)
	if err != nil {
		return err
	}
	if filepath.Base(targetA) != filepath.Base(b) || filepath.Base(targetB) != filepath.Base(a) {
		return fmt.Errorf("%s and %s do not point at each other (targets %q, %q)", trap.Assertion.Path, trap.Assertion.SecondPath, targetA, targetB)
	}
	visited := make(map[string]bool)
	current := a
	for i := 0; i < 8; i++ {
		if visited[current] {
			return nil
		}
		visited[current] = true
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
		target, err := os.Readlink(current)
		if err != nil {
			return err
		}
		current = filepath.Join(filepath.Dir(current), target)
	}
	return fmt.Errorf("following the symlink loop from %s did not terminate within a bounded walk", trap.Assertion.Path)
}

// checkCaseOnlyCollision asserts two distinct real files whose ARCHIVE
// basenames (the exact identity entombArchiveKey compares -- the real
// classifier this trap must trip) collide only by letter case -- the
// case-colliding-archive-names trap.
func checkCaseOnlyCollision(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" || trap.Assertion.SecondPath == "" {
		return fmt.Errorf("trap %q declares fewer than two assertion paths", trap.ID)
	}
	p1 := filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path))
	p2 := filepath.Join(repo, filepath.FromSlash(trap.Assertion.SecondPath))
	info1, err := os.Lstat(p1)
	if err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.Path, err)
	}
	info2, err := os.Lstat(p2)
	if err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.SecondPath, err)
	}
	if !info1.Mode().IsRegular() || !info2.Mode().IsRegular() {
		return fmt.Errorf("%s and %s must both be regular files", trap.Assertion.Path, trap.Assertion.SecondPath)
	}
	if os.SameFile(info1, info2) {
		return fmt.Errorf("%s and %s are the same file, not two distinct files", trap.Assertion.Path, trap.Assertion.SecondPath)
	}
	key1 := entombArchiveKey(filepath.Base(trap.Assertion.Path))
	key2 := entombArchiveKey(filepath.Base(trap.Assertion.SecondPath))
	if key1 != key2 {
		return fmt.Errorf("archive keys for %s and %s do not collide: %q vs %q", trap.Assertion.Path, trap.Assertion.SecondPath, key1, key2)
	}
	if filepath.Base(trap.Assertion.Path) == filepath.Base(trap.Assertion.SecondPath) {
		return fmt.Errorf("%s and %s must differ only by letter case, not be byte-identical basenames", trap.Assertion.Path, trap.Assertion.SecondPath)
	}
	return nil
}

// checkNestedGitRepo asserts a subdirectory that is its own git repository,
// with its own commit, reported as untracked by the outer repository's own
// git status -- the nested-repository trap.
func checkNestedGitRepo(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" {
		return fmt.Errorf("trap %q declares no assertion path", trap.ID)
	}
	nested := filepath.Join(repo, trap.Assertion.Path)
	info, err := os.Stat(filepath.Join(nested, ".git"))
	if err != nil {
		return fmt.Errorf("%s/.git: %w", trap.Assertion.Path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s/.git is not a directory", trap.Assertion.Path)
	}
	out, err := exec.Command("git", "-C", nested, "log", "-1", "--format=%H").CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("%s has no commit of its own: %v %s", trap.Assertion.Path, err, out)
	}
	status, err := gitStatusPorcelain(repo, "--", trap.Assertion.Path)
	if err != nil {
		return err
	}
	if !strings.Contains(status, trap.Assertion.Path) {
		return fmt.Errorf("the outer repository's git status does not report %s as untracked: %q", trap.Assertion.Path, status)
	}
	return nil
}

// checkLongLineage asserts a real source file exists, reachable only through
// a directory chain long enough that specPublicPathLineage -- the real
// function the 2026-09-21 fix touches -- falls back to a digest ID rather
// than the readable "known-public-path-<path>" form, proving the normalized
// lineage genuinely exceeded 40 canonical characters -- the deep-directory-chain
// trap.
func checkLongLineage(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" {
		return fmt.Errorf("trap %q declares no assertion path", trap.ID)
	}
	full := filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path))
	if _, err := os.Stat(full); err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.Path, err)
	}
	id := specPublicPathLineage(trap.Assertion.Path)
	readable := specPublicPathLineagePrefix + trap.Assertion.Path
	if id == readable {
		return fmt.Errorf("specPublicPathLineage(%q) = %q did not fall back to a digest -- the normalized lineage did not exceed 40 characters", trap.Assertion.Path, id)
	}
	return nil
}

// checkStaleSurvey asserts classifySurveyFreshness -- the real classifier,
// never eyeballed dates -- reports Stale for the built project, via a
// genuine commit-count expiry (source_revision_changed AND
// source_revision_expired) -- the stale-survey-snapshot trap.
func checkStaleSurvey(repo string, trap journeyTrap) error {
	result := classifySurveyFreshness(repo, time.Now().UTC())
	if result.Freshness != colony.SurveyFreshnessStale {
		return fmt.Errorf("classifySurveyFreshness(%s) reported %v, want Stale (reasons: %v)", repo, result.Freshness, result.ReasonCodes)
	}
	hasChanged, hasExpired := false, false
	for _, code := range result.ReasonCodes {
		switch code {
		case surveyReasonSourceRevisionChanged:
			hasChanged = true
		case surveyReasonSourceRevisionExpired:
			hasExpired = true
		}
	}
	if !hasChanged || !hasExpired {
		return fmt.Errorf("stale reasons %v do not include both source_revision_changed and source_revision_expired", result.ReasonCodes)
	}
	return nil
}

// checkSupersededPlan asserts the parked planning run's own stage-state.json
// is genuinely awaiting a worker (planningStageAwaitsAWorker) and is
// genuinely superseded (planningRunIsSuperseded) against the specification's
// CURRENT approved revision -- both the real predicates
// discoverParkedPlanningRun itself calls -- the
// specification-correction trap.
func checkSupersededPlan(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" {
		return fmt.Errorf("trap %q declares no assertion path (expected the parked run's ID)", trap.ID)
	}
	runID := trap.Assertion.Path
	state, err := loadPlanningStageStateWithSession(repo, runID, nil)
	if err != nil {
		return fmt.Errorf("load parked planning run state %q: %w", runID, err)
	}
	if !planningStageAwaitsAWorker(nil, state) {
		return fmt.Errorf("parked planning run %q is not genuinely awaiting a worker (stage %q)", runID, state.Stage)
	}
	colonyState, err := loadSpecificationColonyState(repo)
	if err != nil {
		return fmt.Errorf("load colony state: %w", err)
	}
	approved, err := requireApprovedPlanningSpecification(repo, colonyState)
	if err != nil {
		return fmt.Errorf("resolve current approved specification: %w", err)
	}
	if !planningRunIsSuperseded(state, approved.Binding) {
		return fmt.Errorf("parked planning run %q is not superseded by the current approved specification", runID)
	}
	return nil
}

// checkMiddenUnacknowledged asserts the colony's failure log has at least
// one unacknowledged entry AND the colony's steering-signal log has at least
// one signal still marked active past its own expiry -- the
// leftover-junk trap (extended by this plan to also cover the
// pheromone log).
func checkMiddenUnacknowledged(repo string, trap journeyTrap) error {
	if trap.Assertion.Path == "" || trap.Assertion.SecondPath == "" {
		return fmt.Errorf("trap %q declares fewer than two assertion paths", trap.ID)
	}
	middenPath := filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path))
	body, err := os.ReadFile(middenPath)
	if err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.Path, err)
	}
	var mf colony.MiddenFile
	if err := json.Unmarshal(body, &mf); err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.Path, err)
	}
	unacked := 0
	for _, entry := range mf.Entries {
		if entry.Acknowledged == nil || !*entry.Acknowledged {
			unacked++
		}
	}
	if unacked == 0 {
		return fmt.Errorf("%s has no unacknowledged entries", trap.Assertion.Path)
	}

	pheromonesPath := filepath.Join(repo, filepath.FromSlash(trap.Assertion.SecondPath))
	body2, err := os.ReadFile(pheromonesPath)
	if err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.SecondPath, err)
	}
	var pf struct {
		Signals []struct {
			Active    bool   `json:"active"`
			ExpiresAt string `json:"expires_at"`
		} `json:"signals"`
	}
	if err := json.Unmarshal(body2, &pf); err != nil {
		return fmt.Errorf("%s: %w", trap.Assertion.SecondPath, err)
	}
	now := time.Now().UTC()
	stray := false
	for _, signal := range pf.Signals {
		if !signal.Active {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, signal.ExpiresAt)
		if err == nil && expiresAt.Before(now) {
			stray = true
			break
		}
	}
	if !stray {
		return fmt.Errorf("%s has no active signal past its own expiry", trap.Assertion.SecondPath)
	}
	return nil
}

// checkDirtyWorktree asserts git status --porcelain is non-empty and
// includes BOTH a modified tracked path and an untracked path -- the
// dirty-worktree trap.
func checkDirtyWorktree(repo string, trap journeyTrap) error {
	status, err := gitStatusPorcelain(repo, "--untracked-files=all")
	if err != nil {
		return err
	}
	hasModifiedTracked, hasUntracked := false, false
	for _, line := range strings.Split(strings.TrimRight(status, "\n"), "\n") {
		if len(line) < 3 {
			continue
		}
		code := line[:2]
		if code == "??" {
			hasUntracked = true
			continue
		}
		if strings.ContainsAny(code, "MADRC") {
			hasModifiedTracked = true
		}
	}
	if !hasModifiedTracked || !hasUntracked {
		return fmt.Errorf("git status --porcelain does not show both a modified tracked path and an untracked path:\n%s", status)
	}
	return nil
}

// --- Breaking a trap (for the "must be able to fail" and declared-order
// tests) --------------------------------------------------------------------

// breakJourneyTrap mutates a cloned copy of the practice project so the
// named trap's own evidence no longer holds -- used only by
// TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder, and never
// applied to the shared fixture other tests read.
func breakJourneyTrap(t *testing.T, repo string, trap journeyTrap) {
	t.Helper()
	switch trap.Assertion.Kind {
	case "symlink_to_directory", "long_lineage", "case_only_collision":
		if trap.Assertion.Path == "" {
			t.Fatalf("breakJourneyTrap: trap %q has no assertion path", trap.ID)
		}
		if err := os.RemoveAll(filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path))); err != nil {
			t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
		}
	case "symlink_loop":
		for _, p := range []string{trap.Assertion.Path, trap.Assertion.SecondPath} {
			if err := os.RemoveAll(filepath.Join(repo, filepath.FromSlash(p))); err != nil {
				t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
			}
		}
	case "nested_git_repo":
		if err := os.RemoveAll(filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path), ".git")); err != nil {
			t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
		}
	case "stale_survey":
		if err := os.RemoveAll(filepath.Join(repo, filepath.FromSlash(territorySnapshotRelativePath))); err != nil {
			t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
		}
	case "superseded_plan":
		statePath := filepath.Join(repo, filepath.FromSlash(planningStageStateRepositoryPath(trap.Assertion.Path)))
		if err := os.RemoveAll(statePath); err != nil {
			t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
		}
	case "midden_unacknowledged":
		middenPath := filepath.Join(repo, filepath.FromSlash(trap.Assertion.Path))
		if err := os.WriteFile(middenPath, []byte(`{"version":"1","signals":[],"entries":[]}`), 0o644); err != nil {
			t.Fatalf("breakJourneyTrap %q: %v", trap.ID, err)
		}
	case "dirty_worktree":
		// Commit ONLY the one modified tracked file (never `git add -A`,
		// which would also sweep up and commit the unrelated
		// folder-shortcut symlink and the nested repository directory,
		// silently breaking those two traps as a side effect).
		if out, err := exec.Command("git", "-C", repo, "add", "README.md").CombinedOutput(); err != nil {
			t.Fatalf("breakJourneyTrap %q: git add: %v\n%s", trap.ID, err, out)
		}
		if out, err := exec.Command("git", "-C", repo, "commit", "-q", "-m", "journey trap test: force the tracked file clean").CombinedOutput(); err != nil {
			t.Fatalf("breakJourneyTrap %q: git commit: %v\n%s", trap.ID, err, out)
		}
	default:
		t.Fatalf("breakJourneyTrap: no breaker registered for kind %q (trap %q)", trap.Assertion.Kind, trap.ID)
	}
}

// --- Tests -------------------------------------------------------------

// TestMessyPracticeProjectHasEveryTrap builds the practice project once and
// runs one named, genuinely failable subtest per declared trap, in declared
// file order.
func TestMessyPracticeProjectHasEveryTrap(t *testing.T) {
	_, dest := journeyTestSharedProject(t)
	repo := filepath.Join(dest, "repo")
	traps := journeyTestLoadTraps(t)

	for _, trap := range traps {
		trap := trap
		t.Run(trap.ID, func(t *testing.T) {
			checker, ok := journeyTrapCheckers[trap.Assertion.Kind]
			if !ok {
				t.Fatalf("no checker registered for kind %q (trap %q)", trap.Assertion.Kind, trap.ID)
			}
			if err := checker(repo, trap); err != nil {
				t.Fatalf("%s: %v", trap.ID, err)
			}
		})
	}
}

// TestMessyPracticeProjectTrapListAndScriptAgree fails, naming the id, if
// the declared manifest holds a trap the builder script has no case label
// for, or if the script's own case block builds an id the manifest does not
// declare. A purely static check -- no build required.
func TestMessyPracticeProjectTrapListAndScriptAgree(t *testing.T) {
	scriptPath := journeyTestBuilderScript(t)
	body, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read builder script: %v", err)
	}
	script := string(body)

	const marker = `case "$trap_id" in`
	start := strings.Index(script, marker)
	if start < 0 {
		t.Fatal("builder script has no `case \"$trap_id\" in` block")
	}
	esacPattern := regexp.MustCompile(`(?m)^\s*esac\s*$`)
	esacLoc := esacPattern.FindStringIndex(script[start:])
	if esacLoc == nil {
		t.Fatal("builder script's trap_id case block is never closed with `esac`")
	}
	block := script[start : start+esacLoc[0]]

	labelPattern := regexp.MustCompile(`(?m)^\s{4}([a-z][a-z0-9-]*)\)\s*$`)
	scriptIDs := make(map[string]bool)
	for _, match := range labelPattern.FindAllStringSubmatch(block, -1) {
		scriptIDs[match[1]] = true
	}
	if len(scriptIDs) == 0 {
		t.Fatal("found no trap case labels inside the trap_id case block -- the label pattern may need updating")
	}

	traps := journeyTestLoadTraps(t)
	declaredIDs := make(map[string]bool, len(traps))
	for _, trap := range traps {
		declaredIDs[trap.ID] = true
		if !scriptIDs[trap.ID] {
			t.Errorf("declared trap %q has no matching case label in the builder script -- the script cannot build it", trap.ID)
		}
	}
	for id := range scriptIDs {
		if !declaredIDs[id] {
			t.Errorf("the builder script builds %q, which the trap manifest does not declare", id)
		}
	}
}

// TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder breaks the
// first and fifth declared traps (chosen by position, never by re-typing a
// literal id) on a disposable clone of the shared project, then runs every
// checker in declared file order and asserts the two failures are collected
// in exactly that order -- proving the ordering comes from iterating the
// manifest's own slice, never from a map.
func TestMessyPracticeProjectTrapFailuresFollowTheDeclaredOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("clones and mutates a full practice project build; skipped under -short")
	}
	_, dest := journeyTestSharedProject(t)
	repo := journeyTestCloneRepo(t, filepath.Join(dest, "repo"))
	traps := journeyTestLoadTraps(t)
	if len(traps) < 5 {
		t.Fatalf("expected at least 5 declared traps to exercise this test, found %d", len(traps))
	}

	breakJourneyTrap(t, repo, traps[0])
	breakJourneyTrap(t, repo, traps[4])

	var failedInOrder []string
	for _, trap := range traps {
		checker, ok := journeyTrapCheckers[trap.Assertion.Kind]
		if !ok {
			t.Fatalf("no checker registered for kind %q (trap %q)", trap.Assertion.Kind, trap.ID)
		}
		if err := checker(repo, trap); err != nil {
			failedInOrder = append(failedInOrder, trap.ID)
		}
	}

	want := []string{traps[0].ID, traps[4].ID}
	if !reflect.DeepEqual(failedInOrder, want) {
		t.Fatalf("expected exactly these two traps to fail, in this declared order: %v; got: %v", want, failedInOrder)
	}
}

// TestMessyPracticeProjectBuilderRunsCleanTwice builds into a fresh
// destination twice and asserts both runs exit 0 and the marker file names
// all declared trap ids, in declared order, after each run.
func TestMessyPracticeProjectBuilderRunsCleanTwice(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a full practice project twice; skipped under -short")
	}
	bin, _ := journeyTestSharedProject(t)
	script := journeyTestBuilderScript(t)
	dest := filepath.Join(t.TempDir(), "dest")

	traps := journeyTestLoadTraps(t)
	wantIDs := make([]string, 0, len(traps))
	for _, trap := range traps {
		wantIDs = append(wantIDs, trap.ID)
	}

	for run := 1; run <= 2; run++ {
		out, err := exec.Command(script, dest, "--aether-bin", bin).CombinedOutput()
		if err != nil {
			t.Fatalf("run %d: builder script failed: %v\n%s", run, err, out)
		}
		markerPath := filepath.Join(dest, ".journey-practice-project.json")
		body, err := os.ReadFile(markerPath)
		if err != nil {
			t.Fatalf("run %d: read marker: %v", run, err)
		}
		var marker struct {
			TrapIDs []string `json:"trap_ids"`
		}
		if err := json.Unmarshal(body, &marker); err != nil {
			t.Fatalf("run %d: parse marker: %v", run, err)
		}
		if !reflect.DeepEqual(marker.TrapIDs, wantIDs) {
			t.Fatalf("run %d: marker trap_ids = %v, want %v", run, marker.TrapIDs, wantIDs)
		}
	}
}

// TestMessyPracticeProjectBuilderRefusesAForeignDirectory points the builder
// at a directory holding one unrelated file (no marker from a prior run of
// this script) and asserts a non-zero exit naming the destination, with the
// unrelated file untouched.
func TestMessyPracticeProjectBuilderRefusesAForeignDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("would otherwise build a full practice project if the refusal regressed; skipped under -short")
	}
	script := journeyTestBuilderScript(t)
	dest := t.TempDir()
	foreign := filepath.Join(dest, "unrelated.txt")
	original := []byte("this file was not created by build-messy-practice-project.sh\n")
	if err := os.WriteFile(foreign, original, 0o644); err != nil {
		t.Fatalf("write unrelated fixture file: %v", err)
	}

	out, err := exec.Command(script, dest).CombinedOutput()
	if err == nil {
		t.Fatalf("expected the builder to refuse a foreign directory, but it exited 0:\n%s", out)
	}
	if !strings.Contains(string(out), dest) {
		t.Fatalf("refusal output does not name the destination %q:\n%s", dest, out)
	}
	after, err := os.ReadFile(foreign)
	if err != nil {
		t.Fatalf("unrelated file vanished after the refused build: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("unrelated file was modified by the refused build: got %q, want %q", after, original)
	}
}

// TestMessyPracticeProjectBuilderRequiresADestination asserts that with no
// destination argument the script exits non-zero and creates nothing.
func TestMessyPracticeProjectBuilderRequiresADestination(t *testing.T) {
	script := journeyTestBuilderScript(t)
	dest := t.TempDir()
	before, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("read test working directory: %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("test setup: expected an empty working directory, found %d entries", len(before))
	}

	cmd := exec.Command(script)
	cmd.Dir = dest
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected the builder to refuse with no destination argument, but it exited 0:\n%s", out)
	}
	after, err := os.ReadDir(dest)
	if err != nil {
		t.Fatalf("read test working directory after the refused run: %v", err)
	}
	if len(after) != 0 {
		t.Fatalf("builder created files with no destination argument: %v", after)
	}
}
