# Phase 43: Make Learning Flow - Research

**Researched:** 2026-02-22
**Domain:** Aether Learning Pipeline Integration
**Confidence:** HIGH

## Summary

This phase wires the existing learning pipeline so observations automatically flow to QUEEN.md promotions. The components all exist and work individually — the task is connecting them properly.

The learning pipeline has four main functions in `.aether/aether-utils.sh`:
- `learning-observe` (lines 4407-4551) — Records observations with content hash deduplication
- `learning-check-promotion` (lines 4553-4605) — Finds observations meeting thresholds
- `learning-approve-proposals` (lines 5044-5305) — Interactive approval workflow with tick-to-approve UI
- `queen-promote` (lines 4102-4300) — Writes promoted wisdom to QUEEN.md

The pipeline is called from:
- `/ant:continue` — Records learnings after phase advancement, checks for proposals
- `/ant:build` — Records failure observations when builders fail
- `/ant:seal` and `/ant:entomb` — Final promotion checks before archiving

**Primary recommendation:** Ensure `learning-observations.json` is created during `/ant:init`, verify the full pipeline works end-to-end, and test with real learning data.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Proposal Display Mode:** Present proposals one at a time (not batch)
- **Minimal format:** observation text + approve button
- **Three actions available:** Approve / Reject / Skip
- **After user acts:** Auto-show the next pending proposal (no manual continue)
- **Threshold checking:** At end of build (not after each observation)
- **Observation accumulation:** Cumulatively across sessions (forever)
- **Failure behavior:** If QUEEN.md write fails, prompt user to retry; if declined, skip to next proposal, keep failed one pending

### Claude's Discretion
- Specific threshold values (category-based or uniform) — recommend starting with 3
- What happens to observations after promotion (archive, clear, or mark) — recommend archive + reset
- Exact retry prompt wording
- Error message format

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| FLOW-01 | Auto-create learning-observations.json if missing during /ant:init | Template exists at `.aether/templates/learning-observations.template.json`. Init already creates pheromones.json and midden.json using same pattern (lines 290-308 in init.md) |
| FLOW-02 | Verify observations → proposals → promotions → QUEEN.md pipeline | All functions exist and work. Pipeline: `learning-observe` records → `learning-check-promotion` finds threshold-meeting → `learning-approve-proposals` presents UI → `queen-promote` writes to QUEEN.md |
| FLOW-03 | Test end-to-end with real learning | Build.md records observations on failures (lines 765-768). Continue.md checks promotions (lines 1219-1260). Need to verify integration points work together |
</phase_requirements>

## Standard Stack

### Core (Already Exists)
| Component | Location | Purpose |
|-----------|----------|---------|
| `learning-observe` | aether-utils.sh:4407-4551 | Record observation with SHA256 hash deduplication |
| `learning-check-promotion` | aether-utils.sh:4553-4605 | Return proposals meeting thresholds |
| `learning-approve-proposals` | aether-utils.sh:5044-5305 | Interactive tick-to-approve UI |
| `queen-promote` | aether-utils.sh:4102-4300 | Write promoted wisdom to QUEEN.md |
| `colony-prime` | aether-utils.sh:6452-6654 | Load QUEEN.md wisdom for worker context |

### Data Files
| File | Location | Purpose |
|------|----------|---------|
| `learning-observations.json` | `.aether/data/` | Accumulates all observations |
| `learning-deferred.json` | `.aether/data/` | Stores deferred proposals for later review |
| `.promotion-undo.json` | `.aether/data/` | Enables undo within 24h window |
| `QUEEN.md` | `.aether/` | Destination for promoted wisdom |

### Template
| Template | Location | Target |
|----------|----------|--------|
| `learning-observations.template.json` | `.aether/templates/` | `.aether/data/learning-observations.json` |

## Architecture Patterns

### Pattern 1: Observation Recording
**What:** Record observations during builds with content hash deduplication
**When to use:** When builders fail, chaos finds issues, or watcher verification fails
**Example from build.md:**
```bash
# Record observation for potential promotion
bash .aether/aether-utils.sh learning-observe \
  "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" \
  "failure" \
  "${colony_name}" 2>/dev/null || true
```

### Pattern 2: Threshold Checking
**What:** Check if observations meet type-specific thresholds
**When to use:** At end of build or during continue
**Current thresholds (from aether-utils.sh:4515-4523):**
```bash
philosophy) threshold=1 ;;  # Was 5
pattern) threshold=1 ;;      # Was 3
redirect) threshold=1 ;;     # Was 2
stack) threshold=1 ;;        # Unchanged
decree) threshold=0 ;;       # Unchanged (immediate)
failure) threshold=1 ;;      # NEW
```

### Pattern 3: Tick-to-Approve UI
**What:** Present proposals one at a time with minimal format
**When to use:** When proposals exist at end of build/continue
**Flow from learning-approve-proposals:**
1. Display proposal with observation text
2. User selects: Approve / Reject / Skip
3. Auto-show next proposal
4. After all: offer undo option (24h window)

### Pattern 4: Promotion to QUEEN.md
**What:** Write approved wisdom to QUEEN.md with proper formatting
**When to use:** After user approves proposal
**Entry format:**
```markdown
- **${colony_name}** (${timestamp}): ${content}
```

### Anti-Patterns to Avoid
- **Calling learning-approve-proposals mid-build:** Per user decision, wait until end
- **Batch display of proposals:** Present one at a time per locked decision
- **Clearing observations after promotion:** Archive + reset recommended, not delete

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Observation deduplication | Custom hash tracking | `learning-observe` | SHA256 hashing with collision handling |
| Threshold checking | Manual count comparison | `learning-check-promotion` | Type-specific thresholds built-in |
| Interactive approval UI | Custom prompts | `learning-approve-proposals` | Handles selection parsing, deferral, undo |
| QUEEN.md writing | Direct file append | `queen-promote` | Atomic writes, section management, metadata updates |
| Wisdom loading | Manual file reading | `colony-prime` | Combines global + local QUEEN.md, extracts all sections |

**Key insight:** All the complex pieces exist and work. The task is integration, not implementation.

## Common Pitfalls

### Pitfall 1: Missing learning-observations.json
**What goes wrong:** If file doesn't exist, `learning-check-promotion` returns empty proposals silently
**Why it happens:** File is only created when first observation is recorded
**How to avoid:** Create file during `/ant:init` from template (FLOW-01 requirement)
**Warning signs:** "No proposals to review" even after observations should exist

### Pitfall 2: Threshold Mismatch
**What goes wrong:** `learning-observe` uses one set of thresholds, `learning-check-promotion` uses another
**Why it happens:** Code drift — observe has thresholds at lines 4515-4523, check-promotion has different values at lines 4578-4586
**How to avoid:** Verify thresholds match between functions (currently they don't — observe uses 1 for most, check-promotion uses old values like 5 for philosophy)
**Warning signs:** Observation shows threshold_met=true but check-promotion doesn't return it as proposal

### Pitfall 3: Colony Name Resolution Failure
**What goes wrong:** `learning-approve-proposals` extracts colony name from COLONY_STATE.json without checking file exists
**Why it happens:** Line 5075-5077 assumes file exists
**How to avoid:** Add file existence check before jq
**Warning signs:** jq errors in logs, "unknown" colony name in promotions

### Pitfall 4: QUEEN.md Write Failure Handling
**What goes wrong:** If `queen-promote` fails, the pipeline may stop or lose the proposal
**Why it happens:** Error handling is basic — just logs and continues
**How to avoid:** Implement retry logic per user decision (FLOW-02 requirement)
**Warning signs:** Failed promotions disappear instead of staying pending

## Code Examples

### Creating learning-observations.json during init
```bash
# From init.md pattern (lines 290-308)
for template in pheromones midden learning-observations; do
  if [[ "$template" == "midden" ]]; then
    target=".aether/data/midden/midden.json"
  else
    target=".aether/data/${template}.json"
  fi
  if [[ ! -f "$target" ]]; then
    template_file=""
    for path in ~/.aether/system/templates/${template}.template.json .aether/templates/${template}.template.json; do
      if [[ -f "$path" ]]; then
        template_file="$path"
        break
      fi
    done
    if [[ -n "$template_file" ]]; then
      jq 'with_entries(select(.key | startswith("_") | not))' "$template_file" > "$target" 2>/dev/null || true
    fi
  fi
done
```

### Recording an observation
```bash
# From build.md (lines 765-768)
bash .aether/aether-utils.sh learning-observe \
  "Builder ${ant_name} failed on task ${task_id}: ${reason}" \
  "failure" \
  "${colony_name}" 2>/dev/null || true
```

### Checking for proposals
```bash
# From continue.md (lines 1236-1252)
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  bash .aether/aether-utils.sh learning-approve-proposals $verbose_flag
fi
```

### Promoting to QUEEN.md
```bash
# From learning-approve-proposals (line 5229)
promote_result=$(bash "$0" queen-promote "$ptype" "$content" "$colony_name" 2>&1) || {
  echo "Failed to promote: $content"
  # Handle failure per user decision: prompt retry, skip, keep pending
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Threshold 5 for philosophy | Threshold 1 | 2026-02 (recent) | Faster promotion cycle |
| Manual QUEEN.md edits | queen-promote only | v1.0 | Consistent formatting |
| No observation tracking | learning-observe | v1.0+ | Pattern detection across colonies |
| Batch proposal display | One at a time | User decision (CONTEXT.md) | Better UX per user preference |

**Deprecated/outdated:**
- Old threshold values in `learning-select-proposals` (lines 4803-4809) still use 5/3/2/1/0, inconsistent with `learning-observe`

## Open Questions

1. **Threshold Consistency**
   - What we know: `learning-observe` uses threshold=1 for most types
   - What's unclear: `learning-check-promotion` and `learning-select-proposals` use different values
   - Recommendation: Align all functions to use same thresholds (recommend uniform threshold=3 per user discretion)

2. **Post-Promotion Observation Handling**
   - What we know: User deferred this to Claude's discretion
   - What's unclear: Whether to archive, clear, or mark observations
   - Recommendation: Archive to `learning-archived.json` + reset counts, keeping history without re-promoting

3. **Retry Prompt Wording**
   - What we know: User wants retry on QUEEN.md write failure
   - What's unclear: Exact wording and retry count
   - Recommendation: "Write to QUEEN.md failed. Retry? (y/n):" with 3 retry limit

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh:4407-4551` — learning-observe implementation
- `.aether/aether-utils.sh:4553-4605` — learning-check-promotion implementation
- `.aether/aether-utils.sh:5044-5305` — learning-approve-proposals implementation
- `.aether/aether-utils.sh:4102-4300` — queen-promote implementation
- `.aether/aether-utils.sh:6452-6654` — colony-prime implementation
- `.claude/commands/ant/continue.md:1219-1260` — Proposal checking in continue
- `.claude/commands/ant/build.md:765-768` — Observation recording on failure
- `.planning/research/CODEBASE-FLOW.md` — Gap analysis with line numbers

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/init.md:290-308` — Template creation pattern
- `.aether/templates/learning-observations.template.json` — Empty structure

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All functions exist and documented
- Architecture: HIGH — Clear pipeline flow from CODEBASE-FLOW.md
- Pitfalls: HIGH — Direct code analysis with line numbers

**Research date:** 2026-02-22
**Valid until:** 30 days (stable system)

---

*Research complete. Ready for planning.*
