# Aether System Integrity Audit Findings

Audit date: 2026-04-21
Scope: v1.3 phases 17-24 and v1.4 phases 25-30

## Finding 1
- Category: parity
- Severity: info
- File path: `.aether/commands/*.yaml`, `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, `.claude/agents/ant/*`, `.aether/agents-claude/*`, `.codex/agents/*`, `.aether/agents-codex/*`, `.opencode/agents/*`
- What's wrong: No file-set gap found here. All 50 YAML command basenames have Claude and OpenCode wrappers. Claude agent files and `.aether/agents-claude/` are byte-identical across all 25 agents. Codex agent files and `.aether/agents-codex/` are byte-identical across all 25 agents. OpenCode agent filenames match the Claude set.
- What the fix would be: None.
- Status: reported only

## Finding 2
- Category: parity
- Severity: critical
- File path: `.aether/commands/*.yaml`
- What's wrong: 47 of 50 command YAML `description` fields still use legacy decorative emoji prefixes or no leading command emoji at all, so they no longer match the runtime `commandEmojiMap` in `cmd/codex_visuals.go` or the committed Claude/OpenCode wrapper frontmatter. Examples include `medic.yaml`, `plan.yaml`, `status.yaml`, `focus.yaml`, and `build.yaml`.
- What the fix would be: Normalize the YAML descriptions to the same single-emoji descriptions already committed in the wrapper frontmatter so the source definitions match runtime and wrapper truth.
- Status: fixed

## Finding 3
- Category: parity
- Severity: warning
- File path: `.aether/commands/*.yaml`
- What's wrong: 38 YAML `runtime.command` strings are not the exact command strings surfaced in the committed wrapper contracts. Clear examples: `build.yaml` uses `AETHER_OUTPUT_MODE=visual aether build $ARGUMENTS` while wrappers execute `AETHER_OUTPUT_MODE=visual aether build --synthetic $ARGUMENTS`; `focus.yaml`, `feedback.yaml`, and `redirect.yaml` still advertise visual-mode invocations while wrappers call the plain pheromone commands; `status.yaml`, `watch.yaml`, `resume.yaml`, and `pause-colony.yaml` still advertise `$ARGUMENTS` even though the committed wrappers do not.
- What the fix would be: Resolve whether YAML or wrapper behavior is the intended external contract for these commands, then regenerate or normalize one side. I did not bulk-fix this without that decision.
- Status: reported only

## Finding 4
- Category: medic
- Severity: warning
- File path: `.aether/commands/medic.yaml`, `.claude/commands/ant/medic.md`, `.opencode/commands/ant/medic.md`
- What's wrong: The runtime exposes `aether medic --trace <export.json>` for trace diagnostics, but the command YAML and both wrapper docs omit that flag entirely. `medic.yaml` also still carries the old hospital-style emoji in its description while runtime and wrappers use `🩹`.
- What the fix would be: Add the `trace` flag to the YAML and both wrapper docs, and align the YAML description with the runtime/wrapper `🩹` contract.
- Status: fixed

## Finding 5
- Category: ceremony
- Severity: warning
- File path: `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md`, `.claude/commands/ant/init.md`, `.claude/commands/ant/plan.md`, `.claude/commands/ant/seal.md`, `.opencode/commands/ant/build.md`, `.opencode/commands/ant/continue.md`, `.opencode/commands/ant/init.md`, `.opencode/commands/ant/plan.md`, `.opencode/commands/ant/seal.md`
- What's wrong: The state-changing wrappers use section headings, but they do not contain literal runtime-style stage markers in the `── ... ──` format. That misses the ceremony integrity requirement that Phase 28 promised.
- What the fix would be: Add runtime-aligned stage-marker lines to both wrapper surfaces and, ideally, to the command-source generation path so the wrappers and generator stay aligned.
- Status: reported only

## Finding 6
- Category: skills
- Severity: warning
- File path: `.aether/skills-codex/colony/medic/SKILL.md`
- What's wrong: The Codex skill mirror is missing the Medic colony skill even though the source skill exists at `.aether/skills/colony/medic/SKILL.md`. That leaves the Codex mirror at 28 skills instead of the expected 29.
- What the fix would be: Copy the source Medic skill into `.aether/skills-codex/colony/medic/SKILL.md`.
- Status: reported only

## Finding 7
- Category: parity
- Severity: warning
- File path: `AGENTS.md`, `.codex/CODEX.md`
- What's wrong: Codex-facing docs still describe 24 Codex agents and omit Medic from their agent inventories. `AGENTS.md` also still claims 28 skills / 10 colony skills, while the source skill tree contains 29 skills total (11 colony + 18 domain).
- What the fix would be: Update the counts and agent tables to 25 Codex agents, add Medic to the lists, and update the Codex-facing skill counts to 29 / 11+18.
- Status: reported only

## Finding 8
- Category: medic
- Severity: info
- File path: `cmd/medic_cmd.go`, `cmd/medic_scanner.go`, `cmd/medic_repair.go`, `cmd/medic_ceremony.go`, `cmd/medic_trace.go`, `cmd/medic_auto_spawn.go`, `cmd/colony_prime_context.go`
- What's wrong: No implementation gap found in the core medic milestone surface. The command, scanner, repair path, ceremony checks, trace diagnostics, auto-spawn check, last-scan persistence, and colony-prime medic health injection are all present in the Go runtime.
- What the fix would be: None.
- Status: reported only

## Finding 9
- Category: trace
- Severity: info
- File path: `pkg/trace/trace.go`, `cmd/trace_cmds.go`
- What's wrong: No implementation gap found in the core trace milestone surface. `TraceEntry`, `Tracer`, and log helpers exist in `pkg/trace/trace.go`; `trace-replay`, `trace-export`, `trace-summary`, `trace-inspect`, and `trace-rotate` all exist; `trace-inspect` does generate suggestions.
- What the fix would be: None.
- Status: reported only

## Finding 10
- Category: tests
- Severity: info
- File path: `go test ./... -count=1`, `cmd/medic_scanner_test.go`, `cmd/medic_repair_test.go`, `cmd/medic_ceremony_test.go`, `cmd/medic_trace_test.go`, `cmd/medic_auto_spawn_test.go`, `cmd/trace_cmds_test.go`
- What's wrong: No test failure found. `go test ./... -count=1` passed cleanly on 2026-04-21, and the requested medic/trace coverage files exist.
- What the fix would be: None.
- Status: reported only
