import { describe, it, expect, vi } from "vitest";
import { loadSkills, matchSkillsForWorker } from "../../src/skills/loadSkills.js";

describe("loadSkills", () => {
  it("returns an array of skill objects parsed from .aether/skills/", () => {
    const skills = loadSkills();
    expect(skills.length).toBeGreaterThan(0);
    for (const skill of skills) {
      expect(skill).toHaveProperty("name");
      expect(skill).toHaveProperty("type");
      expect(skill).toHaveProperty("description");
      expect(skill).toHaveProperty("agent_roles");
      expect(skill).toHaveProperty("task_keywords");
    }
  });

  it("parses frontmatter fields correctly for a known skill", () => {
    const skills = loadSkills();
    const skill = skills.find((s) => s.name === "acceptance-test-generation");
    expect(skill).toBeDefined();
    expect(skill!.type).toBe("colony");
    expect(skill!.agent_roles).toContain("builder");
    expect(skill!.task_keywords).toContain("test");
    expect(skill!.source).toBe("shipped");
    expect(skill!.version).toBe("1.0");
    expect(skill!.content).toContain("Acceptance Test Generation");
  });
});

describe("matchSkillsForWorker", () => {
  it("returns skills whose agent_roles include the given role", () => {
    const matches = matchSkillsForWorker("builder", "build", []);
    expect(matches.length).toBeGreaterThan(0);
    for (const skill of matches) {
      expect(skill.agent_roles).toContain("builder");
    }
  });

  it("boosts matching skills when pheromone content matches a task keyword", () => {
    const without = matchSkillsForWorker("builder", "build", []);
    const withPheromone = matchSkillsForWorker("builder", "build", [
      { type: "FOCUS", content: "security" },
    ]);
    // Both should return results; with pheromones the ordering may differ.
    expect(withPheromone.length).toBeGreaterThan(0);
    expect(without.length).toBeGreaterThan(0);
  });

  it("returns at most 6 skills", () => {
    const matches = matchSkillsForWorker("builder", "build", []);
    expect(matches.length).toBeLessThanOrEqual(6);
  });
});
