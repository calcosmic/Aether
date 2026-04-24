import assert from "node:assert/strict";
import test from "node:test";
import {
  applyEventToFrame,
  createCeremonyFrame,
  parseEvent,
  parseVisualContract,
  renderActivityFrame,
  renderEvent,
  sanitizeTerminalText
} from "../narrator.js";

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

test("renders caste identity from Go-owned visual metadata", () => {
  const visuals = parseVisualContract({
    ok: true,
    result: {
      castes: {
        builder: {
          emoji: "🔨",
          color: "33",
          label: "Builder"
        }
      }
    }
  });

  const rendered = renderEvent(
    {
      topic: "ceremony.build.spawn",
      payload: {
        caste: "builder",
        name: "Mason-67",
        status: "starting"
      }
    },
    visuals
  );

  assert.match(rendered, /🔨 Builder:Mason-67/);
  assert.match(rendered, /status=starting/);
});

test("applies ANSI color from Go-owned visual metadata when forced", () => {
  const previous = process.env.AETHER_FORCE_COLOR;
  const previousNoColor = process.env.NO_COLOR;
  process.env.AETHER_FORCE_COLOR = "1";
  delete process.env.NO_COLOR;
  try {
    const visuals = parseVisualContract({
      castes: {
        builder: {
          emoji: "🔨",
          color: "33",
          label: "Builder"
        }
      }
    });

    const rendered = renderEvent(
      {
        topic: "ceremony.build.spawn",
        payload: {
          caste: "builder",
          name: "Mason-67",
          status: "starting"
        }
      },
      visuals
    );

    assert.match(rendered, /\u001B\[33mBuilder\u001B\[0m:Mason-67/);
  } finally {
    if (previous === undefined) {
      delete process.env.AETHER_FORCE_COLOR;
    } else {
      process.env.AETHER_FORCE_COLOR = previous;
    }
    if (previousNoColor === undefined) {
      delete process.env.NO_COLOR;
    } else {
      process.env.NO_COLOR = previousNoColor;
    }
  }
});

test("ignores empty event lines", () => {
  assert.equal(parseEvent("   "), null);
});

test("sanitizeTerminalText preserves printable text", () => {
  assert.equal(sanitizeTerminalText("plain text"), "plain text");
});

test("renders rolling activity frame with active and completed workers", () => {
  const visuals = parseVisualContract({
    castes: {
      builder: { emoji: "🔨", label: "Builder" },
      watcher: { emoji: "👁️", label: "Watcher" }
    }
  });
  const frame = createCeremonyFrame();

  for (const raw of [
    {
      topic: "ceremony.build.wave.start",
      payload: {
        phase: 2,
        phase_name: "Narrator launcher",
        wave: 1,
        total: 2,
        status: "starting"
      }
    },
    {
      topic: "ceremony.build.spawn",
      payload: {
        phase: 2,
        wave: 1,
        spawn_id: "builder-1",
        caste: "builder",
        name: "Mason-67",
        task_id: "2.1",
        task: "Wire launch events",
        status: "running",
        tool_count: 3
      }
    },
    {
      topic: "ceremony.build.spawn",
      payload: {
        phase: 2,
        wave: 1,
        spawn_id: "watcher-1",
        caste: "watcher",
        name: "Vigil-17",
        task_id: "2.2",
        task: "Verify JSON safety",
        status: "completed",
        files_modified: ["cmd/narrator_launcher.go"],
        tests_written: ["cmd/narrator_launcher_test.go"]
      }
    },
    {
      topic: "ceremony.build.wave.end",
      payload: {
        phase: 2,
        phase_name: "Narrator launcher",
        wave: 1,
        completed: 1,
        total: 2,
        status: "completed"
      }
    }
  ]) {
    const event = parseEvent(JSON.stringify(raw));
    assert.ok(event);
    applyEventToFrame(frame, event);
  }

  const rendered = renderActivityFrame(frame, visuals);
  assert.match(rendered, /\[CEREMONY\] ceremony\.build\.wave\.end/);
  assert.match(rendered, /\[CEREMONY\] COLONY ACTIVITY phase=2 Narrator launcher/);
  assert.match(rendered, /Wave 1: 1\/2 completed/);
  assert.match(rendered, /Active:\n  🔨 Builder:Mason-67 running task=2\.1 tools=3 Wire launch events/);
  assert.match(rendered, /Completed:\n  👁️ Watcher:Vigil-17 completed task=2\.2 files=1 tests=1 Verify JSON safety/);
});

test("renders blocked workers and truncates long frame text", () => {
  const frame = createCeremonyFrame();
  const longMessage = "x".repeat(160);
  const event = parseEvent(
    JSON.stringify({
      topic: "ceremony.build.spawn",
      payload: {
        phase: 3,
        wave: 2,
        caste: "builder",
        name: "Mason-13",
        status: "failed",
        blockers: ["broken pipe"],
        message: longMessage
      }
    })
  );
  assert.ok(event);
  applyEventToFrame(frame, event);

  const rendered = renderActivityFrame(frame);
  const workerLine = rendered
    .split("\n")
    .find((line) => line.trim().startsWith("builder:Mason-13"));
  assert.match(rendered, /Blocked:/);
  assert.match(rendered, /builder:Mason-13 failed blockers=1/);
  assert.ok(workerLine);
  assert.doesNotMatch(workerLine, new RegExp(longMessage));
  assert.match(workerLine, /\.\.\./);
});

test("keeps multi-wave activity history while current wave advances", () => {
  const frame = createCeremonyFrame();
  for (const raw of [
    {
      topic: "ceremony.build.wave.start",
      payload: { phase: 4, phase_name: "Multi-wave display", wave: 1, total: 1, status: "starting" }
    },
    {
      topic: "ceremony.build.spawn",
      payload: {
        phase: 4,
        wave: 1,
        spawn_id: "arch-1",
        caste: "archaeologist",
        name: "Archive-9",
        task_id: "4.1",
        task: "Excavate prior ceremony code",
        status: "completed",
        token_count: 1200
      }
    },
    {
      topic: "ceremony.build.wave.end",
      payload: { phase: 4, phase_name: "Multi-wave display", wave: 1, completed: 1, total: 1, status: "completed" }
    },
    {
      topic: "ceremony.build.wave.start",
      payload: { phase: 4, phase_name: "Multi-wave display", wave: 2, total: 2, status: "starting" }
    },
    {
      topic: "ceremony.build.spawn",
      payload: {
        phase: 4,
        wave: 2,
        spawn_id: "builder-2",
        caste: "builder",
        name: "Mason-14",
        task_id: "4.2",
        task: "Restore wrapper bridge",
        status: "running",
        tool_count: 6
      }
    },
    {
      topic: "ceremony.build.spawn",
      payload: {
        phase: 4,
        wave: 2,
        spawn_id: "watcher-2",
        caste: "watcher",
        name: "Vigil-22",
        task_id: "4.3",
        task: "Verify wrapper bridge",
        status: "blocked",
        blockers: ["needs Claude Task-tool smoke"]
      }
    }
  ]) {
    const event = parseEvent(JSON.stringify(raw));
    assert.ok(event);
    applyEventToFrame(frame, event);
  }

  const rendered = renderActivityFrame(frame);
  assert.match(rendered, /Wave 2: 0\/2 starting/);
  assert.match(rendered, /Completed:\n  archaeologist:Archive-9 completed task=4\.1 tokens=1200 Excavate prior ceremony code/);
  assert.match(rendered, /Active:\n  builder:Mason-14 running task=4\.2 tools=6 Restore wrapper bridge/);
  assert.match(rendered, /Blocked:\n  watcher:Vigil-22 blocked task=4\.3 blockers=1 Verify wrapper bridge/);
});

test("tracks non-build lifecycle wave events generically", () => {
  const frame = createCeremonyFrame();
  for (const raw of [
    {
      topic: "ceremony.continue.wave.start",
      payload: {
        phase: 5,
        phase_name: "Continue and plan orchestration",
        wave: 1,
        total: 1,
        status: "starting"
      }
    },
    {
      topic: "ceremony.continue.spawn",
      payload: {
        phase: 5,
        wave: 1,
        spawn_id: "watcher-continue",
        caste: "watcher",
        name: "Vigil-41",
        task_id: "5.2",
        task: "Verify wrapper results",
        status: "running"
      }
    },
    {
      topic: "ceremony.continue.wave.end",
      payload: {
        phase: 5,
        phase_name: "Continue and plan orchestration",
        wave: 1,
        completed: 1,
        total: 1,
        status: "completed"
      }
    }
  ]) {
    const event = parseEvent(JSON.stringify(raw));
    assert.ok(event);
    applyEventToFrame(frame, event);
  }

  const rendered = renderActivityFrame(frame);
  assert.match(rendered, /\[CEREMONY\] COLONY ACTIVITY stage=continue phase=5 Continue and plan orchestration/);
  assert.match(rendered, /Wave 1: 1\/1 completed/);
  assert.match(rendered, /Active:\n  watcher:Vigil-41 running task=5\.2 Verify wrapper results/);
});

test("keeps lifecycle context notices for skills pheromones and chambers", () => {
  const frame = createCeremonyFrame();
  for (const raw of [
    {
      topic: "ceremony.skill.activate",
      payload: {
        phase: 6,
        skill: "typescript",
        status: "active",
        message: "Narrator rendering rules injected"
      }
    },
    {
      topic: "ceremony.pheromone.emit",
      payload: {
        phase: 6,
        pheromone_type: "FOCUS",
        strength: 0.85,
        message: "Surface skill activations"
      }
    },
    {
      topic: "ceremony.chamber.seal",
      payload: {
        phase: 6,
        status: "sealed",
        message: "Lifecycle ceremony checkpoint"
      }
    }
  ]) {
    const event = parseEvent(JSON.stringify(raw));
    assert.ok(event);
    applyEventToFrame(frame, event);
  }

  const rendered = renderActivityFrame(frame);
  assert.match(rendered, /Context:/);
  assert.match(rendered, /ceremony\.skill\.activate active skill=typescript Narrator rendering rules injected/);
  assert.match(rendered, /ceremony\.pheromone\.emit pheromone=FOCUS strength=0\.85 Surface skill activations/);
  assert.match(rendered, /ceremony\.chamber\.seal sealed Lifecycle ceremony checkpoint/);
  assert.match(rendered, /Workers: none active yet/);
});

test("keeps lifecycle context notices for archive and wisdom events", () => {
  const frame = createCeremonyFrame();
  for (const raw of [
    {
      topic: "ceremony.chamber.entomb",
      payload: {
        status: "entombed",
        task_id: "20260424-meta-finish-v16",
        message: "Finish v1.6 ceremony events"
      }
    },
    {
      topic: "ceremony.midden.record",
      payload: {
        status: "recorded",
        task: "testing",
        message: "Smoke failure recorded"
      }
    },
    {
      topic: "ceremony.queen.promote",
      payload: {
        status: "promoted",
        task: "Patterns",
        message: "Prefer narrow event hooks"
      }
    },
    {
      topic: "ceremony.hive.store",
      payload: {
        status: "stored",
        task: "docs",
        message: "Keep handoff current"
      }
    },
    {
      topic: "ceremony.hive.promote",
      payload: {
        status: "promoted",
        task: "general",
        message: "Prefer focused fixes"
      }
    }
  ]) {
    const event = parseEvent(JSON.stringify(raw));
    assert.ok(event);
    applyEventToFrame(frame, event);
  }

  const rendered = renderActivityFrame(frame);
  assert.match(rendered, /ceremony\.chamber\.entomb entombed Finish v1\.6 ceremony events/);
  assert.match(rendered, /ceremony\.midden\.record recorded Smoke failure recorded/);
  assert.match(rendered, /ceremony\.queen\.promote promoted Prefer narrow event hooks/);
  assert.match(rendered, /ceremony\.hive\.store stored Keep handoff current/);
  assert.match(rendered, /ceremony\.hive\.promote promoted Prefer focused fixes/);
});
