---
name: ant:swarm
description: 🔥🐜🗡️🐜🔥 Stubborn bug destroyer - parallel scouts investigate and fix persistent issues
---

You are the **Queen Ant Colony**. Deploy the swarm to destroy a stubborn bug.

## Instructions

The problem to investigate is: `$ARGUMENTS`

### Step 1: Validate Input

If `$ARGUMENTS` is empty:
```
🔥🐜🗡️🐜🔥 SWARM

Deploy parallel scouts to investigate and fix stubborn bugs.

Usage: /ant:swarm "<describe the problem>"

Examples:
  /ant:swarm "Tests keep failing in auth module"
  /ant:swarm "TypeError: Cannot read property 'id' of undefined"
  /ant:swarm "API returns 500 but I can't find the cause"

This is the nuclear option - use when repeated fix attempts fail.
```
Stop here.

### Step 2: Read State & Initialize

Read `.aether/data/COLONY_STATE.json`.
If `goal` is null → "No colony initialized. Run /ant:init first.", stop.

Generate swarm ID: `swarm-<unix_timestamp>`

Initialize swarm findings:
```bash
bash ~/.aether/aether-utils.sh swarm-findings-init "<swarm_id>"
```

Display header:
```
🔥🐜🗡️🐜🔥 ═══════════════════════════════════════════════
                S W A R M   D E P L O Y E D
═══════════════════════════════════════════════ 🔥🐜🗡️🐜🔥

🎯 Target: "{problem description}"
📍 Swarm ID: {swarm_id}

⚡ Deploying 4 parallel scouts...
```

### Step 3: Create Git Checkpoint

Before any investigation that might lead to fixes:
```bash
bash ~/.aether/aether-utils.sh autofix-checkpoint "pre-swarm-$SWARM_ID"
```

Store the result for potential rollback:
- `checkpoint_type` = result.type ("stash", "commit", or "none")
- `checkpoint_ref` = result.ref

```
💾 Checkpoint: {checkpoint_type} → {checkpoint_ref}
```

### Step 4: Read Context

Read existing blockers for context:
```bash
bash ~/.aether/aether-utils.sh flag-list --type blocker
```

Read recent activity:
```bash
tail -50 .aether/data/activity.log 2>/dev/null || echo "(no activity log)"
```

Scan recent git commits for context:
```bash
git log --oneline -20 2>/dev/null || echo "(no git history)"
```

### Step 5: Deploy 4 Parallel Scouts

Use the **Task** tool to spawn 4 scouts **in a single message** (parallel execution):

**Scout 1: Git Archaeologist 🏛️**
```
You are the Git Archaeologist scout for swarm {swarm_id}.

PROBLEM: {problem description}

Your mission: Investigate git history to find when this worked and what changed.

Investigation steps:
1. Run `git log --oneline -30` to see recent commits
2. Run `git log -p --since="1 week ago" -- {relevant files}` to see recent changes
3. Run `git blame {suspected file}` if a specific file is mentioned
4. Look for commits that might have introduced the bug

Return JSON:
{
  "scout": "git-archaeologist",
  "confidence": 0.0-1.0,
  "finding": {
    "likely_cause": "What you found",
    "relevant_commits": ["commit hashes"],
    "when_it_broke": "timestamp or commit",
    "evidence": ["specific findings"]
  },
  "suggested_fix": "If obvious from history"
}
```

**Scout 2: Pattern Hunter 🔍**
```
You are the Pattern Hunter scout for swarm {swarm_id}.

PROBLEM: {problem description}

Your mission: Find similar working code in this codebase that solves the same problem.

Investigation steps:
1. Search for similar patterns that work: grep/glob for related code
2. Find how other parts of the codebase handle this
3. Look for test files that demonstrate correct usage
4. Identify patterns that could be applied

Return JSON:
{
  "scout": "pattern-hunter",
  "confidence": 0.0-1.0,
  "finding": {
    "working_examples": ["file:line - description"],
    "applicable_patterns": ["pattern descriptions"],
    "differences": "What's different in broken code"
  },
  "suggested_fix": "Based on working patterns"
}
```

**Scout 3: Error Analyst 💥**
```
You are the Error Analyst scout for swarm {swarm_id}.

PROBLEM: {problem description}

Your mission: Parse the error deeply to identify root cause.

Investigation steps:
1. If stack trace provided, trace through each frame
2. Identify the actual failing line vs where error surfaces
3. Check for common causes: null refs, async issues, type mismatches
4. Look for error handling that might mask the real issue

Return JSON:
{
  "scout": "error-analyst",
  "confidence": 0.0-1.0,
  "finding": {
    "root_cause": "The actual source of the error",
    "error_chain": ["how error propagates"],
    "masked_by": "any error handling hiding the real issue",
    "category": "null-ref|async|type|logic|config|dependency"
  },
  "suggested_fix": "Direct fix for root cause"
}
```

**Scout 4: Web Researcher 🌐**
```
You are the Web Researcher scout for swarm {swarm_id}.

PROBLEM: {problem description}

Your mission: Search external sources for solutions to this exact error.

Investigation steps:
1. Search for the exact error message
2. Look for library/framework documentation
3. Check GitHub issues for similar problems
4. Find Stack Overflow answers

Return JSON:
{
  "scout": "web-researcher",
  "confidence": 0.0-1.0,
  "finding": {
    "known_issue": true/false,
    "documentation_link": "if relevant",
    "similar_issues": ["descriptions of similar problems"],
    "community_solutions": ["approaches others used"]
  },
  "suggested_fix": "From external sources"
}
```

Wait for all 4 scouts to complete.

### Step 6: Collect and Cross-Compare Findings

As each scout returns, add their findings:
```bash
bash ~/.aether/aether-utils.sh swarm-findings-add "{swarm_id}" "{scout_type}" "{confidence}" '{finding_json}'
```

Display each scout's report as they complete:
```
🏛️ Git Archaeologist [{confidence}]
   {summary of finding}

🔍 Pattern Hunter [{confidence}]
   {summary of finding}

💥 Error Analyst [{confidence}]
   {summary of finding}

🌐 Web Researcher [{confidence}]
   {summary of finding}
```

### Step 7: Synthesize Solution

Cross-compare all findings:
1. Identify where scouts agree (high confidence)
2. Note where scouts disagree (investigate further)
3. Weight by confidence scores
4. Prefer findings with concrete evidence

Rank fix options:
```
═══════════════════════════════════════════════
              S O L U T I O N   R A N K I N G
═══════════════════════════════════════════════

#1 [0.85 confidence] {best solution}
   Evidence: {supporting scouts}

#2 [0.72 confidence] {alternative}
   Evidence: {supporting scouts}

#3 [0.45 confidence] {fallback}
   Evidence: {limited support}
```

### Step 8: Apply Best Fix

Select the highest-confidence solution and apply it:

**Command Resolution:** Before running verification, resolve `{build_command}` and `{test_command}` using this priority chain (stop at first match per command):
1. **CLAUDE.md** — Check project CLAUDE.md (in your system context) for explicit build/test commands
2. **CODEBASE.md** — Read `.planning/CODEBASE.md` `## Commands` section
3. **Fallback** — Use project manifest heuristics (e.g., `npm run build`/`npm test` for package.json)

```
🔧 Applying Fix #1...
```

Make the actual code changes using Edit/Write tools.

After applying:
```bash
# Run verification
{build_command} 2>&1 | tail -30
{test_command} 2>&1 | tail -50
```

### Step 9: Verify and Report

**If verification passes:**
```
✅ FIX VERIFIED

Build: PASS
Tests: PASS

🔥🐜🗡️🐜🔥 Swarm successful!

The fix will be confirmed when you run:
  /ant:continue
```

Inject learnings:
- Add FOCUS for the pattern that worked (to constraints.json)
- Add REDIRECT for the anti-pattern that caused the bug (to constraints.json)

Set solution in swarm findings:
```bash
bash ~/.aether/aether-utils.sh swarm-solution-set "{swarm_id}" '{solution_json}'
```

Log success:
```bash
bash ~/.aether/aether-utils.sh activity-log "SWARM_SUCCESS" "Queen" "Swarm {swarm_id} fixed: {brief description}"
```

**If verification fails:**
```
❌ FIX VERIFICATION FAILED

Build: {status}
Tests: {status}

Attempting rollback...
```

Rollback:
```bash
bash ~/.aether/aether-utils.sh autofix-rollback "{checkpoint_type}" "{checkpoint_ref}"
```

Log failure:
```bash
bash ~/.aether/aether-utils.sh activity-log "SWARM_FAILED" "Queen" "Swarm {swarm_id} fix failed verification"
```

Track attempt count. If this is the 3rd failure on the same issue:
```
⚠️ ARCHITECTURAL CONCERN

This problem has resisted 3 swarm attempts.

This suggests:
  - Root cause may be architectural, not implementational
  - Pattern may be fundamentally unsound
  - Different approach needed

Recommended:
  - Review the codebase architecture
  - Consider refactoring vs. patching
  - Create a new phase for structural fix

Swarm will not attempt further fixes on this issue.
```

### Step 10: Cleanup

Archive swarm findings:
```bash
bash ~/.aether/aether-utils.sh swarm-cleanup "{swarm_id}" --archive
```

Display next steps:
```
🐜 Next steps:
   /ant:continue   ⏭️  Verify and advance phase
   /ant:status     📊 View colony status
   /ant:flags      🚩 Check remaining blockers
```
