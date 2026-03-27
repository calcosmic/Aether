---
phase: 27-deterministic-fallback-dedup
verified: 2026-03-27T14:30:00Z
status: passed
score: 11/11 must-haves verified
---

# Phase 27: Deterministic Fallback + Dedup Verification Report

**Phase Goal:** Add a deterministic fallback for builder learning extraction. When AI agents skip learning output, extract learnings from git diff + test results. Also add content normalization to instinct deduplication so semantically similar instincts consolidate (not just SHA-256 exact match).
**Verified:** 2026-03-27T14:30:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| #   | Truth   | Status     | Evidence       |
| --- | ------- | ---------- | -------------- |
| 1   | Creating instinct with "when implementing tests" + existing "when writing tests" consolidates | VERIFIED | `_jaccard_similarity` called on both trigger and action in `_instinct_create` (learning.sh:1567-1568), threshold 0.80, merges via `_state_mutate` (learning.sh:1610-1626). Test `fuzzy dedup: similar trigger+action merges into single instinct` passes. |
| 2   | Content normalization handles whitespace, casing, punctuation | VERIFIED | `_normalize_text` (learning.sh:1391-1432): lowercase via `tr`, strip punctuation via `tr -cd`, collapse whitespace via `awk '{$1=$1};1'`. Test `normalize_text: casing and punctuation normalized` passes. |
| 3   | Synonym substitution maps implementing/creating/building to writing | VERIFIED | awk associative array at learning.sh:1408-1414 maps implementing/creating/building/implement/create/build/write -> writing, plus tests/checking/verifying -> testing, fixing/repairing/patching/fix/repair/patch/resolve -> resolving. Stop words when/while/during/before/after stripped. |
| 4   | Both trigger AND action must independently exceed 80% Jaccard for merge | VERIFIED | `if (( $(echo "$ic_trig_sim >= 0.80" | bc -l) )) && (( $(echo "$ic_act_sim >= 0.80" | bc -l) ))` at learning.sh:1571. Test `fuzzy dedup: only trigger matches does not merge` passes. |
| 5   | Merged instinct averages confidences and keeps longer text | VERIFIED | Confidence averaged via `bc -l` (learning.sh:1595), longer trigger/action kept via `${#ic_trigger} -gt ${#ic_merged_trigger}` (learning.sh:1598-1601). Test `fuzzy dedup: keeps longer text on merge` passes. |
| 6   | When builder produces empty learning.patterns_observed, fallback extracts at least one learning from git diff | VERIFIED | `_learning_extract_fallback` (learning.sh:1747-1933) reads `git diff --stat HEAD~1`, filters noise, categorizes, feeds through instinct-create. Test `fallback fires when no learnings exist` passes. |
| 7   | Fallback produces at most 5 learnings per build | VERIFIED | `lef_categories` capped at 5 via jq `.[:5]` (learning.sh:1854). Loop guard `i<5` at learning.sh:1866. Test `fallback respects 5-learning cap` passes. |
| 8   | Fallback learnings go through instinct-create pipeline | VERIFIED | `bash "$0" instinct-create --trigger ... --action ... --confidence 0.5 --domain ... --source "fallback-phase-$lef_phase" --evidence ...` at learning.sh:1920-1926. |
| 9   | Fallback skips trivial changes (whitespace, package-lock, .aether/data/) | VERIFIED | jq filter at learning.sh:1806-1832: `select((.path | startswith(".aether/data/")) | not)`, `select(.path != "package-lock.json")`, `select((.abs_net >= 3) or (.is_test == true))`. Test `fallback skips trivial changes` passes. |
| 10   | Fallback only fires when patterns_observed is empty | VERIFIED | continue-advance.md Step 2.4 (line 59-69): checks `patterns_count=$(jq ...)` and only calls `learning-extract-fallback` when `patterns_count -eq 0`. |
| 11   | Continue output shows fallback count | VERIFIED | continue-advance.md echoes `fallback_count=$fallback_count` (line 69). continue-finalize.md captures it and appends `($fallback_count from fallback)` to wisdom_parts (lines 97-101). |

**Score:** 11/11 truths verified

### Required Artifacts

| Artifact | Expected    | Status | Details |
| -------- | ----------- | ------ | ------- |
| `.aether/utils/learning.sh` - `_normalize_text` | Text normalization helper | VERIFIED | Lines 1391-1432. 42 lines. Full implementation: lowercase, strip punctuation, collapse whitespace, synonym sub (16 entries), stop word removal. |
| `.aether/utils/learning.sh` - `_jaccard_similarity` | Word-level Jaccard helper | VERIFIED | Lines 1440-1481. 42 lines. Full implementation: normalizes both texts, awk with NUL delimiter, computes intersection/union, printf format. |
| `.aether/utils/learning.sh` - fuzzy dedup in `_instinct_create` | Fuzzy dedup logic | VERIFIED | Lines 1547-1630. Iterates existing instincts, computes Jaccard for trigger+action, merges at 0.80 threshold with averaged confidence, longer text, appended evidence. |
| `.aether/utils/learning.sh` - `_learning_extract_fallback` | Git-diff-based extraction | VERIFIED | Lines 1747-1933. 187 lines. Pre-flight guards, git diff parsing, noise filtering, jq categorization/grouping, instinct-create feed-through, 5-learning cap. |
| `.aether/aether-utils.sh` - subcommand registration | `learning-extract-fallback` case | VERIFIED | Line 3940: `learning-extract-fallback) _learning_extract_fallback "$@" ;;` |
| `.aether/docs/command-playbooks/continue-advance.md` | Fallback wiring Step 2.4 | VERIFIED | Lines 52-74: checks patterns_count, calls learning-extract-fallback when empty, echoes fallback_count for cross-stage capture. |
| `.aether/docs/command-playbooks/continue-finalize.md` | Fallback count in wisdom summary | VERIFIED | Lines 97-101: captures fallback_count, appends "(N from fallback)" to wisdom_parts. |
| `tests/integration/instinct-pipeline.test.js` | 5 fuzzy dedup tests | VERIFIED | Lines 585-799: merge, below-threshold, partial-match, normalization, longer-text tests. All pass. |
| `tests/integration/fallback-extraction.test.js` | 6 fallback tests | VERIFIED | Lines 110-307: fires-on-empty, skips-trivial, 5-cap, test-exception, no-git, no-colony tests. All pass. |

### Key Link Verification

| From | To  | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| `_normalize_text` | `_jaccard_similarity` | Function call | WIRED | `_jaccard_similarity` calls `_normalize_text` on both inputs (learning.sh:1446-1447) |
| `_jaccard_similarity` | `_instinct_create` | Similarity check after exact match fails | WIRED | `_instinct_create` calls `_jaccard_similarity` for both trigger and action (learning.sh:1567-1568), checks 0.80 threshold (line 1571) |
| continue-advance.md Step 2.4 | `_learning_extract_fallback` | Subprocess call | WIRED | `bash .aether/aether-utils.sh learning-extract-fallback` at continue-advance.md line 64 |
| `_learning_extract_fallback` | `_instinct_create` | instinct-create subprocess | WIRED | `bash "$0" instinct-create ...` at learning.sh:1920-1926 |
| continue-finalize.md | `fallback_count` variable | Cross-stage echo pattern | WIRED | `fallback_count="${fallback_count:-0}"` at continue-finalize.md line 98, output appended to wisdom_parts at line 100 |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| PIPE-03 | 27-02 | Builder learning extraction has a deterministic fallback (git-diff-based) when AI agents skip learning output | SATISFIED | `_learning_extract_fallback` function exists, registered as subcommand, wired into continue-advance Step 2.4, fires when patterns_observed empty. 6 integration tests pass. |
| VAL-02 | 27-01 | Instinct deduplication uses content normalization (not just SHA-256 exact match) so semantically similar instincts consolidate | SATISFIED | `_normalize_text` + `_jaccard_similarity` helpers exist, fuzzy dedup in `_instinct_create` at 0.80 threshold for both fields. 5 integration tests pass. |

No orphaned requirements found. Both PIPE-03 and VAL-02 are claimed by plans and implemented.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| None found | - | - | - | - |

No TODOs, FIXMEs, placeholders, empty implementations, or console.log-only stubs found in any modified files.

### Human Verification Required

### 1. End-to-end: Fallback fires in real colony flow

**Test:** Run `/ant:build` then `/ant:continue` in a colony where the builder does not produce learning output.
**Expected:** The continue output includes "(N from fallback)" in the wisdom summary, and instincts appear in COLONY_STATE.json derived from git diff.
**Why human:** Requires a real colony session with an AI builder that skips learning output -- cannot be fully simulated programmatically.

### 2. Fuzzy dedup effectiveness on real-world phrasing

**Test:** Observe instincts created across multiple phases in a real colony. Check whether semantically similar instincts (e.g., "when writing API endpoints" vs "when implementing REST routes") consolidate.
**Expected:** Similar instincts merge rather than proliferate.
**Why human:** Real-world phrasing is more varied than test inputs. The 0.80 Jaccard threshold may be too strict or too loose depending on actual usage patterns.

### Gaps Summary

No gaps found. All must-haves from both plans (27-01 and 27-02) are verified:
- Text normalization pipeline is complete and working (VAL-02)
- Fuzzy dedup with 0.80 Jaccard threshold is wired into instinct-create (VAL-02)
- Deterministic fallback extraction from git diff is implemented and registered (PIPE-03)
- Fallback is wired into continue-advance playbook, fires only when AI learnings empty (PIPE-03)
- Fallback count displayed in continue-finalize wisdom summary (PIPE-04 partial)
- All 11 integration tests pass (5 fuzzy dedup + 6 fallback extraction)
- All 4 commit hashes verified as valid in git history
- No anti-patterns detected

---

_Verified: 2026-03-27T14:30:00Z_
_Verifier: Claude (gsd-verifier)_
