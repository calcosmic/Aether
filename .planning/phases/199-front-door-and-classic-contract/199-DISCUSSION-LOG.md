# Phase 199: Front Door and Classic Contract - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-02
**Phase:** 199-front-door-and-classic-contract
**Areas discussed:** Entry commands, Autopilot boundaries, Orientation view, Lifecycle vocabulary

---

## Entry Commands

### What is the user-facing command model?

| Option | Description | Selected |
|--------|-------------|----------|
| One aether do front door | Center the public journey on a new raw CLI command. | |
| Expanded aether init front door | Make the runtime CLI command the main user vocabulary. | |
| Classic guided sequence | Restore the familiar staged Ant journey. | ✓ |

**User's choice:** Use the Classic guided sequence, with an important correction: Claude Code and OpenCode users invoke `/ant-*`; raw `aether` commands are internal engine plumbing. Codex eventually gets matching `$ant-*` skills rather than an Aether-named user vocabulary.

### What should the Queen recommend after an accepted plan?

| Option | Description | Selected |
|--------|-------------|----------|
| Guided by default | Preselect `/ant-build 1`. | |
| Ask every time | Present the two modes without choosing. | |
| Autopilot by default | Preselect `/ant-run`. | |

**User's choice:** Free-text synthesis — guided and Autopilot are equal options. Neither is preselected or labelled recommended.
**Notes:** “Guided by default” was explicitly corrected: guided is just an option.

### What happens when `/ant-run` is invoked before initialization or planning is complete?

| Option | Description | Selected |
|--------|-------------|----------|
| Route safely | Do no work and identify the exact missing prerequisite. | ✓ |
| Offer to prepare prerequisites | Ask whether Aether should prepare them. | |
| Automatically prepare prerequisites | Initialize or plan without another owner decision. | |

**User's choice:** Route safely to `/ant-init` or `/ant-plan` without starting work.

### How should a valid `/ant-run` engage?

| Option | Description | Selected |
|--------|-------------|----------|
| Show and confirm | Display the contract and ask again before starting. | |
| Show then start | Display the contract, then begin because invocation was consent. | ✓ |
| Start immediately | Begin without an operating summary. | |

**User's choice:** Show the goal, phase range, active pheromones, and pause conditions, then start automatically.

---

## Autopilot Boundaries

### When should `/ant-run` pause?

| Option | Description | Selected |
|--------|-------------|----------|
| Pause only at genuine boundaries | Continue through safe repairs and stop only when authority, safety, or truth requires it. | ✓ |
| Pause after every phase | Require routine owner continuation. | |
| Choose pause cadence when starting | Add another startup decision. | |

**User's choice:** Aim to finish all phases; collect non-blocking issues for the final report and pause only when continuing would be unsafe or dishonest.

### How should pheromones affect an active Autopilot run?

| Option | Description | Selected |
|--------|-------------|----------|
| Apply at the next safe boundary | Queue all signals until active work finishes. | |
| Interrupt active work | Stop active work whenever a signal arrives. | |
| Require a manual pause | Make the owner pause before steering. | |

**User's choice:** Free-text refinement — pheromones should intelligently change course without necessarily stopping the run. FOCUS/FEEDBACK reach active ants where supported; REDIRECT stops only conflicting work; unsupported live delivery is stated and deferred to the next safe boundary.
**Notes:** Acknowledgement and later effect evidence are required so Aether cannot pretend a signal influenced work.

### What happens when `/ant-swarm` is invoked during Autopilot?

| Option | Description | Selected |
|--------|-------------|----------|
| Localized Swarm intervention | Pause only the affected job and continue independent work. | ✓ |
| Pause the whole colony | Stop every active job. | |
| Queue Swarm for the phase boundary | Delay Swarm until the current phase ends. | |

**User's choice:** Checkpoint the affected job, run and verify Swarm, integrate its result, and resume that path while unrelated work continues.

### May Autopilot revise an outdated accepted plan?

| Option | Description | Selected |
|--------|-------------|----------|
| Bounded automatic replan | Revise mechanics while preserving the owner's product contract. | ✓ |
| Always ask before replanning | Require approval for every task or dependency change. | |
| Never revise during a run | Continue following an obsolete plan. | |

**User's choice:** Automatically revise tasks, dependencies, sequencing, and implementation details; pause before changing goal, promised behavior, scope, risk authority, or acceptance criteria.

---

## Orientation View

### What should `/ant-status` show by default?

| Option | Description | Selected |
|--------|-------------|----------|
| Layered Classic dashboard | Lead with a summary and progressively reveal detail. | |
| Everything useful immediately | Show the complete owner-useful colony picture by default. | ✓ |
| Compact summary | Keep the default short. | |

**User's choice:** Restore the full useful Classic dashboard as the default and provide `/ant-status --compact` for a short view.
**Notes:** The full view includes identity, goal, progress, workers and lineage, pheromones, research/Dreams, memory, findings, verification, time/cost, history, blockers, and next choices—not raw plumbing.

### How much should a major command closeout show?

| Option | Description | Selected |
|--------|-------------|----------|
| Focused rich closeout | Show relevant participants, results, evidence, state changes, and next choices. | ✓ |
| Full dashboard every time | Repeat all status information after every command. | |
| Short closeout | Show only a terse success/failure line. | |

**User's choice:** Focused rich closeouts without duplicating the complete status dashboard.

### How strong should Queen voice and ceremony be?

| Option | Description | Selected |
|--------|-------------|----------|
| Strong at lifecycle moments | Use ceremony for consequential transitions and lighter identity elsewhere. | ✓ |
| Strong everywhere | Give every command full ceremony. | |
| Mostly neutral | Minimize the colony voice. | |

**User's choice:** Strong at lifecycle moments, grounded in runtime truth.
**Notes:** The user specifically called for strong Queen voice at lifecycle moments and directed the historical study back to February-April sources.

### How should status and Watch relate?

| Option | Description | Selected |
|--------|-------------|----------|
| Separate status and watch | Keep a snapshot and a live cockpit as complementary views. | ✓ |
| Merge live behavior into status | Make status both snapshot and stream. | |
| Watch only during Autopilot | Hide Watch outside active runs. | |

**User's choice:** `/ant-status` remains the authoritative snapshot and `/ant-watch` remains the live cockpit; an idle Watch shows recent status and activity.

---

## Lifecycle Vocabulary

### How should intentional pause and return commands be consolidated?

| Option | Description | Selected |
|--------|-------------|----------|
| One canonical pause and resume pair | Expose only `/ant-pause` and `/ant-resume`. | ✓ |
| Keep both short and colony-suffixed commands | Document four equivalent commands. | |
| Retain the historical colony-suffixed names | Use only `pause-colony` and `resume-colony`. | |

**User's choice:** One visible `/ant-pause` and one visible `/ant-resume`; carry forward the useful old handoff behavior but remove duplicate names from help, docs, and installed files.
**Notes:** The user rejected the initial suggestion of visible aliases as confusing. Any temporary parser redirect is migration plumbing only.

### What should `/ant-resume` do after an unclean interruption with no pause handoff?

| Option | Description | Selected |
|--------|-------------|----------|
| Smart /ant-resume for clean and unclean returns | Handle both validated handoffs and evidence-based crash recovery. | ✓ |
| Add a separate /ant-recover command | Use a second public command for crashes. | |
| Resume only from clean pause handoffs | Fail unless `/ant-pause` ran first. | |

**User's choice:** One smart `/ant-resume`, with confirmed facts distinguished from reconstructed recovery state.

### How should an owner close a colony that cannot pass normal completion?

| Option | Description | Selected |
|--------|-------------|----------|
| Owner-only force seal with an honest exception record | Preserve an explicitly forced, reasoned, incomplete closure record. | ✓ |
| Keep /ant-abandon as the normal incomplete closure | Discard the incomplete colony through the normal workflow. | |
| Refuse closure until every gate passes | Require every gate to pass before any closure. | |

**User's choice:** `/ant-seal --force --reason "..."` is the owner-only escape hatch; it must never look like successful completion or be invoked autonomously. `/ant-abandon` leaves the ordinary public workflow.

### Should seal and entomb remain distinct?

| Option | Description | Selected |
|--------|-------------|----------|
| Keep distinct seal and entomb stages | Seal retains the crowned colony; entomb later archives, verifies, and clears it. | ✓ |
| Fold entomb behavior into seal | Seal and clear the active colony in one action. | |
| Move archiving to generic maintenance | Remove the biological lifecycle command. | |

**User's choice:** Keep both `/ant-seal` and `/ant-entomb` because they represent different state transitions.
**Notes:** The initial recommendation to merge them was held rather than locked. Inspection of the April source showed the deliberate safety boundary: seal explicitly performed no archiving, while entomb required a valid Crowned Anthill, verified the chamber, and only then reset active state. The user approved preserving that distinction.

---

## the agent's Discretion

- Internal schemas, file organization, exact rendering composition, and any bounded compatibility-redirect duration may be selected during research and planning, but cannot change the locked public behavior.

## Deferred Ideas

- A later milestone should add Codex-native `$ant-*` skills mirroring the stable Claude/OpenCode experience.
- Phase 202 owns substantive live Watch and Swarm event mechanics.
- Phase 203 owns real live pheromone delivery and decision/outcome influence evidence.
- Three keyword-matched pending todos were reviewed and left with their approved owners: specification output in Phase 200, worker turnaround in Phase 201, and platform preflight/timeout work in Phase 203.
