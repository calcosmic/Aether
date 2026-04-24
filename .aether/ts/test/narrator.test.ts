import assert from "node:assert/strict";
import test from "node:test";
import { parseEvent, renderEvent, sanitizeTerminalText } from "../narrator.js";

test("renders ceremony event identity and status", () => {
  const event = parseEvent(
    JSON.stringify({
      topic: "ceremony.build.spawn",
      payload: {
        phase: 2,
        wave: 1,
        caste: "builder",
        name: "Mason-67",
        status: "starting",
        message: "Implement narrator foundation"
      }
    })
  );

  assert.ok(event);
  const rendered = renderEvent(event);

  assert.match(rendered, /\[CEREMONY\]/);
  assert.match(rendered, /ceremony\.build\.spawn/);
  assert.match(rendered, /phase=2/);
  assert.match(rendered, /wave=1/);
  assert.match(rendered, /builder:Mason-67/);
  assert.match(rendered, /status=starting/);
  assert.match(rendered, /Implement narrator foundation/);
});

test("strips terminal control sequences from event fields", () => {
  const rendered = renderEvent({
    topic: "ceremony.build.spawn\u0007",
    payload: {
      caste: "builder\u001B[31m",
      name: "Mason\u001B[0m-67",
      status: "start\u0000ing",
      message: "\u001B[2Jhello\u001F"
    }
  });

  assert.equal(rendered.includes("\u001B"), false);
  assert.equal(rendered.includes("\u0000"), false);
  assert.equal(rendered.includes("\u001F"), false);
  assert.match(rendered, /ceremony\.build\.spawn/);
  assert.match(rendered, /builder:Mason-67/);
  assert.match(rendered, /status=starting/);
  assert.match(rendered, /hello/);
});

test("ignores empty event lines", () => {
  assert.equal(parseEvent("   "), null);
});

test("sanitizeTerminalText preserves printable text", () => {
  assert.equal(sanitizeTerminalText("plain text"), "plain text");
});
