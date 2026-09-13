---
phase: 203-biological-runtime
plan: "08"
subsystem: pheromones
tags: [pheromone-approval, tick-to-approve, quarantine, ast-guards, go]

# Dependency graph
requires:
  - phase: 203-biological-runtime
    provides: "203-05's one-resolver/provenance/quarantine consolidation (resolveEffectivePheromones, colony.PheromoneSignal.Provenance/Quarantined, the real import path stamping Provenance=import/Quarantined=true) -- this plan's approval surface is what clears that quarantine"
provides:
  - "enqueuePendingNote (cmd/pheromone_approval.go): the one function that adds to the shared tick-to-approve queue, used by both a runtime suggestion (via the import wiring added here) and a cross-project import"
  - "approvePendingNote/editPendingNote/rejectPendingNote: the accept/edit/reject verbs on a queued item, with approving an import routing through clearSignalQuarantine -- the ONE function in this runtime permitted to release a quarantine"
  - "colony.PendingSuggestion.Origin/SignalID/Action/ActionAt (pkg/colony/colony.go): pointer-backed, omitempty fields distinguishing a runtime suggestion from a cross-project import and recording the owner's decision with a timestamp"
  - "Three AST/behavior-based ratchets (TestOneApprovalSurface, TestEveryProposalEntersTheOneQueue, TestQuarantineClearsOnlyOnApproval) plus a runtime-behavior guard (TestRuntimeCannotReleaseAQuarantine), each manually verified this session to fail-by-name against a deliberately introduced violation before being restored"
affects: [203-11, "any future plan extending BIO-08's remaining verbs (reinforce/defer/expire/revoke/appeal) or the immutable action history"]

# Actuals (#2632)
actuals:
  tokens: 16141
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reused the AST-based structural-ratchet pattern 203-05 established (go/ast walk of non-test cmd/ sources) for two new singleness invariants: exactly one approval command (TestOneApprovalSurface) and exactly one queue producer path (TestEveryProposalEntersTheOneQueue), plus a release-direction-specific variant of the existing quarantine-assignment detector (assignsFalseToQuarantinedField, narrower than 203-05's assignsQuarantinedField which catches both directions)."
    - "Extended an existing test-only allowlist var (pheromoneJSONWriterAllowlist) from a different file's init() within the same package, rather than editing the file that declares it -- keeps the ratchet's own documented extension contract ('a new writer must be added here deliberately, with a stated reason') while respecting this plan's declared file-ownership boundary."

key-files:
  created:
    - cmd/pheromone_approval.go
    - cmd/pheromone_approval_test.go
  modified:
    - pkg/colony/colony.go
    - cmd/suggest_approve.go
    - cmd/suggest_approve_test.go
    - cmd/exchange.go

key-decisions:
  - "An approved or rejected queued item is now RETAINED (Dismissed=true, with a timestamped Action) rather than deleted from PendingSuggestions, so plan 203-11's immutable action history has a first entry to build on instead of starting empty, per Task 2's explicit action text. This is a genuine behavior change from the pre-existing suggest-approve command (which spliced the item out of the slice), and required updating one pre-existing test's stale assertion (TestSuggestApprove_ApproveSuggestion)."
  - "Approving a runtime suggestion keeps Provenance='runtime' on the resulting pheromone signal (via the existing, unmodified pheromoneProvenanceFromSource classifier in cmd/pheromone_write.go, which is outside this plan's declared files) rather than 'owner'. Provenance tracks content ORIGIN, not who approved it; the fact that the owner approved it is now a structural guarantee -- a suggestion cannot reach pheromones.json without going through this owner-gated path (D-07) -- rather than a relabeled Provenance value. cmd/pheromone_write.go was not touched."
  - "cmd/exchange.go's importPheromonesData was NOT rerouted through writePheromoneSignal, despite Task 1's action text suggesting it. writePheromoneSignal regenerates the signal ID (sig_<timestamp>_<random>), which would have broken two pre-existing, passing tests asserting exact ID preservation across import (TestImportPheromonesSanitizeSkipsMalicious, TestImportPheromonesSanitizeSkipsOversized). Instead, enqueuePendingNote is called directly from the existing per-signal loop after the existing provenance/quarantine stamping (203-05's own code), preserving ID generation and all existing import test assertions while still satisfying the must_haves truth end-to-end."
  - "cmd/exchange.go is outside this plan's declared files_modified (and outside Task 1's own <files> list), but was edited anyway (Rule 2 deviation) because Task 1's own acceptance criteria and must_haves truths require 'importing one note produces one queued item ... linked by SignalID' -- unreachable without touching the real import command. Not owned by any parallel sibling plan this wave; not touched by 203-05, which already landed as part of this worktree's base. This mirrors 203-05's own precedent (editing cmd/hook_cmds.go and cmd/signal_housekeeping.go outside its declared list) in this same phase."
  - "REQUIREMENTS.md's requirements.mark-complete tool declined BIO-07 with not_found, and BIO-08 was correctly reported as ready:false (shared-ID gate, blocked on plan 203-11). Investigated why BIO-07 was declined rather than assuming a hand-edit was warranted: the tool's checkbox regex requires an exact `**ID**` bold span with nothing else inside it, but this file's actual format bolds the whole `ID — description` phrase (`**BIO-07 — Canonical pheromone bus:**`), confirmed by an isolated regex test that reproduces the non-match. This mismatch is pre-existing and file-format-wide (every other checked requirement in this file is in the identical format), not something this plan introduced or can fix within its declared file scope. Hand-edited BIO-07's checkbox to [x] after confirming the tool's own semantics (ready-ids reported BIO-07 as the sole ready ID) and the root cause of the decline; left BIO-08 unchecked since it is genuinely blocked pending 203-11."

requirements-completed: [BIO-07]

coverage:
  - id: D1
    description: "One queue (colony.PendingSuggestion, extended with Origin/SignalID) carries both runtime-proposed suggestions and cross-project imports; each accepted import produces exactly one queued item linked by SignalID to its already-quarantined stored signal; identical imported content deduplicates to one queued item; a legacy queue item with no Origin field reads as a suggestion; listing shows plain-English origin labels with no untranslated repo word."
    requirement: "BIO-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestPendingNoteOrigin"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestPendingNoteQueue"
        status: pass
      - kind: other
        ref: "go build ./... && go test ./cmd ./pkg/colony -run '^(TestPendingNoteQueue|TestPendingNoteOrigin)' -count=1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Approving a runtime suggestion writes it as an active note through the existing single writer (writePheromoneSignal); approving a cross-project import clears ONLY that linked signal's quarantine flag and writes no new signal; editing a queued item stores the owner's wording and recomputes the content hash; rejecting marks the item dismissed and leaves any linked signal quarantined; a --dry-run call performs zero store writes, proven by both a write counter and a full data-directory content hash."
    requirement: "BIO-08"
    verification:
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestPendingNoteApproval"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestApprovalDryRunDoesNotMutate"
        status: pass
      - kind: other
        ref: "go test ./cmd -run '^(TestPendingNoteApproval|TestQuarantineClearsOnlyOnApproval|TestApprovalDryRunDoesNotMutate)$' -count=1"
        status: pass
    human_judgment: true
    rationale: "BIO-08 (the full evidence-backed influence contract: accept/edit/reject/reinforce/defer/expire/revoke/appeal with immutable history) is only partially delivered here -- this plan implements the three owner-facing verbs (accept/edit/reject) and lays the first Action/ActionAt record; the remaining five verbs and the full immutable history belong to plan 203-11, per the plan's own frontmatter and 203-CLASSIC-SYNTHESIS.md's SYN-203-11 mapping. requirements.ready-ids correctly reports BIO-08 as blocked pending that plan, so it is not marked complete in REQUIREMENTS.md."
  - id: D3
    description: "There is exactly one approval command (suggestApproveCmd), exactly one function permitted to clear a signal's quarantine flag (clearSignalQuarantine, called only from approvePendingNote), and exactly one producer path into the shared queue (enqueuePendingNote, with one pre-existing, named-and-reasoned exception for suggest-analyze's own tested batch pipeline). No runtime path outside this owner-gated surface -- the phase-end feedback emitters, the midden threshold emitter, or a second, unrelated import -- can release a quarantine. Each guard was manually verified this session to fail by name against a deliberately introduced violation, then restored."
    requirement: "BIO-07"
    verification:
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestOneApprovalSurface"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestEveryProposalEntersTheOneQueue"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestQuarantineClearsOnlyOnApproval"
        status: pass
      - kind: unit
        ref: "cmd/pheromone_approval_test.go#TestRuntimeCannotReleaseAQuarantine"
        status: pass
    human_judgment: false

# Metrics
duration: 45min
completed: 2026-09-13
status: complete
---

# Phase 203 Plan 08: One Tick-to-Approve Surface for Suggestions and Quarantined Imports Summary

**One shared queue (`colony.PendingSuggestion`, extended with `Origin`/`SignalID`/`Action`/`ActionAt`) now carries both runtime-proposed pheromone suggestions and cross-project imports, cleared by the same `suggest-approve` command's accept/edit/reject verbs -- and `clearSignalQuarantine` is provably the only function in the runtime that can release a quarantined note.**

## Performance

- **Duration:** 45 min (approximate)
- **Started:** 2026-09-13T09:55:00Z (approx.)
- **Completed:** 2026-09-13T10:49:00Z
- **Tasks:** 3 completed
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `cmd/pheromone_approval.go` (new): `enqueuePendingNote` is the one function that adds to the shared tick-to-approve queue (reusing the existing content-hash dedup rule); `approvePendingNote`/`editPendingNote`/`rejectPendingNote` are the accept/edit/reject verbs; `clearSignalQuarantine` is the single, owner-gated function permitted to release a quarantine.
- `colony.PendingSuggestion` gained pointer-backed `Origin`/`SignalID`/`Action`/`ActionAt` fields (Phase 199 rule: legacy items stay readable, defaulting to `suggestion` origin and no action).
- `cmd/exchange.go`'s real `importPheromonesData` (the actual `aether import pheromones` / `/ant-import-signals` command) now enqueues one linked, quarantined queue item per accepted import -- closing an end-to-end gap between 203-05's quarantine-on-write and this plan's owner-release surface.
- `suggest-approve` gained an `--edit` flag; approving now routes through the single writer (`writePheromoneSignal`) or the single quarantine-release function, replacing the command's own duplicated dedup/write logic; the listing shows each item's origin in plain English ("suggested by the program" / "from another project").
- Four guard tests -- three new AST/behavior-based ratchets plus 203-05's pre-existing `TestOnePheromoneWriterOnly` (extended via a named allowlist entry) -- each manually verified this session to fail by name against a deliberately introduced violation, then restored to the passing state.

## Task Commits

Each task was committed atomically (TDD RED/GREEN pairs where the task's own tests and implementation could be cleanly split by file):

1. **Tasks 1+2 (test, RED):** `31b5725c` -- all eight test functions covering the shared queue, approval surface, and quarantine guards; does not compile without the next commit (references undefined `colony.PendingSuggestion` fields and `cmd/pheromone_approval.go` functions).
2. **Tasks 1+2 (feat, GREEN):** `4bf41b1e` -- `pkg/colony/colony.go` fields, `cmd/pheromone_approval.go`, and the `cmd/suggest_approve.go` rewrite. Makes the previous commit's tests compile and pass.
3. **Deviation fix:** `bf3df896` -- wired `cmd/exchange.go`'s real import command into the shared queue (Rule 2, see Deviations).
4. **Deviation fix:** `f6f4d6ed` -- updated one pre-existing test's now-stale "item deleted on approval" assertion (Rule 1, see Deviations).

**Note on task/commit mapping:** Tasks 1 and 2's tests and implementation could not be cleanly split into four separate commits without either a fabricated intermediate RED state or duplicating file content across commits (both tasks' `<files>` lists overlap on `cmd/pheromone_approval.go` and `cmd/pheromone_approval_test.go`) -- the same precedent this repo's own STATE.md records for Phase 193 P02/P04 ("could not be cleanly split by git hunk"). Task 3's tests are included in the same RED commit (`31b5725c`) since they test invariants already established by Task 1+2's GREEN commit; per 203-05's own precedent, "no production code needed... all three pass immediately" once the underlying functions exist.

**Plan metadata:** commit pending (this SUMMARY)

## Files Created/Modified

- `cmd/pheromone_approval.go` -- `enqueuePendingNote`, `findPendingNote`, `clearSignalQuarantine`, `approvePendingNote`, `editPendingNote`, `rejectPendingNote`, `stampPendingNoteAction`, `pendingNoteOrigin`, `pendingNoteOriginLabel`, `pendingNotesToMap`, and the test-seam `pendingNoteWriteCount` counter
- `cmd/pheromone_approval_test.go` -- 8 test functions (23 subtests) covering all three tasks, plus AST helpers (`assignsFalseToQuarantinedField`, `cobraCommandRunClosures`, `funcLitCallsByName`, `funcDeclConstructsPendingSuggestion`) and a documented allowlist extension for `TestOnePheromoneWriterOnly`
- `pkg/colony/colony.go` -- `PendingSuggestion.Origin`/`SignalID`/`Action`/`ActionAt`, `PendingOriginSuggestion`/`PendingOriginImport`/`PendingOrigins()`, `PendingActionAccepted`/`PendingActionEdited`/`PendingActionRejected`
- `cmd/suggest_approve.go` -- rewritten `RunE` routing through the new functions; `--edit` flag added; ~350 lines of pre-existing duplicated/unreachable approve logic removed
- `cmd/suggest_approve_test.go` -- one assertion updated for the intentional retained-not-deleted behavior (see Deviations)
- `cmd/exchange.go` -- `importPheromonesData` now calls `enqueuePendingNote` per accepted import

## Decisions Made

See `key-decisions` in frontmatter for the full list. The most consequential: an approved/rejected queued item is now RETAINED with a timestamped `Action` rather than deleted, laying the first entry for plan 203-11's immutable action history.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Wired the real cross-project import command into the shared queue**
- **Found during:** Task 1
- **Issue:** `cmd/exchange.go`'s `importPheromonesData` (the actual `aether import pheromones` command) never enqueued anything, so a real import would quarantine a signal (per 203-05) with no way for the owner to ever see or release it through this plan's new surface -- the exact must_haves truth this plan's Task 1 requires.
- **Fix:** Added a call to `enqueuePendingNote` per accepted, already-quarantined imported signal, linking by `SignalID`. Deliberately did NOT reroute the signal write itself through `writePheromoneSignal` (see key-decisions) to avoid regenerating signal IDs and breaking two pre-existing, passing ID-preservation tests.
- **Files modified:** `cmd/exchange.go`
- **Verification:** `TestPendingNoteQueue` (new) proves one queued item per import, linked and deduplicated; `TestImportPheromonesSanitizeSkipsMalicious`/`TestImportPheromonesSanitizeSkipsOversized`/`TestImportPheromonesStampsImportProvenanceAndQuarantine` (pre-existing) verified unchanged.
- **Committed in:** `bf3df896`

**2. [Rule 1 - Bug] Updated a pre-existing test's now-stale "deleted on approval" assertion**
- **Found during:** post-implementation regression sweep (`go test ./cmd -run 'Suggest|Pheromone|...'`)
- **Issue:** `TestSuggestApprove_ApproveSuggestion` asserted the approved item was removed from `PendingSuggestions` entirely -- correct before this plan, now stale, since `approvePendingNote` intentionally retains the item (Dismissed=true, with an Action record) per Task 2's explicit design.
- **Fix:** Updated the assertion to check the item is no longer ACTIVE (via `filterActiveSuggestions`, matching pre-existing dismiss semantics) and carries `Action=accepted`, rather than checking for outright deletion.
- **Files modified:** `cmd/suggest_approve_test.go`
- **Verification:** Full `go test ./cmd -run 'Suggest|Pheromone|Import|Midden|PhaseEnd|Exchange|Hive|Decision|Resolver|Quarantine'` passes.
- **Committed in:** `f6f4d6ed`

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 bug fix on a stale test assertion). **Impact:** Both closed genuine gaps between the plan's stated intent (or its own prior-session's now-obsolete test) and what the code did. No file owned by a parallel sibling plan (203-03/203-04/203-07) was touched.

## Issues Encountered

- `requirements.mark-complete BIO-07` reported `not_found` despite `requirements.ready-ids` reporting it ready. Root-caused (not assumed) to a checkbox-regex format mismatch between the tool (expects an exact `**ID**` bold span) and this file's actual format (`**ID — description:**`, matching every other requirement in the file). This is a pre-existing, file-format-wide tool limitation outside this plan's scope to fix; hand-edited the checkbox after confirming the root cause and that `ready-ids` had already cleared it. BIO-08 correctly remains unchecked, blocked on plan 203-11.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- BIO-07's canonical-bus truth now also covers the owner-release side end-to-end: a real cross-project import is quarantined (203-05) AND queued for owner release through the same surface a runtime suggestion uses (this plan).
- Plan 203-11 can extend `colony.PendingSuggestion`'s `Action`/`ActionAt` scalar into a full immutable action history and add the remaining five BIO-08 verbs (reinforce/defer/expire/revoke/appeal), reusing `enqueuePendingNote`/`approvePendingNote`/`editPendingNote`/`rejectPendingNote` as the established single surface rather than building a parallel one.
- No blockers for parallel sibling plans 203-03/203-04/203-07 in this wave; no file overlap occurred.

---
*Phase: 203-biological-runtime*
*Completed: 2026-09-13*

## Self-Check: PASSED

- FOUND: `cmd/pheromone_approval.go`
- FOUND: `cmd/pheromone_approval_test.go`
- FOUND: commit `31b5725c` (test) in `git log --oneline`
- FOUND: commit `4bf41b1e` (feat) in `git log --oneline`
- FOUND: commit `bf3df896` (fix) in `git log --oneline`
- FOUND: commit `f6f4d6ed` (fix) in `git log --oneline`
- Re-ran plan-level task `<verify>` commands: Task 1 `go test ./cmd ./pkg/colony -run '^(TestPendingNoteQueue|TestPendingNoteOrigin)' -count=1` -- PASS; Task 2 `go test ./cmd -run '^(TestPendingNoteApproval|TestQuarantineClearsOnlyOnApproval|TestApprovalDryRunDoesNotMutate)$' -count=1` -- PASS; Task 3 `go test ./cmd -run '^(TestOneApprovalSurface|TestEveryProposalEntersTheOneQueue|TestRuntimeCannotReleaseAQuarantine)$' -count=1` -- PASS
- Guard tests (`TestOneApprovalSurface`, `TestEveryProposalEntersTheOneQueue`, `TestQuarantineClearsOnlyOnApproval`) independently verified to fail-by-name against manually introduced violations, then restored to clean/passing state (`git status --short` empty on the touched-but-reverted file)
- `go build ./...`, `go vet ./...` clean; full regression sweep (`go test ./cmd -run 'Suggest|Pheromone|Import|Midden|PhaseEnd|Exchange|Hive|Decision|Resolver|Quarantine'`, `go test ./pkg/colony/...`, `go test ./cmd -run TestPlatformParityGolden`) all pass
