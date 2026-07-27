---
phase: 160
slug: fail-loudly
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-27
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

Task IDs are assigned by the planner; this map is keyed by requirement and must be
re-keyed to task IDs once PLAN.md files exist.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | TBD | 1 | RETIRE-01 | — | N/A | e2e | `go build ./cmd/aether && aether integrity` | ✅ | ⬜ pending |
| TBD | TBD | 1 | RETIRE-02/03 | — | N/A | invariant | `git diff --stat -- .aether/ts-host .aether/ts` returns empty | ✅ | ⬜ pending |
| TBD | TBD | 1 | RETIRE-04 | — | N/A | unit + doc | `go test ./cmd -run TestPolicySchemaValidation -count=1` + ledger file present | ❌ W0 | ⬜ pending |
| TBD | TBD | 1 | LOUD-01 | — | N/A | static | `go test ./cmd -run TestSurveyLoadAbsentAndUncalled -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-02 | T-160-01 | Critical secret finding in a changed file causes the continue gate to report a non-passing result | integration | `go test ./cmd -run TestContinueAntiPatternGate -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-03 | — | N/A | static | `go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-04 | — | N/A | self-test | `go test ./cmd -run TestAuditDetectsPositionalDrift -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-05 | — | N/A | integration | `go test ./cmd -run TestCommandCallsMatchCobraContracts -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-06 | — | N/A | invariant | `go test ./cmd -run TestLiveWrapperStderrSuppression -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-07 | — | N/A | static/doc | `go test ./cmd -run TestDocsDoNotClaimConsolidationRunsToday -count=1` | ❌ W0 | ⬜ pending |
| TBD | TBD | 2 | LOUD-08 | — | N/A | existing harness | `go test ./cmd -run TestCommandWrappersReferenceRealYamlSources -count=1` | ✅ harness / ❌ fixture | ⬜ pending |
| TBD | TBD | 3 | D-03/D-04/D-05 (debug trio) | T-160-02 | Debug artifacts written on timeout and non-zero exit reuse existing redaction; no raw unsanitized provider output | unit | `go test ./pkg/codex -run TestWorkerDebug -count=1` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Test names above are indicative; the planner may rename, but every requirement must
retain at least one command that fails when the requirement is unmet (CLAUDE.md Definition of Done).

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

- [ ] All tasks have an automated verify command or a Wave 0 dependency
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references above
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s per task
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
