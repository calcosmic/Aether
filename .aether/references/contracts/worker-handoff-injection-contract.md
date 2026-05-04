---
schema_version: "1.0"
id: worker-handoff-injection-contract
kind: contract
category: contracts
title: Worker Handoff Injection Contract
description: "How handoff data is stored, matched, and injected into subsequent worker prompts."
output_types: [handoff-review, injection-review, architecture-review]
agent_roles: [queen, architect, builder, watcher, scout, tracker]
task_types: [handoff, injection, prompt, worker, dispatch]
task_keywords: [handoff, injection, worker, prompt, context, previous, relay, colony-prime, budget, freshness, trim]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4000
---

# Worker Handoff Injection Contract

This contract defines how worker handoffs are created, stored, matched to
subsequent workers, and injected into prompts. Handoffs are the relay notes
passed between workers so each one starts with useful context from its
predecessors.

## For Beginners

When one worker finishes a task and the next one starts, the new worker would
normally spend its first turn rediscovering what the previous worker already
learned. Handoffs solve this: each worker leaves a short note summarizing what
it did, what it found, and what the next worker should know. The runtime stores
these notes and injects them into the next worker's prompt.

## Handoff Storage

### Location

Handoffs are stored in:
```
.aether/data/handoffs/worker-handoffs.json
```

This file is an array of handoff objects, ordered by creation time. The runtime
manages this file using the standard atomic write pattern (temp file + rename)
via the `pkg/storage` locking system.

### Handoff Fields

Each handoff object contains these fields:

| Field | Type | Description |
|-------|------|-------------|
| `changed_files` | `string[]` | Files this worker created or modified |
| `commands_run` | `string[]` | CLI commands the worker executed |
| `verification_status` | `string` | "pass", "fail", or "partial" |
| `known_failures` | `string[]` | Failures encountered and not yet resolved |
| `open_decisions` | `string[]` | Decisions that were deferred or need input |
| `assumptions` | `string[]` | Assumptions the worker made during execution |
| `next_worker_instructions` | `string[]` | Specific guidance for the next worker |
| `do_not_repeat` | `string[]` | Approaches that failed and should not be retried |
| `freshness` | `string` | ISO 8601 timestamp of when the handoff was created |

### Size and Retention

The runtime retains the most recent handoffs per phase. Older handoffs from
completed phases are pruned to prevent unbounded growth. The exact retention
window is determined by the runtime during continue and advance operations.

## Injection Mechanism

### When Injection Happens

Handoffs are injected during worker prompt assembly, specifically by the
`colony-prime` command when building the context capsule for a worker.

### Injection Format

Relevant handoffs are rendered into the worker prompt as a section titled:

```
## Previous Worker Handoffs
```

This section appears after the colony context (QUEEN.md, pheromones, instincts)
and before the task-specific instructions. It is separate from the skills
injection section.

### Matching Logic

Not all handoffs are injected into every worker. The runtime selects handoffs
based on:

1. **Phase relevance.** Only handoffs from the current phase (and optionally
   the immediately preceding phase) are considered.

2. **File overlap.** Handoffs that reference files the current worker is
   expected to modify are prioritized.

3. **Freshness.** Newer handoffs take priority over older ones. The
   `freshness` timestamp is used for ordering.

4. **Budget.** Handoff injection has its own character budget within the
   context capsule. If handoffs exceed the budget, older and less relevant
   ones are trimmed first.

## What Makes a Good Handoff

### Good Handoff Example

```json
{
  "changed_files": ["cmd/queen_decision.go", "cmd/queen_decision_test.go"],
  "commands_run": ["go test ./cmd/... -run Queen -v"],
  "verification_status": "pass",
  "known_failures": [],
  "open_decisions": ["Should we add a 4th tier for deprecation warnings?"],
  "assumptions": ["Circuit breaker resets per phase, not per wave"],
  "next_worker_instructions": [
    "The classification logic is in queen_classify.go, not queen_decision.go",
    "Tests cover all four tiers; add new tests alongside existing ones"
  ],
  "do_not_repeat": [
    "Do not move classification into queen_decision.go -- it caused import cycles"
  ],
  "freshness": "2026-05-04T14:30:00Z"
}
```

### Bad Handoff Example

```json
{
  "changed_files": ["lots of files"],
  "commands_run": [],
  "verification_status": "pass",
  "known_failures": [],
  "open_decisions": [],
  "assumptions": [],
  "next_worker_instructions": ["Just do the same thing"],
  "do_not_repeat": [],
  "freshness": "2026-05-04T14:30:00Z"
}
```

### Characteristics of a Good Handoff

- **Specific file paths** rather than vague descriptions
- **Actual commands run** so the next worker can reproduce or extend results
- **Honest verification status** -- "partial" is more useful than a false "pass"
- **Concrete assumptions** that the next worker can validate or challenge
- **Actionable instructions** -- tell the next worker something it would not
  know from reading the task description alone
- **Meaningful do_not_repeat entries** -- failed approaches that would waste
  the next worker's time

## Contract Obligations

**Builders MUST:**
- Include accurate `changed_files` lists
- Report `known_failures` honestly, even if they seem minor
- Provide specific `next_worker_instructions`
- Add `do_not_repeat` entries for any approach that consumed significant effort

**Watchers MUST:**
- Verify handoff quality as part of verification
- Flag handoffs with missing or vague fields
- Check that `verification_status` matches actual test results

**Queen MUST:**
- Include relevant handoffs in worker prompt assembly
- Respect the handoff budget within the context capsule
- Prune stale handoffs during phase advance

**All Agents MUST NOT:**
- Falsify `verification_status` to "pass" when tests fail
- Omit `do_not_repeat` entries to make output look cleaner
- Overwrite handoffs from other workers
