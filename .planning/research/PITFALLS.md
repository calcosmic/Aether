# Pitfalls Research

**Domain:** Production hardening of multi-agent AI orchestration system (Aether v2 -> v2.1)
**Researched:** 2026-03-23
**Confidence:** HIGH (grounded in 82%-confidence Oracle audit with 55 findings, 17 real midden failures, 572+ test suite, and codebase static analysis)

---

## Critical Pitfalls

### Pitfall 1: The Refactoring Death Spiral -- Fixing Errors Introduces More Errors

**What goes wrong:**
The Oracle found the bug-fix ratio rising from 33.8% to 45.8%, meaning nearly half of all recent changes are fixes for bugs introduced by other changes. When you start a production hardening effort on a codebase in this state, the most natural instinct -- "fix all the error suppression" -- accelerates the spiral. Each error handling fix touches a function used by 10+ callers. Each caller has its own error expectations. Changing the error behavior of a shared function breaks callers that depended on the old (broken) behavior.

This is the single most dangerous pitfall for Aether's hardening milestone. The 338 error-swallowing patterns in aether-utils.sh exist because callers were written to expect silence. Removing the silence without updating every caller path creates cascading failures across the build/continue lifecycle.

**Why it happens:**
The three-layer error suppression (callers suppress memory-capture, memory-capture suppresses sub-steps, sub-steps suppress internals) is load-bearing. The `2>/dev/null || true` pattern on all memory-capture call sites in build-wave.md, build-verify.md, build-complete.md, and continue-advance.md exists because the build lifecycle must not abort when a learning observation fails. If you remove the suppression without adding proper error handling at each layer, a single corrupted learning-observations.json kills all 5 downstream memory steps AND crashes the build.

**How to avoid:**
1. Triage before fixing. The Oracle's Recommendation 3 is correct: categorize the 338 instances into (a) correct suppression (optional/fallback paths -- keep), (b) lazy suppression (hiding real errors -- fix), (c) dangerous suppression (data-writing operations -- critical fix). Start with category (c) only.
2. Never remove `|| true` without replacing it with explicit error handling. The pattern is: `result=$(command 2>/dev/null) || { log_warning "command failed"; handle_fallback; }` -- not just deleting `|| true`.
3. Add a regression test for each suppression point before changing it. If the test does not exist, write the test first, verify it passes with the current behavior, then change the behavior.
4. Track the fix ratio as a metric. If it rises above 50% during hardening, stop and stabilize before continuing.

**Warning signs:**
- Tests that passed before the hardening phase start failing in unrelated areas
- The midden starts accumulating "unbound variable crash" entries (3 already exist from previous work)
- Build or continue commands abort mid-execution with error messages that were previously silent
- The fix-commit ratio in git log exceeds 1:1 (more fix commits than feature commits)

**Phase to address:**
Error handling triage phase -- must come before any modularization work. Triage is read-only analysis; the fixes come after.

**Oracle evidence:** Synthesis Q3 finding 1 (338 error-swallowing patterns), Q3 finding 8 (three-layer error silence), Q5 Recommendation 3 (triage approach), Midden entries (3 unbound variable crashes), Sage analytics (33.8% -> 45.8% rising fix ratio)

---

### Pitfall 2: Monolith Extraction Breaks the Case-Statement Dispatch Contract

**What goes wrong:**
aether-utils.sh uses a single case statement to dispatch all 178 subcommands. When you extract subcommands into separate files (the obvious modularization move), you break the dispatch contract. Functions that were defined in the same file scope can no longer reference each other. Helper functions called by the extracted subcommand become undefined. The conditional module sourcing at lines 26-34 (`[[ -f ]] && source`) means modules can silently fail to load, and extracted code inherits this fragility.

The Oracle confirmed there are no eval/dynamic dispatch patterns -- it is pure case-statement routing. This means extraction is technically safe, but the 76 dead subcommands (43% of total) create a trap: you cannot tell which helper functions are used only by dead code until you trace every call graph.

**Why it happens:**
Large bash scripts accumulate shared state through file-scoped variables and functions. An 11,272-line script with 178 subcommands inevitably has functions that are "shared" only because they happen to be in the same file. When you extract subcommand A to a separate file, you discover it depends on helper function X, which also serves subcommand B. You now must decide: duplicate X, extract X to a shared library, or leave A in the monolith.

The 9 already-extracted utility modules (.aether/utils/*.sh) demonstrate this was done before, but the extraction was partial. hive.sh is 24,358 bytes; skills.sh is 20,152 bytes. These are mini-monoliths themselves.

**How to avoid:**
1. Before extracting anything, run the dead code removal phase. Remove the 76 confirmed-dead subcommands first. This reduces the file by 15-20% and eliminates false dependency signals. The Oracle confirmed these have no indirect invocation mechanism (case-statement dispatch, not eval).
2. Extract in dependency order: leaf functions first (no internal dependencies), then mid-level functions, then orchestrators. Never extract an orchestrator (like memory-capture or colony-prime) before its dependencies.
3. For each extraction, create a "contract test" that calls the extracted function with the same arguments the monolith used and verifies identical output. The existing 572+ tests provide coverage, but they test through the case-statement entry point. You need tests that verify the extracted module independently.
4. Preserve `set -euo pipefail` in every extracted module. The ERR trap behavior changes when crossing `source` boundaries -- a trap set in the parent is not inherited by sourced scripts in all bash versions. Each extracted module needs its own error configuration header.

**Warning signs:**
- "Command not found" errors from subcommands that worked before extraction
- Tests passing when run individually but failing when run as a suite (indicates hidden dependency on shared state from a sourced module)
- The conditional source pattern (`[[ -f ]] && source`) silently skipping a module that was moved during extraction

**Phase to address:**
Dead code removal must come first. Modularization/extraction must come second. They cannot be combined.

**Oracle evidence:** Synthesis Q1 finding 16 (178 subcommands, 76 dead, case-statement dispatch), Q2 finding 2 (silent degradation from conditional sourcing), Q1 finding 14 (CLI/bash separation already clean)

---

### Pitfall 3: Documentation Correction Creates a False Sense of Accuracy

**What goes wrong:**
The Oracle identified 6 instances where documentation says things that are not true (CLAUDE.md claims rolling summary is "highest priority -- never trimmed first" but code trims it first; "125 subcommands" is actually 178; "security gate" oversells the 6-pattern check-antipattern; etc.). The natural response is to update documentation to match reality. But if documentation is corrected while the underlying code is still being refactored, the documentation drifts again within weeks.

Worse: documentation corrections that happen in a separate phase from the code changes create a window where documentation is "accurate" but the code is about to change. Anyone reading the docs during that window gets a false understanding.

**Why it happens:**
Documentation and code evolve at different cadences. A doc fix is a 5-minute PR. The corresponding code fix might take a week. If you batch doc fixes into a "documentation accuracy phase" and code fixes into a separate "reliability phase," the two diverge immediately. This is exactly the pattern that created the 6 documentation inaccuracies in the first place -- the code was changed (v2.0 features added), but docs were not updated atomically.

**How to avoid:**
1. Never fix documentation in isolation. Every documentation change must be paired with either (a) the code change it describes, or (b) a test that verifies the documentation claim. If neither exists, add a `[KNOWN INACCURACY]` annotation rather than silently fixing the doc.
2. Create "doc-truth tests" for critical claims. Example: test that asserts the trim order in colony-prime matches the order documented in CLAUDE.md. This makes documentation drift detectable by CI.
3. Correct the 6 known inaccuracies atomically with the code phase that changes the related behavior. If the rolling-summary trim order is going to change during error handling improvements, fix the documentation at the same time.
4. Audit documentation claims only once, at the end of hardening, not at the beginning. Early documentation work is wasted effort if the underlying code is still changing.

**Warning signs:**
- Documentation PRs with no corresponding test or code changes
- CLAUDE.md edit timestamp is weeks newer than the code it describes
- Users (or the LLM agents themselves) behaving as though documentation claims are true when they are not -- e.g., expecting rolling summary to be preserved under budget pressure because CLAUDE.md says so

**Phase to address:**
Documentation accuracy should be the final phase, after all code changes are complete and tested. Not the first phase.

**Oracle evidence:** Synthesis Pattern 1 (6 documentation accuracy instances), Q4 finding 3 (CLAUDE.md trim order inversion), Q1 finding 16 (subcommand count discrepancy), Q3 finding 5 (security gate label oversells)

---

### Pitfall 4: State File Protection Changes Break the Build/Continue Lifecycle

**What goes wrong:**
COLONY_STATE.json has 219 direct references across 38 of 43 slash commands and 153 references inside aether-utils.sh. It is accessed through two parallel paths: direct jq calls in slash commands AND via aether-utils.sh subcommands. The Oracle recommends per-phase checkpointing (Rec 1) and closing the continue-advance lock gap (Rec 9). Both are correct and necessary. But implementing them incorrectly breaks the build/continue lifecycle.

Specifically: the continue-advance step writes COLONY_STATE.json via the LLM Write tool (not a bash command), bypassing all bash-level locking. If you add a lock requirement to state writes, you must also change how the LLM Write tool interaction works in continue-advance -- or the entire continue flow deadlocks.

**Why it happens:**
The dual-access pattern (jq + subcommands) exists because slash commands are markdown interpreted by Claude Code, not bash scripts. They can call bash subcommands but also use Claude Code's native Write tool. The Write tool does not know about bash file locks. Any hardening that assumes "all state writes go through bash" is wrong.

**How to avoid:**
1. Map all state mutation paths before adding any locking. The Oracle already traced the full build/continue flow in Synthesis Q2 findings 5-8. Use that trace as the authoritative reference.
2. Implement checkpointing (backup before write) separately from locking improvements. Checkpointing is purely additive (copy file before mutation) and cannot break existing flows. Locking changes modify control flow and can deadlock.
3. For the continue-advance LLM Write gap: do not try to add bash locking to the LLM Write tool call. Instead, add a post-write validation step: after the LLM writes COLONY_STATE.json, immediately read it back and validate against schema. This closes the corruption window without changing the write mechanism.
4. Test the full build-then-continue cycle end-to-end after any state protection change. Unit tests on individual subcommands will not catch lifecycle-level deadlocks.

**Warning signs:**
- Continue command hangs indefinitely (deadlock from competing lock acquisitions)
- COLONY_STATE.json contains data from a different phase than expected (race condition between LLM Write and spawn-complete)
- Backup files (.phase-N.bak) are created but never cleaned up, accumulating disk usage
- Tests pass but the autopilot loop hangs at the continue step

**Phase to address:**
State protection phase -- must come after error handling triage (Pitfall 1) but before modularization (Pitfall 2), because state protection changes affect the contract that extracted modules must honor.

**Oracle evidence:** Synthesis Q2 finding 1 (219 references, dual-access), Q2 finding 8 (continue-advance lock gap), Q5 Rec 1 (checkpointing), Q5 Rec 9 (reconciliation)

---

### Pitfall 5: Dead Code Removal Accidentally Kills "Hidden Live" Functions

**What goes wrong:**
The Oracle identified 76 subcommands (43%) as never invoked by any command or playbook. Removing them seems safe -- the Oracle confirmed case-statement dispatch with no eval/dynamic patterns. But some of these "dead" functions are used interactively by operators (via direct `bash .aether/aether-utils.sh <subcommand>` calls in the terminal), used by custom skills, or used by the OpenCode mirror commands that were not included in the Oracle's static analysis scope.

The Oracle analyzed `.claude/commands/ant/` and `.aether/docs/command-playbooks/` for caller references. It did NOT analyze `.opencode/commands/ant/` (40 additional commands), user-created skills at `~/.aether/skills/domain/`, or direct terminal usage patterns.

**Why it happens:**
Static analysis of bash callers can only trace what is written in files within the analyzed scope. Aether has three command surfaces (Claude Code slash commands, OpenCode commands, and direct terminal bash invocation). The Oracle's 85% trust ratio means 15% of findings rely on single sources -- and the dead code finding is based on grep analysis of one command surface.

**How to avoid:**
1. Before removing any "dead" subcommand, check all three surfaces: `.claude/commands/ant/`, `.opencode/commands/ant/`, and grep the entire repo including docs and test files.
2. For subcommands in the "swarm display," "learning display," and "spawning diagnostics" categories identified as dead -- verify these are not used by `/ant:watch`, `/ant:status`, or other display-oriented commands that may call them indirectly through the swarm-display.sh utility.
3. Do not delete dead code immediately. First, add a deprecation warning: `echo "WARNING: subcommand '$1' is deprecated and will be removed" >&2`. Run for one full development cycle. If no one complains, then remove.
4. The 76 dead subcommands include "semantic search engine (6)" and "suggest advanced (4)." These were likely built for future features. Check the roadmap for planned features that might need them before removing.

**Warning signs:**
- OpenCode users report "subcommand not found" errors after a release
- The swarm-display or spawn-tree visualization commands stop working
- A user-created skill references a subcommand that was removed
- Tests in tests/bash/ fail because they test removed subcommands (some bash tests may exercise "dead" code)

**Phase to address:**
Dead code deprecation must come before dead code removal. Deprecation is the first sub-step of modularization. Removal happens one cycle later.

**Oracle evidence:** Synthesis Q1 finding 16 (76 dead subcommands, 43% of total), Q1 finding 18 (no eval/dynamic dispatch confirmed), gaps.md (static analysis ceiling acknowledged)

---

### Pitfall 6: Memory Pipeline Circuit Breaker Creates New Silent Failures

**What goes wrong:**
The Oracle's Recommendation 8 (memory pipeline circuit breaker with file recovery) addresses a real problem: corrupted learning-observations.json kills all 5 downstream memory steps silently. The fix is to reset the corrupted file to its template and retry. But if the circuit breaker itself is implemented with the same error-suppression patterns that caused the original problem, you create a new layer of silent failure.

Specifically: if the template file is missing or corrupted, the "recovery" path fails silently. If the midden-write call (to log the corruption event) fails, the audit trail is lost. If the retry succeeds but the original corruption was caused by a concurrent write, the retry will also be corrupted.

**Why it happens:**
Circuit breaker implementations in bash are inherently fragile because bash has no try/catch mechanism. The `|| { recovery_code; }` pattern looks clean but cannot handle failures within the recovery block without nested error handling. The three-layer suppression problem (Pitfall 1) means adding a circuit breaker adds a fourth layer.

**How to avoid:**
1. Implement the circuit breaker as a separate, independently-tested function -- not inline in memory-capture. Name it `learning_observations_recover` with its own test suite.
2. The recovery path must NOT use `|| true`. If recovery fails, it should return a non-zero exit code that the caller can handle. The caller (memory-capture) can then decide to skip learning capture for this cycle and log the failure.
3. Validate the template file at startup (during module sourcing), not at recovery time. If the template is missing at startup, fail loudly before any colony operations begin.
4. The midden-write in the recovery path should use a separate, guaranteed-write mechanism (direct append to a recovery log file) rather than the midden system itself, which has its own locking issues (Synthesis Q2 finding 7).

**Warning signs:**
- The circuit breaker fires repeatedly (indicates the root cause of corruption was not addressed)
- Learning observations are empty or contain only template data after a build (indicates the breaker fired and reset but new observations were not captured)
- The midden does not contain corruption entries even though the circuit breaker is logging them (indicates the midden write in the recovery path is also failing)

**Phase to address:**
Memory pipeline hardening -- must come after error handling triage (Pitfall 1) because the circuit breaker's error handling approach depends on the triage decisions.

**Oracle evidence:** Synthesis Q4 finding 12 (sequential kill-switch), Q3 finding 8 (three-layer error silence), Q5 Rec 8 (circuit breaker design), Q2 finding 7 (midden graceful degradation data loss race)

---

## Moderate Pitfalls

### Pitfall 7: Hive Brain Type Coercion Fix Breaks Existing Stored Data

**What goes wrong:**
The string-typed confidence bug (confirmed by both REDIRECT signal and midden entry) causes silent exclusion of valid wisdom from hive-read. The fix is trivial: add `(tonumber? // 0)` to the jq filter. But existing hive entries at `~/.aether/hive/wisdom.json` across all user machines have mixed types. The fix makes previously-excluded entries suddenly visible, potentially flooding worker context with stale or low-quality wisdom that was functionally filtered out.

**How to avoid:**
1. Fix the type coercion in hive-read (the read path), and simultaneously add type normalization to hive-store (the write path) so new entries are always numeric.
2. Add a one-time migration: `hive-migrate` subcommand that reads wisdom.json, coerces all confidence values to numbers, and writes back. Run this during `aether update`.
3. Do NOT rely on the fix alone. Some entries may have been stored with artificially high confidence as strings. Review the confidence distribution after migration.

**Warning signs:**
- Workers suddenly reference wisdom that seems irrelevant or outdated (previously-excluded entries now visible)
- Colony-prime budget trimming triggers more frequently (more wisdom entries = more content = hits budget cap sooner)

**Phase to address:** Quick wins / bug fix phase (independent, can be done anytime)

**Oracle evidence:** Synthesis Q3 finding 7, Q4 finding 7, Q5 Rec 4, Midden entry (Chaos finding: String-typed confidence)

---

### Pitfall 8: Test Suite Becomes a Change Blocker Instead of a Safety Net

**What goes wrong:**
Aether has 572+ tests, with 1 currently failing (context-continuity compact signals test). During production hardening, test count typically grows. But if tests are written against current (broken) behavior rather than desired behavior, they cement the bugs they should be catching. Then when you fix the underlying code, dozens of tests fail because they expected the broken behavior.

The existing test failure (`pheromone-prime --compact respects max signal limit` checking for 'COMPACT SIGNALS' string) demonstrates this: the test asserts a specific string in the prompt output, but the code was changed and the string format diverged. This is a test coupled to implementation details, not behavior.

**How to avoid:**
1. Before hardening, audit existing tests for "testing the bug" patterns. Any test that asserts on error-suppression behavior (e.g., "this function returns empty string on error") should be flagged for review.
2. New tests during hardening should assert on desired behavior, not current behavior. Write the test for what the code SHOULD do after the fix, then make it pass.
3. Fix the 1 existing test failure before starting hardening. A test suite that is already red provides no signal -- you cannot distinguish "test broke because of my change" from "test was already broken."
4. Avoid test explosion. The 572+ tests run in 2+ minutes. If hardening doubles the test count without improving signal-to-noise, CI becomes a bottleneck. Focus on high-value integration tests over unit tests for error handling paths.

**Warning signs:**
- Test count grows but the same bugs keep recurring (tests are too narrow)
- Tests that pass with `|| true` in the code under test fail when `|| true` is removed (tests were testing the suppressed path)
- CI takes longer than 5 minutes (developer velocity drops, people skip tests)

**Phase to address:** Test audit should be a sub-task of the error handling triage phase

**Oracle evidence:** Sage analytics (rising fix ratio), 1 currently failing test (context-continuity), 572+ tests as existing safety net

---

### Pitfall 9: Parallel Builder Race Conditions Are Made Worse by Lock Improvements

**What goes wrong:**
The Oracle found midden.json has a specific data loss race (shared temp file path for lockless writes), activity.log and spawn-tree.txt are unprotected (relying on POSIX append atomicity), and COLONY_STATE.json has a continue-advance lock gap. The instinct is to add locking everywhere. But adding locks to paths that previously did not lock creates new deadlock possibilities and performance bottlenecks.

Specifically: midden-write currently falls through to a lockless write when the lock fails (by design -- midden is non-critical). If you make the lock mandatory, a slow builder holding the midden lock blocks all other builders from logging failures. During a cascade of failures (the exact scenario where midden is most needed), the system deadlocks on midden writes instead of recording them.

**How to avoid:**
1. Fix the midden temp file race by using PID-qualified temp files (`midden.json.tmp.$$` instead of `midden.json.tmp`). This eliminates the race without adding locks.
2. For activity.log and spawn-tree.txt, keep the POSIX append pattern. The Oracle confirmed write sizes are well under PIPE_BUF (4096 bytes). Adding locks here is pure overhead with no benefit.
3. Only add locking where the Oracle identified actual gaps: the continue-advance COLONY_STATE.json write. And even there, prefer validation-after-write over locking (see Pitfall 4).
4. Never add a blocking lock (wait-for-lock) in a parallel builder code path. If you must lock, use a try-lock with graceful fallback, same as the existing midden pattern but with a PID-qualified temp file.

**Warning signs:**
- Build times increase noticeably after lock additions (lock contention)
- Parallel builders complete one-at-a-time instead of in parallel (accidental serialization)
- Deadlock during high-failure scenarios (all builders waiting on midden lock)

**Phase to address:** Concurrency fixes -- should be a targeted sub-phase, not a general "add locks everywhere" effort

**Oracle evidence:** Synthesis Q2 findings 6-8 (concurrent write protection analysis), Q2 finding 7 (midden temp file race)

---

### Pitfall 10: Hardening the Planning Quality Creates Infinite Recursion

**What goes wrong:**
The current planning pipeline (`/ant:plan`) produces shallow phases without per-phase research. The Oracle research was specifically commissioned to improve this. But if you make the planning phase too deep (e.g., requiring Oracle-level research for every phase), the planning phase itself becomes a multi-day effort that produces plans that are stale by the time they are executed.

The planning pipeline currently generates phases by having the LLM analyze the colony state and propose a phase breakdown. Adding research requirements means each phase plan triggers research, which produces findings, which may change the plan, which triggers new research. Without a termination condition, planning becomes the bottleneck.

**How to avoid:**
1. Add per-phase research only for phases flagged as "likely needs deeper research" (the GSD pattern). Not every phase needs research -- simple phases ("update documentation," "add tests for X") do not benefit from research overhead.
2. Set a hard time budget for per-phase research: 1 investigation cycle (not a full Oracle RALF loop). If the question cannot be answered in one cycle, flag it for resolution during execution, not planning.
3. Keep the planning output format unchanged. Do not add new required fields to the phase plan schema -- it would break continue.md and autopilot (run.md) which parse the phase plan structure.
4. Research findings should inform the plan but not block it. "Uncertainty" is a valid phase annotation, not a reason to stop planning.

**Warning signs:**
- Planning takes longer than the execution of a simple phase
- Plans are revised 3+ times before any phase is built
- Research findings contradict each other and planning stalls waiting for resolution
- The autopilot refuses to start because the plan schema changed

**Phase to address:** Planning quality improvements should be incremental, not a big-bang rewrite of the planning pipeline

---

### Pitfall 11: Agent Fallback Degradation Goes Unnoticed During Testing

**What goes wrong:**
When a specialized agent (e.g., aether-chaos.md with 200+ lines of discipline) falls back to a general-purpose agent with a one-sentence role description, the system silently continues at dramatically reduced capability. During development, all 22 agents are available. But in production or in non-standard environments, agent resolution can fail. The Oracle identified this but noted it happens silently.

If hardening adds new agent capabilities or changes agent definitions, the fallback path is never updated (because the fallback is a single generic sentence). This means any improvement to agents has zero effect when fallback triggers.

**How to avoid:**
1. Implement the Oracle's Rec 6: log fallback as a midden entry and include a WARNING in build synthesis.
2. Add an agent resolution test that verifies all 22 agents resolve correctly in the test environment. If any agent falls back during tests, the test should fail.
3. When adding new capabilities to an agent definition, also update the fallback description to include at least the critical behavioral rules (output format, boundary declarations).

**Warning signs:**
- Build quality is inconsistent between environments (agents resolve in one, fall back in another)
- Chaos or Gatekeeper findings are unusually shallow (general-purpose agent lacks the discipline of the specialized definition)

**Phase to address:** Agent reliability -- can be addressed alongside the error handling phase since it involves adding logging/detection

**Oracle evidence:** Synthesis Q3 finding 4 (dramatic fallback degradation), Q5 Rec 6 (fallback logging)

---

## Technical Debt Patterns

Shortcuts that seem reasonable during hardening but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Adding `|| true` to new code to "match existing style" | Consistency with current patterns | Perpetuates the three-layer silence that caused the rising fix ratio | Never -- new code should use explicit error handling |
| Extracting modules without contract tests | Faster modularization | Silent regressions when shared helpers change | Never -- extraction without tests is invisible breakage |
| Fixing documentation before fixing code | Looks like progress, addresses known inaccuracies | Documentation drifts again when code changes | Only for the final hardening phase, after code is stable |
| Removing dead code without deprecation cycle | Immediate file size reduction | Breaks undiscovered callers in OpenCode, user skills, or terminal usage | Only for subcommands with zero grep hits across the ENTIRE repo (including .opencode/) |
| Making midden writes mandatory (always locked) | Eliminates data loss race | Creates deadlock under cascade failures | Never -- use PID-qualified temp files instead |
| Adding per-phase research to all phases | Better planning quality | Planning becomes a bottleneck; simple phases do not need research | Only for phases flagged as "likely needs deeper research" |

## "Looks Done But Isn't" Checklist

Things that appear complete during hardening but are missing critical pieces.

- [ ] **Error handling triage:** Often missing the "correct suppression" category -- 338 instances triaged but all marked as "fix" when many are genuinely correct fallback paths. Verify each category has entries.
- [ ] **Dead code removal:** Often missing OpenCode command surface check -- grep `.claude/commands/ant/` and `.aether/docs/command-playbooks/` but forget `.opencode/commands/ant/` (40 commands).
- [ ] **State checkpointing:** Often missing cleanup -- backups created before each phase but never pruned. Verify disk usage after 10+ phases with backups enabled.
- [ ] **Memory circuit breaker:** Often missing the "what if recovery fails" path -- breaker resets to template but does not handle template-not-found. Verify the template exists at startup.
- [ ] **Documentation corrections:** Often missing "doc-truth tests" -- documentation updated but no test verifies the claim remains true. Verify at least the 6 known inaccuracies have corresponding tests.
- [ ] **Type coercion fix:** Often missing the migration step -- hive-read fixed but existing string-typed entries in `~/.aether/hive/wisdom.json` not migrated. Verify migration runs during `aether update`.
- [ ] **Lock improvements:** Often missing deadlock testing -- locks added but never tested under contention. Verify parallel builders can complete when all contend on the same lock.
- [ ] **Autopilot reconciliation:** Often missing the "operator prompt" UX -- desync detected but the pause message is confusing. Verify the pause message shows both values and suggests a resolution.

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Refactoring death spiral (fix ratio > 50%) | MEDIUM | Revert to last known-good commit. Run full test suite to confirm green. Resume with smaller, more targeted changes. Track fix ratio per-commit, not per-phase. |
| Monolith extraction breaks dispatch | LOW | Revert the extraction. The monolith still works. Re-attempt with contract tests written first. |
| Documentation drift after correction | LOW | Run doc-truth tests. Fix any that fail. If no doc-truth tests exist, re-audit documentation against current code. |
| State protection deadlock | HIGH | Revert lock changes. The existing unprotected path has worked for all of v1 and v2. Add checkpointing (purely additive) without lock changes. Retry lock changes with deadlock detection. |
| Dead code removal breaks OpenCode | MEDIUM | Re-add the removed subcommands. Add deprecation warnings. Wait one cycle. Re-check OpenCode surface before removing. |
| Memory circuit breaker creates new silent failures | MEDIUM | Revert to the original fire-and-forget behavior (it at least captures learnings when not corrupted). Rewrite the breaker as an isolated function with its own test suite. |
| Parallel builder deadlock from lock additions | HIGH | Revert all lock changes. Build times will return to normal immediately. Re-attempt with try-lock-with-fallback pattern, never blocking locks. |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1: Refactoring death spiral | Error handling triage (first phase) | Fix ratio tracked per-commit; stays below 50% |
| 2: Monolith extraction breaks dispatch | Modularization (after triage + dead code) | All 572+ tests pass; no "command not found" errors |
| 3: Documentation drift | Documentation accuracy (final phase) | Doc-truth tests exist for 6+ known inaccuracies |
| 4: State protection breaks lifecycle | State protection (after triage, before modularization) | Full build-then-continue cycle completes without deadlock |
| 5: Dead code removal kills live functions | Dead code deprecation (before modularization) | Deprecation warnings logged; zero complaints after one cycle |
| 6: Circuit breaker creates new failures | Memory pipeline hardening (after triage) | Circuit breaker fires and recovers in test; recovery failure path also tested |
| 7: Hive type coercion breaks stored data | Quick wins (independent) | Migration runs cleanly; confidence distribution is reasonable |
| 8: Tests block instead of protect | Test audit (part of triage phase) | 1 existing failure fixed; no tests assert on suppressed behavior |
| 9: Locks cause deadlock | Concurrency fixes (targeted sub-phase) | Parallel builders complete under contention in test |
| 10: Planning recursion | Planning improvements (incremental) | Planning completes in bounded time; autopilot loop unbroken |
| 11: Agent fallback unnoticed | Agent reliability (alongside triage) | Agent resolution test verifies all 22 agents; fallback logs to midden |

## Phase Ordering Implications

Based on pitfall dependencies, the hardening phases MUST be ordered:

1. **Error handling triage** (read-only analysis, informs all other phases)
2. **Quick wins** (type coercion, existing test fix -- independent, no risk)
3. **State protection** (checkpointing is additive; lock changes need triage results)
4. **Dead code deprecation** (adds warnings, no removal -- must precede modularization)
5. **Memory pipeline hardening** (circuit breaker depends on triage decisions)
6. **Concurrency fixes** (targeted, not "add locks everywhere")
7. **Modularization** (depends on dead code removal, state protection, error handling all being stable)
8. **Agent reliability** (can parallel with any phase after triage)
9. **Planning improvements** (incremental, non-blocking)
10. **Documentation accuracy** (LAST -- only after all code is stable)

**Critical ordering constraint:** Modularization (phase 7) depends on phases 1, 3, 4, and 5 all being complete. Attempting modularization before error handling, state protection, and dead code work is the highest-risk mistake.

## Sources

- Oracle Synthesis (82% confidence, 55 findings across 5 questions) -- `/Users/callumcowie/repos/Aether/.aether/oracle/synthesis.md`
- Oracle Gaps (all 5 questions answered, 9 contradictions resolved) -- `/Users/callumcowie/repos/Aether/.aether/oracle/gaps.md`
- Midden records (17 real failure entries) -- `/Users/callumcowie/repos/Aether/.aether/data/midden/midden.json`
- Codebase static analysis (11,272 lines, 418 `2>/dev/null` patterns, 104 `|| true` patterns)
- [Composio: Why AI Pilots Fail in Production](https://composio.dev/blog/why-ai-agent-pilots-fail-2026-integration-roadmap) -- MEDIUM confidence
- [ML Mastery: 5 Production Scaling Challenges for Agentic AI](https://machinelearningmastery.com/5-production-scaling-challenges-for-agentic-ai-in-2026/) -- MEDIUM confidence
- [Axify: What Is Dead Code? A 2025 Guide](https://axify.io/blog/dead-code) -- MEDIUM confidence
- [FreeCodeCamp: How to Refactor Complex Codebases](https://www.freecodecamp.org/news/how-to-refactor-complex-codebases/) -- MEDIUM confidence
- [Sean Goedecke: Mistakes Engineers Make in Large Codebases](https://www.seangoedecke.com/large-established-codebases/) -- MEDIUM confidence
- [IEEE: Refactoring, Bug Fixing, and New Development Effect on Technical Debt](https://ieeexplore.ieee.org/document/9226289/) -- HIGH confidence (peer-reviewed)

---
*Pitfalls research for: Aether v2 production hardening*
*Researched: 2026-03-23*
