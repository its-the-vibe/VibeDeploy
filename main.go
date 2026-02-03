package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/slack-go/slack"
	"gopkg.in/yaml.v3"
)

type Config struct {
	RedisAddr          string
	RedisPassword      string
	SlackToken         string
	BaseDir            string
	RedisPubSub        string
	RedisListName      string
	RedisOutputChannel string
	RedisReactionList  string
	LogLevel           LogLevel
	AllowedReposConfig string
}

const RocketReaction = "rocket"
const GearReaction = "gear"
const VibeDeployType = "vibe-deploy"
const DeploymentCommand = "docker compose up -d"

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// currentLogLevel is set once at startup before any goroutines are created,
// then only read during runtime, so no synchronization is needed
var currentLogLevel = INFO // Default log level

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// parseLogLevel converts a string to a LogLevel
func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

// logDebug logs a debug message
func logDebug(format string, v ...interface{}) {
	if currentLogLevel <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// logInfo logs an info message
func logInfo(format string, v ...interface{}) {
	if currentLogLevel <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

// logWarn logs a warning message
func logWarn(format string, v ...interface{}) {
	if currentLogLevel <= WARN {
		log.Printf("[WARN] "+format, v...)
	}
}

// logError logs an error message
func logError(format string, v ...interface{}) {
	if currentLogLevel <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

type ReactionEvent struct {
	Event struct {
		Type     string `json:"type"`
		User     string `json:"user"`
		Reaction string `json:"reaction"`
		Item     struct {
			Type    string `json:"type"`
			Channel string `json:"channel"`
			Ts      string `json:"ts"`
		} `json:"item"`
	} `json:"event"`
	Authorizations []struct {
		UserID string `json:"user_id"`
		IsBot  bool   `json:"is_bot"`
	} `json:"authorizations"`
}

type PRMetadata struct {
	PRNumber    int    `json:"pr_number"`
	Repository  string `json:"repository"`
	PRUrl       string `json:"pr_url"`
	Author      string `json:"author"`
	Branch      string `json:"branch"`
	EventAction string `json:"event_action"`
}

type AllowedReposConfig struct {
	AllowedRepos []string `yaml:"allowed_repos"`
}

type PoppitCommand struct {
	Repo     string           `json:"repo"`
	Branch   string           `json:"branch"`
	Type     string           `json:"type"`
	Dir      string           `json:"dir"`
	Commands []string         `json:"commands"`
	Metadata *CommandMetadata `json:"metadata,omitempty"`
}

type CommandMetadata struct {
	Channel string `json:"channel"`
	Ts      string `json:"ts"`
}

type CommandOutput struct {
	Metadata *CommandMetadata `json:"metadata"`
	Type     string           `json:"type"`
	Command  string           `json:"command"`
	Output   string           `json:"output"`
}

type SlackReaction struct {
	Reaction string `json:"reaction"`
	Channel  string `json:"channel"`
	Ts       string `json:"ts"`
	Remove   bool   `json:"remove,omitempty"`
}

func loadConfig() Config {
	logLevel := parseLogLevel(getEnv("LOG_LEVEL", "INFO"))
	return Config{
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		SlackToken:         getEnv("SLACK_BOT_TOKEN", ""),
		BaseDir:            getEnv("BASE_DIR", "/app/repos"),
		RedisPubSub:        getEnv("REDIS_PUBSUB_CHANNEL", "slack-relay-reaction-added"),
		RedisListName:      getEnv("REDIS_LIST_NAME", "poppit-commands"),
		RedisOutputChannel: getEnv("REDIS_OUTPUT_CHANNEL", "poppit:command-output"),
		RedisReactionList:  getEnv("REDIS_REACTION_LIST", "slack_reactions"),
		LogLevel:           logLevel,
		AllowedReposConfig: getEnv("ALLOWED_REPOS_CONFIG", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadAllowedRepos loads the list of allowed repositories from the config file
// Returns (nil, nil) if no config file is specified or if the file doesn't exist (allow all repos)
func loadAllowedRepos(configPath string) (map[string]bool, error) {
	// If no config path specified, allow all repos
	if configPath == "" {
		logInfo("No allowed repos config specified, allowing all repositories")
		return nil, nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		logInfo("Allowed repos config file not found at %s, allowing all repositories", configPath)
		return nil, nil
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read allowed repos config: %w", err)
	}

	// Parse YAML
	var config AllowedReposConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse allowed repos config: %w", err)
	}

	// Convert to map for faster lookup
	allowedRepos := make(map[string]bool)
	for _, repo := range config.AllowedRepos {
		allowedRepos[repo] = true
	}

	logInfo("Loaded %d allowed repositories from config", len(allowedRepos))
	return allowedRepos, nil
}

// isRepoAllowed checks if a repository is in the allowed list
// If allowedRepos is nil (no config), all repos are allowed
func isRepoAllowed(repo string, allowedRepos map[string]bool) bool {
	// If no allowlist is configured, allow all repos
	if allowedRepos == nil {
		return true
	}

	// Check if repo is in the allowlist
	return allowedRepos[repo]
}

func main() {
	config := loadConfig()

	// Set the global log level
	currentLogLevel = config.LogLevel

	if config.SlackToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable is required")
	}

	// Load allowed repos configuration
	allowedRepos, err := loadAllowedRepos(config.AllowedReposConfig)
	if err != nil {
		log.Fatalf("Failed to load allowed repos configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	logInfo("Connected to Redis at %s", config.RedisAddr)

	// Setup Slack client
	slackClient := slack.New(config.SlackToken)

	// Subscribe to Redis pub/sub channel
	pubsub := redisClient.Subscribe(ctx, config.RedisPubSub)
	defer pubsub.Close()

	logInfo("Subscribed to Redis channel: %s (log level: %s)", config.RedisPubSub, config.LogLevel.String())

	// Start command output listener in a goroutine
	go listenForCommandOutput(ctx, redisClient, config)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logInfo("Shutting down...")
		cancel()
	}()

	// Process messages
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			logInfo("Context cancelled, exiting")
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			logDebug("Received message from channel: %s", config.RedisPubSub)
			processReactionEvent(ctx, msg.Payload, slackClient, redisClient, config, allowedRepos)
		}
	}
}

func processReactionEvent(ctx context.Context, payload string, slackClient *slack.Client, redisClient *redis.Client, config Config, allowedRepos map[string]bool) {
	var event ReactionEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		logError("Error parsing reaction event: %v", err)
		return
	}

	// Only process rocket emoji reactions
	if event.Event.Reaction != RocketReaction {
		logDebug("Ignoring reaction: %s (not %s)", event.Event.Reaction, RocketReaction)
		return
	}

	// Only process message items
	if event.Event.Item.Type != "message" {
		logDebug("Ignoring item type: %s (not message)", event.Event.Item.Type)
		return
	}

	// Check if the reaction is from the bot itself by comparing with authorizations
	for _, auth := range event.Authorizations {
		if auth.IsBot && auth.UserID == event.Event.User {
			logInfo("Ignoring %s reaction from bot user %s on message %s in channel %s", RocketReaction, event.Event.User, event.Event.Item.Ts, event.Event.Item.Channel)
			return
		}
	}

	logInfo("Processing %s reaction on message %s in channel %s", RocketReaction, event.Event.Item.Ts, event.Event.Item.Channel)

	// Fetch message from Slack
	metadata, err := getMessageMetadata(slackClient, event.Event.Item.Channel, event.Event.Item.Ts)
	if err != nil {
		logError("Error getting message metadata: %v", err)
		return
	}

	if metadata == nil {
		logDebug("No PR metadata found in message, skipping")
		return
	}

	logInfo("Found PR metadata: %s #%d (branch: %s)", metadata.Repository, metadata.PRNumber, metadata.Branch)

	// Check if repository is allowed
	if !isRepoAllowed(metadata.Repository, allowedRepos) {
		logInfo("Repository %s is not in the allowed list, ignoring reaction", metadata.Repository)
		return
	}

	// Publish gear reaction to indicate deployment is starting
	if err := publishSlackReaction(ctx, redisClient, event.Event.Item.Channel, event.Event.Item.Ts, GearReaction, false, config); err != nil {
		logError("Error publishing gear reaction: %v", err)
		// Continue even if reaction fails - deployment should still proceed
	} else {
		logInfo("Published gear reaction for channel %s, message %s", event.Event.Item.Channel, event.Event.Item.Ts)
	}

	// Create and publish Poppit command
	poppitCmd := createPoppitCommand(metadata, config, event.Event.Item.Channel, event.Event.Item.Ts)
	if err := publishPoppitCommand(ctx, redisClient, poppitCmd, config); err != nil {
		logError("Error publishing Poppit command: %v", err)
		return
	}

	logInfo("Successfully published Poppit command for %s branch %s", metadata.Repository, metadata.Branch)
}

func getMessageMetadata(slackClient *slack.Client, channel, timestamp string) (*PRMetadata, error) {
	// Fetch the message
	historyParams := &slack.GetConversationHistoryParameters{
		ChannelID:          channel,
		Latest:             timestamp,
		Inclusive:          true,
		Limit:              1,
		IncludeAllMetadata: true,
	}

	history, err := slackClient.GetConversationHistory(historyParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	if len(history.Messages) == 0 {
		return nil, fmt.Errorf("no messages found")
	}

	message := history.Messages[0]

	// Check if message has metadata
	if len(message.Metadata.EventPayload) == 0 {
		return nil, nil
	}

	// Parse metadata
	metadataJSON, err := json.Marshal(message.Metadata.EventPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadata PRMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse PR metadata: %w", err)
	}

	// Verify required fields are present
	if metadata.Repository == "" || metadata.Branch == "" {
		return nil, nil
	}

	return &metadata, nil
}

func createPoppitCommand(metadata *PRMetadata, config Config, channel, timestamp string) PoppitCommand {
	dir := fmt.Sprintf("%s/%s", config.BaseDir, metadata.Repository)

	return PoppitCommand{
		Repo:   metadata.Repository,
		Branch: metadata.Branch,
		Type:   VibeDeployType,
		Dir:    dir,
		Commands: []string{
			"git fetch origin",
			fmt.Sprintf("git checkout %s", metadata.Branch),
			"git pull",
			"docker compose build",
			"docker compose down",
			DeploymentCommand,
			"git checkout main",
		},
		Metadata: &CommandMetadata{
			Channel: channel,
			Ts:      timestamp,
		},
	}
}

func publishPoppitCommand(ctx context.Context, redisClient *redis.Client, cmd PoppitCommand, config Config) error {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal Poppit command: %w", err)
	}

	if err := redisClient.RPush(ctx, config.RedisListName, payload).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis list: %w", err)
	}

	return nil
}

func listenForCommandOutput(ctx context.Context, redisClient *redis.Client, config Config) {
	// Subscribe to command output channel
	pubsub := redisClient.Subscribe(ctx, config.RedisOutputChannel)
	defer pubsub.Close()

	logInfo("Subscribed to Redis channel: %s", config.RedisOutputChannel)

	// Process messages
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			logInfo("Command output listener context cancelled, exiting")
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			logDebug("Received command output message from channel: %s", config.RedisOutputChannel)
			processCommandOutput(ctx, msg.Payload, redisClient, config)
		}
	}
}

func processCommandOutput(ctx context.Context, payload string, redisClient *redis.Client, config Config) {
	var output CommandOutput
	if err := json.Unmarshal([]byte(payload), &output); err != nil {
		logError("Error parsing command output: %v", err)
		return
	}

	// Only process vibe-deploy type commands
	if output.Type != VibeDeployType {
		logDebug("Ignoring command output type: %s (not %s)", output.Type, VibeDeployType)
		return
	}

	// Only process docker compose up -d command
	if output.Command != DeploymentCommand {
		logDebug("Ignoring command: %s (not %s)", output.Command, DeploymentCommand)
		return
	}

	// Check if metadata is present
	if output.Metadata == nil {
		logWarn("Command output missing metadata (channel and timestamp required), cannot send reaction")
		return
	}

	logInfo("Processing completion for %s in channel %s, message %s", VibeDeployType, output.Metadata.Channel, output.Metadata.Ts)

	// Remove gear reaction to indicate deployment is no longer in progress
	if err := publishSlackReaction(ctx, redisClient, output.Metadata.Channel, output.Metadata.Ts, GearReaction, true, config); err != nil {
		logError("Error removing gear reaction: %v", err)
		// Continue even if reaction removal fails
	} else {
		logInfo("Removed gear reaction for channel %s, message %s", output.Metadata.Channel, output.Metadata.Ts)
	}

	// Publish rocket reaction to indicate success
	if err := publishSlackReaction(ctx, redisClient, output.Metadata.Channel, output.Metadata.Ts, RocketReaction, false, config); err != nil {
		logError("Error publishing rocket reaction: %v", err)
		// Continue even if final reaction fails - deployment was still successful
	} else {
		logInfo("Successfully published rocket reaction for channel %s, message %s", output.Metadata.Channel, output.Metadata.Ts)
	}
}

func publishSlackReaction(ctx context.Context, redisClient *redis.Client, channel, timestamp, reaction string, remove bool, config Config) error {
	slackReaction := SlackReaction{
		Reaction: reaction,
		Channel:  channel,
		Ts:       timestamp,
		Remove:   remove,
	}

	payload, err := json.Marshal(slackReaction)
	if err != nil {
		return fmt.Errorf("failed to marshal slack reaction: %w", err)
	}

	if err := redisClient.RPush(ctx, config.RedisReactionList, payload).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis list: %w", err)
	}

	return nil
}
