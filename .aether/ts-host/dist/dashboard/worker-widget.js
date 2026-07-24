/**
 * Per-worker display state and rendering.
 *
 * Each worker gets a widget with an ora spinner, progress bar,
 * tool/token counts, and elapsed duration. The widget is updated
 * in-place as events arrive.
 */
import ora from "ora";
import chalk from "chalk";
import { getCasteEmoji, getCasteColor, getCasteLabel, } from "../caste-config.js";
// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------
/**
 * Create a new worker widget with an optional ora spinner.
 */
export function createWorkerWidget(state, config) {
    const spinner = ora({ text: "", spinner: "dots" });
    if (state.status === "active") {
        spinner.start();
    }
    else {
        spinner.stop();
    }
    return { state, spinner };
}
/**
 * Render a worker widget as a formatted string.
 */
export function renderWorkerWidget(widget, config) {
    const { state } = widget;
    const emoji = getCasteEmoji(config, state.caste);
    const label = chalk.hex(getCasteColor(config, state.caste))(getCasteLabel(config, state.caste));
    const name = state.name;
    const progressPct = Math.min(100, Math.round((state.tool_count / 20) * 100));
    const progressBar = renderProgressBar(progressPct, 10);
    const toolCount = state.tool_count;
    const tokenCount = formatTokenCount(state.token_count);
    const elapsed = formatDuration(Date.now() - state.startTime);
    const statusIndicator = renderStatusIndicator(state.status);
    return `${emoji} ${label} ${name.padEnd(16)} ${progressBar} ${String(progressPct).padStart(3)}%  Tools:${String(toolCount).padStart(3)}  Tokens:${tokenCount.padStart(6)}  ${elapsed}  ${statusIndicator}`;
}
/**
 * Update a worker widget's state and sync the spinner.
 */
export function updateWorkerWidget(widget, state) {
    widget.state = state;
    if (state.status === "active") {
        if (!widget.spinner?.isSpinning) {
            widget.spinner?.start();
        }
    }
    else {
        widget.spinner?.stop();
    }
}
/**
 * Format a duration in milliseconds as MM:SS (or HH:MM:SS if >= 1 hour).
 */
export function formatDuration(ms) {
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    const pad = (n) => String(n).padStart(2, "0");
    if (hours > 0) {
        return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
    }
    return `${pad(minutes)}:${pad(seconds)}`;
}
// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
function renderProgressBar(percent, width) {
    const filled = Math.round((percent / 100) * width);
    const empty = width - filled;
    return "█".repeat(filled) + "░".repeat(empty);
}
function formatTokenCount(n) {
    if (n >= 1000) {
        return `${(n / 1000).toFixed(1)}k`;
    }
    return String(n);
}
function renderStatusIndicator(status) {
    switch (status) {
        case "active":
            return "...";
        case "completed":
            return "✔";
        case "failed":
            return "✖";
        case "blocked":
            return "⚠";
        default:
            return "?";
    }
}
