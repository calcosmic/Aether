# Single Prompt: Diagnose Aether Self-Reference Cleanup

Copy and paste the entire block below into an AI agent to run the complete diagnosis.

---

```
You are analyzing the Aether repo to identify what is "Aether developing Aether" (self-referential) vs. what is core infrastructure that should be kept.

CONTEXT: Aether is a system for coordinating AI agents on projects. During development, it was configured to use Aether on itself, creating confusing self-referential state. We need to remove the self-reference machinery while preserving all valuable work (bugs found, patterns discovered, useful utilities).

YOUR TASK: Analyze the codebase and produce a cleanup plan with four categories:
1. REMOVE - Self-referential machinery only useful for Aether-on-Aether
2. KEEP - Core infrastructure that works on OTHER projects
3. EXTRACT - Valuable findings to preserve in TODO.md/KNOWN_ISSUES.md before deletion
4. UNCLEAR - Needs human decision

---

## Part 1: Extract All Findings from Oracle Research

Read these files and extract ALL bugs, issues, and gaps:
- .aether/oracle/progress.md (MAIN SOURCE - contains all findings)
- .aether/oracle/research.json
- .aether/oracle/archive/ (any additional findings)

For each finding, provide:
| ID | Type | Severity | Status | File | Description |
|----|------|----------|--------|------|-------------|

Separate into:
- UNFIXED bugs/issues (need to go in TODO.md)
- FIXED bugs/issues (document for reference)
- Architecture gaps (need decision on keep/remove)

---

## Part 2: State Files Analysis

Check which state files exist and analyze them:
- .aether/data/COLONY_STATE.json
- .aether/data/spawn-tree.txt
- .aether/data/swarm-findings.json
- .aether/data/pheromones.json
- .aether/data/activity.log
- .aether/data/flags.json
- .aether/data/learning-log.json
- .aether/data/graveyard.json

For each file that exists:
| File | Purpose | Self-Reference? | Extract Value? | Recommendation |
|------|---------|-----------------|----------------|----------------|

---

## Part 3: Colony Commands Analysis

Analyze these commands:
- .claude/commands/ant/init.md
- .claude/commands/ant/build.md
- .claude/commands/ant/continue.md
- .claude/commands/ant/seal.md
- .claude/commands/ant/entomb.md
- .claude/commands/ant/swarm.md
- .claude/commands/ant/organize.md
- .claude/commands/ant/colonize.md
- .opencode/commands/ant/ mirrors of above

For each command:
| File | Purpose | Works on Other Repos? | Self-References? | Recommendation |

---

## Part 4: Agent Definitions Analysis

Analyze all files in .opencode/agents/aether-*.md

For each agent:
| Agent Name | Purpose | For Aether-Itself? | For Other Projects? | Recommendation |

---

## Part 5: aether-utils.sh Function Analysis

Read .aether/aether-utils.sh and categorize functions:

Group 1 - REMOVE (self-reference only):
- colony-*, spawn-*, queen-*, swarm-*, context-*, flag-*, learning-*, grave-* (if only for self-tracking)

Group 2 - KEEP (general utilities):
- json_ok, json_err, atomic_write support, file locking, model-*, caste-*

Group 3 - UNCLEAR:
- Functions where purpose isn't obvious or might have mixed use

| Function Name | Lines | Purpose | Category |

---

## Part 6: Documentation Analysis

Analyze:
- .aether/docs/ (all files)
- .aether/templates/ (all files)
- .aether/workers.md
- .aether/QUEEN.md
- .aether/coding-standards.md

| File | Purpose | Self-Reference? | Keep? |

---

## Part 7: Dependency Analysis

Search for cross-references:
1. What reads/writes COLONY_STATE.json?
2. What reads/writes .aether/data/?
3. What calls colony-*, queen-*, swarm-* functions?
4. What spawns aether-* agents?

If we remove colony state, what breaks?

---

## FINAL OUTPUT

After analysis, provide:

### 1. Files to DELETE (with full paths)
```
.aether/data/COLONY_STATE.json
...
```

### 2. Files to KEEP
```
.aether/aether-utils.sh (with these functions removed: ...)
...
```

### 3. Content to EXTRACT to TODO.md
```markdown
## Unfixed Bugs
- BUG-###: description (file:line)
...

## Unfixed Issues
- ISSUE-###: description
...

## Architecture Gaps
- GAP-###: description
...
```

### 4. Content to EXTRACT to KNOWN_ISSUES.md
```markdown
## Known Issues
...
```

### 5. Functions to REMOVE from aether-utils.sh
- function_name (line X-Y): reason

### 6. Commands to REMOVE
- path/to/command.md: reason

### 7. UNCLEAR Items (need human decision)
- item: why it's unclear, options

### 8. Removal Order
What order to delete things in to avoid breaking dependencies

### 9. Verification Steps
How to verify the system still works after cleanup
```
