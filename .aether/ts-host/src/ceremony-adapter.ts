/**
 * Runtime ceremony adapter for the TypeScript host.
 *
 * The TS host may decide when ceremony surfaces should be shown, but the Go
 * runtime owns the visual rendering contract. This adapter writes temporary
 * JSON packets and calls `aether ceremony ...` with visual output enabled.
 */

import { execFileSync } from "node:child_process";

import type { GoBridgeOptions } from "./go-bridge.js";
import { writeCompletionFile } from "./go-bridge.js";

export type CeremonyWorkflow =
  | "plan"
  | "build"
  | "continue"
  | "colonize"
  | "swarm"
  | "seal";

export interface CeremonyAdapter {
  renderSpawnPlan(workflow: CeremonyWorkflow, manifestEnvelope: unknown): string;
  renderWaveStart(
    workflow: CeremonyWorkflow,
    manifestEnvelope: unknown,
    executionWave: number
  ): string;
  renderWorkerComplete(workflow: CeremonyWorkflow, workerResult: unknown): string;
  renderCloseout(workflow: CeremonyWorkflow, completionFile: string): string;
}

export type CeremonyCommandRunner = (
  opts: GoBridgeOptions,
  args: string[]
) => string;

const TERMINAL_WORKER_STATUSES = new Set([
  "completed",
  "failed",
  "blocked",
  "timeout",
  "manually-reconciled",
  "code_written",
]);

function runGoCeremonyCommand(opts: GoBridgeOptions, args: string[]): string {
  try {
    return execFileSync(opts.goBinaryPath, args, {
      cwd: opts.cwd,
      env: {
        ...process.env,
        AETHER_OUTPUT_MODE: "visual",
        AETHER_FORCE_COLOR: "1",
      },
      encoding: "utf-8",
      maxBuffer: 10 * 1024 * 1024,
      stdio: ["ignore", "pipe", "pipe"],
    });
  } catch (err: unknown) {
    const typed = err as { status?: unknown; code?: unknown; signal?: unknown };
    const detail =
      typeof typed.status === "number"
        ? `exit status ${typed.status}`
        : typeof typed.code === "string"
          ? typed.code
          : typeof typed.signal === "string"
            ? `signal ${typed.signal}`
            : "execution failed";
    throw new Error(
      `Go ceremony command failed for ${args.slice(0, 3).join(" ")}: ${detail}; subprocess output omitted`
    );
  }
}

function isTerminalWorkerResult(data: unknown): boolean {
  if (typeof data !== "object" || data === null || Array.isArray(data)) {
    return false;
  }
  const status = (data as Record<string, unknown>)["status"];
  return (
    typeof status === "string" &&
    TERMINAL_WORKER_STATUSES.has(status.trim().toLowerCase())
  );
}

export class GoCeremonyAdapter implements CeremonyAdapter {
  constructor(
    private readonly opts: GoBridgeOptions,
    private readonly runner: CeremonyCommandRunner = runGoCeremonyCommand
  ) {}

  renderSpawnPlan(workflow: CeremonyWorkflow, manifestEnvelope: unknown): string {
    const manifestFile = this.writePacket(workflow, "manifest", manifestEnvelope);
    return this.runner(this.opts, [
      "ceremony",
      "spawn-plan",
      "--workflow",
      workflow,
      "--manifest-file",
      manifestFile,
    ]);
  }

  renderWaveStart(
    workflow: CeremonyWorkflow,
    manifestEnvelope: unknown,
    executionWave: number
  ): string {
    const manifestFile = this.writePacket(workflow, "manifest", manifestEnvelope);
    return this.runner(this.opts, [
      "ceremony",
      "wave-start",
      "--workflow",
      workflow,
      "--manifest-file",
      manifestFile,
      "--execution-wave",
      String(executionWave),
    ]);
  }

  renderWorkerComplete(workflow: CeremonyWorkflow, workerResult: unknown): string {
    if (!isTerminalWorkerResult(workerResult)) {
      return "";
    }
    const workerFile = this.writePacket(workflow, "worker", workerResult);
    return this.runner(this.opts, [
      "ceremony",
      "worker-complete",
      "--workflow",
      workflow,
      "--worker-file",
      workerFile,
    ]);
  }

  renderCloseout(workflow: CeremonyWorkflow, completionFile: string): string {
    return this.runner(this.opts, [
      "ceremony",
      "closeout",
      "--workflow",
      workflow,
      "--completion-file",
      completionFile,
    ]);
  }

  private writePacket(
    workflow: CeremonyWorkflow,
    kind: "manifest" | "worker",
    data: unknown
  ): string {
    return writeCompletionFile(
      "aether-ceremony",
      `${workflow}-${kind}.json`,
      data
    );
  }
}

type CeremonyAdapterFactory = (opts: GoBridgeOptions) => CeremonyAdapter;

let _createCeremonyAdapterRef: CeremonyAdapterFactory = (opts) =>
  new GoCeremonyAdapter(opts);

export function createCeremonyAdapter(opts: GoBridgeOptions): CeremonyAdapter {
  return _createCeremonyAdapterRef(opts);
}

export function __setCreateCeremonyAdapter(
  factory: CeremonyAdapterFactory
): void {
  _createCeremonyAdapterRef = factory;
}

export function __restoreCreateCeremonyAdapter(): void {
  _createCeremonyAdapterRef = (opts) => new GoCeremonyAdapter(opts);
}
