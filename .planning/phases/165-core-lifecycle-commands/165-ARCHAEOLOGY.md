# Phase 165 Archaeology — Classic Ceremony Inventory (v5.4.0 → HEAD)

> Produced by the aether-archaeologist agent on 2026-08-02 for the Phase 165
> wrapper rewrite. Canonical input for planning: mine ceremony text from the
> playbooks, inline it, never restore the forbidden mechanisms in §D.

**Tag:** `v5.4.0` = `95f18baf` (Sun Apr 5 2026) · **Change zone:** the four lifecycle wrappers · **Commits on those four files since v5.4.0:** 52

## Structural finding that reframes the task

The premise "v5.4.0 wrappers had rich ceremony" is **half true, and the half that is false matters most**:

| File @ v5.4.0 | Lines | Shape |
|---|---|---|
| `.claude/commands/ant/init.md` | 516 | Rich, inline, self-contained |
| `.claude/commands/ant/plan.md` | 693 | Rich, inline, self-contained |
| `.claude/commands/ant/build.md` | **65** | **Thin pointer to 5 playbooks** |
| `.claude/commands/ant/continue.md` | **60** | **Thin pointer to 4 playbooks** |

Classic build/continue ceremony did not live in the wrapper. It lived in `.aether/docs/command-playbooks/` (2,160 lines across build-prep/context/wave/verify/complete; 2,252 lines across continue-verify/gates/advance/finalize). The v5.4.0 wrapper said only:

> ```
> ## Stage Order
> 1. `.aether/docs/command-playbooks/build-prep.md`
> ...
> ## Execution Contract
> For each stage:
> 1. Read the file with the Read tool.
> 2. Execute the instructions exactly as written.
> ```

**Planner consequence:** to restore build/continue ceremony you must mine the playbooks, not `build.md@v5.4.0`, and you must inline what you take. The playbook files still exist at `.aether/docs/command-playbooks/` (unchanged, now unloaded) — they are the source text, but the loading mechanism is forbidden (see D-1).

---

## A. Ceremony element inventory

### A-1. Queen voice patterns

| # | File @ v5.4.0 | Verbatim |
|---|---|---|
| 1 | `build-prep.md:6` | `You are the **Queen**. You DIRECTLY spawn multiple workers — do not delegate to a single Prime Worker.` |
| 2 | `build-wave.md:14` | `**YOU (the Queen) will spawn workers directly. Do NOT delegate to a single Prime Worker.**` |
| 3 | `init.md:499-501` | `👑 Queen has set the colony's intention` + `   "{approved_intent}"` |
| 4 | `continue-gates.md:19-22` | `🐜 The colony requires actual parallelism:` / `  - Prime Worker MUST spawn specialists for non-trivial work` / `  - A single agent doing everything is NOT a colony` / `  - "Justifications" for not spawning are not accepted` |
| 5 | `continue-gates.md:49-52` | `🐜 Why this matters:` / `  - Builders verify their own work = confirmation bias` / `  - Independent Watchers catch bugs builders miss` / `  - "Build passing" ≠ "App working"` |
| 6 | `continue-finalize.md:385` | `🐜 The colony rests. Well done!` |
| 7 | `plan.md:7` | `You are the **Queen**. Orchestrate research and planning until the selected confidence target is reached within the selected iteration budget.` |

The distinguishing feature of Classic Queen voice: **it explains the reason for a rule in colony terms at the moment the rule fires** (#4, #5), rather than stating the rule abstractly.

### A-2. Caste identity moments

**Spawn plan block** — `build-wave.md:46-74`:
```
━━━ 🐜 S P A W N   P L A N ━━━

Wave 1  — Parallel
  🔨🐜 {Builder-Name}  Task {id}  {description}
  🔨🐜 {Builder-Name}  Task {id}  {description}

Wave 2  — After Wave 1
  🔨🐜 {Builder-Name}  Task {id}  {description}

Verification
  👁️🐜 {Watcher-Name}  Verify all work independently
  {if colony_depth == "full": 🎲🐜 {Chaos-Name}   Resilience testing (after Watcher)}

Total: {N} Builders + 1 Watcher{...} = {total} spawns
```
followed by a **Caste Emoji Legend** (`build-wave.md:64-77`) with the rule `**Every spawn must show its caste emoji.**`

**Per-caste spawn announcement** — a two-line banner+dispatch idiom repeated for every specialist caste:
```
━━━ 🔮 O R A C L E   R E S E A R C H ━━━            (build-wave.md:97-98)
──── 🔮🐜 Spawning {oracle_name} — Phase {phase_id} research ────

━━━ 🏛️ A R C H I T E C T   D E S I G N ━━━          (build-wave.md:171-172)
━━━ 🏺🐜 A R C H A E O L O G I S T ━━━               (build-context.md:230-231)
━━━ 👁️🐜 V E R I F I C A T I O N ━━━                 (build-verify.md:20-21)
━━━ 📊🐜 M E A S U R E R ━━━                          (build-verify.md:146-147)
━━━ 🔌🐜 A M B A S S A D O R ━━━                      (build-wave.md:310-311)
━━━ 🧪🐜 P R O B E ━━━                                (continue-verify.md:183)
━━━ 🔄🐜 W E A V E R ━━━                              (continue-gates.md:191)
━━━ 📦🐜 G A T E K E E P E R ━━━                      (continue-gates.md:308)
━━━ 👥🐜 A U D I T O R ━━━                            (continue-gates.md:401)
```

**Wave announcement variants** — `build-wave.md:258-270`:
```
──── 🔨🐜 Spawning {N} Builders in parallel ────
──── 🐜 Spawning {N} workers ({X} 🔨 Builder, {Y} 🔍 Scout) ────
──── 🔨🐜 Spawning {ant_name} — {task_summary} ────
```

**Worker-in-prompt identity** — every worker prompt opened with its own caste line (`build-wave.md:477`, `:323`, `:106`, `:181`, `build-context.md:240`):
```
You are {Ant-Name}, a 🔨🐜 Builder Ant.
You are {Ambassador-Name}, a 🔌 Ambassador Ant.
You are {oracle_name}, a 🔮 Oracle Ant.
You are {architect_name}, a 🏛️ Architect Ant.
You are {Archaeologist-Name}, a 🏺🐜 Archaeologist Ant.
```

**Immediate per-worker completion line** — `build-wave.md:582-592`, with the explicit streaming instruction:
> `**As each worker result arrives, IMMEDIATELY display a single completion line — do not wait for other workers:**`
```
🔨 {Ant-Name}: {task_description} ({tool_count} tools) ✓
🔨 {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)
```

**Per-worker skill line** — `build-wave.md:463-466`:
```
  🧠 Skills: {colony_count} colony + {domain_count} domain loaded for builder
```

### A-3. Stage / section framing

Three distinct separator weights, used consistently:

| Weight | Glyph | Use | Example |
|---|---|---|---|
| Heavy | `━━━━━━ (50)` | Command-level banner | `build-prep.md:277-279` |
| Medium | `━━━ … ━━━` | Stage marker | `build-wave.md:48` |
| Light | `──── … ────` | Spawn dispatch line | `build-wave.md:259` |
| Rule | `─────── (50)` | Intra-block divider | `plan.md:626` |

Command banners, verbatim:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━     build-prep.md:277
🔨 B U I L D I N G   P H A S E   {id}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Phase {id}: {name}
💾 Git checkpoint saved
```
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━     build-complete.md:330
🔨 B U I L D   S U M M A R Y
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📍 Phase {id}: {name}
🎲 Pattern:  {selected_pattern}

🐜 Workers:  {pass_count} passed  {fail_count} failed  ({total} total)
🛠️ Tools:    {total_tools} calls across all workers
⏱️ Duration: {elapsed}
```
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━     continue-verify.md:132
👁️🐜 V E R I F I C A T I O N   L O O P
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id} — Checking colony work...
```
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━     continue-finalize.md:396
➡️ P H A S E   A D V A N C E M E N T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━     init.md:157 / 495
🥚 A E T H E R   C O L O N Y   I N I T
🥚 A E T H E R   C O L O N Y
```
```
📊🐜🗺️🐜📊 ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 📊🐜🗺️🐜📊    plan.md:615
   C O L O N Y   P L A N
```
Note the **letter-spaced ALL-CAPS title** convention — preserved in the runtime today as `spacedTitle()` in `cmd/codex_visuals.go:347`.

Wave separator — `build-wave.md:744-746`: `━━━ 🐜 Wave {X} of {N} ━━━`

Build header before waves — `build-wave.md:17-21`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Phase {id}: {name} — {N} waves, {M} tasks
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### A-4. Milestone / maturity framing

Weak in v5.4.0 wrappers. Milestone appears only as a **gate check**, not a celebration:
- `plan.md:34-38`, `build-prep.md:101`, `continue-verify.md:29` — `If milestone == "Crowned Anthill": "This colony has been sealed. Start a new colony with /ant:init \"new goal\"."`
- `init.md:136-139` — sealed colony detection: `Previous colony was sealed. Starting fresh colony.`

The only maturity-flavoured ceremony is **project completion** — `continue-finalize.md:367-385`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   🎉 P R O J E C T   C O M P L E T E 🎉
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

👑 Goal Achieved: {goal}
📍 Phases Completed: {total}

🧠 Colony Learnings:
{condensed learnings from memory.phase_learnings}

👑 Wisdom Added to QUEEN.md:
{count} patterns/redirects/philosophies promoted across all phases

🐜 The colony rests. Well done!
```

### A-5. Pheromone presentation style

Classic did **not** hand-format signals — it delegated to a runtime command and then narrated the legend. `build-context.md:23-31`:
```
Then display the active pheromones table by running:
aether pheromone-display

This shows the user exactly what signals are guiding the colony:
- 🎯 FOCUS signals (what to pay attention to)
- 🚫 REDIRECT signals (what to avoid - hard constraints)
- 💬 FEEDBACK signals (guidance to consider)
```
Init offered signals as an editable numbered list — `init.md:225-234`:
```
1. [FOCUS] Testing infrastructure present (47 test files) -- maintain TDD discipline
2. [REDIRECT] Environment files detected -- never commit secrets or .env files
3. [FOCUS] Code quality tools configured -- follow existing lint/format rules

Edit, remove, or add signals as needed. Approved signals will be auto-applied.
```
Mid-build pheromone emission was announced — `build-wave.md:738`:
```
Warning: Midden threshold: "{category}" recurring ({count}x) -- REDIRECT emitted mid-build
```

### A-6. Completion ceremonies

- **Build summary** — A-3 above (`build-complete.md:330`)
- **Verification report** with a per-check grid — `continue-verify.md:284-308`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
👁️🐜 V E R I F I C A T I O N   R E P O R T
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔨 Build        [PASS/FAIL/SKIP]
🔍 Types        [PASS/FAIL/SKIP] (X errors)
🧹 Lint         [PASS/FAIL/SKIP] (X warnings)
🧪 Tests        [PASS/FAIL/SKIP] (X/Y passed)
   Coverage     {percent}% (target: 80%)
   🧪 Probe     [ACTIVE/SKIP] (tests added: X, edge cases: Y)
🔒 Secrets      [PASS/FAIL] (X issues)
📦 Gatekeeper   [PASS/WARN/SKIP] (X critical, X high)
👥 Auditor      [PASS/FAIL] (score: X/100)
📋 Diff         [X files changed]

──────────────────────────────────────────────────
🐜 Success Criteria
  ✅ {criterion 2}: {specific evidence}
  ❌ {criterion 3}: {what's missing}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Overall: READY / NOT READY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
- **Phase advancement** — `continue-finalize.md:396-434`, with learning/wisdom/instinct sections:
```
✅ Phase {prev_id}: {prev_name} -- COMPLETED

🧠 Learnings Extracted:
👑 Wisdom Promoted to QUEEN.md:
   [{type}] {brief claim}
🐜 Instincts Updated:
   [{confidence}] {domain}: {action}
📝 QUEEN.md Updated:
   Build learnings: {written_count} entries
   Instincts promoted: {promoted_instinct_count} entries

─────────────────────────────────────────────────────

➡️ Advancing to Phase {next_id}: {next_name}
   📋 Tasks: {task_count}
   📊 State: READY
```
- **Next Up block** — a consistent closing idiom across all four (`init.md:511-515`, `plan.md:640-646`, `continue-finalize.md:426-434`):
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🐜 Next Up
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   /ant:plan                 📊 Generate execution plan
   /ant:status               📋 Check colony state
   /ant:focus                🎯 Set initial focus

💾 State persisted -- safe to /clear, then run /ant:plan
```
The `💾 State persisted — safe to /clear` line appears in all three (`init.md:508`, `plan.md:646`, `continue-finalize.md:432`).

### A-7. Failure / escalation ceremony (the strongest Classic asset)

`build-wave.md:636-650`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ WAVE FAILURE — BUILD HALTED
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

All {N} workers in Wave {X} failed. Something is fundamentally wrong.

Failed workers:
  {caste_emoji} {Ant-Name}: {task_description} ✗ ({failure_reason} after {tool_count} tools)

Next steps:
  /ant:flags      Review blockers
  /ant:swarm      Auto-repair mode
```
`build-wave.md:662-679`:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ ESCALATION — QUEEN NEEDS YOU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: {failed task description}
Phase: {phase number} — {phase name}

Tried:
  • Worker retry (2 attempts) — {what failed}
  • Parent tried alternate approach — {what failed}
  • Queen reassigned to {other caste} — {what failed}

Options:
  A) {recommended option} — RECOMMENDED
  B) {alternate option}
  C) Skip and continue — this task will be marked blocked

Awaiting your choice.
```
Six gate-failure banners share one template (`⛔` + caste emoji + spaced title + `🚨 Issues` + `🔧 Required Actions` + a closing consequence sentence) — `continue-verify.md:316`, `continue-gates.md:12, 42, 359, 464, 487, 536, 650`.

### A-8. Other visual elements

- **Graveyard caution** — `build-wave.md:451`: `⚰️ Graveyard caution for {ant_name}: {file_1} ({level_1}), {file_2} ({level_2})`; `build-complete.md:79`: `⚰️ Grave recorded: {file} — {ant_name} failed ({summary})`
- **Visual checkpoint (UI)** — `build-complete.md:217-228`: `━━━ 🖼️🐜 V I S U A L   C H E C K P O I N T ━━━`
- **Survey loaded** — `build-context.md:67`: `━━━ 🗺️🐜 S U R V E Y   L O A D E D ━━━`
- **Resumption** — `build-prep.md:105`, `plan.md:49`: `🔄 Resuming: Phase X - Name`
- **Progress bar** — `build-prep.md:287`: `[Phase ${current_phase}/${total_phases}] ${progress_bar}` via `aether generate-progress-bar`
- **Depth label** — `build-prep.md:165-178`: `Depth: {colony_depth} ({label})` with `light -> "Builder only -- fastest"` … plus a first-run tip
- **Pattern announce** — `build-wave.md:242-245`: `━━ Pattern: {pattern_name} ━━`
- **Plan phase display** — `plan.md:628-636`: `📍 Phase {id}: {name} [{STATUS}]` / `🐜 Tasks:` / `✅ Success Criteria:` with icons `[ ] [~] [✓]`
- **ASCII progress in watch files** — `plan.md:126`: `[                    ] 0%`

---

## B. What each v5.4.0 wrapper actually taught

**Verdict: the wrappers were thin; the playbooks were thick.** Rated per file:

| File @ v5.4.0 | Stage purpose | Files to read | Spawn instructions | Stop conditions | Verdict |
|---|---|---|---|---|---|
| `build.md` | ✅ generic only | ❌ (only playbook paths) | ❌ | ✅ generic only | **Thin** |
| `continue.md` | ✅ generic only | ❌ | ❌ | ✅ generic only | **Thin** |
| `init.md` | ✅ | ✅ | n/a | ✅ | **Thick** |
| `plan.md` | ✅ | ✅ | ✅ | ✅ | **Thick** |
| playbooks | ✅ | ✅ | ✅ | ✅ | **Thick** |

**What the thin wrappers did teach** (`build.md@v5.4.0:35-60`) — a genuinely good pattern worth carrying forward as an idea:
```
## Execution Contract
For each stage:
1. Read the file with the Read tool.
2. Execute the instructions exactly as written.
3. Keep an in-memory stage result record:
   - `stage_name`
   - `status` (`ok` or `failed`)
   - `key_outputs` (values needed downstream)
4. If `status == failed`, halt and report the failure with recovery options.

## Required Cross-Stage State
Carry these values forward when produced:
- `phase_id` / `visual_mode` / `verbose_mode` / `suggest_enabled`
- `colony_depth` / `prompt_section` / `wave_results`
- `verification_status` / `synthesis_status` / `next_action`
```
That named-state-carry contract is engineering method, and it has no equivalent in today's wrappers.

**Engineering method in the thick files** — representative quotes:

*Stop conditions, plan.md:679-693 (`## Auto-Termination Safeguards`):*
> `1. **Confidence Threshold**: Loop exits when overall confidence reaches {target_confidence}%`
> `3. **Stall Detection**: If confidence improves < 5% for 2 consecutive iterations, auto-finalize current plan`
> `6. **Escape Hatch**: /ant:plan --accept accepts current plan regardless of confidence`

*Files-to-read routing table, plan.md:146-153 — goal keyword → survey doc:*
> `| UI, frontend, component, page | DISCIPLINES.md, CHAMBERS.md |`
> `| refactor, cleanup | PATHOGENS.md, BLUEPRINT.md |`
(mirrored in `build-context.md:50-58` keyed on phase name)

*Spawn choreography, build-wave.md:251 and 741-749:*
> `**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**`
> `Repeat Step 5.1-5.2 for each subsequent wave, waiting for previous wave to complete.`
> `(DO NOT use run_in_background - multiple Task calls in a single message run in parallel and block until complete)`

*Explicit budget caps, build-wave.md:470-473:*
> `- archaeology_context: cap at 4000 characters. If it exceeds the cap, truncate and append [archaeology truncated].`
> `- midden_context: cap at 2000 characters`
> `- grave_context: cap at 2000 characters per worker`

*Hard stop, build-wave.md:652:*
> `Then STOP — do not proceed to subsequent waves, Watcher, or Chaos. Skip directly to Step 5.9 synthesis with status: "failed".`

*Stage audit gate, build-complete.md:16-30:* HALT if any of the 4 prior stages lacks `"status": "ok"`, with a 3-option recovery menu.

*Structured `<success_criteria>` / `<failure_modes>` / `<read_only>` blocks* — present in `init.md:19-45` and `build-prep.md:40-75`, absent from all four current wrappers. Example, `build-prep.md:41-45`:
> `### Wave Failure Mid-Build`
> `If a worker fails during a build wave:`
> `- Do NOT continue to next wave (failed dependencies will cascade)`
> `- Options: (1) Retry the failed task, (2) Skip and continue with remaining tasks, (3) Abort build`

---

## C. Delta table

Ownership key: **RT** = Go runtime renderer (`cmd/ceremony_cmd.go`, `cmd/codex_visuals.go`) · **WR** = wrapper narration · **GAP** = no owner today.

| # | Element | v5.4.0 | Current | Restore where | Note |
|---|---|---|---|---|---|
| 1 | Command banner (`🔨 B U I L D I N G   P H A S E`) | ✅ | **Degraded** — `renderBanner`/`spacedTitle` exist (`codex_visuals.go:347,365`) but build/continue wrappers emit no phase-open banner | **RT** | `renderOldStyleCeremonyHeader` already used by `spawn-plan` |
| 2 | Spawn plan block w/ per-worker caste lines | ✅ | ✅ **Runtime** — `renderCeremonySpawnPlan` (`ceremony_cmd.go:263`) + `Total: … = N spawns` | **RT (done)** | Wrapper already calls `ceremony spawn-plan` |
| 3 | Caste Emoji Legend printed by wrapper | ✅ | ❌ and **forbidden** | **RT only** | `casteEmojiMap`/`ColorMap`/`LabelMap` at `codex_visuals.go:36/65/94`. Test forbids caste-listing prose in build.md |
| 4 | Wave separator + spawn announcement | ✅ | ✅ **Runtime** — `renderCeremonyWaveStart` (`ceremony_cmd.go:479`) | **RT (done)** | |
| 5 | Per-caste specialist banner (`━━━ 🔮 O R A C L E ━━━`) | ✅ 10 castes | **Degraded** — collapsed into generic wave-start | **RT** | Phase 168 candidate |
| 6 | Streaming per-worker completion line | ✅ | ✅ **Runtime** — `renderCeremonyWorkerComplete` (`ceremony_cmd.go:502`) | **RT (done)** | |
| 7 | Worker prompt caste identity | ✅ | ✅ **Runtime** — inside composed `brief`/`brief_path` | **RT (done)** | Wrapper passes verbatim; guarded |
| 8 | Per-worker skill line | ✅ | ✅ **Runtime** — `renderCeremonySkillAssignments` (`ceremony_cmd.go:412`) | **RT (done)** | |
| 9 | Build summary block | ✅ | **Degraded** — closeout covers it; `🎲 Pattern`, `🛠️ Tools`, `⏱️ Duration` absent | **RT** | Phase 168 |
| 10 | Workflow pattern announce | ✅ | **GAP** — `deriveWorkflowPattern` at `orchestrator.ts:95` writes to **stderr** | **RT** | Computed and never shown |
| 11 | Verification report grid | ✅ | **Degraded** | **RT** | Phase 168 |
| 12 | Gate-failure banners (6 variants) | ✅ | **Degraded** | **RT** | Shape only — see D-6 |
| 13 | Wave-failure / escalation banners | ✅ | **GAP** | **RT** | Strongest Classic asset; Phase 168 |
| 14 | Phase advancement block | ✅ | **Degraded** — learnings/wisdom/instinct sections absent | **RT** | Phase 168 |
| 15 | Project complete block | ✅ | **GAP** in continue path | **RT** | Phase 168 |
| 16 | Next Up block | ✅ | ✅ **Runtime** — `renderNextUp` (`codex_visuals.go:395`) | **RT (done)** | |
| 17 | `💾 State persisted — safe to /clear` | ✅ | ❌ and **forbidden in wrapper** | **RT only** | `continue_wrapper_ceremony_test.go:78` |
| 18 | Init banner + approval loop framing | ✅ | **Degraded** — approval survives; banner + 👑 intention beat gone | **WR** (framing) + **RT** (banner) | `init_ceremony.go` exists |
| 19 | Pheromone table + legend | ✅ | **Degraded** — prose ordering instead of render+narrate | **RT** render, **WR** narrate | |
| 20 | Graveyard caution / grave recorded | ✅ | **GAP** | **RT** | Phase 168 |
| 21 | Visual checkpoint (`ui_touched`) | ✅ | **GAP** | **RT** | Phase 168 |
| 22 | Survey loaded banner | ✅ | **GAP** | **RT** | Phase 168 |
| 23 | `🔄 Resuming: Phase X - Name` | ✅ | **Degraded** | **RT** | |
| 24 | Progress bar `[Phase n/N]` | ✅ | **Degraded** | **RT** | |
| 25 | Depth label + first-run tip | ✅ | **Degraded** — `depth=` k/v, no human label | **RT** | |
| 26 | Plan phase display w/ status icons | ✅ | ✅ **Runtime** — `ceremony_cmd.go:670` | **RT (done)** | |
| 27 | `<success_criteria>`/`<failure_modes>`/`<read_only>` blocks | ✅ | **GAP** in all four | **WR** | Safe wrapper content — restore in 165 |
| 28 | Named cross-stage state carry contract | ✅ | **GAP** | **WR** | Safe wrapper content — restore in 165 |
| 29 | Stop conditions in prose | ✅ | **Degraded** | **WR** | Restore in 165 |
| 30 | Files-to-read routing tables | ✅ | **GAP** | **RT** (survey paths already in brief) | Do not re-derive in wrapper |
| 31 | Spawn choreography rules | ✅ | ✅ **Wrapper** — build.md:89 | **WR (done)** | |
| 32 | Milestone / maturity framing | ⚠️ gate-check only | ⚠️ `milestoneIcon` unused in build/continue | **RT** | Net-new — Phase 168 decides |

---

## D. Regression warnings — do NOT bring these back

### D-1 · CRITICAL — Playbook indirection
- **Evidence:** `build.md@v5.4.0:29-33` and `continue.md@v5.4.0:27-30` Read-load 9 playbooks. CLAUDE.md §Command Playbooks: not loaded since v1.25.
- **Enforced by:** `cmd/build_wrapper_ceremony_test.go:74-82`; `cmd/platform_doc_hygiene_test.go` forbids `"Read the file with the Read tool."` in build.md and playbook names in continue.md.
- **Rule:** mine the playbook text, inline it. Never restore the loader.

### D-2 · CRITICAL — Wrapper state mutation (init)
- **Evidence:** `init.md@v5.4.0:283-357` — `aether state-write` with jq-reconstructed JSON, Write-tool template expansion of COLONY_STATE.json, direct writes of constraints/pheromones/midden/learning-observations.
- **Why removed:** Frankenstein-state corruption (LLM reconstructs full JSON). `wrapper-runtime-ux-contract.md:139` anti-pattern 1.
- **Enforced by:** `platform_doc_hygiene_test.go` forbids `"Write COLONY_STATE.json"` and `"queen-init"` in init.md.
- **Rule:** restore init's banners/ordering/approval copy; all persistence stays behind `aether init --charter-json`.

### D-3 · HIGH — Wrapper state mutation (plan) + tmux watch files
- **Evidence:** `plan.md@v5.4.0:101-132` watch-status/watch-progress writes; `:560-586` COLONY_STATE.json state-write retry ladder.
- **Enforced by:** `platform_doc_hygiene_test.go` forbids `"Update watch files for tmux visibility"` and `"Write COLONY_STATE.json"` in plan.md.
- **Rule:** `plan-finalize` owns plan persistence.

### D-4 · HIGH — Speculative caste naming in wrapper text
- **Evidence:** `build-wave.md@v5.4.0:64-77` Caste Emoji Legend.
- **Enforced by:** `platform_doc_hygiene_test.go` build.md forbidden list. Current guardrail: "Do NOT invent worker names, castes, or waves; use `dispatch_manifest`."
- **Rule:** caste identity comes from `casteIdentity()` via runtime ceremony commands.

### D-5 · HIGH — Context-clear ceremony in continue wrapper
- **Evidence:** commit `ecf68ba4` (2026-04-21) removed it; tests flipped from requiring to forbidding.
- **Enforced by:** `continue_wrapper_ceremony_test.go:78-105`.
- **Rule:** the most explicitly-flipped test in the change zone. Do not restore.

### D-6 · HIGH — Gate logic duplicated in wrapper markdown
- **Evidence:** `continue-gates.md@v5.4.0` reimplements six gates with thresholds in markdown.
- **Rule:** banner shape restorable (from runtime); threshold arithmetic and gate decision trees are not.

### D-7 · MEDIUM — Synthetic build/plan forcing
- **Evidence:** commit `5e6d2cf9` (2026-04-22) "stop forcing synthetic build and plan"; `9f40bf48` removed silent-simulation fallback.
- **Rule:** restored narration must not imply synthetic/simulated dispatch.

### D-8 · MEDIUM — Prompt-injection-shaped transcript in build wrapper
- **Evidence:** `build-prep.md@v5.4.0:29-36` verbatim user transcript as worked example.
- **Rule:** restore the Context Confirmation Rule as neutral instruction, never quoted user text.

### D-9 · MEDIUM — `git stash` / `git commit` from wrapper text
- **Evidence:** `build-prep.md@v5.4.0:264-266` (`git stash push`, with the `--include-untracked` scar note); `continue-finalize.md:276` (`git add -A && git commit`).
- **Rule:** no wrapper-driven git mutation. Checkpointing, if it returns, is runtime work.

### D-10 · MEDIUM — Stale/hardcoded environment assumptions
- **Evidence:** LiteLLM proxy health-curl (`build-prep.md:77-86`); `.aether/aether-utils.sh` gate (`init.md:69`).
- **Enforced by:** `platform_doc_hygiene_test.go` forbids `.aether/aether-utils.sh` in every wrapper.
- **Rule:** neither returns.

### D-11 · LOW — Depth vocabulary drift
- **Evidence:** v5.4.0 `colony_depth ∈ {light, standard, deep, full}` vs current `--verification-depth ∈ {light, standard, heavy}` + Queen smart defaults.
- **Rule:** human-readable depth labels good; four-value vocabulary and `--depth` flag are drift.

---

## Summary block

```
change_zone: .claude/commands/ant/{init,plan,build,continue}.md (+ .opencode mirrors)
regression_risks: 11 — 2 CRITICAL, 4 HIGH, 4 MEDIUM, 1 LOW
stability: 4 volatile files (52 commits since v5.4.0; 18 named "Restore"/"Fix")
tribal_knowledge: 32 ceremony elements inventoried, 9 with no current owner
restoration_count: ceremony "restored" 7 times since v5.4.0
  (bf167427, 5ac63e08, 9ac9718a, 54cadccf, bb16847b, 64c018f6, e8040a66)
top_risk: v5.4.0 build.md/continue.md were playbook stubs — ceremony must be
  mined from .aether/docs/command-playbooks/ and INLINED; the loader is forbidden.
definition_of_done: ship a command/test that FAILS when the ceremony is absent,
  or this becomes restoration #8.
```
