# VibeDeploy

Listen for emoji reactions and deploy feature branches

## Overview

VibeDeploy is a Go service that listens for Slack emoji reactions (specifically the "rocket" emoji) on messages containing PR metadata. When detected, it publishes deployment commands to a Redis list for execution by [Poppit](https://github.com/its-the-vibe/Poppit).

## Features

- Subscribes to Redis pub/sub channel for Slack reaction events
- Filters for "rocket" emoji reactions
- Retrieves message details from Slack API
- Extracts PR metadata from Slack messages
- Publishes deployment commands to Redis list for Poppit execution

## Configuration

Configuration is done via environment variables:

- `REDIS_ADDR` - Redis server address (default: `localhost:6379`)
- `SLACK_BOT_TOKEN` - Slack bot token (required)
- `BASE_DIR` - Base directory for repositories (default: `/app/repos`)
- `REDIS_PUBSUB_CHANNEL` - Redis pub/sub channel to subscribe to (default: `slack-relay-reaction-added`)
- `REDIS_LIST_NAME` - Redis list name for Poppit commands (default: `poppit-commands`)

See `.env.example` for a template.

## Building

### Local Build

```bash
go build -o vibedeploy .
```

### Docker Build

```bash
docker compose build
```

## Running

### Local Run

```bash
export SLACK_BOT_TOKEN=xoxb-your-token
export REDIS_ADDR=localhost:6379
./vibedeploy
```

### Docker Run

```bash
docker compose up -d
```

## Expected Message Format

### Slack Relay Reaction Event

The service expects reaction events in this format:

```json
{
  "event": {
    "type": "reaction_added",
    "user": "U...",
    "reaction": "rocket",
    "item": {
      "type": "message",
      "channel": "C...",
      "ts": "1766236581.981479"
    }
  }
}
```

### Slack Message Metadata

Messages should contain PR metadata in this format:

```json
{
  "pr_number": 42,
  "repository": "its-the-vibe/VibeMerge",
  "pr_url": "https://github.com/its-the-vibe/VibeMerge/pull/42",
  "author": "username123",
  "branch": "feature/add-metadata",
  "event_action": "review_requested"
}
```

### Poppit Command Output

The service publishes commands to Redis in this format:

```json
{
  "repo": "its-the-vibe/VibeMerge",
  "branch": "feature/add-metadata",
  "type": "vibe-deploy",
  "dir": "/app/repos/its-the-vibe/VibeMerge",
  "commands": [
    "git checkout feature/add-metadata",
    "docker compose build",
    "docker compose down",
    "docker compose up -d",
    "git checkout main"
  ]
}
```

## Requirements

- Go 1.24+
- Redis server (external)
- Slack Bot Token with appropriate permissions

