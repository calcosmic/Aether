# Phase 155-02 Summary: Go Boundary Refactor — Policy Extraction

## Objective
Extract structured policy config from `cmd/codex_dispatch_contract.go` and `cmd/review_depth.go` into YAML files under `colony/policies/`, make Go load them at runtime with fallback to hardcoded defaults, and add tests.

## Deliverables

### New Files
- `cmd/policy_loader.go` — Shared YAML policy loading utilities (`loadYAMLPolicy`, `policyPath`, `cachedPolicyLoad`, in-memory cache with `sync.RWMutex`).
- `colony/policies/review-depth.yaml` — Keyword tables (`heavy_keywords`, `security_risk_keywords`, `blast_radius_keywords`) and smart-default reason strings (`smart_default_reasons`).
- `cmd/dispatch_contract_test.go` — 5 tests covering custom policy load, no-file fallback, partial fallback, timeout defaults accessibility, and extended planning fields.
- `cmd/review_depth_test.go` — 4 new tests appended to existing file covering custom heavy keywords, no-file fallback, custom smart-default reasons, and custom keyword usage in `phaseHasHeavyKeywords`.

### Modified Files
- `colony/policies/dispatch-contract.yaml` — Extended with string policy fields: `execution_models`, `deadline_policies`, `dependency_behaviors`, `fallback_behaviors`, `fallback_visibility`, `result_collection_policies`. Existing numeric config preserved.
- `cmd/codex_dispatch_contract.go` — Added `dispatchContractPolicy` struct and wrapper, `loadDispatchContractPolicy()` with `sync.Once`, package-level fallback constants/vars. Refactored `surveyDispatchContractWithTimeout`, `planningDispatchContractWithTimeout`, and `planningDispatchContractForDispatches` to load from policy first, fallback to hardcoded strings.
- `cmd/review_depth.go` — Added `reviewDepthPolicy` struct, `loadReviewDepthPolicy()` with `sync.Once`, package-level fallback slices. Refactored `phaseHasHeavyKeywords` to use `getHeavyKeywords()`, `phaseRiskLevel` to use `getSecurityRiskKeywords()` and `getBlastRadiusKeywords()`. Refactored `renderSmartDepthReason` in `cmd/codex_visuals.go` to use `getSmartDefaultReason()`.

## Build and Test Results
- `go build ./cmd/aether` — success
- `go test ./cmd/... -count=1` — all tests pass (took ~88s)
- `aether version` — reports `1.0.41`

## Coverage
- Dispatch contract: file load, missing file fallback, partial fallback, extended planning fields, timeout defaults.
- Review depth: file load with custom keywords, missing file fallback, custom smart-default reasons, keyword integration in `phaseHasHeavyKeywords`.

## Threat Model Acknowledgment
- T-155-04 / T-155-05 (Tampering): Mitigated by read-only loading and hardcoded fallback on parse error.
- T-155-06 (DoS via malformed YAML): Mitigated by fallback to hardcoded defaults on any parse error; no panic.

## Blockers
None.
