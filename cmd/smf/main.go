package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"5gcore/internal/nf"
)

func main() {
	service := nf.NewService("SMF", "smf", []string{"session-management", "pdu-session"}, "Session Management Function")
	if service.NRFURL == "" {
		service.NRFURL = "http://127.0.0.1:8000"
	}
	if err := service.Register(); err != nil {
		log.Printf("register with NRF failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.HealthHandler())
	mux.HandleFunc("/status", service.StatusHandler())
	mux.HandleFunc("/discover", func(w http.ResponseWriter, r *http.Request) {
		if raw, err := service.Discover("amf"); err == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(raw)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/create-session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid session payload", http.StatusBadRequest)
			return
		}
		imsi := fmt.Sprint(request["imsi"])
		response := map[string]any{
			"status":             "session-created",
			"smf":                service.Name,
			"imsi":               imsi,
			"pduSessionId":       request["pduSessionId"],
			"dnn":                request["dnn"],
			"sNssai":             request["sNssai"],
			"upfAnchor":          "upf:80",
			"sessionAmbr":        "1 Gbps",
			"notificationTarget": "amf",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("SMF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
