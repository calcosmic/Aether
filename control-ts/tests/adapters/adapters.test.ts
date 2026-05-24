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

describe("Adapter stubs", () => {
  it("ClaudeAdapter.dispatch returns completed result with matching taskID", async () => {
    const adapter = new ClaudeAdapter();
    const { result } = await adapter.dispatch({ platform: "claude", config: baseConfig });
    expect(result.status).toBe("completed");
    expect(result.taskID).toBe(baseConfig.taskID);
    expect(result.workerName).toBe(baseConfig.workerName);
    expect(result.caste).toBe(baseConfig.caste);
  });

  it("CodexAdapter.dispatch returns completed result with matching taskID", async () => {
    const adapter = new CodexAdapter();
    const { result } = await adapter.dispatch({ platform: "codex", config: baseConfig });
    expect(result.status).toBe("completed");
    expect(result.taskID).toBe(baseConfig.taskID);
  });

  it("OpenCodeAdapter.dispatch returns completed result with matching taskID", async () => {
    const adapter = new OpenCodeAdapter();
    const { result } = await adapter.dispatch({ platform: "opencode", config: baseConfig });
    expect(result.status).toBe("completed");
    expect(result.taskID).toBe(baseConfig.taskID);
  });

  it("McpAdapter.dispatch returns completed result with matching taskID", async () => {
    const adapter = new McpAdapter();
    const { result } = await adapter.dispatch({ platform: "mcp", config: baseConfig });
    expect(result.status).toBe("completed");
    expect(result.taskID).toBe(baseConfig.taskID);
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

  it("preflight returns available=true for stubs", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      const status = await adapter.preflight();
      expect(status.available).toBe(true);
      expect(status.category).toBe("available");
    }
  });

  it("health returns healthy for all stubs", async () => {
    const adapters: PlatformAdapter[] = [
      new ClaudeAdapter(),
      new CodexAdapter(),
      new OpenCodeAdapter(),
      new McpAdapter(),
    ];
    for (const adapter of adapters) {
      const h = await adapter.health();
      expect(h.status).toBe("healthy");
    }
  });

  it("createAdapter returns correct instances", () => {
    expect(createAdapter("claude")).toBeInstanceOf(ClaudeAdapter);
    expect(createAdapter("codex")).toBeInstanceOf(CodexAdapter);
    expect(createAdapter("opencode")).toBeInstanceOf(OpenCodeAdapter);
    expect(createAdapter("mcp")).toBeInstanceOf(McpAdapter);
  });

  it("getAvailableAdapters returns all four platform names", () => {
    const platforms = getAvailableAdapters();
    expect(platforms).toContain("claude");
    expect(platforms).toContain("codex");
    expect(platforms).toContain("opencode");
    expect(platforms).toContain("mcp");
    expect(platforms.length).toBe(4);
  });

  it("dispatching with an invalid platform throws Unknown platform", () => {
    expect(() => createAdapter("unknown" as any)).toThrow("Unknown platform");
  });
});
