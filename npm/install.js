#!/usr/bin/env node

"use strict";

const { ensureRuntimeBinary } = require("./lib/runtime-installer");

if (process.env.YEELIGHT_HOME_NPM_SKIP_INSTALL === "1") {
  process.exit(0);
}

const result = ensureRuntimeBinary();
if (!result.ok) {
  console.error(result.message);
  process.exit(result.code || 1);
}

console.log(`yeelight-home runtime installed at ${result.binaryPath}`);
