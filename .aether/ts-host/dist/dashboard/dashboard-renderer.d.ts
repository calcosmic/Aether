/**
 * Frame assembly using log-update and boxen.
 *
 * Assembles a full dashboard frame from worker state and chamber activity,
 * then writes it atomically via log-update to prevent torn frames.
 */
import type { WorkerState } from "./worker-widget.js";
import type { ChamberActivity } from "./chamber-map.js";
import type { CeremonyConfig } from "../caste-config.js";
export interface OracleDisplayState {
    phase: string;
    iteration: number;
    active: boolean;
}
export interface DashboardFrameData {
    wave: number;
    totalWaves: number;
    activeWorkers: WorkerState[];
    completedWorkers: WorkerState[];
    failedWorkers: WorkerState[];
    elapsedSeconds: number;
    chamberActivity: ChamberActivity[];
    oracle?: OracleDisplayState | undefined;
}
/**
 * Render a full dashboard frame and atomically replace the previous one.
 */
export declare function renderDashboardFrame(data: DashboardFrameData, config: CeremonyConfig): void;
/**
 * Clear the dashboard from the terminal.
 */
export declare function clearDashboard(): void;
/**
 * Render the header line.
 */
export declare function renderHeader(data: DashboardFrameData): string;
/**
 * Render the footer line.
 */
export declare function renderFooter(data: DashboardFrameData): string;
