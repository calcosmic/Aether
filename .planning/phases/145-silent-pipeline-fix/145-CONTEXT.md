# Phase 145: Silent Pipeline Fix - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Make all data-persistence CLI calls produce honest errors instead of silently dropping output, and ensure all state writes go through `state-mutate` to prevent corruption. This phase removes error suppression (`2>/dev/null || true`) from data-persistence calls in playbooks, adds automatic session.json cleanup on `/ant-init`, enforces event array capping at the Go storage layer, and verifies pending-decisions session scoping across all readers.

Five requirements: PIPE-01 through PIPE-05.

</domain>

<decisions>
## Implementation Decisions

### Failure Policy
- **D-01:** Data-persistence calls (learning, midden, pheromone, memory, spawn tracking) that fail during builds continue with visible error warnings, collected and summarized at build end — non-blocking metadata failures do not halt progress
- **D-02:** State mutations (COLONY_STATE.json via state-mutate) that fail are hard failures — corrupted state is worse than no state, so the build stops

### Claude's Discretion
- User deferred failure policy to Claude's judgment — tiered approach chosen: non-blocking for metadata, hard-fail for state
- Scope of `2>/dev/null || true` removal limited to data-persistence calls per PIPE-01 — legitimate read-only fallbacks (e.g., grep in archaeology) stay unchanged
- Event cap value stays at 100 (current playbook value, reasonable for colony history)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` (PIPE-01 through PIPE-05) — Five requirements defining what honest pipeline behavior means

### Error Suppression Targets
- `.aether/docs/command-playbooks/build-prep.md` — Build preparation playbook
- `.aether/docs/command-playbooks/build-wave.md` — Worker wave dispatch playbook
- `.aether/docs/command-playbooks/build-verify.md` — Build verification playbook
- `.aether/docs/command-playbooks/build-complete.md` — Build completion playbook
- `.aether/docs/command-playbooks/continue-verify.md` — Continue verification playbook
- `.aether/docs/command-playbooks/continue-advance.md` — Continue advancement playbook
- `.aether/docs/command-playbooks/continue-finalize.md` — Continue finalization playbook

### State Mutation Safety
- `cmd/state_cmds.go` — Go runtime `state-mutate` implementation with atomic guards and jq targeting
- `pkg/storage/` — File locking and JSON persistence layer

### Session and Decisions
- `cmd/init_cmd.go` — Init command (needs session.json cleanup addition)
- `cmd/pending_decision.go` — Pending decisions with session scoping (already implemented)

### Event Cap
- `pkg/colony/colony.go` — ColonyState struct and event array management

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/state_cmds.go`: `state-mutate` subcommand — atomic mutation with guard clauses, jq targeting, verify-only and revert modes. Already the correct pattern for all state writes.
- `cmd/pending_decision.go`: `pendingDecisionMatchesScope()` — session-based scoping already implemented in Go runtime. May need verification in playbook readers.
- `pkg/storage/`: File locking and JSON save/load — the foundation for adding event cap enforcement.

### Established Patterns
- Error output uses `outputError()` / `outputErrorMessage()` / `outputWorkflow()` helpers in Go runtime
- Playbook instructions use bash one-liners like `aether learning-capture "..." 2>/dev/null || true` — these are the suppression patterns to remove
- Session ID stored in `.aether/data/session.json` — used for pending-decisions scoping, needs cleanup on init

### Integration Points
- All 7 build/continue playbooks are the primary targets for error suppression removal
- `cmd/init_cmd.go` is where session.json cleanup gets added (PIPE-04)
- `pkg/colony/colony.go` or `pkg/storage/` is where event cap enforcement goes (PIPE-05)
- Playbook readers that check pending decisions need scoping verification (PIPE-03)

</code_context>

<specifics>
## Specific Ideas

No specific requirements — implementation follows the five PIPE requirements directly.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 145-Silent Pipeline Fix*
*Context gathered: 2026-05-20*
