# Pitfalls Research

**Domain:** Adding integration wiring to an existing AI agent orchestration system (markdown playbook files + threshold tuning)
**Researched:** 2026-03-14
**Confidence:** HIGH (system read directly; all playbooks and aether-utils.sh inspected; prior v1.0/v1.1 milestone audits reviewed)

---

## Critical Pitfalls

### Pitfall 1: Playbook Edit Makes an Instruction Competing Rather Than Additive

**What goes wrong:**
A new instruction in a playbook (continue-advance.md, build-wave.md, etc.) conflicts with an existing instruction rather than adding to it. The agent cannot satisfy both simultaneously, so it satisfies whichever appears earlier in the file and silently ignores the new one. The problem is invisible — the agent produces output, passes verification, and no error is surfaced. The integration wiring simply doesn't fire.

For example: continue-advance.md Step 2 already says "Do NOT record a learning if it wasn't actually tested." Adding a new memory-capture call for hypothetical decisions violates that guard, so the agent skips the new call without reporting it. The developer sees no failure — memory just stays empty.

**Why it happens:**
Playbooks are read as natural language instructions by an AI. When two instructions conflict, the agent disambiguates based on position and apparent authority. Earlier instructions in a file read as the "base rule," later additions read as "qualifications." Additions that look like exceptions to the base rule get silently subordinated.

This is different from code, where a compiler catches contradictions. There is no linter for playbook instruction conflicts.

**How to avoid:**
Before adding any instruction, identify the closest existing instruction it could conflict with. State the relationship explicitly: "In addition to the existing guard at Step X..." or "This fires even when [existing condition] because...". Never assume a new instruction is additive by default — treat it as potentially conflicting until proven otherwise.

Test the specific instruction in isolation: spawn a test build with the new playbook and check whether the new behavior fires AND whether the existing behavior still fires. Both must be true.

**Warning signs:**
- New behavior is described in the playbook but produces no output during a build cycle
- Step X completes successfully but Step X+1 behavior (the new call) never executes
- Agent reports completing a step that should have triggered the new integration, but no artifact (pheromone, midden entry, memory capture) was created
- "Silently skipped" appears in agent output about the new call

**Phase to address:**
Phase that adds memory-capture calls to build-wave.md and continue-advance.md. Each new instruction must be verified against all existing instructions in the same step.

---

### Pitfall 2: Lowering the Midden Threshold Creates a Noise Spiral

**What goes wrong:**
The midden REDIRECT auto-emission threshold is currently 3+ occurrences of the same error category (continue-advance.md Step 2.1c). If this is lowered to 1 or 2, the system emits REDIRECT pheromones after a single ephemeral failure (a flaky test, a network timeout, a file-not-found on a path that was later created). Those REDIRECTs persist for 30d. Future builders read them as hard constraints. They avoid the "failing" category entirely — even after it has been fixed.

The spiral: lower threshold → more REDIRECTs → builders avoid areas → areas go untested → more failures in those areas → even more REDIRECTs → builders treat entire subsystems as off-limits.

This is not hypothetical. The system already shows it can happen: REDIRECT signals have "high priority" and are described in the system as "hard constraints — avoid this." A false REDIRECT is a colony-wide constraint issued on weak evidence.

**Why it happens:**
The 3+ threshold was set deliberately. The v1.0 milestone audit notes the threshold prevents ephemeral failures from becoming permanent signals. Lowering it in a playbook edit without understanding the downstream effect on REDIRECT persistence (30d TTL) and priority (hard constraint) misses that the threshold is a calibration, not an arbitrary number.

**How to avoid:**
Do not lower the midden threshold below 3 without also either (a) reducing the REDIRECT TTL, (b) adding an expiry path when the failure stops recurring, or (c) changing REDIRECT to FEEDBACK for auto-emitted signals. If the real problem is that single-occurrence failures don't reach the midden at all (because midden-write is never called), fix the write path — don't lower the aggregation threshold.

The correct fix for "midden is empty" is to add more midden-write call sites (more places that call it), not to lower the threshold at which those calls produce consequences.

**Warning signs:**
- REDIRECT pheromones accumulate rapidly after lowering threshold
- Builder prompts contain many auto:error signals
- Builders begin skipping tasks or reporting "avoiding pattern X" for things that succeeded recently
- /ant:pheromones shows 5+ active REDIRECT signals with source "auto:error"

**Phase to address:**
Phase that addresses midden failure tracking. Fix write path first, evaluate threshold separately only after observing actual accumulation rates.

---

### Pitfall 3: Memory-Capture Calls Produce Observation Count Inflation Without Signal Value

**What goes wrong:**
memory-capture is called at new decision points and failure points. Each call runs learning-observe internally, which increments the observation count for that content hash. If the content is too generic (e.g., "Builder failed on task" or "Phase completed"), many distinct events hash to similar content, inflate observation counts, and trigger auto-promotion to QUEEN.md for content that carries no actionable information.

QUEEN.md fills with entries like "Builder failed on task" (observed 4 times, promoted to pattern). Builders receive this wisdom in colony-prime and learn nothing — or worse, learn to be pessimistic about tasks in general.

The signal-to-noise ratio in colony memory degrades. Over time, the highest-observation-count items are the most generic events, not the most instructive patterns.

**Why it happens:**
memory-capture uses content hashing to detect recurrence. The hash is over the full content string. If content strings are not specific enough (no task name, no category, no distinguishing detail), different events produce similar enough strings that manual review sees them as identical. The observation counter then inflates for a vague claim.

The system has a 30-instinct cap and a 20-phase-learning cap, but these caps evict by lowest confidence, not by specificity. A vague pattern with 5 observations and 0.7 confidence evicts a specific pattern with 2 observations and 0.7 confidence.

**How to avoid:**
Every memory-capture call site must include enough specifics to make the content hash unique to the event class, not the generic event type. Good: `"Builder ${ant_name} failed on task ${task_id}: ${blockers[0]}"`. Bad: `"Builder failed"`.

Add a specificity test before merging any new memory-capture call: if the content could plausibly recur verbatim across many unrelated events, it is too generic. The content should describe a pattern, not an occurrence.

**Warning signs:**
- QUEEN.md contains entries like "Phase completed without notable patterns" or "Builder reported failure"
- /ant:memory-details shows high observation counts for short, vague strings
- Instinct list contains entries with triggers like "when a task arises" or "when a failure occurs" (no specifics)
- colony-prime prompt_section output grows in length without growing in specificity

**Phase to address:**
Phase that wires memory-capture into new call sites. Every new call site must include a content format review.

---

### Pitfall 4: Playbook Instruction Precision vs. Flexibility Miscalibration

**What goes wrong:**
The current problem is that agents skip steps (too loose instructions). The fix is to make instructions more precise. But there is a failure mode in the other direction: instructions that are so prescriptive that the agent interprets them as a strict sequence rather than a pattern, and fails the step entirely if a minor condition is unmet rather than adapting.

Example: If continue-advance.md is edited to say "For each learning, run memory-capture with content exactly equal to learning.claim" — and a learning has a null claim — the agent hits the null, cannot satisfy the exact instruction, and stops the memory pipeline entirely. Under the previous loose instruction, it would have skipped and continued.

The paradox: making instructions more precise catches the "skipping" problem but can introduce "hard stop on edge case" failures that are harder to diagnose.

**Why it happens:**
Natural language precision has a different failure mode than code precision. In code, you handle null explicitly. In a playbook instruction, adding precision can inadvertently remove the agent's discretion to handle edge cases. The agent reads "exactly equal to" as a hard requirement, not a default.

This is especially acute in a system that already has many "CRITICAL:", "MANDATORY:", and "STOP if" markers (all found throughout build-wave.md and continue-verify.md). Adding more strong imperatives raises the overall rigidity of the playbook, making edge-case recovery harder.

**How to avoid:**
For each new instruction, add an explicit graceful degradation path: "If [condition is unmet], skip silently and continue. Never block step advancement due to memory-capture failure." The existing playbooks do this well (e.g., `2>/dev/null || true` in bash snippets, "If any pheromone call fails, log the error and continue").

Mirror that pattern in every new instruction. The integration wiring should be opportunistic, not gating.

**Warning signs:**
- Phase advancement stops at a step that previously worked fine
- Error message from a new step references a null or missing field
- "MANDATORY" or "CRITICAL" keyword added to a new integration step causes it to block on edge cases
- Agent reports "cannot complete step X" for steps that are supposed to be non-blocking

**Phase to address:**
All phases that edit playbooks. Every new integration instruction must include an explicit non-blocking degradation path.

---

### Pitfall 5: Backward Compatibility Breaks at the Schema Layer (Not the Playbook Layer)

**What goes wrong:**
Playbook edits often add new bash calls that read from or write to COLONY_STATE.json or pheromones.json. If those calls assume a field exists that was added in v1.0 but is absent in colonies initialized before v1.0, the call fails silently (or loudly) on real-world colony state files. The playbook itself looks correct — the call is properly written — but it breaks on older state.

This is particularly dangerous for midden-write and memory-capture calls that are added to build-wave.md, because build is run on every phase, on every project where Aether is installed. Any colony with state pre-dating the field will fail at the new call.

**Why it happens:**
The system has an auto-upgrade path in continue-verify.md (version field check, upgrade to v3.0 state). But this path only fires when /ant:continue is run. A colony that runs /ant:build without having run /ant:continue since the upgrade may have old-format state. New calls that assume v3.0 fields (e.g., `.phase.number` in build-wave.md line 344) will return null for older state.

**How to avoid:**
Use defensive jq reads for every new field access. The pattern already used in build-wave.md is the right one: `jq -r '.session_id | split("_")[1] // "unknown"'` — the `// "unknown"` provides a safe fallback. Apply this pattern to every new field read added as part of integration wiring. Never assume a field exists; always provide a fallback.

For new fields being written (not just read), check whether the write is idempotent on both old and new state format. If the write adds a field that didn't exist, it must work on state files that lack it.

**Warning signs:**
- Integration wiring works in the dev environment (current colony) but fails when tested on a fresh colony with /ant:init
- jq reads return null or empty string unexpectedly
- New integration call produces no output on first run of a build after upgrading Aether
- Error message references a field that was added in v1.0 or v1.1

**Phase to address:**
All phases. Schema defensiveness should be a checklist item for every playbook edit that adds jq reads.

---

### Pitfall 6: The "Wiring Is Correct But Never Observed" Problem

**What goes wrong:**
All the integration calls are correctly added to playbooks. Memory-capture fires. Midden-write fires. Pheromone-write fires. But the output of those calls is never displayed to the user during the flow, and there is no easy way to verify after the fact that they ran. The developer declares the integration complete based on code review of the playbook, not on observed behavior.

Later, when the colony memory remains empty, it turns out one of the calls has a silent failure path (`2>/dev/null || true`) that swallows an error. The integration is wired but broken, and there is no signal that it is broken.

This is the same failure mode that created the v1.2 milestone — the calls exist in the playbooks already, but real-world observation shows the memory is empty (decisions [], instincts [], only 1 phase_learning).

**Why it happens:**
The existing system explicitly suppresses output for auto-integration calls. Step 2.1 of continue-advance.md begins with: "This entire step produces NO user-visible output. All pheromone operations run silently." This is correct for UX (avoid noise), but it means that broken wiring is also silent. The developer has no feedback loop to know whether silent steps are running or silently failing.

**How to avoid:**
Add an observability step to each phase's verification criteria. Before marking a phase complete, require evidence that the integration actually fired — not just that the code was correctly added. Specifically: run a test build cycle on a controlled colony state and check the resulting memory, midden, and pheromone files directly. "Call exists in playbook" is not sufficient evidence. "Colony state shows new entry after test build" is.

For each new integration wire, define its success artifact: what file is modified, what field is populated, what count increases. Test against that artifact.

**Warning signs:**
- Phase declared complete based only on code review of playbook (no runtime verification)
- Test suite checks that bash commands exist in the playbook file (text matching), but not that they produce output
- After completing a build cycle on a fresh colony, memory/midden counts are the same as before
- "|| true" appears on a new integration call that has no other error handling

**Phase to address:**
Every phase. Each phase's success criteria must include at least one runtime verification: run a build cycle and verify a specific artifact was created.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Adding `2>/dev/null \|\| true` to every new integration call | Prevents blocking on errors | Swallows real failures silently; integration may never fire and you won't know | Acceptable only after runtime verification confirms the call succeeds. Never as a substitute for verification. |
| Lowering midden threshold instead of adding write sites | Appears to fix "empty midden" with one change | Creates noise spiral via false REDIRECT signals; degrades colony signal quality | Never. Fix the write path instead. |
| Generic content strings in memory-capture | Simpler to write | Observation inflation; QUEEN.md fills with vague patterns; signal degrades | Never in production call sites. Acceptable in test fixtures only. |
| Verifying playbook edit by reading the playbook | Fast review cycle | Does not catch silent failures; integration may look correct but not fire | Never as sole verification. Code review is prerequisite, not substitute, for runtime verification. |
| Adding "MANDATORY" / "CRITICAL" to integration steps | Emphasizes importance | Raises risk that edge cases hard-stop a phase; integration should be opportunistic | Never for memory/midden/pheromone integration. These are always non-blocking. |

## Integration Gotchas

Common mistakes when connecting the integration wiring.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| memory-capture in build-wave.md | Calling memory-capture with content from variables that are null at call time (e.g., `${blockers[0]}` before blockers array is populated) | Read the bash snippet at each call site in build-wave.md; verify all referenced variables are in scope at that point in the flow |
| pheromone-write for decisions | Emitting a pheromone for every decision, including trivial ones ("used npm install") | Gate on decision significance: decisions with less than 10 words or no rationale should be filtered before emitting |
| midden-write for "all failure points" | Adding midden-write after every failed tool call in a builder, creating thousands of low-signal entries | Write to midden for task-level failures (builder returned status: failed), not for individual tool call failures within a successful task |
| instinct-create in continue | Creating instincts for single-phase patterns ("this worked in phase 3") | Instincts should reflect patterns observed across multiple builds or recurring within a phase, not one-off successes |
| Deduplication via existing pheromone check | The deduplication jq query (`.signals[] \| select(.active == true and .source == "auto:decision" ...)`) may miss signals that decayed but left a hash trace | The current deduplication approach checks only active signals; expired signals with same content can re-emit after TTL. This is acceptable behavior but should be documented as intentional. |

## Performance Traps

Patterns that work at small scale but fail at larger ones.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Running midden-recent-failures with high N on every build | continue takes 5-10 seconds just reading the midden before any work | The current call uses N=5 in build-wave.md, N=50 in continue-advance.md Step 2.1c. The N=50 call is for aggregation (finding 3+ recurring categories), which is correct. Keep N bounded. | midden.json exceeds 500 entries (unlikely in normal use, but possible in failure-heavy projects) |
| colony-prime --compact called per-wave rather than per-build | Repeated context assembly at each wave | --compact is already used; keep it. Do not switch to full colony-prime per-wave. | Not applicable at current scale |
| base64 encoding/decoding in bash for jq output | Slow on very long strings | Already used throughout. Acceptable for the entry counts involved (< 100 per cycle). | Not applicable |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Memory-capture content contains sensitive values from worker output | Task descriptions or failure messages containing API keys, passwords, or file paths get promoted to QUEEN.md and distributed | Before merging any new memory-capture call site, review what content string is constructed. Avoid capturing full worker summaries verbatim — extract specific claim types. |
| midden entries from worker failures contain full error output | Error messages may contain partial secrets or internal paths | The current midden-write call in build-wave.md captures `${blockers[0]}` (first blocker only), not full worker output. Maintain this pattern: capture the category and brief message, not the full stack trace or output. |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Auto-pheromone noise from lowered thresholds | User runs /ant:pheromones and sees 10+ auto-emitted REDIRECTs for normal build variation | Keep existing thresholds; fix write path; let user observe accumulation over real builds before adjusting |
| QUEEN.md fills with low-value patterns | User opens QUEEN.md expecting distilled wisdom, finds generic entries | Enforce specificity at every memory-capture call site. Content must be actionable, not descriptive. |
| Integration steps add 10-15 seconds to every build/continue cycle | User notices build cycles slower with no visible benefit if integration is silent | All new integration calls should be non-blocking and run concurrently where possible. The current pattern (`2>/dev/null || true`) is correct for failure handling. Also batch calls where possible (one jq write vs. multiple). |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Playbook edit:** Instruction added to playbook — verify it does not conflict with an earlier instruction in the same step. Read the full step, not just the new paragraph.
- [ ] **Memory-capture call:** Call added at new site — verify the content string is specific (includes task ID, ant name, or similar discriminator). Run a test build and check learning-observations.json for the new entry.
- [ ] **Midden threshold change:** Threshold lowered — verify REDIRECT TTL and priority are still appropriate. Check /ant:pheromones after a test build cycle with the new threshold.
- [ ] **Integration wiring:** All new bash calls use `2>/dev/null || true` — verify each call also has a runtime test that proves it fires, not just that it does not error.
- [ ] **Backward compatibility:** New jq read added — verify it uses `// "default"` fallback for every new field, and test on a fresh /ant:init colony state (not just the current dev colony).
- [ ] **Phase complete:** Build/continue cycle ran in test — verify specific artifacts were created: count entries in midden.json, check pheromones.json for new auto:decision signals, check learning-observations.json for new entries.

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Competing instruction — new step silently skipped | LOW | Find the conflicting instruction; add explicit relationship clause ("In addition to..."). Re-run test build cycle. |
| Midden noise spiral from lowered threshold | MEDIUM | Restore threshold to 3+. Run `jq '.entries \|= map(select(.timestamp > "recent_cutoff"))' midden.json` to prune false entries. Manually expire false REDIRECT pheromones. |
| Memory capture observation inflation from generic content | MEDIUM | Edit content strings at all new call sites to add specifics. Manually remove inflated observations from learning-observations.json for the generic content hash. |
| Integration wiring fire but never observed | LOW | Add runtime verification step to the phase plan before marking complete. Run a controlled build on a fresh colony. |
| Backward compat break on old colony state | LOW | Add `// "fallback"` to the new jq read. Test on a colony initialized with /ant:init (fresh state, not current dev state). |
| Phase stops on edge case from over-precise instruction | LOW | Add graceful degradation clause to the instruction: "If [condition], skip silently." Re-run the failed continue/build. |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Competing instructions in playbook | Any phase editing playbooks | Read full step context before adding; test build confirms both old and new behavior fire |
| Midden threshold noise spiral | Phase addressing midden failure tracking | Fix write call sites first; observe accumulation before adjusting threshold |
| Memory-capture noise (generic content) | Phase wiring memory-capture to new call sites | Check learning-observations.json for new entries after test build; reject generic content strings |
| Instruction precision over-correction | All playbook editing phases | Every new integration instruction includes explicit `|| true` and skip clause |
| Schema backward compat breaks | All phases adding jq reads | Test on fresh /ant:init colony state, not only current dev state |
| Wiring correct but never observed | Every phase | Success criteria requires runtime artifact verification, not just code review |

## Sources

- Direct inspection of `.aether/docs/command-playbooks/continue-advance.md` (Steps 2, 2.1, 2.1a-2.1e, 2.1.5, 2.1.6) — complete existing integration wiring visible; threshold at 3+ occurrences confirmed (HIGH confidence)
- Direct inspection of `.aether/docs/command-playbooks/build-wave.md` (Steps 5.1, 5.2) — midden-write and memory-capture call sites visible; content string patterns at current sites reviewed (HIGH confidence)
- Direct inspection of `.aether/docs/command-playbooks/continue-verify.md` — auto-upgrade path scope confirmed; verification loop gate behavior confirmed (HIGH confidence)
- Direct inspection of `.aether/aether-utils.sh` (midden-write at line 8211, memory-capture at line 5402, learning-observe at line 5148) — implementation behavior confirmed; graceful degradation patterns confirmed (HIGH confidence)
- `.planning/PROJECT.md` — v1.2 goal stated: "decisions [], instincts [], only 1 phase_learning" confirming memory is empty despite wiring existing (HIGH confidence)
- `.planning/MILESTONES.md` — v1.0 shipped all 12 requirements; v1.1 ships 20 requirements; gap diagnosis that wiring exists but isn't producing output (HIGH confidence)
- Anthropic multi-agent engineering blog — silent failure propagation in agent orchestration; instruction competition in natural language prompts (MEDIUM confidence)
- General knowledge of LLM playbook/prompt instruction following — priority ordering, conflicting instruction resolution, specificity vs. flexibility tradeoff (MEDIUM confidence; applies to all documented agent systems)

---
*Pitfalls research for: Adding integration wiring to AI agent orchestration system (Aether v1.2 Integration Gaps)*
*Researched: 2026-03-14*
