package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// episode_index.go is the one ordered lineage of episodes and syntheses
// that status, history and watch share (LIVE-05, LIVE-07, D-11, D-12).
//
// This repository already has three durable record types that each carry
// their own outcome: the Swarm episode (cmd/swarm_episode.go), the saved
// Oracle research document (cmd/oracle_research_doc.go), and the build/
// check attempt record (cmd/build_attempt.go). The failure mode this file
// exists to prevent is inventing a FOURTH record type that duplicates all
// three and then has to be kept in step with them by hand -- exactly the
// drift 202-CLASSIC-SYNTHESIS.md names as this repository's own recurring
// cost. loadColonyEpisodeIndex is a pure projection over the three sources
// that already exist: it creates nothing, writes nothing, and owns no
// identity of its own.

// The kinds an index entry may carry. A build attempt whose CheckFix field
// is set is reported as colonyEpisodeKindCheckAttempt rather than
// colonyEpisodeKindBuildAttempt -- the distinction the owner actually cares
// about -- even though both live in the same durable buildAttemptRecord
// store.
//
// colonyEpisodeKindQuick (release 1.0.85) is a deliberate fourth kind, not
// the second-event-transport TestOneLiveEventModelOnly guards against: a
// quick job's attempt record (cmd/command_truth.go's quickAttemptRecord) is
// persisted through store.SaveJSON, the exact same durable-JSON discipline
// buildAttemptRecord already uses, not a second serialized-timestamp
// transport reaching disk through its own write path.
const (
	colonyEpisodeKindSwarm          = "swarm"
	colonyEpisodeKindOracleResearch = "oracle-research"
	colonyEpisodeKindBuildAttempt   = "build-attempt"
	colonyEpisodeKindCheckAttempt   = "check-attempt"
	colonyEpisodeKindQuick          = "quick"
)

// The record SOURCES loadColonyEpisodeIndex reads. This was deliberately
// three, matching the first three kinds above one-for-one at the source
// level (a build attempt and a check attempt share one source, the build
// attempts directory) -- see TestEpisodeIndexCoversThreeRecordSources --
// and is now four with colonyEpisodeSourceQuick (release 1.0.85), read from
// the durable quick-attempt records a quick job persists
// (cmd/command_truth.go's persistQuickAttempt).
const (
	colonyEpisodeSourceSwarm          = "swarm"
	colonyEpisodeSourceOracleResearch = "oracle-research"
	colonyEpisodeSourceBuildAttempts  = "build-attempts"
	colonyEpisodeSourceQuick          = "quick"
)

// colonyEpisodeCostRef references the spend ledger authority's own key
// (the phase number, cmd/spend_ledger.go) rather than copying a token or
// currency figure onto the entry -- the index must never become a second
// money record. Phase zero means this entry has no ledger to resolve:
// Swarm and Oracle research carry no per-run ledger today (spend_ledger.go's
// two workflows, build and continue, are keyed by phase only), and
// resolving a zero phase correctly yields "not reported" rather than a
// fabricated figure.
type colonyEpisodeCostRef struct {
	Phase int `json:"phase,omitempty"`
}

// colonyEpisodeEntry is one row in the shared lineage: its kind, its
// subject, its outcome (and whether that outcome is actually known), its
// standing through the shared vocabulary (workStandingLabel,
// cmd/partial_work_label.go), its start/end time, a reference to its cost,
// and the path to read it in full.
type colonyEpisodeEntry struct {
	Kind           string               `json:"kind"`
	Subject        string               `json:"subject"`
	Outcome        string               `json:"outcome"`
	OutcomeKnown   bool                 `json:"outcome_known"`
	Standing       workStanding         `json:"standing"`
	StandingReason string               `json:"standing_reason,omitempty"`
	StartedAt      time.Time            `json:"started_at"`
	EndedAt        time.Time            `json:"ended_at"`
	Cost           colonyEpisodeCostRef `json:"cost"`
	Path           string               `json:"path"`
}

// colonyEpisodeIndex is loadColonyEpisodeIndex's whole result: the ordered
// entries (newest first), plus the name of any source that was missing or
// unreadable.
type colonyEpisodeIndex struct {
	Entries     []colonyEpisodeEntry `json:"entries"`
	Unavailable []string             `json:"unavailable_sources,omitempty"`
}

// loadColonyEpisodeIndex reads the three durable record sources that
// already exist -- the Swarm episode store, the saved Oracle research
// documents, and the recorded build/check attempts -- and folds them into
// one ordered slice, newest first, with a stable tiebreak for equal
// timestamps. It is a pure read: nothing here creates, modifies, or
// removes anything. A missing or unreadable source contributes no entries
// and no error -- only its name, in Unavailable, so a caller can tell "this
// colony has never run Swarm" apart from "Swarm ran and recorded nothing".
func loadColonyEpisodeIndex(root string, s *storage.Store) (colonyEpisodeIndex, error) {
	var idx colonyEpisodeIndex

	swarmEntries, swarmAvailable := loadSwarmEpisodeIndexEntries(root)
	if !swarmAvailable {
		idx.Unavailable = append(idx.Unavailable, colonyEpisodeSourceSwarm)
	}
	idx.Entries = append(idx.Entries, swarmEntries...)

	researchEntries, researchAvailable := loadOracleResearchIndexEntries(root)
	if !researchAvailable {
		idx.Unavailable = append(idx.Unavailable, colonyEpisodeSourceOracleResearch)
	}
	idx.Entries = append(idx.Entries, researchEntries...)

	attemptEntries, attemptsAvailable := loadBuildAttemptIndexEntries(s)
	if !attemptsAvailable {
		idx.Unavailable = append(idx.Unavailable, colonyEpisodeSourceBuildAttempts)
	}
	idx.Entries = append(idx.Entries, attemptEntries...)

	quickEntries, quickAvailable := loadQuickAttemptIndexEntries(s)
	if !quickAvailable {
		idx.Unavailable = append(idx.Unavailable, colonyEpisodeSourceQuick)
	}
	idx.Entries = append(idx.Entries, quickEntries...)

	sortColonyEpisodeEntriesNewestFirst(idx.Entries)
	return idx, nil
}

// colonyEpisodeEntryTime is an entry's own recency anchor: its end time
// when recorded, falling back to its start time -- never "now".
func colonyEpisodeEntryTime(entry colonyEpisodeEntry) time.Time {
	if !entry.EndedAt.IsZero() {
		return entry.EndedAt
	}
	return entry.StartedAt
}

// sortColonyEpisodeEntriesNewestFirst orders entries newest first via a
// stable sort. Two entries with equal recency time (including two with no
// recorded time at all) keep their relative order from the append sequence
// above -- and that sequence is itself deterministic across repeated loads,
// because every loader above lists its own directory in sorted order before
// appending anything.
func sortColonyEpisodeEntriesNewestFirst(entries []colonyEpisodeEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return colonyEpisodeEntryTime(entries[i]).After(colonyEpisodeEntryTime(entries[j]))
	})
}

// colonyEpisodeKindLabel names a kind in plain English, for any renderer
// (status, history, watch) that wants to say what an entry is without
// re-deriving the word itself.
func colonyEpisodeKindLabel(kind string) string {
	switch kind {
	case colonyEpisodeKindSwarm:
		return "Swarm run"
	case colonyEpisodeKindOracleResearch:
		return "Oracle research"
	case colonyEpisodeKindBuildAttempt:
		return "Build"
	case colonyEpisodeKindCheckAttempt:
		return "Check"
	case colonyEpisodeKindQuick:
		return "Quick job"
	default:
		return "Episode"
	}
}

// --- Source 1: Swarm episodes -------------------------------------------

// loadSwarmEpisodeIndexEntries reads every persisted Swarm episode
// (.aether/data/swarms/*/episode.json) directly off disk -- deliberately
// not through storage.NewStore, which creates directories on construction
// and would violate this function's read-only guarantee, mirroring
// collectUnverifiedSwarmWork's identical discipline
// (cmd/partial_work_label.go). available is false only when the swarms
// directory itself does not exist or cannot be listed; an existing
// directory with a malformed episode file still returns available true,
// with that file listed as a standing-unknown entry rather than dropped.
func loadSwarmEpisodeIndexEntries(root string) ([]colonyEpisodeEntry, bool) {
	swarmsDir := filepath.Join(root, ".aether", "data", "swarms")
	dirEntries, err := os.ReadDir(swarmsDir)
	if err != nil {
		return nil, false
	}

	names := make([]string, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	var out []colonyEpisodeEntry
	for _, name := range names {
		episodePath := filepath.Join(swarmsDir, name, "episode.json")
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "swarms", name, "episode.json"))

		data, readErr := os.ReadFile(episodePath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			standing, reason := resolveMalformedItemStanding("the episode record could not be read: " + readErr.Error())
			out = append(out, colonyEpisodeEntry{
				Kind: colonyEpisodeKindSwarm, Subject: name,
				Standing: standing, StandingReason: reason, Path: relPath,
			})
			continue
		}

		var record swarmEpisodeRecord
		if jsonErr := json.Unmarshal(data, &record); jsonErr != nil || record.SchemaVersion != swarmEpisodeSchemaVersion {
			standing, reason := resolveMalformedItemStanding("the episode record is malformed or from an incompatible schema version")
			out = append(out, colonyEpisodeEntry{
				Kind: colonyEpisodeKindSwarm, Subject: name,
				Standing: standing, StandingReason: reason, Path: relPath,
			})
			continue
		}

		out = append(out, swarmEpisodeIndexEntryFrom(record, relPath))
	}
	return out, true
}

func swarmEpisodeIndexEntryFrom(record swarmEpisodeRecord, relPath string) colonyEpisodeEntry {
	var startedAt, endedAt time.Time
	if t, err := time.Parse(time.RFC3339Nano, record.StartedAt); err == nil {
		startedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, record.EndedAt); err == nil {
		endedAt = t
	}

	outcome, known := swarmEpisodeIndexOutcome(record)
	standing, reason := swarmEpisodeStanding(record)

	return colonyEpisodeEntry{
		Kind:           colonyEpisodeKindSwarm,
		Subject:        emptyFallback(strings.TrimSpace(record.Target), record.SwarmID),
		Outcome:        outcome,
		OutcomeKnown:   known,
		Standing:       standing,
		StandingReason: reason,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Path:           relPath,
	}
}

// swarmEpisodeIndexOutcome names, in plain words, what a Swarm run ended
// with. An interrupted run names the stage it stopped at; a completed run
// names its verification outcome, falling back to its own recorded status.
// known is false only when the record carries neither a verification
// outcome nor a status -- buildSwarmEpisodeRecord never actually leaves
// both empty, but a malformed or hand-written record might.
func swarmEpisodeIndexOutcome(record swarmEpisodeRecord) (string, bool) {
	if record.Status == swarmEpisodeStatusInterrupted {
		stage := emptyFallback(strings.TrimSpace(record.InterruptedStage), "an unrecorded stage")
		return "interrupted during " + stage, true
	}
	status := strings.TrimSpace(record.VerificationStatus)
	if status == "" {
		status = strings.TrimSpace(record.Status)
	}
	if status == "" {
		return "", false
	}
	return status, true
}

// --- Source 2: Oracle research -------------------------------------------

// loadOracleResearchIndexEntries reads every saved research document
// (.aether/research/*.md), resolving each one's standing through the exact
// same oracleResearchStanding function the research listing already uses
// (cmd/oracle_research_doc.go) -- never a second opinion. available is
// false only when the research directory itself does not exist.
func loadOracleResearchIndexEntries(root string) ([]colonyEpisodeEntry, bool) {
	dir := oracleResearchDir(root)
	if info, statErr := os.Stat(dir); statErr != nil || !info.IsDir() {
		return nil, false
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, false
	}
	sort.Strings(matches)

	var out []colonyEpisodeEntry
	for _, match := range matches {
		relPath, relErr := filepath.Rel(root, match)
		if relErr != nil {
			relPath = match
		}
		relPath = filepath.ToSlash(relPath)

		data, readErr := os.ReadFile(match)
		if readErr != nil {
			standing, reason := resolveMalformedItemStanding("the document could not be read: " + readErr.Error())
			out = append(out, colonyEpisodeEntry{
				Kind: colonyEpisodeKindOracleResearch, Subject: filepath.Base(match),
				Standing: standing, StandingReason: reason, Path: relPath,
			})
			continue
		}

		text := string(data)
		if !strings.HasPrefix(text, "---\n") {
			standing, reason := resolveMalformedItemStanding("the document has no readable front matter")
			out = append(out, colonyEpisodeEntry{
				Kind: colonyEpisodeKindOracleResearch, Subject: filepath.Base(match),
				Standing: standing, StandingReason: reason, Path: relPath,
			})
			continue
		}

		entry := parseOracleResearchFrontMatter(text)
		standing := oracleResearchStanding(entry.Status)
		reason := ""
		if standing != workStandingVerified {
			reason = "finishing the remaining research rounds and reaching a clean completion"
		}

		var recordedAt time.Time
		if t, perr := time.Parse(time.RFC3339, entry.Generated); perr == nil {
			recordedAt = t
		}
		status := strings.TrimSpace(entry.Status)

		out = append(out, colonyEpisodeEntry{
			Kind:           colonyEpisodeKindOracleResearch,
			Subject:        emptyFallback(entry.CoreQuestion, entry.Title),
			Outcome:        status,
			OutcomeKnown:   status != "",
			Standing:       standing,
			StandingReason: reason,
			EndedAt:        recordedAt,
			Path:           relPath,
		})
	}
	return out, true
}

// --- Source 3: build and check attempts -----------------------------------

// loadBuildAttemptIndexEntries reads every recorded build attempt across
// every phase (.aether/data/build/phase-*/attempts/*.json) through the
// already-initialized store, mirroring listBuildAttemptsForPhase's own
// filename discipline (cmd/build_attempt.go) but scanning every phase
// directory rather than requiring one phase number. available is false
// only when the build directory itself does not exist or cannot be listed.
func loadBuildAttemptIndexEntries(s *storage.Store) ([]colonyEpisodeEntry, bool) {
	if s == nil {
		return nil, false
	}
	buildDir := filepath.Join(s.BasePath(), "build")
	phaseDirEntries, err := os.ReadDir(buildDir)
	if err != nil {
		return nil, false
	}

	phaseDirs := make([]string, 0, len(phaseDirEntries))
	for _, entry := range phaseDirEntries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "phase-") {
			phaseDirs = append(phaseDirs, entry.Name())
		}
	}
	sort.Strings(phaseDirs)

	var out []colonyEpisodeEntry
	for _, phaseDir := range phaseDirs {
		attemptsDir := filepath.Join(buildDir, phaseDir, "attempts")
		attemptFiles, readErr := os.ReadDir(attemptsDir)
		if readErr != nil {
			continue
		}

		names := make([]string, 0, len(attemptFiles))
		for _, f := range attemptFiles {
			name := f.Name()
			if f.IsDir() || !strings.HasSuffix(name, ".json") ||
				strings.HasSuffix(name, ".completion.json") ||
				strings.HasSuffix(name, ".start-receipt.json") {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			rel := filepath.ToSlash(filepath.Join("build", phaseDir, "attempts", name))
			var record buildAttemptRecord
			if loadErr := s.LoadJSON(rel, &record); loadErr != nil {
				standing, reason := resolveMalformedItemStanding("the attempt record could not be read: " + loadErr.Error())
				out = append(out, colonyEpisodeEntry{
					Kind: colonyEpisodeKindBuildAttempt, Subject: name,
					Standing: standing, StandingReason: reason, Path: rel,
				})
				continue
			}
			out = append(out, buildAttemptIndexEntryFrom(record, rel))
		}
	}
	return out, true
}

func buildAttemptIndexEntryFrom(record buildAttemptRecord, rel string) colonyEpisodeEntry {
	kind := colonyEpisodeKindBuildAttempt
	subject := emptyFallback(strings.TrimSpace(record.PhaseName), fmt.Sprintf("Phase %d build", record.Phase))
	outcome := strings.TrimSpace(record.Status)

	if record.CheckFix != nil {
		kind = colonyEpisodeKindCheckAttempt
		subject = emptyFallback(strings.TrimSpace(record.CheckFix.Reason), fmt.Sprintf("Phase %d check", record.Phase))
		if fixOutcome := strings.TrimSpace(record.CheckFix.Outcome); fixOutcome != "" {
			outcome = fixOutcome
		}
	}

	standing, reason := buildAttemptIndexStanding(record)

	var startedAt, endedAt time.Time
	if t, err := time.Parse(time.RFC3339Nano, record.StartedAt); err == nil {
		startedAt = t
	}
	endedRaw := record.CompletedAt
	if endedRaw == "" {
		endedRaw = record.UpdatedAt
	}
	if t, err := time.Parse(time.RFC3339Nano, endedRaw); err == nil {
		endedAt = t
	}

	return colonyEpisodeEntry{
		Kind:           kind,
		Subject:        subject,
		Outcome:        outcome,
		OutcomeKnown:   outcome != "",
		Standing:       standing,
		StandingReason: reason,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Cost:           colonyEpisodeCostRef{Phase: record.Phase},
		Path:           rel,
	}
}

// buildAttemptIndexStanding resolves a build or check attempt's standing
// through the shared vocabulary. A check-fix attempt's own recorded
// outcome (fixed/still_failing, checkFixAttemptRecord.Outcome) takes
// precedence when present -- it is the more specific verdict. A record
// with no recorded status at all resolves to standing unknown, never
// verified: a record this repository could not read a status from is never
// presented as a checked result.
func buildAttemptIndexStanding(record buildAttemptRecord) (workStanding, string) {
	if record.CheckFix != nil {
		switch strings.TrimSpace(record.CheckFix.Outcome) {
		case "fixed":
			return workStandingVerified, ""
		case "still_failing":
			return workStandingUsefulNotes, "the fix actually resolving the failing check instead of leaving it failing"
		}
	}
	switch strings.TrimSpace(record.Status) {
	case "":
		return resolveMalformedItemStanding("the attempt has no recorded status")
	case buildAttemptBuilt:
		return workStandingVerified, ""
	case buildAttemptFailed:
		return workStandingUsefulNotes, "the build finishing and passing its own checks instead of failing"
	case buildAttemptInterrupted:
		return workStandingUsefulNotes, "the build finishing past where it was interrupted"
	case buildAttemptPartial:
		return workStandingUsefulNotes, "the remaining tasks finishing and being credited"
	default:
		return workStandingUsefulNotes, "the attempt finishing and reaching a recorded outcome"
	}
}

// --- Source 4: quick jobs ---------------------------------------------------

// loadQuickAttemptIndexEntries reads every durable quick-attempt record
// (.aether/data/quick/attempts/*.json), the exact file persistQuickAttempt
// writes (cmd/command_truth.go) -- a pure read, mirroring
// loadBuildAttemptIndexEntries' own discipline. available is false only
// when the quick/attempts directory itself does not exist or cannot be
// listed -- a colony that has simply never run a quick job.
func loadQuickAttemptIndexEntries(s *storage.Store) ([]colonyEpisodeEntry, bool) {
	if s == nil {
		return nil, false
	}
	dir := filepath.Join(s.BasePath(), "quick", "attempts")
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}

	names := make([]string, 0, len(dirEntries))
	for _, entry := range dirEntries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	var out []colonyEpisodeEntry
	for _, name := range names {
		rel := filepath.ToSlash(filepath.Join("quick", "attempts", name))
		var record quickAttemptRecord
		if loadErr := s.LoadJSON(rel, &record); loadErr != nil {
			standing, reason := resolveMalformedItemStanding("the quick job record could not be read: " + loadErr.Error())
			out = append(out, colonyEpisodeEntry{
				Kind: colonyEpisodeKindQuick, Subject: name,
				Standing: standing, StandingReason: reason, Path: rel,
			})
			continue
		}
		out = append(out, quickAttemptIndexEntryFrom(record, rel))
	}
	return out, true
}

func quickAttemptIndexEntryFrom(record quickAttemptRecord, rel string) colonyEpisodeEntry {
	var startedAt, endedAt time.Time
	if t, err := time.Parse(time.RFC3339Nano, record.StartedAt); err == nil {
		startedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, record.CompletedAt); err == nil {
		endedAt = t
	}

	outcome := string(record.Verdict)
	known := record.Verdict.Valid()
	standing, reason := quickAttemptIndexStanding(record.Verdict)

	return colonyEpisodeEntry{
		Kind:           colonyEpisodeKindQuick,
		Subject:        emptyFallback(strings.TrimSpace(record.Question), "a quick job"),
		Outcome:        outcome,
		OutcomeKnown:   known,
		Standing:       standing,
		StandingReason: reason,
		StartedAt:      startedAt,
		EndedAt:        endedAt,
		Path:           rel,
	}
}

// quickAttemptIndexStanding resolves a quick attempt's standing through the
// shared vocabulary: success or no-change is verified, a blocker/timeout/
// interrupted run has useful notes toward a passing check, and a partial
// verdict (files changed but nothing could be checked) is useful notes
// toward the checks actually running -- never presented as verified when
// the change itself was never confirmed.
func quickAttemptIndexStanding(v colony.WorkOutcome) (workStanding, string) {
	switch v {
	case colony.WorkOutcomeSuccess, colony.WorkOutcomeNoChange:
		return workStandingVerified, ""
	case colony.WorkOutcomeBlocker, colony.WorkOutcomeTimeout, colony.WorkOutcomeInterrupted:
		return workStandingUsefulNotes, "the change passing the project's own checks instead of failing them"
	case colony.WorkOutcomePartial:
		return workStandingUsefulNotes, "the project's checks actually running so the change can be confirmed"
	default:
		return resolveMalformedItemStanding("the quick job has no recorded outcome")
	}
}

// --- Shared cost rendering -------------------------------------------------

// colonyEpisodeLedgers resolves an entry's cost reference to the actual
// ledgers it points at, through the one existing spend authority
// (loadSpendLedgersForPhase, cmd/spend_ledger.go) -- never a second
// accounting path. A zero phase (Swarm and Oracle research today) resolves
// to no ledgers, which the spend renderers already treat as "not reported".
func colonyEpisodeLedgers(ref colonyEpisodeCostRef) []spendLedger {
	if ref.Phase <= 0 {
		return nil
	}
	ledgers, _ := loadSpendLedgersForPhase(ref.Phase)
	return ledgers
}
