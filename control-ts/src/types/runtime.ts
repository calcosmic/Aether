export interface ColonyState {
  version: string;
  goal: string;
  scope: string;
  colony_mode: string;
  state: string;
  current_phase: number;
  session_id: string;
  initialized_at: string;
  plan: {
    generated_at: string;
    confidence: number;
    phases: unknown[];
  };
}

export interface PhaseResult {
  phaseId: string;
  agentId: string;
  status: "completed" | "failed" | "skipped";
  events: import("../schemas/event.schema.js").ColonyEvent[];
  error?: string;
}
