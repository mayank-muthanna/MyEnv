# myenv architecture

## Purpose

`myenv` is a schema-first Go CLI that validates environment configuration and
cross-references it with static JavaScript and TypeScript environment-variable
usage. Its primary value is detecting drift between code and configuration
before deployment.

## Product boundaries

- The canonical schema is `.myenv.yaml`; it also holds optional scan ignore policy. The default value file is `.env`.
- `validate` checks values against the schema.
- `scan` checks static JS/TS environment usage against the schema and scans
  tracked `.env*` files for likely committed credentials.
- `infer` generates a starter schema from `.env`, without copying values.
- Go scanning, Git-history scanning, encryption, hosted secrets, and runtime
  tracing are out of scope for this MVP.

## Commands

| Command | Responsibility | Exit status |
| --- | --- | --- |
| `myenv validate` | Load schema and dotenv file, apply type and rule validation. | `1` when any validation error exists. |
| `myenv scan` | Find static JS/TS env references, compare code, `.env`, and schema keys, and inspect tracked `.env*` files for leaks. | `1` when a hard diagnostic exists. |
| `myenv infer` | Load `.env`, infer conservative rule types, and write a starter schema. | `1` for input/output failures. |

All commands support text diagnostics; `validate` and `scan` also provide JSON
diagnostics for CI and the GitHub Action.

## Schema

The schema maps environment-variable names to rules. Rules compose a small set
of primitives:

```yaml
STRIPE_SECRET_KEY:
  type: string
  required: true
  pattern: '^sk_(live|test)_[A-Za-z0-9]{24,}$'
  secret: true

PORT:
  type: int
  default: '3000'
  range: { min: 1, max: 65535 }
```

Supported types are `string`, `int`, `float`, and `bool`. `required`,
`default`, `pattern`, `range`, and `secret` are the only v1 rule keys.
`pattern` covers prefix, suffix, and format constraints; `range` is valid only
for numeric types. Defaults participate in validation when a key is absent.

## Internal design

- `internal/schema` parses YAML, normalizes rules, validates schema shape, and
  compiles regex patterns once.
- `internal/validate` contains pure value and dotenv validation functions.
- `internal/scanner` walks source files and recognizes static
  `process.env.NAME`, `process.env["NAME"]`, and `import.meta.env.NAME`
  references with file and line locations. Dynamic accesses emit warnings.
- `internal/diff` calculates code-minus-schema and code-minus-dotenv errors, plus unused configuration
  warnings. A `secret: true` variable referenced via `import.meta.env` is an
  error because bundlers can expose it to browsers.
- `internal/ignore` applies schema-owned ignore policy by path, rule ID, code key, or unused config key.
- `internal/leaks` scans tracked `.env*` files with a small curated set of
  high-signal credential patterns and never includes matched values in output.
- `internal/diagnostic` defines a stable severity, rule ID, location, and
  message shape shared by text and JSON reporters.

## Scanning policy

Source scanning includes `.ts`, `.tsx`, `.js`, `.jsx`, `.mjs`, and `.cjs`.
It skips `.git`, `node_modules`, `vendor`, common build directories, custom ignore paths, and source files matched by `.gitignore`. The
scanner only claims static certainty: `process.env[key]` is reported as a
warning rather than guessed. A schema key unused in static source is a warning;
a source key absent from `.env` or the schema is an error. A key present in both config files but absent from static source is a warning.

Leak scanning uses `git ls-files` when available, so ignored local `.env` files
are not falsely treated as committed leaks. It scans tracked `.env*` files,
especially `.env.example`, for Stripe, AWS, Slack, and private-key signatures.

## CI integration

The GitHub Action is a thin wrapper around the CLI: it runs `validate` and
`scan --format json`, fails on CLI errors, writes a job summary, and comments a
short result on pull requests. It contains no duplicate validation logic.

## Verification

Tests cover schema normalization, each validation primitive, source patterns,
diff policies, redacted leak diagnostics, JSON output, and full CLI fixture
flows. `testdata/demo` provides intentionally broken and fixed examples for a
repeatable hackathon demonstration.
