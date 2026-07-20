# Security Policy

## Supported Versions

Security fixes are provided for the latest released minor version of myenv.
Pre-release, development, and older versions may not receive fixes.

| Version | Supported |
| --- | --- |
| 0.1.x | :white_check_mark: |
| < 0.1 | :x: |
| Unreleased source | Best effort only |

If you use the NPM package, update all myenv platform packages and launcher to
the latest matching release before reporting an issue.

## Reporting a Vulnerability

Please do **not** report security vulnerabilities through public GitHub issues,
pull requests, discussions, screenshots, or chat messages.

Use GitHub's private security reporting for this repository:

1. Open the repository on GitHub.
2. Open the **Security** tab.
3. Select **Report a vulnerability**.
4. Provide a private report.

If private reporting is not enabled, contact the repository owner privately on
GitHub and ask for a secure reporting channel. Do not include secrets in the
first message.

### Include in a report

- A clear description of impact.
- Affected myenv version and installation method.
- Reproduction steps using fake or sanitized values only.
- Relevant schema, command, and error output with secrets removed.
- Whether issue affects local CLI, GitHub Action, NPM launcher, encrypted
  dotenv payloads, or CI key handling.
- A suggested fix or mitigation, if you have one.

Never send real API keys, passwords, tokens, private keys, `.env` contents,
`MYENV_DECRYPT_KEY`, or encrypted production payloads.

## Response Process

Maintainers aim to:

1. Acknowledge a valid report within 7 days.
2. Share initial triage or request clarification within 14 days.
3. Work privately on a fix, mitigation, or rejection rationale.
4. Publish a fix and coordinated disclosure when users can update safely.

Timelines depend on severity, reproducibility, and maintainer availability.
Reports that expose active secrets may require immediate credential rotation by
the affected owner; myenv maintainers cannot revoke third-party credentials.

## Security Scope

High-priority reports include:

- Plaintext secret or encryption-key exposure by myenv.
- AES-GCM encryption, decryption, or integrity-check bypasses.
- GitHub Action behavior that exposes `MYENV_DECRYPT_KEY` or decrypted values.
- NPM launcher behavior that runs an unintended binary or command.
- Secret scanner behavior that prints sensitive matched content.
- Path handling that reads, writes, or executes outside expected project paths.

False positives in leak detection, normal schema validation failures, and
feature requests should use public GitHub issues instead.

## Disclosure

Please give maintainers reasonable time to investigate and ship a fix before
public disclosure. Once resolved, maintainers may publish an advisory,
acknowledgment, affected versions, mitigation, and upgrade instructions. Credit
is offered to reporters who want it.
