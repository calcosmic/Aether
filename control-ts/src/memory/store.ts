import { readFileSync, writeFileSync, renameSync, mkdirSync, existsSync } from "fs";
import { resolve, dirname } from "path";
import { projectRoot } from "../utils/projectRoot.js";
import type { ColonyState } from "../types/runtime.js";

const STATE_PATH = resolve(projectRoot, "..", ".aether", "data", "COLONY_STATE.json");

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
  if (!existsSync(STATE_PATH)) {
    return defaultState();
  }
  const raw = readFileSync(STATE_PATH, "utf8");
  const parsed = JSON.parse(raw) as ColonyState;
  return parsed;
}

/**
 * Serialize and write ColonyState atomically to COLONY_STATE.json.
 * Uses a temp file + rename to prevent partial writes on crash.
 * Creates parent directories if needed.
 */
export function writeColonyState(state: ColonyState): void {
  const dir = dirname(STATE_PATH);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }
  const tmpPath = `${STATE_PATH}.tmp`;
  writeFileSync(tmpPath, JSON.stringify(state, null, 2), "utf8");
  renameSync(tmpPath, STATE_PATH);
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
