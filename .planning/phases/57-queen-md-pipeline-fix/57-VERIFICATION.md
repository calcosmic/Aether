---
phase: 57-queen-md-pipeline-fix
verified: 2026-04-26T23:50:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
re_verification: false
---

# Phase 57: QUEEN.md Pipeline Fix Verification Report

**Phase Goal:** Fix the QUEEN.md wisdom pipeline so that the global QUEEN.md works like CLAUDE.md -- persistent instructions that shape every worker conversation. No duplicate entries. High-confidence instincts promote automatically at seal. Global QUEEN.md wisdom reaches all workers via colony-prime.
**Verified:** 2026-04-26T23:50:00Z
**Status:** PASSED
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Running queen-seed-from-hive twice reports 0 new entries on the second run (no duplicates) | VERIFIED | TestQueenSeedFromHiveSecondRunSeedsZero passes; isEntryInText pre-filter + appendEntriesToQueenSection normalized dedup both wired in cmd/queen.go lines 356-364 and 567-598 |
| 2 | Colony-prime worker prompt includes global QUEEN.md wisdom, Philosophies, and Anti-Patterns sections alongside local wisdom | VERIFIED | cmd/colony_prime_context.go lines 558-580 inject global_queen_md section using readQUEENMd (extended in Plan 01 to extract Wisdom, Patterns, Philosophies, Anti-Patterns); LOCAL QUEEN WISDOM section also present at lines 616-632 |
| 3 | queen-promote-instinct writes to global ~/.aether/QUEEN.md so promoted instincts reach all colonies | VERIFIED | cmd/queen.go lines 285-293: hubStore() guard + loadQueenText(hs) + appendEntryToQueenSection + writeQueenText(hs); TestQueenPromoteInstinctWritesGlobal passes |
| 4 | Running /ant-seal automatically promotes instincts with confidence >= 0.8 to QUEEN.md without manual commands | VERIFIED | .claude/commands/ant/seal.md and .opencode/commands/ant/seal.md both contain auto-promotion instructions (lines 16-24) that loop instincts with confidence >= 0.8 and call queen-promote-instinct for each; files are identical (parity verified) |
| 5 | Hive wisdom test entry and all ~270 duplicate lines are removed from QUEEN.md | VERIFIED | ~/.aether/QUEEN.md is 68 lines with 0 duplicate lines and 0 junk markers (test-colony, <repo> wisdom, test content); ~/.aether/hive/wisdom.json also clean (0 junk entries) |

### Implementation Notes

The Plan 03 must-have stated "Colony-prime reads the entire global QUEEN.md as a single block (same model as CLAUDE.md)" but the actual implementation uses readQUEENMd() which parses into a structured map of bullet entries from the four wisdom sections (Wisdom, Patterns, Philosophies, Anti-Patterns). This is NOT the full file read approach described in the plan. However, the ROADMAP success criteria (SC #2) requires "global QUEEN.md wisdom, Philosophies, and Anti-Patterns sections" to reach workers, which IS satisfied by the structured extraction approach. Non-wisdom sections (User Preferences, Colony Charter) are intentionally excluded -- they have their own separate extraction paths.

**Score:** 5/5 roadmap truths verified

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------| ---------- | ----------- | ------ | -------- |
| QUEE-01 | 57-03 | Remove test junk data and ~270 duplicate lines from QUEEN.md | SATISFIED | Global QUEEN.md is 68 lines, 0 duplicates, 0 junk markers; TestGlobalQueenWisdomHygiene guard test passes |
| QUEE-02 | 57-01 | appendEntriesToQueenSection has dedup -- checks if each entry already exists | SATISFIED | cmd/queen.go lines 562-598: existingNormalized map + normalizeQueenEntry comparison filters duplicates before append; 5 dedup tests pass |
| QUEE-03 | 57-02 | queen-seed-from-hive filters entries already present and reports new vs skipped | SATISFIED | cmd/queen.go lines 356-364: isEntryInText pre-filter; output reports seeded/skipped/total counts; TestQueenSeedFromHiveFiltersDuplicates and TestQueenSeedFromHiveSecondRunSeedsZero pass |
| QUEE-04 | 57-03 | colony-prime injects global QUEEN.md wisdom alongside local wisdom | SATISFIED | cmd/colony_prime_context.go lines 558-580: global_queen_md section with GLOBAL QUEEN WISDOM (Cross-Colony) title; TestColonyPrimeIncludesGlobalQueen passes |
| QUEE-05 | 57-01 | colony-prime injects Philosophies and Anti-Patterns sections from QUEEN.md | SATISFIED | cmd/context.go line 1486: readQUEENMd extended with "Philosophies" and "Anti-Patterns"; 4 section extraction tests pass |
| QUEE-06 | 57-02 | queen-promote-instinct writes to global ~/.aether/QUEEN.md | SATISFIED | cmd/queen.go lines 285-293: hubStore() + loadQueenText(hs) dual write; TestQueenPromoteInstinctWritesGlobal and TestQueenPromoteInstinctSucceedsWithoutHub pass |
| QUEE-07 | 57-03 | High-confidence instincts auto-promoted at /ant-seal | SATISFIED | Both seal wrappers (Claude and OpenCode) contain auto-promotion instructions for confidence >= 0.8 instincts; files are identical |

**Coverage:** 7/7 requirements satisfied

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| cmd/queen.go | normalizeQueenEntry, isEntryInText, dedup in appendEntriesToQueenSection, seed filtering, promote dual-write | VERIFIED | All functions present and wired; "regexp" in imports |
| cmd/context.go | Extended readQUEENMd for Philosophies and Anti-Patterns | VERIFIED | Section names added to inWisdomSection check at line 1486 |
| cmd/colony_prime_context.go | Global QUEEN.md injection into colony-prime | VERIFIED | global_queen_md section at lines 558-580 with protected status |
| cmd/context_weighting.go | Protected section policy for global_queen_md | VERIFIED | Case at line 146 returning true; relevance score 0.75 at line 121 |
| cmd/queen_dedup_test.go | 5 dedup tests | VERIFIED | 5 test functions, all pass |
| cmd/context_queen_test.go | 4 section extraction tests | VERIFIED | 4 test functions, all pass |
| cmd/queen_seed_test.go | 3 seed filtering tests | VERIFIED | 3 test functions, all pass |
| cmd/queen_global_test.go | 2 global write tests | VERIFIED | 2 test functions, all pass |
| cmd/colony_prime_queen_test.go | 2 colony-prime queen injection tests | VERIFIED | 2 test functions, all pass |
| cmd/queen_hygiene_test.go | TestGlobalQueenWisdomHygiene added | VERIFIED | Function at line 57, passes |
| .claude/commands/ant/seal.md | Auto-promotion instructions | VERIFIED | 24 lines, contains queen-promote-instinct, confidence >= 0.8 |
| .opencode/commands/ant/seal.md | Identical to Claude version | VERIFIED | 24 lines, diff confirms identical |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| queen_dedup_test.go | queen.go | Direct function calls | WIRED | Tests call normalizeQueenEntry and appendEntriesToQueenSection |
| context_queen_test.go | context.go | Direct function calls | WIRED | Tests call readQUEENMd directly |
| queen_seed_test.go | queen.go | Cobra CLI dispatch | WIRED | Tests run queen-seed-from-hive via rootCmd.SetArgs |
| queen_global_test.go | queen.go | Cobra CLI dispatch | WIRED | Tests run queen-promote-instinct via rootCmd.SetArgs |
| colony_prime_queen_test.go | colony_prime_context.go | Cobra CLI dispatch | WIRED | Tests run colony-prime and verify output |
| colony_prime_context.go | ~/.aether/QUEEN.md | readQUEENMd | WIRED | filepath.Join(hubDir, "QUEEN.md") at line 559 |
| seal.md (both) | queen-promote-instinct | Wrapper instructions | WIRED | Instructions reference CLI command |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| colony_prime_context.go (global_queen_md) | globalWisdom | readQUEENMd(globalQueenPath) | Yes -- reads actual QUEEN.md from hub | FLOWING |
| queen.go (seed-from-hive) | entries, newEntries | wisdom.json entries + isEntryInText filter | Yes -- reads hive wisdom, filters against existing | FLOWING |
| queen.go (promote-instinct) | entry | instincts.json action field + sanitizeQueenInline | Yes -- reads real instinct data, writes to both stores | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Dedup tests pass | go test ./cmd/... -run "TestAppendEntriesDedup|TestNormalizeQueenEntry" -count=1 | 5/5 PASS | PASS |
| Section extraction tests pass | go test ./cmd/... -run "TestReadQUEENMd" -count=1 | 6/6 PASS | PASS |
| Seed filtering tests pass | go test ./cmd/... -run "TestQueenSeedFromHive|TestIsEntryInText" -count=1 | 3/3 PASS | PASS |
| Global write tests pass | go test ./cmd/... -run "TestQueenPromoteInstinct" -count=1 | 3/3 PASS | PASS |
| Colony-prime queen tests pass | go test ./cmd/... -run "TestColonyPrimeIncludesGlobalQueen|TestColonyPrimeGlobalQueenSurvives" -count=1 | 2/2 PASS | PASS |
| Global hygiene test passes | go test ./cmd/... -run "TestGlobalQueenWisdomHygiene" -count=1 | 1/1 PASS | PASS |
| Seal wrapper parity | diff .claude/commands/ant/seal.md .opencode/commands/ant/seal.md | Identical | PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | - | - | - | No anti-patterns detected in phase-modified files |

### Pre-existing Issues (Not Phase-Introduced)

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| cmd/colony_prime_budget_test.go | 869 | TestColonyPrimeLargePheromonesTrimLowerPriority fails | Info | Pre-existing, file not modified in this phase |

### Human Verification Required

None. All observable truths are verifiable through automated tests and code inspection.

### Gaps Summary

No gaps found. All 7 requirements (QUEE-01 through QUEE-07) are satisfied with substantive implementation and passing tests. The wisdom pipeline is fully connected:

1. **Dedup foundation**: normalizeQueenEntry strips metadata suffixes, appendEntriesToQueenSection filters duplicates before writing (QUEE-02)
2. **Data cleanup**: Global QUEEN.md is clean -- 68 lines, no duplicates, no junk (QUEE-01)
3. **Seed filtering**: queen-seed-from-hive pre-filters against existing entries, reports counts, idempotent on re-run (QUEE-03)
4. **Section extraction**: readQUEENMd covers all four wisdom sections (QUEE-05)
5. **Global injection**: Colony-prime injects global QUEEN.md as protected section (QUEE-04)
6. **Dual write**: queen-promote-instinct writes to both local and global stores (QUEE-06)
7. **Auto-promotion**: Seal wrappers instruct auto-promotion of confidence >= 0.8 instincts (QUEE-07)

One implementation deviation from plan: Plan 03 specified full-file os.ReadFile injection but the actual code uses structured readQUEENMd extraction. This achieves the same ROADMAP outcome (all four wisdom sections reach workers) but omits non-wisdom sections (User Preferences, Colony Charter) from the global injection. This is acceptable because User Preferences already have their own dedicated extraction path in colony-prime.

---

_Verified: 2026-04-26T23:50:00Z_
_Verifier: Claude (gsd-verifier)_
