---
phase: 76
slug: ux-improvements
status: active
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-29
---

# Phase 76 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none |
| **Quick run command** | `go test ./cmd/...` |
| **Full suite command** | `go test ./... -race` |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./cmd/...`
- **After every plan wave:** Run `go test ./... -race`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 76-01-T1 | 01 | 1 | UX-01 | T-76-02 | Marker file is boolean, no content to tamper | unit | `go test ./cmd/ -run "TestFirstRun" -count=1` | No (Wave 0) | ⬜ |
| 76-01-T1 | 01 | 1 | UX-01 | T-76-01 | Visual-only output, gated by shouldRenderVisualOutput | unit | `go test ./cmd/ -run "TestFirstRun" -count=1` | No (Wave 0) | ⬜ |
| 76-01-T1 | 01 | 1 | UX-02 | T-76-03 | Friendly errors replace internal text, no regex injection | unit | `go test ./cmd/ -run "TestFriendlyError|TestRenderVisualError" -count=1` | No (Wave 0) | ⬜ |
| 76-01-T1 | 01 | 1 | UX-02 | T-76-01 | strings.Contains only, no regex on error messages | unit | `go test ./cmd/ -run "TestFriendlyError" -count=1` | No (Wave 0) | ⬜ |
| 76-01-T2 | 01 | 1 | UX-03 | T-76-05 | progressbar/v3 pinned, zero transitive deps | unit | `go list -m github.com/schollz/progressbar/v3` | No (Wave 0) | ⬜ |
| 76-01-T2 | 01 | 1 | UX-03 | T-76-04 | Non-TTY fallback via isTerminalWriter gate | unit | `go test ./cmd/ -run "TestCeremonyProgress" -count=1` | No (Wave 0) | ⬜ |
| 76-01-T2 | 01 | 1 | UX-03 | T-76-06 | Nil-safe progress, hardcoded step names | unit | `go test ./cmd/ -run "TestCeremonyProgress" -count=1` | No (Wave 0) | ⬜ |
| 76-02-T1 | 02 | 1 | UX-04 | T-76-07 | Warnings display existing state data, no secrets | unit | `go test ./cmd/ -run "TestComputeWarnings|TestRenderWarningsSection" -count=1` | No (Wave 0) | ⬜ |
| 76-02-T1 | 02 | 1 | UX-04 | T-76-08 | Missing/corrupt files skipped, bounded iteration | unit | `go test ./cmd/ -run "TestComputeWarnings" -count=1` | No (Wave 0) | ⬜ |
| 76-02-T1 | 02 | 1 | UX-04 | T-76-09 | Hardcoded suggestion strings, deterministic from state | unit | `go test ./cmd/ -run "TestWorkflowSuggestions" -count=1` | No (Wave 0) | ⬜ |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| First-run welcome banner renders correctly in a real terminal | UX-01 | Requires interactive terminal (TTY) to verify emoji and alignment | Run `aether status` in a fresh repo with no colony. Verify banner shows with ant emoji, divider, and 3 quick-start commands. |
| Progress bar animates smoothly during build ceremony | UX-03 | Requires interactive terminal to see live progress bar updates | Run `aether build 1` and observe step-based progress with elapsed timing. |
| Dashboard warnings section renders with emoji banner | UX-04 | Requires interactive terminal to verify visual layout | Run `aether status` with a stale colony (last activity >7 days ago). Verify warnings section appears with warning emoji banner. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
