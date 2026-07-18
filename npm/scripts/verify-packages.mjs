import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const npmRoot = join(scriptDirectory, "..");
const packageDirectories = [
  "cli",
  "cli-win32-x64",
  "cli-darwin-arm64",
  "cli-darwin-x64",
  "cli-linux-x64",
  "cli-linux-arm64"
];
const manifests = packageDirectories.map((directory) => {
  const path = join(npmRoot, "packages", directory, "package.json");
  return { directory, ...JSON.parse(readFileSync(path, "utf8")) };
});
const versions = new Set(manifests.map((manifest) => manifest.version));
if (versions.size !== 1) {
  throw new Error("all NPM packages must have same version");
}
for (const manifest of manifests.slice(1)) {
  const binary = manifest.os[0] === "win32" ? "myenv.exe" : "myenv";
  const path = join(npmRoot, "packages", manifest.directory, "bin", binary);
  if (!existsSync(path)) {
    throw new Error(`missing binary for ${manifest.name}: run npm run build:binaries`);
  }
}
console.log(`verified ${manifests.length} packages at version ${manifests[0].version}`);