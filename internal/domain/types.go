package domain

const (
	RocketReaction                = "rocket"
	GearReaction                  = "gear"
	ClassicalBuildingReaction     = "classical_building"
	VibeDeployType                = "vibe-deploy"
	DeploymentCommand             = "docker compose up -d"
	DockerOverrideCommandTemplate = "[dockerOverride] create --override-tag feature"
)

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

type PoppitCommand struct {
	Repo     string           `json:"repo"`
	Branch   string           `json:"branch"`
	Type     string           `json:"type"`
	Dir      string           `json:"dir"`
	Commands []string         `json:"commands"`
	Metadata *CommandMetadata `json:"metadata,omitempty"`
}

type CommandMetadata struct {
	Channel         string `json:"channel"`
	Ts              string `json:"ts"`
	TriggerReaction string `json:"trigger_reaction,omitempty"`
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
