# Phase 4: Pheromone Auto-Emission - Research

**Researched:** 2026-03-06
**Domain:** Bash shell scripting (aether-utils.sh), continue-advance playbook wiring, pheromone-write API, midden failure tracking, memory-capture pipeline
**Confidence:** HIGH

## Summary

Phase 4 wires three automatic pheromone emission sources into the continue-advance.md playbook: (1) decisions recorded during a phase auto-emit FEEDBACK pheromones, (2) recurring error patterns in the midden (3+ occurrences) auto-emit REDIRECT pheromones, and (3) success criteria patterns that recur across phases auto-emit FEEDBACK pheromones. All three sources use the existing `pheromone-write` subcommand (line 6680) and the existing `memory-capture` pipeline (line 5311) -- no new subcommands need to be created.

Critically, much of this wiring already partially exists. The continue-advance.md playbook already has a Step 2.1 "Auto-Emit Phase Pheromones" section that: (a) emits a generic FEEDBACK for phase outcome (Step 2.1a), (b) checks `errors.flagged_patterns[]` for patterns with count >= 2 and emits REDIRECT (Step 2.1b), and (c) expires phase_end signals (Step 2.1c). Additionally, `context-update decision` (line 507-512) already auto-emits a FEEDBACK pheromone when decisions are recorded. The `memory-capture` subcommand (line 5311) already auto-emits pheromones for every event type (learning, failure, success, resolution, etc.) via `pheromone-write`.

However, the requirements ask for something more specific than what exists: (PHER-01) decisions recorded during the current phase should create FEEDBACK pheromones with the decision content during continue (not just when context-update is called); (PHER-02) the threshold should be 3+ midden occurrences (current Step 2.1b checks `errors.flagged_patterns` with count >= 2, which is a different data source from `midden.json`); (PHER-03) success criteria patterns that recur across phases should be detected and emit FEEDBACK. Additionally, all auto-emitted pheromones must be marked with their source type (decision/error/success) so users can distinguish them from manual pheromones.

**Primary recommendation:** Modify continue-advance.md to add three new emission blocks in Step 2.1: (1) a decision-to-FEEDBACK block that reads `memory.decisions` from COLONY_STATE.json (or extracts from CONTEXT.md Recent Decisions table) and emits FEEDBACK with source `"auto:decision"`; (2) a midden-error-to-REDIRECT block that queries `midden-recent-failures`, groups by category, and emits REDIRECT for categories with 3+ occurrences with source `"auto:error"`; (3) a success-criteria-to-FEEDBACK block that compares current phase success criteria against previous phases and emits FEEDBACK for recurring patterns with source `"auto:success"`. No changes needed to colony-prime or build playbooks -- auto-emitted pheromones flow through the existing pheromone-prime pipeline.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| PHER-01 | Key decisions recorded during continue auto-emit FEEDBACK pheromones | `context-update decision` (line 507-512) already emits FEEDBACK with source `"system:decision"` when decisions are recorded. continue-advance.md Step 2.6 calls `context-update decision` for architectural decisions. The gap: decisions in `memory.decisions` or CONTEXT.md accumulated during the phase are not explicitly batch-emitted as pheromones during continue. Need to add a batch emission block in Step 2.1 that reads recent decisions and emits FEEDBACK for each with source `"auto:decision"` |
| PHER-02 | Recurring error patterns (3+ occurrences) auto-emit REDIRECT pheromones | continue-advance.md Step 2.1b already checks `errors.flagged_patterns[]` for count >= 2. The gap: this reads from `errors.flagged_patterns` in COLONY_STATE.json (which may be empty), not from `midden.json` (the actual failure store). Need to read `midden-recent-failures`, group by category/message similarity, detect 3+ occurrences, and emit REDIRECT with source `"auto:error"` and the error pattern text |
| PHER-03 | Success criteria patterns auto-emit FEEDBACK on recurrence across phases | Success criteria are stored per-task in `plan.phases[].tasks[].success_criteria` arrays (confirmed at line 3332). The gap: no code currently compares success criteria across completed phases to detect recurring patterns. Need to extract success criteria from all completed phases, find text similarities, and emit FEEDBACK for recurring criteria with source `"auto:success"` |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | `pheromone-write`, `midden-recent-failures`, `memory-capture` subcommands | Single source of truth for all state operations |
| jq | System-installed | JSON manipulation for COLONY_STATE.json, midden.json, pheromones.json | Used throughout aether-utils.sh for all state reads |
| ava | Installed in package.json | Integration test runner | Project standard, Phases 1-3 tests all use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| continue-advance.md | Playbook | Step 2.1 defines auto-emission blocks | Modify to add three new emission sub-steps |
| pheromone-write | aether-utils.sh subcommand (line 6680) | Creates pheromone signals in pheromones.json | Called for each auto-emitted pheromone |
| midden-recent-failures | aether-utils.sh subcommand (line 9488) | Reads recent failures from midden.json | Query for PHER-02 error pattern detection |
| memory-capture | aether-utils.sh subcommand (line 5311) | Records observations + auto-emits pheromone + auto-promotes | Already used in Step 2.5; could be leveraged for emission |
| pheromone-prime | aether-utils.sh subcommand (line 7273) | Assembles prompt_section with all active signals | Already reads all pheromones including auto-emitted ones |
| colony-prime | aether-utils.sh subcommand (line 7451) | Unified priming (wisdom + signals + instincts + learnings + decisions + blockers) | Already calls pheromone-prime; NO changes needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Direct pheromone-write calls in playbook | memory-capture pipeline for each emission | memory-capture adds observation tracking + auto-promotion on top of pheromone-write. For PHER-01 decisions, the extra tracking is valuable (decisions may recur across phases). For PHER-02 errors, memory-capture with event_type "failure" already emits REDIRECT. Decision: use memory-capture for PHER-01/PHER-03, use direct pheromone-write for PHER-02 (midden already tracks recurrence) |
| Reading decisions from CONTEXT.md markdown table | Reading from COLONY_STATE.json `memory.decisions` | Phase 3 research confirmed `memory.decisions` is always empty -- decisions are stored in CONTEXT.md table by `context-update decision`. However, continue-advance.md Step 2 records phase decisions in `memory.decisions` during state update. Need to use whichever source has data. Recommend: check `memory.decisions` first (most recent phase decisions), fall back to CONTEXT.md table |
| Creating a new `pheromone-auto-emit` subcommand | Inline bash in continue-advance.md playbook | Phase 1-3 proved inline playbook modifications work well. However, this phase has 3 emission sources with some complexity. A utility subcommand could be testable independently. Decision: use inline playbook approach for consistency, but ensure each block is independently testable via existing subcommands |
| Modifying pheromone-prime to show source metadata | Leaving source in pheromone data, only visible via /ant:pheromones | Success criteria #4 requires auto-emitted pheromones to be marked with source so users can distinguish them. The `source` field in pheromone-write already supports arbitrary strings. pheromone-display (line 6889) already shows signal details. pheromone-prime only shows `[strength] content` format. Decision: auto-emitted pheromones use descriptive source prefixes (`auto:decision`, `auto:error`, `auto:success`) and content includes the source type label |

## Architecture Patterns

### File Touch Map

```
MODIFY:
  .aether/docs/command-playbooks/continue-advance.md    # Step 2.1: add decision, midden, success criteria emission blocks
  .aether/docs/command-playbooks/continue-full.md        # Mirror changes from continue-advance.md

CREATE:
  tests/integration/pheromone-auto-emission.test.js      # End-to-end auto-emission tests

NO CHANGE NEEDED:
  .aether/aether-utils.sh                                # All needed subcommands already exist
  .aether/docs/command-playbooks/build-context.md         # Already calls colony-prime, injects prompt_section
  .aether/docs/command-playbooks/build-wave.md            # Already injects { prompt_section } into builder prompts
  .aether/docs/command-playbooks/continue-finalize.md     # No emission logic needed here
  .aether/docs/command-playbooks/continue-gates.md        # No emission logic needed here
  .aether/docs/command-playbooks/continue-verify.md       # No emission logic needed here
```

### Pattern 1: Auto-Emission Placement in Continue-Advance

**What:** The continue-advance.md playbook Step 2.1 "Auto-Emit Phase Pheromones" already contains emission logic. The three new PHER requirements extend this step with additional emission sub-blocks.

**Current Step 2.1 structure:**
```
Step 2.1: Auto-Emit Phase Pheromones (SILENT)
  2.1a: Auto-emit FEEDBACK for phase outcome           <-- already exists
  2.1b: Auto-emit REDIRECT for errors.flagged_patterns  <-- needs replacement (PHER-02)
  2.1c: Expire phase_end signals + archive              <-- already exists, keep

Step 2.1.5: Check for Promotion Proposals               <-- already exists, keep
```

**Proposed Step 2.1 structure:**
```
Step 2.1: Auto-Emit Phase Pheromones (SILENT)
  2.1a: Auto-emit FEEDBACK for phase outcome            <-- keep as-is
  2.1b: Auto-emit FEEDBACK for phase decisions (PHER-01) <-- NEW
  2.1c: Auto-emit REDIRECT for midden error patterns (PHER-02) <-- REPLACE existing 2.1b
  2.1d: Auto-emit FEEDBACK for recurring success criteria (PHER-03) <-- NEW
  2.1e: Expire phase_end signals + archive              <-- renumbered from 2.1c

Step 2.1.5: Check for Promotion Proposals               <-- keep as-is
```

**Why this position:** All auto-emission runs after learnings extraction (Step 2) and before state advancement (Step 4). This ensures decisions, errors, and success patterns from the completed phase are available for emission. The SILENT contract means failures never block phase advancement.

### Pattern 2: Source Marking Convention

**What:** The `source` field in pheromone-write already accepts arbitrary strings. Existing conventions:
- `"user"` -- manual `/ant:focus`, `/ant:redirect`, `/ant:feedback`
- `"worker:continue"` -- auto-emitted during continue (Step 2.1a)
- `"worker:builder"` -- auto-emitted during builds
- `"system"` -- auto-emitted for error patterns (Step 2.1b)
- `"system:decision"` -- auto-emitted by `context-update decision`
- `"system:suggestion"` -- auto-emitted by `suggest-approve`
- `"user:insert-phase"` -- auto-emitted when a phase is inserted

**Proposed new source values:**
- `"auto:decision"` -- PHER-01: decision-sourced FEEDBACK
- `"auto:error"` -- PHER-02: midden error-sourced REDIRECT
- `"auto:success"` -- PHER-03: success criteria-sourced FEEDBACK

**Why `auto:` prefix:** Distinguishes auto-emitted pheromones from user-emitted (`user`), worker-emitted (`worker:`), and system-emitted (`system:`) signals. The `auto:` prefix makes it easy to filter in `pheromone-display` and `pheromone-read`.

### Pattern 3: Decision Data Flow (PHER-01)

**What:** Decisions are recorded in two places: (1) CONTEXT.md "Recent Decisions" markdown table via `context-update decision`, which already auto-emits a FEEDBACK; (2) `memory.decisions` array in COLONY_STATE.json, which is always `[]` per Phase 3 research. However, continue-advance.md Step 2 has a cap enforcement step that mentions "max 30 decisions" -- meaning decisions ARE expected in `memory.decisions` but are populated by the AI during continue, not by a subcommand.

**Extraction approach:** During continue-advance Step 2.1b (new), extract decisions from CONTEXT.md "Recent Decisions" table using the same awk extraction that colony-prime CTX-01 uses (lines 7702-7721). For each decision not already covered by an existing auto:decision pheromone, emit a FEEDBACK with source `"auto:decision"`.

**Deduplication:** Before emitting, check existing pheromones.json for active signals with source `"auto:decision"` or `"system:decision"` containing the same decision text. Skip if already covered. This prevents duplicate emissions when `context-update decision` already emitted one.

### Pattern 4: Midden Error Pattern Detection (PHER-02)

**What:** The midden stores failure entries in `.aether/data/midden/midden.json` with shape `{id, timestamp, category, source, message, reviewed}`. The `midden-recent-failures` subcommand (line 9488) returns them sorted by timestamp.

**Detection approach:** Query `midden-recent-failures` with a high limit (e.g., 50), group entries by `category`, count occurrences per category. For categories with 3+ occurrences (matching the requirement threshold), emit a REDIRECT pheromone with the category name and a summary of the recurring pattern.

**Important difference from existing Step 2.1b:** The current Step 2.1b reads `errors.flagged_patterns[]` from COLONY_STATE.json, which is a different data source from midden.json. The midden is the actual failure store with individual entries; `errors.flagged_patterns` is a higher-level pattern tracker that may not be populated. The new implementation should read from `midden.json` directly.

**jq grouping approach:**
```bash
midden_result=$(bash .aether/aether-utils.sh midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
# Group by category, find categories with 3+ occurrences
recurring=$(echo "$midden_result" | jq -r '[.failures[] | .category] | group_by(.) | map(select(length >= 3)) | map({category: .[0], count: length}) | .[]')
```

### Pattern 5: Success Criteria Recurrence Detection (PHER-03)

**What:** Success criteria are stored in `plan.phases[].tasks[].success_criteria` as string arrays in COLONY_STATE.json (confirmed at line 3332). Phase-level success criteria are in `plan.phases[].success_criteria`.

**Detection approach:** Extract success criteria text from all completed phases, normalize (lowercase, trim), and find criteria that appear in 2+ completed phases. For each recurring criterion, emit a FEEDBACK pheromone noting the pattern.

**jq extraction approach:**
```bash
# Extract all success criteria from completed phases
criteria=$(jq -r '
  [.plan.phases[]
   | select(.status == "completed")
   | .id as $phase_id
   | (
       (.success_criteria // [])[] ,
       (.tasks // [] | .[].success_criteria // [])[]
     )
   | {phase: $phase_id, text: (. | ascii_downcase | gsub("^\\s+|\\s+$"; ""))}
  ]
  | group_by(.text)
  | map(select(length >= 2))
  | map({text: .[0].text, phases: [.[].phase], count: length})
  | .[]
' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")
```

**Note:** Success criteria text matching is fuzzy -- criteria like "Tests pass" and "All tests pass" should ideally match, but exact matching is simpler and less error-prone. Start with exact match (after normalization) and note that fuzzy matching could be added later.

### Pattern 6: Pheromone Content Format for Source Marking

**What:** Success criteria #4 requires auto-emitted pheromones to include their source type so users can distinguish them from manual pheromones.

**Format approach:** Include the source type in the pheromone content text itself, since pheromone-prime shows `[strength] content` and does not show the `source` field:

```
# PHER-01: Decision-sourced FEEDBACK
[decision] Use awk for CONTEXT.md parsing instead of jq

# PHER-02: Error-sourced REDIRECT
[error-pattern] Category "security" recurring (4 occurrences): check for exposed secrets

# PHER-03: Success-sourced FEEDBACK
[success-pattern] "Tests pass without regressions" recurs across 3 phases
```

The `[decision]`, `[error-pattern]`, and `[success-pattern]` labels in the content text make auto-emitted pheromones distinguishable at the prompt level (where builders read them) AND at the display level (where users see them via `/ant:pheromones`). The `source` field additionally stores the machine-readable `"auto:decision"` / `"auto:error"` / `"auto:success"` for programmatic filtering.

### Anti-Patterns to Avoid

- **Modifying aether-utils.sh for this phase:** All needed subcommands exist (`pheromone-write`, `midden-recent-failures`, `memory-capture`). The work is playbook text changes in continue-advance.md, not new code in aether-utils.sh.
- **Emitting pheromones during build instead of continue:** The requirements specify "running /ant:continue" as the trigger for all three emission sources. Build is the wrong lifecycle stage.
- **Overwriting user pheromones:** Auto-emitted pheromones should ADD to the signal set, never modify or remove user-emitted signals. Use lower strength (0.6-0.7) for auto-emitted vs. default user strength (0.7-0.9).
- **Emitting without deduplication:** If continue runs multiple times, each run could re-emit the same pheromones. Must check for existing auto-emitted pheromones with matching content before emitting duplicates.
- **Blocking phase advancement on emission failure:** Step 2.1 is explicitly SILENT -- all pheromone operations use `2>/dev/null || true`. Emission failures must never prevent phase advancement.
- **Creating too many pheromones:** Each source should be capped (e.g., max 3 decision pheromones, max 3 error pattern pheromones, max 2 success criteria pheromones per continue run) to avoid flooding the signal space.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pheromone signal creation | Custom JSON builder | `pheromone-write` subcommand (line 6680) | Handles ID generation, TTL calculation, locking, constraints.json backward compat, strength validation |
| Midden failure querying | Custom jq on midden.json | `midden-recent-failures` subcommand (line 9488) | Already handles missing file, sorting, limiting |
| Observation tracking | Custom tracking in continue playbook | `memory-capture` subcommand (line 5311) | Handles observation counting, deduplication via content hash, auto-promotion check |
| Pheromone injection to builders | Custom prompt assembly | Existing `pheromone-prime` -> `colony-prime` pipeline | Already reads all signals from pheromones.json, applies decay, formats for prompt |

**Key insight:** The pheromone pipeline already has a complete write path (`pheromone-write`) and read path (`pheromone-prime` -> `colony-prime` -> `build-context.md` -> `build-wave.md`). Phase 4 only needs to add new callers of `pheromone-write` in the continue flow. Success criteria #5 (auto-emitted pheromones appear in next build) is satisfied automatically by the existing pipeline -- no build-side changes needed.

## Common Pitfalls

### Pitfall 1: Duplicate Pheromone Emission
**What goes wrong:** Running `/ant:continue` twice on the same phase emits the same auto-pheromones twice.
**Why it happens:** The playbook doesn't check if a pheromone with matching content already exists.
**How to avoid:** Before each emission, check `pheromones.json` for active signals with matching source prefix (`auto:decision`, `auto:error`, `auto:success`) and similar content text. Skip if duplicate found.
**Warning signs:** Pheromone count spikes after multiple continue runs.

### Pitfall 2: Wrong Midden Data Source
**What goes wrong:** Using `errors.flagged_patterns` from COLONY_STATE.json instead of `midden.json` for error pattern detection.
**Why it happens:** The existing Step 2.1b uses `errors.flagged_patterns` which is a different (possibly empty) data structure.
**How to avoid:** Use `midden-recent-failures` subcommand which reads from `midden/midden.json`. Group by `category` field for recurrence detection.
**Warning signs:** No REDIRECT pheromones emitted despite known recurring failures in the midden.

### Pitfall 3: Empty Data Sources
**What goes wrong:** Emission blocks crash when `memory.decisions` is empty, midden has no entries, or there are no completed phases for success criteria comparison.
**Why it happens:** Missing null checks on jq queries.
**How to avoid:** Every data extraction must use `// []` or `// ""` fallbacks, and every emission block must check for empty data before proceeding. Mirror the defensive pattern from existing Step 2.1a-c.
**Warning signs:** Error output during what should be a SILENT step.

### Pitfall 4: Pheromone Content Injection
**What goes wrong:** Decision text or error messages containing shell metacharacters, quotes, or angle brackets corrupt pheromone-write arguments.
**Why it happens:** Decision text is free-form and may contain special characters.
**How to avoid:** `pheromone-write` already sanitizes content (line 6709-6714: replaces `<>`, truncates to 500 chars, rejects injection patterns). Pass content via quoted variables. Use jq's `@sh` for shell-safe escaping when building command strings from jq output.
**Warning signs:** `pheromone-write` exits with validation errors.

### Pitfall 5: Success Criteria Text Matching Is Brittle
**What goes wrong:** Slightly different wording ("Tests pass" vs "All tests pass") prevents detection of recurring success criteria.
**Why it happens:** Exact string matching after normalization still misses semantic duplicates.
**How to avoid:** Start with exact match (after lowercase + trim). This is a known limitation. Log a note that fuzzy matching could be added later. The first pass will catch genuinely identical criteria across phases.
**Warning signs:** Fewer success criteria emissions than expected.

## Code Examples

Verified patterns from the codebase:

### PHER-01: Decision-to-FEEDBACK Emission Block
```bash
# Source: continue-advance.md Step 2.1a pattern + context-update decision (line 507-512)
# Extract recent decisions from CONTEXT.md and emit FEEDBACK for each

decisions=$(awk '
  /^## .*Recent Decisions/ { in_section=1; next }
  in_section && /^\| Date / { next }
  in_section && /^\|[-]+/ { next }
  in_section && /^---/ { exit }
  in_section && /^\| [0-9]{4}-[0-9]{2}/ {
    split($0, fields, "|")
    decision = fields[3]
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", decision)
    if (decision != "") print decision
  }
' .aether/CONTEXT.md 2>/dev/null || echo "")

if [[ -n "$decisions" ]]; then
  emit_count=0
  while IFS= read -r dec && [[ $emit_count -lt 3 ]]; do
    [[ -z "$dec" ]] && continue
    # Deduplication: check if auto:decision pheromone with this text already exists
    existing=$(jq -r --arg text "$dec" '
      [.signals[] | select(.active == true and .source == "auto:decision" and (.content.text | contains($text)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")
    if [[ "$existing" == "0" ]]; then
      bash .aether/aether-utils.sh pheromone-write FEEDBACK \
        "[decision] $dec" \
        --strength 0.6 \
        --source "auto:decision" \
        --reason "Auto-emitted from phase decision during continue" \
        --ttl "30d" 2>/dev/null || true
      emit_count=$((emit_count + 1))
    fi
  done <<< "$decisions"
fi
```

### PHER-02: Midden Error Pattern-to-REDIRECT Emission Block
```bash
# Source: midden-recent-failures subcommand (line 9488) + continue-advance.md Step 2.1b pattern
# Query midden, group by category, emit REDIRECT for 3+ recurring categories

midden_result=$(bash .aether/aether-utils.sh midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
  # Group by category, find categories with 3+ occurrences
  recurring_categories=$(echo "$midden_result" | jq -r '
    [.failures[] | .category]
    | group_by(.)
    | map(select(length >= 3))
    | map({category: .[0], count: length})
    | .[]
    | @base64
  ' 2>/dev/null || echo "")

  emit_count=0
  for encoded in $recurring_categories; do
    [[ $emit_count -ge 3 ]] && break
    category=$(echo "$encoded" | base64 -d | jq -r '.category')
    count=$(echo "$encoded" | base64 -d | jq -r '.count')

    # Deduplication check
    existing=$(jq -r --arg cat "$category" '
      [.signals[] | select(.active == true and .source == "auto:error" and (.content.text | contains($cat)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")

    if [[ "$existing" == "0" ]]; then
      bash .aether/aether-utils.sh pheromone-write REDIRECT \
        "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
        --strength 0.7 \
        --source "auto:error" \
        --reason "Auto-emitted: midden error pattern recurred 3+ times" \
        --ttl "30d" 2>/dev/null || true
      emit_count=$((emit_count + 1))
    fi
  done
fi
```

### PHER-03: Success Criteria Recurrence-to-FEEDBACK Emission Block
```bash
# Source: COLONY_STATE.json plan.phases[].tasks[].success_criteria (line 3332)
# Compare success criteria across completed phases, emit FEEDBACK for recurring ones

recurring_criteria=$(jq -r '
  [.plan.phases[]
   | select(.status == "completed")
   | .id as $phase_id
   | (
       (.success_criteria // [])[] ,
       (.tasks // [] | .[].success_criteria // [])[]
     )
   | {phase: $phase_id, text: (. | ascii_downcase | gsub("^\\s+|\\s+$"; ""))}
  ]
  | group_by(.text)
  | map(select(length >= 2))
  | map({text: .[0].text, phases: [.[].phase] | unique, count: length})
  | .[:2]
  | .[]
  | @base64
' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

for encoded in $recurring_criteria; do
  [[ -z "$encoded" ]] && continue
  text=$(echo "$encoded" | base64 -d | jq -r '.text')
  count=$(echo "$encoded" | base64 -d | jq -r '.count')
  phases=$(echo "$encoded" | base64 -d | jq -r '.phases | join(", ")')

  # Deduplication check
  existing=$(jq -r --arg text "$text" '
    [.signals[] | select(.active == true and .source == "auto:success" and (.content.text | ascii_downcase | contains($text)))] | length
  ' .aether/data/pheromones.json 2>/dev/null || echo "0")

  if [[ "$existing" == "0" ]]; then
    bash .aether/aether-utils.sh pheromone-write FEEDBACK \
      "[success-pattern] \"$text\" recurs across phases $phases" \
      --strength 0.6 \
      --source "auto:success" \
      --reason "Auto-emitted: success criteria pattern recurred across $count phases" \
      --ttl "30d" 2>/dev/null || true
  fi
done
```

### Test Pattern (from Phase 1-3 integration tests)
```javascript
// Source: tests/integration/instinct-pipeline.test.js pattern
const test = require('ava');
const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

function runAetherUtil(tmpDir, command, args = []) {
  const scriptPath = path.join(process.cwd(), '.aether', 'aether-utils.sh');
  const env = {
    ...process.env,
    AETHER_ROOT: tmpDir,
    DATA_DIR: path.join(tmpDir, '.aether', 'data')
  };
  const cmd = `bash "${scriptPath}" ${command} ${args.map(a => `"${a}"`).join(' ')} 2>/dev/null`;
  return execSync(cmd, { encoding: 'utf8', env, cwd: tmpDir });
}

// Test: pheromone-write creates signal with custom source
test('pheromone-write auto:decision source is preserved', async t => {
  const tmpDir = await createTempDir();
  await setupTestColony(tmpDir);

  const result = runAetherUtil(tmpDir, 'pheromone-write', [
    'FEEDBACK', '[decision] Use awk for parsing',
    '--source', 'auto:decision',
    '--strength', '0.6',
    '--reason', 'Auto-emitted from phase decision',
    '--ttl', '30d'
  ]);

  const parsed = JSON.parse(result);
  t.true(parsed.ok);

  // Verify source is stored in pheromones.json
  const pherFile = path.join(tmpDir, '.aether', 'data', 'pheromones.json');
  const pheromones = JSON.parse(fs.readFileSync(pherFile, 'utf8'));
  const signal = pheromones.signals.find(s => s.source === 'auto:decision');
  t.truthy(signal);
  t.is(signal.type, 'FEEDBACK');
  t.true(signal.content.text.includes('[decision]'));

  await cleanupTempDir(tmpDir);
});
```

## State of the Art

| Old Approach (existing) | New Approach (Phase 4) | Impact |
|------------------------|------------------------|--------|
| `context-update decision` emits FEEDBACK with source `"system:decision"` | Batch decision emission during continue with source `"auto:decision"` and `[decision]` label | Decisions emitted at the right lifecycle point (continue) with distinguishable source |
| Step 2.1b checks `errors.flagged_patterns[]` with count >= 2 | Query `midden-recent-failures`, group by category, threshold 3+ | Uses actual failure data (midden.json) instead of potentially empty flagged_patterns |
| No success criteria recurrence detection | Compare completed phase success criteria, emit FEEDBACK for recurring patterns | New capability -- closes the third emission source |
| Source field values: `"worker:continue"`, `"system"` | New source prefix: `"auto:decision"`, `"auto:error"`, `"auto:success"` | Clean namespace for distinguishing auto-emitted from other sources |

**What keeps working unchanged:**
- `pheromone-prime` reads all signals from pheromones.json regardless of source -- auto-emitted signals automatically appear in builder prompts
- `pheromone-display` shows all active signals with their properties -- auto-emitted signals visible via `/ant:pheromones`
- Signal decay and expiration work the same for all sources
- `memory-capture` pipeline continues to work for learning/failure events in Step 2.5

## Open Questions

1. **Decision source ambiguity**
   - What we know: `context-update decision` already emits a FEEDBACK pheromone with source `"system:decision"`. The new PHER-01 emission would create additional FEEDBACK with source `"auto:decision"`.
   - What's unclear: Could this create duplicate signals for the same decision?
   - Recommendation: Add deduplication check that looks for both `"system:decision"` and `"auto:decision"` source signals with matching content before emitting.

2. **Midden category granularity**
   - What we know: Midden entries have a `category` field (e.g., "security", "test", "build") and a `message` field.
   - What's unclear: Are categories consistent enough for reliable grouping? Could the same error type appear under different categories?
   - Recommendation: Group primarily by `category`. If categories are too coarse, secondary grouping by message similarity could be added later.

3. **Success criteria comparison across phases**
   - What we know: Success criteria are free-text strings, potentially written differently each phase even when describing the same concept.
   - What's unclear: How often success criteria text actually recurs verbatim across phases in practice.
   - Recommendation: Start with exact match (after normalization). This is sufficient for the MVP. If too few matches are found, consider adding substring/keyword matching in a later iteration.

4. **Pheromone cap management**
   - What we know: pheromone-write appends signals without limit. Decay and expiration remove old signals.
   - What's unclear: Could auto-emission flood the signal space over many continue cycles?
   - Recommendation: Cap each emission type per continue run (3 decisions, 3 error patterns, 2 success criteria = max 8 new signals per continue). Combined with 30d TTL and decay, this should be manageable.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 5311-5409 -- `memory-capture` subcommand with auto-pheromone emission
- `.aether/aether-utils.sh` lines 6680-6863 -- `pheromone-write` subcommand implementation
- `.aether/aether-utils.sh` lines 7273-7402 -- `pheromone-prime` prompt assembly
- `.aether/aether-utils.sh` lines 8119-8186 -- `midden-write` subcommand
- `.aether/aether-utils.sh` lines 9488-9513 -- `midden-recent-failures` subcommand
- `.aether/aether-utils.sh` lines 507-512 -- `context-update decision` auto-FEEDBACK emission
- `.aether/docs/command-playbooks/continue-advance.md` -- Step 2.1 auto-emission blocks
- `.aether/docs/pheromones.md` -- Pheromone user guide with source conventions

### Secondary (MEDIUM confidence)
- `.aether/aether-utils.sh` lines 3300-3355 -- success_criteria structure in insert-phase
- `.aether/docs/command-playbooks/continue-finalize.md` -- Step 2.6 context-update decision call
- `.planning/phases/03-context-expansion/03-RESEARCH.md` -- CONTEXT.md decision table structure, prompt assembly order

### Tertiary (LOW confidence)
- None -- all findings verified against codebase source

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all subcommands verified in aether-utils.sh source code
- Architecture: HIGH -- continue-advance.md playbook structure directly inspected; modification points identified
- Pitfalls: HIGH -- deduplication, empty data, and injection concerns verified against existing defensive patterns in the codebase

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable domain -- bash playbook + existing subcommands)
