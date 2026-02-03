# Phase 14: Visual Identity - Research

**Researched:** 2026-02-03
**Domain:** Claude-native prompt-driven visual output formatting
**Confidence:** HIGH

## Summary

Phase 14 restores the visual identity lost during the v3-rebuild by adding rich formatted output instructions to existing Claude Code slash command prompts (markdown files in `.claude/commands/ant/`). This is NOT runtime code -- there are no Python scripts, no bash functions, no terminal libraries. The "visual identity" is achieved entirely by instructing Claude (via prompt text in command files) to format its output using specific characters, patterns, and layouts when it generates responses.

The research focused on three areas: (1) what visual patterns existed in v2 that were lost, (2) what the current v3 command prompts look like and where formatting instructions need to be added, and (3) what Unicode box-drawing characters and formatting conventions work reliably when Claude generates text output. Key finding: the v2 system used bash scripts with jq to generate formatted output (456-line status.md, 382-line init.md); the v3 system uses simple prompt instructions that Claude follows when executing commands. The visual identity can be restored by enriching the "Display" steps in each command prompt with specific output templates.

**Primary recommendation:** Add box-drawing header templates, step progress indicator patterns, pheromone decay bar computation instructions, and worker grouping formats directly into the prompt text of the 7 affected command files (init.md, build.md, continue.md, status.md, phase.md, pause-colony.md, resume-colony.md).

## Standard Stack

### Core

This phase has no library dependencies. The "stack" is the set of Unicode characters and formatting conventions used in prompt output templates.

| Component | Purpose | Why Standard |
|-----------|---------|--------------|
| Unicode box-drawing | Headers and section separators | Universally rendered by terminals and text displays |
| Step indicators `[checkmark]/[arrow]/[ ]` | Multi-step progress display | Established convention from v2, clear visual meaning |
| Block characters for bars | Pheromone strength visualization | Simple visual density encoding |
| Emoji status indicators | Worker status grouping | Established in v2 (phase 12) |

### Box-Drawing Character Reference

| Character | Name | Use In Phase 14 |
|-----------|------|-----------------|
| `=` | Double horizontal line | Top/bottom of headers |
| `\|` | Vertical line | Side borders of headers |
| `+` | Corner/junction | Header corners |
| `#` | Hash | Alternative section divider |
| `-` | Dash | Subsection dividers |

**Note on Unicode box-drawing:** Characters like `U+2550` (`═`), `U+2551` (`║`), `U+2554` (`╔`), `U+255A` (`╚`) were used in v2's visualization.py. These render correctly in most terminals. However, since Claude generates text output (not a terminal application), the simpler approach is to use these characters directly in the prompt template and let Claude reproduce them. The v2 status.md already used `╔══╗`/`║  ║`/`╚══╝` patterns successfully.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Unicode box-drawing (`═`, `║`) | ASCII box-drawing (`=`, `\|`, `+`) | Unicode looks better but ASCII is safer for all environments. Recommend Unicode -- it worked in v2. |
| Emoji status indicators | Text-only labels | Emoji provides faster scanning but needs text labels for accessibility. Use both (as v2 did). |
| Full-width progress bars | Numeric-only display | Visual bars are more scannable. Use bars + numeric values together. |

**Installation:**
No installation needed. All formatting is in prompt text.

## Architecture Patterns

### Key Constraint: Claude-Native Prompt Architecture

This is the most critical architectural understanding for this phase. The command files (`.claude/commands/ant/*.md`) are **not scripts that execute code**. They are **instruction documents that Claude reads and follows**. When a user runs `/ant:status`, Claude reads `status.md`, follows the instructions, reads JSON files using Read tool, and outputs formatted text.

Therefore, "adding visual identity" means:
1. Adding output template blocks to the prompt markdown
2. Adding instructions like "Display the following header with these exact characters"
3. Adding computational instructions like "For each pheromone signal, compute current strength and display a bar"

There is NO bash, NO jq, NO Python to write. The prompt tells Claude what to output.

### Pattern 1: Box-Drawing Header Template

**What:** A fixed output template that Claude reproduces at the top of command output
**When to use:** Every major command (init, build, status, phase, continue, pause-colony, resume-colony)

The header provides visual separation and professional appearance. Template for command prompts:

```
Output this header at the top of your response:

===================================================
  AETHER COLONY :: {SECTION_NAME}
===================================================
```

For status command, use a richer header:

```
===================================================
  AETHER COLONY STATUS
  Session: {session_id}
  State: {state}
===================================================
```

**Source:** v2's `visualization.py` used `╔══╗`/`║  ║`/`╚══╝` pattern. The v2 `status.md` (456 lines) had:
```
╔══════════════════════════════════════════════════════════════╗
║  Queen Ant Colony Status                                     ║
╠══════════════════════════════════════════════════════════════╣
║  Session: {session_id}                                       ║
║  State: {state}                                              ║
╚══════════════════════════════════════════════════════════════╝
```

**Recommendation:** Use the full Unicode box-drawing version. It worked in v2, it looks professional, and Claude reproduces it reliably.

### Pattern 2: Step Progress Indicators

**What:** Visual step tracking with `[checkmark]`/`[arrow]`/`[ ]` markers
**When to use:** Multi-step commands (init, build, continue)

The prompt should instruct Claude to display ALL steps at the start and update the display mentally as it works through each step. Since Claude outputs text sequentially (not interactively), the practical approach is:

- Display the step list with current progress at key checkpoints
- At the end, show the completed step list

Template for command prompts:

```
After completing each step, display progress:

  [checkmark] Step 1: Validate Input
  [checkmark] Step 2: Read Current State
  [arrow] Step 3: Write Colony State
  [ ] Step 4: Emit INIT Pheromone
  [ ] Step 5: Display Result
```

**Source:** v2's `init.md` (382 lines) had a bash-based step tracker with `show_step_progress()` function. In v3 Claude-native form, this becomes simple output instructions.

**Practical consideration:** Claude generates output as a stream. It cannot go back and update previous text. So step progress should be displayed ONCE at the end (showing all steps completed) or at key intermediate points. The most practical approach:
1. Show "Starting..." before the work begins
2. Show the completed progress list at the end, before the final result display

### Pattern 3: Pheromone Decay Strength Bar

**What:** Visual bar showing computed pheromone signal strength
**When to use:** Any command that displays pheromone information (status, build, resume-colony)

The commands already instruct Claude to compute pheromone decay using:
```
current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)
```

This phase adds a visual bar representation. Template:

```
For each active pheromone, display:

  {TYPE} [{filled_bar}{empty_space}] {strength:.2f}  "{content}"

Where the bar has 20 characters total:
- Filled portion uses the character: ━
- Empty portion uses spaces
- Number of filled characters = round(current_strength * 20)

Example at strength 0.75:
  FOCUS [━━━━━━━━━━━━━━━     ] 0.75  "WebSocket security"

Example at strength 0.30:
  REDIRECT [━━━━━━              ] 0.30  "Don't use JWT"

If half_life_seconds is null (persistent signal like INIT):
  INIT [━━━━━━━━━━━━━━━━━━━━] 1.00  "Build a REST API" (persistent)
```

**Source:** v2's `status.md` used `show_progress_bar()` bash function. v2's `12-RESEARCH.md` documented this pattern with `━` characters and 20-character width.

### Pattern 4: Worker Status Grouping

**What:** Workers grouped by status with emoji indicators
**When to use:** Status command and any command showing colony worker state

Template:

```
Display workers grouped by status:

WORKERS

  Active:
    {emoji} {worker_name}: {status_detail}

  Idle:
    {emoji} {worker_name}

  Error:
    {emoji} {worker_name}: {error_detail}

Status emoji mapping:
  active  -> ant emoji
  idle    -> white circle emoji
  error   -> red circle emoji

If all workers are idle (common case), display:
  All workers idle — ready for tasking
```

**Source:** v2's `status.md` Step 5 grouped workers by activity state with emoji indicators. The v2 `12-CONTEXT.md` established the emoji mapping (active=ant, idle=white circle, error=red circle, pending=hourglass).

### Pattern 5: Section Dividers

**What:** Visual separators between output sections
**When to use:** Between major sections in status, phase list, and results display

Template:

```
Between sections, use a divider line:

---------------------------------------------------
```

Or for subsections:

```
  ---
```

### Anti-Patterns to Avoid

- **Adding bash code blocks to command prompts:** The v3 commands use Read/Write tools directly. Do NOT add bash scripts or jq commands. The formatting is in the output template text that Claude follows.
- **Interactive progress updates:** Claude outputs text sequentially, it cannot update previous lines. Don't instruct Claude to "update" a progress display. Instead, show progress at checkpoints or at the end.
- **Over-engineering header width:** Headers should be a fixed width (~50-55 characters). Don't calculate dynamic widths.
- **Pheromone bars without numbers:** Always include the numeric strength value alongside the bar. The bar alone is hard to read precisely.
- **Emojis without text labels:** Always pair emojis with text (e.g., "ant ACTIVE" not just "ant emoji"). This was established in v2 phase 12 for accessibility.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Progress bar rendering | Complex character math | Simple instruction: "N filled out of 20" | Claude can count characters; don't over-specify |
| Box-drawing alignment | Character-width calculations | Fixed-width template strings | Just provide the exact template to reproduce |
| Step progress tracking | State machine in prompt | Ordered list with status markers | Claude follows instructions sequentially; it knows what step it's on |
| Worker status extraction | JSON parsing instructions | "Read the workers object from COLONY_STATE.json" | Claude already parses JSON natively via Read tool |
| Pheromone decay computation | External scripts | Inline formula in prompt text | Already present in build.md and status.md; just add visual output |

**Key insight:** This phase adds OUTPUT TEMPLATES to existing prompts. Claude already does all the computation (reads JSON, computes decay, determines status). The gap is purely in how it FORMATS the output. Keep the additions focused on formatting instructions, not computation logic.

## Common Pitfalls

### Pitfall 1: Treating Command Files as Scripts

**What goes wrong:** Adding bash code blocks, jq commands, or Python to command prompts
**Why it happens:** The v2 commands WERE script-based (source event-bus.sh, jq queries). It's tempting to copy that pattern.
**How to avoid:** Remember the v3 constraint: commands use Read/Write/Task tools. Claude reads JSON with Read tool, not jq. All output formatting is in natural language instructions.
**Warning signs:** Seeing `source`, `jq`, `echo`, `printf` in the command prompt edits.

### Pitfall 2: Making Output Templates Too Rigid

**What goes wrong:** Specifying exact character positions, column counts, and padding that Claude can't reliably reproduce
**Why it happens:** Trying to match v2's pixel-perfect bash output
**How to avoid:** Use templates that show the PATTERN, not exact spacing. Claude will approximate the layout well enough.
**Warning signs:** Instructions like "pad to exactly 60 characters" or "align column at position 40".

### Pitfall 3: Forgetting That Some Commands Have No Pheromones to Display

**What goes wrong:** Adding pheromone display to commands where the pheromone array might be empty
**Why it happens:** Not checking the empty case
**How to avoid:** Every pheromone display instruction should include: "If no active signals, display: (none active)"
**Warning signs:** Missing empty-state handling in output templates.

### Pitfall 4: Duplicating Visual Instructions Across Commands

**What goes wrong:** Copy-pasting the same pheromone bar or worker grouping template into 7 command files, making updates painful
**Why it happens:** No shared template mechanism in prompt files
**How to avoid:** Keep visual templates brief and consistent. Define the pattern once in status.md (the primary visual command) and use shorter references in others.
**Warning signs:** Identical 20-line blocks in multiple command files.

### Pitfall 5: Box-Drawing Characters Mangled by Markdown Rendering

**What goes wrong:** Box-drawing characters inside markdown code blocks render differently than expected
**Why it happens:** The command files are markdown. Characters inside ``` blocks are treated as code.
**How to avoid:** Put output templates inside code blocks (```) so they render literally. Claude will reproduce them exactly.
**Warning signs:** Characters rendering as HTML entities or being interpreted as markdown formatting.

## Code Examples

These are NOT code to execute. They are OUTPUT TEMPLATE EXAMPLES to include in command prompts.

### Box-Drawing Header (for status.md)

```
Output this header:

╔══════════════════════════════════════════════════════╗
║  AETHER COLONY STATUS                                ║
╠══════════════════════════════════════════════════════╣
║  Session: <session_id>                               ║
║  State:   <state>                                    ║
║  Goal:    "<goal>"                                   ║
╚══════════════════════════════════════════════════════╝
```

### Simpler Box Header (for init.md, build.md, continue.md, phase.md)

```
Output this header:

╔══════════════════════════════════════════════════════╗
║  AETHER COLONY :: <COMMAND_NAME>                     ║
╚══════════════════════════════════════════════════════╝
```

### Step Progress (for init.md - 5 steps)

```
After all steps complete, display:

  [✓] Step 1: Validate Input
  [✓] Step 2: Read Current State
  [✓] Step 3: Write Colony State
  [✓] Step 4: Emit INIT Pheromone
  [✓] Step 5: Display Result
```

### Step Progress (for build.md - 7 steps)

```
After all steps complete, display:

  [✓] Step 1: Validate
  [✓] Step 2: Read State
  [✓] Step 3: Compute Active Pheromones
  [✓] Step 4: Update State
  [✓] Step 5: Spawn Colony Ant
  [✓] Step 6: Record Outcome
  [✓] Step 7: Display Results
```

### Pheromone Decay Bar (for status.md, build.md)

```
For each pheromone signal in pheromones.json, compute and display:

1. Calculate current_strength:
   - If half_life_seconds is null: current_strength = original strength (persistent)
   - Otherwise: current_strength = strength * e^(-0.693 * elapsed_seconds / half_life_seconds)
   - If current_strength < 0.05: signal has expired, skip it

2. Display as:
   {TYPE} [{"━" * round(current_strength * 20)}{"  " * (20 - round(current_strength * 20))}] {current_strength:.2f}
     "{content}"

Example output:
  INIT     [━━━━━━━━━━━━━━━━━━━━] 1.00  (persistent)
    "Build a REST API with authentication"
  FOCUS    [━━━━━━━━━━━━━━━     ] 0.75
    "WebSocket security"
  REDIRECT [━━━━━━              ] 0.30
    "Don't use JWT for sessions"

If no active signals:
  (no active pheromones)
```

### Worker Status Grouping (for status.md)

```
Read the workers object from COLONY_STATE.json and display grouped by status:

WORKERS

  🐜 Active:
    <worker_name>: currently executing

  ⚪ Idle:
    <worker_name>, <worker_name>, ...

  🔴 Error:
    <worker_name>: <error detail>

If all workers show "idle" status (the common case):
  All 6 workers idle — colony ready

Summary line:
  🐜 <N> active | ⚪ <N> idle | 🔴 <N> error
```

## State of the Art

| Old Approach (v2) | Current Approach (v3) | Impact on Phase 14 |
|--------------------|-----------------------|---------------------|
| Bash scripts generating formatted output | Claude follows prompt instructions | Visual templates go in prompt text, not code |
| `jq` for JSON parsing | Claude's Read tool | No jq queries needed; Claude reads JSON natively |
| `show_progress_bar()` bash function | Output template with bar example | Describe the pattern, not the algorithm |
| 456-line status.md with 12 bash steps | 89-line status.md with 3 simple steps | Add formatting to existing steps, don't restructure |
| `source .aether/utils/event-bus.sh` | Direct Read/Write tool calls | No utility scripts to source |
| Step counter with bash arrays | Sequential output with markers | Show completed list, not live updates |

**Key shift from v2 to v3:** In v2, the prompt contained CODE that generated output. In v3, the prompt contains INSTRUCTIONS that Claude follows to produce output. Phase 14 adds more detailed output instructions, not more code.

## Files Requiring Modification

### Plan 14-01: Box-Drawing Headers and Step Progress

Files to modify:
1. **`.claude/commands/ant/init.md`** (currently 109 lines)
   - Add box-drawing header template to Step 5 (Display Result)
   - Add step progress indicator to the display output
   - Steps: Validate, Read State, Write State, Emit Pheromone, Display Result (5 steps)

2. **`.claude/commands/ant/build.md`** (currently 167 lines)
   - Add box-drawing header template to Step 5 spawn output and Step 7 results
   - Add step progress indicator showing all 7 steps completed
   - Steps: Validate, Read State, Compute Pheromones, Update State, Spawn Ant, Record Outcome, Display Results

3. **`.claude/commands/ant/continue.md`** (currently 69 lines)
   - Add box-drawing header template to Step 5 (Display Result)
   - Add step progress indicator (5 steps: Read State, Determine Next, Clean Pheromones, Update State, Display)

4. **`.claude/commands/ant/status.md`** (currently 89 lines)
   - Add rich box-drawing header with session/state/goal info
   - This is the primary visual command; gets the richest header

5. **`.claude/commands/ant/phase.md`** (currently 72 lines)
   - Add box-drawing header template to both single-phase and list views

### Plan 14-02: Pheromone Decay Bars and Worker Grouping

Files to modify:
1. **`.claude/commands/ant/status.md`** (primary target)
   - Step 2: Add pheromone decay bar display template
   - Step 3: Add worker grouping by status with emoji indicators
   - Add section dividers between status sections

2. **`.claude/commands/ant/build.md`**
   - Step 3: Enhance pheromone display with decay strength bars

3. **`.claude/commands/ant/resume-colony.md`** (currently 66 lines)
   - Step 3: Add pheromone decay bars to the restored state display
   - Add worker grouping to the restored state display

4. **`.claude/commands/ant/pause-colony.md`** (currently 86 lines)
   - Step 5: Add visual formatting to the pause confirmation display

## Open Questions

1. **How verbose should intermediate step progress be?**
   - What we know: Claude outputs text sequentially and cannot update previous lines. v2 used bash to "redisplay" the progress after each step.
   - What's unclear: Whether to show progress at every step boundary or just at the end.
   - Recommendation: Show a brief "Step N: ..." line as Claude works through each step, then show the full completed progress list at the end. This gives real-time feel without attempting interactive updates.

2. **Should ALL 12 commands get box-drawing headers?**
   - What we know: The requirements say "any major command." The smaller commands (focus, redirect, feedback) have brief output.
   - What's unclear: Whether a full box-drawing header on a simple pheromone emission command is overkill.
   - Recommendation: Major commands (init, build, continue, status, phase) get full box-drawing headers. Smaller commands (focus, redirect, feedback, pause-colony, resume-colony, colonize) get a simpler single-line header: `=== AETHER :: COMMAND ===`. The `ant.md` overview command gets the richest header as the entry point.

3. **Should worker grouping show individual workers or just counts?**
   - What we know: COLONY_STATE.json has 6 named workers (colonizer, route-setter, builder, watcher, scout, architect) with status values.
   - What's unclear: In the common case, all 6 workers are "idle". A grouped list of 6 idle workers is verbose.
   - Recommendation: In the common case (all idle), show a compact summary: "All 6 workers idle -- colony ready". Only expand to grouped display when there are mixed statuses.

## Sources

### Primary (HIGH confidence)
- **Codebase analysis** of all 12 current command files in `.claude/commands/ant/` (Read tool)
- **v2 command history** via `git show 5fd89f7:.claude/commands/ant/status.md` (456-line v2 version with full visual dashboard)
- **v2 visualization.py** via `git show 5fd89f7:.aether/visualization.py` (box-drawing patterns, progress bar rendering)
- **v2 init.md** via `git show 5fd89f7:.claude/commands/ant/init.md` (step progress tracker pattern)
- **Phase 12 research** at `.planning/phases/12-visual-indicators-documentation/12-RESEARCH.md` (visual indicator patterns, emoji mapping, progress bar format)
- **Phase 12 context** at `.planning/phases/12-visual-indicators-documentation/12-CONTEXT.md` (emoji design decisions, bar width, grouping strategy)
- **V3 lost features research** at `.planning/research/V3_LOST_FEATURES.md` (comprehensive inventory of what was lost)
- **PROJECT.md** at `.planning/PROJECT.md` (constraints: no new commands, no new scripts, enrich existing prompts)
- **ROADMAP.md** at `.planning/ROADMAP.md` (phase 14 requirements VIS-01 through VIS-04)

### Secondary (MEDIUM confidence)
- None needed. All findings derived from primary codebase analysis.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- No libraries needed; pure formatting in prompt text
- Architecture: HIGH -- Clear understanding of Claude-native prompt model from reading all 12 current commands and comparing with v2 versions
- Pitfalls: HIGH -- Derived from direct comparison of v2 (code-based) vs v3 (prompt-based) architecture
- File modification list: HIGH -- Direct analysis of each command file's structure and line counts

**Research date:** 2026-02-03
**Valid until:** 90 days (stable domain -- Unicode characters and prompt formatting patterns don't change)
