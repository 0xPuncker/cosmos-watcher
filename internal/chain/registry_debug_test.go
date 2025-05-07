package chain

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type TestRegistry struct {
	githubAPIURL     string
	chainRegistryURL string
	client           *http.Client
	logger           *logrus.Logger
}

func (r *TestRegistry) tryChainNameVariations(chainName string) (string, string, bool) {
	mainnetURL := fmt.Sprintf("%s%s/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)
	testnetURL := fmt.Sprintf("%s%s/testnets/%s/chain.json", r.githubAPIURL, r.chainRegistryURL, chainName)

	fmt.Printf("Testing mainnet URL: %s\n", mainnetURL)
	resp, err := r.client.Head(mainnetURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		fmt.Printf("Found in mainnet\n")
		return chainName, "mainnet", true
	}
	if resp != nil {
		resp.Body.Close()
	}

	fmt.Printf("Testing testnet URL: %s\n", testnetURL)
	resp, err = r.client.Head(testnetURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		fmt.Printf("Found in testnet\n")
		return chainName, "testnet", true
	}
	if resp != nil {
		resp.Body.Close()
	}

	fmt.Printf("Not found in either mainnet or testnet\n")
	return "", "", false
}

func TestChainVariations(t *testing.T) {
	registry := &TestRegistry{
		githubAPIURL:     "https://raw.githubusercontent.com",
		chainRegistryURL: "/cosmos/chain-registry/master",
		client:           &http.Client{Timeout: 10 * time.Second},
		logger:           logrus.New(),
	}

	chainName := "lava"
	t.Logf("\nTesting chain: %s", chainName)
	name, network, exists := registry.tryChainNameVariations(chainName)
	t.Logf("Result: name=%s, network=%s, exists=%v", name, network, exists)
}
