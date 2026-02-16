# Phase 12: Colony Visualization - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Users experience immersive real-time colony activity display with ant-themed presentation, collapsible views, and comprehensive metrics. This phase extends the existing `/ant:swarm` command with live updates during colony operations and adds `/ant:maturity` for milestone visualization.

</domain>

<decisions>
## Implementation Decisions

### Real-time display format
- **Scrolling activity log** (like `tail -f`) — not a static dashboard
- Display **exits automatically** when colony work completes
- **Collapses by caste** — shows "Builder: 12 reads, 5 edits" then can expand to see details
- **No timestamps** — clean log without time indicators
- **Indentation for nesting** — child ants indented under parent, always visible (not collapsible groups)
- **Completed ants fade out** — working ants are bold/bright, done ants are dimmed/grayed
- **Token counts shown** — "trophallaxis" shown per task in the log line

### Visual immersion level
- **Fully immersive** — heavy theming throughout with ant terminology
- **Progress indicator** — mix of ASCII progress bar AND animated text ("...excavating...")
- **Milestone visualization** — detailed ASCII art anthill (40+ lines, intricate)
- **Language** — "The colony is foraging...", "3 foragers excavating...", playful ant metaphors

### Chamber activity map
- **Text labels with emoji indicators** — "Fungus Garden 🍄 (3 ants)"
- **Hide empty chambers** — only show zones with activity
- **Extended 5 zones** — Fungus Garden, Nursery, Refuse Pile, Throne Room (Queen), Foraging Trail
- **Visual intensity for activity** — fire/flame icons: "Fungus Garden 🍄🔥🔥"

### Caste presentation
- **Colored emoji + colored text together** — the entire caste indicator is the caste's color
  - Builder (🔨) = Blue
  - Watcher (👁️) = Green
  - Scout (🔍) = Yellow
  - Chaos (🎲) = Red
  - Prime (👑) = Purple/Magenta
- **Parent ants highlighted** — bold AND underlined to distinguish orchestrators from workers
- **Completed ants go neutral** — gray color when finished (not caste color)

### Claude's Discretion
- Exact ASCII art anthill design and growth progression
- Chamber zone emoji selection
- Animation speed for progress indicators
- Exact color codes (which ANSI escape sequences)
- Specific collapse/expand interaction patterns

</decisions>

<specifics>
## Specific Ideas

- "I want it like watching logs tail in real-time" — scrolling format chosen
- Mix of progress bar + animated text for progress indicators
- Detailed (40+ line) ASCII art anthill for maturity visualization
- Parent ants need clear visual distinction (bold + underline)
- Completed work should be clearly distinguished from active work

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 12-colony-visualization*
*Context gathered: 2026-02-14*
