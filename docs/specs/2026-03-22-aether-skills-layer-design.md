# Aether Skills Layer — Design Spec

**Date:** 2026-03-22
**Status:** Draft
**Version:** Aether v2.1.0 (target)
**Oracle Research:** v2.0.0 system integrity audit (11 iterations, 80% confidence, 54 findings)

---

## 1. Problem Statement

Aether v2.0.0 has 43 slash commands and 22 agents but zero skills. The Oracle audit identified systemic gaps that skills would solve:

- **Autonomous-first, validate-after** — the core workflow runs without user input (only 9/43 commands use AskUserQuestion)
- **Pheromone operations invisible during continue** — signal evolution happens silently
- **Visual inconsistency** — 4+ banner styles, errors as plain text while successes get rich formatting
- **Pheromone protocol gaps** — only 3/22 agents have formal signal handling
- **No domain knowledge injection** — workers have no way to receive best-practice guidance for specific technologies

Skills are reusable process guides that teach agents HOW to approach problems. They sit between commands (what to do) and agents (who does it), governing behaviour and injecting domain expertise.

---

## 2. Architecture Overview

### Two Skill Categories

| Category | Purpose | Location (source) | Location (installed) | Count |
|----------|---------|-------------------|---------------------|-------|
| **Colony skills** | How ants behave within the colony | `.aether/skills/colony/` | `~/.aether/skills/colony/` | 10 |
| **Domain skills** | How ants do technical work | `.aether/skills/domain/` | `~/.aether/skills/domain/` | 18 starter + user-created |

Colony skills are internal operating procedures — interaction patterns, visual standards, pheromone handling. They rarely change.

Domain skills are technical knowledge — Tailwind, React, Python, Docker. These grow over time as users add skills for their stack.

### File Structure

```
.aether/skills/                    # SOURCE (in repo, shipped with npm)
├── colony/
│   ├── colony-interaction/
│   │   └── SKILL.md
│   ├── colony-visuals/
│   │   └── SKILL.md
│   ├── pheromone-visibility/
│   │   └── SKILL.md
│   ├── build-discipline/
│   │   └── SKILL.md
│   ├── colony-lifecycle/
│   │   └── SKILL.md
│   ├── context-management/
│   │   └── SKILL.md
│   ├── state-safety/
│   │   └── SKILL.md
│   ├── error-presentation/
│   │   └── SKILL.md
│   ├── pheromone-protocol/
│   │   └── SKILL.md
│   └── worker-priming/
│       └── SKILL.md
│
└── domain/
    ├── react/
    │   └── SKILL.md
    ├── nextjs/
    │   └── SKILL.md
    ├── tailwind/
    │   └── SKILL.md
    ├── typescript/
    │   └── SKILL.md
    ├── python/
    │   └── SKILL.md
    ├── ... (18 total starter skills)
    └── README.md              # Instructions for creating custom skills

~/.aether/skills/                  # INSTALLED (hub-level, cross-colony)
├── colony/                        # Managed by aether update (overwritten)
│   └── (same as source)
└── domain/                        # Starter set + user-created
    ├── react/                     # Managed by aether update
    ├── tailwind/                  # Managed by aether update
    ├── claude-plugins/            # USER-CREATED (never touched by update)
    └── juice-plugins/             # USER-CREATED (never touched by update)
```

### Update Safety

`aether update` manages skills it ships by tracking ownership:
- Skills in the shipped set are overwritten during update
- User-created skills (any folder not in the shipped manifest) are never modified
- A `.manifest.json` in each category lists Aether-owned skill directories

---

## 3. Skill File Format

### Frontmatter

```yaml
---
name: skill-identifier
description: Use when [triggering condition]
type: colony | domain
domains: [tag1, tag2, tag3]
agent_roles: [builder, watcher, scout]
detect_files: ["tailwind.config.*", "*.tsx"]       # Glob patterns for file existence
detect_packages: ["tailwindcss", "@tailwind/ui"]   # Package names in manifests
priority: high | normal | low
version: "1.0"
---
```

| Field | Required | Purpose |
|-------|----------|---------|
| `name` | Yes | Unique identifier, letters/numbers/hyphens only |
| `description` | Yes | Triggering condition (max 1024 chars), starts with "Use when..." |
| `type` | Yes | `colony` (behaviour) or `domain` (technical knowledge) |
| `domains` | Yes | Tags for smart matching against tasks and pheromones |
| `agent_roles` | Yes | Which worker types can receive this skill |
| `detect_files` | No | Glob patterns — skill matches if any file exists (domain skills only) |
| `detect_packages` | No | Package names — checked in package.json, requirements.txt, go.mod, Gemfile, etc. |
| `priority` | No | Tie-breaking when multiple skills match equally (default: normal) |
| `version` | No | Skill version for change tracking (default: "1.0") |

### Frontmatter Parsing

Frontmatter is parsed using a simple line-by-line bash parser (not full YAML):
- Key-value pairs: `name: value`
- Arrays: `[item1, item2]` parsed by stripping brackets and splitting on comma
- No nested objects, no multi-line values
- This matches the complexity level of existing agent definition frontmatter in `.claude/agents/ant/`

### Body

The body is markdown — the full process guide or technical reference. When loaded, the entire body is injected into the worker's context. Same format and behaviour as superpowers skills.

Skills can include supporting files in their directory (examples, reference docs, templates) that the main SKILL.md can reference.

---

## 4. Skill Discovery & Assignment

### Flow (per worker spawn)

```
Step 1: Index Build (once per colony-prime assembly)
  ├── Scan ~/.aether/skills/colony/*/SKILL.md
  ├── Scan ~/.aether/skills/domain/*/SKILL.md
  ├── Read frontmatter only (name, type, domains, agent_roles, detect)
  └── Build in-memory index

Step 2: Codebase Detection (once per colony-prime assembly)
  ├── Scan repo for config files, package manifests, file extensions
  ├── Match against detect patterns in domain skills
  └── Score each domain skill by detection confidence

Step 3: Smart Matching (per worker)
  ├── Filter by agent_roles (builder only gets builder-tagged skills)
  ├── Colony skills: top 3 matching roles loaded (scored by domain relevance)
  ├── Domain skills: top 3 scored by:
  │   ├── detect pattern matches (highest weight)
  │   ├── domain overlap with active pheromone signals
  │   ├── domain overlap with task description
  │   └── priority field (tie-breaker)
  └── Select top 2-3 domain skills by score

Step 4: Full Injection
  ├── Load complete SKILL.md content for matched skills
  ├── Inject into worker prompt alongside pheromone signals
  └── Worker follows skills exactly
```

### Pheromone-Skill Interaction

Pheromones influence skill selection:
- A FOCUS on "security" boosts skills with `security` in their domains
- A REDIRECT on "no inline styles" boosts `tailwind` or `css` skills
- A FEEDBACK about "prefer functional components" boosts `react` skill relevance

This is additive — pheromones don't replace skills, they influence which ones get loaded.

### Budget Considerations

Skills are injected OUTSIDE of colony-prime as a separate step during worker spawn (see Section 8). This avoids modifying colony-prime's existing 9-section budget.

**Skill injection is a separate budget:**
- Colony skills: max 3 per worker (role-filtered), typically 500-2000 chars each
- Domain skills: max 3 per worker (top-scored), typically 1000-3000 chars each
- Total skill budget per worker: 12,000 chars max
- If over budget, domain skills are trimmed first, then colony skills by reverse priority

**Trim order within skills (first trimmed → last trimmed):**
1. Domain skills (lowest priority — trimmed first)
2. Colony skills with `priority: low`
3. Colony skills with `priority: normal`
4. Colony skills with `priority: high` (highest priority — trimmed last)

This is separate from colony-prime's own trim cascade, which remains unchanged.

---

## 5. Colony Skills (10)

### 5.1 colony-interaction

**Problem:** Core workflow runs autonomously with nearly zero user input.

**Domains:** `[interaction, ux, workflow]`
**Agent roles:** `[builder, watcher, route_setter, architect]`

**What it teaches:**
- At key decision points, stop and ask the user a multiple-choice question (AskUserQuestion with 2-4 options)
- The user wants to click options, not type — keep questions plain English, options short
- Mandatory touchpoints: before committing a plan, before starting a build wave, after verification, before advancing phases
- Each option must explain what happens if selected
- Never proceed past a major decision without user confirmation

### 5.2 colony-visuals

**Problem:** 4+ banner styles, inconsistent progress bars, `print-standard-banner` exists but unused.

**Domains:** `[visuals, ux, formatting]`
**Agent roles:** `[builder, watcher, chronicler, scout]`

**What it teaches:**
- Spaced-letter banner format consistently: `━━ T I T L E ━━`
- Progress bars use `generate-progress-bar` utility only
- All output blocks wrapped in `━━━` dividers
- Emoji: one per section header, never in body text
- Every command output: header banner, content, "Next Up" footer

### 5.3 pheromone-visibility

**Problem:** Pheromone operations silent during continue/advance.

**Domains:** `[pheromones, visibility, ux]`
**Agent roles:** `[builder, watcher, scout]`

**What it teaches:**
- Every pheromone operation (create, reinforce, expire, auto-emit) produces a user-visible line
- Format: `🎯 FOCUS emitted: "security" [85%]`
- End of continue: show "Signal Changes" summary table
- Never let pheromone evolution happen invisibly

### 5.4 build-discipline

**Problem:** Workers need consistent implementation standards.

**Domains:** `[building, testing, quality, implementation]`
**Agent roles:** `[builder]`

**What it teaches:**
- Check midden (recent failures) before starting — don't repeat known mistakes
- Respect all REDIRECT signals as hard constraints
- Write tests before implementation where possible
- Log failures to midden before trying alternative approaches
- Stay on task — reference the phase plan, don't drift

### 5.5 colony-lifecycle

**Problem:** `print-next-up()` missing SEALED/COMPLETED state, 2 dead-end commands.

**Domains:** `[lifecycle, routing, workflow, state]`
**Agent roles:** `[builder, watcher, route_setter, architect]`

**What it teaches:**
- Every command output ends with "Next Up" block
- State transitions: IDLE → READY → PLANNING → EXECUTING → SEALED → ENTOMBED → IDLE
- After seal → entomb. After entomb → init
- Never leave the user at a dead end

### 5.6 context-management

**Problem:** Only 5 commands update CONTEXT.md, session-verify-fresh used by 5/43.

**Domains:** `[context, session, recovery, state]`
**Agent roles:** `[builder, watcher, scout, architect]`

**What it teaches:**
- After state-changing operations, update CONTEXT.md via context-update
- Verify file freshness before reading state files
- HANDOFF.md must include: goal, phase, signals, blockers, next step
- If HANDOFF.md missing during resume, fall back to COLONY_STATE.json + CONTEXT.md
- Prompt users to clear context at natural break points

### 5.7 state-safety

**Problem:** Only 4/10+ JSON write paths use atomic_write, no corruption detection.

**Domains:** `[security, state, data-integrity, safety]`
**Agent roles:** `[builder, watcher]`

**What it teaches:**
- All JSON state mutations use atomic_write (temp file → rename)
- Acquire file lock before state file modifications
- Validate JSON after writes (`jq . file > /dev/null`)
- On corrupted reads, log warning and fall back to backup
- Never silently swallow JSON parse errors

### 5.8 error-presentation

**Problem:** Successes get rich formatting, errors get plain text.

**Domains:** `[errors, ux, formatting, diagnostics]`
**Agent roles:** `[builder, watcher, scout]`

**What it teaches:**
- Errors get same visual treatment as successes
- Error format: banner, plain-English description, what's being done about it, what user can do
- Never show raw logs or stack traces to the user
- Group related errors, don't dump individually
- Always end error output with actionable next steps

### 5.9 pheromone-protocol

**Problem:** Only 3/22 agents have formal pheromone handling.

**Domains:** `[pheromones, protocol, signals, compliance]`
**Agent roles:** All 22 agent roles

**What it teaches:**
- REDIRECT = hard constraint, must not violate, verify compliance
- FOCUS = prioritise this area, extra attention and thoroughness
- FEEDBACK = preference/adjustment, incorporate where natural
- Role-specific adaptations (builder: more tests; scout: investigate first)
- Report signal compliance in output

### 5.10 worker-priming

**Problem:** Workers don't know what's in their assembled context or why sections may be missing.

**Domains:** `[context, priming, awareness]`
**Agent roles:** All 22 agent roles

**What it teaches:**
- Your context includes 9 sections: rolling summary, phase learnings, key decisions, hive wisdom, context capsule, user preferences, QUEEN wisdom, blocker warnings, pheromone signals
- Missing sections may have been trimmed for budget — normal, not an error
- Check pheromone signals at end of context before starting work
- User preferences override general patterns
- Hive wisdom = learned patterns from other colonies, not absolute rules

---

## 6. Starter Domain Skills (18)

### Frontend

| Skill | Detect | Domains |
|-------|--------|---------|
| `react` | Files: `*.jsx`, `*.tsx` / Packages: `react` | `[frontend, components, ui]` |
| `nextjs` | Files: `next.config.*` / Packages: `next` | `[frontend, ssr, fullstack]` |
| `vue` | Files: `*.vue`, `vue.config.*` / Packages: `vue` | `[frontend, components, ui]` |
| `tailwind` | Files: `tailwind.config.*` / Packages: `tailwindcss` | `[css, frontend, styling]` |
| `html-css` | Files: `*.html`, `*.css`, `*.scss` | `[frontend, styling, markup]` |
| `svelte` | Files: `*.svelte`, `svelte.config.*` / Packages: `svelte` | `[frontend, components, ui]` |

### Backend

| Skill | Detect | Domains |
|-------|--------|---------|
| `nodejs` | Files: `*.js`, `*.mjs` / Packages: `express`, `fastify` | `[backend, server, api]` |
| `python` | Files: `*.py`, `requirements.txt`, `pyproject.toml` | `[backend, scripting, data]` |
| `django` | Files: `manage.py` / Packages: `django` | `[backend, python, web]` |
| `rails` | Files: `Gemfile`, `config/routes.rb` | `[backend, ruby, web]` |
| `golang` | Files: `go.mod`, `*.go` | `[backend, systems, concurrency]` |

### Data & APIs

| Skill | Detect | Domains |
|-------|--------|---------|
| `postgresql` | Files: `*.sql` / Packages: `pg`, `psycopg2` | `[database, sql, data]` |
| `rest-api` | Files: `openapi.*`, `swagger.*` | `[api, http, design]` |
| `graphql` | Files: `*.graphql` / Packages: `apollo`, `graphql` | `[api, query, schema]` |
| `prisma` | Files: `schema.prisma` / Packages: `@prisma/client` | `[database, orm, schema]` |

### Infrastructure

| Skill | Detect | Domains |
|-------|--------|---------|
| `docker` | Files: `Dockerfile*`, `docker-compose.*` | `[infrastructure, containers, deployment]` |
| `typescript` | Files: `tsconfig.json`, `*.ts` | `[typing, language, safety]` |
| `testing` | Files: `*.test.*`, `*.spec.*` / Packages: `jest`, `vitest`, `pytest` | `[testing, quality, tdd]` |

---

## 7. Skill Creation Wizard (`/ant:skill-create`)

A new slash command that creates domain skills using Oracle research.

### Flow

```
/ant:skill-create "tailwind"

Step 1: Oracle Research
  ├── Web search for Tailwind best practices, common pitfalls
  ├── Codebase scan for existing Tailwind usage patterns
  └── Compile findings (5-10 iteration mini-research)

Step 2: Wizard Questions (AskUserQuestion)
  ├── What aspect to focus on? (utility-first patterns / responsive / theming / all)
  ├── Experience level? (beginner guidance / intermediate patterns / advanced optimization)
  └── Any specific rules? (free text for custom constraints)

Step 3: Generate SKILL.md
  ├── Frontmatter from wizard answers + Oracle findings
  ├── Body from Oracle research synthesis
  └── Write to ~/.aether/skills/domain/{name}/SKILL.md

Step 4: Review
  ├── Show the generated skill to user
  ├── Ask: "Does this look right?"
  └── Iterate if needed
```

### Output Location

User-created skills go to `~/.aether/skills/domain/` and are never touched by `aether update`.

---

## 8. Integration Points

### Skill Injection Architecture

**Skills are injected OUTSIDE colony-prime, not inside it.**

Colony-prime remains unchanged — it still assembles its 9 existing sections (rolling summary, phase learnings, key decisions, hive wisdom, context capsule, user preferences, QUEEN wisdom, blocker warnings, pheromone signals) within its own 8K/4K budget.

Skill injection happens as a separate step during worker spawn in `build-wave.md`:

```
colony-prime assembles prompt_section (unchanged, 9 sections)
    ↓
skill-match runs per worker (role + task + pheromones → matched skills)
    ↓
skill-inject loads matched SKILL.md content (separate 12K budget)
    ↓
Worker prompt = task + prompt_section + skill_section
```

This keeps colony-prime clean and means skills can be added/removed without touching the core context assembly pipeline.

### New aether-utils.sh Subcommands

| Subcommand | Purpose |
|------------|---------|
| `skill-index` | Scan `~/.aether/skills/`, read frontmatter, build index |
| `skill-detect` | Scan repo for config files/packages, match against detect patterns |
| `skill-match` | Given worker role + task + pheromones, return top matched skills |
| `skill-inject` | Load full SKILL.md content for matched skills, format for prompt injection |
| `skill-list` | List all installed skills with type, domains, and detection status |
| `skill-manifest` | Read/update .manifest.json for update safety |
| `skill-diff` | Compare user skill with Aether-shipped version |
| `skill-cache-rebuild` | Rebuild `~/.aether/skills/.index.json` from all SKILL.md frontmatter |

### Index Cache

To support large libraries (100+ skills) without slow startup:
- `skill-index` reads from `~/.aether/skills/.index.json` (cached index)
- Cache is rebuilt when any SKILL.md has a newer mtime than the cache file
- Cache stores: name, type, domains, agent_roles, detect_files, detect_packages, priority, file_path
- `skill-cache-rebuild` forces a full rebuild

### New Slash Command

| Command | Purpose |
|---------|---------|
| `/ant:skill-create "<topic>"` | Skill creation wizard with Oracle research |

### Build Pipeline Changes

**In `build-context.md` (runs once per build):**
1. Call `skill-index` to build/read cached index
2. Call `skill-detect` to scan codebase against detect patterns
3. Store detection results as cross-stage state

**In `build-wave.md` (runs per worker spawn):**
4. Call `skill-match` with worker role + task + pheromones + detection results
5. Call `skill-inject` for matched skills (loads full SKILL.md content)
6. Append skill_section to worker prompt after prompt_section

### Distribution Changes

`package.json` includes `.aether/skills/` in the npm package.
`setupHub()` in the post-install script copies skills to `~/.aether/skills/`, respecting the manifest (never overwriting user-created skills).

---

## 9. Update Safety Design

### Manifest-Based Ownership

Each skill category has a `.manifest.json`:

```json
{
  "managed_by": "aether",
  "version": "2.1.0",
  "skills": [
    "colony-interaction",
    "colony-visuals",
    "pheromone-visibility",
    "build-discipline",
    "colony-lifecycle",
    "context-management",
    "state-safety",
    "error-presentation",
    "pheromone-protocol",
    "worker-priming"
  ]
}
```

During `aether update`:
1. Read `.manifest.json` from installed location
2. Only overwrite directories listed in the manifest
3. Any directory NOT in the manifest is user-created — skip it
4. Update the manifest with any new skills added in the update

### User-Created Skill Protection

- User-created skills have no entry in `.manifest.json`
- `aether update` never modifies, moves, or deletes them
- If a user creates a skill with the same name as a future Aether skill, the user's version takes precedence (no overwrite). `aether update` logs a notice: "Skipped skill '{name}' — user version exists. Run `aether skill-diff {name}` to compare."

---

## 10. Success Criteria

The skills layer is working when:

1. Colony-prime automatically detects which domain skills are relevant based on the codebase
2. Workers receive matched skills in their context and follow them
3. Pheromone signals influence which skills get loaded
4. Users can create custom skills via `/ant:skill-create` with Oracle research
5. `aether update` installs/updates shipped skills without touching user-created ones
6. The 10 colony skills resolve the Oracle's UX findings (more interaction, visible pheromones, consistent visuals, error formatting parity)
7. The system scales — 200 skills in the library doesn't slow down matching or bloat worker prompts

---

## 11. Out of Scope

- Skill marketplace / community sharing (future enhancement)
- Skill versioning / changelogs (skills are updated with Aether releases)
- Skill dependencies (skills are independent units)
- Skill testing framework (future — could adapt superpowers' pressure test approach)
- OpenCode skill parity (future — `.opencode/skills/` mirror)

---

*Design approved in conversation on 2026-03-22. Ready for implementation planning.*
