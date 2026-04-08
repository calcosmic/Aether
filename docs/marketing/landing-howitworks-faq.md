# Landing Page Content: How It Works + FAQ

> Ready for web developer implementation. Includes layout notes, content, and structure.
> Tone: plain English, confident but honest about v1.0.0. No jargon without explanation.

---

## Section: How It Works

<!-- LAYOUT NOTE: 3-column horizontal flow on desktop. Stacks vertically on mobile.
Each step is a card with: number badge (top-left), icon area (center, 64x64),
title (bold, heading level), description (2-3 sentences, regular weight).
Connect the cards with a subtle arrow or dotted line between them.
Consider a subtle animation where each card fades in on scroll. -->

### Step 1: Init -- Tell the colony what you want

Describe your goal in plain English. "Build a REST API for task management." "Create a React dashboard with charts." Whatever you are building, just say it. The colony listens, sets up its nest in your project, and gets every worker aligned to that single goal from the start.

You do not need to write a spec, break down tasks, or plan anything yourself. The colony handles all of that.

<!-- VISUAL NOTE: A text input or chat bubble icon. Something that suggests "speaking" or "typing a goal."
Color suggestion: the project purple (#7B3FE4). Could also show a terminal snippet:
  /ant:init "Build a REST API for task management" -->

### Step 2: Plan -- The colony maps the path

Aether analyzes your goal and breaks it into logical phases. Phase 1 might be project setup and database design. Phase 2 might be authentication. Each phase has clear deliverables, and nothing happens out of order -- the colony builds a solid foundation before it builds walls.

You can review the plan, adjust it, or let it stand. Either way, every worker knows exactly what needs to happen and in what order.

<!-- VISUAL NOTE: A roadmap or blueprint icon. Something that suggests "structure" or "phases."
Color suggestion: a lighter purple or blue (#2563EB).
Could show a mini phase list:
  Phase 1: Foundation
  Phase 2: Auth
  Phase 3: Features
  Phase 4: Polish -->

### Step 3: Build -- Workers execute in parallel waves

This is where the magic happens. Aether deploys specialized workers in parallel -- Builders write code, Probes write tests, Watchers verify quality. They self-organize around your goal, following any guidance you have set. When one phase is done and verified, the colony advances to the next automatically.

You can steer with simple signals (like "focus on security" or "avoid raw SQL") or turn on autopilot and let the colony run. It pauses when something needs your attention and resumes when you are ready.

<!-- VISUAL NOTE: Multiple small ant/worker icons moving in parallel, converging on a target.
Color suggestion: green tones for "action" (#16a34a).
Could show a terminal snippet:
  /ant:run
  [Phase 3] Building... 4 workers deployed
  [Phase 3] Verified. Advancing. -->

---

## Section: FAQ

<!-- LAYOUT NOTE: Accordion-style expandable questions on desktop.
On mobile, stack vertically with all answers visible (or keep accordion).
Group into two columns on wide screens if space allows.
Each question is bold, each answer is regular weight, 2-4 sentences max.
Consider adding a subtle search/filter bar above the FAQ. -->

### What is Aether?

Aether is an open-source tool that helps you build software using a team of AI workers. Instead of one AI trying to do everything, Aether deploys 24 specialized workers -- some write code, some test it, some research problems, some monitor quality. They organize themselves around your goal and work together in parallel, like a real ant colony.

### How is this different from other AI coding tools?

Most AI coding tools give you one assistant that plans, writes code, and checks its own work in a loop. Aether takes a different approach inspired by nature: it uses multiple specialized workers, each with a distinct role, that coordinate through signals instead of being micromanaged. The result is parallel execution, built-in quality checks, and a memory system that learns from every build so your next project starts smarter.

### Do I need to be technical to use it?

You need basic comfort with a terminal (command line) and an AI coding assistant like Claude Code or OpenCode. You do not need to know how to write code yourself, though it helps to understand what you want built. The colony handles the technical details -- you provide the goal and the guidance.

### What is the ant colony metaphor about?

Real ant colonies do not have a central brain telling every ant what to do. Each ant has a specialized role, and they communicate through chemical signals to self-organize around the colony's goals. Aether works the same way: specialized workers (Builder, Watcher, Scout, and 21 others) communicate through "pheromone signals" to coordinate their work. No single worker knows the entire plan -- they just know their role and respond to the signals around them. The result is a system that is more resilient, more parallel, and less reliant on any one component.

### What are pheromone signals?

Pheromone signals are a simple way to guide the colony without micromanaging. There are three types: **FOCUS** tells workers to pay extra attention to something (like "be careful with database migrations"), **REDIRECT** sets a hard constraint (like "never use raw SQL"), and **FEEDBACK** offers gentle course correction (like "keep the code simple"). You emit a signal before a build, and every worker in the next wave sees it and adjusts their behavior accordingly. Signals expire automatically, so stale guidance does not linger.

### What does autopilot mode do?

Autopilot mode (`/ant:run`) chains the entire build cycle together automatically. Instead of running each command by hand, you turn on autopilot and the colony plans, builds, verifies, and advances through every phase on its own. It pauses -- not crashes -- when something needs your attention, like a test failure or a decision it cannot make on its own. Fix the issue, tell it to continue, and it picks up where it left off. You can also set limits, like "run at most 2 phases then stop," to stay in control.

### Is it ready for production use?

Honestly, it is v1.0.0. It works well for the use cases it has been tested on, and people are actively building real projects with it. But it is new, the learning curve is real, and you may encounter rough edges. Think of it as a solid foundation that is being actively improved. If you are comfortable with early-stage tools and do not mind occasional workarounds, it is absolutely usable today. If you need something battle-tested with years of production history, check back in a few months.

### How does learning and memory work across sessions?

Every time the colony builds something, it records observations about what worked and what did not. Those observations are scored for trustworthiness and become "instincts" -- learned patterns that inform future work. The highest-confidence instincts get promoted to a wisdom file (QUEEN.md) that persists on your machine. The very best insights flow into a "Hive Brain" that shares across all your projects, so something learned building one API can help when you build the next one. When you start a new session, the colony reconstructs its full context from these files -- no need to re-explain your project.

### What languages and platforms does it support?

Aether itself is a Go binary that runs on Linux, macOS (including Apple Silicon), and Windows. It works with Claude Code and OpenCode as its AI assistant backends right now, with more platforms planned. The workers can build projects in any programming language -- Go, Python, TypeScript, Rust, Java, you name it. The colony does not care what language your project uses; it cares about the goal you set.

### Is it free and open source?

Yes. Aether is MIT licensed, which means you can use it, modify it, and distribute it however you want, including in commercial projects. The source code is on GitHub, the binary is free to download, and there are no paid tiers, feature gates, or usage limits. If you find it valuable, you can support the project through GitHub Sponsors, but that is entirely optional.

### What if something goes wrong during a build?

The colony has built-in safeguards. A Watcher worker monitors every build for quality. If tests fail, the colony pauses instead of pushing forward. If a blocker is detected, it flags it and waits for you. For stubborn bugs, you can deploy a "swarm" -- four parallel investigators that cross-reference findings and rank solutions. The colony also maintains a full event history, so you can always look back at what happened and why.

---

<!-- LAYOUT NOTE: Below the FAQ, consider a CTA row:
  Left: "Ready to start?" with a /ant:lay-eggs code snippet
  Right: Primary button "Get Started" linking to GitHub
  Below: Secondary links "Read the Docs" | "Join the Discussion" -->

<!-- ADDITIONAL FAQ QUESTION (BONUS, consider if space allows): -->

### Can I use Aether with an existing project?

Yes. Aether has a "colonize" step that scans your existing codebase, catalogs the structure, identifies patterns, and flags potential hazards before any work begins. The colony maps your project's territory first, then builds on top of what is already there. This works for codebases of any size, though larger projects will benefit from more focused goal-setting.
