package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"5gcore/internal/nf"
)

func main() {
	service := nf.NewService("AMF", "amf", []string{"registration", "session-management"}, "Access and Mobility Management Function")
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
		if raw, err := service.Discover("smf"); err == nil {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(raw)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	handleProcedure := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid attach payload", http.StatusBadRequest)
			return
		}
		imsi := fmt.Sprint(request["imsi"])
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		ausfURL, err := service.DiscoverURL("ausf")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to discover AUSF: %v", err), http.StatusServiceUnavailable)
			return
		}
		var authResponse map[string]any
		if err := service.CallJSON(http.MethodPost, ausfURL+"/authenticate", map[string]any{"imsi": imsi, "securityMode": "5G-AKA", "plmn": "00101"}, &authResponse); err != nil {
			http.Error(w, fmt.Sprintf("AUSF authentication failed: %v", err), http.StatusBadGateway)
			return
		}
		udmURL, err := service.DiscoverURL("udm")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to discover UDM: %v", err), http.StatusServiceUnavailable)
			return
		}
		var profile map[string]any
		if err := service.CallJSON(http.MethodPost, udmURL+"/subscriber-profile", map[string]any{"imsi": imsi, "requestedSlice": "slices/default"}, &profile); err != nil {
			http.Error(w, fmt.Sprintf("UDM lookup failed: %v", err), http.StatusBadGateway)
			return
		}
		smfURL, err := service.DiscoverURL("smf")
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to discover SMF: %v", err), http.StatusServiceUnavailable)
			return
		}
		var session map[string]any
		if err := service.CallJSON(http.MethodPost, smfURL+"/create-session", map[string]any{"imsi": imsi, "dnn": "internet", "sNssai": "slice-1", "pduSessionId": 1}, &session); err != nil {
			http.Error(w, fmt.Sprintf("SMF session setup failed: %v", err), http.StatusBadGateway)
			return
		}
		response := map[string]any{
			"procedure":        "5g-registration-and-session-setup",
			"status":           "attach-accepted",
			"imsi":             imsi,
			"amf":              service.Name,
			"authResult":       authResponse,
			"subscriptionInfo": profile,
			"sessionContext":   session,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
	mux.HandleFunc("/procedure/attach", handleProcedure)
	mux.HandleFunc("/procedure/registration", handleProcedure)
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("AMF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
