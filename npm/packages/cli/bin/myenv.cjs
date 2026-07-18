#!/usr/bin/env node
"use strict";

const { spawnSync } = require("node:child_process");
const { binaryName, platformPackage } = require("../lib/platform-package.cjs");

let executable;
try {
  executable = require.resolve(`${platformPackage()}/bin/${binaryName()}`);
} catch {
  console.error(`myenv: no binary package exists for ${process.platform}-${process.arch}.`);
  console.error("Reinstall @mayank_muthanna/myenv, or use a supported platform.");
  process.exit(1);
}

const result = spawnSync(executable, process.argv.slice(2), { stdio: "inherit" });
if (result.error) {
  console.error(`myenv: could not start binary: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);