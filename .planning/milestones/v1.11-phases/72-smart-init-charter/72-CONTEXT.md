# Phase 72: Smart Init Charter - Context

**Gathered:** 2026-04-28
**Status:** Ready for planning

<domain>
## Phase Boundary

Restore the colony charter ceremony: `/ant-init` scans the repo, generates a charter document with 7 sections (Intent, Vision, Governance, Goals, Tech Stack, Key Risks, Constraints), presents it for user approval, and persists the approved charter in COLONY_STATE.json. Includes a full Go-native terminal ceremony for Codex and direct CLI users, alongside the existing markdown wrapper ceremony for Claude Code and OpenCode.

Requirements: INIT-01, INIT-02.

**What this phase delivers:**
- Charter data persisted in COLONY_STATE.json (charter sub-object with 7 fields)
- Go-native terminal ceremony with numbered-list prompts (for Codex and direct CLI)
- Approval flow: proceed, revise goal (re-run research), cancel (clean exit, no state)
- Dual mode: wrapper ceremony (Claude/OpenCode) + Go ceremony (Codex/CLI)
- Pheromone approval, shelf backlog, and charter review in established order

**What this phase does NOT deliver:**
- Deeper codebase analysis (Phase 73 — tech stack details, directory patterns, more pheromone patterns)
- Suggest-analyze during builds (Phase 74)
- Changes to existing wrapper ceremony structure (they keep their markdown rendering)

</domain>

<decisions>
## Implementation Decisions

### Charter Persistence
- **D-01:** Charter data stored as a sub-object in COLONY_STATE.json with fields: `intent`, `vision`, `governance`, `goals`, `tech_stack`, `key_risks`, `constraints`. Wrappers render as markdown for display; runtime reads JSON for downstream reference.
- **D-02:** No separate charter.md file — charter is structured data in JSON, not a document. Wrappers format it for human consumption.

### Charter Document Format
- **D-03:** Charter has 7 sections: Intent, Vision, Governance, Goals (existing 4) + Tech Stack, Key Risks, Constraints (new 3). The new sections pull from data that `init-research` already produces (languages, governance gaps, complexity metrics).
- **D-04:** Phase boundary with Phase 73: Phase 72 uses data that `init-research` already provides. Phase 73 adds deeper analysis (detailed tech stack breakdown, directory structure patterns, more pheromone suggestion patterns).

### Approval Flow Behavior
- **D-05:** Revise goal: user provides a new goal string, init-research re-runs with the new goal, and a fresh charter is presented. Clean restart — no partial state from the first attempt.
- **D-06:** Reject (cancel): clean exit. No COLONY_STATE.json created, no pheromones written, no session.json, no artifacts. User can run `/ant-init` again whenever ready.
- **D-07:** Approval sequence preserved: charter review → pheromone suggestions (tick-to-approve) → shelf backlog → final 3-option approval (proceed / revise / cancel).

### Codex Ceremony Path
- **D-08:** Full Go-native ceremony for Codex and direct CLI users. The Go runtime handles scanning, charter display, and terminal-based approval prompts — no wrapper needed.
- **D-09:** Dual mode: wrapper ceremony (Claude/OpenCode) keeps its markdown rendering and AskUserQuestion flow. Go ceremony (Codex/CLI) uses numbered-list terminal prompts. Both produce the same COLONY_STATE.json output.
- **D-10:** Go-native prompts use numbered list + user types number to select. No new dependencies (zero-new-deps principle from PROJECT.md). Same pattern as existing Go runtime prompts.
- **D-11:** Go ceremony is triggered by `aether init` running without a wrapper (direct CLI/Codex). Wrappers continue to orchestrate their own ceremony and call `aether init` after approval.

### Claude's Discretion
- Exact terminal prompt wording for Go-native ceremony
- Charter section ordering in the rendered output
- How key_risks and constraints are generated from existing scan data (which heuristics map to which sections)
- Whether Go ceremony includes pheromone tick-to-approve or auto-approves all suggestions

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — INIT-01, INIT-02 define the charter ceremony requirements

### Roadmap
- `.planning/ROADMAP.md` — Phase 72 goal, success criteria, dependency on Phase 71

### Existing Init Code (authoritative)
- `cmd/init_cmd.go` — Colony creation flow, COLONY_STATE.json v3.0 schema
- `cmd/init_research.go` — Research command: project detection, governance detection, git history, prior colonies, pheromone suggestions (10 patterns), charter generation (`generateCharter()`, `generatePheromoneSuggestions()`, `detectGovernance()`)
- `cmd/init_research_test.go` — Existing tests for init-research

### Colony State Schema
- `pkg/colony/` — ColonyState struct definition, state constants, scope types

### Platform Wrappers
- `.claude/commands/ant/init.md` — Claude Code init wrapper (charter ceremony, pheromone approval, shelf backlog)
- `.opencode/commands/ant/init.md` — OpenCode init wrapper (identical to Claude Code's)
- `.codex/CODEX.md` — Codex commands and rules (runtime-native, no wrapper ceremony)

### Architecture
- `CLAUDE.md` — Platform policy, UX architecture, wrapper-runtime contract
- `.aether/docs/wrapper-runtime-ux-contract.md` — Full contract for how wrappers delegate to Go runtime

### Research
- `.planning/research/FEATURES.md` §A1 — Feature analysis: what exists, what's missing for Smart Init

### Prior Phase Context
- `.planning/phases/62-lifecycle-ceremony-seal-and-init/62-CONTEXT.md` — Phase 62 decisions about init ceremony architecture (D-03 through D-06)
- `.planning/phases/71-platform-hardening/71-CONTEXT.md` — Phase 71 decisions about CLI flag fixes

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/init_research.go` — Full implementation: `initResearchCmd` with `--goal` and `--target` flags, `generateCharter()` produces 4-field charter, `generatePheromoneSuggestions()` applies 10 deterministic patterns, `detectGovernance()` scans for 21+ config files, `analyzeGitHistory()` extracts commit/contributor/branch data, `detectPriorColonies()` checks chambers/
- `cmd/init_cmd.go` — `initCmd` creates COLONY_STATE.json v3.0, session.json, activity.log, recovery artifacts. Has sealed-colony detection and backup logic.
- `pkg/colony/` — ColonyState struct with existing fields (Goal, Scope, Plan, Memory, Signals, etc.)

### Established Patterns
- State mutations via `store.SaveJSON()` pattern throughout cmd/
- `outputWorkflow()` for JSON output + visual rendering
- Wrapper-runtime contract: wrappers call Go commands, parse JSON output, handle interaction
- Ceremony emission via `emitLifecycleCeremony()` for lifecycle events
- Zero-new-deps principle: Go stdlib + cobra + pkg/storage only

### Integration Points
- `aether init` needs to accept charter data (via flag or stdin) and store it in COLONY_STATE.json
- `aether init-research` needs to output the 3 new charter sections (tech_stack, key_risks, constraints)
- Go-native ceremony needs terminal prompt utility (or reuse existing patterns)
- Wrapper ceremony in init.md already calls init-research and presents charter — needs updated to handle 7 sections
- COLONY_STATE.json schema in `pkg/colony/` needs Charter field added

### Key Gaps (from FEATURES.md)
- Charter data is computed but never stored in COLONY_STATE.json
- Go `aether init` does not call init-research internally — ceremony only works via wrapper
- OpenCode and Codex have no charter ceremony interaction
- Charter only has 4 sections — needs 3 more (tech_stack, key_risks, constraints)

</code_context>

<specifics>
## Specific Ideas

- User wants Codex to have the full ceremony experience, not a degraded path. The Go-native ceremony should feel as complete as the wrapper ceremony.
- The 3 new charter sections (Tech Stack, Key Risks, Constraints) should use data init-research already produces — no new scanning needed for Phase 72.
- User confirmed the Phase 72/73 boundary: 72 uses existing data, 73 adds deeper research.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 72-smart-init-charter*
*Context gathered: 2026-04-28*
