# Phase 198: Put the Thrown-Away Data Back on Screen - Context

**Gathered:** 2026-08-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore the detail the older (v5.4.0) Aether showed and the current program already
calculates but drops before display — per-check verification results, evidence per
requirement, worker durations and tool counts, plan confidence and iterations, resume
per-phase progress, recent decisions, drift warning, specialist finding blocks, and a
blocker heads-up before build — so that the chat view (Claude Code / OpenCode) shows
exactly what the direct command-line view shows for plan, continue and seal. Add the
three genuine gaps: live verification progress during `continue`, a seal confirmation,
and the wisdom review before sealing. Keep the three deliberately-dropped things out
and lock that with a test.

Out of scope: changing what any command *does*; new commands or menu entries; any
repaintable panel, second terminal, or bordered table; anything from the v1.27
"explicitly not in this milestone" list.

</domain>

<decisions>
## Implementation Decisions

All decisions below were made by the owner on 2026-08-29 unless marked (Claude).

### Live progress while checking (SHOW-03)
- **D-01:** Each verification check in `continue` emits a **start line when it begins and a result line when it ends** — e.g. `Running tests…` then `Tests ✓ 12/12 (4s)`. Two lines per check, append-only, in the one terminal (per SEE-03). Never a single in-place updating line.
- **D-02:** A failed check's result line carries a **one-line plain-English reason** inline (`Tests ✗ 2 of 12 failed — login test, export test`). Full detail waits for the closing summary.
- **D-03:** Reviewer workers (Watcher, Auditor, Probe, Gatekeeper…) get the **same live start/finish treatment**, and the finish line carries the duration and tool-call count (`Watcher Keen-12 done (3m 10s, 14 tool calls)`). These are the same duration/tool-count figures SHOW-02 restores in the summary; live lines and summary must read from one source.

### Finishing a project — seal (SHOW-04)
- **D-04:** Before asking "finish this project?", seal shows a **short state-of-play card**: phases done vs total, any checks still failing, any open warnings/flags, and what finishing will do (archive, pool lessons into the hive). Not the full closing summary, not a bare y/n.
- **D-05:** The wisdom review ("what did we learn") runs **before** the confirmation question. Its lessons are recorded even if the owner then says no; saying no leaves the project unsealed.
- **D-06:** If something is failing or unresolved at seal time, the card **names it and asks a second, explicit question** (`Finish anyway with 2 checks failing?`). A yes is recorded together with the named problem, through the same owner-decision mechanism as reviewer waivers — never inferred from card wording. Seal is never silently refused and never warned-only.
- **D-07:** Autopilot **never seals**. It runs up to the last phase, then stops and hands the owner the finish command. — **Reversibility:** costly — it is a documented runtime behaviour a later "hands-off to the end" feature would have to overturn in writing, and the seal-confirmation test must fail if autopilot ever reaches the seal path.

### Heads-up before a build (SHOW-02, blocker advisory)
- **D-08:** When a build starts with a blocker present, the program prints a plain-English heads-up naming it and **asks one question: carry on, or stop and deal with it**. It proceeds as answered. Not warn-and-continue, not refuse.
- **D-09:** "Blocker" means **hard stops only**: the last `continue` on this phase ended blocked/failed, a forced reviewer waiver still waiting on the owner, or an open owner question (the same three "waiting on you" records Phase 197's check-in logic already recognises). FOCUS/FEEDBACK notes and ordinary flags do not trigger it. `--no-checkin` / autopilot paths print the heads-up and continue without asking (they are non-interactive by ruling).

### How much detail by default (SHOW-01, SHOW-02)
- **D-10:** Every restored item appears **every time, in the compact house style** (SEE-12: emoji heading + plain-English parenthetical, one line per item, nested `└──` detail capped per category with an honest `(+N more)`). No separate "show more" command is added.
- **D-11:** Evidence lines read **requirement + what proved it**: `✓ Login works — proved by: 3 tests passed, auth.go present`. Never a bare tick.
- **D-12:** The chat path and the direct path render **one screen from one renderer**: `ceremony closeout` must produce byte-equal output to the direct visual for plan, continue and seal (`TestWrapperPathRendersSameCeremonyAsDirectPath`). The fix is structural — the finalizers' JSON result map is the single source and the closeout renders all of it — not a second copy of each finalizer's screen.

### Carried forward — binding, not re-asked
- **SEE-03 (2026-08-16):** one terminal, append-only lines, no tmux/second window, no repaintable panel. Live progress in D-01/D-03 goes through `emitVisualProgress` / `writeVisualOutput`.
- **SEE-12 (2026-08-16):** headed sections not machine tables; `TestHumanDisplaysUseHeadedSectionsNotMachineTables` and its shrink-only allowlist still apply to every new render.
- **"Verified safe-to-clear line" stays; the old "clear context now?" prompt stays out.** SHOW-05 locks all three drops with a test.
- **Phase 196 D-01:** duration/tool-count/token figures are measured or shown as `—  not reported`; never estimated.
- **Phase 197 S-01/S-04/S-05:** next-step commands come from the shared "what next" logic and are translated per platform only in the visual writer; wrapper triplets are edited byte-identically; all owner-facing text is plain English with repo words translated inline.

### Claude's Discretion
- Exact wording, ordering and emoji of every heading and line, within SEE-12.
- How many "recent decisions" the resume card shows and how the drift note is phrased.
- Where the per-check progress hook lives inside the continue verification loop, provided both the default fast path and the heavy-review path emit it (a guarantee that holds only on one lane is worth nothing — CLAUDE.md).
- Test design for `TestRenderedVisualsShowEveryCarriedField`: it must be an invariant over the finalizer result map (every carried key is rendered or explicitly allow-listed with a reason, allowlist shrink-only), not a named-section check.

### Folded Todos
- **Worker turnaround is too slow** (`.planning/todos/2026-08-27-worker-turnaround-is-too-slow.md`): the new confirmation, wisdom review and blocker question must not add noticeable waiting or extra worker spawns. The wisdom review reuses the existing seal consolidation pass; it does not dispatch a new agent.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### What to restore and what to keep out
- `.planning/research/v1.27-milestone-brief.md` §"6. Put the thrown-away data back on screen" and §"Classic-vs-current display audit (2026-08-22)" — the itemised list of gaps, weakened items, and the deliberately-dropped set.
- `.planning/ROADMAP.md` §"Phase 198" — goal, five success criteria, the two named tests.
- `.planning/REQUIREMENTS.md` SHOW-01…SHOW-05.

### Standing display rulings
- `.planning/decisions/SEE-03-one-terminal-streaming.md` — one terminal, append-only, no panel.
- `.planning/decisions/SEE-12-13-display-standards.md` — headed sections, compact nested detail, summaries explain themselves.
- `.planning/phases/196-see-what-it-cost/196-CONTEXT.md` D-01 — measured-only figures, `—  not reported`.
- `.planning/phases/197-one-answer-to-what-next/197-CONTEXT.md` S-01…S-06 — command-naming chokepoint, shrink-only baselines, triplet parity, plain English.
- `CLAUDE.md` §"Definition of Done" and §"How much proof a change needs" — full rigour for anything that decides or credits; a test must be able to fail; fixtures in runtime shape.

### Wrapper/runtime contract
- `.aether/docs/wrapper-runtime-ux-contract.md` — wrappers frame, runtime owns truth and rendering.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `renderCeremonyCloseoutVisual` (`cmd/ceremony_cmd.go:544`) — the thin chat-path screen; already reads the finalizer result map (`goal`, `completion_phase`, worker/plan summaries, suggestions). Extend it to render everything the map carries rather than build a new renderer.
- Finalizers that own the richer direct screens: `runCodexContinueFinalize` (`cmd/codex_continue_finalize.go:139`), `runCodexBuildFinalize` (`cmd/codex_build_finalize.go:429`), plan finalize (`cmd/codex_plan_finalize.go`), seal finalize. Confidence/iterations rendering already exists at `cmd/codex_visuals.go:1488-1500` but only on the direct path.
- `emitVisualProgress` / `writeVisualOutput` (`cmd/codex_visuals.go`) — the append-only streaming channel; already used in `cmd/codex_continue.go:614-616`. Live check lines (D-01…D-03) go through this.
- `renderDecisionBlock`, `renderStageMarker`, `renderIndentedList` — house-style frames for the seal card (D-04) and blocker question (D-08).
- Phase 197's "what next" resolver and the three "waiting on you" records used by `TestBuildCheckinDecisionMatrix` — reuse for D-09's blocker definition and for D-06's recorded seal override.

### Established Patterns
- Wrappers call `AETHER_OUTPUT_MODE=json aether <workflow>-finalize` then `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow <w> --completion-file …` (`.claude/commands/ant/plan.md:140-148`, `continue.md:196-202`). SHOW-01 is closed by making the second call render the first call's full result.
- Shrink-only allowlists with a named ratchet test (`orphan_allowlist.json`, 197 S-03) — the pattern for `TestRenderedVisualsShowEveryCarriedField`'s exceptions.
- Owner decisions recorded via `decision-answer` / waiver mechanism, never inferred from rendered text.

### Integration Points
- `.claude/commands/ant/{plan,continue,seal}.md` + `.opencode` + flat mirrors — byte-identical triplets; any wrapper change lands in all copies in one plan.
- `aether run` (autopilot) — must stop before seal (D-07) and print, not ask, the build heads-up (D-09).
- Seal path — confirmation card and wisdom review precede state mutation; `--dry-run` inspection must not mutate (CLAUDE.md corollary).

</code_context>

<specifics>
## Specific Ideas

- Live check lines: `Running tests…` → `Tests ✓ 12/12 (4s)`; failure: `Tests ✗ 2 of 12 failed — login test, export test`.
- Reviewer finish line: `Watcher Keen-12 done (3m 10s, 14 tool calls)`.
- Evidence line: `✓ Login works — proved by: 3 tests passed, auth.go present`.
- Seal second question when blocked: `Finish anyway with 2 checks failing?`
- Build heads-up: one question — carry on, or stop and deal with it.

</specifics>

<deferred>
## Deferred Ideas

### Reviewed Todos (not folded)
- **Spec builder command** (`.planning/todos/2026-08-20-spec-builder-feature.md`) — the owner ticked it during review, but it is a new command explicitly parked for v1.28+ in the milestone brief; recorded here so it is not forgotten, not in scope for a display phase.
- **ts-host preflight timeout hardcoded** (`.planning/todos/2026-08-01-ts-host-preflight-hardcoded-timeout.md`) — ticked during review; unrelated to display, stays low-priority in the backlog.

</deferred>

---

*Phase: 198-put-the-thrown-away-data-back-on-screen*
*Context gathered: 2026-08-29*
