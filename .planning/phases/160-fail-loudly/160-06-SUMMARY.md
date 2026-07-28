# Plan 160-06 — Summary

**Status:** Complete (3/3 tasks)
**Requirements:** RETIRE-01, RETIRE-02, RETIRE-03, RETIRE-04
**Completed:** 2026-07-27

> **Execution note:** run inline by the orchestrator rather than a worktree executor. This
> was deliberate: the plan deletes 130 tracked files, and the standard worktree merge-back
> path has a guard that blocks any branch containing deletions. Isolating it would have
> guaranteed a stuck merge.

## What changed

**Task 1 — deletion** (`9077df26`)
`control-ts/` removed: 130 tracked files, 81 MB. It self-described in its own `package.json`
as a retired prototype control plane. Committed standalone, as the ROADMAP required.

**Before deleting, inbound references were mapped:**

| Surface | Result |
|---|---|
| Go source | none (only two past-tense comments in `cmd/policy_schema_test.go`) |
| `//go:embed` directives | none — embeds reference `.aether/ts-host/dist` and `.aether/ts/…` only |
| CI / goreleaser / Makefile / package.json | none |
| Docs | historical audit records under `docs/audits/` — left intact deliberately; they are a record of what was true then |

**Task 2 — RETIRE-02/03 verified, not assumed**
`.aether/ts-host/` and `.aether/ts/` confirmed byte-identical across the deletion commit by
comparing their git tree hashes before and after. Both remain embedded via `//go:embed` in
**repo-root `embedded_assets.go`** — note the path: CONTEXT.md's canonical refs cited
`cmd/embedded_assets.go`, which does not exist. Research caught this; it is corrected here
so a future `grep cmd/embedded_assets.go` doesn't come up empty and conclude nothing
references these trees.

**Task 3 — `cmd/retired_packages_test.go`**

`TestRetiredPackagesStayRetired` — three subtests:
- `control-ts/` does not exist (RETIRE-01)
- `.aether/ts-host/` and `.aether/ts/` do exist (RETIRE-02/03)
- `embedded_assets.go` still embeds both trees — so "nothing references it" can never be
  concluded from a grep that missed the embed line, which is the exact reasoning error that
  would delete a load-bearing directory

`TestRetiredTestsLedgerDispositionsAreHonest` — makes the ledger a contract rather than a
document. Every entry must carry exactly one disposition, and every `recovered-by:<path>`
must point at a file that exists. **A `recovered-by` pointing nowhere is a coverage hole
wearing the label of a recovery** — which is precisely the failure mode RETIRE-04 exists to
prevent.

## Deliberate-regression proof

Ledger entry rewritten to `recovered-by:cmd/does_not_exist_test.go`, then reverted:

```
--- FAIL: TestRetiredTestsLedgerDispositionsAreHonest
    ledger claims coverage was recovered by "cmd/does_not_exist_test.go", but that file
    does not exist — a recovered-by pointing nowhere is a coverage hole labelled as a recovery
```

## Verification — ROADMAP success criterion #6

All four required commands run **after** the deletion:

```
go build ./cmd/aether     → OK
go vet ./...              → clean
go test ./... -count=1    → exit 0 (18 packages)
aether integrity          → 5/5 checks passed
```

`aether integrity` covering source version, binary version, hub version, hub companion
files, and downstream simulation is the strongest single signal that the deletion took
nothing load-bearing with it.

## Files

- `control-ts/` (deleted — 130 tracked files, 81 MB)
- `cmd/retired_packages_test.go` (created — 2 tests, 4 assertions)
- `.aether/docs/retired-tests-ledger.md` (already complete from Plan 01; both entries verified by the new test)

## Commits

- `9077df26` chore(160-06): delete control-ts, the retired prototype control plane
- (this commit) test(160-06): pin control-ts retired and ts-host/ts kept
