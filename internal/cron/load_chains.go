package cron

import (
	"strings"
	"sync"

	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/internal/config"
	"github.com/sirupsen/logrus"
)

type LoadChainsJob struct {
	registry *chain.ChainRegistry
	logger   *logrus.Logger
}

func NewLoadChainsJob(registry *chain.ChainRegistry, logger *logrus.Logger) *LoadChainsJob {
	return &LoadChainsJob{
		registry: registry,
		logger:   logger,
	}
}

func (j *LoadChainsJob) Run() error {
	chainConfig, err := config.LoadChainConfig()
	if err != nil {
		j.logger.Errorf("Failed to load chain config: %v", err)
		return err
	}

	var chainNames []string

	j.logger.Infof("Loading chains from config file...")

	if len(chainConfig.Mainnet) > 0 {
		j.logger.Info("=== Mainnet Chains ===")
		var names []string
		for _, chain := range chainConfig.Mainnet {
			chainNames = append(chainNames, chain.Name)
			names = append(names, chain.Name)
		}
		j.logger.Info("  " + strings.Join(names, ", "))
	}

	if len(chainConfig.Testnet) > 0 {
		j.logger.Info("=== Testnet Chains ===")
		var names []string
		for _, chain := range chainConfig.Testnet {
			chainNames = append(chainNames, chain.Name)
			names = append(names, chain.Name)
		}
		j.logger.Info("  " + strings.Join(names, ", "))
	}

	j.logger.Infof("Total chains to monitor: %d (%d mainnet, %d testnet)",
		len(chainNames),
		len(chainConfig.Mainnet),
		len(chainConfig.Testnet))

	type chainError struct {
		name string
		err  error
	}

	var (
		wg           sync.WaitGroup
		mu           sync.Mutex
		semaphore    = make(chan struct{}, 5)
		failedChains []chainError
	)

	for _, chainName := range chainNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			j.logger.Debugf("Pre-fetching chain info for %s", name)
			_, err := j.registry.GetChainInfo(name, true)
			if err != nil {
				mu.Lock()
				failedChains = append(failedChains, chainError{name: name, err: err})
				mu.Unlock()
				return
			}

			_, err = j.registry.GetUpgradeInfo(name, true)
			if err != nil {
				j.logger.Debugf("No upgrade info available for %s: %v", name, err)
			}
		}(chainName)
	}

	wg.Wait()

	j.registry.SetMonitoredChains(chainNames)

	if len(failedChains) > 0 {
		j.logger.Infof("=== Chains failed to load (%d) ===", len(failedChains))
		for _, fc := range failedChains {
			j.logger.Infof("  %s: %v", fc.name, fc.err)
		}
	}

	j.logger.Infof("Finished loading chains! Successfully loaded %d chains.",
		len(chainNames)-len(failedChains))

	if len(failedChains) > 0 {
		j.logger.Infof("Failed to load %d chains (Check if the chain's name is correct and supported by the chain-registry)", len(failedChains))
	}
	return nil
}
