---
phase: 199-front-door-and-classic-contract
plan: "18"
subsystem: maintenance-transactions
tags: [go, cobra, maintenance, transactions, receipts, rollback, update, migration, registry, stale-preflight, executable-mode]

requires:
  - phase: 199-front-door-and-classic-contract
    plan: "05"
    provides: Shared multi-root lifecycle coordinator, journal, rollback, replay, and typed receipt grammar
  - phase: 199-front-door-and-classic-contract
    plan: "06"
    provides: Expert maintenance catalog and inspection/mutation separation
  - phase: 199-front-door-and-classic-contract
    plan: "09"
    provides: Authoritative lifecycle projection and state-effect vocabulary
  - phase: 199-front-door-and-classic-contract
    plan: "14"
    provides: Transaction recovery and replay precedent
  - phase: 199-front-door-and-classic-contract
    plan: "16"
    provides: Multi-root archive transaction and exact manifest precedent
provides:
  - One schema-versioned preview/transaction/receipt grammar for update, migration, generated/platform sync, and binary replacement
  - Exact owner-and-digest cleanup manifests with confirmed transactional deletion and rollback proof
  - Stable-identity, baseline-closed registry upsert/finalize/remove operations with atomic hub replacement
  - Honest rolled-back or recovery-required results for in-process maintenance interruption after partial commit
  - Pre-mutation stale-publish refusal, typed no-change validation failures, and journaled executable-mode verification
affects: [plan-199-19, plan-199-25, plan-199-29, update, migrate-state, data-clean, registry]

tech-stack:
  added: []
  patterns: [closed typed target set, read-only maintenance preview, expected-baseline closure, coordinator-owned rollback receipt]

key-files:
  created:
    - cmd/maintenance_mutation_199_test.go
    - cmd/maintenance_state_199_test.go
  modified:
    - cmd/lifecycle_transaction.go
    - cmd/e2e_stale_publish_test.go
    - cmd/update_cmd.go
    - cmd/command_truth.go
    - cmd/platform_sync.go
    - cmd/binary_download.go
    - cmd/maintenance.go
    - cmd/registry.go
    - .planning/phases/199-front-door-and-classic-contract/deferred-items.md

key-decisions:
  - "A maintenance preview closes over exact typed targets and expected baseline digests; commit never performs a second broad discovery pass."
  - "Cleanup prefixes, globs, and content guesses are not deletion authority; a versioned aether-runtime manifest with an exact path, owner, and current digest is required."
  - "Registry identity is stableRepoIdentity(canonical path), and duplicate identity/path or active-plus-final-stats conflicts fail before any write."
  - "A live maintenance interruption after durable intent attempts coordinator rollback immediately and returns its rollback receipt; an unprovable rollback retains recovery-required evidence."
  - "Update resolves its repository/module root from lifecycle state, refuses critical stale input before target discovery or download, and journals binary permission bits with the bytes."

patterns-established:
  - "Maintenance mutation: prepare a read-only typed target preview, declare the same closed set to LifecycleTransaction, then return its durable receipt and verification."
  - "Destructive maintenance authority: require schema, owner, checkpoint, contained non-symlink path, and exact baseline digest before confirmation can commit."
  - "Cross-root update order: repository/hub companions, managed platform homes, then verified binary destination under one coordinator transaction."

requirements-completed: [CEC-04, LIFE-06]

duration: 1h 20m
completed: 2026-09-04
---

# Phase 199 Plan 18: Maintenance Mutation Transactions Summary

**Update, migration, generated/platform sync, binary replacement, cleanup, and registry writes now close over exact validated targets and finish with a shared durable receipt, rollback proof, or named recovery action.**

## Performance

- **Duration:** 1 hour 20 minutes
- **Started:** 2026-09-04T15:38:10Z
- **Completed:** 2026-09-04T16:58:14Z
- **Tasks:** 2/2
- **Files changed:** 8 implementation/test files plus the shared deferred-items record

## Accomplishments

- Replaced public update's sequence of direct sync/write/prune operations with a read-only exact target plan and one multi-root lifecycle transaction spanning the repository, selected hub, managed Claude/OpenCode/Codex homes, and optional binary destination.
- Made state migration and rollback schema-versioned, backup-explicit, previewable, and receipted; migration and safety-backup bytes participate in the same transaction.
- Staged and verified downloaded binaries before declaring their exact destination, enforced source/npm/hub version agreement, isolated dev from stable platform homes, and preserved unmanaged/custom files during generated pruning.
- Replaced public data-clean's prefix/content deletion guesses with a versioned one-shot manifest containing exact owner, path, checkpoint, and baseline digest proof; preview is zero-write and `--confirm` is still mandatory.
- Reworked registry add, lifecycle upsert/finalize, and explicit remove through stable repository identity, strict baseline and duplicate validation, one atomic hub replacement, and the shared receipt grammar.
- Closed the final prepare-to-declare baseline window and made live partial-commit faults roll back through coordinator pre-images instead of reporting an ambiguous state effect.
- Gate recovery moved stale-publish classification ahead of platform discovery, staging, and binary download; critical stale now exits nonzero with a typed `none` result, no receipt, and recovery guidance, while warning/info use the existing visual banner.
- Gate recovery also normalizes update to the lifecycle/module root, preserves platform detail labels and restart guidance, carries executable modes in lifecycle manifests and rollback baselines, and verifies staged plus installed binary versions.

## Task Commits

Each TDD task was committed atomically:

1. **Task 1: Transact update, migration, and generated-surface sync**
   - `eece33d1` — test(199-18): add failing maintenance mutation contract (RED)
   - `8490a55a` — feat(199-18): transact maintenance update surfaces (GREEN)
2. **Task 2: Bound cleanup and registry mutations by exact ownership**
   - `880c343c` — test(199-18): add failing maintenance state contracts (RED)
   - `abe4173d` — feat(199-18): bound cleanup and registry mutations (GREEN)

**Plan metadata:** committed with this summary.

### Wave 12 Gate Recovery

- `3cd1c41f` — test(199-18): expose maintenance gate regressions (RED)
- `f505703d` — test(199-18): require source subdirectory normalization (RED)
- `27493c2f` — test(199-18): prove executable mode journal closure (RED)
- `d7781770` — test(199-18): require rollback to restore file mode (RED)
- `df20b816` — fix(199-18): close maintenance update transactions (GREEN)

## Files Created/Modified

- `cmd/maintenance_mutation_199_test.go` — ten-case update/migration/generated/platform/binary preview, rollback, replay, channel, version, and custom-preservation contract.
- `cmd/maintenance_state_199_test.go` — cleanup ownership/rollback and registry baseline/receipt tests, including partial-commit fault injection.
- `cmd/platform_sync.go` — common typed maintenance plan/preview/result, exact sync target builder, expected-baseline closure, transaction commit, rollback, and recovery adaptation.
- `cmd/update_cmd.go` — public update target assembly, parity/version/channel checks, optional platform-home and binary targets, dry-run rendering, and receipted commit.
- `cmd/command_truth.go` — transaction-backed state migration and rollback with exact backup targets.
- `cmd/binary_download.go` — isolated download/checksum/version staging followed by exact binary-destination transaction replacement.
- `cmd/lifecycle_transaction.go` — durable before/desired file-mode evidence, mode-aware apply/replay/receipt verification, and exact rollback-mode restoration.
- `cmd/e2e_stale_publish_test.go` — critical zero-write/download guard, typed early refusal, valid installed-root fixtures, and critical/warning/info visual proof.
- `cmd/maintenance.go` — exact cleanup manifest grammar, zero-write preview, confirmation gate, transaction deletion, and typed command result.
- `cmd/registry.go` — stable `repo_id`, strict baseline/conflict validation, typed upsert/finalize/remove plans, and atomic hub registry receipt.
- `.planning/phases/199-front-door-and-classic-contract/deferred-items.md` — recorded the superseded prefix-only cleanup fixture and contradictory legacy update-card setup.

## Decisions Made

- Preview and commit share one closed typed target list. Source enumeration may suggest targets, but ownership, containment, action, current digest, desired digest, and commit order are fixed before staging.
- Stable update may write verified managed platform artifacts; dev update refuses stable platform-home targets. Binary bytes are downloaded and verified before the destination enters the transaction.
- Public data-clean accepts only the default or explicitly selected manifest below the lifecycle data root. The manifest is consumed in the same transaction, preventing stale authorization from being silently reused.
- Registry mutations migrate legacy entries to stable repository IDs only as part of a validated requested replacement. Baseline mismatch, identity mismatch, duplicate path/identity, and active entries with final stats mutate nothing.
- Existing direct sync/prune helpers remain only as compatibility kernels for callers/tests that have not yet migrated; the public Plan 199-18 update, data-clean, migration, binary, and registry handlers do not call them.
- Critical stale-publish input is a preflight refusal, not a degraded update mode. It runs before platform discovery and download, and all early update validation failures share one typed `state_effect: none` result.
- `--sync-platform-homes` is documented truthfully as stable-only for `update`; dev update continues to refuse writes into stable Claude/OpenCode/Codex homes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Refused implicit binary destination directory creation**

- **Found during:** Task 1 (binary replacement wiring)
- **Issue:** Creating a missing destination parent before the lifecycle transaction would leave an unjournaled durable side effect and weaken exact-root validation.
- **Fix:** Binary download stages bytes in an isolated temporary directory, but the configured durable destination directory must already exist before preview/commit.
- **Files modified:** `cmd/binary_download.go`, `cmd/update_cmd.go`
- **Verification:** `TestMaintenanceMutation199DownloadBinary`, update dry-run, and race-enabled combined Plan 199-18 suite pass.
- **Committed in:** `8490a55a`

**2. [Rule 1 - Bug] Removed ambiguous state effects after a target-commit interruption**

- **Found during:** Task 2 cleanup/registry fault proof
- **Issue:** The shared coordinator intentionally preserves crash evidence, but the first maintenance adapter could return a live in-process fault after bytes changed with the progress record still reading `none` or `committed` and no durable receipt.
- **Fix:** When durable intent exists and no receipt was written, the maintenance adapter now attempts coordinator rollback immediately; success returns a rolled-back receipt, while rollback failure returns a recovery-required receipt and safe next action.
- **Files modified:** `cmd/platform_sync.go`, `cmd/maintenance_state_199_test.go`
- **Verification:** Cleanup and registry target-commit fault tests restore byte-identical pre-images and report `rolled_back`; all 14 exact tests pass normally and with `-race`.
- **Committed in:** `abe4173d`

**3. [Rule 1 - Bug] Restored stale-publish preflight and source-root normalization**

- **Found during:** Wave 12 full-gate audit after Plan 199-18
- **Issue:** Update used the invocation directory for source manifests and classified stale input only after planning/commit, so invocation from `cmd/` failed and a critical stale hub could reach platform validation or a binary download.
- **Fix:** Resolve the repository from the lifecycle store, normalize real source checkouts to the module root, and stop critical stale input before target discovery, staging, download, or transaction evidence.
- **Files modified:** `cmd/update_cmd.go`, `cmd/e2e_stale_publish_test.go`, `cmd/maintenance_mutation_199_test.go`
- **Committed in:** `df20b816`

**4. [Rule 2 - Missing Critical] Unified early update failures under typed no-change output**

- **Found during:** Wave 12 full-gate audit after Plan 199-18
- **Issue:** Hub, version, source, platform, binary, and prepare failures returned raw errors without stating whether durable state changed or how to recover.
- **Fix:** Every pre-commit update refusal now emits operation, no-change outcome, empty transaction/receipt, `state_effect: none`, verification, error, and a safe recovery action.
- **Files modified:** `cmd/update_cmd.go`
- **Committed in:** `df20b816`

**5. [Rule 1 - Bug] Journaled and verified executable binary mode**

- **Found during:** Wave 12 full-gate audit after Plan 199-18
- **Issue:** New binary targets inherited the transaction default `0644`; executable permission was neither journaled nor globally verified, and rollback could not distinguish before/after modes.
- **Fix:** Add explicit mode declarations to the shared lifecycle transaction, include baseline/desired mode in preview and root manifests, verify mode during apply/replay/receipt loading, restore the exact prior mode on rollback, and execute `version` both in isolated staging and after install.
- **Files modified:** `cmd/lifecycle_transaction.go`, `cmd/platform_sync.go`, `cmd/binary_download.go`, `cmd/update_cmd.go`, `cmd/maintenance_mutation_199_test.go`
- **Committed in:** `df20b816`

**6. [Rule 1 - Bug] Preserved platform labels and corrected dev flag guidance**

- **Found during:** Wave 12 full-gate audit after Plan 199-18
- **Issue:** Transaction-root grouping discarded `Agents (codex)` and other public labels, suppressing restart guidance; update help promised a dev opt-in that the locked channel-isolation contract correctly refused.
- **Fix:** Carry labels through targets and previews, group refresh details by those labels, and describe `--sync-platform-homes` as stable-only while retaining dev refusal.
- **Files modified:** `cmd/platform_sync.go`, `cmd/update_cmd.go`, `cmd/maintenance_mutation_199_test.go`
- **Committed in:** `df20b816`

---

**Total deviations:** 6 auto-fixed (2 Rule 2 missing critical, 4 Rule 1 bugs)
**Impact on plan:** These changes close safety gaps required by the plan's all-or-recoverable contract; no new package, mutation grammar, or unrelated surface was added.

## Automated Checks

- PASS — Task 1 RED gate failed before production edits with the maintenance plan/schema/target/preview/prepare contracts undefined.
- PASS — exact Task 1 gate: all 10 `TestMaintenanceMutation199*` cases.
- PASS — Task 2 RED gate failed before production edits with cleanup manifest/request/plan and registry mutation contracts undefined.
- PASS — exact Task 2 gate: all 4 `TestMaintenanceState199*` cases, including ownership rejection, baseline conflict, rollback, and receipt-digest proof.
- PASS — combined 14-case Plan 199-18 selection with the Go race detector (`ok`, 6.389s).
- PASS — complete `TestRegistry*` selection, registry lifecycle wiring, and direct worker-debug retention compatibility lanes.
- PASS — `git diff --check` on implementation and test changes.
- PASS — Wave 12 gate-recovery Plan 199-18 families in normal mode (`go test ./cmd -run '^TestMaintenance(Mutation|State)199' -count=1`, 5.629s) and race mode (`-race`, 6.971s).
- PASS — all named deterministic/stale/info/OK/critical/warning visual update E2Es (46.073s), plus stable/dev publish-update and install/setup/update flows (2.761s).
- PASS — registry and safe cleanup compatibility selection (4.752s), source/command/wiring ratchets (1.040s), `go vet ./...`, and `go build ./cmd/aether` to an isolated `/tmp` binary.
- EXPECTED LEGACY FAILURES — `TestDataCleanConfirm` still demands unsafe prefix deletion; broad `^Test.*Update` retains the already-deferred contradictory update-card setup plus a Plan 199-26 Next Up expectation (`aether build 1` versus authoritative `aether status`).
- EXPECTED STAGED FAILURES — `go test ./... -count=1` ran to completion in 478.414s: every `pkg/*` package passed; `cmd` retained the known Phase 199 migration backlog. Neither Plan 199-18 focused family failed.

## TDD Gate Compliance

- Task 1 RED commit `eece33d1` precedes GREEN commit `8490a55a`; the exact ten-case failure became a ten-case pass without weakening the tests.
- Task 2 RED commit `880c343c` precedes GREEN commit `abe4173d`; the exact four-case failure became a four-case pass, with registry fault proof strengthened during GREEN.
- Gate recovery RED commits `3cd1c41f`, `f505703d`, `27493c2f`, and `d7781770` precede GREEN commit `df20b816`; they cover stale zero-write refusal, typed early failures, module-root normalization, platform labels, executable-mode journaling, replay drift, and rollback-mode restoration.

## Known Stubs

None. Empty cleanup target lists are an intentional typed no-change result when no owned manifest exists, not placeholder data. No TODO/FIXME, mock result, or unconnected data source was introduced.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: filesystem-cleanup-authority | `cmd/maintenance.go` | A local manifest can request file deletion. Authority is restricted to a canonical path below the configured lifecycle data root with exact schema, owner, checkpoint, regular-file/no-symlink proof, and current digest; confirmation and the coordinator transaction remain mandatory. |
| threat_flag: hub-registry-replacement | `cmd/registry.go` | Registry input crosses from a CLI repository path into the selected hub. Stable identity, canonical roots, strict JSON, duplicate/active-state invariants, and a closed baseline digest are validated before one transaction replacement. |
| threat_flag: binary-replacement | `cmd/binary_download.go` | Downloaded executable bytes can replace an installed binary. Download/checksum/version/executable-mode verification completes in isolated staging; mode and bytes are then journaled, globally verified, and checked again at the installed destination. |

## Issues Encountered

- `TestDataCleanConfirm` still expects deletion authority from `test_`/`demo_` prefixes. Restoring it would violate CAP-008, so the legacy expectation is recorded in `deferred-items.md`; public data-clean now preserves those unowned decoys.
- `TestUpdateEndsWithTheCard/when_it_repaired_a_missing_command_copy` fails during fixture setup by removing an already-absent path before update runs. The exact update suite and directly related compatibility lanes pass; the contradictory fixture is deferred.
- The repository-wide command package retains the execution handoff's known staged Phase 199 failures across retired abandon/recover/resume aliases, status/Next Up snapshots, catalog/reachability goldens, and unconfirmed legacy entomb fixtures. Plan-owned normal and race gates are green.
- The installed GSD SDK reproduced its known metadata-format drift: `state.advance-plan` reset the frontmatter percentage to zero, `state.update-progress` could not find this STATE format's progress field, and `requirements.mark-complete` could not parse the already-checked bold em-dash requirement labels. The SDK did advance Plan 18 to 19, count 21 summaries, update ROADMAP to 21/34, and record metrics/decisions/session; the percentage was then corrected directly to 62 and read back after all mutations.
- Protected pre-existing `.planning/config.json`, `.gsd/`, and `199-PATTERNS.md` changes remained untouched and unstaged.
- The Wave 12 recovery audit reproduced the update failures at pre-recovery `e003b1df` and confirmed the corresponding Wave 11 cases passed before Plan 199-18; after recovery, every Plan 199-18-owned deterministic and update E2E is green. The unsafe prefix-clean test and contradictory update-card fixture remain intentionally unchanged.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 199-19 can build the next expert maintenance surface on one preview/transaction/receipt grammar rather than adding command-local mutation semantics.
- Plan 199-25 can apply the same closed target and receipt rules to archive/context work.
- Plan 199-29 can migrate the remaining legacy expectations and enforce the final repository-wide normal/race gate.
- No Plan 199-18-owned blocker remains.

## Self-Check: PASSED

- All ten owned source/test files and this summary exist.
- Original task commits `eece33d1`, `8490a55a`, `880c343c`, and `abe4173d`, plus gate-recovery commits `3cd1c41f`, `f505703d`, `27493c2f`, `d7781770`, and `df20b816`, exist in repository history.
- `git diff --check` is clean before metadata mutation.
- Final gate-recovery metadata readback (no SDK mutation): STATE remains 22 of 34 plans complete (`percent: 65`), current Plan 21, `Ready to execute`, and stopped at `Completed 199-26-PLAN.md`; ROADMAP remains 22/34 plans executed; CEC-04 and LIFE-06 remain checked complete.

---
*Phase: 199-front-door-and-classic-contract*
*Completed: 2026-09-04*
