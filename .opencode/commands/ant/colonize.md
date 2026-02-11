---
name: ant:colonize
description: "🔍🐜🗺️🐜🔍 Analyze codebase and prepare for colony work"
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

### Step 2.5: Command Detection

Detect build, test, type-check, and lint commands from two sources. Track each command with its source attribution (`claude_md` or `heuristic`).

**Source 1 — CLAUDE.md (priority):**

Read `CLAUDE.md` in the project root. If it does not exist, skip to Source 2.
Scan for commands under headings matching: `Commands`, `Scripts`, `Development`, `Build`, `Testing`, `Lint`, or similar.
Also extract inline code blocks containing patterns like `npm`, `npx`, `yarn`, `pnpm`, `cargo`, `go`, `pytest`, `make`, `gradle`, `mvn`.
For each command found, store: `{ label, command, source: "claude_md" }`.

**Source 2 — Heuristic from package manifests:**

Using the manifests found in Step 2, infer commands with this table:

| Manifest | Field/Pattern | Label | Command |
|---|---|---|---|
| package.json | `scripts.test` | test | `npm test` |
| package.json | `scripts.build` | build | `npm run build` |
| package.json | `scripts.lint` | lint | `npm run lint` |
| package.json | `scripts.typecheck` or `scripts.type-check` | typecheck | `npm run typecheck` |
| Cargo.toml | (exists) | test | `cargo test` |
| Cargo.toml | (exists) | build | `cargo build` |
| Cargo.toml | `clippy` in deps | lint | `cargo clippy` |
| pyproject.toml | `[tool.pytest]` or pytest in deps | test | `pytest` |
| pyproject.toml | `[tool.ruff]` or ruff in deps | lint | `ruff check .` |
| pyproject.toml | `[tool.mypy]` or mypy in deps | typecheck | `mypy .` |
| go.mod | (exists) | test | `go test ./...` |
| go.mod | (exists) | build | `go build ./...` |

For each inferred command, only store it if no command with the same label was already found from CLAUDE.md (CLAUDE.md wins per-label). Store as: `{ label, command, source: "heuristic" }`.

If neither source yields any commands, set the detected commands list to empty.

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

## Commands
<for each detected command from Step 2.5>
- **<label>**: `<command>` (<source: claude_md | heuristic>)
<if no commands detected>
No build system detected.
</if>

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

### Step 5.5: Suggest Commands for CLAUDE.md

Skip if all commands came from `claude_md` or none were detected. This is **non-blocking** -- do not edit CLAUDE.md automatically.

For heuristic-sourced commands only, output:

```
💡 Detected commands not yet in CLAUDE.md. Consider adding:
```

Then a fenced code block the user can copy-paste into CLAUDE.md:

```markdown
## Commands
- <label>: `<command>`
```

Then: `Paste the above into your project's CLAUDE.md to skip heuristic detection next time.`
