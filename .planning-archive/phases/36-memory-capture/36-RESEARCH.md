# Phase 36: Memory Capture - Research

**Researched:** 2026-02-21
**Domain:** Colony Memory Systems Integration
**Confidence:** HIGH

## Summary

This phase wires the existing memory systems (learning-observe, learning-approve-proposals, midden) into `/ant:continue` and `/ant:build` commands to automatically capture learnings and failures. The research confirms all required infrastructure exists and is operational.

**Key findings:**
1. `learning-observe` function exists in aether-utils.sh with SHA256 content hashing for deduplication
2. `learning-approve-proposals` provides checkbox-style approval UX from Phase 34
3. `midden/midden.json` exists for archiving expired pheromones (can be extended for failures)
4. Current thresholds: philosophy=5, pattern=3, redirect=2, stack=1, decree=0

**Primary recommendation:** Lower pattern threshold from 3 to 1 (with user approval as quality gate), integrate automatic observation capture into build failure paths, and add midden logging for structured failure storage.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Learning Capture UX:**
- **Automatic observation** — Colony observes patterns during builds without user input; no manual prompting for "what did you learn"
- **Checkbox approval at continue** — Reuse Phase 34's approval flow pattern; user selects which captured learnings to promote
- **Silent skip if empty** — If no learnings were captured, skip the prompt entirely without notice

**Failure Logging Scope:**
- **Build failures** — Worker errors, timeouts, unhandled exceptions
- **Approach changes** — Worker self-reports when trying X doesn't work and switching to Y; requires agent convention for logging
- **All test failures** — Including TDD red-green cycle; captures test failures during development, not just final builds
- **NOT user redirects** — REDIRECT signals are intentional guidance, not failures

**Midden Structure:**
- **One file per type** — `midden/build-failures.md`, `midden/test-failures.md`, `midden/approach-changes.md`
- **Structured YAML/JSON entries** — Each entry includes: timestamp, phase, what failed, why, what worked instead
- **Parseable format** — Tools can read and aggregate failures across phases

**Promotion Threshold:**
- **1 observation + user approval** — Lowered from 5; user approval is the quality gate
- **Rationale** — 5-observation threshold is why QUEEN.md stays empty; if something is worth capturing once and user approves, it's valid wisdom

### Claude's Discretion
- Exact YAML/JSON schema for midden entries
- How to integrate failure logging into existing worker patterns
- Whether to append or prepend entries in midden files
- Error handling when midden write fails

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MEM-01 | /ant:continue asks "What did you learn this phase?" and writes approved learnings to QUEEN.md immediately | `learning-approve-proposals` function exists (line 4723 in aether-utils.sh) with checkbox UX. Phase 34 approval pattern proven. Integration point: Step 2.1.5 in continue.md |
| MEM-02 | /ant:build logs failed approaches to midden/ AND calls learning-observe with type=failure | Current `learning-observe` validates wisdom_type against ["philosophy", "pattern", "redirect", "stack", "decree"]. Need to: (1) add "failure" type OR map failures to "redirect", (2) create midden write functions |
| MEM-03 | Lower promotion threshold to 1 observation + user approval (remove the 5-observation requirement) | Current thresholds at lines 4195-4202: philosophy=5, pattern=3, redirect=2, stack=1, decree=0. Need to modify learning-check-promotion (line 4257) and learning-observe (line 4195) to use threshold=1 for all types |
</phase_requirements>

## Standard Stack

### Core (Already Exists)
| Component | Location | Purpose | Status |
|-----------|----------|---------|--------|
| learning-observe | aether-utils.sh:4087 | Records observations with SHA256 hashing | OPERATIONAL |
| learning-check-promotion | aether-utils.sh:4232 | Returns proposals meeting thresholds | OPERATIONAL |
| learning-approve-proposals | aether-utils.sh:4723 | Checkbox approval workflow | OPERATIONAL |
| learning-display-proposals | aether-utils.sh:4286 | Visual proposal display | OPERATIONAL |
| queen-promote | aether-utils.sh:3782 | Writes wisdom to QUEEN.md | OPERATIONAL |
| midden.json | .aether/data/midden/ | Archives expired pheromones | EXISTS |

### Supporting
| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| parse-selection | aether-utils.sh:3679 | Parses user number input | Already used by approval flow |
| generate_threshold_bar | aether-utils.sh:~3600 | Visual threshold progress | Already integrated |

## Architecture Patterns

### Pattern 1: Automatic Observation Capture
**What:** During build execution, automatically call learning-observe when patterns are detected
**When to use:** Build failures, test failures, approach changes
**Integration points in build.md:**
- Step 5.2 (Process Wave 1 Results): Capture builder failures
- Step 5.5 (Process Watcher Results): Capture verification failures
- Step 5.7 (Process Chaos Ant Results): Capture resilience findings

**Example integration:**
```bash
# After detecting worker failure in build.md
bash .aether/aether-utils.sh learning-observe \
  "Worker {ant_name} failed on {task}: {failure_reason}" \
  "failure" \
  "$colony_name" 2>/dev/null || true
```

### Pattern 2: Checkbox Approval Flow (Phase 34)
**What:** Display proposals with `[ ]` checkboxes, user selects by number
**When to use:** End of phase in continue.md Step 2.1.5
**Current implementation:**
```bash
# From continue.md line 727-735
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo "[]")
if [[ -n "$proposals" ]]; then
  bash .aether/aether-utils.sh learning-approve-proposals
fi
```

### Pattern 3: Midden Structured Logging
**What:** Append structured entries to midden files for later aggregation
**When to use:** Failure capture, approach change logging
**Proposed schema:**
```yaml
# midden/build-failures.md
- timestamp: "2026-02-21T14:30:00Z"
  phase: 36
  colony: "test-colony"
  worker: "Hammer-42"
  task: "3.2"
  what_failed: "API endpoint returned 500"
  why: "Missing error handling for null user"
  what_worked: "Added null check and default response"
  error_type: "runtime"
  files_involved: ["src/api/users.js"]
```

### Pattern 4: Threshold Override
**What:** Lower threshold to 1 but keep user approval as gate
**When to use:** All wisdom types (per MEM-03)
**Implementation:** Modify threshold logic in two locations:
1. learning-observe (line 4195-4202): Change all thresholds to 1
2. learning-check-promotion (line 4257-4265): Change all thresholds to 1

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Content deduplication | Custom hash comparison | learning-observe's SHA256 hashing | Already handles collision detection |
| User selection UI | Custom input parsing | parse-selection function | Handles ranges, validation, edge cases |
| QUEEN.md writing | Direct file manipulation | queen-promote function | Handles formatting, metadata, versioning |
| Threshold visualization | Custom progress bars | generate_threshold_bar | UTF-8/ASCII fallback, colors |

## Common Pitfalls

### Pitfall 1: wisdom_type Validation Failure
**What goes wrong:** learning-observe rejects "failure" type because valid_types array doesn't include it
**Why it happens:** Current valid_types = ("philosophy" "pattern" "redirect" "stack" "decree")
**How to avoid:** Either add "failure" to valid_types OR map failures to "redirect" type
**Location:** aether-utils.sh line 4100

### Pitfall 2: Silent Failures in Build Logging
**What goes wrong:** learning-observe calls fail silently with `|| true`, masking real issues
**Why it happens:** Build.md uses `2>/dev/null || true` pattern to prevent build failures from logging errors
**How to avoid:** Log to activity log when observation recording fails: `activity-log "ERROR" "Queen" "Failed to record observation"`

### Pitfall 3: Midden Directory Permissions
**What goes wrong:** Writing to midden/ fails if directory doesn't exist
**Why it happens:** Only midden.json exists; new .md files need directory structure
**How to avoid:** Ensure `mkdir -p "$DATA_DIR/midden"` before writes

### Pitfall 4: Threshold Inconsistency
**What goes wrong:** Changing thresholds in one place but not another
**Why it happens:** Thresholds defined in both learning-observe (for return value) and learning-check-promotion (for filtering)
**How to avoid:** Modify both locations at lines 4195-4202 AND 4257-4265

### Pitfall 5: Worker Convention for Approach Changes
**What goes wrong:** Workers don't know how to log "tried X, switched to Y"
**Why it happens:** No established pattern for self-reporting approach changes
**How to avoid:** Add convention to worker prompts in build.md, include example in templates

## Code Examples

### Modified Threshold Configuration
```bash
# aether-utils.sh line ~4195 - Update all thresholds to 1
case "$wisdom_type" in
  philosophy) threshold=1 ;;  # Was 5
  pattern) threshold=1 ;;      # Was 3
  redirect) threshold=1 ;;     # Was 2
  stack) threshold=1 ;;        # Already 1
  decree) threshold=0 ;;       # Already 0
  failure) threshold=1 ;;      # NEW: Add failure type
  *) threshold=1 ;;
esac
```

### Failure Observation from Build
```bash
# In build.md Step 5.2, after detecting worker failure:
if [[ "$status" == "failed" ]]; then
  colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json)
  bash .aether/aether-utils.sh learning-observe \
    "Builder $ant_name failed on task $task_id: ${blockers[0]}" \
    "failure" \
    "$colony_name" 2>/dev/null || true

  # Also log to midden
  cat >> .aether/data/midden/build-failures.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: $phase_id
  colony: "$colony_name"
  worker: "$ant_name"
  task: "$task_id"
  what_failed: "${blockers[0]}"
  error_type: "worker_failure"
EOF
fi
```

### Silent Skip Pattern for Empty Learnings
```bash
# In continue.md Step 2.1.5 - Check before showing approval UI
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  # Show approval UI
  bash .aether/aether-utils.sh learning-approve-proposals
else
  # Silent skip - no output
  :
fi
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual learning capture | Automatic observation | Phase 36 (planned) | Colony observes without prompting |
| 5-observation threshold | 1 + user approval | Phase 36 (planned) | Faster wisdom promotion, user as gate |
| Single midden.json | Typed midden files (.md) | Phase 36 (planned) | Human-readable, parseable failures |
| Staged approval | Checkbox tick-to-approve | Phase 34 (2026-02-20) | Familiar Git-style selection |

## Open Questions

1. **Failure type mapping**
   - What we know: learning-observe validates against fixed wisdom_type list
   - What's unclear: Should we add "failure" as new type or map to "redirect"?
   - Recommendation: Add "failure" type for clarity; update valid_types array

2. **Midden file format**
   - What we know: User wants structured YAML/JSON entries
   - What's unclear: Should we use YAML frontmatter in .md or pure JSON?
   - Recommendation: YAML list format (human-readable, parseable)

3. **Test failure capture granularity**
   - What we know: Need to capture TDD red-green cycle
   - What's unclear: Log every test failure or just phase-level summary?
   - Recommendation: Log approach changes ("switched from X to Y") not every failure

4. **Worker self-reporting convention**
   - What we know: Workers need to report approach changes
   - What's unclear: Exact format for "tried X, didn't work, trying Y"
   - Recommendation: Add to worker prompt template with example

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh` lines 4087-4230 - learning-observe implementation
- `.aether/aether-utils.sh` lines 4723-4923 - learning-approve-proposals implementation
- `.claude/commands/ant/continue.md` lines 708-757 - Step 2.1.5 promotion check
- `.claude/commands/ant/build.md` lines 541-617 - Worker spawn and result handling
- `.aether/data/learning-observations.json` - Existing observations file format
- `.aether/data/midden/midden.json` - Existing midden structure

### Secondary (MEDIUM confidence)
- `.planning/phases/34-add-user-approval-ux/34-CONTEXT.md` - Phase 34 decisions
- `.planning/phases/34-add-user-approval-ux/34-01-PLAN.md` - Implementation patterns
- `.aether/docs/QUEEN.md` - Current QUEEN.md format and content

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All components exist and are operational
- Architecture: HIGH - Clear integration points identified
- Pitfalls: MEDIUM-HIGH - Based on existing code analysis

**Research date:** 2026-02-21
**Valid until:** 2026-03-21 (stable components)
