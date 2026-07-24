import { appendFileSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { loadPhaseById } from "../phases/loadPhases.js";
import { runPhase } from "./runPhase.js";
import { readColonyState, updateColonyState } from "../memory/store.js";
import { EventSchema, type ColonyEvent } from "../schemas/event.schema.js";
import { projectRoot } from "../utils/projectRoot.js";
import type { PhaseResult } from "../types/runtime.js";

const EVENTS_DIR = resolve(projectRoot, ".retired-state", "events");
const DEFAULT_EVENTS_FILE = resolve(EVENTS_DIR, "current.ndjson");

function getEventsFile(): string {
  return process.env.AETHER_EVENTS_FILE || DEFAULT_EVENTS_FILE;
}

export interface PlanResult {
  status: "completed" | "failed" | "partial";
  results: PhaseResult[];
  events: ColonyEvent[];
  startedAt: string;
  completedAt: string;
}

function ensureEventsDir(): void {
  const eventsFile = getEventsFile();
  const dir = dirname(eventsFile);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
}

function emitEvent(event: ColonyEvent): ColonyEvent {
  const validated = EventSchema.parse(event);
  const line = JSON.stringify(validated) + "\n";
  appendFileSync(getEventsFile(), line, "utf8");
  return validated;
}

/**
 * Execute a sequence of phases in order.
 *
 * @param sequence Array of phase IDs to run.
 * @param options Optional per-phase inputs and max retries.
 * @returns PlanResult with overall status and per-phase results.
 */
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
    payload: {
      sequence,
      startedAt,
    },
  };
  events.push(emitEvent(planStartEvent));

  // Initialize colony state
  const state = readColonyState();
  if (state.state !== "RUNNING") {
    updateColonyState((s) => ({ ...s, state: "RUNNING" }));
  }

  const results: PhaseResult[] = [];

  for (let i = 0; i < sequence.length; i++) {
    const phaseId = sequence[i];
    const phase = loadPhaseById(phaseId);

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

    const inputs = options?.inputs?.[phaseId] ?? {};
    let phaseResult = await runPhase(phaseId, inputs);
    results.push(phaseResult);
    events.push(...phaseResult.events);

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
          // Retry exhausted — treat as block
          break;
        }
      } else if (policy === "skip") {
        continue;
      } else {
        // block or escalate
        break;
      }
    }

    if (phaseResult.status === "completed") {
      updateColonyState((s) => ({ ...s, current_phase: i + 1 }));
    }
  }

  // Determine overall plan status
  const allCompleted = results.length > 0 && results.every((r) => r.status === "completed");
  const anyFailed = results.some((r) => r.status === "failed");
  const someCompleted = results.some((r) => r.status === "completed");

  let status: PlanResult["status"];
  if (allCompleted || results.length === 0) {
    status = "completed";
  } else if (anyFailed && someCompleted) {
    status = "partial";
  } else {
    status = "failed";
  }

  // Override: if the last result is a failed block/escalate/retry-exhausted, mark as failed
  const lastResult = results[results.length - 1];
  if (lastResult && lastResult.status === "failed") {
    status = "failed";
  }

  // Update colony state based on final status
  if (status === "completed") {
    updateColonyState((s) => ({ ...s, state: "COMPLETED" }));
  } else if (status === "failed") {
    updateColonyState((s) => ({ ...s, state: "FAILED" }));
  }

  const completedAt = new Date().toISOString();
  const planCompleteEvent: ColonyEvent = {
    type: "plan:complete",
    timestamp: completedAt,
    payload: {
      status,
      completedAt,
      phaseCount: results.length,
    },
  };
  events.push(emitEvent(planCompleteEvent));

  return {
    status,
    results,
    events,
    startedAt,
    completedAt,
  };
}
