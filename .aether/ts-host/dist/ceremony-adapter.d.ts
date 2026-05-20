/**
 * Runtime ceremony adapter for the TypeScript host.
 *
 * The TS host may decide when ceremony surfaces should be shown, but the Go
 * runtime owns the visual rendering contract. This adapter writes temporary
 * JSON packets and calls `aether ceremony ...` with visual output enabled.
 */
import type { GoBridgeOptions } from "./go-bridge.js";
export type CeremonyWorkflow = "plan" | "build" | "continue" | "colonize" | "swarm" | "seal";
export interface CeremonyAdapter {
    renderSpawnPlan(workflow: CeremonyWorkflow, manifestEnvelope: unknown): string;
    renderWaveStart(workflow: CeremonyWorkflow, manifestEnvelope: unknown, executionWave: number): string;
    renderWorkerComplete(workflow: CeremonyWorkflow, workerResult: unknown): string;
    renderCloseout(workflow: CeremonyWorkflow, completionFile: string): string;
}
export type CeremonyCommandRunner = (opts: GoBridgeOptions, args: string[]) => string;
export declare class GoCeremonyAdapter implements CeremonyAdapter {
    private readonly opts;
    private readonly runner;
    constructor(opts: GoBridgeOptions, runner?: CeremonyCommandRunner);
    renderSpawnPlan(workflow: CeremonyWorkflow, manifestEnvelope: unknown): string;
    renderWaveStart(workflow: CeremonyWorkflow, manifestEnvelope: unknown, executionWave: number): string;
    renderWorkerComplete(workflow: CeremonyWorkflow, workerResult: unknown): string;
    renderCloseout(workflow: CeremonyWorkflow, completionFile: string): string;
    private writePacket;
}
type CeremonyAdapterFactory = (opts: GoBridgeOptions) => CeremonyAdapter;
export declare function createCeremonyAdapter(opts: GoBridgeOptions): CeremonyAdapter;
export declare function __setCreateCeremonyAdapter(factory: CeremonyAdapterFactory): void;
export declare function __restoreCreateCeremonyAdapter(): void;
/**
 * Render a visible DRY RUN badge to stderr.
 * Per D-06, the badge is shown before and after ceremony output
 * so the user knows it is a preview and no workers will be dispatched.
 */
export declare function renderDryRunBadge(): void;
export {};
