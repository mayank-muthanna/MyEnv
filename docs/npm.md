# NPM Package Scaffold

## What this adds

`myenv` remains one Go CLI. The `npm/` workspace packages compiled Go binaries
behind a small Node launcher so JavaScript users can install and run it like
other NPM CLIs:

```sh
npm install -g @mayank_muthanna/myenv
myenv help
myenv scan
myenv ci
```

`npm` creates the global `myenv` executable from the launcher's `bin` field.
After global install, users run `myenv`, not `npm myenv`.

Users can also run it without global install:

```sh
npx @mayank_muthanna/myenv scan
```

## Package layout

```text
npm/
  packages/
    cli/                 # Node launcher and `myenv` bin mapping
    cli-win32-x64/       # Windows x64 myenv.exe
    cli-darwin-arm64/    # macOS Apple Silicon binary
    cli-darwin-x64/      # macOS Intel binary
    cli-linux-x64/       # Linux x64 binary
    cli-linux-arm64/     # Linux ARM64 binary
  scripts/
    build-platform-binaries.mjs
    verify-packages.mjs
```

`@mayank_muthanna/myenv` has every platform package in `optionalDependencies`. NPM selects
only the package matching the user's operating system and CPU. Launcher then
starts that binary with all original command-line arguments and exit codes.

No Go toolchain is needed by end users. No `postinstall` downloader exists.
This avoids downloading unverified executables during installation.

## Supported platforms

| NPM package | Go target | Command binary |
| --- | --- | --- |
| `@mayank_muthanna/myenv-win32-x64` | `windows/amd64` | `myenv.exe` |
| `@mayank_muthanna/myenv-darwin-arm64` | `darwin/arm64` | `myenv` |
| `@mayank_muthanna/myenv-darwin-x64` | `darwin/amd64` | `myenv` |
| `@mayank_muthanna/myenv-linux-x64` | `linux/amd64` | `myenv` |
| `@mayank_muthanna/myenv-linux-arm64` | `linux/arm64` | `myenv` |

Unsupported systems show a clear launcher error. Add another platform package
and target entry when support is needed.

## Choose package namespace first

Scaffold uses `@myenv/*` names. Before first publish, replace every `@myenv/`
reference below `npm/` with a scope you own, for example:

```text
@your-npm-user/myenv
@your-npm-user/myenv-win32-x64
```

Use same version in all six `package.json` files. NPM scoped public packages
need `publishConfig.access: public`, already present in every manifest.

Do not publish under `@myenv` unless you own that NPM organization.

## Build packages

Requirements for release machine:

- Node.js 18 or newer
- NPM
- Go toolchain matching `go.mod`

From repository root:

```powershell
Push-Location npm
npm.cmd run build:binaries
npm.cmd run verify
Pop-Location
```

`build:binaries` uses Go cross compilation with `CGO_ENABLED=0`. Generated
files are written only to each platform package's `bin/` folder and are ignored
by Git.

## Test before publish

First build, verify, and produce package tarballs:

```powershell
Push-Location npm
npm.cmd run pack:all
Pop-Location
```

This verifies package manifests, platform binaries, launcher mapping tests, and
archive contents. On Windows, confirm packaged binary starts:

```powershell
.\npm\packages\cli-win32-x64\bin\myenv.exe help
```

The launcher depends on NPM optional platform packages. Before they exist in
the NPM registry, a clean `npm install` of launcher tarball may try to resolve
those unpublished packages. This is normal. For a true clean-install test,
publish a prerelease of all platform packages first, then launcher, and test:

```powershell
New-Item -ItemType Directory -Force D:\tmp\myenv-npm-test | Out-Null
Set-Location D:\tmp\myenv-npm-test
npm.cmd init -y
npm.cmd install @mayank_muthanna/myenv@0.1.1-beta.1
npx.cmd myenv help
```

Also verify real CI behavior using your encrypted fixture. Set
`MYENV_DECRYPT_KEY` first:

```powershell
npx.cmd myenv ci --root D:\Projects\myenv\testdata\demoCI\pass `
  --schema D:\Projects\myenv\testdata\demoCI\pass\.myenv.yaml
```
## Publish sequence

1. Confirm working tree and package versions.
2. Log in: `npm.cmd login`.
3. Build and verify packages.
4. Publish platform packages first.
5. Publish launcher last.
6. Test from a clean temporary directory using `npx` and global install.

Example commands from `npm/` after replacing names with your scope:

```powershell
npm.cmd publish --workspace @mayank_muthanna/myenv-win32-x64 --access public
npm.cmd publish --workspace @mayank_muthanna/myenv-darwin-arm64 --access public
npm.cmd publish --workspace @mayank_muthanna/myenv-darwin-x64 --access public
npm.cmd publish --workspace @mayank_muthanna/myenv-linux-x64 --access public
npm.cmd publish --workspace @mayank_muthanna/myenv-linux-arm64 --access public
npm.cmd publish --workspace @mayank_muthanna/myenv --access public
```

NPM versions are immutable. For every later release, bump all six package
versions together, rebuild binaries from tagged Go source, then publish again.
Use a prerelease such as `0.1.1-beta.1` for first external testing.

## Global command troubleshooting

After `npm install -g`, run:

```powershell
myenv help
```

If PowerShell cannot find `myenv`, inspect global executable directory:

```powershell
npm.cmd prefix -g
```

On Windows, add the resulting NPM global bin directory to your user `PATH`,
open a new terminal, then run `myenv help` again.

## Security and release rules

- Build binaries only from reviewed, tagged source.
- Run Go tests and `npm run verify` before publishing.
- Keep decryption keys in GitHub secrets/password managers, never in NPM
  packages or release archives.
- Do not replace platform packages with a postinstall download script.
- Publish platform packages before launcher, otherwise new installs may not find
  matching optional dependency.
- Pin action versions and release from trusted CI when publishing becomes
  routine.