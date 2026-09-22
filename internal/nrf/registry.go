package nrf

import (
	"fmt"
	"strings"
	"sync"
)

type NFRecord struct {
	Name         string   `json:"name"`
	Service      string   `json:"service"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	URL          string   `json:"url"`
	Capabilities []string `json:"capabilities,omitempty"`
	Description  string   `json:"description,omitempty"`
}

type Registry struct {
	mu    sync.RWMutex
	items map[string]NFRecord
}

func NewRegistry() *Registry {
	return &Registry{items: make(map[string]NFRecord)}
}

func (r *Registry) Register(nf NFRecord) NFRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	if nf.URL == "" {
		nf.URL = fmt.Sprintf("http://%s:%d", nf.Host, nf.Port)
	}
	serviceKey := strings.ToLower(nf.Service)
	nameKey := strings.ToLower(nf.Name)
	r.items[serviceKey] = nf
	r.items[nameKey] = nf
	return nf
}

func (r *Registry) Find(service string) (NFRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lookup := strings.ToLower(service)
	if nf, ok := r.items[lookup]; ok {
		return nf, true
	}
	for _, nf := range r.items {
		if strings.EqualFold(nf.Service, service) || strings.EqualFold(nf.Name, service) {
			return nf, true
		}
	}
	return NFRecord{}, false
}

func (r *Registry) List() []NFRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]NFRecord, 0, len(r.items))
	seen := make(map[string]struct{})
	for _, nf := range r.items {
		key := strings.ToLower(nf.Service) + "::" + strings.ToLower(nf.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, nf)
	}
	return out
}
