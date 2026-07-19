# myenv

**myenv** keeps environment configuration honest.

It checks three places that normally drift apart:

1. **Code** — JavaScript and TypeScript references such as `process.env.PORT`.
2. **Schema** — one committed `.myenv.yaml` contract.
3. **Values** — local `.env` files or an encrypted dotenv payload committed with the schema.

When code reads a variable nobody declared, when a value is malformed, or when
old configuration is no longer used, myenv reports it before deployment.

Built for the **OpenAI Build Week** Developer Tools track. This project was
implemented iteratively with Codex: architecture decisions were recorded first,
then Codex-assisted implementation added the Go validation engine, scanner,
interactive CLI, encryption flow, tests, GitHub Action, NPM launcher, and demo
fixtures. See [architecture](docs/architecture.md) for the design and
[usage](docs/usage.md) for deeper command details.

## Install

### Recommended: NPM

Install globally once, then use plain `myenv` everywhere:

```powershell
npm install -g @mayank_muthanna/myenv
myenv help
```

macOS/Linux:

```sh
npm install -g @mayank_muthanna/myenv
myenv help
```

NPM installs only the matching prebuilt binary for Windows, macOS, or Linux. No
Go toolchain is needed for this path.

### Build from source

Requires Go version specified in `go.mod`.

```powershell
New-Item -ItemType Directory -Force bin
$env:GOCACHE = (Join-Path $PWD '.gocache')
go build -o bin/myenv.exe ./cmd/myenv
.\bin\myenv.exe help
```

macOS/Linux:

```sh
mkdir -p bin
go build -o bin/myenv ./cmd/myenv
./bin/myenv help
```

## Quick start

Start inside a project containing a dotenv file:

```env
PORT=3000
DEBUG=false
API_URL=https://api.example.com
STRIPE_SECRET_KEY=sk_test_replace_me
```

Generate a starter contract:

```sh
myenv infer --env .env
```

This creates `.myenv.yaml`. Review it, then run:

```sh
myenv validate
myenv scan
```

Normal flow:

```text
.env changes ? myenv infer ? review .myenv.yaml ? validate ? scan ? commit schema
```

If `.myenv.yaml` already exists, `infer` opens a keyboard menu:

- **Override** — replace schema completely.
- **Sync** — add/remove dotenv keys while preserving rule settings already made.
- **Skip** — leave schema unchanged.

## Schema example

```yaml
PORT:
  type: int
  required: true
  range: { min: 1, max: 65535 }

DEBUG:
  type: bool
  default: "false"

API_URL:
  type: string
  required: true
  pattern: '^https://[A-Za-z0-9.-]+$'

STRIPE_SECRET_KEY:
  type: string
  required: true
  pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$'
  secret: true
```

Supported primitive types: `string`, `int`, `float`, `bool`.

Rules: `required`, `default`, `pattern`, `range`, and `secret`. Use regex
`pattern` only where a normal type check is not enough. When a pattern fails,
myenv gives plain-language hints for recognizable prefixes, allowed choices,
character sets, and lengths.

## What myenv checks

| Command | Purpose |
| --- | --- |
| `myenv infer` | Create or safely sync `.myenv.yaml` from dotenv variable names. |
| `myenv validate` | Check dotenv values against schema types, requirements, patterns, and ranges. |
| `myenv scan` | Compare code, dotenv, and schema; find drift, unused config, dynamic access, client secret exposure, and tracked dotenv leaks. |
| `myenv ci` | CI-safe code/schema check; also validates encrypted values when key is available. |
| `myenv encrypt` | Gzip-compress and AES-256-GCM encrypt a dotenv file into `.myenv.yaml`. |
| `myenv decrypt` | Restore an encrypted dotenv payload using its saved key. |

Run `myenv help` for command overview, or `myenv help scan` for flags and
examples.

## Scan results

`myenv scan` understands static JavaScript/TypeScript patterns:

```ts
process.env.PORT
process.env["STRIPE_SECRET_KEY"]
import.meta.env.VITE_API_URL
```

It reports:

- **Error** — code variable absent from schema or dotenv.
- **Error** — dotenv variable absent from schema.
- **Error** — `secret: true` variable used through browser-facing `import.meta.env`.
- **Warning** — declared config no static code path reads.
- **Warning** — dynamic environment access cannot be proven statically.
- **Warning** — likely real credential in a Git-tracked `.env*` file.

Add intentional exceptions inside `.myenv.yaml` using `ignorePaths`,
`ignoreRules`, `ignoreCode`, and `ignoreUnused`. Generated schemas include
commented examples. Full policy guide: [docs/usage.md](docs/usage.md#ignore-policy).

## Encrypt committed dotenv values

Real dotenv files normally stay out of Git. If a team needs CI to validate
actual values, encrypt them first:

```sh
myenv encrypt --env .env.local
```

This writes an `encryptedEnv` payload at bottom of `.myenv.yaml` and prints a
random key once. Store that key in a password manager and GitHub Actions secret
named `MYENV_DECRYPT_KEY`. The key is never stored in schema or Git.

Decrypt locally only when needed:

```sh
myenv decrypt --key <saved-key> --output .env.local.restored
```

`myenv ci` decrypts **in memory**. It never writes a plaintext dotenv file in
CI and never prints dotenv values or key.

## GitHub Actions

`action.yml` runs `myenv ci`.

- Without `MYENV_DECRYPT_KEY`: code ? schema checks only.
- With `MYENV_DECRYPT_KEY` and `encryptedEnv`: adds encrypted dotenv ? schema
  validation.

Example local action workflow:

```yaml
name: myenv
on: [pull_request]

permissions:
  contents: read

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: ./
        env:
          MYENV_DECRYPT_KEY: ${{ secrets.MYENV_DECRYPT_KEY }}
```

Do not expose this secret to untrusted fork PR code. The action safely works
without it for forks. More security detail: [docs/usage.md](docs/usage.md#ci-and-github-actions).

## Try demos

### Local broken demo

This fixture intentionally fails. It shows invalid values, an undeclared
variable, unused config, and browser secret exposure:

```powershell
myenv validate --schema testdata/demo/.myenv.yaml --env testdata/demo/.env
myenv scan --schema testdata/demo/.myenv.yaml --env testdata/demo/.env --root testdata/demo
```

### GitHub Actions demo

`testdata/demoCI/` contains one passing fixture and one intentional failure.

1. Set repository Actions secret `MYENV_DECRYPT_KEY` using test key described
   in `testdata/demoCI/README.md`.
2. Commit `.github/workflows/myenv-ci-demo.yml`.
3. Open GitHub **Actions** ? **myenv CI demo** ? **Run workflow**.

Expected: `pass fixture` passes. `expected fail fixture` detects its deliberate
errors and the workflow confirms that failure happened.

## Development and tests

```powershell
$env:GOCACHE = (Join-Path $PWD '.gocache')
go test -count=1 -p 1 ./...
```

NPM wrapper release instructions: [docs/npm.md](docs/npm.md).

## Limits and roadmap

Current scanner is static and JavaScript/TypeScript focused. Dynamic variable
names cannot be fully verified. myenv is not a hosted secrets manager, Git
history scanner, or runtime tracer.

Next useful directions: Go/Python source scanners, Git-history secret checks,
CI release automation, more platform binaries, and optional organization-wide
policy packs.