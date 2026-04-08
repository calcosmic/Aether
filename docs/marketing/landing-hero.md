# Landing Page -- Hero Section and Features Grid

**For:** aetherantcolony.com
**Audience:** Developers and technical leads frustrated with single-agent AI tools
**Tone:** Confident but honest. Plain English. No hype words.

---

## Hero Section

### Layout Notes

- Full viewport height. Dark background (near-black or deep purple-black).
- Center-aligned text. Generous whitespace above and below.
- Animated colony illustration or subtle particle background (optional -- do not block text).
- Single primary CTA button below the subheadline. Secondary link below it.
- Badge above headline: "Open Source -- MIT License -- v1.0.0"

### Headline

```
Stop herding cats.
Start a colony.
```

### Subheadline

```
Aether is an open-source AI agent orchestration tool modeled on ant colonies.
24 specialized workers self-organize around your goal -- no prompt engineering, no micromanagement.
You give the goal. The colony figures out the rest.
```

### Primary CTA

```
[Star on GitHub]
```

Link to: https://github.com/calcosmic/Aether

### Secondary Link

```
or install in one command: go install github.com/calcosmic/Aether@latest
```

This line sits directly below the CTA button. Smaller text, monospace for the install command. The install command should be selectable/copyable (click to copy behavior).

### Honesty Badge (below secondary link)

```
v1.0.0 -- fully functional, actively evolving.
```

Keep this small and understated. We are not hiding that this is early. We are proud of what ships today and transparent about where it is going.

---

## Features Grid

### Layout Notes

- 6 cards in a 3x2 grid (desktop), stacking to 2x3 (tablet) and 1x6 (mobile).
- Each card: icon or illustration on top, title, 2-3 sentence description.
- Cards should have a subtle border or background differentiation -- no heavy shadows.
- Order matters: lead with the most differentiating feature (Colony Architecture), end with the most practical (Context Continuity).

---

### Card 1: Colony Architecture

**Icon suggestion:** Ant colony cross-section or network graph

**Title:**
```
24 workers. One goal. No micromanagement.
```

**Description:**
```
Aether does not give you one AI agent that tries to do everything. It gives you 24
specialized workers -- builders write code, watchers verify quality, scouts research
the unfamiliar, trackers hunt bugs, and archaeologists dig through git history.
They self-organize around your goal, divide labor, and work in parallel. Like a real
colony finding the shortest path to food.
```

---

### Card 2: Pheromone Signals

**Icon suggestion:** Concentric ripples or signal waves

**Title:**
```
Guide the colony with signals, not prompts.
```

**Description:**
```
Instead of rewriting prompts every time something goes wrong, emit a signal.
FOCUS tells workers where to pay extra attention. REDIRECT sets hard constraints --
things that will break if ignored. FEEDBACK gently adjusts the colony's approach.
Workers sense these signals and adapt automatically. The colony responds to
guidance the way real ants respond to chemical trails.
```

---

### Card 3: Structural Learning

**Icon suggestion:** Upward arrow or ascending steps

**Title:**
```
Every project makes the colony smarter.
```

**Description:**
```
Aether does not just complete tasks -- it learns from them. Raw observations become
trust-scored instincts. High-confidence instincts promote to a shared wisdom file.
The best insights flow into a Hive Brain that carries across projects. The more you
use Aether, the better it gets -- not because you configured it, but because it
remembered what worked.
```

---

### Card 4: Autopilot Mode

**Icon suggestion:** Play button or circular arrow

**Title:**
```
Set a goal. Go do something else.
```

**Description:**
```
Autopilot chains the build-verify-advance loop across multiple phases automatically.
It is not blind automation -- it pauses intelligently when something needs your
attention: test failures, security concerns, or blockers it cannot resolve. Fix the
issue, resume, and it picks up where it left off. Five commands from zero to shipped.
```

---

### Card 5: Skills System

**Icon suggestion:** Toolbox or interconnected nodes

**Title:**
```
28 skills that make workers domain-smart.
```

**Description:**
```
Workers start capable. Skills make them expert. Aether injects domain knowledge
directly into workers before they spawn -- everything from Go best practices to API
design patterns to security hardening. Ten colony skills for how Aether itself works,
plus 18 domain skills covering the frameworks and languages your project actually uses.
```

---

### Card 6: Context Continuity

**Icon suggestion:** Chain links or unbroken line

**Title:**
```
Your colony survives the weekend.
```

**Description:**
```
Close your laptop on Friday. Open it on Monday. Run one command and the colony
remembers everything -- what phase it is on, what signals are active, what instincts
it has learned, what files it has created. No re-explaining. No pasting context from
old conversations. The colony state lives in your repo, and a compact context capsule
reconstructs the full picture in seconds.
```

---

## Below the Fold (suggested next section)

After the features grid, the next section should be a concrete example or demo.
The README's "From Zero to Shipped" walkthrough is a strong candidate -- it shows
the actual commands and output a user would experience building a real project.

Consider a collapsible/expandable terminal-style walkthrough or a short animated demo.

---

## Copy Notes for the Developer

- Headline is two short lines. Do not combine into one. The line break is intentional -- it creates rhythm.
- Subheadline is three sentences. Each sentence serves a purpose: what it is, how it works, what the result feels like.
- Feature card titles are written as benefit statements, not feature names. "24 workers. One goal." tells you what you get. "Colony Architecture" tells you what it is. Lead with the benefit.
- All copy avoids jargon. "Self-organize" is explained in context. "Pheromone signals" is compared to chemical trails. "Hive Brain" is described as cross-project memory.
- The honesty badge is not a disclaimer. It is a trust signal. Users who discover v1.0.0 after the fact will respect the transparency.
- Tone consistency: this copy matches the voice established in the Twitter thread and Discord launch copy -- direct, conversational, slightly opinionated, never salesy.
