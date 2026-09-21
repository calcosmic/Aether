<!-- Aether-managed: runtime spec at .aether/commands/quick.yaml. Synced by aether update. -->
---
name: ant-quick
description: "⚡ Do one small job — one helper, the project's checks, no ceremony"
---

Use the Go `aether` CLI as the source of truth.

Usage: `/ant-quick "<small job>"` — does the job with one helper (a builder),
then runs the project's own checks over whatever it changed. No build
ceremony, no plan, no check-in.

- Execute `AETHER_OUTPUT_MODE=visual aether quick $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If `$ARGUMENTS` is empty, show `Usage: /ant-quick "<small job>"`.
- Questions (not jobs) go to `/ant-ask`, which answers anything — including a
  question about the code — from one place. `/ant-quick --question
  "<question>"` also works directly if you want the read-only route by hand.
- Do not generate worker names, write spawn logs, update session files, or
  mutate colony state by hand.
- Note whether the project's own checks passed, failed, or could not be run.
- Note: quick jobs run through the **Go subprocess path**, not the Agent
  tool — no spawning ceremony is shown. That is by design: quick means no
  ceremony.
