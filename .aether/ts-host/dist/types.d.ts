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
    attempt_id?: string;
    attempt_path?: string;
    execution_binding?: ExecutionBinding;
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
    dispatch_contract?: DispatchContract;
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
    wave?: number;
    strategy: string;
    worker_count: number;
    castes?: string[];
    reason?: string;
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
    /** Sub-workers requested by this dispatch via structured spawn claims. */
    spawns?: SpawnClaim[];
    skill_count?: number;
    colony_skill_count?: number;
    domain_skill_count?: number;
    matched_skills?: string[];
    handoff_section?: string;
    /** Parent worker name for spawned workers (SPAWN-05). Undefined for manifest workers. */
    parent?: string;
    /** Spawn depth for spawned workers (SPAWN-05). Undefined for manifest workers (defaults to 1). */
    depth?: number;
    /** Context capsule section injected by colony-prime. */
    context_capsule?: string;
    /** Pheromone signals section for this dispatch. */
    pheromone_section?: string;
    /** Task brief for detailed worker instructions. */
    task_brief?: string;
    /** Canonical Go-owned permission request for this caste. */
    permission_profile?: PermissionProfile;
}
export interface PermissionProfile {
    schema_version: number;
    name: "repository_read_only" | "workspace_write" | "scoped_write" | "test_write";
    filesystem: "repository_read_only" | "workspace_write" | "scoped_write" | "test_write";
    shell: string;
    network: string;
    approval: string;
    write_scopes?: string[];
    behavioral_restrictions?: string[];
}
export interface ExecutionBinding {
    schema_version: number;
    run_id: string;
    attempt_id: string;
    manifest_sha256: string;
    workspace_fingerprint: string;
    execution_owner: string;
}
export interface BuildTaskPlan {
    id?: string;
    goal: string;
    status: string;
    wave?: number;
    depends_on?: string[];
}
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
    /** Provider-returned structured artifacts retained for workflow finalizers. */
    artifacts?: Record<string, unknown>;
    /** Planning Scout evidence retained for plan-finalize. */
    scout_report?: unknown;
    /** Route-Setter plan artifact retained for plan-finalize. */
    phase_plan?: unknown;
    /** Sub-workers requested by this worker via structured spawn claims. */
    spawns?: SpawnClaim[];
    handoff?: WorkerHandoff;
}
/** Result from a child worker attached to a parent's handoff (SPAWN-04). */
export interface ChildResult {
    /** Child worker name. */
    name: string;
    /** Terminal status of the child worker. */
    status: string;
    /** Summary from the child worker. */
    summary?: string;
}
export interface WorkerHandoff {
    changed_files?: string[];
    commands_run?: string[];
    verification_status?: string;
    known_failures?: string[];
    open_decisions?: string[];
    assumptions?: string[];
    next_worker_instructions?: string[];
    do_not_repeat?: string[];
    freshness?: string;
    /** Results from spawned child workers (SPAWN-04). */
    child_results?: ChildResult[];
}
/**
 * A structured spawn claim returned by a worker.
 *
 * Workers request sub-workers by emitting an array of SpawnClaim objects
 * in their claims output. The orchestration host reads these claims and
 * decides whether to spawn additional workers.
 */
export interface SpawnClaim {
    /** The worker caste to spawn (e.g., "builder", "scout", "watcher"). */
    caste: string;
    /** Task description for the spawned worker. */
    task: string;
    /** Optional explanation for why this spawn is needed. */
    reason?: string;
}
/**
 * A worker that was spawned from a parent worker's SpawnClaim.
 *
 * Tracks the full lifecycle of a spawned worker including its relationship
 * to the parent, execution depth, and completion state.
 */
export interface SpawnedWorker {
    /** The original spawn claim that triggered this worker. */
    claim: SpawnClaim;
    /** Auto-generated worker name (e.g., "Builder-42"). */
    name: string;
    /** Parent worker name that issued the spawn claim. */
    parent: string;
    /** Spawn depth (1 = child of manifest worker, 2 = grandchild). */
    depth: number;
    /** Terminal status of the spawned worker. */
    status: string;
    /** Completion summary from the spawned worker. */
    summary?: string;
    /** Handoff relay data from the spawned worker, attached to parent's handoff. */
    handoff?: WorkerHandoff;
}
export interface BuildTaskClaim {
    task_id: string;
    files_created?: string[];
    files_modified?: string[];
    tests_written?: string[];
}
export interface BuildClaims {
    files_created: string[];
    files_modified: string[];
    tests_written?: string[];
    task_claims?: BuildTaskClaim[];
    build_phase: number;
    timestamp: string;
}
export interface PlanCompletion {
    plan_manifest?: PlanManifest;
    planning_manifest?: PlanManifest;
    manifest?: PlanManifest;
    dispatches?: PlanningDispatch[];
    results?: PlanningDispatch[];
    workers?: PlanningDispatch[];
    scout_report?: ScoutReport;
    phase_plan?: WorkerPlanArtifact;
    synthesis?: PlanSynthesis;
}
export interface PlanManifest {
    goal?: string;
    root?: string;
    generated_at?: string;
    base_revision_id?: string;
    base_plan_state_hash?: string;
    colony_mode?: string;
    synthetic?: boolean;
    synthetic_warning?: string;
    planning_run_id?: string;
    iteration?: number;
    target_confidence?: number;
    max_iterations?: number;
    previous_confidence?: number;
    previous_evidence_hash?: string;
    selected_gaps?: string[];
    previous_plan_draft?: WorkerPlanArtifact;
    expected_workers?: PlanningDispatch[];
    refresh?: boolean;
    revision?: PlanRevisionContext;
    existing_plan?: boolean;
    existing_phase_count?: number;
    depth?: string;
    granularity?: string;
    granularity_min?: number;
    granularity_max?: number;
    planning_depth?: string;
    verification_depth?: string;
    planning_loop?: PlanningLoop;
    survey?: SurveyContext;
    dispatches?: PlanningDispatch[];
    snapshots?: Record<string, ArtifactSnapshot>;
    dispatch_mode?: string;
    dispatch_contract?: DispatchContract;
    finalize_surface?: string;
    requires_finalizer?: boolean;
    boundary_questions?: DiscussQuestion[];
    boundary_question_count?: number;
    boundary_questions_created?: number;
    boundary_questions_existing?: number;
    orchestrator_boundary_guidance?: OrchestratorBoundaryGuidance;
}
export interface PlanRevisionContext {
    base_revision_id?: string;
    base_plan_state_hash: string;
    reason_type: "manual" | "user_feedback" | "research" | "verification_failure" | "scope_change";
    reason: string;
    evidence?: string[];
    evidence_hash?: string;
    completed_phases?: unknown[];
    superseded_phases?: unknown[];
}
export interface PlanningDispatch {
    stage?: string;
    wave?: number;
    execution_wave?: number;
    caste: string;
    agent_name?: string;
    name: string;
    task: string;
    task_id?: string;
    outputs: string[];
    status: string;
    summary?: string;
    blockers?: string[];
    duration?: number;
    brief?: string;
    files_created?: string[];
    files_modified?: string[];
    scout_report?: ScoutReport;
    phase_plan?: WorkerPlanArtifact;
    skill_section?: string;
    skill_count?: number;
    colony_skill_count?: number;
    domain_skill_count?: number;
    matched_skills?: string[];
    /** Canonical Go-owned permission request for this caste; the host copies
     * it through verbatim and never substitutes its own. */
    permission_profile?: PermissionProfile;
}
export interface PlanSynthesis {
    source: "ts-host";
    reason: string;
    phase_plan: WorkerPlanArtifact;
}
export interface ScoutReport {
    findings: ScoutFinding[];
    gaps: string[];
    confidence: number;
    study_files: string[];
}
export interface WorkerPlanArtifact {
    phases: WorkerPlanPhase[];
    confidence: PlanConfidence;
    gaps?: string[];
    planning_loop?: PlanningLoop;
}
export interface ScoutFinding {
    area: string;
    discovery: string;
    source: string;
}
export interface PlanConfidence {
    knowledge?: number;
    requirements?: number;
    risks?: number;
    dependencies?: number;
    effort?: number;
    overall?: number;
}
export interface WorkerPlanPhase {
    name: string;
    description: string;
    tasks: WorkerPlanTask[];
    success_criteria?: string[];
}
export interface WorkerPlanTask {
    goal: string;
    constraints?: string[];
    hints?: string[];
    success_criteria?: string[];
    depends_on?: string[];
}
export interface PlanningLoop {
    target_confidence: number;
    max_iterations: number;
    stall_threshold: number;
    stall_limit: number;
    accept?: boolean;
    iterations: number;
    stop_reason: string;
    final_confidence: number;
    accepted_below_target?: boolean;
    gaps?: string[];
    history?: PlanningLoopSample[];
}
export interface PlanningLoopSample {
    iteration: number;
    confidence: number;
    delta: number;
    stall_count: number;
    gaps?: string[];
    evidence?: string;
}
export interface SurveyContext {
    SurveyDir?: string;
    SurveyDocs?: string[];
    Languages?: string[];
    Frameworks?: string[];
    Directories?: string[];
    EntryPoints?: string[];
    Dependencies?: string[];
    TestFiles?: string[];
    Issues?: string[];
    SecurityPatterns?: string[];
    SourceAnchors?: string[];
}
export interface ArtifactSnapshot {
    Existed?: boolean;
    ModTime?: string;
    Size?: number;
    ContentHash?: string;
}
export interface ContinueCompletion {
    continue_manifest?: ContinuePlanManifest;
    manifest?: ContinuePlanManifest;
    dispatches?: ContinueExternalDispatch[];
    results?: ContinueExternalDispatch[];
    workers?: ContinueExternalDispatch[];
}
export interface ContinuePlanManifest {
    phase: number;
    phase_name: string;
    root: string;
    generated_at: string;
    colony_mode?: string;
    build_manifest?: string;
    verification: ContinueVerificationReport;
    assessment: ContinueAssessment;
    reconcile_task_ids?: string[];
    worker_timeout_seconds?: number;
    verification_timeout_seconds?: number;
    skip_watchers?: boolean;
    dispatches: ContinueExternalDispatch[];
    dispatch_mode: string;
    finalize_surface: string;
    requires_finalizer: boolean;
    review_depth?: string;
    boundary_questions?: DiscussQuestion[];
    boundary_question_count?: number;
    boundary_questions_created?: number;
    boundary_questions_existing?: number;
    orchestrator_boundary_guidance?: OrchestratorBoundaryGuidance;
}
export interface ContinueExternalDispatch {
    stage: string;
    wave: number;
    execution_wave?: number;
    caste: string;
    agent_name?: string;
    name: string;
    task: string;
    task_id: string;
    timeout_seconds?: number;
    status: string;
    summary?: string;
    blockers?: string[];
    duration?: number;
    report?: string;
    findings?: ReviewFinding[];
    issues?: ReviewFinding[];
    recommendations?: string[];
    weak_spots?: string[];
    edge_cases_discovered?: string[];
    reusable_lessons?: string[];
    brief?: string;
    skill_section?: string;
    skill_count?: number;
    colony_skill_count?: number;
    domain_skill_count?: number;
    matched_skills?: string[];
    handoff?: WorkerHandoff;
    /** Canonical Go-owned permission request for this caste; the host copies
     * it through verbatim and never substitutes its own. */
    permission_profile?: PermissionProfile;
}
export interface VerificationStep {
    name: string;
    command?: string;
    passed: boolean;
    skipped?: boolean;
    timed_out?: boolean;
    timeout_seconds?: number;
    exit_code?: number;
    error_class?: "product" | "environment" | "timeout" | "skipped";
    summary: string;
    output?: string;
}
export interface ClaimVerification {
    present: boolean;
    passed: boolean;
    skipped?: boolean;
    summary: string;
    checked: number;
    mismatches?: string[];
}
export interface WatcherVerification {
    present: boolean;
    passed: boolean;
    status?: string;
    worker?: string;
    summary?: string;
}
export interface ContinueVerificationReport {
    phase: number;
    generated_at: string;
    verification_timeout_seconds?: number;
    steps: VerificationStep[];
    claims: ClaimVerification;
    watcher: WatcherVerification;
    checks_passed: boolean;
    passed: boolean;
    blocking_issues?: string[];
}
export interface ContinueTaskAssessment {
    task_id: string;
    goal: string;
    outcome: string;
    summary: string;
    verified?: boolean;
    reconciled?: boolean;
    dispatch_statuses?: string[];
    recovery_action?: string;
}
export interface ContinueRecoveryPlan {
    reverify_command?: string;
    reconcile_tasks?: string[];
    reconcile_command?: string;
    redispatch_tasks?: string[];
    redispatch_command?: string;
    skip_command?: string;
}
export interface ContinueAssessment {
    phase: number;
    generated_at: string;
    tasks: ContinueTaskAssessment[];
    verification_passed: boolean;
    positive_evidence: boolean;
    partial_success?: boolean;
    operational_issues?: string[];
    reconciled_tasks?: string[];
    redispatch_tasks?: string[];
    blocking_issues?: string[];
    passed: boolean;
    summary: string;
    recovery?: ContinueRecoveryPlan;
}
export interface ReviewFinding {
    domain?: string;
    severity?: string;
    file?: string;
    line?: number;
    category?: string;
    title?: string;
    description?: string;
    suggestion?: string;
    blocking?: boolean;
}
export interface WorkflowProfileContract {
    review_depth?: string;
}
export interface DispatchContract {
    execution_model?: string;
    wave_count?: number;
    worker_count?: number;
    shared_timeout_seconds?: number;
    worker_timeout_seconds?: number;
    deadline_policy?: string;
    dependency_behavior?: string;
    fallback_behavior?: string;
    fallback_visibility?: string[];
    coordination_path?: string;
    artifact_paths?: string[];
    result_artifact_paths?: string[];
    result_collection_policy?: string;
    execution_plan?: BuildExecutionPlan[];
}
export interface QueenWorkflowRecommendation {
    review_depth?: string;
    reason?: string;
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
    selected_reasons?: Record<string, string>;
    pruned_reasons?: Record<string, string>;
    skipped_castes?: string[];
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
}
export interface DiscussQuestion {
    id?: string;
    category: string;
    question: string;
    options: string[];
    reasoning: string;
    hard_constraint?: boolean;
    status?: string;
    source?: string;
}
export interface OrchestratorBoundaryGuidance {
    active: boolean;
    workflow?: string;
    colony_mode?: string;
    pending_count?: number;
    next?: string;
    after_discuss_next?: string;
    summary?: string;
    question_ids?: string[];
    question_sources?: string[];
    question_summaries?: OrchestratorBoundaryQuestionSummary[];
}
export interface OrchestratorBoundaryQuestionSummary {
    id: string;
    source: string;
    question: string;
    options?: string[];
    hard_constraint?: boolean;
}
/**
 * Terminal worker status values accepted by Go finalizers.
 * Matches isTerminalExternalBuildStatus in cmd/codex_build_finalize.go.
 */
export type TerminalWorkerStatus = "completed" | "failed" | "blocked" | "timeout" | "manually-reconciled" | "code_written";
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
export declare const CEREMONY_TOPICS: readonly ["ceremony.build.prewave", "ceremony.build.wave.start", "ceremony.build.spawn", "ceremony.build.tool_use", "ceremony.build.wave.end", "ceremony.build.circuit_break", "ceremony.plan.wave.start", "ceremony.plan.spawn", "ceremony.plan.wave.end", "ceremony.colonize.wave.start", "ceremony.colonize.spawn", "ceremony.colonize.wave.end", "ceremony.continue.wave.start", "ceremony.continue.spawn", "ceremony.continue.wave.end", "ceremony.pheromone.emit", "ceremony.skill.activate", "ceremony.chamber.seal", "ceremony.chamber.entomb", "ceremony.midden.record", "ceremony.queen.promote", "ceremony.hive.store", "ceremony.hive.promote", "ceremony.loop.break", "ceremony.oracle.phase_transition", "ceremony.oracle.iteration"];
export type CeremonyTopic = (typeof CEREMONY_TOPICS)[number];
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
 * A single wisdom entry from the Hive Brain.
 * Mirrors the Go hiveWisdomEntry struct.
 */
export interface HiveWisdomEntry {
    id: string;
    text: string;
    domain: string;
    source_repo: string;
    source_repos: string[];
    confidence: number;
    created_at: string;
    accessed_at: string;
    access_count: number;
}
/**
 * Result from `aether hive-read` in JSON mode.
 */
export interface HiveReadResult {
    entries: HiveWisdomEntry[] | null;
    total: number;
}
/**
 * A single colony registry entry.
 * Mirrors the Go registryEntry struct.
 */
export interface RegistryEntry {
    repo_path: string;
    domains: string[];
    active: boolean;
    registered_at: string;
}
/**
 * Result from `aether registry-list` in JSON mode.
 */
export interface RegistryListResult {
    colonies: RegistryEntry[];
    total: number;
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
/**
 * A confidence measurement produced by a build or continue finalize step.
 *
 * The confidence loop aggregates these across iterations to decide whether
 * to re-dispatch workers or stop iterating.
 */
export interface ConfidenceMetric {
    /** Overall confidence score (0-100). */
    score: number;
    /** Fraction of tests that passed (0-1). */
    test_pass_rate?: number;
    /** Number of source files with passing tests. */
    files_covered?: number;
    /** Blocking issues that prevent further progress. */
    blockers?: string[];
    /** Where this metric was produced. */
    source: "build-finalize" | "continue-finalize" | "worker-claims";
}
/**
 * Ceremony data emitted after each iteration for progress visibility.
 *
 * Used by the ConfidenceLoop to render iteration markers in the build/continue
 * ceremony output (ITER-06).
 */
export interface IterationCeremonyData {
    /** Current iteration number (1-based). */
    iteration: number;
    /** Current confidence score (0-100). */
    confidence: number;
    /** Confidence change from the previous iteration. */
    delta: number;
    /** Workers remaining in the budget. */
    budgetRemaining: number;
    /** Why the iteration stopped (populated only on the final iteration). */
    stopReason?: string;
}
