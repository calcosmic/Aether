/**
 * Dashboard controller, event handler, and worker model.
 *
 * Maintains an in-memory model of active workers keyed by spawn_id,
 * reacts to ceremony events, and drives the renderer to produce
 * atomic dashboard frames.
 */
import type { CeremonyEvent } from "./types.js";
export interface DashboardOptions {
    cwd: string;
    outputMode?: string;
}
export interface Dashboard {
    onEvent(event: CeremonyEvent): void;
    start(): void;
    stop(): void;
}
/**
 * Create a new swarm dashboard.
 */
export declare function createDashboard(opts: DashboardOptions): Dashboard;
