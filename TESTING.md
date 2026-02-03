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

## Testing Command Output Listening

### 1. Simulate Command Output from Poppit

With the service running, publish a command output message:

```bash
redis-cli PUBLISH "poppit:command-output" '{
  "metadata": {
    "channel": "C1234567890",
    "ts": "1766282873.772199"
  },
  "type": "vibe-deploy",
  "command": "docker compose up -d",
  "output": "Container started successfully"
}'
```

### 2. Verify Slack Reaction Published

Check the `slack_reactions` Redis list:

```bash
redis-cli LLEN slack_reactions
redis-cli LPOP slack_reactions
```

You should see:

```json
{
  "reaction": "rocket",
  "channel": "C1234567890",
  "ts": "1766282873.772199"
}
```

### 3. Expected Log Output for Command Output

When processing command output, you should see logs like:

```
Subscribed to Redis channel: poppit:command-output
Received command output message from channel: poppit:command-output
Processing completion for vibe-deploy in channel C1234567890, message 1766282873.772199
Successfully published slack reaction for channel C1234567890, message 1766282873.772199
```

### 4. Negative Test Cases for Command Output

#### Non-vibe-deploy Type

```bash
redis-cli PUBLISH "poppit:command-output" '{
  "metadata": {"channel": "C123", "ts": "123.456"},
  "type": "other-type",
  "command": "docker compose up -d",
  "output": "output"
}'
```

Expected: Service logs "Ignoring command output type: other-type (not vibe-deploy)"

#### Different Command

```bash
redis-cli PUBLISH "poppit:command-output" '{
  "metadata": {"channel": "C123", "ts": "123.456"},
  "type": "vibe-deploy",
  "command": "git checkout main",
  "output": "output"
}'
```

Expected: Service logs "Ignoring command: git checkout main (not docker compose up -d)"

#### Missing Metadata

```bash
redis-cli PUBLISH "poppit:command-output" '{
  "type": "vibe-deploy",
  "command": "docker compose up -d",
  "output": "output"
}'
```

Expected: Service logs "Command output missing metadata, cannot send reaction"

## Expected Log Output

When processing a rocket reaction, you should see logs like:

```
Connected to Redis
Subscribed to Redis channel: slack-relay-reaction-added
Subscribed to Redis channel: poppit:command-output
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

### 2. Bot User Reactions

To test bot reaction filtering, the service now checks the `authorizations` field in the payload to detect if the reaction is from the bot itself.

Publish an event where the bot user adds a reaction (replace `B123BOT` with your bot's user ID):

```redis
PUBLISH slack-relay-reaction-added '{"event":{"type":"reaction_added","user":"B123BOT","reaction":"rocket","item":{"type":"message","channel":"C123","ts":"1234567890.123456"}},"authorizations":[{"user_id":"B123BOT","is_bot":true}]}'
```

Expected: Service logs "Ignoring rocket reaction from bot user B123BOT on message..."

**Note:** The service filters out reactions where the user ID matches a bot in the `authorizations` array. This prevents the bot from triggering deployments on its own reactions without requiring additional API calls.

### 3. Message Without Metadata

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

### No slack reactions generated

- Verify the command output message has the correct format
- Check that `type` is "vibe-deploy" and `command` is "docker compose up -d"
- Ensure metadata with channel and ts is present in the command output

### Connection errors

- Verify Redis server is running and accessible
- Check network connectivity
- Ensure firewall rules allow connections
