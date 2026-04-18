---
name: colony-interaction
description: Use when executing commands that involve major decisions, plan commits, build starts, verifications, or phase advances
type: colony
domains: [interaction, ux, workflow]
agent_roles: [builder, watcher, route_setter, architect]
priority: normal
version: "1.0"
---

# Colony Interaction

## Purpose

The colony workflow must not run autonomously past major decision points. Users want to feel in control of direction. This skill teaches you when and how to pause for user input.

## Literal Command Split

Not all literal `aether ...` commands should be treated the same way.

Safe read-only or maintenance commands such as `aether update`, `aether status`,
`aether version`, `aether history`, and `aether pheromones` still use the fast
path:

- Execute the literal command first.
- Do not convert it into a planning conversation or repo archaeology pass.
- Do not announce skill usage, intent interpretation, or a preflight summary before running the command.
- Do not inspect repo files first unless the command itself fails and you need evidence to explain that failure.
- Do not ask for extra confirmation for safe commands the user explicitly named.
- If the command itself fails or needs unavailable arguments, report the concrete CLI issue and the exact follow-up command.
- After the command returns, keep your own wrapper to one short sentence at most. The CLI output is the primary result.

Do not fast-path colony-shaping commands like `aether init`, `aether plan`,
`aether build`, `aether continue`, `aether run`, or `aether seal`. These still
need the touchpoints below.

For `aether init "<goal>"`, run a short foundation pass first: use
`aether init-research --goal "<goal>" --target .`, gather bounded repo context,
and present an approval summary before running the real init. If your platform
can spawn helpers, use Scout + Architect in read-only mode; otherwise do the
same analysis locally.

## Mandatory Touchpoints

Stop and present a multiple-choice question at each of these moments:

1. **Before committing a plan** -- After generating phases, show the plan summary and ask: "Approve this plan?", "Adjust scope?", "Regenerate with different focus?"
2. **Before starting a build wave** -- Before spawning workers, confirm: "Start building phase N?", "Review tasks first?", "Skip to a different phase?"
3. **After verification** -- When continue finishes verifying, present results and ask: "Advance to next phase?", "Re-run this phase?", "Pause and review?"
4. **Before advancing phases** -- Before moving the phase pointer forward, confirm the user is satisfied with the current state.

## How to Ask

Use AskUserQuestion with 2-4 options. Follow these rules strictly:

- **Plain English only** -- No jargon, no technical IDs, no internal state names.
- **Short options** -- Each option should be 3-8 words. The user wants to click, not read paragraphs.
- **Explain consequences** -- Each option must briefly state what happens if selected. Example: "Start building (spawns 3 workers)" not just "Start building".
- **Default is obvious** -- The most common/expected action should be the first option.
- **Never auto-proceed** -- If you cannot present options (e.g., non-interactive mode), log that you would have paused and continue with the default action. Note this in output.

## Question Format Template

```
[Decision Point Name]
What would you like to do?

1. [Most likely action] -- [what happens]
2. [Alternative action] -- [what happens]
3. [Conservative action] -- [what happens]
```

## Anti-Patterns to Avoid

- Running an entire build-verify-advance cycle without any user confirmation.
- Asking yes/no questions when richer options exist (prefer 3 options over 2).
- Asking technical questions the user cannot answer ("Which serialization format?").
- Presenting options that require code knowledge to evaluate.

## Integration with Autopilot

When running under `/ant:run` or `aether run` (autopilot mode), interaction touchpoints are relaxed -- autopilot handles the build-verify-advance loop. But smart-pause conditions still apply: test failures, security issues, quality gate failures, and replan suggestions all trigger a pause that requires user input.
