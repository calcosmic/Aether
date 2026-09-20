---
phase: 143
slug: build-plan-ceremony-restore
created: 2026-05-18
---

# Phase 143: Validation Strategy

## Test Framework

| Property | Value |
|----------|-------|
| Framework | Node.js built-in test runner (`node:test`) |
| Quick run | `cd .aether/ts-host && npm test 2>&1 \| grep -E "ℹ (tests|pass|fail)"` |
| Full suite | `cd .aether/ts-host && npm test` |
| Go suite | `go test ./... -count=1` |

## Requirements -> Test Map

| Req ID | Behavior | Test Type | Automated Command | Source Plan |
|--------|----------|-----------|-------------------|-------------|
| CEREMONY-01 | PlaybookLoader resolves build playbooks from repo/hub | unit | `node --test ts-host/test/playbook-loader.test.ts` | 143-01 |
| CEREMONY-01 | loadPlaybooksForWorkflow returns 5 build playbooks | unit | `node --test ts-host/test/playbook-loader.test.ts` | 143-01 |
| CEREMONY-01 | renderPlaybookContext truncates to budget | unit | `node --test ts-host/test/playbook-loader.test.ts` | 143-01 |
| CEREMONY-02 | loadPlaybooksForWorkflow returns plan playbooks | unit | `node --test ts-host/test/playbook-loader.test.ts` | 143-01 |
| CEREMONY-02 | Plan playbooks exist at correct paths | static | `test -f .aether/docs/command-playbooks/plan-prep.md && test -f .aether/docs/command-playbooks/plan-dispatch.md` | 143-01 |
| CEREMONY-03 | build.yaml has no orchestration section | static | `grep -c "orchestration:" .aether/commands/build.yaml` returns 0 | 143-01 |
| CEREMONY-03 | plan.yaml has no orchestration section | static | `grep -c "orchestration:" .aether/commands/plan.yaml` returns 0 | 143-01 |
| CEREMONY-04 | TS host is sole conductor for Claude/OpenCode | integration | `node --test ts-host/test/host.test.ts` | 143-02 |
| CEREMONY-04 | Codex ceremony unchanged | regression | `go test ./... -count=1` | 143-02 |
| CEREMONY-05 | Go ceremony events render at correct timing | regression | `cd .aether/ts-host && npm test` | 143-02 |
| CEREMONY-06 | Playbook content injected as document section | unit | `node --test ts-host/test/playbook-loader.test.ts` | 143-01 |
| Regression | All 465+ TS tests pass | full | `cd .aether/ts-host && npm test` | 143-02 |
| Regression | All Go tests pass | full | `go test ./... -count=1` | 143-02 |

## Sampling Rate

- **Per task commit:** `cd .aether/ts-host && npm test 2>&1 | grep -E "ℹ (tests|pass|fail)"`
- **Per wave merge:** `cd .aether/ts-host && npm test`
- **Phase gate:** `cd .aether/ts-host && npm test` AND `go test ./... -count=1`

## Pre-existing Known Failures

2 pre-existing test failures in the TS test suite (unrelated to this phase):
- These existed before Phase 143 work began
- They must not increase beyond 2

## Security

Not applicable. Ceremony rendering is a presentation layer concern with no auth, session management, access control, or cryptography requirements.
