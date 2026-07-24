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
    dryRun: false,
    synthetic: false,
    noDashboard: false,
    skipMiddenCheck: false,
    skipWatchers: false,
    refresh: false,
    force: false,
    forceResurvey: false,
    tasks: [],
    depth: undefined,
    planningDepth: undefined,
    verificationDepth: undefined,
    targetConfidence: undefined,
    maxIterations: undefined,
    accept: false,
    revisionType: undefined,
    revisionReason: undefined,
    revisionEvidence: [],
    verificationTimeout: undefined,
    light: false,
    heavy: false,
    workerTimeout: undefined,
    circuitBreakerThreshold: undefined,
    noSuggest: false,
    verbose: false,
    reconcileTasks: [],
    noLearn: false,
    classicCeremony: false,
    help: false,
    positional: [],
    unknownFlags: [],
    ...overrides,
  };
}

describe("command registry", () => {
  it("lists the current host-supported commands in one place", () => {
    assert.deepEqual(HOST_COMMAND_NAMES, [
      "colonize",
      "plan",
      "build",
      "continue",
      "seal",
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
    assert.deepEqual(
      buildHostGoArgs(parsed({ command: "plan", targetConfidence: "95", maxIterations: "8", accept: true })),
      ["plan", "--plan-only", "--target", "95", "--max-iterations", "8", "--accept"]
    );
    assert.deepEqual(
      buildHostGoArgs(parsed({
        command: "plan",
        refresh: true,
        revisionType: "research",
        revisionReason: "Oracle disproved the assumption",
        revisionEvidence: [".aether/oracle/synthesis.md"],
      })),
      [
        "plan", "--plan-only", "--refresh",
        "--revision-type", "research",
        "--revision-reason", "Oracle disproved the assumption",
        "--revision-evidence", ".aether/oracle/synthesis.md",
      ]
    );
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
    assert.deepEqual(buildHostGoArgs(parsed({ command: "colonize", forceResurvey: true, workerTimeout: "5m" })), [
      "colonize",
      "--plan-only",
      "--force-resurvey",
      "--worker-timeout",
      "5m",
    ]);
    assert.deepEqual(buildHostGoArgs(parsed({ command: "continue", skipWatchers: true, reconcileTasks: ["5.1"] })), [
      "continue",
      "--plan-only",
      "--reconcile-task",
      "5.1",
      "--skip-watchers",
    ]);
    assert.deepEqual(buildHostGoArgs(parsed({ command: "continue", classicCeremony: true })), [
      "continue",
      "--plan-only",
      "--classic-ceremony",
    ]);
    assert.deepEqual(buildHostGoArgs(parsed({ command: "seal", force: true })), ["seal", "--plan-only", "--force"]);
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
