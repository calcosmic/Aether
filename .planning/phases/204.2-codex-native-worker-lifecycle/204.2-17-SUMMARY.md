---
phase: 204.2-codex-native-worker-lifecycle
plan: "17"
subsystem: testing
tags: [admission, storage, evidence, python, gap-closure]
requires:
  - phase: 204.2-16
    provides: Immutable failed race preflight and historical capacity evidence
provides:
  - Candidate admission controls for source and both regression lanes
  - Recorded storage admission and first-failure retention
  - Halted preparation and unexecuted tracer receipts
affects: [204.2-18, 204.2-22, 204.2-23]
tech-stack:
  added: []
  patterns: [stdlib behavioral controls, fail-closed input admission, immutable failed receipts]
key-files:
  created:
    - scripts/test_qualify_native_gap_regression.py
    - .planning/phases/204.2-codex-native-worker-lifecycle/evidence/gap-closure-2/admission-preflight.json
    - .planning/phases/204.2-codex-native-worker-lifecycle/evidence/gap-closure-2/ordinary-tracer.json
  modified:
    - scripts/qualify-native-gap-regression.py
key-decisions:
  - Refuse ignored files actually consumed by go:embed; do not silently delete or omit them.
  - Missing measured storage inputs prevent admission even when currently available bytes exceed the prior exhausted floor.
requirements-completed: []
requirements-addressed: [ANT-04, ANT-05, ANT-07]
status: halted
qualification: incomplete
tasks-completed: 1
tasks-total: 3
completed: 2026-09-18
---

# Phase 204.2 Plan 17: Candidate admission and resource gate Summary

**Ten Python controls pass, but actual preparation stops before copying or building because an ignored embedded file is unadmitted and the storage estimate is incomplete. The ordinary Builder tracer was not started.**

In plain English: the test harness now checks that it knows every file the program would include. That caught a real hidden file in this checkout. It also refuses to guess how much disk a fresh build needs. These are useful safety repairs, not proof that the native-worker phase passes.

## Task disposition

1. **Input admission implemented and unit verified.** Unknown untracked paths are refused before preparation and in source/normal/race validation. Ignored compilation inputs and embedded assets are checked too. Tracked dirty changes and deletions alter identity. Build-relevant symlinks are refused; three existing metadata-only to-prd link paths retain link-target hashes. Exact existing GSD metadata paths are excluded only outside embedded inputs. Each of the four suite launches revalidates immediately before and afterward.
2. **Storage/retention implementation verified; actual normal/race preflight incomplete.** Measurement precedes module copying, lane creation and linking. Tests prove low-space refusal launches nothing, uncertain measurements never pass, and ENOSPC remains the primary diagnostic after a subsequent inventory error. Actual admission never reached copying, cloning, fixture execution or discovery.
3. **Not executed.** The required passing preflight is absent. No real host, Builder, check, terminal or credit was captured. The new ordinary receipt explicitly records not_attempted.

## Verification and commits

`python3 -m unittest discover -s scripts -p 'test_qualify_native_gap_regression.py' -v`: **10 tests passed**. Real disposable Git inventories exercise source and both prepared lanes with added production, test and embed files; a mocked launch sentinel proves none starts. No dependency was installed. Python tests have no Go race variant. The planned actual Go normal/race fixture controls remain unexecuted behind admission.

- `8ed8af8a`: Task1 RED controls (5 tests, 7 failing assertions and 2 errors).
- `17e38d46`: Task1 GREEN candidate admission and immediate launch validation.
- `278167c6`: Task2 RED storage/first-failure controls (2 expected missing-implementation errors).
- `dc062a2a`: Task2 storage admission, first-failure retention, and strengthened admission integration controls.
- `ea8d0b51`: Explicit metadata treatment and durable uncertain-capacity measurement.
- `0651a293`: Failed admission and blocked-tracer receipts.

`git diff --check` passed. Source report revision is `ea8d0b51`; each failed attempt retains its actual revision in the manifest. No captured binary is relabelled: no new candidate binary was built. Reporting commits do not establish a frozen passing candidate.

## Measured blockers and retained evidence

External root: `/tmp/aether-gap2-kw0zk0jl`. Three unique immutable preparation roots retain every failed attempt:

- `admission-preflight`: refused existing unclassified `.gsd/scratch/blocker1.txt`.
- `admission-preflight-2`: refused existing metadata symlink `.claude/skills/to-prd`.
- `admission-preflight-3`: refused **`.codex/.DS_Store`**, an ignored file actually consumed by `embedded_assets.go:13` through `all:.codex`.

The first two diagnostics led to narrowly enumerated non-build metadata exceptions after inspecting current inputs. The third is a genuine consumed-file barrier and was not waived. Its SHA-256 is `a53a881ea87d3cdc683b6e5d42afddd2e906913a853ed6d43086fd2c7b2dd61d`. The file remains untouched. No candidate can be considered admitted until it is explicitly pinned/copied or excluded from actual build inputs through an authorized source change.

The separate read-only `storage-estimate.json` records **6,548,320,256 available bytes** at measurement, **6,050,009,088 bytes** as the historical capacity that already exhausted during Plan16, **1,737,657,830 module-copy bytes**, **552,519,983 standalone-clone bytes**, and **51,789,154 candidate-binary bytes**. The previous normal `go-cache` is unavailable. Therefore required capacity is **unknown**, not 6.05 GB; that historical value is only a failed lower bound. Earlier free space was approximately 5.28 GB. No claim of adequate capacity follows from changing ambient free space.

No current-task compiler/module caches were created, so there is no useful owned cache to reclaim from these attempts. No historical evidence, shared cache or unrelated process was removed. Resuming requires resolving the consumed-input blocker and supplying complete measured capacity evidence with sufficient isolated storage, then a fresh preparation root. Plans18–23 remain dependent on this halted plan.

The checked-in `admission-preflight.json` pins all three manifests, exact argv/exits, stdout/stderr, test log, storage measurement and preservation receipt. `ordinary-tracer.json` links that receipt and explicitly leaves capture identity, binary, Builder, terminal and credit absent. No full regression suite was launched.

## Deviations and remaining obligations

Root owns shared trackers, so this executor did not update STATE.md, ROADMAP.md or requirements. The plan's bounded failure path was taken: Task2 acceptance and Task3 remain incomplete. Three retained attempts classified existing metadata and then stopped at the genuine input barrier. No package installs, shared installation/hub/configuration changes, publication, login, fresh credentials, owner walkthrough or Phase205 changes occurred.

Historical source identities and all original evidence remain unchanged. The prior real-home NPM incident remains unresolved: **31 additions, 64 changes and 17 removals retain unestablished prior state**. The shared installed narrow CalVault hotfix is not this candidate. No ANT requirement is complete and no previous failure is upgraded to a pass.

Stub review found no functional placeholders introduced into the harness. Null tracer fields and unknown required capacity are deliberate missing-evidence markers, not stub implementations. No new network/auth surface is introduced; local file inspection remains the harness's existing trust boundary.

## Self-Check: PASSED

All listed implementation/evidence commits resolve, all four declared artifacts exist, and ten final controls pass. `/tmp/aether-gap2-kw0zk0jl/plan17-integrity.json` verifies **97 protected files** and the root's **123 extended inventory entries**, including executable hashes, with zero mismatches. This check confirms retained changes and evidence integrity only; **plan execution remains halted and native qualification remains incomplete**.
