package main

import (
	"os"
	"testing"
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
			got := parseLogLevel(tt.input)
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

		got := getEnv("TEST_VAR_VIBEDEPLOY", "default")
		if got != "testvalue" {
			t.Errorf("getEnv() = %q, want %q", got, "testvalue")
		}
	})

	t.Run("returns default when env var not set", func(t *testing.T) {
		os.Unsetenv("TEST_VAR_VIBEDEPLOY_MISSING")

		got := getEnv("TEST_VAR_VIBEDEPLOY_MISSING", "mydefault")
		if got != "mydefault" {
			t.Errorf("getEnv() = %q, want %q", got, "mydefault")
		}
	})

	t.Run("returns default when env var is empty string", func(t *testing.T) {
		os.Setenv("TEST_VAR_VIBEDEPLOY_EMPTY", "")
		defer os.Unsetenv("TEST_VAR_VIBEDEPLOY_EMPTY")

		got := getEnv("TEST_VAR_VIBEDEPLOY_EMPTY", "fallback")
		if got != "fallback" {
			t.Errorf("getEnv() = %q, want %q", got, "fallback")
		}
	})
}

func TestIsLegacyDockerApp(t *testing.T) {
	t.Run("nil legacyApps returns false", func(t *testing.T) {
		if isLegacyDockerApp("any/repo", nil) {
			t.Error("isLegacyDockerApp() should return false when legacyApps is nil")
		}
	})

	t.Run("repo in legacyApps is identified as legacy", func(t *testing.T) {
		legacy := map[string]bool{"its-the-vibe/OldApp": true}
		if !isLegacyDockerApp("its-the-vibe/OldApp", legacy) {
			t.Error("isLegacyDockerApp() should return true for a listed legacy repo")
		}
	})

	t.Run("repo not in legacyApps is not legacy", func(t *testing.T) {
		legacy := map[string]bool{"its-the-vibe/OldApp": true}
		if isLegacyDockerApp("its-the-vibe/NewApp", legacy) {
			t.Error("isLegacyDockerApp() should return false for an unlisted repo")
		}
	})
}

func TestIsRepoAllowed(t *testing.T) {
	t.Run("nil allowlist permits all repos", func(t *testing.T) {
		if !isRepoAllowed("any/repo", nil) {
			t.Error("isRepoAllowed() should return true when allowedRepos is nil")
		}
	})

	t.Run("repo in allowlist is permitted", func(t *testing.T) {
		allowed := map[string]bool{"its-the-vibe/VibeMerge": true}
		if !isRepoAllowed("its-the-vibe/VibeMerge", allowed) {
			t.Error("isRepoAllowed() should return true for a listed repo")
		}
	})

	t.Run("repo not in allowlist is denied", func(t *testing.T) {
		allowed := map[string]bool{"its-the-vibe/VibeMerge": true}
		if isRepoAllowed("its-the-vibe/Other", allowed) {
			t.Error("isRepoAllowed() should return false for an unlisted repo")
		}
	})

	t.Run("empty allowlist denies all repos", func(t *testing.T) {
		allowed := map[string]bool{}
		if isRepoAllowed("its-the-vibe/VibeMerge", allowed) {
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

		cmd := createGHAEnabledPoppitCommand(metadata, config, channel, timestamp)

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

		cmd := createGHAEnabledPoppitCommand(metadata, config, channel, timestamp)

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

	cmd := createPoppitCommand(metadata, config, channel, timestamp)

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

	cmd := createMainBranchPoppitCommand(metadata, config, channel, timestamp)

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

func TestLoadVibeDeployConfig(t *testing.T) {
	t.Run("loads combined config file correctly", func(t *testing.T) {
		f, err := os.CreateTemp("", "vibe-config-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		content := `
allowedRepos:
  - its-the-vibe/VibeMerge
  - its-the-vibe/Poppit

legacyDockerApps:
  - its-the-vibe/OldApp1

dockerOverride: /opt/bin/docker-override
`
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		f.Close()

		cfg := Config{
			VibeDeployConfig: f.Name(),
		}

		allowed, legacy, override, err := loadVibeDeployConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error loading config: %v", err)
		}

		if !allowed["its-the-vibe/VibeMerge"] || !allowed["its-the-vibe/Poppit"] {
			t.Errorf("allowedRepos missing expected values: %v", allowed)
		}
		if !legacy["its-the-vibe/OldApp1"] {
			t.Errorf("legacyApps missing expected values: %v", legacy)
		}
		if override != "/opt/bin/docker-override" {
			t.Errorf("dockerOverride = %q, want %q", override, "/opt/bin/docker-override")
		}
	})

	t.Run("falls back to default dockerOverride path if not specified in YAML", func(t *testing.T) {
		f, err := os.CreateTemp("", "vibe-config-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		content := `
allowedRepos:
  - its-the-vibe/VibeMerge
`
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		f.Close()

		cfg := Config{
			VibeDeployConfig: f.Name(),
		}

		_, _, override, err := loadVibeDeployConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if override != DefaultDockerOverride {
			t.Errorf("dockerOverride = %q, want default %q", override, DefaultDockerOverride)
		}
	})

	t.Run("returns defaults when no config file specified or present", func(t *testing.T) {
		cfg := Config{}
		allowed, legacy, override, err := loadVibeDeployConfig(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed != nil {
			t.Errorf("expected nil allowed repos")
		}
		if legacy != nil {
			t.Errorf("expected nil legacy apps")
		}
		if override != DefaultDockerOverride {
			t.Errorf("expected default docker override path")
		}
	})

	t.Run("returns error for invalid combined YAML", func(t *testing.T) {
		f, err := os.CreateTemp("", "bad-vibe-config-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		f.WriteString(": invalid: yaml: [\n")
		f.Close()

		cfg := Config{
			VibeDeployConfig: f.Name(),
		}

		_, _, _, err = loadVibeDeployConfig(cfg)
		if err == nil {
			t.Error("expected error for invalid combined YAML, got nil")
		}
	})
}
