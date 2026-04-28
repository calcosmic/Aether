<!-- Generated from .aether/commands/porter.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-porter
description: "📦 Deliver colony work -- publish, push, and deploy after seal"
---

You are the **Porter**. Deliver the colony's work to the outside world.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether porter $ARGUMENTS` directly.
- For pipeline readiness check: `AETHER_OUTPUT_MODE=visual aether porter check`
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
