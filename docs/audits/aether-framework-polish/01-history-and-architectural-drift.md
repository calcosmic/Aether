# History And Architectural Drift

## Method And Limits

The audit inspected all refs, 4,038 reachable commits, local/remote branches, tags, selected commit contents, deleted/renamed files, tag-to-HEAD diffs, blame-relevant implementation, planning archives, and release workflows. The checkout is unusual: `main` is 306 commits ahead of `origin/main`, the checked-out commit is tagged `v1.24`, and the working tree contains pre-existing unreleased `1.0.42` edits. Historical claims below identify the tag or current working tree when that distinction matters.

Authorship is concentrated: `git shortlog -sn --all` attributes 3,990 commits to one identity, 47 to two closely related identities, and one to automation. This does not invalidate the work, but it means repeated internal plans and milestone reports are not independent validation.

## Original Problem

The root commit, `1d77cf37` on 2026-02-01, framed Aether as autonomous agent spawning, a three-layer memory system, and automatic error prevention. Its runnable implementation was three Python prototypes under `.aether/`, surrounded by Ralph research artifacts and shell monitoring. The README claimed agents would self-organize and the system would never repeat a mistake, while its own next-steps list still deferred orchestration, semantic code understanding, advanced verification, and production deployment.

The durable original insight was not the numerical or novelty claim. It was this product job: fresh model contexts should be able to read durable project state, perform bounded work, leave evidence, and hand the next context enough information to continue.

## GSD And Ralph Inheritance

History contains both `.ralph/` state and later `.claude/get-shit-done/` workflow material. The inherited patterns are visible in:

- Markdown specifications and phase plans as model-readable state.
- Fresh-context execution loops that reread repository artifacts.
- Plan, execute, verify, and advance ceremonies.
- Agent prompts and slash-command wrappers as the primary behavior layer.
- Context budgets, research artifacts, progress files, and completion summaries.
- Later direct inclusion of GSD workflow directories as compatibility/reference material.

Aether's original contribution was to add persistent runtime state, specialized ant identities, steering signals, a Queen coordination surface, and cross-project memory. The historical mistake was treating each new noun as evidence of a new capability before closing the underlying state-to-execution loop.

## Architectural Eras

| Era | Evidence | Intended improvement | Actual architectural effect |
| --- | --- | --- | --- |
| Research prototypes, Feb 1-7 | Root Python files; Ralph state/research | Prove autonomous spawning, memory, error prevention | Demonstrated concepts, not an integrated development lifecycle |
| Shell/prompt framework, `v2.0.2`, Feb 8 | 73 files; 10 shell; 45 Markdown | Package agent workflows and bootstrap | Fast iteration, but prompt prose and file writes were behavior and state |
| Reliability/state expansion, `v3.0.0`, Feb 14 | 283 files; 22 shell; 208 Markdown; tag "Core Reliability & State Management" | Add locks, safe checkpoints, state discipline | Useful safety concepts arrived, but as layers around shell workflow logic |
| Worker Emergence, `v5.0.0`, Feb 20 | 522 files; 59 shell; 349 Markdown | Make worker selection and spawning central | Added role and ceremony surface before real dispatch and completion were consistently proven |
| Classic peak, `v5.4`, Apr 4 | 1,638 files; 184 shell; 1,043 Markdown; shell-to-Go transition begins | Retain expressive wrappers while improving runtime reliability | Established the behavior baseline later releases repeatedly tried to restore |
| Go consolidation, `v1.0.0`, Apr 10 | Version reset; 243 Go; zero shell in tree count | Replace fragile shell state mutation with compiled runtime | Major safety improvement, but Go absorbed orchestration, prompts, ceremony, and platform behavior |
| Truth/recovery hardening, `v1.3` to `v1.7`, Apr 21-25 | "Visual Truth", empty-task fix `add10b56`, stale plan recovery `c5d59801`, "Planning Pipeline Recovery" | Stop no-op success and recover lifecycle flow | Shows both useful hardening and recurring regressions at the completion/state boundary |
| Queen/runtime expansion, `v1.14`, May 4 | 539 Go; "Queen Authority" | Centralize decisions and recovery | Queen became a label for routing, memory, decisions, ceremony, and state projection rather than one bounded component |
| Classic restoration, `v1.17`, May 14 | 54 TypeScript files; "TS host control plane" | Restore live workers, ceremony, Oracle, swarm without giving up Go safety | Created a second orchestration authority alongside direct Go lifecycle commands |
| Contract hardening, `v1.20`, May 15 | "Host Contract Hardening and Wrapper Reality Check" | Align finalizers, wrappers, and host | Repaired seams, but retained multiple public ways to perform the same lifecycle |
| Hybrid salvage, `v1.24`, May 25 | 1,974 files; 644 Go; 176 TypeScript; 151 YAML | Formalize Go/TS/assets boundary | Added a second `control-ts` scaffold with stub adapters while the older `.aether/ts-host` and direct Go orchestration remained |

The `v5.4..HEAD` diff is 3,119 files with roughly 409,000 insertions and 409,000 deletions. `cmd/` is the largest changed area. This is a near-total implementation replacement, not an incremental port.

## Why Churn Repeated

### 1. Behavior Had No Stable Owner

Classic stored behavior in shell and prompt assets. The Go migration made compiled commands authoritative but also embedded routing, prompt text, ceremony, and lifecycle loops. The restoration then added `.aether/ts-host`; `v1.24` added `control-ts`; Claude/OpenCode wrappers continued to add orchestration guidance. The repository's own `ARCHITECTURE_BOUNDARY.md` recognizes this and says Go should own safety, TypeScript orchestration, and assets behavior. Current implementation has not completed that migration.

Underlying problem: a manifest/finalizer boundary exists, but not every public path is required to cross it. The same command name can mean a direct Go loop, a wrapper-driven delegate path, or a TypeScript-hosted lifecycle.

### 2. Presentation And Completion Were Coupled

Commit themes repeatedly restore visible workers, read loops, ceremonies, spawn frames, timeout messaging, and honest completion. This churn occurs because worker narration, process completion, claim collection, and state advancement share loosely related artifacts. A build can emit a completion ceremony for a prepared packet even when worker results failed.

Underlying problem: there is no single durable run record whose terminal status is derived only from accepted worker results and evidence policy.

### 3. Recovery Was Added Per Command

The history repeatedly adds force flags, reconcile commands, stale-session checks, checkpoints, handoffs, and special finalizers. Many are individually useful. They do not form a universal transaction protocol. Planning can leave high-quality artifacts outside canonical state; retry clears them. Resume can reconstruct a goal from `HANDOFF.md` while losing a plan. Build can set `BUILT` after cancellation and depend on continue to discover missing evidence.

Underlying problem: recovery behavior is command-specific instead of replaying or rolling back one canonical transition journal.

### 4. Roles Expanded Faster Than Capabilities

The current 27 castes have distinct prompts, but most share the same model process, state context, and write permissions. Selection is keyword scoring plus hard-coded required castes. Many roles survived because their names, prompt assets, docs, and tests became distributed across three platform formats, not because the runtime proved a unique capability.

Underlying problem: persona specialization was confused with isolation, tools, or deterministic routing value.

### 5. Milestones Validated Scaffolds As Product Behavior

The `v1.24` history explicitly lands adapter stubs, Oracle stubs, in-memory event bridges, snapshot tests, and a demo CLI before live adapters. `control-ts/src/adapters/{claude,codex,opencode,mcp}.ts` returns `status: "completed"`, empty changes, and `available: true`. A milestone can therefore be internally "complete" while its advertised external behavior is simulated.

Underlying problem: acceptance was artifact- and test-count-based, not black-box journey-based.

## Regressions Lost And Restored

| Repeated area | Historical evidence | Root cause |
| --- | --- | --- |
| Worker spawning and visibility | Classic restoration, live ceremony branches, spawn/read-loop fixes | Dispatch truth and UI ceremony lacked one event source |
| Agent skill activation | Skill matching/injection milestones and platform-home fixes | Installed location, runtime matching, and platform discovery are separate systems |
| Timeouts/read loops | Watcher timeout branch, per-worker timeout work, host collection hardening | Process supervision differs between direct Go, host, and wrapper paths |
| Empty/no-op builds | `add10b56`; current claim tests; live false completion remains | File presence and worker prose are still not tied to acceptance criteria |
| Planning recovery | `v1.7`, `--refresh`, finalizer repairs | Artifact writes precede canonical commit, but no resumable planning transaction exists |
| Oracle loops | Repeated Oracle restoration and response-recovery branches | Standalone loop is durable; lifecycle consumers do not own a deterministic research-to-plan link |
| Pause/resume | session freshness, handoff, resume aliases, current reconcile work | `session.json`, `HANDOFF.md`, `CONTEXT.md`, state, and Git baseline duplicate continuity truth |
| Platform parity | Wrapper parity, Codex visual work, platform-home fixes | Platforms have different native capabilities but docs present one conceptual lifecycle |
| Version/release integrity | repeated release branches, version checks, publish runbook | npm publish is optional, assets and package versions are separate, no cross-channel atomic release gate |

## Abstractions Worth Preserving

- Manifest then finalizer for state-changing agent work.
- Atomic, locked deterministic state mutation.
- Context budgets and explicit source ledgers.
- Worker result schemas and claim normalization.
- Pheromones as short-lived steering input, after adding scope/conflict/provenance semantics.
- Queen as the user-facing project coordination projection.
- Oracle as a bounded research loop with durable sources, gaps, confidence, and stop conditions.
- Caste identity when it maps to a distinct job, permission, tool, or evidence contract.
- Worktree isolation when overlap is preflighted and merge evidence is verified.

## Abstractions Retained By Inertia

- Three platform copies of 27 agent definitions with no generated canonical source.
- Two TypeScript control systems plus direct Go orchestration.
- Hundreds of top-level utility commands exposing internal storage primitives.
- Multiple Queen state/audit/recovery files for one phase decision.
- Markdown reports treated sometimes as state, sometimes as display, and sometimes as prompt input.
- Separate castes whose only distinction is prompt wording.
- Ceremony snapshots used as a proxy for actual worker execution.

## Historical Conclusion

Aether's strongest transition was shell-to-Go for deterministic safety. Its most damaging transition was allowing that safety migration to absorb editable orchestration and then restoring orchestration through additional parallel authorities. The next era should not be another language migration. It should be a contract consolidation: one Go-owned state/evidence kernel, one adapter interface, one active orchestration owner, and derived presentation everywhere else.

