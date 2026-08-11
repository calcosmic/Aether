package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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
