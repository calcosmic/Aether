---
phase: 203-biological-runtime
plan: "01"
subsystem: planning-gate
tags: [classic-synthesis, spawn-admission, pheromones, ts-host, mechanism-study]

# Dependency graph
requires:
  - phase: 202-swarm-oracle-and-live-colony
    provides: typed live-event boundary (pkg/events.Bus, CeremonyTopicBuildSpawn) and caste-identity renderer, reused by this study's selected architecture
provides:
  - Signed 203-CLASSIC-SYNTHESIS.md with SYN-203-01..12, six CAP dispositions, six required rulings (a)-(f), and D-01..D-12 traceability to plans 203-02..203-15
  - Widened Classic contract corpus (schema.json + mechanisms.json) accepting SYN-203 identifiers and six V-203-* groups
  - TestClassicContractPhase203MechanismRegistry proving the corpus rejects unregistered/case-variant SYN-203 identifiers by name
affects: [203-02, 203-03, 203-04, 203-05, 203-06, 203-07, 203-08, 203-09, 203-10, 203-11, 203-12, 203-13, 203-14, 203-15]

# Actuals (#2632)
actuals:
  tokens: 26551
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CAP-identifier coverage asserted in aggregate across a phase's SYN-* mechanisms, not per-entry, when only a subset of the phase's SYN rows trace to a routed CAP row"

key-files:
  created:
    - .planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md
  modified:
    - cmd/testdata/classic-contract/v1/schema.json
    - cmd/testdata/classic-contract/v1/mechanisms.json
    - cmd/classic_contract_test.go
    - .planning/REQUIREMENTS.md

key-decisions:
  - "processClaims (.aether/ts-host/src/spawn-orchestrator.ts) is proven LIVE via a full call-chain trace this session (host.ts:1110,1160 -> worker-dispatch.ts:578 -> wave-orchestrator.ts:245,328), correcting RESEARCH.md's claim that it is never called -- this is a second, uncoordinated admission authority on the autopilot lane, not dead code (ruling a)."
  - "Native platform Task-tool nesting is not attempted in Phase 203's launch scope; the proven root/host-mediated subprocess fallback is the only dispatch path in v1, gated by a session-cached probe (ruling d)."
  - "D-11's per-run depth-raise flag is honoured as a separate quantity from spawnTreeBudgetMax's deliberately non-configurable tree budget; every use of the flag is written to the immutable spawn ledger (ruling c)."
  - "trophallaxis-diagnose and trophallaxis-retry (cmd/immune.go) are confirmed orphans (zero callers, in orphan_allowlist.json) and disposed retire-with-proof, freeing the name for BIO-05 (ruling e)."
  - "The pheromone read side has three disagreeing effective-strength predicates (extractSignalTexts, filterSignalsForPrompt, codex_plan.go's REDIRECT scan); one canonical resolver replaces all three, restoring the single-predicate discipline Classic's own _pheromone_read function had (ruling f)."

requirements-completed: [SYNTH-05]

coverage:
  - id: D1
    description: "Signed Phase 203 biological-runtime mechanism synthesis covering SYNTH-05's full scope (worker emergence, help-seeking, delegation, result return, trophallaxis, pheromone propagation, shared-environment effects), all 12 SYN-203 rows, 6 routed CAP dispositions, 6 required rulings, and D-01..D-12 traceability to real plans 203-02..203-15"
    requirement: "SYNTH-05"
    verification:
      - kind: other
        ref: "grep-based acceptance-criteria command from 203-01-PLAN.md Task 1 (all 6 identifier/string checks passed)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Versioned Classic contract corpus (schema.json + mechanisms.json) widened to accept SYN-203 identifiers and Phase 203 groups, rejecting unknown/case-variant identifiers by name, with 60 total mechanisms across Phases 199-203"
    verification:
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase203MechanismRegistry"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractSchema"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicMechanismCoverage"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase201MechanismRegistry"
        status: pass
      - kind: unit
        ref: "cmd/classic_contract_test.go#TestClassicContractPhase202MechanismRegistry"
        status: pass
    human_judgment: false

duration: 64min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 01: Biological-Runtime Mechanism Synthesis Summary

**Signed 203-CLASSIC-SYNTHESIS.md proving the phase's core recruitment mechanism (processClaims on the TypeScript host) is a live, uncoordinated second admission authority rather than dead code, plus a widened Classic contract corpus accepting 12 new SYN-203 mechanism identifiers.**

## Performance

- **Duration:** 64 min
- **Started:** 2026-09-12T23:53:35+02:00
- **Completed:** 2026-09-13T00:57:16+02:00
- **Tasks:** 2 completed
- **Files modified:** 4 (1 created, 3 modified) + REQUIREMENTS.md

## Accomplishments

- Reconstructed the Classic worker-spawn protocol (depth-gated recursive delegation, real-but-voluntary bash admission check, undefined-recovery compressed handoffs) and the pheromone write/read substrate from `v5.4.0`, confirming trophallaxis never existed under any name at either historical anchor (`3a5b81c2`, `v5.4.0`)
- Audited 17 current Go/TypeScript mechanisms with direct `path:line` citations, discovering the phase's single most consequential finding: `processClaims` on the TypeScript host is proven called via a traced call chain (`host.ts:1110,1160` → `worker-dispatch.ts:578` → `wave-orchestrator.ts:245,328`), making it a live second admission authority that never consults the Go ledger — not the orphan RESEARCH.md described
- Recorded all 12 `SYN-203` synthesis decisions and the 6 rulings SYNTH-05's task explicitly requires (the second admission engine, lane asymmetry, D-11's raise-flag tension, native-nesting non-dependency, the squatted trophallaxis name, and three disagreeing pheromone read predicates), each traced to a real Phase 203 plan (203-02 through 203-15, read from their actual frontmatter and objectives this session, not speculated)
- Widened the versioned Classic contract corpus (`schema.json`'s `synthesis_decision` pattern at both occurrences, plus six new `V-203-*` groups) and appended 12 `SYN-203-01..12` mechanism entries to `mechanisms.json`, bringing the registry to 60 total mechanisms across Phases 199-203
- Extended `cmd/classic_contract_test.go` with `TestClassicContractPhase203MechanismRegistry`, proving by negative fixture that an unregistered SYN identifier, a case referencing an unregistered decision, and a case-variant identifier are all refused by name

## Task Commits

Each task was committed atomically:

1. **Task 1: Publish the cited Phase 203 biological-runtime mechanism synthesis** - `12879af4` (docs)
2. **Task 2: Register SYN-203 mechanisms in the versioned Classic corpus** - `33536b01` (test)

**Plan metadata:** commit pending (this SUMMARY + REQUIREMENTS.md)

## Files Created/Modified

- `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md` - The signed mechanism study: Classic reconstruction, current audit, 12 SYN-203 decisions, selected architecture, 6 rejected alternatives, CAP routing table, verification contract, 6 required rulings, D-01..D-12 traceability, unresolved edge probes, and an owner-facing plain-English summary
- `cmd/testdata/classic-contract/v1/schema.json` - Widened `synthesis_decision` pattern (both `$defs/case` and `$defs/mechanism` occurrences) to accept `203-(0[1-9]|1[0-2])`; added `V-203-INTENT`, `V-203-ADMISSION`, `V-203-DISPATCH`, `V-203-RESULT`, `V-203-SUBTREE`, `V-203-SIGNALS` to the case group enum
- `cmd/testdata/classic-contract/v1/mechanisms.json` - Appended 12 `SYN-203-01..12` mechanism entries with titles, dispositions, CAP IDs (where a routed CAP row applies), public commands, source citations, and owning groups; added the synthesis document to `synthesis_sources`
- `cmd/classic_contract_test.go` - Added Phase 203 decisions/groups/capabilities/public-commands lookup tables, extended the cap-id and public-command validation branches, widened both hardcoded `synthesis_sources` exact-equality checks, and added `TestClassicContractPhase203MechanismRegistry`
- `.planning/REQUIREMENTS.md` - Marked `SYNTH-05` complete (checkbox), applied manually after the `requirements.mark-complete` tool reported `not_found` for this ID (see Deviations)

## Decisions Made

- **The Go admission gate becomes the sole recruitment authority; the TypeScript host's local budget arithmetic is deleted, never merely bridged with a "fixed" `processClaims()`.** Two independently-updated counters (`DEFAULT_TOTAL_BUDGET` vs. `spawnTreeBudgetMax`) can drift apart even if each decision consults the other, because a decision against a stale read is a race, not a fix.
- **Native platform Task-tool nesting is not a Phase 203 launch dependency.** Only `aether-queen.md` and `aether-route-setter.md` grant `Task` in their frontmatter (confirmed by direct grep this session, correcting the earlier count that mistakenly implied `aether-architect.md`/`aether-keeper.md` might too); the proven root/host-mediated subprocess fallback is the only v1 dispatch path.
- **One canonical pheromone resolver replaces three disagreeing effective-strength predicates**, restoring the single-predicate discipline Classic's own `_pheromone_read` function had before the current three-way fragmentation (`extractSignalTexts` misses expiry; `filterSignalsForPrompt` is the correct superset; `codex_plan.go`'s REDIRECT scan uses no 0.1 floor at all).
- **CAP-identifier coverage for the new mechanism registry entries is asserted in aggregate, not per-entry**, because only 6 of the 12 SYN-203 decisions trace to a capability row the ledger actually routed to Phase 203 (all six are pheromone-domain; the nine recruitment-domain decisions have no routed CAP counterpart). Fabricating a per-entry CAP association where the study found none would have violated this phase's own prohibition against recording a disposition without cited evidence.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `requirements.mark-complete` reported `not_found` for SYNTH-05**
- **Found during:** post-Task-2 requirements-completion step
- **Issue:** `node gsd-tools.cjs query requirements.mark-complete SYNTH-05` (and with explicit `--cwd`) returned `"not_found": ["SYNTH-05"]` despite `requirements.ready-ids` confirming `1/1 requirement(s) ready to mark complete` against the identical file and ID. The same multi-word-prefix requirement ID (`SYNTH-04`, Phase 202's equivalent) is also still unchecked in REQUIREMENTS.md despite Phase 202 having shipped a complete synthesis, suggesting this is a pre-existing tool quirk affecting the whole `SYNTH-0X` family, not something specific to this plan.
- **Fix:** Manually flipped the checkbox for `SYNTH-05` in `.planning/REQUIREMENTS.md` (the exact "checkbox" surface the tool itself targets per its own `write_set` output).
- **Files modified:** `.planning/REQUIREMENTS.md`
- **Verification:** `grep -n "SYNTH-05" .planning/REQUIREMENTS.md` shows `- [x]`.
- **Committed in:** pending plan-metadata commit

---

**Total deviations:** 1 auto-fixed (1 blocking — tool quirk worked around manually)
**Impact on plan:** No scope creep; the fix only unblocks the requirement-completion bookkeeping this plan's own frontmatter declares.

## Issues Encountered

- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — listed in this plan's `<read_first>` and named by the capability ledger as the authoritative owner brief — is a local-only, gitignored file absent from this fresh worktree checkout. The synthesis relied on the already-derived record of its content in `STATE.md`/`203-CONTEXT.md`/`203-RESEARCH.md`/the capability ledger instead of the primary document itself. This is recorded explicitly in the synthesis document's own confidence rationale (capped at `medium` rather than `high`) and as the first unresolved edge probe under SYNTH-05 — not silently substituted.
- `aether-architect.md` and `aether-keeper.md` contain the string "Task" somewhere in their files (confirmed by an initial broad grep) but do NOT grant the `Task` tool in their `tools:` frontmatter line — verified directly this session (`grep -n "^tools:"` on all four candidate files) before ruling (d) was written, to avoid inheriting a false positive into a locked ruling.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Every routed capability row (CAP-002, CAP-009, CAP-014, CAP-030, CAP-031, CAP-058) carries an independently confirmed disposition with cited evidence; every D-01..D-12 owner decision traces to a real, already-authored Phase 203 plan.
- Plans 203-02 through 203-15 (already present in the phase directory) can now cite this synthesis's `SYN-203-XX` decisions and the widened Classic contract corpus directly, rather than justifying their work by a Classic feature name alone.
- No blockers. The confidence rating on the synthesis is `medium` (not `high`) due to the unavailable Dreams report and six genuine implementation-shape unresolved edge probes (BIO-01/02/04/05/07/08) — none of these block planning, all are explicitly surfaced for the respective plan's implementer.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*
