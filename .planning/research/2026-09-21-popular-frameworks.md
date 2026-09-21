# What Aether should steal from the >1,000-star frameworks

Research date 2026-09-21. Stars / last push verified with `gh api repos/<o>/<r>`. Everything
below was read from the named file via `gh api .../contents/...`, or from the locally
installed GSD Core copy (`~/.claude/gsd-core/`). Marketing claims are flagged as such.

## 0. Roster (all verified)

| Repo | Stars | Last push | Note |
|---|---|---|---|
| obra/superpowers | 289,675 | 2026-09-20 | skills methodology |
| github/spec-kit | 138,206 | 2026-09-21 | Python CLI + slash skills |
| ruvnet/claude-flow (now ruvnet/ruflo) | 72,999 | 2026-09-21 | "agent meta-harness" |
| Fission-AI/OpenSpec | 69,765 | 2026-09-21 | Node CLI + skills |
| gsd-build/get-shit-done | 64,485 | 2026-05-31 | ARCHIVED -> open-gsd/gsd-core (9,691, 2026-09-21); gsd-2 -> open-gsd/gsd-pi (1,251) |
| hesreallyhim/awesome-claude-code | 54,396 | 2026-09-21 | list |
| bmad-code-org/BMAD-METHOD | 53,313 | 2026-09-21 | skills + python scripts |
| wshobson/agents | 39,852 | 2026-09-21 | plugin marketplace |
| davila7/claude-code-templates | 30,888 | 2026-09-21 | config installer (not a workflow; skipped) |
| eyaltoledano/claude-task-master | 28,087 | 2026-04-28 | Node CLI/MCP, tasks.json |
| steveyegge/beads (now gastownhall/beads) | 27,339 | 2026-09-21 | Go CLI + Dolt DB issue tracker |
| SuperClaude-Org/SuperClaude_Framework | 23,902 | 2026-09-15 | 30 commands / 20 agents / 7 modes |
| snarktank/ralph | 21,837 | 2026-02-02 | bash loop + prd.json |
| coleam00/context-engineering-intro (PRP) | 13,872 | 2026-03-16 | 2 commands |
| humanlayer/humanlayer | 11,592 | 2026-06-19 | README says code is deprecated; skipped |
| automazeio/ccpm | 8,383 | 2026-03-18 | PRD->epic->GitHub issues |
| snarktank/ai-dev-tasks | 7,785 | 2025-11-05 | 2 markdown prompts |
| steipete/agent-rules | 5,684 | 2026-05-03 | ARCHIVED; skipped |
| buildermethods/agent-os | 5,434 | 2026-08-29 | standards injection + shape-spec |
| parcadei/Continuous-Claude-v3 | 3,942 | 2026-01-26 | hooks: ledgers + handoffs |
| disler/claude-code-hooks-mastery | 3,926 | 2026-03-04 | hook examples |
| OneRedOak/claude-code-workflows | 3,893 | 2025-09-14 | review workflows (stale) |
| Pimzino/claude-code-spec-workflow | 3,857 | 2025-09-07 | Kiro-style; README says focus moved to spec-workflow-mcp (4,292) |
| gemini-cli-extensions/conductor | 3,744 | 2026-09-01 | tracks: spec.md + plan.md |
| gotalab/cc-sdd | 3,672 | 2026-05-20 | Kiro-style skills (kiro-*) |

## 1. Per-project notes

### OpenSpec (Fission-AI) — the closest thing to "Aether's light path done right"
- Shape: default profile is THREE commands: `/opsx:propose "idea"` (one pass writes proposal.md +
  specs/ + design.md + tasks.md) -> `/opsx:apply` -> `/opsx:archive`. `/opsx:explore` is an optional
  no-stakes thinking partner. The "expanded" profile (`new`, `continue`, `ff`, `verify`,
  `bulk-archive`, `onboard`) is opt-in via `openspec config profile`. (README; docs/opsx.md)
- State: plain markdown per change folder `openspec/changes/<name>/`. The CLI never stores
  progress — `openspec status --change X --json` DERIVES it: an artifact is DONE if its file
  exists, READY if its deps are done, BLOCKED otherwise ("State detection (filesystem existence)",
  docs/opsx.md "Dependency Graph Model"). Chat writes the markdown; program only reads + hands
  out instructions (`openspec instructions <artifact> --json`).
- Verify: `/opsx:verify` = completeness / correctness / coherence, CRITICAL/WARNING/SUGGESTION,
  and explicitly "Does not block archive, but surfaces issues" (docs/commands.md). Archive
  "warns if incomplete" and proceeds.
- Stuck: "Actions, not phases… Dependencies are enablers — they show what's possible, not
  what's required next." `/opsx:update` revises artifacts in any direction mid-build.
  Their own history: the legacy phase-locked workflow was replaced because
  "Linear phases fight against how work actually happens."
- Small vs big: no size routing; same propose for both (weakness). Model: recommends strongest
  model for everything.

### BMAD-METHOD — best scale-adaptive mechanism found
- Shape: `bmad-build "<what I want>"` is the single entry. `skills/bmad-build/step-01-clarify-and-route.md`
  resolves existing state and routes; `step-02-plan.md` writes a short spec then applies
  `route_selection` from `customize.toml`:
  "estimate the lines of code to add or modify. 100 lines or fewer: oneshot. Otherwise: full.
  If the change touches more than five files and is not simple or mechanical, consider upgrading to full."
  `route: auto|oneshot|full` and `review: auto|none|quick|thorough` are pinnable; spec
  frontmatter records `route_source: auto|pinned`.
- Oneshot (`step-oneshot.md`): implement in the main chat -> ONE context-free reviewer subagent
  reading a diff file since `baseline_commit` -> orchestrator classifies each finding
  (high/medium/low/false/maybe-false) -> patch / HALT / defer -> commit -> 1–2 sentence summary.
  Escape upward: if a gap appears mid-build, set `route: 'full'`, `status: 'draft'`, go back to planning.
- State: the spec file's own frontmatter `status: draft|ready-for-dev|in-progress|in-review|done`
  IS the resume pointer. Re-running `bmad-build` with no args lists active specs and offers
  resume/new. Multi-story tracking is `sprint-status.yaml`, written by a python script
  (`sprint_plan.py`), not by chat.
- Verify: `references/claims-check.md` — reviewer treats the spec as "testimony, not evidence"
  and tries to falsify each claim against the diff. `deferred-work.md` is an append-only parking
  lot ("Do not edit old entries or check for duplicates").
- Drift evidence: ships `skills/bmad-sprint-planning/references/fix-sprint-status.md` — a whole
  procedure for rebuilding the status file "when it is broken, hand-mangled, drifted from
  reality", with the rule "prefer the lower status… a false done costs more than a false in-progress".
- Model: "Review subagents must use the same model level as this session."
- Docs openly say "You don't need BMad for an obvious, low-risk edit."

### github/spec-kit
- Shape: constitution (once) -> specify -> plan -> tasks -> implement -> converge; clarify /
  checklist / analyze optional. Now three independent entry points: SDD, a bug path
  (`extensions/bug`: assess -> fix -> test, verdict `verified|partial|failed`, "Missing
  verification is not a successful fix"), and idea assessment.
- State: markdown in `.specify/` + `specs/<feature>/`; `tasks.md` checklist with `[P]` parallel
  markers; chat ticks `[X]` (`templates/commands/implement.md` line ~169). Scripts
  (`check-prerequisites.sh --json`) only locate files.
- Verify: `templates/commands/converge.md` — after implement, compare code to spec/plan/tasks and
  APPEND-ONLY a `## Phase N: Convergence` section of new tasks; "MUST NOT rewrite, renumber,
  reorder, or delete any existing task"; leaves tasks.md byte-identical when converged. Loop
  implement -> converge until "Converged".
- Soft gate: unchecked checklists -> table + "Do you want to proceed anyway? (yes/no)".
- Drift evidence: issue #1024 "Tasks duplicated when checked off during implementation";
  #181 asks for automatic completion tracking because chat ticking is inconsistent.
- OpenSpec's README calls it "Thorough but heavyweight. Rigid phase gates, lots of Markdown."

### obra/superpowers
- Shape: brainstorm -> worktree -> write plan (2–5 min tasks with exact paths) -> execute ->
  TDD -> review -> finish. TWO execution modes (README): subagent-per-task with review each, or
  `executing-plans` = "implements every task inline in the current session with one fresh review
  of the whole branch at the end (cheapest)".
- State: design doc + plan markdown only. No program. Skills auto-activate; no commands to remember.
- Verify: `skills/verification-before-completion/SKILL.md` — "NO COMPLETION CLAIMS WITHOUT FRESH
  VERIFICATION EVIDENCE"; table incl. "Agent completed -> requires VCS diff shows changes; not
  sufficient: agent reports success".
- Stuck: `subagent-driven-development/SKILL.md` — four worker statuses DONE / DONE_WITH_CONCERNS /
  NEEDS_CONTEXT / BLOCKED; "Never… force the same model to retry without changes"; fix loop is
  capped at 5 rounds, rounds 4–5 use "fresh implementer, more capable model".
  `diagnosing-superpowers` reads the session transcript to explain what went wrong.
- Model: "Use the least powerful model that can handle each role"; 1–2 files + complete spec ->
  cheap; multi-file -> standard; design -> most capable; "Always specify the model explicitly" —
  with the honest caveat that cheapest models "routinely take 2-3x the turns".
- Issues show the same pain as Aether: #1251 "reduce overhead across many small tasks",
  #1953 "Context is being consumed too quickly".

### GSD Core (open-gsd/gsd-core; read from local install)
- Shape: new-project -> discuss -> plan -> execute -> verify -> ship, but with TWO explicit bypasses:
  `/gsd-fast` (`workflows/fast.md`): "No PLAN.md, no Task spawning, no research… understand -> do ->
  commit -> log"; scope check "<= 3 file edits, <= 1 minute, no new deps, no research" else it
  redirects to quick. `/gsd-quick` (`workflows/quick.md`): planner + executor only; composable
  opt-in flags `--discuss --research --validate`, `--full` = all.
- State: markdown (`STATE.md`, `ROADMAP.md`, PLAN/SUMMARY) — but increasingly written through a
  program (`gsd-tools.cjs`). fast.md itself documents why: the inline `awk` table arithmetic
  "was the root cause of #2133", replaced by schema-backed `quick-tasks-append` that "fails loud".
- Progress: `hooks/gsd-statusline.js` parses STATE.md and shows current task/phase/percent in the
  always-visible Claude Code status line. `/gsd-next` reads state and routes to the one next step.
- Model: `references/model-profiles.md` — quality/balanced/budget/adaptive/inherit table per agent
  (planner opus, executor sonnet, mappers haiku).
- Drift + dead-end evidence (open issues): #4213 "13 of 15 state.* verbs rewrite progress.percent
  but never the body Progress bar — the two surfaces silently diverge at rc=0"; #4765 resume
  "treats a STALE verification as valid… marking the phase complete without re-verifying";
  #4887 "stale covered_digest has no recovery path"; #4857 "permanent stale". I.e. GSD is
  discovering both of Aether's problems from the other direction.

### beads (Go CLI + Dolt) — the only peer that also bets on a compiled program
- `bd ready` (unblocked work), `bd update --claim` (atomic), `bd close`, `bd prime` (prints
  workflow context + memories; installed as a hook by `bd setup claude`), `bd remember "insight"`.
  Tells agents "Do not use markdown TODO lists for task tracking."
- No workflow at all — it is only the ledger. Zero ceremony by construction.
- Cost of a DB: open issues #6064 journal corruption after upgrade, #5657 concurrent label updates
  "silently lose or duplicate", #2559 (16 comments) can't connect after restart. A program does
  not abolish state bugs; it moves them somewhere the owner can't read.

### gemini-cli-extensions/conductor
- Two steps: `new-track "<desc>"` (spec.md + plan.md, approve) -> `implement`.
- `skills/conductor-setup/assets/workflow.md`: task markers `[ ]` -> `[~]` -> `[x] <commit sha>`;
  phase ends with `[checkpoint: <sha>]`, a proposed manual verification plan for the human, and the
  verification report attached to the commit via `git notes`. `conductor-revert` undoes a
  task/phase/track by those SHAs.

### snarktank/ralph
- PRD -> `prd.json` stories with `passes: false` -> bash loop spawns a FRESH agent per iteration:
  pick top failing story, implement, run typecheck/tests, commit only if green, set
  `passes: true`, append to `progress.txt`. Max-iterations cap is the only escape hatch.
  Memory = git + progress.txt + prd.json. Honest README: "Ralph only works if there are feedback loops".

### ccpm
- PRD -> epic -> tasks -> GitHub issues -> parallel worktree agents. Heavy for a solo owner.
- Worth noting: `skill/ccpm/references/track.md` "Script-First Rule… Run the script; do not
  reconstruct the output manually" — status/standup/next/blocked are 14 bash scripts.

### task-master
- `parse_prd` -> tasks.json -> `next_task` / `expand_task` / `set_task_status`, complexity
  analysis decides which tasks to expand. State = JSON written by CLI/MCP. main/research/fallback
  model roles. Tool-loading tiers `core` (7 tools, ~5k tokens) / `standard` / `all` (36) because
  the full surface cost too much context. Last push April 2026 — momentum gone.

### Others (brief)
- cc-sdd: Kiro-style 6-step; README route table includes "small change with no spec:
  kiro-discovery -> direct implementation". Tasks carry `_Boundary:_` / `_Depends:_` annotations.
- PRP (context-engineering-intro): INITIAL.md -> `/generate-prp` -> `/execute-prp`; the PRP embeds
  runnable validation commands. Two steps, no state, no resume.
- ai-dev-tasks: create-prd.md -> generate-tasks.md -> "start on task 1.1", human approves each sub-task.
- agent-os: v3 shrank to discover/inject standards + shape-spec — the author cut his own ceremony.
- Continuous-Claude-v3: PreCompact hook auto-writes a YAML handoff; SessionStart loads ledger.
  Aether already has the equivalent.
- SuperClaude: 30 commands/20 agents/7 modes of prompt text, no state, no verification.
- wshobson/agents: model tier table (Opus review/architecture, Sonnet docs/tests, Haiku ops).
- ruflo/claude-flow: see "What NOT to copy".
- Pimzino: 4-step bug workflow; author moved to an MCP server + web dashboard.

## 2. What Aether should steal (ranked)

### Part B — light default path (highest value; this is why the owner left)
1. **BMAD's auto-route with a recorded source.** `skills/bmad-build/customize.toml`
   `route_selection` (<=100 LOC -> oneshot; >5 non-mechanical files -> full) + spec frontmatter
   `route`, `route_source: auto|pinned`. One entry command, one short spec, THEN size decides.
   Fits Aether because the Go program can compute it deterministically from the plan's file list
   and the Queen's estimate, and print "treated as small because…". Replaces: the separate
   init -> discuss -> spec approval -> multi-pass plan ladder, and the owner having to choose
   between /ant-quick and /ant-init up front.
2. **OpenSpec's one-pass propose + opt-in expanded profile.** `/opsx:propose` writes all
   planning artifacts in one pass; `openspec config profile` hides the other six commands until
   asked for. Replaces: multi-pass planning as default; shrinks the 64-command menu to ~6 visible.
3. **GSD's two-rung quick ladder with composable flags.** `workflows/fast.md` (inline, <=3 files,
   no subagent, commit, log one row) and `workflows/quick.md` (`--discuss --research --validate`,
   `--full`). Replaces: /ant-quick as a one-shot question; gives an inline rung that still
   records into the Go ledger so memory/tracking (what the owner values) is kept.
4. **BMAD's upward escape from oneshot.** step-oneshot.md "When to stop and replan": write the gap,
   flip `route: full`, `status: draft`, return to planning. Makes a wrong "small" guess cheap.
5. **superpowers' inline execution mode** (`executing-plans`: one context, one final review) as the
   default for one-worker phases — Aether's fast path still spawns a Builder subagent.

### Part A — never a dead end
6. **OpenSpec's "verify surfaces, never blocks" + archive warns-and-proceeds**
   (docs/commands.md `/opsx:verify`, `/opsx:archive`). Policy: every Aether refusal becomes
   a CRITICAL/WARNING line unless work could be lost. Replaces strict-loader "malformed" dead ends.
7. **spec-kit's converge loop.** `templates/commands/converge.md`: a failed check appends new
   tasks (append-only, never renumber) and hands back to build. Replaces: /ant-continue blocking
   on a gate and needing /ant-unblock; "not done" becomes "here are 3 more tasks".
8. **Derive state from evidence instead of validating stored state.** OpenSpec
   `openspec status` (file exists = done). Aether already does this for task credit; extend it
   to resume/status so a stale/odd COLONY_STATE.json is rebuilt, not rejected.
9. **BMAD's "prefer the lower status" repair rule + confirm-then-write**
   (`fix-sprint-status.md`). One `aether repair` that proposes a state table and writes on yes.
10. **superpowers' four worker statuses and capped escalation** (DONE_WITH_CONCERNS /
    NEEDS_CONTEXT / BLOCKED; rounds 4–5 fresh worker on a stronger model; stop at 5).
11. **BMAD's `deferred-work.md` append-only parking lot** for findings not caused by this change —
    reviewers' extra findings stop blocking the phase.
12. **spec-kit's soft gate wording**: show the table, ask "proceed anyway? (yes/no)".

### Part C — screens shown to the owner
13. **GSD's status line** (`hooks/gsd-statusline.js` reads STATE.md). Aether has no `statusLine`
    in `.claude/settings.json` (checked). An `aether statusline` subcommand gives phase / task /
    next command permanently on screen, outside collapsed tool output. Cheapest big win.
14. **ccpm's Script-First Rule** (`references/track.md`): wrappers say "run it, present the output,
    do not reconstruct". Pair with having the wrapper re-print the program's card verbatim in a
    fenced block (the chat's own text is never collapsed).
15. **BMAD's present step**: "one or two sentences… Do not list files, repeat the spec, or walk
    through what you did unless asked" — matches the owner's CLAUDE.md exactly.
16. **Conductor's manual verification plan** at phase end: numbered user-level steps
    ("start the server, click X, expect Y") — suits a non-technical owner who "tests like a user".

### Part D — messy-repo journey as the release gate
17. **BMAD step-01 VCS sanity check**: dirty tree or mismatched branch -> ask, don't refuse; and
    `baseline_commit` captured in the spec so review diffs only this change (untracked included).
18. **Conductor's `[~]` / `[x] <sha>` markers + `[checkpoint: sha]`**: human-readable, and a
    program can verify the SHA exists and touches the claimed files. Gives revert-by-task for free.
19. **ralph's acceptance loop as the gate harness**: stories with `passes:false`, fresh agent per
    iteration, commit only when checks are green, iteration cap. Use it to RUN the ten journeys.
20. **beads `bd prime` / `bd ready`**: one command that prints everything a fresh session needs;
    Aether's session-start card is this already — keep it, make `aether ready` the single "what can
    I do now" answer.

### Required call-outs
- Lightest credible planning flow: OpenSpec `propose -> apply -> archive` (PRP's two commands are
  lighter but have no state or resume, so not credible for Aether).
- Best small-task path: BMAD auto-oneshot for "small but real"; GSD `/gsd-fast` for trivial.
- Best human-readable state: OpenSpec change folder + derived status; with Conductor's SHA markers
  for completion evidence. Recommendation: Go program stays the writer, but EXPORTS a plain
  `tasks.md`-style view the owner can read, and can rebuild its JSON from evidence.
- Best stuck/recovery: spec-kit converge (append tasks) + OpenSpec non-blocking verify +
  BMAD status-in-frontmatter resume.
- Scale-adaptive ceremony: BMAD `route_selection` + `review: auto`; GSD composable flags;
  superpowers per-task model tiering; task-master complexity-driven expansion.

## 3. What NOT to copy
- ruflo/claude-flow swarms, "neural" memory, federation. Issue #1058: hook handlers
  "return {recorded: true}… NO database INSERT", "duration: Math.random()"; #3376 "reports success
  after skipping the model download". Stars are not evidence of function.
- BMAD's persona roster / PRD / PRFAQ / architecture spine for solo work (their own docs say skip).
- spec-kit's constitution + clarify + checklist + analyze chain, and its extension/preset/bundle system.
- ccpm's PRD -> epic -> GitHub-issue sync.
- SuperClaude's 30 commands / 7 modes — surface area without state or verification.
- beads' embedded database (Dolt) — corruption/upgrade/locking issues the owner could never debug.
- superpowers' per-task two-stage review as a default (their own #1251/#1953 complain about it).
- Web dashboards (Pimzino, OpenSpec) — a second window; the owner has said he wants it inline.
- GSD's digest-based "stale verification" machinery (#4887, #4857: no recovery path) — it is
  Aether's dead-end problem reproduced elsewhere.
- More agents. wshobson ships 202; nobody's evidence shows roster size helping.

## 4. Frank assessment: is the Go program unnecessary?
No — but most of what it currently enforces is. The three most-used projects (superpowers,
spec-kit, OpenSpec) keep all state in markdown the chat writes, and ship faster for it. They
also visibly suffer for it: spec-kit #1024 (tasks duplicated while being ticked) and #181;
BMAD ships a dedicated repair procedure for a status file that has "drifted from reality" and
moved its writes into a python script; GSD moved table writes from inline awk into a
schema-backed tool after #2133, and still has #4213 (two progress surfaces "silently diverge at
rc=0") and #4765 (phase marked complete off a stale verification). ralph, task-master, beads and
OpenSpec all put a program between the chat and the ledger. So the direction of travel in the
field is TOWARD Aether's architecture, not away from it. What none of the successful ones do is
let the program say "no": OpenSpec's CLI derives status and hands out instructions; BMAD's
script writes what the human confirmed; spec-kit's scripts only locate files. The program is a
bookkeeper and a calculator, never a bouncer. Aether's defensible core is small — evidence-based
task credit, running the project's checks, memory, the session card — and that core is genuinely
better than anything above. The 5,000 tests guarding admission gates, reviewer-forcing matrices
and canary promotion are where Aether went where no successful project has found users.
GSD's open "no recovery path" issues are the warning: strictness without an exit is the failure
mode, whoever writes the state.
