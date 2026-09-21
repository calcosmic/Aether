<!-- Aether-managed: runtime spec at .aether/commands/improve.yaml. Synced by aether update. -->
---
name: ant-improve
description: "Look at how the colony's own suggestions are doing, or try one by hand"
---

Use the Go `aether` CLI as the source of truth.

- Run `AETHER_OUTPUT_MODE=visual aether improve` with no arguments to see two honest, separate figures: how often the colony's own suggestions genuinely finished something useful, and how often you had to step in and stop something going wrong. These two numbers are never blended into one score. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If the user wants to propose a change -- a "candidate" here means a proposed setting or routing rule, never a change to the program's own code -- route to `aether shadow-declare --id <id> --scope <"project knowledge"|"routing"> --expected-benefit "<why it might help>" --harms "<what could go wrong>" --expires <RFC3339 timestamp> --rollback-plan "<how to undo it>"`.
- If the user wants to try a declared candidate beside the current behaviour and see the verdict, route to `aether shadow-compare --candidate-id <id>`.
- This same comparison and trial also run automatically, on their own, at the end of every check (`aether continue`) -- relay what already happened rather than re-running it by hand.
- Only two kinds of change may ever be tried this way -- what the colony remembers about your project ("project knowledge"), and how it routes a task to a helper ("routing"). Every other kind of change -- your preferences, a skill, a workflow, your actual source code, a security setting, a permission, a verification step, sending or contacting something, or deleting anything -- has no path into this automatic trial at all, no matter how confident the program is.
- Do not read or rewrite shadow/candidates.json or canary/runs.json by hand from this wrapper.
- Do not use manual shell file surgery to declare or compare a candidate.
- If docs and runtime disagree, runtime wins.
- If `$ARGUMENTS` is empty, show the runtime report directly.

**Next steps:**
- `/ant-status` -- see the colony's overall dashboard
- `/ant-continue` -- the check that runs this comparison and trial automatically
