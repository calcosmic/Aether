/**
 * Shared adapter interfaces for the TypeScript control plane.
 * Mirrors Go runtime platform dispatch patterns.
 */

export type Platform = "unknown" | "codex" | "claude" | "opencode" | "fake" | "mcp";

export type AvailabilityCategory =
  | "available"
  | "binary_missing"
  | "auth_probe_failed"
  | "auth_inactive"
  | "invalid_auth_output"
  | "credentials_missing"
  | "probe_skipped"
  | "unsupported_provider"
  | "provider_config_invalid";

export interface AvailabilityStatus {
  platform: Platform;
  available: boolean;
  category: AvailabilityCategory;
  reason: string;
  binary?: string;
}

export interface WorkerConfig {
  agentName: string;
  agentTOMLPath: string;
  caste: string;
  workerName: string;
  taskID: string;
  taskBrief: string;
  contextCapsule: string;
  root: string;
  timeout: number; // milliseconds
  skillSection: string;
  pheromoneSection: string;
  handoffSection: string;
  configOverrides: string[];
  responsePath?: string;
  callbackURL?: string;
}

export interface WorkerResult {
  workerName: string;
  caste: string;
  taskID: string;
  status: "completed" | "failed" | "blocked" | "timeout";
  summary: string;
  filesCreated: string[];
  filesModified: string[];
  testsWritten: string[];
  artifacts: Record<string, unknown>;
  scoutReport?: unknown;
  toolCount: number;
  blockers: string[];
  spawns: string[];
  duration: number; // milliseconds
  rawOutput: string;
  error?: string;
  handoff?: unknown;
}

export interface DispatchParams {
  platform: Platform;
  config: WorkerConfig;
}

export interface DispatchResult {
  result: WorkerResult;
  platform: Platform;
}

export interface HealthStatus {
  status: "healthy" | "degraded" | "unhealthy";
  message?: string;
}

export interface PlatformAdapter {
  readonly platform: Platform;
  dispatch(params: DispatchParams): Promise<DispatchResult>;
  preflight(): Promise<AvailabilityStatus>;
  health(): Promise<HealthStatus>;
}
