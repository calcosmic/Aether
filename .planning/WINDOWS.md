---
schema_version: 1
open_count: 2
waived_count: 0
fixed_count: 2
total_count: 4
last_updated: 2026-08-29T00:00:00.000Z
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
  }
]
````
