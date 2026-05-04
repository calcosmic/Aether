---
schema_version: "1.0"
id: worker-handoff-example
kind: example
category: examples
title: Worker Handoff Example
description: "Concrete example of a useful Aether worker handoff."
output_types: [handoff-example, worker-output-example]
agent_roles: [builder, watcher, queen, scout, tracker]
task_types: [handoff, example, deliverable]
task_keywords: [handoff, example, changed files, verification, next worker, relay, freshness, do not repeat]
workflow_triggers: [build, continue]
priority: normal
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 3600
---
# Worker Handoff Example

## Use When

Use this when a worker needs an example of the expected handoff shape, not just the abstract contract.

For beginners: this shows what a good relay note looks like after real work.

## Example

```json
{
  "changed_files": [
    "cmd/references.go",
    "cmd/references_test.go",
    ".aether/references/rubrics/verification-evidence-ladder.md"
  ],
  "commands_run": [
    "go test ./cmd -run Reference -count=1: pass",
    "go run ./cmd/aether reference-match --role watcher --task \"verify completion evidence\": matched verification-evidence-ladder"
  ],
  "verification_status": "partial",
  "known_failures": [
    "Full cmd suite still has unrelated build-output parsing failures."
  ],
  "open_decisions": [
    "Whether to add more example references in the next content pass."
  ],
  "assumptions": [
    "No global hub publish has been run yet."
  ],
  "next_worker_instructions": "Continue with distribution tests; do not copy references into target repo .aether/references.",
  "do_not_repeat": [
    "Do not restore REFERENCE.md directory layout.",
    "Do not treat ~/.aether/ as source truth while editing this repo."
  ],
  "freshness": "Commands were run after the latest reference loader edits."
}
```

## Notes

A handoff can be shorter than this when the work is small. It should still name changed files, verification, known failures, assumptions, and the next bounded action.
