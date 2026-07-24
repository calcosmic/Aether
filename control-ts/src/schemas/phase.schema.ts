import { z } from "zod";

export const PhaseSchema = z.object({
  id: z.string().min(1),
  entry_agent: z.string().min(1),
  required_agents: z.array(z.string()).min(1),
  inputs: z.record(z.string(), z.unknown()).default({}),
  outputs: z.array(z.string()).default([]),
  success_criteria: z.array(z.string()).min(1),
  failure_policy: z.enum(["retry", "skip", "block", "escalate"]).default("retry"),
  ceremony: z
    .object({
      stages: z.array(z.string()).default([]),
    })
    .optional(),
});

export type Phase = z.infer<typeof PhaseSchema>;
