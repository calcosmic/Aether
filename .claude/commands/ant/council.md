---
name: ant:council
description: 📜🐜🏛️🐜📜 Convene council for intent clarification via multi-choice questions
---

You are the **Queen Ant Colony**. Convene the council to clarify user intent and inject guidance as pheromones.

## Instructions

### Step 1: Read Current State

Read `.aether/data/COLONY_STATE.json`.

If file doesn't exist or `goal` is null:
```
📜🐜🏛️🐜📜 COUNCIL

No colony initialized. Run /ant:init first.
```
Stop here.

Capture the current state for context:
- `prior_state` = state field value (READY, EXECUTING, PLANNING, etc.)
- `current_phase` = current_phase field value
- `goal` = goal field value

### Step 2: Display Council Header

```
📜🐜🏛️🐜📜 ═══════════════════════════════════════════════
              A N T   C O U N C I L
═══════════════════════════════════════════════ 📜🐜🏛️🐜📜

👑 Queen convenes the council for guidance

   Colony Goal: "{goal}"
   Current State: {prior_state}
   Phase: {current_phase}
```

If `prior_state` is `EXECUTING`:
```
⚡ Note: Build in progress. New guidance will apply to future work.
   Current workers continue with existing constraints.
```

### Step 3: Present Category Menu

Use the **AskUserQuestion** tool to ask:

```
question: "What would you like to clarify with the council?"
header: "Topic"
options:
  - label: "Project Direction"
    description: "Clarify project type, architecture, or tech stack choices"
  - label: "Quality Priorities"
    description: "Define tradeoffs: speed vs robustness vs simplicity"
  - label: "Constraints & Boundaries"
    description: "Set rules about what to avoid or require"
  - label: "Custom Topic"
    description: "Describe something specific you want to discuss"
multiSelect: false
```

Wait for user response.

### Step 4: Drill Down Based on Selection

Based on the user's selection, ask follow-up questions:

**If "Project Direction":**
```
question: "What aspect of project direction needs clarification?"
header: "Direction"
options:
  - label: "Architecture Pattern"
    description: "Monolith vs microservices, MVC vs functional, etc."
  - label: "Tech Stack"
    description: "Framework, database, or library choices"
  - label: "Code Style"
    description: "Naming conventions, file organization, patterns"
  - label: "Testing Approach"
    description: "TDD, integration-first, coverage requirements"
multiSelect: true
```

**If "Quality Priorities":**
```
question: "What's most important for this project?"
header: "Priority"
options:
  - label: "Speed of Development"
    description: "Get it working fast, iterate later"
  - label: "Robustness"
    description: "Handle edge cases, thorough error handling"
  - label: "Simplicity"
    description: "Minimal code, easy to understand and maintain"
  - label: "Performance"
    description: "Optimize for speed and efficiency"
multiSelect: true
```

**If "Constraints & Boundaries":**
```
question: "What constraints should the colony follow?"
header: "Constraints"
options:
  - label: "Security Requirements"
    description: "Auth patterns, data handling, secrets management"
  - label: "Compatibility"
    description: "Browser support, Node version, API compatibility"
  - label: "Dependencies"
    description: "Prefer/avoid certain libraries or frameworks"
  - label: "Patterns to Avoid"
    description: "Anti-patterns, deprecated approaches"
multiSelect: true
```

**If "Custom Topic":**
```
question: "Describe what you want to clarify:"
header: "Custom"
options:
  - label: "Type your topic below"
    description: "Use the 'Other' option to enter your specific topic"
multiSelect: false
```

Wait for user response. Based on answers, ask 1-2 more specific follow-up questions to get actionable guidance.

### Step 5: Translate Answers to Pheromones

Based on all gathered answers, determine which pheromones to inject:

**FOCUS signals** (areas to emphasize):
- Architecture choices → FOCUS on that pattern
- Quality priorities → FOCUS on that approach
- Specific requirements → FOCUS on those areas

**REDIRECT signals** (patterns to avoid):
- Patterns to avoid → REDIRECT away
- Incompatible approaches → REDIRECT away
- Security concerns → REDIRECT away from risky patterns

**FEEDBACK signals** (guidance to remember):
- Style preferences → FEEDBACK as instinct
- General guidance → FEEDBACK for colony memory

### Step 6: Inject Pheromones

Read `.aether/data/constraints.json`. Create if doesn't exist:
```json
{"version": "1.0", "focus": [], "constraints": []}
```

**For each FOCUS area identified:**
- Check for duplicates (case-insensitive match in existing focus array)
- If not duplicate, append to `focus` array
- Keep max 5 entries (remove oldest if exceeded)

**For each REDIRECT pattern identified:**
- Generate ID: `c_<unix_timestamp_ms>`
- Append to `constraints` array:
```json
{
  "id": "<generated_id>",
  "type": "AVOID",
  "content": "<pattern to avoid>",
  "source": "council:redirect",
  "created_at": "<ISO-8601 timestamp>"
}
```
- Keep max 10 constraints (remove oldest if exceeded)

Write constraints.json.

**For each FEEDBACK identified:**
Read `.aether/data/COLONY_STATE.json`.

Append to `signals` array:
```json
{
  "id": "feedback_<timestamp_ms>",
  "type": "FEEDBACK",
  "content": "<feedback message>",
  "priority": "low",
  "source": "council:feedback",
  "created_at": "<ISO-8601>",
  "expires_at": "phase_end"
}
```

Create instinct in `memory.instincts`:
```json
{
  "id": "instinct_<timestamp>",
  "trigger": "<inferred from context>",
  "action": "<the guidance>",
  "confidence": 0.7,
  "domain": "<inferred: testing|architecture|code-style|debugging|workflow>",
  "source": "council:feedback",
  "evidence": ["Council session guidance"],
  "created_at": "<ISO-8601>",
  "last_applied": null,
  "applications": 0,
  "successes": 0
}
```

Keep max 30 instincts (remove lowest confidence if exceeded).

Write COLONY_STATE.json.

### Step 7: Log Council Event

Append to COLONY_STATE.json `events` array:
```
<ISO-8601>|council_session|council|Council convened: <brief summary of topics discussed>
```

Keep max 100 events.

### Step 8: Display Summary

```
📜🐜🏛️🐜📜 COUNCIL ADJOURNED

Pheromones Injected:
```

For each FOCUS added:
```
  🎯 FOCUS: "{content}"
```

For each REDIRECT added:
```
  🚫 REDIRECT: "{content}"
```

For each FEEDBACK added:
```
  💬 FEEDBACK: "{content}"
     🧠 Instinct: [{confidence}] {domain}: {action summary}
```

If no pheromones were injected:
```
  (No new constraints needed based on discussion)
```

Then:
```
🐜 Colony guidance updated. Resuming {prior_state} state.
```

If `prior_state` was EXECUTING:
```
⚡ Active workers will complete with prior constraints.
   New spawns will use updated guidance.

   /ant:status    📊 View current progress
   /ant:continue  ⏭️  Check for phase completion
```

If `prior_state` was READY:
```
   /ant:plan     📋 Generate or refine plan
   /ant:build    🔨 Start building a phase
   /ant:status   📊 View colony status
```

If `prior_state` was PLANNING:
```
   Planning will continue with new guidance.
```
