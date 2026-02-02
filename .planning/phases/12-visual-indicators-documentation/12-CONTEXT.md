# Phase 12: Visual Indicators & Documentation - Context

**Gathered:** 2026-02-02
**Status:** Ready for planning

## Phase Boundary

Users see colony activity at a glance through emoji-based status indicators, progress bars, and structured output, with all documentation path references corrected. This phase adds visual feedback to existing commands and fixes documentation inconsistencies — no new capabilities, just clarity and polish.

Scope:
- Emoji activity states (🟢 ACTIVE, ⚪ IDLE, 🔴 ERROR, ⏳ PENDING) for Worker Ant status
- Step progress indicators for multi-step operations
- Visual dashboard in `/ant:status` with emoji indicators
- Pheromone signal strength shown as progress bars
- Path reference corrections in `.aether/utils/` and `.claude/commands/ant/`

## Implementation Decisions

### Status Emoji Design

**Activity states (emoji + text format for accessibility):**
- 🐜 ACTIVE — Worker Ant currently executing a task (domain-specific emoji for clarity)
- ⚪ IDLE — Worker Ant exists but has no work
- ⏳ PENDING — Worker Ant waiting for work (explicit "waiting" indicator)
- 🔴 ERROR — Error-specific emojis for different error types:
  - 🔴 for critical errors
  - 🟡 for non-critical errors
  - ⚠️ for warnings

**Accessibility:** Always pair emojis with text labels (e.g., "🟢 ACTIVE", "⚪ IDLE") for color blindness and screen reader compatibility.

### Progress Bar Format

**Pheromone signal strength (0.0 to 1.0):**
- Medium density: ~20 characters wide (`[━━━━]`) for balanced visibility
- Smooth blocks: Use `━` character for modern, refined aesthetic
- Bar + number: Always show numeric value alongside visual (`[━━━━━━━━] 0.75`)

**Multi-step operations:**
- Step counter only: `Step 1/3: Initializing...` (simpler, step counter is sufficient)
- Use progress bars only for long-running individual steps

### Dashboard Layout (`/ant:status`)

**Worker Ant display (Standard detail per ant):**
- Ant name + status emoji + caste + current task
- Example: `builder-01 🐜 ACTIVE Builder "Building phase 12 components"`

**Structure:**
- Claude's discretion: Choose grouping (by caste, activity state, or phase) based on what's most useful
- Sections with headers: `=== ACTIVE ===`, `=== IDLE ===`, `=== ERROR ===` for scannability
- Colony-level metrics at bottom: total ants, active count, errors (summary footer)

### Step Indicators

**Format during multi-step operations:**
- List style with checkmarks:
  ```
  [✓] Initializing
  [→] Building
  [ ] Verifying
  ```
- Keep all steps visible as progress advances (user sees what's done)

**Long-running operations:**
- Claude's discretion: Add animated spinner only for operations over 60 seconds

**Error handling:**
- Error emoji for failed steps: `[🔴] Step 2/3: Building — error: connection timeout`

### Claude's Discretion

**Areas where Claude has flexibility:**
- Dashboard grouping strategy (by caste, activity state, or phase)
- Spinner threshold for long-running operations (suggested: 60+ seconds)
- Exact spacing, padding, and layout details
- Header formatting and section separators

## Specific Ideas

- Emojis should be universally recognizable in terminal environments
- "I want users to see colony health at a glance"
- Progress bars should work in narrow terminals (80 columns)
- List-style step indicators give full workflow visibility
- Accessibility matters — emojis paired with text labels

## Deferred Ideas

None — discussion stayed within phase scope.

---

*Phase: 12-visual-indicators-documentation*
*Context gathered: 2026-02-02*
