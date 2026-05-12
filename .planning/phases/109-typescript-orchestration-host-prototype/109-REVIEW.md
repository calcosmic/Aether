---
phase: 109-typescript-orchestration-host-prototype
reviewed: 2026-05-12T12:00:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - .aether/ts-host/src/go-bridge.ts
  - .aether/ts-host/src/host.ts
  - .aether/ts-host/src/lifecycle.ts
  - .aether/ts-host/src/types.ts
  - .aether/ts-host/src/worker-dispatch.ts
  - .aether/ts-host/test/boundary.test.ts
  - .aether/ts-host/test/go-bridge.test.ts
  - .aether/ts-host/test/host.test.ts
  - .aether/ts-host/test/lifecycle.test.ts
  - .aether/ts-host/test/worker-dispatch.test.ts
  - .aether/ts-host/tsconfig.build.json
findings:
  critical: 1
  warning: 5
  info: 4
  total: 10
status: issues_found
---

# Phase 109: Code Review Report

**Reviewed:** 2026-05-12T12:00:00Z
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues_found

## Summary

Reviewed the TypeScript orchestration host prototype -- a module that calls Go CLI commands for plan/build/continue lifecycle orchestration and dispatches workers. The codebase is well-structured with strong boundary enforcement preventing direct writes to `.aether/data/`. However, there is one critical bug: `host.ts` test expectations are stale and contradict the current implementation (the lifecycle command is fully implemented, not "not yet implemented"), meaning the test will either pass incorrectly or fail depending on runtime behavior. Additionally, there are several warnings around boundary check bypass patterns, missing entry point file, and error handling gaps.

## Critical Issues

### CR-01: Host test asserts stale "not yet implemented" behavior -- test contradicts actual implementation

**File:** `.aether/ts-host/test/host.test.ts:49-65`
**Issue:** The test at line 49 ("host lifecycle command prints not-yet-implemented message") expects the `lifecycle` command to output "not yet implemented" or "Lifecycle". However, `host.ts` lines 105-122 fully implement the lifecycle command -- it calls `runLifecycle()` and outputs the JSON result. This means:
1. If `runLifecycle()` succeeds, the output will be a JSON result on stdout, and stderr will contain "Plan finalized successfully", "Build finalized successfully", etc. The assertion checks for "not yet implemented" or "Lifecycle" on stderr, so it could spuriously pass due to the case-insensitive match on "Lifecycle" from stderr progress messages like "Lifecycle error:".
2. If `runLifecycle()` fails (no colony state in the test CWD), the error will be "Failed at plan: ..." which may or may not contain "Lifecycle" as a substring.

The test validates the wrong behavior and was never updated when the lifecycle command was implemented. This is a correctness bug in the test suite that masks a real implementation state.

**Fix:**
```typescript
// host.test.ts line 49 -- replace with a test that matches actual behavior
it("host lifecycle command executes and produces JSON output or lifecycle error", () => {
  const result = spawnSync(
    "node",
    ["--import", "tsx", hostPath, "lifecycle"],
    {
      encoding: "utf-8",
      timeout: 10000,
    }
  );

  // Either succeeds with JSON on stdout, or fails with a lifecycle error on stderr
  const stdout = result.stdout ?? "";
  const stderr = result.stderr ?? "";
  const hasJsonOutput = stdout.includes("{");
  const hasError = stderr.includes("Failed at");
  assert.ok(
    hasJsonOutput || hasError,
    `Should produce JSON output or lifecycle error. stdout: ${stdout.slice(0, 200)}, stderr: ${stderr.slice(0, 200)}`
  );
});
```

## Warnings

### WR-01: Boundary check can be bypassed via path traversal using `..` segments

**File:** `.aether/ts-host/src/go-bridge.ts:144-153`
**Issue:** The `assertNoDirectDataWrites` function normalizes backslashes to forward slashes but does not resolve `..` segments. A path like `foo/..aether/data/COLONY_STATE.json` would pass because it does not start with `.aether/data/` after simple string matching, but more critically, a caller providing something like `../../.aether/data/COLONY_STATE.json` would be caught by `includes()` but a crafted path like `/tmp/.aether-data/COLONY_STATE.json` would be incorrectly rejected because `includes(".aether/data/")` matches the substring. The check uses both `startsWith` and `includes`, where `includes` is overly broad (any path containing the substring anywhere is blocked, even legitimate temp paths that happen to contain `.aether/data` as a substring in a parent directory name) while `..` traversal is not handled.

**Fix:**
```typescript
export function assertNoDirectDataWrites(filePath: string): void {
  const normalized = filePath.replace(/\\/g, "/");
  // Resolve .. segments to prevent traversal bypass
  const resolved = normalized.split("/").reduce<string[]>((acc, segment) => {
    if (segment === "..") acc.pop();
    else if (segment !== ".") acc.push(segment);
    return acc;
  }, []).join("/");

  for (const goPath of GO_OWNED_PATHS) {
    if (resolved.startsWith(goPath) || resolved.includes("/" + goPath)) {
      throw new Error(
        `Boundary violation: TS host must not write to Go-owned path "${goPath}". ` +
          `Path attempted: "${filePath}". Use Go finalizer commands instead.`
      );
    }
  }
}
```

### WR-02: `writeCompletionFile` `dir` parameter allows directory traversal outside tmpdir

**File:** `.aether/ts-host/src/go-bridge.ts:171-189`
**Issue:** The `dir` parameter is joined directly with `tmpdir()` using `join()`. If a caller passes `dir` as `../../../etc`, the resulting path would be `join(tmpdir(), "../../../etc")` which resolves to `/etc` on Unix systems. The `writeFileSync` would then attempt to write to a location outside the temp directory. While currently all callers pass safe values (`"aether-lifecycle"`, `"aether-completions"`), the function signature accepts arbitrary strings with no validation.

**Fix:**
```typescript
export function writeCompletionFile(
  dir: string,
  filename: string,
  data: unknown
): string {
  const targetDir = join(tmpdir(), dir);
  const targetPath = join(targetDir, filename);

  // Verify the resolved path is still under tmpdir
  if (!targetPath.startsWith(tmpdir())) {
    throw new Error(
      `Completion file path escapes tmpdir: ${targetPath}`
    );
  }
  // ... rest of function
}
```

### WR-03: Missing `index.ts` entry point referenced by package.json

**File:** `.aether/ts-host/package.json:9` (references `"main": "dist/index.js"`)
**Issue:** `package.json` declares `"main": "dist/index.js"`, but there is no `src/index.ts` file to compile into `dist/index.js`. Any attempt to import this as a package (e.g., `import { callGoJSON } from "@aether/ts-host"`) will fail. The `build` script (`tsc -p tsconfig.build.json`) will compile `src/` files but will not produce `dist/index.js` because the source does not exist. If this module is only used via direct CLI invocation (`node dist/host.js`), the `"main"` field should be updated to `"dist/host.js"`; otherwise, an `index.ts` re-exporting the public API needs to be created.

**Fix:** Either create `src/index.ts` with re-exports:
```typescript
export { callGoJSON, discoverGoBinary, writeCompletionFile, assertNoDirectDataWrites } from "./go-bridge.js";
export type { GoBridgeOptions } from "./go-bridge.js";
export { runLifecycle } from "./lifecycle.js";
export type { LifecycleOptions, LifecycleResult } from "./lifecycle.js";
export { dispatchSingleWorker, dispatchWorkers, toWorkerResults } from "./worker-dispatch.js";
export type { DispatchOptions, DispatchResult } from "./worker-dispatch.js";
```
Or update `package.json` main to `"dist/host.js"`.

### WR-04: `discoverGoBinary` trusts `AETHER_BINARY_PATH` without validation

**File:** `.aether/ts-host/src/go-bridge.ts:48-50`
**Issue:** The function returns `AETHER_BINARY_PATH` immediately without verifying the path exists or is executable. If the environment variable points to a non-existent file, subsequent `callGoJSON` calls will fail with a confusing ENOENT error from `execFileSync` rather than a clear "binary not found" message. More importantly, if the variable is set to an attacker-controlled path (e.g., in a CI environment), the host would execute an arbitrary binary with the full trust of the Go runtime.

**Fix:**
```typescript
export function discoverGoBinary(): string {
  const envPath = process.env["AETHER_BINARY_PATH"];
  if (envPath) {
    if (!existsSync(envPath)) {
      throw new Error(
        `AETHER_BINARY_PATH is set to "${envPath}" but the file does not exist`
      );
    }
    return envPath;
  }
  // ... rest of discovery
}
```

### WR-05: `lifecycle.ts` writes to `.aether/ts-host/` inside the repo working tree -- boundary violation

**File:** `.aether/ts-host/src/lifecycle.ts:238-253`
**Issue:** The `runLifecycle` function creates a directory `.aether/ts-host/` and writes `SIMULATED_BUILD_OUTPUT.txt` directly into the repo's working tree (lines 241-253). This is a direct filesystem write that bypasses the Go runtime, and it writes into `.aether/` which is a Go-owned directory structure. While it is not writing to `.aether/data/` specifically, the boundary contract (`boundary-reference.ts`) defines `.aether/` as owned by Go. The `writeFileSync` call here is also inconsistent with the stated design principle that "The TS host never writes to .aether/ directly."

**Fix:** Write the simulated placeholder to `tmpdir()` instead, or use a Go CLI command to create the placeholder file:
```typescript
const placeholderDir = join(tmpdir(), "aether-ts-host-placeholder");
mkdirSync(placeholderDir, { recursive: true });
const placeholderPath = join(placeholderDir, "SIMULATED_BUILD_OUTPUT.txt");
writeFileSync(placeholderPath, "Simulated build output\n", "utf-8");
// Use an absolute path for file claims
const placeholderRel = placeholderPath;
```

## Info

### IN-01: Test files import from `../src/go-bridge.js` with `.js` extension for TypeScript ESM

**File:** `.aether/ts-host/test/go-bridge.test.ts:26`, `.aether/ts-host/test/boundary.test.ts:23`, `.aether/ts-host/test/worker-dispatch.test.ts:22`, `.aether/ts-host/test/lifecycle.test.ts:22`
**Issue:** All test files import source modules using `.js` extensions (e.g., `from "../src/go-bridge.js"`). This is standard practice for NodeNext module resolution with TypeScript ESM, so this is not a bug. However, the tests run via `tsx` which handles the `.ts` -> `.js` mapping automatically. If tests are ever run against the compiled `dist/` output instead, the imports would need updating. This is noted for awareness only.

### IN-02: `boundary.test.ts` static analysis test may miss indirect write patterns

**File:** `.aether/ts-host/test/boundary.test.ts:176-211`
**Issue:** The static analysis test scans for literal patterns like `writeFileSync.*\.aether\/data` in source files. This would miss:
- Dynamic path construction: `const p = ".aether" + "/data/"; writeFileSync(p, ...)`
- Imports aliased: `import { writeFileSync as wf } from "node:fs"; wf(".aether/data/...", ...)`
- Template literals with computed segments

The test provides reasonable coverage for straightforward violations but is not exhaustive. The runtime `assertNoDirectDataWrites` check is the real enforcement mechanism, making this a defense-in-depth layer.

### IN-03: `worker-dispatch.ts` uses `process.stderr.write` instead of a structured logger

**File:** `.aether/ts-host/src/worker-dispatch.ts:91-98, 149-156, 200-210`
**Issue:** The module writes progress and warning messages directly to `process.stderr.write`. This is acceptable for a prototype but will need a proper logging abstraction for production use. No action needed now.

### IN-04: `host.test.ts` test for "no args" has a potential false-pass path

**File:** `.aether/ts-host/test/host.test.ts:28-46`
**Issue:** The test catches the `execFileSync` exception to get stderr, but if the process somehow succeeds (unlikely but possible if the host behavior changes), `stderr` would be set to the stdout content. The assertion at line 43 checks `stderr.includes("Usage:") || stderr.includes("command")` -- the word "command" is generic enough that it could match unrelated output. The test should verify that the exit code was 1 (which it does at line 39) and also verify stderr content more specifically.

---

_Reviewed: 2026-05-12T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
