package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"5gcore/internal/common"
	"5gcore/internal/nrf"
)

var registry = nrf.NewRegistry()

func main() {
	host := common.GetEnv("NRF_HOST", common.DefaultNRFHost)
	port := common.GetEnvInt("NRF_PORT", common.DefaultNRFPort)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":       "ok",
			"name":         "NRF",
			"service":      "nrf",
			"registrySize": len(registry.List()),
		})
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":     "NRF",
			"service":  "nrf",
			"status":   "ready",
			"registry": registry.List(),
		})
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var nf nrf.NFRecord
		if err := json.NewDecoder(r.Body).Decode(&nf); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if nf.Name == "" || nf.Service == "" {
			http.Error(w, "name and service are required", http.StatusBadRequest)
			return
		}
		if nf.Host == "" {
			nf.Host = common.DefaultNRFHost
		}
		if nf.Port == 0 {
			nf.Port = common.DefaultNRFPort
		}
		if nf.URL == "" {
			nf.URL = fmt.Sprintf("http://%s:%d", nf.Host, nf.Port)
		}
		registered := registry.Register(nf)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":       "registered",
			"name":         registered.Name,
			"service":      registered.Service,
			"registrySize": len(registry.List()),
		})
	})

	http.HandleFunc("/discover/", func(w http.ResponseWriter, r *http.Request) {
		serviceName := strings.TrimPrefix(r.URL.Path, "/discover/")
		if serviceName == "" {
			http.Error(w, "service name is required", http.StatusBadRequest)
			return
		}
		if nf, ok := registry.Find(serviceName); ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(nf)
			return
		}
		http.NotFound(w, r)
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("NRF listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
