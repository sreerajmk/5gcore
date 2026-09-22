package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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
			_, _ = w.Write([]byte(fmt.Sprintf("%v", raw)))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("AMF service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
	_ = os.Stdout
}
