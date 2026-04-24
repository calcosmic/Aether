import readline from "node:readline";
import { pathToFileURL } from "node:url";
import pc from "picocolors";

export interface CeremonyEvent {
  id?: string;
  topic: string;
  payload?: unknown;
  source?: string;
  timestamp?: string;
}

export interface CeremonyPayload {
  phase?: number;
  phase_name?: string;
  wave?: number;
  spawn_id?: string;
  caste?: string;
  name?: string;
  task?: string;
  status?: string;
  message?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function asPayload(value: unknown): CeremonyPayload {
  if (!isRecord(value)) {
    return {};
  }
  return value as CeremonyPayload;
}

export function sanitizeTerminalText(value: unknown): string {
  return String(value)
    .replace(/\u001B\[[0-?]*[ -/]*[@-~]/g, "")
    .replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F-\u009F]/g, "");
}

export function parseEvent(line: string): CeremonyEvent | null {
  const trimmed = line.trim();
  if (trimmed === "") {
    return null;
  }
  const parsed: unknown = JSON.parse(trimmed);
  if (!isRecord(parsed) || typeof parsed.topic !== "string") {
    throw new Error("event is missing a string topic");
  }
  const event: CeremonyEvent = { topic: parsed.topic };
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

export function renderEvent(event: CeremonyEvent): string {
  const payload = asPayload(event.payload);
  const parts: string[] = [pc.bold("[CEREMONY]"), sanitizeTerminalText(event.topic)];

  if (payload.phase !== undefined) {
    parts.push(`phase=${sanitizeTerminalText(payload.phase)}`);
  }
  if (payload.wave !== undefined) {
    parts.push(`wave=${sanitizeTerminalText(payload.wave)}`);
  }
  if (payload.caste !== undefined || payload.name !== undefined) {
    const identityParts: string[] = [];
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
  if (payload.message !== undefined) {
    parts.push(sanitizeTerminalText(payload.message));
  } else if (payload.task !== undefined) {
    parts.push(sanitizeTerminalText(payload.task));
  }

  return parts.filter((part) => part.trim() !== "").join(" ");
}

export function runNarrator(
  input: NodeJS.ReadableStream = process.stdin,
  output: NodeJS.WritableStream = process.stdout,
  errorOutput: NodeJS.WritableStream = process.stderr
): readline.Interface {
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
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : String(error);
      errorOutput.write(`[CEREMONY] invalid event: ${message}\n`);
    }
  });

  return rl;
}

if (process.argv[1] !== undefined && import.meta.url === pathToFileURL(process.argv[1]).href) {
  runNarrator();
}
