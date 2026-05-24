import { z } from "zod";

export const PolicySchema = z.object({
  model_routing: z
    .object({
      default_provider: z.string().min(1).optional(),
      fallback_providers: z.array(z.string()).default([]),
      routing_rules: z
        .array(
          z.object({
            agent_role: z.string().min(1),
            provider: z.string().min(1),
            model: z.string().optional(),
          })
        )
        .default([]),
    })
    .default({ fallback_providers: [], routing_rules: [] }),
  memory_rules: z
    .object({
      max_learnings: z.number().int().positive().default(100),
      auto_promote_threshold: z.number().default(0.75),
      learning_retention_days: z.number().int().positive().default(60),
      instinct_cap: z.number().int().positive().default(30),
      event_cap: z.number().int().positive().default(100),
    })
    .default({
      max_learnings: 100,
      auto_promote_threshold: 0.75,
      learning_retention_days: 60,
      instinct_cap: 30,
      event_cap: 100,
    }),
  skill_creation: z
    .object({
      allowed: z.boolean().default(true),
      auto_approve: z.boolean().default(false),
      max_skills_per_colony: z.number().int().positive().default(50),
      require_wisdom_threshold: z.number().default(0.8),
    })
    .default({
      allowed: true,
      auto_approve: false,
      max_skills_per_colony: 50,
      require_wisdom_threshold: 0.8,
    }),
  safety_gates: z
    .object({
      security_scan: z.boolean().default(true),
      quality_gate: z.boolean().default(true),
      chaos_scan: z.boolean().default(true),
      auditor_score_threshold: z.number().int().positive().default(60),
    })
    .default({
      security_scan: true,
      quality_gate: true,
      chaos_scan: true,
      auditor_score_threshold: 60,
    }),
  dispatch_contract: z
    .object({
      max_workers_per_phase: z.number().int().positive().default(10),
      spawn_depth_limits: z
        .record(z.number().int(), z.number().int().nonnegative())
        .default({ 0: 4, 1: 4, 2: 2, 3: 0 }),
      timeout_defaults: z
        .record(z.string(), z.number().int().positive())
        .default({
          build: 600,
          continue: 300,
          plan: 300,
          verify: 180,
        }),
    })
    .default({
      max_workers_per_phase: 10,
      spawn_depth_limits: { 0: 4, 1: 4, 2: 2, 3: 0 },
      timeout_defaults: {
        build: 600,
        continue: 300,
        plan: 300,
        verify: 180,
      },
    }),
  pheromone_lifecycle: z
    .object({
      default_ttl_days: z.number().int().positive().default(30),
      auto_expire_on_phase_end: z.boolean().default(true),
      max_active_signals: z.number().int().positive().default(100),
      strength_decay_per_day: z.number().default(0.05),
    })
    .default({
      default_ttl_days: 30,
      auto_expire_on_phase_end: true,
      max_active_signals: 100,
      strength_decay_per_day: 0.05,
    }),
  signal_rules: z
    .object({
      hard_constraint_prefixes: z
        .array(z.string().min(1))
        .default(["[error-pattern]", "[redirect]"]),
      auto_emit_on_phase_complete: z.boolean().default(true),
      max_feedback_per_phase: z.number().int().positive().default(3),
    })
    .default({
      hard_constraint_prefixes: ["[error-pattern]", "[redirect]"],
      auto_emit_on_phase_complete: true,
      max_feedback_per_phase: 3,
    }),
  autopilot: z
    .object({
      pause_conditions: z.array(z.string().min(1)).default([
        "test_failure",
        "critical_chaos",
        "security_gate_failure",
        "quality_gate_failure",
        "runtime_verification_needed",
      ]),
      replan_interval: z.number().int().positive().default(3),
      max_phases_per_run: z.number().int().positive().optional(),
    })
    .default({
      pause_conditions: [
        "test_failure",
        "critical_chaos",
        "security_gate_failure",
        "quality_gate_failure",
        "runtime_verification_needed",
      ],
      replan_interval: 3,
    }),
});

export type Policy = z.infer<typeof PolicySchema>;
