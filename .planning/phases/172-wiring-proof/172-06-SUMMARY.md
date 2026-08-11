---
phase: 172-wiring-proof
plan: 06
subsystem: testing
tags: [go, cobra, ci, fence-parsing, wiring-audit]

# Dependency graph
requires:
  - phase: 172-00
    provides: "the cobra-contract audit (extractDocumentedCalls, validateCallAgainstCobra) and the first --limit flag fixes"
  - phase: 172-03
    provides: "the wiring-gate CI step and TestWiringGateStepRunsEveryWiringTest, which enforces every named guard test is actually run in CI"
  - phase: 172-05
    provides: "the phase's prior wiring-proof plans, immediately preceding this gap-closure plan"
provides:
  - "A fence-tolerant extractDocumentedCalls that no longer desyncs on a closing marker glued to trailing content"
  - "Two corrected invocations (midden-recent-failures --limit 50; generate-commit-message flag form) that were previously invisible to the audit"
  - "Fourteen repaired glued fence markers across seven playbook files"
  - "TestExtractorDoesNotDesyncOnGluedFenceMarker pinning the tolerant extractor against desync and against a toggle-and-skip shortcut"
  - "TestAuditedCorpusHasNoGluedFenceMarkers, a corpus-wide sweep sharing auditedFilePaths with the audit itself"
  - "Two deliberate D-01 enrichment classifications (backup-prune-global, temp-clean)"
affects: [wiring-proof, "172-*", future CI-gate phases that touch command_call_audit_test.go]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared per-line extraction body (processDocumentedCallLine) called from both the ordinary and glued-marker code paths, so a fence-toggle change cannot fork into two divergent copies of the backtick-span loop"
    - "Regex/marker literals built by string concatenation (following buildConstraintRe) when a guard test's own source could otherwise resemble the shape it scans for"
    - "Corpus-wide sweep tests share their file enumeration with the function they are proving equivalent to (auditedFilePaths), so the sweep cannot silently drift from what the audit actually reads"

key-files:
  created: []
  modified:
    - cmd/command_call_audit_test.go
    - .aether/docs/command-playbooks/continue-full.md
    - .aether/docs/command-playbooks/continue-advance.md
    - .aether/docs/command-playbooks/continue-finalize.md
    - .aether/docs/command-playbooks/build-full.md
    - .aether/docs/command-playbooks/build-wave.md
    - .aether/docs/command-playbooks/build-verify.md
    - .aether/docs/command-playbooks/build-complete.md
    - .github/workflows/ci.yml
    - .planning/phases/172-wiring-proof/deferred-items.md

key-decisions:
  - "Fixed the two hidden instructions (bare positional to midden-recent-failures; five-positional call to generate-commit-message) against their commands' real cobra contracts, per D-14, since 172-00 already made the identical ruling at four sibling call sites"
  - "backup-prune-global and temp-clean classified as D-01 enrichment, not gate: both are housekeeping/cleanup commands whose failure degrades tidiness only — no verification result, security scan, or gate outcome depends on either"
  - "The fence-toggle tolerance processes content before a glued marker under the CURRENT fence state via one shared helper, rather than a toggle-and-skip shortcut, because the cheap shortcut silently drops real invocations glued to a closing marker (proven by the mandatory second red proof)"

patterns-established:
  - "A marker anywhere on a line (not just as the line's first non-whitespace content) toggles fence state after processing what precedes it, bounded explicitly at exactly one toggle per line — not a general CommonMark parser"

requirements-completed: [WIRE-03]

duration: 55min
completed: 2026-08-11
---

# Phase 172 Plan 06: Fence-Tolerant Wiring Audit Summary

**Fixed a fence-parser blind spot that let a real cobra-contract violation (`midden-recent-failures 50`) sit invisible inside the audited corpus for the whole phase, then closed the defect class with a fence-tolerant extractor and a corpus-wide sweep test.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-11T18:16:00Z
- **Completed:** 2026-08-11T19:11:54Z
- **Tasks:** 3
- **Files modified:** 9 (excludes this SUMMARY)

## Accomplishments

- `continue-full.md:1243`'s bare positional to `midden-recent-failures` (a `cobra.NoArgs` command) and `:1531`'s five-positional call to `generate-commit-message` are both corrected against the commands' real contracts, matching sibling call sites already fixed elsewhere in the repo.
- `extractDocumentedCalls` now tolerates a closing (or opening) triple-backtick marker glued to the end of a content line instead of desyncing its in-fence/out-of-fence parity for the rest of the file — the root cause that hid `continue-full.md:1243` from the audit.
- All fourteen glued markers across seven playbook files are repaired (split onto their own line), and a new test, `TestAuditedCorpusHasNoGluedFenceMarkers`, fails naming file and line if the shape ever returns anywhere in the audited corpus.
- The audited invocation count rose from 954 → 988 as the fourteen previously hidden lines became visible, and the count is provably unaffected by whether the fix is the tolerant extractor or the repaired markdown (equivalence proof, both measured at 988).

## Task Commits

1. **Task 1: Fix the two invocations and classify the two commands the audit is about to see** - `8d180708` (fix)
2. **Task 2: Make the fence toggle see a marker glued to trailing content, and pin it** - `e4a082e5` (feat)
3. **Task 3: Repair all fourteen glued markers, and make the sweep a test rather than a one-off pass** - `c798e5e5` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified

- `cmd/command_call_audit_test.go` - Reworked `extractDocumentedCalls`'s fence toggle into three cases (no marker / marker-first / marker-glued) sharing one `processDocumentedCallLine` helper; added `TestExtractorDoesNotDesyncOnGluedFenceMarker`; factored `auditedFilePaths` out of `collectDocumentedCalls`; added `TestAuditedCorpusHasNoGluedFenceMarkers`; classified `backup-prune-global` and `temp-clean` in `knownEnrichmentSubcommands`
- `.aether/docs/command-playbooks/continue-full.md` - Fixed the two hidden invocations; repaired 3 glued markers
- `.aether/docs/command-playbooks/continue-advance.md` - Repaired 1 glued marker (closes the 172-00 deferred item)
- `.aether/docs/command-playbooks/continue-finalize.md` - Repaired 2 glued markers
- `.aether/docs/command-playbooks/build-full.md` - Repaired 3 glued markers
- `.aether/docs/command-playbooks/build-wave.md` - Repaired 2 glued markers
- `.aether/docs/command-playbooks/build-verify.md` - Repaired 2 glued markers
- `.aether/docs/command-playbooks/build-complete.md` - Repaired 1 glued marker
- `.github/workflows/ci.yml` - Added `TestExtractorDoesNotDesyncOnGluedFenceMarker` and `TestAuditedCorpusHasNoGluedFenceMarkers` to the `Verify subcommand wiring and CLI flag contracts` step's `-run` alternation
- `.planning/phases/172-wiring-proof/deferred-items.md` - Closed the 172-00 `continue-advance.md` fence entry, naming this plan and the guard test that now prevents its return

## Decisions Made

See `key-decisions` in frontmatter. In addition:

- **`backup-prune-global`** is enrichment, not a gate: it prunes stale backup files during Step 4.5 housekeeping. Its failure means old backups linger, not that any verification, security, or gate outcome is compromised.
- **`temp-clean`** is enrichment, not a gate: it deletes temp files during the same housekeeping step. Its failure means temp files linger, not that any verification, security, or gate outcome is compromised.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' acceptance criteria and verify blocks passed on the intended implementation; no auto-fixes, scope changes, or blocking issues were required.

## Mandatory Red Proofs (from plan's `<output>` requirement)

### Task 2, Red Proof 1 — unfixed toggle (marker recognized only when it is the line's first non-whitespace content)

Command: `go test ./cmd -run TestExtractorDoesNotDesyncOnGluedFenceMarker -count=1 -v`

```
=== RUN   TestExtractorDoesNotDesyncOnGluedFenceMarker
    command_call_audit_test.go:879: extractor did not see midden-recent-failures: a closing marker glued to non-invocation content (`  --ttl "30d"` + marker) desynced fence parity and hid the fenced block that follows it; calls = [{File:.../glued.md Line:12 Raw:aether status --json Command:status Args:[--json] InFence:false}]
--- FAIL: TestExtractorDoesNotDesyncOnGluedFenceMarker (0.00s)
FAIL
```

Fails naming `midden-recent-failures`, as required.

### Task 2, Red Proof 2 — toggle-and-skip shortcut (marker recognized anywhere, but content before it discarded rather than processed)

Command: `go test ./cmd -run TestExtractorDoesNotDesyncOnGluedFenceMarker -count=1 -v`

```
=== RUN   TestExtractorDoesNotDesyncOnGluedFenceMarker
    command_call_audit_test.go:895: extractor did not see backup-prune-global: a closing marker glued directly to an invocation line must still yield that invocation, not be silently discarded; calls = [{File:.../glued.md Line:6 Raw:aether midden-recent-failures --limit 50 Command:midden-recent-failures Args:[--limit 50] InFence:true}]
--- FAIL: TestExtractorDoesNotDesyncOnGluedFenceMarker (0.00s)
FAIL
```

Fails naming `backup-prune-global`, as required. Both mutations were reverted; the test then passes with both proofs' mutations reverted.

### Task 3, Re-glue red proof

Command: `go test ./cmd -run TestAuditedCorpusHasNoGluedFenceMarkers -count=1 -v` after re-gluing `continue-full.md`'s repaired `--ttl "30d"` line back onto one line with its closing marker:

```
=== RUN   TestAuditedCorpusHasNoGluedFenceMarkers
    command_call_audit_test.go:1280: 1 line(s) glue a fence marker to trailing content — this desyncs extractDocumentedCalls' fence tracking and hides every invocation after it from the audit, silently, the way continue-full.md:1243 sat unaudited for this whole phase:
          .aether/docs/command-playbooks/continue-full.md:1194: --ttl "30d"```
    command_call_audit_test.go:1283: scanned 215 files in the audited corpus
--- FAIL: TestAuditedCorpusHasNoGluedFenceMarkers (0.01s)
FAIL
```

Fails naming `continue-full.md:1194` and the text `--ttl`, as required. Restored to the repaired state (verified byte-identical to the intended post-Task-3 state via `diff`), then re-ran green:

```
=== RUN   TestAuditedCorpusHasNoGluedFenceMarkers
    command_call_audit_test.go:1283: scanned 215 files in the audited corpus
--- PASS: TestAuditedCorpusHasNoGluedFenceMarkers (0.01s)
PASS
```

## Audited Invocation Count Progression

| Point in the plan | Count | Notes |
|---|---|---|
| End of Task 1 | 954 | Two hidden invocations fixed and two commands classified, but the extractor still cannot see any of the fourteen glued-marker regions — deliberately unchanged, proving the fix touched only content the audit was already blind to. |
| End of Task 2 | 988 | The fence-tolerant extractor now sees the fourteen previously-hidden regions, with the markdown still unrepaired. |
| End of Task 3 | 988 | Markdown repaired (all fourteen markers on their own line); count is identical to Task 2's tolerant-but-unrepaired measurement. |

**Equivalence statement:** the audited count is 988 both with the tolerant extractor reading unrepaired markdown (Task 2) and with the repaired markdown read by the same tolerant extractor (Task 3). The two are provably equivalent — the tolerance is not silently over- or under-counting relative to correct markdown.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP success criterion 3 (WIRE-03) is TRUE as stated: the audit enumerates every `aether …` invocation in its declared corpus, including the region that used to hide, and no live violation of the class it exists to catch remains anywhere inside that corpus.
- The defect class is closed by a command, not a claim: `TestAuditedCorpusHasNoGluedFenceMarkers` fails, naming file and line, if a glued marker reappears anywhere in the audited corpus, and it runs in the named CI step (`TestWiringGateStepRunsEveryWiringTest` confirms this).
- The phase's own carried-forward deferred item (`continue-advance.md`'s glued marker, flagged during 172-00) is closed, not carried further.
- This closes VERIFICATION.md gap 1 in full and is (per the plan's `depends_on`) the final plan of phase 172-wiring-proof.

## Known Stubs

None.

## Threat Flags

None — this plan's threat model (T-172-26 through T-172-30) is fully addressed by the work above; no new, unaddressed security-relevant surface was introduced.

---
*Phase: 172-wiring-proof*
*Completed: 2026-08-11*
