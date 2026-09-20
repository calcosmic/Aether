# Phase 200: Iterative Planning - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-07
**Phase:** 200-iterative-planning
**Areas discussed:** Iteration presentation, Owner interruptions, Spec lifecycle, Research autonomy and stopping

---

## Iteration Presentation

### What should `/ant-plan` show when each Scout → Route-Setter iteration finishes?

| Option | Description | Selected |
|--------|-------------|----------|
| Delta card | Show score changes, fresh evidence, weakest gap, plan changes, and the continue/stop reason; keep citations and full diffs on demand. | ✓ |
| Full report | Print all findings, citations, and the complete plan diff after every iteration; exhaustive but much noisier. | |
| Final summary only | Show brief progress while running and defer iteration details to the final result; cleanest, but weakens the visible improvement loop. | |

**User's choice:** Delta card.

### How should completed iteration cards remain visible?

| Option | Description | Selected |
|--------|-------------|----------|
| Persistent timeline | Append every completed card and retain it with the plan; the Go finalizer already persists intermediate iterations, so visible history matches runtime truth. | ✓ |
| Collapsed timeline | Show every card while planning, then retain a compact summary with iteration details available separately after acceptance. | |
| Latest card only | Replace the prior card each pass and leave full history only in the internal revision ledger; least clutter, but obscures improvement. | |

**User's choice:** Persistent timeline.

### How should each card explain what Route-Setter changed?

| Option | Description | Selected |
|--------|-------------|----------|
| Semantic changes | List added, changed, or removed phases, tasks, dependencies, acceptance checks, and recovery paths; flag any owner-authority boundary separately. | ✓ |
| Narrative summary | Explain the revision in a short paragraph without enumerating structured effects; easier to read but harder to audit. | |
| Raw text diff | Show line-by-line plan edits; exact, but exposes formatting churn and hides meaningful changes. | |

**User's choice:** Semantic changes.

### How should the five confidence dimensions be presented?

| Option | Description | Selected |
|--------|-------------|----------|
| Scores plus evidence | Show whole-number before→after planning-readiness scores, with the fresh evidence and remaining gap that justify each change. | ✓ |
| Readiness bands | Show low, medium, or high readiness with a short explanation; less false precision, but hides the runtime's existing 0–100 movement. | |
| Scores only | Show the five numeric deltas without per-dimension explanations; compact, but confidence can look arbitrary. | |

**User's choice:** Scores plus evidence.

---

## Owner Interruptions

### When should known material owner decisions first be presented?

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence-first checkpoint | Let Scout complete the first grounded pass, then present all known material decisions together with evidence and recommendations. | ✓ |
| Before research | Ask every known owner decision before Scout starts; establishes authority early, but risks generic questions that evidence could answer. | |
| Just in time | Interrupt whenever each decision becomes relevant; highly contextual, but can fragment planning with repeated stops. | |

**User's choice:** Evidence-first checkpoint.
**Notes:** The owner emphasized that Phase 200's thesis is restoring the useful and distinctive February/April-era Aether behavior, not designing a new generic planning UX. Direct inspection of `3a5b81c2`, `v5.0.0`, and `v5.4` was added to the canonical evidence set.

### When should planning pause for a genuinely new later decision?

| Option | Description | Selected |
|--------|-------------|----------|
| At pass boundary | Finish the current Scout → Route-Setter pass, show evidence and provisional impact, then pause before an unauthorized revision is accepted. | ✓ |
| Immediately | Stop as soon as Scout identifies the issue; minimizes speculative work, but interrupts before Route-Setter explains plan impact. | |
| At finalization | Continue all research and collect material decisions at the end; fewer interruptions, but later passes may depend on an unapproved assumption. | |

**User's choice:** At pass boundary.

### What should a material-decision card provide?

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence and recommendation | State the decision, why it matters now, cited evidence, Queen's recommendation, consequences of each viable choice, and what resumes afterward. | ✓ |
| Neutral alternatives | Present evidence and tradeoffs without a Queen recommendation; avoids steering, but weakens the restored advisory role. | |
| Recommendation only | Present the Queen's proposal and impact with minimal alternatives; fastest, but gives the owner less basis to challenge it. | |

**User's choice:** Evidence and recommendation.

### When may planning reuse an earlier owner answer?

| Option | Description | Selected |
|--------|-------------|----------|
| Until impact changes | Reuse it while goal, meaning, and consequences stay equivalent; request revalidation if evidence changes behavior, scope, risk, or acceptance impact. | ✓ |
| Every revision | Reconfirm material answers whenever the plan revision changes; safest mechanically, but recreates a repetitive interview. | |
| Once per goal | Never ask the same decision again during a goal; least friction, but may preserve an answer after its consequences change. | |

**User's choice:** Until impact changes.

---

## Spec Lifecycle

### How should the owner reach `/ant-spec` during the normal journey?

| Option | Description | Selected |
|--------|-------------|----------|
| Automatic handoff | After `/ant-discuss` resolves material questions, the Queen presents a draft spec; `/ant-spec` remains the explicit open/edit/revise command. | ✓ |
| Explicit command | Require the owner to run `/ant-spec` after discussion and before planning; clear boundary, but another mandatory step to remember. | |
| Plan-triggered | Let `/ant-plan` detect a missing approved spec and launch the spec flow; convenient, but makes planning perform two distinct jobs. | |

**User's choice:** Automatic handoff.

### When should a new or materially revised spec become authoritative?

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit approval | Keep it draft until the owner approves the readable contract; only then may a new plan bind to that revision. | ✓ |
| Plan acceptance | Review and accept spec and plan together; fewer checkpoints, but planning elaborates an unapproved destination. | |
| Automatic approval | Approve once material discussion questions are resolved; fastest, but converts inferred intent into an owner contract without sign-off. | |

**User's choice:** Explicit approval.

### How should feature-only work relate to the whole colony-goal spec?

| Option | Description | Selected |
|--------|-------------|----------|
| Scoped revision | Keep one canonical spec lineage; feature work adds or revises identified requirements while preserving unaffected requirements and acceptance IDs. | ✓ |
| Linked child spec | Give each feature an approved child spec beneath the goal spec; clearer isolation, but introduces aggregation and conflict rules. | |
| Standalone specs | Treat goal and feature specs independently; simplest locally, but makes plan, evidence, and Seal authority ambiguous when they overlap. | |

**User's choice:** One canonical specification with feature-scoped revisions.
**Notes:** The owner initially selected Other and said they were unsure. After a plain-language example explaining one source of truth versus overlapping documents, they approved the scoped-revision recommendation.

### What should happen when an approved spec is materially revised?

| Option | Description | Selected |
|--------|-------------|----------|
| Impact-scoped invalidation | Create a linked immutable revision, preserve unchanged evidence, and mark only affected plan tasks and proofs stale until approval and replanning. | ✓ |
| Full plan reset | Invalidate the entire plan and all evidence after any material revision; simple and strict, but discards valid work. | |
| Warn without blocking | Record divergence while allowing planning, building, and Seal to continue; least disruption, but weakens spec authority. | |

**User's choice:** Impact-scoped invalidation.

---

## Research Autonomy and Stopping

### Who should choose planning depth when no quality flag is supplied?

| Option | Description | Selected |
|--------|-------------|----------|
| Queen proposes depth | Use goal risk, novelty, spec gaps, and evidence to recommend a preset; guided users may override. | |
| Owner selects preset | Always ask the owner to choose Fast, Balanced, Deep, or Exhaustive; transparent, but exposes the control each run. | ✓ |
| Fixed Deep default | Silently use Deep unless flags override it; predictable, but ignores trivial or unusually risky goals. | |

**User's choice:** Owner selects preset.
**Notes:** This intentionally overrides the recommended Queen-proposal default. The selected preset then authorizes routine iterations without repeated approval.

### How independently should Scout research within the chosen preset?

| Option | Description | Selected |
|--------|-------------|----------|
| Autonomous within preset | Target the weakest evidenced gaps and explain each pass; pause only for a material owner decision or new external-risk boundary. | ✓ |
| Approve research plan | Ask once to approve proposed research topics before iterations begin; extra control, but duplicates the preset decision. | |
| Approve every pass | Require confirmation before each Scout iteration; maximum control, but recreates stop-start planning. | |

**User's choice:** Autonomous within preset.

### What should happen when confidence stalls or caps below target?

| Option | Description | Selected |
|--------|-------------|----------|
| Authority-aware closeout | Finalize the best honest draft for non-material gaps; pause with an evidence-backed card only when a gap needs owner authority. | ✓ |
| Always ask owner | Offer continue, accept, or guidance whenever the loop stops below target, matching February's interactive stall behavior. | |
| Always finalize | Accept the current draft with disclosed gaps, matching April's automatic stopping even with an unresolved material choice. | |

**User's choice:** Authority-aware closeout.

### How should the final plan become eligible for build or Autopilot?

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit plan acceptance | Present final plan, confidence history, gaps, and recommendation; require owner acceptance before build. | ✓ |
| Conditional auto-accept | Automatically accept plans that reach target with no material gaps; ask only below target or at authority boundaries. | |
| Spec approval covers plan | Treat an approved spec as permission to accept any conforming technical plan; least friction, but removes the final checkpoint. | |

**User's choice:** Explicit plan acceptance.

---

## The Agent's Discretion

- Internal Go types and helper/file organization.
- Whether the machine-verifiable SPEC index is embedded in Markdown or stored as a sibling artifact.
- Exact terminal spacing, semantic colors, and drill-down command/flag composition, provided the default and retained information contracts remain intact.
- Internal retention mechanics and evidence-link representation, provided history is immutable, attributable, and visible through the accepted plan.

## Deferred Ideas

- Queen-led execution and worker-turnaround performance remain Phase 201 work.
- Substantive live Swarm/Watch/Oracle behavior remains Phase 202 work.
- Causal pheromone delivery and TypeScript-host preflight/timeout behavior remain Phase 203 work.
- Learning governance remains Phase 204 work; Codex-native `$ant-*` lifecycle skills remain a later milestone.
