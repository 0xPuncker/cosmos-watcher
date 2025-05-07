package testutil

import (
	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"github.com/sirupsen/logrus"
)

func GetTestUpgradeInfo() *types.UpgradeInfo {
	logger := logrus.New()
	logger.SetOutput(nil)

	registry := chain.NewChainRegistry(logger, "https://raw.githubusercontent.com", "/cosmos/chain-registry/master")
	info, _ := registry.GetUpgradeInfo("neutron", false)
	return info
}
