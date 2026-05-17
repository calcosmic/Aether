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

import type {
  BuildDispatch,
  BuildManifest,
  QueenSpawnBudget,
  WorkerResult,
} from "../src/types.js";
import {
  deriveWorkflowPattern,
  deriveExecutionPolicy,
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
import type { TerminalWorkerStatus } from "../src/types.js";
import { dispatchWave, type WaveOrchestratorOptions } from "../src/wave-orchestrator.js";

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
    task: `Task for ${name}`,
    ...(caste !== undefined ? { caste } : {}),
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

  it("deriveExecutionPolicy preserves an existing spawn budget", () => {
    const spawnBudget: QueenSpawnBudget = {
      max_workers: 10,
      selected_workers: 10,
      worker_count: 10,
      max_selected_castes: 8,
      selected_castes: 7,
      preserved_castes: [
        "auditor",
        "builder",
        "gatekeeper",
        "probe",
        "watcher",
      ],
      required_castes: [
        "auditor",
        "builder",
        "gatekeeper",
        "probe",
        "watcher",
      ],
      policy_added_castes: ["keeper"],
      relevance_threshold: 60,
      budget_unit: "caste",
      reason: "high-risk or production build",
      flow_type: "build",
      risk_level: "low",
      castes: [
        "auditor",
        "builder",
        "gatekeeper",
        "keeper",
        "probe",
        "watcher",
      ],
      counts: {
        auditor: 1,
        builder: 4,
        gatekeeper: 1,
        keeper: 1,
        probe: 1,
        watcher: 1,
      },
    };

    const policy = deriveExecutionPolicy(
      { review_depth: "final-review", reason: "Test reason" },
      "Compliance",
      { spawn_budget: spawnBudget }
    );

    assert.equal(policy.verification_depth, "Heavy");
    assert.equal(policy.review_depth, "final-review");
    assert.deepEqual(policy.spawn_budget, spawnBudget);
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
// Type and default tests
// ---------------------------------------------------------------------------

describe("types and defaults", () => {
  it("code_written is valid TerminalWorkerStatus", () => {
    const status: TerminalWorkerStatus = "code_written";
    assert.equal(status, "code_written");
  });

  it("BuildManifest accepts the Go-authored queen spawn budget contract", () => {
    const manifest: Pick<BuildManifest, "queen_execution_policy"> = {
      queen_execution_policy: {
        verification_depth: "heavy",
        review_depth: "heavy",
        spawn_budget: {
          max_workers: 10,
          selected_workers: 10,
          worker_count: 10,
          max_selected_castes: 8,
          selected_castes: 7,
          budget_unit: "caste",
          selected_reasons: {
            builder: "selected within Queen spawn budget 8 (test)",
          },
          pruned_reasons: {
            architect: "not spawned; outside Queen spawn budget 8 (test)",
          },
          skipped_castes: ["architect"],
          counts: { builder: 4, watcher: 1 },
        },
      },
    };

    assert.equal(
      manifest.queen_execution_policy?.spawn_budget?.counts?.builder,
      4
    );
    assert.equal(
      manifest.queen_execution_policy?.spawn_budget?.selected_reasons?.builder,
      "selected within Queen spawn budget 8 (test)"
    );
    assert.equal(
      manifest.queen_execution_policy?.spawn_budget?.pruned_reasons?.architect,
      "not spawned; outside Queen spawn budget 8 (test)"
    );
    assert.deepEqual(
      manifest.queen_execution_policy?.spawn_budget?.skipped_castes,
      ["architect"]
    );
  });

  it("BuildManifest preserves Go-owned boundary and result collection guidance", () => {
    const manifest: Pick<
      BuildManifest,
      "dispatch_contract" | "orchestrator_boundary_guidance"
    > = {
      dispatch_contract: {
        result_artifact_paths: [
          "${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json",
        ],
        result_collection_policy:
          "A structurally valid completed result wins over a timeout placeholder.",
      },
      orchestrator_boundary_guidance: {
        active: false,
        after_discuss_next: "aether build 4",
      },
    };

    assert.deepEqual(manifest.dispatch_contract?.result_artifact_paths, [
      "${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json",
    ]);
    assert.ok(
      manifest.dispatch_contract?.result_collection_policy?.includes(
        "structurally valid completed result"
      )
    );
    assert.equal(manifest.orchestrator_boundary_guidance?.active, false);
  });

  it("wave orchestrator retryLimit defaults to 1", async () => {
    // Create a minimal wave dispatch with a worker that always fails.
    // With default retryLimit=1, only 1 attempt should be made.
    const dispatches: BuildDispatch[] = [
      makeDispatch("Retry-Default", "builder"),
    ];

    let callCount = 0;
    const failMock = async (
      _opts: WaveOrchestratorOptions,
      dispatch: BuildDispatch
    ): Promise<DispatchResult> => {
      callCount++;
      return {
        name: dispatch.name,
        status: "failed",
        summary: "Always fails",
      };
    };

    const { __setDispatchSingleWorker, __restoreDispatchSingleWorker } =
      await import("../src/wave-orchestrator.js");
    __setDispatchSingleWorker(failMock);

    try {
      const result = await dispatchWave(
        {
          goBinaryPath: "/usr/bin/true",
          cwd: "/tmp",
          simulateWorkers: true,
          parallel: true,
          // retryLimit intentionally omitted to test default
        },
        dispatches
      );

      assert.equal(result.failures.length, 1, "Worker should fail");
      assert.equal(callCount, 1, "Default retryLimit=1 means exactly 1 attempt");
    } finally {
      __restoreDispatchSingleWorker();
    }
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

  it("handleWaveFailures escalates runtime-owned stale clarification failures", () => {
    const actions = handleWaveFailures(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      [
        {
          name: "Worker-1",
          status: "timeout",
          summary: "stale clarification requires aether discuss before rebuild",
        },
      ]
    );

    assert.equal(actions.length, 1);
    assert.equal(actions[0]!.type, "escalate");
    assert.ok(actions[0]!.reason!.includes("blocking"));
  });

  it("handleWaveFailures escalates missed result collection failures", () => {
    const actions = handleWaveFailures(
      { goBinaryPath: "/usr/bin/true", cwd: "/tmp" },
      [
        {
          name: "Worker-1",
          status: "failed",
          summary: "result collection missed worker result artifact",
        },
      ]
    );

    assert.equal(actions.length, 1);
    assert.equal(actions[0]!.type, "escalate");
    assert.ok(actions[0]!.reason!.includes("blocking"));
  });
});
