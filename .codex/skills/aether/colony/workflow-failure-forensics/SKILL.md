---
name: workflow-failure-forensics
description: Use when a phase, build, continue, or colony workflow failed and needs evidence-based post-mortem analysis
type: colony
domains: [forensics, debugging, post-mortem]
agent_roles: [medic, tracker, scout]
workflow_triggers: [medic, continue]
task_keywords: [failure, forensics, post-mortem, what went wrong, unexpected state]
priority: normal
version: "1.0"
---

# Workflow Failure Forensics

## Purpose

When a phase fails, crashes, or produces unexpected results, this skill performs a forensic investigation. It analyzes git history, colony artifacts, pheromone signals, and state files to reconstruct what happened, why it went wrong, and how to prevent it.

## When to Use

- User says "what went wrong?" or "investigate the failure"
- A phase execution failed or produced incorrect results
- Colony is in an unexpected state
- Post-mortem after a failed milestone
- User wants to understand why something didn't work as planned

## Instructions

### 1. Evidence Collection

```
Gather all evidence:
  1. Git log: commits, diffs, and timestamps around failure time
  2. Phase artifacts: PLAN.md, any generated files, partial outputs
  3. Colony state: ROADMAP status, active pheromones, config
  4. Pheromone history: signals emitted before/during failure
  5. Error messages: any crash reports or error outputs
  6. File system state: unexpected files, missing files, corrupt files
  7. Checkpoint data: if worker saved state before failure
```

### 2. Timeline Reconstruction

```
Build a timeline of events:
  {timestamp} -- Phase {N} started
  {timestamp} -- Wave {X} dispatched
  {timestamp} -- Warning pheromone: {message}
  {timestamp} -- Worker reached {budget}%
  {timestamp} -- ERROR: {error message}
  {timestamp} -- Phase marked as FAILED
  
  Identify: Where in the timeline did things diverge from plan?
```

### 3. Root Cause Analysis

```
Analyze failure patterns:

  CONTEXT_EXHAUSTION:
    - Worker ran out of context mid-task
    - Evidence: budget logs showing >90% usage
    - Fix: split phase, use context-budget-manager

  PLAN_AMBIGUITY:
    - Plan was unclear or incomplete
    - Evidence: PLAN.md missing steps, vague descriptions
    - Fix: re-plan with more specificity

  DEPENDENCY_FAILURE:
    - External dependency unavailable or broken
    - Evidence: error messages referencing external systems
    - Fix: add dependency checks, mock unavailable services

  SCOPE_CREEP:
    - Phase tried to do too much
    - Evidence: many files changed beyond plan scope
    - Fix: split into smaller phases

  TOOL_FAILURE:
    - Build tool, test runner, or CI pipeline failed
    - Evidence: error output from tooling
    - Fix: fix tool configuration, update dependencies
```

### 4. Impact Assessment

```
What was affected:
  Files modified before failure: {list}
  Files left in partial state: {list}
  Tests that were added: {list}
  Commits made: {list of hashes}
  
  Is the codebase in a clean state? {yes/no}
  Can work be resumed from checkpoint? {yes/no}
```

### 5. Forensic Report

```
 FORENSIC REPORT -- Phase {N}: {phase_name}
   
   Failure time: {timestamp}
   Duration before failure: {time}
   
   Root cause: {category}
   Description: {detailed explanation}
   
   Timeline:
    {event_1}
    {event_2}
    {event_3}  failure point
   
   Impact:
    {count} files in partial state
    {count} commits to preserve or revert
   
   Recommendations:
   1. {fix_1}
   2. {fix_2}
   3. {prevention_measure}
   
   Recovery options:
   [A] Revert to pre-phase state and re-plan
   [B] Resume from checkpoint (step {N} of PLAN.md)
   [C] Fix the specific failure and continue
   
   Report: .aether/phases/phase-{N}/FORENSICS.md
```

### 6. Prevention Recommendations

```
After diagnosis, suggest preventive measures:
  - Phase size limits (split phases larger than {threshold})
  - Mandatory checkpoint intervals
  - Dependency pre-flight checks
  - Budget monitoring thresholds
  - Plan specificity requirements
```

## Key Patterns

- **Evidence first, conclusions second**: Gather everything before theorizing.
- **Timeline tells the story**: A clear timeline usually reveals the root cause.
- **Don't assign blame to the code**: Focus on process failures, not code failures.
- **Prevention over reaction**: Every forensic report includes prevention measures.

## Output Format

```
 FORENSICS | Phase {N} | Root cause: {category}
   Failed at: {timestamp} | Impact: {files} files
   Recovery: {options}
   Report: FORENSICS.md
```

## Examples

**Context exhaustion:**
> "Root cause: context exhaustion at 94%. Phase 5 had 23 files to modify -- too many for single worker. Recovery: revert to checkpoint, split phase 5 into 5a and 5b."

**Plan ambiguity:**
> "Root cause: PLAN.md had vague step 'implement user flow' without specifying which flow. Worker attempted all flows simultaneously. Fix: re-plan with explicit flow enumeration."
