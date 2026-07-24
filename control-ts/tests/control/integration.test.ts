import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  readFileSync,
  existsSync,
  unlinkSync,
  rmdirSync,
  mkdtempSync,
} from "fs";
import { resolve } from "path";
import { tmpdir } from "os";
import { executePlan } from "../../src/orchestrator/executePlan.js";
import { runPhase } from "../../src/orchestrator/runPhase.js";
import { parseEventLine, type ColonyEvent } from "../../src/schemas/event.schema.js";
import {
  readColonyState,
  writeColonyState,
} from "../../src/memory/store.js";

function cleanupState(): void {
  const statePath = process.env.AETHER_CONTROL_STATE_PATH;
  if (statePath && existsSync(statePath)) {
    unlinkSync(statePath);
  }
}

function getEventsFile(): string {
  return process.env.AETHER_EVENTS_FILE || "";
}

function cleanupEvents(): void {
  const f = getEventsFile();
  if (existsSync(f)) {
    unlinkSync(f);
  }
}

function readEvents(): ColonyEvent[] {
  const eventsFile = getEventsFile();
  if (!eventsFile || !existsSync(eventsFile)) return [];
  const raw = readFileSync(eventsFile, "utf8");
  return raw
    .trim()
    .split("\n")
    .filter((l) => l.trim())
    .map((line) => parseEventLine(line));
}

describe("control plane integration", () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(resolve(tmpdir(), "aether-events-"));
    process.env.AETHER_EVENTS_FILE = resolve(tempDir, "events.ndjson");
    process.env.AETHER_CONTROL_STATE_PATH = resolve(tempDir, "COLONY_STATE.json");
    cleanupState();
    const initial = readColonyState();
    initial.goal = "Integration test colony";
    writeColonyState(initial);
  });

  afterEach(() => {
    cleanupEvents();
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
  });

  it("fails closed on init and emits no false phase completions", async () => {
    const result = await executePlan(["init", "plan", "build"]);
    expect(result.status).toBe("failed");
    expect(result.results.length).toBe(1);

    const events = readEvents();
    const phaseStarts = events.filter((e) => e.type === "phase:start");
    const phaseFailures = events.filter((e) => e.type === "phase:failed");
    const phaseCompletes = events.filter((e) => e.type === "phase:complete");
    expect(phaseStarts.length).toBe(1);
    expect(phaseFailures.length).toBe(1);
    expect(phaseCompletes.length).toBe(0);

    const planStart = events.find((e) => e.type === "plan:start");
    const planComplete = events.find((e) => e.type === "plan:complete");
    expect(planStart).toBeDefined();
    expect(planComplete).toBeDefined();
    expect(planComplete!.payload.status).toBe("failed");
  });

  it("runPhase loads agent and phase from colony assets", async () => {
    const result = await runPhase("init");
    expect(result.phaseId).toBe("init");
    expect(result.agentId).toBe("queen");
    expect(result.status).toBe("failed");
  });

  it("colony state records failure without phase advancement", async () => {
    await executePlan(["init", "plan"]);
    const state = readColonyState();
    expect(state.state).toBe("FAILED");
    expect(state.current_phase).toBe(0);
  });

  it("NDJSON events are parseable and contain required fields", async () => {
    await executePlan(["init"]);
    const eventsFile = getEventsFile();
    expect(existsSync(eventsFile)).toBe(true);

    const raw = readFileSync(eventsFile, "utf8");
    const lines = raw.trim().split("\n").filter((l) => l.trim());
    expect(lines.length).toBeGreaterThanOrEqual(2);

    for (const line of lines) {
      const ev = parseEventLine(line);
      expect(ev.type).toBeDefined();
      expect(ev.timestamp).toBeDefined();
      expect(ev.payload).toBeDefined();
    }
  });
});
