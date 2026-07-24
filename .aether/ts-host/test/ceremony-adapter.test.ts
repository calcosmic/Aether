import { readFileSync } from "node:fs";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  GoCeremonyAdapter,
  type CeremonyCommandRunner,
} from "../src/ceremony-adapter.js";

describe("ceremony adapter", () => {
  it("routes spawn-plan through the Go ceremony command with a temp manifest file", () => {
    const calls: string[][] = [];
    const runner: CeremonyCommandRunner = (_opts, args) => {
      calls.push(args);
      return "spawn plan\n";
    };
    const adapter = new GoCeremonyAdapter(
      { goBinaryPath: "/bin/aether", cwd: process.cwd() },
      runner
    );

    const output = adapter.renderSpawnPlan("build", {
      dispatch_manifest: {
        phase: 1,
        phase_name: "Test",
        dispatches: [{ name: "Brick-1", caste: "builder", execution_wave: 1 }],
      },
    });

    assert.equal(output, "spawn plan\n");
    assert.equal(calls.length, 1);
    assert.deepEqual(calls[0]?.slice(0, 4), [
      "ceremony",
      "spawn-plan",
      "--workflow",
      "build",
    ]);
    const manifestFile = calls[0]?.at(-1);
    assert.ok(manifestFile);
    const packet = JSON.parse(readFileSync(manifestFile, "utf-8")) as {
      dispatch_manifest?: { phase?: number };
    };
    assert.equal(packet.dispatch_manifest?.phase, 1);
  });

  it("routes wave-start, worker-complete, and closeout through Go ceremony subcommands", () => {
    const calls: string[][] = [];
    const runner: CeremonyCommandRunner = (_opts, args) => {
      calls.push(args);
      return `${args[1]}\n`;
    };
    const adapter = new GoCeremonyAdapter(
      { goBinaryPath: "/bin/aether", cwd: process.cwd() },
      runner
    );

    adapter.renderWaveStart("plan", { dispatches: [] }, 7);
    adapter.renderWorkerComplete("continue", {
      name: "Watch-1",
      caste: "watcher",
      status: "completed",
    });
    adapter.renderCloseout("build", "/tmp/completion.json");

    assert.equal(calls.length, 3);
    assert.deepEqual(calls[0]?.slice(0, 6), [
      "ceremony",
      "wave-start",
      "--workflow",
      "plan",
      "--manifest-file",
      calls[0]?.[5],
    ]);
    assert.ok(calls[0]?.includes("--execution-wave"));
    assert.ok(calls[0]?.includes("7"));
    assert.deepEqual(calls[1]?.slice(0, 4), [
      "ceremony",
      "worker-complete",
      "--workflow",
      "continue",
    ]);
    assert.deepEqual(calls[2], [
      "ceremony",
      "closeout",
      "--workflow",
      "build",
      "--completion-file",
      "/tmp/completion.json",
    ]);
  });

  it("does not render worker-complete theatre for non-terminal manifest dispatches", () => {
    const calls: string[][] = [];
    const runner: CeremonyCommandRunner = (_opts, args) => {
      calls.push(args);
      return "worker complete\n";
    };
    const adapter = new GoCeremonyAdapter(
      { goBinaryPath: "/bin/aether", cwd: process.cwd() },
      runner
    );

    const output = adapter.renderWorkerComplete("build", {
      name: "Mason-67",
      caste: "builder",
      status: "planned",
      task: "Build the wall",
    });

    assert.equal(output, "");
    assert.equal(calls.length, 0);
  });
});
