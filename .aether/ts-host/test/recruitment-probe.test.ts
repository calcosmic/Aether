/**
 * Recruitment probe tests (BIO-03, 203-04).
 *
 * Proves the TypeScript-side native-nesting probe (recruitment-probe.ts) and
 * the retired-but-regression-tested cheap auth probe in
 * platform-dispatcher.ts (runProbe, reached via isPlatformAvailable) both
 * honour a configurable, bounded, process-tree-safe budget -- the exact
 * counter-example the folded todo
 * (.planning/todos/pending/2026-08-01-ts-host-preflight-hardcoded-timeout.md)
 * names as still open.
 */

import { afterEach, describe, it } from "node:test";
import assert from "node:assert/strict";
import { existsSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import {
  RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS,
  resolveRecruitmentProbeTimeoutMs,
  probeNativeNesting,
  __resetRecruitmentProbeTimeoutWarnings,
  __resetRecruitmentProbeCache,
  __setRecruitmentProbeLauncher,
  __restoreRecruitmentProbeLauncher,
} from "../src/recruitment-probe.js";
import { isPlatformAvailable } from "../src/platform-dispatcher.js";
import { resolvePreflightTimeoutMs } from "../src/preflight-config.js";

function restoreEnv(key: string, value: string | undefined): void {
  if (value === undefined) {
    delete process.env[key];
    return;
  }
  process.env[key] = value;
}

function mkdtempProviderDir(): string {
  return mkdtempSync(join(tmpdir(), "aether-recruit-probe-fixture-"));
}

describe("recruitment-probe.ts (native-nesting probe)", () => {
  afterEach(() => {
    __resetRecruitmentProbeCache();
    __resetRecruitmentProbeTimeoutWarnings();
    __restoreRecruitmentProbeLauncher();
    delete process.env["AETHER_RECRUIT_PROBE_TIMEOUT"];
    delete process.env["AETHER_RECRUIT_BINARY"];
    delete process.env["AETHER_RECRUIT_PROBE_ARGS"];
  });

  it("resolveRecruitmentProbeTimeoutMs accepts a valid Go duration", () => {
    process.env["AETHER_RECRUIT_PROBE_TIMEOUT"] = "2500ms";
    assert.equal(resolveRecruitmentProbeTimeoutMs(), 2500);
  });

  it("resolveRecruitmentProbeTimeoutMs falls back and warns once per bad value", () => {
    const originalWrite = process.stderr.write.bind(process.stderr);
    let captured = "";
    process.stderr.write = ((chunk: string | Uint8Array): boolean => {
      captured += typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf-8");
      return true;
    }) as typeof process.stderr.write;
    try {
      process.env["AETHER_RECRUIT_PROBE_TIMEOUT"] = "not-a-duration";
      assert.equal(resolveRecruitmentProbeTimeoutMs(), RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS);
      assert.match(captured, /Warning: AETHER_RECRUIT_PROBE_TIMEOUT="not-a-duration"/);

      const firstLength = captured.length;
      assert.equal(resolveRecruitmentProbeTimeoutMs(), RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS);
      assert.equal(captured.length, firstLength, "warning must be deduped per value");
    } finally {
      process.stderr.write = originalWrite;
    }
  });

  it("resolveRecruitmentProbeTimeoutMs defaults to 10 seconds when unset", () => {
    delete process.env["AETHER_RECRUIT_PROBE_TIMEOUT"];
    assert.equal(resolveRecruitmentProbeTimeoutMs(), 10_000);
    assert.equal(RECRUITMENT_PROBE_DEFAULT_TIMEOUT_MS, 10_000);
  });

  it("probeNativeNesting runs in a fresh OS temp directory, never the repo root", async () => {
    const repoRoot = process.cwd();
    let recordedCwd = "";
    __setRecruitmentProbeLauncher(async (_binary, _args, cwd) => {
      recordedCwd = cwd;
      return { output: "NATIVE_NESTING_AVAILABLE=false", timedOut: false, exitCode: 0 };
    });

    const result = await probeNativeNesting();

    assert.ok(recordedCwd.length > 0, "launcher was not invoked");
    assert.notEqual(recordedCwd, repoRoot, "probe must not run in the repository root");
    assert.ok(!existsSync(recordedCwd), "probe directory should be removed once the probe returns");
    assert.equal(result.supported, false);
    assert.equal(result.verdict, "unavailable");
  });

  it("probeNativeNesting reports availability from the launcher's own answer, not a hardcoded table", async () => {
    __setRecruitmentProbeLauncher(async () => ({
      output: "NATIVE_NESTING_AVAILABLE=true",
      timedOut: false,
      exitCode: 0,
    }));
    const result = await probeNativeNesting();
    assert.equal(result.supported, true);
    assert.equal(result.verdict, "available");
  });

  it("probeNativeNesting caches the result for the process -- exactly one launch across two calls", async () => {
    let launches = 0;
    __setRecruitmentProbeLauncher(async () => {
      launches += 1;
      return { output: "NATIVE_NESTING_AVAILABLE=false", timedOut: false, exitCode: 0 };
    });

    const first = await probeNativeNesting();
    const second = await probeNativeNesting();

    assert.equal(launches, 1, "expected exactly one launch across two calls");
    assert.deepEqual(first, second);
  });

  it("probeNativeNesting retries exactly once, and only on a timeout", async () => {
    let calls = 0;
    __setRecruitmentProbeLauncher(async () => {
      calls += 1;
      if (calls === 1) {
        return { output: "", timedOut: true, exitCode: null };
      }
      return { output: "NATIVE_NESTING_AVAILABLE=true", timedOut: false, exitCode: 0 };
    });

    const result = await probeNativeNesting();
    assert.equal(calls, 2, "expected exactly one retry after a timeout");
    assert.equal(result.supported, true);
  });

  it("probeNativeNesting does not retry a non-timeout failure", async () => {
    let calls = 0;
    __setRecruitmentProbeLauncher(async () => {
      calls += 1;
      return { output: "some other error", timedOut: false, exitCode: 1 };
    });

    const result = await probeNativeNesting();
    assert.equal(calls, 1, "a non-timeout failure must not retry");
    assert.equal(result.supported, false);
  });

  it("probe result carries the budget it ran under", async () => {
    process.env["AETHER_RECRUIT_PROBE_TIMEOUT"] = "3300ms";
    __setRecruitmentProbeLauncher(async () => ({
      output: "NATIVE_NESTING_AVAILABLE=false",
      timedOut: false,
      exitCode: 0,
    }));
    const result = await probeNativeNesting();
    assert.equal(result.budgetMs, 3300);
  });

  it("a grandchild holding the pipes open is still torn down when the budget expires", async () => {
    const shPath = "/bin/sh";
    if (!existsSync(shPath)) {
      return;
    }
    const dir = mkdtempProviderDir();
    const scriptPath = join(dir, "codex");
    writeFileSync(
      scriptPath,
      `#!/bin/sh
(sleep 30 &)
exit 0
`,
      { mode: 0o755 }
    );

    process.env["AETHER_RECRUIT_BINARY"] = scriptPath;
    process.env["AETHER_RECRUIT_PROBE_TIMEOUT"] = "300ms";

    const start = Date.now();
    const result = await probeNativeNesting();
    const elapsed = Date.now() - start;

    assert.ok(elapsed < 5_000, `expected the probe to resolve well under 5s despite the grandchild sleep, took ${elapsed}ms`);
    assert.equal(result.supported, false);
  });
});

describe("platform-dispatcher.ts runProbe (retired auth probe, regression-tested only)", () => {
  afterEach(() => {
    delete process.env["AETHER_PREFLIGHT_TIMEOUT"];
    delete process.env["PATH"];
  });

  it("resolveProbeTimeoutMs shares the one AETHER_PREFLIGHT_TIMEOUT setting -- no second, disagreeing knob", async () => {
    // This project deliberately retired a second AETHER_PROBE_TIMEOUT knob on
    // the Go side (pkg/codex/preflight_phase_198_3_test.go,
    // TestLiveReadinessSourceHasOneTimeoutEnvironmentVariable): every live
    // readiness path shares AETHER_PREFLIGHT_TIMEOUT so one setting means one
    // value on both hosts. Introducing AETHER_PROBE_TIMEOUT here, on the TS
    // side only, would reopen exactly that defect on one host -- see this
    // plan's own deviation note in the SUMMARY.
    const { resolveProbeTimeoutMs } = await import("../src/platform-dispatcher.js");
    delete process.env["AETHER_PREFLIGHT_TIMEOUT"];
    assert.equal(resolveProbeTimeoutMs(), resolvePreflightTimeoutMs());
    process.env["AETHER_PREFLIGHT_TIMEOUT"] = "6s";
    assert.equal(resolveProbeTimeoutMs(), 6_000);
  });

  it("platform-dispatcher.ts no longer hardcodes the 5000ms auth-probe timeout", async () => {
    const { readFileSync } = await import("node:fs");
    const { fileURLToPath } = await import("node:url");
    const source = readFileSync(
      fileURLToPath(new URL("../src/platform-dispatcher.ts", import.meta.url)),
      "utf-8"
    );
    assert.ok(!source.includes("timeout: 5000"), "source must not contain the old hardcoded timeout: 5000 literal");
    assert.ok(source.includes("detached"), "source must use detached process-group spawning");
  });

  it("isPlatformAvailable still resolves false when the auth probe reports no credentials", async () => {
    const dir = mkdtempProviderDir();
    const binary = join(dir, "codex");
    writeFileSync(
      binary,
      `#!/bin/sh
echo "Not logged in"
exit 0
`,
      { mode: 0o755 }
    );
    const originalPath = process.env["PATH"];
    const originalCodexPath = process.env["AETHER_CODEX_PATH"];
    process.env["PATH"] = originalPath ? `${dir}:${originalPath}` : dir;
    process.env["AETHER_CODEX_PATH"] = binary;
    try {
      const available = await isPlatformAvailable("codex");
      assert.equal(available, false);
    } finally {
      restoreEnv("PATH", originalPath);
      restoreEnv("AETHER_CODEX_PATH", originalCodexPath);
    }
  });

  it("a grandchild holding the auth probe's pipes open is still torn down when the budget expires", async () => {
    const dir = mkdtempProviderDir();
    const binary = join(dir, "codex");
    writeFileSync(
      binary,
      `#!/bin/sh
(sleep 30 &)
exit 0
`,
      { mode: 0o755 }
    );
    const originalPath = process.env["PATH"];
    const originalCodexPath = process.env["AETHER_CODEX_PATH"];
    process.env["PATH"] = originalPath ? `${dir}:${originalPath}` : dir;
    process.env["AETHER_CODEX_PATH"] = binary;
    process.env["AETHER_PREFLIGHT_TIMEOUT"] = "300ms";
    try {
      const start = Date.now();
      const available = await isPlatformAvailable("codex");
      const elapsed = Date.now() - start;
      assert.ok(elapsed < 5_000, `expected isPlatformAvailable to resolve well under 5s, took ${elapsed}ms`);
      assert.equal(available, false);
    } finally {
      restoreEnv("PATH", originalPath);
      restoreEnv("AETHER_CODEX_PATH", originalCodexPath);
    }
  });
});
