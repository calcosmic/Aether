/**
 * Ceremony narrator: subscribes to the event bridge and dispatches events
 * to the correct renderer based on output mode and TTY status.
 *
 * Writes rendered output directly to process.stdout in real-time.
 */
import type { CeremonyEvent } from "./types.js";
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
/**
 * Create a ceremony narrator that renders events to stdout.
 *
 * @param opts - Narrator options (cwd and optional output mode override).
 * @returns A narrator with onEvent and stop methods.
 */
export declare function createNarrator(opts: NarratorOptions): Narrator;
