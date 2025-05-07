package chain

import (
	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/sirupsen/logrus"
)

func GetTestUpgradeInfo() *types.UpgradeInfo {
	logger := logrus.New()
	logger.SetOutput(nil)

	registry := NewChainRegistry(logger, "https://api.github.com", "/cosmos/chain-registry/master")
	info, _ := registry.GetUpgradeInfo("neutron", false)
	return info
}
