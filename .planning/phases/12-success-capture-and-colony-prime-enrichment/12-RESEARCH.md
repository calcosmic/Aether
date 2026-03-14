# Phase 12: Success Capture and Colony-Prime Enrichment - Research

**Researched:** 2026-03-14
**Domain:** Playbook wiring (build-verify.md, build-complete.md) + shell subcommand modification (colony-prime in aether-utils.sh)
**Confidence:** HIGH (all relevant source files read directly; no external dependencies)

## Summary

Phase 12 adds two specific `memory-capture "success"` calls to existing build playbook steps and modifies the `colony-prime` subcommand to include rolling-summary entries directly in its output. Currently, only failures flow through the memory pipeline; success events are synthesized in build output JSON but never recorded in `learning-observations.json`. The rolling-summary log is populated by `memory-capture` but only reaches builders indirectly through `context-capsule`, which includes only the last 3 entries and drops them first under compact mode word limits.

The implementation surface is small: two new bash code blocks in playbook markdown files (~10 lines each), and one modification to `colony-prime` in `aether-utils.sh` (~15 lines) to read and format the last 5 rolling-summary entries as a dedicated section in `prompt_section`. No new subcommands, no new data files, no new JSON schemas.

**Primary recommendation:** Add `memory-capture "success"` calls at the two specified call sites (build-verify Step 5.7 for chaos resilience, build-complete Step 5.9 for pattern synthesis), then add a `--- RECENT ACTIVITY ---` section to `colony-prime` output that reads the last 5 rolling-summary entries directly (bypassing the word-limited context-capsule path).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
None -- all implementation decisions are at Claude's discretion.

### Claude's Discretion
- **Success recognition scope** -- Whether to capture only the two specified success types (chaos resilience in build-verify, pattern synthesis in build-complete) or also detect additional positive patterns. Success criteria define the minimum; Claude may expand if warranted.
- **Activity awareness depth** -- How much detail rolling-summary entries contain and how they're formatted in colony-prime output. Could be quick headlines or richer summaries with context.
- **Learning balance** -- Whether success entries carry the same weight as failure entries in the learning pipeline, or whether failures remain weighted higher for actionability.
- **Success entry format** -- How success entries in learning-observations.json are structured relative to existing failure entries.
- **Rolling-summary placement** -- Where in colony-prime output the last 5 rolling-summary entries appear and how prominent they are.

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MEM-01 | Success capture fires at build-verify (chaos resilience) and build-complete (pattern synthesis) call sites via memory-capture "success" | `memory-capture` already accepts "success" as event_type (line 5415). Call sites identified at build-verify.md Step 5.7 (after chaos results parsed) and build-complete.md Step 5.9 (synthesis patterns_observed). Exact insertion points documented below. |
| MEM-02 | Rolling-summary last 5 entries fed into colony-prime output so workers have recent activity awareness | `rolling-summary read --json` returns entries array. `colony-prime` currently includes rolling-summary only via `context-capsule --compact` which takes last 3 and drops them first under word limits. A direct read in `colony-prime` (bypassing context-capsule) is needed to guarantee last 5 entries appear. Insertion point documented below. |
</phase_requirements>

## Standard Stack

### Core (no new dependencies)

| Component | Location | Purpose | Role in Phase 12 |
|-----------|----------|---------|-------------------|
| aether-utils.sh | `.aether/aether-utils.sh` | 150 subcommands | Modify `colony-prime` subcommand (~line 7548) |
| build-verify.md | `.aether/docs/command-playbooks/build-verify.md` | Build verification playbook | Add success capture at Step 5.7 |
| build-complete.md | `.aether/docs/command-playbooks/build-complete.md` | Build completion playbook | Add success capture at Step 5.9 |

### Existing Subcommands Used (no modifications needed)

| Subcommand | Line | Purpose | Called By |
|------------|------|---------|-----------|
| `memory-capture` | 5402 | Pipeline: observe + pheromone + auto-promote | New calls in build-verify, build-complete |
| `rolling-summary read` | 8734 | Read last N entries as JSON | New call in colony-prime |
| `rolling-summary add` | 8712 | Append entry to bounded log | Already called by memory-capture (line 5501) |
| `learning-observe` | 5148 | Record observation, increment count | Called internally by memory-capture |

### Subcommand Requiring Modification

| Subcommand | Line | Current Behavior | Required Change |
|------------|------|-----------------|-----------------|
| `colony-prime` | 7548 | Assembles wisdom + signals + learnings + decisions + blockers + context-capsule | Add dedicated rolling-summary section with last 5 entries |

## Architecture Patterns

### Current Colony-Prime Output Structure

The `colony-prime` subcommand builds `cp_final_prompt` by concatenating sections in this order:

```
1. --- QUEEN WISDOM ---          (philosophies, patterns, redirects, stack, decrees)
2. --- CONTEXT CAPSULE ---       (goal, state, phase, signals, decisions, risks, recent narrative)
3. --- PHASE LEARNINGS ---       (validated learnings from previous phases)
4. --- KEY DECISIONS ---         (from CONTEXT.md Recent Decisions table)
5. --- BLOCKER WARNINGS ---      (unresolved blocker flags)
6. [pheromone signals section]   (from pheromone-prime)
```

The context-capsule (section 2) includes up to 3 rolling-summary entries under "Recent narrative:", but drops this section first when compact mode exceeds the word limit (220 words). Since `colony-prime` calls `context-capsule --compact`, rolling-summary entries are the first thing cut.

### Recommended Colony-Prime Output Structure (After Phase 12)

```
1. --- QUEEN WISDOM ---
2. --- CONTEXT CAPSULE ---       (keep as-is, still includes its 3 entries)
3. --- PHASE LEARNINGS ---
4. --- KEY DECISIONS ---
5. --- BLOCKER WARNINGS ---
6. --- RECENT ACTIVITY ---       [NEW: last 5 rolling-summary entries, never truncated]
7. [pheromone signals section]
```

The new "RECENT ACTIVITY" section goes after blockers and before pheromone signals. It is NOT inside context-capsule so it cannot be truncated by word limits. It reads directly from `rolling-summary.log`.

### Pattern: Memory-Capture Call Site Format

All existing `memory-capture` calls in playbook steps follow this exact pattern:

```bash
bash .aether/aether-utils.sh memory-capture \
  "<event_type>" \
  "<specific content string>" \
  "<wisdom_type>" \
  "<source_tag>" 2>/dev/null || true
```

Source: build-verify.md lines 315-319 (failure call), build-wave.md Step 5.2 (builder failure call).

New success calls MUST follow the same pattern:
- `2>/dev/null || true` for non-blocking execution
- Source tag matching the agent that generated the event
- Content string specific enough to be meaningful in learning-observations.json

### Pattern: Success Event Flow Through Memory Pipeline

When `memory-capture "success"` is called, the internal pipeline is:

```
memory-capture "success" "<content>" "pattern" "<source>"
  |
  +-- learning-observe: records to learning-observations.json
  |     content_hash: SHA256 of content (deduplication key)
  |     wisdom_type: "pattern" (success maps to pattern by default, line 5424)
  |     observation_count: increments on repeat
  |
  +-- pheromone-write FEEDBACK: "Learning captured: <content>"
  |     strength: 0.6 (line 5468)
  |     source: <source_tag>
  |     ttl: 30d
  |
  +-- learning-promote-auto: checks if threshold met for QUEEN.md promotion
  |
  +-- rolling-summary add: appends "success" event to rolling-summary.log
```

This is identical to the `learning` event flow. The `success` event type is already handled (line 5465: `learning|success)` case). No modification to `memory-capture` is needed.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Recording success observations | Custom JSON writes to learning-observations.json | `memory-capture "success"` subcommand | memory-capture handles deduplication, pheromone emission, auto-promotion, and rolling-summary in one atomic call |
| Reading rolling-summary entries | Custom file parsing of rolling-summary.log | `rolling-summary read --json` subcommand | Returns properly formatted JSON with entries array; handles missing file gracefully |
| Formatting colony-prime output | External template rendering | Append to `cp_final_prompt` string in existing colony-prime case branch | Follows the established pattern; output is a single concatenated string |

## Common Pitfalls

### Pitfall 1: Success Content Too Generic

**What goes wrong:** Calling `memory-capture "success" "Build completed successfully"` creates a generic observation that inflates counts across unrelated builds. Eventually promotes to QUEEN.md with no actionable content.

**Why it happens:** Success patterns from synthesis JSON (`learning.patterns_observed`) may contain generic descriptions if builders report broadly.

**How to avoid:** Content must include specifics: the pattern trigger + action + evidence. For chaos resilience: include the phase name and scenario count. For pattern synthesis: include the pattern type and trigger.

**Good example:** `"Chaos resilience strong (5/5 scenarios passed): edge case handling in auth module robust"`
**Bad example:** `"Build succeeded"`

### Pitfall 2: Rolling-Summary Section Duplicates Context-Capsule Entries

**What goes wrong:** Colony-prime already includes context-capsule, which has up to 3 rolling-summary entries under "Recent narrative:". Adding a separate "RECENT ACTIVITY" section with 5 entries means the last 3 entries appear twice in the prompt -- once in context-capsule and once in the new section.

**Why it happens:** Two independent paths read from the same `rolling-summary.log` file.

**How to avoid:** Either (a) remove the rolling-summary entries from context-capsule when colony-prime includes its own section, or (b) accept minor duplication since context-capsule may be truncated in compact mode anyway. Recommendation: Option (b) -- accept duplication. The context-capsule path may drop its entries under compact word limits, so the dedicated section serves as the guaranteed path. The overhead of 3 duplicate lines is negligible versus the risk of modifying context-capsule logic.

### Pitfall 3: Colony-Prime Modification Breaks Existing Output

**What goes wrong:** Adding the rolling-summary section changes the `prompt_section` string length, which could break parsing in downstream consumers that expect exact output format.

**Why it happens:** Colony-prime output is consumed as a free-form string injected into worker prompts. No downstream consumer parses it structurally.

**How to avoid:** This is a non-issue. The `prompt_section` is injected into worker prompts as `{prompt_section}` -- an arbitrary-length markdown block. Adding a new section is the same operation as the existing learnings, decisions, and blockers sections that were added in v1.0 and v1.1. No format contract exists beyond "it's a markdown string."

### Pitfall 4: Success Capture on Chaos "Strong" When Chaos Reports Errors

**What goes wrong:** Chaos ant reports `overall_resilience: "strong"` but also has `findings` with severity "medium" or "low". The resilience is strong overall, but the success capture ignores the findings.

**Why it happens:** `overall_resilience` is the chaos ant's aggregate assessment. Individual findings may exist but are below the critical/high threshold.

**How to avoid:** Gate the success capture on `overall_resilience == "strong"` as a condition. If strong, capture it. If moderate or weak, do not capture (existing failure capture handles high/critical findings). This aligns with the success criteria which says "chaos reports strong resilience."

## Code Examples

### MEM-01a: Success Capture in build-verify.md Step 5.7

**Insert location:** After the Chaos Ant completion log (line `bash .aether/aether-utils.sh spawn-complete`) and before Step 5.8.

**Condition:** `overall_resilience == "strong"`

```bash
# Success capture: chaos reports strong resilience (MEM-01)
if [[ "${overall_resilience}" == "strong" ]]; then
  bash .aether/aether-utils.sh memory-capture \
    "success" \
    "Chaos resilience strong: ${summary}" \
    "pattern" \
    "worker:chaos" 2>/dev/null || true
fi
```

Source: follows the exact pattern of the existing failure `memory-capture` call at lines 315-319 of build-verify.md, but inverted for the success case.

### MEM-01b: Success Capture in build-complete.md Step 5.9

**Insert location:** After synthesis JSON is constructed, within the synthesis step. Add after the graveyard recording block but before the error handoff update.

**Condition:** `learning.patterns_observed` array is non-empty in synthesis JSON.

```bash
# Success capture: pattern synthesis from build (MEM-01)
# Cap at 2 success captures per build to prevent inflation
success_capture_count=0
for pattern in patterns_observed; do
  [[ "$success_capture_count" -ge 2 ]] && break
  bash .aether/aether-utils.sh memory-capture \
    "success" \
    "${pattern.trigger}: ${pattern.action} (evidence: ${pattern.evidence})" \
    "pattern" \
    "worker:builder" 2>/dev/null || true
  ((success_capture_count++)) || true
done
```

Note: This is pseudocode showing the logic. In the actual playbook markdown, this will be a natural language instruction that the orchestrating agent executes, since playbooks are not directly executed bash scripts but rather instructions read by an AI agent.

### MEM-02: Rolling-Summary in colony-prime

**Insert location:** In `aether-utils.sh`, within the `colony-prime)` case branch, after the blocker flag injection section (after line ~7893 `# === END blocker flag injection ===`) and before the pheromone signals section (before line ~7895 `# Add pheromone signals section`).

```bash
# === Rolling-summary injection (MEM-02) ===
# Read last 5 entries directly (not via context-capsule which truncates)
cp_roll_count=5
cp_roll_entries=""
if [[ -f "$DATA_DIR/rolling-summary.log" ]]; then
  cp_roll_entries=$(tail -n "$cp_roll_count" "$DATA_DIR/rolling-summary.log" 2>/dev/null | \
    awk -F'|' 'NF >= 4 {printf "- [%s] %s: %s\n", $1, $2, $4}')
fi

if [[ -n "$cp_roll_entries" ]]; then
  cp_final_prompt+=$'\n'"--- RECENT ACTIVITY (Colony Narrative) ---"$'\n'
  cp_final_prompt+="$cp_roll_entries"$'\n'
  cp_final_prompt+="--- END RECENT ACTIVITY ---"$'\n'

  cp_roll_actual=$(echo "$cp_roll_entries" | grep -c '.' || echo "0")
  cp_log_line="$cp_log_line, $cp_roll_actual activity entries"
fi
# === END rolling-summary injection ===
```

Source: Pattern follows the existing blocker flag injection (lines 7870-7892) and decision injection (lines 7826-7840). Uses the same `cp_final_prompt+=` concatenation and `cp_log_line` augmentation pattern.

## State of the Art

| Component | Current State | After Phase 12 | Impact |
|-----------|---------------|----------------|--------|
| `memory-capture "success"` event type | Accepted by code (line 5415) but never called from any playbook | Called from build-verify Step 5.7 and build-complete Step 5.9 | Success events enter learning-observations.json for the first time |
| rolling-summary in colony-prime | Reaches builders only via context-capsule (3 entries, dropped first in compact mode) | Dedicated section in colony-prime (5 entries, never truncated) | Builders always see recent colony activity |
| learning-observations.json entries | Only failure and learning types | Also success type | Balanced observation history for promotion pipeline |

## Open Questions

1. **Should success entries have different promotion thresholds?**
   - What we know: The default promotion threshold for "pattern" wisdom_type is 1 (QUEEN.md METADATA `promotion_thresholds.pattern: 1`). Both successes and failures use wisdom_type "pattern" and share the same threshold.
   - What's unclear: Whether success patterns should promote as aggressively as failure patterns.
   - Recommendation: Keep the same threshold for now. The auto-promotion pipeline (`learning-promote-auto`) already requires recurrence (observation_count >= 2 per the test at line 393-404 in learning-pipeline.test.js). Success events that recur across colonies are genuine patterns worth promoting.

2. **Should the 3 entries in context-capsule be removed once colony-prime has its own section?**
   - What we know: Context-capsule's "Recent narrative" section (3 entries) overlaps with the new colony-prime "RECENT ACTIVITY" section (5 entries). Both read from rolling-summary.log.
   - What's unclear: Whether removing context-capsule entries would break any downstream consumers that only use context-capsule (e.g., non-colony-prime paths).
   - Recommendation: Leave context-capsule unchanged. The duplication is minor (max 3 repeated lines), and context-capsule is used in other contexts beyond colony-prime. Modifying it risks side effects.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 5402-5504: `memory-capture` subcommand -- event types, pipeline flow, rolling-summary call
- `.aether/aether-utils.sh` lines 7548-7933: `colony-prime` subcommand -- full output assembly, section ordering, context-capsule integration
- `.aether/aether-utils.sh` lines 8704-8762: `rolling-summary` subcommand -- add/read operations, pipe-delimited format, bounded at 15 entries
- `.aether/aether-utils.sh` lines 8764-8957: `context-capsule` subcommand -- rolling-summary reads last 3, compact mode truncation order
- `.aether/aether-utils.sh` lines 5148-5247: `learning-observe` subcommand -- content hash deduplication, observation count increment
- `.aether/docs/command-playbooks/build-verify.md` lines 228-361: Steps 5.6-5.8 -- chaos ant spawning, results processing, existing failure memory-capture calls
- `.aether/docs/command-playbooks/build-complete.md` lines 1-313: Steps 5.9-8 -- synthesis, graveyard, handoff, promotion check, build summary
- `.planning/research/ARCHITECTURE.md`: Integration gap analysis, pipeline flow diagrams, anti-patterns
- `.planning/research/PITFALLS.md`: Competing instructions, generic content inflation, silent failure risks
- `tests/integration/learning-pipeline.test.js`: Test patterns for memory-capture, learning-observe, promotion pipeline

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md`: MEM-01 and MEM-02 requirement definitions
- `.planning/ROADMAP.md`: Phase 12 success criteria (last 5 rolling-summary entries)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all files exist and were read directly
- Architecture: HIGH -- colony-prime output assembly fully traced; insertion points identified at exact line numbers
- Pitfalls: HIGH -- all pitfalls derived from direct codebase patterns (generic content inflation confirmed by existing research, duplication path traced through context-capsule code)

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (stable -- this is internal shell scripting and markdown, not external dependencies)
