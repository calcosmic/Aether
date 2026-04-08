# Reddit Post: r/LocalLLaMA

**Title:** I built an open-source multi-agent framework modeled on ant colonies -- and it actually works

---

**TL;DR:** Aether is a MIT-licensed, Go-based multi-agent framework where 24 specialized workers self-organize around your goal using pheromone signals instead of prompt engineering. Runs locally, memory compounds across projects, autopilot mode included. v1.0.0 just dropped.

GitHub: https://github.com/calcosmic/Aether

---

## The problem I was trying to solve

I kept running into the same wall with AI coding tools: "agents" that are really just one LLM in a loop doing everything. Plan, execute, check, repeat. That's not a team -- that's one person doing laps.

The multi-agent frameworks I tried (CrewAI, AutoGen, LangGraph) all had the same issues: Python-only, coordination felt manual, and nothing persisted between sessions. I'd spend more time wiring up agent interactions than actually building.

## What Aether does differently

Instead of a central orchestrator micromanaging agents, I modeled it on how real ant colonies work. No single ant knows the whole plan. They communicate through chemical signals, specialize in different tasks, and self-organize around the colony's goal.

In practice, that looks like:

- **24 specialized workers** -- Builder writes code, Watcher verifies, Scout researches, Tracker hunts bugs, Archaeologist digs through git history. Each has a specific role and caste.
- **Pheromone signals instead of prompt engineering** -- You emit FOCUS (pay attention here), REDIRECT (don't do this), and FEEDBACK (adjust your approach) signals. Every worker in the next wave sees them automatically.
- **Memory that compounds** -- Learnings from one build become instincts. High-confidence instincts promote to a wisdom file (QUEEN.md). The best insights cross into a "Hive Brain" that shares across all your projects.
- **Autopilot mode** -- Set a goal, generate a plan, emit your constraints, and let `/ant:run` handle the build-verify-advance loop across phases. It pauses when something needs your attention.

## Why you might care (for this sub specifically)

- **Fully local** -- Go binary, no cloud dependency. All colony state stays on your machine.
- **MIT licensed** -- do whatever you want with it
- **Works with Claude Code and OpenCode** -- 45 slash commands that plug into your existing workflow
- **Single binary install** -- `go install github.com/calcosmic/Aether@latest` or grab a pre-built release
- **Cross-platform** -- Linux, macOS (including Apple Silicon), Windows
- **28 skills** inject domain knowledge into workers without bloating prompts

## Honest limitations

- v1.0.0 -- this is new. It works for the use cases I've tested, but your mileage may vary.
- Only works with Claude Code and OpenCode right now (more platforms on the roadmap)
- The learning curve is real. The pheromone system is powerful but it's a different mental model than what most people are used to.
- No visual dashboard yet -- everything is CLI/terminal based

## A quick example

```
aether install
cd my-project
/ant:lay-eggs
/ant:init "Build a REST API for task management"
/ant:plan
/ant:focus "database migrations -- use versioned migrations"
/ant:redirect "No raw SQL in application code"
/ant:run
```

Five commands from blank directory to shipped code. The colony generates a phased plan, spawns parallel workers for each phase, verifies the output, and advances. If something breaks, autopilot pauses and waits for you.

## Why Go?

Honestly, because I wanted a single binary with no runtime dependencies. Python frameworks require venvs, dependency management, and tend to be slower for the orchestration layer. Go compiles to one file, runs anywhere, and the concurrency model maps naturally to parallel worker spawning.

## Looking for feedback on

- What platforms should I support next? (Cursor support is the most requested so far)
- Are the 24 castes the right set, or are there obvious gaps?
- Would a web-based colony dashboard be useful, or is CLI sufficient?
- Anyone interested in contributing custom skills or castes?

Full docs, architecture details, and comparison table (vs CrewAI, AutoGen, LangGraph) are in the README. Happy to answer questions here too.

https://github.com/calcosmic/Aether

https://aetherantcolony.com
