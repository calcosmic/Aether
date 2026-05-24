import { executePlan } from "./orchestrator/executePlan.js";
import { parseEventLine } from "./schemas/event.schema.js";
import { readColonyState, updateColonyState } from "./memory/store.js";
import { projectRoot } from "./utils/projectRoot.js";
import { resolve } from "path";
import { readFileSync } from "fs";

const DEFAULT_TASK = "create a small test file and verify it";

function parseTaskArg(): string {
  const idx = process.argv.indexOf("--task");
  if (idx !== -1 && idx + 1 < process.argv.length) {
    return process.argv[idx + 1];
  }
  return DEFAULT_TASK;
}

export async function main(): Promise<void> {
  const task = parseTaskArg();

  // Ensure a clean state for the demo
  updateColonyState(() => ({
    version: "3.0",
    goal: task,
    scope: "project",
    colony_mode: "colony",
    state: "INIT",
    current_phase: 0,
    session_id: "",
    initialized_at: new Date().toISOString(),
    plan: {
      generated_at: new Date().toISOString(),
      confidence: 0,
      phases: [],
    },
  }));

  const result = await executePlan(["init", "plan", "build"], {
    inputs: {
      init: { goal: task },
    },
  });

  const eventsFile = resolve(projectRoot, "..", ".aether", "events", "current.ndjson");
  let eventTypes: string[] = [];
  let eventCount = 0;

  try {
    const raw = readFileSync(eventsFile, "utf8");
    const lines = raw.trim().split("\n").filter(Boolean);
    eventCount = lines.length;
    eventTypes = lines.map((line) => {
      const ev = parseEventLine(line);
      return ev.type;
    });
  } catch {
    // If file doesn't exist or is empty, counts stay at 0
  }

  console.log(`Plan status: ${result.status}`);
  console.log(`Phases executed: ${result.results.length}`);
  console.log(`Events emitted: ${eventCount}`);
  console.log(`Event types: ${eventTypes.join(", ") || "none"}`);

  if (result.status === "completed") {
    process.exit(0);
  } else {
    process.exit(1);
  }
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
