---
phase: 01-infrastructure
plan: 03
type: execute
wave: 1
depends_on: []
files_modified:
  - bin/cli.js
autonomous: true

must_haves:
  truths:
    - "CLI help text clarifies /ant:init is a slash command"
    - "Users understand they need to type /ant:init in Claude Code, not run a CLI command"
    - "All references to /ant:init include context about where to run it"
  artifacts:
    - path: "bin/cli.js"
      provides: "Clarified help text for /ant:init"
      contains: "slash command|Claude Code|/ant:init"
  key_links:
    - from: "CLI error messages"
      to: "User understanding"
      via: "Clear indication /ant:init is a Claude Code slash command"
---

<objective>
Clarify in CLI help and error messages that /ant:init is a Claude Code slash command, not a shell command.

Purpose: Users are confused by messages like "Run /ant:init in this repo first" because they try to run it as a shell command. The /ant:init is a Claude Code slash command that must be typed in the Claude Code chat interface, not in a terminal.
Output: Updated help text and error messages that clearly distinguish between CLI commands and Claude Code slash commands.
</objective>

<execution_context>
@~/.claude/cosmic-dev-system/workflows/execute-plan.md
@~/.claude/cosmic-dev-system/templates/summary.md
</execution_context>

<context>
@/Users/callumcowie/repos/Aether/bin/cli.js

Current problematic messages:
- Line 614: "No repos registered. Run /ant:init in a repo to register it."
- Line 633: "No repos registered. Run /ant:init in a repo to register it."
- Line 710: "Run /ant:init in this repo first."

These messages appear in the context of CLI commands (ant list, ant update-all, ant update), so users naturally think /ant:init is also a CLI command. They try running `/ant:init` or `ant init` in their terminal, which fails.

The fix should clarify that /ant:init is a Claude Code slash command (typed in the Claude Code chat interface with the leading slash).
</context>

<tasks>

<task type="auto">
  <name>Clarify /ant:init as slash command in CLI messages</name>
  <files>bin/cli.js</files>
  <action>
    Update three locations in bin/cli.js where /ant:init is referenced:

    1. Line 614 (in 'list' command, when no repos registered):
       Change: "No repos registered. Run /ant:init in a repo to register it."
       To: "No repos registered. Run the Claude Code slash command /ant:init in a repo to register it."

    2. Line 633 (in 'update-all' command, when no repos registered):
       Change: "No repos registered. Run /ant:init in a repo to register it."
       To: "No repos registered. Run the Claude Code slash command /ant:init in a repo to register it."

    3. Line 710 (in 'update' command, when .aether/ directory missing):
       Change: "Run /ant:init in this repo first."
       To: "Run the Claude Code slash command /ant:init in this repo first."

    The key change is adding "Claude Code slash command" before "/ant:init" to make it clear this is not a shell/CLI command.

    Be careful not to change any other functionality - only update these specific string literals.
  </action>
  <verify>
    grep -n "Claude Code slash command" bin/cli.js | wc -l | grep -q "3" && echo "All 3 messages updated" || echo "Missing updates"
  </verify>
  <done>
    All three references to /ant:init in bin/cli.js now include "Claude Code slash command" clarification.
  </done>
</task>

</tasks>

<verification>
- [ ] Line 614 updated with "Claude Code slash command" prefix
- [ ] Line 633 updated with "Claude Code slash command" prefix
- [ ] Line 710 updated with "Claude Code slash command" prefix
- [ ] No other functionality changed
- [ ] Messages still make grammatical sense
</verification>

<success_criteria>
- Users understand /ant:init is a Claude Code slash command, not a CLI command
- Error messages clearly distinguish between ant CLI commands and Claude Code slash commands
- No confusion between terminal commands and chat interface commands
</success_criteria>

<output>
After completion, create `.planning/phases/01-infrastructure/01-infrastructure-03-SUMMARY.md`
</output>
