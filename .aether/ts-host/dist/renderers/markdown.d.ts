/**
 * Plain text renderer for non-TTY environments.
 *
 * Delegates to the visual renderer, then strips ANSI codes while preserving
 * structure and emojis.
 */
import type { CeremonyConfig } from "../caste-config.js";
import type { CeremonyPayload } from "../types.js";
export interface MarkdownRenderer {
    renderBanner(title: string, font?: string): string;
    renderSpawnFrame(payload: CeremonyPayload, config: CeremonyConfig): string;
    renderStageSeparator(stage: string, config: CeremonyConfig): string;
    renderBox(content: string, options?: {
        borderStyle?: string;
        borderColor?: string;
    }): string;
}
export declare const markdownRenderer: MarkdownRenderer;
