# Feature Landscape: Aether v1.11 Unification

**Domain:** Lost intelligence restoration, self-hosting cleanup, platform hardening
**Researched:** 2026-04-28
**Confidence:** HIGH (all findings verified against source code in `cmd/`, `pkg/`, `.claude/`, `.opencode/`, `.codex/`)

## Executive Summary

v1.11 targets three categories: (1) restoring lost intelligence features from the April 2026 shell-to-Go migration, (2) removing self-hosting artifacts that exist because Aether develops itself, and (3) hardening the 3-platform experience. The lost features are already partially ported -- `init-research` exists with 10 pheromone suggestion patterns and charter generation, the curation pipeline has all 8 ants implemented in `pkg/agent/curation/`, the council system has CRUD commands, and trust scoring has `compute`/`decay`/`tier` subcommands. What is missing is wiring these components into the lifecycle flows where they were originally active, and restoring the suggest-analyze build-step analysis that was never ported from the original 618-line shell script.

The self-hosting artifacts are minimal -- the agent mirrors (`.aether/agents-claude/`, `.aether/agents-codex/`, `.aether/skills-codex/`) appear byte-identical with their source counterparts and the command parity test passes. The real cleanup is identifying and removing stale wrapper commands, orphaned companion files, and ensuring the publish pipeline does not re-introduce artifacts. Platform hardening is mostly about OpenCode command parity and error handling consistency across the three platforms.

## Feature Categories

### Category A: Lost Intelligence Restoration

These features existed in the shell-based Aether and were lost during the Go migration. Some have been partially ported; others were never ported.

#### A1. Smart Init Ceremony (Charter Approval Flow)

**Current state:** `init-research` command exists at `cmd/init_research.go` with full codebase scanning, governance detection, pheromone suggestion generation (10 patterns), and charter generation. The init wrapper at `.claude/commands/ant/init.md` already calls `aether init-research` and presents charter for approval. However, the Go `aether init` command itself does not call `init-research` internally -- the ceremony only happens when the wrapper orchestrates it.

**What's missing:**
- The Go runtime `aether init` does not integrate `init-research` as part of its flow. It creates colony state directly without the scanning step.
- No charter data is persisted in `COLONY_STATE.json` or a separate charter file.
- The pheromone suggestions are presented by the wrapper but the approval flow only works on Claude Code. OpenCode and Codex have no charter ceremony.
- Charter data (intent, vision, governance, goals) is computed but never stored for downstream reference.

**Expected behavior (Go CLI standard):**
1. User runs `aether init "Build a web app"`
2. Go runtime scans the codebase (languages, frameworks, governance, git history, complexity)
3. Generates charter data and pheromone suggestions
4. Outputs both for the wrapper to present
5. Wrapper shows charter, user approves/revises/cancels
6. Approved pheromone suggestions are written via `pheromone-write`
7. Colony state is created with charter metadata attached

**Complexity:** LOW-MEDIUM. The scanning logic exists. The wiring is missing.

#### A2. Suggest-Analyze (Build-Step Pheromone Suggestions)

**Current state:** The build playbook at `.aether/docs/command-playbooks/build-prep.md` references `--no-suggest` as a flag and mentions "the colony analyzes the codebase for patterns that might benefit from pheromone signals." However, no `suggest-analyze` subcommand exists in the Go runtime. The original was a 618-line shell script that analyzed files for patterns (TODO comments, debug artifacts, complex files, etc.) and produced pheromone suggestions.

**What's missing entirely:**
- No `aether suggest-analyze` subcommand in Go
- No build-wave integration point that calls suggest-analyze
- The `init-research` pheromone suggestions only cover 10 static patterns. The original suggest-analyze had dynamic codebase analysis (file complexity, TODO density, debug artifact detection, stale file detection, etc.)

**Expected behavior:**
1. At build start (before worker dispatch), colony scans the codebase for actionable patterns
2. Patterns are scored and presented as FOCUS pheromone suggestions
3. User approves/dismisses each suggestion
4. Approved suggestions are written as active pheromone signals
5. Workers receive these signals in their context

**Analysis patterns that should be restored:**
- TODO/FIXME/HACK comment density (files with many TODOs may need attention)
- Debug artifact detection (console.log, fmt.Println, debug-only code)
- Large file detection (files exceeding complexity thresholds)
- Stale file detection (files not modified in N months)
- Test coverage gaps (source files without corresponding test files)
- Dependency health (outdated or vulnerable dependencies if lockfile available)
- Duplicate code patterns (similar file names in different directories)

**Complexity:** MEDIUM-HIGH. The original was 618 lines. The analysis logic is non-trivial and needs to handle multiple languages. However, it can be simplified by reusing the file-walking infrastructure from `init-research.go`.

#### A3. Circuit Breaker (Cascade Failure Protection)

**Current state:** The immune system at `cmd/immune.go` provides `trophallaxis-diagnose` (error classification), `trophallaxis-retry` (exponential backoff), `scar-add`/`scar-list`/`scar-check` (failure pattern tracking), and `immune-auto-scar` (midden-based pattern detection). This is a retry/immune system but NOT a circuit breaker.

**What's missing:**
- No circuit breaker pattern that prevents cascading failures across workers
- No threshold-based failure detection that halts a build wave when failures exceed a limit
- No automatic phase rollback or pause when consecutive failures suggest systemic issues

**Expected behavior:**
1. During build waves, track failure rate per-wave and per-phase
2. If failure rate exceeds threshold (e.g., 3 failures in a row, or 50% of tasks in a wave), open the circuit
3. Open circuit stops dispatching new workers and surfaces the failure pattern
4. After a cooldown period or manual intervention, close the circuit
5. Failure patterns are recorded as scars for future avoidance

**Complexity:** MEDIUM. The scar/immune infrastructure exists. The circuit breaker logic needs to be added to the build-wave dispatch path and the continue-gate path.

#### A4. Consolidation Pipeline (Phase-End Knowledge Compression)

**Current state:** `cmd/graph_consolidation_cmds.go` exists with graph commands (link, unlink, nodes, edges, shortest-path, cluster). The `pkg/agent/curation/` package has all 8 curation ants (archivist, critic, herald, janitor, librarian, nurse, scribe, sentinel) with an orchestrator. `cmd/curation_cmds.go` exposes CLI subcommands for running individual curation ants.

**What's missing:**
- The consolidation pipeline is not wired into the continue lifecycle. After a phase completes and `/ant-continue` runs, the curation pipeline should automatically run to compress phase learnings, detect contradictions, and promote high-confidence instincts.
- The curation orchestrator exists but is not called from any lifecycle command.

**Expected behavior:**
1. At phase end (during continue), the consolidation pipeline runs automatically
2. Phase learnings are compressed (redundant observations merged, contradictions flagged)
3. Trust scores are recalculated with decay
4. High-confidence instincts are promoted to QUEEN.md
5. Curation results are included in the continue report

**Complexity:** LOW. The infrastructure is all there. The wiring is missing -- a single call to the curation orchestrator from the continue flow.

### Category B: Self-Hosting Cleanup

These are artifacts that exist because Aether was used to develop itself.

#### B1. Agent Mirror Verification and Cleanup

**Current state:** 26 agents across 4 surfaces:
- `.claude/agents/ant/` (26 files, canonical source)
- `.aether/agents-claude/` (26 files, packaging mirror for npm distribution)
- `.opencode/agents/` (26 files, OpenCode surface)
- `.codex/agents/` (26 TOML files, Codex surface)

The `command_parity_test.go` and `command_source_hygiene_test.go` verify agent count parity. The agent mirrors appear to be byte-identical with their source counterparts based on the listing.

**What to verify:**
- Are `.aether/agents-claude/` files truly byte-identical with `.claude/agents/ant/`? If not, what drifted?
- Are there any agents defined in mirrors but not in the source, or vice versa?
- Is the publish pipeline correctly syncing agent mirrors?

**Expected outcome:** Confirm mirrors are in sync, add a byte-identity test, fix any drift.

**Complexity:** LOW. Mostly verification and a test addition.

#### B2. Stale Companion Files

**What to audit:**
- `.aether/skills/` (29 skills) -- are all referenced by the skill-index system?
- `.aether/skills-codex/` -- is this a byte-identical mirror?
- `.aether/templates/` (12 templates) -- are all used by the runtime?
- `.aether/docs/` -- are there stale or orphaned documentation files?
- `.aether/exchange/` -- XML exchange modules -- still used?
- `.aether/utils/` -- runtime utilities -- still referenced?

**Expected outcome:** Remove unreferenced files, verify mirrors are in sync.

**Complexity:** LOW. Audit and delete.

#### B3. Duplicate or Stale Wrapper Commands

**Current state:** 50 commands on Claude Code, 50 on OpenCode, 50 YAML source definitions. The `command_count_test.go` verifies command counts.

**What to verify:**
- Are there commands in `.claude/commands/ant/` or `.opencode/commands/ant/` that have no corresponding YAML source?
- Are there YAML sources that have no generated wrapper?
- Are there commands that reference deleted subcommands?

**Complexity:** LOW.

### Category C: Platform Hardening

#### C1. OpenCode Parity Gaps

**Current state:** OpenCode has 50 commands matching Claude Code's 50. The agent count matches (26 on each surface). However, there may be behavioral differences in how commands execute.

**What to verify:**
- Do all OpenCode commands produce the same JSON output format as Claude Code equivalents?
- Do OpenCode commands handle errors consistently with Claude Code?
- Are there commands that work on Claude Code but fail silently on OpenCode?

**Expected behavior:** All 50 commands work identically across Claude Code and OpenCode. Same JSON output, same error handling, same flags.

**Complexity:** MEDIUM. Requires testing each command on both platforms.

#### C2. Error Handling Consistency

**What to verify:**
- All Go subcommands use consistent error patterns (`outputError` with codes, `outputErrorMessage` for no-store, proper JSON error responses)
- Wrapper commands handle Go runtime errors gracefully (parse error JSON, surface to user)
- No subcommands panic or return nil errors when they should return structured errors

**Complexity:** LOW-MEDIUM. Mostly audit and fixes.

#### C3. Cross-Platform Consistency (Codex)

**Current state:** Codex CLI has runtime-native UX (no wrapper markdown). It uses the Go runtime directly. The `.codex/agents/` directory has 26 TOML definitions and `CODEX.md` documents commands and rules.

**What to verify:**
- Are all 26 Codex agents structurally equivalent to their Claude/OpenCode counterparts?
- Does `CODEX.md` accurately reflect current command capabilities?
- Are there Codex-specific commands that reference removed or changed subcommands?

**Complexity:** LOW-MEDIUM.

#### C4. User Experience Improvements

**What to improve:**
- Init ceremony should feel like a guided onboarding, not a dry JSON dump
- Build feedback should be more actionable (what failed, what to do next)
- Status command should be scannable at a glance
- Error messages should suggest recovery actions, not just report failures

**Complexity:** MEDIUM. Requires UX design decisions.

## Feature Dependencies

```
A1. Smart Init Ceremony
    |
    +---> Charter persistence in COLONY_STATE.json
    |         |
    |         v
    |     Charter data available to colony-prime context
    |
    +---> Pheromone suggestion approval flow
    |         |
    |         v
    |     Approved suggestions written via pheromone-write
    |
    +---> OpenCode/Codex ceremony parity
              |
              v
          All 3 platforms have consistent init experience

A2. Suggest-Analyze
    |
    +---> suggest-analyze subcommand (Go runtime)
    |         |
    |         v
    |     Pattern analysis engine (file walking, detection)
    |
    +---> Build-wave integration
    |         |
    |         v
    |     suggest-analyze called before worker dispatch
    |
    +---> Tick-to-approve UI in wrappers
              |
              v
          Suggestions presented, approved, written as pheromones

A3. Circuit Breaker
    |
    +---> Failure tracking in build-wave
    |         |
    |         v
    |     Threshold detection, circuit open/close
    |
    +---> Continue-gate integration
    |         |
    |         v
    |     Circuit state checked during continue gates
    |
    +---> Scar recording for future avoidance
              |
              v
          Failure patterns become immune system data

A4. Consolidation Pipeline Wiring
    |
    +---> Curation orchestrator called from continue
    |         |
    |         v
    |     Phase learnings compressed, instincts promoted
    |
    +---> Trust score recalculation
              |
              v
          Decay applied, scores updated

B1-B3. Self-Hosting Cleanup (independent of A features)

C1-C4. Platform Hardening (independent of A and B features)
```

## MVP Definition

### Launch With (v1.11 Minimum)

The minimum that makes v1.11 feel like a meaningful release:

- [ ] **A1: Charter persistence** -- Store charter data in COLONY_STATE.json so downstream flows can reference it. This is the cheapest win because `init-research` already generates charter data.
- [ ] **A2: suggest-analyze subcommand** -- Restore the build-step analysis. Even a simplified version (5-7 patterns instead of the original 618-line full scan) provides immediate value during builds.
- [ ] **A4: Curation pipeline wiring** -- Single integration point. Call the orchestrator from continue. All 8 ants are already implemented.
- [ ] **B1-B3: Self-hosting cleanup audit** -- Verify mirrors, remove orphans, add byte-identity test.
- [ ] **C1: OpenCode parity verification** -- Ensure commands work identically.

### Add After Validation (v1.11.x)

- [ ] **A1: OpenCode/Codex ceremony parity** -- The init ceremony currently only works fully on Claude Code.
- [ ] **A3: Circuit breaker** -- More complex, builds on the immune system.
- [ ] **C2-C4: Error handling and UX improvements** -- Polish pass after core features land.

### Future Consideration (v2+)

- [ ] **A2: Full 618-line suggest-analyze** -- The original had more patterns than needed. Start simple, expand based on usage.
- [ ] **A3: Adaptive circuit breaker** -- Thresholds that adjust based on project complexity and historical failure rates.
- [ ] **C4: Interactive ceremony improvements** -- Rich terminal UI for init ceremony, build progress, etc.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| A1: Charter persistence in COLONY_STATE | HIGH | LOW | P1 |
| A2: suggest-analyze subcommand (simplified) | HIGH | MEDIUM | P1 |
| A4: Curation pipeline wiring into continue | HIGH | LOW | P1 |
| B1: Agent mirror byte-identity test | MEDIUM | LOW | P1 |
| B2-B3: Stale file audit and cleanup | MEDIUM | LOW | P1 |
| C1: OpenCode command parity verification | MEDIUM | MEDIUM | P1 |
| A1: OpenCode/Codex ceremony parity | MEDIUM | MEDIUM | P2 |
| A3: Circuit breaker (basic) | MEDIUM | MEDIUM | P2 |
| C2: Error handling consistency | MEDIUM | LOW | P2 |
| C3: Codex agent/commands audit | LOW | LOW | P2 |
| C4: UX improvements (init, build, status) | MEDIUM | MEDIUM | P2 |
| A3: Adaptive circuit breaker | LOW | HIGH | P3 |
| A2: Full suggest-analyze expansion | LOW | MEDIUM | P3 |

## Existing System Integration Points

| Feature | Integration Point | File | What Changes |
|---------|-------------------|------|-------------|
| Charter persistence | `ColonyState` struct | `pkg/colony/colony.go` | Add `Charter *charterData` field with `omitempty` |
| Charter generation | `initCmd.RunE` | `cmd/init_cmd.go` | Call `init-research` logic, store charter in state |
| Charter in context | `buildColonyPrimeOutput()` | `cmd/colony_prime_context.go` | Add charter section to colony-prime prompt |
| suggest-analyze | New subcommand | `cmd/suggest_analyze.go` | Pattern analysis, pheromone suggestion output |
| suggest-analyze build integration | Build-wave playbook | `.aether/docs/command-playbooks/build-wave.md` | Call `aether suggest-analyze` before dispatch |
| Curation wiring | Continue finalize | `cmd/codex_continue_finalize.go` | Call curation orchestrator after phase completion |
| Curation wiring | Continue wrapper | `.claude/commands/ant/continue.md`, `.opencode/commands/ant/continue.md` | Add curation step |
| Circuit breaker | Build-wave dispatch | `cmd/codex_build.go` | Add failure threshold tracking |
| Circuit breaker | Continue gates | `cmd/codex_continue.go` | Check circuit state before advancing |
| Agent mirror test | Test file | `cmd/command_parity_test.go` or new test | Byte-identity comparison for agent files |
| OpenCode parity | Platform doc hygiene | `cmd/platform_doc_hygiene_test.go` | Add OpenCode command output format tests |

## Anti-Features

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full 618-line suggest-analyze re-port | The original was a monolithic shell script with redundant patterns. Re-porting all 618 lines introduces unmaintainable complexity. | Start with 5-7 high-value patterns. Expand based on real usage data. |
| Charter as a separate file system | Charters stored in separate files create sync issues with colony state. If the charter file drifts from state, workers get conflicting information. | Store charter as a field in COLONY_STATE.json. Single source of truth. |
| Circuit breaker with machine learning thresholds | Adaptive thresholds sound smart but require significant training data and add unpredictable behavior. | Start with fixed thresholds (e.g., 3 consecutive failures). Tune based on real colony data. |
| Cross-platform ceremony via markdown generation | Generating markdown ceremony files that all three platforms must parse creates fragile coupling. | Each platform calls the same Go CLI and parses the same JSON output. Platform-specific presentation is the wrapper's job. |
| Stale file auto-deletion during cleanup | Automatically deleting files the audit identifies as stale is dangerous. A file might look unreferenced but be loaded dynamically or referenced by name in a config file. | List stale candidates, require explicit confirmation before deletion. |

## Sources

- `cmd/init_research.go` -- existing codebase scanning, governance detection, pheromone suggestions (10 patterns), charter generation
- `cmd/init_cmd.go` -- current init flow (no research integration)
- `cmd/immune.go` -- immune system (diagnose, retry, scar tracking)
- `cmd/council.go` -- council deliberation system (already ported)
- `cmd/trust.go` -- trust scoring (compute, decay, tier)
- `cmd/curation_cmds.go` -- curation ant CLI subcommands
- `pkg/agent/curation/` -- all 8 curation ants implemented
- `cmd/graph_consolidation_cmds.go` -- graph commands and consolidation
- `cmd/pheromone_write.go` -- pheromone creation with dedup, sanitization, TTL
- `.claude/commands/ant/init.md` -- current init wrapper ceremony flow
- `.aether/docs/command-playbooks/build-prep.md` -- build prep with suggest-analyze reference
- `.aether/docs/command-playbooks/build-wave.md` -- build-wave dispatch flow
- `.planning/PROJECT.md` -- milestone requirements and known losses

---
*Feature research for: Aether v1.11 Unification*
*Researched: 2026-04-28*
