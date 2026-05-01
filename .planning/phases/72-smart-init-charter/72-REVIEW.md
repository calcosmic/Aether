---
phase: 72-smart-init-charter
reviewed: 2026-04-28T12:00:00Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - .claude/commands/ant/init.md
  - .opencode/commands/ant/init.md
  - cmd/codex_visuals.go
  - cmd/init_ceremony_test.go
  - cmd/init_ceremony.go
  - cmd/init_cmd_test.go
  - cmd/init_cmd.go
  - cmd/init_research_test.go
  - cmd/init_research.go
  - pkg/colony/colony_test.go
  - pkg/colony/colony.go
  - pkg/colony/testdata/COLONY_STATE.golden.json
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 72: Code Review Report

**Reviewed:** 2026-04-28T12:00:00Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

This phase implements a "smart init charter" -- adding a `Charter` sub-object to `ColonyState`, a new `init-ceremony` CLI command that runs the full research-scan-charter-approve flow, expanded charter fields (tech stack, key risks, constraints), pheromone suggestion generation, and charter field length validation. The wrapper commands (Claude/OpenCode) were updated to pass charter JSON to `aether init` and display all 7 charter sections.

The core data model and `aether init` path are solid. However, `init-ceremony` has a significant bug: it bypasses the idempotency and sealed-colony protection that `aether init` provides, meaning it can silently overwrite an active colony. There are also several code quality issues.

## Critical Issues

### CR-01: `init-ceremony` bypasses idempotency and sealed colony detection

**File:** `cmd/init_ceremony.go:293-339`
**Issue:** `createCeremonyColony()` unconditionally creates `COLONY_STATE.json`, `session.json`, and artifacts. It does NOT check whether a colony already exists, whether it is active, or whether it is sealed. The `aether init` command (in `init_cmd.go` lines 52-76) has proper idempotency checks: it blocks if an active colony exists, detects sealed colonies, and creates backups before overwriting. `createCeremonyColony` does none of this.

If a user runs `aether init-ceremony` on a directory with an existing active colony, it will silently overwrite the colony state, destroying the current session. If a sealed colony exists, it will overwrite without creating a backup.

**Fix:** Add the same idempotency and sealed-colony detection logic from `init_cmd.go` into `createCeremonyColony()`:

```go
func createCeremonyColony(goal string, scope colony.ColonyScope, charter colony.Charter) error {
    dataDir := store.BasePath()
    statePath := filepath.Join(dataDir, "COLONY_STATE.json")

    // Check idempotency: if COLONY_STATE.json exists, inspect it
    if _, err := os.Stat(statePath); err == nil {
        var existing colony.ColonyState
        if loadErr := store.LoadJSON("COLONY_STATE.json", &existing); loadErr == nil {
            if existing.Goal == nil || strings.TrimSpace(ptrStr(existing.Goal)) == "" || existing.State == colony.StateIDLE {
                goto createFreshColony
            }
            if existing.Milestone == "Crowned Anthill" {
                if sealInProgress(dataDir) {
                    return fmt.Errorf("a seal operation appears to be in progress")
                }
                // Fall through -- backup before overwrite
            } else {
                return fmt.Errorf("colony already initialized (state=%s, phase=%d)", existing.State, existing.CurrentPhase)
            }
        }
    }

createFreshColony:
    // Backup existing state before overwriting
    if _, err := os.Stat(statePath); err == nil {
        backupDir := filepath.Join(dataDir, "backups")
        if err := os.MkdirAll(backupDir, 0755); err == nil {
            backupFile := filepath.Join(backupDir, fmt.Sprintf("COLONY_STATE.pre-init-ceremony.%s.bak", time.Now().Format("20060102-150405")))
            if err := copyFile(statePath, backupFile); err == nil {
                fmt.Fprintf(os.Stderr, "warning: backed up previous colony state to %s\n", backupFile)
            }
        }
    }

    // ... rest of creation logic
}
```

## Warnings

### WR-01: `--charter-json` flag registered on `init-ceremony` but never read

**File:** `cmd/init_ceremony.go:402`
**Issue:** The `--charter-json` flag is registered on `initCeremonyCmd` (line 402), but `runInitCeremony()` never reads it. The flag is documented as being for "non-interactive mode" in the help text, but the non-interactive code path does not exist -- the ceremony always runs the research scan and prompts. A user passing `--charter-json --non-interactive` would get a TTY check bypass followed by a hang on stdin prompt.

**Fix:** Either implement the non-interactive path (read the flag, skip research and prompt), or remove the flag registration and the TTY-skip logic that references it:

```go
// Option A: Remove the unused flag
func init() {
    initCeremonyCmd.Flags().String("target", ".", "Directory to scan")
    initCeremonyCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope: project or meta")
    // Remove: initCeremonyCmd.Flags().Bool("non-interactive", false, "...")
    // Remove: initCeremonyCmd.Flags().String("charter-json", "", "...")
    rootCmd.AddCommand(initCeremonyCmd)
}
```

### WR-02: Dead code -- `tmpCmd` variable in `runCeremonyResearch`

**File:** `cmd/init_ceremony.go:183-188`
**Issue:** A temporary cobra command `tmpCmd` is created, flags are set on it, and `SetArgs` is called -- but it is never executed. Only the later `researchCmd` (line 210) is actually used. The `tmpCmd` is dead code that adds confusion.

**Fix:** Remove the unused `tmpCmd` block:

```go
func runCeremonyResearch(goal, target string) (*colony.Charter, []pheromoneSuggestion, error) {
    // Remove the tmpCmd block entirely (lines 183-188)

    origStdout := stdout
    buf := bytes.NewBuffer(nil)
    stdout = buf
    defer func() { stdout = origStdout }()

    researchCmd := &cobra.Command{
        Use:  "init-research",
        Args: cobra.NoArgs,
        RunE: initResearchCmd.RunE,
    }
    // ... rest
}
```

### WR-03: `validateCharterFieldLength` not called by `init-ceremony`

**File:** `cmd/init_cmd.go:296-318` (defined), `cmd/init_ceremony.go:293` (not called)
**Issue:** The `validateCharterFieldLength` function is only called from `aether init` when `--charter-json` is passed. The `init-ceremony` command populates the charter from `generateCharter()` output, which uses string concatenation from user input (goal string). A very long goal string would produce charter fields exceeding the 2000-char limit. While the ceremony's charter comes from generated text rather than raw user JSON, the inconsistency means `aether init --charter-json` enforces limits while `init-ceremony` does not.

**Fix:** Call `validateCharterFieldLength` in `createCeremonyColony` before saving state:

```go
func createCeremonyColony(goal string, scope colony.ColonyScope, charter colony.Charter) error {
    if err := validateCharterFieldLength(charter); err != nil {
        return fmt.Errorf("charter validation failed: %w", err)
    }
    // ... rest of creation
}
```

### WR-04: Global `stdout` mutation in `runCeremonyResearch` is not concurrency-safe

**File:** `cmd/init_ceremony.go:195-207`
**Issue:** `runCeremonyResearch` temporarily replaces the global `stdout` variable to capture `init-research` output, then restores it via `defer`. While the ceremony flow is single-threaded, this pattern of mutating a package-level variable for output capture is fragile. If two ceremony calls were ever made concurrently (e.g., in tests), the stdout redirect would be corrupted. The same pattern exists in other commands but is worth noting here since `init-ceremony` introduces a new instance of it.

**Fix:** No immediate fix required for correctness (single-threaded), but consider a more robust approach in future iterations, such as passing the output writer as a parameter or using a context-based output sink.

## Info

### IN-01: Dead code -- `hasSuffix` function never called

**File:** `cmd/init_research.go:667-674`
**Issue:** The `hasSuffix` function is defined but never called anywhere in the codebase. It was likely added speculatively.

**Fix:** Remove the unused function.

### IN-02: No test for `validateCharterFieldLength`

**File:** `cmd/init_cmd.go:296-318`
**Issue:** The charter field length validation function has no unit test. It should have at least one test for normal input and one for input exceeding the 2000-char limit.

**Fix:** Add a test:
```go
func TestValidateCharterFieldLength(t *testing.T) {
    // Normal input
    ch := colony.Charter{Intent: "short", Vision: "short"}
    if err := validateCharterFieldLength(ch); err != nil {
        t.Errorf("unexpected error: %v", err)
    }

    // Over-limit input
    long := strings.Repeat("x", 2001)
    ch.Intent = long
    if err := validateCharterFieldLength(ch); err == nil {
        t.Error("expected error for oversized intent field")
    }
}
```

### IN-03: Redundant if/else branches in `runCeremonyResearch`

**File:** `cmd/init_ceremony.go:196-201`
**Issue:** Both branches of the if/else do exactly the same thing:
```go
if origStdout != nil {
    buf = bytes.NewBuffer(nil)
    stdout = buf
} else {
    buf = bytes.NewBuffer(nil)
    stdout = buf
}
```
This is a no-op conditional.

**Fix:** Simplify to:
```go
origStdout := stdout
buf := bytes.NewBuffer(nil)
stdout = buf
defer func() { stdout = origStdout }()
```

---

_Reviewed: 2026-04-28T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
