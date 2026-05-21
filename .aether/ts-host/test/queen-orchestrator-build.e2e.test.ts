/**
 * End-to-end tests for the Queen orchestrator build path.
 *
 * Tests verify:
 * - createQueenOrchestrator.runBuild produces a valid result structure
 * - Workflow pattern is derived from dispatch composition
 * - Builder-Probe Lock is applied correctly
 * - Recovery actions are generated for failed workers
 * - Manifest schema produced matches Go runtime expectations
 *
 * All AI provider calls are mocked via __setDispatchSingleWorker.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import { createQueenOrchestrator, runBuild } from "../src/queen/orchestrator.js";
import type { QueenOrchestratorOptions } from "../src/queen/types.js";
import type { BuildDispatch } from "../src/types.js";
import {
  __setDispatchSingleWorker,
  __restoreDispatchSingleWorker,
} from "../src/wave-orchestrator.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeOpts(overrides?: Partial<QueenOrchestratorOptions>): QueenOrchestratorOptions {
  return {
    goBinaryPath: "/usr/bin/aether",
    cwd: "/tmp/test",
    phase: 1,
    skipMiddenCheck: true,
    simulateWorkers: true,
    parallel: false,
    dashboard: false,
    ...overrides,
  };
}

function makeDispatch(name: string, caste: string): BuildDispatch {
  return {
    stage: "implement",
    wave: 1,
    caste,
    name,
    task: `Task for ${name}`,
    status: "pending",
  };
}

// ---------------------------------------------------------------------------
// Mock setup
// ---------------------------------------------------------------------------

beforeEach(() => {
  __restoreDispatchSingleWorker();
});

afterEach(() => {
  __restoreDispatchSingleWorker();
});

// ---------------------------------------------------------------------------
// Build success path
// ---------------------------------------------------------------------------

describe("orchestrator build success", () => {
  it("runBuild returns success when all workers complete", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [
        makeDispatch("Builder-01", "builder"),
        makeDispatch("Probe-01", "probe"),
      ],
    });

    assert.equal(result.success, true);
    assert.equal(result.workerResults.length, 2);
    assert.equal(result.workerResults[0]!.name, "Builder-01");
    assert.equal(result.workerResults[0]!.status, "completed");
    assert.equal(result.pattern, "SPBV");
    assert.ok(result.recommendation, "Should have a recommendation");
    assert.ok(result.recommendation.review_depth, "Recommendation should have review_depth");
  });

  it("orchestrator factory produces same result as direct call", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
    }));

    const opts = makeOpts();
    const manifest = { dispatches: [makeDispatch("Builder-01", "builder")] };

    const directResult = await runBuild(opts, manifest);
    const factoryResult = await createQueenOrchestrator(opts).runBuild(manifest);

    assert.equal(directResult.success, factoryResult.success);
    assert.equal(directResult.workerResults.length, factoryResult.workerResults.length);
    assert.equal(directResult.pattern, factoryResult.pattern);
  });

  it("derives Deep Research pattern for oracle+scout dispatches", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Researched",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [
        makeDispatch("Oracle-01", "oracle"),
        makeDispatch("Scout-01", "scout"),
      ],
    });

    assert.equal(result.pattern, "Deep Research");
    assert.equal(result.success, true);
  });

  it("derives Compliance pattern for gatekeeper dispatches", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Audited",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [makeDispatch("Gatekeeper-01", "gatekeeper")],
    });

    assert.equal(result.pattern, "Compliance");
  });
});

// ---------------------------------------------------------------------------
// Builder-Probe Lock
// ---------------------------------------------------------------------------

describe("orchestrator builder-probe lock", () => {
  it("downgrades builder to code_written when no probe verifies", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Built",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    // When no probe was dispatched, the lock downgrades the builder but
    // the build still succeeds because no probe was required.
    assert.equal(result.workerResults[0]!.status, "code_written");
    assert.ok(
      result.workerResults[0]!.summary!.includes("Builder-Probe Lock"),
      "Summary should mention lock"
    );
  });

  it("preserves completed when probe verifies builder", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [
        makeDispatch("Builder-01", "builder"),
        makeDispatch("Probe-01", "probe"),
      ],
    });

    assert.equal(result.success, true);
    assert.equal(result.workerResults[0]!.status, "completed");
    assert.equal(result.workerResults[1]!.status, "completed");
  });
});

// ---------------------------------------------------------------------------
// Failure recovery
// ---------------------------------------------------------------------------

describe("orchestrator failure recovery", () => {
  it("returns recovery actions for failed workers", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "failed",
      summary: "Compile error",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    assert.equal(result.success, false);
    assert.ok(result.recoveryActions, "Should have recovery actions");
    assert.equal(result.recoveryActions!.length, 1);
    assert.equal(result.recoveryActions![0]!.type, "retry");
    assert.equal(result.recoveryActions![0]!.worker, "Builder-01");
  });

  it("preserves blocked status in worker results", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "blocked",
      summary: "Gate blocked",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    // "blocked" is not treated as a wave failure by dispatchWave,
    // so the build succeeds but the worker status is preserved.
    assert.equal(result.success, true);
    assert.equal(result.workerResults[0]!.status, "blocked");
    assert.equal(result.workerResults[0]!.summary, "Gate blocked");
  });

  it("includes error message when build fails", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "failed",
      summary: "Error",
      duration: 5,
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    assert.ok(result.error, "Should have error message");
    assert.ok(result.error!.includes("failure"), "Error should mention failures");
  });
});

// ---------------------------------------------------------------------------
// Midden integration
// ---------------------------------------------------------------------------

describe("orchestrator midden check", () => {
  it("includes midden result when not skipped", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
    }));

    const result = await runBuild(makeOpts({ skipMiddenCheck: false }), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    assert.ok(result.middenResult, "Should have midden result");
    assert.equal(typeof result.middenResult!.total, "number");
    assert.equal(typeof result.middenResult!.threshold, "number");
  });

  it("omits midden result when skipped", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
    }));

    const result = await runBuild(makeOpts({ skipMiddenCheck: true }), {
      dispatches: [makeDispatch("Builder-01", "builder")],
    });

    assert.equal(result.middenResult, undefined);
  });
});

// ---------------------------------------------------------------------------
// Manifest schema validation
// ---------------------------------------------------------------------------

describe("manifest schema roundtrip", () => {
  it("produces a result that serializes to valid JSON", async () => {
    __setDispatchSingleWorker(async (_opts, dispatch) => ({
      name: dispatch.name,
      status: "completed",
      summary: "Done",
      duration: 5,
      files_created: ["src/module.ts"],
      files_modified: ["src/helper.ts"],
      tests_written: ["test/module.test.ts"],
    }));

    const result = await runBuild(makeOpts(), {
      dispatches: [
        makeDispatch("Builder-01", "builder"),
        makeDispatch("Probe-01", "probe"),
      ],
    });

    // Verify the result can be serialized and deserialized
    const json = JSON.stringify(result);
    const parsed = JSON.parse(json);

    assert.equal(parsed.success, true);
    assert.equal(parsed.workerResults.length, 2);
    assert.equal(parsed.pattern, "SPBV");
    assert.ok(parsed.recommendation);
    assert.equal(typeof parsed.recommendation.review_depth, "string");
    assert.equal(typeof parsed.recommendation.reason, "string");
  });

});
