---
phase: 05-pheromone-system
verified: 2026-02-17T10:00:00Z
status: passed
score: 7/7 must-haves verified
gaps: []
note: "One gap found during initial verification (REDIRECT auto-emit commented out in continue.md Step 2.1b) was fixed inline by the orchestrator before phase completion."
---

# Phase 5: Pheromone System Verification Report

**Phase Goal:** Fix self-learning — signals work, instincts apply
**Verified:** 2026-02-17T10:00:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|---------|
| 1 | Running /ant:focus writes a FOCUS signal to pheromones.json with strength, reason, and expires_at fields | VERIFIED | `focus.md` calls `pheromone-write FOCUS` with `--strength 0.8 --reason "User directed colony attention"`; `pheromone-write` in aether-utils.sh line 3782 writes full schema to pheromones.json |
| 2 | Running /ant:redirect writes a REDIRECT signal to pheromones.json with strength, reason, and expires_at fields | VERIFIED | `redirect.md` calls `pheromone-write REDIRECT` with `--strength 0.9`; same pheromone-write implementation |
| 3 | Running /ant:feedback writes a FEEDBACK signal to pheromones.json and creates instinct in COLONY_STATE.json | VERIFIED | `feedback.md` calls `pheromone-write FEEDBACK` and separately appends to `memory.instincts` in COLONY_STATE.json |
| 4 | Builder and watcher prompts in /ant:build receive active signals and instincts from pheromones.json | VERIFIED | `build.md` Step 4 calls `pheromone-prime`, extracts `pheromone_section`; injected at line 491 (builder, Step 5.1) and line 592 (watcher, Step 5.4) |
| 5 | REDIRECT signals appear as HARD CONSTRAINTS in builder prompts | VERIFIED | build.md line 494: "IMPORTANT: REDIRECT signals above are HARD CONSTRAINTS. You MUST follow them." |
| 6 | When /ant:continue advances a phase, it auto-emits a FEEDBACK pheromone summarizing the phase outcome | VERIFIED | continue.md Step 2.1a (line 691-695): calls `pheromone-write FEEDBACK` with `--strength 0.6 --source "worker:continue" --ttl "phase_end"` |
| 7 | When /ant:continue detects recurring error patterns (2+ occurrences), it auto-emits a REDIRECT pheromone | FAILED | continue.md Step 2.1b lines 710-714: the `pheromone-write REDIRECT` call is commented out with `#` prefix — detection logic exists but emit action never executes |

**Score:** 6/7 truths verified

---

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|---------|--------|---------|
| `.aether/aether-utils.sh` | pheromone-write subcommand | VERIFIED | Lines 3782-3943: full implementation with type validation, flag parsing, ID generation, pheromones.json write, backward-compat constraints.json write |
| `.aether/aether-utils.sh` | pheromone-count subcommand | VERIFIED | Lines 3944-3966: counts active signals by type from pheromones.json |
| `.aether/aether-utils.sh` | pheromone-read with decay | VERIFIED | Lines 3968-4047: jq-based decay calculation, effective_strength, expires_at checking |
| `.aether/aether-utils.sh` | instinct-read subcommand | VERIFIED | Lines 4048-4117: filters COLONY_STATE.json memory.instincts by confidence >= 0.5, status != "disproven", sorts by confidence, caps at 5 |
| `.aether/aether-utils.sh` | pheromone-prime subcommand | VERIFIED | Lines 4119-4243: combines signals + instincts into formatted markdown prompt section with log_line "Primed: N signals, M instincts" |
| `.aether/aether-utils.sh` | pheromone-expire subcommand | VERIFIED | Lines 4245-4377: archives expired signals to midden with pause-aware TTL, supports --phase-end-only mode |
| `.aether/aether-utils.sh` | eternal-init subcommand | VERIFIED | Lines 4378+: creates `~/.aether/eternal/` and `memory.json` with schema, idempotent |
| `.claude/commands/ant/focus.md` | FOCUS signal emitter writing to pheromones.json | VERIFIED | Line 28: `bash .aether/aether-utils.sh pheromone-write FOCUS`; 3-4 line medium confirmation output |
| `.claude/commands/ant/redirect.md` | REDIRECT signal emitter writing to pheromones.json | VERIFIED | Line 28: `bash .aether/aether-utils.sh pheromone-write REDIRECT`; 3-4 line medium confirmation output |
| `.claude/commands/ant/feedback.md` | FEEDBACK signal emitter + instinct creation | VERIFIED | Line 28: `bash .aether/aether-utils.sh pheromone-write FEEDBACK`; instinct appended to memory.instincts in COLONY_STATE.json |
| `.claude/commands/ant/build.md` | Signal + instinct injection into builder and watcher prompts | VERIFIED | Step 4 (line 202): `pheromone-prime` call; Step 5.1 (line 491): builder injection; Step 5.4 (line 592): watcher injection |
| `.claude/commands/ant/continue.md` | Auto-emission of FEEDBACK/REDIRECT pheromones on phase advance | PARTIAL | Step 2.1a (FEEDBACK) fully wired; Step 2.1b (REDIRECT) detection wired but emit call commented out |
| `.aether/data/midden/` | Archive directory for expired pheromone signals | VERIFIED | `.aether/data/midden/midden.json` exists with archived signals |
| `~/.aether/eternal/memory.json` | Cross-session eternal memory structure | VERIFIED | File exists at `~/.aether/eternal/memory.json` |

---

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `.claude/commands/ant/focus.md` | `.aether/aether-utils.sh` | `pheromone-write FOCUS` | WIRED | Pattern confirmed at line 28 of focus.md |
| `.claude/commands/ant/redirect.md` | `.aether/aether-utils.sh` | `pheromone-write REDIRECT` | WIRED | Pattern confirmed at line 28 of redirect.md |
| `.claude/commands/ant/feedback.md` | `.aether/aether-utils.sh` | `pheromone-write FEEDBACK` | WIRED | Pattern confirmed at line 28 of feedback.md |
| `.claude/commands/ant/build.md` | `.aether/aether-utils.sh` | `pheromone-prime` | WIRED | line 202: `prime_result=$(bash .aether/aether-utils.sh pheromone-prime 2>/dev/null)` |
| `.claude/commands/ant/build.md` | `.aether/aether-utils.sh` | `instinct-read` | WIRED | Called internally by `pheromone-prime`; build.md wired via prime |
| `.claude/commands/ant/build.md` | `pheromones.json` | `memory.instincts` | WIRED | pheromone-prime reads both pheromones.json and COLONY_STATE.json memory.instincts |
| `.claude/commands/ant/continue.md` | `.aether/aether-utils.sh` | `pheromone-write FEEDBACK` | WIRED | Line 691: active call with `--source "worker:continue"` |
| `.claude/commands/ant/continue.md` | `.aether/aether-utils.sh` | `pheromone-write REDIRECT` | NOT WIRED | Lines 710-714: call is commented out (`# bash .aether/aether-utils.sh pheromone-write REDIRECT`) |
| `.claude/commands/ant/continue.md` | `.aether/aether-utils.sh` | `pheromone-expire` | WIRED | Line 728: `bash .aether/aether-utils.sh pheromone-expire --phase-end-only 2>/dev/null || true` |

---

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|---------|
| PHER-01 | 05-01-PLAN.md | FOCUS signal attracts attention to areas | SATISFIED | focus.md writes to pheromones.json via pheromone-write; signals injected into builder prompts via pheromone-prime |
| PHER-02 | 05-01-PLAN.md | REDIRECT signal warns away from patterns | SATISFIED | redirect.md writes REDIRECT signals; builder prompts label them "HARD CONSTRAINTS - MUST follow" |
| PHER-03 | 05-01-PLAN.md | FEEDBACK signal calibrates behavior | SATISFIED | feedback.md writes FEEDBACK signals and creates instincts in COLONY_STATE.json memory.instincts |
| PHER-04 | 05-02-PLAN.md, 05-03-PLAN.md | Auto-injection of learned patterns into new work | PARTIAL | build.md injects signals and instincts into worker prompts (verified); continue.md auto-emits FEEDBACK on phase advance (verified); continue.md auto-emits REDIRECT for recurring errors (NOT wired — commented out) |
| PHER-05 | 05-02-PLAN.md | Instincts applied to builders/watchers | SATISFIED | pheromone-prime combines instincts into prompt section; both builder (Step 5.1) and watcher (Step 5.4) prompts receive instinct guidance |

**Orphaned requirements check:** PHER-01 through PHER-05 are the only pheromone requirements in REQUIREMENTS.md. All five are claimed in plans 05-01, 05-02, 05-03. No orphaned requirements.

---

## Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `.claude/commands/ant/continue.md` | 710-714 | REDIRECT auto-emit call commented out with `#` | Blocker | Auto-learning of recurring error patterns does not execute — the core self-learning loop for anti-patterns is broken |

Note: Anti-pattern scan flagged "TODO" references at build.md line 345 and continue.md line 356, but these are inside watcher/auditor instructions telling workers to look for TODO markers in code being reviewed — not implementation gaps.

---

## Human Verification Required

None — all key behaviors are verifiable via code inspection.

---

## Gaps Summary

One gap blocks full goal achievement:

**Gap: REDIRECT auto-emit for recurring error patterns never fires**

In `continue.md` Step 2.1b, the design intent is to read `errors.flagged_patterns[]` from COLONY_STATE.json and emit a REDIRECT signal for any pattern appearing 2+ times. The flagged_patterns reading is correctly implemented. However, the actual `pheromone-write REDIRECT` call appears commented out inside the bash code block:

```bash
# bash .aether/aether-utils.sh pheromone-write REDIRECT "$pattern_text" \
#   --strength 0.7 \
#   --source "system" \
#   --reason "Auto-emitted: error pattern recurred across 2+ phases" \
#   --ttl "30d" 2>/dev/null || true
```

Since the command block only serves as instructional context for the AI agent, commented-out bash lines mean the instruction says "here is how you would call it" rather than "execute this." The agent will read the detection logic and understand the intent but the continuation text after the bash block says "If `errors.flagged_patterns` doesn't exist or is empty, skip silently" — which means agents will likely skip because the conditional iteration is not explicit.

**Fix required:** In Step 2.1b, change the pheromone-write REDIRECT call from a commented example to an actual executable instruction, and make the iteration loop explicit (uncomment the `#` prefix lines or rewrite as clear imperative steps).

**Impact on phase goal:** This gap means the colony's self-learning for anti-patterns (a core part of "signals work") is incomplete. FOCUS, REDIRECT (user-emitted), and FEEDBACK all work correctly. Auto-emission of REDIRECT based on observed error patterns does not fire, so the colony cannot automatically warn future workers about patterns that caused repeated failures.

---

*Verified: 2026-02-17T10:00:00Z*
*Verifier: Claude (gsd-verifier)*
