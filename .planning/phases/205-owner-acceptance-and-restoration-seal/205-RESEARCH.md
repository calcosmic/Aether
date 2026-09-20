# Phase 205: Owner Acceptance and Restoration Seal - Research

**Researched:** 2026-09-15
**Domain:** Internal repo archaeology (Go CLI runtime + wrapper contract), no external packages
**Confidence:** HIGH for verified code/data claims; MEDIUM for a few interaction-sequence claims explicitly flagged below

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Phase Boundary.** Phase 205 is the walk-through, not more building. Three deliverables in order: (1) Reconciliation (SYNTH-07, PROOF-02, PROOF-03) — audit every Phase 199-204 synthesis artifact against the ratchet, sign every one of the 72 `CAP-001..CAP-072` rows with exactly one evidence-backed disposition; (2) The owner-driven walk-through (PROOF-04, PROOF-05) — ten journeys in a real project, run by the owner alone; (3) Acceptance and seal (PROOF-06) — owner's verdicts, a plain-words limitations card, and only on overall "yes", the milestone seal.

**Out of scope:** new capabilities; making OpenCode or Codex work end-to-end (only an honest card, D-13); Codex "skills as commands" integration; closing the 34 open WINDOWS.md entries (listed, not fixed, unless a journey fails on one); a second discovery interview about the Classic era.

- **D-01:** The owner drives all nine PROOF-05 journeys plus the opening seal (ten total) himself. Claude is not in the room.
- **D-02:** One session, all ten in order, in one fresh Claude Code chat opened in the target project. Pause/resume is the only deliberate close/reopen point.
- **D-03:** Evidence is what the program itself recorded — durable episode ledger, live event trail, colony state, worker handoffs, failure log, owner's verdicts. The owner pastes nothing. A journey whose run left no record is a finding against the program.
- **D-04:** Stuck means that journey fails. No nudge, no rescue, no explanation mid-session.
- **D-05:** Target project: `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System`. Note the space in the path; the program must cope, not be worked around.
- **D-06:** Tidy first — the 764 in-flight files on branch `codex/skills-as-commands` are snapshotted (reversible), session starts from a clean tree. The dirty-worktree journey uses one deliberate small edit the owner makes himself, not the swamp.
- **D-07:** The existing colony (3/3 phases, 20/20 tasks, finished but never sealed) is sealed and archived by the owner himself as the first act of the session (journey 10: finish-and-archive), then the new project starts through the front door.
- **D-08:** New project's goal, verbatim: "improving the power of the system (mds system) to create much more powerful ableton devices and midi devices like complex sequencers and ableton osc devices, like how people have made the ableton track colours one etc..." Owner types it himself on the day.
- **D-09:** One mark per journey (pass/fail/"worked, but…" in owner's own words), then one overall verdict: "feels like Aether again and I trust it to operate" — yes/no. Only the overall yes seals the milestone; a tally never does.
- **D-10:** On a failed/stuck journey: fix what is fixable inside this phase; owner re-runs only the failed journeys in a fresh chat. Not-fixable items go on the limitations card. Nothing sealed with a hidden failure; no automatic full reopen.
- **D-11:** Speed is pass/fail for three everyday journeys (clean small change, dirty-worktree change, pause/resume). Limit: 10 minutes from ask to "built and checked", measured from the program's own run record, never a stopwatch. Long journeys record time but are never failed for length.
- **D-12:** Every journey's elapsed time and cost appear on the card.
- **D-13:** Claude Code is the platform that has to work and the only one the owner watches. For OpenCode (1.1.63 installed) and Codex (0.154.0 installed): PROOF-04 minimum only — run the real public entry points once per platform, write one honest card each. No engineering effort to make either work.
- **D-14:** The pre-session publish IS the v1.28 release (hub predates Phases 203/204). Must bump version, run version-sync, back up the hub, pass `aether integrity`, publish from a worktree pinned to the release commit (oracle-reinstate is a shared checkout). Fixes after a failed journey ship as a follow-on release.
- **D-15:** The limitations card lists every open WINDOWS.md entry (34 at discussion time) in plain words, plus folded notes (D-20, D-22), plus every "not exercised" journey (D-17, D-19). Not a filtered subset.
- **D-16:** The card stays in planning records (this phase's directory + milestone audit). Not in release notes/CHANGELOG.
- **D-17:** The stubborn defect is whatever the session turns up — nothing planted. If nothing breaks, record "not exercised."
- **D-18:** The brief is one page: the task behind each journey, in order, nothing about which command to type. The program's own what-next card and help must carry the owner from there.
- **D-19:** Learned-change rollback journey exercised only if a learned change was actually bad. Otherwise: "not exercised by the owner; undo path proven only by the program's own tests" — explicit deferred proof, never a green tick.
- **D-20 (folded todo):** "Nothing proves who is calling" — the Phase 203 depth-check gap (trusts a caller's self-declared coordinator name). Folded as a named limitations-card line, not work.
- **D-21 (folded todo):** "Worker turnaround is too slow" — folded as a measurement: every journey's elapsed time on the card, and D-11's 10-minute limit.
- **D-22 (folded todo):** "ts-host preflight timeout hardcoded, runs in repo cwd" — routed to Phase 203/BIO-03. Folded only as a limitation line if still open; if BIO-03 retired it, cite the retiring test and leave it off. **This session's finding: BIO-03 retired it — see Common Pitfalls / Defect Ledger below. Leave off the card, cite the retiring evidence.**

### The ten journeys (contract for the brief and evidence plan)

| # | Journey | Kind | In this project |
|---|---|---|---|
| 1 | Seal and archive the finished old project, then start the new one (D-07) | opening | `/ant-seal` → archive → init with the D-08 goal |
| 2 | A clean small change | everyday, 10-min limit | a builder-tooling tweak toward the goal |
| 3 | A change with a deliberate unsaved edit in the way (D-06) | everyday, 10-min limit | owner edits one file first, then asks |
| 4 | Iterative planning | long | the multi-phase plan for the D-08 goal |
| 5 | A stubborn defect (D-17) | long | whatever the checks turn up |
| 6 | Multi-round research | long | a real question from the goal (track-colours / OSC devices) |
| 7 | Pause/resume and interruption recovery | everyday (resume), 10-min limit | close chat mid-run, reopen, resume |
| 8 | Governed recruitment / result use | long | a helper asks for backup; owner sees inline line + family tree |
| 9 | Queued physical or authority work | long | trying a generated device inside Ableton — only the owner can do it |
| 10 | A learned change rolled back (D-19) | long, conditional | only if a learned change was bad |

### Claude's Discretion

- Reconciliation mechanics: how the 36 unsigned rows + 2 Phase-205 rows are signed (extending `199-CLASSIC-COVERAGE.json` + named-test pattern to 200–204 is the obvious route — **confirmed the correct and only extensible pattern this session; see Architecture Patterns**), how the ratchet command failing on missing/duplicate/unsupported disposition is built, how the SYNTH-07 audit of the seven synthesis artifacts is scored, how PROOF-03 four-dimension parity is evidenced per slice (**the fourth dimension is "Safety" — confirmed from the synthesis template's own §7 rubric**), how 204's "Ruling (c)" re-adjudications are recorded.
- Tidy mechanics (D-06): snapshot commit vs. WIP branch, which branch the session runs on.
- Brief and verdict collection: exact shape of the one-page brief, how the owner records marks/overall verdict, how it becomes `205-UAT.md`.
- Evidence read-back: which records are read per journey, how elapsed time is derived from the episode ledger for D-11.
- Platform cards (D-13): how OpenCode/Codex honest runs are driven and rendered.
- Interruption trigger: how the brief phrases "interrupt a run" without naming a command.
- Publish sequence details (D-14) within the CLAUDE.md runbook.
- Grouping into plans and waves.

### Deferred Ideas (OUT OF SCOPE)

- Codex driven by "skills as commands" (later milestone; the in-flight `codex/skills-as-commands` branch in the target project is the owner's own experiment, preserved by D-06, not adopted).
- Making OpenCode or Codex actually work end-to-end (beyond the honest card).
- Closing the open WINDOWS.md entries (listed on the card; fixed only where a journey fails on one).
- Guarding the remaining 30 fixture-bank entries; check-episode token/cost capture (Phase 204's owner-accepted deferrals, WINDOWS 42/48 — appear on the card, not reopened).
- Spec builder feature (routed to Phase 200/PLAN-05, reviewed and left out by the owner).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SYNTH-07 | Every Phase 199-204 directory contains a `CLASSIC-SYNTHESIS.md` following the template; cites exact evidence, records alternatives, links conclusions to plan tasks/tests/CAP rows/verification; 205 rejects label-only restoration. | Synthesis Template Rubric (below) gives the 8 mandatory sections; per-phase audit found in Architecture Patterns / CAP Ledger section shows which phases already satisfy which sections, and flags 200's missing dedicated "CAP row routing table" subsection (content exists, just not under that heading). |
| PROOF-02 | Every CAP-001..CAP-072 row maps exactly once to a requirement/phase; final disposition signed only after the phase study demonstrates historical/current/synthesis; missing/duplicate/unsupported dispositions fail the ratchet. | Full row-by-row routing table extracted from the ledger (72/72 routed); exact machine-readable schema and AST-based ratchet test pattern documented from `199-CLASSIC-COVERAGE.json` + `classic_coverage_199_test.go`, ready to extend to 200-204. |
| PROOF-03 | Each phase proves outcome, behavior, and experience parity while preserving fail-closed safety; no endpoint-count or percentage claims. | Confirmed the 4th ("Safety") dimension comes from the synthesis template's §7 Verification Contract table, present in all 6 existing synthesis docs (200-204). |
| PROOF-04 | Real public entry points prove Claude Code/OpenCode/Codex outcomes or fail early with truthful limitation. | OpenCode 1.1.63 and Codex 0.154.0 confirmed installed on this machine; `.opencode/commands/ant/` (64 files) and `.codex/CODEX.md`/`.codex/agents/*.toml` (27) confirmed present as the real entry points. |
| PROOF-05 | Owner-visible UAT covers the nine named journeys. | Prior UAT file format (`199-UAT.md`, `201-UAT.md`, `202-UAT.md`) fully read; frontmatter + numbered test blocks with `expected:`/`result:` (pass/fail/skipped) is the shape `205-UAT.md` should extend for D-09's tri-state marks. Target project's real state (colony COMPLETED, 3/3 phases, 20/20 tasks, never sealed, HANDOFF.md present since 23 Aug) confirmed this session. |
| PROOF-06 | v1.28 cannot complete until Callum accepts the restored journey feels like Aether and is trustworthy; residual limitations stay explicit, never converted to green metrics. | WINDOWS.md confirmed at 34 open / 14 fixed / 48 total, still current as of this session (`last_updated: 2026-09-15T06:51:56.003Z`); the 8 field-report defects independently re-verified against current `cmd/` source (see Common Pitfalls) so the planner knows exactly which must be fixed before the session can honestly produce this evidence. |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

These bind every task this phase produces, with extra force because this phase is the one that decides "done":

- **Definition of Done:** "A requirement is satisfied only when a command exists that someone can run, and that command fails when the requirement is unmet." Applies directly to PROOF-02's ratchet (missing/duplicate/unsupported CAP disposition must fail a command) and to every fix made to the 8 field-report defects (each fix needs a test that fails without it).
- **How much proof a change needs — full rigour required for:** anything that decides work is complete/credited/verified/advanced (the CAP ratchet, the seal/entomb fixes, the UAT verdict recording); anything a worker/wrapper can influence (untrusted input — the seal-finalize completion JSON, the entomb preflight); anything touching money/tokens/deletion (the episode ledger's cost/time fields the card reports); any claim written into a shipped document about runtime behavior (the synthesis artifacts, the limitations card).
- **A test must be able to fail** — no fixture in a shape the runtime cannot produce. Directly relevant to the CAP coverage ratchet extension: `classicCoverage199ProofIndex` resolves proofs by AST-scanning real `_test.go` function declarations (rejecting comment/string decoys) — the 200-204 extension must reuse this exact mechanism, not a hand-typed proof list.
- **Communication Style / plain English:** every owner-facing card (the brief, the limitations card, the platform cards, the UAT verdict prompts) must translate repo vocabulary inline and must pass `TestVoicedScreensSpeakPlainEnglish` / `TestVoicedScreensCarryNoRawStateToken` if newly registered into the voice corpus (see Code Examples).
- **Verification Commands:** always `-timeout 90m`; check `discovered==executed`. **Research instruction reinforced this: do NOT run the full suite in this session** (shared checkout, ~20 min, another session may be active).
- **Publishing Changes / D-14:** `aether publish` builds, syncs hub, verifies version agreement; `aether integrity` validates the full chain; `aether version --check` confirms binary/hub agreement; publish from a worktree pinned to the release commit because `oracle-reinstate` is a shared checkout (confirmed via memory: a concurrent-session publish previously destroyed the hub, fixed 3da021fe — still back up the hub and run `aether integrity` after).
- **Wrapper-runtime contract:** wrappers never mutate state, never duplicate verification/gating logic. Any defect-1/4/5 fix (seal brief wording, JSON-mode stdout, wrapper drift) must respect this — the fix belongs in the Go runtime and/or the `.aether/commands/*.yaml` source-of-truth, not a wrapper-only patch (`.claude/commands/ant/seal.md`/`entomb.md` are managed-header files, "Synced by `aether update`" — hand-editing the `.claude/` copy alone would drift from the `.opencode/` mirror and the `.aether/commands/*.yaml` source).
- **Session Freshness / Protected Commands:** `init`, `seal`, `entomb` never auto-clear stale files — relevant to defensively deciding what Journey 1 actually invokes (see Common Pitfalls, Defect 2 nuance).

## Summary

Phase 205 has three deliverables and none of them touch an external package — this is 100% internal repo archaeology plus a live owner session. The reconciliation deliverable (SYNTH-07 + PROOF-02 + PROOF-03) has one proven, extensible machine pattern already built in Phase 199: `199-CLASSIC-COVERAGE.json` (a flat list of `{type, id, disposition, modern_home, plan_task, artifact, proofs[], status, historical_evidence}` rows) validated by `cmd/classic_coverage_199_test.go`, which AST-scans `cmd/*_test.go` for real function declarations so a proof string can never resolve to a comment or a string literal. This session confirmed the ledger's full 72-row routing (34 signed in Phase 199; 7/8/8/6/7 = 36 unsigned rows narratively confirmed with disposition + evidence in the 200-204 `CLASSIC-SYNTHESIS.md` "CAP row routing table" sections, but **not yet machine-signed**; 2 rows — CAP-023, CAP-054 — routed to Phase 205 itself; 3 rows — CAP-010, CAP-046, CAP-051 — marked "already proven, Phase 198.3" and independently re-confirmed inside the 200/202/201 synthesis tables respectively). No `200-204-CLASSIC-COVERAGE.json` exists yet. The obvious, and now confirmed-correct, implementation is one JSON+test pair per phase (200 through 205) reusing the exact validator functions Phase 199 wrote, generalized to take a phase parameter instead of the hard-coded 199 constants.

The walk-through deliverable (PROOF-04, PROOF-05) depends on eight field-report defects on the exact journey-1 path (seal → entomb) and on the target project's real, freshly-verified state. This session independently re-verified the field report's 8 defects against the current `oracle-reinstate` HEAD (`3c20a884`, 2026-09-15): **defects 1, 2, 4, 6 are still open** (confirmed by reading the exact current source — no fix landed since 2026-09-14); **defect 3** (visual output never reaching the Claude Code owner) is also still open — the seal/entomb wrappers still shell out to `AETHER_OUTPUT_MODE=visual` with no instruction to relay the card into Claude's own reply; **defect 5** (wrapper-vs-runtime drift, e.g. the wrapper's blanket "Gatekeeper, Auditor, and Probe" dispatch claim) is very likely still open given Gatekeeper dispatch is conditional on named risk signals, not unconditional; **defect 7** (a criterion credited with no verifying command) is narrower than the field report implies — a structural check does require *some* artifact-or-check per criterion, but nothing verifies the specific command text named in the criterion's own prose actually ran, so the exact failure mode the field report hit could recur. **Defect 2 has an important reproduction nuance for Journey 1**: the target project's `.aether/HANDOFF.md` still exists (a stale, 150KB pre-compact snapshot from 23 Aug 2026) because no `/ant-resume` has run there since — entomb's required-source check only fails on absence, not staleness, so simply running seal→entomb on this project *without* an intervening `/ant-resume` will not reproduce the bug even though it is real and will hit any future resume→seal→entomb sequence. Fix the root cause regardless (CONTEXT.md already directs this); do not rely on the specific sequence dodging it.

The target project (`/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System`) was independently re-verified this session: branch `codex/skills-as-commands`, 764 dirty files (matches D-06 exactly), colony `state: COMPLETED`, `plan.phases` (the authoritative top-level array, not the stale `plan.revisions[-1].phases` snapshot) shows all 3 phases / 20 tasks `completed`, milestone already reads "Crowned Anthill" (= "Release ready" per CLAUDE.md's milestone table) despite `.aether/data/seal/outcome.json` not existing — i.e. genuinely never sealed, only a `final-review.json` from 22 Aug exists. WINDOWS.md is confirmed still at 34 open / 14 fixed / 48 total as of `2026-09-15T06:51:56Z` (this session, same day). The hub is confirmed still at v1.0.78 (published 13 Sep, `~/.aether/version.json` and the installed binary both agree), predating Phases 203/204 — D-14's publish premise holds. D-22's folded todo is confirmed retired by BIO-03 (Phase 203's `204-04-PLAN.md`/`SUMMARY.md` citing `TestLiveReadinessSourceHasOneTimeoutEnvironmentVariable`) — it should be left off the limitations card with that citation, not re-flagged as open.

**Primary recommendation:** Extend the exact `199-CLASSIC-COVERAGE.json` + AST-proof-index pattern to phases 200-205 (one JSON + one generalized test file, not five hand-rolled copies), fix the four still-open field-report defects (1, 2, 4, 6 confirmed; 3, 5, 7 likely/narrower) on the runtime side before the owner's session, and build the one-page brief and `205-UAT.md` on the exact `199-UAT.md` frontmatter+numbered-block shape, extended with D-09's tri-state ("pass"/"fail"/"worked, but…") and D-19's "not exercised" vocabulary.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CAP row signing / ratchet | Go runtime test suite (`cmd/*_test.go`) | Planning docs (`.planning/phases/*/*-CLASSIC-COVERAGE.json`) | The ratchet must be a command that fails (Definition of Done); the JSON is data the test validates, never a substitute for the test. |
| SYNTH-07 audit | Planning docs (`*-CLASSIC-SYNTHESIS.md`) | None — this is a document-completeness audit, not runtime behavior | No code owns "does this markdown file have all 8 sections"; it is a structural read, best done as a checklist the planner/auditor runs by hand or a lightweight script over the 6 files. |
| Seal / entomb defect fixes (1, 2, 4, 6) | Go runtime (`cmd/seal_final_review.go`, `cmd/entomb_cmd.go`, `cmd/seal_confirmation.go`, `cmd/queen.go`) | `.aether/commands/seal.yaml`/`entomb.yaml` (source of truth for wrapper text) → generated `.claude/`/`.opencode/` wrappers | Per the wrapper-runtime contract, the fix belongs in Go/YAML source, never a hand-patched `.claude/commands/ant/seal.md` alone (that would desync from the OpenCode mirror). |
| Owner-driven walk-through (10 journeys) | Claude Code session (owner-typed) | Go runtime evidence sources (episode ledger, event trail, colony state, worker handoffs, midden) | D-01/D-03: Claude is not in the room during the walk-through; the program's own durable records, read back afterward, are the only evidence. |
| Platform honesty cards (OpenCode/Codex) | Go runtime rendering (`cmd/codex_visuals.go` translateHintCommandsForPlatform pattern) | `.opencode/commands/ant/*`, `.codex/CODEX.md`/`.codex/agents/*.toml` (existing entry points, unmodified) | D-13 forbids new engineering to make either platform "work" — only running the existing entry point once and rendering what actually happened. |
| Target-project evidence (M4L-AnalogWave-System) | Target project's own `.aether/data/` (episode ledger, COLONY_STATE.json) | This repo's planning docs (`205-UAT.md`, limitations card) | The target project is read-only evidence source after the session; this repo's `.planning/` is where the card and verdicts live (D-16). |
| Release / publish (D-14) | `aether publish` / `aether integrity` / `aether version --check` (Go runtime) | `.aether/docs/publish-update-runbook.md` (documented sequence), a worktree pinned to the release commit | `oracle-reinstate` is a shared checkout — publish must not run on the branch other sessions are committing to. |

## Standard Stack

**Not applicable.** This phase installs no new external packages and introduces no new libraries. All work is: (a) extending an existing internal JSON+Go-test pattern, (b) fixing bugs in existing Go source, (c) writing planning-document artifacts (brief, cards, `205-UAT.md`), and (d) a live owner session in an existing target project. The Package Legitimacy Gate and ecosystem registry checks are skipped for this reason — there is nothing to run `npm view` / `pip index versions` / `cargo search` against.

## Package Legitimacy Audit

**Not applicable — no packages are installed by this phase.** If any implementation task discovers a genuine need for a new dependency (unlikely; none of the 8 defect fixes, the ratchet extension, or the card rendering require one), the planner must re-run the Package Legitimacy Gate at that time.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │   .planning/research/                    │
                    │   v1.28-classic-capability-ledger.md      │
                    │   (72 frozen CAP rows, routed to phases)  │
                    └───────────────┬───────────────────────────┘
                                    │ routes 34/7/8/8/6/7/2 rows
                                    ▼
   ┌────────────┐   ┌──────────────────────────────────────────────────┐
   │ Phase 199  │   │ Phases 200-204: *-CLASSIC-SYNTHESIS.md            │
   │ COVERAGE   │   │ "CAP row routing table" — disposition + evidence  │
   │ .json      │   │ narratively confirmed, but NOT machine-signed     │
   │ (SIGNED,   │   │ (no 200-204-CLASSIC-COVERAGE.json exists yet)     │
   │ 34 rows)   │   └───────────────────┬──────────────────────────────┘
   └─────┬──────┘                       │ Phase 205 must extend the
         │                              │ signing pattern here
         ▼                              ▼
   ┌─────────────────────────────────────────────────────────────┐
   │ classic_coverage_199_test.go pattern (generalize to 200-205): │
   │  1. load {phase}-CLASSIC-COVERAGE.json                        │
   │  2. validateExactSets  — hard-coded expected ID lists per type│
   │  3. validateRequiredFields — no empty/TODO/placeholder values │
   │  4. validateProofResolution — AST-scan cmd/*_test.go for real │
   │     func decls (rejects comment/string decoys), artifact stat │
   │  5. validatePassingStatus — every row's status == "PASS"      │
   │  6. validateAudit — a rendered *-CLASSIC-COVERAGE.md must     │
   │     contain a matching "| ID | disposition | " row            │
   └───────────────────────────┬───────────────────────────────────┘
                                │ PROOF-02 satisfied when 72/72 rows
                                │ signed across 6 coverage JSONs
                                ▼
                    ┌───────────────────────────────┐
                    │ SYNTH-07 audit (all 7 synthesis│
                    │ artifacts have the template's  │
                    │ 8 mandatory sections)          │
                    └───────────────┬───────────────┘
                                    │ reconciliation complete
                                    ▼
   ┌───────────────────────────────────────────────────────────────┐
   │ Pre-session fixes (Common Pitfalls: defects 1,2,4,6 confirmed  │
   │ open; 3,5,7 likely/narrower) land in cmd/ + .aether/commands/  │
   │ *.yaml, THEN `aether publish` (D-14, worktree-pinned) makes    │
   │ them live in the hub the target project reads from             │
   └───────────────────────────────┬───────────────────────────────┘
                                    ▼
   ┌───────────────────────────────────────────────────────────────┐
   │ Owner's one Claude Code session, M4L-AnalogWave-System:         │
   │  Journey 1 (seal old → archive → init new, D-08 goal)           │
   │   → 2,3 (everyday, 10-min) → 4 (plan) → 5 (defect) → 6 (Oracle) │
   │   → 7 (pause/resume, chat closed+reopened) → 8 (recruit)         │
   │   → 9 (Ableton, owner-only) → 10 (rollback, conditional)         │
   │  Claude is NOT present; the owner drives every command himself   │
   └───────────────────────────────┬───────────────────────────────┘
                                    ▼
   ┌───────────────────────────────────────────────────────────────┐
   │ Evidence read-back (D-03, after the session, by Claude):        │
   │  episode ledger (.aether/data/episodes/ledger.json) — elapsed,  │
   │  cost, decisions, terminal result per journey/attempt            │
   │  + colony state, worker handoffs, midden, `aether watch` replay │
   └───────────────────────────────┬───────────────────────────────┘
                                    ▼
   ┌───────────────────────────────────────────────────────────────┐
   │ 205-UAT.md (ten marks, D-09 tri-state) + limitations card       │
   │ (D-15: 34 WINDOWS entries + D-20/D-22 folded notes + "not       │
   │ exercised" journeys) + one overall owner verdict                │
   │  → only "yes" triggers the milestone seal (D-09)                │
   └───────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure (new/extended artifacts)

```
.planning/phases/
├── 200-iterative-planning/200-CLASSIC-COVERAGE.json   # NEW — 7 rows
├── 201-queen-led-work-cycle/201-CLASSIC-COVERAGE.json # NEW — 8 rows
├── 202-swarm-oracle-and-live-colony/202-CLASSIC-COVERAGE.json # NEW — 8 rows
├── 203-biological-runtime/203-CLASSIC-COVERAGE.json   # NEW — 6 rows
├── 204-learning-governor/204-CLASSIC-COVERAGE.json    # NEW — 7 rows
├── 205-owner-acceptance-and-restoration-seal/
│   ├── 205-CLASSIC-COVERAGE.json                      # NEW — CAP-023, CAP-054 (2 rows)
│   ├── 205-BRIEF.md (or similar)                      # NEW — the one-page owner brief (D-18)
│   ├── 205-LIMITATIONS-CARD.md (or similar)            # NEW — D-15 card
│   └── 205-UAT.md                                      # NEW — ten journey marks + overall verdict (D-09)
cmd/
├── classic_coverage_199_test.go                        # EXISTING — pattern to generalize
├── classic_coverage_ratchet_test.go (or similar)        # NEW — generalized validator + master 72-row cross-phase ratchet
├── seal_final_review.go                                 # EDIT — defects 1, 5 (handoff schema sentence, Gatekeeper claim accuracy)
├── seal_confirmation.go                                 # EDIT — defect 4 (JSON-mode stdout pollution)
├── entomb_cmd.go                                        # EDIT — defect 2 (HANDOFF.md hard requirement)
└── queen.go                                             # EDIT — defect 6 (sanitizeQueenInline / lesson-promotion safety filter)
```

### Pattern 1: The AST-based proof-resolution ratchet (extend, don't reinvent)

**What:** `cmd/classic_coverage_199_test.go`'s `buildClassicCoverage199ProofIndex` walks every `cmd/*_test.go` file with `go/parser`, collecting top-level `*ast.FuncDecl` names (rejecting anything inside a comment or string literal), plus every `case.id` from `cmd/testdata/classic-contract/v1/cases.json`. A row's `proofs` array must resolve against this index (`TestX` or `case:known-case`).

**When to use:** Any new `{phase}-CLASSIC-COVERAGE.json` for phases 200-205 must reuse this exact index-building function (parameterized by phase, not five copies) — this is the mechanism that makes `TestClassicCoverage199ProofIndexRejectsCommentAndStringDecoys` pass, i.e. it is what prevents a slop-signed row ("proof": "TestSomethingThatSoundsRight" typed by hand but never compiled).

**Example (verified this session, `cmd/classic_coverage_199_test.go:279-330`):**
```go
// buildClassicCoverage199ProofIndex — the pattern to generalize
func buildClassicCoverage199ProofIndex(root string) (classicCoverage199ProofIndex, error) {
	index := classicCoverage199ProofIndex{tests: map[string]bool{}, cases: map[string]bool{}}
	cmdRoot := filepath.Join(root, "cmd")
	err := filepath.WalkDir(cmdRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil { return err }
		if entry.IsDir() || !strings.HasSuffix(path, "_test.go") { return nil }
		file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil { return parseErr }
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
				index.tests[function.Name.Name] = true
			}
		}
		return nil
	})
	// ... plus cases.json id collection
}
```

### Pattern 2: The `{version, rows: [...]}` coverage-JSON schema

**What:** Every row carries exactly these fields (`cmd/classic_coverage_199_test.go:16-30`): `type` (one of `GOAL`|`REQ`|`RESEARCH`|`CONTEXT`), `id`, `disposition` (one of `keep-current`|`restore-modern`|`replace-better`|`retire-with-proof`), `modern_home`, `plan_task`, `artifact`, `proofs` (array), `status` (must be `PASS`), `historical_evidence`.

**When to use:** Every new phase coverage JSON. `validateClassicCoverage199ExactSets` hard-codes the phase's expected ID lists per type inline in the test — the 200-205 generalization needs either (a) one parameterized test per phase reading its own expected-ID manifest, or (b) a single cross-phase test that unions all six files and checks the full 72-CAP/72-row-plus-REQ/RESEARCH/CONTEXT rows without duplication. Given PROOF-02's exact wording ("every CAP-001..CAP-072 row maps exactly once... missing, duplicate, or unsupported dispositions fail the ratchet"), a master cross-phase test asserting the union of all `type: GOAL` rows across all 6 files equals exactly `CAP-001..CAP-072` with no duplicates is the natural PROOF-02 enforcement point, layered on top of each phase's own coverage file.

### Pattern 3: Episode ledger as the evidence source (D-03)

**What:** `.aether/data/episodes/ledger.json` — `episodeLedgerFile{schema_version, entries: []episodeLedgerRecord}`. Verified fields (`cmd/episode_ledger.go:159-194`, quoted verbatim):
```go
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
	Interventions      []string                `json:"interventions,omitempty"`
	StartedAt          string                  `json:"started_at,omitempty"`
	EndedAt            string                  `json:"ended_at,omitempty"`
	ElapsedSeconds     float64                 `json:"elapsed_seconds,omitempty"`
	Usage              *codex.WorkerUsage      `json:"usage,omitempty"`
	ReportedCostUSD    *float64                `json:"reported_cost_usd,omitempty"`
	TerminalResult     string                  `json:"terminal_result,omitempty"`
	SchemaVersion      int                     `json:"schema_version,omitempty"`
	Lineage            *colony.MemoryRecordLineage `json:"lineage,omitempty"`
}
```
Path constant: `const episodeLedgerPath = "episodes/ledger.json"` (`cmd/episode_ledger.go:44`), store-relative (i.e. under the target project's `.aether/data/`). **Reader:** `aether improve` (`cmd/improvement_cmds.go`, `cmd/improvement_pass.go`, `cmd/improvement_report.go` all call `readEpisodeLedger()`); also touched by `cmd/live_events.go`, `cmd/recovery_orchestrator.go`, `cmd/rollback.go`, `cmd/shadow_cmds.go`. **D-11's 10-minute clock** should read `StartedAt`/`EndedAt`/`ElapsedSeconds` off the matching journey's episode record, never a wall-clock stopwatch — exactly as D-11 specifies.

**When to use:** This is the primary evidence source Claude reads back after the session for elapsed time, cost, and terminal result per journey (D-03, D-11, D-12). `EpisodeKind` and `TerminalResult` are the fields that let the planner map a specific journey number to a specific ledger entry.

### Pattern 4: The voice corpus registration (for any new owner-facing card)

**What:** `cmd/classic_voice_corpus_test.go:41-46` — `registerVoiceScreen(name string, render func(t *testing.T) string)` appends to a package-level `voiceScreenRegistry`; duplicate names panic. Convention: one `init()` per screen, in that screen's own test file (e.g. `classic_voice_nextaction_test.go`), "so two plans never edit the same file."

**When to use:** If the limitations card or a platform honesty card becomes a genuinely new rendered screen (as opposed to a plain planning-doc markdown file that the owner reads, not a runtime-rendered CLI screen), it must register here and pass `TestVoicedScreensSpeakPlainEnglish` + `TestVoicedScreensCarryNoRawStateToken`. **Note:** the brief, the limitations card, and `205-UAT.md` as currently scoped by CONTEXT.md ("stays in the planning records", D-16) are planning documents, not CLI-rendered screens — they likely do NOT need voice-corpus registration unless the planner decides to render them through the CLI. Flag this as a planner decision, not a settled fact.

### Anti-Patterns to Avoid

- **Hand-typing a `proofs` entry without confirming the test exists and compiles.** The whole point of `classicCoverage199ProofIndex`'s AST-based resolution is that a proof string is checked against real declarations, not eyeballed — CLAUDE.md's "a test must be able to fail" rule applies here directly.
- **Patching `.claude/commands/ant/seal.md` alone for defects 1, 3, 5.** These are `<!-- Aether-managed: runtime spec at .aether/commands/seal.yaml. Synced by aether update. -->` files — the fix belongs in the Go runtime (`cmd/seal_final_review.go` etc.) and, where wrapper text itself needs to change, in `.aether/commands/seal.yaml`, from which the `.claude/` and `.opencode/` copies are generated/kept in parity. A `.claude/`-only edit will silently diverge from the OpenCode mirror.
- **Trusting the field report's defect list as still-accurate without re-verification.** This session found all 8 defects require independent re-checking against current source — several files have moved (e.g. `seal_final_review.go` line numbers shifted) and the exact reproduction path (defect 2) has a real nuance on this specific target project (stale HANDOFF.md present, not deleted).
- **Treating "3/3 phases complete" in the target project's `plan.revisions[-1].phases` as authoritative.** That nested array is the plan-time snapshot; the true, current status lives in the top-level `plan.phases` array. Reading the wrong one during evidence read-back would misreport the target project's real state.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signing 36 CAP rows across 200-204 | A new prose-only synthesis note that "counts" as signed | The exact `199-CLASSIC-COVERAGE.json` schema + `classic_coverage_199_test.go` validator pattern, generalized | Phase 199 already solved "how do we make a disposition a fact a command checks" — a second vocabulary invites drift the CONTEXT.md canonical_refs explicitly warn against ("the new ratchet must not duplicate a second vocabulary beside them"). |
| Verifying a `proofs` entry names a real test | A hand-maintained list of "known good" test names | `go/parser`-based AST scan of `cmd/*_test.go`, exactly as `buildClassicCoverage199ProofIndex` does | Prevents exactly the failure class CLAUDE.md's Definition of Done calls out repeatedly (a green check that never really ran). |
| The owner's ten verdicts | A bespoke new verdict schema | The `199-UAT.md`/`201-UAT.md`/`202-UAT.md` frontmatter + numbered `### N. Title` / `expected:` / `result:` block shape, extended with D-09's tri-state vocabulary | `/gsd-verify-work` and the milestone audit already know how to consume this shape; a new shape is unnecessary translation risk. |
| Elapsed time / cost per journey | A stopwatch or a manually-typed duration | The episode ledger's `StartedAt`/`EndedAt`/`ElapsedSeconds`/`Usage`/`ReportedCostUSD` fields, read back via the same `readEpisodeLedger()` function `aether improve` already uses | D-03/D-11 explicitly require "the program's own run record, never a stopwatch." |
| WINDOWS.md entry text on the limitations card | Summarizing from memory or from CONTEXT.md's snapshot | Re-read `.planning/WINDOWS.md` at plan/execute time — it is a live, actively-updated file (last touched same day as this research, 2026-09-15) | Confirmed this session: the count (34/14/48) still matches CONTEXT.md, but the file could change again before the owner's session; treat it as a live source, not a frozen snapshot. |

**Key insight:** every mechanism this phase needs (the ratchet pattern, the UAT format, the episode-ledger reader, the voice-corpus registration) already exists somewhere in this codebase from Phase 199-204's own restoration work. Phase 205's job is almost entirely *extension and honest verification*, not invention — which matches its own charter ("the walk-through, not more building").

## Common Pitfalls

### Defect Ledger — 2026-09-14 field report, independently re-verified this session (2026-09-15, HEAD `3c20a884`)

The field report (`.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md`) named 8 defects on the exact seal→entomb path Journey 1 exercises. CONTEXT.md directs: "the planner must schedule fixes for everything on the ten journeys' path before the session." This session re-read the current source for each:

**Defect 1 — Seal reviewer briefs omit the handoff schema sentence (HIGH). STILL OPEN.**
`renderSealFinalReviewBrief` (`cmd/seal_final_review.go:151-181`, read in full this session) contains no reference to `codex.HandoffFieldsSummary`. By contrast, `cmd/codex_continue_plan.go:332` and `cmd/codex_build.go:4704` both append `fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected. %s\n", codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance)`. Seal results are validated through the same `IsEmptyWorkerHandoff` guard the continue path uses, so a reviewer following the seal brief exactly will still be rejected on the first finalize attempt.

**Defect 2 — `entomb` hard-requires `.aether/HANDOFF.md`, which `/ant-resume` deletes and seal never rewrites (HIGH). STILL OPEN, with an important reproduction nuance.**
`cmd/entomb_cmd.go:367-384` (read in full): `.aether/HANDOFF.md` is item 10 of the `required` slice (`{".aether/HANDOFF.md", "HANDOFF.md", "tombstone_input", filepath.Join(input.Root, ".aether", "HANDOFF.md"), false}`), and `addFile` (line 354) returns `fmt.Errorf("required source %q: %w", logical, err)` on any read failure — no fallback exists (`optionalFiles`, line 478, does not include it; no `synthesiz`/`tombstone` fallback logic exists anywhere in the file). `cmd/session_flow_cmds.go:680`'s resume transaction still declares `tx.DeclareRemoval(lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md"))`. **Nuance confirmed against the actual target project:** `M4L-AnalogWave-System/.aether/HANDOFF.md` currently exists (150,448 bytes, last modified 23 Aug 2026 — a pre-compact snapshot, not a resume artifact) because no `/ant-resume` has run in that project since. Seal itself never touches `HANDOFF.md`. **This means if the owner's session-1 opening sequence does not invoke `/ant-resume` before `/ant-seal`/`/ant-entomb`, the stale file will still be present and entomb's required-source check will pass on existence alone — the bug will not visibly reproduce even though it is real.** The planner must decide: fix the root cause regardless (per CONTEXT.md's explicit direction — "Fix before the session; otherwise the owner's first act fails on a defect already on record"), and must not treat a session that happens to skip `/ant-resume` as proof the defect doesn't exist.

**Defect 3 — Visual ceremony output never reaches the Claude Code owner (HIGH for UX). STILL OPEN.**
`.claude/commands/ant/seal.md` and `entomb.md` still instruct running `AETHER_OUTPUT_MODE=visual aether ceremony ...` / `aether entomb` via Bash with no instruction telling the Queen (Claude) to relay the card's content into her own chat reply — `grep` for "relay"/"verbatim"/"reply"/"show the owner" across both files returned nothing relevant (the one "verbatim" hit in `seal.md:16` is about force-flag preservation, unrelated). The suggested fix from the field report ("the wrappers should tell the Queen to relay each key card in her own reply") has not been implemented.

**Defect 4 — JSON-mode `seal-finalize` prints the human preflight card to stdout before/around the JSON result (MEDIUM). STILL OPEN.**
`runSealPreflightConfirmationGate` (`cmd/seal_confirmation.go:342-364`, read in full) calls `visualFprint(stdout, renderSealPreflightCard(preflight))` unconditionally on every call, before checking `decision.Proceed`; if not proceeding it also writes `renderSealPreflightConfirmationQuestionVisual` to stdout, then returns the JSON envelope via `outputOK`. `writeVisualOutput` (`cmd/codex_visuals.go:646-658`, read in full) confirms: `shouldRenderVisualOutput(w)` only gates *platform-hint command translation*, not whether the text is written — `fmt.Fprint(w, text)` runs unconditionally regardless of `AETHER_OUTPUT_MODE`. A wrapper parsing `seal-finalize --completion-file` stdout as pure JSON will still break exactly as the field report describes.

**Defect 5 — `/ant-seal` wrapper vs. runtime drift (MEDIUM). LIKELY STILL OPEN (MEDIUM confidence — not exhaustively traced).**
`.claude/commands/ant/seal.md:51` still states `dispatches: Gatekeeper, Auditor, and Probe final-review workers` as a flat, unconditional list. `sealReviewRequiredCastes` (`cmd/seal_final_review.go:51-63`) derives the actual caste list from `queenSealReviewSpecs(state, phase, depth)` — i.e. it depends on colony state, phase content, and review depth, not a fixed three-caste roster. Given CLAUDE.md's own "A reviewer is required only by a named risk signal" rule (Gatekeeper triggers only on credentials/auth/payments/release-sign-off signals), the wrapper's blanket claim is very likely inaccurate for a typical colony — matching the field report's own observation that only Auditor and Probe actually ran. Not independently re-confirmed against a live dispatch in this session (would require executing a build/seal, out of scope for read-only research) — flagged MEDIUM confidence, planner should verify against `queenSealReviewSpecs`'s actual logic before scoping a fix.

**Defect 6 — Seal promotes unsafe/noisy "lessons" into QUEEN.md with no content-safety filter (MEDIUM, safety). STILL OPEN.**
`sealFinalReviewReusableLessons` (`cmd/seal_final_review.go:809-821`) collects lesson strings from worker flow steps and findings with no filtering. `writeSealReusableLessonsToQueen` (lines 823-849) calls only `sanitizeQueenInline` before writing. `sanitizeQueenInline` (`cmd/queen.go:784-788`, read in full) only strips `\r`/`\n` and collapses whitespace — no check for `.env`/secret/credential patterns, no check against active REDIRECT signals, and no reuse of `pkg/colony/sanitize.go`'s `SanitizeSignalContent` (confirmed via grep: that function is never called from `seal_final_review.go` or `queen.go`). A worker-reported command that copies a secrets file could still be promoted to QUEEN.md verbatim and shown to the owner as a "Learned habit," exactly as the field report describes.

**Defect 7 — A task criterion can still be credited without a command that verifies its specific claim (MEDIUM). NARROWER GAP than the field report literally describes, but present.**
`cmd/criterion_evidence.go:220-222` (read in full) does refuse a criterion with *zero* artifacts and *zero* checks (`"criterion %q in phase %d has no artifact or verification check"`). This is a real, existing guard — but it only requires *some* declared verification, not that the declared verification actually covers the specific command the criterion's own prose names (e.g. "…and the operator broker suite all pass"). No code path in this file cross-checks criterion text against the actual commands configured. The exact failure mode the field report hit (a criterion crediting from build/types/lint/Jest alone while a named `test:operator` command was never configured) is not structurally prevented today.

**Defect 8 — Test runs filling the owner's disk (LOW for code, HIGH for the owner if it recurs).** Not a code defect to "fix" — an operational precaution. This session's spot-check: `/` has 22 GiB free (35% used, 1.8 TiB total), `~/Library/Caches/go-build` is 26 GiB. Not critically low today, but the planner should have the brief or pre-session checklist include a disk-space check (`df -h`) before the owner's live session, per the field report's own suggestion.

### Pitfall: reading the wrong phase-status array in the target project

`M4L-AnalogWave-System/.aether/data/COLONY_STATE.json` has **two** phase arrays: `plan.revisions[-1].phases` (a frozen plan-generation-time snapshot — all three phases read `pending`/`ready` there) and the top-level `plan.phases` (the live, authoritative array — all three phases read `completed`, 20/20 tasks). Evidence read-back after the session must use the top-level array; using the revision snapshot would wrongly conclude the colony was never finished.

### Pitfall: a second CAP-row vocabulary

CONTEXT.md explicitly warns: "the new ratchet must not duplicate a second vocabulary beside [`cmd/classic_coverage_199_test.go` and `cmd/classic_contract_test.go`]." The 200-204 synthesis docs already use a *third*, informal vocabulary in their "CAP row routing table" markdown sections (`Ledger hypothesis` / `Independently confirmed disposition` / `Evidence` columns) — this is fine as narrative justification feeding the machine-signed row, but the machine-signed row itself (in the new coverage JSONs) must use the exact same field names as `199-CLASSIC-COVERAGE.json` (`disposition`, `modern_home`, `plan_task`, `artifact`, `proofs`, `status`, `historical_evidence`), not a paraphrase.

### Pitfall: 200-CLASSIC-SYNTHESIS.md's CAP routing lives in a table column, not a dedicated section

Phases 201, 202, 203 each have a `### CAP row routing table (Phase N's N rows)` subsection under §6. Phase 200 does not — its CAP rows (CAP-005, CAP-010, CAP-011, CAP-012, CAP-056, CAP-061, CAP-069) appear only as a column inside §6's "Research-to-plan linkage" table (verified: all 7 rows are present there, just not under a matching header). This is not necessarily a defect in the synthesis template (the template's own §6 doesn't mandate a dedicated CAP subsection, only a "CAP rows" column) — but the SYNTH-07 audit should note the structural inconsistency across phases and decide whether to require Phase 200 add the dedicated subsection for audit-readability parity, or accept the column-only form as compliant.

## Code Examples

### The coverage-JSON row shape to replicate (verified, `199-CLASSIC-COVERAGE.json:4`)
```json
{"type":"GOAL","id":"CAP-006","disposition":"replace-better","modern_home":"cmd/entomb_cmd.go","plan_task":"199-21 Task 1","artifact":"cmd/entomb_cmd.go","proofs":["TestLifecycleCloseout199Entomb"],"status":"PASS","historical_evidence":"Classic entomb evidence retained in 199-CLASSIC-SYNTHESIS.md"}
```

### The exact-sets validator to generalize (verified, `cmd/classic_coverage_199_test.go:152-176`)
```go
func validateClassicCoverage199ExactSets(document classicCoverage199Document) error {
	expected := map[string][]string{
		"GOAL":     {"CAP-006", "CAP-007", /* ... 34 total */},
		"REQ":      {"SYNTH-01", "CEC-01", /* ... 12 total */},
		"RESEARCH": {"SYN-199-01", /* ... 10 total */},
		"CONTEXT":  {"D-01", /* ... 17 total */},
	}
	// counts each row.Type/row.ID exactly once against `expected`,
	// and asserts len(document.Rows) == 73 (34+12+10+17 for phase 199)
}
```
For phases 200-205 the `GOAL` (CAP) list is the exact row set from the ledger's phase-mapping table (7/8/8/6/7/2 respectively); `REQ`/`RESEARCH`/`CONTEXT` lists come from that phase's own `REQUIREMENTS.md` mapping row and its own `*-CONTEXT.md`/synthesis decision IDs.

### The 34 open WINDOWS.md entries (verified, `.planning/WINDOWS.md`, `open_count: 34`, checked this session against a live parse — do not trust a cached count, re-read at plan/execute time)
IDs currently open: 2, 4, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 25, 26, 27, 28, 29, 30, 31, 34, 35, 36, 40, 42, 43, 44, 47, 48 — spanning phases 196 through 204. Full one-line descriptions must be re-read from the file at plan/execute time (the table's `description` column contains embedded pipe characters that break naive `awk`/`cut` parsing — use a real markdown-table parser or Python's csv-safe split, as this session did, not shell text tools).

### The UAT block shape to extend (verified, `.planning/phases/199-front-door-and-classic-contract/199-UAT.md:1-20`)
```markdown
---
status: complete
phase: 199-front-door-and-classic-contract
source: [199-01-SUMMARY.md, ...]
started: 2026-09-06T20:51:26Z
updated: 2026-09-06T22:48:18Z
---

## Current Test

[testing complete]

## Tests

### 1. Guided Help and Current Standing
expected: <plain-English description of what should happen>
result: pass
```
`202-UAT.md` shows the vocabulary extends to `result: skipped` (used for all 3 of its tests, "owner rejected the second-terminal watch flow"). For `205-UAT.md`, D-09 requires extending `result:` to the tri-state "pass / fail / worked, but…" (in the owner's own words) plus D-19's exact phrase "not exercised by the owner; undo path proven only by the program's own tests" for a conditional journey that didn't trigger.

## State of the Art

Not applicable in the conventional sense (no external ecosystem to track). The one relevant "old vs. current" shift is internal: Phase 199 established the machine-signed coverage-JSON pattern in 2026-09 as the successor to purely narrative CAP-row disposition text (as still used, unsigned, in the 200-204 synthesis docs). Phase 205 is the point where that narrative form must be converted to the machine-signed form for the remaining 38 rows (36 + 2 owned by 205 itself).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Defect 5 (wrapper's blanket "Gatekeeper, Auditor, and Probe" dispatch claim) is inaccurate for a typical colony. | Common Pitfalls, Defect 5 | Based on reading `sealReviewRequiredCastes`'s dependency on `queenSealReviewSpecs` and CLAUDE.md's named-risk-signal rule, not on executing a live seal dispatch (out of scope for read-only research). If the actual logic always includes Gatekeeper for phase-199-style colonies, the fix scope shrinks or disappears. |
| A2 | Defect 7's structural gap ("no cross-check between criterion prose and configured commands") will recur on the M4L-AnalogWave-System journeys. | Common Pitfalls, Defect 7 | The target project's actual phase criteria were not read in this session; the gap is confirmed in the *mechanism*, not confirmed to bite on this specific project's specific criteria. |
| A3 | The brief/limitations-card/UAT artifacts are planning documents, not CLI-rendered screens, and therefore do not need voice-corpus registration. | Architecture Patterns, Pattern 4 | If the planner instead decides these should be rendered through an `aether` subcommand (e.g. `aether limitations-card`), voice-corpus registration and its two tests become mandatory, adding scope not currently assumed. |
| A4 | The owner's session-1 opening sequence in Journey 1 will not automatically invoke `/ant-resume` before `/ant-seal`. | Summary; Common Pitfalls, Defect 2 | If the session-start hook or the owner's own habit does invoke `/ant-resume` first (plausible — it's the documented "return to a project" command), defect 2 WILL reproduce live during Journey 1, unlike the "dodge" scenario described. Either way the fix should happen; this assumption only affects whether the pre-fix defect would visibly surface if left unfixed. |

## Open Questions

1. **Does `queenSealReviewSpecs` actually make Gatekeeper dispatch conditional, and on what exact signal?**
   - What we know: `sealReviewRequiredCastes` delegates to it; CLAUDE.md's global rule ties Gatekeeper dispatch to five named risk signals (credentials/auth, payments, release sign-off — the last plausibly always true for a seal).
   - What's unclear: whether "release sign-off" as a signal makes Gatekeeper effectively *always* dispatched at seal time, which would make the field report's actual Auditor+Probe-only observation an anomaly rather than the rule, changing defect 5's scope.
   - Recommendation: the planner or an execution-time worker should read `queenSealReviewSpecs` and `queenSealReviewRequiredCastes`'s actual body (not yet read this session) before scoping the defect-5 fix.

2. **What is the exact form the "one-page brief" (D-18) and "limitations card" (D-15) should take — a markdown file the owner reads, or a rendered CLI screen?**
   - What we know: D-16 says the card "stays in the planning records," CONTEXT.md's Claude's Discretion leaves "the exact shape of the one-page brief" open.
   - What's unclear: whether either needs runtime rendering (and therefore voice-corpus registration) or is pure planning-doc markdown.
   - Recommendation: default to plain planning-doc markdown (simpler, satisfies D-16 directly); only add CLI rendering if the planner has a concrete reason the owner needs it inside a live session rather than handed to him beforehand.

3. **Will Journey 1 as scripted actually reproduce the field report's exact seal-dispatch castes, and does the fix change what the owner sees?**
   - What we know: the target colony's `final-review.json` (22 Aug) shows a prior real review ran; a fresh seal this session will generate a new one.
   - What's unclear: whether the pre-session fixes (especially defect 1's handoff-schema sentence) will actually let a first-attempt seal-finalize succeed without a round-trip, which is itself part of what "worked" vs "worked, but…" should mean for Journey 1's mark.
   - Recommendation: the planner should make "first-attempt seal-finalize succeeds" an explicit acceptance signal for Journey 1, derived from the episode ledger (a single seal episode with no retry entry), not from the owner's subjective impression alone.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `aether` (installed CLI) | Every journey | ✓ | 1.0.78 (binary/hub agree, `aether version --check` returned `ok: true`) | — |
| Claude Code | D-01/D-02 (the only platform that must work) | ✓ (this session runs in it) | — | — |
| OpenCode | D-13 honest card | ✓ | 1.1.63 (matches CONTEXT.md) | — |
| Codex CLI | D-13 honest card | ✓ | 0.154.0 (`codex-cli 0.154.0`, matches CONTEXT.md) | — |
| Target project reachable | All ten journeys | ✓ | `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System` exists, git-tracked, `.aether/` present | — |
| Disk space | Test runs during the session (defect 8) | ✓ (today) | 22 GiB free / 1.8 TiB total (35% used); `go-build` cache 26 GiB | Run `go clean -cache` before the session if free space drops below a few GiB, per the field report's own precedent |
| A concurrent session NOT running on `oracle-reinstate` during publish | D-14 | Unverified at research time — must be checked immediately before publish | — | Publish from a pinned worktree regardless (already the D-14 requirement) |

**Missing dependencies with no fallback:** none identified.

**Missing dependencies with fallback:** disk space (fallback: `go clean -cache`, already used successfully in the field report's own incident).

## Sources

### Primary (HIGH confidence — read directly this session)
- `.planning/phases/205-owner-acceptance-and-restoration-seal/205-CONTEXT.md` — full owner decisions
- `.planning/REQUIREMENTS.md` — SYNTH-07, PROOF-01..06 exact wording; requirement-to-phase table
- `.planning/STATE.md` (partial, first 358 lines — sufficient for milestone/phase status; the file's tail is velocity-log detail not relevant to this research)
- `.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md` — full 8-defect report
- `.planning/research/v1.28-classic-capability-ledger.md` — full 72-row ledger and phase-routing table
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json` and `.md`
- `cmd/classic_coverage_199_test.go` — full file, validator logic
- `.planning/research/v1.28-classic-synthesis-template.md` — full 8-section template
- `.planning/phases/{200,201,202,202.1,203,204}-*/{200,201,202,202.1,203,204}-CLASSIC-SYNTHESIS.md` — headers and CAP-row sections read
- `cmd/entomb_cmd.go`, `cmd/seal_final_review.go`, `cmd/seal_confirmation.go`, `cmd/queen.go`, `cmd/session_flow_cmds.go`, `cmd/codex_visuals.go`, `cmd/criterion_evidence.go` — exact current source for defect re-verification
- `cmd/episode_ledger.go` — full struct and path constant
- `.planning/WINDOWS.md` — full live parse (34 open / 14 fixed / 48 total)
- `.planning/todos/pending/2026-08-01-ts-host-preflight-hardcoded-timeout.md` path confirmed; retirement confirmed via `.planning/phases/203-biological-runtime/203-04-PLAN.md` / `203-04-SUMMARY.md` and `203-CLASSIC-SYNTHESIS.md:83`
- `.planning/phases/199-front-door-and-classic-contract/199-UAT.md`, `201-UAT.md`, `202-UAT.md` — UAT format
- `cmd/classic_voice_corpus_test.go` — voice registration mechanism
- `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.aether/data/COLONY_STATE.json`, `.aether/HANDOFF.md`, `.aether/data/seal/` — target project live state, read directly this session
- `~/.aether/version.json`, `~/.aether/system/version.json`, `aether version --check` — hub/binary version agreement
- `git log`, `git status --porcelain | wc -l`, `df -h` — live repo/target/disk state checked this session

### Secondary (MEDIUM confidence)
- Defect 5's Gatekeeper-conditionality conclusion — inferred from `sealReviewRequiredCastes`'s delegation and CLAUDE.md's documented rule, not from executing a live dispatch this session.

### Tertiary (LOW confidence)
- None — this phase involved no external web/package research; every claim above traces to a file read or command run in this session.

## Metadata

**Confidence breakdown:**
- CAP ledger / coverage-JSON mechanics: HIGH — read the schema, the validator, and the full 72-row ledger directly.
- Field-report defect status: HIGH for defects 1, 2, 3, 4, 6, 7 (exact current source read); MEDIUM for defect 5 (inferred, not executed).
- Target project state: HIGH — read `COLONY_STATE.json`, `HANDOFF.md`, `.aether/data/seal/` directly this session.
- Release/version state: HIGH — `aether version --check` executed, hub file read directly.
- Brief/card/UAT format: HIGH for the existing pattern; MEDIUM for how D-09's tri-state and D-15's card should extend it (Claude's Discretion, not yet decided).

**Research date:** 2026-09-15
**Valid until:** This research is unusually time-sensitive — it depends on the exact current state of `cmd/` (defects), `.planning/WINDOWS.md` (live-updated), and the target project's git/colony state, all of which can change with the next commit or the next session touching `oracle-reinstate` or the target project. **Re-verify defect status, WINDOWS.md count, and target-project git/colony state immediately before planning tasks that depend on them if more than a few days elapse**, and always re-verify immediately before the owner's live session.
