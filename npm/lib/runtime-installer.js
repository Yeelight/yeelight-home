"use strict";

const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const crypto = require("node:crypto");
const { spawnSync } = require("node:child_process");

const packageRoot = path.resolve(__dirname, "..", "..");
const packageInfo = require(path.join(packageRoot, "package.json"));
const officialRepo = "Yeelight/yeelight-home";

function ensureRuntimeBinary() {
  let target;
  try {
    target = resolveTarget();
  } catch (error) {
    return { ok: false, code: 2, message: error.message };
  }

  const binaryPath = resolveBinaryPath(target);
  if (fs.existsSync(binaryPath)) {
    return { ok: true, binaryPath };
  }

  try {
    installRuntime(target, binaryPath);
  } catch (error) {
    return {
      ok: false,
      code: 1,
      message: `failed to install yeelight-home runtime: ${error.message}`
    };
  }

  return { ok: true, binaryPath };
}

function resolveTarget() {
  const platformMap = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows"
  };
  const archMap = {
    x64: "amd64",
    arm64: "arm64",
    arm: "armv7"
  };

  const goos = platformMap[process.platform];
  const goarch = archMap[process.arch];
  if (!goos || !goarch) {
    throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
  }
  if (goarch === "armv7" && process.platform !== "linux") {
    throw new Error(`unsupported platform: ${process.platform}/${process.arch}`);
  }

  const repo = process.env.YEELIGHT_HOME_REPO || officialRepo;
  const version = process.env.YEELIGHT_HOME_VERSION || `v${packageInfo.version}`;
  const extension = goos === "windows" ? "zip" : "tar.gz";
  const assetName = `yeelight-home-${goos}-${goarch}.${extension}`;
  const releasePath = version === "latest" ? "latest/download" : `download/${version}`;

  const sources = resolveReleaseSources(repo, releasePath);
  return {
    repo,
    version,
    goos,
    goarch,
    assetName,
    binaryName: goos === "windows" ? "yeelight-home.exe" : "yeelight-home",
    sources: sources.map((source) => ({
      ...source,
      assetUrl: `${source.baseUrl}/${assetName}`,
      checksumsUrl: `${source.baseUrl}/checksums.txt`
    }))
  };
}

function resolveReleaseSources(repo, releasePath) {
  const customBase = (process.env.YEELIGHT_HOME_DOWNLOAD_BASE_URL || "").replace(/\/+$/, "");
  if (customBase) return [{ name: "custom", baseUrl: customBase }];

  const sources = [{ name: "GitHub", baseUrl: `https://github.com/${repo}/releases/${releasePath}` }];
  if (repo === officialRepo) {
    sources.push(
      { name: "Gitee", baseUrl: `https://gitee.com/yeelight/yeelight-home/releases/${releasePath}` },
      { name: "GitCode", baseUrl: `https://api.gitcode.com/Yeelight/yeelight-home/releases/${releasePath}` }
    );
  }
  return sources;
}

function resolveBinaryPath(target) {
  const cacheRoot = process.env.YEELIGHT_HOME_NPM_CACHE_DIR || defaultCacheRoot();
  const repoKey = target.repo.replace(/[^A-Za-z0-9_.-]+/g, "-");
  return path.join(cacheRoot, repoKey, target.version, `${target.goos}-${target.goarch}`, target.binaryName);
}

function defaultCacheRoot() {
  if (process.platform === "win32" && process.env.LOCALAPPDATA) {
    return path.join(process.env.LOCALAPPDATA, "YeelightHome", "npm");
  }
  if (process.platform === "darwin") {
    return path.join(os.homedir(), "Library", "Caches", "yeelight-home", "npm");
  }
  return path.join(os.homedir(), ".cache", "yeelight-home", "npm");
}

function installRuntime(target, binaryPath) {
  const workDir = fs.mkdtempSync(path.join(os.tmpdir(), "yeelight-home-npm-"));
  try {
    const assetPath = path.join(workDir, target.assetName);
    const checksumsPath = path.join(workDir, "checksums.txt");
    downloadVerifiedRelease(target, assetPath, checksumsPath);

    fs.mkdirSync(path.dirname(binaryPath), { recursive: true });
    if (target.goos === "windows") {
      extractZip(assetPath, target.binaryName, binaryPath, workDir);
    } else {
      extractTarGz(assetPath, target.binaryName, binaryPath, workDir);
      fs.chmodSync(binaryPath, 0o755);
    }
  } finally {
    fs.rmSync(workDir, { recursive: true, force: true });
  }
}

function downloadVerifiedRelease(target, assetPath, checksumsPath) {
  const failures = [];
  for (const source of target.sources) {
    try {
      fs.rmSync(assetPath, { force: true });
      fs.rmSync(checksumsPath, { force: true });
      downloadFile(source.assetUrl, assetPath);
      downloadFile(source.checksumsUrl, checksumsPath);
      verifyChecksum(assetPath, checksumsPath, target.assetName);
      return source.name;
    } catch (error) {
      failures.push(`${source.name}: ${error.message}`);
    }
  }
  throw new Error(`all release sources failed (${failures.join("; ")})`);
}

function downloadFile(url, outputPath) {
  const timeoutMs = downloadTimeoutMs();
  const response = spawnSync(process.execPath, [
    "-e",
    `
      const fs = require("node:fs");
      const https = require("node:https");
      const url = process.argv[1];
      const out = process.argv[2];
      const timeout = Number(process.argv[3]);
      function get(current, redirects) {
        const request = https.get(current, res => {
          if ([301, 302, 303, 307, 308].includes(res.statusCode)) {
            if (redirects <= 0) throw new Error("too many redirects");
            res.resume();
            get(new URL(res.headers.location, current).toString(), redirects - 1);
            return;
          }
          if (res.statusCode !== 200) {
            console.error("download failed: " + res.statusCode + " " + current);
            process.exit(1);
          }
          const file = fs.createWriteStream(out, { mode: 0o600 });
          res.pipe(file);
          file.on("finish", () => file.close(() => process.exit(0)));
          file.on("error", err => {
            console.error(err.message);
            process.exit(1);
          });
        }).on("error", err => {
          console.error(err.message);
          process.exit(1);
        });
        request.setTimeout(timeout, () => request.destroy(new Error("download timed out")));
      }
      get(url, 5);
    `,
    url,
    outputPath,
    String(timeoutMs)
  ], {
    stdio: ["ignore", "ignore", "pipe"],
    encoding: "utf8",
    timeout: (timeoutMs * 6) + 5000
  });

  if (response.status !== 0) {
    throw new Error((response.stderr || "").trim() || `download failed: ${url}`);
  }
}

function downloadTimeoutMs() {
  const value = Number(process.env.YEELIGHT_HOME_DOWNLOAD_TIMEOUT_MS || 15000);
  if (!Number.isInteger(value) || value < 1000 || value > 120000) {
    throw new Error("YEELIGHT_HOME_DOWNLOAD_TIMEOUT_MS must be an integer between 1000 and 120000");
  }
  return value;
}

function verifyChecksum(assetPath, checksumsPath, assetName) {
  const checksums = fs.readFileSync(checksumsPath, "utf8").split(/\r?\n/);
  let expected = "";
  for (const line of checksums) {
    const parts = line.trim().split(/\s+/);
    if (parts.length >= 2 && parts[1] === assetName) {
      expected = parts[0];
      break;
    }
  }
  if (!expected) {
    throw new Error(`checksum not found for ${assetName}`);
  }

  const actual = crypto.createHash("sha256").update(fs.readFileSync(assetPath)).digest("hex");
  if (actual !== expected) {
    throw new Error(`checksum mismatch for ${assetName}`);
  }
}

function extractTarGz(assetPath, binaryName, binaryPath, workDir) {
  const extractDir = path.join(workDir, "tar");
  fs.mkdirSync(extractDir, { recursive: true });
  const result = spawnSync("tar", ["-xzf", assetPath, "-C", extractDir], {
    stdio: ["ignore", "ignore", "pipe"],
    encoding: "utf8"
  });
  if (result.status !== 0) {
    throw new Error((result.stderr || "").trim() || "tar extraction failed");
  }
  const extractedBinary = path.join(extractDir, binaryName);
  if (!fs.existsSync(extractedBinary)) {
    throw new Error(`${binaryName} not found in ${path.basename(assetPath)}`);
  }
  fs.copyFileSync(extractedBinary, binaryPath);
}

function extractZip(assetPath, binaryName, binaryPath, workDir) {
  const extractDir = path.join(workDir, "zip");
  fs.mkdirSync(extractDir, { recursive: true });
  const script = `Expand-Archive -LiteralPath '${assetPath.replace(/'/g, "''")}' -DestinationPath '${extractDir.replace(/'/g, "''")}' -Force`;
  const result = spawnSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script], {
    stdio: ["ignore", "ignore", "pipe"],
    encoding: "utf8"
  });
  if (result.status !== 0) {
    throw new Error((result.stderr || "").trim() || "PowerShell Expand-Archive failed");
  }
  const extractedBinary = path.join(extractDir, binaryName);
  if (!fs.existsSync(extractedBinary)) {
    throw new Error(`${binaryName} not found in ${path.basename(assetPath)}`);
  }
  fs.copyFileSync(extractedBinary, binaryPath);
}

module.exports = {
  ensureRuntimeBinary,
  resolveTarget
};
