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
	"fmt"
	"path/filepath"
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

	if store == nil {
		// "No project set up here" is NOT "a project that has just started".
		// Conflating them is how a fresh checkout ends up being told to carry
		// on with a build that does not exist.
		in.NoColony = true
		in.Facts = unavailableLifecycleFacts(resolveAetherRootPath(), time.Now().UTC(), "store is not initialized")
		return in
	}
	root := filepath.Dir(filepath.Dir(store.BasePath()))
	observedAt := time.Now().UTC().Truncate(time.Second)
	facts, _ := loadLifecycleFacts(root, store, observedAt)
	in.Facts = facts
	in.State = facts.State.Value
	in.NoColony = facts.State.Source.Provenance == LifecycleFactMissing || colonyStateIsUnstarted(in.State)
	if in.NoColony {
		return in
	}

	for _, flag := range facts.Blockers.Value {
		if flag.Resolved {
			continue
		}
		in.Flags = append(in.Flags, flag)
		if flag.Source == planFinalizeFailureSource && in.PlanBlocker == nil {
			found := flag
			in.PlanBlocker = &found
		}
	}

	in.Recovery = loadActiveRecoveryGuidanceReadOnly(in.State, store.BasePath())
	// Keep the disk-backed resolver input equivalent to the explicit-state
	// adapter input. Without this fact, the status/card path called a stalled
	// dispatch ordinary "continue" while every lifecycle adapter correctly
	// routed the same open operation through canonical resume.
	in.BuildLooksAbandoned = buildLooksAbandoned(in.State)
	in.HandoffExists = fileExists(handoffDocumentPath())
	pf := colony.PheromoneFile{Signals: facts.Signals.Value}
	in.Signals = extractSignalTextsFrom(&pf, 8)
	in.ActiveTodos = sessionActiveTodosFromState(in.State)

	return in
}

// loadNextActionInputForGreeting is loadNextActionInputForCommand plus the
// memory block (198.2 plan 03): the owner's own preferences, the strongest
// learned habits, and the last helper's relay note. It is the ONLY caller
// that populates nextActionInput.Memory -- hookSessionStartCmd is the ONLY
// caller of this function -- so every other closing card, built through
// loadNextActionInput or loadNextActionInputForCommand, stays byte-identical
// to before this field existed.
//
// Every read here is read-only, the same rule the rest of this file is held
// to: readUserPreferences and loadStrongestRuntimeInstincts read files
// directly, and loadWorkerHandoffRecords reads the handoff log. None of the
// three writes, and none of the three re-derives a ranking or a selection
// rule of its own -- preferences and habits reuse the exact readers named in
// the plan (readUserPreferences pair, loadStrongestRuntimeInstincts), so a
// second implementation of either can never quietly drift from the one this
// file shares with the rest of the runtime.
func loadNextActionInputForGreeting() nextActionInput {
	in := loadNextActionInputForCommand("")
	if in.NoColony {
		return in
	}
	in.Memory = buildNextActionMemory(in.State)
	return in
}

// buildNextActionMemory reads the three memory parts and returns them
// exactly as read -- no ranking, no filtering beyond what each reader already
// does. A part with nothing to say is left as its zero value; the card
// (renderNextActionMemory, cmd/next_action_card.go) is what omits it (D-14).
func buildNextActionMemory(state colony.ColonyState) nextActionMemory {
	var mem nextActionMemory

	hubDir := resolveHubPath()
	aetherRoot := resolveAetherRootPath()
	var prefs []string
	prefs = append(prefs, readUserPreferences(filepath.Join(hubDir, "QUEEN.md"))...)
	prefs = append(prefs, readUserPreferences(filepath.Join(aetherRoot, ".aether", "QUEEN.md"))...)
	if len(prefs) > 0 {
		plural := "s"
		if len(prefs) == 1 {
			plural = ""
		}
		mem.Preferences = fmt.Sprintf("%d preference%s set -- e.g. %s", len(prefs), plural, prefs[0])
	}

	for _, inst := range loadStrongestRuntimeInstincts(store, &state, 3) {
		if action := strings.TrimSpace(inst.Action); action != "" {
			mem.Habits = append(mem.Habits, action)
		}
	}

	mem.RelayNote = latestHandoffSentence()

	return mem
}

// latestHandoffSentence names the last helper and what it left for the next
// one, from the same handoff log the capsule reads
// (renderWorkerHandoffSection / renderHandoffSectionNamed,
// cmd/codex_dispatch_contract.go) -- reduced to one sentence rather than the
// capsule's full markdown section. Empty when no handoff has been recorded.
func latestHandoffSentence() string {
	records, err := loadWorkerHandoffRecords()
	if err != nil || len(records) == 0 {
		return ""
	}

	latest := records[0]
	for _, record := range records[1:] {
		if handoffFreshnessTime(record.Freshness).After(handoffFreshnessTime(latest.Freshness)) {
			latest = record
		}
	}

	worker := strings.TrimSpace(latest.WorkerName)
	if worker == "" {
		worker = "a helper"
	}

	left := strings.TrimSpace(latest.Summary)
	if left == "" && len(latest.NextWorkerInstructions) > 0 {
		left = strings.TrimSpace(latest.NextWorkerInstructions[0])
	}
	if left == "" && len(latest.DoNotRepeat) > 0 {
		left = strings.TrimSpace(latest.DoNotRepeat[0])
	}
	if left == "" {
		return ""
	}

	return fmt.Sprintf("The last helper (%s) left a note for the next one: %s", worker, left)
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
	state, _, err := loadColonyStateWithCompatibilityRepairReadOnlyFromPath(filepath.Join(store.BasePath(), "COLONY_STATE.json"))
	if err != nil {
		return colony.ColonyState{}, false
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
