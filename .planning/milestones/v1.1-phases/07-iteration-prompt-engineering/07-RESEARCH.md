# Phase 7: Iteration Prompt Engineering - Research

**Researched:** 2026-03-13
**Domain:** Oracle prompt engineering, research lifecycle phases, gap-targeted iteration, confidence-driven prioritization, structured state updates from AI iterations
**Confidence:** HIGH

## Summary

Phase 7 transforms oracle.md from a static one-size-fits-all prompt into a phase-aware prompt system that changes behavior based on the research lifecycle stage (survey / investigate / synthesize / verify). The core problem is not "make the AI smarter" but rather "give the AI the right instructions at the right moment so it deepens rather than repeats." Phase 6 laid the data foundation (state.json, plan.json, gaps.md, synthesis.md); Phase 7 makes the iteration prompt exploit that structure to drive targeted, deepening research.

The current oracle.md (69 lines, written in Phase 6) already targets the lowest-confidence question per iteration and writes structured updates -- but it uses a single behavior mode regardless of whether it is the first iteration (broad survey), middle iterations (deep investigation), or final iterations (synthesis/verification). The Phase 7 upgrade introduces: (1) phase transitions in state.json driven by structural conditions (not self-assessment), (2) different prompt instructions per phase that change what the AI focuses on and how it writes findings, (3) iteration counter management in oracle.sh (currently missing), (4) depth enforcement via explicit instructions that reference prior findings and demand new information, and (5) optional `--json-schema` support in Claude CLI to guarantee valid plan.json updates (with prompt-based fallback for OpenCode).

The key architectural insight from Anthropic's own multi-agent research system is "start with short, broad queries, evaluate what's available, then progressively narrow focus." This maps directly to the survey-investigate-synthesize-verify lifecycle. The Ralph pattern (which oracle is based on) uses a simpler model (pick next failing story, implement it, move on), but oracle research requires a more nuanced approach because questions have degrees of confidence, not binary pass/fail.

**Primary recommendation:** Replace oracle.md with a phase-aware prompt that reads `state.json.phase` and emits different instructions per lifecycle stage. Add phase transition logic in oracle.sh (after each iteration) based on structural conditions. Add iteration counter increment in oracle.sh. Add `--json-schema` support for Claude CLI invocations with prompt-based fallback for OpenCode.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
None explicitly locked -- all decisions delegated to Claude's discretion.

### Claude's Discretion
- **Research lifecycle phases** -- How survey / investigate / synthesize / verify phases work, what triggers transitions between them, how prompt behavior changes at each stage
- **Gap targeting strategy** -- How the prompt selects what to research next, confidence scoring approach (0-100%), prioritization logic for lowest-confidence open questions
- **State update format** -- What each iteration writes back to gaps.md, plan.json, and synthesis.md, how the prompt instructs Claude to produce valid structured updates
- **Depth vs repetition prevention** -- How prompts ensure iterations go deeper rather than restating, what constitutes "measurably deeper findings," how the prompt prevents research loops

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOOP-02 | Each iteration reads structured state first, then targets the highest-priority knowledge gap -- gap-driven iteration, not topic-based | The prompt reads all 4 state files in Step 1, then selects the lowest-confidence non-answered question. Phase-specific behavior refines what "targeting" means at each lifecycle stage. |
| LOOP-03 | Oracle uses phase-aware prompts (survey -> investigate -> synthesize -> verify) that change behavior based on research lifecycle stage | Four prompt modes with distinct instructions per phase. Phase transitions driven by oracle.sh based on structural conditions (coverage, confidence thresholds, iteration count). |
| INTL-02 | After each iteration, oracle identifies remaining unknowns and contradictions, updating gaps.md | Every phase includes a gaps.md update step. The survey phase populates initial gaps; investigate refines them; synthesize consolidates; verify confirms resolution. |
| INTL-03 | Per-question confidence scoring (0-100%) drives which areas get researched next | Confidence scoring is already in plan.json (Phase 6). Phase 7 adds: explicit scoring rubric in the prompt, phase-specific confidence guidance, and oracle.sh reading confidence to inform phase transitions. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| oracle.md | Prompt file | Phase-aware iteration prompt with four behavioral modes | This IS the primary deliverable of Phase 7 |
| oracle.sh | Bash orchestrator | Phase transition logic, iteration counter, optional --json-schema | Existing loop; needs phase transition and iteration increment additions |
| aether-utils.sh | ~9,808 lines | Existing validate-oracle-state for post-iteration validation | Already handles state/plan validation from Phase 6 |
| jq | 1.8.1 | JSON state reading/writing in oracle.sh, phase transition logic | Project standard; used throughout |
| ava | ^6.0.0 | Unit/integration tests for phase transitions and prompt behavior | Project standard test runner |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| claude CLI --json-schema | Current | Structured output guarantee for plan.json updates | When claude CLI is detected (not available in opencode) |
| --output-format json | Current | Required alongside --json-schema for structured output | Paired with --json-schema |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Phase transitions in oracle.sh (bash) | Phase transitions in oracle.md (AI self-manages) | AI self-management is unreliable; oracle.sh can enforce transitions structurally based on actual state |
| Four separate prompt files (survey.md, investigate.md, etc.) | Single oracle.md with conditional sections | Single file is simpler to maintain; conditional sections via bash string injection before piping to AI |
| --json-schema for all state files | --json-schema for plan.json only | plan.json is the critical structured file; gaps.md and synthesis.md are markdown (no schema needed); state.json updates are minimal |

## Architecture Patterns

### Recommended Approach: Prompt Injection via oracle.sh

Rather than modifying oracle.md into a static file with all four phase prompts, the recommended pattern is:

1. oracle.md contains the **base prompt** (role, rules, common instructions)
2. oracle.sh reads `state.json.phase` and constructs a **phase-specific directive** as a prefix
3. oracle.sh pipes the combined prompt (phase directive + base prompt) to the AI CLI

This pattern:
- Keeps oracle.md readable (single file, clear structure)
- Gives oracle.sh control over phase transitions (not the AI)
- Allows phase-specific instructions without the AI needing to "choose" which to follow

### Pattern 1: Phase-Aware Prompt Structure

**What:** oracle.md structured with a common base plus phase-specific section injected by oracle.sh
**When to use:** Every iteration

**oracle.md structure (Phase 7 rewrite):**
```markdown
You are an **Oracle Ant** - a deep research agent in the Aether Colony.

## Current Phase
{INJECTED BY ORACLE.SH -- phase-specific instructions appear here}

## Instructions

### Step 1: Read State Files
[same as current -- read all 4 files]

### Step 2: Identify Target
[enhanced with phase-specific targeting]

### Step 3: Research
[enhanced with phase-specific research style]

### Step 4: Update State Files
[enhanced with depth enforcement]

### Step 5: Assess and Complete
[same completion logic]

## Confidence Scoring Rubric
[NEW -- explicit rubric so AI scores consistently]

## Important Rules
[updated with depth-enforcement rules]
```

**Phase-specific directives (constructed by oracle.sh):**

**Survey phase:**
```
## Current Phase: SURVEY (Broad Exploration)
You are in the SURVEY phase. Your goal is to build a broad understanding of all sub-questions.
- Cast a wide net -- get initial findings for every open question, not deep dives on any single one
- Target questions you have NOT yet touched (iterations_touched is empty)
- Aim for 20-40% confidence per question -- enough to know the landscape
- After this iteration, list all discovered unknowns in gaps.md
```

**Investigate phase:**
```
## Current Phase: INVESTIGATE (Deep Research)
You are in the INVESTIGATE phase. Your goal is to deeply research the weakest areas.
- Target the lowest-confidence question and go DEEP -- find primary sources, verify claims, explore edge cases
- You MUST reference what was already found in synthesis.md and ADD NEW information, not restate it
- If you cannot find new information beyond what is in synthesis.md, say so explicitly and move to the next question
- Aim to push confidence above 70% for your target question
- Update gaps.md with specific remaining unknowns (not vague restatements)
```

**Synthesize phase:**
```
## Current Phase: SYNTHESIZE (Connect and Conclude)
You are in the SYNTHESIZE phase. Your goal is to connect findings across questions into a coherent whole.
- Read ALL findings in synthesis.md before doing anything
- Identify connections, patterns, and contradictions ACROSS questions
- Consolidate redundant findings -- if two questions produced overlapping information, merge and attribute
- Resolve contradictions by finding authoritative sources or noting irreconcilable differences
- Push overall confidence toward the target -- fill the remaining specific gaps
```

**Verify phase:**
```
## Current Phase: VERIFY (Validate and Confirm)
You are in the VERIFY phase. Your goal is to validate key claims and close remaining gaps.
- Focus on claims marked in gaps.md contradictions section
- Cross-reference key findings with additional sources
- Confirm or correct confidence scores based on evidence quality
- If a question's findings are well-supported by multiple sources, mark it "answered" with 90%+ confidence
- Final gaps.md should contain only genuinely unresolvable unknowns
```

### Pattern 2: Phase Transition Logic (oracle.sh)

**What:** Structural conditions in oracle.sh that determine when to advance to the next phase
**When to use:** After each iteration, before the next starts

**Transition rules:**

```bash
# Phase transition logic (added to oracle.sh after iteration completes)
determine_phase() {
  local state_file="$1"
  local plan_file="$2"

  local current_phase iteration
  current_phase=$(jq -r '.phase' "$state_file")
  iteration=$(jq -r '.iteration' "$state_file")

  # Read structural metrics from plan.json
  local total_questions touched_count avg_confidence all_above_threshold
  total_questions=$(jq '.questions | length' "$plan_file")
  touched_count=$(jq '[.questions[] | select(.iterations_touched | length > 0)] | length' "$plan_file")
  avg_confidence=$(jq '[.questions[].confidence] | add / length | floor' "$plan_file")
  all_above_threshold=$(jq '[.questions[] | select(.confidence >= 25)] | length' "$plan_file")

  case "$current_phase" in
    survey)
      # Transition to investigate when: all questions touched OR avg confidence >= 25%
      if [ "$touched_count" -ge "$total_questions" ] || [ "$avg_confidence" -ge 25 ]; then
        echo "investigate"
      else
        echo "survey"
      fi
      ;;
    investigate)
      # Transition to synthesize when: avg confidence >= 60% OR fewer than 2 questions below 50%
      local below_50
      below_50=$(jq '[.questions[] | select(.confidence < 50)] | length' "$plan_file")
      if [ "$avg_confidence" -ge 60 ] || [ "$below_50" -le 1 ]; then
        echo "synthesize"
      else
        echo "investigate"
      fi
      ;;
    synthesize)
      # Transition to verify when: avg confidence >= 80%
      if [ "$avg_confidence" -ge 80 ]; then
        echo "verify"
      else
        echo "synthesize"
      fi
      ;;
    verify)
      # Stay in verify until completion
      echo "verify"
      ;;
    *)
      echo "survey"
      ;;
  esac
}
```

**Key design decisions:**
- Transitions are based on **structural conditions** (touched count, average confidence, distribution), not AI self-assessment
- Thresholds are: survey->investigate at 25% avg, investigate->synthesize at 60% avg, synthesize->verify at 80% avg
- These thresholds are initial values -- Phase 8 may tune them empirically
- The AI cannot skip or change phases -- oracle.sh controls transitions

### Pattern 3: Iteration Counter Management

**What:** oracle.sh increments `state.json.iteration` after each AI invocation
**When to use:** After each iteration, before phase transition check
**Why needed:** Currently oracle.sh does NOT increment the iteration counter in state.json (the for loop variable `$i` is local only). The AI needs accurate iteration numbers in state.json.

```bash
# After AI iteration completes, increment iteration in state.json
jq '.iteration += 1 | .last_updated = now | todate' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
```

Note: A simpler approach using `--arg`:
```bash
ITER_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
jq --arg ts "$ITER_TS" '.iteration += 1 | .last_updated = $ts' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
```

### Pattern 4: --json-schema for Structured Plan Updates (Claude CLI Only)

**What:** When claude CLI is detected, use `--json-schema` to guarantee valid plan.json output
**When to use:** Only for Claude CLI invocations; OpenCode falls back to prompt-based enforcement
**Limitation:** `--json-schema` constrains the FINAL output of the AI, not intermediate tool calls. Since oracle iterations use tools (Read, Write, Grep, etc.) and the structured output would be the final response text, this approach needs careful design.

**Important finding:** `--json-schema` works with `--output-format json` and produces output in a `structured_output` field. However, the oracle pattern pipes oracle.md as stdin and expects the AI to use tools to read/write files. The `--json-schema` flag constrains the response text, not file writes. This means:

- The AI still reads state files using tools and writes updated state files using tools
- `--json-schema` could be used to enforce a SUMMARY response (e.g., what question was targeted, what confidence was assigned, whether to mark complete)
- But it CANNOT enforce the structure of files the AI writes via the Write tool

**Recommendation:** Do NOT use `--json-schema` for enforcing plan.json structure in Phase 7. Instead:
1. Continue using prompt-based instructions (write COMPLETE JSON, follow schema)
2. Rely on post-iteration jq validation in oracle.sh (already in place from Phase 6)
3. Phase 8 adds recovery logic if validation fails
4. If `--json-schema` is used at all, use it for a structured response summary (Phase 8 concern)

### Pattern 5: Depth Enforcement in Prompts

**What:** Explicit prompt instructions that force the AI to reference prior findings and demonstrate new information
**When to use:** Investigate and later phases

**Approach:**
```markdown
## Depth Enforcement Rules
1. Before writing ANY finding to synthesis.md, READ the existing findings for your target question
2. Your new findings MUST contain information NOT already in synthesis.md
3. If you write a finding that restates what is already known, you are failing your mission
4. Acceptable new information includes: specific details, concrete examples, source citations, edge cases, limitations, contradictions with existing findings
5. If you cannot find NEW information, write "No new findings beyond existing research" and move to the next-lowest-confidence question
```

**Measurability:** "Deeper findings" can be measured by:
- New key_findings entries in plan.json that differ from existing ones (string comparison)
- Increasing confidence scores (shows evidence accumulation)
- Shrinking gaps.md (fewer open questions, resolved contradictions)
- synthesis.md growing with distinct content per iteration

### Pattern 6: Confidence Scoring Rubric

**What:** Explicit rubric in the prompt so the AI scores confidence consistently
**When to use:** Embedded in oracle.md, referenced every iteration

```markdown
## Confidence Scoring Rubric
Score each question's confidence based on evidence quality:

| Score | Meaning | Evidence Required |
|-------|---------|-------------------|
| 0-20% | Unexplored | No research conducted yet |
| 20-40% | Surface level | Found general information, no specifics |
| 40-60% | Partial understanding | Found specific details from 1-2 sources |
| 60-80% | Good understanding | Multiple sources agree, edge cases known |
| 80-95% | Thorough | Primary sources verified, contradictions resolved |
| 95-100% | Exhaustive | All reasonable angles explored, high-quality sources |

Do NOT inflate confidence. A question with one blog post as evidence is 30%, not 70%.
Do NOT deflate confidence to keep research going. Score honestly.
```

### Anti-Patterns to Avoid

- **AI-controlled phase transitions:** The AI should NOT decide when to transition between survey/investigate/synthesize/verify. It lacks the objectivity to assess its own progress. oracle.sh makes transitions based on structural metrics.
- **Self-assessed confidence as sole driver:** Confidence scores from the AI are subjective. Use them as INPUT to structural transition logic, but don't let the AI's own confidence rating alone drive phase changes. Cross-check with touched_count, gaps resolution, and novelty.
- **Prompt bloat:** The combined prompt (base + phase directive) should stay under 3,000 tokens. Each phase directive should be 100-200 words. More instructions reduce compliance, not increase it.
- **Trying to use --json-schema for file writes:** `--json-schema` constrains response text, not tool calls. Don't architect around it for file structure enforcement.
- **Removing existing Phase 6 behaviors:** Phase 7 ADDS phase-awareness on top of Phase 6's gap targeting. Don't remove the "target lowest-confidence question" logic -- refine it per phase.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON state validation | Custom validation in oracle.md | validate-oracle-state in aether-utils.sh + post-iteration jq check in oracle.sh | Already built in Phase 6; prompt enforcement is unreliable |
| Phase transition timing | AI decides when to switch | oracle.sh structural conditions (touched count, avg confidence) | AI cannot objectively assess its own progress |
| Iteration counter | AI updates iteration field | oracle.sh increments after each call | The AI might forget or corrupt the counter |
| research-plan.md regeneration | AI rewrites it | generate_research_plan function in oracle.sh | Already built in Phase 6; deterministic is better than AI-written |
| Repetition detection | Complex AI self-analysis | Prompt instruction: "read existing findings FIRST, write ONLY new information" | Simple prompt instruction is more reliable than detection algorithms |

**Key insight:** Phase 7 is about prompt engineering -- making the AI do better work within each iteration. The structural controls (phase transitions, iteration counting, validation) belong in oracle.sh, not in the prompt.

## Common Pitfalls

### Pitfall 1: AI Ignores Phase Directives
**What goes wrong:** The AI treats phase-specific instructions as suggestions and researches however it wants
**Why it happens:** Long prompts with multiple instruction sections compete for attention; the AI may default to its general research behavior
**How to avoid:** Put the phase directive at the TOP of the prompt (before any other instructions). Use direct, imperative language ("You MUST", not "Consider"). Keep phase directives short and specific (100-200 words).
**Warning signs:** AI does deep-dives during survey phase; AI does broad surveys during verify phase

### Pitfall 2: Confidence Score Inflation
**What goes wrong:** The AI rates every question at 70-80% after minimal research, causing premature phase transitions
**Why it happens:** LLMs tend toward middle-high confidence; they feel "pretty confident" about surface-level findings
**How to avoid:** Explicit scoring rubric with evidence requirements. The rubric ties scores to concrete evidence quality, not feelings. Phase transition thresholds account for some inflation (25% for survey->investigate is low enough to catch this).
**Warning signs:** Average confidence jumps to 60%+ after 1-2 iterations; synthesis.md has thin findings despite high confidence

### Pitfall 3: Stale Phase After Transition
**What goes wrong:** oracle.sh updates state.json.phase but the next AI iteration still receives the OLD phase directive
**Why it happens:** Phase is determined after iteration N but the prompt for iteration N+1 needs to read the new phase
**How to avoid:** Phase transition logic runs BEFORE constructing the prompt for the next iteration. Sequence: AI runs -> increment iteration -> check/update phase in state.json -> construct next prompt with new phase -> AI runs.
**Warning signs:** AI behavior doesn't change after phase transition logged in oracle.sh output

### Pitfall 4: Prompt Length Exceeds Context Budget
**What goes wrong:** The injected phase directive + base prompt + state file references become too long, causing the AI to miss instructions
**Why it happens:** Phase-specific instructions added without removing or condensing existing instructions
**How to avoid:** Total prompt (oracle.md) should stay under 200 lines / 3,000 tokens. Phase directives are 10-20 lines each. Test the complete prompt size before committing.
**Warning signs:** AI ignores rules listed at the end of the prompt; AI writes partial JSON

### Pitfall 5: Iteration Counter Mismatch
**What goes wrong:** state.json.iteration doesn't match the actual number of completed iterations
**Why it happens:** oracle.sh's for loop variable `$i` and state.json.iteration are independent. Currently state.json.iteration is NEVER incremented (Phase 6 gap).
**How to avoid:** oracle.sh must increment state.json.iteration after each AI call. The increment should happen BEFORE phase transition logic (so transitions see the correct iteration count).
**Warning signs:** state.json shows iteration 0 after multiple iterations; research-plan.md shows "Iteration 0 of 15" permanently

### Pitfall 6: OpenCode Parity
**What goes wrong:** Phase 7 changes to oracle.md and oracle.sh work for Claude CLI but not OpenCode
**Why it happens:** `--json-schema` is Claude-only; opencode uses `opencode run` with different flags
**How to avoid:** Don't use --json-schema in oracle.sh for Phase 7 (recommendation already made). All prompt changes in oracle.md work identically for both CLIs. Phase transitions in oracle.sh use only jq (portable). Test with both CLIs if possible.
**Warning signs:** opencode invocations fail or produce different results

## Code Examples

### Example 1: Phase Transition in oracle.sh (After Each Iteration)

```bash
# Source: New code for Phase 7 -- extends oracle.sh main loop
# Place after jq validation checks, before generate_research_plan

# Increment iteration counter
ITER_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
jq --arg ts "$ITER_TS" '.iteration += 1 | .last_updated = $ts' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"

# Determine and apply phase transition
NEW_PHASE=$(determine_phase "$STATE_FILE" "$PLAN_FILE")
CURRENT_PHASE=$(jq -r '.phase' "$STATE_FILE")
if [ "$NEW_PHASE" != "$CURRENT_PHASE" ]; then
  echo "  Phase transition: $CURRENT_PHASE -> $NEW_PHASE"
  jq --arg phase "$NEW_PHASE" '.phase = $phase' "$STATE_FILE" > "$STATE_FILE.tmp" && mv "$STATE_FILE.tmp" "$STATE_FILE"
fi
```

### Example 2: Prompt Construction with Phase Injection

```bash
# Source: New code for Phase 7 -- replaces direct pipe of oracle.md
# Build the complete prompt by prepending phase-specific directive

build_oracle_prompt() {
  local state_file="$1"
  local prompt_file="$2"

  local phase iteration
  phase=$(jq -r '.phase' "$state_file")
  iteration=$(jq -r '.iteration' "$state_file")

  # Phase-specific directive
  case "$phase" in
    survey)
      cat <<'PHASE_DIRECTIVE'
## Current Phase: SURVEY (Broad Exploration)
You are in the SURVEY phase. Your goal is to build a broad understanding of all sub-questions.
- Cast a wide net -- get initial findings for every open question, not deep dives on any single one
- Target questions you have NOT yet touched (iterations_touched is empty)
- Aim for 20-40% confidence per question -- enough to know the landscape
- After this iteration, list all discovered unknowns in gaps.md
PHASE_DIRECTIVE
      ;;
    investigate)
      cat <<'PHASE_DIRECTIVE'
## Current Phase: INVESTIGATE (Deep Research)
You are in the INVESTIGATE phase. Your goal is to deeply research the weakest areas.
- Target the lowest-confidence question and go DEEP -- find primary sources, verify claims, explore edge cases
- You MUST reference what was already found in synthesis.md and ADD NEW information, not restate it
- If you cannot find new information beyond what synthesis.md already contains, say so and move to the next question
- Aim to push confidence above 70% for your target question
- Update gaps.md with specific remaining unknowns (not vague restatements)
PHASE_DIRECTIVE
      ;;
    synthesize)
      cat <<'PHASE_DIRECTIVE'
## Current Phase: SYNTHESIZE (Connect and Conclude)
You are in the SYNTHESIZE phase. Your goal is to connect findings across questions into a coherent whole.
- Read ALL findings in synthesis.md before doing anything
- Identify connections, patterns, and contradictions ACROSS questions
- Consolidate redundant findings -- merge overlapping information and attribute correctly
- Resolve contradictions by finding authoritative sources or noting irreconcilable differences
- Push overall confidence toward the target -- fill the remaining specific gaps
PHASE_DIRECTIVE
      ;;
    verify)
      cat <<'PHASE_DIRECTIVE'
## Current Phase: VERIFY (Validate and Confirm)
You are in the VERIFY phase. Your goal is to validate key claims and close remaining gaps.
- Focus on claims marked in gaps.md contradictions section
- Cross-reference key findings with additional sources
- Confirm or correct confidence scores based on evidence quality
- If findings are well-supported by multiple sources, mark the question "answered" with 90%+ confidence
- Final gaps.md should contain only genuinely unresolvable unknowns
PHASE_DIRECTIVE
      ;;
  esac

  echo ""
  cat "$prompt_file"
}
```

### Example 3: Updated AI Invocation in oracle.sh Main Loop

```bash
# Source: Replaces line 137 in current oracle.sh
# Build prompt with phase injection and pipe to AI

PROMPT=$(build_oracle_prompt "$STATE_FILE" "$SCRIPT_DIR/oracle.md")
OUTPUT=$(echo "$PROMPT" | $AI_CMD 2>&1 | tee /dev/stderr) || true
```

### Example 4: Phase 7 oracle.md (Complete Rewrite)

```markdown
You are an **Oracle Ant** - a deep research agent in the Aether Colony.

## Your Mission

Research a topic thoroughly. Each iteration targets knowledge gaps and deepens understanding.
You are working through a structured research plan with tracked sub-questions.

## Instructions

### Step 1: Read State Files
Read these files to understand the current research state:
- `.aether/oracle/state.json` -- Session metadata (topic, scope, iteration, phase, confidence)
- `.aether/oracle/plan.json` -- Sub-questions with status, confidence, and key findings
- `.aether/oracle/gaps.md` -- Current knowledge gaps and contradictions
- `.aether/oracle/synthesis.md` -- Accumulated findings organized by question

Note the `iteration` and `phase` fields in state.json -- they tell you where you are in the research lifecycle.

### Step 2: Identify Target
From plan.json, find your research target based on the current phase:
- **Survey:** Target questions with empty `iterations_touched` arrays (untouched questions first). If all are touched, target the lowest-confidence non-answered question.
- **Investigate/Synthesize/Verify:** Target the lowest-confidence question that is NOT "answered".

If all questions are "answered", proceed to Step 5.

### Step 3: Research
Research the target question using available tools.

**Before writing any findings:**
1. Read the existing findings for your target question in synthesis.md
2. Your new findings MUST contain information NOT already in synthesis.md
3. If you cannot find new information, state "No new findings beyond existing research" and target the next-lowest-confidence question instead

**Research approach:**
- Use Glob, Grep, Read for codebase research
- Use WebSearch, WebFetch for web research
- Focus on the specific knowledge gap for your target question
- Find evidence that increases or decreases confidence
- Identify contradictions with existing findings

### Step 4: Update State Files
After researching, update ALL of these files:

**plan.json** -- Update the target question:
- Set `status` to "partial" (useful information found, gaps remain) or "answered" (thoroughly addressed)
- Update `confidence` (0-100) using the scoring rubric below
- Add brief key findings to `key_findings` array -- ONLY genuinely new findings
- Add current iteration number to `iterations_touched` array
- If a question is IRRELEVANT, REMOVE it from the array entirely
- Do NOT add new questions -- work through the original plan
- Write the COMPLETE updated plan.json (not a partial update)

**gaps.md** -- Rewrite entirely with current state:
- "## Open Questions" -- remaining questions with confidence levels
- "## Contradictions" -- any conflicting information discovered
- "## Last Updated" -- iteration number and timestamp

**synthesis.md** -- Update findings for the question you worked on:
- Keep "## Findings by Question" structure with all existing questions
- Add new findings under the relevant question heading (do NOT duplicate existing findings)
- Include question status and confidence in the heading
- Do NOT remove or modify findings for other questions

**state.json** -- Update ONLY these fields:
- `overall_confidence` -- average of all remaining questions' confidence values
- Do NOT change `iteration` or `phase` (oracle.sh manages these)

### Step 5: Assess and Complete
State your assessment: "Confidence: X% -- {brief reason}"

If overall_confidence >= target_confidence (from state.json) OR all remaining questions are "answered":
Output `<oracle>COMPLETE</oracle>`
Otherwise, end normally for another iteration.

## Confidence Scoring Rubric

| Score | Meaning | Evidence Required |
|-------|---------|-------------------|
| 0-20% | Unexplored | No research conducted |
| 20-40% | Surface level | General information, no specifics |
| 40-60% | Partial understanding | Specific details from 1-2 sources |
| 60-80% | Good understanding | Multiple sources agree, edge cases known |
| 80-95% | Thorough | Primary sources verified, contradictions resolved |
| 95-100% | Exhaustive | All reasonable angles explored, high-quality sources |

Do NOT inflate confidence. One blog post = 30%, not 70%.
Do NOT deflate confidence to keep research going. Score honestly.

## Important Rules
- Target ONE question per iteration
- Write COMPLETE JSON files, not partial updates
- Do NOT add new sub-questions
- Remove irrelevant questions entirely
- Reference existing findings BEFORE writing new ones -- no restatements
- Do NOT modify code files or colony state
- Only write to `.aether/oracle/` directory
```

### Example 5: Test Pattern for Phase Transitions

```javascript
// Source: New test file for Phase 7 -- tests/unit/oracle-phase-transitions.test.js
const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const ORACLE_SH = path.join(__dirname, '../../.aether/oracle/oracle.sh');
const PROJECT_ROOT = path.join(__dirname, '../..');

function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-phase-'));
}

function writeState(dir, overrides = {}) {
  const state = {
    version: '1.0', topic: 'Test', scope: 'web', phase: 'survey',
    iteration: 0, max_iterations: 15, target_confidence: 95,
    overall_confidence: 0, started_at: '2026-01-01T00:00:00Z',
    last_updated: '2026-01-01T00:00:00Z', status: 'active',
    ...overrides
  };
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
  return state;
}

function writePlan(dir, questions) {
  const plan = {
    version: '1.0', questions,
    created_at: '2026-01-01T00:00:00Z', last_updated: '2026-01-01T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
  return plan;
}

// Test: survey -> investigate when all questions touched
test('phase transition: survey to investigate when all questions touched', t => {
  const dir = createTmpDir();
  t.teardown(() => fs.rmSync(dir, { recursive: true, force: true }));

  writeState(dir, { phase: 'survey' });
  writePlan(dir, [
    { id: 'q1', text: 'Q1?', status: 'partial', confidence: 30, key_findings: ['f1'], iterations_touched: [1] },
    { id: 'q2', text: 'Q2?', status: 'partial', confidence: 20, key_findings: ['f1'], iterations_touched: [2] }
  ]);

  // Source the function and test (bash function extracted for testing)
  const result = execSync(
    `bash -c 'source "${ORACLE_SH}" --source-only 2>/dev/null; determine_phase "${dir}/state.json" "${dir}/plan.json"'`,
    { encoding: 'utf8', cwd: PROJECT_ROOT }
  ).trim();

  t.is(result, 'investigate');
});
```

### Example 6: Bash Integration Test for Iteration Counter

```bash
# Source: New test for Phase 7 -- tests/bash/test-oracle-phase.sh
#!/bin/bash
# Tests for oracle phase transitions and iteration counter

test_iteration_counter_increments() {
  local tmp_dir
  tmp_dir=$(mktemp -d)

  # Create state.json with iteration 0
  cat > "$tmp_dir/state.json" <<'EOF'
{"version":"1.0","topic":"Test","scope":"web","phase":"survey","iteration":0,"max_iterations":15,"target_confidence":95,"overall_confidence":0,"started_at":"2026-01-01T00:00:00Z","last_updated":"2026-01-01T00:00:00Z","status":"active"}
EOF

  # Simulate iteration increment (same logic as oracle.sh)
  local ts
  ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  jq --arg ts "$ts" '.iteration += 1 | .last_updated = $ts' "$tmp_dir/state.json" > "$tmp_dir/state.json.tmp" && mv "$tmp_dir/state.json.tmp" "$tmp_dir/state.json"

  local iter
  iter=$(jq '.iteration' "$tmp_dir/state.json")

  if [[ "$iter" -eq 1 ]]; then
    test_pass "iteration counter increments from 0 to 1"
  else
    test_fail "iteration counter: expected 1, got $iter"
  fi

  rm -rf "$tmp_dir"
}
```

## File Touch Map

```
MODIFY:
  .aether/oracle/oracle.sh              # Add: iteration increment, phase transition logic
                                         #       (determine_phase function), prompt construction
                                         #       (build_oracle_prompt function), updated AI invocation
  .aether/oracle/oracle.md              # Rewrite: phase-aware instructions, confidence rubric,
                                         #          depth enforcement rules, phase-specific targeting

CREATE:
  tests/unit/oracle-phase-transitions.test.js  # Ava tests for phase transition logic
  tests/bash/test-oracle-phase.sh              # Bash tests for iteration counter and phase transitions

NO CHANGE NEEDED:
  .aether/aether-utils.sh               # Phase enum already validated (survey/investigate/synthesize/verify)
  .claude/commands/ant/oracle.md         # Wizard already writes state.json with phase="survey"
  .opencode/commands/ant/oracle.md       # Same -- no changes needed
  tests/unit/oracle-state.test.js        # Existing validation tests still valid
  tests/bash/test-oracle-state.sh        # Existing session tests still valid
```

## Discretion Recommendations

Based on research into the codebase, prompt engineering best practices (Anthropic's context engineering guide), and the Ralph/RALF pattern, here are recommendations for the discretion areas:

| Area | Recommendation | Rationale |
|------|---------------|-----------|
| Research lifecycle phases | Four stages: survey (broad), investigate (deep), synthesize (connect), verify (validate) | Maps to proven research methodology; Anthropic's own research system uses "broad then progressively narrow" |
| Phase transition triggers | Structural conditions in oracle.sh (touched count, avg confidence thresholds) | AI self-assessment unreliable; structural metrics are objective and measurable |
| Survey->investigate threshold | All questions touched OR avg confidence >= 25% | Low threshold ensures survey doesn't stall; touching all questions ensures breadth before depth |
| Investigate->synthesize threshold | Avg confidence >= 60% OR fewer than 2 questions below 50% | 60% means solid understanding of most areas; ready to connect findings |
| Synthesize->verify threshold | Avg confidence >= 80% | High bar ensures synthesis is meaningful before verification |
| Gap targeting strategy | Lowest-confidence non-answered question, refined per phase (survey prefers untouched) | Simple, deterministic, measurable; phase refinement adds intelligence without complexity |
| Confidence scoring | Explicit rubric (0-20 unexplored, 20-40 surface, 40-60 partial, 60-80 good, 80-95 thorough, 95-100 exhaustive) | Anchors AI scoring to evidence quality; reduces inflation |
| State update format | AI writes COMPLETE JSON files using Write tool; oracle.sh validates and increments iteration | Prevents corruption from partial updates; separates AI content from structural bookkeeping |
| Depth enforcement | Prompt instruction: "read existing findings first, write ONLY new information" | Simpler and more reliable than algorithmic repetition detection |
| --json-schema usage | Do NOT use in Phase 7; defer to Phase 8 if needed | --json-schema constrains response text, not file writes; adds complexity without solving the core problem |
| Prompt architecture | Phase directive injected by oracle.sh at top of oracle.md base prompt | Keeps oracle.md simple; gives oracle.sh control; avoids AI needing to "choose" behavior |

## Open Questions

1. **Phase transition thresholds may need empirical tuning**
   - What we know: 25% / 60% / 80% thresholds are informed by the confidence rubric but not tested with real research sessions
   - What's unclear: Whether these thresholds produce good phase timing in practice
   - Recommendation: Use these values as defaults; Phase 8 (convergence detection) will refine them based on empirical data. The thresholds are easy to change in oracle.sh.

2. **Whether determine_phase should be a bash function in oracle.sh or a subcommand in aether-utils.sh**
   - What we know: Phase transition logic is tightly coupled to the oracle loop; aether-utils.sh is the home for reusable subcommands
   - What's unclear: Whether other commands will need phase transition logic
   - Recommendation: Start as a function in oracle.sh. If Phase 8 or Phase 10 needs it, extract to aether-utils.sh then. YAGNI.

3. **Whether --source-only flag in oracle.sh is practical for unit testing**
   - What we know: Sourcing oracle.sh to test individual functions requires the script to support partial loading
   - What's unclear: Whether oracle.sh's set -e and existing structure allow clean partial sourcing
   - Recommendation: Extract determine_phase into a separate testable script (e.g., oracle-phase.sh) or test via integration tests that invoke the full loop with mocked AI. Simpler approach: test the jq logic directly in bash tests without sourcing oracle.sh.

## State of the Art

| Old Approach (Phase 6) | Phase 7 Approach | Impact |
|------------------------|------------------|--------|
| Single-mode oracle.md prompt | Phase-aware prompt with 4 behavioral modes | AI behavior changes appropriately across research lifecycle |
| No phase transitions | Structural condition-based phase transitions in oracle.sh | Research progresses through survey -> investigate -> synthesize -> verify predictably |
| No iteration counter update | oracle.sh increments state.json.iteration | state.json accurately reflects research progress |
| Generic "target lowest confidence" | Phase-specific targeting (survey: untouched first; investigate: lowest confidence deep dive; synthesize: cross-question connections; verify: claim validation) | Each iteration type serves a distinct purpose |
| No depth enforcement | Prompt requires referencing existing findings before writing new ones | Reduces restatement; forces genuine deepening |
| No confidence rubric | Explicit scoring rubric with evidence requirements | More consistent, less inflated confidence scores |

**Deprecated/outdated after this phase:**
- The Phase 6 oracle.md prompt (static, single-mode) is replaced by the phase-aware version
- The "Do NOT change `phase` (Phase 7 will manage phase transitions)" comment in oracle.md is removed -- Phase 7 IS here

## Sources

### Primary (HIGH confidence)
- `.aether/oracle/oracle.md` -- Current prompt (69 lines) written in Phase 6 -- the baseline being enhanced
- `.aether/oracle/oracle.sh` -- Current orchestrator (172 lines) -- the loop being extended
- `.aether/aether-utils.sh` lines 1203-1274 -- validate-oracle-state with phase enum (survey/investigate/synthesize/verify) already validated
- `.claude/commands/ant/oracle.md` -- Wizard that creates initial state.json with phase="survey"
- `.planning/phases/06-state-architecture-foundation/06-VERIFICATION.md` -- Phase 6 verification confirming all state infrastructure in place
- `.planning/REQUIREMENTS.md` -- LOOP-02, LOOP-03, INTL-02, INTL-03 requirement definitions
- Claude CLI `--help` output -- Confirms `--json-schema` flag exists with JSON Schema validation
- OpenCode CLI `run --help` output -- Confirms `--json-schema` is NOT available in opencode

### Secondary (MEDIUM confidence)
- [Anthropic: Effective Context Engineering for AI Agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) -- "Start broad, progressively narrow focus" pattern; structured note-taking for persistent memory; just-in-time context retrieval
- [Anthropic: How we built our multi-agent research system](https://www.anthropic.com/engineering/multi-agent-research-system) -- Task decomposition prevents overlap; agents assess gaps after tool results; breadth-first then depth investigation
- [Ralph (snarktank/ralph)](https://github.com/snarktank/ralph) -- RALF pattern: pick highest-priority incomplete story, implement, update progress, capture learnings for future iterations

### Tertiary (LOW confidence)
- Phase transition thresholds (25%/60%/80%) -- Informed by rubric mapping but not empirically validated; flagged for tuning in Phase 8
- Claude CLI `--json-schema` interaction with tool-using agents -- Documentation confirms it constrains response text, but the exact interaction with agents that use Write tool during execution needs verification

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies; all tools already in the project
- Architecture (prompt design): HIGH -- based on existing oracle.md, Anthropic's published patterns, and Phase 6 infrastructure
- Architecture (phase transitions): MEDIUM -- transition thresholds are educated defaults, not empirically validated
- Pitfalls: HIGH -- identified from reading actual code (missing iteration counter, OpenCode parity, prompt length concerns)

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (stable domain; prompt engineering patterns don't change rapidly)
