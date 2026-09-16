# Current Codex host capabilities and estimate assumptions

Research date: 2026-09-16. This is a scope assessment, not an implementation or acceptance result.

## Observed in this session

Three native Codex research agents were started independently, and their progress messages returned to the parent. This establishes that this host can delegate research; it does not prove that Aether's installed workflow skills select the right helpers, pass accepted assignments, submit results, or recover an interrupted build.

## Current official documentation

- [Skills](https://learn.chatgpt.com/docs/build-skills): Codex CLI/IDE supports explicit dollar-prefixed skill invocation. Name and description live in SKILL.md. Skill discovery currently documents repository/user `.agents/skills` locations and symlink support; the installed Aether skills are also visible in this session from the legacy `.codex/skills/aether` location. A migration should test actual discovery across the supported Codex clients rather than assume one path is universal.
- [Subagents](https://learn.chatgpt.com/docs/agent-configuration/subagents): native parallel helpers, custom agent TOML files, follow-up instructions, waiting and closing threads are supported. Project or skill instructions can request delegation. Host limits and inherited permissions still apply. This page does not establish every nested-delegation behavior Aether might need.
- [Slash commands](https://learn.chatgpt.com/docs/reference/slash-commands): Codex has built-in slash commands and supports dollar-prefixed skill invocation. A blanket statement that Codex has no slash commands is stale; the task is to supply Aether's custom workflows through the requested `$ant-*` skills.

## Scope assumptions to use in synthesis

1. The owner's requested public names are `$ant-*`, matching the existing Claude Code/OpenCode action names. This is a requirement, not an optional alias recommendation.
2. The installed executable can remain `aether`; public chat skill names do not require renaming the engine, storage directory, module, or worker roles.
3. Functional parity means matching accepted intent, planning/approval, helper assignments and returned results, saved work, verification, recovery, costs/unknown values and completion rules. It does not require identical host UI chrome or identical AI-generated prose.
4. Estimate naming/installation, command coverage, workflow behavior and proof separately. They overlap; do not sum the researchers' independent full-scope totals.
5. Separate shared defects from Codex-specific gaps. Count shared repairs only if needed to satisfy the promised working behavior, and label them.
6. Use disposable projects for development and proof. Keep the already prepared owner walkthrough and its original before-inventory unchanged.
7. Report engineering-effort ranges with confidence and assumptions. AI speed, model usage limits and live-run failures make elapsed calendar promises unreliable.
