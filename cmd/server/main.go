package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dimiro1/banner"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/mattn/go-colorable"
	"github.com/p2p/devops-cosmos-watcher/internal/api"
	"github.com/p2p/devops-cosmos-watcher/internal/chain"
	"github.com/p2p/devops-cosmos-watcher/internal/config"
	"github.com/p2p/devops-cosmos-watcher/internal/cron"
	"github.com/p2p/devops-cosmos-watcher/internal/notifications"
	"github.com/p2p/devops-cosmos-watcher/internal/poller"
	"github.com/sirupsen/logrus"
)

const bannerText = `
{{ .Title "Cosmos Watcher" "" 0 }} 
{{ .AnsiBackground.BrightBlue }}{{ .AnsiColor.White }}
{{ .AnsiReset }}
`

func loggingMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{w, http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.status,
				"duration":   duration.String(),
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
			}).Info("Request processed")
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

type PolkachuUpgrade struct {
	Name        string    `json:"name"`
	Height      int64     `json:"height"`
	TargetDate  time.Time `json:"target_date"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}

type PolkachuResponse struct {
	Data []PolkachuUpgrade `json:"data"`
}

func main() {
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load(".env.local"); err != nil {
			fmt.Printf("No .env or .env.local file found. Using environment variables.\n")
		}
	}

	banner.Init(colorable.NewColorableStdout(), true, true, strings.NewReader(bannerText))

	configPath := flag.String("config", "config/config.json", "path to config file")
	flag.Parse()

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:          false,
		DisableTimestamp:       false,
		TimestampFormat:        "2006-01-02T15:04:05-07:00",
		DisableLevelTruncation: false,
		PadLevelText:           false,
	})

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	githubAPIURL := os.Getenv("GITHUB_API_URL")
	if githubAPIURL == "" {
		githubAPIURL = "https://raw.githubusercontent.com"
	}
	chainRegistryURL := os.Getenv("CHAIN_REGISTRY_BASE_URL")
	if chainRegistryURL == "" {
		chainRegistryURL = "/cosmos/chain-registry/master"
	}

	logger.Debugf("GitHub API URL: %s", githubAPIURL)
	logger.Debugf("Chain Registry URL: %s", chainRegistryURL)

	registry := chain.NewChainRegistry(
		logger,
		githubAPIURL,
		chainRegistryURL,
	)

	handler := api.NewHandler(registry, logger, cfg)

	loadChainsJob := cron.NewLoadChainsJob(registry, logger)
	handler.Scheduler.RegisterTask("load-chains", loadChainsJob.Run)

	if err := loadChainsJob.Run(); err != nil {
		logger.Fatalf("Failed to load initial chains: %v", err)
	}

	router := mux.NewRouter()
	router.Use(loggingMiddleware(logger))

	router.HandleFunc("/api/v1/health", handler.HealthCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/upgrades", handler.GetUpgrades).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chains/{chainName}", handler.GetChainInfo).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/jobs", handler.ListJobs).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/scheduler/start", handler.StartScheduler).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/scheduler/stop", handler.StopScheduler).Methods(http.MethodPost)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	interval, err := time.ParseDuration(cfg.Poller.Interval)
	if err != nil {
		logger.Fatalf("Invalid poller interval: %v", err)
	}
	p := poller.New(registry, logger, interval)

	go p.Start()

	slack, err := notifications.NewSlackService(logger)
	if err != nil {
		logger.Warnf("Failed to initialize Slack service: %v", err)
	}

	startupNotifier := notifications.NewStartupNotifier(registry, slack, logger)
	go startupNotifier.NotifyStartup()

	if err := handler.Scheduler.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Infof("Server started on port %s - Press Ctrl+C to stop.", cfg.Server.Port)

	<-stop
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Stop()
	handler.Scheduler.Stop()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server shutdown failed: %v", err)
	}

	logger.Info("Server stopped")
}
