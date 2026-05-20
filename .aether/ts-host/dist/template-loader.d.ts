/**
 * Template loader: reads editable ceremony templates from disk with YAML frontmatter,
 * falls back to inline defaults when files are missing, and performs variable substitution.
 *
 * Templates live under `.aether/templates/ceremony/` and use `{variable:default}`
 * placeholder syntax in their markdown bodies.
 */
export interface ParsedTemplate {
    frontmatter: Record<string, unknown>;
    body: string;
}
export declare const DEFAULT_TEMPLATES: Record<string, ParsedTemplate>;
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
export declare function parseTemplate(raw: string): ParsedTemplate;
/**
 * Replace `{variable}` and `{variable:default}` placeholders in a template body.
 *
 * @param body - The template body string.
 * @param vars - Map of variable names to values.
 * @returns The body with all placeholders substituted.
 */
export declare function substituteTemplate(body: string, vars: Record<string, string>): string;
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
export declare function loadTemplate(cwd: string, name: string): ParsedTemplate;
