# Phase 6: State Architecture Foundation - Research

**Researched:** 2026-03-13
**Domain:** JSON state file design, bash shell scripting (oracle.sh, oracle.md, aether-utils.sh), jq validation, markdown generation
**Confidence:** HIGH

## Summary

Phase 6 replaces the oracle's current flat-append model (progress.md is appended to, research.json is static) with four structured state files (state.json, plan.json, gaps.md, synthesis.md) plus a human-readable research-plan.md generated from plan.json. The current oracle loop already has: a bash orchestrator (oracle.sh, 134 lines) that spawns stateless AI iterations reading oracle.md as a prompt, topic archiving, session freshness detection via aether-utils.sh, and a stop-signal mechanism. The work is replacing the data layer these iterations read/write -- not changing the loop mechanics.

The existing codebase has strong patterns to follow: validate-state in aether-utils.sh uses inline jq schema checks (type assertions per field), atomic_write provides corruption-safe file writes, session-verify-fresh/session-clear already handle oracle files (currently checking progress.md and research.json -- these need updating to the new file set), and the test suite uses ava for unit/integration tests plus bash test scripts for shell utilities. All existing patterns should be reused.

The key insight from the Ralph pattern (which oracle is based on) is that each iteration is a fresh context that reads state files, does work, and writes updated state files. Phase 6 defines what those state files look like and ensures they are valid JSON. Phase 7 (prompt engineering) will define how iterations read and update them. Phase 8 (orchestrator) will add convergence detection. This phase must design schemas that are forward-compatible with those downstream needs -- particularly confidence scoring fields (Phase 7 uses per-question 0-100%), gap tracking (Phase 7 updates gaps.md each iteration), and convergence metrics (Phase 8 reads structural metrics from state.json).

**Primary recommendation:** Define JSON schemas for state.json and plan.json, create aether-utils.sh subcommands for oracle state CRUD and jq validation, update oracle.md to read/write the new files, update session-verify-fresh/session-clear to reference the new file set, and write tests that verify jq validation catches malformed state.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Executive summary style for research-plan.md -- big picture only (topic, overall status, key findings, what's next)
- Not a detailed dashboard -- a few lines the user can scan quickly
- research-plan.md is the human-readable entry point
- Initial decomposition is mostly fixed -- oracle sets up the plan at the start and sticks to it
- Oracle should NOT keep adding new sub-questions mid-research; it works through the original plan
- If a sub-question turns out to be irrelevant, remove it from the plan entirely (don't leave it marked as skipped)
- Questions that don't produce useful results should be cleaned out, not accumulated
- State files stay in `.aether/oracle/` -- consistent with existing oracle location
- Previous research sessions are archived (e.g., `oracle/archive/`) so past research is recoverable
- New sessions overwrite active state files, but old sessions are preserved in archive

### Claude's Discretion
- Update frequency for research-plan.md (every iteration vs key moments)
- What to emphasize in progress view (findings vs gaps vs both)
- Whether to show the oracle's planned next move
- Confidence representation (labels vs percentages vs hybrid)
- Whether to include an overall progress indicator
- Sub-question granularity (3-4 broad vs 6-8 detailed) -- adapt to topic complexity
- Flat vs hierarchical sub-question structure
- Sub-question status levels (binary vs three-state)
- Whether to flag contradictory source information visibly
- File split between the 4 state files and research-plan.md
- Whether research-plan.md serves as a single overview or information stays distributed

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOOP-01 | Oracle uses structured state files (state.json, plan.json, gaps.md, synthesis.md) to bridge context between stateless iterations -- not flat progress.md append | This phase defines the schemas, creates the files, and wires oracle.md to read/write them instead of appending to progress.md. The validate-state pattern in aether-utils.sh provides the validation blueprint. |
| INTL-01 | Oracle decomposes topic into 3-8 tracked sub-questions with status (open/partial/answered) | plan.json stores the sub-question array with id, text, status, and confidence fields. The wizard (oracle.md command) populates the initial decomposition. Three-state status (open/partial/answered) per user decision flexibility. |
| INTL-04 | Research plan visible as research-plan.md showing questions, status, confidence, and next steps | research-plan.md is generated from plan.json as an executive summary. A bash function or jq pipeline transforms plan.json into scannable markdown. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | All oracle state CRUD operations via subcommands | Single source of truth for state operations; validate-state, session-verify-fresh patterns already exist |
| jq | 1.8.1 (system) | JSON creation, validation, transformation, query | Used throughout aether-utils.sh; the inline schema-check pattern (chk/opt functions) is the project standard |
| oracle.sh | 134 lines | Bash loop orchestrator spawning AI iterations | Existing file; needs minimal changes (file references, archive list) |
| oracle.md | Prompt file | Instructions for each AI iteration | Existing file; needs rewrite to read/write new state files instead of appending progress.md |
| ava | ^6.0.0 | Unit/integration test runner | Project standard; all 490+ tests use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| atomic_write | .aether/utils/atomic-write.sh | Corruption-safe file writes (temp + rename) | When writing state.json and plan.json from aether-utils.sh subcommands |
| file-lock.sh | .aether/utils/file-lock.sh | File locking primitives | If concurrent access to oracle state files is a concern (likely not for Phase 6, but available) |
| bash test framework | tests/bash/test-helpers.sh | Bash-level test helpers (test_start, test_pass, test_fail) | For testing bash subcommands that validate oracle state |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| jq inline validation | JSON Schema (ajv) | ajv requires node; jq is already available everywhere and matches project pattern |
| Separate validate-oracle-state subcommand | Extending existing validate-state | Separate is cleaner; validate-state is already complex; oracle state is a different domain |
| Markdown-only state (no JSON) | Pure JSON | JSON is machine-readable and jq-validatable; markdown is for humans only. Both needed. |

## Architecture Patterns

### Recommended File Layout
```
.aether/oracle/
  state.json              # Session metadata: topic, iteration count, phase, timestamps
  plan.json               # Sub-questions with status/confidence -- THE core structured state
  gaps.md                 # Human-readable knowledge gaps (markdown, not JSON)
  synthesis.md            # Accumulated findings organized by sub-question
  research-plan.md        # GENERATED from plan.json -- user-facing summary
  research.json           # REPLACED by state.json (wizard writes config here today)
  progress.md             # REPLACED by synthesis.md + gaps.md (flat append eliminated)
  oracle.md               # Prompt file (updated to reference new state files)
  oracle.sh               # Loop orchestrator (updated file references)
  archive/                # Archived previous sessions
    2026-03-13-143000/    # Timestamped folder per session
      state.json
      plan.json
      gaps.md
      synthesis.md
      research-plan.md
```

### Pattern 1: state.json Schema
**What:** Session-level metadata and iteration tracking
**When to use:** Read at start of every iteration; updated after every iteration
**Schema:**
```json
{
  "version": "1.0",
  "topic": "string - the research topic",
  "scope": "codebase|web|both",
  "phase": "survey|investigate|synthesize|verify",
  "iteration": 0,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 0,
  "started_at": "ISO-8601 UTC",
  "last_updated": "ISO-8601 UTC",
  "status": "active|complete|stopped"
}
```
**Notes:** The `phase` field (survey/investigate/synthesize/verify) is defined here but not used until Phase 7 (prompt engineering). Phase 6 sets it to "survey" as default. The `overall_confidence` is a derived field (average of sub-question confidences from plan.json) that Phase 7 will compute; Phase 6 initializes it to 0. The `iteration` counter is incremented by oracle.sh after each AI invocation.

### Pattern 2: plan.json Schema
**What:** Sub-question decomposition with per-question tracking
**When to use:** Created at session start; read and updated each iteration
**Schema:**
```json
{
  "version": "1.0",
  "questions": [
    {
      "id": "q1",
      "text": "What is X?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    }
  ],
  "created_at": "ISO-8601 UTC",
  "last_updated": "ISO-8601 UTC"
}
```
**Field details:**
- `id`: Short identifier (q1, q2, etc.)
- `text`: The sub-question text
- `status`: "open" | "partial" | "answered" (three-state per user discretion area; recommended over binary)
- `confidence`: 0-100 integer (Phase 7 will use this to prioritize; Phase 6 initializes to 0)
- `key_findings`: Array of strings -- brief findings discovered for this question (Phase 7 populates)
- `iterations_touched`: Array of iteration numbers that worked on this question (Phase 8 uses for convergence)

**Constraint compliance:** User decided sub-questions are mostly fixed at start. The schema allows removal (delete from array) but not addition. Oracle prompt (oracle.md) will instruct: "Do not add new questions. If a question is irrelevant, remove it entirely."

### Pattern 3: gaps.md Structure
**What:** Human-readable list of remaining knowledge gaps
**When to use:** Updated each iteration; read by next iteration to target gaps
**Structure:**
```markdown
# Knowledge Gaps

## Open Questions
- [q1] What specific mechanism does X use? (confidence: 20%)
- [q3] How does Y interact with Z? (confidence: 45%)

## Contradictions
- Source A says X, but source B says Y (relates to q2)

## Last Updated
Iteration 3 -- 2026-03-13T14:30:00Z
```
**Notes:** Markdown not JSON -- this file is for the AI to read as context and for users to inspect. Phase 7 will formalize how iterations update it. Phase 6 creates the initial empty structure.

### Pattern 4: synthesis.md Structure
**What:** Accumulated research findings organized by sub-question
**When to use:** Appended to each iteration (but organized by question, not flat chronological append)
**Structure:**
```markdown
# Research Synthesis

## Topic
[topic from state.json]

## Findings by Question

### q1: What is X?
**Status:** partial | **Confidence:** 45%

- Finding from iteration 1: ...
- Finding from iteration 3: ...

### q2: How does Y work?
**Status:** answered | **Confidence:** 85%

- Finding from iteration 2: ...

## Last Updated
Iteration 3 -- 2026-03-13T14:30:00Z
```
**Notes:** Unlike current progress.md (pure chronological append), synthesis.md organizes by question. This is the structured replacement. Phase 7 defines how AI iterations write to it; Phase 6 creates the initial empty structure.

### Pattern 5: research-plan.md Generation
**What:** Human-facing executive summary generated from plan.json
**When to use:** Generated after plan.json creation and after each iteration (or at key moments -- discretionary)
**Structure:**
```markdown
# Research Plan

**Topic:** [topic]
**Status:** [active/complete] | **Iteration:** [N] of [max]
**Overall Confidence:** [X]%

## Questions
| # | Question | Status | Confidence |
|---|----------|--------|------------|
| 1 | What is X? | open | 0% |
| 2 | How does Y work? | partial | 45% |
| 3 | Why does Z matter? | answered | 90% |

## Next Steps
[Brief description of what the oracle will investigate next -- derived from lowest-confidence open question]

---
*Generated from plan.json -- do not edit directly*
```
**Notes:** This is a GENERATED file. Users read it; the oracle writes it. The "Next Steps" section satisfies the user's desire for an executive summary showing "what's next." Recommend regenerating after every iteration -- the cost is negligible (a jq pipeline + printf) and the user benefits from always-current information.

### Pattern 6: validate-oracle-state Subcommand
**What:** New aether-utils.sh subcommand for oracle state validation
**When to use:** After file creation, after simulated updates, in tests
**Implementation pattern (following validate-state colony):**
```bash
validate-oracle-state)
  # Validate oracle state files using jq type checks
  case "${1:-}" in
    state)
      jq '
        def chk(f;t): ...same pattern as validate-state...;
        {file:"state.json", checks:[
          chk("version";["string"]),
          chk("topic";["string"]),
          chk("scope";["string"]),
          chk("phase";["string"]),
          chk("iteration";["number"]),
          chk("max_iterations";["number"]),
          chk("target_confidence";["number"]),
          chk("overall_confidence";["number"]),
          chk("started_at";["string"]),
          chk("status";["string"])
        ]} | . + {pass: ...}
      ' "$ORACLE_DIR/state.json"
      ;;
    plan)
      jq '
        def chk(f;t): ...;
        {file:"plan.json", checks:[
          chk("version";["string"]),
          chk("questions";["array"])
        ] + [
          if (.questions | all(has("id","text","status","confidence")))
          then "pass"
          else "fail: questions missing required fields"
          end
        ]} | . + {pass: ...}
      ' "$ORACLE_DIR/plan.json"
      ;;
    all) ... ;;
  esac
  ;;
```

### Pattern 7: Archive on New Session
**What:** Archive current state files before starting a new session
**When to use:** When oracle wizard starts and existing state files are present
**Implementation:** Extend the existing archive pattern in oracle.sh (lines 50-68) to copy all new state files:
```bash
ARCHIVE_FOLDER="$ARCHIVE_DIR/$(date +%Y-%m-%d-%H%M%S)"
mkdir -p "$ARCHIVE_FOLDER"
for f in state.json plan.json gaps.md synthesis.md research-plan.md; do
  [ -f "$SCRIPT_DIR/$f" ] && cp "$SCRIPT_DIR/$f" "$ARCHIVE_FOLDER/"
done
```

### Anti-Patterns to Avoid
- **Flat chronological append:** The current progress.md pattern. Replaced by question-organized synthesis.md. Never go back to "## Iteration N" headers in a single file.
- **Self-assessed confidence as sole metric:** state.json has overall_confidence but Phase 8 adds structural metrics. Do not design the schema assuming confidence alone drives completion.
- **Expanding question set mid-research:** User explicitly locked this. plan.json should shrink (questions removed) or stay stable, never grow.
- **Skipped/irrelevant status:** User said remove irrelevant questions, not mark them skipped. No "skipped" status value.
- **Overloading a single file:** Previous approach put everything in progress.md. The split into 4 files (state.json for metadata, plan.json for structure, gaps.md for unknowns, synthesis.md for findings) is intentional.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| JSON validation | Custom node.js validator | jq inline schema checks (validate-state pattern) | Project standard; no extra dependencies; works in bash |
| Atomic file writes | echo > file | atomic_write from .aether/utils/atomic-write.sh | Prevents corruption on crash/interrupt; existing utility |
| File freshness checking | Custom timestamp comparison | session-verify-fresh subcommand (already handles oracle) | Existing utility; just update the required_docs list |
| Session cleanup | Manual rm commands | session-clear subcommand (already handles oracle) | Existing utility; just update the files list |
| Markdown table generation | String concatenation in bash | jq + printf pipeline | jq can transform plan.json directly to markdown table rows |

**Key insight:** Almost all the infrastructure exists. The work is defining schemas, writing a new validate subcommand, updating file references in 3-4 existing locations, and writing tests.

## Common Pitfalls

### Pitfall 1: Forgetting to Update session-verify-fresh and session-clear
**What goes wrong:** The oracle session management still looks for progress.md and research.json instead of the new state files
**Why it happens:** These are in aether-utils.sh (lines 6488-6658), far from the oracle directory
**How to avoid:** Update the `required_docs` string at line 6490 (verify-fresh) and the `files` string at line 6604 (session-clear) to reference state.json, plan.json, gaps.md, synthesis.md, research-plan.md
**Warning signs:** `/ant:oracle status` reports "no session" when state files exist; `--force` doesn't clear new files

### Pitfall 2: Breaking the Wizard (oracle.md command)
**What goes wrong:** The /ant:oracle slash command still writes research.json and progress.md
**Why it happens:** oracle.md (the slash command at .claude/commands/ant/oracle.md) has hardcoded file references at Steps 2 and 2.5
**How to avoid:** Update the wizard to write state.json and plan.json instead. research-plan.md should be generated from plan.json immediately after creation. Update Step 0c (status display) to read the new files.
**Warning signs:** New state files never get created; old research.json still appears

### Pitfall 3: Invalid JSON After AI Iteration Updates
**What goes wrong:** AI writes malformed JSON to state.json or plan.json, corrupting the session
**Why it happens:** AI models sometimes produce invalid JSON, especially when asked to modify complex nested structures
**How to avoid:** The oracle.md prompt (Step 4) should instruct the AI to write complete valid JSON, not partial updates. Phase 8 adds recovery, but Phase 6 should include jq validation in oracle.sh after each iteration as a safety check.
**Warning signs:** `jq . state.json` fails; subsequent iterations read garbage

### Pitfall 4: Archive Directory Growth
**What goes wrong:** Archive directory grows unbounded with every session
**Why it happens:** Every new session archives all state files; no cleanup policy
**How to avoid:** Not a Phase 6 concern (existing behavior), but be aware. The current archive already has this pattern (line 55-67 in oracle.sh). Keep the same approach.
**Warning signs:** Large .aether/oracle/archive/ directory

### Pitfall 5: research-plan.md Drift from plan.json
**What goes wrong:** research-plan.md shows stale data because it wasn't regenerated after plan.json updated
**Why it happens:** If generation is only at "key moments" instead of every iteration, the file falls behind
**How to avoid:** Recommend generating after every iteration. The cost is trivial (one jq command + printf). Always include the footer "Generated from plan.json -- do not edit directly" to signal the file is derived.
**Warning signs:** User reads research-plan.md and sees different data than what the oracle is actually working on

### Pitfall 6: OpenCode Command Parity
**What goes wrong:** .claude/commands/ant/oracle.md is updated but .opencode/commands/ant/oracle.md is not
**Why it happens:** This project maintains command parity between Claude Code and OpenCode
**How to avoid:** Update both files. The lint:sync script (bin/generate-commands.sh check) may catch structural drift.
**Warning signs:** `npm run lint:sync` fails

## Code Examples

### Example 1: Creating Initial state.json (bash/jq)
```bash
# Source: validate-state pattern in aether-utils.sh lines 1161-1176
ORACLE_DIR=".aether/oracle"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

jq -n \
  --arg topic "$TOPIC" \
  --arg scope "$SCOPE" \
  --argjson max_iter "$MAX_ITERATIONS" \
  --argjson target_conf "$TARGET_CONFIDENCE" \
  --arg ts "$TIMESTAMP" \
  '{
    version: "1.0",
    topic: $topic,
    scope: $scope,
    phase: "survey",
    iteration: 0,
    max_iterations: $max_iter,
    target_confidence: $target_conf,
    overall_confidence: 0,
    started_at: $ts,
    last_updated: $ts,
    status: "active"
  }' > "$ORACLE_DIR/state.json"
```

### Example 2: Creating Initial plan.json from Questions Array
```bash
# Source: research.json creation pattern in oracle.md Step 2
# Questions come from the wizard (currently written as research.json)
jq -n \
  --argjson questions "$QUESTIONS_JSON_ARRAY" \
  --arg ts "$TIMESTAMP" \
  '{
    version: "1.0",
    questions: [
      $questions[] | {
        id: ("q" + (. as $q | $questions | to_entries[] | select(.value == $q) | .key + 1 | tostring)),
        text: .,
        status: "open",
        confidence: 0,
        key_findings: [],
        iterations_touched: []
      }
    ],
    created_at: $ts,
    last_updated: $ts
  }' > "$ORACLE_DIR/plan.json"
```

### Example 3: Generating research-plan.md from plan.json
```bash
# Source: project pattern of jq-to-markdown pipelines
generate_research_plan() {
  local oracle_dir="${1:-.aether/oracle}"
  local state_file="$oracle_dir/state.json"
  local plan_file="$oracle_dir/plan.json"
  local output_file="$oracle_dir/research-plan.md"

  local topic iteration max_iter confidence status
  topic=$(jq -r '.topic' "$state_file")
  iteration=$(jq -r '.iteration' "$state_file")
  max_iter=$(jq -r '.max_iterations' "$state_file")
  confidence=$(jq -r '.overall_confidence' "$state_file")
  status=$(jq -r '.status' "$state_file")

  {
    echo "# Research Plan"
    echo ""
    echo "**Topic:** $topic"
    echo "**Status:** $status | **Iteration:** $iteration of $max_iter"
    echo "**Overall Confidence:** ${confidence}%"
    echo ""
    echo "## Questions"
    echo "| # | Question | Status | Confidence |"
    echo "|---|----------|--------|------------|"
    jq -r '.questions[] | "| \(.id) | \(.text) | \(.status) | \(.confidence)% |"' "$plan_file"
    echo ""
    echo "## Next Steps"
    # Show lowest-confidence open question as the planned next move
    local next
    next=$(jq -r '[.questions[] | select(.status != "answered")] | sort_by(.confidence) | first | .text // "All questions answered"' "$plan_file")
    echo "Next investigation: $next"
    echo ""
    echo "---"
    echo "*Generated from plan.json -- do not edit directly*"
  } > "$output_file"
}
```

### Example 4: jq Validation for plan.json (test-ready)
```bash
# Source: validate-state colony pattern at aether-utils.sh line 1161
validate_plan_json() {
  local plan_file="${1:-.aether/oracle/plan.json}"
  jq '
    def chk(f;t): if has(f) then (if (.[f]|type) as $a | t | any(. == $a) then "pass" else "fail: \(f) is \(.[f]|type), expected \(t|join("|"))" end) else "fail: missing \(f)" end;
    {file:"plan.json", checks:[
      chk("version";["string"]),
      chk("questions";["array"]),
      chk("created_at";["string"]),
      chk("last_updated";["string"]),
      if (.questions | length) >= 1 and (.questions | length) <= 8
        then "pass"
        else "fail: questions count \(.questions | length) outside 1-8 range"
      end,
      if (.questions | all(has("id","text","status","confidence")))
        then "pass"
        else "fail: questions missing required fields (id, text, status, confidence)"
      end,
      if (.questions | all(.status == "open" or .status == "partial" or .status == "answered"))
        then "pass"
        else "fail: invalid status value (must be open|partial|answered)"
      end,
      if (.questions | all(.confidence >= 0 and .confidence <= 100))
        then "pass"
        else "fail: confidence out of 0-100 range"
      end
    ]} | . + {pass: (([.checks[] | select(. == "pass")] | length) == (.checks | length))}
  ' "$plan_file"
}
```

### Example 5: Ava Test Pattern for Oracle State Validation
```javascript
// Source: tests/unit/validate-state.test.js pattern
const test = require('ava');
const { execSync } = require('child_process');
const fs = require('fs');
const os = require('os');
const path = require('path');

const AETHER_UTILS_PATH = path.join(__dirname, '../../.aether/aether-utils.sh');

function createOracleTestDir(stateContent, planContent) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-'));
  if (stateContent) fs.writeFileSync(path.join(tmpDir, 'state.json'), JSON.stringify(stateContent, null, 2));
  if (planContent) fs.writeFileSync(path.join(tmpDir, 'plan.json'), JSON.stringify(planContent, null, 2));
  return tmpDir;
}

test('validate-oracle-state plan detects missing required fields', t => {
  const tmpDir = createOracleTestDir(null, { version: "1.0", questions: [{ id: "q1" }] });
  t.teardown(() => fs.rmSync(tmpDir, { recursive: true, force: true }));

  const result = JSON.parse(execSync(
    `ORACLE_DIR="${tmpDir}" bash "${AETHER_UTILS_PATH}" validate-oracle-state plan`,
    { encoding: 'utf8' }
  ));

  t.false(result.result.pass, 'Should fail when question missing text/status/confidence');
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Flat progress.md append | Structured state files (state.json, plan.json, gaps.md, synthesis.md) | Phase 6 (this phase) | AI iterations can target gaps instead of re-reading entire history |
| Single research.json config | state.json with runtime metadata | Phase 6 (this phase) | Iteration count, phase, confidence tracked machine-readably |
| Questions in research.json (static) | plan.json with per-question status/confidence | Phase 6 (this phase) | Enables gap-driven iteration (Phase 7) and convergence detection (Phase 8) |
| No user-facing progress view | research-plan.md generated from plan.json | Phase 6 (this phase) | User can read executive summary at any time |

**Deprecated/outdated after this phase:**
- `progress.md`: Replaced by synthesis.md + gaps.md. Remove from session-verify-fresh and session-clear.
- `research.json`: Replaced by state.json + plan.json. Remove from session-verify-fresh and session-clear.
- `.last-topic` file: Topic is now in state.json. The comparison logic in oracle.sh can read state.json directly.

## File Touch Map

```
MODIFY:
  .aether/oracle/oracle.sh                  # Update file references, archive list, add jq validation after iteration
  .aether/oracle/oracle.md                  # Rewrite to read/write new state files instead of appending progress.md
  .aether/aether-utils.sh                   # Add validate-oracle-state subcommand; update session-verify-fresh and session-clear oracle file lists
  .claude/commands/ant/oracle.md            # Update wizard (Step 2) to write state.json + plan.json + generate research-plan.md; update Step 0c status display
  .opencode/commands/ant/oracle.md          # Mirror changes from .claude/commands/ant/oracle.md (parity requirement)

CREATE:
  .aether/oracle/state.json                 # Created by wizard on session start (not committed -- .gitignore)
  .aether/oracle/plan.json                  # Created by wizard on session start (not committed)
  .aether/oracle/gaps.md                    # Created by wizard with empty structure (not committed)
  .aether/oracle/synthesis.md               # Created by wizard with empty structure (not committed)
  .aether/oracle/research-plan.md           # Generated from plan.json (not committed)
  tests/unit/oracle-state.test.js           # Ava tests for validate-oracle-state subcommand
  tests/bash/test-oracle-state.sh           # Bash tests for oracle state creation and validation

NO CHANGE NEEDED:
  .aether/utils/atomic-write.sh             # Already exists, used as-is
  .aether/utils/file-lock.sh                # Already exists, available if needed
  tests/unit/oracle-regression.test.js      # Existing regression tests; still valid
  tests/bash/test-session-freshness.sh      # May need new test cases but core framework unchanged
```

## Discretion Recommendations

Based on research into the codebase patterns and user constraints, here are recommendations for the discretion areas:

| Area | Recommendation | Rationale |
|------|---------------|-----------|
| Update frequency for research-plan.md | Every iteration | Cost is negligible (one jq pipeline); user always sees current state; avoids drift pitfall |
| What to emphasize in progress | Both findings and gaps | The table shows status/confidence per question (covers both); executive summary style keeps it scannable |
| Show oracle's planned next move | Yes -- one line at bottom | User said "what's next" is part of the executive summary; lowest-confidence open question is the natural next target |
| Confidence representation | Percentages (0-100%) | Matches Phase 7 per-question scoring (INTL-03); simple to compute and display; jq handles arithmetic natively |
| Overall progress indicator | Yes -- "Iteration N of M" plus "Overall Confidence: X%" | Two numbers the user can scan instantly; both are in state.json |
| Sub-question granularity | Adaptive 3-8, recommend 4-6 for most topics | User said "adapt to topic complexity"; the wizard should generate based on topic breadth |
| Flat vs hierarchical questions | Flat | Hierarchical adds schema complexity with minimal benefit at 3-8 questions; downstream phases don't need hierarchy |
| Status levels | Three-state: open/partial/answered | Matches INTL-01 requirement verbatim; "partial" is meaningful (some findings but not confident) |
| Flag contradictions | Yes, in gaps.md "Contradictions" section | Lightweight; does not clutter research-plan.md; available for AI to reference |
| research-plan.md as single overview | Yes -- single overview that summarizes all state files | User wants "a few lines to scan"; one file is better than hunting through four |

## Open Questions

1. **Backward compatibility with existing oracle sessions**
   - What we know: The archive directory has existing progress.md and research.json files from past sessions
   - What's unclear: Should oracle.sh detect old-format files and auto-migrate, or just archive them?
   - Recommendation: Just archive them. The old files are already in archive/. New sessions always create new-format files. No migration needed.

2. **Whether oracle.sh should validate state files after each iteration**
   - What we know: Phase 8 adds "state validation after each iteration with recovery." Phase 6 success criteria say "state files pass jq validation after creation and after simulated iteration updates."
   - What's unclear: How much validation belongs in Phase 6 vs Phase 8
   - Recommendation: Phase 6 adds a basic `jq -e . state.json` check in oracle.sh after each iteration (catches malformed JSON). Phase 8 adds the full validate-oracle-state + recovery logic. This satisfies the success criteria without over-building.

3. **Whether research.json should be kept for backward compatibility**
   - What we know: The wizard currently writes research.json; oracle.sh reads it
   - What's unclear: Whether to keep research.json alongside state.json or fully replace it
   - Recommendation: Fully replace. state.json contains all the same fields plus new ones. Keeping both creates confusion. The wizard writes state.json; oracle.sh reads state.json.

## Sources

### Primary (HIGH confidence)
- `.aether/oracle/oracle.sh` -- Current orchestrator implementation (134 lines)
- `.aether/oracle/oracle.md` -- Current iteration prompt (35 lines)
- `.claude/commands/ant/oracle.md` -- Current wizard/command handler (378 lines)
- `.aether/aether-utils.sh` lines 1102-1199 -- validate-state pattern (jq schema checks)
- `.aether/aether-utils.sh` lines 6461-6658 -- session-verify-fresh and session-clear for oracle
- `tests/unit/validate-state.test.js` -- Test patterns for state validation
- `tests/bash/test-session-freshness.sh` -- Bash test patterns for session utilities
- `.planning/REQUIREMENTS.md` -- LOOP-01, INTL-01, INTL-04 requirement definitions
- `.planning/ROADMAP.md` -- Phase 6-8 dependency chain and success criteria

### Secondary (MEDIUM confidence)
- [snarktank/ralph GitHub repository](https://github.com/snarktank/ralph) -- Ralph pattern: fresh context per iteration, state files as memory bridge

### Tertiary (LOW confidence)
- None -- all findings verified against project source files

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools already in use in this project; no new dependencies
- Architecture: HIGH -- schemas designed from existing patterns (validate-state, research.json) and forward requirements (Phases 7-8)
- Pitfalls: HIGH -- identified from reading actual code (session-verify-fresh file lists, oracle.md hardcoded paths, OpenCode parity)

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (stable domain; no external dependency changes expected)
