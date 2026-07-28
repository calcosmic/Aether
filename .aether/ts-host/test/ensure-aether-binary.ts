/**
 * Test preload: guarantee an aether binary for the whole suite.
 *
 * Several suites (go-bridge, event-bridge, platform-dispatcher,
 * prompt-assembler, host entry point) exercise src modules that discover and
 * spawn the aether binary via discoverGoBinary(): AETHER_BINARY_PATH, then
 * PATH, then $HOME/.local/bin/aether. On a developer machine with Aether
 * installed that resolves silently — which is exactly how 28 CI failures hid:
 * the runner has no installed binary, and the tests were depending on machine
 * state they never declared.
 *
 * This preload runs before any test file (registered via --import in the npm
 * test script). If AETHER_BINARY_PATH is not already set, it builds the binary
 * from the repo's own source into a cache and points the variable at it, so
 * the suite always tests the code in this checkout — not whatever happens to
 * be installed.
 *
 * Concurrency: node's test runner spawns one process per test file, and each
 * re-runs this preload. The first existence check keeps that cheap, and the
 * build stages to a pid-suffixed name in the same directory before an atomic
 * rename, so concurrent first-runs cannot corrupt the target (POSIX rename
 * over an existing file is an atomic replace; a same-filesystem rename cannot
 * throw EXDEV).
 */
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, renameSync, rmSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

if (!process.env["AETHER_BINARY_PATH"]) {
  const testDir = dirname(fileURLToPath(import.meta.url));
  const repoRoot = join(testDir, "..", "..", "..");
  const cacheDir = join(testDir, "..", "node_modules", ".cache", "aether-test-binary");
  const binaryName = process.platform === "win32" ? "aether.exe" : "aether";
  const target = join(cacheDir, binaryName);

  if (!existsSync(target)) {
    mkdirSync(cacheDir, { recursive: true });
    const staging = `${target}.${process.pid}`;
    try {
      execFileSync("go", ["build", "-o", staging, "./cmd/aether"], {
        cwd: repoRoot,
        stdio: "pipe",
      });
      renameSync(staging, target);
    } catch (error) {
      rmSync(staging, { force: true });
      // A concurrent test process may have won the race; only fail if the
      // binary genuinely is not there.
      if (!existsSync(target)) {
        throw error;
      }
    }
  }

  process.env["AETHER_BINARY_PATH"] = target;
}
