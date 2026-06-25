#!/usr/bin/env node

"use strict";

const { spawnSync } = require("node:child_process");
const { ensureRuntimeBinary } = require("../npm/lib/runtime-installer");

const result = ensureRuntimeBinary();
if (!result.ok) {
  console.error(result.message);
  process.exit(result.code || 1);
}

const child = spawnSync(result.binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  windowsHide: false
});

if (child.error) {
  console.error(child.error.message);
  process.exit(1);
}

process.exit(child.status === null ? 1 : child.status);
