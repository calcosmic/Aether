# Phase 154: Colony Assets - Context

**Gathered:** 2026-05-23
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers editable YAML and Markdown files under `colony/` that define all living colony behaviour: agent identities, prompts, phase rituals, playbooks, and policies.

Six requirements: EXTRACT-01 through EXTRACT-06.

Hard rule from Phase 152: "Compiled code may execute behaviour, but editable assets must define behaviour."

</domain>

<decisions>
## Implementation Decisions

### Agent Completeness
- **D-01:** All 27 castes get YAML definitions under `colony/agents/*.yaml` — not just the 8 core agents. The TS schema from Phase 153 must be updated to accept all 27 roles.

### Platform-Specific Files
- **D-02:** Parallel tracks. `colony/agents/*.yaml` becomes the canonical runtime source for the TS control plane. Existing platform files (`.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml`) stay as manually-maintained platform layers. A generator that produces all three formats from the YAML is a promising future enhancement, but out of scope for this phase.

### Prompt Organization
- **D-03:** One complete prompt per agent. `colony/prompts/builder.md` contains the full Builder prompt, mirroring the existing agent file structure. No fragment assembly logic for this phase — shared sections like "Read Cache Discipline" may be duplicated across prompt files. Fragment assembly can be optimized later.

### Playbook Location
- **D-04:** Move playbooks to `colony/playbooks/`. The existing `.aether/docs/command-playbooks/` files become the source for extraction. `colony/playbooks/` is the new canonical location for behaviour playbooks; `.aether/docs/` remains for reference documentation only.

### Claude's Discretion
- Platform file strategy chosen as parallel tracks (user deferred to Claude)
- Prompt organization chosen as one complete prompt per agent (user deferred to Claude)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Project direction
- `.planning/PROJECT.md` — v1.24 milestone goal, key decisions, and explicit deferrals
- `.planning/ROADMAP.md` §Phase 154 — Success criteria and requirements (EXTRACT-01 through EXTRACT-06)
- `.planning/REQUIREMENTS.md` §EXTRACT-01 through EXTRACT-06 — Detailed extraction requirements
- `.planning/STATE.md` — Current milestone state and accumulated decisions

### Prior phase context
- `.planning/phases/152-boundary-parity/152-CONTEXT.md` — Architecture boundary decisions, visual config format (Markdown+YAML frontmatter)
- `.planning/phases/152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md` — Machine-readable inventory of every Go symbol classified by disposition (MOVE_TO_YAML, MOVE_TO_MARKDOWN, etc.)

### Schema contracts
- `control-ts/src/schemas/agent.schema.ts` — Agent YAML validation schema (must be updated for 27 roles)
- `control-ts/src/schemas/phase.schema.ts` — Phase YAML validation schema
- `control-ts/src/schemas/policy.schema.ts` — Policy YAML validation schema
- `control-ts/src/schemas/event.schema.ts` — Event schema (for context)

### Extraction targets in Go
- `cmd/codex_visuals.go` — Hardcoded caste emojis, ANSI colours, banner templates, stage separators (MOVE_TO_MARKDOWN per audit)
- `cmd/colony_prime_context.go` — Hardcoded prompt section text in context assembly (MIXED per audit)
- `cmd/ceremony_cmd.go` — Ceremony rendering functions (MIXED per audit)
- `cmd/codex_workflow_cmds.go` — Seal ceremony and signal creation (MIXED per audit)

### Existing editable assets (patterns to follow)
- `.aether/workers.md` — Existing editable asset pattern for worker definitions
- `.aether/commands/*.yaml` — Existing YAML-based command definition pattern
- `.aether/skills/` — Skills already loaded from files at runtime

### Existing platform agent definitions (source material)
- `.claude/agents/ant/*.md` — 27 Claude Code agent definitions (Markdown+YAML frontmatter)
- `.opencode/agents/*.md` — 27 OpenCode agent definitions
- `.codex/agents/*.toml` — 27 Codex agent definitions (TOML format)

### Existing playbooks (source material)
- `.aether/docs/command-playbooks/build-prep.md`
- `.aether/docs/command-playbooks/build-context.md`
- `.aether/docs/command-playbooks/build-wave.md`
- `.aether/docs/command-playbooks/build-verify.md`
- `.aether/docs/command-playbooks/build-complete.md`
- `.aether/docs/command-playbooks/continue-verify.md`
- `.aether/docs/command-playbooks/continue-gates.md`
- `.aether/docs/command-playbooks/continue-advance.md`
- `.aether/docs/command-playbooks/continue-finalize.md`
- `.aether/docs/command-playbooks/plan-prep.md`
- `.aether/docs/command-playbooks/plan-dispatch.md`

### Visual/ceremony extraction targets
- `.aether/docs/wrapper-runtime-ux-contract.md` — Wrapper-runtime contract (relevant for visual extraction scope)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `control-ts/src/schemas/*.ts` — Zod schemas already exist and validate agent, phase, event, and policy shapes. AgentSchema needs role enum expansion from 8 to 27 castes.
- `.aether/workers.md` — Existing editable asset pattern; shows how behaviour already lives outside Go for worker definitions.
- `.aether/commands/*.yaml` — YAML source definitions for slash commands; proves the YAML loading path already exists.
- `.aether/skills/` — Skills already loaded from files at runtime; same pattern extends to agents, prompts, and playbooks.

### Established Patterns
- Go runtime loads config from `.aether/` at runtime; `aether update` distributes hub changes downstream. The same mechanism can distribute `colony/` assets.
- Platform wrappers add framing on top of runtime output — they don't replace it. Codex is runtime-native; any visual config must be readable by Go directly.
- Agent definitions use Markdown+YAML frontmatter (`.claude/agents/ant/*.md`). The new `colony/agents/*.yaml` will be pure YAML for machine consumption.
- Playbooks are Markdown files with bash code blocks and instructions. The new `colony/playbooks/*.md` preserves this format.

### Integration Points
- `control-ts/src/agents/loadAgents.ts` (from Phase 156 plans) will read `colony/agents/*.yaml` — the files created in this phase are its input.
- `control-ts/src/phases/loadPhases.ts` (from Phase 156 plans) will read `colony/phases/*.yaml`.
- `control-ts/src/prompts/assemblePrompt.ts` (from Phase 156 plans) will read `colony/prompts/*.md`.
- `aether update` will need to distribute new `colony/` assets alongside existing skills/commands.
- Phase 155 (Go Boundary Refactor) uses the extraction audit to know which Go symbols to refactor — the audit already specifies target locations under `colony/`.

### Counts
- 27 agent definitions across 3 platforms
- 11 existing playbooks in `.aether/docs/command-playbooks/`
- 60 existing command YAMLs in `.aether/commands/`
- No `colony/` directory exists yet — this phase creates it

</code_context>

<specifics>
## Specific Ideas

- Agent YAML format should include: `id`, `role`, `prompt_file`, `allowed_tools`, plus optional `tier` or `category` for grouping the 27 castes.
- Prompt files should mirror the existing agent definition structure: role description, execution flow, discipline sections, critical rules.
- Phase YAMLs should define the colony lifecycle phases: init, discuss, plan, build, continue, seal, plus any auxiliary phases.
- Policy YAML should include model-routing rules, memory-rules, skill-creation policy, and safety-gates per the PolicySchema.
- Visual/ceremony content from `cmd/codex_visuals.go` should be extracted to `colony/ceremony/visuals.md` or similar, not left in Go.

</specifics>

<deferred>
## Deferred Ideas

- **Generator for platform agent files** — A script that reads `colony/agents/*.yaml` and produces `.claude/`, `.opencode/`, `.codex/` agent files. Promising but out of scope for this extraction phase.
- **Prompt fragment assembly** — Shared sections like "Read Cache Discipline" living as fragments. Optimization for later.
- **Web UI for event stream** — Already deferred in PROJECT.md (terminal + NDJSON sufficient).
- **Vector backend for memory** — Already deferred in STATE.md.

</deferred>

---

*Phase: 154-Colony Assets*
*Context gathered: 2026-05-23*
