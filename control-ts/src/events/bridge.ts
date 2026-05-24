/**
 * Cross-platform event bridge for the Aether control plane.
 *
 * This module lets TypeScript read events written by the Go runtime
 * (and vice-versa) via the shared NDJSON event stream at
 * `.aether/events/current.ndjson`.
 *
 * In stub mode, file I/O is simulated with in-memory buffers so that
 * both sides can demonstrate concurrent read/write without a real
 * cross-process mechanism.
 */

import { EventLine, EventType, createEventLine } from "./types.js";
import { writeEvent, getBuffer, clearBuffer } from "./writeEvent.js";
import { readEvents, tailEvents } from "./readEvents.js";

/**
 * Synchronise with the Go event stream by watching for new events.
 *
 * In stub mode this sets up an in-memory listener; in production it
 * would tail `.aether/events/current.ndjson`.
 */
export async function syncEventsWithGo(
  eventStreamPath: string,
  onEvent: (event: EventLine) => void | Promise<void>
): Promise<() => void> {
  const abortController = new AbortController();
  const { signal } = abortController;

  // Start tailing the stream in the background.
  (async () => {
    try {
      for await (const ev of tailEvents(eventStreamPath, {
        pollIntervalMs: 100,
      })) {
        if (signal.aborted) break;
        await onEvent(ev);
      }
    } catch {
      // Tail stops when aborted.
    }
  })();

  // Return a dispose function.
  return () => {
    abortController.abort();
  };
}

/**
 * Read events that were written by the Go side.
 *
 * Yields every event currently in the shared stream, then continues
 * yielding new events as they arrive.
 */
export async function* readGoEvents(
  eventStreamPath: string
): AsyncGenerator<EventLine> {
  // First yield everything that already exists.
  yield* readEvents(eventStreamPath);

  // Then tail for new events, starting from where we left off.
  const buffer = getBuffer(eventStreamPath);
  yield* tailEvents(eventStreamPath, { pollIntervalMs: 100, startFrom: buffer.length });
}

/**
 * Demonstrate cross-platform event compatibility.
 *
 * 1. Writes a TS-origin event.
 * 2. Reads back all events (including any written by Go).
 * 3. Confirms that every line has the agreed NDJSON shape.
 */
export async function demonstrateBridgeCompatibility(
  eventStreamPath: string
): Promise<{
  tsEventWritten: EventLine;
  allEvents: EventLine[];
  formatOk: boolean;
}> {
  // 1. Write a TS event.
  const tsEvent = createEventLine("custom", {
    source: "typescript-bridge",
    message: "Hello from TS",
  });
  await writeEvent(eventStreamPath, tsEvent);

  // 2. Read all events from the shared buffer.
  const allEvents: EventLine[] = [];
  for await (const ev of readEvents(eventStreamPath)) {
    allEvents.push(ev);
  }

  // 3. Verify format compatibility.
  const formatOk = allEvents.every(
    (ev) =>
      typeof ev.type === "string" &&
      typeof ev.timestamp === "string" &&
      ev.payload !== undefined &&
      ev.payload !== null
  );

  return { tsEventWritten: tsEvent, allEvents, formatOk };
}

/**
 * Reset the in-memory bridge state for a given stream path.
 * Useful in tests to guarantee isolation between cases.
 */
export function resetBridge(eventStreamPath: string): void {
  clearBuffer(eventStreamPath);
}
