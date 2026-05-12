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

function parseArgs(argv: string[]): {
  command: string;
  cwd: string;
  positional: string[];
} {
  const args = argv.slice(2); // skip node and script path
  let command = "";
  let cwd = process.cwd();
  const positional: string[] = [];

  for (let i = 0; i < args.length; i++) {
    const arg = args[i]!;
    if (arg === "--cwd" && i + 1 < args.length) {
      cwd = args[++i]!;
    } else if (!command) {
      command = arg;
    } else {
      positional.push(arg);
    }
  }

  return { command, cwd, positional };
}

function printUsage(): void {
  process.stderr.write(
    "Usage: host <command> [options]\n\n" +
      "Commands:\n" +
      "  plan          Call aether plan --plan-only\n" +
      "  build <N>     Call aether build N --plan-only\n" +
      "  continue      Call aether continue --plan-only\n" +
      "  lifecycle     Full plan->build->continue (not yet implemented)\n\n" +
      "Options:\n" +
      "  --cwd <path>  Working directory\n"
  );
}

async function main(): Promise<void> {
  const { command, cwd, positional } = parseArgs(process.argv);

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
      process.stderr.write(
        "Lifecycle orchestration not yet implemented (planned for Plan 03)\n"
      );
      process.exit(0);
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
