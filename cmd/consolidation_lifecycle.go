package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent/curation"
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

// LearningBeatLine renders the single-line, caste-agnostic message body used
// by both renderLearningBeat's terminal beat (cmd/codex_visuals.go) and
// continueLearningFlowStep's ceremony summary (cmd/codex_continue.go), so
// the two surfaces can never drift out of sync (D-06/D-07). Exactly one of
// the four states below is reachable; there is no "silent" fifth branch.
func (s phaseEndConsolidationSummary) LearningBeatLine() string {
	if !s.Ran {
		return "phase advanced WITHOUT consolidation — " + strings.TrimSpace(s.Reason)
	}
	if s.ZeroState() {
		return "colony observed nothing new this phase"
	}
	return fmt.Sprintf("%d promotion candidate(s) -> %d queen-eligible instinct(s)", s.PromotionCandidates, s.QueenEligible)
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
		"ran":                  s.Ran,
		"reason":               s.Reason,
		"instincts_decayed":    s.InstinctsDecayed,
		"instincts_archived":   s.InstinctsArchived,
		"observations_decayed": s.ObservationsDecayed,
		"promotion_candidates": s.PromotionCandidates,
		"queen_eligible":       s.QueenEligible,
		"review_candidates":    s.ReviewCandidates,
		"reread_candidates":    s.RereadCandidates,
	}
}

// sealAntBeat is one curation ant's individually-reported outcome from a
// seal consolidation pass (D-06). Name preserves the orchestrator's fixed
// order (sentinel, nurse, critic, herald, janitor, archivist, librarian,
// scribe) so callers can render eight distinct beats instead of one
// collapsed aggregate string.
type sealAntBeat struct {
	Name    string
	Success bool
	Detail  string
}

// sealConsolidationSummary is the runtime-facing summary of a non-blocking
// seal consolidation attempt: the eight-ant curation pass plus decay/archive
// plus the scribe's written report artifact (LEARN-02). Like
// phaseEndConsolidationSummary, it is returned by value only --
// runSealConsolidation never returns an error type a caller could propagate
// into an abort (D-05). The seal never stops for a learning failure.
type sealConsolidationSummary struct {
	Ran                 bool
	Reason              string
	Ants                []sealAntBeat
	ReportPath          string
	InstinctsDecayed    int
	InstinctsArchived   int
	ObservationsDecayed int
	PromotionCandidates int
	QueenEligible       int
	// QueenPromotedIDs carries the actual instinct IDs pkg/memory's
	// RunConsolidation promoted into QUEEN.md this seal -- not just a count --
	// so Task 2's reconciliation can build an exact skip-set for the
	// subordinate seal-side promotion loop (D-09).
	QueenPromotedIDs []string
	ReviewCandidates int
	RereadCandidates int
}

// sealAntDetail derives a short, human-readable line from one curation ant's
// StepResult for display beside its caste-styled name (Task 3). Known
// per-ant Summary keys (e.g. archivist's "archived", janitor's "removed")
// render as key=value pairs sorted for determinism; a failed step falls back
// to its error text, and a step with nothing useful reported falls back to
// "ok" / "skipped".
func sealAntDetail(sr curation.StepResult) string {
	if !sr.Success {
		if sr.Error != nil {
			return sr.Error.Error()
		}
		if reason, ok := sr.Summary["reason"].(string); ok && reason != "" {
			return reason
		}
		return "failed"
	}
	if len(sr.Summary) == 0 {
		return "ok"
	}
	keys := make([]string, 0, len(sr.Summary))
	for k := range sr.Summary {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, sr.Summary[k]))
	}
	return strings.Join(parts, " ")
}

// runSealConsolidation is the single non-blocking runtime caller for seal
// consolidation (LEARN-02). It runs the full eight-ant curation pass
// (pkg/agent/curation.Orchestrator) directly -- never through the
// consolidation-seal subcommand's own stepInfo aggregation, which collapses
// all eight ants into one "succeeded=N failed=M" string and is exactly what
// makes D-06's per-ant announcements impossible -- then runs the real
// (non-dry-run) decay/archive/promotion pipeline, publishes the
// consolidation.seal event, and persists the scribe's report to
// <.aether>/CURATION-REPORT.md.
//
// This function is never a dry run: consolidationSealCmd's --dry-run branch
// exists purely for the user-invocable inspection path. runSealConsolidation
// always takes the real, mutating path.
//
// On any failure at any stage this never returns an error type the caller
// could propagate into an abort -- it warns unmissably to stderr (D-05),
// naming a sentinel abort explicitly when that is the cause, and returns a
// summary with Ran: false. The seal never stops for a learning failure.
func runSealConsolidation() sealConsolidationSummary {
	if store == nil {
		return sealConsolidationSummary{Ran: false, Reason: "no store initialized"}
	}

	// Self-heal a legacy local QUEEN.md that predates the Instincts section
	// before promoting into it. Non-fatal, mirrors runPhaseEndConsolidation.
	if healErr := ensureQueenInstinctsSection(); healErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to ensure QUEEN.md Instincts section: %v\n", healErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), consolidationLifecycleTimeout)
	defer cancel()

	bus := events.NewBus(store, events.DefaultConfig())

	curResult, curErr := curation.NewOrchestrator(store, bus).Run(ctx, false)

	var ants []sealAntBeat
	var reportPath string
	if curResult != nil {
		ants = make([]sealAntBeat, 0, len(curResult.Steps))
		for _, step := range curResult.Steps {
			ants = append(ants, sealAntBeat{
				Name:    step.Name,
				Success: step.Success,
				Detail:  sealAntDetail(step),
			})
			if step.Name == "scribe" && step.Success {
				if report, ok := step.Summary["report"].(string); ok && report != "" {
					path := filepath.Join(filepath.Dir(store.BasePath()), "CURATION-REPORT.md")
					if writeErr := os.WriteFile(path, []byte(report), 0644); writeErr != nil {
						fmt.Fprintf(os.Stderr, "warning: failed to write %s: %v\n", path, writeErr)
					} else {
						reportPath = path
					}
				}
			}
		}
	}

	pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())
	consResult, consErr := pipeline.RunConsolidation(ctx)
	if consErr == nil && consResult != nil && len(consResult.Errors) > 0 {
		consErr = errors.Join(consResult.Errors...)
	}

	// Publish the seal consolidation event, matching consolidationSealCmd's
	// own literal topic. Publish failures are non-blocking.
	if payload, err := json.Marshal(map[string]string{
		"type":      "consolidation.seal",
		"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}); err == nil {
		_, _ = bus.Publish(ctx, "consolidation.seal", payload, "seal")
	}

	summary := sealConsolidationSummary{Ants: ants, ReportPath: reportPath}

	var reasons []string
	if curErr != nil {
		reason := curErr.Error()
		// T-162-11: a curation sentinel abort (corrupt stores detected) must
		// be distinguishable from a transient failure in the one line the
		// operator sees.
		if strings.Contains(reason, "sentinel abort") {
			reason = "curation sentinel detected corrupt stores: " + reason
		}
		reasons = append(reasons, reason)
	}
	if consErr != nil {
		reasons = append(reasons, consErr.Error())
	}

	if len(reasons) > 0 {
		reason := strings.Join(reasons, "; ")
		fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
		summary.Ran = false
		summary.Reason = reason
		return summary
	}

	summary.Ran = true
	summary.InstinctsDecayed = consResult.InstinctsDecayed
	summary.InstinctsArchived = consResult.InstinctsArchived
	summary.ObservationsDecayed = consResult.ObservationsDecayed
	summary.PromotionCandidates = len(consResult.PromotionCandidates)
	summary.QueenEligible = len(consResult.QueenEligible)
	summary.QueenPromotedIDs = append([]string{}, consResult.QueenEligible...)
	summary.ReviewCandidates = len(consResult.ReviewCandidates)
	summary.RereadCandidates = len(consResult.RereadCandidates)
	return summary
}
