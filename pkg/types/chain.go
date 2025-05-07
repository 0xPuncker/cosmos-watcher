package types

import (
	"fmt"
	"time"
)

type ChainRegistry interface {
	GetUpgradeInfo(chainName string, includeTestnet bool) (*UpgradeInfo, error)
	GetAllChains() ([]string, error)
	GetMonitoredChains() ([]string, error)
	IsUpgradeCached(chainName string) bool
}

type ChainInfo struct {
	Name    string
	Network string
}

type UpgradeInfo struct {
	Name             string    `json:"name"`
	ChainName        string    `json:"chain_name"`
	Height           int64     `json:"height"`
	Info             string    `json:"info"`
	Time             time.Time `json:"time"`
	Version          string    `json:"version"`
	Estimated        bool      `json:"estimated"`
	Network          string    `json:"network"`
	Proposal         string    `json:"proposal"`
	ProposalLink     string    `json:"proposal_link"`
	Guide            string    `json:"guide"`
	BlockLink        string    `json:"block_link"`
	CosmovisorFolder string    `json:"cosmovisor_folder"`
	GitHash          string    `json:"git_hash"`
	Repo             string    `json:"repo"`
	RPC              string    `json:"rpc"`
	API              string    `json:"api"`
}

func (u *UpgradeInfo) GetChainName() string {
	return u.ChainName
}

func (u *UpgradeInfo) GetVersion() string {
	return u.Version
}

func (u *UpgradeInfo) GetHeight() int64 {
	return u.Height
}

func (u *UpgradeInfo) GetEstimatedAt() string {
	return u.Time.Format(time.RFC3339)
}

func (u *UpgradeInfo) GetProposalLink() string {
	return u.ProposalLink
}

func (u *UpgradeInfo) GetNetwork() string {
	return u.Network
}

func (u *UpgradeInfo) GetNodeVersion() string {
	return u.Version
}

func (u *UpgradeInfo) GetBlock() string {
	return fmt.Sprintf("%d", u.Height)
}

func (u *UpgradeInfo) GetEstimatedUpgradeTime() string {
	return u.Time.Format(time.RFC3339)
}

func (u *UpgradeInfo) GetGuide() string {
	return u.Guide
}

func (u *UpgradeInfo) GetBlockLink() string {
	return u.BlockLink
}

func (u *UpgradeInfo) GetCosmovisorFolder() string {
	return u.CosmovisorFolder
}

func (u *UpgradeInfo) GetGitHash() string {
	return u.GitHash
}

func (u *UpgradeInfo) GetRepo() string {
	return u.Repo
}

func (u *UpgradeInfo) GetRPC() string {
	return u.RPC
}

func (u *UpgradeInfo) GetAPI() string {
	return u.API
}
