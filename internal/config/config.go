package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig    `json:"server"`
	GitHub   GitHubConfig    `json:"github"`
	Registry RegistryConfig  `json:"registry"`
	Poller   PollerConfig    `json:"poller"`
	Slack    SlackConfig     `json:"slack"`
	Jobs     types.JobConfig `json:"jobs"`
}

type ServerConfig struct {
	Port         string `json:"port"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
}

type GitHubConfig struct {
	APIURL  string `json:"api_url"`
	Token   string `json:"token"`
	Timeout string `json:"timeout"`
}

type RegistryConfig struct {
	URL             string `json:"url"`
	RefreshInterval string `json:"refresh_interval"`
}

type PollerConfig struct {
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

type SlackConfig struct {
	WebhookURL            string `json:"webhook_url"`
	NotificationThreshold string `json:"notification_threshold"`
}

type ChainConfig struct {
	Mainnet []Chain `yaml:"mainnet"`
	Testnet []Chain `yaml:"testnet"`
}

type Chain struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Network     string `yaml:"network"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if err := godotenv.Load(); err != nil {
			if err := godotenv.Load(".env.local"); err != nil {
				fmt.Printf("No .env or .env.local file found. Using environment variables.\n")
			}
		}

		for _, env := range []string{
			"PORT",
			"GITHUB_API_URL",
			"CHAIN_REGISTRY_BASE_URL",
			"POLLER_INTERVAL",
		} {
			fmt.Printf("%s=%s\n", env, os.Getenv(env))
		}

		return &Config{
			Server: ServerConfig{
				Port: getEnv("PORT", "8080"),
			},
			GitHub: GitHubConfig{
				APIURL: getEnv("GITHUB_API_URL", "https://raw.githubusercontent.com"),
			},
			Registry: RegistryConfig{
				URL: getEnv("CHAIN_REGISTRY_BASE_URL", "/cosmos/chain-registry/master"),
			},
			Poller: PollerConfig{
				Interval: getEnv("POLLER_INTERVAL", "1m"),
			},
		}, nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "8080",
		},
		GitHub: GitHubConfig{
			APIURL: "https://api.github.com",
		},
		Registry: RegistryConfig{
			URL: "https://raw.githubusercontent.com/cosmos/chain-registry/master",
		},
		Poller: PollerConfig{
			Interval: "1m",
		},
	}
}

func LoadChainConfig() (*ChainConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	var configPath string
	for i := 0; i < 3; i++ {
		configPath = filepath.Join(wd, "config", "chains.yaml")
		if _, err := os.Stat(configPath); err == nil {
			break
		}

		if i == 0 {
			configPath = filepath.Join(wd, "chains.yaml")
			if _, err := os.Stat(configPath); err == nil {
				break
			}
		}

		wd = filepath.Dir(wd)
		if wd == "/" {
			return nil, fmt.Errorf("config directory not found")
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ChainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func GetChainInfo(chainName string) (*Chain, error) {
	config, err := LoadChainConfig()
	if err != nil {
		return nil, err
	}

	for _, chain := range config.Mainnet {
		if chain.Name == chainName {
			return &chain, nil
		}
	}

	for _, chain := range config.Testnet {
		if chain.Name == chainName {
			return &chain, nil
		}
	}

	return nil, fmt.Errorf("chain %s not found in configuration", chainName)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
