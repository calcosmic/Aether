/**
 * Legacy TypeScript prompt assembler retained for regression tests only.
 *
 * Production worker prompts are assembled by Go at the same boundary that
 * selects and launches the provider.
 *
 * Loads agent definitions from platform-specific paths, assembles worker
 * prompts matching Go's AssemblePrompt output, and renders the response
 * contract that instructs workers to return structured JSON claims.
 *
 * @deprecated Test/forensic compatibility only; not production prompt truth.
 */
import { existsSync, readFileSync } from "node:fs";
import { homedir } from "node:os";
import { join } from "node:path";
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
export function loadAgentDefinition(cwd, platform, agentName, homeDir) {
    const candidates = agentDefinitionCandidatePaths(cwd, platform, agentName, homeDir);
    for (const filePath of candidates) {
        if (!existsSync(filePath)) {
            continue;
        }
        try {
            return readFileSync(filePath, "utf-8");
        }
        catch (err) {
            const msg = err instanceof Error ? err.message : String(err);
            throw new Error(`Agent definition could not be read: ${agentName} at ${filePath}: ${msg}`);
        }
    }
    throw new Error(`Agent definition not found: ${agentName}; searched: ${candidates.join(", ")}`);
}
// ---------------------------------------------------------------------------
// Prompt assembly
// ---------------------------------------------------------------------------
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
export function assemblePrompt(config) {
    const agentDef = loadAgentDefinition(config.cwd, config.platform, config.agentName, config.homeDir);
    const contextCapsule = firstNonEmpty(config.contextCapsule, renderContextCapsule(config));
    const handoffSection = compactSection(config.handoffSection);
    const skillSection = compactSection(config.skillSection);
    const hiveSection = compactSection(config.hiveSection);
    const pheromoneSection = compactSection(config.pheromoneSection);
    const taskBrief = firstNonEmpty(config.taskBrief, renderTaskBrief(config));
    const responseContract = renderResponseContract(config);
    const parts = [];
    parts.push(agentDef);
    if (contextCapsule)
        parts.push(contextCapsule);
    if (handoffSection)
        parts.push(handoffSection);
    if (skillSection)
        parts.push(skillSection);
    if (hiveSection)
        parts.push(hiveSection);
    if (pheromoneSection)
        parts.push(pheromoneSection);
    parts.push(taskBrief);
    parts.push(responseContract);
    return parts.join("\n\n");
}
function agentDefinitionCandidatePaths(cwd, platform, agentName, homeDir) {
    const base = normalizeAgentBase(agentName);
    const local = localAgentDefinitionPath(cwd, platform, base);
    if (isAetherSourceRoot(cwd)) {
        return [local];
    }
    const home = homeDir ?? homedir();
    return compactStrings([
        globalAgentDefinitionPath(home, platform, base),
        hubAgentDefinitionPath(home, platform, base),
        local,
    ]);
}
function normalizeAgentBase(agentName) {
    return agentName.trim().replace(/\.(md|toml)$/i, "");
}
function localAgentDefinitionPath(cwd, platform, base) {
    switch (platform) {
        case "claude":
            return join(cwd, ".claude", "agents", "ant", `${base}.md`);
        case "opencode":
            return join(cwd, ".opencode", "agents", `${base}.md`);
        case "codex":
            return join(cwd, ".codex", "agents", `${base}.toml`);
    }
}
function globalAgentDefinitionPath(home, platform, base) {
    if (!home.trim())
        return "";
    switch (platform) {
        case "claude":
            return join(home, ".claude", "agents", "ant", `${base}.md`);
        case "opencode":
            return join(home, ".config", "opencode", "agents", `${base}.md`);
        case "codex":
            return join(home, ".codex", "agents", `${base}.toml`);
    }
}
function hubAgentDefinitionPath(home, platform, base) {
    const hub = resolveHubDir(home);
    if (!hub)
        return "";
    switch (platform) {
        case "claude":
            return join(hub, "system", "agents-claude", `${base}.md`);
        case "opencode":
            return join(hub, "system", "agents", `${base}.md`);
        case "codex":
            return join(hub, "system", "codex", `${base}.toml`);
    }
}
function resolveHubDir(home) {
    const explicit = process.env["AETHER_HUB_DIR"]?.trim();
    if (explicit)
        return explicit;
    if (!home.trim())
        return "";
    const hubName = process.env["AETHER_CHANNEL"]?.trim().toLowerCase() === "dev"
        ? ".aether-dev"
        : ".aether";
    return join(home, hubName);
}
function isAetherSourceRoot(cwd) {
    try {
        const goMod = readFileSync(join(cwd, "go.mod"), "utf-8");
        return (goMod.includes("module github.com/calcosmic/Aether") &&
            existsSync(join(cwd, "cmd", "aether", "main.go")));
    }
    catch {
        return false;
    }
}
function compactStrings(values) {
    return values.filter((value) => value.trim() !== "");
}
// ---------------------------------------------------------------------------
// Section renderers
// ---------------------------------------------------------------------------
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
export function renderContextCapsule(config) {
    const parts = [];
    const localQueen = readOptionalText(join(config.cwd, ".aether", "QUEEN.md"));
    if (localQueen) {
        parts.push("## Repo Queen Wisdom (.aether/QUEEN.md)\n\n" + localQueen.slice(0, 2000));
    }
    const home = config.homeDir ?? homedir();
    const globalQueen = readOptionalText(join(home, ".aether", "QUEEN.md"));
    if (globalQueen) {
        parts.push("## Global Queen Wisdom (~/.aether/QUEEN.md)\n\n" + globalQueen.slice(0, 2000));
    }
    return parts.join("\n\n");
}
function readOptionalText(path) {
    if (!existsSync(path)) {
        return "";
    }
    try {
        return readFileSync(path, "utf-8").trim();
    }
    catch {
        return "";
    }
}
function compactSection(value) {
    return typeof value === "string" ? value.trim() : "";
}
function firstNonEmpty(...values) {
    for (const value of values) {
        const compacted = compactSection(value);
        if (compacted) {
            return compacted;
        }
    }
    return "";
}
/**
 * Render a markdown task brief for the worker.
 *
 * @param config - Prompt assembly configuration
 * @returns Task brief string
 */
function renderTaskBrief(config) {
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
export function renderResponseContract(config) {
    const statusValues = config.caste.toLowerCase() === "builder"
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
export function getAgentNameForCaste(caste) {
    const map = {
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
