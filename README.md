# VibeDeploy

[![CI](https://github.com/its-the-vibe/VibeDeploy/actions/workflows/ci.yaml/badge.svg)](https://github.com/its-the-vibe/VibeDeploy/actions/workflows/ci.yaml)

Listen for emoji reactions and deploy feature branches

## Overview

VibeDeploy is a Go service that listens for Slack emoji reactions (specifically the "rocket" emoji) on messages containing PR metadata. When detected, it publishes deployment commands to a Redis list for execution by [Poppit](https://github.com/its-the-vibe/Poppit).

## Features

- Subscribes to Redis pub/sub channel for Slack reaction events
- Filters for "rocket" emoji reactions
- Retrieves message details from Slack API
- Extracts PR metadata from Slack messages
- **Repository filtering** - Optional whitelist configuration to control which repositories can be deployed
- **Immediate feedback** - Sends a gear emoji reaction when deployment starts to provide immediate user feedback
- Publishes deployment commands to Redis list for Poppit execution
- **Command output listening** - Listens for deployment completion, removes the gear emoji, and sends a rocket emoji reaction to indicate success

## Code Organization

The executable entry point and event orchestration are in the root package.
Supporting responsibilities are separated into focused internal packages:

- `internal/config` loads YAML settings and environment-based credentials.
- `internal/domain` defines event, metadata, and command types.
- `internal/deployment` builds deployment commands and applies repository rules.
- `internal/slack` encapsulates Slack message metadata retrieval.
- `internal/redis` encapsulates Redis connections and list operations.

## Configuration

Sensitive credentials are kept in `.env`; all non-sensitive settings belong in
`config.yaml`. Copy the provided templates to create local configuration:

```bash
cp .env.example .env
cp config.example.yaml config.yaml
```

Set `REDIS_PASSWORD` (if required) and `SLACK_BOT_TOKEN` in `.env`. The
application loads `config.yaml` from the repository root by default. Set
`CONFIG_FILE` to override its path.

### Logging Levels

VibeDeploy supports four logging levels:

- **DEBUG** - Detailed information for debugging (includes ignored reactions, received messages, etc.)
- **INFO** - General informational messages (connection status, processing events, successful operations)
- **WARN** - Warning messages (currently unused)
- **ERROR** - Error messages (parsing failures, API errors, etc.)

The default log level is `INFO`, which provides a good balance between visibility and verbosity. Use `DEBUG` for troubleshooting and `ERROR` for production environments where you only want to see failures.

### YAML Configuration

`config.yaml` supports the following settings and defaults:

| Setting | Default |
| --- | --- |
| `redis.addr` | `localhost:6379` |
| `base_dir` | `/app/repos` |
| `redis.pubsub_channel` | `slack-relay-reaction-added` |
| `redis.list_name` | `poppit-commands` |
| `redis.output_channel` | `poppit:command-output` |
| `redis.reaction_list` | `slack_reactions` |
| `log_level` | `INFO` |
| `docker_override` | `../vibebox/docker-override/docker-override` |

`allowed_repos` is an optional repository allowlist; if omitted, all repositories
are allowed. `legacy_docker_apps` lists repositories that use the legacy Docker
deployment workflow. See `config.example.yaml` for a complete template.

For example, to use a custom configuration file locally:

```bash
CONFIG_FILE=/path/to/config.yaml go run .
```

### Deployment Workflows (GHA vs Legacy Docker)

VibeDeploy supports two deployment patterns for feature branch deployments via the rocket emoji:

1. **Legacy Docker Apps**: Repositories listed under `legacy_docker_apps` in the YAML configuration use the traditional deployment flow:
   - `git fetch origin`
   - `git checkout <branch>`
   - `git pull`
   - `docker compose build`
   - `docker compose down`
   - `docker compose up -d`

2. **GHA-Enabled Projects**: Repositories not listed in `legacy_docker_apps` default to the GitHub Actions workflow:
   - `git fetch`
   - `git checkout <branch>`
   - `<dockerOverride> create --override-tag feature`
   - `gh label create "feature" --color "f107a3" --force --repo <repo>`
   - `gh pr edit --add-label "feature" <pr_url>`

## Building

### Local Build

```bash
go build -o vibedeploy .
```

Or use Make:

```bash
make build
```

### Docker Build

```bash
docker compose build
```

## Running CI Locally

Use the Makefile to run the same checks as CI:

```bash
# Lint (go vet)
make lint

# Build
make build

# Run unit tests with race detector and coverage
make test

# Run all CI checks
make ci
```

## Running

### Local Run

```bash
cp config.example.yaml config.yaml
cp .env.example .env
# Set SLACK_BOT_TOKEN in .env, then:
source .env
go run .
```

### Docker Run

```bash
# Ensure config.yaml exists and set redis.addr to host.docker.internal:6379
# when Redis is running on the Docker host.
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
    "git fetch origin",
    "git checkout feature/add-metadata",
    "docker compose build",
    "docker compose down",
    "docker compose up -d",
    "git checkout main"
  ],
  "metadata": {
    "channel": "C123",
    "ts": "1766236581.981479"
  }
}
```

### Command Output Messages

VibeDeploy listens on the `poppit:command-output` channel for command completion messages from Poppit. When it receives a message indicating that the `docker compose up -d` command has completed for a `vibe-deploy` type deployment, it publishes a success reaction.

Expected command output format from Poppit:

```json
{
  "metadata": {
    "channel": "C1234567890",
    "ts": "1766282873.772199"
  },
  "type": "vibe-deploy",
  "command": "docker compose up -d",
  "output": "<command output>"
}
```

### Slack Reaction Messages

VibeDeploy publishes reaction messages to the `slack_reactions` Redis list for SlackLiner to process:

**When deployment starts** (gear emoji to indicate in-progress):
```json
{
  "reaction": "gear",
  "channel": "C1234567890",
  "ts": "1766282873.772199"
}
```

**When deployment completes** (remove gear emoji):
```json
{
  "reaction": "gear",
  "channel": "C1234567890",
  "ts": "1766282873.772199",
  "remove": true
}
```

**When deployment completes** (rocket emoji to indicate success):
```json
{
  "reaction": "rocket",
  "channel": "C1234567890",
  "ts": "1766282873.772199"
}
```

## Requirements

- Go 1.24+
- Redis server (external)
- Slack Bot Token with appropriate permissions
