/**
 * Ceremony narrator: subscribes to the event bridge and dispatches events
 * to the correct renderer based on output mode and TTY status.
 *
 * Writes rendered output directly to process.stdout in real-time.
 */

import {
  loadCeremonyConfig,
  getCasteEmoji,
  getCasteColor,
  getCasteLabel,
  type CeremonyConfig,
} from "./caste-config.js";
import type { CeremonyEvent, CeremonyPayload } from "./types.js";
import { visualRenderer } from "./renderers/visual.js";
import { markdownRenderer } from "./renderers/markdown.js";
import { jsonRenderer } from "./renderers/json.js";
import { loadTemplate, substituteTemplate } from "./template-loader.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface NarratorOptions {
  cwd: string;
  outputMode?: string | undefined;
  /** When true, narrator suppresses stdout writes (e.g. when dashboard is active). */
  suppressOutput?: boolean;
}

export interface Narrator {
  onEvent(event: CeremonyEvent): void;
  stop(): void;
}

interface Renderer {
  renderBanner(title: string, font?: string): string;
  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
  renderStageSeparator(stage: string, config: CeremonyConfig): string;
  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string;
}

// ---------------------------------------------------------------------------
// Narrator factory
// ---------------------------------------------------------------------------

/**
 * Create a ceremony narrator that renders events to stdout.
 *
 * @param opts - Narrator options (cwd and optional output mode override).
 * @returns A narrator with onEvent and stop methods.
 */
export function createNarrator(opts: NarratorOptions): Narrator {
  const config = loadCeremonyConfig(opts.cwd);
  const suppressOutput = opts.suppressOutput ?? false;

  const mode =
    opts.outputMode ??
    process.env["AETHER_OUTPUT_MODE"] ??
    "visual";

  const renderer = selectRenderer(mode);

  const handlers: Record<string, (payload: CeremonyPayload) => string> = {
    "ceremony.build.spawn": (payload) =>
      renderer.renderSpawnFrame(payload, config),
    "ceremony.build.wave.start": () =>
      renderer.renderStageSeparator("Build", config),
    "ceremony.build.wave.end": () =>
      renderer.renderStageSeparator("Build Complete", config),
    "ceremony.chamber.seal": () =>
      renderer.renderBanner("CROWNED ANTHILL"),
    "ceremony.build.summary": (payload) => {
      const content = buildSummaryContent(payload);
      return renderer.renderBox(content, { borderStyle: "round", borderColor: "green" });
    },
    "ceremony.build.closeout": (payload) => {
      const content = buildCloseoutContent(payload);
      return renderer.renderBox(content, { borderStyle: "double", borderColor: "cyan" });
    },
    "ceremony.oracle.phase_transition": (payload) =>
      renderer.renderStageSeparator(
        `Oracle: ${payload.status ?? "unknown"}`,
        config
      ),
    "ceremony.oracle.iteration": (payload) =>
      renderer.renderSpawnFrame(
        {
          caste: "oracle",
          name: `Oracle-${payload.wave ?? 0}`,
          task: `Researching: ${payload.task ?? ""}`,
        },
        config
      ),
  };

  return {
    onEvent(event: CeremonyEvent): void {
      if (suppressOutput) return;

      const handler = handlers[event.topic];
      if (!handler) return;

      const output = handler(event.payload);
      if (output) {
        process.stdout.write(output + "\n");
      }
    },

    stop(): void {
      // No-op: bridge controller handles subprocess cleanup.
    },
  };
}

// ---------------------------------------------------------------------------
// Renderer selection
// ---------------------------------------------------------------------------

function selectRenderer(mode: string): Renderer {
  switch (mode) {
    case "json":
      return jsonRenderer;
    case "markdown":
      return markdownRenderer;
    case "visual":
      return process.stdout.isTTY ? visualRenderer : markdownRenderer;
    default:
      return markdownRenderer;
  }
}

// ---------------------------------------------------------------------------
// Content builders
// ---------------------------------------------------------------------------

function buildSummaryContent(payload: CeremonyPayload): string {
  const parts: string[] = [];
  if (payload.completed !== undefined && payload.total !== undefined) {
    parts.push(`Completed: ${payload.completed}/${payload.total}`);
  }
  if (payload.tool_count !== undefined) {
    parts.push(`Tools: ${payload.tool_count}`);
  }
  if (payload.token_count !== undefined) {
    parts.push(`Tokens: ${payload.token_count}`);
  }
  if (payload.files_created && payload.files_created.length > 0) {
    parts.push(`Files created: ${payload.files_created.length}`);
  }
  if (payload.files_modified && payload.files_modified.length > 0) {
    parts.push(`Files modified: ${payload.files_modified.length}`);
  }
  if (payload.tests_written && payload.tests_written.length > 0) {
    parts.push(`Tests: ${payload.tests_written.length}`);
  }
  if (payload.blockers && payload.blockers.length > 0) {
    parts.push(`Blockers: ${payload.blockers.length}`);
  }
  return parts.join("\n") || "Build summary";
}

function buildCloseoutContent(payload: CeremonyPayload): string {
  const parts: string[] = [];
  if (payload.phase !== undefined) {
    parts.push(`Phase ${payload.phase}`);
  }
  if (payload.status) {
    parts.push(`Status: ${payload.status}`);
  }
  if (payload.message) {
    parts.push(payload.message);
  }
  return parts.join("\n") || "Closeout";
}
