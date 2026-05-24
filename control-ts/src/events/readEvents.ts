/**
 * Event reader stub for the Aether control plane event stream.
 *
 * In stub mode, reads come from the in-memory buffer maintained by
 * `writeEvent.ts`. No actual file I/O occurs.
 */

import { EventLine } from "./types.js";
import { getBuffer } from "./writeEvent.js";

/**
 * Read all events from the in-memory stream for a given path.
 * Yields parsed EventLine objects.
 */
export async function* readEvents(
  streamPath: string
): AsyncGenerator<EventLine> {
  const buffer = getBuffer(streamPath);
  for (const line of buffer) {
    yield line;
  }
}

/**
 * Tail the event stream, yielding new events as they arrive.
 * In stub mode this polls the in-memory buffer at a fixed interval.
 */
export async function* tailEvents(
  streamPath: string,
  options: { pollIntervalMs?: number } = {}
): AsyncGenerator<EventLine> {
  const { pollIntervalMs = 100 } = options;
  let seen = 0;

  while (true) {
    const buffer = getBuffer(streamPath);
    while (seen < buffer.length) {
      yield buffer[seen];
      seen++;
    }
    await sleep(pollIntervalMs);
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
