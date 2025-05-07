package chain

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestChainRegistry_GetChainInfo(t *testing.T) {
	tests := []struct {
		name          string
		responseCode  int
		responseBody  interface{}
		expectedError bool
	}{
		{
			name:         "successful chain info fetch",
			responseCode: http.StatusOK,
			responseBody: ChainInfo{
				Name:    "testchain",
				ChainID: "testchain-1",
				APIs: APIs{
					RPC: []Endpoint{
						{Address: "http://localhost:26657"},
					},
					REST: []Endpoint{
						{Address: "http://localhost:1317"},
					},
				},
				Explorers: []Explorer{
					{
						Kind:   "test",
						URL:    "https://explorer.testchain.com",
						TxPage: "https://explorer.testchain.com/tx/{txHash}",
					},
				},
			},
			expectedError: false,
		},
		{
			name:          "non-existent chain",
			responseCode:  http.StatusNotFound,
			responseBody:  nil,
			expectedError: true,
		},
		{
			name:          "malformed response",
			responseCode:  http.StatusOK,
			responseBody:  "invalid json",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				if tt.responseBody != nil {
					json.NewEncoder(w).Encode(tt.responseBody)
				}
			}))
			defer ts.Close()

			logger := logrus.New()
			registry := NewChainRegistry(logger, "https://api.github.com", ts.URL)

			info, err := registry.GetChainInfo("testchain", false)
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, info)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, info)
				assert.Equal(t, "testchain", info.Name)
			}
		})
	}
}

func TestChainRegistry_GetUpgradeInfo(t *testing.T) {
	tests := []struct {
		name          string
		polkachuResp  interface{}
		expectedError bool
	}{
		{
			name: "successful upgrade info fetch",
			polkachuResp: []PolkachuUpgrade{
				{
					Network:              "testchain",
					NodeVersion:          "v1.0.0",
					Block:                1000000,
					EstimatedUpgradeTime: "2025-01-01T00:00:00Z",
					Guide:                "https://example.com/guide",
				},
			},
			expectedError: false,
		},
		{
			name:          "no upgrades available",
			polkachuResp:  []PolkachuUpgrade{},
			expectedError: false,
		},
		{
			name:          "polkachu error",
			polkachuResp:  nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polkachuServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.polkachuResp != nil {
					json.NewEncoder(w).Encode(tt.polkachuResp)
				} else {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}))
			defer polkachuServer.Close()

			logger := logrus.New()
			registry := NewChainRegistry(logger, "https://api.github.com", "https://chain-registry.example.com")
			registry.cache.Set("testchain", &ChainInfo{
				Name:    "testchain",
				Network: "mainnet",
			}, cache.DefaultExpiration)

			upgrade, err := registry.GetUpgradeInfo("testchain", false)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, upgrade)
				if tt.polkachuResp != nil && len(tt.polkachuResp.([]PolkachuUpgrade)) > 0 {
					polkachuUpgrade := tt.polkachuResp.([]PolkachuUpgrade)[0]
					assert.Equal(t, polkachuUpgrade.NodeVersion, upgrade.Name)
					assert.Equal(t, polkachuUpgrade.Block, upgrade.Height)
				}
			}
		})
	}
}

func TestChainRegistry_GetAllChains(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseCode   int
		expectedChains []string
		expectError    bool
	}{
		{
			name:           "successful response",
			responseBody:   `["cosmoshub", "osmosis", "juno"]`,
			responseCode:   http.StatusOK,
			expectedChains: []string{"cosmoshub", "osmosis", "juno"},
			expectError:    false,
		},
		{
			name:           "empty response",
			responseBody:   `[]`,
			responseCode:   http.StatusOK,
			expectedChains: []string{},
			expectError:    false,
		},
		{
			name:           "server error",
			responseBody:   `{"error": "internal server error"}`,
			responseCode:   http.StatusInternalServerError,
			expectedChains: nil,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			responseBody:   `invalid json`,
			responseCode:   http.StatusOK,
			expectedChains: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			logger := logrus.New()
			cr := NewChainRegistry(logger, "https://api.github.com", server.URL)

			chains, err := cr.GetAllChains()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(chains) != len(tt.expectedChains) {
				t.Errorf("expected %d chains, got %d", len(tt.expectedChains), len(chains))
				return
			}

			for i, chain := range chains {
				if chain != tt.expectedChains[i] {
					t.Errorf("expected chain %s at index %d, got %s", tt.expectedChains[i], i, chain)
				}
			}
		})
	}
}
