# Phase 154: Colony Assets - Pattern Map

**Mapped:** 2026-05-23
**Files analyzed:** 38
**Analogs found:** 38 / 38

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `colony/agents/*.yaml` (27 files) | config | file-I/O | `.aether/commands/build.yaml` | role-match |
| `colony/prompts/*.md` (27 files) | config | file-I/O | `.claude/agents/ant/aether-builder.md` | exact |
| `colony/phases/*.yaml` (6+ files) | config | file-I/O | `control-ts/tests/fixtures/phases/init.yaml` | exact |
| `colony/playbooks/*.md` (7 files) | config | file-I/O | `.aether/docs/command-playbooks/build-full.md` | exact |
| `colony/policies/model-routing.yaml` | config | file-I/O | `control-ts/tests/fixtures/policies/model-routing.yaml` | exact |
| `colony/policies/memory-rules.yaml` | config | file-I/O | `control-ts/src/schemas/policy.schema.ts` | role-match |
| `control-ts/src/schemas/agent.schema.ts` | model | transform | `control-ts/src/schemas/agent.schema.ts` | exact |
| `control-ts/tests/fixtures/agents/*.yaml` (25 new) | test | file-I/O | `control-ts/tests/fixtures/agents/queen.yaml` | exact |
| `control-ts/tests/fixtures/phases/*.yaml` (4+ new) | test | file-I/O | `control-ts/tests/fixtures/phases/init.yaml` | exact |
| `control-ts/tests/schemas/agent.schema.test.ts` | test | transform | `control-ts/tests/schemas/agent.schema.test.ts` | exact |

---

## Pattern Assignments

### `colony/agents/*.yaml` (config, file-I/O)

**Analog:** `.aether/commands/build.yaml` (YAML source definition pattern)

**YAML structure pattern** (lines 1-55):
```yaml
name: ant-build
description: "🔨 Build a phase — Queen dispatches workers, colony self-organizes"
source_of_truth: "Use the Go `aether` CLI as the source of truth."
runtime:
  manifest_command: "aether host build --dry-run $ARGUMENTS"
  finalizer_command: "aether build-finalize $ARGUMENTS --completion-file <completion_file>"
  contract: ".aether/docs/wrapper-host-contract.md"
ownership:
  manifest_authority: "TS host (aether host build --dry-run returns plan-only JSON without dispatching workers)"
  worker_conductor: "Wrapper (spawns workers via platform Agent tool, manages ceremony)"
  state_mutator: "Go runtime (build-finalize commits results)"
```

**Agent YAML target pattern** (from RESEARCH.md Pattern 1):
```yaml
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

**Key conventions to copy:**
- Top-level string keys use lowercase with hyphens
- Descriptions are sentence-cased, quoted strings
- Lists use hyphenated YAML array syntax
- `source_of_truth` field establishes ownership boundary

---

### `colony/prompts/*.md` (config, file-I/O)

**Analog:** `.claude/agents/ant/aether-builder.md`

**Frontmatter extraction rule** (lines 1-7):
```markdown
---
name: aether-builder
description: "Use this agent when implementing code from a plan..."
tools: Read, Write, Edit, Bash, Grep, Glob
color: yellow
model: sonnet
---
```

**Body content pattern** (lines 9-237):
```markdown
<role>
You are a Builder Ant in the Aether Colony — the colony's hands...
</role>

<execution_flow>
## TDD Workflow
...
</execution_flow>

<read_cache_discipline>
## Read Cache Discipline
...
</read_cache_discipline>

<critical_rules>
## Non-Negotiable Rules
...
</critical_rules>

<pheromone_protocol>
## Pheromone Signal Response Protocol
...
</pheromone_protocol>

<return_format>
## Output Format
...
</return_format>

<success_criteria>
## Success Verification
...
</success_criteria>

<failure_modes>
## Failure Handling
...
</failure_modes>

<escalation>
## When to Escalate
...
</escalation>

<boundaries>
## Boundary Declarations
...
</boundaries>
```

**Extraction rule:** Strip everything before first `---` and after second `---`. The body becomes the prompt file. Per D-03, one complete prompt per agent — no fragment assembly.

---

### `colony/phases/*.yaml` (config, file-I/O)

**Analog:** `control-ts/tests/fixtures/phases/init.yaml`

**Phase YAML pattern** (lines 1-20):
```yaml
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

**Also reference:** `control-ts/tests/fixtures/phases/plan.yaml` (lines 1-25):
```yaml
id: "plan"
entry_agent: "scout"
required_agents:
  - "scout"
  - "queen"
inputs:
  goal:
    type: "string"
    required: true
  context:
    type: "object"
    required: false
outputs:
  - "phases"
  - "roadmap"
success_criteria:
  - "Phase list generated"
  - "Roadmap approved"
failure_policy: "retry"
ceremony:
  stages:
    - "research"
    - "draft"
    - "review"
```

---

### `colony/playbooks/*.md` (config, file-I/O)

**Analog:** `.aether/docs/command-playbooks/build-full.md`

**Playbook structure pattern** (lines 1-1713):
```markdown
---
name: ant:build
description: "🔨🐜🏗️🐜🔨 Build a phase with pure emergence"
---

You are the **Queen**. You DIRECTLY spawn multiple workers...

## Instructions

### Step 1: Validate + Read State
...

### Step 2: Update State
...

### Step 3: Git Checkpoint
...
```

**Consolidation rule from RESEARCH.md:** Combine split playbooks (build-prep, build-context, build-wave, build-verify, build-complete) into single `colony/playbooks/build.md`. The split files remain in `.aether/docs/command-playbooks/` as wrapper execution details.

---

### `colony/policies/model-routing.yaml` (config, file-I/O)

**Analog:** `control-ts/tests/fixtures/policies/model-routing.yaml`

**Policy YAML pattern** (lines 1-24):
```yaml
model_routing:
  default_provider: "anthropic"
  fallback_providers:
    - "openai"
  routing_rules:
    - agent_role: "builder"
      provider: "anthropic"
      model: "claude-opus-4"
    - agent_role: "watcher"
      provider: "anthropic"
      model: "claude-sonnet-4"

memory_rules:
  max_learnings: 100
  auto_promote_threshold: 0.75

skill_creation:
  allowed: true
  auto_approve: false

safety_gates:
  security_scan: true
  quality_gate: true
```

---

### `control-ts/src/schemas/agent.schema.ts` (model, transform)

**Analog:** `control-ts/src/schemas/agent.schema.ts` (self — modifying existing file)

**Current role enum** (lines 6-21):
```typescript
import { z } from "zod";
import { existsSync } from "fs";
import { resolve } from "path";
import { projectRoot } from "../utils/projectRoot.js";

export const AgentSchema = z
  .object({
    id: z.string().min(1),
    role: z.enum([
      "builder",
      "watcher",
      "scout",
      "queen",
      "oracle",
      "gatekeeper",
      "auditor",
      "probe",
    ]),
    prompt_file: z.string().min(1),
    allowed_tools: z.array(z.string()).default([]),
  })
```

**superRefine file existence check** (lines 22-39):
```typescript
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

**Required change:** Expand role enum from 8 to 27 castes per D-01:
```typescript
role: z.enum([
  "builder", "watcher", "scout", "queen", "oracle", "gatekeeper", "auditor", "probe",
  "architect", "route-setter", "surveyor-nest", "surveyor-disciplines",
  "surveyor-pathogens", "surveyor-provisions", "keeper", "tracker", "weaver",
  "fixer", "medic", "porter", "ambassador", "chronicler", "measurer",
  "includer", "sage", "chaos", "archaeologist"
]),
```

---

### `control-ts/tests/fixtures/agents/*.yaml` (test, file-I/O)

**Analog:** `control-ts/tests/fixtures/agents/queen.yaml`

**Fixture pattern** (lines 1-8):
```yaml
id: "queen"
role: "queen"
prompt_file: "colony/prompts/queen.md"
allowed_tools:
  - "spawn"
  - "delegate"
  - "signal"
```

**Also reference:** `control-ts/tests/fixtures/agents/builder.yaml` (lines 1-9):
```yaml
id: "builder"
role: "builder"
prompt_file: "colony/prompts/builder.md"
allowed_tools:
  - "read"
  - "write"
  - "edit"
  - "bash"
```

---

### `control-ts/tests/schemas/agent.schema.test.ts` (test, transform)

**Analog:** `control-ts/tests/schemas/agent.schema.test.ts` (self — extending existing tests)

**Test structure pattern** (lines 1-58):
```typescript
import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { parse } from "yaml";
import { AgentSchema } from "../../src/schemas/agent.schema.js";
import { fixturePath } from "../helpers.js";

describe("AgentSchema", () => {
  it("validates queen.yaml fixture", () => {
    const text = readFileSync(fixturePath("agents", "queen.yaml"), "utf8");
    const data = parse(text);
    const result = AgentSchema.parse(data);
    expect(result.id).toBe("queen");
    expect(result.role).toBe("queen");
    expect(result.prompt_file).toBe("colony/prompts/queen.md");
    expect(result.allowed_tools).toContain("spawn");
  });

  it("rejects nonexistent prompt_file", () => {
    expect(() =>
      AgentSchema.parse({
        id: "x",
        role: "builder",
        prompt_file: "nonexistent.md",
      })
    ).toThrow();
  });

  it("rejects path traversal in prompt_file", () => {
    expect(() =>
      AgentSchema.parse({
        id: "bad",
        role: "builder",
        prompt_file: "../../../etc/passwd",
      })
    ).toThrow();
  });
});
```

**New tests to add:** Loop over all 27 agent fixtures, verify each parses and `prompt_file` exists.

---

## Shared Patterns

### YAML Frontmatter Stripping
**Source:** `.claude/agents/ant/aether-builder.md` lines 1-7
**Apply to:** All prompt file extractions
```markdown
---
name: aether-builder
description: "..."
tools: Read, Write, Edit, Bash, Grep, Glob
color: yellow
model: sonnet
---
```
Strip everything from first `---` to second `---` (inclusive). The remaining body becomes `colony/prompts/{role}.md`.

### Agent Frontmatter to YAML Mapping
**Source:** `.claude/agents/ant/aether-builder.md` + RESEARCH.md Pattern 1
**Apply to:** All 27 agent YAML files
| Frontmatter Field | Agent YAML Field | Required |
|---|---|---|
| `name` | `id` | Yes |
| (derived from filename) | `role` | Yes |
| (hardcoded) | `prompt_file` | Yes |
| `tools` | `allowed_tools` | Yes |
| (inferred) | `tier` | Optional |
| `model` | `model` | Optional |
| `color` | `color` | Optional |
| `description` | `description` | Optional |

### Zod Schema Extension Pattern
**Source:** `control-ts/src/schemas/agent.schema.ts` lines 6-21
**Apply to:** AgentSchema role enum expansion
Use `.optional()` for new fields to avoid breaking existing fixtures:
```typescript
tier: z.enum(["core", "orchestration", "surveyor", "specialist", "niche"]).optional(),
model: z.string().optional(),
color: z.string().optional(),
description: z.string().optional(),
```

### Test Fixture Loading
**Source:** `control-ts/tests/helpers.ts`
**Apply to:** All new schema tests
```typescript
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

export const projectRoot = resolve(
  dirname(fileURLToPath(import.meta.url)),
  ".."
);

export function fixturePath(...segments: string[]): string {
  return resolve(projectRoot, "tests", "fixtures", ...segments);
}
```

### Playbook Consolidation
**Source:** `.aether/docs/command-playbooks/build-full.md` + RESEARCH.md Pattern 3
**Apply to:** `colony/playbooks/build.md`, `continue.md`, `plan.md`
- `build.md` = build-prep + build-context + build-wave + build-verify + build-complete
- `continue.md` = continue-verify + continue-gates + continue-advance + continue-finalize
- `plan.md` = plan-prep + plan-dispatch
- `colonize.md`, `oracle.md`, `swarm.md`, `seal.md` = extracted from corresponding Go/cmd logic or existing full playbooks

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `colony/ceremony/*.md` | config | file-I/O | Out of scope for Phase 154; Phase 155 per extraction audit |

---

## Metadata

**Analog search scope:**
- `.claude/agents/ant/*.md` (27 files)
- `.aether/docs/command-playbooks/*.md` (14 files)
- `.aether/commands/*.yaml` (60 files)
- `control-ts/src/schemas/*.ts` (4 files)
- `control-ts/tests/fixtures/**/*.yaml` (5 files)
- `control-ts/tests/schemas/*.test.ts` (1 file)
- `.aether/skills/colony/*/SKILL.md` (55 files)
- `.aether/workers.md`

**Files scanned:** 160+
**Pattern extraction date:** 2026-05-23
