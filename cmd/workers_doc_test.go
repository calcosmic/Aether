package cmd

// Phase 203 Plan 15, Task 2: the guard that the shipped worker instructions
// (.aether/workers.md) describe only mechanisms this runtime actually
// grants. Before this file existed, that document taught a step-by-step
// spawning protocol -- an exact Task-tool invocation with
// subagent_type="general-purpose" -- that no caste has ever been granted,
// and stated a depth-limit table ("Depth 1→4, Depth 2→2, Depth 3→0") that
// contradicted the enforced cap the same document stated three paragraphs
// earlier. Per CLAUDE.md's Definition of Done, a documentation claim about
// runtime behaviour must be testable or removed -- this is that test.

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// workersDocPath is relative to the cmd/ package directory, where `go test
// ./cmd` runs with its working directory.
const workersDocPath = "../.aether/workers.md"

func readWorkersDoc(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(workersDocPath)
	if err != nil {
		t.Fatalf("read %s: %v", workersDocPath, err)
	}
	return string(data)
}

// TestWorkersDocMatchesRuntime fails by name in either direction: if the
// document still teaches a tool invocation no caste is granted, if it states
// a depth limit that disagrees with the limit the runtime actually enforces
// (derived from spawnMaxDelegationDepth, cmd/spawn.go -- never a typed
// literal here), or if it has lost the real recruitment command entirely (a
// guard that only ever checked for ABSENCE of the old text could pass on an
// empty or truncated file, which would be a false certificate).
func TestWorkersDocMatchesRuntime(t *testing.T) {
	content := readWorkersDoc(t)

	// Direction 1: the false mechanism must be gone.
	if strings.Contains(content, "subagent_type") {
		t.Error(".aether/workers.md still instructs workers to spawn via a Task-tool subagent_type invocation -- no caste has ever been granted that tool")
	}
	for _, banned := range []string{"Depth 1→4", "Depth 1 to 4"} {
		if strings.Contains(content, banned) {
			t.Errorf(".aether/workers.md still states the retired spawn-limit table (%q), which contradicts the enforced depth cap stated earlier in the same document", banned)
		}
	}

	// Direction 2: the real mechanism and its limit must be present, and the
	// stated limit must match the runtime's own enforced constant.
	if !strings.Contains(content, "aether recruit") {
		t.Fatal(".aether/workers.md no longer names the real public recruitment command (`aether recruit`) -- a guard that only checks for absent old text would pass on an empty or truncated file")
	}
	if !strings.Contains(content, "not an error") {
		t.Error(".aether/workers.md must state in plain words that a refusal is not an error")
	}
	if !strings.Contains(content, "--max-depth") {
		t.Error(".aether/workers.md must name the per-run depth-raise flag (--max-depth)")
	}
	if !strings.Contains(content, "written permanently") {
		t.Error(".aether/workers.md must state that using the per-run depth-raise flag is recorded permanently, so a raise is never invisible")
	}

	noDeeperThan := fmt.Sprintf("no deeper than %d", spawnMaxDelegationDepth)
	if !strings.Contains(content, noDeeperThan) {
		t.Errorf(".aether/workers.md does not state the enforced depth cap (%q, derived from spawnMaxDelegationDepth=%d) -- it must match the runtime's own constant, not a stale number", noDeeperThan, spawnMaxDelegationDepth)
	}
	noDepthPastCap := fmt.Sprintf("no depth %d", spawnMaxDelegationDepth+1)
	if !strings.Contains(content, noDepthPastCap) {
		t.Errorf(".aether/workers.md does not state that there is %q -- it must match the runtime's own enforced cap (spawnMaxDelegationDepth=%d), not a stale number", noDepthPastCap, spawnMaxDelegationDepth)
	}
}
