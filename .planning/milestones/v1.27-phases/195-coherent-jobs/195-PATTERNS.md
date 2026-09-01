# Phase 195: Coherent Jobs - Pattern Map

**Mapped:** 2026-08-27
**Files analyzed:** 38 new/modified files
**Analogs found:** 38 / 38

## Scope Extraction

The file set below combines the explicit recommended structure in
`195-RESEARCH.md`, the named integration and test files in its verification
matrix, and the wrapper/documentation parity surfaces required by `AGENTS.md`.
Deferred cost display, closing-card redesign, and spec-builder work are excluded.

For a newcomer: Phase 195 is not introducing a second build system. It is moving
the existing “several tasks, one worker” rule earlier in the same build pipeline,
then making the existing completion receipt precise enough to credit only the
tasks that have proof.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/coherent_jobs.go` | service | batch / transform | `cmd/codex_build.go:1132` | role-match |
| `cmd/coherent_jobs_test.go` | test | batch / transform | `cmd/dispatch_coalesce_test.go:42` | exact |
| `cmd/coherent_job_receipts.go` | utility | request-response / transform | `cmd/codex_build_finalize.go:1245` | role-match |
| `cmd/coherent_job_receipts_test.go` | test | request-response / transform | `cmd/wrapper_bundled_completion_test.go:119` | exact |
| `cmd/codex_build.go` | controller | batch / event-driven | current file | exact |
| `cmd/codex_build_worktree.go` | service | file-I/O / event-driven | current file | exact |
| `cmd/codex_build_finalize.go` | controller | request-response / file-I/O | current file | exact |
| `cmd/build_attempt.go` | model / service | file-I/O / event-driven | current file | exact |
| `cmd/ceremony_team_checkin.go` | component | transform / request-response | current file | exact |
| `cmd/codex_workflow_cmds.go` | controller / route | request-response | current file | exact |
| `cmd/contract_schema.go` | utility / config | transform / file-I/O | current file | exact |
| `pkg/codex/worker.go` | model / provider | request-response | current file | exact |
| `pkg/codex/handoff.go` | model / utility | transform | current file | exact |
| `pkg/codex/dispatch.go` | service / model | event-driven / batch | current file | exact |
| `.aether/schemas/completion-packet.schema.json` | config | request-response | `cmd/contract_schema.go:52` (generator) | exact |
| `cmd/dispatch_coalesce_test.go` | test | batch / transform | current file | exact |
| `cmd/merged_dispatch_task_credit_test.go` | test | request-response / CRUD | current file | exact |
| `cmd/wrapper_bundled_completion_test.go` | test | request-response / CRUD | current file | exact |
| `cmd/codex_build_test.go` | test | batch / request-response | current file | exact |
| `cmd/codex_build_worktree_test.go` | test | file-I/O / event-driven | current file | exact |
| `cmd/codex_build_finalize_test.go` | test | request-response / file-I/O | current file | exact |
| `cmd/build_attempt_test.go` | test | file-I/O / event-driven | current file | exact |
| `cmd/build_attempt_external_test.go` | test | request-response / file-I/O | current file | exact |
| `cmd/ceremony_team_checkin_test.go` | test | transform / request-response | current file | exact |
| `cmd/forced_reviewer_waiver_test.go` | test | request-response / CRUD | current file | exact |
| `cmd/orchestrator_boundary_questions_test.go` | test | request-response / CRUD | current file | exact |
| `cmd/contract_schema_test.go` | test | transform / file-I/O | current file | exact |
| `.aether/commands/build.yaml` | config | request-response | current file | exact |
| `.claude/commands/ant/build.md` | config / route | request-response | current file | exact |
| `.claude/commands/ant-build.md` | config / route | request-response | `.claude/commands/ant/build.md` | exact |
| `.opencode/commands/ant/build.md` | config / route | request-response | `.claude/commands/ant/build.md` | exact |
| `.aether/skills/colony/aether-colony-build-cycle/SKILL.md` | config / route | request-response | current file | exact |
| `cmd/command_guide.go` | config / controller | request-response | current file | exact |
| `cmd/command_guide_test.go` | test | request-response | current file | exact |
| `cmd/lifecycle_wrapper_contract_test.go` | test | file-I/O / transform | current file | exact |
| `cmd/parity_test.go` | test | file-I/O / transform | current file | exact |
| `CLAUDE.md` | config / documentation | request-response | current file, lines 237-250 | exact |
| `cmd/cli_flag_audit_test.go` | test | file-I/O / transform | current file | exact |

## Pattern Assignments

### `cmd/coherent_jobs.go` (service, batch / transform)

**Apply also to:** `cmd/codex_build.go`, `cmd/coherent_jobs_test.go`,
`cmd/dispatch_coalesce_test.go`, and the grouping cases in `cmd/codex_build_test.go`.

**Primary analogs:** `cmd/codex_build.go`, `pkg/colony/cycle.go`,
`cmd/queen_judgement.go`, and `cmd/dispatch_coalesce_test.go`.

**Imports pattern** (`cmd/codex_build.go:3-20`): keep graph policy in `cmd`, use
the shared `colony` domain types, and return errors instead of writing state.

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)
```

**Global graph guard** (`pkg/colony/cycle.go:33-103`): call this before any job,
manifest, attempt, or worktree exists. Preserve its typed `MissingDepError` and
`CycleError`; wrap them with the phase/task names and a plain-English repair.

```go
func DetectCycles(phases []Phase) error {
	adj := make(map[string][]string)
	known := make(map[string]bool)
	// ... first validate every DependsOn reference ...
	// ... then use a three-color DFS and return &CycleError{Tasks: cycle} ...
	return nil
}
```

**Existing merge contract** (`cmd/codex_build.go:1150-1231`): preserve the
current union behavior—covered IDs, numbered instructions, and declared paths—
but move the decision to task components before waves.

```go
func mergeDispatchInto(target *codexBuildDispatch, next codexBuildDispatch) {
	if len(target.CoveredTaskIDs) == 0 {
		target.CoveredTaskIDs = []string{target.TaskID}
		target.Task = "1. " + strings.TrimSpace(target.Task)
	}
	target.CoveredTaskIDs = append(target.CoveredTaskIDs, next.TaskID)
	target.Task = strings.TrimSpace(target.Task) + "\n" +
		fmt.Sprintf("%d. %s", len(target.CoveredTaskIDs), strings.TrimSpace(next.Task))
	target.DeclaredPaths = uniqueSortedStrings(
		append(append([]string{}, target.DeclaredPaths...), next.DeclaredPaths...))
}
```

**Meaningful-path input** (`cmd/codex_build_worktree.go:54-85`): derive shared
file edges from the same normalized task declarations the worktree owner uses.
Do not infer paths from arbitrary prose.

```go
func declaredPathsForTask(task colony.Task) []string {
	paths := make([]string, 0, len(task.Hints)+1)
	// evidence artifacts and exact repo-relative file hints only
	return uniqueSortedStrings(paths)
}
```

**Soft-limit pattern** (`cmd/build_print_brief.go:500-528`): reference the named
allowance. Do not introduce a task-count cap or duplicate `6000` in a new file.

```go
const briefTaskContentAllowanceChars = 6000
```

**Additive Queen proposal pattern** (`cmd/queen_judgement.go:120-165,
202-218,284-295`): empty input retains deterministic behavior; reject only the
bad proposal/worker and keep the remaining accepted set.

```go
func queenApplyJudgement(...) queenCasteJudgement {
	normalized, unknown := normalizeProposedCastes(proposed)
	if len(normalized) == 0 {
		return queenCasteJudgement{Final: deterministic, Source: "deterministic"}
	}
	// Refuse one invalid/unexplained member; do not discard the rest.
	return queenCasteJudgement{Final: final, Refused: refused, Source: "queen"}
}
```

Reuse `resolveCasteName` (`cmd/queen_judgement.go:354-382`) for `owner_caste`.
Parse each repeatable `--job-proposal` as one JSON object; do not copy the
comma-splitting `--castes` encoding for a structured multi-field contract.

**Testing pattern** (`cmd/dispatch_coalesce_test.go:42-72,75-97,99-130`): use
real `colony.Task` fixtures, assert the final dispatch count and every
`CoveredTaskID`, and retain a negative fan-out case.

```go
dispatches := waveDispatchesOnly(
	plannedBuildDispatchesForSelectionWithState(
		phase, colony.ColonyState{}, nil, colony.VerificationDepthStandard))
if len(dispatches) != 1 { t.Fatalf("... got %d workers, want 1", len(dispatches)) }
if len(dispatches[0].CoveredTaskIDs) != 3 { t.Fatalf("...") }
```

Add tables for proposal order refusal, safe-group preservation, selected-task
scope, incidental paths, cross-caste defaults, explicit cross-caste ownership,
non-consecutive dependency components, and deterministic soft-limit splits.

---

### `cmd/coherent_job_receipts.go` (utility, request-response / transform)

**Apply also to:** `cmd/coherent_job_receipts_test.go`, `cmd/codex_build.go`,
`cmd/codex_build_finalize.go`, `pkg/codex/worker.go`, `pkg/codex/handoff.go`,
`pkg/codex/dispatch.go`, `cmd/merged_dispatch_task_credit_test.go`,
`cmd/wrapper_bundled_completion_test.go`, and `cmd/codex_build_finalize_test.go`.

**Primary analogs:** `cmd/codex_build_finalize.go`, `pkg/codex/handoff.go`, and
`cmd/wrapper_bundled_completion_test.go`.

**Additive result model** (`pkg/codex/worker.go:66-87`): add optional receipts
to `WorkerResult` and the corresponding `workerClaims` JSON struct; preserve all
existing fields and single-task callers.

```go
type WorkerResult struct {
	WorkerName    string
	TaskID        string
	Status        string
	Summary       string
	FilesCreated  []string
	FilesModified []string
	TestsWritten  []string
	Handoff       WorkerHandoff
}
```

Mirror the additive field on `codexExternalBuildWorkerResult`
(`cmd/codex_build_finalize.go:62-110`). Keep `covered_task_ids` as assignment
coverage; use a distinct runtime-owned `completed_task_ids` for validated
partial credit.

**Validation/normalization pattern** (`pkg/codex/handoff.go:54-106`): normalize
first through one shared function and reject invalid enum values explicitly.

```go
func ValidateWorkerHandoff(h WorkerHandoff) error {
	status := strings.ToLower(strings.TrimSpace(h.VerificationStatus))
	switch status {
	case "", "pass", "passed", "fail", "failed", "partial", "not_run", "not-run", "not run", "unknown":
	default:
		return fmt.Errorf("verification_status must be pass, fail, partial, not_run, or unknown")
	}
	return nil
}
```

Split the shared receipt boundary into two named stages. Stage 1,
`admitCoherentJobTaskReceipts`, returns all named structural violations plus
candidate task claims and normalized sync paths; it validates unique task
membership, successful status, root-contained claim paths, non-empty summary,
passing concrete verification, and task-specific manifest requirements, but it
does not read root artifacts or expose `CompletedTaskIDs`. Stage 2,
`finalizeCoherentJobTaskReceiptEvidence`, consumes only admitted candidates
after their files exist in root, attaches root-computed artifact evidence, and
returns task claims plus `CompletedTaskIDs` only for successful evidence.

**Whole-packet trust-boundary pattern** (`cmd/codex_build_finalize.go:120-145,
1245-1283`): accumulate wrapper-correctable problems as structured violations;
reserve `error` for internal failures. Make the direct and external lanes call
the same receipt admission and root-evidence finalization stages.

```go
type completionContractError struct {
	Violations []contractViolation
}

func mergeExternalBuildResults(...) ([]codexBuildDispatch, []contractViolation, error) {
	var violations []contractViolation
	resultByName := make(map[string]codexExternalBuildWorkerResult, len(results))
// collect every correctable problem before returning; no completion credit here
	return dispatches, violations, nil
}
```

The direct/external in-repo lanes call admission and finalization consecutively.
The worktree lane stores the same admission result, synchronizes only its
candidate paths, and then calls the same finalizer. Do not create lane-specific
validators or overload provisional admission as completion credit.

**Credit pattern** (`cmd/codex_build.go:2421-2461`): keep whole-success
backward compatibility, but let failed/interrupted grouped dispatches expose
only the root-evidence-finalizer-produced completed set.

```go
status := strings.TrimSpace(dispatch.Status)
if status != "completed" && !isNoChangeExternalBuildStatus(status) {
	continue
}
for _, taskID := range dispatchCoveredTaskIDs(dispatch) {
	completed[taskID] = struct{}{}
}
```

Do not change the successful branch above. Add a separate partial branch that
reads validated `CompletedTaskIDs`; never infer it from touched files or from
`CoveredTaskIDs`.

**Task claim pattern** (`cmd/codex_build.go:231-246` and
`pkg/codex/dispatch.go:60-74`): project every finalized receipt claim under its
own task ID, not only the grouped dispatch’s primary task.

```go
type codexBuildTaskClaim struct {
	TaskID        string   `json:"task_id"`
	FilesCreated  []string `json:"files_created,omitempty"`
	FilesModified []string `json:"files_modified,omitempty"`
	TestsWritten  []string `json:"tests_written,omitempty"`
}
```

**Adversarial test pattern** (`cmd/wrapper_bundled_completion_test.go:178-253,
308-364`): assert named rules for unknown, duplicate, fabricated, failed, and
unevidenced claims, then assert the final task state—not merely an intermediate
decision record. Add path-laundering and task-requirement mismatch cases.

---

### `cmd/build_attempt.go` (model / service, file-I/O / event-driven)

**Apply also to:** `cmd/build_attempt_test.go`,
`cmd/build_attempt_external_test.go`, and retry creation from receipt
reconciliation.

**Analog:** current attempt journal.

**Record pattern** (`cmd/build_attempt.go:129-175`): add parent attempt/job
references as optional fields so existing attempt JSON remains readable.

```go
type buildAttemptRecord struct {
	SchemaVersion int                      `json:"schema_version"`
	ID            string                   `json:"id"`
	Phase         int                      `json:"phase"`
	Status        string                   `json:"status"`
	Dispatches    []codexBuildDispatch     `json:"dispatches"`
	History       []buildAttemptTransition `json:"history"`
}
```

**Append-only creation and transition** (`cmd/build_attempt.go:185-300`): create
a new attempt file plus latest pointer, then atomically append transitions. A
retry may link to the original, but must not rewrite the original dispatch,
receipts, claims, or completion digest.

```go
attemptRel := filepath.ToSlash(filepath.Join(
	"build", fmt.Sprintf("phase-%d", phaseNum), "attempts", attemptID+".json"))
if err := store.SaveJSON(attemptRel, record); err != nil { return "", err }

record.History = append(record.History, buildAttemptTransition{
	Status: status, Timestamp: now.Format(time.RFC3339Nano), Summary: strings.TrimSpace(summary),
})
```

Tests should load both files and prove the parent is byte/JSON-equivalent after
the child retry is created; the child must contain only uncredited tasks and
must revalidate dependencies with credited tasks treated as satisfied.

---

### `cmd/codex_build_worktree.go` (service, file-I/O / event-driven)

**Apply also to:** `pkg/codex/dispatch.go`,
`cmd/codex_build_worktree_test.go`, and grouped dispatch conversion in
`cmd/codex_build.go`.

**Analog:** current worktree wave ownership and reconciliation.

**Ownership guard** (`cmd/codex_build_worktree.go:94-115`): keep overlap
rejection for distinct jobs in one wave. Grouping must happen before this
function so intentional same-file tasks arrive as one owner with a unioned path
set.

```go
func validateDeclaredWorktreeOwnership(dispatches []codex.WorkerDispatch) error {
	declaredByWave := map[int]map[string]string{}
	for _, dispatch := range dispatches {
		for _, path := range dispatch.DeclaredPaths {
			if previous, ok := owners[path]; ok && previous != dispatch.TaskID {
				return fmt.Errorf("worktree declared ownership conflict: ...")
			}
		}
	}
	return nil
}
```

Remove the current exclusion at `cmd/codex_build.go:1297-1305`; one grouped job
must become one `codex.WorkerDispatch`/worktree/session with unioned declared
paths.

**Admission-sync-finalization pattern** (`cmd/codex_build_worktree.go:382-479`):
run structural receipt admission first to obtain candidate paths without
credit, sync those paths to root, then run shared root-evidence finalization
before journaling credit. Failed/uncredited edits retain an orphaned worktree.

```go
case accepted[i]:
	if syncErr := syncWorktreeChangesToRoot(root, session.AbsPath, outcome.touched); syncErr != nil {
		dr.Status = "failed"
		preserveWorktree = true
	}
case dr.Status != "completed" && session != nil:
	preserveWorktree = true
```

For a partial receipt, replace `outcome.touched` with the normalized union of
stage-1 admission `SyncPaths` and reject unattributed touched paths. Call
`finalizeCoherentJobTaskReceiptEvidence` after `syncWorktreeChangesToRoot`; only
that output may populate `CompletedTaskIDs`.

**Safe path-I/O pattern** (`cmd/codex_build_worktree.go:950-1005`): retain
`filepath.Clean`, reject absolute/`..` paths, and use repository-relative paths
for receipt-scoped sync.

---

### One-worker check-in policy and compact summary

**Apply to:** `cmd/codex_workflow_cmds.go`, `cmd/ceremony_team_checkin.go`,
`cmd/ceremony_team_checkin_test.go`, `cmd/forced_reviewer_waiver_test.go`,
`cmd/orchestrator_boundary_questions_test.go`, and `cmd/codex_build_test.go`.

**Primary analogs:** `cmd/codex_workflow_cmds.go` and
`cmd/ceremony_team_checkin.go`.

**Flag/input pattern** (`cmd/codex_workflow_cmds.go:130-175,1381-1388`): read all
policy inputs once, reject `--checkin --no-checkin` before plan-only opens an
attempt, and return machine-readable decision fields from the runtime.

```go
noCheckin, _ := cmd.Flags().GetBool("no-checkin")
result["checkin_requested"] = !noCheckin

buildCmd.Flags().Bool("no-checkin", false, "Skip the wrapper's pre-spawn team check-in pause ...")
```

Add the symmetric `checkin` flag, move the decision out of the one-line
assignment above, and evaluate in this order: explicit non-interactive mode,
explicit `--checkin`, pending owner decision, exactly one implementation
dispatch, then multi-worker default. A flag conflict must return before
`runCodexBuildPlanOnlyWithOptions`.

**Pending-decision guard** (`cmd/ceremony_team_checkin.go:123-189`): reuse the
live forced-reviewer/waiver derivation; do not use only `len(dispatches) == 1`.
Also include unanswered orchestrator boundary questions.

**Rendering pattern** (`cmd/ceremony_team_checkin.go:39-86,88-121,222-233`):
derive structured result fields and visual text together from the manifest.
Add a separate compact-summary renderer; do not call the full check-in card and
discard its question.

```go
result := map[string]interface{}{
	"workflow": workflow,
	"required": required,
	"optional": optional,
	"reasons":  reasons,
}
return result, b.String()
```

The compact result/visual must name the deterministic worker, every covered
task, the accepted relationship and benefit, and why no approval is required.

**Test to replace** (`cmd/ceremony_team_checkin_test.go:318-352`): invert
`TestOneWorkerTeamStillPauses` into a runtime policy test. Retain separate tests
that the full renderer still works when explicitly forced or when a pending
decision exists. Add a table covering `--checkin`, `--no-checkin`, both flags,
autopilot, one worker, two workers, forced-reviewer waiver, and generic boundary
question; the conflict row must assert no attempt/manifest mutation.

---

### Completion schema and contract files

**Apply to:** `cmd/contract_schema.go`, `cmd/contract_schema_test.go`, and
`.aether/schemas/completion-packet.schema.json`.

**Analog:** current reflected schema generator.

**Generation pattern** (`cmd/contract_schema.go:52-76`): change Go structs,
regenerate, and commit the generated JSON. Never hand-edit the schema.

```go
func generateCompletionPacketSchemaBytes() ([]byte, error) {
	reflector := &invopopjsonschema.Reflector{}
	schema := reflector.Reflect(&codexExternalBuildCompletion{})
	buf, err := json.MarshalIndent(schema, "", "  ")
	return append(buf, '\n'), err
}
```

**Fail-closed runtime validator** (`cmd/contract_schema.go:123-168`): compile
from in-memory generated bytes, walk all leaf errors, and return a violation per
field. The on-disk schema must not be able to weaken runtime validation.

**Drift workflow** (`cmd/contract_schema.go:255-327`): use
`go run ./cmd/aether contract-schema --write` during implementation and
`go run ./cmd/aether contract-schema --check` at every relevant gate.

---

### Wrapper, Codex-skill, guide, and documentation parity

**Apply to:** `.aether/commands/build.yaml`, the three build wrapper copies,
`.aether/skills/colony/aether-colony-build-cycle/SKILL.md`,
`cmd/command_guide.go`, `cmd/command_guide_test.go`,
`cmd/lifecycle_wrapper_contract_test.go`, `cmd/parity_test.go`,
`cmd/cli_flag_audit_test.go`, and `CLAUDE.md`.

**Authority pattern** (`.aether/commands/build.yaml:3-16,24-31,62-80`): Go owns
the accepted jobs, reasons, check-in decision, completion credit, and state.
Wrappers propose, render, spawn, and submit receipts.

```yaml
runtime:
  manifest_command: "aether build $ARGUMENTS --plan-only"
  finalizer_command: "aether build-finalize $ARGUMENTS --completion-file <go_owned_completion_path>"
codex_orchestration:
  skill: "aether-colony-build-cycle"
  guide: "aether command-guide build --platform codex"
  drift_guard: "Update this YAML, Claude/OpenCode wrappers, the Codex skill, and cmd/command_guide.go together."
```

**Current wrapper check-in seam** (`.claude/commands/ant/build.md:250-276`):
replace the claim that false only means `--no-checkin`. When false because of
the automatic fast path, render the runtime-provided compact summary and
continue without `AskUserQuestion`. When true, keep the existing full card and
waiver flow unchanged.

```markdown
**Reads:** the manifest file ... and `result.checkin_requested` ...

If `checkin_requested` is false (`--no-checkin` was passed), skip this stage entirely.
```

**Codex guide pattern** (`cmd/command_guide.go:301-332`): update the build
`PreSteps` to consume the runtime’s accepted/refused/repaired job decisions and
check-in reason; do not teach Codex to create authoritative manifest groups.

Keep these wrapper files byte-identical after the managed header/frontmatter:

- `.claude/commands/ant/build.md`
- `.claude/commands/ant-build.md`
- `.opencode/commands/ant/build.md`

`CLAUDE.md:237-250` currently states that every one-worker build pauses and that
only `--no-checkin` skips it; update that statement to the Phase 195 policy.

Run the existing parity families rather than creating another checker:

- `cmd/lifecycle_wrapper_contract_test.go` for canonical wrapper structure and
  Claude/OpenCode equality.
- `cmd/parity_test.go` for YAML/wrapper/guide and flag parity.
- `cmd/command_guide_test.go` for Codex command-guide contents.
- `cmd/cli_flag_audit_test.go` for flags referenced by wrapper/docs prose.

## Shared Patterns

### Runtime authority and additive compatibility

**Source:** `cmd/queen_judgement.go:120-165`, `cmd/codex_build.go:22-73`  
**Apply to:** proposal, job, manifest, and receipt fields.

Empty proposal/receipt fields must preserve the pre-Phase-195 single-task and
whole-success behavior. Keep `task_id` as the primary compatibility identity,
`covered_task_ids` as assignment scope, and add distinct fields for job metadata
and validated partial completion.

### Fail closed globally, repair locally

**Source:** `pkg/colony/cycle.go:33-103`, `cmd/queen_judgement.go:202-218`  
**Apply to:** graph preflight and proposal validation.

A missing dependency or cycle blocks dispatch before side effects. A malformed
individual Queen group is refused by name and only its members return to the
automatic planner; accepted groups stay accepted.

### Root-backed evidence before credit

**Source:** `cmd/codex_build_worktree.go:382-479`,
`cmd/build_attempt.go:996-1015`  
**Apply to:** native/external receipts, in-repo/worktree modes.

Worker-supplied paths and hashes are claims. Structural admission normalizes and
binds candidate paths without credit. Sync admitted worktree files to root,
then call shared root-evidence finalization before changing task status.

### Atomic and append-only persistence

**Source:** `cmd/build_attempt.go:185-300`  
**Apply to:** partial terminal outcome and unfinished-only retry.

Use `SaveJSON` for a new attempt and `UpdateJSONAtomically` only to append the
current attempt’s transitions. Never rewrite the original grouped job into its
retry.

### Plain-English errors and summaries

**Source:** `cmd/ceremony_team_checkin.go:88-121`,
`cmd/contract_schema.go:255-327`  
**Apply to:** grouping refusal, dependency cycle, fast-path summary, and flag
conflict.

Name the proposal/job, offending task, unmet dependency or cycle, and the exact
repair. Avoid graph terminology without also explaining what the user must fix.

## No Analog Found

No file is without a usable analog. The two new implementation modules do not
have same-name predecessors, but their policies already exist in narrower form:
`cmd/codex_build.go` for grouping and `cmd/codex_build_finalize.go` plus
`pkg/codex/handoff.go` for evidence validation. Extend those contracts rather
than inventing a parallel engine.

## Metadata

**Analog search scope:** `cmd/`, `pkg/colony/`, `pkg/codex/`, `.aether/commands/`,
`.aether/skills/`, `.aether/schemas/`, `.claude/commands/`, `.opencode/commands/`,
and root platform docs.  
**Strong analogs inspected:** 12 implementation/contract files and 5 focused
test/parity families.  
**Primary analog set:** `cmd/codex_build.go`, `pkg/colony/cycle.go`,
`cmd/queen_judgement.go`, `cmd/codex_build_finalize.go`,
`cmd/build_attempt.go`, `cmd/codex_build_worktree.go`,
`cmd/ceremony_team_checkin.go`, `pkg/codex/handoff.go`, and
`cmd/contract_schema.go`.  
**Pattern extraction date:** 2026-08-27
