/**
 * Caste config loader: reads shared YAML ceremony config and provides typed accessors.
 *
 * Loads `.aether/config/ceremony.yaml` relative to the working directory.
 * Falls back to an inline default config (mirroring Go hardcoded values) if the
 * YAML file is missing, ensuring the TS host never breaks during transition.
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";
import yaml from "js-yaml";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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
  banners: Record<string, { figlet_font: string; text: string }>;
  excavation_phrases: string[];
}

// ---------------------------------------------------------------------------
// Inline defaults (mirror Go cmd/codex_visuals.go)
// ---------------------------------------------------------------------------

export const DEFAULT_CEREMONY_CONFIG: CeremonyConfig = {
  castes: {
    queen: { emoji: "👑", color: "#FF00FF", label: "Queen" },
    builder: { emoji: "🔨", color: "#FFD700", label: "Builder" },
    watcher: { emoji: "👁️", color: "#00FFFF", label: "Watcher" },
    scout: { emoji: "🔍", color: "#00FF00", label: "Scout" },
    colonizer: { emoji: "🗺️", color: "#0000FF", label: "Colonizer" },
    surveyor: { emoji: "📊", color: "#0000FF", label: "Surveyor" },
    architect: { emoji: "🏛️", color: "#FF00FF", label: "Architect" },
    chaos: { emoji: "🎲", color: "#FF0000", label: "Chaos" },
    archaeologist: { emoji: "🏺", color: "#FFFF00", label: "Archaeologist" },
    oracle: { emoji: "🔮", color: "#FF00FF", label: "Oracle" },
    route_setter: { emoji: "📋", color: "#00FFFF", label: "Route-Setter" },
    ambassador: { emoji: "🔌", color: "#00FFFF", label: "Ambassador" },
    auditor: { emoji: "👥", color: "#FFFFFF", label: "Auditor" },
    chronicler: { emoji: "📝", color: "#00FF00", label: "Chronicler" },
    gatekeeper: { emoji: "⚔️", color: "#FF0000", label: "Gatekeeper" },
    guardian: { emoji: "🛡️", color: "#00FFFF", label: "Guardian" },
    includer: { emoji: "♿", color: "#00FFFF", label: "Includer" },
    keeper: { emoji: "📚", color: "#00FF00", label: "Keeper" },
    measurer: { emoji: "⚡", color: "#FFFF00", label: "Measurer" },
    probe: { emoji: "🧪", color: "#00FFFF", label: "Probe" },
    tracker: { emoji: "🐛", color: "#FF0000", label: "Tracker" },
    weaver: { emoji: "🔄", color: "#FF00FF", label: "Weaver" },
    dreamer: { emoji: "💭", color: "#808080", label: "Dreamer" },
    medic: { emoji: "🩹", color: "#00FFFF", label: "Medic" },
    fixer: { emoji: "🔧", color: "#FFD700", label: "Fixer" },
    porter: { emoji: "📦", color: "#00FFFF", label: "Porter" },
    sage: { emoji: "🦉", color: "#808080", label: "Sage" },
  },
  stage_separator: { prefix: "── ", suffix: " ──" },
  naming: {
    deterministic_prefixes: {
      builder: ["Mason", "Carpenter", "Smith", "Stone", "Forge"],
      watcher: ["Hawk", "Owl", "Sentinel", "Vigil", "Sentry"],
      scout: ["Ranger", "Pathfinder", "Trail", "Seeker", "Guide"],
      queen: ["Sovereign", "Monarch", "Regent", "Crown", "Matriarch"],
      architect: ["Forge", "Spire", "Pillar", "Vault", "Keystone"],
      chaos: ["Storm", "Tempest", "Rift", "Void", "Abyss"],
      archaeologist: ["Dust", "Relic", "Shard", "Ruin", "Stratum"],
      oracle: ["Prophet", "Seer", "Vision", "Augur", "Harbinger"],
      route_setter: ["Compass", "Chart", "Meridian", "Vector", "Azimuth"],
      ambassador: ["Envoy", "Diplomat", "Legate", "Courier", "Emissary"],
      auditor: ["Lens", "Scale", "Gauge", "Measure", "Calipers"],
      chronicler: ["Quill", "Ledger", "Archive", "Scroll", "Codex"],
      gatekeeper: ["Bastion", "Bulwark", "Rampart", "Palisade", "Redoubt"],
      guardian: ["Shield", "Aegis", "Ward", "Sanctum", "Haven"],
      includer: ["Bridge", "Span", "Arch", "Gate", "Portal"],
      keeper: ["Vault", "Tome", "Reliquary", "Cache", "Hoard"],
      measurer: ["Chronometer", "Metric", "Datum", "Index", "Benchmark"],
      probe: ["Sounding", "Drill", "Core", "Specimen", "Assay"],
      tracker: ["Spoor", "Trace", "Trail", "Scent", "Imprint"],
      weaver: ["Loom", "Tapestry", "Thread", "Weft", "Pattern"],
      dreamer: ["Reverie", "Muse", "Visions", "Nebula", "Eidolon"],
      medic: ["Salve", "Tincture", "Elixir", "Balm", "Remedy"],
      fixer: ["Mend", "Patch", "Weld", "Splice", "Restore"],
      porter: ["Barge", "Haul", "Tote", "Bearer", "Convey"],
      sage: ["Wisdom", "Axiom", "Tenet", "Precept", "Doctrine"],
      colonizer: ["Pioneer", "Frontier", "Outpost", "Settlement", "Claim"],
      surveyor: ["Transit", "Theodolite", "Plumb", "Datum", "Benchmark"],
    },
  },
  banners: {
    build_start: { figlet_font: "Standard", text: "BUILD" },
    seal_complete: { figlet_font: "Standard", text: "CROWNED ANTHILL" },
  },
  excavation_phrases: [
    "Excavating chamber...",
    "Laying foundation...",
    "Expanding tunnel...",
    "Carrying substrate...",
    "Sealing passage...",
    "Polishing surface...",
    "Fortifying walls...",
  ],
};

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

function isValidCasteConfig(obj: unknown): obj is CasteConfig {
  if (typeof obj !== "object" || obj === null) return false;
  const c = obj as Record<string, unknown>;
  return (
    typeof c.emoji === "string" &&
    typeof c.color === "string" &&
    typeof c.label === "string"
  );
}

function assertCeremonyConfig(obj: unknown): asserts obj is CeremonyConfig {
  if (typeof obj !== "object" || obj === null) {
    throw new Error("Invalid ceremony.yaml: root is not an object");
  }
  const doc = obj as Record<string, unknown>;

  if (
    typeof doc.castes !== "object" ||
    doc.castes === null ||
    Array.isArray(doc.castes)
  ) {
    throw new Error("Invalid ceremony.yaml: missing required key castes");
  }

  if (
    typeof doc.stage_separator !== "object" ||
    doc.stage_separator === null
  ) {
    throw new Error("Invalid ceremony.yaml: missing required key stage_separator");
  }
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Load ceremony config from `.aether/config/ceremony.yaml`.
 *
 * @param cwd - Working directory (repo root).
 * @returns Parsed and validated `CeremonyConfig`.
 * @throws Error if the YAML is malformed or missing required keys.
 */
export function loadCeremonyConfig(cwd: string): CeremonyConfig {
  const configPath = join(cwd, ".aether", "config", "ceremony.yaml");
  let raw: string;
  try {
    raw = readFileSync(configPath, "utf-8");
  } catch {
    // File missing — fall back to hardcoded defaults
    return DEFAULT_CEREMONY_CONFIG;
  }

  const parsed = yaml.load(raw);
  assertCeremonyConfig(parsed);

  // Validate that every caste entry has required fields
  const castes = parsed.castes as Record<string, unknown>;
  for (const [name, cfg] of Object.entries(castes)) {
    if (!isValidCasteConfig(cfg)) {
      throw new Error(
        `Invalid ceremony.yaml: caste "${name}" missing emoji, color, or label`
      );
    }
  }

  return parsed;
}

/**
 * Get the full config object for a caste.
 */
export function getCasteConfig(
  config: CeremonyConfig,
  casteName: string
): CasteConfig | undefined {
  return config.castes[casteName];
}

/**
 * Get a caste's emoji.
 */
export function getCasteEmoji(config: CeremonyConfig, casteName: string): string {
  return config.castes[casteName]?.emoji ?? "❓";
}

/**
 * Get a caste's color (hex).
 */
export function getCasteColor(config: CeremonyConfig, casteName: string): string {
  return config.castes[casteName]?.color ?? "#FFFFFF";
}

/**
 * Get a caste's label.
 */
export function getCasteLabel(config: CeremonyConfig, casteName: string): string {
  return config.castes[casteName]?.label ?? casteName.charAt(0).toUpperCase() + casteName.slice(1);
}
