# Minimum Viable Aether

## Product Definition

The minimum viable core is the smallest system that can take an imperfect goal, preserve clarified intent, execute one bounded phase through a real model adapter, require current evidence, survive interruption, and resume without the user retelling the project.

For beginners: before Aether runs a large colony, it must prove that one planner, one builder, and one reviewer can pass a reliable baton without dropping the plan or claiming unfinished work is done.

## Core Capabilities

| Capability | Core? | Perfect means | Current disposition |
| --- | --- | --- | --- |
| Repository initialization | Yes | Idempotent scaffold, baseline fingerprint, matching installed release | Repair |
| Goal and decision persistence | Yes | Typed goal, decisions, unknowns, alternatives and sources survive sessions | Consolidate |
| Brownfield colonization | Yes | Evidence-cited structure, conventions, history, tests, risks relevant to goal | Repair |
| Iterative planning | Yes | Versioned plan graph can revise future work while preserving completed nodes | Core revision contract implemented; live-provider acceptance pending |
| Iterative research | Yes | Bounded research conclusions link to decisions and invalidate affected plan nodes | Oracle evidence can now drive an explicit hashed revision; typed decision/assumption impact remains |
| Phase execution | Yes | One adapter, explicit workers, bounded ownership, terminal process truth | Repair |
| Evidence-based verification | Yes | Criteria map to files/commands/tests/reviews; missing evidence blocks | Redesign policy on existing machinery |
| Failure recovery | Yes | Transition pauses; retry resumes/rolls back idempotently | Consolidate |
| Context capsule and resume | Yes | Compact projection is current, traceable, sufficient, and rebuildable | Repair |
| Queen project memory | Yes | Readable project coordination/knowledge view derived from typed records | Narrow and rebuild |
| User steering signals | Yes | Scoped, expiring, conflict-resolved, consumed decisions visible | Repair |
| Minimal castes | Yes | Five jobs with distinct contracts; plain-English labels alongside ant identity | Reduce |
| One canonical runtime state | Yes | Journal + evidence ledger + materialized snapshot | Add core replacement/migration |
| One install/update path | Yes | Atomic versioned release set with rollback | Repair |
| Honest platform adapters | Yes | Capabilities/limitations explicit; one required adapter release-qualified | Redesign boundary |
| Observable worker execution | Yes | Every visible worker maps to process/run/result; no narrated terminal state | Repair |
| Final seal/summary | Yes | Evidence manifest, unresolved risk, plan/decision history, reproducible version | Repair after core |

## Minimum Core Colony

| Ant identity | Plain engineering role | Unique responsibility | Permission requirement |
| --- | --- | --- | --- |
| Queen | Coordinator | Chooses next transition from typed state/policy and presents current truth | State API only; no arbitrary repo write |
| Scout | Investigator | Brownfield mapping, focused investigation, evidence gathering | Read-only repository; optional network by explicit research grant |
| Route-Setter | Planner | Creates/revises plan graph from goal, decisions, research, and repository evidence | Plan proposal only; no repo implementation write |
| Builder | Implementer | Owns a bounded set of files/tasks and returns claims/evidence | Scoped workspace write and command profile |
| Watcher | Verifier | Independently checks criteria and can veto advancement | Enforced read-only; separate test command capability |

Oracle remains a named advanced research loop because iterative research and confidence management are distinct. It is not part of every build wave. Colonizer becomes Scout operating four structured lenses. Test authoring is a Builder mode; test verification remains Watcher.

## Optional Extended Colony

These are skills, review policies, or plugins until a unique tool/permission boundary justifies a separate caste:

- Architect: design review skill for Scout/Route-Setter.
- Archaeologist: Git-history investigation skill for Scout.
- Tracker: incident/root-cause mode for Scout.
- Probe: test-authoring mode for Builder or coverage review for Watcher.
- Gatekeeper: security policy/reviewer plugin.
- Auditor: quality rubric for Watcher.
- Includer: accessibility rubric for Watcher.
- Measurer: performance adapter/tool capability.
- Chaos: isolated resilience-testing plugin.
- Weaver: refactoring skill for Builder.
- Keeper + Sage: one Knowledge Curator, disabled until trust pipeline passes.
- Medic + Fixer: deterministic Doctor plus Builder repair mode.
- Chronicler: documentation skill for Builder.
- Ambassador: integration skill with scoped credentials.
- Porter: post-seal delivery plugin requiring human approval.

## Canonical Plan Model

Every phase and task must record:

```text
goal_reference
rationale
dependencies
tasks
expected_artifacts
acceptance_criteria
verification_methods
completion_evidence
decision/research inputs
status and supersession
```

The current `Phase` and `Task` types contain names, descriptions, task goals, dependencies, hints, constraints, and success criteria. They do not encode rationale, expected artifact ownership, verification method, or accepted evidence. Those fields are required before adaptive routing or parallelism is core.

## Core Workflow

```text
start
  -> clarify typed decisions and unknowns
  -> colonize if brownfield
  -> research blocking unknowns
  -> propose editable plan
  -> accept plan revision
  -> execute one phase
  -> verify explicit criteria
  -> advance, repair, or revise
  -> persist context and project knowledge
  -> resume from any terminal/paused transition
  -> seal with evidence manifest
```

No transition after plan acceptance may be inferred from a worker's positive prose. Every transition has a committed journal event and evidence record.

## Progressive Disclosure

### Beginner Path

```bash
aether start "This is what I want to build"
aether status
aether run
aether resume
```

`start` performs setup/init and guides clarification, brownfield discovery, blocking research, and editable plan acceptance. `run` stops at a human decision, failed evidence policy, or project completion. It is not enabled until single-phase acceptance passes.

### Guided Path

```bash
aether init "goal"
aether discuss
aether colonize
aether research "question"
aether plan
aether build 1
aether verify
aether plan revise
aether resume
aether seal
```

`verify` is the plain-English public name for the current `continue` job. Keep `continue` as an alias through one deprecation cycle.

### Advanced Path

```text
aether workers explain
aether signals ...
aether knowledge ...
aether debug ...
aether adapters ...
aether recovery ...
aether extensions ...
```

The 356 current utility commands move under these namespaces or become internal APIs. Advanced users retain access without making beginners learn storage implementation details.

## What Leaves Core

- Parallel worktree execution until overlap/merge tests pass.
- Hive auto-promotion and injection.
- Auto-generated REDIRECT after repeated failure.
- Four-stage specialist gauntlet on every relevant phase.
- Autopilot across multiple phases until one phase is reliable.
- Custom castes and lifecycle hooks as a stable public API.
- Dashboard, marketplace, community packages, multi-user, and new platforms.
- Ceremony beyond truthful goal/phase/worker/evidence/blocker/next state.
- Automatic publish/push/deploy.

## Minimum Observability

Every status/watch surface must show:

```text
Goal
Accepted plan revision
Current phase and transition status
Active worker processes and plain-English purpose
Adapter and permission profile
Recent accepted evidence
Open blockers/unknowns
Active scoped signals and conflicts
Workspace fingerprint/change status
Next legal transition
```

Worker selection explanation must include rule/policy, matched evidence, budget, alternatives pruned, and whether the caste is required or optional.

## Knowledge Boundaries

| Knowledge type | Scope | Core behavior |
| --- | --- | --- |
| User preference | User/global or project override | Typed, editable, explicit precedence |
| Project fact | Repository revision/branch | Evidence citation and freshness |
| Architectural decision | Project/plan revision | Alternatives, rationale, supersession |
| Temporary observation | Run/phase | Expires unless validated |
| Proven convention | Project and evidence range | Multiple independent observations plus tests/review |
| General engineering knowledge | Shipped/versioned skill | Maintainer-reviewed, not learned automatically |
| Cross-project wisdom | Optional user store | Disabled from automatic influence until provenance/contradiction/decay pass |

## Definition Of Core Complete

Core is complete only when Journeys A, B, D, E, G, and H pass against released artifacts; Journey C proves read-only investigation; Journey F proves no leakage. The release acceptance suite must run with a deterministic fake adapter and at least one real platform adapter. Snapshot/prompt-string tests do not substitute for these journeys.
