package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const apiPath = "/api/v1"

func TestHealthCheck(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(nil)

	registry := chain.NewChainRegistry(
		logger,
		"https://api.github.com",
		"/cosmos/chain-registry/master",
	)

	handler := NewHandler(registry, logger, &config.Config{})

	req, err := http.NewRequest("GET", apiPath+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/health", handler.HealthCheck)
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "ok", response["status"])
}

func TestGetChainInfo(t *testing.T) {
	handler := setupTestHandler()

	req, err := http.NewRequest("GET", apiPath+"/chains/cosmoshub", nil)
	if err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"chainName": "cosmoshub",
	}
	req = mux.SetURLVars(req, vars)

	rr := httptest.NewRecorder()
	handler.GetChainInfo(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response chain.ChainInfo
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response.Name != "cosmoshub" {
		t.Errorf("handler returned unexpected chain name: got %v want %v", response.Name, "cosmoshub")
	}
}

func setupTestHandler() *Handler {
	logger := logrus.New()
	logger.SetOutput(nil)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
		},
		GitHub: config.GitHubConfig{
			APIURL: "https://api.github.com",
		},
		Registry: config.RegistryConfig{
			URL: "https://raw.githubusercontent.com/cosmos/chain-registry/master",
		},
		Poller: config.PollerConfig{
			Interval: "5m",
		},
	}

	registry := chain.NewChainRegistry(logger, cfg.Registry.URL, cfg.GitHub.APIURL)
	return NewHandler(registry, logger, cfg)
}
