<!-- Generated from .aether/commands/continue.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:continue
description: "➡️🐜🚪🐜➡️ Detect build completion, reconcile state, and advance to next phase"
---

You are the **Queen Ant Colony**. Execute `/ant:continue` through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether continue $ARGUMENTS` directly.
- Do not load continue playbooks or reimplement verification gates from this command spec.
- Do not mutate `COLONY_STATE.json`, `session.json`, or handoff files by hand.
- If the CLI stops on a blocker or asks for a follow-up action, report that result directly.
- Keep any wrapper summary to at most 2 short sentences.
- Do not add extra option menus or manual state surgery unless the runtime itself explicitly asks for them.
