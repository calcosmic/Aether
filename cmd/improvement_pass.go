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
	"sort"
	"strings"
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
	// improvementPassEventProposalWritten (204-16, SC5d) is the one
	// card-worthy outcome the repeated-intervention proposal trigger below
	// can ever add: a plain-English case was written to an isolated branch
	// for a person to read. There is no "proposal refused" event here --
	// triggerRepeatedInterventionProposal's own refusals (a dirty tree, a
	// branch collision) are non-blocking and recorded only in
	// summary.Failures, never a closing-card line, per this pass's existing
	// non-blocking discipline.
	improvementPassEventProposalWritten improvementPassEventKind = "proposal_written"
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
//
// WR-02 (204-REVIEW.md): because of that call-site guarantee, gatesPassed
// is true on the overwhelming majority of calls, REGARDLESS of whether the
// canary being evaluated (processRunningImprovementCanary, below) actually
// has anything to do with what the current phase's checks exercised. This
// is a deliberate, lightweight, best-effort signal -- "the colony's checks
// are healthy right now" -- not a scope-specific grade of the canary's own
// change. It is not yet a genuine evaluation of "did admitting THIS
// candidate cause a regression in THIS candidate's own scope"; a canary
// whose scope (.aether/data/instincts.json or .aether/data/COLONY_STATE.json)
// the current phase's checks never touch will still read as "gates
// passed" and be completed rather than rolled back, purely because nothing
// else happened to fail on that phase. Treat gatesPassed as a proxy until
// a scope-specific evaluator exists, the same way shadowEvaluator's own
// doc comment (cmd/shadow_cmds.go) names its own placeholder-vs-real
// history, so a future reader does not mistake "gates passed" for "this
// canary was graded."
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

// improvementRefusalPath is the store-relative path recordImprovementRefusal
// and loadImprovementRefusal persist to -- WR-01 (204-REVIEW.md)'s marker
// so a refused declaration is compared and reported exactly once, never on
// every future check forever.
const improvementRefusalPath = "shadow/refusals.json"

// improvementRefusalRecord is one durable "this exact declaration was
// refused" marker, keyed by (CandidateID, ContentDigest) -- the same
// ContentDigest declareShadowCandidate already computes and stores on the
// candidate record, so a genuine edit (a different declaration under the
// same candidate id) carries a different digest and is never mistaken for
// the declaration this marker names.
type improvementRefusalRecord struct {
	CandidateID   string `json:"candidate_id"`
	ContentDigest string `json:"content_digest"`
	Reason        string `json:"reason"`
	RefusedAt     string `json:"refused_at"`
}

// improvementRefusalFile is the on-disk container at improvementRefusalPath.
type improvementRefusalFile struct {
	Entries []improvementRefusalRecord `json:"entries"`
}

// loadImprovementRefusal returns candidateID's own stored refusal marker,
// if one exists -- a missing or unreadable file is read as "never
// refused" rather than an error, since a fresh colony with no refusal
// history is normal starting state, the same convention
// learningEntryHasHelpfulApplication and sweepIgnoredGuidanceApplications
// already use for their own stores.
func loadImprovementRefusal(candidateID string) (improvementRefusalRecord, bool, error) {
	if store == nil {
		return improvementRefusalRecord{}, false, fmt.Errorf("no store initialized")
	}
	var file improvementRefusalFile
	if err := store.LoadJSON(improvementRefusalPath, &file); err != nil {
		if errorIsFileNotExist(err) {
			return improvementRefusalRecord{}, false, nil
		}
		return improvementRefusalRecord{}, false, err
	}
	for _, rec := range file.Entries {
		if rec.CandidateID == candidateID {
			return rec, true, nil
		}
	}
	return improvementRefusalRecord{}, false, nil
}

// recordImprovementRefusal durably marks candidateID's declaration
// (identified by contentDigest) as refused, replacing any prior marker for
// the same candidate id -- a candidate can only ever carry one live
// declaration at a time, so one marker per id is always enough.
func recordImprovementRefusal(candidateID, contentDigest, reason string) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	var file improvementRefusalFile
	return store.UpdateJSONAtomically(improvementRefusalPath, &file, func() error {
		now := time.Now().UTC().Format(time.RFC3339)
		for i := range file.Entries {
			if file.Entries[i].CandidateID == candidateID {
				file.Entries[i].ContentDigest = contentDigest
				file.Entries[i].Reason = reason
				file.Entries[i].RefusedAt = now
				return nil
			}
		}
		file.Entries = append(file.Entries, improvementRefusalRecord{
			CandidateID:   candidateID,
			ContentDigest: contentDigest,
			Reason:        reason,
			RefusedAt:     now,
		})
		return nil
	})
}

// removeImprovementRefusal drops candidateID's own stored refusal marker,
// if any -- called once a re-declared candidate is genuinely admitted, so
// the store does not accumulate a marker for a declaration that is no
// longer live. Best-effort: a missing file or a missing entry is a no-op,
// never an error.
func removeImprovementRefusal(candidateID string) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	var file improvementRefusalFile
	return store.UpdateJSONAtomically(improvementRefusalPath, &file, func() error {
		kept := file.Entries[:0]
		for _, rec := range file.Entries {
			if rec.CandidateID != candidateID {
				kept = append(kept, rec)
			}
		}
		file.Entries = kept
		return nil
	})
}

// processNewImprovementCandidate drives ONE declared candidate with no
// existing canary run through comparison, gate admission, and canary start
// -- refusing (never erroring) at any step, and recording every outcome
// into summary.
//
// WR-01 (204-REVIEW.md): a candidate whose CURRENT declaration was already
// refused on a prior check is skipped entirely here -- no re-comparison, no
// repeated improvementPassEventRefused card line, forever -- via
// loadImprovementRefusal/recordImprovementRefusal's ContentDigest-keyed
// marker. A genuine edit to the declaration (a different ContentDigest
// under the same candidate id) is still compared fresh: the owner may have
// fixed exactly what admitCandidateToCanary objected to.
func processNewImprovementCandidate(record shadowCandidateRecord, summary *improvementPassSummary) {
	if refusal, found, err := loadImprovementRefusal(record.ID); err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("could not check for an existing refusal marker: %v", err))
		return
	} else if found && refusal.ContentDigest == record.ContentDigest {
		// Already reported as refused for this exact declaration -- no
		// event, no failure, no re-comparison. Silent by design, the same
		// way an already-admitted candidate's existing canary run below is
		// silently skipped (it "belongs to the running-canary pass
		// instead").
		return
	}

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
		// WR-01 (204-REVIEW.md): mark this exact declaration refused so it
		// is never re-compared or re-reported again unless it changes.
		// Non-blocking, matching this pass's own contract: a write failure
		// here never turns a refusal into an error, it only means the next
		// check will (harmlessly) re-evaluate and re-report this same
		// declaration once more.
		if markErr := recordImprovementRefusal(record.ID, record.ContentDigest, err.Error()); markErr != nil {
			summary.recordFailure(record.ID, fmt.Sprintf("could not record refusal marker: %v", markErr))
		}
		return
	}
	summary.Admitted++
	// A candidate that was previously refused and has now been genuinely
	// admitted (a real edit fixed whatever was wrong) no longer needs its
	// stale refusal marker. Best-effort: never blocks admission.
	if err := removeImprovementRefusal(record.ID); err != nil {
		summary.recordFailure(record.ID, fmt.Sprintf("could not clear stale refusal marker: %v", err))
	}

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
//
// WR-02 (204-REVIEW.md): gatesPassed is improvementPassPhaseGatesPassed's
// own proxy signal -- see that function's doc comment for the full
// account. Because that signal is almost always true regardless of
// whether this specific canary's own scope was actually implicated by the
// current phase's checks, this function will almost always choose
// "complete" over "roll back" once a canary's bound is reached. This is
// the deliberate, lightweight, best-effort design this pass shipped with
// (204-12), not yet a scope-specific regression check.
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
	summary.Ran = true

	var candidateFile shadowCandidateFile
	if err := store.LoadJSON(shadowCandidateStorePath, &candidateFile); err != nil && !errorIsFileNotExist(err) {
		summary.recordFailure("", fmt.Sprintf("could not read the declared-candidate store: %v", err))
		candidateFile = shadowCandidateFile{}
	}

	canaryFile, err := loadCanaryRunFile()
	if err != nil {
		summary.recordFailure("", fmt.Sprintf("could not read the canary run store: %v", err))
		canaryFile = canaryRunFile{}
	}
	var runningRuns []canaryRun
	for _, run := range canaryFile.Entries {
		if run.Status == canaryRunStatusRunning {
			runningRuns = append(runningRuns, run)
		}
	}

	// Nothing declared and nothing running: no comparison, no gate call, no
	// canary file, no write of any kind (D-04's "costs nothing measurable"
	// path) -- but the repeated-intervention proposal trigger below still
	// runs, since it reads a completely independent signal (the durable
	// episode ledger, not the shadow-candidate/canary stores) and is itself
	// a genuine no-op read when the ledger names no repeated category.
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

	// 204-16 (SC5d, D-01, D-04, D-05): the automatic, evidence-gated
	// source-improvement trigger -- proposeSourceImprovement's first real
	// production caller. Runs unconditionally at the end of every pass, on
	// the same non-blocking contract as everything above: a refusal (dirty
	// tree, branch collision, no store) is recorded into summary.Failures
	// and the check continues.
	triggerRepeatedInterventionProposal(&summary)

	return summary
}

// triggerRepeatedInterventionProposal (204-16-PLAN.md Task 1, SC5d) is
// proposeSourceImprovement's first real production caller: it builds the
// preventable-intervention figure over every episode already on disk
// (readEpisodeLedger, unbounded window -- no sentinel list is needed here,
// only the declared episodeInterventionKind categories
// collectPreventableInterventions already classifies), counts how many
// DISTINCT episodes carry each declared category, and when any single
// category reaches sourceProposalRepeatedInterventionThreshold, proposes a
// source change naming that category and the real episode identifiers that
// carried it.
//
// This function IS the "automatic trigger" WINDOWS.md entry 45 and this
// plan's threat model (T-204-16-01) refer to, and its name is added to
// cmd/source_proposal_test.go's sourceProposalReachabilityEntryPoints in
// the same change that introduces it (T-204-16-01's own mitigation) -- so
// TestSourceProposalCannotMergePublishOrDeploy's call-graph walk covers
// this entry point too, not only proposeSourceImprovement itself.
//
// The candidate identity is derived deterministically from the repeated
// category alone (never from the evidence list or a timestamp), so the
// same repeated problem always proposes under the same identity and
// proposeSourceImprovement's own existing replay rule collapses a second
// run over unchanged evidence to no second branch. Evidence is always the
// real episode identifiers the ledger holds -- never a synthesised or
// hand-typed list. The one file the change set ever writes lives under
// .aether/reviews-archive/proposals/ -- an existing, already-tracked
// archive directory that cmd/install_cmd.go's hubExcludeDirs already
// excludes from every hub sync (any path component named
// "reviews-archive" is skipped, confirmed by reading
// listFilesRecursiveWithExclusion/pathHasExcludedComponent), and which,
// unlike .aether/data/, is NOT listed in this repository's own .gitignore
// -- so the proposal file can actually be `git add`ed and committed onto
// its own isolated branch. .aether/proposals/ (the plan's own suggested
// path) was rejected for this reason: it carries no hubExcludeDirs entry
// of its own today, so it would be synced into every downstream colony's
// hub install the next time this project runs `aether publish` -- see
// 204-16-SUMMARY.md for the full directory-safety check performed here.
func triggerRepeatedInterventionProposal(summary *improvementPassSummary) {
	if store == nil {
		return
	}
	records, err := readEpisodeLedger()
	if err != nil {
		summary.recordFailure("", fmt.Sprintf("could not read the episode ledger for the automatic proposal check: %v", err))
		return
	}

	episodesByCategory := map[string]map[string]bool{}
	for _, entry := range collectPreventableInterventions(records) {
		set := episodesByCategory[entry.Category]
		if set == nil {
			set = map[string]bool{}
			episodesByCategory[entry.Category] = set
		}
		set[entry.EpisodeID] = true
	}

	categories := make([]string, 0, len(episodesByCategory))
	for category := range episodesByCategory {
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		episodeSet := episodesByCategory[category]
		if len(episodeSet) < sourceProposalRepeatedInterventionThreshold {
			continue
		}

		evidence := make([]string, 0, len(episodeSet))
		for episodeID := range episodeSet {
			evidence = append(evidence, episodeID)
		}
		sort.Strings(evidence)

		candidateID := sourceProposalRepeatedInterventionCandidateID(category)
		changes := sourceProposalRepeatedInterventionChangeSet(category, evidence)

		_, created, proposeErr := proposeSourceImprovement(candidateID, evidence, changes)
		if proposeErr != nil {
			summary.recordFailure(candidateID, fmt.Sprintf("could not propose a source change for a repeated intervention: %v", proposeErr))
			continue
		}
		if created {
			summary.addEvent(candidateID, improvementPassEventProposalWritten, category)
		}
	}
}

// sourceProposalRepeatedInterventionCandidateID derives the deterministic
// candidate identity a repeated intervention category always proposes
// under -- from the category text alone, so the same repeated problem
// always resolves to the same candidate and the same proposal file path,
// regardless of which or how many episodes happen to carry it on a given
// run.
func sourceProposalRepeatedInterventionCandidateID(category string) string {
	return "repeated-intervention-" + sourceProposalSanitizeForBranch(category)
}

// sourceProposalRepeatedInterventionProposalsDir is the one directory a
// repeated-intervention proposal ever writes into -- see
// triggerRepeatedInterventionProposal's own doc comment for why this
// directory, and not the plan's originally-suggested .aether/proposals/,
// is the safe choice.
const sourceProposalRepeatedInterventionProposalsDir = ".aether/reviews-archive/proposals"

// sourceProposalRepeatedInterventionChangeSet builds the one-file change
// set a repeated-intervention proposal ever writes: a plain-English case
// naming the repeated category, the real episodes that carried it, and the
// change being proposed -- no code, nothing outside this one file. This is
// a case for a person to read, never an applied change.
func sourceProposalRepeatedInterventionChangeSet(category string, evidence []string) sourceChangeSet {
	path := filepath.Join(sourceProposalRepeatedInterventionProposalsDir, sourceProposalRepeatedInterventionCandidateID(category)+".md")

	var body strings.Builder
	fmt.Fprintf(&body, "# Repeated intervention: %s\n\n", category)
	fmt.Fprintf(&body, "The owner has intervened %d or more times for the same reason -- %q -- across distinct episodes this colony recorded:\n\n", sourceProposalRepeatedInterventionThreshold, category)
	for _, episodeID := range evidence {
		fmt.Fprintf(&body, "- %s\n", episodeID)
	}
	body.WriteString("\nThis case is proposing that the running program change so this stops requiring a person every time. ")
	body.WriteString("Nothing here has been merged, applied, or acted on automatically -- this is a written case on its own isolated branch, for a person to read, evaluate, and act on by hand.\n")

	return sourceChangeSet{
		Files: []sourceChangeSetFile{
			{Path: path, Content: body.String()},
		},
		Message: fmt.Sprintf("propose: stop repeating the intervention %q", category),
	}
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
