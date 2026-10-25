package deployment

import (
	"reflect"
	"testing"

	"github.com/its-the-vibe/VibeDeploy/internal/config"
	"github.com/its-the-vibe/VibeDeploy/internal/domain"
)

func TestRepositoryRules(t *testing.T) {
	if !IsRepoAllowed("any/repo", nil) {
		t.Error("IsRepoAllowed() should allow all repositories when the allowlist is nil")
	}
	if !IsRepoAllowed("its-the-vibe/VibeMerge", map[string]bool{"its-the-vibe/VibeMerge": true}) {
		t.Error("IsRepoAllowed() should allow a listed repository")
	}
	if IsRepoAllowed("its-the-vibe/Other", map[string]bool{"its-the-vibe/VibeMerge": true}) {
		t.Error("IsRepoAllowed() should reject an unlisted repository")
	}
	if IsLegacyDockerApp("any/repo", nil) {
		t.Error("IsLegacyDockerApp() should reject nil configuration")
	}
	if !IsLegacyDockerApp("its-the-vibe/OldApp", map[string]bool{"its-the-vibe/OldApp": true}) {
		t.Error("IsLegacyDockerApp() should identify a listed legacy repository")
	}
}

func TestCreatePoppitCommand(t *testing.T) {
	metadata := &domain.PRMetadata{
		Repository: "its-the-vibe/VibeMerge",
		Branch:     "feature/my-branch",
	}
	cfg := config.Config{BaseDir: "/app/repos"}
	cmd := CreatePoppitCommand(metadata, cfg, "C123", "1234567890.123456")

	if cmd.Repo != metadata.Repository || cmd.Branch != metadata.Branch ||
		cmd.Type != domain.VibeDeployType || cmd.Dir != "/app/repos/its-the-vibe/VibeMerge" {
		t.Errorf("unexpected command fields: %+v", cmd)
	}
	expected := []string{
		"git fetch origin",
		"git checkout feature/my-branch",
		"git pull",
		"docker compose build",
		"docker compose down",
		domain.DeploymentCommand,
	}
	if !reflect.DeepEqual(cmd.Commands, expected) {
		t.Errorf("commands = %#v, want %#v", cmd.Commands, expected)
	}
	if cmd.Metadata == nil || cmd.Metadata.Channel != "C123" || cmd.Metadata.Ts != "1234567890.123456" ||
		cmd.Metadata.TriggerReaction != domain.RocketReaction {
		t.Errorf("unexpected command metadata: %+v", cmd.Metadata)
	}
}

func TestCreateGHAEnabledPoppitCommand(t *testing.T) {
	metadata := &domain.PRMetadata{
		Repository: "its-the-vibe/VibeMerge",
		PRUrl:      "https://github.com/its-the-vibe/VibeMerge/pull/42",
		Branch:     "feature/my-branch",
	}
	cfg := config.Config{BaseDir: "/app/repos"}
	cmd := CreateGHAEnabledPoppitCommand(metadata, cfg, "C123", "1234567890.123456")

	if cmd.Commands[2] != config.DefaultDockerOverride+" create --override-tag feature" {
		t.Errorf("default override command = %q", cmd.Commands[2])
	}
	cfg.DockerOverride = "/custom/bin/docker-override"
	cmd = CreateGHAEnabledPoppitCommand(metadata, cfg, "C123", "1234567890.123456")
	expected := []string{
		"git fetch",
		"git checkout feature/my-branch",
		"/custom/bin/docker-override create --override-tag feature",
		`gh label create "feature" --color "f107a3" --force --repo its-the-vibe/VibeMerge`,
		`gh pr edit --remove-label "feature" https://github.com/its-the-vibe/VibeMerge/pull/42`,
		"sleep 5",
		`gh pr edit --add-label "feature" https://github.com/its-the-vibe/VibeMerge/pull/42`,
	}
	if !reflect.DeepEqual(cmd.Commands, expected) {
		t.Errorf("commands = %#v, want %#v", cmd.Commands, expected)
	}
}

func TestCreateMainBranchPoppitCommand(t *testing.T) {
	metadata := &domain.PRMetadata{Repository: "its-the-vibe/VibeMerge", Branch: "feature/my-branch"}
	cmd := CreateMainBranchPoppitCommand(metadata, config.Config{BaseDir: "/repos"}, "C456", "9999999999.000001")

	if cmd.Branch != "main" || cmd.Dir != "/repos/its-the-vibe/VibeMerge" ||
		cmd.Type != domain.VibeDeployType || cmd.Metadata == nil ||
		cmd.Metadata.TriggerReaction != domain.ClassicalBuildingReaction {
		t.Errorf("unexpected main branch command: %+v", cmd)
	}
}
