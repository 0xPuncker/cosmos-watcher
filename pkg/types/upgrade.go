package types

import "time"

type StartupUpgradeInfo struct {
	Name         string    `json:"name"`
	ChainName    string    `json:"chain_name"`
	Height       int64     `json:"height"`
	Info         string    `json:"info"`
	Time         time.Time `json:"time"`
	Estimated    bool      `json:"estimated"`
	Network      string    `json:"network"`
	ProposalLink string    `json:"proposal_link"`
}
