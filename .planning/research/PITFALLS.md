# Pitfalls Research

**Domain:** Multi-agent colony system hardening (v4.4)
**Researched:** 2026-02-04
**Confidence:** HIGH (grounded in Aether's own field-test data + current multi-agent research)

**Scope:** This research covers NEW pitfalls for v4.4 features only. Pitfalls from v1-v4 (context rot, infinite spawning, JSON corruption, prompt brittleness) are already mitigated. See prior research for those.

---

## Critical Pitfalls

Mistakes that cause rewrites, system instability, or fundamental design flaws.

### CP-1: Recursive Spawning Hits Claude Code's Task Tool Limitation

**What goes wrong:** Aether's worker specs say ants can spawn sub-ants up to depth 3, but Claude Code's Task tool is NOT available to subagents. Subagents spawned via Task cannot themselves use Task to spawn further subagents. The "recursive spawning" that the v4.4 design envisions (Phase Lead spawns workers, workers spawn sub-workers) is broken at depth 2+ in current Claude Code.

**Why it happens:** Claude Code exposes a restricted tool set to subagents: Bash, Glob, Grep, Read, Write, Edit, WebFetch, WebSearch -- but NOT Task. This is a platform constraint, not an Aether bug. GitHub issue #4182 (closed as duplicate, 8 upvotes) confirms this limitation. The current worker specs include spawning instructions that cannot actually execute in subagent context.

**How to detect early:**
- Test actual recursive spawning in a live run -- does a depth-2 ant successfully spawn a depth-3 ant?
- Check if the v5 field test ever achieved depth-2+ spawning (review spawn logs)
- The field notes don't mention recursive spawning succeeding or failing -- it may never have been tested at depth 2+

**Prevention strategy:**
- Before implementing deeper recursive delegation, validate the platform capability with a simple test: spawn a subagent, have it attempt to spawn another subagent via Task tool, observe whether it succeeds
- If Task tool is unavailable to subagents (likely), there IS a viable workaround: `claude -p` via Bash. The parent ant writes context to a temp file, invokes `claude -p` with the worker spec, and reads a structured result file. This loses structured Task tool benefits (progress tracking, type system) but enables recursive delegation
- Design the recursive spawning feature to gracefully degrade: if Task tool is available, use it; if not, fall back to `claude -p` wrapper; if neither works, report the gap to parent and handle inline
- Consider whether the hub-and-spoke model (Queen as sole spawner) is actually the CORRECT architecture given platform constraints, rather than fighting the platform. The Queen already serializes context well -- recursive delegation adds complexity with unclear benefit

**Recovery if it happens:** Fall back to hub-and-spoke with enhanced Phase Lead plans that pre-decompose work so deep nesting is unnecessary.

**Phase to address:** FIRST -- this is a foundational platform constraint that shapes every other feature. Must be validated before designing recursive delegation.

**Confidence:** HIGH -- platform limitation confirmed via GitHub issue #4182 and Claude Code subagent documentation.

---

### CP-2: Context Telephone -- Information Degrades at Each Delegation Level

**What goes wrong:** When Ant A spawns Ant B, it serializes relevant context into the Task prompt. Ant B starts with zero conversation history. If Ant B spawns Ant C (via workaround or future Task tool support), it must serialize its understanding -- already a lossy compression of A's context -- into C's prompt. By depth 3, the original intent is mutated or lost entirely. Google Research calls this the "telephone game" of agent delegation chains.

**Why it happens:** Each Task invocation creates a fresh agent with its own context window. There is no shared memory between parent and child during execution. The parent must decide what context to include, and this decision is itself an LLM judgment call that can be wrong. Multi-agent research documents "intent mutation in delegation chains" where each hop introduces lossy compression.

**How to detect early:**
- Depth-3 agents produce work that diverges from the colony goal
- Sub-agents ask for clarification that was already provided to their parent
- Results from deep delegation chains are lower quality than depth-1 results
- The same task produces different results when executed at depth 1 vs depth 3

**Prevention strategy:**
- Include the VERBATIM colony goal, current phase context, and active pheromones at EVERY spawn level (Aether already does this in worker specs -- preserve this pattern)
- Add a "delegation chain" section to every Task prompt: "Colony goal > Phase goal > Parent task > Your task" so deep agents see the full hierarchy without relying on intermediate summarization
- Keep context lean at each level: pass the colony goal verbatim (NEVER summarize it), pass only relevant pheromones, pass the specific task with acceptance criteria
- Set a quality gate: if a sub-agent's output doesn't reference or align with the colony goal, the parent should re-do the work itself rather than delegating again
- Limit effective depth to 2 (Queen -> Phase Lead -> Worker) until proven that depth 3 adds value. The field test showed most value at depth 1-2

**Recovery if it happens:** Parent ant completes the sub-task inline with full context rather than re-delegating deeper.

**Phase to address:** Same phase as recursive spawning implementation.

**Confidence:** HIGH -- well-documented in multi-agent literature (Google Chain of Agents, ROMA framework, MegaAgent research).

---

### CP-3: Auto-Reviewer Creates Blocking Bottleneck (Review Fatigue)

**What goes wrong:** Auto-spawned reviewer ants block builders from completing their work or create a flood of low-value findings that the colony must process. If every builder output gets reviewed, and reviews produce "request changes" verdicts, the build-review-rebuild cycle loops indefinitely. The 2025 Stack Overflow Developer Survey found 46% of developers distrust AI code review accuracy. Alert fatigue -- desensitization from too many irrelevant warnings -- erodes trust faster than missed issues.

**Why it happens:** The watcher ant already scores everything 8/10 (field note 24). Adding an auto-reviewer that is ALSO uncalibrated will either (a) rubber-stamp everything (useless) or (b) flag too many issues (blocking). When reviewers cannot distinguish "critical bug" from "style preference," everything becomes noise. One real-world team iterated on thresholds for three weeks before finding a workable false positive rate of ~12.5%.

**How to detect early:**
- Review scores cluster around a single value (already happening at 8/10)
- Build-review cycles exceed 2 iterations for the same task
- Colony throughput drops after adding auto-review (more spawns, same or less output)
- User starts dismissing review findings without reading them
- Auto-reviewer flags the same types of issues repeatedly without the colony learning from them

**Prevention strategy:**
- Auto-reviewer must be ADVISORY only, never blocking. It produces findings; the Queen or Phase Lead decides whether to act. Only CRITICAL findings (security vulnerabilities, data loss risks, crashes) should trigger a rebuild request
- Fix the watcher scoring rubric BEFORE adding auto-review. Without calibrated scoring, auto-review is theater. The rubric must produce meaningfully different scores for different quality levels -- test by running watcher on intentionally bad code vs good code and verifying scores differ
- Implement severity-gated display: show top 5 findings by severity in the build output, full report available via /ant:status
- Set a max-iteration cap on build-review cycles: 2 iterations max, then log remaining findings as tech debt and proceed
- Use pheromone signals to control review intensity: strong FEEDBACK pheromone about quality triggers deeper review; no quality signals triggers lightweight check only
- Track auto-reviewer accuracy over time: if findings are dismissed >50% of the time, the reviewer needs recalibration

**Recovery if it happens:** Disable auto-reviewer for the current phase, collect the backlog of findings into a tech debt report, let the user decide what matters.

**Phase to address:** Must be implemented AFTER watcher scoring rubric is fixed (field note 24). Without calibrated scoring, auto-review is meaningless.

**Confidence:** HIGH -- supported by Aether field data (flat 8/10 scores) and industry research on AI code review accuracy (CodeAnt, Stack Overflow survey).

---

### CP-4: Same-File Parallel Write Conflicts (Last-Write-Wins)

**What goes wrong:** Two or more builder ants modify the same file concurrently. Each ant reads the file at spawn time, makes changes in isolation, and writes back. The last ant to write overwrites all changes from earlier ants. This already happened in the v5 field test (note 13): "one builder had to reapply Phase 1 changes" because another builder overwrote its work.

**Why it happens:** Claude Code Task tool agents operate in isolated contexts. They share the filesystem but have no awareness of each other's pending changes. Aether's file-lock.sh prevents simultaneous byte-level writes (race condition), but it does NOT prevent semantic conflicts -- two agents can both read version N of a file, make different changes, and write version N+1a and N+1b, with N+1b destroying N+1a's changes.

**How to detect early:**
- Git diff shows expected changes are missing after a parallel wave completes
- Watcher reports that previously verified work has reverted
- Builder reports "file doesn't contain expected code" when building on another builder's work
- The same learning keeps re-appearing: "parallel workers writing to same file causes conflicts" (field notes 10, 13, 30)

**Prevention strategy:**
- **Task-level file ownership (primary approach):** Phase Lead or Queen must analyze which tasks touch which files and assign overlapping-file tasks to the SAME worker. This is the approach the field test already learned (notes 10, 13, 30). Implement as a rule in route-setter-ant.md and build.md
- **File reservation system:** Before a wave starts, the Queen declares which files each worker will modify. Tasks are reassigned to eliminate overlaps. Add a `files_touched` field to task definitions in PROJECT_PLAN.json
- **Sequential fallback for shared files:** If two tasks MUST modify the same file and MUST be separate workers, run them in separate waves (sequential), never in the same wave (parallel)
- **Do NOT use git worktree isolation.** It is the industry standard for multi-agent coding (Cursor 2.0 uses it) but is heavyweight for Aether's shell-based architecture and would require managing multiple working directories. Not appropriate for v4.4

**Recovery if it happens:** `git diff` to identify missing changes, then re-run the overwritten worker's tasks on the current file state.

**Phase to address:** Early -- this is a known, field-tested bug. Should be in the first batch alongside bug fixes.

**Confidence:** HIGH -- directly observed in Aether v5 field test, confirmed as common pattern in multi-agent coding literature.

---

### CP-5: Two-Tier Learning Creates Stale Global Knowledge

**What goes wrong:** Global learnings (stored in `~/.aether/`) get applied to projects where they are wrong or harmful. A learning like "always use ESM imports" is correct for a Node.js project but wrong for a legacy CommonJS project. As the global tier accumulates learnings from diverse projects, the noise-to-signal ratio increases and learnings start contradicting each other.

**Why it happens:** Promotion from project-local to global is a classification problem: "is this learning universal?" LLMs are bad at this judgment because they lack ground truth about all possible project contexts. The 2025 Agentic Memory paper found that retrieval accuracy degrades as similar memories accumulate, making disambiguation harder, and that "utility-based deletion prevents memory bloat and error propagation, yielding up to 10% performance gains over naive strategies."

**How to detect early:**
- Global learnings contradict project-local conventions (global says "use TypeScript" but project is JavaScript)
- Colony applies a global learning and a watcher flags the result as wrong
- Same learning appears in global tier with conflicting versions from different projects
- Global learning count exceeds ~30 entries without curation

**Prevention strategy:**
- **Conservative promotion:** Default to local-only. A learning must meet ALL of: (a) validated across 2+ projects, (b) not specific to a tech stack/framework/version, (c) user-approved. Never auto-promote
- **User-approved promotion:** When a learning seems universal, present it at project completion (not mid-execution). Batch all promotion candidates into a single review. This avoids promotion spam during builds
- **Tagged learnings:** Every global learning carries metadata: `{source_project, promoted_at, applied_count, success_count, failure_count, tech_tags}`. If failure_count exceeds threshold, auto-demote back to the source project's local tier
- **Scoped retrieval:** When applying global learnings, filter by relevance to current project. A learning tagged `[node, esm, packaging]` should not apply to a Python project. Use the project's tech stack (from COLONY_STATE or colonizer output) as a filter
- **Decay mechanism:** Global learnings that haven't been applied in 6 months lose strength. Reuse the pheromone decay math -- learnings have a half-life. Eventually they fall below threshold and are archived, not deleted
- **Hard cap:** Maximum 50 global learnings. When adding the 51st, the least-used/oldest learning is evicted. This forces quality over quantity

**Recovery if it happens:** Add a `suppress` mechanism: user marks a global learning as "not applicable to this project" without deleting it from the global tier. The colony tracks suppressions to inform future scoped retrieval.

**Phase to address:** Implement the two-tier structure with local-only first. Delay the promotion mechanism to a separate phase. Start with: (1) local learnings enhanced, (2) global tier exists but is empty, (3) manual promotion only. Auto-promotion is a later feature.

**Confidence:** MEDIUM -- the architecture is well-supported by research but the promotion heuristics are unproven in Aether's specific context. Will need empirical tuning.

---

### CP-6: Adaptive Mode Selects Wrong Complexity Level

**What goes wrong:** The system classifies a complex project as "simple" and runs in lightweight mode, skipping Phase Lead planning, caste specialization, and watcher verification. Critical tasks get under-planned and under-verified. Conversely, a simple project gets classified as "complex" and drowns in colony overhead (already experienced in field note 31 -- 21 tasks of config changes with full colony machinery).

**Why it happens:** Complexity classification is inherently ambiguous. A project with 5 tasks might be simple (config changes) or complex (distributed system migration). Task count is a poor proxy for complexity. Google DeepMind's 2025 "Scaling Agent Systems" paper found that "coordination yields diminishing or negative returns once single-agent baselines exceed ~45%" -- the threshold for "when multi-agent helps" is task-dependent, not project-dependent. Their research also showed sequential reasoning tasks degraded 39-70% with multi-agent overhead.

**How to detect early:**
- Lightweight mode produces incomplete or buggy results on a phase that seemed simple
- Full colony mode on a simple phase takes 3x longer with no quality improvement vs. inline execution
- User manually overrides the adaptive mode selection frequently
- Watcher scores are consistently low in lightweight mode (indicating the mode was wrong)

**Prevention strategy:**
- **User confirmation always:** The adaptive mode SUGGESTS, the user confirms. One-line prompt: "Suggested: lightweight mode (3 tasks, no file overlap, no novel domain). Accept? [Y/override]"
- **Conservative default:** When uncertain, default to full colony mode. Unnecessary overhead is recoverable; missed critical review on a complex phase is not
- **Multi-factor classification:** Do NOT use task count alone. Score based on: (a) number of distinct files touched, (b) file overlap between tasks, (c) task dependencies (sequential chain = complex), (d) domain novelty (no prior learnings = complex), (e) pheromone signal intensity (strong FOCUS/REDIRECT = complex situation). Require 3+ "simple" factors to trigger lightweight mode
- **Per-phase, not per-project:** Mode selection happens per phase, not per project. A project might have simple config phases and complex architecture phases. The filmstrip project had phases 3-5 that were simple but phases 1-2 that were genuinely complex
- **Ratchet mechanism:** Start in the suggested mode. If a watcher or reviewer flags issues, automatically escalate to full colony mode for the NEXT phase. Do not try to change mode mid-phase

**Recovery if it happens:** Re-run the phase in full colony mode. The lightweight output becomes context that the full colony can build on rather than starting from scratch.

**Phase to address:** Late -- adaptive mode depends on having reliable quality signals (calibrated watcher, functional auto-reviewer) to detect when the wrong mode was selected.

**Confidence:** MEDIUM -- concept well-supported by DeepMind research, but specific thresholds need empirical tuning in Aether.

---

### CP-7: Archivist Ant Deletes or Archives Important Files

**What goes wrong:** The archivist ant identifies a file as "stale" and archives or recommends deletion, but the file was actually critical -- just rarely accessed. Examples: disaster recovery configs, seasonal feature flags, migration scripts needed only during upgrades, test fixtures accessed only by CI, environment-specific configs (staging, production).

**Why it happens:** Staleness heuristics based on last-modified time or import analysis are unreliable. Meta's SCARF framework found that static analysis alone misidentifies code used via reflection, dynamic imports, or rare execution paths. The problem compounds with LLM-based analysis because the LLM may not understand the full deployment context (CI pipelines, multi-environment setups) from reading files alone.

**How to detect early:**
- Tests start failing after an archivist run
- Builds break because a config or fixture file is missing
- User asks "where did X go?" after an archivist phase
- Git log shows archivist-archived files that are later manually restored

**Prevention strategy:**
- **Report only, never act:** The archivist produces a REPORT with three confidence tiers: (a) confidently stale -- recommend archival with reasoning, (b) probably stale -- recommend user review, (c) unclear -- flag for awareness. The USER decides. This matches field note 14's design exactly -- follow it
- **Protected file patterns:** Maintain a hardcoded list of patterns that are NEVER flagged as stale: `*.test.*`, `*.spec.*`, `*.config.*`, `*.env*`, `Dockerfile*`, `*.lock`, `migrations/*`, `.github/*`, `.ci/*`, `*.fixture.*`. These are commonly low-access but critical
- **Cross-reference before flagging:** Before flagging any file, the archivist must check: (a) is it imported/required by any other file? (b) is it referenced in package.json, CI configs, Dockerfiles, or Makefiles? (c) does it appear in any test configuration? If yes to ANY, it is NOT stale regardless of modification date
- **Dry run enforcement:** The archivist's first 3 runs on any project MUST be report-only. Only after the user has validated accuracy and explicitly opted in should it gain the ability to move files
- **Git-based safety:** Before any archival, confirm the file is committed to git. Archival means moving to `.archived/` with a manifest file explaining why, not deletion. Recovery is always `git checkout -- <file>` or moving back from `.archived/`

**Recovery if it happens:** `git checkout -- <file>` for immediate restore. Review and update protected file patterns to prevent recurrence.

**Phase to address:** Late -- the archivist is a nice-to-have, not a core fix. Implement after bug fixes, conflict prevention, auto-reviewer, and adaptive mode.

**Confidence:** HIGH -- false deletion is a well-documented problem (Meta SCARF framework, Varonis archival research, community reports of AI-generated orphan files).

---

## Technical Debt Patterns

Patterns that don't break immediately but accumulate cost over time.

| Pattern | What Accumulates | Warning Sign | Prevention |
|---------|-----------------|--------------|------------|
| Learning duplication | Same learning stored multiple times with slight wording variations across phases | `memory.json` phase_learnings array contains near-duplicates ("parallel workers cause conflicts" vs "same-file parallel writes lose data") | Implement semantic deduplication before storage: hash content, skip if similarity > 0.8 to existing entry |
| Event log bloat | events.json grows without bound as phases accumulate | File exceeds 100KB, worker startup slows from reading full events | Implement event archival at phase boundaries (same pattern as activity-log-init) |
| Spawn outcome drift | Alpha/beta values in spawn_outcomes accumulate indefinitely, making early outcomes permanently dominate | Bayesian confidence barely moves after 20+ spawns even when recent outcomes differ | Implement sliding window: only count the last 20 spawn outcomes per caste, not lifetime total |
| Pheromone signal accumulation | Decayed pheromones still present in signals array consuming read time | pheromones.json contains 20+ signals, most at negligible strength | pheromone-cleanup exists but must be called at every phase boundary, not just on demand |
| Worker spec size creep | As features are added (activity logging, spawn gates, post-validation, reviewer triggers), worker specs grow past 400 lines | Spawning cost per agent increases; spec injection consumes large portion of Task prompt token budget | Extract shared infrastructure (spawn gates, activity logging, post-validation) into a worker-common section that all specs reference |
| Stale decision log | Decisions from early phases remain in memory.json but are no longer relevant | Decision array at cap (30) with entries from phase 0 still present, colony confused by outdated constraints | Auto-archive decisions older than 5 phases; keep recent decisions that are still constraint-relevant |

---

## Performance Traps

Issues that degrade colony throughput or user experience.

| Trap | Impact | Detection | Mitigation |
|------|--------|-----------|------------|
| Context serialization cost | Queen serializes full state (6 JSON files) into every worker prompt; can be 5K-10K tokens per spawn, reducing worker's available context | Measure worker prompt token count; if >30% is state serialization, overhead is excessive | Only include relevant state per worker type: builders need learnings + task + pheromones; watchers need errors + learnings + pheromones; scouts need pheromones only |
| Sequential wave bottleneck | Queen spawns workers one-at-a-time within waves, waiting for each to complete | Phases with 5+ independent tasks still take 5x single-task time despite being parallelizable | Investigate parallel Task invocation (if Claude Code supports it) or `claude -p` via Bash for fire-and-forget parallel spawning with file-based result collection |
| Activity log I/O contention | Multiple parallel workers append to the same activity.log file simultaneously | Log entries appear interleaved, corrupted, or missing when workers run concurrently | One log file per worker (`activity-{worker_id}.log`), merged by Queen after wave completes |
| Memory.json full read on every spawn | Every worker reads full memory.json at startup even if it only needs the 3 most recent learnings | Worker startup overhead grows linearly with project lifetime | Add a `memory-recent` subcommand to aether-utils.sh that returns only last N learnings and decisions |
| Auto-reviewer doubles spawn count | Every build task gets a follow-up review task, doubling the number of spawns per phase | Colony throughput halves; token costs double; user waits twice as long | Review only at wave boundaries (batch review after N tasks complete), not per-task. Or: only review tasks touching files flagged by pheromones |
| Global learning retrieval overhead | At project start, colony reads all 50 global learnings and filters for relevance | Colonize/init command is slow; irrelevant learnings consume context | Pre-filter global learnings by tech-stack tags before injecting into worker context. Store a project-level "applicable globals" cache |

---

## UX Pitfalls

Issues that degrade the user's experience of the colony.

| Pitfall | User Impact | Detection | Mitigation |
|---------|-------------|-----------|------------|
| Auto-continue removes user agency | User loses ability to course-correct between phases; colony runs ahead and makes irreversible changes | User uses /ant:pause-colony frequently; user expresses surprise at unexpected changes | Auto-continue requires explicit opt-in per session. Prompt once: "Build remaining N phases automatically? [Y/phase-by-phase]". Default to phase-by-phase |
| Review findings overwhelm user | Auto-reviewer produces 15+ findings per phase; user can't process them all and starts ignoring all findings | User stops reading review reports; review acceptance rate drops below 20% | Display top 5 findings by severity in build output. Full report available via /ant:status. Never show LOW-severity findings in main output |
| Adaptive mode feels opaque | System chooses lightweight or full mode without explaining why; user can't build mental model | User asks "why did it skip the planner?" or is surprised when mode changes between phases | ALWAYS display the mode decision and brief reasoning: "Mode: lightweight (3 tasks, no file overlap, no dependencies). Override with /ant:redirect" |
| Learning promotion spam | System constantly asks "promote this learning to global?" during builds | User starts auto-approving without reading; worthless learnings pollute global tier | NEVER prompt for promotion during execution. Batch ALL promotion candidates at project completion as a single review step |
| Archivist creates anxiety | User worries the archivist might delete something important even though it only reports | User refuses to run archivist; user checks git status nervously after every archivist report | Frame archivist output explicitly as SUGGESTIONS: "Found 3 potentially stale files. No action taken. Review at your convenience:" |
| Context collapse during animated output | Animated build indicators (spinners, progress bars) consume LLM context tokens, triggering compaction mid-display (field note 13) | Display cuts off mid-phase; user sees "[context compacted]" during build output | Animations MUST use ANSI escape sequences via Bash tool, NOT LLM-generated text tokens. The LLM should invoke a Bash command that renders the animation, keeping zero animation tokens in the conversation context |
| Recursive delegation feels like a black box | With recursive spawning, user can't tell what's happening 2-3 levels deep; colony feels unpredictable | User asks "what's happening?" during long recursive delegation chains | Activity log must capture entries from ALL depth levels. Display delegation tree: "Queen > Phase Lead > Builder > Scout (researching auth library)" |

---

## "Looks Done But Isn't" Checklist

Features that appear complete in code review but fail in practice.

- [ ] **Recursive spawning "works" in spec but not on platform:** Worker specs contain spawning instructions but the Task tool may not be available to subagents. Test with ACTUAL spawning at depth 2+, not just spec review. If it silently fails, the spec is misleading
- [ ] **File conflict prevention "works" for planned files but not emergent ones:** Phase Lead assigns file ownership, but a builder might create an UNEXPECTED new file that another builder also independently creates. Prevention must handle both planned modifications AND unplanned file creation
- [ ] **Auto-reviewer "works" but scores are meaningless:** A reviewer that always says "approved" or always outputs 8/10 is technically functioning but providing zero signal. Validate by running reviewer on intentionally bad code -- scores MUST vary meaningfully across quality levels
- [ ] **Adaptive mode "works" for the test project but not edge cases:** 1 phase, 2 tasks = lightweight. But 1 phase, 2 tasks, both modifying the same critical production file = should NOT be lightweight. Test with edge cases that look simple but aren't
- [ ] **Learning promotion "works" but promoted learnings are project-specific:** "Use filmstrip v2.1.0 for CLI packaging" is NOT a global learning. Promotion logic must distinguish universal patterns ("group same-file tasks to one worker") from project-specific conventions
- [ ] **Activity log append "works" but phase archives are inaccessible:** Fixing the overwrite bug (field note 19) is necessary but not sufficient. Verify that archived phase logs (`activity-phase-N.log`) are actually readable and that the naming scheme doesn't collide across milestones
- [ ] **Pheromone decay "works" after math fix but signals still accumulate:** Fixing the decay formula (field note 17) makes individual signals decay correctly, but the signals array still grows unbounded. Verify pheromone-cleanup runs at every phase boundary
- [ ] **Watcher scoring rubric "works" but is gameable:** A rubric that awards points for PRESENCE of tests/error-handling/types might score a file with trivial empty tests and bare catch blocks as 9/10. Validate rubric rewards QUALITY not merely presence
- [ ] **Context clear prompting "works" but state isn't fully persisted:** System says "safe to /clear" but a critical state update was in LLM working memory, not yet written to JSON. Clear prompting must call validate-state and confirm ALL files pass before suggesting clear

---

## Recovery Strategies

When pitfalls are encountered despite prevention.

| Failure Mode | Immediate Recovery | Long-term Fix |
|--------------|-------------------|---------------|
| Recursive delegation infinite loop (A spawns B spawns A pattern) | `/ant:pause-colony`, review spawn logs, identify the cycle | Add cycle detection: if an ant spawns the same caste for the same task description within a phase, block the spawn and report |
| File overwritten by parallel worker | `git checkout -- <file>` to restore last committed version, then re-run the overwritten worker on current file state | Implement file reservation in Phase Lead planning; enforce during worker spawning |
| Auto-reviewer blocks all progress | Disable auto-reviewer for current phase, proceed with Queen review only | Tune severity thresholds: only CRITICAL blocks; recalibrate against project-specific quality bar |
| Stale global learning causes incorrect implementation | Mark learning as suppressed for this project via `suppress` flag in global tier; fix the bug manually | Add tech-stack filtering to global learning retrieval; track suppression frequency to auto-demote bad learnings |
| Archivist archives needed file | Restore from `.archived/` directory or `git checkout -- <file>`; update protected file patterns | Expand protected patterns list; require cross-reference check before any file can be flagged |
| Wrong adaptive mode selected mid-project | Re-run the affected phase in correct mode; lightweight output serves as prior art for full colony run | Improve classification factors; make user override prompt more prominent; remember user overrides as project-level learning |
| Context collapse mid-phase | `/ant:resume-colony` to pick up from last checkpoint state; all JSON state files should be intact | Implement proactive clear prompts when context reaches 70% of window; ensure all state is written before suggesting clear |
| Task tool unavailable to subagent | Subagent completes work itself (inline) rather than delegating; reports blocked spawn to parent | Implement `claude -p` fallback wrapper with structured input/output via temp files |
| Delegation chain produces wrong output at depth 3 | Parent ant re-does the sub-task with full context rather than re-delegating | Reduce max depth to 2; include verbatim colony goal at every level; add quality gate checking alignment |

---

## Pitfall-to-Phase Mapping

Recommended ordering based on dependency analysis and field-test priority.

| Priority | Phase Topic | Pitfall(s) Addressed | Why This Order |
|----------|------------|---------------------|---------------|
| 1 | Bug fixes: decay math, activity log append, error phase attribution, decision log wiring | Prerequisite for all other features | These are field-tested bugs with known fixes. Ship first for system credibility and to unblock downstream features |
| 2 | Platform validation: recursive spawning feasibility test | CP-1 (Task tool limitation), CP-2 (context telephone) | Must validate platform constraints BEFORE designing features that depend on recursive delegation. 30 minutes of testing saves weeks of wrong architecture |
| 3 | File conflict prevention: task-level file ownership + reservation system | CP-4 (last-write-wins) | Blocking problem already observed in production. Required before any feature that increases parallelism. Enables safe concurrent worker execution |
| 4 | Watcher scoring calibration: meaningful rubric with variance across quality levels | CP-3 prerequisite (review fatigue) | Auto-reviewer depends on calibrated scoring. Fix the measuring instrument before adding automation that relies on it |
| 5 | Auto-reviewer ants: advisory review after build waves | CP-3 (review fatigue) | Now that scoring works, add automated review as advisory layer. Start lightweight (post-wave only, not per-task) |
| 6 | Two-tier learning: local tier improvements + empty global tier with manual promotion | CP-5 (stale global knowledge) | Start with local-tier quality improvements before adding global promotion complexity. Manual promotion keeps human in the loop |
| 7 | Adaptive complexity mode: per-phase lightweight/full selection with user confirmation | CP-6 (wrong mode selection) | Requires calibrated quality signals (from phases 4-5) to detect when wrong mode was selected |
| 8 | Learning promotion mechanism: auto-suggestion with batch user approval | CP-5 (stale global knowledge) | Only after local tier is proven across 2+ projects and learnings have accumulated to evaluate promotion candidates |
| 9 | Archivist ant: report-only file staleness analysis | CP-7 (false deletion) | Lowest priority, highest risk of user trust damage. Ship last with maximum safety rails and dry-run enforcement |

---

## Sources

### Directly Observed (Aether Field Test -- HIGH confidence)
- v5 Field Notes: 32 notes from 2026-02-04 live test on filmstrip project
- Aether v4.3 codebase: worker specs, aether-utils.sh, state files, command prompts

### Platform Documentation (HIGH confidence)
- [Claude Code Subagent Documentation](https://code.claude.com/docs/en/sub-agents) -- subagent isolation model, available tools
- [Claude Code Issue #4182: Sub-Agent Task Tool Not Exposed](https://github.com/anthropics/claude-code/issues/4182) -- nested Task spawning not supported, closed as duplicate
- [Claude Code Subagents: Common Mistakes & Best Practices](https://claudekit.cc/blog/vc-04-subagents-from-basic-to-deep-dive-i-misunderstood) -- context handoff problem, workarounds
- [Best practices for Claude Code subagents](https://www.pubnub.com/blog/best-practices-for-claude-code-sub-agents/) -- pipeline patterns, tool restriction, orchestration

### Multi-Agent Systems Research (HIGH confidence)
- [Anti-Patterns in Multi-Agent Gen AI Solutions](https://medium.com/@armankamran/anti-patterns-in-multi-agent-gen-ai-solutions-enterprise-pitfalls-and-best-practices-ea39118f3b70) -- delegation chain failures, intent mutation, step budgets
- [30 Failure Modes in Multi-Agent AI](https://medium.com/@rakesh.sheshadri44/the-dark-psychology-of-multi-agent-ai-30-failure-modes-that-can-break-your-entire-system-023bcdfffe46) -- infinite loops, context collapse, catastrophic forgetting, alignment drift
- [ROMA: Recursive Open Meta-Agent Framework](https://arxiv.org/html/2602.01848) -- structured recursive delegation with Atomizer/Planner/Executor/Aggregator pattern
- [Towards a Science of Scaling Agent Systems (Google DeepMind)](https://arxiv.org/html/2512.08296v1) -- capability saturation at 45%, 17.2x error amplification in independent agents, topology-dependent performance, sequential task degradation 39-70%
- [Why Your Multi-Agent System is Failing: The 17x Error Trap](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/) -- centralized coordination (4.4x error) vs independent agents (17.2x error)
- [Why Multi-Agent LLM Systems Fail](https://arxiv.org/html/2503.13657v1) -- systematic failure taxonomy
- [Multi-Agent Coordination Strategies (Galileo)](https://galileo.ai/blog/multi-agent-coordination-strategies) -- step budgets, DAG enforcement, escalation logic

### Context and Delegation Chain Research (HIGH confidence)
- [Chain of Agents: LLMs Collaborating on Long-Context Tasks (Google Research)](https://research.google/blog/chain-of-agents-large-language-models-collaborating-on-long-context-tasks/) -- worker-manager pattern, sequential context passing, "telephone game" mitigation
- [Multi-Agent Orchestration: Running 10+ Claude Instances in Parallel](https://dev.to/bredmond1019/multi-agent-orchestration-running-10-claude-instances-in-parallel-part-3-29da) -- practical parallel Claude agent patterns

### Memory and Learning Research (MEDIUM-HIGH confidence)
- [Agentic Memory: Unified Long-Term and Short-Term Memory Management](https://arxiv.org/html/2601.01885v1) -- learnable memory management, selective addition/deletion, 10% performance gain from utility-based deletion
- [Memory in LLM-based Multi-agent Systems](https://www.researchgate.net/publication/398392208_Memory_in_LLM-based_Multi-agent_Systems_Mechanisms_Challenges_and_Collective_Intelligence) -- shared vs local memory tradeoffs, write contention, O(N^2) communication scaling
- [The Agent's Memory Dilemma: Is Forgetting a Bug or a Feature?](https://medium.com/@tao-hpu/the-agents-memory-dilemma-is-forgetting-a-bug-or-a-feature-a7e8421793d4) -- strategic forgetting improves performance
- [Why Multi-Agent Systems Need Memory Engineering (MongoDB)](https://www.mongodb.com/company/blog/technical/why-multi-agent-systems-need-memory-engineering) -- Anthropic's own coordination failures ("50 subagents for simple queries"), memory engineering patterns

### Code Review Research (MEDIUM confidence)
- [How Accurate Is AI Code Review in 2026?](https://www.codeant.ai/blogs/ai-code-review-accuracy) -- false positive rates, developer trust gap, threshold tuning
- [Top AI Code Review Tools 2026](https://www.secondtalent.com/resources/top-ai-code-review-tools-for-development-teams/) -- alert fatigue, context-aware analysis, ~12.5% manageable false positive rate
- Stack Overflow 2025 Developer Survey -- 46% of developers distrust AI code review accuracy

### File Conflict Prevention (MEDIUM-HIGH confidence)
- [Parallel Agents Are Easy. Shipping Without Chaos Isn't.](https://dev.to/rokoss21/parallel-agents-are-easy-shipping-without-chaos-isnt-1kek) -- file conflict in multi-agent coding, prevention over resolution
- [Multi-Agent Coding: Parallel Development Guide](https://www.digitalapplied.com/blog/multi-agent-coding-parallel-development) -- workspace isolation, modular task decomposition, Cursor 2.0 git worktree approach

### Automated Cleanup Research (MEDIUM confidence)
- [Automating Dead Code Cleanup (Meta Engineering)](https://engineering.fb.com/2023/10/24/data-infrastructure/automating-dead-code-cleanup/) -- SCARF framework, static + dynamic analysis, false positive challenges
- [Stale Data Archiving Best Practices (Varonis)](https://www.varonis.com/blog/4-secrets-for-archiving-stale-data-efficiently) -- identification before disposition, audit trails
