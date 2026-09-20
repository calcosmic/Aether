# Phase 198: Put the Thrown-Away Data Back on Screen - Pattern Map

**Mapped:** 2026-08-29
**Files analyzed:** 10 (all modified, no wholly new files except test files and possibly a small testdata allowlist)
**Analogs found:** 10 / 10 (this phase is pure extension of existing patterns — RESEARCH.md already did most of the analog-finding; this file packages it for the planner with concrete excerpts)

## File Classification

| File to modify | Role | Data Flow | Closest Analog (same file, existing sibling pattern) | Match Quality |
|---|---|---|---|---|
| `cmd/codex_continue.go` (`runVerificationStep`) | service (verification step runner) | request-response / event-driven (emits progress) | `cmd/codex_build_progress.go` (`emitCodexDispatchWorkerFinished`) — existing start/finish emit for workers | role-match (worker progress → check progress) |
| `cmd/codex_continue_finalize.go` (`codexContinueWorkerFlowStep`) | model (result struct) | transform (struct → JSON) | `cmd/codex_build_finalize.go` (`codexExternalBuildWorkerResult`, lines 62-113) — same Duration+ToolCount fields already exist here | exact (field-for-field template) |
| `cmd/codex_build_progress.go` (`emitCodexDispatchWorkerFinished`) | service (progress emitter) | event-driven | itself, lines 143-180 — already renders Duration, just needs ToolCount added | exact (self-extend) |
| `cmd/ceremony_cmd.go` (`renderCeremonyCloseoutVisual`) | component/renderer | transform | `cmd/codex_visuals.go` (`renderPlanVisual`/`renderContinueVisual`/`renderSealVisual`) — the direct-path renderers this function should delegate to | exact (replacement target identified by research) |
| `cmd/closeout_cmd.go` (`closeoutCompletionDetails`, `closeoutWorkerMaps`) | transform (JSON reshaping) | transform | `cmd/finality_parity_test.go` fixture builders — round-trip pattern to preserve | role-match |
| `cmd/codex_workflow_cmds.go` (`sealCmd`, `completeSealRuntime`) | controller (CLI command) | request-response with a new pause point | `cmd/ceremony_team_checkin.go` (`decideBuildCheckin`) — existing pause/gate decision pattern to mirror for seal's new confirmation | role-match (build check-in → seal confirmation) |
| `cmd/context.go` (`buildResumeDashboardResult`) | service (dashboard data builder) | CRUD (read state, assemble view model) | itself — `result["recent"]` already built at lines 276-279, needs a per-phase progress loop added the same way | exact (self-extend) |
| `cmd/codex_visuals.go` (`renderResumeVisual`) | component/renderer | transform | `renderContinueWorkerFlowValue` (lines 2408-2435) — dual-type rendering helper pattern | exact (pattern to copy for any new field) |
| `cmd/ceremony_team_checkin.go` (`buildHasPendingOwnerDecision`) | service (predicate) | request-response | itself — extend with a third predicate reading `continue.json`'s `Advanced` field | exact (self-extend) |
| `.claude/commands/ant/{seal,continue}.md` + `.opencode` mirrors | config/wrapper markdown | request-response (prompt text) | `.claude/commands/ant/continue.md` / `.opencode/commands/ant/continue.md` (byte-identical pair) — the triplet-edit + parity-test pattern | exact |
| New test file(s) for SHOW-03/04/08 | test | — | `cmd/deterministic_floor_test.go` (`TestBothContinueLanesApplyTheSameFloor`), `cmd/ceremony_team_checkin_test.go` (`TestBuildCheckinDecisionMatrix`) | exact |
| New allowlist for `TestRenderedVisualsShowEveryCarriedField` | config/testdata | — | `cmd/display_house_style_test.go` (in-test Go map literal pattern, `TestHumanDisplaysUseHeadedSectionsNotMachineTables`) | exact (research recommends this over the JSON+baseline pattern) |

## Pattern Assignments

### `cmd/codex_continue.go` — `runVerificationStep` (SHOW-03 live check lines)

**Analog:** `cmd/codex_build_progress.go` `emitCodexDispatchWorkerFinished` (lines 143-180) — the existing start/finish emit for reviewer workers, and `cmd/codex_visuals.go`'s `emitVisualProgress`.

**Streaming exit point** (`cmd/codex_visuals.go:424-433`):
```go
func emitVisualProgress(visual string) {
	if !shouldRenderVisualOutput(stdout) || !streamingAllowedForCurrentCommand() {
		return
	}
	visual = strings.TrimSpace(visual)
	writeVisualOutput(stdout, visual+"\n\n")
}
```
Use this exact function for both the "Running tests…" start line and the "Tests ✓ 12/12 (4s)" / "Tests ✗ 2 of 12 failed — …" finish line. Do not `fmt.Fprintln(stdout, ...)` directly — `writeVisualOutput` is the single point that performs command-name translation, so bypassing it risks leaking raw `aether` command names into chat surfaces.

**Existing duration render for workers to mirror** (`cmd/codex_build_progress.go:170-172`, read this session per RESEARCH.md Q3):
```go
// existing pattern: finish line already renders %.1fs from result.WorkerResult.Duration
fmt.Sprintf("%s done (%.1fs, ...)", label, result.WorkerResult.Duration.Seconds())
```
Add `ToolCount` alongside it the same way (D-03) — the field already exists on `pkg/codex/worker.go:80` (`WorkerResult.ToolCount`), it is simply never read at this call site.

**Shared body to hook into** (`cmd/deterministic_floor.go:59-66`, both continue lanes call this — placing the emit here structurally guarantees lane parity):
```go
steps := []codexVerificationStep{
    runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, verificationTimeout),
    runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, verificationTimeout),
    runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, verificationTimeout),
    runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, verificationTimeout),
}
```
`codexVerificationStep` (cmd/codex_continue.go:33-46) has no `Duration` field today — add one (`time.Since` around the shell-out), matching how `codexContinueWorkerFlowStep.Duration` is already populated at `cmd/codex_continue.go:1580`.

**Error handling pattern for the failed-check inline reason (D-02):** `codexVerificationStep.Summary` already carries a plain-English one-liner (struct field, cmd/codex_continue.go:33-46) — reuse it verbatim for the failed finish line rather than inventing a second summary format.

---

### `cmd/codex_continue_finalize.go` — `codexContinueWorkerFlowStep` (SHOW-02/D-03 tool count)

**Analog:** `cmd/codex_build_finalize.go:62-113` (`codexExternalBuildWorkerResult`) — the sibling struct that already has both fields:
```go
// Source: cmd/codex_build_finalize.go:109-110 (read this session)
Duration  float64 `json:"duration,omitempty"`
ToolCount int     `json:"tool_count,omitempty"`
```

**Where to add the mirrored field and its population** (`cmd/codex_continue.go:1568-1585, 1580`):
```go
// existing:
step.Duration = result.WorkerResult.Duration.Seconds()
// add analogously:
step.ToolCount = result.WorkerResult.ToolCount
```
Note `Usage codex.WorkerUsage` on the same struct is deliberately `json:"-"` (never serialized, by design — "a figure that crossed a wire could be asserted by an outside caller rather than measured by the runtime"). `ToolCount` is a plain `int` sourced from the same trusted in-process `WorkerResult`, so it does not carry that same risk — do not apply the `json:"-"` treatment to it.

---

### `cmd/ceremony_cmd.go` — `renderCeremonyCloseoutVisual` (SHOW-01/D-12 structural fix)

**Analog / target to delegate to:** `cmd/codex_visuals.go` `renderPlanVisual` (starts line 1399), `renderContinueVisual`, `renderSealVisual` — the direct-path renderers.

**The exact discard this fixes** (`cmd/codex_visuals.go:362-371`):
```go
func outputWorkflow(result interface{}, visual string) {
	if shouldRenderVisualOutput(stdout) {
		if !strings.HasSuffix(visual, "\n") {
			visual += "\n"
		}
		writeVisualOutput(stdout, visual)
		return
	}
	outputOK(result) // <-- `visual` (the rich, already-computed render) is discarded entirely here
}
```

**The narrow re-derivation to remove/replace** (`cmd/closeout_cmd.go:222-240`):
```go
func closeoutWorkerMaps(raw map[string]interface{}) []map[string]interface{} {
	workers := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, key := range []string{"dispatches", "results", "workers"} { // <-- never "worker_flow"
		for _, worker := range mapSliceValue(raw[key]) {
			...
		}
	}
	return workers
}
```

**Dual-type helper to copy** (`cmd/codex_visuals.go:2408-2435`) — this is the load-bearing precedent that makes routing closeout through the direct renderer safe, because a completion file has already round-tripped through JSON (structs → `map[string]interface{}`, slices of structs → `[]interface{}`):
```go
func renderContinueWorkerFlowValue(b *strings.Builder, raw interface{}) {
	switch flow := raw.(type) {
	case []codexContinueWorkerFlowStep:
		// direct in-process path: full render
	case []interface{}:
		renderContinueWorkerFlowMap(b, flow) // JSON-round-tripped path
	}
}
```
Any new field added to a `render*Visual` function this phase touches must follow this same type-switch shape if it needs to work from both the live struct and the completion-file map.

---

### `cmd/codex_workflow_cmds.go` — seal confirmation gate (SHOW-04, D-04..D-07)

**Analog for the pause/gate decision itself:** `cmd/ceremony_team_checkin.go` `decideBuildCheckin` (lines 406-444) — fixed-order decision function pattern to mirror for seal's new stop-and-ask point:
```go
// pattern: evaluate signals in fixed priority order, return a decision struct
// naming why it paused/didn't pause — never inferred from rendered text
func decideBuildCheckin(input buildCheckinDecisionInput) buildCheckinDecision {
	if input.Autopilot || input.NoCheckin { /* non-interactive */ }
	if input.ExplicitCheckin { /* force pause */ }
	if input.PendingOwnerDecision { /* force pause */ }
	// ...
}
```

**Analog for recording the owner's answer (D-06 "finish anyway"):** `cmd/forced_reviewer_waiver.go:478-500` — the `PendingDecision`-with-`Source`-tag pattern, never inferred from card wording:
```go
// Source: cmd/forced_reviewer_waiver.go:478-497 (read this session)
decision := PendingDecision{
	ID:          fmt.Sprintf("pd_%d", time.Now().UnixNano()),
	Type:        clarificationDecisionType,
	Description: formatClarificationDescription(question, nil),
	Source:      "forced-reviewer-waiver", // for D-06: use "seal-force-confirmation"
	Resolved:    false,
	CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	AttemptID:   attemptID,
}
if phaseID > 0 {
	decision.Phase = &phaseID
}
```
Recorded/consumed at the CLI boundary via `decisionAnswerCmd` (`cmd/handoff_decisions_cmd.go:165-246`) — reuse this same command surface rather than adding a bespoke seal-only flag.

**Ordering note (do not reorder `runSealConsolidation`):** `completeSealRuntime` (cmd/codex_workflow_cmds.go:558-786) already calls `runSealConsolidation()` at line 649, well before the state mutation at line 688 (`state.State = colony.StateCOMPLETED`). D-04/D-05 require **inserting a new stop-and-ask point**, not reordering the wisdom review. Per RESEARCH.md Open Question 2, the card (D-04) needs data available even before consolidation runs (from `checkSealBlockers`, already computed earlier in the function) — plan the confirmation to straddle the existing consolidation call: card → wisdom review → confirmation question → mutation.

**D-07 test analog:** mirror `TestBuildCheckinDecisionMatrix` (cmd/ceremony_team_checkin_test.go:370, table-driven) for a new `TestSealAutopilotNeverReachesSealPath` — assert on the actual dispatch/command path autopilot takes, not an intermediate record (per `TestQueenChoiceReachesTheDispatchList`'s precedent in this codebase of never trusting an intermediate record).

---

### `cmd/context.go` — `buildResumeDashboardResult` (SHOW-02 resume detail)

**Analog:** itself — `result["recent"]` is already built and simply never read by the renderer:
```go
// Source: cmd/context.go:276-279 (read this session)
result["recent"] = map[string]interface{}{
	"decisions": recentDecisions, // extractRecentDecisions(state.Memory.Decisions, 5)
	"events":    recentEvents,    // extractRecentEvents(state.Events, 10)
}
```
`colony.Decision` struct (pkg/colony/colony.go:777-783): `ID, Phase int, Claim, Rationale, Timestamp string` — render these fields directly, one line per decision.

**New per-phase progress loop (genuinely new, no existing analog beyond the overall fraction at cmd/context.go:262-269):** iterate `state.Plan.Phases[].Status` (`colony.PhasePending`/`PhaseInProgress`/`PhaseCompleted`, pkg/colony/colony.go:30-33) and build a parallel per-phase list next to the existing `phase: N/M` fraction.

**Test analog for capturing stdout** (`cmd/context_test.go:56-91`, `TestResumeDashboard`):
```go
func saveGlobalsCmd(t *testing.T) {
	t.Helper()
	origStdout, origStderr, origStore := stdout, stderr, store
	t.Cleanup(func() { stdout, stderr, store = origStdout, origStderr, origStore })
}

func TestResumeDashboard(t *testing.T) {
	saveGlobalsCmd(t)
	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf
	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)
	store = s
	// ... build colony.ColonyState fixture, call the command function, assert on buf.String()
}
```
Use this exact `saveGlobalsCmd` + buffer-swap pattern for any new test asserting on `emitVisualProgress`/`writeVisualOutput` output (SHOW-03's live check lines, SHOW-04's seal card).

---

### `cmd/ceremony_team_checkin.go` — `buildHasPendingOwnerDecision` (D-08/D-09 blocker advisory)

**Analog:** itself, lines 465-478 — extend with a third predicate:
```go
// existing two predicates (cmd/ceremony_team_checkin.go:465-478):
func buildHasPendingOwnerDecision(manifest codexBuildManifest) (bool, string) {
	if hit, reason := riskSignalHitsFromRecords(...); hit { return true, reason }
	if manifest.BoundaryQuestionCount > 0 { return true, "an unanswered orchestrator boundary question" }
	if pendingHandoffDecisions(manifest.Phase) { return true, "a worker's unanswered handoff question" }
	return false, ""
}
```
Add a third check reading the structured `codexContinueReport.Advanced bool` field (cmd/codex_continue.go:107-127, `json:"advanced"`) from the phase's `continue.json` artifact — **do not** parse the pipe-delimited event string `"{ts}|continue_blocked|..."` (cmd/codex_continue.go:4091); RESEARCH.md explicitly flags that as fragile and untested.

**Non-interactive branch still must print (new behavior):** `decideBuildCheckin`'s existing `Autopilot || NoCheckin` branch (line 407-413) sets `Requested: false` — D-09 requires the heads-up line to still print unconditionally even here; only the *asking* is gated by `Requested`.

---

### `.claude/commands/ant/seal.md` / `continue.md` + `.opencode` mirrors (triplet edit + parity test)

**Analog — full triplet-edit example already in the codebase:** `cmd/continue_wrapper_ceremony_test.go` (`TestContinueWrapperCeremonyContract`), the exact pattern to copy for any new required substring added to the seal/continue wrapper prose (e.g. the new seal confirmation prompt text, D-04's card, D-08's blocker question):
```go
func TestContinueWrapperCeremonyContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "continue.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "continue.md"),
	}
	required := []string{
		"AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS",
		"AETHER_OUTPUT_MODE=json aether continue-finalize --completion-file",
		"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file",
		// ... every required substring is checked in BOTH paths via the loop below
	}
	for _, path := range wrapperPaths {
		data, err := os.ReadFile(path)
		content := string(data)
		for _, r := range required {
			if !strings.Contains(content, r) {
				t.Errorf("%s missing required text: %q", path, r)
			}
		}
	}
}
```
**Rule:** any new line added to `.claude/commands/ant/seal.md` for D-04/D-06's confirmation card or D-08's blocker question must be added byte-identically to `.opencode/commands/ant/seal.md` in the same plan/commit, and the sibling `Test{Seal,Continue}WrapperCeremonyContract` (cmd/seal_wrapper_ceremony_test.go, cmd/continue_wrapper_ceremony_test.go) extended with the new required substring so drift is caught structurally, not by memory.

Also relevant: `cmd/lifecycle_wrapper_contract_test.go:135` (`TestLifecycleFlatMirrorsMatchCanonical`) checks a **third**, install-produced flat-mirror copy — no manual edit needed there (it's derived), but don't forget it exists if a new flat mirror fixture needs regenerating during verification.

---

### `TestRenderedVisualsShowEveryCarriedField` — shrink-only allowlist (SHOW-02/D-12 Claude's Discretion)

**Analog — recommended pattern (in-test Go map literal, not the JSON+baseline file):** `cmd/display_house_style_test.go:62-94` (`TestHumanDisplaysUseHeadedSectionsNotMachineTables`):
```go
func TestHumanDisplaysUseHeadedSectionsNotMachineTables(t *testing.T) {
	allowed := map[string]string{
		"queen_wave_lifecycle.go": "six-column numeric wave dispatch counts; a genuine table of numbers",
		"audit_catalog.go":        "generates the markdown command catalog document, not terminal display",
		// one entry per exception, with a written reason — shrink-only by convention/review
	}
	files, _ := filepath.Glob("*.go")
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") { continue }
		if _, ok := allowed[filepath.Base(file)]; ok { continue }
		// ... assert the invariant against every non-allowlisted file
	}
}
```
For `TestRenderedVisualsShowEveryCarriedField`, key the map by **result-map key name** (e.g. `"dispatch_mode"`, `"artifact_source"`) rather than filename, with the reason each key is intentionally provenance-only and not owner-facing. Reserve the heavier JSON+frozen-baseline pattern (`cmd/testdata/orphan_allowlist.json` + `orphan_allowlist_baseline.json`, asserted by `TestOrphanAllowlistOnlyShrinks`, cmd/subcommand_reachability_ratchet_test.go:1496-1520) only if the exception count grows past what's comfortable to review inline — RESEARCH.md judges that unlikely here (a handful of keys per finalizer).

---

### Fixture-construction pattern for SHOW-01/SHOW-02 tests (Pitfall 4 in RESEARCH.md)

**Analog:** `cmd/finality_parity_test.go:20-43` (`parityContinueManifest`, functional-options builder) — the CLAUDE.md "derive fixture values the way the runtime derives them" rule applied to this exact problem. For `TestWrapperPathRendersSameCeremonyAsDirectPath`, do **not** hand-type a `map[string]interface{}` with nested structs — always:
```go
// 1. Build the real typed result via the finalizer function (or a fixture matching its shape)
result := runCodexContinueFinalizeResult(...) // typed structs, slices of structs

// 2. Round-trip it the way a completion file actually is
data, _ := json.Marshal(result)
var asMap map[string]interface{}
json.Unmarshal(data, &asMap)

// 3. Render both and diff
directOutput := renderContinueVisual(result)          // typed path
closeoutOutput := renderCeremonyCloseoutVisual(asMap)  // JSON-round-tripped path (post-fix)
```

## Shared Patterns

### Output streaming (applies to every new line: SHOW-03 live checks, D-03 reviewer finish lines, D-08 blocker question, D-04 seal card)
**Source:** `cmd/codex_visuals.go:424-433` (`emitVisualProgress`), `cmd/codex_visuals.go:447+` (`writeVisualOutput`)
```go
func emitVisualProgress(visual string) {
	if !shouldRenderVisualOutput(stdout) || !streamingAllowedForCurrentCommand() {
		return
	}
	writeVisualOutput(stdout, strings.TrimSpace(visual)+"\n\n")
}
```
**Apply to:** any new emit call this phase adds. Never call `fmt.Fprintln(stdout, ...)` directly for owner-facing text — `writeVisualOutput` is the single point performing command-name translation (`/ant-*` chokepoint per project memory `project_command_naming_chokepoint`).

### Owner-decision recording (applies to D-06 seal override, D-09 blocker heads-up answer)
**Source:** `cmd/forced_reviewer_waiver.go:478-500` (`PendingDecision` with a `Source` tag), consumed via `cmd/handoff_decisions_cmd.go:165-246` (`decisionAnswerCmd`)
**Apply to:** any new "the owner answered X" fact. Never infer an answer from rendered card text — always a named, queryable `PendingDecision` record with a distinct `Source` string (e.g. `"seal-force-confirmation"`), matching `TestOnlyTheOwnerCanWaiveAForcedReviewer`'s precedent (cmd/forced_reviewer_waiver_test.go:74).

### Dual-type rendering (applies to any render function touched by the SHOW-01 structural fix)
**Source:** `cmd/codex_visuals.go:2408-2435` (`renderContinueWorkerFlowValue`)
**Apply to:** every `render*Visual`/`render*Value` function this phase extends or routes closeout through — must type-switch between the live Go struct and the JSON-round-tripped `map[string]interface{}`/`[]interface{}` shape.

### Test stdout capture
**Source:** `cmd/context_test.go:55-79` (`saveGlobalsCmd` + buffer swap), `cmd/deterministic_floor_test.go:143-178` (`TestBothContinueLanesApplyTheSameFloor`, table-driven dual-lane comparison)
**Apply to:** all new tests for SHOW-03 (live lines), SHOW-04 (seal card/confirmation), D-08 (blocker advisory) — swap `stdout`/`stderr`/`store` via `t.Cleanup`, read `buf.String()`.

### Wrapper triplet parity
**Source:** `cmd/continue_wrapper_ceremony_test.go` (`TestContinueWrapperCeremonyContract`), `cmd/seal_wrapper_ceremony_test.go`, `cmd/plan_wrapper_ceremony_test.go`
**Apply to:** every edit to `.claude/commands/ant/{seal,continue}.md` — the `.opencode` mirror must change identically in the same plan, and the parity test's `required` slice must gain the new substring.

## No Analog Found

| Item | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Seal state-of-play card content assembly (D-04: phases done/total + failing checks + open warnings + "what finishing will do") | component/renderer | transform | Genuinely new composite view; nearest analog (`decideBuildCheckin`'s decision struct) covers the gating logic but not this specific card's content — the planner should compose it from already-computed `checkSealBlockers`/`scanHighSeverityOpen` outputs (RESEARCH.md Q7/Open Question 2), not from a prior card template |
| "Drift note" on resume (Claude's Discretion) | transform | No existing computation anywhere in the codebase (zero-match grep, confirmed by RESEARCH.md Q6). RESEARCH.md recommends deriving it from `planRevisionSummary(state.Plan)` (already computed at cmd/context.go:290, itself carried-but-unsurfaced) rather than inventing a new algorithm |
| `codexVerificationStep.Duration` field | model | — | Field does not exist today (confirmed zero matches); add by analogy to `codexContinueWorkerFlowStep.Duration`'s existing population pattern (cmd/codex_continue.go:1580), not copied from an identical prior field |

## Metadata

**Analog search scope:** `cmd/` package only (per RESEARCH.md — no `pkg/` rendering code, no external libraries). RESEARCH.md's own Sources section already performed exhaustive targeted reads of every file in this phase's touch list; this pass re-verified the specific excerpts needed for planner-ready code snippets (allowlist tests, waiver/decision recording, wrapper triplet parity, stdout-capture test harness) that RESEARCH.md referenced but did not quote in full.
**Files scanned this session:** cmd/subcommand_reachability_ratchet_test.go, cmd/display_house_style_test.go, cmd/forced_reviewer_waiver.go, cmd/forced_reviewer_waiver_test.go, cmd/handoff_decisions_cmd.go, cmd/continue_wrapper_ceremony_test.go, cmd/deterministic_floor_test.go, cmd/context_test.go, cmd/ceremony_team_checkin_test.go (plus everything RESEARCH.md already cites by file:line).
**Pattern extraction date:** 2026-08-29
