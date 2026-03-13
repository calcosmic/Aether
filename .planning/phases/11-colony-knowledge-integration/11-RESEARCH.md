# Phase 11: Colony Knowledge Integration and Output Polish - Research

**Researched:** 2026-03-13
**Domain:** Colony instinct/learning promotion from oracle research, research strategy templates, adaptive structured output, topic-aware synthesis formatting
**Confidence:** HIGH

## Summary

Phase 11 closes two gaps in the oracle system. First, oracle research currently produces findings in `.aether/oracle/` but has no bridge back to colony knowledge -- high-confidence findings sit in synthesis.md without becoming instincts or learnings that benefit future colonies. The infrastructure for promoting knowledge already exists (`instinct-create`, `learning-promote`, `learning-observe`, `memory-capture`, `queen-promote`, `learning-promote-auto`) and is well-tested (integration tests in `instinct-pipeline.test.js`, `learning-pipeline.test.js`, `wisdom-promotion.test.js`). The missing piece is a deliberate post-completion step that reads oracle research findings and calls these APIs to promote them.

Second, the oracle's synthesis pass currently produces a one-size-fits-all report structure (Executive Summary, Findings by Question, Open Questions, Methodology Notes, Sources). Different research types have fundamentally different output needs: a technology evaluation should produce a comparison matrix and recommendation; a bug investigation should produce root cause analysis and fix steps; an architecture review should produce dependency maps and risk areas. Phase 11 adds research strategy templates that configure both the initial question decomposition and the final output structure. Templates are selectable in the wizard and stored in state.json as a `template` field. The synthesis pass prompt reads the template type and produces topic-appropriate output sections.

The design principle throughout is: lightweight orchestrator additions + prompt changes. The oracle.sh gains one new function (`promote_to_colony`) that reads completed plan.json findings and calls existing aether-utils.sh subcommands. The wizard gains a template selection question and emits the template type into state.json. The synthesis prompt in `build_synthesis_prompt` reads the template type and includes topic-appropriate section requirements. No new dependencies, no new files beyond tests.

**Primary recommendation:** Add a `promote_to_colony` function in oracle.sh (called after synthesis pass completes, guarded by user confirmation in the slash command); add a `template` field to state.json and a wizard question for template selection; update `build_synthesis_prompt` to emit template-specific output section requirements. Update `validate-oracle-state` for new fields.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| COLN-01 | High-confidence research findings can be promoted to colony instincts/learnings after completion | The `promote_to_colony` function reads plan.json questions with confidence >= 80% and status "answered", extracts key findings, and calls `instinct-create` and `learning-promote` subcommands. This is a post-completion step triggered by the user via `/ant:oracle promote` (new subcommand), not automatic. Existing APIs: `instinct-create --trigger --action --confidence --domain --source --evidence`, `learning-promote <content> <source_project> <source_phase> [tags]`, `memory-capture learning <content> <wisdom_type>`. |
| COLN-02 | Pre-built research strategy templates for common patterns (tech eval, architecture review, bug investigation, best practices) | Four template definitions in the wizard, stored as `template` field in state.json. Each template provides: (1) a set of default sub-questions appropriate to the type, (2) output section directives for the synthesis pass. Templates: `tech-eval` (comparison matrix, pros/cons, recommendation), `architecture-review` (component analysis, dependency risks, scalability assessment), `bug-investigation` (reproduction steps, root cause analysis, fix recommendations), `best-practices` (current state assessment, gap analysis, actionable recommendations). A fifth option `custom` (default) preserves current behavior. |
| OUTP-01 | Final output is a structured, synthesized report with sections, executive summary, and findings organized by sub-question | The existing `build_synthesis_prompt` already requires Executive Summary, Findings by Question (with confidence %), Open Questions, Methodology Notes, and Sources. Phase 11 extends this with confidence-grouped findings within each question section: high-confidence findings (80%+) listed first with full citations, medium-confidence (50-79%) listed with caveats, low-confidence (<50%) listed as tentative. This grouping is a prompt addition to build_synthesis_prompt. |
| OUTP-03 | Output structure adapts to the specific research topic (not one-size-fits-all template) | The `template` field in state.json drives synthesis output structure. `build_synthesis_prompt` reads the template and emits type-specific section requirements. A tech-eval gets Comparison Matrix + Recommendation; a bug investigation gets Root Cause Analysis + Fix Plan; an architecture review gets Component Map + Risk Assessment; best-practices gets Gap Analysis + Action Items. The `custom` template preserves the current generic structure. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| oracle.sh | Bash script (~850 lines) | Add `promote_to_colony` function, update `build_synthesis_prompt` with template-aware sections | Existing orchestrator; all Phase 11 changes go here |
| oracle.md | Prompt file (~169 lines) | No changes needed for Phase 11 (synthesis prompt handles output structure) | Existing; template directives go in build_synthesis_prompt, not oracle.md |
| aether-utils.sh | ~9,808 lines | Existing APIs: `instinct-create`, `learning-promote`, `memory-capture`; update `validate-oracle-state` for new fields | Colony knowledge infrastructure already exists |
| oracle wizard (.claude/commands/ant/oracle.md) | Current | Add template selection question (Question 7), add `promote` subcommand routing | Existing wizard pattern |
| oracle wizard (.opencode/commands/ant/oracle.md) | Current | Mirror template and promote changes for OpenCode parity | Existing parity requirement |
| jq | 1.6+ | Read plan.json findings for promotion, template type from state.json | Project standard |
| ava | ^6.0.0 | Unit tests for promotion logic and template handling | Project standard test runner |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| bash test framework | tests/bash/test-helpers.sh | Bash integration tests for promote_to_colony and template-aware synthesis | Testing new oracle.sh functions |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Template type in state.json | Separate templates/ directory with JSON template files | Over-engineering for 4 templates; state.json field is simpler and follows existing pattern (strategy field) |
| User-triggered promote subcommand | Auto-promote on completion | Prior decision explicitly says "deliberate user-triggered step"; auto-promote would violate oracle's non-invasive guarantee |
| Synthesis prompt template switching | Separate synthesis prompt files per template | Maintenance burden; a single function with case branches is cleaner (same pattern as phase directives) |
| promote_to_colony in oracle.sh | New slash command handler entirely | The promotion logic needs to read oracle state files; keeping it in oracle.sh is natural |

## Architecture Patterns

### Recommended Project Structure

No new files created beyond tests -- all changes modify existing files:

```
.aether/oracle/
  oracle.sh           # MODIFY: add promote_to_colony function, update build_synthesis_prompt with template logic

.aether/aether-utils.sh  # MODIFY: update validate-oracle-state for template field in state.json

.claude/commands/ant/oracle.md     # MODIFY: add template wizard question, add promote subcommand routing
.opencode/commands/ant/oracle.md   # MODIFY: mirror wizard and subcommand changes

tests/
  unit/oracle-colony.test.js       # NEW: ava tests for promotion logic and template-aware synthesis
  bash/test-oracle-colony.sh       # NEW: bash integration tests for promote and template validation
```

### Pattern 1: promote_to_colony Function in oracle.sh

**What:** A function that reads completed oracle plan.json, extracts high-confidence findings, and calls existing colony knowledge APIs to promote them to instincts and learnings.

**When to use:** Called from the `/ant:oracle promote` subcommand after research is complete (status "complete" or "stopped").

**Design:**

```bash
# Promote high-confidence oracle findings to colony knowledge
# Reads plan.json, extracts findings from questions with confidence >= 80%
# Calls instinct-create and learning-promote for each qualifying finding
promote_to_colony() {
  local plan_file="$1"
  local state_file="$2"
  local aether_root="$3"
  local utils="$aether_root/.aether/aether-utils.sh"

  # Verify state is complete or stopped
  local status
  status=$(jq -r '.status // "active"' "$state_file" 2>/dev/null || echo "active")
  if [ "$status" = "active" ]; then
    echo "ERROR: Research is still active. Wait for completion or stop the oracle first."
    return 1
  fi

  # Verify colony state exists (promotion requires active colony)
  if [ ! -f "$aether_root/.aether/data/COLONY_STATE.json" ]; then
    echo "ERROR: No active colony. Run /ant:init first."
    return 1
  fi

  local topic
  topic=$(jq -r '.topic // "unknown"' "$state_file" 2>/dev/null || echo "unknown")

  # Extract high-confidence answered questions (>= 80%)
  local questions
  questions=$(jq -c '[.questions[] | select(.status == "answered" and .confidence >= 80)]' "$plan_file" 2>/dev/null || echo "[]")

  local count
  count=$(echo "$questions" | jq 'length')

  if [ "$count" -eq 0 ]; then
    echo "No findings meet promotion threshold (answered + 80%+ confidence)."
    echo "Lower-confidence findings remain in .aether/oracle/synthesis.md for reference."
    return 0
  fi

  echo "Promoting $count high-confidence findings to colony knowledge..."

  local promoted=0
  local failed=0

  # Process each qualifying question
  echo "$questions" | jq -c '.[]' | while IFS= read -r question; do
    local q_text q_confidence
    q_text=$(echo "$question" | jq -r '.text')
    q_confidence=$(echo "$question" | jq -r '.confidence')

    # Create instinct from the research finding
    bash "$utils" instinct-create \
      --trigger "When researching: $q_text" \
      --action "Oracle found (${q_confidence}% confidence): $(echo "$question" | jq -r '[.key_findings[].text // .key_findings[]] | join("; ")' | head -c 200)" \
      --confidence "$(echo "scale=2; $q_confidence / 100" | bc)" \
      --domain "research" \
      --source "oracle:$topic" \
      --evidence "Oracle research: $q_text" 2>/dev/null || true

    # Promote as learning
    bash "$utils" learning-promote \
      "Oracle: $q_text -- $(echo "$question" | jq -r '[.key_findings[].text // .key_findings[]] | first')" \
      "oracle" \
      "oracle-research" \
      "oracle,research" 2>/dev/null || true

    # Record via memory-capture for observation tracking
    bash "$utils" memory-capture learning \
      "Oracle research finding: $q_text (${q_confidence}%)" \
      "pattern" \
      "oracle:promote" 2>/dev/null || true

    promoted=$((promoted + 1))
  done

  echo ""
  echo "Promoted $promoted findings to colony knowledge."
  echo "  - Instincts created in COLONY_STATE.json"
  echo "  - Learnings stored in learnings.json"
  echo "  - Observations tracked for wisdom promotion"
}
```

**Design rationale:**

- **80% confidence threshold:** Aligns with the confidence rubric: 80-95% means "Primary sources verified, contradictions resolved, limitations known." Below 80% is useful research but not reliable enough to encode as colony instinct.
- **Requires completed research:** Status must be "complete" or "stopped". Active research should not be promoted mid-stream.
- **Uses existing APIs:** `instinct-create` handles deduplication (boosts confidence if trigger+action matches), enforces 30-instinct cap (evicts lowest confidence). `learning-promote` stores in learnings.json with 50-learning cap. `memory-capture` feeds the observation pipeline for queen-promote.
- **Truncation:** Finding text is truncated to 200 chars for instinct action field. Detailed findings remain in synthesis.md.
- **Graceful failure:** Each promotion call is guarded with `|| true`. If one fails, others continue.

### Pattern 2: Research Strategy Templates

**What:** Pre-defined research templates that configure both question decomposition and output structure for common research patterns.

**When to use:** Selected in the wizard (Question 7). Stored in state.json as `template` field.

**Template definitions:**

| Template | Default Questions | Output Sections |
|----------|------------------|-----------------|
| `tech-eval` | "What problem does X solve?", "How does X compare to alternatives?", "What are X's known limitations?", "What is the adoption/community status?", "What is the migration/integration path?" | Comparison Matrix, Pros/Cons, Recommendation, Migration Path |
| `architecture-review` | "What are the main components and their responsibilities?", "What are the dependency relationships?", "Where are the risk areas (coupling, complexity, single points of failure)?", "How does it handle scale/growth?", "What would an expert change?" | Component Map, Dependency Analysis, Risk Assessment, Scalability Analysis, Improvement Recommendations |
| `bug-investigation` | "What is the exact failure behavior?", "What are the reproduction conditions?", "What is the root cause?", "What are possible fixes and their tradeoffs?", "Are there related issues?" | Reproduction Steps, Root Cause Analysis, Fix Recommendations, Related Issues |
| `best-practices` | "What is current industry best practice for X?", "How does our implementation compare?", "What gaps exist between our approach and best practice?", "What is the recommended improvement path?" | Current State Assessment, Best Practice Benchmark, Gap Analysis, Action Plan |
| `custom` (default) | User-defined (current behavior -- AI decomposes freely) | Generic structure (current behavior -- Executive Summary, Findings by Question, etc.) |

**state.json extension:**
```json
{
  "version": "1.1",
  "topic": "...",
  "template": "tech-eval",
  ...
}
```

**Wizard Question 7 (new):**
```
What type of research is this?
```
Options:
1. **Technology evaluation** -- Compare and evaluate a technology, library, or tool
2. **Architecture review** -- Analyze system design, components, and dependencies
3. **Bug investigation** -- Track down and understand a specific bug or issue
4. **Best practices** -- Research recommended approaches for a domain or technique
5. **Custom research** -- Free-form research (Oracle decomposes the topic as it sees fit)

**Question ordering note:** This question should be asked BEFORE Question 2 (Research Depth) and AFTER Question 1 (Research Topic), because the template type informs the depth recommendation. For example, bug investigations are typically "Quick scan" while architecture reviews are "Standard" or "Deep dive". Current wizard question numbering becomes: Q1 Topic, Q2 Template (new), Q3 Depth, Q4 Confidence, Q5 Scope, Q6 Strategy, Q7 Focus Areas.

**Template-aware question generation:** When template is not `custom`, the wizard pre-populates plan.json with the template's default questions instead of having the AI decompose freely. The user still gets the configured question set written to plan.json -- the template just provides a sensible starting point.

### Pattern 3: Template-Aware Synthesis Prompt

**What:** `build_synthesis_prompt` reads the `template` field from state.json and emits template-specific output section requirements.

**When to use:** Every synthesis pass (converged, stopped, max_iterations, interrupted).

**Design:**

```bash
build_synthesis_prompt() {
  local reason="$1"

  # Read template type from state.json
  local template
  template=$(jq -r '.template // "custom"' "$STATE_FILE" 2>/dev/null || echo "custom")

  cat <<SYNTHESIS_DIRECTIVE
## SYNTHESIS PASS (Final Report)

This is the final pass. The oracle loop has ended (reason: $reason).
Produce the best possible research report from the current state.

Read ALL of these files:
- .aether/oracle/state.json -- session metadata
- .aether/oracle/plan.json -- questions, findings, confidence, AND sources registry
- .aether/oracle/synthesis.md -- accumulated findings
- .aether/oracle/gaps.md -- remaining unknowns

If any state file is unreadable, skip it and work with what you have.

Then REWRITE synthesis.md as a structured final report.

SYNTHESIS_DIRECTIVE

  # Emit template-specific sections
  case "$template" in
    tech-eval)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: what was evaluated, key conclusion, recommendation
2. **Comparison Matrix** -- Table comparing the evaluated technology against alternatives on key dimensions (performance, community, learning curve, maturity, ecosystem)
3. **Pros and Cons** -- Bullet lists of advantages and disadvantages with evidence citations
4. **Adoption Assessment** -- Community size, maintenance status, release cadence, corporate backing
5. **Migration/Integration Path** -- Steps to adopt, estimated effort, risks
6. **Recommendation** -- Clear recommendation with confidence level and conditions/caveats
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    architecture-review)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: system overview, key findings, critical risks
2. **Component Map** -- List of major components with responsibilities and boundaries
3. **Dependency Analysis** -- How components connect, coupling assessment, external dependencies
4. **Risk Assessment** -- Single points of failure, complexity hotspots, scaling bottlenecks
5. **Scalability Analysis** -- Current capacity, growth limitations, scaling strategy
6. **Improvement Recommendations** -- Prioritized list of architectural improvements
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    bug-investigation)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 1-2 paragraphs: bug description, root cause, recommended fix
2. **Reproduction Steps** -- Exact steps to reproduce, environment details, frequency
3. **Root Cause Analysis** -- What causes the bug, code paths involved, why it was introduced
4. **Impact Assessment** -- Who is affected, severity, data loss risk
5. **Fix Recommendations** -- Proposed fixes ranked by safety and effort, with tradeoffs
6. **Related Issues** -- Similar bugs, upstream/downstream effects, regression risk
7. **Open Questions** -- Remaining gaps
8. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    best-practices)
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs: domain overview, current state assessment, key recommendations
2. **Best Practice Benchmark** -- What industry/community consensus considers best practice, with evidence
3. **Current State Assessment** -- How the subject compares to best practice (strengths and gaps)
4. **Gap Analysis** -- Specific gaps between current state and best practice, prioritized by impact
5. **Action Plan** -- Ordered steps to close gaps, estimated effort, quick wins highlighted
6. **Open Questions** -- Remaining gaps
7. **Sources** -- All sources with inline citations [S1], [S2] format

TEMPLATE
      ;;
    *)
      # custom or unrecognized: use existing generic structure
      cat <<'TEMPLATE'
### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs summarizing what was found
2. **Findings by Question** -- organized by sub-question, with confidence %. Use inline citations [S1], [S2] linking findings to their sources. Flag single-source findings with (single source) marker.
3. **Open Questions** -- remaining gaps with explanation of what is unknown and why
4. **Methodology Notes** -- how many iterations, which phases completed
5. **Sources** -- List ALL sources from plan.json sources registry: Format: [S1] Title -- URL (accessed: date). Group by type (documentation, blog, codebase, etc.). Note total source count and multi-source coverage percentage.

TEMPLATE
      ;;
  esac

  # Common directives for all templates
  cat <<'COMMON'

### Confidence Grouping:
Within each findings section, group findings by confidence level:
- **High confidence (80%+)** -- list first with full citations
- **Medium confidence (50-79%)** -- list with caveats noted
- **Low confidence (<50%)** -- list as tentative/unverified

Use inline citations [S1], [S2] linking findings to their sources.
Flag single-source findings with (single source) marker.

Also update state.json: set status to "complete" if reason is "converged",
or "stopped" otherwise.

COMMON

  # Append the base oracle.md for tool access and rules
  cat "$SCRIPT_DIR/oracle.md"
}
```

**Design rationale:**

- **Case branch, not separate files:** Same pattern as the phase directives (survey/investigate/synthesize/verify) in `build_oracle_prompt`. Keeps all synthesis logic in one function.
- **Common confidence grouping:** All templates get the confidence grouping directive (OUTP-01 requirement). This is emitted after the template-specific sections.
- **Fallback to current behavior:** The `custom` template (default) preserves the exact current synthesis structure. No existing behavior changes unless a template is explicitly selected.

### Pattern 4: Promote Subcommand in Wizard

**What:** A new `promote` subcommand in the oracle wizard that reads completed research and promotes findings to colony knowledge.

**When to use:** After oracle research is complete (`/ant:oracle promote`).

**Wizard routing addition (Step 0):**
```
2. **If remaining arguments is exactly `promote`** -- go to Step 0d: Promote Findings
```

**Step 0d implementation:**
```markdown
### Step 0d: Promote Findings to Colony

Check if `.aether/oracle/state.json` exists and research is complete.

If state.json does not exist or status is "active":
  Output error: "No completed research to promote. Run /ant:oracle first."
  Stop here.

Read plan.json and count findings with confidence >= 80% and status "answered".

Display summary:
  Oracle Research: <topic>
  Status: <status>
  High-confidence findings: <count> (answering <count> questions)

  These findings will be promoted to:
  - Colony instincts (COLONY_STATE.json)
  - Colony learnings (learnings.json)
  - Observation pipeline (for queen-promote)

Then ask user for confirmation via AskUserQuestion:
  "Promote these findings to colony knowledge?"
  1. Yes, promote all high-confidence findings
  2. No, skip promotion

If yes: call promote_to_colony function from oracle.sh
If no: output "Promotion skipped. Findings remain in .aether/oracle/synthesis.md."
```

**Why user-triggered:** The prior decision explicitly says "deliberate user-triggered step." The oracle's Non-Invasive Guarantee states it "NEVER touches COLONY_STATE.json." The promote subcommand is the bridge: it runs after research is complete, asks for confirmation, and then crosses the boundary to write to colony state. This maintains the guarantee during research while enabling knowledge transfer after.

### Pattern 5: validate-oracle-state Extension

**What:** Update the state.json validation to accept the new `template` field.

**When to use:** Post-iteration validation, tests.

**Addition to existing validation:**
```bash
# After existing strategy and focus_areas checks:
if has("template") then
  enum("template";["tech-eval","architecture-review","bug-investigation","best-practices","custom"])
else "pass"
end
```

**Backward compatibility:** The `template` field is optional. Pre-Phase-11 state.json files without it remain valid. Code that reads the template defaults to `"custom"` via `jq -r '.template // "custom"'`.

### Anti-Patterns to Avoid

- **Auto-promoting on completion:** This violates the oracle's Non-Invasive Guarantee and the prior decision requiring "deliberate user-triggered step." Promotion MUST be explicit.
- **Promoting ALL findings regardless of confidence:** Low-confidence findings (30-50%) are speculative. Promoting them as instincts pollutes the colony knowledge base. The 80% threshold ensures only well-supported findings become instincts.
- **Template logic in oracle.md:** The oracle.md prompt is for iteration behavior (research phase directives). Template-specific output structure belongs in `build_synthesis_prompt` because it only applies to the final synthesis pass, not to individual research iterations.
- **Breaking the custom template default:** All existing behavior must remain unchanged when no template is selected. The `custom` template path must produce identical output to the current `build_synthesis_prompt`.
- **Complex template configuration:** Templates are just a string identifier and a case branch. Do not create a template schema, template loader, template validation, or template files. Four case branches in build_synthesis_prompt is sufficient.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Colony instinct creation | Custom instinct JSON construction | `instinct-create` subcommand | Handles deduplication, confidence boosting, 30-instinct cap, atomic write |
| Learning promotion | Custom learnings.json writer | `learning-promote` subcommand | Handles 50-learning cap, deduplication, tags |
| Observation tracking | Custom observation pipeline | `memory-capture` subcommand | Orchestrates learning-observe + pheromone-write + learning-promote-auto |
| Wisdom promotion checks | Custom threshold logic | `learning-check-promotion` + `queen-promote` | Handles observation counting, threshold policy, QUEEN.md formatting |
| Source attribution in synthesis | Custom citation formatter | AI prompt directives | The AI is good at text formatting; prompt specifies [S1] format |
| Template question generation | AI decomposition per template | Hardcoded default questions | Templates should be predictable and consistent across sessions |

**Key insight:** Phase 11 is an integration phase. The colony knowledge APIs (instinct-create, learning-promote, memory-capture) were built in Phases 1-5 (v1.0) and are well-tested. The synthesis prompt infrastructure was built in Phases 6-9. Phase 11 connects these existing pieces -- it should create minimal new code and maximize reuse.

## Common Pitfalls

### Pitfall 1: promote_to_colony Writes to Colony State During Active Research
**What goes wrong:** If promote is called while the oracle loop is still running, it could write to COLONY_STATE.json while the AI is also writing to oracle files, causing state corruption.
**Why it happens:** The user might try `/ant:oracle promote` while the loop is running in tmux.
**How to avoid:** Check state.json status field. Only promote when status is "complete" or "stopped". Reject with clear error message when status is "active".
**Warning signs:** promote_to_colony called when `status == "active"` in state.json.

### Pitfall 2: Instinct Content Exceeds Useful Length
**What goes wrong:** Oracle findings can be paragraph-length. Creating an instinct with a 500-character action field makes it useless for prompt injection (pheromone-prime has length limits).
**Why it happens:** Key findings in plan.json are detailed research text, not the terse trigger/action format instincts expect.
**How to avoid:** Truncate finding text to ~200 characters for the instinct action field. The full finding remains in synthesis.md and learnings.json for reference.
**Warning signs:** Instincts with very long action fields that get truncated by downstream systems.

### Pitfall 3: Template Questions Are Too Rigid
**What goes wrong:** Template questions do not match the user's specific research topic. A tech-eval template asks "How does X compare to alternatives?" but the user only wants to evaluate one technology, not compare.
**Why it happens:** Template questions are generic by necessity.
**How to avoid:** Template questions are starting points. The wizard pre-populates plan.json with template defaults, but the AI can still adjust questions based on the specific topic. The oracle.md prompt currently says "Do NOT add new questions" -- for template sessions, this constraint should be relaxed to "You may refine template questions to fit the specific topic, but do not add entirely new questions." However, keeping the initial template decomposition is valuable for output structure consistency. Recommend keeping the constraint and making template questions broad enough to accommodate variation.
**Warning signs:** Users frequently selecting "Custom research" because templates don't fit.

### Pitfall 4: Synthesis Prompt Becomes Too Long with Template Sections
**What goes wrong:** The build_synthesis_prompt function, with template sections + confidence grouping + base oracle.md, exceeds useful prompt length and the AI loses focus.
**Why it happens:** Each template adds 10-15 lines of section requirements. Combined with oracle.md (~169 lines) and the synthesis preamble, the total prompt grows.
**How to avoid:** Keep template section definitions concise (one line per section). The current prompt (synthesis preamble + oracle.md) is ~210 lines. Adding template sections adds ~12-15 lines. Total ~225 lines is well within AI attention limits. Do not add lengthy examples or explanations in template sections.
**Warning signs:** AI synthesis output missing template-specific sections because they were buried in the prompt.

### Pitfall 5: OpenCode Command Parity Missed
**What goes wrong:** `.claude/commands/ant/oracle.md` gets template and promote changes but `.opencode/commands/ant/oracle.md` does not.
**Why it happens:** Documented pitfall from every prior phase. This project maintains command parity.
**How to avoid:** Update both wizard files. Run `npm run lint:sync` to catch drift.
**Warning signs:** `npm run lint:sync` fails after wizard update.

### Pitfall 6: Promotion Duplicates Findings Already in Colony
**What goes wrong:** Running `/ant:oracle promote` twice on the same research creates duplicate instincts.
**Why it happens:** User runs promote, forgets, runs it again.
**How to avoid:** `instinct-create` already handles deduplication: if trigger+action matches an existing instinct, it boosts confidence instead of creating a new one. `learning-promote` writes to learnings.json -- duplicate entries are cosmetic but not harmful (cap prevents unbounded growth). The real protection is the existing dedup in instinct-create.
**Warning signs:** Instinct confidence scores jumping higher than expected (from repeated promotion boosting).

## Code Examples

### Example 1: state.json with Template Field

```json
{
  "version": "1.1",
  "topic": "Should we migrate from Express to Fastify?",
  "scope": "both",
  "phase": "survey",
  "iteration": 0,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 0,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z",
  "status": "active",
  "strategy": "adaptive",
  "focus_areas": [],
  "template": "tech-eval"
}
```

### Example 2: plan.json with Template-Derived Questions

```json
{
  "version": "1.1",
  "sources": {},
  "questions": [
    {
      "id": "q1",
      "text": "What problem does Fastify solve that Express does not?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    },
    {
      "id": "q2",
      "text": "How does Fastify compare to Express on performance, ecosystem, and developer experience?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    },
    {
      "id": "q3",
      "text": "What are Fastify's known limitations?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    },
    {
      "id": "q4",
      "text": "What is Fastify's adoption and community status?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    },
    {
      "id": "q5",
      "text": "What is the migration path from Express to Fastify?",
      "status": "open",
      "confidence": 0,
      "key_findings": [],
      "iterations_touched": []
    }
  ],
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z"
}
```

### Example 3: Tech-Eval Synthesis Output (Expected)

```markdown
# Research Synthesis

## Topic
Should we migrate from Express to Fastify?

## Executive Summary

Fastify offers 2-3x throughput improvement over Express in benchmark tests [S1][S2],
primarily due to its schema-based serialization and Radix tree routing. The ecosystem
is mature with 200+ official and community plugins [S3], though Express's plugin
ecosystem remains significantly larger. Migration effort is moderate: most Express
middleware has a Fastify equivalent, but custom middleware requires rewriting [S4].

**Recommendation:** Migrate for new services; keep Express for stable existing services
unless performance is a bottleneck.

## Comparison Matrix

| Dimension | Express | Fastify | Assessment |
|-----------|---------|---------|------------|
| Throughput | ~15k req/s | ~45k req/s | Fastify 3x faster [S1][S2] |
| Ecosystem | 60k+ packages | 200+ plugins | Express larger [S3] |
| Learning curve | Low | Low-Medium | Comparable [S4] |
| TypeScript | Bolt-on | First-class | Fastify better [S3] |
| Maturity | 12+ years | 8+ years | Both mature |

## Pros and Cons

### Fastify Pros
- **High confidence (85%):** 2-3x throughput improvement [S1][S2]
- **High confidence (82%):** First-class TypeScript support [S3]

### Fastify Cons
- **Medium confidence (65%):** Smaller middleware ecosystem (single source) [S3]
- **Medium confidence (60%):** Schema-first approach requires upfront design [S4]

## Adoption Assessment
...

## Migration Path
...

## Recommendation
...

## Open Questions
...

## Sources

### Documentation
- [S1] Fastify Benchmarks -- https://fastify.dev/benchmarks/ (accessed: 2026-03-13)

### GitHub
- [S2] TechEmpower Framework Benchmarks -- https://github.com/TechEmpower/FrameworkBenchmarks (accessed: 2026-03-13)

### Official
- [S3] Fastify Ecosystem -- https://fastify.dev/ecosystem/ (accessed: 2026-03-13)

### Blog
- [S4] Express to Fastify Migration Guide -- https://example.com/migration (accessed: 2026-03-13)
```

### Example 4: promote_to_colony Output (Expected)

```
Oracle Research: Should we migrate from Express to Fastify?
Status: complete
High-confidence findings: 3 (answering 3 questions)

Promoting 3 high-confidence findings to colony knowledge...

  instinct-create: "When researching: What problem does Fastify solve..." -> created (confidence: 0.85)
  learning-promote: "Oracle: What problem does Fastify solve..." -> promoted (1/50)
  memory-capture: observation recorded

  instinct-create: "When researching: How does Fastify compare..." -> created (confidence: 0.82)
  learning-promote: "Oracle: How does Fastify compare..." -> promoted (2/50)
  memory-capture: observation recorded

  instinct-create: "When researching: What is the migration path..." -> created (confidence: 0.80)
  learning-promote: "Oracle: What is the migration path..." -> promoted (3/50)
  memory-capture: observation recorded

Promoted 3 findings to colony knowledge.
  - Instincts created in COLONY_STATE.json
  - Learnings stored in learnings.json
  - Observations tracked for wisdom promotion
```

### Example 5: Test Helper for Template Validation

```javascript
// tests/unit/oracle-colony.test.js pattern
const test = require('ava');
const { execSync } = require('child_process');
const path = require('path');
const fs = require('fs');
const os = require('os');

const ORACLE_SH = path.join(__dirname, '../../.aether/oracle/oracle.sh');

function createTmpDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), 'aether-oracle-colony-'));
}

function writeState(dir, overrides = {}) {
  const defaults = {
    version: '1.1',
    topic: 'Test topic',
    scope: 'both',
    phase: 'verify',
    iteration: 10,
    max_iterations: 15,
    target_confidence: 95,
    overall_confidence: 85,
    started_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T01:00:00Z',
    status: 'complete',
    strategy: 'adaptive',
    focus_areas: [],
    template: 'custom'
  };
  const state = Object.assign({}, defaults, overrides);
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
}

function writePlanWithFindings(dir, questions, sources) {
  const plan = {
    version: '1.1',
    sources: sources || {},
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: q.text || `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence || 0,
      key_findings: q.findings || [],
      iterations_touched: q.touched || []
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

test('validate-oracle-state accepts template field', t => {
  const dir = createTmpDir();
  writeState(dir, { template: 'tech-eval' });
  // ... validate via aether-utils.sh
  t.pass();
});

test('validate-oracle-state accepts state without template field', t => {
  const dir = createTmpDir();
  const state = { version: '1.1', topic: 'Test', scope: 'both', phase: 'survey',
    iteration: 0, max_iterations: 15, target_confidence: 95, overall_confidence: 0,
    started_at: '2026-03-13T00:00:00Z', last_updated: '2026-03-13T00:00:00Z',
    status: 'active' };
  // No template field -- should still validate
  fs.writeFileSync(path.join(dir, 'state.json'), JSON.stringify(state, null, 2));
  // ... validate
  t.pass();
});
```

## State of the Art

| Old Approach (Phase 6-10) | Phase 11 Approach | Impact |
|---------------------------|-------------------|--------|
| Oracle findings stay in .aether/oracle/ only | High-confidence findings promote to colony instincts and learnings | Research knowledge persists beyond the oracle session |
| One-size-fits-all synthesis output | Template-specific output sections (tech-eval, architecture-review, bug-investigation, best-practices) | Output structure matches the research type |
| AI decomposes topic freely into questions | Templates provide default question sets for common patterns | More consistent, predictable research coverage |
| Flat findings in synthesis (no confidence grouping) | Findings grouped by confidence level (high/medium/low) | Reader sees reliability at a glance |
| No post-completion action | `/ant:oracle promote` subcommand | Explicit bridge from oracle to colony knowledge |
| No template field in state.json | `template` field stores research type | Synthesis pass knows output structure to use |

**Deprecated/outdated after this phase:**
- Generic-only synthesis output -- replaced by template-aware synthesis (generic becomes the `custom` fallback)
- Oracle as isolated system -- promote subcommand connects oracle to colony knowledge

## Open Questions

1. **Should promote extract synthesized text from synthesis.md or structured data from plan.json?**
   - What we know: plan.json has structured data (question text, confidence, key_findings array). synthesis.md has human-readable prose.
   - What's unclear: For instinct creation, should the action field be the raw finding text from plan.json or a summarized version from synthesis.md?
   - Recommendation: Use plan.json structured data. It has the confidence scores needed for threshold filtering and the finding texts are more concise than synthesis prose. Synthesis.md is for human reading; plan.json is for machine reading.

2. **Should template default questions be modifiable by the AI during research?**
   - What we know: Current oracle.md says "Do NOT add new sub-questions -- work through the original plan." Templates provide default questions that may not perfectly fit every topic.
   - What's unclear: Should we relax the "no new questions" rule for template sessions?
   - Recommendation: Keep the constraint. Template questions are designed to be broad enough to accommodate variation. If a question is irrelevant, the AI can remove it (existing behavior). The value of templates is predictable output structure -- allowing question changes undermines this.

3. **Should there be a `--dry-run` flag for promote?**
   - What we know: The promote subcommand modifies COLONY_STATE.json (instincts) and learnings.json (learnings).
   - What's unclear: Would users want to preview what would be promoted before committing?
   - Recommendation: The wizard already shows a summary and asks for confirmation. A dry-run flag adds complexity for marginal value. Keep the confirmation question as the safety valve.

4. **What confidence threshold for promotion -- 70% or 80%?**
   - What we know: The confidence rubric defines 60-80% as "Good understanding" and 80-95% as "Thorough." Instincts should be reliable.
   - Recommendation: Use 80%. Below 80% means the research has gaps or unresolved contradictions. Colony instincts should be based on thorough understanding. Users can always re-run research to deepen confidence.

## Sources

### Primary (HIGH confidence)
- `.aether/oracle/oracle.sh` (~850 lines) -- Current orchestrator with all Phase 7-10 additions; verified build_synthesis_prompt on lines 604-638, main loop on lines 748-851, promote integration point after run_synthesis_pass
- `.aether/oracle/oracle.md` (~169 lines) -- Current AI prompt with source tracking, confidence rubric, phase awareness
- `.aether/aether-utils.sh` (~9,808 lines) -- Colony knowledge APIs: instinct-create (lines 7249-7363), instinct-read (lines 7177-7244), learning-promote (lines 1415-1458), learning-observe (lines 5147-5283), learning-promote-auto (lines 5332-5396), memory-capture (lines 5401-5500), queen-promote (lines 4854-4914), validate-oracle-state (lines 1203-1291)
- `.claude/commands/ant/oracle.md` (~477 lines) -- Current wizard with Q1-Q6 questions, subcommand routing (stop, status), state.json/plan.json creation, tmux launch
- `.opencode/commands/ant/oracle.md` -- OpenCode mirror; structural parity with Claude version (uses $normalized_args)
- `tests/integration/instinct-pipeline.test.js` -- Instinct creation, reading, pheromone-prime integration tests
- `tests/integration/learning-pipeline.test.js` -- Learning observation, threshold checking, auto-promotion tests
- `tests/integration/wisdom-promotion.test.js` -- Queen-promote end-to-end tests
- `tests/unit/oracle-steering.test.js` -- Test patterns for oracle.sh function testing (createTmpDir, writeState, writePlan helpers)
- `.planning/REQUIREMENTS.md` -- COLN-01, COLN-02, OUTP-01, OUTP-03 definitions
- `.planning/ROADMAP.md` -- Phase 11 success criteria, dependencies (Phase 9, Phase 10)
- `.planning/phases/09-source-tracking-and-trust-layer/09-RESEARCH.md` -- Phase 9 research patterns (build_synthesis_prompt structure, template for research docs)

### Secondary (MEDIUM confidence)
- Prior decision: "Colony integration (Phase 11) deferred last -- requires all other systems stable" -- confirms dependency ordering
- Prior decision: "Colony integration API (Phase 11) needs deliberate design session before implementation" -- this research serves as that design session
- Prior decision: "Oracle wizard creates 5 structured files replacing research.json and progress.md" -- confirms state.json/plan.json/gaps.md/synthesis.md/research-plan.md structure
- Prior decision: "Flag unsourced findings rather than reject them" -- informs promote threshold design (80% confidence requires multi-source findings)

### Tertiary (LOW confidence)
- Template question sets -- Derived from general research methodology patterns; not validated against user research sessions. May need tuning based on actual usage.
- 80% confidence threshold for promotion -- Reasonable based on confidence rubric but not empirically validated. Could be 70% or 85% without changing the fundamental design.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies; all APIs exist and are tested
- Architecture (promote_to_colony): HIGH -- straightforward integration of existing APIs; deduplication and caps already handled by instinct-create and learning-promote
- Architecture (templates): HIGH -- follows established pattern (strategy field in state.json, case branches in build_oracle_prompt); simple string enum stored in state
- Architecture (template-aware synthesis): MEDIUM -- AI compliance with template-specific sections is probabilistic; clear prompt structure helps but output varies
- Architecture (wizard changes): HIGH -- follows established wizard pattern (Q1-Q6 already exist; adding Q7 is mechanical)
- Pitfalls: HIGH -- identified from reading actual code, understanding API contracts, and learning from Phase 6-10 pitfall patterns

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (stable domain; bash APIs don't change; template patterns are well-established)
