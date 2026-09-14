package cmd

// LEARN-02 (204-04-PLAN.md): the durable, append-only episode and outcome
// ledger. The one genuine typed event bus this system has (pkg/events.Bus)
// forgets anything older than events.DefaultTTL (30 days) on every read --
// correct for a live activity feed, fatal for the record a promotion or
// rollback decision may need to consult weeks or months later. This file is
// the permanent record kept BESIDE that live feed, never a second bus and
// never a second event-shaped type reaching durable storage (204-CLASSIC-
// SYNTHESIS.md ruling (f), SYN-204-04) -- pkg/events.Bus continues to carry
// liveness only (cmd/live_events.go emits a live/v1 topic for every episode
// boundary this file also records durably); this file's own record type
// carries no field whose Go name or JSON tag matches "timestamp" together
// with a kind/type-shaped field, and its only writer never performs a raw
// os.WriteFile/OpenFile/Create -- every write goes through
// store.UpdateJSONAtomically, exactly like the file it follows in shape,
// cmd/recruitment_credit.go's recordRecruitmentCredit.
//
// Schema and lineage (204-03 shared contract): this ledger's file-level
// schema version is the shared colony.CurrentMemorySchemaVersion, and every
// record it writes carries the shared per-record SchemaVersion and
// colony.MemoryRecordLineage that the other live memory stores carry --
// stamped inside recordEpisodeOutcome, the ledger's one writer, so no call
// site can produce a record without them. 204-04 first shipped a local
// stand-in for both because 204-03 landed in a sibling worktree; the
// stand-in was reconciled onto the shared symbols when the two merged.
import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/shadow"
)

// episodeLedgerPath is the store-relative path recordEpisodeOutcome
// persists to. A directory of its own, mirroring credit/records.json's own
// one-store-one-directory convention.
const episodeLedgerPath = "episodes/ledger.json"

// episodeLedgerSchemaVersion is the ledger's current schema version -- the
// shared constant every live memory store declares against (204-03), never
// a private number that could drift from the rest of memory.
const episodeLedgerSchemaVersion = colony.CurrentMemorySchemaVersion

// episodeLedgerRecordKind is the declared, closed vocabulary of every kind
// of fact this ledger may record.
type episodeLedgerRecordKind string

const (
	episodeLedgerRecordKindOpened       episodeLedgerRecordKind = "episode_opened"
	episodeLedgerRecordKindClosed       episodeLedgerRecordKind = "episode_closed"
	episodeLedgerRecordKindIntervention episodeLedgerRecordKind = "intervention_recorded"
)

// episodeLedgerRecordKindVocabulary is the declared, closed set of every
// record kind -- same completeness convention as
// recruitmentCreditOutcomeVocabulary/ColonyLiveTopics(): a kind added to the
// const block above must also be added here.
var episodeLedgerRecordKindVocabulary = []episodeLedgerRecordKind{
	episodeLedgerRecordKindOpened,
	episodeLedgerRecordKindClosed,
	episodeLedgerRecordKindIntervention,
}

func episodeLedgerRecordKindNames() []string {
	names := make([]string, 0, len(episodeLedgerRecordKindVocabulary))
	for _, k := range episodeLedgerRecordKindVocabulary {
		names = append(names, string(k))
	}
	return names
}

func episodeLedgerRecordKindDeclared(kind episodeLedgerRecordKind) bool {
	for _, k := range episodeLedgerRecordKindVocabulary {
		if k == kind {
			return true
		}
	}
	return false
}

// episodeInterventionKind (204-13, SC3a/SC3b, D-10) is the declared, closed
// vocabulary of every owner intervention this program can actually observe
// and record today -- the identical shape episodeLedgerRecordKind above
// already uses. Before this type existed, episodeLedgerRecord.Interventions
// was a free-form string slice with no declared vocabulary and no
// production writer at all, so collectPreventableInterventions
// (cmd/improvement_report.go) would have treated every distinct sentence as
// its own category the moment something DID write one. Add a member here
// ONLY when a real production call site exists to write it --
// TestEveryInterventionKindHasALiveWriter fails a declared member with no
// writer by name, which is deliberate: a speculative member with no writer
// is exactly the failure mode this vocabulary exists to prevent.
type episodeInterventionKind string

const (
	// episodeInterventionKindAnsweredWorkerQuestion is recorded when the
	// owner answers a worker's open decision through recordDecisionAnswer
	// (cmd/handoff_decisions_cmd.go) -- excluding the seal ceremony's own
	// internal "finish anyway?" confirmation gate, which is not a worker's
	// question.
	episodeInterventionKindAnsweredWorkerQuestion episodeInterventionKind = "answered a worker's question"
	// episodeInterventionKindDeclinedForcedReviewer is recorded when the
	// owner declines a forced reviewer through
	// resolveForcedReviewerWaiverPendingDecision succeeding
	// (cmd/forced_reviewer_waiver.go) -- the single site that fires exactly
	// once per real, genuine waiver.
	episodeInterventionKindDeclinedForcedReviewer episodeInterventionKind = "declined a forced reviewer"
	// episodeInterventionKindReleasedQuarantine is recorded when the owner
	// releases a quarantined canary candidate through a genuine
	// Quarantined-true-to-false transition inside releaseCanaryQuarantine
	// (cmd/rollback.go) -- never for a release call that finds the
	// candidate already unquarantined (a no-op, not an intervention).
	episodeInterventionKindReleasedQuarantine episodeInterventionKind = "released a quarantined candidate"
)

// episodeInterventionKindVocabulary is the declared, closed set of every
// intervention kind. Mirrors the episodeLedgerRecordKindVocabulary /
// recruitmentCreditOutcomeVocabulary completeness convention: a kind added
// to the const block above must also be added here, or
// TestInterventionKindVocabularyIsClosed fails.
var episodeInterventionKindVocabulary = []episodeInterventionKind{
	episodeInterventionKindAnsweredWorkerQuestion,
	episodeInterventionKindDeclinedForcedReviewer,
	episodeInterventionKindReleasedQuarantine,
}

func episodeInterventionKindNames() []string {
	names := make([]string, 0, len(episodeInterventionKindVocabulary))
	for _, k := range episodeInterventionKindVocabulary {
		names = append(names, string(k))
	}
	return names
}

func episodeInterventionKindDeclared(kind episodeInterventionKind) bool {
	for _, k := range episodeInterventionKindVocabulary {
		if k == kind {
			return true
		}
	}
	return false
}

// episodeLedgerRecord is one durable fact about one episode: its identity,
// what governed it (runtime/policy versions, acceptance and evaluator
// digests), what it touched (evidence, hard gates, changed decisions), how
// long it took and what it cost, and how it ended. Every optional field
// carries omitempty so an unmeasured or inapplicable fact round-trips as
// absent, never as a zero value that looks deliberate -- Usage and
// ReportedCostUSD are pointers precisely so an unreported figure is nil,
// distinct from a real, provider-reported zero.
type episodeLedgerRecord struct {
	RecordID           string                  `json:"record_id"`
	RecordKind         episodeLedgerRecordKind `json:"record_kind"`
	EpisodeID          string                  `json:"episode_id"`
	EpisodeKind        string                  `json:"episode_kind,omitempty"`
	EpisodeRevision    string                  `json:"episode_revision,omitempty"`
	RuntimeVersion     string                  `json:"runtime_version,omitempty"`
	PolicyVersion      string                  `json:"policy_version,omitempty"`
	AcceptanceDigest   string                  `json:"acceptance_digest,omitempty"`
	EvaluatorDigest    string                  `json:"evaluator_digest,omitempty"`
	EvidenceIDs        []string                `json:"evidence_ids,omitempty"`
	HardGateResults    map[string]bool         `json:"hard_gate_results,omitempty"`
	ChangedDecisionIDs []string                `json:"changed_decision_ids,omitempty"`
	// Interventions stays a []string on disk (JSON-compatible, never the
	// episodeInterventionKind type itself), but every element written by
	// the one production writer, emitColonyLiveInterventionRecorded, is a
	// declared episodeInterventionKind's string form -- validated before
	// the write, never after. No legacy record on disk carries a free-form
	// intervention string: before 204-13, emitColonyLiveInterventionRecorded
	// had zero production callers (confirmed by grep across cmd/), so no
	// migration of existing data is required or written here.
	Interventions   []string           `json:"interventions,omitempty"`
	StartedAt       string             `json:"started_at,omitempty"`
	EndedAt         string             `json:"ended_at,omitempty"`
	ElapsedSeconds  float64            `json:"elapsed_seconds,omitempty"`
	Usage           *codex.WorkerUsage `json:"usage,omitempty"`
	ReportedCostUSD *float64           `json:"reported_cost_usd,omitempty"`
	TerminalResult  string             `json:"terminal_result,omitempty"`
	// SchemaVersion and Lineage are the shared per-record contract every
	// live memory store carries (204-03). Both are stamped by
	// recordEpisodeOutcome on every write and excluded from the record's
	// replay-identity digest, so a replayed open or close still collapses
	// to the same record id.
	SchemaVersion int                         `json:"schema_version,omitempty"`
	Lineage       *colony.MemoryRecordLineage `json:"lineage,omitempty"`
}

// episodeLedgerFile is the on-disk container at episodeLedgerPath.
type episodeLedgerFile struct {
	SchemaVersion int                   `json:"schema_version"`
	Entries       []episodeLedgerRecord `json:"entries"`
}

// episodeLedgerDigestPayload derives the content used for a record's
// identity. For episode_opened and episode_closed kinds this deliberately
// excludes every timestamp-bearing field (StartedAt, EndedAt,
// ElapsedSeconds) so a genuine replay -- the same episode's open or close
// recorded a second time, at a different wall-clock moment -- collapses to
// the SAME record id and writes nothing. For intervention_recorded, the
// timestamp IS part of the identity: two categorically-identical
// interventions genuinely happening at two different moments are two
// distinct facts, never a replay of one.
func episodeLedgerDigestPayload(record episodeLedgerRecord) string {
	digestSource := record
	digestSource.RecordID = ""
	// Schema version and lineage are bookkeeping about the record, not the
	// fact it records: a replay stamped a moment later must still collapse.
	digestSource.SchemaVersion = 0
	digestSource.Lineage = nil
	if record.RecordKind != episodeLedgerRecordKindIntervention {
		digestSource.StartedAt = ""
		digestSource.EndedAt = ""
		digestSource.ElapsedSeconds = 0
	}
	encoded, err := json.Marshal(digestSource)
	if err != nil {
		// A plain struct built entirely of JSON-safe field types never
		// fails to marshal in practice; fall back to the episode id alone
		// rather than panicking on an unreachable path.
		return record.EpisodeID
	}
	return string(encoded)
}

// episodeLedgerRecordID is the deterministic identity key a ledger record
// is stored and replayed under, digesting with crypto/sha256 the way this
// repository already derives content-addressed identities (mirroring
// recruitmentCreditRecordID's role, cmd/recruitment_credit.go). Two facts
// identical in episode, kind and payload digest are one record.
func episodeLedgerRecordID(episodeID string, kind episodeLedgerRecordKind, payloadDigest string) string {
	sum := sha256.Sum256([]byte(episodeID + "|" + string(kind) + "|" + payloadDigest))
	return fmt.Sprintf("episode:%x", sum)
}

// errEpisodeLedgerNoChange is the internal replay sentinel
// recordEpisodeOutcome returns from its own UpdateJSONAtomically mutate
// closure to abort the write on a replay -- mirroring
// errRecruitmentCreditNoChange's role in recordRecruitmentCredit.
var errEpisodeLedgerNoChange = errors.New("episode ledger record already exists")

// recordEpisodeOutcome is the ONE function in cmd/ that writes
// episodes/ledger.json (TestEpisodeLedgerHasOneWriter enforces this by
// name). It refuses a record with no episode identifier by name; refuses a
// close for an episode with no open record by name; refuses a DIFFERENT
// close for an episode that already has one by name (WR-02, 204-REVIEW.md
// -- an identical replay of the same close still collapses silently);
// returns the stored record with a false credited flag on replay; and
// never mutates an existing record under any input.
func recordEpisodeOutcome(record episodeLedgerRecord) (episodeLedgerRecord, bool, error) {
	if store == nil {
		return episodeLedgerRecord{}, false, fmt.Errorf("no store initialized")
	}
	episodeID := strings.TrimSpace(record.EpisodeID)
	if episodeID == "" {
		return episodeLedgerRecord{}, false, fmt.Errorf("episode ledger requires a non-empty episode id")
	}
	record.EpisodeID = episodeID
	if !episodeLedgerRecordKindDeclared(record.RecordKind) {
		return episodeLedgerRecord{}, false, fmt.Errorf(
			"episode ledger record kind %q is not in the declared vocabulary %v", record.RecordKind, episodeLedgerRecordKindNames(),
		)
	}

	var file episodeLedgerFile
	var result episodeLedgerRecord
	updateErr := store.UpdateJSONAtomically(episodeLedgerPath, &file, func() error {
		file.SchemaVersion = episodeLedgerSchemaVersion
		if record.RecordKind == episodeLedgerRecordKindClosed {
			hasOpen := false
			for _, existing := range file.Entries {
				if existing.EpisodeID == episodeID && existing.RecordKind == episodeLedgerRecordKindOpened {
					hasOpen = true
					break
				}
			}
			if !hasOpen {
				return fmt.Errorf("episode ledger refuses to close episode %q: no open record exists for it", episodeID)
			}
		}

		recordID := episodeLedgerRecordID(episodeID, record.RecordKind, episodeLedgerDigestPayload(record))
		for _, existing := range file.Entries {
			if existing.RecordID == recordID {
				result = existing
				return errEpisodeLedgerNoChange
			}
		}

		// WR-02 (204-REVIEW.md): an identical replay of an existing close
		// (caught above, by its identical digest) still collapses silently
		// -- but a DIFFERENT close for an episode that already has one is
		// refused by name, rather than appending a second, distinct
		// episode_closed record that episodeLedgerTerminalRecord would then
		// silently pick between.
		if record.RecordKind == episodeLedgerRecordKindClosed {
			for _, existing := range file.Entries {
				if existing.EpisodeID == episodeID && existing.RecordKind == episodeLedgerRecordKindClosed {
					return fmt.Errorf(
						"episode ledger refuses a second close for episode %q: it is already closed with terminal result %q",
						episodeID, existing.TerminalResult,
					)
				}
			}
		}

		record.RecordID = recordID
		record.SchemaVersion = episodeLedgerSchemaVersion
		lineage := colony.NewMemoryRecordLineage(colony.MemoryProvenanceRuntime, episodeID, episodeLedgerRecordSortTimestamp(record))
		record.Lineage = &lineage
		file.Entries = append(file.Entries, record)
		result = record
		return nil
	})
	if updateErr != nil {
		if errors.Is(updateErr, errEpisodeLedgerNoChange) {
			return result, false, nil
		}
		return episodeLedgerRecord{}, false, updateErr
	}
	return result, true, nil
}

// episodeLedgerRecordSortTimestamp returns the timestamp readEpisodeLedger
// orders by: a record's own ended time when it has one, otherwise its
// started time -- so a still-open episode's record sorts by when it began.
func episodeLedgerRecordSortTimestamp(record episodeLedgerRecord) string {
	if record.EndedAt != "" {
		return record.EndedAt
	}
	return record.StartedAt
}

// sortEpisodeLedgerRecords orders records by timestamp ascending, with the
// record identifier as the tie-break -- an explicit, deterministic
// tie-break so repeated reads of identical underlying data always produce
// byte-identical output.
func sortEpisodeLedgerRecords(records []episodeLedgerRecord) {
	sort.SliceStable(records, func(i, j int) bool {
		ti := episodeLedgerRecordSortTimestamp(records[i])
		tj := episodeLedgerRecordSortTimestamp(records[j])
		if ti != tj {
			return ti < tj
		}
		return records[i].RecordID < records[j].RecordID
	})
}

// readEpisodeLedger returns every stored ledger record, ordered by
// sortEpisodeLedgerRecords. Returns an empty slice (never an error) when
// nothing has ever been recorded.
func readEpisodeLedger() ([]episodeLedgerRecord, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	var file episodeLedgerFile
	if err := store.LoadJSON(episodeLedgerPath, &file); err != nil {
		return []episodeLedgerRecord{}, nil
	}
	records := append([]episodeLedgerRecord{}, file.Entries...)
	sortEpisodeLedgerRecords(records)
	return records, nil
}

// episodeLedgerForEpisode returns every record naming episodeID, in the
// same deterministic order readEpisodeLedger uses.
func episodeLedgerForEpisode(episodeID string) ([]episodeLedgerRecord, error) {
	episodeID = strings.TrimSpace(episodeID)
	if episodeID == "" {
		return []episodeLedgerRecord{}, nil
	}
	all, err := readEpisodeLedger()
	if err != nil {
		return nil, err
	}
	var matches []episodeLedgerRecord
	for _, r := range all {
		if r.EpisodeID == episodeID {
			matches = append(matches, r)
		}
	}
	if matches == nil {
		matches = []episodeLedgerRecord{}
	}
	return matches, nil
}

// episodeLedgerTerminalRecord returns episodeID's own episode_closed record,
// if one has been written, and ok=false otherwise -- an episode with an
// open record and no terminal record is unfinished, never rendered as a
// success and never as absent.
func episodeLedgerTerminalRecord(records []episodeLedgerRecord, episodeID string) (episodeLedgerRecord, bool) {
	for _, r := range records {
		if r.EpisodeID == episodeID && r.RecordKind == episodeLedgerRecordKindClosed {
			return r, true
		}
	}
	return episodeLedgerRecord{}, false
}

// episodeLedgerOpenRecord returns episodeID's own episode_opened record, if
// one has been written, and ok=false otherwise.
func episodeLedgerOpenRecord(records []episodeLedgerRecord, episodeID string) (episodeLedgerRecord, bool) {
	for _, r := range records {
		if r.EpisodeID == episodeID && r.RecordKind == episodeLedgerRecordKindOpened {
			return r, true
		}
	}
	return episodeLedgerRecord{}, false
}

// episodeLedgerEpisodeIDs returns the distinct set of episode ids present
// across records, in first-seen order over records' own already-sorted
// (timestamp-ascending) order -- a stable, deterministic enumeration for
// the derived views below.
func episodeLedgerEpisodeIDs(records []episodeLedgerRecord) []string {
	seen := map[string]bool{}
	var ids []string
	for _, r := range records {
		if seen[r.EpisodeID] {
			continue
		}
		seen[r.EpisodeID] = true
		ids = append(ids, r.EpisodeID)
	}
	return ids
}

// ---------------------------------------------------------------------
// LEARN-02 (204-04-PLAN.md, Task 3): derived, human-readable views over the
// ledger. Each function below is pure over a slice of already-read records
// -- no store read, no clock read -- so the same input always produces the
// same output (TestDerivedViewsAreIdempotent) and neither view can ever
// become an authority the durable record itself is not: the append-only
// record is the authority, every view here is derived from it.
// ---------------------------------------------------------------------

// episodeChangelogEntry is the derived successor to
// changelogCollectPlanDataCmd's ad-hoc output (cmd/changelog.go) -- that
// command has no named entry type today; this declares exactly the field
// set it already emits (date, phase, plan, entry text), so a later caller
// can route changelog collection through the ledger without inventing a
// different shape.
type episodeChangelogEntry struct {
	Date  string `json:"date"`
	Phase string `json:"phase"`
	Plan  string `json:"plan"`
	Entry string `json:"entry"`
}

// episodeSpendSummary totals what the runs in a set of records reported,
// and separately counts the runs that reported nothing -- so a summary
// including unreported runs states how many it could not account for,
// rather than silently treating an unreported figure as zero.
type episodeSpendSummary struct {
	AccountedEpisodes   int
	UnaccountedEpisodes int
	TotalUSDCost        float64
	TotalTokens         int64
}

// episodeOutcomeWording translates a terminal-result value (an internal
// vocabulary token such as "helpful"/"harmful"/"neutral"/"pending", or a
// free-form episode terminal status such as "completed"/"failed") into an
// ordinary sentence fragment -- so no rendered line ever prints the raw
// token verbatim (TestDerivedViewsSpeakTheSharedVoice, TestVoicedScreens
// CarryNoRawStateToken's own discipline, applied here to a screen outside
// the voice corpus by the same rule).
func episodeOutcomeWording(terminalResult string) string {
	switch terminalResult {
	case string(recruitmentCreditOutcomeHelpful):
		return "helped"
	case string(recruitmentCreditOutcomeNeutral):
		return "made no measurable difference"
	case string(recruitmentCreditOutcomeHarmful):
		return "made things worse"
	case string(recruitmentCreditOutcomePending):
		return "outcome not yet verified"
	case "":
		return "no outcome recorded"
	default:
		return strings.ReplaceAll(terminalResult, "_", " ")
	}
}

// renderEpisodeOutcomeSummary is the human-readable phase outcome the
// capability ledger (CAP-067/CAP-070) routes here as a derived view rather
// than a separately written file -- one row per episode, with its kind, its
// terminal result, its elapsed time and its cost, and an explicit no-
// outcome row for an episode with no terminal record, never a success and
// never an omission.
func renderEpisodeOutcomeSummary(records []episodeLedgerRecord) string {
	if len(records) == 0 {
		return voiceLine("status", "No episodes have been recorded yet.") + "\n"
	}
	var sb strings.Builder
	sb.WriteString(voiceLine("status", "Episode outcomes") + "\n")
	for _, episodeID := range episodeLedgerEpisodeIDs(records) {
		open, hasOpen := episodeLedgerOpenRecord(records, episodeID)
		kind := "an unknown kind of run"
		if hasOpen && strings.TrimSpace(open.EpisodeKind) != "" {
			kind = open.EpisodeKind
		}
		terminal, hasTerminal := episodeLedgerTerminalRecord(records, episodeID)
		if !hasTerminal {
			sb.WriteString(voiceLine("blocked", fmt.Sprintf("Episode %s (%s): no outcome recorded yet.", episodeID, kind)) + "\n")
			continue
		}
		elapsed := "elapsed time not recorded"
		if terminal.ElapsedSeconds > 0 {
			elapsed = fmt.Sprintf("took %.0fs", terminal.ElapsedSeconds)
		}
		cost := "cost not reported"
		if terminal.ReportedCostUSD != nil {
			cost = fmt.Sprintf("cost $%.4f", *terminal.ReportedCostUSD)
		}
		glyphKind := "done"
		if terminal.TerminalResult == string(recruitmentCreditOutcomeHarmful) {
			glyphKind = "failed"
		}
		sb.WriteString(voiceLine(glyphKind, fmt.Sprintf(
			"Episode %s (%s): %s, %s, %s.",
			episodeID, kind, episodeOutcomeWording(terminal.TerminalResult), elapsed, cost,
		)) + "\n")
	}
	return sb.String()
}

// collectChangelogEntriesFromLedger is the derived successor to
// changelogCollectPlanDataCmd's ad-hoc output -- the changelog collection
// the capability ledger (CAP-067) likewise routes here as a derived view.
// Only closed episodes carrying enough identity to name a phase and plan
// (EpisodeRevision, following this repo's "phase-plan" episode-revision
// convention) produce an entry; anything else is skipped rather than
// guessed.
func collectChangelogEntriesFromLedger(records []episodeLedgerRecord) []episodeChangelogEntry {
	var entries []episodeChangelogEntry
	for _, episodeID := range episodeLedgerEpisodeIDs(records) {
		terminal, ok := episodeLedgerTerminalRecord(records, episodeID)
		if !ok {
			continue
		}
		phase, plan := episodeLedgerRevisionPhaseAndPlan(terminal.EpisodeRevision)
		if phase == "" && plan == "" {
			continue
		}
		date := terminal.EndedAt
		if date == "" {
			date = terminal.StartedAt
		}
		entries = append(entries, episodeChangelogEntry{
			Date:  date,
			Phase: phase,
			Plan:  plan,
			Entry: episodeOutcomeWording(terminal.TerminalResult),
		})
	}
	if entries == nil {
		entries = []episodeChangelogEntry{}
	}
	return entries
}

// episodeLedgerRevisionPhaseAndPlan splits an EpisodeRevision value of the
// form "phase-plan" (this repo's own phase/plan naming convention, e.g.
// "204-04") into its two halves. A revision carrying no separator, or an
// empty revision, returns two empty strings -- collectChangelogEntriesFrom
// Ledger skips such an episode rather than guessing a phase/plan identity
// that was never recorded.
func episodeLedgerRevisionPhaseAndPlan(revision string) (phase, plan string) {
	revision = strings.TrimSpace(revision)
	if revision == "" {
		return "", ""
	}
	parts := strings.SplitN(revision, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// summariseEpisodeSpend totals what every closed episode in records
// reported, and separately counts the ones that reported nothing -- a run
// with a nil Usage pointer increments the unaccounted count and
// contributes nothing to the totals, never a silent zero.
func summariseEpisodeSpend(records []episodeLedgerRecord) episodeSpendSummary {
	var summary episodeSpendSummary
	for _, episodeID := range episodeLedgerEpisodeIDs(records) {
		terminal, ok := episodeLedgerTerminalRecord(records, episodeID)
		if !ok {
			continue
		}
		reported := false
		if terminal.Usage != nil && !terminal.Usage.Empty() {
			summary.TotalTokens += terminal.Usage.TotalTokens
			reported = true
		}
		if terminal.ReportedCostUSD != nil {
			summary.TotalUSDCost += *terminal.ReportedCostUSD
			reported = true
		}
		if reported {
			summary.AccountedEpisodes++
		} else {
			summary.UnaccountedEpisodes++
		}
	}
	return summary
}

// ---------------------------------------------------------------------
// 204-15 (SC3a): the nine previously writerless episode fields. Every
// function below reads a fact a lane's own boundary already produced --
// this phase's latest durable build attempt, the per-phase gate/
// verification reports the check lanes already write to disk, and the
// process-wide shadow acceptance/evaluator definitions -- rather than
// threading a value through the lane's own long call chain. Reading fresh
// at close time is what lets a single already-registered `defer`, set up
// before any of these facts exist, still assemble the full record once the
// lane actually returns.
// ---------------------------------------------------------------------

// buildEpisodeGateResultNames is the closed set of check names the direct
// build lane's own pre-dispatch blocker advisory (buildStartBlockerSignals,
// cmd/build_blocker_advisory.go) can ever report. The direct build lane
// (runCodexBuildWithOptions) never runs the program's own build/types/lint/
// tests floor itself -- that only runs later, at build-finalize time
// (cmd/codex_build_finalize.go's buildFreeCheckReportFromFloor), a
// SEPARATE lane this plan does not touch -- so this advisory, already
// computed once per direct build before dispatch, is the one real,
// always-available check set this lane's own episode close has to report.
// Documented as a deliberate design choice in 204-15-SUMMARY.md.
var buildEpisodeGateResultNames = []string{"forced-reviewer", "unanswered-question", "last-continue-blocked"}

// buildEpisodeGateResults converts advisory's own already-computed signals
// into a name->passed map covering every name in
// buildEpisodeGateResultNames -- true (clear) unless that signal is
// actually present in advisory.Signals. Always returns a non-empty map
// (the direct build lane always evaluates this advisory before dispatch),
// so this never produces the "fabricated empty map" isVerifiedUsefulSuccess
// would misread as evidence of failure.
func buildEpisodeGateResults(advisory buildBlockerAdvisory) map[string]bool {
	present := make(map[string]bool, len(advisory.Signals))
	for _, s := range advisory.Signals {
		present[s.Name] = true
	}
	results := make(map[string]bool, len(buildEpisodeGateResultNames))
	for _, name := range buildEpisodeGateResultNames {
		results[name] = !present[name]
	}
	return results
}

// buildEpisodeApplicationFacts re-reads phaseNum's own latest durable build
// attempt -- written earlier in the SAME run this function's caller closes,
// before the deferred close reads it -- for the evidence and changed-
// decision identifiers, reusing phaseApplicationEffectEvidenceID /
// phaseApplicationDecisionID's exact derivation shape
// (cmd/application_evidence.go) rather than inventing a second identifier
// scheme. EpisodeRevision is phaseNum joined with the attempt's own ID in
// the "phase-plan" shape episodeLedgerRevisionPhaseAndPlan already parses
// (SplitN on the FIRST "-" keeps the attempt ID, which itself contains
// "-", intact as the second half). ok=false when no durable attempt has
// been recorded for this phase yet -- the earliest failure-return paths in
// both the build and the check lanes -- in which case the caller must
// leave EvidenceIDs, ChangedDecisionIDs and EpisodeRevision absent.
func buildEpisodeApplicationFacts(phaseNum int) (evidenceIDs []string, changedDecisionIDs []string, revision string, ok bool) {
	_, attempt, found := loadLatestBuildAttempt(phaseNum)
	if !found || strings.TrimSpace(attempt.ID) == "" {
		return nil, nil, "", false
	}
	evidenceIDs = []string{phaseApplicationEffectEvidenceID(attempt.ID)}
	idx := 0
	for _, delta := range attempt.KnowledgeDeltas {
		if delta.Kind != buildKnowledgeDeltaKindDecision {
			continue
		}
		changedDecisionIDs = append(changedDecisionIDs, phaseApplicationDecisionID(attempt.ID, idx))
		idx++
	}
	revision = fmt.Sprintf("%d-%s", phaseNum, attempt.ID)
	return evidenceIDs, changedDecisionIDs, revision, true
}

// shadowAcceptanceDigestHex and shadowEvaluatorDigestHex are the hex-string
// forms of shadowAcceptanceCriteriaDefinition's and shadowEvaluator()'s own
// [32]byte digests (cmd/shadow_cmds.go) -- the acceptance criteria and the
// grader currently in force for LEARN-06's canary comparisons, process-wide
// and constant, never a per-lane value. A CHECK is the moment this program
// verifies work against acceptance criteria, so both check lanes record
// these; the direct build lane leaves them absent (it dispatches workers,
// it does not itself grade anything against a frozen evaluator).
func shadowAcceptanceDigestHex() string {
	criteria := shadow.NewAcceptanceCriteria([]byte(shadowAcceptanceCriteriaDefinition))
	digest := criteria.Digest()
	return fmt.Sprintf("%x", digest)
}

func shadowEvaluatorDigestHex() string {
	digest := shadowEvaluator().Digest()
	return fmt.Sprintf("%x", digest)
}

// checkEpisodeCloseRecord (204-15, SC3a) assembles the durable
// episodeLedgerRecord shared by BOTH check lanes -- runCodexContinue
// (cmd/codex_continue.go) and runCodexContinueFinalize
// (cmd/codex_continue_finalize.go) -- at their own episode close. One
// helper, two callers, so the native and delegate lanes cannot record
// different things for what should be the identical fact set (204-15-
// PLAN.md Task 1(b)). Every source here is a fresh, independent read (the
// per-phase gate report already written to disk by either lane, this
// phase's own latest durable build attempt, and the process-wide shadow
// definitions) rather than an in-process value threaded through the
// lane's own call chain -- exactly what lets this one helper work
// unmodified from either lane's deferred close, registered before any of
// these facts exist. HardGateResults is left nil (never a fabricated
// empty map) when no gate report has been written yet for this phase --
// an early-return path before verification ran.
func checkEpisodeCloseRecord(phaseID int, runStatus string) episodeLedgerRecord {
	record := episodeLedgerRecord{TerminalResult: runStatus}

	if store != nil {
		var gates codexContinueGateReport
		gateReportRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), "gates.json"))
		if err := store.LoadJSON(gateReportRel, &gates); err == nil && len(gates.Checks) > 0 {
			results := make(map[string]bool, len(gates.Checks))
			for _, c := range gates.Checks {
				results[c.Name] = c.Passed
			}
			record.HardGateResults = results
		}
	}

	if evidenceIDs, changedDecisionIDs, revision, ok := buildEpisodeApplicationFacts(phaseID); ok {
		record.EvidenceIDs = evidenceIDs
		record.ChangedDecisionIDs = changedDecisionIDs
		record.EpisodeRevision = revision
	}

	record.AcceptanceDigest = shadowAcceptanceDigestHex()
	record.EvaluatorDigest = shadowEvaluatorDigestHex()

	return record
}

// episodeCloseFacts (204-15, SC3a) carries the fields a lane's own later
// processing discovers, for a deferred close registered near the TOP of
// the lane -- before any of these facts exist -- to read once the lane
// actually returns. A pointer so the deferred closure sees every field the
// lane fills in along the way, regardless of how much code runs between
// registration and the point a fact becomes known: Go's declare-before-use
// scoping otherwise forbids an early closure from referencing a variable
// declared later in the same function body, and this is the one field
// (per-worker usage, never serialized -- codexBuildDispatch.Usage and
// codexContinueWorkerFlowStep.Usage both carry `json:"-"`) that cannot be
// recovered by a fresh disk read at close time the way the others above
// are.
type episodeCloseFacts struct {
	Usage           *codex.WorkerUsage
	ReportedCostUSD *float64
}

// addUsage folds one worker's own reported usage into f, following
// wrapperUsageWasReported's existing reported-vs-unreported rule
// (cmd/wrapper_usage_resolve.go) and WorkerUsage.Estimated's existing
// measured-vs-estimated rule: a usage with no source at all contributes
// nothing (T-204-15-02, never a fabricated zero); an estimated usage's
// token counts are still folded in (a genuine record of what was
// estimated, never fabricated) but its cost is never folded into
// ReportedCostUSD (T-204-15-01) -- an estimate must never be presentable
// as a measurement.
func (f *episodeCloseFacts) addUsage(usage codex.WorkerUsage) {
	if f == nil || !wrapperUsageWasReported(usage) {
		return
	}
	if f.Usage == nil {
		f.Usage = &codex.WorkerUsage{}
	}
	f.Usage.InputTokens += usage.InputTokens
	f.Usage.CachedInputTokens += usage.CachedInputTokens
	f.Usage.CacheCreationTokens += usage.CacheCreationTokens
	f.Usage.OutputTokens += usage.OutputTokens
	f.Usage.TotalTokens += usage.TotalTokens
	if usage.Estimated() {
		return
	}
	if f.ReportedCostUSD == nil {
		zero := 0.0
		f.ReportedCostUSD = &zero
	}
	*f.ReportedCostUSD += usage.USDCost
}
