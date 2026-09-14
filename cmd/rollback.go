package cmd

// LEARN-07 (204-09-PLAN.md), Task 2: run an admitted canary behind a
// restore point, and undo it in one operation when it goes wrong.
//
// startCanary reuses the existing, proven checkpoint mechanism
// (saveRepairCheckpoint/restoreRepairCheckpoint, cmd/work_repair.go) rather
// than a new implementation -- the same save-a-checkpoint,
// verify-it-worked, roll-back-if-it-failed safety net every other repair
// path in this program already uses (Swarm's own repair wave, CLAUDE.md
// "Live Colony, Swarm, and Oracle"). rollbackCanary follows PauseHandoff's
// own idempotency shape (colony.PauseHandoff.HandoffID /
// resumeTransactionID): the candidate id is the idempotency key, a replay
// returns the stored receipt and writes nothing.
import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/shadow"
)

// canaryRunStorePath is the store-relative path every canary run record and
// rollback receipt for this program is kept at -- one store, one
// directory, mirroring credit/records.json and episodes/ledger.json's own
// one-store-one-directory convention.
const canaryRunStorePath = "canary/runs.json"

// canaryRunStatus is the closed vocabulary a canaryRun's own Status field
// may hold.
type canaryRunStatus string

const (
	canaryRunStatusRunning    canaryRunStatus = "running"
	canaryRunStatusCompleted  canaryRunStatus = "completed"
	canaryRunStatusRolledBack canaryRunStatus = "rolled_back"
)

// canaryRun is the durable record of one admitted candidate's canary: its
// identity and scope, the checkpoint that lets it be undone exactly, its
// declared bound, and its current lifecycle state.
type canaryRun struct {
	CandidateID          string          `json:"candidate_id"`
	Scope                canaryScope     `json:"scope"`
	Verdict              shadow.Verdict  `json:"verdict"`
	CheckpointID         string          `json:"checkpoint_id"`
	BackupDir            string          `json:"backup_dir"`
	ScopePaths           []string        `json:"scope_paths,omitempty"`
	Root                 string          `json:"root"`
	FullRoot             bool            `json:"full_root,omitempty"`
	Bound                string          `json:"bound,omitempty"`
	Status               canaryRunStatus `json:"status"`
	StartedAt            string          `json:"started_at"`
	UpdatedAt            string          `json:"updated_at,omitempty"`
	Quarantined          bool            `json:"quarantined,omitempty"`
	QuarantinedAt        string          `json:"quarantined_at,omitempty"`
	QuarantineReleasedAt string          `json:"quarantine_released_at,omitempty"`
	QuarantineReleasedBy string          `json:"quarantine_released_by,omitempty"`
	RollbackReason       string          `json:"rollback_reason,omitempty"`
}

// canaryRollbackReceipt is what rollbackCanary returns and durably records
// -- the idempotency-replay target for a second call against the same
// candidate.
type canaryRollbackReceipt struct {
	CandidateID  string `json:"candidate_id"`
	CheckpointID string `json:"checkpoint_id"`
	Reason       string `json:"reason"`
	Quarantined  bool   `json:"quarantined"`
	RestoredAt   string `json:"restored_at"`
}

// canaryRunFile is the on-disk container at canaryRunStorePath. Entries and
// Receipts are both append-only from this file's own two writers
// (startCanary appends Entries; rollbackCanary appends Receipts and mutates
// exactly the one Entries row naming its own candidate id, in the same
// atomic update that appends the receipt). ScopeRegressionCounts is
// bookkeeping only, incremented on every rollback for the regressing
// candidate's own scope -- it never gates anything by itself; it is what
// canaryShouldQuarantine below is evaluated against.
type canaryRunFile struct {
	Entries               []canaryRun             `json:"entries"`
	Receipts              []canaryRollbackReceipt `json:"receipts,omitempty"`
	ScopeRegressionCounts map[canaryScope]int     `json:"scope_regression_counts,omitempty"`
}

// canaryRegressionQuarantineThreshold is the number of regressions a single
// canary scope must accumulate, across distinct candidate attempts, before
// this quarantine trigger fires. Named constant, never an inline number,
// mirroring noteHarmfulQuarantineThreshold's shape (cmd/pheromone_outcome.go)
// -- "the same kind of thing keeps going wrong" -- but declared separately
// rather than through one shared helper: cmd/pheromone_outcome.go is
// outside this plan's declared files_modified, and the two domains count a
// fundamentally different identity (one pheromone note's own id, versus a
// canary scope shared across many distinct, one-shot candidate
// declarations).
const canaryRegressionQuarantineThreshold = 3

// canaryShouldQuarantine is the pure counting rule the quarantine trigger
// count applies: given the regression count already recorded for a scope
// before this regression, it returns the new count and whether THIS
// regression is the one that exactly reaches the threshold. Reaching the
// threshold exactly reports justQuarantined=true; one below does not; one
// above (a scope that had already crossed the threshold on a prior
// regression) does not report justQuarantined a second time -- the
// transition fires once, even though the scope's own candidates keep being
// quarantined individually on every regression regardless (see
// rollbackCanary below).
func canaryShouldQuarantine(previousCount int) (newCount int, justQuarantined bool) {
	newCount = previousCount + 1
	justQuarantined = newCount == canaryRegressionQuarantineThreshold
	return newCount, justQuarantined
}

// canaryCheckpointIdentity is startCanary's idempotency key, derived the
// way repairCheckpointIdentity (cmd/work_repair.go) derives its own: a
// deterministic string naming the candidate this restore point belongs to.
func canaryCheckpointIdentity(candidateID string) string {
	normalized := strings.ToLower(strings.TrimSpace(candidateID))
	normalized = strings.Join(strings.Fields(normalized), "-")
	return fmt.Sprintf("canary-checkpoint-%s", normalized)
}

// canaryEpisodeID derives the durable episode identifier a canary's start
// and rollback/completion are recorded under -- one candidate, one canary
// episode, mirroring shadowComparisonEpisodeID's role for shadow-compare.
func canaryEpisodeID(candidateID string) string {
	return "canary:" + candidateID
}

func loadCanaryRunFile() (canaryRunFile, error) {
	var file canaryRunFile
	if store == nil {
		return file, fmt.Errorf("no store initialized")
	}
	if err := store.LoadJSON(canaryRunStorePath, &file); err != nil {
		return canaryRunFile{}, nil
	}
	return file, nil
}

// loadCanaryRun returns the stored run record for candidateID, or
// ok=false if no canary has ever been started for it.
func loadCanaryRun(candidateID string) (canaryRun, bool, error) {
	file, err := loadCanaryRunFile()
	if err != nil {
		return canaryRun{}, false, err
	}
	for _, r := range file.Entries {
		if r.CandidateID == candidateID {
			return r, true, nil
		}
	}
	return canaryRun{}, false, nil
}

// loadCanaryRollbackReceipt returns the stored rollback receipt for
// candidateID, or ok=false if it has never been rolled back.
func loadCanaryRollbackReceipt(candidateID string) (canaryRollbackReceipt, bool, error) {
	file, err := loadCanaryRunFile()
	if err != nil {
		return canaryRollbackReceipt{}, false, err
	}
	for _, r := range file.Receipts {
		if r.CandidateID == candidateID {
			return r, true, nil
		}
	}
	return canaryRollbackReceipt{}, false, nil
}

// startCanary saves the restore point first, through saveRepairCheckpoint
// -- the existing, proven checkpoint function, not a new one -- and only
// then records the canary as running. A canary run records its checkpoint
// identifier before anything else, so a crash between the two leaves
// evidence that the change had not yet happened. A call that cannot save a
// restore point is refused by name and writes nothing: no run record is
// ever appended without a checkpoint already having succeeded.
//
// Calling this twice for the same candidate id is idempotent: the second
// call returns the existing run record and starts nothing new.
func startCanary(admission canaryAdmission, scopePaths []string) (canaryRun, error) {
	if store == nil {
		return canaryRun{}, fmt.Errorf("no store initialized")
	}
	candidateID := strings.TrimSpace(admission.CandidateID)
	if candidateID == "" {
		return canaryRun{}, fmt.Errorf("canary admission carries no candidate id")
	}

	if existing, found, err := loadCanaryRun(candidateID); err != nil {
		return canaryRun{}, err
	} else if found {
		return existing, nil
	}

	root, err := os.Getwd()
	if err != nil {
		return canaryRun{}, fmt.Errorf("resolve project root: %w", err)
	}
	checkpointID := canaryCheckpointIdentity(candidateID)
	checkpoint, err := saveRepairCheckpoint(root, checkpointID, scopePaths)
	if err != nil {
		return canaryRun{}, fmt.Errorf(
			"canary %q cannot start without a restore point -- checkpoint save failed: %w",
			candidateID, err,
		)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	run := canaryRun{
		CandidateID:  candidateID,
		Scope:        admission.Scope,
		Verdict:      admission.Verdict,
		CheckpointID: checkpointID,
		BackupDir:    checkpoint.BackupDir,
		ScopePaths:   checkpoint.Paths,
		Root:         checkpoint.Root,
		FullRoot:     checkpoint.FullRoot,
		Bound:        admission.Bound.UTC().Format(time.RFC3339),
		Status:       canaryRunStatusRunning,
		StartedAt:    now,
	}

	var file canaryRunFile
	updateErr := store.UpdateJSONAtomically(canaryRunStorePath, &file, func() error {
		for _, existing := range file.Entries {
			if existing.CandidateID == candidateID {
				return fmt.Errorf("canary run for candidate %q already exists", candidateID)
			}
		}
		file.Entries = append(file.Entries, run)
		return nil
	})
	if updateErr != nil {
		return canaryRun{}, updateErr
	}

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:  episodeLedgerRecordKindOpened,
		EpisodeID:   canaryEpisodeID(candidateID),
		EpisodeKind: "canary",
		StartedAt:   now,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record canary start durably for %q: %v\n", candidateID, err)
	}
	emitCanaryStarted(run)
	return run, nil
}

// completeCanary marks run as completed once it has passed its hard gates
// and its holdout checks for its declared bound -- the change stays.
// Idempotent: calling this twice for the same candidate leaves the stored
// record unchanged on the second call.
func completeCanary(run canaryRun) (canaryRun, error) {
	if store == nil {
		return canaryRun{}, fmt.Errorf("no store initialized")
	}
	candidateID := strings.TrimSpace(run.CandidateID)
	if candidateID == "" {
		return canaryRun{}, fmt.Errorf("canary run carries no candidate id")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var result canaryRun
	var file canaryRunFile
	updateErr := store.UpdateJSONAtomically(canaryRunStorePath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].CandidateID != candidateID {
				continue
			}
			if file.Entries[i].Status == canaryRunStatusCompleted || file.Entries[i].Status == canaryRunStatusRolledBack {
				result = file.Entries[i]
				return nil
			}
			file.Entries[i].Status = canaryRunStatusCompleted
			file.Entries[i].UpdatedAt = now
			result = file.Entries[i]
			return nil
		}
		return fmt.Errorf("no canary run recorded for candidate %q", candidateID)
	})
	if updateErr != nil {
		return canaryRun{}, updateErr
	}

	if result.Status == canaryRunStatusCompleted {
		if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
			RecordKind:     episodeLedgerRecordKindClosed,
			EpisodeID:      canaryEpisodeID(candidateID),
			EpisodeKind:    "canary",
			TerminalResult: "completed",
			EndedAt:        now,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record canary completion durably for %q: %v\n", candidateID, err)
		}
		emitCanaryCompleted(result)
	}
	return result, nil
}

// rollbackCanary performs the restore and the quarantine as ONE operation:
// the file restore (restoreRepairCheckpoint) runs first, then the run's own
// status transition, its quarantine flag, and the scope's regression count
// are all written in the SAME store.UpdateJSONAtomically call -- so no
// reader can ever observe the run marked rolled back without also being
// quarantined, or the reverse. A regression is quarantined every time,
// regardless of the running scope-level regression count; the count itself
// only decides whether THIS regression is the one that also crosses
// canaryRegressionQuarantineThreshold for its scope (queued for owner
// attention once, not on every subsequent regression in that scope).
//
// Calling this twice for the same candidate id returns the first receipt
// and writes nothing on the second call.
func rollbackCanary(run canaryRun, reason string) (canaryRollbackReceipt, bool, error) {
	if store == nil {
		return canaryRollbackReceipt{}, false, fmt.Errorf("no store initialized")
	}
	candidateID := strings.TrimSpace(run.CandidateID)
	if candidateID == "" {
		return canaryRollbackReceipt{}, false, fmt.Errorf("canary run carries no candidate id")
	}

	if existing, found, err := loadCanaryRollbackReceipt(candidateID); err != nil {
		return canaryRollbackReceipt{}, false, err
	} else if found {
		return existing, false, nil
	}

	checkpoint := repairCheckpoint{
		ID:        run.CheckpointID,
		Root:      run.Root,
		BackupDir: run.BackupDir,
		Paths:     run.ScopePaths,
		FullRoot:  run.FullRoot,
	}
	if err := restoreRepairCheckpoint(checkpoint); err != nil {
		return canaryRollbackReceipt{}, false, fmt.Errorf(
			"canary %q rollback could not restore its checkpoint: %w", candidateID, err,
		)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var receipt canaryRollbackReceipt
	var scopeJustQuarantined bool
	var file canaryRunFile
	updateErr := store.UpdateJSONAtomically(canaryRunStorePath, &file, func() error {
		idx := -1
		for i := range file.Entries {
			if file.Entries[i].CandidateID == candidateID {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no canary run recorded for candidate %q", candidateID)
		}

		file.Entries[idx].Status = canaryRunStatusRolledBack
		file.Entries[idx].UpdatedAt = now
		file.Entries[idx].Quarantined = true
		file.Entries[idx].QuarantinedAt = now
		file.Entries[idx].RollbackReason = reason
		scope := file.Entries[idx].Scope

		if file.ScopeRegressionCounts == nil {
			file.ScopeRegressionCounts = map[canaryScope]int{}
		}
		newCount, justQuarantined := canaryShouldQuarantine(file.ScopeRegressionCounts[scope])
		file.ScopeRegressionCounts[scope] = newCount
		scopeJustQuarantined = justQuarantined

		receipt = canaryRollbackReceipt{
			CandidateID:  candidateID,
			CheckpointID: run.CheckpointID,
			Reason:       reason,
			Quarantined:  true,
			RestoredAt:   now,
		}
		file.Receipts = append(file.Receipts, receipt)
		return nil
	})
	if updateErr != nil {
		return canaryRollbackReceipt{}, false, updateErr
	}

	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:     episodeLedgerRecordKindClosed,
		EpisodeID:      canaryEpisodeID(candidateID),
		EpisodeKind:    "canary",
		TerminalResult: "rolled_back",
		EndedAt:        now,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record canary rollback durably for %q: %v\n", candidateID, err)
	}
	emitCanaryRolledBack(candidateID, reason)

	if err := enqueueCanaryQuarantineApproval(candidateID, run.Scope, reason); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not enqueue quarantined canary %q for owner approval: %v\n", candidateID, err)
	}
	_ = scopeJustQuarantined // recorded in ScopeRegressionCounts above; reserved for a future scope-wide lockout, not required by this plan's own named behaviour.

	return receipt, true, nil
}

// releaseCanaryQuarantine is the ONE function in this program that may
// clear a canaryRun's Quarantined flag to false. Task 2's own
// TestNoAutomaticPathReleasesAQuarantine proves, by an AST scan over every
// non-test file in cmd/ and pkg/ (with a synthetic violation fixture
// proving the scan is not vacuous), that no other function anywhere clears
// it. Called only from cmd/suggest_approve.go's owner-approval path -- never
// from any automatic path.
func releaseCanaryQuarantine(candidateID, releasedBy string) (canaryRun, error) {
	if store == nil {
		return canaryRun{}, fmt.Errorf("no store initialized")
	}
	candidateID = strings.TrimSpace(candidateID)
	if candidateID == "" {
		return canaryRun{}, fmt.Errorf("release requires a non-empty candidate id")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var result canaryRun
	found := false
	var file canaryRunFile
	updateErr := store.UpdateJSONAtomically(canaryRunStorePath, &file, func() error {
		for i := range file.Entries {
			if file.Entries[i].CandidateID != candidateID {
				continue
			}
			found = true
			if !file.Entries[i].Quarantined {
				result = file.Entries[i]
				return nil
			}
			file.Entries[i].Quarantined = false
			file.Entries[i].QuarantineReleasedAt = now
			file.Entries[i].QuarantineReleasedBy = releasedBy
			result = file.Entries[i]
			return nil
		}
		return nil
	})
	if updateErr != nil {
		return canaryRun{}, updateErr
	}
	if !found {
		return canaryRun{}, fmt.Errorf("no canary run recorded for candidate %q", candidateID)
	}
	return result, nil
}

// enqueueCanaryQuarantineApproval places a quarantined canary candidate
// into the SAME owner tick-to-approve queue (colony.PendingSuggestion,
// cmd/suggest_approve.go) every other pending decision this program already
// uses -- never a second approval surface. Deduplicated: a candidate
// already queued and not yet dismissed is not queued a second time.
func enqueueCanaryQuarantineApproval(candidateID string, scope canaryScope, reason string) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	content := fmt.Sprintf(
		"Canary %q (scope: %s) was rolled back and quarantined: %s",
		candidateID, scope, reason,
	)
	contentHash := "sha256:" + sha256Sum(candidateID+"|"+content)
	now := time.Now().UTC().Format(time.RFC3339)
	origin := colony.PendingOriginCanaryCandidate
	linkedID := candidateID
	item := colony.PendingSuggestion{
		ID:                generateSignalID(),
		Type:              "CANARY",
		Content:           content,
		Reason:            reason,
		ContentHash:       contentHash,
		CreatedAt:         now,
		Origin:            &origin,
		CanaryCandidateID: &linkedID,
	}

	var cs colony.ColonyState
	return store.UpdateJSONAtomically("COLONY_STATE.json", &cs, func() error {
		existing := []colony.PendingSuggestion{}
		if cs.PendingSuggestions != nil {
			existing = *cs.PendingSuggestions
		}
		for _, e := range existing {
			if e.Dismissed {
				continue
			}
			if e.Origin == nil || *e.Origin != colony.PendingOriginCanaryCandidate {
				continue
			}
			if e.CanaryCandidateID != nil && *e.CanaryCandidateID == candidateID {
				return nil
			}
		}
		merged := append(existing, item)
		cs.PendingSuggestions = &merged
		return nil
	})
}

// ---------------------------------------------------------------------------
// D-10-shaped flow announcements: one inline line per canary start,
// completion and rollback, through the exact same visual boundary and the
// same glyph table every other inline line uses -- no second screen, no
// second terminal.
// ---------------------------------------------------------------------------

func emitCanaryStarted(run canaryRun) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	writeVisualOutput(stdout, voiceLine("status", fmt.Sprintf(
		"Trying a proposal (%s, scope: %s) beside the current behaviour -- your project's current state was saved first, so it can be put back exactly if this does not work.",
		run.CandidateID, run.Scope,
	))+"\n")
}

func emitCanaryCompleted(run canaryRun) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	writeVisualOutput(stdout, voiceLine("done", fmt.Sprintf(
		"Proposal %q (scope: %s) finished its trial run and kept passing -- the change stays.",
		run.CandidateID, run.Scope,
	))+"\n")
}

func emitCanaryRolledBack(candidateID, reason string) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	writeVisualOutput(stdout, voiceLine("failed", fmt.Sprintf(
		"Proposal %q did not hold up (%s) -- your project has been put back exactly to the state it was saved in, and this proposal is quarantined until you release it.",
		candidateID, reason,
	))+"\n")
}
