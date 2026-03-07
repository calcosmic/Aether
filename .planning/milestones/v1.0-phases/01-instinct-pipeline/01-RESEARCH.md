# Phase 1: Instinct Pipeline - Research

**Researched:** 2026-03-06
**Domain:** Bash shell scripting (aether-utils.sh), Markdown playbook wiring, JSON state management
**Confidence:** HIGH

## Summary

Phase 1 wires two existing subcommands -- `instinct-create` and `instinct-read` -- into two existing playbooks: `continue-advance.md` (write side) and `build-context.md` / `pheromone-prime` (read side). Both subcommands already exist, are tested at the e2e level, and have well-defined JSON APIs. The primary work is playbook text changes and behavior adjustments, not new code from scratch.

The instinct pipeline has three touch points: (1) `continue-advance.md` Step 3, which already calls `instinct-create` but needs confidence thresholds tightened to >= 0.7 and source diversity widened to include midden/error patterns; (2) `pheromone-prime` in `aether-utils.sh`, which already reads instincts and formats them but needs domain-grouped output; (3) `build-context.md` Step 4, which already calls `colony-prime` and injects `prompt_section` -- this path works today with zero changes needed.

**Primary recommendation:** Modify the continue-advance.md playbook to enforce >= 0.7 confidence threshold, add midden pattern sourcing, and add visible instinct output. Modify pheromone-prime to group instincts by domain. Fix the instinct-read fallthrough bug. Write integration tests covering the create-to-prompt pipeline.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- All three pattern sources feed instinct creation: phase learnings, error patterns (midden), and success patterns
- When a new pattern matches an existing instinct, strengthen the existing instinct (bump confidence score) rather than creating a duplicate
- instinct-create must be called in continue-advance.md after learnings are extracted
- Instincts grouped by domain in the prompt (testing instincts together, architecture instincts together, etc.)
- Injected instincts must be visible in build output so the user can see what the colony knows (not silent)
- A REDIRECT pheromone can override a conflicting instinct at runtime
- When instinct and pheromone conflict, highest confidence signal wins regardless of source
- Instincts are colony-scoped -- they do NOT survive seal/entomb. Cross-colony persistence is QUEEN.md's job (Phase 5)

### Claude's Discretion
- Confidence threshold for instinct creation (0.5-0.9 range)
- Maximum instincts per phase (recommended 3-5 but not hard-capped)
- Instinct verbosity in builder prompts
- Prompt budget allocation between instincts, pheromones, and other context
- Whether instincts decay over time or remain at created confidence

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LEARN-02 | continue-advance calls instinct-create for patterns with confidence >= 0.7 | continue-advance.md Step 3 already calls instinct-create at lines 82-103 but with 0.4-0.7 range; needs threshold tightening and midden/success source additions |
| LEARN-03 | instinct-read results included in colony-prime prompt_section output | pheromone-prime already reads instincts (line 7354-7367) and colony-prime already calls pheromone-prime (line 7560); path exists but needs domain grouping |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | All instinct CRUD operations via subcommands | Single source of truth for all state operations |
| jq | System-installed | JSON manipulation in bash | Used throughout aether-utils.sh for COLONY_STATE.json operations |
| ava | Installed in package.json | Unit/integration test runner | Project standard, all existing tests use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| continue-advance.md | Playbook | Defines continue flow Step 2-3 | Wire instinct-create calls with new confidence rules |
| build-context.md | Playbook | Defines build Step 4 (colony-prime) | Already wired, verify domain grouping |
| pheromone-prime | aether-utils.sh subcommand | Assembles prompt_section with signals + instincts | Modify to group instincts by domain |
| colony-prime | aether-utils.sh subcommand | Unified priming (wisdom + signals + instincts) | Already calls pheromone-prime, no changes needed |

## Architecture Patterns

### File Touch Map

```
MODIFY:
  .aether/docs/command-playbooks/continue-advance.md    # Step 3: tighten thresholds, add midden sourcing
  .aether/aether-utils.sh                                # pheromone-prime: domain grouping; instinct-read: fix bug
  .aether/docs/command-playbooks/continue-finalize.md    # Step 3 display: visible instinct output

CREATE:
  tests/integration/instinct-pipeline.test.js            # End-to-end instinct tests

NO CHANGE NEEDED:
  .aether/docs/command-playbooks/build-context.md        # Already calls colony-prime, injects prompt_section
  .aether/docs/command-playbooks/build-wave.md           # Already injects { prompt_section } into builder prompts
```

### Pattern 1: Subcommand JSON API Contract
**What:** Every aether-utils.sh subcommand returns `{"ok":true,"result":{...}}` or `{"ok":false,"error":{...}}`
**When to use:** All instinct operations follow this pattern
**Example:**
```bash
# instinct-create returns:
{"ok":true,"result":{"instinct_id":"instinct_1709736000","action":"created","confidence":0.7}}

# instinct-create with duplicate returns:
{"ok":true,"result":{"instinct_id":"existing","action":"updated","confidence":0.8}}

# instinct-read returns:
{"ok":true,"result":{"instincts":[...],"total":5,"filtered":3}}
```

### Pattern 2: Playbook Step Structure
**What:** Playbook steps use numbered sections with bash code blocks and display templates
**When to use:** All modifications to continue-advance.md and continue-finalize.md must follow this format
**Example:**
```markdown
### Step N: Description

Explanation text.

Run using the Bash tool with description "Descriptive text...":
\`\`\`bash
bash .aether/aether-utils.sh subcommand --flag value 2>/dev/null || true
\`\`\`
```

### Pattern 3: Test Fixture Setup
**What:** Tests use temp directories with copied aether-utils.sh, isolated COLONY_STATE.json, and AETHER_ROOT/DATA_DIR env vars
**When to use:** All new instinct tests
**Example:**
```javascript
// From tests/unit/context-continuity.test.js
function setupTempAether(tempDir) {
  const repoRoot = path.join(__dirname, '..', '..');
  const srcAetherDir = path.join(repoRoot, '.aether');
  const dstAetherDir = path.join(tempDir, '.aether');
  const dstDataDir = path.join(dstAetherDir, 'data');

  fs.mkdirSync(dstAetherDir, { recursive: true });
  fs.mkdirSync(dstDataDir, { recursive: true });
  fs.copyFileSync(path.join(srcAetherDir, 'aether-utils.sh'), path.join(dstAetherDir, 'aether-utils.sh'));
  fs.cpSync(path.join(srcAetherDir, 'utils'), path.join(dstAetherDir, 'utils'), { recursive: true });
  // ... exchange and schemas dirs
}

function runUtil(tempDir, subcommand, args = []) {
  const env = { ...process.env, AETHER_ROOT: tempDir, DATA_DIR: path.join(tempDir, '.aether', 'data') };
  const cmd = `bash .aether/aether-utils.sh ${subcommand} ${quoted}`;
  return JSON.parse(execSync(cmd, { cwd: tempDir, env, encoding: 'utf8' }));
}
```

### Anti-Patterns to Avoid
- **Modifying build-wave.md for instinct injection:** The `{ prompt_section }` placeholder in the builder prompt template (build-wave.md line 319) already injects whatever colony-prime returns. Do NOT add separate instinct injection logic in build-wave.md.
- **Adding new subcommands:** The existing `instinct-create`, `instinct-read`, `pheromone-prime`, and `colony-prime` subcommands are sufficient. Do not create new ones.
- **Calling instinct-read directly from build playbooks:** colony-prime -> pheromone-prime -> instinct-read is the existing chain. Bypass would create divergent paths.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Duplicate detection | Custom string matching | instinct-create's built-in dedup (line 7195-7223) | Already handles exact trigger+action matching and confidence boosting |
| Instinct cap enforcement | Manual array trimming | instinct-create's 30-instinct cap (line 7253-7260) | Already sorts by confidence and evicts lowest |
| JSON state writes | Direct file writes | atomic_write function (line 7216, 7263) | Prevents corruption on concurrent access |
| Prompt assembly | Custom template concatenation | pheromone-prime + colony-prime chain | Already handles signal prioritization, escaping, JSON encoding |

**Key insight:** The instinct infrastructure is fully built. This phase is wiring, not construction.

## Common Pitfalls

### Pitfall 1: instinct-read Fallthrough Bug
**What goes wrong:** `instinct-read` at line 7122-7124 calls `json_ok` when no instincts exist but does NOT exit/return. Execution falls through to line 7126 which outputs a SECOND JSON blob. This produces invalid double-JSON output.
**Why it happens:** `json_ok()` is defined as `printf` without `exit` (line 65). Other subcommands use `;;` case termination to exit, but the early-return guard at 7122 needs an explicit `exit 0` or the code needs restructuring.
**How to avoid:** Add `exit 0` after line 7123, or restructure as an `else` block around lines 7126-7152.
**Warning signs:** Tests that parse `instinct-read` output on empty colonies may silently succeed if `JSON.parse` takes only the first line, but will fail in contexts that read the full stdout.

### Pitfall 2: Confidence Threshold Mismatch
**What goes wrong:** continue-advance.md Step 3 (line 97-100) currently documents confidence 0.4-0.7, but the success criteria require >= 0.7 for automatic instinct creation.
**Why it happens:** The current playbook was written with lower thresholds for manual use. LEARN-02 requires the >= 0.7 gate.
**How to avoid:** When modifying Step 3, set confidence floor to 0.7 for auto-created instincts. Keep the instinct-create subcommand itself accepting any value (it validates 0-1 range) -- the threshold is a playbook-level policy, not a subcommand-level restriction.
**Warning signs:** If tests pass instincts with confidence < 0.7 and they appear in builder prompts, the threshold gate is not working.

### Pitfall 3: pheromone-prime Instinct Section Not Grouped by Domain
**What goes wrong:** The current instinct formatting in pheromone-prime (line 7420) outputs flat confidence-sorted lines: `[0.8] When X -> Y (architecture)`. User decision requires grouping by domain.
**Why it happens:** The original implementation prioritized confidence sorting over domain grouping.
**How to avoid:** Modify the jq pipeline at line 7420 to group by domain first, then sort by confidence within each group. Use `group_by(.domain)` in jq.
**Warning signs:** If the `--- INSTINCTS ---` section shows mixed domains (testing, architecture, testing), grouping is not applied.

### Pitfall 4: Midden Pattern Sourcing Requires midden-recent-failures
**What goes wrong:** The user wants error patterns from midden to feed instinct creation, but continue-advance.md Step 3 currently only creates instincts from phase observation patterns.
**Why it happens:** Midden integration was not part of the original instinct creation flow.
**How to avoid:** Call `midden-recent-failures` in Step 3 and convert recurring error patterns (count >= 2) into instincts with appropriate trigger/action/domain/confidence. The subcommand exists (line 9310) and returns JSON with failure records.
**Warning signs:** If instincts only come from learnings and never from error patterns, the midden sourcing is missing.

### Pitfall 5: Double JSON Output in Tests
**What goes wrong:** JavaScript test code uses `JSON.parse(execSync(...))` which only parses the first JSON object in stdout. If a subcommand emits double JSON (see Pitfall 1), tests pass silently but runtime consumers that read full stdout fail.
**Why it happens:** `JSON.parse` is lenient about trailing content in some Node.js versions.
**How to avoid:** After fixing the instinct-read bug, add a test that verifies stdout contains exactly one JSON line.

## Code Examples

### instinct-create API (from aether-utils.sh lines 7155-7269)
```bash
# Create a new instinct
bash .aether/aether-utils.sh instinct-create \
  --trigger "when tests fail with timeout errors" \
  --action "increase test timeout to 30s and add retry wrapper" \
  --confidence 0.7 \
  --domain "testing" \
  --source "phase-1" \
  --evidence "Timeout failures in 3 consecutive test runs, resolved by timeout increase"

# Returns on new: {"ok":true,"result":{"instinct_id":"instinct_1709736000","action":"created","confidence":0.7}}
# Returns on dup: {"ok":true,"result":{"instinct_id":"existing","action":"updated","confidence":0.8}}
```

### instinct-read API (from aether-utils.sh lines 7084-7153)
```bash
# Read top 5 instincts with confidence >= 0.5
bash .aether/aether-utils.sh instinct-read --min-confidence 0.5 --max 5

# Read only testing-domain instincts
bash .aether/aether-utils.sh instinct-read --domain "testing"

# Returns: {"ok":true,"result":{"instincts":[{...}],"total":10,"filtered":5}}
```

### Instinct Object Schema (from instinct-create, lines 7226-7251)
```json
{
  "id": "instinct_1709736000",
  "trigger": "when tests fail with timeout errors",
  "action": "increase test timeout to 30s and add retry wrapper",
  "confidence": 0.7,
  "status": "hypothesis",
  "domain": "testing",
  "source": "phase-1",
  "evidence": ["Timeout failures in 3 consecutive test runs"],
  "tested": false,
  "created_at": "2026-03-06T12:00:00Z",
  "last_applied": null,
  "applications": 0,
  "successes": 0,
  "failures": 0
}
```

### pheromone-prime Instinct Output Format (current, lines 7413-7423)
```
--- INSTINCTS (Learned Behaviors) ---
Weight by confidence - higher = stronger guidance:

[0.8] When tests fail with timeout errors -> increase test timeout (testing)
[0.7] When importing modules -> use absolute paths from project root (architecture)
```

### pheromone-prime Instinct Output Format (target: domain-grouped)
```
--- INSTINCTS (Learned Behaviors) ---

Architecture:
  [0.7] When importing modules -> use absolute paths from project root

Testing:
  [0.8] When tests fail with timeout errors -> increase test timeout
  [0.6] When writing integration tests -> mock external HTTP calls
```

### midden-recent-failures API (from aether-utils.sh line 9310)
```bash
# Get 5 most recent failures
bash .aether/aether-utils.sh midden-recent-failures 5

# Returns: {"ok":true,"result":{"count":3,"failures":[{"category":"...","message":"...","source":"...","timestamp":"..."}]}}
```

### Test Pattern for Instinct Pipeline (based on learning-pipeline.test.js)
```javascript
const test = require('ava');

test.serial('instinct-create stores instinct in COLONY_STATE.json', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    // Seed COLONY_STATE.json with memory.instincts: []

    const result = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when X happens',
      '--action', 'do Y instead',
      '--confidence', '0.7',
      '--domain', 'testing',
      '--source', 'phase-1',
      '--evidence', 'observed pattern'
    ]);

    const json = JSON.parse(result);
    t.true(json.ok);
    t.is(json.result.action, 'created');
    t.is(json.result.confidence, 0.7);

    // Verify in state file
    const state = JSON.parse(fs.readFileSync(path.join(tmpDir, '.aether', 'data', 'COLONY_STATE.json')));
    t.is(state.memory.instincts.length, 1);
    t.is(state.memory.instincts[0].trigger, 'when X happens');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
```

## State of the Art

| Current State | Target State | What Changes | Impact |
|---------------|-------------|--------------|--------|
| continue-advance Step 3 creates instincts at 0.4-0.7 confidence | Minimum 0.7 confidence for auto-creation | Threshold enforcement in playbook text | Only high-confidence patterns become instincts |
| Instincts sourced only from phase learnings | Three sources: learnings, midden errors, success patterns | Add midden query + pattern extraction in Step 3 | Richer instinct pool from error history |
| Instincts listed flat by confidence in prompt | Grouped by domain (testing, architecture, etc.) | jq group_by in pheromone-prime | Clearer builder guidance |
| Instinct creation is silent | Visible in continue output | Display block in continue-finalize.md Step 3 | User sees colony learning in action |
| instinct-read has fallthrough bug | Clean single-JSON output | Add exit 0 after early return | Prevents double-output parsing issues |

## Open Questions

1. **Conflict resolution between REDIRECT pheromones and instincts**
   - What we know: User decided "highest confidence signal wins regardless of source." Pheromone effective_strength uses decay (line 7327: `strength * (1 - elapsed/decay_days)`). Instinct confidence is static or only boosted.
   - What's unclear: pheromone-prime currently renders instincts and signals as separate sections. There is no runtime "conflict detection" that removes an instinct when a REDIRECT contradicts it. The sections are both injected and the LLM builder is expected to weigh them.
   - Recommendation: For Phase 1, the current approach (both sections rendered, builder uses judgment) is sufficient. The user decision says "highest confidence wins" -- since both confidence values are visible in the prompt, the builder can apply this rule. No programmatic conflict resolution needed in Phase 1 unless the user explicitly requests it. Document this as a convention in the instinct section header text.

2. **Success pattern sourcing**
   - What we know: User wants "success patterns" as a third source alongside learnings and midden. continue-advance.md Step 2 extracts learnings with a `status` field (hypothesis/validated/disproven).
   - What's unclear: What specifically constitutes a "success pattern" vs a "learning"? Validated learnings with `tested: true` and `status: validated` are the closest match in the current data model.
   - Recommendation: Treat validated learnings (status=validated, tested=true) as the "success pattern" source. These are patterns that were hypothesized AND confirmed working. They should be promoted to instincts with higher base confidence (0.8) than learnings-based instincts (0.7).

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 7084-7269: instinct-read and instinct-create subcommand implementations
- `.aether/aether-utils.sh` lines 7272-7436: pheromone-prime subcommand (instinct rendering)
- `.aether/aether-utils.sh` lines 7438-7658: colony-prime subcommand (unified priming chain)
- `.aether/docs/command-playbooks/continue-advance.md` lines 82-117: existing instinct creation flow
- `.aether/docs/command-playbooks/build-context.md` lines 1-34: colony-prime call and prompt_section injection
- `.aether/docs/command-playbooks/build-wave.md` line 319: `{ prompt_section }` placeholder in builder prompts
- `tests/integration/learning-pipeline.test.js`: end-to-end pipeline test patterns
- `tests/unit/context-continuity.test.js`: pheromone-prime test and fixture setup

### Secondary (MEDIUM confidence)
- `tests/e2e/test-pher.sh` lines 277-325: PHER-05 instinct-read + pheromone-prime integration verification
- `.aether/templates/colony-state.template.json`: COLONY_STATE.json structure with memory.instincts

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all code paths traced through source, subcommands verified with line numbers
- Architecture: HIGH - file touch map derived from actual import chain (colony-prime -> pheromone-prime -> instinct-read)
- Pitfalls: HIGH - instinct-read fallthrough bug verified in source code, confidence threshold mismatch confirmed against success criteria

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (stable codebase, internal tooling)
