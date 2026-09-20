# Deferred Items — Phase 205

## 205-03: `TestEveryLifecycleCommandEndsWithNextAction` fails only in the scoped regression combination, not in isolation

**Found during:** Plan 205-03, Task 3's regression scope check
(`go test ./cmd/ -run 'Entomb|Seal|Resume|SessionFlow|Lifecycle' -count=1 -timeout 90m`).

**Symptom:** `TestEveryLifecycleCommandEndsWithNextAction/updating` fails with
`install failed: failed to validate repository store: storage: repository
containment refused: data root path must not be empty`.

**Scope determination:** Reproduced against a fully reverted copy of this
plan's source changes (`cmd/entomb_cmd.go`, `cmd/entomb_manifest.go`,
`cmd/session_flow_cmds.go`, `cmd/seal_confirmation.go`, and the two updated
pre-existing tests) using the identical `-run` pattern — the failure occurs
identically before and after this plan's changes. Run in isolation
(`-run TestEveryLifecycleCommandEndsWithNextAction` alone) it passes. This is
test-order/pollution flakiness specific to this exact scoped test
combination, not a regression introduced by plan 205-03, and it is not the
current file this plan touches (`cmd/lifecycle_next_action_coverage_test.go`).

**Not in the static 2026-09-14 known-red baseline** (17 entries recorded in
project memory) — so it is recorded here explicitly rather than silently
waved through as "baseline."

**Action:** Not fixed — out of 205-03's scope per the scope-boundary
deviation rule (pre-existing failure in an unrelated file). Deferred to
whichever later 205 plan or phase owns test-suite pollution/isolation
hardening, or to the full-suite gate plan (205-12 per this phase's plan
index) if it recurs there.

**Verification of the classification:** see
`.planning/phases/205-owner-acceptance-and-restoration-seal/205-03-SUMMARY.md`
"Issues Encountered".
