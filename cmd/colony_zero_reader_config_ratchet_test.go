package cmd

// ROADMAP Phase 191 criterion 1 / ROSTER-01 / ROSTER-02 (191-01): the
// zero-reader colony/ config reappearance ratchet.
//
// 191-01 deleted 36 of the 39 files ROADMAP Phase 191 criterion 1 names: all
// 27 colony/agents/*.yaml caste definitions and all 9 colony/phases/*.yaml
// phase definitions. Both groups were built in Phase 154 ("Colony Assets")
// to serve control-ts/, the TypeScript control plane Phase 160 confirmed
// dead and deleted. Fresh re-verification immediately before deletion
// (191-CONTEXT.md, "Criterion 1" section, and 191-PATTERNS.md, "Multi-Surface
// Zero-Readership Proof" section) reconfirmed zero readers across Go source
// (including string-built paths, e.g. every policyPath(...) call site with
// every argument), the TS host, wrapper markdown on all three platforms
// (.claude/, .opencode/, .codex/), skills, and the publish/install
// manifest.
//
// The ROADMAP's remaining THREE -- colony/policies/model-routing.yaml,
// colony/policies/autopilot.yaml, and colony/policies/memory-rules.yaml --
// were initially held back by 191-01: fresh re-verification found a real,
// currently-passing reader the planning pass missed (cmd/policy_schema_test.go's
// TestPolicySchemaRequiredFields read all three via os.ReadFile and t.Fatalf'd
// if any was missing). The orchestrator's ruling at wave-1 merge (191-01-SUMMARY.md
// documents the finding; sibling 191-02 established that schema test is itself
// the deleted Phase-161 policy machinery's validator, not a live consumer):
// the files and their policySchemaChecks rows were deleted TOGETHER in one
// change, after a fresh sweep confirmed zero production readers across Go,
// the TS host, wrappers on all three platforms, skills, and the publish
// manifest. All three are ratcheted below with the other 36 -- ROSTER-02 is
// now fully delivered.
//
// This is deliberately a Tier-1 existence check (191-PATTERNS.md, "The
// Ratchet House Style" section), not an AST scanner: the property being guarded is
// pure file presence/absence, and this repo's own house rule is not to
// over-engineer a check for that ("don't over-engineer" -- 191-CONTEXT.md's
// canonical_refs, citing the phase's own planning brief). Tier 2
// (AST-based structural scanning) is reserved for the skill-lifecycle
// CLI-surface removal elsewhere in this phase, where the property being
// guarded is reachability, not mere presence.
//
// TestZeroReaderColonyConfigsDoNotReappear checks each of the 36
// confirmed-deleted paths individually and by name, using t.Errorf (not
// t.Fatalf), so a single run reports every reappeared file, not just the
// first one found. TestZeroReaderColonyConfigsListIsComplete guards against
// the path list itself silently shrinking or losing its 27/9 shape (a
// future edit dropping or misplacing an entry would make the first test
// pass vacuously for whatever path was dropped).
//
// Fail-then-pass proof (recreating one file from each of the three target
// groups -- one colony/agents/*.yaml, one colony/phases/*.yaml -- confirming
// the ratchet fails naming each recreated file, deleting them again, and
// confirming the ratchet passes) is recorded verbatim in
// 191-01-SUMMARY.md, per 191-PATTERNS.md's mandatory discipline that a
// ratchet which has never been observed to fail is unproven.

import (
	"os"
	"path/filepath"
	"testing"
)

// zeroReaderColonyConfigPaths is the complete, hardcoded list of the 36
// colony/ files Phase 191 Plan 01 confirmed zero-reader and deleted: 27
// colony/agents/*.yaml caste definitions + 9 colony/phases/*.yaml phase
// definitions. Deliberately not computed from a directory listing -- both
// source directories no longer exist post-deletion, which would make a
// listing-based approach vacuous (191-PATTERNS.md's Tier-1 guidance states
// this explicitly). Includes colony/policies/{model-routing,autopilot,
// memory-rules}.yaml, deleted at wave-1 merge together with their
// policy_schema_test.go rows -- see this file's header comment.
var zeroReaderColonyConfigPaths = []string{
	// colony/policies/*.yaml -- the three ruled at wave-1 merge.
	filepath.Join("colony", "policies", "model-routing.yaml"),
	filepath.Join("colony", "policies", "autopilot.yaml"),
	filepath.Join("colony", "policies", "memory-rules.yaml"),

	// colony/agents/*.yaml -- 27 entries.
	filepath.Join("colony", "agents", "ambassador.yaml"),
	filepath.Join("colony", "agents", "archaeologist.yaml"),
	filepath.Join("colony", "agents", "architect.yaml"),
	filepath.Join("colony", "agents", "auditor.yaml"),
	filepath.Join("colony", "agents", "builder.yaml"),
	filepath.Join("colony", "agents", "chaos.yaml"),
	filepath.Join("colony", "agents", "chronicler.yaml"),
	filepath.Join("colony", "agents", "fixer.yaml"),
	filepath.Join("colony", "agents", "gatekeeper.yaml"),
	filepath.Join("colony", "agents", "includer.yaml"),
	filepath.Join("colony", "agents", "keeper.yaml"),
	filepath.Join("colony", "agents", "measurer.yaml"),
	filepath.Join("colony", "agents", "medic.yaml"),
	filepath.Join("colony", "agents", "oracle.yaml"),
	filepath.Join("colony", "agents", "porter.yaml"),
	filepath.Join("colony", "agents", "probe.yaml"),
	filepath.Join("colony", "agents", "queen.yaml"),
	filepath.Join("colony", "agents", "route-setter.yaml"),
	filepath.Join("colony", "agents", "sage.yaml"),
	filepath.Join("colony", "agents", "scout.yaml"),
	filepath.Join("colony", "agents", "surveyor-disciplines.yaml"),
	filepath.Join("colony", "agents", "surveyor-nest.yaml"),
	filepath.Join("colony", "agents", "surveyor-pathogens.yaml"),
	filepath.Join("colony", "agents", "surveyor-provisions.yaml"),
	filepath.Join("colony", "agents", "tracker.yaml"),
	filepath.Join("colony", "agents", "watcher.yaml"),
	filepath.Join("colony", "agents", "weaver.yaml"),

	// colony/phases/*.yaml -- 9 entries.
	filepath.Join("colony", "phases", "build.yaml"),
	filepath.Join("colony", "phases", "colonize.yaml"),
	filepath.Join("colony", "phases", "continue.yaml"),
	filepath.Join("colony", "phases", "discuss.yaml"),
	filepath.Join("colony", "phases", "init.yaml"),
	filepath.Join("colony", "phases", "oracle.yaml"),
	filepath.Join("colony", "phases", "plan.yaml"),
	filepath.Join("colony", "phases", "seal.yaml"),
	filepath.Join("colony", "phases", "swarm.yaml"),
}

// TestZeroReaderColonyConfigsDoNotReappear fails, individually and by name,
// for each of the 36 paths above that exists on disk. It never aggregates
// into a single pass/fail that would hide which specific file came back.
func TestZeroReaderColonyConfigsDoNotReappear(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	for _, rel := range zeroReaderColonyConfigPaths {
		p := filepath.Join(root, rel)
		_, statErr := os.Stat(p)
		switch {
		case statErr == nil:
			t.Errorf("%s has reappeared -- this file was ruled zero-reader and deleted in Phase 191 Plan 01 (ROADMAP criterion 1, ROSTER-01/ROSTER-02). Either it has a genuine new reader -- in which case update 191-CONTEXT.md and this ratchet's path list with that evidence -- or it should be deleted again", filepath.ToSlash(rel))
		case !os.IsNotExist(statErr):
			t.Errorf("%s: unexpected stat error (want a clean \"does not exist\"; this may be masking the real reappearance check behind a permissions or filesystem problem): %v", filepath.ToSlash(rel), statErr)
		}
	}
}

// TestZeroReaderColonyConfigsListIsComplete asserts zeroReaderColonyConfigPaths
// has exactly 39 entries in the expected 3 (colony/policies) + 27
// (colony/agents) + 9 (colony/phases) shape, so a future edit that drops or
// misplaces an entry fails loudly instead of silently narrowing this
// ratchet's coverage.
func TestZeroReaderColonyConfigsListIsComplete(t *testing.T) {
	const wantPolicies = 3
	const wantAgents = 27
	const wantPhases = 9
	const want = wantPolicies + wantAgents + wantPhases

	if len(zeroReaderColonyConfigPaths) != want {
		t.Fatalf("zeroReaderColonyConfigPaths has %d entries, want exactly %d (%d colony/policies/*.yaml + %d colony/agents/*.yaml + %d colony/phases/*.yaml) -- Phase 191 deleted exactly this set; a shorter list silently checks fewer paths than were actually deleted", len(zeroReaderColonyConfigPaths), want, wantPolicies, wantAgents, wantPhases)
	}

	policiesDir := filepath.Join("colony", "policies")
	agentsDir := filepath.Join("colony", "agents")
	phasesDir := filepath.Join("colony", "phases")
	var policies, agents, phases int
	for _, p := range zeroReaderColonyConfigPaths {
		switch filepath.Dir(p) {
		case policiesDir:
			policies++
		case agentsDir:
			agents++
		case phasesDir:
			phases++
		default:
			t.Errorf("unexpected entry %q in zeroReaderColonyConfigPaths -- every entry must live directly under colony/policies/, colony/agents/ or colony/phases/", filepath.ToSlash(p))
		}
	}
	if policies != wantPolicies {
		t.Errorf("found %d colony/policies/*.yaml entries in zeroReaderColonyConfigPaths, want %d", policies, wantPolicies)
	}
	if agents != wantAgents {
		t.Errorf("found %d colony/agents/*.yaml entries in zeroReaderColonyConfigPaths, want %d", agents, wantAgents)
	}
	if phases != wantPhases {
		t.Errorf("found %d colony/phases/*.yaml entries in zeroReaderColonyConfigPaths, want %d", phases, wantPhases)
	}
}
