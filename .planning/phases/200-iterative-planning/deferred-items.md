# Deferred Items

## 2026-09-07 — Phase 199 vocabulary inventory drift

- **Discovered during:** Phase 200 Plan 01 repository-wide verification (`go test ./...`)
- **Failing test:** `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified`
- **Observed mismatch:** `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` contains one `legacy_pause` and one `legacy_resume` occurrence that the Phase 199 vocabulary inventory classifies with count zero.
- **Why deferred:** The failure is confined to Phase 199 documentation/inventory and is unrelated to the Phase 200 `pkg/colony` specification and planning-domain changes. The scoped Phase 200 package tests and vet gate pass.
- **Suggested follow-up:** Reconcile the two UAT occurrences with the exhaustive Phase 199 vocabulary inventory, then rerun `go test ./cmd -run TestCurrentVocabulary199 -count=1`.

## 2026-09-07 — Phase 199 gate receipt timestamp staleness

- **Discovered during:** Phase 200 Plan 02 command-package verification (`go test ./cmd -count=1`)
- **Failing tests:** `TestPhase199GateReceiptSchema` and `TestPhase199GateReceipt`
- **Observed mismatch:** The checked-in Phase 199 receipt now reports stale or invalid execution timestamps for its `go test ./...` gate.
- **Why deferred:** The receipt is Phase 199 historical verification metadata; refreshing or redesigning its time window is unrelated to Phase 200 planning-state validation and migration.
- **Suggested follow-up:** Re-run the Phase 199 gate-receipt workflow with current evidence, then verify `go test ./cmd -run 'TestPhase199GateReceiptSchema|TestPhase199GateReceipt' -count=1`.
