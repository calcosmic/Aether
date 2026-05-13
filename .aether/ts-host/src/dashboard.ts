/**
 * Dashboard controller, event handler, and worker model.
 *
 * Maintains an in-memory model of active workers keyed by spawn_id,
 * reacts to ceremony events, and drives the renderer to produce
 * atomic dashboard frames.
 */

import { loadCeremonyConfig, type CeremonyConfig } from "./caste-config.js";
import type { CeremonyEvent, CeremonyPayload } from "./types.js";
import {
  renderDashboardFrame,
  clearDashboard,
  type DashboardFrameData,
} from "./dashboard/dashboard-renderer.js";
import { createChamberMap, type ChamberMap } from "./dashboard/chamber-map.js";
import type { WorkerState } from "./dashboard/worker-widget.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DashboardOptions {
  cwd: string;
  outputMode?: string;
}

export interface Dashboard {
  onEvent(event: CeremonyEvent): void;
  start(): void;
  stop(): void;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Create a new swarm dashboard.
 */
export function createDashboard(opts: DashboardOptions): Dashboard {
  const config = loadCeremonyConfig(opts.cwd);
  const workers = new Map<string, WorkerState>();
  const chamberMap: ChamberMap = createChamberMap();
  let startTime = 0;
  let currentWave = 0;
  let totalWaves = 0;

  return {
    onEvent(event: CeremonyEvent): void {
      const payload = event.payload;

      switch (event.topic) {
        case "ceremony.build.spawn":
        case "ceremony.plan.spawn":
        case "ceremony.colonize.spawn":
        case "ceremony.continue.spawn": {
          const spawnId = payload.spawn_id ?? payload.name ?? event.id;
          const worker: WorkerState = {
            spawn_id: spawnId,
            caste: payload.caste ?? "unknown",
            name: payload.name ?? "Unknown",
            task: payload.task ?? "",
            status: "active",
            tool_count: payload.tool_count ?? 0,
            token_count: payload.token_count ?? 0,
            files_created: payload.files_created ?? [],
            files_modified: payload.files_modified ?? [],
            startTime: Date.now(),
            lastUpdate: Date.now(),
          };
          workers.set(spawnId, worker);
          break;
        }

        case "ceremony.build.tool_use":
        case "ceremony.plan.tool_use":
        case "ceremony.colonize.tool_use":
        case "ceremony.continue.tool_use": {
          const spawnId = payload.spawn_id ?? payload.name;
          if (spawnId && workers.has(spawnId)) {
            const existing = workers.get(spawnId)!;
            existing.tool_count = payload.tool_count ?? existing.tool_count;
            existing.token_count = payload.token_count ?? existing.token_count;
            existing.lastUpdate = Date.now();
            workers.set(spawnId, existing);
          }
          break;
        }

        case "ceremony.build.wave.end":
        case "ceremony.plan.wave.end":
        case "ceremony.colonize.wave.end":
        case "ceremony.continue.wave.end": {
          const wave = payload.wave ?? currentWave;
          for (const worker of workers.values()) {
            if (worker.status === "active") {
              worker.status = "completed";
              worker.lastUpdate = Date.now();
            }
          }
          break;
        }

        case "ceremony.build.circuit_break":
        case "ceremony.plan.circuit_break":
        case "ceremony.colonize.circuit_break":
        case "ceremony.continue.circuit_break": {
          const spawnId = payload.spawn_id ?? payload.name;
          if (spawnId && workers.has(spawnId)) {
            const existing = workers.get(spawnId)!;
            existing.status = "blocked";
            existing.lastUpdate = Date.now();
            workers.set(spawnId, existing);
          }
          break;
        }

        default: {
          // Handle status-failed on any topic
          if (payload.status === "failed") {
            const spawnId = payload.spawn_id ?? payload.name;
            if (spawnId && workers.has(spawnId)) {
              const existing = workers.get(spawnId)!;
              existing.status = "failed";
              existing.lastUpdate = Date.now();
              workers.set(spawnId, existing);
            }
          }
          break;
        }
      }

      // Update wave tracking from any event that carries wave info
      if (payload.wave !== undefined) {
        currentWave = payload.wave;
      }
      if (payload.total !== undefined && payload.total > totalWaves) {
        totalWaves = payload.total;
      }

      render();
    },

    start(): void {
      startTime = Date.now();
      render();
    },

    stop(): void {
      clearDashboard();
      workers.clear();
    },
  };

  function render(): void {
    if (startTime === 0) return; // not started yet

    chamberMap.update(Array.from(workers.values()));

    const activeWorkers: WorkerState[] = [];
    const completedWorkers: WorkerState[] = [];
    const failedWorkers: WorkerState[] = [];

    for (const worker of workers.values()) {
      switch (worker.status) {
        case "active":
          activeWorkers.push(worker);
          break;
        case "completed":
          completedWorkers.push(worker);
          break;
        case "failed":
          failedWorkers.push(worker);
          break;
        case "blocked":
          failedWorkers.push(worker);
          break;
      }
    }

    const data: DashboardFrameData = {
      wave: currentWave,
      totalWaves: totalWaves || 1,
      activeWorkers,
      completedWorkers,
      failedWorkers,
      elapsedSeconds: Math.floor((Date.now() - startTime) / 1000),
      chamberActivity: chamberMap.activities,
    };

    renderDashboardFrame(data, config);
  }
}
