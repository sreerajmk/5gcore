package main

import (
	"fmt"
	"log"
	"net/http"

	"5gcore/internal/nf"
)

func main() {
	service := nf.NewService("UDM", "udm", []string{"subscription-data", "authentication-data"}, "Unified Data Management")
	if service.NRFURL == "" {
		service.NRFURL = "http://127.0.0.1:8000"
	}
	if err := service.Register(); err != nil {
		log.Printf("register with NRF failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", service.HealthHandler())
	mux.HandleFunc("/status", service.StatusHandler())
	addr := fmt.Sprintf("%s:%d", service.Host, service.Port)
	log.Printf("UDM service starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
