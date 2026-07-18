# Using myenv

`myenv` is a Go command-line tool for keeping environment files, a central
schema, and JavaScript/TypeScript code in agreement.

It answers four questions before deployment:

1. Are values in `.env` valid?
2. Does every variable used by code exist in the schema?
3. Does the schema still contain variables code no longer uses?
4. Did a likely real credential reach a tracked `.env*` file?

## Install and build

### Run from source

Install a supported Go toolchain, clone this repository, then run commands
directly from its root:

```powershell
go run ./cmd/myenv --help
go run ./cmd/myenv validate
```

### Build a binary

Build a Windows executable:

```powershell
New-Item -ItemType Directory -Force bin
go build -o bin/myenv.exe ./cmd/myenv
.\bin\myenv.exe --help
```

Build on macOS or Linux:

```sh
mkdir -p bin
go build -o bin/myenv ./cmd/myenv
./bin/myenv --help
```

After building, replace `go run ./cmd/myenv` in all examples with the binary
path, such as `.\bin\myenv.exe` on Windows.

## Quick start

Start with an existing `.env` file:

```env
PORT=3000
DEBUG=false
STRIPE_SECRET_KEY=sk_test_Abcdefghijklmnopqrstuvwxyz12
```

Generate a starter schema:

```powershell
go run ./cmd/myenv infer
```

This creates `.myenv.yaml`. `infer` copies variable names and infers basic
types, but never copies dotenv values into the schema. Names containing words
such as `SECRET`, `TOKEN`, `PASSWORD`, `PRIVATE`, or ending in `_KEY` are marked
`secret: true` for review.

Review the generated schema, add requirements, then validate:

```powershell
go run ./cmd/myenv validate
go run ./cmd/myenv scan
```

## Encrypt and decrypt dotenv files

`encrypt` reads dotenv bytes exactly as they exist (including comments, ordering,
and whitespace), compresses them with gzip, then encrypts them with AES-256-GCM.
The encrypted payload is written as the final `encryptedEnv` block in the same
`.myenv.yaml` file. The encryption key is **never** written to that file.

```powershell
# Generates a new random key. Save the [KEY] value in a password manager.
myenv encrypt --env .env.local

# Restores exact original dotenv bytes to a safe new path.
myenv decrypt --key <saved-key> --output .env.local.restored
```

To supply your own key, provide a base64url-encoded, 32-byte key to both
commands. This is a raw AES-256 key, not a password or phrase:

```powershell
myenv encrypt --env .env.local --key <your-32-byte-base64url-key>
myenv decrypt --key <your-32-byte-base64url-key> --output .env.local.restored
```

`decrypt` refuses to replace an existing file. Use a different `--output`, or
add `--force` only when replacement is intentional. Losing the key means the
payload cannot be recovered.
## Schema file

The default schema path is `.myenv.yaml`. Its top-level keys are environment
variable names. Each key has a rule object.

```yaml
PORT:
  type: int
  required: true
  range: { min: 1, max: 65535 }

DEBUG:
  type: bool
  default: "false"

STRIPE_SECRET_KEY:
  type: string
  required: true
  pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$'
  secret: true
```

### Rule reference

| Rule | Values | Meaning |
| --- | --- | --- |
| `type` | `string`, `int`, `float`, `bool` | Value conversion to require. Defaults to `string`. |
| `required` | `true` or `false` | Key must be present unless it has a `default`. |
| `default` | string | Value used for validation when key is absent from `.env`. Quote booleans and numbers in YAML. |
| `pattern` | regular expression | Full value requirement defined by regex matching. Use `^` and `$` when entire value must match. |
| `range` | `{ min, max }` | Inclusive numeric limits. Only valid with `int` or `float`. |
| `secret` | `true` or `false` | Marks sensitive value. Prevents use via `import.meta.env`. |

`pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$'` means a Stripe key must start
with `sk_live_` or `sk_test_`, then contain at least 24 letters or digits, and
contain nothing else.

## Commands

### `myenv infer`

Create a starter schema from a dotenv file:

```powershell
go run ./cmd/myenv infer --env .env --output .myenv.yaml
```

Use this once for a new repository or after intentionally adding several
variables. It overwrites the output file, so review or commit current schema
first.

infer also writes a commented ignorePaths, ignoreRules, ignoreCode, and
ignoreUnused template at the top of the generated file. Uncomment only entries
you need.

### `myenv validate`

Validate dotenv values against schema:

```powershell
go run ./cmd/myenv validate
go run ./cmd/myenv validate --schema config/myenv.yaml --env .env.production
go run ./cmd/myenv validate --format json
```

Validation reports errors and exits `1` when:

- A required key is missing.
- A value cannot convert to its declared type.
- A numeric value is outside `range`.
- A value does not match `pattern`.
- `.env` contains a variable absent from the schema.

Text output uses color and compact labels: red `[ERROR]` / `[FAIL]`, yellow
`[WARN]`, green `[PASS]`, blue rule names and `[HINT]`, and gray locations and
separators. JSON output remains uncolored.

Example failure:

```text
MYENV VALIDATE  1 diagnostic
------------------------------------------------------------
[ERROR] invalid-value  PORT must be at most 65535
------------------------------------------------------------
[FAIL] 1 errors, 0 warnings. [HINT] Run "myenv help" for commands and flags.
```

Fix source value or schema rule, then rerun the command.

### `myenv scan`

Cross-reference static source usage, `.env`, and schema:

```powershell
go run ./cmd/myenv scan
go run ./cmd/myenv scan --root . --schema .myenv.yaml --env .env
go run ./cmd/myenv scan --format json
```

Recognized static access forms:

```ts
process.env.PORT
process.env["STRIPE_SECRET_KEY"]
import.meta.env.VITE_API_URL
```

`scan` checks `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.cjs` files. It skips
`.git`, `node_modules`, `vendor`, `dist`, `build`, `.next`, and `coverage`.

Results:

- **Error:** a code variable is missing from `.env`, `.myenv.yaml`, or both.
- **Error:** `.env` contains a variable absent from `.myenv.yaml`.
- **Warning:** a variable present in both config files is unused by static source.

### Examples:

- **Error:** code uses `process.env.NEW_FLAG`, but `NEW_FLAG` is missing from
  `.myenv.yaml`.
- **Warning:** schema contains `OLD_FLAG`, but no static source reference uses
  it.
- **Error:** a `secret: true` schema key is read using `import.meta.env`. Build
  tools can expose these values to browser code.
- **Warning:** dynamic access like `process.env[key]` cannot be proven against
  schema statically.

Dynamic access does not create a false schema declaration. Prefer a direct
static property access when possible.

## Ignore policy

Keep scan policy inside `.myenv.yaml`. It affects scan diagnostics only; it does
not delete or modify source files.

```yaml
# .myenv.yaml
ignorePaths:
  - .nuxt/              # Skip generated Nuxt output.
  - scripts/fixtures/   # Skip a local fixture folder.
ignoreRules:
  - dynamic-env-access  # Hide this diagnostic rule everywhere.

# External/runtime provider variables. Ignore only code-reference findings.
ignoreCode:
  - BETTER_AUTH_SECRET
  - GOOGLE_CLIENT_*
  - HOQAN_FCM_DRY_RUN
  - HOQAN_PLATFORM_SUPERADMIN_SECRET_HASH
  - NODE_ENV
  - NITRO_*
  - NUXT_VITE_NODE_OPTIONS

# Config is intentional but no source file reads it.
ignoreUnused:
  - CONVEX_SELF_HOSTED_ADMIN_KEY
  - HOQAN_BOT_DEV_RUNTIME_TOKEN

PORT:
  type: int
  default: 3000
```

`ignoreCode` suppresses code-reference errors only: missing code declarations
and client-secret exposure for matching keys. It does not hide bad `.env`
values or schema validation errors.

`ignoreUnused` suppresses only `[unused-config-env]` for matching keys. Use it
for deployment/provider settings stored outside application source, such as
Convex-managed settings.

`ignorePaths` accepts root-relative folders/files plus `*` and `**` patterns.
`ignoreRules` matches diagnostic IDs. Source files matched by repository
`.gitignore` are skipped automatically. If Git is unavailable or scan root is
not a Git repository, myenv continues scanning and relies on `ignorePaths`.
## Secret leak checks

During `scan`, myenv asks Git for tracked `.env*` files. It checks those files
for high-signal patterns: Stripe keys, AWS access keys, Slack tokens, and
private-key headers.

It does not scan ignored local `.env` files for committed-leak findings. This
avoids flagging expected developer secrets that are not in version control.
Matched secret values are never printed.

Use placeholders in tracked example files:

```env
# Good: placeholder
STRIPE_SECRET_KEY=sk_test_replace_me

# Bad: realistic credential pattern in a tracked .env.example file
STRIPE_SECRET_KEY=sk_live_123456789012345678901234
```

## CI and GitHub Actions

Use `myenv ci` for automation. It has two safe modes:

1. **Always:** scans static code against `.myenv.yaml`. Missing schema
   declarations fail; unused declarations and dynamic access warn.
2. **Only when a key exists:** reads `MYENV_DECRYPT_KEY`, decrypts the committed
   `encryptedEnv` payload **in memory**, then validates its dotenv values against
   `.myenv.yaml`. It never creates or prints a plaintext dotenv file.

```sh
# No key required. Code versus schema only.
myenv ci --root . --schema .myenv.yaml

# Key available. Adds encrypted dotenv-value validation.
MYENV_DECRYPT_KEY=<saved-key> myenv ci --root . --schema .myenv.yaml
```

The included composite action runs this command. Give it only read permission:

```yaml
name: Environment contract
on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  myenv:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: owner/myenv@<pinned-release-or-commit>
        with:
          schema: .myenv.yaml
          root: .
        env:
          MYENV_DECRYPT_KEY: ${{ secrets.MYENV_DECRYPT_KEY }}
```

If `MYENV_DECRYPT_KEY` is unavailable, including on pull requests from forks,
the action still completes code/schema checks. GitHub does not expose repository
secrets to forked pull requests; this is intentional.

**Security rules:** use a pinned trusted myenv release/commit; never run project
scripts, package installs, or code built from an untrusted pull request in the
same job holding `MYENV_DECRYPT_KEY`; never use `pull_request_target` to check
out untrusted PR code. The command does not write a decrypted file, so no cleanup
step is needed. GitHub hosted runners are also discarded after the job.
## Test current implementation

Run all unit tests:

```powershell
go test ./...
```

Run intentionally broken demo fixtures:

```powershell
go run ./cmd/myenv validate `
  --schema testdata/demo/.myenv.yaml `
  --env testdata/demo/.env

go run ./cmd/myenv scan `
  --schema testdata/demo/.myenv.yaml `
  --env testdata/demo/.env `r`n  --root testdata/demo
```

Failures are expected. Demo includes invalid `PORT`, invalid Stripe format,
an undeclared `NEW_FEATURE_FLAG` source access, an unused schema key, and a
secret exposed through `import.meta.env`. Fix these conditions and rerun for a
clean result.

## Limits

Current scanner is static and JS/TS-only. It cannot resolve dynamic variable
names or prove runtime-only configuration paths. It is not a secret manager,
encryption layer, Git-history scanner, or runtime tracing system.

