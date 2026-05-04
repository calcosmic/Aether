# Phase 90: Learning Foundation - Context

**Gathered:** 2026-05-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Colony learning only fires on verified successful outcomes with full evidence provenance, backed by a unified memory API (new pkg/learn/) and repo isolation. Evidence-gated triggers ensure only fully-green build+continue runs produce durable memory. The unified API wraps existing pkg/memory internally and provides a MemoryStore interface with ColonyStore and HiveStore implementations. Automatic classification at creation time extends the Phase 88 privacy scanner. Learned context feeds into the existing context_ranking.go system as frozen snapshots with between-wave refresh.

This is the trust foundation that Phase 91 (SQLite + FTS5 + pheromone skills) builds on.

</domain>

<decisions>
## Implementation Decisions

### Learning Trigger Points
- **D-01:** Learning fires only after /ant-continue verifies that gates pass AND provenance is valid (SAFE-03/04). Build-complete alone is not sufficient — a build can succeed but gates can still fail.
- **D-02:** All workers must succeed for durable memory. If ANY worker failed, blocked, or errored, the entire phase run is treated as failed for learning purposes. Strictest interpretation of LRN-01.
- **D-03:** Only /ant-build + /ant-continue produces durable learning. Oracle findings, chaos results, archaeology insights, dreams — all transient, never promoted to durable memory.
- **D-04:** Existing continue provenance check (SAFE-03/04) plus gate pass status IS the verification for learning eligibility. No additional learning-specific verification layer needed.

### Memory Store Architecture
- **D-05:** New pkg/learn/ package with a unified LearnStore interface. Existing pkg/memory (trust scoring, observation capture, instinct promotion) becomes an internal implementation detail called by pkg/learn/ but not exposed to external consumers.
- **D-06:** Repo-local colony memory lives in .aether/data/learn/ subdirectory. Physically separate from colony state files. Clean for deletion and export without touching other state.
- **D-07:** MemoryStore interface with two implementations: ColonyStore (JSON in .aether/data/learn/) and HiveStore (wraps existing cmd/hive.go logic). Promotion moves entries between stores explicitly. Decoupled — Phase 91 can swap SQLite into ColonyStore without touching HiveStore.
- **D-08:** Existing pkg/memory call sites (in cmd/ and other packages) need updating to go through pkg/learn/ instead of calling pkg/memory directly.

### Evidence and Classification
- **D-09:** Full structured evidence for every learning entry: source run ID, phase number, all worker names + castes, all files modified (from provenance), gate results summary, confidence score, timestamp, and scope (repo-local by default). Structured JSON, no freeform evidence fields.
- **D-10:** Automatic classification at creation time. No user involvement for initial classification. Privacy scanner runs first, then a classification layer determines: blocked, repo-local, hive-shareable, or needs-user-approval.
- **D-11:** Classification rules extend existing privacy scanner (cmd/security_cmds.go): scanner blocks → blocked; scanner redacts (found paths) → repo-local; scanner passes clean + generic patterns → hive-shareable; ambiguous/partial → needs-user-approval.
- **D-12:** Export/import uses JSON manifest + redacted entries + redaction report. `aether learn export` generates the pack, `aether learn import` shows preview before applying. PRIV-04 trajectory records follow the same format.

### Learned Context Injection
- **D-13:** Learned memory feeds into existing context_ranking.go as ContextCandidate entries. Ranking factors from LRN-04 map to existing scoring: phase → PriorityHint, caste → relevance filter, file path → relevance score, recency → freshness, confidence → trust. Shares the same token budget.
- **D-14:** Workers receive learned context via colony-prime context assembly only (frozen snapshot). Workers do not call pkg/learn/ directly. HIVE-03 requirement satisfied.
- **D-15:** Snapshot is frozen at colony-prime assembly but refreshed between build waves (if parallel execution). Workers in later waves see updated context.
- **D-16:** Learning default is enabled. PRIV-05: can be disabled by config (global default in .planning/config.json) and by per-command flag (--no-learn). When disabled, no capture, no classification, no injection.

### Claude's Discretion
- Hive relationship design: chose MemoryStore interface with ColonyStore + HiveStore implementations over unified scope parameter or separate APIs (D-07)
- Classification rules: chose extend existing privacy scanner over path/pattern heuristics or two-pass approach (D-11)
- Context injection: chose feed into existing context_ranking.go over separate budget or full replacement (D-13)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 88 (Direct Dependency)
- `.planning/phases/88-recovery-foundation/88-CONTEXT.md` — Provenance validation and privacy gate decisions (D-08–D-11) that learning triggers depend on
- `cmd/provenance.go` — Build/continue provenance validation (SAFE-03/04) — learning eligibility checks these
- `cmd/security_cmds.go` — Existing privacy/antipattern scanner — classification layer extends this (D-11)
- `pkg/colony/sanitize.go` — Prompt injection and sanitization patterns — foundation for privacy scanning

### Phase 89 (Related)
- `.planning/phases/89-gate-self-healing-smart-planning/89-CONTEXT.md` — Fixer caste, Oracle confidence, gate-results persistence (context for evidence fields)

### Existing Learning Infrastructure (to be wrapped by pkg/learn/)
- `pkg/memory/memory.go` — Wisdom pipeline package (trust scoring, observation, instinct, consolidation)
- `pkg/memory/observe.go` — Observation capture service
- `pkg/memory/promote.go` — Auto-promotion logic
- `pkg/memory/trust.go` — Trust scoring engine (40/35/25 weighted, 7 tiers)
- `pkg/memory/queen.go` — QUEEN.md promotion
- `pkg/memory/pipeline.go` — Pipeline orchestration
- `pkg/memory/consolidate.go` — Memory consolidation
- `pkg/colony/learning.go` — Observation struct, LearningFile types
- `pkg/colony/instincts.go` — InstinctEntry with provenance, confidence, domain
- `pkg/colony/context_ranking.go` — ContextCandidate scoring (trust, freshness, confirmation, relevance) with budget-aware trimming
- `cmd/learning.go` — learning-observe, learning-check-promotion commands
- `cmd/instinct.go` — instinct-store, instinct-list commands
- `cmd/instinct_runtime.go` — runtime instinct management
- `cmd/hive.go` — hive-init, hive-store, hive-read, hive-abstract, hive-promote (200-entry cap, LRU)

### Requirements
- `.planning/REQUIREMENTS.md` — Full v1.13 requirements: HIVE-01/02/03, LRN-01/02/03/04/05/06, PRIV-03/04/05
- `.planning/ROADMAP.md` § Phase 90 — Success criteria and goal definition

### Research
- `.planning/research/SUMMARY.md` — v1.13 research synthesis (pitfall 5: false learning confidence — addressed by D-02 all-workers-must-succeed)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/colony/context_ranking.go` — `RankContextCandidates()` with `ContextCandidate` struct. Learning entries become ContextCandidates with scores derived from LRN-04 factors. The budget-aware trimming preserves existing behavior.
- `pkg/memory/trust.go` — Trust scoring engine (40/35/25 weighted, 7 trust tiers, 60-day half-life). Called internally by pkg/learn/ for confidence scoring.
- `pkg/memory/observe.go` — `CaptureWithTrust()` observation service. Becomes internal to pkg/learn/ but provides the capture foundation.
- `cmd/security_cmds.go` — `checkAntipattern()` function for content scanning. Extended with classification output (blocked/repo-local/hive-shareable/needs-approval).
- `cmd/hive.go` — `hiveStoreCmd`, `hiveReadCmd`, `hivePromoteCmd`. HiveStore implementation wraps these.
- `cmd/provenance.go` — Provenance validation at build-complete and continue. Learning eligibility checks provenance pass status.
- `pkg/events.Bus` — Event bus with JSONL persistence. Learning hooks subscribe to lifecycle topics.

### Established Patterns
- `ContextCandidate` struct in `pkg/colony/context_ranking.go` — has Trust, Freshness, Relevance, Confirmation scores plus Protected flag. Learning entries map naturally.
- `MemoryStore` pattern: `pkg/storage.Store` provides JSON read/write with file locking. ColonyStore follows this pattern.
- OutputWorkflow pattern: Go runtime returns structured JSON, wrapper markdown renders, Codex gets JSON directly.
- All new struct fields use `omitempty` for backward compatibility.
- Per-phase file naming: `.aether/data/` directory, JSON format.

### Integration Points
- Continue finalize: after gates pass and provenance verified — insert learning capture trigger (LRN-01)
- Colony-prime context assembly: add learned memory as ContextCandidates feeding into `RankContextCandidates()` (LRN-04)
- Between-wave refresh: colony-prime re-assembles context with fresh learning data (HIVE-03 between-wave refresh)
- Privacy scanner: extend `cmd/security_cmds.go` with classification output (PRIV-03)
- pkg/memory migration: existing call sites in cmd/ updated to use pkg/learn/ API instead of pkg/memory directly
- Export/import: new `aether learn export` and `aether learn import` subcommands (LRN-06, PRIV-04)
- Config: add `learning.enabled` to config.json, `--no-learn` flag to build/continue commands (PRIV-05)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches following established patterns.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 90-Learning Foundation*
*Context gathered: 2026-05-01*
