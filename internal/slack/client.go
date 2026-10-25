package slack

import (
	"encoding/json"
	"fmt"

	slackapi "github.com/slack-go/slack"

	"github.com/its-the-vibe/VibeDeploy/internal/domain"
)

type Client struct {
	client *slackapi.Client
}

func NewClient(token string) *Client {
	return &Client{client: slackapi.New(token)}
}

func (c *Client) GetMessageMetadata(channel, timestamp string) (*domain.PRMetadata, error) {
	historyParams := &slackapi.GetConversationHistoryParameters{
		ChannelID:          channel,
		Latest:             timestamp,
		Inclusive:          true,
		Limit:              1,
		IncludeAllMetadata: true,
	}

	history, err := c.client.GetConversationHistory(historyParams)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	if len(history.Messages) == 0 {
		return nil, fmt.Errorf("no messages found")
	}

	message := history.Messages[0]
	if len(message.Metadata.EventPayload) == 0 {
		return nil, nil
	}

	metadataJSON, err := json.Marshal(message.Metadata.EventPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var metadata domain.PRMetadata
	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse PR metadata: %w", err)
	}

	if metadata.Repository == "" || metadata.Branch == "" {
		return nil, nil
	}

	return &metadata, nil
}
