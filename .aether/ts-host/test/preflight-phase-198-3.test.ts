import { afterEach, describe, it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";

import type { GoBridgeOptions } from "../src/go-bridge.js";
import {
  __restoreCallGoJSONAsync,
  __setCallGoJSONAsync,
} from "../src/go-bridge.js";
import {
  preflightGoWorkerProvider,
  type GoWorkerAdapterResponse,
} from "../src/worker-dispatch.js";
import { resolvePreflightTimeoutMs } from "../src/preflight-config.js";

function successfulPreflightResponse(): GoWorkerAdapterResponse {
  return {
    schema_version: 1,
    execution_owner: "go-adapter",
    platform: "fake",
    platform_contract: {},
    availability: { available: true },
    permission_decision: {
      schema_version: 1,
      platform: "fake",
      profile: {
        schema_version: 1,
        name: "workspace_write",
        filesystem: "workspace_write",
        shell: "within_filesystem_boundary",
        network: "provider_default",
        approval: "never",
      },
      allowed: true,
      enforcement: "simulated",
      mechanism: "test",
    },
  };
}

describe("phase-scoped hosted preflight", () => {
  afterEach(() => {
    __restoreCallGoJSONAsync();
    delete process.env["AETHER_PREFLIGHT_TIMEOUT"];
  });

  it("passes build and continue phase once while plan stays explicitly unscoped", async () => {
    const calls: string[][] = [];
    __setCallGoJSONAsync(async <T>(_opts: GoBridgeOptions, args: string[]): Promise<T> => {
      calls.push([...args]);
      return successfulPreflightResponse() as T;
    });

    const bridge: GoBridgeOptions = { goBinaryPath: "/usr/bin/true", cwd: process.cwd() };
    await preflightGoWorkerProvider(bridge, "Build phase 7", 7);
    await preflightGoWorkerProvider(bridge, "Continue", 7);
    await preflightGoWorkerProvider(bridge, "Plan", 0);

    assert.deepEqual(calls, [
      ["internal-worker-adapter", "--preflight", "--phase", "7"],
      ["internal-worker-adapter", "--preflight", "--phase", "7"],
      ["internal-worker-adapter", "--preflight", "--phase", "0"],
    ]);
  });

  it("threads manifest phase at active host call sites and leaves the legacy dispatcher deferred", () => {
    const host = readFileSync(new URL("../src/host.ts", import.meta.url), "utf-8");
    assert.match(host, /preflightHostWorkerDispatch\([\s\S]*?buildManifest\.phase[\s\S]*?\)/);
    assert.match(host, /preflightHostWorkerDispatch\(bridge, "Plan", 0\)/);
    assert.match(host, /preflightHostWorkerDispatch\(bridge, "Continue", continueManifest\.phase\)/);
    assert.ok(!host.includes('from "./platform-dispatcher.js"'));
  });

  it("keeps the active TypeScript timeout on AETHER_PREFLIGHT_TIMEOUT with a 45-second default", () => {
    delete process.env["AETHER_PREFLIGHT_TIMEOUT"];
    assert.equal(resolvePreflightTimeoutMs(), 45_000);
    process.env["AETHER_PREFLIGHT_TIMEOUT"] = "9s";
    assert.equal(resolvePreflightTimeoutMs(), 9_000);
  });
});
