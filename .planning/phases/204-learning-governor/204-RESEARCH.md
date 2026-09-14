# Phase 204: Learning Governor - Research

**Researched:** 2026-09-14
**Domain:** Colony memory truth (schema/provenance/hypothesis-labeling), episode/outcome ledgers, application evidence, evaluation architecture, shadow candidate comparison, and bounded promotion/rollback with retained owner authority
**Confidence:** MEDIUM-HIGH for the current-Go census (HIGH — every claim below is read directly this session, line-cited, and cross-checked against production call sites, not grep alone); MEDIUM for the Classic-mechanism reconstruction (one direct historical citation obtained at `3a5b81c2:.aether/learning.md`, but the full SYNTH-06 reconstruction across all of Classic's learning surfaces — Queen memory, Hive, midden, signal reinforcement — was not exhaustively performed in this pass and is flagged as required planner/synthesis-author work, not skipped by oversight); LOW for anything about the actual content of the `3a5b81c2`/`v5.0.0`/`v5.4` shell-era Hive/midden mechanics beyond what is cited below.

No `CONTEXT.md` exists for this phase — the owner skipped the discussion round for Phase 204. There are therefore no locked decisions to report here. Every question that is genuinely the owner's to answer is instead surfaced in `## Open Questions` below, with a recommended default and the cost of being wrong, as a substitute for that discussion round.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SYNTH-06 | Learning mechanism study: reconstruct how Classic observations, instincts, Queen memory, Hive, midden, signal reinforcement, and learning presentation influenced later work; compare with every current store/reader; choose the smallest architecture that can prove beneficial application and rollback. | `## Current Learning Store Census` is the current-side half of this study (exhaustive, call-graph-verified). The Classic-side half has one direct citation (`3a5b81c2:.aether/learning.md`, quoted in full under Census item 1 and State of the Art) but is NOT exhaustively reconstructed here — flagged as required first-task work for the `204-CLASSIC-SYNTHESIS.md` author (see Open Questions #1). |
| LEARN-01 | Memory truth and lineage: compatible versioned schemas, common provenance; hypotheses/disproven material never labelled/injected as verified; writerless legacy fields get an explicit migration or retirement plan. | Census items 1, 3, 9; a **verified, currently-live violation** is documented under Common Pitfalls #1 — hypothesis-status learning entries are injected into every worker prompt under a "(Verified Outcomes)" header. |
| LEARN-02 | Outcome and intervention ledger: append-only events, idempotent views, immutable IDs, revisions, acceptance/evaluator digests, evidence, hard gates, changed decisions, categorized interventions, time/token/cost, terminal result. | Census item 6 (`pkg/events.Bus`) plus Census item 4 (`credit/records.json`) are the two existing candidate substrates; Architecture Patterns Pattern 1 recommends the combination. |
| LEARN-03 | Real application evidence: available/rendered/consulted/acted-on/ignored/contradicted/helpful/neutral/harmful, not "delivered + phase passed." | Census item 2 is the exact existing mechanism this requirement supersedes; Common Pitfalls #2 documents precisely what it does and does not currently prove. |
| LEARN-04 | Failure-to-evaluation conversion: confirmed incidents become sanitized, deduplicated, versioned regression fixtures. | Census item 8 (midden) is the incident source; Open Questions #3 covers sourcing the 43-task seed bank. |
| LEARN-05 | Permanent evaluation/test architecture: 43-task seed bank, hard-gate sentinels, hidden holdouts, named budgeted gates (fast/focused/integration/provider/overnight/race/release) within observable budgets. | `## Standard Stack` and Common Pitfalls #4 establish the current test-suite reality this must be built on top of without slowing it down. |
| LEARN-06 | Independent shadow comparison: immutable candidate declares scope/benefit/harms/expiry/rollback, runs beside a frozen baseline, cannot select or edit its own evaluator. | Genuinely greenfield — Open Questions #4; Architecture Patterns Pattern 4 proposes the package-boundary mechanism. |
| LEARN-07 | Tiered promotion/rollback: only proven reversible low-risk knowledge/routing may canary; preferences/skills/workflows/source/security/deletion/permission/verification/external-action retain required authority. | Census items 3, 9, 10; Architecture Patterns Pattern 3 maps the existing "forced-reviewer, owner-only-waiver" mechanism onto this requirement as reusable precedent. |
| LEARN-08 | Honest improvement reporting; hard failures non-gameable; source changes only via an isolated, independently verified branch/PR that cannot self-approve/merge/publish/deploy. | Common Pitfalls #5 (anti-gaming attack surface); the `.github/workflows` sandbox restriction (Open Questions #5) is a real, already-confirmed environmental constraint on this requirement's mechanism. |
</phase_requirements>

## Summary

This phase's central discovery is the same shape as Phase 203's: most of the *hard* infrastructure this phase needs already exists, well-built, and is either half-wired or fully orphaned — not because it is missing, but because a prior phase built the write side or the read side and stopped. The single most consequential finding is that **Phase 203 already built the exact outcome vocabulary (`helpful` / `neutral` / `harmful` / `pending`) and evidence-gated, replay-safe credit ledger LEARN-02/LEARN-03/LEARN-06 need** — `recordRecruitmentCredit` (`cmd/recruitment_credit.go:184`, writing `credit/records.json`) refuses to record an outcome without both a `ChangedDecisionID` and `EffectEvidenceID`, is idempotent by content-addressed record ID, and its consumer (`tuneNoteStrengthFromOutcomes`, `cmd/pheromone_outcome.go:196`) is genuinely wired into both continue lanes and runs on every phase advance [VERIFIED: `cmd/codex_continue.go:1214`, `cmd/codex_continue_finalize.go:655`]. But `recordRecruitmentCredit` itself has **zero non-test callers anywhere in the codebase** [VERIFIED: `grep -rn "recordRecruitmentCredit(" cmd/*.go` returns only its own definition and its test file, this session] — no CLI subcommand exposes it, no wrapper or worker instruction ever tells anything to call it. The read side runs every continue against an always-empty store. This is this project's own documented signature failure, reproduced inside the exact phase (203) that most recently warned against it.

The second load-bearing finding directly proves LEARN-01's stated risk is not hypothetical but **already live**: `captureContinueLearning` writes a `learn.Entry` with `Status: learn.StatusHypothesis` on every continue [VERIFIED: `cmd/codex_continue_finalize.go:1886`], and `learnStore.List(learn.EntryFilter{MinConfidence: 0.3, Limit: 20})` — the call that injects entries into every worker's prompt — applies **no `Status` filter at all** [VERIFIED: `cmd/colony_prime_context.go:725-728`; `pkg/learn/colony_store.go:119-146` shows `EntryFilter.Status` is only applied `if filter.Status != ""`]. The injected section header literally reads `"## LEARNED MEMORY (Verified Outcomes)"` [VERIFIED: `cmd/colony_prime_context.go:731`]. `StatusValidated`/`StatusDisproven` are only ever set by `cmd/learning_cmds.go:553,596` — an orphaned CLI with no production caller [VERIFIED: audit `high | orphaned | learning-* proposal/promotion CLI (15 subcommands)`, `.planning/audits/2026-08-30-whole-system-audit.md:19`, cross-checked this session]. So every worker on every phase is, right now, shown unverified hypotheses under a header claiming they are verified.

Third: the 2026-08-30 audit that seeded the CAP ledger's `CAP-001`/`CAP-055`/`CAP-057` rows found the observation→instinct pipeline "starved at the source" — `learning-observations.json` had no live writer. That finding is **now stale**: `cmd/memory_feed.go` (file-dated 2026-09-10, after the audit) added `recordDispatchWorkerOutcome` as the one boundary both build lanes call [VERIFIED: `cmd/codex_build.go:2690`, `cmd/codex_build_finalize.go:2437`], which calls `learn.NewObservationService(store, bus).CaptureWithTrust(...)` [VERIFIED: `cmd/memory_feed.go:434`], which genuinely writes `learning-observations.json` [VERIFIED: `pkg/memory/observe.go:65,98,147`]. `runPhaseEndConsolidation` (called every continue, both lanes: `cmd/codex_continue.go:1201`, `cmd/codex_continue_finalize.go:645`) now reads real observations, not zero. **CAP-001's frozen disposition is materially further along than the ledger states**; this phase's real gap is application-*evidence richness* (Common Pitfalls #2), not observation *capture*, which Phase 198.2/203-era work already fixed. The same is true of `CAP-057` (eternal-memory promotion on pheromone expiry): `cmd/phase_end_signals.go:233` calls `appendEternalMemoryEntry` when `signalIsWorthKeeping` — also live, also fixed since the audit.

Fourth: this project has, in production, exactly **one** genuine append-only typed event bus (`pkg/events.Bus`, backed by `event-bus.jsonl`, `AppendJSONL`), and Phase 202's live/v1 topic vocabulary (`pkg/events/colony_live.go`) is already built on top of it, not a second bus [VERIFIED: STATE.md Phase 202 decision log: "Extend pkg/events.Bus with typed swarm.\*/oracle.\*/watch.\* topics rather than reviving the dead cmd/event_types.go trio or building a second bus... pkg/events.Bus already has three live callers"]. But this bus is **not unconditionally durable**: `Query`/`Replay` both exclude any event whose `ExpiresAt <= now` [VERIFIED: `pkg/events/bus.go:183,225,315`], and `DefaultTTL = 30` days [VERIFIED: `pkg/events/event.go:15`]. An outcome ledger that must prove "beneficial candidate beats frozen baseline" months later cannot silently lose its evidence to a 30-day TTL. LEARN-02 must extend this bus (never build a second one — `TestOneLiveEventModelOnly` forbids it) but must either configure a much longer/permanent TTL for its own outcome topics or add a durable snapshot/rollup view alongside it — see Open Questions #2.

Fifth, the graph layer named in CLAUDE.md's "Structural Learning Stack" ("jq-based graph layer for instinct relationships") is doubly orphaned, proven by call-graph not just grep: `pkg/graph` is imported by exactly one file (`cmd/graph_consolidation_cmds.go`) [VERIFIED: this session], and the curation ant whose name most plausibly should populate it — `Librarian` — never references `graph.` at all; it only does read-only inventory counting across four JSON files [VERIFIED: `pkg/agent/curation/librarian.go`, read in full this session, 103 lines, zero `graph.` references]. The 8-ant curation orchestrator itself (`archivist`, `critic`, `herald`, `janitor`, `librarian`, `nurse`, `scribe`, `sentinel`) runs only at seal (`runSealConsolidation`, `cmd/consolidation_lifecycle.go:455`), never at continue — this matches CLAUDE.md's own documented claim exactly, confirmed by direct code read, not merely trusted.

**Primary recommendation:** Do not build a second outcome ledger, a second event bus, or a second learning-application vocabulary. Wire real production writers into the two things Phase 203 already built correctly but left unconnected (`recordRecruitmentCredit` and its `helpful/neutral/harmful/pending` vocabulary; `pkg/events.Bus`'s live/v1 topics), extend them to cover instincts/memory-items/specialist-contributions (the vocabulary already declares these `recruitmentContributionKind` values but nothing ever uses them), close the LEARN-01 hypothesis-labeling gap with a one-line Status filter fix that is independently worth shipping regardless of the rest of this phase, and build LEARN-04/05/06/07/08 (seed bank, shadow comparison, promotion/rollback) as genuinely new, smaller-scoped work layered on top of that now-truthful substrate — reusing this codebase's already-proven checkpoint/rollback pattern (Swarm's git-checkpoint-then-verify-then-rollback) and its already-proven owner-only-authority pattern (forced-reviewer waiver) rather than inventing new versions of either.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Memory schema/provenance/hypothesis-labeling truth (LEARN-01) | API/Backend (`pkg/colony`, `pkg/learn`, `pkg/memory` type definitions + the one read boundary each store has) | — | Every store already has exactly one canonical Go type; the fix is closing gaps in existing read/write boundaries, not adding a tier. |
| Episode/outcome ledger (LEARN-02) | API/Backend (`pkg/events.Bus`, extended) | Storage (a durable rollup view alongside the TTL-bounded live bus) | "One event boundary" is this codebase's structural rule (`TestOneLiveEventModelOnly`); durability is a storage-tier concern layered on top, not a second bus. |
| Application evidence capture (LEARN-03) | API/Backend (extends `cmd/instinct_application.go` + `cmd/recruitment_credit.go`) | Client (worker self-reports a claim; Go independently verifies it, mirroring `reRunBuilderReportedEvidence`'s existing discipline) | This codebase already refuses to trust worker-self-reported completion claims at face value (build_attempt.go's re-run discipline); application evidence must follow the same rule. |
| Failure→fixture conversion (LEARN-04) | API/Backend (new: reads `midden.json` + `.planning/audits/`) | — | Midden is already the canonical, sanitized, atomically-written failure record; fixtures are a derived, versioned view of it. |
| Test/evaluation architecture (LEARN-05) | Build tooling (Go build tags / `go test` targeting, `Makefile`) | API/Backend (a seed-bank JSON manifest) | No existing tier owns test tiering today (confirmed: zero build tags in this repo); this is genuinely new tooling-tier work, not a runtime capability. |
| Shadow candidate comparison (LEARN-06) | API/Backend (a new, structurally isolated Go package with no setter back into the evaluator it runs against) | — | Must be provably impossible for a candidate to edit its own grader; a package boundary with a content-addressed, read-only evaluator reference is this codebase's closest existing idiom (see Architecture Patterns Pattern 4). |
| Promotion/canary/rollback authority (LEARN-07) | API/Backend (extends the existing forced-reviewer/owner-waiver mechanism) | Client (owner-facing approval surface, same tick-to-approve queue Phase 203 already built for pheromone suggestions) | `colony.PendingSuggestion` + `suggest_approve.go` is already the reusable owner-approval primitive this codebase uses for exactly this shape of decision. |
| Source-change boundary (LEARN-08) | External (git branch/PR, GitHub) | API/Backend (Go proposes; never merges, publishes, or deploys) | `.github/workflows` edits are already sandbox-refused to the build executor in this environment [VERIFIED: `.planning/WINDOWS.md:43`] — the "cannot self-approve/merge" requirement has a real, already-enforced backstop for at least the CI-config subset of source changes. |

## Current Learning Store Census

Every store below was traced from its Go type definition through its actual production write and read call sites, read directly this session — not inferred from a subcommand's existence or a doc's description of it.

| # | Store | File(s) | Schema/Provenance | Live Writer? | Live Reader? | Verdict |
|---|-------|---------|--------------------|:---:|:---:|---|
| 1 | `instincts.json` (standalone instinct store) | `pkg/colony/instincts.go` (`InstinctEntry`, `InstinctsFile`) | File-level `version` only; `ApplicationHistory []interface{}` (untyped `map[string]interface{}`, not a typed struct) — weaker than pheromones' pointer-backed typed fields | YES — `PromoteService.Promote` (`pkg/memory/promote.go:38`), called from consolidation | YES — injected into worker briefs (`cmd/colony_prime_context.go:609-617`, skips only `Archived`, no trust-tier filter per audit finding `low\|orphaned` row 106) | **Live, but schema is the weakest-typed of the major stores; no per-entry version.** |
| 2 | `instinct-deliveries.json` + application-history entries in `instincts.json` (the LEARN-03 precursor) | `cmd/instinct_application.go` | Untyped `map[string]interface{}` history entries; delivery = substring match of `inst.Action` in the worker's context capsule text | YES — `recordInstinctDeliveries` (called where the capsule is built) and `recordInstinctApplicationsForPhase` (called from `runPhaseEndConsolidation`, `cmd/instinct_application.go:149`) | YES — `SummarizeInstinctApplications` (`pkg/memory/instinct_stats.go:24`) feeds decay/confidence/QUEEN-eligibility | **Live, but see Common Pitfalls #2 — every recorded "application" has `success: true` unconditionally, because "reaching this function means the phase durably advanced," not because anything was verified acted-on or helpful.** |
| 3 | `entries.json` (`learn.Entry`, hypothesis-status durable learning) | `pkg/learn/colony_store.go`, `pkg/learn/learn.go` | `Status` field with a real, declared vocabulary (`StatusHypothesis`/`StatusValidated`/`StatusDisproven`) — the vocabulary exists but nothing keeps the labeling honest at the read side | YES — `captureContinueLearning` (`cmd/codex_continue_finalize.go:1886`), writes `Status: learn.StatusHypothesis` on every continue | YES — `colony_prime_context.go:724-728`, injected into every worker's prompt with **no `Status` filter** under header `"## LEARNED MEMORY (Verified Outcomes)"` | **Live and actively mislabeling — see Common Pitfalls #1. `StatusValidated`/`StatusDisproven` writers (`cmd/learning_cmds.go:553,596`) are orphaned CLI-only.** |
| 4 | `credit/records.json` (`recruitmentCreditRecord`, CEC-07/BIO-08's outcome ledger) | `cmd/recruitment_credit.go` | Excellent: closed, declared `helpful/neutral/harmful/pending` vocabulary, requires both `ChangedDecisionID` and `EffectEvidenceID`, content-addressed idempotent record ID, four-value `recruitmentContributionKind` (`recruitment_result`/`note`/`memory_item`/`specialist_contribution`) | **NO** — `recordRecruitmentCredit` (`cmd/recruitment_credit.go:184`) has zero non-test callers anywhere; no CLI, no wrapper, no worker instruction | YES — `tuneNoteStrengthFromOutcomes` (`cmd/pheromone_outcome.go:196`) runs on every continue (`cmd/codex_continue.go:1214`, `cmd/codex_continue_finalize.go:655`) against an always-empty store | **The single highest-value orphan in this census. Best-shaped store in the whole system; genuinely zero production data ever reaches it. `recruitmentContributionMemoryItem` and `recruitmentContributionSpecialist` are declared and unused — exactly the two kinds LEARN-03 needs for instincts and specialist findings.** |
| 5 | `midden.json` (failure record) | `pkg/colony/midden.go` (`MiddenEntry`, `MiddenFile`) | File-level `version`; entry has `Category`/`Source`/`Message`/`Reviewed`/`Acknowledged*`/`Tags` — no `Provenance` or severity/confidence field | YES — `appendMiddenEntryOnce` (`cmd/memory_feed.go:348`, called from `recordDispatchWorkerOutcome`) and `appendMiddenEntry` (`cmd/midden_shared.go:33`, called from autopilot retry-exhaustion and spawn-budget ceiling) | YES — 19 distinct reader call sites (`grep` this session), including `colony_prime_context.go`, `context.go`, `status.go` | **Live and healthy — the best census entry for LEARN-04's source of confirmed incidents.** Eight of ten `midden-*` CLI subcommands remain orphaned per audit (collect/handle-revert/cross-pr/prune etc.) — not blocking, but confirms the CLI surface is far ahead of what actually runs. |
| 6 | `event-bus.jsonl` (`pkg/events.Bus`, the one genuine typed event bus) | `pkg/events/bus.go`, `pkg/events/colony_live.go`, `pkg/events/ceremony.go` | `Event{ID, Topic, Payload, Source, Timestamp, TTLDays, ExpiresAt}`; Phase 202's `live/v1` topic vocabulary and Phase 203's ceremony topics both publish through this same bus and file | YES — many production callers (`spawn.go`, `status.go`, `oracle_promote.go`, `memory_feed.go`, `consolidation_lifecycle.go`, `ceremony_emitter.go`, per STATE.md Phase 202 decision log) | YES — `Query`/`Replay`, but **both exclude any event with `ExpiresAt <= now`** (30-day `DefaultTTL`) | **Live and the correct extension point for LEARN-02 — but not unconditionally durable. See Common Pitfalls #3.** `event-bus-cleanup` (physical deletion) is orphaned CLI-only, but the *logical* 30-day exclusion on read already happens without it. |
| 7 | `worker-handoffs.json` | `cmd/codex_dispatch_contract.go` | `workerHandoffRecord`, fixed entry-count cap via `pruneWorkerHandoffRecords` | YES — `persistDispatchWorkerHandoff`, called from `recordDispatchWorkerOutcome` | YES — injected as "Previous Worker Handoffs" | **Live, bounded-window (not full-history) store; fine as a recent-context feed, not a candidate for a durable episode ledger.** |
| 8 | `pheromones.json` (pheromone signals) | `pkg/colony/pheromones.go` | The best-shaped schema in this census: typed, pointer-backed, omitempty `Provenance` (`owner`/`runtime`/`learning`/`import`/`unknown`), `Quarantined`, `Pinned`, `DeferredUntil`, `RevokedAt` — all with explicit legacy-compatible nil-reads-as-safe-default semantics documented inline | YES — single chokepoint `writePheromoneSignal` (`cmd/pheromone_write.go:43`) | YES — `resolvePheromoneSection` (worker brief) and `pheromoneOutcomeReadCreditRecords`/`tuneNoteStrengthFromOutcomes` (outcome tuning) | **Live and healthy; the schema model to imitate for other stores under LEARN-01.** |
| 9 | `~/.aether/hive/wisdom.json` (cross-colony wisdom) | `cmd/hive.go` | `hiveWisdomData`, 200-cap LRU, repo-count-based confidence tiers (2/3/4+ repos → 0.70/0.85/0.95) | YES — `promoteToHiveWithReference`, called from seal only (`cmd/codex_workflow_cmds.go:712`) AND from phase-end consolidation per CLAUDE.md's Wisdom Pipeline table (both paths use the same gate, ≥0.8 confidence, same `AETHER_HIVE_POLICY` switch) | YES — `context_weighting.go:18-73` reads hive wisdom for worker briefs, but **never bumps `access_count`/`last_accessed`**, so the 200-cap LRU eviction is driven only by the orphaned manual `hive-read` CLI — an entry the runtime actually uses can still be evicted as if unused | **Live but its own LRU input is only partially wired.** `hive-init`/`hive-abstract`/`hive-revoke`/`hive-search` remain orphaned CLI-only per audit. |
| 10 | `~/.aether/eternal/memory.json` (eternal fallback memory) | `cmd/internal_cmds.go` | Simple `content`/`category`/`confidence`/`reason` entries | YES — `appendEternalMemoryEntry`, called from `phase_end_signals.go:233` on worthwhile pheromone expiry | Fallback reader only when hive has no domain match | **Live — this fixes the audit's now-stale `CAP-057` finding.** `eternal-init`/`eternal-store` CLI remain orphaned. |
| 11 | Skill lifecycle store (SQLite + `SKILL.md` files) | `pkg/learn/skills.go` (`SkillService`, SQLite `skills` table) | A **structurally different, third schema shape** from every other store above: SQLite rows + YAML frontmatter files, `active`/`stale`/`archived` stage vocabulary, `Confidence`, `AutoCreated`, `SourceRunID` | Partially — skill creation/patch CLI exists; production auto-creation call sites not traced in this pass | Not traced in this pass | **Flagged, not fully censused — schema-incompatible with the JSON-file stores above; directly relevant to LEARN-01's "compatible versioned schemas" and LEARN-07's "skills retain required authority" (skills are explicitly named in LEARN-07's authority-retention list). Recommend a dedicated planner task to trace this store's writers before finalizing LEARN-01's schema-compatibility work.** |
| 12 | Instinct-relationship graph (`pkg/graph`) | `cmd/graph_consolidation_cmds.go` | jq-based graph edges | **NO** — imported by exactly one file, that file's own commands are orphaned CLI | **NO** — `Librarian` (the curation ant most plausibly responsible) does pure inventory counting, zero `graph.` references (verified by reading the full 103-line file) | **Doubly orphaned by direct call-graph proof, confirming the audit's grep-only finding at a deeper level.** Recommend `retire-with-proof` per the synthesis template's disposition vocabulary unless this phase finds a genuine LEARN-02/03 use for relationship edges — do not build LEARN-0x on top of an already-proven-unused layer without a concrete new consumer. |
| 13 | Build-attempt "knowledge delta" evidence (`attachBuildKnowledgeDeltas`) | `cmd/build_attempt.go`, `cmd/build_knowledge_deltas.go` | Structured decision/learning delta payload, rendered via `lifecycleCloseoutKnowledgeDeltaEvidence` | YES, as of now — `cmd/codex_build_finalize.go:918` and `cmd/codex_build.go:1385` both call it with `deriveBuildKnowledgeDeltas(...)` | YES — `cmd/lifecycle_closeout.go:268`, `cmd/work_closeout.go:291` | **STATE.md's own Phase 201 decision log says this "has no live production writer" — that claim is now stale; confirmed live this session.** Cited here as a direct warning: STATE.md decision-log claims, like CLAUDE.md claims, go stale as later phases land, and must be re-verified against code before being cited as current fact in the CLASSIC-SYNTHESIS.md. |

**Curation ants (the 8-ant seal pass):** `archivist`, `critic`, `herald`, `janitor`, `librarian`, `nurse`, `scribe`, `sentinel`, coordinated by `curation.Orchestrator`. Confirmed live at seal only (`runSealConsolidation`, `cmd/consolidation_lifecycle.go:455`), never at continue (`runPhaseEndConsolidation` uses the simpler `learn.NewPipeline`/`pkg/memory.ConsolidationService` path, `cmd/consolidation_lifecycle.go:212-227`). This matches CLAUDE.md's documented claim exactly — verified by direct read of both call sites, not merely trusted.

**Trust scoring (`pkg/memory/trust.go`, Wisdom Pipeline stage 1a):** used internally by `PromoteService.Promote` (`pkg/memory/promote.go:56-61`) — NOT orphaned in the sense the audit's `trust-score-compute` CLI row implies (that CLI subcommand is indeed uncalled, but the underlying `trust.Calculate`/`Tier` functions ARE called, just not through the CLI). The audit's finding is technically correct about the CLI surface but could mislead a reader into thinking trust scoring never runs; it does, in-process, on every promotion.

**Test suite reality (relevant to LEARN-05):** 5,317 `func Test*` definitions across 781 `*_test.go` files [VERIFIED: `grep -c "^func Test" cmd/*.go | awk -F: '{s+=$2} END {print s}'`, `find cmd pkg -name "*_test.go" | wc -l`, this session]. Zero existing Go build tags for test tiering (`go:build integration`, `//go:build race`, etc. — none found) [VERIFIED: `grep -rln "go:build integration\|+build integration\|//go:build race"` returns nothing]. Combined with the already-known fact that the full `cmd` suite needs ~21 minutes and `-timeout 90m`, and an unqualified `go test ./...` silently truncates at Go's default 10-minute per-package timeout (CLAUDE.md's own "Verification Commands" section, independently reconfirmed by this project's own memory of the incident) — **LEARN-05's budgeted-gate tiering (fast/focused/integration/provider/overnight/race/release) is genuinely greenfield tooling work; nothing in this repo currently tiers tests at all.**

## Standard Stack

This phase is internal Go wiring plus new test/evaluation tooling inside an existing monorepo. No new runtime dependency is anticipated for LEARN-01/02/03/06/07/08. LEARN-05's budgeted test gates most naturally use Go's own `-run`/build-tag mechanism plus `Makefile`/CI target definitions — not a new test framework.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| (none new) | — | — | `pkg/storage.Store.UpdateJSONAtomically`, `pkg/events.Bus`, `pkg/colony`, and the standard library `testing` package cover every mechanism LEARN-01..08 needs. |

**Version verification:** `go.mod` declares `go 1.26.5` [VERIFIED: `go.mod:3`, read this session]. `go version` on this machine reports `go1.26.5 darwin/arm64` — matches. No package installs are proposed by this research.

### Package Legitimacy Audit

Not applicable — no external packages are installed or recommended. If a genuine gap surfaces during implementation (e.g., a content-addressing library for evaluator digests, though `crypto/sha256` in the standard library already covers this need — see Pattern 4), run the Package Legitimacy Gate protocol at that time.

## Architecture Patterns

### System Architecture Diagram

```text
 Build/continue worker completes a task
   │
   ▼
 recordDispatchWorkerOutcome (cmd/memory_feed.go) — ALREADY the one boundary
   │  both build lanes call. Already writes: learning-observations.json,
   │  midden.json, worker-handoffs.json.
   │
   ├──► (LEARN-02, extend) emit a live/v1-family "episode outcome" event on
   │      the SAME pkg/events.Bus Phase 202 already uses — never a second
   │      bus. Configure a long/permanent TTL for this topic family so
   │      Query/Replay do not silently exclude it after 30 days (see Open
   │      Questions #2).
   │
   ├──► (LEARN-03, extend) when the outcome concerns a delivered instinct
   │      or pheromone note, call recordRecruitmentCredit — ALREADY BUILT,
   │      currently unreached — with the real ChangedDecisionID/
   │      EffectEvidenceID this boundary now has visibility into. This is
   │      the single highest-leverage wiring task in the whole phase.
   │
   └──► (LEARN-01, fix independently) instinct/instinct-delivery writes
          continue to use recordInstinctDeliveries/recordInstinctApplications
          ForPhase (cmd/instinct_application.go) — extend the "success"
          determination to read the SAME credit record (helpful/neutral/
          harmful) instead of unconditionally writing success:true on any
          phase that merely advanced.
                                                       │
                                                       ▼
 Phase-end consolidation (runPhaseEndConsolidation, every continue)
   — pkg/memory.ConsolidationService.Run: decay, archive, QUEEN-eligibility
   — NOW reads real observations (memory_feed.go fix, post-audit)
   — QUEEN-eligibility gate (Confidence>=0.75 AND Applications>=3) should
     require at least one HELPFUL-outcome application, not merely 3
     applications of any (currently uniformly "successful") kind
                                                       │
                                                       ▼
 (LEARN-04, new) Confirmed midden.json incidents (Reviewed=true,
   Acknowledged=true) become candidates for the versioned 43-task seed bank
   — sanitized via colony.SanitizeSignalContent (already the untrusted-input
   discipline every other midden-fed surface uses), deduplicated, tagged
   with provenance back to the originating midden entry ID
                                                       │
                                                       ▼
 (LEARN-05, new) Named budgeted gates run subsets of the seed bank + the
   existing 5,317-test corpus, tiered by Go build tag / -run pattern —
   fast/focused run on every commit, integration/provider/overnight/race/
   release run on a slower cadence; a hidden holdout subset is NEVER
   listed in any test name a candidate or its author can grep for
                                                       │
                                                       ▼
 (LEARN-06, new) Shadow comparison package — runs a frozen baseline and an
   immutable candidate against the SAME visible-task + hidden-holdout set,
   in the SAME process, candidate has a read-only reference to a content-
   addressed (SHA-256) digest of its own evaluator — no setter exists on
   that reference type (see Pattern 4)
                                                       │
                                                       ▼
 (LEARN-07, new) Promotion gate — reuses the "forced reviewer, owner-only
   waiver" pattern (Pattern 3): only reversible, low-risk project-knowledge/
   routing candidates may enter a bounded canary; skills/workflows/source/
   security/deletion/permission/verification/external-action changes are
   refused by name at this gate, same discipline as
   TestReviewerForcedOnlyByNamedRisk
                                                       │
                                                       ▼
 Canary runs; regression against hard gates OR the hidden holdout triggers
   atomic rollback (reuse Swarm's git-checkpoint-before-mutation-then-
   verify-then-rollback pattern, cmd/swarm_cmd.go / work_repair.go) and
   quarantine (reuse the SAME quarantine mechanism Phase 203 already built
   for pheromone notes, cmd/pheromone_outcome.go's noteHarmfulQuarantine
   Threshold pattern — generalize it, do not duplicate it)
                                                       │
                                                       ▼
 (LEARN-08) Source-change candidates NEVER reach this canary path directly:
   they are proposed as an isolated branch/PR only. The build executor is
   already sandbox-refused from editing .github/workflows in this
   environment [VERIFIED: WINDOWS.md:43] — a real, already-enforced partial
   backstop against a candidate silently altering its own CI gate.
```

### Recommended Project Structure

```
cmd/
├── learning_status_vocabulary.go   # LEARN-01: fixes the hypothesis-labeling
│                                    #   gap — a shared, tested Status-aware
│                                    #   filter every "(Verified Outcomes)"-
│                                    #   labeled render path must call
├── episode_ledger.go                # LEARN-02: new live/v1-family topics +
│                                     #   a durable rollup view alongside the
│                                     #   TTL-bounded pkg/events.Bus reads
├── application_evidence.go          # LEARN-03: extends
│                                     #   instinct_application.go's outcome
│                                     #   determination to read real credit
│                                     #   records instead of unconditional
│                                     #   success:true
├── fixture_conversion.go            # LEARN-04: midden.json (confirmed,
│                                     #   acknowledged incidents) -> versioned
│                                     #   sanitized fixture bank entries
├── eval_gates.go                    # LEARN-05: budgeted-gate manifest +
│                                     #   CLI surface (which tags/patterns
│                                     #   each named gate runs)
├── shadow_comparison/                # LEARN-06: ISOLATED package — no
│   ├── evaluator.go                  #   symbol in this package may be
│   ├── candidate.go                  #   imported and then have its
│   └── baseline.go                   #   evaluator field reassigned; see
│                                      #   Pattern 4
├── promotion_gate.go                # LEARN-07: extends the existing
│                                     #   forced-reviewer/owner-waiver
│                                     #   pattern with a canary-specific
│                                     #   authority-category refusal list
└── rollback.go                      # LEARN-07: generalizes Swarm's
                                      #   checkpoint-verify-rollback and
                                      #   Phase 203's quarantine pattern

.github/workflows/                   # LEARN-08 CI-config subset: sandbox-
                                      #   refused to the build executor
                                      #   already (WINDOWS.md #26); any
                                      #   change here must be owner-escalated,
                                      #   never worker-authored+merged
```

### Pattern 1: Extend the existing credit ledger; do not build a parallel outcome store
**What:** `recordRecruitmentCredit` (`cmd/recruitment_credit.go:184`) already has the exact closed vocabulary (`helpful`/`neutral`/`harmful`/`pending`) and evidence-gating (`ChangedDecisionID` + `EffectEvidenceID` both required for an earned outcome) LEARN-02/03 need. Its `recruitmentContributionKind` vocabulary already declares `memory_item` and `specialist_contribution` — unused today. LEARN-02/03's episode/application-evidence work should call this exact function with real `contributionID`s for instincts, pheromone notes, and specialist findings, not invent a second store with a similar shape.
**When to use:** Any time this phase needs to record that a contribution (instinct, note, memory item, specialist finding) changed a decision and what the later verified outcome was.
**Example:**
```go
// Source: cmd/recruitment_credit.go:184 (read this session)
func recordRecruitmentCredit(contributionID string, kind recruitmentContributionKind,
	changedDecisionID, effectEvidenceID string, outcome recruitmentCreditOutcome,
	recordedAt string) (recruitmentCreditRecord, bool, error) {
	// ... refuses both-empty (no credit), decision-without-effect (pending),
	// effect-without-decision (refused: outcome cannot exist without
	// something it is the outcome OF) ...
}
```

### Pattern 2: Hypothesis vs. verified is a read-side filter that already exists but is unapplied
**What:** `learn.EntryFilter.Status` (`pkg/learn/colony_store.go`) already supports filtering by status; `colony_prime_context.go:725-728` simply never passes it. LEARN-01's fix is not "invent a status field" — the field, the vocabulary, and the filter machinery all already exist. The fix is passing `Status: learn.StatusValidated` (or rendering hypothesis-status entries under an honestly-labeled, separate section) at every call site currently labeling injected content "(Verified Outcomes)."
**When to use:** Every place this codebase renders `learn.Entry` content to a worker or the owner.
**Example:**
```go
// Source: cmd/colony_prime_context.go:724-728 (read this session) — THE BUG
learnEntries, _ := learnStore.List(learn.EntryFilter{
	MinConfidence: 0.3, // filter out very low confidence
	Limit:         20,  // cap entries to prevent budget exhaustion
	// Status: "" (unset) -- StatusHypothesis entries pass this filter
	// unchanged and are rendered under "## LEARNED MEMORY (Verified Outcomes)"
})
```

### Pattern 3: Reuse the forced-reviewer / owner-only-waiver authority pattern for canary promotion
**What:** CLAUDE.md's "Team Check-In and Owner Decisions" section documents a proven, tested pattern: a small, named set of risk signals forces a specific authority gate; only the owner (never the Queen, never autopilot) can waive it; the waiver is scoped to exactly one signal on one phase and does not silently generalize. `TestReviewerForcedOnlyByNamedRisk`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestAutopilotNeverWaives` are the existing locks. LEARN-07's authority-retention list (preferences/skills/workflows/source/security/deletion/permission/verification/external-action) is structurally the same shape: a named, closed list of categories, each mapped to a refusal, waivable only by direct owner action, never by an automatic learning process.
**When to use:** The promotion gate that decides whether a beneficial candidate may enter a canary, and which categories of change may never enter one at all.
**When NOT to use:** Do not copy the reviewer-*dispatch* mechanics (worker spawning) — only the authority-check *shape* (named signal → mandatory gate → owner-only waiver, scoped and recorded) transfers.

### Pattern 4: A content-addressed, read-only evaluator reference (structural tamper-resistance)
**What:** LEARN-06 requires it be structurally impossible for a candidate to select or edit its own evaluator/acceptance criteria. The Go-idiomatic mechanism available in this codebase's standard library already: compute `sha256.Sum256` over the evaluator's serialized definition at build/freeze time, store that digest as the only thing the comparison records alongside the candidate, and expose the evaluator to the candidate's own code path (if any) only through an interface with **no setter method** — a struct field that is unexported outside the `shadow_comparison` package, assigned once at package-init or freeze time, and never mutated. The freeze test is: `TestCandidateCannotAlterItsEvaluatorDigest` (name to be assigned by the planner) — construct a candidate, attempt every code path that could reach the evaluator reference, assert the digest is byte-identical before and after.
**When to use:** LEARN-06's evaluator/acceptance-criteria isolation requirement.
**Example (proposed shape, not yet built):**
```go
// Proposed shape for shadow_comparison/evaluator.go — not yet present in
// this codebase; follows this project's existing content-addressing idiom
// (e.g. cmd/recruitment_credit.go's recruitmentCreditRecordID uses a
// deterministic derived key the same way).
type FrozenEvaluator struct {
	digest [32]byte // sha256, set once at NewFrozenEvaluator, never exported
	run    func(candidate, task any) EvalResult
}

func NewFrozenEvaluator(def []byte, run func(any, any) EvalResult) FrozenEvaluator {
	return FrozenEvaluator{digest: sha256.Sum256(def), run: run}
}
// No SetDigest, no SetRun -- the type has no mutator. A candidate holding
// only a FrozenEvaluator value (not a pointer to the package's internal
// construction state) cannot alter what it is graded by.
```

### Anti-Patterns to Avoid
- **A second event bus for episodes/outcomes.** `TestOneLiveEventModelOnly` (Phase 202) already forbids this; extend `pkg/events.Bus`'s topic vocabulary instead.
- **A second outcome/credit ledger for instincts, separate from `credit/records.json`.** The vocabulary Phase 203 built already declares the contribution kinds this phase needs (`memory_item`, `specialist_contribution`); using it is strictly less work than building a parallel one, and avoids the exact "two competing implementations" pattern this project's own CLAUDE.md names as its documented failure mode.
- **Trusting a worker's self-reported "instinct_outcomes": [{"success": true/false}]" claim at face value.** Classic's design (`3a5b81c2:.aether/learning.md`) had workers self-report per-instinct success/failure directly in their output. This codebase's own existing discipline for builder-reported claims (`reRunBuilderReportedEvidence`, referenced in STATE.md's Phase 193 decision log: "Builder-reported commands are re-run... never `sh -c`") treats worker self-report as untrusted input requiring independent verification. LEARN-03 must not simply revive Classic's self-report mechanism unverified.
- **Building the 43-task seed bank, budgeted gates, or shadow comparison as a slower parallel test runner competing with the existing `go test ./...` invocation.** The existing suite is already measured at ~21 minutes and already has a known truncation trap (unqualified `go test ./...` silently runs a fraction and reports clean). LEARN-05's gates must be genuinely faster subsets/supersets of the SAME test binary, using build tags or `-run` patterns — not a second test framework.
- **Restoring the graph layer (`pkg/graph`) as a dependency of LEARN-02/03 without a concrete new consumer.** It is doubly orphaned today (no writer, no reader); reviving it as decoration for this phase without a proven use would recreate the exact orphan this phase is supposed to be closing others of.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Outcome vocabulary (helpful/neutral/harmful) + evidence-gated recording | A new outcome-ledger type/file | `recordRecruitmentCredit` / `credit/records.json` (`cmd/recruitment_credit.go`) | Already declares the exact closed vocabulary and contribution-kind taxonomy LEARN-02/03 need; already replay-safe (content-addressed record ID); already has a live, tested consumer (`tuneNoteStrengthFromOutcomes`) waiting for real data. |
| Typed, replay-safe, append-only event stream | A new event log / second bus | `pkg/events.Bus` (`event-bus.jsonl`), extended with new topic constants in `pkg/events/colony_live.go`'s style | This codebase has an explicit, test-enforced "one event boundary" rule (`TestOneLiveEventModelOnly`); Phase 202's own extension pattern (adding topic constants, not a new `NewBus` call) is the precedent to follow. |
| Owner tick-to-approve queue for a learning candidate awaiting promotion authority | A new approval UI/store | `colony.PendingSuggestion` + `suggest_approve.go` | Already the reusable mechanism Phase 203 reused for pheromone suggestions and cross-project import quarantine; the same shape (list/approve/dismiss, content-hash dedup) fits a canary-candidate approval queue directly. |
| Checkpoint-before-mutation, verify, rollback-on-failure | A new rollback engine for learning candidates | Swarm's existing git-checkpoint-then-verify-then-rollback pattern (`cmd/swarm_cmd.go`, `cmd/work_repair.go`) plus `PauseHandoff`'s transaction/receipt idempotency pattern (`pkg/colony/lifecycle.go:474-503`) | Both are proven in production for a structurally identical "try a mutation, verify it, undo it atomically if verification fails" shape; LEARN-07's canary rollback is the same shape applied to a learned policy/routing change instead of a code fix. |
| Quarantine-on-repeated-harm threshold | A new quarantine mechanism | `noteHarmfulQuarantineThreshold` pattern (`cmd/pheromone_outcome.go:44-49`) | Already built, tested, and matches `middenAutoRedirectThreshold`'s existing "N recurrences triggers an automatic action" shape used elsewhere in this codebase — generalize the threshold-count pattern, don't reinvent it. |
| Sanitizing worker/import-authored content before it becomes a stored fixture or evaluated criterion | New XSS/injection scrubbing | `colony.SanitizeSignalContent` (already the untrusted-input discipline every midden writer uses, per STATE.md's Phase 198.1 decision log) | Failure-to-fixture conversion (LEARN-04) takes worker-produced or repository-produced text and turns it into a stored, later-injected artifact — exactly the untrusted-input case this function already exists for. |
| Content-addressed identity for an evaluator/acceptance-criteria digest | A custom hashing scheme | `crypto/sha256` (standard library, already the idiom `recruitmentCreditRecordID`'s deterministic-key pattern follows conceptually) | No new dependency; matches this codebase's existing preference for deterministic, content-derived identity over random IDs where replay-safety matters. |

**Key insight:** This phase's risk profile is identical to Phase 203's, restated: the primitives (an outcome vocabulary with evidence gating, an append-only typed event bus, an owner-approval queue, a checkpoint/rollback pattern, a quarantine-threshold pattern, a sanitizer) all already exist, are well-tested in isolation, and in the single most important case (`credit/records.json`) are already wired on the READ side into production. The actual engineering risk is shipping LEARN-01..08 as new files that reference these primitives in a doc comment but never call the writer function — reproducing, inside the exact phase whose job is to catch this failure mode, the failure mode it exists to catch.

## Common Pitfalls

### Pitfall 1: Injecting hypothesis-status content under a "(Verified Outcomes)" header — CURRENTLY LIVE
**What goes wrong:** Every worker, on every phase, is shown content from `entries.json` under the literal header `"## LEARNED MEMORY (Verified Outcomes)"`, but the entries injected have never been distinguished from `StatusHypothesis` entries that have neither been validated nor disproven.
**Why it happens:** `captureContinueLearning` writes `Status: learn.StatusHypothesis` unconditionally [VERIFIED: `cmd/codex_continue_finalize.go:1886`]. `colony_prime_context.go:725-728`'s `learn.EntryFilter{MinConfidence: 0.3, Limit: 20}` does not set `Status`, and `EntryFilter`'s own implementation only filters on `Status` `if filter.Status != ""` [VERIFIED: `pkg/learn/colony_store.go:119-146`]. `StatusValidated`/`StatusDisproven` are set only by the orphaned `learning-approve-proposals`/`learning-disprove` CLI (`cmd/learning_cmds.go:553,596`), which has zero production callers per the 2026-08-30 audit's `orphaned` classification, re-confirmed by grep this session.
**How to avoid:** LEARN-01's minimum fix — pass `Status: learn.StatusValidated` explicitly at `colony_prime_context.go:725`, and either drop unvalidated entries from this section entirely or render them under an honestly-labeled separate heading (e.g. `"## RECENT OBSERVATIONS (unverified)"`). This is independently worth shipping as a standalone fix regardless of the rest of Phase 204's scope, because it is a currently-active, silent trust violation.
**Warning signs:** Any test asserting `learn.Entry` content appears in a worker capsule should also assert the entry's `Status`; a test that only checks presence, not status, will not catch a regression here (mirrors CLAUDE.md's own "a test that only checks for a named section cannot catch its replacement" principle).

### Pitfall 2: "Delivered + phase passed" is not application evidence
**What goes wrong:** `recordInstinctApplicationsForPhase` (`cmd/instinct_application.go:149-196`) records an application entry with `"success": true` for every instinct whose action text was found (substring match) in the worker's capsule, on any phase that durably advanced — with the doc comment explicitly stating: *"Reaching this function means the phase durably advanced and its gates passed — runPhaseEndConsolidation's own doc comment already states that contract, so success is recorded as true on that basis; this is not a second pass/fail signal."* There is no code path that records `"success": false`, no "contradicted," no "ignored," and no "harmful" for an instinct application anywhere in this codebase.
**Why it happens:** This mechanism (Phase 198.1's `instinct-deliveries.json`) was built to close a narrower, real gap — "was this instinct ever genuinely shown to a worker" — and does so correctly. It was never designed to answer LEARN-03's much richer question — was it acted on, ignored, or contradicted, and did applying it help, do nothing, or hurt.
**How to avoid:** Extend `recordInstinctApplicationsForPhase` (or a sibling function) to derive its success/outcome value from `recordRecruitmentCredit`'s `helpful`/`neutral`/`harmful` vocabulary once a real `ChangedDecisionID`/`EffectEvidenceID` pair exists for that instinct's delivery — rather than unconditionally writing `true`. Do not simply add more boolean flags to `ApplicationHistory`'s untyped `map[string]interface{}` shape; LEARN-01 also wants this schema tightened to a typed struct.
**Warning signs:** A LEARN-03 test that only asserts an `ApplicationHistory` entry exists, without asserting its recorded outcome differs between a phase that genuinely benefited from the instinct and one that did not, has not proven anything new — it would already pass against the current code.

### Pitfall 3: Treating `event-bus.jsonl` as a permanent outcome ledger without addressing its 30-day TTL
**What goes wrong:** LEARN-02 requires immutable, durable outcome records that a canary/rollback decision may need to reference weeks or months later. `pkg/events.Bus`'s `Query`/`Replay` (`pkg/events/bus.go:183,225,315`) both exclude any event whose `ExpiresAt <= now`, and `DefaultTTL = 30` (days) [VERIFIED: `pkg/events/event.go:15`]. A naive extension that simply adds new topic constants without addressing the TTL will silently lose outcome evidence after 30 days — exactly the "unrecoverable evidence" failure LEARN-02 is meant to prevent.
**Why it happens:** The bus's TTL/pruning design is correct and desirable for its existing purpose (live activity feed for `aether watch`, which genuinely should not accumulate forever) but was not designed with a multi-month outcome ledger in mind.
**How to avoid:** Either (a) construct a separate `events.Config` for outcome-ledger topics with a much longer or effectively-unbounded `DefaultTTL`, reusing the same `Bus`/file mechanics but a distinct config — NOT a second bus instance targeting a different file, which would violate `TestOneLiveEventModelOnly`'s spirit even if it technically passes; or (b) add a durable, periodically-materialized rollup view (a JSON file summarizing terminal outcomes, written once per episode close, that never expires) alongside the live bus. See Open Questions #2 for the tradeoff the planner/owner should resolve explicitly.
**Warning signs:** A LEARN-02 test that seeds an event, waits (or fast-forwards) past 30 days, and asserts the outcome is still queryable — if this test does not exist, the TTL gap has not been proven closed.

### Pitfall 4: Building a slower, competing test runner for LEARN-05 instead of tiering the existing one
**What goes wrong:** LEARN-05 names named budgeted gates (fast/focused/integration/provider/overnight/race/release). A naive implementation adds a second test-running mechanism (a custom Go program that shells to individual test binaries, a new task runner) rather than tiering the existing `go test ./...` invocation via build tags or `-run` patterns. This doubles maintenance and risks the exact truncation trap this repo already has one documented incident of (unqualified `go test ./...` silently completing a fraction of the suite and reporting a clean-looking summary).
**Why it happens:** "Budgeted gates" sounds like a new tool is needed; it is not — Go's own `-run`, `-short`, and build-tag mechanisms already exist for this purpose and this repo has zero existing usage of them (verified: no build tags found anywhere in `cmd/` or `pkg/`).
**How to avoid:** Define named gates as `go test` invocations with specific `-tags`/`-run` scoping against the existing corpus, verify each gate's own `discovered == executed` count matches its expected test count (the same discipline CLAUDE.md's Verification Commands section already requires for the full suite), and add the 43-task seed bank as new, appropriately-tagged tests within the SAME `cmd`/`pkg` test tree, not a separate harness.
**Warning signs:** Any new dependency added to `go.mod` for "test orchestration" or "test tiering" should be treated as a Package Legitimacy Gate trigger and questioned first — the standard library and existing `Makefile`/CI patterns are very likely sufficient.

### Pitfall 5: Confusing "the graph layer exists" with "the graph layer is a foundation to build on"
**What goes wrong:** CLAUDE.md's "Structural Learning Stack" section names a "jq-based graph layer for instinct relationships" as part of the shipped architecture. A planner reading only CLAUDE.md (not the code) could reasonably assign LEARN-02/03 tasks that "extend" or "populate" this layer, assuming it already has partial data or a partial consumer.
**Why it happens:** The layer genuinely exists as Go code (`pkg/graph`, `cmd/graph_consolidation_cmds.go`) and is genuinely documented as part of the architecture — it reads as built infrastructure.
**How to avoid:** Confirmed this session by direct read (not grep alone): `pkg/graph` is imported by exactly one file, whose own commands are orphaned CLI-only, and the curation ant (`Librarian`) most plausibly responsible for populating it does pure inventory counting with zero `graph.` references. Treat it as `retire-with-proof` candidate territory per the synthesis template's own disposition vocabulary, unless this phase identifies a genuinely new, concrete consumer for relationship-edge queries — do not build new work ON TOP OF it as though it were load-bearing.
**Warning signs:** A plan task whose only justification is "the graph layer already exists for this" without naming the specific edge type it will write and the specific query call site that will read it.

## Code Examples

### The existing outcome vocabulary and evidence gate (extend this — do not parallel it)
```go
// Source: cmd/recruitment_credit.go:184-280 (read this session)
func recordRecruitmentCredit(contributionID string, kind recruitmentContributionKind,
	changedDecisionID, effectEvidenceID string, outcome recruitmentCreditOutcome,
	recordedAt string) (recruitmentCreditRecord, bool, error) {
	// ... neither ID present -> no record at all (earns no credit) ...
	// ... decision present, no effect yet -> recorded as "pending" ...
	// ... effect present, no decision -> refused (an outcome cannot exist
	//     without something it is the outcome OF) ...
	// ... both present -> outcome must be helpful/neutral/harmful ...
	// replay of the SAME (contributionID, changedDecisionID) pair returns
	// the first stored record and writes nothing
}
```

### The unwired read side that already runs every continue (wire a real writer, not a new reader)
```go
// Source: cmd/pheromone_outcome.go:196-211 (read this session)
func tuneNoteStrengthFromOutcomes() pheromoneOutcomeTuningResult {
	// ...
	records, err := pheromoneOutcomeReadCreditRecords() // reads credit/records.json
	// ...
	result.Ran = true
	if len(records) == 0 {
		return result // <-- THIS is what happens on every continue today
	}
	// ... helpful/neutral/harmful -> strength moves; 3 harmful -> quarantine
}
```

### The verified LEARN-01 bug: no Status filter at the render site
```go
// Source: cmd/colony_prime_context.go:722-731 (read this session)
learnStore := learn.NewColonyStore(store)
learnEntries, _ := learnStore.List(learn.EntryFilter{
	MinConfidence: 0.3,
	Limit:         20,
	// Status is unset -- StatusHypothesis entries are not excluded
})
if len(learnEntries) > 0 {
	writeSectionHeader(&learnSB, "learned_memory", "## LEARNED MEMORY (Verified Outcomes)\n\n")
	// ...
}
```

### The event bus's TTL-bounded read (address before treating it as an outcome ledger)
```go
// Source: pkg/events/bus.go:162-230 (read this session)
func (b *Bus) Query(ctx context.Context, topicPattern string, since time.Time, limit int) ([]Event, error) {
	// ...
	if evt.ExpiresAt <= now { // events older than DefaultTTL (30 days) are
		continue               // invisible to Query, regardless of the
	}                           // underlying JSONL file still holding the bytes
	// ...
}
```

### Classic's own design intent for per-instinct application outcome (historical citation)
```json
// Source: git show 3a5b81c2:.aether/learning.md (read this session, v3.1
// baseline, "Reporting Learned Patterns" section) — worker-self-reported,
// per-instinct, both directions:
"instinct_outcomes": [
  {"id": "instinct_id_1", "success": true},
  {"id": "instinct_id_2", "success": false}
]
```
Note: Classic's own confidence-adjustment design explicitly moved confidence **down** on failure ("Confidence decreases when: Pattern leads to errors or rework... Contradicting evidence appears") — the current Go mechanism has no negative branch for instinct applications at all (Pitfall 2). This is a genuine "current is thinner than Classic's own design intent" finding for the synthesis author to weigh — with the caveat that Classic's mechanism was worker-self-reported and this codebase's own modern discipline (re-running builder-reported evidence rather than trusting it) argues against reviving self-report unverified (see Anti-Patterns).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Classic (v3.1, `3a5b81c2`): instincts stored inline in `COLONY_STATE.json`'s `memory.instincts`; worker self-reports `instinct_outcomes` per application; confidence moves ±0.1 on success/failure | Standalone `instincts.json` with trust-scoring, separate `instinct-deliveries.json` for genuinely-delivered proof, but success is a phase-outcome inference, not a per-instinct self-report or independently verified signal | v1.11 ("feat: add trust-scored instinct storage, standalone instinct store, and jq graph layer") introduced the standalone store; Phase 198.1 (2026-08-30) introduced delivery-recording and the "used" gate | Delivery proof is stronger than Classic's honor-system injection; per-application outcome richness is currently weaker than Classic's design intent (no negative branch at all). |
| 2026-08-30 audit: "observation→instinct→QUEEN promotion... starved at the source... nothing captures actual observations" | `cmd/memory_feed.go` (dated 2026-09-10, after the audit) wires `recordDispatchWorkerOutcome` into both build lanes, genuinely writing `learning-observations.json` via `ObservationService.CaptureWithTrust` | Between the 2026-08-30 audit and 2026-09-14 (this research) | `CAP-001`'s frozen ledger disposition is materially stale; the real remaining gap is application-evidence richness (LEARN-03), not observation capture. |
| 2026-08-30 audit: "pheromone-expire → eternal-store promotion... HEAD expiry never stores to eternal" | `cmd/phase_end_signals.go:233` calls `appendEternalMemoryEntry` on worthwhile pheromone expiry | Same window as above | `CAP-057`'s frozen disposition is also stale; confirm before assuming this is unaddressed work for Phase 204. |
| Phase 203's own CONTEXT.md (D-07/D-09): owner explicitly asked for outcome-weighted note tuning with immutable history | Built exactly as asked, wired into both continue lanes as a reader — but the writer (`recordRecruitmentCredit`) that would feed it real data was never connected | Phase 203 (2026-09-12/13) | The single highest-value finding of this research: the feature the owner explicitly requested in the immediately preceding phase is running against an empty store today. |

**Deprecated/outdated:**
- The 2026-08-30 audit's `CAP-001` and `CAP-057` rows (`.planning/research/v1.28-classic-capability-ledger.md`) — both describe a gap that later phases (198.1-era memory_feed work, and the 2026-09-10 `memory_feed.go` addition) have since closed. Treat the frozen ledger's disposition text as historical routing input, not current-state fact, and re-verify before citing in `204-CLASSIC-SYNTHESIS.md`.
- STATE.md's Phase 201 decision-log claim that `attachBuildKnowledgeDeltas`/`lifecycleCloseoutKnowledgeDeltaEvidence` "has no live production writer" — confirmed stale this session; both are called in production.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The full Classic mechanism reconstruction (Queen memory, Hive, midden, signal reinforcement at `3a5b81c2`/`v5.0.0`/`v5.4`) was not exhaustively performed in this research pass — only `.aether/learning.md` at `3a5b81c2` was read directly. | Summary, State of the Art, Open Questions #1 | If the synthesis author skips this and treats this RESEARCH.md as a complete SYNTH-06 mechanism study, the CLASSIC-SYNTHESIS.md gate ("cites exact historical and current evidence") will be under-evidenced on the historical side even though the current-Go side is exhaustively evidenced. |
| A2 | `learn.Entry`'s `StatusValidated`/`StatusDisproven` transition is genuinely unreachable in production (only via orphaned CLI) — verified by grep for write sites this session, not by exhaustively tracing every possible caller across the whole binary. | Common Pitfalls #1, Census item 3 | Low risk — the grep was specific (`entry.Status = learn.StatusValidated` / `learn.StatusDisproven`) and returned exactly the two `cmd/learning_cmds.go` lines; a hidden third writer is unlikely but not provably impossible without a full call-graph tool. |
| A3 | The skill lifecycle store (`pkg/learn/skills.go`, SQLite-backed) was flagged but not fully censused — its production write call sites were not traced in this pass. | Census item 11 | If LEARN-01's schema-compatibility work or LEARN-07's authority-retention work proceeds without tracing this store, a schema-incompatible or authority-unenforced skill-learning path could remain outside this phase's coverage. |
| A4 | No new external package is needed for any LEARN requirement, including LEARN-05's test tiering and LEARN-06's content-addressed evaluator digest. | Standard Stack, Pattern 4 | If implementation discovers a genuine need (e.g. a dedicated test-orchestration tool), the Package Legitimacy Gate must be run at that time — this claim only says nothing is pre-approved. |
| A5 | The `.github/workflows` sandbox restriction cited from `.planning/WINDOWS.md:43` is a durable environmental property of this build system, not a one-off observation from a single incident. | Architectural Responsibility Map, Common Pitfalls (LEARN-08), Open Questions #5 | If this restriction is environment-specific (e.g. only true for the Claude Code executor and not for a future CI-driven promotion path), LEARN-08's mechanism cannot rely on it as a universal backstop and needs an independent enforcement layer. |

**If this table is empty:** N/A — see rows above; all other claims in this research carry direct `path:line` citations from code read this session.

## Open Questions

1. **Does the CLASSIC-SYNTHESIS.md author need to perform the full Classic-side mechanism reconstruction (Queen memory promotion, Hive cross-colony sharing, midden reinforcement, signal-strength adjustment at the shell-script era) as a dedicated research/planning task, or is the one direct citation obtained here (`.aether/learning.md` at `3a5b81c2`) sufficient given how much of Classic's shell-era learning code was already "deprecated... migrated to Go binary" by that same baseline commit?**
   - What we know: `3a5b81c2` (v3.1, Feb 2026) post-dates the shell-to-Go migration commit (`6732b0ba`, "Deprecate dead shell scripts"); `.aether/learning.md` at that ref is a markdown design spec, not the executable shell script itself, and the git log shows `instinct-store.sh`/`learning.sh` existed and were removed well before v3.1.
   - What's unclear: Whether SYNTH-06's phrase "reconstruct how Classic observations, instincts, Queen memory, Hive, midden, signal reinforcement... actually influenced later work" requires reading the pre-migration shell scripts (older commits, not yet located precisely in this pass) in addition to the post-migration markdown spec, and whether the Hive/midden-reinforcement mechanics existed in a meaningfully different form pre- vs. post-migration.
   - Recommendation: Treat the reconstruction in this document as a strong starting point, not a completed SYNTH-06 study. The planner's plan-01 (which must produce `204-CLASSIC-SYNTHESIS.md`) should include an explicit task to `git log`/`git show` the pre-`6732b0ba` shell-script commits for Hive/midden/signal-reinforcement specifically, following the same direct-citation discipline used here for `.aether/learning.md`.

2. **Should the LEARN-02 outcome ledger extend `pkg/events.Bus` with a much longer/permanent TTL for its own topic family, or add a separate durable rollup view alongside the existing TTL-bounded live bus?**
   - What we know: `pkg/events.Bus`'s `Query`/`Replay` exclude events past a 30-day `DefaultTTL`; `TestOneLiveEventModelOnly` forbids a second bus; `events.Config.DefaultTTL` and `JSONLFile` are both per-`Bus`-instance configurable fields.
   - What's unclear: Whether "one event boundary" is interpreted strictly enough by that test's actual assertion to forbid a second `events.NewBus(...)` call targeting a different file/TTL even when it reuses the identical `Bus` type — this was not read in this pass (the test itself, not just its name, needs reading before the planner commits to either option).
   - Recommendation: Read `TestOneLiveEventModelOnly`'s actual assertion body before deciding; if it asserts "exactly one `events.NewBus` call site in production code," a second `Bus` instance (even same type, different config) would fail it, and the durable-rollup-view alternative becomes mandatory rather than optional.

3. **Where should the 43-task seed bank's tasks actually come from — midden confirmed incidents, `.planning/audits/*.md` findings, phase `VERIFICATION.md`/`REVIEW.md` files, or some combination, and who curates the final 43?**
   - What we know: `midden.json` is the healthiest existing incident-record store (Census item 5); `.planning/audits/2026-08-30-whole-system-audit.md` alone contains 284 findings (only a subset "confirmed"); phase `VERIFICATION.md`/`REVIEW.md` files exist per-phase but were not censused for count or structure in this pass.
   - What's unclear: Whether 43 is meant to be reached primarily from currently-confirmed incidents (which may be fewer or more than 43 today — not counted in this pass) or is a target size the planner should curate down/up to.
   - Recommendation: Count actual `Reviewed=true`/`Acknowledged=true` midden entries plus `status: confirmed` audit findings before committing to a sourcing plan; if the honest confirmed-incident count is well under 43, LEARN-04's fixture bank should grow organically toward 43 rather than being padded with synthetic tasks to hit the number, matching this project's Definition of Done (a fixture must be able to fail against a real, previously-confirmed defect, never a plausible-looking literal).

4. **Should LEARN-06's shadow comparison run candidate and baseline in the SAME process (in-memory, same Go binary invocation) or as two separate process invocations compared after the fact?**
   - What we know: This is genuinely greenfield — no existing precedent in this codebase for running two competing implementations side-by-side and comparing results. The isolation requirement (candidate cannot edit its own evaluator) is easier to guarantee structurally within one process (Go package boundaries, unexported fields) than across two process boundaries (which would need OS-level isolation instead).
   - What's unclear: Whether "candidate" in this requirement's context means a learned routing/policy value (data), a code change (a Go function), or both — the mechanism differs significantly (a data-valued candidate can be compared in-process trivially; a code-valued candidate compared in-process risks the candidate's own code executing with access to the same memory as its evaluator, undermining exactly the isolation LEARN-06 wants).
   - Recommendation: Scope LEARN-06's first implementation to data/policy-valued candidates only (matching LEARN-07's explicit "proven reversible low-risk project knowledge or routing" canary scope) — this sidesteps the harder code-candidate isolation problem entirely and matches the requirement's own stated boundary that source changes go through the separate LEARN-08 branch/PR path, never the canary path.

5. **Is the `.github/workflows` sandbox restriction (confirmed at `.planning/WINDOWS.md:43` for this Claude Code environment) a property the LEARN-08 mechanism may rely on, or must LEARN-08's "cannot self-approve/merge/publish/deploy" guarantee be enforced independently of any specific execution environment's sandboxing?**
   - What we know: The restriction is real and already observed in production (`WINDOWS.md #26`, resolved by "the executor was sandboxed out of .github/workflows and correctly escalated rather than forcing it").
   - What's unclear: Whether this sandboxing is guaranteed across every supported platform (Claude Code, OpenCode, Codex) and every execution mode (interactive, autopilot), or is specific to the Claude Code tool-permission model and could be absent on a different platform lane.
   - Recommendation: Treat the sandbox as a valuable defense-in-depth signal, never the sole enforcement mechanism — LEARN-08's actual guarantee should be structural (Go code that proposes a branch/PR literally has no code path calling a merge/publish/deploy API), with the sandbox as a second, independent layer, not the only layer.

## Sources

### Primary (HIGH confidence — read directly this session)
- `cmd/recruitment_credit.go`, `cmd/pheromone_outcome.go`, `cmd/pheromone_resolver.go`, `cmd/pheromone_write.go`, `cmd/pheromone_influence.go`, `cmd/pheromone_approval.go` — the credit ledger, outcome-weighted tuning, and pheromone schema (BIO-07/08 substrate this phase extends).
- `cmd/instinct_application.go`, `pkg/memory/instinct_stats.go`, `pkg/memory/promote.go`, `pkg/memory/consolidate.go`, `pkg/memory/observe.go`, `pkg/memory/pipeline.go`, `pkg/colony/instincts.go`, `pkg/colony/learning.go` — the instinct/observation pipeline, its consolidation gate, and its current application-recording mechanism.
- `pkg/learn/colony_store.go`, `pkg/learn/learn.go`, `cmd/colony_prime_context.go` (lines 700-760), `cmd/codex_continue_finalize.go` (lines 1817-1900), `cmd/learning_cmds.go` (lines 550-600) — the `learn.Entry` hypothesis/validated/disproven vocabulary and the verified LEARN-01 injection bug.
- `pkg/events/bus.go`, `pkg/events/event.go`, `pkg/events/colony_live.go`, `pkg/events/ceremony.go`, `cmd/consolidation_lifecycle.go` — the event bus, its TTL behavior, and the phase-end/seal consolidation split.
- `cmd/memory_feed.go`, `cmd/codex_build.go` (line 2690), `cmd/codex_build_finalize.go` (lines 2437, 918), `cmd/build_attempt.go`, `cmd/build_knowledge_deltas.go`, `cmd/lifecycle_closeout.go` — the observation-capture fix (post-audit) and the knowledge-delta wiring (also post-STATE.md-decision-log).
- `pkg/colony/midden.go`, `cmd/midden_shared.go` — the midden schema and writer chokepoints.
- `pkg/agent/curation/librarian.go` (full file, 103 lines) plus `cmd/graph_consolidation_cmds.go` import grep — the doubly-orphaned graph layer proof.
- `pkg/codex/permission_profile.go` (lines 1-30) — schema-versioning precedent (`PermissionProfileSchemaVersion`).
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md` (lines 1-763), `.planning/ROADMAP.md` (Phase 204 section), `.planning/research/v1.28-classic-capability-ledger.md`, `.planning/research/v1.28-classic-synthesis-template.md`, `.planning/audits/2026-08-30-whole-system-audit.md` (memory_context section, lines 1-130), `.planning/WINDOWS.md` (learning/memory-relevant entries), `.planning/phases/203-biological-runtime/203-RESEARCH.md`, `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md` (headings/structure).
- `git show 3a5b81c2:.aether/learning.md` (full file, 254 lines) — the one direct Classic-era citation obtained this session.
- `go.mod` (line 3), `go version`, `node --version` — environment verification.

### Secondary (MEDIUM confidence)
- CLAUDE.md's "Wisdom Pipeline," "Hive Brain," "Structural Learning Stack," and "The Core Insight" sections — cross-checked against code this session; largely accurate, with the STATE.md-Phase-201-decision-log staleness noted as an explicit caution.

### Tertiary (LOW confidence)
- None used — no web sources were needed for this phase; every claim traces to this repository's own source, git history, or planning documents.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; verified against `go.mod` and installed toolchain directly.
- Current-Go census (Standard Stack section notwithstanding — this refers to the store census): HIGH — every store's writer/reader claim is backed by a direct `path:line` read and, for the most consequential claims (credit ledger orphan, LEARN-01 injection bug), an explicit production-caller grep cross-check, not inference from a subcommand's existence.
- Classic-mechanism reconstruction (SYNTH-06's historical half): MEDIUM — one direct, full citation obtained; explicitly flagged as incomplete for the synthesis author (Open Questions #1), not silently treated as sufficient.
- LEARN-05/06 architecture recommendations: MEDIUM — grounded in this repo's real test-suite metrics and existing patterns, but LEARN-06 in particular is genuinely greenfield with no direct precedent to verify against; the proposed evaluator-digest mechanism is a reasoned application of this codebase's existing idioms, not a pattern read from existing code.

**Research date:** 2026-09-14
**Valid until:** 14 days — this phase's census depends on several very-recently-landed fixes (`cmd/memory_feed.go` dated 2026-09-10, Phase 203 landed 2026-09-12/13); re-verify the credit-ledger-orphan and LEARN-01-injection-bug findings immediately before implementation in case an intervening commit has already addressed either.
