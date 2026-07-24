import { describe, it, expect } from "vitest";
import { loadPhases, loadPhaseById } from "../../src/phases/loadPhases.js";

describe("loadPhases", () => {
  it("returns an array of 9 Phase objects", () => {
    const phases = loadPhases();
    expect(phases).toHaveLength(9);
    for (const phase of phases) {
      expect(phase).toHaveProperty("id");
      expect(phase).toHaveProperty("entry_agent");
      expect(phase).toHaveProperty("required_agents");
      expect(phase).toHaveProperty("success_criteria");
    }
  });

  it("sorts phases by id for determinism", () => {
    const phases = loadPhases();
    const ids = phases.map((p) => p.id);
    const sortedIds = [...ids].sort((a, b) => a.localeCompare(b));
    expect(ids).toEqual(sortedIds);
  });

  it("validates all files against PhaseSchema", () => {
    expect(() => loadPhases()).not.toThrow();
  });
});

describe("loadPhaseById", () => {
  it("returns the init phase", () => {
    const phase = loadPhaseById("init");
    expect(phase).not.toBeNull();
    expect(phase!.id).toBe("init");
    expect(phase!.entry_agent).toBe("queen");
  });

  it("returns null for nonexistent id", () => {
    const phase = loadPhaseById("nonexistent");
    expect(phase).toBeNull();
  });
});
