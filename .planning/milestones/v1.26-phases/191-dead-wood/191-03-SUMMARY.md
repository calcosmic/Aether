---
phase: 191-dead-wood
plan: 03
subsystem: cli-tooling
tags: [yaml, cli-visuals, dead-code, go, testing, caste-identity]

# Dependency graph
requires:
  - phase: 191-dead-wood (plan 01/02, same wave)
    provides: nothing shared -- 191-03 touches zero files any sibling plan touches (verified in 191-CONTEXT.md's domain/canonical_refs)
provides:
  - colony/ceremony/visuals.md deleted (ROADMAP criterion 2, loader 2 of 4)
  - cmd/visuals_config.go's loadVisualsConfig() documented as a permanently-fallback path (D-04)
  - Three permanent regression tests proving zero observable change from the deletion
  - A proven, fail-then-pass reappearance ratchet
affects: [191-07 (final verification), any future plan touching cmd/codex_visuals.go or cmd/visuals_config.go]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Frozen testdata fixture (byte-exact `cp`, verified with `diff`) as the permanent record of a deleted config file's content, read at test time via the existing *PathOverride seam -- avoids retyping large/unicode file content into a Go string literal"
    - "Forensic regression test (asserts a historical fact stays true) alongside the main before/after lock, when the file being deleted turns out to have never worked for an independent reason"

key-files:
  created:
    - cmd/testdata/visuals_config_original.md
    - cmd/testdata/visuals_config_original_parseable.md
    - cmd/visuals_config_ratchet_test.go
  modified:
    - cmd/visuals_config.go
    - cmd/codex_visuals_test.go
  deleted:
    - colony/ceremony/visuals.md

key-decisions:
  - "No fold edits were needed in cmd/codex_visuals.go: every field colony/ceremony/visuals.md defined already matched the compiled defaults exactly, for every shared key"
  - "colony/ceremony/visuals.md's YAML frontmatter had a pre-existing, independent parse defect (aether_wordmark block-literal indentation) that made loadVisualsConfig() return nil for this file unconditionally, in every dev checkout, forever -- confirmed directly, not inferred"
  - "aether_wordmark's absolute left-margin is provably unrecoverable from the broken file (the minimal valid fix changes what margin the parsed value carries); the compiled default is kept as-is since it was already the only value ever rendered"
  - "loadVisualsConfig() kept as a permanently-fallback function, not deleted, per 191-CONTEXT.md D-04's explicit discretion"

patterns-established:
  - "When a fold-and-delete target fails to parse, add a dedicated forensic test proving that fact, separate from the main before/after byte-identity lock -- do not silently let the byte-identity test pass vacuously without explaining why"

requirements-completed: []

# Metrics
duration: 40min
completed: 2026-08-21
---

# Phase 191 Plan 03: Fold-and-Delete visuals.md Summary

**Deleted `colony/ceremony/visuals.md` (the caste emoji/color/label CLI config) after discovering it had never actually loaded, in any dev checkout, ever -- a pre-existing YAML indentation bug meant the file was already 100% inert, so nothing needed folding.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-21T01:02:00Z (approx, first fixture write)
- **Completed:** 2026-08-21T01:42:00Z
- **Tasks:** 2/2 completed
- **Files modified:** 6 (1 deleted, 2 modified, 3 created)

## Accomplishments

- `colony/ceremony/visuals.md` deleted -- confirmed zero behavioral change, two independent ways (Go-level regression test and a real dev-checkout CLI capture, byte-for-byte, plain and ANSI-color output)
- Discovered and documented a genuine, previously-unknown defect: the file's `aether_wordmark` YAML block-literal scalar had inconsistent indentation that made its *entire* frontmatter document fail to parse, meaning `loadVisualsConfig()` has returned `nil` for this file unconditionally in every dev checkout since it was written -- independent of, and in addition to, this criterion's already-documented CWD-relative-only defect
- Three permanent regression tests added, each with a distinct, honest job (forensic record, before/after byte-identity lock, written-value comparison) rather than one test that would have passed vacuously without explaining why
- A proven, fail-then-pass reappearance ratchet for the deleted file

## Task Commits

Each task was committed atomically:

1. **Task 1: Baseline-capture, diff-and-fold, delete visuals.md** - `d6801efe` (feat)
2. **Task 2: Add the reappearance ratchet with fail-then-pass proof** - `90759250` (test)

_No plan-metadata commit required beyond this SUMMARY per the executor's objective (STATE.md/ROADMAP.md explicitly not updated by this plan)._

## Files Created/Modified

- `colony/ceremony/visuals.md` - Deleted. The CWD-relative caste-identity/command-emoji/wordmark/divider config loader's source file.
- `cmd/visuals_config.go` - Doc comment added to `loadVisualsConfig()` recording why the file is gone and that the function is intentionally kept as a permanently-fallback path (D-04). No logic changed.
- `cmd/codex_visuals.go` - **Not modified.** No divergent values were found to fold (see Field-by-Field Diff Table below).
- `cmd/codex_visuals_test.go` - Three new tests: `TestVisualsConfigOriginalFileHadPreexistingParseDefect`, `TestVisualsConfigFoldedDefaultsMatchOriginalFile`, `TestVisualsConfigOriginalFileValuesMatchCompiledDefaults`.
- `cmd/testdata/visuals_config_original.md` - Byte-exact frozen copy of the deleted file (made with `cp`, verified with `diff`), used as the permanent "before" input.
- `cmd/testdata/visuals_config_original_parseable.md` - The frozen copy with exactly one line changed (`aether_wordmark: |` -> `aether_wordmark: |5`, an explicit YAML block-indentation indicator) so it can actually be parsed; zero content bytes touched, verified with `diff`.
- `cmd/visuals_config_ratchet_test.go` - `TestVisualsConfigDoesNotReappear`, a Tier-1 file-existence ratchet reusing the existing `repoRootForCommandSourceTest()` helper.

## Field-by-Field Diff Table

Every field `colony/ceremony/visuals.md` defined, compared against `cmd/codex_visuals.go`'s compiled defaults, for every key the file names (not a sample):

| Field | File entries | Compiled entries | Divergent entries |
|---|---|---|---|
| `caste_emoji_map` | 26 castes | 26 (+9 curation-ant castes the file predates) | **0** |
| `caste_color_map` | 26 castes | 26 (+9 curation-ant castes) | **0** |
| `caste_label_map` | 26 castes | 26 (+9 curation-ant castes) | **0** |
| `command_emoji_map` | 86 commands | 86 (+2 commands the file predates: `abandon`, `ask`) | **0** |
| `caste_prefixes` | 23 castes, 8 prefixes each | 23 castes, identical prefixes in identical order | **0** |
| `default_prefixes` | 8 entries | 8 entries, identical order | **0** |
| `visual_divider` | `"━"` x44 + `"\n"` | identical | **0** |
| `aether_wordmark` | Same glyph sequence, same 6-line "AETHER" block-letter art, same intentional 1-column stagger on row 1 (the "A" apex) | Same art, same stagger | **Not provable as a byte value** -- see finding below; glyph *content* confirmed identical line-by-line |

Extra keys present only in the compiled Go maps (curation ants, `abandon`, `ask`) are not a divergence: `visuals.md`'s own documentation states "If a caste or command is omitted, the hardcoded default is used as a fallback," and every accessor (`fileCasteEmoji`, `commandEmoji`, etc.) already implements exactly that `(value, ok)` -> fallback pattern.

**Conclusion: zero divergent values. No fold edits to `cmd/codex_visuals.go` were needed.**

## The Parse Defect Finding

While building the before/after regression test, `loadVisualsConfig()` returned `nil` for a byte-exact copy of the real, currently-deleted file -- not the expected "loaded successfully" result. Investigating directly (not guessing) found the cause: the YAML frontmatter's `aether_wordmark: |` block-literal scalar auto-detects its indentation baseline from its first content line (6 leading spaces), but the following five lines used only 5 -- one less than that baseline. Per YAML's block-scalar rules this is invalid and ends the block mid-document; `gopkg.in/yaml.v3` then fails to parse everything after it (`yaml: line 196: did not find expected key`). Because the whole frontmatter is unmarshalled as one document, this one defect silently invalidated *every* field in the file, not only the wordmark.

This was verified directly against the real repository file (not the copy) before any edits were made, and is now locked in permanently by `TestVisualsConfigOriginalFileHadPreexistingParseDefect`.

**What this means:** `loadVisualsConfig()` has returned `nil` for this file, unconditionally, in every dev checkout, forever -- independent of, and stacked on top of, this criterion's already-documented CWD-relative-only defect. The compiled defaults in `cmd/codex_visuals.go` were already the *only* value that has ever actually rendered, anywhere, in any environment. Deleting the file does not change behavior that was ever observed by anyone.

For the six discrete, directly-comparable fields (every field except the wordmark), a second, minimally-corrected fixture (`cmd/testdata/visuals_config_original_parseable.md`, a single-line fix: `aether_wordmark: |` -> `aether_wordmark: |5`, an explicit YAML indentation indicator that touches no content byte) confirms the file's *written* values matched the compiled defaults exactly regardless -- so even setting the parse defect aside entirely, there was nothing to fold.

For `aether_wordmark` specifically: the minimal valid fix necessarily changes the value's absolute left-margin (an explicit `|5` indicator strips exactly 5 columns from every line, producing a flush-left variant), so no byte-exact reconstruction of "what the file's author intended" is possible -- only that the same characters, in the same order and the same intentional row-1 stagger (the "A" glyph's apex), are present. `TestVisualsConfigOriginalFileValuesMatchCompiledDefaults` asserts this glyph-content equality explicitly and documents why absolute margin is not asserted. The compiled default's margin (already the only value ever rendered) is kept exactly as it was.

## Before/After Proof

### Layer 1 -- Go-level regression test (permanent)

`TestVisualsConfigFoldedDefaultsMatchOriginalFile` renders every caste's emoji/ANSI-color/label/composed-identity, every command's emoji, every caste's name-prefix list (5 seeds each) plus the default-prefix fallback, the wordmark, and the divider -- once with the frozen original file loaded via `visualsPathOverride`, once with the file entirely absent -- and asserts every one of the 700+ rendered entries is byte-identical. Key sets are read from the compiled Go maps directly (not hardcoded), so the test cannot silently shrink into a sample.

```
=== RUN   TestVisualsConfigOriginalFileHadPreexistingParseDefect
--- PASS: TestVisualsConfigOriginalFileHadPreexistingParseDefect (0.00s)
=== RUN   TestVisualsConfigFoldedDefaultsMatchOriginalFile
--- PASS: TestVisualsConfigFoldedDefaultsMatchOriginalFile (0.00s)
=== RUN   TestVisualsConfigOriginalFileValuesMatchCompiledDefaults
--- PASS: TestVisualsConfigOriginalFileValuesMatchCompiledDefaults (0.00s)
```

### Layer 2 -- real dev-checkout CLI capture

Built the real `aether` binary and ran `aether ceremony spawn-plan --workflow build --manifest-file <manifest covering 11 castes>` from this repo's actual root (the only environment where the CWD-relative `colony/ceremony/visuals.md` path ever resolves at all), twice: once with the file temporarily restored from the verified-identical frozen fixture, once with it genuinely deleted. Each run used its own fresh, isolated `COLONY_DATA_DIR` (see Methodology Note below) so the comparison is a true apples-to-apples single-variable change (file present vs. absent).

Plain output diff:
```
$ diff clean-before.txt clean-after.txt
$ echo $?
0
```

ANSI-color output diff (confirms `caste_color_map` end-to-end, not just the stripped-color path):
```
$ diff clean-before-color.txt clean-after-color.txt
$ echo $?
0
```

Both empty -- byte-for-byte identical. Sample of the captured output (file-present run, identical to file-absent run):

```
━━ 🐜 W E L C O M E   T O   A E T H E R ━━
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
...
━━━ 🔨 S P A W N   P L A N ━━━

👑🐜 Queen Orchestration: workflow=build
...
  🔨🐜 Builder [sonnet] Hammer-1  Task 1.1  Fold visuals.md into codex_visuals.go
  👁️🐜 Watcher [sonnet] Keen-2  Task 1.2  Verify the fold is byte-identical
  🔍🐜 Scout [sonnet] Swift-3  Task 1.3  Research the loader call sites
  🏛️🐜 Architect [opus] Blueprint-4  Task 1.4  Design the fold-and-delete process
  🔮🐜 Oracle [opus] Sage-5  Task 1.5  Deep research the dev-checkout resolution
  ⚔️🐜 Gatekeeper [opus] Guard-6  Task 1.6  Security gate the deletion
  🩹🐜 Medic [sonnet] Heal-7  Task 1.7  Diagnose colony health post-deletion
  📦🐜 Porter [sonnet] Carry-8  Task 1.8  Publish the fold to the hub
  🏺🐜 Archaeologist [opus] Relic-9  Task 1.9  Excavate the visuals.md history
  🔧🐜 Fixer [sonnet] Chip-10  Task 1.10  Autonomously repair any fold mismatch
...
  👥🐜 Auditor [opus] Review-11  Task verify  Independent quality gate
```

### Reappearance ratchet -- fail-then-pass proof

```
$ go test ./cmd/ -run TestVisualsConfigDoesNotReappear -v   # before recreating the file
--- PASS: TestVisualsConfigDoesNotReappear (0.00s)

$ cp cmd/testdata/visuals_config_original.md colony/ceremony/visuals.md   # temporarily recreate
$ go test ./cmd/ -run TestVisualsConfigDoesNotReappear -v
    visuals_config_ratchet_test.go:34: colony/ceremony/visuals.md has reappeared -- this file
    was ruled zero-behavioral-effect and deleted in Phase 191 plan 03 (ROADMAP criterion 2, the
    third of the four CWD-relative silent-fallback loaders); if it is back, either it has a new
    reader (update the fold in cmd/codex_visuals.go with the evidence and this ratchet) or it
    should be deleted again
--- FAIL: TestVisualsConfigDoesNotReappear (0.00s)

$ rm colony/ceremony/visuals.md   # remove again
$ go test ./cmd/ -run TestVisualsConfigDoesNotReappear -v
--- PASS: TestVisualsConfigDoesNotReappear (0.00s)
```

## Full Verification

```
$ go build ./...              # exit 0
$ go vet ./...                 # exit 0
$ go test ./cmd/ -run 'TestVisualsConfig|TestCasteIdentity|TestFileCaste' -v -count=1
    12 tests, all PASS, including TestCasteIdentityUsesHouseStyle unmodified
$ go test ./cmd/... -count=1   # ok, run 3 times across the plan (pre-deletion, post-Task-1,
                                 post-Task-2), all clean, zero regressions
```

## Decisions Made

- **No fold edits needed.** Every field the file defined matched the compiled defaults exactly for every shared key -- the diff-and-fold process was followed faithfully, and its honest conclusion was "nothing diverges."
- **Wordmark margin kept as compiled, not reconstructed.** Given the file's own parse defect makes its intended absolute indentation genuinely unrecoverable, the already-live compiled default was retained unchanged rather than guessing at an "intended" value.
- **Three tests instead of one.** A single before/after test using the true broken original would have passed *vacuously* (both sides hit the same nil-config fallback, proving nothing about fold-correctness on its own). Splitting into a forensic test (proves the historical parse failure), the main before/after lock (proves deletion changes nothing, honestly grounded in why), and a written-values test (proves there was nothing to fold, independent of the parse defect) avoids a test that passes for the wrong reason -- directly the CLAUDE.md Definition of Done concern this repo's own history warns about.
- **`loadVisualsConfig()` kept, not deleted**, per 191-CONTEXT.md's D-04 explicit discretion; only a doc comment was added recording why the file is gone.

## Deviations from Plan

None that required Rule 1-4 auto-fixes to any production code path -- `cmd/codex_visuals.go` needed zero edits. The one substantive addition beyond the plan's literal text is the parse-defect discovery and its dedicated forensic test, which is squarely inside the plan's own instruction to "read colony/ceremony/visuals.md field by field" and "prove the fold is complete" -- it is the accurate result of doing that reading rigorously, not a scope change.

### Methodology Note (not a code deviation)

The first CLI baseline-capture attempt showed a spurious one-line diff (a first-run "Welcome to Aether" banner present in the "before" capture, absent in the "after" capture). This traced to `checkAndEmitFirstRun` writing a `.aether/data/.welcomed` marker file on first invocation, and my second capture reusing the same real repo `.aether/data/` directory the first one had already "used up." Unrelated to the fold. Resolved by pointing each capture at its own fresh, isolated directory via the existing `COLONY_DATA_DIR` environment variable (`pkg/storage.ResolveDataDir`) -- CWD stayed the real repo root throughout, so the actual `colony/ceremony/visuals.md` CWD-relative resolution under test was unaffected. A stray `.aether/data/.welcomed` file from an earlier, uncontrolled capture attempt remains in the local working tree; it is gitignored (`.aether/.gitignore` covers `data/`), untracked, and harmless.

## Issues Encountered

None blocking. The parse-defect discovery (above) required redesigning the test approach mid-task but did not require any user decision or architectural change.

## User Setup Required

None - no external service configuration required.

## Known Stubs

None.

## Next Phase Readiness

- `colony/ceremony/visuals.md` is gone; `cmd/codex_visuals.go`'s compiled defaults are confirmed the sole source of caste-identity/command-emoji/wordmark/divider rendering, proven two independent ways.
- Sibling loader plans (191-02: colony-prime.md + dispatch-contract.yaml; 191-04: review-depth.yaml) touch disjoint files and were not affected by this plan's work.
- 191-07 (final verification, Wave 2) can proceed once all Wave-1 plans land; nothing in this plan touched `colony/policies/oracle-phase-directives.yaml` or any auxiliary command.
- No blockers.

---
*Phase: 191-dead-wood*
*Completed: 2026-08-21*

## Self-Check: PASSED

- FOUND: cmd/visuals_config.go, cmd/codex_visuals.go, cmd/codex_visuals_test.go
- FOUND: cmd/testdata/visuals_config_original.md, cmd/testdata/visuals_config_original_parseable.md, cmd/visuals_config_ratchet_test.go
- FOUND: .planning/phases/191-dead-wood/191-03-SUMMARY.md
- CONFIRMED DELETED: colony/ceremony/visuals.md
- FOUND commit: d6801efe (Task 1)
- FOUND commit: 90759250 (Task 2)
