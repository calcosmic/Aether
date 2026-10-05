# Phase 206 — Screens Reach the Owner: context

**Requirements:** UED-01 to UED-06. **Decision:** `.planning/decisions/2026-09-21-v1.29-use-it-every-day.md`.

**Already built (release 1.0.88, branch `release-1.0.88`):** UED-01 to UED-04, plus three fixes found
in real use the same day (research helper evidence, pause with folder shortcuts, planning stuck on a
superseded specification) and the no-project status flags fix. The approved plan for that work is
`~/.claude/plans/okay-i-want-you-composed-coral.md`.

**Still to build:** UED-05 (the direct route) and UED-06 (status line).
- Verified 2026-09-21 in a real `claude -p` run: a PostToolUse hook on Bash returning
  `{"systemMessage": <tool stdout>}` is delivered to the user as a `system/informational` message.
  The hook input carries `tool_response.stdout`. Every line arrived prefixed
  `PostToolUse:Bash says: `; how that looks in the real window is for the owner to judge. The
  research notes report a 10,000-character cap. Stop hooks are overridden after 8 consecutive blocks.
- Owner decisions: long-running commands show the finished card plus the what-next card, not the
  running commentary; enforcement stays as the backstop.
- Status line: Aether registers no `statusLine` today. Its text must come from the shared what-next
  decision in `cmd/next_action.go`.
