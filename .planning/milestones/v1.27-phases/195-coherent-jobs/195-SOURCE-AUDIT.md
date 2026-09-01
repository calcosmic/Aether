# Phase 195 Multi-Source Coverage Audit

Every in-scope item from the phase goal, requirements, research, and locked
context decisions maps to at least one executable plan. Deferred ideas and work
explicitly assigned to later phases are recorded as exclusions, not gaps.

| SOURCE | ID | Feature/Requirement | Plan | Status | Notes |
|--------|----|---------------------|------|--------|-------|
| GOAL | — | Queen groups related tasks into one runtime-validated job; defaults group shared files/dependencies; six CalVault batches become one worker | 01, 03, 08 | COVERED | Pure planner, in-repo integration, and worktree proof |
| REQ | JOBS-01 | Queen job proposal with reason; dependency-order violation refused by name | 01, 03 | COVERED | Structured relationship/benefit and local repair |
| REQ | JOBS-02 | Brief carries every task contract; full/partial completion credits exact covered tasks end to end | 03, 04, 06, 07, 08 | COVERED | Two-stage native/external/worktree receipts, claims, credit, retry |
| REQ | JOBS-03 | No-proposal shared-file/dependency clustering beyond adjacent same-caste steps; CalVault one worker | 01, 03 | COVERED | Component planning occurs before waves |
| REQ | JOBS-04 | Grouped jobs work in worktree mode | 03, 08 | COVERED | Same plan before ownership; one worktree and root-backed partial sync |
| RESEARCH | — | Whole-plan DetectCycles preflight with named missing/cycle repair and zero side effects | 01, 03 | COVERED | Global hard refusal before manifest/attempt/worktree |
| RESEARCH | — | One canonical `planCoherentJobs` layer before task waves/worktree ownership | 01, 03 | COVERED | Post-wave coalescer loses authority |
| RESEARCH | — | Structured proposal fields; local refusal preserves safe proposals | 01, 03 | COVERED | Queen proposes, Go validates/persists |
| RESEARCH | — | Meaningful exact paths join; housekeeping/dependency metadata paths do not | 01 | COVERED | Named pure policy and table |
| RESEARCH | — | Automatic grouping preserves castes; explicit cross-caste group requires suitable owner reason | 01 | COVERED | Relevance/owner validation retained |
| RESEARCH | — | Soft limit uses projected brief content and existing 6,000-character allowance; no task-count cap | 01 | COVERED | Dependency-safe split only |
| RESEARCH | — | Selected-task proposals cannot pull completed/unselected tasks back into execution | 01, 03 | COVERED | Contract + command side-effect tests |
| RESEARCH | — | Job manifest metadata, every-task brief, stable grouped name, single-task compatibility, truthful `covered tasks` wording | 03 | COVERED | Reachability and compatibility integration |
| RESEARCH | — | Additive task_receipts contract on native/external worker results and generated schema | 02 | COVERED | One shared Go type; schema generated, not hand-edited |
| RESEARCH | — | Shared two-stage receipt contract: structural admission yields candidate claims/sync paths without credit; post-sync root finalization alone yields task claims/CompletedTaskIDs | 04, 06, 08 | COVERED | Same stages in native, external, and worktree lanes |
| RESEARCH | — | Whole-success without receipts remains compatible; failed group credits only explicit receipts | 04, 06 | COVERED | Full and four-of-six fixtures |
| RESEARCH | — | Append-only unfinished-only retry linked to original attempt/job | 07 | COVERED | Credited dependencies treated satisfied; parent immutable |
| RESEARCH | — | Worktree full success uses one worktree/unioned paths; unrelated overlap guard stays | 08 | COVERED | Real CalVault worktree fixture |
| RESEARCH | — | Partial worktree admission produces sync paths without credit; sync precedes shared root finalization; other edits remain recoverable | 08 | COVERED | Open Question 1 explicitly RESOLVED and mapped to 04/06/08 |
| RESEARCH | — | Runtime-owned check-in policy, compact summary, pending-decision predicate, early flag conflict | 05 | COVERED | Open Question 2 explicitly RESOLVED and mapped to 05 |
| RESEARCH | — | Canonical YAML, three wrappers, Codex skill, command guide, CLAUDE.md, CLI/schema parity | 02, 09, 10 | COVERED | Generated schema in 02; parity-critical surfaces atomically synchronized in 09; owner docs/gates in 10 |
| RESEARCH | — | Fail-then-pass CalVault and adversarial tests; full/race/vet/build/schema gates | 01, 03, 08, 10 | COVERED | Both execution modes prove one worker; final phase gates isolated in 10 |
| CONTEXT | D-01 | Dependency chains or meaningful shared implementation files group; incidental bookkeeping overlap does not | 01 | COVERED | Exact path evidence and exclusion table |
| CONTEXT | D-02 | Automatic grouping preserves caste boundaries; explicit cross-caste ownership requires reason and validation | 01 | COVERED | Proposal owner contract |
| CONTEXT | D-03 | Soft size limit; unusually large cluster splits unless Queen explains end-to-end ownership | 01 | COVERED | Brief allowance, not task count |
| CONTEXT | D-04 | Every job reason names relationship and benefit | 01, 03, 05 | COVERED | Structured fields; manifest and compact summary render them |
| CONTEXT | D-05 | Unsafe order refused by name with offending task/dependency; never runs | 01, 03 | COVERED | Pure decision and zero-side-effect integration |
| CONTEXT | D-06 | Keep safe jobs and visibly repair only affected group; no silent reorder/global explosion | 01, 03 | COVERED | Safe-proposal preservation regression |
| CONTEXT | D-07 | Real cycle blocks phase with named cycle and plan-repair action | 01, 03 | COVERED | DetectCycles preflight before side effects |
| CONTEXT | D-08 | Honest partial proof credits only explicit task receipts | 04, 06, 08 | COVERED | Admission grants no credit; root finalization drives exact native, external, and worktree credit |
| CONTEXT | D-09 | Receipt binds task requirements, claimed files, verification evidence through manifest validation | 02, 04, 06, 08 | COVERED | Wire type plus shared two-stage admission/root-evidence boundary |
| CONTEXT | D-10 | Recovery is unfinished-only, dependency-safe, append-only, parent-linked, and never redoes proof | 07, 08 | COVERED | Both completion lanes plus worktree recovery |
| CONTEXT | D-11 | Exactly one decision-free implementation worker automatically skips blocking check-in | 05 | COVERED | Group size in tasks does not affect decision |
| CONTEXT | D-12 | Compact non-blocking summary names worker/tasks/reason/no-approval cause | 05, 09, 10 | COVERED | Runtime renderer, atomic wrapper consumption, and owner guidance |
| CONTEXT | D-13 | Pending owner decision, especially named-risk waiver, keeps blocking check-in | 05 | COVERED | Forced waiver + generic boundary question tests |
| CONTEXT | D-14 | `--checkin` forces pause; conflict with `--no-checkin` is named and mutation-free | 05, 09 | COVERED | Runtime flag/policy plus surfaced contract |
| CONTEXT | Folded todo | One-worker build goes straight through without becoming invisible | 05, 10 | COVERED | Behavior lands in 05; preserved todo resolved/moved in 10 |
| CONTEXT | Deferred | User-facing spec-builder command | — | EXCLUDED | Explicitly v1.28+ |
| CONTEXT | Deferred | Grouped-job cost/token rendering | — | EXCLUDED | Phase 196 |
| CONTEXT | Deferred | Universal closing card and richer live display | — | EXCLUDED | Phases 197–198 |

## Audit Result

No `MISSING` rows. No phase split is required: the ten plans keep each
implementation context below the planner's quality threshold while delivering
the full locked scope.
