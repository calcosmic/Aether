# CalVault bookkeeping repair and qualification resume

Local stable **1.0.81** is installed. In plain English, Aether now gives its reviewer the current test results before asking for a verdict, remembers proof from earlier tasks after a targeted rebuild, and stops combining an old failure with a later successful retry.

## Source and delivery

- Main implementation: `2e043983`; final reviewer guidance: `4f11bebf`.
- Isolated stable source: `bc5b2b58`, followed by `b37b22e3`. This checkout contains the previously installed routing baseline plus this repair, not the unqualified native lifecycle candidate.
- Binary: `/Users/callumcowie/.local/bin/aether`; hub: `/Users/callumcowie/.aether`; both report **1.0.81**. Exact binary hash and source checkout are in `bookkeeping-hotfix-2026-09-18.json`.
- Clean source publish, all five integrity checks, disposable publish/setup/update, and installed-hub disposable setup/update passed. No GitHub push, tag, or npm publication.
- The existing npm dependency advisory surfaced during publish remains outside this repair; no dependency changes were made.

## What was repaired

1. The direct continue path saves current deterministic verification before launching its Watcher. The saved report explicitly says review is pending and cannot represent completed verification. A write failure blocks dispatch and advancement.
2. Continue reconstructs claims for other tasks from the latest eligible saved attempt, requiring the same plan revision and repository. Original hashes are retained; files are not rehashed into artificial historical proof. Current-task results, newer failures, and synthetic-evidence exclusions take precedence.
3. Historical task statuses come from the latest terminal attempt per task, so a successful retry is no longer combined with an obsolete blocked status. Failed lifecycle commits cannot contribute completed credit.
4. Reviewer briefs distinguish current verification from historical continue/review outcomes and attempt-specific result-collection reports.

## Validation

- Focused verification/claims/criteria regression selection: **46 passing cases normally and 46 with race detection**, including the saved CalVault replay. The isolated stable checkout passed the same selection.
- A final added brief assertion initially exposed omitted result-collection guidance in the Watcher path. That omission was repaired; final snapshot and brief controls passed normally and with race detection. Raw failure and success logs are retained separately.
- Snapshot replay reconstructed two task-claim groups from the latest build's one and resolved task 1.2 to a single completed result. No worker was launched against the immutable fixture.
- CalVault's actual restoration verifier passed: 487 backed-up/restored files and 1,444 migration-source hashes across 1,230 destinations.

## Live CalVault result

The installed runtime ran standard `continue` after a fresh complete colony-data backup. Supported `--reconcile-task` and `--read-only-artifact` options recorded three verified unchanged artifacts omitted by the targeted rebuild: `baseline-manifest.json`, `verify_restoration.py`, and `writer-dependencies.json`. No lifecycle JSON was hand-edited.

The actual Codex Watcher **completed and passed**. While it was running, `verification.json` existed with current results and `watcher.status=pending`; its final report contains the real completed reviewer verdict. The missing-file, obsolete-result conflict, and phase-evidence issues no longer block this run.

Phase 1 honestly remains **BUILT**, not advanced. The only remaining verification blocker is the accepted task 1.2 criterion requiring `Agent Ecosystem/Restoration/appearance-preview.png`. The owner-approved live appearance and superseded preview work are recorded in `Rehearsal.md` and FOCUS, but the accepted criterion has not been revised. The screenshot was not fabricated, and the plan was not silently rewritten. The runtime's suggested repeat reconciliation command does not fix that scope mismatch, so it was not repeated unchanged.

## Full lifecycle qualification

Still **gaps_found**. The installed Codex CLI is now 0.155.0, so its version, debug, prompt-input, app-server, and protocol-schema help were freshly captured with executable/output hashes. Those exposed surfaces do not establish the supported invocation-bound full effective-settings/model-facing-tools export or attributed cancellation observation-end export required by Plans18–23. Prompt reconstruction and protocol schemas were not substituted for actual session evidence.

No unchanged expensive qualification trial was repeated, no ANT requirement was marked complete, and Phase205's owner checkpoint was untouched. This successful live regression repair does not qualify the broader lifecycle.

## Evidence and rollback

Durable root: `/Users/callumcowie/.aether-backups/calvault-bookkeeping-resume-20260918`.

- `delivery.json`: precise runtime/source identity and real CalVault verdict.
- `evidence-manifest.json`: hashes of retained evidence, backups, and reports.
- `rollback/manifest.json`: 1.0.80 binary/hub/platform-home restore locations; 1,350 files were checked byte-for-byte before installation.
- `calvault-data-before-live/`: complete colony data before live continue.
- `calvault-after/`: final verification, continuation, claims, state, and historical report copies.
- `host-recheck/receipt.json`: 0.155.0 capability inspection and remaining provenance gap.

The original 08:43 CalVault snapshot and earlier qualification captures remain separate and unchanged.
