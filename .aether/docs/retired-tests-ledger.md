# Retired Tests Ledger

Updated: 2026-07-27

This file records every test file removed from this repository during the
v1.25 "Switch It On" milestone. The rule it enforces (RETIRE-04): no test
leaves the repository without a recorded disposition — either its coverage
survived in a named replacement test, or its loss is knowingly accepted and
written down, never silently dropped.

Each entry below carries four labeled fields: original path (repo-relative
path of the deleted file), what it covered (the concrete behaviour the test
asserted), disposition (exactly one of `dead-with-no-replacement` or
`recovered-by:<path-to-surviving-test>`), and removed in (the commit or
phase/plan that removed it).

### `control-ts/tests/schemas/policy.schema.test.ts`

- **Original path:** `control-ts/tests/schemas/policy.schema.test.ts`
- **What it covered:** Zod schema validation of the policy YAML field
  surface — `model_routing.default_provider`, `memory_rules.*`,
  `skill_creation.*`, `safety_gates.*`, `dispatch_contract.*`,
  `pheromone_lifecycle.*`, `signal_rules.*`, `autopilot.*`.
- **Disposition:** `recovered-by:cmd/policy_schema_test.go`. The Go
  replacement is a strict improvement in scope, not just language: it reads
  the live `colony/policies/*.yaml` files directly, while the retired TS
  test read fixture copies under `control-ts/tests/fixtures/policies/`
  that could silently drift from what the runtime actually loads.
- **Removed in:** Phase 160 Plan 06 (`control-ts/` deletion).

### `.aether/ts-host/test/playbook-loader.test.ts`

- **Original path:** `.aether/ts-host/test/playbook-loader.test.ts`
- **What it covered:** The TS host's playbook loader — loading
  `.aether/docs/command-playbooks/*.md` and injecting them into worker
  prompts.
- **Disposition:** `dead-with-no-replacement`. This is a retroactive entry:
  the feature it tested was deliberately removed, not relocated. Since
  v1.25, build/continue execution behavior lives in the host-manifest flow
  (`aether host build` → `dispatch_manifest` → `build-finalize`) instead of
  playbook injection, so there is no successor test to name.
- **Removed in:** commit `b2b41486`.
