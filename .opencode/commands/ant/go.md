---
name: ant-go
description: "Do what you asked for — the program picks a quick job or a planned one and says which"
---
<!-- Aether-managed: runtime spec at .aether/commands/go.yaml. Synced by aether update. -->

Use the Go `aether` CLI as the source of truth.

- Pass the owner's sentence straight through as `$ARGUMENTS`. Do not paraphrase it, shorten it, re-word it, or decide anything about it yourself.
- Run `AETHER_OUTPUT_MODE=visual aether go $ARGUMENTS`. The runtime measures how much of this project the sentence actually names and picks small or big from that — never from adjectives in the sentence — then says in plain English which route it picked and the recorded fact that picked it, before doing anything else.
- Relay the screen the runtime just drew, unchanged, before anything else in your reply. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- If the screen says the quick route: there is nothing more to do. The job is already done.
- If the screen says the planning route and no project is recorded here yet: run `aether init "<the same sentence>"`, then run `aether go "<the same sentence>"` again. This is the one case that takes two calls, and running it again is always safe (it is idempotent) — the second call now finds a recorded project, derives and approves a specification from the same sentence, and proceeds down the planning route from there.
- If the screen says the planning route: it also says, in plain English, whether it wrote down what was asked for itself or is planning the owner's own already-approved specification — either way nothing further is needed to get planning started. Two things this door has already settled and must never be put back to the owner: the planning preset is `fast`, and the specification is already handled. Skip `/ant-plan`'s own "Approved Specification Preflight" and "Choose Planning Preset" sections, and carry out the rest of `/ant-plan`'s own flow by name starting at its dispatch stage — request `aether host plan --preset fast $ARGUMENTS` directly to get the real dispatch manifest, then follow `/ant-plan`'s own Scout Stage, First-Pass Owner Decision Boundary, Route-Setter Stage, Iteration Card and Timeline, Continue/Pause/Stop, Candidate Review, and Exact Candidate Acceptance sections exactly as written, all the way to a real accepted plan. Once accepted, carry out `/ant-build`'s own flow the same way to build it, then tell the owner the one command to check the result (`/ant-continue`). This hand-off must not stop anywhere along the way — to ask the owner to clarify intent, to draft or approve a specification by hand, or to hand any other choice of how to proceed back to the owner — the single door already settled all of that. `/ant-discuss`, `/ant-spec`, and a deeper planning preset (balanced, deep, or exhaustive) all still work exactly as they do today, any time the owner asks for one of them by name.
- This command never refuses the job. It either does it right there (the small route, through the one-helper quick path) or names the route it picked instead and carries it onward (the big route).
- Never decide small or big yourself, and never re-word the route sentence the runtime prints — that decision belongs to the runtime alone.
- Do not write colony state files, session files, or shadow/canary files by hand from this command spec.
- If docs and runtime disagree, runtime wins.
- If `$ARGUMENTS` is empty, show the runtime's usage message directly.

**Next steps:**
- `/ant-status` -- see the project's overall dashboard
- `/ant-continue` -- check what was built and move on
