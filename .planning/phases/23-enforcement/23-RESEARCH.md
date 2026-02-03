# Phase 23: Enforcement - Research

**Researched:** 2026-02-03
**Domain:** Shell enforcement gates for spawn limits and pheromone quality in the Aether ant colony system
**Confidence:** HIGH

## Summary

Phase 23 adds two new subcommands to `aether-utils.sh` (`spawn-check` and `pheromone-validate`) and wires them as hard enforcement gates into 6 worker specs and `continue.md`. It also adds a post-action validation checklist to all 6 worker specs. This is a targeted edit phase -- no new files, no new commands, no architectural changes.

The existing `aether-utils.sh` is 201 lines with 11 subcommands. Adding 2 new subcommands will add approximately 30-40 lines, staying well within the <300 line target. The current `COLONY_STATE.json` workers object uses simple string values (`"idle"`, `"active"`) for 6 named workers -- counting non-idle workers gives the active worker count. Spawn depth is NOT currently tracked in `COLONY_STATE.json`, which means the `spawn-check` subcommand needs to either (a) accept depth as a parameter passed by the spawning worker, or (b) add a `spawn_depth` field to the state. Option (a) is recommended because depth is a per-call-chain property, not a global state.

The 6 worker specs share an identical "You Can Spawn Other Ants" section (lines 156-244 in architect-ant.md, similar in others). The enforcement gate (`spawn-check`) must be inserted at the start of this section as a mandatory pre-spawn step. The existing "Spawn Confidence Check" (Bayesian advisory) becomes step 2 after the hard gate passes.

**Primary recommendation:** Execute as two plans: Plan 1 adds both subcommands to `aether-utils.sh` (ENFO-01, ENFO-03). Plan 2 wires the enforcement gates into worker specs and `continue.md` (ENFO-02, ENFO-04, ENFO-05).

## Standard Stack

No new libraries or tools needed. All work edits existing markdown and shell files.

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| aether-utils.sh | 0.1.0 | Central utility layer -- receives 2 new subcommands | Single entry point for all deterministic colony operations |
| jq | system | JSON processing within shell | Already used by all existing subcommands in aether-utils.sh |

### Supporting
None required -- this is pure edit work on existing files.

### Alternatives Considered
None -- the approach is fully specified in the requirements.

## Architecture Patterns

### Pattern 1: spawn-check Subcommand Design

**What:** A new `spawn-check` subcommand that reads `COLONY_STATE.json`, counts non-idle workers, accepts spawn depth as parameter, and returns pass/fail JSON.

**Interface:**
```bash
bash .aether/aether-utils.sh spawn-check <current_depth>
```

**Returns on pass:**
```json
{"ok":true,"result":{"pass":true,"active_workers":2,"max_workers":5,"current_depth":1,"max_depth":3}}
```

**Returns on fail:**
```json
{"ok":true,"result":{"pass":false,"reason":"worker_limit","active_workers":5,"max_workers":5,"current_depth":1,"max_depth":3}}
```

**Why depth is a parameter, not state:**
- Spawn depth is per-call-chain: ant (depth 1) -> sub-ant (depth 2) -> sub-sub-ant (depth 3)
- Each spawned ant knows its own depth from context (it was told by its parent)
- Storing depth in `COLONY_STATE.json` would require tracking per-agent state, which contradicts the simple `workers` model (6 named workers with string status)
- The worker spec already tells ants: "Max depth 3 (ant -> sub-ant -> sub-sub-ant, no deeper)"
- The spawning worker passes its own depth + 1 when calling spawn-check

**Implementation pattern (follows existing subcommand style):**
```bash
  spawn-check)
    depth="${1:-1}"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
    json_ok "$(jq --arg d "$depth" '
      (.workers | to_entries | map(select(.value != "idle")) | length) as $active |
      ($d | tonumber) as $depth |
      {
        pass: ($active < 5 and $depth <= 3),
        active_workers: $active,
        max_workers: 5,
        current_depth: $depth,
        max_depth: 3
      } | if .pass == false then
        . + {reason: (if $active >= 5 then "worker_limit" elif $depth > 3 then "depth_limit" else "unknown" end)}
      else . end
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
```

**Key design decisions:**
- Worker count check uses `<= 5` (per requirement), meaning `< 5` non-idle workers allows spawning since the new spawn would make it 5. Alternatively, the check could use `< 5` to allow spawning when fewer than 5 are active, since the spawn itself would bring it to 5. The requirement says "worker count (<= 5)" which means the count including the new spawn must be <= 5. Since we're checking BEFORE spawning, the threshold should be `active_workers < 5` (strictly less than -- spawning one more would make it 5, which is the max).
- Depth check uses `<= 3` (per requirement). If `current_depth` is 3, the ant is already at max and should NOT spawn deeper. The check should be `current_depth < 3` to allow spawning (spawned ant would be at depth current_depth + 1).
- Actually, re-reading the requirement: "checks worker count (<= 5) and spawn depth (<= 3)". This means the current state must satisfy count <= 5 AND depth <= 3 for the check to PASS. Since a depth-3 ant can't spawn (would create depth 4), the semantics are: `active_workers <= 5 AND current_depth <= 3` where `current_depth` is the depth of the proposed CHILD. But this conflicts -- depth 3 child is the max. So the check should be: the CALLER's depth < 3 (allowing the child to be at depth 3), and the active workers (including the future spawn) <= 5.
- **Recommended semantics:** `pass = (active_workers < 5) AND (current_depth < 3)` where `current_depth` is the CALLER's depth. This means: caller at depth 1 can spawn (child at depth 2), caller at depth 2 can spawn (child at depth 3), caller at depth 3 cannot spawn (would exceed max).

### Pattern 2: pheromone-validate Subcommand Design

**What:** A new `pheromone-validate` subcommand that checks pheromone content is non-empty and meets minimum length.

**Interface:**
```bash
bash .aether/aether-utils.sh pheromone-validate <content>
```

**Returns on pass:**
```json
{"ok":true,"result":{"pass":true,"length":45,"min_length":20}}
```

**Returns on fail:**
```json
{"ok":true,"result":{"pass":false,"reason":"too_short","length":12,"min_length":20}}
```

**Implementation pattern:**
```bash
  pheromone-validate)
    content="${1:-}"
    len=${#content}
    if [[ -z "$content" ]]; then
      json_ok '{"pass":false,"reason":"empty","length":0,"min_length":20}'
    elif [[ $len -lt 20 ]]; then
      json_ok "{\"pass\":false,\"reason\":\"too_short\",\"length\":$len,\"min_length\":20}"
    else
      json_ok "{\"pass\":true,\"length\":$len,\"min_length\":20}"
    fi
    ;;
```

**Note:** Uses shell string length (`${#content}`) rather than jq -- simpler and avoids quoting issues with arbitrary pheromone content passed through jq. The content is a single shell argument, so the caller must quote it.

### Pattern 3: Worker Spec Enforcement Gate (inserted before spawning)

**What:** A mandatory pre-spawn check in each worker spec's "You Can Spawn Other Ants" section.

**Where to insert:** Before the current "To spawn:" instructions, after the caste list.

**Pattern:**
```markdown
### Spawn Enforcement Gate (Mandatory)

Before spawning any ant, you MUST run the spawn-check gate:

Use the Bash tool to run:
```
bash .aether/aether-utils.sh spawn-check <your_depth>
```

Where `<your_depth>` is your current spawn depth (1 if you were spawned directly by the Queen/build command, 2 if spawned by another ant, 3 if spawned by a sub-ant).

This returns JSON: `{"ok":true,"result":{"pass":true|false,...}}`.

**If `pass` is false: DO NOT SPAWN. Report the blocked spawn to your parent:**
```
Spawn blocked: {reason} (active_workers: {N}, depth: {N})
Task that needed spawning: {description}
```

**If `pass` is true:** Proceed with spawning.

If the command fails, DO NOT SPAWN. Treat failure as a blocked spawn.
```

### Pattern 4: continue.md Pheromone Validation Gate

**What:** A validation call before writing auto-emitted pheromones in Step 4.5.

**Where to insert:** After constructing the pheromone content string, before appending to pheromones.json.

**Pattern:**
```markdown
Before appending the pheromone, validate its content using the Bash tool:
```
bash .aether/aether-utils.sh pheromone-validate "<content>"
```

This returns JSON: `{"ok":true,"result":{"pass":true|false,...}}`.

**If `pass` is false:** Do not write the pheromone. Log an event instead:
```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "pheromone_rejected",
  "source": "continue",
  "content": "Auto-pheromone rejected: <reason> (length: <N>, min: 20)",
  "timestamp": "<ISO-8601 UTC>"
}
```

**If `pass` is true:** Proceed to append the pheromone as normal.
```

### Pattern 5: Post-Action Validation Checklist

**What:** A deterministic checklist added to each worker spec's "Quality Standards" section (or as a new section after it).

**Pattern:**
```markdown
## Post-Action Validation (Mandatory)

Before reporting your results to the parent, complete these deterministic checks:

1. **State Validation:** Use the Bash tool to run:
   ```
   bash .aether/aether-utils.sh validate-state colony
   ```
   If `pass` is false, report the validation failure in your output.

2. **Spawn Limits Verified:** Confirm you did not exceed 5 sub-ants and did not spawn beyond depth 3. Report your spawn count in your output.

3. **Output Complete:** Verify your report follows the Output Format specified in this spec.

If any check fails, include the failure in your report — do not silently skip it.
```

### Anti-Patterns to Avoid
- **Adding spawn tracking to COLONY_STATE.json:** Don't add `spawn_depth` or `active_spawn_count` fields to the JSON -- depth is per-call-chain, not global state.
- **Making pheromone-validate read pheromones.json:** The subcommand validates a CONTENT STRING, not a file. It takes content as a parameter.
- **Rewriting the entire worker spec:** Targeted insertions only. The spawning section and quality standards section get additions, not replacements.
- **Making enforcement advisory:** The spawn-check gate must be a HARD GATE ("DO NOT SPAWN" on fail), not advisory like the existing Bayesian confidence check.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Counting active workers | Inline jq in worker spec prompts | `aether-utils.sh spawn-check` | Centralizes logic, consistent threshold, returns structured JSON |
| Checking spawn depth | "Count how deep you are" prompt instructions | `aether-utils.sh spawn-check <depth>` | Deterministic shell check vs. LLM self-assessment |
| Validating pheromone content | "Make sure your pheromone is descriptive" prompt text | `aether-utils.sh pheromone-validate` | Length check is deterministic; LLM "make sure" instructions are unreliable |
| Post-action state validation | "Check that state is consistent" prompt text | `aether-utils.sh validate-state colony` | Already exists, returns structured pass/fail |

**Key insight:** The entire point of this phase is replacing advisory prompt text ("Max 5 sub-ants per ant") with deterministic shell checks that return pass/fail JSON. Every enforcement gate must call a shell utility, not rely on the LLM following instructions.

## Common Pitfalls

### Pitfall 1: Depth Parameter Semantics Confusion
**What goes wrong:** The spawn-check passes when it should fail (or vice versa) because depth semantics are ambiguous.
**Why it happens:** Is "depth 3" the caller's depth or the child's depth? Does "max depth 3" mean 3 levels total or 3 spawning levels?
**How to avoid:** Define clearly: the parameter is the CALLER'S depth. The check passes if `caller_depth < 3` (meaning the caller can spawn a child at `caller_depth + 1`). The Queen/build command spawns at depth 0 (implicit), so the first ant is depth 1. Depth 1 can spawn (child at 2). Depth 2 can spawn (child at 3). Depth 3 cannot spawn. This matches the spec: "Max depth 3 (ant -> sub-ant -> sub-sub-ant, no deeper)."
**Warning signs:** Tests show depth-3 ants successfully spawning, or depth-1 ants being blocked.

### Pitfall 2: Worker Spec Spawning Section Drift Between Castes
**What goes wrong:** The 6 worker specs have slightly different spawning section text, and the enforcement gate is inserted inconsistently.
**Why it happens:** While the spawning sections are structurally identical, they have caste-specific spawning scenarios and slightly different line numbers.
**How to avoid:** Verify each spec's spawning section structure before editing. All 6 specs have: (1) "You Can Spawn Other Ants" header, (2) caste list, (3) "To spawn:" instructions, (4) "Spawn Confidence Check" section, (5) "Spawning Scenario" section, (6) "Spawn limits:" at the end. Insert the enforcement gate between (2) and (3) -- after the caste list, before "To spawn:".
**Warning signs:** Some worker specs have the gate, others don't. Or the gate is in different positions across specs.

### Pitfall 3: Shell Quoting in pheromone-validate
**What goes wrong:** Pheromone content with special characters (quotes, newlines, dollar signs) breaks the shell command.
**Why it happens:** The content is passed as a shell argument: `bash .aether/aether-utils.sh pheromone-validate "<content>"`. If content contains double quotes, the command breaks.
**How to avoid:** The continue.md instructions should use single quotes for the pheromone-validate call, or the subcommand should read from stdin for content with special characters. Recommendation: accept content as a quoted argument (the LLM generating the call can handle quoting), but also handle the edge case gracefully with a `json_err` on missing/empty arguments.
**Warning signs:** pheromone-validate works in simple tests but fails with real pheromone content.

### Pitfall 4: help Text Not Updated
**What goes wrong:** `aether-utils.sh help` doesn't list the 2 new subcommands.
**Why it happens:** The help text is a hardcoded JSON string on line 38.
**How to avoid:** Update line 38 to add `spawn-check` and `pheromone-validate` to the commands array. Post-addition list should be 13 commands: help, version, pheromone-decay, pheromone-effective, pheromone-batch, pheromone-cleanup, pheromone-validate, validate-state, spawn-check, memory-compress, error-add, error-pattern-check, error-summary.
**Warning signs:** `bash .aether/aether-utils.sh help` lists only 11 commands after the phase is done.

### Pitfall 5: Post-Action Checklist Conflicts with Existing Quality Standards
**What goes wrong:** The new "Post-Action Validation" section duplicates or contradicts the existing "Quality Standards" section in worker specs.
**Why it happens:** Each worker spec already has a "Quality Standards" section with caste-specific checks ("Key patterns are identified", "Question is thoroughly answered", etc.). The new post-action checklist adds generic deterministic checks.
**How to avoid:** Make the new section clearly separate from the existing quality standards. Name it "Post-Action Validation (Mandatory)" and place it AFTER the existing "Quality Standards" section. The existing section is about work quality (subjective); the new section is about compliance checks (deterministic).
**Warning signs:** The existing quality standards checkboxes get removed or modified.

### Pitfall 6: Forgetting the Depth Parameter in Spawning Instructions
**What goes wrong:** Worker specs tell ants to call `spawn-check` but don't explain how to determine their own depth.
**Why it happens:** The concept of "your depth" is implicit -- ants aren't currently told their depth level.
**How to avoid:** The build.md spawning prompt (Step 5) must include depth context: "You are at depth 1." Worker spec spawning instructions must say: "When spawning, tell the child its depth: include 'You are at depth <your_depth + 1>' in the TASK section." The spawn-check call must use the ant's own depth: `spawn-check <your_depth>`.
**Warning signs:** All spawn-check calls use depth 1, regardless of actual nesting level.

## Code Examples

### Example 1: spawn-check Subcommand (ENFO-01)

**Source:** Follows existing subcommand patterns in aether-utils.sh

```bash
  spawn-check)
    depth="${1:-1}"
    [[ -f "$DATA_DIR/COLONY_STATE.json" ]] || json_err "COLONY_STATE.json not found"
    json_ok "$(jq --arg d "$depth" '
      (.workers | to_entries | map(select(.value != "idle")) | length) as $active |
      ($d | tonumber) as $depth |
      {
        pass: ($active < 5 and $depth < 3),
        active_workers: $active,
        max_workers: 5,
        current_depth: $depth,
        max_depth: 3
      } | if .pass == false then
        . + {reason: (if $active >= 5 then "worker_limit" elif $depth >= 3 then "depth_limit" else "unknown" end)}
      else . end
    ' "$DATA_DIR/COLONY_STATE.json")"
    ;;
```

### Example 2: pheromone-validate Subcommand (ENFO-03)

**Source:** Follows existing subcommand patterns in aether-utils.sh

```bash
  pheromone-validate)
    content="${1:-}"
    len=${#content}
    if [[ -z "$content" ]]; then
      json_ok '{"pass":false,"reason":"empty","length":0,"min_length":20}'
    elif [[ $len -lt 20 ]]; then
      json_ok "{\"pass\":false,\"reason\":\"too_short\",\"length\":$len,\"min_length\":20}"
    else
      json_ok "{\"pass\":true,\"length\":$len,\"min_length\":20}"
    fi
    ;;
```

### Example 3: Worker Spec Enforcement Gate Insertion (ENFO-02)

**Before (architect-ant.md lines 156-170):**
```markdown
## You Can Spawn Other Ants

When you encounter a capability gap, spawn a specialist using the Task tool.

**Available castes and their spec files:**
- **colonizer** ...
- **route-setter** ...
- **builder** ...
- **watcher** ...
- **scout** ...
- **architect** ...

**To spawn:**
1. Use the Read tool to read the caste's spec file
...
```

**After:**
```markdown
## You Can Spawn Other Ants

When you encounter a capability gap, spawn a specialist using the Task tool.

**Available castes and their spec files:**
- **colonizer** ...
- **route-setter** ...
- **builder** ...
- **watcher** ...
- **scout** ...
- **architect** ...

### Spawn Gate (Mandatory)

Before spawning, you MUST pass the spawn-check gate. Use the Bash tool to run:
```
bash .aether/aether-utils.sh spawn-check <your_depth>
```

Where `<your_depth>` is your current spawn depth (1 if spawned by the build command, 2 if spawned by another ant, 3 if spawned by a sub-ant).

This returns JSON: `{"ok":true,"result":{"pass":true|false,...}}`.

**If `pass` is false: DO NOT SPAWN.** Report the blocked spawn to your parent:
```
Spawn blocked: {reason} (active_workers: {N}, depth: {N})
Task that needed spawning: {description}
```

**If `pass` is true:** Proceed to the confidence check and then spawn.

If the command fails, DO NOT SPAWN. Treat failure as a blocked spawn.

**To spawn:**
1. Use the Read tool to read the caste's spec file
...
```

### Example 4: continue.md Pheromone Validation (ENFO-04)

**Before (continue.md lines 170-171):**
```markdown
Append these to the `signals` array in `pheromones.json`. Use the Write tool to write the updated file.
```

**After:**
```markdown
Before appending each pheromone, validate its content. Use the Bash tool to run:
```
bash .aether/aether-utils.sh pheromone-validate "<the pheromone content string>"
```

This returns JSON: `{"ok":true,"result":{"pass":true|false,...}}`.

**If `pass` is false:** Do not append this pheromone. Instead, append a rejection event to events.json:
```json
{
  "id": "evt_<unix_timestamp>_<4_random_hex>",
  "type": "pheromone_rejected",
  "source": "continue",
  "content": "Auto-pheromone rejected: <reason> (length: <N>, min: 20)",
  "timestamp": "<ISO-8601 UTC>"
}
```

**If `pass` is true:** Append the pheromone to the `signals` array in `pheromones.json`.

If the command fails, skip validation and append the pheromone anyway (fail-open for auto-emitted pheromones).

Use the Write tool to write the updated pheromones.json.
```

### Example 5: Post-Action Validation Checklist (ENFO-05)

**Inserted after the existing "Quality Standards" section in each worker spec:**

```markdown
## Post-Action Validation (Mandatory)

Before reporting your results, complete these deterministic checks:

1. **State Validation:** Use the Bash tool to run:
   ```
   bash .aether/aether-utils.sh validate-state colony
   ```
   If `pass` is false, include the validation failure in your report.

2. **Spawn Accounting:** Report your spawn count: "Spawned: {N}/5 sub-ants". Confirm you did not exceed depth limits.

3. **Report Format:** Verify your report follows the Output Format section above.

Include check results at the end of your report:
```
Post-Action Validation:
  State: {pass|fail}
  Spawns: {N}/5 (depth {your_depth}/3)
  Format: {pass|fail}
```
```

## State of the Art

| Old Approach | Current Approach (Phase 23) | Why |
|--------------|---------------------------|-----|
| "Max 5 sub-ants per ant" (advisory text) | `spawn-check` shell gate (deterministic pass/fail) | LLMs ignore advisory limits; shell code enforces them |
| "Max depth 3" (advisory text) | `spawn-check <depth>` parameter (deterministic) | Depth counting by LLMs is unreliable |
| No pheromone content validation | `pheromone-validate` shell gate | Prevents empty/trivial auto-pheromones from polluting the signal space |
| No post-action compliance checks | `validate-state colony` mandatory call | Workers verify state consistency before reporting done |

**Context from Phase 22:** Phase 22 established the pattern of replacing inline LLM logic with utility calls. Phase 23 extends this pattern from computation (pheromone decay, error counting) to enforcement (spawn gates, validation gates).

## Open Questions

1. **Worker count semantics: inclusive or exclusive?**
   - What we know: COLONY_STATE.json has 6 named workers with string status. The requirement says "worker count (<= 5)". The spawning section says "Max 5 sub-ants per ant."
   - What's unclear: Does "worker count <= 5" mean the 6 named workers in COLONY_STATE.json, or the sub-ants spawned by a single parent? The named workers (colonizer, route-setter, builder, watcher, scout, architect) are 6 total, but only some are "active" at any time.
   - Recommendation: Count non-idle workers in COLONY_STATE.json. The check is "can this colony support one more active worker?" If 5 of 6 workers are already non-idle, deny further spawns. This is a colony-wide limit, not per-parent. The per-parent "max 5 sub-ants" limit is enforced by the prompt text and is harder to track deterministically (would require per-parent spawn counting infrastructure). The shell check covers the colony-wide safety limit.

2. **Depth parameter bootstrapping**
   - What we know: The build.md Step 5 spawns the "Phase Lead" ant without any depth context currently. Worker specs tell ants they can spawn at "max depth 3" but don't tell them their own depth.
   - What's unclear: How does the Phase Lead ant know it's at depth 1?
   - Recommendation: Update build.md Step 5 to include "You are at depth 1." in the spawn prompt. Worker spec spawn instructions should include "Tell the child: 'You are at depth {your_depth + 1}.'" This propagates depth through the spawn chain.

3. **Fail-open vs fail-closed for pheromone-validate**
   - What we know: Requirement says "rejects invalid pheromones." But auto-emitted pheromones from continue.md are generated by the LLM based on phase learnings.
   - What's unclear: If `pheromone-validate` fails (command error, not content failure), should the pheromone be rejected (fail-closed) or accepted (fail-open)?
   - Recommendation: Fail-open for command errors (the pheromone was generated, just can't be validated). Fail-closed for content failures (the content genuinely fails the check). This matches the pattern in other utility calls: "If the command fails, [reasonable fallback]."

## Detailed File Edit Locations

### ENFO-01: spawn-check subcommand in aether-utils.sh

**File:** `.aether/aether-utils.sh`
**Insert after:** The `error-summary` case block (line 196), before the `*)` default case (line 198)
**Also update:** Line 38 help text -- add `spawn-check` to the commands array
**Lines added:** ~15

### ENFO-02: All 6 worker specs

**Files (all in `.aether/workers/`):**
| File | Insert gate after line | Section |
|------|----------------------|---------|
| architect-ant.md | After line 166 (caste list) | Before "To spawn:" |
| builder-ant.md | After line 166 (caste list) | Before "To spawn:" |
| colonizer-ant.md | After line 166 (caste list) | Before "To spawn:" |
| route-setter-ant.md | After line 166 (caste list) | Before "To spawn:" |
| scout-ant.md | After line 166 (caste list) | Before "To spawn:" |
| watcher-ant.md | After line 280 (caste list) | Before "To spawn:" |

**Also update:** Each spec's "Spawn limits:" section at the end -- change from advisory to reference the gate:
```
**Spawn limits (enforced by spawn-check):**
- Max 5 active workers colony-wide
- Max depth 3 (ant -> sub-ant -> sub-sub-ant, no deeper)
- If spawn-check fails, don't spawn — report the gap to parent
```

**Also update:** build.md Step 5 spawn prompt -- add depth context: "You are at depth 1."

### ENFO-03: pheromone-validate subcommand in aether-utils.sh

**File:** `.aether/aether-utils.sh`
**Insert after:** `pheromone-cleanup` case block (around line 75), keeping pheromone-related subcommands grouped
**Also update:** Line 38 help text -- add `pheromone-validate` to the commands array
**Lines added:** ~10

### ENFO-04: continue.md auto-pheromone validation

**File:** `.claude/commands/ant/continue.md`
**Insert before:** Line 171 ("Append these to the `signals` array...")
**Lines added:** ~20

### ENFO-05: Post-action validation checklist in all 6 worker specs

**Files (all in `.aether/workers/`):**
| File | Insert after | Section |
|------|-------------|---------|
| architect-ant.md | After "Quality Standards" section (line 154) | New section |
| builder-ant.md | After "Implementation Principles" section (line 150) | New section |
| colonizer-ant.md | After "Quality Standards" section (line 150) | New section |
| route-setter-ant.md | After "Planning Heuristics" section (line 152) | New section |
| scout-ant.md | After "Quality Standards" section (line 164) | New section |
| watcher-ant.md | After "Output Format" section (line 264) | New section |

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` -- direct reading, 201 lines, 11 subcommands, all interfaces verified
- `.aether/workers/architect-ant.md` -- direct reading, spawning section structure verified (lines 156-244)
- `.aether/workers/builder-ant.md` -- direct reading, spawning section structure verified (lines 152-240)
- `.aether/workers/colonizer-ant.md` -- direct reading, spawning section structure verified (lines 152-241)
- `.aether/workers/route-setter-ant.md` -- direct reading, spawning section structure verified (lines 154-243)
- `.aether/workers/scout-ant.md` -- direct reading, spawning section structure verified (lines 166-255)
- `.aether/workers/watcher-ant.md` -- direct reading, spawning section structure verified (lines 266-355)
- `.claude/commands/ant/continue.md` -- direct reading, auto-pheromone step verified (lines 135-184)
- `.claude/commands/ant/build.md` -- direct reading, Step 5 spawn prompt structure verified (lines 104-175)
- `.aether/data/COLONY_STATE.json` -- direct reading, workers structure verified (6 named workers with string status)
- `.planning/REQUIREMENTS.md` -- ENFO-01 through ENFO-05 requirements verified
- `.planning/ROADMAP.md` -- Phase 23 scope and success criteria verified
- `.planning/phases/22-cleanup/22-VERIFICATION.md` -- Phase 22 completion verified, clean baseline

### Secondary (MEDIUM confidence)
None needed -- all findings from primary source reading.

### Tertiary (LOW confidence)
None -- all findings verified from direct file reading.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new tools, extends existing aether-utils.sh with 2 subcommands
- Architecture: HIGH -- subcommand patterns copied from existing code, insertion points verified by line number
- Pitfalls: HIGH -- all 6 pitfalls identified from structural analysis of existing files
- Code examples: HIGH -- all examples follow verified existing patterns in aether-utils.sh

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (stable -- internal project files, not external dependencies)
