package types

// ChainConfig represents a chain configuration
type ChainConfig struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Network     string `yaml:"network"`
}

// ChainsConfig represents the configuration for mainnet and testnet chains
type ChainsConfig struct {
	Mainnet []ChainConfig `yaml:"mainnet"`
	Testnet []ChainConfig `yaml:"testnet"`
}
