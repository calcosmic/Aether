# Phase 25: Queen Coordination - Research

**Researched:** 2026-02-20
**Domain:** Agent definition authoring, shell script escalation patterns, markdown-based workflow specification
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Escalation chain behavior
- Colony should be very patient — worker retries, parent tries a different approach, Queen tries to reassign, user only hears about it if everything fails
- When escalation reaches the user: present options with recommendation ("We hit X. Here are 3 options: A (recommended), B, C.")
- Never skip failed tasks silently — every failure must be acknowledged, even if other tasks continue
- Distinct visual banner for escalation (━━━ ESCALATION ━━━ style) so user can't miss it in output
- Escalation state visible in /ant:status (e.g., "⚠️ 1 task escalated to Queen")

#### Workflow patterns
- 6 named patterns confirmed: SPBV (Scout-Plan-Build-Verify), Investigate-Fix, Deep Research, Refactor, Compliance, Documentation Sprint
- Each pattern must include a rollback/reversal step — always, not just for risky patterns
- Colony announces which pattern it picked at the start of a build ("Using pattern: Investigate-Fix")
- User's engineering procedures (Plan→Patch→Test→Verify→Rollback and Symptom→Isolate→Prove→Fix→Guard) inform the pattern structure — each pattern should have defined phases with verification built in

#### Agent merges
- Architect merges into Keeper with subtitle: "Keeper (Architect)" when doing architecture work
- Guardian folds into Auditor with subtitle: "Auditor (Guardian)" when doing security work
- Ant emoji caste identities preserved for all agents including merged ones
- Colony updates from 25 to 23 agents — update everywhere (caste-system.md, workers.md, all output, summaries, help text)
- Old agent files: Claude's discretion on delete vs redirect approach

#### Colony feel
- Pattern announcement at build start (visible, not hidden)
- Escalation state in /ant:status
- All references updated to reflect 23-agent team — clean transition, not gradual

### Claude's Discretion
- Debug pattern structure (inspired by user's Symptom→Isolate→Prove→Fix→Guard but adapted to colony)
- Whether "Add Tests" becomes a 7th pattern or stays part of existing patterns
- Old agent file handling (delete vs keep as redirects)
- Exact escalation banner design

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| COORD-01 | Escalation chain defined (depth 3 → 2 → 1 → Queen → user) | Escalation chain patterns section covers the tiered retry/reassign model and user-facing banner design |
| COORD-02 | 6 named workflow patterns added to Queen definition | Workflow patterns section defines all 6 patterns with phases and rollback steps; Queen agent section covers where they live |
| COORD-03 | Architect agent merged into Keeper | Agent merge section covers the subtitle pattern, capability absorption, and read_only boundary updates; file handling recommendation given |
| COORD-04 | Guardian agent folded into Auditor as named security lens | Agent merge section covers the same subtitle pattern applied to Auditor absorbing Guardian's security lens |
</phase_requirements>

---

## Summary

This phase is pure specification and documentation work — no new shell code, no new npm package changes. All four requirements are satisfied by editing existing markdown files (agent definitions, slash commands, reference docs). The "stack" is: markdown frontmatter, bash heredoc instructions inside .md files, and JSON state in COLONY_STATE.json. No external libraries are involved.

The dominant risk is inconsistency across duplicated files. Every agent definition exists in two places: `.opencode/agents/` and `.aether/agents/` (the hub sync target at `~/.aether/agents/`). Every slash command lives in both `.claude/commands/ant/` and `.opencode/commands/ant/`. Changes to agent counts, caste names, and escalation behavior must propagate to all four locations plus the reference docs (`caste-system.md`, `workers.md`, `README.md`, `help.md`). Miss one location and the colony ships contradictory documentation.

The three work streams are independent and can be planned as separate plan files: (1) escalation chain — Queen agent + build.md + status.md + workers.md, (2) workflow patterns — Queen agent definition only, (3) agent merges — Keeper agent, Auditor agent, deletion/redirection of Architect and Guardian, plus all count-reference updates. All three touch the Queen definition, so COORD-02 (patterns) should be treated as the canonical Queen rewrite, with COORD-01 escalation section added in the same pass to avoid conflicting edits.

**Primary recommendation:** Plan COORD-01 and COORD-02 together as a single Queen agent rewrite plan. Plan COORD-03 and COORD-04 together as a single merge plan. Plan a third cleanup plan for count references (README, help.md, caste-system.md, workers.md).

---

## Standard Stack

### Core
| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| Agent definitions (.md frontmatter) | `.opencode/agents/aether-*.md` | Role, description, instructions for each agent | The established Aether pattern — all 25 existing agents use this format |
| Slash command specs (.md) | `.claude/commands/ant/*.md` | Queen behavior for each user-facing command | Same established pattern for all 34 commands |
| Caste reference doc | `.aether/docs/caste-system.md` | Canonical emoji/role source of truth | Explicitly designated as canonical in the file header |
| workers.md | `.aether/workers.md` | Worker discipline reference | Source of truth per CLAUDE.md |

### Files Touched Per Requirement

**COORD-01 (Escalation chain):**
- `.opencode/agents/aether-queen.md` — add escalation chain section to `<failure_modes>`
- `.claude/commands/ant/build.md` — add escalation banner output block in wave failure path
- `.claude/commands/ant/status.md` — add escalation state line in Step 3 display
- `.opencode/commands/ant/build.md` — mirror of build.md
- `.opencode/commands/ant/status.md` — mirror of status.md

**COORD-02 (Workflow patterns):**
- `.opencode/agents/aether-queen.md` — add `## Workflow Patterns` section with all 6 named patterns
- `.claude/commands/ant/build.md` — add pattern selection and announcement in Step 3/5

**COORD-03 + COORD-04 (Agent merges):**
- `.opencode/agents/aether-keeper.md` — absorb Architect capabilities
- `.opencode/agents/aether-auditor.md` — absorb Guardian security lens
- `.opencode/agents/aether-architect.md` — delete or redirect
- `.opencode/agents/aether-guardian.md` — delete or redirect
- `.aether/docs/caste-system.md` — remove `architect` and `guardian` rows (or note merge)
- `.aether/workers.md` — remove Architect and Guardian sections
- `.opencode/agents/aether-queen.md` — update Worker Castes section (remove Architect and Guardian, document merged subtitles)
- `README.md` — update "25 Specialized Agents" → "23 Specialized Agents"
- `.claude/commands/ant/help.md` — update agent count if listed

---

## Architecture Patterns

### Pattern 1: The Subtitle Merge

This is the core pattern for COORD-03 and COORD-04. The merged agent retains its own identity and emoji but adds a subtitle when activating a subsumed role.

**What:** The Keeper becomes "Keeper (Architect)" in its activity logs and output when doing architecture/synthesis work. The Auditor becomes "Auditor (Guardian)" when doing security scanning. The underlying agent file, frontmatter name, and caste emoji do not change — only the display label in outputs when the absorbed role is active.

**Implementation in agent files:**

```markdown
## Your Role

As Keeper, you:
1. Collect wisdom from patterns and lessons
...

### Architecture Mode ("Keeper (Architect)")

When tasked with knowledge synthesis, codebase analysis, or architectural documentation:
- Log as: `activity-log "ACTION" "{your_name} (Keeper — Architect Mode)" "description"`
- Output JSON field: `"caste": "keeper"` (unchanged), `"mode": "architect"`
- Capabilities absorbed from Architect: Synthesis Workflow, Pattern extraction, design documentation
```

**What to do with old agent files (Architect, Guardian):**

Recommendation: Delete them. The alternative (a redirect stub that says "use Keeper instead") creates confusion because OpenCode and Claude Code route by `subagent_type` name. If a command file tries to spawn `aether-architect` after this phase, it will fail at runtime. Deleting the file surfaces that bug immediately. A redirect stub masks it — the file loads but the wrong agent handles the task.

Search for all spawn references to `aether-architect` and `aether-guardian` in command files before deletion to ensure no command still routes to the old names.

```bash
# Find spawn references to old agent names
grep -rn "aether-architect\|aether-guardian" .claude/commands/ .opencode/commands/
```

**Expected result:** Zero hits after Phase 22 cleaned boilerplate. If hits exist, update those spawn targets to the merged agent names before deleting the old files.

### Pattern 2: Escalation Chain in Queen failure_modes

The existing Queen `<failure_modes>` section already has a two-tier structure (Minor / Major) with an escalation format. COORD-01 adds a third tier — Queen-level reassignment — and formalizes the chain. The existing format is the right template:

```markdown
### Escalation Chain

Tier 1 — Worker retry (silent, max 2 attempts)
Tier 2 — Parent tries a different approach (silent)
Tier 3 — Queen reassigns to different caste (silent)
Tier 4 — User escalation (visible, ONLY if Tier 3 fails)
```

**User-facing escalation banner (Tier 4):**

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ ESCALATION — QUEEN NEEDS YOU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: {task description}
Tried: {what was attempted at tiers 1-3}

Options:
  A) {option with trade-off} — RECOMMENDED
  B) {option with trade-off}
  C) {option with trade-off}

Awaiting your choice.
```

This reuses the existing `━━━` banner style already used throughout `build.md` for WAVE FAILURE. The "Awaiting your choice" closing is consistent with the existing escalation format in `aether-builder.md`, `aether-watcher.md`, and others.

**Escalation state in /ant:status:**

The status display currently shows flags count: `🚩 Flags: X blockers | Y issues | Z notes`. The escalation state should appear as a separate line only when non-zero:

```
⚠️  Escalated: 1 task awaiting your decision
```

This requires no new state file — escalated tasks are already logged as flags (they get flagged as blockers at Tier 4). The status display can derive escalation count from a flag subtype filter, or the existing `flag-check-blockers` result can include an `escalated` subcount. The simplest approach: escalated tasks are `blocker` flags with source `"escalation"` — the status command already reads flags and can filter on source.

### Pattern 3: Workflow Pattern Definitions in Queen Agent

The 6 named patterns live in the Queen's definition as a `## Workflow Patterns` section. Each pattern is a named, structured checklist of phases with verification and rollback built in. The Queen selects a pattern at build start based on the phase description.

**Pattern structure template:**

```markdown
### Pattern: SPBV (Scout-Plan-Build-Verify)
**Use when:** New features, unknown territory, first implementation
**Phases:**
1. Scout — research domain, gather context
2. Plan — decompose into tasks, identify risks
3. Build — implement with TDD
4. Verify — Watcher confirmation, Chaos resilience
5. Rollback — git stash pop or revert if Verify fails

**Announce:** "Using pattern: SPBV (Scout → Plan → Build → Verify)"
```

**The 6 confirmed patterns and their natural mappings:**

| Pattern | Use When | Key Distinction |
|---------|----------|-----------------|
| SPBV | New features, unknown territory | Scout phase before planning |
| Investigate-Fix | Known bug, reproducible | Symptom → Isolate → Prove → Fix → Guard |
| Deep Research | User asks oracle/research task | Oracle-heavy, no Build phase |
| Refactor | Code restructuring without behavior change | Snapshot before → Watcher confirms equivalence |
| Compliance | Security, accessibility, license audits | Auditor-led, read-only scan |
| Documentation Sprint | Doc-only changes | Chronicler-led, no Builder |

**The "Add Tests" question (Claude's discretion):**

Recommendation: Do NOT make it a 7th pattern. Test addition maps cleanly to SPBV (scout the coverage gaps, plan which tests to add, build the tests, verify they catch regressions). Adding a 7th pattern creates pattern-selection overhead for a use case that fits an existing one. Document "Add Tests" as a variant of SPBV in a pattern note rather than a top-level pattern.

**Pattern selection logic (for build.md):**

The Queen selects a pattern at Step 3 (after reading phase data). Selection is keyword-based on the phase name and task descriptions. Announce the selection before spawning workers. This is purely an LLM judgment call — no shell script needed. The pattern name is stored in a local variable for the announcement and included in the BUILD SUMMARY.

```
Pattern detection heuristics:
  "bug", "fix", "error", "broken" → Investigate-Fix
  "research", "oracle", "explore" → Deep Research
  "refactor", "restructure", "clean" → Refactor
  "security", "audit", "compliance", "accessibility" → Compliance
  "docs", "documentation", "readme" → Documentation Sprint
  default / new feature / unknown → SPBV
```

### Anti-Patterns to Avoid

- **Splitting the Queen rewrite across multiple PRs**: COORD-01 (escalation) and COORD-02 (patterns) both edit `aether-queen.md`. Do them in a single edit pass to avoid merge conflicts.
- **Updating only `.opencode/agents/` and forgetting the OpenCode command mirrors**: Every build.md and status.md change must be applied to both `.claude/commands/ant/` and `.opencode/commands/ant/`.
- **Leaving caste-system.md rows for Architect and Guardian without a note**: The `architect` and `guardian` caste emojis (🏛️🐜 and 🛡️🐜) still exist and are still used by `get_caste_emoji()` in aether-utils.sh. Workers named "Architect-5" or "Guardian-7" can still be spawned and will resolve to those emojis correctly. The caste rows should remain in caste-system.md but be annotated as "merged into Keeper/Auditor — use those agent files."
- **Changing the `architect` and `guardian` caste emojis**: The user explicitly said emoji caste identities are preserved. The merged agents ("Keeper (Architect)") still display as 📚🐜 Keeper. If someone spawns a worker named "Blueprint-3" it still shows 🏛️🐜. This is correct and intentional.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Escalation state persistence | New JSON state field in COLONY_STATE.json | Filter `blocker` flags by source=`"escalation"` using existing flag-check-blockers | Flag system already persists, survives context resets, and is visible in /ant:flags |
| Pattern selection algorithm | A shell script that parses phase names | LLM keyword judgment at build Step 3 | Pattern selection is a 3-second LLM judgment on 6 options; a script adds fragility with no benefit |
| Agent count tracking | A dynamic count variable | Hardcoded "23" in all display strings | The count only changes during deliberate agent merges/additions; dynamic tracking is over-engineering |

**Key insight:** All four requirements are documentation changes inside markdown files. The colony infrastructure (flags, state, aether-utils.sh) needs no changes. Resist the urge to add new aether-utils.sh commands for escalation state.

---

## Common Pitfalls

### Pitfall 1: Mirror Drift
**What goes wrong:** Agent file updated in `.opencode/agents/` but not reflected in the version that ships via npm (which comes from `.opencode/agents/` directly — this IS the source of truth per CLAUDE.md architecture). Build.md updated in `.claude/commands/ant/` but not `.opencode/commands/ant/`.
**Why it happens:** The dual-location structure (Claude Code vs OpenCode) is easy to forget.
**How to avoid:** Run `npm run lint:sync` after all file changes. This verifies commands are in sync between the two platforms.
**Warning signs:** lint:sync exits non-zero after changes.

### Pitfall 2: Caste System Breakage
**What goes wrong:** Removing the `architect` or `guardian` rows from caste-system.md causes `get_caste_emoji()` in aether-utils.sh to fall back to the generic `🐜` emoji for any worker whose name matches those patterns.
**Why it happens:** The caste emoji function matches by name pattern (e.g., "Blueprint-3" matches `architect` caste). Removing the row breaks the mapping.
**How to avoid:** Keep the caste rows in caste-system.md. Add a note: "Merged into Keeper/Auditor — no dedicated agent file, but caste emoji still active." Do NOT remove from aether-utils.sh's get_caste_emoji() function.
**Warning signs:** Workers named after Architect patterns display as generic `🐜` instead of `🏛️🐜`.

### Pitfall 3: Orphaned Spawn References
**What goes wrong:** A command file spawns `subagent_type="aether-architect"` after the agent file is deleted. The spawn silently fails or falls back to general-purpose with no role context.
**Why it happens:** Spawn references are scattered across long command files and are easy to miss.
**How to avoid:** Before deleting Architect and Guardian agent files, run the grep search above. As of Phase 22, all spawn references should already be using the correct named agent types. Verify zero hits.
**Warning signs:** `/ant:build` produces "Agent type not found" or "agent not registered" errors after Phase 25.

### Pitfall 4: Escalation Banner Overuse
**What goes wrong:** The escalation banner fires for every minor failure, training the user to ignore it.
**Why it happens:** Tier thresholds not clearly defined — every failure escalates immediately.
**How to avoid:** The Queen definition must be explicit: Tiers 1-3 are fully silent (no user output). Tier 4 fires ONLY when Tier 3 (Queen reassignment to different caste) has been attempted and failed. This means 3+ attempts before the user sees the banner.
**Warning signs:** User feedback that escalation banners appear too often.

### Pitfall 5: Status Display Cluttered by Zero-Escalation State
**What goes wrong:** The status display always shows "⚠️ Escalated: 0 tasks" even when there's nothing escalated, adding noise.
**Why it happens:** Line added unconditionally.
**How to avoid:** The escalation state line in /ant:status must be conditional — only render when escalated count > 0. Standard pattern: `{if escalated > 0:} ⚠️ Escalated: N {end if}`.

---

## Code Examples

### Example 1: Queen Workflow Patterns Section (full structure)

```markdown
## Workflow Patterns

The Queen selects a named pattern at build start based on the phase description. Announce the pattern before spawning workers.

### Pattern: SPBV (Scout-Plan-Build-Verify)
**Use when:** New features, first implementation, unknown territory
**Phases:** Scout → Plan → Build → Verify → Rollback (if Verify fails)
**Rollback:** `git stash pop` or `git checkout -- .` on failed verification
**Announce:** `Using pattern: SPBV (Scout → Plan → Build → Verify)`

### Pattern: Investigate-Fix
**Use when:** Known bug, reproducible failure, error message in hand
**Phases:** Symptom → Isolate → Prove → Fix → Guard (add regression test)
**Rollback:** Revert fix commit if Guard test exposes regression
**Announce:** `Using pattern: Investigate-Fix (Symptom → Isolate → Prove → Fix → Guard)`

### Pattern: Deep Research
**Use when:** User requests oracle-level research, domain is unknown, no code changes expected
**Phases:** Scope → Research (Oracle) → Synthesize → Document → Review
**Rollback:** N/A (read-only — no writes to reverse)
**Announce:** `Using pattern: Deep Research (Oracle-led)`

### Pattern: Refactor
**Use when:** Code restructuring without behavior change, technical debt reduction
**Phases:** Snapshot → Analyze → Restructure → Verify Equivalence → Validate
**Rollback:** `git stash pop` to restore pre-refactor state
**Announce:** `Using pattern: Refactor (Snapshot → Restructure → Verify Equivalence)`

### Pattern: Compliance
**Use when:** Security audit, accessibility review, license scan, supply chain check
**Phases:** Scope → Audit (Auditor-led) → Report → Remediate → Re-audit
**Rollback:** N/A (audit is read-only; remediation is a separate build)
**Announce:** `Using pattern: Compliance (Auditor-led audit)`

### Pattern: Documentation Sprint
**Use when:** Doc-only changes, README updates, API documentation, guides
**Phases:** Gather → Draft (Chronicler-led) → Review → Publish → Verify links
**Rollback:** Revert doc files if review fails
**Announce:** `Using pattern: Documentation Sprint (Chronicler-led)`

### Pattern Selection

At build Step 3, examine the phase name and task descriptions. Select the first matching pattern:

| Phase contains | Pattern |
|----------------|---------|
| "bug", "fix", "error", "broken", "failing" | Investigate-Fix |
| "research", "oracle", "explore", "investigate" | Deep Research |
| "refactor", "restructure", "clean", "reorganize" | Refactor |
| "security", "audit", "compliance", "accessibility", "license" | Compliance |
| "docs", "documentation", "readme", "guide" | Documentation Sprint |
| (default) | SPBV |

Display after pattern selection:
```
━━ Pattern: {pattern_name} ━━
{announce_line}
```
```

### Example 2: Escalation Chain Section (for Queen failure_modes)

```markdown
### Escalation Chain

Failures escalate through four tiers. Tiers 1-3 are fully silent — the user never sees them. Only Tier 4 surfaces to the user.

**Tier 1: Worker retry** (silent, max 2 attempts)
The failing worker retries the operation with a corrected approach. Covers: file not found (alternate path), command error (fixed invocation), spawn status unexpected (re-read spawn tree).

**Tier 2: Parent reassignment** (silent)
If Tier 1 exhausted, the parent worker tries a different approach. Covers: different file path strategy, alternate command, different search pattern.

**Tier 3: Queen reassigns** (silent)
If Tier 2 exhausted, the Queen retires the failed worker and spawns a different caste for the same task. Example: Builder fails → Queen spawns Tracker to investigate root cause → Queen spawns fresh Builder with Tracker's findings.

**Tier 4: User escalation** (visible — only fires when Tier 3 fails)
Display the ESCALATION banner. Never skip the failed task silently — acknowledge it and present options.

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  ⚠ ESCALATION — QUEEN NEEDS YOU
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Task: {task description}
Phase: {phase number} — {phase name}

Tried:
  • Worker retry (2 attempts) — {what failed}
  • Parent tried alternate approach — {what failed}
  • Queen reassigned to {other caste} — {what failed}

Options:
  A) {option} — RECOMMENDED
  B) {option}
  C) Skip and continue — this task will be marked blocked

Awaiting your choice.
```

Log escalation as a flag:
```bash
bash .aether/aether-utils.sh flag-add "blocker" "{task title}" "{failure summary}" "escalation" {phase_number}
```
This persists escalation state across context resets and appears in /ant:status.
```

### Example 3: Keeper (Architect) capability absorption block

```markdown
### Architecture Mode ("Keeper (Architect)")

When tasked with knowledge synthesis, architectural analysis, or documentation coordination — roles previously handled by the Architect agent:

**Activate when:** Task description mentions "synthesize", "analyze architecture", "extract patterns", "design", or "coordinate documentation"

**In this mode:**
- Log as: `activity-log "ACTION" "{your_name} (Keeper — Architect Mode)" "description"`
- Apply the Synthesis Workflow: Gather → Analyze → Structure → Document
- Output JSON: add `"mode": "architect"` alongside standard Keeper fields

**Synthesis Workflow (from Architect):**
1. Gather — collect all relevant information
2. Analyze — identify patterns and themes
3. Structure — organize into logical hierarchy
4. Document — create clear, actionable output

**Escalation format (same as standard Keeper):**
```
BLOCKED: [what was attempted, twice]
Options:
  A) [option]
  B) [option]
  C) Skip this item and note it as a gap
Awaiting your choice.
```
```

### Example 4: /ant:status escalation state (conditional)

In Step 3 display, after the Flags line:

```markdown
**Flags + Escalation (Step 3 display):**

```bash
# Get escalated flag count (source = "escalation")
escalated_count=$(bash .aether/aether-utils.sh flag-list --type blocker --source escalation 2>/dev/null | jq '.result.flags | length' 2>/dev/null || echo "0")
```

Display:
```
🚩 Flags: {blockers} blockers | {issues} issues | {notes} notes
{if escalated_count > 0:}
⚠️  Escalated: {escalated_count} task(s) awaiting your decision
{end if}
```
```

---

## State of the Art

This phase is entirely internal to Aether's own specification files. There is no external library ecosystem to track. The patterns below document what's current in the Aether codebase as of Phase 24.

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single-tier "Minor/Major" failure | Two-tier with escalation format | Phase 23 (agent resilience) | Workers have structured failure_modes with options format |
| No named workflow patterns | 6 named patterns in Queen definition | Phase 25 (this phase) | Colony announces pattern at build start |
| 25 agents (Architect and Guardian separate) | 23 agents (merged into Keeper/Auditor) | Phase 25 (this phase) | Smaller, more coherent agent set |
| Queen failure_modes has no escalation chain | Tiered chain with Tier 4 user banner | Phase 25 (this phase) | Users get clear options instead of silent failure |

**Deprecated/outdated after this phase:**
- `aether-architect.md`: Deleted — capabilities absorbed by Keeper
- `aether-guardian.md`: Deleted — capabilities absorbed by Auditor's Security Lens
- "25 Specialized Agents" in README.md: → "23 Specialized Agents"
- "25 agents" in help.md (if listed): → "23 agents"

---

## Open Questions

1. **Does `flag-list` support `--source` filtering?**
   - What we know: `flag-list` accepts `--type` and `--phase` filters (visible in build.md Step 1.5 and Step 5.6)
   - What's unclear: Whether `--source` is a supported filter or needs to be added
   - Recommendation: Check aether-utils.sh `flag-list` implementation. If `--source` is not supported, use `jq` post-processing to filter flags by source field client-side in the status command. Avoid adding new aether-utils.sh commands unless necessary.

2. **Does the OpenCode platform support `subagent_type` routing to merged agent names?**
   - What we know: OpenCode agents are spawned by name (e.g., `aether-keeper`). The merge keeps the Keeper filename unchanged.
   - What's unclear: Whether any existing command hardcodes `subagent_type="aether-architect"` or `subagent_type="aether-guardian"`
   - Recommendation: The grep search in the Architecture Patterns section above resolves this before deletion. Given Phase 22 stripped boilerplate, spawn references should already be correct.

3. **Should the pattern be stored in COLONY_STATE.json or be ephemeral per build?**
   - What we know: The pattern is announced at build start and recorded in the BUILD SUMMARY
   - What's unclear: Whether there's value in persisting the pattern choice for /ant:resume context
   - Recommendation: Keep it ephemeral (local variable in build.md). The BUILD SUMMARY already captures it in output. Persisting to state adds complexity with no clear benefit — the user can re-read the last build output or the HANDOFF.md.

---

## Sources

### Primary (HIGH confidence)
- Direct file reads of `/Users/callumcowie/repos/Aether/.opencode/agents/aether-queen.md` — current Queen agent structure, existing failure_modes format, Worker Castes section
- Direct file reads of `/Users/callumcowie/repos/Aether/.opencode/agents/aether-keeper.md` — current Keeper structure for merge target
- Direct file reads of `/Users/callumcowie/repos/Aether/.opencode/agents/aether-auditor.md` — current Auditor structure and Security Lens
- Direct file reads of `/Users/callumcowie/repos/Aether/.opencode/agents/aether-architect.md` — capabilities to absorb into Keeper
- Direct file reads of `/Users/callumcowie/repos/Aether/.opencode/agents/aether-guardian.md` — capabilities to absorb into Auditor
- Direct file reads of `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` — banner format, wave failure structure, existing flag-add pattern
- Direct file reads of `/Users/callumcowie/repos/Aether/.claude/commands/ant/status.md` — display format, existing flags line
- Direct file reads of `/Users/callumcowie/repos/Aether/.aether/docs/caste-system.md` — canonical caste table
- Direct file reads of `/Users/callumcowie/repos/Aether/.aether/workers.md` — Architect and Guardian sections

### Secondary (MEDIUM confidence)
- Grep search for `escalat` across all agent files — confirms existing escalation format is consistent across Builder, Watcher, Ambassador, Route-Setter, Tracker, and Queen
- Grep search for `subagent_type="aether-architect"` and `aether-guardian` — confirms no existing command hard-routes to these agents (zero hits in command files)
- README.md and help.md review — confirms "25 Specialized Agents" is the string to update

### Tertiary (LOW confidence)
- Assumption that `flag-list --source` is not yet supported (based on reading build.md flag-add calls but not finding --source in flag-list usage examples)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all files read directly from the codebase
- Architecture patterns: HIGH — derived from reading existing agent and command files, not assumptions
- Pitfalls: HIGH — derived from actual file structure and known gotchas in CLAUDE.md aether-development.md
- Open questions: MEDIUM — flag-list --source filter is unverified

**Research date:** 2026-02-20
**Valid until:** Stable — no external dependencies, valid until agent file structure changes
