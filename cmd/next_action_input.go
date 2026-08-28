package cmd

// Phase 197 plan 01 -- the impure half.
//
// This file does ALL the reading for the "what next" answer and none of the
// deciding. It calls readers that already exist and are already tested, puts
// their results into nextActionInput, and returns. If a branch needs deciding
// it belongs in resolveNextAction (cmd/next_action.go), where it can be tested
// without a filesystem.
//
// It is READ-ONLY, and that is asserted rather than promised
// (TestLoadNextActionInputDoesNotMutate fingerprints every file under the
// project's data directory before and after). This repository has shipped an
// inspection command that quietly wrote to saved state for months while its own
// flag documentation said "report without modifying"; that is the failure this
// file's design and that assertion exist to prevent.

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// loadNextActionInput gathers everything the resolver needs from the project on
// disk. Nothing is written.
func loadNextActionInput() nextActionInput {
	return loadNextActionInputForCommand("")
}

// loadNextActionInputForCommand is loadNextActionInput plus the name of the
// command that just ran, when the caller knows it. That is one of the eight
// things the owner is told -- what changed -- and only the caller knows it.
func loadNextActionInputForCommand(lastCommand string) nextActionInput {
	in := nextActionInput{LastCommand: strings.TrimSpace(lastCommand)}

	state, ok := readColonyStateWithoutWriting()
	if !ok {
		// "No project set up here" is NOT "a project that has just started".
		// Conflating them is how a fresh checkout ends up being told to carry
		// on with a build that does not exist.
		in.NoColony = true
		return in
	}
	in.State = state

	if flags, ok := loadFlagsFile(store); ok {
		for _, flag := range flags.Decisions {
			if flag.Resolved {
				continue
			}
			in.Flags = append(in.Flags, flag)
		}
	}
	if blocker, ok := activePlanFinalizeFailureFlag(store); ok {
		found := blocker
		in.PlanBlocker = &found
	}

	in.Recovery = loadActiveRecoveryGuidance(state)
	in.HandoffExists = fileExists(handoffDocumentPath())
	in.Signals = extractSignalTexts(8)
	in.ActiveTodos = sessionActiveTodosFromState(state)
	in.BuildLooksAbandoned = buildLooksAbandoned(state)

	return in
}

// readColonyStateWithoutWriting reads and normalises the saved colony state.
//
// It deliberately does NOT use loadActiveColonyState. That path runs two
// repairs -- repairLegacyNumericStringFields and repairMissingPlanFromArtifacts
// -- and BOTH persist their result back to COLONY_STATE.json. Correct for a
// command that is about to act; wrong for one that is only being asked what to
// do next. The legacy-numeric repair is applied here in memory only, so an old
// state file still reads correctly without the act of reading it rewriting the
// project's saved data.
func readColonyStateWithoutWriting() (colony.ColonyState, bool) {
	if store == nil {
		return colony.ColonyState{}, false
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		raw, rawErr := store.LoadRawJSON("COLONY_STATE.json")
		if rawErr != nil {
			return colony.ColonyState{}, false
		}
		repairedRaw, repaired, repairErr := repairLegacyNumericStringFields(raw)
		if repairErr != nil || !repaired {
			return colony.ColonyState{}, false
		}
		if err := json.Unmarshal(repairedRaw, &state); err != nil {
			return colony.ColonyState{}, false
		}
	}

	// A state file with no goal is a leftover, not a project.
	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		return colony.ColonyState{}, false
	}
	return normalizeLegacyColonyState(state), true
}

// buildLooksAbandoned reports whether a build has been sitting past the
// abandoned-build threshold with no dispatch record behind it. It is a fact
// about the project, gathered here; what to advise about it is the resolver's
// decision.
func buildLooksAbandoned(state colony.ColonyState) bool {
	if state.State != colony.StateEXECUTING && state.State != colony.StateBUILT {
		return false
	}
	if state.CurrentPhase < 1 || state.BuildStartedAt == nil {
		return false
	}
	if time.Since(state.BuildStartedAt.UTC()) < abandonedBuildThreshold {
		return false
	}
	return !loadCodexContinueManifest(state.CurrentPhase).Present
}
