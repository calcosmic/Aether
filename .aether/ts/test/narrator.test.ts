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

test("renders the shared Go ceremony payload shape", () => {
  const rendered = renderEvent({
    topic: "ceremony.build.wave.end",
    payload: {
      phase: 2,
      phase_name: "Event protocol",
      wave: 3,
      spawn_id: "spawn_123",
      caste: "watcher",
      name: "Vigil-17",
      task_id: "2.2",
      task: "Verify stream protocol",
      status: "complete",
      skill: "testing",
      pheromone_type: "FOCUS",
      strength: 0.8,
      completed: 2,
      total: 3,
      tool_count: 4,
      token_count: 1200,
      files_created: ["a"],
      files_modified: ["b", "c"],
      tests_written: ["d"],
      blockers: ["none"],
      success_criteria: ["green"]
    }
  });

  assert.match(rendered, /phase_name=Event protocol/);
  assert.match(rendered, /spawn=spawn_123/);
  assert.match(rendered, /watcher:Vigil-17/);
  assert.match(rendered, /task_id=2\.2/);
  assert.match(rendered, /skill=testing/);
  assert.match(rendered, /pheromone=FOCUS/);
  assert.match(rendered, /strength=0\.8/);
  assert.match(rendered, /progress=2\/3/);
  assert.match(rendered, /tools=4/);
  assert.match(rendered, /tokens=1200/);
  assert.match(rendered, /created=1/);
  assert.match(rendered, /modified=2/);
  assert.match(rendered, /tests=1/);
  assert.match(rendered, /blockers=1/);
  assert.match(rendered, /criteria=1/);
  assert.match(rendered, /status=complete/);
  assert.match(rendered, /Verify stream protocol/);
});

test("ignores empty event lines", () => {
  assert.equal(parseEvent("   "), null);
});

test("sanitizeTerminalText preserves printable text", () => {
  assert.equal(sanitizeTerminalText("plain text"), "plain text");
});
