package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/slack-go/slack"
)

type Config struct {
	RedisAddr     string
	SlackToken    string
	BaseDir       string
	RedisPubSub   string
	RedisListName string
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
}

type PRMetadata struct {
	PRNumber    int    `json:"pr_number"`
	Repository  string `json:"repository"`
	PRUrl       string `json:"pr_url"`
	Author      string `json:"author"`
	Branch      string `json:"branch"`
	EventAction string `json:"event_action"`
}

type PoppitCommand struct {
	Repo     string   `json:"repo"`
	Branch   string   `json:"branch"`
	Type     string   `json:"type"`
	Dir      string   `json:"dir"`
	Commands []string `json:"commands"`
}

func loadConfig() Config {
	return Config{
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		SlackToken:    getEnv("SLACK_BOT_TOKEN", ""),
		BaseDir:       getEnv("BASE_DIR", "/app/repos"),
		RedisPubSub:   getEnv("REDIS_PUBSUB_CHANNEL", "slack-relay-reaction-added"),
		RedisListName: getEnv("REDIS_LIST_NAME", "poppit-commands"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	config := loadConfig()

	if config.SlackToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	// Setup Slack client
	slackClient := slack.New(config.SlackToken)

	// Subscribe to Redis pub/sub channel
	pubsub := redisClient.Subscribe(ctx, config.RedisPubSub)
	defer pubsub.Close()

	log.Printf("Subscribed to Redis channel: %s", config.RedisPubSub)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Process messages
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, exiting")
			return
		case msg := <-ch:
			if msg == nil {
				continue
			}
			log.Printf("Received message from channel: %s", config.RedisPubSub)
			processReactionEvent(ctx, msg.Payload, slackClient, redisClient, config)
		}
	}
}

func processReactionEvent(ctx context.Context, payload string, slackClient *slack.Client, redisClient *redis.Client, config Config) {
	var event ReactionEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("Error parsing reaction event: %v", err)
		return
	}

	// Only process rocket emoji reactions
	if event.Event.Reaction != "rocket" {
		log.Printf("Ignoring reaction: %s (not rocket)", event.Event.Reaction)
		return
	}

	// Only process message items
	if event.Event.Item.Type != "message" {
		log.Printf("Ignoring item type: %s (not message)", event.Event.Item.Type)
		return
	}

	log.Printf("Processing rocket reaction on message %s in channel %s", event.Event.Item.Ts, event.Event.Item.Channel)

	// Fetch message from Slack
	metadata, err := getMessageMetadata(slackClient, event.Event.Item.Channel, event.Event.Item.Ts)
	if err != nil {
		log.Printf("Error getting message metadata: %v", err)
		return
	}

	if metadata == nil {
		log.Println("No PR metadata found in message, skipping")
		return
	}

	log.Printf("Found PR metadata: %s #%d (branch: %s)", metadata.Repository, metadata.PRNumber, metadata.Branch)

	// Create and publish Poppit command
	poppitCmd := createPoppitCommand(metadata, config)
	if err := publishPoppitCommand(ctx, redisClient, poppitCmd, config); err != nil {
		log.Printf("Error publishing Poppit command: %v", err)
		return
	}

	log.Printf("Successfully published Poppit command for %s branch %s", metadata.Repository, metadata.Branch)
}

func getMessageMetadata(slackClient *slack.Client, channel, timestamp string) (*PRMetadata, error) {
	// Fetch the message
	historyParams := &slack.GetConversationHistoryParameters{
		ChannelID: channel,
		Latest:    timestamp,
		Inclusive: true,
		Limit:     1,
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
	if message.Metadata.EventPayload == nil || len(message.Metadata.EventPayload) == 0 {
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

func createPoppitCommand(metadata *PRMetadata, config Config) PoppitCommand {
	dir := fmt.Sprintf("%s/%s", config.BaseDir, metadata.Repository)

	return PoppitCommand{
		Repo:   metadata.Repository,
		Branch: metadata.Branch,
		Type:   "vibe-deploy",
		Dir:    dir,
		Commands: []string{
			fmt.Sprintf("git checkout %s", metadata.Branch),
			"docker compose build",
			"docker compose down",
			"docker compose up -d",
			"git checkout main",
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
