package poller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPollerConfiguration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	testCases := []struct {
		name     string
		interval time.Duration
	}{
		{"Default interval", 5 * time.Minute},
		{"Short interval", 1 * time.Minute},
		{"Long interval", 1 * time.Hour},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			registry := chain.NewChainRegistry(logger, "https://example.com", "/test")
			poller := New(registry, logger, tc.interval)

			assert.NotNil(t, poller)
			assert.Equal(t, tc.interval, poller.interval)
		})
	}
}

func TestPollerUpdateCycle(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	height := 1000
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"name": "testchain",
			"chain_id": "testchain-1",
			"version": "v1.0.0",
			"height": %d
		}`
		height += 100 // Increment height for each call
		w.Write([]byte(fmt.Sprintf(response, height)))
	}))
	defer server.Close()

	registry := chain.NewChainRegistry(logger, server.URL, "/test")
	registry.SetMonitoredChains([]string{"testchain"})

	poller := New(registry, logger, 100*time.Millisecond)
	assert.NotNil(t, poller)

	go poller.Start()

	time.Sleep(350 * time.Millisecond)

	poller.Stop()

	chainInfo, err := registry.GetChainInfo("testchain", false)
	assert.NoError(t, err)
	assert.NotNil(t, chainInfo)
	assert.Equal(t, "testchain", chainInfo.Name)
	assert.Equal(t, "v1.0.0", chainInfo.Version)
	assert.Greater(t, chainInfo.Height, int64(1000)) // Height should have increased
}
