---
name: design-prototyping
description: Use when UI or interaction ideas need throwaway visual prototypes before implementation decisions are locked
type: colony
domains: [design, prototyping, ui, ux, visual-exploration]
agent_roles: [architect, scout, oracle]
workflow_triggers: [discuss, plan]
task_keywords: [design, sketch, prototype, variant, mockup, interaction]
priority: normal
version: "1.0"
---

# Design Prototyping

## Purpose

Multi-variant design prototyping through throwaway HTML mockups. When you can't decide how something should look or feel, this skill generates 2-3 visual variants side by side so you can compare, react, and iterate before writing production code. Sketches are disposable -- the design decisions survive, the throwaway HTML does not.

## When to Use

- Multiple UI directions exist and you need to see them to decide
- A feature's interaction pattern is unclear -- "should it be a modal, an inline expand, or a separate page?"
- Stakeholders need to see something tangible before they can give feedback
- You want to test whether a layout works before building the real thing
- A design concept needs to be communicated to the team for alignment

## Instructions

### 1. Frame the Design Question

Every sketch session starts with a design question -- not a feature spec. The question defines what you're exploring.

**Good design questions:**
- "How should users navigate between projects -- sidebar, tabs, or breadcrumb?"
- "Where does the notification panel live -- top bar, slide-out, or floating?"
- "How do we show loading states -- skeleton, spinner, or progressive reveal?"

**Sketch card format:**

```markdown
## Sketch: {name}

**Design question:** {what you're deciding}
**Context:** {where this lives in the product}
**Constraints:** {fixed requirements -- must support mobile, must fit in sidebar, etc.}
**Variants:** {2-3 approaches to compare}
```

### 2. Generate Variants

Create each variant as a standalone HTML file in `sketches/{name}/`:

```
sketches/{name}/
  README.md            design question, variant descriptions, decision
  variant-a.html       first approach
  variant-b.html       second approach
  variant-c.html       third approach (optional)
  shared/              shared CSS/images if needed
```

**Variant rules:**
- Each variant must be a complete, self-contained HTML file (open in browser, works immediately)
- Use inline CSS -- no external dependencies, no build step
- Responsive: include a viewport meta tag so it works on mobile
- Visually distinct -- variants should feel genuinely different, not just color swaps
- Functional enough to click through the key interaction
- Annotated with comments explaining the design rationale

**Variant dimensions to explore:**

| Dimension | Options |
|-----------|---------|
| Layout | Sidebar, top nav, split pane, full-page, overlay |
| Density | Spacious, compact, dense data |
| Interaction | Click-driven, hover-driven, drag-and-drop, keyboard-first |
| Navigation | Drill-down, flat, wizard, tabs |
| Feedback | Subtle, prominent, animated, static |
| Tone | Playful, professional, minimal, rich |

### 3. Present for Comparison

When presenting variants, frame the trade-offs clearly:

```markdown
## Variant A: {name}
**Approach:** {one-line description}
**Strengths:** {what it does well}
**Weaknesses:** {what it sacrifices}
**Best for:** {when this variant shines}

## Variant B: {name}
{same structure}

## Variant C: {name} (if applicable)
{same structure}
```

Ask the human to react emotionally first, then analytically:
1. "Which one made you go 'oh nice'?"
2. "Which one would frustrate you after using it for a week?"
3. "Is there a variant you like parts of that could combine with another?"

### 4. Iterate on Selection

Once a direction is chosen (or a combination), iterate rapidly:

- **Round 2:** Refine the chosen variant with specific feedback
- **Round 3:** Polish details -- spacing, colors, edge cases
- **Final:** Produce the design reference that implementation will follow

Each iteration replaces the previous file. No version history needed -- this is a sketchpad, not a museum.

### 5. Package Design Decisions

When the sketch converges on a direction, capture the decisions:

```markdown
# Design Decision: {feature}

## Chosen Direction
{description of the selected approach}

## Why This One
{reasoning based on comparison and feedback}

## Design Specifications
- **Layout:** {specifics}
- **Interaction pattern:** {specifics}
- **Responsive behavior:** {specifics}
- **Color and typography:** {specifics}
- **Edge cases handled:** {list}

## What We Rejected and Why
| Variant | Why Not |
|---------|---------|
| A | {reason} |
| B | {reason} |

## Reference Sketch
{path to the final HTML file}

## Implementation Notes
{anything the builder needs to know that isn't obvious from the sketch}
```

## Key Patterns

### The Contrast Principle
Variants should represent genuinely different philosophies, not minor variations. If Variant A is "sidebar navigation," Variant B should be "top tabs" not "sidebar with different icons."

### The Emotional First Pattern
People know what they like before they know why. Ask for gut reactions before analytical evaluation. The emotional response reveals the real requirement.

### The Frankenstein Allowance
It's valid to say "I like A's layout but B's color scheme." Combine freely -- sketches are cheap.

### The Mobile-First Check
After choosing a direction, always sketch how it collapses on mobile. Many beautiful desktop designs become unusable on small screens.

### The Edge Case Snapshot
Before finalizing, sketch at least one edge case: empty state, error state, or maximum-content state. These often reveal layout problems that the happy path hides.

## Output Format

- Writes HTML variants to `sketches/{name}/`
- Produces design decision document as `sketches/{name}/DECISION.md`
- Returns summary and file paths to the conversation

## Examples

### Example 1: Navigation design

```
Human: "Not sure how to organize the settings page -- there are 12 sections."
Sketcher: Framing sketch: "How should users navigate 12 settings sections -- sidebar list, grouped tabs, or accordion panels?"
Generating 3 variants in sketches/settings-nav/...
  variant-a.html -- Sidebar with grouped sections
  variant-b.html -- Tab groups with sub-sections
  variant-c.html -- Accordion panels with search
Open each in your browser and tell me which feels right.
```

### Example 2: Data display

```
Human: "We need to show real-time metrics but I don't want a boring dashboard."
Sketcher: Framing sketch: "How should real-time metrics feel -- status-bar glance, immersive dashboard, or ambient background?"
Generating variants in sketches/metrics-display/...
  variant-a.html -- Minimal status bar with sparklines
  variant-b.html -- Full dashboard with cards and charts
  variant-c.html -- Ambient background that shifts color with metrics
Which one matches the experience you're imagining?
```

### Example 3: Iteration

```
Human: "Variant A is closest, but the sidebar feels too wide and I want the search at the top."
Sketcher: Iterating on Variant A -- narrowing sidebar to 200px, moving search to top with auto-focus...
Updated sketches/settings-nav/variant-a.html -- take a look.
Also generated the mobile collapse view: variant-a-mobile.html
```
