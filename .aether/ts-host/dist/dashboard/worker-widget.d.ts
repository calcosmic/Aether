/**
 * Per-worker display state and rendering.
 *
 * Each worker gets a widget with an ora spinner, progress bar,
 * tool/token counts, and elapsed duration. The widget is updated
 * in-place as events arrive.
 */
import type { Ora } from "ora";
import { type CeremonyConfig } from "../caste-config.js";
export interface WorkerState {
    spawn_id: string;
    caste: string;
    name: string;
    task: string;
    status: "active" | "completed" | "failed" | "blocked";
    tool_count: number;
    token_count: number;
    files_created: string[];
    files_modified: string[];
    startTime: number;
    lastUpdate: number;
}
export interface WorkerWidget {
    state: WorkerState;
    spinner?: Ora;
}
/**
 * Create a new worker widget with an optional ora spinner.
 */
export declare function createWorkerWidget(state: WorkerState, config: CeremonyConfig): WorkerWidget;
/**
 * Render a worker widget as a formatted string.
 */
export declare function renderWorkerWidget(widget: WorkerWidget, config: CeremonyConfig): string;
/**
 * Update a worker widget's state and sync the spinner.
 */
export declare function updateWorkerWidget(widget: WorkerWidget, state: WorkerState): void;
/**
 * Format a duration in milliseconds as MM:SS (or HH:MM:SS if >= 1 hour).
 */
export declare function formatDuration(ms: number): string;
