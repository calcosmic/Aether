# 

> Aether Colony -- Codex CLI Instructions

This file provides project-level instructions for Codex CLI (equivalent to
CLAUDE.md for Claude Code). Aether manages this colony.

Colony goal: 

---

## How Aether Works in Codex CLI

Pick an ant skill in Codex to start the matching workflow. The skill reads Aether's
instructions; the `aether` executable still owns state, checks, and authorization.

Aether's Codex actions use skills, with exactly nine public names:

| Skill | Workflow |
|-------|----------|
| `$ant-init` | Start a colony |
| `$ant-discuss` | Clarify intent |
| `$ant-oracle` | Research a concern |
| `$ant-colonize` | Survey existing code |
| `$ant-plan` | Plan phases |
| `$ant-build` | Build a phase |
| `$ant-continue` | Verify and advance |
| `$ant-swarm` | Route work or watch workers |
| `$ant-seal` | Seal and retain work for review |

The remaining 55 action skills and native-worker parity belong to later inserted
phases. These nine names establish entrypoints, not complete lifecycle parity.

The selected installation root is `~/.codex/skills/aether/`, qualified with Codex
CLI 0.154.0. Each public `ant-<command>/SKILL.md` reads private support relative to
its own installed file, not the working directory:

- `support/aether-colony-creation.md`
- `support/aether-colony-research.md`
- `support/aether-colony-build-cycle.md`

These three ordinary files are private support, not public helper skills. Worker
skills still arrive automatically in runtime dispatch briefs. Start a fresh Codex
session after skill files are copied or removed; unchanged updates need no refresh.

When a user types a literal `aether ...` passthrough command such as `status`,
`update`, `focus`, `pheromones`, or `reference-list`, execute that exact command
first. The installed binary and `aether --help` are the runtime source of truth.

For the nine lifecycle actions above, run or inspect
`aether command-guide <command> --platform codex` and follow the matching public
skill and its private support. If the user explicitly says raw, exact,
no-interview, or no-orchestration, execute the requested CLI command directly.
Do not reinterpret a literal passthrough command as a vague workflow request.

For lifecycle shell execution, prefer `AETHER_OUTPUT_MODE=visual aether ...`
unless the user explicitly wants JSON. Preserve exact arguments. Do not preface
literal passthrough execution with repo archaeology or skill narration; the CLI
output is primary, with at most one short sentence of extra explanation.

Agent definitions live in `.codex/agents/*.toml` (TOML format).

---

## Key Commands

| Command | Purpose |
|---------|---------|
| `aether init "<goal>"` | Start a colony with a goal |
| `aether plan` | Generate project phases |
| `aether build <N>` | Execute a phase |
| `aether continue` | Verify, learn, advance |
| `aether status` | Colony dashboard |
| `aether run` | Autopilot all phases |
| `aether focus "<area>"` | Guide colony attention |
| `aether redirect "<pattern>"` | Hard constraint -- avoid this |
| `aether pheromones` | View active signals |
| `aether seal` | Seal colony and retain active state for review |
| `aether entomb` | Optional explicit owner-invoked archive-and-clear alternative |

---

## Key Directories

| Directory | Purpose |
|-----------|---------|
| `.codex/agents/` | Codex CLI agent definitions (TOML) |
| `.aether/` | Colony companion files |
| `.aether/data/` | Local state (never versioned) |
| `.aether/skills/` | Colony and domain skills |
| `.aether/templates/` | State and config templates |

---

## Typical Workflow

In Codex chat, use `$ant-init`, `$ant-plan`, `$ant-build 1`, then `$ant-continue`.
The direct CLI route remains available:

```bash
aether setup                        # Set up Aether in this repo
aether init "Build feature X"       # Start colony
aether colonize                     # Analyze existing code (optional)
aether plan                         # Generate phases
aether focus "security"             # Guide colony (optional)
aether build 1                      # Execute phase 1
aether continue                     # Verify and advance
aether run                          # Or autopilot all phases

# After completing:
aether seal                         # Seal and retain active state for review
aether status                       # Review the retained sealed state first
aether entomb                       # Optional explicit owner-invoked archive-and-clear alternative
aether init "next project goal"     # Only after a successful archive-and-clear receipt
```

### Sealed colony: review before archive

After `aether seal`, whether it is verified or a forced-incomplete closure, the
active colony remains retained for review; a forced-incomplete seal is never
verified completion.

1. First run `aether status` to review the retained sealed state.
2. `aether entomb` is an optional, explicit owner-invoked archive-and-clear
   alternative; it is never automatic or required after sealing.
3. A forced-incomplete marker remains visible in `aether status` and optional
   `aether entomb`.
4. Only after a successful archive-and-clear receipt has verified the archive
   and cleared active state may you run `aether init` for a new goal.

---

*Generated by Aether setup. Customize this file for your project.*
