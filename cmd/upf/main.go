package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"5gcore/internal/nf"
)

func main() {
	service := nf.NewService("UPF", "upf", []string{"user-plane", "forwarding"}, "User Plane Function")
	if service.NRFURL == "" {
		service.NRFURL = "http://127.0.0.1:8000"
	}
	if err := service.Register(); err != nil {
		log.Printf("register with NRF failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.HealthHandler())
	mux.HandleFunc("/status", service.StatusHandler())
	mux.HandleFunc("/forward", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid forward payload", http.StatusBadRequest)
			return
		}
		response := map[string]any{
			"status":       "forwarding-active",
			"upf":          service.Name,
			"source":       request["source"],
			"destination":  request["destination"],
			"trafficClass": "QCI-9",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("UPF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
