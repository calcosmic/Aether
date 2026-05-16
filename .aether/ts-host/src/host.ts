/**
 * TypeScript orchestration host entry point.
 *
 * Invoked as: node .aether/ts-host/dist/host.js <command> [options]
 *
 * Commands:
 *   plan       -- Call `aether plan --plan-only` and print JSON manifest
 *   build <N>  -- Call `aether build N --plan-only` and print JSON manifest
 *   continue   -- Call `aether continue --plan-only` and print JSON manifest
 *   oracle     -- Run the Oracle lifecycle loop
 *   lifecycle  -- Run the plan -> build -> continue lifecycle sequence
 *   watch      -- Show colony status through the host display surface
 *   swarm      -- Show or plan swarm activity through the host display surface
 *
 * Options:
 *   --cwd <path>  Working directory (default: process.cwd())
 */

import { callGoJSON, discoverGoBinary } from "./go-bridge.js";
import type { GoBridgeOptions } from "./go-bridge.js";
import {
  buildHostGoArgs,
  getHostCommandDefinition,
  listHostCommandDefinitions,
  type ParsedHostArgs,
} from "./command-registry.js";
import { runGoJSONCommand } from "./go-command.js";

export { buildHostGoArgs } from "./command-registry.js";
export type { ParsedHostArgs } from "./command-registry.js";

// Mutable reference for test injection.
let _callGoJSONRef = callGoJSON;

/** Test-only: inject a mock callGoJSON. */
export function __setCallGoJSON(fn: typeof callGoJSON): void {
  _callGoJSONRef = fn;
}

/** Test-only: restore the real callGoJSON. */
export function __restoreCallGoJSON(): void {
  _callGoJSONRef = callGoJSON;
}
import { runLifecycle, type LifecycleOptions } from "./lifecycle.js";
import { runOracleLifecycle, type OracleLifecycleOptions } from "./oracle-lifecycle.js";
import { runWatchDisplay, type WatchDisplayOptions } from "./watch-display.js";
import { runSwarmDisplay, type SwarmDisplayOptions } from "./swarm-display.js";
import { createNarrator } from "./narrator.js";
import { startEventBridge } from "./event-bridge.js";

/** Parse command-line arguments for the TS host. */
export function parseArgs(argv: string[]): ParsedHostArgs {
  const args = argv.slice(2); // skip node and script path
  let command = "";
  let cwd = process.cwd();
  let simulate = false;
  let synthetic = false;
  let noDashboard = false;
  let skipMiddenCheck = false;
  let skipWatchers = false;
  let refresh = false;
  let force = false;
  const tasks: string[] = [];
  let depth: string | undefined = undefined;
  let planningDepth: string | undefined = undefined;
  let verificationDepth: string | undefined = undefined;
  let verificationTimeout: string | undefined = undefined;
  let light = false;
  let heavy = false;
  let workerTimeout: string | undefined = undefined;
  let circuitBreakerThreshold: string | undefined = undefined;
  let noSuggest = false;
  let verbose = false;
  const reconcileTasks: string[] = [];
  let noLearn = false;
  let help = false;
  const positional: string[] = [];
  const unknownFlags: string[] = [];

  for (let i = 0; i < args.length; i++) {
    let arg = args[i]!;
    const rawArg = arg;
    let inlineValue: string | undefined;
    if (arg.startsWith("--")) {
      const eq = arg.indexOf("=");
      if (eq > 2) {
        inlineValue = arg.slice(eq + 1);
        arg = arg.slice(0, eq);
      }
    }
    const nextValue = (): string | undefined => {
      if (inlineValue !== undefined) return inlineValue;
      if (i + 1 < args.length) return args[++i]!;
      return undefined;
    };
    const readValue = (flag: string): string | undefined => {
      const value = nextValue();
      if (value === undefined) {
        unknownFlags.push(`${flag} requires a value`);
      }
      return value;
    };
    if (arg === "--cwd") {
      cwd = readValue(arg) ?? cwd;
    } else if (arg === "--simulate") {
      simulate = true;
    } else if (arg === "--synthetic") {
      synthetic = true;
    } else if (arg === "--no-dashboard") {
      noDashboard = true;
    } else if (arg === "--skip-midden-check") {
      skipMiddenCheck = true;
    } else if (arg === "--skip-watchers") {
      skipWatchers = true;
    } else if (arg === "--refresh") {
      refresh = true;
    } else if (arg === "--force") {
      force = true;
    } else if (arg === "--task") {
      const value = readValue(arg);
      if (value !== undefined) tasks.push(value);
    } else if (arg === "--depth") {
      depth = readValue(arg);
    } else if (arg === "--planning-depth") {
      planningDepth = readValue(arg);
    } else if (arg === "--verification-depth") {
      verificationDepth = readValue(arg);
    } else if (arg === "--verification-timeout") {
      verificationTimeout = readValue(arg);
    } else if (arg === "--light") {
      light = true;
    } else if (arg === "--heavy") {
      heavy = true;
    } else if (arg === "--worker-timeout") {
      workerTimeout = readValue(arg);
    } else if (arg === "--circuit-breaker-threshold") {
      circuitBreakerThreshold = readValue(arg);
    } else if (arg === "--no-suggest") {
      noSuggest = true;
    } else if (arg === "--verbose") {
      verbose = true;
    } else if (arg === "--reconcile-task") {
      const value = readValue(arg);
      if (value !== undefined) reconcileTasks.push(value);
    } else if (arg === "--no-learn") {
      noLearn = true;
    } else if (arg === "--help" || arg === "-h") {
      help = true;
    } else if (arg.startsWith("-")) {
      unknownFlags.push(rawArg);
    } else if (!command) {
      command = rawArg;
    } else {
      positional.push(rawArg);
    }
  }

  return {
    command,
    cwd,
    simulate,
    synthetic,
    noDashboard,
    skipMiddenCheck,
    skipWatchers,
    refresh,
    force,
    tasks,
    depth,
    planningDepth,
    verificationDepth,
    verificationTimeout,
    light,
    heavy,
    workerTimeout,
    circuitBreakerThreshold,
    noSuggest,
    verbose,
    reconcileTasks,
    noLearn,
    help,
    positional,
    unknownFlags,
  };
}

function printUsage(): void {
  const commandLines = listHostCommandDefinitions()
    .map((definition) => `  ${definition.usage.padEnd(14)} ${definition.description}`)
    .join("\n");
  process.stderr.write(
    "Usage: host <command> [options]\n\n" +
      "Commands:\n" +
      commandLines + "\n\n" +
      "Options:\n" +
      "  --cwd <path>           Working directory\n" +
      "  --simulate             Run in simulation mode (no real worker spawning)\n" +
      "  --synthetic            Forward Go synthetic mode for plan/build/continue\n" +
      "  --no-dashboard         Disable live dashboard, use plain text output\n" +
      "  --skip-midden-check    Skip pre-build midden threshold check\n" +
      "  --skip-watchers        Skip continue watcher workers when Go allows it\n" +
      "  --refresh              Refresh an existing plan\n" +
      "  --force                Forward Go force aliases for plan/build\n" +
      "  --task <id>            Limit build dispatch to a task id (repeatable)\n" +
      "  --depth <level>        fast | balanced | deep | exhaustive\n" +
      "  --planning-depth <lvl> light | standard | deep\n" +
      "  --verification-depth <lvl> light | standard | heavy\n" +
      "  --verification-timeout <dur> Override continue verification timeout\n" +
      "  --light                Force light review\n" +
      "  --heavy                Force heavy review\n" +
      "  --worker-timeout <dur> Override per-worker timeout (e.g. 5m, 15m)\n" +
      "  --circuit-breaker-threshold <n> Forward build circuit-breaker threshold\n" +
      "  --no-suggest           Skip build suggestion analysis\n" +
      "  --verbose              Forward verbose build output mode\n" +
      "  --reconcile-task <id>  Mark continue task reconciliation (repeatable)\n" +
      "  --no-learn             Disable continue learning capture when supported\n"
  );
}

async function main(): Promise<void> {
  const parsed = parseArgs(process.argv);
  const { command, cwd, simulate, noDashboard, skipMiddenCheck, help, positional } = parsed;

  if (help || !command) {
    printUsage();
    process.exit(help ? 0 : 1);
  }

  const goBinaryPath = discoverGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd };
  const definition = getHostCommandDefinition(command);

  if (!definition) {
    process.stderr.write(`Unknown command: ${command}\n`);
    printUsage();
    process.exit(1);
  }

  if (parsed.unknownFlags.length > 0) {
    process.stderr.write(`Error: Unsupported host flag(s): ${parsed.unknownFlags.join(", ")}\n`);
    process.exit(1);
  }

  switch (definition.runner) {
    case "go-json": {
      let args: string[];
      try {
        args = buildHostGoArgs(parsed)!;
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        process.stderr.write(`Error: ${message}\n`);
        process.exit(1);
      }
      const result = runGoJSONCommand(bridge, args, _callGoJSONRef);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "oracle-lifecycle": {
      const topic = positional[0] || "auto";
      const oracleOpts: OracleLifecycleOptions = {
        goBinaryPath,
        cwd,
        topic,
        simulateWorkers: simulate,
        dashboard: !noDashboard,
      };

      const result = await runOracleLifecycle(oracleOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "lifecycle": {
      const phaseArg = positional[0];
      const oracleTopicArg = positional[1];
      if (phaseArg && isNaN(parseInt(phaseArg, 10))) {
        process.stderr.write("Error: lifecycle phase must be a number\n");
        process.exit(1);
      }
      if (simulate) {
        process.stderr.write("Running in simulation mode\n");
      }
      const lifecycleOpts: LifecycleOptions = {
        goBinaryPath,
        cwd,
        simulateWorkers: simulate,
        dashboard: !noDashboard,
        skipMiddenCheck,
      };
      if (phaseArg) {
        lifecycleOpts.phase = parseInt(phaseArg, 10);
      }
      if (oracleTopicArg) {
        lifecycleOpts.runOracle = true;
        lifecycleOpts.oracleTopic = oracleTopicArg;
      }

      const narrator = createNarrator({
        cwd,
        outputMode: process.env["AETHER_OUTPUT_MODE"],
        suppressOutput: !noDashboard && process.stdout.isTTY,
      });

      const eventBridge = await startEventBridge({
        goBinaryPath,
        cwd,
        onEvent: (evt) => {
          narrator.onEvent(evt);
        },
      });

      const result = await runLifecycle(lifecycleOpts);

      await eventBridge.stop();
      narrator.stop();

      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "watch-display": {
      const watchOpts: WatchDisplayOptions = {
        goBinaryPath,
        cwd,
        dashboard: !noDashboard,
      };

      const result = await runWatchDisplay(watchOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "swarm-display": {
      const target = positional[0] || "";
      const swarmOpts: SwarmDisplayOptions = {
        goBinaryPath,
        cwd,
        target,
        dashboard: !noDashboard,
        planOnly: true,
      };

      const result = await runSwarmDisplay(swarmOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }
  }
}

import { fileURLToPath } from "node:url";
import { resolve } from "node:path";

const isMainModule = fileURLToPath(import.meta.url) === resolve(process.argv[1]!);
if (isMainModule) {
  main().catch((err: unknown) => {
    const message = err instanceof Error ? err.message : String(err);
    process.stderr.write(`Fatal: ${message}\n`);
    process.exit(1);
  });
}
