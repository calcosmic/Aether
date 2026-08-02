/**
 * CR-03 gap closure (164-11 Task 2): a failing `plan-research-escalate`
 * currently crashes the whole plan run, because `runResearchConfidenceLoop`'s
 * escalation round calls `_callGoJSONRef` unwrapped. That one Go subcommand
 * failure -- for any reason, not just the CR-03 phase-resolution bug fixed in
 * Task 1 -- must degrade to a named warning and let the plan run finish
 * (D-08: research is enrichment, never a gate).
 *
 * Two of the three cases below drive the REAL `aether plan-research-escalate`
 * binary against a `cwd` with no colony state, so the real `execFileSync` in
 * `go-bridge.ts` genuinely throws (`AETHER_BINARY_PATH`, guaranteed present by
 * `ensure-aether-binary.ts`, and `__restoreCallGoJSON()` -- no `__setCallGoJSON`
 * mock). This is the same failure this task's dist probe reproduces verbatim
 * against the CURRENT compiled dist during planning ("failed to load colony
 * state: no colony initialized").
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";

import {
  parseArgs,
  __setDispatchWorkers,
  __restoreDispatchWorkers,
  __setCallGoJSON,
  __restoreCallGoJSON,
  runResearchConfidenceLoop,
} from "../src/host.js";
import type { GoBridgeOptions, callGoJSON } from "../src/go-bridge.js";
import type { BuildDispatch } from "../src/types.js";
import type { DispatchOptions, DispatchResult } from "../src/worker-dispatch.js";

// ---------------------------------------------------------------------------
// Fixture helpers (mirrors research-confidence-loop.test.ts's own private
// helpers -- not exported there, so duplicated narrowly here)
// ---------------------------------------------------------------------------

/** Scores exactly 35 (base 20 + one filled section (5) + self-assessment 10)
 * and never moves, tripping diminishing_returns by round 3. */
const MEDIUM_MARKDOWN = `## Hive Wisdom (Pre-existing Knowledge)
Some prior wisdom text that is definitely long enough to count as filled content for the section-filled check.
`;

/** Create a temp repo root with the phase-research directory but deliberately
 * NO colony state -- the real plan-research-escalate binary must genuinely
 * fail "no colony initialized" against this cwd. */
function makeFixtureRepo(): string {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), "research-escalate-degrade-"));
  fs.mkdirSync(path.join(repoRoot, ".aether", "data", "phase-research"), { recursive: true });
  return repoRoot;
}

function writePhaseResearch(repoRoot: string, phaseId: number, markdown: string): void {
  fs.writeFileSync(
    path.join(repoRoot, ".aether", "data", "phase-research", `phase-${phaseId}-research.md`),
    markdown
  );
}

function researchDispatch(phaseId: number, scoutName: string) {
  return {
    stage: "phase_research",
    caste: "scout",
    name: scoutName,
    task: `Research domain knowledge for phase ${phaseId}`,
    task_id: `plan-research-phase-${phaseId}`,
    outputs: [`phase-${phaseId}-research.md`],
    status: "planned",
  };
}

function phaseIdFromDispatch(dispatch: BuildDispatch): number | undefined {
  const match = /^plan-research-phase-(\d+)$/.exec(dispatch.task_id ?? "");
  return match ? parseInt(match[1]!, 10) : undefined;
}

interface RoundHarness {
  rounds: string[][];
  stderrOutput: string;
}

function createHarness(): RoundHarness {
  return { rounds: [], stderrOutput: "" };
}

/** Same shape as research-confidence-loop.test.ts's installDispatchMock:
 * records dispatched worker names per round and writes the next markdown
 * fixture for each active research phase. This is the provider-sink mock --
 * it cannot be a real spawn in a test -- and stays installed in every case
 * below, including the real-binary escalation cases. Escalation dispatches
 * (Oracle) are unrelated to `contentByPhase` and are simply recorded and
 * marked completed. */
function installDispatchMock(
  harness: RoundHarness,
  repoRoot: string,
  contentByPhase: Map<number, (round: number) => string>
): void {
  const roundCountByPhase = new Map<number, number>();
  __setDispatchWorkers(async (_opts: DispatchOptions, dispatches: BuildDispatch[]): Promise<DispatchResult[]> => {
    harness.rounds.push(dispatches.map((d) => d.name));
    const results: DispatchResult[] = [];
    for (const dispatch of dispatches) {
      const phaseId = phaseIdFromDispatch(dispatch);
      const result: DispatchResult = { name: dispatch.name, status: "completed", summary: "Research complete" };
      if (phaseId !== undefined) {
        const round = (roundCountByPhase.get(phaseId) ?? 0) + 1;
        roundCountByPhase.set(phaseId, round);
        const contentFn = contentByPhase.get(phaseId);
        if (contentFn) {
          writePhaseResearch(repoRoot, phaseId, contentFn(round));
        }
      }
      results.push(result);
    }
    return results;
  });
}

function captureStderr(harness: RoundHarness): () => void {
  const original = process.stderr.write.bind(process.stderr);
  process.stderr.write = ((chunk: unknown, ...args: unknown[]) => {
    if (typeof chunk === "string") harness.stderrOutput += chunk;
    return original(chunk as string | Uint8Array, ...(args as [BufferEncoding]));
  }) as typeof process.stderr.write;
  return () => {
    process.stderr.write = original;
  };
}

/** Install a fake `_callGoJSONRef` that answers `plan-research-escalate`
 * with a valid oracle dispatch, for the happy-path case only. */
function installSuccessfulEscalateMock(calls: string[][]): void {
  __setCallGoJSON((<T>(_opts: GoBridgeOptions, args: string[]): T => {
    if (args[0] !== "plan-research-escalate") {
      throw new Error(`unexpected callGoJSON invocation in escalation-degrade test: ${args.join(" ")}`);
    }
    calls.push(args);
    const phaseFlagIdx = args.indexOf("--phase");
    const phaseId = phaseFlagIdx >= 0 ? parseInt(args[phaseFlagIdx + 1]!, 10) : 0;
    return {
      dispatch: {
        stage: "phase_research",
        caste: "oracle",
        agent_name: "aether-oracle",
        name: `Oracle-Escalated-phase-${phaseId}`,
        task: `Escalated Oracle research for phase ${phaseId}`,
        task_id: `plan-research-escalate-phase-${phaseId}`,
        outputs: [`phase-${phaseId}-research.md`],
        status: "planned",
      },
    } as T;
  }) as typeof callGoJSON);
}

describe("runResearchConfidenceLoop escalation degrade (CR-03, D-08)", () => {
  let harness: RoundHarness;
  let restoreStderr: () => void;

  beforeEach(() => {
    __restoreDispatchWorkers();
    __restoreCallGoJSON();
    harness = createHarness();
    restoreStderr = captureStderr(harness);
  });

  afterEach(() => {
    restoreStderr();
    __restoreDispatchWorkers();
    __restoreCallGoJSON();
  });

  it("escalation failure does not kill the plan run", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(harness, repoRoot, new Map([[7, () => MEDIUM_MARKDOWN]]));
    // Deliberately no __setCallGoJSON mock: the real plan-research-escalate
    // binary runs against repoRoot, which has no colony state, so the real
    // execFileSync in go-bridge.ts genuinely throws.

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "deep", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: process.env["AETHER_BINARY_PATH"]!, cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [researchDispatch(7, "Scout-Antenna-07")]);

    assert.equal(summary.phases.length, 1, "the summary still contains the stalled phase");
    assert.equal(summary.phases[0]!.phaseId, 7);
    assert.equal(summary.phases[0]!.stopReason, "diminishing_returns");
    assert.ok(summary.phases[0]!.finalConfidence < 95, "confidence remains below the deep target (95)");
    assert.deepEqual(summary.escalations, [], "a failed escalation must not be recorded as successful");

    assert.match(
      harness.stderrOutput,
      /phase 7/,
      "the warning names the phase"
    );
    assert.match(
      harness.stderrOutput,
      /Oracle escalation (is )?unavailable|escalation failed|escalation unavailable/i,
      "the warning states Oracle escalation is unavailable"
    );
    assert.match(
      harness.stderrOutput,
      /continu/i,
      "the warning states planning continues"
    );
  });

  it("a failing escalation for one phase does not suppress another phase's escalation", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(
      harness,
      repoRoot,
      new Map([
        [8, () => MEDIUM_MARKDOWN],
        [9, () => MEDIUM_MARKDOWN],
      ])
    );
    __restoreCallGoJSON();

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "deep", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: process.env["AETHER_BINARY_PATH"]!, cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [
      researchDispatch(8, "Scout-Antenna-08"),
      researchDispatch(9, "Scout-Antenna-09"),
    ]);

    const phaseIds = summary.phases.map((p) => p.phaseId).sort();
    assert.deepEqual(phaseIds, [8, 9], "both stalled phases are attempted and reported");
    for (const phase of summary.phases) {
      assert.equal(phase.stopReason, "diminishing_returns");
    }
    assert.deepEqual(summary.escalations, [], "neither failed escalation is recorded as successful");
    assert.match(harness.stderrOutput, /phase 8/);
    assert.match(harness.stderrOutput, /phase 9/);
  });

  it("successful escalation still dispatches", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(harness, repoRoot, new Map([[10, () => MEDIUM_MARKDOWN]]));
    const calls: string[][] = [];
    installSuccessfulEscalateMock(calls);

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "deep", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [researchDispatch(10, "Scout-Antenna-10")]);

    assert.equal(calls.length, 1, "exactly one plan-research-escalate call");
    assert.deepEqual(summary.escalations, [10], "the successful escalation surfaces in the summary");
    assert.ok(
      harness.rounds.some((round) => round.includes("Oracle-Escalated-phase-10")),
      "the Oracle escalation dispatch is still passed to the dispatcher"
    );
  });
});
