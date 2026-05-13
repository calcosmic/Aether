/**
 * Template loader: reads editable ceremony templates from disk with YAML frontmatter,
 * falls back to inline defaults when files are missing, and performs variable substitution.
 *
 * Templates live under `.aether/templates/ceremony/` and use `{variable:default}`
 * placeholder syntax in their markdown bodies.
 */

import { readFileSync } from "node:fs";
import { join } from "node:path";
import yaml from "js-yaml";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ParsedTemplate {
  frontmatter: Record<string, unknown>;
  body: string;
}

// ---------------------------------------------------------------------------
// Inline defaults (fallback when disk files are missing)
// ---------------------------------------------------------------------------

export const DEFAULT_TEMPLATES: Record<string, ParsedTemplate> = {
  "banner-build-start": {
    frontmatter: {
      figlet_font: "Standard",
      emoji: "🔨",
      title: "BUILD",
    },
    body: "\n{figlet_banner}\n\n── {stage} ──\n\n{content}\n",
  },
  "banner-seal-complete": {
    frontmatter: {
      figlet_font: "Standard",
      emoji: "👑",
      title: "CROWNED ANTHILL",
    },
    body: "\n{figlet_banner}\n\n── {stage} ──\n\n{content}\n",
  },
  "spawn-frame": {
    frontmatter: {},
    body: "{emoji} {label} {name}  {task}\n",
  },
  "stage-separator": {
    frontmatter: {},
    body: "{prefix}{stage}{suffix}\n",
  },
  "build-summary": {
    frontmatter: {
      borderStyle: "round",
      borderColor: "green",
    },
    body: "\n{content}\n",
  },
  "closeout-ritual": {
    frontmatter: {
      borderStyle: "double",
      borderColor: "cyan",
    },
    body: "\n{content}\n",
  },
};

// ---------------------------------------------------------------------------
// Parsing
// ---------------------------------------------------------------------------

/**
 * Split a raw template string into YAML frontmatter and markdown body.
 *
 * Frontmatter is delimited by `---\n` at the start and `\n---\n` after the YAML block.
 * Everything after the second `---` is the body.
 *
 * @param raw - The raw template file contents.
 * @returns Parsed frontmatter object and body string.
 * @throws Error if the frontmatter delimiters are missing.
 */
export function parseTemplate(raw: string): ParsedTemplate {
  const match = raw.match(/^---\r?\n([\s\S]*?)^---\r?\n/m);
  if (!match) {
    throw new Error("Invalid template: missing YAML frontmatter");
  }

  const frontmatterRaw = match[1] ?? "";
  const body = raw.slice(match[0].length);

  const frontmatter = yaml.load(frontmatterRaw) as Record<string, unknown>;

  return {
    frontmatter: frontmatter ?? {},
    body,
  };
}

// ---------------------------------------------------------------------------
// Substitution
// ---------------------------------------------------------------------------

/**
 * Replace `{variable}` and `{variable:default}` placeholders in a template body.
 *
 * @param body - The template body string.
 * @param vars - Map of variable names to values.
 * @returns The body with all placeholders substituted.
 */
export function substituteTemplate(
  body: string,
  vars: Record<string, string>
): string {
  return body.replace(/{(\w+)(?::([^}]*))?}/g, (_match, key, fallback) => {
    return vars[key] ?? fallback ?? "";
  });
}

// ---------------------------------------------------------------------------
// Loading
// ---------------------------------------------------------------------------

/**
 * Load a named ceremony template from disk, falling back to inline defaults.
 *
 * Resolves `{cwd}/.aether/templates/ceremony/{name}.md`. If the file is missing,
 * looks up `name` in `DEFAULT_TEMPLATES`. If neither exists, throws.
 *
 * @param cwd - Repository root (working directory).
 * @param name - Template name (without `.md` extension).
 * @returns Parsed template with frontmatter and body.
 * @throws Error if the template is not found on disk or in defaults.
 */
export function loadTemplate(cwd: string, name: string): ParsedTemplate {
  const filePath = join(cwd, ".aether", "templates", "ceremony", `${name}.md`);

  let raw: string;
  try {
    raw = readFileSync(filePath, "utf-8");
  } catch {
    const fallback = DEFAULT_TEMPLATES[name];
    if (fallback) {
      return fallback;
    }
    throw new Error(`Template not found: ${name}`);
  }

  return parseTemplate(raw);
}
