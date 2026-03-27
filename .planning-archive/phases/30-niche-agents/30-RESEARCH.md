# Phase 30: Niche Agents - Research

**Researched:** 2026-02-20
**Domain:** Claude Code subagent authoring (8 niche agents) + test suite expansion
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Plan organization:**
- Group agents by similarity into 2-3 plans (Claude's discretion on exact grouping)
- All plans run in wave 1 (parallel) — agents are independent, no dependencies between them
- Separate verification plan that updates TEST-05 expected count from 14 to 22 and runs the full test suite
- Verification plan runs in wave 2 (after all agent plans complete)

**Read/Write permissions:**
- Chaos agent: read-only (no Write/Edit). Analyzes code for edge cases but cannot modify anything
- Ambassador: full access as specced (Read, Write, Edit, Bash, Grep, Glob) — needs Bash for SDK installs and API calls
- Chronicler: Claude's discretion on whether to include Edit alongside Write
- Sage: Claude's discretion on whether to add Write for persisting analysis reports
- All other niche agents (Archaeologist, Gatekeeper, Includer, Measurer): read-only as specced in requirements

**Agent triggers (routing descriptions):**
- Archaeologist: primary value is regression prevention — excavates git history to find patterns of what was done before, ensures we're not repeating past mistakes or undoing previous fixes
- Gatekeeper: Claude's discretion on exact scope (dependencies only vs dependencies + import graphs)
- Includer: Claude's discretion on depth — assess based on typical project needs
- Measurer: Claude's discretion — scope for general profiling across project types, not just Aether-specific
- Each description must name a specific trigger case, not a generic role label (per success criteria)

**Agent depth and quality:**
- Equal depth across all 8 agents — no agent gets less attention than others
- Same 8-section XML template as Phases 28-29 (role, execution_flow, critical_rules, return_format, success_criteria, failure_modes, escalation, boundaries)
- Fresh designs — NOT mechanical ports from OpenCode definitions. Use the PWR template and design each agent to be best-in-class
- Same high quality bar as Phases 28-29: genuinely powerful agents you'd actually want to invoke

### Claude's Discretion

- Exact plan grouping (how to batch the 8 agents into 2-3 plans)
- Chronicler Edit tool inclusion
- Sage Write tool inclusion
- Gatekeeper trigger scope
- Includer and Measurer execution flow depth
- Internal execution flow details for all agents

### Deferred Ideas (OUT OF SCOPE)

- OpenCode agent sync (keeping Claude Code and OpenCode agents in parity) — listed as future requirement, not this phase
- Agent A/B testing for routing effectiveness — future requirement
- Agent metrics/telemetry — future requirement
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| NICHE-01 | Chaos agent upgraded — resilience testing, edge cases, boundary probing. Tools: Read, Bash, Grep, Glob (no Write/Edit) | OpenCode source in `.opencode/agents/aether-chaos.md` provides content foundation; locked as read-only; 5-category investigation framework (edge cases, boundary conditions, error handling, state corruption, unexpected inputs) is solid and ports well |
| NICHE-02 | Archaeologist agent upgraded — git history excavation, tribal knowledge surfacing. Tools: Read, Bash, Grep, Glob (no Write/Edit) | OpenCode source in `.opencode/agents/aether-archaeologist.md`; primary value framed as regression prevention (locked decision); git history investigation tools (git log, git blame, git show, git log --follow) are all Bash-based and supported |
| NICHE-03 | Ambassador agent upgraded — third-party API integration, SDK setup. Tools: Read, Write, Edit, Bash, Grep, Glob | OpenCode source in `.opencode/agents/aether-ambassador.md`; full tool set required for SDK installs + API calls; security boundary (no secrets in files) is well-established in source |
| NICHE-04 | Chronicler agent upgraded — documentation generation, knowledge preservation. Tools: Read, Write, Grep, Glob | OpenCode source in `.opencode/agents/aether-chronicler.md`; Edit inclusion is Claude's discretion (recommendation: include Edit — needed for inline JSDoc/TSDoc comment updates in existing files); write scope well-defined in source |
| NICHE-05 | Gatekeeper agent upgraded — dependency audit, license compliance, supply chain review. Tools: Read, Grep, Glob (no Write/Edit/Bash) | OpenCode source in `.opencode/agents/aether-gatekeeper.md`; no Bash means no `npm audit` — research finds this is appropriate (read manifests, flag findings, let Builder run remediation commands); scope recommendation: dependencies + import graph analysis via Grep/Glob is achievable without Bash |
| NICHE-06 | Measurer agent upgraded — performance profiling, bottleneck detection. Tools: Read, Bash, Grep, Glob (no Write/Edit) | OpenCode source in `.opencode/agents/aether-measurer.md`; Bash enables running benchmark commands; scope should be cross-project (not Aether-specific) with static analysis fallback when dynamic profiling is unavailable |
| NICHE-07 | Includer agent upgraded — accessibility audit, WCAG compliance. Tools: Read, Grep, Glob (no Write/Edit/Bash) | OpenCode source in `.opencode/agents/aether-includer.md`; no Bash means no automated scanner execution — static code analysis of HTML/ARIA/CSS is achievable; WCAG 2.1 AA criteria provide a concrete checklist |
| NICHE-08 | Sage agent upgraded — analytics, pattern extraction, trend analysis. Tools: Read, Grep, Glob, Bash (no Write/Edit) | OpenCode source in `.opencode/agents/aether-sage.md`; Bash enables git log analysis, file counting, timestamp extraction; Write discretion: recommend NOT including Write (Sage is analysis-only, persisting reports is Builder's job); analysis scope should be cross-project |
</phase_requirements>

---

## Summary

Phase 30 creates 8 niche Claude Code subagents in `.claude/agents/ant/`, completing the full 22-agent roster. The existing AVA test suite (`tests/unit/agent-quality.test.js`) already covers all quality gates dynamically — new agents pass TEST-01 through TEST-04 and the body quality check automatically once they exist. The only code change required is updating `EXPECTED_AGENT_COUNT` from 22 to 22 in TEST-05 (it's already set to 22; what's needed is verifying all 22 agents pass) and expanding `READ_ONLY_CONSTRAINTS` in the test to include the 6 new read-only niche agents.

Each niche agent has an OpenCode source file that provides content direction, but the Claude Code versions must be fresh designs following the 8-section XML template established in Phases 28-29. The hardest design work is Archaeologist (framed as regression prevention, not just archaeology), Gatekeeper (achieving useful dependency analysis without Bash/npm audit), and Includer (static code accessibility analysis without an automated scanner). The most straightforward agents are Ambassador and Chronicler (clear tool sets, well-understood domains).

The verification plan is the only wave-2 dependency: it expands `READ_ONLY_CONSTRAINTS` in the test file, confirms all 22 agents pass TEST-01 through TEST-05 and the body quality check, and confirms the full `npm test` suite is clean.

**Primary recommendation:** Group agents into 2 plans (read-only investigators: Chaos + Archaeologist + Gatekeeper + Includer + Measurer + Sage; writers: Ambassador + Chronicler), run both in wave 1 in parallel, then run the verification plan in wave 2.

---

## Standard Stack

### Core

| Component | Version/Path | Purpose | Why Standard |
|-----------|-------------|---------|--------------|
| Claude Code agent format | `.claude/agents/ant/*.md` | File format for all 8 agents | Established in Phase 27; all existing 14 agents follow this format; test suite validates compliance |
| AVA test runner | 6.4.1 (installed) | Runs agent quality tests | Already in project; `npm test` invokes `ava` targeting `tests/unit/**/*.test.js` |
| js-yaml | 4.1.1 (installed) | YAML frontmatter parsing in tests | Already in project; used by `tests/unit/agent-quality.test.js` already |
| 8-section XML body template | Established Phase 27 | Body structure for all agents | All 14 existing agents use this; test suite enforces presence of all 8 sections |

### OpenCode Source References (Content Direction, Not Templates)

| OpenCode Source | Provides | How To Use |
|----------------|----------|-----------|
| `.opencode/agents/aether-chaos.md` | 5-category investigation framework, severity guide, output JSON schema | Read for content direction; strip activity-log; rewrite in 8-section XML format |
| `.opencode/agents/aether-archaeologist.md` | Git investigation tools, key findings categories, output JSON schema | Read for content direction; reframe as regression prevention; strip activity-log |
| `.opencode/agents/aether-ambassador.md` | Integration patterns, error handling strategy, security considerations | Read for content direction; strip activity-log; expand to 8-section format |
| `.opencode/agents/aether-chronicler.md` | Documentation types, writing principles, output JSON schema, write scope | Read for content direction; strip activity-log; resolve Edit tool decision |
| `.opencode/agents/aether-gatekeeper.md` | Dependency audit dimensions, license categories, severity levels | Read for content direction; redesign for Bash-free operation; strip activity-log |
| `.opencode/agents/aether-includer.md` | WCAG dimensions (visual, motor, cognitive, hearing), compliance levels | Read for content direction; redesign for Bash-free/scanner-free static analysis; strip activity-log |
| `.opencode/agents/aether-measurer.md` | Performance dimensions, optimization strategies, output JSON schema | Read for content direction; reframe for cross-project use; strip activity-log |
| `.opencode/agents/aether-sage.md` | Analysis areas, visualization types, output JSON schema | Read for content direction; strip activity-log; resolve Write tool decision |

### Gold Standard Reference Files (Existing Claude Code Agents)

| Agent | Lines | What To Reference |
|-------|-------|-------------------|
| `aether-tracker.md` | 265 | Read-only agent with Bash — boundary enforcement pattern, evidence-based investigation |
| `aether-auditor.md` | 266 | Strictest read-only (no Bash) — how to do deep analysis without tool execution |
| `aether-builder.md` | 187 | Full tool set — escalation format, tiered failure handling |
| `aether-watcher.md` | 244 | Read-only with Bash — verification workflow structure |
| `aether-surveyor-disciplines.md` | 416 | Longest agent — how to handle complex multi-domain analysis |

### No New Dependencies

Zero new libraries required. No `npm install` step needed. Both the agent files and the test update use already-installed packages.

---

## Architecture Patterns

### Pattern 1: The 8-Section XML Template (Mandatory for All Agents)

All 8 niche agents MUST use this structure. The test suite enforces it.

```
---
name: aether-{role}
description: "Use this agent when {specific trigger case}. ..."
tools: {comma-separated list}
model: inherit
---

<role>
[Colony identity, core mandate, read-only/write declaration]
</role>

<execution_flow>
[Numbered workflow steps — what the agent actually does]
</execution_flow>

<critical_rules>
[Non-negotiable rules — named iron laws]
</critical_rules>

<return_format>
[Output JSON schema with example — ALL fields documented]
</return_format>

<success_criteria>
[Self-check list before reporting complete]
</success_criteria>

<failure_modes>
[Tiered: minor (retry up to 2), major (STOP immediately). Never fail silently.]
</failure_modes>

<escalation>
[When to route to Queen vs Builder vs other specialist. No sub-worker spawning.]
</escalation>

<boundaries>
[Global protected paths + agent-specific boundaries]
</boundaries>
```

**Minimum viable section:** 50 characters of content. The body quality test enforces this. Each section needs substantive content — the OpenCode sources tend to be thin (100-130 lines total); the Claude Code versions should target 180-260 lines.

### Pattern 2: Tool Assignments Per Agent

```yaml
# Chaos (NICHE-01) — resilience testing, read-only
tools: Read, Bash, Grep, Glob

# Archaeologist (NICHE-02) — git history, read-only
tools: Read, Bash, Grep, Glob

# Ambassador (NICHE-03) — API integration, full access
tools: Read, Write, Edit, Bash, Grep, Glob

# Chronicler (NICHE-04) — documentation writing
# RECOMMENDATION: include Edit (see Architecture section)
tools: Read, Write, Edit, Grep, Glob

# Gatekeeper (NICHE-05) — dependency audit, most restrictive
tools: Read, Grep, Glob

# Measurer (NICHE-06) — performance profiling, read-only with Bash
tools: Read, Bash, Grep, Glob

# Includer (NICHE-07) — accessibility audit, most restrictive
tools: Read, Grep, Glob

# Sage (NICHE-08) — analytics and trend analysis
# RECOMMENDATION: no Write (see Architecture section)
tools: Read, Grep, Glob, Bash
```

### Pattern 3: Description Style — Specific Trigger Case Required

Per success criteria: "each niche agent description names a specific trigger case — not a generic role label."

**Anti-pattern (generic role label):**
```
"Use this agent for accessibility audits and WCAG compliance."
```

**Correct (specific trigger case):**
```
"Use this agent when you need to audit an interface for accessibility — static analysis of HTML structure, ARIA attributes, color contrast ratios, and keyboard navigation patterns against WCAG 2.1 AA criteria. Invoked when a component is complete and needs accessibility sign-off before merge, or when a user reports accessibility issues. Returns findings with WCAG criterion references and suggested fixes for Builder. Do NOT use for implementation fixes (use aether-builder)."
```

Key differentiators to include in each description:
- **What triggers it specifically** (not "for X purposes" but "when Y situation arises")
- **What it returns** (so callers know what they get)
- **What it routes to** (so callers know what comes next)
- **What NOT to use it for** (routing guardrails)

### Pattern 4: Test Suite Expansion for READ_ONLY_CONSTRAINTS

The existing test file at `tests/unit/agent-quality.test.js` currently registers only 2 read-only agents:

```javascript
const READ_ONLY_CONSTRAINTS = {
  'aether-tracker': { forbidden: ['Write', 'Edit'] },
  'aether-auditor': { forbidden: ['Write', 'Edit', 'Bash'] },
};
```

The verification plan must expand this to include the 6 new read-only niche agents:

```javascript
const READ_ONLY_CONSTRAINTS = {
  // Phase 29 agents
  'aether-tracker':       { forbidden: ['Write', 'Edit'] },
  'aether-auditor':       { forbidden: ['Write', 'Edit', 'Bash'] },
  // Phase 30 niche agents — read-only
  'aether-chaos':         { forbidden: ['Write', 'Edit'] },
  'aether-archaeologist': { forbidden: ['Write', 'Edit'] },
  'aether-gatekeeper':    { forbidden: ['Write', 'Edit', 'Bash'] },
  'aether-includer':      { forbidden: ['Write', 'Edit', 'Bash'] },
  'aether-measurer':      { forbidden: ['Write', 'Edit'] },
  'aether-sage':          { forbidden: ['Write', 'Edit'] },
};
```

**Why gatekeeper and includer also ban Bash:** Per NICHE-05 and NICHE-07 specs, Gatekeeper has `Read, Grep, Glob` (no Bash) and Includer has `Read, Grep, Glob` (no Bash). The test must enforce this.

**EXPECTED_AGENT_COUNT stays 22** — it was already set to 22 in Phase 29. No change to that constant. The test passes when all 22 agents exist.

### Pattern 5: Colony Cross-Reference Pattern (Escalation Wiring)

Each niche agent should reference the specific colony workers it routes to. This makes the colony feel connected.

| Agent | Primary Escalation | Secondary Escalation |
|-------|-------------------|---------------------|
| Chaos | Builder (fix implementation) | Queen (if systemic weakness found) |
| Archaeologist | Builder (for changes informed by history) | Queen (architectural decisions about legacy code) |
| Ambassador | Queen (for API key/credential decisions) | Builder (for implementation) |
| Chronicler | Builder (if source contradicts docs — Builder fixes source) | Queen (if scope is ambiguous) |
| Gatekeeper | Builder (dependency updates) | Queen (for license compliance decisions) |
| Measurer | Builder (optimization implementation) | Queen (for architectural performance decisions) |
| Includer | Builder (HTML/ARIA fixes) | Queen (if scope requires design system changes) |
| Sage | Queen (for strategic decisions from analysis) | Keeper (for knowledge preservation of trends) |

### Pattern 6: Chronicler Edit Tool Decision

**Recommendation: INCLUDE Edit in Chronicler's tools.**

Rationale: The primary use case for Edit vs Write is modifying existing files in-place. Chronicler's job includes updating inline JSDoc/TSDoc comments within existing source files. Without Edit, Chronicler would need to use Write to overwrite entire files to add/update a comment — which is error-prone and risks losing non-comment content. Edit's surgical replacement capability is exactly right for "add JSDoc above this function" operations.

Chronicler's `<boundaries>` section should restrict Edit to comment lines only: "Do not use Edit to modify logic, only to add or update documentation comments (JSDoc/TSDoc)."

Final tool set: `Read, Write, Edit, Grep, Glob`

### Pattern 7: Sage Write Tool Decision

**Recommendation: Do NOT include Write in Sage's tools.**

Rationale: Sage is an analysis-only agent. Its value is in extracting patterns and returning findings to the caller. If Sage needs to persist a report, the caller (Queen or Builder) can take the Sage return and write it. Adding Write to Sage would blur the analysis/action boundary. Sage with Write could drift into creating files "for convenience" — undermining the clean read-only analysis posture that makes its findings trustworthy.

Final tool set: `Read, Grep, Glob, Bash`

Note: Bash enables Sage to run `git log --oneline`, file counting, and timestamp-based analysis — this is the right power set for trend analysis without write capability.

### Anti-Patterns to Avoid

- **Keeping activity-log calls:** Every OpenCode niche agent has `bash .aether/aether-utils.sh activity-log "..."`. Remove completely. TEST-04 catches this.
- **Thin sections from thin OpenCode sources:** The OpenCode niche agents are thin (roughly 100-120 lines). A direct port would fail the 50-char minimum per section. Each Claude Code version must expand with real execution flow detail.
- **Generic descriptions:** "Use this agent for dependency management" fails the success criteria. Must name a specific trigger case.
- **Missing `model: inherit`:** All Phase 28-29 agents include this field. Include in all 8 niche agents.
- **Gatekeeper suggesting npm audit fix:** Gatekeeper has no Bash. Any remediation suggestions go in the `recommendations` array. Builder runs the actual fix commands.
- **Includer citing automated scanner results:** Includer has no Bash, so it cannot run axe, Lighthouse, or WAVE. Its analysis must be based on static code inspection only. Completion report must state "manual static analysis" as the testing method.
- **Archaeologist framed as "history" instead of "regression prevention":** The locked decision is specific — Archaeologist's primary value is preventing regression. Description and execution_flow must lead with this framing.
- **Ambassador writing API keys to files:** Ambassador's critical_rules must have a named iron law: "Never write credentials, API keys, or secrets to any tracked file. Document the required env var names and instruct the user to set them." This is the most important safety rule for Ambassador.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML frontmatter validation | Custom validation logic | Existing test suite (TEST-01 through body quality) catches all issues automatically | Already runs; any malformed agent file fails npm test immediately |
| Dependency vulnerability lookup | Custom CVE lookup in Gatekeeper body | `npm audit` output via Bash (Gatekeeper has no Bash) — document `npm audit` in `recommendations` for Builder to run | Gatekeeper can inspect manifests statically; dynamic scanning is Builder's domain |
| Accessibility scanner execution | Includer running axe/Lighthouse | Static WCAG 2.1 AA checklist applied to source code | Includer has no Bash; static analysis is achievable and honest — no fabricated scores |
| Agent content templates | Copy-pasting section headers across agents | Builder reads the gold standard (`aether-tracker.md` or `aether-auditor.md`) for each new agent and reimagines fresh | Direct copy of structure without fresh content produces thin agents that fail body quality checks |

---

## Common Pitfalls

### Pitfall 1: OpenCode Sources Are Too Thin for Direct Port

**What goes wrong:** The OpenCode niche agents are 100-120 lines each. A line-for-line port to the 8-section XML format would produce agents that fail the body quality check (50-char minimum per section) or pass the check with technically-present but useless sections.

**Why it happens:** OpenCode agents were written for a different format (activity-log pattern, no explicit 8-section structure). The content that exists is correct but sparse.

**How to avoid:** Each niche agent needs fresh content written specifically for its sections. Use the OpenCode source for domain knowledge (what the agent knows, what it investigates, what its output looks like), not for body text. Target 180-260 lines per agent — comparable to tracker and auditor.

**Warning signs:** If any niche agent body is under 100 lines total, it is under-specified. The OpenCode sources are all under 120 lines; the target should be 150-260 lines for the Claude Code versions.

### Pitfall 2: Gatekeeper Using npm audit Without Bash

**What goes wrong:** Gatekeeper's execution_flow describes running `npm audit` or `pip audit` as part of its workflow, but Gatekeeper has no Bash tool.

**Why it happens:** The OpenCode Gatekeeper failure_modes section mentions "npm audit, pip audit" — direct port to execution_flow would be incorrect.

**How to avoid:** Gatekeeper performs static analysis only — reading `package.json`, `package-lock.json`, `requirements.txt`, etc. via Read, Grep, and Glob. For known vulnerability patterns it identifies from manifest inspection, it uses Grep to search for CVE advisories in lock files. It recommends `npm audit` in its output as an action for Builder to take, but does not run it itself.

**What Gatekeeper CAN do without Bash:**
- Read `package.json` for dependency list and semver ranges
- Read `package-lock.json` for exact resolved versions
- Grep lock file for known vulnerable version patterns
- Read `LICENSE` files to identify dependency licenses
- Glob for dependency manifest files across monorepo structures

**Warning signs:** Gatekeeper execution_flow contains `npm audit`, `pip audit`, `yarn audit`, or any command invocation.

### Pitfall 3: Includer Fabricating Compliance Percentages

**What goes wrong:** Includer's return format includes `compliance_percent`. Without automated testing tools, any percentage is an estimate — and the OpenCode version's failure_modes say "Never fabricate compliance scores."

**Why it happens:** Automated tools (axe, Lighthouse, WAVE) produce compliance percentages. Includer with no Bash cannot run them.

**How to avoid:** Includer's `return_format` should include `compliance_percent` with a required `analysis_method` field that must say "manual static analysis" (not "automated scan"). The `success_criteria` must state: "Compliance percentage reflects only what was manually verified from source code — do not estimate beyond what was actually inspected."

**Warning signs:** Any compliance_percent value appears without citing the specific source code reviewed to derive it.

### Pitfall 4: Archaeologist Not Framed as Regression Prevention

**What goes wrong:** Archaeologist's description and execution_flow focus on "understanding history" (the general archaeologist metaphor) rather than "preventing regression" (the locked framing from CONTEXT.md).

**Why it happens:** The OpenCode source explicitly frames Archaeologist as "the colony's historian" and focuses on "why things are the way they are." The CONTEXT.md reframes its primary value as regression prevention.

**How to avoid:** The description must lead with the regression prevention framing: "Use this agent to excavate git history before making changes in an area — its primary job is ensuring you're not repeating past mistakes or undoing previously applied fixes." The `execution_flow` should have an explicit "regression check" step: after analyzing history, explicitly check whether the proposed change area has had bugs fixed, workarounds applied, or deliberate architectural choices that look like oddities but serve a purpose.

**Warning signs:** Archaeologist's description does not mention "regression" or "prevent undoing" or equivalent language.

### Pitfall 5: Ambassador Writing Credentials to Files

**What goes wrong:** Ambassador, in the process of setting up SDK authentication, writes an API key or secret to a configuration file that gets tracked by git.

**Why it happens:** SDK setup flows often involve writing credentials to config files. Without an explicit iron law in critical_rules, Ambassador might do this.

**How to avoid:** Ambassador's `<critical_rules>` must have a named iron law — "Credentials Iron Law" or similar — that is as prominent as Builder's "TDD Iron Law" or Tracker's "Diagnose Only" rule. The iron law must be: "Never write API keys, secrets, tokens, or credentials to any tracked file. If an SDK requires credentials, document the required environment variable names and instruct the user to set them. Run `grep -r 'KEY\|SECRET\|TOKEN' {integration_files}` as part of success_criteria to verify no literals appear."

**Warning signs:** Ambassador success_criteria does not include a credentials scan step.

### Pitfall 6: Missing READ_ONLY_CONSTRAINTS Update in Tests

**What goes wrong:** The 8 new niche agents are created, TEST-05 passes (22 agents found), but TEST-03 does not verify read-only constraints for the new agents — so Gatekeeper could have Write in its tools and the test would not catch it.

**Why it happens:** `READ_ONLY_CONSTRAINTS` in `tests/unit/agent-quality.test.js` is a registry that currently only includes `aether-tracker` and `aether-auditor`. New read-only agents are NOT automatically added — the registry must be explicitly updated.

**How to avoid:** The verification plan (wave 2) must update `READ_ONLY_CONSTRAINTS` to include all 6 read-only niche agents with their specific forbidden tool lists. This is the most critical code change in the verification plan.

**Warning signs:** `npm test` passes for TEST-03 after Phase 30 without the registry update — that is a false negative, not a real pass.

### Pitfall 7: TEST-05 Already Set to 22 — Planner Must Not Change It

**What goes wrong:** A planner or executor assumes TEST-05 needs to be "updated to 22" and changes `EXPECTED_AGENT_COUNT`. It is already 22.

**Why it happens:** The CONTEXT.md says "updates TEST-05 expected count from 14 to 22" — this is describing Phase 30's goal (making 22 agents exist), not a code change needed. The constant was set to 22 in Phase 29 as a forward-looking milestone.

**How to avoid:** The verification plan should verify that TEST-05 passes (22 agents found = 22 expected), not change the constant. The only code changes in the verification plan are: (1) expand `READ_ONLY_CONSTRAINTS`, and (2) run `npm test` to confirm all tests pass including TEST-05.

**Warning signs:** Any plan task that says "change EXPECTED_AGENT_COUNT from X to 22."

---

## Code Examples

Verified patterns from existing codebase:

### Frontmatter Templates for All 8 Niche Agents

```yaml
# Chaos (NICHE-01) — read-only resilience tester
---
name: aether-chaos
description: "Use this agent to stress-test code before or after changes — probing edge cases, boundary conditions, and error handling gaps that normal testing misses. Invoke when a feature is built and needs adversarial review, or when a bug appears that \"shouldn't be possible.\" Returns findings with severity ratings and reproduction steps. Fix implementation goes to aether-builder; missing test coverage goes to aether-probe."
tools: Read, Bash, Grep, Glob
model: inherit
---

# Archaeologist (NICHE-02) — read-only regression prevention
---
name: aether-archaeologist
description: "Use this agent before modifying code in an area with a complex or uncertain history — its primary job is regression prevention. Excavates git history to surface past bugs that were fixed, deliberate architectural choices that look like oddities, and areas that have been unstable. Returns a stability map and tribal knowledge report so you don't undo previous work. Do NOT use for implementation (use aether-builder) or refactoring (use aether-weaver)."
tools: Read, Bash, Grep, Glob
model: inherit
---

# Ambassador (NICHE-03) — full access API integrator
---
name: aether-ambassador
description: "Use this agent when adding a new third-party API integration, migrating to a new SDK version, or implementing webhook handlers. Ambassador researches the API, implements the integration with proper error handling (timeout, auth failure, rate limits), and verifies connectivity. Never commits credentials — documents required env vars for user to set. Routes implementation questions to aether-builder; SDK or auth decisions to Queen."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

# Chronicler (NICHE-04) — documentation writer (with Edit for inline comments)
---
name: aether-chronicler
description: "Use this agent when documentation is missing, outdated, or needs to be generated from code — READMEs, API docs, JSDoc/TSDoc inline comments, architecture diagrams, and changelogs. Invoke after a feature is complete and needs documentation, or when documentation gaps are identified in an audit. Does not modify source logic — documentation only. Reports gaps it cannot fill for Builder or Keeper to address."
tools: Read, Write, Edit, Grep, Glob
model: inherit
---

# Gatekeeper (NICHE-05) — read-only, most restrictive (no Bash)
---
name: aether-gatekeeper
description: "Use this agent when adding new dependencies, before a release, or when a security review is needed — audits dependency manifests for vulnerabilities, license compliance, and supply chain risks without running any commands. Static analysis of package.json, lock files, and license declarations. Returns findings with severity ratings and recommended actions for Builder to execute. Do NOT use for dependency updates (use aether-builder)."
tools: Read, Grep, Glob
model: inherit
---

# Measurer (NICHE-06) — read-only with Bash for profiling
---
name: aether-measurer
description: "Use this agent when performance is degrading, before optimization work to establish a baseline, or when bottlenecks need identification. Profiles code paths, runs benchmarks, analyzes algorithmic complexity, and identifies bottlenecks with file-level specificity. Returns prioritized optimization recommendations with estimated impact. Implementation goes to aether-builder; architectural performance decisions go to Queen."
tools: Read, Bash, Grep, Glob
model: inherit
---

# Includer (NICHE-07) — read-only, most restrictive (no Bash)
---
name: aether-includer
description: "Use this agent when an interface needs accessibility review — static analysis of HTML structure, ARIA attributes, semantic markup, color contrast (from CSS/design tokens), and keyboard navigation patterns against WCAG 2.1 AA criteria. Invoke before merge when accessibility is required, or when a user reports accessibility issues. Returns violations with WCAG criterion references and fix suggestions for Builder. Analysis is manual/static — no automated scanner."
tools: Read, Grep, Glob
model: inherit
---

# Sage (NICHE-08) — analytics, no Write (analysis-only posture)
---
name: aether-sage
description: "Use this agent to extract patterns and trends from project history — development velocity, bug density, knowledge concentration, churn hotspots, and quality trajectories over time. Invoke when retrospective analysis is needed, when decisions require data support, or when the colony needs to understand its own health. Returns findings, trends, and prioritized recommendations. Strategic decisions go to Queen; knowledge preservation goes to aether-keeper."
tools: Read, Grep, Glob, Bash
model: inherit
---
```

### READ_ONLY_CONSTRAINTS Expansion for Verification Plan

```javascript
// In tests/unit/agent-quality.test.js — replace existing READ_ONLY_CONSTRAINTS
const READ_ONLY_CONSTRAINTS = {
  // Phase 29 — specialists
  'aether-tracker':       { forbidden: ['Write', 'Edit'] },
  'aether-auditor':       { forbidden: ['Write', 'Edit', 'Bash'] },
  // Phase 30 — niche agents (read-only set)
  'aether-chaos':         { forbidden: ['Write', 'Edit'] },
  'aether-archaeologist': { forbidden: ['Write', 'Edit'] },
  'aether-gatekeeper':    { forbidden: ['Write', 'Edit', 'Bash'] },
  'aether-includer':      { forbidden: ['Write', 'Edit', 'Bash'] },
  'aether-measurer':      { forbidden: ['Write', 'Edit'] },
  'aether-sage':          { forbidden: ['Write', 'Edit'] },
};
```

### Return Format JSON Templates (Per Agent)

```json
// Chaos return format
{
  "ant_name": "{your name}",
  "caste": "chaos",
  "task_id": "{task_id}",
  "status": "completed" | "failed" | "blocked",
  "target": "{what was investigated}",
  "files_investigated": [],
  "scenarios": [
    {
      "id": 1,
      "category": "edge_cases" | "boundary_conditions" | "error_handling" | "state_corruption" | "unexpected_inputs",
      "status": "finding" | "resilient",
      "severity": "CRITICAL" | "HIGH" | "MEDIUM" | "LOW" | "INFO" | null,
      "title": "{finding title}",
      "description": "{detailed description}",
      "reproduction_steps": [],
      "expected_behavior": "{what should happen}",
      "actual_behavior": "{what would happen instead}"
    }
  ],
  "summary": {
    "total_findings": 0,
    "critical": 0,
    "high": 0,
    "medium": 0,
    "resilient_categories": 0
  },
  "top_recommendation": "{single most important action}"
}

// Archaeologist return format
{
  "ant_name": "{your name}",
  "caste": "archaeologist",
  "task_id": "{task_id}",
  "status": "completed" | "failed" | "blocked",
  "target": "{what was excavated}",
  "site_overview": {
    "total_commits_analyzed": 0,
    "date_range": "YYYY-MM-DD to YYYY-MM-DD"
  },
  "regression_risks": [
    {
      "area": "{file or module}",
      "risk": "Previously fixed bug that could be reintroduced if X is changed",
      "commit_evidence": "{commit hash and message}",
      "recommendation": "Do not modify without reviewing commit {hash}"
    }
  ],
  "stability_map": {
    "stable": [],
    "volatile": [],
    "unstable_with_context": []
  },
  "tribal_knowledge": [],
  "tech_debt_markers": [],
  "summary_for_newcomers": "{plain language overview}"
}

// Gatekeeper return format
{
  "ant_name": "{your name}",
  "caste": "gatekeeper",
  "task_id": "{task_id}",
  "status": "completed" | "failed" | "blocked",
  "summary": "{what was audited}",
  "dependencies_scanned": 0,
  "security": {
    "critical": 0,
    "high": 0,
    "medium": 0,
    "low": 0,
    "findings": [
      {"package": "", "version": "", "severity": "", "advisory": "", "recommendation": ""}
    ]
  },
  "licenses": {
    "permissive": [],
    "weak_copyleft": [],
    "strong_copyleft": [],
    "unknown": [],
    "compliance_risk": "none" | "low" | "medium" | "high"
  },
  "outdated_packages": [],
  "recommendations": [
    {"priority": 1, "action": "Run npm audit and apply patches for CRITICAL/HIGH findings", "builder_command": "npm audit fix"}
  ]
}

// Includer return format
{
  "ant_name": "{your name}",
  "caste": "includer",
  "task_id": "{task_id}",
  "status": "completed" | "failed" | "blocked",
  "summary": "{what was audited}",
  "wcag_level_targeted": "AA",
  "analysis_method": "manual static analysis",
  "files_analyzed": [],
  "violations": [
    {
      "wcag_criterion": "1.4.3",
      "criterion_name": "Contrast (Minimum)",
      "location": "{file}:{line}",
      "issue": "{what is wrong}",
      "fix": "{what Builder should do}"
    }
  ],
  "compliance_percent": 0,
  "dimensions_checked": ["visual", "motor", "cognitive", "hearing"],
  "recommendations": []
}
```

### Ambassador Credentials Iron Law (Critical Rules Section)

```markdown
<critical_rules>
## Non-Negotiable Rules

### Credentials Iron Law
Never write API keys, tokens, secrets, or credentials to any tracked file. This applies to:
- Source code files (.js, .ts, .py, etc.)
- Configuration files (config.json, settings.yaml, etc.)
- Any file that could be committed to git

When an SDK or API requires credentials:
1. Document the environment variable name needed (e.g., `STRIPE_SECRET_KEY`)
2. Implement using `process.env.STRIPE_SECRET_KEY` in code
3. Instruct the user to set it in their environment
4. Never hardcode, never echo, never log secrets

**Verification step (mandatory before returning complete):**
```bash
grep -r "KEY\|SECRET\|TOKEN\|PASSWORD" {integration_files} --include="*.js" --include="*.ts"
```
The result must show only `process.env.*` references, never literal values.

If asked to "just hardcode it temporarily" — refuse. There is no temporary in git history.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| OpenCode agents with activity-log | Claude Code agents with structured JSON returns | Phase 27 | No side effects; callers get machine-readable data |
| Thin niche agent definitions (~120 lines) | Fully specified 8-section XML bodies (~180-260 lines) | Phase 30 | Body quality test enforces substance; agents are genuinely useful, not stubs |
| TEST-05 failing (14 agents, 22 expected) | TEST-05 passing (22 agents, 22 expected) | Phase 30 | Colony roster complete; test suite green |
| READ_ONLY_CONSTRAINTS covering 2 agents | READ_ONLY_CONSTRAINTS covering 8 read-only agents | Phase 30 (verification plan) | TEST-03 enforces tool restrictions on all read-only agents, not just Phase 29 agents |
| Gatekeeper implying `npm audit` access | Gatekeeper static analysis via manifest inspection | Phase 30 | Honest capability declaration; no phantom tool use |
| Includer implying automated scanner | Includer manual static WCAG analysis | Phase 30 | No fabricated compliance scores; findings are verifiable |

**Deprecated patterns from OpenCode sources (all 8 agents):**
- All `bash .aether/aether-utils.sh activity-log ...` invocations — TEST-04 catches these
- All `bash .aether/aether-utils.sh spawn-can-spawn ...` invocations
- Emoji in role descriptions (🔌 Ambassador, ♿ Includer, ⚡ Measurer, 📝 Chronicler, 📦 Gatekeeper, 📜 Sage) — strip these; Claude Code agents use plain text role names

---

## Plan Grouping Recommendation (Claude's Discretion)

**Recommendation: 2 agent plans + 1 verification plan**

**Wave 1 — Parallel:**

Plan A: "Read-only investigators" (6 agents)
- Chaos, Archaeologist, Gatekeeper, Includer, Measurer, Sage
- Why grouped: all have no Write/Edit; all are analysis-and-report agents; similar design pattern
- One plan handles all 6; files are independent

Plan B: "Writers" (2 agents)
- Ambassador, Chronicler
- Why grouped: both have Write capability; both create files as primary output
- Simpler grouping because these 2 have the most distinct behavior from each other and from the 6 read-only agents

**Wave 2 — Sequential (after wave 1):**

Plan C: "Verification"
- Expands READ_ONLY_CONSTRAINTS in tests/unit/agent-quality.test.js
- Runs `npm test` to confirm all 22 agents pass all 6 test functions
- Confirms TEST-05 passes (22 found, 22 expected)

**Alternative considered:** 3 plans in wave 1 (read-only investigators, read-only with Bash, writers). Rejected because the Bash distinction (Chaos, Archaeologist, Measurer, Sage have Bash; Gatekeeper and Includer don't) doesn't change the design pattern enough to warrant a separate plan.

---

## Open Questions

1. **Chronicler with Edit — confirm write scope in boundaries**
   - Decision made: include Edit
   - What needs documenting in the agent: boundaries section must explicitly state Edit is permitted ONLY for documentation comments (JSDoc/TSDoc), not for modifying logic
   - Risk: without this explicit boundary, Edit could be used to modify source code beyond documentation

2. **Gatekeeper import graph analysis (Claude's discretion area)**
   - Context: CONTEXT.md says "dependencies only vs dependencies + import graphs"
   - Recommendation: include import graph analysis via Grep/Glob (search `require()`, `import` statements)
   - This is achievable without Bash and adds significant value — understanding what imports what helps identify why a dependency is present and what would break if it were removed
   - The execution_flow should have an "import graph" step that uses Grep to trace module dependencies

3. **Includer scope depth (Claude's discretion area)**
   - Context: CONTEXT.md says "assess based on typical project needs"
   - Recommendation: focus on what static code analysis can definitively find
   - Definitive via static analysis: missing alt text, missing form labels, missing ARIA roles, non-semantic HTML (divs where buttons/headers should be), hardcoded color values that likely fail contrast
   - Exclude from scope: color contrast ratio calculation (requires rendered output), keyboard navigation flow (requires interactive testing), screen reader compatibility (requires runtime)
   - State scope clearly in boundaries: "Includer performs static source code analysis only. Dynamic accessibility concerns (contrast ratios from computed styles, keyboard navigation in complex SPAs, screen reader compatibility) require runtime testing with automated tools — note these as gaps, not findings."

4. **Measurer cross-project scope (Claude's discretion area)**
   - Context: CONTEXT.md says "scope for general profiling across project types, not just Aether-specific"
   - Recommendation: execution_flow should have a "detect project type" step (Node.js, Python, Go, etc.) and use appropriate benchmarking patterns for each
   - Node.js: `node --prof`, performance.now(), jest --runInBand with timing
   - Python: cProfile, timeit, memory_profiler
   - General: Big-O analysis from source reading, database query pattern analysis
   - Static analysis fallback: when no profiling tool is available, analyze algorithmic complexity from code reading and document the tooling gap

---

## Sources

### Primary (HIGH confidence)

- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-chaos.md` — Content foundation for NICHE-01; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-archaeologist.md` — Content foundation for NICHE-02; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-ambassador.md` — Content foundation for NICHE-03; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-chronicler.md` — Content foundation for NICHE-04; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-gatekeeper.md` — Content foundation for NICHE-05; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-includer.md` — Content foundation for NICHE-07; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-measurer.md` — Content foundation for NICHE-06; read directly
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-sage.md` — Content foundation for NICHE-08; read directly
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-tracker.md` — Gold standard for read-only with Bash pattern
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-auditor.md` — Gold standard for most-restrictive read-only pattern
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-builder.md` — Gold standard for full-access agent pattern
- `/Users/callumcowie/repos/Aether/tests/unit/agent-quality.test.js` — Current test suite; READ_ONLY_CONSTRAINTS registry requires expansion; EXPECTED_AGENT_COUNT already 22
- `/Users/callumcowie/repos/Aether/.planning/phases/29-specialist-agents-agent-tests/29-RESEARCH.md` — Established patterns for Claude Code agent format
- `/Users/callumcowie/repos/Aether/.planning/phases/29-specialist-agents-agent-tests/29-VERIFICATION.md` — Confirmed Phase 29 state: 14 agents, TEST-05 failing intentionally

### Secondary (MEDIUM confidence)

- `/Users/callumcowie/repos/Aether/.planning/phases/29-specialist-agents-agent-tests/29-03-PLAN.md` — Test suite implementation details; READ_ONLY_CONSTRAINTS pattern for extension

---

## Metadata

**Confidence breakdown:**
- Agent format and frontmatter: HIGH — identical to Phase 28-29, all source files read directly; test suite validates automatically
- Tool assignments: HIGH — all 8 assignments are in REQUIREMENTS.md; discretion items (Chronicler Edit, Sage Write) resolved with rationale
- Test suite expansion: HIGH — exact code change identified (READ_ONLY_CONSTRAINTS registry); EXPECTED_AGENT_COUNT already correct
- Plan grouping: HIGH — 2 plans + 1 verification is the simplest correct structure; rationale documented
- Gatekeeper without Bash: HIGH — static manifest analysis approach verified by reviewing what Read/Grep/Glob can achieve
- Includer without Bash: HIGH — static WCAG analysis scope documented with explicit capability boundaries
- OpenCode source content direction: HIGH — all 8 source files read directly
- Agent body depth estimates (180-260 lines target): MEDIUM — based on Phase 29 agent sizes (265-271 lines for tracker/auditor); actual depth depends on execution flow richness

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (agent format stable; test suite stable; Claude Code subagent format well-established)
