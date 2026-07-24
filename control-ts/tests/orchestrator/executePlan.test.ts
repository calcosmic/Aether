import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import {
  readFileSync,
  existsSync,
  unlinkSync,
  mkdirSync,
  rmdirSync,
  mkdtempSync,
} from "fs";
import { resolve, dirname } from "path";
import { tmpdir } from "os";
import {
  executePlan,
  type PlanResult,
} from "../../src/orchestrator/executePlan.js";
import { parseEventLine, type ColonyEvent } from "../../src/schemas/event.schema.js";
import {
  readColonyState,
  writeColonyState,
  updateColonyState,
} from "../../src/memory/store.js";

function cleanupState(): void {
  const statePath = process.env.AETHER_CONTROL_STATE_PATH;
  if (statePath && existsSync(statePath)) {
    unlinkSync(statePath);
  }
}

function readEvents(): ColonyEvent[] {
  const eventsFile = process.env.AETHER_EVENTS_FILE;
  if (!eventsFile || !existsSync(eventsFile)) return [];
  const raw = readFileSync(eventsFile, "utf8");
  return raw
    .trim()
    .split("\n")
    .filter((l) => l.trim())
    .map((line) => parseEventLine(line));
}

describe("executePlan", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(resolve(tmpdir(), "aether-events-"));
    process.env.AETHER_EVENTS_FILE = resolve(tempDir, "events.ndjson");
    process.env.AETHER_CONTROL_STATE_PATH = resolve(tempDir, "COLONY_STATE.json");
    cleanupState();
    // Seed a fresh colony state
    const initial = readColonyState();
    initial.goal = "Test colony";
    writeColonyState(initial);
  });

  afterEach(() => {
    if (tempDir && existsSync(tempDir)) {
      const files = require("fs").readdirSync(tempDir);
      for (const f of files) {
        unlinkSync(resolve(tempDir, f));
      }
      rmdirSync(tempDir);
    }
    delete process.env.AETHER_EVENTS_FILE;
    delete process.env.AETHER_CONTROL_STATE_PATH;
    cleanupState();
    vi.restoreAllMocks();
  });

  it("fails on the first phase instead of simulating a completed plan", async () => {
    const result = await executePlan(["init", "plan", "build"]);
    expect(result.status).toBe("failed");
    expect(result.results.length).toBe(1);
    expect(result.results[0].phaseId).toBe("init");
    expect(result.results[0].status).toBe("failed");
  });

  it("emits plan:start and plan:complete events to NDJSON", async () => {
    const result = await executePlan(["init", "plan"]);
    const events = readEvents();
    const startEvent = events.find((e) => e.type === "plan:start");
    const completeEvent = events.find((e) => e.type === "plan:complete");

    expect(startEvent).toBeDefined();
    expect(startEvent!.payload.sequence).toEqual(["init", "plan"]);
    expect(startEvent!.payload.startedAt).toBeDefined();

    expect(completeEvent).toBeDefined();
    expect(completeEvent!.payload.status).toBe("failed");
    expect(completeEvent!.payload.phaseCount).toBe(1);
    expect(completeEvent!.payload.completedAt).toBeDefined();
  });

  it("stops execution when a phase fails with failure_policy 'block'", async () => {
    const mockRunPhase = vi.fn()
      .mockResolvedValueOnce({
        phaseId: "init",
        agentId: "queen",
        status: "completed",
        events: [],
      })
      .mockResolvedValueOnce({
        phaseId: "plan",
        agentId: "scout",
        status: "failed",
        events: [],
        error: "Plan generation failed",
      });

    vi.doMock("../../src/orchestrator/runPhase.js", () => ({
      runPhase: mockRunPhase,
    }));

    vi.doMock("../../src/phases/loadPhases.js", async () => {
      const actual = await vi.importActual<typeof import("../../src/phases/loadPhases.js")>(
        "../../src/phases/loadPhases.js"
      );
      return {
        ...actual,
        loadPhaseById: (id: string) => {
          if (id === "plan") {
            return {
              id: "plan",
              entry_agent: "scout",
              required_agents: ["scout"],
              inputs: {},
              outputs: [],
              success_criteria: ["Plan generated"],
              failure_policy: "block",
            };
          }
          return actual.loadPhaseById(id);
        },
      };
    });

    const { executePlan: executePlanMocked } = await import(
      "../../src/orchestrator/executePlan.js?block=1"
    );

    const result = await executePlanMocked(["init", "plan", "build"]);
    expect(result.status).toBe("failed");
    expect(result.results.length).toBe(2);
    expect(result.results[1].status).toBe("failed");
  });

  it("continues to next phase when a phase fails with failure_policy 'skip'", async () => {
    const mockRunPhase = vi.fn()
      .mockResolvedValueOnce({
        phaseId: "init",
        agentId: "queen",
        status: "completed",
        events: [],
      })
      .mockResolvedValueOnce({
        phaseId: "plan",
        agentId: "scout",
        status: "failed",
        events: [],
        error: "Plan generation failed",
      })
      .mockResolvedValueOnce({
        phaseId: "build",
        agentId: "queen",
        status: "completed",
        events: [],
      });

    vi.doMock("../../src/orchestrator/runPhase.js", () => ({
      runPhase: mockRunPhase,
    }));

    vi.doMock("../../src/phases/loadPhases.js", async () => {
      const actual = await vi.importActual<typeof import("../../src/phases/loadPhases.js")>(
        "../../src/phases/loadPhases.js"
      );
      return {
        ...actual,
        loadPhaseById: (id: string) => {
          if (id === "plan") {
            return {
              id: "plan",
              entry_agent: "scout",
              required_agents: ["scout"],
              inputs: {},
              outputs: [],
              success_criteria: ["Plan generated"],
              failure_policy: "skip",
            };
          }
          return actual.loadPhaseById(id);
        },
      };
    });

    const { executePlan: executePlanMocked } = await import(
      "../../src/orchestrator/executePlan.js?skip=1"
    );

    const result = await executePlanMocked(["init", "plan", "build"]);
    expect(result.status).toBe("partial");
    expect(result.results.length).toBe(3);
    expect(result.results[1].status).toBe("failed");
    expect(result.results[2].status).toBe("completed");
  });

  it("retries a failed phase once with failure_policy 'retry' then treats as block", async () => {
    const mockRunPhase = vi.fn()
      .mockResolvedValueOnce({
        phaseId: "init",
        agentId: "queen",
        status: "completed",
        events: [],
      })
      .mockResolvedValueOnce({
        phaseId: "build",
        agentId: "queen",
        status: "failed",
        events: [],
        error: "Build failed first time",
      })
      .mockResolvedValueOnce({
        phaseId: "build",
        agentId: "queen",
        status: "failed",
        events: [],
        error: "Build failed second time",
      });

    vi.doMock("../../src/orchestrator/runPhase.js", () => ({
      runPhase: mockRunPhase,
    }));

    const { executePlan: executePlanMocked } = await import(
      "../../src/orchestrator/executePlan.js?retry=1"
    );

    const result = await executePlanMocked(["init", "build"]);
    expect(result.status).toBe("failed");
    expect(result.results.length).toBe(3); // init + build (first) + build (retry)
    expect(mockRunPhase).toHaveBeenCalledTimes(3);
  });

  it("leaves current_phase unchanged and records failed state", async () => {
    await executePlan(["init", "plan"]);
    const state = readColonyState();
    expect(state.state).toBe("FAILED");
    expect(state.current_phase).toBe(0);
  });

  it("returns immediately with completed status for an empty sequence", async () => {
    const result = await executePlan([]);
    expect(result.status).toBe("completed");
    expect(result.results).toEqual([]);
    expect(result.events.length).toBeGreaterThanOrEqual(2);
  });

  it("passes per-phase inputs from options to runPhase", async () => {
    const mockRunPhase = vi.fn().mockResolvedValue({
      phaseId: "init",
      agentId: "queen",
      status: "completed",
      events: [],
    });

    vi.doMock("../../src/orchestrator/runPhase.js", () => ({
      runPhase: mockRunPhase,
    }));

    const { executePlan: executePlanMocked } = await import(
      "../../src/orchestrator/executePlan.js?inputs=1"
    );

    await executePlanMocked(["init"], {
      inputs: { init: { goal: "Custom goal" } },
    });

    expect(mockRunPhase).toHaveBeenCalledWith("init", { goal: "Custom goal" });
  });
});
