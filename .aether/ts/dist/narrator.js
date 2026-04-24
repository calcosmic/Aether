import readline from "node:readline";
import { realpathSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
function isRecord(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}
function asPayload(value) {
    if (!isRecord(value)) {
        return {};
    }
    return value;
}
export function sanitizeTerminalText(value) {
    return String(value)
        .replace(/\u001B\[[0-?]*[ -/]*[@-~]/g, "")
        .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/g, "");
}
export function parseEvent(line) {
    const trimmed = line.trim();
    if (trimmed === "") {
        return null;
    }
    const parsed = JSON.parse(trimmed);
    if (!isRecord(parsed) || typeof parsed.topic !== "string") {
        throw new Error("event is missing a string topic");
    }
    const event = { topic: parsed.topic };
    if (typeof parsed.id === "string") {
        event.id = parsed.id;
    }
    if ("payload" in parsed) {
        event.payload = parsed.payload;
    }
    if (typeof parsed.source === "string") {
        event.source = parsed.source;
    }
    if (typeof parsed.timestamp === "string") {
        event.timestamp = parsed.timestamp;
    }
    return event;
}
export function renderEvent(event) {
    const payload = asPayload(event.payload);
    const parts = ["[CEREMONY]", sanitizeTerminalText(event.topic)];
    if (payload.phase !== undefined) {
        parts.push(`phase=${sanitizeTerminalText(payload.phase)}`);
    }
    if (payload.phase_name !== undefined) {
        parts.push(`phase_name=${sanitizeTerminalText(payload.phase_name)}`);
    }
    if (payload.wave !== undefined) {
        parts.push(`wave=${sanitizeTerminalText(payload.wave)}`);
    }
    if (payload.spawn_id !== undefined) {
        parts.push(`spawn=${sanitizeTerminalText(payload.spawn_id)}`);
    }
    if (payload.caste !== undefined || payload.name !== undefined) {
        const identityParts = [];
        if (payload.caste !== undefined) {
            identityParts.push(sanitizeTerminalText(payload.caste));
        }
        if (payload.name !== undefined) {
            identityParts.push(sanitizeTerminalText(payload.name));
        }
        parts.push(identityParts.join(":"));
    }
    if (payload.status !== undefined) {
        parts.push(`status=${sanitizeTerminalText(payload.status)}`);
    }
    if (payload.task_id !== undefined) {
        parts.push(`task_id=${sanitizeTerminalText(payload.task_id)}`);
    }
    if (payload.skill !== undefined) {
        parts.push(`skill=${sanitizeTerminalText(payload.skill)}`);
    }
    if (payload.pheromone_type !== undefined) {
        parts.push(`pheromone=${sanitizeTerminalText(payload.pheromone_type)}`);
    }
    if (payload.strength !== undefined) {
        parts.push(`strength=${sanitizeTerminalText(payload.strength)}`);
    }
    if (payload.completed !== undefined && payload.total !== undefined) {
        parts.push(`progress=${sanitizeTerminalText(payload.completed)}/${sanitizeTerminalText(payload.total)}`);
    }
    if (payload.tool_count !== undefined) {
        parts.push(`tools=${sanitizeTerminalText(payload.tool_count)}`);
    }
    if (payload.token_count !== undefined) {
        parts.push(`tokens=${sanitizeTerminalText(payload.token_count)}`);
    }
    if (payload.files_created !== undefined) {
        parts.push(`created=${sanitizeTerminalText(payload.files_created.length)}`);
    }
    if (payload.files_modified !== undefined) {
        parts.push(`modified=${sanitizeTerminalText(payload.files_modified.length)}`);
    }
    if (payload.tests_written !== undefined) {
        parts.push(`tests=${sanitizeTerminalText(payload.tests_written.length)}`);
    }
    if (payload.blockers !== undefined) {
        parts.push(`blockers=${sanitizeTerminalText(payload.blockers.length)}`);
    }
    if (payload.success_criteria !== undefined) {
        parts.push(`criteria=${sanitizeTerminalText(payload.success_criteria.length)}`);
    }
    if (payload.message !== undefined) {
        parts.push(sanitizeTerminalText(payload.message));
    }
    else if (payload.task !== undefined) {
        parts.push(sanitizeTerminalText(payload.task));
    }
    return parts.filter((part) => part.trim() !== "").join(" ");
}
export function runNarrator(input = process.stdin, output = process.stdout, errorOutput = process.stderr) {
    const rl = readline.createInterface({
        input,
        crlfDelay: Infinity
    });
    rl.on("line", (line) => {
        try {
            const event = parseEvent(line);
            if (event !== null) {
                output.write(`${renderEvent(event)}\n`);
            }
        }
        catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            errorOutput.write(`[CEREMONY] invalid event: ${message}\n`);
        }
    });
    return rl;
}
function realpathOrResolve(path) {
    try {
        return realpathSync(path);
    }
    catch {
        return resolve(path);
    }
}
function isEntrypoint(importURL, argvPath) {
    if (argvPath === undefined) {
        return false;
    }
    return realpathOrResolve(fileURLToPath(importURL)) === realpathOrResolve(argvPath);
}
if (isEntrypoint(import.meta.url, process.argv[1])) {
    runNarrator();
}
