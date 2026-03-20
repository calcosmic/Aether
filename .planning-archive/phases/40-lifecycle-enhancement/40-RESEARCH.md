# Phase 40: Lifecycle Enhancement - Research

**Researched:** 2026-02-22
**Domain:** Agent Integration (Chronicler + Ambassador)
**Confidence:** HIGH

## Summary

Phase 40 integrates two specialist agents into the Aether colony lifecycle: **Chronicler** for documentation coverage auditing and **Ambassador** for external API/SDK integration. This research establishes the integration patterns, agent capabilities, and modification points needed for implementation.

**Chronicler Integration:** The Chronicler agent (`.opencode/agents/aether-chronicler.md`) is a documentation specialist that surveys API docs, READMEs, and guides. It operates with a read-only posture — no Bash tool, Edit restricted to JSDoc/TSDoc comments only. For `/ant:seal` integration, Chronicler must spawn at Step 5.5 (before ceremony display), survey documentation coverage, and report gaps without blocking the seal process.

**Ambassador Integration:** The Ambassador agent (`.opencode/agents/aether-ambassador.md`) handles external API integrations including OAuth, rate limiting, circuit breakers, and retry patterns. Unlike other agents that spawn during verification, Ambassador acts as a caste replacement — when a build task involves external APIs, the Queen spawns Ambassador instead of Builder. Ambassador returns an integration plan for a Builder to execute.

**Primary recommendation:** Follow the established agent integration pattern from Phases 38-39: conditional spawn with logging, JSON output parsing, midden integration for non-blocking findings, and clear step sequencing in command files.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LIF-01 | Chronicler spawns in `/ant:seal` Step 5.5 before ceremony display | seal.md Step 5 is "Update Milestone", Step 6 is "Write CROWNED-ANTHILL.md", Step 7 is ceremony display. Insert at 5.5. |
| LIF-02 | Chronicler surveys documentation coverage (API docs, READMEs, guides) | Chronicler agent has built-in documentation types: README, API docs, Guides, Changelogs, Code comments, Architecture docs |
| LIF-03 | Chronicler reports gaps but doesn't block seal | Follow Probe/Measurer pattern: non-blocking with midden logging |
| LIF-04 | Ambassador replaces Builder when task involves external API/SDK/OAuth | build.md Step 5.1 spawns Builders — add keyword detection for API/SDK/OAuth tasks |
| LIF-05 | Ambassador handles rate limiting, circuit breakers, retry patterns | Ambassador agent has these patterns built into its role definition |
| LIF-06 | Ambassador returns integration plan for Builder to execute | Ambassador outputs JSON with integration plan; Builder executes in subsequent wave |
</phase_requirements>

## Standard Stack

### Core Integration Pattern
| Component | Source | Purpose | Pattern |
|-----------|--------|---------|---------|
| Agent spawn | Task tool with `subagent_type` | Spawn specialist agents | Conditional based on triggers |
| Spawn tracking | `aether-utils.sh spawn-log` | Track agent lifecycle | Log before spawn, complete after |
| Swarm display | `aether-utils.sh swarm-display-*` | Visual feedback | Update status throughout |
| Midden logging | `aether-utils.sh midden-write` | Non-blocking findings | Category + message + source |
| JSON output | Agent return format | Structured data exchange | Parse for gate decisions |

### Supporting Utilities
| Utility | Location | Purpose | When to Use |
|---------|----------|---------|-------------|
| `generate-ant-name` | aether-utils.sh | Name generation | Before every agent spawn |
| `spawn-log` | aether-utils.sh | Lifecycle tracking | Log spawn events |
| `spawn-complete` | aether-utils.sh | Completion tracking | Log agent completion |
| `midden-write` | aether-utils.sh | Finding persistence | Log non-blocking findings |
| `activity-log` | aether-utils.sh | Audit trail | Log significant actions |

## Architecture Patterns

### Pattern 1: Conditional Agent Spawn (Chronicler in seal.md)

**What:** Agent spawns only when conditions are met, with graceful skip otherwise.

**When to use:** For non-essential agents that provide value but shouldn't block workflow.

**Example from Phase 39-01 (Probe):**
```markdown
#### Step 1.5.1: Probe Coverage Agent (Conditional)

**Test coverage improvement — runs when coverage < 80% AND tests pass.**

1. **Check coverage threshold condition:**
   - If coverage_percent >= 80%: Skip Probe silently
   - If coverage_percent < 80% AND tests passed: Proceed to spawn Probe

2. **If skipping Probe:**
```
🧪🐜 Probe: Coverage at {coverage_percent}% — {reason_for_skip}
```
Continue to Phase 5: Secrets Scan.

3. **If spawning Probe:**
   a. Generate Probe name and dispatch
   b. Update swarm display
   c. Display spawn message
   d. Spawn Probe agent with Task tool
   e. Parse JSON output
   f. Log findings to midden
   g. Continue (non-blocking)
```

**Chronicler adaptation:**
- Check if docs exist (README.md, docs/ directory, API docs)
- If no documentation found: Skip silently
- If documentation exists: Spawn Chronicler to survey coverage
- Always non-blocking — seal proceeds regardless

### Pattern 2: Caste Replacement (Ambassador in build.md)

**What:** Replace default caste (Builder) with specialist caste (Ambassador) for specific task types.

**When to use:** When a task requires specialized expertise that differs from standard implementation.

**Detection logic from build.md Step 5.1:**
```bash
# In task analysis, check for API/integration keywords:
for task in wave_tasks; do
  if [[ "$task" =~ (API|SDK|OAuth|integration|webhook|external) ]]; then
    caste="ambassador"
  else
    caste="builder"
  fi
done
```

**Ambassador spawn flow:**
1. Detect API/SDK/OAuth keywords in task description
2. Spawn Ambassador instead of Builder
3. Ambassador researches external API, designs integration
4. Ambassador returns integration plan JSON
5. Queen spawns Builder in next wave to execute plan

### Pattern 3: Non-Blocking with Midden Logging

**What:** Agent findings are logged for review but never block workflow progression.

**When to use:** For advisory agents that identify issues but shouldn't halt progress.

**Example from Phase 38-01 (Gatekeeper high CVEs):**
```bash
# Log to midden for later review
bash .aether/aether-utils.sh midden-write "security" "High CVEs found: ${high_cve_count}" "gatekeeper"
```

**Chronicler adaptation:**
```bash
# Log documentation gaps to midden
for gap in $gaps; do
  bash .aether/aether-utils.sh midden-write "documentation" "${gap}" "chronicler"
done
```

### Pattern 4: Sequential Agent Waves

**What:** Agents spawn in sequence with dependencies, passing context between waves.

**When to use:** When one agent's output is required input for another.

**Ambassador-Builder sequence:**
```
Wave 1: Ambassador (research + design)
  ↓ (returns integration plan)
Wave 2: Builder (execute plan)
```

### Pattern 5: JSON Output Contract

**What:** Agents return structured JSON for programmatic processing.

**Standard fields:**
```json
{
  "ant_name": "string",
  "caste": "string",
  "status": "completed|failed|blocked",
  "summary": "string",
  "tool_count": 0,
  "blockers": []
}
```

**Chronicler-specific fields:**
```json
{
  "documentation_created": [],
  "documentation_updated": [],
  "pages_documented": 0,
  "coverage_percent": 0,
  "gaps_identified": []
}
```

**Ambassador-specific fields:**
```json
{
  "endpoints_integrated": [],
  "authentication_method": "",
  "rate_limits_handled": true,
  "error_scenarios_covered": [],
  "integration_plan": {
    "files_to_create": [],
    "files_to_modify": [],
    "env_vars_needed": [],
    "test_strategy": ""
  }
}
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Agent lifecycle tracking | Custom logging | `spawn-log` / `spawn-complete` | Integrated with colony state |
| Finding persistence | Custom files | `midden-write` | Structured, searchable, reviewable |
| Name generation | Random strings | `generate-ant-name` | Consistent colony naming |
| Visual feedback | Print statements | `swarm-display-*` | Real-time status across all agents |
| Documentation coverage analysis | Manual checklists | Chronicler agent | Automated, consistent, thorough |
| API integration patterns | Ad-hoc code | Ambassador agent | Tested patterns for auth, retries, circuit breakers |

**Key insight:** The Aether system provides utilities for common agent integration tasks. Using these ensures consistency with existing gates (Gatekeeper, Auditor, Probe, Measurer) and proper integration with colony tracking.

## Common Pitfalls

### Pitfall 1: Blocking Non-Critical Agents

**What goes wrong:** Making Chronicler block seal ceremony when documentation gaps are found.

**Why it happens:** Confusion about which agents should block vs. advise.

**How to avoid:** Follow LIF-03 requirement explicitly — Chronicler is advisory only. Document this in the step description.

**Warning signs:** Step description says "check" instead of "survey and report".

### Pitfall 2: Missing Spawn Logging

**What goes wrong:** Agent spawns but doesn't appear in spawn tree or activity log.

**Why it happens:** Forgetting to call `spawn-log` before Task tool spawn.

**How to avoid:** Always use the three-call pattern:
1. `generate-ant-name` — get name
2. `spawn-log` — log the spawn
3. `swarm-display-update` — visual feedback
4. Task tool — actual spawn

### Pitfall 3: Ambassador-Builder Handoff Failure

**What goes wrong:** Ambassador returns plan but Builder doesn't receive it or plan format is wrong.

**Why it happens:** Unclear JSON contract between agents.

**How to avoid:** Define explicit integration plan schema in Ambassador output. Builder prompt must reference this schema.

### Pitfall 4: Step Numbering Drift

**What goes wrong:** Inserting Step 5.5 breaks subsequent step references.

**Why it happens:** seal.md has hardcoded step numbers in multiple places.

**How to avoid:** Search for all "Step X:" references after insertion point and increment accordingly.

### Pitfall 5: Chronicler Tool Misconfiguration

**What goes wrong:** Chronicler spawns with Bash tool despite Phase 30 decision restricting it.

**Why it happens:** Not respecting prior architectural decisions about agent capabilities.

**How to avoid:** Honor Phase 30-02 decision: Chronicler has no Bash tool, Edit restricted to JSDoc/TSDoc comments.

## Code Examples

### Chronicler Spawn in seal.md (Step 5.5)

```markdown
### Step 5.5: Chronicler Documentation Survey (Conditional)

**Documentation coverage audit — runs when documentation exists.**

1. **Check for documentation:**
   Run using the Bash tool with description "Checking for documentation...":
   ```bash
   doc_count=0
   [[ -f README.md ]] && ((doc_count++))
   [[ -d docs ]] && doc_count=$((doc_count + $(find docs -type f | wc -l)))
   [[ -f API.md ]] && ((doc_count++))
   echo "{\"doc_count\": $doc_count}"
   ```

2. **If no documentation found:**
   ```
   📝 Chronicler: No documentation found — skipping survey
   ```
   Continue to Step 6.

3. **If documentation exists:**

   a. Generate Chronicler name:
   Run using the Bash tool with description "Naming chronicler...": `bash .aether/aether-utils.sh generate-ant-name "chronicler"`

   b. Log spawn:
   Run using the Bash tool with description "Dispatching chronicler...": `bash .aether/aether-utils.sh spawn-log "Queen" "chronicler" "{chronicler_name}" "Documentation coverage survey"`

   c. Update display:
   Run using the Bash tool with description "Updating display...": `bash .aether/aether-utils.sh swarm-display-update "{chronicler_name}" "chronicler" "surveying" "Documentation coverage survey" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 0`

   d. Display:
   ```
   📝 Chronicler {chronicler_name} spawning — Surveying documentation coverage...
   ```

   e. Spawn Chronicler agent:
   Use Task tool with `subagent_type="aether-chronicler"`, include `description: "📝 Chronicler {Name}: Documentation coverage survey"`:

   ```xml
   <mission>
   Survey documentation coverage for the colony.
   </mission>

   <work>
   1. Check README.md for completeness (quick start, installation, usage)
   2. Check docs/ directory for guides and API documentation
   3. Check inline code comments (JSDoc/TSDoc)
   4. Identify documentation gaps
   5. Report coverage percentage and missing areas
   </work>

   <context>
   Colony goal: {goal}
   Phase count: {total_phases}
   Documentation found: {doc_count} files
   </context>

   <output>
   Return ONLY this JSON:
   {
     "ant_name": "{Chronicler-Name}",
     "caste": "chronicler",
     "status": "completed|failed|blocked",
     "summary": "What was surveyed",
     "coverage_percent": 0,
     "gaps_identified": ["list of missing documentation"],
     "pages_documented": 0,
     "tool_count": 0
   }
   </output>
   ```

   f. Parse JSON output and log gaps to midden:
   For each gap in `gaps_identified`:
   Run using the Bash tool with description "Logging documentation gap...": `bash .aether/aether-utils.sh midden-write "documentation" "{gap}" "chronicler"`

   g. Log completion:
   Run using the Bash tool with description "Recording chronicler completion...": `bash .aether/aether-utils.sh spawn-complete "{chronicler_name}" "completed" "Survey complete: {coverage_percent}% coverage"`

   h. Display:
   ```
   📝 Chronicler: Documentation survey complete — {coverage_percent}% coverage, {gap_count} gaps logged to midden
   ```

4. **Continue to Step 6** (never block — LIF-03 requirement)
```

### Ambassador Caste Replacement in build.md (Step 5.1 modification)

```markdown
### Step 5.1: Spawn Wave 1 Workers (Parallel)

**CRITICAL: Spawn ALL Wave 1 workers in a SINGLE message using multiple Task tool calls.**

**Task Analysis with Caste Selection:**

For each task in Wave 1:
1. Check for API/integration keywords:
   Run using the Bash tool with description "Analyzing task type...":
   ```bash
   task_desc="{task_description}"
   if [[ "${task_desc,,}" =~ (api|sdk|oauth|integration|webhook|external|third.party) ]]; then
     echo '{"caste": "ambassador", "reason": "External API integration detected"}'
   else
     echo '{"caste": "builder", "reason": "Standard implementation task"}'
   fi
   ```

2. **If caste is "ambassador":**

   a. Generate Ambassador name:
   Run using the Bash tool with description "Naming ambassador...": `bash .aether/aether-utils.sh generate-ant-name "ambassador"`

   b. Log spawn:
   Run using the Bash tool with description "Dispatching ambassador...": `bash .aether/aether-utils.sh spawn-log "Queen" "ambassador" "{ambassador_name}" "API integration design: {task_description}"`

   c. Display:
   ```
   🔌 Ambassador {ambassador_name} spawning — Designing API integration...
   ```

   d. Spawn Ambassador agent:
   Use Task tool with `subagent_type="aether-ambassador"`, include `description: "🔌 Ambassador {Name}: API integration design"`:

   ```xml
   <mission>
   Design integration with external API/SDK for the assigned task.
   </mission>

   <work>
   1. Research the external API/SDK thoroughly
   2. Design integration patterns (client wrapper, circuit breaker, retry)
   3. Plan authentication approach (OAuth, API keys, etc.)
   4. Design rate limiting and error handling
   5. Create integration plan for Builder execution
   </work>

   <context>
   Task: {task_description}
   Colony goal: {colony_goal}
   </context>

   <output>
   Return ONLY this JSON:
   {
     "ant_name": "{Ambassador-Name}",
     "caste": "ambassador",
     "status": "completed|failed|blocked",
     "summary": "Integration design complete",
     "endpoints_integrated": [],
     "authentication_method": "",
     "rate_limits_handled": true,
     "error_scenarios_covered": ["timeout", "auth", "rate_limit"],
     "integration_plan": {
       "files_to_create": [],
       "files_to_modify": [],
       "env_vars_needed": [],
       "dependencies_to_install": [],
       "test_strategy": ""
     },
     "tool_count": 0,
     "blockers": []
   }
   </output>
   ```

   e. Store integration plan for next wave:
   The `integration_plan` object will be passed to the Builder in Wave 2.

3. **If caste is "builder":**
   (Standard builder spawn as existing)
```

### Builder Execution of Ambassador Plan (Wave 2)

```markdown
### Step 5.3: Spawn Wave 2 Workers (Sequential)

**For tasks that had Ambassador design in Wave 1:**

Spawn Builder with Ambassador's integration plan:

```xml
<mission>
Execute the integration plan designed by Ambassador {ambassador_name}.
</mission>

<work>
1. Review the integration plan below
2. Create/modify files as specified
3. Implement authentication as designed
4. Add rate limiting and error handling
5. Write tests per test strategy
</work>

<integration_plan>
{json_output_from_ambassador}
</integration_plan>

<context>
Original task: {task_description}
Ambassador: {ambassador_name}
</context>
```
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual documentation review | Chronicler automated survey | Phase 40 | Consistent coverage auditing |
| Builder handles API integration | Ambassador designs, Builder executes | Phase 40 | Specialized expertise for external integrations |
| Ad-hoc OAuth/retry patterns | Standardized Ambassador patterns | Phase 40 | Reliable, tested integration patterns |
| Documentation gaps discovered late | Chronicler finds gaps at seal | Phase 40 | Earlier awareness of documentation debt |

**Integration pattern evolution:**
- Phase 38: Gatekeeper (conditional, blocking on critical) + Auditor (always, blocking on score)
- Phase 39: Probe (conditional on coverage, non-blocking) + Measurer (conditional on keywords, non-blocking)
- Phase 40: Chronicler (conditional on docs, non-blocking) + Ambassador (caste replacement for API tasks)

## Open Questions

1. **Chronicler Documentation Detection**
   - What we know: Need to check for README.md, docs/, API.md
   - What's unclear: Threshold for "documentation exists" — 1 file? 3 files?
   - Recommendation: Use doc_count >= 1 as threshold; skip only if absolutely no docs found

2. **Ambassador Keyword Detection**
   - What we know: Keywords like "API", "SDK", "OAuth" should trigger Ambassador
   - What's unclear: False positives (e.g., "API documentation" is docs task, not integration)
   - Recommendation: Require keyword + context check (task modifies code, not just docs)

3. **Integration Plan Format**
   - What we know: Ambassador returns structured JSON with plan
   - What's unclear: Exact schema for all integration types (REST, GraphQL, SDK, webhook)
   - Recommendation: Start with flexible schema, evolve based on usage

4. **Chronicler Coverage Calculation**
   - What we know: Chronicler reports coverage_percent
   - What's unclear: How coverage is calculated (files? lines? topics?)
   - Recommendation: Document that coverage is heuristic-based (files with docs / total files)

## Sources

### Primary (HIGH confidence)
- `.opencode/agents/aether-chronicler.md` — Agent capabilities, output format, boundaries
- `.opencode/agents/aether-ambassador.md` — Agent capabilities, integration patterns, output format
- `.claude/commands/ant/seal.md` — Current seal command structure, step sequencing
- `.claude/commands/ant/build.md` — Current build command structure, wave spawning
- `.planning/phases/38-security-gates/38-01-SUMMARY.md` — Gatekeeper integration pattern
- `.planning/phases/38-security-gates/38-02-SUMMARY.md` — Auditor integration pattern
- `.planning/phases/39-quality-coverage/39-01-SUMMARY.md` — Probe integration pattern (conditional, non-blocking)
- `.planning/phases/39-quality-coverage/39-02-SUMMARY.md` — Measurer integration pattern (keyword detection)
- `.planning/REQUIREMENTS.md` — LIF-01 through LIF-06 requirements

### Secondary (MEDIUM confidence)
- Phase 30 decisions in STATE.md — Chronicler has no Bash tool, Edit restricted to comments
- Agent integration pattern from Phases 38-39 — Spawn logging, midden integration, JSON output

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Established patterns from Phases 38-39
- Architecture: HIGH — Clear agent definitions and integration points
- Pitfalls: MEDIUM — Based on pattern analysis, limited historical data

**Research date:** 2026-02-22
**Valid until:** 30 days (stable patterns)

**Files to modify:**
1. `.claude/commands/ant/seal.md` — Add Step 5.5 (Chronicler), renumber subsequent steps
2. `.claude/commands/ant/build.md` — Modify Step 5.1 (Ambassador caste replacement), add Wave 2 for plan execution

**Dependencies:**
- Requires Phase 38-39 patterns to be in place (midden-write, spawn-log utilities)
- Chronicler and Ambassador agent definitions must exist in `.opencode/agents/`
