/**
 * Tests for `runResearchConfidenceLoop` (RESEARCH-07 / RESEARCH-08), the
 * construction site that gives the plan path a real per-phase
 * `ConfidenceLoop`, mirroring the build path's iteration shape.
 *
 * Drives the function directly with a fake dispatcher installed through
 * host.ts's existing dispatch-mock hook, and controls the confidence score
 * by writing real `phase-N-research.md` fixture files into a temp repo root
 * between rounds. Scoring goes through the real `ResearchConfidenceEvaluator`
 * (Plan 03) rather than a stub, keeping the test honest about that contract.
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";
import * as fs from "node:fs";
import * as os from "node:os";
import * as path from "node:path";
import { fileURLToPath } from "node:url";

import {
  parseArgs,
  __setDispatchWorkers,
  __restoreDispatchWorkers,
  runResearchConfidenceLoop,
} from "../src/host.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";
import type { BuildDispatch } from "../src/types.js";
import type { DispatchOptions, DispatchResult } from "../src/worker-dispatch.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

/** A fully-evidenced artifact (see research-confidence.test.ts) — scores 100
 * with gaps=0, and 85 with gaps=1, against a repo root containing the two
 * files it cites in "Files to Study". */
const FULL_MARKDOWN = `# Phase 999 Research: Test Phase

## Hive Wisdom (Pre-existing Knowledge)
No relevant hive wisdom found for this phase, but a similar pattern applied successfully in an earlier phase.

## Key Patterns
- **Evidence scoring:** confidence-evaluator.ts already implements a base/bonus/penalty/clamp shape worth mirroring (Source: .aether/ts-host/src/confidence-evaluator.ts)
- **Depth binding:** planningLoopPreset already encodes the four target/iteration tiers this phase needs to copy (Source: cmd/codex_plan.go)

## External Context
No external research needed for this phase.

## Gotchas
- **Loop conflation:** three separate confidence loops exist in this codebase and mixing them up would wire the wrong one (Source: .planning/phases/164-research-feeds-planning/164-RESEARCH.md)
- **Reproducibility:** using Date.now or Math.random would break the deterministic scoring requirement (Source: https://example.com/definition-of-done)

## Recommended Approach
Treat this as wiring existing pieces together rather than building new subsystems, mirroring the iteration pattern already proven elsewhere in the host.

## Files to Study
- confidence-loop.ts
- confidence-evaluator.ts
`;

/** Scores exactly 30 (base 20 + self-assessment 10, gaps=0, no sections). */
const LOW_MARKDOWN = "";

/** Scores exactly 35 (base 20 + one filled section (5) + self-assessment 10). */
const MEDIUM_MARKDOWN = `## Hive Wisdom (Pre-existing Knowledge)
Some prior wisdom text that is definitely long enough to count as filled content for the section-filled check.
`;

/** Create a temp repo root with the two files FULL_MARKDOWN cites as evidence. */
function makeFixtureRepo(): string {
  const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), "research-loop-"));
  fs.mkdirSync(path.join(repoRoot, ".aether", "data", "phase-research"), { recursive: true });
  fs.writeFileSync(path.join(repoRoot, "confidence-loop.ts"), "// fixture\n");
  fs.writeFileSync(path.join(repoRoot, "confidence-evaluator.ts"), "// fixture\n");
  return repoRoot;
}

function writePhaseResearch(repoRoot: string, phaseId: number, markdown: string): void {
  fs.writeFileSync(
    path.join(repoRoot, ".aether", "data", "phase-research", `phase-${phaseId}-research.md`),
    markdown
  );
}

/** Minimal research dispatch shape matching PlanningDispatch's required fields. */
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

/** Extract "plan-research-phase-<ID>" -> ID from a dispatched BuildDispatch. */
function phaseIdFromDispatch(dispatch: BuildDispatch): number | undefined {
  const match = /^plan-research-phase-(\d+)$/.exec(dispatch.task_id ?? "");
  return match ? parseInt(match[1]!, 10) : undefined;
}

interface RoundHarness {
  /** One entry per dispatch round; each entry lists the dispatched worker names. */
  rounds: string[][];
  stderrOutput: string;
}

function createHarness(): RoundHarness {
  return { rounds: [], stderrOutput: "" };
}

/**
 * Install a fake dispatcher that: (1) records which workers were dispatched
 * each round, (2) writes the next markdown fixture for each active phase
 * (looked up from `contentByPhase`, keyed by round number, 1-indexed),
 * (3) optionally reports blockers to drive selfAssessedGaps, and
 * (4) returns a "completed" DispatchResult for every dispatched worker.
 */
function installDispatchMock(
  harness: RoundHarness,
  repoRoot: string,
  contentByPhase: Map<number, (round: number) => { markdown: string; gaps?: number }>
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
          const { markdown, gaps } = contentFn(round);
          writePhaseResearch(repoRoot, phaseId, markdown);
          if (gaps) {
            result.blockers = Array.from({ length: gaps }, (_, i) => `gap-${i + 1}`);
          }
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

// ---------------------------------------------------------------------------
// Behavioural tests
// ---------------------------------------------------------------------------

describe("runResearchConfidenceLoop (RESEARCH-07 / RESEARCH-08)", () => {
  let harness: RoundHarness;
  let restoreStderr: () => void;

  beforeEach(() => {
    __restoreDispatchWorkers();
    harness = createHarness();
    restoreStderr = captureStderr(harness);
  });

  afterEach(() => {
    restoreStderr();
    __restoreDispatchWorkers();
  });

  it("zero research dispatches produce zero dispatch rounds and no ceremony", async () => {
    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "balanced", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: "/tmp" };

    const summary = await runResearchConfidenceLoop(bridge, parsed, []);

    assert.deepEqual(summary, { phases: [], escalations: [] });
    assert.equal(harness.rounds.length, 0, "no dispatch rounds should occur");
    assert.equal(harness.stderrOutput, "", "no ceremony lines should be emitted");
  });

  it("a never-improving phase at balanced depth produces exactly 6 dispatch rounds", async () => {
    const repoRoot = makeFixtureRepo();
    // Alternate between two low scores (30 <-> 35, delta magnitude 5) so the
    // loop never trips diminishing_returns (which needs abs(delta) < 5) and
    // never approaches the balanced target (90).
    installDispatchMock(
      harness,
      repoRoot,
      new Map([[1, (round: number) => ({ markdown: round % 2 === 0 ? MEDIUM_MARKDOWN : LOW_MARKDOWN })]])
    );

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "balanced", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [researchDispatch(1, "Scout-Antenna-01")]);

    assert.equal(harness.rounds.length, 6, "should dispatch exactly 6 rounds, never more");
    assert.equal(summary.phases.length, 1);
    assert.equal(summary.phases[0]!.phaseId, 1);
    assert.equal(summary.phases[0]!.iterations, 6);
    assert.equal(summary.phases[0]!.stopReason, "max_iterations_met");
  });

  it("a phase reaching target stops after one round while a sibling below target continues", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(
      harness,
      repoRoot,
      new Map([
        // Phase 1: fully-evidenced from round 1 -> scores 100, well above the
        // balanced target (90); stops after its first round.
        [1, () => ({ markdown: FULL_MARKDOWN })],
        // Phase 2: never improves, alternating 30/35 as in the test above.
        [2, (round: number) => ({ markdown: round % 2 === 0 ? MEDIUM_MARKDOWN : LOW_MARKDOWN })],
      ])
    );

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "balanced", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [
      researchDispatch(1, "Scout-Antenna-01"),
      researchDispatch(2, "Scout-Antenna-02"),
    ]);

    // Round 1 dispatches both phases; round 2 onward dispatches phase 2 only.
    assert.ok(harness.rounds.length >= 2, "phase 2 should still be iterating after round 1");
    assert.deepEqual(
      new Set(harness.rounds[0]),
      new Set(["Scout-Antenna-01", "Scout-Antenna-02"]),
      "round 1 dispatches both phases"
    );
    for (let i = 1; i < harness.rounds.length; i++) {
      assert.ok(
        !harness.rounds[i]!.includes("Scout-Antenna-01"),
        `round ${i + 1} should not re-dispatch phase 1's Scout`
      );
      assert.ok(
        harness.rounds[i]!.includes("Scout-Antenna-02"),
        `round ${i + 1} should still dispatch phase 2's Scout`
      );
    }

    const phase1Summary = summary.phases.find((p) => p.phaseId === 1)!;
    assert.equal(phase1Summary.iterations, 1);
    assert.equal(phase1Summary.stopReason, "confidence_target_met");
    assert.equal(phase1Summary.finalConfidence, 100);
  });

  it("--accept stops every phase after its first iteration with stop reason accepted", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(
      harness,
      repoRoot,
      new Map([
        [1, () => ({ markdown: LOW_MARKDOWN })],
        [2, () => ({ markdown: LOW_MARKDOWN })],
      ])
    );

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "balanced", "--accept", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [
      researchDispatch(1, "Scout-Antenna-01"),
      researchDispatch(2, "Scout-Antenna-02"),
    ]);

    assert.equal(harness.rounds.length, 1, "accept should stop every phase after one round");
    assert.equal(summary.phases.length, 2);
    for (const phase of summary.phases) {
      assert.equal(phase.iterations, 1);
      assert.equal(phase.stopReason, "accepted");
    }
  });

  it("emits a ceremony line per phase per iteration with phase number, Scout name, and confidence", async () => {
    const repoRoot = makeFixtureRepo();
    installDispatchMock(
      harness,
      repoRoot,
      new Map([[7, (round: number) => ({ markdown: round % 2 === 0 ? MEDIUM_MARKDOWN : LOW_MARKDOWN })]])
    );

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "fast", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    await runResearchConfidenceLoop(bridge, parsed, [researchDispatch(7, "Scout-Antenna-07")]);

    const iterationLines = harness.stderrOutput
      .split("\n")
      .filter((line) => line.includes("Iteration") && line.includes("confidence"));
    assert.equal(iterationLines.length, 4, "fast depth caps at 4 iterations, one ceremony line each");
    for (const line of iterationLines) {
      assert.match(line, /Scout-Antenna-07 phase 7/);
      assert.match(line, /confidence \d+%/);
    }
  });

  it("the early-accept prompt appears at most once per phase across the whole run", async () => {
    const repoRoot = makeFixtureRepo();
    // Score 85 every round (gaps=1 against the fully-evidenced fixture):
    // within 5 of the balanced target (90) from round 1 (near-target trigger),
    // then trips diminishing_returns once enough flat history accumulates
    // (stalled-below-target trigger) -- both should collapse to one prompt.
    installDispatchMock(
      harness,
      repoRoot,
      new Map([[3, () => ({ markdown: FULL_MARKDOWN, gaps: 1 })]])
    );

    const parsed = parseArgs(["node", "host.js", "plan", "--depth", "balanced", "--simulate"]);
    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/aether", cwd: repoRoot };

    const summary = await runResearchConfidenceLoop(bridge, parsed, [researchDispatch(3, "Scout-Antenna-03")]);

    assert.equal(summary.phases[0]!.stopReason, "diminishing_returns");
    assert.equal(summary.phases[0]!.finalConfidence, 85);

    const promptLines = harness.stderrOutput
      .split("\n")
      .filter((line) => line.includes("accept now or keep digging"));
    assert.equal(promptLines.length, 1, "the early-accept prompt should appear exactly once");
    assert.match(promptLines[0]!, /phase 3 research at 85%/);
  });
});

// ---------------------------------------------------------------------------
// Construction-site invariant (regression guard for RESEARCH-07)
// ---------------------------------------------------------------------------

describe("host.ts construction-site invariant", () => {
  it("constructs at least two ConfidenceLoop instances (build path + research path)", () => {
    const hostSource = fs.readFileSync(path.join(__dirname, "..", "src", "host.ts"), "utf8");
    const codeLines = hostSource
      .split("\n")
      .filter((line) => {
        const trimmed = line.trim();
        return trimmed !== "" && !trimmed.startsWith("//") && !trimmed.startsWith("*");
      });
    const occurrences = codeLines.filter((line) => line.includes("new ConfidenceLoop(")).length;
    assert.ok(
      occurrences >= 2,
      `expected at least 2 non-comment 'new ConfidenceLoop(' construction sites, found ${occurrences}. ` +
        "If this dropped to 1, the research construction site (RESEARCH-07) was deleted."
    );
  });
});
