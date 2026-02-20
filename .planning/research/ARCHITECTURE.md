# Architecture Research

**Domain:** Claude Code subagent integration with Aether colony system
**Researched:** 2026-02-20
**Confidence:** HIGH (direct source inspection — bin/cli.js, update-transaction.js, package.json, build.md, agent files)

> **Note:** This file was updated for the Claude Code subagents milestone. The previous content (v1.3 The Great Restructuring) has been superseded by the actual v4.0 architecture that shipped.

---

## Current Architecture Baseline (Verified at v4.0)

Before recommending changes, the actual state of the distribution pipeline:

### Agent Registries (Two Exist, One Distributed)

```
Aether Repo
├── .opencode/agents/          ← 22 agents, packaged in npm, distributed to all repos
│   ├── aether-builder.md
│   ├── aether-queen.md
│   └── ... (22 total)
│
└── .claude/agents/            ← 11 agents, NOT packaged, NOT distributed
    ├── gsd-executor.md        ← GSD system agents, local-only by design
    ├── gsd-planner.md
    └── ... (11 GSD agents)
```

The `package.json` files array currently includes `.opencode/agents/` but NOT `.claude/agents/`:

```json
"files": [
  "bin/",
  ".claude/commands/ant/",
  ".opencode/commands/ant/",
  ".opencode/agents/",          ← distributed
  ".opencode/opencode.json",
  ".aether/",
  ...
]
```

### Distribution Flow (Current)

```
npm install -g .
    ↓ (setupHub() in bin/cli.js lines 980-989)
~/.aether/system/agents/        ← .opencode/agents/ contents only

aether update (in any registered repo)
    ↓ (UpdateTransaction lines 861-865)
target-repo/.opencode/agents/   ← agents land here only
target-repo/.claude/agents/     ← does NOT exist (not synced)
```

### How Slash Commands Currently Spawn Subagents

From `build.md` (verified lines 392, 567, 765, 843):

```
# Archaeologist spawn:
Task tool with subagent_type="aether-archaeologist"

# Builder spawn:
Task tool with subagent_type="aether-builder"

# Watcher spawn:
Task tool with subagent_type="aether-watcher"

# Chaos spawn:
Task tool with subagent_type="aether-chaos"
```

Claude Code resolves `subagent_type="aether-builder"` by looking up `.claude/agents/aether-builder.md`. These agent files do not currently exist. The fallback comment in `build.md` documents the current workaround:

```
# FALLBACK: If "Agent type not found", use general-purpose and inject role:
# "You are an Archaeologist Ant - git historian..."
```

This confirms the slash commands are already wired for Claude Code subagents — the agent files simply do not yet exist.

---

## Recommended Architecture

### Where Agent Files Should Live

**`.claude/agents/aether-*.md`** in the Aether repo, distributed via the hub.

Three reasons this is the only viable location:

1. Claude Code's subagent resolution is hardcoded: `subagent_type="aether-builder"` resolves to `.claude/agents/aether-builder.md`. The slash commands already use this convention.

2. The GSD system already demonstrates this works: `gsd-executor.md`, `gsd-planner.md` etc. in `.claude/agents/` are functioning Claude Code subagents.

3. Keeping Claude Code and OpenCode agents separate by directory (`.claude/agents/` vs `.opencode/agents/`) matches each runtime's native convention and prevents cross-contamination.

### File Naming

**Use `aether-*` prefix matching OpenCode agents exactly** (e.g., `aether-builder.md`, `aether-watcher.md`).

The slash commands already reference `subagent_type="aether-builder"`. Changing the name prefix would require modifying every slash command that uses `subagent_type`. There is no choice to make here — the name is already decided by 34 existing command files.

### Agent File Format Difference

**OpenCode format** (`.opencode/agents/aether-builder.md`):
```markdown
---
name: aether-builder
description: "Use this agent for code implementation..."
---

You are a Builder Ant...
```

**Claude Code format** (`.claude/agents/aether-builder.md`):
```markdown
---
name: aether-builder
description: "Use this agent for code implementation..."
tools: Read, Write, Edit, Bash, Grep, Glob
color: cyan
---

You are a Builder Ant...
```

The only frontmatter difference is `tools` and `color`. Role content (TDD discipline, failure modes, success criteria, output JSON format, spawn protocol) should be identical between the two versions.

---

## System Overview After Integration

```
┌─────────────────────────────────────────────────────────────────────┐
│                      AETHER REPO (SOURCE)                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  .opencode/agents/          .claude/agents/                         │
│  ├── aether-builder.md      ├── aether-builder.md  ← NEW            │
│  ├── aether-watcher.md      ├── aether-watcher.md  ← NEW            │
│  ├── aether-queen.md        ├── aether-queen.md    ← NEW            │
│  └── ... (22 agents)        ├── ... (22 aether-*)  ← NEW            │
│                             └── gsd-*.md           ← existing, kept │
│                                                                     │
│  package.json files[] adds ".claude/agents/"  ← 1 line change       │
└────────────────────────────────┬────────────────────────────────────┘
                                 │ npm install -g .
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         HUB (~/.aether/)                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  system/                                                            │
│  ├── agents/              ← OpenCode agents (existing)              │
│  └── agents-claude/       ← Claude Code agents (NEW hub path)       │
│                                                                     │
└────────────────────────────────┬────────────────────────────────────┘
                                 │ aether update
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     TARGET REPO (WORKING COPY)                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  .opencode/agents/        ← OpenCode agents (existing)              │
│  .claude/agents/          ← Claude Code agents (NEW destination)    │
│  .claude/commands/ant/    ← Slash commands (existing, unchanged)    │
│  .aether/                 ← System files (unchanged)                │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Component Responsibilities

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| Slash commands (`.claude/commands/ant/`) | Orchestration, wave management, state updates | Subagents via Task tool (`subagent_type`) |
| Claude Code agents (`.claude/agents/aether-*.md`) | Specialized task execution, returns JSON | Parent via Task tool return value |
| OpenCode agents (`.opencode/agents/aether-*.md`) | Same roles, OpenCode runtime | OpenCode orchestration |
| Hub agents-claude (`~/.aether/system/agents-claude/`) | Distribution store for Claude agents | Source repo (written by setupHub), target repos (read by UpdateTransaction) |
| `bin/cli.js` setupHub() | Pushes Claude agents to hub on `npm install -g .` | `.claude/agents/` → `~/.aether/system/agents-claude/` |
| UpdateTransaction | Pulls Claude agents from hub to target repo | `~/.aether/system/agents-claude/` → `.claude/agents/` |

---

## Data Flow: Slash Command to Subagent

```
User runs /ant:build 3
    ↓
build.md executes (Queen context in Claude Code)
    ↓
Reads COLONY_STATE.json, constraints.json, pheromone signals
    ↓
bash .aether/aether-utils.sh generate-ant-name "builder"
    → Returns: "Hammer-42"
    ↓
Task tool call:
  subagent_type = "aether-builder"
  description   = "Builder Hammer-42: Task 3.1 — implement auth endpoint"
  prompt        = "[full builder prompt with goal, task, pheromones, queen wisdom]"
    ↓
Claude Code resolves: .claude/agents/aether-builder.md
    ↓
Subagent runs with injected prompt context
  (goal, task_id, archaeology_context, queen_wisdom, pheromone_section)
    ↓
Returns JSON:
  { ant_name, task_id, status, summary,
    tool_count, files_created, files_modified,
    tests_written, blockers }
    ↓
Queen processes results, updates spawn tree, colony state
```

---

## Project Structure Changes

### New Files to Create

```
.claude/agents/
├── aether-builder.md          ← Core caste (every build)
├── aether-watcher.md          ← Core caste (every build)
├── aether-chaos.md            ← Core caste (every build)
├── aether-archaeologist.md    ← Used in pre-build scan
├── aether-scout.md            ← Research phases
├── aether-queen.md            ← If used as nested subagent
├── aether-route-setter.md     ← Planning phases
├── aether-weaver.md           ← Refactor phases
├── aether-probe.md            ← Test coverage
├── aether-tracker.md          ← Bug investigation
├── aether-chronicler.md       ← Documentation phases
├── aether-keeper.md           ← Knowledge/architecture
├── aether-auditor.md          ← Security/compliance
├── aether-sage.md             ← Analytics
├── aether-measurer.md         ← Performance
├── aether-includer.md         ← Accessibility
├── aether-gatekeeper.md       ← Dependency security
├── aether-ambassador.md       ← Third-party API integration
├── aether-surveyor-disciplines.md
├── aether-surveyor-nest.md
├── aether-surveyor-pathogens.md
└── aether-surveyor-provisions.md
│
│   (GSD agents stay, untouched)
├── gsd-executor.md            ← existing
└── gsd-*.md                   ← existing (10 more)
```

### Modified Files (3 files, low risk)

| File | Change | Risk |
|------|--------|------|
| `package.json` | Add `".claude/agents/"` to files array | Low — additive |
| `bin/cli.js` | Add Claude agents sync block in setupHub() (~10 lines) | Low — additive block |
| `bin/lib/update-transaction.js` | Add Claude agents sync path (~5 lines) | Medium — test sync |

### Unmodified Files

| File | Why Untouched |
|------|---------------|
| `.claude/commands/ant/build.md` | Already uses correct `subagent_type` values |
| All other 33 slash commands | Agent references already correct |
| `.opencode/agents/*.md` | OpenCode pipeline unchanged |
| `.aether/aether-utils.sh` | No agent-specific logic to change |
| `gsd-*.md` agent files | GSD system independent |

---

## Code Changes Required

### 1. `package.json` — Add Claude agents to files array

```json
"files": [
  "bin/",
  ".claude/commands/ant/",
  ".claude/agents/",            ← ADD THIS LINE
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

### 2. `bin/cli.js` — Add Claude agents sync in setupHub()

After line 989 (existing `.opencode/agents/` sync block), insert:

```javascript
// Sync .claude/agents/ -> ~/.aether/system/agents-claude/
const HUB_AGENTS_CLAUDE = path.join(HUB_SYSTEM_DIR, 'agents-claude');
fs.mkdirSync(HUB_AGENTS_CLAUDE, { recursive: true });
const claudeAgentsSrc = path.join(PACKAGE_DIR, '.claude', 'agents');
if (fs.existsSync(claudeAgentsSrc)) {
  const result = syncDirWithCleanup(claudeAgentsSrc, HUB_AGENTS_CLAUDE);
  log(`  Hub agents (claude): ${result.copied} files -> ${HUB_AGENTS_CLAUDE}`);
  if (result.removed.length > 0) {
    log(`  Hub agents (claude): removed ${result.removed.length} stale files`);
    for (const f of result.removed) log(`    - ${f}`);
  }
}
```

### 3. `bin/lib/update-transaction.js` — Sync to `.claude/agents/` in target repos

After line 865 (existing `.opencode/agents/` sync block), insert:

```javascript
// Sync Claude Code agents from hub
const HUB_AGENTS_CLAUDE = path.join(this.HUB_SYSTEM_DIR, 'agents-claude');
const repoClaudeAgents = path.join(this.repoPath, '.claude', 'agents');
if (fs.existsSync(HUB_AGENTS_CLAUDE)) {
  results.agentsClaude = this.syncDirWithCleanup(
    HUB_AGENTS_CLAUDE, repoClaudeAgents, { dryRun }
  );
}
```

Also add `HUB_AGENTS_CLAUDE` to:
- `this.targetDirs` (line 177) for git dirty-file checks
- `verifyIntegrity()` (after line 918) for post-sync verification
- Summary reporting (copy pattern from existing `agents` metrics)

---

## Build Order

Dependencies drive this sequence:

**Step 1 — Create core caste agents (aether-builder, aether-watcher, aether-chaos)**

These are spawned in every single build. Without them, `/ant:build` falls back to general-purpose agents. Do these first and test a build to confirm the `subagent_type` resolution works.

**Step 2 — Create aether-archaeologist and aether-scout**

Used in pre-build archaeology scan and research phases. Needed for complete build workflow.

**Step 3 — Create remaining specialized agents in parallel**

Weaver, probe, tracker, chronicler, keeper, auditor, sage, measurer, includer, gatekeeper, ambassador, route-setter, surveyor variants. These are only invoked for specific phase types and can be written concurrently.

**Step 4 — Add `.claude/agents/` to `package.json` files array**

Do this once core agents exist (Steps 1-2 complete). Shipping an empty or near-empty `.claude/agents/` directory would be harmless but confusing.

**Step 5 — Add sync logic to `bin/cli.js` and `update-transaction.js`**

Distribution infrastructure. Requires Step 4 to be meaningful. Test the full chain: `npm install -g .` → inspect `~/.aether/system/agents-claude/` → `aether update` in a test repo → confirm `.claude/agents/aether-builder.md` exists in target repo.

**Step 6 — Integration test**

Run `/ant:build` on a real phase. Verify agents are resolved by type, not falling back to general-purpose. Confirm JSON output format is correct.

---

## Architectural Patterns

### Pattern 1: Mirror Agents Across Runtimes (Not Replace)

**What:** Maintain parallel agent files — `.opencode/agents/aether-builder.md` and `.claude/agents/aether-builder.md` — with identical role content but different frontmatter.

**When to use:** Always. OpenCode and Claude Code are separate runtimes used in different contexts.

**Trade-offs:** Duplication of role content (roughly 22 files × 2 = 44 files). Mitigation: extract shared role content into a reference document that both agent files point to, or use a generation script that produces both from a single source. The duplication is manageable because role content changes slowly.

### Pattern 2: Inject Context via Prompt, Not Agent File

**What:** The agent file defines the role and discipline. The spawning slash command injects runtime context (goal, task, pheromones, archaeology results, queen wisdom) into the prompt parameter of the Task tool call.

**When to use:** Always. This is already how `build.md` works — the Builder prompt template includes `{ pheromone_section }`, `{ archaeology_context }`, `{ queen_wisdom_section }` injection points.

**Trade-offs:** Agent files stay lightweight and stable. Context injection makes each spawn correctly oriented without requiring the agent to read external files at startup.

### Pattern 3: Separate Hub Paths by Runtime

**What:** Claude Code agents sync to `~/.aether/system/agents-claude/`, OpenCode agents to `~/.aether/system/agents/`. Each maps to its own destination in target repos.

**When to use:** Always. Mixing them in a single hub directory and filtering by name at sync time is fragile.

**Trade-offs:** Requires two sync blocks in setupHub() and UpdateTransaction. This is three extra function calls, not a meaningful complexity increase.

---

## Anti-Patterns

### Anti-Pattern 1: Diverging Role Content Between Runtimes

**What people do:** Write different TDD discipline, different failure modes, different output JSON structure for the same caste in `.opencode/agents/` vs `.claude/agents/`.

**Why it's wrong:** A Builder in Claude Code should behave identically to a Builder in OpenCode. The colony behavior becomes inconsistent and bugs appear only in specific contexts.

**Do this instead:** Keep role content (everything below the frontmatter `---`) identical between the two files. Only the frontmatter (`tools`, `color`) differs.

### Anti-Pattern 2: Distributing GSD Agents to All Repos

**What people do:** Add `.claude/agents/` to `package.json files[]` without filtering — this ships `gsd-executor.md`, `gsd-planner.md` and all other GSD agents to every Aether user's repo.

**Why it's wrong:** GSD agents have different spawn protocols, different output formats, and reference GSD-specific tooling (`gsd-tools.cjs`). They make no sense in a non-GSD project.

**Do this instead:** Either (a) filter by prefix in the sync function — only copy `aether-*.md` files — or (b) move Aether Claude Code agents into a subdirectory like `.claude/agents/ant/` that maps cleanly to a separate hub path. Option (b) is cleaner architecturally. Option (a) is faster to implement.

### Anti-Pattern 3: Mixing Claude Agents Into the OpenCode Hub Path

**What people do:** Reuse `~/.aether/system/agents/` as the hub path for both OpenCode and Claude Code agents.

**Why it's wrong:** UpdateTransaction syncs everything in `agents/` to `.opencode/agents/`. Claude Code agent files landing in `.opencode/agents/` confuse OpenCode, which tries to load them with its own frontmatter parser and tool declarations.

**Do this instead:** Use a distinct hub path (`agents-claude/`) that maps to `.claude/agents/` in target repos. Keep the two registries strictly separated throughout the distribution chain.

### Anti-Pattern 4: Treating the Fallback as Permanent

**What people do:** See the fallback comment in `build.md` ("If Agent type not found, use general-purpose and inject role") and decide the agent files are optional.

**Why it's wrong:** The fallback discards the tool restrictions and context setup that a registered Claude Code agent provides. Registered agents can have `tools` restrictions; general-purpose agents cannot. The fallback is a migration bridge, not an intended operating mode.

**Do this instead:** Create the agent files. The fallback becomes unreachable once all `aether-*.md` files exist.

---

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Current (1 repo, 1 user) | Single hub, direct sync. No changes needed. |
| Multiple repos, 1 user | Hub already handles N repos — this is the designed case. No changes. |
| Multiple users, shared hub | Not a current use case. Hub is per-user at `~/.aether/`. |

The subagent integration does not change the scaling characteristics of the distribution system. Hub sync is local filesystem copy — no network involved.

---

## Integration Points Summary

### New vs Modified vs Unchanged

**New (agent files):**
- 22 files in `.claude/agents/aether-*.md` (matching the 22 OpenCode agents)

**Modified (3 files):**
- `package.json` — add `.claude/agents/` to files array
- `bin/cli.js` — add Claude agents sync block in setupHub()
- `bin/lib/update-transaction.js` — add Claude agents sync and verify paths

**Unchanged (slash commands work as-is):**
- All 34 files in `.claude/commands/ant/` — already reference correct `subagent_type` values
- All OpenCode infrastructure — separate pipeline, untouched
- `.aether/aether-utils.sh` — no agent-specific changes needed
- `gsd-*.md` agent files — independent, untouched

---

## Sources

All findings HIGH confidence — direct source inspection:

- `/Users/callumcowie/repos/Aether/package.json` — files array, confirmed `.claude/agents/` absent
- `/Users/callumcowie/repos/Aether/bin/cli.js` lines 73-79, 956-989, 1172-1179 — hub paths, setupHub(), Claude commands sync
- `/Users/callumcowie/repos/Aether/bin/lib/update-transaction.js` lines 167-177, 861-865 — sync target paths, agents distribution destination
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` lines 392, 567, 765, 843 — `subagent_type` references, fallback comment
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-builder.md` — OpenCode agent frontmatter format
- `/Users/callumcowie/repos/Aether/.claude/agents/gsd-executor.md` — working Claude Code agent frontmatter format (tools, color fields)
- `/Users/callumcowie/repos/Aether/.opencode/agents/aether-queen.md` lines 52-71 — full caste list with subagent_type values

---
*Architecture research for: Claude Code subagent integration with Aether*
*Researched: 2026-02-20*
