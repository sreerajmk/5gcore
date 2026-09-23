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
	host := getEnv("UE_HOST", "127.0.0.1")
	port := getEnvInt("UE_PORT", 8006)
	gnbURL := getEnv("GNB_URL", "http://127.0.0.1:8010")
	amfURL := getEnv("AMF_URL", "http://127.0.0.1:8001")
	imsi := getEnv("UE_IMSI", "310150123456789")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "ue-sim",
			"imsi":    imsi,
			"gNB":     gnbURL,
			"amf":     amfURL,
		})
	})

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"imsi":           imsi,
			"deviceState":    "idle",
			"gNBURL":         gnbURL,
			"amfURL":         amfURL,
			"lastUpdated":    time.Now().UTC().Format(time.RFC3339),
			"simulatorReady": true,
		})
	})

	mux.HandleFunc("/lifecycle", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"imsi":              imsi,
			"deviceState":       "idle",
			"ueLifecycleState":  "detached",
			"pduSessionState":   "inactive",
			"registrationState": "not-registered",
			"lastUpdated":       time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/attach", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		payload := map[string]any{
			"imsi":         imsi,
			"securityMode": "5G-AKA",
			"plmn":         "00101",
		}
		if r.Body != nil {
			var request map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err == nil {
				for key, value := range request {
					payload[key] = value
				}
			}
		}
		if value, ok := payload["imsi"]; ok && value != "" {
			imsi = fmt.Sprint(value)
		}

		var response map[string]any
		if err := callJSON(http.MethodPost, gnbURL+"/attach", payload, &response); err != nil {
			http.Error(w, fmt.Sprintf("gNodeB attach failed: %v", err), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ue":                 "simulator",
			"imsi":               imsi,
			"targetGNB":          gnbURL,
			"targetAMF":          amfURL,
			"attachRequest":      payload,
			"gNBResponse":        response,
			"simulatorTimestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		payload := map[string]any{"imsi": imsi, "securityMode": "5G-AKA"}
		var response map[string]any
		if err := callJSON(http.MethodPost, gnbURL+"/ue/register", payload, &response); err != nil {
			http.Error(w, fmt.Sprintf("gNodeB registration failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ue":           "simulator",
			"imsi":         imsi,
			"registration": response,
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/release", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		payload := map[string]any{"imsi": imsi}
		var response map[string]any
		if err := callJSON(http.MethodPost, gnbURL+"/ue/release", payload, &response); err != nil {
			http.Error(w, fmt.Sprintf("gNodeB release failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ue":          "simulator",
			"imsi":        imsi,
			"release":     response,
			"lastUpdated": time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/deactivate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		payload := map[string]any{"imsi": imsi, "reason": "ue-triggered-deactivation"}
		var response map[string]any
		if err := callJSON(http.MethodPost, gnbURL+"/ue/release", payload, &response); err != nil {
			http.Error(w, fmt.Sprintf("gNodeB deactivation failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ue":           "simulator",
			"imsi":         imsi,
			"deactivation": response,
			"lastUpdated":  time.Now().UTC().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/session-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var response map[string]any
		if err := callJSON(http.MethodGet, gnbURL+"/pdu-session/status?imsi="+imsi, nil, &response); err != nil {
			http.Error(w, fmt.Sprintf("gNodeB session status lookup failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/simulate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var attach map[string]any
		if err := callJSON(http.MethodPost, gnbURL+"/attach", map[string]any{"imsi": imsi, "securityMode": "5G-AKA", "plmn": "00101"}, &attach); err != nil {
			http.Error(w, fmt.Sprintf("simulate gNodeB attach failed: %v", err), http.StatusBadGateway)
			return
		}
		var status map[string]any
		if err := callJSON(http.MethodGet, gnbURL+"/pdu-session/status?imsi="+imsi, nil, &status); err != nil {
			http.Error(w, fmt.Sprintf("simulate gNodeB status lookup failed: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ue":        "simulator",
			"imsi":      imsi,
			"attach":    attach,
			"pduStatus": status,
		})
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	log.Printf("UE simulator listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
