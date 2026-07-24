/**
 * Caste config loader: reads shared YAML ceremony config and provides typed accessors.
 *
 * Loads `.aether/config/ceremony.yaml` relative to the working directory.
 * Falls back to an inline default config (mirroring Go hardcoded values) if the
 * YAML file is missing, ensuring the TS host never breaks during transition.
 */
export interface CasteConfig {
    emoji: string;
    color: string;
    label: string;
}
export interface CeremonyConfig {
    castes: Record<string, CasteConfig>;
    stage_separator: {
        prefix: string;
        suffix: string;
    };
    naming: {
        deterministic_prefixes: Record<string, string[]>;
    };
    banners: Record<string, {
        figlet_font: string;
        text: string;
    }>;
    excavation_phrases: string[];
}
export declare const DEFAULT_CEREMONY_CONFIG: CeremonyConfig;
/**
 * Load ceremony config from `.aether/config/ceremony.yaml`.
 *
 * @param cwd - Working directory (repo root).
 * @returns Parsed and validated `CeremonyConfig`.
 * @throws Error if the YAML is malformed or missing required keys.
 */
export declare function loadCeremonyConfig(cwd: string): CeremonyConfig;
/**
 * Get the full config object for a caste.
 */
export declare function getCasteConfig(config: CeremonyConfig, casteName: string): CasteConfig | undefined;
/**
 * Get a caste's emoji.
 */
export declare function getCasteEmoji(config: CeremonyConfig, casteName: string): string;
/**
 * Get a caste's color (hex).
 */
export declare function getCasteColor(config: CeremonyConfig, casteName: string): string;
/**
 * Get a caste's label.
 */
export declare function getCasteLabel(config: CeremonyConfig, casteName: string): string;
