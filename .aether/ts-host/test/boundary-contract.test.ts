/**
 * Boundary contract tests: runtime enforcement of the Go/TS boundary.
 *
 * Verifies that write attempts to `.aether/data/` are rejected, read attempts
 * on allowlisted paths are permitted, and BoundaryViolationError behaves correctly.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

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
    assert.ok(!path.includes(".aether/data"), "Should never write to .aether/data");
  });
});
