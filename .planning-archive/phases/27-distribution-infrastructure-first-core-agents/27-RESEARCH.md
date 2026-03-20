# Phase 27: Distribution Infrastructure + First Core Agents - Research

**Researched:** 2026-02-20
**Domain:** Claude Code agent distribution pipeline, agent file format, npm packaging
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Agent file format:**
- Files named `aether-{role}.md` (e.g. `aether-builder.md`, `aether-watcher.md`)
- Files live in `.claude/agents/ant/` in both source and target repos
- Descriptions written as routing triggers — specific trigger cases that tell the Task tool WHEN to use the agent, not generic role labels
- Full XML body ported from OpenCode agent definitions — all instructions, failure modes, success criteria carried over
- All 8 PWR standards (PWR-01 through PWR-08) required for every agent, no exceptions

**Distribution pipeline:**
- Target repos receive agents at `.claude/agents/ant/` via `aether update`
- Hub path and pipeline approach are Claude's discretion — pick what fits existing architecture best
- GSD agent isolation is Claude's discretion — determine if directory structure alone is sufficient

**Conversion approach:**
- Builder and Watcher are template/exemplar conversions — future phases copy their structure exactly
- Spawn calls handled at Claude's discretion — determine best replacement pattern per agent
- Every converted agent must verify loading in Claude Code (appears in `/agents` output) — catches silent YAML issues
- All 8 PWR standards must pass for every converted agent

**Cleanup on removal:**
- Auto-delete: if agent file exists in target but not in hub, remove it during `aether update`
- Show changes: `aether update` lists added, updated, and removed agent files
- Overwrite/conflict behavior is Claude's discretion — match existing system file handling
- Idempotency approach is Claude's discretion — balance accuracy and speed

### Claude's Discretion

- Model field in frontmatter — decide based on what's proven to work in Claude Code
- Hub path structure — fit the existing hub layout
- Pipeline integration — same vs separate path from .aether/ system files
- GSD agent isolation mechanism
- Spawn call replacement pattern
- Conflict handling on local modifications
- Idempotency check method (content vs existence)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DIST-01 | `.claude/agents/ant/` added to package.json `files` array (GSD agents in parent dir excluded) | npm `files` field scoping — subdirectory entries like `.claude/agents/ant/` include only that subtree, not parent `.claude/agents/` |
| DIST-02 | cli.js `setupHub()` syncs `.claude/agents/ant/` to new hub path `~/.aether/system/agents-claude/` | `setupHub()` already has the `syncDirWithCleanup()` pattern working for 4 other source→hub paths; add a 5th for agents-claude |
| DIST-03 | update-transaction.js syncs from `~/.aether/system/agents-claude/` to target repo `.claude/agents/ant/` | `syncFiles()` already handles 4 hub→target paths via `syncDirWithCleanup()`; add a 5th; also add to `targetDirs`, `verifyIntegrity()`, `checkHubAccessible()` |
| DIST-04 | `npm pack --dry-run` confirms ant/ agents included, GSD agents excluded | Verified: `files` array scoping excludes parent directory contents by default |
| DIST-05 | `npm install -g .` populates hub with ant agents | Depends on DIST-01 + DIST-02 being complete |
| DIST-06 | `aether update` in a target repo delivers ant agents to `.claude/agents/ant/` | Depends on DIST-03 being complete |
| DIST-07 | Stale agent cleanup — removing agent from source removes it from target | `syncDirWithCleanup()` already performs cleanup phase; new path inherits this behavior automatically |
| DIST-08 | Hash-based skip — second run skips unchanged files (idempotent) | `syncDirWithCleanup()` already uses SHA-256 hash comparison; new path inherits this automatically |
| CORE-02 | Builder agent upgraded — XML body with TDD discipline, 3-Fix Rule, structured return format, coding standards. Tools: Read, Write, Edit, Bash, Grep, Glob | OpenCode builder already has all required content; conversion is Claude Code format adaptation + PWR standards pass |
| CORE-03 | Watcher agent upgraded — XML body with verification checklist, quality gates, structured pass/fail return. Tools: Read, Bash, Grep, Glob (no Write/Edit) | OpenCode watcher already has all required content; conversion requires removing Write/Edit from tools; Read-only enforcement is a key PWR-07 requirement |
| PWR-01 | Every agent has detailed execution flow | Builder and Watcher OpenCode definitions already have `<execution_flow>`-equivalent content; verify it's in XML and numbered |
| PWR-02 | Every agent has critical rules for common failure modes | Builder has "TDD Iron Law" and "3-Fix Rule"; Watcher has "Iron Law: Evidence before approval"; carry both over |
| PWR-03 | Every agent has structured return format | Both OpenCode agents have JSON output format blocks; carry them over |
| PWR-04 | Every agent has success criteria — self-verification checklist | Both OpenCode agents have `<success_criteria>` sections; carry them over |
| PWR-05 | Every agent has failure modes with escalation | Both OpenCode agents have `<failure_modes>` sections; carry them over |
| PWR-06 | Routing-effective descriptions | Current OpenCode descriptions are generic role labels; must rewrite as specific trigger cases |
| PWR-07 | Explicit tools field on every agent | Must add `tools:` frontmatter field to every Claude Code agent; Watcher verified as no Write/Edit |
| PWR-08 | All OpenCode-specific patterns removed | Remove: spawn calls (bash aether-utils.sh spawn-*), activity-log calls, aether-utils.sh flag-add calls. Replace with: escalation instructions, structured return, graceful degradation |
</phase_requirements>

---

## Summary

Phase 27 has two distinct workstreams: the distribution pipeline and the first two agent conversions. These are coupled — Builder and Watcher must be distributed through the proven pipeline, so both workstreams must succeed.

**Distribution pipeline:** The existing codebase already has a complete, tested pipeline for 4 source→hub→target paths (`.aether/` system files, Claude commands, OpenCode commands, OpenCode agents, rules). Adding Claude Code agents as a 5th path follows exactly the same pattern. The `syncDirWithCleanup()` function already provides hash-based idempotency (DIST-08) and stale file removal (DIST-07) for free. The primary work is adding a new `HUB_AGENTS_CLAUDE` constant and wiring it in 4 places: `cli.js setupHub()`, `update-transaction.js syncFiles()`, `update-transaction.js verifyIntegrity()`, and `update-transaction.js checkHubAccessible()`. The `package.json files` array already shows `.claude/commands/ant/` as a precedent — adding `.claude/agents/ant/` follows the same pattern and automatically excludes the GSD agents in the parent `.claude/agents/` directory.

**Agent conversion:** Claude Code subagents use Markdown files with YAML frontmatter. The critical pitfalls are: YAML malformation silently drops agents (verify via `/agents`), tool inheritance over-permissions agents (explicit `tools:` field required), and subagents cannot spawn other subagents (spawn calls must be removed). The OpenCode Builder and Watcher already contain all the substantive content needed (XML sections, failure modes, success criteria, return format). The conversion work is: adapt the frontmatter to Claude Code format, rewrite the description as routing triggers, add explicit `tools:` field, and remove all OpenCode-specific patterns (spawn calls, activity-log, flag-add).

**Primary recommendation:** Wire the new `agents-claude` hub path identically to the existing `commands/claude` hub path — it's the same code pattern applied to a new directory pair. For the agent conversions, treat the OpenCode definitions as the content source and the GSD agents as the format reference.

---

## Standard Stack

### Core

| Component | Version/Path | Purpose | Why Standard |
|-----------|-------------|---------|--------------|
| Node.js `fs` module | Built-in | File sync operations in cli.js and update-transaction.js | Already used throughout — no new dependency |
| Node.js `crypto` module | Built-in | SHA-256 hash comparison for idempotency | Already used by `hashFileSync()` in update-transaction.js |
| `syncDirWithCleanup()` | Existing function in update-transaction.js | Sync hub dir to target repo dir with hash skip and stale removal | Proven function with 15+ unit tests covering all required behaviors |
| YAML frontmatter | Claude Code spec | Agent configuration (name, description, tools, model) | Required by Claude Code — not optional |

### Supporting

| Component | Path | Purpose | When to Use |
|-----------|------|---------|-------------|
| `syncAetherToHub()` | cli.js | Sync .aether/ to hub (has EXCLUDE_DIRS logic) | Used for .aether/ system files only — not for agents path |
| `listFilesRecursive()` | cli.js / update-transaction.js | Enumerate files for sync | Already handles dotfile skip, used by syncDirWithCleanup |
| `hashFileSync()` | update-transaction.js | SHA-256 file comparison | Called automatically by syncDirWithCleanup |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New `HUB_AGENTS_CLAUDE` constant | Reuse existing `HUB_AGENTS` constant | Reusing would mix OpenCode agents and Claude Code agents in same hub dir — bad separation. Separate paths for separate targets. |
| `syncDirWithCleanup()` for agent sync | `syncAetherToHub()` | syncAetherToHub has EXCLUDE_DIRS logic specific to .aether/ structure — not appropriate for a flat agents directory |
| `tools: inherit` (omit tools field) | Explicit tools list | Tools inheritance silently over-permissions agents per confirmed pitfall — explicit list is required by PWR-07 |

---

## Architecture Patterns

### Recommended Project Structure

```
Source (Aether repo)
├── .claude/
│   ├── agents/              ← GSD agents (NOT distributed)
│   │   ├── gsd-*.md
│   │   └── ant/             ← NEW: ant agents (distributed)
│   │       ├── aether-builder.md
│   │       └── aether-watcher.md
│   └── commands/ant/        ← Slash commands (already distributed)
│
Hub (~/.aether/)
└── system/
    ├── agents-claude/        ← NEW: Claude Code ant agents
    │   ├── aether-builder.md
    │   └── aether-watcher.md
    └── agents/              ← Existing: OpenCode agents (unchanged)
        └── aether-*.md
│
Target repo (any repo with aether update)
└── .claude/
    ├── agents/              ← GSD agents (already there, user's own)
    └── agents/ant/          ← NEW: ant agents from hub
        ├── aether-builder.md
        └── aether-watcher.md
```

### Pattern 1: Hub Path Naming — `agents-claude`

**What:** Use `~/.aether/system/agents-claude/` for Claude Code agents, keeping `~/.aether/system/agents/` for OpenCode agents.

**Why:** The existing `agents/` path already syncs to `.opencode/agents/` in target repos. A separate `agents-claude/` path provides clean separation and avoids any risk of cross-contamination. This matches the existing `commands/claude` vs `commands/opencode` naming pattern already in use.

**Evidence:** Current hub structure:
```
~/.aether/system/
├── commands/
│   ├── claude/       ← Claude Code commands
│   └── opencode/     ← OpenCode commands
├── agents/           ← OpenCode agents (existing)
└── agents-claude/    ← Claude Code agents (NEW — fits pattern)
```

### Pattern 2: Wiring a New Sync Path

Adding a 5th sync path in `cli.js setupHub()`:

```javascript
// Source: existing pattern (lines 956-967 of cli.js for claude commands)

// NEW: Sync .claude/agents/ant/ -> ~/.aether/system/agents-claude/
const HUB_AGENTS_CLAUDE = path.join(HUB_SYSTEM_DIR, 'agents-claude');
const claudeAgentsSrc = path.join(PACKAGE_DIR, '.claude', 'agents', 'ant');
if (fs.existsSync(claudeAgentsSrc)) {
  const result = syncDirWithCleanup(claudeAgentsSrc, HUB_AGENTS_CLAUDE);
  log(`  Hub agents (claude): ${result.copied} files -> ${HUB_AGENTS_CLAUDE}`);
  if (result.removed.length > 0) {
    log(`  Hub agents (claude): removed ${result.removed.length} stale files`);
    for (const f of result.removed) log(`    - ${f}`);
  }
}
```

And in `update-transaction.js syncFiles()`:

```javascript
// Source: existing pattern (lines 861-865 of update-transaction.js for opencode agents)

// NEW: Sync agents-claude from hub to .claude/agents/ant/
const repoClaudeAgents = path.join(this.repoPath, '.claude', 'agents', 'ant');
if (fs.existsSync(this.HUB_AGENTS_CLAUDE)) {
  results.agents_claude = this.syncDirWithCleanup(this.HUB_AGENTS_CLAUDE, repoClaudeAgents, { dryRun });
}
```

### Pattern 3: Claude Code Agent Frontmatter

Verified against official documentation (https://code.claude.com/docs/en/sub-agents):

```markdown
---
name: aether-builder
description: Use this agent when implementing code, creating files, executing builds, or running commands to make a plan real. Spawned by /ant:build and /ant:continue when the colony needs hands-on implementation work. Also use directly when a task requires TDD discipline, the 3-Fix Rule for debugging, or systematic file creation.
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

[XML body with execution_flow, critical_rules, return_format, success_criteria, failure_modes]
```

**Required fields:** `name`, `description`, `tools` (explicit — no inheritance)
**Recommended fields:** `model: inherit` (default behavior, explicit for clarity)
**Optional fields (not needed for these agents):** `permissionMode`, `maxTurns`, `memory`, `hooks`, `skills`, `background`, `isolation`

### Pattern 4: GSD Agent Isolation

**What:** Directory structure alone is sufficient for GSD agent isolation. GSD agents live in `.claude/agents/` (parent directory). Ant agents live in `.claude/agents/ant/` (subdirectory).

**How package.json excludes GSD agents:**
The `files` array in package.json uses `.claude/agents/ant/` — this is a specific subdirectory path. npm includes only files matching this path, not files in the parent `.claude/agents/` directory. This is confirmed by the existing `.claude/commands/ant/` entry which only packages the `ant/` subdirectory, not the parent `.claude/commands/` directory.

**Verification command:** `npm pack --dry-run | grep claude/agents` — should show only `ant/` files, no `gsd-*.md` files.

### Pattern 5: Spawn Call Replacement

**What:** OpenCode agents use `bash .aether/aether-utils.sh spawn-*` calls for worker spawning. Claude Code subagents cannot spawn other subagents (confirmed by official docs: "Subagents cannot spawn other subagents").

**Replacement approach:** Remove spawn machinery entirely. Replace the conceptual behavior with escalation instructions in the failure modes section:

```
# Remove (OpenCode pattern):
bash .aether/aether-utils.sh spawn-can-spawn {your_depth}
bash .aether/aether-utils.sh generate-ant-name "{caste}"
bash .aether/aether-utils.sh spawn-log "{your_name}" "{caste}" "{child_name}" "{task}"

# Replace with (escalation instruction):
If task is 3x larger than expected or requires genuinely different expertise,
STOP and escalate to the calling command with:
- What was attempted
- Why it exceeded scope
- What specialized work is needed
The calling command (e.g., /ant:build) will handle re-routing.
```

### Pattern 6: Activity-Log Replacement

**What:** OpenCode agents use `bash .aether/aether-utils.sh activity-log` calls for progress tracking. Claude Code agents have no equivalent mechanism.

**Replacement approach:** Remove activity-log calls. Progress is communicated through the agent's structured return format. Add a note at the top of the agent body: "Progress is tracked through structured returns, not activity logs."

### Pattern 7: Conflict Handling on Local Modifications

**What:** When `aether update` runs and a target repo has local modifications to agent files, how should conflicts be handled?

**Recommendation:** Match existing system file handling — overwrite. The `syncDirWithCleanup()` function already overwrites files when hashes differ. Agent files in target repos are system files (not user data), so they should be overwritten just like command files and `.aether/` system files. The git dirty check in `updateRepo()` already warns users about dirty files in target dirs; adding `.claude/agents/ant` to `targetDirs` handles this consistently.

### Anti-Patterns to Avoid

- **Omitting the `tools:` field:** Claude Code inherits ALL tools if `tools:` is omitted. This over-permissions agents. Every agent must have an explicit `tools:` list per PWR-07.
- **Reusing `HUB_AGENTS` constant for claude agents:** The existing `HUB_AGENTS` points to `~/.aether/system/agents/` which syncs to `.opencode/agents/`. Reusing it would mix agent formats.
- **Using `syncAetherToHub()` for claude agents:** This function has EXCLUDE_DIRS logic for `.aether/` private directories — wrong for a flat agents directory.
- **Leaving spawn calls in agent body:** Claude Code subagents cannot spawn other subagents. Spawn calls will simply fail silently or error. Must be removed per PWR-08.
- **YAML malformation:** Any YAML syntax error in frontmatter causes the agent to be silently dropped from `/agents`. Common causes: unescaped colons in description, missing quotes around values with special characters. Verify with `/agents` after creation.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Hash-based file comparison | Custom hash logic | Existing `hashFileSync()` + `syncDirWithCleanup()` | Already tested with 15+ unit tests covering all edge cases |
| Stale file removal | Custom cleanup loop | Existing `syncDirWithCleanup()` cleanup phase | Already implemented and tested |
| Directory creation | Custom mkdir logic | `syncDirWithCleanup()` calls `fs.mkdirSync(dest, { recursive: true })` | Built into the sync function |
| Agent YAML validation | Custom frontmatter parser | Rely on Claude Code's `/agents` command to verify loading | Official verification path; catches all parse errors |

**Key insight:** 80%+ of the distribution infrastructure already exists. The new `agents-claude` path is a configuration addition, not a new implementation.

---

## Common Pitfalls

### Pitfall 1: YAML Malformation Silently Drops Agents

**What goes wrong:** Agent file exists on disk but doesn't appear in `/agents` output. No error is shown.
**Why it happens:** Claude Code parses YAML frontmatter at session start. Any syntax error causes the entire file to be silently skipped.
**How to avoid:** After creating each agent file, verify it appears in `/agents`. Common YAML problems:
  - Unescaped colons in description values (e.g., `description: Use for X: Y` — colon in value breaks YAML)
  - Use double-quoted strings for descriptions: `description: "Use for X: Y"`
  - Unescaped special characters (`{`, `}`, `[`, `]`, `#`)
**Warning signs:** Agent file exists but `/agents` doesn't list it

### Pitfall 2: Tool Inheritance Over-Permissions Agents

**What goes wrong:** Watcher agent can write files even though its role is read-only.
**Why it happens:** Omitting `tools:` field causes the agent to inherit ALL tools from parent conversation, including Write and Edit.
**How to avoid:** Every agent must have an explicit `tools:` field. For Watcher: `tools: Read, Bash, Grep, Glob`. Do NOT include Write or Edit.
**Warning signs:** Agent can perform operations not intended by its role

### Pitfall 3: Spawn Calls Left in Agent Body

**What goes wrong:** Agent attempts to spawn a sub-worker and fails.
**Why it happens:** Claude Code confirmed: "Subagents cannot spawn other subagents." The restriction is architectural.
**How to avoid:** Remove all `bash .aether/aether-utils.sh spawn-*` calls. Replace with escalation instructions that tell the agent to return a "blocked" status with what specialist work is needed.
**Warning signs:** Any `spawn-can-spawn`, `generate-ant-name`, or `spawn-log` calls in agent body

### Pitfall 4: Missing `HUB_AGENTS_CLAUDE` in `update-transaction.js`

**What goes wrong:** `cli.js setupHub()` syncs to hub correctly, but `aether update` never delivers agents to target repos.
**Why it happens:** `update-transaction.js` has its own set of hub path constants (duplicated from cli.js). Both files must be updated.
**How to avoid:** Search for every place `HUB_AGENTS` is referenced in update-transaction.js and add the parallel `HUB_AGENTS_CLAUDE` reference.
**Files that need the new constant:** `cli.js` (setupHub), `update-transaction.js` (constructor, syncFiles, verifyIntegrity, checkHubAccessible)

### Pitfall 5: Forgetting `targetDirs` in `update-transaction.js`

**What goes wrong:** Git dirty check doesn't protect `.claude/agents/ant/` from being overwritten without warning.
**Why it happens:** The `targetDirs` array drives the git dirty file check before update. New target directories must be added.
**How to avoid:** Add `.claude/agents/ant` to `this.targetDirs` in `update-transaction.js` constructor. Also add to `targetDirs` in `updateRepo()` in cli.js.

### Pitfall 6: Checkpoint Allowlist Not Updated

**What goes wrong:** `aether update --force` (which uses git stash checkpoint) doesn't capture `.claude/agents/ant/` files. Changes to ant agents get lost in stash operations.
**Why it happens:** The `CHECKPOINT_ALLOWLIST` in cli.js controls which files are included in checkpoint stashes. New managed directories must be explicitly added.
**How to avoid:** Add `.claude/agents/ant/**` to `CHECKPOINT_ALLOWLIST` in cli.js.

### Pitfall 7: `init.js` Has Stale Hub Path References

**What goes wrong:** `aether init` (new repo initialization) doesn't deliver claude agents even after setupHub and update work correctly.
**Why it happens:** `bin/lib/init.js` has its own hub path constants using an older path structure (`HUB_DIR/agents` not `HUB_SYSTEM_DIR/system/agents`). The `init` path is separate from the `update` path.
**Warning signs:** `aether update` delivers agents but `aether init` on a fresh repo doesn't.
**How to avoid:** Check init.js — currently `HUB_AGENTS = path.join(HUB_DIR, 'agents')` uses the old path, not `HUB_SYSTEM/agents`. May need updating depending on whether init is in scope for this phase. If out of scope, document as a known gap.

### Pitfall 8: Description Routing Effectiveness

**What goes wrong:** Task tool rarely delegates to Builder or Watcher because descriptions don't match real trigger cases.
**Why it happens:** Current OpenCode descriptions (`"Use this agent for code implementation..."`) are generic role labels. Claude uses exact description matching to decide delegation.
**How to avoid:** Write descriptions as specific trigger cases:
  - Good: `"Use this agent when implementing code from a plan, creating files to spec, running builds, or applying TDD cycles. Spawned by /ant:build and /ant:continue."`
  - Bad: `"The builder turns plans into working code."`
**Warning signs:** Agent doesn't appear in delegation decisions even when task matches role

---

## Code Examples

Verified patterns from official sources and existing codebase:

### Claude Code Agent File Format (Official)

```markdown
---
name: aether-builder
description: "Use this agent when implementing code, creating files, executing builds, or running commands. Use when TDD discipline or the 3-Fix Rule is needed. Spawned by /ant:build and /ant:continue."
tools: Read, Write, Edit, Bash, Grep, Glob
model: inherit
---

<execution_flow>
## Execution Flow

1. Read task specification completely before writing any code
2. RED: Write failing test first
3. VERIFY RED: Run test, confirm failure
4. GREEN: Write minimal code to pass
5. VERIFY GREEN: Run test, confirm pass
6. REFACTOR: Clean up while staying green
</execution_flow>
...
```

### New Hub Constant (cli.js)

```javascript
// Add after existing HUB_AGENTS constant (line 76)
const HUB_AGENTS_CLAUDE = path.join(HUB_SYSTEM_DIR, 'agents-claude');
```

### setupHub() Addition (cli.js)

```javascript
// Add after the existing opencode agents sync block (after line 989)
// Sync .claude/agents/ant/ -> ~/.aether/system/agents-claude/
const claudeAgentsSrc = path.join(PACKAGE_DIR, '.claude', 'agents', 'ant');
if (fs.existsSync(claudeAgentsSrc)) {
  fs.mkdirSync(HUB_AGENTS_CLAUDE, { recursive: true });
  const result = syncDirWithCleanup(claudeAgentsSrc, HUB_AGENTS_CLAUDE);
  log(`  Hub agents (claude): ${result.copied} files, ${result.skipped} unchanged -> ${HUB_AGENTS_CLAUDE}`);
  if (result.removed.length > 0) {
    log(`  Hub agents (claude): removed ${result.removed.length} stale files`);
    for (const f of result.removed) log(`    - ${f}`);
  }
}
```

### Constructor Addition (update-transaction.js)

```javascript
// Add after HUB_AGENTS constant (line 167)
this.HUB_AGENTS_CLAUDE = path.join(this.HUB_SYSTEM_DIR, 'agents-claude');

// Add to targetDirs array (line 177)
this.targetDirs = ['.aether', '.claude/commands/ant', '.claude/rules', '.opencode/commands/ant', '.opencode/agents', '.claude/agents/ant'];
```

### syncFiles() Addition (update-transaction.js)

```javascript
// Add after existing agents sync block (after line 865)
// Sync claude agents from hub to .claude/agents/ant/
const repoClaudeAgents = path.join(this.repoPath, '.claude', 'agents', 'ant');
if (fs.existsSync(this.HUB_AGENTS_CLAUDE)) {
  results.agents_claude = this.syncDirWithCleanup(this.HUB_AGENTS_CLAUDE, repoClaudeAgents, { dryRun });
}
```

### verifyIntegrity() Addition (update-transaction.js)

```javascript
// Add after line 918
verifyDir(this.HUB_AGENTS_CLAUDE, path.join(this.repoPath, '.claude', 'agents', 'ant'));
```

### package.json files array addition

```json
"files": [
  "bin/",
  ".claude/commands/ant/",
  ".claude/agents/ant/",    // NEW
  ".opencode/commands/ant/",
  ".opencode/agents/",
  ".opencode/opencode.json",
  ".aether/",
  "README.md",
  "LICENSE",
  "DISCLAIMER.md",
  "CHANGELOG.md"
]
```

### CHECKPOINT_ALLOWLIST addition (cli.js)

```javascript
const CHECKPOINT_ALLOWLIST = [
  '.aether/*.md',
  '.claude/commands/ant/**',
  '.claude/agents/ant/**',    // NEW
  '.opencode/commands/ant/**',
  '.opencode/agents/**',
  'bin/cli.js',
];
```

### Removed OpenCode Patterns (Builder example)

```bash
# REMOVE these from agent body:
bash .aether/aether-utils.sh activity-log "ACTION" "{your_name} (Builder)" "description"
bash .aether/aether-utils.sh spawn-can-spawn {your_depth}
bash .aether/aether-utils.sh generate-ant-name "{caste}"
bash .aether/aether-utils.sh spawn-log "{your_name}" "{caste}" "{child_name}" "{task}"
bash .aether/aether-utils.sh flag-add "blocker" "{issue_title}" "{description}" "verification" {phase}

# REPLACE spawn section with:
## When to Escalate
If you encounter a task 3x larger than expected or requiring genuinely different expertise,
STOP and return status "blocked" with:
- what_attempted: [what you tried]
- escalation_reason: [why it exceeded scope]
- specialist_needed: [what type of work is required]
The calling orchestrator (/ant:build, /ant:continue) handles re-routing.
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| runtime/ staging directory | Direct .aether/ packaging | Phase 20 (v4.0) | .aether/ is now source of truth, published directly |
| bin/sync-to-runtime.sh | bin/validate-package.sh | Phase 20 | Validation only, no copying |
| Agents distributed alongside system files | Agents in separate `agents/` hub path | v3.x | Clean separation from .aether/ system files |
| No Claude Code agents | `.claude/agents/ant/` distributed via hub | Phase 27 (this phase) | First Claude Code-native agents |

---

## Open Questions

1. **init.js hub path for claude agents**
   - What we know: `bin/lib/init.js` uses `HUB_DIR/agents` (old path, without `system/`). The `aether init` command uses init.js to set up new repos, separate from `aether update`.
   - What's unclear: Whether init should also deliver claude agents, and whether init.js needs a parallel claude-agents sync path added.
   - Recommendation: Check if `aether init` is expected to deliver agents (likely yes). If so, init.js needs `HUB_AGENTS_CLAUDE = path.join(HUB_SYSTEM, 'agents-claude')` added and a sync call. This is a small addition but easy to miss. Mark as task in plan.

2. **Result tracking for agents_claude in output**
   - What we know: The `syncFiles()` result object has `agents` key. Adding `agents_claude` adds a new key that callers (cli.js updateRepo reporting) must handle.
   - What's unclear: Whether to merge agents_claude counts into the existing `agents` count (simpler reporting) or track separately (more precise).
   - Recommendation: Merge into `agents` count for CLI reporting. The `sync_result` object can have a separate `agents_claude` key for internal use. This avoids changing CLI output format.

3. **Verification of `/agents` command loading**
   - What we know: Agents load at session start. Manual file creation requires restart or `/agents` reload.
   - What's unclear: Exact CLI or command sequence to verify agent loading without manual Claude Code session.
   - Recommendation: The plan should include a manual verification step: after creating agent files, run `/agents` in a Claude Code session and confirm both agents appear by name. This cannot be automated via bash.

---

## Sources

### Primary (HIGH confidence)
- Official Claude Code docs, https://code.claude.com/docs/en/sub-agents — Complete frontmatter spec, tool inheritance rules, spawn restriction, `/agents` verification, scope table
- `/Users/callumcowie/repos/Aether/bin/cli.js` — setupHub() implementation, existing sync patterns, CHECKPOINT_ALLOWLIST, PACKAGE_DIR, hub constants
- `/Users/callumcowie/repos/Aether/bin/lib/update-transaction.js` — syncFiles(), syncDirWithCleanup(), verifyIntegrity(), checkHubAccessible(), targetDirs, HUB_AGENTS constants
- `/Users/callumcowie/repos/Aether/package.json` — files array structure showing existing `.claude/commands/ant/` precedent
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-builder.md` — Complete Builder content for conversion
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-watcher.md` — Complete Watcher content for conversion
- `/Users/callumcowie/repos/Aether/.planning/REQUIREMENTS.md` — PWR-01 through PWR-08 definitions, DIST-01 through DIST-08 definitions
- `/Users/callumcowie/repos/Aether/tests/unit/cli-sync.test.js` — Existing syncDirWithCleanup test patterns (15 tests covering hash skip, cleanup, idempotency)

### Secondary (MEDIUM confidence)
- `npm pack --dry-run` output — Confirmed existing files array scoping excludes parent `.claude/agents/` when only `.claude/agents/ant/` is listed
- Hub directory listing `~/.aether/system/` — Confirmed existing `agents/` path for OpenCode agents, no `agents-claude/` path yet

---

## Metadata

**Confidence breakdown:**
- Distribution pipeline changes: HIGH — all code is in-repo, patterns are established, confirmed by reading existing implementations
- Agent file format: HIGH — verified against official Claude Code docs
- Agent content conversion: HIGH — OpenCode source files read directly, changes are well-defined (remove patterns, rewrite description, add tools field)
- Spawn/activity-log replacement: HIGH — official docs confirm spawn restriction; replacement approach is straightforward
- init.js gap: MEDIUM — flagged as open question; needs investigation during planning

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (Claude Code agent format is stable; pipeline code is in-repo and won't drift)
