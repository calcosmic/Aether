<!-- Aether-managed: runtime spec at .aether/commands/pheromones.yaml. Synced by aether update. -->
---
name: ant-pheromones
description: "🎯 View and manage active pheromone signals"
---

Use the Go `aether` CLI as the source of truth.

- Use `AETHER_OUTPUT_MODE=visual aether pheromones` to inspect the active steering surface.
- If `$ARGUMENTS` names one signal type, route to `AETHER_OUTPUT_MODE=visual aether pheromones --type <FOCUS|REDIRECT|FEEDBACK>`.
- If `$ARGUMENTS` asks to clear stale signals, run `AETHER_OUTPUT_MODE=json aether signal-housekeeping` and summarize the runtime result in one short sentence.
- If `$ARGUMENTS` asks to expire one signal, run `AETHER_OUTPUT_MODE=json aether pheromone-expire --id <signal_id>`.
- Point new steering writes to `aether focus "..."`, `aether redirect "..."`, and `aether feedback "..."`.
- Do not read or rewrite raw colony state files or pheromone files by hand from this wrapper.
- Do not use manual shell file surgery to manage signals.
- If docs and runtime disagree, runtime wins.
- If `$ARGUMENTS` is empty, show the runtime display directly.

**Next steps:**
- `/ant-focus` / `/ant-redirect` / `/ant-feedback` — write new steering
- `/ant-build <phase>` — signals are injected into worker briefs
