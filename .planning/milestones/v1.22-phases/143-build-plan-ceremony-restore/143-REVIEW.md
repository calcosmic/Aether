---
phase: 143-build-plan-ceremony-restore
reviewed: 2026-05-18T00:00:00Z
depth: standard
files_reviewed: 8
files_reviewed_list:
  - .aether/ts-host/src/playbook-loader.ts
  - .aether/ts-host/test/playbook-loader.test.ts
  - .aether/docs/command-playbooks/plan-prep.md
  - .aether/docs/command-playbooks/plan-dispatch.md
  - .aether/commands/build.yaml
  - .aether/commands/plan.yaml
  - .aether/ts-host/src/host.ts
  - .aether/ts-host/test/host.test.ts
findings:
  critical: 2
  warning: 5
  info: 3
  total: 10
status: issues_found
---

# Phase 143: Code Review Report

**Reviewed:** 2026-05-18
**Depth:** standard
**Files Reviewed:** 8
**Status:** issues_found

## Summary

Reviewed the ceremony restore implementation across playbook loading, plan playbooks, command YAML definitions, and the host orchestration entry point. Two critical bugs found: the build iteration loop can dispatch workers indefinitely because the spawn orchestrator is never updated after re-fetching the manifest (the re-creation comment on line 846 is dead code), and the plan command is missing the `spawnOrchestrator` option in its dispatch call, so spawn budget tracking is silently skipped for plan workers. Several warnings around playbook content accumulation on re-dispatch, test reliance on ambient hub state, and a broken code fence in the plan-dispatch playbook.

## Critical Issues

### CR-01: Build iteration loop has no spawn budget enforcement -- infinite re-dispatch possible

**File:** `.aether/ts-host/src/host.ts:834-846`
**Issue:** After re-fetching the manifest on iteration, the code updates `dispatches` but the comment on line 845-846 says "Re-create spawn orchestrator with updated budget for the new dispatches" without any actual re-creation code. The `spawnOrchestrator` created on line 739 retains its original `consumedBudget` from the first dispatch set. When the loop re-enters `dispatchBuildWave`, the orchestrator's budget is stale, so it cannot correctly enforce `max_workers`. If Go keeps returning dispatches on each re-fetch, the while-true loop on line 764 will keep dispatching workers indefinitely because neither the spawn budget check nor the orchestrator limits will catch the runaway condition.

Furthermore, the `ConfidenceLoop` may also fail to stop the loop if its internal `budgetRemaining` never reaches zero -- the loop only decrements by `lastWaveResult.workerCount`, but if the re-fetched dispatches are always a small number, `budgetRemaining` stays positive, and `confidenceLoop.evaluate` returns `shouldContinue: true`.

**Fix:**
```typescript
// After line 834 (dispatches = newDispatches;), add:
const newSpawnBudget = (reFetchManifest as Record<string, unknown>)?.queen_execution_policy != null
  ? ((reFetchManifest as Record<string, unknown>).queen_execution_policy as Record<string, unknown>)?.spawn_budget != null
    ? (((reFetchManifest as Record<string, unknown>).queen_execution_policy as Record<string, unknown>).spawn_budget as Record<string, unknown>)?.max_workers as number | undefined ?? 20
    : 20
  : 20;
spawnOrchestrator = createSpawnOrchestrator({
  goBinaryPath: bridge.goBinaryPath,
  cwd: bridge.cwd,
  totalBudget: newSpawnBudget,
  consumedBudget: newDispatches.length,
  currentDepth: spawnOrchestrator.currentDepth + 1,
});
```

Also add a hard upper bound on `iterationCount` as a safety guard:
```typescript
if (iterationCount > 20) {
  renderIterationComplete("safety_max_iterations");
  break;
}
```

### CR-02: Plan command missing spawnOrchestrator in DispatchOptions -- spawn budget untracked

**File:** `.aether/ts-host/src/host.ts:936-941`
**Issue:** The `runDispatchedPlanCommand` creates its `DispatchOptions` without a `spawnOrchestrator` property. The build command (line 615-620) correctly passes `spawnOrchestrator` to control spawn budgets. The plan command omits it entirely, meaning the `dispatchWorkers` function will operate without any spawn budget enforcement for planning workers. This is inconsistent and means plan dispatches have unbounded spawn capability.

**Fix:**
```typescript
// Add before dispatchOpts at line 936:
const spawnOrchestrator = createSpawnOrchestrator({
  goBinaryPath: bridge.goBinaryPath,
  cwd: bridge.cwd,
  totalBudget: dispatches.length, // Plan typically has fixed dispatches
  consumedBudget: 0,
  currentDepth: 1,
});

const dispatchOpts: DispatchOptions = {
  goBinaryPath: bridge.goBinaryPath,
  cwd: bridge.cwd,
  simulateWorkers: parsed.simulate,
  spawnOrchestrator,
};
```

## Warnings

### WR-01: Build iteration playbook context accumulates on re-dispatch

**File:** `.aether/ts-host/src/host.ts:838-842`
**Issue:** On subsequent build iterations, playbook context is re-appended to `d.task_brief` even though the re-fetched dispatches from Go may already carry the task_brief from the original manifest. Combined with the first-injection at lines 711-714, and the iteration feedback at lines 599-605, the `task_brief` string can grow unboundedly across iterations: each re-dispatch appends another copy of the full playbook context. If the iteration loop runs multiple times, the brief becomes redundant and wastes the token budget.

**Fix:** Before appending playbook context, check if it is already present:
```typescript
if (buildPlaybookContext && !((d.task_brief ?? d.task ?? "").includes("Relevant Playbooks"))) {
  d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + buildPlaybookContext;
}
```

### WR-02: Plan playbook `plan-dispatch.md` has broken code fence at Step 4

**File:** `.aether/docs/command-playbooks/plan-dispatch.md:58-62`
**Issue:** The code fence that starts on line 58 with triple backticks is never closed. The `aether spawn-complete` command on line 60 and the closing triple backticks on line 61 are rendered as part of the fenced bash block that starts on line 58, when they should be separate. This means the playbook consumer will execute `aether spawn-complete` inside the bash code fence along with `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete...`, or will see the closing backticks as part of the code. Either way, the ceremony + spawn-complete sequence is structurally malformed.

**Fix:**
```markdown
2. Render worker-complete ceremony:
```bash
AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <worker_file>
```
3. Call `aether spawn-complete --name "{name}" --status "{status}" --summary "{summary}"`
```

### WR-03: Playbook loader test `loads from hub` relies on ambient hub state

**File:** `.aether/ts-host/test/playbook-loader.test.ts:130-137`
**Issue:** The test `it("loads from hub when not found in repo")` asserts that `build-prep.md` exists in the global hub (`~/.aether/system/docs/command-playbooks/`). This test will fail in environments where the hub has not been populated (e.g., CI/CD, fresh machines, or after clearing the hub). The test has no setup step to ensure the file exists and no skip condition for missing hub state. This makes the test flaky outside the developer's local machine.

**Fix:** Create the hub file in a temp directory within the test, or mock `homedir()` to point at the temp directory so the test is self-contained:
```typescript
it("loads from hub when not found in repo", () => {
  // Set up a fake hub in tmpDir
  const hubDir = join(tmpDir, ".aether");
  const hubPlaybookDir = join(hubDir, "system", "docs", "command-playbooks");
  mkdirSync(hubPlaybookDir, { recursive: true });
  writeFileSync(join(hubPlaybookDir, "build-prep.md"), "# Hub Build Prep");
  // Mock homedir or use a version of resolvePlaybookCandidates that accepts hubDir
});
```

### WR-04: Continue command missing playbook injection (inconsistency with build and plan)

**File:** `.aether/ts-host/src/host.ts:973-1044`
**Issue:** The `runDispatchedContinueCommand` function does not load or inject playbook context into worker task briefs. Both `runDispatchedBuildCommand` (lines 709-715) and `runDispatchedPlanCommand` (lines 912-918) load playbooks for their respective workflows and inject them. The continue command skips this step entirely, even though `continue` has its own playbook files registered in the `WORKFLOW_PLAYBOOKS` map (`continue-verify.md`, `continue-gates.md`, etc.). This is an inconsistency that means continue workers operate without playbook context.

**Fix:** Add playbook injection to `runDispatchedContinueCommand`, following the same pattern:
```typescript
// After hive section injection (around line 996):
const continuePlaybooks = loadPlaybooksForWorkflow(bridge.cwd, "continue");
const continuePlaybookContext = renderPlaybookContext(continuePlaybooks);
if (continuePlaybookContext) {
  for (const d of dispatches) {
    d.task_brief = (d.task_brief ?? d.task ?? "") + "\n\n" + continuePlaybookContext;
  }
}
```

### WR-05: `--max-iterations` parsed as string but compared as number without NaN guard

**File:** `.aether/ts-host/src/host.ts:752-753`
**Issue:** `parsed.maxIterations` is typed as `string | undefined`. The code calls `parseInt(parsed.maxIterations, 10)` and passes it directly to `loopOpts.maxIterations`. If the user passes a non-numeric value like `--max-iterations abc`, `parseInt` returns `NaN`, which propagates into `ConfidenceLoop` and silently breaks loop termination logic (since no numeric comparison with `NaN` returns true).

**Fix:**
```typescript
if (parsed.maxIterations) {
  const parsed = parseInt(parsed.maxIterations, 10);
  if (!isNaN(parsed) && parsed > 0) {
    loopOpts.maxIterations = parsed;
  }
}
```

## Info

### IN-01: `plan-prep.md` displays phase count that may not exist for fresh colonies

**File:** `.aether/docs/command-playbooks/plan-prep.md:20`
**Issue:** The status display template shows `Phase {current_phase}/{total_phases}` but for a colony that has not yet been planned, `total_phases` may be 0 or undefined. This results in confusing output like `Phase 0/0` before the first plan is generated.

**Fix:** Conditionally display the phase line only when `total_phases > 0`, or use an alternative message like "No phases planned yet" for fresh colonies.

### IN-02: Duplicate `import { fileURLToPath }` and `import { resolve }` at bottom of host.ts

**File:** `.aether/ts-host/src/host.ts:1239-1240`
**Issue:** The imports at the bottom of the file for `fileURLToPath` from `node:url` and `resolve` from `node:path` appear after all the main code and are only used in the `isMainModule` check. While this works in ESM, colocating imports at the bottom of the file is unconventional and reduces discoverability.

**Fix:** Move these imports to the top of the file with the other import statements.

### IN-03: Unused `definition` parameter in `runDispatchedPlanCommand` and `runDispatchedContinueCommand`

**File:** `.aether/ts-host/src/host.ts:887, 973`
**Issue:** `runDispatchedPlanCommand` and `runDispatchedContinueCommand` do not take a `definition` parameter (unlike `runDispatchedBuildCommand` and `runDryRunDispatchedCommand` which do). This is not a bug but is an inconsistency in the function signatures. The `definition` parameter is only used by the build runner for `ceremonyWorkflow`, while plan and continue hardcode their workflow names.

**Fix:** No fix required, but consider passing `definition` for consistency or extracting the workflow name from the definition in the main switch.

---

_Reviewed: 2026-05-18_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
