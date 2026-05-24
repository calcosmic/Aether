import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { parse } from "yaml";
import { PhaseSchema } from "../../src/schemas/phase.schema.js";
import { fixturePath } from "../helpers.js";

describe("PhaseSchema", () => {
  it("validates init.yaml fixture", () => {
    const text = readFileSync(fixturePath("phases", "init.yaml"), "utf8");
    const data = parse(text);
    const result = PhaseSchema.parse(data);
    expect(result.id).toBe("init");
    expect(result.entry_agent).toBe("queen");
    expect(result.required_agents).toContain("queen");
    expect(result.failure_policy).toBe("block");
    expect(result.ceremony?.stages).toEqual(["setup", "validate", "commit"]);
  });

  it("validates plan.yaml fixture", () => {
    const text = readFileSync(fixturePath("phases", "plan.yaml"), "utf8");
    const data = parse(text);
    const result = PhaseSchema.parse(data);
    expect(result.id).toBe("plan");
    expect(result.entry_agent).toBe("scout");
    expect(result.required_agents).toContain("scout");
    expect(result.failure_policy).toBe("retry");
    expect(result.ceremony?.stages).toEqual(["research", "draft", "review"]);
  });

  it("rejects missing required_agents", () => {
    expect(() =>
      PhaseSchema.parse({
        id: "x",
        entry_agent: "y",
        success_criteria: ["a"],
      })
    ).toThrow();
  });

  it("rejects missing success_criteria", () => {
    expect(() =>
      PhaseSchema.parse({
        id: "x",
        entry_agent: "y",
        required_agents: ["a"],
      })
    ).toThrow();
  });

  it("validates all 9 phase fixtures", () => {
    const phases = [
      "init",
      "discuss",
      "plan",
      "build",
      "continue",
      "seal",
      "colonize",
      "oracle",
      "swarm",
    ];
    for (const name of phases) {
      const text = readFileSync(fixturePath("phases", `${name}.yaml`), "utf8");
      const data = parse(text);
      const result = PhaseSchema.parse(data);
      expect(result.id).toBe(name);
      expect(result.required_agents.length).toBeGreaterThanOrEqual(1);
      expect(result.success_criteria.length).toBeGreaterThanOrEqual(1);
    }
  });
});
