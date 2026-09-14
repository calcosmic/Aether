package cmd

// SYN-204-02 (204-03-PLAN.md Task 1, LEARN-01): cmd-package-local names for
// pkg/colony's shared memory schema and provenance contract. The contract
// itself lives in pkg/colony (pkg/colony/memory_schema.go), not here:
// pkg/memory's PromoteService.Promote and pkg/learn's ColonyStore.Add both
// already import pkg/colony for the record types (InstinctEntry,
// MiddenEntry) they stamp, but neither package may import package cmd --
// cmd imports both of them, and the reverse would cycle. Declaring these
// as cmd-local aliases/wrappers of the SAME colony.* symbols (never a
// second, competing declaration) is what lets cmd's own writer
// (appendMiddenEntry, cmd/midden_shared.go) and this file's Task 3
// field-writer census refer to the ONE declared contract under the names
// this phase's plan uses.

import "github.com/calcosmic/Aether/pkg/colony"

// memoryStoreSchemaVersion / memoryStoreLegacySchemaVersion are cmd-local
// aliases of colony.CurrentMemorySchemaVersion /
// colony.LegacyMemorySchemaVersion -- the SAME two integers every writer
// (cmd's appendMiddenEntry, pkg/memory's PromoteService.Promote,
// pkg/learn's ColonyStore.Add) stamps against.
const (
	memoryStoreSchemaVersion       = colony.CurrentMemorySchemaVersion
	memoryStoreLegacySchemaVersion = colony.LegacyMemorySchemaVersion
)

// memoryProvenanceKind / memoryRecordLineage are cmd-local type aliases
// (not copies -- `type X = Y` is a genuine alias, so a colony.MemoryRecordLineage
// value and a memoryRecordLineage value are the identical Go type) of
// colony's shared provenance vocabulary and lineage shape.
type (
	memoryProvenanceKind = colony.MemoryProvenanceKind
	memoryRecordLineage  = colony.MemoryRecordLineage
)

// memoryStoreSchemaReadable reports whether version is a version this
// runtime can read: the legacy version or the current one. Delegates to
// colony.MemoryStoreSchemaReadable -- the ONE declared readability rule,
// never reimplemented here.
func memoryStoreSchemaReadable(version int) bool {
	return colony.MemoryStoreSchemaReadable(version)
}

// ---------------------------------------------------------------------------
// Field-level writer census (204-03-PLAN.md Task 3, LEARN-01), and the
// owner's Task 4 field-disposition decision applied on top of it.
//
// memoryStoreFieldWriters, memoryStoreFieldExceptions and
// memoryStoreFieldRetired are keyed "<store>.<json field name>" --
// namespaced by store because several of the seven live memory stores this
// census covers share field names (id/timestamp/created_at/schema_version/
// lineage), and a flat, unqualified map would silently conflate them. The
// seven stores, and the store-name prefix cmd/memory_schema_test.go's
// census uses for each, are declared together with the record types
// themselves in that test file's liveMemoryStoreCensusTypes -- not
// duplicated here. The episode ledger (cmd/episode_ledger.go) joined as the
// seventh store in 204-15 (SC3a) -- WINDOWS.md entry 38 recorded it as
// omitted precisely because registering it would have exposed nine
// writerless fields; 204-15 gave every one of those fields a real writer
// first, so it registers clean.
// ---------------------------------------------------------------------------

// memoryStoreFieldWriter documents the concrete runtime function that fills
// a census-registered field. Mirrors memoryPackPartWriter's shape
// (cmd/capsule_writer_invariant_198_2_test.go) for a different invariant --
// this one census's writer entries name a field, not a whole memory-pack
// section.
type memoryStoreFieldWriter struct {
	writer string // function / file:line that performs the write
}

// memoryStoreFieldWriters is the writer map validated by
// TestEveryMemoryStoreFieldHasALiveWriter. Every entry was confirmed this
// session by direct grep/read against this worktree's own HEAD -- never
// typed as a plausible-looking guess (CLAUDE.md's Definition of Done).
var memoryStoreFieldWriters = map[string]memoryStoreFieldWriter{
	// instinct (colony.InstinctEntry) -- pkg/memory/promote.go's
	// PromoteService.Promote is the single write chokepoint for a new or
	// dedup-reinforced instinct; pkg/memory/consolidate.go's decay/archive
	// steps (ConsolidationService.Run) reinforce trust_score/confidence/
	// archived on every phase-end pass.
	"instinct.id":                  {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.trigger":             {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.action":              {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.domain":              {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.trust_score":         {writer: "memory.PromoteService.Promote; memory.ConsolidationService.Run decay step (pkg/memory/consolidate.go)"},
	"instinct.trust_tier":          {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.confidence":          {writer: "memory.PromoteService.Promote; memory.ConsolidationService.Run decay step (pkg/memory/consolidate.go)"},
	"instinct.provenance":          {writer: "memory.PromoteService.Promote (pkg/memory/promote.go)"},
	"instinct.application_history": {writer: "recordInstinctApplicationsForPhase (cmd/instinct_application.go); instinct-apply CLI (cmd/internal_cmds.go)"},
	"instinct.archived":            {writer: "memory.PromoteService.Promote eviction step; memory.ConsolidationService.Run archive step (pkg/memory/consolidate.go)"},
	"instinct.schema_version":      {writer: "memory.PromoteService.Promote (pkg/memory/promote.go, 204-03-PLAN.md Task 1)"},
	"instinct.lineage":             {writer: "memory.PromoteService.Promote (pkg/memory/promote.go, 204-03-PLAN.md Task 1)"},

	// midden (colony.MiddenEntry) -- cmd/midden_shared.go's
	// appendMiddenEntry is the single write chokepoint for a new entry;
	// midden-acknowledge/midden-tag (cmd/midden_cmds.go) mutate an
	// existing one.
	"midden.id":              {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.timestamp":       {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.category":        {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.source":          {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.message":         {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.reviewed":        {writer: "appendMiddenEntry (cmd/midden_shared.go)"},
	"midden.acknowledged":    {writer: "midden-acknowledge (cmd/midden_cmds.go)"},
	"midden.acknowledged_at": {writer: "midden-acknowledge (cmd/midden_cmds.go)"},
	"midden.tags":            {writer: "appendMiddenEntry; midden-tag (cmd/midden_cmds.go)"},
	"midden.schema_version":  {writer: "appendMiddenEntry (cmd/midden_shared.go, 204-03-PLAN.md Task 1)"},
	"midden.lineage":         {writer: "appendMiddenEntry (cmd/midden_shared.go, 204-03-PLAN.md Task 1)"},

	// learn (learn.Entry) -- pkg/learn/colony_store.go's ColonyStore.Add is
	// the single write chokepoint for id/status/schema_version/lineage
	// defaults; the caller supplies content/evidence/classification/phase/
	// confidence at each of the three real construction sites.
	"learn.id":             {writer: "learn.ColonyStore.Add (pkg/learn/colony_store.go)"},
	"learn.content":        {writer: "captureContinueLearning (cmd/codex_continue_finalize.go); learn-add (cmd/learning_cmds.go); promoteOracleFindingAsLearning (cmd/oracle_promote.go)"},
	"learn.evidence":       {writer: "captureContinueLearning (cmd/codex_continue_finalize.go); learn-add (cmd/learning_cmds.go)"},
	"learn.classification": {writer: "captureContinueLearning (cmd/codex_continue_finalize.go); promoteOracleFindingAsLearning (cmd/oracle_promote.go)"},
	"learn.created_at":     {writer: "hiveWisdomToEntry (pkg/learn/hive_store.go)"},
	"learn.phase":          {writer: "captureContinueLearning (cmd/codex_continue_finalize.go); learn-add (cmd/learning_cmds.go)"},
	"learn.caste":          {writer: "promoteOracleFindingAsLearning (cmd/oracle_promote.go)"},
	"learn.file_path":      {writer: "hiveWisdomToEntry (pkg/learn/hive_store.go)"},
	"learn.confidence":     {writer: "captureContinueLearning (cmd/codex_continue_finalize.go); learn-add (cmd/learning_cmds.go); promoteOracleFindingAsLearning (cmd/oracle_promote.go)"},
	"learn.redacted":       {writer: "learn.ExportPack (pkg/learn/export.go)"},
	"learn.status":         {writer: "learn.ColonyStore.Add default; captureContinueLearning (cmd/codex_continue_finalize.go)"},
	"learn.schema_version": {writer: "learn.ColonyStore.Add (pkg/learn/colony_store.go, 204-03-PLAN.md Task 1)"},
	"learn.lineage":        {writer: "learn.ColonyStore.Add (pkg/learn/colony_store.go, 204-03-PLAN.md Task 1)"},

	// pheromone (colony.PheromoneSignal) -- cmd/pheromone_write.go's
	// writePheromoneSignal is the single write chokepoint for a new
	// signal; cmd/pheromone_influence.go's revoke/defer/pin/reinforce
	// functions and cmd/signal_housekeeping.go's expiry archiver mutate an
	// existing one.
	"pheromone.id":                  {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.type":                {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.priority":            {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.source":              {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.created_at":          {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.expires_at":          {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.active":              {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.strength":            {writer: "writePheromoneSignal (cmd/pheromone_write.go); tuneNoteStrengthFromOutcomes (cmd/pheromone_outcome.go)"},
	"pheromone.reason":              {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.content":             {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.content_hash":        {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.reinforcement_count": {writer: "writePheromoneSignal dedup step (cmd/pheromone_write.go); pheromone_influence.go reinforcement writer"},
	"pheromone.archived_at":         {writer: "cmd/signal_housekeeping.go expiry archiver"},
	"pheromone.tags":                {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.source_phase":        {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.provenance":          {writer: "writePheromoneSignal (cmd/pheromone_write.go)"},
	"pheromone.quarantined":         {writer: "writePheromoneSignal import-quarantine step (cmd/pheromone_write.go); cmd/pheromone_influence.go"},
	"pheromone.deferred_until":      {writer: "cmd/pheromone_influence.go deferNote"},
	"pheromone.revoked_at":          {writer: "cmd/pheromone_influence.go revokeNote"},
	"pheromone.pinned":              {writer: "cmd/pheromone_influence.go pinNote/unpinNote"},

	// credit (recruitmentCreditRecord) -- cmd/recruitment_credit.go's
	// recordRecruitmentCredit is, by its own doc comment, the ONE function
	// in cmd/ that writes credit/records.json; every field is set there.
	"credit.record_id":           {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.contribution_id":     {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.contribution_kind":   {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.changed_decision_id": {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.effect_evidence_id":  {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.outcome":             {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},
	"credit.recorded_at":         {writer: "recordRecruitmentCredit (cmd/recruitment_credit.go)"},

	// handoff (workerHandoffRecord) -- cmd/codex_dispatch_contract.go's
	// buildWorkerHandoffRecord is the single construction site; every
	// field is set there.
	"handoff.id":                       {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.workflow":                 {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.phase":                    {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.wave":                     {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.worker_name":              {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.caste":                    {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.task_id":                  {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.status":                   {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.summary":                  {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.changed_files":            {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.commands_run":             {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.verification_status":      {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.known_failures":           {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.open_decisions":           {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.assumptions":              {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.next_worker_instructions": {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.do_not_repeat":            {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},
	"handoff.freshness":                {writer: "buildWorkerHandoffRecord (cmd/codex_dispatch_contract.go)"},

	// episode (episodeLedgerRecord, cmd/episode_ledger.go) -- the durable
	// episode/outcome ledger, joined as the seventh census store in 204-15
	// (SC3a). recordEpisodeOutcome is the ONE function in cmd/ that writes
	// episodes/ledger.json (TestEpisodeLedgerHasOneWriter enforces this by
	// name); it stamps record_id/schema_version/lineage on every write and
	// normalizes episode_id. The nine fields WINDOWS.md entry 38 recorded as
	// writerless (evidence_ids, hard_gate_results, changed_decision_ids,
	// usage, reported_cost_usd, interventions, episode_revision,
	// acceptance_digest, evaluator_digest) each gained a real production
	// writer in 204-15: buildEpisodeGateResults/buildEpisodeApplicationFacts/
	// checkEpisodeCloseRecord/episodeCloseFacts.addUsage (all
	// cmd/episode_ledger.go), consumed by the build lane's, both check
	// lanes', and the swarm lane's own deferred episode closes
	// (cmd/codex_build.go, cmd/codex_continue.go,
	// cmd/codex_continue_finalize.go, cmd/swarm_cmd.go).
	"episode.record_id":            {writer: "recordEpisodeOutcome (cmd/episode_ledger.go)"},
	"episode.record_kind":          {writer: "recordEpisodeLedgerOpen/recordEpisodeLedgerClose/emitColonyLiveOutcomeRecorded/emitColonyLiveInterventionRecorded (cmd/live_events.go)"},
	"episode.episode_id":           {writer: "recordEpisodeOutcome (cmd/episode_ledger.go), normalized from every call site's own episodeID"},
	"episode.episode_kind":         {writer: "recordEpisodeLedgerOpen/recordEpisodeLedgerClose/emitColonyLiveOutcomeRecorded/emitColonyLiveInterventionRecorded (cmd/live_events.go)"},
	"episode.episode_revision":     {writer: "buildEpisodeApplicationFacts (cmd/episode_ledger.go, 204-15), consumed by cmd/codex_build.go's, cmd/codex_continue.go's and cmd/codex_continue_finalize.go's deferred episode closes"},
	"episode.runtime_version":      {writer: "episodeCloseBasics (cmd/live_events.go, 204-15); recordEpisodeLedgerOpen (cmd/live_events.go)"},
	"episode.policy_version":       {writer: "episodeCloseBasics (cmd/live_events.go, 204-15); recordEpisodeLedgerOpen (cmd/live_events.go)"},
	"episode.acceptance_digest":    {writer: "checkEpisodeCloseRecord via shadowAcceptanceDigestHex (cmd/episode_ledger.go, 204-15), consumed by both check lanes' deferred closes"},
	"episode.evaluator_digest":     {writer: "checkEpisodeCloseRecord via shadowEvaluatorDigestHex (cmd/episode_ledger.go, 204-15), consumed by both check lanes' deferred closes"},
	"episode.evidence_ids":         {writer: "buildEpisodeApplicationFacts (cmd/episode_ledger.go, 204-15), consumed by cmd/codex_build.go's, cmd/codex_continue.go's and cmd/codex_continue_finalize.go's deferred episode closes"},
	"episode.hard_gate_results":    {writer: "buildEpisodeGateResults (cmd/episode_ledger.go, 204-15) on cmd/codex_build.go's close; checkEpisodeCloseRecord (cmd/episode_ledger.go, 204-15) on both check lanes' closes, reading the phase's own gates.json"},
	"episode.changed_decision_ids": {writer: "buildEpisodeApplicationFacts (cmd/episode_ledger.go, 204-15), consumed by cmd/codex_build.go's, cmd/codex_continue.go's and cmd/codex_continue_finalize.go's deferred episode closes"},
	"episode.interventions":        {writer: "emitColonyLiveInterventionRecorded (cmd/live_events.go, 204-13)"},
	"episode.started_at":           {writer: "recordEpisodeLedgerOpen (cmd/live_events.go); emitColonyLiveInterventionRecorded (cmd/live_events.go, 204-13)"},
	"episode.ended_at":             {writer: "episodeCloseBasics (cmd/live_events.go, 204-15)"},
	"episode.elapsed_seconds":      {writer: "episodeCloseBasics (cmd/live_events.go, 204-15)"},
	"episode.usage":                {writer: "episodeCloseFacts.addUsage (cmd/episode_ledger.go, 204-15), consumed by cmd/codex_build.go's and cmd/swarm_cmd.go's deferred episode closes"},
	"episode.reported_cost_usd":    {writer: "episodeCloseFacts.addUsage (cmd/episode_ledger.go, 204-15), consumed by cmd/codex_build.go's and cmd/swarm_cmd.go's deferred episode closes"},
	"episode.terminal_result":      {writer: "recordEpisodeLedgerClose (cmd/live_events.go); cmd/codex_build.go's, cmd/codex_continue.go's, cmd/codex_continue_finalize.go's and cmd/swarm_cmd.go's own deferred episode-close records (204-15)"},
	"episode.schema_version":       {writer: "recordEpisodeOutcome (cmd/episode_ledger.go)"},
	"episode.lineage":              {writer: "recordEpisodeOutcome (cmd/episode_ledger.go)"},
}

// memoryStoreFieldExceptions is the shrink-only allowlist for a
// census-registered field that this session's census found has no
// production writer. Every entry must carry a one-sentence reason. Seeded
// from the census's own first real run against this worktree's HEAD, per
// this plan's own instruction -- not pre-guessed.
//
// 204-03-PLAN.md Task 4 (LEARN-01) put the full census in front of the
// owner, who chose "mixture" (decide field by field, per the recommendation
// beside each): retire instinct.related_instincts (see
// memoryStoreFieldRetired below -- it is no longer in this map, which is
// why the exception count below dropped from 4 to 3 and the floor with
// it), and keep these remaining three, recorded here as knowingly empty
// rather than connected or retired.
var memoryStoreFieldExceptions = map[string]string{
	"midden.acknowledge_reason": "declared alongside Acknowledged/AcknowledgedAt for a future reviewer-supplied reason; midden-acknowledge (cmd/midden_cmds.go) sets Acknowledged/AcknowledgedAt but never this field. Owner reviewed the full field census on 2026-09-14 (204-03-PLAN.md Task 4, mixture decision) and chose to keep it, recorded knowingly empty: cheap and plausibly useful, but nothing in this phase's own planned work needs it yet.",
	"learn.parent_id":           "declared for a future hypothesis-lineage link; no production writer sets it (the only other ParentID writer in the tree, cmd/exchange.go, targets an unrelated type, exchange.ColonyEntry). Owner reviewed the full field census on 2026-09-14 (204-03-PLAN.md Task 4, mixture decision) and chose to keep it, recorded knowingly empty: plausibly useful, but nothing in this phase's own planned work needs it yet.",
	"pheromone.scope":           "declared for a future global/local pheromone distinction; only a sync-time merge picker (firstScopePtr, cmd/pheromone_sync.go) reads an existing value -- nothing originates one. Owner reviewed the full field census on 2026-09-14 (204-03-PLAN.md Task 4, mixture decision) and chose to keep it, recorded knowingly empty: plausibly useful, but nothing in this phase's own planned work needs it yet.",
}

// memoryStoreFieldExceptionFloor pins the maximum tolerated size of
// memoryStoreFieldExceptions -- the exception list may only shrink.
// Widening this map requires widening this constant in the SAME reviewed
// change, with a written reason for each new entry. See
// TestMemoryStoreFieldExceptionsOnlyShrink. Dropped from 4 to 3 when the
// owner's Task 4 decision moved instinct.related_instincts out of this map
// and into memoryStoreFieldRetired.
const memoryStoreFieldExceptionFloor = 3

// memoryStoreRetiredField documents a field the owner has explicitly agreed
// to retire (204-03-PLAN.md Task 4, LEARN-01) -- distinct from
// memoryStoreFieldExceptions, whose entries stay knowingly empty rather
// than retired. Retiring a field never deletes a stored record and never
// removes the struct field itself (which stays, so a pre-retirement
// record's own value still reads back correctly); it only means no
// production writer fills it on a newly-created record any longer. Both
// fields are mandatory and non-empty -- a retirement recorded without the
// owner's agreement date is refused by name (see
// TestRetiredFieldWithoutOwnerAgreementIsRefused), because retiring is the
// one irreversible action this phase's checkpoint named and must never
// happen by omission.
type memoryStoreRetiredField struct {
	reason        string // what the field was for, and why nobody can name a use
	ownerAgreedOn string // date the owner recorded agreement to retire (must be non-empty)
}

// memoryStoreFieldRetired is the retired-field list validated alongside
// memoryStoreFieldWriters and memoryStoreFieldExceptions by
// TestEveryMemoryStoreFieldHasALiveWriter. A field named here counts as
// accounted-for in the census, exactly like a writer or an exception does.
var memoryStoreFieldRetired = map[string]memoryStoreRetiredField{
	"instinct.related_instincts": {
		reason:        "a future link between related instincts; the only code that would ever read it (pkg/graph) is doubly orphaned per 204-CLASSIC-SYNTHESIS.md ruling (e), which forbids any plan in this phase from citing pkg/graph's existence as justification for new work -- nobody in this phase can name a use, by the phase's own prior ruling.",
		ownerAgreedOn: "2026-09-14",
	},
}
