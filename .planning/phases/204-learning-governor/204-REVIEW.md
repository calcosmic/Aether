---
phase: 204-learning-governor
reviewed: 2026-09-14T00:00:00Z
depth: standard
files_reviewed: 39
files_reviewed_list:
  - cmd/application_evidence.go
  - cmd/autopilot_lessons.go
  - cmd/codex_continue_finalize.go
  - cmd/colony_prime_context.go
  - cmd/consolidation_lifecycle.go
  - cmd/episode_ledger.go
  - cmd/eval_gates.go
  - cmd/fixture_conversion.go
  - cmd/improvement_report.go
  - cmd/instinct_application.go
  - cmd/instinct.go
  - cmd/internal_cmds.go
  - cmd/learning_status_vocabulary.go
  - cmd/live_events.go
  - cmd/memory_feed.go
  - cmd/memory_schema.go
  - cmd/midden_shared.go
  - cmd/pheromone_approval.go
  - cmd/promotion_gate.go
  - cmd/recruitment_credit.go
  - cmd/rollback.go
  - cmd/shadow_cmds.go
  - cmd/source_proposal.go
  - cmd/suggest_approve.go
  - Makefile
  - pkg/colony/colony.go
  - pkg/colony/instincts.go
  - pkg/colony/memory_schema.go
  - pkg/colony/midden.go
  - pkg/events/colony_live.go
  - pkg/learn/colony_store.go
  - pkg/learn/difficulty.go
  - pkg/learn/learn.go
  - pkg/memory/consolidate.go
  - pkg/memory/instinct_stats.go
  - pkg/memory/promote.go
  - pkg/shadow/baseline.go
  - pkg/shadow/candidate.go
  - pkg/shadow/comparison.go
  - pkg/shadow/evaluator.go
findings:
  critical: 2
  warning: 3
  info: 0
  total: 5
status: issues_found
---

# Phase 204: Code Review Report

**Reviewed:** 2026-09-14T00:00:00Z
**Depth:** standard
**Files Reviewed:** 39
**Status:** issues_found

## Summary

Phase 204 ("Learning Governor") adds a large body of new infrastructure: an
evidence-gated credit ledger, a guidance-application state machine, a durable
episode ledger, seven named eval gates, a regression fixture bank, an
improvement report with deliberately-uncombinable figures, a structurally
isolated shadow-evaluation package (`pkg/shadow`), a promotion gate over it,
a canary start/rollback mechanism, and a propose-only source-improvement
boundary. The code is unusually well-documented, follows this repository's
own completeness conventions (closed vocabularies with names()/declared()
helpers) consistently, and several of the riskiest surfaces (the shadow
package's structural isolation, the promotion gate's retained-authority
split, the source-proposal's merge/push/publish/deploy prohibition) are
deliberately over-engineered against exactly the failure modes CLAUDE.md
warns about.

Two genuine defects were found that meet the BLOCKER bar: a path-traversal
gap in the new source-proposal writer, and a missing status guard in the
canary rollback path that lets an already-completed (kept) change be
silently reverted. Neither is wired to a live command yet (`shadow-declare`,
`shadow-compare`, and the canary/source-proposal functions have no Cobra
command or call site in this diff), which bounds today's blast radius, but
both are real defects in code that ships in this commit and will fire the
moment a caller is added — they should be fixed now rather than carried
forward as a "future wiring" problem. Three further issues meet the WARNING
bar: a lossy boolean-to-outcome mapping in `instinct-apply`, a missing
double-close guard in the new episode ledger, and a credit-ledger cross
product that inflates records and owner-visible output beyond what the
underlying evidence supports.

## Critical Issues

### CR-01: Path traversal in the source-proposal file writer

**File:** `cmd/source_proposal.go:239-253`
**Issue:** `sourceProposalApplyChangeSet` writes every file in a `sourceChangeSet` to `filepath.Join(root, relPath)` with no validation that `relPath` stays inside `root`. `relPath` comes directly from `sourceChangeSetFile.Path`, which is exactly the kind of value CLAUDE.md's Definition of Done treats as untrusted input ("Anything a worker or a wrapper can influence: parsed output, submitted packets, reported evidence. Treat it as untrusted input."). A change set whose path contains `..` segments (e.g. `"../../../.git/hooks/pre-commit"` or a deep-enough `"../../../../etc/cron.d/x"`) resolves, after `filepath.Join`'s own `Clean`, to a location outside the repository entirely, and `os.MkdirAll` + `os.WriteFile` will create it. This is not merely a theoretical concern here — the whole point of `proposeSourceImprovement` (LEARN-08) is to let this system, or a future automated candidate, hand it a change set describing what to write; the function that is supposed to be "propose-only" and structurally incapable of merge/push/publish/deploy has no equivalent structural guard against writing outside the one directory it is allowed to touch.
**Fix:**
```go
func sourceProposalApplyChangeSet(root string, changes sourceChangeSet) error {
    if len(changes.Files) == 0 {
        return fmt.Errorf("source proposal change set carries no files")
    }
    absRoot, err := filepath.Abs(root)
    if err != nil {
        return fmt.Errorf("resolve proposal root: %w", err)
    }
    paths := make([]string, 0, len(changes.Files))
    for _, f := range changes.Files {
        relPath := strings.TrimSpace(f.Path)
        if relPath == "" {
            return fmt.Errorf("source proposal change set carries a file with an empty path")
        }
        if filepath.IsAbs(relPath) {
            return fmt.Errorf("source proposal change set carries an absolute path %q -- refused", relPath)
        }
        full := filepath.Join(absRoot, relPath)
        // filepath.Join already Cleans; the resulting path must still be
        // inside absRoot, otherwise a "../" segment escaped it.
        if full != absRoot && !strings.HasPrefix(full, absRoot+string(filepath.Separator)) {
            return fmt.Errorf("source proposal change set path %q escapes the proposal root -- refused", relPath)
        }
        if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
            return fmt.Errorf("create directory for %s: %w", relPath, err)
        }
        if err := os.WriteFile(full, []byte(f.Content), 0o644); err != nil {
            return fmt.Errorf("write %s: %w", relPath, err)
        }
        paths = append(paths, relPath)
    }
    // ... unchanged from here
}
```

### CR-02: `rollbackCanary` can revert an already-completed canary, silently discarding a change already reported as kept

**File:** `cmd/rollback.go:317-406` (and `cmd/rollback.go:257-302` for `completeCanary`)
**Issue:** `rollbackCanary` guards against being called twice for the same candidate (via `loadCanaryRollbackReceipt`), but it never checks whether the run has already reached `canaryRunStatusCompleted`. `completeCanary` sets `Status = "completed"`, writes a durable `episode_closed` record with `TerminalResult: "completed"`, and calls `emitCanaryCompleted`, which tells the owner in plain English: *"Proposal %q ... finished its trial run and kept passing -- the change stays."* If `rollbackCanary` is subsequently invoked for that same candidate (e.g. a delayed/duplicate hard-gate re-evaluation, a retried recovery pass, or simply a caller bug elsewhere in the program that has not yet been written), it does not consult `run.Status` at all: it restores the pre-canary checkpoint (undoing the very change that was just told to the owner as "kept"), overwrites `Status` to `"rolled_back"`, and quarantines the candidate — with no error, no refusal, and a durable episode record that now reports `"rolled_back"` for something the ledger, two paragraphs earlier, already recorded as `"completed"`. This is exactly the class of defect CLAUDE.md calls out as full-rigour territory ("Anything that decides work is complete, credited, verified, or advanced" and "touching ... deletion"): a completed/kept change is data the owner already relied on, and this path can silently reverse it. There is no test in `cmd/rollback_test.go` covering "rollback after complete" (only `TestCompleteCanaryMarksTheChangeAsKept` and rollback-replay/idempotency tests exist).
**Fix:**
```go
func rollbackCanary(run canaryRun, reason string) (canaryRollbackReceipt, bool, error) {
    ...
    candidateID := strings.TrimSpace(run.CandidateID)
    if candidateID == "" {
        return canaryRollbackReceipt{}, false, fmt.Errorf("canary run carries no candidate id")
    }

    if existing, found, err := loadCanaryRollbackReceipt(candidateID); err != nil {
        return canaryRollbackReceipt{}, false, err
    } else if found {
        return existing, false, nil
    }

    // Re-read the current stored status rather than trusting the caller's
    // own `run` argument, which may be stale by the time this is called.
    current, found, err := loadCanaryRun(candidateID)
    if err != nil {
        return canaryRollbackReceipt{}, false, err
    }
    if !found {
        return canaryRollbackReceipt{}, false, fmt.Errorf("no canary run recorded for candidate %q", candidateID)
    }
    if current.Status == canaryRunStatusCompleted {
        return canaryRollbackReceipt{}, false, fmt.Errorf(
            "canary %q already completed and kept -- rollback refuses to undo a change already reported as kept",
            candidateID,
        )
    }
    ...
}
```

## Warnings

### WR-01: `instinct-apply --success` collapses "did not help" into "harmful"

**File:** `cmd/internal_cmds.go:496-501`
**Issue:** The manual owner-invoked grading path now maps the boolean `--success` flag directly onto the credit ledger's three-way outcome vocabulary: `true` → `helpful`, `false` → `harmful`. There is no way to record `neutral` through this command. An owner who ran `instinct-apply <id> --success=false` to mean "this guidance didn't help, but it didn't actively cause harm either" now produces the same signal as "this guidance actively made things worse" — both increment `Failures`/`HarmfulApplications` in `SummarizeInstinctApplications` (`pkg/memory/instinct_stats.go`), which feeds `InstinctNeedsReview`, `InstinctUsefulnessScore`, decay in `ConsolidationService.Run`, and the new `queenPromotionHelpfulFloor` eligibility check. A single ambiguous flag now silently biases every one of those downstream signals toward "harmful" whenever it should have been "neutral".
**Fix:** Either add a third flag value (e.g. `--outcome helpful|neutral|harmful` replacing `--success`, with `--success` kept as a deprecated alias defaulting to `helpful`/`neutral` rather than `harmful`), or map `--success=false` to `neutral` instead of `harmful` and require an explicit `--harmful` flag for the stronger claim.

### WR-02: Episode ledger has no guard against a second `episode_closed` record for the same episode

**File:** `cmd/episode_ledger.go:182-236` (see also `episodeLedgerTerminalRecord`, `cmd/episode_ledger.go:302-313`)
**Issue:** `recordEpisodeOutcome` refuses a `episode_closed` write when no `episode_opened` record exists for the episode, but it never checks whether an `episode_closed` record *already* exists. `episodeLedgerDigestPayload` only clears timestamp fields for opened/closed kinds — `Usage`, `ReportedCostUSD`, `TerminalResult`, `EvidenceIDs`, `HardGateResults` and `ChangedDecisionIDs` are all part of the replay-identity digest. So a caller that closes the same episode twice with a different `TerminalResult` (e.g. `"completed"` then, moments later on a retried finalize path, `"failed"`) does not get deduplicated as a replay — it appends a *second*, distinct `episode_closed` record. `episodeLedgerTerminalRecord` then returns the first match found while walking `records` in `sortEpisodeLedgerRecords`'s ascending-by-timestamp order, i.e. the *earliest* closed record — silently discarding whichever close happened later, even if the later one is the more authoritative result. Both `buildImprovementReport` (via `isVerifiedUsefulSuccess`/`assembleHardFailureList`) and the canary/shadow-compare flows read through this same lookup, so a double-close in any of them would silently report a stale outcome rather than erroring or reconciling.
**Fix:** Add an explicit refusal (or an idempotent "supersede" path with its own audit trail) when a `episode_closed` record already exists for `episodeID` at the top of the `RecordKind == episodeLedgerRecordKindClosed` branch in `recordEpisodeOutcome`, mirroring the existing "no open record" refusal:
```go
if record.RecordKind == episodeLedgerRecordKindClosed {
    hasOpen := false
    alreadyClosed := false
    for _, existing := range file.Entries {
        if existing.EpisodeID != episodeID {
            continue
        }
        if existing.RecordKind == episodeLedgerRecordKindOpened {
            hasOpen = true
        }
        if existing.RecordKind == episodeLedgerRecordKindClosed {
            alreadyClosed = true
        }
    }
    if !hasOpen {
        return fmt.Errorf("episode ledger refuses to close episode %q: no open record exists for it", episodeID)
    }
    if alreadyClosed {
        return fmt.Errorf("episode ledger refuses a second close for episode %q: it is already closed", episodeID)
    }
}
```

### WR-03: Credit ledger crosses every delivered instinct against every decision delta, inflating records and owner-visible output

**File:** `cmd/application_evidence.go:174-198` (`recordPhaseApplicationCredit`)
**Issue:** For every contribution (delivered instinct) on the phase, the function calls `recordRecruitmentCredit` once per decision-kind knowledge delta on the phase's latest build attempt (`for _, contributionID := range contributions { for _, decisionID := range decisionIDs { ... } }`). A phase with, say, 4 delivered instincts and 5 recorded decisions produces 20 credit records, all sharing the identical outcome (`phaseApplicationEffectAndOutcome` is computed once per phase, not per decision) — none of which establishes that a *particular* instinct actually influenced a *particular* decision, only that both happened to occur on the same phase. This is a documented simplification (the function's own doc comment acknowledges credit is derived from co-occurrence, not causal attribution), but two side effects are worth flagging as quality issues rather than accepted design: (1) `recordRecruitmentCredit` fires `emitInlineDecisionChangedLine` for every *new* record, so the owner can see the same contribution ID announced as "decision changed" multiple times in one phase-end pass, once per unrelated decision; (2) the ledger itself grows O(instincts × decisions) per phase rather than O(instincts), which will make `recruitmentCreditAll`/`recruitmentCreditForDecision` increasingly expensive to read and increasingly noisy to audit as a phase's own decision count grows.
**Fix:** Consider tightening attribution before writing a credit record — e.g. only crediting a contribution against a decision whose own summary text contains the contribution's action text (the same "text actually present" signal `corroborateGuidanceClaim` already uses a few functions away in the same file), and/or deduplicating the inline announcement per contribution per phase-end pass rather than per (contribution, decision) pair.

---

_Reviewed: 2026-09-14T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
