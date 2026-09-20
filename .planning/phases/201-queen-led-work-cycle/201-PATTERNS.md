# Phase 201: Queen-Led Work Cycle - Pattern Map

**Mapped:** 2026-09-10
**Files analyzed:** 9 (7 extend existing, 2 net-new suggested by RESEARCH.md)
**Analogs found:** 9 / 9

This phase is consolidation, not greenfield design (RESEARCH.md "Primary
recommendation"). Every file below is either an existing tested file to
extend in place, or a new file whose closest analog is another already-tested
Go file in `cmd/` doing the same *shape* of thing (pure judgement function,
durable append-only record, or dispatch-lane collapse). The mandatory
`201-CLASSIC-SYNTHESIS.md` (SYNTH-07 gate) is not a source file in this
codebase sense — it is a planning-gate deliverable and has no analog; do not
look for one.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/queen_judgement.go` (extend: add verification-boundary decision, D-01/D-02) | service (pure judgement fn) | request-response | itself — `queenApplyJudgement`/`queenCasteJudgement` (same file, same shape) | exact — extend in place |
| `cmd/codex_verify_advance.go` (NEW, suggested — D-04 unified accept/verify/advance) | service (orchestrator/collapser) | request-response | `cmd/deterministic_floor.go` (`runDeterministicFloor`) — already the single-function-many-callers pattern this file must replicate one layer up | exact — same collapse pattern, one layer higher |
| `cmd/codex_continue.go` (modify: becomes thin caller of new shared fn) | controller | request-response | itself (current in-process lane) | exact — refactor target |
| `cmd/codex_continue_plan.go` (modify: becomes thin caller) | controller | request-response | `cmd/codex_continue.go` (sibling lane already calling `runDeterministicFloor` identically) | exact |
| `cmd/codex_continue_finalize.go` (modify: becomes thin caller, fixes reconciliation-rejection bug) | controller | request-response | `cmd/codex_continue_plan.go` (external/JSON-completion sibling lane) | role-match — finalize has extra untrusted-JSON validation the other two don't |
| `cmd/build_attempt.go` (extend: D-05..D-08 outcome/cost/time/orphan fields) | model (durable record) | CRUD (append-only) | itself — `buildAttemptRecord` + `checkFixAttemptRecord`'s narrow-setter discipline (same file) | exact |
| `cmd/autopilot_policy.go` (extend: D-09..D-12 general bounded-repair, reusing repair ledger shape) | service (policy/ledger) | CRUD (budgeted, receipted) | itself — `autopilotRepairLedger`/`autopilotRepairReceipt` (same file) | exact |
| `cmd/job_telemetry.go` (NEW, suggested — D-13..D-16 per-job segment timing) | model + service (instrumentation) | event-driven / transform | `cmd/verification_scope.go` (`verificationScope` struct + `deriveVerificationScope`) — same "typed mode + reason, never a silent estimate" shape D-13's "unmeasured, never derived" rule needs | role-match — closest existing "honest, never-estimated status field" precedent |
| `pkg/codex/handoff.go` (consume, not modify — D-15b slimmer briefs is a selection/rendering change over this existing type) | model | transform | itself | exact — no schema change expected |

## Pattern Assignments

### `cmd/queen_judgement.go` — verification-boundary decision (D-01/D-02)

**Analog:** same file, `queenCasteJudgement` / `queenApplyJudgement` (lines 29-218, read this session)

**Core struct pattern to mirror** (lines 29-68):
```go
// queenCasteJudgement is the outcome of reconciling a proposed team with the
// floors and ceiling the runtime owns.
type queenCasteJudgement struct {
	Proposed        []string
	Final           []string
	Added           []string          // safety castes the proposal omitted and the runtime restored
	Dropped         []string          // proposed castes removed to fit the worker budget
	Refused         []string          // proposed castes the phase gives no work to
	Unknown         []string          // proposed names that are not dispatchable castes
	Rationale       string            // the Queen's TEAM summary — one string for the whole proposal
	Reasons         map[string]string // per-worker reason, keyed by caste
	RefusedNoReason []string
	Source          string            // "queen" or "deterministic"
}
```

**Reconciliation pattern — proposal as input, never authority** (lines 124-135, `queenApplyJudgement` empty-proposal branch):
```go
func queenApplyJudgement(proposed []string, rationale string, phase colony.Phase, flowType string, state colony.ColonyState, reasons ...map[string]string) queenCasteJudgement {
	normalized, unknown := normalizeProposedCastes(proposed)
	// An empty proposal is not an error and not an empty team: it means no
	// judgement was offered, so the deterministic engine decides. That keeps
	// every existing caller working unchanged and makes the model path
	// additive.
	if len(normalized) == 0 {
		dispatches := queenOrchestrate(phase, flowType, state)
		// ... returns queenCasteJudgement{Final: ..., Source: "deterministic"}
	}
	// ... required castes restored regardless of proposal (never negotiable)
}
```

**What to copy for D-01:** RESEARCH.md's Open Question #1 recommends *not* extending `queenCasteJudgement` itself but adding a small sibling type with the identical discipline:
```go
type verificationBoundaryDecision struct {
	Choice string // "build_end" | "continue"
	Reason string // required when Choice == "build_end" per D-02
	Source string // "queen" or "deterministic" (mirrors queenCasteJudgement.Source)
}
```
The reconciliation rule to copy: the Queen's choice is a *proposal*; the runtime records it as a single durable fact that both build-time and continue-time dispatch code read (per RESEARCH.md Pitfall 2) — never two independent derivations. This is the exact discipline `TestQueenChoiceReachesTheDispatchList` already enforces for team choice; D-01 needs the same test shape for boundary choice.

**Summary-rendering pattern** (lines 76-104, `Summary()` method) — copy the plain-English, no-jargon rendering style (`"Queen chose: ..."`, `"Added X — required for this phase regardless of the proposal."`) for the boundary decision's own summary line.

---

### `cmd/codex_verify_advance.go` (NEW) — D-04 unified accept/verify/advance

**Analog:** `cmd/deterministic_floor.go` (`runDeterministicFloor`, lines 1-60+, read this session)

**Single-function-many-callers pattern to replicate one layer up:**
```go
// runDeterministicFloor computes the deterministic floor... It never
// dispatches anything and never consults a depth flag, a review policy, or a
// caste proposal... This is the single body both runCodexContinueVerification
// (cmd/codex_continue.go, the in-process lane) and
// runCodexContinueVerificationSnapshot (cmd/codex_continue_plan.go, the
// wrapper/external lane) call, so lane parity is structural rather than a
// discipline (TestBothContinueLanesApplyTheSameFloor).
func runDeterministicFloor(ctx context.Context, root string, phase colony.Phase,
	manifest codexContinueManifest, watcher codexWatcherVerification,
	verificationTimeout time.Duration) deterministicFloorResult
```

**What to copy:** the doc-comment discipline itself — state explicitly, in the new function's comment, which callers become thin wrappers around it (per RESEARCH.md D-04/Pitfall 1: extract the *decision* logic — assessment, review dispatch, gate evaluation, advancement — into one function all three continue lanes call with lane-normalized inputs). Do **not** invent a new envelope type; operate on the existing shared types already used across all three lanes: `codexContinueManifest`, `codexContinueVerificationReport`, `codexContinueAssessment` (confirmed present in both `codex_continue.go` and `codex_continue_plan.go` via `grep` this session). The third lane (`codex_continue_finalize.go`) keeps its own untrusted-JSON parsing (`loadExternalContinueCompletion`, `validateExternalContinueState`) but must call the same shared decision function afterward, exactly as `runDeterministicFloor` is already called identically by both existing lanes.

**Verification-scope precedent for "honest, not a guess"** (`cmd/verification_scope.go` lines 12-47) — same discipline the new file must apply to the boundary/verification decision: a typed mode enum + a plain-English `Reason` field, never a silently-derived value.

---

### `cmd/codex_continue.go` / `cmd/codex_continue_plan.go` / `cmd/codex_continue_finalize.go` — collapse to thin callers

**Analog:** each other (they are already three parallel implementations of the same lifecycle stage — the phase's job is to make them converge, not diverge further)

**Existing shared-call precedent already proven** (grep this session, `cmd/deterministic_floor.go` doc comment): both `runCodexContinue` (`codex_continue.go:637`) and `runCodexContinueVerificationSnapshot` (`codex_continue_plan.go:269`) already call `runDeterministicFloor` identically. D-04 generalizes this exact pattern to the *decision* layer above the deterministic floor (assessment + review-gate + advancement), and extends it to the third lane's entry point (`runCodexContinueFinalize`, `codex_continue_finalize.go:139`).

**Known bug this collapse fixes** (RESEARCH.md/CONCERNS.md, cited for planner context): `codex_continue_finalize.go`'s `validateExternalContinueState`/completion-contract validation path (`completionContractError`, `contractViolation`) currently blocks finalize when reconciliation is supplied, while the direct lane (`codex_continue.go`) accepts it. Do not carry this divergence into the new shared function — the shared decision function's reconciliation-acceptance rule must be single-sourced.

---

### `cmd/build_attempt.go` — outcome/cost/time/orphan fields (D-05..D-08)

**Analog:** same file, `buildAttemptRecord` (lines 190-244) and `checkFixAttemptRecord`'s narrow-setter discipline (referenced doc comments)

**Append-only, narrow-setter pattern to copy exactly:**
```go
type buildAttemptRecord struct {
	SchemaVersion    int
	ID               string
	Phase            int
	Status           string
	// ...
	History          []buildAttemptTransition
	OutOfBandVerification *outOfBandVerificationRecord `json:"out_of_band_verification,omitempty"`
	// FreeChecks is set ONLY by attachBuildFreeCheckReport, called from
	// runCodexBuildFinalize when skipVerify is false. It is a report, never
	// an advancement gate.
	FreeChecks *buildFreeCheckReport `json:"free_checks,omitempty"`
	// CheckFix is set ONLY on a NEW attempt record created for D-02/D-03's
	// single bounded automatic builder fix attempt (attachCheckFixAttempt,
	// called from the continue verification path).
	CheckFix *checkFixAttemptRecord `json:"check_fix,omitempty"`
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	ParentJobName   string `json:"parent_job_name,omitempty"`
}
```

**What to copy for D-05..D-08:** new fields (outcome verdict, cost, per-attempt time, credited/orphan file lists) must follow the same "set ONLY by function X" doc-comment discipline and `omitempty` JSON tags so old attempt JSON keeps decoding — proven by `TestFixAttemptNeverOverwritesTheFirstResult` for the existing `CheckFix` field. Never overwrite `Status`, `Dispatches`, or `Claims`; add narrowly-scoped setter functions (mirror `attachCheckFixAttempt`, `attachBuildFreeCheckReport`) rather than a general mutator.

---

### `cmd/autopilot_policy.go` — general bounded repair (D-09..D-12)

**Analog:** same file, `autopilotRepairLedger` / `autopilotRepairReceipt` (lines 494-533+, read this session)

**Budgeted, receipted, one-shot repair pattern to copy exactly:**
```go
type autopilotRepairEvaluation struct {
	Eligible bool
	Pause    bool
	Reason   string
}

type autopilotRepairReceipt struct {
	ID              string
	Phase           int
	Attempt         string
	Check           string
	Status          autopilotRepairStatus // "planned" | "passed" | "failed"
	PreparedAt      string
	CompletedAt     string
	BudgetBefore    int
	BudgetRemaining int
	Verification    autopilotRepairVerification
}

type autopilotRepairLedger struct {
	SchemaVersion  int
	InvocationID   string
	InitialBudget  int
	Remaining      int
	Receipts       []autopilotRepairReceipt
	Debt           []colony.LifecycleIssue
	Blockers       []colony.LifecycleIssue
}
```

**What to copy for D-09..D-12:** the "one checkpointed round" shape is already here — `Status` is a closed enum (`autopilotRepairPlanned`/`Passed`/`Failed`, no retry state), and budget is tracked before/after per receipt. D-09's "no second automatic attempt" must be enforced the same way this ledger already enforces single-attempt-per-check: extend/generalize this ledger's use to build/continue's own repair rather than building a second budget-tracking struct (RESEARCH.md "Don't Hand-Roll" table + Anti-Pattern: silent second automatic repair attempt).

**Checkpoint-identity integration point (D-10, Pitfall 3):** resolve whether this repair ledger's checkpoint reuses the `/ant-pause`/`/ant-resume` handoff-ID idempotency pattern from Phase 199 before writing new checkpoint code — this must be an explicit `SYN-201-*` synthesis decision, not an assumption (RESEARCH.md Assumption A2).

---

### `cmd/job_telemetry.go` (NEW) — per-job segment timing (D-13..D-16)

**Analog:** `cmd/verification_scope.go` — `verificationScope` struct (lines 32-47) and `deriveVerificationScope`'s "honest, never a guess" discipline

**Honest-status-field pattern to copy exactly:**
```go
// verificationScope records how much of the project's tests this
// verification pass actually ran, and why...
type verificationScope struct {
	Mode         string   `json:"mode"`   // one of a closed set of named modes
	Packages     []string `json:"packages,omitempty"`
	PackageCount int      `json:"package_count,omitempty"`
	Reason       string   `json:"reason"` // plain-English, no repo jargon
}
```

**What to copy for D-13/D-14:** each of the eight named segments (queue, preflight, model, tool-call, context, work, verification, wait) needs a field that is either a real measured duration *or* an explicit "unmeasured" sentinel — never a derived remainder (RESEARCH.md Pitfall 4, and the project's own cost-ledger precedent: `"'Reported' keys on the usage source tag, never on a number ... An estimated ledger row renders the dash sentinel exactly like an empty one"` — STATE.md Phase 196). Mirror that exact "reported vs. unreported, sentinel not estimate" discipline for each timing segment. Bind the record to the same durable attempt `ID` already used for evidence and cost (`buildAttemptRecord.ID`) rather than inventing a separate identity.

---

### `pkg/codex/handoff.go` — slimmer briefs / compact handoffs (D-15b)

**Analog:** itself — no schema change expected, this is a selection/rendering change over existing data

**Existing type to reuse as-is** (lines 10-20):
```go
type WorkerHandoff struct {
	ChangedFiles           stringList `json:"changed_files,omitempty"`
	CommandsRun            stringList `json:"commands_run,omitempty"`
	VerificationStatus     string     `json:"verification_status,omitempty"`
	KnownFailures          stringList `json:"known_failures,omitempty"`
	OpenDecisions          stringList `json:"open_decisions,omitempty"`
	Assumptions            stringList `json:"assumptions,omitempty"`
	NextWorkerInstructions stringList `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            stringList `json:"do_not_repeat,omitempty"`
	Freshness              string     `json:"freshness,omitempty"`
}
```
**What to copy:** D-15b is "slimmer briefs and compact handoffs" — a rendering/selection change over data this type already carries (RESEARCH.md "Don't Hand-Roll" table). Do not add a second handoff schema; change what is *selected* for injection into a worker's brief, following whatever budget/trim-order convention already governs colony-prime context assembly (CLAUDE.md "Token Budget" trim order, same discipline: lowest-priority content trimmed first, blockers never trimmed).

## Shared Patterns

### Pure judgement function: proposal in, validated/recorded decision out
**Source:** `cmd/queen_judgement.go` (`queenApplyJudgement`), `cmd/coherent_jobs.go` (`planCoherentJobs`)
**Apply to:** D-01 boundary decision, D-07 next-action recommendation
```go
// planCoherentJobs validates the whole phase before considering a proposal,
// then accepts safe Queen jobs and returns only refused members to automatic
// planning. It is intentionally pure: it never writes state, creates an
// attempt, constructs a manifest, or touches a worktree.
func planCoherentJobs(phase colony.Phase, seeds []coherentJobTask, proposals []coherentJobProposal) (*coherentJobPlan, error)
```
Every "the Queen chooses X" decision this phase adds must be a pure function: input is a proposal, output states exactly what was kept/changed/refused and why, and the function itself never mutates state or dispatches anything.

### Single shared decision function, many thin lane callers
**Source:** `cmd/deterministic_floor.go` (`runDeterministicFloor`)
**Apply to:** D-04 (all three continue lanes), D-01 (build-time and continue-time dispatch code must read one boundary-choice record, not two independent derivations — RESEARCH.md Pitfall 2)

### Append-only durable record with narrow setters
**Source:** `cmd/build_attempt.go` (`buildAttemptRecord`, `checkFixAttemptRecord`)
**Apply to:** D-05..D-08 result-truth fields, D-13..D-16 telemetry (bind to the same attempt `ID`)
```go
// CheckFix is set ONLY on a NEW attempt record created for D-02/D-03's single
// bounded automatic builder fix attempt (attachCheckFixAttempt, called from
// the continue verification path). ... never overwriting Status, Dispatches,
// or Claims.
```

### Budgeted, receipted, one-shot repair ledger
**Source:** `cmd/autopilot_policy.go` (`autopilotRepairLedger`)
**Apply to:** D-09..D-12 general bounded recovery — generalize, do not duplicate

### Honest "unmeasured/unreported" sentinel, never an estimate
**Source:** `cmd/verification_scope.go` (`verificationScope`), and the cost ledger precedent named in STATE.md Phase 196 ("'Reported' keys on the usage source tag, never on a number")
**Apply to:** D-06 cost line, D-13/D-14 per-segment telemetry — a field with no direct evidence renders as unmeasured/unreported, never derived or guessed

## No Analog Found

None. Every file in scope either extends an existing tested file in place or has a direct structural analog elsewhere in `cmd/` for the same pattern shape (pure judgement fn, shared single-caller function, append-only record, budgeted ledger, honest-sentinel field). The one genuinely new capability — per-job telemetry (D-13..D-16) — still has a strong shape-analog in `verificationScope`'s honest-status-field discipline, so it is listed as role-match rather than no-analog.

## Metadata

**Analog search scope:** `cmd/` (Go command implementations), `pkg/codex/` (shared worker-handoff type)
**Files scanned:** `cmd/queen_judgement.go`, `cmd/coherent_jobs.go`, `cmd/build_attempt.go`, `cmd/autopilot_policy.go`, `cmd/deterministic_floor.go`, `cmd/verification_scope.go`, `cmd/codex_continue.go`, `cmd/codex_continue_plan.go`, `cmd/codex_continue_finalize.go`, `pkg/codex/handoff.go`
**Pattern extraction date:** 2026-09-10
