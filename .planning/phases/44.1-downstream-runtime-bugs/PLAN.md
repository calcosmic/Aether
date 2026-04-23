# Phase 44.1: Downstream Runtime Bugs - Plan

**Created:** 2026-04-24
**Status:** Ready for execution

## Bug 1: False Skills (codex) Count

**Root cause:** `countEntriesInDir` (cmd/update_cmd.go:450) skips directories (`if entry.IsDir() { continue }`). The `skills-codex/` directory contains `colony/` and `domain/` subdirectories, each with SKILL.md files. The function counts 0-1 flat files instead of the 29 nested SKILL.md files.

**Fix:** Change `countEntriesInDir` to accept a `recursive` flag. For skills-codex, use recursive mode that counts SKILL.md files in all subdirectories.

**Files:** cmd/update_cmd.go, cmd/integrity_cmd.go (same issue)

### Task 1.1: Fix countEntriesInDir
- Add `countEntriesInDirRecursive` that walks subdirectories counting matching files
- Update the skills-codex check at line 423 and integrity_cmd.go:287 to use recursive count
- Keep the existing `countEntriesInDir` for flat-directory checks (commands, agents)

### Task 1.2: Test the fix
- Verify existing tests still pass
- Add test case for recursive counting

---

## Bug 2: Plan --refresh Guard Too Rigid

**Root cause:** `cmd/codex_plan.go:152` blocks refresh when `state.CurrentPhase > 0`. But when the colony is READY and no build has started, refresh should be allowed.

**Fix:** Check whether any phase has been built (in_progress/completed), not just whether CurrentPhase > 0. A colony in READY state with Phase 1 "ready" (not started) should allow refresh.

**Files:** cmd/codex_plan.go

### Task 2.1: Relax the refresh guard
- Replace `state.CurrentPhase > 0` with a check that no phase is in_progress or completed
- A phase is "built" if its Status is not Pending or Ready
- Keep the guard for genuinely active colonies

### Task 2.2: Clear stale planning artifacts on refresh
- When refresh is allowed, delete ROUTE-SETTER.md, phase-plan.json, and phase-research/* before starting
- Keep SCOUT.md if it was worker-written (preserve real scout output)
- This ensures the route-setter can write fresh artifacts

---

## Bug 3: Default Scout Timeout Too Low

**Root cause:** `cmd/codex_dispatch_contract.go:11` sets `planningScoutTimeout = 5 * time.Minute`. In larger repos, the scout often exceeds 5 minutes.

**Fix:** Raise default to 15 minutes for both scout and route-setter. Keep surveyor and review timeouts at 5m.

**Files:** cmd/codex_dispatch_contract.go

### Task 3.1: Raise planning timeouts
- Change `planningScoutTimeout` from 5m to 15m
- Change `planningRouteSetterTimeout` from 5m to 15m
- Keep `surveyorDispatchTimeout` and `continueReviewTimeout` at 5m

---

## Bug 4: Route-Setter Blocked by Fallback Artifacts

**Root cause:** When the scout times out, the runtime writes fallback ROUTE-SETTER.md and phase-plan.json. On a subsequent run (even with --refresh), `shouldPreserveWorkerArtifact` sees these fallback artifacts as "existing" and preserves them, blocking the real route-setter's output.

**Fix:** When `dispatchMode == "fallback"`, write a `.fallback-marker` in the planning directory. On the next plan run, if the marker exists, clear fallback artifacts before starting (keep real worker artifacts). Also: when the route-setter succeeds after a fallback scout, the route-setter artifacts must overwrite the fallback ones.

**Files:** cmd/codex_plan.go, cmd/codex_worker_artifacts.go

### Task 4.1: Add fallback artifact marking
- When dispatch falls back to local synthesis, write `.aether/data/planning/.fallback-marker`
- On plan start, if marker exists, delete it and clear fallback-only artifacts
- Preserve artifacts that were written by real workers (check modtime vs snapshot)

### Task 4.2: Allow route-setter to overwrite fallback
- In `shouldPreserveWorkerArtifact`, if the artifact was written during a fallback (marker exists), don't preserve it
- The route-setter should be able to replace fallback content with real content

---

## Execution Order

1. Bug 3 (timeout) — simplest, one-line change
2. Bug 1 (skills count) — well-isolated
3. Bug 2 (refresh guard) — moderate
4. Bug 4 (fallback overwrite) — most complex, builds on Bug 2

## Acceptance Test

After all fixes:
1. `go test ./... -race` passes
2. `go vet ./...` clean
3. Skills count check reports correctly for nested skill trees
4. `aether plan --refresh` works when colony is READY with no builds
5. Planning workers get 15m timeout by default
