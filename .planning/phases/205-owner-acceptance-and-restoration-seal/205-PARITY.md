# Phase 205: Restoration Parity Record

This is PROOF-03's four-dimension parity ledger. PROOF-03 says a restored
piece of old ("Classic") behaviour only counts as genuinely brought back when
it can show real evidence on four separate things: that the owner gets a
useful result, that the mechanism genuinely runs (not just looks like it
runs), that the owner can understand what happened, and that the modern
program's safety rules still hold. A capability row ("CAP row" -- one
numbered entry in this project's master list of old features that needed a
home in the new program) is a "slice" here: one restored piece of behaviour
this record must carry a row for.

Each row below names one or more slices and, for each of the four
dimensions, either real evidence naming a public command a person can
actually run, or an honest "not evidenced" statement with the reason. A test
count, a percentage, or a combined score is never offered as completion
evidence anywhere in this document -- `go test ./cmd/ -run TestClassicParity`
is the command that checks this ledger holds together; it does not itself
prove any one piece of evidence is true.

Where two slices are genuinely delivered by the same mechanism and share the
same real evidence, they appear as one row naming both -- never as two rows
each independently crediting the same proof.

## Phase 199

Phase 199's 34 signed capability rows are grouped by the shared mechanism
that actually delivers them (the same area column 199-CLASSIC-SYNTHESIS.md's
own "Routed Phase 199 capability coverage" table already uses), each row
citing the phase's own `V-199-*` verification contract items and the exact
passing proof named in `199-CLASSIC-COVERAGE.json` -- never a single blanket
statement applied uniformly to all 34 without regard for which of them it
genuinely covers.

### CAP-006, CAP-007, CAP-052, CAP-064

- **Outcome**: Evidenced: Archiving a finished project keeps the useful record (what was learned, what signals were set, what was left unfinished) instead of losing it, while the review step before archiving stays separate and reviewable. | Path: aether entomb
- **Behavior**: Evidenced: V-199-TXN-04 requires entomb to publish and verify the archive and tombstone before any clearing, with replay-safe recovery; V-199-DIGEST-01 requires any content or cross-reference mismatch to block clearing outright. Proven by TestEntombManifest199Build, TestEntombManifest199CrossReferences, TestEntombManifest199Deterministic. | Path: aether entomb
- **Experience**: Evidenced: TestLifecycleCloseout199Entomb confirms the closing screen tells the owner plainly what was archived and what was verified, not a bare "done." | Path: aether entomb
- **Safety**: Evidenced: V-199-DIGEST-01 and V-199-TXN-04 together mean a broken or mismatched archive is never silently accepted -- active state is only cleared after the archive is verified. | Path: aether entomb

### CAP-008, CAP-062, CAP-065

- **Outcome**: Evidenced: Cleanup, cache, and integrity work happen as contained, explained maintenance the owner can preview and roll back -- not silent background changes. | Path: aether maintenance
- **Behavior**: Evidenced: V-199-MAINT-01 requires expert operations to expose dry-run/result/rollback truth without entering the ordinary journey; V-199-TXN-05 requires maintenance to follow validate, stage, commit, then receipt-or-rollback. Proven by TestMaintenanceState199CleanupOwnership, TestMaintenanceSkills199, TestMaintenanceInspectionStructuredResult. | Path: aether maintenance
- **Experience**: Evidenced: A maintenance run shows the owner exactly what it previewed, what it changed, and how to undo it, in one place. | Path: aether maintenance
- **Safety**: Evidenced: A failed or partial maintenance step rolls back rather than leaving the project in a half-changed state (V-199-TXN-05). | Path: aether maintenance

### CAP-013

- **Outcome**: Evidenced: Setting a FOCUS note still steers the colony's attention immediately, exactly as it did before. | Path: aether focus
- **Behavior**: Evidenced: V-199-AGENCY-01 requires signal acceptance, delivery boundary, and fallback to be distinguishable, machine-readable outcomes; TestPheromoneLifecycle_FocusSignal proves the FOCUS write path in cmd/pheromone_write.go does exactly that. | Path: aether focus
- **Experience**: Evidenced: The command reports honestly whether the note was accepted and delivered, rather than a generic "ok." | Path: aether focus
- **Safety**: Evidenced: V-199-TRUTH-01 forbids any renderer from claiming live acknowledgement or causal effect without typed evidence, so a FOCUS note is never shown as "in effect" without real proof it was stored. | Path: aether focus

### CAP-015

- **Outcome**: Evidenced: The owner can browse what actually happened in the project, in order, without hunting through raw files. | Path: aether history
- **Behavior**: Evidenced: TestLifecycleHistory199FocusedProjection proves history renders from the same shared read-only lifecycle/event projection every other status view uses (cmd/history.go, cmd/next_action.go), not a second, competing source of truth. | Path: aether history
- **Experience**: Evidenced: V-199-PROJECTION-01 requires every ordinary view -- including history -- to agree on episode, revision, phase, blockers, evidence status, and next action. | Path: aether history
- **Safety**: Evidenced: V-199-READONLY-03 requires status, phase, history, and recovery inspection to leave repository and state digests unchanged -- browsing history can never itself change the project. | Path: aether history

### CAP-016

- **Outcome**: Evidenced: The owner sees one normal, guided set of commands rather than the program's full internal command surface. | Path: aether help
- **Behavior**: Evidenced: TestCommandGuideLifecycle199 proves the help output is generated from the canonical `.aether/commands/help.yaml` source and cmd/wrapper_command_names.go, not hand-maintained separately per platform. | Path: aether help
- **Experience**: Evidenced: Expert and internal commands are kept out of the everyday guide, so the owner is never shown a command meant only for the program's own maintenance. | Path: aether help
- **Safety**: Evidenced: Listing help never mutates anything -- it is a pure read of the same canonical command source every platform wrapper is generated from. | Path: aether help

### CAP-017, CAP-068

- **Outcome**: Evidenced: Starting a new project does the required first-run setup work and tells the owner what it did, instead of the owner needing to run separate plumbing commands. | Path: aether init
- **Behavior**: Evidenced: TestLifecycleCloseout199Init and TestInitWithCharterJSONFlag prove init performs the registry setup and writes one visible charter/episode contract from the owner's accepted goal (cmd/init_cmd.go, cmd/registry.go). | Path: aether init
- **Experience**: Evidenced: The owner sees one plain-English written charter for the goal they approved, not a raw registry file. | Path: aether init
- **Safety**: Evidenced: V-199-READONLY-01 requires help and init's prerequisite inspection to cause no mutation before the owner's intent is accepted -- nothing is written until the owner has actually agreed to the goal. | Path: aether init

### CAP-018, CAP-020, CAP-049, CAP-050, CAP-059, CAP-060

- **Outcome**: Evidenced: One status screen shows real health, capability, and readiness facts, plus what research exists and how fresh the project's understanding of the codebase is, plus one safe next action -- not a decorative score and not stale information silently reused. | Path: aether status
- **Behavior**: Evidenced: V-199-TERRITORY-01 requires freshness and refresh policy to cover never/fresh/stale/unavailable/future/baseline-change cases; V-199-PROJECTION-01 requires the shared projection status reads from. Proven by TestLifecycleStatus199FullOrder, TestLifecycleStatus199NoInventedEvidence, TestLifecycleCloseout199Colonize, TestLifecycleNextAction. | Path: aether status
- **Experience**: Evidenced: The status screen states plainly when survey information is stale rather than silently treating it as current, and shows local research count and the latest item honestly. | Path: aether status
- **Safety**: Evidenced: V-199-READONLY-03 requires status inspection to leave repository/state digests unchanged, and V-199-TRUTH-01 forbids inventing evidence the program cannot actually see. | Path: aether status

### CAP-019

- **Outcome**: Evidenced: Moving colony state between versions keeps a safety copy and can be reversed if something goes wrong. | Path: aether migrate-state
- **Behavior**: Evidenced: TestMigrateStateRollbackRestoresExactBackupAndKeepsSafetyCopy proves the migration in cmd/command_truth.go is versioned and atomic, exactly restoring the prior backup on rollback. | Path: aether migrate-state
- **Experience**: Evidenced: The command reports plainly whether the migration succeeded, and what would be restored on a rollback. | Path: aether migrate-state
- **Safety**: Evidenced: A failed or rejected migration keeps the safety copy in place and never leaves state in a half-migrated shape. | Path: aether migrate-state

### CAP-026, CAP-036, CAP-037, CAP-038, CAP-039, CAP-040, CAP-041, CAP-042

- **Outcome**: Evidenced: Closing a project produces one focused, evidence-backed closure card with an honest seal recommendation, preserves project memory with its source, and never claims completion the evidence does not support. | Path: aether seal
- **Behavior**: Evidenced: V-199-SEAL-01 requires normal closure to be fail-closed, evidence-backed, and to retain active sealed state; V-199-FORCE-01 requires a forced closure to record owner, reason, and residuals and never count as normal success; V-199-TXN-03 requires seal to perform no closure effects before successful preflight and owner confirmation. Proven by TestLifecycleCloseout199Seal, TestLifecycleCloseout199ForcedSeal, TestLifecycleCloseout199Refusal, TestSealPromotionSurvivesWithRepoNameInText. | Path: aether seal
- **Experience**: Evidenced: The closing summary and next-project handoff are grounded in exact evidence (cmd/codex_visuals.go, cmd/next_action.go) and retained lessons/future work are shown without being counted as completed scope. | Path: aether seal
- **Safety**: Evidenced: V-199-SEAL-01 fails closed on blockers, malformed truth, missing evidence, and unresolved owner checkpoints before any closure effect runs (V-199-TXN-03). | Path: aether seal

### CAP-027, CAP-028

- **Outcome**: Evidenced: The owner can see the accepted plan's phase list and one phase's dependencies and success criteria without the program recomputing or reinterpreting truth that already lives in the accepted plan. | Path: aether phase
- **Behavior**: Evidenced: TestLifecyclePhase199List and TestLifecyclePhase199FocusedProjection prove phase views read directly from cmd/phase.go and the accepted `.planning/ROADMAP.md`. | Path: aether phase
- **Experience**: Evidenced: Phase detail shows dependencies and success criteria exactly as accepted, not a paraphrase. | Path: aether phase
- **Safety**: Evidenced: V-199-READONLY-03 requires phase inspection to leave repository/state digests unchanged. | Path: aether phase

### CAP-032, CAP-033, CAP-034, CAP-035

- **Outcome**: Evidenced: Coming back to a project after a break restores understandable context and one clear next action, and Autopilot survives being paused or interrupted mid-run without duplicating work. | Path: aether resume
- **Behavior**: Evidenced: V-199-RESUME-01 requires clean and reconstructed resume to converge on one honest next action with provenance; V-199-REPLAY-01 requires retry/replay to be idempotent across handoff, attempts, decisions, worktrees, and cleanup. Proven by TestLifecycleCloseout199Resume, TestResumeWrapperContract199, TestAutopilotContract199CompletionDoesNotSeal, TestRuntimeRecoveryCompatibility199. | Path: aether resume
- **Experience**: Evidenced: Resume states plainly whether its recovery point was validated from a real saved handoff or safely reconstructed from evidence, and the two are never shown as the same thing. | Path: aether resume
- **Safety**: Evidenced: V-199-READONLY-04 requires an invalid Autopilot entry to leave all scoped durable paths unchanged; the legacy `resume-colony` name is an invisible compatibility alias for the one canonical resume transaction, never a second code path. | Path: aether resume

### CAP-053

- **Outcome**: Evidenced: Updating the program to a newer version can be checkpointed and automatically rolled back if the installation or sync fails, instead of leaving a broken install. | Path: aether update
- **Behavior**: Evidenced: TestMaintenanceMutation199Update proves cmd/update_cmd.go checkpoints before the update and rolls back automatically on a failed installation or sync. | Path: aether update
- **Experience**: Evidenced: A failed update reports plainly that it rolled back and why, rather than leaving the owner guessing whether the program still works. | Path: aether update
- **Safety**: Evidenced: The checkpoint-and-rollback design means a failed update run can never leave the installed program in a half-updated, broken state. | Path: aether update

## Phase 203

### CAP-002, CAP-014

- **Outcome**: Evidenced: Every worker brief and the owner-facing pheromone display read FOCUS/REDIRECT/FEEDBACK signals through the same rule for what counts as "still active," so a signal the owner set once behaves the same wherever it is read back. | Path: aether pheromone-display
- **Behavior**: Evidenced: TestOneEffectivePheromonePredicate and TestEveryBriefReaderUsesTheResolver both pass against the single canonical resolver in cmd/pheromone_resolver.go (203-05 Task 3); the resolver is the only place strength/expiry/revocation logic lives. | Path: aether pheromone-display
- **Experience**: Evidenced: A note's live strength, decay, and status render in one formatted table an owner can read directly, rather than being separately inferred by each reader. | Path: aether pheromone-display
- **Safety**: Evidenced: A revoked or expired note is excluded by the same resolver every reader calls, so a stale or withdrawn note can never silently reappear in one reader while correctly excluded in another (SYN-203-10). | Path: aether pheromone-display

### CAP-009

- **Outcome**: Evidenced: The end-of-run family tree shows the owner who asked another helper for backup, whether the request was granted or refused, and what it cost -- a record drawn from the real recruitment and credit decision, not a guess. | Path: aether watch
- **Behavior**: Evidenced: TestAgencyEvidenceFromTrophallaxisDecision passes against cmd/agency_contract.go's real trophallaxis-decision/recruitment-credit join (203-12 Task 2) rather than a placeholder. | Path: aether watch
- **Experience**: Evidenced: The inline recruit/refusal lines and the closing family tree render through the caste-identity system Phase 202 already built, so a helper joining or being refused looks and reads like every other worker line. | Path: aether watch
- **Safety**: Evidenced: A refusal never stalls the requesting worker -- it continues its task alone and the command still reports success, never a failure. | Path: aether recruit

### CAP-030

- **Outcome**: Evidenced: A hard "never do this" REDIRECT note the owner set is honoured the same way during planning as everywhere else it is read, instead of planning silently using a looser rule that let a withdrawn note keep influencing plans. | Path: aether plan
- **Behavior**: Evidenced: TestActiveStrongExpiredExcludedByEveryReader confirms codex_plan.go's REDIRECT scan calls the same resolver as every other reader (203-05 Task 1), closing the exact defect 203-CLASSIC-SYNTHESIS.md names as its clearest single row. | Path: aether plan
- **Experience**: Evidenced: Planning output honours REDIRECT constraints the same way the owner already sees them rendered through the shared signal display. | Path: aether plan
- **Safety**: Evidenced: An expired or revoked REDIRECT note can no longer keep shaping plans after it should have stopped mattering -- the single-resolver rule applies to planning exactly as it applies to every other reader. | Path: aether plan

### CAP-031

- **Outcome**: Evidenced: The owner can permanently withdraw a "never do this" note with one command, and it stays withdrawn -- it cannot come back through any other path in the program. | Path: aether pheromone-display
- **Behavior**: Evidenced: TestRevokedNoteStaysOut proves the real BIO-08 revoke verb in cmd/pheromone_influence.go (203-11 Task 2) is a permanent state with no runtime path back, not a soft, reversible flag. | Path: aether pheromone-display
- **Experience**: Evidenced: A revoked note is shown as revoked in the same signal table every other note appears in, with a recorded reason and actor. | Path: aether pheromone-display
- **Safety**: Evidenced: No runtime path anywhere in the program can bring a revoked note back into effect; only the owner can revoke, and the revoke is written down. | Path: aether pheromone-display

### CAP-058

- **Outcome**: Evidenced: The owner reviews and dismisses a suggested note through one shared tick-to-approve queue, instead of four separate, narrower legacy helpers each doing a piece of the same job. | Path: aether suggest-approve
- **Behavior**: Evidenced: TestSuggestApprove_DismissSuggestion passes against the real, scope-narrowed dismiss verb in cmd/suggest_approve.go (203-08 Task 2); the other four named legacy helpers this phase considered stay out of this phase's boundary, recorded in 203-CLASSIC-SYNTHESIS.md's own CAP-058 routing note. | Path: aether suggest-approve
- **Experience**: Evidenced: A dismissal is confirmed back to the owner through the same command that showed the suggestion in the first place, with no second, unexplained surface. | Path: aether suggest-approve
- **Safety**: Evidenced: Dismissing a suggestion only ever affects that one suggestion's own record; the shared queue's other approve/reinforce/pin actions are unaffected. | Path: aether suggest-approve
