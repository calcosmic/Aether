---
phase: 109
slug: typescript-orchestration-host-prototype
created: 2026-05-12
---

# Validation Strategy: Phase 109 - TypeScript Orchestration Host Prototype

## Test Framework

| Property | Values |
|----------|--------|
| Framework | tsx --test (Node built-in test runner via tsx) |
| Config file | None -- tsx --test is zero-config |
| Quick run | `cd .aether/ts-host && npx tsx --test test/*.test.ts` |
| Full suite | `cd .aether/ts-host && npx tsx --test test/*.test.ts && cd ../.. && go test ./cmd/... -run TestTsHost -count=1` |

## Requirement-to-Test Mapping

| Req ID | Behavior | Test Type | Automated Command | Plan |
|--------|----------|-----------|-------------------|------|
| HOST-01 | Host entry point runs as Node script | unit | `npx tsx --test test/host.test.ts` | 109-01 |
| HOST-02 | Host calls plan-only and gets JSON manifest | integration | `npx tsx --test test/go-bridge.test.ts` | 109-01 |
| HOST-03 | Host dispatches workers with spawn-log/complete | integration | `npx tsx --test test/worker-dispatch.test.ts` | 109-02 |
| HOST-04 | Host calls finalizers and state changes | integration | `npx tsx --test test/lifecycle.test.ts` | 109-03 |
| HOST-05 | Host never writes .aether/data/ directly | unit | `npx tsx --test test/boundary.test.ts` | 109-03 |
| HOST-06 | Spawn lifecycle events recorded via Go CLI | integration | `npx tsx --test test/worker-dispatch.test.ts` | 109-02 |
| HOST-07 | Full lifecycle completes or documents blocker | e2e | `npx tsx --test test/lifecycle.test.ts` | 109-03 |

## Sampling Rate

- **Per task commit:** `cd .aether/ts-host && npx tsx --test test/*.test.ts`
- **Per wave merge:** `cd .aether/ts-host && npx tsx --test test/*.test.ts && cd ../.. && go test ./cmd/... -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

## Test Files

| File | Covers | Plan |
|------|--------|------|
| `test/host.test.ts` | HOST-01 entry point | 109-01 |
| `test/go-bridge.test.ts` | HOST-02 Go subprocess calls | 109-01 |
| `test/worker-dispatch.test.ts` | HOST-03, HOST-06 spawn lifecycle | 109-02 |
| `test/lifecycle.test.ts` | HOST-04, HOST-07 full lifecycle | 109-03 |
| `test/boundary.test.ts` | HOST-05 no .aether/data writes | 109-03 |

## Verification Dimensions

1. **Compilation:** TypeScript compiles with strict mode (`npx tsc --noEmit`)
2. **Unit tests:** Go bridge, host entry point, boundary enforcement
3. **Integration tests:** Worker dispatch with real Go CLI spawn-log/complete
4. **E2E tests:** Full plan -> build -> continue lifecycle with state assertions
5. **Boundary enforcement:** grep confirms no .aether/data/ writes in TS source
6. **Go regression:** Existing Go test suite passes with TS host present

---

*Phase: 109-typescript-orchestration-host-prototype*
*Created: 2026-05-12*
