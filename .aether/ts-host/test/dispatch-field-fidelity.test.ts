/**
 * Field-fidelity invariant for `toWorkerDispatches`, the Go->worker
 * conversion boundary (CR-01/CR-02 from 164-VERIFICATION.md).
 *
 * `toWorkerDispatches` silently dropped `permission_profile` and `brief` for
 * months. Every test that touched this boundary mocked it, so the drop never
 * surfaced. These tests drive the REAL exported function -- no mock -- so a
 * future dropped field fails loudly here instead of at Go's exact-equality
 * permission check in production.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";
import * as fs from "node:fs";
import * as path from "node:path";

import { toWorkerDispatches } from "../src/host.js";
import { permissionProfileForCaste } from "../src/worker-dispatch.js";
import { REPO_ROOT } from "./repo-root.js";

// ---------------------------------------------------------------------------
// Field classification (module-level so it documents itself)
// ---------------------------------------------------------------------------

/**
 * Fields that `toWorkerDispatches` intentionally does NOT map through to the
 * worker dispatch. Each entry is here because it either never reaches the
 * worker boundary (result-shaped fields the host writes back to the
 * manifest, not fields a worker consumes) or is superseded by a different
 * output field checked separately. A future Go field added to
 * `PlanningDispatch`/`ContinueExternalDispatch` and not classified here fails
 * Test 1 until someone consciously decides which bucket it belongs in.
 */
const INTENTIONALLY_UNMAPPED = new Set<string>([
  // Plan-side: manifest bookkeeping / result fields, not worker input.
  "agent_name", // agent selection resolved server-side by caste, not forwarded verbatim
  "outputs", // manifest's expected-output list; the worker learns its output path from task_brief
  "files_created", // populated by the worker's own result, never an input field
  "files_modified", // populated by the worker's own result, never an input field
  "scout_report", // populated by the worker's own result, never an input field
  "phase_plan", // populated by the worker's own result, never an input field
  // Continue-side: same class of result/manifest-only fields.
  "timeout_seconds", // continue's per-dispatch timeout is applied by the host's DispatchOptions, not copied onto BuildDispatch
  "report", // populated by the worker's own result, never an input field
  "findings", // populated by the worker's own result, never an input field
  "issues", // populated by the worker's own result, never an input field
  "recommendations", // populated by the worker's own result, never an input field
  "weak_spots", // populated by the worker's own result, never an input field
  "edge_cases_discovered", // populated by the worker's own result, never an input field
  "reusable_lessons", // populated by the worker's own result, never an input field
  "handoff", // populated by the worker's own result, never an input field
]);

/**
 * Fields that survive under a different name on the output. `brief` (Go's
 * plan-time field name) becomes `task_brief` (the field the worker prompt
 * actually reads), because build/continue paths already inject `task_brief`
 * directly and plan-time dispatches never had that host-injected field.
 */
const RENAMED: Record<string, string> = {
  brief: "task_brief", // Go emits `brief`; the worker request field is `task_brief`
};

// ---------------------------------------------------------------------------
// Test 1: full-field invariant over a Go-shaped plan dispatch
// ---------------------------------------------------------------------------

describe("toWorkerDispatches field fidelity", () => {
  it("preserves every PlanningDispatch field not explicitly classified as unmapped or renamed", () => {
    const scoutPermissionProfile = {
      schema_version: 1,
      name: "workspace_write",
      filesystem: "workspace_write",
      shell: "within_filesystem_boundary",
      network: "provider_default",
      approval: "never",
      behavioral_restrictions: [
        "write phase research artifacts under .aether/data/phase-research only",
      ],
    };

    const planSourceDispatch: Record<string, unknown> = {
      stage: "phase_research",
      wave: 1,
      execution_wave: 1,
      caste: "scout",
      agent_name: "aether-scout",
      name: "Antenna-1",
      task: "Research domain knowledge for phase 3: Wire exporter",
      task_id: "plan-research-phase-3",
      outputs: ["phase-3-research.md"],
      status: "planned",
      summary: "research summary",
      blockers: ["blocked on survey"],
      duration: 12.5,
      brief: "## Output\nfindings body\n",
      files_created: ["a.md"],
      files_modified: ["b.md"],
      scout_report: { findings: [], gaps: [] },
      phase_plan: { phases: [] },
      skill_section: "matched skill guidance",
      skill_count: 2,
      colony_skill_count: 1,
      domain_skill_count: 1,
      matched_skills: ["skill-a", "skill-b"],
      permission_profile: scoutPermissionProfile,
    };

    const [output] = toWorkerDispatches([planSourceDispatch] as never);
    assert.ok(output, "expected toWorkerDispatches to return one dispatch");
    const outputRecord = output as unknown as Record<string, unknown>;

    for (const key of Object.keys(planSourceDispatch)) {
      if (INTENTIONALLY_UNMAPPED.has(key)) continue;
      const outputKey = RENAMED[key] ?? key;
      assert.deepStrictEqual(
        outputRecord[outputKey],
        planSourceDispatch[key],
        `expected field "${key}" to survive toWorkerDispatches as "${outputKey}"`
      );
    }
  });

  it("carries a Go-shaped scout permission_profile through unchanged", () => {
    const scoutProfile = {
      schema_version: 1,
      name: "workspace_write",
      filesystem: "workspace_write",
      shell: "within_filesystem_boundary",
      network: "provider_default",
      approval: "never",
      behavioral_restrictions: [
        "write phase research artifacts under .aether/data/phase-research only",
      ],
    };
    const scoutDispatch: Record<string, unknown> = {
      stage: "phase_research",
      wave: 1,
      caste: "scout",
      name: "Antenna-2",
      task: "Research domain knowledge for phase 4",
      status: "planned",
      outputs: ["phase-4-research.md"],
      permission_profile: scoutProfile,
    };

    const [output] = toWorkerDispatches([scoutDispatch] as never);

    assert.ok(output, "expected toWorkerDispatches to return one dispatch");
    assert.deepStrictEqual(output.permission_profile, scoutProfile);
  });

  it("promotes a phase_research brief's six research sections into task_brief, and lets an existing task_brief win", () => {
    const sixSectionBrief = [
      "## Output",
      "## Hive Wisdom (Pre-existing Knowledge)",
      "## Key Patterns",
      "## External Context",
      "## Gotchas",
      "## Recommended Approach",
      "## Files to Study",
    ].join("\n");

    const briefOnlyDispatch: Record<string, unknown> = {
      stage: "phase_research",
      wave: 1,
      caste: "scout",
      name: "Antenna-3",
      task: "Research domain knowledge for phase 5",
      status: "planned",
      outputs: ["phase-5-research.md"],
      brief: sixSectionBrief,
    };

    const [briefOutput] = toWorkerDispatches([briefOnlyDispatch] as never);
    assert.ok(briefOutput, "expected toWorkerDispatches to return one dispatch");
    const taskBrief = String(briefOutput.task_brief ?? "");
    for (const heading of [
      "## Output",
      "## Hive Wisdom (Pre-existing Knowledge)",
      "## Key Patterns",
      "## External Context",
      "## Gotchas",
      "## Recommended Approach",
      "## Files to Study",
    ]) {
      assert.ok(
        taskBrief.includes(heading),
        `expected task_brief to include heading "${heading}", got: ${taskBrief}`
      );
    }

    // Build/continue paths already inject `task_brief` directly; that
    // host-injected value must keep winning over a Go-emitted `brief`.
    const bothFieldsDispatch: Record<string, unknown> = {
      ...briefOnlyDispatch,
      name: "Antenna-4",
      task_brief: "host-injected build-path brief",
    };
    const [bothOutput] = toWorkerDispatches([bothFieldsDispatch] as never);
    assert.ok(bothOutput, "expected toWorkerDispatches to return one dispatch");
    assert.strictEqual(bothOutput.task_brief, "host-injected build-path brief");
  });

  it("mirrors Go's repositoryReadOnlyCastes map exactly, so the TS fallback cannot drift again", () => {
    const goSourcePath = path.join(REPO_ROOT, "pkg", "codex", "permission_profile.go");
    const goSource = fs.readFileSync(goSourcePath, "utf8");

    const mapLiteralMatch = goSource.match(
      /var repositoryReadOnlyCastes = map\[string\]struct\{\}\{([\s\S]*?)\n\}/
    );
    assert.ok(mapLiteralMatch, "could not locate repositoryReadOnlyCastes map literal in permission_profile.go");
    const mapBody = mapLiteralMatch![1] ?? "";

    const readOnlyCastes: string[] = [];
    const entryPattern = /"([a-z_]+)":\s*\{\}/g;
    let entryMatch: RegExpExecArray | null;
    while ((entryMatch = entryPattern.exec(mapBody)) !== null) {
      const caste = entryMatch[1];
      if (caste !== undefined) readOnlyCastes.push(caste);
    }
    assert.ok(readOnlyCastes.length > 0, "expected at least one read-only caste in permission_profile.go");

    for (const caste of readOnlyCastes) {
      assert.strictEqual(
        permissionProfileForCaste(caste).filesystem,
        "repository_read_only",
        `expected caste "${caste}" (Go read-only set) to resolve to repository_read_only in the TS fallback`
      );
    }

    for (const caste of ["scout", "route_setter", "builder", "oracle"]) {
      assert.notStrictEqual(
        readOnlyCastes.includes(caste),
        true,
        `test fixture assumption broken: "${caste}" unexpectedly appears in Go's read-only set`
      );
      assert.strictEqual(
        permissionProfileForCaste(caste).filesystem,
        "workspace_write",
        `expected caste "${caste}" to resolve to workspace_write in the TS fallback, matching Go's canonical map`
      );
    }
  });
});
