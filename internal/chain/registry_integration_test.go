package chain

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestChainRegistry_ChainExists(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	githubAPIURL := os.Getenv("GITHUB_API_URL")
	if githubAPIURL == "" {
		githubAPIURL = "https://raw.githubusercontent.com"
	}

	chainRegistryURL := os.Getenv("CHAIN_REGISTRY_BASE_URL")
	if chainRegistryURL == "" {
		chainRegistryURL = "/cosmos/chain-registry/master"
	}

	registry := NewChainRegistry(logger, githubAPIURL, chainRegistryURL)

	testCases := []struct {
		name          string
		chainName     string
		shouldExist   bool
		expectedError bool
	}{
		{"Osmosis exists", "osmosis", true, false},
		{"Cosmos Hub exists", "cosmoshub", true, false},
		{"Celestia exists", "celestia", true, false},

		{"Astria mainnet doesn't exist", "astriamainnet", false, true},
		{"Berachain doesn't exist", "berachain", false, true},
		{"Lombard doesn't exist", "lombard", false, true},

		{"Empty chain name", "", false, true},
		{"Invalid characters", "invalid/chain", false, true},
		{"Very long chain name", "verylongchainnamethatshouldnotexist", false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exists := registry.ChainExists(tc.chainName)
			assert.Equal(t, tc.shouldExist, exists, "ChainExists returned unexpected result")


			info, err := registry.GetChainInfo(tc.chainName, false)
			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, info)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, info)
				assert.Equal(t, tc.chainName, info.Name)
			}
		})
	}

	chainsToValidate := []string{"osmosis", "astriamainnet", "cosmoshub", "berachain"}
	existingChains := registry.FilterExistingChains(chainsToValidate)

	assert.Equal(t, 2, len(existingChains))
	assert.Contains(t, existingChains, "osmosis")
	assert.Contains(t, existingChains, "cosmoshub")
	assert.NotContains(t, existingChains, "astriamainnet")
	assert.NotContains(t, existingChains, "berachain")
}

func TestChainRegistry_NonExistentChains(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	githubAPIURL := os.Getenv("GITHUB_API_URL")
	if githubAPIURL == "" {
		githubAPIURL = "https://raw.githubusercontent.com"
	}

	chainRegistryURL := os.Getenv("CHAIN_REGISTRY_BASE_URL")
	if chainRegistryURL == "" {
		chainRegistryURL = "/cosmos/chain-registry/master"
	}

	registry := NewChainRegistry(logger, githubAPIURL, chainRegistryURL)

	nonExistentChains := []struct {
		name    string
		network string
	}{
		{"astriamainnet", "mainnet"},
		{"berachain", "mainnet"},
		{"lombard", "mainnet"},
		{"namada", "mainnet"},
		{"story", "mainnet"},
		{"astriatestnet", "testnet"},
		{"axonetestnet", "testnet"},
		{"berachaintestnet", "testnet"},
		{"dimensiontestnet", "testnet"},
		{"haqqtestnet", "testnet"},
		{"lombardtestnet", "testnet"},
		{"mezotestnet", "testnet"},
		{"namadatestnet", "testnet"},
		{"storytestnet", "testnet"},
	}

	for _, tc := range nonExistentChains {
		t.Run(tc.name, func(t *testing.T) {
			info, err := registry.GetChainInfo(tc.name, false)
			assert.Error(t, err)
			assert.Nil(t, info)
			assert.Contains(t, err.Error(), "404 Not Found")

			upgradeInfo, err := registry.GetUpgradeInfo(tc.name, false)
			assert.Error(t, err)
			assert.Nil(t, upgradeInfo)
		})
	}

	existingChains := []struct {
		name    string
		network string
	}{
		{"osmosis", "mainnet"},
		{"cosmoshub", "mainnet"},
		{"celestia", "mainnet"},
	}

	for _, tc := range existingChains {
		t.Run(tc.name, func(t *testing.T) {
			info, err := registry.GetChainInfo(tc.name, true)
			if err != nil {
				t.Logf("Warning: Failed to fetch chain info for %s: %v", tc.name, err)
				t.SkipNow()
				return
			}

			assert.NotNil(t, info)
			assert.Equal(t, tc.name, info.Name)
			assert.Equal(t, tc.network, info.Network)

			assert.NotEmpty(t, info.ChainID, "ChainID should not be empty")
			assert.NotNil(t, info.LastUpdated, "LastUpdated should not be nil")
		})
	}
}

func TestChainRegistry_ErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	registry := NewChainRegistry(logger, "https://invalid.example.com", "/invalid/path")

	info, err := registry.GetChainInfo("nonexistentchain", false)
	assert.Error(t, err)
	assert.Nil(t, info)

	upgradeInfo, err := registry.GetUpgradeInfo("nonexistentchain", false)
	assert.Error(t, err)
	assert.Nil(t, upgradeInfo)

	chains, err := registry.GetAllChains()
	assert.Error(t, err)
	assert.Nil(t, chains)
}
