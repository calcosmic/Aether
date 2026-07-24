import { resolve, dirname } from "path";
import { fileURLToPath } from "url";

export const projectRoot = resolve(
  dirname(fileURLToPath(import.meta.url)),
  "../.."
);
