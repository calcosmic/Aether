# Phase 203: Biological Runtime - Pattern Map

**Mapped:** 2026-09-12
**Files analyzed:** 8 (from RESEARCH.md "Recommended Project Structure")
**Analogs found:** 8 / 8

This phase is almost entirely extend-don't-fork work: every new file has a
direct, already-tested analog it must call into rather than reimplement.
Per RESEARCH.md's own risk framing, the failure mode to guard against is a
new file that *references* these analogs in a comment but never actually
calls them.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/recruitment_intent.go` (BIO-01) | model/validation | request-response | `cmd/spawn.go` (`spawnDecisionInput`) | role-match (superset wire shape) |
| `cmd/recruitment_admission.go` (BIO-02) | service (admission gate) | request-response | `cmd/spawn.go` (`spawnCanSpawnDecision`) + `cmd/spawn_budget.go` + `cmd/spawn_ancestor.go` | exact (same chokepoint, extend not fork) |
| `cmd/recruitment_dispatch.go` (BIO-03) | service (dispatch) | event-driven / process-spawn | `.aether/ts-host/src/platform-dispatcher.ts` (`spawnWorker`) | exact (proven depth-1 pattern) |
| `.aether/ts-host/src/recruitment-probe.ts` (BIO-03 probe) | utility | request-response (bounded probe) | `.aether/ts-host/src/platform-dispatcher.ts` (subprocess pattern) | role-match |
| `cmd/recruitment_result.go` (BIO-04) | model + service (idempotent binding) | event-driven / exactly-once | `pkg/colony/lifecycle.go` (`PauseHandoff`, `LifecycleTransactionReference`, `LifecycleReceiptReference`) | exact (reuse verbatim) |
| `cmd/trophallaxis.go` (BIO-05) | model + service (handback + ack) | request-response | `cmd/codex_dispatch_contract.go` (`workerHandoffRecord`) | exact (structural precedent) |
| `cmd/pheromone_resolver.go` (BIO-07) | service (resolver) | transform/read | `cmd/pheromone_write.go` (`writePheromoneSignal`) + `cmd/codex_build.go` (`resolvePheromoneSection`, naive read to extend) | role-match (write side exact; read side is the actual gap) |
| `cmd/pheromone_outcome.go` (BIO-08/CEC-07) | service (outcome join) | event-driven | `cmd/agency_contract.go` (`AgencyReceiptEvidence`, `SwarmPhase202Limitation` stub) + `cmd/suggest_approve.go` (`colony.PendingSuggestion` tick-to-approve) | role-match (approval queue exact; outcome join is the stub to replace) |
| (support) permission check in BIO-02 | utility | pure/in-memory | `pkg/codex/permission_profile.go` (`PermissionProfileForCaste`) | exact |
| (support) path check in BIO-02 | utility | pure/in-memory | `cmd/spend_session_capture.go` (`validateSpendContainedPath`) | exact |
| (support) live event topic for BIO-06 | event/model | pub-sub | `pkg/events/ceremony.go` (`CeremonyTopicBuildSpawn`) | exact (extend with `ceremony.recruitment.*` family, don't replace) |

## Pattern Assignments

### `cmd/recruitment_intent.go` (BIO-01)

**Analog:** `cmd/spawn.go` — `spawnDecisionInput` / `spawnDecisionResult` (lines 276-300)

**Core struct pattern to extend (not replace):**
```go
// Source: cmd/spawn.go:276-292
type spawnDecisionInput struct {
	RequesterName        string
	RequesterDepth       int
	DepthIsAuthoritative bool
	Caste                string
	Task                 string
}
```
BIO-01's `RecruitmentIntent` is a documented **superset** of this shape:
authenticated parent/attempt identity, requested caste/capability, bounded
objective, reason, evidence, permissions, urgency, scope, cost limits. Model
it as a distinct type (greenfield — nothing named `RecruitmentIntent` exists
today, per RESEARCH.md) but keep the same discipline: exported fields, no
behavior in the struct itself, a companion `...Result` type with an
`Allowed bool` / `Reason string` / `Detail string` shape identical to
`spawnDecisionResult` (lines 294-300) so downstream admission code can share
one deny-rendering convention.

**Convention note:** `spawnDecisionResult.Reason` uses exact fixed strings
("depth", "budget", "ancestor-cycle") never free text — BIO-01/02's new
reasons ("permission", "path", "cost", "duplicate") must follow the same
enum-by-string convention so callers can switch on `Reason` reliably.

---

### `cmd/recruitment_admission.go` (BIO-02)

**Analog:** `cmd/spawn.go` — `spawnCanSpawnDecision` (lines 307-330), extended with checks from `cmd/spawn_budget.go` (`spawnTreeBudgetReason`) and `cmd/spawn_ancestor.go` (`spawnAncestorCycleReason`)

**The exact chokepoint to extend, verbatim (do not fork a parallel admission path):**
```go
// Source: cmd/spawn.go:307-330 (read this session)
var spawnCanSpawnDecision = func(in spawnDecisionInput) spawnDecisionResult {
	prospectiveDepth := in.RequesterDepth + 1
	if prospectiveDepth > spawnMaxDelegationDepth {
		requesterName := in.RequesterName
		if requesterName == "" {
			requesterName = "the requester"
		}
		return spawnDecisionResult{
			Allowed: false,
			Reason:  "depth",
			Detail: fmt.Sprintf(
				"%s is at depth %d; a helper spawned from here would be depth %d, past the cap of %d",
				requesterName, in.RequesterDepth, prospectiveDepth, spawnMaxDelegationDepth,
			),
		}
	}
	if reason := spawnTreeBudgetReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "budget", Detail: reason}
	}
	if reason := spawnAncestorCycleReason(in); reason != "" {
		return spawnDecisionResult{Allowed: false, Reason: "ancestor-cycle", Detail: reason}
	}
	return spawnDecisionResult{Allowed: true}
}
```
BIO-02 adds four more ordered steps inside this same function (permission,
path, cost, duplicate) — each denies-by-name with a `Detail` sentence, same
convention as the depth check's `fmt.Sprintf` line above. **This is a package-level `var`, not a plain `func`, deliberately** — comment at
`cmd/spawn.go:304-306` explains it exists so a test can substitute a deny
answer for a single test case. Preserve that shape.

**Ancestor-cycle refusal-by-name pattern (mirror for permission/path/cost/duplicate):**
```go
// Source: cmd/spawn_ancestor.go:100-108
return fmt.Sprintf(
	"%s was already asked to do this task by %s at depth %d; spawning it again would repeat work already in progress above",
	in.Caste, ancestor.AgentName, ancestor.Depth,
)
```
Every denial names the caste, the reason class, and a human sentence — never
a bare boolean (per RESEARCH.md Pattern 2 and CONTEXT.md D-06).

**D-19 fail-closed discipline (copy this exactly):**
```go
// Source: cmd/spawn_ancestor.go:95-98
chain, err := spawnAncestorChain(st, in.RequesterName)
if err != nil {
	// D-19 fail-closed: an unreadable chain must deny, never allow.
	return fmt.Sprintf("ancestor chain unreadable (%v): refusing to spawn", err)
}
```
Every new BIO-02 check (permission, path, cost, duplicate) must fail-closed
identically: an unreadable or ambiguous state is a deny, never a silent
allow.

**Permission check substrate — call directly, do not reimplement:**
```go
// Source: pkg/codex/permission_profile.go:57-77
func PermissionProfileForCaste(caste string) PermissionProfile {
	caste = normalizePermissionCaste(caste)
	if _, ok := repositoryReadOnlyCastes[caste]; ok {
		return PermissionProfile{ /* ... */ }
	}
	profile := PermissionProfile{ /* WorkspaceWrite default */ }
	profile.BehavioralRestrictions = behavioralRestrictionsForCaste(caste)
	return profile
}
```

**Path containment substrate — call directly, do not reimplement:**
```go
// Source: cmd/spend_session_capture.go:75-103
func validateSpendContainedPath(root, claimed, label string) (string, error) {
	// absolute-path check, filepath.Clean, symlink-resolved comparison,
	// filepath.Rel + leading-".." rejection
}
```
This exact bug class (macOS symlink resolution pre/post file creation) was
already found and fixed here — reuse verbatim per RESEARCH.md's "Don't
Hand-Roll" table.

**Budget/cost — reuse the SAME ledger, no new dial (D-12):**
```go
// Source: cmd/spawn_budget.go:1-24 (constant + doc)
const spawnTreeBudgetMax = 20 // whole-run ceiling, deliberately not configurable
```
D-12 requires recruits draw from this same team budget — call
`spawnTreeBudgetState()` / `spawnTreeBudgetReason()`, never a second counter.

---

### `cmd/recruitment_dispatch.go` + `.aether/ts-host/src/recruitment-probe.ts` (BIO-03)

**Analog:** `.aether/ts-host/src/platform-dispatcher.ts` — `spawnWorker` (lines 399-458)

**The proven depth-1 subprocess dispatch pattern (default mechanism for depth-2 too):**
```ts
// Source: .aether/ts-host/src/platform-dispatcher.ts:399-416 (read this session)
export async function spawnWorker(config: WorkerConfig): Promise<SpawnResult> {
  const binary = resolveBinaryName(config.platform);
  const args = buildArgs(config);
  const abortController = new AbortController();
  const timeoutMs = 10 * 60 * 1000; // 10 minutes
  const timeoutId = setTimeout(() => abortController.abort(), timeoutMs);

  return new Promise<SpawnResult>((resolve) => {
    const child = spawn(binary, args, {
      cwd: config.root,          // <-- the "exact workspace lease" BIO-03 wants
      signal: abortController.signal,
      env: { ...process.env },
    });
    child.stdout?.on("data", (chunk: Buffer) => { /* collect */ });
    child.stderr?.on("data", (chunk: Buffer) => { /* collect */ });
    child.on("error", (err: Error) => { /* ABORT_ERR handling on timeout */ });
    // ...
  });
}
```
Per RESEARCH.md's Pitfall 1 and Anti-Patterns: the probe must run BEFORE the
admission gate reports allow, must be cached once per session (never
per-recruitment — Pitfall 5 latency constraint), and native Task-tool
nesting is only attempted when the probe positively confirms it — subprocess
via this exact pattern is the default, not the fallback-of-last-resort.

**Configurable timeout note:** the folded todo (`2026-08-01-ts-host-preflight-hardcoded-timeout.md`)
names this file's hardcoded `10 * 60 * 1000` as the counter-example BIO-03
must retire — the probe's own timeout must be a configurable, bounded value,
not a repeat of this hardcoded pattern.

---

### `cmd/recruitment_result.go` (BIO-04)

**Analog:** `pkg/colony/lifecycle.go` — `PauseHandoff` (lines 474-499) + `LifecycleTransactionReference` / `LifecycleReceiptReference` (lines 360-373)

**The exact idempotency shape to reuse (ID + transaction + optional receipt):**
```go
// Source: pkg/colony/lifecycle.go:474-499 (read this session)
type PauseHandoff struct {
	SchemaVersion      string                        `json:"schema_version"`
	HandoffID          string                        `json:"handoff_id"`
	Command            string                        `json:"command"`
	OutcomeKind        OutcomeKind                   `json:"outcome_kind"`
	// ... AttemptID, RunID, SafeBoundary, RestartPoint, ContextDigest ...
	Transaction        LifecycleTransactionReference `json:"transaction"`
	Receipt            *LifecycleReceiptReference    `json:"receipt,omitempty"`
	Recovery           *LifecycleRecovery            `json:"recovery,omitempty"`
	Provenance         RecoveryProvenance            `json:"provenance"`
}
```
`RecruitmentResult` should carry the same `HandoffID`-shaped field (a
`RecruitmentResultID` or similar) plus its own `Transaction`/`Receipt` pair,
so a duplicate/replayed completion report matches against the stored ID and
returns the existing receipt rather than re-applying the result — exactly
STATE.md's documented pause/resume guarantee. Do not invent a second
idempotency mechanism (RESEARCH.md "Don't Hand-Roll" table, row 4).

---

### `cmd/trophallaxis.go` (BIO-05)

**Analog:** `cmd/codex_dispatch_contract.go` — `workerHandoffRecord` (lines 487-506)

**Direct structural precedent — same fields, add acknowledgement + decision-join layer:**
```go
// Source: cmd/codex_dispatch_contract.go:487-506 (read this session)
type workerHandoffRecord struct {
	ID                     string   `json:"id"`
	Workflow               string   `json:"workflow,omitempty"`
	Phase                  int      `json:"phase,omitempty"`
	Wave                   int      `json:"wave,omitempty"`
	WorkerName             string   `json:"worker_name"`
	Caste                  string   `json:"caste,omitempty"`
	TaskID                 string   `json:"task_id,omitempty"`
	Status                 string   `json:"status,omitempty"`
	Summary                string   `json:"summary,omitempty"`
	ChangedFiles           []string `json:"changed_files,omitempty"`
	CommandsRun            []string `json:"commands_run,omitempty"`
	VerificationStatus     string   `json:"verification_status,omitempty"`
	KnownFailures          []string `json:"known_failures,omitempty"`
	OpenDecisions          []string `json:"open_decisions,omitempty"`
	Assumptions            []string `json:"assumptions,omitempty"`
	NextWorkerInstructions []string `json:"next_worker_instructions,omitempty"`
	DoNotRepeat            []string `json:"do_not_repeat,omitempty"`
	Freshness              string   `json:"freshness,omitempty"`
}
```
`TrophallaxisPacket` copies this shape (child result carrier) and adds what
BIO-05 explicitly requires beyond it: an explicit acknowledgement field from
the parent (or one named follow-on consumer) and a join to the
`LifecycleDecision` it produced — a packet is not "reached" for CEC-07
purposes until acknowledged.

---

### `cmd/pheromone_resolver.go` (BIO-07)

**Analog (write side, already exact — do not add a sixth writer):** `cmd/pheromone_write.go` — `writePheromoneSignal` (lines 43-)
```go
// Source: cmd/pheromone_write.go:20-43 (read this session)
// writePheromoneSignal is the extracted body of pheromoneWriteCmd's RunE ...
// it never calls outputError/outputOK itself so non-CLI callers ... can
// write a signal without producing CLI output.
func writePheromoneSignal(sigType, content, priority, source, reason, ttl string, strength float64, tags []string) (colony.PheromoneSignal, bool, error) {
	if store == nil {
		return colony.PheromoneSignal{}, false, fmt.Errorf("no store initialized")
	}
	if sigType == "" || content == "" {
		return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 1, msg: "flags --type and --content are required"}
	}
	// type validation, priority defaulting, strength defaulting, TTL parsing ...
}
```
This is ALREADY the single write chokepoint per RESEARCH.md — three of the
five named call sites already funnel through it via
`createPheromoneSignal`/`persistPheromoneSignal`. BIO-07's actual gap is the
**read/resolve side**: `resolvePheromoneSection` (`cmd/codex_build.go:4749`)
is a naive read with no provenance or quarantine concept. `pheromone_resolver.go`
should be the ONE effective scope/strength/quarantine resolver that replaces
that naive read — same singular-chokepoint discipline applied to reads.

**Suggestion/import quarantine substrate — reuse this queue verbatim (D-07/D-10, one approval surface not two):**
```go
// Source: pkg/colony/colony.go:326-334
type PendingSuggestion struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // FOCUS, REDIRECT, or FEEDBACK
	Content     string `json:"content"`
	Reason      string `json:"reason"`
	ContentHash string `json:"content_hash"`
	CreatedAt   string `json:"created_at"`
	Dismissed   bool   `json:"dismissed"`
}
```
```go
// Source: cmd/suggest_approve.go:472-486
func filterActiveSuggestions(suggestions *[]colony.PendingSuggestion) []colony.PendingSuggestion {
	if suggestions == nil {
		return []colony.PendingSuggestion{}
	}
	var active []colony.PendingSuggestion
	for _, s := range *suggestions {
		if !s.Dismissed {
			active = append(active, s)
		}
	}
	if active == nil {
		active = []colony.PendingSuggestion{}
	}
	return active
}
```
D-07 (suggested notes) and D-10 (quarantined imports) both route through
this exact struct and its approve/dismiss list commands (`cmd/suggest_approve.go`
init() registers `--dry-run`, `--approve`, `--dismiss`, `--dismiss-all`
flags) — do not build a second "quarantine inbox."

---

### `cmd/pheromone_outcome.go` (BIO-08 / CEC-07)

**Analog (the stub to replace, not a pattern to copy):** `cmd/agency_contract.go`
```go
// Source: cmd/agency_contract.go:15
SwarmPhase202Limitation = "Typed live checkpoint, pause, and resume machinery awaits Phase 202."
```
```go
// Source: cmd/agency_contract.go:30-36
type AgencyReceiptEvidence struct {
	// ...
	ChangedDecision *colony.LifecycleDecision `json:"changed_decision,omitempty"`
	EffectEvidence  *colony.LifecycleEvidence `json:"effect_evidence,omitempty"`
}
```
```go
// Source: cmd/agency_contract.go:139-157 — the join logic already present, currently starved of real data
if (facts.ChangedDecision == nil) != (facts.EffectEvidence == nil) {
	// both-or-neither invariant already enforced
}
if facts.ChangedDecision != nil {
	decision := *facts.ChangedDecision
	if err := requireAgencyEvidence("measured effect", facts.EffectEvidence); err != nil { /* ... */ }
	if !agencyContainsID(decision.EvidenceIDs, facts.EffectEvidence.ID) { /* ... */ }
	receipt.EffectEvidence = []colony.LifecycleEvidence{*facts.EffectEvidence}
	receipt.Evidence = appendAgencyEvidence(receipt.Evidence, *facts.EffectEvidence)
	measuredEffect = fmt.Sprintf("decision %s (evidence %s)", decision.ID, facts.EffectEvidence.ID)
}
```
The shape and the both-or-neither invariant are already correct and
well-tested; the gap is purely that `facts.ChangedDecision`/`EffectEvidence`
are never populated from a real Phase-202 event today, and the stub string
at line 357 (`Limitation: SwarmPhase202Limitation`) is still assigned. This
file's job is to query the Phase 202 ceremony/event trail
(`CeremonyTopicBuildSpawn` family in `pkg/events/ceremony.go:8`) and populate
these fields with genuine evidence, then delete the stub assignment. Per
CLAUDE.md's own Pitfall 4 warning: a test must assert a real
`ChangedDecision` gets populated from a real recorded event, not merely that
the struct shape exists.

---

## Shared Patterns

### Fail-closed on every unreadable state (D-19 discipline)
**Source:** `cmd/spawn_ancestor.go:95-98`, `cmd/spawn_budget.go` doc comment (lines 68-72)
**Apply to:** Every new BIO-02/03/04 check — permission, path, cost, duplicate,
platform probe, idempotency lookup. An unreadable or ambiguous state is
always a deny/refuse, never a silent allow.

### Refusal-by-name, never a bare boolean
**Source:** `cmd/spawn.go:316-325`, `cmd/spawn_ancestor.go:100-108`
**Apply to:** Every BIO-02 deny path and D-03/D-06 refusal render — name the
caste, the reason class (exact string enum, e.g. "depth"/"budget"/"ancestor-cycle"/"permission"/"path"/"cost"/"duplicate"), and a human `Detail` sentence.

### Extend the singular chokepoint, never fork a parallel one
**Source:** `cmd/spawn.go` doc comment lines 304-306; `cmd/pheromone_write.go` doc comment lines 20-28
**Apply to:** `spawnCanSpawnDecision` (BIO-02) and `writePheromoneSignal`
(BIO-07 write side) — both are deliberately structured so new checks/callers
are ADDED to the one function, not routed around it. `pkg/agent.SpawnTree`
(BIO-06) is the same discipline applied to state: extend the existing
whole-run ledger, do not build a second tree.

### Idempotent completion via HandoffID + Transaction + Receipt
**Source:** `pkg/colony/lifecycle.go:474-499`
**Apply to:** BIO-04's `RecruitmentResult` — a replayed/duplicated completion
report must match the stored ID and return the existing receipt rather than
re-apply.

### Tick-to-approve queue, one surface for suggestions AND quarantined imports
**Source:** `pkg/colony/colony.go:326-334` (`PendingSuggestion`), `cmd/suggest_approve.go:472-486`
**Apply to:** D-07 (suggested pheromones) and D-10 (quarantined cross-project
imports) — both land in this one queue/approve/dismiss mechanism, never two
separate ceremonies.

## No Analog Found

None. RESEARCH.md's own conclusion — every BIO-01..08 primitive already has
a direct, well-tested analog in this codebase — held up under this pass; the
support checks (permission, path) also have exact analogs. The only
genuinely greenfield piece is the `RecruitmentIntent` struct's specific field
set (BIO-01), which is a superset composition of existing shapes rather than
a novel pattern.

## Metadata

**Analog search scope:** `cmd/`, `pkg/agent/`, `pkg/colony/`, `pkg/codex/`, `pkg/events/`, `.aether/ts-host/src/`
**Files scanned (read directly this session):** `cmd/spawn.go`, `cmd/spawn_budget.go`, `cmd/spawn_ancestor.go`, `pkg/colony/lifecycle.go`, `cmd/codex_dispatch_contract.go`, `cmd/pheromone_write.go`, `cmd/agency_contract.go`, `cmd/suggest_approve.go`, `pkg/colony/colony.go`, `pkg/codex/permission_profile.go`, `cmd/spend_session_capture.go`, `pkg/events/ceremony.go`, `.aether/ts-host/src/platform-dispatcher.ts`
**Pattern extraction date:** 2026-09-12
