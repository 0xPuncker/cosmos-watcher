package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func StartServer(ctx context.Context, handler *Handler, port string) error {
	router := mux.NewRouter()

	router.Use(loggingMiddleware(handler.logger))
	router.Use(corsMiddleware)

	router.HandleFunc("/api/v1/health", handler.HealthCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/upgrades/mainnet", handler.GetMainnetUpgrades).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/upgrades/testnet", handler.GetTestnetUpgrades).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/chains/{chainName}", handler.GetChainInfo).Methods(http.MethodGet)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			handler.logger.Fatalf("Server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	return nil
}

func loggingMiddleware(logger *logrus.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := newResponseWriter(w)
			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			var durationStr string
			if duration < time.Millisecond {
				durationStr = fmt.Sprintf("%.2fµs", float64(duration.Microseconds()))
			} else if duration < time.Second {
				durationStr = fmt.Sprintf("%.2fms", float64(duration.Milliseconds()))
			} else {
				durationStr = fmt.Sprintf("%.2fs", duration.Seconds())
			}

			logger.WithFields(logrus.Fields{
				"method":     r.Method,
				"path":       r.URL.Path,
				"status":     rw.status,
				"duration":   durationStr,
				"user_agent": r.UserAgent(),
				"remote_ip":  r.RemoteAddr,
			}).Info("Request processed")
		})
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
