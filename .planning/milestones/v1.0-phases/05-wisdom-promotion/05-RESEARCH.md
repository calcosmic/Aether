# Phase 5: Wisdom Promotion - Research

**Researched:** 2026-03-07
**Domain:** Bash shell scripting (aether-utils.sh), continue-advance/continue-full playbook wiring, seal.md command, colony-prime prompt assembly, QUEEN.md wisdom pipeline
**Confidence:** HIGH

## Summary

Phase 5 closes the loop on the Aether learning lifecycle by wiring three specific integration points: (1) `learning-promote-auto` is called during continue to auto-promote observations that meet recurrence thresholds, (2) `queen-promote` is called during seal to graduate qualifying observations before colony archival, and (3) `queen-read` output is included in `colony-prime`'s `prompt_section` so builders receive accumulated wisdom.

The critical finding is that MOST of this wiring already exists. `colony-prime` (line 7451 of aether-utils.sh) already extracts QUEEN.md wisdom and renders it as a "QUEEN WISDOM (Eternal Guidance)" section in its `prompt_section` output -- satisfying QUEEN-03 and success criteria #3 and #4 out of the box. The continue flow (continue-advance.md Step 2.5) already calls `memory-capture`, which internally calls `learning-promote-auto` with "auto" thresholds (philosophy: 3, pattern: 2, etc.) -- partially satisfying QUEEN-01. Additionally, continue-advance.md Step 2.1.5 already runs `learning-check-promotion` + `learning-approve-proposals` to present proposals to the user. The seal.md command Step 3.6 also runs `learning-check-promotion` + `learning-approve-proposals` for a wisdom review before sealing.

The remaining gaps are subtle but real: (A) the requirement QUEEN-01 says "continue-finalize calls learning-promote-auto" but the current continue flow calls it through `memory-capture` in continue-advance Step 2.5, not in continue-finalize -- the planner must determine whether this counts as satisfied or needs explicit wiring in continue-finalize; (B) QUEEN-02 says "seal.md calls queen-promote for observations meeting thresholds" but seal.md currently uses the interactive approval workflow (`learning-approve-proposals`) rather than directly calling `queen-promote` for threshold-meeting observations; (C) success criteria #1 says "Running /ant:continue on a colony with observations meeting promotion thresholds creates entries in QUEEN.md" which IS satisfied by the existing `memory-capture` -> `learning-promote-auto` pipeline, but only for observations meeting the HIGHER "auto" thresholds (2+ for patterns, 3+ for philosophy). The LOWER "propose" thresholds (all 1+) only trigger the interactive approval workflow.

**Primary recommendation:** Verify that existing wiring satisfies each success criterion by running end-to-end tests. For any gaps, make targeted additions to the continue and seal playbooks rather than restructuring what already works. The test plan (05-04) is the highest-value deliverable -- it proves the pipeline works end-to-end.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| QUEEN-01 | continue-finalize calls learning-promote-auto to check promotion thresholds | `memory-capture` in continue-advance Step 2.5 already calls `learning-promote-auto` internally (line 5398). Step 2.1.5 runs `learning-check-promotion` + `learning-approve-proposals` for lower-threshold proposals. Gap: the call is in continue-advance, not continue-finalize. Need to either (a) add explicit `learning-promote-auto` call in continue-finalize, or (b) document that continue-advance satisfies this. |
| QUEEN-02 | seal.md calls queen-promote for observations meeting thresholds | seal.md Step 3.6 calls `learning-check-promotion` + `learning-approve-proposals` which internally uses `queen-promote`. The approval is interactive (user-driven), not automatic. Gap: the requirement says seal.md calls `queen-promote` directly. Need to either (a) add a batch `queen-promote` call for auto-threshold observations, or (b) verify the existing interactive path satisfies the intent. |
| QUEEN-03 | queen-read output included in colony-prime prompt_section for builder context | ALREADY SATISFIED. `colony-prime` (line 7451-7836) reads QUEEN.md, extracts all wisdom sections, and renders them as "QUEEN WISDOM (Eternal Guidance)" in `prompt_section`. Verified by running `colony-prime --compact` and confirming wisdom appears in output. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| aether-utils.sh | ~9,808 lines | `learning-promote-auto`, `queen-promote`, `queen-read`, `colony-prime`, `memory-capture`, `learning-check-promotion`, `learning-approve-proposals` subcommands | Single source of truth for all state operations |
| jq | System-installed | JSON manipulation for COLONY_STATE.json, learning-observations.json, pheromones.json | Used throughout aether-utils.sh for all state reads |
| ava | Installed in package.json | Integration test runner | Project standard, Phases 1-4 tests all use ava |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| continue-advance.md | Playbook | Step 2.5 (memory-capture + learning-promote-auto), Step 2.1.5 (proposal check + approval) | May need minor modifications for explicit QUEEN-01 wiring |
| continue-full.md | Playbook (monolithic) | Contains same steps as continue-advance.md | Must mirror any changes made to continue-advance.md |
| continue-finalize.md | Playbook | Steps 2.2-2.7 (handoff, changelog, commit, context clear, completion display) | Candidate for QUEEN-01 explicit wiring if continue-advance isn't sufficient |
| seal.md | Command | Step 3.6 wisdom approval | May need modification for QUEEN-02 direct promotion |
| colony-prime | aether-utils.sh subcommand (line 7451) | Unified priming (wisdom + signals + instincts + learnings + decisions + blockers) | Already reads QUEEN.md -- NO changes needed for QUEEN-03 |
| learning-pipeline.test.js | Integration test file | Existing tests for observe -> check -> promote -> prime pipeline | Extend or verify, don't rewrite |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Adding explicit `learning-promote-auto` call in continue-finalize | Relying on existing `memory-capture` call in continue-advance | The existing call through `memory-capture` already works. Adding a second call in continue-finalize risks double-promotion (same learning promoted twice). Recommendation: verify existing path, add guard if adding explicit call |
| Direct `queen-promote` in seal.md | Keeping interactive `learning-approve-proposals` workflow | The interactive workflow gives users control but doesn't fulfill "calls queen-promote" literally. Could add a separate batch auto-promotion step before the interactive review. |

## Architecture Patterns

### Current Data Flow (Already Wired)

```
continue-advance.md Step 2.5:
  memory-capture(learning)
    -> learning-observe (record in learning-observations.json)
    -> pheromone-write (emit FEEDBACK)
    -> learning-promote-auto (check AUTO thresholds: philosophy=3, pattern=2, etc.)
       -> IF threshold met: queen-promote -> QUEEN.md updated

continue-advance.md Step 2.1.5:
  learning-check-promotion (check PROPOSE thresholds: all=1, decree=0)
    -> IF proposals found: learning-approve-proposals (interactive)
       -> IF user approves: queen-promote -> QUEEN.md updated

seal.md Step 3.6:
  learning-check-promotion (check PROPOSE thresholds)
    -> IF proposals found: learning-approve-proposals (interactive)
       -> IF user approves: queen-promote -> QUEEN.md updated

colony-prime (called by build-context.md):
  queen-read -> extract QUEEN.md sections
  pheromone-prime -> extract signals + instincts
  context-capsule -> compact context
  learning extraction -> previous phase learnings
  decision extraction -> CONTEXT.md decisions
  blocker extraction -> flags.json blockers
  -> COMBINED prompt_section output with "QUEEN WISDOM" section
```

### File Touch Map

```
LIKELY NO CHANGES:
  .aether/aether-utils.sh                                # All subcommands already exist and work
  .aether/docs/command-playbooks/build-context.md         # Already calls colony-prime, injects prompt_section
  .aether/docs/command-playbooks/build-wave.md            # Already injects { prompt_section } into builder prompts

POSSIBLE MODIFICATIONS (depends on gap analysis):
  .aether/docs/command-playbooks/continue-advance.md      # Step 2.5 or new step for explicit QUEEN-01 call
  .aether/docs/command-playbooks/continue-full.md         # Mirror any continue-advance changes
  .aether/docs/command-playbooks/continue-finalize.md     # Candidate for explicit QUEEN-01 wiring
  .claude/commands/ant/seal.md                            # Step 3.6 for QUEEN-02 direct promotion

CREATE:
  tests/integration/wisdom-promotion.test.js              # End-to-end wisdom promotion + injection tests
```

### Pattern 1: Threshold Policy (Two-Tier System)

**What:** Wisdom promotion uses TWO threshold tiers:
- **"propose" thresholds** (lower): philosophy=1, pattern=1, redirect=1, stack=1, decree=0, failure=1. Used by `learning-check-promotion` and `queen-promote`. Determines when a learning QUALIFIES for promotion.
- **"auto" thresholds** (higher): philosophy=3, pattern=2, redirect=2, stack=2, decree=0, failure=2. Used by `learning-promote-auto`. Determines when a learning is auto-promoted WITHOUT user approval.

**When to use:** The auto tier is for recurrence-validated patterns that don't need human review. The propose tier is for the interactive `learning-approve-proposals` workflow where the user approves/rejects.

**Source:** `get_wisdom_threshold()` function at line 938 of aether-utils.sh.

### Pattern 2: Promotion Guard (Idempotency)

**What:** Both `learning-promote-auto` and `queen-promote` include guards against double-promotion:
- `learning-promote-auto` (line 5282): `grep -Fq -- "$content" "$queen_file"` -- checks if content already exists in QUEEN.md
- `queen-promote` (line 4796-4817): Checks observation count against threshold in learning-observations.json

**When to use:** Any new promotion wiring MUST rely on these existing guards. Never add a raw `queen-promote` call without the upstream `learning-observe` observation tracking.

### Pattern 3: Playbook Mirroring

**What:** continue-advance.md and continue-full.md must stay in sync. continue-full.md is the monolithic version containing all steps. continue-advance.md is the split playbook containing Steps 2-2.1.5.

**When to use:** Any change to continue-advance.md must be mirrored in continue-full.md. This was established in Phase 4 and must be maintained.

### Anti-Patterns to Avoid

- **Adding queen-promote calls without observation tracking:** `queen-promote` validates against `learning-observations.json`. Calling it for content that was never `learning-observe`'d will fail with "No observations found for this content." Always ensure the observation is recorded first.
- **Calling learning-promote-auto twice for the same learning:** If memory-capture in Step 2.5 already calls learning-promote-auto, adding another explicit call risks double-processing. Use the idempotency guard (grep check) or verify the call path doesn't overlap.
- **Modifying colony-prime for QUEEN-03:** colony-prime already renders queen wisdom. Any modification to the wisdom section risks breaking the tested format. QUEEN-03 is satisfied -- leave colony-prime alone.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wisdom promotion to QUEEN.md | Custom QUEEN.md parser/writer | `queen-promote` subcommand (line 4764) | Handles section finding, entry formatting, evolution log update, metadata stats update, temp file for atomic write |
| Observation threshold checking | Custom threshold logic | `learning-check-promotion` (line 5195) + `get_wisdom_threshold()` (line 938) | Centralized threshold policy, returns proposals with ready flag |
| Auto-promotion with recurrence guard | Custom observation counting | `learning-promote-auto` (line 5242) | Checks auto thresholds, content deduplication via grep, delegates to queen-promote |
| QUEEN.md reading for prompts | Custom QUEEN.md parser | `queen-read` subcommand (line 4327) | Two-level loading (global + local), section extraction, metadata parsing, priming flags |
| Combined prompt assembly | Custom prompt builder | `colony-prime` subcommand (line 7451) | Unified wisdom + signals + instincts + learnings + decisions + blockers assembly |

**Key insight:** All five subcommands needed for wisdom promotion already exist and are tested. The work is purely about ensuring they are CALLED at the right lifecycle points (continue and seal), not about building new capabilities.

## Common Pitfalls

### Pitfall 1: Double-Promotion Through Overlapping Call Paths
**What goes wrong:** Adding an explicit `learning-promote-auto` call in continue-finalize while memory-capture in continue-advance Step 2.5 already calls it internally.
**Why it happens:** The memory-capture -> learning-promote-auto chain is non-obvious (it's inside the subcommand, not visible in the playbook).
**How to avoid:** If adding an explicit call, use a different wisdom_type or check auto_promoted flag from memory-capture result before calling again. Or verify the existing path is sufficient and skip the explicit call.
**Warning signs:** Same content appears twice in QUEEN.md Patterns section.

### Pitfall 2: Seal Promotion Before Observation Recording
**What goes wrong:** Seal.md calls queen-promote for learnings that were never passed through learning-observe, causing "No observations found" validation errors.
**Why it happens:** Not all colony learnings go through the memory-capture pipeline -- some are extracted during continue-advance Step 2 and stored only in COLONY_STATE.json memory.phase_learnings without learning-observe.
**How to avoid:** Before promoting in seal, ensure each learning has been recorded via learning-observe. Either batch-observe all pending learnings first, or use the existing learning-approve-proposals workflow which checks learning-observations.json.
**Warning signs:** queen-promote returns E_VALIDATION_FAILED with "No observations found for this content."

### Pitfall 3: Empty QUEEN.md Sections in Prompt
**What goes wrong:** colony-prime outputs "QUEEN WISDOM (Eternal Guidance)" section header even when all sections are empty/placeholder text, bloating the prompt.
**Why it happens:** The section rendering checks for non-empty strings but placeholder text like "No philosophies recorded yet" counts as non-empty.
**How to avoid:** colony-prime already handles this at lines 7606-7621 by checking `!= "null"`. But for a pristine QUEEN.md, placeholder text may still appear. Verify by testing with a fresh QUEEN.md template. Note: this is a pre-existing behavior, not introduced by Phase 5. The current global QUEEN.md has content, masking this edge case.
**Warning signs:** Builder prompts contain placeholder text from QUEEN.md sections.

### Pitfall 4: Forgetting continue-full.md Mirror
**What goes wrong:** Changes to continue-advance.md are not mirrored to continue-full.md, causing the monolithic playbook to diverge.
**Why it happens:** continue-full.md is the comprehensive single-file version used by some commands; continue-advance.md is the split version. Both must contain the same logic.
**How to avoid:** Every change to continue-advance.md must be copied to the corresponding section in continue-full.md. Test both playbooks.
**Warning signs:** /ant:continue behaves differently depending on which playbook is loaded.

### Pitfall 5: Interactive Approval in Seal Blocks Colony Sealing
**What goes wrong:** learning-approve-proposals in seal Step 3.6 uses `read -r choice` for interactive input, which can block if run in a non-interactive context or if the user doesn't understand the prompt.
**Why it happens:** The approval workflow was designed for interactive terminal use.
**How to avoid:** This is existing behavior and is acceptable for seal (which is always interactive). But if adding a batch auto-promotion step, use `--yes` flag or bypass the interactive workflow entirely for auto-threshold observations.
**Warning signs:** Seal ceremony hangs waiting for input.

## Code Examples

Verified patterns from the codebase:

### Existing memory-capture -> learning-promote-auto Chain (continue-advance Step 2.5)
```bash
# Source: .aether/docs/command-playbooks/continue-advance.md, Step 2.5
# This is the current call path that satisfies QUEEN-01 partially

colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

current_phase_learnings=$(jq -r --argjson phase "$current_phase" '.memory.phase_learnings[] | select(.phase == $phase)' .aether/data/COLONY_STATE.json 2>/dev/null || echo "")

if [[ -n "$current_phase_learnings" ]]; then
  echo "$current_phase_learnings" | jq -r '.learnings[]?.claim // empty' 2>/dev/null | while read -r claim; do
    if [[ -n "$claim" ]]; then
      bash .aether/aether-utils.sh memory-capture "learning" "$claim" "pattern" "worker:continue" 2>/dev/null || true
    fi
  done
fi

# memory-capture internally calls:
#   1. learning-observe (records observation, increments count)
#   2. pheromone-write (emits FEEDBACK)
#   3. learning-promote-auto (checks AUTO thresholds, promotes if met)
```

### Existing Proposal Check + Approval (continue-advance Step 2.1.5)
```bash
# Source: .aether/docs/command-playbooks/continue-advance.md, Step 2.1.5

proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  verbose_flag=""
  [[ "$ARGUMENTS" == *"--verbose"* ]] && verbose_flag="--verbose"
  bash .aether/aether-utils.sh learning-approve-proposals $verbose_flag
fi
```

### Existing Seal Wisdom Review (seal.md Step 3.6)
```bash
# Source: .claude/commands/ant/seal.md, Step 3.6

proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  bash .aether/aether-utils.sh learning-approve-proposals
fi
```

### Direct queen-promote Call (for batch promotion in seal)
```bash
# Source: .aether/aether-utils.sh queen-promote subcommand (line 4764)
# Usage: queen-promote <type> <content> <colony_name>

bash .aether/aether-utils.sh queen-promote "pattern" "Always validate inputs" "my-colony"
# Returns: {"ok":true,"result":{"type":"pattern","content":"Always validate inputs",...}}
```

### colony-prime Wisdom Output (QUEEN-03 -- already working)
```bash
# Source: .aether/aether-utils.sh colony-prime subcommand (line 7451)
# The prompt_section output includes this block when QUEEN.md has entries:

# --- QUEEN WISDOM (Eternal Guidance) ---
#
# Philosophies:
# - **colony-a** (2026-02-15): Test-driven development ensures quality
#
# Patterns:
# - **colony-b** (2026-02-15): Always validate inputs
#
# Redirects (AVOID these):
# - **colony-c** (2026-02-15): Never skip security checks
#
# --- END QUEEN WISDOM ---
```

### Test Pattern (from Phase 1-4 integration tests)
```javascript
// Source: tests/integration/learning-pipeline.test.js pattern
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

async function setupTestColony(tmpDir) {
  const aetherDir = path.join(tmpDir, '.aether');
  const dataDir = path.join(aetherDir, 'data');
  await fs.promises.mkdir(dataDir, { recursive: true });

  // QUEEN.md template with METADATA on single line
  const isoDate = new Date().toISOString();
  const queenTemplate = `# QUEEN.md -- Colony Wisdom
> Last evolved: ${isoDate}
> Colonies contributed: 0
> Wisdom version: 1.0.0
---
## Philosophies
Core beliefs...
*No philosophies recorded yet*
---
## Patterns
Validated approaches...
*No patterns recorded yet*
...
<!-- METADATA {"version":"1.0.0","last_evolved":"${isoDate}","stats":{...}} -->`;

  await fs.promises.writeFile(path.join(aetherDir, 'QUEEN.md'), queenTemplate);
  await fs.promises.writeFile(
    path.join(dataDir, 'learning-observations.json'),
    JSON.stringify({ observations: [] }, null, 2)
  );
  // Also create COLONY_STATE.json with goal + phases for colony-prime
  // Also create pheromones.json for colony-prime
}
```

## State of the Art

| Current State | Gap | Required Change | Impact |
|--------------|-----|-----------------|--------|
| `memory-capture` calls `learning-promote-auto` in continue-advance Step 2.5 | Call is indirect (inside memory-capture), not in continue-finalize | Verify this path satisfies QUEEN-01, or add explicit call | Low -- existing path works |
| seal.md Step 3.6 uses interactive `learning-approve-proposals` | No direct `queen-promote` call for auto-threshold observations | Add batch promotion for observations meeting auto thresholds before interactive review | Medium -- adds automatic promotion path |
| colony-prime outputs "QUEEN WISDOM (Eternal Guidance)" section | Success criteria says "Colony Wisdom" section name | Verify the existing section name satisfies the intent (naming is cosmetic) | None -- section exists |
| Existing tests cover observe -> check -> promote -> prime pipeline | No tests verify the end-to-end flow through continue -> seal -> build | Add integration tests for the complete lifecycle | High -- proves the wiring works |

**What keeps working unchanged:**
- colony-prime already reads QUEEN.md and renders wisdom in prompt_section
- build-context.md already calls colony-prime and injects prompt_section into builder prompts
- build-wave.md already passes `{ prompt_section }` into builder prompt templates
- learning-promote-auto already checks for duplicate content in QUEEN.md before promoting
- queen-promote already validates observation counts against thresholds

## Open Questions

1. **Does memory-capture in continue-advance Step 2.5 satisfy QUEEN-01?**
   - What we know: `memory-capture` calls `learning-promote-auto` internally (line 5398). The call is in continue-advance.md, not continue-finalize.md. The requirement text says "continue-finalize calls learning-promote-auto."
   - What's unclear: Is the requirement specifying the exact playbook file, or the continue lifecycle broadly? The current wiring IS in the continue flow, just in a different split file.
   - Recommendation: The intent is "during /ant:continue, observations meeting promotion thresholds get promoted." This is already happening through memory-capture. Add a comment in continue-finalize acknowledging the promotion happens in continue-advance Step 2.5. If the planner decides explicit wiring is needed in continue-finalize, add a guard to prevent double-promotion.

2. **Should seal.md promote automatically or only through interactive approval?**
   - What we know: seal.md Step 3.6 uses `learning-approve-proposals` (interactive). The requirement says "seal.md calls queen-promote for observations meeting thresholds."
   - What's unclear: Should observations meeting auto thresholds be promoted WITHOUT user approval during seal? Or is the interactive workflow sufficient?
   - Recommendation: Add a batch auto-promotion step BEFORE the interactive review in seal Step 3.6. For observations meeting the higher "auto" thresholds, auto-promote silently. For observations meeting only the lower "propose" thresholds, continue using the interactive approval. This gives both automatic promotion for well-established patterns and human review for newer observations.

3. **colony-prime section naming**
   - What we know: colony-prime outputs "QUEEN WISDOM (Eternal Guidance)". Success criteria #4 says 'colony-prime output includes a "Colony Wisdom" section.'
   - What's unclear: Is the exact label important?
   - Recommendation: "QUEEN WISDOM (Eternal Guidance)" is more descriptive and consistent with the existing codebase. Keep the current label. The success criteria checks for the PRESENCE of a wisdom section, not the exact label.

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 938-972 -- `get_wisdom_threshold()` and `get_wisdom_thresholds_json()` threshold policy
- `.aether/aether-utils.sh` lines 4280-4325 -- `queen-init` subcommand
- `.aether/aether-utils.sh` lines 4327-4478 -- `queen-read` subcommand with two-level loading
- `.aether/aether-utils.sh` lines 4764-4920 -- `queen-promote` subcommand with threshold validation
- `.aether/aether-utils.sh` lines 5057-5193 -- `learning-observe` subcommand for observation tracking
- `.aether/aether-utils.sh` lines 5195-5240 -- `learning-check-promotion` subcommand
- `.aether/aether-utils.sh` lines 5242-5309 -- `learning-promote-auto` subcommand with auto thresholds
- `.aether/aether-utils.sh` lines 5311-5409 -- `memory-capture` subcommand (calls learning-observe + pheromone-write + learning-promote-auto)
- `.aether/aether-utils.sh` lines 5844-5993 -- `learning-approve-proposals` interactive workflow
- `.aether/aether-utils.sh` lines 7451-7836 -- `colony-prime` subcommand (already renders QUEEN wisdom)
- `.aether/docs/command-playbooks/continue-advance.md` -- Step 2.5 (memory-capture calls), Step 2.1.5 (proposal check)
- `.aether/docs/command-playbooks/continue-full.md` -- Monolithic version with same steps
- `.aether/docs/command-playbooks/continue-finalize.md` -- Steps 2.2-2.7 (post-advancement)
- `.claude/commands/ant/seal.md` -- Step 3.6 wisdom approval, full seal ceremony
- `.aether/QUEEN.md` -- Current QUEEN.md file with existing wisdom entries
- `.aether/docs/QUEEN-SYSTEM.md` -- QUEEN system documentation
- `.aether/docs/queen-commands.md` -- Queen command reference
- `tests/integration/learning-pipeline.test.js` -- Existing end-to-end pipeline tests

### Secondary (MEDIUM confidence)
- `.aether/docs/command-playbooks/build-context.md` -- Step 4 colony-prime call and prompt_section injection
- `.aether/docs/command-playbooks/build-wave.md` -- { prompt_section } template variable in builder prompts
- `.planning/phases/04-pheromone-auto-emission/04-RESEARCH.md` -- Phase 4 research (auto-emission patterns, playbook mirroring requirements)
- `.planning/ROADMAP.md` -- Phase 5 description and success criteria
- `.planning/REQUIREMENTS.md` -- QUEEN-01, QUEEN-02, QUEEN-03 requirement definitions

### Tertiary (LOW confidence)
- None -- all findings verified against codebase source

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all subcommands verified in aether-utils.sh source code with exact line numbers
- Architecture: HIGH -- colony-prime already wires QUEEN wisdom into prompts (verified by running the command); continue and seal playbooks inspected directly
- Pitfalls: HIGH -- double-promotion, observation-before-promotion, and mirror requirements identified from Phase 4 precedent and code inspection
- Existing wiring coverage: HIGH -- verified that QUEEN-03 is already satisfied, QUEEN-01 is partially satisfied, QUEEN-02 is partially satisfied

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable domain -- bash playbook + existing subcommands)
