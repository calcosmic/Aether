/**
 * Project area activity visualization.
 *
 * Groups workers by directory prefix of files_created and files_modified,
 * then renders a chamber map with progress bars and worker counts.
 */

import type { WorkerState } from "./worker-widget.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ChamberActivity {
  directory: string;
  progress: number;
  workerCount: number;
}

export interface ChamberMap {
  activities: ChamberActivity[];
  update(workers: WorkerState[]): void;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Create a new empty chamber map.
 */
export function createChamberMap(): ChamberMap {
  const activities: ChamberActivity[] = [];

  return {
    activities,
    update(workers: WorkerState[]) {
      const dirMap = new Map<string, { totalProgress: number; workerCount: number; workerIds: Set<string> }>();

      for (const worker of workers) {
        const dirs = new Set<string>();
        for (const f of worker.files_created ?? []) {
          dirs.add(extractDirectoryPrefix(f));
        }
        for (const f of worker.files_modified ?? []) {
          dirs.add(extractDirectoryPrefix(f));
        }

        const progress = Math.min(100, Math.round((worker.tool_count / 20) * 100));

        for (const dir of dirs) {
          const existing = dirMap.get(dir);
          if (existing) {
            existing.totalProgress += progress;
            if (!existing.workerIds.has(worker.spawn_id)) {
              existing.workerIds.add(worker.spawn_id);
              existing.workerCount += 1;
            }
          } else {
            dirMap.set(dir, {
              totalProgress: progress,
              workerCount: 1,
              workerIds: new Set([worker.spawn_id]),
            });
          }
        }
      }

      const next: ChamberActivity[] = [];
      for (const [directory, data] of dirMap) {
        next.push({
          directory,
          progress: Math.round(data.totalProgress / data.workerCount),
          workerCount: data.workerCount,
        });
      }

      // Sort by progress descending, then by directory name
      next.sort((a, b) => {
        if (b.progress !== a.progress) return b.progress - a.progress;
        return a.directory.localeCompare(b.directory);
      });

      // Replace in place to keep the same array reference
      activities.length = 0;
      activities.push(...next);
    },
  };
}

/**
 * Render a ChamberMap instance as a formatted string section.
 */
export function renderChamberMap(map: ChamberMap): string {
  return renderChamberMapData(map.activities);
}

/**
 * Render chamber activity data directly (used by the dashboard renderer
 * to avoid constructing a ChamberMap just for rendering).
 */
export function renderChamberMapData(activities: ChamberActivity[]): string {
  if (activities.length === 0) {
    return "Chamber Activity\n  (no activity yet)";
  }

  const lines: string[] = ["Chamber Activity"];
  const top = activities.slice(0, 5);
  for (const act of top) {
    const bar = renderProgressBar(act.progress, 10);
    const dir = act.directory.padEnd(20);
    lines.push(`  ${dir} ${bar} ${String(act.progress).padStart(3)}%  (${act.workerCount} worker${act.workerCount === 1 ? "" : "s"})`);
  }
  return lines.join("\n");
}

/**
 * Extract the directory prefix from a file path.
 * Returns "." if there is no directory separator.
 */
export function extractDirectoryPrefix(path: string): string {
  const lastSlash = path.lastIndexOf("/");
  if (lastSlash === -1) return ".";
  return path.slice(0, lastSlash);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function renderProgressBar(percent: number, width: number): string {
  const filled = Math.round((percent / 100) * width);
  const empty = width - filled;
  return "█".repeat(filled) + "░".repeat(empty);
}
