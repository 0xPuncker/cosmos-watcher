package poller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xPuncker/cosmos-watcher/internal/chain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewRegistryPoller(t *testing.T) {
	tests := []struct {
		name          string
		interval      time.Duration
		expectedError bool
	}{
		{
			name:          "default interval",
			interval:      5 * time.Minute,
			expectedError: false,
		},
		{
			name:          "custom interval",
			interval:      1 * time.Minute,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logrus.New()
			registry := chain.NewChainRegistry(logger, "https://api.github.com", "https://chain-registry.example.com")

			poller := NewRegistryPoller(registry, logger, tt.interval)

			assert.NotNil(t, poller)
			assert.Equal(t, logger, poller.logger)
			assert.Equal(t, registry, poller.registry)
			assert.Equal(t, tt.interval, poller.interval)
		})
	}
}

func TestUpdateChain(t *testing.T) {
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`["chain1", "chain2"]`))
		} else if r.URL.Path == "/chain1/chain.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "chain1",
				"chain_id": "chain1-1",
				"version": "v1.0.0",
				"height": 1000000,
				"apis": {
					"rpc": [{"address": "http://localhost:26657"}],
					"rest": [{"address": "http://localhost:1317"}]
				}
			}`))
		} else if r.URL.Path == "/chain2/chain.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "chain2",
				"chain_id": "chain2-1",
				"version": "v1.0.0",
				"height": 2000000,
				"apis": {
					"rpc": [{"address": "http://localhost:26657"}],
					"rest": [{"address": "http://localhost:1317"}]
				}
			}`))
		}
	}))
	defer registryServer.Close()

	logger := logrus.New()
	registry := chain.NewChainRegistry(logger, "https://api.github.com", registryServer.URL)

	poller := NewRegistryPoller(registry, logger, 5*time.Minute)

	err := poller.updateChain("chain1")
	assert.NoError(t, err)
}

func TestUpdate(t *testing.T) {
	registryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_list" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`["chain1", "chain2"]`))
		} else if r.URL.Path == "/chain1/chain.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "chain1",
				"chain_id": "chain1-1",
				"version": "v1.0.0",
				"height": 1000000,
				"apis": {
					"rpc": [{"address": "http://localhost:26657"}],
					"rest": [{"address": "http://localhost:1317"}]
				}
			}`))
		} else if r.URL.Path == "/chain2/chain.json" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "chain2",
				"chain_id": "chain2-1",
				"version": "v1.0.0",
				"height": 2000000,
				"apis": {
					"rpc": [{"address": "http://localhost:26657"}],
					"rest": [{"address": "http://localhost:1317"}]
				}
			}`))
		}
	}))
	defer registryServer.Close()

	logger := logrus.New()
	registry := chain.NewChainRegistry(logger, "https://api.github.com", registryServer.URL)

	poller := NewRegistryPoller(registry, logger, 5*time.Minute)

	poller.update()
}
