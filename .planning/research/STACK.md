# Stack Research: Living Wisdom Pipeline -- Making It Actually Populate

**Domain:** Multi-agent colony orchestration -- wisdom accumulation, hive brain, and agent definition gaps
**Researched:** 2026-03-27
**Confidence:** HIGH (based on direct codebase analysis of 22 agent files, 9 domain modules, 580+ tests, and build/continue playbook flow)

---

## Executive Summary

The wisdom pipeline exists as complete, tested shell infrastructure but has a **liveness gap**: QUEEN.md sections and hive brain entries remain template-only in real colony work because the pipeline depends on colony work actually running through the full build/continue flow, and the flow itself has several friction points where wisdom extraction is optional, silently skipped, or depends on AI agent behavior rather than deterministic shell.

This research identifies **three categories of work** needed to close the gap:

1. **Agent definition files** -- Oracle and Architect are referenced in docs but have no dedicated `.md` agent files in `.claude/agents/ant/` or `.opencode/agents/`. They are currently handled as inline prompts or absorbed by other agents.
2. **Build flow integration hardening** -- The continue-advance and continue-finalize playbooks have the right steps but depend on colony name extraction from `COLONY_STATE.json`, observation threshold gating, and AI agent willingness to extract learnings. These need deterministic fallbacks.
3. **Hive brain population** -- `hive-promote` only fires during `/ant:seal` (colony completion), meaning cross-colony wisdom never accumulates unless a colony is fully sealed. Mid-colony promotion hooks are missing.

---

## Recommended Stack (Changes Only)

### New Agent Definition Files

| File | Purpose | Model | Tools | Why |
|------|---------|-------|-------|-----|
| `.claude/agents/ant/aether-oracle.md` | Dedicated Oracle agent for deep research (RALF loop) | opus | Read, Write, Edit, Bash, Grep, Glob, WebSearch, WebFetch | Oracle is currently invoked via `/ant:oracle` command wizard only -- no agent definition means Queen cannot spawn an Oracle worker during builds. The Deep Research workflow pattern in `aether-queen.md` references "Oracle-led" but has no agent to delegate to. |
| `.claude/agents/ant/aether-architect.md` | Dedicated Architect agent for system design decisions | opus | Read, Write, Edit, Bash, Grep, Glob | Workers.md line 647 says "Architect responsibilities are now handled by Keeper" but the merge is incomplete: there is no architecture-specific agent that Queen can spawn for design work. The caste emoji `🏛️🐜` is still defined and used in naming. |
| `.opencode/agents/aether-oracle.md` | OpenCode mirror of Oracle agent | inherit | Read, Write, Edit, Bash, Grep, Glob, WebSearch, WebFetch | Agent parity requirement from CLAUDE.md. |
| `.opencode/agents/aether-architect.md` | OpenCode mirror of Architect agent | inherit | Read, Write, Edit, Bash, Grep, Glob | Agent parity requirement. |
| `.aether/agents-claude/aether-oracle.md` | Packaging mirror for Claude agents | opus | (same as .claude) | Byte-identical mirror requirement. |
| `.aether/agents-claude/aether-architect.md` | Packaging mirror for Architect | opus | (same as .claude) | Byte-identical mirror requirement. |

**Total: 6 new agent `.md` files.**

**Oracle agent design notes:**
- Currently the Oracle is a slash command wizard (`/ant:oracle`) that launches a bash RALF loop via `oracle.sh` in a tmux session. It is NOT a spawnable agent.
- The Queen's "Deep Research" pattern says "Oracle-led" but cannot actually spawn an Oracle because no agent definition exists.
- The Oracle agent should be a read-heavy research agent with WebSearch/WebFetch for external research, NOT the tmux-based loop. The tmux loop stays as the `/ant:oracle` command. The agent is for when Queen needs to spawn an Oracle worker during a build.
- Key difference from Scout: Oracle has Write tool (can write findings), deeper research mandate, and `model: opus` (needs reasoning depth). Scout is `model: sonnet`, read-only, quick lookup.

**Architect agent design notes:**
- Workers.md line 647 says "merged into Keeper" but this merge is incomplete:
  - Keeper (`aether-keeper.md`) exists and handles knowledge curation
  - Route-Setter (`aether-route-setter.md`) exists and handles phase planning
  - Neither handles the Architect's original role: making system design decisions during builds
  - The caste emoji `🏛️🐜` is still defined and used in ant naming
- The Architect agent should bridge the gap: when Queen encounters a phase that needs architecture decisions, she should be able to spawn an Architect rather than trying to make those decisions herself or delegating to Keeper (who curates existing knowledge, not designs new architecture).

### Build Flow Integration Hardening

| Component | Current State | Required Change | Why |
|-----------|--------------|-----------------|-----|
| `continue-advance.md` Step 2 (learnings extraction) | Depends on AI agent extracting learnings from phase work and writing them to `memory.phase_learnings` | Add deterministic fallback: if no learnings extracted by AI, run a bash-based learning extraction from git diff + test results | AI agents frequently skip learning extraction because it is not a hard gate. A fallback ensures wisdom always accumulates. |
| `continue-advance.md` Step 2.5 (memory-capture loop) | Correctly calls `memory-capture` for each learning | No change needed -- this step is well-designed | Already fires `learning-observe` + `pheromone-write` + `learning-promote-auto` for each learning. |
| `continue-advance.md` Step 2.1.6 (batch auto-promotion) | Sweeps all observations and auto-promotes those meeting thresholds | No change needed -- this step is well-designed | Correctly iterates `learning-observations.json` and calls `learning-promote-auto`. |
| `continue-advance.md` Step 2.1.7 (queen-write-learnings) | Correctly calls `queen-write-learnings` for current phase | No change needed | Already bypasses observation thresholds -- every build writes learnings. |
| `continue-advance.md` Step 3c (instinct promotion to QUEEN.md) | Sweeps instincts with confidence >= 0.8 and promotes via `queen-promote-instinct` | No change needed | Correctly runs every `/ant:continue`. |
| `build-complete.md` Step 5.9 (success capture) | Captures up to 2 patterns from `learning.patterns_observed` | **FRAGILE**: depends on builder including `learning.patterns_observed` in synthesis JSON. Most builders don't include this field. | Need to add a fallback pattern extraction step that runs regardless of builder output. |
| Colony name extraction | `jq -r '.session_id | split("_")[1]' COLONY_STATE.json` used in multiple places | Create a dedicated `colony-name` subcommand that handles edge cases (missing session_id, malformed format) | Colony name is used by `memory-capture`, `queen-promote`, `hive-promote`, and `learning-promote-auto`. A single source of truth prevents silent failures. |

### Hive Brain Population

| Component | Current State | Required Change | Why |
|-----------|--------------|-----------------|-----|
| `hive-promote` call site | Only in `/ant:seal` Step 3.7 | Add `hive-promote` call to `/ant:continue` Step 2.1.7 (after queen-write-learnings) | Hive only gets populated when a colony is sealed. Most colonies never get sealed. Mid-colony promotion ensures cross-colony wisdom accumulates. |
| `queen-seed-from-hive` call site | Only in `/ant:init` Step 322 | Add call to `/ant:build` Step 4 (colony-prime loading) as a pre-build seed step | Hive wisdom is only seeded at colony init. If a colony runs multiple phases, new hive wisdom from other colonies never arrives. |
| Domain tag detection | `_domain_detect()` exists but is only called during seal | Call `_domain_detect()` during `/ant:continue` when promoting to hive | Domain tags are needed for `hive-promote --domain`. Without them, wisdom has no domain scoping and retrieval is less useful. |

### What NOT to Touch

| Component | Why Not |
|-----------|---------|
| `queen.sh` (1242 lines) | Complete, well-tested. The functions `_queen_init`, `_queen_read`, `_queen_promote`, `_queen_write_learnings`, `_queen_promote_instinct`, `_queen_seed_from_hive`, `_queen_migrate` all work correctly. The problem is not in these functions -- it is in the calling code. |
| `hive.sh` (562 lines) | Complete, well-tested. `_hive_init`, `_hive_store`, `_hive_read`, `_hive_abstract`, `_hive_promote` all work correctly. The problem is call sites. |
| `learning.sh` (1553 lines) | Complete, well-tested. The full pipeline (`_learning_observe` -> `_learning_promote_auto` -> `queen-promote` -> `instinct-create`) works end-to-end. The problem is that it is not triggered reliably. |
| `pheromone.sh` (`_colony_prime`) | Context assembly works. The problem is upstream -- if QUEEN.md has no content, colony-prime correctly injects empty sections. |
| Agent definitions (existing 22) | No changes needed to existing agent `.md` files. The new Oracle and Architect agents are additive. |
| `aether-utils.sh` dispatcher | Only needs new subcommand for `colony-name` (trivial 10-line addition). No changes to existing dispatch. |
| Test suite (580+ tests) | No existing tests need modification. New tests for new functionality only. |

---

## Installation

```bash
# No new npm packages needed -- all changes are bash + markdown files

# New agent files to create (6 files):
# .claude/agents/ant/aether-oracle.md
# .claude/agents/ant/aether-architect.md
# .opencode/agents/aether-oracle.md
# .opencode/agents/aether-architect.md
# .aether/agents-claude/aether-oracle.md
# .aether/agents-claude/aether-architect.md

# Existing files to modify (4 files):
# .aether/aether-utils.sh (add colony-name subcommand)
# .aether/docs/command-playbooks/continue-advance.md (add fallback learning extraction + hive-promote)
# .aether/docs/command-playbooks/build-complete.md (add deterministic pattern extraction)
# .claude/commands/ant/build.md (if hive-seed step needed)
```

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| New Oracle agent `.md` | Add Oracle logic to Scout agent | Scout is `model: sonnet` (fast, cheap) and read-only. Oracle needs `model: opus` (deep reasoning) and Write tool for synthesis documents. Merging would dilute both roles. |
| New Architect agent `.md` | Keep merged in Keeper | Keeper's role is "knowledge curation and pattern archiving" -- it preserves existing wisdom. Architect's role is "making new design decisions" -- it creates new wisdom. These are fundamentally different operations. |
| Add `hive-promote` to `/ant:continue` | Add `hive-promote` to `/ant:build` | Build is about implementation, not wisdom accumulation. Continue is the right place -- it already extracts learnings and promotes instincts. Hive promotion is a natural extension of Step 2.1.7. |
| Deterministic learning extraction from git diff | Keep relying on AI agent extraction | AI agents work well when the phase produces clear learnings, but silently skip extraction when they are focused on implementation. A deterministic fallback ensures wisdom ALWAYS accumulates, even if lower quality. |
| `colony-name` subcommand | Keep inline `jq` extraction | The colony name extraction pattern `jq -r '.session_id | split("_")[1]'` is repeated in at least 6 locations across playbooks and utility code. A single subcommand is DRY and handles edge cases (missing session_id returns "unknown" instead of producing errors). |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Modifying `queen-promote` threshold logic | Thresholds are calibrated and tested. Changing them risks either noise (too low) or silence (too high). | Add more call sites instead of lowering thresholds |
| Adding new wisdom types | The 6 existing types (philosophy, pattern, redirect, stack, decree, failure) cover all colony learning categories | Map new learning sources to existing types |
| Making hive-promote synchronous/blocking | Hive promotion should never block phase advancement. The existing non-blocking pattern from seal.md is correct. | Fire-and-forget with error logging, same as seal.md |
| Creating a new "wisdom daemon" or background process | Adds complexity, failure modes, and maintenance burden. The existing on-demand pattern (promote when work completes) is sufficient. | Hook into existing build/continue flow |

---

## Stack Patterns by Variant

**If colony runs single-phase builds (common for small tasks):**
- The `/ant:continue` step will fire once and extract learnings from that single phase
- Hive promotion will fire once with whatever observations accumulated
- This is sufficient -- single-phase colonies don't produce enough wisdom to justify more complexity

**If colony runs multi-phase builds (5+ phases):**
- Each `/ant:continue` will extract learnings from the completed phase
- Observations accumulate across phases (same content hash increments count)
- After 2+ phases with similar learnings, `learning-promote-auto` fires and promotes to QUEEN.md
- Hive promotion fires each continue, gradually building cross-colony wisdom
- This is the primary use case for the deterministic fallback -- multi-phase colonies produce the most learning opportunities

**If colony is sealed (`/ant:seal`):**
- Seal already promotes high-confidence instincts to hive (Step 3.7)
- Adding hive-promote to continue means seal promotes a second time (idempotent due to dedup)
- No conflict -- hive-store's dedup handles same-content promotion gracefully

---

## Version Compatibility

| Component | Compatible With | Notes |
|-----------|-----------------|-------|
| New agent `.md` files | Claude Code 1.0.x, OpenCode 0.x | Standard frontmatter format used by all 22 existing agents |
| `colony-name` subcommand | aether-utils.sh dispatcher (v2.0+) | New subcommand in existing dispatcher case statement |
| `continue-advance.md` changes | All existing phases | Changes are additive (new steps + fallbacks), not modifying existing flow |
| `build-complete.md` changes | All existing phases | Pattern extraction fallback runs after synthesis, does not replace it |
| `hive-promote` in continue | `hive.sh` (v1.0+) | `_hive_promote` already handles all edge cases (dedup, confidence boosting, LRU eviction) |

---

## Root Cause Analysis: Why Wisdom Stays Empty

After analyzing the full pipeline, the root cause is not a single bug but a **chain of soft dependencies**:

```
Build completes
  -> Builder MAY include learning.patterns_observed in synthesis JSON (Step 5.9)
    -> If missing: no success patterns captured, memory-capture never fires for successes
  -> Continue fires, AI extracts learnings to memory.phase_learnings (Step 2)
    -> If AI skips: no learnings extracted, pipeline gets nothing
  -> memory-capture fires for each learning (Step 2.5)
    -> learning-observe records observation (always works if called)
    -> pheromone-write emits FEEDBACK (always works if called)
    -> learning-promote-auto checks threshold (works but needs threshold to be met)
  -> queen-write-learnings fires (Step 2.1.7)
    -> Writes to QUEEN.md Build Learnings (always works if learnings exist)
  -> queen-promote-instinct fires for confidence >= 0.8 (Step 3c)
    -> Promotes to QUEEN.md Instincts (always works if instincts exist with high confidence)
  -> hive-promote fires ONLY during seal (Step 3.7 of seal.md)
    -> NEVER fires during normal build/continue flow
```

**The break points:**
1. Builder synthesis JSON may not include `learning.patterns_observed` -- no fallback
2. AI agent may not extract learnings during continue -- no fallback
3. Hive promotion is seal-only -- no mid-colony promotion
4. Colony name extraction can silently fail if `session_id` is missing -- no fallback

**The fix:** Add deterministic fallbacks at each break point so wisdom accumulates even when the AI-dependent path skips.

---

## Sources

- HIGH: Direct codebase analysis of `.aether/utils/queen.sh` (1242 lines), `.aether/utils/hive.sh` (562 lines), `.aether/utils/learning.sh` (1553 lines)
- HIGH: Playbook analysis of `build-complete.md`, `continue-advance.md`, `continue-finalize.md`
- HIGH: Agent definition analysis of all 22 existing agents in `.claude/agents/ant/`
- HIGH: workers.md line 647 (Architect merge note), oracle.md (Oracle command wizard)
- HIGH: QUEEN.md template (`.aether/templates/QUEEN.md.template`)
- HIGH: Local QUEEN.md content showing actual population state (1 codebase pattern, 1 build learning, 1 instinct from ~20 phases of work)
- HIGH: `aether-utils.sh` dispatcher (subcommand registration)
- HIGH: `colony-prime` in `pheromone.sh` (context assembly logic)

---
*Stack research for: Living Wisdom Pipeline -- Making It Actually Populate*
*Researched: 2026-03-27*
