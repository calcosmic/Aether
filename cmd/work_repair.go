package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	evaluation := classifyAutopilotRepairFailure(failure, ledger.Remaining)
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
