---
status: issues-found
reviewer: automated
phase: 71-platform-hardening
date: 2026-04-28
depth: standard
files_reviewed: 11
files_reviewed_list:
  - cmd/chamber.go
  - cmd/codex_workflow_cmds.go
  - cmd/compatibility_cmds.go
  - cmd/council.go
  - cmd/flag_cmds.go
  - cmd/flags.go
  - cmd/learning_cmds.go
  - cmd/midden_cmds.go
  - cmd/pheromone_mgmt.go
  - cmd/pheromone_write.go
  - cmd/state_cmds.go
findings:
  critical: 1
  warning: 7
  info: 2
  total: 10
---

# Code Review: Phase 71

**Reviewed:** 2026-04-28
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues-found

## Summary

Phase 71 adds missing subcommands (`chamber-compare`, `suggest-approve`, `versions`, council parent, `flag-create` alias) and missing CLI flags (`--phase` on `flag-list`, `--deferred`/`--verbose` on `learning-approve-proposals`, `--window` on `midden-cross-pr-analysis`, `--synthetic` on `continue`, `--export-file` on `pheromone-merge-back`, `--phase-end-only` on `pheromone-expire`, `--verify-only`/`--revert` on `state-mutate`). The most severe finding is a regression: `--verify-only` and `--revert` flags on `state-mutate` had their handling code deleted but their flag registrations preserved, making them silently accepted but ignored. A second critical bug returns the wrong flag's timestamp after resolution. Six additional flags are registered but never consumed, which will mislead users who pass them expecting an effect.

## Critical Issues

### CR-01: `state-mutate --verify-only` and `--revert` flags silently ignored (regression)

**File:** `cmd/state_cmds.go:860-861`
**Issue:** The diff shows that the code handling `--verify-only` and `--revert` flags was deleted from the `stateMutateCmd.RunE` function, but the flag registrations were preserved with updated help text. Users passing `aether state-mutate --verify-only --guard task-complete:1.2` will see the guard silently skipped and a full mutation performed instead of a dry-run check. Similarly, `aether state-mutate --revert task-complete:1.2` will attempt a normal mutation rather than reverting a guard entry.

The deleted code:
```go
// verify-only mode: check guard condition without mutating
verifyOnly, _ := cmd.Flags().GetBool("verify-only")
if verifyOnly && guard != "" {
    outputOK(map[string]interface{}{"guard": guard, "allowed": true, "mode": "verify-only"})
    return nil
}

// revert mode: remove a guard entry from state
revert, _ := cmd.Flags().GetString("revert")
if revert != "" {
    return executeRevertGuard(revert)
}
```

**Fix:** Either restore the deleted handling code (including the `executeRevertGuard` function), or remove the flag registrations entirely. If the intent was to deprecate these flags, remove the registrations and update any documentation or callers.

### CR-02: `flag-resolve` returns wrong flag's timestamp

**File:** `cmd/flag_cmds.go:173`
**Issue:** After resolving a flag by ID, the output always returns `ff.Decisions[0].ResolvedAt` (the timestamp of the first flag in the list) instead of the resolved flag's timestamp. If the resolved flag is not the first entry, the returned timestamp is incorrect.

```go
outputOK(map[string]interface{}{
    "resolved":  true,
    "id":        id,
    "message":   message,
    "timestamp": ff.Decisions[0].ResolvedAt,  // BUG: always index 0
})
```

**Fix:** Capture the timestamp during the loop and use it in the output:
```go
var resolvedAt string
for i := range ff.Decisions {
    if ff.Decisions[i].ID == id {
        ff.Decisions[i].Resolved = true
        ff.Decisions[i].ResolvedAt = time.Now().UTC().Format(time.RFC3339)
        resolvedAt = ff.Decisions[i].ResolvedAt
        ff.Decisions[i].Resolution = message
        found = true
        break
    }
}
// ...
outputOK(map[string]interface{}{
    "resolved":  true,
    "id":        id,
    "message":   message,
    "timestamp": resolvedAt,
})
```

## Warnings

### WR-01: Six flags registered but never consumed in RunE

**Files:** Multiple files (see below)
**Issue:** Six flags added in this phase are registered via `Flags().Bool/String` but never read inside their command's `RunE` function. Passing these flags has zero effect, which will confuse users.

| Flag | File | Line |
|------|------|------|
| `--deferred` | `cmd/learning_cmds.go` | 479 |
| `--verbose` | `cmd/learning_cmds.go` | 480 |
| `--window` | `cmd/midden_cmds.go` | 507 |
| `--synthetic` | `cmd/codex_workflow_cmds.go` | 973 |
| `--export-file` | `cmd/pheromone_mgmt.go` | 296 |
| `--phase-end-only` | `cmd/pheromone_write.go` | 317 |

**Fix:** For each flag, either implement the handling logic in the RunE function, or remove the registration if the feature is not yet ready. If these are placeholders for future work, consider adding a `// TODO: implement` comment at the registration site so reviewers can track them.

### WR-02: `chamber-compare` is a stub returning empty results

**File:** `cmd/chamber.go:172-192`
**Issue:** The `chamber-compare` command always returns empty `matches` and `diffs` arrays regardless of input. The `name` parameter is accepted but never used to look up any chamber data or colony state. If a user or wrapper script relies on the output to detect differences, it will always see "no differences."

**Fix:** Either implement the comparison logic (load chamber manifest and current colony state, compare fields), or mark the command explicitly as a placeholder (e.g., output `"status": "not_implemented"`).

### WR-03: `suggest-approve` always returns empty suggestions

**File:** `cmd/compatibility_cmds.go:120-140`
**Issue:** The `suggest-approve` command loads colony state but returns a hardcoded empty `suggestions` array. The `dry_run` flag is read but only echoed back in the output. This command appears to be a stub that does nothing useful.

**Fix:** Implement the suggestion loading and approval logic, or clearly mark as not yet implemented.

### WR-04: `chamberCreateCmd` accepts unsanitized `name` for directory creation

**File:** `cmd/chamber.go:26-36`
**Issue:** The `--name` parameter is used directly in `filepath.Join` to create a directory. A name containing path separators (e.g., `../../etc`) would create directories outside the `.aether/chambers/` directory. While this is a local CLI tool with a trusted user, it is inconsistent with the defensive coding patterns used elsewhere in the codebase (e.g., `colony.SanitizeSignalContent`).

**Fix:** Validate that `name` does not contain path separators or `..` components:
```go
if strings.ContainsAny(name, "/\\") || strings.Contains(name, "..") {
    outputError(1, "chamber name must not contain path separators or '..'", nil)
    return nil
}
```

### WR-05: `resolveValue` panics on empty string input

**File:** `cmd/state_cmds.go:655`
**Issue:** If `resolveValue` receives an empty string (after trimming), execution falls through all early-return checks and reaches `expr[0]` on an empty string, causing a panic. While this is unlikely to be triggered in normal usage, `state-mutate` accepts arbitrary expressions from users.

```go
if (expr[0] == '{' || expr[0] == '[') && json.Valid([]byte(expr)) {
```

**Fix:** Add an early guard:
```go
if len(expr) == 0 {
    return `""`
}
```

### WR-06: `rand.Read` return value unchecked (crypto/rand)

**File:** `cmd/pheromone_write.go:80`
**Issue:** `rand.Read(rnd)` is called without checking its error return. While `crypto/rand.Read` is documented to always return `len(rnd), nil` on supported platforms, Go's static analysis tools and linters flag unchecked returns from `io.Reader`. This pattern is repeated across the codebase (9 occurrences in cmd/), but the new `pheromone_write.go` usage is in the phase 71 scope.

**Fix:** This is low risk but should be addressed for consistency with Go best practices:
```go
if _, err := rand.Read(rnd); err != nil {
    outputError(2, fmt.Sprintf("failed to generate random bytes: %v", err), nil)
    return nil
}
```

### WR-07: `learning-undo-promotions` ignores save error for observation revert

**File:** `cmd/learning_cmds.go:423`
**Issue:** The `store.SaveJSON("learning-observations.json", obsFile)` call ignores the error return. If saving fails, the instincts are archived but the corresponding observations are not reverted, leaving the system in an inconsistent state.

```go
if reverted > 0 {
    store.SaveJSON("learning-observations.json", obsFile)
}
```

**Fix:** Check the error and report it:
```go
if reverted > 0 {
    if err := store.SaveJSON("learning-observations.json", obsFile); err != nil {
        outputError(2, fmt.Sprintf("failed to revert observations: %v", err), nil)
        return nil
    }
}
```

## Info

### IN-01: `versionsCmd` does not show repo version despite help text claiming it

**File:** `cmd/compatibility_cmds.go:142-152`
**Issue:** The short description says "Show version information for binary, hub, and repo" but the output only includes `binary` and `hub` keys. The repo version is missing.

**Fix:** Either update the short description to "Show version information for binary and hub" or add a repo version field to the output.

### IN-02: `councilCmd` parent command has no RunE or subcommand routing

**File:** `cmd/council.go:282-285`
**Issue:** The new `councilCmd` parent command has no `RunE` function. When a user runs `aether council` without a subcommand, cobra will print the help text but not execute anything useful. This is acceptable for a grouping command, but it may confuse users who expect `aether council` to show something (e.g., history or budget).

**Fix:** This is fine as-is for a parent grouping command. No action required.

---

_Reviewed: 2026-04-28_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
