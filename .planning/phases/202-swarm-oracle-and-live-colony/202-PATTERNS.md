# Phase 202: Swarm, Oracle, and Live Colony - Pattern Map

**Mapped:** 2026-09-11
**Files analyzed:** 9 (per RESEARCH.md "Recommended Project Structure")
**Analogs found:** 9 / 9

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `pkg/events/bus.go` (topic constants + typed payload structs for `swarm.*`/`oracle.*`/`watch.*`) | service (event bus) | pub-sub/event-driven | `pkg/events/ceremony.go` (`CeremonyTopic*` constants + `CeremonyPayload`) | exact |
| `cmd/swarm_lens.go` (NEW — hypothesis struct, compare/rank) | service (transform) | transform/batch | `cmd/oracle_loop.go` (`oracleWorkerResponse`/`oracleWorkerFinding`/`oracleWorkerEvidence`) | exact (explicitly named reuse target in RESEARCH.md Pattern 3) |
| `cmd/swarm_repair_checkpoint.go` (NEW — thin adapter over checkpoint primitives) | service (adapter) | CRUD (save/restore) | `cmd/work_repair.go` (`runBoundedRepairRound`, `repairRoundInput`/`repairRoundOutcome`, `saveRepairCheckpoint`/`restoreRepairCheckpoint`) | exact (CONTEXT.md forbids a third checkpoint mechanism — reuse, don't re-pattern) |
| `cmd/swarm_cmd.go` (MODIFY — insert lens-compare step between investigation and fix waves) | controller (workflow orchestrator) | event-driven/batch | same file, `runSwarmDestroy` (existing linear pipeline) | exact (self-analog; extend in place) |
| `cmd/oracle_loop.go` (MODIFY — emit `oracle.*` events at existing `oracleStateFile` mutation points) | controller (workflow orchestrator) | event-driven | `cmd/spawn.go` (`emitSpawnTreeCeremony` — existing `bus.Publish` call site) | exact (only known example of a Go mutation site calling `events.Bus.Publish` outside the learning pipeline) |
| `cmd/oracle_preset.go` (NEW — Fast/Balanced/Deep/Exhaustive label mapping over `oracleDepthLevels`) | config/mapping | transform | `cmd/codex_plan.go` (`planningPresetPolicies`, `planningPresetPolicy`, `resolvePlanningPreset`) | exact (this is the literal vocabulary D-09 requires reusing) |
| `cmd/compatibility_cmds.go` (MODIFY — `watchCmd` branches live vs replay-backed vs idle) | controller (CLI command) | request-response | same file, `buildIdleWatchResult` (existing idle-only branch) | exact (self-analog; existing idle branch becomes branch 3 of 3) |
| `cmd/status.go` / `cmd/history.go` (MODIFY — read swarm/oracle episode records) | controller (CLI command / read projection) | request-response | `cmd/status.go` `loadRecentLoopBreakEvents`/`renderLoopSafetySection` (existing bus-query + render pattern) | exact (self-analog; extend query scope to `swarm.*`/`oracle.*` topics and episode stores) |
| `cmd/event_types.go`, `cmd/event_stream.go`, `cmd/event_bridge.go` (RETIRE) | dead code | n/a | n/a | n/a — no analog needed; delete/fold constants into `pkg/events` per Pattern 1 |

## Pattern Assignments

### `pkg/events/bus.go` — new topic constants + typed payloads (service, pub-sub)

**Analog:** `pkg/events/ceremony.go`

**Topic constant pattern** (lines 6-34):
```go
const (
	CeremonyTopicBuildPrewave          = "ceremony.build.prewave"
	CeremonyTopicBuildWaveStart        = "ceremony.build.wave.start"
	CeremonyTopicBuildSpawn            = "ceremony.build.spawn"
	...
	CeremonyTopicOraclePhaseTransition = "ceremony.oracle.phase_transition"
	CeremonyTopicOracleIteration       = "ceremony.oracle.iteration"
)
```
Follow this exact dotted-namespace convention for new topics: `swarm.lens_started`, `swarm.hypothesis_formed`, `swarm.checkpoint_saved`, `swarm.checkpoint_restored`, `oracle.round_started`, `oracle.confidence_changed`, `oracle.contradiction_found`, `watch.snapshot` — note `CeremonyTopicOracleIteration` already exists, so check for overlap before adding a duplicate Oracle topic.

**Payload struct + serialization pattern** (from `CeremonyPayload`, line 67 area):
```go
func (p CeremonyPayload) RawMessage() (json.RawMessage, error) {
	// marshal the typed payload struct to json.RawMessage for Bus.Publish
}
```
New payload structs (`SwarmLensPayload`, `OracleRoundPayload`, etc.) should mirror `CeremonyPayload`'s shape: a plain struct with JSON tags and a `RawMessage()` method, never a bare `map[string]interface{}`.

**Publish signature to call against** (`pkg/events/bus.go:47`):
```go
func (b *Bus) Publish(ctx context.Context, topic string, payload json.RawMessage, source string) (*Event, error)
```

**Query/replay signature** (used by `status.go`):
```go
evts, err := bus.Query(context.Background(), events.CeremonyTopicLoopBreak, since, 5)
```

---

### `cmd/swarm_lens.go` (NEW) — hypothesis compare/rank (service, transform)

**Analog:** `cmd/oracle_loop.go` (`oracleWorkerResponse` family)

**Shape to mirror as a Swarm-owned sibling type** (`cmd/oracle_loop.go:418-436`):
```go
type oracleWorkerResponse struct {
	QuestionID     string                `json:"question_id"`
	Status         string                `json:"status"`
	Confidence     int                   `json:"confidence"`
	Summary        string                `json:"summary"`
	Findings       []oracleWorkerFinding `json:"findings,omitempty"`
	Gaps           []string              `json:"gaps,omitempty"`
	Contradictions []string              `json:"contradictions,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
}

type oracleWorkerFinding struct {
	Text     string                 `json:"text"`
	Evidence []oracleWorkerEvidence `json:"evidence,omitempty"`
}

type oracleWorkerEvidence struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	Type     string `json:"type"`
}
```
Per RESEARCH.md Open Question 2 and the Phase 201 synthesis precedent ("a small sibling type... never extending it"): define `swarmHypothesis{LensName, Claim string, Confidence int, Evidence []swarmHypothesisEvidence, Contradictions []string, ProposedRepair string}` as a **new Swarm-owned type with the same field shape**, not a literal reuse of the Oracle type — this avoids coupling Swarm's evolution to Oracle's package boundary.

**What it must NOT be** — the current all-text concatenation this replaces (`cmd/swarm_cmd.go:274-372` / `renderSwarmFindingSummary`): a free-text string handed straight to the fix wave, with no structured comparison. The new `swarm_lens.go` must produce a structured, ranked artifact (per-lens hypothesis list, contradictions surfaced, one selected repair with reasoning) — rendered as "one card at the end" per D-06.

**Existing type to extend/reference for source data** (`cmd/swarm_cmd.go:44-57`):
```go
type swarmWorkerResponse struct {
	Role           string   `json:"role"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	Findings       []string `json:"findings,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	RootCause      string   `json:"root_cause,omitempty"`
	Recommendation string   `json:"recommendation,omitempty"`
	ProposedFix    string   `json:"proposed_fix,omitempty"`
	FilesTouched   []string `json:"files_touched,omitempty"`
	TestsWritten   []string `json:"tests_written,omitempty"`
	Verification   []string `json:"verification,omitempty"`
}
```
Note it has **no confidence field today** — the lens layer must add confidence scoring (either on this type or by mapping into the new `swarmHypothesis` type at the comparison step).

---

### `cmd/swarm_repair_checkpoint.go` (NEW) — checkpoint adapter (service, CRUD save/restore)

**Analog:** `cmd/work_repair.go`

**Primitives to call, not reimplement** (`cmd/work_repair.go` function signatures cited in RESEARCH.md, verified lines 240-459):
```go
func repairCheckpointIdentity(phase int, check string) string
func saveRepairCheckpoint(root, checkpointID string, paths []string) (repairCheckpoint, error)
func restoreRepairCheckpoint(checkpoint repairCheckpoint) error
func emitRepairCheckpointSaved(phase int, check string)
func emitRepairCheckpointRestored(phase int, check string)
```

**What the new adapter file does:** Swarm's repair round is target-shaped (free-text `target`), not phase+check-shaped. `swarm_repair_checkpoint.go` should define a generalized identity function (Swarm's own version of `repairCheckpointIdentity`, keyed by `swarmID`/target fingerprint instead of phase+check) and call `saveRepairCheckpoint`/`restoreRepairCheckpoint` directly — it must NOT call `runBoundedRepairRound` or `applyBoundedCheckFixRepair` (those are phase/check-shaped wrappers owned by Phase 201's continue flow), and it must NOT reimplement snapshot/restore logic.

**Doc-comment convention to follow** (top of `work_repair.go`, lines 18-38): explain in a comment why this is the one checkpoint path being reused, referencing CONTEXT.md's "not a third mechanism" constraint, mirroring the existing file's own self-documentation style.

---

### `cmd/swarm_cmd.go` (MODIFY) — insert lens-compare step (controller, event-driven orchestration)

**Analog:** same file, existing `runSwarmDestroy` (self-extension)

**Current linear pipeline to extend** (verified `cmd/swarm_cmd.go:274-372`):
```go
investigation := buildSwarmInvestigationPlans(root, target)
...
investigationRuns, err := executeSwarmWave(ctx, root, swarmID, target, investigation, "", invoker)
...
findingSummary := renderSwarmFindingSummary(investigationRuns)
fixPlans := buildSwarmFixPlans(root, target)
...
builderRuns, err := executeSwarmWave(ctx, root, swarmID, target, fixPlans, findingSummary, invoker)
```

**Insertion point:** between `investigationRuns` and `findingSummary`/`fixPlans` — replace (or supplement) `renderSwarmFindingSummary`'s free-text concatenation with a call into `swarm_lens.go`'s compare/rank function, then call `swarm_repair_checkpoint.go`'s save-checkpoint before dispatching `fixPlans`, and call its restore function if re-verification after the fix wave fails (with announcements per D-05, mirroring `emitRepairCheckpointSaved`/`emitRepairCheckpointRestored`'s announce-at-the-moment style).

**Caste-to-lens task descriptions already present** (`cmd/swarm_cmd.go:1071-1093`, keep unchanged, extend with a 4th case):
```go
func swarmTaskForCaste(caste string) string {
	switch caste {
	case "tracker":
		return "Reproduce the issue, trace the failure path, and identify the most likely root cause."
	case "scout":
		return "Search the repo for the most relevant files, patterns, tests, and documentation tied to the reported bug."
	case "archaeologist":
		return "Inspect git history and prior fixes around the bug area to identify historical context, fragile zones, and regressions."
	...
```

**Existing post-run durability call — do not replace** (`cmd/swarm_cmd.go:349-361`):
```go
persistSwarmResultOutcome(store, swarmResultRecord{
	SwarmID, Target, Status, RootCause, Solution, Recommendation, Workers, Files, Tests, Blockers, CompletedAt,
})
```
This must remain the single path feeding `evaluateSwarmStrikeHistory`/three-strike escalation unchanged.

---

### `cmd/oracle_loop.go` (MODIFY) — event emission at mutation points (controller, event-driven)

**Analog:** `cmd/spawn.go` (`emitSpawnTreeCeremony`)

**Publish call-site pattern to replicate** (`cmd/spawn.go:139,236-246`):
```go
eventID := emitSpawnTreeCeremony(events.CeremonyPayload{
	// ...typed fields...
})

func emitSpawnTreeCeremony(payload events.CeremonyPayload) string {
	raw, err := payload.RawMessage()
	...
	bus := events.NewBus(store, events.DefaultConfig())
	evt, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, raw, "aether-spawn")
	...
}
```

**Mutation points to hook in `oracle_loop.go`** (verified struct fields, lines 300-337): every place `oracleStateFile.Iteration`, `.OverallConfidence`, `.ActiveQuestionText`, `.Contradictions`, `.OpenGaps`, or `.Novelty` is assigned during `runOracleLoop` needs an adjacent `emitOracle*` call following the `emitSpawnTreeCeremony` shape — build the typed payload, marshal via `RawMessage()`, publish, non-fatal on error (mirror `spawn.go`'s error-tolerant pattern; a failed event emit must never abort the research loop).

**Fields already available to put in payloads** (no new state to add — verified `cmd/oracle_loop.go:300-337`):
```go
type oracleStateFile struct {
	Iteration          int            `json:"iteration,omitempty"`
	MaxIterations      int            `json:"max_iterations,omitempty"`
	TargetConfidence   int            `json:"target_confidence,omitempty"`
	OverallConfidence  int            `json:"overall_confidence,omitempty"`
	ActiveQuestionID   string         `json:"active_question_id,omitempty"`
	ActiveQuestionText string         `json:"active_question_text,omitempty"`
	OpenGaps           []string       `json:"open_gaps,omitempty"`
	Contradictions     []string       `json:"contradictions,omitempty"`
	Novelty            noveltyTracker `json:"novelty,omitempty"`
}
```

---

### `cmd/oracle_preset.go` (NEW) — Fast/Balanced/Deep/Exhaustive label mapping (config, transform)

**Analog:** `cmd/codex_plan.go` (`planningPresetPolicies`)

**Exact vocabulary/shape to mirror** (`cmd/codex_plan.go:205-224`):
```go
type planningPresetPolicy struct {
	ID               string
	Label            string
	TargetConfidence int                 `json:"target"`
	PassCap          int                 `json:"max_iterations"`
}

var planningPresetPolicies = []planningPresetPolicy{
	{ID: planningStagePresetFast, Label: "Fast", TargetConfidence: 80, PassCap: 4},
	{ID: planningStagePresetBalanced, Label: "Balanced", TargetConfidence: 90, PassCap: 6},
	{ID: planningStagePresetDeep, Label: "Deep", TargetConfidence: 95, PassCap: 8},
	{ID: planningStagePresetExhaustive, Label: "Exhaustive", TargetConfidence: 99, PassCap: 12},
}
```

**Existing data being relabeled, values unchanged** (`cmd/oracle_loop.go:57-64`):
```go
var oracleDepthLevels = map[string]oracleDepthConfig{
	"quick":      {5, 60, "Quick", "Fast overview, up to 5 iterations"},
	"balanced":   {15, 85, "Balanced", "Standard research, up to 15 iterations"},
	"standard":   {15, 85, "Balanced", "Standard research, up to 15 iterations"},
	"deep":       {30, 95, "Deep", "Deep dive, up to 30 iterations"},
	"exhaustive": {50, 99, "Exhaustive", "Marathon convergence, up to 50 iterations"},
	"marathon":   {50, 99, "Exhaustive", "Marathon convergence, up to 50 iterations"},
}
```
`oracle_preset.go` should be a **pure label-mapping layer**: map `quick→Fast`, `standard`/`balanced`→`Balanced`, `deep→Deep`, `marathon`/`exhaustive`→`Exhaustive`, displaying each preset's own confidence target and iteration cap alongside the shared name — per D-09 and Pitfall 4, `oracleDepthLevels`'s numeric values must NOT change.

---

### `cmd/compatibility_cmds.go` (MODIFY) — three-way `watchCmd` branch (controller, request-response)

**Analog:** same file, existing idle-only branch (self-extension)

**Current unconditional idle command** (verified lines 59-70):
```go
var watchCmd = &cobra.Command{
	Use:         "watch",
	Short:       "Show the honest idle watch fallback",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = cmd // --once/--interval remain accepted compatibility flags.
		result := buildIdleWatchResult(resolveAetherRoot(), store, time.Now().UTC())
		outputWorkflow(result, renderIdleWatchVisual(result))
		return nil
	},
}
```

**Idle result builder to keep as branch 3 of 3** (verified lines 415-455) — note its explicit unconditional liveness suppression, which the new logic must make conditional rather than delete:
```go
// A spawn-tree row is durable history, not a typed live event. Preserve
// terminal rows for the compact snapshot and suppress every live-looking
// status until Phase 202 supplies an event source with liveness semantics.
...
return map[string]interface{}{
	"schema_version": LifecycleResultSchemaVersion,
	"mode":           "idle_watch", "command": "watch",
	"idle_message":    "No ants are active right now",
	"live_capability": "unsupported", "active_count": 0,
	...
}
```
The comment `"until Phase 202 supplies an event source with liveness semantics"` is this phase's own named trigger — the branch logic (live episode running → dashboard; no live episode but history exists → D-03 replay summary; no history at all → this exact idle path unchanged) must resolve in Go (`RunE`), never in a wrapper, per RESEARCH.md Pattern 2 and Anti-Pattern guidance.

---

### `cmd/status.go` / `cmd/history.go` (MODIFY) — episode surfacing (controller, request-response)

**Analog:** same file, `loadRecentLoopBreakEvents` / `renderLoopSafetySection`

**Existing bus-query-then-render pattern to replicate for swarm/oracle topics** (verified `cmd/status.go:389-419`):
```go
func loadRecentLoopBreakEvents(s *storage.Store) []events.Event {
	...
	bus := events.NewBus(s, events.DefaultConfig())
	evts, err := bus.Query(context.Background(), events.CeremonyTopicLoopBreak, since, 5)
	...
}

func renderLoopSafetySection(loopEvents []events.Event) string {
	...
	var payload events.CeremonyPayload
	...
}
```
New functions (`loadRecentSwarmEpisodes`, `loadRecentOracleSyntheses`) should follow this exact shape: query the bus (or the durable `swarmResultRecord`/oracle research doc stores directly, per D-11's "read, don't replace" instruction) and render a section, added alongside the existing loop-safety section rather than replacing it.

**Durable records already available to read (no new store needed):**
- `persistSwarmResultOutcome(store, swarmResultRecord{...})` — `cmd/swarm_cmd.go:349-361`
- `saveOracleResearchDocument(paths, state, plan, name)` / `runResearchList(root)` — `cmd/oracle_research_doc.go:47-335`

---

## Shared Patterns

### Event publish/query round-trip
**Source:** `pkg/events/bus.go` (`Publish`, `Query`), used today by `cmd/spawn.go` (write side) and `cmd/status.go` (read side)
**Apply to:** `cmd/oracle_loop.go` new emission calls, `cmd/swarm_cmd.go` new lens/checkpoint emission calls, `cmd/status.go`/`cmd/compatibility_cmds.go` new read calls.
```go
// write side (cmd/spawn.go:246)
evt, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, raw, "aether-spawn")

// read side (cmd/status.go:395)
evts, err := bus.Query(context.Background(), events.CeremonyTopicLoopBreak, since, 5)
```

### Checkpoint save/restore with announcement
**Source:** `cmd/work_repair.go` (`saveRepairCheckpoint`/`restoreRepairCheckpoint`/`emitRepairCheckpointSaved`/`emitRepairCheckpointRestored`)
**Apply to:** `cmd/swarm_repair_checkpoint.go`, and the call sites added inside `cmd/swarm_cmd.go`'s `runSwarmDestroy`. Save must be announced before the fix wave dispatches; restore must be announced only on failed re-verification (D-05's literal requirement).

### Sibling type over cross-package extension
**Source:** Phase 201 synthesis precedent, applied to `oracleWorkerResponse`/`oracleWorkerFinding`/`oracleWorkerEvidence` (`cmd/oracle_loop.go:418-436`)
**Apply to:** `cmd/swarm_lens.go`'s new hypothesis type — same field shape, Swarm-owned, not literally imported/extended from Oracle's type.

### Preset label mapping over existing numeric machinery
**Source:** `cmd/codex_plan.go` (`planningPresetPolicies`)
**Apply to:** `cmd/oracle_preset.go` — reuse the label vocabulary and struct shape; never rewrite the underlying numeric machinery (`oracleDepthLevels`) to match.

### Caste visual identity (D-04 classic presentation)
**Source:** `cmd/codex_visuals.go` — `casteEmojiMap` (line 42), `casteColorMap` (line 89), `casteLabelMap` (line 132), `casteEmoji()`/`casteLabel()`/`casteANSIColor()`/`casteIdentity()` (lines 5554-5666)
**Apply to:** any new rendering in the live dashboard, Swarm lens display, and Oracle round display — reuse these functions rather than inventing new emoji/color/label maps per command, per CLAUDE.md's Caste Identity System and the UX Architecture table (colony framing in wrapper markdown, but the underlying renderer stays this one Go source).

## No Analog Found

None — every file in RESEARCH.md's Recommended Project Structure has a strong same-repo analog (several are self-analogs, i.e. the same file being extended in place). The `cmd/event_types.go`/`event_stream.go`/`event_bridge.go` trio needs no analog: RESEARCH.md's own recommendation is retirement/folding into `pkg/events`, not building a replacement from a pattern.

## Metadata

**Analog search scope:** `cmd/` (swarm_*.go, oracle_*.go, work_repair.go, compatibility_cmds.go, status.go, spawn.go, codex_plan.go, codex_visuals.go), `pkg/events/` (bus.go, event.go, ceremony.go)
**Files scanned:** 15 read directly this session (plus RESEARCH.md/CONTEXT.md as primary input)
**Pattern extraction date:** 2026-09-11
</content>
