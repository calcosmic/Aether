---
name: phase-context-gathering
description: Use when a phase needs implementation context, file inventory, or gray-area analysis before work proceeds
type: colony
domains: [analysis, codebase-intelligence, requirements]
agent_roles: [scout, architect, watcher]
workflow_triggers: [plan, build]
task_keywords: [context, gray area, insufficient, unfamiliar, dependencies]
priority: normal
version: "1.0"
---

# Phase Context Gathering

## Purpose
Adaptively gathers implementation context by scanning the codebase, detecting gray areas (underspecified regions), and deep-diving into each. Produces a CONTEXT.md that gives builders everything they need without ambiguity.

## When to Use
- A builder reports "insufficient context" for a task
- The architect identifies gray areas during plan review
- `aether build` fails due to missing information about existing code
- A phase touches unfamiliar parts of the codebase
- Before planning a phase that has dependencies on existing code
- When onboarding a new builder agent to an ongoing colony

## Instructions

### Step 1 -- Define Context Scope
1. Read the target phase description from `.aether/roadmap.md`.
2. Read the SPEC.md and PLAN.md (if they exist) for the phase.
3. Extract keywords, file patterns, module names, and API references from the phase description.
4. List the context areas to investigate:
   - **Direct dependencies**: files and modules the phase must modify or import
   - **Upstream interfaces**: APIs or data structures the phase consumes
   - **Downstream consumers**: code that depends on what this phase produces
   - **Conventions**: coding patterns, naming schemes, and architectural decisions in nearby code
   - **Test patterns**: how existing tests are structured in relevant areas

### Step 2 -- Broad Scan
Execute a parallel scan across the codebase:

```
For each context area:
  1. Glob for files matching keywords/patterns
  2. Grep for relevant function names, types, or constants
  3. Read top-level directory structure of related modules
  4. Check for existing tests in the area
```

Record findings in a structured inventory:
```
area: {name}
files_found: [{paths}]
relevant_patterns: [{patterns found}]
test_coverage: {exists | partial | missing}
confidence: {high | medium | low}
```

### Step 3 -- Gray Area Detection
Identify gray areas where information is incomplete:

1. **Missing implementations**: Referenced but not found (e.g., an import that resolves to nothing)
2. **Undocumented interfaces**: Functions/classes with no type signatures or comments
3. **Ambiguous connections**: Two modules that seem related but the relationship is unclear
4. **Dead code**: Code that might be superseded -- unclear if it should be removed or extended
5. **Convention conflicts**: Multiple patterns for the same thing (e.g., two error-handling approaches)

Score each gray area:
```
severity: {blocking | significant | minor}
blocking: cannot proceed without resolving
significant: could cause rework if wrong assumption made
minor: unlikely to affect outcome
```

### Step 4 -- Deep Dive
For each blocking or significant gray area:

1. Read the surrounding code (50 lines around each reference)
2. Trace imports and function call chains (up to 3 hops)
3. Check git history for recent changes in the area (`git log --oneline -10 -- {path}`)
4. Look for documentation files, README sections, or ADRs related to the area
5. Check for TODO/FIXME/HACK comments that might indicate known issues

For each resolved gray area, record:
```
area: {name}
finding: {what was discovered}
resolution: {how to handle it}
confidence: {high | medium | low}
source: {file:line or git:hash}
```

### Step 5 -- Unresolvable Areas
If a gray area cannot be resolved through codebase investigation:
1. Document it explicitly as an "Open Question"
2. Assign a default approach with rationale
3. Flag it for queen/architect review
4. Include the impact if the default approach is wrong

### Step 6 -- Write CONTEXT.md
Produce the context document:

```markdown
# Phase {N} Context: {Title}

## Summary
{2-3 sentence overview of the codebase landscape relevant to this phase}

## File Inventory
| Area | Files | Status | Confidence |
|------|-------|--------|------------|
| {area} | {paths} | {status} | {H/M/L} |

## Key Interfaces
### {Interface Name}
- Location: {file:line}
- Signature: {type signature or API shape}
- Purpose: {what it does}
- Used by: {consumers}

## Established Patterns
### {Pattern Name}
- Where used: {files/modules}
- Example: {code snippet or reference}
- Applies to: {what this phase should follow}

## Gray Areas Resolved
### {Area Name}
- Finding: {what was discovered}
- Resolution: {what builders should do}
- Confidence: {H/M/L}
- Source: {file:line}

## Open Questions
### {Question}
- Impact if wrong: {consequence}
- Default approach: {recommended path}
- Needs: {queen | architect} review

## Test Landscape
- Framework: {test runner and libraries}
- Pattern: {how tests are organized}
- Coverage in area: {assessment}
- Existing fixtures: {relevant test data factories}
```

## Key Patterns

### Import Tracing
When you encounter an unfamiliar import, follow it: read the source file, check its exports, and understand its role. Never assume an import does what its name suggests -- verify.

### Convention Inference
Look at 3-5 nearby files to infer conventions. A single file might be an outlier. Consistent patterns across multiple files indicate a real convention.

### Confidence Calibration
- **High**: Multiple confirming sources (code + tests + docs)
- **Medium**: Code exists but no tests/docs to confirm intent
- **Low**: Only one reference found, could be deprecated or incidental

## Output Format
- `.aether/phases/{phase}/CONTEXT.md` -- the gathered context
- Updates to `.aether/phases/{phase}/state.md` with `context_gathered: true`

## Examples

### Example 1 -- API Endpoint Phase
Phase requires adding new REST endpoints. Context gatherer scans:
1. Existing route files -> finds `/src/routes/` directory with Express router pattern
2. Middleware -> discovers auth middleware at `/src/middleware/auth.ts`
3. Validation -> finds Joi schema pattern in existing endpoints
4. Tests -> discovers `/tests/routes/` with supertest-based integration tests

CONTEXT.md documents the Express+Joi+supertest stack, the auth middleware contract, and the test file naming convention. No gray areas -- confidence high.

### Example 2 -- Gray Area Discovery
Phase requires "email notifications." Context gatherer finds:
1. An old email utility in `/src/utils/email.ts` (undocumented)
2. A newer notification service in `/src/services/notifications.ts`
3. Both are imported in different modules

Gray area: which email system should the phase use? Deep dive reveals the utility is legacy (last commit 6 months ago) and the service is current (active development). Resolution: use the notification service, document the legacy utility as deprecated. Open question flagged if the legacy utility should be removed.

### Example 3 -- Scout Report for New Builder
New builder agent joins colony for phase 7. Context gatherer produces a comprehensive CONTEXT.md covering: project architecture overview, directory structure, key module responsibilities, testing conventions, and a list of the 5 most recently modified files in the phase scope. Builder starts with full situational awareness.
