/**
 * TypeScript orchestration host entry point.
 *
 * Invoked as: node .aether/ts-host/dist/host.js <command> [options]
 *
 * Commands:
 *   plan       -- Call `aether plan --plan-only` and print JSON manifest
 *   build <N>  -- Call `aether build N --plan-only` and print JSON manifest
 *   continue   -- Call `aether continue --plan-only` and print JSON manifest
 *   lifecycle  -- Full plan -> build -> continue sequence (not yet implemented)
 *
 * Options:
 *   --cwd <path>  Working directory (default: process.cwd())
 */

import type { BuildManifest, ContinueCompletion, PlanCompletion } from "./types.js";
import { callGoJSON, discoverGoBinary } from "./go-bridge.js";
import type { GoBridgeOptions } from "./go-bridge.js";

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
export function parseArgs(argv: string[]): {
  command: string;
  cwd: string;
  simulate: boolean;
  noDashboard: boolean;
  skipMiddenCheck: boolean;
  depth: string | undefined;
  planningDepth: string | undefined;
  verificationDepth: string | undefined;
  light: boolean;
  heavy: boolean;
  workerTimeout: string | undefined;
  help: boolean;
  positional: string[];
} {
  const args = argv.slice(2); // skip node and script path
  let command = "";
  let cwd = process.cwd();
  let simulate = false;
  let noDashboard = false;
  let skipMiddenCheck = false;
  let depth: string | undefined = undefined;
  let planningDepth: string | undefined = undefined;
  let verificationDepth: string | undefined = undefined;
  let light = false;
  let heavy = false;
  let workerTimeout: string | undefined = undefined;
  let help = false;
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i]!;
    if (arg === "--cwd" && i + 1 < args.length) {
      cwd = args[++i]!;
    } else if (arg === "--simulate") {
      simulate = true;
    } else if (arg === "--no-dashboard") {
      noDashboard = true;
    } else if (arg === "--skip-midden-check") {
      skipMiddenCheck = true;
    } else if (arg === "--depth" && i + 1 < args.length) {
      depth = args[++i]!;
    } else if (arg === "--planning-depth" && i + 1 < args.length) {
      planningDepth = args[++i]!;
    } else if (arg === "--verification-depth" && i + 1 < args.length) {
      verificationDepth = args[++i]!;
    } else if (arg === "--light") {
      light = true;
    } else if (arg === "--heavy") {
      heavy = true;
    } else if (arg === "--worker-timeout" && i + 1 < args.length) {
      workerTimeout = args[++i]!;
    } else if (arg === "--help" || arg === "-h") {
      help = true;
    } else if (!command) {
      command = arg;
    } else {
      positional.push(arg);
    }
  }

  return { command, cwd, simulate, noDashboard, skipMiddenCheck, depth, planningDepth, verificationDepth, light, heavy, workerTimeout, help, positional };
}

function printUsage(): void {
  process.stderr.write(
    "Usage: host <command> [options]\n\n" +
      "Commands:\n" +
      "  plan          Call aether plan --plan-only\n" +
      "  build <N>     Call aether build N --plan-only\n" +
      "  continue      Call aether continue --plan-only\n" +
      "  oracle [topic] Run Oracle RALF lifecycle loop (iterate -> dispatch -> finalize)\n" +
      "  lifecycle [N] [topic] Full plan->build->continue sequence (default phase: 1)\n" +
      "                      Optional Oracle topic runs Oracle research before plan.\n" +
      "  watch         Show colony status, optionally with live dashboard\n" +
      "  swarm [target] Show swarm plan for a target problem\n\n" +
      "Options:\n" +
      "  --cwd <path>           Working directory\n" +
      "  --simulate             Run in simulation mode (no real worker spawning)\n" +
      "  --no-dashboard         Disable live dashboard, use plain text output\n" +
      "  --skip-midden-check    Skip pre-build midden threshold check\n" +
      "  --depth <level>        fast | balanced | deep | exhaustive\n" +
      "  --planning-depth <lvl> light | standard | deep\n" +
      "  --verification-depth <lvl> light | standard | heavy\n" +
      "  --light                Force light review\n" +
      "  --heavy                Force heavy review\n" +
      "  --worker-timeout <dur> Override per-worker timeout (e.g. 5m, 15m)\n"
  );
}

async function main(): Promise<void> {
  const { command, cwd, simulate, noDashboard, skipMiddenCheck, depth, planningDepth, verificationDepth, light, heavy, workerTimeout, help, positional } = parseArgs(process.argv);

  if (help || !command) {
    printUsage();
    process.exit(help ? 0 : 1);
  }

  const goBinaryPath = discoverGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd };

  switch (command) {
    case "plan": {
      const args = ["plan", "--plan-only"];
      if (depth) args.push("--depth", depth);
      if (planningDepth) args.push("--planning-depth", planningDepth);
      if (verificationDepth) args.push("--verification-depth", verificationDepth);
      if (simulate) args.push("--synthetic");
      if (workerTimeout) args.push("--worker-timeout", workerTimeout);
      const result = _callGoJSONRef<PlanCompletion>(bridge, args);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "build": {
      const phase = positional[0];
      if (!phase) {
        process.stderr.write("Error: build requires a phase number\n");
        process.exit(1);
      }
      const args = ["build", phase, "--plan-only"];
      if (simulate) args.push("--synthetic");
      if (light) args.push("--light");
      if (workerTimeout) args.push("--worker-timeout", workerTimeout);
      const result = _callGoJSONRef<BuildManifest>(bridge, args);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "continue": {
      const args = ["continue", "--plan-only"];
      if (verificationDepth) args.push("--verification-depth", verificationDepth);
      if (light) args.push("--light");
      if (heavy) args.push("--heavy");
      if (simulate) args.push("--synthetic");
      if (workerTimeout) args.push("--worker-timeout", workerTimeout);
      const result = _callGoJSONRef<ContinueCompletion>(bridge, args);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "oracle": {
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

      const bridge = await startEventBridge({
        goBinaryPath,
        cwd,
        onEvent: (evt) => {
          narrator.onEvent(evt);
        },
      });

      const result = await runLifecycle(lifecycleOpts);

      await bridge.stop();
      narrator.stop();

      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "watch": {
      const watchOpts: WatchDisplayOptions = {
        goBinaryPath,
        cwd,
        dashboard: !noDashboard,
      };

      const result = await runWatchDisplay(watchOpts);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "swarm": {
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

    default:
      process.stderr.write(`Unknown command: ${command}\n`);
      printUsage();
      process.exit(1);
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
