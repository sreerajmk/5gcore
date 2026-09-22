package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"5gcore/internal/nf"
)

var (
	sessionStateMu sync.RWMutex
	sessionState   = map[string]map[string]any{}
)

func getSessionState(imsi string) map[string]any {
	sessionStateMu.RLock()
	defer sessionStateMu.RUnlock()
	if state, ok := sessionState[imsi]; ok {
		return state
	}
	return nil
}

func setSessionState(imsi string, state map[string]any) {
	sessionStateMu.Lock()
	defer sessionStateMu.Unlock()
	sessionState[imsi] = state
}

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
		pduSessionID := request["pduSessionId"]
		state := map[string]any{
			"imsi":         imsi,
			"state":        "active",
			"smf":          service.Name,
			"pduSessionId": pduSessionID,
			"dnn":          request["dnn"],
			"sNssai":       request["sNssai"],
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		}
		setSessionState(imsi, state)
		response := map[string]any{
			"status":             "session-created",
			"smf":                service.Name,
			"imsi":               imsi,
			"pduSessionId":       pduSessionID,
			"dnn":                request["dnn"],
			"sNssai":             request["sNssai"],
			"upfAnchor":          "upf:80",
			"sessionAmbr":        "1 Gbps",
			"notificationTarget": "amf",
			"sessionLifecycle":   state,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	mux.HandleFunc("/session/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		imsi := r.URL.Query().Get("imsi")
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := getSessionState(imsi)
		if state == nil {
			state = map[string]any{
				"imsi":        imsi,
				"state":       "inactive",
				"smf":         service.Name,
				"lastUpdated": time.Now().UTC().Format(time.RFC3339),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("/release-session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid release request", http.StatusBadRequest)
			return
		}
		imsi := fmt.Sprint(request["imsi"])
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := map[string]any{
			"imsi":         imsi,
			"state":        "inactive",
			"smf":          service.Name,
			"sessionState": "deactivated",
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		}
		setSessionState(imsi, state)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":       "session-released",
			"imsi":         imsi,
			"smf":          service.Name,
			"sessionState": "inactive",
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		})
	})
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("SMF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
