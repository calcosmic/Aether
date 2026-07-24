# Disposable User-Journey Results

## Environment

- Date: 2026-07-21
- Host: macOS, arm64, Europe/Zurich
- Source binary: `/tmp/aether-audit-current`, built from the audited dirty checkout, reports `1.0.42`
- Repository checkout: `ef12c596` (`v1.24`) plus pre-existing changes
- Model runtime used for live A/plan/build: Codex CLI; a brownfield run unexpectedly selected Claude despite a Codex pin
- Disposable root: `/tmp/aether-framework-polish/`
- Output mode: JSON unless a live visual behavior was under inspection
- Test changes were confined to disposable repositories. One `control-ts` test erroneously deleted the live local Aether state file; it was restored semantically from `.aether/data/.cache_COLONY_STATE.json` and verified with `cmp`.

## Result Summary

| Journey | Result | Highest severity | Main finding |
| --- | --- | --- | --- |
| A. Greenfield application | Fail | P0 | Useful Oracle/planner work was not transactionally committed; interrupted build claimed success; final verification could be bypassed |
| B. Brownfield feature | Fail | P0 | Fallback colonization was generic, ignored Git history, left narrated workers spawned, and planning missed the requested feature |
| C. Difficult bug | Fail at contract level | P0 | Tracker was selected, but three Builders plus five review castes were also selected and no investigator write isolation exists |
| D. Interrupted phase | Fail with partial recovery | P0 | Cancellation set `BUILT` and `ok:true`; continue detected missing evidence, but state/ceremony were false |
| E. Plan invalidated by research | Fail | P0 | Assumptions can be recorded, but plan refresh is forbidden after any completed phase |
| F. Memory contamination | Fail | P0 | Arbitrary repo names produce high-confidence global wisdom and unrelated projects receive it |
| G. Parallel conflict | Fail | P0 | Both overlapping workers were accepted/marked merged while one file version was silently lost |
| H. Upgrade | Blocked/fail | P0 | Public npm install fails; direct install permits binary/companion version skew, so migration cannot be trusted end to end |

## Post-Remediation Compiled Reruns

The original live results below are retained as historical evidence. Three deterministic, compiled reruns now cover repaired core paths without pretending that the public release artifacts have changed.

### Core Install-To-Seal Journey

Exact test: `go test ./cmd -run TestCLICompiledInstallToSealJourney -count=1 -v`

Environment: disposable HOME/repository/cache, compiled `./cmd/aether`, and a separate compiled provider fixture executed through the production Codex subprocess contract.

Observed result: **Pass**.

The journey installed source-candidate assets, ran `lay-eggs`, initialized an ambiguous goal, surfaced and resolved discussion questions, ran one Oracle research pass with a persisted research plan, accepted a `bound_v1` plan, emitted a FOCUS signal, built and continued every phase through real provider processes, enforced every criterion, resumed from a new CLI process, recommended seal rather than entomb, and produced `CROWNED-ANTHILL.md`. Provider logs contained real Oracle, Builder, and Watcher processes.

The first rerun found that generated Discovery phases could not pass the implementation no-op gate even though their workers were intentionally read-only. The repair now requires durable per-task worker reports plus criteria/Watcher proof for Discovery while retaining the source-change requirement for implementation phases. The next rerun found and repaired the incorrect completed-colony `resume -> entomb` route.

### Install, Update, And Migration Journey

Exact test: `go test ./cmd -run TestCLICompiledInstallUpdateMigrationContract -count=1 -v`

Observed result: **Pass**.

The journey installed the compiled candidate into an isolated HOME, set up a repository, added project-owned Queen content and a custom skill, seeded a version-3.0 plan without evidence bindings, ran `update --force`, and proved those three project-owned artifacts were unchanged. `migrate-state` classified the plan as `legacy_unbound`, saved the original state bytes exactly, emitted a concrete rollback command, and preserved a readable migrated plan. `versions` reported matching binary and hub versions, and Claude/OpenCode/Codex agent assets existed in their declared homes.

### Research-Invalidated Plan Journey

Exact test: `go test ./cmd -run TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild -count=1 -v`

Observed result: **Pass for the deterministic source candidate**.

The journey starts with phase 1 completed and phase 2 invalidated, supplies the actual Oracle synthesis path, runs a compiled `plan --refresh`, and proves that completed phase JSON is byte-equivalent before and after. The accepted revision records its parent, reason, evidence path, input-content hash, planning evidence hash, preserved phase IDs, superseded phase IDs, and replacement phase IDs. A new process restores that revision through `resume`; the next build is bound to it and cannot select the completed task. Separate finalizer tests reject changed base state and changed evidence without mutating canonical state.

Limit: the compiled journey uses explicit synthetic planning so the state contract is deterministic. Real Oracle/Scout/Route-Setter provider execution through the same revision boundary remains a release acceptance test.

Remaining result: public npm/archive installation, N-1 release migration, interrupted activation, live-provider plan revision, host-enforced caste permissions, cross-project learning trust, and full parallel-wave transactions are still unproven or failing.

## Journey A: Greenfield Application

### Goal And Commands

Repository: `/tmp/aether-framework-polish/greenfield`

```bash
git init
aether lay-eggs
aether init "Build a tool for notes"
aether discuss --dry-run
aether discuss ...resolved answers...
aether oracle "What is the smallest useful notes tool?" --depth quick --max-iterations 1
aether plan --depth fast --max-iterations 1 --accept --worker-timeout 5m
aether plan --synthetic --accept
aether build 3 --light --worker-timeout 5m
aether resume
aether continue --plan-only
```

### Expected

Discussion identifies target user, interface, persistence, and constraints. Research resolves uncertainty. Planning commits coherent phases linked to findings. Build creates a small notes tool. Verification runs a behavior check and resume restores exact in-flight status.

### Actual

- Setup and init succeeded. Setup files immediately appeared as unreconciled implementation changes.
- Initial context capsule was only 24 words.
- Discussion asked generic questions about existing surface, optimization scope, and verification bar, not notes-specific product questions. Answers became long-lived REDIRECT signals.
- Oracle ran for roughly 98 seconds, persisted questions/sources/gaps/synthesis, stopped at 13% confidence, and recommended a sensible local single-user CRUD tool with plain-text persistence. Only one of six questions was answered.
- Real planning ran a Scout and Route-Setter serially. The Route-Setter wrote a detailed, goal-specific plan that referenced Oracle artifacts, but its process timed out after writing. The command emitted an error envelope yet exited 0. Canonical state still had no plan.
- Retrying with synthetic planning removed/overwrote the useful artifacts and created generic discovery/architecture/implementation phases.
- After disposable phase skips, the small implementation phase selected Architect, Keeper, two Builders, Probe, and Watcher, all serial. Architect timed out; the runtime continued. The process was interrupted during Keeper.
- The interrupted command returned `ok:true`, set state `BUILT`, and recorded `build_completed`. All dispatches were failed, timed out, or cancelled; claims were empty; tasks remained in progress.
- `resume` suggested `continue`. `continue --plan-only` correctly detected missing evidence and recommended redispatch.

### Files And Final State

Durable Oracle files under `.aether/oracle/`; planning reports under `.aether/data/planning/`; build manifests/results under `.aether/data/build/`; canonical state ended `BUILT` with no implementation evidence.

### Result

**Fail, P0.** Research and agent reasoning can be useful, and continue has valuable recovery checks. The state transition and completion presentation are not trustworthy.

### Reproduction

Use a fresh repository, real Codex adapter, a 5-minute per-worker timeout, and interrupt after the first timeout. Inspect command exit status, `COLONY_STATE.json`, build manifest/result collection, and claims.

## Journey B: Brownfield Feature

### Goal And Commands

Repository: a clean local full-history clone at `/tmp/aether-framework-polish/brownfield`, checked out at `v1.24`.

```bash
aether lay-eggs
aether init "Add a command that explains why a worker was selected"
AETHER_WORKER_PLATFORM=codex aether colonize
aether plan --synthetic --accept
```

### Expected

Colonization identifies Go/Cobra registration, `caste_relevance.go`, command catalog conventions, tests, and the relevant selection history. Plan targets one explainability command and reuses current selection metadata.

### Actual

- Colonize scanned 1,133 files and 133 directories; graph output contained 701 files and 574 edges.
- Despite `AETHER_WORKER_PLATFORM=codex`, dispatch attempted Claude and surfaced a real Claude provider 402 error. It then used local fallback synthesis and returned `ok:true`.
- The output listed surveyors as `spawned` although fallback synthesis ran instead.
- `BLUEPRINT.md` was a broad directory map. `PATHOGENS.md` focused heavily on `.planning` documents and produced false/noisy debt observations. `TRAILS.md` missed local storage nuance.
- No survey artifact cited commit, blame, or Git-history evidence.
- Synthetic plan phases were "Contract and gap mapping", "Colonize orchestration", and "Planning orchestration", unrelated to implementing the requested worker-selection explanation.

### Result

**Fail, P0.** Brownfield scanning works mechanically, but historical/convention understanding and goal-grounded planning are not proven. Platform selection is also incorrect.

## Journey C: Difficult Bug

### Setup

Repository: `/tmp/aether-framework-polish/bug-journey`. A valid focused phase was inserted into disposable canonical state to isolate routing behavior:

- Task 1.1: investigate why a Codex pin can select Claude, without editing implementation files.
- Task 1.2: implement repair after 1.1.
- Task 1.3: add regression tests after 1.2.

Command:

```bash
aether build 1 --plan-only --light
```

### Expected

Tracker gets a read-only investigation step. Builder cannot start until accepted diagnosis exists. Probe/test author and Watcher run only after the fix. Permissions enforce non-editing investigation.

### Actual

- Nine workers across nine serial execution waves were planned: Gatekeeper, Tracker, three Builders, Probe, Auditor, Chaos, and Watcher.
- The first investigation task was still assigned a Builder wave in addition to the Tracker pre-wave.
- "Read-only" is prompt language. Codex invokes every caste with `workspace-write`; Claude uses `bypassPermissions`.
- The Queen budget reported seven selected castes despite a nominal maximum of six because policy-added/required castes overflow the conceptual budget.
- Skill matching injected unrelated API security and Supabase material into this platform-selection bug.

### Result

**Fail, P0.** Role names separate narrative responsibilities, but execution permissions and task ownership do not preserve forensic evidence.

## Journey D: Interrupted Phase

This used Journey A's real build plus a separate corrupted-state experiment.

### Interrupted Build

The process was terminated during the second serial worker after the first worker timed out. Expected state was a paused/incomplete run with safe retry. Actual state was `BUILT`, response `ok:true`, event `build_completed`, empty claims, and failed/cancelled dispatches. Continue later detected the evidence gap.

### Corrupted State

A disposable copy of the bug journey had `COLONY_STATE.json` truncated to one byte:

```bash
aether status
aether resume
```

`status` emitted a useful corruption error and recovery options but exited 0. `resume` rebuilt state from `HANDOFF.md`, returned `resumed:true`, and deleted the handoff. It preserved the goal but lost the entire three-task plan and reset current phase to 0 with zero phases.

### Result

**Fail, P0.** Aether often pauses enough artifacts to diagnose a failure, but its public state and status labels can assert a stronger recovery/completion than occurred.

## Journey E: Plan Invalidated By Research

### Commands

The disposable bug plan was marked completed, then:

```bash
aether assumptions-analyze
aether plan --refresh --synthetic
```

### Expected

A contradicted assumption records evidence, identifies affected tasks/phases, creates a new plan revision, preserves completed unaffected work, and explains the change.

### Actual

`assumptions-analyze` created three generic assumptions and one FOCUS. `plan --refresh` returned:

```text
cannot force-replan after completed phases; archive this colony and start a new one
```

The error envelope again exited 0. There is no built-in operation to revise affected future phases while preserving completed work.

### Result

**Fail, P0.** Aether supports corrective phase insertion, not genuine iterative replanning.

### Post-Remediation Result

**Pass for the deterministic source candidate; live-provider proof pending.** `plan --refresh` no longer rejects a completed prefix. It preserves completed phases exactly, archives immutable revision snapshots, replaces only unfinished work, remaps dependencies, records rationale and evidence, exposes the active revision in status/resume/context, and rejects stale planner/build packets. The historical failure above remains evidence of the released behavior that motivated the repair.

## Journey F: Memory Contamination

### Commands And Cases

Two isolated homes and projects were used.

1. Promote incompatible project-specific conventions, one tabs and one spaces.
2. Promote the same unverified generic lesson from four arbitrary source names.
3. Assemble context for an unrelated Python spaces-only formatter.

Representative commands:

```bash
aether hive-promote --instinct "Always prefer tabs ..." --source-repo fake-one
# repeat with fake-two, fake-three, fake-four
aether colony-prime --workflow build --role builder --task "Python spaces-only formatter"
```

### Actual

- Four arbitrary distinct strings count as four independent repositories and raise confidence to 0.95.
- Hive entries have no evidence IDs, contradiction relationship, branch scope, validation status, revocation, or decay.
- One abstraction replaced a project name with `<repo>`; prompt sanitization then rejected the whole Hive section as structural markup, making the store inert.
- With sanitizer-safe text, unrelated project context received high-confidence "Always prefer tabs" advice with weak relevance.
- Global writes use direct `os.WriteFile` without a cross-process lock.

### Result

**Fail, P0.** Hive is simultaneously unsafe when injected and inert when its own abstraction triggers sanitization.

## Journey G: Parallel Conflict

### Controlled Adapter

A disposable fake Codex executable returned valid structured results. Each Builder created the same untracked file, `shared.txt`, with its own worker name. Worktree mode was enabled and only the two builder tasks were forced.

```bash
aether parallel-mode set worktree
aether build 3 --force --task 3.1 --task 3.2
```

### Expected

Ownership preflight rejects overlap, or merge detects a content conflict and pauses for reconciliation. At most one claim is accepted until resolution.

### Actual

- The two workers ran concurrently in separate worktrees.
- Both created different contents for `shared.txt`.
- Both worktrees were marked `merged`, both branches/worktrees were removed, and both claims were accepted.
- Root `shared.txt` contained only the later worker's content. The first result was silently lost.
- No conflict or overlapping ownership was reported.

The subsequent:

```bash
aether continue --skip-watchers --light --no-learn
```

marked the whole project `COMPLETED`. Build, typecheck, lint, and tests each reported `passed:true`, `skipped:true`, `no command resolved; skipped`. No notes application or tests existed.

### Result

**Fail, P0.** Parallel execution is real, but conflict safety and evidence-based completion are not.

## Journey H: Upgrade And Release

### Public npm Install

```bash
HOME=/tmp/... AETHER_BINARY_DEST=/tmp/... \
  npx --yes aether-colony@latest
```

Result: exit 1, `Binary aether not found in extracted archive`. npm latest was `1.0.22`; that release archive contained `Aether`.

### Direct Current Install

```bash
aether install --channel stable --download-binary
```

Using source `1.0.42`, companion assets were installed first, then download failed because GitHub has no `v1.0.42` release.

### Forced Older Binary

```bash
aether install --channel stable --download-binary \
  --binary-version 1.0.34
```

This succeeded with checksum verification but installed a `1.0.34` binary beside a hub reporting `1.0.42`. No mismatch failure occurred.

### State Migration And Preservation

Blocked as an end-to-end public journey because a new user cannot establish the documented npm starting point. Source-based dev install/update did preserve local data and copy 887 hub files, but this is not a substitute for a released N-to-N+1 migration. Rollback is not exposed as a tested user operation.

### Result

**Fail, P0.** Asset integrity is stronger than release-set integrity. The system verifies an archive checksum but does not atomically verify that binary, companion assets, npm package, migrations, and platform files are the same release.

## Pheromone Sub-Experiment

The audit wrote duplicate FOCUS, conflicting REDIRECT, sourced FEEDBACK, and an immediately expired signal.

- Duplicate identical FOCUS was correctly deduplicated and reinforced.
- Identical text could exist simultaneously as FOCUS and REDIRECT without conflict detection.
- Worker context injected both; the prompt says REDIRECT wins, but runtime did not record a resolved policy decision.
- A zero-TTL signal remained displayed as active until housekeeping mutated storage, while prompt loading filtered by current expiry.

Conclusion: pheromones are currently useful expiring prompt inputs, not yet a coordination primitive. A real primitive needs scope, conflict policy, consumer acknowledgment, deterministic routing/policy effects, and outcome traceability.

## Test-Safety Incident

Running `npm test` in `control-ts` failed because `tests/control/integration.test.ts` resolves `TEST_STATE_PATH` to the repository's actual `.aether/data/COLONY_STATE.json`. Its cleanup deletes that file. Parallel tests raced, yielding `ENOENT`. The file was restored from the cache and semantic equality verified.

This test must be disabled in shared/source worktrees until it uses a temporary project root. It is direct evidence that the proposed control plane does not yet respect the Go-only state ownership boundary even in its tests.
