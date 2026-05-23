import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { parse } from "yaml";
import { PolicySchema } from "../../src/schemas/policy.schema.js";
import { fixturePath } from "../helpers.js";

describe("PolicySchema", () => {
  it("validates model-routing.yaml", () => {
    const text = readFileSync(fixturePath("policies", "model-routing.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.model_routing.default_provider).toBe("anthropic");
    expect(result.memory_rules.max_learnings).toBe(100);
    expect(result.skill_creation.allowed).toBe(true);
    expect(result.safety_gates.security_scan).toBe(true);
  });

  it("provides defaults for empty object", () => {
    const result = PolicySchema.parse({});
    expect(result.model_routing).toBeDefined();
    expect(result.memory_rules).toBeDefined();
    expect(result.skill_creation).toBeDefined();
    expect(result.safety_gates).toBeDefined();
    expect(result.memory_rules.max_learnings).toBe(100);
    expect(result.skill_creation.allowed).toBe(true);
    expect(result.safety_gates.security_scan).toBe(true);
  });

  it("rejects empty provider in routing rule", () => {
    expect(() =>
      PolicySchema.parse({
        model_routing: {
          routing_rules: [{ agent_role: "x", provider: "" }],
        },
      })
    ).toThrow();
  });
});
