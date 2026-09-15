# Phase 205: Local release record — Aether 1.0.79

**Aether 1.0.79 is installed and verified on this machine, and the owner's prepared project has been updated.** The old installation has a verified backup and guarded restore commands below. In plain English: the walkthrough will use the new program, and the previous installation can be put back without guessing which files changed.

Plan **205-13 is complete**. The actual owner walkthrough and acceptance are still pending; **PROOF-05 remains incomplete**. Neither project was sealed or archived. No branch or tag was pushed, no remote release was created, and nothing was published to npm.

Recorded **2026-09-16, Europe/Zurich**; the command receipts use UTC on 2026-09-15. The operational verification completed at **2026-09-15T23:44:57.228928+00:00**.

## Versions and exact revisions

| Surface | Before | After |
|---|---|---|
| Installed `~/.local/bin/aether` | 1.0.78 | 1.0.79 |
| Shared `~/.aether/version.json` | 1.0.78 | 1.0.79 |
| Source `.aether/version.json` and `npm/package.json` | 1.0.78 | 1.0.79 |
| Owner project's `.aether/data/installed-version.json` | 1.0.63 | 1.0.79 |

The project marker records its last companion-file update. Its older value does not contradict the machine's previously installed 1.0.78 binary.

| Role | Full commit |
|---|---|
| Assigned worktree's expected initial HEAD | `6b1ca37f884e144e734ed66e3aa60052291f255a` |
| Revision covered by the parent full test gate | `8391fef9cfc7ba122d8834d83009c4006ed5af3e` |
| Committed release source, task 1 | `3be6765150092517cb72f7e53e725380b29f2f53` |
| Owner-project update, task 3 | `fe1ec13284bf60a3e3205cbc7028f1c275eee5e2` |

All Aether edits and commits were made in `/Users/callumcowie/repos/Aether/.claude/worktrees/agent-p205-13-codex-20260916`, on `worktree-agent-p205-13-codex-20260916`. The main checkout and unrelated worktrees were not changed. Parent-owned `STATE.md`, `ROADMAP.md` and `REQUIREMENTS.md` were left to the parent.

### Why the existing full gate applies

The release and tested revisions have identical Git object IDs and modes for **1,517 runtime/dependency files**, including **all 1,321 Go files**, 194 TypeScript-host files, and `go.mod`/`go.sum`. The complete delta has 13 paths: five parent readiness/state documents already present at dispatch, plus this plan's eight version/documentation/changelog files. No Go runtime, test, dependency or TypeScript implementation changed. `source-proof.json` records the comparison and every changed path. The durable `release-source.tar.gz` contains the exact committed release tree; its embedded Git commit ID was verified.

The committed [readiness report](205-PRE-SESSION-READINESS.md) and parent's final review were verified before mutations. The parent ran build, vet, normal tests and race tests serially in a disposable clone. Build and vet exited 0. **Both full test commands exited 1**, with the same approved 17 failures, no new top-level failure, no race warning and no timeout. Each command-package run discovered and executed **5,573 tests across 59 groups**; all 18 other tested packages passed. These are completed baseline-relative results, not an all-green result. The full suites were not duplicated.

Parent evidence is retained under `/Users/callumcowie/.aether-backups/phase205-execution-evidence-20260915/repaired-gate/`. Final-review SHA-256: `9aba08e0135f861126c5902eaf7cd600e0781a7741df8424295bd104ad4beb23`; results SHA-256: `57d4599d7b7765a6a53b48740c70726927321b508ba91c5efb2191c8068f41c0`. Committed readiness SHA-256: `96d5b94b2dcc827de6fd0b191a451a0181a3fb8422cf9c2f52549572e27afcac`.

## What the local release contains

- **Phase 203, worker/runtime work:** shared limits for additional workers, recorded handoffs and outcomes, and outcome-based adjustment of guidance.
- **Phase 204, learning work:** durable run records, checks around promotion and rollback, and reports separating useful outcomes from owner intervention.
- **Phase 205, finish/archive repairs:** missing handoff notes are handled during archive, resume refreshes them, chat receives finish/archive messages, machine-readable output stays valid, and completion checks reject named tests that never ran.

The changelog names these changes without claiming owner acceptance. Accepted limitations remain in the planning records. In particular, this release does not claim the already recorded unconnected observation-to-proof promotion path is working, or that every cost value is available.

### Version choice and focused checks

There is no `VERSION` file. The stale plan command was corrected to use `.aether/version.json`; `make version-sync` updated npm metadata and the five current-version documents. A second sync was identical. The remaining `1.0.78` strings belong to dated planning evidence or the parent-owned pre-release state notes; the release's current-version declarations all agree.

Read-only local tags/history, remote tags, the GitHub releases API and npm version history contained no `1.0.79`. The highest observed GitHub release was `v1.0.43`; npm also contains historical versions from an older numbering scheme. The selected version is the next unused patch above the installed 1.0.78. Raw observations are in `receipts/local-tags.log`, `local-version-history.log`, `remote-tags.log`, `remote-releases.log` and `npm-versions.log`.

| Check | Result |
|---|---|
| `make version-sync`, including repeat/idempotence check | Passed |
| `make build BINARY=.../preflight-aether` | Passed; Makefile supplies `-X github.com/calcosmic/Aether/cmd.Version=1.0.79` |
| Built program's `version` | 1.0.79 |
| Focused Go release tests | Passed: `TestDocumentedVersionsMatchVersionFile`, `TestPackedNPMReleaseCandidateContract`, `TestReleaseVersionAgreementDetectsNpmMismatch` |
| `npm --prefix npm test` | 11 passed, no failures or skips |
| `npm pack --dry-run --json` in `npm/` | 1.0.79 package, four files |
| Normal task-commit hooks | Ran; Aether's advisory package hook warned because its referenced `bin/validate-package.sh` is absent |

The exact focused Go command, timeout, output and exit status are in `receipts/focused-release-checks.json` and `.log`. No hook was bypassed.

## Complete backup and publish

Backup: `/Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/`.

Manifest: `manifest.json`, SHA-256 **`fc5dbd9da775c909d32bd3fad2c9f4443a50d88f1136612f343c8a944f477073`**.

All **1,427 files / 59,946,533 bytes**, directory paths and modes were compared between backup and live installation. There were no symlinks. The original backup bytes matched its manifest, and the live installation still matched the backup; no fresh backup was needed. The full comparison was repeated immediately before publish, with a passing receipt. This covers the binary, entire hub and all eight platform-home groups.

The table uses paths relative to `/Users/callumcowie`. The last column counts added/modified/removed paths, including directories.

| Installation group | Backup files | Backup bytes | After files | After bytes | Path delta A/M/D |
|---|---:|---:|---:|---:|---:|
| `.aether` | 1,006 | 6,271,642 | 926 | 5,538,359 | 3/24/88 |
| `.local/bin/aether` | 1 | 50,388,898 | 1 | 50,861,106 | 0/1/0 |
| `.claude/commands` | 99 | 410,554 | 100 | 414,380 | 1/2/0 |
| `.claude/agents/ant` | 27 | 353,027 | 27 | 355,049 | 0/3/0 |
| `.codex/agents` | 93 | 1,263,045 | 93 | 1,263,045 | 0/0/0 |
| `.codex/skills/aether` | 14 | 54,518 | 14 | 54,518 | 0/0/0 |
| `.opencode/command` | 65 | 244,824 | 66 | 248,650 | 1/2/0 |
| `.opencode/agent` | 28 | 356,455 | 28 | 358,477 | 0/3/0 |
| `.config/opencode/commands/ant` | 63 | 241,206 | 64 | 245,032 | 1/2/0 |
| `.config/opencode/agents` | 31 | 362,364 | 31 | 364,386 | 0/3/0 |

The hub cleanup removed 83 obsolete program files and five directories, including old generated JavaScript beside TypeScript sources and settings copied from old worktree paths. The complete path-by-path before/after record is `installation-delta.json`. Shared preferences and other hub records were preserved.

### Pinned publisher

A disposable clone was fetched by exact commit hash, checked out detached at `3be6765150092517cb72f7e53e725380b29f2f53`, and verified clean before and after publishing:

`/private/tmp/aether-phase205-execution/plan13-release-checkout`

Immediately before publishing, process inspection found zero other publishers and no relevant Git lock. The exact bootstrap action was:

```sh
cd /tmp/aether-phase205-execution/plan13-release-checkout
AETHER_OUTPUT_MODE=visual go run ./cmd/aether publish --channel stable --binary-dest "$HOME/.local/bin"
/Users/callumcowie/.local/bin/aether integrity --source --channel stable --json
/Users/callumcowie/.local/bin/aether version --check
```

All three commands exited **0**. All **five source integrity checks passed**. The installed binary's hash and modification time changed; `publish-verification.json` records both. Failure handling was prepared to restore the verified installation immediately on a failed publish/integrity/version/cleanliness check; it was not needed.

### Verified platform counts

The current publisher discards its internal per-platform counters instead of printing them. The following separate read-only audit compared the exact source file set and hashes with the saved before-state and live destinations. This is a documented presentation deviation from the plan; the numbers are measured, not attributed to nonexistent CLI output.

| Platform destination | Copied/updated | Unchanged, matching source | Preserved local variants |
|---|---:|---:|---:|
| `.claude/commands` | 3 | 61 | 0 |
| `.claude/agents/ant` | 3 | 24 | 0 |
| `.opencode/command` | 3 | 61 | 0 |
| `.opencode/agent` | 3 | 25 | 0 |
| `.config/opencode/commands/ant` | 3 | 61 | 0 |
| `.config/opencode/agents` | 3 | 25 | 0 |
| `.codex/agents` | 0 | 7 | 20 |
| `.codex/skills/aether` | 0 | 14 | 0 |

Every platform has a nonzero copied-or-unchanged count. The hub itself matches the release source for **64 Claude commands, 64 OpenCode commands, 27 Claude agents, 28 OpenCode agents and 27 Codex agents**. Twenty existing user-level Codex agent variants were preserved by the publisher's local-change policy; their exact paths are in `platform-verification.json`. All 27 hub Codex definitions match source. The 14 installed Codex shims remained unchanged.

## Owner project update and preservation

Target: `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System`.

The branch was already clean on `p205-owner-session-20260915`, at `363fd07630d9028d2d1a18edf3f5cc21e7dece2d`. No snapshot or branch switch was repeated. All 1,305 entries in plan 12's broad `.aether/` preservation inventory matched before the update, including modes and modification times. A dry-run identified exactly 14 managed project writes and no platform-home changes; the old contents of those managed targets were separately saved.

```sh
cd "/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System"
aether update --force
aether version --check
aether integrity --json
```

The update exited **0**, reported **14 files copied and 381 unchanged**, and produced verified transaction `update-20260915T233925.601030000Z`. Its actual JSON receipt is saved in `receipts/target-update.log`. The program reports 1.0.79 from this project, all four consumer integrity checks pass, and the project's own marker now states **1.0.79** with timestamp `2026-09-15T23:39:25Z`.

The 14 writes were two tracked Claude guidance/settings files, eleven ignored generated TypeScript-host files, and the ignored installed-version marker. Only `.claude/rules/aether-colony.md` and `.claude/settings.json` were committed, using normal hooks, as `fe1ec13284bf60a3e3205cbc7028f1c275eee5e2` (“Update Aether guidance and session hooks to 1.0.79”). The main and nested session working trees are clean.

### Existing colony records

The installed-version marker is managed installation metadata, although it lives under `data/` and was included in plan 12's broad inventory. Updating that one marker is intentional. **All other 1,304 existing records, including 382 existing files under `.aether/data/`, retain their bytes, modes and modification times.** Their common content digest is `62f40d4940109e2c865c382ecb39e6f6674a06a4a71973db13a0ce920cd448f0`.

`COLONY_STATE.json` still has SHA-256 **`883c58156730f89d329acf97aeb1ce954ba96efb3c6b5996ccf3ec2a83cbaa0d`**. The colony remains `COMPLETED`, with its original three completed phases and twenty completed tasks, and no `seal/outcome.json`. It was not sealed or archived.

The unchanged original main branch remains `codex/skills-as-commands` at `ccf182644fd57eea8f063466002cd9b062d9be45`. The main snapshot `p205-owner-work-snapshot-20260915` remains at `e8941d6f5e1f3f1834357d4746e7ffcf0f77b962`. The nested research repository, `MaxForLive_Vault/MDS-System-Archive/03-Research/pattern-verification-full`, keeps its same-named snapshot at `f09a3a27e7a81f3e50bb5e79b86d0d97c9a0f61f`, and its original `master` and session branch at `285a68a9bfa9a3899ac3fd0a03fde516c6e871fc`.

### New recovery records and clean Git status

The update generated **55 root-journal files** under `.aether-transactions/update-20260915T233925.601030000Z/` and **15 new data transaction files**. These are new update receipts; no old record was rewritten. All 70 were copied to durable `target-update-records/`, with a hash inventory in `target-update-records-manifest.json`.

The root journal was untracked and would leave the prepared session dirty. One exact local exclusion was appended to the target's `.git/info/exclude` to retain it without committing runtime records:

```text
/.aether-transactions/update-20260915T233925.601030000Z/
```

The previous exclude bytes are saved as `target-git-info-exclude-before`; SHA-256 changed from `b92bc278b3e11d7a2cbf4d4dfab43a6c917de5a90ed3191247486f4f881d3e64` to `9165df9202e691130b7031c65288afe256807a158ac17815f58fe87da6eab9bf`. The prior three owner-cache exclusions remain intact. This local preparation adjustment is recorded in `target-local-exclude.json`; no broad ignore rule or source change was added.

## Exact rollback instructions

The shared-installation script is stored beside the verified backup:

`/Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/rollback-205-13.py`

It validates pinned inventory hashes and every backup file, then checks the entire current hub/binary and every changed platform path against the observed post-publish state. It refuses later owner edits before restoring anything. It restores the previous binary, complete hub and changed platform files; deletes only recorded introduced paths; and preserves unrelated platform assets. New content in a directory it would remove also causes refusal. Six disposable-file scenarios passed, including damaged backup refusal, later-edit refusal, symlink restoration and preservation of unrelated edits.

The optional target undo is fully prepared as `target-rollback.py` in the evidence directory. It validates all 14 targets and saved bytes, requires the clean original session branch at the recorded update commit, reverts the two-file Git commit through normal hooks, and restores the twelve ignored program/installation files. Recovery journals and their exact local exclusion are retained for the audit trail. A later owner commit or edit makes it stop for review.

To restore the shared installation **and** undo this one target update, keep Aether idle and run all checks before either restore:

```sh
python3 /Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/rollback-205-13.py --check
python3 /Users/callumcowie/.aether-backups/phase205-release-20260916/target-rollback.py --check

python3 /Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/rollback-205-13.py --restore
python3 /Users/callumcowie/.aether-backups/phase205-release-20260916/target-rollback.py --restore

python3 /Users/callumcowie/.aether-backups/phase205-pre-release-20260915T213324Z/rollback-205-13.py --verify-restored
python3 /Users/callumcowie/.aether-backups/phase205-release-20260916/target-rollback.py --verify-restored

cd /Users/callumcowie/.aether-backups/phase205-release-20260916
/Users/callumcowie/.local/bin/aether version --check
/Users/callumcowie/.local/bin/aether integrity --json
git -C "/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System" status --porcelain
```

After restoration, the binary and hub must report **1.0.78**, the consumer integrity check must pass, and the project must be clean. The target's marker returns to its exact pre-update **1.0.63**. The shared script alone restores only the machine installation; both scripts undo the complete local release/update action. The pre-release 1.0.78 consumer integrity check passed before publishing. **Both live dry-run/check modes passed; no successful installation was undone.** A refused check requires review of the named changed path, not a forced overwrite.

## Durable receipts

Root: `/Users/callumcowie/.aether-backups/phase205-release-20260916/`.

The operational `evidence-manifest.json` covers **167 files**, with size and SHA-256 for each. It includes exact command/exit/log receipts, source and installation inventories, backup helpers, rollback scripts, saved target program bytes, the 70 copied update records and an exact release-source archive. Every logged command's output digest was checked. Later documentation-commit/audit receipts are supplemental and are not claimed to be inside this frozen operational manifest.

| Receipt, relative to that durable root | SHA-256 |
|---|---|
| `evidence-manifest.json` | `8dd9fe70625b8253d456d967eff103c5478c638ff47b7ab70834578c26d4635d` |
| `source-proof.json` | `82460b75eb157f747c8463c16d1bab437339412c20b0b3bab5993764495d24ec` |
| `installation-before.json` | `a0011255d647ec563be2dd7e99aafefa5e8a333e5691e93401ec5743b4aa0716` |
| `installation-after.json` | `b7077216c21fde5b7c487530e4831725e698e036e363f67492bcf3ca4b9c4063` |
| `installation-delta.json` | `41db4d6529b8fdb9ca11869fd0045ccd8bb625f3dc3027a2e70c61004bf78022` |
| `publish-verification.json` | `2c1f1aa8ca2628c7106915fc3616ed7c96e6a5d0275346ddbc3c34fb0df18737` |
| `platform-verification.json` | `9b5151430334aa17dbb4c7913f93674a3aa732aa2d7eaf6bf580cb71a74ceed7` |
| `receipts/publish.log` | `3eeaac258f282b468b745b35e0579f832d731db49d41bfaed1b96fa82e9a995d` |
| `receipts/integrity-after.log` | `4ebb2c71e92eea21997cf879d40032d6cb8f37cd94cfe7ad3e14f15cc12cfece` |
| `receipts/version-check-after.log` | `0c6240b875fedfd43642140f71a6b6d74f16e5d1d26754aa23dbd8278d40b50d` |
| `target-before.json` | `5e8d7c0cec3fc6ef2e96e0fbbb05067ec9b446bd28f1ab71730721785cb2038f` |
| `target-after.json` | `c20f4c1d4fbebf3aafaf040099baaae1b16024103f2f382031b6ef71fb2e39db` |
| `receipts/target-update.log` | `943c8ac80319ec2c2e93c5483729d9f39c647e1eb6d4c0473bd01522771c1085` |
| `target-update-records-manifest.json` | `52f933e39fb482599aa89f4dff405cb2d29a02fc111f81c0d2d037edaf336a83` |
| `target-final.json` | `43e36766da57d26095cc33499d32891669d2f31e3dedddfa95daddb9fdca1490` |
| `rollback-205-13.py` | `3f415c5c98dec28d5a20fd437af17b25ec015c59bab073048422774fce57915a` |
| `target-rollback.py` | `9f2df11e0125991e615f0c126f32a4a936d6b7669f0aeb3cca3b5f860001525c` |
| `release-source.tar.gz` | `7f8a450148366b826aa6b40908c52d8033b886184fc553bb731a20db09d04d62` |

## Remaining limits and follow-up

- The owner's ten-task walkthrough, timing judgments and overall acceptance remain pending. `requirements-completed` stays empty in the plan summary; neither the phase nor owner session is marked complete.
- The unchanged 17-test baseline and 34 open planning-register entries remain recorded. This release does not waive them.
- The publisher's npm install reported a high-severity advisory for existing **`js-yaml` 4.3.1**, [GHSA-2883-xcg3-v3hh](https://github.com/advisories/GHSA-2883-xcg3-v3hh), concerning CPU exhaustion from YAML merge sources. A read-only `npm audit --json` confirmed one high-severity finding and exited 1; npm reports a fix in 4.3.2. The tested dependency lockfile was not changed. `receipts/npm-audit-diagnostic.log` retains the finding for a separately verified follow-up.
- The existing publisher preserves twenty local Codex agent variants and omits per-platform count output. Both are disclosed above; the hub/source content and measured installed counts are verified.

The stale `VERSION` assumption, omitted native counters, managed marker classification, and exact local journal exclusion are the execution deviations. They are documented without claiming additional runtime fixes or owner acceptance.
