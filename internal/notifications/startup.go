package notifications

import (
	"fmt"
	"sync"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/sirupsen/logrus"
)

type StartupNotifier struct {
	registry     types.ChainRegistry
	slack        *SlackService
	logger       *logrus.Logger
	initialDelay time.Duration
}

func NewStartupNotifier(registry types.ChainRegistry, slack *SlackService, logger *logrus.Logger) *StartupNotifier {
	return &StartupNotifier{
		registry:     registry,
		slack:        slack,
		logger:       logger,
		initialDelay: 5 * time.Second,
	}
}

func (n *StartupNotifier) NotifyStartup() error {

	time.Sleep(n.initialDelay)

	chains, err := n.registry.GetMonitoredChains()
	if err != nil {
		return fmt.Errorf("failed to get monitored chains: %w", err)
	}

	var (
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, 5)
	)

	for _, chainName := range chains {
		wg.Add(1)
		go func(chain string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if n.registry.IsUpgradeCached(chain) {
				n.logger.Debugf("Skipping initial check for %s - already cached", chain)
				return
			}

			info, err := n.registry.GetUpgradeInfo(chain, false)
			if err != nil {
				n.logger.Errorf("Failed to get upgrade info for %s: %v", chain, err)
				return
			}
			if info != nil {
				n.logger.Infof("Found upgrade info for %s: version=%s height=%d time=%s",
					chain, info.Version, info.Height, info.Time)
			}
		}(chainName)
	}

	wg.Wait()
	return nil
}
