import { describe, it, expect, beforeEach } from "vitest";
import {
  createEventLine,
  writeEvent,
  createEventStream,
  readEvents,
  tailEvents,
  getBuffer,
  clearBuffer,
} from "../../src/events/index.js";

describe("event stream", () => {
  const streamPath = "/tmp/test-events.ndjson";

  beforeEach(() => {
    clearBuffer(streamPath);
  });

  describe("createEventLine", () => {
    it("produces an event line with type, timestamp, and payload", () => {
      const line = createEventLine("phase_start", { phase: "1", plan: "01" });
      expect(line.type).toBe("phase_start");
      expect(line.timestamp).toBeDefined();
      expect(new Date(line.timestamp).toISOString()).toBe(line.timestamp);
      expect(line.payload).toEqual({ phase: "1", plan: "01" });
    });

    it("produces distinct timestamps for successive calls", async () => {
      const a = createEventLine("custom", { note: "a" });
      await new Promise((r) => setTimeout(r, 5));
      const b = createEventLine("custom", { note: "b" });
      expect(new Date(a.timestamp).getTime()).toBeLessThan(
        new Date(b.timestamp).getTime()
      );
    });
  });

  describe("writeEvent", () => {
    it("returns successfully and stores the event", async () => {
      const line = createEventLine("agent_selected", {
        agentId: "builder-1",
        caste: "builder",
        task: "impl",
      });
      await writeEvent(streamPath, line);
      const buffer = getBuffer(streamPath);
      expect(buffer).toHaveLength(1);
      expect(buffer[0]).toEqual(line);
    });
  });

  describe("createEventStream", () => {
    it("writes multiple events and closes", async () => {
      const stream = createEventStream(streamPath);
      await stream.write(
        createEventLine("task_planned", {
          taskId: "t1",
          description: "do thing",
          dependencies: [],
        })
      );
      await stream.write(
        createEventLine("worker_spawned", {
          workerId: "w1",
          caste: "watcher",
        })
      );
      await stream.close();
      expect(getBuffer(streamPath)).toHaveLength(2);
    });
  });

  describe("readEvents", () => {
    it("yields parsed events from the buffer", async () => {
      const line = createEventLine("tool_call", {
        tool: "read",
        args: { path: "/tmp" },
      });
      await writeEvent(streamPath, line);
      const collected: Awaited<ReturnType<typeof readEvents>>[] = [];
      for await (const ev of readEvents(streamPath)) {
        collected.push(ev);
      }
      expect(collected).toHaveLength(1);
      expect(collected[0]).toEqual(line);
    });

    it("yields nothing when buffer is empty", async () => {
      const collected: Awaited<ReturnType<typeof readEvents>>[] = [];
      for await (const ev of readEvents(streamPath)) {
        collected.push(ev);
      }
      expect(collected).toHaveLength(0);
    });
  });

  describe("tailEvents", () => {
    it("yields new events as they arrive", async () => {
      const line = createEventLine("verification_result", {
        taskId: "t1",
        passed: true,
      });

      const collected: Awaited<ReturnType<typeof tailEvents>>[] = [];
      const iterator = tailEvents(streamPath, { pollIntervalMs: 10 });

      // Start consuming
      const consume = (async () => {
        for await (const ev of iterator) {
          collected.push(ev);
          if (collected.length >= 1) break;
        }
      })();

      // Write after a short delay
      await new Promise((r) => setTimeout(r, 20));
      await writeEvent(streamPath, line);

      await consume;
      expect(collected).toHaveLength(1);
      expect(collected[0]).toEqual(line);
    });
  });

  describe("round-trip", () => {
    it("write then read returns the same events", async () => {
      const lines = [
        createEventLine("phase_start", { phase: "1", plan: "01" }),
        createEventLine("memory_update", {
          key: "k1",
          value: "v1",
          source: "test",
        }),
        createEventLine("run_seal", { phase: "1", plan: "01", summary: "ok" }),
      ];

      for (const line of lines) {
        await writeEvent(streamPath, line);
      }

      const collected: Awaited<ReturnType<typeof readEvents>>[] = [];
      for await (const ev of readEvents(streamPath)) {
        collected.push(ev);
      }

      expect(collected).toHaveLength(lines.length);
      expect(collected).toEqual(lines);
    });
  });

  describe("NDJSON shape", () => {
    it("serializes to one JSON object per line with no trailing comma", () => {
      const line = createEventLine("custom", { hello: "world" });
      const ndjson = JSON.stringify(line);
      expect(ndjson).not.toContain("\n");
      expect(ndjson.endsWith(",")).toBe(false);
      const parsed = JSON.parse(ndjson);
      expect(parsed).toEqual(line);
    });
  });
});
