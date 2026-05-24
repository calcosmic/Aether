/**
 * Adapter registry and factory for platform adapters.
 */

export type {
  Platform,
  AvailabilityCategory,
  AvailabilityStatus,
  WorkerConfig,
  WorkerResult,
  DispatchParams,
  DispatchResult,
  HealthStatus,
  PlatformAdapter,
} from "./types.js";

import { ClaudeAdapter } from "./claude.js";
import { CodexAdapter } from "./codex.js";
import { OpenCodeAdapter } from "./opencode.js";
import { McpAdapter } from "./mcp.js";
import type { Platform, PlatformAdapter } from "./types.js";

const registry: Record<string, new () => PlatformAdapter> = {
  claude: ClaudeAdapter,
  codex: CodexAdapter,
  opencode: OpenCodeAdapter,
  mcp: McpAdapter,
};

export function createAdapter(platform: Platform): PlatformAdapter {
  const AdapterClass = registry[platform];
  if (!AdapterClass) {
    throw new Error("Unknown platform: " + platform);
  }
  return new AdapterClass();
}

export function getAvailableAdapters(): Platform[] {
  return Object.keys(registry) as Platform[];
}
