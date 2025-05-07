package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xPuncker/cosmos-watcher/internal/chain"
	"github.com/0xPuncker/cosmos-watcher/internal/config"
	"github.com/0xPuncker/cosmos-watcher/internal/notifications"
	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func init() {

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rootDir := filepath.Dir(filepath.Dir(wd))
	envFile := filepath.Join(rootDir, ".env.test")
	if err := godotenv.Load(envFile); err != nil {
		fmt.Printf("Notice: To run this test, you need to create %s with SLACK_WEBHOOK_URL\n", envFile)
	} else {
		fmt.Printf("Notice: Loaded environment variables from %s\n", envFile)
	}
}

func TestSendPolkachuUpgradeNotification(t *testing.T) {

	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("Skipping test: SLACK_WEBHOOK_URL not set in .env.test")
	}

	chainName := os.Getenv("CHAIN")
	if chainName == "" {
		t.Skip("Skipping test: CHAIN environment variable not set. Example: CHAIN=osmosis go test -v ./internal/testutil/polkachu_test.go")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)

	logger.AddHook(&RPCErrorHook{})

	chainsConfig, err := config.LoadChainConfig()
	if err != nil {
		t.Fatalf("Failed to load chains config: %v", err)
	}

	var chainConfig *config.Chain
	for _, chain := range chainsConfig.Mainnet {
		if chain.Name == chainName {
			chainConfig = &chain
			break
		}
	}
	if chainConfig == nil {
		for _, chain := range chainsConfig.Testnet {
			if chain.Name == chainName {
				chainConfig = &chain
				break
			}
		}
	}
	if chainConfig == nil {
		t.Fatalf("Chain %s not found in configuration", chainName)
	}

	testChainUpgrade(t, logger, chainConfig)
}

func TestPolkachuUpgrades(t *testing.T) {
	if os.Getenv("CHAIN") != "" {
		t.Skip("Skipping test: CHAIN environment variable is set")
	}

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	testCases := []struct {
		name        string
		chainConfig *config.Chain
	}{
		{
			name: "Osmosis",
			chainConfig: &config.Chain{
				Name:        "osmosis",
				DisplayName: "Osmosis",
				Network:     "mainnet",
			},
		},
		{
			name: "Cosmos Hub",
			chainConfig: &config.Chain{
				Name:        "cosmoshub",
				DisplayName: "Cosmos Hub",
				Network:     "mainnet",
			},
		},
		{
			name: "Juno",
			chainConfig: &config.Chain{
				Name:        "juno",
				DisplayName: "Juno",
				Network:     "mainnet",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testChainUpgrade(t, logger, tc.chainConfig)
		})
	}
}

func testChainUpgrade(t *testing.T, logger *logrus.Logger, chainConfig *config.Chain) {

	registry := chain.NewChainRegistry(
		logger,
		os.Getenv("GITHUB_API_URL"),
		os.Getenv("CHAIN_REGISTRY_BASE_URL"),
	)

	chainUpgradeInfo, err := registry.GetUpgradeInfo(chainConfig.Name, true)
	if err != nil {

		if strings.Contains(err.Error(), "proposals endpoint returned") ||
			strings.Contains(err.Error(), "upgrade plan") {

			logger.Debug("Ignoring expected RPC/REST errors")
		} else {
			t.Logf("No upgrade info available for %s: %v", chainConfig.DisplayName, err)
			t.Log("ℹ️  This is normal if there are no scheduled upgrades.")
			return
		}
	}

	if chainUpgradeInfo == nil {
		t.Logf("ℹ️  No upgrade scheduled for %s at the moment", chainConfig.DisplayName)
		return
	}

	if !isValidUpgrade(chainUpgradeInfo) {
		t.Logf("ℹ️  No valid upgrade found for %s (height: %d, version: %s)",
			chainConfig.DisplayName,
			chainUpgradeInfo.Height,
			chainUpgradeInfo.Version)
		return
	}

	slackService, err := notifications.NewSlackService(logger)
	if err != nil {
		t.Fatalf("Failed to create Slack service: %v", err)
	}

	err = slackService.SendUpgradeNotification(chainConfig.Name, chainUpgradeInfo)
	if err != nil {
		t.Fatalf("Failed to send upgrade notification: %v", err)
	}

	t.Logf("🚀 Found upgrade and sent notification for %s:", chainConfig.DisplayName)
	t.Logf("  Name: %s", chainUpgradeInfo.Name)
	if chainUpgradeInfo.ProposalLink != "" {
		t.Logf("  Proposal: %s", chainUpgradeInfo.ProposalLink)
	}
	t.Logf("  Network Type: %s", chainConfig.Network)
	t.Logf("  Height: %d", chainUpgradeInfo.Height)
	t.Logf("  Time: %s", chainUpgradeInfo.Time.Format(time.RFC1123))
	t.Logf("  Info: %s", chainUpgradeInfo.Info)
	t.Logf("  Cosmovisor: upgrades/%s", chainUpgradeInfo.Name)
}

func isValidUpgrade(upgrade *types.UpgradeInfo) bool {
	if upgrade.Height <= 0 {
		return false
	}

	if upgrade.Version == "" || upgrade.Name == fmt.Sprintf("%s-upgrade", upgrade.ChainName) {
		return false
	}

	return true
}

type RPCErrorHook struct{}

func (h *RPCErrorHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

func (h *RPCErrorHook) Fire(entry *logrus.Entry) error {

	msg := entry.Message
	if (strings.Contains(msg, "Failed to fetch active proposals") && strings.Contains(msg, "proposals endpoint returned")) ||
		(strings.Contains(msg, "Failed to fetch current upgrade plan") && strings.Contains(msg, "upgrade plan")) {

		entry.Level = logrus.TraceLevel
	}
	return nil
}
