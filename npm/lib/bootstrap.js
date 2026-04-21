"use strict";

const crypto = require("node:crypto");
const fs = require("node:fs");
const fsp = require("node:fs/promises");
const http = require("node:http");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");
const { spawnSync, execFileSync } = require("node:child_process");
const packageJson = require("../package.json");

const REPO_OWNER = "calcosmic";
const REPO_NAME = "Aether";
const DEFAULT_AETHER_VERSION = packageJson.version;
const PACKAGE_VERSION = packageJson.version;
const MAX_REDIRECTS = 5;
const BANNER = `
      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
`;

function printBootstrapHelp() {
  console.log(BANNER.trimEnd());
  console.log("");
  console.log("Aether npm bootstrap");
  console.log("");
  console.log("Usage:");
  console.log("  npx --yes aether-colony@latest");
  console.log("  npx --yes aether-colony@latest -- <aether args>");
  console.log("");
  console.log("What it does:");
  console.log("  1. Downloads the matching Go release binary for your platform");
  console.log("  2. Installs it into a stable local directory");
  console.log("  3. Runs the real `aether` CLI");
  console.log("");
  console.log("Flags:");
  console.log("  --bootstrap-help          Show this help");
  console.log("  --bootstrap-version       Print the npm wrapper version");
  console.log("  --aether-version <ver>    Override the Go release version to install");
  console.log("  --dest <path>             Override the install directory for the Go binary");
  console.log("");
  console.log("Examples:");
  console.log("  npx --yes aether-colony@latest");
  console.log("  npx --yes aether-colony@latest -- status");
  console.log("  npx --yes aether-colony@latest -- update --force --download-binary");
}

function normalizeArgs(argv) {
  const args = [...argv];
  const bootstrap = {
    help: false,
    wrapperVersion: false,
    aetherVersion: DEFAULT_AETHER_VERSION,
    dest: null,
    passthrough: []
  };

  for (let i = 0; i < args.length; i += 1) {
    const arg = args[i];
    if (arg === "--") {
      bootstrap.passthrough = args.slice(i + 1);
      return bootstrap;
    }
    if (arg === "--bootstrap-help") {
      bootstrap.help = true;
      continue;
    }
    if (arg === "--bootstrap-version") {
      bootstrap.wrapperVersion = true;
      continue;
    }
    if (arg === "--aether-version") {
      bootstrap.aetherVersion = normalizeVersion(args[i + 1]);
      i += 1;
      continue;
    }
    if (arg.startsWith("--aether-version=")) {
      bootstrap.aetherVersion = normalizeVersion(arg.split("=", 2)[1]);
      continue;
    }
    if (arg === "--dest") {
      bootstrap.dest = args[i + 1];
      i += 1;
      continue;
    }
    if (arg.startsWith("--dest=")) {
      bootstrap.dest = arg.split("=", 2)[1];
      continue;
    }
    bootstrap.passthrough = args.slice(i);
    return bootstrap;
  }

  return bootstrap;
}

function normalizeVersion(version) {
  return String(version || "").trim().replace(/^v/, "");
}

function detectPlatform(nodePlatform = process.platform, nodeArch = process.arch) {
  const osMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows"
  };
  const archMap = {
    x64: "amd64",
    arm64: "arm64"
  };

  const osName = osMap[nodePlatform];
  const archName = archMap[nodeArch];
  if (!osName || !archName) {
    throw new Error(`Unsupported platform: ${nodePlatform}/${nodeArch}. Supported: darwin/linux/windows + x64/arm64.`);
  }
  return { os: osName, arch: archName };
}

function archiveFilename(version, platform) {
  const ext = platform.os === "windows" ? ".zip" : ".tar.gz";
  return `aether_v${version}_${platform.os}_${platform.arch}${ext}`;
}

function checksumsFilename(version) {
  return `aether_v${version}_checksums.txt`;
}

function releaseBaseURL(version) {
  return `https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/v${version}`;
}

function archiveURL(version, platform) {
  return `${releaseBaseURL(version)}/${archiveFilename(version, platform)}`;
}

function checksumsURL(version) {
  return `${releaseBaseURL(version)}/${checksumsFilename(version)}`;
}

function binaryName(platform) {
  return platform.os === "windows" ? "aether.exe" : "aether";
}

function defaultDestDir(homeDir = os.homedir(), nodePlatform = process.platform) {
  if (process.env.AETHER_BINARY_DEST) {
    return process.env.AETHER_BINARY_DEST;
  }
  if (nodePlatform === "win32") {
    return path.join(homeDir, ".aether", "bin");
  }
  const localBin = path.join(homeDir, ".local", "bin");
  if (fs.existsSync(localBin) && fs.statSync(localBin).isDirectory()) {
    return localBin;
  }
  return path.join(homeDir, ".aether", "bin");
}

function installedBinaryPath(destDir, platform) {
  return path.join(destDir, binaryName(platform));
}

function hubVersionFile(homeDir = os.homedir()) {
  return path.join(homeDir, ".aether", "version.json");
}

function hasHubInstalled(homeDir = os.homedir()) {
  return fs.existsSync(hubVersionFile(homeDir));
}

function isBinaryOnPath(binaryPath) {
  const pathEntries = String(process.env.PATH || "").split(path.delimiter);
  return pathEntries.some((entry) => path.resolve(entry || ".") === path.resolve(path.dirname(binaryPath)));
}

function parseChecksum(content, filename) {
  const lines = String(content).split(/\r?\n/);
  for (const line of lines) {
    const parts = line.split(/\s{2,}/);
    if (parts.length >= 2 && parts[1] === filename) {
      return parts[0].trim();
    }
  }
  throw new Error(`Checksum not found for ${filename}`);
}

function parseVersionOutput(stdout) {
  const trimmed = String(stdout || "").trim();
  if (!trimmed) {
    return "";
  }
  try {
    const parsed = JSON.parse(trimmed);
    if (parsed && parsed.ok === true) {
      return normalizeVersion(parsed.result);
    }
  } catch (_) {
    // fall through
  }
  return normalizeVersion(trimmed);
}

function getInstalledVersion(binaryPath) {
  if (!fs.existsSync(binaryPath)) {
    return "";
  }
  const result = spawnSync(binaryPath, ["version"], {
    encoding: "utf8",
    env: {
      ...process.env,
      AETHER_OUTPUT_MODE: "json"
    }
  });
  if (result.status !== 0) {
    return "";
  }
  return parseVersionOutput(result.stdout);
}

function needsInstall(binaryPath, targetVersion) {
  const installedVersion = getInstalledVersion(binaryPath);
  return installedVersion !== normalizeVersion(targetVersion);
}

function requestWithRedirects(url, redirectsLeft = MAX_REDIRECTS) {
  return new Promise((resolve, reject) => {
    const transport = url.startsWith("https:") ? https : http;
    const req = transport.get(
      url,
      {
        headers: {
          "User-Agent": `${packageJson.name}/${PACKAGE_VERSION}`
        }
      },
      (res) => {
        const status = res.statusCode || 0;
        if ([301, 302, 303, 307, 308].includes(status) && res.headers.location) {
          if (redirectsLeft <= 0) {
            reject(new Error(`Too many redirects while fetching ${url}`));
            res.resume();
            return;
          }
          const nextURL = new URL(res.headers.location, url).toString();
          res.resume();
          resolve(requestWithRedirects(nextURL, redirectsLeft - 1));
          return;
        }
        if (status < 200 || status >= 300) {
          reject(new Error(`HTTP ${status} for ${url}`));
          res.resume();
          return;
        }
        resolve(res);
      }
    );
    req.on("error", reject);
  });
}

async function downloadText(url) {
  const response = await requestWithRedirects(url);
  const chunks = [];
  for await (const chunk of response) {
    chunks.push(chunk);
  }
  return Buffer.concat(chunks).toString("utf8");
}

async function downloadFile(url, destination) {
  const response = await requestWithRedirects(url);
  await fsp.mkdir(path.dirname(destination), { recursive: true });
  const out = fs.createWriteStream(destination);
  await new Promise((resolve, reject) => {
    response.pipe(out);
    response.on("error", reject);
    out.on("error", reject);
    out.on("finish", resolve);
  });
  await fsp.chmod(destination, 0o644);
}

async function sha256File(filePath) {
  const hash = crypto.createHash("sha256");
  const stream = fs.createReadStream(filePath);
  return new Promise((resolve, reject) => {
    stream.on("data", (chunk) => hash.update(chunk));
    stream.on("error", reject);
    stream.on("end", () => resolve(hash.digest("hex")));
  });
}

function extractArchive(archivePath, extractDir, platform) {
  fs.mkdirSync(extractDir, { recursive: true });
  if (platform.os === "windows") {
    execFileSync(
      "powershell.exe",
      [
        "-NoLogo",
        "-NoProfile",
        "-NonInteractive",
        "-Command",
        `Expand-Archive -LiteralPath '${archivePath.replace(/'/g, "''")}' -DestinationPath '${extractDir.replace(/'/g, "''")}' -Force`
      ],
      { stdio: "ignore" }
    );
    return;
  }

  execFileSync("tar", ["-xzf", archivePath, "-C", extractDir], {
    stdio: "ignore"
  });
}

function findBinaryRecursively(baseDir, binaryFile) {
  const entries = fs.readdirSync(baseDir, { withFileTypes: true });
  for (const entry of entries) {
    const fullPath = path.join(baseDir, entry.name);
    if (entry.isFile() && entry.name === binaryFile) {
      return fullPath;
    }
    if (entry.isDirectory()) {
      const nested = findBinaryRecursively(fullPath, binaryFile);
      if (nested) {
        return nested;
      }
    }
  }
  return "";
}

async function installReleaseBinary(version, destDir) {
  const platform = detectPlatform();
  const archiveFile = archiveFilename(version, platform);
  const tmpRoot = await fsp.mkdtemp(path.join(os.tmpdir(), "aether-npm-"));
  const archivePath = path.join(tmpRoot, archiveFile);
  const extractDir = path.join(tmpRoot, "extract");
  const destPath = installedBinaryPath(destDir, platform);

  try {
    const checksums = await downloadText(checksumsURL(version));
    const expected = parseChecksum(checksums, archiveFile);
    await downloadFile(archiveURL(version, platform), archivePath);
    const actual = await sha256File(archivePath);
    if (actual !== expected) {
      throw new Error(`Checksum mismatch for ${archiveFile}`);
    }

    extractArchive(archivePath, extractDir, platform);
    const extractedBinary = findBinaryRecursively(extractDir, binaryName(platform));
    if (!extractedBinary) {
      throw new Error(`Binary ${binaryName(platform)} not found in extracted archive`);
    }

    await fsp.mkdir(destDir, { recursive: true });
    const tmpDest = `${destPath}.tmp-${process.pid}`;
    await fsp.copyFile(extractedBinary, tmpDest);
    if (platform.os !== "windows") {
      await fsp.chmod(tmpDest, 0o755);
    }
    await fsp.rm(destPath, { force: true });
    await fsp.rename(tmpDest, destPath);
    return destPath;
  } finally {
    await fsp.rm(tmpRoot, { recursive: true, force: true });
  }
}

function runAether(binaryPath, args) {
  const result = spawnSync(binaryPath, args, {
    stdio: "inherit",
    env: process.env
  });
  if (typeof result.status === "number") {
    process.exit(result.status);
  }
  if (result.error) {
    throw result.error;
  }
  process.exit(1);
}

async function main(argv) {
  const options = normalizeArgs(argv);
  if (options.help) {
    printBootstrapHelp();
    return;
  }
  if (options.wrapperVersion) {
    console.log(PACKAGE_VERSION);
    return;
  }

  const aetherVersion = normalizeVersion(options.aetherVersion);
  if (!aetherVersion) {
    throw new Error("Missing Aether release version for bootstrap");
  }

  const platform = detectPlatform();
  const destDir = path.resolve(options.dest || defaultDestDir());
  const binaryPath = installedBinaryPath(destDir, platform);
  const firstInstall = !fs.existsSync(binaryPath);
  const hubMissing = !hasHubInstalled();

  console.log(BANNER.trimEnd());
  console.log("");
  console.log(`Bootstrapping Aether ${aetherVersion} via npm package ${packageJson.name}@${PACKAGE_VERSION}`);
  console.log(`Install directory: ${destDir}`);

  if (needsInstall(binaryPath, aetherVersion)) {
    console.log(`Downloading Go release binary for ${platform.os}/${platform.arch}...`);
    await installReleaseBinary(aetherVersion, destDir);
    console.log(`Installed ${binaryPath}`);
  } else {
    console.log(`Found Aether ${aetherVersion} already installed at ${binaryPath}`);
  }

  if (!isBinaryOnPath(binaryPath)) {
    console.log("");
    console.log("PATH note:");
    console.log(`  Add ${destDir} to your PATH if you want to run \`aether\` directly outside npm.`);
  }

  const passthrough = options.passthrough.length > 0 ? options.passthrough : ["install"];
  if ((firstInstall || hubMissing) && passthrough[0] !== "install") {
    console.log("");
    console.log("Initial setup detected. Initializing companion files before handing off to the requested command.");
    const installResult = spawnSync(binaryPath, ["install"], {
      stdio: "inherit",
      env: process.env
    });
    if (installResult.status !== 0) {
      process.exit(typeof installResult.status === "number" ? installResult.status : 1);
    }
  }

  console.log("");
  console.log(`Handing off to: ${path.basename(binaryPath)} ${passthrough.join(" ")}`);
  runAether(binaryPath, passthrough);
}

module.exports = {
  MAX_REDIRECTS,
  archiveFilename,
  archiveURL,
  binaryName,
  checksumsFilename,
  checksumsURL,
  defaultDestDir,
  detectPlatform,
  installedBinaryPath,
  main,
  normalizeArgs,
  normalizeVersion,
  parseChecksum,
  parseVersionOutput,
  hasHubInstalled,
  releaseBaseURL
};
