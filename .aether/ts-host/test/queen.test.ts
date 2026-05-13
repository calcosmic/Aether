/**
 * Unit tests for the Queen orchestrator module.
 *
 * Tests verify:
 * - Workflow pattern derivation from dispatch composition
 * - Builder-Probe Lock downgrade behavior
 * - Probe verification detection
 * - Builder-Probe Lock satisfaction checks
 * - Midden summary formatting
 * - Wave failure recovery action mapping
 *
 * Tests calling Go CLI (midden, escalation) mock callGoJSON or use heuristics.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import type { BuildDispatch, WorkerResult } from "../src/types.js";
import {
  deriveWorkflowPattern,
  mapVerificationDepth,
  formatQueenRecommendation,
} from "../src/queen/workflow-patterns.js";
import {
  applyBuilderProbeLock,
  hasProbeVerification,
  isBuilderProbeLockSatisfied,
} from "../src/queen/builder-probe-lock.js";
import { formatMiddenSummary } from "../src/queen/midden-check.js";
import { handleWaveFailures } from "../src/queen/escalation.js";
import type { DispatchResult } from "../src/worker-dispatch.js";
import type { MiddenCheckResult } from "../src/queen/types.js";

// ---------------------------------------------------------------------------
// Test data helpers
// ---------------------------------------------------------------------------

function makeDispatch(
  name: string,
  caste: string,
  wave = 1
): BuildDispatch {
  return {
    stage: "implement",
    wave,
    caste,
    name,
    task: `Task for ${name}`,
    status: "pending",
  };
}

function makeWorkerResult(
  name: string,
  status: string,
  caste?: string
): WorkerResult {
  return {
    name,
    status,
    caste,
    task: `Task for ${name}`,
  };
}

// ---------------------------------------------------------------------------
// Workflow pattern tests
// ---------------------------------------------------------------------------

describe("workflow-patterns", () => {
  it("deriveWorkflowPattern returns Deep Research for oracle+scout", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Oracle-1", "oracle"),
      makeDispatch("Scout-1", "scout"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "Deep Research");
  });

  it("deriveWorkflowPattern returns SPBV for builder-only", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "SPBV");
  });

  it("deriveWorkflowPattern returns Investigate-Fix for chaos", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Chaos-1", "chaos"),
      makeDispatch("Builder-1", "builder"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "Investigate-Fix");
  });

  it("deriveWorkflowPattern returns Refactor for weaver without test castes", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Weaver-1", "weaver"),
      makeDispatch("Builder-1", "builder"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "Refactor");
  });

  it("deriveWorkflowPattern returns Compliance for gatekeeper", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Gatekeeper-1", "gatekeeper"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "Compliance");
  });

  it("deriveWorkflowPattern returns Documentation Sprint for chronicler", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Chronicler-1", "chronicler"),
    ];
    assert.equal(deriveWorkflowPattern(dispatches), "Documentation Sprint");
  });

  it("mapVerificationDepth maps fast to Fast", () => {
    assert.equal(mapVerificationDepth("fast"), "Fast");
  });

  it("mapVerificationDepth maps final-review to Heavy", () => {
    assert.equal(mapVerificationDepth("final-review"), "Heavy");
  });

  it("formatQueenRecommendation formats correctly", () => {
    const rec = { review_depth: "standard", reason: "Test reason" };
    assert.equal(formatQueenRecommendation(rec), "standard: Test reason");
  });
});

// ---------------------------------------------------------------------------
// Builder-Probe Lock tests
// ---------------------------------------------------------------------------

describe("builder-probe-lock", () => {
  it("applyBuilderProbeLock downgrades builder when no probe", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Builder-1", "completed", "builder"),
    ];

    const lockResult = applyBuilderProbeLock(results, dispatches);

    assert.equal(lockResult.downgraded, true);
    assert.equal(lockResult.results[0]!.status, "code_written");
    assert.ok(lockResult.results[0]!.summary!.includes("Builder-Probe Lock"));
  });

  it("applyBuilderProbeLock preserves completed when probe verified", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
      makeDispatch("Probe-1", "probe"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Builder-1", "completed", "builder"),
      makeWorkerResult("Probe-1", "completed", "probe"),
    ];

    const lockResult = applyBuilderProbeLock(results, dispatches);

    assert.equal(lockResult.downgraded, false);
    assert.equal(lockResult.results[0]!.status, "completed");
  });

  it("hasProbeVerification returns true when probe completed", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
      makeDispatch("Probe-1", "probe"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Probe-1", "completed", "probe"),
    ];

    assert.equal(hasProbeVerification(results, dispatches), true);
  });

  it("hasProbeVerification returns false when no probe completed", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Builder-1", "completed", "builder"),
    ];

    assert.equal(hasProbeVerification(results, dispatches), false);
  });

  it("isBuilderProbeLockSatisfied returns true when no builders", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Probe-1", "probe"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Probe-1", "completed", "probe"),
    ];

    assert.equal(isBuilderProbeLockSatisfied(results, dispatches), true);
  });

  it("isBuilderProbeLockSatisfied returns true when no probes", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Builder-1", "completed", "builder"),
    ];

    assert.equal(isBuilderProbeLockSatisfied(results, dispatches), true);
  });

  it("isBuilderProbeLockSatisfied returns false when builder but probe failed", () => {
    const dispatches: BuildDispatch[] = [
      makeDispatch("Builder-1", "builder"),
      makeDispatch("Probe-1", "probe"),
    ];
    const results: WorkerResult[] = [
      makeWorkerResult("Builder-1", "completed", "builder"),
      makeWorkerResult("Probe-1", "failed", "probe"),
    ];

    assert.equal(isBuilderProbeLockSatisfied(results, dispatches), false);
  });
});

// ---------------------------------------------------------------------------
// Midden check tests
// ---------------------------------------------------------------------------

describe("midden-check", () => {
  it("formatMiddenSummary shows threshold breached", () => {
    const result: MiddenCheckResult = {
      exceeded: true,
      total: 5,
      threshold: 3,
      categories: { build: 3, test: 2 },
    };

    const summary = formatMiddenSummary(result);
    assert.ok(summary.includes("THRESHOLD BREACHED"));
    assert.ok(summary.includes("5 entries"));
    assert.ok(summary.includes("build: 3"));
    assert.ok(summary.includes("test: 2"));
  });

  it("formatMiddenSummary shows within limits", () => {
    const result: MiddenCheckResult = {
      exceeded: false,
      total: 1,
      threshold: 3,
      categories: {},
    };

    const summary = formatMiddenSummary(result);
    assert.ok(summary.includes("within limits"));
    assert.ok(summary.includes("1 entries"));
  });
});

// ---------------------------------------------------------------------------
// Escalation tests
// ---------------------------------------------------------------------------

describe("escalation", () => {
  it("handleWaveFailures maps failed workers to retry actions", () => {
    const failures: DispatchResult[] = [
      { name: "Worker-1", status: "failed", summary: "Compile error" },
      { name: "Worker-2", status: "failed", summary: "Test failure" },
    ];

    const actions = handleWaveFailures(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      failures
    );

    assert.equal(actions.length, 2);
    assert.equal(actions[0]!.type, "retry");
    assert.equal(actions[0]!.worker, "Worker-1");
    assert.ok(actions[0]!.reason!.includes("recoverable"));
    assert.equal(actions[1]!.type, "retry");
    assert.equal(actions[1]!.worker, "Worker-2");
  });

  it("handleWaveFailures maps blocked workers to escalate actions", () => {
    const failures: DispatchResult[] = [
      { name: "Worker-1", status: "blocked", summary: "Gate blocked" },
    ];

    const actions = handleWaveFailures(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      failures
    );

    assert.equal(actions.length, 1);
    assert.equal(actions[0]!.type, "escalate");
    assert.equal(actions[0]!.worker, "Worker-1");
    assert.ok(actions[0]!.reason!.includes("blocking"));
  });

  it("handleWaveFailures maps timeout workers to fixer_dispatch actions", () => {
    const failures: DispatchResult[] = [
      { name: "Worker-1", status: "timeout", summary: "Timed out" },
    ];

    const actions = handleWaveFailures(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      failures
    );

    assert.equal(actions.length, 1);
    assert.equal(actions[0]!.type, "fixer_dispatch");
    assert.equal(actions[0]!.worker, "Worker-1");
    assert.ok(actions[0]!.reason!.includes("requires-attempt"));
  });
});
