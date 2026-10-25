package deployment

import (
	"fmt"
	"strings"

	"github.com/its-the-vibe/VibeDeploy/internal/config"
	"github.com/its-the-vibe/VibeDeploy/internal/domain"
)

func IsRepoAllowed(repo string, allowedRepos map[string]bool) bool {
	if allowedRepos == nil {
		return true
	}
	return allowedRepos[repo]
}

func IsLegacyDockerApp(repo string, legacyApps map[string]bool) bool {
	if legacyApps == nil {
		return false
	}
	return legacyApps[repo]
}

func CreatePoppitCommand(metadata *domain.PRMetadata, cfg config.Config, channel, timestamp string) domain.PoppitCommand {
	dir := fmt.Sprintf("%s/%s", cfg.BaseDir, metadata.Repository)

	return domain.PoppitCommand{
		Repo:   metadata.Repository,
		Branch: metadata.Branch,
		Type:   domain.VibeDeployType,
		Dir:    dir,
		Commands: []string{
			"git fetch origin",
			fmt.Sprintf("git checkout %s", metadata.Branch),
			"git pull",
			"docker compose build",
			"docker compose down",
			domain.DeploymentCommand,
		},
		Metadata: commandMetadata(channel, timestamp, domain.RocketReaction),
	}
}

func CreateGHAEnabledPoppitCommand(metadata *domain.PRMetadata, cfg config.Config, channel, timestamp string) domain.PoppitCommand {
	dir := fmt.Sprintf("%s/%s", cfg.BaseDir, metadata.Repository)

	dockerOverridePath := cfg.DockerOverride
	if dockerOverridePath == "" {
		dockerOverridePath = config.DefaultDockerOverride
	}

	dockerOverrideCmd := strings.ReplaceAll(domain.DockerOverrideCommandTemplate, "[dockerOverride]", dockerOverridePath)

	return domain.PoppitCommand{
		Repo:   metadata.Repository,
		Branch: metadata.Branch,
		Type:   domain.VibeDeployType,
		Dir:    dir,
		Commands: []string{
			"git fetch",
			fmt.Sprintf("git checkout %s", metadata.Branch),
			dockerOverrideCmd,
			fmt.Sprintf("gh label create \"feature\" --color \"f107a3\" --force --repo %s", metadata.Repository),
			// idempotently remove label "feature", useful if we want to retrigger a workflow run
			fmt.Sprintf("gh pr edit --remove-label \"feature\" %s", metadata.PRUrl),
			"sleep 5",
			fmt.Sprintf("gh pr edit --add-label \"feature\" %s", metadata.PRUrl),
		},
		Metadata: commandMetadata(channel, timestamp, domain.RocketReaction),
	}
}

func CreateMainBranchPoppitCommand(metadata *domain.PRMetadata, cfg config.Config, channel, timestamp string) domain.PoppitCommand {
	dir := fmt.Sprintf("%s/%s", cfg.BaseDir, metadata.Repository)

	return domain.PoppitCommand{
		Repo:   metadata.Repository,
		Branch: "main",
		Type:   domain.VibeDeployType,
		Dir:    dir,
		Commands: []string{
			"git checkout main",
			"git pull",
			"docker compose build",
			"docker compose down",
			domain.DeploymentCommand,
		},
		Metadata: commandMetadata(channel, timestamp, domain.ClassicalBuildingReaction),
	}
}

func commandMetadata(channel, timestamp, reaction string) *domain.CommandMetadata {
	return &domain.CommandMetadata{
		Channel:         channel,
		Ts:              timestamp,
		TriggerReaction: reaction,
	}
}
