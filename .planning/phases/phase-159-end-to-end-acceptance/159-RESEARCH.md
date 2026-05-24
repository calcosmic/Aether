# Phase 159: End-to-End Acceptance — Research

**Researched:** 2026-05-24
**Domain:** Hybrid system validation — Go runtime spine + TypeScript control plane + editable colony assets
**Confidence:** HIGH

## Summary

Phase 159 is the capstone validation phase of the v1.24 Hybrid Architecture Salvage milestone. All prior phases (152-158) have built the pieces: architecture boundary docs, Classic parity checklist, behaviour extraction audit, TypeScript control plane with schemas/loaders/runner/orchestrator, colony assets (agents, prompts, phases, playbooks, policies), Go boundary refactor loading from files, and the NDJSON event stream. Phase 159 validates that these pieces integrate correctly and that no hardcoded behaviour remains in Go.

The primary research finding is that the codebase is in excellent shape for acceptance testing. All 132 TypeScript tests pass, all Go tests pass, colony assets are fully populated (27 agents, 9 phases, 28 prompts, 7 playbooks, 9 policies), and Go already loads visuals, prompts, policies, and dispatch contracts from files. The remaining work is: (1) add missing npm scripts (`test:control`, `aether:control`), (2) build a demo CLI entry point, (3) run grep/audits for hardcoded behaviour, (4) verify the parity checklist coverage, and (5) wire everything together for end-to-end demonstration.

**Primary recommendation:** Build a minimal `aether:control` CLI in `control-ts/src/cli.ts` that accepts `--task` and runs `executePlan` with event emission, add the missing npm scripts, create a `test:control` test suite covering runner + orchestrator integration, and run automated audits for hardcoded prompt text and agent behaviour in Go.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Go test validation | Go runtime spine | — | Go owns its own test suite; no regressions allowed |
| Schema validation | TypeScript control plane | — | Zod schemas validate colony asset files |
| Control plane test | TypeScript control plane | — | Phase runner + orchestrator tests live in TS |
| Demo flow execution | TypeScript control plane | Go runtime (event stream) | TS orchestrator drives the flow; Go emits events |
| Hardcoded behaviour audit | Static analysis (grep) | — | Automated audit of Go source for prompt/agent strings |
| Extraction audit verification | Documentation | Static analysis | Compare audit classifications against actual code |
| Classic parity verification | Documentation | Manual + automated | Checklist in `.aether/docs/PARITY_CLASSIC_VS_GO.md` |

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Zod | 4.4.3 | Schema validation | Already used; validates all colony assets [VERIFIED: local package.json] |
| yaml | 2.9.0 | YAML parsing | Already used; parses colony asset files [VERIFIED: local package.json] |
| vitest | 4.1.7 | Test runner | Already configured; 132 tests passing [VERIFIED: runtime check] |
| TypeScript | 6.0.3 | Type safety | Already configured with strict mode [VERIFIED: local package.json] |
| Go | 1.26.1 | Runtime spine | All tests passing, binary builds [VERIFIED: runtime check] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| tsx | 4.22.3 | TS execution | For `aether:control` CLI script and demo flow [VERIFIED: local package.json] |
| Node.js fs | built-in | File I/O | Event stream writes, state reads [VERIFIED: runtime check] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| tsx for CLI | compiled `dist/` + node | tsx is faster for dev/demo; no build step needed for acceptance |
| vitest for `test:control` | separate test file + `vitest run tests/orchestrator` | Using vitest's file filter is cleaner; no new test framework needed |

**Version verification:**
```bash
node --version   # v26.0.0
npm --version    # 11.12.1
go version       # go1.26.1 darwin/arm64
npx vitest --version  # vitest/4.1.7
```

---

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                     ACCEPTANCE VALIDATION                        │
│                                                                  │
│   npm run test:schemas    →  Zod validates colony/*.yaml        │
│   npm run test:control    →  runPhase + executePlan tests       │
│   npm run aether:control  →  CLI demo: init→plan→build→verify→seal│
│   go test ./...           →  Go runtime spine regression-free   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌─────────────────┐    ┌───────────────┐
│  Go Runtime   │    │  TS Control Plane│    │  Colony Assets │
│   (cmd/)      │◄──►│  (control-ts/)   │    │  (colony/)     │
│               │NDJSON│                │    │                │
│  - event stream│    │  - runPhase.ts   │◄───│  - agents/*.yaml│
│  - state mutate│    │  - executePlan.ts│    │  - phases/*.yaml│
│  - tests pass │    │  - schemas/*.ts  │    │  - prompts/*.md │
│               │    │  - adapters/*.ts │    │  - policies/*.yaml│
└───────────────┘    └─────────────────┘    └───────────────┘
```

### Recommended Project Structure (for this phase)

```
control-ts/
├── package.json              # ADD: test:control, aether:control scripts
├── src/
│   ├── cli.ts                # NEW: CLI entry point for demo flow
│   └── ... (existing modules)
└── tests/
    ├── schemas/              # EXISTING: 25 schema tests passing
    ├── orchestrator/         # EXISTING: runPhase + executePlan tests
    └── control/              # NEW: integration test for test:control
```

### Pattern 1: Demo CLI with Task Argument
**What:** A TypeScript CLI that accepts `--task "description"`, runs a phase sequence, and emits events.
**When to use:** For TEST-04 (demo flow runs end-to-end).
**Example:**
```typescript
// Source: inferred from executePlan.ts and runPhase.ts patterns
import { executePlan } from "./orchestrator/executePlan.js";
import { parseEventLine } from "./schemas/event.schema.js";
import { readFileSync } from "fs";

async function main() {
  const task = process.argv.includes("--task")
    ? process.argv[process.argv.indexOf("--task") + 1]
    : "default task";

  const result = await executePlan(["init", "plan", "build"]);

  // Verify events were written
  const eventsFile = process.env.AETHER_EVENTS_FILE || ".aether/events/current.ndjson";
  const lines = readFileSync(eventsFile, "utf8").trim().split("\n").filter(Boolean);
  const events = lines.map(parseEventLine);

  console.log("Status:", result.status);
  console.log("Events:", events.length);
  console.log("Types:", events.map((e) => e.type));
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

### Pattern 2: npm Script as Test Gate
**What:** `npm run test:control` runs a specific vitest filter covering phase runner + orchestrator.
**When to use:** For TEST-03 (TS control plane test passes).
**Example:**
```json
{
  "scripts": {
    "test:control": "vitest run tests/orchestrator tests/agents tests/phases tests/prompts tests/skills tests/memory",
    "aether:control": "tsx src/cli.ts"
  }
}
```

### Pattern 3: Grep Audit for Hardcoded Behaviour
**What:** Automated shell script that greps Go source for patterns indicating hardcoded prompt/agent/phase text.
**When to use:** For TEST-05 and TEST-06.
**Example:**
```bash
#!/bin/bash
# audit-hardcoded.sh

echo "=== Hardcoded prompt text audit ==="
# Look for agent directive patterns in non-test Go files
grep -rn '"You are' cmd/ --include="*.go" | grep -v '_test.go' | grep -v 'fallback\|error\|fmt.Errorf'

echo "=== Hardcoded phase ritual audit ==="
# Look for ceremony/ritual strings
grep -rn 'ceremony\|ritual\|playbook' cmd/ --include="*.go" -l | grep -v '_test.go'

echo "=== Agent behaviour only in Go ==="
# Cross-reference: agents exist in colony/agents/ but behaviour strings still in Go
grep -rn 'builder\|watcher\|scout\|queen' cmd/ --include="*.go" -l | wc -l
```

### Anti-Patterns to Avoid
- **Do NOT run actual LLM API calls in acceptance tests.** The demo flow should use stub adapters (already built in Phase 157).
- **Do NOT modify Go runtime behaviour during acceptance.** This phase validates; it does not refactor.
- **Do NOT skip fallback text in the audit.** The requirement explicitly allows fallback/error text in Go — the audit must distinguish between "behaviour definition" (forbidden) and "fallback/error" (allowed).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test runner for control plane | Custom test harness | vitest (already configured) | 132 tests already passing; adding a filter is trivial |
| CLI argument parsing | Manual argv slicing | `process.argv` with simple checks | For a demo script, minimist is overkill; simple argv parsing is sufficient |
| NDJSON event validation | Custom parser | `parseEventLine` from `event.schema.ts` | Already tested and handles error cases gracefully |
| Colony state read/write | Direct fs calls without atomicity | `memory/store.ts` (atomic rename) | Already implemented and tested |
| Hardcoded behaviour detection | Manual code review only | `grep` + `152-BEHAVIOUR_EXTRACTION_AUDIT.md` | The audit doc already classified 94 symbols; automate the verification |

**Key insight:** All the infrastructure needed for acceptance already exists. The phase is about wiring, scripting, and verification — not building new complex systems.

---

## Common Pitfalls

### Pitfall 1: False Positives in Hardcoded Behaviour Audit
**What goes wrong:** The grep flags legitimate fallback strings, error messages, or test fixtures as "hardcoded behaviour."
**Why it happens:** Simple keyword matching cannot distinguish between "Your task is to..." as an agent prompt vs. "Your task is to..." as an error message or test data.
**How to avoid:** Use the `152-BEHAVIOUR_EXTRACTION_AUDIT.md` as the authoritative map. Cross-reference grep findings against the audit classifications. Only flag items classified as KEEP_IN_GO or MIXED that still contain behaviour strings after Phase 155.
**Warning signs:** Audit report shows strings from `_test.go` files or `fmt.Errorf` calls.

### Pitfall 2: Event Stream Path Mismatch Between Go and TS
**What goes wrong:** Go writes to `.aether/events/current.ndjson` while TS reads from a different path (e.g., `/tmp/events.ndjson` in tests).
**Why it happens:** Tests use `process.env.AETHER_EVENTS_FILE` overrides; the demo CLI might not set the same path.
**How to avoid:** Use `GetEventStreamPath()` (Go) and `resolve(projectRoot, "..", ".aether", "events", "current.ndjson")` (TS) consistently. The demo CLI should not override the path unless explicitly requested.
**Warning signs:** Demo runs successfully but no events appear in `.aether/events/current.ndjson`.

### Pitfall 3: Colony State Collision Between Tests and Demo
**What goes wrong:** Running the demo CLI overwrites `COLONY_STATE.json` used by tests, or vice versa.
**Why it happens:** Both `executePlan` and tests write to the same `.aether/data/COLONY_STATE.json` path.
**How to avoid:** Tests already use `cleanupState()` to remove the file before/after. The demo CLI should document that it mutates colony state. For CI/isolation, use `AETHER_EVENTS_FILE` and a temp state path.
**Warning signs:** Tests fail after running the demo, or demo shows unexpected state from prior test runs.

### Pitfall 4: Missing npm Scripts Breaking Success Criteria
**What goes wrong:** `npm run test:control` or `npm run aether:control` do not exist, making TEST-03 and TEST-04 impossible to verify.
**Why it happens:** These scripts are referenced in REQUIREMENTS.md and ROADMAP.md but do not exist in `control-ts/package.json`.
**How to avoid:** Add the scripts as part of this phase. Verify they work before declaring the phase complete.
**Warning signs:** Running the documented commands produces "Missing script" errors.

### Pitfall 5: Parity Checklist Coverage Below 50%
**What goes wrong:** The parity checklist has 15+ items but fewer than 8 are verifiable against the hybrid system.
**Why it happens:** Some items (e.g., "Probe coverage analysis") are marked GAP or PENDING in the checklist. The 50% threshold requires at least 8 verifiable items.
**How to avoid:** Count the items explicitly. The checklist already has 15 rows with statuses: MATCH (10), DEGRADED (2), GAP (1), INTENTIONALLY_CHANGED (6 in Known Gaps). The MATCH + DEGRADED items (12) are verifiable. 12/15 = 80%, which exceeds 50%.
**Warning signs:** Disagreement about what counts as "verifiable."

---

## Code Examples

### Verified patterns from official sources:

### Running a phase and checking events
```typescript
// Source: control-ts/tests/orchestrator/runPhase.test.ts
import { runPhase } from "../../src/orchestrator/runPhase.js";
import { parseEventLine } from "../../src/schemas/event.schema.js";
import { readFileSync } from "fs";

const result = await runPhase("init");
expect(result.status).toBe("completed");
expect(result.events.map((e) => e.type)).toContain("phase:start");

// Verify NDJSON file
const raw = readFileSync(eventsFile, "utf8");
const lines = raw.trim().split("\n").filter(Boolean);
for (const line of lines) {
  const ev = parseEventLine(line);
  expect(ev.type).toBeDefined();
  expect(ev.timestamp).toBeDefined();
}
```

### Executing a plan sequence
```typescript
// Source: control-ts/tests/orchestrator/executePlan.test.ts
import { executePlan } from "../../src/orchestrator/executePlan.js";

const result = await executePlan(["init", "plan", "build"]);
expect(result.status).toBe("completed");
expect(result.results.length).toBe(3);
expect(result.results[0].phaseId).toBe("init");
```

### Loading and validating colony assets
```typescript
// Source: control-ts/src/agents/loadAgents.ts
import { loadAgents } from "../../src/agents/loadAgents.js";

const agents = loadAgents();
expect(agents).toHaveLength(27);
for (const agent of agents) {
  expect(agent).toHaveProperty("id");
  expect(agent).toHaveProperty("role");
  expect(agent).toHaveProperty("prompt_file");
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded visuals in Go (`cmd/codex_visuals.go`) | Load from `colony/ceremony/visuals.md` | Phase 155 | Visuals are human-editable without recompiling |
| Hardcoded prompt sections in Go (`cmd/colony_prime_context.go`) | Load from `colony/prompts/colony-prime.md` | Phase 155 | Prompt templates editable in Markdown |
| Hardcoded dispatch contract rules | Load from `colony/policies/dispatch-contract.yaml` | Phase 155 | Spawn budgets and timeouts configurable |
| Hardcoded review depth keywords | Load from `colony/policies/review-depth.yaml` | Phase 155 | Depth rules configurable |
| Go-only event bus | Go + TS share NDJSON stream | Phase 158 | Cross-platform observable truth |
| Go-only orchestration | TS control plane with Go spine | Phase 156 | Faster iteration on orchestration logic |

**Deprecated/outdated:**
- Direct `COLONY_STATE.json` writes from playbooks: replaced by `state-mutate` (Phase 145)
- Silent error suppression on data-persistence CLI calls: removed (Phase 145)

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The 5 hardcoded "Your task is to..." strings in `cmd/oracle_loop.go` are the only remaining agent directive strings in non-test Go files | Standard Stack / Audit | If more exist, the audit underreports hardcoded behaviour |
| A2 | `152-BEHAVIOUR_EXTRACTION_AUDIT.md` is current as of Phase 155 completion | Don't Hand-Roll | If Phase 155 introduced new hardcoded strings after the audit, the audit is stale |
| A3 | The parity checklist's 12 MATCH/DEGRADED items out of 15 total meets the "at least 50% verifiable" threshold | Common Pitfalls | If DEGRADED items do not count as "verifiable," the threshold drops to 10/15 = 67%, which still passes |
| A4 | Stub adapters (Phase 157) are sufficient for the demo flow; no real LLM calls needed | Code Examples | If the demo is expected to produce real output, stubs are insufficient |
| A5 | `npm run test:control` can be implemented as a vitest filter over existing test directories | Architecture Patterns | If new integration tests are needed beyond existing tests, the script definition changes |

---

## Open Questions

1. **Does the demo CLI need to perform actual file I/O (create a test file), or is running the phase sequence sufficient?**
   - What we know: TEST-04 says "create a small test file and verify it."
   - What's unclear: Whether this means the TS control plane writes a file via an adapter, or just runs phases.
   - Recommendation: Implement the demo as `executePlan(["init", "plan", "build"])` with event emission. The "create a test file" can be simulated by the builder adapter stub writing to a temp path. Document this as simulated.

2. **Should the hardcoded behaviour audit be a one-off script or a permanent CI check?**
   - What we know: TEST-05 and TEST-06 are acceptance criteria for this phase.
   - What's unclear: Whether the audit script should be checked into the repo for future regression detection.
   - Recommendation: Create `scripts/audit-hardcoded.sh` and run it as part of this phase. Suggest making it a CI step in a future milestone.

3. **What is the exact definition of "verifiable" for the parity checklist 50% threshold?**
   - What we know: The checklist has MATCH, DEGRADED, GAP, and INTENTIONALLY_CHANGED statuses.
   - What's unclear: Whether DEGRADED counts as "verifiable" (it means Classic and Go disagree but the behaviour is observable).
   - Recommendation: Count MATCH + DEGRADED as verifiable = 12/15 = 80%. Document this counting method explicitly.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS control plane | Yes | v26.0.0 | — |
| npm | Package management | Yes | 11.12.1 | — |
| Go | Runtime spine | Yes | 1.26.1 | — |
| vitest | Test runner | Yes | 4.1.7 | — |
| tsx | TS execution | Yes | 4.22.3 | — |
| colony/ assets | Loaders, schemas | Yes | — | — |
| .aether/ data dir | State, events | Yes | — | — |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None.

---

## Validation Architecture

> workflow.nyquist_validation is absent in .planning/config.json — treated as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | vitest 4.1.7 |
| Config file | `control-ts/vitest.config.ts` |
| Quick run command | `cd control-ts && npm test` |
| Full suite command | `cd control-ts && npm run test` (same as quick — all tests run fast) |
| Go test command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TEST-01 | Go tests pass with no regressions | integration | `go test ./...` | Yes — all passing |
| TEST-02 | Zod schemas validate against sample YAML | unit | `cd control-ts && npm run test:schemas` | Yes — 25 passing |
| TEST-03 | Phase runner and orchestrator tests pass | unit | `cd control-ts && npm run test:control` | No — script missing |
| TEST-04 | Demo flow runs end-to-end with events | integration | `cd control-ts && npm run aether:control -- --task "..."` | No — script and CLI missing |
| TEST-05 | No prompt text hardcoded in Go (except fallback/error) | static analysis | `scripts/audit-hardcoded.sh` | No — script missing |
| TEST-06 | No agent behaviour exists only in Go | static analysis | Cross-reference audit doc + grep | Partial — audit exists, automation missing |
| TEST-07 | Classic parity checklist exists and >=50% verifiable | documentation | Manual verification against `.aether/docs/PARITY_CLASSIC_VS_GO.md` | Yes — doc exists |

### Sampling Rate
- **Per task commit:** `cd control-ts && npm test` + `go test ./...`
- **Per wave merge:** Full suite (same as quick — tests are fast)
- **Phase gate:** All 7 success criteria must be verifiable before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `control-ts/package.json` — add `test:control` and `aether:control` scripts
- [ ] `control-ts/src/cli.ts` — CLI entry point for demo flow
- [ ] `scripts/audit-hardcoded.sh` — automated grep audit for hardcoded behaviour
- [ ] `tests/control/` or integration test file — covers TEST-03 explicitly

*(Existing test infrastructure covers TEST-01, TEST-02, and partially TEST-06/07 via docs.)*

---

## Security Domain

> This phase is validation-only; no new security-sensitive code is written. The existing security posture from prior phases applies.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth changes in this phase |
| V3 Session Management | No | No session changes |
| V4 Access Control | No | No access control changes |
| V5 Input Validation | Yes | Zod schemas validate all colony asset inputs |
| V6 Cryptography | No | No crypto changes |

### Known Threat Patterns for This Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in prompt_file | Tampering | AgentSchema superRefine rejects `..` in paths [VERIFIED: agent.schema.ts] |
| Malformed YAML causing crashes | Denial of Service | Loaders catch parse errors and throw with context [VERIFIED: loadAgents.ts, loadPhases.ts] |
| NDJSON injection | Tampering | EventSchema validates structure before write [VERIFIED: event.schema.ts] |

---

## Sources

### Primary (HIGH confidence)
- `control-ts/package.json` — dependency versions and scripts [VERIFIED: local file]
- `control-ts/vitest.config.ts` — test configuration [VERIFIED: local file]
- `control-ts/src/orchestrator/runPhase.ts` — phase runner implementation [VERIFIED: local file]
- `control-ts/src/orchestrator/executePlan.ts` — plan executor implementation [VERIFIED: local file]
- `cmd/event_types.go`, `cmd/event_writer.go`, `cmd/event_reader.go` — Go event stream [VERIFIED: local file]
- `cmd/prompt_template_loader.go` — Go prompt loading from file [VERIFIED: local file]
- `cmd/visuals_config.go` — Go visuals loading from file [VERIFIED: local file]
- `cmd/policy_loader.go` — Go policy loading from file [VERIFIED: local file]
- `cmd/review_depth.go` — Go review depth loading from file [VERIFIED: local file]
- `cmd/codex_dispatch_contract.go` — Go dispatch contract loading from file [VERIFIED: local file]
- `.aether/docs/PARITY_CLASSIC_VS_GO.md` — parity checklist [VERIFIED: local file]
- `.planning/phases/152-boundary-parity/152-BEHAVIOUR_EXTRACTION_AUDIT.md` — extraction audit [VERIFIED: local file]

### Secondary (MEDIUM confidence)
- Phase 155 research (`155-RESEARCH.md`) — boundary refactor findings [CITED: local planning doc]
- Phase 156 research (`156-RESEARCH.md`) — control plane core findings [CITED: local planning doc]
- Phase 157 research (`157-RESEARCH.md`) — adapters and oracle findings [CITED: local planning doc]
- Phase 158 research (`158-RESEARCH.md`) — event stream findings [CITED: local planning doc]

### Tertiary (LOW confidence)
- None — all claims verified against local source files or official docs.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all versions verified against local files and runtime
- Architecture: HIGH — all modules exist and are tested; only wiring remains
- Pitfalls: HIGH — based on direct observation of code patterns and test behaviour

**Research date:** 2026-05-24
**Valid until:** 2026-06-24 (stable stack, low churn expected)
