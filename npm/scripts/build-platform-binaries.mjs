import { chmodSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const scriptDirectory = dirname(fileURLToPath(import.meta.url));
const npmRoot = join(scriptDirectory, "..");
const repositoryRoot = join(npmRoot, "..");
const targets = [
  { packageName: "cli-win32-x64", goos: "windows", goarch: "amd64", binary: "myenv.exe" },
  { packageName: "cli-darwin-arm64", goos: "darwin", goarch: "arm64", binary: "myenv" },
  { packageName: "cli-darwin-x64", goos: "darwin", goarch: "amd64", binary: "myenv" },
  { packageName: "cli-linux-x64", goos: "linux", goarch: "amd64", binary: "myenv" },
  { packageName: "cli-linux-arm64", goos: "linux", goarch: "arm64", binary: "myenv" }
];

for (const target of targets) {
  const output = join(npmRoot, "packages", target.packageName, "bin", target.binary);
  mkdirSync(dirname(output), { recursive: true });
  const result = spawnSync("go", ["build", "-trimpath", "-ldflags=-s -w", "-o", output, "./cmd/myenv"], {
    cwd: repositoryRoot,
    env: { ...process.env, CGO_ENABLED: "0", GOOS: target.goos, GOARCH: target.goarch },
    stdio: "inherit"
  });
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
  if (target.goos !== "windows") {
    chmodSync(output, 0o755);
  }
  console.log(`built ${target.goos}-${target.goarch}: ${output}`);
}