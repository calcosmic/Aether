---
phase: 205-owner-acceptance-and-restoration-seal
plan: 13
subsystem: release
tags: [local-release, versioning, backup, preservation, verification]
requires:
  - phase: 205-12
    provides: Committed readiness decision and preserved owner session
provides:
  - Local stable Aether 1.0.79 built from an exact committed source revision
  - Complete verified installation backup and guarded rollback scripts
  - Updated clean owner project with existing colony records preserved
affects: [205-14, 205-17, owner-walkthrough, PROOF-05]
actuals:
  tokens: 9706
  tasks: 3
  commits: 3
tech-stack:
  added: []
  patterns:
    - Compare committed runtime objects with the revision covered by the full gate
    - Pin before/after inventories and refuse rollback over later owner edits
key-files:
  created:
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-RELEASE-RECORD.md
    - .planning/phases/205-owner-acceptance-and-restoration-seal/205-13-SUMMARY.md
  modified:
    - .aether/version.json
    - npm/package.json
    - README.md
    - CLAUDE.md
    - AGENTS.md
    - .codex/CODEX.md
    - .opencode/OPENCODE.md
    - CHANGELOG.md
    - /Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.claude/settings.json
    - /Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.claude/rules/aether-colony.md
    - /Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System/.git/info/exclude
key-decisions:
  - Use canonical version.json and Makefile version flags; do not create VERSION
  - Publish locally from detached commit 3be6765150092517cb72f7e53e725380b29f2f53
  - Retain the approved 17-failure baseline and existing dependency limitations
  - Separate installation metadata from colony history when checking preservation
  - Keep PROOF-05 incomplete until the actual owner walkthrough and acceptance
patterns-established:
  - Durable source archive plus checksum-indexed command, installation and target receipts
requirements-completed: []
coverage:
  - id: D1
    description: Source metadata and versioned build agree on unused patch 1.0.79
    verification:
      - kind: integration
        ref: cmd/release_candidate_blackbox_test.go#TestPackedNPMReleaseCandidateContract
        status: pass
      - kind: unit
        ref: cmd/doc_version_truth_test.go#TestDocumentedVersionsMatchVersionFile
        status: pass
    human_judgment: false
  - id: D2
    description: Local installation verified with a complete backup and guarded undo
    verification:
      - kind: other
        ref: /Users/callumcowie/.aether-backups/phase205-release-20260916/receipts/integrity-after.json
        status: pass
      - kind: other
        ref: /Users/callumcowie/.aether-backups/phase205-release-20260916/receipts/rollback-selftest.json
        status: pass
      - kind: other
        ref: /Users/callumcowie/.aether-backups/phase205-release-20260916/receipts/rollback-after-target-check.json
        status: pass
    human_judgment: false
  - id: D3
    description: Owner project updated to 1.0.79 with original records and snapshots preserved
    verification:
      - kind: other
        ref: /Users/callumcowie/.aether-backups/phase205-release-20260916/target-final.json
        status: pass
      - kind: other
        ref: /Users/callumcowie/.aether-backups/phase205-release-20260916/receipts/target-record-preservation.json
        status: pass
    human_judgment: false
  - id: D4
    description: Actual owner walkthrough and overall acceptance remain pending
    requirement: PROOF-05
    verification: []
    human_judgment: true
    rationale: This release prepares the owner session; no owner journey or acceptance was observed.
duration: 27 min
completed: 2026-09-16
status: complete
---

# Phase 205 Plan 13: Local release and verification summary

**Aether 1.0.79 is installed locally from a pinned source commit, the prepared owner project is updated and clean, and complete backup/restore evidence is retained.** In plain English: the owner can now test the new program, with a checked way back to the old installation. The owner's actual walkthrough and acceptance remain pending; `PROOF-05` is incomplete.

## Performance and scope

- **Tasks:** 3 of 3 complete.
- **Measured execution window:** 2026-09-15T23:25:13.891862+00:00 to 2026-09-15T23:52:35.091217+00:00 (27 minutes, beginning with durable evidence initialization).
- **Tracked Aether files:** 10, including this summary; eight release metadata/documentation files and two planning artifacts.
- **Target tracked files:** two managed Claude guidance/settings files. One exact local Git exclusion retains this update's recovery journal. Eleven generated program files and the installed-version marker were updated under existing ignore rules.
- **Actuals basis:** 38821 characters in the realized nine-file Aether task diff and two-file target diff, divided by four and rounded up: 9706 estimated tokens. This excludes this summary, generated installed copies and local operational helpers; it is not a harness token count. Three task commits exclude the separate summary metadata commit.

All Aether work stayed in `/Users/callumcowie/repos/Aether/.claude/worktrees/agent-p205-13-codex-20260916`, branch `worktree-agent-p205-13-codex-20260916`. The only other mutations were the explicitly authorized installation, target project, backups and disposable pinned clone. Parent-owned `STATE.md`, `ROADMAP.md` and `REQUIREMENTS.md` were not edited.

## Accomplishments

- Prepared **1.0.79**, the next unused patch above the installed **1.0.78**, and synchronized the canonical version, npm metadata, current-version docs and plain-English changelog.
- Revalidated a complete **1,427-file / 59,946,533-byte** backup; published from the exact release commit; passed five source integrity checks and binary/hub version agreement; saved a guarded undo script beside the backup.
- Updated the owner's project marker from **1.0.63 to 1.0.79**, committed only the two tracked managed files, and verified unchanged colony records and original/snapshot branches. The prepared session is clean and remains unsealed/unarchived.

## Task commits

All commits used normal hooks. No Git push, remote tag, GitHub release or npm publication occurred.

| Task | Repository / action | Commit or durable receipt |
|---|---|---|
| 1 | Release source: version, synchronized docs and changelog | `3be6765150092517cb72f7e53e725380b29f2f53` |
| 2 | Local publish and verification; no repository source mutation | `receipts/publish.json`, `publish-verification.json`, `installation-before.json`, `installation-after.json` in the durable evidence root |
| 3 | Owner-project managed guidance and session hooks | `fe1ec13284bf60a3e3205cbc7028f1c275eee5e2` |
| 3 | Complete release record, restore commands and evidence | `79adc910584375a70fc859ed6db67a63e2f4affe` |

**Plan metadata:** this summary is committed separately after task work. No task-2 Git commit is invented for an external installation action.

## Files created and modified

- `.aether/version.json`, `npm/package.json`: release 1.0.79; no `VERSION` file was created.
- `README.md`, `CLAUDE.md`, `AGENTS.md`, `.codex/CODEX.md`, `.opencode/OPENCODE.md`: current-version declarations synchronized by `make version-sync`.
- `CHANGELOG.md`: concise Phase 203 worker/runtime, Phase 204 learning and Phase 205 finish/archive changes. No limitations card or owner acceptance claim.
- [205-RELEASE-RECORD.md](205-RELEASE-RECORD.md): exact source/gate hashes, versions, backup counts, per-platform results, downstream preservation, limitations and complete guarded rollback commands.
- Owner target `.claude/settings.json` and `.claude/rules/aether-colony.md`: generated session-start hooks and current Aether guidance, committed on its existing session branch.
- Owner target `.git/info/exclude`: one exact new transaction-directory exclusion, with original bytes and before/after hashes preserved.
- Durable local artifacts: installation inventories, source archive, captured commands and exit codes, saved target program bytes, 70 copied transaction records, and shared/target rollback scripts. The operational manifest indexes 167 files.

## Verification and evidence

### Full gate reused without overclaiming

The committed readiness report and parent's final review were checksum-verified before mutations. The gate covers `8391fef9cfc7ba122d8834d83009c4006ed5af3e`. Build and vet passed; both complete normal/race suites exited **1** with exactly the unchanged **17 known failures**. Both command-package runs discovered and executed **5,573 tests in 59 groups**, all 18 other tested packages passed, and there were no additional top-level failures, race warnings or timeouts.

The release preserves identical Git objects/modes for **1,517 runtime/dependency files**, including **1,321 Go files**, 194 TypeScript-host files, and `go.mod`/`go.sum`. The only delta from the tested revision is five already integrated parent readiness/state artifacts and eight release metadata/docs files. The full suites were not repeated and no runtime/test changes were introduced.

### Checks run for this plan

| Check | Outcome |
|---|---|
| Version sync and repeat/idempotence check | Passed; current release declarations all 1.0.79 |
| Build with Makefile version linker flag; built `version` | Passed; reports 1.0.79 |
| Three focused Go version/packed-release tests | Passed, including the five packed-release subcases |
| npm bootstrap tests / dry-run packaging | 11 tests passed; four-file 1.0.79 package |
| Backup integrity and live comparison, repeated immediately before publish | All ten groups matched; original backup reused |
| Process/Git preflight and pinned clone | No other publisher or relevant Git lock; detached source clean before/after |
| Source bootstrap publish / source integrity / version agreement | All exited 0; five integrity checks passed |
| Installed platform/source audit | Nonzero copied-or-unchanged counts on every platform; all hub definitions matched source |
| Rollback fixtures and live dry-runs | Six fixture scenarios passed; shared and target check modes passed without undoing the release |
| Target `update --force` / version / consumer integrity | Passed; 14 copied, 381 unchanged; own marker 1.0.79; four integrity checks passed |
| Existing records, snapshots and final target cleanliness | Passed; details below |

The publisher omits native per-platform count output. Independent file/hash accounting found: Claude commands **3 updated + 61 unchanged**, Claude agents **3 + 24**, OpenCode commands **3 + 61** at both home layouts, OpenCode agents **3 + 25** at both layouts, Codex agents **7 unchanged plus 20 preserved local variants**, and **14 unchanged Codex shims**. The hub has 64/64 Claude/OpenCode commands and 27/28/27 Claude/OpenCode/Codex agents matching source.

### Owner records and prepared branches

All **1,304 existing colony/extended records**, including **382 data files**, kept their bytes, permissions and modification times. The one other entry in plan 12's broad inventory is the intentionally updated installed-version marker. It is installation metadata, not colony history. The common unchanged-record digest is `62f40d4940109e2c865c382ecb39e6f6674a06a4a71973db13a0ce920cd448f0`.

`COLONY_STATE.json` retains SHA-256 **`883c58156730f89d329acf97aeb1ce954ba96efb3c6b5996ccf3ec2a83cbaa0d`**: original completed three-phase/twenty-task colony, no seal outcome. The main original/snapshot branches and the nested research original/snapshot/session branches retain their recorded hashes. Neither branch setup nor experiment snapshot was repeated.

The update added 55 root-journal files and 15 data-transaction files, all retained and copied to durable evidence. Exactly `/.aether-transactions/update-20260915T233925.601030000Z/` was excluded locally so those recovery records do not dirty Git. The three prior cache exclusions remain intact. Only the two legitimate tracked managed-file changes were committed. Both main and nested target working trees are clean.

### Durable receipt locations

Operational evidence: `/Users/callumcowie/.aether-backups/phase205-release-20260916/`.

- `evidence-manifest.json`: `8dd9fe70625b8253d456d967eff103c5478c638ff47b7ab70834578c26d4635d`; 167 indexed artifacts, checked against sizes/hashes.
- Full pre-publish backup: `/Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/`; manifest SHA-256 `fc5dbd9da775c909d32bd3fad2c9f4443a50d88f1136612f343c8a944f477073`.
- Shared rollback script: that backup's `rollback-205-13.py`, SHA-256 `3f415c5c98dec28d5a20fd437af17b25ec015c59bab073048422774fce57915a`.
- Target undo: evidence-root `target-rollback.py`, SHA-256 `9f2df11e0125991e615f0c126f32a4a936d6b7669f0aeb3cca3b5f860001525c`.
- The release record contains the complete restore and validation commands plus individual receipt hashes. Restoring the shared script covers the binary, entire hub and changed platform assets; target undo also restores the previous project program files/marker. Both refuse later owner edits.

## Decisions and deviations

1. **Canonical version/build correction:** the plan's `VERSION` and bare-build examples were stale. Used `.aether/version.json`, `make version-sync` and the Makefile's version linker flag, verified the resulting binary, and preserved historical version references. Eight-file release commit `3be67651`.
2. **Publisher count output:** runtime publish drops its internal per-platform counters. Recorded native output unchanged and supplied an independent read-only source/before/after hash audit. No runtime change or duplicated publish was made to add presentation output.
3. **Installation marker classification:** plan 12's 383-data/1,305-extended preservation inventory includes the managed update marker. Its deliberate 1.0.63→1.0.79 update is recorded separately; every other 382-data/1,304-extended entry is unchanged.
4. **New transaction journal:** the authorized update leaves a new untracked recovery journal. Retained all bytes, copied all 70 new recovery records durably, and added one exact local Git exclusion. No existing owner work was discarded or broadly hidden.

These four execution corrections preserve the approved release scope. The three source/target task commits are listed above; the installation-only corrections are recorded by durable receipts, not fictional repository commits.

## Issues and limitations

- The accepted full gate still has 17 known test failures and the planning register still has 34 open entries. This plan does not waive them or call the suites all-green.
- The publisher preserves twenty user-level Codex agent variants, although the complete hub Codex set matches source. Their exact paths remain in the release record's linked evidence.
- The normal Aether source-commit hook emitted its advisory package-validation warning because `bin/validate-package.sh` is absent. Hooks were not bypassed; focused packaging checks passed.
- During publish, npm reported one high-severity advisory for the existing locked **js-yaml 4.3.1**, `GHSA-2883-xcg3-v3hh` (YAML merge-source CPU exhaustion). Read-only `npm audit --json` exited 1 and reports a fix in 4.3.2. No dependency change was made outside the approved metadata release; the finding is retained for a separately verified follow-up.
- Actual owner journeys, timing/cost judgments, Ableton behavior and overall acceptance have not been observed. **PROOF-05 is pending.**

## User setup required

No new service setup is required. The machine and prepared target now have the release needed for the separately scheduled owner session.

## Next phase readiness

Plan 205-13's local release and verification are complete. The release record and durable rollback evidence are ready for the parent and owner-session plans. This executor did not advance shared phase state, mark the owner session complete, seal/archive either project, or publish remotely.

## Self-Check: PASSED

Confirmed the actual task commits and their file sets, committed release record, all cited durable hashes, the unchanged runtime comparison, final target cleanliness and snapshot refs, intentional metadata/journal exceptions, and empty `requirements-completed`. Both rollback check modes passed against the live successful installation. No full-suite rerun or owner acceptance was claimed.
