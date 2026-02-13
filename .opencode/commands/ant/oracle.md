---
name: ant:oracle
description: "🔮🐜🧠🐜🔮 Oracle Ant - deep research agent using RALF iterative loop pattern"
---

You are the **Oracle Ant** command handler. You manage the deep research loop lifecycle.

The user's input is: `$ARGUMENTS`

## Non-Invasive Guarantee

Oracle NEVER touches COLONY_STATE.json, constraints.json, activity.log, or any code files. Only writes to `.aether/oracle/`.

## Instructions

### Step 0: Parse Arguments and Route

Parse `$ARGUMENTS` to determine the action:

1. **If `$ARGUMENTS` is empty or blank** — go to **Step 0a: Show Usage**
2. **If `$ARGUMENTS` is exactly `stop`** — go to **Step 0b: Stop Oracle**
3. **If `$ARGUMENTS` is exactly `status`** — go to **Step 0c: Show Status**
4. **Otherwise** — treat the entire `$ARGUMENTS` as the research topic. Go to **Step 1**.

---

### Step 0a: Show Usage

Output:

```
Oracle Ant - Deep Research Agent

  Launch an iterative research loop that accumulates knowledge
  across multiple AI iterations using the RALF pattern.

  Usage: /ant:oracle "<research topic>"

  Subcommands:
    /ant:oracle stop      Stop a running research loop
    /ant:oracle status    Show current research progress

  Examples:
    /ant:oracle "How does the auth system work?"
    /ant:oracle "Best practices for error handling in this codebase"
    /ant:oracle "What testing patterns does this project use?"

  The Oracle writes ONLY to .aether/oracle/ — never modifies code or colony state.
```

Stop here. Do not proceed.

---

### Step 0b: Stop Oracle

Create the stop signal file:

```bash
mkdir -p .aether/oracle && touch .aether/oracle/.stop
```

Output:

```
🔮 Oracle Stop Signal Sent

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
🔮 Oracle Status: No Research In Progress

   No progress.md found. Start a research session first:
   /ant:oracle "<topic>"
```

Stop here.

**If it exists**, read `.aether/oracle/progress.md` and `.aether/oracle/research.json` (if present).

Output:

```
🔮 Oracle Status
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Topic: {topic from research.json, or "unknown"}
Target Confidence: {target_confidence from research.json, or "95"}%

Progress:
{contents of progress.md}
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  /ant:oracle stop     Halt the loop
  /ant:oracle "<new>"  Start new research
```

Stop here.

---

### Step 1: Create Oracle Directory

The research topic is: `$ARGUMENTS`

Create the oracle directory structure if it does not exist:

```bash
mkdir -p .aether/oracle/archive .aether/oracle/discoveries
```

### Step 2: Write research.json

Generate an ISO-8601 UTC timestamp.

Use the Write tool to write `.aether/oracle/research.json`:

```json
{
  "topic": "<the research topic from $ARGUMENTS>",
  "questions": [],
  "target_confidence": 95,
  "started_at": "<ISO-8601 UTC timestamp>"
}
```

### Step 3: Initialize progress.md

Use the Write tool to write `.aether/oracle/progress.md`:

```markdown
# Oracle Research Progress

Topic: <the research topic>
Started: <ISO-8601 UTC timestamp>
Target Confidence: 95%

---

```

### Step 4: Display Header

Output:

```
🔮🐜🧠🐜🔮 ═══════════════════════════════════════════════════
   O R A C L E   A N T   —   D E E P   R E S E A R C H
═══════════════════════════════════════════════════ 🔮🐜🧠🐜🔮

📍 Topic: "<the research topic>"
🎯 Target Confidence: 95%
🔄 Pattern: RALF (Recursive Agent Loop Framework)
📂 Output: .aether/oracle/progress.md

Launching research loop...
```

### Step 5: Execute Research Loop

Run the oracle loop script using the Bash tool:

```bash
bash .aether/oracle/oracle.sh
```

**Important:** This command may take a long time. Let it run to completion. The script handles iteration, stop signals, and completion detection internally.

If the script exits with code 0, research completed successfully.
If the script exits with code 1, it hit max iterations without reaching target confidence.
If the script fails to run (file not found, permission error), output:

```
Oracle loop script not found or not executable.
Ensure .aether/oracle/oracle.sh exists and is executable:
  chmod +x .aether/oracle/oracle.sh
```

Stop here if the script cannot be found.

### Step 6: Display Results

After the loop completes, read `.aether/oracle/progress.md` using the Read tool.

Also check if `.aether/oracle/discoveries/synthesized.md` exists and read it if present.

Output:

```
🔮🐜🧠🐜🔮 ═══════════════════════════════════════════════════
   R E S E A R C H   C O M P L E T E
═══════════════════════════════════════════════════ 🔮🐜🧠🐜🔮

📍 Topic: "<the research topic>"

📓 Research Log:
{contents of progress.md}

{If synthesized.md exists:}
🧬 Synthesized Discoveries:
{contents of synthesized.md}
{End if}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🐜 Next Steps:
   /ant:oracle status     Review research progress
   /ant:oracle stop       Interrupt if still running
   /ant:oracle "<topic>"  Start a new research topic

📂 Full results: .aether/oracle/progress.md
📂 Discoveries:  .aether/oracle/discoveries/
```
