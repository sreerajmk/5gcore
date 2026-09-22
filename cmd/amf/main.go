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
	ueStateMu sync.RWMutex
	ueState   = map[string]map[string]any{}
)

func getUEState(imsi string) map[string]any {
	ueStateMu.RLock()
	defer ueStateMu.RUnlock()
	if state, ok := ueState[imsi]; ok {
		return state
	}
	return nil
}

func setUEState(imsi string, state map[string]any) {
	ueStateMu.Lock()
	defer ueStateMu.Unlock()
	ueState[imsi] = state
}

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
	mux.HandleFunc("/ue/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid registration payload", http.StatusBadRequest)
			return
		}
		imsi := fmt.Sprint(request["imsi"])
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := map[string]any{
			"imsi":             imsi,
			"state":            "registered",
			"ueLifecycleState": "connected",
			"amf":              service.Name,
			"lastUpdated":      time.Now().UTC().Format(time.RFC3339),
			"securityMode":     request["securityMode"],
		}
		setUEState(imsi, state)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("/ue/status/", func(w http.ResponseWriter, r *http.Request) {
		imsi := r.URL.Path[len("/ue/status/"):]
		state := getUEState(imsi)
		if state == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state)
	})
	mux.HandleFunc("/ue/release", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid release payload", http.StatusBadRequest)
			return
		}
		imsi := fmt.Sprint(request["imsi"])
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := getUEState(imsi)
		if state == nil {
			http.NotFound(w, r)
			return
		}
		reason := fmt.Sprint(request["reason"])
		if reason == "" {
			reason = "ue-triggered-release"
		}
		state["ueLifecycleState"] = "idle"
		state["state"] = "detached"
		state["lastUpdated"] = time.Now().UTC().Format(time.RFC3339)
		state["releaseReason"] = reason
		state["pduSessionState"] = map[string]any{
			"pduSessionId": 1,
			"state":        "inactive",
			"dnn":          "internet",
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		}
		setUEState(imsi, state)

		smfURL, err := service.DiscoverURL("smf")
		if err == nil {
			_ = service.CallJSON(http.MethodPost, smfURL+"/release-session", map[string]any{"imsi": imsi, "reason": reason}, nil)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":           "released",
			"imsi":             imsi,
			"reason":           reason,
			"ueLifecycleState": state["ueLifecycleState"],
			"pduSessionState":  state["pduSessionState"],
		})
	})
	mux.HandleFunc("/ue/lifecycle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		imsi := r.URL.Query().Get("imsi")
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := getUEState(imsi)
		if state == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"imsi":              imsi,
			"state":             state["state"],
			"ueLifecycleState":  state["ueLifecycleState"],
			"registrationState": state["state"],
			"pduSessionState":   state["pduSessionState"],
			"lastUpdated":       state["lastUpdated"],
		})
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
		ueState := map[string]any{
			"imsi":             imsi,
			"state":            "authenticating",
			"ueLifecycleState": "connected",
			"amf":              service.Name,
			"lastUpdated":      time.Now().UTC().Format(time.RFC3339),
		}
		setUEState(imsi, ueState)

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
		ueState["state"] = "authenticated"
		ueState["securityContext"] = authResponse
		setUEState(imsi, ueState)

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
		ueState["subscriberProfile"] = profile
		setUEState(imsi, ueState)

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
		upfURL, err := service.DiscoverURL("upf")
		if err == nil {
			var upfForward map[string]any
			_ = service.CallJSON(http.MethodPost, upfURL+"/forward", map[string]any{"imsi": imsi, "source": "amf", "destination": "internet", "pduSessionId": 1}, &upfForward)
			session["upfForwarding"] = upfForward
		}
		ueState["state"] = "registered"
		ueState["ueLifecycleState"] = "connected"
		ueState["pduSessionState"] = map[string]any{
			"pduSessionId": 1,
			"state":        "active",
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
			"dnn":          "internet",
		}
		setUEState(imsi, ueState)

		response := map[string]any{
			"procedure": "5g-registration-and-session-setup",
			"status":    "attach-accepted",
			"imsi":      imsi,
			"amf":       service.Name,
			"registrationState": map[string]any{
				"state":            "registered",
				"ueLifecycleState": "connected",
				"lastUpdated":      ueState["lastUpdated"],
			},
			"pduSessionLifecycle": map[string]any{
				"pduSessionId": 1,
				"state":        "active",
				"transition":   []string{"establishing", "active"},
				"dnn":          "internet",
			},
			"authResult":       authResponse,
			"subscriptionInfo": profile,
			"sessionContext":   session,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
	mux.HandleFunc("/procedure/attach", handleProcedure)
	mux.HandleFunc("/procedure/registration", handleProcedure)
	mux.HandleFunc("/pdu-session/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		imsi := r.URL.Query().Get("imsi")
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		state := getUEState(imsi)
		if state == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state["pduSessionState"])
	})
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("AMF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
