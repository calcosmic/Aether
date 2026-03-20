# Phase 23: Agent Resilience - Research

**Researched:** 2026-02-19
**Domain:** LLM agent definition authoring — failure modes, success criteria, safety boundaries in markdown
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Failure behavior:**
- Agents should try to fix problems autonomously first (2 attempts max)
- If recovery fails, escalate by presenting 2-3 concrete options with trade-offs — let the user choose
- Tiered by severity: minor issues (missing file, command fail) → retry silently; major issues (state corruption, data loss risk) → stop immediately and escalate
- No silent failures — if an agent gives up, it explains what happened

**Success signals:**
- Agents report what they produced/changed AND confirm they verified their own work ("Created 3 files, ran validation, all checks pass")
- Success criteria are agent-specific — each agent defines what "done right" means for its role (builder checks code works, watcher checks tests pass, scout checks sources found)
- High-stakes agents (builder, queen, watcher) get peer review from another agent; lower-risk agents self-verify only
- If self-check fails, agent retries automatically (within 2-attempt limit) before escalating

**Safety boundaries:**
- Read-only declarations are advisory — written into agent definitions as rules the LLM reads and respects
- Tiered boundary approach (Claude's discretion on specifics):
  - Globally protected paths (colony state, user data, dreams, checkpoints)
  - Per-agent boundaries based on what each agent's role actually needs to touch

**Prioritization:**
- Tier agents by risk level based on what they can modify (Claude classifies)
- High-risk agents (those that modify files, state, git) → detailed failure modes, strict boundaries, peer review
- Low-risk agents (read-only roles like chronicler, sage) → lighter treatment
- Format: XML tags (`<failure_modes>`, `<success_criteria>`, `<read_only>`) — consistent with Aether's XML convention for LLM-readable structure

**Scope:**
- Both OpenCode agents (`.opencode/agents/`) AND Claude Code slash commands (`.claude/commands/ant/`)
- Matches existing post-Phase-22 cleanup format with new XML-tagged sections added

### Claude's Discretion
- Exact risk tier classification for all 25 agents
- Which specific paths go in each agent's read-only list
- How safety violations are handled (severity-based response)
- Which agents need peer review vs self-verify
- Protected path recommendations beyond current set (colony state, user data, .env)

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| RESIL-01 | Add failure modes to agent definitions | Risk tier classification, failure behavior patterns, XML format template |
| RESIL-02 | Add success criteria to agent definitions | Agent-specific success signals, self-check patterns, peer review classification |
| RESIL-03 | Add read-only declarations to agent definitions | Global protected paths, per-agent boundary mapping, safety violation response |
</phase_requirements>

## Summary

Phase 23 adds structured resilience sections to all 25 agent definitions and 33 slash commands. The work is purely editorial — writing markdown that LLM agents read and follow. There is no runtime enforcement layer. Each agent gets three XML-tagged sections appended to its definition: `<failure_modes>` (how to handle failures), `<success_criteria>` (what done looks like), and `<read_only>` (paths not to touch).

The critical research finding is that agents already fall into clear risk tiers based on their role. Read-only investigators (archaeologist, chaos, sage, measurer, includer, gatekeeper) should have the tightest `<read_only>` declarations since their value proposition depends on not modifying anything. File-writing agents (builder, weaver, chronicler) need failure modes that explicitly address partial-write scenarios. Orchestrators (queen, route-setter) need failure modes that cover spawn failures and state corruption.

Slash commands require a different approach: they already contain complex step-by-step workflows with their own error handling. Adding resilience sections there means identifying the one or two critical failure points per command rather than enumerating every edge case.

**Primary recommendation:** Tier all 25 agents into HIGH/MEDIUM/LOW risk, write XML sections using the templates below, and focus the most detailed treatment on the 7 high-risk agents (queen, builder, watcher, weaver, route-setter, and the two stateful slash commands: init and build).

## Standard Stack

### Core (what we are authoring into)

| Agent Type | Count | Format | Post-Phase-22 State |
|------------|-------|--------|---------------------|
| OpenCode agents | 24 files | YAML frontmatter + markdown | Clean — boilerplate stripped |
| workers.md | 1 file | Pure markdown reference | Clean |
| Claude Code slash commands | 33 files | YAML frontmatter + markdown | Mix of XML-structured and prose |

**No libraries.** This phase touches only markdown files. The "stack" is the existing XML convention already present in surveyor agents and the build command.

### Existing XML Patterns in Codebase

The surveyor agents (`aether-surveyor-nest.md`, `aether-surveyor-disciplines.md`, `aether-surveyor-pathogens.md`, `aether-surveyor-provisions.md`) already use XML-tagged sections:

```xml
<role>...</role>
<consumption>...</consumption>
<philosophy>...</philosophy>
<process>
  <step name="...">...</step>
</process>
<critical_rules>...</critical_rules>
<success_criteria>...</success_criteria>
```

This is the established pattern. New sections follow the same convention.

## Architecture Patterns

### Agent Risk Tier Classification

After reading all 24 agents, here is the evidence-based tier classification:

**HIGH RISK — Can modify files, state, git, or coordinate spawns:**

| Agent | Risk Basis | Peer Review | Failure Depth |
|-------|-----------|-------------|---------------|
| `aether-queen` | Writes COLONY_STATE.json, spawns workers, drives git checkpoints | Watcher spawned by build command | Deep — state corruption catastrophic |
| `aether-builder` | Creates/modifies production files, writes tests | Watcher (mandatory) | Deep — partial writes leave broken state |
| `aether-watcher` | Creates flags (persistent blockers) via `flag-add` | Self-verify with Queen escalation | Deep — false negatives block phases |
| `aether-weaver` | Refactors existing files, behavior-preserving changes | Self-verify (tests must pass) | Deep — behavior regression possible |
| `aether-route-setter` | Writes phase plans into COLONY_STATE.json | Self-verify | Medium — bad plan corrupts roadmap |
| `aether-ambassador` | Writes integration code, handles API keys | Self-verify | Medium — auth/secrets sensitive |
| `aether-tracker` | Can apply fixes (Step 5 in its workflow) | Self-verify | Medium — fix may introduce regression |

**MEDIUM RISK — Writes documentation or analysis output files:**

| Agent | Risk Basis | Peer Review |
|-------|-----------|-------------|
| `aether-chronicler` | Writes README, docs, API docs | Self-verify |
| `aether-probe` | Writes test files | Self-verify |
| `aether-architect` | Writes synthesis documents | Self-verify |
| `aether-keeper` | Archives patterns to `.aether/data/` area | Self-verify |

**LOW RISK — Read-only investigators:**

| Agent | Read-Only Basis |
|-------|----------------|
| `aether-archaeologist` | Explicitly states "You NEVER modify code. You NEVER modify colony state." |
| `aether-chaos` | Explicitly states "You NEVER modify code. You NEVER fix what you find." |
| `aether-scout` | Research and reporting only |
| `aether-sage` | Analytics only, reads history |
| `aether-auditor` | Reviews, does not fix |
| `aether-guardian` | Scans for vulnerabilities, does not patch |
| `aether-measurer` | Benchmarks, does not optimize |
| `aether-includer` | Audits accessibility, does not fix |
| `aether-gatekeeper` | Audits dependencies, does not update |
| `aether-surveyor-nest` | Writes to `.aether/data/survey/` only |
| `aether-surveyor-disciplines` | Writes to `.aether/data/survey/` only |
| `aether-surveyor-pathogens` | Writes to `.aether/data/survey/` only |
| `aether-surveyor-provisions` | Writes to `.aether/data/survey/` only |

### Globally Protected Paths (All Agents)

Sourced from `.claude/rules/security.md` and CLAUDE.md:

```
.aether/data/COLONY_STATE.json    — Colony state (precious)
.aether/data/constraints.json     — Pheromone signals
.aether/data/flags.json           — Blockers (only watcher may create via flag-add)
.aether/data/checkpoints/         — Session checkpoints
.aether/locks/                    — File locks
.aether/dreams/                   — Dream journal (user notes)
.env*                             — Environment secrets
.claude/settings.json             — Hook configuration
.github/workflows/                — CI configuration
```

Surveyor agents write to `.aether/data/survey/` — this is their designated output area, not a protected path for them.

### XML Section Templates

**Template for HIGH-RISK agent `<failure_modes>`:**

```xml
<failure_modes>
## Failure Modes

### Minor Failures (retry silently, max 2 attempts)
- Missing expected file → re-read directory, try alternate path
- Command exits non-zero → read error output, diagnose, retry with fix
- Tool call returns unexpected format → re-attempt with explicit format request

### Major Failures (stop immediately, escalate)
- State corruption detected (malformed JSON in colony state) → STOP, do not write, escalate
- Data loss risk (destructive operation on protected path) → STOP, explain what was about to happen
- 2 retry attempts exhausted on any single failure → STOP, escalate with options

### Escalation Format
When escalating, present:
1. What failed and what was attempted
2. 2-3 concrete options the user can choose from, each with trade-offs
3. Recommended option with rationale

Never fail silently. If giving up, say what happened.
</failure_modes>
```

**Template for LOW-RISK agent `<failure_modes>`:**

```xml
<failure_modes>
## Failure Modes

### Minor Failures (retry once)
- File not found → try alternate path or glob pattern
- Command returns no results → broaden search, note what was tried

### Escalation
If after 2 attempts a search yields nothing useful, report honestly:
- What was searched
- What was found (or not found)
- Recommended next step for the colony

Never fabricate findings. "Nothing found" is a valid and useful result.
</failure_modes>
```

**Template for HIGH-RISK agent `<success_criteria>`:**

```xml
<success_criteria>
## Success Criteria

### Self-Check (always required)
Before reporting complete:
1. Verify every file you created/modified exists and is readable
2. Run the project's verification command (build/test/lint)
3. Confirm the specific outcome matches the task description

### Report Format
Your completion report MUST include:
- What you produced (files created, modified, commands run)
- Evidence you verified your work (command output, test results)
- Any issues found during self-check and how they were resolved

### Peer Review Trigger
This agent's work is reviewed by Watcher after completion. Your output is not final until Watcher approves. If Watcher requests fixes, re-attempt within the 2-attempt limit.

Example: "Created 3 files, ran `npm test` (24 passed, 0 failed), all success criteria met."
</success_criteria>
```

**Template for LOW-RISK agent `<success_criteria>`:**

```xml
<success_criteria>
## Success Criteria

### Self-Check
Before reporting complete:
1. Confirm you found what was asked (or documented what was not found)
2. Verify your output is complete and accurate

### Report Format
Your completion report MUST include:
- What you found/produced
- Key sources or evidence
- Confidence level on your findings

Example: "Investigated 12 files, identified 3 stability patterns, 2 areas of concern. All findings have file+line citations."
</success_criteria>
```

**Template for HIGH-RISK agent `<read_only>` (builder example):**

```xml
<read_only>
## Protected Paths

You MUST NOT modify these paths under any circumstances:

### Globally Protected (all agents)
- `.aether/data/COLONY_STATE.json` — Colony state, written only by Queen/build command
- `.aether/data/constraints.json` — Pheromone signals
- `.aether/data/flags.json` — Blocker registry
- `.aether/locks/` — File locks
- `.aether/dreams/` — User notes
- `.env*` — Secrets
- `.claude/settings.json` — Hook config
- `.github/workflows/` — CI config

### Builder-Specific Boundaries
- Do not modify other agents' output files (leave watcher, chaos output untouched)
- Do not modify `.aether/aether-utils.sh` unless task explicitly targets it
- Do not delete files — create or modify only

### If You Encounter a Protected Path
Stop. Report what you were about to do and why. Present the user with options.
</read_only>
```

**Template for LOW-RISK (truly read-only) agent `<read_only>`:**

```xml
<read_only>
## Protected Paths

You are a read-only agent. You MUST NOT write or modify any files.

### Only Permitted Write Operations
None. You investigate and report only.

### Globally Protected (for reference)
- `.aether/data/` — Colony state
- `.aether/dreams/` — User notes
- `.env*` — Secrets

### If Asked to Modify Files
Refuse. Explain your role is investigation only. Suggest which agent should handle the modification.
</read_only>
```

**Template for Surveyor agents (limited write scope) `<read_only>`:**

```xml
<read_only>
## Protected Paths

You may ONLY write to `.aether/data/survey/`. All other paths are read-only.

### Permitted Write Locations
- `.aether/data/survey/BLUEPRINT.md`
- `.aether/data/survey/CHAMBERS.md`
(or whichever survey documents this agent produces)

### Globally Protected
- `.aether/data/COLONY_STATE.json` — Never touch
- `.aether/data/constraints.json` — Never touch
- `.aether/dreams/` — Never touch
- `.env*` — Never touch

### If a Task Would Require Writing Outside Survey Directory
Stop and escalate. That is outside your designated scope.
</read_only>
```

### Slash Command Treatment

The 33 slash commands (`/ant:build`, `/ant:init`, `/ant:swarm`, etc.) are different from agent files. They already contain detailed, step-by-step workflows with their own error handling (wave failures, verification failures, stale session checks). Adding full resilience XML to every command would be noise.

**Recommended approach for slash commands:**
- Add a brief `<failure_modes>` block near the top of commands that write persistent state (init, build, seal, entomb, lay-eggs)
- Skip or add minimal treatment for read-only/display commands (status, watch, flags, history, help)
- Focus on the one or two catastrophic failure points per command, not enumeration of every error

**High-priority slash commands for resilience treatment:**

| Command | Risk | Key Failure Mode to Document |
|---------|------|------------------------------|
| `init.md` | HIGH | Colony state overwrite with no backup |
| `build.md` | HIGH | Wave failure leaves files half-written |
| `lay-eggs.md` | HIGH | Plan writing fails mid-write |
| `seal.md` | HIGH | Archive operation corrupts source |
| `entomb.md` | HIGH | Archival with no rollback |
| `colonize.md` | MEDIUM | Survey writes over existing survey |

**Low-priority slash commands (read-only or display):**

`status.md`, `watch.md`, `flags.md`, `history.md`, `help.md`, `tunnels.md`, `maturity.md`, `archaeology.md`, `oracle.md`, `resume.md`, `pause-colony.md`, `resume-colony.md`, `flag.md`, `feedback.md`, `focus.md`, `redirect.md`, `verify-castes.md`, `dream.md`, `interpret.md`, `update.md`, `migrate-state.md`, `organize.md`, `council.md`, `phase.md`, `plan.md`, `continue.md`

### Placement in Agent Files

Resilience sections go at the END of each agent definition, after the existing Output Format section. This preserves the clean top-to-bottom readability: role → workflow → output format → resilience.

```
[existing agent content]
...
## Output Format
[existing JSON format]

<failure_modes>
...
</failure_modes>

<success_criteria>
...
</success_criteria>

<read_only>
...
</read_only>
```

**Exception:** Surveyor agents already have `<success_criteria>` at the end. These should be updated in-place rather than appended.

### Per-Agent Boundary Details

**Builder** — Can write to any project file. Must not touch:
- `.aether/data/COLONY_STATE.json`, `.aether/data/constraints.json`, `.aether/data/flags.json`
- `.aether/aether-utils.sh` (unless task explicitly targets it)
- `.aether/dreams/`, `.env*`, `.claude/settings.json`, `.github/workflows/`

**Watcher** — Reads files, runs commands, creates flags via `flag-add` only. Must not:
- Edit any source files (that's builder's job)
- Write to COLONY_STATE.json directly
- Delete files

**Queen** — Writes COLONY_STATE.json via aether-utils.sh, creates HANDOFF.md, updates context. Must not:
- Write to `.aether/dreams/`
- Write directly to `.env*`
- Run destructive git operations (reset --hard, push --force)

**Weaver** — Modifies existing source files, refactors. Same boundaries as builder plus:
- Must not change test expectations without changing implementation

**Route-Setter** — Writes phase plans into colony state via aether-utils.sh. Must not:
- Directly edit COLONY_STATE.json (use aether-utils.sh commands only)
- Modify existing code

**Ambassador** — Writes integration code and documentation. Must not:
- Write secrets/API keys to tracked files
- Modify `.env*` (document what env vars are needed, don't set them)

**Tracker** — Investigates and applies fixes. Essentially high-risk builder with debugging focus:
- Same boundaries as builder
- The 3-Fix Rule already in its definition is a natural failure mode — reference it in `<failure_modes>`

**Chronicler** — Writes to docs/ and README. Must not:
- Touch `.aether/data/`
- Modify source code (only documentation)

**Probe** — Writes test files. Must not:
- Modify source code (only test files)
- Delete existing tests

**Archaeologist** — Already explicitly read-only in definition. Reinforce:
- No writes anywhere
- No colony state access beyond reading

**Chaos** — Already explicitly read-only in definition. Same as archaeologist.

**Scout, Sage, Auditor, Guardian, Measurer, Includer, Gatekeeper** — All read-only investigators. No writes permitted.

**Architect, Keeper** — Write to analysis/pattern files. Must not touch colony state or source code.

**Surveyor agents** — Write to `.aether/data/survey/` only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Enforcement layer | Runtime path-checking code | Advisory text in agent definitions | The decision is locked: read-only is advisory only |
| New XML tag schema | Custom tags beyond the 3 decided | `<failure_modes>`, `<success_criteria>`, `<read_only>` | Consistency with context decision |
| Per-command success criteria for all 33 slash commands | Full XML treatment on every command | Selective treatment on high-risk commands only | Noise vs signal; commands already have detailed workflows |

## Common Pitfalls

### Pitfall 1: Over-specifying failure modes for read-only agents
**What goes wrong:** Writing exhaustive failure mode sections for agents like Archaeologist that cannot affect system state. Results in long boilerplate that obscures the truly important rule ("don't write anything").
**Why it happens:** Treating all agents the same regardless of risk tier.
**How to avoid:** LOW-risk agents get 3-4 sentences max. HIGH-risk agents get detailed treatment.
**Warning signs:** Failure mode section is longer than the agent's main workflow.

### Pitfall 2: Contradicting existing behavior in agent definitions
**What goes wrong:** Writing `<failure_modes>` that say "retry up to 3 times" when the builder already has a 3-Fix Rule that triggers architectural escalation at 3 failures. Creates confusing contradictions.
**Why it happens:** Not reading each agent's existing content before writing additions.
**How to avoid:** Read every agent's current content before writing its resilience sections. Reference existing rules (like the 3-Fix Rule) rather than restating them differently.
**Warning signs:** New section contradicts language in the existing agent body.

### Pitfall 3: Generic protected path lists that don't match agent role
**What goes wrong:** Telling the Chronicler not to touch `.aether/data/COLONY_STATE.json` is obvious; telling it not to modify source code is more useful context. Generic lists waste space.
**Why it happens:** Copy-pasting the global protected paths without adding agent-specific context.
**How to avoid:** Global paths are always included, but per-agent boundary section adds the agent-specific nuance that matters.

### Pitfall 4: Success criteria that duplicate the output format
**What goes wrong:** Agent already has a JSON output format. Writing success criteria that just restate the output schema fields.
**Why it happens:** Conflating "what to return" with "how to verify you're done."
**How to avoid:** Success criteria focus on verification actions taken before returning, not the structure of what is returned.

### Pitfall 5: Forgetting slash commands have a different format
**What goes wrong:** Applying agent-style resilience sections to slash command files that already have step-by-step workflows. The result is section duplication.
**Why it happens:** Treating slash commands like agents.
**How to avoid:** Slash commands get a lighter touch — a brief resilience block addressing the command's specific catastrophic failure point, not full section treatment.

## Code Examples

### High-Risk Agent — Builder Complete Addition

```markdown
<failure_modes>
## Failure Modes

### Minor Failures (retry silently, max 2 attempts)
- File not found → re-read directory listing, try alternate path
- Command exits non-zero → read full error output, diagnose root cause, retry once with fix
- Test fails unexpectedly → check if test file is new (may need dependency setup), retry

### Major Failures (stop and escalate immediately)
- Protected path detected in write target → STOP, do not write, escalate with explanation
- State corruption risk (malformed COLONY_STATE.json detected) → STOP, do not modify
- 2 retry attempts exhausted on any single task → STOP, escalate

### Escalation Format
Present to the user:
1. What failed and what was attempted (including both retry attempts)
2. 2-3 concrete options with trade-offs:
   - Option A: [specific action] — [trade-off]
   - Option B: [specific action] — [trade-off]
   - Option C: Skip this task — [impact]
3. Your recommendation with rationale

Never fail silently. If a task cannot be completed, say so explicitly.
</failure_modes>

<success_criteria>
## Success Criteria

### Self-Check (required before reporting complete)
1. Every file you created or modified exists and is readable
2. Run: `npm test` (or equivalent for this project) — all tests pass
3. Run: `npm run build` (or equivalent) — no compilation errors
4. The specific deliverable described in your task exists and matches the description

### Peer Review
Your work is reviewed independently by a Watcher. The build is not complete until Watcher approves. If Watcher finds issues, address them within the 2-attempt limit.

### Completion Report Must Include
- Files created: [list with paths]
- Files modified: [list with paths]
- Verification run: [command and result]
- Example: "Created 2 files, modified 1, ran `npm test` (18 passed), all criteria met."
</success_criteria>

<read_only>
## Protected Paths

### Never Modify (Global)
- `.aether/data/COLONY_STATE.json` — Written only by aether-utils.sh commands
- `.aether/data/constraints.json` — Pheromone signals (Queen manages these)
- `.aether/data/flags.json` — Blocker registry (Watcher manages via flag-add)
- `.aether/locks/` — File lock directory
- `.aether/dreams/` — User session notes
- `.env*` — Environment secrets
- `.claude/settings.json` — Hook configuration
- `.github/workflows/` — CI configuration

### Builder-Specific Boundaries
- Do not modify `.aether/aether-utils.sh` unless your task explicitly targets it
- Do not delete files (create and modify only)
- Do not modify files belonging to another agent's output (watcher reports, chaos findings)

### If You Detect a Protected Path in Your Target
Stop immediately. Report: "Task would require writing to [protected path]. This is outside my permitted scope." Present options to the user.
</read_only>
```

### Low-Risk Agent — Archaeologist Complete Addition

```markdown
<failure_modes>
## Failure Modes

### Minor Failures (retry once)
- git command not available → note in output, proceed with file-based analysis only
- File path not found → search with glob for similar paths, note what was tried

### Escalation
After 2 attempts, if investigation yields insufficient findings, report honestly:
- What was searched and examined
- What was and was not found
- Recommended next steps for the colony

Never fabricate findings. An honest "insufficient evidence" is more valuable than speculative conclusions.
</failure_modes>

<success_criteria>
## Success Criteria

### Self-Check
Before reporting complete:
1. Every finding cites a specific file, line, or commit
2. Stability map covers all files in scope (even if result is "stable")
3. Output format matches the JSON schema below

### Completion Report Must Include
Evidence for all claims. Example: "Excavated 8 files, 47 commits reviewed. 3 stability findings, 2 workarounds identified (see tribal_knowledge)."
</success_criteria>

<read_only>
## Protected Paths

You are a strictly read-only agent. You investigate and report only.

### No Writes Permitted
Do not create, modify, or delete any files. Do not update colony state.

### If Asked to Modify Something
Refuse. Explain your role is investigation only. Suggest the appropriate agent for modifications (Builder for code, Chronicler for docs, Queen for colony state).
</read_only>
```

### Slash Command — Brief Resilience Block for `init.md`

```markdown
<resilience>
## Critical Failure Points

### Colony State Overwrite Risk
If COLONY_STATE.json already exists and contains a valid active colony:
- Stop before overwriting
- Warn: "Active colony detected with goal: [goal]. Overwriting will lose this data."
- Present options: (1) Archive first with /ant:seal, (2) Continue and overwrite, (3) Cancel

### Write Failure Mid-Init
If writing COLONY_STATE.json fails partway through:
- Do not leave a partial state file
- Remove the incomplete file
- Report the failure with the original error

### Recovery
If init leaves system in broken state, user can delete `.aether/data/COLONY_STATE.json` and run init again.
</resilience>
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Agents with no explicit failure handling | Agents with `<failure_modes>` XML section | Phase 23 | LLM has explicit instruction for failure paths |
| Generic "proceed" success signals | Agent-specific success criteria with verification steps | Phase 23 | Agents know what "done" means for their role |
| Implicit safety assumptions | Explicit `<read_only>` declarations | Phase 23 | Protected paths documented in agent context |
| Surveyor agents had ad-hoc `<success_criteria>` | Consistent pattern across all agents | Phase 23 | Uniformity enables pattern recognition by LLM |

**What already exists and must be preserved:**
- Surveyor agents have `<success_criteria>` — update these, do not duplicate
- Builder has "3-Fix Rule" and "The Iron Law" — new failure_modes must reference, not replace these
- Watcher has "The Watcher's Iron Law" — same treatment
- Archaeologist and Chaos have explicit read-only statements — strengthen, do not overwrite

## Open Questions

1. **Slash command scope for this phase**
   - What we know: 33 slash commands exist; they already have complex workflow steps with error handling
   - What's unclear: Whether phase scope includes all 33 or just high-risk subset
   - Recommendation: Address high-risk 6 (init, build, lay-eggs, seal, entomb, colonize) in this phase; leave display commands for later

2. **Peer review specification for watcher**
   - What we know: Watcher is classified as HIGH-risk because false negatives block phases
   - What's unclear: Who reviews the watcher's output when watcher itself is the reviewer?
   - Recommendation: Watcher self-verifies with explicit requirement to re-run every verification command fresh; Queen escalates if watcher quality_score < 7

3. **Surveyor agents: update vs. append**
   - What we know: Surveyors already have `<success_criteria>` in their definitions (in `<process>` section)
   - What's unclear: Whether to update those existing tags or append new ones
   - Recommendation: Update in-place so the XML is consistent; add `<failure_modes>` and `<read_only>` as new sections

## Sources

### Primary (HIGH confidence)
- Direct file reads of all 24 `.opencode/agents/` files — agent roles, existing content, current format
- Direct file reads of `.claude/rules/security.md` — authoritative protected paths list
- Direct file reads of CLAUDE.md — additional protected paths and architecture
- Phase 23 CONTEXT.md — locked decisions that constrain all choices

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` — reference for how commands handle failure currently (wave failure halt, verification failure flagging)
- `.claude/commands/ant/swarm.md` — reference for how commands escalate to user (ranked solution display)
- `.claude/rules/aether-specific.md` — protected path confirmation

### Tertiary (LOW confidence)
- General LLM agent system prompt best practices from training knowledge — XML structured sections improve instruction following; tiered severity reduces false escalations

## Metadata

**Confidence breakdown:**
- Agent inventory and risk tiers: HIGH — based on direct file reads of all agents
- XML template format: HIGH — follows existing pattern in surveyor agents
- Protected path list: HIGH — sourced from security.md and CLAUDE.md directly
- Slash command treatment: MEDIUM — judgment call on scope vs. noise trade-off
- Peer review classification: MEDIUM — reasonable from role analysis, not validated against runtime behavior

**Research date:** 2026-02-19
**Valid until:** 2026-03-20 (stable domain — agent definitions change slowly)
