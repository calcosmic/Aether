# Aether

## What This Is

A self-managing development assistant using ant colony metaphor that prevents context rot. Users install it, it guides them through work with clear commands, tells them when to clear context, and maintains state across sessions. The colony learns from each phase and improves over time. As of v3.0, the wisdom system is complete — pheromones evolve automatically, learnings accumulate, and user approval gates promotion to QUEEN.md.

**Current State:** v5.0 Agent Integration shipped — 8 specialist agents integrated into colony commands

## Shipped: v5.0 Agent Integration

**Completed:** 2026-02-22
**Goal:** Integrate 8 specialist agents into existing commands for better security, quality, and efficiency

**Delivered:**
- ✅ P0: Gatekeeper + Auditor in `/ant:continue` — security gates, quality review
- ✅ P1: Probe + Measurer in `/ant:continue` and `/ant:build` — coverage, performance
- ✅ P2: Chronicler + Ambassador in `/ant:seal` and `/ant:build` — documentation, API integration
- ✅ P3: Sage + Weaver in `/ant:seal` and `/ant:continue` — analytics, proactive refactoring

**24/24 requirements verified.**

## Core Value

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

Researching v4.0 Colony Context Enhancement — investigating existing architecture and patterns

### Validated (continued)

- ✓ **Worker Emergence** — v2.0
  - ✓ 22 Claude Code subagents shipped (Builder, Watcher, Queen, Scout, Route-Setter, 4 Surveyors, Keeper, Tracker, Probe, Weaver, Auditor, Chaos, Archaeologist, Gatekeeper, Includer, Measurer, Sage, Ambassador, Chronicler)
  - ✓ Agent distribution pipeline (npm install → hub sync → aether update)
  - ✓ 6 AVA tests for agent quality (frontmatter, tools, naming, content)
  - ✓ Bash line wrapping bug fixed (58 instances across 7 files)
  - ✓ Docs curated (.aether/docs/ from 14 to 8 files)
  - ✓ repo-structure.md added
  - ✓ README updated for v5.0

- ✓ **Pheromone Evolution** — v3.0
  - ✓ PHER-EVOL-01: Pheromones automatically injected at key workflow points
  - ✓ PHER-EVOL-02: End-of-phase pheromone proposals displayed for user approval
  - ✓ PHER-EVOL-03: Tick-to-approve UX for proposed pheromones
  - ✓ PHER-EVOL-04: Pheromone history tracking in colony state

- ✓ **QUEEN.md System** — v3.0
  - ✓ QUEEN-01: QUEEN.md file structure with 5 wisdom categories
  - ✓ QUEEN-02: queen-init command creates QUEEN.md from template
  - ✓ QUEEN-03: queen-read command returns wisdom as JSON for worker priming
  - ✓ QUEEN-04: queen-promote command with type validation
  - ✓ QUEEN-05: Metadata block with version, stats, thresholds

- ✓ **Command Integration** — v3.0
  - ✓ INT-01: init.md calls queen-init after bootstrap
  - ✓ INT-02: build.md calls queen-read before spawning workers
  - ✓ INT-03: continue.md promotes validated learnings to QUEEN.md
  - ✓ INT-04: seal.md promotes final colony wisdom
  - ✓ INT-05: entomb.md promotes wisdom before archiving

- ✓ **Metadata & Tracking** — v3.0
  - ✓ META-01: Promotion thresholds enforced per wisdom type
  - ✓ META-02: Evolution log tracks wisdom changes over time
  - ✓ META-03: Stats block tracks counts per category
  - ✓ META-04: colonies_contributed tracks wisdom origins

- ✓ **Observation Tracking** — v3.0
  - ✓ OBS-01: learning-observe function records observations to learning-observations.json
  - ✓ OBS-02: learning-check-promotion function returns proposals meeting thresholds
  - ✓ OBS-03: Observation counts accumulate across colonies
  - ✓ OBS-04: Content hashing prevents duplicate observation entries

- ✓ **Unified Priming** — v3.0
  - ✓ PRIME-01: colony-prime function combines wisdom + signals + instincts
  - ✓ PRIME-02: build.md uses colony-prime for unified worker context
  - ✓ PRIME-03: Workers receive structured colony context

### Shipped

**v1.3 The Great Restructuring** (phases 20-25):
- PIPE-01 through PIPE-03: Distribution simplified — runtime/ eliminated
- TMPL-01 through TMPL-06: Template foundation — 5 templates extracted and wired
- AGENT-01 through AGENT-04: Agent boilerplate cleaned
- RESIL-01 through RESIL-03: Failure modes and success criteria on all agents
- WIRE-01 through WIRE-05: Commands wired to templates
- COORD-01 through COORD-04: Queen escalation chain, workflow patterns, agent merges

**v1.4 Deep Cleanup (partial)** (phase 26):
- CLEAN-01 through CLEAN-10: File audit — dead files removed, repo cleaned

### Out of Scope

- Model-per-caste routing — configuration exists but effectiveness unverified
- YAML command generator — 13,573 lines duplicated across .claude/ and .opencode/, generator unused
- Offline/mobile support — CLI-only tool
- ANSI color codes in chat — renders as garbage in Claude Code
- Animated spinners — not supported in Claude Code chat
- Full XML migration of all 25 agents — do gradually as agents are touched, not as dedicated project
- JSON Schema validation system — templates themselves are the improvement
- File lock protocol for parallel builders — solve when builders actually collide
- Phase scratch pad for shared context — solve when agents demonstrably miss sibling context
- Queen architecture split (hub + project) — solve when cross-repo coordination is needed
- Caste metrics tracking — nice for analytics, not urgent
- Template Registry and versioning — premature optimization

## Context

Shipped through v5.0 with ~12,352 lines of shell (aether-utils.sh), 36 Claude Code commands, 22 Claude Code subagents. 490+ tests passing. All documented bugs fixed. Distribution chain correct end-to-end. Error codes fully standardized. Templates extracted and wired. Agent definitions cleaned and hardened. Wisdom system complete with observation tracking and user approval gates. 8 specialist agents integrated into colony workflows.

Tech stack: Bash, jq, xmllint/xmlstarlet, Node.js CLI wrapper.

Eight milestones shipped:
- v1.0: 46/46 requirements — full repair and stabilization
- v1.1: 14/15 requirements — visual polish and identity
- v1.2: 24/24 requirements — hardening and reliability
- v1.3: 24/24 requirements — templates, agent cleanup, pipeline, Queen coordination
- v1.4: 10/10 requirements (partial) — file audit and dead file removal
- v2.0: 49 requirements — 22 Claude Code subagents shipped
- v3.0: 25/25 requirements — wisdom and pheromone evolution
- v4.0: 6/6 requirements — memory pipeline (learnings, changelog, visibility)
- v5.0: 24/24 requirements — 8 specialist agents integrated into commands

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

*Last updated: 2026-02-22 after v5.0 milestone shipped*
