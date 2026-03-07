# Phase 2: Learnings Injection - Research

**Researched:** 2026-03-06
**Domain:** Bash shell scripting (aether-utils.sh), colony-prime prompt assembly, JSON state management
**Confidence:** HIGH

## Summary

Phase 2 wires `memory.phase_learnings` from COLONY_STATE.json into the colony-prime output so builders automatically receive validated learnings from previous phases. Currently, colony-prime assembles its `prompt_section` from three sources: QUEEN wisdom, context-capsule, and pheromone-prime (signals + instincts). Phase learnings are stored in COLONY_STATE.json but are never read or formatted for builder injection -- this is the gap.

The implementation follows the exact pattern established in Phase 1 (instinct pipeline): add a reading/formatting step inside colony-prime that extracts validated learnings from previous phases, formats them as actionable guidance text, and appends the result to `prompt_section`. No new subcommands are needed. The `learning-inject` subcommand that already exists reads from a separate `learnings.json` file (global/cross-colony learnings) and is NOT the right tool for this -- Phase 2 reads from `memory.phase_learnings` in COLONY_STATE.json directly.

The data structure is well-defined: each phase_learning entry has `{id, phase, phase_name, learnings: [{claim, status, tested, evidence, disproven_by}], timestamp}`. The `status` field has three values: `hypothesis`, `validated`, `disproven`. Only entries with `status: "validated"` should be injected (per success criteria requirement 3). The `phase` field enables filtering to only include learnings from phases prior to `current_phase`.

**Primary recommendation:** Add a phase-learnings extraction block to colony-prime (after context-capsule, before pheromone signals) that reads `memory.phase_learnings` from COLONY_STATE.json, filters for validated claims from previous phases, formats them as actionable text grouped by phase, and appends to `prompt_section`. Update `log_line` to include learning count. Write integration tests following the instinct-pipeline.test.js pattern.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LEARN-01 | Validated phase learnings auto-inject into builder prompts via colony-prime | colony-prime (line 7451-7671) assembles prompt_section but currently has NO phase_learnings extraction; the injection point is identified between context-capsule (line 7625-7631) and pheromone signals (line 7633-7636) |
| LEARN-04 | Phase learnings from previous phases visible to current phase builders | memory.phase_learnings stores per-phase entries with a `phase` field; filtering `where phase < current_phase` enables cross-phase visibility; current_phase is read at colony-prime line 7551 (via context-capsule) but must be read directly in colony-prime itself |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | colony-prime subcommand modification for learnings injection | Single source of truth for all prompt assembly |
| jq | System-installed | JSON filtering and formatting of phase_learnings from COLONY_STATE.json | Used throughout aether-utils.sh; handles the filtering (status=validated, phase < current) and grouping |
| ava | Installed in package.json | Integration test runner | Project standard, all Phase 1 tests use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| colony-prime | aether-utils.sh subcommand (line 7451) | Unified priming -- add phase learnings section here | Modify to extract and format learnings from COLONY_STATE.json |
| build-context.md | Playbook | Calls colony-prime, injects prompt_section into builder | Already wired -- NO changes needed (same as Phase 1 finding) |
| build-wave.md | Playbook | Builder prompt template with `{ prompt_section }` placeholder | Already wired -- NO changes needed |
| continue-advance.md | Playbook | Writes phase_learnings to COLONY_STATE.json during continue | Already writes correctly (verified in source); no changes needed for Phase 2 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Inline phase_learnings extraction in colony-prime | New subcommand `learning-prime` | Adds unnecessary complexity; Phase 1 proved inline extraction works (instincts in pheromone-prime) |
| Reading from memory.phase_learnings | Using existing `learning-inject` subcommand | learning-inject reads from learnings.json (global cross-colony), NOT from COLONY_STATE.json memory.phase_learnings; wrong data source |
| Modifying pheromone-prime | Adding to colony-prime directly | Learnings are not pheromones or instincts; colony-prime is the correct assembly point for new context sections |

## Architecture Patterns

### File Touch Map

```
MODIFY:
  .aether/aether-utils.sh                                # colony-prime: add phase learnings extraction and formatting

CREATE:
  tests/integration/learnings-injection.test.js           # End-to-end learnings injection tests

NO CHANGE NEEDED:
  .aether/docs/command-playbooks/build-context.md         # Already calls colony-prime, injects prompt_section
  .aether/docs/command-playbooks/build-wave.md            # Already injects { prompt_section } into builder prompts
  .aether/docs/command-playbooks/continue-advance.md      # Already writes phase_learnings correctly
```

### Pattern 1: Colony-Prime Section Assembly
**What:** colony-prime builds `prompt_section` by concatenating discrete sections in order: QUEEN wisdom -> context-capsule -> pheromone signals (including instincts). Each section is conditionally included only when data exists.
**When to use:** Adding phase learnings follows this same pattern -- extract, format, conditionally append.
**Insertion point:** Between context-capsule (line 7631) and pheromone signals (line 7633). This positions learnings after colony status context but before active signals, which matches the information hierarchy (historical context before current guidance).

```bash
# Current colony-prime assembly order (lines 7593-7636):
cp_final_prompt=""

# 1. QUEEN wisdom (lines 7596-7622)
cp_final_prompt+="--- QUEEN WISDOM ---"
# ... philosophies, patterns, redirects, stack, decrees ...
cp_final_prompt+="--- END QUEEN WISDOM ---"

# 2. Context capsule (lines 7625-7631)
cp_capsule_raw=$("$SCRIPT_DIR/aether-utils.sh" context-capsule --compact --json 2>/dev/null)
cp_final_prompt+="$cp_capsule_prompt"

# === INSERT PHASE LEARNINGS HERE === (new for Phase 2)

# 3. Pheromone signals + instincts (lines 7633-7636)
cp_final_prompt+="$cp_prompt_section"
```

### Pattern 2: Phase Learnings Data Shape
**What:** Each entry in `memory.phase_learnings` has a nested structure with individual claims inside a learnings array
**When to use:** jq extraction must handle the nested structure

```json
{
  "id": "learning_1709736000",
  "phase": 1,
  "phase_name": "instinct-pipeline",
  "learnings": [
    {
      "claim": "Use >= 0.7 confidence threshold for instinct creation to avoid noise",
      "status": "validated",
      "tested": true,
      "evidence": "Lower thresholds produced instincts from unverified patterns"
    },
    {
      "claim": "Midden error patterns should get higher confidence than success patterns",
      "status": "hypothesis",
      "tested": false,
      "evidence": "Theoretical assumption, not yet tested"
    }
  ],
  "timestamp": "2026-03-06T12:00:00Z"
}
```

**Filtering rules:**
1. Only include entries where `phase < current_phase` (previous phases only)
2. Within each entry, only include claims where `status == "validated"` (not hypothesis or disproven)
3. Inherited learnings (phase == "inherited") should also be included if validated

### Pattern 3: Log Line Update
**What:** colony-prime's `log_line` reports what was primed: currently `"Primed: N signals, M instincts"`. Phase 2 should extend this.
**When to use:** After extracting learnings count, update log_line to include learning count.
**Target format:** `"Primed: N signals, M instincts, L learnings"`

### Anti-Patterns to Avoid
- **Modifying build-context.md or build-wave.md:** The `{ prompt_section }` placeholder in the builder prompt template already injects whatever colony-prime returns. This was confirmed in Phase 1 (01-02 SUMMARY: "No changes needed to build-context.md or build-wave.md"). Do NOT add separate learnings injection logic there.
- **Creating a new subcommand:** colony-prime is the assembly point. Adding a `learning-prime` subcommand would create indirection without benefit. Read directly from COLONY_STATE.json within colony-prime (same pattern as instinct-read within pheromone-prime).
- **Using learning-inject subcommand:** This reads from `learnings.json` (global cross-colony file), NOT from `memory.phase_learnings` in COLONY_STATE.json. Different data source, different purpose.
- **Injecting raw JSON:** Success criteria explicitly requires "actionable guidance (not raw JSON)". Must format claims as readable text.
- **Including hypothesis or disproven learnings:** Success criteria requires "only validated learnings". The jq filter must check `status == "validated"`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Reading COLONY_STATE.json | Custom file parsing | `jq` with DATA_DIR path variable | Already established pattern in colony-prime (lines 7551, 7571) |
| Phase number comparison | String matching | jq numeric comparison `.phase < $current_phase` | Phase numbers can be integers or "inherited" string; jq handles type coercion |
| Prompt section assembly | Template engine | String concatenation with `$'\n'` | Exact pattern used by QUEEN wisdom and instinct sections in colony-prime |
| JSON escaping for output | Manual escaping | `jq -Rs '.'` pipe pattern | Already used at colony-prime line 7639 |
| Duplicate claim filtering | Custom dedup logic | jq `unique_by(.claim)` | Inherited learnings may duplicate phase learnings; jq handles this |

**Key insight:** The extraction and formatting code is straightforward jq + bash string assembly. The pattern is identical to how pheromone-prime reads instincts from COLONY_STATE.json and how colony-prime reads QUEEN wisdom. No novel infrastructure needed.

## Common Pitfalls

### Pitfall 1: "inherited" Phase Learnings Handling
**What goes wrong:** Phase learnings from sealed/inherited colonies have `"phase": "inherited"` (a string, not a number). A naive `phase < current_phase` numeric comparison in jq will fail or produce unexpected results.
**Why it happens:** When a colony inherits learnings from a completed colony, the phase is set to `"inherited"` rather than a number.
**How to avoid:** In the jq filter, handle both numeric and string phase values: `select((.phase | type) == "string" or .phase < $current_phase)`. Alternatively, treat "inherited" as phase 0 (always included since current_phase is always >= 1 during a build).
**Warning signs:** If inherited learnings never appear in builder prompts, the type check is failing silently.

### Pitfall 2: Empty Learnings Produce Empty Section
**What goes wrong:** If no validated learnings exist from previous phases, an empty "--- PHASE LEARNINGS ---" header appears in the builder prompt, wasting token budget.
**Why it happens:** The section is rendered unconditionally.
**How to avoid:** Check the extracted learnings count before appending the section. Only add the "--- PHASE LEARNINGS ---" block if at least one validated claim was found. This matches the pattern in colony-prime where QUEEN wisdom is conditionally rendered (lines 7603-7622).
**Warning signs:** Builder prompts show "--- PHASE LEARNINGS ---" followed immediately by "--- END ---" with no content between them.

### Pitfall 3: Log Line Integration with pheromone-prime
**What goes wrong:** The `log_line` in colony-prime currently comes from pheromone-prime's output (`cp_log_line` at line 7586). Adding learnings count requires modifying this line AFTER pheromone-prime returns.
**Why it happens:** colony-prime delegates signal/instinct counting to pheromone-prime and uses its log_line directly.
**How to avoid:** After extracting learnings, append to the existing log_line: `cp_log_line="$cp_log_line, $cp_learning_count learnings"`. Do not modify pheromone-prime's log output.
**Warning signs:** If the build output shows "Primed: 3 signals, 2 instincts" without a learnings count, the log_line was not updated.

### Pitfall 4: Phase 1 Build Has No Previous Learnings
**What goes wrong:** When building phase 1 (current_phase == 1, or 0 if zero-indexed), there are no previous phases to draw learnings from. The extraction should produce zero results gracefully.
**Why it happens:** The first phase has no predecessors.
**How to avoid:** The jq filter `select(.phase < 1)` will correctly return nothing when there are no phase-0 or inherited learnings. Ensure the code handles empty results without errors. Test this case explicitly.
**Warning signs:** Error output or "null" in prompt_section when building the first phase.

### Pitfall 5: Learnings Cap Overflow
**What goes wrong:** `memory.phase_learnings` is capped at 20 entries (continue-advance.md line 156), but each entry can have multiple claims in its `learnings` array. If a colony with many phases accumulates hundreds of validated claims, the prompt_section becomes too large.
**Why it happens:** The cap is on entries (grouped by phase), not individual claims.
**How to avoid:** Apply a secondary cap on total claims injected (e.g., max 15 validated claims). Take the most recent claims first (by timestamp), dropping older ones when over budget. This mirrors the `--max-instincts` pattern in pheromone-prime.
**Warning signs:** Builder prompt_section is unusually long, or token limits are hit during builds with many completed phases.

### Pitfall 6: Compact vs Non-Compact Mode
**What goes wrong:** colony-prime supports `--compact` mode (line 7457-7460) which is used by build-context.md. The learnings section must respect this flag.
**Why it happens:** Compact mode is designed for token-constrained contexts. It passes `--max-signals 8 --max-instincts 3` to pheromone-prime.
**How to avoid:** In compact mode, limit the number of injected learnings (e.g., max 5 claims). In non-compact mode, allow more (e.g., max 15). Check the `cp_compact` variable.
**Warning signs:** Compact mode produces the same learnings output as non-compact, bloating the prompt.

## Code Examples

### Phase Learnings Data Structure (from archived colony state)
```json
{
  "memory": {
    "phase_learnings": [
      {
        "id": "learning_inherited_1",
        "phase": "inherited",
        "learnings": [
          {
            "claim": "Claude Code global sync works by copying commands from .claude/commands/ to ~/.claude/commands/",
            "status": "validated",
            "tested": true,
            "evidence": "Validated in prior colony session"
          }
        ],
        "source": "inherited:completion-report",
        "timestamp": "2026-02-13T20:40:00Z"
      },
      {
        "id": "learning_1709736000",
        "phase": 1,
        "phase_name": "instinct-pipeline",
        "learnings": [
          {
            "claim": "Confidence floor >= 0.7 prevents noise instincts",
            "status": "validated",
            "tested": true,
            "evidence": "Lower thresholds produced spurious instincts"
          },
          {
            "claim": "Untested theory about X",
            "status": "hypothesis",
            "tested": false,
            "evidence": "Initial observation only"
          }
        ],
        "timestamp": "2026-03-06T12:00:00Z"
      }
    ]
  }
}
```

### jq Extraction Pattern (validated claims from previous phases)
```bash
# Extract validated claims from phases before current_phase
# Handles both numeric phases and "inherited" string phase
cp_learnings=$(jq -r \
  --argjson current "$cp_current_phase" \
  --argjson max_claims "$cp_max_learnings" \
  '
  (.memory.phase_learnings // [])
  | map(select(
      (.phase | type) == "string"
      or (.phase | tonumber) < $current
    ))
  | map({
      phase: .phase,
      phase_name: (.phase_name // ""),
      claims: [
        .learnings[]
        | select(.status == "validated")
        | .claim
      ]
    })
  | map(select(.claims | length > 0))
  | . as $groups
  | [foreach $groups[] as $g (0; . + ($g.claims | length); .)] as $running
  | [range($groups | length)]
  | map(
      $groups[.] | .claims = .claims[:([($max_claims - ($running[.] - (.claims | length))), 0] | max)]
    )
  | map(select(.claims | length > 0))
  ' "$cp_state_file" 2>/dev/null || echo "[]")
```

Note: The above jq is complex for cap enforcement. A simpler approach (recommended) is to flatten all validated claims, sort by timestamp, take top N, then re-group for display.

### Simpler jq Extraction (recommended)
```bash
# Simpler approach: extract all validated claims, cap, then format
cp_learning_claims=$(jq -r \
  --argjson current "$cp_current_phase" \
  --argjson max "$cp_max_learnings" \
  '
  [
    (.memory.phase_learnings // [])[]
    | select((.phase | type) == "string" or (.phase | tonumber) < $current)
    | .phase as $p | .phase_name as $pn |
    .learnings[]
    | select(.status == "validated")
    | {phase: $p, phase_name: $pn, claim: .claim}
  ]
  | .[:$max]
  ' "$cp_state_file" 2>/dev/null || echo "[]")
```

### Formatting Pattern (actionable guidance, not raw JSON)
```bash
# Format validated claims as actionable guidance grouped by phase
# Target output:
#
# --- PHASE LEARNINGS (Previous Phase Insights) ---
#
# Phase 1 (instinct-pipeline):
#   - Confidence floor >= 0.7 prevents noise instincts
#   - Error patterns should get higher confidence than success patterns
#
# Inherited:
#   - Claude Code global sync works by copying commands to ~/.claude/commands/
#
# --- END PHASE LEARNINGS ---

cp_learning_section=""
cp_learning_count=$(echo "$cp_learning_claims" | jq 'length' 2>/dev/null || echo "0")

if [[ "$cp_learning_count" -gt 0 ]]; then
  cp_learning_section="--- PHASE LEARNINGS (Previous Phase Insights) ---"$'\n'

  cp_learning_lines=$(echo "$cp_learning_claims" | jq -r '
    group_by(.phase)
    | map({
        phase: .[0].phase,
        phase_name: .[0].phase_name,
        claims: [.[].claim]
      })
    | sort_by(if .phase == "inherited" then -1 else .phase end)
    | .[]
    | "\n"
      + (if .phase == "inherited" then "Inherited"
         elif .phase_name != "" then "Phase " + (.phase | tostring) + " (" + .phase_name + ")"
         else "Phase " + (.phase | tostring)
         end)
      + ":"
      + "\n" + (.claims | map("  - " + .) | join("\n"))
  ' 2>/dev/null || echo "")

  if [[ -n "$cp_learning_lines" ]]; then
    cp_learning_section+="$cp_learning_lines"$'\n'
  fi

  cp_learning_section+=$'\n'"--- END PHASE LEARNINGS ---"
fi
```

### Colony-Prime Insertion Point (line references from current source)
```bash
# In colony-prime, after context-capsule block (line 7631) and before pheromone signals (line 7633):

# Add compact context capsule for low-token continuity
cp_capsule_prompt=""
cp_capsule_raw=$("$SCRIPT_DIR/aether-utils.sh" context-capsule --compact --json 2>/dev/null) || cp_capsule_raw=""
cp_capsule_prompt=$(echo "$cp_capsule_raw" | jq -r '.result.prompt_section // ""' 2>/dev/null || echo "")
if [[ -n "$cp_capsule_prompt" ]]; then
  cp_final_prompt+=$'\n'"$cp_capsule_prompt"$'\n'
fi

# === NEW: Phase learnings injection ===
cp_current_phase=$(jq -r '.current_phase // 0' "$cp_state_file" 2>/dev/null || echo "0")
cp_max_learnings=15
if [[ "$cp_compact" == "true" ]]; then
  cp_max_learnings=5
fi
# ... extraction and formatting code from above ...
if [[ -n "$cp_learning_section" ]]; then
  cp_final_prompt+=$'\n'"$cp_learning_section"$'\n'
fi
# === END phase learnings injection ===

# Add pheromone signals section
if [[ -n "$cp_prompt_section" && "$cp_prompt_section" != "null" ]]; then
  cp_final_prompt+=$'\n'"$cp_prompt_section"
fi
```

### Test Pattern (from instinct-pipeline.test.js)
```javascript
test.serial('colony-prime includes validated learnings from previous phases', async (t) => {
  const tmpDir = await createTempDir();
  try {
    // Setup colony at phase 3 with learnings from phases 1 and 2
    await setupTestColony(tmpDir, {
      currentPhase: 3,
      phaseLearnings: [
        {
          id: 'learning_1',
          phase: 1,
          phase_name: 'foundation',
          learnings: [
            { claim: 'Always validate JSON before parsing', status: 'validated', tested: true, evidence: 'Parse errors crashed worker' },
            { claim: 'Untested theory', status: 'hypothesis', tested: false, evidence: 'Just a guess' }
          ],
          timestamp: '2026-03-06T12:00:00Z'
        },
        {
          id: 'learning_2',
          phase: 2,
          phase_name: 'wiring',
          learnings: [
            { claim: 'Use --compact flag in tests', status: 'validated', tested: true, evidence: 'Reduces token usage' }
          ],
          timestamp: '2026-03-06T13:00:00Z'
        }
      ]
    });

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const json = JSON.parse(result);
    t.true(json.ok);

    const prompt = json.result.prompt_section;
    // Validated claims should appear
    t.true(prompt.includes('Always validate JSON before parsing'));
    t.true(prompt.includes('Use --compact flag in tests'));
    // Hypothesis claims should NOT appear
    t.false(prompt.includes('Untested theory'));
    // Section header should exist
    t.true(prompt.includes('PHASE LEARNINGS'));
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
```

## State of the Art

| Current State | Target State | What Changes | Impact |
|---------------|-------------|--------------|--------|
| colony-prime has NO phase learnings extraction | colony-prime reads memory.phase_learnings and injects validated claims | New extraction + formatting block in colony-prime | Builders see what was learned in previous phases |
| prompt_section contains: wisdom + capsule + signals/instincts | prompt_section contains: wisdom + capsule + LEARNINGS + signals/instincts | New section between capsule and signals | Richer builder context |
| log_line reports "N signals, M instincts" | log_line reports "N signals, M instincts, L learnings" | Append learning count to existing log_line | User sees learning count in build output |
| Phase learnings written by continue but never read | Full write-read cycle: continue writes, colony-prime reads | Closes the data flow gap | Colony learning becomes visible |

## Open Questions

1. **Ordering of learnings in the prompt section**
   - What we know: Learnings should be grouped by phase. Phase 1 established domain-grouping for instincts.
   - What's unclear: Should learnings be ordered by phase number (ascending, oldest first) or reverse chronological (newest first)?
   - Recommendation: Ascending phase order (oldest first). This gives builders the full progression of colony learning, and the most recent learnings (most relevant) appear last and closest to the signals/instincts section. Also, "inherited" learnings should appear first since they represent the most foundational knowledge.

2. **Interaction with existing context-capsule**
   - What we know: context-capsule already reads `memory.decisions` from COLONY_STATE.json and includes them in its compact output. It does NOT read phase_learnings.
   - What's unclear: Should learnings also be added to context-capsule, or only to colony-prime?
   - Recommendation: Only add to colony-prime. Context-capsule is designed for minimal continuity context (state/phase/next-action/decisions/risks). Phase learnings are richer context that belongs in the full priming payload. Adding to both would create duplication.

3. **Maximum claims cap in compact mode**
   - What we know: Compact mode limits instincts to 3 (via pheromone-prime --max-instincts 3) and signals to 8.
   - What's unclear: What's the right cap for learnings in compact mode?
   - Recommendation: 5 claims in compact mode, 15 in non-compact. This keeps the learnings section proportional to the instincts section (which has 3-5 items). Adjustable via a variable at the top of the extraction block.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 7451-7671: colony-prime subcommand -- verified assembly order, identified insertion point
- `.aether/aether-utils.sh` lines 7273-7449: pheromone-prime subcommand -- verified instinct extraction pattern (Phase 1 reference)
- `.aether/docs/command-playbooks/continue-advance.md` lines 14-40: phase_learnings write structure (id, phase, phase_name, learnings[{claim,status,tested,evidence}])
- `.aether/docs/command-playbooks/build-context.md` lines 1-34: confirmed colony-prime call and prompt_section injection (no changes needed)
- `.aether/docs/command-playbooks/build-wave.md` line 319: confirmed `{ prompt_section }` placeholder in builder prompts (no changes needed)
- `.aether/chambers/v1-1-bug-fixes-update-system-repair-20260215-064940/colony-state.json` lines 32-104: real-world phase_learnings data with "inherited" phase entries
- `tests/integration/instinct-pipeline.test.js`: test pattern with setupTestColony, runAetherUtil, createTempDir helpers
- `.planning/phases/01-instinct-pipeline/01-02-SUMMARY.md`: confirmed no changes needed to build-context.md or build-wave.md

### Secondary (MEDIUM confidence)
- `.aether/templates/colony-state.template.json`: COLONY_STATE.json structure showing memory.phase_learnings as array placeholder
- `.aether/docs/command-playbooks/continue-advance.md` line 156: cap enforcement -- max 20 phase_learnings

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all code paths traced through source, insertion point identified at specific line numbers
- Architecture: HIGH - file touch map derived from actual colony-prime assembly chain; Phase 1 confirmed that build-context.md and build-wave.md need no changes
- Pitfalls: HIGH - inherited phase type issue verified in real archived colony state data; empty section, cap overflow, and compact mode pitfalls derived from existing code patterns

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable codebase, internal tooling)
