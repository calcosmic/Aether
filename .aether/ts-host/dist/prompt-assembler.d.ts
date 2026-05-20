/**
 * Prompt assembler for the TypeScript orchestration host.
 *
 * Loads agent definitions from platform-specific paths, assembles worker
 * prompts matching Go's AssemblePrompt output, and renders the response
 * contract that instructs workers to return structured JSON claims.
 *
 * Satisfies TS-01 (real worker dispatch).
 */
import type { Platform } from "./platform-dispatcher.js";
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
    /** Go-provided colony-prime context section. */
    contextCapsule?: string | undefined;
    /** Go-provided previous worker handoff section. */
    handoffSection?: string | undefined;
    /** Go-provided skill injection section. */
    skillSection?: string | undefined;
    /** Cross-colony hive wisdom section. Injected after skills when present. */
    hiveSection?: string | undefined;
    /** Go-provided pheromone section, if not folded into context. */
    pheromoneSection?: string | undefined;
    /** Go-provided task brief. */
    taskBrief?: string | undefined;
    /** Test override for global Queen scope. */
    homeDir?: string | undefined;
}
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
export declare function loadAgentDefinition(cwd: string, platform: Platform, agentName: string, homeDir?: string): string;
/**
 * Assemble a complete worker prompt from agent definition, context, and task.
 *
 * Sections are assembled in order:
 * 1. Agent Definition
 * 2. Context Capsule (Go-provided, or a local/global Queen fallback)
 * 3. Worker Handoff Section
 * 4. Skill Section
 * 4a. Hive Wisdom Section (when present)
 * 5. Pheromone Section
 * 6. Task Brief
 * 7. Response Contract
 *
 * @param config - Prompt assembly configuration
 * @returns Fully assembled prompt string
 */
export declare function assemblePrompt(config: PromptAssemblyConfig): string;
/**
 * Render a simplified context capsule fallback.
 *
 * Real worker dispatch should pass Go's colony-prime prompt section through
 * `contextCapsule`. This fallback only keeps direct prompt assembly usable in
 * tests or platforms that have not yet fetched Go context.
 *
 * @param config - Prompt assembly configuration
 * @returns Context capsule string (may be empty if no context available)
 */
export declare function renderContextCapsule(config: PromptAssemblyConfig): string;
/**
 * Render the response contract that instructs the worker to return
 * structured JSON claims.
 *
 * @param config - Prompt assembly configuration
 * @returns Response contract markdown string
 */
export declare function renderResponseContract(config: PromptAssemblyConfig): string;
/**
 * Map a caste name to its canonical agent definition name.
 *
 * @param caste - Caste name (e.g. "builder", "watcher")
 * @returns Agent name (e.g. "aether-builder", "aether-watcher")
 */
export declare function getAgentNameForCaste(caste: string): string;
