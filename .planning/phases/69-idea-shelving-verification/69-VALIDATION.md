---
phase: 69
slug: idea-shelving-verification
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-28
---

# Phase 69 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./cmd/ -run Shelf -v -count=1` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/ -run Shelf -v -count=1`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 69-01-01 | 01 | 1 | SHELF-01 to SHELF-05 | — | N/A | grep + unit | `go test ./cmd/... -run "TestShelf\|TestDetectExpiredFocus\|TestDetectLowConfidenceInstinct\|TestDetectUnresolvedFlag\|TestDetectRecurringRedirect\|TestDetectNoCandidates\|TestDetectDeduplicates\|TestLoadActiveShelf\|TestPromoteShelfEntry\|TestDismissShelfEntry\|TestShelfEntryToTodo\|TestFormatShelfForInit\|TestInitShelfBacklogOutput\|TestCopyShelfToChamber\|TestShelfChamberSummary" -count=1 2>&1 \| grep -c "^--- FAIL" \| xargs -I{} test {} -eq 0` | ✅ W0 | ⬜ pending |
| 69-01-02 | 01 | 1 | SHELF-01 to SHELF-05 | — | N/A | file + grep | `test -f .planning/phases/65-idea-shelving/65-VERIFICATION.md && grep -q "status: passed" .planning/phases/65-idea-shelving/65-VERIFICATION.md && grep -q "SHELF-0[1-5]" .planning/phases/65-idea-shelving/65-VERIFICATION.md` | ✅ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. 23 shelf tests already exist and pass. Storage layer (flock) provides concurrent write safety. No new test infrastructure needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| OpenCode wrapper parity observation | SHELF-02, SHELF-03, SHELF-05 | Documentation gap analysis requires cross-file comparison | Compare .opencode/commands/ant/init.md and entomb.md against .claude/ equivalents |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
