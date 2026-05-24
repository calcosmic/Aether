/**
 * Event writer stub for the Aether control plane event stream.
 *
 * In stub mode, writes are accumulated in an in-memory buffer
 * and also logged to the console. No actual file I/O occurs.
 */

import { EventLine } from "./types.js";

/** In-memory buffer simulating the NDJSON stream. */
const memoryBuffers = new Map<string, EventLine[]>();

/**
 * Append a single event line to the in-memory stream.
 * In a real implementation this would append NDJSON to `streamPath`.
 */
export async function writeEvent(
  streamPath: string,
  eventLine: EventLine
): Promise<void> {
  const buffer = memoryBuffers.get(streamPath) ?? [];
  buffer.push(eventLine);
  memoryBuffers.set(streamPath, buffer);

  // Stub-mode visibility: log the NDJSON line
  console.log("[event-write]", JSON.stringify(eventLine));
}

/**
 * Create an event stream writer for a given path.
 * Returns an object with `write` and `close` methods.
 */
export function createEventStream(streamPath: string): {
  write: (eventLine: EventLine) => Promise<void>;
  close: () => Promise<void>;
} {
  return {
    async write(eventLine: EventLine): Promise<void> {
      await writeEvent(streamPath, eventLine);
    },
    async close(): Promise<void> {
      // Stub: nothing to flush in memory-only mode
      console.log("[event-close] stream closed:", streamPath);
    },
  };
}

/**
 * Retrieve the current in-memory buffer for a stream path.
 * Useful for testing and debugging.
 */
export function getBuffer(streamPath: string): readonly EventLine[] {
  return memoryBuffers.get(streamPath) ?? [];
}

/**
 * Clear the in-memory buffer for a stream path.
 * Useful for testing isolation.
 */
export function clearBuffer(streamPath: string): void {
  memoryBuffers.delete(streamPath);
}
