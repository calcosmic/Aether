---
name: ant:swarm
description: "🔥🐜🗡️🐜🔥 Real-time colony swarm display + stubborn bug destroyer"
---

You are the **Queen Ant Colony**. Deploy the swarm to destroy a stubborn bug or view real-time colony activity.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

### Quick View Mode (No Arguments)

If `$normalized_args` is empty or equals "--watch":

Run the real-time swarm display:
```bash
bash .aether/utils/swarm-display.sh
```

This shows:
- Active ants with caste colors and emojis (🔨 Builder in blue, etc.)
- Tool usage stats per ant (📖5 🔍3 ✏️2 ⚡1)
- Trophallaxis metrics (🍯 token consumption)
- Timing information (elapsed time per ant)
- Chamber activity map (which nest zones have active ants)
- Animated status phrases ("excavating...", "foraging...")

Display updates automatically as ants start/complete work.
Press Ctrl+C to exit.

### Bug Destruction Mode (With Arguments)

The problem to investigate is: `$normalized_args`

#### Step 1: Validate Input

If `$normalized_args` is empty:
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

#### Step 2: Read State & Initialize

Read `.aether/data/COLONY_STATE.json`.
If `goal` is null → "No colony initialized. Run /ant:init first.", stop.

Generate swarm ID: `swarm-<unix_timestamp>`

Initialize swarm findings:
```bash
bash .aether/aether-utils.sh swarm-findings-init "<swarm_id>"
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

#### Step 3: Create Git Checkpoint

Before any investigation that might lead to fixes:
```bash
bash .aether/aether-utils.sh autofix-checkpoint "pre-swarm-$SWARM_ID"
```

Store the result for potential rollback:
- `checkpoint_type` = result.type ("stash", "commit", or "none")
- `checkpoint_ref` = result.ref

```
💾 Checkpoint: {checkpoint_type} → {checkpoint_ref}
```

#### Step 4: Read Context

Read existing blockers for context:
```bash
bash .aether/aether-utils.sh flag-list --type blocker
```

Read recent activity:
```bash
tail -50 .aether/data/activity.log 2>/dev/null || echo "(no activity log)"
```

Scan recent git commits for context:
```bash
git log --oneline -20 2>/dev/null || echo "(no git history)"
```

#### Step 5: Deploy 4 Parallel Scouts

Use the **Task** tool to spawn 4 scouts **in a single message** (parallel execution):

**Scout 1: 🏛️ Git Archaeologist**
```
You are {swarm_id}-Archaeologist, a 🏛️ Scout Ant.

Investigate git history for: {problem description}

Steps:
1. Run `git log --oneline -30`
2. Run `git log -p --since="1 week ago" -- {relevant files}`
3. Run `git blame {suspected file}` if mentioned
4. Find commits that introduced the bug

Return ONLY this JSON:
{"scout": "git-archaeologist", "confidence": 0.0-1.0, "finding": {"likely_cause": "...", "relevant_commits": [], "when_it_broke": "...", "evidence": []}, "suggested_fix": "..."}
```

**Scout 2: 🔍 Pattern Hunter**
```
You are {swarm_id}-PatternHunter, a 🔍 Scout Ant.

Find working patterns for: {problem description}

Steps:
1. Grep/glob for related working code
2. Find how other parts handle this
3. Look for test files showing correct usage
4. Identify applicable patterns

Return ONLY this JSON:
{"scout": "pattern-hunter", "confidence": 0.0-1.0, "finding": {"working_examples": [], "applicable_patterns": [], "differences": "..."}, "suggested_fix": "..."}
```

**Scout 3: 💥 Error Analyst**
```
You are {swarm_id}-ErrorAnalyst, a 🔍 Scout Ant.

Analyze error: {problem description}

Steps:
1. Trace through stack trace frames
2. Identify actual failing line vs surface error
3. Check for null refs, async issues, type mismatches
4. Look for error handling masking the issue

Return ONLY this JSON:
{"scout": "error-analyst", "confidence": 0.0-1.0, "finding": {"root_cause": "...", "error_chain": [], "masked_by": "...", "category": "null-ref|async|type|logic|config|dependency"}, "suggested_fix": "..."}
```

**Scout 4: 🌐 Web Researcher**
```
You are {swarm_id}-WebResearcher, a 🔍 Scout Ant.

Research external solutions for: {problem description}

Steps:
1. Search for exact error message
2. Find library/framework docs
3. Check GitHub issues
4. Find Stack Overflow answers

Return ONLY this JSON:
{"scout": "web-researcher", "confidence": 0.0-1.0, "finding": {"known_issue": true/false, "documentation_link": "...", "similar_issues": [], "community_solutions": []}, "suggested_fix": "..."}
```

Wait for all 4 scouts to complete.

#### Step 6: Collect and Cross-Compare Findings

As each scout returns, add their findings:
```bash
bash .aether/aether-utils.sh swarm-findings-add "{swarm_id}" "{scout_type}" "{confidence}" '{finding_json}'
```

Display each scout's report as they complete:
```
🏛️ Archaeologist [{confidence}]
   {summary of finding}

🔍 PatternHunter [{confidence}]
   {summary of finding}

💥 ErrorAnalyst [{confidence}]
   {summary of finding}

🌐 WebResearcher [{confidence}]
   {summary of finding}
```

#### Step 7: Synthesize Solution

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

#### Step 8: Apply Best Fix

Select the highest-confidence solution and apply it:

**Command Resolution:** Before running verification, resolve `{build_command}` and `{test_command}` using this priority chain (stop at first match per command):
1. **CLAUDE.md** — Check project CLAUDE.md (in your system context) for explicit build/test commands
2. **CODEBASE.md** — Read `.aether/data/codebase.md` `## Commands` section
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

#### Step 9: Verify and Report

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
bash .aether/aether-utils.sh swarm-solution-set "{swarm_id}" '{solution_json}'
```

Log success:
```bash
bash .aether/aether-utils.sh activity-log "SWARM_SUCCESS" "Queen" "Swarm {swarm_id} fixed: {brief description}"
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
bash .aether/aether-utils.sh autofix-rollback "{checkpoint_type}" "{checkpoint_ref}"
```

Log failure:
```bash
bash .aether/aether-utils.sh activity-log "SWARM_FAILED" "Queen" "Swarm {swarm_id} fix failed verification"
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

#### Step 10: Cleanup

Archive swarm findings:
```bash
bash .aether/aether-utils.sh swarm-cleanup "{swarm_id}" --archive
```

Display next steps:
```
🐜 Next steps:
   /ant:continue   ⏭️  Verify and advance phase
   /ant:status     📊 View colony status
   /ant:flags      🚩 Check remaining blockers
```
