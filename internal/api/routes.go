package api

import (
	"github.com/gorilla/mux"
)

func SetupRoutes(router *mux.Router, handler *Handler) {
	router.HandleFunc("/api/v1/health", handler.HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/upgrades", handler.GetUpgrades).Methods("GET")
	router.HandleFunc("/api/v1/chains/{chainName}", handler.GetChainInfo).Methods("GET")
}
