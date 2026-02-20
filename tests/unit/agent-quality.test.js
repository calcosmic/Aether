/**
 * Agent Quality Test Suite
 *
 * Enforces quality standards on ALL agent files in .claude/agents/ant/
 * covering frontmatter completeness, naming conventions, read-only tool
 * restrictions, forbidden body patterns, agent count tracking, and body
 * quality (XML sections present, no empty sections, minimum content length).
 *
 * TEST-05 now passes — Phase 30 completed the full 22-agent roster.
 */

'use strict';

const test = require('ava');
const yaml = require('js-yaml');
const fs = require('fs');
const path = require('path');

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

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
    // body is everything after second --- delimiter
    const body = parts.slice(2).join('---');
    return { frontmatter, body, filename: path.basename(filePath) };
  } catch (e) {
    return null; // Malformed YAML — caught as test failure
  }
}

// IMPORTANT: yaml.load() returns tools as a STRING, not an array.
// Always use parseTools() for set-based comparisons to avoid substring false positives.
function parseTools(toolsString) {
  if (!toolsString) return [];
  return toolsString.split(',').map(t => t.trim()).filter(Boolean);
}

// ---------------------------------------------------------------------------
// Read-only constraints registry
// ---------------------------------------------------------------------------

// Forbidden tools per read-only agent type (forbidden-only approach — flexible
// against future tool additions while enforcing the constraint that matters)
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

// ---------------------------------------------------------------------------
// Forbidden OpenCode body patterns
// ---------------------------------------------------------------------------

// All patterns matched as "aether-utils.sh <command>" (the actual bash invocation form)
// rather than bare strings. This avoids false positives when agents legitimately DOCUMENT
// which patterns they must not invoke (e.g., the Queen's critical_rules section includes
// "Do not use: `activity-log`, `spawn-can-spawn`..." as documentation — that is a prohibition,
// not an invocation).
//
// Every actual OpenCode invocation routes through aether-utils.sh:
//   bash .aether/aether-utils.sh activity-log "ACTION" ...
//   bash .aether/aether-utils.sh spawn-can-spawn ...
// Matching on the aether-utils.sh prefix eliminates the false positive class entirely.
const FORBIDDEN_PATTERNS = [
  { pattern: /aether-utils\.sh activity-log/, name: 'activity-log (aether-utils.sh invocation)' },
  { pattern: /aether-utils\.sh spawn-can-spawn/, name: 'spawn-can-spawn (aether-utils.sh invocation)' },
  { pattern: /aether-utils\.sh generate-ant-name/, name: 'generate-ant-name (aether-utils.sh invocation)' },
  { pattern: /aether-utils\.sh spawn-log/, name: 'spawn-log (aether-utils.sh invocation)' },
  { pattern: /aether-utils\.sh spawn-complete/, name: 'spawn-complete (aether-utils.sh invocation)' },
];

// ---------------------------------------------------------------------------
// Required XML sections (established in Phase 27)
// ---------------------------------------------------------------------------

const REQUIRED_XML_SECTIONS = [
  'role',
  'execution_flow',
  'critical_rules',
  'return_format',
  'success_criteria',
  'failure_modes',
  'escalation',
  'boundaries',
];

const MIN_SECTION_CONTENT_LENGTH = 50;

function checkBodyQuality(body) {
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
        errors.push(
          `Section ${openTag} too short (${content.length} chars < ${MIN_SECTION_CONTENT_LENGTH} required — possible placeholder stub)`
        );
      }
    }
  }
  return errors;
}

// ---------------------------------------------------------------------------
// TEST-01: Frontmatter completeness
// ---------------------------------------------------------------------------

test('TEST-01: all agent files have required YAML frontmatter', t => {
  const files = getAgentFiles();
  t.true(files.length > 0, 'No agent files found in .claude/agents/ant/');

  for (const filePath of files) {
    const filename = path.basename(filePath);
    const parsed = parseAgentFile(filePath);

    t.truthy(parsed, `${filename}: failed to parse frontmatter — malformed YAML or missing --- delimiters`);
    if (!parsed) continue;

    t.truthy(parsed.frontmatter.name, `${filename}: missing required frontmatter field: name`);
    t.truthy(parsed.frontmatter.description, `${filename}: missing required frontmatter field: description`);
    t.truthy(parsed.frontmatter.tools, `${filename}: missing required frontmatter field: tools`);
  }
});

// ---------------------------------------------------------------------------
// TEST-02: Name pattern validation
// ---------------------------------------------------------------------------

test('TEST-02: agent names match aether-{role} pattern', t => {
  const NAME_PATTERN = /^aether-[a-z][a-z0-9-]+$/;
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    const filename = path.basename(filePath, '.md');
    if (!parsed) continue;

    // Frontmatter name must match aether-{role} pattern
    t.regex(
      parsed.frontmatter.name,
      NAME_PATTERN,
      `${filename}: name "${parsed.frontmatter.name}" does not match aether-{role} pattern (lowercase, hyphens only)`
    );

    // Frontmatter name must match filename (minus .md extension)
    t.is(
      parsed.frontmatter.name,
      filename,
      `${filename}: frontmatter name "${parsed.frontmatter.name}" does not match filename "${filename}"`
    );
  }
});

// ---------------------------------------------------------------------------
// TEST-03: Read-only enforcement
// ---------------------------------------------------------------------------

test('TEST-03: read-only agents have no forbidden tools', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    const agentName = parsed.frontmatter.name;
    const constraints = READ_ONLY_CONSTRAINTS[agentName];
    if (!constraints) continue; // Not a constrained agent — skip

    // Use parseTools() to get array — never call .includes() on raw string
    const tools = parseTools(parsed.frontmatter.tools);

    for (const forbidden of constraints.forbidden) {
      t.false(
        tools.includes(forbidden),
        `${agentName}: must not have "${forbidden}" in tools (read-only agent constraint)`
      );
    }
  }
});

// ---------------------------------------------------------------------------
// TEST-04: No OpenCode-specific patterns in agent body
// ---------------------------------------------------------------------------

test('TEST-04: no agent body contains OpenCode-specific invocations', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    for (const { pattern, name } of FORBIDDEN_PATTERNS) {
      t.false(
        pattern.test(parsed.body),
        `${parsed.filename}: found forbidden OpenCode pattern "${name}" in agent body`
      );
    }
  }
});

// ---------------------------------------------------------------------------
// TEST-05: Agent count
// ---------------------------------------------------------------------------

// Phase 30 complete — all 22 agents shipped:
// Phase 27-28: queen, builder, watcher, probe, weaver, keeper, scout, route-setter, surveyor (9)
// Phase 29: tracker, auditor, (+ probe/weaver/keeper/watcher adjustments) → 14 total
// Phase 30: ambassador, archaeologist, chaos, chronicler, gatekeeper, includer, measurer, sage (8)
const EXPECTED_AGENT_COUNT = 22;

test('TEST-05: agent count matches expected 22', t => {
  const files = getAgentFiles();
  t.is(
    files.length,
    EXPECTED_AGENT_COUNT,
    `Expected ${EXPECTED_AGENT_COUNT} agents, found ${files.length}. Remaining: ${EXPECTED_AGENT_COUNT - files.length} agents needed (Phase 30).`
  );
});

// ---------------------------------------------------------------------------
// Body quality: all 8 XML sections present, non-empty, >= 50 chars each
// ---------------------------------------------------------------------------

test('body quality: all agents have 8 XML sections with adequate content', t => {
  const files = getAgentFiles();

  for (const filePath of files) {
    const parsed = parseAgentFile(filePath);
    if (!parsed) continue;

    const errors = checkBodyQuality(parsed.body);
    t.deepEqual(
      errors,
      [],
      `${parsed.filename}: body quality errors:\n  - ${errors.join('\n  - ')}`
    );
  }
});

// ---------------------------------------------------------------------------
// CLEAN-03: No bash wrapping bug in command files
// ---------------------------------------------------------------------------

/**
 * Scans command files for the bash wrapping bug pattern where
 * `with description "..."` appears INSIDE bash code blocks instead of
 * in the instruction prose above them.
 *
 * The bug causes "with: command not found" errors when Claude Code
 * executes the bash blocks literally.
 *
 * Correct pattern:
 *   Run using the Bash tool with description "Doing something...":
 *   ```bash
 *   bash .aether/aether-utils.sh some-command "args"
 *   ```
 *
 * Bug pattern (FAILS):
 *   ```bash
 *   bash .aether/aether-utils.sh some-command "args" with description "Doing something..."
 *   ```
 */
function findBashWrappingBug(commandDirs) {
  const errors = [];

  for (const dir of commandDirs) {
    if (!fs.existsSync(dir)) continue;

    const files = fs.readdirSync(dir)
      .filter(f => f.endsWith('.md'))
      .map(f => path.join(dir, f));

    for (const filePath of files) {
      const content = fs.readFileSync(filePath, 'utf8');
      const lines = content.split('\n');
      let inBashBlock = false;
      let blockStartLine = 0;

      for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        // Track bash code block boundaries
        if (line.startsWith('```bash')) {
          inBashBlock = true;
          blockStartLine = i + 1;
          continue;
        }
        if (line.startsWith('```') && inBashBlock) {
          inBashBlock = false;
          continue;
        }

        // Check for the bug pattern inside bash blocks
        if (inBashBlock && / with description /.test(line)) {
          const relPath = path.relative(process.cwd(), filePath);
          errors.push(
            `${relPath}:${i + 1}: bash wrapping bug — 'with description' inside code block ` +
            `(block starts at line ${blockStartLine})`
          );
        }
      }
    }
  }

  return errors;
}

test('CLEAN-03: no bash commands contain "with description" suffix inside code blocks', t => {
  const commandDirs = [
    path.join(__dirname, '../../.claude/commands/ant'),
    path.join(__dirname, '../../.opencode/commands/ant'),
  ];

  const errors = findBashWrappingBug(commandDirs);

  t.deepEqual(
    errors,
    [],
    `Found bash wrapping bug pattern in command files:\n  - ${errors.join('\n  - ')}\n\n` +
    'The description should be in instruction prose ABOVE the bash block, not inside it.'
  );
});
