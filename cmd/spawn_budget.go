package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
)

// spawnTreeBudgetMax is the whole-run helper ceiling (D-02): fixed at 20 and
// deliberately not configurable — the user rejected a config key for this
// value explicitly, because a limit you can raise under pressure is a limit
// that gets raised. It counts every helper spawned anywhere in the run,
// including the coordinator's own workers (D-03): a wave of 8 workers has
// already consumed 8 of the 20, so 8-wide x 2-deep (73 workers) never once
// breaks the depth rule but is stopped here instead. This is a deliberately
// different quantity from the Queen's per-wave, pre-dispatch worker-selection
// cap defined in cmd/queen_spawn_budget.go (a range of 4 to 8) — D-04
// requires the wave cap and the tree budget to never collapse into one
// number, so this file shares no identifier and no value with that other one.
const spawnTreeBudgetMax = 20

// spawnTreeBudgetWarnFraction is the D-13 passive-awareness threshold: once
// a run has consumed at least this fraction of spawnTreeBudgetMax, spawn-log's
// own output carries a plain-English warning line so the operator does not
// have to watch a second window for it.
const spawnTreeBudgetWarnFraction = 0.75

// spawnTreeBudget is the whole-run tree-budget result. The "...Budget" struct
// shape with a Reason string is borrowed from cmd/queen_spawn_budget.go's
// per-wave selection type's naming convention on purpose — that is where the
// borrowing stops. This struct counts the whole run's spawn tree at
// spawn-record time; the other file's type counts one wave's pre-dispatch
// selection. They must never be conflated (D-04).
type spawnTreeBudget struct {
	Max       int
	Consumed  int
	Remaining int
	Reason    string
}

// spawnTreeBudgetState computes the current run's consumption against
// spawnTreeBudgetMax.
//
// D-19 fail-closed note: every error path here returns a non-nil error, and
// spawnTreeBudgetReason turns every such error into a deny — this function
// never assumes the budget is free when something cannot be read.
//
// The one deliberate exception is a colony that has never begun a run:
// CurrentRun() returning ok=false. That is treated as Consumed 0, not as a
// denial, because a colony that has never begun a run legitimately has no
// spawns yet, and denying here would lock a fresh colony out of spawning
// entirely (T-173-24). This is the one place this budget does not fail
// closed, and it is a named, justified decision, not an accident.
func spawnTreeBudgetState() (spawnTreeBudget, error) {
	if store == nil {
		return spawnTreeBudget{}, fmt.Errorf("no store initialized")
	}
	st := agent.NewSpawnTree(store, "spawn-tree.txt")

	run, ok, err := st.CurrentRun()
	if err != nil {
		return spawnTreeBudget{}, fmt.Errorf("resolve current run: %w", err)
	}
	if !ok {
		// No run has ever begun for this colony: a legitimately empty run,
		// not an unverifiable one. See the function comment above.
		return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: 0, Remaining: spawnTreeBudgetMax}, nil
	}

	entries, err := st.EntriesForRun(run.ID)
	if err != nil {
		return spawnTreeBudget{}, fmt.Errorf("load spawns for run %q: %w", run.ID, err)
	}

	consumed := 0
	for _, e := range entries {
		if strings.EqualFold(strings.TrimSpace(e.Status), agent.SpawnStatusAbandoned) {
			continue
		}
		consumed++
	}

	remaining := spawnTreeBudgetMax - consumed
	if remaining < 0 {
		remaining = 0
	}

	return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: consumed, Remaining: remaining}, nil
}

// spawnTreeBudgetReason is the whole-run tree-budget check contract
// (D-02/D-03/D-04): called from spawnCanSpawnDecision as the second of its
// three checks, after depth and before ancestor-cycle. A non-empty return is
// a human-readable deny reason; an empty return means allow.
func spawnTreeBudgetReason(in spawnDecisionInput) string {
	state, err := spawnTreeBudgetState()
	if err != nil {
		// D-19: fail closed. A budget that cannot be verified is never
		// treated as free.
		return fmt.Sprintf("whole-run helper budget unverifiable (%v): refusing to spawn", err)
	}

	if state.Consumed >= state.Max {
		spawnTreeBudgetCeilingToMidden(state, in)
		requesterName := in.RequesterName
		if requesterName == "" {
			requesterName = "the requester"
		}
		return fmt.Sprintf(
			"whole-run helper budget exhausted: %d of %d helpers already spawned in this run; %s may not spawn another",
			state.Consumed, state.Max, requesterName,
		)
	}

	return ""
}

// spawnTreeBudgetCeilingToMidden writes a midden record when the whole-run
// budget is exhausted (D-11). It is deliberately called only from the
// exhausted branch of spawnTreeBudgetReason: a routine depth refusal is
// expected behaviour, not a fault, and logging every refusal to the failure
// record would drown real faults and leak task text. Every error here is
// swallowed — a failed midden write must never turn a refusal into an
// allow, nor an allow into a refusal.
func spawnTreeBudgetCeilingToMidden(state spawnTreeBudget, in spawnDecisionInput) {
	if store == nil {
		return
	}
	requesterName := in.RequesterName
	if requesterName == "" {
		requesterName = "unknown requester"
	}

	var mf colony.MiddenFile
	if err := store.LoadJSON("midden.json", &mf); err != nil {
		mf = colony.MiddenFile{Version: "1.0.0", Entries: []colony.MiddenEntry{}}
	}
	if mf.Entries == nil {
		mf.Entries = []colony.MiddenEntry{}
	}

	ts := time.Now().UTC()
	entry := colony.MiddenEntry{
		ID:        fmt.Sprintf("midden_%d_%d", ts.Unix(), os.Getpid()),
		Timestamp: ts.Format(time.RFC3339),
		Category:  "delegation-budget",
		Source:    "spawn-budget",
		Message: fmt.Sprintf(
			"%s hit the whole-run helper budget: %d of %d helpers already spawned in this run",
			requesterName, state.Consumed, state.Max,
		),
		Reviewed: false,
		Tags:     []string{},
	}
	mf.Entries = append(mf.Entries, entry)

	_ = store.SaveJSON("midden.json", mf)
}

// spawnTreeBudgetWarning is D-13's passive-awareness line: once a run has
// consumed at least spawnTreeBudgetWarnFraction of spawnTreeBudgetMax (and
// has not yet hit the ceiling, which spawnTreeBudgetReason already reports on
// its own terms), spawn-log's output carries this one plain-English
// sentence — no jargon, no identifiers, no JSON. No dashboard, no polling,
// no second window (D-13's explicit scope limit).
func spawnTreeBudgetWarning(state spawnTreeBudget) string {
	threshold := int(float64(state.Max) * spawnTreeBudgetWarnFraction)
	if state.Consumed < threshold || state.Consumed >= state.Max {
		return ""
	}
	return fmt.Sprintf(
		"This run has used %d of the %d helpers it is allowed to spawn, and it will stop letting new helpers start once it reaches that limit.",
		state.Consumed, state.Max,
	)
}
