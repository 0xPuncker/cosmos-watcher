package poller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xPuncker/cosmos-watcher/internal/chain"
	"github.com/sirupsen/logrus"
)

type RegistryPoller struct {
	registry *chain.ChainRegistry
	logger   *logrus.Logger
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewRegistryPoller(registry *chain.ChainRegistry, logger *logrus.Logger, interval time.Duration) *RegistryPoller {
	return &RegistryPoller{
		registry: registry,
		logger:   logger,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (p *RegistryPoller) Start(ctx context.Context) {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.update()

	for {
		select {
		case <-ticker.C:
			p.update()
		case <-ctx.Done():
			return
		case <-p.stop:
			return
		}
	}
}

func (p *RegistryPoller) Stop() {
	close(p.stop)
	p.wg.Wait()
}

func (p *RegistryPoller) update() {
	chains, err := p.registry.GetMonitoredChains()
	if err != nil {
		p.logger.Errorf("Failed to get monitored chains: %v", err)
		return
	}

	for _, chainName := range chains {
		if err := p.updateChain(chainName); err != nil {
			p.logger.Errorf("Failed to update chain %s: %v", chainName, err)
		}
	}
}

func (p *RegistryPoller) updateChain(chainName string) error {
	chainInfo, err := p.registry.GetChainInfo(chainName, false)
	if err != nil {
		return fmt.Errorf("failed to get chain info: %w", err)
	}

	upgradeInfo, err := p.registry.GetUpgradeInfo(chainName, true)
	if err != nil {
		return fmt.Errorf("failed to get upgrade info: %w", err)
	}

	p.logger.Infof("Updated chain %s: version %s, height %d", chainName, chainInfo.Version, chainInfo.Height)
	p.logger.Infof("Updated upgrade info for %s: %s at height %d", chainName, upgradeInfo.Version, upgradeInfo.Height)

	return nil
}
