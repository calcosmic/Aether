import { describe, it, expect } from "vitest";
import { readFileSync } from "fs";
import { EventSchema, parseEventLine } from "../../src/schemas/event.schema.js";
import { fixturePath } from "../helpers.js";

describe("EventSchema", () => {
  it("validates sample.ndjson lines", () => {
    const lines = readFileSync(fixturePath("events", "sample.ndjson"), "utf8")
      .split("\n")
      .filter(Boolean);
    expect(lines.length).toBeGreaterThan(0);
    lines.forEach((line) => {
      const result = parseEventLine(line);
      expect(result.type).toBeTruthy();
      expect(result.timestamp).toBeTruthy();
      expect(result.payload).toBeDefined();
    });
  });

  it("rejects empty line", () => {
    expect(() => parseEventLine("")).toThrow(/Empty/);
  });

  it("rejects invalid JSON", () => {
    expect(() => parseEventLine("not json")).toThrow();
  });

  it("rejects missing timestamp", () => {
    expect(() => parseEventLine('{"type":"x"}')).toThrow();
  });

  it("rejects empty type", () => {
    expect(() =>
      parseEventLine('{"type":"","timestamp":"2026-05-22T10:00:00Z","payload":{}}')
    ).toThrow();
  });
});
