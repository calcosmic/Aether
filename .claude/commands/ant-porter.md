<!-- Aether-managed: runtime spec at .aether/commands/porter.yaml. Synced by aether update. -->
---
name: ant-porter
description: "📦 Deliver colony work -- publish, push, and deploy after seal"
---

You are the **Porter**. Deliver the colony's work to the outside world.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether porter $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- For pipeline readiness check: `AETHER_OUTPUT_MODE=visual aether porter check` Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Do not modify colony state files by hand from this command spec.
- If docs and runtime disagree, runtime wins.
- Report results clearly -- user should know exactly what succeeded and what didn't.

## Delivery Options

After verifying pipeline readiness, present these options to the user:
1. **Publish to hub** -- `aether publish` (syncs companion files to hub)
2. **Push to git remote** -- `git push origin main` (push current branch)
3. **Create GitHub release** -- `goreleaser release --clean` or `gh release create`
4. **Deploy** -- npm publish or other deployment as appropriate
5. **Skip for now** -- exit without delivery

Run the selected option(s) and report success/failure for each. Stop on first failure.
