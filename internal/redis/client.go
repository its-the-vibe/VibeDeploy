package redis

import (
	"context"
	"encoding/json"
	"fmt"

	redisapi "github.com/redis/go-redis/v9"

	"github.com/its-the-vibe/VibeDeploy/internal/config"
	"github.com/its-the-vibe/VibeDeploy/internal/domain"
)

type Client struct {
	client *redisapi.Client
}

func NewClient(config config.Config) *Client {
	return &Client{client: redisapi.NewClient(&redisapi.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
	})}
}

func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Client) Subscribe(ctx context.Context, channel string) *redisapi.PubSub {
	return c.client.Subscribe(ctx, channel)
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) PublishPoppitCommand(ctx context.Context, cmd domain.PoppitCommand, config config.Config) error {
	payload, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal Poppit command: %w", err)
	}

	if err := c.client.RPush(ctx, config.RedisListName, payload).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis list: %w", err)
	}

	return nil
}

func (c *Client) PublishSlackReaction(ctx context.Context, channel, timestamp, reaction string, remove bool, config config.Config) error {
	slackReaction := domain.SlackReaction{
		Reaction: reaction,
		Channel:  channel,
		Ts:       timestamp,
		Remove:   remove,
	}

	payload, err := json.Marshal(slackReaction)
	if err != nil {
		return fmt.Errorf("failed to marshal slack reaction: %w", err)
	}

	if err := c.client.RPush(ctx, config.RedisReactionList, payload).Err(); err != nil {
		return fmt.Errorf("failed to push to Redis list: %w", err)
	}

	return nil
}
