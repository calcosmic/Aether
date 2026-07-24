import { appendFileSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { loadPhaseById } from "../phases/loadPhases.js";
import { loadAgentById } from "../agents/loadAgents.js";
import { EventSchema, type ColonyEvent } from "../schemas/event.schema.js";
import { projectRoot } from "../utils/projectRoot.js";

import type { PhaseResult } from "../types/runtime.js";
import { RETIRED_CONTROL_PLANE_MESSAGE } from "../retired.js";

// Re-export for convenience
export type { PhaseResult };

const EVENTS_DIR = resolve(projectRoot, ".retired-state", "events");
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
 * and returns a failed PhaseResult. This retired prototype must never report
 * successful execution because it has no production provider adapter.
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

  const failedEvent: ColonyEvent = {
    type: "phase:failed",
    timestamp: new Date().toISOString(),
    payload: {
      phaseId,
      agentId: agent.id,
      error: RETIRED_CONTROL_PLANE_MESSAGE,
    },
  };
  events.push(emitEvent(failedEvent));

  return {
    phaseId,
    agentId: agent.id,
    status: "failed",
    events,
    error: RETIRED_CONTROL_PLANE_MESSAGE,
  };
}
