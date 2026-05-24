import type {
  Platform,
  PlatformAdapter,
  DispatchParams,
  DispatchResult,
  AvailabilityStatus,
  HealthStatus,
} from "./types.js";

export class ClaudeAdapter implements PlatformAdapter {
  readonly platform: Platform = "claude";

  async dispatch(params: DispatchParams): Promise<DispatchResult> {
    const { config } = params;
    const result = {
      workerName: config.workerName,
      caste: config.caste,
      taskID: config.taskID,
      status: "completed" as const,
      summary: "Claude Code dispatch completed (stub)",
      filesCreated: [],
      filesModified: [],
      testsWritten: [],
      artifacts: {},
      toolCount: 0,
      blockers: [],
      spawns: [],
      duration: 0,
      rawOutput: "",
    };
    return { result, platform: this.platform };
  }

  async preflight(): Promise<AvailabilityStatus> {
    return {
      platform: this.platform,
      available: true,
      category: "available",
      reason: "stub adapter",
    };
  }

  async health(): Promise<HealthStatus> {
    return { status: "healthy" };
  }
}
