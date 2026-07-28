/**
 * Ceremony snapshot tests.
 *
 * Captures renderer output for all ceremony templates and compares against
 * stored baseline snapshots in test/__snapshots__/*.txt.
 *
 * Snapshot update: AETHER_UPDATE_SNAPSHOTS=1 npm test
 */

import { execFileSync } from "node:child_process";
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { visualRenderer } from "../src/renderers/visual.js";
import { markdownRenderer } from "../src/renderers/markdown.js";
import { jsonRenderer } from "../src/renderers/json.js";
import { loadCeremonyConfig } from "../src/caste-config.js";
import {
  GoCeremonyAdapter,
  type CeremonyCommandRunner,
} from "../src/ceremony-adapter.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Derived from this file's location (test/ -> ts-host/ -> .aether/ -> repo).
// This was a hardcoded developer-machine path; on CI the cwd did not exist and
// every Go spawn died with ENOENT — the suite could only ever pass on the one
// laptop the path pointed at.
const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..");

// The Go binary shows a one-time first-run welcome banner in visual mode when
// a checkout has no colony data (cmd/ux_firstrun.go), then drops a .welcomed
// marker. Developer machines have colony state so the banner never fires; on a
// fresh CI checkout it prepends itself to the FIRST ceremony render and breaks
// that snapshot only. Seed the marker so the fixture is declared rather than
// inherited from whichever machine runs the tests. (.aether/data is local-only
// and gitignored; the marker is the runtime's own suppression mechanism.)
{
  const dataDir = join(REPO_ROOT, ".aether", "data");
  mkdirSync(dataDir, { recursive: true });
  const welcomeMarker = join(dataDir, ".welcomed");
  if (!existsSync(welcomeMarker)) {
    writeFileSync(welcomeMarker, "");
  }
}
const config = loadCeremonyConfig(REPO_ROOT);
const goCeremonyRunner: CeremonyCommandRunner = (_opts, args) => {
  const env: NodeJS.ProcessEnv = {
    ...process.env,
    AETHER_OUTPUT_MODE: "visual",
    NO_COLOR: "1",
  };
  delete env["AETHER_FORCE_COLOR"];
  delete env["CLICOLOR_FORCE"];
  return execFileSync("go", ["run", "./cmd/aether", ...args], {
    cwd: REPO_ROOT,
    env,
    encoding: "utf-8",
    maxBuffer: 10 * 1024 * 1024,
    stdio: ["ignore", "pipe", "pipe"],
  });
};
const goCeremony = new GoCeremonyAdapter(
  { goBinaryPath: "go", cwd: REPO_ROOT },
  goCeremonyRunner
);

const buildManifestEnvelope = {
  dispatch_manifest: {
    phase: 3,
    phase_name: "Ceremony Contract",
    dispatches: [
      {
        name: "Bolt-69",
        caste: "builder",
        task: "Align TS ceremony adapter lifecycle snapshots with Go contract names",
        execution_wave: 11,
        status: "planned",
      },
    ],
    execution_plan: [
      {
        execution_wave: 11,
        stage: "wave",
        strategy: "parallel",
        worker_count: 1,
        reason: "Phase 3 task 3.3",
      },
    ],
  },
};

const continueManifestEnvelope = {
  continue_manifest: {
    phase: 3,
    phase_name: "Ceremony Contract",
    dispatches: [
      {
        name: "Keen-37",
        caste: "watcher",
        task: "Verify TS ceremony adapter lifecycle snapshots",
        execution_wave: 12,
        status: "planned",
      },
    ],
    execution_plan: [
      {
        execution_wave: 12,
        stage: "verification",
        strategy: "serial",
        worker_count: 1,
        reason: "Verify worker result evidence",
      },
    ],
  },
  dispatches: [
    {
      name: "Keen-37",
      caste: "watcher",
      task: "Verify TS ceremony adapter lifecycle snapshots",
      execution_wave: 12,
      status: "planned",
    },
  ],
};

const SNAPSHOT_DIR = join(__dirname, "__snapshots__");
const UPDATE_SNAPSHOTS = process.env["AETHER_UPDATE_SNAPSHOTS"] === "1";

// ---------------------------------------------------------------------------
// Snapshot helpers
// ---------------------------------------------------------------------------

function loadSnapshot(name: string): string | undefined {
  const path = join(SNAPSHOT_DIR, `${name}.txt`);
  if (!existsSync(path)) return undefined;
  return readFileSync(path, "utf-8");
}

function saveSnapshot(name: string, content: string): void {
  if (!existsSync(SNAPSHOT_DIR)) {
    mkdirSync(SNAPSHOT_DIR, { recursive: true });
  }
  const path = join(SNAPSHOT_DIR, `${name}.txt`);
  writeFileSync(path, content, "utf-8");
}

function assertSnapshot(name: string, actual: string): void {
  if (UPDATE_SNAPSHOTS) {
    saveSnapshot(name, actual);
    return;
  }

  const expected = loadSnapshot(name);
  if (expected === undefined) {
    saveSnapshot(name, actual);
    process.stderr.write(
      `Warning: created missing snapshot ${name}.txt (run with AETHER_UPDATE_SNAPSHOTS=1 to regenerate)\n`
    );
    return;
  }

  if (actual !== expected) {
    const lines = actual.split("\n");
    const expectedLines = expected.split("\n");
    const maxLen = Math.max(lines.length, expectedLines.length);
    const diff: string[] = [];
    for (let i = 0; i < maxLen; i++) {
      const a = lines[i] ?? "(missing)";
      const e = expectedLines[i] ?? "(missing)";
      if (a !== e) {
        diff.push(`  line ${i + 1}:`);
        diff.push(`    expected: ${JSON.stringify(e)}`);
        diff.push(`    actual:   ${JSON.stringify(a)}`);
      }
    }
    throw new Error(
      `Snapshot mismatch for ${name}.txt\n${diff.join("\n")}\n\n` +
        `Run AETHER_UPDATE_SNAPSHOTS=1 to regenerate snapshots.`
    );
  }
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("ceremony snapshots", () => {
  it("renderBanner(BUILD) matches snapshot", () => {
    const result = visualRenderer.renderBanner("BUILD");
    assertSnapshot("banner-build-start", result);
  });

  it("renderBanner(CROWNED ANTHILL) matches snapshot", () => {
    const result = visualRenderer.renderBanner("CROWNED ANTHILL");
    assertSnapshot("banner-seal-complete", result);
  });

  it("Go renderSpawnPlan(build) matches snapshot", () => {
    const result = goCeremony.renderSpawnPlan("build", buildManifestEnvelope);
    assertSnapshot("spawn-frame-builder", result);
  });

  it("renderSpawnFrame(oracle) matches snapshot", () => {
    const result = visualRenderer.renderSpawnFrame(
      { caste: "oracle", name: "Seer-42", task: "Research patterns" },
      config
    );
    assertSnapshot("spawn-frame-oracle", result);
  });

  it("Go renderWaveStart(build) matches snapshot", () => {
    const result = goCeremony.renderWaveStart("build", buildManifestEnvelope, 11);
    assertSnapshot("stage-separator-build", result);
  });

  it("Go renderWaveStart(continue) matches snapshot", () => {
    const result = goCeremony.renderWaveStart(
      "continue",
      continueManifestEnvelope,
      12
    );
    assertSnapshot("stage-separator-continue", result);
  });

  it("renderBox(build-summary) matches snapshot", () => {
    const result = visualRenderer.renderBox(
      "Workers: 2 completed  0 blocked  0 failed  (2 total)\nWorker Results\n✓ Builder Bolt-69  Task 3.3 — Adapter snapshots aligned\n✓ Watcher Keen-37 — Verified focused ceremony tests",
      { borderStyle: "round", borderColor: "green" }
    );
    assertSnapshot("build-summary", result);
  });

  it("renderBox(closeout-ritual) matches snapshot", () => {
    const result = visualRenderer.renderBox(
      "Phase 1\nStatus: Crowned Anthill",
      { borderStyle: "double", borderColor: "cyan" }
    );
    assertSnapshot("closeout-ritual", result);
  });

  it("markdownRenderer strips ANSI while preserving structure", () => {
    const result = markdownRenderer.renderBanner("TEST");
    assert.ok(!result.includes("\x1b["), "Markdown should not contain ANSI codes");
    // Should still contain figlet art characters
    assert.ok(/[|_\\/()]/.test(result), "Should preserve figlet structure");
  });

  it("jsonRenderer returns empty strings for all methods", () => {
    assert.equal(jsonRenderer.renderBanner("TEST"), "");
    assert.equal(
      jsonRenderer.renderSpawnFrame(
        { caste: "builder", name: "Mason-67", task: "Build the wall" },
        config
      ),
      ""
    );
    assert.equal(jsonRenderer.renderStageSeparator("Build", config), "");
    assert.equal(jsonRenderer.renderBox("Hello"), "");
  });
});
