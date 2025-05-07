package cron

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/0xPuncker/cosmos-watcher/internal/chain"
	"github.com/0xPuncker/cosmos-watcher/internal/notifications"
	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type UpgradeChecker struct {
	registry   *chain.ChainRegistry
	logger     *logrus.Logger
	slack      *notifications.SlackService
	cron       *cron.Cron
	lastChecks map[string]time.Time
	mu         sync.RWMutex
}

func NewUpgradeChecker(registry *chain.ChainRegistry, logger *logrus.Logger, slack *notifications.SlackService) *UpgradeChecker {
	return &UpgradeChecker{
		registry:   registry,
		logger:     logger,
		slack:      slack,
		cron:       cron.New(),
		lastChecks: make(map[string]time.Time),
	}
}

func (uc *UpgradeChecker) Start() error {
	_, err := uc.cron.AddFunc("@hourly", uc.checkUpgrades)
	if err != nil {
		return fmt.Errorf("failed to schedule cron job: %w", err)
	}

	uc.cron.Start()
	uc.logger.Info("Upgrade checker cron job started")

	go uc.checkUpgrades()

	return nil
}

func (uc *UpgradeChecker) Stop() {
	ctx := uc.cron.Stop()
	<-ctx.Done()
	uc.logger.Info("Upgrade checker cron job stopped")
}

func (uc *UpgradeChecker) CheckUpgrades() {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	chains, err := uc.registry.GetMonitoredChains()
	if err != nil {
		uc.logger.WithError(err).Error("Failed to get monitored chains")
		return
	}

	uc.logger.WithField("chain_count", len(chains)).Info("Found monitored chains")

	for _, chain := range chains {
		uc.logger.WithField("chain", chain).Debug("Processing chain")

		if !uc.registry.ChainExists(chain) {
			uc.logger.WithField("chain", chain).Debug("Chain not found in registry, skipping")
			continue
		}

		info, err := uc.registry.GetChainInfo(chain, false)
		if err != nil {
			if strings.Contains(err.Error(), "invalid URL scheme") {
				uc.logger.WithFields(logrus.Fields{
					"chain": chain,
					"error": err,
				}).Debug("Invalid chain URL format, skipping")
				continue
			}
			if strings.Contains(err.Error(), "404 Not Found") {
				uc.logger.WithFields(logrus.Fields{
					"chain": chain,
					"error": err,
				}).Debug("Chain info not found, skipping")
				continue
			}
			uc.logger.WithFields(logrus.Fields{
				"chain": chain,
				"error": err,
			}).Error("Failed to get chain info")
			continue
		}

		uc.logger.WithFields(logrus.Fields{
			"chain":   chain,
			"network": info.Network,
			"version": info.Version,
			"height":  info.Height,
		}).Debug("Retrieved chain info")

		if info.Network != "mainnet" && info.Network != "testnet" {
			uc.logger.WithFields(logrus.Fields{
				"chain":   chain,
				"network": info.Network,
			}).Debug("Skipping non-mainnet/testnet chain")
			continue
		}

		upgradeInfo, err := uc.registry.GetUpgradeInfo(chain, false)
		if err != nil {
			if strings.Contains(err.Error(), "received HTML response") {
				uc.logger.WithFields(logrus.Fields{
					"chain": chain,
					"error": err,
				}).Debug("Invalid response format, skipping")
				continue
			}
			uc.logger.WithFields(logrus.Fields{
				"chain": chain,
				"error": err,
			}).Warn("Failed to get upgrade info")
			continue
		}

		if upgradeInfo == nil {
			uc.logger.WithField("chain", chain).Debug("No upgrade info found")
			continue
		}

		uc.logger.WithFields(logrus.Fields{
			"chain":   chain,
			"name":    upgradeInfo.Name,
			"version": upgradeInfo.Version,
			"height":  upgradeInfo.Height,
			"network": upgradeInfo.Network,
			"time":    upgradeInfo.Time.Format(time.RFC3339),
		}).Debug("Retrieved upgrade info")

		lastCheck, exists := uc.lastChecks[chain]
		if !exists {
			uc.logger.WithField("chain", chain).Debug("First check for chain")
		} else {
			uc.logger.WithFields(logrus.Fields{
				"chain":      chain,
				"last_check": lastCheck.Format(time.RFC3339),
			}).Debug("Previous check found")
		}

		if !exists || lastCheck != upgradeInfo.Time {
			typesUpgradeInfo := &types.UpgradeInfo{
				Name:             upgradeInfo.Name,
				ChainName:        chain,
				Height:           upgradeInfo.Height,
				Info:             upgradeInfo.Info,
				Time:             upgradeInfo.Time,
				Version:          upgradeInfo.Version,
				Estimated:        true,
				Network:          upgradeInfo.Network,
				ProposalLink:     upgradeInfo.Proposal,
				Guide:            upgradeInfo.Guide,
				BlockLink:        fmt.Sprintf("https://www.mintscan.io/%s/blocks/%d", chain, upgradeInfo.Height),
				CosmovisorFolder: fmt.Sprintf("upgrades/%s", upgradeInfo.Version),
				GitHash:          upgradeInfo.GitHash,
				Repo:             upgradeInfo.Repo,
				RPC:              upgradeInfo.RPC,
				API:              upgradeInfo.API,
			}

			uc.logger.WithFields(logrus.Fields{
				"chain":   chain,
				"name":    typesUpgradeInfo.Name,
				"height":  typesUpgradeInfo.Height,
				"network": typesUpgradeInfo.Network,
				"time":    typesUpgradeInfo.Time.Format(time.RFC3339),
			}).Info("New upgrade found")

			if uc.slack != nil {
				if err := uc.slack.SendUpgradeNotification(chain, typesUpgradeInfo); err != nil {
					uc.logger.WithFields(logrus.Fields{
						"chain": chain,
						"error": err,
					}).Error("Failed to send Slack notification")
				} else {
					uc.logger.WithField("chain", chain).Info("Slack notification sent successfully")
				}
			} else {
				uc.logger.WithField("chain", chain).Debug("Slack service not configured, skipping notification")
			}

			uc.lastChecks[chain] = upgradeInfo.Time
			uc.logger.WithFields(logrus.Fields{
				"chain": chain,
				"time":  upgradeInfo.Time.Format(time.RFC3339),
			}).Debug("Updated last check time")
		} else {
			uc.logger.WithFields(logrus.Fields{
				"chain": chain,
				"time":  upgradeInfo.Time.Format(time.RFC3339),
			}).Debug("No new upgrades found")
		}
	}

	uc.logger.Info("Completed checking all chains")
}

func (uc *UpgradeChecker) checkUpgrades() {
	uc.logger.Info("Starting upgrade check cycle")
	uc.CheckUpgrades()
	uc.logger.Info("Completed upgrade check cycle")
}
