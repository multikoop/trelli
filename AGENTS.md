# AGENTS.md

## Purpose

Guidance for future development of the `trelli` CLI in this repository.

## Scope

- Primary tool: `trelli` Go CLI in `cmd/trelli/main.go`.
- Trello integration target for live verification: board `trelli.sandbox` (`XobnRsYv`) only.

## Engineering Guidelines

- Keep the binary dependency-light (standard library first).
- Preserve CLI stability: avoid breaking existing flags/subcommands without clear migration notes.
- Ensure `trelli -h` and `trelli --help` remain comprehensive and accurate.
- Support both human-readable and `--json` output for automation.

## API and Security

- Credentials come from `TRELLO_API_KEY` and `TRELLO_TOKEN` or flags.
- Never hardcode or print secrets.
- Never echo environment variables containing tokens in logs or CI output.
- Prefer least-destructive live tests; clean up test artifacts (archive temporary cards).

## Local Validation

Use local Go cache inside workspace when sandbox blocks system cache:

```bash
mkdir -p .cache/go-build
GOCACHE=$(pwd)/.cache/go-build go test ./...
GOCACHE=$(pwd)/.cache/go-build go build ./...
```

## Suggested Future Work

- Split `main.go` into packages (`internal/trello`, `internal/output`, `internal/cli`).
- Add unit tests for argument parsing and list-name resolution.
- Add integration tests gated by explicit env flag (e.g. `TRELLO_INTEGRATION=1`).
- Add automated tap formula update tooling once the tap repository is established.
