<!-- Aether-managed: runtime spec at .aether/commands/ask.yaml. Synced by aether update. -->
---
name: ant-ask
description: "💭 Ask the colony anything — auto-primed with its full memory"
---

Use the Go `aether` CLI as the source of truth.

Ask the colony a question — "where are we?", "why did phase 3 block?", "what
changed since yesterday?" — and answer it from the colony's own memory, with
zero setup. This is the one ask command: it also answers questions about the
code itself, not only the colony's own memory.

1. If `$ARGUMENTS` is empty, show `Usage: /ant-ask "<question>"`.
2. Run `AETHER_OUTPUT_MODE=json aether colony-prime --question "$ARGUMENTS"`.
   The runtime assembles the colony's full briefing — state, plan, steering
   signals, learnings, decisions, blockers, worker handoffs, and recent
   activity — ranked by relevance to the question, read-only.
3. Answer the question conversationally FROM the briefing's `context` field.
   Plain English, translate every colony term, cite what the answer rests on
   ("the phase record shows…", "a blocker from the security review says…").
4. Do NOT spawn workers for a colony question — the briefing already holds
   the colony's memory; a spawn adds latency and no information.
   Do NOT invent beyond the briefing: when the colony has no record of
   something, say exactly that ("the colony doesn't have a record of that —
   try `/ant-history` or ask me to dig").
5. If the question is about the CODE rather than the colony (how a function
   works, where something is implemented), run
   `AETHER_OUTPUT_MODE=visual aether quick --question "$ARGUMENTS"` — a
   read-only scout at the repo, changes nothing — and report its answer
   directly. Do not merely point the user at `/ant-quick`; this command
   answers any question itself.
6. If the runtime reports no colony, say there is no colony here yet and
   point to `/ant-init "<goal>"`.
