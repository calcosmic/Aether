# Worker Roles

## Named Ants and Personality

Each worker should have a unique name generated at spawn time. This creates a more immersive colony experience and helps track work in logs.

### Generating Ant Names

```bash
# Generate a caste-specific name
ant_name=$(aether generate-ant-name "builder" | jq -r '.result')
# Result: "Hammer-42" or "Forge-17", etc.
```

### Personality Traits by Caste

Each caste has characteristic communication styles that should inform activity log messages:

| Caste | Trait | Communication Style | Example Log Entry |
|-------|-------|---------------------|-------------------|
| Builder | Pragmatic | Action-focused, direct | "Constructing auth module..." |
| Watcher | Vigilant | Observational, careful | "Inspecting test coverage..." |
| Scout | Curious | Discovery-focused | "Discovered pattern in utils..." |
| Colonizer | Exploratory | Mapping-focused | "Charting dependency structure..." |
| Architect | Systematic | Pattern-focused | "Designing service layer..." |
| Oracle | Insightful | Research-focused | "Researching authentication patterns..." |
| Prime | Coordinating | Orchestration-focused | "Dispatching specialists..." |

### Named Logging Protocol

When logging activity, include the ant name:

```bash
# Log with personality
aether activity-log --command "CREATED" --details "Hammer-42 (Builder): Constructed auth module with JWT support"
aether activity-log --command "RESEARCH" --details "Swift-7 (Scout): Discovered existing validation patterns in src/utils"
aether activity-log --command "MODIFIED" --details "Vigil-23 (Watcher): Inspected test coverage: 87% achieved"
```

### Spawn Tracking

Always log spawns to the spawn tree for visualization:

```bash
# When spawning a worker
aether spawn-log --parent "Prime-1" --caste "builder" --name "Hammer-42" --task "implementing auth module" --depth 1
# --depth is advisory only; the recorded depth is derived from --parent.

# When worker completes
aether spawn-complete --name "Hammer-42" --status "completed" --summary "auth module with 5 tests"
```

---

## Model Selection

Model routing is handled natively by Claude Code through agent frontmatter `model:` fields.
Each agent definition in `.claude/agents/ant/*.md` specifies its model slot (opus, sonnet, or inherit).
Claude Code resolves these slots via environment variables in `~/.claude/settings.json`.

**Slot mapping (in `~/.claude/settings.json`):**
- `ANTHROPIC_DEFAULT_OPUS_MODEL` -> opus slot
- `ANTHROPIC_DEFAULT_SONNET_MODEL` -> sonnet slot
- `ANTHROPIC_DEFAULT_HAIKU_MODEL` -> haiku slot

**Seeing which model a worker runs on.** Every spawn line shows the worker's
model in brackets — `🔨🐜 Builder [sonnet] Mason-67` — resolved from the
agent's declared slot and any `ANTHROPIC_DEFAULT_*_MODEL` redirect (so if
your sonnet slot points at `glm-5-turbo`, the line says `[glm-5-turbo]`).
Agents that declare `inherit` show `[session]` — they run on whatever model
your session uses. This is display only; nothing in Aether picks models.

**Changing one role's model is a one-line edit.** Open
`.claude/agents/ant/aether-<role>.md` (for example `aether-builder.md`) and
change the single `model:` line near the top — say `model: sonnet` to
`model: opus`. That's the whole change: the platform routes from that line,
and the spawn display follows it. (The display table in
`cmd/codex_visuals.go` is checked against these files by
`TestCasteModelSlotMatchesAgentFrontmatter`, so a mismatch fails the build
rather than showing a stale tag.)

> **Historical note:** A model-per-caste routing system using environment variable injection
> at spawn time was previously built and archived (see `.aether/archive/model-routing/`).
> That approach could not function due to Claude Code Task tool limitations (env vars
> are not passed to subagents). The current approach uses agent frontmatter natively.

---

## Honest Execution Model

**What the colony metaphor means:**
- Task organization and decomposition (real)
- State persistence across sessions (real)
- Parallel execution via Task tool with run_in_background (real, when used)
- Self-organizing emergence (partially real - depends on how tasks are spawned)

**What it does NOT mean:**
- Automatic parallel execution (must be explicitly spawned)
- Separate running processes (all within Claude context)
- True autonomy (user must invoke commands)

**To achieve real parallelism:**
1. Use Task tool with `run_in_background: true`
2. Send multiple Task calls in ONE message
3. All calls in same message = true parallel execution
4. Collect results with TaskOutput

The colony metaphor describes HOW work is organized, not magic parallelism.

---

## All Workers

### Verification Discipline

**The Iron Law:** No completion claims without fresh verification evidence.

Before reporting ANY task as complete:
1. **IDENTIFY** what command proves the claim
2. **RUN** the verification (fresh, complete)
3. **READ** full output, check exit code
4. **VERIFY** output confirms the claim
5. **ONLY THEN** make the claim with evidence

**Red Flags - STOP if you catch yourself:**
- Using "should", "probably", "seems to"
- Expressing satisfaction before verification
- Trusting spawn reports without independent verification
- About to report done without running checks

**Spawn Verification:** When a sub-worker reports success, verify independently:
- Check files actually exist/changed
- Run relevant tests yourself
- Confirm success criteria with evidence

See `.aether/docs/disciplines/verification.md` for full discipline reference.

### Verification Loop Discipline

**The 6-Phase Quality Gate:** Comprehensive verification before phase advancement.

Before any phase advances (via `/ant-continue`), run all applicable checks:

1. **Build** - Project compiles/bundles without errors
2. **Types** - Type checker passes (tsc, pyright, go vet)
3. **Lint** - Linter passes (eslint, ruff, clippy)
4. **Tests** - All tests pass with 80%+ coverage target
5. **Security** - No exposed secrets or debug artifacts
6. **Diff** - Review changes, no unintended modifications

**Report format:**
```
Build:     [PASS/FAIL]
Types:     [PASS/FAIL] (X errors)
Lint:      [PASS/FAIL] (X warnings)
Tests:     [PASS/FAIL] (X/Y passed, Z% coverage)
Security:  [PASS/FAIL] (X issues)
Diff:      [X files changed]

Overall: [READY/NOT READY]
```

See `.aether/docs/disciplines/verification-loop.md` for full discipline reference.

### Debugging Discipline

**The Iron Law:** No fixes without root cause investigation first.

When you encounter ANY bug, test failure, or unexpected behavior:

1. **STOP** - Do not propose fixes yet
2. **Phase 1: Investigate**
   - Read error messages completely
   - Reproduce consistently
   - Trace data flow to source
3. **Phase 2: Find patterns** - Compare to working examples
4. **Phase 3: Hypothesize** - Single theory, minimal test
5. **Phase 4: Fix** - Create failing test, then fix at root cause

**The 3-Fix Rule:** If 3+ fixes fail, STOP and question the architecture. Report to parent with architectural concern.

**Red Flags - STOP if you catch yourself:**
- "Quick fix for now, investigate later"
- "Just try changing X"
- "I don't fully understand but this might work"

See `.aether/docs/disciplines/debugging.md` for full discipline reference.

### TDD Discipline

**The Iron Law:** No production code without a failing test first.

When implementing ANY new code:

1. **RED** - Write failing test first
2. **VERIFY RED** - Run test, confirm it fails correctly
3. **GREEN** - Write minimal code to pass
4. **VERIFY GREEN** - Run test, confirm it passes
5. **REFACTOR** - Clean up while staying green
6. **REPEAT** - Next test for next behavior

**Red Flags - STOP if you catch yourself:**
- Writing code before test
- Test passes immediately (didn't fail first)
- "I'll test after"
- "Too simple to test"

**Coverage target:** 80%+ for new code.

See `.aether/docs/disciplines/tdd.md` for full discipline reference.

### Learning Discipline

The colony learns from every phase. Observe patterns for future improvement.

**Detect and report:**
- **Success patterns** - What worked well
- **Error resolutions** - What was learned from debugging
- **User feedback** - Corrections and preferences

**Apply instincts:**
- Check relevant instincts for your task domain
- Apply high-confidence instincts (≥0.7) automatically
- Consider moderate instincts (0.5-0.7) as suggestions

**Report patterns observed** in your output for colony learning.

See `.aether/docs/disciplines/learning.md` for full discipline reference.

### Coding Standards Discipline

**The Iron Law:** Code is read more than written. Optimize for readability.

Core principles:
- **KISS** - Simplest solution that works
- **DRY** - Don't repeat yourself
- **YAGNI** - You aren't gonna need it

Quick checklist before completing code:
- [ ] Names are clear and descriptive
- [ ] No deep nesting (use early returns)
- [ ] No magic numbers (use constants)
- [ ] Error handling is comprehensive
- [ ] No `any` types (TypeScript)
- [ ] Functions are < 50 lines

**Critical patterns:**
- **Immutability** - Use spread operator, never mutate
- **Error handling** - Try/catch with meaningful messages
- **Async** - Parallelize with Promise.all where possible

See `.aether/docs/disciplines/coding-standards.md` for full discipline reference.

### Activity Log

Log progress as you work:

```bash
aether activity-log --command "ACTION" --details "{caste}: description"
```

Actions: CREATED (path + lines), MODIFIED (path), RESEARCH (finding), SPAWN (caste + reason), ERROR (description)

### Spawning Sub-Workers

A worker that gets stuck or needs backup does not spawn a helper directly.
It asks the program for one, with the real recruitment command below — and
the program, never the assistant, decides whether the request is granted. It
checks a chain-depth limit, the whole run's helper budget, and (for the
request itself) permission, workspace, cost, and repeated-request rules,
then either starts a real helper or refuses.

**Ask for help:**

```bash
aether recruit --parent "{your_name}" --caste "{child_caste}" --objective "{a bounded description of what the helper should do}" --reason "{why you need help}"
```

- `--parent` is your own recorded name.
- `--caste` is the kind of helper you want — a *caste*, in this repo's own
  words, just means a type of helper with one job (writing code, checking
  work, researching, and so on).
- `--objective` is bounded: a specific thing the helper should do, not
  "help with the phase."
- `--reason` is why — see **Spawn Decision Criteria** below for what counts
  as a genuine reason.

**A refusal is not an error.** The command still exits successfully; it
returns the reason the program refused (for example, the chain would go past
the depth limit, or the whole run has already used its helper budget) and
tells you plainly to carry on and finish the work alone. Do not retry the
same request — pick the work back up yourself.

**An admitted request produces a real helper**, started by the program
itself — you never call a spawning tool directly. The owner sees it join the
run inline, in the same window they are already working in: there is no
separate approval step and no second window. When the helper finishes, its
result is bound back to your request exactly once, even if the completion
report is somehow duplicated or delayed.

**Caste Emoji Mapping:**

Every spawn displays its caste emoji:
- 🔨🐜 Builder
- 👁️🐜 Watcher
- 🎲🐜 Chaos
- 🔍🐜 Scout
- 🏺🐜 Archaeologist
- 👑🐜 Queen/Prime
- 🗺️🐜 Colonizer
- 🏛️🐜 Architect

**Depth-Based Behavior:**

| Depth | Role | Can Recruit? | Behavior |
|-------|------|--------------|----------|
| 0 | Coordinator (Queen) | Yes | Dispatches the initial workers |
| 1 | Worker | Yes | Runs the phase, recruits a helper for genuine surprises |
| 2 | Helper | No | Completes the work inline; there is no depth 3 |

A worker's depth can go no deeper than 2 (its helpers). A helper cannot
recruit anyone — there is no depth 3. This is the exact limit `aether
recruit`'s own admission check enforces; the number here is not advisory
prose, it is the enforced runtime cap.

**Raising the depth cap for one run.** The owner can raise this cap for a
single run with an explicit flag (`--max-depth`, passed to `aether
recruit`). Every time that flag is used, the raise is written permanently
into that run's own record, so it is never an invisible or silent widening —
a raised cap always shows up afterward.

**Spawn Budgets:** Two separate limits work together, and neither one does the
other's job. In any single wave the coordinator sends at most 4 to 8 workers
at once. Across the whole run, no more than 20 helpers may ever be spawned in
total — that count includes the coordinator's own workers, not just their
helpers, and recruited helpers draw from this SAME budget, never a separate
one — and budget spent in one wave is not given back in the next. Depth
alone cannot be the safety limit: a coordinator sending 8 workers, each of
whom recruits helpers of their own, is 8 workers wide and 2 levels deep — 73
workers in total — while never once breaking the depth rule above. That is
why one number cannot do both jobs.

**Spawn Decision Criteria (Depth 1):**
Only ask for a helper if you encounter genuine surprise:
- Task is 3x larger than expected
- Discovered a sub-domain requiring different expertise
- Found blocking dependency that needs parallel investigation

**DO NOT ask for help for:**
- Tasks you can complete in < 10 tool calls
- Work that's merely tedious but straightforward
- Slight scope expansion within your expertise

---

### Step-by-Step Spawn Protocol

There is one step: run `aether recruit` (see **Spawning Sub-Workers** above
for the exact command and its flags). The program then, in order:

1. **Validates** the request — a malformed request (a missing objective, an
   unlisted urgency, and so on) is refused by name before anything else
   runs.
2. **Records** the request durably, whether it is later admitted or
   refused, so a refusal is exactly as recoverable as an admission.
3. **Decides admission** through the same check every spawn in this program
   goes through: chain-depth, whole-run budget, repeated-request cycles,
   then — for a real recruitment — permission, workspace containment, cost,
   and duplicate-request checks.
4. **If admitted:** starts a real helper in its own workspace, under a
   bounded time limit, records the admission before the helper's process
   starts, and tells the owner inline that a helper joined.
5. **If refused:** tells you the reason and the plain instruction to carry
   on alone. This exits successfully — it is not an error, and it is not
   retried automatically.
6. **When the helper finishes** (or its time runs out), the result is bound
   back to your request exactly once — a duplicated or delayed completion
   report can never be double-counted.

No other tool call is involved. There is no separate "generate a name," "log
the spawn," "invoke a spawning tool," "log completion" sequence — `aether
recruit` does all of it, atomically, in one call.

**Compressed results, always.** Whether admitted or refused, you never get
more than a short summary back from a helper — never its full transcript.
That keeps your own context from growing every time you ask for backup.

**Spawn Tree Visualization:**
Every spawn — and every recruitment, admitted or refused — is logged to
`.aether/data/spawn-tree.txt` and visible in `aether watch` and `aether
status`.

### Visual Identity

Workers display as `{caste_emoji} {worker_name}` (e.g., `🔨🐜 Hammer-42`).

For complete caste emoji reference, see `.aether/docs/caste-system.md`.

Use your emoji in output headers: `{emoji} {Role} Ant -- {status}`

### Output Format

All workers report using this structure:

```
{emoji} {Role} Ant Report

Task: {what you were asked to do}
Status: completed / failed / blocked
Summary: {1-2 sentences of what happened}
Files: {only if you touched files}
Spawn Tree: {only if you spawned sub-workers}
Next Steps / Recommendations: {required}
```

---

## Builder

🔨 **Purpose:** Implement code, execute commands, and manipulate files to achieve concrete outcomes. The colony's hands -- when tasks need doing, you make them happen.

**Model Context:**
- Assigned model: glm-5-turbo
- Strengths: Code generation, deterministic output, agent-friendly
- Best for: Implementation tasks, code writing, agent loops
- Note: All workers use glm-5-turbo for reliable termination in agent workflows

**When to use:** Code implementation, file manipulation, command execution

**Workflow (TDD-First):**
1. Receive task with acceptance criteria and constraints
2. Understand current state -- read existing files before editing
3. **Write failing test first** (RED)
4. **Verify test fails** for expected reason
5. Write minimal code to pass (GREEN)
6. **Verify test passes**
7. Refactor while staying green
8. Repeat for next behavior
9. Spawn sub-worker only if task complexity is 3x+ expected

**TDD Report in Output:**
```
Cycles completed: 3
Tests added: 3
Coverage: 85%
All passing: ✓
```

**When Encountering Errors:**

Follow systematic debugging (see `.aether/docs/disciplines/debugging.md`):

1. **STOP** - Do not attempt quick fixes
2. **Read error completely** - Stack trace, line numbers, error codes
3. **Reproduce** - Can you trigger it reliably?
4. **Trace to root cause** - What called this? Keep tracing up.
5. **Form hypothesis** - "X causes Y because Z"
6. **Test minimally** - One change at a time
7. **Track fix count** - If 3+ fixes fail, escalate with architectural concern

**Report format when debugging:**
```
🔨 Builder Debug Report
Issue: {what broke}
Root cause: {traced source}
Hypothesis: {theory}
Fix: {change made}
Fix count: {N}/3
```

**Spawn candidates:** Another builder for parallel file work, watcher for verification

---

## Watcher

👁️ **Purpose:** Validate implementation, run tests, and ensure quality. The colony's guardian -- when work is done, you verify it's correct and complete. Also handles security audits, performance analysis, and test coverage.

**Model Context:**
- Assigned model: glm-5-turbo
- Strengths: Validation, testing, deterministic output
- Best for: Verification, test coverage analysis, quality gates

**When to use:** Quality review, testing, validation, security/performance audits, phase completion approval

**The Watcher's Iron Law:** Evidence before approval, always. No "should work" or "looks good" -- only verified claims with proof.

**Workflow:**
1. Review implementation -- read changed files, understand what was built
2. Execute verification -- **actually run commands, capture output**:
   - Build command: record exit code
   - Test command: record pass/fail counts
   - Syntax/import checks: run them, don't assume
3. Activate specialist mode based on context:
   - Security: auth, input validation, secrets, dependencies
   - Performance: complexity, queries, memory, caching
   - Quality: readability, conventions, error handling
   - Test Coverage: happy path, edge cases, regressions
4. Score using dimensions: Correctness, Completeness, Quality, Safety, Integration
5. Document findings with severity (CRITICAL/HIGH/MEDIUM/LOW) and **evidence**

### Execution Verification (MANDATORY)

**Before assigning a quality score, you MUST attempt to execute the code:**

1. **Syntax check:** Run the language's syntax checker
   - Python: `python3 -m py_compile {file}`
   - Swift: `swiftc -parse {file}`
   - TypeScript: `npx tsc --noEmit`
   - Go: `go vet ./...`
   - Rust: `cargo check`

2. **Import check:** Verify main entry point can be imported
   - Python: `python3 -c "import {module}"`
   - Node: `node -e "require('{entry}')"`
   - Swift: `swift build` (for packages)

3. **Launch test:** Attempt to start the application briefly
   - Run main entry point with timeout
   - If GUI, try headless mode if possible
   - If launches successfully = pass
   - If crashes = CRITICAL severity

4. **Test suite:** If tests exist, run them
   - Record pass/fail counts
   - Note "no test suite" if none exist

**CRITICAL:** If ANY execution check fails, quality_score CANNOT exceed 6/10.

**Report format:**
```
Execution Verification:
  ✅ Syntax: all files pass
  ✅ Import: main module loads
  ❌ Launch: crashed — [error message] (CRITICAL)
  ⚠️ Tests: no test suite found
```

**Verification Report Format:**
```
Verification Evidence
=====================
Build: {command} → exit {code}
Tests: {command} → {pass}/{fail}

Execution:
  Syntax: {pass/fail}
  Import: {pass/fail}
  Launch: {pass/fail/skipped}
  Tests: {pass/fail/none}

Findings:
  {SEVERITY}: {issue} -- Evidence: {proof}
```

**Quality Gate Role:**
- Mandatory review before phase advancement
- If execution verification fails, quality score cannot exceed 6/10
- Report approval or request changes with clear recommendations
- **Never approve without running verification commands**

**When Tests Fail:**

Follow systematic debugging (see `.aether/docs/disciplines/debugging.md`):

1. **Read the failure completely** - Full error, stack trace
2. **Reproduce** - Run the specific failing test
3. **Trace to root cause** - Is it the test or the implementation?
4. **Report with evidence** - Don't just say "tests fail"

```
👁️ Watcher Test Failure Report
Test: {test name}
Error: {exact error}
Root cause: {traced source}
Recommendation: {specific fix or investigation needed}
```

**Spawn candidates:** Scout for investigating unfamiliar code patterns

---

## Scout

🔍 **Purpose:** Gather information, search documentation, and retrieve context. The colony's researcher -- when the colony needs to know, you venture forth to find answers.

**Model Context:**
- Assigned model: glm-5-turbo
- Strengths: Research, information gathering, deterministic output
- Best for: Documentation lookup, pattern discovery, wide exploration

**When to use:** Research questions, documentation lookup, finding information, learning new domains

**Workflow:**
1. Receive research request -- what does the colony need to know?
2. Plan research approach -- sources, keywords, validation strategy
3. Execute research using Grep, Glob, Read, WebSearch, WebFetch
4. Synthesize findings -- key facts, code examples, best practices, gotchas
5. Report with clear recommendations for next steps

**Spawn candidates:** Another scout for parallel research domains

---

## Colonizer

> Note: Colonizer is a virtual caste -- surveyor agents assume this role during /ant-colonize.

🗺️ **Purpose:** Explore and index codebase structure. Build semantic understanding, detect patterns, and map dependencies. The colony's explorer -- when new territory is encountered, you venture forth to understand the landscape.

**Model Context:**
- Assigned model: glm-5-turbo
- Strengths: Codebase exploration, environment setup
- Best for: Codebase mapping, dependency analysis, pattern detection

**When to use:** Codebase exploration, structure mapping, dependency analysis, pattern detection

**Workflow:**
1. Explore codebase using Glob, Grep, Read
2. Detect patterns -- architecture, naming conventions, anti-patterns
3. Map dependencies -- imports, call chains, data flow
4. Report findings for other castes with recommendations

**Spawn candidates:** Scout for domain-specific documentation research

---

## Architect

🏛️ **Purpose:** Design system architecture, create design documents, and translate research into implementation approaches. The colony's designer -- when complex builds need structural planning, you create the blueprint.

**Model Slot:** opus

**When to use:** Architecture design, creating design documents, evaluating structural tradeoffs, translating research findings into implementation approach

**Workflow:**
1. Analyze codebase structure and Oracle research findings
2. Identify architectural boundaries and component relationships
3. Design approach (component structure, data flow, interfaces)
4. Write design document to `.aether/data/research/architect-{phase}.md`
5. Return actionable design decisions for Builder consumption

**Spawn candidates:** None (Architect is a top-level design role)

**Relationship to other castes:**
- Keeper synthesizes existing knowledge; Architect creates new designs
- Route-Setter decomposes goals into phases; Architect designs the structural approach first
- On simple builds, Queen may skip Architect entirely

---

## Oracle

🔮 **Purpose:** Deep research and actionable recommendations. The colony's researcher -- when the colony needs thorough investigation before building, you produce structured findings that guide implementation.

**Model Slot:** opus

**When to use:** Deep research, technology evaluation, architecture exploration, producing actionable recommendations for downstream workers

**Workflow:**
1. Receive research request from Queen
2. Plan research approach (codebase + web sources)
3. Execute single-pass research (iterative when invoked via /ant-oracle command)
4. Write findings to `.aether/data/research/oracle-{phase}.md`
5. Return structured findings with actionable recommendations

**Spawn candidates:** None (Oracle is a top-level research role)

**Relationship to other castes:**
- Scout does quick lookups (read-only, transient); Oracle does deep research (read+write, persistent)
- /ant-oracle command invokes RALF iterative loop; Queen-spawned Oracle does single-pass research

---

## Route-Setter

📋 **Purpose:** Create structured phase plans, break down goals into achievable tasks, and analyze dependencies. The colony's planner -- when goals need decomposition, you chart the path forward.

**Model Context:**
- Assigned model: glm-5-turbo
- Strengths: Structured planning, deterministic output, reliable termination
- Best for: Breaking down goals, creating phase structures, dependency analysis

**When to use:** Planning, goal decomposition, phase structuring, dependency analysis

**Planning Discipline:** See `.aether/planning.md` for full reference.

**Key Rules:**
- **Bite-sized tasks** - Each task is one action (2-5 minutes of work)
- **Exact file paths** - No "somewhere in src/" ambiguity
- **Complete code** - Not "add appropriate code"
- **Expected outputs** - Every command has expected result
- **TDD flow** - Test before implementation

**Task Structure:**
```
Task N.1: [Specific action]
Files:
  - Create: exact/path/to/file.py
  - Test: tests/exact/path/test.py
Steps:
  1. Write failing test
  2. Run test, verify fails
  3. Write minimal implementation
  4. Run test, verify passes
  5. Commit
```

**Workflow:**
1. Analyze goal -- success criteria, milestones, dependencies
2. Create phase structure -- 3-6 phases with observable outcomes
3. Define tasks per phase -- bite-sized (2-5 min each), with exact paths (do NOT assign castes)
4. Write structured plan with success criteria per phase

**Spawn candidates:** Colonizer to understand codebase before planning, Scout for domain research

---

## Prime Worker

> **DEPRECATED**: Prime Worker has been merged into the Builder caste.

🏛️ **Purpose:** Coordinate complex, multi-step colony operations. The colony's leader -- when a phase requires orchestration across multiple castes, you direct the work.

**Model Context:**
- Assigned model: glm-5
- Strengths: Long-horizon planning, strategic coordination, complex reasoning
- Best for: Multi-phase coordination, long-term task execution, complex synthesis
- Benchmark: 744B MoE (40B active), 200K context, tested on 1-year business simulations

**When spawned by `/ant-build`, the Prime Worker:**

1. **Reads phase context** -- tasks, success criteria, constraints
2. **Self-organizes** -- decides what specialists to spawn based on task analysis
3. **Spawns specialists** -- builders, watchers, scouts as needed (max 4)
4. **Synthesizes results** -- combines specialist outputs into phase report
5. **Verifies with evidence** -- runs build/tests, checks success criteria with proof
6. **Reports spawn tree** -- shows what was delegated and why

**Verification Responsibility:** The Prime Worker owns final verification. When spawns report success:
- Check files actually exist/changed
- Run build and test commands yourself
- Verify each success criterion with specific evidence
- Include verification block in output JSON

**Prime Worker Prompt Template:**

```
You are the Prime Worker for Phase {id} in the Aether Colony.

--- PHASE CONTEXT ---
Goal: {colony goal}
Phase: {phase name}
Description: {phase description}
Tasks:
{list tasks with IDs and descriptions}
Success Criteria:
{list success criteria}

--- CONSTRAINTS ---
{constraints from constraints.json}

--- YOUR MISSION ---
1. Analyze the tasks and decide how to organize the work
2. Spawn specialists as needed (builders, watchers, scouts)
3. Synthesize their results
4. Verify success criteria are met
5. Report what was accomplished

--- SPAWN LIMITS ---
Max 4 direct spawns (depth 2)
Each spawn can spawn max 2 more (depth 3)
Total cap: 10 workers for this phase

--- WORKER SPECS ---
Read .aether/workers.md for role definitions.

--- OUTPUT FORMAT ---
{
  "status": "completed" | "completed_no_change" | "failed" | "blocked",
  "summary": "What the phase accomplished",
  "tasks_completed": ["1.1", "1.2"],
  "tasks_failed": [],
  "files_created": [],
  "files_modified": [],
  "spawn_tree": {
    "builder-1": {"task": "...", "status": "completed", "children": []},
    "watcher-1": {"task": "...", "status": "completed", "children": []}
  },
  "quality_notes": "Any concerns or recommendations",
  "ui_touched": true | false
}
```

---

## Wisdom Pipeline

Workers participate in the wisdom pipeline through their work products:

1. **Build work** produces observations automatically during continue -- not via a wrapper-invoked
   `memory-capture` call, but through `pkg/learn`'s own in-process capture, `captureContinueLearning()`
   (`cmd/codex_continue_finalize.go:1458`), called from both continue paths (`cmd/codex_continue.go:962`,
   the default path, and `cmd/codex_continue_finalize.go:521`, the heavy-review path). Eligible runs are
   recorded as a hypothesis entry in `pkg/learn`'s own colony store (`.aether/data/learn/`), then handed
   to phase-end consolidation (`runPhaseEndConsolidation`, same call site) for further curation.
2. **Observations** auto-promote to instincts after threshold (2 for patterns)
3. **Instincts** are stored in COLONY_STATE.json with confidence scores
4. **High-confidence instincts** (>= 0.8) are promoted to Hive Brain at seal
5. **Hive wisdom** flows back into future worker prompts via colony-prime

Key subcommands: `memory-capture`, `instinct-create`, `queen-promote`, `hive-promote`, `hive-read`

See CLAUDE.md "Wisdom Pipeline" section for full stage details.
