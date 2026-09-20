# Honest No-Change Results Implementation Plan (Stage 1, Work Package 1)

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:executing-plans. Steps use checkbox syntax.

**Goal:** An honest "nothing needed changing" worker result is accepted end-to-end (with mandatory evidence), and a rate-limit/quota interruption is a recognized resumable terminal outcome — closing the spec §2.4 failure ("an honest `no changes required` result was rejected") per owner rulings D6 and D7.

**Architecture:** Extend the existing external-result vocabulary in `cmd/codex_build_finalize.go` (normalize → terminal-set → merge validations → provenance) plus the worker-facing contract in `pkg/codex/worker.go`. No new stores; schema regenerated in lockstep (191.1 byte-sync test).

**Spec:** `.planning/research/priority-spec-v3-backlog.md` Stage 1; semantics fixed by `.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md` D6 (status `completed_no_change`, disposition `verified_existing`) and D7 (worker result `interrupted` with reason).

## Global constraints
- Branch `oracle-reinstate` (current). Zero new test failures vs green baseline. Never push.
- D6: ONE success status — `completed_no_change`; `verified_existing` is a disposition on it, never a second status.
- Evidence rule: a no-change completion must carry `summary` + handoff `verification_status: pass` + non-empty `commands_run` — otherwise a named contract violation (`no_change_evidence`). No free-pass loophole.
- `interrupted` is terminal but NOT success: never completes a task, never grants covered credit, maps to handoff verification `not_run`.

### Task 1: Vocabulary + terminal set
Modify `normalizeExternalBuildStatus` (finalize:1635): aliases `no_change|no-change|nochange|unchanged|completed_no_change` → `completed_no_change`; `verified_existing|already_complete|already_correct` → `completed_no_change`; `suspended_quota|rate_limit|rate_limited` → `interrupted`. `isTerminalExternalBuildStatus` (:1651): add `completed_no_change`, `interrupted`. Add helpers `isSuccessfulExternalBuildStatus` (completed | completed_no_change | manually-reconciled) and `isNoChangeExternalBuildStatus`. Table-test first (`cmd/no_change_results_test.go`).

### Task 2: `Disposition` field + schema
Add `Disposition string json:"disposition,omitempty"` to `codexExternalBuildWorkerResult` (values validated: empty or `verified_existing`; violation otherwise). Raw status `verified_existing`/`already_*` sets Disposition when empty. Carry onto `codexBuildDispatch` (add field if absent) and into the dispatch summary suffix ` (verified existing)`. Regenerate `.aether/schemas/completion-packet.schema.json` via its existing sync test/generator (find the 191.1 mechanism; run with its update flag).

### Task 3: Merge-path evidence gate
In `mergeExternalBuildResults` after handoff validation (~:1433): no-change results missing evidence → violation rule `violationRuleNoChangeEvidence` with message naming the three required pieces. Tests: `TestNoChangeResultWithEvidenceIsAccepted` (raw status `verified_existing`, summary, handoff pass + commands_run → zero violations, dispatch status `completed_no_change`, disposition set) and `TestNoChangeResultWithoutEvidenceIsRejected`.

### Task 4: Claimant + downstream success checks
- `genuineSuccess` (:1312) → `isSuccessfulExternalBuildStatus`; `hasEvidence` accepts a no-change claimant's `commands_run` as evidence (comment why).
- Audit every `== "completed"` in finalize + attempt/task-completion marking (grep; includes :1470, :1958, build_attempt task status writes): sites meaning "succeeded" use the success helper; sites meaning "produced files" stay.
- `verificationStatusForWorkerStatus` (dispatch_contract): `completed_no_change`→`pass`, `interrupted`→`not_run`.
- Test: `TestInterruptedIsTerminalButNotSuccess` (no status violation; task not completed; no covered-credit granting).

### Task 5: Provenance both directions
`validateBuildProvenance`: evidenced no-change implementation result satisfies provenance (zero files); evidence-free no-change still rejected (SAFE-02 spirit). `traceContinueProvenance`: `completed_no_change` counts as completed and is exempt from the Outputs requirement. Tests: `TestProvenanceAcceptsEvidencedNoChangeBuild`, `TestContinueProvenanceAcceptsNoChangeDispatch`.

### Task 6: Worker-facing contract
`pkg/codex/worker.go`: statusLine gains `completed_no_change` (both variants); add contract bullet — when the work already satisfies acceptance, report `completed_no_change` with `disposition: verified_existing`, put the verification commands in handoff `commands_run`, and NEVER fabricate an edit to satisfy accounting; `interrupted` bullet for quota/rate-limit stops (preserve state, it will be resumed). Update the `Status` doc comment (:72). Check `pkg/codex` result validation for status allowlists and extend. Test: `TestResponseContractOffersNoChangeOutcome`.

### Task 7: Full verification + docs
`go build ./cmd/aether && go vet ./...`; full `go test ./cmd/... ./pkg/... -count=1` (zero new failures); race pass on new tests; refresh goldens only via their sanctioned update flags if catalogs shifted; CHANGELOG Unreleased entry; CLAUDE.md one-line note if any documented behavior changed; commit per task, imperative <72 chars.
