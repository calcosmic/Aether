# Phase 29: Colony Intelligence & Quality Signals - Research

**Researched:** 2026-02-04
**Domain:** Multi-agent orchestration patterns, code quality rubric design, project complexity classification, wave-based parallel task scheduling
**Confidence:** HIGH

## Summary

Phase 29 adds six capabilities to the Aether colony system: multi-colonizer synthesis (INT-01), aggressive wave parallelism (INT-03), Phase Lead auto-approval (INT-04), watcher scoring rubric (INT-05), colony overhead adaptation (INT-07), and adaptive complexity mode (ARCH-03). All six are prompt-level changes to existing `.claude/commands/ant/` markdown files plus a new `mode` field in COLONY_STATE.json -- no new scripts, no new libraries, no infrastructure changes.

The core challenge is that all six features are design problems, not implementation problems. The watcher scoring rubric needs specific dimensions and weights that produce varied scores. The colonizer synthesis needs a merge pattern that handles disagreement. The complexity mode needs thresholds that are reasonable. The parallelism needs heuristics that are safe. Each requires a concrete design baked into the prompts.

Research confirms the standard approach for each: (1) multi-dimensional rubrics with chain-of-thought reasoning produce better LLM scoring than single-score prompts; (2) multi-agent debate with a synthesizer agent is the established pattern for merging independent reviews; (3) file-based dependency analysis is the correct heuristic for wave parallelism in this context; (4) project size proxies (file count, directory depth, language count) are sufficient for complexity classification without running LOC counters.

**Primary recommendation:** Implement as 3 plans: Plan 1 covers multi-colonizer synthesis (INT-01) + complexity mode (INT-07/ARCH-03) since both modify colonize.md. Plan 2 covers watcher scoring rubric (INT-05) since it modifies watcher-ant.md and build.md. Plan 3 covers wave parallelism (INT-03) + auto-approval (INT-04) since both modify the Phase Lead prompt in build.md.

## Standard Stack

### Core

No new libraries. All changes are to existing markdown prompt files and one JSON state file.

| File | Current Location | Purpose | Change Type |
|------|-----------------|---------|-------------|
| colonize.md | `.claude/commands/ant/colonize.md` | Codebase analysis | Spawn 3 colonizers + synthesis + complexity mode |
| build.md | `.claude/commands/ant/build.md` | Phase execution | Phase Lead parallelism + auto-approval |
| watcher-ant.md | `.aether/workers/watcher-ant.md` | Quality validation | Scoring rubric with weighted dimensions |
| COLONY_STATE.json | `.aether/data/COLONY_STATE.json` | Colony state | Add `mode` field |

### Supporting

| Tool | Purpose | Already Exists |
|------|---------|---------------|
| `aether-utils.sh validate-state` | Verify state persistence | Yes |
| `aether-utils.sh spawn-check` | Worker spawn gate | Yes |
| Task tool (subagent_type="general-purpose") | Spawn colonizer agents | Yes -- used in colonize.md Step 4 |
| Glob/Grep/Read | File enumeration for complexity detection | Yes -- available to all ants |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 3 sequential Task tool spawns for colonizers | Parallel Task tool spawns (background) | Sequential is safer -- colonizers read but don't write, so no conflict risk, but parallel would be faster. However, Claude Code's Task tool spawns sequentially from a command prompt context. The colonize command already uses Task tool delegation. Use sequential spawns -- simpler, reliable, and colonization is a one-time cost. |
| Prompt-level file dependency heuristics | Static analysis tooling | Massive overkill. The Phase Lead already parses task descriptions for file paths. Enhancing that heuristic is sufficient. |
| LOC counting for complexity | File count + directory depth | LOC requires reading every file which is slow and expensive. File count + directory depth are fast proxies available via Glob. |

## Architecture Patterns

### Recommended File Changes
```
.claude/commands/ant/
  colonize.md           # MODIFY: Steps 3-6 (3 colonizers + synthesis + complexity mode)
  build.md              # MODIFY: Step 5a (parallelism heuristics) + Step 5b (auto-approval)
.aether/
  workers/
    watcher-ant.md      # MODIFY: Scoring rubric section (replace flat scoring with rubric)
  data/
    COLONY_STATE.json   # MODIFY: Add "mode" field (LIGHTWEIGHT/STANDARD/FULL)
```

### Pattern 1: Multi-Colonizer Synthesis with Disagreement Flagging

**What:** Spawn 3 colonizer ants sequentially via Task tool. Each independently reviews the codebase. A synthesis step merges their findings, flagging disagreements.

**When to use:** During `/ant:colonize` Step 4.

**Design:**

The 3 colonizers get distinct specialization lenses (Claude's discretion -- this is the recommended approach based on multi-agent research showing that role-based heterogeneity reduces shared blind spots):

| Colonizer | Lens | Focus Areas |
|-----------|------|-------------|
| Colonizer 1: Structure | Architecture & Organization | Directory structure, entry points, module boundaries, build system, dependency graph |
| Colonizer 2: Patterns | Code Quality & Conventions | Naming conventions, design patterns, anti-patterns, code style, error handling patterns |
| Colonizer 3: Stack | Technology & Dependencies | Tech stack, framework versions, dependency health, configuration approach, deployment patterns |

Each colonizer outputs a structured report with the same schema:

```
Findings:
  - category: "<structure|patterns|stack>"
    finding: "<specific observation>"
    confidence: <HIGH|MEDIUM|LOW>
    evidence: "<file path or pattern that supports this>"
```

**Synthesis step** (done by the Queen in colonize.md, not a separate agent):
1. Collect all 3 reports
2. Group findings by topic (e.g., "architecture", "tech stack", "conventions")
3. Where 2+ colonizers agree on a finding: include as HIGH confidence
4. Where colonizers disagree: flag explicitly with both views
5. Produce unified synthesis report

**Disagreement display format:**
```
DISAGREEMENT: {topic}
  Colonizer 1 (Structure): {view}
  Colonizer 3 (Stack): {opposing view}
  Resolution: User decision needed
```

**Individual colonizer reports storage:** Save to `.aether/temp/colonizer-{1,2,3}-report.txt` for internal reference. Only the synthesis report is displayed to the user.

### Pattern 2: Watcher Scoring Rubric with Weighted Dimensions

**What:** Replace the current single "quality_score: 1-10" prompt with a multi-dimensional rubric that forces the watcher to evaluate specific aspects independently before producing an overall score.

**When to use:** In watcher-ant.md, applied during every watcher verification.

**Design -- 5 dimensions with weights:**

| Dimension | Weight | What It Measures | Score Range |
|-----------|--------|------------------|-------------|
| Correctness | 0.30 | Does the code work? Syntax valid, imports resolve, launch succeeds, tests pass | 0-10 |
| Completeness | 0.25 | Were all task requirements addressed? Are success criteria met? | 0-10 |
| Quality | 0.20 | Readability, naming, error handling, separation of concerns | 0-10 |
| Safety | 0.15 | No hardcoded secrets, no destructive operations, proper validation | 0-10 |
| Integration | 0.10 | Fits existing patterns, doesn't break conventions, backwards compatible | 0-10 |

**Overall score calculation:**
```
overall = round(correctness * 0.30 + completeness * 0.25 + quality * 0.20 + safety * 0.15 + integration * 0.10)
```

**Why these weights:** Correctness is weighted highest because code that doesn't work is worthless regardless of how clean it looks (this is already enforced by the existing "cannot exceed 6/10 if execution fails" rule). Completeness is second because incomplete delivery blocks the phase. Quality and Safety are moderate because they matter but won't block progress for medium-severity issues. Integration is lowest because minor convention violations are fixable later.

**Chain-of-thought requirement:** The watcher MUST evaluate each dimension separately BEFORE computing the overall score. This prevents the "everything is 8/10" failure mode by forcing the LLM to reason about each aspect individually. Research confirms this improves scoring discrimination by 10-15%.

**Rubric anchors (to prevent score inflation):**

| Score | Meaning | Anchor Description |
|-------|---------|-------------------|
| 1-2 | Critical failure | Code doesn't parse, missing files, fundamentally broken |
| 3-4 | Major issues | Code runs but has critical bugs, missing major requirements |
| 5-6 | Functional with issues | Code works but has notable quality problems, incomplete features |
| 7-8 | Good | Code works well, minor issues, most requirements met |
| 9-10 | Excellent | Clean, complete, well-tested, follows all conventions |

**Key insight from research:** More granular rubrics reduce LLM alignment (accuracy drops from 76% to 57% going from binary to 5-way). The 5-dimension approach with 10-point scales is at the edge of practical granularity. Anchor descriptions are essential to maintain consistency.

### Pattern 3: Wave Parallelism with File Dependency Analysis

**What:** Enhance the Phase Lead's planning prompt to more aggressively assign independent tasks to the same wave, using file-based dependency analysis as the primary grouping heuristic.

**When to use:** In build.md Step 5a, the Phase Lead prompt.

**Parallelism heuristics (ordered by priority):**

1. **Explicit task dependencies:** If task B depends_on task A, they go in different waves (existing behavior -- keep this)
2. **File overlap:** Tasks touching the same file go to the same worker in the same wave (existing CONFLICT PREVENTION RULE -- keep this)
3. **Default-parallel:** Tasks with NO explicit dependency and NO file overlap go in the SAME wave (new -- currently tasks default to sequential)

**The key change:** Currently the Phase Lead tends to create sequential waves even when tasks are independent. The new instruction reverses the default: tasks are parallel UNLESS there's a reason to serialize.

**File overlap detection prompt enhancement:**
```
For EACH task, list the files it will likely touch.
Two tasks are INDEPENDENT if they have zero file overlap.
Independent tasks go in the SAME wave with different workers.
Tasks are DEPENDENT if they share files or one needs the other's output.
Dependent tasks go in SEQUENTIAL waves or the SAME worker.

DEFAULT: Assume tasks are parallel unless you can identify a specific dependency.
```

**Conflict detection at execution time (Queen backup):**
After each wave completes, the Queen checks if any two workers in that wave modified the same file (from their activity log entries). If detected, halt and report the conflict to the user. This is already partially implemented in build.md Step 5c sub-step 2b -- it needs to be enhanced to also check post-execution, not just pre-execution.

**Wave display format (shown to user in Step 5b):**
```
Wave 1 (3 parallel workers):
  1. builder-ant: Set up auth module (tasks 1.1, 1.2) -> src/auth/
  2. builder-ant: Create API routes (tasks 1.3, 1.4) -> src/routes/
  3. scout-ant: Research OAuth providers (task 1.5)

Wave 2 (depends on Wave 1):
  4. builder-ant: Integrate auth with routes (task 1.6) -> src/auth/, src/routes/
     Needs: auth module (Wave 1.1), API routes (Wave 1.2)

Parallelism: 5/6 tasks in Wave 1 (83%)
```

### Pattern 4: Phase Lead Auto-Approval for Simple Phases

**What:** When a phase's complexity is below a threshold, the Phase Lead's plan is auto-approved without asking the user.

**When to use:** In build.md Step 5b, the plan checkpoint.

**Complexity threshold design:**
A phase is "simple" if ALL of the following are true:
- Task count <= 4
- Total worker count <= 2
- Wave count <= 2
- No tasks modify files touched by another task (zero conflict potential)
- No dependency on external systems (no API calls, no database, no deployment)

**Rationale:** These thresholds are conservative. A phase with 4 or fewer tasks, 2 or fewer workers, and 2 or fewer waves is small enough that the plan is self-evident. Asking the user to confirm a 2-task, 1-wave plan adds friction without value.

**Display when auto-approved:**
```
Phase Lead Plan (auto-approved -- simple phase: {N} tasks, {N} workers, {N} waves):

  Wave 1:
    1. builder-ant: {task description}
    2. builder-ant: {task description}

Proceeding to execution...
```

**Display when NOT auto-approved (normal flow):**
```
Phase Lead Plan:

  Wave 1 (2 parallel workers):
    ...
  Wave 2 (depends on Wave 1):
    ...

Proceed with this plan? (yes / describe changes)
```

### Pattern 5: Adaptive Colony Mode (LIGHTWEIGHT/STANDARD/FULL)

**What:** Set colony operational mode during colonization based on project complexity indicators. Stored in COLONY_STATE.json `mode` field.

**When to use:** In colonize.md, after colonizer synthesis, before displaying results.

**Mode definitions:**

| Mode | Threshold | Behavior Changes |
|------|-----------|-----------------|
| LIGHTWEIGHT | <20 source files, <3 directories deep, 1 language | Skip watcher verification (Step 5.5 in build.md), auto-approve all plans, single-colonizer (skip multi-colonizer), max 1 worker per wave |
| STANDARD | 20-200 source files, 3-6 directories deep, 1-3 languages | Normal behavior as currently implemented |
| FULL | >200 source files, >6 directories deep, >3 languages, or monorepo detected | All features enabled, additional watcher specialist modes activated, more aggressive parallelism (up to 4 workers per wave) |

**Complexity detection method (in colonize.md, before spawning colonizers):**

```bash
# Count source files (exclude node_modules, .git, etc.)
find . -type f \( -name "*.ts" -o -name "*.js" -o -name "*.py" -o -name "*.go" -o -name "*.rs" -o -name "*.java" \) \
  -not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/dist/*" -not -path "*/build/*" | wc -l

# Max directory depth
find . -type d -not -path "*/node_modules/*" -not -path "*/.git/*" | awk -F/ '{print NF}' | sort -n | tail -1

# Language count
find . -type f -not -path "*/node_modules/*" -not -path "*/.git/*" | grep -oP '\.[^./]+$' | sort -u | \
  grep -cE '\.(ts|js|py|go|rs|java|rb|php|swift|kt|c|cpp|cs)'
```

**Note:** Use Bash tool for these counts. They run once during colonization -- the cost is negligible.

**COLONY_STATE.json schema addition:**
```json
{
  "mode": "STANDARD",
  "mode_set_at": "<ISO-8601 UTC>",
  "mode_indicators": {
    "source_files": 45,
    "max_depth": 4,
    "languages": 2
  }
}
```

**How mode affects other commands:**
- build.md Step 5b: Check `COLONY_STATE.json.mode`. If LIGHTWEIGHT, auto-approve always. If STANDARD, use complexity threshold. If FULL, always require approval.
- build.md Step 5.5: Check mode. If LIGHTWEIGHT, skip watcher verification entirely.
- build.md Step 5a: Check mode. If FULL, instruct Phase Lead to parallelize more aggressively (up to 4 concurrent workers).

### Anti-Patterns to Avoid

- **Scoring without rubric:** Asking the watcher "rate this 1-10" without dimensions always produces 7-8/10. Must force per-dimension evaluation.
- **Single-colonizer with "be thorough":** Adding more instructions to one colonizer does not replace multiple independent perspectives. Different agents find different things.
- **Parallelizing everything:** Not all tasks can be parallel. The Phase Lead must respect file overlap constraints. Default-parallel does NOT mean force-parallel.
- **LOC counting for complexity:** Running `wc -l` on every file is slow and the LLM doesn't need exact numbers. File count is a sufficient proxy.
- **Complex mode switching logic:** Mode should be set once during colonization and read as a simple field. Do not add dynamic mode-switching during builds.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File dependency detection | Custom static analysis | Prompt-level heuristic (Phase Lead parses task descriptions for file paths) | Static analysis would need language-specific parsers. The Phase Lead already reads task descriptions and can identify file paths. Enhancement to existing heuristic is sufficient. |
| Code complexity scoring | Custom LOC counter / cyclomatic complexity tool | Glob + wc (file count proxy) | LLM doesn't need precise metrics. Rough file count classifies projects correctly at the LIGHTWEIGHT/STANDARD/FULL granularity. |
| Report merging | Custom diff/merge algorithm | Queen prompt-level synthesis | The Queen reads 3 text reports and synthesizes. This is exactly what LLMs are good at. No algorithm needed. |
| Score calibration | Training data / regression model | Anchored rubric with examples | LLMs respond well to rubric anchors ("score 3-4 means major issues"). This is the established LLM-as-Judge pattern. |

**Key insight:** Every feature in this phase is a prompt engineering change. The temptation to build infrastructure (scripts, tools, algorithms) should be resisted. The existing Task tool + prompt pattern handles everything.

## Common Pitfalls

### Pitfall 1: Watcher Score Inflation
**What goes wrong:** Watcher assigns 8/10 to everything because the LLM defaults to "this looks fine."
**Why it happens:** Without explicit dimensions and anchors, LLMs treat scoring as a politeness exercise.
**How to avoid:** Multi-dimensional rubric with chain-of-thought. Require the watcher to evaluate each dimension separately and show its reasoning BEFORE producing the overall score. Include anchor descriptions for each score range.
**Warning signs:** All phases receiving 7-9/10 regardless of actual quality.

### Pitfall 2: Colonizer Convergence
**What goes wrong:** All 3 colonizers produce identical reports because they all see the same codebase.
**Why it happens:** Without distinct lenses, LLMs converge on the same "comprehensive overview."
**How to avoid:** Give each colonizer a distinct specialization lens in the prompt. Structure > Patterns > Stack ensures minimal overlap while covering the full picture.
**Warning signs:** Synthesis report has no disagreements and looks like a single report repeated 3 times.

### Pitfall 3: Over-Parallelization
**What goes wrong:** Phase Lead puts everything in Wave 1, workers conflict on shared files.
**Why it happens:** "Default-parallel" instruction interpreted as "always parallel."
**How to avoid:** Keep the CONFLICT PREVENTION RULE from Phase 27. Default-parallel only applies to tasks with zero file overlap. The Queen backup check (Step 5c sub-step 2b) catches any mistakes.
**Warning signs:** Workers overwriting each other's changes. Activity log shows MERGE entries.

### Pitfall 4: Complexity Detection False Positives
**What goes wrong:** A project with 5 source files but 500 test files gets classified as FULL mode.
**Why it happens:** File count doesn't distinguish source from test/generated files.
**How to avoid:** Exclude test directories, build output, and generated files from the count. The find command excludes `node_modules/`, `.git/`, `dist/`, `build/`. Also exclude `test/`, `tests/`, `__tests__/`, `spec/`.
**Warning signs:** Small projects running in FULL mode with unnecessary overhead.

### Pitfall 5: Auto-Approval of Complex Plans
**What goes wrong:** A phase with subtle inter-task dependencies gets auto-approved and workers conflict.
**Why it happens:** The complexity threshold only checks task count, worker count, and wave count -- not conceptual complexity.
**How to avoid:** Conservative threshold (4 tasks max). Include the "no shared files" condition. When in doubt, don't auto-approve.
**Warning signs:** Auto-approved phases producing lower quality scores than user-approved phases.

### Pitfall 6: Colonize Command Timeout
**What goes wrong:** 3 sequential colonizer Task tool spawns take too long and the user gets impatient.
**Why it happens:** Each colonizer gets a full codebase review. 3x the work of current single colonizer.
**How to avoid:** Each colonizer has a focused lens (not "review everything"). Structure colonizer does directory analysis, not file-by-file review. Patterns colonizer samples representative files, not all files. Stack colonizer checks package manifests and configs, not source.
**Warning signs:** Colonization taking >5 minutes on medium projects.

## Code Examples

### Example 1: Multi-Colonizer Spawn Pattern in colonize.md

```markdown
### Step 4: Spawn Three Colonizer Ants

Spawn 3 colonizer ants sequentially. Each gets a distinct specialization lens.

**Colonizer 1 (Structure):**
Use the Task tool with subagent_type="general-purpose":

{same prompt header as current Step 4, but with this mission:}

--- YOUR MISSION ---
You are Colonizer 1 of 3 (Structure Lens).

Focus ONLY on architecture and organization:
1. Directory structure and module boundaries
2. Main entry points and how they connect
3. Build system and scripts
4. Dependency graph between modules
5. File organization conventions

Do NOT analyze code quality or tech stack details -- other colonizers handle those.

Output your findings as:
  COLONIZER 1 (STRUCTURE) REPORT
  {structured findings}

**Colonizer 2 (Patterns):**
{same prompt header, different mission:}

--- YOUR MISSION ---
You are Colonizer 2 of 3 (Patterns Lens).

Focus ONLY on code quality and conventions:
1. Naming conventions (files, variables, functions, classes)
2. Design patterns in use (and anti-patterns)
3. Error handling approach
4. Code style and formatting
5. Documentation patterns

Sample 5-10 representative files across the codebase.
Do NOT map the full directory structure -- Colonizer 1 handles that.

**Colonizer 3 (Stack):**
{same prompt header, different mission:}

--- YOUR MISSION ---
You are Colonizer 3 of 3 (Stack Lens).

Focus ONLY on technology and dependencies:
1. Languages and frameworks in use (with versions)
2. Dependency health (outdated? vulnerable?)
3. Configuration approach (env vars, config files, hardcoded)
4. Build/deploy pipeline
5. External service integrations

Check package manifests (package.json, requirements.txt, Cargo.toml, etc.)
Do NOT review individual source files -- other colonizers handle that.
```

### Example 2: Watcher Scoring Rubric Addition to watcher-ant.md

```markdown
## Scoring Rubric (Mandatory)

Before assigning quality_score, you MUST evaluate each dimension independently.
Show your reasoning for each dimension. Then compute the weighted overall score.

| Dimension | Weight | Evaluate |
|-----------|--------|----------|
| Correctness (0.30) | Does code run? Syntax valid? Imports resolve? Tests pass? |
| Completeness (0.25) | All task requirements addressed? Success criteria met? |
| Quality (0.20) | Readable? Good naming? Error handling? Single responsibility? |
| Safety (0.15) | No secrets in code? No destructive ops? Input validated? |
| Integration (0.10) | Fits existing patterns? Conventions followed? Backwards compatible? |

Score each dimension 0-10 using these anchors:
- 1-2: Critical failure (doesn't parse, missing files, fundamentally broken)
- 3-4: Major issues (runs but has critical bugs, missing major requirements)
- 5-6: Functional with issues (works but notable quality problems)
- 7-8: Good (works well, minor issues, most requirements met)
- 9-10: Excellent (clean, complete, well-tested, follows conventions)

Output format:
  Scoring Rubric:
    Correctness:  {score}/10 — {1-line reason}
    Completeness: {score}/10 — {1-line reason}
    Quality:      {score}/10 — {1-line reason}
    Safety:       {score}/10 — {1-line reason}
    Integration:  {score}/10 — {1-line reason}

  Overall: {weighted_score}/10
  = round(C*0.30 + Co*0.25 + Q*0.20 + S*0.15 + I*0.10)

IMPORTANT: Evaluate each dimension BEFORE computing the overall score.
Do NOT decide the overall score first and reverse-engineer dimensions.
```

### Example 3: COLONY_STATE.json Mode Field

```json
{
  "goal": "Build a real-time chat app",
  "state": "READY",
  "current_phase": 0,
  "mode": "STANDARD",
  "mode_set_at": "2026-02-04T12:00:00Z",
  "mode_indicators": {
    "source_files": 45,
    "max_depth": 4,
    "languages": 2
  },
  "session_id": "sess_1234",
  "initialized_at": "2026-02-04T12:00:00Z",
  "workers": { ... },
  "spawn_outcomes": { ... }
}
```

### Example 4: Phase Lead Auto-Approval Check in build.md

```markdown
### Step 5b: Plan Checkpoint

After the Phase Lead returns:

**Auto-Approval Check:**
Read COLONY_STATE.json. Check the `mode` field.

If mode is "LIGHTWEIGHT": auto-approve the plan. Skip user confirmation.
Display: "Plan auto-approved (LIGHTWEIGHT mode). Proceeding to execution..."

If mode is "STANDARD" or "FULL":
  Count from the Phase Lead's plan:
  - task_count: total tasks
  - worker_count: total workers assigned
  - wave_count: total waves
  - shared_files: whether any two workers in the same wave touch the same file

  If task_count <= 4 AND worker_count <= 2 AND wave_count <= 2 AND shared_files == false:
    Auto-approve. Display: "Plan auto-approved (simple phase: {tasks} tasks, {workers} workers, {waves} waves)."
    Proceed to Step 5c.

  Otherwise:
    Display the plan. Ask: "Proceed with this plan? (yes / describe changes)"
    {existing plan checkpoint logic}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single colonizer, single pass | Multi-agent independent review with synthesis | 2024-2025 (multi-agent debate papers) | Reduces shared blind spots, catches more findings |
| Single quality score "rate 1-10" | Multi-dimensional rubric with chain-of-thought | 2025 (LLM-as-Judge research) | Improves score discrimination by 10-15%, reduces inflation |
| Sequential task execution by default | Default-parallel with dependency analysis | Standard in build systems (Make, Bazel) | Reduces build time proportional to parallelizable work |
| One-size-fits-all colony overhead | Adaptive complexity modes | New for Aether (inspired by COCOMO model categorization) | Small projects don't pay overhead tax of full colony features |

**Deprecated/outdated:**
- Single-score prompts for LLM quality assessment: Research shows multi-dimensional rubrics consistently outperform
- Flat "is this good?" evaluation: Chain-of-thought before scoring is the established best practice

## Open Questions

Things that couldn't be fully resolved:

1. **Colonizer parallelism via Task tool**
   - What we know: Claude Code Task tool supports up to 10 concurrent tasks
   - What's unclear: Whether the colonize.md command prompt (which runs as the main agent, not as a Task) can spawn 3 Task tool calls in parallel, or whether they must be sequential. Prior Aether commands (build.md Step 5c) spawn workers sequentially within waves.
   - Recommendation: Keep sequential spawning for now. If colonization is too slow, revisit parallel spawning in a future phase. The colonization is a one-time cost per project.

2. **Post-execution file conflict detection**
   - What we know: The Queen already does pre-execution file overlap checking (Step 5c sub-step 2b). Post-execution detection would check activity logs after each wave.
   - What's unclear: Whether activity log entries reliably capture all file paths modified by workers. Workers log CREATED/MODIFIED entries but may not catch every file.
   - Recommendation: Add post-wave conflict check as best-effort. Parse activity log entries for file paths. If two workers in the same wave show MODIFIED entries for the same file, halt. Accept that coverage is not 100%.

3. **Scoring rubric calibration**
   - What we know: The 5-dimension rubric with anchors should produce varied scores based on research
   - What's unclear: Whether the specific weights (0.30, 0.25, 0.20, 0.15, 0.10) are optimal for this context
   - Recommendation: Ship with these weights and observe. The weights are easy to adjust in future phases if scoring patterns reveal imbalances.

## Sources

### Primary (HIGH confidence)
- Existing Aether codebase: colonize.md, build.md, continue.md, watcher-ant.md, COLONY_STATE.json -- read directly
- Phase 27 and 28 research and plans -- established patterns for this milestone
- Field notes v5 -- direct user observations that motivated these requirements

### Secondary (MEDIUM confidence)
- [Qodo: Code Quality Metrics 2026](https://www.qodo.ai/blog/code-quality-metrics-2026/) -- code review quality dimensions
- [Label Your Data: LLM as a Judge](https://labelyourdata.com/articles/llm-as-a-judge) -- chain-of-thought scoring, rubric design
- [arXiv: Rubric Is All You Need](https://arxiv.org/html/2503.23989v1) -- question-specific rubric design for code evaluation
- [arXiv: MAR Multi-Agent Reflexion](https://arxiv.org/html/2512.20845) -- multi-agent debate and synthesis patterns
- [ACM: LLM-Based Multi-Agent Systems for SE](https://dl.acm.org/doi/10.1145/3712003) -- conflict identification in multi-agent collaboration
- [COCOMO Model](https://en.wikipedia.org/wiki/COCOMO) -- project size classification (Organic/Semi-Detached/Embedded)
- [Claude Code Task Tool](https://dev.to/bhaidar/the-task-tool-claude-codes-agent-orchestration-system-4bf2) -- parallel subagent capabilities and limitations
- [Claude Code Subagents](https://platform.claude.com/docs/en/agent-sdk/subagents) -- subagent architecture documentation

### Tertiary (LOW confidence)
- [Monte Carlo: LLM-As-Judge Best Practices](https://www.montecarlodata.com/blog-llm-as-judge/) -- integer scoring scale recommendation
- [Green Report: Multi-Step Grading Rubrics](https://www.thegreenreport.blog/articles/multi-step-grading-rubrics-with-llms-for-answer-evaluation/multi-step-grading-rubrics-with-llms-for-answer-evaluation.html) -- threshold-based quality gates
- [arXiv: Rubric-Conditioned LLM Grading](https://arxiv.org/html/2601.08843) -- granularity vs alignment tradeoff finding (accuracy drops 76% -> 57%)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all changes are to existing files, no new dependencies, patterns established in Phases 27-28
- Architecture: HIGH -- multi-colonizer synthesis, scoring rubric, wave parallelism, and complexity modes are well-defined prompt engineering patterns with clear design
- Pitfalls: HIGH -- directly observed issues from field notes (Note 10: merge conflicts, Note 13: same-file overwrites) and established LLM scoring research (inflation, convergence)

**Research date:** 2026-02-04
**Valid until:** 2026-03-04 (30 days -- stable domain, all changes are to internal prompts)
