<!-- Aether-managed: runtime spec at .aether/commands/quick.yaml. Synced by aether update. -->
---
name: ant-quick
description: "⚡ Do one small job — one helper, the project's checks, no ceremony"
---

Use the Go `aether` CLI as the source of truth.

Usage: `/ant-quick "<small job>"` — does the job with one helper (a builder),
then runs the project's own checks over whatever it changed. No build
ceremony, no plan, no check-in.

- Execute `AETHER_OUTPUT_MODE=visual aether quick $ARGUMENTS` directly.
- If `$ARGUMENTS` is empty, show `Usage: /ant-quick "<small job>"`.
- Questions (not jobs) go to `/ant-ask`, which answers anything — including a
  question about the code — from one place. `/ant-quick --question
  "<question>"` also works directly if you want the read-only route by hand.
- Do not generate worker names, write spawn logs, update session files, or
  mutate colony state by hand.
- Report the CLI result directly, including whether the project's own checks
  passed, failed, or could not be run.
- Note: quick jobs run through the **Go subprocess path**, not the Agent
  tool — no spawning ceremony is shown. That is by design: quick means no
  ceremony.
