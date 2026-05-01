# Phase 65: Idea Shelving - Context

**Gathered:** 2026-04-27
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 65 builds a persistent "not now, but don't forget" system for colonies. When ideas come up during a colony that would cause scope creep, they get shelved instead of lost. At seal, promising but unimplemented ideas are captured. At init, the user reviews the backlog and can promote items into the new colony as todos. Shelved ideas survive entomb — archived to chambers, not lost.

**What this phase delivers:**
- A persistent shelf file (`.aether/data/shelf.json`) stores deferred ideas with metadata
- `/ant-seal` detects shelf candidates and prompts user with checkbox list to approve shelving
- `/ant-init` shows the full backlog as a numbered list; user promotes/defer per item; promoted ideas become specific todos
- Recurring REDIRECT pheromones (same content hash across 2+ phases) get reviewed alongside other candidates at seal
- Entomb copies `shelf.json` into the chamber directory as a standalone file

**What this phase does NOT deliver:**
- Smart filtering or relevance matching at init (user sees full backlog)
- Auto-promotion without user interaction
- Cross-colony shelf sharing (shelf is per-machine, like other colony data)
- Web UI or non-CLI shelf browsing

</domain>

<decisions>
## Implementation Decisions

### Auto-Shelving at Seal
- **D-01:** Both explicit user shelving AND automatic detection. User can run `/ant-shelve "idea"` mid-colony. Seal also auto-detects candidates from colony state.
- **D-02:** Seal auto-detects: expired FOCUS pheromones (never addressed), low-confidence instincts (0.5-0.8), unresolved flags, and user notes/dreams with explicit language ("TODO", "idea", "future", "consider").
- **D-03:** Seal presents ALL detected candidates in a single checkbox prompt (same tick-to-approve pattern as pheromones). User ticks which ones to shelf. Unticked items are discarded.
- **D-04:** Recurring REDIRECT pheromones (same content hash across 2+ phases) appear in the same seal prompt as "Permanent guidance candidates". User decides which become permanent.
- **D-05:** Auto-detected items get `auto_detected: true` in shelf.json. Explicitly shelved items get `auto_detected: false`.

### Init Surfacing
- **D-06:** Init shows the FULL backlog — no filtering by repo type, domain, or keyword. User sees everything shelved from all prior colonies.
- **D-07:** Init presents shelf items as a numbered list with per-item options: "Promote to this colony" / "Keep on shelf" / "Delete permanently".
- **D-08:** Promoted ideas become specific todos (tracked work items), not broad goals or pheromones. They are added to the colony's todo tracking for assignment to phases later.
- **D-09:** If the backlog is empty, init silently skips the shelf step — no prompt.

### Shelf Data Model
- **D-10:** Each shelf entry has: `text`, `source` (phase/user/colony), `created_at`, `category` (instinct/pheromone/user-note/redirect), `confidence` (for auto-detected), `tags` (keywords), `promoted_to` (colony/phase if promoted), `status` (shelved/promoted/dismissed), `auto_detected` (bool).
- **D-11:** Shelf entries are never deleted — `status: dismissed` marks them as removed but preserves history.
- **D-12:** `tags` are populated from keyword extraction for auto-detected items, or user-provided for explicit items.

### Entomb Preservation
- **D-13:** Entomb copies the active `shelf.json` into the chamber directory as a standalone file (`chamber-XXX/shelf.json`). This preserves the full data model.
- **D-14:** The chamber manifest gets a summary line: "Shelved ideas: N (M promoted, P dismissed)" for human readability.
- **D-15:** When a new colony is initialized after entomb, init does NOT read from chamber shelf.json — only from the active `~/.aether/data/shelf.json`. Chambers are archive-only.

### Claude's Discretion
- Exact wording of the seal shelf prompt
- Number of items shown per page if backlog is large
- Whether to show dismissed items in init (recommend: no)
- Exact todo format when promoted (standardize with existing todo system)
- Keyword extraction algorithm for auto-tagging

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Seal ceremony (integration point)
- `cmd/codex_workflow_cmds.go` -- sealCmd, buildSealSummary
- `cmd/flags.go` -- flag system with severity
- `cmd/pheromone_write.go` -- pheromone CRUD, expiry logic, content_hash
- `.claude/commands/ant/seal.md` -- seal wrapper (where shelf prompt UX goes)
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` -- Phase 62 decisions on seal ceremony

### Init ceremony (integration point)
- `cmd/init_cmd.go` -- init command, colony creation flow
- `cmd/init_research.go` -- init-research implementation
- `.claude/commands/ant/init.md` -- init wrapper (where shelf surfacing UX goes)
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` -- Phase 62 decisions on init ceremony

### Entomb ceremony (integration point)
- `cmd/entomb_cmd.go` -- entomb command, chamber creation
- `.claude/commands/ant/entomb.md` -- entomb wrapper
- `.planning/phases/63-lifecycle-ceremony-status-entomb-resume/63-CONTEXT.md` -- Phase 63 decisions on entomb ceremony

### Pheromone system
- `cmd/pheromone_write.go` -- pheromone CRUD, type filtering, dedup (content_hash)
- `cmd/pheromone_dedup_test.go` -- dedup tests

### Requirements
- `.planning/REQUIREMENTS.md` -- SHELF-01 through SHELF-05 requirements
- `.planning/ROADMAP.md` -- Phase 65 success criteria

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `pkg/storage/` -- File locking and JSON store already handles `.aether/data/` files
- Pheromone content_hash dedup logic -- reuse for detecting recurring REDIRECTs
- Instinct storage in COLONY_STATE.json -- reuse confidence values and source tracking
- Todo system in `.planning/todos/` -- promoted shelf items should integrate here

### Established Patterns
- Wrapper-runtime contract: Go runtime outputs structured data, platform wrappers handle interactive UX
- Tick-to-approve pattern used in init for pheromones -- reuse for seal shelf prompt
- Phase 62 seal logs suggestions but doesn't auto-execute (same pattern: detect → present → user approves)
- Phase 63 entomb logs near-miss wisdom to chamber manifest (parallel pattern for shelf archiving)

### Integration Points
- Seal wrapper needs new shelf-detection step after blocker check, before archive
- Init wrapper needs new shelf-surfacing step after charter, before colony creation
- Entomb Go runtime needs to copy shelf.json into chamber directory
- Colony-prime or init-research may need to read shelf.json for context injection

</code_context>

<specifics>
## Specific Ideas

User's framing: "shelving is literally just ideas and things that we need to do in later developments because it's not wise to do it in the current colony because the scope is too big. It's ideas to do, flags, that kind of thing."

This means the system should feel lightweight — not a heavy project management tool, just a "don't lose the good ideas" safety net.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 65-idea-shelving*
*Context gathered: 2026-04-27*
