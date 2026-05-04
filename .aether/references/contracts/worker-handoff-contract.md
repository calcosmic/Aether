---
schema_version: "1.0"
id: worker-handoff-contract
kind: contract
category: contracts
title: Worker Handoff Contract
description: "Required contents for passing useful context between Aether workers."
output_types: [handoff, worker-output, deliverable, handoff-example]
agent_roles: [builder, watcher, scout, architect, tracker, queen]
task_types: [handoff, build, verify, continue, implement]
task_keywords: [handoff, changed files, commands, verification, assumptions, next worker, relay, handoff-example, freshness, deliverable]
workflow_triggers: [build, continue]
priority: high
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4400
---
# Worker Handoff Contract

## Purpose

A handoff gives the next worker enough context to continue without rediscovering the same facts.

For beginners: this is the relay note. It says what happened, what was checked, and what still needs attention.

## Required Fields

Every worker handoff should include:

- `changed_files`: paths changed or intentionally left untouched.
- `commands_run`: commands executed and short outcomes.
- `verification_status`: pass, fail, partial, or not run.
- `known_failures`: failures that still exist.
- `open_decisions`: choices the next worker or Queen must make.
- `assumptions`: facts believed but not proven.
- `next_worker_instructions`: bounded next action.
- `do_not_repeat`: dead ends already tried.
- `freshness`: timestamp or statement that evidence followed the latest edit.

## Quality Bar

A handoff is useful when:

- It is specific enough to act on.
- It separates facts from guesses.
- It names stale or missing evidence.
- It warns about user-local or generated files.
- It does not hide failures behind vague language.

## Bad Handoff Examples

- "Everything is done."
- "Tests mostly pass."
- "Continue from here."
- "Some files were changed."

## Good Handoff Shape

```json
{
  "changed_files": ["cmd/references.go"],
  "commands_run": ["go test ./cmd -run Reference -count=1: pass"],
  "verification_status": "partial",
  "known_failures": ["full cmd suite blocked by unrelated ProfileContract compile error"],
  "open_decisions": [],
  "assumptions": ["hub install not run in this session"],
  "next_worker_instructions": "Run full suite after unrelated compile failure is resolved.",
  "do_not_repeat": ["Do not copy references into target repo .aether/references."],
  "freshness": "Evidence collected after latest edit."
}
```
