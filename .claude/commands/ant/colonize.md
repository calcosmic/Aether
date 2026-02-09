---
name: ant:colonize
description: 🔍🐜🗺️🐜🔍 Analyze codebase and prepare for colony work
---

You are the **Queen**. Perform initial codebase analysis.

## Instructions

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.

If `goal: null` -> output "No colony initialized. Run /ant:init first.", stop.
If `plan.phases` not empty -> output "Colony already has phases. Use /ant:continue.", stop.

### Step 2: Surface Scan

Use Glob to find key files (read up to 20 total).

**Package manifests:**
- package.json, Cargo.toml, pyproject.toml, go.mod, Gemfile, pom.xml, build.gradle

**Documentation:**
- README.md, README.*, docs/README.md

**Entry points:**
- src/index.*, src/main.*, main.*, app.*, lib/index.*, index.*

**Config:**
- tsconfig.json, .eslintrc.*, jest.config.*, vite.config.*, webpack.config.*

Read found files. Extract:
- Tech stack (language, framework, key dependencies)
- Entry points (main files)
- Key directories (src/, lib/, tests/, etc.)
- File counts per top-level directory

### Step 3: Write CODEBASE.md

Create `.planning/CODEBASE.md` (ensure `.planning/` exists first):

```markdown
# Codebase Overview

**Stack:** <language> + <framework>
**Entry:** <main entry points>

**Structure:**
- <dir>/ (<count> files)
- ...

**Key Dependencies:**
- <dep1>: <purpose>
- <dep2>: <purpose>
- ...

**Test Location:** <tests/ or __tests__/ or similar>

**Notes:**
- <any notable patterns or conventions observed>
```

Keep output under 50 lines. Focus on what's relevant to the colony goal.

### Step 4: Update State

Read `.aether/data/COLONY_STATE.json`. Update:
- Set `state` to `"IDLE"` (ready for planning)

Write Event: Append to the `events` array as pipe-delimited string:
`"<ISO-8601 UTC>|codebase_colonized|colonize|Codebase analyzed: <primary language/framework>"`

If the `events` array exceeds 100 entries, remove the oldest entries to keep only 100.

Write the updated COLONY_STATE.json.

### Step 5: Confirm

Output header:

```
🔍🐜🗺️🐜🔍 ═══════════════════════════════════════════════════
   C O D E B A S E   A N A L Y S I S
═══════════════════════════════════════════════════ 🔍🐜🗺️🐜🔍
```

Then output:

```
Codebase analysis complete.
See: .planning/CODEBASE.md

Stack: <language> + <framework>
Entry: <main entry point>
Files: <total count> across <N> directories

Next:
  /ant:plan              Generate project plan
  /ant:focus "<area>"    Inject focus before planning
  /ant:redirect "<pat>"  Inject constraint before planning
```
