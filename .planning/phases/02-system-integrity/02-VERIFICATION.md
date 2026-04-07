---
phase: 02-system-integrity
verified: 2026-04-07T19:30:00Z
status: gaps_found
score: 2/5
overrides_applied: 0
gaps:
  - truth: "A user pheromone with source='user' or source='cli' is never flagged as a test artifact by isTestArtifact, regardless of content"
    status: failed
    reason: "The isTestArtifact function in cmd/suggest.go does NOT check the source field. The fix was committed (f2aa4e8b) but that commit is NOT an ancestor of HEAD -- the 02-02 commit (4e9377db) branched from pre-02-01 state, reverting all 02-01 changes. The function still uses only id-prefix and content-substring checks, so user pheromones containing 'test' or 'demo' will be false-positively flagged."
    artifacts:
      - path: "cmd/suggest.go"
        issue: "No source field check exists in isTestArtifact. Lines 7-27 show only HasPrefix and Contains checks."
    missing:
      - "Add source field check: if source == 'user' || source == 'cli' { return false } before existing checks"
      - "Create TestIsTestArtifact test function covering source='user' and source='cli' cases"
  - truth: "Running backup-prune-global without --confirm produces a dry-run preview and deletes nothing"
    status: failed
    reason: "backup-prune-global has NO --confirm flag. The flag registration and dry-run logic were committed in f2aa4e8b but reverted by the 02-02 commit (4e9377db). Currently the command deletes files immediately when run with no safety gate. Line 217 shows only --cap flag registered. Lines 155-160 show os.Remove called unconditionally."
    artifacts:
      - path: "cmd/maintenance.go"
        issue: "No --confirm flag on backupPruneGlobalCmd (line 217). Delete loop runs unconditionally (lines 155-160)."
    missing:
      - "Add backupPruneGlobalCmd.Flags().Bool('confirm', false, ...) to init()"
      - "Add dry-run short-circuit before the delete loop: if !confirm { output dry_run:true; return nil }"
      - "Add TestBackupPruneGlobalConfirmGate test"
  - truth: "Running temp-clean without --confirm produces a dry-run preview and deletes nothing"
    status: failed
    reason: "temp-clean has NO --confirm flag. The flag registration and dry-run logic were committed in f2aa4e8b but reverted by the 02-02 commit (4e9377db). Currently the command deletes old temp files immediately when run. Lines 200-205 show os.Remove called unconditionally in the loop."
    artifacts:
      - path: "cmd/maintenance.go"
        issue: "No --confirm flag on tempCleanCmd. Delete loop runs unconditionally (lines 200-205)."
    missing:
      - "Add tempCleanCmd.Flags().Bool('confirm', false, ...) to init()"
      - "Collect removable files first, then add dry-run short-circuit before deletion"
      - "Add TestTempCleanConfirmGate test"
---

# Phase 2: System Integrity Verification Report

**Phase Goal:** Eliminate data-loss risks from hygiene commands and ensure all Go commands run cleanly on a fresh install.
**Verified:** 2026-04-07T19:30:00Z
**Status:** gaps_found
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User pheromone with source='user' or 'cli' is never flagged by isTestArtifact | FAILED | cmd/suggest.go lines 7-27: no source field check. Fix committed in f2aa4e8b but NOT an ancestor of HEAD (02-02 branched from pre-02-01 state, reverting all 02-01 changes) |
| 2 | backup-prune-global without --confirm produces dry-run preview, deletes nothing | FAILED | cmd/maintenance.go line 217: no --confirm flag registered. Lines 155-160: os.Remove called unconditionally |
| 3 | temp-clean without --confirm produces dry-run preview, deletes nothing | FAILED | cmd/maintenance.go: no --confirm flag on tempCleanCmd. Lines 200-205: os.Remove called unconditionally |
| 4 | Error messages in modified files follow prefix: description. remediation hint format | FAILED | cmd/helpers.go: no remediation hint comment (removed by 02-02 commit). cmd/maintenance.go error messages lack prefixes (e.g., line 42: "failed to parse pheromones.json" has no "data-clean:" prefix) |
| 5 | All registered aether subcommands run without panic on a fresh install | VERIFIED | cmd/smoke_test.go TestSmokeCommands passes with 238 subcommands (239 minus skipped "serve"). go test ./cmd/ -run TestSmokeCommands -count=1 passes in 1.4s |

**Score:** 2/5 truths verified

### Root Cause Analysis

The 02-01 plan changes were committed on a separate branch (commits f2aa4e8b, 9311d017, 20fc74dd) that is NOT an ancestor of HEAD. The 02-02 plan then branched from the pre-02-01 state (commit a564b1a4), and its commit 4e9377db rewrote suggest.go, maintenance.go, and helpers.go back to their pre-02-01 state. This means:

- All isTestArtifact source-field fixes were lost
- All --confirm safety gates on backup-prune-global and temp-clean were lost
- All error message convention additions were lost
- All TestIsTestArtifact, TestBackupPruneGlobalConfirmGate, and TestTempCleanConfirmGate tests were never merged

The git graph shows the divergence clearly:
```
* db91c845 (HEAD) chore(02-02)
* b8c7898d test(02-02)
* 4e9377db feat(02-02) -- rewrote files to pre-02-01 state
* e1490450 docs(02-01) -- summary only
| * 9311d017 docs(02-01) -- error convention (ORPHANED)
| * f2aa4e8b feat(02-01) -- source check + confirm gates (ORPHANED)
| * 20fc74dd test(02-01) -- tests for above (ORPHANED)
|/
* a564b1a4 docs(02)
```

The 02-01 feature commits (f2aa4e8b, 20fc74dd, 9311d017) exist in the repo but are not reachable from HEAD.

### Deferred Items

No deferred items -- all 5 failures are Phase 2 scope and not addressed by any later phase in the roadmap.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/suggest.go` | isTestArtifact with source field check | STUB | Function exists but lacks source='user'/'cli' guard. Only has id-prefix and content-substring checks (original fragile logic) |
| `cmd/maintenance.go` | backup-prune-global and temp-clean with --confirm flags | STUB | data-clean has --confirm (pre-existing). backup-prune-global and temp-clean have NO --confirm -- they delete unconditionally |
| `cmd/helpers.go` | Error message convention documented | STUB | No "remediation hint" comment exists. outputError function unchanged from pre-phase state |
| `cmd/maintenance_test.go` or equivalent | Tests for isTestArtifact fix and confirmation gates | MISSING | No TestIsTestArtifact, TestBackupPruneGlobalConfirmGate, or TestTempCleanConfirmGate test functions exist anywhere in cmd/ |
| `cmd/smoke_test.go` | Smoke test suite for all subcommands | VERIFIED | 85 lines, TestSmokeCommands iterates rootCmd.Commands(), 238 commands pass, serve skipped |
| `cmd/deprecated_cmds.go` | DELETED | VERIFIED | File does not exist |
| `cmd/deprecated_cmds_test.go` | DELETED | VERIFIED | File does not exist |
| `cmd/suggest_cmds_test.go` | DELETED | VERIFIED | File does not exist |
| `.aether/utils/*.sh` | All deleted | VERIFIED | find .aether/utils/ -name "*.sh" returns 0 results |
| `.aether/utils/oracle/oracle.md` | PRESERVED | VERIFIED | File exists |
| `.aether/utils/hooks/clash-pre-tool-use.js` | PRESERVED | VERIFIED | File exists |
| `.aether/utils/queen-to-md.xsl` | PRESERVED | VERIFIED | File exists |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| cmd/maintenance.go | cmd/suggest.go | isTestArtifact function call | WIRED | data-clean calls isTestArtifact at line 64. However, the function lacks the source guard, making this a data-loss risk |
| cmd/maintenance.go | cmd/helpers.go | outputError calls | WIRED | Multiple outputError/outputErrorMessage calls throughout maintenance.go |
| cmd/smoke_test.go | rootCmd.Commands() | iterates all registered subcommands | WIRED | Line 21: commands := rootCmd.Commands() |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| cmd/maintenance.go (data-clean) | isTestArtifact(signal) | signal["source"] field | DISCONNECTED | isTestArtifact never reads signal["source"] -- user/cli signals can be false-positively flagged |
| cmd/smoke_test.go | rootCmd.Commands() | cobra registration | FLOWING | All 239 commands registered, 238 tested successfully |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Build compiles | go build ./cmd/ | Compiled successfully | PASS |
| go vet passes | go vet ./cmd/ | No issues | PASS |
| Smoke tests pass | go test ./cmd/ -run TestSmokeCommands -count=1 | 238/239 pass (serve skipped) | PASS |
| Full test suite | go test ./... -count=1 -timeout 300s | All pass except pre-existing pkg/exchange failure | PASS |
| isTestArtifact tests exist | go test ./cmd/ -run TestIsTestArtifact -count=1 | "no tests to run" | FAIL |
| backup-prune has --confirm | grep confirm cmd/maintenance.go | Only on data-clean, NOT on backup-prune-global or temp-clean | FAIL |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| INTG-01 | 02-02 | All primary commands run without error on clean installation | SATISFIED | Smoke tests pass for 238 subcommands |
| INTG-02 | 02-02 | No orphaned shell scripts in active code paths | SATISFIED | 0 .sh files in .aether/utils/, command specs updated |
| INTG-03 | 02-01 | Error handling produces consistent, readable, actionable log output | BLOCKED | Error convention comment and prefixed messages were reverted by 02-02 |
| INTG-04 | 02-01 | isTestArtifact no longer false-positives on legitimate user data | BLOCKED | Source field check was reverted by 02-02 |
| INTG-05 | 02-01 | backup-prune-global and temp-clean require confirmation before deleting | BLOCKED | --confirm gates were reverted by 02-02 |
| INTG-06 | 02-02 | All 524+ existing tests pass with no regressions | SATISFIED | go test ./... passes (except pre-existing pkg/exchange failure) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| cmd/suggest.go | 7-27 | isTestArtifact lacks source field check -- user pheromones with "test" or "demo" content will be deleted by data-clean | BLOCKER | Data loss: user-created pheromones falsely flagged as test artifacts |
| cmd/maintenance.go | 155-160 | backup-prune-global deletes files unconditionally with no --confirm safety gate | BLOCKER | Data loss: accidental backup deletion with no preview |
| cmd/maintenance.go | 200-205 | temp-clean deletes files unconditionally with no --confirm safety gate | BLOCKER | Data loss: accidental temp file deletion with no preview |

### Human Verification Required

None -- all failures are verifiable programmatically.

### Gaps Summary

Three of five roadmap success criteria failed due to a branch merge issue. The 02-01 plan was executed on a separate commit chain (f2aa4e8b -> 9311d017) that was NOT merged into the main branch before 02-02 was built. The 02-02 commit (4e9377db) branched from the pre-02-01 state and rewrote the same files, effectively reverting all 02-01 changes.

The orphaned 02-01 commits contain the correct implementations:
- `f2aa4e8b` -- isTestArtifact source guard + --confirm gates on backup-prune-global and temp-clean
- `20fc74dd` -- Tests for the above (TestIsTestArtifact, TestBackupPruneGlobalConfirmGate, TestTempCleanConfirmGate)
- `9311d017` -- Error message convention comment in helpers.go

The fix requires cherry-picking or re-applying these changes on top of the current HEAD. The 02-02 changes (deprecated code removal, smoke tests) are correctly in place and should be preserved.

---

_Verified: 2026-04-07T19:30:00Z_
_Verifier: Claude (gsd-verifier)_
