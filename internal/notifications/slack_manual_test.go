package notifications

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/config"
	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type PolkachuUpgrade struct {
	ChainName            string `json:"chain_name"`
	NodeVersion          string `json:"node_version"`
	Block                int64  `json:"block"`
	EstimatedUpgradeTime string `json:"estimated_upgrade_time"`
	Guide                string `json:"guide"`
	ProposalLink         string `json:"proposal"`
	Status               string `json:"status"`
	Description          string `json:"description"`
}

func TestSlackNotificationManual(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootDir := filepath.Dir(filepath.Dir(wd))
	t.Logf("Project root directory: %s", rootDir)

	err = godotenv.Load(filepath.Join(rootDir, ".env.test"))
	if err != nil {
		t.Log("No .env.test file found, using environment variables")
	}

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("SLACK_WEBHOOK_URL not set")
	}

	chainName := os.Getenv("CHAIN_NAME")
	if chainName == "" {
		t.Skip("CHAIN not set")
	}

	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to project root directory: %v", err)
	}

	chainConfig, err := config.LoadConfig("")
	if err != nil {
		t.Fatalf("Failed to load chain config: %v", err)
	}

	chain := chainConfig.GetChainByName(chainName)
	if chain == nil {
		t.Fatalf("Chain %s not found in configuration", chainName)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	resp, err := http.Get("https://polkachu.com/api/v2/chain_upgrades")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var upgrades []PolkachuUpgrade
	if err := json.NewDecoder(resp.Body).Decode(&upgrades); err != nil {
		t.Fatal(err)
	}

	var testUpgrade *PolkachuUpgrade
	for _, upgrade := range upgrades {
		if upgrade.ChainName == chain.DisplayName {
			testUpgrade = &upgrade
			break
		}
	}

	if testUpgrade == nil {
		t.Skipf("No upgrade found for %s chain", chain.DisplayName)
	}

	estimatedTime, err := time.Parse("2006-01-02T15:04:05.000000Z", testUpgrade.EstimatedUpgradeTime)
	if err != nil {
		t.Fatal(err)
	}

	cosmovisorFolder := fmt.Sprintf("upgrades/%s", testUpgrade.NodeVersion)

	upgradeInfo := &types.UpgradeInfo{
		Name:             testUpgrade.NodeVersion,
		Height:           testUpgrade.Block,
		Info:             testUpgrade.Description,
		Time:             estimatedTime,
		Estimated:        true,
		Network:          chain.Network,
		ProposalLink:     testUpgrade.ProposalLink,
		Guide:            testUpgrade.Guide,
		Version:          testUpgrade.NodeVersion,
		CosmovisorFolder: cosmovisorFolder,
		BlockLink:        fmt.Sprintf("https://www.mintscan.io/%s/blocks/%d", chainName, testUpgrade.Block),
	}

	slackService, err := NewSlackService(logger)
	if err != nil {
		t.Fatal(err)
	}

	err = slackService.SendUpgradeNotification(chain.Name, upgradeInfo)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Successfully sent notification for %s upgrade:", chain.DisplayName)
	t.Logf("Version: %s", upgradeInfo.Name)
	t.Logf("Height: %d", upgradeInfo.Height)
	t.Logf("Time: %s", upgradeInfo.Time.Format(time.RFC3339))
	t.Logf("Network: %s", upgradeInfo.Network)
	t.Logf("Cosmovisor Folder: %s", upgradeInfo.CosmovisorFolder)
	t.Logf("Guide: %s", upgradeInfo.Guide)
	t.Logf("Proposal: %s", upgradeInfo.ProposalLink)
	t.Logf("Block Link: %s", upgradeInfo.BlockLink)
}
