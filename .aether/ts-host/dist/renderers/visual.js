/**
 * ANSI visual renderer: produces colored terminal output with figlet banners,
 * caste identity frames, stage separators, and boxen-framed boxes.
 *
 * All methods return strings (do NOT write to stdout).
 */
import chalk from "chalk";
import figlet from "figlet";
import boxen from "boxen";
import { getCasteEmoji, getCasteColor, getCasteLabel, } from "../caste-config.js";
export const visualRenderer = {
    renderBanner(title, font) {
        const banner = figlet.textSync(title, { font: font ?? "Standard" });
        return banner
            .split("\n")
            .map((line) => chalk.cyan(line))
            .join("\n");
    },
    renderSpawnFrame(payload, config) {
        const emoji = getCasteEmoji(config, payload.caste ?? "");
        const label = getCasteLabel(config, payload.caste ?? "");
        const color = getCasteColor(config, payload.caste ?? "");
        return `${emoji} ${chalk.hex(color)(label)} ${payload.name ?? ""}  ${payload.task ?? ""}\n`;
    },
    renderStageSeparator(stage, config) {
        const prefix = config.stage_separator.prefix;
        const suffix = config.stage_separator.suffix;
        return `${prefix}${stage}${suffix}\n`;
    },
    renderBox(content, options) {
        return boxen(content, {
            padding: 1,
            margin: 1,
            borderStyle: (options?.borderStyle ?? "round"),
            borderColor: options?.borderColor ?? "green",
        });
    },
};
