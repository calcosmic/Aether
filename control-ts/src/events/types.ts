/**
 * Event domain types for the Aether control plane event stream.
 *
 * The event stream is the shared observable truth between Go and TypeScript.
 * Both sides must agree on the NDJSON line format:
 *   { "type": "...", "timestamp": "...", "payload": { ... } }
 */

export type EventType =
  | "phase_start"
  | "agent_selected"
  | "task_planned"
  | "worker_spawned"
  | "tool_call"
  | "verification_result"
  | "memory_update"
  | "run_seal"
  | "custom";

export interface PhaseStartPayload {
  phase: string;
  plan: string;
  goal?: string;
}

export interface AgentSelectedPayload {
  agentId: string;
  caste: string;
  task: string;
}

export interface TaskPlannedPayload {
  taskId: string;
  description: string;
  dependencies: string[];
}

export interface WorkerSpawnedPayload {
  workerId: string;
  caste: string;
  parentAgentId?: string;
}

export interface ToolCallPayload {
  tool: string;
  args: Record<string, unknown>;
  result?: unknown;
}

export interface VerificationResultPayload {
  taskId: string;
  passed: boolean;
  message?: string;
}

export interface MemoryUpdatePayload {
  key: string;
  value: unknown;
  source: string;
}

export interface RunSealPayload {
  phase: string;
  plan: string;
  summary: string;
}

export interface CustomPayload {
  [key: string]: unknown;
}

export type EventPayload =
  | { type: "phase_start"; payload: PhaseStartPayload }
  | { type: "agent_selected"; payload: AgentSelectedPayload }
  | { type: "task_planned"; payload: TaskPlannedPayload }
  | { type: "worker_spawned"; payload: WorkerSpawnedPayload }
  | { type: "tool_call"; payload: ToolCallPayload }
  | { type: "verification_result"; payload: VerificationResultPayload }
  | { type: "memory_update"; payload: MemoryUpdatePayload }
  | { type: "run_seal"; payload: RunSealPayload }
  | { type: "custom"; payload: CustomPayload };

export interface EventLine {
  type: EventType;
  timestamp: string;
  payload: EventPayload["payload"];
}

/**
 * Create a structured event line with the current ISO timestamp.
 */
export function createEventLine<T extends EventType>(
  type: T,
  payload: Extract<EventPayload, { type: T }>["payload"]
): EventLine {
  return {
    type,
    timestamp: new Date().toISOString(),
    payload,
  };
}

export type EventWriter = {
  write: (event: EventLine) => Promise<void>;
  close: () => Promise<void>;
};

export type EventReader = {
  read: () => AsyncIterable<EventLine>;
  tail: (options?: { pollIntervalMs?: number }) => AsyncIterable<EventLine>;
};
