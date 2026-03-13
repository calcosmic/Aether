# Phase 9: Source Tracking and Trust Layer - Research

**Researched:** 2026-03-13
**Domain:** Citation tracking, source attribution, multi-source confidence validation, inline citation formatting in AI research output, structured state file schema extensions
**Confidence:** HIGH

## Summary

Phase 9 adds source tracking to every factual claim in oracle research output. Currently, the oracle's key_findings in plan.json are plain strings with no provenance -- findings like "React 19 uses concurrent rendering" have no URL, no title, no date, no way to verify. The confidence rubric in oracle.md already references source count ("1-2 sources" = 40-60%, "multiple sources" = 60-80%) but does not structurally require tracking which sources were used. Phase 9 makes source tracking structural: findings carry their sources as data, single-source claims are flagged automatically, and the final report includes both a sources section and inline citations.

The design is constrained by the existing architecture: plan.json is the structured state, synthesis.md is markdown, and oracle.md is the AI prompt. The source tracking layer touches all three. In plan.json, key_findings evolves from an array of strings to an array of objects carrying source references. A new top-level `sources` registry in plan.json acts as a deduplicated source catalog (keyed by ID) referenced by findings. In oracle.md, the prompt instructs the AI to record sources with every finding and cite them inline. In synthesis.md, the synthesis pass produces inline citations linking to the sources section. In oracle.sh, a new trust scoring function computes per-finding trust levels from source count, and the synthesis prompt is updated to include source and citation requirements.

The key architectural insight is that source tracking is a prompt + schema problem, not a tool problem. The AI already has access to URLs via WebSearch/WebFetch. The gap is that the prompt does not require capturing them, and the schema has no place to put them. Phase 9 closes both gaps. The oracle.sh orchestrator gains a lightweight post-iteration function that counts sources per finding and flags single-source claims. The AI does the hard work of recording sources; oracle.sh does the mechanical verification. This approach aligns with recent academic research on deep research agents: DeepTRACE (2025) found that citation accuracy in LLM research systems ranges only 40-80% without structural enforcement, and systems frequently leave claims unsupported by their own listed sources. Structural verification (oracle.sh counting source_ids) is essential to prevent this.

**Primary recommendation:** Extend plan.json with a `sources` registry and evolve `key_findings` from string arrays to structured objects with source references. Update oracle.md to require source recording on every finding. Add a `compute_trust_scores` function in oracle.sh that flags single-source findings. Update the synthesis pass prompt to require inline citations and a sources section.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TRST-01 | Every claim tracks its source (URL + title + date) | The `sources` registry in plan.json stores {id, url, title, date_accessed, type} per source. Every key_finding in plan.json is extended from a plain string to a {text, source_ids, iteration} object linking to source IDs. The oracle.md prompt requires the AI to record sources for every finding. |
| TRST-02 | Single-source claims flagged as low confidence; key claims require 2+ independent sources | The `compute_trust_scores` function in oracle.sh counts unique source_ids per finding. Findings with 1 source_id are flagged as "low_trust" in plan.json. The confidence rubric in oracle.md is extended to explicitly cap single-source findings at 50% and require 2+ independent sources for high-confidence claims. This enforces the existing rubric rule ("A single source without corroboration caps at 50%") structurally. |
| TRST-03 | Sources collected in a dedicated section with inline citations in findings | The synthesis pass prompt (build_synthesis_prompt in oracle.sh) is updated to require: (1) a "## Sources" section at the end of synthesis.md listing all sources with IDs, URLs, titles, and dates; (2) inline citation markers [S1], [S2] etc. within finding text linking to the sources section. The final research-plan.md generation is updated to show trust level per question. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| oracle.sh | Bash script (654 lines) | Add compute_trust_scores, update build_synthesis_prompt | Existing; all Phase 9 orchestrator changes go here |
| oracle.md | Prompt file (127 lines) | Require source tracking in every finding | Existing; primary AI behavior change |
| plan.json | State file | Extended schema with sources registry and structured findings | Existing; schema evolution |
| jq | 1.6+ | Source counting, trust computation, schema validation | Project standard; already used throughout |
| aether-utils.sh | ~9,808 lines | Update validate-oracle-state plan validation for new schema | Existing validation infrastructure |
| ava | ^6.0.0 | Unit tests for trust scoring and schema validation | Project standard test runner |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| bash test framework | tests/bash/test-helpers.sh | Bash integration tests for trust functions | Testing compute_trust_scores and schema validation |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Sources registry in plan.json | Separate sources.json file | Extra file to manage; plan.json already has the questions array that references sources; keeping them co-located is simpler |
| Structured key_findings objects | Keep strings, add sources as separate array | Breaks the connection between a finding and its sources; structured objects maintain 1:1 linkage |
| Source IDs as string references (S1, S2) | Numeric indices into array | String IDs are more resilient to array reordering; easier for the AI to reference consistently |
| Trust scoring in oracle.sh (bash) | Trust scoring in the AI prompt | AI self-assessment is unreliable (Phase 8 lesson); oracle.sh computes trust from structural data |

## Architecture Patterns

### Recommended Project Structure

No new files created -- all changes modify existing files:

```
.aether/oracle/
  oracle.sh           # MODIFY: add compute_trust_scores, update build_synthesis_prompt
  oracle.md           # MODIFY: add source tracking requirements, update confidence rubric

.aether/aether-utils.sh  # MODIFY: update validate-oracle-state plan validation for new schema

.claude/commands/ant/oracle.md     # MODIFY: update wizard initial plan.json creation (version 1.1, empty sources)
.opencode/commands/ant/oracle.md   # MODIFY: mirror wizard changes (parity)

tests/
  unit/oracle-trust.test.js        # NEW: ava tests for trust scoring
  bash/test-oracle-trust.sh        # NEW: bash integration tests for trust
```

### Pattern 1: Sources Registry in plan.json

**What:** A top-level `sources` object in plan.json that acts as a deduplicated catalog of all sources encountered during research.

**When to use:** Created empty at session start; populated by the AI each iteration; read by oracle.sh for trust computation.

**Schema extension to plan.json:**
```json
{
  "version": "1.1",
  "sources": {
    "S1": {
      "url": "https://docs.example.com/guide",
      "title": "Official Documentation Guide",
      "date_accessed": "2026-03-13",
      "type": "documentation"
    },
    "S2": {
      "url": "https://github.com/repo/issue/42",
      "title": "Performance regression in v3.2",
      "date_accessed": "2026-03-13",
      "type": "github"
    }
  },
  "questions": [
    {
      "id": "q1",
      "text": "How does X work?",
      "status": "partial",
      "confidence": 55,
      "key_findings": [
        {
          "text": "X uses a B-tree index for fast lookups",
          "source_ids": ["S1", "S2"],
          "iteration": 2
        },
        {
          "text": "X has a 100ms latency SLA",
          "source_ids": ["S1"],
          "iteration": 3
        }
      ],
      "iterations_touched": [1, 2, 3]
    }
  ],
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z"
}
```

**Design rationale:**

- **Object keyed by ID, not array:** Using `"S1": {...}` instead of `[{id: "S1", ...}]` makes it easy for both the AI and jq to look up sources by ID. The AI can write `"source_ids": ["S1"]` without needing to know array indices.
- **Deduplicated:** The same URL appearing in multiple iterations maps to the same source ID. The AI is instructed to reuse existing source IDs when citing the same URL.
- **Type field:** Categorizes sources (documentation, blog, github, academic, codebase, forum, official). This enables downstream filtering and credibility weighting. For codebase-only research, sources are file paths rather than URLs.
- **date_accessed vs publication date:** We track date_accessed (when the oracle found it) because many web sources lack clear publication dates. date_accessed is always knowable.

**Version bump:** plan.json version goes from "1.0" to "1.1" to signal the schema change. Backward compatibility: older plan.json files without `sources` or with string key_findings still work -- oracle.sh and validate-oracle-state handle both formats. Verified: no code in oracle.sh or aether-utils.sh checks for `version == "1.0"` specifically; the validate-oracle-state function only validates that version is a string type (line 1216 of aether-utils.sh: `chk("version";["string"])`).

### Pattern 2: Structured key_findings

**What:** Each finding in key_findings evolves from a plain string to a structured object with source linkage and iteration tracking.

**When to use:** Every time the AI records a finding.

**Old format (Phase 6-8):**
```json
"key_findings": ["X uses B-tree indexing", "X has 100ms latency SLA"]
```

**New format (Phase 9):**
```json
"key_findings": [
  {
    "text": "X uses B-tree indexing",
    "source_ids": ["S1", "S2"],
    "iteration": 2
  },
  {
    "text": "X has 100ms latency SLA",
    "source_ids": ["S1"],
    "iteration": 3
  }
]
```

**Fields:**
- `text`: The finding text (same as the old string value)
- `source_ids`: Array of source registry IDs supporting this finding. Must have at least one entry.
- `iteration`: Which iteration discovered this finding (for provenance tracking)

**Backward compatibility:** The convergence metrics in oracle.sh currently count findings on line 233 with `jq '[.questions[].key_findings | length] | add // 0'`. This still works because `length` counts array elements regardless of whether they are strings or objects. The `compute_trust_scores` function will need to handle both formats during the transition (for any plan.json files created before Phase 9 is deployed). Verified against actual oracle.sh code -- line 233 is the only reference to key_findings.

### Pattern 3: compute_trust_scores Function in oracle.sh

**What:** A function that computes trust levels for all findings based on source count and flags single-source claims.

**When to use:** After each iteration, alongside convergence metrics computation (called right after `update_convergence_metrics` in the main loop).

**Design:**

```bash
# Compute trust scores from plan.json source tracking data
# Writes trust metadata to plan.json (trust_summary field)
compute_trust_scores() {
  local plan_file="$1"

  # Check if plan.json uses the new structured findings format
  local has_structured
  has_structured=$(jq '
    [.questions[].key_findings[] | type] | if length == 0 then false else any(. == "object") end
  ' "$plan_file" 2>/dev/null || echo "false")

  if [ "$has_structured" != "true" ]; then
    # Pre-Phase-9 plan.json with string findings -- skip trust computation
    return 0
  fi

  local total_findings single_source multi_source no_source
  total_findings=$(jq '[.questions[].key_findings[]] | length' "$plan_file" 2>/dev/null || echo "0")
  single_source=$(jq '[.questions[].key_findings[] | select(type == "object" and (.source_ids | length) == 1)] | length' "$plan_file" 2>/dev/null || echo "0")
  multi_source=$(jq '[.questions[].key_findings[] | select(type == "object" and (.source_ids | length) >= 2)] | length' "$plan_file" 2>/dev/null || echo "0")
  no_source=$(jq '[.questions[].key_findings[] | select(type == "object" and ((.source_ids // []) | length) == 0)] | length' "$plan_file" 2>/dev/null || echo "0")

  local trust_ratio=0
  if [ "$total_findings" -gt 0 ]; then
    trust_ratio=$(( multi_source * 100 / total_findings ))
  fi

  jq --argjson total "$total_findings" \
     --argjson single "$single_source" \
     --argjson multi "$multi_source" \
     --argjson nosrc "$no_source" \
     --argjson ratio "$trust_ratio" \
     '.trust_summary = {
       total_findings: $total,
       single_source: $single,
       multi_source: $multi,
       no_source: $nosrc,
       trust_ratio: $ratio
     }' "$plan_file" > "$plan_file.tmp" && mv "$plan_file.tmp" "$plan_file"
}
```

**Trust ratio:** The percentage of findings backed by 2+ sources. This is a structural metric (like convergence) that oracle.sh computes from data, not from AI self-assessment. It can be displayed in research-plan.md and used as a quality signal.

**Why in oracle.sh, not the AI prompt:** Same principle as convergence detection (Phase 8). The AI records sources; oracle.sh counts them. DeepTRACE research (2025) found that LLM citation accuracy ranges only 40-80% without structural enforcement, and systems frequently produce claims unsupported by their listed sources. Having oracle.sh verify source counts provides ground truth independent of the AI's self-assessment.

**Integration point in main loop:** After line 605 (`update_convergence_metrics "$STATE_FILE" "$PLAN_FILE"`), add `compute_trust_scores "$PLAN_FILE"`.

### Pattern 4: Oracle.md Source Tracking Requirements

**What:** Updates to the oracle.md prompt that require the AI to track sources with every finding.

**When to use:** Applies to every iteration across all phases.

**Key additions to oracle.md:**

1. **Step 3 (Research) additions:**
```markdown
**Source Tracking (MANDATORY):**
For every new finding, you MUST record:
- The source URL (or file path for codebase research)
- The source title/description
- The date you accessed it

Register sources in plan.json under the `sources` object using sequential IDs
(S1, S2, S3...). Reuse existing source IDs if citing the same URL again.

Source types: "documentation", "blog", "github", "academic", "codebase", "forum", "official"

For codebase research, use file paths as URLs:
  "url": "src/components/Button.tsx",
  "title": "Button component source",
  "type": "codebase"
```

2. **Step 4 (Update State Files) key_findings format change:**
```markdown
**plan.json:** Update the target question:
- Add findings as OBJECTS (not strings):
  {"text": "finding text", "source_ids": ["S1", "S2"], "iteration": <current>}
- Every finding MUST have at least one source_id
- Add new sources to the top-level `sources` registry
- Reuse existing source IDs for the same URL
```

3. **Confidence Rubric update** (integrate with existing rubric):
```markdown
**Source-backed confidence rules:**
- 0 sources: Finding is UNSUPPORTED -- do not record it
- 1 source: Single-source claim, capped at 50% contribution to question confidence
- 2+ sources: Multi-source claim, full confidence contribution
- The overall question confidence should reflect the source backing of its findings
```

4. **Synthesis pass additions** (oracle.md final rule):
```markdown
- In synthesis.md, use inline citations [S1], [S2] etc. linking to the sources
  registry in plan.json. Include a "## Sources" section at the end listing all
  sources with their IDs, URLs, titles, and access dates.
```

### Pattern 5: Updated Synthesis Pass Prompt

**What:** The build_synthesis_prompt function in oracle.sh is updated to require citation formatting in the final report.

**When to use:** On every synthesis pass (converged, stopped, max_iterations, interrupted).

**Current build_synthesis_prompt location:** oracle.sh lines 422-454. The current "Required Sections" list has 4 items (Executive Summary, Findings by Question, Open Questions, Methodology Notes). Phase 9 adds a 5th: Sources.

**Addition to build_synthesis_prompt:**
```bash
build_synthesis_prompt() {
  local reason="$1"

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

Then REWRITE synthesis.md as a structured final report:

### Required Sections:
1. **Executive Summary** -- 2-3 paragraphs summarizing what was found
2. **Findings by Question** -- organized by sub-question, with confidence %
   - Use inline citations [S1], [S2] linking findings to their sources
   - Flag single-source findings with "(single source)" marker
3. **Open Questions** -- remaining gaps with explanation of what is unknown and why
4. **Methodology Notes** -- how many iterations, which phases completed
5. **Sources** -- List ALL sources from plan.json sources registry:
   - Format: [S1] Title -- URL (accessed: date)
   - Group by type (documentation, blog, codebase, etc.)
   - Note total source count and multi-source coverage percentage

Also update state.json: set status to "complete" if reason is "converged",
or "stopped" otherwise.

SYNTHESIS_DIRECTIVE

  # Append the base oracle.md for tool access and rules
  cat "$SCRIPT_DIR/oracle.md"
}
```

### Pattern 6: Updated research-plan.md Generation

**What:** The generate_research_plan function is updated to show trust data.

**When to use:** After each iteration (same as before, but with new section).

**Current generate_research_plan location:** oracle.sh lines 29-65. Currently outputs: header, questions table, next steps.

**Addition to generate_research_plan:**
```bash
# After the existing questions table, add trust summary if available
local trust_ratio
trust_ratio=$(jq '.trust_summary.trust_ratio // -1' "$plan_file" 2>/dev/null || echo "-1")
if [ "$trust_ratio" -ge 0 ]; then
  local total single multi
  total=$(jq '.trust_summary.total_findings // 0' "$plan_file")
  single=$(jq '.trust_summary.single_source // 0' "$plan_file")
  multi=$(jq '.trust_summary.multi_source // 0' "$plan_file")
  echo ""
  echo "## Source Trust"
  echo "| Total Findings | Multi-Source | Single-Source | Trust Ratio |"
  echo "|----------------|-------------|---------------|-------------|"
  echo "| $total | $multi | $single | ${trust_ratio}% |"
fi
```

### Pattern 7: Backward-Compatible Validation

**What:** Update validate-oracle-state in aether-utils.sh to validate the new plan.json schema while accepting old format.

**When to use:** Post-iteration validation, tests.

**Current validate-oracle-state plan validation location:** aether-utils.sh lines 1233-1259. Current checks: version (string), questions (array), created_at (string), last_updated (string), questions count 1-8, question fields (id, text, status, confidence, key_findings, iterations_touched), status enum, confidence range.

**Design:**

The existing plan validation checks for `has("id","text","status","confidence","key_findings","iterations_touched")` on each question. The new validation:

1. Accepts both string and object key_findings (backward compatible)
2. If key_findings contains objects, validates they have `text` and `source_ids` fields
3. If `sources` registry exists, validates each entry has `url`, `title`, `date_accessed`
4. Does NOT require sources or structured findings (pre-Phase-9 files remain valid)

```bash
# Additional validation checks for Phase 9 fields (optional, backward compatible)
if (.sources // null) != null then
  if (.sources | type) == "object" then
    if (.sources | to_entries | all(.value | has("url","title","date_accessed"))) then "pass"
    else "fail: sources entries missing required fields (url, title, date_accessed)"
    end
  else "fail: sources must be an object"
  end
else "pass"  # sources field is optional
end,
if ([.questions[].key_findings[] | type] | any(. == "object")) then
  if ([.questions[].key_findings[] | select(type == "object") | has("text","source_ids")] | all) then "pass"
  else "fail: structured findings missing required fields (text, source_ids)"
  end
else "pass"  # string findings are still valid (pre-Phase-9)
end
```

### Pattern 8: Phase Directive Source Reminders

**What:** Each phase directive in build_oracle_prompt gets a one-line source tracking reminder.

**When to use:** All phases (survey, investigate, synthesize, verify).

**Current build_oracle_prompt location:** oracle.sh lines 117-200. Each case branch outputs a phase directive then falls through to `cat "$oracle_md"`.

**Example for investigate phase (add to existing directive):**
```bash
investigate)
  cat <<'DIRECTIVE'
## Current Phase: INVESTIGATE

Target the lowest-confidence question and go DEEP. You MUST reference existing
findings in synthesis.md and ADD NEW information, not restate what is already there.
Aim to push confidence above 70% for your target question.

**Source tracking is MANDATORY this iteration.** Every new finding must have:
- At least one source_id linking to the sources registry in plan.json
- Register new sources; reuse existing source IDs for the same URL

Update gaps.md with specific remaining unknowns. If you find contradictions with
existing findings, document them explicitly. One thoroughly investigated question
per iteration is better than shallow passes on many.

---

DIRECTIVE
  ;;
```

### Anti-Patterns to Avoid

- **Trusting the AI to count its own sources:** The AI may claim "3 sources support this" while only providing 1 URL. oracle.sh counts the actual source_ids array length -- same principle as convergence detection. DeepTRACE found that "large fractions of statements remain unsupported by their own listed sources" -- structural verification is essential.
- **Requiring sources on codebase-only research iteration 1:** During survey phase on codebase-only scope, findings may come from reading code directly. These should use file paths as "urls" with type "codebase", not be blocked because they lack a web URL.
- **Separate sources file:** Adding sources.json alongside plan.json creates synchronization headaches. Sources belong in plan.json because they are directly referenced by findings in the same file. One file, one source of truth.
- **Complex source credibility scoring:** ADVN-03 in REQUIREMENTS.md explicitly defers "Source credibility scoring (domain authority, recency)" to future work. Phase 9 only tracks source presence and count, not source quality.
- **Breaking convergence metrics:** The novelty delta computation on oracle.sh line 233 (`jq '[.questions[].key_findings | length] | add // 0'`) counts array elements. This works whether elements are strings or objects. Do NOT change this computation -- it naturally supports the new format.
- **Inline citations in plan.json:** plan.json is machine-readable state. Inline citations belong in synthesis.md (human-readable output). plan.json stores structured references (source_ids arrays).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Source deduplication | Custom dedup logic in oracle.sh | AI deduplication via prompt instruction + jq check | The AI sees all existing sources when it reads plan.json; instructing it to reuse IDs is simpler than post-hoc dedup |
| Trust computation | AI self-assessment of source quality | compute_trust_scores in oracle.sh (counts source_ids) | Same principle as convergence -- structural metrics, not AI opinion |
| Source validation | URL checker/validator | Basic jq type checks in validate-oracle-state | We cannot validate URLs are real (no HTTP client in bash); checking that they exist and have the right fields is sufficient |
| Citation formatting | Custom markdown renderer | AI-generated inline citations via synthesis prompt | The AI is good at text formatting; the prompt specifies the [S1] format |
| Version migration | Schema migration script | Backward-compatible code + version bump | oracle.sh handles both string and object key_findings; no migration needed |

**Key insight:** Phase 9 is primarily a PROMPT change backed by a SCHEMA change. The orchestrator (oracle.sh) does minimal work: counting sources and flagging single-source findings. The AI does the heavy lifting of recording sources. The schema ensures the data is structurally sound.

## Common Pitfalls

### Pitfall 1: AI Fails to Record Sources Consistently
**What goes wrong:** The AI records sources for some findings but not others, especially during the investigate phase when it is deep in codebase analysis.
**Why it happens:** The AI prioritizes depth of research over citation discipline. Source recording is a secondary behavior that gets dropped under cognitive load. DeepTRACE's research confirms this is endemic: citation accuracy across LLM systems ranges only 40-80%.
**How to avoid:** Make the source tracking instruction prominent in every phase directive, not just in oracle.md's general rules. Each phase directive (survey/investigate/synthesize/verify) should have a one-line reminder: "Source tracking is MANDATORY this iteration." Additionally, compute_trust_scores flags unsourced findings (no_source count), making the gap visible in research-plan.md.
**Warning signs:** trust_summary.no_source > 0 in plan.json; findings with empty source_ids arrays.

### Pitfall 2: Source IDs Collide or Skip
**What goes wrong:** The AI creates S1, S2, S5 (skipping S3, S4) or creates two different S1 entries for different URLs.
**Why it happens:** The AI does not count existing source IDs carefully across iterations. Different iterations may lose track of the highest ID.
**How to avoid:** The prompt instructs: "Check existing sources in plan.json before adding new ones. Use the next available sequential ID." Additionally, validate-oracle-state can check for duplicate IDs or gaps, but gaps are cosmetic (not harmful) and not worth blocking on.
**Warning signs:** jq '.sources | keys' shows non-sequential IDs. Not a critical issue -- trust computation works on source_ids arrays regardless of ID naming.

### Pitfall 3: Breaking Existing Tests with Schema Change
**What goes wrong:** Existing oracle tests in test-oracle-phase.sh, test-oracle-convergence.sh, oracle-convergence.test.js, oracle-phase-transitions.test.js fail because they create plan.json with string key_findings.
**Why it happens:** The new schema changes key_findings from strings to objects, and existing tests hardcode string format.
**How to avoid:** Make ALL changes backward-compatible. The compute_trust_scores function explicitly checks for structured findings (`type == "object"`) and skips trust computation for string findings. Existing tests continue to work because they use string findings, and trust scoring is simply not invoked. New tests validate the structured format. validate-oracle-state accepts both formats. Verified: existing tests use `writePlan()` helper (oracle-convergence.test.js line 26) with string findings arrays -- these remain valid.
**Warning signs:** `npx ava tests/unit/oracle-convergence.test.js` or `bash tests/bash/test-oracle-convergence.sh` fails after Phase 9 changes.

### Pitfall 4: Synthesis Pass Ignores Sources
**What goes wrong:** The final synthesis.md has no inline citations or sources section despite plan.json having sources data.
**Why it happens:** The synthesis pass prompt does not include clear enough instructions about citation formatting, or the AI treats it as optional.
**How to avoid:** The build_synthesis_prompt function explicitly lists "5. Sources" as a Required Section and provides the exact format: "[S1] Title -- URL (accessed: date)." The synthesis directive is assertive: "Use inline citations [S1], [S2] linking findings to their sources" -- not "consider adding citations."
**Warning signs:** Final synthesis.md lacks "## Sources" section; grep for "[S" in synthesis.md returns no matches.

### Pitfall 5: Large Sources Registry Bloats plan.json
**What goes wrong:** A 50-iteration research session accumulates 200+ sources in plan.json, making the file unwieldy for the AI to read.
**Why it happens:** Web research generates many URLs; the AI is instructed to register every source.
**How to avoid:** This is unlikely to be a practical problem for most sessions. Even 200 source entries at ~100 bytes each is only 20KB -- well within JSON handling limits. If it becomes an issue, a future optimization could trim sources not referenced by any finding. For Phase 9, do not add source limits or pruning.
**Warning signs:** plan.json exceeding 100KB. Monitor during real usage.

### Pitfall 6: OpenCode Command Parity Missed
**What goes wrong:** .claude/commands/ant/oracle.md wizard is updated with new plan.json initial structure (empty sources, version 1.1), but .opencode/commands/ant/oracle.md is not.
**Why it happens:** Same pitfall documented in Phase 6 research. This project maintains command parity across Claude Code and OpenCode. Verified: .opencode/commands/ant/oracle.md exists and is a structural mirror of .claude/commands/ant/oracle.md (different argument handling on line 8 -- uses `$normalized_args` vs `$ARGUMENTS`).
**How to avoid:** Update both wizard files. Run `npm run lint:sync` to catch structural drift.
**Warning signs:** `npm run lint:sync` fails after wizard update.

## Code Examples

### Example 1: Initial plan.json Creation with Sources Registry (Wizard)

```json
{
  "version": "1.1",
  "sources": {},
  "questions": [
    {
      "id": "q1",
      "text": "What is X?",
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

### Example 2: plan.json After Two Iterations with Sources

```json
{
  "version": "1.1",
  "sources": {
    "S1": {
      "url": "https://docs.react.dev/reference/react",
      "title": "React API Reference",
      "date_accessed": "2026-03-13",
      "type": "documentation"
    },
    "S2": {
      "url": "https://github.com/facebook/react/releases/tag/v19.0.0",
      "title": "React 19 Release Notes",
      "date_accessed": "2026-03-13",
      "type": "github"
    },
    "S3": {
      "url": "src/components/App.tsx",
      "title": "App component source",
      "date_accessed": "2026-03-13",
      "type": "codebase"
    }
  },
  "questions": [
    {
      "id": "q1",
      "text": "What rendering model does React 19 use?",
      "status": "partial",
      "confidence": 55,
      "key_findings": [
        {
          "text": "React 19 uses concurrent rendering by default with automatic batching",
          "source_ids": ["S1", "S2"],
          "iteration": 1
        },
        {
          "text": "The app currently uses React 18 createRoot which already enables concurrent features",
          "source_ids": ["S3"],
          "iteration": 2
        }
      ],
      "iterations_touched": [1, 2]
    }
  ],
  "trust_summary": {
    "total_findings": 2,
    "single_source": 1,
    "multi_source": 1,
    "no_source": 0,
    "trust_ratio": 50
  },
  "created_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T01:00:00Z"
}
```

### Example 3: Synthesis.md with Inline Citations (Expected Output)

```markdown
# Research Synthesis

## Topic
React 19 migration assessment

## Executive Summary

React 19 introduces concurrent rendering as the default mode, eliminating the need for
explicit opt-in [S1]. The release includes automatic batching for all state updates and
a new compiler that replaces manual memoization [S1][S2]. Our codebase currently uses
React 18's createRoot API, which already enables some concurrent features [S3].

Migration risk is moderate. The main breaking change is the removal of legacy context API
(single source) [S2], which affects 3 components in our codebase [S3].

## Findings by Question

### q1: What rendering model does React 19 use?
**Status:** partial | **Confidence:** 55%

- React 19 uses concurrent rendering by default with automatic batching [S1][S2]
- The app currently uses React 18 createRoot which already enables concurrent features [S3] (single source)

### q2: What are the breaking changes?
**Status:** partial | **Confidence:** 40%

- Legacy context API removed in React 19 [S2] (single source)

## Open Questions

- What is the performance impact of concurrent rendering on our specific use case? (no sources found)
- Does the new compiler work with our custom Babel plugins? (single source, needs verification)

## Methodology Notes

Completed 5 iterations across survey and investigate phases.

## Sources

### Documentation
- [S1] React API Reference -- https://docs.react.dev/reference/react (accessed: 2026-03-13)

### GitHub
- [S2] React 19 Release Notes -- https://github.com/facebook/react/releases/tag/v19.0.0 (accessed: 2026-03-13)

### Codebase
- [S3] App component source -- src/components/App.tsx (accessed: 2026-03-13)

**Source Coverage:** 3 sources | 1/2 findings multi-sourced (50% trust ratio)
```

### Example 4: Backward-Compatible Findings Count for Convergence

```bash
# This existing jq query on oracle.sh line 233 still works with both formats
# Strings: ["finding1", "finding2"] -- length = 2
# Objects: [{"text":"...", "source_ids":["S1"]}, ...] -- length = 2
current_findings=$(jq '[.questions[].key_findings | length] | add // 0' "$plan_file" 2>/dev/null || echo "0")
```

### Example 5: Test Helper for Structured Findings (New Tests)

```javascript
// tests/unit/oracle-trust.test.js pattern
function writePlanWithSources(dir, questions, sources) {
  const plan = {
    version: '1.1',
    sources: sources || {},
    questions: questions.map((q, i) => ({
      id: `q${i + 1}`,
      text: `Question ${i + 1}?`,
      status: q.status || 'open',
      confidence: q.confidence,
      key_findings: q.findings || [],
      iterations_touched: q.touched || []
    })),
    created_at: '2026-03-13T00:00:00Z',
    last_updated: '2026-03-13T00:00:00Z'
  };
  fs.writeFileSync(path.join(dir, 'plan.json'), JSON.stringify(plan, null, 2));
}

// Example usage: plan with structured findings and sources
writePlanWithSources(dir, [
  {
    confidence: 55,
    touched: [1, 2],
    status: 'partial',
    findings: [
      { text: 'Finding with two sources', source_ids: ['S1', 'S2'], iteration: 1 },
      { text: 'Finding with one source', source_ids: ['S1'], iteration: 2 }
    ]
  }
], {
  S1: { url: 'https://docs.example.com', title: 'Example Docs', date_accessed: '2026-03-13', type: 'documentation' },
  S2: { url: 'https://github.com/example', title: 'Example Repo', date_accessed: '2026-03-13', type: 'github' }
});
```

## State of the Art

| Old Approach (Phase 6-8) | Phase 9 Approach | Impact |
|--------------------------|------------------|--------|
| key_findings as plain strings | key_findings as structured objects with source_ids | Every finding is traceable to its sources |
| No sources tracking | Sources registry in plan.json (deduplicated, typed) | All sources cataloged in one place |
| Confidence rubric mentions sources but doesn't enforce | Source count structurally enforced: 1 source = capped at 50% | Single-source claims cannot inflate confidence |
| No trust metrics | compute_trust_scores in oracle.sh | Trust ratio visible in research-plan.md |
| Synthesis has no citations | Inline [S1] citations + Sources section | Final report is verifiable |
| build_synthesis_prompt has no citation requirements | Synthesis directive requires inline citations + sources section | Every synthesis pass produces cited output |
| validate-oracle-state checks basic plan.json fields | Extended validation for sources and structured findings | Schema enforcement for trust data |
| Phase directives have no source reminders | Each phase directive includes source tracking reminder | Reinforces citation discipline every iteration |

**Deprecated/outdated after this phase:**
- plan.json version "1.0" -- replaced by "1.1" (old format still accepted)
- String key_findings -- replaced by structured objects (old format still accepted during transition)
- Synthesis.md without citations -- synthesis pass now requires inline citations

## Open Questions

1. **How aggressive should unsourced finding enforcement be?**
   - What we know: The prompt will instruct "every finding must have at least one source." oracle.sh can count no_source findings.
   - What's unclear: Should oracle.sh actively reject unsourced findings (strip them from plan.json)? Or just flag them?
   - Recommendation: Flag only. oracle.sh reports no_source count in trust_summary but does not modify findings. The AI might occasionally produce a valid insight without a URL (especially from reasoning or codebase analysis). Stripping findings is destructive and could lose valuable research. The trust_summary makes the gap visible.

2. **Should the version bump from 1.0 to 1.1 break anything?**
   - What we know: validate-oracle-state checks `chk("version";["string"])` -- it validates the type but not the value. Grep confirms no code checks for `version == "1.0"` specifically in oracle.sh or aether-utils.sh.
   - Recommendation: The bump is safe. No migration needed.

3. **How should codebase-only research handle source tracking?**
   - What we know: For scope="codebase", the oracle reads local files, not web URLs.
   - Recommendation: File paths serve as source URLs for codebase research. The type field is "codebase" and the URL is the file path (e.g., `"url": "src/components/Button.tsx"`). The key requirement is traceability, not web URLs.

## Sources

### Primary (HIGH confidence)
- `.aether/oracle/oracle.sh` (654 lines) -- Current orchestrator with all Phase 7+8 additions; verified key_findings reference on line 233, build_synthesis_prompt on lines 422-454, generate_research_plan on lines 29-65, build_oracle_prompt on lines 117-200, main loop on lines 564-645
- `.aether/oracle/oracle.md` (127 lines) -- Current phase-aware prompt with confidence rubric; verified "single source without corroboration caps at 50%" rule on line 112
- `.aether/aether-utils.sh` lines 1203-1274 -- validate-oracle-state implementation; verified plan validation checks on lines 1233-1259 and version type-only check on line 1216
- `.claude/commands/ant/oracle.md` (432 lines) -- Current wizard/command handler; verified plan.json initial creation in Step 2
- `.opencode/commands/ant/oracle.md` -- OpenCode wizard mirror; verified structural parity with Claude version
- `tests/unit/oracle-convergence.test.js` (522 lines) -- Existing convergence test patterns; verified writePlan helper uses string findings on line 33
- `tests/bash/test-oracle-convergence.sh` (353 lines) -- Existing bash test patterns; verified write_plan helper uses string findings
- `.planning/REQUIREMENTS.md` -- TRST-01, TRST-02, TRST-03 definitions
- `.planning/ROADMAP.md` -- Phase 9 success criteria and dependencies
- `.planning/phases/06-state-architecture-foundation/06-RESEARCH.md` -- plan.json schema origin (version 1.0)
- `.planning/phases/08-orchestrator-upgrade/08-RESEARCH.md` -- Convergence metrics pattern (trust follows same approach)

### Secondary (MEDIUM confidence)
- [DeepTRACE: Auditing Deep Research AI Systems for Tracking Reliability Across Citations and Evidence](https://arxiv.org/html/2509.04499v1) -- Citation accuracy ranges 40-80% without structural enforcement; systems frequently leave claims unsupported by listed sources; recommends citation matrix verification
- [A Comprehensive Survey of Deep Research: Systems, Methodologies, and Applications](https://arxiv.org/html/2506.12594v1) -- Source attribution and verification identified as critical technical challenge; recommends source diversity assessment
- [Deep Research: A Survey of Autonomous Research Agents](https://arxiv.org/html/2508.12752v1) -- Phase I agentic search systems produce answers supported by explicit citations; pipeline architecture recommended for citation metadata flow

### Tertiary (LOW confidence)
- Specific source ID format (S1, S2 vs numeric) -- Reasonable convention but not validated against any established standard; chosen for readability and AI reference ease
- Trust ratio as a meaningful metric -- The percentage threshold is not empirically validated; it's a starting point that makes the gap visible without prescribing a quality gate

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies; all tools already exist in the project
- Architecture (plan.json schema): HIGH -- extends existing schema with backward-compatible fields; follows established patterns; verified against actual code
- Architecture (compute_trust_scores): HIGH -- simple jq queries counting source_ids; follows same pattern as compute_convergence; integration point verified at line 605 of oracle.sh
- Architecture (oracle.md prompt changes): MEDIUM -- prompt effectiveness depends on AI compliance; instructions are clear but AI behavior is inherently variable; DeepTRACE confirms citation accuracy is 40-80% even with good prompting
- Architecture (synthesis citations): MEDIUM -- citation formatting quality depends on AI output; prompt is prescriptive but results may vary
- Pitfalls: HIGH -- identified from reading actual code and understanding AI behavioral patterns from Phase 7/8 experience; confirmed by DeepTRACE's findings on LLM citation reliability

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (stable domain; bash/jq patterns don't change; AI citation patterns are well-established)
