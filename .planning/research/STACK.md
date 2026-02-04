# Stack Research: Colony Hardening & Real-World Readiness

**Domain:** Multi-agent colony system hardening (v4.4)
**Researched:** 2026-02-04
**Confidence:** HIGH

---

## Executive Summary

This research investigates six technical domains needed for Aether v4.4: CLI animation, recursive spawning, two-tier learning, pheromone decay math, automated code review patterns, and file conflict prevention. All solutions stay within the bash+jq constraint. The most critical finding is the **root cause of the pheromone decay bug** (field note 17): during the filmstrip test, `.aether/aether-utils.sh` did not exist in the target repo, so Claude fell back to LLM-computed math and got it wrong. The fix is both defensive math AND ensuring utilities are available in target repos.

**Key findings:**
1. **CLI animation** is a non-starter in Claude Code's execution model -- Task tool output is buffered, not streamed. Focus on rich static output (progress bars, structured reports) rather than live spinners.
2. **Recursive spawning** is blocked by Claude Code platform -- sub-agents cannot spawn further sub-agents via Task tool. The existing depth-tracking pattern is the correct workaround.
3. **Decay math** is correct in aether-utils.sh but needs defensive guards (clamp negative elapsed, cap at initial strength). The real fix is ensuring utils are deployable to target repos.
4. **Two-tier learning** maps cleanly to Claude Code's existing `~/.claude/` (global) and `.claude/` (project) directory structure.
5. **File conflict prevention** requires task-level coordination (same-file tasks to same worker), not file-level locking.

---

## Recommended Stack

### Core Technologies

All v4.4 additions use existing stack. No new dependencies.

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| **bash** | 4.0+ (macOS ships 3.2; Homebrew provides 5.x) | All utility functions, decay math, activity logging | Already the utility layer language. No change needed. |
| **jq** | 1.6+ | JSON manipulation, `exp` for decay math, `fromdate` for timestamps | Already used. IEEE754 double precision is sufficient for decay calculations (verified). |
| **ANSI escape codes** | Standard (ECMA-48) | Color-coded output per caste, progress bars | No dependency needed -- `printf "\e[32m"` works in all modern terminals. More reliable than `tput` for Claude Code's execution context. |
| **Claude Code Task tool** | Current | Agent spawning (single-level only) | Platform constraint: sub-agents CANNOT spawn sub-agents. Recursive patterns must use workarounds. |
| **JSON files** | RFC 8259 | Two-tier learning storage (`~/.aether/learnings.json` for global, `.aether/data/memory.json` for project) | Stays consistent with existing state management. No new storage technology. |
| **mkdir -p / noclobber** | POSIX | Atomic lock acquisition, file conflict prevention | Already used in `file-lock.sh`. The `(set -o noclobber; echo $$ > "$lock_file")` pattern is correct for local FS. |

### Supporting Patterns (New for v4.4)

| Pattern | Purpose | Implementation |
|---------|---------|----------------|
| **Static progress bars** | Show phase/worker completion visually | `printf` with ANSI colors + Unicode block chars. `filled = round(progress * 20)` of `\u2588` chars. Already partially implemented in build.md Step 5c. |
| **Caste color coding** | Visual distinction per worker type | Fixed ANSI color per caste: colonizer=cyan(36), route-setter=yellow(33), builder=green(32), watcher=magenta(35), scout=blue(34), architect=white(37). |
| **Defensive decay math** | Prevent strength growth bug | Three guards: clamp elapsed >= 0, cap result <= initial strength, floor at 0.001 (skip computation if elapsed > 10 * half_life). |
| **Same-file task grouping** | Prevent parallel write conflicts | Phase Lead groups tasks touching the same file to a single worker. Prompt-level enforcement, not file-level locking. |
| **Global learning store** | Cross-project learnings | `~/.aether/global_learnings.json` with promotion from project memory. Architect ant synthesizes, user approves. |
| **Spawn depth tracking** | Enforce recursion limits in prompt-only regime | Depth counter passed in every spawn prompt (`You are at depth N`). spawn-check utility enforces max_depth=3. Already implemented. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| **aether-utils.sh** (extended) | All new deterministic operations | Add subcommands: `learning-promote`, `learning-global-read`, `activity-log-append` (fix overwrite bug), `error-add-phased` (add phase field). Stays under 400 lines. |
| **Git worktrees** | NOT recommended for this system | Overkill for Aether's hub-and-spoke model. Workers write sequentially, not in parallel filesystem isolation. |

---

## Critical Fix: Pheromone Decay Math

### Root Cause Analysis (HIGH confidence)

The field note reports FOCUS signal growing from 0.7 to 8.005. Working backwards:

```
0.7 * e^x = 8.005
e^x = 11.436
x = ln(11.436) = 2.436

Since x = -0.693 * elapsed / half_life:
-0.693 * elapsed / 3600 = 2.436
elapsed = -12,653 seconds (NEGATIVE)
```

A negative elapsed time means `(now - created_at)` was negative -- the system believed the pheromone was created in the future.

**Root cause is NOT the formula.** The formula in `aether-utils.sh` is mathematically correct (verified by direct jq testing). The root cause is **deployment**: during the filmstrip test, `.aether/aether-utils.sh` did not exist in the target repo (field note 6 confirms this). When the utility call fails, the prompt instructs Claude to "fall back to manual calculation." Claude (an LLM) attempted to compute `e^(-0.693 * t / h)` and got the sign wrong, producing exponential GROWTH instead of decay.

### The Fix (Two Parts)

**Part 1: Defensive math in aether-utils.sh** (guards against any future edge cases):

```bash
pheromone-decay)
  [[ $# -ge 3 ]] || json_err "Usage: pheromone-decay <strength> <elapsed_seconds> <half_life>"
  json_ok "$(jq -n --arg s "$1" --arg e "$2" --arg h "$3" '
    ($s|tonumber) as $strength |
    ([$e|tonumber, 0] | max) as $elapsed |     # GUARD 1: clamp elapsed >= 0
    ($h|tonumber) as $half_life |
    if $elapsed > ($half_life * 10) then
      {strength: 0}                             # GUARD 2: skip computation, effectively zero
    else
      ($strength * ((-0.693147180559945 * $elapsed / $half_life) | exp)) as $decayed |
      {strength: ([$decayed, $strength] | min | . * 1000000 | round / 1000000)}  # GUARD 3: cap at initial
    end
  ')"
  ;;
```

Same guards needed in `pheromone-batch` and `pheromone-cleanup`.

**Part 2: Eliminate LLM fallback path.** When the utility is unavailable, commands should NOT attempt manual math. Instead:
- If `pheromone-batch` fails, treat all pheromones as active at their initial strength (fail-open, slightly wrong but never catastrophically wrong)
- Remove all "fall back to manual multiplication" instructions from worker specs
- The LLM should NEVER compute `exp()`, `ln()`, or any transcendental function

### Formula Reference

The correct half-life exponential decay formula:

```
N(t) = N0 * e^(-ln(2) * t / t_half)

Where:
  N(t)    = current strength
  N0      = initial strength
  t       = elapsed seconds (MUST be >= 0)
  t_half  = half-life in seconds
  ln(2)   = 0.693147180559945

Equivalently:
  N(t) = N0 * (1/2)^(t / t_half)

In jq:
  .strength * ((-0.693147180559945 * $elapsed / .half_life_seconds) | exp)
```

jq's `exp` function uses IEEE754 double precision (C math library). Precision is sufficient -- 15-16 significant digits, more than enough for signal strengths rounded to 3-6 decimal places.

**Known jq issues to guard against:**
- `fromdate` does NOT support fractional seconds (strips via regex `sub("\\.[0-9]+Z$";"Z")` -- already handled)
- `fromdate` has a known DST/timezone bug on some macOS versions (tested on current system: NOT affected in CET/February)
- `now` may return exponential notation -- use `now | floor` for integer epoch

---

## CLI Animation & Visual Output

### What Works in Claude Code

Claude Code's Task tool returns output only on completion -- there is no streaming of sub-agent output to the user. This means:

| Pattern | Works? | Why |
|---------|--------|-----|
| Live spinners (background process) | NO | Task tool buffers all output. User sees nothing until agent returns. |
| Streaming progress updates | NO | No stdout streaming from sub-agents to parent. |
| Static progress bars between workers | YES | Queen displays progress after each worker completes (already implemented in build.md Step 5c). |
| Color-coded output per caste | YES | ANSI escape codes in printf output. Rendered when Queen displays results. |
| Rich structured reports | YES | Worker reports rendered by Queen with emoji + ANSI formatting. |
| Activity log polling | YES | Queen reads activity log between worker spawns (implemented in v4.3). |

### Recommended Color Scheme

Use ANSI 256-color mode for caste identification. These colors are visually distinct even in default terminal themes:

```bash
# Caste color definitions (ANSI escape codes)
COLOR_COLONIZER="\e[36m"    # Cyan
COLOR_ROUTESETTER="\e[33m"  # Yellow
COLOR_BUILDER="\e[32m"      # Green
COLOR_WATCHER="\e[35m"      # Magenta
COLOR_SCOUT="\e[34m"        # Blue
COLOR_ARCHITECT="\e[37m"    # White/bright
COLOR_QUEEN="\e[1;33m"      # Bold yellow
COLOR_RESET="\e[0m"         # Reset

# Usage in aether-utils.sh output:
printf "${COLOR_BUILDER}[BUILDER]${COLOR_RESET} Created: %s\n" "$file"
```

**Why ANSI over tput:** Claude Code executes bash in a pseudo-terminal. `tput` queries terminfo for capabilities, adding overhead and a potential failure point if `TERM` is not set correctly. ANSI escape codes are a direct standard (ECMA-48, 1976) supported by every modern terminal. For a utility that runs thousands of times, hardcoded ANSI is simpler and more reliable.

### Progress Bar Pattern

Already partially implemented. The canonical pattern for aether-utils.sh:

```bash
# progress-bar subcommand: progress-bar <completed> <total> <label>
progress-bar)
  completed="${1:-0}"
  total="${2:-1}"
  label="${3:-Progress}"
  width=20
  filled=$(( completed * width / total ))
  empty=$(( width - filled ))
  bar=$(printf '%*s' "$filled" '' | tr ' ' '#')
  space=$(printf '%*s' "$empty" '' | tr ' ' '-')
  printf "\r%s [%s%s] %d/%d" "$label" "$bar" "$space" "$completed" "$total"
  ;;
```

Unicode block characters (`\u2588`) are prettier but may not render in all terminals. Stick with ASCII `#` and `-` for maximum compatibility, with emoji prefix for caste identification.

### What NOT to Build

| Pattern | Why Not |
|---------|---------|
| Background spinner processes | Claude Code sandbox kills background processes. Task tool is synchronous. |
| ncurses/dialog TUI | External dependency, overkill for status output. |
| tmux pane splitting | Requires tmux installation, not available in all Claude Code environments. |
| Animated cursor movement | ANSI cursor codes (`\e[A`, `\e[2K`) are fragile in buffered output contexts. |

---

## Recursive Agent Spawning

### Platform Reality (HIGH confidence)

Claude Code's Task tool enforces **single-level delegation**. Sub-agents spawned via Task cannot themselves use the Task tool -- it is not available in their tool set.

This was confirmed by:
- GitHub Issue #4182 (July 2025): explicit feature request for nested sub-agent spawning
- GitHub Issue #1770 (June 2025): testing revealed agents resort to workarounds (bash subprocess calls) when Task tool is unavailable
- Claude Code documentation: Task tool creates ephemeral workers with isolated 200k context windows, no recursive access

### Aether's Current Workaround (Already Correct)

Aether already handles this via prompt-level depth tracking:

1. Worker specs include spawn-check gate: `bash .aether/aether-utils.sh spawn-check <depth>`
2. Depth passed in every spawn prompt: `You are at depth <N>.`
3. Max depth enforced at 3 levels
4. Max 5 active workers colony-wide

**This is the right pattern.** The v4.4 improvement is not "enable recursive spawning" (platform-blocked) but rather:

### Improvement: Smarter Hub-and-Spoke

Since recursive spawning is impossible, optimize the existing hub-and-spoke:

| Change | How | Why |
|--------|-----|-----|
| **Phase Lead spawns workers directly** | Already implemented in v4.3 -- Queen spawns, not Phase Lead | Avoids the need for Phase Lead to have Task tool access |
| **Worker result chaining** | Pass previous worker output as context to dependent workers | Workers in Wave 2 get Wave 1 results without needing to spawn scouts |
| **Capability gap reporting** | Workers report "I need X" in their output instead of spawning | Queen reads reports and spawns follow-up workers |
| **Auto-reviewer pattern** | Queen auto-spawns watcher after every builder | Already implemented in Step 5.5. Extend to auto-spawn debugger on failure. |

### Auto-Spawned Reviewer/Debugger Pattern (field note 8)

The Queen already spawns a mandatory watcher (Step 5.5). Extend this to:

```
After each worker completes:
  IF worker reported ERROR:
    Auto-spawn debugger (builder-ant with error context)
    Retry up to 2 times (already implemented in Step 5c)

After watcher completes:
  IF watcher quality_score < 6:
    Auto-spawn builder with watcher's issue list as fix instructions
    Re-run watcher verification on fixes

After all phases complete:
  Auto-spawn architect for tech debt synthesis
  Auto-spawn watcher for project-wide quality report
```

This gives the EFFECT of recursive spawning (builder encounters problem, debugger fixes it, watcher verifies) while staying within the flat hub-and-spoke model.

---

## Two-Tier Learning System

### Architecture

Maps to Claude Code's existing directory conventions:

```
~/.aether/                          # Global (cross-project)
  global_learnings.json             # Promoted learnings
  global_errors.json                # Cross-project error patterns
  config.json                       # User preferences

.aether/data/                       # Project-specific (per-repo)
  memory.json                       # Project learnings (already exists)
  errors.json                       # Project errors (already exists)
```

### Learning Promotion Mechanism

```
PROJECT LEARNING (auto, every phase)
  |
  v
PROMOTION CANDIDATE (architect-ant synthesis, batch)
  |
  v
USER APPROVAL (optional gate)
  |
  v
GLOBAL LEARNING (persists across projects)
```

**Promotion criteria** (architect-ant evaluates):
1. **Project-agnostic** -- Does this learning apply beyond this specific codebase? ("Use parameterized SQL" = yes. "The auth module is in src/auth/" = no.)
2. **Repeated** -- Has this learning appeared in 2+ projects? Auto-promote.
3. **High-confidence** -- Was this learning validated by watcher verification?
4. **Actionable** -- Can a worker act on this learning without additional context?

### Implementation in aether-utils.sh

```bash
learning-promote)
  # Move a learning from project to global
  [[ $# -ge 2 ]] || json_err "Usage: learning-promote <learning_id> <reason>"
  learning_id="$1"
  reason="$2"
  global_file="$HOME/.aether/global_learnings.json"
  project_file="$DATA_DIR/memory.json"

  # Ensure global directory exists
  mkdir -p "$HOME/.aether"
  [[ -f "$global_file" ]] || echo '{"learnings":[],"promoted_count":0}' > "$global_file"

  # Extract learning from project memory
  learning=$(jq --arg id "$learning_id" '.phase_learnings[] | select(.id == $id)' "$project_file")
  [[ -n "$learning" ]] || json_err "Learning $learning_id not found"

  # Add to global with promotion metadata
  updated=$(jq --argjson learn "$learning" --arg reason "$reason" --arg ts "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '
    .learnings += [$learn + {promoted_at: $ts, promotion_reason: $reason}] |
    .promoted_count += 1
  ' "$global_file")
  atomic_write "$global_file" "$updated"
  json_ok '"promoted"'
  ;;

learning-global-read)
  # Read global learnings for injection into worker context
  global_file="$HOME/.aether/global_learnings.json"
  [[ -f "$global_file" ]] || json_ok '{"learnings":[]}'
  json_ok "$(cat "$global_file")"
  ;;
```

### Integration Points

| Command | How It Uses Global Learnings |
|---------|------------------------------|
| `/ant:init` | Read global learnings and inject relevant ones into colony context |
| `/ant:build` | Include global learnings in Phase Lead prompt as "cross-project wisdom" |
| `/ant:continue` | After extracting phase learnings, suggest promotion candidates to user |
| `/ant:status` | Show count of global learnings available |

---

## File Conflict Prevention

### The Problem (field notes 10, 13)

Multiple builders editing the same file in parallel causes last-write-wins conflicts. Phase 1: one builder overwrote another's work. Phase 2: same issue recurred.

### Solution: Prevention Over Resolution (HIGH confidence)

The 2025 multi-agent coordination consensus is clear: **prevent conflicts at task assignment time, don't resolve them after the fact.**

### Implementation: Same-File Task Grouping

This is a **prompt-level** solution, not a code-level one. The Phase Lead's task assignment prompt already groups tasks into waves. Add a constraint:

```markdown
--- CONFLICT PREVENTION RULE ---
Tasks that modify the same file MUST be assigned to the same worker.
Before assigning tasks, check which files each task will likely touch.
Group file-overlapping tasks together. This prevents parallel write conflicts.

Example:
  Task 1: Add auth routes to routes/index.ts
  Task 2: Add API routes to routes/index.ts
  -> Both touch routes/index.ts -> assign to SAME builder-ant

  Task 3: Write auth middleware in middleware/auth.ts
  -> Different file -> can go to a DIFFERENT builder-ant
```

### Backup: File-Level Locking (Already Exists)

The existing `file-lock.sh` provides file-level locking via `noclobber`:

```bash
# Atomic lock acquisition (already implemented)
(set -o noclobber; echo $$ > "$lock_file") 2>/dev/null
```

This is correct for local filesystem use. No changes needed. The limitation is that workers spawned via Task tool run in the same process context and share the filesystem -- locking prevents corruption but doesn't prevent logical conflicts (two workers both successfully writing different content to the same file, with the last write winning).

**The lock prevents corruption. Task grouping prevents logical conflicts. Both are needed.**

### What NOT to Do

| Anti-Pattern | Why |
|--------------|-----|
| Git worktrees per worker | Overkill. Workers run sequentially within waves. Merge conflicts would require git resolution logic -- more complexity than the problem warrants. |
| File-level OCC (optimistic concurrency) | Would require read-before-write versioning. Adds per-file version tracking overhead to every write. |
| Distributed lock manager | External dependency. Aether runs on local filesystem only. |

---

## Alternatives Considered

| Category | Recommended | Alternative | Why Not Alternative |
|----------|-------------|-------------|---------------------|
| CLI animation | Static progress bars + color | Live spinners via background bash processes | Task tool buffers output. Background processes killed by sandbox. Spinner would spin in the void. |
| Decay math | jq `exp()` with defensive guards | `bc -l` for arbitrary precision | jq is already loaded for JSON processing. IEEE754 double is more than sufficient. Adding `bc` is a new dependency for zero practical benefit. |
| Color output | ANSI escape codes (`\e[32m`) | `tput setaf 2` | `tput` queries terminfo database, adds syscall overhead, fails if TERM unset. ANSI is direct, standard since 1976. |
| File conflict prevention | Task grouping in Phase Lead prompt | File-level locks per write | Locks prevent corruption but not logical conflicts. Task grouping prevents both. |
| Recursive spawning | Hub-and-spoke with capability gap reporting | `claude -p` subprocess calls (hack) | Loses all context, no observability, unreliable, and actively discouraged by Anthropic. |
| Global learnings | `~/.aether/global_learnings.json` | SQLite database in `~/.aether/` | External dependency (sqlite3 binary). JSON is consistent with existing stack. Global learnings are small (tens to hundreds of entries). |
| Learning promotion | Architect-ant synthesis + user approval gate | Automatic frequency-based promotion | Risk of promoting false positives. User gate prevents bad learnings from propagating. Low overhead (few promotions per project). |
| Activity log persistence | Append mode with phase rotation | Single growing log file | Current bug is overwrite. Fix is append (`>>`) with archival at phase boundaries. Rotation prevents unbounded growth. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| **Node.js/Python for any utility** | Violates bash+jq-only constraint. Adds runtime dependency. | bash+jq handles everything v4.4 needs. |
| **`bc` for decay math** | Unnecessary precision (jq's IEEE754 double is sufficient). New dependency. | jq `exp()` function with defensive guards. |
| **`tput` for colors** | Indirect (queries terminfo), fragile if TERM unset, extra syscall per color change. | Direct ANSI escape codes: `\e[32m` for green, `\e[0m` for reset. |
| **Background bash processes for animation** | Claude Code sandbox may kill background processes. Task tool is synchronous -- user sees nothing during execution. | Rich static output displayed by Queen between worker spawns. |
| **tmux/screen for parallel output** | External dependency. Not available in all Claude Code environments. Not useful -- workers are sequential within waves. | Activity log + Queen-driven display between spawns. |
| **Git worktrees for workspace isolation** | Massive overkill. Workers write sequentially. Merge resolution adds more complexity than it prevents. | Same-file task grouping in Phase Lead prompt. |
| **Vector databases for learning storage** | External dependency. Overkill for tens-to-hundreds of learning entries. | JSON files with jq queries. Full-text search unnecessary -- learnings are short strings. |
| **LLM-computed math (any transcendental function)** | Root cause of the 8.005 decay bug. LLMs cannot reliably compute exp(), ln(), or trigonometric functions. | Always use aether-utils.sh. If utility unavailable, fail-open with raw strength values, NEVER attempt manual computation. |
| **`flock` command for file locking** | Not available on all macOS versions by default. `flock` is a Linux util-linux command, not POSIX. | `noclobber` pattern (`set -o noclobber; echo $$ > lockfile`) -- already implemented, POSIX-compliant. |

---

## Subcommand Budget

Current aether-utils.sh: 16 subcommands, ~265 lines.
Constraint: stay under 400 lines total.

| New Subcommand | Lines (est.) | Purpose |
|----------------|-------------|---------|
| `learning-promote` | ~20 | Move project learning to global store |
| `learning-global-read` | ~5 | Read global learnings for context injection |
| `error-add-phased` | ~5 | Wrapper around error-add that includes phase number |
| `activity-log-append` | ~3 | Fix: use >> instead of > for activity log writes |
| `progress-bar` | ~10 | Formatted progress bar output with ANSI color |
| **Total new** | **~43** | **Estimated total: ~308 lines** |

Stays well under 400-line budget.

---

## Version Compatibility

| Component | Requires | macOS Default | Notes |
|-----------|----------|---------------|-------|
| bash | 4.0+ | 3.2 (but Homebrew provides 5.x) | Associative arrays need 4.0+. Most Aether code works on 3.2. |
| jq | 1.6+ | Not installed (Homebrew) | `exp()` function available since jq 1.5. `fromdate` since 1.5. |
| ANSI escapes | Any VT100-compatible terminal | Terminal.app, iTerm2 both support | Universal support since ~1978. |
| mkdir -p | POSIX | Built-in | Used for directory creation. |
| noclobber | POSIX | Built-in bash option | Used for atomic lock creation. |
| `~/.aether/` directory | Filesystem | Always available | Global learning store location. |

---

## Sources

### Primary (HIGH confidence -- verified by testing)

- **Aether codebase analysis** -- Direct examination of `aether-utils.sh` (265 lines, 16 subcommands), `file-lock.sh` (123 lines), `atomic-write.sh` (214 lines), all 6 worker specs, and all 13 commands.
- **Decay math verification** -- Tested jq `exp()` directly with known values. Formula is correct. Confirmed negative elapsed produces growth (root cause of 8.005 bug).
- **Filmstrip test data** -- Read actual `pheromones.json` from `/Users/callumcowie/Desktop/aether test/.aether/data/` confirming pheromone format and timestamps.
- **Claude Code Task tool limitation** -- [Sub-Agent Task Tool Not Exposed When Launching Nested Agents (Issue #4182)](https://github.com/anthropics/claude-code/issues/4182), [Parent-Child Agent Communication (Issue #1770)](https://github.com/anthropics/claude-code/issues/1770)
- **Claude Code settings hierarchy** -- [Claude Code settings documentation](https://code.claude.com/docs/en/settings) confirms `~/.claude/` for global, `.claude/` for project scope.

### Secondary (MEDIUM confidence -- verified with official sources)

- **ANSI escape codes** -- [ANSI escape code (Wikipedia)](https://en.wikipedia.org/wiki/ANSI_escape_code), [ANSI Escape Codes reference (GitHub Gist)](https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797)
- **File locking in bash** -- [BashFAQ/045](https://mywiki.wooledge.org/BashFAQ/045) (Greg's Wiki, canonical bash reference), [flock(2) man page](https://man7.org/linux/man-pages/man2/flock.2.html)
- **jq math functions** -- [jq 1.7 Manual](https://jqlang.org/manual/v1.7/) (exp, fromdate, now documented), [fromdate fractional seconds issue #1117](https://github.com/jqlang/jq/issues/1117)
- **Bash spinners** -- [How to Write Better Bash Spinners](https://willcarh.art/blog/how-to-write-better-bash-spinners), [Bash Spinner for Long Running Tasks (Baeldung)](https://www.baeldung.com/linux/bash-show-spinner-long-tasks)
- **Multi-agent file conflict patterns** -- [Parallel Agents Are Easy. Shipping Without Chaos Isn't.](https://dev.to/rokoss21/parallel-agents-are-easy-shipping-without-chaos-isnt-1kek), [Multi-Agent Coordination Strategies (Galileo)](https://galileo.ai/blog/multi-agent-coordination-strategies)

### Tertiary (LOW confidence -- single source or training data)

- **Two-tier memory patterns** -- [Practical Memory Patterns for Agent Workflows (AIS)](https://www.ais.com/practical-memory-patterns-for-reliable-longer-horizon-agent-workflows/) (describes promotion rules from personal -> team -> organization), [Memory OS of AI Agent (arXiv)](https://arxiv.org/abs/2506.06326)
- **Claude Code Task tool internals** -- [The Task Tool: Claude Code's Agent Orchestration System (DEV Community)](https://dev.to/bhaidar/the-task-tool-claude-codes-agent-orchestration-system-4bf2), [Claude Code Swarm Orchestration (GitHub Gist)](https://gist.github.com/kieranklaassen/4f2aba89594a4aea4ad64d753984b2ea)

---

## Confidence Assessment

| Area | Confidence | Reasoning |
|------|------------|-----------|
| **Decay math fix** | HIGH | Root cause verified by testing actual data and formula. Guards are standard defensive programming. |
| **CLI animation constraints** | HIGH | Task tool buffering is a platform fact, not an opinion. Verified via Claude Code docs and issues. |
| **Recursive spawning limitation** | HIGH | Platform constraint confirmed by multiple GitHub issues and documentation. |
| **File conflict prevention** | HIGH | Same-file grouping is the consensus pattern across multi-agent systems in 2025. Already proven in Aether's own field test. |
| **Two-tier learning architecture** | MEDIUM | Directory structure maps to Claude Code conventions. Promotion mechanism is design opinion based on general patterns, not proven in Aether specifically. |
| **Color scheme choices** | MEDIUM | ANSI codes are universal, but specific color-to-caste mapping is an aesthetic choice that may need user testing. |
| **Subcommand budget** | HIGH | Line counts estimated from existing similar subcommands. 308 lines is well under 400-line constraint. |

---

## Open Questions (Phase-Specific Research)

1. **Deployment model for target repos** -- How does `.aether/` get into repos where Aether is used? This is the REAL fix for the decay bug. Options: bootstrap via `/ant:init`, global install, symlink. Deferred per field note 16 but affects v4.4 reliability.
2. **Learning promotion UX** -- Should promotion be inline (during `/ant:continue`) or batch (via a new `/ant:synthesize` flow)? Depends on how often promotions happen in practice.
3. **Auto-continue mode** -- How much user agency to preserve? Full auto (run all phases) vs semi-auto (auto-continue but pause on watcher failures)? Design question, not stack question.
4. **Swarm mode compatibility** -- Claude Code is developing native multi-agent coordination (TeammateTool). If/when this ships, Aether's hub-and-spoke may need rearchitecting. Monitor but don't build for it yet.
