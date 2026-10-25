package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		level    LogLevel
		expected string
	}{
		{"DEBUG", DEBUG, "DEBUG"},
		{"debug", DEBUG, "DEBUG"},
		{"INFO", INFO, "INFO"},
		{"warn", WARN, "WARN"},
		{"ERROR", ERROR, "ERROR"},
		{"invalid", INFO, "INFO"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			if got := ParseLogLevel(test.input); got != test.level {
				t.Errorf("ParseLogLevel(%q) = %v, want %v", test.input, got, test.level)
			}
			if got := test.level.String(); got != test.expected {
				t.Errorf("LogLevel(%d).String() = %q, want %q", test.level, got, test.expected)
			}
		})
	}

	if got := LogLevel(99).String(); got != "UNKNOWN" {
		t.Errorf("LogLevel(99).String() = %q, want UNKNOWN", got)
	}
}

func TestGetEnv(t *testing.T) {
	t.Setenv("VIBEDEPLOY_CONFIG_TEST", "configured")
	if got := GetEnv("VIBEDEPLOY_CONFIG_TEST", "default"); got != "configured" {
		t.Errorf("GetEnv() = %q, want configured", got)
	}

	t.Setenv("VIBEDEPLOY_CONFIG_TEST", "")
	if got := GetEnv("VIBEDEPLOY_CONFIG_TEST", "fallback"); got != "fallback" {
		t.Errorf("GetEnv() = %q, want fallback", got)
	}
	if got := GetEnv("VIBEDEPLOY_CONFIG_MISSING", "default"); got != "default" {
		t.Errorf("GetEnv() = %q, want default", got)
	}
}

func TestLoadFromFile(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "redis-secret")
	t.Setenv("SLACK_BOT_TOKEN", "slack-secret")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte(`
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
`)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if cfg.RedisAddr != "redis.internal:6379" ||
		cfg.RedisPubSub != "test-events" ||
		cfg.RedisListName != "test-commands" ||
		cfg.RedisOutputChannel != "test-output" ||
		cfg.RedisReactionList != "test-reactions" ||
		cfg.BaseDir != "/srv/repos" ||
		cfg.LogLevel != DEBUG ||
		cfg.DockerOverride != "/opt/bin/docker-override" {
		t.Errorf("unexpected loaded configuration: %+v", cfg)
	}
	if cfg.RedisPassword != "redis-secret" || cfg.SlackToken != "slack-secret" {
		t.Errorf("credentials were not loaded from the environment")
	}
	if !cfg.AllowedRepos["its-the-vibe/VibeMerge"] || !cfg.LegacyDockerApps["its-the-vibe/OldApp"] {
		t.Errorf("repository lists were not loaded: allowed=%v legacy=%v", cfg.AllowedRepos, cfg.LegacyDockerApps)
	}
}

func TestLoadFromFileDefaultsAndErrors(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("SLACK_BOT_TOKEN", "")

	cfg, err := LoadFromFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadFromFile() for missing file error = %v", err)
	}
	if cfg.RedisAddr != "localhost:6379" ||
		cfg.RedisPubSub != "slack-relay-reaction-added" ||
		cfg.RedisListName != "poppit-commands" ||
		cfg.RedisOutputChannel != "poppit:command-output" ||
		cfg.RedisReactionList != "slack_reactions" ||
		cfg.BaseDir != "/app/repos" ||
		cfg.LogLevel != INFO ||
		cfg.DockerOverride != DefaultDockerOverride ||
		cfg.AllowedRepos != nil ||
		cfg.LegacyDockerApps != nil {
		t.Errorf("unexpected defaults: %+v", cfg)
	}

	invalidPath := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(invalidPath, []byte(": invalid: yaml: [\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromFile(invalidPath); err == nil {
		t.Error("LoadFromFile() with invalid YAML returned no error")
	}

	if _, err := LoadFromFile(t.TempDir()); err == nil {
		t.Error("LoadFromFile() with a directory returned no error")
	}
}

func TestLoadUsesConfigFileEnvironmentVariable(t *testing.T) {
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("SLACK_BOT_TOKEN", "")
	configPath := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(configPath, []byte("redis:\n  addr: redis.custom:6379\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", configPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.RedisAddr != "redis.custom:6379" {
		t.Errorf("RedisAddr = %q, want redis.custom:6379", cfg.RedisAddr)
	}
}
