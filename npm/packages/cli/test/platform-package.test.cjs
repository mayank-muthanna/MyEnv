"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const { binaryName, platformPackage } = require("../lib/platform-package.cjs");

test("selects platform packages", () => {
  assert.equal(platformPackage("win32", "x64"), "@mayank_muthanna/myenv-win32-x64");
  assert.equal(platformPackage("darwin", "arm64"), "@mayank_muthanna/myenv-darwin-arm64");
  assert.equal(platformPackage("linux", "x64"), "@mayank_muthanna/myenv-linux-x64");
});

test("selects executable names", () => {
  assert.equal(binaryName("win32"), "myenv.exe");
  assert.equal(binaryName("linux"), "myenv");
});

test("rejects unsupported platforms", () => {
  assert.throws(() => platformPackage("freebsd", "x64"), /unsupported platform/);
});