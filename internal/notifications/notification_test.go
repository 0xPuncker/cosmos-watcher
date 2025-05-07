package notifications

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func TestNotificationServiceManual(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	rootDir := filepath.Dir(filepath.Dir(wd))
	err = godotenv.Load(filepath.Join(rootDir, ".env.test"))
	if err != nil {
		t.Log("No .env.test file found, using environment variables")
	}

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("SLACK_WEBHOOK_URL not set")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	slackService, err := NewSlackService(logger)
	if err != nil {
		t.Fatal(err)
	}
	notificationService := NewNotificationService(slackService)

	upgradeTime, _ := time.Parse("2006-01-02 15:04:05.000000 -0700",
		"2025-04-25 18:15:05.697245 +0200")

	upgradeInfo := &types.UpgradeInfo{
		Name:         "v1.0.0",
		ChainName:    "osmosis",
		Height:       1000000,
		Info:         "This is a test upgrade with all possible links",
		Time:         upgradeTime,
		Version:      "v1.0.0",
		Estimated:    true,
		Network:      "mainnet",
		ProposalLink: "https://www.mintscan.io/osmosis/proposals/1",
		Guide:        "https://docs.osmosis.zone/upgrades/v1.0.0",
		BlockLink:    "https://www.mintscan.io/osmosis/blocks/1000000",
	}

	err = notificationService.SendUpgradeNotification("osmosis", upgradeInfo)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Successfully sent test notification with all links")
	t.Logf("Chain: %s", upgradeInfo.ChainName)
	t.Logf("Name: %s", upgradeInfo.Name)
	t.Logf("Height: %d", upgradeInfo.Height)
	t.Logf("Time: %s", upgradeInfo.Time.Format("2006-01-02 15:04:05.000000 -0700"))
	t.Logf("Network: %s", upgradeInfo.Network)
	t.Log("Links:")
	t.Log("📄 View Proposal | 📚 View Guide | 🔍 View Block")
}
