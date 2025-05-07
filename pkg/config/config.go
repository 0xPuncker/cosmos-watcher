package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"gopkg.in/yaml.v3"
)

type Chain struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Network     string `yaml:"network"`
}

type Config struct {
	Mainnet []Chain `yaml:"mainnet"`
	Testnet []Chain `yaml:"testnet"`
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config/chains.yaml"
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (c *Config) GetChainNames(networkType string) []string {
	var chains []Chain
	switch networkType {
	case "mainnet":
		chains = c.Mainnet
	case "testnet":
		chains = c.Testnet
	default:
		return nil
	}

	names := make([]string, len(chains))
	for i, chain := range chains {
		names[i] = chain.Name
	}
	return names
}

func (c *Config) GetChainByName(name string) *Chain {
	for _, chain := range c.Mainnet {
		if chain.Name == name {
			return &chain
		}
	}

	for _, chain := range c.Testnet {
		if chain.Name == name {
			return &chain
		}
	}

	return nil
}

func (c *Chain) ToChainConfig() *types.ChainConfig {
	return &types.ChainConfig{
		Name:        c.Name,
		DisplayName: c.DisplayName,
		Network:     c.Network,
	}
}

func GetChainInfo(chainName string) (*Chain, error) {
	config, err := LoadConfig("")
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
