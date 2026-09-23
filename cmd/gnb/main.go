package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func callJSON(method, target string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, target, body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request to %s failed: %s: %s", target, resp.Status, string(bodyBytes))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func main() {
	host := getEnv("GNB_HOST", "127.0.0.1")
	port := getEnvInt("GNB_PORT", 8010)
	amfURL := getEnv("AMF_URL", "http://127.0.0.1:8001")
	cellID := getEnv("GNB_CELL_ID", "cell-1")
	registrationArea := getEnv("GNB_REGISTRATION_AREA", "area-1")
	activeCells := map[string]string{"cell-1": registrationArea}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "gNodeB-stub",
			"cellId":  cellID,
			"amf":     amfURL,
		})
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":          "gNodeB-stub",
			"cellId":           cellID,
			"amf":              amfURL,
			"registrationArea": registrationArea,
			"connectedUes":     0,
			"activeCells":      activeCells,
			"lastUpdated":      time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/attach", func(w http.ResponseWriter, r *http.Request) {
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
		selectedCell := cellID
		if value, ok := request["cellId"]; ok && value != nil {
			selectedCell = fmt.Sprint(value)
			activeCells[selectedCell] = registrationArea
		}
		if imsi == "" {
			http.Error(w, "imsi is required", http.StatusBadRequest)
			return
		}
		attachResult := "success"
		if selectedCell == "" {
			attachResult = "failed"
		}
		var response map[string]any
		if attachResult == "success" {
			if err := callJSON(http.MethodPost, amfURL+"/procedure/attach", request, &response); err != nil {
				http.Error(w, fmt.Sprintf("AMF attach failed: %v", err), http.StatusBadGateway)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":          "gNodeB-stub",
			"cellId":           selectedCell,
			"registrationArea": registrationArea,
			"cellSelection":    map[string]any{"selectedCell": selectedCell, "reason": "best-signal"},
			"attachResult":     attachResult,
			"attachRequest":    request,
			"amfResponse":      response,
			"activeCells":      activeCells,
			"lastUpdated":      time.Now().UTC().Format(time.RFC3339),
		})
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
		request["registrationArea"] = registrationArea
		var response map[string]any
		if err := callJSON(http.MethodPost, amfURL+"/ue/register", request, &response); err != nil {
			http.Error(w, fmt.Sprintf("AMF registration failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":           "gNodeB-stub",
			"cellId":            cellID,
			"registrationArea":  registrationArea,
			"registrationState": "registered",
			"registration":      response,
		})
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
		request["cellId"] = cellID
		request["registrationArea"] = registrationArea
		var response map[string]any
		if err := callJSON(http.MethodPost, amfURL+"/ue/release", request, &response); err != nil {
			http.Error(w, fmt.Sprintf("AMF release failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":          "gNodeB-stub",
			"cellId":           cellID,
			"registrationArea": registrationArea,
			"releaseState":     "idle",
			"release":          response,
		})
	})

	mux.HandleFunc("/pdu-session/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		imsi := r.URL.Query().Get("imsi")
		var response map[string]any
		if err := callJSON(http.MethodGet, amfURL+"/pdu-session/status?imsi="+imsi, nil, &response); err != nil {
			http.Error(w, fmt.Sprintf("AMF session status failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":          "gNodeB-stub",
			"cellId":           cellID,
			"registrationArea": registrationArea,
			"activeCell":       cellID,
			"imsi":             imsi,
			"pduStatus":        response,
		})
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("gNodeB stub listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
