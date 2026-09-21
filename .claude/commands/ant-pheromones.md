<!-- Aether-managed: runtime spec at .aether/commands/pheromones.yaml. Synced by aether update. -->
---
name: ant-pheromones
description: "🎯 View and manage active pheromone signals"
---

Use the Go `aether` CLI as the source of truth.

- Use `AETHER_OUTPUT_MODE=visual aether pheromones` to inspect the active steering surface. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If `$ARGUMENTS` names one signal type, route to `AETHER_OUTPUT_MODE=visual aether pheromones --type <FOCUS|REDIRECT|FEEDBACK>`. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If `$ARGUMENTS` asks to clear stale signals, run `AETHER_OUTPUT_MODE=json aether signal-housekeeping` and report the result.
- If `$ARGUMENTS` asks to expire one signal, run `AETHER_OUTPUT_MODE=json aether pheromone-expire --id <signal_id>`.
- **Adjusting signals is a multiple-choice conversation, not a memory test.**
  After showing the display, if the user wants to change steering (or asked
  vaguely, e.g. "clean these up"), ask with the AskUserQuestion tool
  (multi-select), one option per active signal in plain English — "Retire:
  <signal text>" / "Strengthen: <signal text>" — plus "leave everything as
  is". Retire via `AETHER_OUTPUT_MODE=json aether pheromone-expire --id
  <signal_id>`; strengthen by re-running the matching write command
  (`aether focus/redirect/feedback "<same text>"`) — a duplicate write
  reinforces the existing signal to full strength instead of duplicating it.
- **Adopting recommendations:** when the display shows a "Suggested Steering"
  section, offer those proposals the same way (multi-select, plain-English
  consequence per option, plus "none of these"); approve with
  `AETHER_OUTPUT_MODE=json aether suggest-approve --approve <id>`, reject with
  `--dismiss <id>`. Nothing is written without an explicit pick.
- If the user asks for fresh recommendations, run
  `AETHER_OUTPUT_MODE=json aether suggest-analyze` and then offer the results
  as above. (It also runs automatically at the end of every build.)
- Point new steering writes to `aether focus "..."`, `aether redirect "..."`, and `aether feedback "..."`.
- Do not read or rewrite raw colony state files or pheromone files by hand from this wrapper.
- Do not use manual shell file surgery to manage signals.
- If docs and runtime disagree, runtime wins.
- If `$ARGUMENTS` is empty, show the runtime display directly.

**Next steps:**
- `/ant-focus` / `/ant-redirect` / `/ant-feedback` — write new steering
- `/ant-build <phase>` — signals are injected into worker briefs
