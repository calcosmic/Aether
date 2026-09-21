<!-- Aether-managed: runtime spec at .aether/commands/update.yaml. Synced by aether update. -->
---
name: ant-update
description: "🔄 Update Aether safely from the global hub (transactional)"
---

You are the **Queen Ant Colony**. Update this repo's Aether system through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether update $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not reimplement hub checks, dry-run previews, cache clears, or transactional sync from this command spec.
- Do not describe a no-op update as requiring a workflow follow-up unless the CLI itself does.
- Report the CLI update summary and any restart guidance directly.

## What this command does — and does not — update

`/ant-update` syncs companion files from the global hub (`~/.aether/`): settings, rules, platform docs, and the global `~/.claude` commands and agents. It **does not update the binary** — the `aether` runtime at `~/.local/bin/aether` is untouched unless you pass `--download-binary`.

Freshness is bounded by the last `aether publish` run in the Aether source repo. If nothing was published, there is nothing new to sync.

## Hub-ahead-of-binary warning

If the CLI reports the hub version is **newer than the binary version**, surface it prominently — do not bury it in the summary:

```
⚠️ VERSION SKEW: hub is at <hub-version> but the installed runtime binary is <binary-version>.
Wrappers and runtime may disagree. Fix from the Aether source repo with:
  aether publish
or download the released binary here with:
  /ant-update --force --download-binary
```

`aether publish` has no slash wrapper — it is run from the Aether source repo,
so it keeps its CLI form. `/ant-update` does have one, and takes the same flags,
so name it the way the reader types it.

This skew means command wrappers may describe behavior the installed runtime does not have — the worst failure mode for a wrapper/runtime split. Treat it as action-required, not informational.
