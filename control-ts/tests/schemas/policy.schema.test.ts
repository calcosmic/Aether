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

  it("validates memory-rules.yaml", () => {
    const text = readFileSync(fixturePath("policies", "memory-rules.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.memory_rules.max_learnings).toBe(100);
    expect(result.memory_rules.learning_retention_days).toBe(60);
    expect(result.memory_rules.instinct_cap).toBe(30);
    expect(result.memory_rules.event_cap).toBe(100);
  });

  it("validates skill-creation.yaml", () => {
    const text = readFileSync(fixturePath("policies", "skill-creation.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.skill_creation.allowed).toBe(true);
    expect(result.skill_creation.max_skills_per_colony).toBe(50);
    expect(result.skill_creation.require_wisdom_threshold).toBe(0.8);
  });

  it("validates safety-gates.yaml", () => {
    const text = readFileSync(fixturePath("policies", "safety-gates.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.safety_gates.security_scan).toBe(true);
    expect(result.safety_gates.chaos_scan).toBe(true);
    expect(result.safety_gates.auditor_score_threshold).toBe(60);
  });

  it("validates dispatch-contract.yaml", () => {
    const text = readFileSync(fixturePath("policies", "dispatch-contract.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.dispatch_contract.max_workers_per_phase).toBe(10);
    expect(result.dispatch_contract.spawn_depth_limits).toEqual({ 0: 4, 1: 4, 2: 2, 3: 0 });
    expect(result.dispatch_contract.timeout_defaults).toEqual({
      build: 600,
      continue: 300,
      plan: 300,
      verify: 180,
    });
  });

  it("validates pheromone-lifecycle.yaml", () => {
    const text = readFileSync(fixturePath("policies", "pheromone-lifecycle.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.pheromone_lifecycle.default_ttl_days).toBe(30);
    expect(result.pheromone_lifecycle.auto_expire_on_phase_end).toBe(true);
    expect(result.pheromone_lifecycle.max_active_signals).toBe(100);
    expect(result.pheromone_lifecycle.strength_decay_per_day).toBe(0.05);
  });

  it("validates signal-rules.yaml", () => {
    const text = readFileSync(fixturePath("policies", "signal-rules.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.signal_rules.auto_emit_on_phase_complete).toBe(true);
    expect(result.signal_rules.max_feedback_per_phase).toBe(3);
    expect(result.signal_rules.hard_constraint_prefixes).toContain("[error-pattern]");
    expect(result.signal_rules.hard_constraint_prefixes).toContain("[redirect]");
  });

  it("validates autopilot.yaml", () => {
    const text = readFileSync(fixturePath("policies", "autopilot.yaml"), "utf8");
    const data = parse(text);
    const result = PolicySchema.parse(data);
    expect(result.autopilot.replan_interval).toBe(3);
    expect(result.autopilot.pause_conditions).toContain("test_failure");
    expect(result.autopilot.pause_conditions).toContain("critical_chaos");
    expect(result.autopilot.pause_conditions).toContain("security_gate_failure");
    expect(result.autopilot.pause_conditions).toContain("quality_gate_failure");
    expect(result.autopilot.pause_conditions).toContain("runtime_verification_needed");
    expect(result.autopilot.max_phases_per_run).toBeUndefined();
  });

  it("provides defaults for empty object", () => {
    const result = PolicySchema.parse({});
    expect(result.model_routing).toBeDefined();
    expect(result.memory_rules).toBeDefined();
    expect(result.skill_creation).toBeDefined();
    expect(result.safety_gates).toBeDefined();
    expect(result.dispatch_contract).toBeDefined();
    expect(result.pheromone_lifecycle).toBeDefined();
    expect(result.signal_rules).toBeDefined();
    expect(result.autopilot).toBeDefined();
    expect(result.memory_rules.max_learnings).toBe(100);
    expect(result.skill_creation.allowed).toBe(true);
    expect(result.safety_gates.security_scan).toBe(true);
    expect(result.dispatch_contract.max_workers_per_phase).toBe(10);
    expect(result.pheromone_lifecycle.default_ttl_days).toBe(30);
    expect(result.signal_rules.auto_emit_on_phase_complete).toBe(true);
    expect(result.autopilot.replan_interval).toBe(3);
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
