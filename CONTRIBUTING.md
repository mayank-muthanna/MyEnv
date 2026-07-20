# Community Guidelines

Thank you for helping improve **myenv**. This project protects environment
configuration by keeping code, dotenv values, and `.myenv.yaml` contracts in
sync. Contributions should make that contract safer, clearer, and easier to
use.

## Before contributing

- Read the [README](README.md) for setup and normal usage.
- Read [docs/architecture.md](docs/architecture.md) before changing core
  behavior.
- Read [docs/usage.md](docs/usage.md) before changing CLI output, schema rules,
  encryption, ignore policy, or CI behavior.
- Search existing issues and pull requests before opening a new one.
- Keep one issue or pull request focused on one problem.

## Reporting bugs

A useful bug report includes:

1. myenv version and installation method (`npm`, source build, or binary).
2. Operating system and shell.
3. Exact command run.
4. Expected behavior and actual behavior.
5. Minimal safe reproduction: `.myenv.yaml`, sanitized dotenv values, and a
   small source example when relevant.
6. Full error output, with all secrets removed.

Never paste API keys, passwords, tokens, encrypted dotenv keys, real `.env`
files, or GitHub Actions secrets into issues, pull requests, logs, or screenshots.
Replace values with obvious placeholders such as `sk_test_replace_me`.

## Feature requests

Explain the problem before proposing implementation. Good requests describe:

- Who needs the feature.
- Current workflow and failure mode.
- Desired behavior and an example command or schema.
- Alternatives considered.
- Whether the change affects local development, CI, or both.

myenv favors a small primitive set and predictable CLI behavior. Avoid adding
hosted services, SDKs, dashboards, or broad language parsing without a clear,
validated use case.

## Pull requests

1. Fork repository and create focused branch.
2. Make smallest change that fixes root problem.
3. Add or update tests when behavior changes.
4. Update README or relevant `docs/` page when user-facing behavior changes.
5. Run checks locally.
6. Open pull request with problem, solution, test evidence, and any tradeoffs.

Pull request descriptions should answer:

```text
Problem:
What changed:
How tested:
Docs changed:
Security or compatibility impact:
```

Do not mix refactors, formatting sweeps, dependency upgrades, and feature work
in one pull request unless they are required for the same fix.

## Local checks

Run Go tests from repository root:

```powershell
$env:GOCACHE = (Join-Path $PWD '.gocache')
go test -count=1 -p 1 ./...
```

When changing NPM launcher packages:

```powershell
Push-Location npm
npm.cmd run build:binaries
npm.cmd run verify
Pop-Location
```

When changing scanner, validation, encryption, or CI behavior, also exercise
relevant demos under `testdata/demo/` and `testdata/demoCI/`.

## Code and documentation expectations

- Keep Go code small, explicit, and testable.
- Keep validation functions free of printing or process exits where possible.
- Preserve existing exit-code behavior: errors fail; warnings do not.
- Never print plaintext dotenv values, encryption keys, or detected secret values.
- Keep output usable in terminals and JSON mode stable for CI.
- Use clear, plain-language diagnostics and hints.
- Document new flags, schema fields, diagnostics, or security behavior.
- Do not add a dependency unless standard library or current project packages
  cannot solve the problem clearly.

## Security disclosures

Do not create a public issue for a suspected secret leak, encryption weakness,
or security vulnerability.

Remove sensitive data from any reproduction. Use repository private security
advisories if enabled; otherwise contact the repository owner privately through
GitHub. Include impact, affected version, reproduction steps, and a suggested
fix only when safe to share.

## Community behavior

Be respectful, constructive, and specific. Challenge ideas, not people.
Assume good intent, welcome beginners, and help keep discussion focused on the
project. Harassment, discrimination, threats, or sharing another person's
sensitive information are not acceptable.

Maintainers may edit, close, or remove content that violates these guidelines
or puts users, secrets, or project security at risk.

## License and ownership

Only submit work you have the right to contribute. By submitting a contribution,
you allow project maintainers to use, modify, and distribute it under the
repository's license when one is added or updated.
