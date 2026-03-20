# Phase 22: Agent Boilerplate Cleanup - Research

**Researched:** 2026-02-19
**Domain:** Agent definition file content analysis and cleanup
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- Safety-first approach — user explicitly concerned about breaking things
- Batch processing: clean 5-8 agents at a time, verify each batch before continuing
- Target: "focused but complete" — each agent reads like a full job description, just without the parts shared across all agents
- When in doubt, keep it — only strip what's clearly redundant
- Generic AI prompting tips ("be thorough", "think step by step"): keep the ones that genuinely help performance, strip pure filler
- Tool availability lists: Claude decides per agent whether these add value
- Project description sections: Claude decides per agent whether project context is needed
- Colony rules (pheromones, castes, milestones): Claude decides per agent whether these are needed
- Structure/template: Claude decides based on best practices (shared core + flexible extras expected)
- Similar agents (multiple scouts/builders): Claude decides whether to share base definitions or keep independent
- Agent naming: YES, fix names that don't match what the agent actually does
- Agent merging: OUT OF SCOPE — strictly boilerplate cleanup, no boundary changes
- Near-duplicate sections (90% same, 10% different): Claude decides per case based on importance of differences
- Outdated references: DO NOT fix — only strip boilerplate. Outdated content is a separate task
- Verification: test each batch after cleanup to confirm agents still spawn and respond correctly

### Claude's Discretion

- Whether to leave pointers ("see workers.md") or clean-remove stripped sections — decide based on what works best per section
- Colony rules per agent — some agents need them, some don't
- Tool lists per agent — remove if purely noise, keep if they guide agent behavior
- Project descriptions per agent — remove or condense based on whether agent needs context
- Shared vs independent structure for similar agents
- Which generic prompting tips actually improve agent performance

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope. Agent merging was explicitly ruled out of scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| AGENT-01 | Strip redundant sections from all 25 agents | Research identifies 4 universally redundant sections totaling ~19 lines per agent; confirms what is safe to remove |
| AGENT-02 | Ensure each agent reads like a focused job description | Research confirms that unique role content (workflow, domain knowledge, output format) is already well-differentiated; boilerplate is distinct from unique content |
| AGENT-03 | Fix agent names that don't match actual behavior | Research identifies 12 agents with old-format descriptions that should be updated to match the "Use this agent for..." pattern used by 12 other agents |
| AGENT-04 | Batch verification between cleanup groups | Research defines natural batching groups by agent cluster (core 5, development 4, knowledge 4, quality 4, special 4, surveyors 4) |
</phase_requirements>

---

## Summary

All 25 agent files in `/Users/callumcowie/repos/Aether/.opencode/agents/` were read in full. The agents divide into two structural families: 20 flat-markdown agents (80-138 lines each) and 4 XML-structured surveyor agents (209-334 lines each), plus `workers.md` (a 1,034-line reference document, not itself an agent).

The flat-markdown agents share a consistent structural skeleton with four sections that are either completely identical or trivially varied across all agents: "Aether Integration" (identical in all 20), "Activity Logging" (identical except the caste name in the log command), "Depth-Based Behavior" (identical table with only the caste name varying in row 1), and the "Reference" footer (identical in all 20). These four sections account for approximately 19 lines per agent and add no unique information — everything they contain is already established by the spawn context the Queen/Prime Worker provides at invocation time.

The surveyor agents (nest, disciplines, pathogens, provisions) use a different XML-tagged structure and have zero overlap with these boilerplate patterns. They do not need the same cleanup.

**Primary recommendation:** Remove the four universally redundant sections from all 20 flat-markdown agents in batches of 5-8, then update old-format descriptions to the "Use this agent for..." pattern for 12 agents that still use the shorter legacy description style.

---

## Standard Stack

### Core

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| Markdown file editing | — | In-place file editing | Files are markdown, direct edit is the right tool |
| OpenCode agent frontmatter | YAML | Agent routing and invocation | Platform-defined format |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `npm run lint:sync` | current | Verify Claude/OpenCode command parity | After any agent file changes |
| `npm test` | current | Run full test suite | After each batch |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Direct file editing per agent | Sed/awk batch script | Batch script is faster but harder to inspect; direct editing is safer for a first-pass cleanup |

---

## Architecture Patterns

### Agent File Groups

The 25 agents divide into natural cleanup batches aligned with their caste clusters:

```
Batch 1 - Core Agents (5 agents):
  aether-queen.md          (138 lines)
  aether-builder.md        (134 lines)
  aether-watcher.md        (127 lines)
  aether-scout.md          (93 lines)
  aether-route-setter.md   (85 lines)

Batch 2 - Development Cluster (4 agents):
  aether-weaver.md         (87 lines)
  aether-probe.md          (91 lines)
  aether-ambassador.md     (97 lines)
  aether-tracker.md        (91 lines)

Batch 3 - Knowledge Cluster (4 agents):
  aether-chronicler.md     (80 lines)
  aether-keeper.md         (106 lines)
  aether-auditor.md        (111 lines)
  aether-sage.md           (98 lines)

Batch 4 - Quality Cluster (4 agents):
  aether-guardian.md       (107 lines)
  aether-measurer.md       (119 lines)
  aether-includer.md       (108 lines)
  aether-gatekeeper.md     (107 lines)

Batch 5 - Special Agents (3 agents):
  aether-archaeologist.md  (91 lines)
  aether-chaos.md          (98 lines)
  aether-architect.md      (66 lines)

Batch 6 - Surveyor Cluster (4 agents):
  aether-surveyor-nest.md        (272 lines)
  aether-surveyor-disciplines.md (334 lines)
  aether-surveyor-pathogens.md   (209 lines)
  aether-surveyor-provisions.md  (277 lines)
```

Note: workers.md (1,034 lines) is a reference document, not an agent — it is not in scope for cleanup.

### Boilerplate Taxonomy (what to strip)

Four sections are universally redundant in all 20 flat-markdown agents:

**Section 1: "Aether Integration" — STRIP**

Appears identically in all 20 flat-markdown agents:
```markdown
## Aether Integration

This agent operates as a **specialist worker** within the Aether Colony system. You:
- Report to the Queen/Prime worker who spawns you
- Log activity using Aether utilities
- Follow depth-based spawning rules
- Output structured JSON reports
```

Queen variant is slightly different (says "orchestrator" instead of "specialist worker") but still fully redundant — the spawn context already tells the agent its role.

Why safe to strip: The spawn prompt from workers.md (lines 717-763) already tells every agent who it reports to, what depth it is at, and to output JSON. This section adds zero actionable information.

**Section 2: "Depth-Based Behavior" table — STRIP**

Appears in all 20 flat-markdown agents with only the row-1 caste name varying:
```markdown
## Depth-Based Behavior

| Depth | Role | Can Spawn? |
|-------|------|------------|
| 1 | Prime {Caste} | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |
```

The Queen has a slightly different 4-row table (includes depth 0). Both are redundant — spawn limits are injected via spawn context at invocation time.

Why safe to strip: The spawn prompt already tells agents their depth and spawn limits. The table's rules do not change per-agent.

**Section 3: "Reference" footer — STRIP**

Appears identically in all 20 flat-markdown agents:
```markdown
## Reference

Full worker specifications: `.aether/workers.md`
```

Why safe to strip: This line does not cause any LLM to actually read workers.md. It is inert text pointing to a file that is not automatically loaded. The 2026-02-18 architect analysis (in `docs/plans/`) confirmed this explicitly: "this line does not actually cause the LLM to read the file — it is inert text."

Exceptions: aether-archaeologist.md and aether-chaos.md have slightly different Reference sections:
- archaeologist: adds `Archaeology command documentation: .claude/commands/ant/archaeology.md`
- chaos: adds `Chaos command documentation: .claude/commands/ant/chaos.md`

These additional lines are still inert but note that specific commands exist. Decision call: strip the Reference section header and base line; keep the command-specific lines if they plausibly help the agent (archaeologist and chaos already have their workflows fully described — these lines add nothing).

**Section 4: "Activity Logging" — STRIP the section, compress to inline**

Appears in all 20 agents with only the caste name varying:
```markdown
## Activity Logging

Log {verb} as you work:
```bash
bash .aether/aether-utils.sh activity-log "ACTION" "{your_name} ({Caste})" "description"
```

Actions: {CASTE-SPECIFIC-ACTIONS}
```

Why this is a judgment call: The log command syntax and the Actions list are useful. The section header and surrounding prose are not. Two options:
- Option A: Strip entirely (~5 lines saved per agent, log command lost)
- Option B: Keep the bash command + Actions line, remove the section header and introductory sentence

Recommendation: Option B. The log command and action keywords are genuinely caste-specific and functional. Strip the `## Activity Logging` header and the preceding sentence; keep the code block and Actions line inline as part of the role section or a brief note.

Actually on reflection, given the user's "when in doubt keep it" preference and safety-first approach: keep the Activity Logging section as-is. The 5 lines are genuinely useful reference for the agent and the caste-specific Actions list differs per agent. This is not pure filler — it is operational instruction. Keep it.

**Final count of what to strip per flat-markdown agent:**
- Section 1 (Aether Integration): ~6 lines
- Section 2 (Depth-Based Behavior): ~5 lines
- Section 3 (Reference): ~3 lines
- **Total:** ~14 lines per agent stripped

**Estimated result:** Agents go from 80-138 lines to roughly 66-124 lines. A modest but meaningful reduction.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Identifying all redundant sections | Custom diff tooling | Manual per-file inspection using the taxonomy above | The boilerplate is well-catalogued; tooling overhead not worth it for 20 files |
| Verifying agent behavior | New test suite | `npm test` + `npm run lint:sync` | Existing tests cover distribution integrity |

**Key insight:** This is a content edit task, not a code task. The risk is accidentally removing unique content; the defense is reading each file carefully and using the explicit taxonomy above.

---

## Common Pitfalls

### Pitfall 1: Stripping Unique Content That Looks Like Boilerplate

**What goes wrong:** The "Spawning Sub-Workers" section in aether-builder.md and the "Spawning" section in aether-scout.md look similar to generic depth content but contain unique, agent-specific spawn rules.

**Why it happens:** Both sections discuss spawning, and the surface structure resembles the Depth-Based Behavior table. But builder's spawning section explains WHEN to spawn (3x complexity rule, specific criteria, code block for how to run spawn utilities). Scout's spawning section describes spawning a parallel scout for parallel research. These are unique to the agent.

**How to avoid:** Use the explicit taxonomy. Only strip the four listed sections. Do not strip anything else without deliberate per-case judgment.

**Warning signs:** If the section contains a numbered list of when/why to spawn, it is unique content. If it is a 3-column table with depth numbers, it is the boilerplate Depth-Based Behavior section.

### Pitfall 2: The Watcher's Flag-Creation Section

**What goes wrong:** aether-watcher.md contains a "Creating Flags for Failures" section with a bash command for `flag-add`. This is unique watcher-specific content that could be mistaken for generic boilerplate (it contains a code block with aether-utils.sh, similar to Activity Logging).

**Why it happens:** Activity Logging is boilerplate. Flag creation looks similar. But flag creation is a watcher-specific capability.

**How to avoid:** Flag creation stays. The test: does this section appear in other agents? No — flag-add is only in the watcher.

### Pitfall 3: Queen's Spawn Protocol vs Depth Table

**What goes wrong:** The Queen has both a "Spawn Protocol" section (unique, with generate-ant-name commands) AND a "Depth-Based Behavior" table (boilerplate). They are adjacent. Stripping the depth table while keeping the spawn protocol requires careful distinction.

**Why it happens:** Both are about spawning. But the Spawn Protocol contains actual shell commands the Queen uses to spawn workers — that is unique operational content. The Depth-Based Behavior table is the generic 4-row depth rules table that is identical (with minor row 0 addition) to all other agents.

**How to avoid:** Keep the "Spawn Protocol" section and the "Spawn Limits" bullet list. Strip only the "Depth-Based Behavior" table itself.

### Pitfall 4: Architect and Route-Setter Model Context Sections

**What goes wrong:** aether-architect.md and aether-route-setter.md each have a "Model Context" section listing a specific model (glm-5, kimi-k2.5) and benchmark scores. These are not standard boilerplate — they appear in only 2 agents.

**Why it happens:** The 2026-02-18 architect research recommended removing model references. But the user's context says this phase is boilerplate cleanup, not "outdated content removal." Model context sections are not part of the four-section boilerplate taxonomy.

**How to avoid:** Do not strip Model Context sections in this phase. They are not the identified boilerplate. Noted in STATE.md as tech debt but flagged as deferred ("outdated references: DO NOT fix — only strip boilerplate").

### Pitfall 5: Surveyor Agents Don't Need the Same Cleanup

**What goes wrong:** Someone applies the flat-markdown boilerplate taxonomy to the XML-structured surveyor agents.

**Why it happens:** Phase scope says "all 25 agents" but the surveyors are structurally different — they use XML tags, have no Aether Integration section, no Depth-Based Behavior table, and no Reference footer.

**How to avoid:** Surveyor agents (nest, disciplines, pathogens, provisions) have zero of the four boilerplate sections. They do have a `tools:` line in frontmatter that the other agents lack. They should be reviewed for any genuinely redundant content, but the standard taxonomy does not apply to them.

For surveyors, review: do they have any near-duplicate prose across the four files? They share similar `<consumption>` table structures — but these tables are unique per surveyor (different document names, different phase types loaded). Keep them.

Surveyor batch recommendation: In batch 6, review each surveyor for any genuinely redundant sections unique to this family. Likely finding: nothing to strip. The surveyors are already lean and prescriptive.

---

## Code Examples

### Identifying the Boilerplate Sections

Confirmed by reading all 20 flat-markdown agents. The exact text of the three sections to strip:

**Section 1 — Aether Integration (exact text in 19 of 20 agents):**
```
## Aether Integration

This agent operates as a **specialist worker** within the Aether Colony system. You:
- Report to the Queen/Prime worker who spawns you
- Log activity using Aether utilities
- Follow depth-based spawning rules
- Output structured JSON reports
```

**Section 1 — Aether Integration (Queen variant — also safe to strip):**
```
## Aether Integration

This agent operates as the **orchestrator** of the Aether Colony system. You:
- Set colony intention and manage state
- Spawn specialized workers by caste
- Log activity using Aether utilities
- Synthesize results and advance phases
- Output structured JSON reports
```

**Section 2 — Depth-Based Behavior (typical, 19 of 20 non-Queen agents):**
```
## Depth-Based Behavior

| Depth | Role | Can Spawn? |
|-------|------|------------|
| 1 | Prime {CasteName} | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |
```

**Section 2 — Depth-Based Behavior (Queen variant — 4-row version):**
```
## Depth-Based Behavior

| Depth | Role | Can Spawn? |
|-------|------|------------|
| 0 | Queen | Yes (max 4) |
| 1 | Prime Worker | Yes (max 4) |
| 2 | Specialist | Only if surprised |
| 3 | Deep Specialist | No |
```

**Section 3 — Reference (base form, 18 of 20 agents):**
```
## Reference

Full worker specifications: `.aether/workers.md`
```

**Section 3 — Reference (archaeologist variant):**
```
## Reference

Full worker specifications: `.aether/workers.md`
Archaeology command documentation: `.claude/commands/ant/archaeology.md`
```

**Section 3 — Reference (chaos variant):**
```
## Reference

Full worker specifications: `.aether/workers.md`
Chaos command documentation: `.claude/commands/ant/chaos.md`
```

For archaeologist and chaos: strip the base "Full worker specifications" line. The command-specific lines add marginal value; the user's "when in doubt keep it" rule suggests keeping those single extra lines. However, given that these lines are also inert (they don't cause any file to be read), stripping them entirely is also defensible. Call: strip entirely — simplicity wins and both agents have fully detailed workflows that make these references unnecessary.

---

### Description Format Standardization

12 agents use an older short description format. 12 agents use the newer "Use this agent for..." format. The pattern should be consistent:

**Old format (12 agents to update):**
```
description: "Builder ant - implements code, executes commands, manipulates files"
description: "Watcher ant - validates, tests, ensures quality, guards the colony"
description: "Scout ant - researches, gathers information, explores documentation"
description: "Queen ant orchestrator for Aether colony - coordinates phases and spawns workers"
description: "Route-setter ant - creates structured phase plans and analyzes dependencies"
description: "Archaeologist ant - git historian that excavates why code exists"
description: "Architect ant - synthesizes knowledge and coordinates documentation"
description: "Chaos ant - resilience tester that probes edge cases and boundary conditions"
description: "Surveyor ant - maps architecture and directory structure for colony intelligence"
description: "Surveyor ant - maps coding conventions and testing patterns for colony intelligence"
description: "Surveyor ant - identifies technical debt, bugs, and concerns for colony health"
description: "Surveyor ant - maps technology stack and external integrations for colony intelligence"
```

**New format target (use these 12 as the pattern):**
```
description: "Use this agent for third-party API integration, SDK setup, and external service connectivity. The ambassador bridges your code with external systems."
description: "Use this agent for code review, quality audits, and compliance checking. The auditor examines code with specialized lenses for security, performance, and maintainability."
description: "Use this agent for documentation generation, README updates, and API documentation. The chronicler preserves knowledge in written form."
description: "Use this agent for dependency management, supply chain security, and license compliance. The gatekeeper guards what enters your codebase."
description: "Use this agent for security audits, vulnerability scanning, and threat assessment. The guardian patrols for security vulnerabilities and protects the codebase."
description: "Use this agent for accessibility audits, WCAG compliance checking, and inclusive design validation. The includer ensures all users can access your application."
description: "Use this agent for knowledge curation, pattern extraction, and maintaining project wisdom. The keeper organizes patterns and maintains institutional memory."
description: "Use this agent for performance profiling, bottleneck detection, and optimization analysis. The measurer benchmarks and optimizes system performance."
description: "Use this agent for test generation, mutation testing, and coverage analysis. The probe digs deep to expose hidden bugs and edge cases."
description: "Use this agent for analytics, trend analysis, and extracting insights from project history. The sage reads patterns in data to guide decisions."
description: "Use this agent for systematic bug investigation, root cause analysis, and debugging complex issues. The tracker follows error trails to their source."
description: "Use this agent for code refactoring, restructuring, and improving code quality without changing behavior. The weaver transforms tangled code into clean patterns."
```

**Decision on description updates:** This is within AGENT-03 scope (fix names that don't match). The old descriptions are technically correct but shorter and less useful for agent routing. Updating them to the "Use this agent for..." pattern is low-risk and directly serves the "focused job description" goal. Update all 12 as part of the batches.

Note on the four surveyor agents: their descriptions all say "Surveyor ant - {specific mapping description}" which follows the old format but their descriptions are actually quite good (specific about what they map). Update them to "Use this agent for..." format too.

---

## State of the Art

| Old Approach | Current Approach | Applies To |
|--------------|------------------|------------|
| Full "Aether Integration" section | Remove — spawn context already provides this | Phase 22 work |
| Full Depth-Based Behavior table | Remove — spawn context already provides this | Phase 22 work |
| "Reference: workers.md" footer | Remove — inert text that doesn't actually load the file | Phase 22 work |
| Short description format ("Builder ant - ...") | "Use this agent for..." format | Description updates in Phase 22 |

**Deferred (not Phase 22):**
- Converting to XML structure (flagged as high-impact in architect review but not in scope)
- Adding failure_modes and success_criteria sections (Phase 23: Agent Resilience)
- Full XML rewrite of flat-markdown agents (deferred in STATE.md decisions)
- Agent merging (Architect into Keeper, Guardian into Auditor) (out of scope for Phase 22)

---

## Open Questions

1. **Activity Logging section: keep or strip?**
   - What we know: The section is 5 lines. The content is unique per agent (different Actions keywords). The log command itself is useful operational guidance.
   - What's unclear: Whether stripping it would measurably harm agent behavior.
   - Recommendation: Keep the Activity Logging section. The caste-specific Actions list ("Actions: CREATED, MODIFIED, EXECUTING, DEBUGGING, ERROR" vs "Actions: RESEARCH, DISCOVERED, SYNTHESIZING...") is genuinely agent-specific guidance, not generic boilerplate. The user's "when in doubt keep it" rule applies here.

2. **Verification method for each batch**
   - What we know: `npm test` and `npm run lint:sync` exist and cover distribution integrity.
   - What's unclear: Whether there is a quick way to confirm an agent still "loads" correctly without actually spawning it in a live session.
   - Recommendation: After each batch, run `npm run lint:sync` and `npm test`. For a more thorough check, visually confirm each edited file still has all its unique sections intact (role, workflow, output format). The test is primarily "did anything accidentally get deleted that was unique."

3. **Surveyor agents: description updates**
   - What we know: All 4 surveyor descriptions follow the old format. The context decision table indicates the user wants description names fixed.
   - What's unclear: Whether the surveyor descriptions need the "Use this agent for..." pattern since they are invoked differently (by command, not by agent selection).
   - Recommendation: Update surveyor descriptions to "Use this agent for..." pattern for consistency. Low-risk change.

---

## Sources

### Primary (HIGH confidence)

- Direct reading of all 25 agent files in `/Users/callumcowie/repos/Aether/.opencode/agents/` — full content of each file inspected
- `docs/plans/2026-02-18-agent-definition-architecture-plan.md` — 1,061-line LLM architect analysis confirming boilerplate sections and token cost estimates
- `docs/plans/2026-02-18-agent-improvement-synthesis.md` — synthesis document confirming what both analyses agreed on
- `.planning/phases/22-agent-boilerplate-cleanup/22-CONTEXT.md` — locked user decisions

### Secondary (MEDIUM confidence)

- `workers.md` read in full — confirmed it is a reference document loaded by developers, not by agents at runtime; confirms the "Reference" footer in agents is indeed inert

### Tertiary (LOW confidence)

- None required — all findings are directly observable in the agent files

---

## Metadata

**Confidence breakdown:**
- What to strip (the 3 sections): HIGH — confirmed identical across all 20 flat-markdown agents by direct inspection
- Activity Logging decision (keep): HIGH — unique caste-specific content confirmed
- Description format standardization: HIGH — pattern visible in the 12 newer agents
- Surveyor handling: HIGH — confirmed they have zero of the boilerplate sections
- Verification approach: MEDIUM — `npm test` and `npm run lint:sync` are confirmed to exist; their exact coverage of agent file changes is not tested

**Research date:** 2026-02-19
**Valid until:** Until agent file structure changes (stable; these files change rarely)

---

## Quick Reference for Planner

**Files to edit:** 20 flat-markdown agent files (all `.opencode/agents/aether-*.md` except the 4 surveyors, plus the surveyors for description updates only)

**Three sections to strip from each flat-markdown agent:**
1. `## Aether Integration` (with its 4-5 bullet points)
2. `## Depth-Based Behavior` (with its 3-4 row table)
3. `## Reference` (with the `Full worker specifications: .aether/workers.md` line)

**One additional change — description format:**
Update old-format descriptions to "Use this agent for..." pattern in 12 agents.

**Do NOT strip:**
- `## Activity Logging` (caste-specific Actions list has unique value)
- Any section unique to that agent (TDD Discipline, Debugging Discipline, Verification Workflow, Security Domains, etc.)
- Any content in the 4 surveyor agents (they have none of the boilerplate)

**Batch order and sizes:**
- Batch 1: Core 5 (queen, builder, watcher, scout, route-setter)
- Batch 2: Development 4 (weaver, probe, ambassador, tracker)
- Batch 3: Knowledge 4 (chronicler, keeper, auditor, sage)
- Batch 4: Quality 4 (guardian, measurer, includer, gatekeeper)
- Batch 5: Special 3 (archaeologist, chaos, architect)
- Batch 6: Surveyor 4 (description updates only — no boilerplate to strip)

**Verify after each batch:** `npm run lint:sync && npm test`
