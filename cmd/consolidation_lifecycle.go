package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
)

// consolidationLifecycleTimeout bounds runPhaseEndConsolidation so a wedged
// or very slow consolidation run can never hang a phase advance (T-162-12).
// Consolidation is enrichment, not a gate (D-05), so it gets the same budget
// as the general command timeout convention in cmd/timeouts.go rather than
// the longer BuildTimeout.
const consolidationLifecycleTimeout = 30 * time.Second

// phaseEndConsolidationSummary is the runtime-facing summary of a
// non-blocking phase-end consolidation attempt (D-04, D-05). It is returned
// by value only -- runPhaseEndConsolidation never returns an error type a
// caller could propagate into an abort, because learning is enrichment, not
// a gate.
type phaseEndConsolidationSummary struct {
	Ran                 bool
	Reason              string
	InstinctsDecayed    int
	InstinctsArchived   int
	ObservationsDecayed int
	PromotionCandidates int
	QueenEligible       int
	ReviewCandidates    int
	RereadCandidates    int
}

// ZeroState reports whether this consolidation ran but found nothing worth
// promoting. PromotionCandidates == 0 && QueenEligible == 0 is the planner's
// resolution of RESEARCH.md Open Question 3: it matches D-06's example
// wording ("3 observations -> 2 learnings -> 1 new instinct") and treats
// decay/archive counts as detail rather than headline -- a phase can decay
// stale trust scores every single run without that being "something new"
// worth a non-zero learning beat (D-07).
func (s phaseEndConsolidationSummary) ZeroState() bool {
	return s.PromotionCandidates == 0 && s.QueenEligible == 0
}

// runPhaseEndConsolidation is the single non-blocking runtime caller for
// phase-end consolidation (D-04). It is invoked from both continue paths
// (cmd/codex_continue.go, cmd/codex_continue_finalize.go) only after a phase
// has durably advanced -- never mid-phase, never speculatively.
//
// This function is never a dry run: consolidationPhaseEndCmd's --dry-run
// branch (the dry-run consolidation service constructor in
// cmd/graph_consolidation_cmds.go) exists purely for the user-invocable
// inspection path and is pinned by TestConsolidationPhaseEndDryRunDoesNotMutate.
// runPhaseEndConsolidation always takes the real, mutating path -- it must
// never route through that dry-run constructor.
//
// On failure this never returns an error type the caller could propagate
// into an abort -- it warns unmissably to stderr (D-05) and returns a
// summary with Ran: false. Learning is enrichment, not a gate.
func runPhaseEndConsolidation(phaseID int) phaseEndConsolidationSummary {
	_ = phaseID // reserved for future phase-scoped consolidation reporting

	if store == nil {
		return phaseEndConsolidationSummary{Ran: false, Reason: "no store initialized"}
	}

	bus := events.NewBus(store, events.DefaultConfig())
	pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())

	// Self-heal a legacy local QUEEN.md that predates the Instincts section
	// before promoting into it. Non-fatal: mirrors consolidationPhaseEndCmd's
	// real-path branch (cmd/graph_consolidation_cmds.go). A missing section
	// degrades to pkg/memory's existing silent no-op, not a crash.
	if healErr := ensureQueenInstinctsSection(); healErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to ensure QUEEN.md Instincts section: %v\n", healErr)
	}

	// Bounded timeout so a wedged consolidation can never hang a phase
	// advance (T-162-12); the phase-advance record is already committed by
	// the time this function is reached.
	ctx, cancel := context.WithTimeout(context.Background(), consolidationLifecycleTimeout)
	defer cancel()

	result, err := pipeline.RunConsolidation(ctx)

	// ConsolidationService.Run (pkg/memory/consolidate.go) records per-step
	// failures (e.g. a corrupt instincts.json) into result.Errors instead of
	// returning a top-level error -- the pipeline is deliberately
	// non-blocking internally too. Treat a non-empty result.Errors the same
	// as a top-level err: both mean "consolidation did not run cleanly."
	if err == nil && result != nil && len(result.Errors) > 0 {
		err = errors.Join(result.Errors...)
	}

	if err != nil {
		reason := err.Error()
		// T-162-11: a curation sentinel abort (corrupt stores detected) must
		// be distinguishable from a transient failure in the one line the
		// operator sees.
		if strings.Contains(reason, "sentinel abort") {
			reason = "curation sentinel detected corrupt stores: " + reason
		}
		fmt.Fprintf(os.Stderr, "phase advanced WITHOUT consolidation — %v\n", reason)
		return phaseEndConsolidationSummary{Ran: false, Reason: reason}
	}

	return phaseEndConsolidationSummary{
		Ran:                 true,
		InstinctsDecayed:    result.InstinctsDecayed,
		InstinctsArchived:   result.InstinctsArchived,
		ObservationsDecayed: result.ObservationsDecayed,
		PromotionCandidates: len(result.PromotionCandidates),
		QueenEligible:       len(result.QueenEligible),
		ReviewCandidates:    len(result.ReviewCandidates),
		RereadCandidates:    len(result.RereadCandidates),
	}
}

// attachConsolidationSummary stores s under result["consolidation"] using the
// same snake_case keys consolidation-phase-end's own JSON output uses
// (instincts_decayed, instincts_archived, observations_decayed,
// promotion_candidates, queen_eligible, review_candidates,
// reread_candidates), plus ran and reason. Keeping the key names identical
// to the subcommand's output means the inspection path and the runtime path
// report the same shape.
func attachConsolidationSummary(result map[string]interface{}, s phaseEndConsolidationSummary) {
	if result == nil {
		return
	}
	result["consolidation"] = map[string]interface{}{
		"ran":                   s.Ran,
		"reason":                s.Reason,
		"instincts_decayed":     s.InstinctsDecayed,
		"instincts_archived":    s.InstinctsArchived,
		"observations_decayed":  s.ObservationsDecayed,
		"promotion_candidates":  s.PromotionCandidates,
		"queen_eligible":        s.QueenEligible,
		"review_candidates":     s.ReviewCandidates,
		"reread_candidates":     s.RereadCandidates,
	}
}
