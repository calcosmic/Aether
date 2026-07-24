# Phase 153: TypeScript Scaffold & Schemas - Research

**Researched:** 2026-05-22
**Domain:** TypeScript control-plane library, Zod schema validation, YAML/JSON config loading
**Confidence:** HIGH

## Summary

This phase bootstraps `control-ts/`, a TypeScript control-plane library that will eventually own orchestration behaviour extracted from Go. The immediate deliverables are the project scaffold, Zod schemas for colony asset types, and a test suite that validates schemas against sample YAML fixtures.

The architecture boundary from Phase 152 makes the scope clear: Go stays the runtime spine; TypeScript owns orchestration logic and asset validation. Phase 153 is the foundation for all subsequent TS work (Phases 156-158). It must set patterns that downstream phases follow without rethinking.

**Primary recommendation:** Use a minimal, strict TypeScript toolchain (tsc + tsx + vitest) with Zod for schemas, the `yaml` package for parsing, and a flat `src/` structure organised by domain (schemas/, loaders/, types/). Avoid Vite, Next.js, or any web-framework baggage — this is a Node library, not a web app.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Schema validation | TypeScript control plane | — | Zod schemas validate YAML assets before Go or TS consumes them |
| YAML/JSON parsing | TypeScript control plane | — | `yaml` package parses colony asset files |
| File existence checks | TypeScript control plane | — | Node `fs` validates `prompt_file` references at load time |
| NDJSON event validation | TypeScript control plane | Go runtime | TS defines event schema; Go emits events that must conform |
| Build/test toolchain | TypeScript control plane | — | Vitest + tsx + tsc, no web framework |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| typescript | 5.8.x | Type checking, declaration emit | Required by Zod; strict mode mandatory [VERIFIED: npm registry 5.8.3] |
| zod | 3.25.x | Schema validation + static type inference | De-facto standard for TS schema validation; zero deps [VERIFIED: npm registry 3.25.2] |
| yaml | 2.9.x | YAML 1.2 parsing | Maintained byeemeli, spec-compliant, no native deps [VERIFIED: npm registry 2.9.0] |
| vitest | 3.2.x | Test runner | Native TS/ESM support, faster than Jest for libraries [VERIFIED: npm registry 3.2.7] |
| tsx | 4.22.x | TS execution without pre-build | Fast `tsx` CLI for scripts and `npm run` commands [VERIFIED: npm registry 4.22.3] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @types/node | 22.x | Node.js built-in type definitions | Always — needed for `fs`, `path`, `process` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Zod | Joi, Yup, Valibot | Zod has best TS inference and ecosystem; Valibot is smaller but newer and less mature |
| Vitest | Jest | Jest requires more config for ESM/TS; Vitest is zero-config for this use case |
| tsx | ts-node | ts-node is slower and has more config surface; tsx is drop-in replacement |
| yaml | js-yaml | `yaml` is more spec-compliant (YAML 1.2); js-yaml is older and has edge-case bugs |

**Installation:**
```bash
npm install zod yaml
npm install -D typescript vitest tsx @types/node
```

**Version verification:** All versions verified against npm registry on 2026-05-22.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────┐
│            Colony Asset Files                │
│  colony/agents/*.yaml                        │
│  colony/phases/*.yaml                        │
│  colony/policies/*.yaml                      │
│  colony/prompts/*.md                         │
└──────────────┬──────────────────────────────┘
               │ read + parse (yaml package)
               ▼
┌─────────────────────────────────────────────┐
│      control-ts/src/loaders/*.ts             │
│  - loadAgents.ts    (read dir → parse YAML) │
│  - loadPhases.ts    (read dir → parse YAML) │
│  - loadPolicies.ts  (read dir → parse YAML) │
└──────────────┬──────────────────────────────┘
               │ validate + infer types
               ▼
┌─────────────────────────────────────────────┐
│      control-ts/src/schemas/*.ts             │
│  - agent.schema.ts   (Zod object)            │
│  - phase.schema.ts   (Zod object)            │
│  - event.schema.ts   (Zod object)            │
│  - policy.schema.ts  (Zod object)            │
└──────────────┬──────────────────────────────┘
               │ .parse() → typed objects
               ▼
┌─────────────────────────────────────────────┐
│      control-ts/src/types/*.ts               │
│  - export inferred types (Agent, Phase, ...) │
└──────────────────────────────────────────────┘
```

### Recommended Project Structure

```
control-ts/
├── package.json
├── tsconfig.json
├── vitest.config.ts
├── src/
│   ├── index.ts                 # public API exports
│   ├── schemas/
│   │   ├── agent.schema.ts      # SCHEMA-01
│   │   ├── phase.schema.ts      # SCHEMA-02
│   │   ├── event.schema.ts      # SCHEMA-03
│   │   └── policy.schema.ts     # SCHEMA-04
│   ├── types/
│   │   └── index.ts             # re-exports of z.infer<typeof ...>
│   ├── loaders/
│   │   ├── loadAgents.ts        # CONTROL-02 (stub)
│   │   ├── loadPhases.ts        # CONTROL-03 (stub)
│   │   └── loadPolicies.ts      # CONTROL-09/10 (stub)
│   └── utils/
│       ├── fileExists.ts        # shared fs helper
│       └── readYamlDir.ts       # shared YAML dir loader
├── tests/
│   ├── fixtures/
│   │   ├── agents/
│   │   │   └── queen.yaml       # sample agent for schema tests
│   │   ├── phases/
│   │   │   └── init.yaml        # sample phase for schema tests
│   │   ├── policies/
│   │   │   └── model-routing.yaml
│   │   └── events/
│   │       └── sample.ndjson    # sample NDJSON lines
│   ├── schemas/
│   │   ├── agent.schema.test.ts
│   │   ├── phase.schema.test.ts
│   │   ├── event.schema.test.ts
│   │   └── policy.schema.test.ts
│   └── helpers.ts               # shared test utilities
└── colony/                      # symlinked or copied from repo root
    ├── agents/
    ├── phases/
    ├── policies/
    └── prompts/
```

### Pattern 1: Zod Schema with File Existence Validation
**What:** Validate that a `prompt_file` field points to an existing file on disk.
**When to use:** Any schema that references external files (agents, phases, policies).
**Example:**
```typescript
// Source: zod.dev error-customization docs + WebSearch verified patterns
import { z } from "zod";
import { existsSync } from "fs";
import { resolve } from "path";

export const AgentSchema = z.object({
  id: z.string().min(1),
  role: z.enum(["builder", "watcher", "scout", "queen", "oracle", "gatekeeper", "auditor", "probe"]),
  prompt_file: z.string().min(1),
  allowed_tools: z.array(z.string()).default([]),
}).superRefine((data, ctx) => {
  const resolved = resolve(data.prompt_file);
  if (!existsSync(resolved)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: `prompt_file does not exist: ${resolved}`,
      path: ["prompt_file"],
    });
  }
});

export type Agent = z.infer<typeof AgentSchema>;
```

### Pattern 2: NDJSON Event Schema
**What:** A Zod schema that validates each line of an NDJSON stream.
**When to use:** Event stream validation (Phase 158).
**Example:**
```typescript
// Source: NDJSON spec github.com/ndjson/ndjson-spec
import { z } from "zod";

export const EventSchema = z.object({
  type: z.string().min(1),
  timestamp: z.string().datetime({ offset: true }),
  payload: z.record(z.unknown()),
});

export type ColonyEvent = z.infer<typeof EventSchema>;

// Line-by-line validation helper
export function validateEventLine(line: string): ColonyEvent {
  const trimmed = line.trim();
  if (!trimmed) throw new Error("Empty event line");
  const parsed = JSON.parse(trimmed);
  return EventSchema.parse(parsed);
}
```

### Pattern 3: YAML Fixture Testing
**What:** Load a YAML file in a test, parse it, and assert the Zod schema accepts it.
**When to use:** All schema test files (`npm run test:schemas`).
**Example:**
```typescript
// Source: Vitest docs + Zod docs
import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { parse } from "yaml";
import { AgentSchema } from "../../src/schemas/agent.schema";

describe("AgentSchema", () => {
  it("validates a sample queen agent YAML", () => {
    const yamlText = readFileSync("tests/fixtures/agents/queen.yaml", "utf-8");
    const data = parse(yamlText);
    const result = AgentSchema.parse(data);
    expect(result.id).toBe("queen");
    expect(result.role).toBe("queen");
  });
});
```

### Anti-Patterns to Avoid
- **Using `any` in schemas:** Use `z.unknown()` instead, then narrow with `.transform()` or `.refine()`.
- **Synchronous `fs.existsSync` in hot paths:** Acceptable at load time (one-off), but never inside a loop processing many files.
- **Bundling a web framework:** This is a Node library; Vite/Next.js/Remix add unnecessary complexity and dependency weight.
- **Validating NDJSON by loading the whole file:** Always stream line-by-line; files may grow unbounded.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| YAML parsing | Custom regex parser | `yaml` (eemeli) | YAML 1.2 spec compliance, anchors, merges |
| Schema validation | Manual `if/else` checks | Zod | Type inference, error messages, composability |
| TS execution | `ts-node` with heavy config | `tsx` | Zero-config, fast, handles ESM |
| Test runner | Custom runner script | Vitest | Watch mode, coverage, snapshot, native TS |
| File path resolution | String concatenation | `path.resolve` | Cross-platform, absolute path guarantees |

**Key insight:** The control plane's job is to validate and orchestrate, not to reinvent parsing or testing infrastructure. Every custom parser is a future bug.

## Runtime State Inventory

> This phase is greenfield — no rename/refactor/migration. No runtime state to inventory.

## Common Pitfalls

### Pitfall 1: Zod `refine` vs `superRefine`
**What goes wrong:** Using `.refine()` when you need to emit multiple issues (e.g., file missing AND path not absolute). `.refine()` stops at the first failure and only allows one message.
**Why it happens:** `.refine()` is simpler to write; developers reach for it by default.
**How to avoid:** Use `.superRefine()` whenever you need to validate multiple independent conditions or emit multiple error messages. It gives you `ctx.addIssue()` for each problem.
**Warning signs:** A schema that checks both file existence and file type but only reports one error at a time.

### Pitfall 2: YAML `null` vs `undefined`
**What goes wrong:** YAML `~` or empty values parse as `null`, but Zod `.optional()` expects `undefined`. This causes validation failures on seemingly valid YAML.
**Why it happens:** YAML spec treats `~` and empty scalar as `null`; JavaScript/TypeScript distinguishes `null` and `undefined`.
**How to avoid:** Use `z.union([z.string(), z.null()]).optional()` or preprocess with `.transform((v) => v ?? undefined)` when loading YAML.
**Warning signs:** Tests fail with "Expected string, received null" on fields that look optional in YAML.

### Pitfall 3: Relative path resolution from wrong cwd
**What goes wrong:** `fs.existsSync(path)` resolves relative to `process.cwd()`, which may be `control-ts/` during tests but repo root during runtime. A path like `colony/prompts/queen.md` fails in one context and passes in another.
**Why it happens:** Node's `fs` module uses `process.cwd()` for relative paths; tests and runtime have different working directories.
**How to avoid:** Always resolve paths relative to a known anchor (e.g., `__dirname`, `import.meta.dirname`, or a configurable `projectRoot`). Pass the anchor as a parameter to loader functions.
**Warning signs:** Schema tests pass when run from `control-ts/` but fail when run from repo root, or vice versa.

### Pitfall 4: NDJSON lines containing embedded newlines
**What goes wrong:** A JSON string value contains an unescaped `\n`, breaking the line-oriented format. The NDJSON parser treats it as two separate events.
**Why it happens:** JSON.stringify does not strip newlines from string values; manual string concatenation can inject them.
**How to avoid:** Always use `JSON.stringify()` for emission. When parsing, split on `\n` and validate each line independently. Reject lines that fail `JSON.parse()` rather than crashing the stream.
**Warning signs:** Event stream consumers report "Unexpected token" errors mid-stream, or events appear truncated.

## Code Examples

### Agent Schema (SCHEMA-01)
```typescript
// Source: zod.dev + verified patterns
import { z } from "zod";
import { existsSync } from "fs";
import { resolve } from "path";

export const AgentSchema = z.object({
  id: z.string().min(1, "id is required"),
  role: z.enum([
    "builder", "watcher", "scout", "queen",
    "oracle", "gatekeeper", "auditor", "probe"
  ]),
  prompt_file: z.string().min(1, "prompt_file is required"),
  allowed_tools: z.array(z.string()).default([]),
}).superRefine((data, ctx) => {
  const resolved = resolve(data.prompt_file);
  if (!existsSync(resolved)) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: `Referenced prompt_file does not exist: ${resolved}`,
      path: ["prompt_file"],
    });
  }
});

export type Agent = z.infer<typeof AgentSchema>;
```

### Phase Schema (SCHEMA-02)
```typescript
// Source: REQUIREMENTS.md §SCHEMA-02
import { z } from "zod";

export const PhaseSchema = z.object({
  id: z.string().min(1),
  entry_agent: z.string().min(1),
  required_agents: z.array(z.string()).min(1),
  inputs: z.record(z.unknown()).default({}),
  outputs: z.array(z.string()).default([]),
  success_criteria: z.array(z.string()).min(1),
  failure_policy: z.enum(["retry", "skip", "block", "escalate"]).default("retry"),
  ceremony: z.object({
    stages: z.array(z.string()).default([]),
  }).optional(),
});

export type Phase = z.infer<typeof PhaseSchema>;
```

### Event Schema (SCHEMA-03)
```typescript
// Source: NDJSON spec + zod.dev
import { z } from "zod";

export const EventSchema = z.object({
  type: z.string().min(1),
  timestamp: z.string().datetime({ offset: true }),
  payload: z.record(z.unknown()),
});

export type ColonyEvent = z.infer<typeof EventSchema>;

export function parseEventLine(line: string): ColonyEvent {
  const trimmed = line.trim();
  if (!trimmed) throw new Error("Empty line");
  return EventSchema.parse(JSON.parse(trimmed));
}
```

### Policy Schema (SCHEMA-04)
```typescript
// Source: REQUIREMENTS.md §SCHEMA-04
import { z } from "zod";

export const PolicySchema = z.object({
  model_routing: z.object({
    default_provider: z.string().min(1),
    fallback_providers: z.array(z.string()).default([]),
    routing_rules: z.array(z.object({
      agent_role: z.string(),
      provider: z.string(),
      model: z.string().optional(),
    })).default([]),
  }).default({}),
  memory_rules: z.object({
    max_learnings: z.number().int().positive().default(100),
    auto_promote_threshold: z.number().default(0.75),
  }).default({}),
  skill_creation: z.object({
    allowed: z.boolean().default(true),
    auto_approve: z.boolean().default(false),
  }).default({}),
  safety_gates: z.object({
    security_scan: z.boolean().default(true),
    quality_gate: z.boolean().default(true),
  }).default({}),
});

export type Policy = z.infer<typeof PolicySchema>;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Joi / Yup for TS validation | Zod | 2021+ | Better type inference, smaller bundle, composable |
| Jest + ts-jest | Vitest | 2022+ | Native ESM/TS, no transform pipeline duplication |
| ts-node | tsx | 2023+ | Faster, zero config, handles modern TS features |
| js-yaml | `yaml` (eemeli) | 2020+ | YAML 1.2 spec compliance, better anchor/merge support |

**Deprecated/outdated:**
- `ts-node` for CLI scripts: slower, more config, less reliable with ESM.
- `Joi` for TypeScript: poor inference, larger bundle, class-based API less ergonomic.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `yaml` package (eemeli) parses all colony YAML files correctly | Standard Stack | If it fails on anchors/merges in `.aether/commands/*.yaml`, we may need js-yaml fallback |
| A2 | `existsSync` is acceptable for one-time load-time validation | Pattern 1 | If asset counts grow to 1000+, async `fs.promises.access` may be needed |
| A3 | Vitest `environment: 'node'` is sufficient; no browser/DOM testing needed | Standard Stack | If future phases add TUI/web UI, test env would need updating |
| A4 | Zod v3.x is stable enough; v4 migration not needed for this milestone | Standard Stack | If Zod v4 breaks APIs, schema files need rewrite |

## Open Questions

1. **Where does `control-ts/` live relative to the Go repo?**
   - What we know: It will be a subdirectory `control-ts/` inside the Aether repo.
   - What's unclear: Whether it should be a separate npm package/workspace or just a folder.
   - Recommendation: Start as a plain folder with its own `package.json`. Do not introduce npm workspaces or monorepo tooling unless Phase 156+ demands it.

2. **How are sample YAML fixtures kept in sync with real colony assets?**
   - What we know: Tests need fixtures; real assets live in `colony/` (created in Phase 154).
   - What's unclear: Whether fixtures should be copies, symlinks, or generated from schemas.
   - Recommendation: Use hand-written fixtures in `tests/fixtures/` that mirror the expected shape. They serve as contract tests — if Phase 154 changes the shape, these tests catch it.

3. **Should schemas live in `control-ts/` only, or also be generated for Go?**
   - What we know: Go runtime will load YAML and needs to validate it too.
   - What's unclear: Whether to generate Go structs from Zod schemas, or maintain parallel validation.
   - Recommendation: Keep Zod as the source of truth for now. Go can do lightweight validation (required fields) at load time. Full cross-language schema generation is out of scope for v1.24.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Runtime + tests | Yes | v26.0.0 | — |
| npm | Package management | Yes | 11.12.1 | — |
| npx | CLI execution | Yes | 11.12.1 | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest 3.2.x |
| Config file | `control-ts/vitest.config.ts` |
| Quick run command | `npm run test` (runs all tests) |
| Full suite command | `npm run test:schemas` (dedicated schema validation suite) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CONTROL-01 | `npm install` builds without errors | smoke | `cd control-ts && npm install && npx tsc --noEmit` | No — Wave 0 |
| SCHEMA-01 | Agent schema validates sample YAML | unit | `npx vitest run tests/schemas/agent.schema.test.ts` | No — Wave 0 |
| SCHEMA-02 | Phase schema validates sample YAML | unit | `npx vitest run tests/schemas/phase.schema.test.ts` | No — Wave 0 |
| SCHEMA-03 | Event schema validates NDJSON lines | unit | `npx vitest run tests/schemas/event.schema.test.ts` | No — Wave 0 |
| SCHEMA-04 | Policy schema validates sample YAML | unit | `npx vitest run tests/schemas/policy.schema.test.ts` | No — Wave 0 |

### Sampling Rate
- **Per task commit:** `npx vitest run` (fast, <5s expected)
- **Per wave merge:** `npm run test:schemas` (full schema suite)
- **Phase gate:** All schema tests green + `tsc --noEmit` passes before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `control-ts/package.json` — project manifest
- [ ] `control-ts/tsconfig.json` — strict TypeScript config
- [ ] `control-ts/vitest.config.ts` — test runner config
- [ ] `control-ts/tests/fixtures/` — sample YAML/NDJSON fixtures
- [ ] `control-ts/tests/schemas/*.test.ts` — four schema test files
- [ ] `control-ts/src/schemas/*.schema.ts` — four schema source files

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Not in scope for this phase |
| V3 Session Management | No | Not in scope for this phase |
| V4 Access Control | No | Not in scope for this phase |
| V5 Input Validation | Yes | Zod schemas validate all YAML/JSON inputs |
| V6 Cryptography | No | Not in scope for this phase |

### Known Threat Patterns for {stack}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in `prompt_file` | Tampering | Resolve paths against a known `projectRoot`; reject `..` segments |
| YAML bomb / excessive anchors | Denial of Service | Limit YAML parse depth/document size; use `yaml` package options |
| Malformed NDJSON injection | Tampering | Per-line JSON.parse with try/catch; reject invalid lines |

## Sources

### Primary (HIGH confidence)
- [npm registry] — `zod@3.25.2`, `yaml@2.9.0`, `vitest@3.2.7`, `tsx@4.22.3`, `typescript@5.8.3`
- [zod.dev](https://zod.dev) — Schema validation API, error customization, `superRefine`
- [vitest.dev](https://vitest.dev) — Test runner config, Node environment, typecheck
- [NDJSON spec](https://github.com/ndjson/ndjson-spec) — Line delimiters, encoding, parsing rules

### Secondary (MEDIUM confidence)
- [WebSearch: NDJSON best practices](https://ndjson.com/use-cases/data-streaming/) — Event stream patterns, schema evolution
- [WebSearch: Zod file path validation](https://search.brave.com/search?q=Zod+schema+validate+file+path+exists+Node.js+fs+TypeScript+pattern) — Community patterns for `refine` + `fs.existsSync`

### Tertiary (LOW confidence)
- None — all claims verified against primary sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against npm registry
- Architecture: HIGH — derived from Phase 152 boundary docs and REQUIREMENTS.md
- Pitfalls: MEDIUM-HIGH — based on Zod docs + common Node.js patterns; some edge cases (YAML null handling) are experiential

**Research date:** 2026-05-22
**Valid until:** 2026-06-22 (Zod and Vitest are stable; versions unlikely to change meaningfully within 30 days)
