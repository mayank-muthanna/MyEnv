# myenv

`myenv` prevents environment configuration drift by validating a schema and
cross-referencing it with JavaScript/TypeScript environment usage.

## Quick start

```sh
go run ./cmd/myenv infer --env .env
go run ./cmd/myenv validate
go run ./cmd/myenv scan
```

The schema file is `.myenv.yaml`. Supported rule keys are `type`, `required`,
`default`, `pattern`, `range`, and `secret`; types are `string`, `int`,
`float`, and `bool`.

Run the intentionally broken demo with:

```sh
go run ./cmd/myenv validate --schema testdata/demo/.myenv.yaml --env testdata/demo/.env
go run ./cmd/myenv scan --schema testdata/demo/.myenv.yaml --env testdata/demo/.env --root testdata/demo
```

Use `--format json` for CI-friendly diagnostics. The included `action.yml` is
a thin GitHub Action wrapper around the same CLI checks.

## GitHub Action

Use the local action from a workflow with pull-request write permission so it
can update its summary comment:

```yaml
permissions:
  contents: read
  pull-requests: write

steps:
  - uses: actions/checkout@v4
  - uses: ./.github/actions/myenv
```

When publishing the action from this repository, replace the local `uses`
value with its `owner/repository@ref` reference.

## Hackathon notes

This project was designed and implemented iteratively with Codex: the
architecture, pure validation layer, source scanner, diagnostics, tests, and
demo scenario are kept in the repository so the decisions are reviewable.

## Roadmap

Go source scanning, broader language parsers, Git-history scans, encrypted
dotenv files, hosted secret management, and runtime tracing are intentionally
out of scope for this MVP.

