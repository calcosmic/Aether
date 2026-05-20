/**
 * Plain text renderer for non-TTY environments.
 *
 * Delegates to the visual renderer, then strips ANSI codes while preserving
 * structure and emojis.
 */
import stripAnsi from "strip-ansi";
import { visualRenderer } from "./visual.js";
export const markdownRenderer = {
    renderBanner(title, font) {
        return stripAnsi(visualRenderer.renderBanner(title, font));
    },
    renderSpawnFrame(payload, config) {
        return stripAnsi(visualRenderer.renderSpawnFrame(payload, config));
    },
    renderStageSeparator(stage, config) {
        return stripAnsi(visualRenderer.renderStageSeparator(stage, config));
    },
    renderBox(content, options) {
        return stripAnsi(visualRenderer.renderBox(content, options));
    },
};
