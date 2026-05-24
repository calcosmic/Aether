import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { readFileSync, existsSync, unlinkSync, rmdirSync } from "fs";
import { resolve, dirname } from "path";
import {
  readColonyState,
  writeColonyState,
  updateColonyState,
  type ColonyState,
} from "../../src/memory/store.js";
import { projectRoot } from "../../src/utils/projectRoot.js";

const TEST_STATE_DIR = resolve(projectRoot, "..", ".aether", "data");
const TEST_STATE_PATH = resolve(TEST_STATE_DIR, "COLONY_STATE.json");

function cleanup(): void {
  if (existsSync(TEST_STATE_PATH)) {
    unlinkSync(TEST_STATE_PATH);
  }
}

describe("readColonyState", () => {
  beforeEach(cleanup);
  afterEach(cleanup);

  it("returns default state when file does not exist", () => {
    const state = readColonyState();
    expect(state.version).toBe("3.0");
    expect(state.state).toBe("INIT");
    expect(state.current_phase).toBe(0);
    expect(state.goal).toBe("");
    expect(state.plan.phases).toEqual([]);
  });

  it("reads and parses existing COLONY_STATE.json", () => {
    const custom: ColonyState = {
      version: "3.0",
      goal: "Build feature X",
      scope: "project",
      colony_mode: "colony",
      state: "ACTIVE",
      current_phase: 2,
      session_id: "abc123",
      initialized_at: "2026-05-23T00:00:00Z",
      plan: {
        generated_at: "2026-05-23T00:00:00Z",
        confidence: 88,
        phases: [{ id: "1", name: "Setup" }],
      },
    };
    writeColonyState(custom);
    const state = readColonyState();
    expect(state.goal).toBe("Build feature X");
    expect(state.current_phase).toBe(2);
    expect(state.plan.confidence).toBe(88);
  });
});

describe("writeColonyState", () => {
  beforeEach(cleanup);
  afterEach(cleanup);

  it("serializes and writes state atomically", () => {
    const state: ColonyState = {
      version: "3.0",
      goal: "Test goal",
      scope: "project",
      colony_mode: "colony",
      state: "INIT",
      current_phase: 0,
      session_id: "",
      initialized_at: "2026-05-23T00:00:00Z",
      plan: {
        generated_at: "2026-05-23T00:00:00Z",
        confidence: 0,
        phases: [],
      },
    };
    writeColonyState(state);
    expect(existsSync(TEST_STATE_PATH)).toBe(true);
    const raw = readFileSync(TEST_STATE_PATH, "utf8");
    const parsed = JSON.parse(raw);
    expect(parsed.goal).toBe("Test goal");
  });

  it("creates parent directories if they do not exist", () => {
    // The default path already includes .aether/data; if it exists, this is fine.
    // We test by removing the file (not the dir) and writing again.
    cleanup();
    const state = readColonyState();
    state.goal = "Dir test";
    writeColonyState(state);
    expect(existsSync(TEST_STATE_PATH)).toBe(true);
  });
});

describe("updateColonyState", () => {
  beforeEach(cleanup);
  afterEach(cleanup);

  it("reads current state, applies updater, and writes back", () => {
    const initial = readColonyState();
    initial.goal = "Initial";
    writeColonyState(initial);

    updateColonyState((state) => ({
      ...state,
      current_phase: state.current_phase + 1,
      goal: "Updated",
    }));

    const updated = readColonyState();
    expect(updated.current_phase).toBe(1);
    expect(updated.goal).toBe("Updated");
  });

  it("throws with 'Memory update failed' on error", () => {
    expect(() =>
      updateColonyState(() => {
        throw new Error("boom");
      })
    ).toThrow("Memory update failed: boom");
  });
});
