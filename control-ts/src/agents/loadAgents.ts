import { readdirSync, readFileSync, statSync } from "fs";
import { resolve } from "path";
import { parse } from "yaml";
import { AgentSchema, type Agent } from "../schemas/agent.schema.js";
import { projectRoot } from "../utils/projectRoot.js";

const AGENTS_DIR = resolve(projectRoot, "..", "colony", "agents");

/**
 * Load and validate all agent definitions from colony/agents/*.yaml.
 * @returns Sorted array of validated Agent objects.
 * @throws Error if directory is missing, unreadable, or any file fails validation.
 */
export function loadAgents(): Agent[] {
  let files: string[];
  try {
    files = readdirSync(AGENTS_DIR).filter((f) => f.endsWith(".yaml"));
  } catch (err) {
    throw new Error(
      `Failed to read agents directory: ${AGENTS_DIR} — ${(err as Error).message}`
    );
  }

  if (files.length === 0) {
    throw new Error(`No .yaml files found in agents directory: ${AGENTS_DIR}`);
  }

  const agents: Agent[] = [];
  for (const file of files) {
    const filePath = resolve(AGENTS_DIR, file);
    let raw: string;
    try {
      raw = readFileSync(filePath, "utf8");
    } catch (err) {
      throw new Error(
        `Failed to read agent file: ${filePath} — ${(err as Error).message}`
      );
    }

    let parsed: unknown;
    try {
      parsed = parse(raw);
    } catch (err) {
      throw new Error(
        `Failed to parse YAML in ${filePath} — ${(err as Error).message}`
      );
    }

    try {
      const agent = AgentSchema.parse(parsed);
      agents.push(agent);
    } catch (err) {
      const zodError = err as { message: string; issues?: unknown[] };
      const details = zodError.issues
        ? JSON.stringify(zodError.issues, null, 2)
        : zodError.message;
      throw new Error(
        `Validation failed for ${filePath}:\n${details}`
      );
    }
  }

  agents.sort((a, b) => a.id.localeCompare(b.id));
  return agents;
}

/**
 * Find a single agent by its id.
 * @param id Agent identifier.
 * @returns The matching Agent, or null if not found.
 */
export function loadAgentById(id: string): Agent | null {
  const agents = loadAgents();
  return agents.find((a) => a.id === id) ?? null;
}
