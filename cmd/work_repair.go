package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// runBoundedRepairRound (D-09) is the one bounded, checkpointed repair path
// build and check both use: save a checkpoint, run one repair wave, verify
// once more, and on continued failure restore the checkpoint exactly and
// return a paused result. There is no retry loop and no second wave.
//
// It reuses the existing budgeted, receipted ledger types
// (autopilotRepairLedger / autopilotRepairReceipt / autopilotRepairEvaluation,
// cmd/autopilot_policy.go) rather than a second budget tracker, and the
// checkpoint identity is an idempotency key (SYN-201-10, mirroring
// colony.PauseHandoff.HandoffID / resumeTransactionID's own replay pattern):
// replaying the same phase+check identity returns the existing receipt
// instead of producing a second recovery effect, and a second automatic
// attempt for the same failing verification is refused by name rather than
// silently re-run.
//
// This file has no lifecycle-transaction dependency: the checkpoint itself
// is a plain directory snapshot of the caller's declared permitted scope
// (input.PermittedScope), copied via the same backupCopyFile/backupCopyDir
// helpers medic's own repair backup already uses (cmd/medic_repair.go) --
// reused, not re-implemented.
type repairRoundInput struct {
	Phase          int
	Attempt        string
	Check          string
	Root           string
	Evidence       []string
	PermittedScope []string
	PlannedAction  string
	Baseline       string
	SafetySafe     bool
	AuthoritySafe  bool
	ScopeSafe      bool
}

// repairRoundOutcome is what runBoundedRepairRound reports back to its
// caller: whether the round actually ran, whether an existing receipt was
// replayed instead, whether it passed, and -- on a failed round -- that the
// checkpoint was restored.
type repairRoundOutcome struct {
	Ran          bool
	Replayed     bool
	Passed       bool
	Restored     bool
	CheckpointID string
	Receipt      autopilotRepairReceipt
	Reason       string
}

// runBoundedRepairRound executes (or replays) exactly one bounded repair
// round for the failing verification described by input.
//
// now supplies the clock (defaults to time.Now().UTC()); persist durably
// saves the ledger after the planned and completed receipt (required,
// mirroring executeAutopilotRepair's own contract); repair performs the one
// repair wave; verify re-runs the same check exactly once and reports
// pass/fail plus evidence. announceSave/announceRestore let the caller
// render D-10's flow announcements at the exact moment each occurs -- save
// before the repair wave dispatches, restore only after a failed
// re-verification -- and may be nil.
func runBoundedRepairRound(
	ledger *autopilotRepairLedger,
	input repairRoundInput,
	now func() time.Time,
	persist func(autopilotRepairLedger) error,
	repair func() error,
	verify func() (bool, []string, error),
	announceSave func(checkpointID string),
	announceRestore func(checkpointID string),
) (repairRoundOutcome, error) {
	if ledger == nil {
		return repairRoundOutcome{}, fmt.Errorf("repair ledger is nil")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	checkpointID := repairCheckpointIdentity(input.Phase, input.Check)

	// Idempotent replay (SYN-201-10): a checkpoint identity that already
	// has a receipt is a spent round -- refused by name, not re-run,
	// whether this is a genuine second automatic attempt or a replayed
	// call for the identical identity. Both of D-09's "no second attempt"
	// and "replay returns the existing receipt" guarantees are the same
	// mechanism here on purpose.
	if existing, ok := findAutopilotRepairReceiptByCheckpoint(*ledger, checkpointID); ok {
		return repairRoundOutcome{
			Ran: false, Replayed: true, Passed: existing.Status == autopilotRepairPassed,
			Receipt: existing, CheckpointID: checkpointID,
			Reason: fmt.Sprintf("repair round for phase %d check %q is already spent (receipt %s)", input.Phase, input.Check, existing.ID),
		}, nil
	}

	failure := autopilotRepairFailure{
		Phase: input.Phase, Attempt: input.Attempt, Check: input.Check,
		Evidence: input.Evidence, PlannedAction: input.PlannedAction, Baseline: input.Baseline,
		PermittedScope: input.PermittedScope, ScopeSafe: input.ScopeSafe,
		SafetySafe: input.SafetySafe, AuthoritySafe: input.AuthoritySafe,
	}
	// repairEligibilityEvaluation (CAP-024) wraps classifyAutopilotRepairFailure
	// with the unresolved-flags and recurring-failure-class inputs -- the
	// eligibility decision itself is unchanged, only the recorded reason is
	// enriched to name which input drove it.
	evaluation := repairEligibilityEvaluation(failure, ledger.Remaining)
	if !evaluation.Eligible {
		return repairRoundOutcome{Ran: false, CheckpointID: checkpointID, Reason: evaluation.Reason}, nil
	}

	checkpoint, err := saveRepairCheckpoint(input.Root, checkpointID, input.PermittedScope)
	if err != nil {
		return repairRoundOutcome{}, fmt.Errorf("save repair checkpoint: %w", err)
	}
	defer os.RemoveAll(checkpoint.BackupDir)

	if announceSave != nil {
		announceSave(checkpointID)
	}

	receipt, execErr := executeAutopilotRepair(ledger, failure, now, persist, repair, func(string) (bool, []string, error) {
		return verify()
	})
	if execErr != nil && receipt.ID == "" {
		// Nothing was completed -- the round never meaningfully ran (a
		// setup or persistence failure before any receipt existed), so
		// there is nothing on disk to restore.
		return repairRoundOutcome{CheckpointID: checkpointID}, execErr
	}

	// Tag the completed receipt with its checkpoint identity so a future
	// replay call can find it (findAutopilotRepairReceiptByCheckpoint
	// above). executeAutopilotRepair already persisted the receipt without
	// this tag; this is the one place CheckpointID is durably attached.
	for i := range ledger.Receipts {
		if ledger.Receipts[i].ID == receipt.ID {
			ledger.Receipts[i].CheckpointID = checkpointID
			receipt = ledger.Receipts[i]
		}
	}
	if persist != nil {
		_ = persist(*ledger)
	}

	outcome := repairRoundOutcome{
		Ran: true, Passed: receipt.Status == autopilotRepairPassed,
		Receipt: receipt, CheckpointID: checkpointID,
	}

	if !outcome.Passed {
		if restoreErr := restoreRepairCheckpoint(checkpoint); restoreErr != nil {
			return outcome, fmt.Errorf("restore repair checkpoint: %w", restoreErr)
		}
		outcome.Restored = true
		if announceRestore != nil {
			announceRestore(checkpointID)
		}
	}

	return outcome, nil
}

// repairEligibilityEvaluation (CAP-024) extends classifyAutopilotRepairFailure
// (cmd/autopilot_policy.go) with the two further inputs bounded recovery
// must consume alongside the failing check it already evaluates: unresolved
// blocker flags (readBlockerSnapshot, the one blocker-truth store this
// plan's Task 1 also writes into) and a recurring failure class. There is
// still exactly one eligibility decision -- Eligible/Pause come from
// classifyAutopilotRepairFailure alone, unchanged -- this function only
// enriches the recorded Reason so it names exactly which of the three
// inputs (unresolved flags, a recurring failure class, or the failing check
// itself) drove it, and so a phase carrying unresolved flags is never
// reported as though it had nothing left to repair.
//
// Recurring failure classes are read by consuming the existing REDIRECT
// signal emitMiddenThresholdRedirect already writes once a midden.json
// category crosses its three-unacknowledged-failure threshold
// (cmd/phase_end_signals.go) -- never by recounting that threshold a second
// time here (CAP-024: one recovery model, no second counting path).
func repairEligibilityEvaluation(failure autopilotRepairFailure, remainingBudget int) autopilotRepairEvaluation {
	evaluation := classifyAutopilotRepairFailure(failure, remainingBudget)

	blockers, _ := readBlockerSnapshot(store)
	hasFlags := blockers.Count > 0
	_, hasRecurring := recurringFailureClassSignal()

	switch {
	case hasFlags && hasRecurring:
		evaluation.Reason = fmt.Sprintf("%s (also driven by: %d unresolved flag(s) and a recurring failure class)", evaluation.Reason, blockers.Count)
	case hasFlags:
		evaluation.Reason = fmt.Sprintf("%s (also driven by: %d unresolved flag(s))", evaluation.Reason, blockers.Count)
	case hasRecurring:
		evaluation.Reason = fmt.Sprintf("%s (also driven by: a recurring failure class)", evaluation.Reason)
	}

	return evaluation
}

// recurringFailureClassSignal reports whether an active REDIRECT signal is
// already recorded -- the exact signal emitMiddenThresholdRedirect writes
// once a midden.json failure category crosses the three-unacknowledged
// threshold (cmd/phase_end_signals.go, middenAutoRedirectThreshold). This
// consumes that already-computed signal rather than re-scanning midden.json
// and recounting the threshold a second time.
func recurringFailureClassSignal() (colony.PheromoneSignal, bool) {
	if store == nil {
		return colony.PheromoneSignal{}, false
	}
	pf, err := loadPheromoneFileWithFallback(store)
	if err != nil {
		return colony.PheromoneSignal{}, false
	}
	for _, sig := range pf.Signals {
		if sig.Type == "REDIRECT" && sig.Active {
			return sig, true
		}
	}
	return colony.PheromoneSignal{}, false
}

// repairCheckpointIdentity is D-09's idempotency key (SYN-201-10): a
// deterministic string naming the failing verification a repair round is
// for -- phase and check, not the attempt that discovered it, because "the
// same failing verification" persists across re-runs (matching
// checkFixAttemptRecord's own one-per-phase-per-check-ever discipline,
// cmd/check_fix_attempt.go's maxAutomaticCheckFixAttempts).
func repairCheckpointIdentity(phase int, check string) string {
	normalized := strings.ToLower(strings.TrimSpace(check))
	normalized = strings.Join(strings.Fields(normalized), "-")
	return fmt.Sprintf("repair-checkpoint-%d-%s", phase, normalized)
}

// findAutopilotRepairReceiptByCheckpoint is the one read path for "does this
// checkpoint identity already have a receipt" -- the idempotent-replay
// lookup runBoundedRepairRound uses before ever taking a new checkpoint.
func findAutopilotRepairReceiptByCheckpoint(ledger autopilotRepairLedger, checkpointID string) (autopilotRepairReceipt, bool) {
	for _, receipt := range ledger.Receipts {
		if receipt.CheckpointID == checkpointID {
			return receipt, true
		}
	}
	return autopilotRepairReceipt{}, false
}

// repairCheckpoint is the saved snapshot a bounded repair round takes before
// its one repair wave runs, so a failed round can be restored exactly.
type repairCheckpoint struct {
	ID        string
	Root      string
	BackupDir string
	Paths     []string
	FullRoot  bool
	CreatedAt time.Time
}

// saveRepairCheckpoint copies exactly the given paths (files or directories,
// relative to root) into an isolated temp directory. An empty paths list
// checkpoints the entire root. A path that does not exist yet at checkpoint
// time is simply skipped -- restoring it later means removing whatever the
// repair wave created there, not restoring content that never existed.
func saveRepairCheckpoint(root, checkpointID string, paths []string) (repairCheckpoint, error) {
	backupDir, err := os.MkdirTemp("", "aether-repair-checkpoint-*")
	if err != nil {
		return repairCheckpoint{}, fmt.Errorf("create checkpoint dir: %w", err)
	}

	unique := uniqueSortedStrings(paths)
	if len(unique) == 0 {
		if err := backupCopyDir(root, backupDir); err != nil {
			return repairCheckpoint{}, fmt.Errorf("checkpoint root: %w", err)
		}
		return repairCheckpoint{ID: checkpointID, Root: root, BackupDir: backupDir, FullRoot: true, CreatedAt: time.Now().UTC()}, nil
	}

	for _, rel := range unique {
		src := filepath.Join(root, rel)
		dst := filepath.Join(backupDir, rel)
		info, statErr := os.Stat(src)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				continue
			}
			return repairCheckpoint{}, fmt.Errorf("stat checkpoint source %s: %w", rel, statErr)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return repairCheckpoint{}, fmt.Errorf("prepare checkpoint dir for %s: %w", rel, err)
		}
		if info.IsDir() {
			if err := backupCopyDir(src, dst); err != nil {
				return repairCheckpoint{}, fmt.Errorf("checkpoint dir %s: %w", rel, err)
			}
			continue
		}
		if err := backupCopyFile(src, dst); err != nil {
			return repairCheckpoint{}, fmt.Errorf("checkpoint file %s: %w", rel, err)
		}
	}
	return repairCheckpoint{ID: checkpointID, Root: root, BackupDir: backupDir, Paths: unique, CreatedAt: time.Now().UTC()}, nil
}

// restoreRepairCheckpoint puts the working tree back exactly as
// saveRepairCheckpoint found it -- never touching anything outside the
// checkpointed paths (or the whole root, for a FullRoot checkpoint).
func restoreRepairCheckpoint(checkpoint repairCheckpoint) error {
	if checkpoint.FullRoot {
		entries, err := os.ReadDir(checkpoint.Root)
		if err != nil {
			return fmt.Errorf("restore root: read current: %w", err)
		}
		for _, entry := range entries {
			if err := os.RemoveAll(filepath.Join(checkpoint.Root, entry.Name())); err != nil {
				return fmt.Errorf("restore root: clear %s: %w", entry.Name(), err)
			}
		}
		if err := backupCopyDir(checkpoint.BackupDir, checkpoint.Root); err != nil {
			return fmt.Errorf("restore root: %w", err)
		}
		return nil
	}

	for _, rel := range checkpoint.Paths {
		src := filepath.Join(checkpoint.BackupDir, rel)
		dst := filepath.Join(checkpoint.Root, rel)

		info, statErr := os.Stat(src)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				if removeErr := os.RemoveAll(dst); removeErr != nil {
					return fmt.Errorf("restore %s: remove: %w", rel, removeErr)
				}
				continue
			}
			return fmt.Errorf("restore %s: stat backup: %w", rel, statErr)
		}
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("restore %s: clear current: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("restore %s: prepare dir: %w", rel, err)
		}
		if info.IsDir() {
			if err := backupCopyDir(src, dst); err != nil {
				return fmt.Errorf("restore dir %s: %w", rel, err)
			}
			continue
		}
		if err := backupCopyFile(src, dst); err != nil {
			return fmt.Errorf("restore file %s: %w", rel, err)
		}
	}
	return nil
}

// repairCheckpointDirectoryDigest is a stable content digest over exactly
// the given paths (or the whole root when paths is empty) -- relative path
// plus file bytes, sorted -- used to prove a restored working tree is
// byte-identical to its state at checkpoint time.
func repairCheckpointDirectoryDigest(root string, paths []string) (string, error) {
	scan := uniqueSortedStrings(paths)
	if len(scan) == 0 {
		scan = []string{"."}
	}

	var files []string
	for _, rel := range scan {
		full := filepath.Join(root, rel)
		walkErr := filepath.WalkDir(full, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() {
				return nil
			}
			relPath, relErr := filepath.Rel(root, p)
			if relErr != nil {
				return relErr
			}
			files = append(files, relPath)
			return nil
		})
		if walkErr != nil && !os.IsNotExist(walkErr) {
			return "", walkErr
		}
	}
	sort.Strings(files)

	h := sha256.New()
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			return "", err
		}
		h.Write([]byte(f))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// emitRepairCheckpointSaved and emitRepairCheckpointRestored are D-10's flow
// announcements: rendered through the exact same visual-mode-gated flow
// every other continue narration uses (emitContinueVerificationStart's own
// pattern, cmd/codex_continue.go), never only in a log or a result envelope.
// Plain English throughout, no repository-invented vocabulary (CLAUDE.md).
func emitRepairCheckpointSaved(phase int, check string) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	writeVisualOutput(stdout, fmt.Sprintf(
		"Saving your project's current state for phase %d before trying one automatic fix for the %s check, so it can be put back exactly if the fix does not work.\n",
		phase, check,
	))
}

func emitRepairCheckpointRestored(phase int, check string) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	writeVisualOutput(stdout, fmt.Sprintf(
		"The automatic fix for the %s check did not work, so your project has been put back exactly to the state it was saved in for phase %d.\n",
		check, phase,
	))
}

// applyBoundedCheckFixRepair wraps applyAutomaticCheckFixAttempt (D-02's
// existing single bounded builder fix attempt, cmd/check_fix_attempt.go)
// with D-09's checkpoint/restore discipline and D-10's flow announcements,
// without altering the fix attempt's own eligibility or dispatch logic.
// Eligibility is checked read-only via planCheckFixAttempt before anything
// is saved, so a call with nothing to fix takes no checkpoint and announces
// nothing -- matching the pre-existing no-op behavior exactly.
//
// This is the direct check lane's production wiring (cmd/codex_continue.go).
func applyBoundedCheckFixRepair(ctx context.Context, root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, floor deterministicFloorResult, buildWatcher codexWatcherVerification, workerTimeout, verificationTimeout time.Duration, reviewerDispatched bool) (deterministicFloorResult, *checkFixAttemptRecord) {
	record, eligible := planCheckFixAttempt(state, phase, manifest, floor, reviewerDispatched)
	if !eligible {
		return floor, nil
	}

	scopePaths := repairScopePathsForCheckFix(record, manifest, phase)
	checkpoint, err := saveRepairCheckpoint(root, repairCheckpointIdentity(phase.ID, record.Check), scopePaths)
	if err != nil {
		// A checkpoint that cannot be saved must never silently block the
		// existing D-02 fix attempt -- fall back to the pre-201-09
		// behavior (no checkpoint, no restore) rather than refuse to try.
		return applyAutomaticCheckFixAttempt(ctx, root, state, phase, manifest, floor, buildWatcher, workerTimeout, verificationTimeout, reviewerDispatched)
	}
	defer os.RemoveAll(checkpoint.BackupDir)

	emitRepairCheckpointSaved(phase.ID, record.Check)

	// D-12: deliver this same-run attempt's own failure evidence (Task 1's
	// midden entry, written by recordDispatchWorkerOutcome before this check
	// ever ran) into the repair wave's brief -- before the repair wave
	// dispatches, so the second attempt does not repeat the first attempt's
	// mistake. plannedCheckFixBuilderDispatch's ContextCapsule already reads
	// resolveCodexWorkerContext(), which already surfaces active pheromone
	// signals; deliverFailureBornRepairSignal reuses that existing steering
	// channel rather than adding a new one.
	if _, attempt, ok := loadLatestBuildAttempt(phase.ID); ok {
		deliverFailureBornRepairSignal(phase.ID, record.Check, attempt.ID)
	}

	newFloor, fixed := applyAutomaticCheckFixAttempt(ctx, root, state, phase, manifest, floor, buildWatcher, workerTimeout, verificationTimeout, reviewerDispatched)

	if fixed != nil && fixed.Outcome == "still_failing" {
		if restoreErr := restoreRepairCheckpoint(checkpoint); restoreErr == nil {
			emitRepairCheckpointRestored(phase.ID, record.Check)
		}
	}
	return newFloor, fixed
}

// repairScopePathsForCheckFix derives the checkpoint scope for a D-02 check
// fix attempt from the same claimed-files data the failure index itself was
// built from (buildCheckFailureIndex, cmd/check_fix_attempt.go) -- the files
// the implicated task(s) already claimed, never a fresh guess.
func repairScopePathsForCheckFix(record checkFixAttemptRecord, manifest codexContinueManifest, phase colony.Phase) []string {
	claims := loadRawBuildClaimsForScope(manifest)
	sets := criterionClaimSets(claims)

	implicated := map[string]bool{}
	for taskID := range sets {
		if taskID == "" {
			continue
		}
		for _, id := range record.FailureIndex.ImplicatedTaskIDs {
			if id == taskID {
				for path := range sets[taskID] {
					implicated[path] = true
				}
			}
		}
	}
	if len(implicated) == 0 {
		return nil
	}
	paths := make([]string, 0, len(implicated))
	for path := range implicated {
		paths = append(paths, path)
	}
	return paths
}

// failureBornRepairSignal (D-12) reads the failure evidence Task 1 wrote
// (cmd/memory_feed.go's recordDispatchWorkerOutcome -> midden.json, tagged
// via middenAttemptTagPrefix) for the exact attempt driving this repair
// round, and returns the newest matching entry's own sanitised message --
// worker text that was already sanitised once, at storage time
// (sanitizedWorkerSentence), and is never re-sanitised into a different
// value here on replay. Returns "" when this attempt produced no failure
// record (nothing to steer around) or when store is unavailable.
func failureBornRepairSignal(attemptID string) string {
	attemptID = strings.TrimSpace(attemptID)
	if store == nil || attemptID == "" {
		return ""
	}
	mf, err := loadMiddenFile(store)
	if err != nil {
		return ""
	}
	var newest colony.MiddenEntry
	found := false
	for _, entry := range mf.Entries {
		if entry.Category != middenCategoryWorkerFailed {
			continue
		}
		if middenEntryAttemptID(entry) != attemptID {
			continue
		}
		if !found || entry.Timestamp > newest.Timestamp {
			newest = entry
			found = true
		}
	}
	if !found {
		return ""
	}
	return newest.Message
}

// deliverFailureBornRepairSignal (D-12) writes the same-run failure-born
// signal into the ONE existing steering-signal channel every dispatch
// already reads from -- active pheromone signals (writePheromoneSignal,
// read back by both resolveCodexWorkerContext, which
// plannedCheckFixBuilderDispatch's ContextCapsule already calls, and
// composeBuildManifestBrief's own "## Pheromone Signals" section) -- never
// a new section. Called once, right after the repair checkpoint is saved
// and before the repair wave dispatches (see applyBoundedCheckFixRepair),
// so delivery happens inside this same run rather than at the next phase
// boundary.
//
// Content is truncated (never silently dropped) to signalContentSafeLimit
// -- the same budget-with-visible-ellipsis discipline
// emitMiddenThresholdRedirect already applies to the near-identical
// "recurring failure" signal (cmd/phase_end_signals.go) -- so a forced trim
// reports the omission via the trailing "..." rather than a signal that
// silently vanishes. A write failure is warned to stderr and never blocks
// the repair round (writing failure evidence never fails a build or a
// check). Returns the delivered text ("" when there was nothing to
// deliver), so a caller can assert on exactly what reached the channel.
func deliverFailureBornRepairSignal(phase int, check, attemptID string) string {
	signal := failureBornRepairSignal(attemptID)
	if strings.TrimSpace(signal) == "" {
		return ""
	}
	text := truncateSignalContent(fmt.Sprintf(
		"Same-run repair signal for phase %d: the %s check just failed with %s -- do not repeat that same action for this repair.",
		phase, check, signal,
	), signalContentSafeLimit)
	if _, _, err := writePheromoneSignal("REDIRECT", text, "", "aether continue", "", "", 0, nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not deliver same-run repair signal for phase %d check %q: %v\n", phase, check, err)
		return ""
	}
	return text
}

// repairHandback is D-11's four-element failed-repair handback: a
// plain-language diagnosis of what is failing, what the repair attempted
// and why it did not take, the restored safe position, and exactly one
// concrete action for the owner. Named fields, not one prose blob, so each
// element can be checked independently.
type repairHandback struct {
	Diagnosis        string
	AttemptedAndWhy  string
	RestoredPosition string
	OwnerAction      LifecycleCloseoutRecommendedAction
}

// buildFailedRepairHandback assembles D-11's handback from a failed repair
// round's own outcome and the failing check's captured output -- never a
// summary string. The diagnosis reuses compactFailureExcerpts
// (cmd/check_fix_attempt.go, D-02's own bounded-index precedent) rather
// than a second truncation implementation. It renders through the same
// closeout ceremony every other outcome uses (colony.WorkOutcomeBlocker,
// plan 201-04), and its one owner action reuses recommendedActionForWorkOutcome
// (plan 201-07) rather than adding a second recommendation path.
func buildFailedRepairHandback(outcome repairRoundOutcome, failingCheck, failingOutput string, phaseNum int) (repairHandback, LifecycleCloseoutDetails, error) {
	excerpts, _ := compactFailureExcerpts(failingOutput, "")
	diagnosis := strings.TrimSpace(strings.Join(excerpts, " "))
	if diagnosis == "" {
		diagnosis = fmt.Sprintf("the %s check is still failing", failingCheck)
	}

	plannedAction := strings.TrimSpace(outcome.Receipt.PlannedAction)
	if plannedAction == "" {
		plannedAction = "an automatic fix"
	}
	attempted := fmt.Sprintf("Tried: %s. It did not fix the %s check -- verification failed again after the fix ran.", plannedAction, failingCheck)

	restored := fmt.Sprintf("Your project has been put back exactly to the state it was saved in, just before the automatic fix ran (checkpoint %s).", outcome.CheckpointID)

	_, attempt, _ := loadLatestBuildAttempt(phaseNum)
	if strings.TrimSpace(attempt.Error) == "" {
		attempt.Error = diagnosis
	}
	action, err := recommendedActionForWorkOutcome(colony.WorkOutcomeBlocker, attempt)
	if err != nil {
		return repairHandback{}, LifecycleCloseoutDetails{}, err
	}

	handback := repairHandback{
		Diagnosis: diagnosis, AttemptedAndWhy: attempted, RestoredPosition: restored, OwnerAction: action,
	}

	details := LifecycleCloseoutDetails{
		WorkOutcome: colony.WorkOutcomeBlocker,
		Summary:     diagnosis,
		Blockers: []colony.LifecycleIssue{
			{ID: fmt.Sprintf("%s-diagnosis", outcome.CheckpointID), Summary: diagnosis},
			{ID: fmt.Sprintf("%s-attempted", outcome.CheckpointID), Summary: attempted},
		},
		StandingInstructions: []string{restored},
	}
	return handback, details, nil
}
