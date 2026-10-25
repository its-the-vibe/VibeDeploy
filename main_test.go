package main

import (
	"os"
	"testing"

	appconfig "github.com/its-the-vibe/VibeDeploy/internal/config"
	"github.com/its-the-vibe/VibeDeploy/internal/deployment"
	"github.com/its-the-vibe/VibeDeploy/internal/domain"
)

type Config = appconfig.Config
type LogLevel = appconfig.LogLevel
type PRMetadata = domain.PRMetadata

const (
	DEBUG                     = appconfig.DEBUG
	INFO                      = appconfig.INFO
	WARN                      = appconfig.WARN
	ERROR                     = appconfig.ERROR
	DefaultConfigFile         = appconfig.DefaultConfigFile
	DefaultDockerOverride     = appconfig.DefaultDockerOverride
	RocketReaction            = domain.RocketReaction
	ClassicalBuildingReaction = domain.ClassicalBuildingReaction
	VibeDeployType            = domain.VibeDeployType
)

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{LogLevel(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"warn", WARN},
		{"ERROR", ERROR},
		{"error", ERROR},
		{"", INFO},
		{"INVALID", INFO},
		{"verbose", INFO},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := appconfig.ParseLogLevel(tt.input)
			if got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	t.Run("returns env var when set", func(t *testing.T) {
		os.Setenv("TEST_VAR_VIBEDEPLOY", "testvalue")
		defer os.Unsetenv("TEST_VAR_VIBEDEPLOY")

		got := appconfig.GetEnv("TEST_VAR_VIBEDEPLOY", "default")
		if got != "testvalue" {
			t.Errorf("getEnv() = %q, want %q", got, "testvalue")
		}
	})

	t.Run("returns default when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR_VIBEDEPLOY_MISSING")

		got := appconfig.GetEnv("TEST_VAR_VIBEDEPLOY_MISSING", "mydefault")
		if got != "mydefault" {
			t.Errorf("getEnv() = %q, want %q", got, "mydefault")
		}
	})

	t.Run("returns default when env var is empty string", func(t *testing.T) {
		os.Setenv("TEST_VAR_VIBEDEPLOY_EMPTY", "")
		defer os.Unsetenv("TEST_VAR_VIBEDEPLOY_EMPTY")

		got := appconfig.GetEnv("TEST_VAR_VIBEDEPLOY_EMPTY", "fallback")
		if got != "fallback" {
			t.Errorf("getEnv() = %q, want %q", got, "fallback")
		}
	})
}

func TestIsLegacyDockerApp(t *testing.T) {
	t.Run("nil legacyApps returns false", func(t *testing.T) {
		if deployment.IsLegacyDockerApp("any/repo", nil) {
			t.Error("isLegacyDockerApp() should return false when legacyApps is nil")
		}
	})

	t.Run("repo in legacyApps is identified as legacy", func(t *testing.T) {
		legacy := map[string]bool{"its-the-vibe/OldApp": true}
		if !deployment.IsLegacyDockerApp("its-the-vibe/OldApp", legacy) {
			t.Error("isLegacyDockerApp() should return true for a listed legacy repo")
		}
	})

	t.Run("repo not in legacyApps is not legacy", func(t *testing.T) {
		legacy := map[string]bool{"its-the-vibe/OldApp": true}
		if deployment.IsLegacyDockerApp("its-the-vibe/NewApp", legacy) {
			t.Error("isLegacyDockerApp() should return false for an unlisted repo")
		}
	})
}

func TestIsRepoAllowed(t *testing.T) {
	t.Run("nil allowlist permits all repos", func(t *testing.T) {
		if !deployment.IsRepoAllowed("any/repo", nil) {
			t.Error("isRepoAllowed() should return true when allowedRepos is nil")
		}
	})

	t.Run("repo in allowlist is permitted", func(t *testing.T) {
		allowed := map[string]bool{"its-the-vibe/VibeMerge": true}
		if !deployment.IsRepoAllowed("its-the-vibe/VibeMerge", allowed) {
			t.Error("isRepoAllowed() should return true for a listed repo")
		}
	})

	t.Run("repo not in allowlist is denied", func(t *testing.T) {
		allowed := map[string]bool{"its-the-vibe/VibeMerge": true}
		if deployment.IsRepoAllowed("its-the-vibe/Other", allowed) {
			t.Error("isRepoAllowed() should return false for an unlisted repo")
		}
	})

	t.Run("empty allowlist denies all repos", func(t *testing.T) {
		allowed := map[string]bool{}
		if deployment.IsRepoAllowed("its-the-vibe/VibeMerge", allowed) {
			t.Error("isRepoAllowed() should return false when allowlist is empty")
		}
	})
}

func TestCreateGHAEnabledPoppitCommand(t *testing.T) {
	t.Run("default dockerOverride path when empty in config", func(t *testing.T) {
		metadata := &PRMetadata{
			PRNumber:   42,
			Repository: "its-the-vibe/VibeMerge",
			PRUrl:      "https://github.com/its-the-vibe/VibeMerge/pull/42",
			Branch:     "feature/my-branch",
		}
		config := Config{
			BaseDir:       "/app/repos",
			RedisListName: "poppit-commands",
		}
		channel := "C123"
		timestamp := "1234567890.123456"

		cmd := deployment.CreateGHAEnabledPoppitCommand(metadata, config, channel, timestamp)

		expectedCommands := []string{
			"git fetch",
			"git checkout feature/my-branch",
			"../vibebox/docker-override/docker-override create --override-tag feature",
			"gh label create \"feature\" --color \"f107a3\" --force --repo its-the-vibe/VibeMerge",
			"gh pr edit --add-label \"feature\" https://github.com/its-the-vibe/VibeMerge/pull/42",
		}

		if len(cmd.Commands) != len(expectedCommands) {
			t.Fatalf("Commands length = %d, want %d", len(cmd.Commands), len(expectedCommands))
		}

		for i, expected := range expectedCommands {
			if cmd.Commands[i] != expected {
				t.Errorf("Command[%d] = %q, want %q", i, cmd.Commands[i], expected)
			}
		}
	})

	t.Run("custom dockerOverride path injected correctly", func(t *testing.T) {
		metadata := &PRMetadata{
			PRNumber:   42,
			Repository: "its-the-vibe/VibeMerge",
			PRUrl:      "https://github.com/its-the-vibe/VibeMerge/pull/42",
			Branch:     "feature/my-branch",
		}
		config := Config{
			BaseDir:        "/app/repos",
			RedisListName:  "poppit-commands",
			DockerOverride: "/custom/bin/docker-override",
		}
		channel := "C123"
		timestamp := "1234567890.123456"

		cmd := deployment.CreateGHAEnabledPoppitCommand(metadata, config, channel, timestamp)

		expectedOverrideCmd := "/custom/bin/docker-override create --override-tag feature"
		if cmd.Commands[2] != expectedOverrideCmd {
			t.Errorf("Command[2] = %q, want %q", cmd.Commands[2], expectedOverrideCmd)
		}
	})
}

func TestCreatePoppitCommand(t *testing.T) {
	metadata := &PRMetadata{
		PRNumber:   42,
		Repository: "its-the-vibe/VibeMerge",
		Branch:     "feature/my-branch",
	}
	config := Config{
		BaseDir:       "/app/repos",
		RedisListName: "poppit-commands",
	}
	channel := "C123"
	timestamp := "1234567890.123456"

	cmd := deployment.CreatePoppitCommand(metadata, config, channel, timestamp)

	if cmd.Repo != metadata.Repository {
		t.Errorf("Repo = %q, want %q", cmd.Repo, metadata.Repository)
	}
	if cmd.Branch != metadata.Branch {
		t.Errorf("Branch = %q, want %q", cmd.Branch, metadata.Branch)
	}
	if cmd.Type != VibeDeployType {
		t.Errorf("Type = %q, want %q", cmd.Type, VibeDeployType)
	}
	if cmd.Dir != "/app/repos/its-the-vibe/VibeMerge" {
		t.Errorf("Dir = %q, want %q", cmd.Dir, "/app/repos/its-the-vibe/VibeMerge")
	}
	if len(cmd.Commands) == 0 {
		t.Error("Commands should not be empty")
	}
	if cmd.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if cmd.Metadata.Channel != channel {
		t.Errorf("Metadata.Channel = %q, want %q", cmd.Metadata.Channel, channel)
	}
	if cmd.Metadata.Ts != timestamp {
		t.Errorf("Metadata.Ts = %q, want %q", cmd.Metadata.Ts, timestamp)
	}
	if cmd.Metadata.TriggerReaction != RocketReaction {
		t.Errorf("Metadata.TriggerReaction = %q, want %q", cmd.Metadata.TriggerReaction, RocketReaction)
	}
}

func TestCreateMainBranchPoppitCommand(t *testing.T) {
	metadata := &PRMetadata{
		PRNumber:   7,
		Repository: "its-the-vibe/VibeMerge",
		Branch:     "feature/some-branch",
	}
	config := Config{
		BaseDir:       "/repos",
		RedisListName: "poppit-commands",
	}
	channel := "C456"
	timestamp := "9999999999.000001"

	cmd := deployment.CreateMainBranchPoppitCommand(metadata, config, channel, timestamp)

	if cmd.Repo != metadata.Repository {
		t.Errorf("Repo = %q, want %q", cmd.Repo, metadata.Repository)
	}
	if cmd.Branch != "main" {
		t.Errorf("Branch = %q, want %q", cmd.Branch, "main")
	}
	if cmd.Type != VibeDeployType {
		t.Errorf("Type = %q, want %q", cmd.Type, VibeDeployType)
	}
	if cmd.Dir != "/repos/its-the-vibe/VibeMerge" {
		t.Errorf("Dir = %q, want %q", cmd.Dir, "/repos/its-the-vibe/VibeMerge")
	}
	if len(cmd.Commands) == 0 {
		t.Error("Commands should not be empty")
	}
	if cmd.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}
	if cmd.Metadata.TriggerReaction != ClassicalBuildingReaction {
		t.Errorf("Metadata.TriggerReaction = %q, want %q", cmd.Metadata.TriggerReaction, ClassicalBuildingReaction)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "redis-secret")
	t.Setenv("SLACK_BOT_TOKEN", "slack-secret")

	t.Run("loads configuration values from YAML", func(t *testing.T) {
		configFile := t.TempDir() + "/config.yaml"
		content := `
redis:
  addr: redis.internal:6379
  pubsub_channel: test-events
  list_name: test-commands
  output_channel: test-output
  reaction_list: test-reactions
base_dir: /srv/repos
log_level: debug
allowed_repos:
  - its-the-vibe/VibeMerge
legacy_docker_apps:
  - its-the-vibe/OldApp
docker_override: /opt/bin/docker-override
`
		if err := os.WriteFile(configFile, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}

		config, err := appconfig.LoadFromFile(configFile)
		if err != nil {
			t.Fatalf("unexpected error loading config: %v", err)
		}
		if config.RedisAddr != "redis.internal:6379" ||
			config.RedisPubSub != "test-events" ||
			config.RedisListName != "test-commands" ||
			config.RedisOutputChannel != "test-output" ||
			config.RedisReactionList != "test-reactions" ||
			config.BaseDir != "/srv/repos" ||
			config.LogLevel != DEBUG ||
			config.DockerOverride != "/opt/bin/docker-override" {
			t.Errorf("unexpected config values: %+v", config)
		}
		if config.RedisPassword != "redis-secret" || config.SlackToken != "slack-secret" {
			t.Errorf("sensitive settings were not loaded from the environment")
		}
		if !config.AllowedRepos["its-the-vibe/VibeMerge"] {
			t.Errorf("allowed repos missing expected value: %v", config.AllowedRepos)
		}
		if !config.LegacyDockerApps["its-the-vibe/OldApp"] {
			t.Errorf("legacy apps missing expected value: %v", config.LegacyDockerApps)
		}
	})

	t.Run("uses defaults when config file is missing", func(t *testing.T) {
		config, err := appconfig.LoadFromFile(t.TempDir() + "/missing.yaml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config.RedisAddr != "localhost:6379" ||
			config.RedisPubSub != "slack-relay-reaction-added" ||
			config.RedisListName != "poppit-commands" ||
			config.RedisOutputChannel != "poppit:command-output" ||
			config.RedisReactionList != "slack_reactions" ||
			config.BaseDir != "/app/repos" ||
			config.LogLevel != INFO ||
			config.DockerOverride != DefaultDockerOverride {
			t.Errorf("unexpected defaults: %+v", config)
		}
		if config.AllowedRepos != nil || config.LegacyDockerApps != nil {
			t.Errorf("expected nil repo lists by default, got allowed=%v legacy=%v", config.AllowedRepos, config.LegacyDockerApps)
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		configFile := t.TempDir() + "/invalid.yaml"
		if err := os.WriteFile(configFile, []byte(": invalid: yaml: [\n"), 0600); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}
		if _, err := appconfig.LoadFromFile(configFile); err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}

func TestLoadConfigUsesConfigFileEnvironmentVariable(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("REDIS_ADDR", "ignored-env-value")
	configFile := t.TempDir() + "/custom.yaml"
	if err := os.WriteFile(configFile, []byte("redis:\n  addr: redis.from-file:6379\n"), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	t.Setenv("CONFIG_FILE", configFile)

	config, err := appconfig.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.RedisAddr != "redis.from-file:6379" {
		t.Errorf("RedisAddr = %q, want config file value", config.RedisAddr)
	}
}

func TestLoadConfigDefaultsToConfigYaml(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("SLACK_BOT_TOKEN", "")
	t.Setenv("CONFIG_FILE", "")
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(DefaultConfigFile, []byte("redis:\n  addr: redis.default-file:6379\n"), 0600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	config, err := appconfig.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.RedisAddr != "redis.default-file:6379" {
		t.Errorf("RedisAddr = %q, want default config file value", config.RedisAddr)
	}
}
