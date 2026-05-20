/**
 * Unit tests for host.ts parseArgs flag parsing.
 *
 * Tests verify:
 * - All documented flags parse correctly
 * - Flag values are extracted and typed correctly
 * - Positional args are captured
 * - Help flag is detected
 * - Unknown commands are passed through (main() handles the error)
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { parseArgs } from "../src/host.js";

describe("parseArgs", () => {
  it("parses plan with depth and planning-depth", () => {
    const result = parseArgs([
      "node",
      "host.js",
      "plan",
      "--depth",
      "balanced",
      "--planning-depth",
      "standard",
      "--target",
      "95",
      "--max-iterations",
      "8",
      "--accept",
    ]);

    assert.equal(result.command, "plan");
    assert.equal(result.depth, "balanced");
    assert.equal(result.planningDepth, "standard");
    assert.equal(result.targetConfidence, "95");
    assert.equal(result.maxIterations, "8");
    assert.equal(result.accept, true);
    assert.equal(result.cwd, process.cwd());
    assert.equal(result.help, false);
  });

  it("parses equals-form value flags", () => {
    const result = parseArgs([
      "node", "host.js",
      "plan",
      "--depth=balanced",
      "--planning-depth=deep",
      "--verification-depth=heavy",
      "--worker-timeout=5m",
      "--cwd=/tmp/aether",
    ]);

    assert.equal(result.command, "plan");
    assert.equal(result.depth, "balanced");
    assert.equal(result.planningDepth, "deep");
    assert.equal(result.verificationDepth, "heavy");
    assert.equal(result.workerTimeout, "5m");
    assert.equal(result.cwd, "/tmp/aether");
  });

  it("parses plan refresh, force, and synthetic flags", () => {
    const result = parseArgs([
      "node", "host.js",
      "plan",
      "--refresh",
      "--force",
      "--synthetic",
    ]);

    assert.equal(result.command, "plan");
    assert.equal(result.refresh, true);
    assert.equal(result.force, true);
    assert.equal(result.synthetic, true);
  });

  it("parses colonize force-resurvey", () => {
    const result = parseArgs(["node", "host.js", "colonize", "--force-resurvey", "--worker-timeout=5m"]);

    assert.equal(result.command, "colonize");
    assert.equal(result.forceResurvey, true);
    assert.equal(result.workerTimeout, "5m");
  });

  it("parses continue with verification-depth and light", () => {
    const result = parseArgs(["node", "host.js", "continue", "--verification-depth", "heavy", "--light"]);

    assert.equal(result.command, "continue");
    assert.equal(result.verificationDepth, "heavy");
    assert.equal(result.light, true);
    assert.equal(result.heavy, false);
  });

  it("parses watch with no-dashboard", () => {
    const result = parseArgs(["node", "host.js", "watch", "--no-dashboard"]);

    assert.equal(result.command, "watch");
    assert.equal(result.noDashboard, true);
    assert.equal(result.simulate, false);
  });

  it("parses swarm with target and no-dashboard", () => {
    const result = parseArgs(["node", "host.js", "swarm", "test bug", "--no-dashboard"]);

    assert.equal(result.command, "swarm");
    assert.deepStrictEqual(result.positional, ["test bug"]);
    assert.equal(result.noDashboard, true);
  });

  it("parses build with positional and light flag", () => {
    const result = parseArgs(["node", "host.js", "build", "1", "--light"]);

    assert.equal(result.command, "build");
    assert.deepStrictEqual(result.positional, ["1"]);
    assert.equal(result.light, true);
  });

  it("parses build task and release-hardening flags", () => {
    const result = parseArgs([
      "node", "host.js",
      "build", "5",
      "--task", "5.1",
      "--task=5.2",
      "--force",
      "--circuit-breaker-threshold=4",
      "--no-suggest",
      "--verbose",
    ]);

    assert.equal(result.command, "build");
    assert.deepStrictEqual(result.positional, ["5"]);
    assert.deepStrictEqual(result.tasks, ["5.1", "5.2"]);
    assert.equal(result.force, true);
    assert.equal(result.circuitBreakerThreshold, "4");
    assert.equal(result.noSuggest, true);
    assert.equal(result.verbose, true);
  });

  it("parses oracle with topic and simulate", () => {
    const result = parseArgs(["node", "host.js", "oracle", "security audit", "--simulate"]);

    assert.equal(result.command, "oracle");
    assert.deepStrictEqual(result.positional, ["security audit"]);
    assert.equal(result.simulate, true);
  });

  it("parses lifecycle with phase and skip-midden-check", () => {
    const result = parseArgs(["node", "host.js", "lifecycle", "2", "--skip-midden-check"]);

    assert.equal(result.command, "lifecycle");
    assert.deepStrictEqual(result.positional, ["2"]);
    assert.equal(result.skipMiddenCheck, true);
  });

  it("parses help flag", () => {
    const result = parseArgs(["node", "host.js", "--help"]);

    assert.equal(result.help, true);
    assert.equal(result.command, "");
  });

  it("parses short help flag", () => {
    const result = parseArgs(["node", "host.js", "-h"]);

    assert.equal(result.help, true);
  });

  it("parses custom cwd", () => {
    const result = parseArgs(["node", "host.js", "plan", "--cwd", "/tmp"]);

    assert.equal(result.cwd, "/tmp");
  });

  it("parses worker-timeout", () => {
    const result = parseArgs(["node", "host.js", "plan", "--worker-timeout", "5m"]);

    assert.equal(result.workerTimeout, "5m");
  });

  it("parses heavy flag", () => {
    const result = parseArgs(["node", "host.js", "continue", "--heavy"]);

    assert.equal(result.heavy, true);
  });

  it("parses skip-watchers flag", () => {
    const result = parseArgs(["node", "host.js", "continue", "--skip-watchers"]);

    assert.equal(result.skipWatchers, true);
  });

  it("parses continue-only review flags", () => {
    const result = parseArgs([
      "node", "host.js",
      "continue",
      "--reconcile-task", "5.1",
      "--reconcile-task=5.2",
      "--verification-timeout=30m",
      "--no-learn",
    ]);

    assert.equal(result.command, "continue");
    assert.deepStrictEqual(result.reconcileTasks, ["5.1", "5.2"]);
    assert.equal(result.verificationTimeout, "30m");
    assert.equal(result.noLearn, true);
  });

  it("parses continue classic ceremony mode", () => {
    const result = parseArgs(["node", "host.js", "continue", "--classic-ceremony"]);

    assert.equal(result.command, "continue");
    assert.equal(result.classicCeremony, true);
  });

  it("parses seal force", () => {
    const result = parseArgs(["node", "host.js", "seal", "--force"]);

    assert.equal(result.command, "seal");
    assert.equal(result.force, true);
  });

  it("records unknown flags instead of positionalizing them", () => {
    const result = parseArgs(["node", "host.js", "build", "1", "--mystery", "value"]);

    assert.equal(result.command, "build");
    assert.deepStrictEqual(result.positional, ["1", "value"]);
    assert.deepStrictEqual(result.unknownFlags, ["--mystery"]);
  });

  it("passes unknown command through (main() handles error)", () => {
    const result = parseArgs(["node", "host.js", "unknown"]);

    assert.equal(result.command, "unknown");
    assert.equal(result.help, false);
  });
});
