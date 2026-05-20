import { callGoJSON, type GoBridgeOptions } from "./go-bridge.js";
export type GoJSONCaller = typeof callGoJSON;
export declare function assertSafeGoArgs(args: readonly string[]): string[];
export declare function runGoJSONCommand<T>(bridge: GoBridgeOptions, args: readonly string[], caller?: GoJSONCaller): T;
