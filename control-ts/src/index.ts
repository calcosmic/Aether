/**
 * Public API entry point for @aether/control.
 * Re-exports schemas, types, and loaders.
 */
export const AETHER_CONTROL_VERSION = "0.1.0";

// Schemas
export { AgentSchema } from "./schemas/agent.schema.js";
export { PhaseSchema } from "./schemas/phase.schema.js";
export { EventSchema } from "./schemas/event.schema.js";
export { PolicySchema } from "./schemas/policy.schema.js";

// Types
export type { Agent, Phase, ColonyEvent, Policy, Skill, ColonyState, PhaseResult, PlanResult } from "./types/index.js";

// Loaders
export { loadAgents, loadAgentById } from "./agents/loadAgents.js";
export { loadPhases, loadPhaseById } from "./phases/loadPhases.js";
export { assemblePrompt, assemblePromptForAgent } from "./prompts/assemblePrompt.js";
export { loadSkills, matchSkillsForWorker } from "./skills/loadSkills.js";

// Runtime
export { readColonyState, writeColonyState, updateColonyState } from "./memory/store.js";
export { runPhase } from "./orchestrator/runPhase.js";
export { executePlan } from "./orchestrator/executePlan.js";

// Adapters
export { createAdapter, getAvailableAdapters } from "./adapters/index.js";
export type {
  Platform,
  AvailabilityStatus,
  WorkerConfig,
  WorkerResult,
  DispatchParams,
  DispatchResult,
  HealthStatus,
  PlatformAdapter,
} from "./adapters/types.js";

// Utilities
export { projectRoot } from "./utils/projectRoot.js";
