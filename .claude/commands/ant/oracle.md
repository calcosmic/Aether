---
name: ant:oracle
description: "🔮🐜🧠🐜🔮🐜 Oracle Ant - deep research agent using RALF iterative loop pattern"
---

You are the **Oracle Ant** command handler. You configure and launch a deep research loop that runs autonomously in a separate process.

The user's input is: `$ARGUMENTS`

## Non-Invasive Guarantee

Oracle NEVER touches COLONY_STATE.json, constraints.json, activity.log, or any code files. Only writes to `.aether/oracle/`.

## Instructions

### Step 0: Parse Arguments and Route

Parse `$ARGUMENTS` to determine the action:

1. Check for flags:
   - If contains `--no-visual`: set `visual_mode = false` (visual is ON by default)
   - If contains `--force` or `--force-research`: set `force_research = true`
   - Otherwise: set `visual_mode = true`, `force_research = false`
   - Remove flags from arguments before routing

2. **If remaining arguments is exactly `stop`** — go to **Step 0b: Stop Oracle**
3. **If remaining arguments is exactly `status`** — go to **Step 0c: Show Status**
4. **Otherwise** — go to **Step 0.5: Initialize Visual Mode** then **Step 1: Research Wizard**

### Step 0.5: Initialize Visual Mode (if enabled)

Display visual header:
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔮🐜🧠🐜🔮  O R A C L E  —  R e s e a r c h  M o d e
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Oracle peering into the depths...
```

---

### Step 0b: Stop Oracle

Create the stop signal file by running using the Bash tool with description "Stopping oracle research...":

```bash
mkdir -p .aether/oracle && touch .aether/oracle/.stop
```

Output:

```
🔮🐜 Oracle Stop Signal Sent

   Created .aether/oracle/.stop
   The research loop will halt at the end of the current iteration.

   To check final results: /ant:oracle status
```

Stop here. Do not proceed.

---

### Step 0c: Show Status

Check if `.aether/oracle/research-plan.md` exists using the Read tool.

**If it does NOT exist**, output:

```
🔮🐜 Oracle Status: No Research In Progress

   No active research session. Start one:
   /ant:oracle
```

Stop here.

**If it exists**, read `.aether/oracle/research-plan.md` and `.aether/oracle/state.json` (if present).

Output:

```
🔮🐜 Oracle Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Topic:       {topic from state.json, or "unknown"}
Iteration:   {iteration} of {max_iterations}
Status:      {status}

{contents of research-plan.md}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  /ant:oracle stop     Halt the loop
  /ant:oracle          Start new research
```

Stop here.

---

### Step 1: Research Wizard

This is the setup phase. The Oracle asks questions to configure the research before launching.

Output the header:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔮🐜🧠🐜🔮  O R A C L E   A N T  —  R E S E A R C H   W I Z A R D
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**If `$ARGUMENTS` is not empty and not a subcommand**, use it as the initial topic suggestion. Otherwise, the topic will be asked in Question 1.

Now ask questions using AskUserQuestion. Ask them one at a time so each answer can inform the next question.

**Question 1: Research Topic**

If `$ARGUMENTS` already contains a topic, skip this question and use that as the topic.

Otherwise ask:

```
What should the Oracle research?
```

Options:
1. **Codebase analysis** — Deep dive into how this codebase works (architecture, patterns, conventions)
2. **External research** — Research a technology, library, or concept using web search
3. **Both** — Combine codebase exploration with external research

Then use a follow-up AskUserQuestion with a free-text prompt:

```
Describe the research topic in detail. The more specific, the better the Oracle's results.
```

(The user will type their topic via the "Other" free-text option.)

**Question 2: Research Depth**

```
How deep should the Oracle go?
```

Options:
1. **Quick scan (5 iterations)** — Surface-level overview, fast results
2. **Standard research (15 iterations)** — Thorough investigation, good balance
3. **Deep dive (30 iterations)** — Exhaustive research, leaves no stone unturned
4. **Marathon (50 iterations)** — Maximum depth, may take hours

**Question 3: Confidence Target**

```
When should the Oracle consider the research complete?
```

Options:
1. **80% confidence** — Good enough for a first pass, stops early
2. **90% confidence** — Solid understanding, most questions answered
3. **95% confidence (recommended)** — Thorough, few gaps remaining
4. **99% confidence** — Near-exhaustive, won't stop until almost everything is known

**Question 4: Research Scope** (only if topic involves codebase)

```
Should the Oracle also search the web, or stay within the codebase?
```

Options:
1. **Codebase only** — Only use Glob, Grep, Read to explore local files
2. **Codebase + web** — Also use WebSearch and WebFetch for docs, best practices, prior art
3. **Web only** — Focus on external research (libraries, concepts, techniques)

**Question 5: Search Strategy**

(Ask this question for all research types.)

```
How should the Oracle approach the research?
```

Options:
1. **Adaptive (recommended)** -- Oracle decides when to go broad vs deep based on research progress
2. **Breadth-first** -- Cover all questions with initial findings before going deep on any single one
3. **Depth-first** -- Pick the most important question and investigate it exhaustively before moving on

**Question 6: Focus Areas** (optional)

```
Are there specific aspects you want the Oracle to prioritize?
```

Options:
1. **No specific focus** -- Let the Oracle decide what to investigate first
2. **Yes, I have focus areas** -- I want to steer the research toward specific aspects

If the user selects option 2, ask a follow-up AskUserQuestion with free-text:

```
List your focus areas (comma-separated). Example: "security implications, performance under load, migration path"
```

Parse the comma-separated response into individual focus area strings.

After collecting all answers, proceed to Step 2.

---

### Step 1.5: Check for Stale Oracle Session

Before starting new research, check for existing oracle session files.

Capture session start time:
```bash
ORACLE_START=$(date +%s)
```

Check for stale files by running using the Bash tool with description "Checking for stale oracle session...":
```bash
stale_check=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
has_stale=$(echo "$stale_check" | jq -r '.stale | length')
has_progress=$(echo "$stale_check" | jq -r '.fresh | length')

if [[ "$has_stale" -gt 0 ]] || [[ "$has_progress" -gt 0 ]]; then
  # Found existing oracle session
  if [[ "$force_research" == "true" ]]; then
    bash .aether/aether-utils.sh session-clear --command oracle
    echo "Cleared stale oracle session for fresh research"
  else
    # Existing session found - prompt user
    echo "Found existing oracle session. Options:"
    echo "  /ant:oracle status     - View current session"
    echo "  /ant:oracle --force    - Restart with fresh session"
    echo "  /ant:oracle stop       - Stop current session"
    # Don't proceed - let user decide
    exit 0
  fi
fi
```

---

### Step 2: Configure Research

Create the oracle directory structure by running using the Bash tool with description "Setting up oracle research...":

```bash
mkdir -p .aether/oracle/archive .aether/oracle/discoveries
```

Generate an ISO-8601 UTC timestamp.

**Archive previous research if it exists:**

Check if `.aether/oracle/state.json` exists. If it does, run using the Bash tool with description "Archiving previous research...":

```bash
ARCHIVE_TS=$(date +%Y-%m-%d-%H%M%S)
mkdir -p .aether/oracle/archive/$ARCHIVE_TS
for f in state.json plan.json gaps.md synthesis.md research-plan.md; do
  [ -f ".aether/oracle/$f" ] && cp ".aether/oracle/$f" ".aether/oracle/archive/$ARCHIVE_TS/"
done
```

**Write state.json** (replaces research.json):

Use the Write tool to create `.aether/oracle/state.json`:

```json
{
  "version": "1.1",
  "topic": "<the research topic>",
  "scope": "<codebase|web|both>",
  "phase": "survey",
  "iteration": 0,
  "max_iterations": <number from depth choice>,
  "target_confidence": <number from confidence choice>,
  "overall_confidence": 0,
  "started_at": "<ISO-8601 UTC timestamp>",
  "last_updated": "<ISO-8601 UTC timestamp>",
  "status": "active",
  "strategy": "<strategy from Question 5: adaptive|breadth-first|depth-first>",
  "focus_areas": [<array of focus area strings from Question 6, or empty array if no focus>]
}
```

**Emit focus area pheromones** (if any focus areas were set):

For each focus area string from Question 6, run using the Bash tool with description "Emitting focus area pheromones...":

```bash
bash .aether/aether-utils.sh pheromone-write FOCUS "$focus_area" \
  --strength 0.8 --source "oracle:wizard" \
  --reason "Focus area set in oracle wizard" --ttl "24h" 2>/dev/null || true
```

**Write plan.json:**

Break the topic into 3-8 sub-questions (adapt to topic complexity — broader topics get more questions, focused topics fewer). Use the Write tool to create `.aether/oracle/plan.json`:

```json
{
  "version": "1.1",
  "sources": {},
  "questions": [
    {
      "id": "q1",
      "text": "<specific research question>",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    }
  ],
  "created_at": "<ISO-8601 UTC timestamp>",
  "last_updated": "<ISO-8601 UTC timestamp>"
}
```

**Write gaps.md** (initial empty structure):

Use the Write tool to create `.aether/oracle/gaps.md`:

```markdown
# Knowledge Gaps

## Open Questions
(No research conducted yet)

## Contradictions
(None identified)

## Last Updated
Iteration 0 -- <ISO-8601 UTC timestamp>
```

**Write synthesis.md** (initial empty structure):

Use the Write tool to create `.aether/oracle/synthesis.md`:

```markdown
# Research Synthesis

## Topic
<the research topic>

## Findings by Question
(No findings yet -- research has not started)

## Last Updated
Iteration 0 -- <ISO-8601 UTC timestamp>
```

**Generate research-plan.md:**

After writing plan.json, generate research-plan.md as the executive summary. Use the Write tool to create `.aether/oracle/research-plan.md`:

```markdown
# Research Plan

**Topic:** <topic>
**Status:** active | **Iteration:** 0 of <max>
**Overall Confidence:** 0%

## Questions
| # | Question | Status | Confidence |
|---|----------|--------|------------|
| q1 | <question text> | open | 0% |
| q2 | ... | open | 0% |

## Next Steps
Next investigation: <text of q1, the first question>

---
*Generated from plan.json -- do not edit directly*
```

#### Step 2.5: Verify Oracle Files Are Fresh

Verify that state.json, plan.json, gaps.md, synthesis.md, and research-plan.md were created successfully by running using the Bash tool with description "Verifying oracle files...":
```bash
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
fresh_count=$(echo "$verify_result" | jq -r '.fresh | length')

if [[ "$fresh_count" -lt 5 ]]; then
  echo "Warning: Oracle files not properly initialized"
fi
```

Proceed to Step 3.

---

### Step 3: Launch

Output the research configuration summary, showing the sub-questions from plan.json:

```
🔮 Research Configured
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Topic:       <topic>
🔄 Iterations:  <max_iterations>
🎯 Confidence:  <target_confidence>%
🔍 Scope:       <scope>
📐 Strategy:    <strategy>
🎯 Focus:       <focus areas comma-separated, or "None">

📋 Sub-Questions:
   q1. <question text from plan.json>
   q2. <question text from plan.json>
   q3. <question text from plan.json>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Now launch the loop. Try tmux first, fall back to manual.

**Try tmux** by running using the Bash tool with description "Launching oracle in tmux...":

```bash
tmux new-session -d -s oracle "cd $(pwd) && bash .aether/oracle/oracle.sh; echo ''; echo '🔮🐜 Oracle loop finished. Press any key to close.'; read -n1" 2>/dev/null && echo "TMUX_OK" || echo "TMUX_FAIL"
```

**If TMUX_OK:**

```
🔮🐜 Oracle Launched
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   The Oracle is researching in a background tmux session.

   👁️  Watch live:     tmux attach -t oracle
   📊 Check status:   /ant:oracle status
   🛑 Stop early:     /ant:oracle stop

   Research progress visible at .aether/oracle/research-plan.md
   The Oracle will stop when it reaches {target_confidence}% confidence
   or completes {max_iterations} iterations.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   You can keep working. The Oracle runs independently.
```

Stop here.

**If TMUX_FAIL** (tmux not installed or error):

```
🔮 Ready to Launch
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

   tmux not available. Run this in a separate terminal:

   cd {current_working_directory}
   bash .aether/oracle/oracle.sh

   Then come back here:
   📊 Check status:   /ant:oracle status
   🛑 Stop early:     /ant:oracle stop

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

Stop here.
