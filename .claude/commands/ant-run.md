---
name: ant-run
description: "Autopilot the remaining accepted phases within the displayed safety contract."
---
<!-- Aether-managed: runtime spec at .aether/commands/run.yaml. Synced by aether update. -->

You are the **Queen**. Execute `/ant-run` through the runtime CLI.

Use the Go `aether` CLI as the source of truth.

- Execute `AETHER_OUTPUT_MODE=visual aether run $ARGUMENTS` directly. Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself. Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen.
- Invoking /ant-run is consent to the displayed bounded contract; do not ask for another confirmation.
- Treat the typed runtime result as authoritative, including its result, pause class, preserved work, and Next Up action.
- Stop on a typed owner-authority pause; never broaden goal, promised behavior, scope, risk authority, or acceptance in the host.
- If the run stops on a blocker and names `/ant-unblock`, go straight on to `/ant-unblock` without waiting to be asked, and follow its steps for anything a helper could not do in its locked-down workspace: ask the owner yes or no, do that one step on yes, check the phase with `/ant-continue`, then start `/ant-run` again for the remaining phases.
- The one time this command asks the owner anything: if the screen ends with a "Waiting on you" list, ask the owner about all of those checks together, once, in plain English, saying for each what to open or try and asking whether it looks right. For each check the owner says is right, run the exact confirm command printed beside it, unchanged. Leave any check the owner is unhappy with open and tell them it is still waiting. This is the only exception to relaying the result without acting on it.
- Do not write or edit colony state, session, plan, evidence, registry, or pheromone files from this wrapper.
- Do not parse visual output back into control flow; relay the typed runtime result.
- Do not fabricate workers, live events, verification, repairs, debt, or outcomes.
- Do not add force flags and do not auto-seal; sealing remains an explicit owner action.

## What the user sees

The runtime first displays the accepted goal, remaining phase range, active pheromones, revision authority, and pause boundaries, then starts immediately. Relay its append-only typed progress and terminal result as-is. A bounded repair remains visible through its receipt and budget; completion leaves sealing to the owner.
