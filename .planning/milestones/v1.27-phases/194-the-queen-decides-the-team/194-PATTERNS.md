# Phase 194: The Queen Decides the Team - Pattern Map

**Mapped:** 2026-08-23
**Files analyzed:** 13 modified (no new files) + 6 wrapper/doc files
**Analogs found:** 13 / 13 — self-referential, same as Phase 193. This phase edits existing decision functions in place; each file is its own best analog. Line numbers below were verified against current source and mostly match CONTEXT.md's hints (noted where they drift).

## File Classification

| Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/queen_spawn_budget.go` | policy / floor calculator | request-response, deterministic decision | itself — `queenBuildSafetyRequiredCastes` (139), `queenPhaseHasSecuritySignal` (187), `queenBuildSafetyReviewRequired` (211) | exact (edit in place) |
| `cmd/queen_judgement.go` | reconciler (proposal vs. floor vs. budget) | request-response, deterministic decision | itself — `queenApplyJudgement` (91), `Summary()` (54), `queenCasteDecisionSummary` (365) | exact (edit in place) |
| `cmd/caste_relevance.go` | selector / scorer | request-response, deterministic decision | itself — `isAlwaysRequired` (316), `queenCandidateDispatches` (158), `spawnThreshold` (271) | exact (edit in place) |
| `cmd/review_depth.go` | risk/depth classifier | request-response, deterministic decision | itself — `phaseRiskLevel` (288), `resolveSmartVerificationDepth` (321), `loadReviewDepthPolicy` (41) | exact (edit in place) |
| `cmd/codex_continue.go` | orchestrator (in-process continue lane) | request-response, deterministic pipeline | itself — `queenContinueReviewSpecsWithJudgement`, `runCodexContinueReview`, `plannedContinueReviewDispatches` (grep below for current lines) | exact (edit in place) |
| `cmd/codex_continue_plan.go` | orchestrator (wrapper/external continue lane) | request-response, deterministic pipeline | `plannedExternalContinueDispatches`; mirrors `codex_continue.go`'s review-spec logic | role-match, cross-file mirror (same two-lane discipline as Phase 193) |
| `cmd/codex_dispatch_contract.go` | model / contract struct | CRUD-like (build/serialize a manifest struct) | itself — `SelectedReasons`/`PrunedReasons` fields (461-462), composed at ~570-585 | exact (edit in place, reuse the existing slot) |
| `cmd/ceremony_team_checkin.go` | presentation (renders card from manifest) | request-response, read-only render | itself — `renderCeremonyTeamCheckin` (30), `casteRosterProduces` (118) | exact (edit in place) |
| `cmd/codex_workflow_cmds.go` | CLI flag wiring | request-response, flag parsing | itself — `--castes`/`--caste-reason`/`--heavy`/`--light`/`--no-checkin` flag definitions | exact (edit in place) |
| `cmd/codex_build.go` | orchestrator (build-side dispatch planner) | event-driven (dispatch list construction) | itself — `plannedBuildDispatchesWithJudgement`, `manifest.CasteDecision` | exact (edit in place) |
| `cmd/codex_build_finalize.go` | orchestrator (persists forced-reviewer record) | request-response, report recording | itself, plus `cmd/build_attempt.go`'s terminal-record pattern (Phase 193 precedent) | exact (edit in place) |
| `cmd/codex_continue_finalize.go` | orchestrator (wrapper lane, subset check) | request-response, deterministic pipeline | itself — `continueReviewCastesArePlannedSubset` | exact (edit in place) |
| `cmd/autopilot.go` | caller (no-proposal path) | request-response | itself — the call site that passes no `--castes`, exercising the D-11 fallback | exact (edit in place) |

All target files already exist and already implement the exact decision points this phase changes — same situation as Phase 193. There is no "new file" case.

## Pattern Assignments

### `cmd/queen_spawn_budget.go` — the floor (TEAM-01, TEAM-02, D-06, D-07)

**Analog:** itself, `queenBuildSafetyRequiredCastes` (lines 139-174)

**Current shape (verified, matches CONTEXT.md's ~139 hint):**
```go
func queenBuildSafetyRequiredCastes(phase colony.Phase) []string {
	// Watcher is the only unconditional member: it is the check, and a build
	// with nothing verifying it is a build that reports success by assertion.
	required := []string{"watcher"}
	if effectiveQueenPhaseMode(phase) != colony.PhaseModeDiscovery {
		required = append(required, "builder")
	}
	if queenPhaseProducesTestableCode(phase) {
		required = append(required, "probe")
	}
	riskLevel := phaseRiskLevel(phase)
	if riskLevel == "high" || queenBuildSafetyReviewRequired(phase) {
		required = append(required, "auditor")
	}
	if riskLevel == "high" || queenPhaseHasSecuritySignal(phase) {
		required = append(required, "gatekeeper")
	}
	sort.Strings(required)
	return required
}
```
This is the direct target of **D-06** ("watcher always" and the production-mode `auditor` branch go), **D-07** (build-side required castes shrink to `builder` on non-discovery phases; `probe` keeps only its negative gate), and **D-04** (one reviewer per signal, not `riskLevel == "high"` triggering both auditor and gatekeeper together). The comment block above `required := []string{"watcher"}` (lines 140-141) is itself a claim this phase falsifies — CONTEXT.md's D-06 directly contradicts "Watcher is the only unconditional member." Follow the existing style: keep the doc comments, rewrite them to state the new rule and why (this file's existing comments — see `queenPhaseHasSecuritySignal`'s comment at 176-186 and `queenBuildSafetyReviewRequired`'s at nothing (bare) — are the house style: explain the failure mode a naive version had, cite the specific phase example that broke it).

**`queenPhaseHasSecuritySignal`** (lines 187-209) is the closest existing analog for **D-01**'s five-signal table — it already does word-boundary-ish keyword matching via `matchesAnyKeyword` against `collectPhaseText(phase)` (name + description + tasks, matching D-02's `collectPhaseText` reference). D-01 needs this narrowed to the five named signals (credentials/auth, payments, data deletion, migrations, release sign-off) and split into two reviewer targets per D-04 — do not keep one flat list feeding both `auditor` and `gatekeeper` off the same `riskLevel == "high"` check. `queenBuildSafetyReviewRequired` (211-226), which currently treats `PhaseModeProduction` alone as a signal, is the literal function D-06 deletes (its mode branch) — but its keyword list already includes "release"/"signoff" terms useful for D-01's release-sign-off signal, so lift the wording rather than starting fresh.

**`queenPhaseProducesTestableCode`** (345-364) already implements exactly the D-07 negative gate CONTEXT.md describes ("probe is never required; it keeps only its negative rule") — untouched by this phase except that its unconditional "required" partner shrinks.

**`queenRequiredCastesForBudget`** (121-130) and `queenBuildSafetyRequiredCaste` (132-137) are the call chain feeding the floor into `queenSpawnBudgetForPhase` (27) — the struct shape (`RequiredCastes []string`, `Reason string`) at lines 11-17 is unchanged; this phase changes what populates the slice, not the struct.

---

### `cmd/queen_judgement.go` — reconciling proposal, floor, and reasons (TEAM-03, D-08, D-09)

**Analog:** itself, `queenApplyJudgement` (91-185) and `Summary()` (54-83)

**Current shape:** `queenApplyJudgement` takes `proposed []string, rationale string` — **one rationale string for the whole team**, exactly what D-08 says "no longer satisfies TEAM-03 for any worker." The struct `queenCasteJudgement` (31-51) already has `Rationale string` as a single field (46-47) — D-08's per-worker reason needs a new field shape here, most naturally a `map[string]string` (caste -> reason) sitting alongside `Added`/`Dropped`/`Refused`/`Unknown` (36-45), following the exact style of those four slices: each is "list of castes with one classification," a reasons map is "map of caste to one sentence," same granularity switch Phase 193's `SelectedReasons`/`PrunedReasons` map already uses in `codex_dispatch_contract.go` (461-462) — **reuse that map shape, don't invent a new one.**

`Summary()` (54-83) is the render pattern to extend for D-08's "refused for that worker only, by name" requirement — it already has this exact shape for `Refused`:
```go
if len(j.Refused) > 0 {
    b.WriteString(fmt.Sprintf(" Refused %s — nothing in this phase for it to do.",
        strings.Join(j.Refused, ", ")))
}
```
A new `refused_no_reason: [probe]`-style branch (D-08's literal example) should follow this exact `if len(j.X) > 0 { fmt.Sprintf(...) }` block pattern — one more case in the same builder.

**The refusal loop at lines 118-129** (checking `casteRelevanceScore(phase, caste) == 0` to refuse a proposed caste with nothing to do) is the direct precedent for D-08's "no reason" refusal: same shape — loop over `normalized`, build a `refused` list, `continue` past the caste rather than including it. A `caste-reason` map arriving with a proposal should merge into this same loop: caste refused if (a) score is 0 (existing) **or** (b) no reason was supplied for it and it's not in `required` (new, D-08).

**`queenCasteDecisionSummary`** (365-, `map[string]interface{}` builder including `summary["rationale"]` at 387-388) is the JSON-manifest surface this phase must also update to carry the per-caste reason map into `caste_decision` for the manifest (D-10 says this travels to check-in card + manifest + dispatch record).

---

### `cmd/caste_relevance.go` — selector, fallback engine (TEAM-04, D-09, D-11, D-12)

**Analog:** itself, `queenCandidateDispatches` (158-195), `isAlwaysRequired` (316-)

**The rationale string D-09 forbids as a reason** (line 176):
```go
rationale := fmt.Sprintf("Score %d >= threshold %d for %s flow", score, threshold, flowType)
if always {
    score = 100
    rationale = fmt.Sprintf("%s is always required for %s flow", profile.Caste, flowType)
}
```
This is the literal `queenCandidateDispatches` line CONTEXT.md cites at "~176" (confirmed) as the thing D-09 forbids appearing in a reason slot ("never 'Score 20 >= threshold 15'"). D-09 requires this replaced with — or accompanied by — a plain-English sentence per caste ("Tracker — the phase describes a bug to investigate"). Since D-11 also removes this function's role in *selecting* optional specialists (it becomes refusal-only / candidate-listing), the rewrite here is really: keep `casteRelevanceScore`/`spawnThreshold` for the *refusal* check (score == 0 refuses a proposal), but stop `queenCandidateDispatches` from being the thing `queenOrchestrate` (154-156) calls to build the no-proposal team — that becomes D-11's "builder plus any forced reviewer" fallback in `queen_spawn_budget.go`'s required-castes list plus (D-12) one `scout` on discovery.

**`isAlwaysRequired`** (316-364, continue-flow switch already read above) is the direct edit target for D-13's "continue's required set by depth: light/standard → none; heavy → the panel" — currently light returns `watcher` and standard returns `watcher || probe-if-testable`; both need their `watcher` return dropped to match D-06 (watcher no longer unconditional) while heavy's four-caste panel stays as the explicit `--heavy` ceiling.

**`applySpecialRules`** (223-268) is the keyword-scoring switch (`case "gatekeeper": ... phaseRiskLevel(phase) == "high"`, `case "auditor": ... phase.Mode == colony.PhaseModeProduction`) — these per-caste 100-score short-circuits duplicate the floor logic in `queen_spawn_budget.go` and are exactly the kind of "production mode ⇒ auditor" / "high risk ⇒ gatekeeper" implicit floor D-06 says goes; if this switch still auto-scores auditor/gatekeeper to 100 on production/high-risk after the budget floor is rewritten, the two are out of sync — D-05's "one derivation, one boundary" applies here too even though it's phrased about build/continue.

---

### `cmd/review_depth.go` — risk classification and the position-based heavy escalation (D-06)

**Analog:** itself, `resolveSmartVerificationDepth` (321-350), `phaseRiskLevel` (288-297)

**The position-final rule D-06 removes** (lines 330 and 340-341, both branches):
```go
if risk == "high" || position == "final" {
    return colony.VerificationDepthHeavy
}
```
This exact `position == "final"` clause appears twice (production branch at 330, prototype/maintenance branch at 340) — D-06 says "position no longer raises verification depth; an explicit `--heavy` still does," so both occurrences of `|| position == "final"` are deleted, leaving `risk == "high"` alone as the heavy trigger. `phasePositionLevel` itself may become dead code here if nothing else calls it — check before deleting the helper.

`loadReviewDepthPolicy` (41-) and `securityRiskKeywordsFallback` (62-88) are the existing extension point CONTEXT.md flags as "Claude's discretion — whether it remains the extension point" for D-01's five-signal vocabulary; `phaseRiskLevel` currently matches on **phase name only** (288, deliberate per its own comment to avoid "session"/"token"/"password" false positives from description text) — D-01/D-02's signal table is a **different, more specific** vocabulary than this file's blast-radius/security-risk keyword lists, so it likely needs its own function rather than reusing `phaseRiskLevel`, but should follow the same "name only, not full text" discipline this file already learned the hard way (comment at 286-287 names the exact false-positive words D-02 also calls out: "token" alone is a known false alarm).

---

### `cmd/codex_dispatch_contract.go` — the reason-carrying struct (D-10)

**Analog:** itself, lines 461-462 (fields) and ~570-585 (composition)

```go
SelectedReasons         map[string]string `json:"selected_reasons,omitempty"`
PrunedReasons           map[string]string `json:"pruned_reasons,omitempty"`
...
if contract.SelectedReasons == nil {
    contract.SelectedReasons = make(map[string]string)
}
contract.SelectedReasons[decision.Caste] = rationale
...
if contract.PrunedReasons == nil {
    contract.PrunedReasons = make(map[string]string)
}
contract.PrunedReasons[decision.Caste] = rationale
```
**This is the exact slot D-10 says "the manifest (`selected_reasons` is the existing per-caste slot)" reuses.** No new manifest field is needed for the reason map itself — the work is making `rationale` here a plain-English per-worker sentence (fed from `queenApplyJudgement`'s new reason map) instead of the `queenCandidateDispatches` score-arithmetic string it currently receives.

---

### `cmd/ceremony_team_checkin.go` — the check-in card (D-03, D-09, D-14)

**Analog:** itself, `renderCeremonyTeamCheckin` (30-114), `casteRosterProduces` (118-130)

**The fallback D-09 forbids** (lines 55-58):
```go
reason := strings.TrimSpace(selectedReasons[caste])
if reason == "" {
    reason = casteRosterProduces(manifest, caste)
}
```
This is the literal fallback-to-generic-blurb D-09 names ("the card may show it as 'what it does', never in the reason slot"). Since D-08 says a worker with no reason is refused rather than sent with a placeholder, this fallback should mostly stop firing once the upstream reason map is always populated — but if it's kept as a last-resort display (never as the thing that satisfies TEAM-03), it needs a visible label distinguishing "what it does" from "why it's here today," per D-09's instruction that the card may show it as "what it does" but never present it as the reason.

**Card render loop** (67-104) — the `REQUIRED`/`OPTIONAL` marking (73-77) and the `reason := reasons[caste]` append (81-84) is the exact place D-03's waive choice and D-10's "REQUIRED now means exactly 'builder, or forced by a named signal (signal shown)'" attach: extend the per-caste line to show the signal name when `requiredSet[caste]` is true because of a forced reviewer (D-05: "a security reviewer will check this — this touches logins"), and add the waive affordance/record here (D-03/D-14) following the same `map[string]interface{}` result-building convention at 106-113.

---

### `cmd/codex_continue.go` / `cmd/codex_continue_plan.go` — two-lane mirroring (D-05)

**Analog:** the Phase 193 precedent — same file pair, same discipline

Reuse Phase 193's documented shared pattern verbatim: **"every rule this phase adds needs its counterpart checked/updated in both `codex_continue.go` (in-process) and `codex_continue_plan.go`/`codex_continue_finalize.go` (wrapper/external lane)."** D-05 explicitly restates this as the WINDOWS #1 closure: "the build manifest and the check-in card announce it... The build records the forced set... `continue` reads that record, adds any D-02 file-detected signal, and dispatches. One derivation, one boundary." Grep for `queenContinueReviewSpecsWithJudgement`, `runCodexContinueReview`, `plannedContinueReviewDispatches` (in-process) and `plannedExternalContinueDispatches` (external) at build time to get current line numbers — CONTEXT.md's ~1269/~1331/~1431/~320 hints should be re-verified before editing since this file is large (4248 lines) and may have shifted since 193 landed.

---

### `cmd/codex_build.go` / `cmd/codex_build_finalize.go` — recording the forced-reviewer signal (D-05)

**Analog:** Phase 193's precedent again — `codex_build_finalize.go`'s report-not-gate pattern via `build_attempt.go`'s terminal record (see 193-PATTERNS.md's `recordBuildAttemptTerminal` section)

D-05 requires the build to **record** the forced set (caste, signal, reason, source=plan wording) as a decision, not just dispatch it — follow the same "extend the terminal-record shape, don't invent a parallel report file" instruction Phase 193 already established for this exact file pair.

---

## Shared Patterns

### Doc-comment discipline: name the failure mode, cite the phase example
**Source:** `cmd/queen_spawn_budget.go` — nearly every function in this file (139-141, 161-167, 176-186, 228-229, 341-343, 350-359)
**Apply to:** every rewritten function in this phase. House style is: state what the naive version does wrong, name the concrete phase title that broke it, then state the current rule. This phase's rewrites of `queenBuildSafetyRequiredCastes`, `queenPhaseHasSecuritySignal`, `resolveSmartVerificationDepth` should keep this discipline rather than becoming terse.

### Per-caste map (`map[string]string`), never a single string, for anything "one per worker"
**Source:** `cmd/codex_dispatch_contract.go` `SelectedReasons`/`PrunedReasons` (461-462); `cmd/ceremony_team_checkin.go`'s `reasons map[string]string` (53)
**Apply to:** D-08/D-09/D-10's per-worker reason — this shape already exists in two of the three places it needs to reach; the third (`queenCasteJudgement.Rationale string` in `queen_judgement.go`) is the one place still holding the old single-string shape and is this phase's actual gap.

### "Assert the spawn list, not the decision record"
**Source:** Phase 193's `TestQueenChoiceReachesTheDispatchList` precedent, restated in this phase's own CONTEXT.md (Established Patterns)
**Apply to:** every new test in this phase (`TestReviewerForcedOnlyByNamedRisk`, `TestNoWorkerWithoutStatedReason`, `TestOneTaskBugFixIsOneWorkerPlusChecks`) — assert on the actual dispatch list / manifest that reaches the wrapper, not on an intermediate struct that a later function could silently override (the exact bug `TestQueenChoiceReachesTheDispatchList` caught: a recorded choice deleted one line later by `applyBuildDispatchPolicyCastes`).

### Two-lane parity (in-process vs. wrapper/external continue)
**Source:** Phase 193 `cmd/codex_continue.go` vs `cmd/codex_continue_plan.go`/`codex_continue_finalize.go`
**Apply to:** D-05 specifically calls this out again ("the 2026-08-21 review gate found the in-process lane half-wired") — no shared helper enforces this; it remains a manual-parity discipline.

### Retired-test ledger entries (RETIRE-04 format)
**Source:** `.aether/docs/retired-tests-ledger.md` (existing entries from Phase 193)
**Apply to:** D-15's five retirements (`TestWatcherIsAlwaysRequiredOnBuild`, `TestQueenCannotDropTheWatcher`, `TestSafetyCastesSurviveProbeGating`, `TestHighRiskPhaseKeepsBothReviewers` or rewrite, positive half of `TestProbeIsRequiredOnlyWhereItCanFindSomething`) — cite `.planning/decisions/2026-08-22-queen-decides-program-checks.md` (D11) as the disposition source, following whatever field structure the ledger's existing entries use (read the ledger file directly before writing new entries — not read in this pass).

## No Analog Found

None — every target file already contains the exact decision function this phase rewrites, same as Phase 193. The wrapper markdown triplets (`.claude/commands/ant/build.md`, `continue.md` + `.opencode` + flat-mirror copies) and `CLAUDE.md` are prose edits, not code with a "pattern" — treat them as find-the-stale-claim-and-correct-it edits; CONTEXT.md already quotes the exact stale sentence to remove ("A build always gets a Watcher; credential, auth, and release-gate work always gets a security review").

## Metadata

**Analog search scope:** `cmd/` — the thirteen files named in CONTEXT.md's "Code this phase changes," cross-referenced against `cmd/queen_judgement_test.go`, `cmd/queen_probe_gating_test.go`, `cmd/caste_relevance.go` for current line numbers.
**Files scanned:** `cmd/queen_spawn_budget.go` (full), `cmd/queen_judgement.go` (lines 1-200), `cmd/caste_relevance.go` (lines 140-340), `cmd/review_depth.go` (lines 280-360), `cmd/ceremony_team_checkin.go` (full), `cmd/codex_dispatch_contract.go` (grep only, lines 461-462/570-585), plus function-signature greps across `codex_continue.go`, `codex_continue_plan.go`, `codex_build.go`, `codex_build_finalize.go`, `codex_continue_finalize.go`, `autopilot.go`, `codex_workflow_cmds.go`.
**Line-number drift from CONTEXT.md:** all `cmd/queen_spawn_budget.go`, `cmd/queen_judgement.go`, `cmd/caste_relevance.go`, `cmd/review_depth.go` hints verified exact. `cmd/codex_continue.go`/`codex_continue_plan.go` line hints (~1269/~1331/~1431/~320) were not re-verified by direct read in this pass (file is 4248 lines) — planner should grep for the named functions before citing exact line numbers in a plan.
**Pattern extraction date:** 2026-08-23
