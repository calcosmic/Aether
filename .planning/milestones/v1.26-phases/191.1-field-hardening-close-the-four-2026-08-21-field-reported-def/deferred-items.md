# Deferred Items

Out-of-scope discoveries found while executing phase 191.1 plans. Not fixed here per the
executor's scope boundary (only auto-fix issues directly caused by the current task's own
file changes).

## 191.1-01: `TestPackedNPMReleaseCandidateContract` fails on a pre-existing version mismatch

- **Status:** acknowledged
- **Deferred at:** v1.26 close, 2026-08-22 — carried per .planning/research/v1.27-milestone-brief.md

- **Found during:** Task 2 verification (`go test ./cmd/... -count=1`).
- **Symptom:** `npm version "1.0.59" does not match source version "1.0.60"` (release_candidate_blackbox_test.go:51).
- **Cause:** `npm/package.json` was last bumped to 1.0.59 in commit `86e967f4` ("release: bump
  version to v1.0.59 and sync docs"). Something else in the source tree now reports 1.0.60,
  and the npm package version was never bumped to match. This is release/version bookkeeping
  drift, unrelated to any file this plan touches
  (`cmd/build_finalize_deadlock_test.go`, `cmd/codex_build_finalize.go`, `cmd/codex_build.go`,
  `.aether/schemas/completion-packet.schema.json`, `cmd/wrapper_bundled_completion_test.go`).
- **Confirmed pre-existing:** reproduces identically before and after this plan's changes
  (verified by stashing the plan's production diff and re-running the test in isolation --
  the failure message and line number are unchanged either way).
- **Not fixed:** out of scope for this plan; requires a version bump in `npm/package.json`
  and possibly a release/publish pass, which is an orchestrator/owner decision, not a
  field-hardening test fix.
