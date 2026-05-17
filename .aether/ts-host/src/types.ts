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
 * The TS host also models explicit host-owned synthesis envelopes for planning
 * completions so synthesized plans are not represented as worker evidence.
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
  provider_diagnostics?: string;
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
  synthesis?: PlanSynthesis;
}

export interface PlanManifest {
  [key: string]: unknown;
}

export interface PlanningDispatch {
  name?: string;
  status?: string;
  summary?: string;
  phase_plan?: WorkerPlanArtifact;
  [key: string]: unknown;
}

export interface PlanSynthesis {
  source: "ts-host";
  reason: string;
  phase_plan: WorkerPlanArtifact;
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

/**
 * Go-authored Queen spawn budget metadata from queen_execution_policy.
 * The TS host may preserve this object, but Go remains the source of truth.
 */
export interface QueenSpawnBudget {
  max_workers?: number;
  selected_workers?: number;
  worker_count?: number;
  max_selected_castes?: number;
  selected_castes?: number;
  pruned_workers?: number;
  pruned_castes?: number;
  preserved_castes?: string[];
  required_castes?: string[];
  policy_added_castes?: string[];
  overflow_required_workers?: number;
  relevance_threshold?: number;
  budget_unit?: string;
  reason?: string;
  flow_type?: string;
  risk_level?: string;
  castes?: string[];
  counts?: Record<string, number>;
}

export interface QueenExecutionPolicy {
  verification_depth?: string;
  review_depth?: string;
  spawn_budget?: QueenSpawnBudget;
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
  | "manually-reconciled"
  | "code_written";

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
  "ceremony.oracle.phase_transition",
  "ceremony.oracle.iteration",
] as const;

export type CeremonyTopic = (typeof CEREMONY_TOPICS)[number];

// ---------------------------------------------------------------------------
// Oracle iteration types (cmd/oracle_iterate_cmd.go)
// ---------------------------------------------------------------------------

/**
 * Outer result envelope from `aether oracle-iterate --plan-only`.
 * callGoJSON returns the inner `result` field, which matches this shape.
 */
export interface OracleIterationManifest {
  /** Whether the iteration is valid. */
  ok: boolean;
  /** The iteration manifest describing this Oracle loop step. */
  iteration_manifest: OracleIterationState;
}

/**
 * State of a single Oracle iteration as returned by Go.
 * Mirrors the Go `iterationManifest` struct.
 */
export interface OracleIterationState {
  /** Research topic. */
  topic: string;
  /** Research depth: quick, balanced, deep, exhaustive. */
  depth: string;
  /** Maximum number of iterations allowed. */
  max_iterations: number;
  /** Target confidence percentage (1-100). */
  confidence_target: number;
  /** Current iteration number (1-based). */
  current_iteration: number;
  /** Workers assigned to this iteration. */
  workers: OracleWorker[];
  /** When true, this iteration is resuming from a prior interrupt. */
  resuming?: boolean;
}

/**
 * A single Oracle worker dispatch.
 * Mirrors the Go `oracleWorker` struct.
 */
export interface OracleWorker {
  /** Worker name (e.g., Oracle-01). */
  name: string;
  /** Worker caste (always "oracle"). */
  caste: string;
  /** Task description. */
  task: string;
  /** Detailed brief for the worker. */
  brief: string;
}

/**
 * An Oracle research question.
 * For future use if the Go command adds question fields.
 */
export interface OracleQuestion {
  /** Question identifier. */
  id: string;
  /** Question text. */
  text: string;
  /** Status: open, answered, shelved. */
  status: string;
  /** Confidence in the answer (0-100). */
  confidence?: number;
}

/**
 * Stop conditions for the Oracle RALF loop.
 * Computed by the TS host from manifest fields, not returned by Go.
 */
export interface OracleStopConditions {
  /** True when current confidence meets or exceeds the target. */
  confidence_met: boolean;
  /** True when current iteration reaches max_iterations. */
  max_iterations_met: boolean;
  /** True when the user manually requested a stop. */
  manual_stop: boolean;
  /** True when no progress was made in the last iteration. */
  no_progress: boolean;
}

/**
 * Question count summary.
 * Computed by the TS host.
 */
export interface OracleQuestionCounts {
  /** Total questions. */
  total: number;
  /** Questions with status "answered". */
  answered: number;
  /** Questions touched in the current iteration. */
  touched: number;
}

/**
 * Paths to Oracle workspace files.
 * For reference only — the TS host never writes to these directly.
 */
export interface OracleWorkspacePaths {
  /** Path to the persisted Oracle state file. */
  state_path: string;
  /** Path to the research plan. */
  plan_path: string;
  /** Path to the gaps analysis. */
  gaps_path: string;
  /** Path to the synthesis document. */
  synthesis_path: string;
  /** Path to the research plan document. */
  research_plan_path: string;
}

/**
 * Result from `aether oracle-iterate-finalize --completion-file`.
 * Mirrors the Go `oracleFinalizeResult` struct.
 */
export interface OracleIterationCompletion {
  /** Whether finalization succeeded. */
  ok: boolean;
  /** Path to the persisted state file. */
  state_path: string;
  /** Current confidence after this iteration. */
  current_confidence: number;
  /** Target confidence percentage. */
  confidence_target: number;
  /** Whether another iteration should run. */
  should_continue: boolean;
  /** Suggested next command for the TS host. */
  next_command: string;
}

/**
 * Response from a single Oracle worker after dispatch.
 * Built by the TS host from the DispatchResult.
 */
export interface OracleWorkerResponse {
  /** Question identifier this response addresses. */
  question_id: string;
  /** Worker status: completed, failed, blocked. */
  status: string;
  /** Runtime-owned confidence in the findings (0-100), when provided by Go. */
  confidence?: number;
  /** Summary of the worker's research output. */
  summary: string;
  /** Structured findings from the worker. */
  findings?: OracleWorkerFinding[];
  /** Identified knowledge gaps. */
  gaps?: string[];
  /** Contradictions found during research. */
  contradictions?: string[];
  /** Primary recommendation from the worker. */
  recommendation?: string;
}

/**
 * A single finding reported by an Oracle worker.
 */
export interface OracleWorkerFinding {
  /** Finding text. */
  text: string;
  /** Supporting evidence for the finding. */
  evidence?: OracleWorkerEvidence[];
}

/**
 * Evidence supporting an Oracle worker finding.
 */
export interface OracleWorkerEvidence {
  /** Evidence title or description. */
  title: string;
  /** Where the evidence was found (file, URL, etc.). */
  location: string;
  /** Evidence type: code, doc, test, external. */
  type: string;
}

// ---------------------------------------------------------------------------
// Status types (cmd/status.go)
// ---------------------------------------------------------------------------

/**
 * Summary of a single active pheromone signal.
 * Mirrors the visual-mode pheromone row in renderPheromoneSummary.
 */
export interface PheromoneSummary {
  /** Signal type: FOCUS, REDIRECT, or FEEDBACK. */
  type: string;
  /** Signal content text. */
  content: string;
  /** Decay-adjusted signal strength. */
  strength: number;
}

/**
 * Colony memory health metrics.
 * Mirrors the summary rendered in renderMemoryHealthTable.
 */
export interface MemoryHealth {
  /** Number of event bus entries. */
  events: number;
  /** Number of recorded learnings. */
  learnings: number;
  /** Number of applied instincts. */
  instincts: number;
  /** Number of active pheromone signals. */
  pheromones: number;
}

/**
 * Result from `aether status` in JSON mode.
 * Mirrors the Go buildStatusResult map[string]interface{}.
 *
 * Required fields are always present. Optional fields map to Go omitempty.
 */
export interface StatusResult {
  /** Colony goal string. */
  goal: string;
  /** Colony state label (e.g., "planning", "executing", "completed"). */
  state: string;
  /** Current phase number (0 if not started). */
  current_phase: number;
  /** Total number of phases in the plan. */
  total_phases: number;
  /** Number of phases completed so far. */
  phases_completed: number;
  /** Name of the current phase. */
  phase_name: string;
  /** Number of tasks completed in the current phase. */
  tasks_completed: number;
  /** Total number of tasks in the current phase. */
  tasks_total: number;
  /** Colony mode (e.g., "standard", "deep"). */
  colony_mode?: string;
  /** Whether this is an agent-delegate session. */
  agent_delegate_session?: boolean;
  /** Display phase number (may differ from current_phase during recovery). */
  display_phase?: number;
  /** Actionable warnings computed from colony state. */
  warnings?: string[];
}

// ---------------------------------------------------------------------------
// Swarm types (cmd/swarm_cmd.go)
// ---------------------------------------------------------------------------

/**
 * A single worker plan inside a swarm manifest.
 * Mirrors the Go swarmWorkerPlan struct.
 */
export interface SwarmWorkerPlan {
  /** Stage name (e.g., "investigation", "fix", "verification"). */
  stage?: string;
  /** Wave number (1-based). */
  wave: number;
  /** Worker name (deterministic). */
  name: string;
  /** Worker caste. */
  caste: string;
  /** Worker role (usually same as caste). */
  role: string;
  /** Task description. */
  task: string;
  /** Optional task identifier. */
  task_id?: string;
  /** Agent name for dispatch. */
  agent_name: string;
  /** Detailed brief for the worker. */
  brief?: string;
  /** Expected output file paths. */
  output_paths?: string[];
  /** Response contract describing expected worker output shape. */
  response_contract?: Record<string, unknown>;
  /** Per-worker timeout in seconds. */
  timeout_seconds?: number;
}

/**
 * Swarm dispatch manifest from `aether swarm --plan-only`.
 * Mirrors the Go swarmManifest struct.
 */
export interface SwarmManifest {
  /** Workflow name (always "swarm"). */
  workflow: string;
  /** Dispatch mode (e.g., "plan-only", "agent-delegate"). */
  dispatch_mode: string;
  /** Whether a finalizer command is required after worker dispatch. */
  requires_finalizer: boolean;
  /** RFC3339 timestamp when the manifest was generated. */
  generated_at: string;
  /** Absolute path to the workspace root. */
  root: string;
  /** Unique swarm identifier. */
  swarm_id: string;
  /** Target problem description. */
  target: string;
  /** Number of waves in the swarm. */
  wave_count: number;
  /** Number of workers in the swarm. */
  worker_count: number;
  /** Total run timeout in seconds. */
  run_timeout_seconds: number;
  /** Per-worker timeout in seconds. */
  worker_timeout_seconds: number;
  /** Dispatch contract describing execution model and constraints. */
  dispatch_contract: Record<string, unknown>;
  /** Array of worker dispatches. */
  dispatches: SwarmWorkerPlan[];
  /** Execution plan as generic maps (mirrors Go []map[string]interface{}). */
  execution_plan: Record<string, unknown>[];
  /** Finalizer command to run after workers complete. */
  finalizer_command: string;
}
