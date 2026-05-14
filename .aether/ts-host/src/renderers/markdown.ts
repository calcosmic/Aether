/**
 * Plain text renderer for non-TTY environments.
 *
 * Delegates to the visual renderer, then strips ANSI codes while preserving
 * structure and emojis.
 */

import stripAnsi from "strip-ansi";
import { visualRenderer } from "./visual.js";
import type { CeremonyConfig } from "../caste-config.js";
import type { CeremonyPayload } from "../types.js";
import type { VisualRenderer } from "./visual.js";

export interface MarkdownRenderer {
  renderBanner(title: string, font?: string): string;
  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
  renderStageSeparator(stage: string, config: CeremonyConfig): string;
  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string;
}

export const markdownRenderer: MarkdownRenderer = {
  renderBanner(title: string, font?: string): string {
    return stripAnsi(visualRenderer.renderBanner(title, font));
  },

  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string {
    return stripAnsi(visualRenderer.renderSpawnFrame(payload, config));
  },

  renderStageSeparator(stage: string, config: CeremonyConfig): string {
    return stripAnsi(visualRenderer.renderStageSeparator(stage, config));
  },

  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string {
    return stripAnsi(visualRenderer.renderBox(content, options));
  },
};
