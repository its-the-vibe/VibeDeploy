# About VibeDeploy

VibeDeploy is a Go service that listens for Slack emoji reactions (specifically the "rocket" emoji) on messages containing PR metadata. When detected, it publishes deployment commands to a Redis list for execution by [Poppit](https://github.com/its-the-vibe/Poppit).

## Project Structure

- `main.go` - Main application with all core logic including:
  - Redis pub/sub subscription for Slack reaction events
  - Slack API integration for retrieving message metadata
  - Poppit command generation and publishing
- `README.md` - Project documentation
- `TESTING.md` - Manual testing guide
- `Dockerfile` - Container configuration
- `docker-compose.yml` - Docker Compose setup
- `.env.example` - Environment variable template

## Configuration

Only sensitive credentials are configured via environment variables:

- `REDIS_PASSWORD` - Redis password (optional)
- `SLACK_BOT_TOKEN` - Slack bot token (required)

Non-sensitive settings are stored in `config.yaml`, which is loaded from the repository root by default. `CONFIG_FILE` can override the path. See `config.example.yaml` for Redis settings, base directory, logging level, allowed repositories, legacy Docker apps, and Docker override path.

## Building and Running

### Local Development

Build the project:
```bash
go build -o vibedeploy .
```

Run locally:
```bash
cp .env.example .env
cp config.example.yaml config.yaml
# Set SLACK_BOT_TOKEN in .env
source .env
go run .
```

Or run directly:
```bash
source .env
go run main.go
```

### Docker

Build and run with Docker Compose:
```bash
docker compose build
docker compose up -d
```

## Testing

This project currently uses manual testing. See `TESTING.md` for detailed testing instructions.

Key testing steps:
1. Start Redis server
2. Run the service (locally or via Docker)
3. Use Redis CLI to publish test reaction events
4. Verify logs show proper event processing
5. Check Redis list for published Poppit commands

### Manual Testing Example

```bash
# Start Redis
docker run -d -p 6379:6379 redis:alpine

# Run service
source .env
go run main.go

# In another terminal, publish test event
redis-cli PUBLISH slack-relay-reaction-added '{"event":{"type":"reaction_added","reaction":"rocket","item":{"type":"message","channel":"C123","ts":"1234567890.123456"}}}'
```

## Key Principles

### Code Style
- Follow standard Go formatting (use `gofmt`)
- Use standard Go naming conventions
- Keep functions focused and single-purpose
- Use descriptive variable names

### Error Handling
- Always check and handle errors appropriately
- Log errors with context using `log.Printf`
- Return errors from functions rather than logging internally when appropriate
- Use `fmt.Errorf` with `%w` for error wrapping

### Logging
- Use standard `log` package for logging
- Include context in log messages (channel ID, message timestamp, repository, branch, etc.)
- Log key events: connection status, message processing, command publishing

### Dependencies
- Minimize external dependencies
- Use well-maintained libraries:
  - `github.com/redis/go-redis/v9` for Redis
  - `github.com/slack-go/slack` for Slack API

### Configuration
- Keep only `REDIS_PASSWORD` and `SLACK_BOT_TOKEN` in `.env`; store non-sensitive settings in `config.yaml`
- Provide sensible defaults where possible
- Document configuration options in README.md, `.env.example`, and `config.example.yaml`
- Fail fast if required configuration is missing

### Data Structures
- Use clear struct definitions with JSON tags for serialization
- Validate required fields before processing
- Use pointer receivers when structs may be nil

## Architecture Notes

The service follows a simple event-driven architecture:
1. Subscribe to Redis pub/sub channel for Slack reaction events
2. Filter for "rocket" emoji reactions on messages
3. Fetch message details from Slack API
4. Extract PR metadata from message
5. Generate deployment commands
6. Publish to Redis list for Poppit consumption

## Requirements

- Go 1.24+
- Redis server (external dependency)
- Slack Bot Token with permissions to read messages and reactions

## Integration Points

### Slack Relay
Expects reaction events in this format:
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

### Slack Messages
Messages must contain PR metadata in the following format:
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

### Poppit Commands
Publishes commands to Redis in this format:
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
