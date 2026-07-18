"use strict";

const packages = {
  "win32-x64": "@mayank_muthanna/myenv-win32-x64",
  "darwin-arm64": "@mayank_muthanna/myenv-darwin-arm64",
  "darwin-x64": "@mayank_muthanna/myenv-darwin-x64",
  "linux-x64": "@mayank_muthanna/myenv-linux-x64",
  "linux-arm64": "@mayank_muthanna/myenv-linux-arm64"
};

function platformPackage(platform = process.platform, architecture = process.arch) {
  const packageName = packages[`${platform}-${architecture}`];
  if (!packageName) {
    throw new Error(`unsupported platform ${platform}-${architecture}`);
  }
  return packageName;
}

function binaryName(platform = process.platform) {
  return platform === "win32" ? "myenv.exe" : "myenv";
}

module.exports = { binaryName, platformPackage };