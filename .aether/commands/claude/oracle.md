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

If `visual_mode` is true:
```bash
# Generate session ID
oracle_id="oracle-$(date +%s)"

# Initialize swarm display
bash .aether/aether-utils.sh swarm-display-init "$oracle_id"
bash .aether/aether-utils.sh swarm-display-update "Oracle" "oracle" "researching" "Deep research in progress" "Colony" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0
```

Display visual header:
```
🔮🐜🧠🐜🔮 ═══════════════════════════════════════════════
          O R A C L E  —  R e s e a r c h  M o d e
═══════════════════════════════════════════════ 🔮🐜🧠🐜🔮

Oracle peering into the depths...
```

---

### Step 0b: Stop Oracle

Create the stop signal file:

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

Check if `.aether/oracle/progress.md` exists using the Read tool.

**If it does NOT exist**, output:

```
🔮🐜 Oracle Status: No Research In Progress

   No progress.md found. Start a research session:
   /ant:oracle
```

Stop here.

**If it exists**, read `.aether/oracle/progress.md` and `.aether/oracle/research.json` (if present).

Count the number of `## Iteration` headings in progress.md to determine iterations completed.

Output:

```
🔮🐜 Oracle Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Topic:       {topic from research.json, or "unknown"}
Confidence:  {target_confidence}%
Iterations:  {completed} / {max_iterations}
Started:     {started_at}

Progress:
{last 50 lines of progress.md}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  /ant:oracle stop     Halt the loop
  /ant:oracle          Start new research
```

Stop here.

---

### Step 1: Research Wizard

This is the setup phase. The Oracle asks questions to configure the research before launching.

Output the header:

```
🔮🐜🧠🐜🔮 ═══════════════════════════════════════════════════
   O R A C L E   A N T   —   R E S E A R C H   W I Z A R D
═══════════════════════════════════════════════════ 🔮🐜🧠🐜🔮
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

After collecting all answers, proceed to Step 2.

---

### Step 1.5: Check for Stale Oracle Session

Before starting new research, check for existing oracle session files.

Capture session start time:
```bash
ORACLE_START=$(date +%s)
```

Check for stale files:
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

Create the oracle directory structure:

```bash
mkdir -p .aether/oracle/archive .aether/oracle/discoveries
```

Generate an ISO-8601 UTC timestamp.

**Archive previous research if it exists:**

Check if `.aether/oracle/progress.md` exists. If it does:

```bash
DATE=$(date +%Y-%m-%d)
TIMESTAMP=$(date +%H%M%S)
mkdir -p .aether/oracle/archive
cp .aether/oracle/progress.md ".aether/oracle/archive/${DATE}-${TIMESTAMP}-progress.md" 2>/dev/null || true
cp .aether/oracle/research.json ".aether/oracle/archive/${DATE}-${TIMESTAMP}-research.json" 2>/dev/null || true
```

**Write research.json:**

Use the Write tool to write `.aether/oracle/research.json`:

```json
{
  "topic": "<the research topic>",
  "scope": "<codebase|web|both>",
  "questions": [
    "<break the topic into 3-5 specific research questions>"
  ],
  "max_iterations": <number from depth choice>,
  "target_confidence": <number from confidence choice>,
  "started_at": "<ISO-8601 UTC timestamp>"
}
```

The `questions` array is important — break the user's topic into 3-5 concrete, specific questions that the Oracle should answer. These guide each iteration.

**Write progress.md:**

Use the Write tool to write `.aether/oracle/progress.md`:

```markdown
# Oracle Research Progress

**Topic:** <the research topic>
**Started:** <ISO-8601 UTC timestamp>
**Target Confidence:** <N>%
**Max Iterations:** <N>
**Scope:** <codebase|web|both>

## Research Questions
1. <question 1>
2. <question 2>
3. <question 3>
...

---

```

#### Step 2.5: Verify Oracle Files Are Fresh

Verify that progress.md and research.json were created successfully:
```bash
verify_result=$(bash .aether/aether-utils.sh session-verify-fresh --command oracle "" "$ORACLE_START")
fresh_count=$(echo "$verify_result" | jq -r '.fresh | length')

if [[ "$fresh_count" -lt 2 ]]; then
  echo "Warning: Oracle files not properly initialized"
fi
```

Proceed to Step 3.

---

### Step 3: Launch

Output the research configuration summary:

```
🔮 Research Configured
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📍 Topic:       <topic>
🔄 Iterations:  <max_iterations>
🎯 Confidence:  <target_confidence>%
🔍 Scope:       <scope>

📋 Research Questions:
   1. <question>
   2. <question>
   3. <question>

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Now launch the loop. Try tmux first, fall back to manual.

**Try tmux:**

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

   Results will accumulate in .aether/oracle/progress.md
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

Stop here.
