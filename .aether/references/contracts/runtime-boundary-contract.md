---
schema_version: "1.0"
id: runtime-boundary-contract
kind: contract
category: contracts
title: Runtime Boundary Contract
description: "Contract defining ownership boundaries between Go runtime, TypeScript orchestration host, editable assets, and Bash glue."
output_types: [boundary-review, architecture-review, integration-test]
agent_roles: [architect, builder, watcher, queen, chronicler]
task_types: [boundary, contract, architecture, hybrid, migration]
task_keywords: [boundary, contract, go, typescript, bash, assets, runtime, orchestration, hybrid, migration]
workflow_triggers: [plan, build, continue, seal]
priority: critical
version: "1.0"
source: "aether-native"
render:
  mode: full
  max_chars: 4200
---
# Runtime Boundary Contract

## Purpose

This contract defines ownership across four layers of the Aether runtime: the Go binary, the TypeScript orchestration host, editable assets (Markdown/YAML/TOML), and Bash glue scripts. Every runtime behavior has exactly one owner.

For beginners: Go is the engine, TypeScript is the driver, assets are the manual, and Bash is the jumper cables.

## Ownership: Go Runtime

| Responsibility | File | What Go Owns |
|---------------|------|-------------|
| State mutation (COLONY_STATE, session, pheromones) | cmd/codex_build_finalize.go, cmd/codex_continue_finalize.go | Atomic writes with file locking, provenance validation |
| Build manifest generation (plan-only) | cmd/codex_build.go | JSON manifest without state mutation |
| Build finalization + provenance | cmd/codex_build_finalize.go | Validates manifest, commits state |
| Plan finalization | cmd/codex_plan_finalize.go | Validates plan manifest |
| Continue finalization + gates | cmd/codex_continue_finalize.go | Verification gates, state advance |
| Visual rendering | cmd/codex_visuals.go | ANSI banners, caste identity, stage markers |
| Dispatch contract structures | cmd/codex_dispatch_contract.go | Worker dispatch metadata |
| Command guide metadata | cmd/command_guide.go | Registered command metadata |

## Ownership: TypeScript Host

| Responsibility | Classification | Rationale |
|---------------|---------------|-----------|
| Lifecycle orchestration (calls Go plan-only, dispatches workers, calls finalizers) | Restore in TS | Classic bin/ orchestration pattern |
| Platform worker dispatch (spawns Claude/OpenCode/Codex per Go manifest) | Restore in TS | spawn-logger.js behavior |
| Spawn tracking (records spawn-log/spawn-complete via Go CLI) | Restore in TS | Restores visible worker activity |
| Error handling (typed errors for boundary violations) | Restore in TS | errors.js behavior |
| Ceremony rendering (delegates to @aether/ceremony-narrator) | Keep separate | Rendering, not control plane |

## Ownership: Editable Assets

| Asset Type | Location | Who May Edit |
|-----------|----------|-------------|
| Command playbooks | .aether/docs/command-playbooks/*.md | Humans |
| Agent definitions | .claude/agents/, .opencode/agents/, .codex/agents/ | Humans |
| Command YAML sources | .aether/commands/*.yaml | Humans |
| Skills | .aether/skills/**/SKILL.md | Humans |
| Templates | .aether/templates/ | Humans |
| Contract documents | .aether/references/contracts/*.md | Humans |

## Ownership: Bash

Bash is limited to small glue scripts: smoke tests, setup checks, release helpers, developer scripts. Bash MUST NOT own state mutation, verification, orchestration, or any logic that Go or TS host already provides.

## Classic Behavior Classification

| Module (v5.4.0) | Purpose | Classification | Rationale |
|-----------------|---------|---------------|-----------|
| spawn-logger.js | Worker spawn tracking | Restore in TS | Orchestration concern |
| state-guard.js | State write protection | Keep in Go | Safety-critical |
| caste-colors.js | ANSI caste identity | Keep in Go | Visual rendering |
| event-types.js | Event bus definitions | Keep in Go | Runtime state |
| file-lock.js | Atomic file locking | Keep in Go | Safety-critical |
| state-sync.js | Cross-process sync | Obsolete | Go handles atomically |
| banner.js | Phase banners | Keep in Go | Visual rendering |
| colors.js | ANSI color maps | Keep in Go | Visual rendering |
| logger.js | Structured logging | Restore in TS | Orchestration concern |
| init.js | Colony initialization | Keep in Go | State mutation |
| interactive-setup.js | Interactive prompts | Obsolete | Replaced by discuss flow |
| nestmate-loader.js | Agent loading | Obsolete | Go handles dispatch |
| binary-downloader.js | Binary downloads | Keep in Go | Safety-critical |
| update-transaction.js | Atomic updates | Keep in Go | Safety-critical |
| version-gate.js | Version checks | Keep in Go | Safety-critical |
| errors.js | Error definitions | Restore in TS | Orchestration concern |

## Anti-Patterns

1. **No TS Direct State Writes** — The TS host MUST NOT write to .aether/data/ directly. All state mutation goes through Go finalizers. Direct writes bypass provenance validation and file locking.
2. **No Visual Output Parsing as Authority** — The TS host MUST NOT parse ANSI/visual output to extract state. Use `AETHER_OUTPUT_MODE=json` for all programmatic consumption. Visual output is for humans only.
3. **No Wrapper-Owned Recovery Menus** — Wrappers and the TS host MUST NOT create option menus or recovery paths that do not come from the Go runtime itself. Runtime authority prevents conflicting recovery paths.

Supporting anti-patterns: no worker invention (use manifest only), no verification/gate duplication, no runtime contradiction (runtime wins over docs), no boundary question ownership in wrapper state.

## Rules

1. Go owns all state mutation — no other process writes .aether/data/
2. TS host calls Go plan-only for manifests, Go finalizers for commits
3. TS host never invents workers — dispatches only from Go manifest
4. Editable assets are human-editable but follow their own contracts
5. Bash is glue only — no state, verification, or orchestration logic
6. If docs and runtime disagree, runtime wins

## Review Questions

- Does the change respect the Go/TS/Assets/Bash ownership boundary?
- Does any TS code write directly to .aether/data/?
- Does any code parse visual output instead of using JSON mode?
- Are workers dispatched only from the Go manifest?
- Does the change introduce recovery menus not from the runtime?

## Failure Signals

- TS code imports or writes files in .aether/data/
- Wrapper code scrapes ANSI output for state
- Workers appear that are not in the dispatch manifest
- Recovery menus appear that do not come from aether command output

## References

- .aether/docs/wrapper-runtime-ux-contract.md
- .aether/references/contracts/command-wrapper-contract.md
- .aether/references/contracts/state-file-contract.md
- .aether/docs/source-of-truth-map.md
- .aether/docs/hybrid-runtime-strategy-research.md
- cmd/codex_build.go, cmd/codex_build_finalize.go, cmd/codex_continue_finalize.go
- cmd/codex_visuals.go, cmd/codex_dispatch_contract.go
