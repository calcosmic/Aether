package cmd

import (
	"path/filepath"
	"testing"
)

// TestVerifyOutOfBandHasNoLifecycleCaller is the reachability ratchet
// T-191.1-03-01 requires: `aether verify-out-of-band` must never be
// reachable from any automatic lifecycle entry point (build, continue,
// init, resume-colony, run, build-finalize, continue-finalize) -- exactly
// like worktree-reap (cmd/worktree_destruction_reachability_test.go). It
// reuses that file's own AST call-graph infrastructure directly --
// buildCmdFuncGraph, reachableFrom, lifecycleEntryCommands -- rather than
// reimplementing a weaker, name-grep-based check (191.1-PATTERNS.md
// Pattern 3). A grep-based reachability ratchet was already defeated once in
// this codebase (CR-05: a second destructive function existed under a name
// no hardcoded string matched, and the grep-based test passed anyway) --
// this guard cannot be defeated the same way, because it walks the real
// call graph rather than searching for literal names.
//
// Two things are checked, mirroring
// TestNoLifecycleReachableFunctionDestroysWorktreeWithoutSafetyGate's own
// two-part shape:
//
//  1. The command's own Cobra node ("cobra:verify-out-of-band") is not in
//     the set reachable from any lifecycle entry point.
//  2. closeOutOfBandCeremony -- the ONE function that both closes a build
//     attempt with out-of-band provenance and calls advancePhase with
//     Source "verify-out-of-band" (cmd/verify_out_of_band.go) -- is not in
//     that reachable set, under its own name. A future indirection that
//     wires this function into an automatic path is caught here whether or
//     not the command itself is also touched.
//
// Fails loudly, never vacuously: asserts the graph actually found both the
// command and the function before asserting either is unreachable.
func TestVerifyOutOfBandHasNoLifecycleCaller(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}

	if g.entriesFound == 0 {
		t.Fatalf("found zero Cobra command entry points while scanning %d files -- a guard that finds no entry points to check would pass vacuously forever", g.filesScanned)
	}
	if g.funcsIndexed == 0 {
		t.Fatalf("found zero top-level functions while scanning %d files -- a guard that finds no functions to check would pass vacuously forever", g.filesScanned)
	}

	const commandName = "verify-out-of-band"
	const closingFuncName = "closeOutOfBandCeremony"

	cmdNode, ok := g.cobraEntryNodes[commandName]
	if !ok {
		t.Fatalf("expected to find the %q command among %d indexed entry points, but it was missing -- this guard cannot assert an isolation property against a command it cannot locate", commandName, g.entriesFound)
	}
	if _, ok := g.calls[closingFuncName]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- this guard cannot assert an isolation property against a function it cannot locate (renamed? moved?)", closingFuncName, g.funcsIndexed)
	}

	var startNodes []string
	var missingEntries []string
	for _, cmdName := range lifecycleEntryCommands {
		node, ok := g.cobraEntryNodes[cmdName]
		if !ok {
			missingEntries = append(missingEntries, cmdName)
			continue
		}
		startNodes = append(startNodes, node)
	}
	if len(missingEntries) > 0 {
		t.Fatalf("expected to find Cobra commands named %v among %d indexed entry points, but these were missing: %v -- update lifecycleEntryCommands or investigate why the command was not found", lifecycleEntryCommands, g.entriesFound, missingEntries)
	}

	reachable := reachableFrom(g, startNodes)

	if reachable[cmdNode] {
		t.Errorf("the operator-invoked command %q is reachable from an automatic lifecycle path -- verify-out-of-band must stay behind a human explicitly typing the command, never wired into build/continue/init/resume-colony/run/build-finalize/continue-finalize", commandName)
	}
	if reachable[closingFuncName] {
		t.Errorf("%s (the ceremony's state-mutating close -- closes a build attempt with out-of-band provenance and calls advancePhase with Source \"verify-out-of-band\") is reachable from an automatic lifecycle path under its own name -- this function may only be called from verify-out-of-band's own RunE, which itself must only run when a human types the command", closingFuncName)
	}
}
