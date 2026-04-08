# Pheromone Display Enhancement Plan

**Created:** 2026-02-21
**Status:** ✅ COMPLETED
**Priority:** High - User-requested feature gap

---

## Problem Statement

The pheromone system works but is invisible:

1. **Council command exists** - `/ant:council` proposes pheromones via multiple choice
2. **Pheromone signals exist** - Stored in `.aether/data/pheromones.json`
3. **Colony-prime loads them** - But only shows a count like "Primed: 2 signals"
4. **User never sees them** - The actual content is hidden

**User's exact words:** "I think it's important to create a significant plan here for how to display the pheromones that are being injected."

---

## What Exists (Don't Rebuild)

| Component | Location | Status |
|-----------|----------|--------|
| Council command | `.claude/commands/ant/council.md` | ✅ Works - multi-choice pheromone proposal |
| Focus command | `.claude/commands/ant/focus.md` | ✅ Works - emits FOCUS signal |
| Redirect command | `.claude/commands/ant/redirect.md` | ✅ Works - emits REDIRECT signal |
| Feedback command | `.claude/commands/ant/feedback.md` | ✅ Works - emits FEEDBACK signal |
| Pheromone storage | `.aether/data/pheromones.json` | ✅ Works - stores signals |
| Pheromone-prime | `aether-utils.sh:6211` | ✅ Works - loads signals with decay |
| Colony-prime | `aether-utils.sh:6337` | ✅ Works - combines wisdom + signals |
| Pheromone-count | `aether-utils.sh:6036` | ✅ Works - counts active signals |

---

## What's Missing

### 1. **Pheromone Display Function** (aether-utils.sh)

A new `pheromone-display` function that outputs a formatted table:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A C T I V E   P H E R O M O N E S
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎯 FOCUS (Pay attention here)
   1. [85%] "security" — injected 2d ago
   2. [72%] "performance optimization" — injected 5d ago

🚫 REDIRECT (Hard constraints - DO NOT do this)
   1. [90%] "use ORM" — injected 1d ago
   2. [68%] "jQuery" — injected 7d ago

💬 FEEDBACK (Guidance to consider)
   1. [65%] "prefer composition over inheritance" — injected 3d ago

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Total: 5 active signals | Decay: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d
```

### 2. **Build Command Enhancement** (build.md)

After Step 4 (Load Colony Context), add Step 4.1:

```markdown
### Step 4.1: Display Active Pheromones

Run using the Bash tool:
```bash
bash .aether/aether-utils.sh pheromone-display
```

This displays the formatted pheromone table to the user so they can see what guidance is active.
```

### 3. **Status Command Enhancement** (status.md)

Add pheromone summary to `/ant:status` output:

```markdown
### Pheromones Active

🎯 FOCUS: 2 signals
🚫 REDIRECT: 1 signal
💬 FEEDBACK: 1 signal

Run /ant:pheromones for full details
```

### 4. **New Pheromones Command** (pheromones.md)

A dedicated command for viewing/managing pheromones:

```markdown
---
name: ant:pheromones
description: "🎯🐜🚫🐜💬 View and manage active pheromone signals"
---

Usage:
  /ant:pheromones          # Display all active signals
  /ant:pheromones focus    # Show only FOCUS signals
  /ant:pheromones redirect # Show only REDIRECT signals
  /ant:pheromones feedback # Show only FEEDBACK signals
  /ant:pheromones clear    # Clear all expired signals
  /ant:pheromones expire <id> # Expire a specific signal
```

---

## Implementation Plan

### Phase 1: pheromone-display Function (aether-utils.sh)

**Location:** After `pheromone-count` (~line 6058)

```bash
pheromone-display)
  # Display active pheromones in formatted table
  # Usage: pheromone-display [type]
  #   type: Optional filter (focus/redirect/feedback)
  # Returns: Formatted table string

  pd_file="$DATA_DIR/pheromones.json"
  pd_type="${1:-all}"
  pd_now=$(date +%s)

  if [[ ! -f "$pd_file" ]]; then
    echo "No pheromones file found. Run /ant:init to initialize colony."
    exit 0
  fi

  # Build display using same decay logic as pheromone-read
  # ... (implementation details)
```

**Output format:**
- Header with box drawing
- Grouped by type (FOCUS, REDIRECT, FEEDBACK)
- Each signal shows: effective strength %, content, age
- Footer with totals and decay info

### Phase 2: Update build.md

Add Step 4.1 after colony-prime call to display pheromones visibly.

### Phase 3: Update status.md

Add pheromone summary section after memory health display.

### Phase 4: Create pheromones.md Command

New slash command for dedicated pheromone viewing and management.

---

## Display Locations

| Where | What | When |
|-------|------|------|
| `/ant:build` | Full table | Before spawning workers |
| `/ant:status` | Summary counts | In colony overview |
| `/ant:council` | After injection | Show what was added |
| `/ant:pheromones` | Full table + management | On demand |
| `/ant:continue` | Summary | After phase completion |

---

## Visual Design

### Full Table (for /ant:build, /ant:pheromones)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
   A C T I V E   P H E R O M O N E S
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎯 FOCUS (Pay attention here)
   1. [85%] "security"
      └── injected 2d ago, expires in 28d
   2. [72%] "performance optimization"
      └── injected 5d ago, expires in 25d

🚫 REDIRECT (Hard constraints - DO NOT do this)
   1. [90%] "use ORM"
      └── injected 1d ago, expires in 59d
   2. [68%] "jQuery"
      └── injected 7d ago, expires in 53d

💬 FEEDBACK (Guidance to consider)
   1. [65%] "prefer composition over inheritance"
      └── injected 3d ago, expires in 87d

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
5 signals active | Decay rates: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d
```

### Summary (for /ant:status)

```
┌─────────────────────────────────────────────────────────┐
│ PHEROMONES                                               │
├─────────────────────────────────────────────────────────┤
│ 🎯 FOCUS: 2    🚫 REDIRECT: 1    💬 FEEDBACK: 1         │
│                                                          │
│ Strongest: "use ORM" [90%]                              │
│ Newest: "security" [85%] - 2d ago                       │
│                                                          │
│ Run /ant:pheromones for details                         │
└─────────────────────────────────────────────────────────┘
```

### Injection Confirmation (for /ant:council, /ant:focus, etc.)

```
✓ Pheromone injected:
  🎯 FOCUS: "security"
  Strength: 0.8 | Expires: 30d

Active signals: 5
```

---

## Files to Create/Modify

| File | Action | Purpose |
|------|--------|---------|
| `.aether/aether-utils.sh` | Modify | Add `pheromone-display` function |
| `.claude/commands/ant/build.md` | Modify | Add Step 4.1 - display pheromones |
| `.claude/commands/ant/status.md` | Modify | Add pheromone summary section |
| `.claude/commands/ant/pheromones.md` | Create | New command for viewing/managing |
| `.opencode/commands/ant/pheromones.md` | Create | OpenCode version |

---

## Testing

1. Run `/ant:focus "test signal"` and verify it appears in display
2. Run `/ant:redirect "avoid this"` and verify it appears in display
3. Run `/ant:build 1` and verify pheromone table shows before workers spawn
4. Run `/ant:status` and verify pheromone summary appears
5. Test decay: Create signal, wait, verify strength decreases

---

## Success Criteria

- [x] `/ant:build` displays full pheromone table before spawning workers
- [x] `/ant:status` shows pheromone counts and strongest/newest
- [x] `/ant:pheromones` command exists for dedicated viewing
- [x] `/ant:council` displays what was injected after session (already existed)
- [x] All signal types (FOCUS/REDIRECT/FEEDBACK) visible
- [x] Decay strength shown as percentage
- [x] Age and expiry shown per signal
