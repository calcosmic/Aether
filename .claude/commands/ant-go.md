<!-- Aether-managed: runtime spec at .aether/commands/go.yaml. Synced by aether update. -->
---
name: ant-go
description: "Do what you asked for — the program picks a quick job or a planned one and says which"
---

Use the Go `aether` CLI as the source of truth.

- Pass the owner's sentence straight through as `$ARGUMENTS`. Do not paraphrase it, shorten it, re-word it, or decide anything about it yourself.
- Run `AETHER_OUTPUT_MODE=visual aether go $ARGUMENTS`. The runtime measures how much of this project the sentence actually names and picks small or big from that — never from adjectives in the sentence — then says in plain English which route it picked and the recorded fact that picked it, before doing anything else.
- Relay the screen the runtime just drew, unchanged, before anything else in your reply. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If the screen says the quick route: there is nothing more to do. The job is already done.
- If the screen says the planning route and no project is recorded here yet: run `aether init "<the same sentence>"`, then run `aether go "<the same sentence>"` again. This is the one case that takes two calls, and running it again is always safe (it is idempotent) — the second call now finds a recorded project and proceeds down the planning route from there.
- If the screen says the planning route and a project is already recorded here: follow the planning route exactly the way `/ant-plan`'s own wrapper already does, using the same sentence as the job.
- This command never refuses the job. It either does it right there (the small route, through the one-helper quick path) or names the route it picked instead and carries it onward (the big route).
- Never decide small or big yourself, and never re-word the route sentence the runtime prints — that decision belongs to the runtime alone.
- Do not write colony state files, session files, or shadow/canary files by hand from this command spec.
- If docs and runtime disagree, runtime wins.
- If `$ARGUMENTS` is empty, show the runtime's usage message directly.

**Next steps:**
- `/ant-status` -- see the project's overall dashboard
- `/ant-continue` -- check what was built and move on
