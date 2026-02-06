---
name: ant:colonize
description: Colonize codebase - analyze existing code before starting project
---

You are the **Queen**. Your only job is to emit a signal and let the colony explore.

## Instructions

### Step 1: Read State

Use the Read tool to read `.aether/data/COLONY_STATE.json`.

Extract:
- `goal`, `state` from top level
- `signals` array (pheromones)

If `COLONY_STATE.json` has `goal: null`, output:

```
No colony initialized. Run /ant:init "<goal>" first.

Colonization works best when the colony knows the goal,
so it can focus analysis on what's relevant.
```

Stop here.

### Step 2: Compute Active Pheromones

Use the Bash tool to run:
```
bash ~/.aether/aether-utils.sh pheromone-batch
```

This returns JSON: `{"ok":true,"result":[...signals with current_strength...]}`. Parse the `result` array. Filter out signals where `current_strength < 0.05`.

If the command fails, treat as "no active pheromones."

Format:

```
ACTIVE PHEROMONES:
- {TYPE} (strength {current_strength:.2f}): "{content}"
```

### Step 2.5: Detect Project Complexity

Use the Bash tool to measure project complexity:

1. **Source file count** (exclude node_modules, .git, dist, build, test, tests, __tests__, spec):
   ```
   find . -type f \( -name "*.ts" -o -name "*.js" -o -name "*.py" -o -name "*.go" -o -name "*.rs" -o -name "*.java" -o -name "*.rb" -o -name "*.php" -o -name "*.swift" -o -name "*.kt" -o -name "*.c" -o -name "*.cpp" -o -name "*.cs" \) -not -path "*/node_modules/*" -not -path "*/.git/*" -not -path "*/dist/*" -not -path "*/build/*" -not -path "*/test/*" -not -path "*/tests/*" -not -path "*/__tests__/*" -not -path "*/spec/*" | wc -l
   ```

2. **Max directory depth** (exclude node_modules, .git):
   ```
   find . -type d -not -path "*/node_modules/*" -not -path "*/.git/*" | awk -F/ '{print NF}' | sort -n | tail -1
   ```

3. **Language count**:
   ```
   find . -type f -not -path "*/node_modules/*" -not -path "*/.git/*" | grep -oE '\.[^./]+$' | sort -u | grep -cE '\.(ts|js|py|go|rs|java|rb|php|swift|kt|c|cpp|cs)'
   ```

**Classify mode based on results:**

- **LIGHTWEIGHT:** <20 source files AND <3 directories deep AND 1 language
- **FULL:** >200 source files OR >6 directories deep OR >3 languages OR monorepo detected (multiple package.json/Cargo.toml/go.mod in different directories)
- **STANDARD:** everything else

Store the mode classification and indicators for use in Step 4 and Step 7.

### Step 3: Update State

Use Write tool to update `COLONY_STATE.json`:
- Set `state` to `"EXECUTING"`
- Set `workers.colonizer` to `"active"`

After updating state, display the colonize header using Bash tool (bold yellow -- Queen color):
```
bash -c 'printf "\e[1;33m+=====================================================+\e[0m\n"'
bash -c 'printf "\e[1;33m|  AETHER COLONY :: COLONIZE                          |\e[0m\n"'
bash -c 'printf "\e[1;33m+=====================================================+\e[0m\n\n"'
bash -c 'printf "  Goal: %s\n  Mode: %s\n\n" "{goal}" "{mode}"'
```

### Step 4: Spawn Colonizer Ants

**Mode check:** If mode from Step 2.5 is **LIGHTWEIGHT**, skip to **Step 4-LITE** below. Otherwise (STANDARD or FULL), use the multi-colonizer pattern in this step.

**Step 4 (STANDARD/FULL mode): Spawn Three Colonizer Ants**

Spawn 3 colonizer ants SEQUENTIALLY via Task tool (`subagent_type="general-purpose"`). Each gets the same prompt header but with a distinct specialization lens. Do NOT hardcode which castes to spawn beyond these 3. Let the colony self-organize within each lens.

Each colonizer receives this common prompt header, followed by its specific mission:

```
You are an ant in the Aether Queen Ant Colony.

The Queen has signalled: colonize the codebase.

--- COLONY CONTEXT ---

Goal: "{goal}"

--- ACTIVE PHEROMONES ---
{pheromone block from Step 2}

Respond to REDIRECT pheromones as hard constraints (things to avoid).
Respond to FOCUS pheromones by prioritizing those areas.

--- HOW THE COLONY WORKS ---

You are autonomous. There is no orchestrator. You decide how to explore this codebase.

If you need help, spawn specialists. See ~/.aether/workers.md for role definitions:
  - colonizer: Explore/index codebase
  - route-setter: Plan and break down work
  - builder: Implement code, run commands
  - watcher: Validate, test, quality check
  - scout: Research, find information
  - architect: Synthesize knowledge, extract patterns

To spawn another ant:
1. Read their spec file with the Read tool
2. Use the Task tool (subagent_type="general-purpose") with prompt containing:
   --- WORKER SPEC ---
   {full contents of the spec file}
   --- ACTIVE PHEROMONES ---
   {copy the pheromone block above}
   --- TASK ---
   {what you need them to do}

Spawned ants can spawn further ants. Max depth 3, max 5 sub-ants per ant.
```

Before each colonizer spawn, display progress in cyan via Bash tool. After each colonizer returns, display completion.

**Colonizer 1 (Structure):**

Before spawn:
```
bash -c 'printf "\e[36m%-14s\e[0m %s (%d/%d)\n" "[COLONIZER]" "Analyzing Structure..." 1 3'
```
After return:
```
bash -c 'printf "\e[36m%-14s\e[0m %s ... \e[32mDONE\e[0m\n" "[COLONIZER]" "Structure analysis"'
```

Append this mission to the common header:

```
--- YOUR MISSION ---

You are Colonizer 1 of 3 (Structure Lens).

Focus ONLY on architecture and organization:
1. Directory structure and module boundaries
2. Main entry points and how they connect
3. Build system and scripts
4. Dependency graph between modules
5. File organization conventions

Do NOT analyze code quality or tech stack details — other colonizers handle those.

Use Glob, Grep, and Read tools to explore. Report your findings as:

COLONIZER 1 (STRUCTURE) REPORT
Findings:
  - category: "structure"
    finding: "<specific observation>"
    confidence: <HIGH|MEDIUM|LOW>
    evidence: "<file path or pattern>"
```

**Colonizer 2 (Patterns):**

Before spawn:
```
bash -c 'printf "\e[36m%-14s\e[0m %s (%d/%d)\n" "[COLONIZER]" "Analyzing Patterns..." 2 3'
```
After return:
```
bash -c 'printf "\e[36m%-14s\e[0m %s ... \e[32mDONE\e[0m\n" "[COLONIZER]" "Patterns analysis"'
```

Append this mission to the common header:

```
--- YOUR MISSION ---

You are Colonizer 2 of 3 (Patterns Lens).

Focus ONLY on code quality and conventions:
1. Naming conventions (files, variables, functions, classes)
2. Design patterns in use (and anti-patterns)
3. Error handling approach
4. Code style and formatting
5. Documentation patterns

Sample 5-10 representative files across the codebase.
Do NOT map the full directory structure — Colonizer 1 handles that.

Use Glob, Grep, and Read tools to explore. Report your findings as:

COLONIZER 2 (PATTERNS) REPORT
Findings:
  - category: "patterns"
    finding: "<specific observation>"
    confidence: <HIGH|MEDIUM|LOW>
    evidence: "<file path or pattern>"
```

**Colonizer 3 (Stack):**

Before spawn:
```
bash -c 'printf "\e[36m%-14s\e[0m %s (%d/%d)\n" "[COLONIZER]" "Analyzing Stack..." 3 3'
```
After return:
```
bash -c 'printf "\e[36m%-14s\e[0m %s ... \e[32mDONE\e[0m\n" "[COLONIZER]" "Stack analysis"'
```

Append this mission to the common header:

```
--- YOUR MISSION ---

You are Colonizer 3 of 3 (Stack Lens).

Focus ONLY on technology and dependencies:
1. Languages and frameworks in use (with versions)
2. Dependency health (outdated? vulnerable?)
3. Configuration approach (env vars, config files, hardcoded)
4. Build/deploy pipeline
5. External service integrations

Check package manifests (package.json, requirements.txt, Cargo.toml, etc.).
Do NOT review individual source files — other colonizers handle that.

Use Glob, Grep, and Read tools to explore. Report your findings as:

COLONIZER 3 (STACK) REPORT
Findings:
  - category: "stack"
    finding: "<specific observation>"
    confidence: <HIGH|MEDIUM|LOW>
    evidence: "<file path or pattern>"
```

Save individual colonizer reports to `.aether/temp/colonizer-{1,2,3}-report.txt`.

### Step 4.5: Synthesize Colonizer Reports

**This step is performed by the Queen (main agent), NOT a separate Task tool spawn.**

Only runs for STANDARD/FULL mode. If LIGHTWEIGHT, skip this step (Step 4-LITE output is used directly).

Display synthesis start using Bash tool (bold yellow -- Queen color):
```
bash -c 'printf "\n\e[1;33m%-14s\e[0m %s\n" "[QUEEN]" "Synthesizing colonizer reports..."'
```

1. Collect all 3 colonizer reports from Step 4
2. Group findings by topic (architecture, tech stack, conventions, concerns)
3. Where 2+ colonizers agree on a finding: include as HIGH confidence
4. Where colonizers disagree: flag explicitly:

```
DISAGREEMENT: {topic}
  Colonizer {N} ({Lens}): {view}
  Colonizer {M} ({Lens}): {opposing view}
  Resolution: User decision needed
```

5. Produce a unified synthesis report for display in Step 6

After synthesis completes, display using Bash tool:
```
bash -c 'printf "\e[1;33m%-14s\e[0m %s ... \e[32mDONE\e[0m\n" "[QUEEN]" "Synthesis complete"'
```

### Step 4-LITE: Single Colonizer (LIGHTWEIGHT mode)

Only runs when mode from Step 2.5 is LIGHTWEIGHT. Spawns a single colonizer ant covering all lenses.

Before spawn, display progress using Bash tool (cyan -- colonizer color):
```
bash -c 'printf "\e[36m%-14s\e[0m %s\n" "[COLONIZER]" "Analyzing codebase (lightweight)..."'
```

Use the **Task tool** with `subagent_type="general-purpose"`:

```
You are an ant in the Aether Queen Ant Colony.

The Queen has signalled: colonize the codebase.

--- COLONY CONTEXT ---

Goal: "{goal}"

--- ACTIVE PHEROMONES ---
{pheromone block from Step 2}

Respond to REDIRECT pheromones as hard constraints (things to avoid).
Respond to FOCUS pheromones by prioritizing those areas.

--- HOW THE COLONY WORKS ---

You are autonomous. There is no orchestrator. You decide how to explore this codebase.

If you need help, spawn specialists. See ~/.aether/workers.md for role definitions:
  - colonizer: Explore/index codebase
  - route-setter: Plan and break down work
  - builder: Implement code, run commands
  - watcher: Validate, test, quality check
  - scout: Research, find information
  - architect: Synthesize knowledge, extract patterns

To spawn another ant:
1. Read their spec file with the Read tool
2. Use the Task tool (subagent_type="general-purpose") with prompt containing:
   --- WORKER SPEC ---
   {full contents of the spec file}
   --- ACTIVE PHEROMONES ---
   {copy the pheromone block above}
   --- TASK ---
   {what you need them to do}

Spawned ants can spawn further ants. Max depth 3, max 5 sub-ants per ant.

--- YOUR MISSION ---

Understand this codebase. Analyze:
1. Directory structure and file organization
2. Main entry points and key modules
3. Architecture patterns and design decisions
4. Tech stack (languages, frameworks, dependencies)
5. Code conventions (naming, formatting, style)
6. Dependencies between components

Focus on what's relevant to the colony goal.

Use Glob, Grep, and Read tools to explore. Report your findings.
```

After colonizer returns, display completion using Bash tool:
```
bash -c 'printf "\e[36m%-14s\e[0m %s ... \e[32mDONE\e[0m\n" "[COLONIZER]" "Analysis complete"'
```

### Step 5: Persist Findings

After colonization completes (either synthesis report from Step 4.5 or single colonizer report from Step 4-LITE), save findings so they survive the session.

Read `.aether/data/COLONY_STATE.json`. Append a decision record to the `memory.decisions` array:

```json
{
  "id": "dec_<unix_timestamp>_<4_random_hex>",
  "type": "colonization",
  "content": "<For STANDARD/FULL: summarize the unified synthesis from all 3 colonizers — include key structure, pattern, and stack findings plus any disagreements flagged. For LIGHTWEIGHT: summarize the single colonizer's findings. Cover project type, tech stack, architecture patterns, conventions, and recommendations — keep under 500 chars>",
  "context": "Codebase colonized for goal: <goal>",
  "phase": 0,
  "timestamp": "<ISO-8601 UTC>"
}
```

If the `memory.decisions` array exceeds 30 entries, remove the oldest entries to keep only 30.

**Write Event:** Append to the `events` array as pipe-delimited string:
`"<timestamp>|codebase_colonized|colonize|Codebase colonized ({LIGHTWEIGHT|STANDARD|FULL} mode): <project type>, <primary language/framework>"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Use the Write tool to write the updated COLONY_STATE.json.

### Step 5.5: Inject Global Learnings

After persisting colonization findings, inject relevant global learnings as FEEDBACK pheromones.

1. **Check for global learnings file:**
   Run:
   ```
   bash ~/.aether/aether-utils.sh learning-inject "<tech_keywords>"
   ```

   Where `<tech_keywords>` is a comma-separated list of technology keywords extracted from the colonization findings just persisted in Step 5. Derive these from:
   - Languages detected (e.g., "typescript", "python", "go")
   - Frameworks detected (e.g., "react", "express", "django")
   - Domain keywords (e.g., "api", "frontend", "cli")

   Example: for a TypeScript React project, run:
   ```
   bash ~/.aether/aether-utils.sh learning-inject "typescript,react,frontend"
   ```

   This returns JSON: `{"ok":true,"result":{"learnings":[...],"count":N}}`

2. **If count is 0 or learnings array is empty:** Display "No matching global learnings found." and skip to Step 6.

3. **For each relevant learning returned:**
   Append a FEEDBACK pheromone to the `signals` array in COLONY_STATE.json:
   ```json
   {
     "id": "global_inject_<unix_timestamp>_<4_random_hex>",
     "type": "FEEDBACK",
     "content": "Global learning: <learning.content>",
     "strength": 0.5,
     "half_life_seconds": 86400,
     "created_at": "<ISO-8601 UTC>",
     "source": "global:inject",
     "auto": true
   }
   ```

   Before appending, validate the content:
   ```
   bash ~/.aether/aether-utils.sh pheromone-validate "Global learning: <learning.content>"
   ```
   - If pass:false -> skip this learning (content too short)
   - If pass:true -> append to signals array
   - If command fails -> append anyway (fail-open)

4. **Write updated COLONY_STATE.json** with the injected FEEDBACK pheromones in `signals` array.

5. **Display injected learnings** using Bash tool (bold yellow -- Queen color):
   ```
   bash -c 'printf "\n\e[1;33m%-14s\e[0m Injected %d global learnings as FEEDBACK pheromones:\n" "[QUEEN]" {count}'
   ```
   For each injected learning:
   ```
   bash -c 'printf "  \e[33mFEEDBACK\e[0m (0.5, 24h): %s\n" "<first 80 chars of learning content>"'
   ```

   Note: 24-hour half-life (vs normal 6-hour FEEDBACK) ensures learnings persist through the planning phase.

### Step 6: Display Results

Display the colonization results. For STANDARD/FULL mode, display the unified synthesis report from Step 4.5 (NOT individual colonizer reports — those are stored in `.aether/temp/` for reference). For LIGHTWEIGHT mode, display the single colonizer's report from Step 4-LITE.

Before the results display, show step progress checkmarks using Bash tool:
```
bash -c 'printf "\n  \e[32m✓\e[0m Step 1: Read State\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 2: Compute Pheromones\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 2.5: Detect Complexity\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 3: Update State\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 4: Colonize (%s mode)\n" "{mode}"'
bash -c 'printf "  \e[32m✓\e[0m Step 5: Persist Findings\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 5.5: Inject Global Learnings\n"'
bash -c 'printf "  \e[32m✓\e[0m Step 6: Display Results\n\n"'
```

Display the colored result header using Bash tool (cyan -- colonizer color):
```
bash -c 'printf "\e[36m+=====================================================+\e[0m\n"'
bash -c 'printf "\e[36m|  CODEBASE COLONIZED                                 |\e[0m\n"'
bash -c 'printf "\e[36m+=====================================================+\e[0m\n\n"'
```

Analyze the findings to suggest specific pheromones:

```

  Goal: "{goal}"
  Mode: {LIGHTWEIGHT|STANDARD|FULL}

{synthesis report (STANDARD/FULL) or single colonizer report (LIGHTWEIGHT)}

  Findings saved to COLONY_STATE.json

Suggested Pheromone Injections:
  Based on colonization findings:

  /ant:focus "<specific area identified from the ant's report>"
    Why: <concrete reason derived from the analysis — reference actual finding>

  {if the ant identified problematic patterns, anti-patterns, or risks:}
  /ant:redirect "<specific pattern to avoid based on the ant's findings>"
    Why: <concrete reason derived from the analysis — reference actual finding>

  Skip these if you want the colony to plan without guidance.

Next:
  /ant:plan              Generate project plan
  /ant:focus "<area>"    Inject focus before planning
  /ant:redirect "<pat>"  Inject constraint before planning
```

**CRITICAL:** The pheromone suggestions MUST be derived from the ACTUAL colonizer ant report returned in Step 4. Analyze the ant's specific findings — its tech stack observations, architectural patterns, code quality issues, conventions detected — and formulate 1-2 concrete focus/redirect suggestions that reference those findings. Do NOT output generic boilerplate like "consider focusing on important areas."

If the ant's report contains no clear focus areas or problematic patterns, display instead:

```
  No specific pheromone injections suggested — analysis was clean.
  You can still inject guidance manually if you have preferences.
```

### Step 7: Reset State

Use Write tool to update `.aether/data/COLONY_STATE.json`:
- Set `state` to `"READY"`
- Set `workers.colonizer` to `"idle"`
- Set `mode` to the classified mode from Step 2.5 (`"LIGHTWEIGHT"`, `"STANDARD"`, or `"FULL"`)
- Set `mode_set_at` to the current ISO-8601 UTC timestamp
- Set `mode_indicators` to the complexity detection results:
  ```json
  "mode_indicators": {
    "source_files": <count from Step 2.5>,
    "max_depth": <count from Step 2.5>,
    "languages": <count from Step 2.5>
  }
  ```

Add these fields to the existing Write -- do NOT create a separate write operation.

### Step 8: Persistence Confirmation

After resetting state in Step 7, display:

```
---
All state persisted. Safe to /clear context if needed.
  State: .aether/data/ (6 files validated)
  Resume: /ant:resume-colony
```
