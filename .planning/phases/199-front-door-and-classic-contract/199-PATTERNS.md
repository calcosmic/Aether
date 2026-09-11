# Phase 199: Front Door and Classic Contract - Pattern Map

**Mapped:** 2026-09-03
**Files analyzed:** 22 path families (new files, modified seams, generated surfaces, and removals)
**Analogs found:** 22 / 22 path families have a usable exact, role, or partial analog
**Primary code scope:** `cmd/`, `pkg/colony/`, `pkg/storage/`, `.aether/commands/`, generated Claude/OpenCode command surfaces

## Planning Summary

Phase 199 should not create another lifecycle state machine beside the existing one. Its safest pattern is:

1. load persisted state without repairing or writing it;
2. normalize that state into one typed lifecycle-facts model;
3. resolve one pure projection from those facts;
4. render that same projection into JSON, visual output, help, and closeouts; and
5. route every mutating workflow through a validate/stage/commit/rollback transaction boundary.

In plain English: gather the facts once, decide what they mean once, then let every command repeat the same truthful answer. For commands that change files, prepare the whole change before touching live state and leave a recovery record if the process is interrupted.

The current code contains strong pieces of that design, but not the complete transaction or recovery protocol. The planner must copy the good pieces named below and must not copy the legacy sequential-write behavior called out under **Patterns to Avoid**.

## File Classification

Paths in braces are a single implementation family and should normally land in the same plan or in dependency order. “Conditional” means RESEARCH.md recommends the path only if the abstraction is genuinely reusable outside `cmd`.

| New/Modified File or Family | Change | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|---|
| `pkg/colony/lifecycle.go`, `pkg/colony/lifecycle_test.go` | new, recommended | model, test | transform | `pkg/colony/session.go`; `pkg/colony/state_machine.go`; lifecycle fields in `pkg/colony/colony.go` | role-match |
| `cmd/lifecycle_facts.go` | new | utility/model | file-I/O → batch facts | `cmd/state_load.go`; `cmd/next_action_input.go` | exact |
| `cmd/lifecycle_projection.go` | new | utility/model | pure transform | `cmd/next_action.go`; `cmd/next_action_card.go` | exact |
| `cmd/lifecycle_transaction.go` | new | service/utility | staged file-I/O | `pkg/storage/storage.go`; migration/rollback code in `cmd/command_truth.go` | partial: no multi-artifact transaction exists |
| `pkg/storage/*transaction*.go`, tests | conditional new/modify | utility/service | file-I/O | `pkg/storage/storage.go` | role-match; keep in `cmd` unless broadly reusable |
| `cmd/lifecycle_projection_test.go` | new | test | transform/batch | resolver tests around `cmd/next_action.go`; parity table tests | exact |
| `cmd/lifecycle_transaction_test.go` | new | test | file-I/O/failure injection | migration rollback and black-box finalizer tests | role-match |
| `cmd/classic_contract_test.go` | new | test/harness | batch + subprocess request-response | `cmd/blackbox_harness_test.go`; `cmd/classic_command_parity_test.go` | exact |
| `cmd/testdata/classic-contract/v1/manifest.json`, `{front-door,orientation,autopilot,pause-resume,closure,maintenance}/**` | new | config/test fixture | batch/transform/file-I/O | `.aether/commands/classic-command-parity.json`; fixture loaders in `cmd/classic_command_parity_test.go` | role-match; new semantic schema |
| `cmd/{state_load,next_action,next_action_input,next_action_card,lifecycle_helpers}.go` | modify | utility/model | file-I/O → pure transform | the same next-action pipeline | exact/self |
| `cmd/{status,phase,history,context,codex_visuals}.go` | modify | controller/view | request-response + transform | `cmd/next_action_card.go`; shared closeout helpers in `cmd/codex_visuals.go` | exact |
| `cmd/{root,init_cmd,survey_staleness,codex_colonize,codex_colonize_finalize,codex_plan_finalize}.go` | modify | controller/service | request-response + file-I/O | `cmd/codex_colonize_finalize.go`; `.aether/commands/colonize.yaml` | role-match; root grouping has no precedent |
| `cmd/{compatibility_cmds,autopilot_policy,run_visuals}.go` | modify | controller/service/view | event-driven batch loop + request-response | current typed autopilot policy and run renderers | exact/self, behavior must change |
| `cmd/{session_flow_cmds,session_compat}.go` | modify | controller/service | file-I/O + request-response | current pause/resume evidence gathering; transaction analog above | partial: sequential writes are unsafe |
| `cmd/{codex_workflow_cmds,seal_final_review,seal_confirmation,entomb_cmd,codex_visuals}.go` | modify | controller/service/view | staged file-I/O + request-response | current seal/entomb preflight and renderers; transaction analog above | partial: current commit order is unsafe |
| `cmd/{maintenance,integrity_cmd,chamber,registry,source_check,update_cmd,command_truth}.go` and co-located tests | modify | controller/service | batch + file-I/O | `cmd/command_truth.go` migration/rollback command | role-match |
| `cmd/testing_main_test.go` | modify | test utility | file-I/O cleanup | isolated ownership patterns in `cmd/blackbox_harness_test.go` | role-match; current cleanup must not be copied |
| `.aether/commands/{help,init,colonize,plan,build,run,status,watch,phase,history,pause,resume,seal,entomb}.yaml` | modify | config/provider | request-response orchestration | `.aether/commands/colonize.yaml`; `.aether/commands/seal.yaml` | exact |
| `.aether/commands/resume-colony.yaml`; generated `pause-colony`/`resume-colony` surfaces | remove | config/generated surface | transform | managed-file pruning in `cmd/platform_sync.go` | exact |
| `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`, flattened legacy mirrors affected by the canonical YAML | regenerate/remove | generated config | transform | generated `ant/status.md` files; generator in `cmd/platform_sync.go` | exact; generated output, do not hand-author |
| `cmd/{wrapper_command_names,command_guide,platform_sync,source_check}.go`, parity/install/update/publish tests, `.aether/commands/classic-command-parity.json` | modify | config/service/test | batch + file-I/O | `cmd/classic_command_parity_test.go`; `cmd/platform_sync.go` | exact |
| Product-facing help/rules/contracts/playbooks containing current pause/resume or closure vocabulary | modify selectively | documentation/config | transform | canonical YAML vocabulary and generated-header ownership rules | role-match |

### Product-facing text inventory

The research and repository search identify these live-documentation families for a deliberate vocabulary sweep:

- `.claude/rules/aether-colony.md`
- `.claude/commands/ant-help.md`, `.claude/commands/ant-organize.md`, and current pause/resume command mirrors
- `.aether/docs/command-playbooks/continue-{finalize,full}.md`
- `cmd/contracts/resume.md`
- corresponding `.claude/commands/ant/` and `.opencode/commands/ant/` generated command files

Only change current product vocabulary. Historical event records, Dreams, archived reports, and quoted historical decisions are evidence and should remain untouched.

## Pattern Assignments

### 1. `cmd/lifecycle_facts.go` and lifecycle read paths

**Role / flow:** utility-model; read-only file-I/O into a typed batch of facts.

**Primary analog:** `cmd/state_load.go` lines 17-40 and 80-102.

**Companion analog:** `cmd/next_action_input.go` lines 3-16, 37-69, and 165-199.

**Imports pattern:** use direct package imports from the module and standard-library error/path packages; do not add a global cache or a UI dependency. `next_action_input.go` keeps reading separate from deciding, which is the correct dependency direction.

**Core pattern to copy:**

```go
// cmd/state_load.go:17-40
loadActiveColonyStateReadOnly(...)
    -> resolve the active state path
    -> Store.LoadJSON(...)
    -> apply compatibility normalization in memory
    -> return state and path without SaveJSON/AtomicWrite
```

```go
// cmd/next_action_input.go:37-69
loadNextActionInput(...)
    -> collect persisted lifecycle state
    -> collect adjacent evidence
    -> return a typed input; do not decide or render here
```

**Validation/error pattern:** return explicit load/parse errors rather than turning them into “no colony.” Preserve the distinction between absent evidence and malformed evidence. Compatibility repair is allowed only in memory on observation paths.

**Apply to:**

- `status`, `phase`, `history`, `context`, next-action, early `run`, and help/orientation inspection;
- clean/unclean resume evidence collection;
- seal/entomb preflight before the mutation transaction begins.

**Do not copy:** the mutating loader in `cmd/state_load.go` lines 42-77. It persists compatibility repair and therefore violates D2’s zero-mutation early-run contract when used during orientation.

---

### 2. `cmd/lifecycle_projection.go` and every orientation/view consumer

**Role / flow:** pure model/utility; facts → semantic projection.

**Primary analog:** `cmd/next_action.go` lines 3-31, 70-137, 161-216, and 503-527.

**Renderer analog:** `cmd/next_action_card.go` lines 3-29, 45-117, 120-131, and 227-300.

**Core pattern to copy:**

```text
typed lifecycle facts
    -> one pure resolver (`resolveNextAction` is the existing precedent)
    -> typed result with stable semantic fields
    -> platform translation only at the render boundary
    -> JSON and visual renderers consume the same result
```

The most important existing split is documented at `cmd/next_action_card.go:3-29`: the renderer does not decide. Platform-specific spelling is translated only when output is rendered (`cmd/next_action_card.go:120-131`). Keep those properties when the narrower next-action model becomes the shared Phase 199 lifecycle projection.

**Stable result-extension pattern** (`cmd/next_action_card.go:227-273`):

```go
// Existing compatibility approach:
applyNextActionToResult(...)
// adds the shared typed answer to a command result without silently removing
// older compatibility keys in the same release.
```

Use this for progressive migration: status, phase, history, run engage, closeouts, help, resume, and closure should gain the common projection before duplicated derivations are deleted.

**Validation/error pattern:** the resolver accepts already-loaded facts and returns a deterministic value. File errors belong to the facts loader; formatting errors belong to the renderer. Do not perform writes, shell execution, or terminal detection in the resolver.

**Required Phase 199 corrections to the analog:**

- `cmd/next_action.go:568-580` currently treats `entomb` as the expected last step after seal. Under D17, entomb is optional archive-and-clear, not required continuation.
- `cmd/next_action.go:617-628` currently favors `discuss` when no plan exists. Under D3, an accepted goal should present plan and build/run as equal choices after init.
- A sealed outcome must distinguish normal completion from forced-incomplete closure; it cannot be represented by a single `sealed=true`/“complete” projection.

**Apply to:**

- `cmd/status.go`: replace its duplicated progress derivation and ensure `--compact` is a projection/render option, not a different truth model;
- `cmd/phase.go` and `cmd/history.go`: stop independently rebuilding lifecycle maps from raw JSON;
- `cmd/context.go`: consume facts without invoking compatibility writes;
- `cmd/run_visuals.go`: render goal, intended phase range, active pheromones, and pause conditions from the same projection before execution;
- `cmd/codex_visuals.go`: use one typed outcome for result JSON and closing ceremony.

---

### 3. `pkg/colony/lifecycle.go`

**Role / flow:** durable schema/model; JSON records and pure transitions.

**Analogs:**

- `pkg/colony/session.go` lines 1-19 for small, flat, typed JSON records;
- `pkg/colony/state_machine.go` lines 1-38 for pure transition validation with explicit errors;
- `pkg/colony/colony.go` lines 326-470 for compatibility fields, `json` tags, and `omitempty` conventions.

**Schema pattern to copy:**

```go
// pkg/colony/session.go: small durable-record convention
type ... struct {
    ... string `json:"..."`
    ... time.Time `json:"..."`
}
```

```text
// pkg/colony/state_machine.go: transition convention
typed current state + requested transition
    -> validate allowed transition
    -> return explicit error for an illegal transition
    -> do not perform persistence in the model package
```

Keep durable semantic facts here only when they are shared by multiple commands. Renderer labels, ANSI styling, Cobra flags, and wrapper names remain in `cmd`.

**Required records/enums inferred from CONTEXT.md and RESEARCH.md:**

- normal seal vs forced-incomplete seal outcome;
- pause handoff identity and exact-once transaction status;
- resume evidence classification: `confirmed`, `reconstructed`, `conflicting`, `unknown`;
- transaction/journal phase sufficient for deterministic recovery;
- archive manifest entry containing source path, archived path, digest, and verification status.

Prefer additive fields with `omitempty` where older state must continue loading. Tests should cover absent legacy fields, not just newly written state.

---

### 4. `cmd/lifecycle_transaction.go` and conditional `pkg/storage` support

**Role / flow:** service/utility; validate → stage → durable intent → commit → verify → cleanup/rollback.

**Closest partial analog:** `pkg/storage/storage.go` lines 17-38, 45-77, 79-137, 139-157, and 196-227.

**Recoverability analog:** `cmd/command_truth.go` lines 460-538 and 549-612.

`pkg/storage` already provides the correct single-file building blocks:

```text
// pkg/storage/storage.go:45-137
lock target
    -> write a temporary file
    -> validate serialized JSON where appropriate
    -> rename atomically
    -> unlock
```

`cmd/command_truth.go` adds the closest command-level recovery shape:

```text
// cmd/command_truth.go:460-588
validate and normalize all input first
    -> create a backup
    -> save the migration
    -> on rollback, create a pre-rollback safety backup
    -> restore through Store.UpdateFile
```

Its containment guard is directly reusable (`cmd/command_truth.go:590-612`): accept only the expected relative backup area/name and reject symlinks with `Lstat` before reading or restoring.

**New protocol required by Phase 199:** the existing analogs are per-file or migration-specific; they are not a complete multi-artifact transaction. The new helper should expose explicit phases and durable evidence similar to:

```text
validate all source/evidence paths
    -> stage every output in an owned transaction directory
    -> fsync/close staged files as required by the chosen durability contract
    -> persist a typed intent/journal record
    -> commit each target using same-filesystem rename where possible
    -> verify expected digests/state invariants
    -> mark committed and remove staging evidence

on interruption:
    read intent + target/staging evidence
    -> deterministically finish or roll back
    -> classify ambiguity instead of guessing
```

**Containment and validation rules:**

- Resolve and validate every path before the first live write.
- `pkg/storage/storage.go:315-322` accepts absolute paths, so the transaction caller must enforce workspace/owned-root containment; the store itself is not a sandbox.
- Reject symlink traversal for backups, archive inputs, journals, and destructive clear targets.
- Require explicit ownership for cleanup artifacts.
- A best-effort error log is not a rollback protocol.

**Placement rule:** keep lifecycle policy and journal semantics in `cmd/lifecycle_transaction.go`. Move a primitive to `pkg/storage` only if it is command-independent and useful to at least two non-lifecycle callers. This avoids turning `pkg/storage` into a second workflow engine.

**Tests:** table-drive failures before staging, after staging, after intent persistence, during each commit step, during verification, and during cleanup. Re-run recovery twice to prove idempotence.

---

### 5. Host prepare/finalize commands

**Role / flow:** controller/service; request-response orchestration plus one authoritative commit.

**Primary analog:** `cmd/codex_colonize_finalize.go` lines 16-49, 52-89, 109-137, 333-465.

**Canonical wrapper analog:** `.aether/commands/colonize.yaml` lines 1-37.

**Imports/command pattern:** Cobra handlers remain thin: parse flags/input, load the typed manifest, call one finalizer, apply common closeout, and emit through the normal output mode. The typed completion result at `cmd/codex_colonize_finalize.go:16-23` is the preferred boundary.

**Authority/guard pattern** (`cmd/codex_colonize_finalize.go:109-137`):

```text
validate manifest mode
    -> require the finalizer contract
    -> require actual dispatch records
    -> validate repository root and freshness
    -> compare the captured workspace baseline
    -> only then begin mutation
```

**Declared-output validation** (`cmd/codex_colonize_finalize.go:333-440`): merge worker results, require declared outputs, validate every claimed path, and reject claims outside the allowed root. Record real spawn-tree evidence (`cmd/codex_colonize_finalize.go:450-465`) rather than synthesizing successful workers.

**Wrapper contract** (`.aether/commands/colonize.yaml:12-37`):

```yaml
# Canonical shape, abbreviated from the current wrapper
# 1. ask the runtime for a JSON manifest
# 2. dispatch real host workers from that manifest
# 3. submit their declared results to the runtime finalizer
# 4. let the runtime perform state mutation and closeout
# Guardrails: wrappers do not write colony state or parse visual output.
```

Apply this split to init/territory/plan completion, pause/resume, and seal/entomb wherever host work is needed. The wrapper may orchestrate; the runtime finalizer owns validation and mutation.

**Important correction:** `cmd/codex_colonize_finalize.go:139+` begins direct sequential writes after validation. Preserve its manifest/claim/baseline guards, but route Phase 199 multi-artifact effects through the shared transaction before considering the operation complete.

---

### 6. Front door: `root.go`, init, help, and early run

**Role / flow:** controller/view; request-response, with a strict observation-only branch.

**Closest analogs:** lifecycle projection family above and the thin command registration style in `cmd/root.go:169-210`.

**Command pattern:** root registration and persistent setup remain centralized. The current root persistent pre-run initializes storage and calls first-run handling (`cmd/root.go:169-199`); early `run` must be audited so this setup cannot mutate before the zero-mutation decision is returned.

**No exact analog:** the repository currently has no Cobra `GroupID`/progressive-help hierarchy. Follow the Cobra API and RESEARCH.md contract, then test the visible command tree. Do not invent aliases to make grouping easier.

**Required projection behavior:**

- No accepted goal: exact next command is `aether init` / `/ant-init` on wrapper platforms.
- Accepted goal but no plan: present plan and build/run as equal choices, not a false forced sequence.
- Early run: exact `init` or `plan` recommendation, no state repair, no session mirror, no timestamp update, no registry touch, no temp artifact.
- Valid run: show goal, intended phase range, active pheromones, and pause conditions, then begin the loop.

Use before/after workspace fingerprints modeled on `cmd/next_action_input.go`’s read-only invariant and the black-box harness below. A successful exit code is insufficient evidence of zero mutation.

---

### 7. Territory checks and plan finalization

**Role / flow:** service/controller; read facts → policy enum → optional host workflow → transaction.

**Primary analog:** `cmd/survey_staleness.go` lines 1-90 for timestamp validation, bounded context, and current notice calculation.

**Finalizer analog:** `cmd/codex_colonize_finalize.go` as described above.

Retain the staleness function’s early input validation and bounded operation. Replace free-form notice-only behavior with a typed outcome that can represent at least fresh, missing, stale-but-usable, and refresh-required. The shared projection should explain the outcome; the finalizer should own any state update.

Do not make `plan` parse human/visual output from `colonize`. Exchange typed JSON or an internal Go value.

---

### 8. Autopilot: `compatibility_cmds.go`, `autopilot_policy.go`, `run_visuals.go`

**Role / flow:** controller/service/view; event-driven batch loop.

**Existing analog to retain:** typed policy decisions and typed pause/complete render paths in `cmd/run_visuals.go` lines 107-191.

**Current entry seam:** `cmd/compatibility_cmds.go` lines 259-320 and 650-1140. The loop already separates command plumbing, state loading, engage output, and policy evaluation sufficiently to insert the shared projection.

**Required corrections:**

- Replace the mutating compatibility load at `cmd/compatibility_cmds.go:712` and `978-984` on early orientation paths with the read-only lifecycle facts loader.
- Replace the raw no-plan error at `cmd/compatibility_cmds.go:716-718` with the shared typed recommendation.
- Expand engage output beyond the current goal/current/max fields at `cmd/run_visuals.go:19-31`.
- Contract bounded repair categories, warnings/debt, and authority-boundary pause reasons in the semantic corpus now. Do not claim that Phase 199 implements later substantive repair behavior.

The policy result, visual message, JSON result, and final persisted pause reason must use one enum/reason code. Rendering should not infer why a loop stopped from prose.

---

### 9. Pause/resume: `session_flow_cmds.go` and `session_compat.go`

**Role / flow:** controller/service; evidence collection plus transactional file-I/O.

**Reusable portion of the current analog:** `cmd/session_flow_cmds.go` lines 36-72 for freshness/evidence gathering and typed precondition checks.

**Do not copy:**

- visible alias registration at `cmd/session_flow_cmds.go:75-80` and `176-180`;
- pause’s state-then-artifact sequential writes at `cmd/session_flow_cmds.go:91-113`;
- resume’s sequential writes at `cmd/session_flow_cmds.go:200-261`;
- `ensureLegacySessionMirror` in `cmd/session_compat.go:102+`, especially when reached from a read-only dashboard.

**Target pattern:**

```text
pause:
  collect evidence -> validate owner/workspace -> stage state + handoff
  -> persist exact-once intent -> atomically publish both -> verify

resume:
  collect state + handoff + workspace evidence without writing
  -> classify confirmed/reconstructed/conflicting/unknown
  -> if safe, transact the resumed state and recovery record
  -> if ambiguous, stop with evidence and no invented fact
```

Only `/ant-pause` and `/ant-resume` are generated/documented public commands. Any compatibility parser redirects must be bounded and hidden; they must not create Cobra commands, help entries, YAML aliases, generated markdown, or parity records.

---

### 10. Seal and entomb

**Role / flow:** controller/service/view; authoritative closure and optional archive transaction.

**Current preflight analog:** `cmd/codex_workflow_cmds.go` lines 532-611 for force flags, owner reason validation, and override typing.

**Current render/summary seams:** `cmd/codex_workflow_cmds.go` lines 858-875 and 1415-1591; `cmd/codex_visuals.go` lines 3262-3330.

**Current entomb inventory analog:** `cmd/entomb_cmd.go` lines 493-565 for manifest construction and artifact enumeration.

Retain explicit owner-only force and reason validation. Replace the single generic “sealed/complete” outcome with typed normal and forced-incomplete outcomes. The summary must derive completed/remaining counts from facts; the current code reports `len(plan)` as completed (`cmd/codex_workflow_cmds.go:1444`) and the current visual renderer always presents final completion (`cmd/codex_visuals.go:3262-3330`). Those are examples to replace, not copy.

**Seal transaction order:** all final review and authority checks first; stage report/state/registry/signal changes; persist intent; commit; verify; then render the result. The current path expires signals and saves state before all report/registry effects (`cmd/codex_workflow_cmds.go:772-875`), so it can expose partial closure.

**Entomb transaction order:**

```text
inventory owned artifacts with Lstat
    -> reject out-of-root/symlink surprises
    -> copy to staged archive
    -> compute source and archive digests
    -> write manifest with path + digest + classification
    -> verify every archived artifact and manifest
    -> atomically publish archive
    -> only then clear active state through the transaction
```

The current manifest is metadata-only (`cmd/entomb_cmd.go:493-514`), copy follows ordinary `Stat`/non-durable writes (`567-620`), and verification checks existence rather than content (`652-665`). Keep its artifact inventory idea, but do not treat those checks as sufficient for D18. Entomb remains optional and the sealed state stays usable until entomb explicitly succeeds.

---

### 11. Maintenance command family

**Role / flow:** grouped controllers over batch/file-I/O services.

**Primary analog:** `cmd/command_truth.go` lines 83-132 for a Cobra command with explicit flags and lines 460-650 for typed result, prevalidation, reversible action, and human rendering.

Group existing raw operations from:

- `cmd/maintenance.go`
- `cmd/integrity_cmd.go`
- `cmd/chamber.go`
- `cmd/registry.go`
- `cmd/source_check.go`
- `cmd/update_cmd.go`
- migration/rollback in `cmd/command_truth.go`

The public `/ant-*` family should present a small expert grouping over these runtime primitives. Exact raw subcommand spelling is the agent’s discretion, but every destructive/reversible operation should share the lifecycle transaction and containment checks rather than growing another backup convention.

Return a typed result containing action, target, backup/journal identifier, verification status, and recovery command. Render that result in plain language; do not make callers parse the renderer.

---

### 12. Canonical YAML wrappers and generated Claude/OpenCode surfaces

**Role / flow:** config/provider; canonical orchestration → generated platform commands.

**Canonical analog:** `.aether/commands/colonize.yaml` lines 1-37.

**Generated analog:** `.claude/commands/ant/status.md` and `.opencode/commands/ant/status.md` lines 1-19.

The YAML file owns command description, runtime invocation, wrapper orchestration, and guardrails. Generated markdown carries the managed header and mirrors that source. Therefore:

1. edit `.aether/commands/*.yaml`;
2. update generator/catalog logic where the command set changes;
3. regenerate both platform namespaces;
4. let managed-file pruning remove retired aliases;
5. assert byte/semantic parity in tests.

Do not hand-author a generated `.md` file as the primary fix.

**Removal pattern:** managed generated-file detection and pruning in `cmd/platform_sync.go` lines 387-681. Use it to remove obsolete generated alias files safely. Delete canonical `.aether/commands/resume-colony.yaml`; remove `pause-colony` from the pause YAML aliases; ensure both nested and flattened legacy managed mirrors are pruned where the generator owns them.

**Authority guardrails to repeat across lifecycle YAML:** wrappers dispatch real workers only when the runtime manifest requests them, pass machine-readable results to the runtime finalizer, never edit lifecycle state, never parse ANSI/visual output, and never invent completion or spawn records.

---

### 13. `cmd/classic_contract_test.go` and versioned corpus

**Role / flow:** black-box test harness; fixture-driven subprocess request-response plus causal state assertions.

**Primary harness analog:** `cmd/blackbox_harness_test.go` lines 20-169.

**Static catalog analog:** `cmd/classic_command_parity_test.go` lines 12-61, 74-175, and 178-254.

**Isolation pattern** (`cmd/blackbox_harness_test.go:36-169`): build/reuse test binaries, create a fresh repository and isolated home/temp environment per case, run the actual CLI subprocess, and capture stdout/stderr/exit status. Keep fixtures independent of the developer’s real hub and colony.

**Causal/idempotence pattern:**

- finalizer replay and tamper rejection at `cmd/blackbox_harness_test.go:728-799`;
- update/migration preservation and exact backup/rollback assertions at `cmd/blackbox_harness_test.go:1172-1296`.

Those tests prove more than text parity: they invoke the binary, inspect artifacts, replay operations, tamper with evidence, and compare restored bytes. Phase 199 corpus cases should use the same standard.

**Static matrix pattern:** `cmd/classic_command_parity_test.go` already loads a typed, versioned-looking catalog, verifies required command names, wrapper/guide parity, finalizer authority constraints, uniqueness, and wrapper existence. Preserve these checks, but do not mistake them for the new semantic contract.

**Recommended v1 case shape:**

```json
{
  "id": "front-door.early-run.requires-init",
  "area": "front-door",
  "version": 1,
  "arrange": {
    "state_fixture": "...",
    "files": []
  },
  "act": {
    "argv": ["run"]
  },
  "assert": {
    "exit_code": 0,
    "semantic_fields": {},
    "state_delta": "none",
    "required_artifacts": [],
    "forbidden_artifacts": []
  }
}
```

The exact schema is new, but the test discipline is not. Each case needs a semantic assertion and a causal filesystem/state assertion. Human wording may have limited token checks; ANSI snapshots must not be the source of truth.

**Minimum corpus groups required by the decisions:**

- `front-door/`: empty repo, accepted goal, no plan, visible command hierarchy, no public pause/resume aliases;
- `orientation/`: full/compact status agreement, phase/history/common lifecycle facts, read-only dashboard;
- `autopilot/`: engage contract, bounded repair warning/debt contract, unsafe/authority pause contract;
- `pause-resume/`: exact-once pause, clean resume, reconstructed/conflicting/unknown recovery, replay;
- `closure/`: normal seal, forced-incomplete seal, usable sealed state, entomb digest verification and failed-clear prevention;
- `maintenance/`: grouping, reversible mutation, containment/symlink rejection, recovery instructions.

---

### 14. `cmd/testing_main_test.go`

**Role / flow:** test infrastructure; owned-resource cleanup.

**Analog:** per-test isolated resources and explicit paths in `cmd/blackbox_harness_test.go:95-169`.

The current suite cleanup at `cmd/testing_main_test.go:220-259` enumerates broad git worktree/branch patterns (`phase-*` and one-segment `feature/*`) and can delete user-owned resources. Replace it before relying on the new corpus.

Use an exact ownership registry created by the test process:

```text
test creates resource
    -> record exact absolute worktree path + exact branch/ref + test run ID
cleanup
    -> validate record belongs to this run and allowed temp root
    -> remove only those exact recorded resources
```

Do not discover cleanup targets from broad branch prefixes, repository-wide enumeration, or unresolved environment variables.

## Shared Patterns

### Read-only means byte-for-byte no mutation

**Source:** `cmd/state_load.go:17-40`, `cmd/next_action_input.go:165-199`.

**Apply to:** early run, help/orientation, status, phase, history, context, resume diagnosis, seal preflight.

Assert state files, registry files, session mirrors, timestamps, temp paths, and untracked artifact inventory before and after. In-memory compatibility normalization is acceptable; persistence is not.

### One typed truth, multiple renderers

**Source:** `cmd/next_action.go:503-527`, `cmd/next_action_card.go:3-29,120-131,227-273`, `cmd/codex_visuals.go:820-846`.

**Apply to:** every lifecycle command, engage output, pause/closeout, normal/forced seal, JSON and visual modes.

The result object is authoritative. Visual output and wrapper spelling are projections of it.

### Runtime owns facts and mutations; wrappers own host orchestration

**Source:** `cmd/codex_colonize_finalize.go:109-137,333-465`; `.aether/commands/colonize.yaml:12-37`.

**Apply to:** colonize/plan finalization, pause/resume, seal/entomb, and any maintenance action requiring host work.

Wrappers submit typed evidence. Runtime code validates and commits it.

### Validate containment before opening a mutation path

**Source:** `cmd/command_truth.go:590-612`.

**Apply to:** backups, transaction staging, pause handoffs, resume recovery evidence, entomb archive, cleanup, and test-owned resources.

Use `Lstat` where symlink behavior matters and explicitly prove the resolved path is beneath the owned root.

### Every mutation has a typed recovery story

**Source:** `cmd/command_truth.go:517-588`; black-box rollback assertions at `cmd/blackbox_harness_test.go:1172-1296`.

**Apply to:** lifecycle transaction, maintenance, pause/resume, seal, entomb.

The result should name the backup/journal and the next recovery action. Tests compare restored bytes and support safe replay.

### Generated command surfaces come from YAML

**Source:** `.aether/commands/colonize.yaml:1-37`; managed headers in generated status wrappers; `cmd/platform_sync.go:387-681`.

**Apply to:** all Claude/OpenCode `/ant-*` lifecycle commands and alias removal.

Canonical YAML, generator/catalog, both generated namespaces, and parity tests change as one unit.

### No authentication layer is involved

This CLI has no web authentication/authorization middleware analog. The relevant guard is local owner authority: explicit flags, a non-empty reason for forced incomplete closure, workspace/root validation, and typed manifest evidence. Do not add an unrelated auth abstraction.

## Patterns to Avoid

| Legacy Pattern | Location | Why it must not be copied |
|---|---|---|
| Observation through a compatibility loader that writes repaired state | `cmd/state_load.go:42-77`; callers in status/run | Violates zero-mutation orientation and makes a read command causally significant. |
| Read dashboard backfilling a legacy session mirror | `cmd/context.go:99-125`; `cmd/session_compat.go:102+` | A status/dashboard query must not create lifecycle artifacts. |
| Sequential state and handoff writes | `cmd/session_flow_cmds.go:91-113,200-261` | Crash can leave pause/resume half-applied. |
| Sequential seal effects with best-effort registry/report completion | `cmd/codex_workflow_cmds.go:772-875` | Exposes a sealed state before all required closure evidence exists. |
| Treating forced seal as normal completion | `cmd/codex_workflow_cmds.go:1435-1445`; `cmd/codex_visuals.go:3262-3330` | Invents completion and erases remaining work. |
| Treating entomb as the required post-seal next step | `cmd/next_action.go:568-580` | Conflicts with D17; sealed state must remain usable. |
| Metadata/existence-only archive verification | `cmd/entomb_cmd.go:493-665` | Does not prove the archive matches what will be cleared. |
| Broad test cleanup by branch prefix | `cmd/testing_main_test.go:220-259` | Can remove user-owned worktrees/branches. |
| Static wrapper-name parity as the whole contract | `cmd/classic_command_parity_test.go` | Proves inventory, not behavior or state causality. |
| Hand-editing generated command markdown | `.claude/commands/ant/`, `.opencode/commands/ant/` | Will drift from YAML and be overwritten by sync/update. |

## No Complete Analog Found

These gaps are expected. The planner should use the locked decisions and RESEARCH.md design, while composing from the partial patterns above.

| Needed Capability | Role | Data Flow | Closest Partial Source | Missing Piece |
|---|---|---|---|---|
| Progressive Cobra help/command grouping | controller/config | request-response | `cmd/root.go` command registration | No existing `GroupID` hierarchy or classic front-door grouping. |
| Multi-artifact lifecycle transaction | service | file-I/O | `pkg/storage/storage.go`; `cmd/command_truth.go` | No durable intent journal coordinating several live files. |
| Evidence-based unclean resume reconciliation | service/model | file-I/O → transform | pause/resume evidence gathering | No shared confirmed/reconstructed/conflicting/unknown classifier. |
| Digest-verified archive then clear | service | file-I/O | `cmd/entomb_cmd.go` inventory | Current manifest has no content digests and verification is existence-only. |
| Versioned semantic/causal corpus schema | test/config | batch + subprocess/file-I/O | parity JSON and black-box harness | Existing matrix is static; no fixture schema combines output semantics with expected state delta. |

## Suggested Dependency Order for Planning

1. Test safety and v1 corpus harness/manifest.
2. Durable lifecycle records plus read-only fact loader.
3. Pure shared projection and migration of orientation consumers.
4. Transaction/journal primitive with failure-injection tests.
5. Front door, territory, and early-run contracts.
6. Pause/resume and autopilot contracts.
7. Seal/entomb and maintenance contracts.
8. Canonical YAML changes, alias deletion, generation/pruning, full parity and semantic corpus run.

This ordering makes the semantic corpus an executable contract before the riskiest mutations change, and it keeps generated surface updates until the runtime command vocabulary is stable.

## Acceptance Search Gates

The plan should include repository-wide, current-product checks for:

- public `pause-colony` and `resume-colony` command/alias/help entries;
- readers calling the mutating active-state loader or `ensureLegacySessionMirror`;
- forced closure rendered as ordinary completion;
- `entomb` described as a mandatory next step;
- generated files whose managed source no longer exists;
- broad branch/worktree cleanup prefixes in test infrastructure.

Search results in historical Dreams, archived events, fixtures intentionally testing legacy input, and migration diagnostics are not automatically defects. Classify them by ownership and purpose.

## Metadata

**Analog search scope:** `cmd/`, `pkg/colony/`, `pkg/storage/`, `.aether/commands/`, `.aether/docs/`, `.claude/`, `.opencode/`

**Strong analog families:** 5 — read-only truth loading, pure typed projection, reversible single-file mutation, host manifest/finalizer authority, causal black-box testing

**Exact/self-match path families:** 11

**Role/partial-match path families:** 11

**Complete analog gaps:** 5 capabilities, each with a partial source named above

**Project guidance applied:** root `AGENTS.md`, `cmd/AGENTS.md`; project-local `.agents/skills/to-prd/SKILL.md` was indexed but is not applicable to pattern mapping

**Pattern extraction date:** 2026-09-03
