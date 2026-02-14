---
name: trelli
description: Manage Trello boards, lists, cards, comments, and checklists using the trelli CLI.
homepage: https://developer.atlassian.com/cloud/trello/rest/
metadata: {"codex":{"emoji":"📋","requires":{"bins":["trelli"],"env":["TRELLO_API_KEY","TRELLO_TOKEN"]}}}
---

# trelli Skill

Use `trelli` instead of raw `curl` to interact with Trello efficiently.

Default board is `trelli.sandbox` (`XobnRsYv`) unless overridden.

## Setup

1. Get API key and token from https://trello.com/app-key
2. Set credentials:

```bash
export TRELLO_API_KEY="your-api-key"
export TRELLO_TOKEN="your-token"
export TRELLO_BOARD_ID="XobnRsYv"
```

3. Build CLI:

```bash
go build -o trelli ./cmd/trelli
```

## Usage

### Show help

```bash
./trelli --help
./trelli help cards
```

### List boards

```bash
./trelli boards list
```

### List lists in sandbox board

```bash
./trelli lists list --board XobnRsYv
```

### List cards in a list

```bash
./trelli cards list --list <listId>
# or
./trelli cards list --board XobnRsYv --list-name "To Do"
```

### Create a card

```bash
./trelli cards create --board XobnRsYv --list-name "To Do" --name "Card Title" --desc "Card description"
```

### Move a card

```bash
./trelli cards move --card <cardId> --board XobnRsYv --list-name "Doing"
```

### Add and read comments

```bash
./trelli comments add --card <cardId> --text "Your comment"
./trelli comments list --card <cardId>
```

### Checklists

```bash
./trelli checklists create --card <cardId> --name "Checklist"
./trelli checklists add-item --checklist <checklistId> --name "Task item"
./trelli checklists list --card <cardId>
./trelli checklists set-item --card <cardId> --item <itemId> --state complete
```

### Archive card

```bash
./trelli cards archive --card <cardId>
```

## Notes

- Prefer `--json` for scripting.
- Use `--board XobnRsYv` to stay scoped to `trelli.sandbox`.
- Keep credentials secret.
