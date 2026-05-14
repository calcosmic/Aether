/**
 * Event bridge: consume Go ceremony events via JSONL replay + streaming.
 *
 * The bridge calls `aether event-bus-replay` for historical events, then spawns
 * `aether event-bus-subscribe --stream` to tail new events as NDJSON.
 *
 * Boundary contract: this module NEVER writes to `.aether/data/`. It only reads
 * from the JSONL stream. Any write-mode attempt on a Go-owned path throws
 * BoundaryViolationError.
 */

import { spawn, type ChildProcess } from "node:child_process";
import { createInterface } from "node:readline";

import { callGoJSON, type GoBridgeOptions } from "./go-bridge.js";
import type { CeremonyEvent, CeremonyPayload } from "./types.js";
import {
  assertNoWriteToData,
  BoundaryViolationError,
} from "./boundary-reference.js";

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

export interface EventBridgeOptions extends GoBridgeOptions {
  /** Topic filter (supports trailing * wildcard). Default: "ceremony.*" */
  filter?: string;
  /** Polling interval for the stream in milliseconds. Default: 250 */
  pollIntervalMs?: number;
  /** Replay events since this ISO-8601 timestamp. Default: "" (all) */
  replaySince?: string;
  /** Called for every validated ceremony event. */
  onEvent: (event: CeremonyEvent) => void;
}

// ---------------------------------------------------------------------------
// Controller
// ---------------------------------------------------------------------------

export interface EventBridgeController {
  /** Stop the stream and clean up resources. */
  stop(): Promise<void>;
}

// ---------------------------------------------------------------------------
// State guards
// ---------------------------------------------------------------------------

/** Maximum number of seen event IDs to retain before LRU eviction. */
const MAX_SEEN_EVENTS = 10000;

// ---------------------------------------------------------------------------
// Boundary guard
// ---------------------------------------------------------------------------

/**
 * Runtime boundary enforcement: reject any attempt to open a Go-owned path
 * in write mode. The event bridge is read-only by design.
 */
function guardWriteMode(path: string, mode?: string): void {
  assertNoWriteToData(path, mode);
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Start the event bridge: replay historical events, then stream new ones.
 *
 * @returns A controller with a `stop()` method.
 */
export async function startEventBridge(
  opts: EventBridgeOptions
): Promise<EventBridgeController> {
  const filter = opts.filter ?? "ceremony.*";
  const pollIntervalMs = opts.pollIntervalMs ?? 250;
  const replaySince = opts.replaySince ?? "";

  // Boundary guard: event bridge must never write to the JSONL file
  guardWriteMode(".aether/data/event-bus.jsonl", "read");

  const seen = new Set<string>();

  // --- Replay historical events -------------------------------------------
  const replayArgs = ["event-bus-replay", "--topic", filter];
  if (replaySince) {
    replayArgs.push("--since", replaySince);
  }

  let replayEvents: CeremonyEvent[] = [];
  try {
    const replayResult = callGoJSON<{ events: CeremonyEvent[] }>(
      opts,
      replayArgs
    );
    replayEvents = replayResult.events ?? [];
  } catch {
    // If replay fails (e.g., no store initialised), continue to stream.
    replayEvents = [];
  }

  for (const evt of replayEvents) {
    if (evt.id && seen.has(evt.id)) continue;
    if (evt.id) seen.add(evt.id);
    if (isValidCeremonyEvent(evt)) {
      opts.onEvent(evt);
    }
  }

  // --- Stream new events via NDJSON subprocess ----------------------------
  const streamArgs = [
    "event-bus-subscribe",
    "--stream",
    "--filter",
    filter,
    "--poll-interval",
    `${pollIntervalMs}ms`,
  ];

  const child = spawn(opts.goBinaryPath, streamArgs, {
    cwd: opts.cwd,
    env: { ...process.env, AETHER_OUTPUT_MODE: "json" },
    stdio: ["ignore", "pipe", "pipe"],
  });

  // Log stderr with prefix
  child.stderr?.on("data", (chunk: Buffer) => {
    const lines = chunk.toString("utf-8").split("\n").filter(Boolean);
    for (const line of lines) {
      process.stderr.write(`[event-bridge] ${line}\n`);
    }
  });

  // Parse NDJSON lines from stdout
  const rl = createInterface({ input: child.stdout! });

  rl.on("line", (line) => {
    if (!line.trim()) return;
    let parsed: unknown;
    try {
      parsed = JSON.parse(line);
    } catch {
      // Malformed NDJSON line — skip
      return;
    }
    if (!isValidCeremonyEvent(parsed)) return;
    const evt = parsed as CeremonyEvent;
    if (evt.id && seen.has(evt.id)) return;
    if (evt.id) {
      seen.add(evt.id);
      if (seen.size > MAX_SEEN_EVENTS) {
        // Simple LRU: clear half the set when max is reached
        const toDelete = Math.floor(MAX_SEEN_EVENTS / 2);
        const iter = seen.values();
        for (let i = 0; i < toDelete; i++) {
          const val = iter.next().value;
          if (val !== undefined) seen.delete(val);
        }
      }
    }
    opts.onEvent(evt);
  });

  child.on("error", (err) => {
    process.stderr.write(`[event-bridge] subprocess error: ${err.message}\n`);
  });

  // --- Controller ----------------------------------------------------------
  const controller: EventBridgeController = {
    async stop() {
      rl.close();
      child.stdout?.destroy();
      child.stderr?.destroy();
      if (!child.killed) {
        child.kill("SIGTERM");
        // Wait for subprocess to actually exit (max 2 seconds)
        await new Promise<void>((resolve) => {
          const timeout = setTimeout(() => {
            child.removeListener("exit", onExit);
            if (!child.killed) {
              child.kill("SIGKILL");
            }
            resolve();
          }, 2000);
          const onExit = () => {
            clearTimeout(timeout);
            resolve();
          };
          child.once("exit", onExit);
        });
      }
      seen.clear();
    },
  };

  return controller;
}

/**
 * Convenience wrapper to stop an event bridge controller.
 */
export async function stopEventBridge(controller: EventBridgeController): Promise<void> {
  await controller.stop();
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

function isValidCeremonyEvent(obj: unknown): obj is CeremonyEvent {
  if (typeof obj !== "object" || obj === null) return false;
  const e = obj as Record<string, unknown>;
  if (typeof e.id !== "string") return false;
  if (typeof e.topic !== "string") return false;
  if (typeof e.payload !== "object" || e.payload === null) return false;
  if (typeof e.source !== "string") return false;
  if (typeof e.timestamp !== "string") return false;
  if (typeof e.ttl_days !== "number") return false;
  if (typeof e.expires_at !== "string") return false;
  return true;
}

// Re-export BoundaryViolationError so consumers can catch it
export { BoundaryViolationError };
