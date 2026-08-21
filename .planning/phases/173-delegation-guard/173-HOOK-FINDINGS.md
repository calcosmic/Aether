# 173-HOOK-FINDINGS — Does the PreToolUse hook fire inside a subagent?

Captured: 2026-08-13
Binary: `aether version` → `{"ok":true,"result":"1.0.53"}` (rebuilt and installed 2026-08-13 from working tree at commit following 985d4422, including the file-based capture fallback described under Method)

## Method

A real two-level nested dispatch was performed from a live Claude Code session
in this repository (permission mode `bypassPermissions`):

- **Level 1:** the session's coordinator used the Agent tool to dispatch a
  `general-purpose` subagent ("middle agent").
- **Level 2:** the middle agent used its own Agent tool to dispatch a further
  `general-purpose` subagent whose entire prompt was "Reply with the single
  word: leaf". The leaf replied `leaf`; the middle agent returned both results.

**Capture-switch deviation (recorded, not hidden):** the plan specified
delivering the capture destination via the `AETHER_HOOK_CAPTURE_FILE`
environment variable set at Claude Code launch. Three operator relaunch
attempts failed to deliver the variable into the Claude Code process
environment (verified via `ps eww` on the live process: no `AETHER_` variable
present). The recorder in `cmd/hook_cmds.go` therefore gained a second opt-in
switch: a sentinel file `~/.aether/hook-capture-path` whose first line names
the destination. The env var, when present, still wins. The fallback is
off-by-default, operator-created, and never changes the hook's allow/deny
answer — asserted by `TestHookPreToolUseCapturesViaSentinelFileWhenEnvUnset`
and the hardened `TestHookPreToolUseCapturesRawPayloadOnlyWhenCaptureFileIsSet`.
The capture chain was proven live before the experiment: a direct stdin pipe
into the installed binary with the env var explicitly unset produced a
captured line via the sentinel route.

The capture file lived outside the repository (session scratch directory) and
is not committed; its full contents are quoted verbatim below.

*Post-capture note:* the sentinel-file fallback was removed the same day,
after the phase code review (173-REVIEW.md CR-01) showed a home-directory
switch file is creatable by any worker's ordinary Write tool. The evidence
below was captured while the fallback existed and is unaffected; the
recorder is once again env-var-only, locked by
`TestHookCaptureHasNoFileBasedSwitch`.

## Raw captured payloads (verbatim, in capture order)

```json
{"session_id":"55e92197-a80f-40f5-95c7-c59f8aa931b6","transcript_path":"/Users/callumcowie/.claude/projects/-Users-callumcowie-repos-Aether/55e92197-a80f-40f5-95c7-c59f8aa931b6.jsonl","cwd":"/Users/callumcowie/repos/Aether","prompt_id":"f9d46c55-52d8-4d7e-bb66-0cffc0cc78d4","permission_mode":"bypassPermissions","effort":{"level":"xhigh"},"hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Nested dispatch experiment level 1","prompt":"You are the middle agent in a two-level dispatch experiment. Do exactly this, nothing more:\n\n1. Run `pwd` with your Bash tool and note your working directory.\n2. Use your Agent tool (subagent_type: \"general-purpose\") to dispatch ONE subagent whose entire prompt is exactly: \"Reply with the single word: leaf\". Wait for its reply.\n3. Return exactly two lines: your working directory, and the leaf agent's reply.","subagent_type":"general-purpose","run_in_background":false},"tool_use_id":"toolu_018MGHrN1TKQUwkXaKYRPr95"}
{"session_id":"55e92197-a80f-40f5-95c7-c59f8aa931b6","transcript_path":"/Users/callumcowie/.claude/projects/-Users-callumcowie-repos-Aether/55e92197-a80f-40f5-95c7-c59f8aa931b6.jsonl","cwd":"/Users/callumcowie/repos/Aether","prompt_id":"f9d46c55-52d8-4d7e-bb66-0cffc0cc78d4","permission_mode":"bypassPermissions","agent_id":"ae93ff782863d564f","agent_type":"general-purpose","effort":{"level":"xhigh"},"hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Dispatch leaf agent","prompt":"Reply with the single word: leaf","subagent_type":"general-purpose","run_in_background":false},"tool_use_id":"toolu_01Nt94dhMgsnKLBM341vYtrT"}
```

## The four questions, answered from the captured lines only

**1. How many payloads were captured with a dispatch tool_name?**
Two, both `tool_name: "Agent"`. Payload 1 is the coordinator's own dispatch of
the middle agent (its `tool_input.prompt` is the middle-agent instructions).
Payload 2 is the middle agent's dispatch of the leaf (its `tool_input.prompt`
is exactly "Reply with the single word: leaf"). The hook fired inside the
subagent context.

**2. Which identifying fields are present on the inner (subagent-originated) payload?**
All three hypothesised fields appear, plus more:
- `agent_id: "ae93ff782863d564f"` — present **only** on the inner payload. It
  exactly matches the agent id the platform returned to the coordinator for
  the middle agent, so `agent_id` identifies the **requester** (the subagent
  making the dispatch), not the target.
- `agent_type: "general-purpose"` — present only on the inner payload.
- `session_id: "55e92197-a80f-40f5-95c7-c59f8aa931b6"` — present on **both**
  payloads, identical values (the session, not the agent).
- Also on both: `transcript_path`, `cwd`, `prompt_id`, `permission_mode`,
  `effort`, `tool_use_id`.
- The coordinator's own payload carries **no** `agent_id`/`agent_type` — the
  absence of `agent_id` is itself a reliable "requester is the top-level
  conversation" signal.

**3. Does `agent_type` echo the `subagent_type` the Task tool was invoked with?**
`agent_type` describes the **requester**: the middle agent was itself
dispatched as `general-purpose` and its dispatch carries
`agent_type: "general-purpose"`. The target's type travels separately in
`tool_input.subagent_type`. Bounded caveat: requester and target types were
both `general-purpose` in this run, so this observation cannot by itself
exclude "echoes the target" — but `agent_id`'s exact match to the requester's
id makes the requester reading the consistent one.

**4. What is the exact tool_name string?**
`Agent` — on both payloads. Not `Task`. The `Agent|Task` matcher in
`.claude/settings.json` covers it either way; a matcher of `Task` alone would
never fire on this runtime.

VERDICT: HOOK_FIRES_IN_SUBAGENT

## Consequence for plan 07

Plan 07 implements the deny path against the observed field names, and the
guard covers both the coordinator's dispatches and helper-level dispatches.
Concretely: match on `tool_name == "Agent"` (accepting `"Task"` for forward
compatibility, since the matcher registration covers both); treat a payload
carrying a non-empty `agent_id` as a subagent-originated dispatch (depth ≥ 1
requester) and a payload without `agent_id` as the coordinator's own dispatch
(depth-0 requester). The current official hooks documentation was right and
GitHub issue anthropics/claude-code#34692 does not describe this runtime.

## Named residue

There is no mapping from Claude Code's `agent_id` (e.g.
`ae93ff782863d564f`) to Aether's spawn-tree `AgentName`. This plan did not
create one; it only recorded which identifying fields the platform supplies.
Any requester-depth resolution in plan 07 is therefore a heuristic over the
fields observed above — presence/absence of `agent_id` and the value of
`agent_type` — not an identity lookup into Aether's own spawn records.
