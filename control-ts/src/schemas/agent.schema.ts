import { z } from "zod";
import { existsSync } from "fs";
import { resolve } from "path";
import { projectRoot } from "../utils/projectRoot.js";

export const AgentSchema = z
  .object({
    id: z.string().min(1),
    role: z.enum([
      "builder",
      "watcher",
      "scout",
      "queen",
      "oracle",
      "gatekeeper",
      "auditor",
      "probe",
      "architect",
      "route-setter",
      "surveyor-nest",
      "surveyor-disciplines",
      "surveyor-pathogens",
      "surveyor-provisions",
      "keeper",
      "tracker",
      "weaver",
      "fixer",
      "medic",
      "porter",
      "ambassador",
      "chronicler",
      "measurer",
      "includer",
      "sage",
      "chaos",
      "archaeologist",
    ]),
    prompt_file: z.string().min(1),
    allowed_tools: z.array(z.string()).default([]),
  })
  .superRefine((data, ctx) => {
    if (data.prompt_file.includes("..")) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `prompt_file contains path traversal: ${data.prompt_file}`,
        path: ["prompt_file"],
      });
      return;
    }
    const resolved = resolve(projectRoot, data.prompt_file);
    if (!existsSync(resolved)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: `prompt_file does not exist: ${resolved}`,
        path: ["prompt_file"],
      });
    }
  });

export type Agent = z.infer<typeof AgentSchema>;
