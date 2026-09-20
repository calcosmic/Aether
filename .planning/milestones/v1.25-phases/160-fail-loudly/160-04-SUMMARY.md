# Plan 160-04 — Summary

**Status:** Complete (3/3 tasks)
**Requirements:** LOUD-02
**Completed:** 2026-07-27

> **Provenance note:** tasks 1–2 were executed by the assigned executor agent. That agent
> was terminated mid-task-3 by an account spend limit (not a code failure). The orchestrator
> committed its in-progress work, then completed task 3 inline — including the deliberate-
> regression proof, which uncovered a defect in the agent's own test (below).

## Why this plan existed

`aether check-antipattern` had been repaired at the CLI level and pinned by
`cmd/security_gate_drift_test.go` — but **nothing called it**. Its intended caller,
`.aether/docs/command-playbooks/continue-gates.md`, is dead documentation whose loader was
deleted in commit `b2b41486`. The live Gatekeeper agent has tools `Read, Grep, Glob, Write`
— no Bash — so it cannot invoke a CLI command at all, and its role is dependency/license
auditing, not secret scanning.

ROADMAP success criterion #2 — "the Gatekeeper security gate actually executes during
continue, and its pass/fail result visibly affects the continue outcome" — was therefore
**not met** by the earlier CLI fix. This plan closes that.

## What changed

**Task 1 — one scanner, not two** (`e0457356`)
The scan logic in `cmd/security_cmds.go` was extracted into a shared unexported function
(`scanFileForAntipatterns`) called by both the `check-antipattern` CLI command and the new
gate check, so the two can never drift apart.

**Task 2 — the missing producer** (`d8276de7`)
`checkAntiPatternGate(files []string) (gateCheck, gateCheck)` added in `cmd/gate.go`,
following the existing `checkNoCriticalFlags` / `checkAllTasksCompleted` pattern. It fills
the `"anti_pattern"` classification slot that had sat fully specified — `softBlock` tier,
recovery template, auto-resolve threshold — with **zero producers** since it was written.

Per decision D-01 it returns *two* checks:
- `anti_pattern` — findings. Stays `softBlock`, as `gate.go` already decided.
- `anti_pattern_executed` — **hard-blocks when the scan cannot run.** A phase must not pass
  with its safety gate unexecuted. This is in `alwaysRunGates`, so it cannot be skipped.

**Task 3 — live wiring + proof** (`80612ee3`, plus the inline completion)
Wired into `runCodexContinueGates` (`cmd/codex_continue.go` ~line 2944) on the **default**
continue path, not only heavy review. Reuses the changed-file list already assembled during
continue (`FilesCreated` + `FilesModified` + `TestsWritten`), surfaced as
`codexClaimVerification.ScannedFiles` — no second notion of "what changed".

## The defect found while proving it

The plan required proving the wiring test fails when the call site is removed. It did not.

`TestContinueAntiPatternGateIsWiredIntoThePipeline` originally asserted only that checks
*named* `anti_pattern` and `anti_pattern_executed` appeared in the report. Replacing the
producer call with two literal `gateCheck{Name: "anti_pattern", Passed: true}` structs
satisfied it — **the test passed with the security gate completely gutted.**

This is the exact failure mode CLAUDE.md's Definition of Done names:
> *"A test that only checks for a named section cannot catch its replacement."*

Rewritten to compare the pipeline's emitted checks against what `checkAntiPatternGate`
produces for the same input — Passed **and** Detail. A stub diverges on Detail even when it
matches on name.

**Verified both directions:**

| State | Result |
|-------|--------|
| Wiring present | `ok` — PASS |
| Call site stubbed | `--- FAIL` — `pipeline Detail "" does not match checkAntiPatternGate's "scanned 0 changed file(s), no critical patterns" — the call site appears to be stubbed rather than calling the producer` |

Worth recording: the other four tests in this file **did** catch the stub. The requirement
was never actually unguarded — but the one test *named* as the wiring guard was the one that
couldn't see it, which is worse than having no such test, because its name invites trust.

## Verification

```
go build ./cmd/aether                        → clean
go vet ./cmd                                 → clean
go test ./cmd -run 'AntiPattern|Antipattern' → ok (all 6 gate tests + CLI tests)
```

## Files

- `cmd/security_cmds.go` (modified — shared scan function extracted)
- `cmd/gate.go` (modified — `checkAntiPatternGate` producer added)
- `cmd/codex_continue.go` (modified — live call site + `ScannedFiles` on the claim verification)
- `cmd/continue_antipattern_gate_test.go` (created — 6 tests, wiring test strengthened)

## Commits

- `e0457356` refactor(160-04): extract shared antipattern scan function
- `d8276de7` feat(160-04): add checkAntiPatternGate producer and hard-block scan gate
- `80612ee3` wip(160-04): live gate wiring + integration test (agent halted by spend limit)
- (this commit) test(160-04): make the wiring test detect a stubbed call site
