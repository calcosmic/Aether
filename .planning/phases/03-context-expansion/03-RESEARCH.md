# Phase 3: Context Expansion - Research

**Researched:** 2026-03-06
**Domain:** Bash shell scripting (aether-utils.sh), colony-prime prompt assembly, CONTEXT.md decision extraction, flags.json blocker injection
**Confidence:** HIGH

## Summary

Phase 3 closes the last two context gaps in the colony-prime prompt assembly pipeline: (1) key decisions recorded in `.aether/CONTEXT.md` never reach builders, and (2) escalated blocker flags in `.aether/data/flags.json` are invisible to builders even when they directly affect the work being done. After Phase 1 (instincts) and Phase 2 (learnings), colony-prime assembles its `prompt_section` from: QUEEN wisdom, context-capsule, phase learnings, and pheromone signals (including instincts). Neither CONTEXT.md decisions nor blocker flags flow through this pipeline.

The CONTEXT.md file has a "Recent Decisions" section formatted as a markdown table with columns: Date, Decision, Rationale, Made By. This is populated by `context-update decision` (lines 477-514 of aether-utils.sh), which also auto-emits a FEEDBACK pheromone for each decision. However, FEEDBACK pheromones are weak signals (strength 0.7, labeled "Flexible guidance") and easily drowned out by other signals in compact mode. The requirement CTX-01 asks colony-prime to extract the actual decision text as distinct builder context, not rely on the side-effect FEEDBACK pheromone.

For blocker flags (CTX-02), `flags.json` stores blocker entries with `type: "blocker"` and `severity: "critical"`. These are created during build escalations (build-wave.md line 476), verification failures (build-verify.md line 330), and chaos testing (build-verify.md line 292). Currently, `flag-check-blockers` counts them for gate checks in continue-gates.md, but their titles and descriptions never appear in builder prompts. The requirement asks these to appear as REDIRECT-priority warnings that are distinguishable from user-set REDIRECT pheromones.

**Primary recommendation:** Add two new extraction blocks to colony-prime: (1) a CONTEXT.md decision extraction block that parses the "Recent Decisions" markdown table and formats key decisions as actionable context, placed after phase learnings and before pheromone signals; (2) a flags.json blocker injection block that reads unresolved blocker flags for the current phase and formats them as REDIRECT-priority warnings with a distinct "BLOCKER WARNING" label (not "REDIRECT") to distinguish from user pheromones. Both blocks follow the established conditional-section pattern: extract data, check if non-empty, format as readable text, append to `cp_final_prompt`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CTX-01 | colony-prime reads CONTEXT.md and extracts key decisions for builder injection | CONTEXT.md has a "Recent Decisions" markdown table (lines 331-335 of context-update init); colony-prime currently has NO CONTEXT.md reading; extraction should parse the table and inject decision text between phase learnings and pheromone signals in prompt assembly |
| CTX-02 | Escalated blocker flags inject as REDIRECT warnings into builder prompts | flags.json stores blockers with type="blocker", severity="critical" (flag-add at line 1893); colony-prime currently has NO flag reading; extraction should read unresolved blockers for current phase via jq on flags.json and format as distinguishable REDIRECT-priority warnings |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | colony-prime subcommand modification for CONTEXT.md + blocker injection | Single source of truth for all prompt assembly |
| jq | System-installed | JSON filtering of flags.json blocker entries | Used throughout aether-utils.sh |
| awk/sed | System-installed | Markdown table parsing for CONTEXT.md decision extraction | Used throughout aether-utils.sh for text processing |
| ava | Installed in package.json | Integration test runner | Project standard, Phase 1 and 2 tests use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| colony-prime | aether-utils.sh subcommand (line 7451) | Unified priming -- add decision + blocker sections here | Modify to extract decisions from CONTEXT.md and blockers from flags.json |
| build-context.md | Playbook | Calls colony-prime, injects prompt_section into builder | Already wired -- NO changes needed (confirmed in Phase 1 and 2) |
| build-wave.md | Playbook | Builder prompt template with `{ prompt_section }` placeholder | Already wired -- NO changes needed |
| context-update decision | aether-utils.sh subcommand (line 477) | Writes decisions to CONTEXT.md markdown table | Already works; Phase 3 reads what it writes |
| flag-add / flag-check-blockers | aether-utils.sh subcommands (lines 1893, 1968) | Writes and counts flags in flags.json | Already works; Phase 3 reads flag details for injection |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Parsing CONTEXT.md markdown table directly | Reading `memory.decisions` from COLONY_STATE.json | `memory.decisions` is currently always empty (`[]`) -- decisions are stored in CONTEXT.md markdown table by `context-update decision`, not in COLONY_STATE.json. Would need to also wire decision writing to COLONY_STATE.json, which is out of scope for Phase 3 |
| Reading flags.json directly in colony-prime | Using existing `flag-check-blockers` subcommand | flag-check-blockers only returns counts, not flag titles/descriptions. Need full flag details for prompt injection |
| Adding blocker text to existing REDIRECT pheromone section | Creating a distinct "BLOCKER WARNING" section | Success criteria #4 requires blocker warnings to be distinguishable from user-set REDIRECT pheromones. A distinct section header solves this cleanly |
| Creating new subcommands (decision-prime, blocker-prime) | Inline extraction in colony-prime | Phase 1 and 2 proved that inline extraction in colony-prime works and avoids unnecessary indirection |

## Architecture Patterns

### File Touch Map

```
MODIFY:
  .aether/aether-utils.sh                                # colony-prime: add CONTEXT.md decision extraction + blocker flag injection

CREATE:
  tests/integration/context-expansion.test.js             # End-to-end context expansion tests

NO CHANGE NEEDED:
  .aether/docs/command-playbooks/build-context.md         # Already calls colony-prime, injects prompt_section
  .aether/docs/command-playbooks/build-wave.md            # Already injects { prompt_section } into builder prompts
  .aether/CONTEXT.md                                      # Already has "Recent Decisions" table (read-only for Phase 3)
  .aether/data/flags.json                                 # Already has blocker entries (read-only for Phase 3)
```

### Pattern 1: Colony-Prime Section Assembly Order (After Phase 2)
**What:** colony-prime builds `prompt_section` by concatenating discrete sections in order. Each section is conditionally included only when data exists.
**Current order (after Phase 2):**

```bash
cp_final_prompt=""

# 1. QUEEN wisdom (lines 7596-7622)
#    "--- QUEEN WISDOM (Eternal Guidance) ---"

# 2. Context capsule (lines 7625-7631)
#    "--- CONTEXT CAPSULE ---"

# 3. Phase learnings (lines 7633-7692) [Added in Phase 2]
#    "--- PHASE LEARNINGS (Previous Phase Insights) ---"

# === INSERT CONTEXT.MD DECISIONS HERE === (new for Phase 3, CTX-01)
# === INSERT BLOCKER WARNINGS HERE === (new for Phase 3, CTX-02)

# 4. Pheromone signals + instincts (lines 7694-7697)
#    "--- ACTIVE SIGNALS (Colony Guidance) ---"
```

**Why this position:** Decisions and blocker warnings are more immediate and action-relevant than historical learnings, but less structurally important than pheromone signals (which carry user-set hard constraints). Placing them between learnings and signals follows the information hierarchy: historical context -> current decisions -> active signals.

### Pattern 2: CONTEXT.md Decision Table Structure
**What:** The "Recent Decisions" section in `.aether/CONTEXT.md` is a markdown table written by `context-update decision` (line 477-514).
**Table format:**
```markdown
## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2026-02-16 | Remove /ant:export and /ant:import commands | User wants system integration, not new commands | User |
| 2026-02-16 | Complete Phase 4 | All exchange modules built and tested | Queen |
```

**Extraction approach:** Use awk to extract rows from the "Recent Decisions" table in CONTEXT.md. Each row between the table header separator (`|------|...`) and the next `---` section divider contains: date, decision text, rationale, and maker. Extract the Decision and Rationale columns for builder injection.

### Pattern 3: Blocker Flag Data Structure
**What:** Flags in `.aether/data/flags.json` have this shape:
```json
{
  "id": "flag_1771258216_8199",
  "type": "blocker",
  "severity": "critical",
  "title": "xml-utils.sh not integrated with aether-utils.sh",
  "description": "The XML utilities exist as standalone...",
  "source": "verification",
  "phase": 1,
  "created_at": "2026-02-16T16:10:16Z",
  "resolved_at": null,
  "auto_resolve_on": "build_pass"
}
```

**Extraction approach:** Use jq to filter unresolved blockers for the current phase (same logic as flag-check-blockers but extracting title + description + source instead of just counting).

### Pattern 4: Distinguishable Blocker Warnings
**What:** Success criteria #4 requires blocker-originated REDIRECT warnings to be distinguishable from user-set REDIRECT pheromones.
**Solution:** Use a distinct section header and format:

```
--- BLOCKER WARNINGS (Active Build Blockers) ---
[source: verification] xml-utils.sh not integrated with aether-utils.sh
  The XML utilities exist as standalone...
--- END BLOCKER WARNINGS ---
```

This is visually and semantically distinct from the pheromone REDIRECT section:
```
REDIRECT (HARD CONSTRAINTS - MUST follow):
[0.9] Never modify COLONY_STATE.json directly
```

The "BLOCKER WARNINGS" header, `[source: ...]` prefix, and separate section boundaries make them unambiguous.

### Anti-Patterns to Avoid
- **Modifying build-context.md or build-wave.md:** The `{ prompt_section }` placeholder already injects whatever colony-prime returns. Confirmed in Phase 1 and 2 that NO changes are needed there.
- **Reading the entire CONTEXT.md file into the prompt:** Success criteria #3 explicitly requires extracting only key decisions, not the entire file. CONTEXT.md is 5+ KB and includes session notes, activity logs, and health bars -- all irrelevant to builders.
- **Using pheromone-write to create REDIRECT signals from blockers:** This would make blockers indistinguishable from user REDIRECT pheromones (violating success criteria #4). Instead, inject blocker text directly into prompt_section with distinct formatting.
- **Creating a new subcommand (decision-prime, blocker-prime):** Phase 1 and 2 proved inline extraction in colony-prime works. No new subcommands needed.
- **Writing blockers to pheromones.json:** Blockers already live in flags.json. Duplicating them to pheromones.json would create sync issues and conflate two distinct systems.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CONTEXT.md markdown parsing | Full markdown parser | awk-based table row extraction | CONTEXT.md has a fixed structure; awk handles the specific "Recent Decisions" table format. No need for a general markdown parser |
| Blocker flag reading | Custom file parsing | jq filters on flags.json | flags.json is standard JSON; jq handles the filtering (type==blocker, resolved_at==null, phase match) |
| Prompt section assembly | Template engine | String concatenation with `$'\n'` | Exact pattern used by every other section in colony-prime |
| JSON escaping for output | Manual escaping | `jq -Rs '.'` pipe pattern | Already used at colony-prime line 7700 |
| Phase-scoped blocker filtering | Custom phase matching | jq with phase == $current OR phase == null | Same logic as flag-check-blockers (line 1979-1985) |

**Key insight:** Both data sources (CONTEXT.md decisions and flags.json blockers) already exist and are well-structured. The work is purely connecting existing data to an existing injection pipeline. No new data structures, files, or subcommands needed.

## Common Pitfalls

### Pitfall 1: CONTEXT.md Might Not Exist
**What goes wrong:** If `.aether/CONTEXT.md` doesn't exist (e.g., fresh colony before first build), colony-prime would error trying to read it.
**Why it happens:** CONTEXT.md is created by `context-update init` which runs during colony init. But if a colony was initialized before context-update was implemented, or if the file was deleted, it won't exist.
**How to avoid:** Check `[[ -f "$ctx_file" ]]` before attempting to parse. If missing, skip the decisions section silently (no error, no empty section). Same pattern as pheromone-prime's check for pheromones.json at line 7571.
**Warning signs:** colony-prime fails with file-not-found error during builds on fresh or legacy colonies.

### Pitfall 2: Empty Decisions Table
**What goes wrong:** CONTEXT.md exists but the "Recent Decisions" table has no data rows (only the header). The extraction produces empty output but the section header still appears.
**Why it happens:** No decisions have been recorded yet for this colony.
**How to avoid:** After extraction, check if any decision rows were found. Only append the "--- KEY DECISIONS ---" section if at least one decision was extracted. Matches the conditional pattern used throughout colony-prime.
**Warning signs:** Builder prompts show "--- KEY DECISIONS ---" followed immediately by "--- END KEY DECISIONS ---".

### Pitfall 3: Decision Text Contains Pipe Characters
**What goes wrong:** If a decision description contains `|` characters (markdown table delimiter), the awk parsing breaks and produces garbled output.
**Why it happens:** The `context-update decision` subcommand writes user-provided text directly into the markdown table without escaping pipes.
**How to avoid:** When parsing, treat the first and last `|` as delimiters but join any intermediate fields. Or better: since `context-update decision` controls what goes in, the risk is low. Add a defensive sed to strip extra pipes if needed.
**Warning signs:** Decision text appears truncated or corrupted in builder prompts.

### Pitfall 4: flags.json Might Not Exist
**What goes wrong:** If no flags have ever been created, `.aether/data/flags.json` doesn't exist.
**Why it happens:** flags.json is created on first `flag-add` call. A colony that has never encountered a blocker won't have it.
**How to avoid:** Check `[[ -f "$flags_file" ]]` before attempting to parse. If missing, skip the blocker section silently. Same pattern as flag-check-blockers (line 1974).
**Warning signs:** colony-prime fails with file-not-found error.

### Pitfall 5: Prompt Size Bloat
**What goes wrong:** A colony with many decisions and many blockers produces a bloated prompt_section that wastes tokens.
**Why it happens:** No cap on how many decisions or blockers are injected.
**How to avoid:** Cap decisions at 5 in non-compact mode, 3 in compact mode. Cap blockers at 3 in non-compact, 2 in compact. These are reasonable limits that keep the builder focused on the most recent/relevant information. Take most recent decisions (bottom of the markdown table = most recent). Take all unresolved blockers for the current phase (these are inherently limited and urgent).
**Warning signs:** Builder prompt_section becomes unusually long, or token limits are hit.

### Pitfall 6: Blockers From Wrong Phase
**What goes wrong:** Blocker flags from a different phase appear in the current phase's builder prompts, creating confusion.
**Why it happens:** Some blockers have `phase: null` (global) while others have `phase: N` (phase-specific). Naive filtering shows all unresolved blockers.
**How to avoid:** Use the same filtering logic as `flag-check-blockers` (line 1979-1985): include blockers where `phase == current_phase OR phase == null`. Phase-null blockers are global concerns. Phase-specific blockers only show for their phase.
**Warning signs:** Builders see blocker warnings about issues in a different phase.

### Pitfall 7: CONTEXT.md Decision Extraction vs memory.decisions Confusion
**What goes wrong:** Developer assumes `memory.decisions` in COLONY_STATE.json has the decision data, but it's always empty `[]`.
**Why it happens:** `context-update decision` writes to the CONTEXT.md markdown table AND emits a FEEDBACK pheromone, but does NOT write to `memory.decisions` in COLONY_STATE.json. The `memory.decisions` field exists in the template but is never populated by the decision subcommand.
**How to avoid:** Read from `.aether/CONTEXT.md` directly (the file), not from COLONY_STATE.json. This is where the actual decision records live.
**Warning signs:** If you query `memory.decisions`, you get `[]` and conclude there are no decisions.

## Code Examples

### CONTEXT.md Decision Extraction (awk)
```bash
# Extract decision rows from CONTEXT.md "Recent Decisions" table
# CONTEXT.md path follows established convention
cp_ctx_file="$AETHER_ROOT/.aether/CONTEXT.md"

cp_decisions=""
if [[ -f "$cp_ctx_file" ]]; then
  # Extract table rows between "## Recent Decisions" header and the next "---" separator
  # Skip the table header row (|------|...) and extract Decision + Rationale columns
  cp_decisions=$(awk '
    /^## .*Recent Decisions/ { in_section=1; next }
    in_section && /^\| Date / { next }         # Skip table header
    in_section && /^\|[-]+/ { next }            # Skip separator row
    in_section && /^---/ { exit }               # Stop at next section
    in_section && /^\| [0-9]{4}-[0-9]{2}/ {     # Match date-prefixed rows
      # Extract Decision (field 3) and Rationale (field 4)
      split($0, fields, "|")
      decision = fields[3]
      rationale = fields[4]
      # Trim whitespace
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", decision)
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", rationale)
      if (decision != "") {
        if (rationale != "" && rationale != "-") {
          print decision " (" rationale ")"
        } else {
          print decision
        }
      }
    }
  ' "$cp_ctx_file" 2>/dev/null || echo "")
fi
```

### Blocker Flag Extraction (jq)
```bash
# Extract unresolved blocker flags for current phase
cp_flags_file="$DATA_DIR/flags.json"

cp_blockers=""
if [[ -f "$cp_flags_file" ]]; then
  cp_blockers=$(jq -r \
    --argjson phase "$cp_current_phase" \
    '
    .flags
    | map(select(
        .type == "blocker"
        and .resolved_at == null
        and (.phase == $phase or .phase == null)
      ))
    | map("[source: " + (.source // "unknown") + "] " + .title + "\n  " + (.description // ""))
    | .[]
    ' "$cp_flags_file" 2>/dev/null || echo "")
fi
```

### Decision Section Formatting
```bash
# Format extracted decisions for prompt injection
cp_max_decisions=5
if [[ "$cp_compact" == "true" ]]; then
  cp_max_decisions=3
fi

cp_decision_section=""
if [[ -n "$cp_decisions" ]]; then
  # Take only the last N decisions (most recent at bottom of table)
  cp_trimmed=$(echo "$cp_decisions" | tail -n "$cp_max_decisions")

  cp_decision_section="--- KEY DECISIONS (Active Decisions) ---"$'\n'
  while IFS= read -r line; do
    [[ -n "$line" ]] && cp_decision_section+="- $line"$'\n'
  done <<< "$cp_trimmed"
  cp_decision_section+="--- END KEY DECISIONS ---"

  cp_decision_count=$(echo "$cp_trimmed" | grep -c '.' || echo "0")
fi
```

### Blocker Section Formatting
```bash
# Format blocker flags for prompt injection (distinct from REDIRECT pheromones)
cp_max_blockers=3
if [[ "$cp_compact" == "true" ]]; then
  cp_max_blockers=2
fi

cp_blocker_section=""
if [[ -n "$cp_blockers" ]]; then
  cp_blocker_count=$(echo "$cp_blockers" | grep -c '^\[source:' || echo "0")

  if [[ "$cp_blocker_count" -gt 0 ]]; then
    cp_blocker_section="--- BLOCKER WARNINGS (Unresolved Build Blockers) ---"$'\n'
    cp_blocker_section+="These are critical issues that MUST be addressed. Treat as REDIRECT-priority."$'\n'

    # Take first N blockers (most important/recent)
    blocker_idx=0
    while IFS= read -r line; do
      if [[ "$blocker_idx" -ge "$cp_max_blockers" ]]; then break; fi
      [[ -n "$line" ]] && cp_blocker_section+="$line"$'\n'
      ((blocker_idx++)) || true
    done <<< "$cp_blockers"

    cp_blocker_section+="--- END BLOCKER WARNINGS ---"
  fi
fi
```

### Colony-Prime Insertion Point (after Phase 2)
```bash
# In colony-prime, after phase learnings block (line ~7692) and before pheromone signals (line ~7694):

# === End phase learnings injection === (existing from Phase 2)

# === NEW: CONTEXT.md decision injection (CTX-01) ===
cp_ctx_file="$AETHER_ROOT/.aether/CONTEXT.md"
# ... extraction code from above ...
if [[ -n "$cp_decision_section" ]]; then
  cp_final_prompt+=$'\n'"$cp_decision_section"$'\n'
  cp_log_line="$cp_log_line, $cp_decision_count decisions"
fi
# === END CONTEXT.md decision injection ===

# === NEW: Blocker flag injection (CTX-02) ===
cp_flags_file="$DATA_DIR/flags.json"
# ... extraction code from above ...
if [[ -n "$cp_blocker_section" ]]; then
  cp_final_prompt+=$'\n'"$cp_blocker_section"$'\n'
  cp_log_line="$cp_log_line, $cp_blocker_count blockers"
fi
# === END blocker flag injection ===

# Add pheromone signals section (existing)
if [[ -n "$cp_prompt_section" && "$cp_prompt_section" != "null" ]]; then
  cp_final_prompt+=$'\n'"$cp_prompt_section"
fi
```

### Complete Prompt Assembly Order (After Phase 3)
```
--- QUEEN WISDOM (Eternal Guidance) ---
  Philosophies, Patterns, Redirects, Stack Wisdom, Decrees
--- END QUEEN WISDOM ---

--- CONTEXT CAPSULE ---
  Goal, State, Phase, Next action, Signals, Decisions, Risks, Rolling summary

--- PHASE LEARNINGS (Previous Phase Insights) ---    [Phase 2]
  Phase 1 (instinct-pipeline):
    - Confidence floor >= 0.7 prevents noise instincts
  Inherited:
    - Claude Code global sync works by copying commands
--- END PHASE LEARNINGS ---

--- KEY DECISIONS (Active Decisions) ---              [Phase 3 NEW]
  - Remove /ant:export and /ant:import commands (User wants system integration)
  - Complete Phase 4 (All exchange modules built and tested)
--- END KEY DECISIONS ---

--- BLOCKER WARNINGS (Unresolved Build Blockers) --- [Phase 3 NEW]
  These are critical issues that MUST be addressed. Treat as REDIRECT-priority.
  [source: verification] xml-utils.sh not integrated
    The XML utilities exist as standalone...
--- END BLOCKER WARNINGS ---

--- ACTIVE SIGNALS (Colony Guidance) ---
  FOCUS (Pay attention to): ...
  REDIRECT (HARD CONSTRAINTS - MUST follow): ...
  FEEDBACK (Flexible guidance): ...
--- INSTINCTS (Learned Behaviors) ---
  ...
--- END COLONY CONTEXT ---
```

### Test Pattern
```javascript
test.serial('colony-prime includes CONTEXT.md decisions in prompt', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir, { currentPhase: 2 });

    // Write a CONTEXT.md with decisions
    const contextMd = `# Aether Colony -- Current Context

## Recent Decisions

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2026-03-06 | Use awk for parsing | Simpler than regex | Queen |
| 2026-03-06 | Cap at 5 decisions | Prompt budget | Colony |

---

## Recent Activity
`;
    await fs.promises.writeFile(
      path.join(tmpDir, '.aether', 'CONTEXT.md'),
      contextMd
    );

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const json = JSON.parse(result);
    t.true(json.ok);

    const prompt = json.result.prompt_section;
    t.true(prompt.includes('KEY DECISIONS'));
    t.true(prompt.includes('Use awk for parsing'));
    t.true(prompt.includes('Cap at 5 decisions'));
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('colony-prime includes blocker warnings from flags.json', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir, { currentPhase: 2 });

    // Write flags.json with an unresolved blocker for phase 2
    const flags = {
      version: 1,
      flags: [{
        id: "flag_test_001",
        type: "blocker",
        severity: "critical",
        title: "Tests failing on module X",
        description: "Integration tests for module X return timeout errors",
        source: "verification",
        phase: 2,
        created_at: "2026-03-06T12:00:00Z",
        resolved_at: null,
        auto_resolve_on: "build_pass"
      }]
    };
    await fs.promises.writeFile(
      path.join(tmpDir, '.aether', 'data', 'flags.json'),
      JSON.stringify(flags, null, 2)
    );

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const json = JSON.parse(result);
    t.true(json.ok);

    const prompt = json.result.prompt_section;
    t.true(prompt.includes('BLOCKER WARNINGS'));
    t.true(prompt.includes('Tests failing on module X'));
    t.true(prompt.includes('[source: verification]'));
    // Must NOT appear in the REDIRECT section
    t.false(prompt.includes('REDIRECT (HARD CONSTRAINTS'));
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('blocker warnings are distinguishable from user REDIRECT pheromones', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir, { currentPhase: 1 });

    // Add both a REDIRECT pheromone and a blocker flag
    // ... setup pheromones.json with REDIRECT signal ...
    // ... setup flags.json with blocker ...

    const result = runAetherUtil(tmpDir, 'colony-prime');
    const json = JSON.parse(result);
    const prompt = json.result.prompt_section;

    // REDIRECT section should contain user pheromone
    t.true(prompt.includes('REDIRECT (HARD CONSTRAINTS'));
    // BLOCKER section should contain flag-originated warning
    t.true(prompt.includes('BLOCKER WARNINGS'));
    // They should be in different sections
    const redirectIdx = prompt.indexOf('REDIRECT (HARD CONSTRAINTS');
    const blockerIdx = prompt.indexOf('BLOCKER WARNINGS');
    t.true(blockerIdx < redirectIdx); // blockers before signals
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
```

## State of the Art

| Current State | Target State | What Changes | Impact |
|---------------|-------------|--------------|--------|
| colony-prime has NO CONTEXT.md reading | colony-prime extracts key decisions from CONTEXT.md "Recent Decisions" table | New extraction + formatting block in colony-prime | Builders see what was decided |
| colony-prime has NO flag reading | colony-prime reads unresolved blockers from flags.json and injects as REDIRECT-priority warnings | New extraction + formatting block in colony-prime | Builders are warned about active blockers |
| prompt_section: wisdom + capsule + learnings + signals/instincts | prompt_section: wisdom + capsule + learnings + DECISIONS + BLOCKERS + signals/instincts | Two new conditional sections | Richer, more complete builder context |
| log_line reports "N signals, M instincts, L learnings" | log_line reports "N signals, M instincts, L learnings, D decisions, B blockers" | Append counts to existing log_line | User sees decision and blocker counts in build output |
| Blocker flags only checked at gates (continue-gates) | Blocker flags visible to builders during work | Proactive vs reactive blocker awareness | Builders can avoid known issues before hitting them |
| Decisions reach builders only as weak FEEDBACK pheromones | Decisions appear as distinct "KEY DECISIONS" context | Stronger, clearer decision visibility | Decisions are not lost in signal noise |

## Open Questions

1. **Should decisions from CONTEXT.md or COLONY_STATE.json be the source?**
   - What we know: `context-update decision` writes to CONTEXT.md markdown table AND auto-emits a FEEDBACK pheromone, but does NOT write to `memory.decisions` in COLONY_STATE.json. The `memory.decisions` array is always empty.
   - What's unclear: Should Phase 3 also wire `context-update decision` to write to `memory.decisions`? This would provide a JSON-structured source that's easier to parse.
   - Recommendation: For Phase 3, read from CONTEXT.md directly (it has the actual data). Wiring `memory.decisions` population is a separate concern and can be done later. Keep Phase 3 focused on the read path, not the write path.

2. **How many decisions should be injected?**
   - What we know: Success criteria #3 says "extracts only key decisions (not the entire CONTEXT.md file) to keep prompt size manageable." The context-capsule already shows 3 decisions from `memory.decisions`.
   - What's unclear: Exact cap number.
   - Recommendation: 5 decisions in non-compact mode, 3 in compact. These are the most recent (bottom of markdown table). This keeps the section small while covering the most relevant decisions.

3. **Should resolved blockers be included?**
   - What we know: flags.json has `resolved_at` field. Current flag-check-blockers only counts unresolved.
   - Recommendation: Only unresolved blockers (`resolved_at == null`). Resolved blockers are historical context, not active warnings. This matches the "escalated blocker" language in the requirement.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 7451-7732: colony-prime subcommand -- verified full assembly order, identified insertion points after Phase 2 additions
- `.aether/aether-utils.sh` lines 7633-7692: Phase 2 learnings injection block -- confirmed insertion point for Phase 3 sections
- `.aether/aether-utils.sh` lines 477-514: `context-update decision` subcommand -- verified CONTEXT.md markdown table write format
- `.aether/aether-utils.sh` lines 1893-1966: `flag-add` subcommand -- verified flags.json data structure
- `.aether/aether-utils.sh` lines 1968-1995: `flag-check-blockers` subcommand -- verified phase-scoped filtering logic
- `.aether/aether-utils.sh` lines 7273-7449: pheromone-prime subcommand -- verified REDIRECT signal formatting (to ensure blocker format is distinct)
- `.aether/CONTEXT.md` (actual file): verified "Recent Decisions" table format and content
- `.aether/data/flags.json` (actual file): verified blocker flag structure with resolved/unresolved states
- `.aether/data/COLONY_STATE.json` (actual file): verified `memory.decisions` is empty `[]`
- `.aether/docs/command-playbooks/build-context.md`: confirmed colony-prime call and prompt_section injection (no changes needed)
- `.aether/docs/command-playbooks/build-wave.md`: confirmed `{ prompt_section }` placeholder in builder prompts (no changes needed)
- `.aether/docs/command-playbooks/build-wave.md` line 476: confirmed blocker flag creation during escalation
- `.aether/docs/command-playbooks/build-verify.md` lines 292, 330: confirmed blocker flag creation during chaos testing and verification
- `tests/integration/learnings-injection.test.js`: test pattern with setupTestColony, runAetherUtil, createTempDir helpers
- `.planning/phases/02-learnings-injection/02-RESEARCH.md`: established research patterns and architecture decisions that carry forward

### Secondary (MEDIUM confidence)
- `.aether/templates/colony-state.template.json`: COLONY_STATE.json structure showing `memory.decisions: []` as default
- `.aether/docs/command-playbooks/continue-finalize.md` line 237: confirmed decisions are logged via `context-update decision`

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all code paths traced through source, insertion points identified at specific line numbers
- Architecture: HIGH - file touch map derived from actual colony-prime assembly chain; Phase 1 and 2 confirmed that build-context.md and build-wave.md need no changes
- Pitfalls: HIGH - CONTEXT.md structure verified from actual file; flags.json structure verified from actual file; empty decisions array verified from actual COLONY_STATE.json; all pitfalls derived from real code analysis

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable codebase, internal tooling)
