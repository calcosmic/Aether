"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");

const bootstrap = require("../lib/bootstrap");

test("detectPlatform maps supported platforms", () => {
  assert.deepEqual(bootstrap.detectPlatform("darwin", "arm64"), { os: "darwin", arch: "arm64" });
  assert.deepEqual(bootstrap.detectPlatform("linux", "x64"), { os: "linux", arch: "amd64" });
  assert.deepEqual(bootstrap.detectPlatform("win32", "x64"), { os: "windows", arch: "amd64" });
});

test("detectPlatform rejects unsupported platforms", () => {
  assert.throws(() => bootstrap.detectPlatform("freebsd", "x64"), /Unsupported platform/);
  assert.throws(() => bootstrap.detectPlatform("linux", "ia32"), /Unsupported platform/);
});

test("archive helpers match goreleaser naming", () => {
  const platform = { os: "darwin", arch: "arm64" };
  assert.equal(bootstrap.archiveFilename("1.0.17", platform), "aether_v1.0.17_darwin_arm64.tar.gz");
  assert.equal(
    bootstrap.archiveURL("1.0.17", platform),
    "https://github.com/calcosmic/Aether/releases/download/v1.0.17/aether_v1.0.17_darwin_arm64.tar.gz"
  );
  assert.equal(
    bootstrap.checksumsURL("1.0.17"),
    "https://github.com/calcosmic/Aether/releases/download/v1.0.17/aether_v1.0.17_checksums.txt"
  );
});

test("normalizeArgs separates bootstrap flags from passthrough args", () => {
  const parsed = bootstrap.normalizeArgs(["--aether-version", "1.2.3", "--dest", "/tmp/aether", "--", "status"]);
  assert.equal(parsed.aetherVersion, "1.2.3");
  assert.equal(parsed.dest, "/tmp/aether");
  assert.deepEqual(parsed.passthrough, ["status"]);
});

test("parseChecksum extracts the right checksum line", () => {
  const content = [
    "aaa111  aether_v1.0.17_linux_amd64.tar.gz",
    "bbb222  aether_v1.0.17_darwin_arm64.tar.gz"
  ].join("\n");
  assert.equal(bootstrap.parseChecksum(content, "aether_v1.0.17_darwin_arm64.tar.gz"), "bbb222");
});

test("parseVersionOutput handles JSON and plain text", () => {
  assert.equal(bootstrap.parseVersionOutput("{\"ok\":true,\"result\":\"1.0.17\"}"), "1.0.17");
  assert.equal(bootstrap.parseVersionOutput("v1.0.17"), "1.0.17");
});
