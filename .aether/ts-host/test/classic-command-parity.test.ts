import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { buildHostGoArgs, parseArgs } from "../src/host.js";

interface ClassicCommandRecord {
  name: string;
  category: string;
  runtime_command: string;
  ts_host_surface: string;
  state_mutation_owner: string;
  mutates_aether_data: boolean;
  ts_write_policy: string;
  finalizer_required: boolean;
  finalizer_command: string;
  ceremony_steps: string[];
}

interface ClassicCommandMatrix {
  runtime_authority: string;
  commands: ClassicCommandRecord[];
}

const requiredCommands = [
  "init",
  "discuss",
  "colonize",
  "plan",
  "build",
  "continue",
  "seal",
  "oracle",
  "swarm",
  "watch",
  "status",
  "resume",
  "focus",
  "redirect",
  "feedback",
  "pheromones",
];

function loadMatrix(): ClassicCommandMatrix {
  const here = dirname(fileURLToPath(import.meta.url));
  const matrixPath = join(here, "..", "..", "commands", "classic-command-parity.json");
  return JSON.parse(readFileSync(matrixPath, "utf8")) as ClassicCommandMatrix;
}

function recordsByName(): Map<string, ClassicCommandRecord> {
  const records = new Map<string, ClassicCommandRecord>();
  for (const record of loadMatrix().commands) {
    assert.ok(!records.has(record.name), `duplicate command in matrix: ${record.name}`);
    records.set(record.name, record);
  }
  return records;
}

describe("classic command parity matrix", () => {
  it("covers the classic lifecycle and signal commands", () => {
    const records = recordsByName();

    for (const command of requiredCommands) {
      assert.ok(records.has(command), `matrix should include ${command}`);
    }
  });

  it("documents that TypeScript is conductor, not state authority", () => {
    const matrix = loadMatrix();
    assert.match(matrix.runtime_authority, /not runtime authority/);

    for (const record of matrix.commands) {
      assert.equal(record.ts_write_policy, "never-write-aether-data", `${record.name} TS write policy`);
      assert.notEqual(record.state_mutation_owner, "typescript-host", `${record.name} state owner`);
      if (record.mutates_aether_data) {
        assert.match(record.state_mutation_owner, /^go-/, `${record.name} state owner`);
      }
      if (record.finalizer_required) {
        assert.match(record.finalizer_command, /--completion-file/, `${record.name} finalizer command`);
      }
      assert.ok(record.ceremony_steps.length > 0, `${record.name} ceremony steps`);
    }
  });

  it("matches host-supported command names parsed by the TS host", () => {
    const records = recordsByName();
    const hostCommands = ["colonize", "plan", "build", "continue", "seal", "oracle", "swarm", "watch"];

    for (const command of hostCommands) {
      const record = records.get(command);
      assert.ok(record, `matrix should include host command ${command}`);
      assert.notEqual(record.ts_host_surface, "none", `${command} should document its host surface`);
      const argv = command === "build" ? ["node", "host.js", command, "1"] : ["node", "host.js", command];
      assert.equal(parseArgs(argv).command, command);
    }
  });

  it("uses actual host routing logic for plan/build/continue option pass-through", () => {
    const cases: Array<{ argv: string[]; want: string[] }> = [
      {
        argv: [
          "node",
          "host.js",
          "plan",
          "--depth",
          "deep",
          "--planning-depth",
          "standard",
          "--verification-depth",
          "heavy",
          "--worker-timeout",
          "5m",
        ],
        want: [
          "plan",
          "--plan-only",
          "--depth",
          "deep",
          "--planning-depth",
          "standard",
          "--verification-depth",
          "heavy",
          "--worker-timeout",
          "5m",
        ],
      },
      {
        argv: ["node", "host.js", "build", "1", "--light", "--heavy", "--verification-depth", "heavy", "--simulate", "--worker-timeout", "10m"],
        want: [
          "build",
          "1",
          "--plan-only",
          "--synthetic",
          "--light",
          "--heavy",
          "--verification-depth",
          "heavy",
          "--worker-timeout",
          "10m",
        ],
      },
      {
        argv: ["node", "host.js", "continue", "--verification-depth=heavy", "--light", "--heavy", "--skip-watchers"],
        want: ["continue", "--plan-only", "--verification-depth", "heavy", "--light", "--heavy", "--skip-watchers"],
      },
      {
        argv: ["node", "host.js", "colonize", "--force-resurvey", "--worker-timeout", "5m"],
        want: ["colonize", "--plan-only", "--force-resurvey", "--worker-timeout", "5m"],
      },
      {
        argv: ["node", "host.js", "continue", "--classic-ceremony"],
        want: ["continue", "--plan-only", "--classic-ceremony"],
      },
      {
        argv: ["node", "host.js", "seal", "--force"],
        want: ["seal", "--plan-only", "--force"],
      },
    ];

    for (const testCase of cases) {
      assert.deepEqual(buildHostGoArgs(parseArgs(testCase.argv)), testCase.want, testCase.argv.join(" "));
    }
  });

  it("marks colonize and seal as implemented TS host orchestration targets", () => {
    const records = recordsByName();
    assert.equal(records.get("colonize")?.ts_host_surface, "orchestration-manifest");
    assert.equal(records.get("seal")?.ts_host_surface, "orchestration-manifest");
  });
});

// ---------------------------------------------------------------------------
// Ceremony Marker Presence from Parity Matrix (D-02)
// ---------------------------------------------------------------------------

describe("ceremony marker presence from parity matrix", () => {
  it("dispatched commands have full ceremony lifecycle in parity matrix", () => {
    const records = recordsByName();

    // Commands whose wrappers use the 4-stage dispatched ceremony:
    // spawn-plan, wave-start, worker-complete, closeout
    const dispatchedWithFullLifecycle = ["colonize", "plan", "build"];
    for (const command of dispatchedWithFullLifecycle) {
      const record = records.get(command);
      assert.ok(record, `matrix should include ${command}`);
      assert.ok(
        record.ceremony_steps.includes("spawn-plan"),
        `${command} should have spawn-plan ceremony step`
      );
      assert.ok(
        record.ceremony_steps.includes("wave-start"),
        `${command} should have wave-start ceremony step`
      );
      assert.ok(
        record.ceremony_steps.includes("worker-complete"),
        `${command} should have worker-complete ceremony step`
      );
      assert.ok(
        record.ceremony_steps.includes("closeout"),
        `${command} should have closeout ceremony step`
      );
    }

    // swarm has its own wave structure (spawn-plan, investigation-wave, fix-wave,
    // verification-wave, closeout) but still has spawn-plan and closeout bookends
    const swarmRecord = records.get("swarm")!;
    assert.ok(swarmRecord.ceremony_steps.includes("spawn-plan"), "swarm should have spawn-plan");
    assert.ok(swarmRecord.ceremony_steps.includes("closeout"), "swarm should have closeout");
    assert.equal(swarmRecord.ceremony_steps.length, 5, "swarm should have 5 ceremony steps");

    // continue has its own ceremony lifecycle (verification, gates, advance-or-block)
    const continueRecord = records.get("continue")!;
    assert.deepEqual(
      continueRecord.ceremony_steps,
      ["verification", "gates", "advance-or-block"],
      "continue should have verification ceremony lifecycle"
    );

    // seal has its own ceremony lifecycle (final-review, worker-complete, closeout, porter-readiness)
    const sealRecord = records.get("seal")!;
    assert.ok(
      sealRecord.ceremony_steps.includes("final-review"),
      "seal should have final-review ceremony step"
    );
    assert.ok(
      sealRecord.ceremony_steps.includes("closeout"),
      "seal should have closeout ceremony step"
    );

    // oracle has its own ceremony lifecycle (research-scope, iterate, confidence-check, promote-findings)
    const oracleRecord = records.get("oracle")!;
    assert.deepEqual(
      oracleRecord.ceremony_steps,
      ["research-scope", "iterate", "confidence-check", "promote-findings"],
      "oracle should have research ceremony lifecycle"
    );
  });

  it("commands with spawn-plan ceremony step have closeout too", () => {
    const matrix = loadMatrix();
    for (const record of matrix.commands) {
      if (record.ceremony_steps.includes("spawn-plan")) {
        assert.ok(
          record.ceremony_steps.includes("closeout"),
          `${record.name} has spawn-plan but is missing closeout -- lifecycle gap`
        );
      }
    }
  });

  it("all 18 commands have non-empty ceremony steps", () => {
    const matrix = loadMatrix();
    assert.equal(
      matrix.commands.length,
      18,
      `parity matrix should have 18 commands, found ${matrix.commands.length}`
    );
    for (const record of matrix.commands) {
      assert.ok(
        record.ceremony_steps.length > 0,
        `${record.name} should have non-empty ceremony_steps`
      );
    }
  });
});
