package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// deriveBuildKnowledgeDeltas is CAP-066's pure derivation of an attempt's own
// content-level decision and learning deltas from the worker handoffs its
// own workers actually left behind. It has exactly two callers: the
// native/in-repo build lane (cmd/codex_build.go, immediately after that
// lane's own attempt is durably sealed) and the external/wrapper
// build-finalize lane (cmd/codex_build_finalize.go, alongside its existing
// attachResultFilePrecision/attachBuildPlanRealityReport block) -- both
// reporting-only call sites that hand the result straight to
// attachBuildKnowledgeDeltas (cmd/build_attempt.go), the only writer.
//
// Given this attempt's own phase number and its own resolved dispatches, it
// loads every persisted worker handoff record (loadWorkerHandoffRecords,
// cmd/codex_dispatch_contract.go) and keeps only the records that belong to
// THIS attempt: same phase, and a worker name found among dispatches. A kept
// record's OpenDecisions become deltas of kind "decision"; its DoNotRepeat
// and NextWorkerInstructions become deltas of kind "learning" -- the same
// two-kind vocabulary buildAttemptKnowledgeDelta's own doc comment declares
// (cmd/codex_build_finalize.go).
//
// Ordering is deterministic across repeated calls on the same input: by
// worker name in the attempt's own dispatch order, then by each matching
// record in the order it was persisted, then by sentence order within that
// record's own fields (OpenDecisions, then DoNotRepeat, then
// NextWorkerInstructions). Blank sentences are skipped; an exact
// (kind, summary) duplicate -- whether from the same worker or two different
// workers -- is kept once, at its first occurrence in that order.
//
// A worker's sentence is untrusted input: these summaries are replayed onto
// an owner-facing card (lifecycleCloseoutKnowledgeDeltaEvidence,
// cmd/lifecycle_closeout.go), so every sentence is run through the same
// sanitiser this repository already applies to worker text before storage
// (colony.SanitizeSignalContent, mirroring sanitizedWorkerSentence in
// cmd/memory_feed.go). A sentence that fails sanitisation is dropped rather
// than stored unsafely or replaced with a placeholder.
//
// Performs no write of any kind -- it only reads already-persisted handoff
// records and returns a value. An attempt whose workers left none of these
// returns nil, not an empty non-nil slice, so attachBuildKnowledgeDeltas can
// record absence as absence rather than an empty list presented as evidence.
func deriveBuildKnowledgeDeltas(phaseNum int, dispatches []codexBuildDispatch) []buildAttemptKnowledgeDelta {
	if len(dispatches) == 0 {
		return nil
	}

	// This attempt's own worker names, in dispatch order -- the ordering
	// anchor for the deterministic output below, and the membership test
	// that keeps a same-phase handoff belonging to a DIFFERENT attempt's
	// worker from contributing anything.
	var workerOrder []string
	wantedWorkers := map[string]struct{}{}
	for _, dispatch := range dispatches {
		name := strings.TrimSpace(dispatch.Name)
		if name == "" {
			continue
		}
		if _, seen := wantedWorkers[name]; seen {
			continue
		}
		wantedWorkers[name] = struct{}{}
		workerOrder = append(workerOrder, name)
	}
	if len(workerOrder) == 0 {
		return nil
	}

	records, err := loadWorkerHandoffRecords()
	if err != nil || len(records) == 0 {
		return nil
	}

	// Group this attempt's own records by worker name, preserving both the
	// order records were persisted in (the slice order loadWorkerHandoffRecords
	// returns) and, within each record, its own field order.
	byWorker := map[string][]workerHandoffRecord{}
	for _, record := range records {
		if record.Phase != phaseNum {
			continue
		}
		name := strings.TrimSpace(record.WorkerName)
		if _, wanted := wantedWorkers[name]; !wanted {
			continue
		}
		byWorker[name] = append(byWorker[name], record)
	}

	seenDelta := map[string]struct{}{}
	var deltas []buildAttemptKnowledgeDelta
	appendDelta := func(kind, raw string) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return
		}
		sanitized, sanitizeErr := colony.SanitizeSignalContent(trimmed)
		if sanitizeErr != nil {
			// Untrusted worker text that fails sanitisation is dropped, not
			// stored under a placeholder -- unlike a single required
			// attribution sentence (sanitizedWorkerSentence), a knowledge
			// delta has no obligation to exist at all.
			return
		}
		key := kind + "\x00" + sanitized
		if _, dup := seenDelta[key]; dup {
			return
		}
		seenDelta[key] = struct{}{}
		deltas = append(deltas, buildAttemptKnowledgeDelta{Kind: kind, Summary: sanitized})
	}

	for _, workerName := range workerOrder {
		for _, record := range byWorker[workerName] {
			for _, sentence := range record.OpenDecisions {
				appendDelta("decision", sentence)
			}
			for _, sentence := range record.DoNotRepeat {
				appendDelta("learning", sentence)
			}
			for _, sentence := range record.NextWorkerInstructions {
				appendDelta("learning", sentence)
			}
		}
	}

	return deltas
}
