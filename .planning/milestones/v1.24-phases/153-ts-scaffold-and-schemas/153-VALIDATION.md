---
phase: 153
slug: ts-scaffold-and-schemas
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-22
---

# Phase 153 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 3.2.x |
| **Config file** | control-ts/vitest.config.ts |
| **Quick run command** | `npx vitest run` |
| **Full suite command** | `npm run test:schemas` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `npx vitest run`
- **After every plan wave:** Run `npm run test:schemas`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 153-01-01 | 01 | 1 | CONTROL-01 | — | npm install + tsc --noEmit passes | smoke | `cd control-ts && npm install && npx tsc --noEmit` | ❌ W0 | ⬜ pending |
| 153-01-02 | 01 | 1 | SCHEMA-01 | T-153-01 | Agent schema rejects missing prompt_file | unit | `npx vitest run tests/schemas/agent.schema.test.ts` | ❌ W0 | ⬜ pending |
| 153-01-03 | 01 | 1 | SCHEMA-02 | — | Phase schema validates sample YAML | unit | `npx vitest run tests/schemas/phase.schema.test.ts` | ❌ W0 | ⬜ pending |
| 153-01-04 | 01 | 1 | SCHEMA-03 | T-153-02 | Event schema validates NDJSON lines | unit | `npx vitest run tests/schemas/event.schema.test.ts` | ❌ W0 | ⬜ pending |
| 153-01-05 | 01 | 1 | SCHEMA-04 | — | Policy schema validates sample YAML | unit | `npx vitest run tests/schemas/policy.schema.test.ts` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `control-ts/package.json` — project manifest with scripts and deps
- [ ] `control-ts/tsconfig.json` — strict TypeScript config
- [ ] `control-ts/vitest.config.ts` — test runner config
- [ ] `control-ts/tests/fixtures/` — sample YAML/NDJSON fixtures for all schema types
- [ ] `control-ts/tests/schemas/*.test.ts` — four schema test files
- [ ] `control-ts/src/schemas/*.schema.ts` — four schema source files

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Schema ergonomics review | CONTROL-01 | Human judgment on API surface | Review `src/types/index.ts` exports for clarity and completeness |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
