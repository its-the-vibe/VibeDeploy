package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/its-the-vibe/VibeDeploy/internal/config"
	redisclient "github.com/its-the-vibe/VibeDeploy/internal/redis"
	slackclient "github.com/its-the-vibe/VibeDeploy/internal/slack"
)

var currentLogLevel = config.INFO

func logDebug(format string, v ...interface{}) {
	if currentLogLevel <= config.DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func logInfo(format string, v ...interface{}) {
	if currentLogLevel <= config.INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

func logWarn(format string, v ...interface{}) {
	if currentLogLevel <= config.WARN {
		log.Printf("[WARN] "+format, v...)
	}
}

func logError(format string, v ...interface{}) {
	if currentLogLevel <= config.ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	currentLogLevel = cfg.LogLevel

	if cfg.SlackToken == "" {
		log.Fatal("SLACK_BOT_TOKEN environment variable is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redisClient := redisclient.NewClient(cfg)
	defer redisClient.Close()

	if err := redisClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	logInfo("Connected to Redis at %s", cfg.RedisAddr)

	slackClient := slackclient.NewClient(cfg.SlackToken)

	pubsub := redisClient.Subscribe(ctx, cfg.RedisPubSub)
	defer pubsub.Close()

	logInfo("Subscribed to Redis channel: %s (log level: %s)", cfg.RedisPubSub, cfg.LogLevel.String())

	go listenForCommandOutput(ctx, redisClient, cfg)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logInfo("Shutting down...")
		cancel()
	}()

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
			logDebug("Received message from channel: %s", cfg.RedisPubSub)
			processReactionEvent(ctx, msg.Payload, slackClient, redisClient, cfg)
		}
	}
}
