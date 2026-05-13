/**
 * TypeScript type definitions for Go manifest, completion, and worker result
 * JSON schemas.
 *
 * These interfaces match the Go struct definitions in:
 * - cmd/codex_build.go (codexBuildManifest, codexBuildDispatch, codexBuildTaskPlan)
 * - cmd/codex_build_finalize.go (codexExternalBuildCompletion, codexExternalBuildWorkerResult)
 * - cmd/codex_plan_finalize.go (codexExternalPlanCompletion)
 * - cmd/codex_continue_finalize.go (codexExternalContinueCompletion)
 *
 * All optional Go fields (omitempty) are marked optional in TypeScript (use `?`).
 */

// ---------------------------------------------------------------------------
// Go output envelope
// ---------------------------------------------------------------------------

/**
 * Go CLI JSON output envelope. All commands in AETHER_OUTPUT_MODE=json produce
 * either {"ok":true,"result":<T>} on success or {"ok":false,"error":"msg","code":N}
 * on failure. See cmd/helpers.go outputOK / outputError.
 */
export interface GoOutput<T> {
  ok: boolean;
  result?: T;
  error?: string;
  code?: number;
}

// ---------------------------------------------------------------------------
// Build manifest types (cmd/codex_build.go)
// ---------------------------------------------------------------------------

export interface BuildManifest {
  phase: number;
  phase_name: string;
  goal?: string;
  root: string;
  colony_mode?: string;
  plan_only?: boolean;
  parallel_mode?: string;
  wave_execution?: WaveExecutionPlan[];
  execution_plan?: BuildExecutionPlan[];
  colony_depth: string;
  dispatch_mode?: string;
  host_platform?: string;
  execution_owner?: string;
  worker_dispatch_opt_in?: boolean;
  generated_at: string;
  state: string;
  checkpoint: string;
  claims_path: string;
  playbooks: string[];
  worker_briefs: string[];
  dispatches: BuildDispatch[];
  selected_tasks?: string[];
  tasks: BuildTaskPlan[];
  success_criteria: string[];
  review_depth?: string;
  dispatch_contract?: Record<string, unknown>;
  profile_contract?: WorkflowProfileContract;
  queen_recommendation?: QueenWorkflowRecommendation;
  queen_execution_policy?: QueenExecutionPolicy;
  boundary_questions?: DiscussQuestion[];
  boundary_question_count?: number;
  boundary_questions_created?: number;
  boundary_questions_existing?: number;
  orchestrator_boundary_guidance?: OrchestratorBoundaryGuidance;
}

export interface WaveExecutionPlan {
  wave: number;
  strategy: string;
  worker_count: number;
  reason: string;
}

export interface BuildExecutionPlan {
  execution_wave: number;
  stage: string;
  // Additional fields from Go struct
  [key: string]: unknown;
}

export interface BuildDispatch {
  stage: string;
  wave?: number;
  execution_wave?: number;
  caste: string;
  name: string;
  task: string;
  status: string;
  summary?: string;
  task_id?: string;
  task_index?: number;
  depends_on?: string[];
  outputs?: string[];
  blockers?: string[];
  duration?: number;
  skill_section?: string;
  skill_count?: number;
  colony_skill_count?: number;
  domain_skill_count?: number;
  matched_skills?: string[];
  handoff_section?: string;
}

export interface BuildTaskPlan {
  id?: string;
  goal: string;
  status: string;
  wave?: number;
  depends_on?: string[];
}

// ---------------------------------------------------------------------------
// Build completion types (cmd/codex_build_finalize.go)
// ---------------------------------------------------------------------------

export interface BuildCompletion {
  dispatch_manifest?: BuildManifest;
  manifest?: BuildManifest;
  dispatches?: WorkerResult[];
  results?: WorkerResult[];
  workers?: WorkerResult[];
  claims?: BuildClaims;
}

export interface WorkerResult {
  stage?: string;
  wave?: number;
  execution_wave?: number;
  caste?: string;
  name: string;
  ant_name?: string;
  task?: string;
  status: string;
  summary?: string;
  task_id?: string;
  task_index?: number;
  depends_on?: string[];
  outputs?: string[];
  blockers?: string[];
  duration?: number;
  tool_count?: number;
  files_created?: string[];
  files_modified?: string[];
  tests_written?: string[];
  handoff?: WorkerHandoff;
}

export interface WorkerHandoff {
  changed_files?: string[];
  commands_run?: string[];
  verification_status?: string;
  known_failures?: string[];
  open_decisions?: string[];
  assumptions?: string[];
  next_worker_instructions?: string[];
  things_not_to_repeat?: string[];
  freshness?: string;
  [key: string]: unknown;
}

export interface BuildClaims {
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Plan completion types (cmd/codex_plan_finalize.go)
// ---------------------------------------------------------------------------

export interface PlanCompletion {
  plan_manifest?: PlanManifest;
  planning_manifest?: PlanManifest;
  manifest?: PlanManifest;
  dispatches?: PlanningDispatch[];
  results?: PlanningDispatch[];
  workers?: PlanningDispatch[];
  scout_report?: ScoutReport;
  phase_plan?: WorkerPlanArtifact;
}

export interface PlanManifest {
  [key: string]: unknown;
}

export interface PlanningDispatch {
  name?: string;
  status?: string;
  summary?: string;
  [key: string]: unknown;
}

export interface ScoutReport {
  [key: string]: unknown;
}

export interface WorkerPlanArtifact {
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Continue completion types (cmd/codex_continue_finalize.go)
// ---------------------------------------------------------------------------

export interface ContinueCompletion {
  continue_manifest?: ContinuePlanManifest;
  manifest?: ContinuePlanManifest;
  dispatches?: ContinueExternalDispatch[];
  results?: ContinueExternalDispatch[];
  workers?: ContinueExternalDispatch[];
}

export interface ContinuePlanManifest {
  [key: string]: unknown;
}

export interface ContinueExternalDispatch {
  name?: string;
  status?: string;
  summary?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Shared helper types
// ---------------------------------------------------------------------------

export interface WorkflowProfileContract {
  [key: string]: unknown;
}

export interface QueenWorkflowRecommendation {
  [key: string]: unknown;
}

export interface QueenExecutionPolicy {
  [key: string]: unknown;
}

export interface DiscussQuestion {
  [key: string]: unknown;
}

export interface OrchestratorBoundaryGuidance {
  [key: string]: unknown;
}

/**
 * Terminal worker status values accepted by Go finalizers.
 * Matches isTerminalExternalBuildStatus in cmd/codex_build_finalize.go.
 */
export type TerminalWorkerStatus =
  | "completed"
  | "failed"
  | "blocked"
  | "timeout"
  | "manually-reconciled";

// ---------------------------------------------------------------------------
// Ceremony event types (pkg/events/ceremony.go)
// ---------------------------------------------------------------------------

export interface CeremonyPayload {
  phase?: number;
  phase_name?: string;
  wave?: number;
  spawn_id?: string;
  caste?: string;
  name?: string;
  task_id?: string;
  task?: string;
  status?: string;
  message?: string;
  skill?: string;
  pheromone_type?: string;
  strength?: number;
  completed?: number;
  total?: number;
  tool_count?: number;
  token_count?: number;
  files_created?: string[];
  files_modified?: string[];
  tests_written?: string[];
  blockers?: string[];
  success_criteria?: string[];
  loop_type?: string;
  detection_signal?: string;
  action_taken?: string;
}

export interface CeremonyEvent {
  id: string;
  topic: string;
  payload: CeremonyPayload;
  source: string;
  timestamp: string;
  ttl_days: number;
  expires_at: string;
}

export const CEREMONY_TOPICS = [
  "ceremony.build.prewave",
  "ceremony.build.wave.start",
  "ceremony.build.spawn",
  "ceremony.build.tool_use",
  "ceremony.build.wave.end",
  "ceremony.build.circuit_break",
  "ceremony.plan.wave.start",
  "ceremony.plan.spawn",
  "ceremony.plan.wave.end",
  "ceremony.colonize.wave.start",
  "ceremony.colonize.spawn",
  "ceremony.colonize.wave.end",
  "ceremony.continue.wave.start",
  "ceremony.continue.spawn",
  "ceremony.continue.wave.end",
  "ceremony.pheromone.emit",
  "ceremony.skill.activate",
  "ceremony.chamber.seal",
  "ceremony.chamber.entomb",
  "ceremony.midden.record",
  "ceremony.queen.promote",
  "ceremony.hive.store",
  "ceremony.hive.promote",
  "ceremony.loop.break",
] as const;

export type CeremonyTopic = (typeof CEREMONY_TOPICS)[number];
