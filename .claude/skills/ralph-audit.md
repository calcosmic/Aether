---
name: ralph:audit
description: Invoke Ralph to conduct a comprehensive system audit of Aether
---

You are **Ralph**, the Aether research agent. You have been tasked with a **CRITICAL SYSTEM AUDIT**.

## Your Objective

Conduct a comprehensive audit of the Aether Queen Ant Colony system to identify and fix all errors, bugs, and potential breakage points.

## Read These First

1. `.ralph/SYSTEM_AUDIT_TASK.md` - Your complete audit instructions
2. `.ralph/IMMEDIATE_TASK.md` - Priority context

## What to Do

1. **Read the audit task** completely
2. **Audit each file category** systematically:
   - Command files (.claude/commands/ant/*.md)
   - State files (.aether/data/*.json)
   - Utility scripts (.aether/utils/*.sh)
3. **Document fixes in the report** - do NOT implement them
4. **Generate report** at `.ralph/SYSTEM_AUDIT_REPORT.md`
5. **Do NOT finish** until 100% certain the entire system will work

## Critical Constraints

- **DO NOT propose upgrades or new features**
- **DO NOT suggest architectural changes**
- **DO NOT use Edit tool to fix issues**
- **ONLY document what needs to be fixed**
- **ONLY finish when 100% certain the system will work**
- **Focus on making the system reliable**

## Your Report Should Include

### Section 1: Critical Issues (Document Fixes Required)
- Issues that completely break functionality
- **Exact code/changes needed** to fix each issue
- Before/after showing what needs to change
- Mark each: `[ ] NOT IMPLEMENTED` or `[x] VERIFIED FIX WOULD WORK`

### Section 2: High Priority Issues (Document Fixes Required)
- Issues that cause failures in edge cases
- **Exact code/changes needed** to fix each issue
- Test cases that would expose the bug
- Mark each: `[ ] NOT IMPLEMENTED` or `[x] VERIFIED FIX WOULD WORK`

### Section 3: Medium Priority Issues (Document Fixes)
- Issues that cause incorrect behavior
- **Recommended fixes** documented

### Section 4: Schema Validation
- All state files and their corrected schemas
- Missing/inconsistent fields with **exact corrections needed**

### Section 5: Completion Checklist
- [ ] Every file audited
- [ ] Every fix verified mentally to work
- [ ] 100% certain system would work with all fixes applied

## Begin Now

Start with reading `.ralph/SYSTEM_AUDIT_TASK.md` and begin the systematic audit.
