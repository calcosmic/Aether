/**
 * Barrel export for the Aether control plane event stream.
 */

export {
  writeEvent,
  createEventStream,
  getBuffer,
  clearBuffer,
} from "./writeEvent.js";

export { readEvents, tailEvents } from "./readEvents.js";

export type {
  EventType,
  EventLine,
  EventPayload,
  PhaseStartPayload,
  AgentSelectedPayload,
  TaskPlannedPayload,
  WorkerSpawnedPayload,
  ToolCallPayload,
  VerificationResultPayload,
  MemoryUpdatePayload,
  RunSealPayload,
  CustomPayload,
  EventWriter,
  EventReader,
} from "./types.js";

export { createEventLine } from "./types.js";
