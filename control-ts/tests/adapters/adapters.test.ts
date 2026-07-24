import { describe, it, expect } from "vitest";
import {
  createAdapter,
  getAvailableAdapters,
} from "../../src/adapters/index.js";
import { ClaudeAdapter } from "../../src/adapters/claude.js";
import { CodexAdapter } from "../../src/adapters/codex.js";
import { OpenCodeAdapter } from "../../src/adapters/opencode.js";
import { McpAdapter } from "../../src/adapters/mcp.js";
import type { WorkerConfig, PlatformAdapter } from "../../src/adapters/types.js";

const baseConfig: WorkerConfig = {
  agentName: "test-agent",
  agentTOMLPath: "/tmp/test.toml",
  caste: "builder",
  workerName: "TestWorker",
  taskID: "task-123",
  taskBrief: "test brief",
  contextCapsule: "capsule",
  root: "/tmp",
  timeout: 30000,
  skillSection: "",
  pheromoneSection: "",
  handoffSection: "",
  configOverrides: [],
};

describe("retired adapters", () => {
  it("every retired adapter refuses to dispatch instead of reporting completion", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      await expect(
        adapter.dispatch({ platform: adapter.platform, config: baseConfig })
      ).rejects.toThrow("retired experimental control plane");
    }
  });

  it("all adapters implement PlatformAdapter interface", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      expect(typeof adapter.dispatch).toBe("function");
      expect(typeof adapter.preflight).toBe("function");
      expect(typeof adapter.health).toBe("function");
      expect(adapter.platform).toBeDefined();
    }
  });

  it("preflight reports every retired adapter unavailable", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      const status = await adapter.preflight();
      expect(status.available).toBe(false);
      expect(status.category).toBe("provider_config_invalid");
    }
  });

  it("health reports every retired adapter unhealthy", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      const h = await adapter.health();
      expect(h.status).toBe("unhealthy");
    }
  });

  it("createAdapter returns correct instances", () => {
    expect(createAdapter("claude")).toBeInstanceOf(ClaudeAdapter);
    expect(createAdapter("codex")).toBeInstanceOf(CodexAdapter);
    expect(createAdapter("opencode")).toBeInstanceOf(OpenCodeAdapter);
    expect(createAdapter("mcp")).toBeInstanceOf(McpAdapter);
  });

  it("getAvailableAdapters returns no production adapters", () => {
    expect(getAvailableAdapters()).toEqual([]);
  });

  it("dispatching with an invalid platform throws Unknown platform", () => {
    expect(() => createAdapter("unknown" as any)).toThrow("Unknown platform");
  });
});
