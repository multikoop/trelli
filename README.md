# trelli

`trelli` is a Go CLI for fast Trello workflows: boards, lists, cards, comments, and checklists.

The default board is the Trello sandbox board `trelli.sandbox` (`XobnRsYv`).

## Requirements

- Go 1.22+
- Trello API credentials:
  - `TRELLO_API_KEY`
  - `TRELLO_TOKEN`

Get credentials from:
- https://trello.com/app-key

## Build

```bash
go build -o trelli ./cmd/trelli
```

## Installation

### Homebrew (tap)

After you publish releases and maintain your tap formula:

```bash
brew install multikoop/tap/trelli
```

### Local binary

```bash
go build -o trelli ./cmd/trelli
./trelli --help
```

## Configuration

Environment variables:

```bash
export TRELLO_API_KEY="your-key"
export TRELLO_TOKEN="your-token"
export TRELLO_BOARD_ID="XobnRsYv"  # optional, defaults to sandbox board
```

You can also pass credentials and board via flags:

```bash
./trelli --key "$TRELLO_API_KEY" --token "$TRELLO_TOKEN" --board XobnRsYv boards list
```

## Help

```bash
./trelli -h
./trelli --help
./trelli help cards
./trelli cards --help
./trelli version
```

## Global Options

- `--key <key>`: Trello API key
- `--token <token>`: Trello API token
- `--board <idOrShortLink>`: default board for commands that need board context
- `--json`: emit raw JSON
- `-h`, `--help`: show help

## Commands

### Boards

```bash
./trelli boards list [--filter <text>]
```

### Lists

```bash
./trelli lists list [--board <boardIdOrShortLink>]
```

### Cards

```bash
./trelli cards list --list <listId> [--limit <n>]
./trelli cards list --list-name <name> [--board <boardIdOrShortLink>] [--limit <n>]
./trelli cards show --card <cardId>
./trelli cards create (--list <listId> | --list-name <name>) --name <title> [--desc <text>] [--due <iso8601>] [--labels <id1,id2>] [--members <id1,id2>] [--board <boardIdOrShortLink>]
./trelli cards move --card <cardId> (--list <listId> | --list-name <name>) [--board <boardIdOrShortLink>]
./trelli cards archive --card <cardId>
```

### Comments

```bash
./trelli comments list --card <cardId> [--limit <n>]
./trelli comments add --card <cardId> --text <comment>
```

### Checklists

```bash
./trelli checklists list --card <cardId>
./trelli checklists create --card <cardId> --name <checklistName>
./trelli checklists add-item --checklist <checklistId> --name <itemName> [--checked]
./trelli checklists set-item --card <cardId> --item <itemId> --state <complete|incomplete>
```

## Release and Brew Publishing

Files added for release automation:

- `.goreleaser.yaml`
- `.github/workflows/release.yml`
- `.github/workflows/ci.yml`
- `scripts/release.sh`
- `scripts/update-tap-formula.sh`
- `scripts/verify-release.sh`
- `docs/RELEASING.md`
- `packaging/homebrew/trelli.rb` (formula template)

Quick flow:

```bash
scripts/release.sh X.Y.Z
# update Formula/trelli.rb in your tap repo with new version + checksums
scripts/update-tap-formula.sh X.Y.Z ../homebrew-tap/Formula/trelli.rb
TRELLI_FORMULA_PATH=../homebrew-tap/Formula/trelli.rb TRELLI_BREW_TAP=multikoop/tap scripts/verify-release.sh X.Y.Z
```

Detailed guide: `docs/RELEASING.md`.

## Practical Sandbox Workflow

List lists on the sandbox board:

```bash
./trelli lists list --board XobnRsYv
```

Create a card in list `To Do` by name:

```bash
./trelli cards create --board XobnRsYv --list-name "To Do" --name "Implement API smoke flow" --desc "Create/checklist/comment/readback"
```

Add a comment:

```bash
./trelli comments add --card <cardId> --text "Smoke flow started"
```

Create checklist and add item:

```bash
./trelli checklists create --card <cardId> --name "Verification"
./trelli checklists add-item --checklist <checklistId> --name "Confirm comments"
```

Read back comments and checklists:

```bash
./trelli comments list --card <cardId>
./trelli checklists list --card <cardId>
```

Archive card:

```bash
./trelli cards archive --card <cardId>
```

## Security Notes

- Keep `TRELLO_API_KEY` and `TRELLO_TOKEN` secret.
- Do not place tokens in committed files or scripts.
- Avoid passing tokens in command history when possible; prefer environment variables.
