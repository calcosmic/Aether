/**
 * Prompt assembler for the TypeScript orchestration host.
 *
 * Loads agent definitions from platform-specific paths, assembles worker
 * prompts matching Go's AssemblePrompt output, and renders the response
 * contract that instructs workers to return structured JSON claims.
 *
 * Satisfies TS-01 (real worker dispatch).
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";

import type { Platform } from "./platform-dispatcher.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Configuration for assembling a worker prompt. */
export interface PromptAssemblyConfig {
  /** Repository root (cwd). */
  cwd: string;
  /** Worker caste (builder, watcher, etc.). */
  caste: string;
  /** Deterministic worker name. */
  name: string;
  /** Task description. */
  task: string;
  /** Target platform. */
  platform: Platform;
  /** Agent name (e.g. "aether-builder"). */
  agentName: string;
}

// ---------------------------------------------------------------------------
// Agent definition loading
// ---------------------------------------------------------------------------

/**
 * Load the raw agent definition for a platform and agent name.
 *
 * Resolves the correct path per platform:
 * - Claude: `.claude/agents/ant/${agentName}.md`
 * - OpenCode: `.opencode/agents/${agentName}.md`
 * - Codex: `.codex/agents/${agentName}.toml`
 *
 * @param cwd - Repository root
 * @param platform - Target platform
 * @param agentName - Agent name (e.g. "aether-builder")
 * @returns Raw file content as string
 * @throws Error if the agent definition file is not found
 */
export function loadAgentDefinition(
  cwd: string,
  platform: Platform,
  agentName: string
): string {
  let filePath: string;
  switch (platform) {
    case "claude":
      filePath = join(cwd, ".claude", "agents", "ant", `${agentName}.md`);
      break;
    case "opencode":
      filePath = join(cwd, ".opencode", "agents", `${agentName}.md`);
      break;
    case "codex":
      filePath = join(cwd, ".codex", "agents", `${agentName}.toml`);
      break;
  }

  try {
    return readFileSync(filePath, "utf-8");
  } catch {
    throw new Error(`Agent definition not found: ${agentName} at ${filePath}`);
  }
}

// ---------------------------------------------------------------------------
// Prompt assembly
// ---------------------------------------------------------------------------

/**
 * Assemble a complete worker prompt from agent definition, context, and task.
 *
 * Sections are assembled in order:
 * 1. Agent Definition
 * 2. Context Capsule (stub — to be filled with QUEEN.md, colony goal, etc.)
 * 3. Worker Handoff Section (stub)
 * 4. Skill Section (stub)
 * 5. Pheromone Section (stub)
 * 6. Task Brief
 * 7. Response Contract
 *
 * Stubs for context capsule, handoff, skills, and pheromones are included
 * as placeholders for future waves.
 *
 * @param config - Prompt assembly configuration
 * @returns Fully assembled prompt string
 */
export function assemblePrompt(config: PromptAssemblyConfig): string {
  const agentDef = loadAgentDefinition(config.cwd, config.platform, config.agentName);

  const contextCapsule = renderContextCapsule(config);
  const handoffSection = ""; // stub
  const skillSection = ""; // stub
  const pheromoneSection = ""; // stub
  const taskBrief = renderTaskBrief(config);
  const responseContract = renderResponseContract(config);

  const parts: string[] = [];
  parts.push(agentDef);
  if (contextCapsule) parts.push(contextCapsule);
  if (handoffSection) parts.push(handoffSection);
  if (skillSection) parts.push(skillSection);
  if (pheromoneSection) parts.push(pheromoneSection);
  parts.push(taskBrief);
  parts.push(responseContract);

  return parts.join("\n\n");
}

// ---------------------------------------------------------------------------
// Section renderers
// ---------------------------------------------------------------------------

/**
 * Render a simplified context capsule.
 *
 * For now, this loads QUEEN.md if it exists and includes a colony goal
 * placeholder. Full context capsule assembly (skills, pheromones, hive
 * wisdom, etc.) will be added in a later wave.
 *
 * @param config - Prompt assembly configuration
 * @returns Context capsule string (may be empty if no context available)
 */
function renderContextCapsule(config: PromptAssemblyConfig): string {
  const parts: string[] = [];

  // Try to load QUEEN.md from hub
  try {
    const { homedir } = require("node:os");
    const queenPath = join(homedir(), ".aether", "QUEEN.md");
    const queenContent = readFileSync(queenPath, "utf-8");
    // Take first 2000 chars as a compact context capsule
    parts.push("## Colony Wisdom (QUEEN.md)\n\n" + queenContent.slice(0, 2000));
  } catch {
    // QUEEN.md is optional
  }

  return parts.join("\n\n");
}

/**
 * Render a markdown task brief for the worker.
 *
 * @param config - Prompt assembly configuration
 * @returns Task brief string
 */
function renderTaskBrief(config: PromptAssemblyConfig): string {
  return `# Task: ${config.name}

Caste: ${config.caste}
Agent: ${config.agentName}

## Goal

${config.task}

## Success Criteria

- Complete the task described above.
- Report all files created, modified, and tests written.
- Return a JSON claims object as your final response.
`;
}

/**
 * Render the response contract that instructs the worker to return
 * structured JSON claims.
 *
 * @param config - Prompt assembly configuration
 * @returns Response contract markdown string
 */
export function renderResponseContract(config: PromptAssemblyConfig): string {
  const statusValues =
    config.caste.toLowerCase() === "builder"
      ? "code_written, completed, failed, blocked"
      : "completed, failed, blocked";

  return `## Final Response Contract

Return ONLY a single JSON object as your final response.
- Do not wrap the JSON in markdown code fences.
- Use repo-relative paths rooted at "${config.cwd}" in files_created, files_modified, and tests_written.
- Set status to one of: ${statusValues}.
- Report blockers truthfully. If blocked, explain why in blockers.
- Include handoff with changed_files, commands_run, verification_status, known_failures, open_decisions, assumptions, next_worker_instructions, do_not_repeat, and freshness.
- Keep summary concise and concrete.

Required JSON fields:
- ant_name (string)
- caste (string)
- task_id (string)
- status (string)
- summary (string)
- files_created (array of strings)
- files_modified (array of strings)
- tests_written (array of strings)
- tool_count (integer)
- blockers (array of strings)
- spawns (array of strings)
- handoff (object)
`;
}

// ---------------------------------------------------------------------------
// Caste-to-agent mapping
// ---------------------------------------------------------------------------

/**
 * Map a caste name to its canonical agent definition name.
 *
 * @param caste - Caste name (e.g. "builder", "watcher")
 * @returns Agent name (e.g. "aether-builder", "aether-watcher")
 */
export function getAgentNameForCaste(caste: string): string {
  const map: Record<string, string> = {
    builder: "aether-builder",
    watcher: "aether-watcher",
    scout: "aether-scout",
    architect: "aether-architect",
    chaos: "aether-chaos",
    archaeologist: "aether-archaeologist",
    oracle: "aether-oracle",
    route_setter: "aether-route-setter",
    ambassador: "aether-ambassador",
    auditor: "aether-auditor",
    chronicler: "aether-chronicler",
    gatekeeper: "aether-gatekeeper",
    guardian: "aether-guardian",
    includer: "aether-includer",
    keeper: "aether-keeper",
    measurer: "aether-measurer",
    probe: "aether-probe",
    tracker: "aether-tracker",
    weaver: "aether-weaver",
    dreamer: "aether-dreamer",
    medic: "aether-medic",
    fixer: "aether-fixer",
    porter: "aether-porter",
    sage: "aether-sage",
    colonizer: "aether-colonizer",
    surveyor: "aether-surveyor",
    queen: "aether-queen",
  };
  return map[caste.toLowerCase()] ?? `aether-${caste}`;
}
