# Phase 200: Iterative Planning - Pattern Map

**Mapped:** 2026-09-07
**Scope source:** `200-CONTEXT.md` and `200-RESEARCH.md`
**Files/families classified:** 14
**Primary analogue families:** 5
**Analogue coverage:** 14 / 14 (11 exact/self analogues, 3 composite role matches)

Phase 200 is mostly an extension of existing planning, revision, discussion, rendering, and contract-test machinery. The planner should preserve those seams rather than introduce a second planning subsystem. Paths below are the concrete files named or implied by the phase context and research; helper-file splits remain implementation discretion.

## File Classification

| New/Modified File or Family | Role | Data Flow | Closest Current Analogue | Match Quality |
|---|---|---|---|---|
| `pkg/colony/colony.go` and/or new `pkg/colony/spec.go` | model | file-I/O, transform | `pkg/colony/colony.go:294-319`, `421-489` | exact role / composite domain |
| `cmd/codex_plan.go` | controller, provider | request-response, event-driven | its current manifest and iteration engine at `cmd/codex_plan.go:192-298`, `1024-1257`, `1978-2154` | exact |
| `cmd/codex_plan_finalize.go` | service | event-driven, file-I/O, transform | its validation/finalization pipeline at `cmd/codex_plan_finalize.go:299-545`, `693-1004` | exact |
| `cmd/plan_revision.go` | service, utility | transform, file-I/O | its immutable plan-revision activation at `cmd/plan_revision.go:86-357` | exact |
| `cmd/discuss.go` and pending-decision helpers | controller, service | request-response, file-I/O | scoped question intake/resolution at `cmd/discuss.go:139-493`, `800-889` | exact |
| new `cmd/spec*.go` | controller, service | request-response, file-I/O, transform | composite of `cmd/discuss.go:139-493` and `cmd/plan_revision.go:250-357` | role match |
| `cmd/state_extra.go` | controller | request-response, file-I/O | atomic phase insertion at `cmd/state_extra.go:356-519` | exact; mutation target changes |
| `cmd/state_load.go` | migration, utility | file-I/O, transform | legacy normalization at `cmd/state_load.go:18-200` | exact |
| `cmd/codex_visuals.go` | component | transform | plan renderer and common lifecycle primitives at `cmd/codex_visuals.go:497-607`, `1416-1756` | exact |
| `.aether/commands/{plan,discuss,spec}.yaml` | config | request-response | `.aether/commands/plan.yaml:1-81`, `.aether/commands/discuss.yaml:1-25` | exact for plan/discuss; role match for spec |
| `.claude/commands/ant/{plan,discuss,spec}.md` and `.opencode/commands/ant/{plan,discuss,spec}.md` | component, config | request-response | existing managed plan/discuss wrapper pairs | exact for plan/discuss; role match for spec |
| `cmd/command_guide.go`, `.aether/skills/colony/aether-colony-build-cycle/SKILL.md`, `.aether/skills/colony/aether-colony-research/SKILL.md`, relevant distributed docs | provider, config | request-response | existing plan/discuss guide and skill entries | exact |
| `cmd/testdata/classic-contract/v1/{schema.json,mechanisms.json,cases.json}` and command inventory fixtures | test fixture, config | batch, transform | current strict Classic contract corpus | exact |
| co-located `cmd/*_test.go` and `pkg/colony/*_test.go` additions | test | request-response, file-I/O, transform | `cmd/plan_revision_test.go`, `cmd/classic_contract_test.go`, `cmd/plan_wrapper_cards_test.go` | exact |

## Pattern Assignments

### 1. Specification, candidate, revision, and migration domain

**Applies to:** `pkg/colony/colony.go` and/or `pkg/colony/spec.go`, `cmd/plan_revision.go`, `cmd/state_load.go`

**Primary analogues:** the accepted-charter binding and immutable plan-revision model in `pkg/colony/colony.go`, plus the content-addressed revision constructor in `cmd/plan_revision.go`.

**Versioned accepted-contract pattern** (`pkg/colony/colony.go:294-319`):

```go
const AcceptedCharterSchemaVersion = "accepted-charter-v1"

type AcceptedCharter struct {
	SchemaVersion    string   `json:"schema_version"`
	CharterID        string   `json:"charter_id"`
	ContentHash      string   `json:"content_hash"`
	AcceptedAt       string   `json:"accepted_at"`
	SourceQuestionID string   `json:"source_question_id"`
	RevisionReason   string   `json:"revision_reason,omitempty"`
	RevisionEvidence []string `json:"revision_evidence,omitempty"`
}
```

Reuse this for a versioned specification binding and acceptance receipt: explicit schema version, stable ID, content hash, timestamp, and exact source/acceptance token. Do not infer approval from a file existing or from a planning command completing.

**Immutable predecessor-linked history pattern** (`pkg/colony/colony.go:459-489`):

```go
type PlanRevision struct {
	ID                 string              `json:"id"`
	ParentRevisionID   string              `json:"parent_revision_id,omitempty"`
	CreatedAt          string              `json:"created_at"`
	Reason             PlanRevisionReason  `json:"reason"`
	Evidence           []string            `json:"evidence,omitempty"`
	EvidenceHash       string              `json:"evidence_hash,omitempty"`
	BaseDefinitionHash string              `json:"base_definition_hash,omitempty"`
	PlanDefinitionHash string              `json:"plan_definition_hash"`
	Phases             []Phase             `json:"phases"`
	PhaseIDMappings    map[string]string   `json:"phase_id_mappings,omitempty"`
}

const (
	PlanEvidencePolicyLegacyUnbound PlanEvidencePolicy = "legacy_unbound"
	PlanEvidencePolicyBoundV1       PlanEvidencePolicy = "bound_v1"
)
```

The new specification lineage should follow the same append-only shape: immutable revision snapshots, a predecessor ID, content hash, reason, requirement delta, and explicit legacy binding state. Plan candidates should bind to the approved specification revision and to the planning run that produced them.

**Hash-derived revision identity pattern** (`cmd/plan_revision.go:334-357`):

```go
func newPlanRevision(parentID string, reason colony.PlanRevisionReason, evidence []string,
	evidenceHash, baseHash, runID string, phases []colony.Phase,
	mappings map[string]string, now time.Time) (colony.PlanRevision, error) {
	definitionHash, err := stablePlanDefinitionHash(phases)
	if err != nil {
		return colony.PlanRevision{}, err
	}
	revisionID := shortContentID("plan-rev", definitionHash+"\x00"+parentID+"\x00"+string(reason))
	return colony.PlanRevision{
		ID: revisionID, ParentRevisionID: parentID, CreatedAt: now.UTC().Format(time.RFC3339),
		Reason: reason, Evidence: append([]string(nil), evidence...), EvidenceHash: evidenceHash,
		BaseDefinitionHash: baseHash, PlanningRunID: runID,
		PlanDefinitionHash: definitionHash, Phases: clonePhases(phases),
		PhaseIDMappings: cloneStringMap(mappings),
	}, nil
}
```

Reuse the constructor discipline for `SpecRevision` and `PlanCandidate`: compute identity from semantic content and lineage inputs, copy slices/maps, and never retain mutable aliases. The semantic plan delta can be stored beside the candidate/revision, but activation should still use stable definition hashes rather than lifecycle status fields.

**Compatibility normalization pattern** (`cmd/state_load.go:146-200`):

```go
func normalizeLegacyColonyState(state *colony.ColonyState) bool {
	changed := false
	if state.PlanEvidencePolicy == "" {
		if len(state.PlanRevisions) == 0 {
			state.PlanEvidencePolicy = colony.PlanEvidencePolicyLegacyUnbound
		} else {
			state.PlanEvidencePolicy = colony.PlanEvidencePolicyBoundV1
		}
		changed = true
	}
	// ...normalize old lifecycle aliases without inventing new facts...
	return changed
}
```

Additive fields should be `omitempty` where absence is meaningful. Decode old state as an explicit legacy/unbound condition, preserve existing buildability, and only require a current approved specification for new plans or material revisions. Unknown future major schema versions should fail closed; missing fields from old state should not be treated as owner approval.

### 2. Staged Scout/Route iteration and exact plan-candidate acceptance

**Applies to:** `cmd/codex_plan.go`, `cmd/codex_plan_finalize.go`, and any small extracted iteration/delta helper files

**Primary analogue:** the existing manifest-driven planning run and completion-packet finalizer.

**Persisted planning envelope pattern** (`cmd/codex_plan.go:218-279`):

```go
type codexPlanManifest struct {
	SchemaVersion     string                  `json:"schema_version"`
	RunID             string                  `json:"run_id"`
	Iteration         int                     `json:"iteration"`
	ConfidenceTarget  int                     `json:"confidence_target"`
	MaxIterations     int                     `json:"max_iterations"`
	PreviousEvidence []string                `json:"previous_evidence,omitempty"`
	SelectedGaps      []string                `json:"selected_gaps,omitempty"`
	PreviousDraft     []colony.Phase          `json:"previous_draft,omitempty"`
	ExpectedWorkers   []codexPlanDispatch     `json:"expected_workers"`
	Revision          *codexPlanRevisionContext `json:"revision,omitempty"`
	SurveySnapshot    *codexPlanSurveySnapshot `json:"survey_snapshot,omitempty"`
	ContextCapsule    *codexContextCapsule    `json:"context_capsule,omitempty"`
}
```

Extend this envelope instead of creating hidden in-memory state. Add the selected preset, stage/round identity, approved specification binding, prior readiness snapshot, weakest gap, expected worker identity, and any exact decision-resume token needed by the next transition. A manifest should authorize one expected stage, not the whole future loop.

**Run construction pattern** (`cmd/codex_plan.go:1098-1217`):

```go
runSeed, err := loadOrSeedCodexPlanIteration(state, opts, revisionContext)
if err != nil {
	return nil, err
}
runID := codexPlanRunID(state.Goal, runSeed.Iteration, revisionContext)
dispatches := buildCodexPlanDispatches(runID, runSeed.Iteration, researchWaves)
// ...assemble worker briefs, survey snapshot, contract, and context capsule...
manifest := codexPlanManifest{
	SchemaVersion: codexPlanManifestSchemaVersion,
	RunID: runID,
	Iteration: runSeed.Iteration,
	PreviousEvidence: append([]string(nil), runSeed.PreviousEvidence...),
	SelectedGaps: append([]string(nil), runSeed.SelectedGaps...),
	PreviousDraft: clonePhases(runSeed.PreviousDraft),
	ExpectedWorkers: dispatches,
}
```

The Scout pass, material-decision checkpoint, Route-Setter pass, and iteration receipt should be explicit transitions around this construction. Resume from a persisted checkpoint, not by replaying earlier completed worker stages.

**Validate fully before mutation, then revalidate under lock** (`cmd/codex_plan_finalize.go:452-495`):

```go
if err := writePlanningArtifacts(root, completion, provenance); err != nil {
	return nil, err
}
err = store.UpdateJSONAtomically(colonyStatePath(root), func(current *colony.ColonyState) error {
	if current.Goal != manifest.Goal {
		return fmt.Errorf("planning goal changed while finalizing")
	}
	if err := validatePlanRevisionBase(current, manifest.Revision); err != nil {
		return err
	}
	return activateGeneratedPlan(current, phases, manifest.Revision, manifest.RunID, now)
})
```

Keep the outer validation pipeline and the inside-lock state/base recheck, but change the terminal mutation: a completed planning run persists a non-active candidate and iteration timeline. Only the exact owner acceptance command/token may activate that candidate. The command should reject stale candidate ID, stale spec revision, stale plan base, replayed receipt, or a mismatched run/stage.

**Durable intermediate-pass pattern** (`cmd/codex_plan_finalize.go:932-1004`):

```go
func persistIntermediatePlanningIteration(root string, state *colony.ColonyState,
	manifest codexPlanManifest, completion codexPlanCompletionPacket,
	loop codexPlanLoopEvaluation) (map[string]any, error) {
	iterationDir := filepath.Join(root, ".aether", "data", "planning", manifest.RunID, "iterations")
	// write evidence and iteration artifacts, then save next iteration state
	if err := store.SaveJSON(colonyStatePath(root), state); err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "iteration_complete",
		"run_id": manifest.RunID,
		"next_command": "aether plan",
	}, nil
}
```

The ordered iteration-delta timeline belongs in durable state/artifacts using the existing run/iteration directory convention. Each receipt needs fresh evidence, before/after five-dimension readiness, causal evidence references per moved dimension, the weakest remaining gap, semantic added/changed/removed plan elements, owner-authority flags, and a closed stop reason.

**Current stop evaluator to evolve** (`cmd/codex_plan_finalize.go:784-864`):

```go
func evaluateCodexPlanLoop(manifest codexPlanManifest, completion codexPlanCompletionPacket) codexPlanLoopEvaluation {
	if completion.Accepted {
		return codexPlanLoopEvaluation{Stop: true, Reason: "accepted"}
	}
	if completion.Confidence.Overall >= manifest.ConfidenceTarget {
		return codexPlanLoopEvaluation{Stop: true, Reason: "target_reached"}
	}
	if manifest.Iteration >= manifest.MaxIterations {
		return codexPlanLoopEvaluation{Stop: true, Reason: "iteration_cap"}
	}
	// ...stall/current-gap handling...
}
```

Replace the generic `accepted` shortcut with the locked taxonomy: `target_reached`, `diminishing_returns`, `unresolved_material_stall`, and `iteration_cap`. Below-target closure is valid only for non-material remaining uncertainty; material uncertainty pauses for an owner decision. Keep the owner-selected preset independent from achieved readiness.

### 3. Evidence-first material decisions, specification command, and living-plan changes

**Applies to:** `cmd/discuss.go`, new `cmd/spec*.go`, pending-decision helpers, `cmd/state_extra.go`

**Primary analogues:** scoped/deduplicated discussion entries and plan-revision activation.

**Grounded, typed, deduplicated intake pattern** (`cmd/discuss.go:139-230`):

```go
scope, err := currentDiscussIntentScope(root, state)
if err != nil {
	return nil, err
}
sourceHash := discussSourceHash(question.Source)
entry := colony.PendingQuestion{
	ID: question.ID, Category: question.Category,
	Question: question.Question, Why: question.Why,
	Options: append([]string(nil), question.Options...),
	SourceHash: sourceHash, GoalHash: scope.GoalHash,
	PreferenceSessionID: scope.PreferenceSessionID,
}
if pendingQuestionExists(state.PendingQuestions, entry) {
	return map[string]any{"status": "duplicate", "question_id": entry.ID}, nil
}
state.PendingQuestions = append(state.PendingQuestions, entry)
```

Reuse the stable source hash, goal hash, preference-session scope, typed category, and duplicate suppression for planning decision cards. Extend the payload to include evidence-backed options, recommendation, consequences, materiality, affected requirements/phases, and the exact resume token. Do not reuse the current generic menu-generation text at `cmd/discuss.go:495-699`; Phase 200 replaces that behavior with consolidated evidence-first questions derived after the first Scout pass.

**Exact resolution and stale-scope rejection pattern** (`cmd/discuss.go:350-438`):

```go
if question.ID != questionID {
	continue
}
if !pendingQuestionInScope(question, scope) {
	return nil, fmt.Errorf("question %q is stale for the active goal or preference session", questionID)
}
question.Answer = strings.TrimSpace(answer)
question.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
if err := store.SaveJSON(colonyStatePath(root), state); err != nil {
	return nil, err
}
```

Apply the same semantics to specification approval, feature-revision approval, material planning answers, and plan-candidate acceptance: exact ID/token, current scope, single use, persisted audit data, and stale/replay failure. A prior answer may be reused only after explicitly checking that its scope and evidence are still applicable.

**Current living-plan mutation seam** (`cmd/state_extra.go:394-503`):

```go
err = store.UpdateJSONAtomically(path, func(state *colony.ColonyState) error {
	before := clonePhases(state.Plan.Phases)
	if err := validatePhaseInsertRequest(state, request); err != nil {
		return err
	}
	// current implementation inserts and renumbers directly here
	state.Plan.Phases = insertPhase(state.Plan.Phases, request)
	state.Plan.Phases = renumberPhaseOrdinals(state.Plan.Phases)
	return persistPhaseInsertRecovery(state, before, request)
})
```

Preserve the atomic read/validate/recheck boundary, but route accepted insertions and revisions through the same spec-revision → impact closure → candidate → exact acceptance → immutable plan-revision machinery. Unaffected requirement IDs and plan content stay byte/semantically stable; only affected checks, dependencies, recovery notes, and phases may be invalidated or replaced.

**New `spec` command pattern:** use the same Cobra command shape as `discuss` (`cmd/discuss.go:59-129`), return structured result fields for renderers, and keep domain operations in testable helpers. `/ant-spec` must be a real public command on Claude/OpenCode, while Codex guidance must accurately translate it to the native `aether spec` command rather than claiming Codex slash-command support.

### 4. Runtime rendering and canonical cross-platform projection

**Applies to:** `cmd/codex_visuals.go`, canonical YAML, managed Claude/OpenCode wrappers, `cmd/command_guide.go`, shipped Codex skills, and related distributed docs

**Primary analogues:** the existing plan renderer, canonical plan/discuss YAML, managed wrapper pairs, and parity tests.

**Shared visual vocabulary** (`cmd/codex_visuals.go:497-564`):

```go
func renderBanner(command string) {
	fmt.Println(renderAetherWordmark(command))
}

func renderStageMarker(name string) {
	fmt.Printf("\n%s\n", styleMuted("── "+name+" ──"))
}

func renderNextUp(command, explanation string) {
	if strings.TrimSpace(command) == "" {
		return
	}
	fmt.Printf("\n%s %s\n", styleNextUp("Next up:"), translateCommandForPlatform(command))
	if strings.TrimSpace(explanation) != "" {
		fmt.Println(styleMuted(explanation))
	}
}
```

Use these primitives for the delta card, decision pause, draft-spec/approval state, candidate summary, and next action. Keep no-color output meaningful. Do not put authority logic in the renderer.

**Typed-field renderer branching** (`cmd/codex_visuals.go:1416-1467`):

```go
func renderPlanVisual(payload map[string]any) {
	renderBanner("plan")
	status := stringValue(payload, "status")
	if boolValue(payload, "repair_required") {
		renderPlanRepair(payload)
		return
	}
	if boolValue(payload, "iteration_complete") {
		renderPlanIteration(payload)
		return
	}
	// ...manifest/final plan branches using structured fields...
}
```

Extend the structured result contract first, then render explicit status/reason fields. The renderer may format scores and deltas but must not decide whether evidence is applicable, a decision is material, a spec is approved, or a candidate is accepted.

**Canonical command-source pattern** (`.aether/commands/plan.yaml:1-18`):

```yaml
name: plan
description: Plan implementation phases for the active colony
runtime:
  command: aether plan
  output_mode: visual
metadata:
  codex_skill: aether-colony-build-cycle
wrapper_additions:
  - id: planning-depth
    stage: pre
```

Update `plan.yaml` and `discuss.yaml`, add `spec.yaml`, then keep both managed wrapper projections semantically aligned. The existing plan YAML's two old preflight decisions (depth and research mode) should not coexist as a second owner-selection path once the locked preset flow is implemented.

**Cross-surface drift guard** (`cmd/command_guide.go:545-552`):

```go
var intelligentCommandDriftGuards = []string{
	"Keep canonical YAML, Claude wrapper, OpenCode wrapper, Codex skill, and command guide aligned.",
	"Keep Go runtime output authoritative for state transitions and structured results.",
}
```

Add `spec` and the revised plan/discuss stages to the guide and appropriate shipped skills. Preserve the project rule that Go owns lifecycle state, YAML owns canonical command intent, and wrappers/skills only orchestrate and present it.

**Managed-wrapper parity pattern** (`cmd/plan_wrapper_cards_test.go:53-96`):

```go
claude := readWrapper(t, ".claude/commands/ant/plan.md")
opencode := readWrapper(t, ".opencode/commands/ant/plan.md")
yaml := readWrapper(t, ".aether/commands/plan.yaml")

for _, heading := range expectedHeadings {
	assertHeadingOrder(t, claude, heading)
	assertHeadingOrder(t, opencode, heading)
}
for _, key := range runtimeKeys {
	assertContains(t, claude, key)
	assertContains(t, opencode, key)
	assertContains(t, yaml, key)
}
```

Create equivalent assertions for the spec flow and new plan/discuss cards. Keep managed headers, ordered stage parity, runtime key references, and retired-heading checks. If the public-command inventory is intentionally expanded for `spec`, update `.aether/commands/classic-command-parity.json` and `cmd/classic_command_parity_test.go` together; do not weaken the exact inventory assertion.

### 5. Contract corpus and co-located behavioral tests

**Applies to:** Classic contract fixtures, command/wrapper parity tests, and new Go unit/integration tests

**Primary analogues:** `cmd/classic_contract_test.go`, `cmd/plan_revision_test.go`, and `cmd/testdata/classic-contract/v1/*`.

**Closed, causal corpus model** (`cmd/classic_contract_test.go:56-129`):

```go
type classicContractCase struct {
	ID         string                    `json:"id"`
	Group      string                    `json:"group"`
	Invocation classicContractInvocation `json:"invocation"`
	Expected   classicContractExpected   `json:"expected"`
	State      classicContractState      `json:"state"`
	Replay     *classicContractReplay    `json:"replay,omitempty"`
}

type classicContractExpected struct {
	Outcome            string   `json:"outcome"`
	ReasonCode         string   `json:"reason_code"`
	RequiredArtifacts  []string `json:"required_artifacts,omitempty"`
	ForbiddenArtifacts []string `json:"forbidden_artifacts,omitempty"`
	StateAssertions    []string `json:"state_assertions,omitempty"`
	CausalFields       []string `json:"causal_fields,omitempty"`
}
```

Represent the Phase 200 journeys as semantic cases, not terminal snapshots. Required coverage should include each preset; happy-path iterations; each stop reason; every material-decision pause position; stale/replay/mismatched receipt failures; spec create/approve/revise; affected-only invalidation; candidate refusal and exact acceptance; legacy migration; and paired Claude/OpenCode semantics.

**Strict fixture decoding pattern** (`cmd/classic_contract_test.go:752-765`):

```go
decoder := json.NewDecoder(bytes.NewReader(data))
decoder.DisallowUnknownFields()
if err := decoder.Decode(&value); err != nil {
	return fmt.Errorf("decode strict JSON: %w", err)
}
if decoder.More() {
	return fmt.Errorf("expected exactly one JSON value")
}
```

Extend the versioned schema, cases, mechanisms, decision IDs/groups, and source citations together. Prefer an explicit allowed decision set or a narrowly updated Phase 199/200 expression; do not broaden validation until typos silently pass.

**Immutable-revision behavioral test pattern** (`cmd/plan_revision_test.go:14-93`):

```go
before := clonePhases(state.Plan.Phases)
if err := activateGeneratedPlan(&state, candidate, revisionContext, runID, now); err != nil {
	t.Fatal(err)
}
if diff := cmp.Diff(before[0], state.PlanRevisions[0].Phases[0]); diff != "" {
	t.Fatalf("completed phase changed (-want +got):\n%s", diff)
}
```

Use the same before/after assertions for specification immutability, preserved requirement IDs, impact-scoped invalidation, non-active candidates, and exact acceptance. Also retain content-bound evidence tests (`cmd/plan_revision_evidence_test.go:12-50`) so changing evidence bytes invalidates the receipt.

## Shared Patterns

### Validation and mutation boundary

Validate schema, identities, worker chain, evidence, workspace snapshot, spec/base revision, and stop/decision rules before writing. Then re-read and revalidate goal, candidate, spec revision, and plan base inside `store.UpdateJSONAtomically`. This prevents a valid-but-stale completion packet from mutating a newer state.

### Stable identities and hashes

Use canonical semantic content for IDs and content hashes. Keep lifecycle status out of plan definition hashes. Bind every completion, decision, approval, and acceptance receipt to the exact goal/run/iteration/stage/spec/base it authorizes.

### Owner authority

Owner choice is an explicit persisted event. `approved`, `answered`, and `accepted` are not inferred from generated files, renderer output, confidence scores, or worker claims. Replayed or stale tokens fail clearly and leave state unchanged.

### Structured runtime, thin presentation

Go runtime helpers decide transitions and return stable structured fields. `cmd/codex_visuals.go` formats those fields. YAML, Claude/OpenCode wrappers, Codex skills, and command guidance orchestrate the same stages without independently scoring readiness or materiality.

### Errors

Follow the existing contextual-error style: return `fmt.Errorf("<operation/context>: %w", err)` from helpers, use closed reason codes in structured results, and reserve successful pause states for expected owner decisions. Validation failures must not leave partial authoritative state.

### Backward compatibility

Use additive optional fields, explicit legacy policy values, and normalization that records what is known without fabricating consent. Existing legacy plans remain buildable. New plans and material revisions are held to the current approved-spec/candidate contract.

## Composite Analogues / No Exact Existing Domain Type

No file family is without a useful role analogue, but four Phase 200 concepts have no exact existing type and must combine the patterns above:

| New Concept | Compose From | Important Difference |
|---|---|---|
| `Specification` / `SpecRevision` | `AcceptedCharter` + `PlanRevision` | One canonical goal lineage, immutable feature revisions, requirement-level delta |
| `PlanningIterationReceipt` | planning manifest + intermediate iteration artifact | Five per-dimension causal evidence links and semantic plan delta after every Scout→Route pass |
| `MaterialDecisionCard` | scoped `PendingQuestion` + stale-scope resolution | Evidence-first consolidated batch, recommendation/consequences, stage-specific resume |
| `PlanCandidate` / acceptance receipt | completion packet + plan revision activation | Candidate is persisted but non-active until exact owner acceptance |

The planner should use these composite analogues rather than invent a parallel store, renderer, approval mechanism, or platform-only state machine.

## Planner Guardrails Derived From Existing Patterns

1. Keep planning state under the existing `.aether/data/planning/<run>/...` and colony-state persistence conventions; avoid a second database or opaque transcript parser.
2. Make the state-machine transition legalities pure/testable where possible, and make CLI handlers thin.
3. Preserve completed-phase immutability and stable IDs when revising a living plan.
4. Treat every iteration receipt as ordered durable history attached to the candidate/accepted plan.
5. Make all five readiness dimensions whole numbers and require fresh applicable evidence for each increase.
6. Make stop and pause reasons closed enums/reason codes so renderers and contract cases can assert them.
7. Update canonical YAML, both wrapper platforms, command guide, shipped skill guidance, public inventory, docs, and parity tests as one contract change.
8. Run focused tests first, then `go test ./...` and `go test ./... -race` before phase completion.

## Metadata

**Analogue search scope:** `cmd/`, `pkg/colony/`, `.aether/commands/`, `.aether/skills/colony/`, `.claude/commands/ant/`, `.opencode/commands/ant/`, and `cmd/testdata/`

**Concrete artefacts inspected:** 25 code, command, wrapper, skill, test, and fixture files, plus Phase 200 context and research

**Pattern extraction date:** 2026-09-07
