package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// swarm_episode.go is the one durable, replay-safe record of what a single
// Swarm run tried and what happened (LIVE-05, CAP-047). It is additive to
// the existing durable result record (swarmResultRecord, cmd/swarm_strikes.go)
// -- strike evaluation still reads that record exclusively, and nothing here
// changes its shape or the path saveSwarmResultRecord writes it through.

// swarmEpisodeSchemaVersion guards loadSwarmEpisode against reading an
// episode shaped by a future, incompatible version of this file.
const swarmEpisodeSchemaVersion = 1

// swarmEpisodeStatusCompleted and swarmEpisodeStatusInterrupted are the only
// two status values an episode may carry. An interrupted episode is never
// presented as a completed one.
const (
	swarmEpisodeStatusCompleted   = "completed"
	swarmEpisodeStatusInterrupted = "interrupted"
)

// swarmEpisodeStageInvestigation/Fix/Verification name the stage an
// interrupted run stopped at -- the wave whose dispatch was attempted (or
// under way) when the run stopped.
const (
	swarmEpisodeStageInvestigation = "investigation"
	swarmEpisodeStageFix           = "fix"
	swarmEpisodeStageVerification  = "verification"
)

// swarmEpisodeRetentionClassStandard is the only retention class an ordinary
// Swarm episode carries today. It exists as a named field (not a bare bool)
// so a future class (e.g. one exempt from age-based cleanup) can be added
// without a schema change.
const swarmEpisodeRetentionClassStandard = "standard"

// swarmEpisodeLens names one of the four fixed investigation lenses as it
// appears on the episode: whether it reported in this run, and why not when
// it did not.
type swarmEpisodeLens struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Caste    string `json:"caste"`
	Reported bool   `json:"reported"`
	Reason   string `json:"reason,omitempty"`
}

// swarmEpisodeComparison mirrors swarmComparison's decision-relevant fields
// (cmd/swarm_lens.go) onto the durable record -- the shared causes,
// contradictions, ranked candidates, and the one selected repair with its
// reason.
type swarmEpisodeComparison struct {
	SharedCauses    []swarmSharedCause   `json:"shared_causes,omitempty"`
	Contradictions  []swarmContradiction `json:"contradictions,omitempty"`
	Ranked          []swarmRankedRepair  `json:"ranked_repairs,omitempty"`
	Selected        *swarmRankedRepair   `json:"selected_repair,omitempty"`
	SelectionReason string               `json:"selection_reason,omitempty"`
}

// swarmEpisodeCheckpoint records whether the D-05/LIVE-04 repair checkpoint
// was saved before the fix wave, and whether it was restored afterward.
type swarmEpisodeCheckpoint struct {
	Saved    bool `json:"saved"`
	Restored bool `json:"restored"`
}

// swarmEpisodeCostReference points at the ledger keys the spend authority
// already uses (cmd/spend_ledger.go, cmd/spawn_runs.go) rather than copying a
// token or currency figure onto the episode -- the episode must never become
// a second money record. SpawnRunID is the runtime-issued spawn-tree run
// identifier (agent.SpawnRun.ID via beginRuntimeSpawnRun) this Swarm run was
// recorded under; it is empty only when no store was initialized to begin a
// run at all.
type swarmEpisodeCostReference struct {
	SpawnRunID string `json:"spawn_run_id,omitempty"`
}

// swarmEpisodeRetentionMeta carries the episode's retention class. Age and
// eligibility are deliberately NOT stored here -- eligibility depends on
// every OTHER episode for the same target (is this the latest one? is it
// part of a live strike sequence?), so a value frozen at write time would go
// stale the moment a newer episode for the same target is written.
// planSwarmEpisodeRetention (cmd/swarm_episode.go, Task 2) computes age and
// eligibility fresh, every time, from the whole current episode set.
type swarmEpisodeRetentionMeta struct {
	Class string `json:"class"`
}

// swarmLearningProposal is Task 3's scoped focus/avoid-this offer, attached
// to the episode as a proposal only -- see proposeSwarmLearningFromEpisode.
type swarmLearningProposal struct {
	Kind      string   `json:"kind"`
	Scope     string   `json:"scope"`
	Text      string   `json:"text"`
	Lenses    []string `json:"lenses,omitempty"`
	EpisodeID string   `json:"episode_id"`
}

// swarmEpisodeRecord is the one durable, replay-safe record of a single
// Swarm run: its lenses, hypotheses, comparison, checkpoint, verification
// outcome, strike standing, and a reference to its reported cost. It is
// bound to the existing swarmResultRecord (cmd/swarm_strikes.go) by sharing
// the same SwarmID, never replacing it.
type swarmEpisodeRecord struct {
	SchemaVersion     int    `json:"schema_version"`
	SwarmID           string `json:"swarm_id"`
	Target            string `json:"target"`
	TargetFingerprint string `json:"target_fingerprint"`
	Status            string `json:"status"`
	InterruptedStage  string `json:"interrupted_stage,omitempty"`
	StartedAt         string `json:"started_at"`
	EndedAt           string `json:"ended_at"`

	Lenses     []swarmEpisodeLens     `json:"lenses"`
	Hypotheses []swarmHypothesis      `json:"hypotheses,omitempty"`
	Comparison swarmEpisodeComparison `json:"comparison"`

	Checkpoint swarmEpisodeCheckpoint `json:"checkpoint"`

	// VerificationStatus is the verification wave's own outcome
	// (summarizeSwarmOutcome(watcherRuns)'s status), or "not_run" when no
	// repair was selected and no fix/verification wave ever dispatched.
	VerificationStatus string `json:"verification_status"`

	StrikeStanding swarmStrikeHistory `json:"strike_standing"`

	Cost swarmEpisodeCostReference `json:"cost"`

	Retention swarmEpisodeRetentionMeta `json:"retention"`

	LearningProposal *swarmLearningProposal `json:"learning_proposal,omitempty"`
}

// swarmEpisodeBuildParams is the input to buildSwarmEpisodeRecord -- named
// fields rather than a long positional argument list, since runSwarmDestroy
// has several distinct call sites (the honest no-evidence completion, the
// restore-failure completion, the ordinary success completion, and each of
// the three interrupted branches) that each fill a different subset.
type swarmEpisodeBuildParams struct {
	SwarmID            string
	Target             string
	Status             string
	InterruptedStage   string
	StartedAt          time.Time
	EndedAt            time.Time
	Comparison         swarmComparison
	CheckpointSaved    bool
	CheckpointRestored bool
	VerificationStatus string
	StrikeStanding     swarmStrikeHistory
	SpawnRunID         string
}

// swarmEpisodePath is the one path an episode is ever written to or read
// from, mirroring the existing swarms/<id>/result.json convention
// (cmd/swarm_strikes.go).
func swarmEpisodePath(swarmID string) string {
	return filepath.ToSlash(filepath.Join("swarms", strings.TrimSpace(swarmID), "episode.json"))
}

// buildSwarmEpisodeLenses reports, for each of the four fixed investigation
// lenses (cmd/swarm_lens.go), whether it produced a hypothesis in this run
// and why not when it did not -- reusing findHypothesisForLens and
// missingReasonForLens rather than re-deriving the same answer twice.
func buildSwarmEpisodeLenses(comparison swarmComparison) []swarmEpisodeLens {
	out := make([]swarmEpisodeLens, 0, len(swarmLenses))
	for _, lens := range swarmLenses {
		entry := swarmEpisodeLens{ID: lens.ID, Label: lens.Label, Caste: lens.Caste}
		if _, ok := findHypothesisForLens(comparison.Hypotheses, lens.ID); ok {
			entry.Reported = true
		} else {
			entry.Reason = missingReasonForLens(comparison.MissingLenses, lens.ID)
		}
		out = append(out, entry)
	}
	return out
}

func swarmEpisodeComparisonFrom(c swarmComparison) swarmEpisodeComparison {
	return swarmEpisodeComparison{
		SharedCauses:    append([]swarmSharedCause{}, c.SharedCauses...),
		Contradictions:  append([]swarmContradiction{}, c.Contradictions...),
		Ranked:          append([]swarmRankedRepair{}, c.Ranked...),
		Selected:        c.Selected,
		SelectionReason: c.SelectionReason,
	}
}

// buildSwarmEpisodeRecord assembles the durable record from the run's own
// already-computed values. It performs no I/O and no validation of its own
// -- persistSwarmEpisode owns both, so a caller can build a record purely in
// memory (e.g. to attach a Task 3 learning proposal) before the one write.
func buildSwarmEpisodeRecord(p swarmEpisodeBuildParams) swarmEpisodeRecord {
	target := strings.TrimSpace(p.Target)
	return swarmEpisodeRecord{
		SchemaVersion:      swarmEpisodeSchemaVersion,
		SwarmID:            strings.TrimSpace(p.SwarmID),
		Target:             target,
		TargetFingerprint:  swarmTargetFingerprint(target),
		Status:             p.Status,
		InterruptedStage:   p.InterruptedStage,
		StartedAt:          p.StartedAt.UTC().Format(time.RFC3339Nano),
		EndedAt:            p.EndedAt.UTC().Format(time.RFC3339Nano),
		Lenses:             buildSwarmEpisodeLenses(p.Comparison),
		Hypotheses:         append([]swarmHypothesis{}, p.Comparison.Hypotheses...),
		Comparison:         swarmEpisodeComparisonFrom(p.Comparison),
		Checkpoint:         swarmEpisodeCheckpoint{Saved: p.CheckpointSaved, Restored: p.CheckpointRestored},
		VerificationStatus: p.VerificationStatus,
		StrikeStanding:     p.StrikeStanding,
		Cost:               swarmEpisodeCostReference{SpawnRunID: strings.TrimSpace(p.SpawnRunID)},
		Retention:          swarmEpisodeRetentionMeta{Class: swarmEpisodeRetentionClassStandard},
	}
}

// persistSwarmEpisode validates the episode's identity through the same
// durable-identifier validation the external finalizer already performs
// (validateDurableSwarmID, cmd/swarm_issuance.go) before any write -- a
// caller-supplied identifier that is not the runtime-issued convention is
// refused, naming it, and nothing is written. This is the ONLY function that
// writes swarms/<id>/episode.json.
func persistSwarmEpisode(s *storage.Store, record swarmEpisodeRecord) error {
	if s == nil {
		return fmt.Errorf("persist swarm episode: no store initialized")
	}
	record.SwarmID = strings.TrimSpace(record.SwarmID)
	if _, err := validateDurableSwarmID(s, record.SwarmID); err != nil {
		return fmt.Errorf("persist swarm episode: %w", err)
	}
	record.Target = strings.TrimSpace(record.Target)
	if record.Target == "" {
		return fmt.Errorf("persist swarm episode: target is required")
	}
	record.TargetFingerprint = swarmTargetFingerprint(record.Target)
	switch record.Status {
	case swarmEpisodeStatusCompleted:
		record.InterruptedStage = ""
	case swarmEpisodeStatusInterrupted:
		if strings.TrimSpace(record.InterruptedStage) == "" {
			return fmt.Errorf("persist swarm episode: an interrupted episode must name the stage it stopped at")
		}
	default:
		return fmt.Errorf("persist swarm episode: status must be %q or %q, got %q", swarmEpisodeStatusCompleted, swarmEpisodeStatusInterrupted, record.Status)
	}
	if strings.TrimSpace(record.Retention.Class) == "" {
		record.Retention.Class = swarmEpisodeRetentionClassStandard
	}
	record.SchemaVersion = swarmEpisodeSchemaVersion
	if err := s.SaveJSON(swarmEpisodePath(record.SwarmID), record); err != nil {
		return fmt.Errorf("persist swarm episode: %w", err)
	}
	return nil
}

// loadSwarmEpisode is the one read path for an episode. A missing, corrupt,
// or schema-mismatched file returns ok == false rather than a partially
// populated record.
func loadSwarmEpisode(s *storage.Store, swarmID string) (swarmEpisodeRecord, bool) {
	swarmID = strings.TrimSpace(swarmID)
	if s == nil || swarmID == "" {
		return swarmEpisodeRecord{}, false
	}
	var record swarmEpisodeRecord
	if err := s.LoadJSON(swarmEpisodePath(swarmID), &record); err != nil {
		return swarmEpisodeRecord{}, false
	}
	if record.SchemaVersion != swarmEpisodeSchemaVersion {
		return swarmEpisodeRecord{}, false
	}
	return record, true
}

// swarmEpisodeFileDigest hashes the raw on-disk episode.json bytes for
// swarmID. Used both by Task 2's retention preview (so a later removal
// request can be checked against exactly what was previewed) and by tests
// proving a re-read never changes anything.
func swarmEpisodeFileDigest(s *storage.Store, swarmID string) (string, error) {
	swarmDir, err := validateDurableSwarmID(s, swarmID)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(swarmDir, "episode.json"))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// spawnRunIDFrom reads the spawn-tree run identifier off a runtimeSpawnRun
// handle -- the reference stored on the episode's Cost field. A nil handle
// (no store initialized) yields the empty string.
func spawnRunIDFrom(handle *runtimeSpawnRun) string {
	if handle == nil {
		return ""
	}
	return strings.TrimSpace(handle.Run.ID)
}

// persistInterruptedSwarmEpisode is the one call site for the "the run
// stopped after any of the three waves" case: it re-derives the current
// strike standing (nothing was written to result.json on this path, so this
// reflects whatever strike history already existed), builds the episode with
// status interrupted, and persists it. Errors are logged, not returned --
// mirroring the existing checkpoint-save fallback in runSwarmDestroy (warn
// and proceed, never block the caller's own error return with a second
// failure from bookkeeping).
func persistInterruptedSwarmEpisode(swarmID, target, stage string, startedAt time.Time, comparison swarmComparison, checkpointSaved bool, spawnRunID string) {
	strikeStanding, _ := evaluateSwarmStrikeHistory(store, target)
	record := buildSwarmEpisodeRecord(swarmEpisodeBuildParams{
		SwarmID:            swarmID,
		Target:             target,
		Status:             swarmEpisodeStatusInterrupted,
		InterruptedStage:   stage,
		StartedAt:          startedAt,
		EndedAt:            time.Now().UTC(),
		Comparison:         comparison,
		CheckpointSaved:    checkpointSaved,
		CheckpointRestored: false,
		VerificationStatus: "not_run",
		StrikeStanding:     strikeStanding,
		SpawnRunID:         spawnRunID,
	})
	if err := persistSwarmEpisode(store, record); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not persist interrupted swarm episode for %q: %v\n", target, err)
	}
}

// swarmMidRunInterruptFunc is a test seam (mirroring newSwarmWorkerInvoker /
// swarmRestoreRepairCheckpointFunc, cmd/swarm_cmd.go and
// cmd/swarm_repair_checkpoint.go) called once, in runSwarmDestroy, right
// after the investigation wave completes and before the fix wave dispatches.
// Production always leaves this a no-op. A test sets it to cancel the run's
// own context, proving the interrupted-episode path against a genuine
// ctx.Err() surfaced by the real public Swarm path, rather than a
// timing-dependent sleep racing a timeout.
var swarmMidRunInterruptFunc = func(cancel func()) {}

// --- Task 2: retention and cleanup that act on a named episode (CAP-048) ---
//
// planSwarmEpisodeRetention/removeSwarmEpisodes never sweep by directory
// prefix or content guess. A removal names an exact episode identifier
// together with the digest it was previewed at (the manifest-and-digest
// discipline this repository already applies to maintenance deletion, e.g.
// safeIdentifierSegment/validateDurableSwarmID above); a stale or mismatched
// digest refuses the whole operation before anything is deleted.

// swarmEpisodeRetentionEntry is one episode's retention verdict: whether it
// is eligible for removal right now, and why. Eligibility is recomputed
// fresh on every call from the CURRENT full episode set -- it is
// deliberately never cached on the episode record itself (see
// swarmEpisodeRetentionMeta's doc comment), so a preview always reflects
// what is true right now, not what was true when the episode was written.
type swarmEpisodeRetentionEntry struct {
	SwarmID    string `json:"swarm_id"`
	Target     string `json:"target"`
	Class      string `json:"class"`
	AgeSeconds int64  `json:"age_seconds"`
	Eligible   bool   `json:"eligible"`
	Reason     string `json:"reason"`
	Digest     string `json:"digest"`
}

// listSwarmEpisodes scans swarms/*/episode.json and returns every episode
// that loads cleanly, mirroring the directory-scan discipline
// evaluateSwarmStrikeHistoryAgainstPlan already uses for result.json. A
// missing swarms directory yields an empty, non-error result.
func listSwarmEpisodes(s *storage.Store) ([]swarmEpisodeRecord, error) {
	if s == nil {
		return nil, fmt.Errorf("list swarm episodes: no store initialized")
	}
	swarmsDir := filepath.Join(s.BasePath(), "swarms")
	entries, err := os.ReadDir(swarmsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list swarm episodes: read swarms: %w", err)
	}
	var episodes []swarmEpisodeRecord
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		record, ok := loadSwarmEpisode(s, entry.Name())
		if !ok {
			continue
		}
		episodes = append(episodes, record)
	}
	return episodes, nil
}

// planSwarmEpisodeRetention is a pure read: it computes and returns the
// eligible-for-removal set with a stated reason per episode, writing,
// creating, or initializing nothing. Two rules make an episode ineligible,
// checked in order:
//
//  1. It is the most recent (by EndedAt) episode recorded for its target --
//     the one entry point an owner or a later Swarm run would look at first.
//  2. It belongs to a target's currently unresolved strike sequence --
//     re-derived from evaluateSwarmStrikeHistory (cmd/swarm_strikes.go), the
//     exact same evidence the three-strike escalation guard reads, so
//     retention and strike counting can never disagree about which episodes
//     are "still live".
//
// Every other episode is eligible, with a plain reason stating so.
func planSwarmEpisodeRetention(s *storage.Store, now time.Time) ([]swarmEpisodeRetentionEntry, error) {
	episodes, err := listSwarmEpisodes(s)
	if err != nil {
		return nil, err
	}

	latestByTarget := map[string]string{}
	latestTimeByTarget := map[string]time.Time{}
	for _, ep := range episodes {
		endedAt, perr := time.Parse(time.RFC3339Nano, ep.EndedAt)
		if perr != nil {
			continue
		}
		fp := ep.TargetFingerprint
		if cur, ok := latestTimeByTarget[fp]; !ok || endedAt.After(cur) {
			latestTimeByTarget[fp] = endedAt
			latestByTarget[fp] = ep.SwarmID
		}
	}

	strikeLiveSwarmIDs := map[string]bool{}
	seenTargets := map[string]bool{}
	for _, ep := range episodes {
		if seenTargets[ep.TargetFingerprint] {
			continue
		}
		seenTargets[ep.TargetFingerprint] = true
		history, herr := evaluateSwarmStrikeHistory(s, ep.Target)
		if herr != nil {
			continue
		}
		for _, e := range history.Evidence {
			strikeLiveSwarmIDs[e.SwarmID] = true
		}
	}

	out := make([]swarmEpisodeRetentionEntry, 0, len(episodes))
	for _, ep := range episodes {
		entry := swarmEpisodeRetentionEntry{
			SwarmID:  ep.SwarmID,
			Target:   ep.Target,
			Class:    ep.Retention.Class,
			Eligible: true,
			Reason:   "not the latest episode for its target and not part of an unresolved strike sequence",
		}
		if endedAt, perr := time.Parse(time.RFC3339Nano, ep.EndedAt); perr == nil {
			age := now.Sub(endedAt)
			if age > 0 {
				entry.AgeSeconds = int64(age.Seconds())
			}
		}
		switch {
		case latestByTarget[ep.TargetFingerprint] == ep.SwarmID:
			entry.Eligible = false
			entry.Reason = "this is the most recent episode recorded for its target"
		case strikeLiveSwarmIDs[ep.SwarmID]:
			entry.Eligible = false
			entry.Reason = "this episode is part of an unresolved strike sequence for its target"
		}
		if digest, derr := swarmEpisodeFileDigest(s, ep.SwarmID); derr == nil {
			entry.Digest = digest
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SwarmID < out[j].SwarmID })
	return out, nil
}

// swarmEpisodeRemovalRequest is one caller-named episode to remove: its
// exact identifier and the digest it was previewed at (from
// planSwarmEpisodeRetention's Digest field). A prefix, a glob, or an
// identifier that does not match the runtime-issued swarm-id convention is
// refused by validateDurableSwarmID before anything is inspected further.
type swarmEpisodeRemovalRequest struct {
	SwarmID string
	Digest  string
}

// swarmEpisodeRemovalResult names exactly what was removed.
type swarmEpisodeRemovalResult struct {
	Removed []string `json:"removed"`
}

// removeSwarmEpisodes validates every request -- identifier AND digest --
// before deleting anything. A digest that no longer matches the currently
// stored episode is refused, naming the mismatched episode, and the WHOLE
// batch is refused with it: a removal request is a promise made against a
// specific preview, and honoring part of a batch whose preview has gone
// stale would remove something the caller never actually confirmed.
// Removing an episode deletes only its episode.json -- result.json (the
// strike-history source of truth) and the swarm's issuance record are never
// touched, so removal can never alter strike history.
func removeSwarmEpisodes(s *storage.Store, requests []swarmEpisodeRemovalRequest) (swarmEpisodeRemovalResult, error) {
	if s == nil {
		return swarmEpisodeRemovalResult{}, fmt.Errorf("remove swarm episodes: no store initialized")
	}
	if len(requests) == 0 {
		return swarmEpisodeRemovalResult{}, nil
	}

	type validatedRemoval struct {
		path    string
		swarmID string
	}
	toDelete := make([]validatedRemoval, 0, len(requests))
	for _, req := range requests {
		swarmID := strings.TrimSpace(req.SwarmID)
		swarmDir, err := validateDurableSwarmID(s, swarmID)
		if err != nil {
			return swarmEpisodeRemovalResult{}, fmt.Errorf("remove swarm episodes: invalid identifier %q: %w", req.SwarmID, err)
		}
		episodePath := filepath.Join(swarmDir, "episode.json")
		info, statErr := os.Lstat(episodePath)
		if statErr != nil {
			return swarmEpisodeRemovalResult{}, fmt.Errorf("remove swarm episodes: episode %q was not found: %w", swarmID, statErr)
		}
		if !info.Mode().IsRegular() {
			return swarmEpisodeRemovalResult{}, fmt.Errorf("remove swarm episodes: episode %q is not a regular file", swarmID)
		}
		currentDigest, derr := swarmEpisodeFileDigest(s, swarmID)
		if derr != nil {
			return swarmEpisodeRemovalResult{}, fmt.Errorf("remove swarm episodes: read current digest for %q: %w", swarmID, derr)
		}
		requestedDigest := strings.TrimSpace(req.Digest)
		if requestedDigest == "" || currentDigest != requestedDigest {
			return swarmEpisodeRemovalResult{}, fmt.Errorf(
				"remove swarm episodes: episode %q has changed since it was previewed (digest mismatch) -- refusing the whole removal", swarmID,
			)
		}
		toDelete = append(toDelete, validatedRemoval{path: episodePath, swarmID: swarmID})
	}

	removed := make([]string, 0, len(toDelete))
	for _, v := range toDelete {
		if err := os.Remove(v.path); err != nil {
			return swarmEpisodeRemovalResult{Removed: removed}, fmt.Errorf("remove swarm episodes: delete %q: %w", v.swarmID, err)
		}
		removed = append(removed, v.swarmID)
	}
	return swarmEpisodeRemovalResult{Removed: removed}, nil
}

// --- Task 3: propose a scoped note from successful evidence, without
// claiming it worked (CAP-045) ---

// swarmLearningProposalKindFocus and swarmLearningProposalKindAvoid are the
// only two proposal kinds proposeSwarmLearningFromEpisode ever produces:
// naming an area the corroborated hypotheses agreed on, or naming the
// pattern a non-selected candidate shared, whichever the episode's own
// comparison supports.
const (
	swarmLearningProposalKindFocus = "focus"
	swarmLearningProposalKindAvoid = "avoid"
)

// proposeSwarmLearningFromEpisode derives at most one proposed note from an
// episode whose repair passed verification, was never rolled back, and had
// usable evidence behind it. Every other case -- a failed repair, a
// restored (rolled-back) repair, or no selected repair at all -- proposes
// nothing, because there is nothing corroborated to offer.
//
// When the investigation's hypotheses shared an agreed cause
// (Comparison.SharedCauses), the proposal is a focus note naming that area.
// Otherwise, when the ranking produced a runner-up candidate that was
// considered but not selected, the proposal is an avoid-this note naming
// the pattern that runner-up shared. Absent both -- a single, unshared
// hypothesis with no runner-up -- nothing is proposed; a lone claim is not
// corroboration.
//
// The proposal is attached to the episode as a PROPOSAL only: nothing here
// writes an active pheromone signal, and no field on swarmLearningProposal
// asserts the note was applied, used, or effective -- measuring that is the
// next phase's boundary (202-CLASSIC-SYNTHESIS.md).
func proposeSwarmLearningFromEpisode(episode swarmEpisodeRecord) *swarmLearningProposal {
	if episode.Status != swarmEpisodeStatusCompleted {
		return nil
	}
	if episode.VerificationStatus != "completed" {
		return nil
	}
	if episode.Checkpoint.Restored {
		// The repair was rolled back -- verification did not actually hold.
		return nil
	}
	if episode.Comparison.Selected == nil {
		return nil
	}
	if len(episode.Hypotheses) == 0 {
		return nil
	}

	var kind, scope string
	var lenses []string
	var text string
	switch {
	case len(episode.Comparison.SharedCauses) > 0:
		cause := episode.Comparison.SharedCauses[0]
		kind = swarmLearningProposalKindFocus
		scope = cause.Cause
		lenses = append([]string{}, cause.Lenses...)
		text = fmt.Sprintf(
			"Swarm run %s corroborated and successfully repaired an issue related to: %s. Consider giving this area extra attention.",
			episode.SwarmID, scope,
		)
	case len(episode.Comparison.Ranked) > 1:
		runnerUp := episode.Comparison.Ranked[1]
		kind = swarmLearningProposalKindAvoid
		scope = runnerUp.Repair
		lenses = append([]string{}, runnerUp.Lenses...)
		text = fmt.Sprintf(
			"Swarm run %s applied a different repair than %q, which was considered but not selected. Consider avoiding that pattern here.",
			episode.SwarmID, scope,
		)
	default:
		// A single, unshared hypothesis with no runner-up: nothing to offer.
		return nil
	}

	sanitized, err := colony.SanitizeSignalContent(text)
	if err != nil {
		// Worker-authored text that fails sanitization is dropped rather
		// than stored unsanitized or with a fallback wording that could
		// itself carry the rejected content.
		return nil
	}

	return &swarmLearningProposal{
		Kind:      kind,
		Scope:     scope,
		Text:      sanitized,
		Lenses:    lenses,
		EpisodeID: episode.SwarmID,
	}
}
