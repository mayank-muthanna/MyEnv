# NPM distribution workspace

This directory turns myenv Go binaries into NPM packages:

- `@mayank_muthanna/myenv`: Node launcher that publishes `myenv` command.
- `@mayank_muthanna/myenv-<platform>`: one binary package for each supported platform.

Build artifacts are intentionally ignored. Run `npm run build:binaries` before
packing or publishing.