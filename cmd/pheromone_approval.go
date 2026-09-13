package cmd

import (
	"fmt"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// pendingNoteWriteCount is a test seam counting every store persistence this
// file's mutating functions perform for the shared tick-to-approve queue
// (colony.PendingSuggestion) or a linked pheromone signal. A --dry-run call
// must return before incrementing this counter -- TestApprovalDryRunDoesNotMutate
// asserts the counter is unchanged across a dry-run call, proving an
// inspection never mutates state (CLAUDE.md's Definition of Done corollary:
// "an inspection or --dry-run command must not mutate state"). Production
// code never reads this value.
var pendingNoteWriteCount int

// pendingNoteOrigin reports a queued item's origin, defaulting a legacy item
// with no Origin field (written before this plan) to
// colony.PendingOriginSuggestion -- an import is new to this plan, so no
// pre-existing item can genuinely be one.
func pendingNoteOrigin(item colony.PendingSuggestion) string {
	if item.Origin == nil || *item.Origin == "" {
		return colony.PendingOriginSuggestion
	}
	return *item.Origin
}

// pendingNoteOriginLabel renders a queued item's origin in plain English.
// Per CLAUDE.md's mandate, no repo-invented word ("pheromone", "colony",
// "quarantine") reaches the owner untranslated: an imported item reads as
// coming from another project, never as "an imported cross-colony signal".
func pendingNoteOriginLabel(item colony.PendingSuggestion) string {
	if pendingNoteOrigin(item) == colony.PendingOriginImport {
		return "from another project"
	}
	return "suggested by the program"
}

// enqueuePendingNote is the one function that adds an item to the shared
// tick-to-approve queue -- the SAME colony.PendingSuggestion queue
// suggest-analyze already populates in COLONY_STATE.json's
// PendingSuggestions field. Both a runtime-proposed suggestion and a
// cross-project import route through this function so the owner learns one
// approval surface, never two (D-07/D-10, 203-CLASSIC-SYNTHESIS.md
// SYN-203-11).
//
// Deduplication reuses the existing content-hash comparison
// (mergePendingSuggestions in cmd/suggest_analyze.go already establishes the
// same "same hash, not dismissed" rule): a new item whose content hash
// matches an existing, non-dismissed queued entry is reported as a
// duplicate and not added again -- no second dedup mechanism.
//
// origin must be colony.PendingOriginSuggestion or colony.PendingOriginImport.
// signalID, when non-empty, links the queued item to the colony.PheromoneSignal
// it names (an import's already-quarantined stored note) so approve/reject
// can find and act on that signal.
func enqueuePendingNote(sigType, content, reason, origin, signalID string) (colony.PendingSuggestion, bool, error) {
	if store == nil {
		return colony.PendingSuggestion{}, false, fmt.Errorf("no store initialized")
	}

	contentHash := "sha256:" + sha256Sum(content)
	now := time.Now().UTC().Format(time.RFC3339)

	item := colony.PendingSuggestion{
		ID:          generateSignalID(),
		Type:        sigType,
		Content:     content,
		Reason:      reason,
		ContentHash: contentHash,
		CreatedAt:   now,
		Dismissed:   false,
	}
	o := origin
	item.Origin = &o
	if signalID != "" {
		s := signalID
		item.SignalID = &s
	}

	duplicated := false
	var cs colony.ColonyState
	err := store.UpdateJSONAtomically("COLONY_STATE.json", &cs, func() error {
		existing := []colony.PendingSuggestion{}
		if cs.PendingSuggestions != nil {
			existing = *cs.PendingSuggestions
		}
		for _, e := range existing {
			if !e.Dismissed && e.ContentHash == contentHash {
				duplicated = true
				return nil
			}
		}
		merged := append(existing, item)
		cs.PendingSuggestions = &merged
		return nil
	})
	if err != nil {
		return colony.PendingSuggestion{}, false, err
	}
	pendingNoteWriteCount++
	if duplicated {
		return colony.PendingSuggestion{}, true, nil
	}
	return item, false, nil
}

// findPendingNote loads the current colony state and returns the pending
// item with the given ID plus its index in cs.PendingSuggestions, or
// ok=false if the item (or the colony state) cannot be found.
func findPendingNote(id string) (cs colony.ColonyState, item colony.PendingSuggestion, idx int, ok bool) {
	if store == nil {
		return colony.ColonyState{}, colony.PendingSuggestion{}, -1, false
	}
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		return cs, colony.PendingSuggestion{}, -1, false
	}
	if cs.PendingSuggestions == nil {
		return cs, colony.PendingSuggestion{}, -1, false
	}
	for i, s := range *cs.PendingSuggestions {
		if s.ID == id {
			return cs, s, i, true
		}
	}
	return cs, colony.PendingSuggestion{}, -1, false
}

// clearSignalQuarantine is the ONE function in this runtime permitted to set
// a stored pheromone signal's Quarantined flag to false. Every quarantine
// release -- today, an owner-approved cross-project import (D-10) -- must go
// through here. TestOneApprovalSurface's source scan fails by name if a
// second function anywhere in cmd/ assigns false to a PheromoneSignal's
// Quarantined field.
func clearSignalQuarantine(signalID string) (colony.PheromoneSignal, bool, error) {
	if store == nil {
		return colony.PheromoneSignal{}, false, fmt.Errorf("no store initialized")
	}
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return colony.PheromoneSignal{}, false, err
	}
	for i := range pf.Signals {
		if pf.Signals[i].ID == signalID {
			cleared := false
			pf.Signals[i].Quarantined = &cleared
			if err := store.SaveJSON("pheromones.json", pf); err != nil {
				return colony.PheromoneSignal{}, false, err
			}
			pendingNoteWriteCount++
			return pf.Signals[i], true, nil
		}
	}
	return colony.PheromoneSignal{}, false, nil
}

// stampPendingNoteAction records the owner's decision on a queued item with
// a timestamp, then marks it no longer pending. This is deliberately the
// first entry in what plan 203-11 extends into a full immutable action
// history (BIO-08's remaining verbs: reinforce/defer/expire/revoke/appeal) --
// today it is one action, one timestamp, kept on the item rather than
// deleted, so that later history has something to build on instead of
// starting empty. Marking Dismissed alongside the action preserves
// suggest-approve's pre-existing "no longer pending" listing behavior
// unchanged: filterActiveSuggestions already excludes dismissed items, so an
// actioned item stops appearing as pending exactly as it did before this
// plan.
func stampPendingNoteAction(item *colony.PendingSuggestion, action string) {
	item.Dismissed = true
	a := action
	item.Action = &a
	now := time.Now().UTC().Format(time.RFC3339)
	item.ActionAt = &now
}

// pendingNoteActionResult carries the outcome of approvePendingNote,
// editPendingNote, or rejectPendingNote.
type pendingNoteActionResult struct {
	Found      bool
	WouldApply bool
	Item       colony.PendingSuggestion
	Signal     *colony.PheromoneSignal
}

// approvePendingNote is the only function that approves a queued item,
// whether it originated as a runtime suggestion or a cross-project import.
// Approving a suggestion writes it as an active note through the existing
// single writer (writePheromoneSignal). Approving an import writes NOTHING
// new -- the note already exists, quarantined, since the import path
// created it -- it only clears that stored note's quarantine flag through
// clearSignalQuarantine, the one function permitted to do so. A --dry-run
// call returns a preview and performs no store write at all (no queue
// update, no signal write, no quarantine clear).
func approvePendingNote(id string, dryRun bool) (pendingNoteActionResult, error) {
	if store == nil {
		return pendingNoteActionResult{}, fmt.Errorf("no store initialized")
	}
	cs, item, idx, ok := findPendingNote(id)
	if !ok {
		return pendingNoteActionResult{Found: false}, nil
	}
	if dryRun {
		return pendingNoteActionResult{Found: true, WouldApply: true, Item: item}, nil
	}

	result := pendingNoteActionResult{Found: true}

	if pendingNoteOrigin(item) == colony.PendingOriginImport {
		if item.SignalID == nil || *item.SignalID == "" {
			return pendingNoteActionResult{}, fmt.Errorf("queued import %q has no linked signal id", id)
		}
		sig, found, err := clearSignalQuarantine(*item.SignalID)
		if err != nil {
			return pendingNoteActionResult{}, err
		}
		if !found {
			return pendingNoteActionResult{}, fmt.Errorf("linked signal %q for queued import %q was not found", *item.SignalID, id)
		}
		result.Signal = &sig
	} else {
		priority := "normal"
		switch item.Type {
		case "REDIRECT":
			priority = "high"
		case "FEEDBACK":
			priority = "low"
		}
		sig, _, err := writePheromoneSignal(item.Type, item.Content, priority, "aether-suggest", item.Reason, "", 0.7, nil)
		if err != nil {
			return pendingNoteActionResult{}, err
		}
		result.Signal = &sig
	}

	stampPendingNoteAction(&item, colony.PendingActionAccepted)
	pending := *cs.PendingSuggestions
	pending[idx] = item
	cs.PendingSuggestions = &pending
	if err := store.SaveJSON("COLONY_STATE.json", cs); err != nil {
		return pendingNoteActionResult{}, err
	}
	pendingNoteWriteCount++
	result.Item = item
	return result, nil
}

// editPendingNote stores the owner's replacement wording for a queued item,
// recomputing its content hash from the edited text and recording that the
// item was edited. Editing never approves or writes a pheromone signal
// itself and never touches Dismissed -- accepting or rejecting the (now
// edited) item is still a separate, later decision. A --dry-run call
// returns a preview and writes nothing.
func editPendingNote(id, newContent string, dryRun bool) (pendingNoteActionResult, error) {
	if store == nil {
		return pendingNoteActionResult{}, fmt.Errorf("no store initialized")
	}
	cs, item, idx, ok := findPendingNote(id)
	if !ok {
		return pendingNoteActionResult{Found: false}, nil
	}
	if dryRun {
		preview := item
		preview.Content = newContent
		preview.ContentHash = "sha256:" + sha256Sum(newContent)
		return pendingNoteActionResult{Found: true, WouldApply: true, Item: preview}, nil
	}

	item.Content = newContent
	item.ContentHash = "sha256:" + sha256Sum(newContent)
	edited := colony.PendingActionEdited
	item.Action = &edited
	now := time.Now().UTC().Format(time.RFC3339)
	item.ActionAt = &now

	pending := *cs.PendingSuggestions
	pending[idx] = item
	cs.PendingSuggestions = &pending
	if err := store.SaveJSON("COLONY_STATE.json", cs); err != nil {
		return pendingNoteActionResult{}, err
	}
	pendingNoteWriteCount++
	return pendingNoteActionResult{Found: true, Item: item}, nil
}

// rejectPendingNote marks a queued item dismissed and rejected. It never
// touches a linked stored signal: a rejected import's quarantine flag is
// left exactly as it was -- only approvePendingNote's clearSignalQuarantine
// call may clear it. A --dry-run call returns a preview and writes nothing.
func rejectPendingNote(id string, dryRun bool) (pendingNoteActionResult, error) {
	if store == nil {
		return pendingNoteActionResult{}, fmt.Errorf("no store initialized")
	}
	cs, item, idx, ok := findPendingNote(id)
	if !ok {
		return pendingNoteActionResult{Found: false}, nil
	}
	if dryRun {
		return pendingNoteActionResult{Found: true, WouldApply: true, Item: item}, nil
	}

	stampPendingNoteAction(&item, colony.PendingActionRejected)
	pending := *cs.PendingSuggestions
	pending[idx] = item
	cs.PendingSuggestions = &pending
	if err := store.SaveJSON("COLONY_STATE.json", cs); err != nil {
		return pendingNoteActionResult{}, err
	}
	pendingNoteWriteCount++
	return pendingNoteActionResult{Found: true, Item: item}, nil
}

// pendingNotesToMap renders queued items for suggest-approve's listing
// output, adding origin and a plain-English origin label alongside the
// fields cmd/suggest_analyze.go's pendingSuggestionsToMap already produced,
// so an imported item is visibly distinguishable from a runtime suggestion
// in the same list.
func pendingNotesToMap(suggestions []colony.PendingSuggestion) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(suggestions))
	for _, s := range suggestions {
		result = append(result, map[string]interface{}{
			"id":           s.ID,
			"type":         s.Type,
			"content":      s.Content,
			"reason":       s.Reason,
			"content_hash": s.ContentHash,
			"created_at":   s.CreatedAt,
			"dismissed":    s.Dismissed,
			"origin":       pendingNoteOrigin(s),
			"origin_label": pendingNoteOriginLabel(s),
		})
	}
	return result
}
