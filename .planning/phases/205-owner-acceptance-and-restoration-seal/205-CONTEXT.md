# Phase 205: Owner Acceptance and Restoration Seal - Context

**Gathered:** 2026-09-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 205 is the walk-through, not more building. ROADMAP.md fixes its goal:
**reconcile every promise to public evidence, prove the complete restored
journey in real repositories, and obtain owner acceptance before sealing
v1.28.** Requirements: SYNTH-07, PROOF-02, PROOF-03, PROOF-04, PROOF-05,
PROOF-06.

Three deliverables, in the order the phase should produce them:

1. **Reconciliation (SYNTH-07, PROOF-02, PROOF-03).** Every Phase 199–204
   synthesis artifact is audited against the research-to-execution ratchet,
   and every one of the 72 `CAP-001..CAP-072` rows ends with exactly one
   evidence-backed disposition signed by its phase's synthesis. Today only
   Phase 199 has signed its 34 rows (`199-CLASSIC-COVERAGE.json`, one named
   passing test per row). Phases 200–204 carry routing tables inside their
   `CLASSIC-SYNTHESIS.md` files (7 + 8 + 8 + 6 + 7 = 36 rows) but none is
   signed into the master ledger; 2 rows are routed to Phase 205 itself
   (`CAP-023` documentation accuracy → PROOF-03, `CAP-054` provider/caste
   readiness → PROOF-04); Phase 202.1 owns no CAP rows; and Phase 204's
   synthesis "Ruling (c)" says two frozen dispositions are stale and must be
   re-adjudicated. Missing, duplicate, or unsupported dispositions must fail
   a command, per the Definition of Done. This is engineering the program
   does and checks — the owner is never asked about a CAP row.
2. **The owner-driven walk-through (PROOF-04, PROOF-05).** Ten journeys in a
   real project, run by the owner alone, with the program's own records as
   the evidence — see D-01..D-19.
3. **Acceptance and seal (PROOF-06).** The owner's verdicts, a plain-words
   limitations card, and — only on an overall "yes" — the milestone seal.

**Out of scope:** new capabilities; making OpenCode or Codex work (only an
honest card, D-13); Codex "skills as commands" integration (deferred);
closing the 34 open WINDOWS.md entries (they are listed, not fixed, unless
a journey fails on one — D-10); a second discovery interview about the
Classic era (the comprehensive report and ledger already hold those answers).

</domain>

<decisions>
## Implementation Decisions

### How the walk-through runs (owner, 2026-09-15)

- **D-01:** The owner drives **all nine PROOF-05 journeys plus the opening
  seal (ten in total) himself.** Claude is not in the room. No split, no
  Claude-driven journeys with reports.
- **D-02:** **One session, all ten in order, in one fresh Claude Code chat**
  opened in the target project — a real working day, not one chat per
  journey and not this planning chat. The pause/resume journey is the only
  point where that chat is deliberately closed and reopened.
- **D-03:** **Evidence is what the program itself recorded.** After the
  session Claude reads the target project's own records — the durable
  episode ledger (cost, elapsed time, decisions, how each run ended — Phase
  204 D-07), the live event trail, colony state, worker handoffs, the
  failure log — and the owner's verdicts. The owner pastes nothing. Every
  journey must therefore leave a durable, read-back-able record; a journey
  whose run left no record is a finding against the program, not a gap in
  the evidence plan.
- **D-04:** **Stuck means that journey fails.** The owner stops, notes where
  he got lost, and moves on. No nudge, no rescue, no explanation mid-session.
  The confusion itself is the finding ("if I can't figure it out, it's not
  done").

### Where the journeys run (owner, 2026-09-15)

- **D-05:** Target project: **`/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System`** —
  the owner's real Max for Live device-builder project (Python `m4l_builder/`
  + `.mds/` tooling, pytest, `.aether/`, `.claude/`, `.opencode/`, `.codex/`
  all present). Not a copy, not a throwaway. Note the **space in the path**:
  the program must cope; treat it as a real-world stress, not something to
  work around by relocating the project.
- **D-06:** **Tidy first.** Before the session, the 764 in-flight files on
  branch `codex/skills-as-commands` (352 modified, 231 untracked, 181
  deleted — the owner's own Codex skills-as-commands experiment) are saved
  as a snapshot so nothing is lost and everything is reversible, and the
  session starts from a **clean working tree**. The "change with unsaved
  edits in the way" journey then uses **one deliberate small edit the owner
  makes himself**, not the swamp. — **Reversibility:** reversible — a
  snapshot commit/branch is the whole mechanism; the in-flight work must
  remain retrievable by name and the brief must say which branch the
  session runs on.
- **D-07:** The existing colony there is **finished (3/3 phases, 20/20
  tasks) but never sealed.** The owner **seals and archives it himself as
  the first act of the session** (the tenth journey: finish-and-archive,
  restored in Phase 199), then starts the new project through the front
  door. Claude does not pre-archive it.
- **D-08:** The new project's goal, in the owner's words: *"improving the
  power of the system (mds system) to create much more powerful ableton
  devices and midi devices like complex sequencers and ableton osc devices,
  like how people have made the ableton track colours one."* The brief
  states the goal in those terms; the owner types it in his own words on the
  day.

### The acceptance bar (owner, 2026-09-15)

- **D-09:** **One mark per journey, then one overall verdict.** Each of the
  ten gets pass / fail / "worked, but…" in the owner's own words. Then one
  overall answer: *"feels like Aether again and I trust it to operate"* —
  yes or no. **Only the overall yes seals the milestone; a tally never
  does.** The program may compute the tally; it may not compute the seal.
- **D-10:** **On a failed or stuck journey: fix what is fixable inside this
  phase; the owner re-runs only the failed journeys in a fresh chat.**
  Anything not fixable inside the phase goes on the limitations card in
  plain words and the owner decides whether to seal with it explicit.
  Nothing is sealed with a hidden failure; nothing forces a repeat of the
  whole session; no automatic full reopen of the phase.
- **D-11:** **Speed is a pass/fail criterion for the three everyday
  journeys** — the clean small change, the change with a deliberate unsaved
  edit in the way, and pause/resume. **Limit: 10 minutes** from the owner's
  ask to being told "built and checked", **measured from the program's own
  run record**, never a stopwatch. A journey over the limit fails on time
  alone even if the work was correct. The long journeys (iterative plan,
  stubborn defect, multi-round research, recruitment, owner-only work,
  rollback, seal) record their time on the card but are never failed for
  being long.
- **D-12:** Every journey's **elapsed time and cost** appear on the card
  (the worker-turnaround note folded as a measurement, see Folded Todos).

### Platforms and the release (owner, 2026-09-15)

- **D-13:** **Claude Code is the platform that has to work and the only one
  the owner watches.** Owner's words: *"later i want to make codex work with
  aether using $ skills as commands but for now maybe just make it work with
  claude first."* For OpenCode (1.1.63 installed) and Codex (0.154.0
  installed) this phase does the PROOF-04 minimum and nothing more: the
  program runs the real public entry points once per platform and writes one
  honest card each — what works, what stops early with a truthful message,
  what is not supported. **No engineering effort to make either work**; no
  claim is made for either that a real run did not show.
- **D-14:** **The pre-session publish is the release.** Because the hub copy
  the owner's projects use (v1.0.78, published 13 Sep) predates the
  completion of Phases 203 and 204, a publish is mandatory before the
  session; the owner chose that this one publish *is* the v1.28 release,
  with the seal afterwards recording acceptance. Fixes after a failed
  journey ship as a follow-on release. — **Reversibility:** costly — the
  release lands in every project on the machine before acceptance; a journey
  that fails means a broken version is live until a follow-on publish, so
  the publish must bump the version, run version-sync, back up the hub, and
  pass `aether integrity` (CLAUDE.md "Publishing Changes"; memory: never
  republish under the same number; publish from a worktree pinned to the
  release commit because `oracle-reinstate` is a shared checkout).
- **D-15:** The limitations card the owner signs lists **every open
  WINDOWS.md entry (34 at the time of discussion) in plain words**, plus the
  folded notes (D-20, D-22), plus every journey marked "not exercised"
  (D-17, D-19). Not a filtered subset, not a count.
- **D-16:** The card **stays in the planning records** (this phase's
  directory and the milestone audit). It does not go into the release notes
  or CHANGELOG.

### The ten concrete tasks (owner, 2026-09-15)

- **D-17:** **The stubborn defect is whatever the session turns up** — the
  first real failure the program's own checks hit during the build journeys
  becomes the bug the four-lens investigation chases. Nothing is planted. If
  nothing breaks, that journey is recorded as **"not exercised"** on the card.
- **D-18:** **The brief is one page: the task behind each journey, in order,
  and nothing about which command to type.** The program's own what-next
  card and help must carry the owner from there; if they cannot, that is a
  finding about Aether. "Stuck = fail" applies to the product, never to
  "what should I ask for".
- **D-19:** **The learned-change rollback journey is exercised only if a
  learned change was actually bad.** Nothing is pre-declared to guarantee a
  rollback. If no bad learned change arises, the card records **"not
  exercised by the owner; undo path proven only by the program's own tests"**
  — explicit deferred proof, never a green tick.

### The ten journeys (the contract the brief and the evidence plan realise)

| # | Journey (PROOF-05 / ROADMAP SC3) | Kind | In this project |
|---|---|---|---|
| 1 | Seal and archive the finished old project, then start the new one (D-07) | opening | `/ant-seal` → archive → init with the D-08 goal |
| 2 | A clean small change | everyday, 10-min limit | a builder-tooling tweak toward the goal |
| 3 | A change with a deliberate unsaved edit in the way (D-06) | everyday, 10-min limit | owner edits one file first, then asks |
| 4 | Iterative planning | long | the multi-phase plan for the D-08 goal |
| 5 | A stubborn defect (D-17) | long | whatever the checks turn up |
| 6 | Multi-round research | long | a real question from the goal, e.g. how track-colours / OSC devices are built and what the builder must produce |
| 7 | Pause/resume and interruption recovery | everyday (resume), 10-min limit | close the chat mid-run, reopen, resume |
| 8 | Governed recruitment / result use | long | a helper asks for backup; owner sees the inline line and the family tree |
| 9 | Queued physical or authority work | long | trying a generated device inside Ableton — only the owner can do it |
| 10 | A learned change rolled back (D-19) | long, conditional | only if a learned change was bad |

### Claude's Discretion

- **Reconciliation mechanics:** how the 36 unsigned rows and the 2 Phase-205
  rows are signed (extending the `199-CLASSIC-COVERAGE.json` + named-test
  pattern to 200–204 is the obvious route), how the ratchet command that
  fails on a missing/duplicate/unsupported disposition is built, how the
  SYNTH-07 audit of the seven synthesis artifacts is scored, how PROOF-03
  four-dimension parity is evidenced per slice, and how 204's "Ruling (c)"
  re-adjudications are recorded.
- **Tidy mechanics (D-06):** snapshot commit vs. WIP branch, and which branch
  the session runs on — constrained only by "nothing lost, fully reversible,
  clean tree at start, branch named in the brief".
- **Brief and verdict collection:** the exact shape of the one-page brief,
  how the owner records his ten marks and the overall verdict, and how that
  becomes `205-UAT.md`.
- **Evidence read-back:** which records are read for each journey and how
  elapsed time is derived from the episode ledger for D-11.
- **Platform cards (D-13):** how the OpenCode and Codex honest runs are
  driven and rendered.
- **Interruption trigger:** how the brief phrases "interrupt a run" as a
  task without naming a command.
- **Publish sequence details (D-14)** within the CLAUDE.md runbook.
- **Grouping into plans and waves.**

### Folded Todos

- **D-20 — "Nothing proves who is calling"**
  (`.planning/todos/2026-09-13-nothing-proves-who-is-calling.md`): the openly
  recorded Phase 203 gap — the depth check trusts a caller's self-declared
  coordinator name. Folded as **a named line on the limitations card**, not
  as work.
- **D-21 — "Worker turnaround is too slow"**
  (`.planning/todos/2026-08-27-worker-turnaround-is-too-slow.md`): folded as
  **a measurement** — every journey's elapsed time on the card, and the
  10-minute limit of D-11 for the everyday journeys.
- **D-22 — "ts-host preflight timeout hardcoded, runs in repo cwd"**
  (`.planning/todos/2026-08-01-ts-host-preflight-hardcoded-timeout.md`):
  routed to Phase 203 / BIO-03. Folded **only as a limitation line if the
  planner finds it still open**; if BIO-03 retired it, cite the retiring
  test and leave it off.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The phase's own contract
- `.planning/ROADMAP.md` §"Phase 205: Owner Acceptance and Restoration Seal" — goal, five success criteria, and §"Guardrails and exclusions".
- `.planning/REQUIREMENTS.md` — SYNTH-07 (line 29), PROOF-02..PROOF-06 (lines 107–111); PROOF-01 (line 106) is already satisfied and defines the fixture vocabulary the others build on.
- `.planning/research/v1.28-classic-synthesis-template.md` — what every synthesis artifact must contain (the SYNTH-07 audit rubric).

### The 72-row reconciliation
- `.planning/research/v1.28-classic-capability-ledger.md` — the frozen 72 rows (§"Frozen 72-row ledger"), the owner-approved routing table (§"v1.28 requirement and phase mapping"), and the only signed evidence so far (§"Phase 199 signed implementation evidence").
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json` and `199-CLASSIC-COVERAGE.md` — the machine-readable signing pattern (row → disposition → modern home → plan task → artifact → named proof → status) to extend to 200–204.
- `cmd/classic_coverage_199_test.go` and `cmd/classic_contract_test.go` — the existing ratchet tests over CAP rows; the new ratchet must not duplicate a second vocabulary beside them.
- `.planning/audits/2026-08-30-whole-system-audit.json` — the source of the 72 rows (`confirmed[].status == "dropped"`), with the full claim and old/current evidence per row.
- `.aether/dreams/2026-09-01-comprehensive-aether-colony-review.md` — the authoritative owner brief the ledger defers to; where a row's disposition is genuinely unresolved, this report answers before the owner is asked.

### The seven synthesis artifacts to audit (SYNTH-07) and their verification
- `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-SYNTHESIS.md`, `199-UAT.md` (12/12 passed)
- `.planning/phases/200-iterative-planning/200-CLASSIC-SYNTHESIS.md`, `200-VERIFICATION.md` (no UAT file)
- `.planning/phases/201-queen-led-work-cycle/201-CLASSIC-SYNTHESIS.md` §"CAP row routing table", `201-VERIFICATION.md`, `201-UAT.md` (74/74 passed)
- `.planning/phases/202-swarm-oracle-and-live-colony/202-CLASSIC-SYNTHESIS.md` §"CAP row routing table", `202-VERIFICATION.md`, `202-UAT.md` (0/3 — all skipped: owner rejected the second-terminal watch flow)
- `.planning/phases/202.1-classic-visual-voice/202.1-CLASSIC-SYNTHESIS.md`, `202.1-VERIFICATION.md` (no CAP rows)
- `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md` §"CAP row routing table", `203-VERIFICATION.md` (no UAT file)
- `.planning/phases/204-learning-governor/204-CLASSIC-SYNTHESIS.md` (§"Ruling (c)" — two stale dispositions to re-adjudicate), `204-CLASSIC-HISTORICAL-EVIDENCE.md`, `204-VERIFICATION.md` (11/11 with two owner-accepted deferrals, 2026-09-15; no UAT file)
- `.planning/phases/203-biological-runtime/203-CONTEXT.md` D-04 — the standing hard constraint: the owner never uses a second terminal; all liveness renders inline.

### Limitations and folded notes
- `.planning/WINDOWS.md` — 48 entries, 34 open at discussion time; every open entry goes on the card (D-15).
- `.planning/todos/2026-09-13-nothing-proves-who-is-calling.md` (D-20)
- `.planning/todos/2026-08-27-worker-turnaround-is-too-slow.md` (D-21)
- `.planning/todos/2026-08-01-ts-host-preflight-hardcoded-timeout.md` (D-22)

### The target project (D-05..D-08)
- `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.aether/data/COLONY_STATE.json` — the finished, unsealed colony (goal: vault-as-source-of-truth; 3/3 phases complete; `parallel_mode: in-repo`).
- `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/HANDOFF.md` — the March backlog (stale device names; useful for task ideas only).
- `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/CLAUDE.md`, `AGENTS.md`, `.mds/`, `m4l_builder/`, `docs/VAL_RULES_BACKLOG.md` — what "the MDS system" is; the brief's tasks are drawn from here, never invented.

### Rules that bind this phase
- `CLAUDE.md` — "Definition of Done" and "How much proof a change needs" (this phase decides work is complete: full rigour, both lanes); "Communication Style" (the brief, the card, and every owner-facing screen in plain English); "Verification Commands" (`-timeout 90m`, check `discovered==executed`); "Publishing Changes" and `.aether/docs/publish-update-runbook.md` (D-14).
- `.aether/docs/wrapper-runtime-ux-contract.md` — wrappers never mutate state; anything the walk-through touches must hold on the wrapper path the owner actually types.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`199-CLASSIC-COVERAGE.json` + `cmd/classic_coverage_199_test.go`** — the only working "signed row" mechanism; extend, don't reinvent.
- **Episode ledger (`cmd/episode_ledger.go`, Phase 204)** — every build/continue/plan/oracle/swarm/recovery run leaves a durable record with cost, elapsed time, decisions, interventions and how it ended, outliving the live feed's 30-day window. This is the evidence source for D-03 and the clock for D-11.
- **`aether watch` replay branch** — renders the most recent episode from the same event trail; usable to reconstruct the session after the fact.
- **`aether improve`** — read-only report of what the program tried, promoted, or rolled back (journey 10 / D-19).
- **`aether publish`, `aether integrity`, `aether version --check`, `make version-sync`** — the D-14 release path, already runbooked.
- **Prior `*-UAT.md` files and `/gsd-verify-work`** — the verdict format (expected / result / reason per test, summary counts) the owner's ten marks should land in.
- **Voice corpus tests (`TestVoicedScreensSpeakPlainEnglish`, `TestVoicedScreensCarryNoRawStateToken`)** — any new owner-facing card (limitations card, platform cards) must pass them.

### Established Patterns
- Wiring tests assert the real call site, never an intermediate record; allowlists and floors only shrink; dry-run/inspection commands never mutate (the ratchet and the cards are inspection surfaces).
- The full `cmd` suite needs ~20 minutes and `-timeout 90m`; an unqualified run truncates silently. Concurrent sessions on `oracle-reinstate` corrupt gates — serialise suites and publish from a pinned worktree.
- A known-red baseline exists (memory 2026-09-14: 17 pre-existing failures); check it before calling a red test a regression.
- Owner-facing text: every repo word translated inline, no IDs where a sentence belongs (CLAUDE.md top block).

### Integration Points
- The master ledger and per-phase coverage JSONs (PROOF-02 signing).
- `REQUIREMENTS.md` checkboxes for SYNTH-07 and PROOF-02..06; `WINDOWS.md` (read for D-15, written only if a journey fix closes an entry).
- `CHANGELOG.md` / version files for D-14 (limitations card explicitly excluded, D-16).
- GSD milestone closeout (`/gsd-audit-milestone`, `/gsd-complete-milestone`) after the overall yes — the GSD seal follows the owner's acceptance, never precedes it.
- The target project's `.aether/data/` (read-only evidence source after the session; the program's own commands are the only writer during it).

</code_context>

<specifics>
## Specific Ideas

- Owner's goal for the new project, verbatim: *"Its about improving the power of the system (mds system) to create much more powerful ableton devices and midi devices like complex seqeuncers and ableton osc devices, like how people have made the ableton track colours one etc..."*
- Owner on platforms, verbatim: *"later i want to make codex work with aether using $ skills as commands but for now maybe just make it work with claude first."*
- Phase 202's walk-through was skipped in full because it required a second terminal; Phase 203 D-04 made inline-only a hard constraint. Journeys 5, 6 and 8 must be visible inline in the chat the owner is using — a journey that only shows life in `aether watch` fails D-04's spirit.
- The owner's stated bar, from memory: "as snappy as plain Claude"; a one-plan job at ~30 minutes is why he stopped using Aether. D-11's 10-minute limit is the operational form of that bar.
- The hub the owner's projects use is v1.0.78 published 13 Sep; Phases 203 (14 Sep) and 204 (15 Sep) are not in it. Without D-14's publish the session would test stale code.
- The target path contains a space (`Max 9`). Earlier real-use sessions in downstream repos (CosmicDashboard) surfaced genuine planning-pipeline bugs; this is the same class of evidence — chase the symptom, not the owner's diagnosis.
- Ledger state at discussion: 34 rows signed (all Phase 199), 36 routed to 200–204 and unsigned, 2 routed to 205, 3 marked delivered in Phase 198.3 (`CAP-010`, `CAP-046`, `CAP-051` — verify they are signed, not merely labelled).

</specifics>

<deferred>
## Deferred Ideas

- **Codex driven by "skills as commands"** — the owner wants this later; ROADMAP already parks Codex lifecycle-skill work for a later milestone. The in-flight `codex/skills-as-commands` branch in the target project is the owner's own experiment and is preserved by D-06, not adopted.
- **Making OpenCode or Codex actually work end-to-end** — beyond the honest card (D-13), not this phase.
- **Closing the open WINDOWS.md entries** — listed on the card (D-15); fixed only where a journey fails on one (D-10).
- **Guarding the remaining 30 fixture-bank entries; check-episode token/cost capture** — the two Phase 204 owner-accepted deferrals (WINDOWS 42, 48); they appear on the card, they are not reopened here.

### Reviewed Todos (not folded)
- **Spec builder — a readable specification before building** (`.planning/todos/2026-08-20-spec-builder-feature.md`) — routed to Phase 200 / PLAN-05; a capability, not proof work. Left out by the owner.

</deferred>

---

*Phase: 205-owner-acceptance-and-restoration-seal*
*Context gathered: 2026-09-15*
