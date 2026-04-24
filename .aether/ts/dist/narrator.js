import readline from "node:readline";
import { readFileSync, realpathSync } from "node:fs";
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
export function parseVisualContract(value) {
    const candidate = isRecord(value) && isRecord(value.result) ? value.result : value;
    if (!isRecord(candidate) || !isRecord(candidate.castes)) {
        throw new Error("visual contract is missing castes");
    }
    const castes = {};
    for (const [key, rawVisual] of Object.entries(candidate.castes)) {
        if (!isRecord(rawVisual)) {
            continue;
        }
        const visual = {};
        if (typeof rawVisual.emoji === "string") {
            visual.emoji = rawVisual.emoji;
        }
        if (typeof rawVisual.color === "string") {
            visual.color = rawVisual.color;
        }
        if (typeof rawVisual.label === "string") {
            visual.label = rawVisual.label;
        }
        castes[normalizeCaste(key)] = visual;
    }
    return { castes };
}
export function sanitizeTerminalText(value) {
    return String(value)
        .replace(/\u001B\[[0-?]*[ -/]*[@-~]/g, "")
        .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/g, "");
}
function truncateDisplayText(value, maxLength = 96) {
    const text = sanitizeTerminalText(value).trim().replace(/\s+/g, " ");
    if (text.length <= maxLength) {
        return text;
    }
    if (maxLength <= 3) {
        return text.slice(0, maxLength);
    }
    return `${text.slice(0, maxLength - 3).trim()}...`;
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
function normalizeCaste(caste) {
    return caste.trim().toLowerCase().replace(/[-\s]+/g, "_");
}
function formatIdentity(payload, visuals) {
    if (payload.caste === undefined && payload.name === undefined) {
        return null;
    }
    const caste = payload.caste === undefined ? "" : sanitizeTerminalText(payload.caste);
    const normalizedCaste = normalizeCaste(caste);
    const visual = normalizedCaste === "" ? undefined : visuals?.castes[normalizedCaste];
    const label = visual?.label === undefined ? caste : sanitizeTerminalText(visual.label);
    const emoji = visual?.emoji === undefined ? "" : `${sanitizeTerminalText(visual.emoji)} `;
    const name = payload.name === undefined ? "" : sanitizeTerminalText(payload.name);
    if (label !== "" && name !== "") {
        return `${emoji}${label}:${name}`;
    }
    if (label !== "") {
        return `${emoji}${label}`;
    }
    if (name !== "") {
        return name;
    }
    return null;
}
function formatWorkerIdentity(worker, visuals) {
    const caste = worker.caste === undefined ? "" : sanitizeTerminalText(worker.caste);
    const normalizedCaste = normalizeCaste(caste);
    const visual = normalizedCaste === "" ? undefined : visuals?.castes[normalizedCaste];
    const label = visual?.label === undefined ? caste : sanitizeTerminalText(visual.label);
    const emoji = visual?.emoji === undefined ? "" : `${sanitizeTerminalText(visual.emoji)} `;
    const name = worker.name === undefined ? "" : sanitizeTerminalText(worker.name);
    if (label !== "" && name !== "") {
        return `${emoji}${label}:${name}`;
    }
    if (label !== "") {
        return `${emoji}${label}`;
    }
    if (name !== "") {
        return name;
    }
    return worker.key;
}
export function renderEvent(event, visuals) {
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
    const identity = formatIdentity(payload, visuals);
    if (identity !== null) {
        parts.push(identity);
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
export function createCeremonyFrame() {
    return {
        workers: {},
        waves: {},
        notices: []
    };
}
export function applyEventToFrame(frame, event) {
    const payload = asPayload(event.payload);
    frame.last_event = event;
    const lifecycle = lifecycleFromTopic(event.topic);
    if (lifecycle !== undefined) {
        frame.lifecycle = lifecycle;
    }
    if (payload.phase !== undefined) {
        frame.phase = payload.phase;
    }
    if (payload.phase_name !== undefined) {
        frame.phase_name = sanitizeTerminalText(payload.phase_name);
    }
    if (payload.wave !== undefined) {
        frame.current_wave = payload.wave;
    }
    if (isWaveEvent(event.topic)) {
        const wave = payload.wave ?? frame.current_wave ?? 0;
        if (wave > 0) {
            const key = String(wave);
            const existing = frame.waves[key] ?? { wave };
            const next = { ...existing };
            assignDefined(next, "total", payload.total);
            assignDefined(next, "completed", payload.completed);
            assignDefined(next, "status", payload.status);
            assignDefined(next, "message", payload.message);
            frame.waves[key] = next;
        }
    }
    const workerEvent = payload.caste !== undefined || payload.name !== undefined || payload.spawn_id !== undefined;
    if (workerEvent) {
        const key = workerKey(payload, frame);
        const existing = frame.workers[key] ?? { key };
        const next = { ...existing };
        assignDefined(next, "phase", payload.phase ?? frame.phase);
        assignDefined(next, "phase_name", payload.phase_name ?? frame.phase_name);
        assignDefined(next, "wave", payload.wave ?? frame.current_wave);
        assignDefined(next, "spawn_id", payload.spawn_id);
        assignDefined(next, "caste", payload.caste);
        assignDefined(next, "name", payload.name);
        assignDefined(next, "task_id", payload.task_id);
        assignDefined(next, "task", payload.task);
        assignDefined(next, "status", payload.status);
        assignDefined(next, "message", payload.message);
        assignDefined(next, "tool_count", payload.tool_count);
        assignDefined(next, "token_count", payload.token_count);
        assignDefined(next, "files_created", payload.files_created?.length);
        assignDefined(next, "files_modified", payload.files_modified?.length);
        assignDefined(next, "tests_written", payload.tests_written?.length);
        assignDefined(next, "blockers", payload.blockers?.length);
        frame.workers[key] = next;
    }
    if (!workerEvent && shouldTrackNotice(event.topic)) {
        appendNotice(frame, event.topic, payload, lifecycle);
    }
    return frame;
}
function lifecycleFromTopic(topic) {
    const parts = topic.split(".");
    if (parts[0] !== "ceremony" || parts[1] === undefined || parts[1].trim() === "") {
        return undefined;
    }
    return sanitizeTerminalText(parts[1]);
}
function isWaveEvent(topic) {
    const parts = topic.split(".");
    return parts[0] === "ceremony"
        && parts[2] === "wave"
        && (parts[3] === "start" || parts[3] === "end");
}
function shouldTrackNotice(topic) {
    return topic.startsWith("ceremony.") && !isWaveEvent(topic);
}
function appendNotice(frame, topic, payload, lifecycle) {
    const key = `${topic}:${frame.notices.length}`;
    const notice = {
        key,
        topic: sanitizeTerminalText(topic)
    };
    assignDefined(notice, "lifecycle", lifecycle);
    assignDefined(notice, "phase", payload.phase ?? frame.phase);
    assignDefined(notice, "phase_name", payload.phase_name ?? frame.phase_name);
    assignDefined(notice, "status", payload.status);
    assignDefined(notice, "message", payload.message ?? payload.task);
    assignDefined(notice, "skill", payload.skill);
    assignDefined(notice, "pheromone_type", payload.pheromone_type);
    assignDefined(notice, "strength", payload.strength);
    frame.notices = [...frame.notices, notice].slice(-5);
}
function assignDefined(target, key, value) {
    if (value !== undefined) {
        target[key] = value;
    }
}
function workerKey(payload, frame) {
    if (payload.spawn_id !== undefined && payload.spawn_id.trim() !== "") {
        return sanitizeTerminalText(payload.spawn_id);
    }
    return [
        payload.phase ?? frame.phase ?? "",
        payload.wave ?? frame.current_wave ?? "",
        payload.caste ?? "",
        payload.name ?? "",
        payload.task_id ?? ""
    ].map((part) => sanitizeTerminalText(part)).join(":");
}
export function renderActivityFrame(frame, visuals) {
    const lines = [];
    if (frame.last_event !== undefined) {
        lines.push(renderEvent(frame.last_event, visuals));
    }
    const titleParts = ["[CEREMONY]", "COLONY ACTIVITY"];
    if (frame.lifecycle !== undefined && frame.lifecycle !== "build") {
        titleParts.push(`stage=${truncateDisplayText(frame.lifecycle, 24)}`);
    }
    if (frame.phase !== undefined) {
        titleParts.push(`phase=${sanitizeTerminalText(frame.phase)}`);
    }
    if (frame.phase_name !== undefined && frame.phase_name.trim() !== "") {
        titleParts.push(truncateDisplayText(frame.phase_name, 48));
    }
    lines.push(titleParts.join(" "));
    const currentWave = frame.current_wave === undefined ? undefined : frame.waves[String(frame.current_wave)];
    if (currentWave !== undefined) {
        lines.push(`Wave ${currentWave.wave}: ${formatWaveProgress(currentWave)}`);
    }
    appendNoticeSection(lines, frame.notices);
    const workers = Object.values(frame.workers).sort(compareWorkers);
    if (workers.length === 0) {
        lines.push("Workers: none active yet");
        return lines.join("\n");
    }
    const active = workers.filter((worker) => isActiveStatus(worker.status));
    const completed = workers.filter((worker) => worker.status === "completed");
    const blocked = workers.filter((worker) => isBlockedStatus(worker.status));
    const other = workers.filter((worker) => !active.includes(worker) && !completed.includes(worker) && !blocked.includes(worker));
    appendWorkerSection(lines, "Active", active, visuals);
    appendWorkerSection(lines, "Completed", completed, visuals);
    appendWorkerSection(lines, "Blocked", blocked, visuals);
    appendWorkerSection(lines, "Other", other, visuals);
    return lines.join("\n");
}
function compareWorkers(a, b) {
    return (a.wave ?? 0) - (b.wave ?? 0) || a.key.localeCompare(b.key);
}
function formatWaveProgress(wave) {
    if (wave.completed !== undefined && wave.total !== undefined) {
        return `${wave.completed}/${wave.total} ${wave.status ?? "running"}`;
    }
    if (wave.total !== undefined) {
        return `0/${wave.total} ${wave.status ?? "running"}`;
    }
    return wave.status ?? "running";
}
function isActiveStatus(status) {
    return status === "starting" || status === "running" || status === "active" || status === "spawned";
}
function isBlockedStatus(status) {
    return status === "failed" || status === "timeout" || status === "blocked";
}
function appendWorkerSection(lines, label, workers, visuals) {
    if (workers.length === 0) {
        return;
    }
    lines.push(`${label}:`);
    for (const worker of workers) {
        lines.push(`  ${formatWorkerLine(worker, visuals)}`);
    }
}
function appendNoticeSection(lines, notices) {
    if (notices.length === 0) {
        return;
    }
    lines.push("Context:");
    for (const notice of notices) {
        lines.push(`  ${formatNoticeLine(notice)}`);
    }
}
function formatNoticeLine(notice) {
    const details = [truncateDisplayText(notice.topic, 48)];
    if (notice.status !== undefined) {
        details.push(sanitizeTerminalText(notice.status));
    }
    if (notice.skill !== undefined) {
        details.push(`skill=${truncateDisplayText(notice.skill, 32)}`);
    }
    if (notice.pheromone_type !== undefined) {
        details.push(`pheromone=${truncateDisplayText(notice.pheromone_type, 24)}`);
    }
    if (notice.strength !== undefined) {
        details.push(`strength=${sanitizeTerminalText(notice.strength)}`);
    }
    if (notice.message !== undefined && notice.message.trim() !== "") {
        details.push(truncateDisplayText(notice.message));
    }
    return details.join(" ");
}
function formatWorkerLine(worker, visuals) {
    const identity = formatWorkerIdentity(worker, visuals);
    const details = [identity];
    if (worker.status !== undefined) {
        details.push(worker.status);
    }
    if (worker.task_id !== undefined) {
        details.push(`task=${truncateDisplayText(worker.task_id, 32)}`);
    }
    if (worker.tool_count !== undefined) {
        details.push(`tools=${worker.tool_count}`);
    }
    if (worker.token_count !== undefined) {
        details.push(`tokens=${worker.token_count}`);
    }
    const fileCount = (worker.files_created ?? 0) + (worker.files_modified ?? 0);
    if (fileCount > 0) {
        details.push(`files=${fileCount}`);
    }
    if ((worker.tests_written ?? 0) > 0) {
        details.push(`tests=${worker.tests_written}`);
    }
    if ((worker.blockers ?? 0) > 0) {
        details.push(`blockers=${worker.blockers}`);
    }
    const note = worker.message ?? worker.task;
    if (note !== undefined && note.trim() !== "") {
        details.push(truncateDisplayText(note));
    }
    return details.join(" ");
}
export function runNarrator(input = process.stdin, output = process.stdout, errorOutput = process.stderr, visuals) {
    const rl = readline.createInterface({
        input,
        crlfDelay: Infinity
    });
    const frame = createCeremonyFrame();
    rl.on("line", (line) => {
        try {
            const event = parseEvent(line);
            if (event !== null) {
                applyEventToFrame(frame, event);
                output.write(`${renderActivityFrame(frame, visuals)}\n`);
            }
        }
        catch (error) {
            const message = error instanceof Error ? error.message : String(error);
            errorOutput.write(`[CEREMONY] invalid event: ${message}\n`);
        }
    });
    return rl;
}
export function loadVisualsFromPath(path) {
    return parseVisualContract(JSON.parse(readFileSync(path, "utf8")));
}
function parseCLIVisuals(args) {
    const visualFlagIndex = args.findIndex((arg) => arg === "--visuals");
    if (visualFlagIndex === -1) {
        return undefined;
    }
    const visualPath = args[visualFlagIndex + 1];
    if (visualPath === undefined || visualPath.trim() === "") {
        throw new Error("--visuals requires a path");
    }
    return loadVisualsFromPath(visualPath);
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
    runNarrator(process.stdin, process.stdout, process.stderr, parseCLIVisuals(process.argv.slice(2)));
}
