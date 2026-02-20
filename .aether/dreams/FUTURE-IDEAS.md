# Aether Future Ideas

Organized from dream journals (2026-02-11 through 2026-02-16) and the in-conversation display plan. These are ideas worth revisiting — not commitments.

---

## Memory & Learning

### Cross-Session Memory Persistence
The colony forgets everything between sessions. Instincts, learnings, and patterns discovered by one session die when it ends. Each new session rediscovers wave-based spawning independently.

**The opportunity:** Wire completion reports and instincts into session initialization so new colonies inherit what past colonies learned. The data already exists in completion-report.md files — nothing reads them at startup.

*Source: Dream 2026-02-11 #5, Dream 2026-02-15 #7*

### Contextual Consumption
The colony produces enormous amounts of data about itself — dreams, telemetry, tests, logs — but consumes little of it. Make the Queen read dreams. Make builders consult telemetry. Make tests meaningful.

**The opportunity:** Connect existing data producers to decision-making. The infrastructure is built; the wiring is missing.

*Source: Dream 2026-02-15 closing reflection*

### Calibration Loops
Tighter feedback between what the colony thinks it is and what it actually is. Dreams observe gaps but the colony doesn't systematically close them.

**The opportunity:** Periodic self-audits where the colony measures its own fidelity — are workers following the scripts? Are configured systems actually running?

*Source: Dream 2026-02-16 closing reflection*

---

## Colony Architecture

### Multi-Colony Democracy
The colony is designed around a single Queen controlling everything. Future plans mention multiple colonies working simultaneously. True parallel colonies would require the Queen to become a coordinator, not a commander.

**The opportunity:** Evolve the monarchy into a federation where multiple colony sessions can work on different goals simultaneously with shared state.

*Source: Dream 2026-02-16 #5*

### Iron Law Runtime Enforcement
Iron Laws (TDD, verification-before-approval) are text instructions, not runtime constraints. A Builder can skip TDD and still report success. The Watcher validates results, not process.

**The opportunity:** Build runtime verification that checks process compliance, not just output quality. Distinguish "followed TDD" from "wrote code then added tests after."

*Source: Dream 2026-02-11 #6*

### Model Routing Verification
Model-per-caste configuration exists (model-profiles.yaml) but execution is unproven. ANTHROPIC_MODEL may not be inherited by spawned workers.

**The opportunity:** Verify that `/ant:verify-castes` Step 3 actually spawns a test worker with the correct model. Prove the routing works, don't just configure it.

*Source: Dream 2026-02-14 #3*

---

## User Experience

### In-Conversation Colony Display
Make colony activity visible inside Claude's conversation output, not in separate tmux windows. A compact, emoji-based display showing active ants, progress bars, and tool counts.

**Status:** Detailed implementation plan exists with exact line numbers, function code, and command update map. Ready to build.

**The opportunity:** Replace ANSI-based terminal display with plain-text emoji display for 6 agent-spawning commands.

*Source: in-conversation-display-plan.md (full spec)*

### Ritual Density Reduction
build.md has 1,043 lines with 20+ steps and sub-steps. The gap between design and execution is inevitable at this complexity.

**The opportunity:** Simplify command scripts. When adding steps, first consider if existing steps can be consolidated.

*Source: Dream 2026-02-16 #2*

---

## Tooling

### YAML Command Generator
13,573 lines duplicated between .claude/commands/ and .opencode/commands/. A YAML-based generation system was designed (src/commands/README.md) but never built.

**The opportunity:** Define commands once in YAML, generate platform-specific versions. Would eliminate manual sync and the lint:sync verification step.

*Source: Dream 2026-02-14 #4, Dream 2026-02-11 #4*

### XML Eternal Memory
XML support for pheromone exchange, queen-wisdom storage, and multi-colony registry. Six phases planned but work hadn't started as of Feb 2026.

**Risk noted:** May encounter the same fate as model routing — beautifully built but archived when complexity overwhelms benefit. Shell convenience vs XML verbosity is a real tension.

*Source: Dream 2026-02-16 #3*

### Archaeologist Confidence Thresholds
The Archaeologist Ant assumes git histories are clear. When history is ambiguous, it should report uncertainty rather than guessing.

**The opportunity:** Add confidence scoring to archaeology output. "I don't know why this exists" is more valuable than a confident wrong answer.

*Source: Dream 2026-02-11 #2*

---

## Quality

### Test Coverage Audit
Tests exist but purpose is unclear for some (cli-telemetry.test.js, cli-override.test.js). Passing tests may be false confidence.

**The opportunity:** For each test file, document what behavior it verifies and what it misses. Ensure tests catch real problems.

*Source: Dream 2026-02-15 #3*

### Documentation-Reality Alignment
Handoff documents claim completion while TO-DOs hold undone work. The story told by completion docs is cleaner than reality.

**The opportunity:** Cross-reference completion claims against the actual backlog before declaring victory.

*Source: Dream 2026-02-16 #6*

---

*Compiled: 2026-02-20 | Update when ideas are implemented or new dreams surface insights*
