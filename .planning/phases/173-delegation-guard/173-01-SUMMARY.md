---
phase: 173-delegation-guard
plan: 01
subsystem: infra
tags: [hooks, claude-code, spawn-guard, evidence, go]

# Dependency graph
requires:
  - phase: 172-wiring-proof
    provides: the stop-rule discipline (bounded recorded claims over assumed ones) this plan's verdict format follows
provides:
  - a dated, evidence-backed verdict that Claude Code's PreToolUse hook DOES fire when a subagent calls the dispatch tool (VERDICT HOOK_FIRES_IN_SUBAGENT in 173-HOOK-FINDINGS.md)
  - the observed payload contract plan 07's deny path matches on — tool_name is "Agent" (not "Task"); subagent-originated dispatches carry agent_id/agent_type; coordinator dispatches carry neither
  - agent_id proven to identify the REQUESTER (exact match with the platform-returned agent id of the dispatching subagent)
  - an opt-in raw payload recorder with two operator-created switches — AETHER_HOOK_CAPTURE_FILE env var, or sentinel file ~/.aether/hook-capture-path — proven to never change the hook's answer
affects: [173-07, 173-10]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Verdict-by-execution: contested platform behavior settled by running a real two-level nested dispatch and committing the raw captured payloads, not by choosing between contradicting documents"
    - "Dual opt-in switch: hook-side capture readable from env var OR sentinel file, because the hook inherits Claude Code's launch environment which the operator cannot reliably inject into"

key-files:
  created:
    - .planning/phases/173-delegation-guard/173-HOOK-FINDINGS.md
  modified:
    - cmd/hook_cmds.go
    - cmd/hook_cmds_test.go
    - .claude/settings.json

key-decisions:
  - "Task 2's restart-with-env-var checkpoint was replaced after three failed operator relaunches: the recorder gained a file-based fallback switch (~/.aether/hook-capture-path) so capture works regardless of how the session was launched. Deviation recorded in 173-HOOK-FINDINGS.md § Method."
  - "Task 3 executed inline by the orchestrator (executors have no Agent tool; the experiment requires a real two-level dispatch), per the phase handoff protocol."
  - "Capture switch removed immediately after the experiment — recording stopped the moment the evidence was committed."

patterns-established:
  - "Absence of agent_id on a PreToolUse dispatch payload identifies a depth-0 (coordinator) requester; presence identifies a subagent requester"

# Metrics
duration: paused mid-plan 2026-08-13 morning, resumed and completed 2026-08-13
completed: 2026-08-13
---

# Phase 173 Plan 01: Hook payload evidence run

**One-liner:** A real two-level nested dispatch proved Claude Code's PreToolUse
hook fires inside subagents and carries agent_id/agent_type, so plan 07's deny
path builds on observed fields — tool_name "Agent" — not on documentation.

## What was built

- **Recorder (Task 1, commit 985d4422):** `claudeHookInput` gained
  `agent_id`/`agent_type`/`session_id` fields; `readClaudeHookInput` returns raw
  stdin bytes; `captureRawHookPayload` appends them to an operator-named file.
  `.claude/settings.json` gained the `Agent|Task` PreToolUse matcher.
- **Fallback switch (commit 3210de2a):** capture destination resolvable from
  `~/.aether/hook-capture-path` when the env var is absent, with test
  `TestHookPreToolUseCapturesViaSentinelFileWhenEnvUnset`; the off-by-default
  test hardened against real sentinel files leaking in via `$HOME`.
- **Evidence (Task 3, commit b2405b79):** `173-HOOK-FINDINGS.md` with both raw
  payloads verbatim, the four answers, VERDICT, consequence for plan 07, and
  the named residue (no agent_id→AgentName bridge; requester-depth resolution
  in plan 07 is a heuristic, not an identity lookup).

## Deviations

- **Checkpoint mechanism replaced:** Task 2 assumed the operator could relaunch
  Claude Code with `AETHER_HOOK_CAPTURE_FILE` set. Three attempts failed to
  deliver the variable into the process environment (`ps eww` confirmed). Fixed
  structurally: the hook binary is re-executed fresh on every event, so a
  rebuilt binary reading a sentinel file took effect mid-session with no
  restart. The env-var switch remains primary.
- **Task 3 run inline** by the phase orchestrator instead of a spawned
  executor, as required by the experiment's need for the real dispatch tool.

## Self-Check: PASSED

- All plan verify commands pass (hook tests, vet, build, settings matchers,
  VERDICT/sections greps, no capture artifact in the repo tree).
- Both commits present: 3210de2a (fallback switch), b2405b79 (findings).
