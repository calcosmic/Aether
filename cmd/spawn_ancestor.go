package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/agent"
)

// normalizeSpawnTask applies exactly these transformations, and no others,
// so the function can be reasoned about and tested exhaustively:
//   - lowercase the whole string;
//   - replace every run of Unicode whitespace with a single space;
//   - strip leading and trailing spaces;
//   - strip a trailing full stop.
//
// strings.Fields already splits on runs of Unicode whitespace and discards
// leading/trailing whitespace, so Fields+Join covers the first three rules
// in one step; TrimSuffix covers the fourth. Nothing else is collapsed —
// no stemming, no stop-word removal, no further punctuation stripping, no
// truncation. T-173-31: every additional transformation widens the set of
// distinct tasks that collide, which turns a cycle guard into a
// false-positive generator.
func normalizeSpawnTask(task string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(task)), " ")
	return strings.TrimSuffix(normalized, ".")
}

// spawnAncestorChain returns the requester's own entry followed by each
// ancestor up to the root, nearest first. It parses the tree exactly once
// via st.Parse() and walks the returned slice in memory — never
// re-parsing per hop.
//
// Termination is bounded by a visited-name set (not a hop-count limit): a
// name already seen in the chain ends the walk immediately, which is both
// simpler than a hop cap and correct even if the cap's arithmetic ever
// drifted from spawnMaxDelegationDepth. T-173-29: this is what stops a
// corrupted or hand-edited tree with a self-referential parent link from
// looping forever inside the guard.
//
// If startName resolves to no entry and is not a coordinator sentinel, the
// chain cannot be established and an error names it — this is the D-19
// fail-closed source spawnAncestorCycleReason turns into a deny reason.
func spawnAncestorChain(st *agent.SpawnTree, startName string) ([]agent.SpawnEntry, error) {
	entries, err := st.Parse()
	if err != nil {
		return nil, err
	}

	// Last write wins: iterate in file order so a later entry for the same
	// AgentName (e.g. a re-spawn) overrides an earlier one in the map, which
	// is what "taking the latest such entry" means.
	byName := make(map[string]agent.SpawnEntry, len(entries))
	for _, e := range entries {
		byName[e.AgentName] = e
	}

	start, ok := byName[startName]
	if !ok {
		if spawnParentIsRoot(startName) {
			return nil, nil
		}
		return nil, fmt.Errorf("no recorded spawn entry for %q", startName)
	}

	chain := []agent.SpawnEntry{start}
	visited := map[string]bool{start.AgentName: true}
	current := start
	for !spawnParentIsRoot(current.ParentName) {
		parent, ok := byName[current.ParentName]
		if !ok || visited[parent.AgentName] {
			break
		}
		chain = append(chain, parent)
		visited[parent.AgentName] = true
		current = parent
	}
	return chain, nil
}

// spawnAncestorCycleReason is the ancestor-chain cycle check (SPAWN-05):
// called from spawnCanSpawnDecision as the third of its three checks, after
// depth and budget. A non-empty return is a human-readable deny reason; an
// empty return means allow.
//
// A depth cap alone cannot catch this: an A-to-B-to-A chain never exceeds
// the cap (each hop is only one level down) and never terminates on its
// own, so the hazard here is repetition along the ancestor chain, not a
// literal graph cycle (a spawn-tree entry names one parent and is written
// once). This walks the requester's own ancestor chain and refuses a
// caste-and-normalised-task pair that already appears above it, naming the
// ancestor it matched (D-10).
func spawnAncestorCycleReason(in spawnDecisionInput) string {
	if in.RequesterName == "" || spawnParentIsRoot(in.RequesterName) {
		// The coordinator has no ancestors, so there is nothing to repeat.
		return ""
	}

	st := agent.NewSpawnTree(store, "spawn-tree.txt")
	chain, err := spawnAncestorChain(st, in.RequesterName)
	if err != nil {
		// D-19 fail-closed: an unreadable chain must deny, never allow.
		return fmt.Sprintf("ancestor chain unreadable (%v): refusing to spawn", err)
	}

	normalizedTask := normalizeSpawnTask(in.Task)
	for _, ancestor := range chain {
		if !strings.EqualFold(ancestor.Caste, in.Caste) {
			continue
		}
		if normalizeSpawnTask(ancestor.Task) != normalizedTask {
			continue
		}
		return fmt.Sprintf(
			"%s was already asked to do this task by %s at depth %d; spawning it again would repeat work already in progress above",
			in.Caste, ancestor.AgentName, ancestor.Depth,
		)
	}
	return ""
}
