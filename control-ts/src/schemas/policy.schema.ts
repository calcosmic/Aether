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
    })
    .default({ max_learnings: 100, auto_promote_threshold: 0.75 }),
  skill_creation: z
    .object({
      allowed: z.boolean().default(true),
      auto_approve: z.boolean().default(false),
    })
    .default({ allowed: true, auto_approve: false }),
  safety_gates: z
    .object({
      security_scan: z.boolean().default(true),
      quality_gate: z.boolean().default(true),
    })
    .default({ security_scan: true, quality_gate: true }),
});

export type Policy = z.infer<typeof PolicySchema>;
