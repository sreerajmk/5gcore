package nf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Service struct {
	Name         string
	Service      string
	Host         string
	Port         int
	Capabilities []string
	Description  string
	NRFURL       string
}

func NewService(name, service string, capabilities []string, description string) *Service {
	h := os.Getenv("NF_HOST")
	if h == "" {
		h = "127.0.0.1"
	}
	p := 8000
	if value := os.Getenv("NF_PORT"); value != "" {
		_, _ = fmt.Sscanf(value, "%d", &p)
	}
	if service == "amf" {
		p = 8001
	} else if service == "smf" {
		p = 8002
	} else if service == "ausf" {
		p = 8003
	} else if service == "upf" {
		p = 8004
	} else if service == "udm" {
		p = 8005
	}
	return &Service{
		Name:         name,
		Service:      service,
		Host:         h,
		Port:         p,
		Capabilities: capabilities,
		Description:  description,
		NRFURL:       os.Getenv("NRF_URL"),
	}
}

func (s *Service) nrfBaseURL() string {
	if s.NRFURL == "" {
		s.NRFURL = "http://127.0.0.1:8000"
	}
	return s.NRFURL
}

func (s *Service) Register() error {
	payload, err := json.Marshal(map[string]any{
		"name":         s.Name,
		"service":      s.Service,
		"host":         s.Host,
		"port":         s.Port,
		"url":          fmt.Sprintf("http://%s:%d", s.Host, s.Port),
		"capabilities": s.Capabilities,
		"description":  s.Description,
	})
	if err != nil {
		return err
	}
	resp, err := http.Post(s.nrfBaseURL()+"/register", "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("NRF registration failed: %s", resp.Status)
	}
	return nil
}

func (s *Service) Discover(service string) (map[string]any, error) {
	url := s.nrfBaseURL() + "/discover/" + service
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("discovery failed: %s", resp.Status)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) DiscoverURL(service string) (string, error) {
	value, err := s.Discover(service)
	if err != nil {
		return "", err
	}
	if rawURL, ok := value["url"].(string); ok && rawURL != "" {
		return rawURL, nil
	}
	if host, ok := value["host"].(string); ok && host != "" {
		port, _ := value["port"].(float64)
		return fmt.Sprintf("http://%s:%v", host, int(port)), nil
	}
	return "", fmt.Errorf("no URL discovered for %s", service)
}

func (s *Service) CallJSON(method, target string, payload any, out any) error {
	if target == "" {
		return fmt.Errorf("empty target URL")
	}
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
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request to %s failed: %s", target, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Service) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "ok",
			"name":        s.Name,
			"service":     s.Service,
			"description": s.Description,
		})
	}
}

func (s *Service) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":         s.Name,
			"service":      s.Service,
			"status":       "registered",
			"nrf":          s.nrfBaseURL(),
			"capabilities": s.Capabilities,
			"description":  s.Description,
		})
	}
}
