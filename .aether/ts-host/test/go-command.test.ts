import { describe, it } from "node:test";
import assert from "node:assert/strict";

import { assertSafeGoArgs, runGoJSONCommand } from "../src/go-command.js";
import type { GoBridgeOptions } from "../src/go-bridge.js";

describe("go command wrapper", () => {
  it("passes argv arrays through without splitting arguments that contain spaces", () => {
    const bridge: GoBridgeOptions = { goBinaryPath: "/bin/aether", cwd: "/repo" };
    let captured: string[] = [];

    const result = runGoJSONCommand(
      bridge,
      ["swarm", "--plan-only", "Auth panic when session is missing"],
      <T>(_bridge: GoBridgeOptions, args: string[]): T => {
        captured = args;
        return { ok: true } as T;
      }
    );

    assert.deepEqual(captured, ["swarm", "--plan-only", "Auth panic when session is missing"]);
    assert.deepEqual(result, { ok: true });
  });

  it("rejects missing command args before invoking Go", () => {
    assert.throws(() => assertSafeGoArgs([]), /must include a command/);
  });

  it("rejects null bytes in command args", () => {
    assert.throws(() => assertSafeGoArgs(["plan", "bad\0arg"]), /null byte/);
  });
});
