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
	// CountedWholeLedger is true when Consumed came from counting live
	// entries across the WHOLE ledger (Gap B, see spawnTreeBudgetState)
	// rather than from the current run's own time window.
	// spawnTreeBudgetReason uses this to say so in the exhausted-budget deny
	// sentence, so an operator is not left wondering why the count is higher
	// than the run they are watching.
	CountedWholeLedger bool
	// WholeLedgerCause records WHY the whole ledger was counted, so the deny
	// message can name the actual condition instead of guessing. The v1.0.55
	// field failure was chased for hours because the message claimed "no run
	// is recorded" when a run existed but had ended. Values:
	// whole-ledger-no-run, whole-ledger-run-ended, whole-ledger-live-floor.
	WholeLedgerCause string
}

// The three WholeLedgerCause values. Each maps to a distinct sentence in
// spawnTreeBudgetReason — if a new whole-ledger path is added, it needs its
// own cause and its own sentence, not a reuse of these.
const (
	wholeLedgerNoRun     = "whole-ledger-no-run"
	wholeLedgerRunEnded  = "whole-ledger-run-ended"
	wholeLedgerLiveFloor = "whole-ledger-live-floor"
)

// spawnTreeBudgetState computes the current run's consumption against
// spawnTreeBudgetMax.
//
// D-19 fail-closed note: every error path here returns a non-nil error, and
// spawnTreeBudgetReason turns every such error into a deny — this function
// never assumes the budget is free when something cannot be read.
//
// Ledger integrity (Gap A, 173-VERIFICATION.md's Gap 1) is established with
// st.Parse() BEFORE the run window is resolved at all — this ordering is the
// whole point and must never move later "for efficiency". If the integrity
// check sits after CurrentRun()'s !ok early return, then corrupting
// spawn-tree.txt AND deleting spawn-runs.json — two ordinary shell commands —
// takes the no-run-yet exception below and manufactures a fresh budget: the
// same exploit wearing one extra step.
//
// The absent-ledger exception is narrower than it looks at first read (Gap
// B): it applies ONLY when the ledger itself is absent, zero-byte, or
// whitespace-only — the same three cases the parser treats as "no ledger
// yet" — not merely when no run is recorded. A ledger that already holds
// entries proves the colony is not fresh, whether or not a run was ever
// begun, so when CurrentRun() reports ok=false against a NON-EMPTY ledger,
// this function counts every live entry across the WHOLE ledger instead of
// reporting a free budget. 173-VERIFICATION.md's second reset route — `rm
// .aether/data/spawn-runs.json` against a valid, full ledger — used to take
// this branch and report Consumed:0; it now counts the ledger instead.
//
// This does NOT deny when no run resolves against a non-empty ledger, and
// that is a deliberate choice, not an oversight: nothing begins a run except
// beginRuntimeSpawnRun (cmd/spawn_runs.go), so "the ledger has entries and no
// run was ever recorded" is ROUTINE legitimate use — cmd/spawn_enforce_test.go
// and cmd/spawn_ancestor_test.go both build multi-hop spawn chains this way
// with no run ever begun, and .aether/workers.md:292 documents workers
// calling this guard directly, outside any lifecycle command.
//
// The whole-ledger fallbacks count LIVE entries only. They used to count
// everything non-abandoned, which turned the append-only ledger into a
// lifetime meter: a repo whose colony had ever finished more than 20 helpers
// could never spawn again (the field failure counted 117 completed helpers
// from months of history against a per-run cap of 20, and no recovery
// command could clear it). Live-only counting keeps the tamper floor that
// matters — erasing spawn-runs.json still counts every in-flight helper, so
// a run cannot free its OWN live spend that way (T-173-70's core case) —
// while the narrowed residual (deleting the run record after helpers have
// completed frees their slots) is no stronger than the already-accepted
// T-173-67 residual: an attacker who rewrites the ledger in well-formed pipe
// format can drop entries outright, and no counting rule here can detect
// that.
func spawnTreeBudgetState() (spawnTreeBudget, error) {
	if store == nil {
		return spawnTreeBudget{}, fmt.Errorf("no store initialized")
	}
	st := agent.NewSpawnTree(store, "spawn-tree.txt")

	// Gap A: establish ledger integrity before anything else touches the run
	// window. See the doc comment above — this ordering is load-bearing.
	entries, err := st.Parse()
	if err != nil {
		return spawnTreeBudget{}, fmt.Errorf("verify spawn-tree.txt: %w", err)
	}

	// CurrentRun()'s own error path already fails closed (plan 11): an
	// obstructed or unreadable spawn-runs.json (a directory at its path, a
	// permission denial, invalid JSON) returns a non-nil error here, not an
	// empty run history.
	run, ok, err := st.CurrentRun()
	if err != nil {
		return spawnTreeBudget{}, fmt.Errorf("resolve current run: %w", err)
	}
	if !ok {
		// Gap B: no run has ever been recorded for this colony. The
		// fresh-colony exception (T-173-24) applies only when the ledger
		// itself is empty — see the doc comment above for why a non-empty
		// ledger is counted rather than refused here.
		if len(entries) == 0 {
			return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: 0, Remaining: spawnTreeBudgetMax}, nil
		}

		// Count LIVE entries only, not everything non-abandoned. The ledger
		// is append-only across the colony's whole lifetime, so a finished
		// helper from May must not consume June's budget: a real repo
		// accumulated 117 completed entries and became permanently unable to
		// spawn against the cap of 20. Live-only counting keeps the
		// anti-tamper floor (erasing spawn-runs.json still counts every
		// in-flight helper — T-173-70's spirit), and the residual — freeing
		// completed entries by deleting the run record — is no stronger than
		// the already-accepted T-173-67 residual (an attacker who can shell
		// into .aether/data can rewrite the ledger lines outright).
		consumed := 0
		for _, e := range entries {
			if agent.IsLiveSpawnStatus(e.Status) {
				consumed++
			}
		}
		remaining := spawnTreeBudgetMax - consumed
		if remaining < 0 {
			remaining = 0
		}
		return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: consumed, Remaining: remaining, CountedWholeLedger: consumed > 0, WholeLedgerCause: wholeLedgerNoRun}, nil
	}

	entriesForRun, err := st.EntriesForRun(run.ID)
	if err != nil {
		return spawnTreeBudget{}, fmt.Errorf("load spawns for run %q: %w", run.ID, err)
	}

	consumed := 0
	for _, e := range entriesForRun {
		if strings.EqualFold(strings.TrimSpace(e.Status), agent.SpawnStatusAbandoned) {
			continue
		}
		consumed++
	}

	// WR-08 (173-REVIEW.md): a resolved run is a trustworthy counting window
	// only while it is ACTIVE. CurrentRun() returns the recorded run whether
	// or not it has ended, and an ended run's closed [StartedAt, EndedAt]
	// window can exclude every live entry in the ledger -- hand-editing
	// spawn-runs.json's status or timestamps (no deletion, no ledger
	// tampering) used to make a full 20-helper ledger report Consumed:0.
	// The two rules below only ever RAISE the count, so a legitimately
	// active run -- whose own spawns always land inside its open-ended
	// window -- is never penalised:
	//
	//   - run not active: there is no live window at all. Count the whole
	//     ledger's non-abandoned entries, exactly as the no-run branch
	//     above does.
	//   - run active, but live entries exist outside its window: count
	//     every live entry in the whole ledger as a floor. Unreaped ghosts
	//     counting against the budget is sanctioned behaviour (plan 09) --
	//     spawn-reap, not window scoping, is how budget is legitimately
	//     freed.
	countedWholeLedger := false
	wholeLedgerCause := ""
	liveWhole := 0
	for _, e := range entries {
		if agent.IsLiveSpawnStatus(e.Status) {
			liveWhole++
		}
	}
	if !agent.IsActiveSpawnRunStatus(run.Status) {
		// The recorded run has ended, so its closed window is not a
		// trustworthy picture of what is happening NOW — but this is also
		// the ROUTINE post-run path (workers call spawn-log from later
		// processes after the lifecycle command's run closed), not only a
		// tamper route. Keep the ended run's own window count as the base
		// (that spend was real) and floor it by every live entry anywhere in
		// the ledger. Counting everything non-abandoned here is what bricked
		// long-lived repos: every completed helper since the colony's first
		// day counted against a per-run cap of 20.
		if liveWhole > consumed {
			consumed = liveWhole
			countedWholeLedger = true
			wholeLedgerCause = wholeLedgerRunEnded
		}
	} else {
		liveWindow := 0
		for _, e := range entriesForRun {
			if agent.IsLiveSpawnStatus(e.Status) {
				liveWindow++
			}
		}
		if liveWhole > liveWindow && liveWhole > consumed {
			consumed = liveWhole
			countedWholeLedger = true
			wholeLedgerCause = wholeLedgerLiveFloor
		}
	}

	remaining := spawnTreeBudgetMax - consumed
	if remaining < 0 {
		remaining = 0
	}

	return spawnTreeBudget{Max: spawnTreeBudgetMax, Consumed: consumed, Remaining: remaining, CountedWholeLedger: countedWholeLedger, WholeLedgerCause: wholeLedgerCause}, nil
}

// spawnBudgetPreflight answers, before a single worker is dispatched, whether
// the whole plan fits in what is left.
//
// The per-spawn check below is a ceiling, and a ceiling checked one worker at a
// time lets a ten-worker build begin with three slots free and stall on the
// fourth -- which is what happened on 2026-08-14, twice, and is why the failure
// record holds "27 of 20" and "37 of 20". Refusing up front costs nothing;
// stalling half way costs every worker that already ran.
//
// Refuse rather than trim: the dispatch list is ordered by wave, so trimming
// its tail drops later tasks and returns a build that looks finished and is not.
// A refusal the operator can act on is honest; a silent partial build is not.
func spawnBudgetPreflight(planned int, state spawnTreeBudget) (bool, string) {
	if planned <= 0 {
		return true, ""
	}
	if planned <= state.Remaining {
		return true, ""
	}
	return false, fmt.Sprintf(
		"This phase needs %d workers and only %d of the %d for this run are left. "+
			"Nothing has been started. Helpers from an earlier command that never reported "+
			"finishing still count — run `aether spawn-orphans` to see them, or start a new session.",
		planned, state.Remaining, state.Max,
	)
}

// spawnCompleteTolerant records a worker's completion even when the ledger has
// no matching entry.
//
// Returning an error here is what leaked the budget. A live run recorded
// Chip-49 with recorded:true, the matching completion answered `spawn_tree:
// agent "Chip-49" not found`, and the retry consumed a second slot for the same
// worker. Worse, the entry stayed live, and a live entry outside the current
// window raises the next command's consumed count -- so one failed completion
// taxes every command after it.
//
// A completion for a worker nobody recorded is not a fault worth stopping for:
// the worker is finished either way, and the only question is whether the
// colony keeps paying for it.
func spawnCompleteTolerant(tree *agent.SpawnTree, name, status, summary string, _ time.Time) error {
	if tree == nil {
		return nil
	}
	err := tree.UpdateStatus(name, status, summary)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "not found") {
		return nil
	}
	return err
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
		if state.CountedWholeLedger {
			// The count came from the whole ledger rather than the current
			// run's window. Name the ACTUAL cause: the v1.0.55 message
			// claimed "no run is recorded" for every whole-ledger count,
			// including the run-ended and live-floor paths where a run very
			// much existed — an operator chased that phantom for hours.
			why := "no run is recorded"
			switch state.WholeLedgerCause {
			case wholeLedgerRunEnded:
				why = "the recorded run has ended, so helpers still marked running anywhere in the ledger are counted"
			case wholeLedgerLiveFloor:
				why = "helpers still marked running exist outside the current run's window"
			}
			return fmt.Sprintf(
				"whole-run helper budget exhausted: %d of %d helpers already spawned (counted across the entire ledger because %s); %s may not spawn another — `aether spawn-orphans` lists helpers that never reported finishing",
				state.Consumed, state.Max, why, requesterName,
			)
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
