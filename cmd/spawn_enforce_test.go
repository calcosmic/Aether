package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
)

// WIRE-02 / D-13 / D-14.
//
// .aether/workers.md:292 instructs every worker to run
// `aether spawn-can-spawn {your_depth} --enforce` before spawning a child
// agent. Before this plan, that exact invocation errored twice over:
// `Args: cobra.NoArgs` rejected the positional depth, and `--enforce` was not
// registered at all. This file proves the fix two ways: that the manual's
// own string now resolves and executes (not a copy of the string — the file
// itself), and that --enforce's deny path is real machinery reachable by a
// test, not a flag that parses and does nothing (D-13's explicit rejection
// of an inert accepted-but-ignored flag).

// spawnCanSpawnInvocationRe finds the documented invocation line in
// workers.md and captures everything after `aether spawn-can-spawn`.
var spawnCanSpawnInvocationRe = regexp.MustCompile(`aether\s+spawn-can-spawn(\s+.*)?$`)

// extractSpawnCanSpawnInvocation reads workers.md at the given repo root,
// finds the line documenting the spawn-can-spawn invocation, and returns its
// argument tokens (excluding the command name itself). It tracks the manual
// instead of hardcoding a copy of it: if workers.md's wording ever changes,
// this test starts validating the new string, not a stale duplicate.
func extractSpawnCanSpawnInvocation(t *testing.T, root string) []string {
	t.Helper()
	path := filepath.Join(root, ".aether", "workers.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		m := spawnCanSpawnInvocationRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		rest := m[1]
		// The documented form is inside a shell command substitution:
		// `result=$(aether spawn-can-spawn {your_depth} --enforce)` — strip a
		// trailing close-paren/backtick that belongs to the shell wrapper,
		// not to the invocation itself.
		rest = strings.TrimRight(rest, ")` \t")
		tokens := tokenizeShellLike(strings.TrimSpace(rest))
		return tokens
	}

	t.Fatalf("no line in %s contains `aether spawn-can-spawn` — a test that finds nothing to check would pass vacuously forever", path)
	return nil
}

// TestSpawnCanSpawnAcceptsDocumentedInvocation is the WIRE-02 criterion,
// pinned to the manual: the exact invocation .aether/workers.md:292
// documents must resolve against the real cobra command tree and execute
// successfully.
func TestSpawnCanSpawnAcceptsDocumentedInvocation(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	tokens := extractSpawnCanSpawnInvocation(t, root)
	if len(tokens) == 0 {
		t.Fatalf("extracted invocation from workers.md has no arguments: tokens=%v", tokens)
	}

	// Resolve the extracted argument vector through cobra's own Find, the
	// same machinery the CLI uses, rather than assuming the command name.
	fullArgs := append([]string{"spawn-can-spawn"}, tokens...)
	target, remaining, err := rootCmd.Find(fullArgs)
	if err != nil {
		t.Fatalf("rootCmd.Find(%v) failed: %v", fullArgs, err)
	}
	if target == nil || target != spawnCanSpawnCmd {
		t.Fatalf("rootCmd.Find(%v) resolved to %v, want spawnCanSpawnCmd", fullArgs, target)
	}

	// Every --flag in the extracted call must be registered on the resolved
	// command, and every remaining positional must satisfy its own Args
	// validator — the exact two failure modes workers.md:292 hit before this
	// plan (unknown flag, and a rejected positional).
	var positionals []string
	for _, tok := range remaining {
		if strings.HasPrefix(tok, "--") {
			name := strings.TrimPrefix(tok, "--")
			if eq := strings.Index(name, "="); eq >= 0 {
				name = name[:eq]
			}
			if target.Flags().Lookup(name) == nil {
				t.Fatalf("documented invocation uses --%s, which is not registered on %s", name, target.Name())
			}
			continue
		}
		positionals = append(positionals, tok)
	}
	if target.Args == nil {
		t.Fatalf("%s has no Args validator to check the documented positional against", target.Name())
	}
	if err := target.Args(target, positionals); err != nil {
		t.Fatalf("documented positional args %v rejected by %s's own Args validator: %v", positionals, target.Name(), err)
	}

	// Now actually execute it in-process, with the concrete depth the
	// placeholder {your_depth} stands in for, and confirm the observable
	// behaviour workers.md:292 promises: it runs and exits 0.
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn", "5", "--enforce"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("documented invocation (depth 5, --enforce) failed: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("documented invocation set exit code %d; workers.md:292 promises it exits 0: %s", code, errBuf.String())
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("documented invocation not ok: %s", buf.String())
	}
	result := env["result"].(map[string]interface{})
	if result["can_spawn"] != true {
		t.Fatalf("expected can_spawn true, got: %s", buf.String())
	}
	if depth, _ := result["depth"].(float64); depth != 5 {
		t.Fatalf("expected depth 5, got: %s", buf.String())
	}
}

// TestSpawnCanSpawnEnforceDeniesWithNonZeroExit is the D-13 assertion:
// --enforce must carry real semantics. It drives the deny path through the
// decision seam spawnCanSpawnDecision and proves both halves of the
// contract: --enforce turns a deny into a non-zero exit, and its absence
// does not silently gate (deny without --enforce still exits 0, with the
// deny visible in the JSON payload).
func TestSpawnCanSpawnEnforceDeniesWithNonZeroExit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	origDecision := spawnCanSpawnDecision
	defer func() { spawnCanSpawnDecision = origDecision }()
	spawnCanSpawnDecision = func(depth int) (bool, string) {
		return false, "depth exceeds cap"
	}

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Half 1: deny + --enforce => non-zero exit, message names the depth.
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn", "7", "--enforce"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("deny + --enforce did not set a non-zero exit code; enforcement is not real: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "7") {
		t.Fatalf("deny + --enforce error output does not name the depth: %s", errBuf.String())
	}

	// Half 2: same deny WITHOUT --enforce => zero exit, deny visible in JSON.
	// This is the half that proves --enforce is what upgrades a report into
	// a gate, and that its absence does not silently gate. Cobra local flags
	// persist across Execute() calls within a single test (resetRootCmd only
	// resets them at test cleanup), so --enforce must be explicitly reset
	// here or half 1's true value would leak into half 2.
	buf.Reset()
	errBuf.Reset()
	if err := spawnCanSpawnCmd.Flags().Set("enforce", "false"); err != nil {
		t.Fatalf("reset --enforce flag: %v", err)
	}
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn", "7"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deny without --enforce should still succeed as a report: %v\nstderr: %s", err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("deny without --enforce set exit code %d; absence of --enforce must not silently gate: %s", code, errBuf.String())
	}
	env := parseEnvelope(t, buf.String())
	result, _ := env["result"].(map[string]interface{})
	if result == nil || result["can_spawn"] != false {
		t.Fatalf("deny without --enforce does not surface the deny in the JSON payload: %s", buf.String())
	}
}

// Plan 173-02, task 3 (SPAWN-02 / SPAWN-06 red-proofs).
//
// The six tests below prove the derived-depth recorder (Task 1) and the
// corrected .aether/workers.md (Task 2) by execution, per CLAUDE.md's
// Definition of Done and D-22: each fails when the thing it names is
// untrue, not merely when read.

// spawnLogArgs builds a full spawn-log argument vector with a fixed caste
// and task, varying only parent, name and the caller-claimed depth — shared
// by every test below so the six do not each grow a slightly different
// argument-building helper.
func spawnLogArgs(parent, name, claimedDepth string) []string {
	return []string{"spawn-log", "--parent", parent, "--caste", "builder", "--name", name, "--task", "t", "--depth", claimedDepth}
}

// runSpawnLogExpectingSuccess executes spawn-log with args, failing the test
// if the command errors or exits non-zero, and returns the parsed JSON
// result payload.
func runSpawnLogExpectingSuccess(t *testing.T, buf, errBuf *bytes.Buffer, args []string) map[string]interface{} {
	t.Helper()
	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%v failed: %v\nstderr: %s", args, err, errBuf.String())
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 {
		t.Fatalf("%v exited %d: %s", args, code, errBuf.String())
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("%v not ok: %s", args, buf.String())
	}
	result, _ := env["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("%v returned no result: %s", args, buf.String())
	}
	return result
}

// TestSpawnLogDerivesDepthFromRecordedParent is the direct SPAWN-02 proof:
// a three-level tree recorded entirely with --depth 0 on every call must
// still report the true derived depths (1, 2, 3) once parsed back out of
// the spawn tree — the recorder, not the caller, is the authority.
func TestSpawnLogDerivesDepthFromRecordedParent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "W1", "0"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("W1", "H1", "0"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("H1", "X1", "0"))

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("parse spawn tree: %v", err)
	}
	depthByName := map[string]int{}
	for _, e := range entries {
		depthByName[e.AgentName] = e.Depth
	}

	if got, want := depthByName["W1"], 1; got != want {
		t.Fatalf("W1 recorded depth = %d, want %d", got, want)
	}
	if got, want := depthByName["H1"], 2; got != want {
		t.Fatalf("H1 recorded depth = %d, want %d", got, want)
	}
	if got, want := depthByName["X1"], 3; got != want {
		t.Fatalf("X1 recorded depth = %d, want %d", got, want)
	}
}

// TestSpawnTreeDepthReportsTwoForAThreeLevelTree is D-07's fixed number and
// doubles as the SPAWN-06 proof: coordinator 0, worker 1, helper 2.
func TestSpawnTreeDepthReportsTwoForAThreeLevelTree(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "W1", "0"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("W1", "H1", "0"))

	buf.Reset()
	errBuf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-depth"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-depth failed: %v\nstderr: %s", err, errBuf.String())
	}
	env := parseEnvelope(t, buf.String())
	result, _ := env["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("spawn-tree-depth returned no result: %s", buf.String())
	}
	maxDepth, _ := result["max_depth"].(float64)
	if int(maxDepth) != 2 {
		t.Fatalf("max_depth = %v, want 2 (D-07: coordinator 0, worker 1, helper 2)", result["max_depth"])
	}
}

// TestSpawnLogIgnoresCallerSuppliedDepth is the T-173-06 proof: the same
// Queen -> W1 spawn recorded once with --depth 0 and once with --depth 9
// must record depth 1 both times, and the --depth 9 run's JSON payload must
// report the discrepancy (claimed_depth 9, depth 1) rather than hide it
// (T-173-10).
func TestSpawnLogIgnoresCallerSuppliedDepth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runWithClaimedDepth := func(claimedDepth string) map[string]interface{} {
		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		return runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgs("Queen", "W1", claimedDepth))
	}

	result0 := runWithClaimedDepth("0")
	result9 := runWithClaimedDepth("9")

	if got, _ := result0["depth"].(float64); int(got) != 1 {
		t.Fatalf("--depth 0: recorded depth = %v, want 1", result0["depth"])
	}
	if got, _ := result9["depth"].(float64); int(got) != 1 {
		t.Fatalf("--depth 9: recorded depth = %v, want 1", result9["depth"])
	}
	if got, _ := result9["claimed_depth"].(float64); int(got) != 9 {
		t.Fatalf("--depth 9: claimed_depth = %v, want 9 (the discrepancy must be reported, not hidden)", result9["claimed_depth"])
	}
}

// TestSpawnLogRefusesAnUnknownParentAndWritesNothing is the D-09/D-19
// no-trace proof: a spawn naming a parent that is neither a recorded entry
// nor a coordinator sentinel must exit non-zero, name the parent in its
// error, and leave spawn-tree.txt byte-identical to before the attempt.
func TestSpawnLogRefusesAnUnknownParentAndWritesNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	beforeLen := 0
	if data, err := store.ReadFile("spawn-tree.txt"); err == nil {
		beforeLen = len(data)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs(spawnLogArgs("Ghost-99", "X1", "1"))
	_ = rootCmd.Execute()

	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("spawn-log with unknown parent did not exit non-zero: stdout=%s stderr=%s", buf.String(), errBuf.String())
	}
	// outputError writes its JSON envelope to stderr, not stdout — mirroring
	// TestSpawnCanSpawnEnforceDeniesWithNonZeroExit's half 1.
	env := parseEnvelope(t, errBuf.String())
	errMsg, _ := env["error"].(string)
	if !strings.Contains(errMsg, "Ghost-99") {
		t.Fatalf("error message does not name the unresolved parent %q: %s", "Ghost-99", errBuf.String())
	}

	afterLen := 0
	if data, err := store.ReadFile("spawn-tree.txt"); err == nil {
		afterLen = len(data)
	}
	if afterLen != beforeLen {
		t.Fatalf("spawn-tree.txt byte length changed after a refused spawn: before=%d after=%d — a refusal must write nothing", beforeLen, afterLen)
	}
}

// TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent is T-173-11's
// invariant proof: every literal --parent value appearing on a spawn-log
// line anywhere in the live corpus (.claude/commands/ant, .opencode/commands/ant,
// .aether — excluding template placeholders such as {your_name}) must
// either be a recognised coordinator sentinel (spawnRootParentNames) or
// appear elsewhere in the same corpus as a --name literal (meaning it names
// a recorded child, not a coordinator). A future wrapper introducing a
// fourth coordinator name that spawnRootParentNames does not know about
// fails this test by name, rather than silently taking the wrong branch.
func TestSpawnRootSentinelsCoverEveryDocumentedCoordinatorParent(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	dirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
		filepath.Join(repoRoot, ".aether"),
	}

	parentRe := regexp.MustCompile(`--parent\s+"([^"]*)"`)
	nameRe := regexp.MustCompile(`--name\s+"([^"]*)"`)

	type occurrence struct {
		value string
		file  string
	}
	var parentOccurrences []occurrence
	names := map[string]bool{}

	for _, dir := range dirs {
		if _, statErr := os.Stat(dir); statErr != nil {
			continue
		}
		walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, walkFileErr error) error {
			if walkFileErr != nil {
				return walkFileErr
			}
			if info.IsDir() {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, line := range strings.Split(string(data), "\n") {
				if m := nameRe.FindStringSubmatch(line); m != nil {
					names[m[1]] = true
				}
				if !strings.Contains(line, "spawn-log") {
					continue
				}
				m := parentRe.FindStringSubmatch(line)
				if m == nil {
					continue
				}
				if strings.Contains(m[1], "{") {
					continue
				}
				parentOccurrences = append(parentOccurrences, occurrence{value: m[1], file: path})
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", dir, walkErr)
		}
	}

	if len(parentOccurrences) == 0 {
		t.Fatalf("found no --parent literal(s) on spawn-log lines across %v — a test that finds nothing to check would pass vacuously forever", dirs)
	}

	var unrecognized []string
	for _, occ := range parentOccurrences {
		if spawnParentIsRoot(occ.value) {
			continue
		}
		if names[occ.value] {
			continue
		}
		unrecognized = append(unrecognized, fmt.Sprintf("%q (%s)", occ.value, occ.file))
	}
	if len(unrecognized) > 0 {
		sort.Strings(unrecognized)
		t.Fatalf("found --parent literal(s) that are neither a coordinator sentinel (%s) nor a recorded child (--name literal) anywhere in the corpus:\n  %s",
			strings.Join(spawnRootParentNames, ", "), strings.Join(unrecognized, "\n  "))
	}
}

// workersMdTableRowRe matches a markdown table row beginning with an
// integer Depth cell, capturing the Depth cell and the Can Spawn? cell
// (the third pipe-delimited column in the Depth-Based Behavior table:
// Depth | Role | Can Spawn? | Max Sub-Spawns | Behavior).
var workersMdTableRowRe = regexp.MustCompile(`^\|\s*(\d+)\s*\|([^|]*)\|([^|]*)\|`)

// TestWorkersMdStatesOneDepthConvention is a content-level assertion
// reading the live .aether/workers.md, following
// extractSpawnCanSpawnInvocation's "read the real file, do not duplicate
// its string" approach. RESEARCH.md Pitfall 1 is explicit that a literal
// --depth 0 absence check alone would pass while the Depth-Based Behavior
// table's residue survived (a depth-2+ row still permitting further
// delegation) — this test asserts both.
func TestWorkersMdStatesOneDepthConvention(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	path := filepath.Join(repoRoot, ".aether", "workers.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.Contains(line, "spawn-log") && strings.Contains(line, "--depth 0") {
			t.Fatalf("line contains both spawn-log and --depth 0 — the recorded depth is derived from --parent, and every documented example must claim depth 1 for a child, not 0: %q", line)
		}
	}

	for _, line := range lines {
		m := workersMdTableRowRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		depth, convErr := strconv.Atoi(strings.TrimSpace(m[1]))
		if convErr != nil {
			continue
		}
		canSpawnCell := m[3]
		if depth >= 2 && strings.Contains(canSpawnCell, "Yes") {
			t.Fatalf("table row for depth %d has a Can Spawn cell containing %q — D-01 forbids delegation at depth 2 or deeper: %q", depth, canSpawnCell, line)
		}
	}

	if strings.Contains(strings.ToLower(content), "global cap") {
		t.Fatalf(".aether/workers.md still contains a case-insensitive match for %q — the wrong 10-worker cap must be removed", "global cap")
	}

	if !strings.Contains(content, "20") {
		t.Fatalf(".aether/workers.md does not contain the substring %q — the whole-run tree budget of 20 must be stated", "20")
	}
}
