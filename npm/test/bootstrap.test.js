"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");

const bootstrap = require("../lib/bootstrap");
const packageJson = require("../package.json");

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
  assert.equal(bootstrap.archiveFilename("1.2.3", platform), "Aether_1.2.3_darwin_arm64.tar.gz");
  assert.equal(
    bootstrap.archiveURL("1.2.3", platform),
    "https://github.com/calcosmic/Aether/releases/download/v1.2.3/Aether_1.2.3_darwin_arm64.tar.gz"
  );
  assert.equal(
    bootstrap.checksumsURL("1.2.3"),
    "https://github.com/calcosmic/Aether/releases/download/v1.2.3/checksums.txt"
  );
});

test("normalizeArgs separates bootstrap flags from passthrough args", () => {
  const parsed = bootstrap.normalizeArgs(["--aether-version", "1.2.3", "--dest", "/tmp/aether", "--", "status"]);
  assert.equal(parsed.aetherVersion, "1.2.3");
  assert.equal(parsed.dest, "/tmp/aether");
  assert.deepEqual(parsed.passthrough, ["status"]);
});

test("bootstrap defaults to the published package version", () => {
  const parsed = bootstrap.normalizeArgs([]);
  assert.equal(parsed.aetherVersion, packageJson.version);
});

test("parseChecksum extracts the right checksum line", () => {
  const content = [
    "aaa111  Aether_1.2.3_linux_amd64.tar.gz",
    "bbb222  Aether_1.2.3_darwin_arm64.tar.gz"
  ].join("\n");
  assert.equal(bootstrap.parseChecksum(content, "Aether_1.2.3_darwin_arm64.tar.gz"), "bbb222");
});

test("parseVersionOutput handles JSON and plain text", () => {
  assert.equal(bootstrap.parseVersionOutput("{\"ok\":true,\"result\":\"1.2.3\"}"), "1.2.3");
  assert.equal(bootstrap.parseVersionOutput("v1.2.3"), "1.2.3");
});

test("release base URL permits HTTPS mirrors and loopback acceptance servers", () => {
  const original = process.env[bootstrap.RELEASE_BASE_URL_ENV];
  try {
    process.env[bootstrap.RELEASE_BASE_URL_ENV] = "https://releases.example.test/aether/";
    assert.equal(bootstrap.releaseBaseURL("1.2.3"), "https://releases.example.test/aether/v1.2.3");
    process.env[bootstrap.RELEASE_BASE_URL_ENV] = "http://127.0.0.1:8123";
    assert.equal(bootstrap.releaseBaseURL("1.2.3"), "http://127.0.0.1:8123/v1.2.3");
  } finally {
    restoreEnv(bootstrap.RELEASE_BASE_URL_ENV, original);
  }
});

test("release base URL rejects insecure remote mirrors and embedded credentials", () => {
  const original = process.env[bootstrap.RELEASE_BASE_URL_ENV];
  try {
    process.env[bootstrap.RELEASE_BASE_URL_ENV] = "http://releases.example.test/aether";
    assert.throws(() => bootstrap.releaseBaseURL("1.2.3"), /must use HTTPS/);
    process.env[bootstrap.RELEASE_BASE_URL_ENV] = "https://user:secret@releases.example.test/aether";
    assert.throws(() => bootstrap.releaseBaseURL("1.2.3"), /must not contain credentials/);
  } finally {
    restoreEnv(bootstrap.RELEASE_BASE_URL_ENV, original);
  }
});

test("activation refuses a wrong-version binary without replacing the installed binary", async () => {
  const root = await fsp.mkdtemp(path.join(os.tmpdir(), "aether-bootstrap-test-"));
  const installed = path.join(root, process.platform === "win32" ? "aether.exe" : "aether");
  const staged = path.join(root, process.platform === "win32" ? "staged.exe" : "staged");
  try {
    await fsp.copyFile(process.execPath, installed);
    await fsp.copyFile(process.execPath, staged);
    const before = await fsp.readFile(installed);
    await assert.rejects(
      bootstrap.activateReleaseBinary(staged, installed, "0.0.0-impossible"),
      /Downloaded binary version mismatch/
    );
    assert.deepEqual(await fsp.readFile(installed), before);
    assert.equal(fs.existsSync(bootstrap.rollbackBinaryPath(installed)), false);
  } finally {
    await fsp.rm(root, { recursive: true, force: true });
  }
});

test("interrupted activation restores the previous binary before retry", async () => {
  const root = await fsp.mkdtemp(path.join(os.tmpdir(), "aether-bootstrap-recover-"));
  const installed = path.join(root, process.platform === "win32" ? "aether.exe" : "aether");
  const rollback = bootstrap.rollbackBinaryPath(installed);
  try {
    await fsp.copyFile(process.execPath, rollback);
    const result = await bootstrap.recoverInterruptedInstall(installed, process.version);
    assert.equal(result, "restored");
    assert.equal(fs.existsSync(installed), true);
    assert.equal(fs.existsSync(rollback), false);
  } finally {
    await fsp.rm(root, { recursive: true, force: true });
  }
});

function restoreEnv(name, value) {
  if (value === undefined) {
    delete process.env[name];
  } else {
    process.env[name] = value;
  }
}
