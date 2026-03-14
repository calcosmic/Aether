# Phase 14: Decision-Pheromone and Learning-Instinct Verification - Research

**Researched:** 2026-03-14
**Domain:** Bash shell (aether-utils.sh), playbook markdown (continue-advance.md), ava integration tests
**Confidence:** HIGH

## Summary

Phase 14 addresses two distinct bugs in the continue-advance playbook that prevent the colony learning loops from working as designed. DEC-01 is a format alignment issue where decisions are emitted as pheromones through two different code paths using inconsistent content formats, causing the deduplication check to potentially miss already-emitted signals. LRN-01 is a confidence scoring issue where instincts created from learnings always receive fixed confidence values (0.6 or 0.7) regardless of how many times the learning has been observed across colonies, even though the observation count data is available.

Both fixes are small in scope: DEC-01 requires aligning the pheromone content format between `context-update decision` and continue-advance Step 2.1b, plus tightening the dedup check. LRN-01 requires adding a recurrence-calibrated confidence formula to `instinct-create` or `learning-promote-auto` in aether-utils.sh. The test infrastructure (ava + integration test helpers) is well-established from Phase 4 and can be reused directly.

**Primary recommendation:** Fix both issues in aether-utils.sh bash code, update the continue-advance playbook instructions, and add targeted integration tests that verify the exact success criteria.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DEC-01 | Decision-to-pheromone dedup format alignment verified and fixed so auto-emitted decision pheromones are correctly deduplicated in continue-advance Step 2.1b | Research fully maps the two emission paths (`context-update decision` at line 508 and Step 2.1b in continue-advance.md), identifies the format divergence (`"Decision: X -- Y"` vs `"[decision] X"`), and traces the dedup jq query. See "DEC-01 Analysis" below. |
| LRN-01 | Instinct confidence uses recurrence-calibrated scoring based on observation_count from learning-observations.json rather than fixed 0.7 | Research identifies all instinct creation call sites (`learning-promote-auto` line 5384, continue-advance Steps 3/3a/3b), the available `observation_count` data in learning-observations.json, and the confidence formula requirements. See "LRN-01 Analysis" below. |
</phase_requirements>

## DEC-01 Analysis: Decision-Pheromone Format Alignment

### The Two Emission Paths

**Path A -- `context-update decision` (aether-utils.sh line 508):**
When a decision is recorded during a build, it auto-emits a pheromone:
```bash
bash "$0" pheromone-write FEEDBACK "Decision: $decision -- $rationale" \
  --strength 0.65 \
  --source "system:decision" \
  --reason "Auto-emitted from architectural decision" \
  --ttl "30d"
```
- Content format: `"Decision: {decision_text} -- {rationale}"`
- Source: `"system:decision"`
- Strength: `0.65`

**Path B -- continue-advance Step 2.1b:**
When continue runs, it extracts decisions from CONTEXT.md and emits pheromones:
```bash
bash .aether/aether-utils.sh pheromone-write FEEDBACK \
  "[decision] $dec" \
  --strength 0.6 \
  --source "auto:decision" \
  --reason "Auto-emitted from phase decision during continue" \
  --ttl "30d"
```
- Content format: `"[decision] {decision_text}"`
- Source: `"auto:decision"`
- Strength: `0.6`

### The Dedup Check

Step 2.1b runs this jq query before emitting:
```bash
existing=$(jq -r --arg text "$dec" '
  [.signals[] | select(.active == true and (.source == "auto:decision" or .source == "system:decision") and (.content.text | contains($text)))] | length
' .aether/data/pheromones.json 2>/dev/null || echo "0")
```

Where `$dec` is the raw decision text extracted from the CONTEXT.md "Decision" column (e.g., `"Remove /ant:export commands"`).

### Format Mismatch Scenarios

| Scenario | Already in pheromones.json | `$dec` value | `contains($dec)` result | Dedup works? |
|----------|---------------------------|--------------|------------------------|-------------|
| Path A emitted first | `"Decision: Remove X -- because Y"` | `"Remove X"` | true | YES |
| Path B emitted first | `"[decision] Remove X"` | `"Remove X"` | true | YES |
| Decision has special chars | `"Decision: Use 'awk' for parsing"` | `"Use 'awk' for parsing"` | true | YES |

**Finding:** The `contains()` substring check technically works in most cases because the raw decision text is a substring of both formats. However, the formats are inconsistent and the dedup is fragile:

1. **Inconsistent signal format** -- Workers reading pheromones see `"Decision: X -- Y"` from some decisions and `"[decision] X"` from others, making pattern recognition harder.
2. **Strength divergence** -- Path A uses 0.65, Path B uses 0.6. Minor but unnecessary inconsistency.
3. **Potential `contains()` false positives** -- If a short decision text like `"Use X"` is a substring of a different decision `"Don't Use X for Y"`, `contains()` would incorrectly match. An exact-match or normalized match would be more robust.

### Recommended Fix

**Align to one format:** Use `"[decision] {text}"` consistently across both paths (Step 2.1b's format is cleaner and more parseable). Update `context-update decision` (line 508) to match. Also align source to `"auto:decision"` from both paths (or keep `"system:decision"` for real-time emission and just check both in dedup, which it already does).

**Files to edit:**
- `.aether/aether-utils.sh` -- `context-update decision` handler (~line 508)
- `.aether/docs/command-playbooks/continue-advance.md` -- Step 2.1b dedup (if tightening needed)

**Confidence:** HIGH -- Both code paths are fully traced and the fix is straightforward string format alignment.

## LRN-01 Analysis: Recurrence-Calibrated Instinct Confidence

### Current Behavior

Instincts are created with fixed confidence values at three call sites:

| Call Site | Location | Fixed Confidence | observation_count available? |
|-----------|----------|-----------------|----------------------------|
| learning-promote-auto | aether-utils.sh line 5384 | 0.6 | YES (line 5356) |
| Step 3 (phase patterns) | continue-advance.md line 91 | 0.7 | NO (agent decision) |
| Step 3a (midden errors) | continue-advance.md line 122 | 0.8 | NO (agent decision) |
| Step 3b (success patterns) | continue-advance.md line 140 | 0.7 | NO (agent decision) |

### Required Behavior

From success criteria:
- observation_count=1 -> confidence 0.7
- Each additional observation increases confidence
- Cap at 0.9

### Recommended Formula

```
confidence = min(0.7 + (observation_count - 1) * 0.05, 0.9)
```

| observation_count | confidence |
|-------------------|------------|
| 1 | 0.70 |
| 2 | 0.75 |
| 3 | 0.80 |
| 4 | 0.85 |
| 5+ | 0.90 (cap) |

### Where to Implement

**Option A: Modify `instinct-create` to auto-calibrate** -- Add an optional `--recurrence-calibrate` flag or automatic lookup of learning-observations.json. This would make all callers benefit automatically.

**Option B: Modify `learning-promote-auto` only** -- Since this is the main automated path, fix it here. The playbook calls in Steps 3/3a/3b are agent-driven and the agent already has discretion over confidence values (the guidelines say "0.7-0.9 based on evidence strength").

**Recommended: Option B with a supporting helper.** The primary automated path (`learning-promote-auto`) should use the formula, since it already has `observation_count` at line 5356. For the playbook steps, add a helper subcommand (e.g., `learning-confidence`) that agents can call to get a calibrated confidence from observation count, and update the playbook instructions to use it.

However, the simplest approach that satisfies all 4 success criteria is:

1. In `learning-promote-auto` (line 5384): Replace `--confidence 0.6` with a computed value based on `observation_count`.
2. In the continue-advance playbook Steps 3/3b: Update the instructions to tell agents to look up observation count via `learning-observe` (which returns it) before calling `instinct-create`, and apply the formula.

**But the cleanest single-point fix:** Add the formula to `instinct-create` itself behind an optional `--from-learning "content text"` flag that makes it look up the observation count and compute confidence automatically. This way:
- `learning-promote-auto` passes `--from-learning "$content"` instead of `--confidence 0.6`
- Playbook agents can pass `--from-learning "$claim"` to get auto-calibrated confidence
- Existing callers that pass `--confidence` explicitly still work (backward compatible)

**Files to edit:**
- `.aether/aether-utils.sh` -- `instinct-create` handler (add observation-based confidence)
- `.aether/aether-utils.sh` -- `learning-promote-auto` handler (use new flag or compute inline)
- `.aether/docs/command-playbooks/continue-advance.md` -- Steps 3/3b instructions (update confidence guidelines)

**Confidence:** HIGH -- The observation_count data is already collected and stored, the formula is straightforward, and the instinct-create handler is well-structured for modification.

## Architecture Patterns

### Playbook Files Touched (No Overlap with Phase 13)

Phase 13 edits: `build-wave.md`, `continue-verify.md`
Phase 14 edits: `continue-advance.md` (and `aether-utils.sh` for DEC-01 and LRN-01)

Both phases touch `aether-utils.sh` but in different functions:
- Phase 13: `midden-write` and related midden functions
- Phase 14: `context-update decision`, `instinct-create`, `learning-promote-auto`

No function overlap. Safe to parallelize.

### Test Pattern (Reusable from Phase 4)

Existing integration tests in `tests/integration/pheromone-auto-emission.test.js` and `tests/integration/instinct-pipeline.test.js` provide a well-established pattern:

1. Create temp directory
2. Set up test colony with `setupTestColony(tmpDir, opts)`
3. Run `aether-utils.sh` commands via `runAetherUtil(tmpDir, command, args)`
4. Assert on output JSON and file contents
5. Clean up temp directory

**Important:** Integration tests live in `tests/integration/` but the ava config only runs `tests/unit/**/*.test.js`. Integration tests must be run explicitly or the ava config must be updated. For Phase 14 tests, add them as unit tests in `tests/unit/` (following the same helper pattern) so they run with `npm test`.

### Recommended Project Structure for Changes

```
.aether/
  aether-utils.sh           # Fix context-update decision format (DEC-01)
                             # Fix instinct-create confidence calibration (LRN-01)
                             # Fix learning-promote-auto confidence (LRN-01)

.aether/docs/command-playbooks/
  continue-advance.md        # Update Step 2.1b dedup (DEC-01)
                             # Update Steps 3/3b confidence guidelines (LRN-01)

tests/unit/
  decision-dedup.test.js     # NEW: DEC-01 verification tests
  instinct-confidence.test.js # NEW: LRN-01 verification tests
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Dedup logic | Custom dedup in every emission path | Single dedup check in `pheromone-write` or a `pheromone-dedup` helper | Dedup logic in the playbook (bash in markdown) is fragile and untestable |
| Confidence formula | Inline math in every call site | Centralized formula in `instinct-create` | One formula, many callers -- centralize to avoid drift |
| Observation count lookup | Repeated jq calls across call sites | Use `learning-observe` return value or `instinct-create --from-learning` | Data is already available; avoid redundant file reads |

## Common Pitfalls

### Pitfall 1: Floating Point Arithmetic in Bash
**What goes wrong:** Bash does not support floating point arithmetic. `$(( 0.7 + 0.05 ))` fails.
**Why it happens:** The confidence formula requires decimal math.
**How to avoid:** Use `awk` or `bc` for floating point calculations:
```bash
confidence=$(awk "BEGIN { printf \"%.2f\", 0.7 + ($obs_count - 1) * 0.05 }" )
# Or use jq:
confidence=$(echo "$obs_count" | jq --argjson count "$obs_count" 'null | [0.7 + ($count - 1) * 0.05, 0.9] | min')
```
**Warning signs:** Syntax errors when running `instinct-create` with computed confidence.

### Pitfall 2: jq `contains()` False Positives
**What goes wrong:** Short decision text matches unrelated decisions as substrings.
**Why it happens:** `contains("use")` matches `"Don't use X"`, `"Use Y for Z"`, etc.
**How to avoid:** Normalize decision text before comparison. Consider exact match on a hash or use `startswith("[decision]")` combined with the decision text.
**Warning signs:** Dedup incorrectly preventing new decision pheromones.

### Pitfall 3: Atomic Write Ordering
**What goes wrong:** Multiple pheromone writes from the same continue run corrupt pheromones.json.
**Why it happens:** Step 2.1b loops over decisions and calls `pheromone-write` for each. Each call reads/writes the file.
**How to avoid:** `pheromone-write` already uses `acquire_lock` and `atomic_write`, so this is handled. But verify the lock works correctly in rapid succession.
**Warning signs:** Missing signals, truncated JSON.

### Pitfall 4: Test Isolation
**What goes wrong:** Tests depend on system-level COLONY_STATE.json instead of temp directory.
**Why it happens:** AETHER_ROOT or DATA_DIR not properly set in test environment.
**How to avoid:** Always pass `AETHER_ROOT` and `DATA_DIR` environment variables pointing to temp dir (the existing test helpers already do this).
**Warning signs:** Tests modify the real .aether/data/ directory.

## Code Examples

### DEC-01: Aligned Format for context-update decision

Current (aether-utils.sh line 508):
```bash
bash "$0" pheromone-write FEEDBACK "Decision: $decision -- $rationale" \
  --strength 0.65 \
  --source "system:decision" \
  --reason "Auto-emitted from architectural decision" \
  --ttl "30d" 2>/dev/null || true
```

Fixed:
```bash
bash "$0" pheromone-write FEEDBACK "[decision] $decision" \
  --strength 0.6 \
  --source "auto:decision" \
  --reason "Auto-emitted from architectural decision" \
  --ttl "30d" 2>/dev/null || true
```

This aligns: content format (`[decision] X`), strength (0.6), and source (`auto:decision`) with Step 2.1b. The dedup in Step 2.1b already checks for `auto:decision` source, so switching from `system:decision` to `auto:decision` is safe. If keeping `system:decision` is preferred (to distinguish real-time vs batch emission), the dedup already checks both sources.

### LRN-01: Recurrence-Calibrated Confidence in instinct-create

Add to `instinct-create` handler (after parsing arguments, before creating the instinct):
```bash
# LRN-01: If --from-learning flag is set, compute confidence from observation count
if [[ -n "$ic_from_learning" ]]; then
  obs_file="$DATA_DIR/learning-observations.json"
  if [[ -f "$obs_file" ]]; then
    lrn_hash="sha256:$(echo -n "$ic_from_learning" | sha256sum | cut -d' ' -f1)"
    lrn_obs_count=$(jq -r --arg hash "$lrn_hash" \
      '.observations[]? | select(.content_hash == $hash) | .observation_count // 1' \
      "$obs_file" 2>/dev/null | head -1)
    [[ -z "$lrn_obs_count" ]] && lrn_obs_count=1
    # Formula: 0.7 + (count-1)*0.05, capped at 0.9
    ic_confidence=$(awk -v c="$lrn_obs_count" 'BEGIN {
      v = 0.7 + (c - 1) * 0.05
      if (v > 0.9) v = 0.9
      printf "%.2f", v
    }')
  fi
fi
```

Update `learning-promote-auto` (line 5381-5387):
```bash
# Compute recurrence-calibrated confidence
lp_confidence=$(awk -v c="$observation_count" 'BEGIN {
  v = 0.7 + (c - 1) * 0.05
  if (v > 0.9) v = 0.9
  printf "%.2f", v
}')

bash "$0" instinct-create \
  --trigger "When working on $wisdom_type patterns" \
  --action "$content" \
  --confidence "$lp_confidence" \
  --domain "$wisdom_type" \
  --source "promoted_from_learning" \
  --evidence "Auto-promoted after $observation_count observations" 2>/dev/null || true
```

### Test Example: DEC-01 Dedup Verification

```javascript
test.serial('dedup catches system:decision pheromone when auto:decision would duplicate', async (t) => {
  const tmpDir = await createTempDir();
  try {
    // Pre-populate with a system:decision signal (emitted by context-update decision)
    await setupTestColony(tmpDir, {
      pheromoneSignals: [{
        id: 'sig_feedback_sys_001',
        type: 'FEEDBACK',
        priority: 'low',
        source: 'auto:decision',  // aligned source
        created_at: new Date().toISOString(),
        expires_at: new Date(Date.now() + 30*86400000).toISOString(),
        active: true,
        strength: 0.6,
        content: { text: '[decision] Use awk for parsing' }
      }]
    });

    // Simulate the dedup check from Step 2.1b
    const dec = 'Use awk for parsing';
    const result = execSync(`jq -r --arg text "${dec}" '[.signals[] | select(.active == true and (.source == "auto:decision" or .source == "system:decision") and (.content.text | contains($text)))] | length' "${tmpDir}/.aether/data/pheromones.json"`, { encoding: 'utf8' }).trim();

    t.is(result, '1', 'Should find existing signal and prevent duplicate');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});
```

### Test Example: LRN-01 Recurrence-Calibrated Confidence

```javascript
test.serial('instinct confidence is 0.7 for observation_count=1', async (t) => {
  const tmpDir = await createTempDir();
  try {
    await setupTestColony(tmpDir);
    // Create a learning observation with count=1
    const obsFile = path.join(tmpDir, '.aether', 'data', 'learning-observations.json');
    const content = 'Always validate inputs before processing';
    const hash = crypto.createHash('sha256').update(content).digest('hex');
    await fs.promises.writeFile(obsFile, JSON.stringify({
      observations: [{
        content_hash: `sha256:${hash}`,
        content: content,
        wisdom_type: 'pattern',
        observation_count: 1,
        first_seen: new Date().toISOString(),
        last_seen: new Date().toISOString(),
        colonies: ['test-colony']
      }]
    }));

    // Call instinct-create with --from-learning
    const result = runAetherUtil(tmpDir, 'instinct-create', [
      '--trigger', 'when processing user input',
      '--action', content,
      '--from-learning', content,
      '--domain', 'architecture',
      '--source', 'test',
      '--evidence', 'test'
    ]);

    const json = JSON.parse(result);
    t.is(json.result.confidence, 0.7, 'Should be 0.7 for observation_count=1');
  } finally {
    await cleanupTempDir(tmpDir);
  }
});

test.serial('instinct confidence increases with observation_count, capped at 0.9', async (t) => {
  // Similar test with observation_count=5 -> confidence should be 0.9
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No decision pheromones | Decisions auto-emit FEEDBACK pheromones | Phase 4 (v1.0, 2026-03-06) | Decisions visible to workers |
| Fixed instinct confidence | Confidence guidelines (0.7/0.8/0.9) | Phase 1 (v1.0, 2026-03-06) | Agent discretion, not data-driven |
| No learning observations | learning-observations.json with counts | Phase 5 (v1.0, 2026-03-07) | Recurrence tracking available |

**Key insight:** The observation_count infrastructure was built in Phase 5 for wisdom promotion thresholds, but was never wired into instinct confidence scoring. The data exists -- it just needs to be used.

## Open Questions

1. **Should `context-update decision` switch source to `auto:decision` or keep `system:decision`?**
   - What we know: Step 2.1b dedup already checks both sources. Using `auto:decision` everywhere simplifies the dedup and aligns formats.
   - What's unclear: Whether downstream consumers distinguish between real-time (`system:decision`) and batch (`auto:decision`) emission.
   - Recommendation: Switch to `auto:decision` everywhere for simplicity. The `reason` field already distinguishes the emission context. If the distinction matters, keep both sources but ensure format alignment.

2. **Should the `--from-learning` flag be added to `instinct-create`, or should the formula be computed inline in `learning-promote-auto`?**
   - What we know: Adding to `instinct-create` centralizes the logic. Computing inline in `learning-promote-auto` is simpler but means the playbook steps still use fixed values.
   - Recommendation: Compute inline in `learning-promote-auto` for the automated path. Update the playbook instructions to tell agents to use observation count when available. This is the minimal change that satisfies all success criteria.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` -- Full source code read of `context-update decision` (line 508), `instinct-create` (line 7252-7367), `learning-promote-auto` (line 5333-5400), `memory-capture` (line 5402-5504), `pheromone-write` (line 6774-6957), `learning-observe` (line 5160-5284)
- `.aether/docs/command-playbooks/continue-advance.md` -- Full playbook read, Steps 2, 2.1b, 3, 3a, 3b
- `.aether/data/learning-observations.json` -- Live data structure with 11 entries showing observation_count field
- `.aether/data/pheromones.json` -- Live pheromone data showing signal formats
- `tests/integration/pheromone-auto-emission.test.js` -- Existing test patterns for decision dedup
- `tests/integration/instinct-pipeline.test.js` -- Existing test patterns for instinct creation/boosting

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- DEC-01 and LRN-01 requirement definitions
- `.planning/ROADMAP.md` -- Phase dependency and scope information
- `.aether/CONTEXT.md` -- Live CONTEXT.md showing decision table format

## Metadata

**Confidence breakdown:**
- DEC-01 analysis: HIGH -- Both code paths fully traced, format mismatch documented with line numbers
- LRN-01 analysis: HIGH -- All instinct creation call sites identified, observation_count data verified in live file
- Architecture patterns: HIGH -- Direct code inspection of aether-utils.sh
- Pitfalls: HIGH -- Based on known bash arithmetic limitations and observed code patterns
- Test patterns: HIGH -- Existing test infrastructure examined and verified working (530 tests pass)

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (stable codebase, no external dependencies)
