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

func TestLoadAllowedRepos(t *testing.T) {
	t.Run("empty config path returns nil", func(t *testing.T) {
		repos, err := loadAllowedRepos("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repos != nil {
			t.Error("expected nil allowedRepos for empty config path")
		}
	})

	t.Run("non-existent file returns nil", func(t *testing.T) {
		repos, err := loadAllowedRepos("/tmp/nonexistent-vibedeploy-test.yml")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repos != nil {
			t.Error("expected nil allowedRepos for missing config file")
		}
	})

	t.Run("valid yaml file is loaded correctly", func(t *testing.T) {
		f, err := os.CreateTemp("", "allowed-repos-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		content := "allowed_repos:\n  - its-the-vibe/VibeMerge\n  - its-the-vibe/Poppit\n"
		if _, err := f.WriteString(content); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		f.Close()

		repos, err := loadAllowedRepos(f.Name())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repos == nil {
			t.Fatal("expected non-nil allowedRepos")
		}
		if !repos["its-the-vibe/VibeMerge"] {
			t.Error("expected its-the-vibe/VibeMerge to be allowed")
		}
		if !repos["its-the-vibe/Poppit"] {
			t.Error("expected its-the-vibe/Poppit to be allowed")
		}
		if repos["its-the-vibe/Other"] {
			t.Error("expected its-the-vibe/Other not to be allowed")
		}
	})

	t.Run("invalid yaml returns error", func(t *testing.T) {
		f, err := os.CreateTemp("", "bad-yaml-*.yml")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		if _, err := f.WriteString(": invalid: yaml: [\n"); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}
		f.Close()

		_, err = loadAllowedRepos(f.Name())
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})
}
