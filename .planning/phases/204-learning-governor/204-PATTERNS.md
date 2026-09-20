# Phase 204: Learning Governor - Pattern Map

**Mapped:** 2026-09-14
**Files analyzed:** 10 (9 production files/packages + 1 synthesis doc; test analogs listed per file)
**Analogs found:** 10 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `cmd/learning_status_vocabulary.go` (LEARN-01) | utility/filter | request-response (read-side filter fix) | `cmd/colony_prime_context.go` (bug site) + `pkg/learn/colony_store.go` (`EntryFilter`/`List`) | exact — the fix is a one-argument change at an already-read call site |
| `cmd/episode_ledger.go` (LEARN-02) | service/store | event-driven + CRUD | `pkg/events/colony_live.go` (topic vocabulary) + `cmd/recruitment_credit.go` (evidence-gated ledger) | exact — extends both existing chokepoints, never a new bus/store |
| `cmd/application_evidence.go` (LEARN-03) | service | event-driven → CRUD | `cmd/instinct_application.go` (the function being extended) + `cmd/recruitment_credit.go` (the outcome source) | exact |
| `cmd/fixture_conversion.go` (LEARN-04) | service/transform | batch/transform | `cmd/midden_shared.go` + `pkg/colony/midden.go` (source), `colony.SanitizeSignalContent` (sanitizer) | role-match |
| `cmd/eval_gates.go` (LEARN-05) | config/tooling | batch | none in `cmd/` (genuinely greenfield — no existing Go build-tag tiering); nearest structural analog is `Makefile` + CLAUDE.md's "Verification Commands" discipline | no analog (see below) |
| `cmd/shadow_comparison/{evaluator,candidate,baseline}.go` (LEARN-06) | service, isolated package | transform/pure-function | `cmd/recruitment_credit.go`'s content-addressed ID pattern (`recruitmentCreditRecordID`) for the digest idiom; `cmd/classic_voice_event_test.go`'s `TestOneLiveEventModelOnly`/structural-boundary style for the "no second X" test shape | partial — vocabulary/ID idiom matches, package-isolation shape is new |
| `cmd/promotion_gate.go` (LEARN-07) | middleware/gate | request-response | `colony.PendingSuggestion` + `cmd/suggest_approve.go` (tick-to-approve queue) + CLAUDE.md's forced-reviewer/owner-waiver pattern | exact — same authority shape (named signal → mandatory gate → owner-only waiver) |
| `cmd/rollback.go` (LEARN-07) | service | transactional/event-driven | `cmd/work_repair.go` (`saveRepairCheckpoint`/`restoreRepairCheckpoint`) + `cmd/pheromone_outcome.go` (`noteHarmfulQuarantineThreshold`) | exact |
| `.github/workflows/*` (LEARN-08, CI-config subset) | config | n/a | out of scope — sandbox-refused to the build executor; owner-escalated only, no code analog needed | n/a |
| `204-CLASSIC-SYNTHESIS.md` | doc | n/a | `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md` | exact — same phase-mechanism-study genre, prior phase |

## Pattern Assignments

### `cmd/learning_status_vocabulary.go` (utility, request-response filter fix)

**Analog:** `cmd/colony_prime_context.go` (the bug site) + `pkg/learn/colony_store.go` (the filter that already exists but is unused)

**The bug this file fixes** (`cmd/colony_prime_context.go:724-735`):
```go
// Learned Memory -- durable learning entries from successful builds (D-13, D-14, D-15, HIVE-03)
learnStore := learn.NewColonyStore(store)
learnEntries, _ := learnStore.List(learn.EntryFilter{
	MinConfidence: 0.3, // filter out very low confidence
	Limit:         20,  // cap entries to prevent budget exhaustion (Pitfall 5)
})
if len(learnEntries) > 0 {
	var learnSB strings.Builder
	writeSectionHeader(&learnSB, "learned_memory", "## LEARNED MEMORY (Verified Outcomes)\n\n")
	for _, entry := range learnEntries {
		learnSB.WriteString(fmtOrFallback("learned_memory", func(t *sectionTemplate) string { return t.EntryFormat }, "- [Phase %d] %s (confidence: %.0f%%, classification: %s)\n",
			entry.Phase, entry.Content, entry.Confidence*100, entry.Classification))
	}
```
`Status` is never set on the filter, so `StatusHypothesis` entries pass through unfiltered and are rendered under a header that claims "Verified Outcomes."

**The already-built filter machinery it must call** (`pkg/learn/colony_store.go:118-146`):
```go
// List returns entries matching the given filter. Returns empty slice if none match.
func (c *ColonyStore) List(filter EntryFilter) ([]Entry, error) {
	entries, err := c.loadEntries()
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, e := range entries {
		if filter.Phase != 0 && e.Phase != filter.Phase {
			continue
		}
		if filter.Classification != "" && e.Classification != filter.Classification {
			continue
		}
		if filter.MinConfidence > 0 && e.Confidence < filter.MinConfidence {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}
	if result == nil {
		result = []Entry{}
	}
	return result, nil
}
```
**Core pattern to copy:** the filter's `if filter.Status != "" { ... }` short-circuit already exists — the new file's job is a shared helper (e.g. `learningVerifiedOutcomesFilter()`) every "(Verified Outcomes)"-labeled render path calls, passing `Status: learn.StatusValidated` (or splitting hypothesis entries into an honestly-labeled second section). Do not touch `pkg/learn/colony_store.go` itself — it is already correct; the defect is entirely at call sites like `colony_prime_context.go:725`.

**Test analog:** `cmd/instinct_application_test.go` and `cmd/recruitment_credit_test.go` both show this codebase's preferred shape for "assert on the real read path, not a hand-built fixture" — follow `newCreditTrophallaxisFixture`'s style (drive the real write path, then assert on the real read boundary) rather than constructing a `learn.Entry` literal directly. Per Pitfall 1's own warning: any test must assert `Status`, not just presence, or it cannot catch a regression here.

---

### `cmd/episode_ledger.go` (service, event-driven + CRUD)

**Analogs:** `pkg/events/colony_live.go` (topic-constant extension idiom) + `cmd/recruitment_credit.go` (evidence-gated write boundary to call, not duplicate)

**Imports/topic-vocabulary pattern to copy** (`pkg/events/colony_live.go:1-40`):
```go
package events

import "encoding/json"

// Live-colony topic vocabulary (live/v1). This is the one versioned typed
// event model every lifecycle lane -- build, continue, plan, swarm, oracle,
// recovery -- publishes through and every renderer (aether watch) reads
// through. ... Do not build a second event bus, revive the
// retired cmd/event_types.go trio, or add a topic here without also adding
// it to ColonyLiveTopics().
const (
	LiveTopicEpisodeStarted     = "live.episode.started"
	LiveTopicEpisodeEnded       = "live.episode.ended"
	...
	LiveTopicRecruitAdmitted = "live.recruit.admitted"
```
**Core pattern:** add new `LiveTopic...` constants in this exact same file/const block (e.g. `LiveTopicOutcomeRecorded`), register them in `ColonyLiveTopics()`, and emit through the existing single boundary (`cmd/live_events.go`'s `emitColonyLive`) — never a second `NewBus` call (`TestOneLiveEventModelOnly` forbids it). Per Pitfall 3, configure a distinct `events.Config` with a long/unbounded TTL for this topic family, or add a durable rollup view — do not let `DefaultTTL = 30` (days, `pkg/events/event.go:15`) silently expire outcome evidence.

**Evidence-gate boundary to call, not re-implement** (`cmd/recruitment_credit.go:184` signature and contract, full excerpt already in Pattern 1 below) — the episode ledger's terminal-result recording should call `recordRecruitmentCredit`, not write a parallel outcome file.

**Test analog:** `cmd/recruitment_credit_test.go` — see the `newCreditTrophallaxisFixture` helper (drives the real pack/acknowledge/record-decision path, not a struct literal) and the file's own `go/ast`-parsing test style (it parses `cmd/*.go` to assert `recordRecruitmentCredit` is the ONLY writer of its path — copy this "ONE function may write X" AST-enforcement shape for the episode ledger's own single-writer boundary).

---

### `cmd/application_evidence.go` (service, event-driven → CRUD)

**Analog:** `cmd/instinct_application.go` (the function being extended) + `cmd/recruitment_credit.go` (the real outcome source it must read instead of unconditionally writing `true`)

**The gap this file closes** (documented directly in RESEARCH.md Pitfall 2): `recordInstinctApplicationsForPhase` (`cmd/instinct_application.go:149-196`) records `"success": true` unconditionally for every phase that durably advanced — there is no `false`, `contradicted`, `ignored`, or `harmful` path anywhere in that file today.

**Core pattern:** extend (or add a sibling to) `recordInstinctApplicationsForPhase` to derive its outcome from `recordRecruitmentCredit`'s `helpful`/`neutral`/`harmful` vocabulary once a real `ChangedDecisionID`/`EffectEvidenceID` pair exists for that instinct's delivery, using the same `recruitmentCreditOutcome` type `cmd/recruitment_credit.go` declares. Do not add more `bool` flags to `ApplicationHistory`'s untyped `map[string]interface{}` shape — LEARN-01 wants this tightened to a typed struct, matching `pheromones.json`'s pointer-backed typed-field model (Census item 8) as the schema to imitate.

**Test analog:** `cmd/instinct_application_test.go` for the existing delivery/application assertions to extend; `cmd/recruitment_credit_test.go` for the "assert the outcome differs between a phase that helped and one that didn't" shape Pitfall 2 explicitly requires (a test that only asserts an entry *exists* proves nothing new).

---

### `cmd/fixture_conversion.go` (service/transform, batch)

**Analog:** `cmd/midden_shared.go` + `pkg/colony/midden.go` (source of confirmed incidents) + `colony.SanitizeSignalContent` (the untrusted-input discipline to reuse)

**Read-side pattern to copy** (`cmd/midden_shared.go:1-26`):
```go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const middenCanonicalPath = "midden.json"

func loadMiddenFile(s *storage.Store) (colony.MiddenFile, error) {
	var mf colony.MiddenFile
	err := s.LoadJSON(middenCanonicalPath, &mf)
	return mf, err
}
```
**Core pattern:** `fixture_conversion.go` reads via `loadMiddenFile`, filters to `Reviewed=true, Acknowledged=true` entries, runs each through `colony.SanitizeSignalContent` (the same sanitizer every other midden-fed surface already uses per STATE.md's Phase 198.1 decision log — do not write a new scrubber), deduplicates, and writes a new versioned fixture-bank file tagged with provenance back to the originating `MiddenEntry.ID`. Mirror `appendMiddenEntry`'s `UpdateJSONAtomically`-based append discipline (shown in the same file, lines 29+) for the fixture bank's own writer, rather than a separate Load-then-Save pair.

**Test analog:** existing midden reader tests (grep `cmd/*midden*_test.go`) for the read-and-filter shape; `cmd/recruitment_credit_test.go`'s content-addressed-ID/dedup assertion style for "the same incident fed twice produces one fixture, not two."

---

### `cmd/eval_gates.go` (config/tooling, batch)

**No close analog exists in this codebase** — confirmed by RESEARCH.md: zero existing Go build tags anywhere in `cmd/` or `pkg/`. Do not build a second test runner or task orchestrator (Pitfall 4). Use Go's own `-tags`/`-run` scoping against the existing 5,317-test corpus; define named gates (fast/focused/integration/provider/overnight/race/release) as `go test` invocations in `Makefile`/CI target definitions, each verified for `discovered == executed` per CLAUDE.md's own Verification Commands discipline. Planner should treat this as genuinely new tooling work, not a pattern-copy task.

---

### `cmd/shadow_comparison/{evaluator,candidate,baseline}.go` (isolated package, transform)

**Analog (ID/digest idiom):** `cmd/recruitment_credit.go`'s deterministic content-addressed record ID pattern (`recruitmentCreditRecordID`, used to make replay idempotent) — the same idiom RESEARCH.md's Pattern 4 explicitly names as the precedent for `FrozenEvaluator`'s `sha256.Sum256` digest.

**Analog (structural "cannot be bypassed" test shape):** `cmd/classic_voice_event_test.go` — its `TestLifecycleEventSentenceTypeCannotBeBypassed`, `TestOneLiveEventModelOnly`, and (per the phase-context brief) `TestOneAdmissionAuthority` are this codebase's proven pattern for proving a boundary structurally, not just behaviorally.

**Proposed shape (from RESEARCH.md, not yet in the codebase — follow this exactly):**
```go
// Proposed shape for shadow_comparison/evaluator.go
type FrozenEvaluator struct {
	digest [32]byte // sha256, set once at NewFrozenEvaluator, never exported
	run    func(candidate, task any) EvalResult
}

func NewFrozenEvaluator(def []byte, run func(any, any) EvalResult) FrozenEvaluator {
	return FrozenEvaluator{digest: sha256.Sum256(def), run: run}
}
// No SetDigest, no SetRun -- the type has no mutator.
```
**Core pattern:** unexported struct fields, assigned once at construction, with zero mutator methods — the same "no setter exists" discipline this codebase already uses structurally (see `cmd/pheromone_write.go`'s single-chokepoint-writer pattern for pheromones). Package boundary (`shadow_comparison/`) must be genuinely isolated: no symbol re-exported that would let a candidate reach the evaluator's internal `run` field.

**Test analog:** name a `TestCandidateCannotAlterItsEvaluatorDigest` test following `cmd/classic_voice_event_test.go`'s "attempt every code path that could reach X, assert X unchanged before/after" shape.

---

### `cmd/promotion_gate.go` (middleware/gate, request-response)

**Analog:** `colony.PendingSuggestion` + `cmd/suggest_approve.go` (the reusable tick-to-approve queue)

**Pattern to copy** (`cmd/suggest_approve.go:1-24`):
```go
package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// suggestApproveCmd is the ONE tick-to-approve surface for everything that
// needs the owner's decision before it can take effect: a runtime-proposed
// pheromone suggestion (from suggest-analyze) and a cross-project import
// held quarantined on arrival both wait in the same queue and are cleared
// by this same command (D-07/D-10). There is deliberately no second
// approval command -- TestOneApprovalSurface fails by name if one appears.
var suggestApproveCmd = &cobra.Command{
	Use:   "suggest-approve",
	Short: "Review, edit, and approve pheromone suggestions and quarantined imports",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		approveID, _ := cmd.Flags().GetString("approve")
		editText, _ := cmd.Flags().GetString("edit")
		dismissID, _ := cmd.Flags().GetString("dismiss")
		dismissAll, _ := cmd.Flags().GetBool("dismiss-all")
		...
```
**Core pattern:** extend `colony.PendingSuggestion`'s queue shape (list/approve/dismiss, content-hash dedup) for canary-candidate approval rather than inventing a new approval store — `promotion_gate.go` should add a candidate kind to the SAME queue `suggest_approve.go` already reads, not a competing command (mirrors this file's own explicit "no second approval command" rule, `TestOneApprovalSurface`). For the authority-refusal list (preferences/skills/workflows/source/security/deletion/permission/verification/external-action), copy CLAUDE.md's documented forced-reviewer shape: named signal → mandatory gate → owner-only waiver, scoped to one signal at a time, never inferred from worker count or wording — see `TestReviewerForcedOnlyByNamedRisk`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestAutopilotNeverWaives` as the locking tests to mirror by name for the new gate.

**Test analog:** `cmd/suggest_approve_test.go` for the queue-approval assertion shape; the review-gate tests named in CLAUDE.md (`TestReviewerForcedOnlyByNamedRisk`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestAutopilotNeverWaives`) for the authority-refusal shape — find their current file via `grep -rn "func TestReviewerForcedOnlyByNamedRisk" cmd/` at implementation time.

---

### `cmd/rollback.go` (service, transactional/event-driven)

**Analogs:** `cmd/work_repair.go` (checkpoint/restore pattern) + `cmd/pheromone_outcome.go` (quarantine-threshold pattern to generalize, not duplicate)

**Checkpoint/restore functions to follow the shape of** (`cmd/work_repair.go`):
```go
func saveRepairCheckpoint(root, checkpointID string, paths []string) (repairCheckpoint, error)
func restoreRepairCheckpoint(checkpoint repairCheckpoint) error
func repairCheckpointDirectoryDigest(root string, paths []string) (string, error)
func repairCheckpointIdentity(phase int, check string) string
```
**Quarantine-threshold pattern to generalize** (`cmd/pheromone_outcome.go:1-49`):
```go
// noteStrengthHelpfulStep/-Neutral-/-Harmful- are the declared amounts
// tuneNoteStrengthFromOutcomes moves a note's strength by, one movement per
// credit record...
const (
	noteStrengthHelpfulStep = 0.10
	noteStrengthNeutralStep = -0.02
	noteStrengthHarmfulStep = -0.15
)
const (
	noteStrengthFloor   = 0.0
	noteStrengthCeiling = 1.0
)
// noteHarmfulQuarantineThreshold is the number of harmful credit records a
// single note must accumulate before tuneNoteStrengthFromOutcomes
// automatically quarantines it -- the same threshold-count shape
// middenAutoRedirectThreshold already uses...
const noteHarmfulQuarantineThreshold = 3
```
**Core pattern:** `rollback.go` should call `saveRepairCheckpoint` before any canary mutation, run verification, and call `restoreRepairCheckpoint` on failure — exactly Swarm's proven save→verify→rollback shape — and generalize `noteHarmfulQuarantineThreshold`'s named-constant threshold-count idiom (never an inline literal) for "N regressions against the hidden holdout triggers automatic rollback + quarantine." Also reuse `pkg/colony/lifecycle.go`'s `PauseHandoff` idempotent transaction/receipt pattern (handoff ID as idempotency key) for replay-safety of the rollback action itself.

**Test analog:** `cmd/work_repair_test.go` and `cmd/work_repair_handback_test.go` for checkpoint/restore assertions; `cmd/pheromone_outcome_test.go` for threshold-triggered-quarantine assertions.

---

### `204-CLASSIC-SYNTHESIS.md` (doc)

**Analog:** `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md` — same genre (reconstruct a Classic-era mechanism, compare against every current store/reader, name a disposition per store using the synthesis template's vocabulary: keep/retire-with-proof/etc.). RESEARCH.md's own Open Questions #1 flags that the Classic-side reconstruction (Queen memory, Hive, midden, signal reinforcement, learning presentation) was not exhaustively performed in the research pass — this is the author's first required task, using the one direct citation already obtained (`3a5b81c2:.aether/learning.md`) as a starting point, not a complete source.

---

## Shared Patterns

### Evidence-gated, replay-safe write boundary
**Source:** `cmd/recruitment_credit.go:184-280` (`recordRecruitmentCredit`)
**Apply to:** `episode_ledger.go`, `application_evidence.go`, and any LEARN-02/03/07 code that records an outcome. Never build a parallel store — call this function (or its direct successor) with real `ChangedDecisionID`/`EffectEvidenceID` values. The AST-based "only ONE function may write this path" test style in `cmd/recruitment_credit_test.go` should be copied for any new single-writer boundary these files introduce.

### One event bus, extended by topic constants only
**Source:** `pkg/events/colony_live.go:1-40`, enforced by `TestOneLiveEventModelOnly`
**Apply to:** `episode_ledger.go`. New topics go in this file's `const` block and `ColonyLiveTopics()`; never a second `NewBus` call or a second JSONL file for events.

### Named-constant threshold, never inline literal
**Source:** `cmd/pheromone_outcome.go`'s `noteHarmfulQuarantineThreshold` / `middenAutoRedirectThreshold` (`cmd/phase_end_signals.go`)
**Apply to:** `rollback.go`'s regression/quarantine trigger count, `fixture_conversion.go`'s dedup/confirmation thresholds if any.

### Owner-only waiver, named signal, scoped
**Source:** CLAUDE.md "Team Check-In and Owner Decisions"; codified by `TestReviewerForcedOnlyByNamedRisk`, `TestOnlyTheOwnerCanWaiveAForcedReviewer`, `TestAutopilotNeverWaives`, `TestWaiverCoversOneSignalOnOnePhase`
**Apply to:** `promotion_gate.go`'s authority-retention refusal list (preferences/skills/workflows/source/security/deletion/permission/verification/external-action).

### Untrusted-input sanitization before storage
**Source:** `colony.SanitizeSignalContent` (referenced from every midden writer, per STATE.md Phase 198.1 decision log)
**Apply to:** `fixture_conversion.go` (worker/repo-authored midden content becoming a stored, later-injected fixture).

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `cmd/eval_gates.go` | config/tooling | batch | Zero existing Go build-tag/test-tiering usage anywhere in this repo (verified: no `go:build integration`/`race` tags found). Genuinely new tooling work; build on Go's own `-tags`/`-run` mechanism and `Makefile`/CI targets, not a copied Go source pattern. |
| `.github/workflows/*` changes (LEARN-08) | config | n/a | Sandbox-refused to the build executor in this environment (`WINDOWS.md:43`); this is an owner-escalation path, not a pattern-copy task. |

## Metadata

**Analog search scope:** `cmd/`, `pkg/events/`, `pkg/learn/`, `pkg/colony/`, prior-phase `.planning/phases/203-biological-runtime/`
**Files scanned:** `cmd/recruitment_credit.go`+test, `cmd/pheromone_outcome.go`+test, `cmd/instinct_application.go`+test, `cmd/learning_cmds.go`+test, `cmd/colony_prime_context.go`, `pkg/learn/colony_store.go`, `pkg/events/colony_live.go`, `cmd/suggest_approve.go`+test, `cmd/swarm_cmd.go`, `cmd/work_repair.go`+tests, `cmd/midden_shared.go`, `pkg/colony/midden.go`, `cmd/classic_voice_event_test.go`, `.planning/phases/203-biological-runtime/203-CLASSIC-SYNTHESIS.md`
**Pattern extraction date:** 2026-09-14
