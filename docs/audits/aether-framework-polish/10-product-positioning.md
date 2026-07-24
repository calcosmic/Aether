# Product Positioning And Competitive Comparison

## Evidence Boundary

This comparison uses Aether's audited runtime and live experiments. Competitor capabilities are based on their official repositories/documentation, not equivalent hands-on journey testing. It therefore compares advertised/product architecture and observable friction, not controlled outcome quality.

Primary references:

- [GSD architecture](https://github.com/gsd-build/get-shit-done/blob/main/docs/ARCHITECTURE.md)
- [Superpowers repository](https://github.com/obra/superpowers) and its [v5.1 removals](https://github.com/obra/superpowers/releases)
- [BMAD Method repository](https://github.com/bmad-code-org/bmad-method) and [agent reference](https://docs.bmad-method.org/reference/agents/)
- [GitHub Spec Kit](https://github.com/github/spec-kit/blob/main/README.md)
- [OpenSpec](https://github.com/Fission-AI/OpenSpec)
- [Geoffrey Huntley's Ralph explanation](https://ghuntley.com/ralph/)
- [Claude Code subagent/tool restrictions](https://code.claude.com/docs/en/sub-agents) and [project memory](https://code.claude.com/docs/en/memory)
- [OpenAI Codex agent loop](https://openai.com/index/unrolling-the-codex-agent-loop/) and [sandbox/worktree product model](https://openai.com/index/introducing-the-codex-app/)

## What Aether Is

Aether should be a **repository-resident, model-independent development control system** that preserves verified intent, decisions, evidence, plan revisions, work state, and project knowledge across agent sessions, then uses a small ant colony to investigate, plan, implement, and verify the next bounded transition.

It is not primarily a prompt pack. Prompts and ant roles are its editable behavior layer. Its differentiator must be deterministic continuity and evidence, not the number of roles.

## What Aether Is Not

- Not a replacement coding model or IDE.
- Not an autonomous software factory whose worker prose is trusted.
- Not a general multi-agent research platform.
- Not a collection of hundreds of commands users must learn.
- Not a global knowledge base that automatically treats repeated prose as truth.
- Not platform-equivalent when native permissions/subagents differ.
- Not polished while public install and false completion remain broken.

## Primary User

An individual developer or small technical team using Claude Code, OpenCode, or Codex on a multi-session greenfield or brownfield project where:

- requirements evolve;
- research can invalidate assumptions;
- the repository is too large for repeated full rediscovery;
- work spans context resets/days/models;
- evidence and recovery matter more than a one-shot code patch;
- the user wants visible control without becoming a workflow administrator.

It is not initially for large multi-user organizations, nontechnical app generation, or users seeking the fastest possible one-file edit.

## Painful Problem

Coding agents can implement a bounded request well, but long-running projects lose intent across sessions, repeat discovery, drift from decisions, and confuse plausible completion with verified completion. Existing methodology packs improve agent discipline; native agents improve tool execution and parallelism. Aether's job is to preserve and enforce the verified project state between those interactions.

## Comparative Matrix

Ratings are qualitative: strong, medium, weak, or unproven. "Aether now" reflects this audit; "Aether target" reflects the minimum core, not a current claim.

| Dimension | Aether now | Aether target | GSD | Superpowers | BMAD | Spec Kit | OpenSpec | Ralph | Plain Claude/Codex |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Time to first useful result | Weak | Medium | Medium | Strong | Weak/medium | Medium | Strong | Strong | Strong |
| Context continuity | Medium but contradictory | Strong | Markdown/project artifacts | Session/plan skills | Structured artifacts/workflows | Feature artifacts | Current specs + change deltas | Code/files between fresh loops | Native memory/session, platform-specific |
| Planning quality | Sometimes strong, recovery weak | Strong/revisioned | Structured multi-agent plans | Deliberate brainstorm/design/plan | Deep, scale-adaptive workflows | Spec -> plan -> tasks | Proposal/design/tasks | Depends on prompt/files | Model-dependent |
| Brownfield suitability | Mechanical scan, weak history grounding | Strong evidence-cited | Advertised brownfield workflows | General repository workflow | Explicit brownfield/quick flow | More 0-to-1 oriented, evolving | Strong change/delta model | Repository is state | Strong local exploration, no methodology |
| Research integration | Real Oracle, disconnected | Typed decision loop | Research agents | Research via skills/agent | Analyst/research workflows | Research within planning | Proposal research by agent | Prompt-dependent | Native browsing/tools |
| Replanning | Fails after completed phase | Strong plan revisions | Artifact/workflow dependent | Rewrite plan conversationally | Workflow dependent | Evolving spec flow | Strong delta/archive concept | Repeated loop changes files | Conversational/native plan |
| Steering | Pheromones work as prompt input | Scoped decision primitive | Commands/context | Conversation/skills | Menus/workflows | Spec/constitution | Proposal edits | Prompt/progress file | Conversation/instructions |
| Verification | Strong structures, false completion remains | Criteria-bound evidence | Verification agents | TDD/review discipline | QA/Test Architect options | Analyze/checklists/tasks | Validate/apply/archive | Loop termination discipline | Agent tests/review, human judgment |
| Failure recovery | Many pieces, inconsistent | Journal/idempotent | File-based context rebuild | Agent/session behavior | Artifact/workflow recovery | Artifact rerun | Change folders preserve intent | Fresh loop naturally retries | Native sessions/worktrees |
| Memory quality | Project/Hive split and unsafe | Typed, scoped, revocable | Project docs/context | Project instructions/skills | Optional agent memory/artifacts | Constitution/specs | Specs/changes | Repository/progress | Platform memory/instructions |
| Model independence | Three adapters, behavior differs | Explicit contract | Many integrations | Many integrations | Many tools | Many integrations | Many assistants | Any CLI loop | No, each native product |
| Complexity | Very high | Moderate core + extensions | High roster/workflows | Low/moderate | High | Moderate | Low/moderate | Very low | Low framework overhead |
| Transparency/debugging | Low/medium | Strong event/evidence trail | Prompt/artifact inspection | Skill text | Workflow artifacts/menus | Markdown artifacts | Simple folders/deltas | Shell loop/files | Tool logs/session UI |
| Install friction | Broken public path | One atomic path | Package/install flow | Native plugin paths | One npx installer, interactive | `uv` CLI + init | npm/init | Script/prompt | Native product install |

## Where Aether Is Already Better

At the component level, not the whole product:

1. Its Go finalizer/manifests, stale claim checks, path normalization, state locks, context budgets, and checksum downloader are stronger than a pure prompt methodology.
2. Oracle persists iterative questions, sources, gaps, confidence and synthesis beyond one chat.
3. Signals offer a concise, expiring steering vocabulary with deduplication, even though conflict/outcome semantics are incomplete.
4. The repository contains explicit recovery, result collection, reconciliation and evidence concepts that simpler frameworks do not attempt.
5. It has the beginnings of one state model usable across model vendors.

## Where Aether Is Only More Ambitious

- Self-organizing castes: current routing is hard-coded/keyword scoring plus model prompts.
- Hive learning: current confidence does not establish truth or safe relevance.
- Cross-platform parity: three launch paths exist, but permissions and semantics differ.
- Genuine adaptive planning: research and assumptions do not transactionally revise completed-plan futures.
- Multi-agent verification: more reviewers do not prevent skipped checks and false completion.
- Parallel colony work: concurrency exists, but overlapping results can be lost.

## Where Aether Is Objectively Worse Today

1. A clean public npm installation fails.
2. The beginner surface is far larger and slower than a plugin/skill/spec workflow.
3. A tiny notes phase selected six serial workers and consumed more than six minutes before interruption.
4. State and completion can contradict files/process results.
5. Debugging requires understanding Go, two TypeScript systems, wrappers, global hub, repo state and platform sessions.
6. A plan cannot be revised after completed work without starting a new colony.
7. Native Claude/Codex can enforce tool/worktree boundaries that Aether currently bypasses or flattens.

## Ideas Aether Should Adopt

### From Superpowers

- Prefer a small number of general-purpose agents plus self-contained skill/prompt templates.
- Remove deprecated stubs and named agents that add no unique value. Superpowers' release notes explicitly document both kinds of removal.
- Keep the workflow automatic and conversational rather than requiring users to memorize a taxonomy.
- Test skill behavior, not only file presence.

### From Spec Kit

- Make the artifact chain legible: intent/spec -> plan -> tasks -> implementation.
- Add cross-artifact consistency analysis and requirement checklists.
- Keep active feature identity explicit rather than inferred from Git alone.

### From OpenSpec

- Separate current truth from proposed changes.
- Represent evolving behavior as deltas with explicit added/modified/removed requirements and scenarios.
- Archive accepted changes into the canonical specification.

### From BMAD

- Adapt workflow depth to task scale so a bug does not receive a project-sized ceremony.
- Present context-aware "what next" help and modular advanced packs.
- Keep specialized personas as guided collaboration where they add cognitive value, not as mandatory subprocesses.

### From Ralph

- Preserve the simple invariant that a fresh context reads durable state and the changing repository.
- Use a small explicit stop condition and let each loop make measurable progress.
- Avoid putting the entire orchestration theory in every worker prompt.

### From Native Claude/Codex

- Use native permissions, hooks, subagents, sandboxing, worktrees, cancellation and telemetry through adapters instead of reimplementing or bypassing them.
- Treat native platform capability as an advantage and disclose when an adapter cannot match it.

## Ideas That Would Weaken Aether

- Becoming only another spec folder/template system; it would lose deterministic continuity/recovery.
- Replacing ant identity with generic corporate role names; the mental model is useful when backed by real contracts.
- Making every platform identical at the cost of ignoring native permissions/subagents.
- Moving state mutation back into prompts/wrappers for faster iteration.
- Adopting Ralph's minimal loop without evidence gates; that would abandon Aether's strongest technical investment.
- Matching BMAD/GSD roster size or Spec Kit integration count as a product metric.

## Blunt Answers

### 1. Where is Aether already better?

In deterministic primitives: locked state helpers, manifests/finalizers, claim freshness, context budgeting, durable Oracle artifacts, steering storage, and checksum downloads.

### 2. Where is it only more ambitious?

Trusted learning, self-organization, adaptive routing, plan revision, parallel safety, and cross-platform parity.

### 3. Where is it objectively worse?

Install reliability, first-result latency, command/architecture complexity, truthful completion, permission isolation, and debug cost.

### 4. Which competing ideas should it adopt?

Superpowers' role/skill economy and removal discipline; OpenSpec's truth/change split; Spec Kit's traceable artifact chain; BMAD's scale adaptation; Ralph's small loop; native adapters' real sandbox/tool contracts.

### 5. Which ideas would weaken its identity?

Reducing Aether to prompts/spec templates, hiding state entirely in platform sessions, discarding Queen/signals/Oracle rather than making them truthful, or adopting a platform-specific runtime.

### 6. What does Aether solve that GSD does not?

Potentially: one portable, transactional project state and evidence history across models/sessions; explicit recovery; short-lived steering; scoped project learning; and a research-to-plan revision graph. Today only parts are implemented.

### 7. Which GSD problems did Aether recreate at greater complexity?

Prompt/asset duplication, large agent rosters, ceremony-heavy flows, behavior encoded in Markdown, context assembly uncertainty, and wrapper/runtime drift, now spread across compiled and TypeScript layers too.

### 8. What would make a GSD user switch?

A five-minute demo where they change models after an interrupted phase, Aether reconstructs exact state, research invalidates one assumption, the plan revises without losing completed work, a worker implements, verification blocks a no-op, and seal shows traceable evidence.

### 9. What would make them uninstall in ten minutes?

The current public npm failure; six workers for a tiny task; platform/auth fallback they did not select; hundreds of commands; long mythology before useful output; or one false "complete" result.

## Defensible Claims Now

- Aether includes a Go CLI with persistent colony state and many deterministic lifecycle utilities.
- It can launch real Claude, OpenCode and Codex processes under supported local configurations.
- It creates structured worker manifests/results and performs file/freshness checks.
- Oracle can persist iterative research artifacts.
- Signals are stored, deduplicated, expired and injected as model context.
- Context assembly uses bounded budgets and source prioritization.
- Release downloads can be checksum-verified.

Each needs limitations in public copy.

## Claims To Remove Until Proven

- Polished, dependable, production-ready framework.
- Reliable public one-command install/update.
- Same lifecycle/permissions across all platforms.
- Self-organizing or emergent worker colony.
- Safe parallel implementation.
- Evidence-based completion in all paths.
- Research automatically changes plans.
- Context always survives resets without loss.
- Hive safely improves unrelated projects.
- The system never repeats mistakes.
- Any numerical reliability or quality-improvement percentage not produced by the acceptance corpus.

## Minimum Public Demo

Use a small brownfield CLI with one seeded provider-selection bug and one misleading initial assumption.

1. Install exact release with one command and show matching versions.
2. `aether start` captures goal and cites relevant repository/history evidence.
3. Scout investigates read-only; Oracle/source fixture contradicts assumption.
4. Route-Setter revises future plan while preserving one completed setup node.
5. Builder writes scoped fix/test.
6. First verification catches a deliberate missing criterion and blocks.
7. Interrupt the repair, start a new model session, and `aether resume` without restating context.
8. Repair passes deterministic checks and read-only Watcher.
9. Show which signal changed routing and why.
10. Seal emits plan/decision/evidence/residual-risk manifest.

This proves Aether's differentiator. A hero animation or a 27-worker task list does not.

## One-Sentence Positioning

**Aether is a persistent development control system that helps coding agents carry verified intent, evidence, and next actions across sessions and models.**

