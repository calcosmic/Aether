---
phase: 164
slug: research-feeds-planning
status: secured
threats_open: 0
asvs_level: default
created: 2026-08-02
---

# Security Audit — Phase 164: Research Feeds Planning

**Audit type:** State B (created from artifacts — no prior SECURITY.md existed)
**Plans audited:** 164-01 through 164-11 (11 plans, including gap-closure plans 10 and 11)
**Threats in register:** 34 (register authored at plan time, `register_authored_at_plan_time: true`)
**ASVS level:** default
**block_on:** default

Namespacing note: plans 01–09 use IDs `T-164-01`..`T-164-27`. Plans 10 and 11
restart numbering (`T-01`..`T-04` and `T-05`..`T-07` respectively); these are
namespaced below as `164-10/T-01..04` and `164-11/T-05..07` to avoid collision
with the Plan 01–09 series, per the launching instructions.

Verification method: every `mitigate` threat was checked by reading the cited
implementation file(s) and confirming the described code exists and does what
the mitigation plan claims — not by trusting PLAN/SUMMARY prose. Where
feasible, the actual test suite was executed (not just re-read) to confirm
claimed tests pass. All `accept` threats were checked for continued factual
accuracy of their rationale against current code, and are logged below as the
project's Accepted Risks Log.

---

## Threat Verification (mitigate disposition)

| Threat ID | Category | Component | Evidence | Status |
|---|---|---|---|---|
| T-164-01 | Tampering | `renderPhaseResearchBrief` survey section | `cmd/phase_research.go:158-197` (`renderPhaseResearchSurveySection`) interpolates only `codexSurveyContext` fields (`SurveyDocs`, `Languages`, `Frameworks`, `Dependencies`) derived from `loadCodexSurveyContext`, never raw phase description text, into the survey section | CLOSED |
| T-164-03 | DoS | replan re-dispatch of every phase | `cmd/phase_research.go:87` gates the staleness skip on `!reresearch`; `cmd/codex_plan.go` computes `reresearch` as `opts.Refresh && iteration == 1` (bounded to first iteration of a refresh run) | CLOSED |
| T-164-04 | Tampering | `computePhaseResearchProposal` reason text | `cmd/phase_research_decision.go:28-35` (`phaseResearchReasons` fixed map) and lines 94-113 show only the matched signal token (e.g. `hint.DomainGaps[0]`) is `fmt.Sprintf`'d into the template — the raw phase description is never interpolated | CLOSED |
| T-164-06 | Repudiation | user override of Queen's recommendation | `cmd/phase_research_decision.go:225-236` (`phaseResearchDecisionResolution`) records resulting direction + phase in `PendingDecision.Resolution`; verified via `TestResearchDecisionRecordsUseExistingStore` (passing) | CLOSED |
| T-164-07 | Tampering | worker inflating its own confidence | `.aether/ts-host/src/research-confidence.ts:138-144`: evidence (sections 30 + citations 25 + files 15 = 70/100) dominates; `SELF_ASSESSMENT_BONUS = 10` only adds a capped bonus or subtracts (`GAP_PENALTY_CAP = 20`), never substitutes | CLOSED |
| T-164-08 | Information Disclosure | `fs.existsSync` on worker-supplied path | `.aether/ts-host/src/research-confidence.ts:262-272`: `path.resolve(repoRoot, rawPath)` + `path.relative` escape check (`relative.startsWith("..")`  \|\| `path.isAbsolute(relative)`) skips any path escaping `repoRoot`; only a boolean returned | CLOSED |
| T-164-09 | DoS | pathological research markdown | `.aether/ts-host/src/research-confidence.ts:159-179` (`splitSections`) is a single forward pass, no backtracking-prone regex; `MAX_FILES_CHECKED = 50` (line 130, applied at line 261) caps existence checks | CLOSED |
| T-164-10 | Tampering | `renderGranularityReason` | `cmd/plan_depth_proposal.go:60-81`: goal text is only measured via `strings.Fields` word count; the reason string is drawn solely from the fixed `granularityReasons` map | CLOSED |
| T-164-11 | Tampering | depth values reaching CLI flags | `cmd/plan_depth_proposal.go` (`buildGranularityKnob`/`buildPlanningDepthKnob`/`buildVerificationDepthKnob`) only emits values from the fixed option sets. `resolvePlanningDepth` (`cmd/codex_plan.go:1241-1254`) explicitly errors on an unrecognized value. **Caveat:** `resolveVerificationDepthSmart` (`cmd/review_depth.go:355-361`) does not error on an unknown value — it calls `colony.NormalizeVerificationDepth`, whose `default` case silently coerces to `VerificationDepthStandard` rather than rejecting. This is a degrade-safe fallback (no privilege escalation, no arbitrary value reaches downstream logic) but the mitigation text's claim that verification depth "rejects unknown values" is not literally accurate — it silently normalizes instead. Not a blocker: the card itself never emits an out-of-set value, and the fallback is safe. | CLOSED (documentation caveat noted) |
| T-164-12 | Spoofing | card presenting a recommendation the runtime did not compute | `cmd/plan_depth_proposal.go:153-211` — `computeDepthProposal` and `renderDepthProposalCard` are both pure Go functions; Plan 09 wires the wrapper to print the result verbatim (see T-164-26 below) | CLOSED |
| T-164-13 | Tampering | `--flip` phase ID parsing | `cmd/phase_research_decision_cmd.go:60-79` (`parseFlipPhaseIDs`): `strconv.Atoi` per token, matched only against `candidateIDs` (in-scope unresolved decisions); unparsable/unknown tokens land in `invalid_flips`, never mutate a record | CLOSED |
| T-164-14 | Elevation of Privilege | stale decision from a different goal/session gating dispatch | `cmd/phase_research_decision_cmd.go:145,165` use `pendingDecisionMatchesScope` on every read and `stampPendingDecisionScope` on every write | CLOSED |
| T-164-15 | DoS | unanswered batch silently disabling research | `cmd/codex_plan.go:251-252,811-817,1175-1176` — `research_awaiting_approval` / `research_warning` present in manifest struct and both result-map branches | CLOSED |
| T-164-16 | DoS | unbounded research iteration | `.aether/ts-host/src/host.ts:803,828` — `researchLoopOptions(depth, RESEARCH_LOOP_DEFAULT_BUDGET)` binds `maxIterations`/`confidenceTarget` per depth tier; `ConfidenceLoop` (unmodified, `git diff` empty) enforces stop conditions | CLOSED |
| T-164-17 | Tampering | `--target`/`--max-iterations` forcing a costly run | `.aether/ts-host/src/host.ts:708-715` (`clampResearchConfidenceTarget` 70-99, `clampResearchMaxIterations` 2-12) applied at lines 807,813 before entering `ConfidenceLoopOptions` | CLOSED |
| T-164-19 | Tampering | research markdown reaching Route-Setter prompt | `cmd/codex_plan.go:930-945` (`renderRouteSetterResearchContent`) wraps content via `resolvePhaseResearchSection`, which frames it under `## Phase Research`, appended after the Route-Setter's own mission text | CLOSED |
| T-164-20 | DoS | oversized research drowning planner brief | `cmd/codex_plan.go:919` (`routeSetterResearchBudgetChars = 12000`) plus per-phase `phaseResearchBriefBudgetChars = 3500` (`cmd/phase_research.go:202`); over-budget phases degrade to a closing pointer line | CLOSED |
| T-164-21 | Repudiation | plan silently shipping without claimed research | `cmd/codex_plan_finalize.go:384,465-466` — `research_failed_phases` + `research_warning` on finalize result, independent of the artifact's own `**Research status:**` line | CLOSED |
| T-164-22 | Elevation of Privilege | escalating scout → oracle broadening write access | `pkg/codex/permission_profile.go:99-100` — explicit `case "oracle":` returns a scoped restriction (`.aether/oracle`, `.aether/data/phase-research`); previously fell to `default: return nil`. `ResolvePermissionProfile` (lines 116-130) still enforces exact-equality | CLOSED |
| T-164-23 | Tampering | phase ID crossing host→Go boundary | `cmd/phase_research_escalate.go:49-56,98-103` (`findPhaseResearchCandidate`) returns `ok=false` for unmatched IDs; caller emits `outputError`, never a panic | CLOSED |
| T-164-24 | DoS | repeated escalation of same phase | `.aether/ts-host/src/host.ts:837-901,919-944` — a phase enters `escalationCandidates` at most once (only when its own loop finishes), the escalation round runs once after the loop drains, and `isEscalationEligibleDepth` gates by depth | CLOSED |
| T-164-25 | Tampering | depth values reaching CLI flags (Plan 09 wiring) | Same code path as T-164-11 (`computeDepthProposal`/card); see T-164-11 note re: verification-depth's silent-normalize fallback | CLOSED (documentation caveat noted) |
| T-164-26 | Spoofing | wrapper inventing a recommendation the runtime did not compute | `.claude/commands/ant/plan.md` / `.opencode/commands/ant/plan.md` instruct printing `result.depth_proposal_card` / `result.research_proposal_card` verbatim; guardrail "Do NOT compose depth recommendations, reasons, or research recommendations in this wrapper" present in both wrappers; `TestPlanWrapperCardsParity` (run, PASS) asserts ordered-heading parity and runtime-key references | CLOSED |
| T-164-27 | Repudiation | autopilot silently choosing depth/research scope | `cmd/phase_research_decision_cmd.go` `--auto` path records `auto-accepted (autopilot)` resolutions via `phaseResearchDecisionResolution`; wrapper instructs printing the returned `log_line` under `/ant-run` (confirmed present in `.claude/commands/ant/plan.md`) | CLOSED |
| 164-10/T-01 | Elevation of Privilege | `toWorkerDispatches` permission_profile copy-through | `.aether/ts-host/src/host.ts:411-413` (source) and `.aether/ts-host/dist/host.js:413` (compiled) both copy `dispatch.permission_profile` through verbatim, never construct one; `ResolvePermissionProfile` still enforces exact-equality on arrival | CLOSED |
| 164-10/T-02 | Elevation of Privilege | TS `permissionProfileForCaste` fallback widening scout | `.aether/ts-host/src/worker-dispatch.ts:397` and compiled `dist/worker-dispatch.js:257` — read-only predicate narrowed to `normalized === "includer"` only, matching `pkg/codex/permission_profile.go`'s `repositoryReadOnlyCastes`; cross-language invariant test (`dispatch-field-fidelity.test.ts`, run, PASS) derives the expected set from the Go source at test time | CLOSED |
| 164-10/T-04 | Information Disclosure | wrapper docs claiming a sandbox guarantee the runtime doesn't enforce | `grep -rn "Scout.*repository_read_only\|repository_read_only.*Scout"` across `.aether/commands/plan.yaml`, all three wrapper markdown files, and `cmd/command_guide.go` returns zero matches (verified directly) | CLOSED |
| 164-11/T-05 | DoS | unwrapped `_callGoJSONRef` for `plan-research-escalate` | `.aether/ts-host/src/host.ts:927-943` (per-candidate call) and `945-961` (dispatch wave) both wrapped in try/catch that emits a warning and continues; verified live by running `research-escalation-degrade.test.ts` (3/3 PASS) against the real compiled `dist/host.js` behavior class | CLOSED |
| 164-11/T-07 | Elevation of Privilege | escalated Oracle dispatch for a phase resolved from the draft | `cmd/phase_research_escalate.go:41` still calls `attachPlanningDispatchSkillAssignments`, which stamps `codex.PermissionProfileForCaste("oracle")`; Plan 11 changes phase resolution only, not the profile path — confirmed unchanged | CLOSED |

**Mitigate threats closed: 27 / 27** (all `mitigate`-disposition threats in the register, including the 6 gap-closure `mitigate` threats from Plans 10–11).

---

## Accepted Risks Log (accept disposition — 5 threats)

The following risks were declared `accept` at plan time. Each rationale was
re-verified against current code (not merely re-read from the plan) before
being logged here as accepted.

| Threat ID | Category | Component | Rationale (verified against current code) | Status |
|---|---|---|---|---|
| T-164-02 | Information Disclosure | survey doc paths in the research brief | Paths are repository-relative under `.aether/data/survey` (`cmd/phase_research.go:164`), the same shape `renderPlanningWorkerBrief` already emits. Scout's canonical permission profile remains `workspace_write` with a behavioral restriction scoping writes to `.aether/data/phase-research` (`pkg/codex/permission_profile.go:93-94`) — confirmed unchanged by this phase. Accepted: repo-relative path disclosure to a worker that already reads the repo is not a meaningful boundary crossing. | ACCEPTED |
| T-164-05 | Tampering | hint signals over/under-triggering a research recommendation | `computePhaseResearchHint` (`cmd/phase_research_decision.go:125-155`) only feeds `Recommend`/`Reason` — cosmetic/cost outcome (one extra or missing dispatch), never a permission or access-control decision. Confirmed no hint value reaches `permission_profile` or any gating logic beyond the research/skip recommendation. Accepted: worst case is cost/quality, not a security boundary crossing. | ACCEPTED |
| T-164-18 | Information Disclosure | ceremony lines echoing research content to the terminal | `renderIterationCeremony` (`.aether/ts-host/src/host.ts:676-687`) constructs lines only from `iterationCount`, `currentConfidence`, `delta`, `budgetRemaining`, and `stopReason` — confirmed no markdown body text from `phase-N-research.md` is interpolated into any ceremony or escalation line. Accepted: matches the pre-existing build-path ceremony contract. | ACCEPTED |
| 164-10/T-03 | Tampering | `brief` promoted to `task_brief` and injected into worker prompt | `renderPhaseResearchBrief` (`cmd/phase_research.go:123-156`) builds the brief from colony-owned state (goal, phase name/description, survey context) — not user-supplied at dispatch time, and this plan introduces no new author of that content; it only fixes the pipe that was silently dropping the field. Accepted: no new trust boundary, existing prompt-injection sanitization applies to a different field class (pheromones). | ACCEPTED |
| 164-11/T-06 | Tampering | `previous_plan_draft` in `planning/iteration-state.json` steering escalation | Confirmed this file lives under `.aether/data/` (protected path, colony-owned scratch), written only by the plan-finalize path, and is the same seed source `planningManifestIterationSeed` already uses for the plan-only path (`cmd/codex_plan.go`). Escalated Oracle's write scope is unchanged and still bounded by `behavioralRestrictionsForCaste("oracle")`. Accepted: reading this file here narrows a prior divergence (CR-03) rather than widening trust. | ACCEPTED |

**Accepted risks logged: 5 / 5.**

---

## Test Evidence (executed during this audit, not merely re-read)

The following were run directly against the current tree to confirm claimed
mitigations function, rather than trusting PLAN/SUMMARY claims:

```
go test ./cmd/... -run 'TestPlanResearchDispatchResolvesAtWorkerBoundary|TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary|TestPlanResearchDispatchCarriesSixSectionMission|TestOracleEscalationDispatchNamesTheStall|TestOracleEscalationKeepsScopedWriteAccess' -v -count=1
  => ALL PASS (13 subtests)

go test ./cmd/... -run 'TestPlanWrapperCardsParity|TestPlanManifestCarriesDepthProposal|TestPlanManifestCarriesResearchProposal|TestPlanManifestWarnsWhenResearchBatchUnanswered|TestResearchDispatchGatedOnApproval|TestPlanResearchApproveRecordsDecisions' -v -count=1
  => ALL PASS

(cd .aether/ts-host && node --import tsx --import ./test/ensure-aether-binary.ts --test test/dispatch-field-fidelity.test.ts test/research-escalation-degrade.test.ts)
  => 7/7 PASS (field-fidelity invariant test + real-binary escalation degrade test)

go build ./cmd/aether && go vet ./cmd/... ./pkg/codex/...
  => clean

grep -rn "Scout.*repository_read_only|repository_read_only.*Scout" .aether/commands/plan.yaml .claude/commands/ant/plan.md .claude/commands/ant-plan.md .opencode/commands/ant/plan.md cmd/command_guide.go
  => no matches (stale sandbox claim confirmed removed)
```

---

## Unregistered Flags

None. Every SUMMARY.md `## Threat Flags` section (01, 03, 06, 08, 10, 11) reported
"None" and each explicitly cross-referenced its own plan's threat model as
covering all new attack surface introduced. No new endpoints, auth paths, file
access patterns, or schema changes were found during this audit's code reading
that fall outside the 34-entry register.

---

## Summary

**Threats Closed:** 34 / 34 (27 mitigate + 5 accept, all now CLOSED; 2 threats
— T-164-11 and T-164-25 — carry a documentation caveat about
`resolveVerificationDepthSmart`'s silent-normalize behavior rather than an
explicit-reject, but the underlying security property holds because the card
never emits an out-of-set value and the fallback default is safe.)

**Threats Open:** 0

No implementation files were modified during this audit. This document is the
first SECURITY.md for this phase (State B).
