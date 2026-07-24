import { readFileSync, writeFileSync, renameSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { projectRoot } from "../utils/projectRoot.js";
import type { ColonyState } from "../types/runtime.js";

const RETIRED_STATE_PATH = resolve(projectRoot, ".retired-state", "COLONY_STATE.json");

export function colonyStatePath(): string {
  const override = process.env.AETHER_CONTROL_STATE_PATH?.trim();
  return override ? resolve(override) : RETIRED_STATE_PATH;
}

function defaultState(): ColonyState {
  const now = new Date().toISOString();
  return {
    version: "3.0",
    goal: "",
    scope: "project",
    colony_mode: "colony",
    state: "INIT",
    current_phase: 0,
    session_id: "",
    initialized_at: now,
    plan: {
      generated_at: now,
      confidence: 0,
      phases: [],
    },
  };
}

/**
 * Read and parse COLONY_STATE.json.
 * Returns a default empty state if the file does not exist.
 */
export function readColonyState(): ColonyState {
  const statePath = colonyStatePath();
  if (!existsSync(statePath)) {
    return defaultState();
  }
  const raw = readFileSync(statePath, "utf8");
  const parsed = JSON.parse(raw) as ColonyState;
  return parsed;
}

/**
 * Serialize and write ColonyState atomically to COLONY_STATE.json.
 * Uses a temp file + rename to prevent partial writes on crash.
 * Creates parent directories if needed.
 */
export function writeColonyState(state: ColonyState): void {
  const statePath = colonyStatePath();
  const dir = dirname(statePath);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  const tmpPath = `${statePath}.tmp`;
  writeFileSync(tmpPath, JSON.stringify(state, null, 2), "utf8");
  renameSync(tmpPath, statePath);
}

/**
 * Read current state, apply updater function, and write back atomically.
 * Throws "Memory update failed: {message}" on error.
 */
export function updateColonyState(updater: (state: ColonyState) => ColonyState): void {
  try {
    const current = readColonyState();
    const next = updater(current);
    writeColonyState(next);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`Memory update failed: ${message}`);
  }
}
