/**
 * JSON passthrough renderer: returns empty strings for all methods.
 *
 * Rationale: Go runtime already handles AETHER_OUTPUT_MODE=json.
 * The narrator should not write visual output in json mode.
 */
export const jsonRenderer = {
    renderBanner() {
        return "";
    },
    renderSpawnFrame() {
        return "";
    },
    renderStageSeparator() {
        return "";
    },
    renderBox() {
        return "";
    },
};
