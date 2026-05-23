import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { parse } from "yaml";
import { AgentSchema } from "../../src/schemas/agent.schema.js";
import { fixturePath } from "../helpers.js";

describe("AgentSchema", () => {
  it("validates queen.yaml fixture", () => {
    const text = readFileSync(fixturePath("agents", "queen.yaml"), "utf8");
    const data = parse(text);
    const result = AgentSchema.parse(data);
    expect(result.id).toBe("queen");
    expect(result.role).toBe("queen");
    expect(result.prompt_file).toBe("colony/prompts/queen.md");
    expect(result.allowed_tools).toContain("spawn");
  });

  it("validates builder.yaml fixture", () => {
    const text = readFileSync(fixturePath("agents", "builder.yaml"), "utf8");
    const data = parse(text);
    const result = AgentSchema.parse(data);
    expect(result.id).toBe("builder");
    expect(result.role).toBe("builder");
    expect(result.prompt_file).toBe("colony/prompts/builder.md");
    expect(result.allowed_tools).toContain("write");
  });

  it("rejects nonexistent prompt_file", () => {
    expect(() =>
      AgentSchema.parse({
        id: "x",
        role: "builder",
        prompt_file: "nonexistent.md",
      })
    ).toThrow();
  });

  it("rejects empty id", () => {
    expect(() =>
      AgentSchema.parse({
        id: "",
        role: "queen",
        prompt_file: "colony/prompts/queen.md",
      })
    ).toThrow();
  });

  it("rejects path traversal in prompt_file", () => {
    expect(() =>
      AgentSchema.parse({
        id: "bad",
        role: "builder",
        prompt_file: "../../../etc/passwd",
      })
    ).toThrow();
  });
});
