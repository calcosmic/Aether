# Codex Native Parity Handoff

**Prepared:** 2026-08-25

**Repository:** `/Users/callumcowie/repos/Aether`

**Current branch when audited:** `oracle-reinstate`

**Current HEAD when audited:** `bce085ce`

**Status:** Research and recommendation complete; implementation must wait for
the paused Phase 194 Claude/GSD review-fix to be preserved and completed.

> [!CAUTION]
> Read **Phase 194 recovery hazard** before running any GSD review/fix,
> worktree cleanup, branch deletion, or Codex parity implementation. The final
> Phase 194 fix currently exists only as uncommitted code in a Claude-created
> worktree. A fresh GSD fixer run can destroy it.

## Purpose

This document hands off the complete assessment of what is required to make
Aether a first-class Codex experience using native Codex Agent Skills and
native Codex subagents instead of relying on Claude slash-command wrappers.

It records:

- the unfinished Phase 194/GSD state that must be protected;
- what Aether already supports in Codex;
- the semantic, orchestration, packaging, and testing gaps;
- the recommended architecture, which does not require a new framework;
- effort estimates and implementation phases;
- acceptance criteria for promoting Codex from secondary to primary support;
- a restart checklist and suggested prompt for the next implementation session.

This was a read-only audit performed through three independent review tracks:

1. Claude/OpenCode versus Codex command and agent parity.
2. Unfinished GSD/Phase 194 state and recovery hazards.
3. Packaging, permissions, tests, delivery risks, and effort.

No Phase 194 files, worktrees, branches, runtime data, or implementation files
were changed during the audit. This handoff is the only artifact created by
this audit; the checkout already contains other dirty and untracked user/GSD
artifacts that must be preserved.

## Executive verdict

Yes, native Codex parity is feasible without adding another orchestration
framework.

Aether already has most of the structural plumbing:

- 27 native Codex agent definitions;
- Go-owned dispatch manifests and completion finalizers;
- typed permission profiles;
- a headless `codex exec` worker path;
- generated Codex command skills;
- full lifecycle skill source;
- a TypeScript host that separates manifest preparation from worker dispatch;
- visible ceremony commands and spawn ledgers.

The missing work is primarily semantic synchronization, a reliable native
Codex conductor, installation/update correctness, and tests that prove behavior
rather than file presence.

### Effort estimate

These are alternative scope levels, not additive estimates.

| Target | Focused engineering effort | Outcome |
|---|---:|---|
| Working demonstration | 2–4 days | Nine lifecycle skills use native Codex subagents on the happy path |
| Production lifecycle parity | 10–15 days | Reliable `init -> plan -> build -> continue -> seal`, including recovery and bounded multi-agent waves |
| Defensible full Aether parity | 15–25 days | All 64 commands and all 27 agent contracts classified, synchronized, tested, installed, and released |

**Recommended first target:** production lifecycle parity. Do not make a
platform-wide parity claim from the working demonstration alone.

## Plain-English model

- A Claude slash command is a recipe and a button.
- A Codex skill is the Codex version of that recipe.
- A Codex custom subagent is one Aether worker caste.
- The Go runtime remains the accountant and safety authority: it decides the
  manifest, validates terminal results, finalizes state, and rejects stale or
  malformed completion packets.
- The parent Codex agent acts as a bounded conductor. It may interview,
  synthesize, spawn workers, wait for them, and summarize, but it must not invent
  workers or hand-edit `.aether/data`.

The hard part is not asking Codex to spawn several reviewers. The hard part is
ensuring exact caste routing, wave dependencies, permission boundaries,
terminal-result validation, interruption recovery, and exactly-once
finalization.

## Phase 194 recovery hazard — handle first

### Active unfinished work

The main checkout was audited at:

- branch: `oracle-reinstate`;
- HEAD: `bce085ce`;
- colony/GSD status: Phase 194 still `verifying`;
- Phase 194 roadmap checkbox: not complete.

The paused fixer worktree is:

```text
/Users/callumcowie/repos/Aether/.claude/worktrees/rf-194-56551-1787510689
```

Its branch is:

```text
gsd-reviewfix/194-56551
```

It is based on the same `bce085ce` commit and its current dirty diff contains
exactly 138 unstaged, uncommitted insertions in:

- `cmd/codex_build.go`;
- `cmd/forced_reviewer_waiver.go`;
- `cmd/forced_reviewer_waiver_test.go`.

The change appears to implement the intended CR-01 repair: reopen the
forced-reviewer decision window for a retried interactive build, without
opening it for non-interactive or automatic-fix paths. Its new regression test
models attempt 1 dispatching and an owner declining before attempt 2 dispatches.

The recovery sentinel is:

```text
.planning/phases/194-the-queen-decides-the-team/
  .review-fix-recovery-pending.json
```

It points at the worktree and review-fix branch above.

### Why a fresh GSD fixer is unsafe

The installed GSD recovery logic assumes an existing recovery sentinel belongs
to an already-committed orphaned worktree. It may run a forced worktree removal
and branch deletion before starting again.

That assumption is false here: the current fix is uncommitted. Running a fresh
`/gsd-code-review --fix`, or manually cleaning the worktree/branch/sentinel,
could permanently discard the final Phase 194 fix.

### Safe recovery sequence

Preferred sequence after Claude usage resets:

1. Resume the original Claude review/fix session.
2. Inspect the existing worktree and confirm its dirty diff is still present.
3. Finish and run focused tests for the three-file retry fix.
4. Stage only the three named fix files in the review-fix worktree and commit
   them atomically on `gsd-reviewfix/194-56551`. Do not use `git add .` or
   include unrelated dirty/untracked artifacts.
5. Produce the final `194-REVIEW-FIX.md` record.
6. Fast-forward or otherwise safely integrate the review-fix branch into
   `oracle-reinstate`.
7. Independently review the final fix; the current uncommitted version has not
   itself been reviewed.
8. Re-run Phase 194 verification. The present verification artifact predates
   the full review-fix sequence.
9. After inspecting each intended planning record, stage those records by
   explicit path and commit the review, fix, verification, retained iteration
   history, and intentionally deferred one-worker UX todo. Do not use
   `git add .`; the main checkout contains unrelated dirty/untracked GSD state.
10. Only after the work is durable, remove the worktree, temporary branch, and
    recovery sentinel through the normal GSD completion path.
11. Complete the Phase 194 transition before starting parity implementation.

If the original Claude session cannot be resumed, preserve the dirty worktree
in a durable commit or patch **before** running any recovery or cleanup command.

### Reserved collision area

Until Phase 194 is complete, do not implement Codex parity changes in:

- `cmd/codex_build.go`;
- `cmd/forced_reviewer_waiver.go`;
- `cmd/forced_reviewer_waiver_test.go`;
- `.planning/phases/194-the-queen-decides-the-team/**`;
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md`;
- the active GSD worktree or branch.

Start parity work from the finalized Phase 194 tip, not the current dirty main
checkout and not the dirty review-fix worktree.

## Existing Codex foundation

### Native Codex capabilities

Current Codex supports the high-level primitives needed for this design, though
Aether must still enforce or honestly report permission and scoped-write
limitations:

- Agent Skills package instructions, references, scripts, and assets with
  progressive disclosure;
- explicit skill invocation and implicit description-based matching;
- native subagents and visible subagent threads;
- custom agent roles in `.codex/agents/*.toml`;
- repository guidance through `AGENTS.md`.

Official references:

- <https://learn.chatgpt.com/docs/build-skills>
- <https://learn.chatgpt.com/docs/agent-configuration/subagents>
- <https://learn.chatgpt.com/docs/agent-configuration/agents-md>

### Existing Aether Codex assets

1. **27 Codex caste definitions**

   Canonical source: `.codex/agents/*.toml`.

   These include Builder, Watcher, Queen, Scout, Route-Setter, Architect,
   specialists, four Surveyors, and delivery/review roles.

2. **14 installed Aether Codex skills on this machine**

   Five shared/router shims plus nine command-shaped skills:

   - `aether-init`
   - `aether-discuss`
   - `aether-oracle`
   - `aether-colonize`
   - `aether-plan`
   - `aether-build`
   - `aether-continue`
   - `aether-swarm`
   - `aether-seal`

   The generator is `cmd/platform_sync.go`, particularly
   `codexSkillShims`, `codexCommandSkillShims`, and
   `renderCodexCommandSkillShimBody`.

3. **Full lifecycle skill source**

   Canonical source includes:

   - `.aether/skills/colony/aether-colony-creation/SKILL.md`
   - `.aether/skills/colony/aether-colony-research/SKILL.md`
   - `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`

   Installation intentionally emits smaller generated shims instead of copying
   all 86 Aether worker skills into Codex discovery. Worker skill content is
   matched and injected into each runtime brief in-process.

4. **Go-owned execution and state contracts**

   The runtime already provides:

   - plan-only/manifest paths;
   - caste names and deterministic worker identities;
   - task IDs, execution waves, prompts/brief paths, skill sections, and
     permission profiles;
   - spawn-log and spawn-complete ledgers;
   - completion packet validation;
   - build-completion staging for crash recovery;
   - plan/build/continue/colonize/swarm/seal finalizers;
   - stale-manifest and stale-completion rejection.

5. **Two existing execution lanes**

   - Headless/direct Go dispatch launches `codex exec` workers.
   - Wrapper/native orchestration can request a manifest without dispatch and
     let the host platform spawn visible workers before finalizing through Go.

The correct design should preserve both lanes rather than pretending they are
identical.

## Current parity gaps

### Gap 1 — Phase 194 build and continue semantics are absent from Codex

Claude and OpenCode build wrappers now include:

- Queen-selected optional castes;
- `--castes` and per-caste `--caste-why` inputs;
- plain-English inclusion/exclusion reasons;
- a visible owner team check-in;
- proceed, trim, forced-reviewer decline, and redirect choices;
- manifest re-fetch after a trim, decline, or redirect;
- exact prompt composition:
  `context_capsule + brief/brief_path + skill_section + latest owner steering`;
- open-decision collection between waves;
- forced-reviewer decision-window closure when dispatch starts;
- Go-owned completion staging and finalization.

Claude/OpenCode continue wrappers now let the Queen choose and justify the
heavy reviewer team, while forcing reviewers only for five named risks:

- authentication or credentials;
- payments;
- data deletion;
- database migrations;
- release sign-off.

The current Codex guide in `cmd/command_guide.go` fetches manifests and asks for
visible subagent panels, but does not include the complete Phase 194 team
selection, check-in, reviewer waiver, prompt composition, and closeout
behavior. Generated Codex skills faithfully reproduce this incomplete guide.

This is the clearest proof that the remaining issue is semantic synchronization,
not merely installing another skill.

### Gap 2 — command classification is inaccurate

The repository currently has 64 `.aether/commands/*.yaml` definitions. Only
nine commands receive intelligent command-shaped Codex skills. Most others are
classified by `command-guide` as literal:

```text
AETHER_OUTPUT_MODE=visual aether <yaml-filename> $ARGUMENTS
```

At least eight of those generated mappings have no same-named Cobra command in
the current binary:

- `archaeology`
- `ask`
- `assumptions`
- `chaos`
- `dream`
- `interpret`
- `organize`
- `profile`

Five are pure agent workflows in Claude/OpenCode rather than literal runtime
commands: `archaeology`, `chaos`, `dream`, `interpret`, and `organize`.

Three map to differently named low-level commands:

- `ask` uses `colony-prime --question`;
- `assumptions` uses `assumptions-analyze`, `assumptions-list`, and
  `assumptions-validate`;
- `profile` uses `profile-read`, `profile-update`, and `behavior-observe`.

Other meaningful wrapper intelligence is also lost when treated as literal:

- `skill-create` is an Oracle research and user-wizard workflow, not merely a
  low-level skill writer;
- `queen-compose` has an empty-input interview path;
- `lay-eggs` has recovery and verification behavior;
- `bump-version` has release follow-up behavior;
- curated help performs routing rather than just printing Cobra help.

Several of the nine intelligent guides also lag their current YAML/wrapper
sources, notably Oracle proposal/brief approval, init research/shelf routing,
and discuss question grounding/persistence.

Every one of the 64 commands needs an explicit, tested disposition:

1. native intelligent Codex skill;
2. intentional literal CLI passthrough;
3. alias/composition over real low-level commands;
4. intentionally unsupported or retired.

Full parity does not require 64 tiny skills. It does require 64 honest mappings.

### Gap 3 — native Codex conducting is advisory, not mechanically complete

The lifecycle skill currently says to use visible Task/subagent panels, but it
does not define a full Codex-native adapter protocol.

Required conductor behavior:

- normalize every manifest caste/`agent_name` to an exact Codex `agent_type`;
- fail closed if a required role is unavailable;
- use only workers contained in the fresh manifest;
- ensure the skill and runtime never both own dispatch;
- preserve execution-wave dependencies;
- queue workers when manifest width exceeds host concurrency;
- keep later waves blocked until all prerequisites reach acceptable terminal
  states;
- record exactly one spawn and one terminal completion per dispatch;
- collect and validate structured terminal results;
- reject task, caste, identity, stage, wave, or status mismatches;
- stop downstream waves after failure, timeout, cancellation, or incomplete
  results;
- assemble or normalize completion packets deterministically;
- stage and finalize exactly once;
- resume safely after parent interruption;
- support steering/decision answers without reconstructing runtime-owned prompt
  sections.

For native Codex dispatch, prefer passing only the runtime-composed prompt and
using no inherited conversational history when possible. The manifest brief
already carries the intended colony context, and duplicating parent history can
increase prompt size, leak unrelated context, and undermine deterministic
brief composition.

### Gap 4 — concurrency and nested delegation need explicit rules

The audited Codex host allowed four active agents including the coordinator,
while Aether manifests can budget four to eight workers in a wave.

The conductor must queue excess workers inside the same wave without starting a
dependent wave early.

Aether also has a `spawn-can-spawn --enforce` budget guard. The Codex Builder
definition permits child spawning, but does not consistently require that
runtime guard before every nested spawn. Either:

- add and test the guard in native role instructions; or
- disable nested native delegation until it can be mechanically enforced.

### Gap 5 — permission parity remains limited

`pkg/codex/permission_profile.go` currently makes only Includer mechanically
`repository_read_only`. Other roles receive workspace-write sandboxing plus
behavioral restrictions such as tests-only, docs-only, survey-only, or
analysis-only.

The direct `codex exec` adapter can select read-only or workspace-write mode.
Native Codex subagent roles do not currently provide a dynamic per-spawn
permission override in Aether's conductor contract.

Required behavior:

- ensure the native Includer role is truly read-only;
- do not claim that narrow `scoped_write` or `test_write` restrictions are
  host-enforced when they are only prompt instructions;
- reject a manifest profile when the selected native host cannot enforce the
  minimum required boundary;
- retain an explicit platform-difference contract for unavoidable limitations.

### Gap 6 — prompt budgets can drop skills in the headless lane

The direct Codex prompt assembler has a 24K-character default and trims the
skill section first when the finished prompt exceeds it. Separately, Aether's
derived assembled worker-context ceiling is 23.7K characters: a 4K delivered
compact capsule, 8K skill injection, 3.5K phase research, 2.2K code-graph
context, and a 6K allowance for task content. The direct lane also adds caste
instructions and prompt framing, so the two ceilings are close enough that
large valid worker contexts can lose their skill section.

This makes worker skill content least reliable on the largest tasks. The native
lane should pass the runtime-composed prompt without reconstruction, while the
headless lane needs explicit budget tests and a deliberate retention policy.

### Gap 7 — install and update do not reliably target active Codex

The source install path currently writes Codex agents and generated skills
under `homeDir/.codex`, including:

```text
~/.codex/agents
~/.codex/skills/aether
```

It does not consistently resolve a custom `CODEX_HOME`. This audited session
uses `/Users/callumcowie/.codex-5x`; it works because that installation contains
symlinks back to `.codex`, not because Aether resolved the active home.

The normal hub-backed `aether update` path syncs Codex agents but does not
regenerate or copy the global Codex command skills. Users can therefore retain
stale installed skills after updating Aether.

Required delivery changes:

- resolve explicit `CODEX_HOME`, with documented fallback;
- support `$HOME/.agents/skills` for preferred user-level discovery and repo
  `.agents/skills` where a repository-owned skill is intended, while retaining
  the deprecated-but-compatible `$CODEX_HOME/skills` path deliberately;
- refresh shipped skills during `aether update`;
- prune removed shipped skills without deleting custom skills;
- preserve stable/dev home isolation;
- verify installed agent and skill hashes/versions in integrity and medic;
- give accurate restart/discovery guidance after installation.

### Gap 8 — stale counts and stale parity claims

The repository contains 64 command definitions and 14 generated Codex skills,
but some medic, update, release, and documentation checks still describe 60
commands or five Codex shims.

Counts should be derived from the canonical catalog and `codexSkillShims()`
rather than duplicated constants.

The existing Codex gap map predates Phase 194 and overstates current lifecycle
parity. The classic command parity matrix tracks YAML, Claude, and OpenCode
coverage, but not Codex semantics.

The platform contract currently and correctly marks Codex as:

- secondary support tier;
- worker dispatch proven through `codex exec`;
- named caste routing limited;
- permission isolation limited;
- native command surface unavailable.

Rather than changing the existing slash-command statement to something
misleading, add a distinct `native_skill_surface` capability and mark it proven
only after installation, discovery, semantic, and live invocation tests pass.

### Gap 9 — tests prove presence rather than behavior

Focused existing parity tests passed during the audit even though the Phase 194
semantic drift above is real.

Current tests mainly prove:

- expected agent filenames exist;
- command guides have non-empty steps or expected anchor phrases;
- generated skills reproduce the command guide;
- YAML/Claude/OpenCode wrappers exist for command names.

An incomplete guide and a perfectly generated incomplete skill therefore pass
together.

Required improvement: normalize each platform workflow into a shared semantic
contract or workflow AST/fingerprint and compare behaviors, allowing only
explicitly recorded platform differences.

## Recommended target architecture

### Do not introduce a third-party orchestration framework

Use native Codex Agent Skills and custom subagents as the presentation and
execution surface. Retain Aether's existing Go manifests and finalizers as the
state/safety authority.

### Borrow GSD's packaging shape, not its execution ownership

The GSD example at:

```text
/Users/callumcowie/repos/gsd-2/gsd-orchestrator/
```

uses one `SKILL.md` with routed workflow and reference files. It is a good
progressive-disclosure packaging model. GSD itself runs as a subprocess that
owns its internal planning/build loop; Aether should not copy that ownership
model because Aether already exposes manifests, native worker dispatch, and
finalizers.

Suggested source package:

```text
aether-orchestrator/
  SKILL.md
  workflows/
    create.md
    research.md
    plan.md
    colonize.md
    build.md
    continue.md
    swarm.md
    seal.md
  references/
    command-disposition.md
    manifest-contract.md
    caste-routing.md
    prompt-composition.md
    permission-profiles.md
    terminal-result-schema.md
    completion-and-finalizers.md
    interruption-recovery.md
```

Keep small command-shaped skills such as `aether-build` and `aether-plan` as
discoverable entrypoints if useful. They should route to the shared package and
runtime guide rather than duplicate full workflows.

### Establish one semantic source

Do not hand-copy the Claude Markdown wrappers into another permanent Codex-only
surface.

Extend the structured `.aether/commands/*.yaml` and/or typed Go command catalog
so the canonical workflow describes:

- runtime command and low-level command mappings;
- interviews and decision moments;
- user check-ins and approval semantics;
- team/reviewer proposals and reasons;
- prompt-channel ordering;
- manifest and boundary gates;
- native caste dispatch and wave dependencies;
- terminal-result requirements;
- staging and finalizer selection;
- retry, replay, and recovery behavior;
- explicit platform-specific differences.

Generate or validate Claude, OpenCode, and Codex surfaces from that contract.
The existing Codex skill generator in `cmd/platform_sync.go` is already half of
this architecture.

### Preserve two intentional Codex lanes

1. **Interactive/native Codex**

   Agent Skills + native custom subagents + visible worker threads + Go
   manifest/finalizer.

2. **Headless/raw CLI**

   Existing Go adapter + `codex exec` workers + structured runtime output.

Test both. Do not claim the process-based headless path proves native Codex
skill/subagent behavior.

## Implementation plan

### Phase 0 — safely close Phase 194

**Estimate:** 0.5–1 day after Claude access resumes.

- Preserve, finish, test, commit, and independently review CR-01.
- Re-run verification and complete the GSD phase transition.
- Keep the one-worker check-in todo explicitly deferred unless the owner chooses
  to schedule it.
- Create the parity branch from the finalized Phase 194 commit.

### Phase 1 — define the parity contract

**Estimate:** 1–2 days.

- Inventory all 64 commands.
- Assign every command one explicit Codex disposition.
- Define lifecycle parity acceptance clauses.
- Define permitted platform differences.
- Add a separate `native_skill_surface` platform capability.
- Decide whether command-specific skills remain aliases or whether a single
  orchestrator skill is the main public entrypoint.

### Phase 2 — create one semantic workflow source

**Estimate:** 3–5 days.

- Bring build and continue up to the finalized Phase 194 behavior.
- Align Oracle, init, and discuss with current YAML/wrapper semantics.
- Define exact context-capsule, brief, skill, and decision-answer ordering.
- Generate Codex skills from the canonical contract.
- Reduce the shared lifecycle skill to a thin router or generate it from the
  same source.
- Reject nonexistent literal mappings during generation or tests.

### Phase 3 — harden the native Codex conductor

**Estimate:** 4–7 days.

- Implement exact caste-to-agent-type mapping.
- Add bounded same-wave scheduling and dependency barriers.
- Add terminal result/schema validation.
- Add failure, timeout, cancellation, and partial-wave propagation.
- Add spawn ledger and exactly-once checks.
- Add deterministic completion-packet normalization/assembly.
- Add interruption/replay recovery.
- Enforce or disable nested native delegation.
- Preserve one launch owner per manifest.

Implementation can live primarily in the generated skill protocol and existing
runtime helpers. Add a small deterministic CLI helper only where LLM-authored
JSON or replay behavior cannot be made reliable through existing finalizers.

### Phase 4 — audit all 27 agent contracts

**Estimate:** 3–5 days, partially parallel with Phase 3.

- Compare role intent, output schema, handoff, delegation, permissions, and
  stop conditions across Claude, OpenCode, and Codex.
- Record intentional differences in a machine-readable allowlist.
- Make native read-only roles mechanically read-only where possible.
- Align Builder nested-spawn rules with `spawn-can-spawn --enforce`.
- Validate the canonical `.codex/agents` tree; investigate the stale
  `.aether/codex` mirror before retiring it.

### Phase 5 — fix installation, update, and release delivery

**Estimate:** 2–4 days.

- Honor default and custom `CODEX_HOME`.
- Install current Agent Skill layout while retaining deliberate compatibility.
- Refresh skills during normal update.
- Preserve custom skills and prune removed managed skills.
- Derive command/agent/skill counts dynamically.
- Add skill hashes/versions to integrity and medic checks.
- Cover npm package and GoReleaser artifacts.
- Maintain stable/dev isolation.

### Phase 6 — acceptance and live proof

**Estimate:** 2–4 days.

- Add deterministic fake-provider integration tests.
- Run a real Codex smoke test in a disposable repository.
- Test default and custom Codex homes.
- Exercise one-worker, multi-worker, high-risk reviewer, retry, failure,
  interruption, and resume paths.
- Promote Codex support only after evidence is stored and reviewed.

## Required tests and acceptance gates

### Semantic contract

- Every YAML command has exactly one Codex disposition.
- Every literal mapping resolves to a real Cobra command.
- Generated skills have valid, unique, deterministic frontmatter/content.
- Build includes `--castes`, all `--caste-why` values, team check-in,
  trim/decline re-fetch, open-decision steering, completion staging, and exact
  finalization.
- Continue includes reviewer selection, named-risk forcing/waiver, heavy
  manifest dispatch, and exact `continue-finalize` invocation.
- Claude, OpenCode, and Codex normalize to the same workflow contract except
  for explicitly allowlisted differences.

### Native multi-agent execution

- Capacity-one, capacity-three, and over-capacity wave tests.
- No dependent wave begins before every prerequisite reaches a valid terminal
  state.
- Exact `agent_type`; exactly one spawn per manifest dispatch; no unmanifested
  worker.
- Completed, blocked, failed, timed-out, cancelled, and incomplete results.
- Partial-wave failure stops downstream work.
- Interrupted/replayed runs do not double-spawn or double-finalize.
- Nested spawning is budget-guarded or unavailable.
- Unsupported permission profiles fail closed.

### Runtime and state

- Boundary guidance prevents spawning until a fresh manifest is fetched.
- Stale manifests and stale completion packets are rejected.
- Temporary worker/completion artifacts remain outside `.aether/data` until
  accepted by Go-owned staging/finalizers.
- Same-phase retry works, including the Phase 194 reviewer-decline case.
- Completion recovery is idempotent.
- Dirty/uncommitted worktrees are never force-removed by recovery.

### Packaging and release

- Source install, publish, and update create the correct skills under default
  and custom Codex homes.
- Update replaces stale managed skills and preserves custom skills.
- Dev operations cannot overwrite stable assets without explicit opt-in.
- Packed release checks include at least `aether-build/SKILL.md`, the complete
  expected skill set, all 27 agents, and expected guide/version hashes.
- Integrity and medic inspect what users actually have installed.

### Live acceptance proof

Use a fresh disposable repository and a genuinely separate custom
`CODEX_HOME`.

Prove:

1. Codex discovers Aether skills and all expected agent roles.
2. `init -> plan -> build -> continue -> seal` completes through the native
   skill lane.
3. A multi-worker build uses exact manifest castes and respects the actual host
   concurrency limit.
4. No worker is invented, silently skipped, or double-spawned.
5. Team selection, trim, forced-reviewer decline, and retry behave correctly.
6. Continue selects and runs appropriate independent reviewers.
7. Partial failure prevents dependent work.
8. An interrupted run resumes without double-finalization.
9. Recovery preserves dirty worktrees.
10. `aether update` replaces stale managed skills while preserving custom ones.

## MDS and GSD distinction

If “MDS” refers to the Max for Live MDS support in Aether, it demonstrates a
useful but smaller pattern:

- `cmd/codex_colonize.go` detects MDS/Max for Live repositories;
- `cmd/codex_plan.go` selects MDS-specific planning templates and verification
  surfaces.

That is a domain adapter. It adds structured Codex context for one type of
repository; it does not reproduce a complete platform orchestration surface.

If “MDS” was intended to mean GSD, the source package at
`/Users/callumcowie/repos/gsd-2/gsd-orchestrator` is a good Agent Skill
packaging reference. However:

- the active Phase 194 workflow is still Claude-owned;
- the current GSD review/fix is unfinished;
- no GSD-specific Codex skills, agents, or workflows were installed under the
  audited Codex home;
- GSD's subprocess ownership model should not replace Aether's existing
  manifests and finalizers.

## Source map

Read these first when resuming:

### Active Phase 194/GSD state

- `.planning/phases/194-the-queen-decides-the-team/194-REVIEW.md`
- `.planning/phases/194-the-queen-decides-the-team/194-REVIEW-FIX.md`
- `.planning/phases/194-the-queen-decides-the-team/194-VERIFICATION.md`
- `.planning/phases/194-the-queen-decides-the-team/.review-fix-recovery-pending.json`
- `.planning/STATE.md`
- `.planning/ROADMAP.md`
- `.planning/todos/pending/2026-08-23-one-worker-build-skips-the-checkin-pause.md`
- dirty fixer worktree:
  `/Users/callumcowie/repos/Aether/.claude/worktrees/rf-194-56551-1787510689`

### Current wrapper semantics

- `.claude/commands/ant/build.md`
- `.claude/commands/ant/continue.md`
- `.aether/commands/build.yaml`
- `.aether/commands/continue.yaml`
- `.aether/commands/oracle.yaml`
- `.aether/commands/init.yaml`
- `.aether/commands/discuss.yaml`

### Codex orchestration and packaging

- `cmd/command_guide.go`
- `cmd/platform_sync.go`
- `cmd/install_cmd.go`
- `cmd/update_cmd.go`
- `pkg/codex/platform_contract.go`
- `pkg/codex/platform_dispatch.go`
- `pkg/codex/worker.go`
- `pkg/codex/prompt.go`
- `pkg/codex/permission_profile.go`
- `.codex/agents/*.toml`
- `.aether/skills/colony/aether-colony-creation/SKILL.md`
- `.aether/skills/colony/aether-colony-research/SKILL.md`
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`

### Contracts and tests

- `.aether/docs/ARCHITECTURE_BOUNDARY.md`
- `.aether/docs/wrapper-host-contract.md`
- `.aether/docs/source-of-truth-map.md`
- `.aether/docs/codex-ant-workflow-gap-map.md` — useful history, but stale
- `.aether/docs/manual-provider-smoke-checklist.md`
- `cmd/command_guide_test.go`
- `cmd/classic_command_parity_test.go`
- `cmd/parity_test.go`
- `cmd/platform_sync_test.go`
- `cmd/codex_e2e_test.go`
- `cmd/update_cmd_test.go`
- `cmd/release_candidate_blackbox_test.go`

### GSD packaging reference

- `/Users/callumcowie/repos/gsd-2/gsd-orchestrator/SKILL.md`
- `/Users/callumcowie/repos/gsd-2/gsd-orchestrator/workflows/`
- `/Users/callumcowie/repos/gsd-2/gsd-orchestrator/references/`

## Decisions recommended for the owner

1. Finish Phase 194 before implementation. This is mandatory for safety, not a
   preference.
2. Target production lifecycle parity first rather than claiming immediate
   parity across all 64 commands.
3. Use one shared Aether orchestrator skill with focused references; retain
   small command skills only as entrypoints.
4. Keep Go as the only state/finalizer authority.
5. Treat native Codex and headless `codex exec` as two supported lanes with
   separate acceptance tests.
6. Extend a structured canonical command contract instead of manually copying
   Claude wrapper prose.
7. Do not mark Codex primary until a real provider smoke test and recovery test
   pass.

## Do not do these things

- Do not rerun GSD code-review fix while the dirty recovery worktree is
  uncommitted.
- Do not force-remove the Phase 194 worktree or delete its branch/sentinel.
- Do not start parity work from the current dirty checkout.
- Do not copy all 86 Aether worker skills into Codex discovery.
- Do not create 64 empty command skills merely to make a count match.
- Do not use Claude Markdown as a second permanent source of truth for Codex.
- Do not let both the Go runtime and native Codex own the same dispatch.
- Do not claim a behavioral restriction is a sandbox guarantee.
- Do not promote Codex support based only on substring/file-presence tests.

## Restart checklist

After Claude usage resets:

- [ ] Read this handoff.
- [ ] Inspect `git status` in both the main checkout and the Phase 194 worktree.
- [ ] Confirm the three-file dirty fix still exists.
- [ ] Resume/finish the original GSD reviewer-fixer flow safely.
- [ ] Commit and independently review the CR-01 retry fix.
- [ ] Re-run Phase 194 verification and complete the phase transition.
- [ ] Confirm the main working tree state and choose a clean parity branch.
- [ ] Re-audit build/continue semantics at the finalized Phase 194 tip.
- [ ] Create the explicit 64-command Codex disposition matrix.
- [ ] Plan the production lifecycle parity phases from the architecture above.

## Suggested restart prompt

```text
Read .planning/research/CODEX-NATIVE-PARITY-HANDOFF.md completely.

First, do not start Codex parity implementation. Inspect and safely complete the
paused Phase 194 GSD review-fix using the existing worktree and recovery
sentinel. Preserve all dirty work before any cleanup. Run focused tests,
independently review the final CR-01 fix, re-run Phase 194 verification, and
complete the phase transition.

After Phase 194 is durable and the implementation branch is clean, revalidate
the handoff against the finalized tip and create an implementation plan for
production native Codex lifecycle parity. Keep Go manifests/finalizers as the
state authority, use one shared Codex orchestrator skill plus native Codex
subagents, classify all 64 commands explicitly, and include install/update,
custom CODEX_HOME, interruption recovery, bounded concurrency, semantic parity,
and live-provider acceptance tests.
```

## Final recommendation

Wait for Claude access to resume and safely finish Phase 194. Then treat native
Codex lifecycle parity as a focused medium-sized integration project of roughly
10–15 engineering days. Only extend to the 15–25 day full command/agent audit
after the native lifecycle has passed real Codex acceptance testing.
