package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/internal/config"
	"github.com/p2p/devops-cosmos-watcher/internal/cron"
	"github.com/p2p/devops-cosmos-watcher/internal/notifications"
	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	registry       *chain.ChainRegistry
	logger         *logrus.Logger
	config         *config.Config
	Scheduler      *cron.Scheduler
	upgradeChecker *cron.UpgradeChecker
}

type ChainUpgrade struct {
	Name             string `json:"name"`
	Network          string `json:"network"`
	Version          string `json:"version"`
	Height           int64  `json:"height,omitempty"`
	EstimatedAt      string `json:"estimated_at,omitempty"`
	Guide            string `json:"guide,omitempty"`
	ProposalLink     string `json:"proposal_link,omitempty"`
	BlockLink        string `json:"block_link,omitempty"`
	CosmovisorFolder string `json:"cosmovisor_folder,omitempty"`
	GitHash          string `json:"git_hash,omitempty"`
	Repo             string `json:"repo,omitempty"`
	RPC              string `json:"rpc,omitempty"`
	API              string `json:"api,omitempty"`
}

type UpgradesResponse struct {
	Chains      []ChainUpgrade `json:"chains"`
	LastUpdated time.Time      `json:"last_updated"`
}

func NewHandler(registry *chain.ChainRegistry, logger *logrus.Logger, cfg *config.Config) *Handler {
	scheduler := cron.NewScheduler(logger, types.JobConfig{
		MaxConcurrent: cfg.Jobs.MaxConcurrent,
		Predefined:    cfg.Jobs.Predefined,
	})

	slack, err := notifications.NewSlackService(logger)
	if err != nil {
		logger.Warnf("Failed to initialize Slack service: %v", err)
	}

	upgradeChecker := cron.NewUpgradeChecker(registry, logger, slack)

	scheduler.RegisterTask("check-upgrades", func() error {
		start := time.Now()
		logger.WithFields(logrus.Fields{
			"task":      "check-upgrades",
			"timestamp": start.Format(time.RFC3339),
		}).Info("Starting scheduled upgrade check")

		upgradeChecker.CheckUpgrades()

		duration := time.Since(start)
		logger.WithFields(logrus.Fields{
			"task":      "check-upgrades",
			"duration":  duration.String(),
			"timestamp": time.Now().Format(time.RFC3339),
		}).Info("Completed scheduled upgrade check")

		return nil
	})

	loadChainsJob := cron.NewLoadChainsJob(registry, logger)
	scheduler.RegisterTask("load-chains", loadChainsJob.Run)

	if err := scheduler.LoadPredefinedJobs(cfg.Jobs.Predefined); err != nil {
		logger.Fatalf("Failed to load predefined jobs: %v", err)
	}

	return &Handler{
		registry:       registry,
		logger:         logger,
		config:         cfg,
		Scheduler:      scheduler,
		upgradeChecker: upgradeChecker,
	}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

func (h *Handler) GetMainnetUpgrades(w http.ResponseWriter, r *http.Request) {
	upgrades, err := h.registry.GetUpgrades("mainnet")
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upgrades)
}

func (h *Handler) GetTestnetUpgrades(w http.ResponseWriter, r *http.Request) {
	upgrades, err := h.registry.GetUpgrades("testnet")
	if err != nil {
		h.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upgrades)
}

func (h *Handler) GetChainInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chainName := vars["chainName"]

	chainInfo, err := h.registry.GetChainInfo(chainName, false)
	if err != nil {
		h.handleError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chainInfo)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	jobs := h.Scheduler.ListJobs()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs":        jobs,
		"active_jobs": len(jobs),
	})
}

func (h *Handler) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]

	enabled, description, err := h.Scheduler.GetJobStatus(jobName)
	if err != nil {
		h.handleError(w, err, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        jobName,
		"enabled":     enabled,
		"description": description,
	})
}

func (h *Handler) StartScheduler(w http.ResponseWriter, r *http.Request) {
	if err := h.Scheduler.Start(); err != nil {
		h.handleError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "scheduler started successfully",
	})
}

func (h *Handler) StopScheduler(w http.ResponseWriter, r *http.Request) {
	h.Scheduler.Stop()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "scheduler stopped successfully",
	})
}

func (h *Handler) GetUpgrades(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	chains, err := h.registry.GetMonitoredChains()
	if err != nil {
		h.logger.Errorf("Failed to get monitored chains: %v", err)
		http.Error(w, "Failed to get monitored chains", http.StatusInternalServerError)
		return
	}

	h.logger.Debugf("Found %d monitored chains", len(chains))

	response := UpgradesResponse{
		Chains:      make([]ChainUpgrade, 0),
		LastUpdated: time.Now(),
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		semaphore = make(chan struct{}, 10)
	)

	for _, chainName := range chains {
		select {
		case <-ctx.Done():
			h.logger.Error("Request timeout while processing chains")
			http.Error(w, "Request timeout", http.StatusGatewayTimeout)
			return
		default:
			wg.Add(1)
			go func(name string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				upgradeInfo, err := h.registry.GetUpgradeInfo(name, false)
				if err != nil {
					h.logger.Debugf("Failed to get upgrade info for %s: %v", name, err)
					return
				}

				if upgradeInfo != nil {
					mu.Lock()
					response.Chains = append(response.Chains, ChainUpgrade{
						Name:             upgradeInfo.GetChainName(),
						Network:          upgradeInfo.GetNetwork(),
						Version:          upgradeInfo.GetVersion(),
						Height:           upgradeInfo.GetHeight(),
						EstimatedAt:      upgradeInfo.GetEstimatedUpgradeTime(),
						Guide:            upgradeInfo.GetGuide(),
						ProposalLink:     upgradeInfo.GetProposalLink(),
						BlockLink:        upgradeInfo.GetBlockLink(),
						CosmovisorFolder: upgradeInfo.GetCosmovisorFolder(),
						GitHash:          upgradeInfo.GetGitHash(),
						Repo:             upgradeInfo.GetRepo(),
						RPC:              upgradeInfo.GetRPC(),
						API:              upgradeInfo.GetAPI(),
					})
					mu.Unlock()
				}
			}(chainName)
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		h.logger.Error("Request timeout while waiting for chains to process")
		http.Error(w, "Request timeout", http.StatusGatewayTimeout)
		return
	case <-done:

		sort.Slice(response.Chains, func(i, j int) bool {
			if response.Chains[i].Name == response.Chains[j].Name {
				return response.Chains[i].Network < response.Chains[j].Network
			}
			return response.Chains[i].Name < response.Chains[j].Name
		})

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		if err := json.NewEncoder(w).Encode(response); err != nil {
			h.logger.Errorf("Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

		h.logRequestProcessed(r, http.StatusOK)
	}
}

func (h *Handler) handleError(w http.ResponseWriter, err error, code int) {
	h.logger.Error(err)
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

func (h *Handler) logRequestProcessed(r *http.Request, status int) {
	duration := time.Since(time.Now())

	var durationStr string
	if duration < time.Millisecond {
		durationStr = fmt.Sprintf("%.2fms", float64(duration.Microseconds())/1000.0)
	} else if duration < time.Second {
		durationStr = fmt.Sprintf("%.2fms", float64(duration.Milliseconds()))
	} else {
		durationStr = fmt.Sprintf("%.2fs", duration.Seconds())
	}

	h.logger.WithFields(logrus.Fields{
		"method":     r.Method,
		"path":       r.URL.Path,
		"status":     status,
		"duration":   durationStr,
		"remote_ip":  r.RemoteAddr,
		"user_agent": r.UserAgent(),
	}).Info("Request processed")
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/health", h.HealthCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/upgrades/mainnet", h.GetMainnetUpgrades).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/upgrades/testnet", h.GetTestnetUpgrades).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chains/{chainName}", h.GetChainInfo).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/jobs", h.ListJobs).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/jobs/{name}", h.GetJobStatus).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/scheduler/start", h.StartScheduler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/scheduler/stop", h.StopScheduler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/upgrades", h.GetUpgrades).Methods(http.MethodGet)
	router.ServeHTTP(w, r)
}
