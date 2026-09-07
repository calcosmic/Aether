# Deferred Items

## 2026-09-07 — Phase 199 vocabulary inventory drift

- **Discovered during:** Phase 200 Plan 01 repository-wide verification (`go test ./...`)
- **Failing test:** `TestCurrentVocabulary199/tracked-occurrences-are-exhaustively-classified`
- **Observed mismatch:** `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` contains one `legacy_pause` and one `legacy_resume` occurrence that the Phase 199 vocabulary inventory classifies with count zero.
- **Why deferred:** The failure is confined to Phase 199 documentation/inventory and is unrelated to the Phase 200 `pkg/colony` specification and planning-domain changes. The scoped Phase 200 package tests and vet gate pass.
- **Suggested follow-up:** Reconcile the two UAT occurrences with the exhaustive Phase 199 vocabulary inventory, then rerun `go test ./cmd -run TestCurrentVocabulary199 -count=1`.

