import { callGoJSON, type GoBridgeOptions } from "./go-bridge.js";

export type GoJSONCaller = typeof callGoJSON;

export function assertSafeGoArgs(args: readonly string[]): string[] {
  if (args.length === 0) {
    throw new Error("Go command args must include a command");
  }
  return args.map((arg, index) => {
    if (typeof arg !== "string") {
      throw new Error(`Go command arg ${index} must be a string`);
    }
    if (arg.includes("\0")) {
      throw new Error(`Go command arg ${index} contains a null byte`);
    }
    return arg;
  });
}

export function runGoJSONCommand<T>(
  bridge: GoBridgeOptions,
  args: readonly string[],
  caller: GoJSONCaller = callGoJSON
): T {
  return caller<T>(bridge, assertSafeGoArgs(args));
}
