---
name: trelli
description: Manage Trello boards, lists, cards, comments, and checklists using the trelli CLI.
version: "1.0"
homepage: https://github.com/multikoop/trelli/ based on https://developer.atlassian.com/cloud/trello/rest/
---

# trelli Skill

Use `trelli` cli to interact with Trello efficiently.

Install it by `brew install multikoop/tap/trelli`

## Setup

1. Get API key and token from https://trello.com/app-key
2. Set credentials:

```bash
export TRELLO_API_KEY="your-api-key"
export TRELLO_TOKEN="your-token"
export TRELLO_BOARD_ID="<id> (smt like XobnRsYv)"
```

3. Build CLI:

```bash
go build -o trelli ./cmd/trelli
```

## Usage

### Help and command discovery

```bash
./trelli -h
./trelli help cards
./trelli cards --help
```

### Global options (all commands)

- `--key <key>`: Trello API key (default `TRELLO_API_KEY`)
- `--token <token>`: Trello token (default `TRELLO_TOKEN`)
- `--board <id>`: Board id/shortLink default (default `TRELLO_BOARD_ID`)
- `--json`: Raw JSON output for scripts/automation

### Typical workflows

#### Discover boards and lists

```bash
./trelli boards list
./trelli boards list --filter sandbox
./trelli lists list --board <boardId>
```

#### List and inspect cards

```bash
./trelli cards list --list <listId>
./trelli cards list --board <boardId> --list-name "To Do" --limit 50
./trelli cards show --card <cardId>
```

#### Create cards (simple and advanced)

```bash
./trelli cards create --board <boardId> --list-name "To Do" --name "Card Title"
./trelli cards create --board <boardId> --list-name "To Do" --name "Release 1.0" --desc "Ship checklist" --due 2026-02-14T18:00:00Z --labels <labelId1,labelId2> --members <memberId1,memberId2>
```

#### Move and archive cards

```bash
./trelli cards move --card <cardId> --board <boardId> --list-name "Doing"
./trelli cards move --card <cardId> --list <targetListId>
./trelli cards archive --card <cardId>
```

#### Add and read comments

```bash
./trelli comments add --card <cardId> --text "Your comment"
./trelli comments list --card <cardId> --limit 20
```

#### Manage checklists and items

```bash
./trelli checklists list --card <cardId>
./trelli checklists create --card <cardId> --name "Checklist"
./trelli checklists add-item --checklist <checklistId> --name "Task item"
./trelli checklists add-item --checklist <checklistId> --name "Already done" --checked
./trelli checklists set-item --card <cardId> --item <itemId> --state complete
./trelli checklists set-item --card <cardId> --item <itemId> --state incomplete
```

### JSON output for automation

```bash
./trelli boards list --json
./trelli cards show --card <cardId> --json
./trelli comments list --card <cardId> --limit 100 --json
```

## Notes

- Prefer `--json` for scripting.
- Use `--board <boardId>` to stay scoped to `your board name`.
- Keep credentials secret.
