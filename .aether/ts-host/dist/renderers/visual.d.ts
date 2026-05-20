/**
 * ANSI visual renderer: produces colored terminal output with figlet banners,
 * caste identity frames, stage separators, and boxen-framed boxes.
 *
 * All methods return strings (do NOT write to stdout).
 */
import { type CeremonyConfig } from "../caste-config.js";
import type { CeremonyPayload } from "../types.js";
export interface VisualRenderer {
    renderBanner(title: string, font?: string): string;
    renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
    renderStageSeparator(stage: string, config: CeremonyConfig): string;
    renderBox(content: string, options?: {
        borderStyle?: string;
        borderColor?: string;
    }): string;
}
export declare const visualRenderer: VisualRenderer;
