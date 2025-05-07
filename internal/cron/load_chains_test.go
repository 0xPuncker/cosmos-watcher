package cron

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/joho/godotenv"
	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func findRootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func TestLoadChainsJob(t *testing.T) {
	rootDir, err := findRootDir()
	require.NoError(t, err, "Failed to find project root")

	err = godotenv.Load(filepath.Join(rootDir, ".env.test"))
	if err != nil {
		t.Logf("No .env.test file found, using default test values")
		os.Setenv("GITHUB_API_URL", "https://raw.githubusercontent.com")
		os.Setenv("CHAIN_REGISTRY_BASE_URL", "/cosmos/chain-registry/master")
		os.Setenv("LOG_LEVEL", "debug")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	registry := chain.NewChainRegistry(
		logger,
		os.Getenv("GITHUB_API_URL"),
		os.Getenv("CHAIN_REGISTRY_BASE_URL"),
	)

	job := NewLoadChainsJob(registry, logger)
	assert.NotNil(t, job)

	expectedConfig, err := config.LoadChainConfig()
	require.NoError(t, err, "Failed to load chains.yaml")

	expectedChains := make(map[string]bool)
	for _, chain := range expectedConfig.Mainnet {
		expectedChains[chain.Name] = false
	}
	for _, chain := range expectedConfig.Testnet {
		expectedChains[chain.Name] = false
	}

	err = job.Run()
	assert.NoError(t, err)

	loadedChains, err := registry.GetMonitoredChains()
	assert.NoError(t, err)
	assert.NotEmpty(t, loadedChains, "Expected monitored chains to be loaded")

	sort.Strings(loadedChains)

	t.Log("########################################################")
	t.Log("Mainnet chains:")
	for _, chainName := range loadedChains {
		for _, mainnetChain := range expectedConfig.Mainnet {
			if chainName == mainnetChain.Name {
				t.Logf("  - %s (%s)", chainName, mainnetChain.DisplayName)
				break
			}
		}
	}

	t.Log("Testnet chains:")
	for _, chainName := range loadedChains {
		for _, testnetChain := range expectedConfig.Testnet {
			if chainName == testnetChain.Name {
				t.Logf("  - %s (%s)", chainName, testnetChain.DisplayName)
				break
			}
		}
	}

	for _, chainName := range loadedChains {
		_, exists := expectedChains[chainName]
		assert.True(t, exists, "Unexpected chain loaded: %s", chainName)
		expectedChains[chainName] = true
	}

	missingChains := []string{}
	for chainName, wasLoaded := range expectedChains {
		if !wasLoaded {
			missingChains = append(missingChains, chainName)
		}
	}
	assert.Empty(t, missingChains, "Some chains from chains.yaml were not loaded: %v", missingChains)

	t.Logf("Successfully loaded and verified %d chains", len(loadedChains))

	os.Unsetenv("GITHUB_API_URL")
	os.Unsetenv("CHAIN_REGISTRY_BASE_URL")
	os.Unsetenv("LOG_LEVEL")
}
