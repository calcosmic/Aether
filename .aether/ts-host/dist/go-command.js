import { callGoJSON } from "./go-bridge.js";
export function assertSafeGoArgs(args) {
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
export function runGoJSONCommand(bridge, args, caller = callGoJSON) {
    return caller(bridge, assertSafeGoArgs(args));
}
