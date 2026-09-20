# Build Check-In and Spawn Economy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Builds pause for owner approval of the worker team, workers' open questions route to the owner instead of sibling agents, and five confirmed useless-spawn leaks are closed.

**Architecture:** All gating changes land in the existing Queen selection chain (`cmd/caste_relevance.go`, `cmd/queen_spawn_budget.go`). Two new read-only runtime commands (`ceremony team-checkin`, `handoff-decisions`) and one writer (`decision-answer`) reuse the manifest, handoff store, and pending-decisions/CLARIFIED-INTENT machinery. Wrapper changes are presentation-only edits to `build.md`/`continue.md`, mirrored byte-identically across `.claude` and `.opencode`.

**Tech Stack:** Go (cobra CLI, `cmd/` package tests with `go test ./cmd/ -run <Name>`), markdown wrappers.

**Spec:** `docs/superpowers/specs/2026-08-21-build-checkin-and-spawn-economy-design.md`

## Global Constraints

- Branch: `build-checkin-spawn-economy`. Never push. Commit messages: imperative, <72 chars.
- **Pre-existing red baseline:** 11 visual tests fail on this branch's parent (`TestPlanVisualOutput`, `TestBuildVisualOutputShowsSpawnPlan`, `TestColonizeVisualOutputShowsDispatchPreview`, `TestContinueBlockedVisualOutputShowsWorkerFlow`, `TestContinueVisualOutputShowsColonyCompleteStageMarker`, `TestPrintNextUpVisualOutput`, `TestRenderBinaryActionVisualPublishGuidanceSeparatesRepoSetupFromUpdate`, `TestRenderUpdateVisualShowsRemovedAssets`, `TestSetupVisualOutput`, `TestPauseResumePatrolPhaseAndHistoryVisualOutput`, `TestCeremonyCloseoutBlockedPathRendersBlockedNotCompletion`) — they expect `/ant-*` spelling while the runtime emits `aether ...`. Do NOT fix them; the completion bar is **zero NEW failures**.
- Wrapper command files exist in TWO mirrored copies: `.claude/commands/ant/<name>.md` and `.opencode/commands/ant/<name>.md`. Every wrapper edit is applied byte-identically to both. `TestBuildWrapperStageSkeletonAndParity` enforces heading parity; new stages need the Purpose/Reads/Stop-conditions skeleton.
- Wrappers never mutate state; all data comes from runtime commands. New inspection commands (`team-checkin`, `handoff-decisions`) must not write anything — each gets a does-not-mutate test.
- Never write to `.aether/data/` directly in tests; use the test store helpers existing tests use (grep `newTestStore\|setupTestStore` in `cmd/*_test.go` and copy the prevailing fixture pattern).
- `casteRelevanceRegistry`, `isAlwaysRequired`, `casteAllowedForFlow`, `isCasteSuppressed` live in `cmd/caste_relevance.go` (450 lines; line refs below are current as of commit 2ee5c8ff).

---

### Task 1: Seal's Probe requires testable code (C1)

**Files:**
- Modify: `cmd/caste_relevance.go:343-351` (the `case "seal":` branch of `isAlwaysRequired`)
- Test: Create `cmd/queen_seal_gate_test.go`

**Interfaces:**
- Consumes: `queenOrchestrate(phase, "seal", state)`, `HasCaste(dispatches, caste)`, `queenPhaseProducesTestableCode(phase)` — all existing in `cmd`.
- Produces: no new symbols; behavior change only.

- [ ] **Step 1: Write the failing test.** Copy the doc-only and code-producing phase fixtures from the existing `TestProbeIsRequiredOnlyWhereItCanFindSomething` (grep for it in `cmd/`; reuse its `colony.Phase` literals verbatim so "produces testable code" means the same thing in both tests). Then:

```go
func TestSealProbeRequiresTestableCode(t *testing.T) {
	docPhase := /* doc-only phase fixture copied from TestProbeIsRequiredOnlyWhereItCanFindSomething */
	codePhase := /* code-producing phase fixture from the same test */
	standard := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthStandard)}
	heavy := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthHeavy)}

	if HasCaste(queenOrchestrate(docPhase, "seal", standard), "probe") {
		t.Fatalf("standard seal of a documentation-only phase must not require a Probe: there is no code for it to cover")
	}
	if !HasCaste(queenOrchestrate(codePhase, "seal", standard), "probe") {
		t.Fatalf("standard seal of a code-producing phase must keep its Probe")
	}
	if !HasCaste(queenOrchestrate(docPhase, "seal", heavy), "probe") {
		t.Fatalf("heavy seal is an explicit request for the full gauntlet; Probe stays even on a doc phase")
	}
	if !HasCaste(queenOrchestrate(docPhase, "seal", standard), "auditor") {
		t.Fatalf("gating Probe must not disturb the seal Auditor")
	}
}
```

- [ ] **Step 2: Run it, verify it fails.** `go test ./cmd/ -run TestSealProbeRequiresTestableCode -count=1` → FAIL on the first assertion (probe currently unconditionally required at standard seal).
- [ ] **Step 3: Implement.** In `isAlwaysRequired`, `case "seal":`, default branch — change

```go
		default:
			return caste == "auditor" || caste == "probe"
```

to

```go
		default:
			// Probe on a documentation-only final phase has no code to cover —
			// the same gate build and continue already apply. Heavy stays
			// unconditional above: heavy is an explicit request for breadth.
			return caste == "auditor" || (caste == "probe" && queenPhaseProducesTestableCode(phase))
```

- [ ] **Step 4: Run the test, verify PASS**, then `go test ./cmd/ -run 'TestSeal|TestQueen' -count=1` for collateral.
- [ ] **Step 5: Commit** `fix: gate seal probe on phase producing testable code`

---

### Task 2: Swarm stops always-summoning Scout and Archaeologist (C3)

**Files:**
- Modify: `cmd/caste_relevance.go:334-342` (the `case "swarm":` branch of `isAlwaysRequired`)
- Test: Create `cmd/queen_swarm_gate_test.go`

**Interfaces:**
- Consumes: the swarm synthetic-phase constructor in `cmd/swarm_cmd.go:856-882` (grep `func swarm` there for its exact name — call the REAL constructor in the test, not a hand-built phase, because its fixed task text could contain caste keywords and hand-built fixtures would hide that).
- Produces: behavior change only.

- [ ] **Step 1: Write the failing test.**

```go
func TestSwarmTrivialBugSkipsHistoryAndResearch(t *testing.T) {
	phase := /* real swarm phase constructor */ ("fix typo in README header")
	state := colony.ColonyState{}
	got := queenOrchestrate(phase, "swarm", state)
	for _, core := range []string{"tracker", "builder", "watcher"} {
		if !HasCaste(got, core) {
			t.Fatalf("swarm must always keep %s", core)
		}
	}
	if HasCaste(got, "archaeologist") {
		t.Fatalf("a trivial typo hunt must not summon a git-history specialist")
	}
	if HasCaste(got, "scout") {
		t.Fatalf("a trivial typo hunt must not summon a research specialist")
	}
}

func TestSwarmHistoryBugKeepsArchaeologist(t *testing.T) {
	phase := /* real swarm phase constructor */ ("regression after legacy migration refactor of the history module")
	if !HasCaste(queenOrchestrate(phase, "swarm", colony.ColonyState{}), "archaeologist") {
		t.Fatalf("a bug rooted in legacy/migration history should still select the Archaeologist on keyword relevance")
	}
}
```

- [ ] **Step 2: Run, verify first test fails** (scout/archaeologist currently always required). If it fails instead because the synthetic phase's own boilerplate task text trips scout/archaeologist keywords past threshold 35, that boilerplate is part of the bug: neutralize the keyword-bearing words in the constructor's fixed task text as part of Step 3 (e.g. "Inspect prior fixes around the bug area" instead of "git history"), keeping the per-dispatch wave task texts in `swarm_cmd.go:919` unchanged — those are instructions to an already-selected worker, not selection inputs. Verify with `git grep -n "collectPhaseText"` which fields feed selection before deciding.
- [ ] **Step 3: Implement.** In `isAlwaysRequired`, `case "swarm":` — change

```go
		return caste == "tracker" ||
			caste == "scout" ||
			caste == "archaeologist" ||
			caste == "builder" ||
			caste == "watcher"
```

to

```go
		// Scout and Archaeologist used to be unconditionally required, so a
		// one-line typo fix paid for a researcher and a git-history dig that
		// could only report finding nothing. They now ride on keyword
		// relevance like every other specialist; the investigate/fix/verify
		// trio stays mandatory because a swarm without them is not a swarm.
		return caste == "tracker" ||
			caste == "builder" ||
			caste == "watcher"
```

(the `gatekeeper`+high-risk rule above this stays untouched)

- [ ] **Step 4: Run both tests → PASS; run `go test ./cmd/ -run 'Swarm' -count=1`** and update any existing swarm-roster expectations that asserted the five-caste floor (adjust them to the new trio + relevance rule, keeping their intent).
- [ ] **Step 5: Commit** `fix: swarm summons scout/archaeologist on relevance, not always`

---

### Task 3: No build-selectable caste without a build dispatch path (C2)

**Files:**
- Modify: `cmd/caste_relevance.go:368-371` (`casteAllowedForFlow`, `case "build":`), `cmd/codex_build.go:1266-1349` (extract dispatch-table caste names into package vars if they are inline literals)
- Test: Create `cmd/queen_build_dispatchability_test.go`

**Interfaces:**
- Produces: `queenBuildPreWaveCastes []string`, `queenBuildPostWaveCastes []string`, `queenBuildTaskFallbackCastes []string` — package-level vars in `cmd/codex_build.go` naming exactly the castes those code paths can dispatch (extracted from the existing literals at :1266-1308, :1310-1332, :1334-1349; the dispatch code iterates these vars so table and code cannot drift).

- [ ] **Step 1: Write the failing test** (invariant style, per the repo's stated preference for invariants over named sections):

```go
// A caste the Queen may select for a build must be able to produce at least
// one dispatch in the build composer. route_setter passed the flow filter but
// appeared in no dispatch table, so selecting it consumed a budget slot and
// silently displaced a specialist that would actually have run.
func TestEveryBuildSelectableCasteCanDispatch(t *testing.T) {
	dispatchable := stringSet(append(append(append([]string{"watcher", "probe", "builder"},
		queenBuildPreWaveCastes...), queenBuildPostWaveCastes...), queenBuildTaskFallbackCastes...))
	for _, profile := range casteRelevanceRegistry {
		if !casteAllowedForFlow(profile.Caste, "build") {
			continue
		}
		if !dispatchable[profile.Caste] {
			t.Errorf("caste %q is selectable for build but no build dispatch path can spawn it — it would waste a budget slot", profile.Caste)
		}
	}
}
```

- [ ] **Step 2: Run, verify it fails on `route_setter`** (after the extraction refactor compiles). Extraction must be behavior-neutral: run `go test ./cmd/ -run 'TestBuild' -count=1` before and after to prove it.
- [ ] **Step 3: Implement the fix.** In `casteAllowedForFlow`:

```go
	case "build":
		// route_setter plans phases; the build composer has no dispatch for
		// it, so selecting it only displaced a real specialist.
		return caste != "route_setter" && !strings.HasPrefix(caste, "surveyor-")
```

- [ ] **Step 4: Test passes; full `go test ./cmd/ -count=1 -run 'TestQueen|TestBuild'` green.**
- [ ] **Step 5: Commit** `fix: exclude dispatch-less castes from build selection`

---

### Task 4: Continue's fast path honors the Queen's `--castes` proposal (C4a)

**Files:**
- Modify: `cmd/codex_continue.go:844` (call site), `:1319` (`runCodexContinueReview` signature), `:1343` (pass-through), `:1419` (`plannedContinueReviewDispatches` signature), `:1428` (use judgement variant)
- Test: Create `cmd/continue_fastpath_castes_test.go`

**Interfaces:**
- Consumes: `codexContinueOptions.QueenCastes []string` / `.QueenCasteReason string` (`cmd/codex_continue.go:150-157`), `queenContinueReviewSpecsWithJudgement(phase, reviewDepth, proposed, reason)` (`:1257`).
- Produces: `runCodexContinueReview(root, phase, manifest, verification, assessment, workerTimeout, reviewDepth, skipWatchers, queenCastes []string, queenCasteReason string)` and `plannedContinueReviewDispatches(..., reviewDepth, queenCastes []string, queenCasteReason string)` — update every caller (grep both names; non-test callers today: `codex_continue.go:844`, `:1343`).

- [ ] **Step 1: Write the failing test.** Mirror the fixture style of `cmd/queen_orchestration_regression_test.go:32` (it already builds a phase and calls `queenContinueReviewSpecs`). Test at the dispatch level:

```go
func TestContinueFastPathHonoursCasteProposal(t *testing.T) {
	phase := /* standard non-security phase fixture, same style as queen_orchestration_regression_test.go */
	base := plannedContinueReviewDispatches(t.TempDir(), phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, nil, "")
	proposed := plannedContinueReviewDispatches(t.TempDir(), phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, &codex.FakeInvoker{}, time.Minute, colony.VerificationDepthStandard, []string{"measurer"}, "perf phrasing without perf keywords")
	if containsDispatchCaste(base, "measurer") {
		t.Fatalf("fixture broken: measurer must not be in the unproposed baseline")
	}
	if !containsDispatchCaste(proposed, "measurer") {
		t.Fatalf("the fast continue path ignored the Queen's --castes proposal; only the heavy plan-only path honoured it")
	}
}
```

with a small local helper `containsDispatchCaste(ds []codex.WorkerDispatch, caste string)` checking `WorkerDispatch.Caste`. If `plannedContinueReviewDispatches` needs a non-nil store or state, copy the setup from whichever existing test already calls it (grep `plannedContinueReviewDispatches` in `cmd/*_test.go`).

- [ ] **Step 2: Run — it must fail to compile** (signature lacks the params). That IS the red state; proceed.
- [ ] **Step 3: Implement.** Thread the two params through both signatures; at `:844` pass `options.QueenCastes, options.QueenCasteReason`; at `:1428` replace `queenContinueReviewSpecs(phase, reviewDepth)` with `queenContinueReviewSpecsWithJudgement(phase, reviewDepth, queenCastes, queenCasteReason)`.
- [ ] **Step 4: Test passes; run `go test ./cmd/ -run 'Continue' -count=1`** and fix any caller/test the signature change touched (mechanical param additions of `nil, ""`).
- [ ] **Step 5: Commit** `fix: fast continue path honors queen caste proposal`

---

### Task 5: Keeper stops arriving on incidental words (C4b)

**Files:**
- Modify: `cmd/caste_relevance.go:45` (keeper registry entry)
- Test: Add to `cmd/continue_fastpath_castes_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestContinueDoesNotSummonKeeperOnIncidentalWords(t *testing.T) {
	incidental := /* phase whose name+tasks contain "document the standard pattern" but no knowledge-preservation intent */
	if HasCaste(queenOrchestrate(incidental, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("the words standard/document/pattern are everyday phase vocabulary; two incidental hits must not buy a Keeper run")
	}
	preservation := /* phase whose text contains "preserve knowledge and conventions for future workers" */
	if !HasCaste(queenOrchestrate(preservation, "continue", colony.ColonyState{}), "keeper") {
		t.Fatalf("a genuine knowledge-preservation phase should still select the Keeper")
	}
}
```

- [ ] **Step 2: Run, verify the first assertion fails** ("document"+"standard"+"pattern" = 3 hits × 10 + base 10 = 40 ≥ threshold 30 today).
- [ ] **Step 3: Implement.** Change the keeper entry to

```go
	{Caste: "keeper", Keywords: []string{"knowledge", "convention", "preserve", "wisdom", "institutional"}, BaseScore: 10},
```

("pattern", "standard", "document" removed — they are the vocabulary of ordinary engineering phases, not of knowledge preservation; "document" already belongs to chronicler).

- [ ] **Step 4: Both assertions pass; `go test ./cmd/ -run 'Keeper|Continue|Queen' -count=1`.**
- [ ] **Step 5: Commit** `fix: keeper keywords no longer match everyday phase words`

---

### Task 6: Colonize honors light depth (C5)

**Files:**
- Modify: `cmd/caste_relevance.go:329-333` (`case "colonize":` in `isAlwaysRequired`), `:387-404` (`isCasteSuppressed`), `cmd/queen_spawn_budget.go:310-311` (colonize budget)
- Test: Create `cmd/queen_colonize_depth_test.go`

- [ ] **Step 1: Write the failing test:**

```go
func TestColonizeLightDepthTrimsSurveyors(t *testing.T) {
	phase := /* the same synthetic colonize phase queenSurveyorSpecs builds — call the real helper in cmd/codex_colonize.go:648-662 if exported enough, else replicate its constant text */
	light := colony.ColonyState{VerificationDepth: string(colony.VerificationDepthLight)}
	standard := colony.ColonyState{}

	got := queenOrchestrate(phase, "colonize", light)
	if len(got) != 2 || !HasCaste(got, "surveyor-nest") || !HasCaste(got, "surveyor-provisions") {
		t.Fatalf("light colonize should send exactly the structure and dependency surveyors, got %+v", got)
	}
	full := queenOrchestrate(phase, "colonize", standard)
	for _, s := range []string{"surveyor-nest", "surveyor-provisions", "surveyor-disciplines", "surveyor-pathogens"} {
		if !HasCaste(full, s) {
			t.Fatalf("standard colonize must keep all four surveyors; missing %s", s)
		}
	}
}
```

- [ ] **Step 2: Run, verify FAIL** (light currently returns all four).
- [ ] **Step 3: Implement.** (a) In `isAlwaysRequired` `case "colonize":`, require only nest+provisions when `stateVerificationDepth(state) == colony.VerificationDepthLight`, all four otherwise. (b) In `isCasteSuppressed`, mirror the existing seal-light pattern at `:397-399`:

```go
	if flowType == "colonize" && stateVerificationDepth(state) == colony.VerificationDepthLight {
		return oneOf(caste, "surveyor-disciplines", "surveyor-pathogens")
	}
```

(c) In `queenMaxWorkersForBudget` `case "colonize":` return `2, "light territory survey"` when depth is light, else the existing `4, "territory survey"`.
- [ ] **Step 4: Test passes; `go test ./cmd/ -run 'Colonize' -count=1`.**
- [ ] **Step 5: Commit** `feat: light verification depth trims colonize to two surveyors`

---

### Task 7: `aether handoff-decisions` — read-only listing of workers' open questions (B1)

**Files:**
- Create: `cmd/handoff_decisions_cmd.go`
- Test: Create `cmd/handoff_decisions_cmd_test.go`

**Interfaces:**
- Consumes: `loadWorkerHandoffRecords()` (`cmd/codex_dispatch_contract.go:909`), `workerHandoffRecord` fields `OpenDecisions/WorkerName/Caste/Phase/Workflow/Freshness`, `loadPendingDecisionFile()`/`PendingDecisionFile` (`cmd/pending_decision.go`), `resolvedClarifiedIntentEntries` question texts (`cmd/discuss.go:911`), `outputOK`/`outputWorkflow` output helpers, `store` global.
- Produces: command `handoff-decisions` with flags `--phase int` (0 = all), `--json` implicit via `AETHER_OUTPUT_MODE`; JSON result `{"count": n, "decisions": [{"id": "<12-hex sha256 prefix of normalized text>", "question": ..., "worker": ..., "caste": ..., "phase": n, "workflow": ...}]}`; helper `pendingHandoffDecisions(phaseFilter int) []handoffDecision` used again by Task 12's continue surfacing.

- [ ] **Step 1: Write the failing tests** — three behaviors:

```go
func TestHandoffDecisionsListsUnansweredOpenDecisions(t *testing.T) {
	// seed store with one workerHandoffRecord carrying OpenDecisions:
	// ["Should exported CSV include archived rows?"] via store.UpdateFile on
	// workerHandoffsPath (copy the seeding shape from an existing handoff test —
	// grep workerHandoffsPath in cmd/*_test.go)
	// then: got := pendingHandoffDecisions(0)
	// assert len==1, question matches, worker/caste populated
}

func TestHandoffDecisionsExcludesAnsweredQuestions(t *testing.T) {
	// seed the same record PLUS a resolved clarification whose question text
	// normalizes equal (Task 8's decision-answer writes this shape);
	// assert pendingHandoffDecisions(0) is empty
}

func TestHandoffDecisionsDoesNotMutate(t *testing.T) {
	// snapshot the bytes of workerHandoffsPath and pending-decisions.json,
	// run the cobra command RunE, assert bytes unchanged
	// (copy the pattern from TestConsolidationPhaseEndDryRunDoesNotMutate)
}
```

- [ ] **Step 2: Run → FAIL (symbols undefined).**
- [ ] **Step 3: Implement.** Normalization for dedupe/ids: `strings.ToLower(strings.Join(strings.Fields(text), " "))`; id = `fmt.Sprintf("%x", sha256.Sum256([]byte(norm)))[:12]`. "Answered" = a `PendingDecision` with `Resolved && Resolution != ""` whose parsed question (`parseClarificationDescription(d.Description)`) OR raw description normalizes equal. Read-only: only `loadWorkerHandoffRecords` + `loadPendingDecisionFile`; no Save/Update calls anywhere in the command. Register in `init()` on `rootCmd` like `pendingDecisionListCmd`.
- [ ] **Step 4: Tests pass.**
- [ ] **Step 5: Commit** `feat: handoff-decisions lists workers' open questions`

---

### Task 8: `aether decision-answer` — one-shot record of an owner answer (B2)

**Files:**
- Create: append to `cmd/handoff_decisions_cmd.go`
- Test: Append to `cmd/handoff_decisions_cmd_test.go`

**Interfaces:**
- Consumes: `clarificationDecisionType` const and the clarification `Description` writer format used by discuss (grep `clarificationDecisionType` in `cmd/discuss.go` for the exact composer — reuse it so `parseClarificationDescription` round-trips), `stampPendingDecisionScope`/`loadCurrentPendingDecisionScope`, `store.SaveJSON`/`LoadJSON` (same load-modify-save as `pendingDecisionResolveCmd`), `clarifiedIntentPromptRenderResult()` (`cmd/discuss.go:1022`).
- Produces: command `decision-answer` with flags `--question` (required), `--answer` (required), `--phase int`, `--source` (default `worker-handoff`); JSON result `{"id": ..., "recorded": true, "prompt_section": "## CLARIFIED INTENT\n\n- <q> => <a>\n..."}` — `prompt_section` is the full freshly-rendered clarified-intent block, which wrappers append verbatim to later-wave worker prompts.

- [ ] **Step 1: Write the failing tests:**

```go
func TestDecisionAnswerRendersIntoClarifiedIntent(t *testing.T) {
	// run decision-answer RunE with --question "Should exported CSV include archived rows?" --answer "No — active rows only"
	// then: lines := clarifiedIntentPromptRenderResult().Lines
	// assert one line contains both the question and "No — active rows only"
}

func TestDecisionAnswerReturnsPromptSection(t *testing.T) {
	// capture the command's outputOK payload (follow how existing cmd tests
	// capture output — grep outputOK assertions) and assert prompt_section
	// starts with "## CLARIFIED INTENT" and contains the answer
}
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** Build the `PendingDecision` with `Type: clarificationDecisionType`, description via the discuss composer, `Resolved: true`, `Resolution: answer`, `ResolvedAt`/`CreatedAt` stamped, `Source: source`, optional `Phase`, scope stamped — then append+save exactly like `pendingDecisionAddCmd` does (load, append, `store.SaveJSON`). Render `prompt_section` by joining `clarifiedIntentPromptRenderResult().Lines` under the `## CLARIFIED INTENT` heading (mirror `colony_prime_context.go:877-883`).
- [ ] **Step 4: Tests pass. Also re-run Task 7's exclusion test** — an answer recorded through this verb must make `handoff-decisions` stop listing that question.
- [ ] **Step 5: Commit** `feat: decision-answer records owner rulings mid-build`

---

### Task 9: End-to-end wiring test — answered question reaches the next worker (B)

**Files:**
- Test: Create `cmd/open_decision_wiring_test.go`

- [ ] **Step 1: Write the test** (should pass immediately if Tasks 7-8 are correct — it exists as the Definition-of-Done lock):

```go
// The full relay: a worker leaves an open decision -> the owner answers it ->
// the answer is injected into subsequent worker context -> the question stops
// being listed. If any link breaks, workers go back to inheriting guesses.
func TestResolvedOpenDecisionReachesNextWorkerPrompt(t *testing.T) {
	// 1. seed a handoff record with the open decision (as in Task 7)
	// 2. assert pendingHandoffDecisions(0) lists it
	// 3. run decision-answer for it
	// 4. capsule := resolveCodexWorkerContext() (grep its test usage for setup)
	//    assert capsule/prompt content contains "CLARIFIED INTENT" and the answer
	// 5. assert pendingHandoffDecisions(0) no longer lists it
}
```

- [ ] **Step 2: Run → must PASS; if it fails, fix Tasks 7-8, not the test.**
- [ ] **Step 3: Commit** `test: lock open-decision relay from worker to next prompt`

---

### Task 10: `aether ceremony team-checkin` renderer (A, runtime half)

**Files:**
- Create: `cmd/ceremony_team_checkin.go`
- Modify: `cmd/ceremony_cmd.go` `init()` (register subcommand + flags, mirroring `ceremonySpawnPlanCmd` at `:78-89`, `:128-130`)
- Modify: `cmd/codex_workflow_cmds.go:1343` region — add `buildCmd.Flags().Bool("no-checkin", false, "Skip the pre-spawn team check-in pause (wrapper reads this; the runtime plan is unchanged)")`, and surface it in the plan-only result as `"checkin_requested": !noCheckin` next to where `dispatch_manifest` is attached (`cmd/codex_build.go:328-362`)
- Test: Create `cmd/ceremony_team_checkin_test.go`

**Interfaces:**
- Consumes: manifest JSON envelope (same file `spawn-plan` reads), keys: `dispatches[].caste/name/task`, `queen_execution_policy.spawn_budget.required_castes/selected_reasons/pruned_reasons` (`cmd/codex_dispatch_contract.go:551-587`), `caste_roster` produces-prose (`cmd/queen_judgement.go:335-356`); render helpers `renderOldStyleCeremonyHeader`, `writeCeremonyDispatchLine`, `casteIdentity`/`casteLabel` (`cmd/codex_visuals.go`), `outputWorkflow`.
- Produces: `renderCeremonyTeamCheckin(workflow string, manifest map[string]interface{}, dispatches []ceremonyDispatch) (map[string]interface{}, string)`; JSON result `{"required": ["watcher", ...], "optional": ["measurer", ...], "reasons": {caste: reason}, "pruned": {caste: reason}}` — the wrapper builds its trim question from THIS JSON, never by parsing the visual.

- [ ] **Step 1: Write the failing tests:**

```go
func TestTeamCheckinCardShowsReasonAndRequiredMarking(t *testing.T) {
	// build a manifest fixture: two dispatches (watcher, measurer);
	// spawn_budget.required_castes=["watcher"];
	// selected_reasons={"watcher": "always required...", "measurer": "Score 40 >= threshold 30..."};
	// pruned_reasons={"chaos": "pruned by worker budget..."}
	// result, visual := renderCeremonyTeamCheckin("build", manifest, dispatches)
	// assert visual contains "REQUIRED" on the watcher line and "OPTIONAL" on measurer's,
	// each caste's reason text, and a "Not sent" line naming chaos with its reason;
	// assert result JSON buckets watcher under required, measurer under optional
}

func TestTeamCheckinDoesNotMutate(t *testing.T) { /* same snapshot pattern as Task 7 */ }
```

- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement.** Card layout: header via `renderOldStyleCeremonyHeader(commandEmoji("build"), "Team Check-In")`; one line per unique dispatch caste — caste identity in house style, `REQUIRED`/`OPTIONAL` tag (required = membership in `spawn_budget.required_castes`), an em-dash, then the reason from `selected_reasons[caste]` falling back to the roster's `produces` prose; then `Not sent:` block from `pruned_reasons`; footer: `Required workers stay — they are the safety floor. Optional workers can be trimmed.`
- [ ] **Step 4: Tests pass; run `go test ./cmd/ -run 'Ceremony' -count=1`.**
- [ ] **Step 5: Commit** `feat: ceremony team-checkin renders the approvable roster`

---

### Task 11: Team Check-In stage in the build wrapper (A, wrapper half)

**Files:**
- Modify: `.claude/commands/ant/build.md` (insert new `## Team Check-In` stage between `## Runtime Spawn Ceremony` ending ~line 239 and `## Worker Spawning` at ~line 241)
- Modify: `.opencode/commands/ant/build.md` (byte-identical edit)
- Modify: `.aether/commands/build.yaml` IF the source-check test compares stage lists (run `go test ./cmd/ -run 'SourceCheck|Wrapper' -count=1` first to find out; add the matching step only if a test demands it)
- Test: Extend the required-strings list in `cmd/platform_doc_hygiene_test.go` (~line 200 block) with the team-checkin command line; `TestBuildWrapperStageSkeletonAndParity` must stay green

**Stage text to insert (identical in both files):**

```markdown
## Team Check-In

🐜 The colony shows its team; the owner has the last word before anyone moves.

**Purpose:** Pause after the spawn plan renders and let the user approve, trim, or redirect the team before any worker spawns. Required workers are the safety floor and are never offered for removal — the runtime re-adds them regardless, so offering the choice would be a lie.

**Reads:** the manifest file written in Dispatch Manifest, and `result.checkin_requested` from the plan-only result.

If `checkin_requested` is false (`--no-checkin` was passed), skip this stage entirely.

1. Render the runtime-owned check-in card:

```
AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony team-checkin --workflow build --manifest-file <manifest_file>
```

2. Fetch the same card as data: `AETHER_OUTPUT_MODE=json aether ceremony team-checkin --workflow build --manifest-file <manifest_file>` and read `result.required`, `result.optional`, `result.reasons`.
3. Ask the user (AskUserQuestion, single question): "The Queen picked this team. Proceed?" with options:
   - "Proceed with this team" (recommended) — spawn as planned.
   - "Trim optional workers" — follow up with ONE multi-select question listing ONLY `result.optional` entries, each labeled with its plain-English job and reason. Never list a required caste.
   - "Redirect first" — route to `aether discuss`, then request a fresh manifest exactly as the Guided Boundary Gate does; never reuse the pre-discuss manifest.
4. On trim: re-fetch `AETHER_OUTPUT_MODE=json aether build $ARGUMENTS --plan-only --castes <kept optional castes> --caste-reason "owner check-in trim"`, overwrite the manifest file with the new manifest, re-render the spawn ceremony for the new plan, and record the preference: `AETHER_OUTPUT_MODE=json aether memory-capture "owner trimmed <dropped castes> from the phase <n> build team"`. Relay `caste_decision.summary` in plain English — anything the runtime added back must be said out loud.
5. Autopilot (`/ant-run`) never runs this stage — it does not run this wrapper.

**Stop conditions:** The user has been asked and their pick applied. Never spawn from a manifest the user asked to trim without re-fetching it.
```

- [ ] **Step 1: Make the hygiene test fail first.** Add `"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony team-checkin --workflow build --manifest-file <manifest_file>"` to the build.md required-strings list in `cmd/platform_doc_hygiene_test.go`; run `go test ./cmd/ -run 'TestLifecycleCommandDocsPreferRuntimeCLI' -count=1` → FAIL (string absent).
- [ ] **Step 2: Insert the stage text into BOTH build.md files** (byte-identical; keep the Guided Boundary Gate ahead of it, unchanged).
- [ ] **Step 3: Run** `go test ./cmd/ -run 'TestLifecycleCommandDocsPreferRuntimeCLI|TestBuildWrapperStageSkeletonAndParity|TestBuildWrapperCeremonyContract|TestBuildMdOwnershipHandshake' -count=1` → all PASS (fix skeleton-density complaints by keeping the Purpose/Reads/Stop-conditions structure above).
- [ ] **Step 4: Commit** `feat: build wrapper pauses at a team check-in before spawning`

---

### Task 12: Wave-boundary questions + post-build/continue surfacing (B, wrapper half)

**Files:**
- Modify: `.claude/commands/ant/build.md` + `.opencode/commands/ant/build.md` — Worker Spawning wave loop (~lines 266-275) and After the Build (~lines 306-323)
- Modify: `.claude/commands/ant/continue.md` + `.opencode/commands/ant/continue.md` — Suggested Steering checkpoint (~lines 228-235)
- Test: `TestBuildWrapperVerbatimBriefBulletsStayMirrored` and `TestBuildWrapperStageSkeletonAndParity` updated/green; continue's equivalent parity test green

**Wave-loop addition** — append as step 8 of the "For each manifest wave" list (both files, identical):

```markdown
8. After the wave's workers return: collect `handoff.open_decisions` from their terminal results. For each question not already answered this build (compare normalized text), ask the user (AskUserQuestion, at most 4 per wave; carry extras to the next boundary). Every question gets the option "Let the colony proceed on its current assumption" — an unanswered question never blocks the build. For each real answer, record it: `AETHER_OUTPUT_MODE=json aether decision-answer --question "<q>" --answer "<a>" --phase <n>` and keep the returned `prompt_section`. For every LATER wave's workers, append the newest `prompt_section` verbatim after `dispatch.skill_section` — it is runtime-rendered owner steering, delivered exactly like the capsule and brief.
```

Also amend the prompt-assembly sentences (the line-259 bullet and step 5 at line 272) to name the optional trailing section: `+ the newest decision-answer prompt_section when one exists (runtime-rendered; never wrapper-written)`. Update the mirrored-bullet expectations in `cmd/build_wrapper_ceremony_test.go` (`TestBuildWrapperVerbatimBriefBulletsStayMirrored`) to the new sentence.

**After the Build addition** (both files): before the existing AskUserQuestion, run `AETHER_OUTPUT_MODE=json aether handoff-decisions --pending --phase <n>`; if `count > 0`, add a fourth option "Answer the workers' open questions first (<count> waiting)" which walks the same ask-and-`decision-answer` loop, then re-asks the original question. Sync `.aether/commands/build.yaml` post_build step 5's option list if the source-check test compares it.

**continue.md addition** (both files, in the Suggested Steering stage): run `AETHER_OUTPUT_MODE=json aether handoff-decisions --pending --phase <n>`; if any, ask them (same ≤4 rule and "proceed on assumption" option) and record via `decision-answer` before rendering the steering multi-select.

- [ ] **Step 1: Update test expectations first** (mirrored-bullet strings + any continue hygiene strings) → run, verify FAIL.
- [ ] **Step 2: Apply the four wrapper edits byte-identically per platform pair.**
- [ ] **Step 3: Run** `go test ./cmd/ -run 'Wrapper|Hygiene|Parity|LifecycleCommandDocs' -count=1` → PASS.
- [ ] **Step 4: Commit** `feat: workers' open questions reach the owner at wave boundaries`

---

### Task 13: Worker-side rule — ask the owner, don't research around it

**Files:**
- Modify: the open_decisions instruction text everywhere it is stated to workers: grep `"choices the next worker or Queen must make"` and `open_decisions` across `cmd/*.go`, `pkg/codex/*.go`, `.aether/references/contracts/worker-handoff-contract.md:36`
- Test: extend whichever existing contract-text test asserts the handoff instruction (grep the current sentence in `cmd/*_test.go`); add the new sentence to its expectations

- [ ] **Step 1: Update the test expectation first** with the added sentence → FAIL.
- [ ] **Step 2: Append to the open_decisions contract text at every site (identical sentence):** `When you hit a preference or product judgement the project owner could answer, do not research around it or guess silently: record the question in open_decisions, take the least-committal path, and note the assumption in assumptions.`
- [ ] **Step 3: Tests PASS.**
- [ ] **Step 4: Commit** `feat: workers route judgement calls to open_decisions, not guesses`

---

### Task 14: Full verification, docs, changelog

- [ ] **Step 1:** `go build ./cmd/aether && go vet ./...`
- [ ] **Step 2:** `go test ./pkg/... -count=1` → all green. `go test ./cmd/ -count=1` → compare failures against the 11-test baseline in Global Constraints; **zero new failures allowed**.
- [ ] **Step 3:** `go test ./cmd/ -race -run 'HandoffDecisions|DecisionAnswer|TeamCheckin|SealProbe|Swarm|Colonize|FastPath|Keeper|Dispatchability|OpenDecision' -count=1` → green.
- [ ] **Step 4:** Update `CLAUDE.md`: add a short "Team Check-In and Owner Decisions" subsection under Queen-Owned Orchestration naming the new commands, the always-pause default, `--no-checkin`, and the enforcing test names (every claim testable). Update the Quality Gates/continue notes if they now misstate reviewer selection.
- [ ] **Step 5:** Append CHANGELOG entry (follow the existing format at the top of `CHANGELOG.md`).
- [ ] **Step 6: Commit** `docs: document team check-in and owner decision routing`

## Self-Review Notes

- Spec coverage: A → Tasks 10-11; B → Tasks 7-9, 12-13; C1-C5 → Tasks 1-6; DoD tests named per task; no-mutate tests in Tasks 7 and 10. Spec's "record trim as observation" → Task 11 step 4 (`memory-capture`). Spec's Codex note needs no task: Codex never runs these wrappers and the new ceremony verb is print-only there by construction.
- Deliberate deviations from spec text: none. The spec's open choice (which verb records a trim) is resolved to `memory-capture`.
- Type consistency: `pendingHandoffDecisions` (Task 7) is reused by Task 12's wrapper via the CLI, not Go; `prompt_section` key name is identical in Tasks 8 and 12.
