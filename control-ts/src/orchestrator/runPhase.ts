import { appendFileSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { loadPhaseById } from "../phases/loadPhases.js";
import { loadAgentById } from "../agents/loadAgents.js";
import { EventSchema, type ColonyEvent } from "../schemas/event.schema.js";
import { projectRoot } from "../utils/projectRoot.js";

import type { PhaseResult } from "../types/runtime.js";

// Re-export for convenience
export type { PhaseResult };

const EVENTS_DIR = resolve(projectRoot, "..", ".aether", "events");
const DEFAULT_EVENTS_FILE = resolve(EVENTS_DIR, "current.ndjson");

function getEventsFile(): string {
  return process.env.AETHER_EVENTS_FILE || DEFAULT_EVENTS_FILE;
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
 * Run a phase by ID.
 * Loads the phase definition, resolves the entry agent, emits NDJSON events,
 * and returns a PhaseResult. Simulation only — no actual adapter calls.
 */
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
      payload: {
        phaseId,
        agentId: agent.id,
        inputs: inputs ?? {},
      },
    };
    events.push(emitEvent(startEvent));

    // Simulate execution — no actual adapter call in Phase 156
    await new Promise<void>((resolve) => setTimeout(resolve, 0));

    const completeEvent: ColonyEvent = {
      type: "phase:complete",
      timestamp: new Date().toISOString(),
      payload: {
        phaseId,
        agentId: agent.id,
        status: "completed",
      },
    };
    events.push(emitEvent(completeEvent));

    return {
      phaseId,
      agentId: agent.id,
      status: "completed",
      events,
    };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    const failedEvent: ColonyEvent = {
      type: "phase:failed",
      timestamp: new Date().toISOString(),
      payload: {
        phaseId,
        agentId: agent.id,
        error: message,
      },
    };
    events.push(emitEvent(failedEvent));

    return {
      phaseId,
      agentId: agent.id,
      status: "failed",
      events,
      error: message,
    };
  }
}
