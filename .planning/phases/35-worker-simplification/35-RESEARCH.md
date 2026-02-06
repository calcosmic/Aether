# Phase 35: Worker Simplification - Research

**Researched:** 2026-02-06
**Domain:** Worker role specification consolidation
**Confidence:** HIGH

## Summary

This research analyzed the 6 existing worker spec files (1,866 total lines) to identify what must be preserved versus what can be removed per SIMP-04 requirements and CONTEXT.md decisions. The current worker specs contain massive duplication: approximately 185 lines of boilerplate repeated across all 6 workers (1,110 lines total), plus role-specific content that is overly prescriptive.

The simplification approach is straightforward: extract the core role identity (purpose, when to use, signal keywords) into a single consolidated file, move all shared boilerplate to a small "All Workers" section, and eliminate the pheromone math, validation checklists, and spawning ceremony that added no real value.

**Primary recommendation:** Create a single `workers.md` file with a ~30 line shared section followed by ~25-30 lines per role, targeting 200 total lines through aggressive boilerplate removal.

## Content Analysis

### Current Line Breakdown by Section

Each worker spec contains these sections (approximate lines per worker):

| Section | Lines | Verdict |
|---------|-------|---------|
| Header + Purpose | 8 | **KEEP** (compress to 3-5) |
| Visual Identity | 32 | **REMOVE** (move emoji to role line) |
| Pheromone Sensitivity | 8 | **REPLACE** with keyword list |
| Pheromone Math | 28 | **REMOVE** (no more math) |
| Combination Effects | 12 | **REMOVE** (too prescriptive) |
| Feedback Interpretation | 12 | **REMOVE** (trust Claude) |
| Event Awareness | 20 | **REMOVE** (workers read what they need) |
| Memory Reading | 20 | **REMOVE** (workers read what they need) |
| Workflow | 10 | **KEEP** (compress to 3-5) |
| Role-specific content | 15-100 | **KEEP** (compress heavily) |
| Output Format | 15-35 | **REPLACE** with 5-line template |
| Quality Standards | 10 | **REMOVE** (implicit) |
| Activity Log | 25 | **MOVE** to shared section |
| Post-Action Validation | 25 | **REMOVE** |
| Requesting Sub-Spawns | 35 | **MOVE** to shared section |

**Watcher-specific extras (205 lines):**
- Execution Verification: 45 lines -> REMOVE (trust watcher)
- Specialist Modes: 115 lines -> REMOVE entirely
- Scoring Rubric: 45 lines -> REMOVE (trust watcher)

### Boilerplate vs Unique Content

**Identical across all 6 workers (~185 lines each, 1,110 total):**
- Visual Identity template (32 lines)
- Pheromone Math instructions (28 lines)
- Event Awareness reading protocol (20 lines)
- Memory Reading protocol (20 lines)
- Activity Log section (25 lines)
- Post-Action Validation (25 lines)
- Requesting Sub-Spawns (35 lines)

**Role-specific but removable (~55 lines each, 330 total):**
- Pheromone Sensitivity table (8 lines) - replaced by keyword list
- Combination Effects table (12 lines) - trust Claude
- Feedback Interpretation table (12 lines) - trust Claude
- Verbose output templates (15-35 lines) - standardize

**Truly role-specific content (to preserve, compressed):**
- Purpose statement (keep, compress)
- Workflow hints (keep, compress)
- Quality standards (implicit)

## Standard Stack

Not applicable - this is documentation consolidation, not code implementation.

### Tools/Commands Involved

| Tool | Purpose |
|------|---------|
| Read | Read existing worker specs |
| Write | Create consolidated workers.md |
| Bash | Count lines for verification |

## Architecture Patterns

### Recommended File Structure

```
.aether/
└── workers.md           # Single consolidated file (~200 lines)
```

**Remove entirely:**
```
.aether/workers/
├── architect-ant.md     # DELETE
├── builder-ant.md       # DELETE
├── colonizer-ant.md     # DELETE
├── route-setter-ant.md  # DELETE
├── scout-ant.md         # DELETE
└── watcher-ant.md       # DELETE
```

### Pattern: Consolidated Role Definition

**What:** Single file with shared section + per-role definitions
**Structure:**

```markdown
# Worker Roles

## All Workers (shared)
[Activity log pattern - 5 lines]
[Spawn request format - 10 lines]
[Visual identity pattern - 5 lines]
[Standard output template - 10 lines]
Total: ~30 lines

## Builder
[Purpose - 2 sentences]
[When to use - 1 line]
[Signals - keyword list]
[Workflow hints - 5 lines]
Total: ~25 lines

## Watcher
[Purpose - 2 sentences]
[When to use - 1 line]
[Signals - keyword list]
[Workflow hints - 8 lines, includes quality gate role]
Total: ~28 lines

... (4 more roles)
```

### Pattern: Signal Keyword Lists (from CONTEXT.md)

Per-role keywords instead of sensitivity matrices:

```markdown
**Signals:** FOCUS, REDIRECT
```

Interpretation: "If you see FOCUS, prioritize that area. If you see REDIRECT, avoid that approach."

No math. No thresholds. No combination effects tables.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Role selection | Complex matching rules | Claude's judgment | One-liner hints sufficient |
| Signal response | Sensitivity math | Keyword recognition | "If X, do Y" is enough |
| Output formatting | Elaborate templates | Standard 5-line format | Let worker format naturally |
| Quality validation | Detailed checklists | Implicit standards | Workers know what "quality" means |

**Key insight:** The current worker specs treat Claude like a rules engine. Claude is an LLM that understands intent. "Build this feature" is enough; we don't need 50 lines of "how to build features."

## Common Pitfalls

### Pitfall 1: Under-specifying Role Boundaries
**What goes wrong:** Without clear role purpose, workers become generic and interchangeable
**Why it happens:** Over-simplification loses role identity
**How to avoid:** Keep purpose statement (2-3 sentences) and "when to use" hint
**Warning signs:** Queen always picks builder for everything

### Pitfall 2: Losing the Quality Gate
**What goes wrong:** Watcher stops being special, quality reviews don't happen
**Why it happens:** Watcher role flattened too much
**How to avoid:** Watcher section explicitly mentions review-before-advance role
**Warning signs:** Phases advance without quality check

### Pitfall 3: Breaking Command Integration
**What goes wrong:** build.md and continue.md break when looking for worker specs
**Why it happens:** Commands hard-code paths like `~/.aether/workers/{caste}-ant.md`
**How to avoid:** Update commands to reference new location, or keep old structure as symlinks
**Warning signs:** "File not found" errors during build

### Pitfall 4: Preserving Too Much
**What goes wrong:** "Simplified" spec is still 800+ lines
**Why it happens:** Fear of losing functionality
**How to avoid:** Target 200 lines aggressively, trust Claude's judgment
**Warning signs:** Per-role sections exceeding 40 lines

## Integration Points

### Commands That Reference Worker Specs

**build.md (Step 5c.c):**
```markdown
c. **Read worker spec:** Read `~/.aether/workers/{caste}-ant.md`
```

**build.md (Step 6):**
```markdown
1. Read `~/.aether/workers/watcher-ant.md`
```

**Required changes:**
1. Update build.md to read `~/.aether/workers.md` instead
2. Extract relevant role section from consolidated file
3. OR: create stub files that redirect to consolidated file

### Command Sensitivity Matrix (build.md Step 4)

build.md contains a hardcoded sensitivity table:
```
                INIT  FOCUS  REDIRECT  FEEDBACK
  colonizer     1.0   0.7    0.3       0.5
  ...
```

**Decision:** This can be removed if signals become keyword-based per CONTEXT.md decisions.

## Transformation Strategy

### What to Keep (Verbatim or Compressed)

| Content | Current | Target | Notes |
|---------|---------|--------|-------|
| Role purpose | 2-4 sentences | 2 sentences | Core identity |
| "When to use" | None | 1 line | New addition per CONTEXT.md |
| Signal keywords | 8-line table | 1 line | e.g., "Signals: FOCUS, REDIRECT" |
| Workflow | 10 lines | 5 lines | Core steps only |

### What to Remove Entirely

| Content | Lines Saved | Reason |
|---------|-------------|--------|
| Pheromone math | 28 x 6 = 168 | SIMP-03 removes math |
| Combination effects | 12 x 6 = 72 | Trust Claude |
| Feedback interpretation | 12 x 6 = 72 | Trust Claude |
| Event/Memory reading | 40 x 6 = 240 | Workers read what they need |
| Post-action validation | 25 x 6 = 150 | Removed per CONTEXT.md |
| Visual identity (verbose) | 25 x 6 = 150 | Keep emoji only |
| Watcher specialist modes | 115 | Trust watcher judgment |
| Watcher scoring rubric | 45 | Trust watcher judgment |
| Watcher execution verification | 45 | Trust watcher judgment |

### What to Consolidate in Shared Section

| Content | Lines | Notes |
|---------|-------|-------|
| Activity log pattern | 5 | Command + brief usage |
| Spawn request format | 10 | Format + caste list |
| Visual identity | 5 | Emoji list + usage |
| Standard output template | 10 | Task/Status/Summary/Files/Next |
| **Total** | **~30** | |

## Target Output Format

Per CONTEXT.md, simplified standard template for all workers:

```markdown
## Output Format

Report using this structure:
- Task: {what you were asked to do}
- Status: completed / failed / blocked
- Summary: {1-2 sentences of what happened}
- Files: {only if you touched files}
- Next Steps / Recommendations: {required}
```

This replaces the 15-35 line elaborate templates currently in each worker spec.

## Line Budget

| Section | Lines | Running Total |
|---------|-------|---------------|
| Header | 5 | 5 |
| All Workers (shared) | 30 | 35 |
| Builder | 25 | 60 |
| Watcher | 28 | 88 |
| Scout | 25 | 113 |
| Colonizer | 25 | 138 |
| Architect | 25 | 163 |
| Route-setter | 25 | 188 |
| **Total** | **~188** | Target: 200 |

## Open Questions

1. **Command file updates:** Should build.md/continue.md be updated in this phase or a subsequent phase?
   - Recommendation: Update in this phase to maintain consistency
   - Risk: Scope creep if we tackle command simplification here

2. **Old worker files:** Delete entirely or leave stubs?
   - Recommendation: Delete entirely, update commands
   - Alternative: Leave stubs that `cat` the consolidated file

3. **Watcher special role:** How much watcher-specific content is needed for quality gates?
   - Recommendation: 3-5 extra lines mentioning review-before-advance role
   - The watcher's PURPOSE already implies quality checking

## Sources

### Primary (HIGH confidence)
- `.aether/workers/*.md` - Read all 6 worker specs in full
- `.claude/commands/ant/build.md` - Read to identify integration points
- `.claude/commands/ant/continue.md` - Read to identify integration points
- `.planning/phases/35-worker-simplification/35-CONTEXT.md` - User decisions

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` - SIMP-04 requirement definition

## Metadata

**Confidence breakdown:**
- Content analysis: HIGH - Direct file reading
- Removal decisions: HIGH - Based on CONTEXT.md user decisions
- Line budget: MEDIUM - Estimates, actual may vary +/- 20%
- Integration points: HIGH - Direct grep/read of command files

**Research date:** 2026-02-06
**Valid until:** 2026-03-06 (stable domain, internal documentation)
