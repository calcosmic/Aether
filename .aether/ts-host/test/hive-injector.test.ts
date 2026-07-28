/**
 * Unit tests for the hive wisdom injection module.
 *
 * Covers:
 * - HIVE-01: hive-read call and formatting
 * - HIVE-02: domain tag resolution from registry
 * - HIVE-03: empty wisdom returns empty string
 * - HIVE-04: failure logs warning and returns empty string
 * - HIVE-06: stack tags (go, typescript, cli) passed through
 * - HIVE-07: partial match discount (0.5) and sorting
 * - Token budget enforcement
 */

import { describe, it, beforeEach, afterEach } from "node:test";
import assert from "node:assert/strict";

import {
  readHiveWisdom,
  formatHiveWisdomSection,
  resolveDomainTags,
  type HiveInjectorOptions,
} from "../src/hive-injector.js";
import { REPO_ROOT } from "./repo-root.js";
import {
  __setCallGoJSON,
  __restoreCallGoJSON,
  type GoBridgeOptions,
} from "../src/go-bridge.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeEntry(overrides: Partial<import("../src/types.js").HiveWisdomEntry> = {}): import("../src/types.js").HiveWisdomEntry {
  return {
    id: "1",
    text: "Test wisdom",
    domain: "go",
    source_repo: "a",
    source_repos: ["a"],
    confidence: 0.9,
    created_at: "2026-05-18T00:00:00Z",
    accessed_at: "2026-05-18T00:00:00Z",
    access_count: 1,
    ...overrides,
  };
}

function captureStderr(): { lines: string[]; restore: () => void } {
  const lines: string[] = [];
  const original = process.stderr.write.bind(process.stderr);
  const originalMethod = process.stderr.write;
  process.stderr.write = ((chunk: unknown, _encoding?: unknown, _cb?: unknown) => {
    if (typeof chunk === "string") lines.push(chunk);
    return original(chunk as string | Uint8Array, _encoding as BufferEncoding, _cb as ((err?: Error | null) => void));
  }) as typeof process.stderr.write;
  return {
    lines,
    restore: () => {
      process.stderr.write = originalMethod;
    },
  };
}

const defaultOpts: HiveInjectorOptions = {
  goBinaryPath: "/usr/bin/aether",
  cwd: REPO_ROOT,
  domains: ["go"],
  minConfidence: 0.5,
  maxEntries: 10,
  budgetChars: 2500,
};

// ---------------------------------------------------------------------------
// readHiveWisdom
// ---------------------------------------------------------------------------

describe("readHiveWisdom", () => {
  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("calls hive-read with correct args (HIVE-01)", async () => {
    let capturedArgs: string[] | undefined;
    __setCallGoJSON(<T>(_opts: GoBridgeOptions, args: string[]): T => {
      capturedArgs = args;
      return {
        entries: [makeEntry({ text: "Test wisdom", domain: "go", confidence: 0.9 })],
        total: 1,
      } as T;
    });

    const result = await readHiveWisdom(defaultOpts);
		assert.deepEqual(capturedArgs, ["hive-read", "--for-worker", "--min-confidence", "0.5"]);
    assert.ok(result.includes("HIVE WISDOM"));
    assert.ok(result.includes("(go, 0.90) Test wisdom"));
  });

  it("returns empty string when entries is null (HIVE-03)", async () => {
    __setCallGoJSON(<T>(): T => ({ entries: null, total: 0 } as T));
    const result = await readHiveWisdom(defaultOpts);
    assert.equal(result, "");
  });

  it("returns empty string when entries is empty array", async () => {
    __setCallGoJSON(<T>(): T => ({ entries: [], total: 0 } as T));
    const result = await readHiveWisdom(defaultOpts);
    assert.equal(result, "");
  });

  it("logs warning and returns empty string on hive-read failure (HIVE-04)", async () => {
    __setCallGoJSON(() => {
      throw new Error("corrupted wisdom.json");
    });
    const capture = captureStderr();
    try {
      const result = await readHiveWisdom(defaultOpts);
      assert.equal(result, "");
      const stderrText = capture.lines.join("");
      assert.ok(
        stderrText.includes("Warning: hive-read failed: corrupted wisdom.json"),
        `Expected warning in stderr, got: ${stderrText}`
      );
    } finally {
      capture.restore();
    }
  });

  it("enforces token budget by dropping oldest entries", async () => {
    const entries = Array.from({ length: 20 }, (_, i) =>
      makeEntry({
        id: String(i + 1),
        text: "A".repeat(180),
        domain: "go",
        confidence: 0.9,
        created_at: new Date(Date.now() - i * 86400000).toISOString(),
      })
    );
    __setCallGoJSON(<T>(): T => ({ entries, total: 20 } as T));
    const result = await readHiveWisdom(defaultOpts);
    assert.ok(result.length <= 2500, `Result length ${result.length} exceeds budget`);
  });

  it("truncates single entry if it exceeds budget", async () => {
    __setCallGoJSON(<T>(): T => ({
      entries: [makeEntry({ text: "B".repeat(3000), domain: "go", confidence: 0.9 })],
      total: 1,
    } as T));
    const result = await readHiveWisdom(defaultOpts);
    assert.ok(result.length <= 2500, `Result length ${result.length} exceeds budget`);
    assert.ok(result.includes("## HIVE WISDOM (Cross-Colony Patterns)"), "Header should be preserved");
  });
});

// ---------------------------------------------------------------------------
// formatHiveWisdomSection
// ---------------------------------------------------------------------------

describe("formatHiveWisdomSection", () => {
  it("produces correct markdown with header and entries (HIVE-01)", () => {
    const out = formatHiveWisdomSection([
      makeEntry({ text: "Prefer table-driven tests", domain: "go", confidence: 0.95 }),
      makeEntry({ text: "Use strict mode", domain: "typescript", confidence: 0.8 }),
    ]);
    assert.ok(out.startsWith("## HIVE WISDOM (Cross-Colony Patterns)"));
    assert.ok(out.includes("(go, 0.95) Prefer table-driven tests"));
    assert.ok(out.includes("(typescript, 0.80) Use strict mode"));
  });

  it("returns empty string for empty entries", () => {
    assert.equal(formatHiveWisdomSection([]), "");
  });

  it("formats confidence to two decimal places", () => {
    const out = formatHiveWisdomSection([makeEntry({ confidence: 0.5 })]);
    assert.ok(out.includes("(go, 0.50)"));
  });
});

// ---------------------------------------------------------------------------
// resolveDomainTags
// ---------------------------------------------------------------------------

describe("resolveDomainTags", () => {
  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("returns domains for matching repo path (HIVE-02, HIVE-06)", async () => {
    __setCallGoJSON(<T>(): T => ({
      colonies: [
        {
          repo_path: REPO_ROOT,
          domains: ["go", "typescript", "cli"],
          active: true,
          registered_at: "2026-05-18T00:00:00Z",
        },
      ],
    } as T));
    const result = await resolveDomainTags({
      goBinaryPath: "/usr/bin/aether",
      cwd: REPO_ROOT,
    });
    assert.deepEqual(result, ["go", "typescript", "cli"]);
  });

  it("returns empty array when no colony matches", async () => {
    __setCallGoJSON(<T>(): T => ({
      colonies: [
        {
          repo_path: "/other/repo",
          domains: ["python"],
          active: true,
          registered_at: "2026-05-18T00:00:00Z",
        },
      ],
    } as T));
    const result = await resolveDomainTags({
      goBinaryPath: "/usr/bin/aether",
      cwd: REPO_ROOT,
    });
    assert.deepEqual(result, []);
  });

  it("returns empty array when registry-list fails", async () => {
    __setCallGoJSON(() => {
      throw new Error("registry corrupted");
    });
    const result = await resolveDomainTags({
      goBinaryPath: "/usr/bin/aether",
      cwd: REPO_ROOT,
    });
    assert.deepEqual(result, []);
  });

  it("matches by resolved path (symlinks/normalization)", async () => {
    __setCallGoJSON(<T>(): T => ({
      colonies: [
        {
          repo_path: REPO_ROOT + "/", // trailing slash variant
          domains: ["go", "typescript"],
          active: true,
          registered_at: "2026-05-18T00:00:00Z",
        },
      ],
    } as T));
    const result = await resolveDomainTags({
      goBinaryPath: "/usr/bin/aether",
      cwd: REPO_ROOT,
    });
    assert.deepEqual(result, ["go", "typescript"]);
  });
});

// ---------------------------------------------------------------------------
// Relevance scoring (HIVE-07)
// ---------------------------------------------------------------------------

describe("relevance scoring", () => {
  afterEach(() => {
    __restoreCallGoJSON();
  });

  it("applies full confidence for exact domain match", async () => {
    __setCallGoJSON(<T>(): T => ({
      entries: [makeEntry({ domain: "go", confidence: 0.9, text: "Go tip" })],
      total: 1,
    } as T));
    const result = await readHiveWisdom({ ...defaultOpts, domains: ["go"] });
    assert.ok(result.includes("(go, 0.90) Go tip"));
  });

  it("applies 0.5 discount for partial domain match and sorts correctly", async () => {
    __setCallGoJSON(<T>(): T => ({
      entries: [
        makeEntry({ domain: "typescript", confidence: 0.8, text: "TS tip" }),
        makeEntry({ domain: "go", confidence: 0.5, text: "Go tip" }),
      ],
      total: 2,
    } as T));
    const result = await readHiveWisdom({ ...defaultOpts, domains: ["go"] });
    const goIndex = result.indexOf("Go tip");
    const tsIndex = result.indexOf("TS tip");
    assert.ok(goIndex > 0, "Go tip should appear in output");
    assert.ok(tsIndex > 0, "TS tip should appear in output");
    assert.ok(goIndex < tsIndex, "Go tip should appear before TS tip because 0.5 > 0.4");
  });

  it("sorts by discounted confidence descending", async () => {
    __setCallGoJSON(<T>(): T => ({
      entries: [
        makeEntry({ domain: "go", confidence: 0.6, text: "Mid" }),
        makeEntry({ domain: "go", confidence: 0.95, text: "High" }),
        makeEntry({ domain: "typescript", confidence: 0.9, text: "Low" }),
      ],
      total: 3,
    } as T));
    const result = await readHiveWisdom({ ...defaultOpts, domains: ["go"] });
    const highIndex = result.indexOf("High");
    const midIndex = result.indexOf("Mid");
    const lowIndex = result.indexOf("Low");
    assert.ok(highIndex < midIndex, "High (0.95) should come before Mid (0.6)");
    assert.ok(midIndex < lowIndex, "Mid (0.6) should come before Low (0.45)");
  });
});
