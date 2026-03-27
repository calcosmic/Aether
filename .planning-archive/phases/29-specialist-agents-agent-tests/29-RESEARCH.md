# Phase 29: Specialist Agents + Agent Tests - Research

**Researched:** 2026-02-20
**Domain:** Claude Code subagent authoring (specialist agents) + AVA test suite for agent file quality enforcement
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Agent depth & personality:**
- Full 8-section XML body for ALL 5 specialists — same depth as Phase 28 agents (Queen, Scout, Surveyors)
- Port structure from OpenCode source, rewrite content for Claude Code context — not a copy-paste, a reimagining
- Light colony flavor — functional first, colony references where natural (e.g., "escalate to the Queen" not "escalate to the orchestrator"). Matches Phase 28 tone.

**Test strictness & scope:**
- Tests validate ALL agent files that exist (not just the 5 new ones) — dynamic count that grows as phases ship
- TEST-05 (count=22) starts as a failing target until Phase 30 completes — that's intentional, not a bug
- Add body quality checks beyond the 5 requirements: verify XML sections present, no empty sections, minimum content length to catch lazy ports
- Claude's Discretion: test file organization (one file vs multiple), exact tool validation approach (forbidden-only vs exact match vs hybrid)

**Read-only boundaries:**
- Tracker: diagnose + suggest — returns root cause analysis AND a suggested fix, but doesn't apply it. Builder makes the change.
- Auditor: structured findings — returns file, line, severity, category, description, suggestion. No narrative review.
- Probe: Claude decides whether it writes + runs tests or writes only, based on the existing test workflow
- Claude's Discretion: Probe's run/write scope

**Agent specialization:**
- Keeper is ONE unified agent — "maintain project knowledge" encompasses architecture understanding AND wisdom management. No mode split.
- Weaver runs tests before + after refactoring. If tests break, it reverts. Behavior preservation is enforced, not just documented.
- Cross-reference escalation: specialists reference each other (Tracker → Builder for fixes, Auditor → Queen for security issues). Colony feels connected.
- Claude's Discretion: description style (generic vs colony-aware) — pick what routes best in Claude Code

### Claude's Discretion

- Test file organization: one file vs multiple
- Exact tool validation approach: forbidden-only vs exact match vs hybrid
- Probe's run/write scope
- Description style (generic vs colony-aware)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| SPEC-01 | Keeper agent upgraded — architecture mode + wisdom management, structured knowledge returns. Tools: Read, Write, Edit, Bash, Grep, Glob | Full tool set gives Keeper write access for knowledge base files; same tool set as Builder. OpenCode source: `.opencode/agents/aether-keeper.md` |
| SPEC-02 | Tracker agent upgraded — systematic bug investigation with scientific method, root cause analysis. Tools: Read, Bash, Grep, Glob (no Write/Edit) | Read-only for source files; Bash for running reproduction commands; no Write/Edit enforces diagnose-only boundary. OpenCode source: `.opencode/agents/aether-tracker.md` |
| SPEC-03 | Probe agent upgraded — test generation, coverage analysis, edge case discovery. Tools: Read, Write, Edit, Bash, Grep, Glob | Full tool set gives Probe ability to write test files; Write scoped to test directories in boundaries section. OpenCode source: `.opencode/agents/aether-probe.md` |
| SPEC-04 | Weaver agent upgraded — code refactoring with behavior preservation guarantees. Tools: Read, Write, Edit, Bash, Grep, Glob | Full tool set; Bash needed to run tests before/after refactor. Revert behavior enforced by failure_modes. OpenCode source: `.opencode/agents/aether-weaver.md` |
| SPEC-05 | Auditor agent upgraded — code review + security lens mode, structured finding returns. Tools: Read, Grep, Glob (no Write/Edit/Bash) | Most restrictive specialist — read-only, no shell access. Belt-and-suspenders: tools field enforces this AND tests validate it. OpenCode source: `.opencode/agents/aether-auditor.md` |
| TEST-01 | AVA test validates all agent files have required YAML frontmatter (name, description, tools) | js-yaml v4.1.1 (already in dependencies) parses YAML frontmatter after stripping `---` delimiters. AVA 6.4.1 (already installed) runs the test file. No new dependencies. |
| TEST-02 | AVA test validates agent names match `aether-{role}` pattern (lowercase, hyphens only) | Regex: `/^aether-[a-z][a-z0-9-]+$/`. Applied to the `name` frontmatter field AND to the filename (minus `.md` extension). Both must match. |
| TEST-03 | AVA test validates read-only agents have no Write/Edit in tools field | Read-only set: Tracker (no Write/Edit), Auditor (no Write/Edit/Bash). Validated by checking tools array parsed from frontmatter against forbidden tool list per agent. |
| TEST-04 | AVA test validates no agent body contains spawn calls or activity-log requirements | Regex patterns against raw file content (below the closing `---` of frontmatter). Patterns: `activity-log`, `spawn-can-spawn`, `generate-ant-name`, `spawn-log`, `spawn-complete`. |
| TEST-05 | AVA test validates agent count matches expected (22 agents in ant/ directory) | Current count: 9. Phase 29 adds 5 → 14. TEST-05 hardcodes 22. This test FAILS until Phase 30 delivers the remaining 8. Intentional — it's a tracking mechanism, not a defect. |
</phase_requirements>

---

## Summary

Phase 29 has two parallel workstreams: (1) creating 5 specialist Claude Code agents and (2) building an AVA test suite that enforces quality standards across all agent files. Both workstreams are self-contained — the agent work follows the exact format established in Phase 27–28, and the test work uses dependencies already in the project (AVA 6.4.1, js-yaml 4.1.1).

For the agents: the OpenCode source files (`.opencode/agents/aether-{keeper,tracker,probe,weaver,auditor}.md`) exist and contain the right content. The job is to reimagine them in the Claude Code format — stripping OpenCode-specific patterns (activity-log, spawn calls), rewriting for the 8-section XML structure, adding explicit tools fields in frontmatter, and ensuring descriptions route correctly. Each specialist has a different tool restriction that must be enforced both in the frontmatter and verified by the test suite. The hardest design work is Tracker's "diagnose + suggest, no apply" boundary and Auditor's "read-only, no Bash" boundary.

For the tests: js-yaml is already a project dependency (used in `bin/cli.js`), so YAML frontmatter parsing is available without adding a new package. The test suite reads all `.md` files from `.claude/agents/ant/` dynamically, parses their frontmatter, and checks five categories of quality gates. The critical implementation decision is whether to use a single test file with multiple test cases or separate files per category. TEST-05 is intentionally failing at phase completion — it tracks 22 agents but only 14 will exist after Phase 29. This is documented as expected behavior.

**Primary recommendation:** One test file (`tests/unit/agent-quality.test.js`) covering all 5 test requirements, with a helper function that parses YAML frontmatter from markdown agent files. Agent files: write each as a standalone file in `.claude/agents/ant/`, following the Phase 28 template exactly.

---

## Standard Stack

### Core

| Component | Version/Path | Purpose | Why Standard |
|-----------|-------------|---------|--------------|
| AVA test runner | 6.4.1 (installed) | Runs agent quality tests | Already in project; `npm test` invokes `ava` targeting `tests/unit/**/*.test.js` |
| js-yaml | 4.1.1 (installed) | Parses YAML frontmatter from agent files | Already in project (`dependencies`, not just `devDependencies`); same library used in `bin/cli.js` |
| Node.js `fs` module | Built-in | Reads agent files from disk | No mocking needed — these tests read real files, not mock data |
| Node.js `path` module | Built-in | Constructs absolute paths to agent files | Required to locate `.claude/agents/ant/` relative to project root |
| Phase 28 agent format | `.claude/agents/ant/*.md` | Structural template for all 5 specialist agents | Established in Phase 27, validated in Phase 28 — identical 8-section XML body |

### Supporting

| Component | Path | Purpose | When to Use |
|-----------|------|---------|-------------|
| OpenCode source: Keeper | `.opencode/agents/aether-keeper.md` | Content reference for SPEC-01 | Read first; strip activity-log, rewrite for 8 XML sections, add tools field |
| OpenCode source: Tracker | `.opencode/agents/aether-tracker.md` | Content reference for SPEC-02 | Read first; already has good debugging content; remove Write/Edit from tools, strip activity-log |
| OpenCode source: Probe | `.opencode/agents/aether-probe.md` | Content reference for SPEC-03 | Read first; test generation strategies carry over; strip activity-log |
| OpenCode source: Weaver | `.opencode/agents/aether-weaver.md` | Content reference for SPEC-04 | Read first; refactoring techniques carry over; add test-before/after enforcement to failure_modes |
| OpenCode source: Auditor | `.opencode/agents/aether-auditor.md` | Content reference for SPEC-05 | Read first; audit dimensions and severity ratings carry over; enforce total read-only (no Bash) |
| Phase 28 builder (template) | `.claude/agents/ant/aether-builder.md` | Gold standard for 8-section XML format | Reference for section names, style, return_format JSON, escalation chain structure |

### No New Dependencies

Zero new libraries required. Both workstreams use already-installed packages. Do not add any `npm install` step.

---

## Architecture Patterns

### Pattern 1: Frontmatter for Read-Only Enforcement

**What:** The `tools:` field in YAML frontmatter is the primary read-only enforcement mechanism. The test suite is the secondary (belt-and-suspenders) check.

**Tool assignments per spec:**

```yaml
# Keeper (SPEC-01) — full write access for knowledge base
tools: Read, Write, Edit, Bash, Grep, Glob

# Tracker (SPEC-02) — no Write/Edit; diagnose only
tools: Read, Bash, Grep, Glob

# Probe (SPEC-03) — full write access for test files
tools: Read, Write, Edit, Bash, Grep, Glob

# Weaver (SPEC-04) — full write access for source files + Bash to run tests
tools: Read, Write, Edit, Bash, Grep, Glob

# Auditor (SPEC-05) — most restrictive; no Write/Edit/Bash
tools: Read, Grep, Glob
```

**Why it matters:** Claude Code restricts what tools an agent can invoke based on the frontmatter `tools:` list. An Auditor without Write/Edit/Bash in its tools field cannot modify files even if its body instructs it to. This is platform-level enforcement — the body text is a secondary layer.

### Pattern 2: YAML Frontmatter Parsing in Tests

**What:** Agent files have this structure:
```
---
name: aether-keeper
description: "..."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

<role>
... body ...
```

**Parsing approach:**
1. Read file content as string with `fs.readFileSync(filePath, 'utf8')`
2. Split on `---` delimiters to extract frontmatter block
3. Parse frontmatter block with `yaml.load(frontmatterText)` (js-yaml)
4. Body = everything after the second `---`

**Code pattern (verified against js-yaml 4.1.1 API):**
```javascript
const yaml = require('js-yaml');
const fs = require('fs');
const path = require('path');

function parseAgentFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  // Agent files start with ---\n and have a closing ---\n
  const parts = content.split(/^---\s*$/m);
  // parts[0] = '' (before first ---)
  // parts[1] = frontmatter text
  // parts[2] = body text
  if (parts.length < 3) return null;
  const frontmatter = yaml.load(parts[1]);
  const body = parts[2];
  return { frontmatter, body, raw: content };
}
```

**Warning:** `yaml.load()` parses the frontmatter as a plain object. The `tools` field will be a string like `"Read, Write, Edit, Bash, Grep, Glob"` — NOT an array. Parse it with `.split(',').map(t => t.trim())` to get an array for set-based comparisons.

### Pattern 3: Dynamic Agent File Discovery

**What:** Tests discover agent files dynamically using `fs.readdirSync` rather than a hardcoded list. This means the test suite automatically covers newly added agents without code changes.

```javascript
const AGENTS_DIR = path.join(__dirname, '../../.claude/agents/ant');

function getAgentFiles() {
  return fs.readdirSync(AGENTS_DIR)
    .filter(f => f.endsWith('.md'))
    .map(f => path.join(AGENTS_DIR, f));
}
```

**Why dynamic:** TEST-05 requires a count check (22 agents). All other tests run for every file found. If a new agent file is malformed, the tests catch it automatically. This matches the requirement: "Tests validate ALL agent files that exist."

### Pattern 4: TEST-05 Intentional Failure Pattern

**What:** TEST-05 hardcodes the expected count as 22. After Phase 29, the actual count will be 14. The test fails. This is intentional — it tracks a future completion milestone.

**Implementation:**
```javascript
test('TEST-05: agent count matches expected 22', t => {
  const files = getAgentFiles();
  // NOTE: This test fails until Phase 30 completes (adds remaining 8 agents).
  // Expected: 22 (full colony). Current after Phase 29: 14. This is intentional.
  t.is(files.length, 22);
});
```

**Important:** Add a comment in the test explaining this is a known intentional failure. The PLAN.md should document that `npm test` will show one failing test after Phase 29 is complete — that is the expected state.

### Pattern 5: Body Quality Checks (Beyond the 5 Requirements)

**What:** The user decision adds body quality checks to catch lazy ports:
- All 8 XML sections must be present
- No section should be empty
- Minimum content length (prevents placeholder stubs)

**The 8 required XML sections (established in Phase 27):**
```
<role>, <execution_flow>, <critical_rules>, <return_format>,
<success_criteria>, <failure_modes>, <escalation>, <boundaries>
```

**Check approach:**
```javascript
const REQUIRED_SECTIONS = [
  'role', 'execution_flow', 'critical_rules', 'return_format',
  'success_criteria', 'failure_modes', 'escalation', 'boundaries'
];

function checkBodyQuality(body) {
  const errors = [];
  for (const section of REQUIRED_SECTIONS) {
    const openTag = `<${section}>`;
    const closeTag = `</${section}>`;
    if (!body.includes(openTag)) errors.push(`Missing section: <${section}>`);
    if (!body.includes(closeTag)) errors.push(`Unclosed section: </${section}>`);
    // Check for non-empty sections
    const start = body.indexOf(openTag) + openTag.length;
    const end = body.indexOf(closeTag);
    if (end > start) {
      const sectionContent = body.slice(start, end).trim();
      if (sectionContent.length < 50) {
        errors.push(`Section <${section}> appears too short (${sectionContent.length} chars) — possible placeholder`);
      }
    }
  }
  return errors;
}
```

**Minimum content length:** 50 characters per section is a reasonable floor — even a one-sentence description exceeds this, while an empty `<role></role>` or `<role>TODO</role>` would fail.

### Pattern 6: TEST-04 Forbidden Pattern Detection

**What:** TEST-04 checks that agent bodies contain no OpenCode-only patterns.

**Forbidden patterns:**
```javascript
const FORBIDDEN_PATTERNS = [
  /activity-log/,           // bash .aether/aether-utils.sh activity-log
  /spawn-can-spawn/,        // bash .aether/aether-utils.sh spawn-can-spawn
  /generate-ant-name/,      // bash .aether/aether-utils.sh generate-ant-name
  /spawn-log/,              // bash .aether/aether-utils.sh spawn-log
  /spawn-complete/,         // bash .aether/aether-utils.sh spawn-complete
];
```

**Note:** Do NOT add `spawn` as a bare pattern — it appears legitimately in escalation text like "Do NOT attempt to spawn sub-workers." Only the specific compound patterns above are OpenCode artifacts. Verified against Phase 27 RESEARCH.md (PWR-08 removal checklist).

### Pattern 7: TEST-03 Read-Only Validation

**What:** TEST-03 validates that agents designated as read-only have the correct tool restrictions.

**Read-only registry approach (recommended):**
```javascript
// Forbidden tools per read-only agent type
const READ_ONLY_CONSTRAINTS = {
  'aether-tracker': { forbidden: ['Write', 'Edit'] },
  'aether-auditor': { forbidden: ['Write', 'Edit', 'Bash'] },
};

// In the test:
const agentName = parsed.frontmatter.name;
if (READ_ONLY_CONSTRAINTS[agentName]) {
  const tools = parsed.frontmatter.tools.split(',').map(t => t.trim());
  for (const forbidden of READ_ONLY_CONSTRAINTS[agentName].forbidden) {
    t.false(tools.includes(forbidden),
      `${agentName}: must not have ${forbidden} in tools (read-only agent)`);
  }
}
```

**Why forbidden-only (not exact match):** An exact match would break if new valid tools are added in future phases. The forbidden-only approach checks the constraint that matters (read-only enforcement) without over-specifying. This is the "Claude's Discretion" area — forbidden-only wins on flexibility.

### Pattern 8: Agent Description Style — Colony-Aware Routing Triggers

**What:** This is a "Claude's Discretion" area. The research finding from Phase 27 (PWR-06) says descriptions should be "routing triggers, not role labels." For specialist agents, colony-aware descriptions that reference the workers they collaborate with will route better.

**Recommendation:** Use colony-aware descriptions that name the contexts where each specialist is invoked:
- Keeper: "Use this agent to maintain project knowledge, extract patterns, and manage architectural wisdom. Invoked by Queen during Documentation Sprint and Deep Research patterns."
- Tracker: "Use this agent to investigate bugs systematically and identify root causes. Returns root cause analysis and a suggested fix — Builder applies the fix. Do NOT use for implementation (use aether-builder)."
- Probe: "Use this agent to generate tests, analyze coverage gaps, and discover edge cases. Invoked by Queen and Builder when test coverage is insufficient or a feature needs test-first development."
- Weaver: "Use this agent to refactor code without changing behavior. Runs tests before and after refactoring — if tests break, Weaver reverts immediately."
- Auditor: "Use this agent for code review, security audits, and compliance checks. Strictly read-only — returns structured findings only. For security escalations, routes to Queen."

**Key differentiators to include in descriptions:**
- Tracker: "Returns analysis and suggested fix — Builder applies" (reinforces the diagnose-not-apply boundary)
- Auditor: "strictly read-only" (reinforces no-write boundary)
- Weaver: "reverts if tests break" (behavior preservation guarantee)

### Anti-Patterns to Avoid

- **Keeping activity-log calls:** Every OpenCode agent has `bash .aether/aether-utils.sh activity-log` lines. These must be removed from all 5 specialists. TEST-04 will catch if they're left in.
- **Giving Tracker Write/Edit:** Per SPEC-02, Tracker diagnoses only. The locked decision says "Tracker → Builder for fixes." If Tracker has Write/Edit, the boundary breaks. TEST-03 will catch this.
- **Giving Auditor Bash:** Auditor is the most restricted specialist (no Write/Edit/Bash). Even Read-only agents sometimes have Bash for running linters — Auditor explicitly does not. TEST-03 will catch any Bash in Auditor's tools.
- **Static agent list in tests:** Hardcoding `['aether-keeper.md', 'aether-tracker.md', ...]` in the test means Phase 30 agents won't be tested automatically. Use `fs.readdirSync` for dynamic discovery.
- **Missing `model: inherit` in frontmatter:** All Phase 28 agents include `model: inherit`. This should carry forward to the 5 Phase 29 agents for consistency.
- **Silently ignoring count mismatch (TEST-05):** Some implementations may be tempted to skip TEST-05 or make it a soft warning. It must be a hard-failing test case — that's how it tracks the Phase 30 milestone.
- **Empty XML sections:** The Keeper OpenCode source is notably thin (113 lines total). When reimagining for Claude Code, each section needs real content, not placeholders. The body quality check (50 char minimum per section) will catch this.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing | Custom `---`-delimiter string parsing | `js-yaml` (`yaml.load()`) | Already installed; handles edge cases (escaped colons, multiline strings, type coercion) |
| File discovery | Manual hardcoded list of agent files | `fs.readdirSync(AGENTS_DIR).filter(f => f.endsWith('.md'))` | Dynamic — covers Phase 30 agents automatically |
| Read-only enforcement at runtime | Logic in agent body text alone | `tools:` frontmatter field (platform enforces) + tests (belt-and-suspenders) | Platform-level tool restriction is authoritative; body text guides behavior, not enforces it |
| XML section parsing | Full XML parser | Simple `body.includes('<role>')` checks | Agent files use pseudo-XML tag markers, not valid XML. A full parser would overkill and fail on the non-valid XML. String matching is sufficient. |

**Key insight:** Every needed tool is already installed. The only new file is the test file itself.

---

## Common Pitfalls

### Pitfall 1: js-yaml Parses `tools` Field as a String, Not an Array

**What goes wrong:** Test code calls `.includes('Write')` on `frontmatter.tools` (a string) and gets false positives. `"Read, Write, Glob".includes('Write')` is true, but `"Glob, Bash".includes('ash')` is also (incorrectly) true.
**Why it happens:** YAML's `tools: Read, Write, Edit` is a string value, not a YAML list. A YAML list would be `tools:\n  - Read\n  - Write`. The frontmatter uses comma-separated strings per Claude Code convention.
**How to avoid:** Always parse with `.split(',').map(t => t.trim())` before comparison. Use `.includes('Write')` on the resulting array, not on the raw string.
**Warning signs:** Tests pass for exact tool names but would also match substrings of tool names.

### Pitfall 2: Frontmatter Parsing Fails Silently When `---` Delimiter Is Inconsistent

**What goes wrong:** `content.split(/^---\s*$/m)` yields fewer than 3 parts — test code throws on `parts[1]` being undefined.
**Why it happens:** Some files may have trailing spaces after `---`, or the file doesn't start with `---` on line 1.
**How to avoid:** Check `parts.length >= 3` before proceeding. If a file fails to parse, report it as a test failure for that file rather than crashing the entire test run. Add this as an explicit test case in TEST-01.
**Warning signs:** One malformed file causes the entire test suite to crash instead of reporting the individual failure.

### Pitfall 3: TEST-04 Over-Matching "spawn" as a Forbidden Pattern

**What goes wrong:** Adding bare `/spawn/` as a forbidden pattern causes false failures because agents legitimately use "spawn" in phrases like "Do NOT attempt to spawn sub-workers."
**Why it happens:** "spawn" is both a legitimate English word in the context of escalation rules AND an OpenCode bash command prefix.
**How to avoid:** Only match the compound patterns: `activity-log`, `spawn-can-spawn`, `generate-ant-name`, `spawn-log`, `spawn-complete`. These are unambiguously OpenCode artifacts. Verified against Phase 27 PWR-08 removal checklist.

### Pitfall 4: Keeper Agent Too Thin — The OpenCode Source Is Minimal

**What goes wrong:** Keeper's OpenCode source is only ~113 lines, much of it boilerplate. A direct port would produce an agent that fails the body quality check (50-char minimum per section).
**Why it happens:** The OpenCode Keeper was a thin wrapper with architecture mode layered on. The Claude Code version needs to reimagine the content, not just strip OpenCode patterns.
**How to avoid:** Keeper requires the most original writing of the 5 specialists. The sections that need substantial content: `<execution_flow>` (synthesis workflow, knowledge organization), `<critical_rules>` (never overwrite curated patterns), `<return_format>` (structured JSON with patterns_archived, patterns_updated). Use Builder as the depth reference.
**Warning signs:** If the Keeper agent body is under 100 lines total, it's probably under-specified.

### Pitfall 5: Weaver Revert Behavior Not Enforced in failure_modes

**What goes wrong:** Weaver's body describes "behavior preservation" but doesn't specify what happens when tests break post-refactor.
**Why it happens:** The locked decision says "If tests break, it reverts. Behavior preservation is enforced, not just documented." The OpenCode Weaver mentions "Prefer small, incremental changes" but is weak on the revert protocol.
**How to avoid:** In Weaver's `<failure_modes>`, make revert behavior explicit and automatic: "If tests fail post-refactor, STOP immediately. Revert to pre-refactor state (`git stash pop` or revert individual files). Do not attempt to fix the new test failures — that is no longer refactoring, it is bug introduction." Include this as a Major failure trigger.
**Warning signs:** Weaver's failure_modes section doesn't mention "revert" or "git stash" or equivalent rollback language.

### Pitfall 6: Auditor Returning Narrative Rather Than Structured Findings

**What goes wrong:** Auditor produces prose analysis instead of the structured finding format (file, line, severity, category, description, suggestion).
**Why it happens:** The locked decision says "Auditor: structured findings — returns file, line, severity, category, description, suggestion. No narrative review." This is a departure from the OpenCode Auditor which has a less strict format.
**How to avoid:** In Auditor's `<return_format>`, define the exact JSON structure required. Each finding in the `issues` array must have: `file`, `line`, `severity`, `category`, `description`, `suggestion`. Make this a `<critical_rules>` item: "Every finding must cite a specific file and line number. Unsupported findings are not findings — they are guesses."

### Pitfall 7: Test File NOT Covered by ava.files Pattern

**What goes wrong:** New test file is not picked up by `npm test`.
**Why it happens:** `package.json` has `"ava": { "files": ["tests/unit/**/*.test.js"] }`. A test file outside `tests/unit/` or not matching `*.test.js` won't run.
**How to avoid:** Place the test file at `tests/unit/agent-quality.test.js`. This matches the existing pattern exactly.

---

## Code Examples

Verified patterns from official sources and codebase analysis:

### YAML Frontmatter Parser for Agent Files

```javascript
// Source: js-yaml 4.1.1 API (Context7 verified) + project test patterns
const yaml = require('js-yaml');
const fs = require('fs');
const path = require('path');

const AGENTS_DIR = path.join(__dirname, '../../.claude/agents/ant');

function getAgentFiles() {
  return fs.readdirSync(AGENTS_DIR)
    .filter(f => f.endsWith('.md'))
    .sort()
    .map(f => path.join(AGENTS_DIR, f));
}

function parseAgentFile(filePath) {
  const content = fs.readFileSync(filePath, 'utf8');
  const parts = content.split(/^---\s*$/m);
  if (parts.length < 3) return null;
  try {
    const frontmatter = yaml.load(parts[1]);
    const body = parts.slice(2).join('---'); // body is everything after second ---
    return { frontmatter, body, filename: path.basename(filePath) };
  } catch (e) {
    return null; // Malformed YAML — caught as test failure
  }
}

function parseTools(toolsString) {
  if (!toolsString) return [];
  return toolsString.split(',').map(t => t.trim()).filter(Boolean);
}
```

### TEST-01: Frontmatter Completeness Check

```javascript
// Source: existing test patterns in tests/unit/cli-manifest.test.js
const test = require('ava');

test('TEST-01: all agent files have required YAML frontmatter fields', t => {
  const files = getAgentFiles();
  t.true(files.length > 0, 'No agent files found in .claude/agents/ant/');

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    const filename = path.basename(filePath);

    t.truthy(parsed, `${filename}: failed to parse frontmatter`);
    if (!parsed) continue;

    t.truthy(parsed.frontmatter.name, `${filename}: missing required field: name`);
    t.truthy(parsed.frontmatter.description, `${filename}: missing required field: description`);
    t.truthy(parsed.frontmatter.tools, `${filename}: missing required field: tools`);
  }
});
```

### TEST-02: Name Pattern Check

```javascript
test('TEST-02: agent names match aether-{role} pattern', t => {
  const NAME_PATTERN = /^aether-[a-z][a-z0-9-]+$/;
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    const filename = path.basename(filePath, '.md');
    if (!parsed) continue;

    // Frontmatter name must match pattern
    t.regex(parsed.frontmatter.name, NAME_PATTERN,
      `${filename}: name "${parsed.frontmatter.name}" does not match aether-{role} pattern`);

    // Filename must match name in frontmatter
    t.is(parsed.frontmatter.name, filename,
      `${filename}: frontmatter name "${parsed.frontmatter.name}" does not match filename`);
  }
});
```

### TEST-03: Read-Only Enforcement Check

```javascript
const READ_ONLY_CONSTRAINTS = {
  'aether-tracker': { forbidden: ['Write', 'Edit'] },
  'aether-auditor': { forbidden: ['Write', 'Edit', 'Bash'] },
};

test('TEST-03: read-only agents have no write tools in frontmatter', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    const agentName = parsed.frontmatter.name;
    const constraints = READ_ONLY_CONSTRAINTS[agentName];
    if (!constraints) continue; // Not a read-only agent

    const tools = parseTools(parsed.frontmatter.tools);
    for (const forbidden of constraints.forbidden) {
      t.false(tools.includes(forbidden),
        `${agentName}: must not have ${forbidden} in tools (read-only agent constraint)`);
    }
  }
});
```

### TEST-04: Forbidden Pattern Check

```javascript
const FORBIDDEN_PATTERNS = [
  { pattern: /activity-log/, name: 'activity-log (OpenCode pattern)' },
  { pattern: /spawn-can-spawn/, name: 'spawn-can-spawn (OpenCode pattern)' },
  { pattern: /generate-ant-name/, name: 'generate-ant-name (OpenCode pattern)' },
  { pattern: /\bspawn-log\b/, name: 'spawn-log (OpenCode pattern)' },
  { pattern: /spawn-complete/, name: 'spawn-complete (OpenCode pattern)' },
];

test('TEST-04: agent bodies contain no OpenCode-only spawn or activity-log calls', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    for (const { pattern, name } of FORBIDDEN_PATTERNS) {
      t.false(pattern.test(parsed.body),
        `${parsed.filename}: found forbidden pattern "${name}" in agent body`);
    }
  }
});
```

### TEST-05: Agent Count Check (Intentionally Failing Until Phase 30)

```javascript
const EXPECTED_AGENT_COUNT = 22; // Full colony after Phase 30

test('TEST-05: agent count matches expected (22 after Phase 30)', t => {
  const files = getAgentFiles();
  // NOTE: This test FAILS until Phase 30 completes.
  // After Phase 29: 14 agents (9 existing + 5 new specialists)
  // After Phase 30: 22 agents (14 + 8 remaining)
  // Expected failure is intentional — this tracks the Phase 30 milestone.
  t.is(files.length, EXPECTED_AGENT_COUNT,
    `Expected ${EXPECTED_AGENT_COUNT} agents, found ${files.length}. Remaining: ${EXPECTED_AGENT_COUNT - files.length} agents needed (Phase 30).`);
});
```

### Body Quality Check (Beyond the 5 Requirements)

```javascript
const REQUIRED_XML_SECTIONS = [
  'role', 'execution_flow', 'critical_rules', 'return_format',
  'success_criteria', 'failure_modes', 'escalation', 'boundaries'
];
const MIN_SECTION_CONTENT_LENGTH = 50;

function checkBodyQuality(body, filename) {
  const errors = [];
  for (const section of REQUIRED_XML_SECTIONS) {
    const openTag = `<${section}>`;
    const closeTag = `</${section}>`;
    if (!body.includes(openTag)) {
      errors.push(`Missing section: ${openTag}`);
      continue;
    }
    if (!body.includes(closeTag)) {
      errors.push(`Unclosed section: ${closeTag}`);
      continue;
    }
    const start = body.indexOf(openTag) + openTag.length;
    const end = body.indexOf(closeTag);
    if (end > start) {
      const content = body.slice(start, end).trim();
      if (content.length < MIN_SECTION_CONTENT_LENGTH) {
        errors.push(`Section ${openTag} too short (${content.length} chars < ${MIN_SECTION_CONTENT_LENGTH} required)`);
      }
    }
  }
  return errors;
}

test('agent bodies contain all required XML sections with adequate content', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    const errors = checkBodyQuality(parsed.body, parsed.filename);
    t.deepEqual(errors, [],
      `${parsed.filename}: body quality errors: ${errors.join('; ')}`);
  }
});
```

### Specialist Agent Frontmatter Templates

```yaml
# Keeper (SPEC-01)
---
name: aether-keeper
description: "Use this agent to maintain project knowledge, extract architectural patterns, and manage institutional wisdom. Invoked during Documentation Sprint and Deep Research patterns when the colony needs knowledge synthesis. Do NOT use for implementation (use aether-builder) or code review (use aether-auditor)."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

# Tracker (SPEC-02)
---
name: aether-tracker
description: "Use this agent to investigate bugs systematically and identify root causes. Returns root cause analysis AND a suggested fix — Builder applies the fix. Tracker does not modify files. Do NOT use for implementation (use aether-builder) or refactoring (use aether-weaver)."
tools: Read, Bash, Grep, Glob
model: inherit
---

# Probe (SPEC-03)
---
name: aether-probe
description: "Use this agent to generate tests, analyze coverage gaps, and discover edge cases. Probe writes test files only — never modifies source code. Invoked by Queen and Builder when coverage is insufficient or test-first development is needed."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

# Weaver (SPEC-04)
---
name: aether-weaver
description: "Use this agent to refactor code without changing behavior. Weaver runs tests before and after every refactoring step — if tests break, it reverts immediately. Do NOT use for new features (use aether-builder) or bug fixes (use aether-tracker + aether-builder)."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

# Auditor (SPEC-05)
---
name: aether-auditor
description: "Use this agent for code review, security audits, and compliance checks. Strictly read-only — returns structured findings (file, line, severity, category, description, suggestion). For security escalations, routes to Queen. Do NOT use for fixes (use aether-builder) or test additions (use aether-probe)."
tools: Read, Grep, Glob
model: inherit
---
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No agent tests | AVA test suite enforcing frontmatter, naming, read-only, and body quality | Phase 29 | Malformed agent files fail `npm test` — YAML errors no longer silently drop agents |
| OpenCode activity-log pattern | Structured JSON return format | Phase 27 (established) | Agents return data to callers; no side effects through bash logging |
| OpenCode agents in `.opencode/agents/` | Claude Code agents in `.claude/agents/ant/` | Phase 27 (pipeline), Phase 28 (content) | Distributed via hub to all repos |
| Tracker applies fixes | Tracker diagnoses + suggests only; Builder applies | Phase 29 (new boundary) | Cleaner separation of concerns; Tracker is provably safe (no write tools) |
| Auditor with optional Bash | Auditor: no Bash, no Write, no Edit | Phase 29 (tightened from OpenCode) | Most restrictive specialist; platform-level enforcement via tools field |
| Manual YAML verification | `/agents` command + test suite | Phase 27 + Phase 29 | Two-layer verification: manual (human runs `/agents`) + automated (test suite) |

**Deprecated in Phase 29 (from OpenCode sources being ported):**
- All `activity-log` bash calls in Keeper, Tracker, Probe, Weaver, Auditor
- All `spawn-*` bash calls in any of the 5 specialists
- Tracker's implied "and fix it" behavior (now diagnose-only)
- Auditor's Bash access (explicitly removed in Claude Code version)

---

## Open Questions

1. **Probe's run/write scope (Claude's Discretion)**
   - What we know: SPEC-03 gives Probe full tool access (Read, Write, Edit, Bash, Grep, Glob). The locked decision says "Claude decides whether it writes + runs tests or writes only, based on the existing test workflow."
   - Existing test workflow: `npm test` = AVA targeting `tests/unit/**/*.test.js`. Bash is available to Probe.
   - Recommendation: Probe SHOULD both write AND run tests. Rationale: Probe has Bash in its tools specifically so it can run the test suite and verify its generated tests actually pass. Writing tests without running them is incomplete work. In Probe's `<success_criteria>`, require: "Run all new tests — they must pass. Run existing tests — no regressions introduced." This matches the OpenCode Probe success criteria and is practical with Bash available.

2. **Test file organization (Claude's Discretion)**
   - Options: (A) One file `tests/unit/agent-quality.test.js`, (B) Multiple files `tests/unit/agent-frontmatter.test.js`, `tests/unit/agent-naming.test.js`, etc.
   - Recommendation: ONE file. The 5 test requirements plus body quality checks are coherent as a suite — they all test agent file quality. Splitting into multiple files adds navigation overhead without improving clarity. The existing test files are per-feature (cli-hash, cli-manifest, colony-state) and all tests for agents-as-a-feature belong together.
   - Counter-argument: Multiple files would allow `npm test -- --match "TEST-05"` style filtering. Weak enough not to override the single-file recommendation.

3. **Agent count after Phase 29 (context for TEST-05)**
   - Current: 9 agents in `.claude/agents/ant/`
   - After Phase 29 adds 5 specialists: 14 agents
   - Phase 30 must add 8 more to reach 22 (the remaining OpenCode agents not yet ported: ambassador, archaeologist, chaos, chronicler, gatekeeper, includer, measurer, sage)
   - TEST-05 will show 1 failing test in `npm test` output after Phase 29. This is the intended state.

---

## Sources

### Primary (HIGH confidence)

- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-keeper.md` — Content source for SPEC-01
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-tracker.md` — Content source for SPEC-02
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-probe.md` — Content source for SPEC-03
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-weaver.md` — Content source for SPEC-04
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-auditor.md` — Content source for SPEC-05
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-builder.md` — Gold standard template for 8-section XML format
- `/Users/callumcowie/repos/Aether/.claude/agents/ant/aether-surveyor-pathogens.md` — Phase 28 format reference (surveyor pattern)
- `/Users/callumcowie/repos/Aether/package.json` — AVA config (`ava.files` pattern), dependency versions (AVA 6.4.1, js-yaml 4.1.1)
- `/Users/callumcowie/repos/Aether/.planning/phases/28-orchestration-layer-surveyor-variants/28-RESEARCH.md` — Phase 28 patterns (PWR-08 forbidden patterns, YAML malformation risk, routing description principles)
- Context7 `/avajs/ava` — AVA 6.x API: test structure, beforeEach, serial tests, t.is/t.truthy/t.regex assertions
- Context7 `/nodeca/js-yaml` — js-yaml 4.x API: `yaml.load()` single document parsing, options, exception handling

### Secondary (MEDIUM confidence)

- `/Users/callumcowie/repos/Aether/tests/unit/cli-manifest.test.js` — Existing test pattern reference (how proxyquire and sinon are used; general AVA test structure)
- `/Users/callumcowie/repos/Aether/tests/unit/cli-hash.test.js` — Simpler AVA test reference (before/afterEach pattern)

---

## Metadata

**Confidence breakdown:**
- Agent format and frontmatter: HIGH — identical to Phase 28, all source files read directly
- Tool assignments per spec: HIGH — SPEC requirements are explicit; tools field is the mechanism
- AVA test patterns: HIGH — AVA 6.4.1 API verified via Context7; existing test files confirm project patterns
- js-yaml frontmatter parsing: HIGH — js-yaml 4.1.1 API verified via Context7; library already installed
- Read-only constraint implementation (TEST-03): HIGH — forbidden-only approach chosen (Claude's Discretion resolved)
- Probe's run/write scope: MEDIUM — Claude's Discretion, recommendation made but not externally verified
- TEST-05 intentional failure: HIGH — explicitly confirmed by user decision; 14 agents after Phase 29, 22 expected

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (agent format stable; AVA/js-yaml APIs stable; source OpenCode files won't change)
