# Phase 154: Colony Assets - Research

**Researched:** 2026-05-23
**Domain:** Editable colony behaviour assets (YAML/Markdown extraction)
**Confidence:** HIGH

## Summary

Phase 154 extracts all living colony behaviour from compiled Go code and platform-specific agent files into editable YAML and Markdown files under `colony/`. This is the content-creation phase that populates the schemas defined in Phase 153 with real data. The work splits into six requirement tracks: agent definitions (EXTRACT-01), prompts (EXTRACT-02), phase definitions (EXTRACT-03), playbooks (EXTRACT-04), model-routing policies (EXTRACT-05), and memory rules (EXTRACT-06).

The primary research finding is that all source material already exists and is well-structured. The 27 agent definitions live in `.claude/agents/ant/*.md` with consistent Markdown+YAML frontmatter format. The 11 canonical playbooks live in `.aether/docs/command-playbooks/*.md`. The Zod schemas from Phase 153 are already implemented and tested (17 tests passing). The extraction audit from Phase 152 provides a machine-readable map of what behaviour strings to extract from Go. No new libraries or tools are needed -- this phase is pure content extraction, transformation, and file creation.

**Primary recommendation:** Create `colony/` directory tree, populate it by transforming existing source material into the schema-validated formats, update the AgentSchema role enum from 8 to 27 castes, and verify with `npm run test:schemas`.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Agent YAML definitions | File system (`colony/agents/`) | TS control plane (validation) | Behaviour must be human-editable; TS validates at load time |
| Prompt Markdown | File system (`colony/prompts/`) | TS control plane (assembly) | Prompt text is narrative content best in Markdown |
| Phase YAML definitions | File system (`colony/phases/`) | TS control plane (execution) | Phase orchestration rules are structured config |
| Playbook Markdown | File system (`colony/playbooks/`) | TS control plane (rendering) | Playbooks are procedural instructions in Markdown |
| Policy YAML | File system (`colony/policies/`) | TS control plane (enforcement) | Routing and memory rules are structured config |
| Visual/ceremony extraction | File system (`colony/ceremony/`) | Go runtime (rendering) | Go reads and renders; content lives in files |
| Schema validation | TS control plane | -- | Zod schemas are the contract between files and code |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Zod | 4.4.3 | Schema validation | Already used in control-ts; validates YAML against TS types |
| yaml | 2.9.0 | YAML parsing | Already in control-ts dependencies; parses colony asset files |
| vitest | 4.1.7 | Test runner | Already configured; runs schema validation tests |
| TypeScript | 6.0.3 | Type safety | Already configured with strict mode |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Node.js fs | built-in | File existence checks | AgentSchema superRefine validates prompt_file exists |
| Node.js path | built-in | Path resolution | Resolves prompt_file against projectRoot |

**Installation:** Already installed in `control-ts/` from Phase 153.

**Version verification:**
- Zod 4.4.3: verified via `npm view zod version` and package-lock [VERIFIED: npm registry]
- yaml 2.9.0: verified via package.json [VERIFIED: local package.json]
- vitest 4.1.7: verified via `npx vitest --version` [VERIFIED: runtime check]

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     SOURCE MATERIAL                              │
│  .claude/agents/ant/*.md    ← 27 agent definitions (MD+YAML FM) │
│  .aether/docs/command-playbooks/*.md ← 11 playbooks             │
│  .aether/commands/*.yaml    ← Command definitions (pattern ref)  │
│  cmd/*.go                   ← Go symbols (extraction audit)      │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ extract + transform
┌─────────────────────────────────────────────────────────────────┐
│                     COLONY/ ASSETS (NEW)                         │
│  colony/agents/*.yaml       ← Pure YAML, machine-readable       │
│  colony/prompts/*.md        ← Full prompt text per agent        │
│  colony/phases/*.yaml       ← Phase lifecycle definitions       │
│  colony/playbooks/*.md      ← Canonical playbook content        │
│  colony/policies/*.yaml     ← Model routing + memory rules      │
│  colony/ceremony/*.md       ← Visual/ceremony templates         │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼ validate + load
┌─────────────────────────────────────────────────────────────────┐
│                     TS CONTROL PLANE                             │
│  control-ts/src/schemas/*.ts     ← Zod schemas (Phase 153)      │
│  control-ts/src/agents/loadAgents.ts   ← Reads colony/agents/   │
│  control-ts/src/phases/loadPhases.ts   ← Reads colony/phases/   │
│  control-ts/src/prompts/assemblePrompt.ts ← Reads colony/prompts/│
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
colony/
├── agents/
│   ├── queen.yaml
│   ├── builder.yaml
│   ├── watcher.yaml
│   ├── scout.yaml
│   ├── oracle.yaml
│   ├── gatekeeper.yaml
│   ├── auditor.yaml
│   ├── probe.yaml
│   ├── architect.yaml
│   ├── route-setter.yaml
│   ├── surveyor-nest.yaml
│   ├── surveyor-disciplines.yaml
│   ├── surveyor-pathogens.yaml
│   ├── surveyor-provisions.yaml
│   ├── keeper.yaml
│   ├── tracker.yaml
│   ├── weaver.yaml
│   ├── fixer.yaml
│   ├── medic.yaml
│   ├── porter.yaml
│   ├── ambassador.yaml
│   ├── chronicler.yaml
│   ├── measurer.yaml
│   ├── includer.yaml
│   ├── sage.yaml
│   ├── chaos.yaml
│   └── archaeologist.yaml
├── prompts/
│   ├── queen.md
│   ├── builder.md
│   ├── watcher.md
│   ├── scout.md
│   ├── oracle.md
│   ├── gatekeeper.md
│   ├── auditor.md
│   ├── probe.md
│   └── (other 19 agents as needed)
├── phases/
│   ├── init.yaml
│   ├── discuss.yaml
│   ├── plan.yaml
│   ├── build.yaml
│   ├── continue.yaml
│   └── seal.yaml
├── playbooks/
│   ├── build.md
│   ├── continue.md
│   ├── plan.md
│   ├── colonize.md
│   ├── oracle.md
│   ├── swarm.md
│   └── seal.md
└── policies/
    ├── model-routing.yaml
    └── memory-rules.yaml
```

### Pattern 1: Agent YAML from Markdown Frontmatter
**What:** Transform existing `.claude/agents/ant/*.md` (Markdown+YAML frontmatter) into pure YAML `colony/agents/*.yaml`.
**When to use:** For all 27 agent definitions.
**Example:**
```yaml
# Source: .claude/agents/ant/aether-builder.md frontmatter
# Transformed to: colony/agents/builder.yaml
id: builder
role: builder
prompt_file: colony/prompts/builder.md
allowed_tools:
  - read
  - write
  - edit
  - bash
  - grep
  - glob
tier: core
model: sonnet
color: yellow
description: "Use this agent when implementing code from a plan..."
```

### Pattern 2: Prompt File Extraction
**What:** Extract the Markdown body (after frontmatter) from each agent definition into a standalone `colony/prompts/*.md` file.
**When to use:** For all agents that have prompt content in their definition.
**Key insight:** Per D-03 (CONTEXT.md), one complete prompt per agent. No fragment assembly for this phase.
**Example:**
```markdown
---
# colony/prompts/builder.md

<role>
You are a Builder Ant in the Aether Colony...
</role>

<execution_flow>
## TDD Workflow
...
</execution_flow>

(etc -- full content from .claude/agents/ant/aether-builder.md body)
```

### Pattern 3: Playbook Consolidation
**What:** Move canonical playbooks from `.aether/docs/command-playbooks/` to `colony/playbooks/`.
**When to use:** For all playbook content that defines colony behaviour.
**Key decision:** Per D-04 (CONTEXT.md), `.aether/docs/command-playbooks/` becomes the source for extraction. `colony/playbooks/` is the new canonical location.
**Note:** The split playbooks (build-prep, build-context, build-wave, build-verify, build-complete, continue-verify, continue-gates, continue-advance, continue-finalize, plan-prep, plan-dispatch) may be consolidated into higher-level playbooks (build.md, continue.md, plan.md) or kept as-is depending on planner decision.

### Pattern 4: Schema Role Enum Expansion
**What:** Update AgentSchema role enum from 8 core castes to all 27 castes.
**When to use:** Required by D-01 (CONTEXT.md): "All 27 castes get YAML definitions... The TS schema from Phase 153 must be updated to accept all 27 roles."
**Example:**
```typescript
// Source: control-ts/src/schemas/agent.schema.ts
role: z.enum([
  "builder", "watcher", "scout", "queen", "oracle", "gatekeeper", "auditor", "probe",
  "architect", "route-setter", "surveyor-nest", "surveyor-disciplines",
  "surveyor-pathogens", "surveyor-provisions", "keeper", "tracker", "weaver",
  "fixer", "medic", "porter", "ambassador", "chronicler", "measurer",
  "includer", "sage", "chaos", "archaeologist"
])
```

### Anti-Patterns to Avoid
- **Don't create fragment assembly logic:** Per D-03, shared sections like "Read Cache Discipline" may be duplicated across prompt files. Fragment assembly is deferred.
- **Don't generate platform agent files:** Per Deferred Ideas, a generator for `.claude/`, `.opencode/`, `.codex/` files is out of scope for this phase.
- **Don't delete source files:** The existing `.claude/agents/ant/*.md` and `.aether/docs/command-playbooks/*.md` stay in place as manually-maintained platform layers (per D-02).
- **Don't hardcode new behaviour in Go:** The hard rule from Phase 152 is "Compiled code may execute behaviour, but editable assets must define behaviour."

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing | Custom parser | `yaml` package (already in deps) | Battle-tested, handles spec edge cases |
| Schema validation | Manual type guards | Zod schemas (already implemented) | Declarative, generates TS types, good errors |
| File existence check | fs.stat inline | superRefine pattern (already in AgentSchema) | Consistent error messages, path traversal protection |
| Path resolution | process.cwd() + join | projectRoot helper (already exists) | Works regardless of where script is invoked |

**Key insight:** All tooling for this phase already exists in `control-ts/`. The work is content transformation, not infrastructure building.

## Runtime State Inventory

This phase creates new files under `colony/` -- a greenfield directory. No existing runtime state needs migration.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None -- `colony/` does not exist yet | N/A |
| Live service config | None -- no external services involved | N/A |
| OS-registered state | None | N/A |
| Secrets/env vars | None | N/A |
| Build artifacts | None -- new directory | N/A |

**Nothing found in category:** All categories verified as empty -- this phase creates new assets, does not modify existing runtime state.

## Common Pitfalls

### Pitfall 1: Schema Drift Between Agent YAML and Prompt Files
**What goes wrong:** Agent YAML references `prompt_file: colony/prompts/builder.md` but the actual file is named `builder-prompt.md` or lives elsewhere.
**Why it happens:** Manual file creation without cross-referencing.
**How to avoid:** Use the AgentSchema's existing `superRefine` file existence check -- it will throw a ZodError if the referenced prompt file does not exist.
**Warning signs:** `npm run test:schemas` fails with "prompt_file does not exist" errors.

### Pitfall 2: Inconsistent Role Names
**What goes wrong:** Agent YAML uses `role: surveyor_nest` but the schema expects `surveyor-nest` (hyphen vs underscore).
**Why it happens:** The existing agent files use hyphenated names (`aether-surveyor-nest.md`) but someone might use underscores in YAML.
**How to avoid:** Use the exact role names from the existing agent filenames. The 27 roles are: builder, watcher, scout, queen, oracle, gatekeeper, auditor, probe, architect, route-setter, surveyor-nest, surveyor-disciplines, surveyor-pathogens, surveyor-provisions, keeper, tracker, weaver, fixer, medic, porter, ambassador, chronicler, measurer, includer, sage, chaos, archaeologist.

### Pitfall 3: Missing Tier/Model/Color Metadata
**What goes wrong:** Agent YAML only includes id/role/prompt_file/allowed_tools but omits tier, model, color that the TS control plane might need for orchestration.
**Why it happens:** The Phase 153 schema only required id/role/prompt_file/allowed_tools.
**How to avoid:** The schema can be extended with `.optional()` fields for tier, model, color. The existing agent frontmatter contains this data (extracted via research above).

### Pitfall 4: Playbook Content Duplication vs Consolidation
**What goes wrong:** Creating both split playbooks (build-prep, build-wave) and consolidated playbooks (build) leads to confusion about which is canonical.
**Why it happens:** The existing `.aether/docs/command-playbooks/` has both split files and `build-full.md` / `continue-full.md`.
**How to avoid:** Per D-04, `colony/playbooks/` is the new canonical location. Decide during planning whether to consolidate or keep split. The research recommendation is to consolidate into ~7 high-level playbooks (build, continue, plan, colonize, oracle, swarm, seal) since the split files are wrapper execution details, not behaviour definitions.

### Pitfall 5: Frontmatter Leaking into Prompt Files
**What goes wrong:** When extracting prompt content from `.claude/agents/ant/*.md`, the YAML frontmatter (name, description, tools, color, model) is accidentally included in the `colony/prompts/*.md` file.
**Why it happens:** The source files have frontmatter; the prompt files should be pure Markdown body.
**How to avoid:** Strip everything before the first `---` and after the second `---` when extracting prompt content.

## Code Examples

### Verified patterns from official sources:

### Creating an Agent YAML Fixture
```yaml
# Source: control-ts/tests/fixtures/agents/queen.yaml (from Phase 153)
id: "queen"
role: "queen"
prompt_file: "colony/prompts/queen.md"
allowed_tools:
  - "spawn"
  - "delegate"
  - "signal"
```

### Creating a Phase YAML Fixture
```yaml
# Source: control-ts/tests/fixtures/phases/init.yaml (from Phase 153)
id: "init"
entry_agent: "queen"
required_agents:
  - "queen"
inputs:
  goal:
    type: "string"
    required: true
outputs:
  - "colony_state"
  - "session_id"
success_criteria:
  - "COLONY_STATE.json created"
  - "session.json initialized"
failure_policy: "block"
ceremony:
  stages:
    - "setup"
    - "validate"
    - "commit"
```

### Schema Validation with File Existence
```typescript
// Source: control-ts/src/schemas/agent.schema.ts (from Phase 153)
export const AgentSchema = z
  .object({
    id: z.string().min(1),
    role: z.enum([/* 27 roles */]),
    prompt_file: z.string().min(1),
    allowed_tools: z.array(z.string()).default([]),
  })
  .superRefine((data, ctx) => {
    if (data.prompt_file.includes("..")) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `prompt_file contains path traversal: ${data.prompt_file}`,
        path: ["prompt_file"],
      });
      return;
    }
    const resolved = resolve(projectRoot, data.prompt_file);
    if (!existsSync(resolved)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `prompt_file does not exist: ${resolved}`,
        path: ["prompt_file"],
      });
    }
  });
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Agent behaviour in Go hardcoded strings | Editable YAML under `colony/` | Phase 154 (this phase) | Humans can edit agent definitions without recompiling |
| Agent behaviour in `.claude/agents/*.md` only | Canonical YAML + platform-specific wrappers | Phase 154 (this phase) | Machine-readable source enables TS control plane |
| Playbooks in `.aether/docs/command-playbooks/` | Canonical `colony/playbooks/*.md` | Phase 154 (this phase) | Clear separation between reference docs and behaviour |
| 8 core agents in schema | 27 castes in schema | Phase 154 (this phase) | Full colony support in TS control plane |

**Deprecated/outdated:**
- Hardcoded agent definitions in Go: being extracted in Phase 155
- Platform agent files as sole source of truth: replaced by `colony/agents/*.yaml` as canonical

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | All 27 agent definitions have consistent Markdown+YAML frontmatter format | Standard Stack | If some agents lack frontmatter, extraction script needs special handling |
| A2 | The `colony/` directory should be at repo root (sibling to `cmd/`, `pkg/`, `.aether/`) | Architecture Patterns | If wrong path, all prompt_file references and schema tests break |
| A3 | Existing platform files (`.claude/agents/`, `.opencode/agents/`, `.codex/agents/`) remain manually maintained | Architecture Patterns | If user expects auto-generation, they'll be disappointed; but this is explicitly deferred |
| A4 | Prompt files should contain the full Markdown body from existing agent files, not trimmed/summarized | Pattern 2 | If wrong, TS control plane won't have complete prompts for workers |
| A5 | Playbooks should be consolidated (not split) in `colony/playbooks/` | Pattern 3 | If planner chooses split, adjust file list accordingly |
| A6 | The 8 "core" castes from Phase 153 schema are: builder, watcher, scout, queen, oracle, gatekeeper, auditor, probe | Pattern 4 | If wrong, the 19 additional roles may be incorrect |

## Open Questions

1. **Should agent YAML include tier/model/color metadata?**
   - What we know: Existing agent frontmatter has `color`, `model`, `tools`. The Phase 153 schema only requires id/role/prompt_file/allowed_tools.
   - What's unclear: Does the TS control plane need tier/model/color for orchestration decisions?
   - Recommendation: Include tier/model/color as optional fields in the YAML (not required by schema) so they're available without breaking validation.

2. **How many phase YAMLs are needed?**
   - What we know: Requirements mention init, discuss, plan, build, continue, seal. Fixtures from Phase 153 have init.yaml and plan.yaml.
   - What's unclear: Are discuss, build, continue, seal also needed as phase YAMLs in this phase?
   - Recommendation: Yes -- create all 6 lifecycle phases plus any auxiliary phases (colonize, oracle, swarm) if they have distinct orchestration behaviour.

3. **Should playbooks be consolidated or kept split?**
   - What we know: Existing playbooks are split (build-prep, build-wave, etc.) but there are also full versions (build-full.md).
   - What's unclear: Does the TS control plane need granular playbooks or high-level ones?
   - Recommendation: Consolidate into ~7 high-level playbooks (build, continue, plan, colonize, oracle, swarm, seal) for the canonical location. The split files remain in `.aether/docs/command-playbooks/` as wrapper execution details.

4. **What ceremony/visual content should be extracted to `colony/ceremony/`?**
   - What we know: The extraction audit identifies 14 MOVE_TO_MARKDOWN items (visuals, ceremonies, etc.).
   - What's unclear: Is ceremony extraction in scope for Phase 154 or deferred to Phase 155?
   - Recommendation: Phase 154 focuses on EXTRACT-01 through EXTRACT-06 (agents, prompts, phases, playbooks, policies). Ceremony/visual extraction is Phase 155 work per the audit.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS control plane | Yes | 26.0.0 | -- |
| npm | Package management | Yes | 11.12.1 | -- |
| TypeScript | Schema compilation | Yes | 5.9.3 | -- |
| Vitest | Test runner | Yes | 4.1.7 | -- |
| Zod | Schema validation | Yes | 4.4.3 | -- |
| yaml | YAML parsing | Yes | 2.9.0 | -- |
| Go | Runtime tests | Yes | 1.24+ | -- |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 4.1.7 |
| Config file | `control-ts/vitest.config.ts` |
| Quick run command | `cd control-ts && npx vitest run tests/schemas/agent.schema.test.ts` |
| Full suite command | `cd control-ts && npm run test:schemas` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXTRACT-01 | Agent YAML validates against schema | unit | `npx vitest run tests/schemas/agent.schema.test.ts` | Yes (Phase 153) |
| EXTRACT-02 | Prompt files exist and are readable | unit | Custom test reading colony/prompts/*.md | No -- Wave 0 |
| EXTRACT-03 | Phase YAML validates against schema | unit | `npx vitest run tests/schemas/phase.schema.test.ts` | Yes (Phase 153) |
| EXTRACT-04 | Playbook files exist and are readable | unit | Custom test reading colony/playbooks/*.md | No -- Wave 0 |
| EXTRACT-05 | Policy YAML validates against schema | unit | `npx vitest run tests/schemas/policy.schema.test.ts` | Yes (Phase 153) |
| EXTRACT-06 | Memory rules YAML validates | unit | Extend policy.schema.test.ts | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `cd control-ts && npx vitest run tests/schemas/<relevant>.test.ts`
- **Per wave merge:** `cd control-ts && npm run test:schemas`
- **Phase gate:** Full schema test suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `control-ts/tests/fixtures/agents/*.yaml` -- need all 27 agent fixtures (only queen.yaml and builder.yaml exist)
- [ ] `control-ts/tests/fixtures/phases/*.yaml` -- need all lifecycle phases (only init.yaml and plan.yaml exist)
- [ ] `control-ts/tests/fixtures/policies/memory-rules.yaml` -- new fixture needed
- [ ] Tests verifying all 27 agent YAMLs parse correctly -- new test needed
- [ ] Tests verifying all phase YAMLs parse correctly -- new test needed
- [ ] Tests verifying prompt files are referenced correctly -- new test needed

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | N/A -- no auth in this phase |
| V3 Session Management | No | N/A -- no sessions |
| V4 Access Control | Yes | Path traversal prevention in AgentSchema superRefine |
| V5 Input Validation | Yes | Zod schema validation on all YAML files |
| V6 Cryptography | No | N/A |
| V7 Error Handling | Yes | Schema validation errors must not leak file system paths in production |
| V12 File Upload | Yes | YAML files are "uploaded" from disk; validation prevents malicious structures |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in prompt_file | Tampering | superRefine rejects `..` segments [VERIFIED: existing schema] |
| YAML bomb / excessive nesting | Denial of Service | Use yaml package defaults; files are small and trusted in this context |
| Malformed YAML causing crashes | Denial of Service | Wrap yaml.parse in try/catch; schema validation catches structural issues |
| Missing prompt_file reference | Information Disclosure | Schema validation fails fast with clear error; no runtime surprises |

## Sources

### Primary (HIGH confidence)
- `control-ts/src/schemas/*.ts` -- Schema implementations from Phase 153
- `control-ts/tests/fixtures/**/*.yaml` -- Existing fixture files
- `.claude/agents/ant/*.md` -- All 27 agent definitions (verified via glob + read)
- `.aether/docs/command-playbooks/*.md` -- All playbook files (verified via glob + read)
- `.planning/phases/152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md` -- Machine-readable extraction map
- `.planning/phases/154-colony-assets/154-CONTEXT.md` -- Locked decisions from discuss phase

### Secondary (MEDIUM confidence)
- `control-ts/package.json` -- Dependency versions
- `.aether/commands/*.yaml` -- Command definition patterns (referenced as existing YAML patterns)

### Tertiary (LOW confidence)
- None -- all claims verified against primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools already installed and verified working
- Architecture: HIGH -- source material exists and is well-structured; decisions locked in CONTEXT.md
- Pitfalls: MEDIUM-HIGH -- based on observed patterns in source files and schema behaviour; some assumptions about TS control plane needs

**Research date:** 2026-05-23
**Valid until:** 2026-06-23 (stable -- this is a content extraction phase, not a fast-moving dependency phase)
