# Phase 13: Midden Write Path Expansion - Research

**Researched:** 2026-03-14
**Domain:** Midden failure tracking, pheromone emission, playbook instruction wiring
**Confidence:** HIGH

## Summary

Phase 13 expands the midden write path so that all agent failure types produce structured entries in midden.json, approach changes feed both midden and the memory pipeline, and a new intra-phase threshold check emits REDIRECT pheromones mid-build when the same error category recurs 3+ times.

This is a pure playbook-wiring phase. Every subcommand needed already exists and is tested (`midden-write`, `midden-recent-failures`, `memory-capture`, `pheromone-write`). The work is adding calls to these subcommands at the right points in the build-wave.md and build-verify.md playbooks. No new subcommands, no schema changes, no new state files.

**Primary recommendation:** Three plans -- (1) wire `midden-write` + `memory-capture` at all missing failure points across build-wave.md and build-verify.md, (2) add approach-change capture using `midden-write` + `memory-capture` in the builder prompt's approach-change block, (3) add intra-phase midden threshold check after wave failure processing. All edit build-wave.md and build-verify.md only (not continue-advance.md or continue-gates.md, which Phase 14 edits).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MID-01 | All failure types (Watcher, Chaos, verification, Gatekeeper, Auditor) write to midden via midden-write | Gap analysis in "Current Midden Write Coverage" section maps every failure point; 3 of 5 agent failure types already have `midden-write` calls but are missing `memory-capture`; Gatekeeper and Auditor findings in continue-gates.md are OUT OF SCOPE for this phase (Phase 14 edits continue playbooks) -- but their existing `midden-write` calls already satisfy MID-01 for those agents |
| MID-02 | Approach changes during builds are captured to midden and memory-capture as abandoned-approach events | Current approach-change block in build-wave.md only writes to `.aether/midden/approach-changes.md` markdown file; needs `midden-write "abandoned-approach"` + `memory-capture "failure"` calls added |
| MID-03 | Intra-phase midden threshold check fires during build waves so REDIRECT pheromones can emit mid-build | continue-advance.md Step 2.1c already implements threshold check at phase-end; build-wave.md needs identical logic after Step 5.2 wave completion processing |
</phase_requirements>

## Current Midden Write Coverage

This is the critical map of what currently writes to midden and what is missing.

### Two Midden Storage Systems

**IMPORTANT:** The codebase has two parallel midden storage systems:

| Storage | Path | Format | Used by |
|---------|------|--------|---------|
| Structured JSON | `.aether/data/midden/midden.json` | JSON with entries array | `midden-write` subcommand, `midden-recent-failures` query |
| Markdown files | `.aether/midden/*.md` | YAML-ish list entries | Heredoc `cat >>` in playbook code blocks |

The structured JSON store (`midden.json`) is the authoritative midden. The markdown files (`approach-changes.md`, `build-failures.md`, `test-failures.md`) are write-only logs that are never queried programmatically. The `midden-recent-failures` subcommand reads only from `midden.json`.

**Implication for this phase:** The markdown file writes can coexist with `midden-write` calls. The approach-change block currently writes ONLY to the markdown file and NOT to `midden.json`. Adding `midden-write` makes approach changes queryable by `midden-recent-failures` and therefore visible to the intra-phase threshold check.

### Failure Point Coverage Matrix

| Agent | Failure Type | Writes to midden.json | Writes to .md file | Calls memory-capture | Playbook File |
|-------|-------------|----------------------|-------------------|---------------------|---------------|
| Builder | Worker failure | NO (gap) | YES (build-failures.md) | YES | build-wave.md Step 5.2 |
| Builder | Approach change | NO (gap) | YES (approach-changes.md) | NO (gap) | build-wave.md Builder prompt |
| Watcher | Verification failure | NO (gap) | YES (test-failures.md) | YES | build-verify.md Step 5.8 |
| Chaos | Critical/high finding | NO (gap) | YES (build-failures.md) | YES | build-verify.md Step 5.7 |
| Gatekeeper | High-severity CVEs | YES | NO | NO (gap) | continue-gates.md Step 1.8 |
| Auditor | High-severity quality | YES | NO | NO (gap) | continue-gates.md Step 1.9 |
| Measurer | Performance findings | YES | NO | NO | build-verify.md Step 5.5.1 |
| Probe | Coverage findings | YES | NO | NO | continue-verify.md Step 1.5.1 |
| Ambassador | Integration plan | YES | NO | NO | build-wave.md Step 5.1.1 |
| Weaver | Refactoring | YES | NO | NO | continue-gates.md Step 1.7.1 |

**Key findings:**

1. **Builder, Watcher, Chaos failures write to markdown files but NOT to midden.json** -- they use heredoc `cat >>` to append to `.aether/midden/*.md` files instead of calling `midden-write`. This means `midden-recent-failures` never sees these failures and the intra-phase threshold check cannot detect them.

2. **Gatekeeper and Auditor already call midden-write** but do NOT call `memory-capture`. However, continue-gates.md is owned by Phase 14 (different playbook files), so adding `memory-capture` there is Phase 14's scope.

3. **Approach changes write ONLY to a markdown file** with no `midden-write` or `memory-capture` call.

### Scope Boundary: Which Playbooks This Phase Edits

Per the roadmap: "Phases 13 and 14 parallelizable -- they edit different playbook files (build-wave/continue-verify vs continue-advance)."

| Playbook | Phase 13 edits | Phase 14 edits |
|----------|---------------|---------------|
| build-wave.md | YES (Steps 5.1-5.3, Builder prompt) | NO |
| build-verify.md | YES (Steps 5.4-5.8) | NO |
| build-complete.md | NO (already handled by Phase 12) | NO |
| continue-verify.md | NO | NO |
| continue-gates.md | NO | YES |
| continue-advance.md | NO | YES |
| continue-finalize.md | NO | NO |
| build-full.md | YES (mirror of split playbooks) | YES (mirror) |
| continue-full.md | NO | YES (mirror) |

**CRITICAL:** build-full.md is a monolithic mirror of ALL split build playbooks. Any edit to build-wave.md or build-verify.md MUST also be mirrored in build-full.md at the corresponding line numbers. The same applies for continue-full.md with continue playbook edits.

## Architecture Patterns

### Pattern 1: Adding midden-write at Existing Failure Points

**What:** Add a `midden-write` call alongside the existing heredoc markdown write.
**When to use:** Builder failures (Step 5.2), Watcher failures (Step 5.8), Chaos findings (Step 5.7).

**Example (from existing Gatekeeper pattern in continue-gates.md):**
```bash
bash .aether/aether-utils.sh midden-write "security" "High CVEs found: $high_count" "gatekeeper"
```

The pattern is: `midden-write <category> <message> <source>`
- `category`: groups entries for threshold detection (e.g., "worker_failure", "verification", "resilience", "abandoned-approach")
- `message`: human-readable description of the failure
- `source`: which agent produced it (e.g., "builder", "watcher", "chaos")

**Placement:** Add the `midden-write` call AFTER the existing heredoc block (do not remove the heredoc -- it serves as a human-readable log). The `memory-capture` call already exists at these points, so no change needed there.

### Pattern 2: Approach-Change Capture

**What:** Add `midden-write` + `memory-capture` calls in the builder prompt's approach-change block.
**When to use:** MID-02 requirement.

**Current approach-change block in build-wave.md (lines 343-357):**
```bash
colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")
phase_num=$(jq -r '.phase.number // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

cat >> .aether/midden/approach-changes.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: ${phase_num}
  colony: "${colony_name}"
  worker: "{Ant-Name}"
  task: "{task_id}"
  tried: "initial approach that failed"
  why_it_failed: "reason it didn't work"
  switched_to: "new approach that worked"
EOF
```

**What to add after this block:**
```bash
# Write to structured midden for threshold detection
bash .aether/aether-utils.sh midden-write "abandoned-approach" "Tried: initial approach that failed. Switched to: new approach. Reason: reason it didn't work" "builder" 2>/dev/null || true

# Enter memory pipeline for learning observation tracking
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Approach abandoned: initial approach that failed -> new approach (reason it didn't work)" \
  "failure" \
  "worker:builder" 2>/dev/null || true
```

**Category choice:** Use `"abandoned-approach"` as the category for midden-write. This is distinct from `"worker_failure"` or `"verification"` and allows the threshold check to detect patterns of approach abandonment separately from hard failures.

### Pattern 3: Intra-Phase Midden Threshold Check

**What:** After processing wave results, query midden for recurring error categories and emit REDIRECT if any category reaches 3+ occurrences.
**When to use:** After Step 5.2 (wave result processing) in build-wave.md.

**Reference implementation (from continue-advance.md Step 2.1c):**
```bash
midden_result=$(bash .aether/aether-utils.sh midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
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
    [[ -z "$encoded" ]] && continue
    category=$(echo "$encoded" | base64 -d | jq -r '.category')
    count=$(echo "$encoded" | base64 -d | jq -r '.count')

    existing=$(jq -r --arg cat "$category" '
      [.signals[] | select(.active == true and .source == "auto:error" and (.content.text | contains($cat)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")

    if [[ "$existing" == "0" ]]; then
      bash .aether/aether-utils.sh pheromone-write REDIRECT \
        "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
        --strength 0.7 \
        --source "auto:error" \
        --reason "Auto-emitted: midden error pattern recurred 3+ times mid-build" \
        --ttl "30d" 2>/dev/null || true
      emit_count=$((emit_count + 1))
    fi
  done
fi
```

**Adaptation for build-wave.md:**
- Place this check AFTER Step 5.2 (wave results processed) and AFTER Step 5.7 (chaos results processed)
- The check runs after each wave's failures are logged, so it can detect cross-wave patterns within the same phase
- Use the same 3+ threshold and dedup logic as continue-advance.md
- Cap at 3 REDIRECT emissions per build (same cap as continue)
- Display a visible alert if a REDIRECT is emitted mid-build:
  ```
  ⚠️ Midden threshold: "{category}" recurring ({count}x) -- REDIRECT emitted mid-build
  ```

### Anti-Patterns to Avoid

- **Removing the markdown heredoc writes:** These serve as a human-readable audit log. Keep them alongside the new `midden-write` calls. Both storage systems serve different purposes.
- **Adding `midden-write` inside the worker prompt itself:** Workers run in subagent sandboxes. The midden-write call should happen in the Queen's wave processing logic (Steps 5.2, 5.7, 5.8), not inside the worker prompt.
- **Running threshold check inside worker prompts:** Workers cannot emit REDIRECT pheromones. The threshold check must run in the Queen's orchestration layer between waves.
- **Editing continue-gates.md or continue-advance.md:** These are Phase 14's territory. Even though Gatekeeper/Auditor findings lack `memory-capture` calls, that wiring belongs to Phase 14.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Writing to midden.json | Manual jq appends | `midden-write` subcommand | Handles locking, ID generation, graceful degradation |
| Querying midden failures | Raw jq on midden.json | `midden-recent-failures` subcommand | Returns structured JSON with count and sorted failures |
| Emitting REDIRECT pheromones | Manual JSON writes to pheromones.json | `pheromone-write REDIRECT` subcommand | Handles ID generation, dedup, strength, TTL, source tracking |
| Recording to memory pipeline | Direct writes to learning-observations.json | `memory-capture` subcommand | Orchestrates observe -> pheromone -> auto-promotion chain |
| Threshold detection jq | Custom grouping logic | Copy the existing jq from continue-advance.md Step 2.1c | Already tested, handles edge cases |

**Key insight:** Every subcommand needed for this phase is already implemented and tested in aether-utils.sh. The phase is 100% playbook instruction editing -- no shell code changes.

## Common Pitfalls

### Pitfall 1: Dual Storage Confusion
**What goes wrong:** Editing `midden.json` directly instead of calling `midden-write`, or expecting `midden-recent-failures` to read from the markdown files.
**Why it happens:** Two midden paths exist (`.aether/data/midden/midden.json` and `.aether/midden/*.md`).
**How to avoid:** Always use `midden-write` for structured entries. The markdown files are write-only human logs.
**Warning signs:** Test expects `midden-recent-failures` to return data but no `midden-write` call was made.

### Pitfall 2: build-full.md Desync
**What goes wrong:** Editing build-wave.md or build-verify.md but forgetting to mirror changes in build-full.md.
**Why it happens:** build-full.md is a monolithic copy of all split build playbooks.
**How to avoid:** For every edit to build-wave.md or build-verify.md, find the corresponding section in build-full.md and apply the same edit.
**Warning signs:** Split playbook tests pass but full-playbook behavior differs.

### Pitfall 3: Category Naming Inconsistency
**What goes wrong:** Using different category strings for the same failure type, breaking threshold detection.
**Why it happens:** Each failure point is written independently; no central category registry.
**How to avoid:** Use these standardized categories:
  - `"worker_failure"` -- builder worker returning status: failed
  - `"verification"` -- watcher verification failure
  - `"resilience"` -- chaos critical/high finding
  - `"abandoned-approach"` -- builder approach change
  - `"security"` -- gatekeeper finding (already exists)
  - `"quality"` -- auditor finding (already exists)
**Warning signs:** Threshold check groups by category but never reaches 3 because the same type uses different category strings.

### Pitfall 4: Memory-Capture Missing for Approach Changes
**What goes wrong:** Adding `midden-write` for approach changes but forgetting `memory-capture`, so the learning-observations pipeline never sees the abandoned approach.
**Why it happens:** MID-02 requires both midden AND learning-observations.json capture.
**How to avoid:** Every `midden-write` for a failure event should be paired with `memory-capture "failure"` to enter the learning pipeline.

### Pitfall 5: Threshold Check Placement
**What goes wrong:** Placing the threshold check inside the per-worker processing loop instead of after the entire wave is processed.
**Why it happens:** Unclear timing in the wave lifecycle.
**How to avoid:** The threshold check runs ONCE per wave, after ALL worker results for that wave are processed (after Step 5.2 loop completes), not after each individual worker result.

## Code Examples

### Example 1: Adding midden-write to Builder Failure (Step 5.2)

Current code (build-wave.md lines 393-419):
```bash
# Existing heredoc write to build-failures.md
cat >> .aether/midden/build-failures.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  ...
EOF

# Existing memory-capture call
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" \
  "failure" \
  "worker:builder" 2>/dev/null || true
```

**Add between heredoc and memory-capture:**
```bash
# Write to structured midden for threshold detection (MID-01)
bash .aether/aether-utils.sh midden-write "worker_failure" "Builder ${ant_name} failed on task ${task_id}: ${blockers[0]:-$failure_reason}" "builder" 2>/dev/null || true
```

### Example 2: Adding midden-write to Chaos Finding (Step 5.7)

Current code (build-verify.md lines 294-320):
```bash
# Existing heredoc write to build-failures.md
cat >> .aether/midden/build-failures.md << EOF
...
EOF

# Existing memory-capture call
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Resilience issue found: ${finding.title} (${finding.severity})" \
  "failure" \
  "worker:chaos" 2>/dev/null || true
```

**Add between heredoc and memory-capture:**
```bash
# Write to structured midden for threshold detection (MID-01)
bash .aether/aether-utils.sh midden-write "resilience" "Chaos finding: ${finding.title} (${finding.severity})" "chaos" 2>/dev/null || true
```

### Example 3: Adding midden-write to Watcher Failure (Step 5.8)

Current code (build-verify.md lines 347-373):
```bash
# Existing heredoc write to test-failures.md
cat >> .aether/midden/test-failures.md << EOF
...
EOF

# Existing memory-capture call
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Verification failed: ${issue_title} - ${issue_description}" \
  "failure" \
  "worker:watcher" 2>/dev/null || true
```

**Add between heredoc and memory-capture:**
```bash
# Write to structured midden for threshold detection (MID-01)
bash .aether/aether-utils.sh midden-write "verification" "Watcher verification failed: ${issue_title}" "watcher" 2>/dev/null || true
```

### Example 4: Intra-Phase Threshold Check (after wave processing)

Place after Step 5.2 wave completion:
```bash
# Intra-phase midden threshold check (MID-03)
midden_result=$(bash .aether/aether-utils.sh midden-recent-failures 50 2>/dev/null || echo '{"count":0,"failures":[]}')
midden_count=$(echo "$midden_result" | jq '.count // 0')

if [[ "$midden_count" -gt 0 ]]; then
  recurring_categories=$(echo "$midden_result" | jq -r '
    [.failures[] | .category]
    | group_by(.)
    | map(select(length >= 3))
    | map({category: .[0], count: length})
    | .[]
    | @base64
  ' 2>/dev/null || echo "")

  redirect_emit_count=0
  for encoded in $recurring_categories; do
    [[ $redirect_emit_count -ge 3 ]] && break
    [[ -z "$encoded" ]] && continue
    category=$(echo "$encoded" | base64 -d | jq -r '.category')
    count=$(echo "$encoded" | base64 -d | jq -r '.count')

    existing=$(jq -r --arg cat "$category" '
      [.signals[] | select(.active == true and .source == "auto:error" and (.content.text | contains($cat)))] | length
    ' .aether/data/pheromones.json 2>/dev/null || echo "0")

    if [[ "$existing" == "0" ]]; then
      bash .aether/aether-utils.sh pheromone-write REDIRECT \
        "[error-pattern] Category \"$category\" recurring ($count occurrences)" \
        --strength 0.7 \
        --source "auto:error" \
        --reason "Auto-emitted: midden error pattern recurred 3+ times mid-build" \
        --ttl "30d" 2>/dev/null || true
      redirect_emit_count=$((redirect_emit_count + 1))
    fi
  done

  if [[ $redirect_emit_count -gt 0 ]]; then
    echo "Warning: Midden threshold triggered -- $redirect_emit_count REDIRECT pheromone(s) emitted mid-build"
  fi
fi
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Markdown-only midden files | `midden-write` subcommand to JSON | v1.0 (Phase 36 archive) | Structured storage, queryable |
| Failures logged but not queried | `midden-recent-failures` injected into worker context | v1.0 | Workers avoid known failure patterns |
| REDIRECT from midden only at phase-end | REDIRECT at both phase-end and mid-build (MID-03) | This phase | Faster behavioral correction |
| Approach changes write-only | Approach changes feed midden + memory pipeline | This phase | Abandoned approaches become learnable |

## Recommended Plan Structure

### Plan 13-01: Wire midden-write at All Failure Points (MID-01)

**Files edited:** build-wave.md, build-verify.md, build-full.md
**What:** Add `midden-write` calls at Builder failure (Step 5.2), Chaos finding (Step 5.7), and Watcher failure (Step 5.8) points. These three agents currently write to markdown files and call `memory-capture`, but do NOT write to structured `midden.json`.
**Scope:** 3 insertion points in split playbooks + 3 corresponding mirror points in build-full.md = 6 edits total.
**Verification:** After a build with a failed worker, `midden-recent-failures` should return entries with the correct category and source.

### Plan 13-02: Approach-Change Capture (MID-02)

**Files edited:** build-wave.md, build-full.md
**What:** Expand the approach-change block in the Builder worker prompt to call `midden-write "abandoned-approach"` and `memory-capture "failure"` after the existing markdown heredoc write.
**Scope:** 1 insertion point in build-wave.md + 1 mirror in build-full.md = 2 edits.
**Verification:** After a build where a builder changes approach, both `midden.json` entries and `learning-observations.json` entries should contain the abandoned approach data.

### Plan 13-03: Intra-Phase Midden Threshold Check (MID-03)

**Files edited:** build-wave.md, build-full.md
**What:** Add a midden threshold check block after wave result processing (after Step 5.2 completes). If any midden category reaches 3+ occurrences, emit a REDIRECT pheromone mid-build. Copy the threshold logic directly from continue-advance.md Step 2.1c.
**Scope:** 1 insertion point in build-wave.md + 1 mirror in build-full.md = 2 edits.
**Dependency:** Requires MID-01 (Plan 13-01) to be applied first so that Builder/Watcher/Chaos failures actually appear in midden.json for the threshold check to detect.
**Verification:** In a build with 3+ failures of the same category, `pheromones.json` should contain a new REDIRECT signal with source `"auto:error"` emitted during the build (not deferred to continue).

## Open Questions

1. **Should the threshold check also run after Step 5.7 (Chaos results)?**
   - What we know: Chaos findings write to midden with category "resilience". If 3+ resilience issues are found, a REDIRECT would be warranted.
   - What's unclear: Chaos runs once per build (not per wave), so 3+ resilience findings from a single Chaos run would all come from one agent.
   - Recommendation: YES, run the threshold check after Step 5.7 as well. A single Chaos run reporting 3+ findings of the same severity is a strong enough signal. The threshold check is cheap (one `midden-recent-failures` call + jq grouping) and the dedup prevents duplicate REDIRECT emissions.

2. **Should the Gatekeeper/Auditor memory-capture gap be addressed here?**
   - What we know: Gatekeeper and Auditor in continue-gates.md already call `midden-write` but not `memory-capture`. Adding `memory-capture` would complete the pipeline.
   - What's unclear: The scope boundary says Phase 14 owns continue-gates.md.
   - Recommendation: Leave for Phase 14. The scope boundary is explicit: "Phases 13 and 14 parallelizable -- they edit different playbook files." MID-01 is already satisfied for Gatekeeper/Auditor because they DO call `midden-write`. The `memory-capture` gap is a separate concern (T5 in the feature research).

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 8230-8295 -- `midden-write` subcommand implementation
- `.aether/aether-utils.sh` lines 9600-9625 -- `midden-recent-failures` subcommand implementation
- `.aether/aether-utils.sh` lines 5402-5501 -- `memory-capture` subcommand implementation
- `.aether/aether-utils.sh` lines 6774-6854 -- `pheromone-write` subcommand implementation
- `.aether/docs/command-playbooks/build-wave.md` -- current Builder failure and approach-change handling
- `.aether/docs/command-playbooks/build-verify.md` -- current Watcher and Chaos failure handling
- `.aether/docs/command-playbooks/continue-advance.md` lines 234-285 -- existing midden threshold check (reference implementation)
- `.aether/docs/command-playbooks/continue-gates.md` -- Gatekeeper/Auditor midden-write calls (already in place)
- `tests/integration/pheromone-auto-emission.test.js` -- existing test coverage for midden-recent-failures and REDIRECT emission
- `.planning/research/FEATURES.md` -- feature landscape analysis for v1.2 (T2, T4, T5 features map to MID-01/02/03)

### Secondary (MEDIUM confidence)
- `.planning/REQUIREMENTS.md` -- MID-01, MID-02, MID-03 requirement definitions
- `.planning/ROADMAP.md` -- phase scope boundaries and parallelization constraints

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all subcommands exist, are tested, and have known interfaces
- Architecture: HIGH -- the pattern (add midden-write alongside existing heredoc) is consistent with how Gatekeeper/Auditor already work
- Pitfalls: HIGH -- dual storage system and build-full.md mirroring are well-documented gotchas from previous phases

**Research date:** 2026-03-14
**Valid until:** 2026-04-14 (stable infrastructure, no expected changes)
