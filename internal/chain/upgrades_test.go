package chain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Failed to get working directory: %v", err))
	}

	rootDir := filepath.Dir(filepath.Dir(wd))
	envFile := filepath.Join(rootDir, ".env.test")
	if err := godotenv.Load(envFile); err != nil {
		os.Setenv("GITHUB_API_URL", "https://raw.githubusercontent.com")
		os.Setenv("CHAIN_REGISTRY_BASE_URL", "/cosmos/chain-registry/master")
		fmt.Printf("Notice: Using default environment variables for testing\n")
	} else {
		fmt.Printf("Notice: Loaded environment variables from %s\n", envFile)
	}
}

func TestPolkachuAPI(t *testing.T) {
	resp, err := http.Get("https://polkachu.com/api/v2/chain_upgrades")
	if err != nil {
		t.Fatalf("Failed to call Polkachu API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Polkachu API returned status %d", resp.StatusCode)
	}

	var upgrades []PolkachuUpgrade
	if err := json.NewDecoder(resp.Body).Decode(&upgrades); err != nil {
		t.Fatalf("Failed to decode Polkachu response: %v", err)
	}

	if len(upgrades) == 0 {
		t.Log("Notice: No upgrades found in Polkachu API")
		return
	}

	t.Logf("Found %d upgrades in Polkachu API", len(upgrades))
	for _, upgrade := range upgrades {
		t.Logf("Network: %s, Version: %s, Block: %d, Time: %s",
			upgrade.Network,
			upgrade.NodeVersion,
			upgrade.Block,
			upgrade.EstimatedUpgradeTime,
		)
	}
}

func TestFetchUpgrades(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	githubAPIURL := os.Getenv("GITHUB_API_URL")
	if githubAPIURL == "" {
		githubAPIURL = "https://raw.githubusercontent.com"
		t.Logf("Notice: Using default GITHUB_API_URL: %s", githubAPIURL)
	}

	chainRegistryURL := os.Getenv("CHAIN_REGISTRY_BASE_URL")
	if chainRegistryURL == "" {
		chainRegistryURL = "/cosmos/chain-registry/master"
		t.Logf("Notice: Using default CHAIN_REGISTRY_BASE_URL: %s", chainRegistryURL)
	}

	registry := NewChainRegistry(
		logger,
		githubAPIURL,
		chainRegistryURL,
	)

	testCases := []struct {
		name      string
		chainName string
	}{
		{"Neutron", "neutron"},
		{"Osmosis", "osmosis"},
		{"Cosmos Hub", "cosmoshub"},
		{"Stride", "stride"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry.SetChainName(tc.chainName)

			t.Log("Testing Polkachu API...")
			polkachuUpgrade, err := registry.fetchPolkachuUpgrades(tc.chainName)
			if err != nil {
				t.Logf("Notice: Polkachu API error: %v", err)
			}
			if polkachuUpgrade != nil {
				estimatedTime, _ := time.Parse(time.RFC3339, polkachuUpgrade.EstimatedUpgradeTime)
				t.Logf("Polkachu upgrade found for %s: Version=%s, Height=%d, Time=%s, Info=%s",
					tc.chainName,
					polkachuUpgrade.NodeVersion,
					polkachuUpgrade.Block,
					estimatedTime.Format(time.RFC3339),
					polkachuUpgrade.Guide,
				)
			} else {
				t.Logf("Notice: No upgrades found from %s in Polkachu", tc.chainName)
			}

			upgradeInfo, err := registry.GetUpgradeInfo(tc.chainName, false)
			if err != nil {
				t.Logf("Notice: Combined upgrade info error: %v", err)
			}
			if upgradeInfo != nil {
				if upgradeInfo.Height == 1000000 {
					t.Logf("Notice: Using placeholder upgrade info for %s", tc.chainName)
				} else {
					t.Logf("Final upgrade info for %s: Version=%s, Height=%d, Time=%s, Info=%s",
						tc.chainName,
						upgradeInfo.Name,
						upgradeInfo.Height,
						upgradeInfo.Time.Format(time.RFC3339),
						upgradeInfo.Info,
					)
				}
			}
		})
	}
}
