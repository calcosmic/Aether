---
schema_version: 1
open_count: 3
waived_count: 0
fixed_count: 5
total_count: 8
last_updated: 2026-08-30T14:49:33.437Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 193 | unmet-truth | cmd/caste_relevance.go |  | Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal (isAlwaysRequired computes the same required-caste independently on both flows); 193-02 only removed the watcher's implicit build-side dispatch. Flagged in 193-02-PLAN.md frontmatter as unclassified/not-auto-resolved; scoped to Phase 194 (moves the required-caste floor). | fixed |  | 2026-08-22T13:55:31.245Z | 2026-08-23T16:04:14.666Z |
| 2 | 196 | deviation | pkg/codex/worker.go |  | assembledPromptChars deleted outside the plan's files_modified; dead after the fallback removal | open |  | 2026-08-28T10:11:03.801Z |  |
| 3 | 197 | stub | cmd/codex_visuals.go |  | The pause card's Handoff line still reads 'Colony handoff saved for later resumption' - repo-invented word 'colony' with no plain-English gloss. Found by TestNextActionCardSpeaksPlainEnglish; the plain-English check was scoped to the new next-action card because rewording the ten legacy cards is plans 197-04 and 197-06's declared scope. | fixed |  | 2026-08-28T19:48:11.715Z | 2026-08-29T00:00:00.000Z |
| 4 | 197 | deviation | cmd/next_action.go |  | 197-02 edited cmd/next_action.go, which is outside its declared files_modified, to change one word in the failed-phase recommendation ('Running it again' -> 'The next step is to retry it') so the pre-existing TestWorkflowSuggestionsFailedPhase assertion on the word 'retry' kept passing without weakening it. | open |  | 2026-08-28T19:48:15.796Z |  |
| 5 | 198 | deviation | cmd/codex_visuals.go |  | renderPlanVisual's confidence branch only type-asserts to map[string]interface{}; both runCodexPlanWithOptions and runCodexPlanFinalize always store confidence as a codexPlanConfidence struct, so the Confidence line never renders on the direct or chat path. Out of scope for 198-04 (prohibited from editing that file, owned by a same-wave plan); needs a dual-type fix mirroring planning_loop's existing struct/map switch. | fixed |  | 2026-08-29T17:49:53.592Z | 2026-08-29T18:28:52.436Z |
| 6 | 198 | deviation | cmd/codex_visuals.go |  | renderPlanVisual never reads result["research_warning"] (which renderResearchFailedWarning's own doc comment says exists "so the omission is durable and visible" when a phase was planned without its research), result["research_failed_phases"] (the phase IDs that fed that warning), or result["gaps"] (a completed plan's own unresolved gaps). Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json) because cmd/codex_visuals.go was owned this wave by sibling plan 198-08. | fixed |  | 2026-08-29T19:15:56.000Z | 2026-08-29T20:05:27.000Z |
| 7 | 198 | deviation | cmd/codex_plan_finalize.go |  | runCodexPlanFinalize never calls closeLifecycleRun (unlike continue-finalize/completeSealRuntime), so the plan finalizer's own suggested result["next"] command never folds into the unified next-action envelope renderLifecycleClosing reads back -- the closing card instead independently resolves a next step from live colony state, which usually matches but is not guaranteed to. Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json, "plan completed" and "plan mid-loop") because cmd/codex_plan_finalize.go was outside that plan's declared files. | fixed |  | 2026-08-29T19:15:56.000Z | 2026-08-29T20:05:27.000Z |
| 8 | 198.2 | deviation | cmd/codex_continue_finalize.go |  | externalContinueReviewReport does not append a 'review wave skipped' narration step when no reviewers were dispatched, unlike the direct lane's runCodexContinueReview -- discovered by 198.2-05's dual-lane outcome-text parity test, out of that plan's scope to fix | open |  | 2026-08-30T14:49:33.437Z |  |

````json
[
  {
    "id": 1,
    "kind": "unmet-truth",
    "phase": "193",
    "file": "cmd/caste_relevance.go",
    "line": null,
    "description": "Probe/auditor/gatekeeper still legitimately double-dispatch on build AND continue with no explicit Queen proposal (isAlwaysRequired computes the same required-caste independently on both flows); 193-02 only removed the watcher's implicit build-side dispatch. Flagged in 193-02-PLAN.md frontmatter as unclassified/not-auto-resolved; scoped to Phase 194 (moves the required-caste floor).",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-22T13:55:31.245Z",
    "resolved_at": "2026-08-23T16:04:14.666Z"
  },
  {
    "id": 2,
    "kind": "deviation",
    "phase": "196",
    "file": "pkg/codex/worker.go",
    "line": null,
    "description": "assembledPromptChars deleted outside the plan's files_modified; dead after the fallback removal",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-28T10:11:03.801Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "stub",
    "phase": "197",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "The pause card's Handoff line still reads 'Colony handoff saved for later resumption' - repo-invented word 'colony' with no plain-English gloss. Found by TestNextActionCardSpeaksPlainEnglish; the plain-English check was scoped to the new next-action card because rewording the ten legacy cards is plans 197-04 and 197-06's declared scope.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-28T19:48:11.715Z",
    "resolved_at": "2026-08-29T00:00:00.000Z"
  },
  {
    "id": 4,
    "kind": "deviation",
    "phase": "197",
    "file": "cmd/next_action.go",
    "line": null,
    "description": "197-02 edited cmd/next_action.go, which is outside its declared files_modified, to change one word in the failed-phase recommendation ('Running it again' -> 'The next step is to retry it') so the pre-existing TestWorkflowSuggestionsFailedPhase assertion on the word 'retry' kept passing without weakening it.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-28T19:48:15.796Z",
    "resolved_at": null
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "renderPlanVisual's confidence branch only type-asserts to map[string]interface{}; both runCodexPlanWithOptions and runCodexPlanFinalize always store confidence as a codexPlanConfidence struct, so the Confidence line never renders on the direct or chat path. Out of scope for 198-04 (prohibited from editing that file, owned by a same-wave plan); needs a dual-type fix mirroring planning_loop's existing struct/map switch.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T17:49:53.592Z",
    "resolved_at": "2026-08-29T18:28:52.436Z"
  },
  {
    "id": 6,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_visuals.go",
    "line": null,
    "description": "renderPlanVisual never reads result[\"research_warning\"] (which renderResearchFailedWarning's own doc comment says exists \"so the omission is durable and visible\" when a phase was planned without its research), result[\"research_failed_phases\"] (the phase IDs that fed that warning), or result[\"gaps\"] (a completed plan's own unresolved gaps). Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json) because cmd/codex_visuals.go was owned this wave by sibling plan 198-08.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T19:15:56.000Z",
    "resolved_at": "2026-08-29T20:05:27.000Z"
  },
  {
    "id": 7,
    "kind": "deviation",
    "phase": "198",
    "file": "cmd/codex_plan_finalize.go",
    "line": null,
    "description": "runCodexPlanFinalize never calls closeLifecycleRun (unlike continue-finalize/completeSealRuntime), so the plan finalizer's own suggested result[\"next\"] command never folds into the unified next-action envelope renderLifecycleClosing reads back -- the closing card instead independently resolves a next step from live colony state, which usually matches but is not guaranteed to. Found and allow-listed, not fixed, in 198-09-PLAN.md Task 1 (cmd/testdata/rendered_field_allowlist.json, \"plan completed\" and \"plan mid-loop\") because cmd/codex_plan_finalize.go was outside that plan's declared files.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-29T19:15:56.000Z",
    "resolved_at": "2026-08-29T20:05:27.000Z"
  },
  {
    "id": 8,
    "kind": "deviation",
    "phase": "198.2",
    "file": "cmd/codex_continue_finalize.go",
    "line": null,
    "description": "externalContinueReviewReport does not append a 'review wave skipped' narration step when no reviewers were dispatched, unlike the direct lane's runCodexContinueReview -- discovered by 198.2-05's dual-lane outcome-text parity test, out of that plan's scope to fix",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-30T14:49:33.437Z",
    "resolved_at": null
  }
]
````
