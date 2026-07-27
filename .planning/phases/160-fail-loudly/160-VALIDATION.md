---
phase: 160
slug: fail-loudly
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-27
updated: 2026-07-27
---

# Phase 160 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `160-RESEARCH.md` § Validation Architecture.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), invoked via `go test` |
| **Config file** | none — standard `go test ./...` |
| **Quick run command** | `go test ./cmd/... ./pkg/... -count=1` |
| **Full suite command** | `go test ./... -count=1 -race` |
| **Estimated runtime** | ~60–180 seconds full suite |

Supplementary phase-gate commands (not `go test`): `go build ./cmd/aether`, `go vet ./...`, `aether integrity`, `git diff --stat -- .aether/ts-host .aether/ts`.

---

## Sampling Rate

- **After every task commit:** Run the specific new test(s) for that task's requirement, e.g. `go test ./cmd -run TestName -count=1`
- **After every plan wave:** Run `go test ./cmd/... ./pkg/... -count=1`
- **Before `/gsd-verify-work`:** `go test ./... -count=1 -race`, `go build ./cmd/aether`, `go vet ./...`, and `aether integrity` all green
- **Max feedback latency:** 30 seconds for per-task runs

---

## Per-Task Verification Map

Re-keyed against the 8 plans as written (2026-07-27). Task-level IDs are assigned
inside each PLAN.md; this map is keyed by requirement → plan → command.

| Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 160-01 | 1 | RETIRE-04 (safety net) | — | N/A | unit + doc | `go test ./cmd -run TestPolicySchemaRequiredFields -count=1` + `.aether/docs/retired-tests-ledger.md` present | ❌ W0 | ⬜ pending |
| 160-02 | 1 | LOUD-01 | — | N/A | static | `go test ./cmd -run TestSurveyLoadAbsentAndUncalled -count=1` | ❌ W0 | ⬜ pending |
| 160-02 | 1 | LOUD-03 | — | N/A | static | `go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1` (delivered by 160-07) | ❌ W0 | ⬜ pending |
| 160-03 | 1 | LOUD-06 | — | N/A | invariant | `go test ./cmd -run TestLiveWrapperStderrSuppressionCount -count=1` | ❌ W0 | ⬜ pending |
| 160-03 | 1 | LOUD-07 | — | N/A | static/doc | `go test ./cmd -run TestDocsDoNotClaimConsolidationRunsToday -count=1` | ❌ W0 | ⬜ pending |
| 160-04 | 1 | LOUD-02 | T-160-01 | A critical secret finding in a changed file makes the continue gate report a non-passing result; inability to scan hard-blocks | integration | `go test ./cmd -run 'TestContinueAntiPatternGate\|TestAntiPatternScanFailureHardBlocks' -count=1` | ❌ W0 | ⬜ pending |
| 160-05 | 1 | LOUD-09 | T-160-02 | Debug writes on timeout and non-zero exit reuse the existing redaction/sanitization path; no raw unsanitized provider output reaches disk | unit | `go test ./pkg/codex -run TestWriteHostedWorkerOutputDebug -count=1` and `go test ./cmd -run TestDataCleanWorkerDebug -count=1` | ❌ W0 | ⬜ pending |
| 160-06 | 2 | RETIRE-01/02/03 | — | N/A | e2e + invariant | `go test ./cmd -run TestRetiredPackagesStayRetired -count=1`; `go build ./cmd/aether && aether integrity`; `git diff --stat -- .aether/ts-host .aether/ts` empty | ✅ / ❌ W0 | ⬜ pending |
| 160-07 | 2 | LOUD-04 | — | N/A | self-test | `go test ./cmd -run TestAuditDetectsPositionalDrift -count=1` | ❌ W0 | ⬜ pending |
| 160-07 | 2 | LOUD-05 | — | N/A | integration (real execution, not regex) | `go test ./cmd -run TestCommandCallsExecuteAsDocumented -count=1` | ❌ W0 | ⬜ pending |
| 160-07 | 2 | D-01 classification | — | Gate-classified calls must halt when they cannot execute | invariant | `go test ./cmd -run TestGateClassifiedCallsHaveGateWiring -count=1` | ❌ W0 | ⬜ pending |
| 160-08 | 2 | LOUD-08 | — | N/A | static | `go test ./cmd -run TestSlashCommandGuidancePointsAtRealCommands -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Every requirement retains at least one command that fails when the requirement is
unmet (CLAUDE.md Definition of Done). Executors may rename tests; they may not drop
the failing-on-regression property.

---

## Wave 0 Requirements

- [ ] `cmd/policy_schema_test.go` — RETIRE-04 replacement for `control-ts/tests/schemas/policy.schema.test.ts` (must exist before or alongside the deletion)
- [ ] `.aether/docs/retired-tests-ledger.md` — RETIRE-04 ledger, including retroactive entry for `.aether/ts-host/test/playbook-loader.test.ts` (deleted in `b2b41486`)
- [ ] Generalized command-call audit test (LOUD-03/04/05) — one audit, three requirements
- [ ] LOUD-06 stderr-suppression invariant test over the two live wrapper directories
- [ ] LOUD-07 static-doc test over `CLAUDE.md`, `AGENTS.md`, `.aether/docs/structural-learning-stack.md`
- [ ] LOUD-02 live-path gate test (must exercise the gate pipeline, not only the CLI command)
- [ ] `.aether/commands/unblock.yaml` + `.claude/commands/ant/unblock.md` + `.opencode/commands/ant/unblock.md` — LOUD-08 fixtures
- [ ] Debug-artifact tests for the timeout and non-zero-exit paths (D-03)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| A previously-broken call failing now shows a visible error in the terminal rather than silence | ROADMAP SC#1 | Terminal legibility for a human operator cannot be asserted by a Go test | Run `/ant-build` or `/ant-continue` with a deliberately broken call; confirm the failure is readable in the terminal without opening a transcript |
| The `check-antipattern` result visibly affects the continue outcome | ROADMAP SC#2 | The automated test proves the gate result; that a human *sees* it is a UX judgement | Run `/ant-continue` on a phase with a planted hardcoded secret; confirm the gate outcome is stated in the output |
| Goldens regenerated correctly after `/ant-unblock` is added | LOUD-08 | `-update-golden` output needs human review before commit | Run the golden update, inspect the diff for unintended churn |

---

## Validation Sign-Off

- [x] All tasks have an automated verify command or a Wave 0 dependency — verified against all 8 plans; no `MISSING` markers
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references above — Wave 0 gap-fillers are delivered as wave-1 tasks in plans 01–05
- [x] No watch-mode flags
- [x] Feedback latency < 30s per task
- [x] `nyquist_compliant: true` set in frontmatter
- [ ] `wave_0_complete` — flips to true during execution, once the wave-1 test files exist and run

**Approval:** approved 2026-07-27 (plan-checker: 0 blockers)
