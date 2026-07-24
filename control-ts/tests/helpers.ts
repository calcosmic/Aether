import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

/**
 * Absolute path to the control-ts project root.
 * Resolves relative to this file's location to avoid cwd-relative pitfalls.
 */
export const projectRoot = resolve(
  dirname(fileURLToPath(import.meta.url)),
  ".."
);

/**
 * Build an absolute path to a test fixture.
 * @param segments Path segments under tests/fixtures/
 * @returns Absolute path string
 */
export function fixturePath(...segments: string[]): string {
  return resolve(projectRoot, "tests", "fixtures", ...segments);
}
