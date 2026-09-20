# Phase 159: End-to-End Acceptance — Pattern Map

**Mapped:** 2026-05-24
**Files analyzed:** 7
**Analogs found:** 7 / 7

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `control-ts/package.json` | config | request-response | `control-ts/package.json` (existing) | exact |
| `control-ts/src/cli.ts` | utility | request-response | `control-ts/src/orchestrator/executePlan.ts` | role-match |
| `control-ts/tests/control/integration.test.ts` | test | request-response | `control-ts/tests/orchestrator/executePlan.test.ts` | exact |
| `scripts/audit-hardcoded.sh` | utility | batch | `scripts/smoke-test-classic.sh` | role-match |
| `cmd/oracle_loop.go` (read-only audit target) | service | request-response | `cmd/oracle_loop.go` | exact |
| `control-ts/src/orchestrator/runPhase.ts` (read-only pattern source) | service | event-driven | `control-ts/src/orchestrator/runPhase.ts` | exact |
| `control-ts/src/orchestrator/executePlan.ts` (read-only pattern source) | service | event-driven | `control-ts/src/orchestrator/executePlan.ts` | exact |

---

## Pattern Assignments

### `control-ts/package.json` (config, request-response)

**Analog:** `control-ts/package.json` (self — modify existing)

**Existing scripts pattern** (lines 11-16):
```json
{
  "scripts": {
    "build": "tsc",
    "typecheck": "tsc --noEmit",
    "test": "vitest run",
    "test:schemas": "vitest run tests/schemas"
  }
}
```

**Pattern to copy:** Add new scripts following the same `vitest run <path>` and `tsx <entry>` conventions.

---

### `control-ts/src/cli.ts` (utility, request-response)

**Analog:** `control-ts/src/orchestrator/executePlan.ts`

**Imports pattern** (lines 1-9):
```typescript
import { appendFileSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { loadPhaseById } from "../phases/loadPhases.js";
import { runPhase } from "./runPhase.js";
import { readColonyState, updateColonyState } from "../memory/store.js";
import { EventSchema, type ColonyEvent } from "../schemas/event.schema.js";
import { projectRoot } from "../utils/projectRoot.js";
import type { PhaseResult } from "../types/runtime.js";
```

**Core pattern — executePlan with event emission** (lines 47-176):
```typescript
export async function executePlan(
  sequence: string[],
  options?: {
    inputs?: Record<string, Record<string, unknown>>;
    maxRetries?: number;
  }
): Promise<PlanResult> {
  ensureEventsDir();
  const startedAt = new Date().toISOString();
  const events: ColonyEvent[] = [];

  const planStartEvent: ColonyEvent = {
    type: "plan:start",
    timestamp: startedAt,
    payload: { sequence, startedAt },
  };
  events.push(emitEvent(planStartEvent));

  // ... run phases, collect results ...

  const completedAt = new Date().toISOString();
  const planCompleteEvent: ColonyEvent = {
    type: "plan:complete",
    timestamp: completedAt,
    payload: { status, completedAt, phaseCount: results.length },
  };
  events.push(emitEvent(planCompleteEvent));

  return { status, results, events, startedAt, completedAt };
}
```

**Error handling pattern** (lines 81-91):
```typescript
if (!phase) {
  const notFoundResult: PhaseResult = {
    phaseId,
    agentId: "",
    status: "failed",
    events: [],
    error: "Phase not found",
  };
  results.push(notFoundResult);
  break;
}
```

**CLI-specific pattern to add:**
```typescript
// Simple argv parsing (no external deps)
const task = process.argv.includes("--task")
  ? process.argv[process.argv.indexOf("--task") + 1]
  : "default task";

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
```

---

### `control-ts/tests/control/integration.test.ts` (test, request-response)

**Analog:** `control-ts/tests/orchestrator/executePlan.test.ts`

**Imports pattern** (lines 1-23):
```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  readFileSync,
  existsSync,
  unlinkSync,
  mkdirSync,
  rmdirSync,
  mkdtempSync,
} from "fs";
import { resolve, dirname } from "path";
import { tmpdir } from "os";
import {
  executePlan,
  type PlanResult,
} from "../../src/orchestrator/executePlan.js";
import { parseEventLine, type ColonyEvent } from "../../src/schemas/event.schema.js";
import {
  readColonyState,
  writeColonyState,
  updateColonyState,
} from "../../src/memory/store.js";
import { projectRoot } from "../../src/utils/projectRoot.js";
```

**Test isolation pattern** (lines 44-68):
```typescript
describe("executePlan", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(resolve(tmpdir(), "aether-events-"));
    process.env.AETHER_EVENTS_FILE = resolve(tempDir, "events.ndjson");
    cleanupState();
    const initial = readColonyState();
    initial.goal = "Test colony";
    writeColonyState(initial);
  });

  afterEach(() => {
    if (tempDir && existsSync(tempDir)) {
      const files = require("fs").readdirSync(tempDir);
      for (const f of files) {
        unlinkSync(resolve(tempDir, f));
      }
      rmdirSync(tempDir);
    }
    delete process.env.AETHER_EVENTS_FILE;
    cleanupState();
    vi.restoreAllMocks();
  });
```

**Assertion pattern** (lines 70-80):
```typescript
it("runs a full sequence of phases in order and returns completed status", async () => {
  const result = await executePlan(["init", "plan", "build"]);
  expect(result.status).toBe("completed");
  expect(result.results.length).toBe(3);
  expect(result.results[0].phaseId).toBe("init");
});
```

**NDJSON verification pattern** (lines 82-96):
```typescript
it("emits plan:start and plan:complete events to NDJSON", async () => {
  const result = await executePlan(["init", "plan"]);
  const events = readEvents();
  const startEvent = events.find((e) => e.type === "plan:start");
  const completeEvent = events.find((e) => e.type === "plan:complete");

  expect(startEvent).toBeDefined();
  expect(startEvent!.payload.sequence).toEqual(["init", "plan"]);
  expect(completeEvent).toBeDefined();
  expect(completeEvent!.payload.status).toBe("completed");
});
```

---

### `scripts/audit-hardcoded.sh` (utility, batch)

**Analog:** `scripts/smoke-test-classic.sh`

**Script structure pattern** (lines 1-38):
```bash
#!/usr/bin/env bash
set -euo pipefail

PASS_COUNT=0

pass() {
    echo "PASS: $1"
    PASS_COUNT=$((PASS_COUNT + 1))
}

fail() {
    echo "FAIL: $1"
    echo "  $2"
    exit 1
}
```

**Grep audit pattern** (from RESEARCH.md Pattern 3):
```bash
echo "=== Hardcoded prompt text audit ==="
grep -rn '"You are' cmd/ --include="*.go" | grep -v '_test.go' | grep -v 'fallback\|error\|fmt.Errorf'

echo "=== Hardcoded phase ritual audit ==="
grep -rn 'ceremony\|ritual\|playbook' cmd/ --include="*.go" -l | grep -v '_test.go'

echo "=== Agent behaviour only in Go ==="
grep -rn 'builder\|watcher\|scout\|queen' cmd/ --include="*.go" -l | wc -l
```

**Exit summary pattern** (from smoke test line 234):
```bash
echo "=== All audit checks passed (${PASS_COUNT}/N) ==="
```

---

### `cmd/oracle_loop.go` (read-only audit target)

**Analog:** `cmd/oracle_loop.go` (self — audit only, no modifications)

**Hardcoded directive pattern** (lines 1544-1557):
```go
func buildOraclePhaseDirective(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "survey":
		return "Your task is to survey the landscape. Identify key concepts, existing solutions, and open questions. Do not form conclusions yet."
	case "verify":
		return "Your task is to verify previous findings. Test assumptions, look for contradictions, and assess confidence levels."
	case "investigate":
		return "Your task is to investigate specific questions. Deep-dive into the most promising areas identified in the survey."
	case "synthesize":
		return "Your task is to synthesize all findings into a coherent report. Connect dots, resolve contradictions, and formulate recommendations."
	default:
		return "Investigate pass: deepen the lowest-confidence unresolved question with new source-backed findings."
	}
}
```

**Note for planner:** These 5 strings in `cmd/oracle_loop.go` are the known remaining hardcoded agent directives. The audit script must flag them. All other `"You are"` occurrences are in test files (`cmd/codex_colonize_test.go`) or user-facing recipes (`cmd/recipes.go` line 56), which are allowed per the boundary contract.

---

### `control-ts/src/orchestrator/runPhase.ts` (read-only pattern source)

**Analog:** `control-ts/src/orchestrator/runPhase.ts` (self)

**Event emission pattern** (lines 28-33):
```typescript
function emitEvent(event: ColonyEvent): ColonyEvent {
  const validated = EventSchema.parse(event);
  const line = JSON.stringify(validated) + "\n";
  appendFileSync(getEventsFile(), line, "utf8");
  return validated;
}
```

**Phase execution pattern** (lines 40-111):
```typescript
export async function runPhase(
  phaseId: string,
  inputs?: Record<string, unknown>
): Promise<PhaseResult> {
  const phase = loadPhaseById(phaseId);
  if (!phase) {
    throw new Error(`Phase not found: ${phaseId}`);
  }

  const agent = loadAgentById(phase.entry_agent);
  if (!agent) {
    throw new Error(`Entry agent not found: ${phase.entry_agent}`);
  }

  ensureEventsDir();
  const events: ColonyEvent[] = [];

  try {
    const startEvent: ColonyEvent = {
      type: "phase:start",
      timestamp: new Date().toISOString(),
      payload: { phaseId, agentId: agent.id, inputs: inputs ?? {} },
    };
    events.push(emitEvent(startEvent));

    // Simulate execution
    await new Promise<void>((resolve) => setTimeout(resolve, 0));

    const completeEvent: ColonyEvent = {
      type: "phase:complete",
      timestamp: new Date().toISOString(),
      payload: { phaseId, agentId: agent.id, status: "completed" },
    };
    events.push(emitEvent(completeEvent));

    return { phaseId, agentId: agent.id, status: "completed", events };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    const failedEvent: ColonyEvent = {
      type: "phase:failed",
      timestamp: new Date().toISOString(),
      payload: { phaseId, agentId: agent.id, error: message },
    };
    events.push(emitEvent(failedEvent));
    return { phaseId, agentId: agent.id, status: "failed", events, error: message };
  }
}
```

---

### `control-ts/src/orchestrator/executePlan.ts` (read-only pattern source)

**Analog:** `control-ts/src/orchestrator/executePlan.ts` (self)

**PlanResult interface** (lines 17-23):
```typescript
export interface PlanResult {
  status: "completed" | "failed" | "partial";
  results: PhaseResult[];
  events: ColonyEvent[];
  startedAt: string;
  completedAt: string;
}
```

**Failure policy handling** (lines 98-123):
```typescript
if (phaseResult.status === "failed") {
  const policy = phase.failure_policy;

  if (policy === "retry") {
    const maxRetries = options?.maxRetries ?? 1;
    let retried = false;
    for (let r = 0; r < maxRetries; r++) {
      phaseResult = await runPhase(phaseId, inputs);
      results.push(phaseResult);
      events.push(...phaseResult.events);
      if (phaseResult.status !== "failed") {
        retried = true;
        break;
      }
    }
    if (!retried) {
      break;
    }
  } else if (policy === "skip") {
    continue;
  } else {
    // block or escalate
    break;
  }
}
```

**State update pattern** (lines 125-127, 151-155):
```typescript
if (phaseResult.status === "completed") {
  updateColonyState((s) => ({ ...s, current_phase: i + 1 }));
}

if (status === "completed") {
  updateColonyState((s) => ({ ...s, state: "COMPLETED" }));
} else if (status === "failed") {
  updateColonyState((s) => ({ ...s, state: "FAILED" }));
}
```

---

## Shared Patterns

### Event Stream Path Resolution
**Source:** `control-ts/src/orchestrator/runPhase.ts` (lines 13-18)
**Apply to:** `cli.ts`, `runPhase.ts`, `executePlan.ts`, all test files
```typescript
const EVENTS_DIR = resolve(projectRoot, "..", ".aether", "events");
const DEFAULT_EVENTS_FILE = resolve(EVENTS_DIR, "current.ndjson");

function getEventsFile(): string {
  return process.env.AETHER_EVENTS_FILE || DEFAULT_EVENTS_FILE;
}
```

### Test Isolation (Temp Directory + Env Override)
**Source:** `control-ts/tests/orchestrator/runPhase.test.ts` (lines 21-38)
**Apply to:** All new test files
```typescript
let tempDir: string;

beforeEach(() => {
  tempDir = mkdtempSync(resolve(tmpdir(), "aether-events-"));
  process.env.AETHER_EVENTS_FILE = resolve(tempDir, "events.ndjson");
});

afterEach(() => {
  cleanupEvents();
  if (tempDir && existsSync(tempDir)) {
    const files = require("fs").readdirSync(tempDir);
    for (const f of files) {
      unlinkSync(resolve(tempDir, f));
    }
    rmdirSync(tempDir);
  }
  delete process.env.AETHER_EVENTS_FILE;
});
```

### State Cleanup
**Source:** `control-ts/tests/orchestrator/executePlan.test.ts` (lines 27-31, 57-67)
**Apply to:** All integration tests that touch colony state
```typescript
const TEST_STATE_PATH = resolve(projectRoot, "..", ".aether", "data", "COLONY_STATE.json");

function cleanupState(): void {
  if (existsSync(TEST_STATE_PATH)) {
    unlinkSync(TEST_STATE_PATH);
  }
}

// In beforeEach: cleanupState() + seed fresh state
// In afterEach: cleanupState()
```

### NDJSON Event Validation
**Source:** `control-ts/src/schemas/event.schema.ts` (lines 11-26)
**Apply to:** Any file reading NDJSON lines
```typescript
export function parseEventLine(line: string): ColonyEvent {
  const trimmed = line.trim();
  if (!trimmed) {
    throw new Error("Empty event line");
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (err) {
    const context = trimmed.length > 80 ? trimmed.slice(0, 80) + "..." : trimmed;
    throw new Error(
      `Invalid JSON in event line: ${err instanceof Error ? err.message : String(err)} (line: ${context})`
    );
  }
  return EventSchema.parse(parsed);
}
```

### Adapter Stub Pattern
**Source:** `control-ts/src/adapters/claude.ts` (lines 10-46)
**Apply to:** Demo CLI if it needs adapter references
```typescript
export class ClaudeAdapter implements PlatformAdapter {
  readonly platform: Platform = "claude";

  async dispatch(params: DispatchParams): Promise<DispatchResult> {
    const { config } = params;
    const result = {
      workerName: config.workerName,
      caste: config.caste,
      taskID: config.taskID,
      status: "completed" as const,
      summary: "Claude Code dispatch completed (stub)",
      filesCreated: [],
      filesModified: [],
      testsWritten: [],
      artifacts: {},
      toolCount: 0,
      blockers: [],
      spawns: [],
      duration: 0,
      rawOutput: "",
    };
    return { result, platform: this.platform };
  }

  async preflight(): Promise<AvailabilityStatus> {
    return {
      platform: this.platform,
      available: true,
      category: "available",
      reason: "stub adapter",
    };
  }

  async health(): Promise<HealthStatus> {
    return { status: "healthy" };
  }
}
```

---

## No Analog Found

No files in this phase lack a close analog. All new files have strong pattern matches in the existing codebase.

---

## Metadata

**Analog search scope:**
- `control-ts/src/**/*.ts`
- `control-ts/tests/**/*.test.ts`
- `scripts/*.sh`
- `cmd/*.go` (grep-only for audit patterns)

**Files scanned:** 30+
**Pattern extraction date:** 2026-05-24
