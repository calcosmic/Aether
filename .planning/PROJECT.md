# Aether Repair & Stabilization

## What This Is

A self-managing development assistant using ant colony metaphor that prevents context rot. Users install it, it guides them through work with clear commands, tells them when to clear context, and maintains state across sessions. The colony learns from each phase and improves over time.

**Current State:** v1.0 shipped. All 46 requirements verified PASS. The system works end-to-end: users can start a colony, set context via pheromones, plan and build phases, track progress, archive completed colonies to chambers, and restore context in new sessions. XML exchange enables cross-colony transfer.

## Core Value

If everything else fell away, Aether's essential value is:
- **Context preservation** — prevents context rot across Claude Code sessions
- **Clear workflow guidance** — tells users what command to run next
- **Self-improving** — learns from each phase via pheromones/instincts

## Requirements

### Validated

- ✓ **Command Infrastructure** — v1.0
  - ✓ CMD-01: /ant:lay-eggs starts new colony with pheromone preservation
  - ✓ CMD-02: /ant:init initializes after lay-eggs
  - ✓ CMD-03: /ant:colonize analyzes existing codebase
  - ✓ CMD-04: /ant:plan generates project plan
  - ✓ CMD-05: /ant:build executes phase with worker spawning
  - ✓ CMD-06: /ant:continue verifies, extracts learnings, advances phase
  - ✓ CMD-07: /ant:status shows colony dashboard
  - ✓ CMD-08: All commands find correct files (no hallucinations)

- ✓ **Visual Experience** — v1.0
  - ✓ VIS-01: Swarm display shows ants working
  - ✓ VIS-02: Emoji caste identity visible
  - ✓ VIS-03: Colors for different castes
  - ✓ VIS-04: Progress indication during builds
  - ✓ VIS-05: Ant-themed stage banners
  - ✓ VIS-06: GSD-style phase transitions

- ✓ **Context Rot Prevention** — v1.0
  - ✓ CTX-01: Session state persists across /clear
  - ✓ CTX-02: Clear next command guidance
  - ✓ CTX-03: Context document for next session

- ✓ **State Integrity** — v1.0
  - ✓ STA-01: COLONY_STATE.json updates correctly
  - ✓ STA-02: No file path hallucinations
  - ✓ STA-03: Files in correct repositories

- ✓ **Pheromone System** — v1.0
  - ✓ PHER-01: FOCUS signal works
  - ✓ PHER-02: REDIRECT signal works
  - ✓ PHER-03: FEEDBACK signal works
  - ✓ PHER-04: Auto-injection of learned patterns
  - ✓ PHER-05: Instincts applied to builders/watchers

- ✓ **Colony Lifecycle** — v1.0
  - ✓ LIF-01: /ant:seal creates Crowned Anthill milestone
  - ✓ LIF-02: /ant:entomb archives colony to chambers
  - ✓ LIF-03: /ant:tunnels browses archived colonies

- ✓ **Advanced Workers** — v1.0
  - ✓ ADV-01: /ant:oracle deep research (RALF loop)
  - ✓ ADV-02: /ant:chaos resilience testing
  - ✓ ADV-03: /ant:archaeology git history analysis
  - ✓ ADV-04: /ant:dream philosophical wanderer
  - ✓ ADV-05: /ant:interpret validates dreams

- ✓ **XML Integration** — v1.0
  - ✓ XML-01: Pheromones via XML format
  - ✓ XML-02: Wisdom exchange via XML
  - ✓ XML-03: Registry via XML

- ✓ **Session Management** — v1.0
  - ✓ SES-01: /ant:pause-colony saves state
  - ✓ SES-02: /ant:resume-colony restores context
  - ✓ SES-03: /ant:watch shows live visibility

- ✓ **Colony Documentation** — v1.0
  - ✓ DOC-01: Phase learnings extracted
  - ✓ DOC-02: Colony memories persist
  - ✓ DOC-03: Progress tracked with ant metaphors
  - ✓ DOC-04: Handoff documents use ant themes

- ✓ **Error Handling** — v1.0
  - ✓ ERR-01: No 401 authentication errors
  - ✓ ERR-02: No infinite spawn loops
  - ✓ ERR-03: Clear error messages

### Active

(None — next milestone will define new requirements)

### Out of Scope

- Model-per-caste routing — configuration exists but effectiveness unverified
- YAML command generator — 13,573 lines duplicated across .claude/ and .opencode/, generator unused
- Offline/mobile support — CLI-only tool

## Context

Shipped v1.0 with ~5,000 lines of shell (aether-utils.sh + utils/), 34 Claude Code commands, 33 OpenCode commands.
Tech stack: Bash, jq, xmllint/xmlstarlet, Node.js CLI wrapper.
Full e2e test suite at tests/e2e/ with 12 test scripts and automated requirements matrix.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Repair first, features later | Don't add features to broken foundation | ✓ Good — systematic phases worked |
| Proxy verification strategy | Can't run slash commands from bash | ✓ Good — static analysis + subcommand execution |
| JSON stays primary, XML for exchange | XML too verbose for internal storage | ✓ Good — clean separation |
| Bash 3.2 compatibility | macOS ships bash 3.2 | ✓ Good — file-based results instead of associative arrays |
| Pheromone dual-write (JSON + constraints.json) | Backward compatibility | ✓ Good — old commands still work |
| Seal-first enforcement for entomb | Prevent archiving incomplete colonies | ✓ Good — belt-and-suspenders check |
| Time-agnostic session restore | No 24h staleness, identical restore regardless of gap | ✓ Good — simpler and more reliable |

## Constraints

- **Must work in Claude Code** — primary platform
- **Visual simplicity** — no loads of terminal text
- **Reliability first** — working > feature-rich
- **Self-contained** — minimal external dependencies
- **UI Style** — GSD-style stage banners with ant-themed names

---

*Last updated: 2026-02-18 after v1.0 milestone*
