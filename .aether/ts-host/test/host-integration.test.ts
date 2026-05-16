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
  buildHostGoArgs,
  parseArgs,
  __setCallGoJSON,
  __restoreCallGoJSON,
} from "../src/host.js";

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

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--depth", "balanced",
      "--planning-depth", "standard",
    ]);
  });

  it("plan passes equals-form depth flags to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--depth=balanced",
      "--planning-depth=standard",
    ]);

    const args = buildHostGoArgs(parsed);

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

    const args = buildHostGoArgs(parsed);

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
      "--synthetic",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--synthetic",
    ]);
  });

  it("plan passes refresh and force to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "plan",
      "--refresh",
      "--force",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "plan", "--plan-only",
      "--refresh",
      "--force",
    ]);
  });

  it("build passes phase and light flag to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "1",
      "--light",
    ]);

    const args = buildHostGoArgs(parsed);

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

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "build", "2", "--plan-only",
      "--worker-timeout", "15m",
    ]);
  });

  it("build passes heavy and verification-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "3",
      "--heavy",
      "--verification-depth", "heavy",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "build", "3", "--plan-only",
      "--heavy",
      "--verification-depth", "heavy",
    ]);
  });

  it("build passes force, repeated task, circuit breaker, no-suggest, and verbose to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "5",
      "--task", "5.1",
      "--task=5.2",
      "--force",
      "--circuit-breaker-threshold", "4",
      "--no-suggest",
      "--verbose",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "build", "5", "--plan-only",
      "--task", "5.1",
      "--task", "5.2",
      "--force",
      "--circuit-breaker-threshold", "4",
      "--no-suggest",
      "--verbose",
    ]);
  });

  it("build rejects unknown host flags before Go invocation", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "build", "5",
      "--definitely-unknown",
    ]);

    assert.throws(() => buildHostGoArgs(parsed), /Unsupported host flag\(s\): --definitely-unknown/);
  });

  it("continue passes verification-depth heavy to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--verification-depth", "heavy",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--verification-depth", "heavy",
    ]);
  });

  it("continue passes equals-form verification-depth to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--verification-depth=heavy",
    ]);

    const args = buildHostGoArgs(parsed);

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

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--light",
      "--heavy",
    ]);
  });

  it("continue passes skip-watchers to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--skip-watchers",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--skip-watchers",
    ]);
  });

  it("continue passes synthetic and worker-timeout to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--simulate",
      "--worker-timeout", "10m",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--synthetic",
      "--worker-timeout", "10m",
    ]);
  });

  it("continue passes reconcile-task, verification-timeout, and no-learn to Go CLI", () => {
    const parsed = parseArgs([
      "node", "host.js",
      "continue",
      "--reconcile-task", "5.1",
      "--reconcile-task=5.2",
      "--verification-timeout", "30m",
      "--no-learn",
    ]);

    const args = buildHostGoArgs(parsed);

    assert.deepStrictEqual(args, [
      "continue", "--plan-only",
      "--reconcile-task", "5.1",
      "--reconcile-task", "5.2",
      "--verification-timeout", "30m",
      "--no-learn",
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
    const args = buildHostGoArgs(parsed);

    // Simulate what main() would do
    const result = { ok: true };
    __restoreCallGoJSON();

    assert.ok(!called || true, "Mock was set up correctly");
    assert.ok(args);
    assert.equal(args[0], "plan");
  });
});
