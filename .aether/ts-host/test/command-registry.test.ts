import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  HOST_COMMAND_NAMES,
  buildHostGoArgs,
  getHostCommandDefinition,
  listHostCommandDefinitions,
  type ParsedHostArgs,
} from "../src/command-registry.js";

function parsed(overrides: Partial<ParsedHostArgs>): ParsedHostArgs {
  return {
    command: "",
    cwd: "/repo",
    simulate: false,
    synthetic: false,
    noDashboard: false,
    skipMiddenCheck: false,
    skipWatchers: false,
    refresh: false,
    force: false,
    tasks: [],
    depth: undefined,
    planningDepth: undefined,
    verificationDepth: undefined,
    verificationTimeout: undefined,
    light: false,
    heavy: false,
    workerTimeout: undefined,
    circuitBreakerThreshold: undefined,
    noSuggest: false,
    verbose: false,
    reconcileTasks: [],
    noLearn: false,
    help: false,
    positional: [],
    unknownFlags: [],
    ...overrides,
  };
}

describe("command registry", () => {
  it("lists the current host-supported commands in one place", () => {
    assert.deepEqual(HOST_COMMAND_NAMES, [
      "plan",
      "build",
      "continue",
      "oracle",
      "lifecycle",
      "watch",
      "swarm",
    ]);
  });

  it("documents command ownership and finalizer boundaries", () => {
    for (const definition of listHostCommandDefinitions()) {
      assert.equal(definition.literalPassthrough, false, `${definition.command} literal passthrough`);
      assert.ok(definition.category, `${definition.command} category`);
      assert.ok(definition.runner, `${definition.command} runner`);
      if (definition.category === "orchestrated" && definition.command !== "oracle") {
        assert.match(definition.goPlanCommand ?? "", /--plan-only/, `${definition.command} plan command`);
        assert.match(definition.finalizerCommand ?? "", /--completion-file/, `${definition.command} finalizer command`);
      }
    }
  });

  it("builds Go manifest args from registry definitions", () => {
    assert.deepEqual(buildHostGoArgs(parsed({ command: "plan", refresh: true, force: true, depth: "fast" })), [
      "plan",
      "--plan-only",
      "--refresh",
      "--force",
      "--depth",
      "fast",
    ]);
    assert.deepEqual(buildHostGoArgs(parsed({ command: "build", positional: ["2"], tasks: ["5.1", "5.2"], heavy: true, force: true })), [
      "build",
      "2",
      "--plan-only",
      "--task",
      "5.1",
      "--task",
      "5.2",
      "--force",
      "--heavy",
    ]);
    assert.deepEqual(buildHostGoArgs(parsed({ command: "continue", skipWatchers: true, reconcileTasks: ["5.1"] })), [
      "continue",
      "--plan-only",
      "--reconcile-task",
      "5.1",
      "--skip-watchers",
    ]);
  });

  it("rejects unsupported flags before direct manifest arg construction", () => {
    assert.throws(
      () => buildHostGoArgs(parsed({ command: "build", positional: ["2"], unknownFlags: ["--banana"] })),
      /Unsupported host flag\(s\): --banana/
    );
  });

  it("keeps display commands out of direct manifest arg construction", () => {
    assert.equal(getHostCommandDefinition("watch")?.runner, "watch-display");
    assert.equal(buildHostGoArgs(parsed({ command: "watch" })), undefined);
  });
});
