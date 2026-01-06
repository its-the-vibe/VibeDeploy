# VibeDeploy Manual Testing Guide

This document describes how to manually test VibeDeploy.

## Prerequisites

1. A running Redis server
2. A Slack Bot Token with appropriate permissions
3. A Slack workspace with the bot installed

## Setup

1. Create a `.env` file based on `.env.example`:

```bash
cp .env.example .env
```

2. Edit `.env` and set your actual values:
   - `SLACK_BOT_TOKEN` - Your Slack bot token
   - `REDIS_ADDR` - Your Redis server address

3. Start Redis (if not already running):

```bash
# Using Docker
docker run -d -p 6379:6379 redis:alpine

# Or use your existing Redis instance
```

## Running the Service

### Option 1: Run locally

```bash
source .env
go run main.go
```

### Option 2: Run with Docker

```bash
docker compose up
```

## Testing the Service

### 1. Simulate a Reaction Event

Use Redis CLI to publish a test event:

```bash
redis-cli
```

Then publish a test reaction event:

```redis
PUBLISH slack-relay-reaction-added '{"token":"test","team_id":"T123","event":{"type":"reaction_added","user":"U123","reaction":"rocket","item":{"type":"message","channel":"C123","ts":"1234567890.123456"},"event_ts":"1234567890.123456"},"type":"event_callback"}'
```

### 2. Verify Behavior

The service should:
1. Log that it received the message
2. Log that it's processing a rocket reaction
3. Attempt to fetch the message from Slack
4. If the message has PR metadata, publish a Poppit command to Redis

### 3. Check Poppit Commands

In Redis CLI, check if commands were published:

```redis
LLEN poppit-commands
LPOP poppit-commands
```

You should see a JSON payload like:

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

## Expected Log Output

When processing a rocket reaction, you should see logs like:

```
Connected to Redis
Subscribed to Redis channel: slack-relay-reaction-added
Received message from channel: slack-relay-reaction-added
Processing rocket reaction on message 1234567890.123456 in channel C123
Found PR metadata: its-the-vibe/VibeMerge #42 (branch: feature/add-metadata)
Successfully published Poppit command for its-the-vibe/VibeMerge branch feature/add-metadata
```

## Negative Test Cases

### 1. Non-rocket Reactions

Publish an event with a different reaction:

```redis
PUBLISH slack-relay-reaction-added '{"event":{"type":"reaction_added","reaction":"thumbsup","item":{"type":"message","channel":"C123","ts":"1234567890.123456"}}}'
```

Expected: Service logs "Ignoring reaction: thumbsup (not rocket)"

### 2. Message Without Metadata

If the Slack message doesn't have PR metadata, the service should log:
"No PR metadata found in message, skipping"

## Troubleshooting

### Service won't start

- Check that `SLACK_BOT_TOKEN` is set
- Verify Redis is accessible at `REDIS_ADDR`

### No Poppit commands generated

- Ensure the Slack message has PR metadata in the correct format
- Check Slack API token permissions
- Verify the message timestamp is correct

### Connection errors

- Verify Redis server is running and accessible
- Check network connectivity
- Ensure firewall rules allow connections
