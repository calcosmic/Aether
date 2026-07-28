---
phase: 160-fail-loudly
plan: 01
subsystem: testing / policy validation
tags: [go-test, yaml, retire-04, safety-net]
dependency-graph:
  requires: []
  provides:
    - "cmd/policy_schema_test.go: TestPolicySchemaRequiredFields (Go replacement for control-ts's policy schema validation)"
    - ".aether/docs/retired-tests-ledger.md: deleted-test disposition ledger"
  affects:
    - "160-06 (control-ts deletion) — this plan is its prerequisite safety net"
tech-stack:
  added: []
  patterns:
    - "table-driven required-field checks over map[string]interface{} decoded YAML, with a dotted-path lookup helper"
key-files:
  created:
    - cmd/policy_schema_test.go
    - .aether/docs/retired-tests-ledger.md
  modified: []
decisions:
  - "Read live colony/policies/*.yaml directly rather than porting the TS test's fixture-file reads — a strict improvement, not a like-for-like port"
  - "Table-driven checks keyed by {file, dottedPath, kind, want} so a future required field is one row, not a new code block"
metrics:
  duration: "~35 min"
  completed: "2026-07-27"
---

# Phase 160 Plan 01: Fail-Loudly Safety Net for Policy Schema Summary

Go test `TestPolicySchemaRequiredFields` ports the field-presence and shallow-type
checks of `control-ts/tests/schemas/policy.schema.test.ts` onto the live
`colony/policies/*.yaml` files, and a new retired-tests ledger records the
disposition of that TS test plus one retroactive entry, so Plan 06's deletion of
`control-ts/` cannot silently drop coverage.

## What Was Built

### Task 1: `cmd/policy_schema_test.go`

A table-driven test (`policySchemaChecks []policyFieldCheck`) asserts presence and
type of every required field across the eight live policy files under
`colony/policies/`: `model-routing.yaml`, `memory-rules.yaml`,
`skill-creation.yaml`, `safety-gates.yaml`, `dispatch-contract.yaml`,
`pheromone-lifecycle.yaml`, `signal-rules.yaml`, `autopilot.yaml`. Each row is
`{file, dottedPath, kind, want}`; a dotted-path lookup helper
(`lookupPolicyDottedPath`) walks the decoded `map[string]interface{}` document.
Kinds supported: `string`, `bool`, `numeric`, `map` (non-empty), `listContains`
(all wanted entries present in a list). Failure messages name the file and exact
dotted path, e.g. `colony/policies/safety-gates.yaml: safety_gates.security_scan
missing or not a bool`.

One implementation detail not anticipated by the plan text: `yaml.v3` decodes a
YAML mapping into `interface{}` as `map[string]interface{}` only when every key
is a string. `dispatch_contract.spawn_depth_limits` has bare-integer YAML keys
(`0`, `1`, `2`, `3`), so it decodes to `map[interface{}]interface{}` instead. The
`"map"` check (`isPolicyNonEmptyMap`) accepts both map kinds — documented inline
in the helper's doc comment so a future reader doesn't reintroduce the narrower
check and reopen the same false failure.

### Task 2: `.aether/docs/retired-tests-ledger.md`

Seeded with exactly two `###` entries matching `known-issues.md`'s field-label
style:
1. `control-ts/tests/schemas/policy.schema.test.ts` — `recovered-by:cmd/policy_schema_test.go`
2. `.aether/ts-host/test/playbook-loader.test.ts` — `dead-with-no-replacement`, retroactive, removed in commit `b2b41486`

## Divergence Finding (required by the plan's Output section)

The plan asked for a check of whether `control-ts/tests/fixtures/policies/*.yaml`
(what the TS test actually read) had drifted from the live
`colony/policies/*.yaml` files (what the new Go test reads). All eight fixture
files were diffed against their live counterparts:

- Seven files (`autopilot.yaml`, `memory-rules.yaml`, `model-routing.yaml`,
  `pheromone-lifecycle.yaml`, `safety-gates.yaml`, `signal-rules.yaml`,
  `skill-creation.yaml`) are byte-for-byte identical between fixture and live.
- `dispatch-contract.yaml` differs: the live file is a strict superset. It adds
  `execution_models`, `deadline_policies`, `dependency_behaviors`,
  `fallback_behaviors`, `fallback_visibility`, and `result_collection_policies`
  on top of the fields the fixture (and the TS test) covered. None of the
  fields this plan's required-field surface checks (`max_workers_per_phase`,
  `spawn_depth_limits`, `timeout_defaults`) are affected — the extra keys are
  additive, not divergent.

No drift was found that would change what "required field" means for this test.
This is recorded in the test's doc comment as well as here, per the plan's
instruction to note divergence findings for Phase 161.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `map` check kind failed on `spawn_depth_limits`'s integer-keyed YAML map**
- **Found during:** Task 1, first test run
- **Issue:** The initial `"map"` check only accepted `map[string]interface{}`.
  `dispatch_contract.spawn_depth_limits` has integer YAML keys (`0:`, `1:`, ...),
  which `yaml.v3` decodes to `map[interface{}]interface{}` when unmarshaled into
  `interface{}`, so the check reported the field as missing.
- **Fix:** Added `isPolicyNonEmptyMap`, accepting both `map[string]interface{}`
  and `map[interface{}]interface{}`, with a doc comment explaining the `yaml.v3`
  behavior so it isn't "fixed" back to the narrower check later.
- **Files modified:** `cmd/policy_schema_test.go`
- **Commit:** `90dd4605`

**2. [Rule 1 - Bug] Doc comment accidentally matched the acceptance criterion's forbidden substring**
- **Found during:** Task 1, acceptance-criteria check
- **Issue:** The doc comment explaining that the test reads live files (not
  fixtures) literally contained the string `control-ts/tests/fixtures/policies/`,
  which made `grep -c 'control-ts/tests/fixtures' cmd/policy_schema_test.go`
  return 1 instead of the required 0.
- **Fix:** Reworded the comment to describe the fixture location without using
  that literal path substring.
- **Files modified:** `cmd/policy_schema_test.go`
- **Commit:** `90dd4605`

**3. [Rule 1 - Bug] Ledger intro paragraph inflated the `**Disposition:**` grep count**
- **Found during:** Task 2, acceptance-criteria check
- **Issue:** The ledger's intro paragraph described the four field labels using
  the same bold `**Disposition:**` markdown, so
  `grep -c '\*\*Disposition:\*\*' .aether/docs/retired-tests-ledger.md` returned
  3 instead of the required 2 (matching both entries).
- **Fix:** Rewrote the intro to describe the field names in prose without
  reusing the exact bolded label syntax.
- **Files modified:** `.aether/docs/retired-tests-ledger.md`
- **Commit:** `e9f445fe`

## Manual Verification Performed

Per the plan's acceptance criteria, `security_scan: true` was manually deleted
from `colony/policies/safety-gates.yaml`, confirming
`go test ./cmd -run TestPolicySchemaRequiredFields -count=1` exits non-zero with
message `colony/policies/safety-gates.yaml: safety_gates.security_scan missing
or not a bool`. The line was restored immediately afterward;
`git diff --stat colony/policies/safety-gates.yaml` confirms zero uncommitted
changes to that file.

## Known Stubs

None.

## Threat Flags

None. This plan's only new surface (`cmd/policy_schema_test.go`) reads
repository-controlled YAML during `go test`, matching the threat model's stated
trust boundary (repo files → test process) with no new externally-reachable
code path.

## Self-Check: PASSED

- `cmd/policy_schema_test.go` — FOUND
- `.aether/docs/retired-tests-ledger.md` — FOUND
- Commit `90dd4605` — FOUND
- Commit `e9f445fe` — FOUND
