/**
 * Claims parser unit tests.
 *
 * Verifies direct JSON parsing, code-fence stripping, trailing JSON block
 * extraction, error handling, and validation.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  parseWorkerClaims,
  stripCodeFences,
  extractJSONBlock,
  validateWorkerClaims,
} from "../src/claims-parser.js";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("claims-parser", () => {
  it("parseWorkerClaims parses direct JSON", () => {
    const stdout = JSON.stringify({
      status: "completed",
      summary: "done",
      ant_name: "Builder-01",
      caste: "builder",
    });
    const claims = parseWorkerClaims(stdout);
    assert.equal(claims.status, "completed");
    assert.equal(claims.summary, "done");
    assert.equal(claims.ant_name, "Builder-01");
  });

  it("parseWorkerClaims strips code fences", () => {
    const stdout = '```json\n{"status":"completed","summary":"done"}\n```';
    const claims = parseWorkerClaims(stdout);
    assert.equal(claims.status, "completed");
    assert.equal(claims.summary, "done");
  });

  it("parseWorkerClaims extracts trailing JSON block", () => {
    const stdout = 'some text\nmore text\n{"status":"completed","summary":"done"}';
    const claims = parseWorkerClaims(stdout);
    assert.equal(claims.status, "completed");
    assert.equal(claims.summary, "done");
  });

  it("parseWorkerClaims throws on unparseable output", () => {
    assert.throws(
      () => parseWorkerClaims("not json"),
      /Failed to parse worker claims/,
      "Should throw for unparseable output"
    );
  });

  it("validateWorkerClaims throws on missing required field", () => {
    assert.throws(
      () => validateWorkerClaims({ summary: "done" }),
      /missing required field: status/,
      "Should throw when status is missing"
    );
  });

  it("stripCodeFences removes json code fences", () => {
    const result = stripCodeFences('```json\n{"a":1}\n```');
    assert.equal(result, '{"a":1}');
  });

  it("stripCodeFences removes generic code fences", () => {
    const result = stripCodeFences('```\n{"a":1}\n```');
    assert.equal(result, '{"a":1}');
  });

  it("extractJSONBlock finds last JSON object", () => {
    const result = extractJSONBlock('prefix {"a":1} suffix');
    assert.equal(result, '{"a":1}');
  });

  it("extractJSONBlock throws when no JSON present", () => {
    assert.throws(
      () => extractJSONBlock("no braces here"),
      /No JSON block found/,
      "Should throw when no JSON block exists"
    );
  });
});
