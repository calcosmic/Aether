# Architecture Boundary: Go, TypeScript, and Editable Assets

> **Version:** v1.24
> **Last Updated:** 2026-05-22
> **Applies to:** Phases 152-159

---

## The Hard Rule

**Compiled code may execute behaviour, but editable assets must define behaviour.**

This means:
- Go can run the logic, but it must not be the only place where that logic's rules live.
- If a human cannot open a text file and change what an agent does, where it is routed, or what it is told, then the architecture has leaked behaviour into compiled code.

For dummies: The engine (Go) can drive the car, but the map, the driver instructions, and the paint job must be things you can edit with a text editor — not hidden inside the engine block.

---

## Three-Tier Boundary

| Layer | Owner | Examples | Rationale |
|-------|-------|----------|-----------|
| **Go Runtime Spine** | Go | State mutation, verification, CLI truth, file locking, install/update/publish, recovery, event emission | Safety-critical, must be compiled, tested, and atomic. Go is the sole authority for `.aether/data/` writes. |
| **TypeScript Control Plane** | TypeScript | Orchestration, agent loader, phase runner, prompt assembler, event stream consumer, platform adapters | Living behaviour that needs iteration speed, platform awareness, and prompt contract testing. Never writes state directly. |
| **Editable Assets** | Markdown/YAML/JSON | Agent definitions, prompts, phases, playbooks, policies, skills, visual config, ceremony templates | Human-editable, version-controlled, distributed by `aether update`. These are the colony's brain, not its engine. |
| **Bash Glue** | Bash | Small wrapper scripts, platform-specific command dispatch, smoke tests, release helpers | Minimal glue only. No state mutation, no orchestration logic, no ceremony templates. |

---

## Asset-Type Matrix

| Asset Type | Current Location | Target Location | Format | Status |
|------------|------------------|-----------------|--------|--------|
| Agent definitions | `.claude/agents/ant/*.md`, `.opencode/agents/*.md`, `.codex/agents/*.toml` | `colony/agents/*.yaml` | YAML + Markdown | Planned (Phase 154) |
| Prompt text | Hardcoded in Go render functions (e.g. `cmd/colony_prime_context.go`) | `colony/prompts/*.md` | Markdown | Planned (Phase 154) |
| Phase definitions | Hardcoded in Go workflow commands | `colony/phases/*.yaml` | YAML | Planned (Phase 154) |
| Playbooks | `.aether/docs/command-playbooks/*.md` | `colony/playbooks/*.md` | Markdown | Planned (Phase 154) |
| Model-routing policy | `.aether/workers.md` and agent frontmatter | `colony/policies/model-routing.yaml` | YAML | Planned (Phase 154) |
| Memory/skill-creation policy | Hardcoded in Go (`cmd/skills.go`, `cmd/learning.go`) | `colony/policies/*.yaml` | YAML | Planned (Phase 154) |
| Visual config | Hardcoded in `cmd/codex_visuals.go` | `.aether/config/visuals.md` | Markdown + YAML frontmatter | Planned (Phase 155) |
| Ceremony templates | Hardcoded in Go render functions | `colony/ceremony/*.md` | Markdown | Planned (Phase 154) |
| Command definitions | `.aether/commands/*.yaml` | `.aether/commands/*.yaml` | YAML | Keep — already editable |
| Worker specs | `.aether/workers.md` | `.aether/workers.md` | Markdown | Keep — already editable |
| Skills | `.aether/skills/` | `.aether/skills/` | Markdown | Keep — already editable |
| Event schema | Hardcoded in Go + TS | `control-ts/src/schemas/event.schema.ts` | Zod + TS | Planned (Phase 153) |

---

## Integration Points

The three tiers communicate through well-defined contracts:

1. **Go emits NDJSON events** → TypeScript control plane reads them from `.aether/events/current.ndjson`
2. **TS control plane calls Go CLI** for state mutation (e.g. `aether state-mutate`, `aether build-finalize`). It never writes `.aether/data/` directly.
3. **Go loads editable assets at runtime** from `colony/` and `.aether/` (e.g. visuals, prompts, policies)
4. **`aether update` distributes new assets** alongside existing skills/commands, keeping hub and repo in sync
5. **Platform wrappers** (Claude, OpenCode) add presentation framing but must not mutate state or duplicate verification logic

For dummies: Go is the vault — only Go opens the safe. TypeScript is the concierge — it arranges everything but asks Go to actually change the records. The text files are the menu — anyone can edit what's offered.

---

## Migration Sequence (Phases 152-159)

| Phase | Focus | Asset Types Moving |
|-------|-------|-------------------|
| **152** | Boundary & Parity | Docs only — define what goes where |
| **153** | TS Scaffold & Schemas | Zod schemas for agents, phases, events, policies |
| **154** | Colony Assets | Agents, prompts, phases, playbooks, policies, ceremony templates |
| **155** | Go Boundary Refactor | Go loads visuals, prompts, policies from files instead of hardcoding |
| **156** | TS Control Plane Core | Agent loader, phase loader, prompt assembler, phase runner, orchestrator |
| **157** | TS Adapters & Oracle | Platform adapter stubs, Oracle loop stub |
| **158** | Event Stream | NDJSON event stream as shared observable truth |
| **159** | End-to-End Acceptance | Validate full lifecycle, verify no hardcoded behaviour remains |

---

## Decision Log

| Decision | ID | Rationale |
|----------|-----|-----------|
| Move hardcoded visuals from compiled Go to editable config files | D-01 | Visuals are behaviour, not safety logic. Users should be able to change caste colours without recompiling. |
| One shared visual config file across all platforms | D-02 | Caste identity is runtime truth, not platform quirk. Codex, Claude, and OpenCode should render the same identity. |
| Format: Markdown with YAML frontmatter | D-03 | Matches existing `.aether/agents/` and `.claude/agents/` patterns. No new format to learn. |
| Location: `.aether/config/visuals.md` | D-04 | Central, version-controlled, distributed by `aether update`. |
| Full inventory of all Classic v5.4.0 behaviours | D-05 | Need a complete baseline before deciding what to restore, improve, or drop. |
| When Classic and Go disagree, user decides per item | D-06 | No blanket "Classic wins" or "Go wins" — each behaviour is evaluated on merit. |
| Verification: automated golden/snapshot tests + manual checklist | D-07 | 9 flagship workflows get automated proof; edge cases and ceremony get human checklist. |
| File-level classification for most Go files | D-08 | Practical and readable. Function-level notes only for mixed files. |
| Agent-assisted audit: automated first pass + agent review | D-09 | Scale: 80+ Go files. Heuristics catch obvious cases; agents handle boundary judgement. |
| Mixed files annotated as `MIXED` with function-level notes | D-10 | Some files (e.g. `cmd/colony_prime_context.go`) contain both spine logic and prompt strings. |
| ARCHITECTURE_BOUNDARY.md and PARITY_CLASSIC_VS_GO.md live in `.aether/docs/` | D-11 | Distributed system reference docs, available to all platforms after `aether update`. |
| BEHAVIOUR_EXTRACTION_AUDIT.md lives in `.planning/phases/152-boundary-parity/` | D-12 | Phase artifact, not user-facing. Kept with planning context for downstream phase use. |
| Format: plain reference doc markdown | D-13 | ADR template is too rigid for a checklist and an audit table. Plain markdown is readable and diff-friendly. |

---

## Cross-References

- **Parity checklist:** See [PARITY_CLASSIC_VS_GO.md](PARITY_CLASSIC_VS_GO.md) for the Classic v5.4.0 behaviour baseline.
- **Requirements:** See `.planning/REQUIREMENTS.md` §Boundary & Parity for BOUNDARY-01, BOUNDARY-02, BOUNDARY-03.
- **v1.18 milestone:** See `.planning/milestones/v1.18-ROADMAP.md` for the Classic baseline and golden test coverage.
- **Hybrid strategy research:** See `.aether/docs/hybrid-runtime-strategy-research.md` for the research that led to this boundary.
- **Classic command parity:** See `.aether/docs/classic-command-parity-matrix.md` for the command-by-command runtime authority map.

---

*Documented for Phase 152. Serves as the north star for Phases 153-159.*
