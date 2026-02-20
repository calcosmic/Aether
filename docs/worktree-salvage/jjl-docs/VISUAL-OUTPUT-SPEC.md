# Aether Visual Output Specification

## Problem Statement

Current issues:
1. Workers output their entire RETURN FORMAT templates to the user
2. No real-time visibility into what workers are doing
3. Scrolling bash commands instead of clean task updates
4. No hierarchical tree view of workers

## Target Visual Experience

### Real-Time Worker Tree (updates in place)

```
🔨 COLONY BUILD — Phase 1: Foundation
═══════════════════════════════════════

👑 Queen
├── 🔨 Anvil-51      Task 1.1  Creating database schema...
│   └── 🔍 Scout-12  Sub-task  Checking existing tables...
├── 🔨 Hammer-12     Task 1.2  Setting up auth middleware...
│   └── ✅ Complete
├── 👁️ Vigil-23     Verifying  Running tests...
└── 🎲 Chaos-8      Testing   Edge case exploration...

Progress: [████████████░░░░░░░░] 60%
```

### Activity Stream (compact, no scrolling)

```
10:05:01  🥚 Queen      Phase 1 started
10:05:02  🔨 Anvil-51   Task 1.1: Creating database schema
10:05:03  🔍 Scout-12   Spawned by Anvil-51: Checking tables
10:05:05  🔨 Hammer-12  Task 1.2: Setting up auth
10:05:08  🔨 Hammer-12  → Complete
10:05:09  👁️ Vigil-23   Verification started
```

### Worker Spawn (instant notification)

```
🐜 SPAWNED  🔨 Anvil-51 (Builder)  Task 1.1: Database schema
```

### Worker Complete (instant notification)

```
✅ DONE     🔨 Anvil-51 (Builder)  Task 1.1  3 files created
```

## Implementation Requirements

### 1. Worker Output Rules

**NEVER output these to user:**
- ❌ "--- YOUR TASK ---" headers
- ❌ "--- RETURN FORMAT ---" sections
- ❌ JSON schema explanations
- ❌ Instructions on how to return data

**DO output:**
- ✅ Progress updates: "Task 1.1: Creating tables..."
- ✅ Completion: "✅ Task 1.1 complete — 3 files created"
- ✅ Spawn events: "🐜 Spawning Scout-12 to check tables"

### 2. Worker Prompt Structure (clean output)

```markdown
You are {name}, a {emoji} {Caste} Ant.

Task: {task_description}

Work and report progress:
- Use activity-log for each step: bash .aether/aether-utils.sh activity-log "ACTION" "{name}" "description"
- Update swarm display: bash .aether/aether-utils.sh swarm-display-update ...
- When spawning sub-workers, announce: "🐜 Spawning {child_name} for {reason}"

Return ONLY this JSON at completion (no other text):
{"status": "completed|failed", "summary": "...", "files_created": [...]}
```

### 3. Visual Update Protocol

**Every worker must:**

1. **On spawn**: Call `swarm-display-update` with status "excavating"
2. **Every 30 seconds OR at task switch**: Update swarm display with new task description
3. **On sub-spawn**: Log "🐜 Spawning {child} for {reason}"
4. **On complete**: Call `spawn-complete` and return clean JSON

### 4. Caste Emoji Mapping

| Caste | Emoji | Color (if supported) |
|-------|-------|---------------------|
| Builder | 🔨 | Cyan |
| Watcher | 👁️ | Green |
| Scout | 🔍 | Yellow |
| Chaos | 🎲 | Red |
| Archaeologist | 🏺 | Magenta |
| Colonizer | 🧹 | Blue |
| Architect | 🏛️ | White |
| Prime/Queen | 🥚 | Bold |
| Oracle | 🔮 | Magenta |
| Dreamer | 💭 | Blue |
| Interpreter | 🔍 | Yellow |
| Chaos | 🎲 | Red |
| Alate | 🪽 | Cyan |

### 5. Task Display Format

```
{emoji} {Name:<12} {Task:<30} {Status}
```

Examples:
```
🔨 Anvil-51     Task 1.1: Database schema    ●●● 60%
👁️ Vigil-23     Verifying auth module        observing...
🎲 Chaos-8      Testing edge cases           disrupting...
```

### 6. Tree View Format (for final summary)

```
👑 Queen
├── 🔨 Anvil-51      Task 1.1  [✅ done]
│   └── 🔍 Scout-12  Sub-task  [✅ done]
├── 🔨 Hammer-12     Task 1.2  [✅ done]
└── 👁️ Vigil-23      Verify    [✅ done]
    └── 🎲 Chaos-8   Stress    [✅ done]
```

### 7. Progress Bar Standard

```javascript
// Render progress bar
function progressBar(percent, width = 20) {
  const filled = Math.round((percent / 100) * width);
  const empty = width - filled;
  return '█'.repeat(filled) + '░'.repeat(empty);
}

// Usage
progressBar(60)  // ████████████░░░░░░░░
```

### 8. No Scroll Rule

**Updates should modify display in place, not scroll:**
- Use `swarm-display.json` as shared state
- `swarm-display.sh` renders current state (clears screen)
- Workers only update the JSON, don't print progress
- User runs `/ant:watch` to see live display
- Without watch mode: only see spawn/complete events

## Worker Output Examples

### Good Output

```
🐜 Spawning Scout-12 to check existing tables
🔍 Scout-12: Found 3 existing tables
🔍 Scout-12: Table 'users' needs migration
✅ Scout-12 complete — analysis done
```

### Bad Output (what we have now)

```
--- YOUR TASK ---
Check existing database tables

--- RETURN FORMAT ---
Task: {what you were asked to do}
Status: completed / failed / blocked
Summary: {1-2 sentences}
Files: {only if you touched files}
Spawn Tree: {only if you spawned sub-workers}
Next Steps / Recommendations: {required}

--- MODEL CONTEXT ---
Assigned model: kimi-k2.5
Caste: scout
...
```

## Migration Checklist

Update all command files to remove:
- [ ] "--- YOUR TASK ---" headers
- [ ] "--- RETURN FORMAT ---" sections
- [ ] "--- MODEL CONTEXT ---" sections
- [ ] Detailed JSON schema explanations
- [ ] Instructional text meant for prompt only

Replace with:
- [ ] Clean spawn announcements
- [ ] Progress via activity-log only
- [ ] Clean completion JSON only
- [ ] Visual tree in final summary

## Commands to Update

1. `/ant:build` — Clean worker prompts
2. `/ant:swarm` — Clean scout prompts
3. `/ant:oracle` — Clean researcher prompts
4. `/ant:colonize` — Clean colonizer prompts
5. `/ant:dream` — Clean dreamer prompts
6. `/ant:watch` — Ensure clean rendering

## Status

- [x] Visual specification defined
- [ ] Worker prompts cleaned
- [ ] Return format minimized
- [ ] Tree view implemented
- [ ] Progress bars standardized
