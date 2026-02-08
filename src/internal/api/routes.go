package api

import (
	"encoding/json"
	"net/http"
)

func (s *APIServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"version": s.version,
	})
}

func (s *APIServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.cfg)
}

func (s *APIServer) handleRules(w http.ResponseWriter, r *http.Request) {
	header := s.server.GetRewriter().HeaderRules()
	body := s.server.GetRewriter().BodyRules()
	redirect := s.server.GetRewriter().RedirectRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"header":   header,
		"body":     body,
		"redirect": redirect,
	})
}

func (s *APIServer) handleHeaderRules(w http.ResponseWriter, r *http.Request) {
	header := s.server.GetRewriter().HeaderRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(header)
}

func (s *APIServer) handleBodyRules(w http.ResponseWriter, r *http.Request) {
	body := s.server.GetRewriter().BodyRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func (s *APIServer) handleRedirectRules(w http.ResponseWriter, r *http.Request) {
	redirect := s.server.GetRewriter().RedirectRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(redirect)
}
