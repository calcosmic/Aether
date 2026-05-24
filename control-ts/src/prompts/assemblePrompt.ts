import { readFileSync } from "fs";
import { resolve } from "path";
import { projectRoot } from "../utils/projectRoot.js";
import type { Agent } from "../schemas/agent.schema.js";

const PROMPTS_DIR = resolve(projectRoot, "..", "colony", "prompts");

function buildContextInjection(context?: {
  skills?: string[];
  pheromones?: string[];
}): string {
  const parts: string[] = [];

  if (context?.skills && context.skills.length > 0) {
    parts.push("## Injected Skills");
    for (const skill of context.skills) {
      parts.push(`- ${skill}`);
    }
  }

  if (context?.pheromones && context.pheromones.length > 0) {
    if (parts.length > 0) parts.push("");
    parts.push("## Active Pheromones");
    for (const signal of context.pheromones) {
      parts.push(`- ${signal}`);
    }
  }

  if (parts.length === 0) return "";
  return "\n\n" + parts.join("\n");
}

/**
 * Assemble a prompt for a given role by reading its Markdown file.
 * @param role The agent role name (matches the filename without .md).
 * @param context Optional skills and pheromones to inject.
 * @returns The assembled prompt string.
 * @throws Error if the prompt file does not exist.
 */
export function assemblePrompt(
  role: string,
  context?: { skills?: string[]; pheromones?: string[] }
): string {
  const filePath = resolve(PROMPTS_DIR, `${role}.md`);
  let base: string;
  try {
    base = readFileSync(filePath, "utf8");
  } catch (err) {
    throw new Error(
      `Prompt file not found: ${filePath} — ${(err as Error).message}`
    );
  }
  return base + buildContextInjection(context);
}

/**
 * Assemble a prompt for a specific agent using its prompt_file field.
 * @param agent The agent definition.
 * @param context Optional skills and pheromones to inject.
 * @returns The assembled prompt string.
 * @throws Error if the referenced prompt file does not exist.
 */
export function assemblePromptForAgent(
  agent: Agent,
  context?: { skills?: string[]; pheromones?: string[] }
): string {
  const filePath = resolve(projectRoot, "..", agent.prompt_file);
  let base: string;
  try {
    base = readFileSync(filePath, "utf8");
  } catch (err) {
    throw new Error(
      `Prompt file not found: ${filePath} — ${(err as Error).message}`
    );
  }
  return base + buildContextInjection(context);
}
