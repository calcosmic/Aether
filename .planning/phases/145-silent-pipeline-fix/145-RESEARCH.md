# Phase 145: Silent Pipeline Fix - Research

**Researched:** 2026-05-20
**Method:** Context-driven analysis (discuss-phase + codebase survey)

## Problem Statement

The shell-to-Go migration (v5.4.0) introduced 80+ error suppressions (`2>/dev/null || true`) across playbook instructions. When data-persistence CLI calls (learning capture, midden write, pheromone update, memory capture, spawn tracking) fail, the error is silently swallowed. The system appears to work but data is silently lost.

Additionally:
- Playbook instructions still write `COLONY_STATE.json` directly instead of using `state-mutate`
- `session.json` is not cleaned on `/ant-init`, causing stale decisions to leak between colonies
- Event array capping exists in playbook instructions but not in the Go storage layer
- Pending decisions session scoping exists in Go but needs verification across all readers

## Technical Findings

### 1. Error Suppression Patterns

The suppression pattern `2>/dev/null || true` appears in:
- `.aether/docs/command-playbooks/build-*.md` — learning capture, midden writes
- `.aether/docs/command-playbooks/continue-*.md` — event logging, pheromone updates
- `.claude/commands/ant/*.md` — wrapper-level state mutations
- Playbook wave scripts — spawn tracking, memory capture

**Root cause:** The original shell implementation used this pattern to prevent playbook failures from cascading. In the Go runtime, failures should be surfaced honestly.

### 2. state-mutate Status

The Go runtime already implements `state-mutate` with:
- Atomic file writes (write temp + rename)
- Targeted jq expressions (no full JSON reconstruction)
- Used by: `colony-prime`, `pheromone-write`, `event-bus-publish`

**Gap:** Playbook instructions and some wrapper markdown still write `COLONY_STATE.json` directly via shell `jq` or `node`, bypassing the safe path.

### 3. Session Cleanup

Current `/ant-init` flow:
1. Creates `.aether/data/COLONY_STATE.json`
2. Sets colony goal
3. Does NOT clear `.aether/data/session.json`

**Impact:** Stale `session.json` from a previous colony carries forward `pending_decisions`, `context_cleared`, `baseline_commit` — all scoped to the OLD colony. This causes decisions from prior work to leak into the new colony.

### 4. Event Capping

Current state: Playbook instructions mention capping at 100 events, but the Go `storage` layer does not enforce this. Any command that appends events can overflow the array.

**Fix location:** `pkg/storage/` or the event bus publish path in Go.

### 5. Pending Decisions Scoping

Go runtime has `pendingDecisionMatchesScope()` but needs verification that:
- `/ant-flag` reads only current session's decisions
- `/ant-plan` filters by session scope
- `/ant-council` doesn't surface stale decisions

## Implementation Approach

### Wave 1: Error Suppression Removal (PIPE-01)
- Audit all `.aether/docs/command-playbooks/*.md` for `2>/dev/null || true`
- Replace with honest error handling
- For non-critical calls: continue build but collect errors for end-of-build summary
- For state mutations: fail hard (corrupted state is worse than no state)

### Wave 2: state-mutate Enforcement (PIPE-02)
- Find all direct `COLONY_STATE.json` writes in playbooks and wrappers
- Replace with `state-mutate` CLI calls
- Verify `state-mutate` supports all mutation patterns needed

### Wave 3: Session Cleanup (PIPE-03)
- Add `session.json` cleanup to `/ant-init` Go command
- Ensure cleanup is atomic (don't leave partial state)

### Wave 4: Event Capping (PIPE-05)
- Add event array cap (100 entries) to Go storage layer
- FIFO eviction when cap exceeded
- Log eviction for debugging

### Wave 5: Pending Decisions Verification (PIPE-04)
- Verify all pending-decision readers filter by session scope
- Add test coverage for cross-session isolation

## Risks

1. **Breaking existing playbooks:** Removing `|| true` may cause builds to fail that previously "succeeded" with silent data loss. This is the intended behavior change.
2. **Wrapper/runtime mismatch:** If wrappers expect silent failures but runtime now surfaces them, wrapper error handling may need updates.
3. **State-mutate coverage gap:** Some playbook mutations may use jq patterns that `state-mutate` doesn't support yet.

## Validation Strategy

- Run `aether build` and verify learning/midden/pheromone calls fail visibly when the underlying command is broken
- Run `aether init` and verify `session.json` is cleared
- Verify event array stays capped at 100 under load
- Test cross-session decision isolation

---

*Phase: 145-silent-pipeline-fix*
*Research: context-driven + codebase survey*
