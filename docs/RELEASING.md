---
summary: "Release checklist for trelli (GitHub release + Homebrew tap)"
---

# Releasing `trelli`

This follows the `gogcli` model: Git tag triggers GoReleaser assets, then update a separate Homebrew tap formula.

Use the helper scripts:

```sh
scripts/release.sh X.Y.Z
scripts/verify-release.sh X.Y.Z
```

## 0) Prerequisites

- Clean working tree on `main`
- `gh` authenticated for the target GitHub repo
- Access to your Homebrew tap repo (`multikoop/homebrew-tap`)
- No secrets in command args or logs; release flow does not need Trello credentials

## 1) Prepare changelog

Update `CHANGELOG.md` with a version section:

- `## X.Y.Z - YYYY-MM-DD`

Do not leave `Unreleased` for the release section.

## 2) Cut tag and publish release

```sh
scripts/release.sh X.Y.Z
```

This will:
- run `go test ./...` and `go build ./...`
- validate changelog section
- create and push `vX.Y.Z` tag (if missing)
- create/update GitHub release notes from changelog

The GitHub release workflow (`.github/workflows/release.yml`) publishes binaries and `checksums.txt`.

## 3) Update Homebrew tap formula

In your tap repo, update `Formula/trelli.rb`:

- `version "X.Y.Z"`
- URLs for each OS/arch asset from `releases/download/vX.Y.Z/...`
- matching `sha256` values from `checksums.txt`

Formula template exists at `packaging/homebrew/trelli.rb` in this repo.

You can auto-update the formula:

```sh
scripts/update-tap-formula.sh X.Y.Z ../homebrew-tap/Formula/trelli.rb
```

## 4) Verify release end-to-end

```sh
TRELLI_FORMULA_PATH=../homebrew-tap/Formula/trelli.rb \
TRELLI_BREW_TAP=multikoop/tap \
scripts/verify-release.sh X.Y.Z
```

This validates:
- GitHub release notes and assets
- successful `release.yml` run for the tag
- checksums in formula match `checksums.txt`
- optional Homebrew install/test when `TRELLI_BREW_TAP` is set
