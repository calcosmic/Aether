import { describe, it, expect } from "vitest";
import {
  assemblePrompt,
  assemblePromptForAgent,
} from "../../src/prompts/assemblePrompt.js";
import { AgentSchema } from "../../src/schemas/agent.schema.js";

describe("assemblePrompt", () => {
  it("reads builder.md and returns non-empty string", () => {
    const prompt = assemblePrompt("builder");
    expect(typeof prompt).toBe("string");
    expect(prompt.length).toBeGreaterThan(0);
    expect(prompt).toContain("Builder Ant");
  });

  it("reads queen.md and returns non-empty string", () => {
    const prompt = assemblePrompt("queen");
    expect(typeof prompt).toBe("string");
    expect(prompt.length).toBeGreaterThan(0);
    expect(prompt).toContain("Queen");
  });

  it("throws for nonexistent role", () => {
    expect(() => assemblePrompt("nonexistent")).toThrow(
      /Prompt file not found/
    );
  });

  it("injects skills and pheromones when provided", () => {
    const prompt = assemblePrompt("builder", {
      skills: ["tdd", "refactoring"],
      pheromones: ["focus: security", "redirect: no eval"],
    });
    expect(prompt).toContain("## Injected Skills");
    expect(prompt).toContain("- tdd");
    expect(prompt).toContain("- refactoring");
    expect(prompt).toContain("## Active Pheromones");
    expect(prompt).toContain("- focus: security");
    expect(prompt).toContain("- redirect: no eval");
  });

  it("does not inject sections when context is empty", () => {
    const prompt = assemblePrompt("builder");
    expect(prompt).not.toContain("## Injected Skills");
    expect(prompt).not.toContain("## Active Pheromones");
  });
});

describe("assemblePromptForAgent", () => {
  it("reads the agent's prompt_file and returns the prompt text", () => {
    const agent = AgentSchema.parse({
      id: "builder",
      role: "builder",
      prompt_file: "colony/prompts/builder.md",
    });
    const prompt = assemblePromptForAgent(agent);
    expect(typeof prompt).toBe("string");
    expect(prompt.length).toBeGreaterThan(0);
    expect(prompt).toContain("Builder Ant");
  });

  it("injects context for the agent", () => {
    const agent = AgentSchema.parse({
      id: "queen",
      role: "queen",
      prompt_file: "colony/prompts/queen.md",
    });
    const prompt = assemblePromptForAgent(agent, {
      skills: ["orchestration"],
    });
    expect(prompt).toContain("## Injected Skills");
    expect(prompt).toContain("- orchestration");
  });

  it("throws for nonexistent prompt_file", () => {
    const agent = {
      id: "fake",
      role: "builder",
      prompt_file: "colony/prompts/fake.md",
      allowed_tools: [],
    };
    expect(() => assemblePromptForAgent(agent)).toThrow(
      /Prompt file not found/
    );
  });
});
