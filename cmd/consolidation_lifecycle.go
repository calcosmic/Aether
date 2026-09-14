package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent/curation"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
)

// consolidationLifecycleTimeout bounds runPhaseEndConsolidation so a wedged
// or very slow consolidation run can never hang a phase advance (T-162-12).
// Consolidation is enrichment, not a gate (D-05), so it gets the same budget
// as the general command timeout convention in cmd/timeouts.go rather than
// the longer BuildTimeout. A var (not a const) so tests can shorten it to
// prove the bound is real. The bound is enforced by
// runConsolidationStageBounded, NOT by context.WithTimeout alone: a context
// deadline cannot preempt a synchronous callee that never polls ctx.Done(),
// and storage.FileLocker's flock blocks with no deadline (WR-02).
var consolidationLifecycleTimeout = 30 * time.Second

const (
	consolidationQueenStorePath      = "QUEEN.md"
	consolidationQueenRepositoryPath = ".aether/QUEEN.md"
)

// writeConsolidationQueenPromotion is the repository-authorized side of the
// memory pipeline's typed Queen promotion boundary. Eligibility stays in
// pkg/memory; cmd only turns the already-approved instinct into the existing
// local Queen entry and commits that one repository-relative target through
// the Plan 27-28 mutation session.
func writeConsolidationQueenPromotion(ctx context.Context, dataRoot string, instinct colony.InstinctEntry, colonyName string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("promote local Queen instinct: %w", err)
	}
	if strings.TrimSpace(dataRoot) == "" {
		return fmt.Errorf("promote local Queen instinct: no store initialized")
	}
	repositoryRoot, err := planningRepositoryRoot(dataRoot)
	if err != nil {
		return fmt.Errorf("promote local Queen instinct: %w", err)
	}

	return withPlanningMutationSession(repositoryRoot, "consolidation-queen-promotion", func(session *planningMutationSession) error {
		current, exists, err := session.ReadFile(lifecycleTransactionRootRepository, consolidationQueenRepositoryPath)
		if err != nil {
			return fmt.Errorf("read local Queen: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("promote local Queen instinct: %w", err)
		}

		text := string(current)
		if !exists {
			text = queenDefaultContent
		}
		text = ensureConsolidationQueenInstinctsSectionText(text)
		entry := fmt.Sprintf("- [instinct] **%s** (%.2f): When %s, then %s",
			instinct.Domain, instinct.Confidence, instinct.Trigger, instinct.Action)
		updated := appendEntryToQueenSection(text, "Instincts", entry)
		if strings.TrimSpace(updated) == "" {
			return fmt.Errorf("promote local Queen instinct: refusing empty Queen content")
		}
		if exists && updated == string(current) {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("promote local Queen instinct: %w", err)
		}
		if err := commitPlanningSessionTargets(session, "consolidation-queen-promotion", "consolidation-queen-promotion", []planningSessionTarget{{
			Root: lifecycleTransactionRootRepository, Path: consolidationQueenRepositoryPath, Content: []byte(updated),
		}}); err != nil {
			return fmt.Errorf("write local Queen: %w", err)
		}
		return nil
	})
}

// ensureConsolidationQueenInstinctsSectionText is the mutation-free form of
// ensureQueenInstinctsSection. Keeping the derivation in memory lets the
// repository transaction own the only filesystem write, including the legacy
// section self-heal.
func ensureConsolidationQueenInstinctsSectionText(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "## Instincts" {
			return text
		}
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return text + "\n## Instincts\n> Instincts promoted by the consolidation pipeline.\n"
}

// runConsolidationStageBounded runs fn on its own goroutine and waits for
// either completion or ctx expiry, returning false when ctx expired first.
// On expiry the goroutine is deliberately abandoned -- it may be parked
// inside a blocking flock with no cancellation point (WR-02: neither
// ConsolidationService.Run nor storage.FileLocker checks ctx mid-step).
// That leak is the accepted cost of a hard bound: both callers warn and
// return immediately, the process exits shortly after, and the abandoned
// goroutine's captured results are never read after a timeout, so there is
// no data race on them.
func runConsolidationStageBounded(ctx context.Context, fn func()) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

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
	// QueenPromoted carries the actual instinct IDs pkg/memory's
	// RunConsolidation promoted into QUEEN.md this phase -- not just a count
	// (198.1-03/FEED-03). Sourced from the pipeline's own QueenPromoted
	// (writes that succeeded), never from QueenEligible: an eligible instinct
	// whose write failed must not be reported as reaching the Queen file.
	QueenPromoted []string
	// InstinctApplicationsRecorded is recordInstinctApplicationsForPhase's own
	// return value for this phase -- how many instincts gained a fresh,
	// honest use this phase because a worker was genuinely given them.
	InstinctApplicationsRecorded int
	// FailuresRecorded is the count of unacknowledged midden.json entries as
	// of this phase-end call -- the "failures recorded" figure folded into
	// this phase's completion note (198.1-04/FEED-04), sourced from the
	// failure log rather than the promotion pipeline. Exposed here so a test
	// can assert the note's content against the runtime's own reported
	// number instead of typing a literal.
	FailuresRecorded int
	// AutoRedirectsEmitted is emitMiddenThresholdRedirect's own return value
	// for this call -- how many midden.json failure categories crossed the
	// auto-REDIRECT threshold and got (or reinforced) a steering signal
	// (198.1-04/FEED-04).
	AutoRedirectsEmitted int
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
	line := fmt.Sprintf("%d promotion candidate(s) -> %d queen-eligible instinct(s)", s.PromotionCandidates, s.QueenEligible)
	if len(s.QueenPromoted) > 0 {
		line += fmt.Sprintf(", %d promoted to the Queen file", len(s.QueenPromoted))
	}
	return line
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
	if store == nil {
		return phaseEndConsolidationSummary{Ran: false, Reason: "no store initialized"}
	}

	// LEARN-03 (204-02-PLAN.md Task 1, ruling (a)): the first real
	// production writer into the evidence-gated credit ledger
	// (cmd/recruitment_credit.go) -- placed BEFORE
	// recordInstinctApplicationsForPhase below (204-03-PLAN.md Task 2,
	// SYN-204-05/06: reordered from the original placement after it).
	// recordInstinctApplicationsForPhase now derives each entry's outcome
	// by looking up this SAME phase's own credit record
	// (recruitmentCreditForContribution) -- if credit were recorded
	// second, every application entry this phase ever writes would read
	// "pending" forever, since recordInstinctApplicationsForPhase's own
	// per-phase idempotency guard (instinctAlreadyAppliedForPhase) means a
	// phase is recorded exactly once and never revisited. Placing credit
	// first is what makes the lookup findable on the very same pass.
	// Never a Go error, never blocks this phase advance -- its own return
	// value shares phaseEndConsolidationSummary's non-blocking shape (see
	// cmd/application_evidence.go for the writer's own doc comment). This
	// function is itself already invoked from cmd/codex_continue.go and
	// cmd/codex_continue_finalize.go -- the one call site that puts this
	// on both check lanes.
	recordPhaseApplicationCredit(phaseID)

	// Record which instincts this phase actually delivered and applied, so
	// the decay/confidence step below (STEP 1 of
	// pkg/memory.ConsolidationService.Run) reads the freshly-recorded
	// application history for this phase, and a repeated phase-end pass for
	// the same phase adds no further entries (198.1-03/FEED-03).
	applicationsRecorded := recordInstinctApplicationsForPhase(phaseID)

	bus := events.NewBus(store, events.DefaultConfig())
	pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())

	// Bounded timeout so a wedged consolidation can never hang a phase
	// advance (T-162-12); the phase-advance record is already committed by
	// the time this function is reached. The goroutine+select wrapper is
	// what makes the bound real: the pipeline never polls ctx between
	// steps, and a stale flock inside storage.FileLocker blocks with no
	// deadline (WR-02).
	ctx, cancel := context.WithTimeout(context.Background(), consolidationLifecycleTimeout)
	defer cancel()

	var result *learn.ConsolidationResult
	var err error
	if !runConsolidationStageBounded(ctx, func() {
		result, err = pipeline.RunConsolidation(ctx)
	}) {
		reason := fmt.Sprintf("consolidation timed out after %s", consolidationLifecycleTimeout)
		fmt.Fprintf(os.Stderr, "phase advanced WITHOUT consolidation — %v\n", reason)
		return phaseEndConsolidationSummary{Ran: false, Reason: reason}
	}

	// ConsolidationService.Run (pkg/memory/consolidate.go) records per-step
	// failures (e.g. a corrupt instincts.json) into result.Errors instead of
	// returning a top-level error -- the pipeline is deliberately
	// non-blocking internally too. Treat a non-empty result.Errors the same
	// as a top-level err: both mean "consolidation did not run cleanly" --
	// EXCEPT a bare "file does not exist" for instincts.json or
	// learning-observations.json, which is a fresh colony's normal starting
	// state, not corruption. Every step inside ConsolidationService.Run
	// already treats a missing file as "start empty"; without this filter
	// runPhaseEndConsolidation would report "WITHOUT consolidation" on every
	// colony's very first phase even though nothing is actually wrong
	// (found while proving 198.1-03's end-to-end test against a colony that
	// starts with neither file).
	if err == nil && result != nil && len(result.Errors) > 0 {
		if real := realConsolidationErrors(result.Errors); len(real) > 0 {
			err = errors.Join(real...)
		}
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

	summary := phaseEndConsolidationSummary{
		Ran:                          true,
		InstinctsDecayed:             result.InstinctsDecayed,
		InstinctsArchived:            result.InstinctsArchived,
		ObservationsDecayed:          result.ObservationsDecayed,
		PromotionCandidates:          len(result.PromotionCandidates),
		QueenEligible:                len(result.QueenEligible),
		ReviewCandidates:             len(result.ReviewCandidates),
		RereadCandidates:             len(result.RereadCandidates),
		QueenPromoted:                append([]string{}, result.QueenPromoted...),
		InstinctApplicationsRecorded: applicationsRecorded,
		FailuresRecorded:             countUnacknowledgedMiddenEntries(store),
	}

	// Feed the memory (198.1-04, FEED-04): a finished phase leaves a note
	// naming what it produced, and a run of the same failure leaves an
	// automatic don't-do-this note -- both AFTER the summary above is fully
	// assembled, so their counts are the ones this summary reports. Neither
	// emission can fail this phase advance; both warn to stderr and continue.
	var cs colony.ColonyState
	_ = store.LoadJSON("COLONY_STATE.json", &cs)
	phase, found := colonyPhaseByID(cs, phaseID)
	if !found {
		phase = colony.Phase{ID: phaseID}
	}
	emitPhaseCompletionFeedback(phase, summary, summary.PromotionCandidates, summary.FailuresRecorded)
	summary.AutoRedirectsEmitted = emitMiddenThresholdRedirect()

	return summary
}

// realConsolidationErrors filters out "file does not exist" load failures
// (e.g. a fresh colony's first phase, before instincts.json or
// learning-observations.json has ever been written) from a consolidation
// step's per-step error list. That condition is normal starting state, not
// corruption -- pkg/memory.ConsolidationService.Run's own steps already
// treat a missing file as "start empty" for every mutation that follows.
// Any other error (corrupt JSON, permission failure, a genuine I/O fault)
// still fails loudly and is returned unfiltered.
func realConsolidationErrors(errs []error) []error {
	real := make([]error, 0, len(errs))
	for _, e := range errs {
		if errors.Is(e, fs.ErrNotExist) {
			continue
		}
		// Queen callback failures are already reported by the pipeline and are
		// excluded from QueenPromoted. Preserve consolidation's established
		// enrichment-not-gate contract: the failed promotion stays failed, but
		// it does not invalidate unrelated decay/archive work.
		if strings.HasPrefix(e.Error(), "queen promote ") {
			continue
		}
		real = append(real, e)
	}
	return real
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
	// subordinate seal-side promotion loop (D-09). Sourced from the
	// pipeline's QueenPromoted (writes that succeeded), never from
	// QueenEligible: an eligible instinct whose write failed must stay OUT
	// of the skip-set so the subordinate writer can recover it, and out of
	// the promoted report so CROWNED-ANTHILL.md stays honest (WR-01).
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

	ctx, cancel := context.WithTimeout(context.Background(), consolidationLifecycleTimeout)
	defer cancel()

	bus := events.NewBus(store, events.DefaultConfig())

	// Bounded like the phase-end path (WR-02): the orchestrator checks
	// ctx.Done() only BETWEEN ant steps, so an in-step block (e.g. a stale
	// flock) is uncancellable without the goroutine+select wrapper. The
	// store pointer is captured locally BEFORE the goroutine starts: an
	// abandoned goroutine must never read the package global, which the
	// caller (or a test teardown) may reassign after the timeout fires.
	orchestrator := curation.NewOrchestrator(store, bus)
	var curResult *curation.CurationResult
	var curErr error
	if !runConsolidationStageBounded(ctx, func() {
		curResult, curErr = orchestrator.Run(ctx, false)
	}) {
		reason := fmt.Sprintf("consolidation timed out after %s", consolidationLifecycleTimeout)
		fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
		return sealConsolidationSummary{Ran: false, Reason: reason}
	}

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

	summary := sealConsolidationSummary{Ants: ants, ReportPath: reportPath}

	// CR-01: a curation failure must short-circuit BEFORE the mutating
	// pipeline runs. The sentinel exists to guard the stores before anything
	// acts on them -- running decay/archive/promotion against a colony the
	// sentinel just flagged corrupt would defeat that guard. And if the
	// pipeline had run anyway, returning Ran:false with an empty
	// QueenPromotedIDs would hand completeSealRuntime an empty skip-set
	// after QUEEN.md was already written, reintroducing the D-09
	// double-write on the failure path.
	if curErr != nil {
		reason := curErr.Error()
		// T-162-11: a curation sentinel abort (corrupt stores detected) must
		// be distinguishable from a transient failure in the one line the
		// operator sees.
		if strings.Contains(reason, "sentinel abort") {
			reason = "curation sentinel detected corrupt stores: " + reason
		}
		fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
		summary.Ran = false
		summary.Reason = reason
		return summary
	}

	pipeline := learn.NewPipeline(store, bus, pipelineConfigForStore())
	var consResult *learn.ConsolidationResult
	var consErr error
	if !runConsolidationStageBounded(ctx, func() {
		consResult, consErr = pipeline.RunConsolidation(ctx)
	}) {
		// The abandoned pipeline goroutine may still be mid-run, so reading
		// consResult here would race; the skip-set stays empty. A duplicate
		// "## Wisdom" entry in this extreme case is the accepted residual
		// cost of never hanging a seal (WR-02).
		reason := fmt.Sprintf("consolidation timed out after %s", consolidationLifecycleTimeout)
		fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
		summary.Ran = false
		summary.Reason = reason
		return summary
	}
	if consErr == nil && consResult != nil && len(consResult.Errors) > 0 {
		if real := realConsolidationErrors(consResult.Errors); len(real) > 0 {
			consErr = errors.Join(real...)
		}
	}

	// Publish the seal consolidation event, matching consolidationSealCmd's
	// own literal topic. Publish failures are non-blocking.
	if payload, err := json.Marshal(map[string]string{
		"type":      "consolidation.seal",
		"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
	}); err == nil {
		_, _ = bus.Publish(ctx, "consolidation.seal", payload, "seal")
	}

	if consErr != nil {
		reason := consErr.Error()
		fmt.Fprintf(os.Stderr, "colony sealed WITHOUT consolidation — %v\n", reason)
		summary.Ran = false
		summary.Reason = reason
		// The pipeline's steps are individually non-blocking, so a failed run
		// may still have written QUEEN.md promotions before the failing step.
		// The D-09 skip-set must reflect what was actually written even when
		// Ran is false, or completeSealRuntime double-writes on this path.
		if consResult != nil {
			summary.QueenPromotedIDs = append([]string{}, consResult.QueenPromoted...)
		}
		return summary
	}

	summary.Ran = true
	summary.InstinctsDecayed = consResult.InstinctsDecayed
	summary.InstinctsArchived = consResult.InstinctsArchived
	summary.ObservationsDecayed = consResult.ObservationsDecayed
	summary.PromotionCandidates = len(consResult.PromotionCandidates)
	summary.QueenEligible = len(consResult.QueenEligible)
	summary.QueenPromotedIDs = append([]string{}, consResult.QueenPromoted...)
	summary.ReviewCandidates = len(consResult.ReviewCandidates)
	summary.RereadCandidates = len(consResult.RereadCandidates)
	return summary
}
