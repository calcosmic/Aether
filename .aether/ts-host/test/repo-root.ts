/**
 * Repo root derived from this file's location (test/ -> ts-host/ -> .aether/
 * -> repo). Shared by every test that needs a real filesystem path into the
 * checkout.
 *
 * History: a dozen test files hardcoded one developer's absolute repo path.
 * Locally everything passed — the machine was part of the fixture — while on
 * CI each file failed only after the failures in front of it were fixed,
 * costing a full CI cycle per file. One derived constant ends the class.
 */
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

export const REPO_ROOT = join(dirname(fileURLToPath(import.meta.url)), "..", "..", "..");
