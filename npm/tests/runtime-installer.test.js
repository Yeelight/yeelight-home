"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");

const { resolveTarget } = require("../lib/runtime-installer");

test("official releases fall back from GitHub to Gitee and GitCode", () => {
  withEnvironment({
    YEELIGHT_HOME_REPO: undefined,
    YEELIGHT_HOME_DOWNLOAD_BASE_URL: undefined
  }, () => {
    const target = resolveTarget();
    assert.deepEqual(target.sources.map(({ name }) => name), ["GitHub", "Gitee", "GitCode"]);
    assert.match(target.sources[0].assetUrl, /^https:\/\/github\.com\/Yeelight\/yeelight-home\/releases\/download\/v/);
    assert.match(target.sources[1].assetUrl, /^https:\/\/gitee\.com\/yeelight\/yeelight-home\/releases\/download\/v/);
    assert.match(target.sources[2].assetUrl, /^https:\/\/api\.gitcode\.com\/Yeelight\/yeelight-home\/releases\/download\/v/);
    for (const source of target.sources) assert.match(source.checksumsUrl, /\/checksums\.txt$/);
  });
});

test("custom repositories do not silently fall back to official mirrors", () => {
  withEnvironment({
    YEELIGHT_HOME_REPO: "example/private-fork",
    YEELIGHT_HOME_DOWNLOAD_BASE_URL: undefined
  }, () => {
    const target = resolveTarget();
    assert.deepEqual(target.sources.map(({ name }) => name), ["GitHub"]);
    assert.match(target.sources[0].assetUrl, /github\.com\/example\/private-fork/);
  });
});

test("an explicit download base is the only selected source", () => {
  withEnvironment({ YEELIGHT_HOME_DOWNLOAD_BASE_URL: "https://mirror.example/releases/v1/" }, () => {
    const target = resolveTarget();
    assert.deepEqual(target.sources.map(({ name }) => name), ["custom"]);
    assert.match(target.sources[0].assetUrl, /^https:\/\/mirror\.example\/releases\/v1\//);
  });
});

function withEnvironment(values, callback) {
  const previous = {};
  for (const [name, value] of Object.entries(values)) {
    previous[name] = process.env[name];
    if (value === undefined) delete process.env[name];
    else process.env[name] = value;
  }
  try {
    callback();
  } finally {
    for (const [name, value] of Object.entries(previous)) {
      if (value === undefined) delete process.env[name];
      else process.env[name] = value;
    }
  }
}
