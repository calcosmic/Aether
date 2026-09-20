# Phase 189: Complete Worker Contract - Pattern Map

**Mapped:** 2026-08-20
**Files analyzed:** 3 criteria, ~14 primary files touched or read, 0 net-new source files (one
net-new test file, one net-new constant in an existing file)
**Analogs found:** 3 / 3 (every change has a same-repo precedent to copy; none is invented from
nothing)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `pkg/codex/handoff.go` (modified — new exported const) | data (canonical schema description) | — | `WorkerHandoff`'s own JSON tags in the same file — the const is a prose restatement of tags already there | exact (single source, same file) |
| `pkg/codex/worker.go` (modified — refactor only) | controller (native-dispatch prompt assembly) | request-response | its own pre-existing `renderResponseContract` body, unchanged in output | exact (byte-identical output required) |
| `cmd/codex_build.go` (modified — `findDispatchTasks`, brief render loop, `composeBuildManifestBrief` append) | controller (worker brief composition) | request-response | its own pre-existing `findDispatchTask` (single-task shape) and `composeBuildManifestBrief`'s pheromone-append (wrapper-external-only content shape) | exact (both source shapes already exist in this file) |
| `cmd/codegraph_context.go` (modified, bonus) | consumer (relevance text gathering) | batch (text-signal collection) | same `findDispatchTask` single-task bug as above, same fix | exact |
| `cmd/codex_build_test.go` (modified — new invariant test) | test (proportion/invariant) | batch | `TestBuildWorkerBriefIsMostlyTask`'s own existing shape (dispatch + phase fixture, call renderer, assert on output) | exact |
| `cmd/codex_continue_plan.go` (modified — new manifest fields, dispatch-construction append) | controller (external manifest composition) | request-response | `cmd/codex_build.go`'s `codexBuildManifest.ContextCapsule` (manifest-level field) + `composeBuildManifestBrief`'s append shape | exact (cross-file precedent, same repo) |
| `cmd/codex_continue_plan_test.go` / `cmd/codex_continue_test.go` (modified — new tests) | test (invariant + fixture) | batch | `TestContinuePlanOnlyEnforcesCriterionEvidence`'s existing fixture-then-assert shape; `writeTestPheromones` for signal seeding | exact |
| `.claude/commands/ant/continue.md`, `.claude/commands/ant-continue.md`, `.opencode/commands/ant/continue.md` (modified, byte-identical) | presentation (wrapper instruction prose) | — | `.claude/commands/ant/build.md`'s own "context_capsule (read once) + brief + skill_section" instruction (lines 259, 272) | exact (cross-command precedent, same repo) |
| `.codex/CODEX.md` (modified — 2-line prose fix) | presentation (reference doc) | — | its own existing "Skills"/"Pheromone Signals" sentences, extended by one word each | exact |
| `cmd/continue_wrapper_ceremony_test.go` (modified — new required-substring assertions) | test (contract/substring) | batch | its own existing `TestContinueWrapperCeremonyContract` `required`/`inOrder` slice shape | exact |

## Pattern Assignments

### The single-task-only brief lookup (the bug, and its two call sites)

**Source of the bug** (`cmd/codex_build.go:2829-2839`, `findDispatchTask`):
```go
func findDispatchTask(phase colony.Phase, dispatch codexBuildDispatch) *colony.Task {
	if dispatch.TaskID == "" {
		return nil
	}
	for i := range phase.Tasks {
		if buildTaskID(phase.Tasks[i], i) == dispatch.TaskID {
			return &phase.Tasks[i]
		}
	}
	return nil
}
```
Both call sites (`cmd/codex_build.go:2708`, `cmd/codegraph_context.go:113`) use this to fetch
exactly one task, even when `dispatch.CoveredTaskIDs` names several.

**The existing, ALREADY-correct helper to build on** (`cmd/codex_build.go:2167-2184`,
`dispatchCoveredTaskIDs` — do not reimplement this logic):
```go
func dispatchCoveredTaskIDs(dispatch codexBuildDispatch) []string {
	if len(dispatch.CoveredTaskIDs) > 0 {
		out := make([]string, 0, len(dispatch.CoveredTaskIDs))
		for _, id := range dispatch.CoveredTaskIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	}
	if taskID := strings.TrimSpace(dispatch.TaskID); taskID != "" {
		return []string{taskID}
	}
	return nil
}
```
`findDispatchTasks` (new, plural) should be a thin wrapper: call `dispatchCoveredTaskIDs(dispatch)`,
then resolve each ID against `phase.Tasks` using the exact same `buildTaskID(phase.Tasks[i], i)`
matching `findDispatchTask` already does — just for every ID, not the first.

**The merge-numbering convention to mirror** (`cmd/codex_build.go:1077-1091`,
`mergeDispatchInto` — already does per-task numbering for `Task`, the pattern the new
Constraints/Hints/Task-Success-Criteria loop should visually match):
```go
func mergeDispatchInto(target *codexBuildDispatch, next codexBuildDispatch) {
	if len(target.CoveredTaskIDs) == 0 {
		target.CoveredTaskIDs = []string{target.TaskID}
		target.Task = "1. " + strings.TrimSpace(target.Task)
	}
	target.CoveredTaskIDs = append(target.CoveredTaskIDs, next.TaskID)
	target.Task = strings.TrimSpace(target.Task) + "\n" +
		fmt.Sprintf("%d. %s", len(target.CoveredTaskIDs), strings.TrimSpace(next.Task))
	...
}
```
Note this only numbers when a SECOND task joins (`len(target.CoveredTaskIDs) == 0` guards the
first-step numbering) — the new Constraints/Hints/SuccessCriteria loop should follow the same rule:
no `**Task 1:**` label at all when there is only one covered task, matching today's unnumbered
single-task output exactly.

---

### The wrapper-external-only content layer (the placement fix for the handoff schema)

**Source — the doc comment that already states the rule this phase must follow**
(`cmd/codex_build.go:3108-3117`):
```go
// composeBuildManifestBrief is the single source of the worker prompt that
// ships in the plan-only manifest. It is the base task brief plus the steering
// sections the wrapper has no other channel for: pheromone signals and prior
// worker handoffs.
//
// The Go subprocess path deliberately does NOT use this composition — it
// delivers PheromoneSection and HandoffSection separately through WorkerConfig
// and pkg/codex/prompt.go, so embedding them in the shared renderer would
// duplicate them there. --print-brief uses this composer so what the user
// inspects is exactly what the manifest carries.
func composeBuildManifestBrief(root string, phase colony.Phase, dispatch codexBuildDispatch, startedAt time.Time) string {
	var b strings.Builder
	b.WriteString(renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt))

	if pheromoneSection := resolvePheromoneSection(); pheromoneSection != "" {
		content := strings.TrimSpace(strings.TrimPrefix(pheromoneSection, "### Active Pheromone Signals"))
		if content != "" {
			b.WriteString("\n## Pheromone Signals\n\n")
			b.WriteString(content)
			b.WriteString("\n")
		}
	}
	...
	return b.String()
}
```
The handoff-schema note is exactly the same class of content (present via a separate channel on the
native path, absent entirely on the wrapper-external path) — append it here, following the same
`if content != ""` / heading-or-inline-sentence shape already used for pheromone signals just above
it. Update the doc comment's first paragraph to also name "the handoff/return schema" alongside
"pheromone signals and prior worker handoffs."

**The proof that the native path already has a separate channel** (`cmd/codex_build.go:1698-1711`,
inside `executeCodexBuildDispatches`):
```go
workerDispatch := codex.WorkerDispatch{
	...
	TaskBrief:         renderCodexBuildWorkerBrief(root, phase, dispatch, startedAt),
	ContextCapsule:    capsule,
	HandoffSection:    dispatch.HandoffSection,
	...
	PheromoneSection:  pheromoneSection,
	...
}
```
`TaskBrief` here comes from `renderCodexBuildWorkerBrief` directly (NOT `composeBuildManifestBrief`)
— confirming that whatever `composeBuildManifestBrief` adds on top never reaches the native path
redundantly, while whatever `renderCodexBuildWorkerBrief` itself gains reaches BOTH paths. This is
the mechanical reason D-04 places the handoff-schema note in the wrapper (composed) layer, not the
shared renderer.

**The identical shape on the continue side** (`cmd/codex_continue_plan.go:290-340`,
`plannedExternalContinueDispatches` — confirmed its only caller is `runCodexContinuePlanOnly`,
line 141, i.e. genuinely external-only):
```go
dispatches = append(dispatches, codexContinueExternalDispatch{
	...
	Brief:         renderCodexContinueWatcherBrief(root, phase, manifest, verification.Steps, verification.Claims, verification.Watcher, workerTimeout),
	SkillSection:  watcherSkillAssignment.Section,
	...
})
...
dispatches = append(dispatches, codexContinueExternalDispatch{
	...
	Brief:         renderCodexContinueReviewBrief(root, phase, manifest, verification, assessment, spec),
	SkillSection:  assignment.Section,
	...
})
```
Wrap each `renderCodexContinueWatcherBrief(...)`/`renderCodexContinueReviewBrief(...)` call with a
small helper (e.g. `continueExternalBriefWithHandoffSchema(rendered string) string`) that appends
the same `codex.HandoffFieldsSummary`-based sentence, mirroring `composeBuildManifestBrief`'s shape
exactly. Do not modify `renderCodexContinueWatcherBrief`/`renderCodexContinueReviewBrief` themselves
— they are shared with continue's native/internal dispatch path
(`plannedContinueReviewDispatches`/`plannedContinueWatcherDispatch`, `cmd/codex_continue.go:1387,
1781`), which already gets the schema via the same `AssembleHostedPrompt`+`renderResponseContract`
composition build's native path uses.

---

### The manifest-level "resolve once" field (the pattern for continue's new capsule/pheromone)

**Source** (`cmd/codex_build.go:114` — the struct field; `cmd/codex_build.go:1685,1688` — resolved
once, reused for every dispatch in the same build):
```go
// codexBuildManifest:
ContextCapsule            string                                `json:"context_capsule,omitempty"`
```
```go
capsule := resolveCodexWorkerContext()
...
pheromoneSection := resolvePheromoneSection()
```

**The gap to close** (`cmd/codex_continue_plan.go:43-67`, `codexContinuePlanManifest` — no
equivalent field at all today):
```go
type codexContinuePlanManifest struct {
	Phase                     int                             `json:"phase"`
	...
	Dispatches                []codexContinueExternalDispatch `json:"dispatches"`
	DispatchMode              string                          `json:"dispatch_mode"`
	...
}
```
Add `ContextCapsule string \`json:"context_capsule,omitempty"\`` and
`PheromoneSection string \`json:"pheromone_section,omitempty"\`` to this struct, and inside
`runCodexContinuePlanOnly` (`cmd/codex_continue_plan.go:142-161`, the `plan :=
codexContinuePlanManifest{...}` literal), set both by calling `resolveCodexWorkerContext()` and
`resolvePheromoneSection()` once each — the same two functions build already calls, already
imported into this package, no new resolver needed.

---

### The wrapper instruction to extend (build's phrasing, copied to continue)

**Source** (`.claude/commands/ant/build.md:259,272`):
```
Read `dispatch_manifest.context_capsule` ONCE from the manifest — it is not per-dispatch data,
reuse the same value for every worker this build spawns — and prepend it VERBATIM ahead of the
brief, then append `dispatch.skill_section` when present.
...
5. The worker's prompt = `dispatch_manifest.context_capsule` (read once, prepended verbatim) + the
brief read VERBATIM from `dispatch.brief_path` when present (falling back to `dispatch.brief`
inline when it is absent) + `dispatch.skill_section` when present. Nothing else, nothing invented.
```

**The gap to close** (`.claude/commands/ant/continue.md:133-134,168,176`):
```
**Reads:** the manifest returned by `aether host continue --dry-run`; each dispatch's
runtime-provided `brief` verbatim.
...
Spawn reviewers as visible live Task/subagent panels. Do not set `run_in_background`. Pass each
dispatch's runtime-provided `brief` verbatim.
...
5. Pass each dispatch's runtime-provided `brief` verbatim.
```
Rewrite to the same "capsule once + brief + skill_section" formula, adding `pheromone_section`
alongside capsule (both manifest-level, both read once): e.g. "The worker's prompt =
`continue_manifest.context_capsule` + `continue_manifest.pheromone_section` (both read once,
prepended verbatim) + the reviewer's own `brief` + `dispatch.skill_section` when present." Apply the
identical text to `.claude/commands/ant-continue.md` and `.opencode/commands/ant/continue.md` —
these three files are byte-identical today and must remain so (see Shared Patterns below).

## Shared Patterns

### One canonical constant, multiple readers converge on it
**Source:** Phase 188's D-02/D-04 (`loadMiddenFile`, `continueSupersededResult`) — "all readers
converge on one canonical helper," not independent hand-copies.
**Apply to:** `codex.HandoffFieldsSummary` (`pkg/codex/handoff.go`), referenced from
`renderResponseContract` (unchanged output), `composeBuildManifestBrief` (new), and
`plannedExternalContinueDispatches`'s dispatch-construction sites (new) — three readers, one source.

### Wrapper-external-only content is added in the composition wrapper, never the shared renderer
**Source:** `composeBuildManifestBrief`'s own pre-existing doc comment and pheromone-append shape.
**Apply to:** the handoff-schema note, for both build (`composeBuildManifestBrief`) and continue
(`plannedExternalContinueDispatches`'s call sites) — never inside
`renderCodexBuildWorkerBrief`/`renderCodexContinueReviewBrief`/`renderCodexContinueWatcherBrief`
themselves.

### Manifest-level fields for colony-wide content, resolved once
**Source:** `codexBuildManifest.ContextCapsule`, resolved once in `executeCodexBuildDispatches`/
`printWorkerBriefs`, reused for every dispatch.
**Apply to:** `codexContinuePlanManifest.ContextCapsule` / `.PheromoneSection` (new).

### Byte-identical wrapper triplet, written once and copied
**Source:** `TestLifecycleFlatMirrorsMatchCanonical` (`cmd/lifecycle_wrapper_contract_test.go:129`)
— confirmed empirically this phase that `.claude/commands/ant/continue.md`,
`.claude/commands/ant-continue.md`, and `.opencode/commands/ant/continue.md` are byte-identical
today.
**Apply to:** all three files receive the SAME new text, in the SAME place, in the same edit — never
hand-diverged.

### A guard/test that finds nothing must fail, not pass
**Source:** Phase 187's and 188's own ratchets.
**Apply to:** the new criterion-1 test must use a REAL multi-task dispatch fixture (not a
zero-task edge case); the new criterion-3 test must assert the manifest fields are actually
non-empty against seeded state/signals, not merely that the struct fields compile.

## No Analog Found

| File/Test | Role | Data Flow | Reason |
|---|---|---|---|
| The duplication-proof tests (asserting `renderCodexBuildWorkerBrief`/`renderCodexContinueReviewBrief` output does NOT contain the handoff-schema sentence, while the composed/external output DOES) | test (negative + positive invariant pair) | batch | No existing test in this codebase asserts a NEGATIVE containment fact about a shared renderer's output to prove a duplication-avoidance design decision — this is new because the design decision (D-04/D-06) is itself new; the closest structural relative is `TestBuildWorkerBriefOmitsPlaybooks` (asserts a forbidden substring is ABSENT), reused as the shape but applied to new content. |

## Metadata

**Analog search scope:** `cmd/codex_build*.go`, `cmd/codex_continue*.go`, `cmd/codegraph_context.go`,
`cmd/build_print_brief.go`, `pkg/codex/*.go`, `.claude/commands/ant/{build,continue}.md`,
`.opencode/commands/ant/continue.md`, `.codex/CODEX.md`
**Files scanned directly (Read or targeted grep):** `cmd/codex_build.go`, `cmd/codex_build_test.go`,
`cmd/codex_build_finalize.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`,
`cmd/codex_continue_plan_test.go`, `cmd/codex_continue_finalize.go`, `cmd/codex_continue_test.go`,
`cmd/codegraph_context.go`, `cmd/build_print_brief.go`, `cmd/lifecycle_wrapper_contract_test.go`,
`cmd/continue_wrapper_ceremony_test.go`, `pkg/codex/handoff.go`, `pkg/codex/worker.go`,
`pkg/codex/worker_test.go`, `pkg/codex/dispatch.go`, `.claude/commands/ant/build.md`,
`.claude/commands/ant/continue.md`, `.opencode/commands/ant/build.md`,
`.opencode/commands/ant/continue.md`, `.codex/CODEX.md`
**Pattern extraction date:** 2026-08-20
