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
import { runLifecycle, type LifecycleOptions } from "./lifecycle.js";
import { createNarrator } from "./narrator.js";
import { startEventBridge } from "./event-bridge.js";

function parseArgs(argv: string[]): {
  command: string;
  cwd: string;
  simulate: boolean;
  noDashboard: boolean;
  skipMiddenCheck: boolean;
  positional: string[];
} {
  const args = argv.slice(2); // skip node and script path
  let command = "";
  let cwd = process.cwd();
  let simulate = false;
  let noDashboard = false;
  let skipMiddenCheck = false;
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
    } else if (!command) {
      command = arg;
    } else {
      positional.push(arg);
    }
  }

  return { command, cwd, simulate, noDashboard, skipMiddenCheck, positional };
}

function printUsage(): void {
  process.stderr.write(
    "Usage: host <command> [options]\n\n" +
      "Commands:\n" +
      "  plan          Call aether plan --plan-only\n" +
      "  build <N>     Call aether build N --plan-only\n" +
      "  continue      Call aether continue --plan-only\n" +
      "  lifecycle [N] Full plan->build->continue sequence (default phase: 1)\n\n" +
      "Options:\n" +
      "  --cwd <path>        Working directory\n" +
      "  --simulate          Run in simulation mode (no real worker spawning)\n" +
      "  --no-dashboard      Disable live dashboard, use plain text output\n" +
      "  --skip-midden-check Skip pre-build midden threshold check\n"
  );
}

async function main(): Promise<void> {
  const { command, cwd, simulate, noDashboard, skipMiddenCheck, positional } = parseArgs(process.argv);

  if (!command) {
    printUsage();
    process.exit(1);
  }

  const goBinaryPath = discoverGoBinary();
  const bridge: GoBridgeOptions = { goBinaryPath, cwd };

  switch (command) {
    case "plan": {
      const result = callGoJSON<PlanCompletion>(bridge, [
        "plan",
        "--plan-only",
        "--depth",
        "fast",
      ]);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "build": {
      const phase = positional[0];
      if (!phase) {
        process.stderr.write("Error: build requires a phase number\n");
        process.exit(1);
      }
      const result = callGoJSON<BuildManifest>(bridge, [
        "build",
        phase,
        "--plan-only",
      ]);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "continue": {
      const result = callGoJSON<ContinueCompletion>(bridge, [
        "continue",
        "--plan-only",
      ]);
      process.stdout.write(JSON.stringify(result, null, 2) + "\n");
      break;
    }

    case "lifecycle": {
      const phaseArg = positional[0];
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

    default:
      process.stderr.write(`Unknown command: ${command}\n`);
      printUsage();
      process.exit(1);
  }
}

main().catch((err: unknown) => {
  const message = err instanceof Error ? err.message : String(err);
  process.stderr.write(`Fatal: ${message}\n`);
  process.exit(1);
});
