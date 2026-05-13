/**
 * JSON passthrough renderer: returns empty strings for all methods.
 *
 * Rationale: Go runtime already handles AETHER_OUTPUT_MODE=json.
 * The narrator should not write visual output in json mode.
 */

import type { CeremonyConfig } from "../caste-config.js";
import type { CeremonyPayload } from "../types.js";

export interface JSONRenderer {
  renderBanner(title: string, font?: string): string;
  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
  renderStageSeparator(stage: string, config: CeremonyConfig): string;
  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string;
}

export const jsonRenderer: JSONRenderer = {
  renderBanner(): string {
    return "";
  },

  renderSpawnFrame(): string {
    return "";
  },

  renderStageSeparator(): string {
    return "";
  },

  renderBox(): string {
    return "";
  },
};
