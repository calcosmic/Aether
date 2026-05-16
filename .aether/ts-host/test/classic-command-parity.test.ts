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
    const hostCommands = ["plan", "build", "continue", "oracle", "swarm", "watch"];

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
    ];

    for (const testCase of cases) {
      assert.deepEqual(buildHostGoArgs(parseArgs(testCase.argv)), testCase.want, testCase.argv.join(" "));
    }
  });

  it("keeps colonize and seal marked as missing TS host orchestration targets", () => {
    const records = recordsByName();
    assert.equal(records.get("colonize")?.ts_host_surface, "missing-orchestration-target");
    assert.equal(records.get("seal")?.ts_host_surface, "missing-orchestration-target");
  });
});
