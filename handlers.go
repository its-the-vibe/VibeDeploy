package main

import (
	"context"
	"encoding/json"

	"github.com/its-the-vibe/VibeDeploy/internal/config"
	"github.com/its-the-vibe/VibeDeploy/internal/deployment"
	"github.com/its-the-vibe/VibeDeploy/internal/domain"
	redisclient "github.com/its-the-vibe/VibeDeploy/internal/redis"
	slackclient "github.com/its-the-vibe/VibeDeploy/internal/slack"
)

func processReactionEvent(ctx context.Context, payload string, slackClient *slackclient.Client, redisClient *redisclient.Client, cfg config.Config) {
	var event domain.ReactionEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		logError("Error parsing reaction event: %v", err)
		return
	}

	reaction := event.Event.Reaction
	if reaction != domain.RocketReaction && reaction != domain.ClassicalBuildingReaction {
		logDebug("Ignoring reaction: %s (not %s or %s)", reaction, domain.RocketReaction, domain.ClassicalBuildingReaction)
		return
	}

	if event.Event.Item.Type != "message" {
		logDebug("Ignoring item type: %s (not message)", event.Event.Item.Type)
		return
	}

	for _, auth := range event.Authorizations {
		if auth.IsBot && auth.UserID == event.Event.User {
			logInfo("Ignoring %s reaction from bot user %s on message %s in channel %s", reaction, event.Event.User, event.Event.Item.Ts, event.Event.Item.Channel)
			return
		}
	}

	logInfo("Processing %s reaction on message %s in channel %s", reaction, event.Event.Item.Ts, event.Event.Item.Channel)

	metadata, err := slackClient.GetMessageMetadata(event.Event.Item.Channel, event.Event.Item.Ts)
	if err != nil {
		logError("Error getting message metadata: %v", err)
		return
	}

	if metadata == nil {
		logDebug("No PR metadata found in message, skipping")
		return
	}

	logInfo("Found PR metadata: %s #%d (branch: %s)", metadata.Repository, metadata.PRNumber, metadata.Branch)

	if !deployment.IsRepoAllowed(metadata.Repository, cfg.AllowedRepos) {
		logInfo("Repository %s is not in the allowed list, ignoring reaction", metadata.Repository)
		return
	}

	if err := redisClient.PublishSlackReaction(ctx, event.Event.Item.Channel, event.Event.Item.Ts, domain.GearReaction, false, cfg); err != nil {
		logError("Error publishing gear reaction: %v", err)
	} else {
		logInfo("Published gear reaction for channel %s, message %s", event.Event.Item.Channel, event.Event.Item.Ts)
	}

	var poppitCmd domain.PoppitCommand
	if reaction == domain.ClassicalBuildingReaction {
		poppitCmd = deployment.CreateMainBranchPoppitCommand(metadata, cfg, event.Event.Item.Channel, event.Event.Item.Ts)
	} else if deployment.IsLegacyDockerApp(metadata.Repository, cfg.LegacyDockerApps) {
		poppitCmd = deployment.CreatePoppitCommand(metadata, cfg, event.Event.Item.Channel, event.Event.Item.Ts)
	} else {
		poppitCmd = deployment.CreateGHAEnabledPoppitCommand(metadata, cfg, event.Event.Item.Channel, event.Event.Item.Ts)
	}
	if err := redisClient.PublishPoppitCommand(ctx, poppitCmd, cfg); err != nil {
		logError("Error publishing Poppit command: %v", err)
		return
	}

	logInfo("Successfully published Poppit command for %s branch %s", metadata.Repository, poppitCmd.Branch)
}

func listenForCommandOutput(ctx context.Context, redisClient *redisclient.Client, cfg config.Config) {
	pubsub := redisClient.Subscribe(ctx, cfg.RedisOutputChannel)
	defer pubsub.Close()

	logInfo("Subscribed to Redis channel: %s", cfg.RedisOutputChannel)

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
			logDebug("Received command output message from channel: %s", cfg.RedisOutputChannel)
			processCommandOutput(ctx, msg.Payload, redisClient, cfg)
		}
	}
}

func processCommandOutput(ctx context.Context, payload string, redisClient *redisclient.Client, cfg config.Config) {
	var output domain.CommandOutput
	if err := json.Unmarshal([]byte(payload), &output); err != nil {
		logError("Error parsing command output: %v", err)
		return
	}

	if output.Type != domain.VibeDeployType {
		logDebug("Ignoring command output type: %s (not %s)", output.Type, domain.VibeDeployType)
		return
	}

	if output.Command != domain.DeploymentCommand {
		logDebug("Ignoring command: %s (not %s)", output.Command, domain.DeploymentCommand)
		return
	}

	if output.Metadata == nil {
		logWarn("Command output missing metadata (channel and timestamp required), cannot send reaction")
		return
	}

	logInfo("Processing completion for %s in channel %s, message %s", domain.VibeDeployType, output.Metadata.Channel, output.Metadata.Ts)

	if err := redisClient.PublishSlackReaction(ctx, output.Metadata.Channel, output.Metadata.Ts, domain.GearReaction, true, cfg); err != nil {
		logError("Error removing gear reaction: %v", err)
	} else {
		logInfo("Removed gear reaction for channel %s, message %s", output.Metadata.Channel, output.Metadata.Ts)
	}

	completionReaction := output.Metadata.TriggerReaction
	if completionReaction == "" {
		completionReaction = domain.RocketReaction
	}

	if err := redisClient.PublishSlackReaction(ctx, output.Metadata.Channel, output.Metadata.Ts, completionReaction, false, cfg); err != nil {
		logError("Error publishing %s reaction: %v", completionReaction, err)
	} else {
		logInfo("Successfully published %s reaction for channel %s, message %s", completionReaction, output.Metadata.Channel, output.Metadata.Ts)
	}
}
