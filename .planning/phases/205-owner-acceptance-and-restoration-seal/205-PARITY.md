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

## Phase 200

Phase 200's seven signed capability rows were signed by plan 205-07, which ran
each row's proof individually and recorded it passing. Rows cite the phase's
own `V-200-*` verification contract items from 200-CLASSIC-SYNTHESIS.md and
the exact proof named in `200-CLASSIC-COVERAGE.json`. Phase 200 has no owner
walk-through record (no 200-UAT.md exists); 200-VERIFICATION.md first found
gaps in candidate identity and specification digests, and plan 200-30 later
bound the exact proposal, semantic change and authority impact into the
identity the owner accepts. Where an Experience line below rests on a
verifier's inspection rather than the owner's own eyes, it says so.

### CAP-005

- **Outcome**: Evidenced: Before planning, the owner is only interrupted for a choice that genuinely needs the owner -- how the product should behave, who has authority, how much risk to accept, what is in scope, or what "done" means -- and never for a question the evidence already answers or a generic preference. | Path: aether discuss
- **Behavior**: Evidenced: V-200-LOOP-01 requires Go to issue and validate each alternating planning stage; classifyPlanningDecision in cmd/planning_decision.go is the one boundary both the discuss command (cmd/discuss.go) and the plan finalize path (cmd/codex_plan_finalize.go) call to classify a candidate. Proven by TestPlanningDecisionClassifyMaterialAndSuppressesNonOwnerChoices (run by 205-07). | Path: aether discuss
- **Experience**: Evidenced: V-200-PLATFORM-01 requires the decision shown to the owner to be understandable and never wrapper-invented; 200-VERIFICATION.md independently verified its row 200-07, "the owner is interrupted only for material choices evidence cannot answer." No owner walk-through of Phase 200 was recorded, so this rests on the verifier's inspection, not the owner's own eyes. | Path: aether discuss
- **Safety**: Evidenced: V-200-ACCEPT-01 requires explicit owner acceptance before a plan can ground execution; the material-decision boundary only narrows what the owner is asked, never widens what is accepted without the owner -- an evidence-answerable candidate is suppressed from the owner's queue, never recorded as an owner choice. | Path: aether discuss

### CAP-010, CAP-011, CAP-012

- **Outcome**: Evidenced: The owner can slot a new phase into an accepted plan without losing anything already agreed: the prior plan is kept exactly, finished phases stay finished, and the change only takes effect once the owner explicitly accepts it. | Path: aether phase-insert
- **Behavior**: Evidenced: V-200-REVISION-01 requires an insert or revision to preserve history and to mutate the active plan only through acceptance. Proven by TestInsertPhaseCandidatePreservesActiveRevisionAndAcceptsExactly (the candidate leaves the predecessor plan revision byte-for-byte intact), TestInsertPhaseImmutableStableIDsAndCompletedStatus (stable identities and completed status survive acceptance), and TestInsertPhaseRefusesMissingCoverageWithoutWrites (refusal before any write), all run by 205-07 against the phase-insert-candidate session in cmd/plan_revision.go and cmd/state_extra.go. | Path: aether phase-insert
- **Experience**: Evidenced: The inserted phase keeps the predecessor's stable and display identity while receiving a genuinely new ordinal, so the owner reads the same phase names they accepted plus one new entry, never a silently renumbered plan (V-200-PLATFORM-01). No Phase 200 owner walk-through exists, so this rests on the proof's rendered identity fields, not the owner's own eyes. | Path: aether phase-insert
- **Safety**: Evidenced: V-200-SPEC-01 and V-200-ACCEPT-01: a candidate lacking approved specification coverage is refused before any write and the on-disk state is asserted byte-identical afterwards (TestInsertPhaseRefusesMissingCoverageWithoutWrites); 200-VERIFICATION.md's finding that candidate identity omitted the semantic delta was closed by plan 200-30, which bound the exact proposal, semantic change and authority impact into what the owner accepts. | Path: aether phase-insert

### CAP-056, CAP-061, CAP-069

- **Outcome**: Evidenced: Planning draws on every allowed kind of evidence -- including lessons shared from the owner's other projects (the "hive") and the owner's own context notes -- but only while that evidence is genuinely current, so a plan is never quietly grounded on something withdrawn, out of date, or from a different goal. | Path: aether plan
- **Behavior**: Evidenced: V-200-CONFIDENCE-01 requires planning confidence to rest on evidenced gaps rather than unrelated evidence. collectPlanningEvidence in cmd/planning_evidence.go is called by the plan command (cmd/codex_plan.go), the plan finalize path (cmd/codex_plan_finalize.go) and phase research (cmd/phase_research_decision.go). Proven by TestPlanningEvidenceCollectAllKindsInStableOrder (all eight allowed source kinds, deterministic order), TestPlanningEvidenceFreshRejectsInactiveHiveStates (revoked, quarantined, dormant and superseded shared lessons all rejected as stale by one freshness gate), and TestPlanningEvidenceScopeRequiresExactPlanningFrontier (a context input goes stale the instant goal, session, specification or base plan drifts), all run by 205-07. | Path: aether plan
- **Experience**: Not evidenced: the catalogue records which evidence was admitted or refused and why, but no proof or owner walk-through in Phase 200 shows that admission or refusal rendered to the owner in plain words; Phase 200 has no owner walk-through record, and 200-VERIFICATION.md left the legibility of planning output as an open human check.
- **Safety**: Evidenced: V-200-REPLAY-01 and V-200-REVISION-01: only the affected scope is invalidated when a source drifts, and evidence from an inactive shared-lesson state can never support planning confidence -- the same freshness gate applies to every source kind, so no kind of evidence gets a looser rule. | Path: aether plan

## Phase 201

Phase 201's eight signed capability rows were signed by plan 205-07, which ran
each row's proof individually and recorded it passing. Rows cite the phase's
own `V-201-*` verification contract items from 201-CLASSIC-SYNTHESIS.md and
the exact proof named in `201-CLASSIC-COVERAGE.json`. Phase 201 is the one
phase in this record with an owner walk-through: 201-UAT.md records five
tests the owner confirmed by eye on 2026-09-11 plus an automated case per
plan deliverable; Experience lines below name the walk-through test they
rest on, and say plainly when no owner-facing evidence exists.

### CAP-003

- **Outcome**: Evidenced: When a helper fails during a build, the failure is written to the project's failure log (the "midden") tied to the exact build attempt and job it happened in, so the next helper and the owner can tell which run it belonged to instead of finding a loose note. | Path: aether build
- **Behavior**: Evidenced: V-201-IDENTITY requires every recorded fact to be attempt-scoped; recordDispatchWorkerOutcome in cmd/memory_feed.go resolves the phase's latest durable build attempt and binds the failure record to that attempt and job. Proven by TestFailureEvidenceCarriesTheAttemptIdentity (run by 205-07); 201-UAT.md's automated case for plan 201-10 confirms the record is written on either build lane. | Path: aether build
- **Experience**: Not evidenced: the signed proof asserts the stored record's attempt binding; no proof or owner walk-through shows the attempt-bound failure entry rendered on an owner-facing screen, and none of 201-UAT.md's five owner-confirmed tests covered the failure log.
- **Safety**: Evidenced: V-201-OUTCOME: a failed helper is a recorded outcome, never a silent absence -- one record per failure through the single memory-feed boundary -- and 201-UAT.md's automated case confirms evidence written by an interrupted attempt stays bound to the attempt that produced it, with a second attempt against the same phase never adopting or overwriting it. | Path: aether build

### CAP-004, CAP-051

- **Outcome**: Evidenced: A blocker a helper reports during a build lands in the same list of blockers the owner already reads on the status screen, and the same list that decides whether the phase may advance or close -- one list, one count, three readers. | Path: aether status
- **Behavior**: Evidenced: V-201-BOUNDARY and V-201-OUTCOME: a worker-reported blocker becomes a durable colony.FlagEntry (source escalation, attempt identifier set) appended into the same pending-decisions file that advancement, status and closure already read (TestBlockerTruthIsOneStore, pkg/colony/flags.go), and readBlockerSnapshotEvidence in cmd/status.go remains the single escalated-blocker counting function, counting that helper-reported blocker with zero changes (TestEscalatedCountHasOneCountingPath, re-run by 205-07 this milestone rather than carrying forward the Phase 198.3 "already proven" label). | Path: aether build
- **Experience**: Evidenced: the status screen's escalated-blocker count is produced by that one counting function, so the helper's blocker is visible where the owner already looks, with no second helper-only list to know about. No owner-confirmed test in 201-UAT.md covered a helper-reported blocker specifically; this rests on the automated case for plan 201-10 and the counting proof. | Path: aether status
- **Safety**: Evidenced: because there is exactly one store and one counting path, advancement can never treat a phase as unblocked while the status screen shows a blocker, or the reverse; the automated case in 201-UAT.md confirms a phase with unresolved flags is never reported as having nothing wrong. | Path: aether continue

### CAP-022

- **Outcome**: Evidenced: A phase only passes its check because the program's own deterministic checks passed -- never because a reviewer, a helper's own report, or a different code path said so -- and that holds on every way the check can be run. | Path: aether continue
- **Behavior**: Evidenced: V-201-BOUNDARY requires exactly one verification boundary; the unified accept/verify/advance body (runContinueAcceptVerifyAdvance, cmd/codex_verify_advance.go) routes the direct, plan-only and external finalize lanes through the one deterministic floor. Proven by TestDeterministicFloorIsTheOnlySourceOfAPass (run by 205-07); 201-UAT.md's automated cases record the finalize lane accepting a reconciliation exactly as the direct lane does, and an AST guard that refuses by name any new top-level function evaluating continue gates independently. | Path: aether continue
- **Experience**: Evidenced: V-201-IDENTITY requires the owner to be handed exactly one next action; 201-UAT.md's automated case records that a real advancing check and a real blocked check each end with the resolved verdict label, one recommended next action with its reason and alternatives, and exactly one cost-and-time block, on both continue lanes. 201-UAT.md test 1 (owner-confirmed) covers the build-side card naming the exact checks that ran. | Path: aether continue
- **Safety**: Evidenced: V-201-REPAIR and V-201-AUTOPILOT: the floor is never weakened for speed, and autopilot reaches the same decision body rather than a second interpretation (201-UAT.md automated case, proven by call-graph reachability); the live defect the synthesis named -- one lane accepting a reconciliation another lane rejected -- is closed with one acceptance rule. | Path: aether run

### CAP-024

- **Outcome**: Evidenced: When a check fails and the program considers an automatic repair, it looks at unresolved blockers and at whether this kind of failure keeps recurring, and its recorded reason names which of those actually drove the decision -- so a phase with open blockers is never reported as having nothing wrong. | Path: aether continue
- **Behavior**: Evidenced: V-201-REPAIR: repairEligibilityEvaluation in cmd/work_repair.go wraps classifyAutopilotRepairFailure with unresolved blocker flags and a recurring failure class as inputs and names the driving one, inside the single bounded-recovery model rather than a separate patrol mechanism. Proven by TestRepairEvaluationNamesItsDrivingInput (run by 205-07). | Path: aether continue
- **Experience**: Evidenced: 201-UAT.md test 4, owner-confirmed: the repair announces saving a safety checkpoint before it starts and restoring it if re-verification fails, in plain English with no invented project words, on the main checking path. | Path: aether continue
- **Safety**: Evidenced: V-201-REPAIR: exactly one checkpointed repair round per failing verification, a second automatic attempt refused by name, and a failed repair restoring the checkpoint (201-UAT.md automated case). The chat-driven check path was already missing the announcements before the phase and was declared out of scope -- a named limitation, not a silent one. | Path: aether continue

### CAP-029

- **Outcome**: Evidenced: A quick one-off request runs on the same footing as any other piece of work: it opens a real attempt, says honestly whether anything changed, and runs the program's checks when something did -- instead of the old read-only, research-only shape. | Path: aether quick
- **Behavior**: Evidenced: V-201-IDENTITY: the quick path in cmd/command_truth.go opens a real attempt, records a no-change or checked verdict, and produces exactly one attempt-bound failure record on error. Proven by TestQuickRunsOnTheSharedAttemptModel (run by 205-07); 201-UAT.md's automated case for plan 201-11 confirms the same. | Path: aether quick
- **Experience**: Not evidenced: the proof asserts the attempt and verdict records; no owner walk-through or rendering proof for the quick request's own closing card exists in 201-UAT.md.
- **Safety**: Evidenced: an error on a quick request produces exactly one attempt-bound failure record -- never zero and never a duplicate -- and a request that changed something cannot bypass the deterministic checks. | Path: aether quick

### CAP-066

- **Outcome**: Evidenced: What a build's helpers decided and learned is kept against that exact build attempt and shown read-only where it is next used, so lessons are no longer a separate comparison the owner has to run by hand. | Path: aether build
- **Behavior**: Evidenced: V-201-IDENTITY: deriveBuildKnowledgeDeltas in cmd/build_knowledge_deltas.go derives an attempt's own decision and learning deltas from its workers' handoffs, attached on both build lanes (cmd/codex_build.go, cmd/codex_build_finalize.go). Proven by TestDeriveBuildKnowledgeDeltas (run by 205-07); 201-UAT.md's automated cases confirm both lanes attach the deltas and that a write failure never fails an otherwise complete build. | Path: aether build
- **Experience**: Evidenced: 201-UAT.md test 3, owner-confirmed: decision and lesson notes are saved against the exact attempt and shown read-only at the moment the next decision uses them. At that walk-through no producer yet wrote into the pipe; plan 201-18 then made the handoff-derived deltas that producer, and its automated case records the attempt's own deltas reaching the closeout as evidence without duplication. | Path: aether build
- **Safety**: Evidenced: two attempts of the same phase never share deltas, and the derivation is deterministic, deduplicated and sanitised -- it can attach nothing that did not come from the attempt's own handoffs (201-UAT.md automated case). The legacy chamber-compare command is not revived as a second mechanism. | Path: aether build

### CAP-071

- **Outcome**: Evidenced: Claims and verification records are stored against the exact attempt they belong to, and an older record written at the old root-level location is still read -- but only through the same safety checks a new record gets. | Path: aether build
- **Behavior**: Evidenced: V-201-IDENTITY: attemptBoundArtifactPath in cmd/attempt_artifacts.go is the one canonical write path, and the legacy-compatible reader is validated by the identical containment rules. Proven by TestLegacyArtifactReadIsValidatedIdentically (run by 205-07); 201-UAT.md's automated case confirms a malformed legacy artifact is refused. | Path: aether continue
- **Experience**: Not evidenced: this is a storage-location discipline with no owner-facing screen of its own; the proof asserts path containment and validation, not anything the owner reads.
- **Safety**: Evidenced: no remaining write path uses the bare legacy root filenames, so a new record can never overwrite an unrelated attempt's, and a malformed legacy file is refused rather than trusted. | Path: aether continue

## Phase 202

Phase 202's eight signed capability rows were signed by plan 205-07, which ran
each row's proof individually and recorded it passing. Rows cite the phase's
own `V-202-*` verification contract items from 202-CLASSIC-SYNTHESIS.md and
the exact proof named in `202-CLASSIC-COVERAGE.json`. One disclosure governs
every Experience line in this section: all three of Phase 202's owner
walk-throughs in 202-UAT.md were skipped, so no owner has ever watched a live
bug-hunt ("Swarm"), research run ("Oracle") or cleanup in the shapes these
rows describe. Every proof here is a program test; none of the Experience
lines below claims owner-witnessed evidence that does not exist.

### CAP-021

- **Outcome**: Evidenced: A finished research answer leads with what to do, then how sure the program is and what is still unsettled, and keeps the source trail underneath -- never sources first. | Path: aether oracle
- **Behavior**: Evidenced: V-202-ORACLE: renderOracleFinalSynthesis in cmd/oracle_synthesis.go produces a fixed-order document (Recommendation, Confidence, What Is Still Unsettled, Sources, Evidence Trail) with every recommendation-section claim traced to a cited source, on top of the already-durable Phase 198.2 synthesis storage. Proven by TestSynthesisLeadsWithTheRecommendation (run by 205-07). | Path: aether oracle
- **Experience**: Not evidenced: all three of Phase 202's owner walk-throughs (202-UAT.md) were skipped, so no owner has read a live research answer in this shape; the only evidence is the program test above, which fixes the section order but cannot show the owner understood it.
- **Safety**: Evidenced: V-202-ORACLE: the research loop's numeric economics (oracleDepthLevels) are unchanged, and every recommendation claim must trace to a cited source, so the answer can never state more than its own evidence trail supports. | Path: aether oracle

### CAP-044, CAP-063

- **Outcome**: Evidenced: A bug-hunt repair saves a safety copy of the project before it changes anything, verifies its own fix, and puts the safety copy back if the fix failed -- the same undo net every other automatic repair in the program already uses. | Path: aether swarm
- **Behavior**: Evidenced: V-202-REPAIR: runSwarmDestroy saves a checkpoint through a thin adapter over Phase 201's checkpoint/restore primitive before the fix wave dispatches (TestSwarmRepairCheckpointsBeforeTheFixWave), and restores it on a failed verification and never on a pass (TestSwarmRepairRollsBackOnFailedVerification), both in cmd/swarm_repair_checkpoint.go and both run by 205-07. | Path: aether swarm
- **Experience**: Not evidenced: 202-UAT.md's owner walk-throughs were all skipped, so no owner has watched a live repair checkpoint or roll back; the two program tests above prove the order of operations, not what the owner saw.
- **Safety**: Evidenced: V-202-REPAIR: there is exactly one checkpoint primitive -- Phase 201's -- and no second implementation; a fix wave is never dispatched before its checkpoint exists, and a passing fix is never rolled back. | Path: aether swarm

### CAP-045

- **Outcome**: Evidenced: After a bug-hunt repair genuinely passes, the program proposes at most one short steering note ("focus here" or "avoid this") drawn from what the repair showed -- a proposal for the owner, never a note written into effect on its own. | Path: aether swarm
- **Behavior**: Evidenced: V-202-EPISODE: proposeSwarmLearningFromEpisode in cmd/swarm_episode.go proposes at most one scoped note from a passed, non-rolled-back repair's evidence, sanitized and never written as an active signal. Proven by TestSuccessfulSwarmProposesOneScopedNote (run by 205-07). Measuring the note's later effect is explicitly Phase 204's boundary, per 202-CLASSIC-SYNTHESIS.md's own CAP-045 ruling. | Path: aether swarm
- **Experience**: Not evidenced: no owner has seen a proposed note come out of a live bug-hunt (202-UAT.md skipped); the proof fixes the count and scope of the proposal, not how it reads to the owner.
- **Safety**: Evidenced: a failed or rolled-back repair proposes nothing, and the proposal passes the same content sanitization every steering note gets; it cannot become an active signal without a separate write the owner controls. | Path: aether swarm

### CAP-046

- **Outcome**: Evidenced: If the same problem survives three automatic repair attempts, the program stops repairing and hands the owner a plain-language case naming all three attempts instead of trying a fourth time. | Path: aether swarm
- **Behavior**: Evidenced: V-202-REPAIR: evaluateSwarmStrikeHistory and ensureSwarmEscalationForHistory in cmd/swarm_strikes.go were preserved unmodified through plan 202-07's checkpoint and lens refactor, as SYN-202-08 required. Proven by TestThirdStrikeRendersAnArchitecturalCase, re-run by 205-07 this milestone rather than carrying forward the Phase 198.3 "already proven" label. | Path: aether swarm
- **Experience**: Not evidenced: no owner has watched a third strike escalate live (202-UAT.md skipped); the proof asserts the rendered case names all three attempts, not that the owner read it.
- **Safety**: Evidenced: V-202-REPAIR names a strike-history regression as a merge-blocking failure; the escalation fires on the third strike and no fourth automatic attempt is made. | Path: aether swarm

### CAP-047

- **Outcome**: Evidenced: Every bug-hunt run leaves exactly one durable record of what it did and how it ended, findable afterwards from the project's history rather than only in a transient log. | Path: aether swarm
- **Behavior**: Evidenced: V-202-EPISODE and V-202-EVENTS: every Swarm run produces exactly one durable, issuance-bound, replay-safe episode carrying its outcome, bound to the existing swarmResultRecord rather than a second competing record (cmd/swarm_episode.go). Proven by TestSwarmRunProducesOneReplaySafeEpisode (run by 205-07). | Path: aether swarm
- **Experience**: Not evidenced: no owner has looked a finished bug-hunt up afterwards (202-UAT.md skipped); the proof asserts one record per run, not what the owner sees.
- **Safety**: Evidenced: a replayed run collapses onto the same episode instead of producing a duplicate, and the episode is one record on the one event model -- V-202-EVENTS forbids a second event bus. | Path: aether swarm

### CAP-048

- **Outcome**: Evidenced: Finished bug-hunt records are kept, never swept; removing one requires naming the exact record and the exact version previewed, and any mismatch refuses the whole batch. | Path: aether swarm-cleanup
- **Behavior**: Evidenced: V-202-EPISODE: removeSwarmEpisodes in cmd/swarm_episode.go requires the exact episode identifier plus the digest it was previewed at and refuses the batch on any mismatch -- never a directory sweep. Proven by TestSwarmRemovalRequiresIdentifierAndDigest (run by 205-07). | Path: aether swarm-cleanup
- **Experience**: Not evidenced: no owner has run a live cleanup (202-UAT.md skipped); the proof asserts the refusal rule, not what the owner is shown.
- **Safety**: Evidenced: a loose prefix, a stale digest or a missing identifier removes nothing at all, and the "archive, never delete" discipline is carried as retention metadata on the durable episode record. | Path: aether swarm-cleanup

### CAP-072

- **Outcome**: Evidenced: The owner's history and status screens can find every kind of past run -- bug-hunts, saved research, and build/check attempts -- through one shared index rather than three separate places. | Path: aether history
- **Behavior**: Evidenced: V-202-EPISODE: loadColonyEpisodeIndex in cmd/episode_index.go reads one shared projection over Swarm episodes, saved Oracle research, and build/check attempts, and is called by both the history and status commands (cmd/history.go, cmd/status.go). Proven by TestEpisodeIndexCoversThreeRecordSources (run by 205-07). | Path: aether history
- **Experience**: Not evidenced: no owner has browsed the index live (202-UAT.md skipped); the proof asserts the three sources are covered, not how the listing reads.
- **Safety**: Evidenced: the index is read-only -- one projection over existing records, not a fourth parallel record type -- so listing history can never alter a run's record. | Path: aether status

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

## Phase 204

Phase 204's seven signed capability rows were signed by plan 205-08, which ran
each row's proof individually and recorded it passing. 204-CLASSIC-SYNTHESIS.md's
verification contract names its dimensions in terms of the phase's own
`SYN-204-*` decisions rather than `V-204-*` items, so rows cite those.
204-VERIFICATION.md passed with no human verification required, and no owner
walk-through record exists for Phase 204. Two honest findings are recorded
below rather than rounded up: the seven named test gates (CAP-025) run only
through a developer build target, not any public program command; and the two
derived views CAP-067 and CAP-070 name (a generated changelog and a generated
phase-outcome page over the durable outcome ledger) have no production caller
anywhere in the repository -- the ledger they would read is real and written
on every lane, but nothing the owner can run reaches either view yet.

### CAP-001

- **Outcome**: Evidenced: A lesson the program remembers earns credit only when a real decision changed and a real effect was measured afterwards -- never merely because the lesson was handed to a helper and the phase happened to pass. | Path: aether continue
- **Behavior**: Evidenced: SYN-204-03 and SYN-204-05: recordPhaseApplicationCredit in cmd/application_evidence.go is the credit ledger's first real production writer, deriving decision and effect identifiers from the phase's own durable build-attempt evidence, and is called from phase-end consolidation on both check lanes (cmd/consolidation_lifecycle.go). Proven by TestPhaseApplicationCreditTracerEndToEnd (run by 205-08). Observation capture itself was already fixed before the phase began, so the ledger's restore-modern disposition is confirmed correct in direction (204-CLASSIC-SYNTHESIS.md ruling (c)). | Path: aether continue
- **Experience**: Not evidenced: the credit record is written at the end of a check and read by the note-tuning pass, but no proof or owner walk-through shows the credited outcome rendered to the owner; Phase 204 has no owner walk-through record and 204-VERIFICATION.md required no human check.
- **Safety**: Evidenced: SYN-204-06: credit requires both facts, a changed decision and a measured effect; a helper's own claim, or delivery of the lesson alone, earns nothing. | Path: aether continue

### CAP-025

- **Outcome**: Not evidenced: the seven named test gates (fast, focused, integration, provider, overnight, race, release) serve the people maintaining the program, not an owner using it; nothing the owner runs reaches them, so there is no owner-facing result to point at.
- **Behavior**: Not evidenced: the gates genuinely run, but only through the developer build target `make eval-gate-<name>` (Makefile, plan 204-07), which is neither an aether command nor a menu command and so cannot be named as a public path here. TestEvalGateVocabularyMatchesTheManifest (run by 205-08) proves the manifest and the gate vocabulary agree, and 204-07's own summary records the fast gate completing with every discovered test executed.
- **Experience**: Not evidenced: there is no owner-facing screen; a truncated gate run now fails by name instead of looking clean, but that message is read by a developer, not the owner.
- **Safety**: Not evidenced: the discovered-equals-executed check and the unshrinkable sentinel list are real (SYN-204-08), but they guard the test suite, not any public program path; recorded plainly so the slice is not silently rounded up to owner-facing restoration.

### CAP-043

- **Outcome**: Evidenced: The program can suggest a new skill it thinks would help, but never creates one on its own: the suggestion goes into the same tick-to-approve queue the owner already uses, approving creates the skill, and declining creates nothing. | Path: aether skill-create
- **Behavior**: Evidenced: SYN-204-10: both propose and auto modes raise the identical proposal into the owner's existing queue (colony.PendingSuggestion, pkg/learn/difficulty.go); the earlier propose default was a silent no-op, corrected by 204-CLASSIC-SYNTHESIS.md ruling (d). Proven by TestApprovingASkillProposalCreatesTheSkill (run by 205-08). | Path: aether suggest-approve
- **Experience**: Evidenced: the proposal appears in the one approval queue the owner already reads for suggested steering notes, so there is no second, skill-only place to check; approving or declining is confirmed back through that same command. | Path: aether suggest-approve
- **Safety**: Evidenced: SYN-204-10: skills sit on the retained-authority list refused by name from the automatic route, so no lesson the program learns can touch a skill without the owner's approval, and declining creates nothing. | Path: aether suggest-approve

### CAP-055

- **Outcome**: Evidenced: When the program records whether a lesson it applied actually worked, the answer comes from the real, evidence-gated credit record -- it is no longer marked a success automatically every time a phase advanced. | Path: aether continue
- **Behavior**: Evidenced: SYN-204-05 and SYN-204-06: recordInstinctApplicationsForPhase in cmd/instinct_application.go derives each application's outcome from the credit record rather than recording success unconditionally, and is called from phase-end consolidation on both check lanes (cmd/consolidation_lifecycle.go). Proven by TestApplicationOutcomeComesFromCreditNotFromAdvancement (run by 205-08). | Path: aether continue
- **Experience**: Not evidenced: the application outcome is a stored record read by later tuning; no proof or owner walk-through shows it rendered to the owner, and Phase 204 has no owner walk-through record.
- **Safety**: Evidenced: the negative branch Classic's own honour-system self-report never independently verified now exists: an application with no credit records no success, and a helper's own word is never the deciding input. | Path: aether continue

### CAP-057

- **Outcome**: Evidenced: When a steering note expires, one that proved valuable -- a hard "never do this" note, or one that was reinforced -- is kept in long-term memory, while a throwaway note is let go, never promoted merely because time passed. | Path: aether pheromone-expire
- **Behavior**: Evidenced: promoteExpiredSignalToEternal in cmd/phase_end_signals.go, called from the expiry path in cmd/pheromone_write.go, promotes only what signalIsWorthKeeping accepts. Proven by TestExpiringAValuableNoteKeepsItInLongTermMemory and TestExpiringAThrowawayNoteKeepsNothing (both run by 205-08). One named limitation: Classic's own gate was a decayed effective-strength threshold of 80 percent, and no Phase 204 plan re-derived the current gate against that exact value -- the equivalence proven is direction (never solely on elapsed time), not a numeric match (204-CLASSIC-SYNTHESIS.md ruling (c), recorded by 205-08). | Path: aether pheromone-expire
- **Experience**: Not evidenced: the promotion writes to the hub's long-term memory, warning on standard error if that write fails; no proof or owner walk-through shows the owner told which expiring notes were kept and which were dropped.
- **Safety**: Evidenced: an unreinforced, non-REDIRECT note promotes nothing on expiry, and a long-term-memory write failure warns but never fails the expiry that produced it. | Path: aether pheromone-expire

### CAP-067

- **Outcome**: Not evidenced: collectChangelogEntriesFromLedger in cmd/episode_ledger.go, the derived changelog view this slice names, has no production caller anywhere in the repository -- nothing the owner runs produces a changelog from the outcome ledger yet, so there is no useful result to point at.
- **Behavior**: Not evidenced: the view is a pure function exercised only by its own tests. TestDerivedViewsAreIdempotent and TestDerivedViewOverNoEpisodesIsEmptyNotAnError (run by 205-08) prove it is idempotent and that an empty ledger is empty rather than an error, but 204-CLASSIC-SYNTHESIS.md's own Behavior failure case -- a mechanism with a real production caller of zero -- applies to it today. The durable ledger it would read (recordEpisodeOutcome, written by every lifecycle lane's episode boundary) is real; the view over it is not yet wired.
- **Experience**: Not evidenced: no screen renders the derived changelog.
- **Safety**: Not evidenced: the view cannot alter the append-only ledger it reads, but with no public path reaching it there is no fail-closed behaviour to observe; recorded plainly rather than rounded up.

### CAP-070

- **Outcome**: Not evidenced: renderEpisodeOutcomeSummary in cmd/episode_ledger.go, the generated phase-outcome page this slice names, has no production caller anywhere in the repository; the owner cannot yet obtain a human-readable outcome page from the ledger.
- **Behavior**: Not evidenced: the view is a pure function exercised only by its own tests. TestEpisodeWithNoOutcomeRendersAsNoOutcome (run by 205-08) proves an episode without an outcome renders honestly as no outcome rather than an invented one, but the same zero-caller failure case from 204-CLASSIC-SYNTHESIS.md applies.
- **Experience**: Not evidenced: no screen renders the generated outcome page.
- **Safety**: Not evidenced: the never-invent-an-outcome rule is proven on the pure function only; nothing the owner can run reaches it.
