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
];
