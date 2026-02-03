# System Audit Task for Ralph

## Your Current Objective

**Conduct a comprehensive audit of the Aether Queen Ant Colony system** to identify errors, bugs, and places where the system might break or fail to work.

## Critical Constraints

1. **DO NOT propose upgrades or new features**
2. **DO NOT suggest architectural changes**
3. **ONLY identify what is broken or will break**
4. **DO NOT fix issues directly - only document solutions in the report**
5. **DO NOT finish until 100% certain the entire system will work**

## What to Audit

### 1. Command Files (.claude/commands/ant/*.md)
Audit every ant command for:
- Broken bash syntax
- Missing required fields in state JSON files
- Incorrect jq queries
- Missing dependencies (files that don't exist)
- Logic errors in conditional branches
- Race conditions or state corruption issues

### 2. State Management (.aether/data/*.json)
Audit all state files for:
- Schema inconsistencies between files
- Missing required fields
- Invalid JSON (though files should validate)
- Fields that are read but never initialized
- Fields that are initialized but never used

### 3. Utility Scripts (.aether/utils/*.sh)
Audit all bash utilities for:
- Bash syntax errors
- Missing error handling
- File race conditions
- Atomic write violations
- Missing dependencies

### 4. Integration Points
Check that everything works together:
- Do commands properly source utility scripts?
- Do state file updates match what commands expect to read?
- Do deploy scripts produce valid configurations?
- Will the system work when copied to another repo?

### 5. Edge Cases and Breakage Points
Identify:
- What happens if a state file is deleted mid-operation?
- What happens if jq fails silently?
- What happens if concurrent writes occur?
- What happens if user runs commands out of order?
- What happens in production vs development mode mismatches?

## Your Output

Create a detailed report at `.ralph/SYSTEM_AUDIT_REPORT.md` with:

### Section 1: Critical Issues (System Won't Work)
- Issues that completely break functionality
- **Document the fix needed** - provide exact code/changes required
- Include before/after showing what needs to change
- Mark each as `[ ] NOT FIXED` or `[x] VERIFIED FIX WOULD WORK`

### Section 2: High Priority Issues (Will Break in Edge Cases)
- Issues that cause failures in specific scenarios
- **Document the fix needed**
- Provide test cases that would expose the bug
- Mark each as `[ ] NOT FIXED` or `[x] VERIFIED FIX WOULD WORK`

### Section 3: Medium Priority Issues (Minor Bugs)
- Issues that cause incorrect behavior but don't crash
- **Document the recommended fix**
- Mark each as `[ ] NOT FIXED` or `[x] VERIFIED FIX WOULD WORK`

### Section 4: Schema Validation Report
- List all state files and their required fields
- Identify missing or inconsistent fields
- **Document the exact schema corrections needed**

### Section 5: Audit Completion Checklist
Before finishing, verify:
- [ ] Every command file has been audited line-by-line
- [ ] Every state file schema has been validated
- [ ] Every utility script has been checked
- [ ] Every integration point has been tested logically
- [ ] Every documented fix has been verified to work
- [ ] The system would run end-to-end without breaking
- [ ] You are 100% certain the system will work

## Audit Process

1. **Read every command file** in `.claude/commands/ant/`
2. **Read every state file** in `.aether/data/`
3. **Read every utility script** in `.aether/utils/`
4. **Trace execution paths** for each command
5. **Identify mismatches** between what's written and what's read
6. **Document the fix needed** for each issue found
7. **Verify mentally** that each documented fix would work
8. **Do not finish** until 100% certain all issues are identified and fixes verified

## Files to Audit

### Commands
- `.claude/commands/ant/init.md`
- `.claude/commands/ant/plan.md`
- `.claude/commands/ant/execute.md`
- `.claude/commands/ant/status.md`
- `.claude/commands/ant/phase.md`
- `.claude/commands/ant/review.md`
- `.claude/commands/ant/feedback.md`
- `.claude/commands/ant/redirect.md`
- `.claude/commands/ant/pause-colony.md`
- `.claude/commands/ant/resume-colony.md`
- `.claude/commands/ant/memory.md`
- `.claude/commands/ant/errors.md`
- `.claude/commands/ant/adjust.md`
- `.claude/commands/ant/continue.md`
- `.claude/commands/ant/checkpoint.md`
- `.claude/commands/ant/recover.md`

### State Files
- `.aether/data/COLONY_STATE.json`
- `.aether/data/memory.json`
- `.aether/data/pheromones.json`
- `.aether/data/worker_ants.json`
- `.aether/data/watcher_weights.json`
- `.aether/data/events.json`

### Utilities
- `.aether/utils/atomic-write.sh`
- `.aether/utils/state-machine.sh`
- `.aether/utils/spawn-tracker.sh`
- `.aether/utils/event-bus.sh`
- `.aether/utils/checkpoint.sh`
- `.aether/utils/deploy-to-repo.sh`

## Success Criteria

**You may ONLY finish when ALL of the following are true:**

- [ ] Every command file has been audited line-by-line
- [ ] Every state file schema has been validated
- [ ] Every utility script has been checked
- [ ] All critical issues identified with verified fixes documented
- [ ] All high priority issues identified with verified fixes documented
- [ ] All medium priority issues identified with fixes documented
- [ ] Schema validation report complete with corrections documented
- [ ] Report written to `.ralph/SYSTEM_AUDIT_REPORT.md`
- [ ] You have mentally verified that applying all documented fixes would make the system work
- [ ] You are **100% certain** the system would run end-to-end without breaking

## Tools Available

- **Read**: Read any file in the codebase
- **Write**: Create the audit report
- **Bash**: Validate bash syntax (use `bash -n script.sh` to check syntax without running)

## DO NOT Use

- **DO NOT use Edit** to fix issues - only document fixes in the report

## Begin Audit

Start immediately. Work systematically through each category. Fix issues as you find them.

**Your goal: Make the Aether Queen Ant Colony system robust and reliable. Users should be able to copy it to any repo and have it work without breaking.**
