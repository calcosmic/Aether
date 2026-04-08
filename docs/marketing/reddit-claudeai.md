# Reddit Post: r/ClaudeAI

**Title:** Built a multi-agent colony system for Claude Code -- 24 specialized workers that self-organize around your goals

---

**TL;DR:** Aether adds 24 specialized agent workers to Claude Code that self-organize around your project goals. Builders write code, Watchers verify, Scouts research, Trackers hunt bugs -- all in parallel. Uses pheromone signals to steer behavior instead of rewriting prompts. Memory compounds across sessions. MIT licensed, just hit v1.0.0.

GitHub: https://github.com/calcosmic/Aether

---

## What this actually is

If you've been using Claude Code for a while, you've probably hit the same ceiling I did: one agent trying to be everything. It plans, it codes, it tests, it reviews -- and when things get complex, it starts dropping context or losing track of what it was doing.

Aether splits that work across 24 specialized workers, each with a defined role:

- **Builder** -- writes code following TDD
- **Watcher** -- independent verification and quality checks
- **Scout** -- researches unfamiliar territory before builders touch it
- **Tracker** -- traces and hunts bugs
- **Archaeologist** -- excavates git history for context
- **Oracle** -- deep autonomous research via iterative loops
- **Chaos** -- resilience and edge case testing
- **Chronicler** -- documentation generation
- ...and 16 more

They work in waves. A builder writes code, a watcher verifies it, a probe writes tests. If something is unfamiliar, a scout investigates first. No single agent is trying to do everything.

## How it works with Claude Code

Aether is a Go binary that installs alongside Claude Code. It adds 45 slash commands to your Claude session. The workflow looks like:

```
/ant:init "Build a REST API with auth and task tracking"
/ant:plan
/ant:focus "payment flow security"
/ant:redirect "No raw SQL -- parameterized queries only"
/ant:build 1
/ant:continue
/ant:build 2
...
/ant:seal
```

Each `/ant:build` spawns a wave of parallel workers. Each `/ant:continue` verifies the output, extracts learnings, and advances to the next phase.

Or just use autopilot:

```
/ant:run
```

It chains the build-verify-advance loop automatically and pauses when something needs your attention.

## The pheromone system (the part I'm most excited about)

Instead of rewriting your system prompt every time you want to change agent behavior, you emit pheromone signals:

- **FOCUS** -- "Pay extra attention to database migrations this phase"
- **REDIRECT** -- "Never use raw SQL -- parameterized queries only"
- **FEEDBACK** -- "Code is getting too abstract, prefer simple implementations"

Every worker in the next build wave sees these signals and adjusts behavior accordingly. Signals expire at phase end by default, or you can set wall-clock TTLs.

The colony also emits its own signals. After each phase, it auto-generates FEEDBACK about what worked and what failed. If the same error pattern recurs across builds, it auto-emits a REDIRECT.

This means the colony gets smarter about your project over time without you manually tuning prompts.

## Memory that actually persists

Here's what frustrated me most about Claude Code sessions: context loss. You /clear, start a new session, and have to re-explain everything.

Aether solves this with a multi-stage memory pipeline:

1. **Raw observations** from each build
2. **Trust-scored instincts** (0.2-1.0 confidence)
3. **QUEEN.md wisdom** (instincts scoring 0.80+ get promoted)
4. **Hive Brain** (the best insights cross to other projects)

When you run `/ant:resume` after a session break, the colony reconstructs its full context from state files, active signals, and accumulated instincts. You don't re-explain anything.

## Context for Claude Code users specifically

- **Not a replacement for Claude Code** -- it builds on top of it. Claude Code is the runtime; Aether is the orchestration layer.
- **All colony state is local** -- nothing leaves your machine. State lives in `.aether/` in your repo and `~/.aether/` for cross-project wisdom.
- **Works with your existing projects** -- run `/ant:colonize` on an existing codebase and it maps the structure before building.
- **OpenCode support too** -- if you use OpenCode, Aether works there as well.

## Honest take

This is v1.0.0. The architecture is solid and it works well for the projects I've used it on, but it's new. The pheromone model is a different way of thinking about agent coordination and there's a learning curve. The CLI-only interface won't be for everyone.

What I can say is: once you get the mental model, it genuinely changes how you work with Claude Code. Having a Builder that writes code while a Watcher independently verifies it, while a Scout researches the unfamiliar parts, is a fundamentally different experience than one agent trying to juggle everything.

## Looking for feedback

If you try it, I'd especially love to hear:

- Does the pheromone signal model make sense, or does it feel overengineered?
- Are the 24 castes the right breakdown, or is it too many/too few?
- What's missing for your Claude Code workflow?
- Any interest in contributing skills or castes?

Happy to answer questions about the architecture, the pheromone system, or anything else.

https://github.com/calcosmic/Aether

https://aetherantcolony.com
