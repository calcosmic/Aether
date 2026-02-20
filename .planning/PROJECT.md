# Aether

## What This Is

A self-managing development assistant using ant colony metaphor that prevents context rot. Users install it, it guides them through work with clear commands, tells them when to clear context, and maintains state across sessions. The colony learns from each phase and improves over time. As of v1.2, the foundation is hardened — all documented bugs fixed, distribution chain cleaned up, error codes standardized, and lock safety guaranteed with 446 tests passing.

**Current State:** v1.3 in progress. The Great Restructuring — comprehensive architecture simplification across agent definitions, template system, team coordination, distribution pipeline, queen architecture, and wisdom inheritance.

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

- ✓ **Noise Reduction** — v1.1
  - ✓ NOISE-01: Human-readable bash descriptions on all 34 commands
  - ✓ NOISE-02: ~40% call consolidation in high-complexity commands
  - ✓ NOISE-03: Version check cached per session with TTL
  - ⚠ NOISE-04: Session IDs removed from most output (3 session-management commands retain for debugging)

- ✓ **Visual Identity** — v1.1
  - ✓ VIS-01: Caste emojis next to ant names in worker output
  - ✓ VIS-02: "Next Up" block on every command completion
  - ✓ VIS-03: /ant:status progress bar for phase/task completion
  - ✓ VIS-04: Consistent ━━━━ banner and divider style
  - ✓ VIS-05: Caste emoji unified to single caste-system.md source

- ✓ **Build Progress** — v1.1
  - ✓ PROG-01: Spawn announcements before parallel waves
  - ✓ PROG-02: Worker completion lines with caste emoji, name, task, tool count
  - ✓ PROG-03: Swarm display gated to tmux-only
  - ✓ PROG-04: Task descriptions include caste emoji and ant name

- ✓ **Distribution Reliability** — v1.1
  - ✓ DIST-01: "Already up to date" detection works correctly
  - ✓ DIST-02: Atomic .update-pending sentinel for partial failure recovery

- ✓ **Distribution Chain** — v1.2
  - ✓ DIST-01: update-transaction.js reads from hub/system/ not hub root
  - ✓ DIST-02: EXCLUDE_DIRS covers commands, agents, rules inside hub/system/
  - ✓ DIST-03: Dead duplicates removed (.aether/agents/, .aether/commands/)
  - ✓ DIST-04: caste-system.md added to sync allowlist
  - ✓ DIST-05: Phantom planning.md removed from allowlists
  - ✓ DIST-06: Old 2.x npm versions deprecated on registry

- ✓ **Lock Safety** — v1.2
  - ✓ LOCK-01: No lock deadlocks on jq failure in flag operations
  - ✓ LOCK-02: Trap-based lock cleanup fires on all exit paths
  - ✓ LOCK-03: atomic-write backup race fixed
  - ✓ LOCK-04: context-update uses file locking

- ✓ **Error Handling Standardization** — v1.2
  - ✓ ERR-01: json_err fallback handles error codes correctly
  - ✓ ERR-02: All json_err calls use E_* constants (zero hardcoded strings)
  - ✓ ERR-03: Error code standards documented for contributors
  - ✓ ERR-04: Error path test coverage for lock and flag operations

- ✓ **Architecture Gaps** — v1.2
  - ✓ ARCH-01: queen-init resolves templates via hub path
  - ✓ ARCH-02: State files validated against schema on load
  - ✓ ARCH-03: Spawn-tree entries cleaned up on session end
  - ✓ ARCH-04: Failed Task spawns have retry logic
  - ✓ ARCH-05: queen-* commands documented
  - ✓ ARCH-06: queen-read validates JSON output
  - ✓ ARCH-07: model-get/model-list have exec error handling
  - ✓ ARCH-08: Help command lists all commands including queen-*
  - ✓ ARCH-09: Feature detection doesn't race with error handler
  - ✓ ARCH-10: Temp files cleaned up via exit trap

### Active

## Current Milestone: v1.3 The Great Restructuring

**Goal:** Make Aether more reliable by extracting embedded structures into templates, cleaning up agent definitions, simplifying the distribution pipeline, and defining how agents handle failure. Focused on changes that improve real-world reliability, not theoretical architecture.

**Distribution Simplification:**
- PIPE-01: runtime/ staging directory eliminated — npm package reads directly from .aether/
- PIPE-02: sync-to-runtime.sh replaced with direct packaging approach
- PIPE-03: Pre-commit hook updated for simplified pipeline

**Template Foundation:**
- TMPL-01: colony-state.json.template created with self-documenting annotations
- TMPL-02: constraints.json.template created
- TMPL-03: crowned-anthill.md.template created
- TMPL-04: handoff.md.template created
- TMPL-05: worker-result.json.template created
- TMPL-06: Templates added to distribution pipeline and sync allowlist

**Agent Cleanup:**
- AGENT-01: "Aether Integration" boilerplate removed from all agents
- AGENT-02: Depth-Based Behavior section removed entirely from all agents
- AGENT-03: workers.md reference footer removed from all agents
- AGENT-04: Dead model references deferred — outdated content cleanup is a separate task from boilerplate stripping

**Agent Resilience:**
- RESIL-01: Failure modes defined for all agents (cannot_complete, unexpected_complexity, 3-fix escalation)
- RESIL-02: Success criteria checklist added to all agents
- RESIL-03: Read-only vs read-write explicitly declared per agent

**Template Integration:**
- WIRE-01: init.md reads colony-state template instead of inline JSON
- WIRE-02: init.md reads constraints template instead of inline JSON
- WIRE-03: seal.md reads crowned-anthill template instead of heredoc
- WIRE-04: entomb.md reads handoff template instead of heredoc
- WIRE-05: build.md references worker-result template for agent output

**Queen Coordination:**
- COORD-01: Escalation chain defined (depth 3 → 2 → 1 → Queen → user)
- COORD-02: 6 named workflow patterns added to Queen definition
- COORD-03: Architect agent merged into Keeper
- COORD-04: Guardian agent folded into Auditor as named security lens

### Out of Scope

- Model-per-caste routing — configuration exists but effectiveness unverified
- YAML command generator — 13,573 lines duplicated across .claude/ and .opencode/, generator unused
- Offline/mobile support — CLI-only tool
- ANSI color codes in chat — renders as garbage in Claude Code
- Animated spinners — not supported in Claude Code chat
- Additional ASCII art — diminishing returns, adds noise
- Full XML migration of all 25 agents — do gradually as agents are touched, not as dedicated project
- JSON Schema validation system — templates themselves are the improvement
- File lock protocol for parallel builders — solve when builders actually collide
- Phase scratch pad for shared context — solve when agents demonstrably miss sibling context
- Queen architecture split (hub + project) — solve when cross-repo coordination is needed
- Caste metrics tracking — nice for analytics, not urgent
- Template Registry and versioning — premature optimization
- A/B testing framework for agent formats — interesting but expensive to build

## Context

Shipped v1.2 with ~5,435 lines of shell (aether-utils.sh), 34 Claude Code commands, 33 OpenCode commands. 446 tests passing (415 AVA + 31 bash), 0 failures. All documented bugs (BUG-002 through BUG-012) fixed. Distribution chain correct end-to-end. Error codes fully standardized with contributor documentation.

Tech stack: Bash, jq, xmllint/xmlstarlet, Node.js CLI wrapper.

Three milestones shipped:
- v1.0: 46/46 requirements — full repair and stabilization
- v1.1: 14/15 requirements — visual polish and identity
- v1.2: 24/24 requirements — hardening and reliability

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
| Noise before visual polish | No point polishing output if 30+ headers dominate | ✓ Good — Phase 10 cleared path for 11 |
| Unicode-only visual elements | ANSI color codes stripped by Claude Code | ✓ Good — ━━━━ banners + █░ progress bars work everywhere |
| State-routed Next Up blocks | Dynamic adaptation beats hardcoded suggestions | ✓ Good — colony state drives guidance |
| Canonical caste-system.md | 3 separate definitions caused emoji drift | ✓ Good — single source, references everywhere |
| tmux-gated swarm display | Swarm updates fire uselessly in chat context | ✓ Good — chat users see summary only |
| .update-pending sentinel | Partial failures left inconsistent state | ✓ Good — atomic detection and recovery |
| Session IDs kept in session commands | Debugging value outweighs cosmetic concern | ⚠️ Revisit — NOISE-04 partial |
| Uniform trap pattern for locks | Two parallel tracking systems caused disagreement | ✓ Good — single acquire→trap→work→release pattern |
| Friendly error messages with "Try:" | Machine-readable codes need human-readable guidance | ✓ Good — every error includes recovery suggestion |
| Hub-first template resolution | npm-installed users couldn't find templates | ✓ Good — hub path checked before dev runtime/ |
| Composed EXIT trap | Individual traps from sourced files overwrite each other | ✓ Good — _aether_exit_cleanup calls all cleanups |
| Additive-only state migration | Never remove fields, only add missing defaults | ✓ Good — no data loss on schema upgrade |
| Scope v1.3 to reliability, not architecture | LLM architect review: ~40% of research solved theoretical problems | Pending — focused on templates, failure modes, pipeline simplification |
| Defer Queen split, file locks, schemas | Solve real problems first, build infrastructure when needed | Pending — revisit if concrete need arises |
| Templates before XML rewrite | Template system is highest-impact single improvement | Pending — additive migration: create first, wire later |

## Constraints

- **Must work in Claude Code** — primary platform
- **Visual simplicity** — no loads of terminal text
- **Reliability first** — working > feature-rich
- **Self-contained** — minimal external dependencies
- **UI Style** — GSD-style stage banners with ant-themed names
- **No ANSI colors** — Unicode + emoji only

---

*Last updated: 2026-02-19 after v1.3 milestone start*
