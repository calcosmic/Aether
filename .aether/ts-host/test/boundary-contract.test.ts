/**
 * Boundary contract tests: runtime enforcement of the Go/TS boundary.
 *
 * Verifies that write attempts to `.aether/data/` are rejected, read attempts
 * on allowlisted paths are permitted, and BoundaryViolationError behaves correctly.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve } from "node:path";

import {
  assertNoDirectDataWrites,
  writeCompletionFile,
} from "../src/go-bridge.js";
import {
  assertNoWriteToData,
  BoundaryViolationError,
  GO_OWNED_PATHS,
  ALLOWED_READ_PATHS,
} from "../src/boundary-reference.js";

describe("boundary-contract", () => {
  it("BoundaryViolationError has correct name and message", () => {
    const err = new BoundaryViolationError("test message");
    assert.equal(err.name, "BoundaryViolationError");
    assert.ok(err.message.includes("test message"));
  });

  it("assertNoWriteToData throws for .aether/data/event-bus.jsonl when mode is write", () => {
    assert.throws(
      () => assertNoWriteToData(".aether/data/event-bus.jsonl", "write"),
      BoundaryViolationError,
      "Should throw BoundaryViolationError for write mode"
    );
  });

  it("assertNoWriteToData throws for .aether/data/event-bus.jsonl when mode is omitted", () => {
    assert.throws(
      () => assertNoWriteToData(".aether/data/event-bus.jsonl"),
      BoundaryViolationError,
      "Should throw when mode is omitted"
    );
  });

  it("assertNoWriteToData does NOT throw for .aether/data/event-bus.jsonl when mode is read", () => {
    assert.doesNotThrow(
      () => assertNoWriteToData(".aether/data/event-bus.jsonl", "read"),
      "Should allow read mode on allowlisted path"
    );
  });

  it("assertNoWriteToData rejects other .aether/data/ paths in read mode", () => {
    assert.throws(
      () => assertNoWriteToData(".aether/data/COLONY_STATE.json", "read"),
      BoundaryViolationError,
      "Should reject read on non-allowlisted path"
    );
  });

  it("assertNoDirectDataWrites still rejects all GO_OWNED_PATHS", () => {
    for (const path of GO_OWNED_PATHS) {
      assert.throws(
        () => assertNoDirectDataWrites(path),
        /Boundary violation/,
        `Should reject ${path}`
      );
    }
  });

  it("ALLOWED_READ_PATHS contains event-bus.jsonl", () => {
    assert.ok(
      ALLOWED_READ_PATHS.includes(".aether/data/event-bus.jsonl"),
      "ALLOWED_READ_PATHS should include event-bus.jsonl"
    );
  });

  it("writeCompletionFile respects boundary and writes to tmpdir", () => {
    const path = writeCompletionFile("boundary-contract-test", "test.json", {
      ok: true,
    });
    const resolvedPath = resolve(path);
    const resolvedTmpdir = resolve(tmpdir());

    assert.ok(
      resolvedPath.startsWith(resolvedTmpdir),
      `Should write completion files under tmpdir: ${path}`
    );
    assert.ok(!path.includes(".aether/data"), "Should never write to .aether/data");
    assert.ok(existsSync(path), `Completion file should exist: ${path}`);
    assert.deepEqual(
      JSON.parse(readFileSync(path, "utf-8")),
      { ok: true },
      "Completion file should contain the caller-provided payload"
    );
    assert.doesNotThrow(
      () => assertNoDirectDataWrites(path),
      "Completion files are host-owned temporary finalizer inputs"
    );
  });

  it("keeps planning completion payloads as temp finalizer inputs, not data writes", () => {
    const path = writeCompletionFile("aether-lifecycle", "plan-completion.json", {
      result: {
        plan_manifest: { goal: "boundary test" },
        synthesis: {
          source: "ts-host",
          reason: "test fixture",
        },
      },
    });

    assert.ok(
      resolve(path).startsWith(resolve(tmpdir())),
      `Plan completion file should be temporary: ${path}`
    );
    assert.ok(
      !path.includes(".aether/data"),
      `Plan completion file should not be in Go-owned data paths: ${path}`
    );
    assert.doesNotThrow(
      () => assertNoDirectDataWrites(path),
      "Plan completion files are safe host-owned inputs to plan-finalize"
    );
    assert.throws(
      () => assertNoWriteToData(".aether/data/planning/phase-plan.json", "write"),
      BoundaryViolationError,
      "Persisted planning artifacts are Go finalizer-owned"
    );
    assert.throws(
      () => assertNoWriteToData(".aether/data/COLONY_STATE.json", "write"),
      BoundaryViolationError,
      "Colony state remains Go finalizer-owned"
    );
  });
});
