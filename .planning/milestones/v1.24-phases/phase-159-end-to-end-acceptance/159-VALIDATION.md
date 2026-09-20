---
phase: 159
slug: end-to-end-acceptance
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-24
---

# Phase 159 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest (npm scripts) |
| **Config file** | `control-ts/vitest.config.ts` |
| **Quick run command** | `go test ./... && npm run test:schemas` |
| **Full suite command** | `go test ./... && npm run test:schemas && npm run test:control` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run full suite (`go test ./... && npm run test:schemas && npm run test:control`)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 159-01-01 | 01 | 1 | TEST-01 | — | N/A | unit | `go test ./...` | ✅ | ⬜ pending |
| 159-01-02 | 01 | 1 | TEST-02 | — | N/A | unit | `npm run test:schemas` | ✅ | ⬜ pending |
| 159-01-03 | 01 | 1 | TEST-03 | — | N/A | unit | `npm run test:control` | ❌ W0 | ⬜ pending |
| 159-02-01 | 02 | 1 | TEST-04 | — | N/A | integration | `npm run aether:control -- --task ...` | ❌ W0 | ⬜ pending |
| 159-02-02 | 02 | 1 | TEST-05 | — | N/A | audit | `scripts/audit-hardcoded.sh` | ❌ W0 | ⬜ pending |
| 159-02-03 | 02 | 1 | TEST-06 | — | N/A | audit | `scripts/audit-extraction.sh` | ❌ W0 | ⬜ pending |
| 159-03-01 | 03 | 1 | TEST-07 | — | N/A | audit | `cat .aether/docs/PARITY_CLASSIC_VS_GO.md` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `control-ts/package.json` — add `test:control` and `aether:control` scripts
- [ ] `scripts/audit-hardcoded.sh` — grep-based audit for prompt text in Go
- [ ] `scripts/audit-extraction.sh` — verify no agent behaviour exists only in Go

*Existing infrastructure: Go tests, schema tests, and colony assets are all present. Only npm scripts and audit scripts need creation.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Demo flow visual review | TEST-04 | Requires human judgment of event completeness | Run `npm run aether:control -- --task "create a small test file and verify it"` and inspect event output |
| Parity checklist completeness | TEST-07 | Requires human judgment of coverage | Verify `.aether/docs/PARITY_CLASSIC_VS_GO.md` has ≥50% items marked MATCH/DEGRADED |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
