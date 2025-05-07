package poller

import (
	"sync"
	"time"

	"github.com/0xPuncker/cosmos-watcher/internal/chain"
	"github.com/sirupsen/logrus"
)

type Poller struct {
	registry *chain.ChainRegistry
	logger   *logrus.Logger
	interval time.Duration
	stop     chan struct{}
	wg       sync.WaitGroup
}

func New(registry *chain.ChainRegistry, logger *logrus.Logger, interval time.Duration) *Poller {
	return &Poller{
		registry: registry,
		logger:   logger,
		interval: interval,
		stop:     make(chan struct{}),
	}
}

func (p *Poller) Start() {
	p.wg.Add(1)
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.update()
		case <-p.stop:
			return
		}
	}
}

func (p *Poller) Stop() {
	close(p.stop)
	p.wg.Wait()
}

func (p *Poller) update() {
	p.logger.Debug("Starting poller update cycle")
	chains, err := p.registry.GetMonitoredChains()
	if err != nil {
		p.logger.Errorf("Failed to get monitored chains: %v", err)
		return
	}

	p.logger.Debugf("Checking updates for %d chains", len(chains))
	for _, chainName := range chains {
		p.logger.Debugf("Checking chain: %s", chainName)
		if err := p.updateChain(chainName); err != nil {
			p.logger.Errorf("Failed to update chain %s: %v", chainName, err)
		}
	}
	p.logger.Debug("Completed poller update cycle")
}

func (p *Poller) updateChain(chainName string) error {
	chainInfo, err := p.registry.GetChainInfo(chainName, false)
	if err != nil {
		return err
	}
	p.logger.Debugf("Chain %s info updated: %+v", chainName, chainInfo)

	upgradeInfo, err := p.registry.GetUpgradeInfo(chainName, true)
	if err != nil {
		return err
	}
	if upgradeInfo != nil {
		p.logger.Infof("Chain %s upgrade info: Version=%s, Height=%d, Time=%s",
			chainName,
			upgradeInfo.Version,
			upgradeInfo.Height,
			upgradeInfo.Time.Format(time.RFC3339),
		)
	}
	return nil
}
