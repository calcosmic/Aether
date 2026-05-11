# Phase 5 Lifecycle Integration And Parity

Updated: 2026-05-08

This note gives later Phase 5 workers one place to check the expected docs,
runtime, wrapper, and changelog shape for Lifecycle Integration And Parity.

For dummies: Orchestrator Mode may need the user to answer a short boundary
question before work starts. The runtime should say that in machine-readable
JSON, wrappers should send the user to `aether discuss`, and finalizers should
refuse stale manifests before they write anything.

## Current Verified Surface

- Colony mode is runtime state. `pkg/colony` defines `colony` and
  `orchestrator`, and `EffectiveColonyMode()` treats missing or invalid legacy
  values as `colony`.
- Phase 4 added Orchestrator-only boundary questions to plan-only manifests for
  `plan`, `build`, `continue`, and `seal`.
- Boundary question sources use:

```text
orchestrator:<workflow>:phase:<N>:<category>[:hard]
```

- The `:hard` marker stays at the end of the source string.
- Boundary questions reuse `PendingDecision` entries with
  `type: "clarification"`; resolved answers flow through the existing
  clarified-intent pipeline.
- Default Colony Mode and legacy no-mode colonies must not create boundary
  questions.
- Existing output fields are:
  `colony_mode`, `boundary_questions`, `boundary_question_count`,
  `boundary_questions_created`, and `boundary_questions_existing`.

## Phase 5 Target Contract

Add a runtime-owned guidance object to the top-level result and matching
manifest for `plan`, `build`, `continue`, and `seal`:

```json
{
  "orchestrator_boundary_guidance": {
    "active": true,
    "workflow": "build",
    "colony_mode": "orchestrator",
    "pending_count": 1,
    "next": "aether discuss",
    "after_discuss_next": "aether build 5",
    "summary": "Resolve Orchestrator boundary questions before spawning build workers.",
    "question_ids": ["pd_..."],
    "question_sources": ["orchestrator:build:phase:5:build-scope:hard"]
  }
}
```

Rules:

- `active` is true only when `colony_mode` is `orchestrator` and unresolved
  Orchestrator boundary clarifications still exist.
- `pending_count` must be computed from the current unresolved clarification
  pending decisions by ID/source, not from a stale
  `boundary_question_count` copied from an old manifest.
- `next` should be `aether discuss` when guidance is active.
- `after_discuss_next` should preserve the lifecycle command the user should
  retry after resolving the question, such as `aether plan`,
  `aether build <phase>`, `aether continue`, or `aether seal`.
- Wrappers and Codex skills must use this guidance object for routing. Existing
  `boundary_questions*` fields can remain for display and compatibility, but
  should not be the routing source of truth.

## Finalizer Validation

Every finalizer that consumes a host-spawned manifest must validate before any
state, report, session, spawn-tree, or artifact write:

- The manifest root matches the current workspace root.
- The manifest `colony_mode` matches the active state's
  `EffectiveColonyMode()`.
- The manifest came from the matching plan-only or agent-delegate workflow.
- The workflow-specific state still matches:
  - `plan-finalize`: active goal, granularity, refresh rules, and no completed
    phases when refresh would replace history.
  - `build-finalize`: requested phase, current phase, selected tasks, phase
    status, and plan presence.
  - `continue-finalize`: current built phase, matching build manifest, review
    depth, and non-abandoned worker evidence.
  - `seal-finalize`: final completed phase, force flag, and final-review
    dispatch identity.

If validation fails, return a clear error and do not write partial output. This
is the safety bar for stale manifests, parallel sessions, and wrapper retries.

## Wrapper And Codex Parity

Update these surfaces together whenever Phase 5 routing changes:

- `cmd/command_guide.go`
- `.aether/commands/plan.yaml`
- `.aether/commands/build.yaml`
- `.aether/commands/continue.yaml`
- `.aether/commands/seal.yaml`
- `.claude/commands/ant/{plan,build,continue,seal}.md`
- `.opencode/commands/ant/{plan,build,continue,seal}.md`
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`
- Contract docs or runtime contract files for affected commands.

Required wrapper behavior:

- Request the JSON manifest and save the envelope outside `.aether/data/`.
- Parse `orchestrator_boundary_guidance` before rendering spawn ceremonies or
  spawning workers.
- If guidance is active, stop the lifecycle command, show the summary, route to
  `aether discuss`, and tell the user to rerun `after_discuss_next` after the
  answer is resolved.
- After a guided answer is resolved, request a fresh plan-only manifest. Do not
  reuse the stale pre-discuss manifest.
- Do not ask, answer, or store boundary questions in wrapper markdown, Codex
  skills, or chat-only state.

## Documentation Updates For Implementers

The runtime contract is now reflected in the active wrapper/docs surfaces:

- `.aether/docs/wrapper-runtime-ux-contract.md` with the
  `orchestrator_boundary_guidance` routing contract.
- `.aether/docs/source-of-truth-map.md` if any ownership language changes.
- `.aether/docs/README.md` if this note moves or graduates into a permanent
  lifecycle contract.
- `CHANGELOG.md` under `## [Unreleased]`.

Changelog entries should continue to preserve this shape:

```markdown
### Added
- Runtime-owned Orchestrator boundary guidance for plan/build/continue/seal
  manifests, including `next` and `after_discuss_next` routing.

### Changed
- Claude, OpenCode, and Codex lifecycle orchestration now stop before spawning
  workers when Orchestrator Mode has unresolved boundary clarifications.

### Fixed
- Lifecycle finalizers now reject stale or mismatched manifests before writing
  state or review artifacts.
```

Only claim wrapper parity after the command guide, YAML, Claude/OpenCode
wrappers, Codex skill, and tests are updated in the same change.

## Verification Checklist

Use focused tests first, then broaden if the implementation touched shared
contracts:

```bash
go test ./cmd -run 'Test(OrchestratorBoundary|ColonyModeManifest|CommandGuide|Wrapper|Finalize)' -count=1
go test ./pkg/colony -run TestColonyMode -count=1
go test ./cmd -run 'Test.*Parity|Test.*Source|Test.*Contract' -count=1
go test ./...
```

Also inspect the docs surface:

```bash
rg -n 'orchestrator_boundary_guidance|after_discuss_next|boundary_question_count|aether discuss' \
  cmd .aether/docs .aether/commands .claude/commands/ant .opencode/commands/ant \
  .aether/skills/colony/aether-colony-build-cycle/SKILL.md
```

## Do Not Repeat

- Do not change `PendingDecision` or `pending-decisions.json` schema.
- Do not create boundary questions outside Orchestrator Mode.
- Do not let finalizers create or save boundary questions.
- Do not let wrappers own question state.
- Do not parse visual output as state.
- Do not add repo-local `.codex/skills/aether` source files.
