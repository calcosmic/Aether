/**
 * Frame assembly using log-update and boxen.
 *
 * Assembles a full dashboard frame from worker state and chamber activity,
 * then writes it atomically via log-update to prevent torn frames.
 */

import logUpdate from "log-update";
import boxen from "boxen";
import chalk from "chalk";
import type { WorkerState } from "./worker-widget.js";
import { renderWorkerWidget, formatDuration } from "./worker-widget.js";
import type { ChamberActivity } from "./chamber-map.js";
import { renderChamberMapData } from "./chamber-map.js";
import type { CeremonyConfig } from "../caste-config.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface DashboardFrameData {
  wave: number;
  totalWaves: number;
  activeWorkers: WorkerState[];
  completedWorkers: WorkerState[];
  failedWorkers: WorkerState[];
  elapsedSeconds: number;
  chamberActivity: ChamberActivity[];
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Render a full dashboard frame and atomically replace the previous one.
 */
export function renderDashboardFrame(
  data: DashboardFrameData,
  config: CeremonyConfig
): void {
  const frame = buildFrame(data, config);
  logUpdate(frame);
}

/**
 * Clear the dashboard from the terminal.
 */
export function clearDashboard(): void {
  logUpdate.clear();
}

/**
 * Render the header line.
 */
export function renderHeader(data: DashboardFrameData): string {
  const elapsed = formatDuration(data.elapsedSeconds * 1000);
  return `Swarm Dashboard — Wave ${data.wave} of ${data.totalWaves} — ${elapsed}`;
}

/**
 * Render the footer line.
 */
export function renderFooter(data: DashboardFrameData): string {
  return `Active: ${data.activeWorkers.length} | Completed: ${data.completedWorkers.length} | Failed: ${data.failedWorkers.length}`;
}

// ---------------------------------------------------------------------------
// Frame builder
// ---------------------------------------------------------------------------

function buildFrame(data: DashboardFrameData, config: CeremonyConfig): string {
  const lines: string[] = [];

  lines.push(renderHeader(data));
  lines.push("");

  // Active workers
  if (data.activeWorkers.length > 0) {
    lines.push(chalk.bold(`Active Workers (${data.activeWorkers.length})`));
    for (const worker of data.activeWorkers) {
      lines.push("  " + renderWorkerWidget({ state: worker }, config));
    }
  } else {
    lines.push(chalk.bold("Active Workers"));
    lines.push("  Waiting for workers...");
  }
  lines.push("");

  // Completed workers
  if (data.completedWorkers.length > 0) {
    lines.push(chalk.bold(`Completed (${data.completedWorkers.length})`));
    for (const worker of data.completedWorkers) {
      lines.push("  " + renderWorkerWidget({ state: worker }, config));
    }
    lines.push("");
  }

  // Failed workers
  if (data.failedWorkers.length > 0) {
    lines.push(chalk.bold(`Failed (${data.failedWorkers.length})`));
    for (const worker of data.failedWorkers) {
      lines.push("  " + renderWorkerWidget({ state: worker }, config));
    }
    lines.push("");
  }

  // Chamber activity
  lines.push(renderChamberMapData(data.chamberActivity));
  lines.push("");

  lines.push(renderFooter(data));

  const content = lines.join("\n");

  return boxen(content, {
    padding: { top: 0, bottom: 0, left: 1, right: 1 },
    margin: 0,
    borderStyle: "single",
    borderColor: "cyan",
    title: "Aether Swarm",
    titleAlignment: "center",
  });
}
