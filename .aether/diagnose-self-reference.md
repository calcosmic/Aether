# Diagnosis: Aether Self-Reference Cleanup

## Context

The Aether system was designed to coordinate AI agents on projects. During development, it was configured to use Aether on itself (Aether developing Aether), creating confusing self-referential state. We need to identify what to remove while preserving all valuable work.

## Your Task

Analyze the codebase to categorize every component into one of four categories:

1. **REMOVE** - Self-referential machinery (colony state tracking Aether's own development)
2. **KEEP** - Core infrastructure that works on OTHER projects
3. **EXTRACT** - Valuable findings/bugs to preserve before deletion
4. **UNCLEAR** - Needs human decision

---

## Research Areas

### Area 1: Colony/Swarm State Files

**Files to analyze:**
- `.aether/data/COLONY_STATE.json`
- `.aether/data/spawn-tree.txt`
- `.aether/data/swarm-findings.json`
- `.aether/data/pheromones.json`
- `.aether/data/activity.log`
- `.aether/data/flags.json`
- `.aether/data/learning-log.json`
- `.aether/data/graveyard.json`

**For each file, determine:**
- Is this tracking Aether's own development or is it a template/example?
- Does it contain unique valuable data (bugs, learnings, patterns)?
- Can it be safely deleted after extraction?

**Output format:**
```
FILE: <path>
CATEGORY: REMOVE | KEEP | EXTRACT | UNCLEAR
VALUABLE_CONTENT: <list specific items if EXTRACT>
DEPENDENCIES: <what reads/writes this file>
CONFIDENCE: 1-10
REASONING: <brief explanation>
```

---

### Area 2: Oracle Research System

**Files to analyze:**
- `.aether/oracle/progress.md`
- `.aether/oracle/research.json`
- `.aether/oracle/.last-topic`
- `.aether/oracle/archive/` (all files)

**For each file, determine:**
- What research was conducted?
- List ALL bugs found (BUG-001 through BUG-012)
- List ALL issues found (ISSUE-001 through ISSUE-007)
- List ALL gaps found (GAP-001 through GAP-010)
- Which findings are still relevant/unfixed?
- Is this research about Aether itself or general methodology?

**Output format:**
```
FILE: <path>
CATEGORY: REMOVE | KEEP | EXTRACT | UNCLEAR
FINDINGS_SUMMARY:
  - BUG-###: <description> - <status: fixed/unfixed/unclear>
  - ISSUE-###: <description> - <status>
  - GAP-###: <description> - <status>
ACTION_REQUIRED: <what to do with this file>
CONFIDENCE: 1-10
```

---

### Area 3: Colony Workflow Commands

**Files to analyze:**
- `.claude/commands/ant/init.md`
- `.claude/commands/ant/build.md`
- `.claude/commands/ant/continue.md`
- `.claude/commands/ant/seal.md`
- `.claude/commands/ant/entomb.md`
- `.claude/commands/ant/swarm.md`
- `.claude/commands/ant/organize.md`
- `.claude/commands/ant/colonize.md`
- `.opencode/commands/ant/` mirrors of above

**For each file, determine:**
- Is this command designed to work on ANY repo or specifically Aether-on-Aether?
- Does it contain valuable patterns/ideas that should be preserved?
- What other files/functions does it reference?

**Output format:**
```
FILE: <path>
CATEGORY: REMOVE | KEEP | EXTRACT | UNCLEAR
PURPOSE: <what this command does>
WORKS_ON_OTHER_REPOS: YES | NO | PARTIALLY
VALUABLE_PATTERNS: <list any reusable patterns>
DEPENDENCIES: <what it calls/requires>
CONFIDENCE: 1-10
```

---

### Area 4: Agent Definitions

**Files to analyze:**
- `.opencode/agents/aether-*.md` (all files)
- Any agent definitions in `.claude/agents/`

**For each file, determine:**
- Is this agent designed to work on Aether itself or other projects?
- What caste/specialization does it represent?
- Is it referenced by any commands we're keeping?

**Output format:**
```
FILE: <path>
CATEGORY: REMOVE | KEEP | EXTRACT | UNCLEAR
AGENT_TYPE: <name>
DESIGNED_FOR: AETHER_ITSELF | OTHER_PROJECTS | BOTH
REFERENCED_BY: <list commands that use it>
CONFIDENCE: 1-10
```

---

### Area 5: Utility Functions in aether-utils.sh

**File to analyze:** `.aether/aether-utils.sh`

**For each function, determine:**

1. **Colony management functions:**
   - `colony-*` functions
   - `queen-*` functions
   - `swarm-*` functions
   - `spawn-*` functions
   - `context-*` functions
   - `flag-*` functions
   - `learning-*` functions
   - `grave-*` functions
   - `view-state-*` functions

2. **Core utility functions:**
   - `json_ok` / `json_err`
   - `atomic_write` usage
   - File locking functions
   - `model-*` functions
   - `caste-*` functions

**Output format:**
```
FUNCTION: <name>
CATEGORY: REMOVE | KEEP | REFACTOR | UNCLEAR
PURPOSE: <what it does>
USED_BY: <what calls it>
USED_FOR_SELF_REFERENCE: YES | NO | PARTIALLY
CONFIDENCE: 1-10
```

---

### Area 6: Documentation and Templates

**Files to analyze:**
- `.aether/docs/` (all files)
- `.aether/templates/` (all files)
- `.aether/visualizations/` (all files)
- `.aether/workers.md`
- `.aether/QUEEN.md`
- `.aether/coding-standards.md`

**For each file, determine:**
- Is this documentation about Aether's own development or general usage?
- Does it contain valuable patterns/standards?
- Is it referenced by commands we're keeping?

**Output format:**
```
FILE: <path>
CATEGORY: REMOVE | KEEP | EXTRACT | UNCLEAR
CONTENT_TYPE: SELF_REFERENCE | GENERAL_DOCS | BOTH
VALUABLE_CONTENT: <list if applicable>
CONFIDENCE: 1-10
```

---

### Area 7: Dependency Graph

**Task:** Map all dependencies between components

**Output a dependency graph showing:**
- Which files read which other files
- Which commands call which functions
- Which agents are used by which commands
- What breaks if we remove X

**Output format:**
```
COMPONENT: <name>
DEPENDS_ON: <list>
DEPENDENTS: <list>
SAFE_TO_REMOVE: YES | NO | AFTER_<condition>
```

---

## Final Deliverable

After completing all areas, produce:

### 1. Summary Table

| Component | Category | Confidence | Action Required |
|-----------|----------|------------|-----------------|
| ... | ... | ... | ... |

### 2. Extraction List

Items that must be extracted before deletion:

```
- BUG-###: <description> → TODO.md
- ISSUE-###: <description> → KNOWN_ISSUES.md
- Pattern: <description> → IMPLEMENTATION_NOTES.md
```

### 3. Safe Removal List

Files that can be deleted after extraction (full paths):

```
.aether/data/COLONY_STATE.json
.aether/oracle/...
...
```

### 4. Keep List

Files/functions that must be preserved:

```
.aether/aether-utils.sh (with these functions: ...)
.aether/workers.md
...
```

### 5. Unclear Items Requiring Human Decision

```
<item> - <why it's unclear> - <options>
```

### 6. Risk Assessment

- What could break if we remove X?
- What tests should we run after cleanup?
- How do we verify the system still works on OTHER projects?

---

## Constraints

- Be conservative: if unsure, mark as UNCLEAR
- Preserve ALL bugs, issues, gaps, and learnings
- Do not recommend deletion of anything that might be useful on other projects
- Check for hardcoded paths referencing `.aether/data/`
- Verify no circular dependencies will be broken
