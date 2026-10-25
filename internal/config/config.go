package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RedisAddr          string
	RedisPassword      string
	SlackToken         string
	BaseDir            string
	RedisPubSub        string
	RedisListName      string
	RedisOutputChannel string
	RedisReactionList  string
	LogLevel           LogLevel
	DockerOverride     string
	AllowedRepos       map[string]bool
	LegacyDockerApps   map[string]bool
}

type VibeDeployYamlConfig struct {
	Redis struct {
		Addr          string `yaml:"addr"`
		PubSubChannel string `yaml:"pubsub_channel"`
		ListName      string `yaml:"list_name"`
		OutputChannel string `yaml:"output_channel"`
		ReactionList  string `yaml:"reaction_list"`
	} `yaml:"redis"`
	BaseDir          string   `yaml:"base_dir"`
	LogLevel         string   `yaml:"log_level"`
	AllowedRepos     []string `yaml:"allowed_repos"`
	LegacyDockerApps []string `yaml:"legacy_docker_apps"`
	DockerOverride   string   `yaml:"docker_override"`
}

const (
	DefaultConfigFile     = "config.yaml"
	DefaultDockerOverride = "../vibebox/docker-override/docker-override"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

func Load() (Config, error) {
	return LoadFromFile(GetEnv("CONFIG_FILE", DefaultConfigFile))
}

func LoadFromFile(configPath string) (Config, error) {
	config := Config{
		RedisPassword:      GetEnv("REDIS_PASSWORD", ""),
		SlackToken:         GetEnv("SLACK_BOT_TOKEN", ""),
		RedisAddr:          "localhost:6379",
		BaseDir:            "/app/repos",
		RedisPubSub:        "slack-relay-reaction-added",
		RedisListName:      "poppit-commands",
		RedisOutputChannel: "poppit:command-output",
		RedisReactionList:  "slack_reactions",
		LogLevel:           INFO,
		DockerOverride:     DefaultDockerOverride,
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return Config{}, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var yamlConfig VibeDeployYamlConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return Config{}, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	if yamlConfig.Redis.Addr != "" {
		config.RedisAddr = yamlConfig.Redis.Addr
	}
	if yamlConfig.Redis.PubSubChannel != "" {
		config.RedisPubSub = yamlConfig.Redis.PubSubChannel
	}
	if yamlConfig.Redis.ListName != "" {
		config.RedisListName = yamlConfig.Redis.ListName
	}
	if yamlConfig.Redis.OutputChannel != "" {
		config.RedisOutputChannel = yamlConfig.Redis.OutputChannel
	}
	if yamlConfig.Redis.ReactionList != "" {
		config.RedisReactionList = yamlConfig.Redis.ReactionList
	}
	if yamlConfig.BaseDir != "" {
		config.BaseDir = yamlConfig.BaseDir
	}
	if yamlConfig.LogLevel != "" {
		config.LogLevel = ParseLogLevel(yamlConfig.LogLevel)
	}
	if yamlConfig.DockerOverride != "" {
		config.DockerOverride = yamlConfig.DockerOverride
	}
	if yamlConfig.AllowedRepos != nil {
		config.AllowedRepos = make(map[string]bool, len(yamlConfig.AllowedRepos))
		for _, repo := range yamlConfig.AllowedRepos {
			config.AllowedRepos[repo] = true
		}
	}
	if yamlConfig.LegacyDockerApps != nil {
		config.LegacyDockerApps = make(map[string]bool, len(yamlConfig.LegacyDockerApps))
		for _, repo := range yamlConfig.LegacyDockerApps {
			config.LegacyDockerApps[repo] = true
		}
	}

	return config, nil
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
