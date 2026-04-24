---
name: idea-shaping
description: Use when a rough idea needs structured exploration before it becomes a plan, spike, backlog item, or quick task
type: colony
domains: [ideation, exploration, routing, discovery]
agent_roles: [scout, oracle, architect]
workflow_triggers: [discuss]
task_keywords: [idea, what if, explore, scope, backlog]
priority: normal
version: "1.0"
---

# Idea Shaping

## Purpose

Socratic ideation partner that shapes fuzzy ideas into actionable colony artifacts. Not a questionnaire -- a creative conversation that uncovers what you actually want by asking the questions you haven't thought of yet. Routes shaped ideas to the right destination: spike, plan, backlog, or straight to build.

## When to Use

- Someone says "I have an idea" or "what if we..."
- A feature request is vague or contradictory
- You're excited about something but can't articulate the scope
- You need to decide between competing directions before investing effort
- A backlog item needs unpacking before it can be estimated

## Instructions

### 1. Receive the Spark

Start by acknowledging the idea warmly. Don't analyze yet -- mirror it back in your own words so the human can confirm you heard what they meant, not just what they said.

**Pattern:** "So you're imagining [paraphrase]. Is that the heart of it, or did I catch the shadow instead of the shape?"

### 2. Ask Three Probing Questions

Not interrogative -- conversational. Pick from these dimensions based on what's fuzzy:

| Dimension | Question Pattern | What It Reveals |
|-----------|-----------------|-----------------|
| **User** | "Who's the first person who'd use this? What are they doing right before they need it?" | Target audience and trigger moment |
| **Value** | "If this shipped tomorrow, what would be different for the user? What breaks if we don't build it?" | Priority and urgency |
| **Shape** | "Walk me through the happy path -- what happens first, then what, then what?" | Scope and boundaries |
| **Edge** | "What's the weirdest way someone could use this? What could go hilariously wrong?" | Failure modes and guardrails |
| **Trade-off** | "Would you rather have this working perfectly for 10 people, or working okay for 1000?" | Quality vs. reach, MVP definition |
| **Neighbor** | "Does this replace something we already have, or does it live next to it?" | Integration points, duplication risk |

Ask exactly three questions in one breath. Not one-by-one. Let the human answer in any order.

### 3. Listen for the Route Signal

As the conversation unfolds, watch for these signals:

| Signal | Route To | Why |
|--------|----------|-----|
| "I'm not sure if it's possible" or technical unknowns | **Feasibility Spike** | Need evidence before planning |
| Clear scope, known tech, ~1-5 phases | **Phase Plan** | Ready for structured execution |
| Great idea, but not now | **Backlog** | Park it with enough context to revive |
| Multiple competing visions | **Design Sketch** | Need visual comparison to decide |
| "Just try it, we'll see" | **Quick Task** | Small enough to just do |
| Needs research or comparison | **Research Isolator** | External knowledge needed first |

### 4. Shape and Route

Once the idea is clear enough, produce a shaped artifact:

```markdown
# Idea: {name}

## The Spark
{1-2 sentence original idea, in the human's words}

## What We Learned
{3-5 bullet insights from the conversation}

## Shaped Scope
- **Who it's for:** {target user}
- **What it does:** {core behavior}
- **What it doesn't do:** {explicit out-of-scope}
- **Success looks like:** {observable outcome}

## Recommended Route: {spike | plan | backlog | sketch | quick-task | research}
## Rationale: {why this route}
## Next Step: {specific action to take}
```

### 5. Handle "I Don't Know Yet"

If the human can't answer after two rounds of probing, that's fine. Some ideas need to percolate. Offer these options:

- **Seed it** -- plant it with trigger conditions so it resurfaces when context aligns
- **Sketch it** -- throw together a rough visual to make it tangible
- **Sleep on it** -- save to notes with `#idea` tag and revisit next session

Never force a route. An idea that's not ready isn't a failure -- it's a seed.

## Key Patterns

### The Yes-And Pattern
When the human offers a partial idea, build on it instead of poking holes:
> "Yes! And if we did that, users could also [extension]. How does that feel?"

### The Either-Or Pattern
When the idea could go two ways, present both as valid and let them choose:
> "I see two shapes for this. Option A: [concise vision]. Option B: [alternative]. Both are real -- which feels more like what you imagined?"

### The Simplify Pattern
When scope creeps in, reel it back gently:
> "Love it. Let's find the smallest version that still delivers the thrill. What's the one thing that makes this worth building?"

### The Unstick Pattern
When conversation stalls, try a perspective shift:
> "If you were explaining this to a smart 12-year-old, what would you say?"
> "If this were a physical object in a store, what would the packaging say?"

## Output Format

Writes `IDEA-{slug}.md` to the colony's notes directory, or outputs directly to conversation for ephemeral exploration.

## Examples

### Example 1: Vague feature idea

```
Human: "I want the app to feel alive, not static"
Explorer: "Ooh, I love that instinct. Let me make sure I hear you -- you want users to feel like the app is responding to them in real-time, not just loading pages. Is that the vibe?
Three quick questions:
1. When you say 'alive,' are you thinking movement, or are you thinking the content changes based on what they do?
2. Is there an app you've used where you felt that way -- that we could use as a reference?
3. Would you rather start with one spot that feels magical, or a subtle aliveness everywhere?"
```

### Example 2: Competing directions

```
Human: "We need a dashboard. Or maybe a command center. Or both?"
Explorer: "Two different beasts! A dashboard shows you what's happening. A command center lets you do something about it.
Which matters more right now -- seeing the picture, or taking action from it?
Because here's the thing: a great dashboard that you can gradually grow action buttons on... that might give you both without building two things."
```

### Example 3: Routed outcome

```markdown
# Idea: Real-time Collaboration Signals

## The Spark
"Make the app feel alive -- show when others are online, typing, looking at the same thing"

## What We Learned
- This is about presence, not chat
- Target is small teams (2-8 people) working on shared docs
- Slack's "typing" indicator is the closest reference
- Want it subtle, not distracting

## Shaped Scope
- **Who it's for:** Small teams collaborating on shared documents
- **What it does:** Shows presence indicators (who's viewing, who's editing) in real-time
- **What it doesn't do:** Chat, comments, notifications (those exist elsewhere)
- **Success looks like:** Users notice when a teammate joins their document without anyone telling them

## Recommended Route: spike
## Rationale: WebSocket infrastructure choice is unknown -- need to validate feasibility and perf impact before committing to a plan
## Next Step: Run feasibility-spiker on WebSocket vs SSE vs polling for presence signals
```
