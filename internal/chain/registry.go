package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/p2p/devops-cosmos-watcher/internal/notifications"
	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type Upgrade interface {
	GetChainName() string
	GetVersion() string
	GetHeight() int64
	GetEstimatedAt() string
	GetProposalLink() string
	GetNetwork() string
	GetNodeVersion() string
	GetBlock() string
	GetEstimatedUpgradeTime() string
	GetGuide() string
	GetBlockLink() string
	GetCosmovisorFolder() string
	GetGitHash() string
	GetRepo() string
	GetRPC() string
	GetAPI() string
}

type ChainRegistry struct {
	cache            *cache.Cache
	logger           *logrus.Logger
	client           *http.Client
	baseURL          string
	slack            *notifications.SlackService
	chainName        string
	chains           map[string]*ChainInfo
	mu               sync.RWMutex
	monitoredChains  []string
	githubAPIURL     string
	chainRegistryURL string
	API              string
}

type ChainInfo struct {
	Name        string     `json:"name"`
	ChainID     string     `json:"chain_id"`
	Network     string     `json:"network"`
	Version     string     `json:"version"`
	Height      int64      `json:"height"`
	APIs        APIs       `json:"apis"`
	Explorers   []Explorer `json:"explorers"`
	LastUpdated time.Time  `json:"last_updated"`
}

type UpgradeInfo struct {
	Name             string    `json:"name"`
	ChainName        string    `json:"chain_name"`
	Network          string    `json:"network"`
	Version          string    `json:"version"`
	Height           int64     `json:"height"`
	Time             time.Time `json:"time"`
	Info             string    `json:"info"`
	LastUpdated      time.Time `json:"last_updated"`
	Proposal         string    `json:"proposal"`
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
	return u.Proposal
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
	return u.Info
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

type APIs struct {
	RPC  []Endpoint `json:"rpc"`
	REST []Endpoint `json:"rest"`
}

type Endpoint struct {
	Address string `json:"address"`
}

type Explorer struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	TxPage string `json:"tx_page"`
	Kind   string `json:"kind"`
}

type PolkachuUpgrade struct {
	Network              string `json:"network"`
	ChainName            string `json:"chain_name"`
	Repo                 string `json:"repo"`
	NodeVersion          string `json:"node_version"`
	CosmovisorFolder     string `json:"cosmovisor_folder"`
	GitHash              string `json:"git_hash"`
	Proposal             string `json:"proposal"`
	Block                int64  `json:"block"`
	BlockLink            string `json:"block_link"`
	EstimatedUpgradeTime string `json:"estimated_upgrade_time"`
	Guide                string `json:"guide"`
	RPC                  string `json:"rpc"`
	API                  string `json:"api"`
}

type PolkachuResponse struct {
	Data []PolkachuUpgrade `json:"data"`
}

const (
	chainInfoCacheKey   = "chain_info:%s"
	upgradeInfoCacheKey = "upgrade_info:%s"
)

func NewChainRegistry(logger *logrus.Logger, githubAPIURL, chainRegistryURL string) *ChainRegistry {
	godotenv.Load()
	baseURL := githubAPIURL
	if strings.HasPrefix(chainRegistryURL, "http") {
		baseURL = chainRegistryURL
	} else if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	chainRegistryURL = strings.TrimRight(chainRegistryURL, "/")

	logger.Debugf("Base URL: %s", baseURL)
	logger.Debugf("Chain Registry URL: %s", chainRegistryURL)

	slackService, err := notifications.NewSlackService(logger)
	if err != nil {
		logger.Warnf("Failed to initialize Slack service: %v", err)
		slackService = nil
	}

	// Configure HTTP client with timeouts
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	return &ChainRegistry{
		cache:            cache.New(5*time.Minute, 10*time.Second),
		logger:           logger,
		client:           client,
		baseURL:          baseURL,
		slack:            slackService,
		chains:           make(map[string]*ChainInfo),
		monitoredChains:  []string{},
		githubAPIURL:     githubAPIURL,
		chainRegistryURL: chainRegistryURL,
	}
}

func (r *ChainRegistry) SetChainName(chainName string) {
	r.chainName = chainName
}

func (r *ChainRegistry) GetMonitoredChains() ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.monitoredChains, nil
}

func (r *ChainRegistry) GetUpgradeInfo(chainName string, forceRefresh bool) (*types.UpgradeInfo, error) {
	if chainName == "" {
		return nil, fmt.Errorf("chain name cannot be empty")
	}

	// Try to get from cache first if not forcing refresh
	if !forceRefresh {
		if cached, found := r.cache.Get(fmt.Sprintf(upgradeInfoCacheKey, chainName)); found {
			r.logger.Debugf("Found cached upgrade info for %s", chainName)
			if cached == nil {
				return nil, nil
			}
			return cached.(*types.UpgradeInfo), nil
		}
	}

	// Get chain info under a read lock first
	r.mu.RLock()
	chain, exists := r.chains[chainName]
	r.mu.RUnlock()

	// If we need to refresh or chain doesn't exist, get it under a write lock
	if forceRefresh || !exists {
		r.mu.Lock()
		var err error
		chain, err = r.fetchChainInfo(chainName)
		if err != nil {
			r.mu.Unlock()
			// Cache the nil result to prevent repeated failed lookups
			r.cache.Set(fmt.Sprintf(chainInfoCacheKey, chainName), nil, 5*time.Minute)
			return nil, err
		}
		r.chains[chainName] = chain
		r.mu.Unlock()
	}

	if chain == nil {
		// Cache the nil result to prevent repeated failed lookups
		r.cache.Set(fmt.Sprintf(chainInfoCacheKey, chainName), nil, 5*time.Minute)
		return nil, fmt.Errorf("chain %q not found", chainName)
	}

	// Try to get upgrade info from chain registry first
	chainUpgrade, err := r.getUpgradeInfoFromChain(chainName)
	if err != nil {
		r.logger.Debugf("Failed to get upgrade info from Chain Registry for %s: %v", chainName, err)
	} else if chainUpgrade != nil {
		upgradeInfo := r.convertUpgradeInfo(chainName, chain, chainUpgrade)
		// Cache the result
		r.cache.Set(fmt.Sprintf(upgradeInfoCacheKey, chainName), upgradeInfo, 5*time.Minute)
		return upgradeInfo, nil
	}

	// If that fails, try Polkachu
	polkachuUpgrade, err := r.fetchPolkachuUpgrades(chainName)
	if err != nil {
		r.logger.Debugf("Failed to get upgrade info from Polkachu for %s: %v", chainName, err)
	} else if polkachuUpgrade != nil {
		upgradeInfo := r.convertUpgradeInfo(chainName, chain, polkachuUpgrade)
		// Cache the result
		r.cache.Set(fmt.Sprintf(upgradeInfoCacheKey, chainName), upgradeInfo, 5*time.Minute)
		return upgradeInfo, nil
	}

	r.logger.Debugf("No upgrade information found for chain %s", chainName)
	// Cache the nil result to prevent repeated failed lookups
	r.cache.Set(fmt.Sprintf(upgradeInfoCacheKey, chainName), nil, 5*time.Minute)
	return nil, nil
}

func (r *ChainRegistry) GetMainnetUpgrades() ([]*types.UpgradeInfo, error) {
	// Get a copy of monitored chains under lock
	r.mu.RLock()
	chains := make([]string, len(r.monitoredChains))
	copy(chains, r.monitoredChains)
	r.mu.RUnlock()

	var (
		upgrades  = make([]*types.UpgradeInfo, 0)
		mu        sync.Mutex
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, 10) // Limit concurrent requests
	)

	for _, chainName := range chains {
		wg.Add(1)
		go func(chain string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			info, err := r.GetUpgradeInfo(chain, false)
			if err != nil {
				r.logger.Debugf("Skipping chain %q: %v", chain, err)
				return
			}
			if info != nil && info.Network == "mainnet" {
				mu.Lock()
				upgrades = append(upgrades, info)
				mu.Unlock()
			}
		}(chainName)
	}

	wg.Wait()
	return upgrades, nil
}

func (r *ChainRegistry) GetTestnetUpgrades() ([]*types.UpgradeInfo, error) {
	// Get a copy of monitored chains under lock
	r.mu.RLock()
	chains := make([]string, len(r.monitoredChains))
	copy(chains, r.monitoredChains)
	r.mu.RUnlock()

	var (
		upgrades  = make([]*types.UpgradeInfo, 0)
		mu        sync.Mutex
		wg        sync.WaitGroup
		semaphore = make(chan struct{}, 10) // Limit concurrent requests
	)

	for _, chainName := range chains {
		wg.Add(1)
		go func(chain string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			info, err := r.GetUpgradeInfo(chain, false)
			if err != nil {
				r.logger.Debugf("Skipping chain %q: %v", chain, err)
				return
			}
			if info != nil && info.Network == "testnet" {
				mu.Lock()
				upgrades = append(upgrades, info)
				mu.Unlock()
			}
		}(chainName)
	}

	wg.Wait()
	return upgrades, nil
}

func (r *ChainRegistry) GetChainInfo(chainName string, forceRefresh bool) (*ChainInfo, error) {
	// Try to get from cache first if not forcing refresh
	if !forceRefresh {
		if cached, found := r.cache.Get(fmt.Sprintf(chainInfoCacheKey, chainName)); found {
			r.logger.Debugf("Found cached chain info for %s", chainName)
			if cached == nil {
				return nil, fmt.Errorf("chain %q not found (cached result)", chainName)
			}
			return cached.(*ChainInfo), nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if !forceRefresh {
		if info, ok := r.chains[chainName]; ok {
			return info, nil
		}
	}

	// Try mainnet path first
	mainnetURL := fmt.Sprintf("%s%s/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)
	r.logger.Debugf("Attempting to fetch chain info from mainnet registry: %s", mainnetURL)
	info, err := r.fetchChainInfoFromURL(mainnetURL)
	if err != nil {
		// If mainnet fails, try testnet path
		testnetURL := fmt.Sprintf("%s%s/testnets/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)
		r.logger.Debugf("Mainnet fetch failed, trying testnet registry: %s", testnetURL)
		info, err = r.fetchChainInfoFromURL(testnetURL)
		if err != nil {
			// Cache the nil result to prevent repeated failed lookups
			r.cache.Set(fmt.Sprintf(chainInfoCacheKey, chainName), nil, 5*time.Minute)

			// Check if either error was due to network issues
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "no such host") ||
				strings.Contains(err.Error(), "i/o timeout") {
				return nil, fmt.Errorf("network error while fetching chain info for %q: %v", chainName, err)
			}

			// If chain is not found in either registry, provide a clear message
			if strings.Contains(err.Error(), "404") {
				return nil, fmt.Errorf("chain %q not found in either mainnet or testnet registry", chainName)
			}

			return nil, fmt.Errorf("failed to fetch chain info for %q: %v", chainName, err)
		}
		info.Network = "testnet"
		r.logger.Debugf("Successfully fetched chain info for %q from testnet registry", chainName)
	} else {
		info.Network = "mainnet"
		r.logger.Debugf("Successfully fetched chain info for %q from mainnet registry", chainName)
	}

	// Set the chain name if it's not already set
	if info.Name == "" {
		info.Name = chainName
	}

	r.chains[chainName] = info
	// Cache the result
	r.cache.Set(fmt.Sprintf(chainInfoCacheKey, chainName), info, 5*time.Minute)
	return info, nil
}

func (r *ChainRegistry) fetchChainInfo(chainName string) (*ChainInfo, error) {
	// Store original chain name for error messages
	originalName := chainName

	// Clean the chain name by removing any URL components or slashes
	chainName = r.cleanChainName(chainName)
	if chainName == "" {
		return nil, fmt.Errorf("invalid chain name: %q", originalName)
	}

	chainBaseName, network, exists := r.tryChainNameVariations(chainName)
	if !exists {
		if strings.HasSuffix(chainName, "testnet") {
			baseChainName := strings.TrimSuffix(chainName, "testnet")
			chainBaseName, network, exists = r.tryChainNameVariations(baseChainName)
		}

		if !exists {
			r.logger.Infof("Chain %q not found in chain-registry. Tried variations: %q, %q",
				chainName,
				chainName,
				strings.TrimSuffix(chainName, "testnet"))
			return nil, fmt.Errorf("chain %q not found in chain-registry", chainName)
		}
	}

	var url string
	if network == "mainnet" {
		url = fmt.Sprintf("%s/%s/%s/chain.json",
			strings.TrimRight(r.githubAPIURL, "/"),
			strings.TrimLeft(r.chainRegistryURL, "/"),
			chainBaseName)
	} else {
		url = fmt.Sprintf("%s/%s/testnets/%s/chain.json",
			strings.TrimRight(r.githubAPIURL, "/"),
			strings.TrimLeft(r.chainRegistryURL, "/"),
			chainBaseName)
	}

	chainInfo, err := r.fetchChainInfoFromURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chain info for %q from %s: %v", chainName, network, err)
	}

	chainInfo.Network = network
	if chainInfo.Name == "" {
		chainInfo.Name = chainName
	}

	return chainInfo, nil
}

func (r *ChainRegistry) cleanChainName(name string) string {
	if strings.Contains(name, "/") {
		parts := strings.Split(name, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			part := parts[i]
			if part == "chain.json" || part == "testnets" {
				continue
			}
			if part != "" && part != "master" && !strings.Contains(part, "github") {
				name = part
				break
			}
		}
	}

	// Remove file extensions
	if idx := strings.LastIndex(name, "."); idx != -1 {
		name = name[:idx]
	}

	// Remove any remaining slashes and spaces
	name = strings.Trim(strings.TrimSpace(name), "/")

	return name
}

func (r *ChainRegistry) tryChainNameVariations(chainName string) (string, string, bool) {
	// Clean up chain name
	chainName = r.cleanChainName(chainName)
	if chainName == "" {
		return "", "", false
	}

	// Ensure base URLs are properly formatted
	githubBase := strings.TrimRight(r.githubAPIURL, "/")
	registryBase := strings.TrimLeft(r.chainRegistryURL, "/")

	// Try mainnet first
	mainnetURL := fmt.Sprintf("%s/%s/%s/chain.json", githubBase, registryBase, chainName)
	r.logger.Debugf("Checking mainnet URL: %s", mainnetURL)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "HEAD", mainnetURL, nil)
	if err != nil {
		r.logger.Debugf("Failed to create request for mainnet URL: %v", err)
		return "", "", false
	}

	resp, err := r.client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		r.logger.Debugf("Found chain %q in mainnet registry", chainName)
		return chainName, "mainnet", true
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Try testnet
	testnetURL := fmt.Sprintf("%s/%s/testnets/%s/chain.json", githubBase, registryBase, chainName)
	r.logger.Debugf("Checking testnet URL: %s", testnetURL)

	req, err = http.NewRequestWithContext(ctx, "HEAD", testnetURL, nil)
	if err != nil {
		r.logger.Debugf("Failed to create request for testnet URL: %v", err)
		return "", "", false
	}

	resp, err = r.client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		r.logger.Debugf("Found chain %q in testnet registry", chainName)
		return chainName, "testnet", true
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Try removing numeric suffixes (e.g., "osmosis-1" -> "osmosis")
	baseChainName := chainName
	if idx := strings.LastIndexAny(chainName, "-_"); idx != -1 {
		suffix := chainName[idx+1:]
		if _, err := strconv.Atoi(suffix); err == nil {
			baseChainName = chainName[:idx]
			r.logger.Debugf("Trying base chain name without numeric suffix: %q", baseChainName)

			// Try mainnet with base name
			mainnetURL = fmt.Sprintf("%s/%s/%s/chain.json", githubBase, registryBase, baseChainName)
			r.logger.Debugf("Checking mainnet URL with base name: %s", mainnetURL)

			req, err = http.NewRequestWithContext(ctx, "HEAD", mainnetURL, nil)
			if err != nil {
				r.logger.Debugf("Failed to create request for base name mainnet URL: %v", err)
				return "", "", false
			}

			resp, err = r.client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				r.logger.Debugf("Found chain %q in mainnet registry using base name %q", chainName, baseChainName)
				return baseChainName, "mainnet", true
			}
			if resp != nil {
				resp.Body.Close()
			}

			// Try testnet with base name
			testnetURL = fmt.Sprintf("%s/%s/testnets/%s/chain.json", githubBase, registryBase, baseChainName)
			r.logger.Debugf("Checking testnet URL with base name: %s", testnetURL)

			req, err = http.NewRequestWithContext(ctx, "HEAD", testnetURL, nil)
			if err != nil {
				r.logger.Debugf("Failed to create request for base name testnet URL: %v", err)
				return "", "", false
			}

			resp, err = r.client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				r.logger.Debugf("Found chain %q in testnet registry using base name %q", chainName, baseChainName)
				return baseChainName, "testnet", true
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}

	// Log all attempted variations at once
	variations := []string{chainName}
	if baseChainName != chainName {
		variations = append(variations, baseChainName)
	}
	r.logger.Infof("Chain %q not found in chain registry. Attempted variations: %s", chainName, strings.Join(variations, ", "))
	return "", "", false
}

func (r *ChainRegistry) fetchChainInfoFromURL(url string) (*ChainInfo, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	var chainInfo ChainInfo
	if err := json.NewDecoder(resp.Body).Decode(&chainInfo); err != nil {
		return nil, err
	}

	return &chainInfo, nil
}

func (r *ChainRegistry) fetchPolkachuUpgrades(chainName string) (*PolkachuUpgrade, error) {
	polkachuURL := "https://polkachu.com/api/v2/chain_upgrades"
	resp, err := r.client.Get(polkachuURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Polkachu API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("polkachu API returned non-200 status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if response looks like HTML
	if strings.HasPrefix(strings.TrimSpace(string(bodyBytes)), "<") {
		return nil, fmt.Errorf("received HTML response instead of JSON")
	}

	// First try to unmarshal as direct array
	var upgrades []PolkachuUpgrade
	err = json.Unmarshal(bodyBytes, &upgrades)
	if err != nil {
		// If direct array fails, try wrapped response
		var response PolkachuResponse
		if err := json.Unmarshal(bodyBytes, &response); err != nil {
			r.logger.WithFields(logrus.Fields{
				"chain": chainName,
				"error": err,
				"body":  string(bodyBytes[:min(len(bodyBytes), 1000)]), // Log first 1000 chars of response
			}).Debug("Failed to parse Polkachu response")
			return nil, fmt.Errorf("failed to parse Polkachu response: %w", err)
		}
		upgrades = response.Data
	}

	// Log the number of upgrades found
	r.logger.WithFields(logrus.Fields{
		"chain":          chainName,
		"upgrades_count": len(upgrades),
	}).Debug("Retrieved upgrades from Polkachu")

	// Find matching upgrade
	for _, upgrade := range upgrades {
		// Try exact match first
		if upgrade.ChainName == chainName {
			return &upgrade, nil
		}

		// Try case-insensitive match
		if strings.EqualFold(upgrade.ChainName, chainName) {
			r.logger.WithFields(logrus.Fields{
				"chain":          chainName,
				"polkachu_chain": upgrade.ChainName,
				"version":        upgrade.NodeVersion,
				"block":          upgrade.Block,
				"estimated_time": upgrade.EstimatedUpgradeTime,
			}).Debug("Found chain with case-insensitive match")
			return &upgrade, nil
		}
	}

	return nil, fmt.Errorf("no upgrade found for chain %s", chainName)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *ChainRegistry) GetAllChains() ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var chains []string
	for chain := range r.chains {
		chains = append(chains, chain)
	}
	return chains, nil
}

func (r *ChainRegistry) GetUpgrades(chainID string) ([]Upgrade, error) {
	chains, err := r.GetMonitoredChains()
	if err != nil {
		return nil, fmt.Errorf("failed to get monitored chains: %w", err)
	}

	var upgrades []Upgrade
	for _, chainName := range chains {
		chainInfo, err := r.GetChainInfo(chainName, false)
		if err != nil {
			r.logger.Warnf("Failed to get chain info for %s: %v", chainName, err)
			continue
		}

		if chainInfo.Network != chainID {
			continue
		}

		upgradeInfo, err := r.GetUpgradeInfo(chainName, false)
		if err != nil {
			r.logger.Warnf("Failed to get upgrade info for %s: %v", chainName, err)
			continue
		}

		if upgradeInfo != nil {
			upgrades = append(upgrades, upgradeInfo)
		}
	}

	return upgrades, nil
}

func (r *ChainRegistry) SetMonitoredChains(chains []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.monitoredChains = chains
}

func (r *ChainRegistry) ChainExists(chainName string) bool {
	if chainName == "" {
		return false
	}

	mainnetURL := fmt.Sprintf("%s%s/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)
	resp, err := r.client.Head(mainnetURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return true
	}
	if resp != nil {
		resp.Body.Close()
	}

	testnetURL := fmt.Sprintf("%s%s/testnets/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)
	resp, err = r.client.Head(testnetURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return true
	}
	if resp != nil {
		resp.Body.Close()
	}

	return false
}

func (r *ChainRegistry) FilterExistingChains(chains []string) []string {
	var existingChains []string
	for _, chain := range chains {
		if r.ChainExists(chain) {
			existingChains = append(existingChains, chain)
		}
	}
	return existingChains
}

func (r *ChainRegistry) getUpgradeInfoFromChain(chainName string) (*types.UpgradeInfo, error) {
	url := fmt.Sprintf("%s%s/%s/upgrades.json", r.githubAPIURL, r.chainRegistryURL, chainName)
	resp, err := r.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	var upgradeInfo types.UpgradeInfo
	if err := json.NewDecoder(resp.Body).Decode(&upgradeInfo); err != nil {
		return nil, err
	}

	return &upgradeInfo, nil
}

func (r *ChainRegistry) convertUpgradeInfo(chainName string, chain *ChainInfo, upgrade interface{}) *types.UpgradeInfo {
	switch u := upgrade.(type) {
	case *types.UpgradeInfo:
		return &types.UpgradeInfo{
			Name:             u.Name,
			ChainName:        chainName,
			Height:           u.Height,
			Info:             u.Info,
			Time:             u.Time,
			Version:          u.Name,
			Estimated:        true,
			Network:          chain.Network,
			ProposalLink:     u.ProposalLink,
			Guide:            u.Info,
			BlockLink:        "",
			CosmovisorFolder: fmt.Sprintf("upgrades/%s", u.Name),
			GitHash:          "",
			Repo:             "",
			RPC:              "",
			API:              "",
		}
	case *PolkachuUpgrade:
		estimatedTime, err := time.Parse(time.RFC3339, u.EstimatedUpgradeTime)
		if err != nil {
			r.logger.Warnf("Failed to parse time from Polkachu response: %v", err)
			estimatedTime = time.Now()
		}

		r.logger.WithFields(logrus.Fields{
			"chain":          chainName,
			"node_version":   u.NodeVersion,
			"network":        u.Network,
			"block":          u.Block,
			"estimated_time": u.EstimatedUpgradeTime,
			"repo":           u.Repo,
			"git_hash":       u.GitHash,
			"block_link":     u.BlockLink,
			"proposal_link":  u.Proposal,
			"rpc":            u.RPC,
			"api":            u.API,
		}).Debug("Processing Polkachu upgrade info")

		return &types.UpgradeInfo{
			Name:             chainName,
			ChainName:        chainName,
			Height:           u.Block,
			Info:             u.Guide,
			Time:             estimatedTime,
			Version:          u.NodeVersion,
			Estimated:        true,
			Network:          chain.Network,
			ProposalLink:     u.Proposal,
			Guide:            u.Guide,
			BlockLink:        u.BlockLink,
			CosmovisorFolder: u.CosmovisorFolder,
			GitHash:          u.GitHash,
			Repo:             u.Repo,
			RPC:              u.RPC,
			API:              u.API,
		}
	default:
		return nil
	}
}

func (r *ChainRegistry) IsUpgradeCached(chainName string) bool {
	_, found := r.cache.Get(fmt.Sprintf("upgrade_info:%s", chainName))
	return found
}
