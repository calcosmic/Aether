# Aether Root-Cause and Recovery Report

**Date:** 2026-08-04

**Revision examined:** `2b86e1914123c8b2736017d29f36b7dffd225825`

**Companion audit:** [AETHER_END_TO_END_READINESS_REAUDIT_2026-08-04.md](AETHER_END_TO_END_READINESS_REAUDIT_2026-08-04.md)

**Purpose:** Explain why a previously useful Aether became difficult to trust, what the Go migration actually changed, and what should be done to make Aether a boringly reliable daily driver again.

**Change boundary:** Report only. No runtime, wrapper, installation, state, or release code was changed.

## Direct answer

The user's memory is credible: Aether could feel substantially more useful before the current architecture, even though the old implementation had technical weaknesses.

The decline was not caused simply by choosing Go, and it was not caused by adding too few features. It was caused by a **premature big-bang migration followed by architectural accretion**:

1. A working, prompt-native workflow was ported into compiled commands before behavioral parity was proven.
2. The migration's own audit failed, but the migration was still marked shipped.
3. The old runtime, fallback, fresh-install tests, and end-to-end tests were deleted before the replacement passed them.
4. Later recovery work added a TypeScript host, more manifests, more wrappers, more generated surfaces, more state, and more verification contracts around the incomplete Go replacement.
5. Milestones measured component completion and test counts more often than the user's actual outcome: install Aether, give it a task, let the Queen coordinate useful workers, resume safely, and finish without manually repairing the framework.

So the plain-English answer is: **yes, adding more has made the product worse in important ways—but because every addition created another boundary or policy that had to agree with all the others.** The useful core did not need this many moving parts.

The situation is recoverable. Aether does not need another total rewrite. It needs a product freeze, a smaller authority model, a restored real-user acceptance gate, and removal of redundant layers from the primary Claude Code path.

## What Aether is supposed to be

Aether's concept is simpler than its current implementation:

> Aether is an orchestration ledger for native coding agents. The Queen turns a goal into phases, gives specialized workers clear tasks and relevant skills, records what happened, independently verifies the result, and makes the work resumable.

That implies six essential capabilities:

1. **Queen orchestration** — decide which castes are useful and dispatch them through the host platform's real agent tools.
2. **Movement tracking** — record the goal, phase, tasks, workers, handoffs, blockers, and verification status.
3. **Context networking** — pass the right findings between agents without requiring the user to repeat them.
4. **Skills** — discover skill files from known folders, match a small number to the current task, and show why they were selected.
5. **Independent verification** — a Watcher or specialist checks work produced by a different worker.
6. **Rich ceremony** — make the colony legible and enjoyable without letting presentation own state or execution.

Everything else must justify itself by making one of those capabilities safer or easier. It must not become a second product inside Aether.

## The precise historical failure

### 1. Classic Aether had a coherent execution model

Before the cutover, Claude Code received a short Queen wrapper that loaded staged playbooks. The platform model performed the orchestration using its native tools. The shell runtime handled state and deterministic helpers.

This was technically untidy, but conceptually aligned:

```text
user -> Queen prompt -> staged playbooks -> native agents
                         |
                         +-> shell state/helpers
```

The model saw the method and the ceremony directly. There were fewer serialization boundaries between the Queen's decision and a worker being spawned. This is particularly valuable for cheaper models: they perform better when given one visible workflow and small, concrete tasks instead of being asked to reconcile multiple protocols.

Classic was not flawless. Git history contains many shell fixes for JSON escaping, file locking, state isolation, error suppression, and portability. Going back to raw shell as the permanent architecture would restore those risks. But the main workflow and the user's mental model were much closer to one another.

### 2. The Go conversion was attempted in days, not migrated seam by seam

The first Go storage commit landed on 2026-04-01 (`e4842659`). By 2026-04-04, the project had introduced storage, events, memory, graph, agents, LLM integration, hundreds of CLI commands, and a parity harness. Commit `28dfcef9` then rewired the command assets and playbooks to Go.

That is not inherently wrong, but it was an extremely wide change boundary. It simultaneously changed:

- state representation;
- storage and locking;
- event behavior;
- memory promotion;
- agent execution;
- CLI arguments and envelopes;
- command wrappers;
- install and distribution behavior;
- output and error behavior.

A safe migration would have replaced one seam at a time while the old end-to-end path remained executable. Instead, nearly the entire runtime boundary moved before one real user workflow was proven.

### 3. The migration audit explicitly said it had not passed

The archived v5.4 milestone audit at tag `v5.4.0` recorded:

- **5 of 37 requirements satisfied**;
- **12 partial**;
- **20 unsatisfied**;
- distribution not started;
- several phases missing verification;
- integration and flow checks still pending;
- a fail gate triggered.

The underlying parity summary was also much weaker than the eventual release language implied:

| Parity evidence | Recorded result |
|---|---:|
| Overlapping command cases | 195 |
| Passed | 70 |
| Skipped because the Go command was not ported | 105 |
| Parity gaps detected | 43 |
| Known breaks accepted | 12 |

The Go-only smoke suite primarily proved that commands did not panic and returned a JSON envelope. Some dependent swarm commands were expected to return `ok:false` because the tests intentionally ran without their prerequisites. That is useful component evidence, but it is not proof that a colony works.

Despite this, the milestone was archived as `SHIPPED`, and later documentation described all 37 requirements as complete. That is the first decisive break between **project ceremony** and **product truth**.

### 4. The fallback and behavioral oracle were deleted at exactly the wrong time

On 2026-04-05, commit `0063be8b` removed the shell and Node runtime. That commit deleted approximately:

| Removed area | Files | Deleted lines |
|---|---:|---:|
| Aether shell runtime | 57 | 30,214 |
| Test suite under `tests/` | 114 | 47,795 |
| Vendored GSD agents, commands, and workflows | 191 | 44,337 |

This included the 430-line fresh-install test and the old lifecycle, update, state, learning, skills, Oracle, swarm, and integration suites.

The most revealing evidence is that the same commit added `.claude/PROMPT-GO-CONVERSION.md`. That document stated:

- only 153 of 305 shell subcommands were implemented in Go;
- 28 critical commands used by slash commands were still missing;
- the replace-versus-coexist decision was still open;
- GSD was to stay in place;
- the shell should remain operational during transition.

The implementation commit did the opposite of its own transition plan. This is the single clearest root-cause event in Aether's history.

### 5. The months after cutover are a restoration loop

Since the first Go product release, the repository has accumulated 736 commits. The milestone names tell the story without interpretation:

- Visual Truth and Core Hardening
- Live Dispatch Truth and Recovery
- Runtime Truth Recovery
- Planning Pipeline Recovery
- Colony Recovery
- Aether Unification
- Hybrid Runtime Boundary and Orchestration Recovery
- Classic Restoration
- Hybrid Runtime Parity and Release Gate
- TypeScript Host Cutover
- Host Contract Hardening and Wrapper Reality Check
- Live Colony
- Grounded Planning and Ceremony Restore
- Daily Driver Reliability
- Hybrid Architecture Salvage
- the current Fail Loudly, Context Reaches Workers, Research Feeds Planning, Core Lifecycle Commands, and Prove It programme

These are not all wasted efforts. Many contain good fixes. But the repeated use of *truth*, *restore*, *recovery*, *parity*, *reliability*, and *salvage* shows that Aether has repeatedly repaired symptoms of the same incomplete boundary migration.

### 6. The TypeScript host restored behavior by adding another critical boundary

The hybrid research correctly concluded that Go should own safety rather than the living behavior of the colony. However, the implementation added a TypeScript host between the wrapper and Go while the wrapper remained the actual native-agent conductor.

The current Claude build path is approximately:

```text
Claude wrapper
  -> aether host build --dry-run
     -> TypeScript host
        -> aether build --plan-only
           -> Go manifest and state machinery
  -> Claude wrapper parses returned manifest
  -> Claude wrapper spawns native workers
  -> wrapper calls Go spawn-log / spawn-complete
  -> wrapper calls Go build-finalize
```

The TypeScript host can also dispatch workers in a different path, which is why the wrapper has to warn not to invoke it without `--dry-run` or it will duplicate spawning.

This is an ownership smell. A pass-through layer now requires Node, npm dependencies, built `dist` files, preflight configuration, platform adapters, schemas, and another test suite. The current clean-install failure on missing `js-yaml` is a direct consequence.

The host was intended to restore Aether's soul. In the primary wrapper path, it instead became another place where the same plan can be represented incorrectly.

## Why more tests did not make it feel reliable

Aether currently has strong unit and race-test coverage. That is valuable. The problem is that most failures now live **between** tested components.

For example:

- Go can correctly generate a manifest while the installed TypeScript host lacks its dependency.
- Update can correctly copy Aether's files while incorrectly replacing GSD's settings.
- A binary and hub can both report version 1.0.46 while containing code from different commits.
- Every state helper can pass isolated tests while the lifecycle performs several separate writes and fails halfway through.
- A wrapper contract test can prove the words `handoff` and `build-finalize` exist without proving a cheap model successfully follows the workflow.
- The race detector can pass while logical same-process locking still permits lost updates.

The current project tests components heavily but does not make the complete installed product the final authority.

In plain English: all the kitchen appliances can pass inspection while the meal still never reaches the table.

## What became worse as features were added

### Interface multiplication

Current Aether has roughly:

- 125,000 lines of production Go;
- 11,500 lines of TypeScript-host source;
- 401 registered CLI commands;
- separate Claude and OpenCode wrappers;
- YAML command sources and generated/mirrored surfaces;
- Codex skills and TOML agent definitions;
- global hub copies and repository-local copies;
- source, npm, GitHub release, installed binary, stable hub, and dev hub identities.

The problem is not line count by itself. It is the number of places that must agree for one build to work.

### Behavior moved away from the editable Queen

Classic Aether's method lived where the model could read it. During the Go migration, orchestration, ceremony, memory policy, and prompt behavior were progressively compiled or split among runtime code, wrappers, YAML, and later TypeScript.

This made simple behavioral changes require synchronized edits across several surfaces. When one surface lagged, the product became internally contradictory.

### Error handling became strict without becoming self-healing

The recent fail-loud work is correct: silently continuing after a broken verification command is dangerous. But a daily-driver system needs both halves:

1. identify the exact failure;
2. preserve a consistent state and offer one deterministic recovery action.

Aether often has the first half but not the second. It exposes more truth, yet leaves the user to repair manifests, state, installation, settings, or phase bookkeeping. That feels worse than the old permissive path even when it is technically more honest.

### New capabilities were built ahead of the acceptance gate

Learning, Hive behavior, research confidence loops, all-caste selection, typed controls, rich ceremony, and unreachable-command recovery can all be valuable. They should not have been ahead of:

- safe filesystem boundaries;
- correct state locking and phase identity;
- a clean install that can plan;
- safe coexistence with the development host;
- one exact release identity;
- repeated real tasks with a cheap model and no framework repair.

This is why the score barely changed after 216 commits: the work was real, but it was not ordered by the user's risk.

## What was genuinely better before

The following qualities should be restored intentionally:

1. **The Queen was visibly in charge.** The model saw a comprehensible staged method and used native worker tools directly.
2. **The workflow matched the metaphor.** Goal, phase, worker wave, verification, learning, next phase.
3. **The happy path was small.** Five commands could take a user from setup to completion.
4. **The ceremony explained the work.** It was not merely a renderer around opaque manifests.
5. **Skills and context were part of the worker brief.** They did not require the user to know internal commands.
6. **Cheaper models received explicit method.** They were asked to execute bounded steps rather than reconcile an architecture.

The shell's unsafe writes, fragile parsing, portability problems, and silent errors should not be restored. The behavior is the baseline, not the implementation.

## The target architecture

### One primary path

For Claude Code, the daily-driver path should become:

```text
user
  -> thin Queen wrapper / skill
     -> Go kernel produces a compact plan-only packet
     -> Queen spawns native Claude workers
     -> workers return structured handoffs
     -> Go kernel atomically validates and records the result
     -> event renderer shows ceremony and next action
```

Concretely, a build should be:

```text
/ant-build 3
  -> aether build 3 --plan-only
  -> spawn exactly the returned wave through Claude's native Agent tool
  -> aether build-finalize 3 --completion-file ...
```

The TypeScript host should be removed from the primary Claude wrapper path. It can remain temporarily as an experimental headless/provider adapter until its unique value is proven. It must not be required for install, plan, build, continue, resume, or seal.

This is not an anti-TypeScript decision. It removes a redundant hop. If a future platform genuinely cannot use native wrappers, the host can serve that platform behind a separate adapter contract.

### Go as a small safety kernel

Go should own only deterministic operations where compilation is helpful:

- path validation and safe file operations;
- one atomic state transaction API;
- schema validation and migration;
- locks with real ownership semantics;
- plan-only manifests and finalizers;
- installation, update, content manifests, and release integrity;
- append-only events and recovery snapshots;
- deterministic skill discovery and matching;
- verification command execution and exit-code capture.

Go should not decide the Queen's personality, prose, research reasoning, ceremony wording, or provider-specific agent behavior.

### Editable assets as the colony brain

Markdown, YAML, JSON, and TOML should define:

- caste purpose and tool permissions;
- Queen method;
- worker task-packet template;
- skill metadata and content;
- ceremony copy;
- planning and verification policy.

There should be one canonical asset for each behavior. Platform wrappers may be generated from it, but generated files must carry content hashes and must not be edited independently.

### Native platforms own worker execution

Claude Code, OpenCode, and Codex already know how to execute agents. Aether should organize those capabilities rather than becoming another provider runtime.

This removes provider API keys, SDK drift, duplicate preflights, and process supervision from the normal interactive path. It also lets the user choose a cheaper native model without Aether reimplementing the provider.

### Ceremony as a projection, not control logic

Rich ceremony should stay. It is part of what makes Aether understandable and enjoyable.

It should render from an append-only event stream:

```text
phase framed -> worker spawned -> worker completed -> watcher verdict -> phase advanced
```

If ceremony fails, the build should remain valid. If the build fails, ceremony must never claim success.

### Skills should be boring and visible

The skill system only needs to:

1. scan canonical global and repository folders;
2. parse small metadata;
3. match by workflow, caste, task, and detected stack;
4. choose a bounded number;
5. insert them into the worker packet;
6. show `selected because ...` in an inspect command.

The Queen may override a match, but users should never have to manually copy skill content into prompts or repair generated mirrors.

## Designing for cheap models

Cheap-model performance is not mainly a routing problem. It is an ambiguity and context problem.

Every worker should receive one compact packet:

| Field | Purpose |
|---|---|
| Objective | One concrete outcome |
| Owned files | Exact write boundary |
| Relevant context | Only facts needed for this task |
| Constraints | Hard prohibitions and user preferences |
| Skills | At most the small matched set |
| Verification | Exact commands or observable checks |
| Handoff schema | What the next agent needs |

The packet should not ask a Builder to understand Aether's state model, host ownership, finalizer contract, ceremony rules, planning depth, Hive policy, or release architecture. The Queen and kernel handle those.

The independent Watcher should receive the task criteria and resulting diff, not the Builder's reasoning. That preserves the original colony advantage without requiring expensive models everywhere.

The correct cheap-model milestone is therefore not automatic model allocation. It is:

> Three representative tasks complete repeatedly on the user's chosen cheap model with no manual Aether repair.

## Exact recovery programme

### Recovery Phase 0 — Stop expanding

Immediately freeze new feature work and default-on cross-repository learning.

Allowed work:

- P0 safety fixes;
- state correctness;
- install/update coexistence;
- removal of redundant critical-path layers;
- real acceptance tests;
- release provenance.

Paused work:

- expanding the public command surface;
- reconnecting all unreachable commands;
- new castes;
- default-on Hive promotion or retrieval;
- further ceremony features not required by the golden workflow;
- additional host abstractions.

Exit condition: one signed, one-page product contract defines the primary user, primary platform, seven lifecycle actions, state authority, and non-goals.

### Recovery Phase 1 — Establish the real behavioral baseline

Do not assume the best baseline from tag names. Test both likely candidates:

- `v5.3.3`: before the broad Go wiring;
- `v5.4.0`: the last pre-deletion hybrid state.

Run them in disposable downstream repositories using the same three tasks and the user's preferred cheap model. Record:

- what the Queen says and asks;
- which workers actually spawn;
- what context each worker receives;
- how skills are selected;
- how interruptions resume;
- what state changes;
- how much manual intervention is required;
- what genuinely feels useful.

The result is a behavioral specification, not a rollback branch.

Exit condition: the user approves a concise golden-workflow transcript and behavior checklist.

### Recovery Phase 2 — Repair the safety kernel

Fix before any further orchestration work:

1. swarm/chamber path containment;
2. tar and ZIP extraction containment, link handling, and malicious fixtures;
3. same-process lock ownership and lock-upgrade semantics;
4. one unambiguous phase identity model;
5. transactional init, continue, and seal operations;
6. patched Go toolchain and TypeScript dependencies while the host still exists.

Exit condition: adversarial tests pass, state fault-injection tests leave either the old complete state or the new complete state, and no partial lifecycle state is observable.

### Recovery Phase 3 — Make install and update uneventful

Produce one install artifact containing everything the selected primary path needs. A clean install must not require a repair update.

Requirements:

- no repo-local `node_modules` for the primary Claude path;
- `.claude/settings.json` is structurally merged, never replaced wholesale;
- Aether owns only namespaced hook entries;
- repeated install/update is idempotent;
- uninstall can remove only Aether-owned entries;
- GSD hooks, permissions, and unrelated settings survive byte-for-byte where possible;
- the installed binary, assets, and source commit share one signed content manifest.

Exit condition: install, update, repeated update, and uninstall all pass against a repository containing GSD and unrelated Claude settings.

### Recovery Phase 4 — Collapse to one vertical slice

Make only this path authoritative on Claude Code:

```text
install -> init -> plan -> build -> continue -> resume -> seal
```

For each lifecycle action:

- the wrapper performs platform-native interaction;
- Go exposes one plan-only/finalize pair where agents are involved;
- Go owns every persistent mutation;
- failures return one recovery command;
- the TypeScript host is not on the path;
- ceremony reads events and does not duplicate state decisions.

Internal compatibility commands may remain temporarily, but they are hidden from the user contract and cannot be counted as product readiness.

Exit condition: a completely new environment runs the whole path without manual edits to `.aether`, `.claude`, state JSON, manifests, or installed dependencies.

### Recovery Phase 5 — Make `Prove It` the gate, not the final phase

The current Phase 171 idea is correct but ordered too late. Real downstream proof must run throughout recovery.

Use three fixed benchmark types:

1. a small bug fix in an existing codebase;
2. a multi-file feature requiring planning, Builder, and Watcher;
3. an interrupted build that resumes and completes.

Run each at least twice with the user's chosen cheap model in a downstream repository. A run passes only when:

- real workers spawned;
- claimed files and tests exist;
- an independent verification result was recorded;
- interruption did not lose or duplicate work;
- no Aether internals were manually edited;
- no undocumented retry or environment repair was needed;
- ceremony matched actual worker events;
- total user involvement was limited to intent approval and genuine product decisions.

Any failure becomes a release blocker. It must not be reclassified as optional UAT or documentation debt.

### Recovery Phase 6 — Release one exact product

Publish only the exact candidate that passed the downstream suite.

Release identity must bind:

- semantic version;
- Git commit;
- binary hash;
- asset manifest hash;
- platform wrapper hashes;
- dependency lock hashes;
- build toolchain version.

The same candidate must be tested through the same channel users install. Local source success does not authorize a GitHub or npm release.

Exit condition: a clean machine installs the public candidate and repeats the golden workflow with no difference from the tested candidate.

### Recovery Phase 7 — Reintroduce optional depth carefully

Only after the daily-driver gate stays green should Aether re-enable:

- cross-colony Hive learning;
- full Oracle confidence loops;
- automatic all-caste selection;
- advanced swarm dashboards;
- additional lifecycle variants;
- headless TypeScript/provider orchestration.

Each feature must be removable or disabled without changing the core lifecycle outcome.

## How the current GSD roadmap should change

The current roadmap should not continue in its existing order.

Recommended disposition:

| Current work | Recommendation |
|---|---|
| Phase 160 Fail Loudly | Keep the improvements; add deterministic rollback/recovery |
| Phase 161 Cheap Models | Reframe around benchmark success, not automatic allocation |
| Phase 162 Switch On Learning | Pause; do not add cross-repo writers before locks and provenance are fixed |
| Phase 163 Context Reaches Workers | Keep only context proven useful by golden task packets |
| Phase 164 Research Feeds Planning | Keep off the default path until the simple planning workflow passes |
| Phase 165 Core Lifecycle Commands | Preserve useful method, then simplify ownership and remove the host hop |
| Phase 166 Full Colony On Demand | Pause; castes may remain installed without all being runtime-active |
| Phase 167 Typed Control | Limit to invariants required by the vertical slice |
| Phases 168-169 Your Eyes Back | Preserve ceremony goals; implement as event projection after correctness |
| Phase 170 Reclaim The Unreachable | Replace with prune-or-justify; do not reconnect commands merely because they exist |
| Phase 171 Prove It | Move forward and make it the continuous acceptance gate |

GSD itself should apply three hard rules while finishing Aether:

1. A phase cannot be called shipped when mandatory UAT is incomplete.
2. Test counts cannot substitute for a clean installed lifecycle run.
3. No new subsystem may be added to fix a boundary until removal or simplification of the existing boundary has been evaluated first.

## What not to do

### Do not perform another full rewrite

A full TypeScript rewrite or a full shell rollback would repeat the same risk. Preserve the valuable Go kernel, but shrink its responsibility and surface.

### Do not fix only the six P0 findings and resume feature work

Those fixes are necessary, not sufficient. Without the authority simplification and real acceptance gate, new boundary failures will replace them.

### Do not use test volume as reassurance

Keep the tests, but classify them honestly:

- unit proof;
- contract proof;
- installed black-box proof;
- real provider/user proof.

Only the latter two establish daily-driver readiness.

### Do not restore every historical feature

The 401-command catalog is not a measure of value. Unreachable or duplicate commands should be deleted unless the golden workflow or a current user story requires them.

### Do not make rich ceremony part of correctness

Ceremony matters, but it must be downstream of truthful events. It should never be able to make a failed or partial operation look complete.

### Do not make the user the integration test

The user should not need to know about hub paths, generated mirrors, host dependencies, manifest schemas, state repairs, wrapper finalizers, or version alignment. If those surface during normal use, the product has failed its abstraction.

## The finish-line definition

Aether is ready to become the user's go-to framework when all of the following are true at the same exact public candidate:

- one install command produces a complete working installation;
- existing GSD and Claude configuration survives install and update;
- the Queen can init, plan, build, continue, resume, and seal using native agents;
- workers receive compact, relevant context and matched skills automatically;
- cheap-model benchmark tasks repeatedly complete without Aether repair;
- state remains consistent through crashes and concurrent activity;
- unsafe paths and archive entries are rejected;
- every failure gives one truthful reason and one safe recovery action;
- ceremony reflects real events;
- source, binary, hub, wrappers, npm, and GitHub identify the same bytes;
- no mandatory acceptance item is pending.

That is a smaller target than the current roadmap, but it is a much stronger product.

## Final conclusion

Aether did not become unusable because its core idea was too ambitious. Its core idea is coherent and useful.

It strayed because the project treated a wide runtime port as complete when its own evidence said otherwise, deleted the working reference too early, and then spent successive milestones rebuilding lost behavior through additional boundaries. Go exposed and solved some real safety problems, but it was also allowed to absorb orchestration behavior it should not own. TypeScript was then added to recover that behavior without removing the now-redundant path.

The correct response is not to keep circling and not to abandon the project. It is to stop, reduce, and prove:

1. preserve the old behavior as a specification;
2. retain Go as a boring safety kernel;
3. let the Queen use native platform agents directly;
4. remove the TypeScript host from the primary interactive path;
5. make skills automatic and inspectable;
6. make real cheap-model downstream tasks the gate for every phase;
7. publish only an exact candidate that passed that gate.

The desired product is not 401 commands and several orchestration engines. It is a Queen, a trustworthy ledger, a network of specialized workers, useful skills, independent verification, and a clear view of where the work is going. Aether can still be that.
