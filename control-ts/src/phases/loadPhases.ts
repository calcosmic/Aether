import { readdirSync, readFileSync } from "fs";
import { resolve } from "path";
import { parse } from "yaml";
import { PhaseSchema, type Phase } from "../schemas/phase.schema.js";
import { projectRoot } from "../utils/projectRoot.js";

const PHASES_DIR = resolve(projectRoot, "..", "colony", "phases");

/**
 * Load and validate all phase definitions from colony/phases/*.yaml.
 * @returns Sorted array of validated Phase objects.
 * @throws Error if directory is missing, unreadable, or any file fails validation.
 */
export function loadPhases(): Phase[] {
  let files: string[];
  try {
    files = readdirSync(PHASES_DIR).filter((f) => f.endsWith(".yaml"));
  } catch (err) {
    throw new Error(
      `Failed to read phases directory: ${PHASES_DIR} — ${(err as Error).message}`
    );
  }

  if (files.length === 0) {
    throw new Error(`No .yaml files found in phases directory: ${PHASES_DIR}`);
  }

  const phases: Phase[] = [];
  for (const file of files) {
    const filePath = resolve(PHASES_DIR, file);
    let raw: string;
    try {
      raw = readFileSync(filePath, "utf8");
    } catch (err) {
      throw new Error(
        `Failed to read phase file: ${filePath} — ${(err as Error).message}`
      );
    }

    let parsed: unknown;
    try {
      parsed = parse(raw);
    } catch (err) {
      throw new Error(
        `Failed to parse YAML in ${filePath} — ${(err as Error).message}`
      );
    }

    try {
      const phase = PhaseSchema.parse(parsed);
      phases.push(phase);
    } catch (err) {
      const zodError = err as { message: string; issues?: unknown[] };
      const details = zodError.issues
        ? JSON.stringify(zodError.issues, null, 2)
        : zodError.message;
      throw new Error(`Validation failed for ${filePath}:\n${details}`);
    }
  }

  phases.sort((a, b) => a.id.localeCompare(b.id));
  return phases;
}

/**
 * Find a single phase by its id.
 * @param id Phase identifier.
 * @returns The matching Phase, or null if not found.
 */
export function loadPhaseById(id: string): Phase | null {
  const phases = loadPhases();
  return phases.find((p) => p.id === id) ?? null;
}
