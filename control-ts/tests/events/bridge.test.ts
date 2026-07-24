import { describe, it, expect, beforeEach } from "vitest";
import {
  createEventLine,
  writeEvent,
  readEvents,
  tailEvents,
  getBuffer,
  clearBuffer,
} from "../../src/events/index.js";
import {
  syncEventsWithGo,
  readGoEvents,
  demonstrateBridgeCompatibility,
  resetBridge,
} from "../../src/events/bridge.js";

describe("cross-platform event bridge", () => {
  const streamPath = "/tmp/bridge-test.ndjson";

  beforeEach(() => {
    resetBridge(streamPath);
  });

  describe("syncEventsWithGo", () => {
    it("receives events written by the Go side (simulated)", async () => {
      const received: ReturnType<typeof createEventLine>[] = [];

      const stop = await syncEventsWithGo(streamPath, async (ev) => {
        received.push(ev);
      });

      // Simulate a Go-side write.
      const goEvent = createEventLine("phase_start", {
        phase: "1",
        plan: "01",
        goal: "test",
      });
      await writeEvent(streamPath, goEvent);

      // Wait for polling interval.
      await new Promise((r) => setTimeout(r, 150));
      stop();

      expect(received.length).toBeGreaterThanOrEqual(1);
      expect(received[received.length - 1].type).toBe("phase_start");
    });
  });

  describe("readGoEvents", () => {
    it("yields existing events then new ones", async () => {
      const existing = createEventLine("agent_selected", {
        agentId: "builder-1",
        caste: "builder",
        task: "impl",
      });
      await writeEvent(streamPath, existing);

      const collected: ReturnType<typeof createEventLine>[] = [];
      const iterator = readGoEvents(streamPath);

      // Consume with a timeout so the infinite tail does not hang us.
      const consume = (async () => {
        for await (const ev of iterator) {
          collected.push(ev);
          if (collected.length >= 2) break;
        }
      })();

      // Write a second event after a short delay.
      await new Promise((r) => setTimeout(r, 50));
      const next = createEventLine("worker_spawned", {
        workerId: "w1",
        caste: "watcher",
      });
      await writeEvent(streamPath, next);

      await consume;
      expect(collected).toHaveLength(2);
      expect(collected[0].type).toBe("agent_selected");
      expect(collected[1].type).toBe("worker_spawned");
    });
  });

  describe("demonstrateBridgeCompatibility", () => {
    it("writes a TS event and confirms format compatibility", async () => {
      const result = await demonstrateBridgeCompatibility(streamPath);

      expect(result.tsEventWritten.type).toBe("custom");
      expect(result.tsEventWritten.payload).toEqual({
        source: "typescript-bridge",
        message: "Hello from TS",
      });
      expect(result.allEvents.length).toBeGreaterThanOrEqual(1);
      expect(result.formatOk).toBe(true);
    });

    it("sees events written by Go (simulated)", async () => {
      // Simulate a Go-side write.
      const goEvent = createEventLine("memory_update", {
        key: "k1",
        value: 42,
        source: "go-bridge",
      });
      await writeEvent(streamPath, goEvent);

      const result = await demonstrateBridgeCompatibility(streamPath);

      // The TS event was appended after the Go event.
      expect(result.allEvents.length).toBeGreaterThanOrEqual(2);
      const types = result.allEvents.map((e) => e.type);
      expect(types).toContain("memory_update");
      expect(types).toContain("custom");
      expect(result.formatOk).toBe(true);
    });
  });

  describe("concurrent access", () => {
    it("handles interleaved writes from both sides", async () => {
      const events = Array.from({ length: 10 }, (_, i) =>
        createEventLine(i % 2 === 0 ? "tool_call" : "verification_result", {
          seq: i,
          side: i % 2 === 0 ? "go" : "ts",
        })
      );

      // Write all events concurrently.
      await Promise.all(events.map((ev) => writeEvent(streamPath, ev)));

      const collected: ReturnType<typeof createEventLine>[] = [];
      for await (const ev of readEvents(streamPath)) {
        collected.push(ev);
      }

      expect(collected).toHaveLength(10);
      // Ordering is preserved because writes are sequential in the buffer.
      for (let i = 0; i < 10; i++) {
        expect(collected[i].payload).toEqual({ seq: i, side: i % 2 === 0 ? "go" : "ts" });
      }
    });
  });

  describe("event ordering", () => {
    it("preserves timestamp order for sequential writes", async () => {
      const a = createEventLine("custom", { note: "a" });
      await new Promise((r) => setTimeout(r, 10));
      const b = createEventLine("custom", { note: "b" });
      await new Promise((r) => setTimeout(r, 10));
      const c = createEventLine("custom", { note: "c" });

      await writeEvent(streamPath, a);
      await writeEvent(streamPath, b);
      await writeEvent(streamPath, c);

      const collected: ReturnType<typeof createEventLine>[] = [];
      for await (const ev of readEvents(streamPath)) {
        collected.push(ev);
      }

      expect(collected).toHaveLength(3);
      const timestamps = collected.map((e) => new Date(e.timestamp).getTime());
      expect(timestamps[0]).toBeLessThanOrEqual(timestamps[1]);
      expect(timestamps[1]).toBeLessThanOrEqual(timestamps[2]);
    });
  });

  describe("format compatibility", () => {
    it("produces identical JSON shape from both sides", async () => {
      const tsEvent = createEventLine("phase_start", { phase: "1" });
      await writeEvent(streamPath, tsEvent);

      const goEvent = createEventLine("phase_start", { phase: "1" });
      await writeEvent(streamPath, goEvent);

      const buffer = getBuffer(streamPath);
      expect(buffer).toHaveLength(2);

      // Both events should have the exact same keys.
      const keysA = Object.keys(buffer[0]).sort();
      const keysB = Object.keys(buffer[1]).sort();
      expect(keysA).toEqual(keysB);

      // Both should be valid NDJSON lines (no newlines inside).
      const ndjsonA = JSON.stringify(buffer[0]);
      const ndjsonB = JSON.stringify(buffer[1]);
      expect(ndjsonA).not.toContain("\n");
      expect(ndjsonB).not.toContain("\n");
    });
  });
});
