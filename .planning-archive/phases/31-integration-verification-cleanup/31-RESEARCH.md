---
phase: 31-integration-verification-cleanup
researched: 2026-02-20
domain: Integration verification, docs curation, bash bug fixing, repo documentation
confidence: HIGH
---

# Phase 31: Integration Verification + Cleanup - Research

**Researched:** 2026-02-20
**Domain:** Agent resolution wiring, bash line wrapping bug, docs audit, README/repo-structure
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Docs curation:**
- Priority: developer reference docs (architecture, known issues, error codes) over user guides
- Audience: the project owner re-orienting after a break — not new contributors
- Cut docs go to `.aether/docs/archive/` (not deleted)
- Claude audits all docs and recommends which 8-10 to keep — no must-keep list locked

**Repo documentation:**
- repo-structure.md lives in repo root (next to README.md)
- High-level overview only — top-level directories with one-line descriptions
- README updated to feature v2.0 agents as a key capability
- README tone: action-oriented — show what commands do, what agents exist, make it feel powerful

**Verification scope:**
- Verify the path from agent return → slash command → COLONY_STATE.json actually works (don't just trust existing code)
- Bash line wrapping bug: Claude investigates, identifies, fixes, and adds a test case
- Claude decides verification depth (wiring-only vs real invocation) and test approach (automated vs manual) based on risk

**Cleanup boundaries:**
- .planning/ directory: keep everything as-is (project history)
- Light tidy only: fix obviously misplaced files or stale state, don't reorganize
- Phase 31 includes marking v2.0 as shipped (update ROADMAP.md + STATE.md)
- v2.0 "done" = version bump to 2.0.0 in package.json + git tag + npm publish

### Claude's Discretion
- Which specific docs survive the 8-10 trim (audit and recommend)
- Verification depth: wiring-only vs one real agent invocation
- Whether INT-02 needs automated tests or manual spot-checks
- Exact structure and content of repo-structure.md
- How to find and fix the bash line wrapping bug

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INT-01 | /ant:build resolves `subagent_type="aether-builder"` to `.claude/agents/ant/aether-builder.md` | Hub sync gap identified — 20 of 22 agents missing from hub; npm install -g . must be run first |
| INT-02 | Agent output format compatible with consuming slash commands (structured JSON returns) | Builder return schema documented; build.md parsing logic verified; format match confirmed |
| INT-03 | Colony state correctly updated after agent runs (COLONY_STATE.json reflects agent work) | State update wiring traced through build.md Step 2; continue.md Step 2.4 advance flow documented |
| CLEAN-01 | .aether/docs/ reduced to 8-10 actively-maintained docs | 14 current files audited; 8 keepers identified; 6 candidates for archive |
| CLEAN-02 | docs/plans/ and .planning/ directories organized | docs/ directory does not exist at root; .planning/ is complete as-is; minimal work required |
| CLEAN-03 | Bash line wrapping bug fixed | Bug located and fully characterized: `bash cmd with description "text"` inside ```bash code blocks; primary offender is swarm.md (13 instances) |
| CLEAN-04 | repo-structure.md created and README updated | Current root structure mapped; README content assessed for v2.0 update scope |
</phase_requirements>

## Summary

Phase 31 is a closing phase for the v2.0 Worker Emergence milestone. The work is in four areas: (1) integration verification that the agent system works end-to-end, (2) a concrete bash bug fix in swarm.md and related command files, (3) docs curation from 14 files down to 8-10, and (4) repo documentation — repo-structure.md and a README update.

The core technical risk is the hub distribution gap: the repo has 22 agents in `.claude/agents/ant/` but the hub only has 2, because `npm install -g .` has not been run since phases 28-30 added 20 new agents. This must be addressed before INT-01 can pass. The fix is straightforward — run `npm install -g .` — but the verification plan must explicitly include this step.

The bash line wrapping bug is real and well-characterized. The old pattern `bash cmd with description "text"` appears inside `\`\`\`bash` code blocks in swarm.md (13 instances), colonize.md (11), entomb.md (11), init.md (7), seal.md (8), and plan.md (7). When Claude Code treats these blocks as literal bash, the `with description "text"` suffix causes a bash error. The fix is to either remove the description suffix (bash tool descriptions belong in the instruction text, not inside the command), or restructure to use the new inline pattern.

**Primary recommendation:** Run `npm install -g .` first to sync all 22 agents to the hub. Then verify INT-01 by confirming the agent resolution path. Fix swarm.md's bash blocks as the primary bug. Audit docs and archive 6 candidates. Write repo-structure.md and update README.

## Standard Stack

### Core (this phase)
| Tool/File | Purpose | Notes |
|-----------|---------|-------|
| `npm install -g .` | Sync all 22 agents to hub | Must run before INT-01 verification |
| `npm pack --dry-run` | Verify package contents | Confirms agent inclusion |
| `bash .aether/aether-utils.sh` | Colony state utilities | Used in verification |
| AVA test framework | Test suite runner | Existing tests used for baseline |
| `.claude/agents/ant/*.md` | Agent definition files | 22 agents, all present in repo |
| `.aether/data/COLONY_STATE.json` | Colony state tracking | INT-03 verification target |

### Affected Command Files (bash bug)
| File | Instances | Priority |
|------|-----------|----------|
| `swarm.md` | 13 | HIGH — most affected, functional bug |
| `colonize.md` | 11 | HIGH — frequently used |
| `entomb.md` | 11 | MEDIUM |
| `seal.md` | 8 | MEDIUM |
| `init.md` | 7 | MEDIUM |
| `plan.md` | 7 | MEDIUM |
| `pause-colony.md` | 1 | LOW |

## Architecture Patterns

### INT-01: Agent Resolution Path

Claude Code resolves `subagent_type="aether-builder"` by looking for an agent file named `aether-builder.md` in the project's `.claude/agents/` directory hierarchy (including subdirectories like `ant/`). The full resolution chain is:

```
Source repo:     .claude/agents/ant/aether-builder.md (22 files)
                        ↓ npm install -g .
Hub:             ~/.aether/system/agents-claude/ (currently only 2 files — gap)
                        ↓ aether update (or npm install syncs directly)
~/.claude/agents/ant/  (only 2 files currently)
                        ↓ Claude Code reads at session start
Task tool:       subagent_type="aether-builder" resolves correctly
```

**Key finding:** The Aether repo itself has all 22 agents in `.claude/agents/ant/`. For verification IN the Aether repo, INT-01 can be tested today (agent files exist). For TARGET REPOS, the hub gap means only 2 agents would resolve. The fix is `npm install -g .` which syncs all 22 to hub and then to `~/.claude/agents/ant/`.

**Verification approach (wiring-only, not live invocation):** Check file exists at resolution path, verify frontmatter is parseable, confirm `name: aether-builder` matches the subagent_type value. A live invocation adds risk (COLONY_STATE.json pollution, resource cost) with minimal additional signal — the Phase 27 verification already proved the mechanism works.

### INT-02: Agent Return Format Compatibility

Builder agent (`aether-builder.md`) returns:

```json
{
  "ant_name": "string",
  "caste": "builder",
  "task_id": "string",
  "status": "completed|failed|blocked",
  "summary": "string",
  "files_created": [],
  "files_modified": [],
  "tests_written": [],
  "tdd": { "cycles_completed": 3, "tests_added": 3, "coverage_percent": 85, "all_passing": true },
  "blockers": []
}
```

Build.md (the consuming slash command) expects at Step 5.2:
- `ant_name` — for display line `"{Ant-Name}: {task_description} ({tool_count} tools) ✓"`
- `tool_count` — reported in display (agent returns this but build.md injects it at prompt level)
- `status` — for wave failure detection
- `files_created`, `files_modified` — passed to watcher and used in synthesis
- `blockers` — used for failure reason extraction

**Finding:** The agent return format matches what build.md expects. The only mismatch is `tool_count` — build.md prompts the builder to include it in the JSON, but the agent's `return_format` section doesn't list it in the schema. This is a documentation gap, not a functional one — build.md explicitly tells the spawned worker to include `tool_count` in its inline prompt.

**Verification approach:** Schema comparison (already done above). No automated test needed — the format is simple enough that code review catches mismatches. Mark INT-02 as VERIFIED by inspection.

### INT-03: Colony State Update Wiring

The state update chain for a successful build:

1. `/ant:build` Step 2: Sets `state = "EXECUTING"`, `current_phase = N`, phase `status = "in_progress"`, writes to COLONY_STATE.json
2. Agent runs and returns JSON
3. `/ant:build` Step 5.9: Synthesizes results but does NOT write state
4. `/ant:continue` Step 2.4: Marks phase `status = "completed"`, all tasks `status = "completed"`, sets `state = "READY"`, writes COLONY_STATE.json

**Key insight:** `/ant:build` does NOT update COLONY_STATE.json with agent outputs. It only sets EXECUTING state. The actual state advancement happens in `/ant:continue`. This is by design — `/ant:build` says explicitly "Build does NOT update task statuses or advance state. Run /ant:continue to mark tasks completed."

**Verification approach:** Trace the code path (done). Verify that after a build, COLONY_STATE.json has `state: "EXECUTING"` and that `/ant:continue` can read and advance from that state. The wiring exists — verification is a code audit, not a live run.

### CLEAN-03: Bash Line Wrapping Bug

**Nature of the bug:** The old instruction format in command files places the Bash tool description as a suffix on the bash command itself, inside a `\`\`\`bash` code block:

```bash
bash .aether/aether-utils.sh swarm-findings-init "<swarm_id>" with description "Initializing swarm findings..."
```

When Claude Code reads this as a bash code block and executes it literally, bash sees `with` as a command and outputs:

```
bash: with: command not found
```

**The new (correct) pattern** places the description in instruction text, not in the command:

```
Run using the Bash tool with description "Initializing swarm findings...": `bash .aether/aether-utils.sh swarm-findings-init "<swarm_id>"`
```

**Files affected and instance counts (verified by code scan):**
- `swarm.md`: 13 instances (highest priority — `/ant:swarm` is commonly used)
- `colonize.md`: 11 instances
- `entomb.md`: 11 instances
- `seal.md`: 8 instances
- `init.md`: 7 instances
- `plan.md`: 7 instances
- `pause-colony.md`: 1 instance

**The long-line variant:** swarm.md line 147 is a 757-character single bash command chaining four `swarm-display-update` calls with `&&`, plus the `with description` suffix. This needs to be broken into separate calls or reformatted.

**Fix strategy:**
1. For most cases: move `with description "text"` out of the bash block into surrounding instruction text
2. For the 757-char chained command: break into separate step with individual commands or use `\` line continuation inside the block
3. Structural fix: the description belongs on the line BEFORE the code block, not inside it

**Test case:** Add an AVA test (or bash test) that scans command files for `bash.*with description` patterns inside `\`\`\`bash` code blocks and fails if any are found. This is a grep-based lint check.

### CLEAN-01: Docs Audit Recommendation

Current docs in `.aether/docs/` (14 files + disciplines/ subdir with 7 files):

| File | Size | Keep/Archive | Reason |
|------|------|--------------|--------|
| `README.md` | 2KB | KEEP | Index — navigation essential |
| `caste-system.md` | 3.9KB | KEEP | Referenced by system; core reference |
| `error-codes.md` | 11.9KB | KEEP | Active E_* constants; developer reference |
| `known-issues.md` | 12KB | KEEP | Active bugs; re-orientation essential |
| `pheromones.md` | 7.4KB | KEEP | User guide; still actively used |
| `QUEEN-SYSTEM.md` | 4.9KB | KEEP | Queen wisdom docs; active system |
| `queen-commands.md` | 3.1KB | KEEP | Command docs; queen-init/read/promote |
| `QUEEN.md` | 2.2KB | KEEP | Generated wisdom file; system-maintained |
| `QUEEN_ANT_ARCHITECTURE.md` | 14.5KB | ARCHIVE | Large design doc; superseded by agent files |
| `implementation-learnings.md` | 3.4KB | ARCHIVE | Historical learnings; not actively referenced |
| `constraints.md` | 3.2KB | ARCHIVE | Constraint definitions; content now in agents |
| `pathogen-schema.md` | 3.9KB | ARCHIVE | Schema docs; only needed if using pathogens actively |
| `pathogen-schema-example.json` | 1KB | ARCHIVE | Example JSON; archive with schema |
| `progressive-disclosure.md` | 3.8KB | ARCHIVE | Design philosophy; not referenced by active code |
| `disciplines/` (subdir, 7 files) | — | KEEP as-is | Referenced by continue.md; worker training |

**Result:** 8 keepers + disciplines/ subdir. This achieves the 8-10 target with room for one additional keeper if the audit reveals a doc that must stay.

**Archive path:** `.aether/docs/archive/` (create this directory)

**What "archive" means:** Move files there, update README.md to note they exist in archive. Nothing is deleted.

### CLEAN-04: Repo Structure and README

**Current repo root (top-level directories and key files):**
```
Aether/
├── bin/            — CLI tools: cli.js, validate-package.sh, lib/
├── src/            — Source (currently thin — main logic in bin/)
├── tests/          — AVA unit tests + bash integration tests
├── runtime/        — Legacy staging (check if this still exists)
├── node_modules/   — npm dependencies (gitignored)
├── .aether/        — Colony system: workers.md, aether-utils.sh, docs/, utils/, templates/
├── .claude/        — Claude Code: commands/ant/, agents/ant/, rules/
├── .opencode/      — OpenCode: commands/ant/, agents/
├── .planning/      — Development roadmap, phases, requirements (not distributed)
├── .github/        — CI configuration
├── README.md       — Project overview
├── CLAUDE.md       — Claude Code instructions
├── package.json    — npm package config (version 4.0.0 → 2.0.0)
├── CHANGELOG.md    — Version history
├── TO-DOS.md       — Development backlog
└── LICENSE, DISCLAIMER.md
```

**repo-structure.md content approach:** One-line descriptions, no deep nesting, focus on what each top-level directory is for. Audience is the owner coming back after a break.

**README v2.0 update scope:**
- Current README mentions "23 Specialized Agents" — update count/framing to reflect v2.0 status
- Add `/ant:build` example that explicitly calls out agent spawning by name
- Show what `/agents` looks like with aether agents loaded
- Tone: "run this command, this agent appears, here's what it does" — concrete and powerful
- Version badge: update to reflect 2.0.0 milestone

**Version bump note:** The CONTEXT.md specifies "version bump to 2.0.0 in package.json". The current package.json shows `"version": "4.0.0"`. This would be a version DOWNGRADE numerically. The last published npm version is 3.1.17. The user explicitly chose "2.0.0" to align the npm package with the milestone naming ("v2.0 Worker Emergence"). This is an unusual choice but was explicitly decided — the planner should follow it without second-guessing. The git tag will be `v2.0.0`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead |
|---------|-------------|-------------|
| Agent file linting | Custom parser | AVA test with YAML parsing (already exists in agent-quality.test.js) |
| Hub sync verification | Custom sync code | `npm install -g .` + `ls ~/.aether/system/agents-claude/` |
| Package contents check | File-counting scripts | `npm pack --dry-run` |
| Colony state validation | Direct JSON parsing | `bash .aether/aether-utils.sh validate-state colony` |

## Common Pitfalls

### Pitfall 1: Verifying Agents Without Syncing Hub First
**What goes wrong:** Testing agent resolution against hub that only has 2 of 22 agents. INT-01 passes for builder/watcher but would fail for queen, scout, and every Phase 28-30 agent.
**How to avoid:** Run `npm install -g .` BEFORE any hub-dependent verification. Confirm `~/.aether/system/agents-claude/` has 22 files.
**Warning signs:** `ls ~/.aether/system/agents-claude/ | wc -l` returns 2.

### Pitfall 2: Treating CLEAN-02 as Substantial Work
**What goes wrong:** The requirement says "docs/plans/ and .planning/ organized" but `docs/` doesn't exist at root, and .planning/ is intentionally preserved as-is (project history). Spending time reorganizing creates busywork.
**How to avoid:** CLEAN-02 is satisfied by: confirming docs/ doesn't exist (it doesn't), confirming .planning/ is complete and organized (it is). Record this as verified-by-inspection.
**Warning signs:** Starting to move .planning/ phase directories around.

### Pitfall 3: Version Bump Confusion
**What goes wrong:** package.json says 4.0.0, CONTEXT.md says bump to 2.0.0. A planner might interpret this as "bump to 5.0.0" or get confused and skip the version bump.
**How to avoid:** Follow CONTEXT.md literally — set `"version": "2.0.0"` in package.json. This aligns the npm version with the milestone naming. It's unconventional but it's the user's explicit decision.
**Warning signs:** Bumping to 5.0.0 or questioning the decision in the plan.

### Pitfall 4: Fixing the Wrong Bash Bug
**What goes wrong:** Claude spots "long lines" and tries to add line continuations with `\` everywhere. The actual bug is `with description "text"` appearing as a bash command suffix INSIDE code blocks — the description is the bug, not the line length.
**How to avoid:** The fix is structural: move description text OUT of bash blocks, INTO instruction prose. Don't just shorten lines with `\`.
**Warning signs:** Adding `\` line continuation to 300+ char commands inside code blocks.

### Pitfall 5: Treating INT-02 as Needing a Live Invocation
**What goes wrong:** Running `/ant:build` to test INT-02 modifies COLONY_STATE.json, creates checkpoints, spawns workers — all side effects. The format compatibility is static and can be verified by schema comparison.
**How to avoid:** INT-02 is verified by reading the agent's `return_format` section and comparing it to what build.md's Step 5.2 expects. No live invocation needed.
**Warning signs:** Proposing to run /ant:build as part of INT-02 verification.

## Code Examples

### Hub Sync Verification Pattern
```bash
# Source: Phase 27 verification (27-VERIFICATION.md)
# Run before INT-01 verification to ensure hub is current
npm install -g .
ls ~/.aether/system/agents-claude/ | wc -l  # Should be 22
ls ~/.claude/agents/ant/ | wc -l            # Should be 22 (synced by install)
```

### Bash Bug Detection Pattern (for test case)
```javascript
// Source: pattern derived from existing agent-quality.test.js structure
// Add to tests/unit/ or tests/bash/
// Scan command files for 'with description' inside bash code blocks

const COMMANDS_DIR = path.join(__dirname, '../../.claude/commands/ant');

test('CLEAN-03: no bash commands contain "with description" suffix', t => {
  const files = fs.readdirSync(COMMANDS_DIR).filter(f => f.endsWith('.md'));

  for (const file of files) {
    const content = fs.readFileSync(path.join(COMMANDS_DIR, file), 'utf8');
    const lines = content.split('\n');
    let inBashBlock = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line.startsWith('```')) inBashBlock = !inBashBlock;
      if (inBashBlock && /^bash\s+.*with description/.test(line.trim())) {
        t.fail(`${file} line ${i+1}: bash command contains "with description" suffix (bash wrapping bug)`);
      }
    }
  }
  t.pass();
});
```

### Correct vs Buggy Bash Instruction Pattern
```markdown
# WRONG (causes "with: command not found" in bash):
\`\`\`bash
bash .aether/aether-utils.sh swarm-findings-init "<swarm_id>" with description "Initializing..."
\`\`\`

# CORRECT (description in prose, command clean):
Run using the Bash tool with description "Initializing swarm findings...":
\`\`\`bash
bash .aether/aether-utils.sh swarm-findings-init "<swarm_id>"
\`\`\`
```

### COLONY_STATE.json State Flow Verification
```bash
# Source: build.md Step 2, continue.md Step 2.4
# After /ant:build:
jq '.state' .aether/data/COLONY_STATE.json  # Should be "EXECUTING"
# After /ant:continue:
jq '.state' .aether/data/COLONY_STATE.json  # Should be "READY"
jq '.plan.phases[0].status' .aether/data/COLONY_STATE.json  # Should be "completed"
```

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| `bash cmd with description "text"` inside code blocks | `Run using the Bash tool with description "text": \`bash cmd\`` | Old pattern causes bash errors; new pattern is instruction prose + clean command |
| 22 agents in OpenCode only | 22 agents in both `.claude/agents/ant/` and `.opencode/agents/` | Claude Code can now resolve agents by subagent_type |
| npm version misaligned with milestone | npm 2.0.0 aligned with "v2.0 Worker Emergence" milestone | External version signal matches internal milestone naming |

**Deprecated/outdated:**
- `with description "text"` as bash command suffix: replaced by instruction-level description pattern
- Hub having only 2 agents: replaced by full 22-agent sync via `npm install -g .`

## Open Questions

1. **Version bump interpretation**
   - What we know: CONTEXT.md says "version bump to 2.0.0 in package.json". Current version is 4.0.0. Last published is 3.1.17.
   - What's unclear: Did the user intend 2.0.0 (aligning npm with milestone names) or meant "next major version" (which would be 5.0.0)?
   - Recommendation: Follow CONTEXT.md literally — set 2.0.0. It was an explicit decision in the discussion phase. If wrong, it's easily corrected.

2. **Disciplines subdir counting toward 8-10 target**
   - What we know: CLEAN-01 says "8-10 actively-maintained documents". Disciplines has 7 files in a subdir.
   - What's unclear: Does the 8-10 count include disciplines/ files?
   - Recommendation: Treat disciplines/ as its own unit (keep whole subdir), count 8 root-level docs as satisfying "8-10". The audience re-orientation use case cares about root-level docs; disciplines are worker training docs referenced by the system.

3. **Swarm.md 757-char chained command refactoring**
   - What we know: Line 147 chains 4 swarm-display-update calls with `&&` into one line, plus the description suffix.
   - What's unclear: Should these be broken into 4 separate `Run using the Bash tool` instructions or kept chained?
   - Recommendation: Break into separate calls. Each `swarm-display-update` has a distinct purpose; separate instructions are more readable and individually describable.

## Sources

### Primary (HIGH confidence)
- Direct code inspection of `.claude/commands/ant/*.md` — bash wrapping bug locations
- Direct inspection of `.claude/agents/ant/` — 22 agents present
- `ls ~/.aether/system/agents-claude/` — hub gap confirmed (2 of 22)
- `ls ~/.claude/agents/ant/` — hub delivery confirmed (2 of 22)
- Phase 27 VERIFICATION.md — resolution mechanism proven, format contracts established
- `agent-quality.test.js` — existing test patterns for new lint check

### Secondary (MEDIUM confidence)
- CONTEXT.md decisions — user locked decisions from /gsd:discuss-phase
- REQUIREMENTS.md CLEAN-03 analysis and v1.4-REQUIREMENTS.md bug description
- Planning phase 26 VERIFICATION.md — detailed CLEAN-08 gap analysis

### Tertiary (LOW confidence)
- Version bump interpretation — user said "2.0.0" but current is 4.0.0; assuming literal intent

## Metadata

**Confidence breakdown:**
- INT-01 (agent resolution): HIGH — file system verified, Phase 27 mechanism proven; hub gap confirmed and fixable
- INT-02 (format compatibility): HIGH — schema comparison done; build.md parsing logic read
- INT-03 (state update wiring): HIGH — code path traced through build.md and continue.md
- CLEAN-03 (bash bug): HIGH — bug locations confirmed by code scan; fix pattern established
- CLEAN-01 (docs audit): MEDIUM — recommendation made based on file inspection; user may prefer different 8
- CLEAN-04 (repo-structure/README): MEDIUM — content plan established; exact wording discretionary
- Version bump: LOW — literal interpretation of user decision; see Open Questions

**Research date:** 2026-02-20
**Valid until:** 30 days (stable codebase, changes unlikely)
