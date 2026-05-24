import { describe, it, expect } from "vitest";
import { loadAgents, loadAgentById } from "../../src/agents/loadAgents.js";

describe("loadAgents", () => {
  it("returns an array of 27 Agent objects", () => {
    const agents = loadAgents();
    expect(agents).toHaveLength(27);
    for (const agent of agents) {
      expect(agent).toHaveProperty("id");
      expect(agent).toHaveProperty("role");
      expect(agent).toHaveProperty("prompt_file");
    }
  });

  it("sorts agents by id for determinism", () => {
    const agents = loadAgents();
    const ids = agents.map((a) => a.id);
    const sortedIds = [...ids].sort((a, b) => a.localeCompare(b));
    expect(ids).toEqual(sortedIds);
  });

  it("validates all files against AgentSchema", () => {
    // If any file is invalid, loadAgents throws.
    expect(() => loadAgents()).not.toThrow();
  });
});

describe("loadAgentById", () => {
  it("returns the queen agent", () => {
    const agent = loadAgentById("queen");
    expect(agent).not.toBeNull();
    expect(agent!.id).toBe("queen");
    expect(agent!.role).toBe("queen");
  });

  it("returns null for nonexistent id", () => {
    const agent = loadAgentById("nonexistent");
    expect(agent).toBeNull();
  });
});
