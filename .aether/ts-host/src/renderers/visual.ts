/**
 * ANSI visual renderer: produces colored terminal output with figlet banners,
 * caste identity frames, stage separators, and boxen-framed boxes.
 *
 * All methods return strings (do NOT write to stdout).
 */

import chalk from "chalk";
import figlet from "figlet";
import boxen from "boxen";
import {
  getCasteEmoji,
  getCasteColor,
  getCasteLabel,
  type CeremonyConfig,
} from "../caste-config.js";
import type { CeremonyPayload } from "../types.js";

export interface VisualRenderer {
  renderBanner(title: string, font?: string): string;
  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
  renderStageSeparator(stage: string, config: CeremonyConfig): string;
  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string;
}

export const visualRenderer: VisualRenderer = {
  renderBanner(title: string, font?: string): string {
    const banner = figlet.textSync(title, { font: font ?? "Standard" });
    return banner
      .split("\n")
      .map((line) => chalk.cyan(line))
      .join("\n");
  },

  renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string {
    const emoji = getCasteEmoji(config, payload.caste ?? "");
    const label = getCasteLabel(config, payload.caste ?? "");
    const color = getCasteColor(config, payload.caste ?? "");
    return `${emoji} ${chalk.hex(color)(label)} ${payload.name ?? ""}  ${payload.task ?? ""}\n`;
  },

  renderStageSeparator(stage: string, config: CeremonyConfig): string {
    const prefix = config.stage_separator.prefix;
    const suffix = config.stage_separator.suffix;
    return `${prefix}${stage}${suffix}\n`;
  },

  renderBox(
    content: string,
    options?: { borderStyle?: string; borderColor?: string }
  ): string {
    return boxen(content, {
      padding: 1,
      margin: 1,
      borderStyle: (options?.borderStyle ?? "round") as "single" | "double" | "round" | "bold" | "singleDouble" | "doubleSingle" | "classic",
      borderColor: options?.borderColor ?? "green",
    });
  },
};
