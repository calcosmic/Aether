package cmd

// LEARN-06/LEARN-07 (204-12-PLAN.md): the automatic pass that drives a
// declared, beneficial, promotable candidate from comparison, through the
// gate, into a canary, and out through completion or rollback -- entirely
// by itself, at the end of every check, on both check lanes (owner
// decisions D-01, D-04, D-05). This file introduces no new store, no new
// episode kind, and no new checkpoint mechanism: every mutation below goes
// through runShadowCompare, admitCandidateToCanary, startCanary,
// completeCanary and rollbackCanary -- functions this closure's own gap
// (SC4b/SC5b) found real, tested and completely uncalled.
import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// improvementPassEventKind is the closed vocabulary an improvementPassEvent
// may carry -- ONE card-worthy outcome per candidate this pass touches
// (D-03): a comparison it ran but did not admit, a canary it started, one
// it kept, or one it undid. The comparison and admission steps that lead to
// "started" are reflected only in this summary's own Compared/Admitted
// counters, never as their own separate closing-card line.
type improvementPassEventKind string

const (
	improvementPassEventRefused    improvementPassEventKind = "refused"
	improvementPassEventStarted    improvementPassEventKind = "started"
	improvementPassEventCompleted  improvementPassEventKind = "completed"
	improvementPassEventRolledBack improvementPassEventKind = "rolled_back"
)

// improvementPassEvent is one thing this pass did (or refused to do) to one
// candidate, kept structured so the closing-card renderer can turn it into
// plain English without ever printing the candidate id or verdict token
// itself (204-12, D-03).
type improvementPassEvent struct {
	CandidateID string                   `json:"candidate_id"`
	Kind        improvementPassEventKind `json:"kind"`
	Detail      string                   `json:"detail,omitempty"`
}

// improvementPassSummary is the runtime-facing, non-blocking summary of one
// runAutomaticImprovementPass call. Returned by value only, mirroring
// phaseEndConsolidationSummary's and phaseApplicationCreditSummary's own
// shape -- this pass never returns a Go error a caller could propagate into
// an abort.
type improvementPassSummary struct {
	Ran                  bool
	CandidatesConsidered int
	Compared             int
	Admitted             int
	Refused              int
	CanariesStarted      int
	CanariesCompleted    int
	CanariesRolledBack   int
	Events               []improvementPassEvent
	Failures             []string
}

// improvementPassZeroState reports whether this pass ran and genuinely did
// nothing -- no candidate declared, no canary running. The closing-card
// renderer uses this to render NOTHING AT ALL rather than an empty-looking
// line (204-12, D-03).
func (s improvementPassSummary) improvementPassZeroState() bool {
	return len(s.Events) == 0
}

// improvementPassProjectKnowledgeScopePaths and
// improvementPassRoutingScopePaths name exactly the files each of the two
// canary-promotable scopes may ever change. startCanary's own checkpoint
// must cover exactly these paths, never the whole repository -- an EMPTY
// scope-paths list checkpoints the entire working tree
// (saveRepairCheckpoint's own documented behaviour), which is exactly the
// unbounded blast radius the two-scope boundary (cmd/promotion_gate.go)
// exists to prevent. "Project knowledge" is this project's own learned
// memory of itself; "routing" is the durable colony state a routing
// decision reads.
var improvementPassProjectKnowledgeScopePaths = []string{
	filepath.Join(".aether", "data", "instincts.json"),
}

var improvementPassRoutingScopePaths = []string{
	filepath.Join(".aether", "data", "COLONY_STATE.json"),
}

// improvementPassScopePaths derives the checkpoint scope from the admitted
// scope alone -- never any other input, and never empty for a recognised
// scope (an empty result refuses the canary rather than risk a full-root
// checkpoint; see processNewImprovementCandidate).
func improvementPassScopePaths(scope canaryScope) []string {
	switch scope {
	case canaryScopeProjectKnowledge:
		return improvementPassProjectKnowledgeScopePaths
	case canaryScopeRouting:
		return improvementPassRoutingScopePaths
	default:
		return nil
	}
}

func (s *improvementPassSummary) addEvent(candidateID string, kind improvementPassEventKind, detail string) {
	s.Events = append(s.Events, improvementPassEvent{CandidateID: candidateID, Kind: kind, Detail: detail})
}

func (s *improvementPassSummary) recordFailure(candidateID, reason string) {
	if candidateID != "" {
		reason = fmt.Sprintf("%s: %s", candidateID, reason)
	}
	s.Failures = append(s.Failures, reason)
	fmt.Fprintf(os.Stderr, "improvement pass: %s\n", reason)
}

// improvementPassPhaseGatesPassed reads phaseID's own gate outcome from
// what is already available at this boundary -- the phase's latest durable
// build attempt's own free-check report (cmd/build_attempt.go) -- rather
// than re-running anything. Absent evidence (no durable attempt recorded,
// or no free-check report on it) is read as passed: this pass's one
// production call site (runPhaseEndConsolidation) is itself only ever
// reached after the current phase's own checks have already passed
// (its own doc comment: "Gates have passed by this point"), so missing
// evidence here never means "assume failed."
func improvementPassPhaseGatesPassed(phaseID int) (passed bool, failingGate string) {
	_, attempt, ok := loadLatestBuildAttempt(phaseID)
	if !ok || attempt.FreeChecks == nil {
		return true, ""
	}
	if attempt.FreeChecks.Passed {
		return true, ""
	}
	if len(attempt.FreeChecks.Failed) > 0 {
		return false, attempt.FreeChecks.Failed[0]
	}
	return false, ""
}

// improvementPassCanaryBoundExceeded parses run's own recorded bound and
// reports whether it has been exceeded as of now -- reusing
// canaryBoundExceeded's exact rule (cmd/promotion_gate.go) rather than a
// second one.
func improvementPassCanaryBoundExceeded(run canaryRun, now time.Time) (bool, error) {
	bound, err := time.Parse(time.RFC3339, run.Bound)
	if err != nil {
		return false, fmt.Errorf("parse canary bound %q: %w", run.Bound, err)
	}
	return canaryBoundExceeded(canaryAdmission{Bound: bound}, now), nil
}

// processNewImprovementCandidate drives ONE declared candidate with no
// existing canary run through comparison, gate admission, and canary start
// -- refusing (never erroring) at any step, and recording every outcome
// into summary.
func processNewImprovementCandidate(record shadowCandidateRecord, summary *improvementPassSummary) {
	if _, found, err := loadCanaryRun(record.ID); err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("could not check for an existing canary run: %v", err))
		return
	} else if found {
		// Already has a canary run (running, completed, or rolled back) --
		// this candidate is not "new" and belongs to the running-canary
		// pass below instead, if it is still running.
		return
	}

	outcome, err := runShadowCompare(record.ID)
	if err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("comparison failed: %v", err))
		return
	}
	summary.Compared++

	candidate, err := shadowCandidateToDomain(record)
	if err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("could not rebuild the declared candidate: %v", err))
		return
	}

	admission, err := admitCandidateToCanary(candidate, outcome.Comparison)
	if err != nil {
		// Refused is the one card-worthy outcome for this candidate this
		// pass -- comparison and admission are internal steps toward it,
		// never their own separate closing-card line (D-03: one line per
		// candidate outcome).
		summary.Refused++
		summary.addEvent(record.ID, improvementPassEventRefused, err.Error())
		return
	}
	summary.Admitted++

	scopePaths := improvementPassScopePaths(admission.Scope)
	if len(scopePaths) == 0 {
		summary.recordFailure(record.ID, fmt.Sprintf("no declared checkpoint scope for admitted scope %q -- refusing to start a canary that would checkpoint the entire project", admission.Scope))
		return
	}

	if _, err := startCanary(admission, scopePaths); err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("could not start the canary: %v", err))
		return
	}
	summary.CanariesStarted++
	// Started is this candidate's one card-worthy outcome for this pass
	// (D-03) -- the comparison and the admission that led here are
	// reflected in the Compared/Admitted counters, never a separate line.
	summary.addEvent(record.ID, improvementPassEventStarted, "")
}

// processRunningImprovementCanary drives ONE canary already in status
// running through completion or rollback once its bound is reached.
func processRunningImprovementCanary(run canaryRun, gatesPassed bool, failingGate string, summary *improvementPassSummary) {
	exceeded, err := improvementPassCanaryBoundExceeded(run, time.Now())
	if err != nil {
		summary.recordFailure(run.CandidateID, fmt.Sprintf("could not evaluate the canary's own bound: %v", err))
		return
	}
	if !exceeded {
		return
	}

	if gatesPassed {
		if _, err := completeCanary(run); err != nil {
			summary.recordFailure(run.CandidateID, fmt.Sprintf("could not complete the canary: %v", err))
			return
		}
		summary.CanariesCompleted++
		summary.addEvent(run.CandidateID, improvementPassEventCompleted, "")
		return
	}

	reason := "this phase's own checks did not pass"
	if failingGate != "" {
		reason = fmt.Sprintf("this phase's own check failed: %s", failingGate)
	}
	if _, _, err := rollbackCanary(run, reason); err != nil {
		summary.recordFailure(run.CandidateID, fmt.Sprintf("could not roll back the canary: %v", err))
		return
	}
	summary.CanariesRolledBack++
	summary.addEvent(run.CandidateID, improvementPassEventRolledBack, reason)
}

// runAutomaticImprovementPass is the automatic pass (204-12, D-01, D-04):
// with a declared candidate or a running canary, it drives comparison,
// gate admission, canary start, and canary completion/rollback, entirely
// by itself; with neither, it does no comparison, no gate call, and writes
// no canary or shadow file of any kind. It never returns an error a caller
// could propagate into an abort -- every failure is captured into the
// summary and warned to stderr, exactly like
// phaseEndConsolidationSummary's and recordPhaseApplicationCredit's own
// non-blocking contract.
//
// This function carries no actor, caller-identity, coordinator, autopilot
// or waiver parameter of any kind, and its body branches on none -- the
// only two scopes it can ever promote are the two admitCandidateToCanary
// itself already restricts to (cmd/promotion_gate.go); every retained-
// authority scope is refused there, by that same gate, with no waiver this
// pass could ever offer.
func runAutomaticImprovementPass(phaseID int) improvementPassSummary {
	summary := improvementPassSummary{}
	if store == nil {
		return summary
	}

	var candidateFile shadowCandidateFile
	if err := store.LoadJSON(shadowCandidateStorePath, &candidateFile); err != nil && !errorIsFileNotExist(err) {
		summary.Ran = true
		summary.recordFailure("", fmt.Sprintf("could not read the declared-candidate store: %v", err))
		candidateFile = shadowCandidateFile{}
	}

	canaryFile, err := loadCanaryRunFile()
	if err != nil {
		summary.Ran = true
		summary.recordFailure("", fmt.Sprintf("could not read the canary run store: %v", err))
		canaryFile = canaryRunFile{}
	}
	var runningRuns []canaryRun
	for _, run := range canaryFile.Entries {
		if run.Status == canaryRunStatusRunning {
			runningRuns = append(runningRuns, run)
		}
	}

	if len(candidateFile.Entries) == 0 && len(runningRuns) == 0 {
		// Nothing declared and nothing running: no comparison, no gate
		// call, no canary file, no write of any kind (D-04's "costs
		// nothing measurable" path).
		summary.Ran = true
		return summary
	}
	summary.Ran = true

	for _, record := range candidateFile.Entries {
		summary.CandidatesConsidered++
		processNewImprovementCandidate(record, &summary)
	}

	if len(runningRuns) > 0 {
		gatesPassed, failingGate := improvementPassPhaseGatesPassed(phaseID)
		for _, run := range runningRuns {
			processRunningImprovementCanary(run, gatesPassed, failingGate, &summary)
		}
	}

	return summary
}

// errorIsFileNotExist reports whether err is (or wraps) the standard
// "file does not exist" sentinel -- the honest "nothing declared yet"
// state, as distinct from a genuine read/parse failure this pass must
// still surface rather than silently swallow (matching
// realConsolidationErrors' own fs.ErrNotExist filter in
// cmd/consolidation_lifecycle.go).
func errorIsFileNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
