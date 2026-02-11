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
	header := s.Server.GetRewriter().HeaderRules()
	body := s.Server.GetRewriter().BodyRules()
	redirect := s.Server.GetRewriter().RedirectRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"header":   header,
		"body":     body,
		"redirect": redirect,
	})
}

func (s *APIServer) handleHeaderRules(w http.ResponseWriter, r *http.Request) {
	header := s.Server.GetRewriter().HeaderRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(header)
}

func (s *APIServer) handleBodyRules(w http.ResponseWriter, r *http.Request) {
	body := s.Server.GetRewriter().BodyRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func (s *APIServer) handleRedirectRules(w http.ResponseWriter, r *http.Request) {
	redirect := s.Server.GetRewriter().RedirectRules()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(redirect)
}

func (s *APIServer) handleRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.RestartSystem(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("success"))
}
