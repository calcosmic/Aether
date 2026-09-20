# Phase 151 Research: End-to-End Proof

## Domain Context

This is the final phase of v1.23 Daily Driver Reliability. The goal is to prove Aether works as a tool, not just as a codebase. All prior phases fixed foundations; this phase validates the user experience.

## Key Sources

- `cmd/e2e_v113_test.go` — existing e2e test patterns
- `cmd/init_cmd_test.go` — init test patterns
- `.aether/ts-host/src/queen/orchestrator.ts` — TypeScript host dispatch
- `cmd/command_guide.go` — command catalog (for PROOF-03)
- `cmd/testing_main_test.go` — test helpers

## Known Gaps

### E2E Lifecycle (PROOF-01)
No existing test creates a fresh repo and runs the full lifecycle. The closest is `e2e_v113_test.go` which tests a subset. Need:
- Downstream repo creation helper
- Aether installation into temp repo
- Full 7-step lifecycle execution
- State validation at each step

### TS Host E2E (PROOF-02)
The TS host (`aether-dev`) has tests but not comprehensive e2e for all 5 lifecycle commands. Need:
- Mock AI provider layer
- Manifest schema validation
- Go runtime roundtrip tests

### Public Utility Tests (PROOF-03)
Many utility commands lack dedicated tests:
- pause-colony, resume-colony — session management
- data-clean — artifact cleanup
- insert-phase — plan mutation
- quick — agent spawning (needs mock)
- preferences, verify-castes, bump-version, maturity — simple output tests

## Decision Log

- **Decision 1:** Use Go tests for PROOF-01 and PROOF-03, TS tests for PROOF-02. Don't mix languages in a single test file.
- **Decision 2:** Mock the worker invoker for `quick` and build/continue tests to avoid requiring real AI credentials.
- **Decision 3:** Use `--dry-run` flags where available (bump-version) to avoid side effects.
- **Decision 4:** Temp dirs with `t.Cleanup` for all tests — no persistent test artifacts.
