/**
 * Project area activity visualization.
 *
 * Groups workers by directory prefix of files_created and files_modified,
 * then renders a chamber map with progress bars and worker counts.
 */
import type { WorkerState } from "./worker-widget.js";
export interface ChamberActivity {
    directory: string;
    progress: number;
    workerCount: number;
}
export interface ChamberMap {
    activities: ChamberActivity[];
    update(workers: WorkerState[]): void;
}
/**
 * Create a new empty chamber map.
 */
export declare function createChamberMap(): ChamberMap;
/**
 * Render a ChamberMap instance as a formatted string section.
 */
export declare function renderChamberMap(map: ChamberMap): string;
/**
 * Render chamber activity data directly (used by the dashboard renderer
 * to avoid constructing a ChamberMap just for rendering).
 */
export declare function renderChamberMapData(activities: ChamberActivity[]): string;
/**
 * Extract the directory prefix from a file path.
 * Returns "." if there is no directory separator.
 */
export declare function extractDirectoryPrefix(path: string): string;
