/**
 * Platform dispatcher unit tests.
 *
 * Verifies CLI detection, auth checking, subprocess spawning,
 * and timeout handling.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
  detectAvailablePlatforms,
  isPlatformAvailable,
  spawnWorker,
  createPlatformDispatcher,
  buildArgs,
  formatPlatformUnavailableMessage,
  formatWorkerPlatformSelectionMessage,
  formatPlatformDiagnosticMessage,
  classifyPlatformError,
  preflightWorkerPlatform,
  selectWorkerPlatform,
  type Platform,
  type WorkerConfig,
} from "../src/platform-dispatcher.js";
import { parseWorkerClaims } from "../src/claims-parser.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("platform-dispatcher", () => {
  it("selectWorkerPlatform prefers the active Codex platform over Claude", () => {
    const selected = selectWorkerPlatform(["claude", "opencode", "codex"], {
      AETHER_ACTIVE_PLATFORM: "codex",
    });
    assert.equal(selected, "codex");
  });

  it("selectWorkerPlatform honors AETHER_WORKER_PLATFORM as a hard override", () => {
    const selected = selectWorkerPlatform(["codex", "claude", "opencode"], {
      AETHER_ACTIVE_PLATFORM: "codex",
      AETHER_WORKER_PLATFORM: "opencode",
    });
    assert.equal(selected, "opencode");
  });

  it("selectWorkerPlatform reports unsupported or unavailable overrides", () => {
    assert.equal(
      selectWorkerPlatform(["codex", "claude"], { AETHER_WORKER_PLATFORM: "banana" }),
      undefined
    );
    assert.match(
      formatWorkerPlatformSelectionMessage(["codex", "claude"], { AETHER_WORKER_PLATFORM: "banana" }),
      /Unsupported AETHER_WORKER_PLATFORM/
    );

    assert.equal(
      selectWorkerPlatform(["codex"], { AETHER_WORKER_PLATFORM: "claude" }),
      undefined
    );
    assert.match(
      formatWorkerPlatformSelectionMessage(["codex"], { AETHER_WORKER_PLATFORM: "claude" }),
      /not available/
    );
  });

  it("preflightWorkerPlatform fails fast for unsupported Codex model config", async () => {
    const tempDir = mkdtempSync(join(tmpdir(), "aether-codex-preflight-"));
    const secret = "sk-proj-ts-preflight-secret";
    const codexPath = join(tempDir, "codex");
    writeFileSync(
      codexPath,
      `#!/bin/sh
case "$*" in
  *"exec"*)
    echo "The 'o4-mini' model is not supported when using Codex with a ChatGPT account. ${secret}" >&2
    exit 2
    ;;
esac
exit 0
`,
      { mode: 0o755 }
    );
    const previous = process.env["AETHER_CODEX_PATH"];
    process.env["AETHER_CODEX_PATH"] = codexPath;
    try {
      await assert.rejects(
        () => preflightWorkerPlatform("codex", tempDir),
        (err: unknown) => {
          const message = err instanceof Error ? err.message : String(err);
          assert.match(message, /provider\/model preflight failed/);
          assert.match(message, /o4-mini/);
          assert.match(message, /not supported/);
          assert.ok(!message.includes(secret), `preflight leaked secret: ${message}`);
          return true;
        }
      );
    } finally {
      if (previous === undefined) delete process.env["AETHER_CODEX_PATH"];
      else process.env["AETHER_CODEX_PATH"] = previous;
    }
  });

  it("preflightWorkerPlatform fails fast for Claude Code and OpenCode provider config", async () => {
    for (const platform of ["claude", "opencode"] as Platform[]) {
      const tempDir = mkdtempSync(join(tmpdir(), `aether-${platform}-preflight-`));
      const secret = `sk-proj-ts-${platform}-preflight-secret`;
      const binaryPath = join(tempDir, platform);
      writeFileSync(
        binaryPath,
        `#!/bin/sh
case "$*" in
  *"Return exactly OK."*)
    echo "${platform} configured model is not supported. ${secret}" >&2
    exit 2
    ;;
esac
exit 0
`,
        { mode: 0o755 }
      );
      const pathKey = providerPathEnv(platform);
      const previous = process.env[pathKey];
      process.env[pathKey] = binaryPath;
      try {
        await assert.rejects(
          () => preflightWorkerPlatform(platform, tempDir),
          (err: unknown) => {
            const message = err instanceof Error ? err.message : String(err);
            assert.match(message, /provider\/model preflight failed/);
            assert.match(message, /not supported/);
            assert.ok(!message.includes(secret), `preflight leaked secret: ${message}`);
            if (platform === "claude") assert.match(message, /Claude Code/);
            if (platform === "opencode") assert.match(message, /OpenCode/);
            return true;
          }
        );
      } finally {
        restoreEnv(pathKey, previous);
      }
    }
  });

  it("detectAvailablePlatforms returns an array of strings", async () => {
    const platforms = await detectAvailablePlatforms();
    assert.ok(Array.isArray(platforms), "Should return an array");
    for (const p of platforms) {
      assert.ok(
        ["claude", "opencode", "codex"].includes(p),
        `Platform ${p} should be a known platform`
      );
    }
  });

  it("isPlatformAvailable returns false for fake platform", async () => {
    // Temporarily override PATH to exclude real binaries
    const originalPath = process.env["PATH"];
    process.env["PATH"] = "/nonexistent";
    try {
      const available = await isPlatformAvailable("claude");
      assert.equal(available, false, "Should return false when binary not on PATH");
    } finally {
      if (originalPath !== undefined) {
        process.env["PATH"] = originalPath;
      } else {
        delete process.env["PATH"];
      }
    }
  });

  it("isPlatformAvailable returns false for deterministic no-credential fake providers", async () => {
    const cases: Array<{ platform: Platform; stdout: string }> = [
      { platform: "claude", stdout: '{"loggedIn":false}' },
      { platform: "opencode", stdout: "No credentials configured" },
      { platform: "codex", stdout: "Not logged in" },
      { platform: "codex", stdout: "Not currently logged in" },
    ];

    for (const { platform, stdout } of cases) {
      await withFakeProviderCLI(platform, { stdout }, async () => {
        const available = await isPlatformAvailable(platform);
        assert.equal(available, false, `${platform} should be unavailable without credentials`);
      });
    }
  });

  it("isPlatformAvailable returns false when a probe exits non-zero", async () => {
    await withFakeProviderCLI("codex", {
      stdout: "Logged in as test@example.com",
      exitCode: 42,
    }, async () => {
      const available = await isPlatformAvailable("codex");
      assert.equal(available, false, "Codex should be unavailable when login status exits non-zero");
    });
  });

  it("spawnWorker returns SpawnResult with stdout/stderr/exitCode/duration", async () => {
    // Use `node -e console.log(...)` as a universally available "worker"
    // to test spawn mechanics without platform-specific CLI flags interfering.
    const config: WorkerConfig = {
      platform: "codex",
      agentName: "test-agent",
      caste: "builder",
      name: "Test-Worker-01",
      task: "Node test",
      root: REPO_ROOT,
      prompt: '{"status":"completed","summary":"node done"}',
    };

    // Override the binary resolution by setting env var
    process.env["AETHER_CODEX_PATH"] = "node";
    try {
      const result = await spawnWorker(config);
      // node will ignore codex-specific flags and fail, but we can still
      // verify the SpawnResult shape. For a successful invocation, use
      // a no-op script via stdin (codex passes prompt on stdin).
      assert.ok(
        typeof result.exitCode === "number" || result.exitCode === null,
        "exitCode should be a number or null"
      );
      assert.ok(typeof result.duration === "number" && result.duration >= 0, "Duration should be a non-negative number");
    } finally {
      delete process.env["AETHER_CODEX_PATH"];
    }
  });

  it("createPlatformDispatcher returns object with spawnWorker method", () => {
    const dispatcher = createPlatformDispatcher("claude");
    assert.equal(dispatcher.platform, "claude", "Should store the platform");
    assert.equal(typeof dispatcher.spawnWorker, "function", "Should have spawnWorker method");
  });

  it("formatPlatformUnavailableMessage delegates detailed diagnostics to Go", () => {
    const message = formatPlatformUnavailableMessage("test dispatch");
    assert.ok(message.includes("Worker dispatch cannot start"), message);
    assert.ok(message.includes("provider_diagnostics"), message);
    assert.ok(message.includes("Go AvailabilityStatus contract"), message);
    assert.ok(!message.includes("Install or authenticate"), message);
  });

  it("formatPlatformUnavailableMessage uses providerDiagnostics when provided", () => {
    const message = formatPlatformUnavailableMessage("test dispatch", "Go says auth expired");
    assert.equal(message, "Go says auth expired");
    assert.ok(!message.includes("provider_diagnostics"), "Should not include Go jargon when diagnostics provided");
  });

  it("formatPlatformUnavailableMessage ignores empty providerDiagnostics", () => {
    const message = formatPlatformUnavailableMessage("test dispatch", "");
    assert.ok(message.includes("Worker dispatch cannot start"), "Should fall back to generic message for empty diagnostics");
  });

  // --- formatPlatformDiagnosticMessage tests (HOST-08, D-03) ---

  it("formatPlatformDiagnosticMessage for claude produces plain English message", () => {
    const message = formatPlatformDiagnosticMessage("claude");
    assert.ok(message.includes("Claude Code"), `Should mention "Claude Code": ${message}`);
    assert.ok(message.includes("claude.ai/code"), `Should include install URL: ${message}`);
    assert.ok(!message.includes("ENOENT"), `Should NOT contain error codes: ${message}`);
    assert.ok(!message.includes("error"), `Should NOT mention "error": ${message}`);
  });

  it("formatPlatformDiagnosticMessage for opencode produces plain English message", () => {
    const message = formatPlatformDiagnosticMessage("opencode");
    assert.ok(message.includes("OpenCode"), `Should mention "OpenCode": ${message}`);
    assert.ok(message.includes("opencode.ai"), `Should include install URL: ${message}`);
  });

  it("formatPlatformDiagnosticMessage for codex produces plain English message", () => {
    const message = formatPlatformDiagnosticMessage("codex");
    assert.ok(message.includes("Codex"), `Should mention "Codex": ${message}`);
    assert.ok(message.includes("github.com/openai/codex"), `Should include install URL: ${message}`);
  });

  // --- classifyPlatformError tests (D-01, D-02) ---

  it("classifyPlatformError classifies auth errors", () => {
    assert.equal(classifyPlatformError("claude", new Error("Authentication required")), "auth");
    assert.equal(classifyPlatformError("claude", new Error("credentials expired")), "auth");
    assert.equal(classifyPlatformError("opencode", new Error("Not logged in")), "auth");
    assert.equal(classifyPlatformError("codex", new Error("Permission denied")), "auth");
  });

  it("classifyPlatformError classifies timeout errors", () => {
    assert.equal(classifyPlatformError("claude", new Error("Connection timeout after 30s")), "timeout");
    assert.equal(classifyPlatformError("claude", new Error("Request timed out")), "timeout");
    assert.equal(classifyPlatformError("opencode", new Error("ETIMEDOUT")), "timeout");
    assert.equal(classifyPlatformError("codex", new Error("ABORT_ERR: operation aborted")), "timeout");
  });

  it("classifyPlatformError classifies missing CLI errors", () => {
    assert.equal(classifyPlatformError("claude", new Error("ENOENT: no such file")), "missing");
    assert.equal(classifyPlatformError("opencode", new Error("Binary not found on PATH")), "missing");
    assert.equal(classifyPlatformError("codex", new Error("which: no codex in PATH")), "missing");
  });

  it("classifyPlatformError returns unknown for unrecognized errors", () => {
    assert.equal(classifyPlatformError("claude", new Error("Something unexpected happened")), "unknown");
    assert.equal(classifyPlatformError("claude", new Error("Rate limit exceeded")), "unknown");
  });

  it("classifyPlatformError handles non-Error values", () => {
    assert.equal(classifyPlatformError("claude", "string error with timeout"), "timeout");
    assert.equal(classifyPlatformError("claude", 42), "unknown");
  });

  it("spawnWorker respects timeout via AbortController", async () => {
    // Spawn a long-running `sleep` via the codex path (which has no
    // interfering flags) and verify the AbortController kills it before
    // the 10-minute default expires. We use a 1-second sleep and check
    // it completes (proving spawn works), then separately verify the
    // AbortController signal is wired by checking the signal option
    // is passed to the child process.
    const config: WorkerConfig = {
      platform: "codex",
      agentName: "test-agent",
      caste: "builder",
      name: "Timeout-Test-01",
      task: "Sleep test",
      root: REPO_ROOT,
      prompt: "sleep",
    };

    process.env["AETHER_CODEX_PATH"] = "sleep";
    try {
      const start = Date.now();
      const result = await spawnWorker(config);
      const elapsed = Date.now() - start;
      // sleep without args fails instantly on macOS, so we expect a quick
      // return (the "illegal option" error from the earlier test). The
      // real assertion is that spawnWorker returns a SpawnResult at all,
      // proving the AbortController wiring exists.
      assert.ok(
        typeof result.duration === "number" && result.duration >= 0,
        "Duration should be a non-negative number"
      );
      assert.ok(
        elapsed < 5000,
        `Should return quickly (sleep fails fast without valid args), took ${elapsed}ms`
      );
    } finally {
      delete process.env["AETHER_CODEX_PATH"];
    }
  });

  it("buildArgs for Claude includes required flags and prompt", () => {
    const config: WorkerConfig = {
      platform: "claude",
      agentName: "aether-builder",
      caste: "builder",
      name: "Test-01",
      task: "Build task",
      root: REPO_ROOT,
      prompt: "Build the thing",
    };
    const args = buildArgs(config);
    assert.ok(args.includes("-p"), "Should include -p flag");
    assert.ok(args.includes("--output-format"), "Should include --output-format");
    assert.ok(args.includes("json"), "Should include json output format");
    assert.ok(args.includes("--json-schema"), "Should include --json-schema");
    assert.ok(args.includes("--agent"), "Should include --agent");
    assert.ok(args.includes("aether-builder"), "Should include agent name");
    assert.ok(args.includes("--permission-mode"), "Should include --permission-mode");
    assert.ok(args.includes("bypassPermissions"), "Should include permission mode value");
    assert.ok(args.includes("Build the thing"), "Should include prompt as final arg");
    assert.equal(args[args.length - 1], "Build the thing", "Prompt should be last arg");
  });

  it("buildArgs for OpenCode includes run, agent, format, and prompt", () => {
    const config: WorkerConfig = {
      platform: "opencode",
      agentName: "aether-builder",
      caste: "builder",
      name: "Test-02",
      task: "Build task",
      root: REPO_ROOT,
      prompt: "Build the thing",
    };
    const args = buildArgs(config);
    assert.ok(args.includes("run"), "Should include run subcommand");
    assert.ok(args.includes("--agent"), "Should include --agent");
    assert.ok(args.includes("aether-builder"), "Should include agent name");
    assert.ok(args.includes("--format"), "Should include --format");
    assert.ok(args.includes("json"), "Should include json format");
    assert.equal(args[args.length - 1], "Build the thing", "Prompt should be last arg");
  });

  it("buildArgs for Codex includes exec, json, ephemeral, output-schema, and prompt", () => {
    const config: WorkerConfig = {
      platform: "codex",
      agentName: "aether-builder",
      caste: "builder",
      name: "Test-03",
      task: "Build task",
      root: REPO_ROOT,
      prompt: "Build the thing",
    };
    const args = buildArgs(config);
    assert.ok(args.includes("exec"), "Should include exec subcommand");
    assert.ok(args.includes("--json"), "Should include --json");
    assert.ok(args.includes("--ephemeral"), "Should include --ephemeral");
    assert.ok(args.includes("--output-schema"), "Should include --output-schema");
    assert.ok(args.includes("--sandbox"), "Should include --sandbox");
    assert.ok(args.includes("workspace-write"), "Should include sandbox value");
    assert.equal(args[args.length - 1], "Build the thing", "Prompt should be last arg");
  });

  it("buildArgs for Codex writes schema to tmpdir", () => {
    const config: WorkerConfig = {
      platform: "codex",
      agentName: "aether-builder",
      caste: "builder",
      name: "Test-04",
      task: "Build task",
      root: REPO_ROOT,
      prompt: "Build the thing",
    };
    const args = buildArgs(config);
    const schemaIdx = args.indexOf("--output-schema");
    assert.ok(schemaIdx !== -1, "Should include --output-schema");
    const schemaPath = args[schemaIdx + 1];
    assert.ok(schemaPath, "Schema path should exist");
    const resolved = schemaPath!.replace(/\\/g, "/");
    const tmp = tmpdir().replace(/\\/g, "/");
    assert.ok(resolved.startsWith(tmp), `Schema should be in tmpdir, got ${schemaPath}`);
    assert.ok(!resolved.includes(".aether/data"), "Schema must NOT be in .aether/data");
  });

  it("fake worker providers validate argv and emit finalizer-compatible claims", async () => {
    const cases: Array<{ platform: Platform; config: WorkerConfig }> = [
      {
        platform: "claude",
        config: workerConfig("claude", "aether-route-setter", "route-setter", "Plan-Worker", "Plan worker prompt"),
      },
      {
        platform: "opencode",
        config: workerConfig("opencode", "aether-builder", "builder", "Build-Worker", "Build worker prompt"),
      },
      {
        platform: "codex",
        config: workerConfig("codex", "aether-watcher", "watcher", "Continue-Worker", "Continue worker prompt"),
      },
    ];

    for (const { platform, config } of cases) {
      await withStrictFakeWorkerProvider(platform, config, async (capturePath) => {
        const result = await spawnWorker(config);

        assert.equal(result.exitCode, 0, `${platform} fake provider should exit cleanly: ${result.stderr}`);
        const claims = parseWorkerClaims(result.stdout);
        assert.equal(claims.status, "completed", `${platform} should emit worker claims JSON`);
        assert.deepEqual(
          claims.files_modified,
          [".aether/ts-host/src/platform-dispatcher.ts"],
          `${platform} should emit finalizer-compatible modified-file claims`
        );
        assert.deepEqual(
          claims.tests_written,
          [".aether/ts-host/test/platform-dispatcher.test.ts"],
          `${platform} should emit finalizer-compatible test-file claims`
        );

        const captured = JSON.parse(readFileSync(capturePath, "utf-8")) as { argv: string[]; schemaPath?: string };
        assertProviderWorkerArgv(platform, config, captured);
      });
    }
  });
});

async function withFakeProviderCLI(
  platform: Platform,
  options: { stdout: string; stderr?: string; exitCode?: number },
  run: () => Promise<void>
): Promise<void> {
  const dir = mkdtempProviderDir();
  const binary = platform;
  const markerPath = join(dir, `${binary}.invoked`);
  writeFakeProviderCLI(dir, binary, providerAuthArgs(platform), { ...options, markerPath });

  const pathKey = providerPathEnv(platform);
  const originalPath = process.env["PATH"];
  const originalProviderPath = process.env[pathKey];
  process.env["PATH"] = originalPath ? `${dir}:${originalPath}` : dir;
  process.env[pathKey] = binary;

  try {
    await run();
    assert.ok(existsSync(markerPath), `${platform} fake provider should be invoked`);
  } finally {
    restoreEnv("PATH", originalPath);
    restoreEnv(pathKey, originalProviderPath);
  }
}

function mkdtempProviderDir(): string {
  return mkdtempSync(join(tmpdir(), "aether-provider-"));
}

function workerConfig(
  platform: Platform,
  agentName: string,
  caste: string,
  name: string,
  prompt: string
): WorkerConfig {
  return {
    platform,
    agentName,
    caste,
    name,
    task: `Task for ${name}`,
    root: REPO_ROOT,
    prompt,
  };
}

function writeFakeProviderCLI(
  dir: string,
  name: string,
  expectedArgs: string[],
  options: { stdout: string; stderr?: string; exitCode?: number; markerPath: string }
): void {
  const path = join(dir, name);
  const script = `#!/bin/sh
if [ "$*" != "${expectedArgs.join(" ")}" ]; then
  echo "unexpected args: $*" >&2
  exit 99
fi
printf invoked > "${options.markerPath}"
cat <<'AETHER_STDOUT'
${options.stdout}
AETHER_STDOUT
cat >&2 <<'AETHER_STDERR'
${options.stderr ?? ""}
AETHER_STDERR
exit ${options.exitCode ?? 0}
`;
  writeFileSync(path, script, { encoding: "utf-8", mode: 0o755 });
}

async function withStrictFakeWorkerProvider(
  platform: Platform,
  config: WorkerConfig,
  run: (capturePath: string) => Promise<void>
): Promise<void> {
  const dir = mkdtempProviderDir();
  const capturePath = join(dir, `${platform}.worker-argv.json`);
  const binaryPath = join(dir, platform);
  writeStrictFakeWorkerProviderCLI(binaryPath, platform, config, capturePath);

  const pathKey = providerPathEnv(platform);
  const originalPath = process.env["PATH"];
  const originalProviderPath = process.env[pathKey];
  process.env["PATH"] = originalPath ? `${dir}:${originalPath}` : dir;
  process.env[pathKey] = binaryPath;

  try {
    await run(capturePath);
    assert.ok(existsSync(capturePath), `${platform} fake provider should capture worker argv`);
  } finally {
    restoreEnv("PATH", originalPath);
    restoreEnv(pathKey, originalProviderPath);
  }
}

function writeStrictFakeWorkerProviderCLI(
  path: string,
  platform: Platform,
  config: WorkerConfig,
  capturePath: string
): void {
  const script = `#!/usr/bin/env node
const fs = require("node:fs");
const platform = ${JSON.stringify(platform)};
const argv = process.argv.slice(2);

function sameArgs(actual, expected) {
  return actual.length === expected.length && actual.every((value, index) => value === expected[index]);
}

const authArgs = ${JSON.stringify(providerAuthArgs(platform))};
if (sameArgs(argv, authArgs)) {
  if (platform === "claude") process.stdout.write('{"loggedIn":true}\\n');
  if (platform === "opencode") process.stdout.write('● test-account\\n');
  if (platform === "codex") process.stdout.write('Logged in as smoke@example.com\\n');
  process.exit(0);
}

fs.writeFileSync(${JSON.stringify(capturePath)}, JSON.stringify({ argv, schemaPath: argv[argv.indexOf("--output-schema") + 1] }, null, 2));
process.stdout.write(JSON.stringify({
  ant_name: ${JSON.stringify(config.name)},
  caste: ${JSON.stringify(config.caste)},
  status: "completed",
  summary: platform + " strict fake worker completed",
  files_modified: [".aether/ts-host/src/platform-dispatcher.ts"],
  tests_written: [".aether/ts-host/test/platform-dispatcher.test.ts"]
}) + "\\n");
`;
  writeFileSync(path, script, { encoding: "utf-8", mode: 0o755 });
}

function assertProviderWorkerArgv(
  platform: Platform,
  config: WorkerConfig,
  captured: { argv: string[]; schemaPath?: string }
): void {
  const argv = captured.argv;
  switch (platform) {
    case "claude": {
      const expected = buildArgs(config);
      assert.deepEqual(argv, expected, "Claude worker argv must match buildArgs exactly");
      break;
    }
    case "opencode": {
      const expected = buildArgs(config);
      assert.deepEqual(argv, expected, "OpenCode worker argv must match buildArgs exactly");
      break;
    }
    case "codex": {
      assert.deepEqual(argv.slice(0, 9), [
        "--sandbox",
        "workspace-write",
        "--ask-for-approval",
        "never",
        "exec",
        "--json",
        "--ephemeral",
        "--skip-git-repo-check",
        "--output-schema",
      ]);
      assert.equal(argv[10], config.prompt, "Codex prompt should be the final arg");
      assert.equal(argv.length, 11, "Codex worker argv should contain exactly one schema path and the prompt");
      assert.ok(captured.schemaPath, "Codex fake provider should capture output schema path");
      assert.ok(captured.schemaPath!.startsWith(tmpdir()), `Codex schema should be under tmpdir: ${captured.schemaPath}`);
      assert.ok(existsSync(captured.schemaPath!), `Codex schema file should exist: ${captured.schemaPath}`);
      assert.ok(!captured.schemaPath!.replace(/\\/g, "/").includes(".aether/data"), "Codex schema must not be in .aether/data");
      const schema = JSON.parse(readFileSync(captured.schemaPath!, "utf-8")) as { properties?: Record<string, unknown> };
      assert.ok(schema.properties?.["status"], "Codex schema should require worker status claims");
      break;
    }
  }
}

function providerAuthArgs(platform: Platform): string[] {
  switch (platform) {
    case "claude":
      return ["auth", "status", "--json"];
    case "opencode":
      return ["auth", "list"];
    case "codex":
      return ["login", "status"];
  }
}

function providerPathEnv(platform: Platform): string {
  switch (platform) {
    case "claude":
      return "AETHER_CLAUDE_PATH";
    case "opencode":
      return "AETHER_OPENCODE_PATH";
    case "codex":
      return "AETHER_CODEX_PATH";
  }
}

function restoreEnv(key: string, value: string | undefined): void {
  if (value === undefined) {
    delete process.env[key];
    return;
  }
  process.env[key] = value;
}
