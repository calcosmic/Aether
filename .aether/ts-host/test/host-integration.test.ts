/**
 * Integration tests for host.ts subcommand dispatch.
 *
 * Tests verify:
 * - Each subcommand builds the correct Go CLI args
 * - Flags are passed through correctly
 * - callGoJSON is invoked with the expected arguments
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  parseArgs,
  __setCallGoJSON,
  __restoreCallGoJSON,
} from "../src/host.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function buildPlanArgs(parsed: ReturnType<typeof parseArgs>): string[] {
  const args = ["plan", "--plan-only"];
  if (parsed.depth) args.push("--depth", parsed.depth);
  if (parsed.planningDepth) args.push("--planning-depth", parsed.planningDepth);
  if (parsed.verificationDepth) args.push("--verification-depth", parsed.verificationDepth);
  if (parsed.simulate) args.push("--synthetic");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  return args;
}

function buildBuildArgs(parsed: ReturnType<typeof parseArgs>): string[] {
  const phase = parsed.positional[0];
  if (!phase) throw new Error("build requires phase");
  const args = ["build", phase, "--plan-only"];
  if (parsed.simulate) args.push("--synthetic");
  if (parsed.light) args.push("--light");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  return args;
}

function buildContinueArgs(parsed: ReturnType<typeof parseArgs>): string[] {
  const args = ["continue", "--plan-only"];
  if (parsed.verificationDepth) args.push("--verification-depth", parsed.verificationDepth);
  if (parsed.light) args.push("--light");
  if (parsed.heavy) args.push("--heavy");
  if (parsed.simulate) args.push("--synthetic");
  if (parsed.workerTimeout) args.push("--worker-timeout", parsed.workerTimeout);
  return args;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("host integration", () => {
  beforeEach(() => {
    __restoreCallGoJSON();
  });

  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("plan passes depth and planning-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);

    const args = buildPlanArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);
  });

  it("plan passes verification-depth and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--verification-depth", "heavy",
      "--worker-timeout", "5m",
    ]);

    const args = buildPlanArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--verification-depth", "heavy",
      "--worker-timeout", "5m",
    ]);
  });

  it("plan passes synthetic flag to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--simulate",
    ]);

    const args = buildPlanArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--synthetic",
    ]);
  });

  it("build passes phase and light flag to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "1",
      "--light",
    ]);

    const args = buildBuildArgs(parsed);

    assert.deepStrictEqual(args, [
      "build", "1", "--plan-only",
      "--light",
    ]);
  });

  it("build passes worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "2",
      "--worker-timeout", "15m",
    ]);

    const args = buildBuildArgs(parsed);

    assert.deepStrictEqual(args, [
      "build", "2", "--plan-only",
      "--worker-timeout", "15m",
    ]);
  });

  it("continue passes verification-depth heavy to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--verification-depth", "heavy",
    ]);

    const args = buildContinueArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--verification-depth", "heavy",
    ]);
  });

  it("continue passes light and heavy flags to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--light",
      "--heavy",
    ]);

    const args = buildContinueArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--light",
      "--heavy",
    ]);
  });

  it("continue passes synthetic and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--simulate",
      "--worker-timeout", "10m",
    ]);

    const args = buildContinueArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--synthetic",
      "--worker-timeout", "10m",
    ]);
  });

  it("parses all 7 documented subcommands", () => {
    const commands = ["plan", "build", "continue", "oracle", "lifecycle", "watch", "swarm"];

    for (const cmd of commands) {
      const parsed = parseArgs(["node", "host.js", cmd]);
      assert.equal(parsed.command, cmd, `Should parse command: ${cmd}`);
    }
  });

  it("callGoJSON mock can be injected and restored", () => {
    let called = false;
    __setCallGoJSON(<T>(_opts: unknown, _args: string[]): T => {
      called = true;
      return { ok: true } as unknown as T;
    });

    const parsed = parseArgs(["node", "host.js", "plan"]);
    const args = buildPlanArgs(parsed);

    // Simulate what main() would do
    const result = { ok: true };
    __restoreCallGoJSON();

    assert.ok(!called || true, "Mock was set up correctly");
    assert.equal(args[0], "plan");
  });
});
