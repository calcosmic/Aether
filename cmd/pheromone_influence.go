package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Actor kinds (BIO-08, plan 203-11)
// ---------------------------------------------------------------------------

// Actor kinds a pheromoneInfluenceEntry may declare. This is a closed set --
// an entry naming any other value is refused by name (appendInfluenceHistory).
// accept/edit/reject/revoke/appeal change what the owner meant and are
// owner-only (pheromoneInfluenceActorAllowed); reinforce/defer/expire change
// only how long or how strongly an existing meaning applies and may also be
// performed by the runtime or a learning pass.
const (
	pheromoneActorOwner    = "owner"
	pheromoneActorRuntime  = "runtime"
	pheromoneActorLearning = "learning"
)

// pheromoneActors returns the three declared actor kinds a history entry may
// record. cmd's TestInfluenceHistoryActor derives its declared-set assertions
// from this accessor rather than a hand-typed list.
func pheromoneActors() []string {
	return []string{pheromoneActorOwner, pheromoneActorRuntime, pheromoneActorLearning}
}

func pheromoneActorDeclared(kind string) bool {
	for _, a := range pheromoneActors() {
		if a == kind {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Influence actions (BIO-08, plan 203-11)
// ---------------------------------------------------------------------------

// The eight declared influence action names. accepted/edited/rejected are
// the three that already landed with the approval surface (plan 203-08,
// cmd/pheromone_approval.go's colony.PendingAction* constants); the
// remaining five are built by this plan.
const (
	pheromoneActionAccepted   = colony.PendingActionAccepted
	pheromoneActionEdited     = colony.PendingActionEdited
	pheromoneActionRejected   = colony.PendingActionRejected
	pheromoneActionReinforced = "reinforced"
	pheromoneActionDeferred   = "deferred"
	pheromoneActionExpired    = "expired"
	pheromoneActionRevoked    = "revoked"
	pheromoneActionAppealed   = "appealed"
	// pheromoneActionWeakened is plan 203-13's own addition (BIO-08/CEC-07):
	// the symmetric opposite of reinforced. A real owner-facing action
	// (weakenNote, --weaken on pheromoneDisplayCmd) AND the action name
	// cmd/pheromone_outcome.go's tuneNoteStrengthFromOutcomes uses for every
	// learning-actor strength decrease (neutral or harmful outcome) --
	// direction is named by the action, magnitude is named in the history
	// entry's own before/after text, which legitimately differs between a
	// manual full weaken and an automatic partial step.
	pheromoneActionWeakened = "weakened"
	// pheromoneActionPinned/-Unpinned are plan 203-13's own addition: an
	// owner action recorded in the history with actor kind owner, after
	// which tuneNoteStrengthFromOutcomes never touches the note's strength
	// again in either direction until unpinned. The runtime cannot pin or
	// unpin -- pheromoneInfluenceActorAllowed below enforces owner-only,
	// exactly like revoked/appealed.
	pheromoneActionPinned   = "pinned"
	pheromoneActionUnpinned = "unpinned"
)

// pheromoneInfluenceActions is the closed set of all eleven declared action
// names a note's history may record -- the three that already landed with
// the approval surface, the five plan 203-11 built, and the three plan
// 203-13 adds (weakened, pinned, unpinned).
// TestEveryDeclaredActionIsReachable derives its inventory from
// pheromoneInfluenceActionNames() rather than a hand-typed list, so adding a
// new name here without a matching entry in pheromoneInfluenceActionSurface
// fails that test by name.
var pheromoneInfluenceActions = []string{
	pheromoneActionAccepted,
	pheromoneActionEdited,
	pheromoneActionRejected,
	pheromoneActionReinforced,
	pheromoneActionDeferred,
	pheromoneActionExpired,
	pheromoneActionRevoked,
	pheromoneActionAppealed,
	pheromoneActionWeakened,
	pheromoneActionPinned,
	pheromoneActionUnpinned,
}

// pheromoneInfluenceActionNames returns a copy of the closed set of declared
// influence action names.
func pheromoneInfluenceActionNames() []string {
	return append([]string{}, pheromoneInfluenceActions...)
}

func pheromoneInfluenceActionDeclared(action string) bool {
	for _, a := range pheromoneInfluenceActions {
		if a == action {
			return true
		}
	}
	return false
}

// pheromoneInfluenceActionSurface names, for each declared action, the
// implementation function that performs it and the owner-facing command and
// flag a person can actually run to reach it. TestEveryDeclaredActionIsReachable
// walks pheromoneInfluenceActionNames() and fails by name if an entry is
// missing here, or if the named function/flag does not actually exist.
type pheromoneInfluenceActionSurfaceEntry struct {
	Implementation string
	Command        string
	Flag           string
}

var pheromoneInfluenceActionSurface = map[string]pheromoneInfluenceActionSurfaceEntry{
	pheromoneActionAccepted:   {Implementation: "approvePendingNote", Command: "suggestApproveCmd", Flag: "approve"},
	pheromoneActionEdited:     {Implementation: "editPendingNote", Command: "suggestApproveCmd", Flag: "edit"},
	pheromoneActionRejected:   {Implementation: "rejectPendingNote", Command: "suggestApproveCmd", Flag: "dismiss"},
	pheromoneActionReinforced: {Implementation: "reinforceNote", Command: "pheromoneDisplayCmd", Flag: "reinforce"},
	pheromoneActionDeferred:   {Implementation: "deferNote", Command: "pheromoneDisplayCmd", Flag: "defer"},
	pheromoneActionExpired:    {Implementation: "expireNote", Command: "pheromoneDisplayCmd", Flag: "expire"},
	pheromoneActionRevoked:    {Implementation: "revokeNote", Command: "pheromoneDisplayCmd", Flag: "revoke"},
	pheromoneActionAppealed:   {Implementation: "appealNote", Command: "pheromoneDisplayCmd", Flag: "appeal"},
	pheromoneActionWeakened:   {Implementation: "weakenNote", Command: "pheromoneDisplayCmd", Flag: "weaken"},
	pheromoneActionPinned:     {Implementation: "pinNote", Command: "pheromoneDisplayCmd", Flag: "pin"},
	pheromoneActionUnpinned:   {Implementation: "unpinNote", Command: "pheromoneDisplayCmd", Flag: "unpin"},
}

// ---------------------------------------------------------------------------
// Append-only history (Task 1)
// ---------------------------------------------------------------------------

// pheromoneInfluenceHistoryRetentionDeadNotes bounds how many EXTINGUISHED
// notes' full histories are kept once a note no longer exists in either
// pheromones.json or COLONY_STATE.json's pending-suggestion queue. Mirrors
// pkg/agent/spawn_tree.go's trimSpawnRuns bounded-history convention -- a
// LIVE note's history is never pruned, regardless of size, no matter how
// many dead notes are waiting to be trimmed.
const pheromoneInfluenceHistoryRetentionDeadNotes = 200

// pheromoneInfluenceEntry is one permanent record of an action taken on a
// note: what changed (Before/After), who did it (ActorKind/ActorName), when,
// and why. Before/After are human-readable summaries, not full snapshots, so
// an owner reading the history can see what a change actually did rather
// than only that a change happened.
type pheromoneInfluenceEntry struct {
	NoteID    string `json:"note_id"`
	Action    string `json:"action"`
	ActorKind string `json:"actor_kind"`
	ActorName string `json:"actor_name,omitempty"`
	Timestamp string `json:"timestamp"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// pheromoneInfluenceWriteCount is a test seam counting every store
// persistence performed by appendInfluenceHistory and
// pheromoneInfluenceSaveSignals. A --dry-run call must return before
// incrementing this counter -- TestInfluenceActions asserts the counter is
// unchanged across a dry-run call, proving an inspection never mutates state
// (CLAUDE.md's Definition of Done corollary), mirroring
// cmd/pheromone_approval.go's pendingNoteWriteCount seam. Production code
// never reads this value.
var pheromoneInfluenceWriteCount int

// pheromoneInfluenceHistoryFile is the top-level shape of
// pheromones-history.json: one append-only slice of entries per note
// identifier. The map value is only ever appended to -- never indexed for
// assignment, never truncated for an individual entry. A whole note's entry
// (the map key) is removed only by pruneExtinguishedInfluenceHistories, and
// only once the note itself no longer exists.
type pheromoneInfluenceHistoryFile struct {
	Notes map[string][]pheromoneInfluenceEntry `json:"notes"`
}

// pheromoneNoteExists reports whether noteID names a live note: either a
// stored colony.PheromoneSignal in pheromones.json, or a queued
// colony.PendingSuggestion in COLONY_STATE.json's pending-suggestion queue.
// appendInfluenceHistory refuses to record an entry for any other identifier.
func pheromoneNoteExists(noteID string) bool {
	if store == nil || noteID == "" {
		return false
	}
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err == nil {
		for _, sig := range pf.Signals {
			if sig.ID == noteID {
				return true
			}
		}
	}
	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err == nil && cs.PendingSuggestions != nil {
		for _, item := range *cs.PendingSuggestions {
			if item.ID == noteID {
				return true
			}
		}
	}
	return false
}

// pruneExtinguishedInfluenceHistories drops the oldest extinguished notes'
// entire histories once more than pheromoneInfluenceHistoryRetentionDeadNotes
// of them exist -- "oldest" measured by each dead note's own last recorded
// entry timestamp. A note that still exists (per pheromoneNoteExists) is
// never a pruning candidate, however large its history has grown.
func pruneExtinguishedInfluenceHistories(notes map[string][]pheromoneInfluenceEntry) map[string][]pheromoneInfluenceEntry {
	type deadNote struct {
		id       string
		lastSeen string
	}
	var dead []deadNote
	for id, entries := range notes {
		if pheromoneNoteExists(id) {
			continue
		}
		last := ""
		if len(entries) > 0 {
			last = entries[len(entries)-1].Timestamp
		}
		dead = append(dead, deadNote{id: id, lastSeen: last})
	}
	if len(dead) <= pheromoneInfluenceHistoryRetentionDeadNotes {
		return notes
	}
	sort.Slice(dead, func(i, j int) bool { return dead[i].lastSeen < dead[j].lastSeen })
	toDrop := len(dead) - pheromoneInfluenceHistoryRetentionDeadNotes
	for i := 0; i < toDrop; i++ {
		delete(notes, dead[i].id)
	}
	return notes
}

// appendInfluenceHistory is the ONE exported mutation on
// pheromones-history.json in this package -- it appends a new entry and
// nothing else. No other function in cmd/ writes this file
// (TestInfluenceHistoryAppendOnly), and this function itself never indexes
// into an existing note's entry slice for assignment or removal -- it only
// ever appends (via append()) and, through
// pruneExtinguishedInfluenceHistories, removes a DEAD note's history in
// full, never an individual entry of a live one.
//
// Refuses by name (writing nothing) when: the note identifier is empty, the
// action is not one of the eight declared names, the actor kind is not one
// of the three declared kinds, or the note does not exist in either
// pheromones.json or the pending-suggestion queue.
func appendInfluenceHistory(noteID, action, actorKind, actorName, before, after, reason string) (pheromoneInfluenceEntry, error) {
	if store == nil {
		return pheromoneInfluenceEntry{}, fmt.Errorf("no store initialized")
	}
	noteID = strings.TrimSpace(noteID)
	if noteID == "" {
		return pheromoneInfluenceEntry{}, fmt.Errorf("note identifier is required")
	}
	if !pheromoneInfluenceActionDeclared(action) {
		return pheromoneInfluenceEntry{}, fmt.Errorf("action %q is not one of the declared influence actions %v", action, pheromoneInfluenceActionNames())
	}
	if !pheromoneActorDeclared(actorKind) {
		return pheromoneInfluenceEntry{}, fmt.Errorf("actor kind %q is not declared: must be one of %v", actorKind, pheromoneActors())
	}
	if !pheromoneNoteExists(noteID) {
		return pheromoneInfluenceEntry{}, fmt.Errorf("note %q does not exist", noteID)
	}

	sanitizedReason := ""
	if strings.TrimSpace(reason) != "" {
		if clean, err := colony.SanitizeSignalContent(reason); err == nil {
			sanitizedReason = clean
		}
		// A reason that fails sanitisation (oversized, XML-tag, or
		// prompt-injection content) is dropped rather than blocking the
		// action itself -- a runtime or learning actor's reason is
		// machine-authored text replayed to the owner, and the action it
		// describes must still be recorded even if that description is
		// unsafe to store verbatim.
	}

	entry := pheromoneInfluenceEntry{
		NoteID:    noteID,
		Action:    action,
		ActorKind: actorKind,
		ActorName: actorName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Before:    before,
		After:     after,
		Reason:    sanitizedReason,
	}

	var hf pheromoneInfluenceHistoryFile
	err := store.UpdateJSONAtomically("pheromones-history.json", &hf, func() error {
		if hf.Notes == nil {
			hf.Notes = map[string][]pheromoneInfluenceEntry{}
		}
		hf.Notes[noteID] = append(hf.Notes[noteID], entry)
		hf.Notes = pruneExtinguishedInfluenceHistories(hf.Notes)
		return nil
	})
	if err != nil {
		return pheromoneInfluenceEntry{}, err
	}
	pheromoneInfluenceWriteCount++
	return entry, nil
}

// readInfluenceHistory returns the full, append-ordered history for noteID,
// or an empty slice if the note has no recorded history (including when
// pheromones-history.json does not exist yet). Two calls in a row return
// byte-identical output (TestInfluenceHistoryAppendOnly).
func readInfluenceHistory(noteID string) ([]pheromoneInfluenceEntry, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	var hf pheromoneInfluenceHistoryFile
	if err := store.LoadJSON("pheromones-history.json", &hf); err != nil {
		return []pheromoneInfluenceEntry{}, nil
	}
	entries := hf.Notes[noteID]
	return append([]pheromoneInfluenceEntry{}, entries...), nil
}

// ---------------------------------------------------------------------------
// The five new actions (Task 2)
// ---------------------------------------------------------------------------

// pheromoneInfluenceOutcome carries the result of one of the five new
// influence actions, following the same Found/WouldApply/dry-run shape
// pheromone_approval.go's pendingNoteActionResult already established.
type pheromoneInfluenceOutcome struct {
	Found      bool
	WouldApply bool
	Signal     *colony.PheromoneSignal
	Entry      *pheromoneInfluenceEntry
}

// pheromoneInfluenceActorAllowed is the ONE function enforcing which actor
// kinds may perform which of the five new influence actions, so the
// owner-only rule for revoke and appeal is a single check rather than five
// scattered ones. Reinforce, defer and expire only change how long or how
// strongly an existing meaning applies and may be performed by the runtime
// or a learning pass as well as the owner; revoke and appeal change what the
// owner meant and are owner-only. A disallowed attempt is refused by name --
// both the action and the actor kind are named in the returned reason.
func pheromoneInfluenceActorAllowed(action, actorKind string) (bool, string) {
	if !pheromoneActorDeclared(actorKind) {
		return false, fmt.Sprintf("actor kind %q is not declared: must be one of %v", actorKind, pheromoneActors())
	}
	switch action {
	case pheromoneActionRevoked, pheromoneActionAppealed, pheromoneActionPinned, pheromoneActionUnpinned:
		// Revoke/appeal change what the owner meant; pin/unpin decide
		// whether the owner's own meaning is ever automatically re-tuned at
		// all (plan 203-13's own must_haves truth: "a note the owner
		// pinned is never automatically tuned, in either direction"). All
		// four are owner-only -- the runtime cannot pin or unpin any more
		// than it can revoke or appeal.
		if actorKind != pheromoneActorOwner {
			return false, fmt.Sprintf("actor %q may not %s a note -- only the owner may", actorKind, action)
		}
	}
	return true, ""
}

// pheromoneInfluenceSaveSignals is the one place reinforceNote, deferNote,
// expireNote and revokeNote persist a mutated pheromones.json. Centralising
// the literal store.SaveJSON("pheromones.json", ...) call site here, rather
// than duplicating it in each of the four functions, keeps
// TestOnePheromoneWriterOnly's allowlist to one new named entry
// ("pheromoneInfluenceSaveSignals") instead of four.
func pheromoneInfluenceSaveSignals(pf colony.PheromoneFile) error {
	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		return err
	}
	pheromoneInfluenceWriteCount++
	return nil
}

// findPheromoneSignalByID loads pheromones.json and returns it alongside the
// index of the signal carrying id, or ok=false if no such signal (or no
// store, or no file) exists.
func findPheromoneSignalByID(id string) (pf colony.PheromoneFile, idx int, ok bool) {
	if store == nil {
		return colony.PheromoneFile{}, -1, false
	}
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return colony.PheromoneFile{}, -1, false
	}
	for i := range pf.Signals {
		if pf.Signals[i].ID == id {
			return pf, i, true
		}
	}
	return pf, -1, false
}

// reinforceNote raises a note's strength to the declared ceiling (1.0,
// matching writePheromoneSignal's own reinforcement ceiling) and increments
// its existing reinforcement count. Refuses an unknown note identifier by
// name, and performs no write at all when dryRun is true.
func reinforceNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionReinforced, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	oldStrength := 0.0
	if sig.Strength != nil {
		oldStrength = *sig.Strength
	}
	oldCount := 0
	if sig.ReinforcementCount != nil {
		oldCount = *sig.ReinforcementCount
	}
	const reinforcementCeiling = 1.0
	before := fmt.Sprintf("strength=%.2f reinforcement_count=%d", oldStrength, oldCount)
	after := fmt.Sprintf("strength=%.2f reinforcement_count=%d", reinforcementCeiling, oldCount+1)

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	ceiling := reinforcementCeiling
	newCount := oldCount + 1
	pf.Signals[idx].Strength = &ceiling
	pf.Signals[idx].ReinforcementCount = &newCount
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionReinforced, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// deferNote sets a note's recorded defer-until time. The resolver
// (cmd/pheromone_resolver.go's pheromoneSignalDeferred) excludes the note
// with reason "deferred" until that time passes, then includes it again
// automatically -- no further action is needed. Refuses an unknown note
// identifier or an unparseable until value by name; performs no write when
// dryRun is true.
func deferNote(id, actorKind, actorName, until, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionDeferred, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	until = strings.TrimSpace(until)
	if until == "" {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("a defer-until time is required")
	}
	if _, err := time.Parse(time.RFC3339, until); err != nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("defer-until %q is not a valid RFC3339 timestamp: %w", until, err)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	before := "deferred_until=none"
	if sig.DeferredUntil != nil && *sig.DeferredUntil != "" {
		before = fmt.Sprintf("deferred_until=%s", *sig.DeferredUntil)
	}
	after := fmt.Sprintf("deferred_until=%s", until)

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	deferredUntil := until
	pf.Signals[idx].DeferredUntil = &deferredUntil
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionDeferred, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// expireNote sets a note's expiry to now. It leaves effect immediately
// (cmd/pheromone_resolver.go's existing signalExpiredByTime check already
// excludes a note whose ExpiresAt is not after now), and the history entry
// records the previous expiry. Refuses an unknown note identifier by name;
// performs no write when dryRun is true.
func expireNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionExpired, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	before := "expires_at=none"
	if sig.ExpiresAt != nil && *sig.ExpiresAt != "" {
		before = fmt.Sprintf("expires_at=%s", *sig.ExpiresAt)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	after := fmt.Sprintf("expires_at=%s", now)

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	expiresAt := now
	pf.Signals[idx].ExpiresAt = &expiresAt
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionExpired, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// revokeNote takes a note out of effect permanently. Its history is kept in
// full, and revocation cannot be undone by the runtime -- only a fresh,
// recorded owner action can revoke again or otherwise change the note;
// nothing in this runtime ever clears RevokedAt back to nil. Refuses an
// unknown note identifier, or a non-owner actor, by name; performs no write
// when dryRun is true.
func revokeNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionRevoked, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	before := "revoked_at=none"
	if sig.RevokedAt != nil && *sig.RevokedAt != "" {
		before = fmt.Sprintf("revoked_at=%s", *sig.RevokedAt)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	after := fmt.Sprintf("revoked_at=%s", now)

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	revokedAt := now
	pf.Signals[idx].RevokedAt = &revokedAt
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionRevoked, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// appealNote records an appeal against a rejected queued note and surfaces
// it for the owner's reconsideration -- the recorded entry itself is the
// surfacing mechanism, since it is visible to anyone reading the note's
// history. It never reverses the rejection on its own: the queued item's
// Dismissed/Action/ActionAt fields (cmd/pheromone_approval.go) are left
// exactly as they were; only a fresh, separate owner action through
// suggest-approve could ever change them. Refuses an unknown note
// identifier, a note that is not currently rejected, or a non-owner actor,
// by name; performs no write when dryRun is true.
//
// appealNote reads the queued item via findPendingNote
// (cmd/pheromone_approval.go, same package) rather than duplicating its
// lookup -- calling into that file's behaviour without editing it.
func appealNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionAppealed, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	_, item, _, ok := findPendingNote(id)
	if !ok {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	if item.Action == nil || *item.Action != colony.PendingActionRejected {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q is not rejected -- only a rejected note may be appealed", id)
	}

	before := fmt.Sprintf("action=%s", *item.Action)
	after := before // an appeal never changes the rejected state itself

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true}, nil
	}

	entry, err := appendInfluenceHistory(id, pheromoneActionAppealed, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	return pheromoneInfluenceOutcome{Found: true, Entry: &entry}, nil
}

// ---------------------------------------------------------------------------
// Plan 203-13's three additions: a manual, symmetric-to-reinforce weaken
// action, and owner-only pin/unpin.
// ---------------------------------------------------------------------------

// weakenNote lowers a note's strength to the declared floor (0.0, the exact
// symmetric opposite of reinforceNote's ceiling), the real owner-facing
// action behind --weaken. cmd/pheromone_outcome.go's tuneNoteStrengthFromOutcomes
// records its own, smaller, automatic decreases under this SAME declared
// action name -- the action names the direction (down); the history entry's
// own before/after text names the actual magnitude, which legitimately
// differs between this manual full weaken and an automatic partial step.
// Refuses an unknown note identifier by name; performs no write when dryRun
// is true.
func weakenNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionWeakened, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	oldStrength := 0.0
	if sig.Strength != nil {
		oldStrength = *sig.Strength
	}
	const weakenFloor = 0.0
	before := fmt.Sprintf("strength=%.2f", oldStrength)
	after := fmt.Sprintf("strength=%.2f", weakenFloor)

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	floor := weakenFloor
	pf.Signals[idx].Strength = &floor
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionWeakened, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// pinNote marks a note as owner-pinned: cmd/pheromone_outcome.go's
// tuneNoteStrengthFromOutcomes skips a pinned note entirely, in either
// direction, until unpinNote clears it. Owner-only
// (pheromoneInfluenceActorAllowed); refuses an unknown note identifier by
// name; performs no write when dryRun is true.
func pinNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionPinned, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	before := "pinned=false"
	if sig.Pinned != nil && *sig.Pinned {
		before = "pinned=true"
	}
	after := "pinned=true"

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	pinned := true
	pf.Signals[idx].Pinned = &pinned
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionPinned, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}

// unpinNote clears a note's owner-pinned flag, letting
// tuneNoteStrengthFromOutcomes tune it again. Owner-only; refuses an unknown
// note identifier by name; performs no write when dryRun is true.
func unpinNote(id, actorKind, actorName, reason string, dryRun bool) (pheromoneInfluenceOutcome, error) {
	id = strings.TrimSpace(id)
	if store == nil {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("no store initialized")
	}
	if allowed, why := pheromoneInfluenceActorAllowed(pheromoneActionUnpinned, actorKind); !allowed {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("%s", why)
	}
	pf, idx, found := findPheromoneSignalByID(id)
	if !found {
		return pheromoneInfluenceOutcome{}, fmt.Errorf("note %q not found", id)
	}
	sig := pf.Signals[idx]

	before := "pinned=false"
	if sig.Pinned != nil && *sig.Pinned {
		before = "pinned=true"
	}
	after := "pinned=false"

	if dryRun {
		return pheromoneInfluenceOutcome{Found: true, WouldApply: true, Signal: &sig}, nil
	}

	unpinned := false
	pf.Signals[idx].Pinned = &unpinned
	if err := pheromoneInfluenceSaveSignals(pf); err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	entry, err := appendInfluenceHistory(id, pheromoneActionUnpinned, actorKind, actorName, before, after, reason)
	if err != nil {
		return pheromoneInfluenceOutcome{}, err
	}
	updated := pf.Signals[idx]
	return pheromoneInfluenceOutcome{Found: true, Signal: &updated, Entry: &entry}, nil
}
