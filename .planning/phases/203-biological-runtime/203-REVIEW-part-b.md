---
phase: 203-biological-runtime
part: b
reviewed: 2026-09-13T21:28:43Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - cmd/pheromone_resolver.go
  - cmd/pheromone_write.go
  - cmd/pheromone_loader.go
  - cmd/pheromone_mgmt.go
  - cmd/pheromone_approval.go
  - cmd/pheromone_influence.go
  - cmd/pheromone_outcome.go
  - cmd/signal_housekeeping.go
  - cmd/suggest_approve.go
  - cmd/exchange.go
  - cmd/immune.go
  - cmd/phase_end_signals.go
  - cmd/trophallaxis.go
  - pkg/colony/pheromones.go
  - pkg/colony/colony.go
  - cmd/pheromone_resolver_test.go
  - cmd/pheromone_approval_test.go
  - cmd/pheromone_influence_test.go
  - cmd/pheromone_outcome_test.go
  - cmd/suggest_approve_test.go
  - cmd/exchange_import_sanitize_test.go
  - cmd/trophallaxis_test.go
status: issues
critical: 2
warning: 4
info: 2
---

# Phase 203 (Part B): Code Review Report — Pheromone bus, influence history, trophallaxis, credit

**Reviewed:** 2026-09-13T21:28:43Z
**Depth:** standard
**Files Reviewed:** 15 source files + 7 matching test files
**Status:** issues_found

## Summary

BIO-07's "one sanitizer/normalizer/writer/resolver" claim holds up well:
`resolveEffectivePheromones` genuinely is the one active/expiry/strength
predicate, `writePheromoneSignal` genuinely is the one create/reinforce
writer, `clearSignalQuarantine` genuinely is the one quarantine-release path,
and the import path (`cmd/exchange.go`) does unconditionally stamp
`Provenance=import` + `Quarantined=true` before anything is persisted.

BIO-08's history claim does not hold up. The eight-action, append-only,
recorded-actor history is real for five of the eight actions
(reinforce/defer/expire/revoke/appeal — plus weaken/pin/unpin from 203-13),
but the three actions that predate it — **accept, edit, reject** — never
write to `pheromones-history.json` at all, have no actor recorded anywhere,
and remain a single mutable scalar field on the queued item. The codebase's
own comment on `PendingSuggestion.Action` documents this as the state before
extension ("not that history itself"), and the extension never happened. The
test that is supposed to prove every declared action is reachable
(`TestEveryDeclaredActionIsReachable`) only checks that a named function and
CLI flag exist — it never calls the function and reads the history back, so
it cannot fail on this gap. This is the project's own named failure mode #2
(a test built in a shape that cannot fail).

A second boundary problem: the "owner-only" gate on revoke/appeal/pin/unpin
is enforced purely by trusting a self-reported `--actor` CLI flag that
defaults to `"owner"` when omitted. Nothing authenticates the calling
process. Any script or worker capable of invoking the `aether` binary — which
is most of them, by this project's own architecture — can revoke a REDIRECT
hard constraint or pin/unpin a note's tuning simply by running the command
with no `--actor` flag (or an explicit `--actor owner`), defeating the
owner-only boundary CLAUDE.md documents as a must-have truth for this plan.

Outcome-weighted tuning (`tuneNoteStrengthFromOutcomes`) is correctly wired
into both continue lanes (confirmed by call-site grep, not just doc claims),
correctly skips pinned/revoked notes, and correctly quarantines after three
harmful records. It has a real, if narrow-window, idempotency bug: a
partial failure (signal strength saved, but the paired history-append call
then fails) is not marked processed and not rolled back, so a retry
re-applies the same step a second time.

Two further findings on the exchange/trophallaxis surfaces, plus two Info
items, are below.

## Critical Issues

### CR-01: BIO-08's append-only history is not applied to accept/edit/reject — no actor recorded, no history entry, ever

**File:** `cmd/pheromone_approval.go:209-215, 226-283, 291-318, 320-342`
**Issue:**
`stampPendingNoteAction` (the function `approvePendingNote`/`rejectPendingNote`
call) only sets `item.Dismissed`, `item.Action`, and `item.ActionAt` on the
queued `colony.PendingSuggestion` — it never calls
`appendInfluenceHistory` (`cmd/pheromone_influence.go:262`), the one function
the codebase itself declares as "the ONE exported mutation on
pheromones-history.json." `editPendingNote` does the same: it rewrites
`item.Content`/`ContentHash`/`Action` in place and never appends to history
either.

`pkg/colony/colony.go:347-353` documents this directly: *"Action and ActionAt
record the owner's decision... This is deliberately a single scalar record,
not a list: it is the first entry plan 203-11 extends into a full immutable
action history (BIO-08's remaining verbs), not that history itself."* Plan
203-11 built the five new actions' history plumbing (reinforce/defer/
expire/revoke/appeal) but never went back and wired accept/edit/reject into
it. Three consequences:

1. **Not append-only.** `item.Action` is a single field that a later action
   (e.g. edit-then-approve) silently overwrites. There is no record of the
   fact that a note was edited before it was approved.
2. **No actor recorded at all.** `PendingSuggestion` has no actor field, and
   `stampPendingNoteAction`/`editPendingNote`/`rejectPendingNote` take no
   actor parameter — the review task's own requirement ("a recorded actor")
   is unmet for exactly these three verbs.
3. **False certificate.** `pheromoneInfluenceActionSurface` (`cmd/pheromone_influence.go:128-140`)
   declares `accepted`/`edited`/`rejected` as reachable via
   `approvePendingNote`/`editPendingNote`/`rejectPendingNote`, and
   `TestEveryDeclaredActionIsReachable` (`cmd/pheromone_influence_test.go:654`)
   is the test guarding this — but it only checks that the named function and
   CLI flag *exist*, never that calling the function actually produces a
   history entry. Every test in `cmd/pheromone_approval_test.go` and
   `cmd/suggest_approve_test.go` that calls `approvePendingNote`/
   `editPendingNote`/`rejectPendingNote` never once calls
   `readInfluenceHistory` afterward (confirmed by grep — zero occurrences).
   The claim "suggested notes support accept, edit, reject... on an
   APPEND-ONLY history with a recorded actor" is provably false for three of
   the eight declared verbs, and no test can catch a regression or
   confirm the gap.

**Failure scenario:** An owner reviews a suggested REDIRECT, edits its
wording, and approves it. `readInfluenceHistory(queuedItemID)` returns an
empty slice — there is no record that an edit happened, what the original
wording was, or who (which actor) approved it. If the same note is later
revoked, the revoke entry is the *first and only* entry in its history, even
though the note went through edit and approval first.

**Fix:** Have `stampPendingNoteAction` (or its three callers) accept an
`actorKind`/`actorName` pair and call `appendInfluenceHistory(item.ID,
pheromoneActionAccepted|Edited|Rejected, actorKind, actorName, before, after,
reason)` on every accept/edit/reject, mirroring the five newer actions. Then
extend `TestEveryDeclaredActionIsReachable` (or a sibling test) to actually
invoke each surfaced action and assert a matching history entry landed —
closing the "reachable but unverified" gap the current test leaves open.

---

### CR-02: Owner-only pheromone actions are gated by a self-reported CLI flag with no caller authentication

**File:** `cmd/pheromone_mgmt.go:402-441` (default `actor := pheromoneActorOwner`), `cmd/pheromone_influence.go:358-375` (`pheromoneInfluenceActorAllowed`)
**Issue:** `revokeNote`, `appealNote`, `pinNote`, and `unpinNote` are
documented as owner-only — "the runtime cannot pin or unpin any more than it
can revoke or appeal" — and `pheromoneInfluenceActorAllowed` does correctly
refuse a call that declares `actorKind != pheromoneActorOwner`. But the only
place these functions are ever invoked in production is
`runPheromoneInfluenceFlags` in `cmd/pheromone_mgmt.go`, which computes the
actor as:
```go
actor := pheromoneActorOwner
if strings.TrimSpace(actorFlag) != "" {
    actor = actorFlag
}
```
i.e. it defaults to `"owner"` whenever the caller does not pass `--actor`,
and otherwise trusts whatever string the caller *did* pass. There is no
mechanism anywhere in this call path that verifies the process invoking
`aether pheromones --revoke <id>` is actually the human owner rather than an
autonomous worker, a wrapper script, or any other process capable of
shelling out to the `aether` binary — which, per this project's own
architecture, includes ordinary build/continue workers with Bash access.

**Failure scenario:** A worker (or a compromised/adversarially-steered
worker, or simply a buggy wrapper script) runs
`aether pheromones --revoke <redirect-note-id>` with no `--actor` flag. The
call succeeds: `actor` defaults to `"owner"`, `pheromoneInfluenceActorAllowed`
sees a declared-owner actor and allows it, and a REDIRECT hard constraint —
explicitly documented elsewhere in this codebase as something "nothing in
this runtime ever clears... back to nil" except a genuine owner action — is
permanently revoked by a non-owner process. The same applies to `--pin`/
`--unpin`, which can silently exempt a note from (or re-expose it to)
automatic outcome-weighted tuning.

This is a materially worse version of the already-tracked "recruitment depth
check trusts a coordinator name on its own word" gap (documented in
`.planning/WINDOWS.md` and CLAUDE.md's Biological Runtime section) — that gap
is at least written down as an accepted, tracked limitation. This one is not
tracked anywhere.

**Fix:** At minimum, require an explicit, out-of-band signal that the caller
is an interactive/owner session (e.g. a session token, an environment
variable set only by the genuine interactive wrapper, or refusing to default
to `"owner"` at all and instead requiring `--actor` to be passed explicitly
with some corroborating evidence) before treating a CLI invocation as
owner-authorized for revoke/appeal/pin/unpin. Short of a real authentication
mechanism, at least record this as a known, tracked limitation the way the
recruitment depth-check gap already is, so it isn't silently assumed solved.

## Warnings

### WR-01: Outcome tuning can double-apply a strength step on partial failure

**File:** `cmd/pheromone_outcome.go:302-323`
**Issue:** In `tuneNoteStrengthFromOutcomes`, the per-record loop does:
```go
pf.Signals[idx].Strength = &newStrength
if err := pheromoneInfluenceSaveSignals(pf); err != nil {
    stateErr = err
    continue // correctly not marked processed
}
...
if _, herr := appendInfluenceHistory(...); herr != nil {
    stateErr = herr
    continue // NOT marked processed -- but the strength write above already succeeded
}
newlyProcessed[rec.RecordID] = true
```
If `pheromoneInfluenceSaveSignals` succeeds (the stepped strength is already
persisted to `pheromones.json`) but the subsequent `appendInfluenceHistory`
call fails, the record is left off `newlyProcessed`/`ProcessedRecordIDs`. The
very next run of this pass (next `/ant-continue`) will process the same
credit record again, re-reading the *already-stepped* `oldStrength` and
applying `step` a second time — silently double-counting one credit record's
effect on the note's strength, contradicting the file's own stated
guarantee: "so a second run over the same records changes nothing."

**Failure scenario:** A disk hiccup or lock contention causes
`appendInfluenceHistory`'s `store.UpdateJSONAtomically` call to fail exactly
once, right after the paired `pheromoneInfluenceSaveSignals` succeeded. The
next check run reapplies the same helpful/harmful step to the note,
compounding the effect of one recorded outcome into two.

**Fix:** Either persist the strength change and the history append inside a
single atomic transaction (so both succeed or neither does), or mark the
record's ID reserved/pending *before* mutating strength and only clear the
reservation on full success — so a retry after a partial failure re-attempts
the whole record rather than re-deriving `oldStrength` from an already-moved
value.

### WR-02: A failed skip-history write for pinned/revoked notes is silently swallowed and never retried

**File:** `cmd/pheromone_outcome.go:275-300`
**Issue:** In the pinned and revoked skip branches, `appendInfluenceHistory`'s
error is captured into `stateErr` but `newlyProcessed[rec.RecordID] = true`
executes unconditionally afterward, regardless of whether the write
succeeded. If the skip-history write fails, the credit record is marked
processed anyway — the audit trail entry ("skipped: pinned"/"skipped:
revoked") is permanently lost, and it will never be retried on a future run
because the record ID is now in `ProcessedRecordIDs`.

**Failure scenario:** A note is pinned; a harmful credit record arrives for
it; the history-append write fails once (e.g. transient I/O error). The
skip is never recorded anywhere, and the credit record is consumed silently
— there's no way to reconstruct after the fact that this note was ever
considered for tuning at that point.

**Fix:** Only set `newlyProcessed[rec.RecordID] = true` when
`appendInfluenceHistory` actually succeeds (mirror the pattern already used
in the main tuning branch immediately below, which correctly `continue`s
without marking processed on a history-write failure).

### WR-03: Re-importing the same pheromone export creates duplicate signal IDs and permanently orphans the duplicate's quarantine

**File:** `cmd/exchange.go:377, 474-495`
**Issue:** `runImportPheromones` always calls `importPheromonesData(inputPath,
xmlData, "")` — `sourcePrefix` is always empty on the plain CLI import path
(only `cmd/entomb_cmd.go`'s archive-restore path supplies a prefix). With no
prefix, an imported signal keeps its exact source-colony ID
(`sanitizeImportedSignalPrefix` returns `""`, so the `if prefix != ""`
rewrite in the loop at line 455 never runs). The merge step then does:
```go
file.Signals = append(file.Signals, sanitized...)
```
with no check against existing IDs. Importing the same export file twice (a
plausible ordinary mistake, or a periodic sync job) appends a second signal
with the *same ID* as the first.

Worse, the linked `PendingSuggestion` queue entry created immediately after
(`enqueuePendingNote`, `cmd/pheromone_approval.go:60-109`) dedupes by
*content hash*, not by signal ID — so the second import's queue item is
silently dropped as a duplicate (`duplicated = true`, nothing enqueued), even
though its corresponding quarantined signal *was* persisted to
`pheromones.json`. That second quarantined signal now has no linked pending
item at all, and `approvePendingNote`'s import branch is the only path
permitted to clear a quarantine — so this duplicate can never be approved
through the normal owner surface. It is a permanent, invisible quarantined
orphan.

**Failure scenario:** An owner runs `aether import pheromones export.xml`
twice (e.g. after a failed first attempt they assume didn't take). The
second run appends a duplicate-ID, permanently-quarantined signal with no
path to release it, and `findPheromoneSignalByID` (used by
`reinforceNote`/`revokeNote`/etc.) will operate on whichever of the two
same-ID entries appears first in the slice, silently shadowing the other.

**Fix:** De-duplicate imported signals by ID (or content hash) against
existing `pheromones.json` entries before appending, and apply an ID prefix
(source-colony name, or a content-hash-derived prefix) unconditionally on
every import path — not only the entomb restore path — so cross-colony ID
collisions cannot happen at all.

### WR-04: Trophallaxis packet sanitization skips two worker-authored free-text fields it claims to cover

**File:** `cmd/trophallaxis.go:305-334`
**Issue:** `trophallaxisSanitizePacketText`'s own doc comment states it "runs
`colony.SanitizeSignalContent` over every free-text field on packet, because
these fields are worker-authored and are replayed verbatim into later worker
briefs." The implementation sanitizes `Summary`, `VerificationStatus`,
`KnownFailures`, `OpenDecisions`, `Assumptions`, `NextWorkerInstructions`, and
`DoNotRepeat` — but never touches `packet.ChangedFiles` or
`packet.CommandsRun`, both of which are copied verbatim from
`input.Handoff.ChangedFiles`/`CommandsRun` (lines 272-277) with no sanitizer
call anywhere in the pack path. `cmd/trophallaxis_test.go` has no test that
feeds an injection payload through `ChangedFiles`/`CommandsRun` and checks it
gets rejected or escaped, matching the gap.

**Failure scenario:** A worker's handoff record reports a "changed file" or
"command run" string that is actually crafted prompt-injection text (e.g. a
line containing `ignore previous instructions` formatted to look like a
file path). It passes through `packTrophallaxisPacket` untouched and is
stored in `recruitment/packets.json`, bypassing the same integrity boundary
every other free-text field on the packet is subject to. (No current reader
in this codebase was found replaying these two fields into a live worker
prompt, so this is a latent gap rather than a confirmed exploited path today
— but the code's own comment says it should already be closed.)

**Fix:** Route `packet.ChangedFiles` and `packet.CommandsRun` through
`trophallaxisSanitizeFieldList` exactly like the other list fields, or
explicitly document (and test) why file paths/commands are treated as a
lower-trust class that is validated differently — the current silent gap
between the comment's claim and the code should not persist either way.

## Info

### IN-01: `-Inf` signal strength is excluded under the wrong reason

**File:** `cmd/context.go:1473-1503` (`computeEffectiveStrength`), `cmd/pheromone_resolver.go:108-122`
**Issue:** A signal with `Strength == -Inf` computes `effective := strength *
factor` = `-Inf`, then the `if effective < 0 { effective = 0 }` clamp fires
(since `-Inf < 0` is true), collapsing the result to exactly `0.0`. Back in
`resolveEffectivePheromones`, `eff < pheromoneEffectiveFloor` (`0 < 0.1`) is
true, so the signal is excluded with reason `"below-floor"` — never reaching
`pheromoneSignalMalformed`'s `math.IsInf` check, which would have correctly
labeled it `"malformed"`. (By contrast, `+Inf` and `NaN` strengths are
correctly caught as malformed, since `NaN < 0.1` and `+Inf < 0.1` are both
`false` in Go, so those fall through to the malformed check as intended.)
**Fix:** Have `computeEffectiveStrength` (or its caller) special-case a
non-finite input strength before the decay/clamp arithmetic, so
`pheromoneSignalMalformed`'s check always runs before the floor check for
every non-finite value, not just the ones the clamp doesn't happen to zero
out.

### IN-02: Prompt-injection heuristic phrase list is narrow

**File:** `pkg/colony/prompt_integrity.go:70-77`
**Issue:** `promptInjectionRuleSpecs` matches exactly five fixed phrasings
("ignore previous instructions", "ignore all previous", "disregard...",
"you are now", "new instructions:"). Trivial rewordings ("please set aside
the above", "from now on, act as...", "forget the earlier rules") are not
covered. This is a best-effort heuristic layer rather than a complete
defense, which may be an accepted trade-off for v1, but is worth naming
explicitly since CLAUDE.md calls this specific sanitizer out as "a real
security boundary."
**Fix:** No action required if this is an accepted v1 scope limit; otherwise
broaden the pattern set or note the limitation in the pheromone system docs.

---

_Reviewed: 2026-09-13T21:28:43Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
